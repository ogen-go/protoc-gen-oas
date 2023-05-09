package gen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/go-faster/errors"
	"github.com/go-faster/yaml"
	"github.com/ogen-go/ogen"
)

// ErrNoMethods reports that service have no methods.
var ErrNoMethods = errors.New("protoc-gen-oas: service has no methods")

// NewGenerator returns new Generator instance.
func NewGenerator(protoFiles []*protogen.File, opts ...GeneratorOption) (*Generator, error) {
	g := new(Generator)

	fs, err := NewFiles(protoFiles)
	if err != nil {
		return nil, err
	}

	messages := make(map[string]struct{})

	for _, f := range fs {
		if isSkip := !f.Generate; isSkip {
			continue
		}

		for _, service := range f.Services {
			for _, method := range service.Methods {
				g.methods = append(g.methods, method)
				messages[method.Response.Name.String()] = struct{}{}
				messages[method.Request().Name.String()] = struct{}{}
			}
		}
	}

	for _, f := range fs {
		for _, message := range f.Messages {
			if _, ok := messages[message.Name.String()]; ok {
				g.messages = append(g.messages, message)
			}
		}
	}

	if len(g.methods) == 0 {
		return nil, ErrNoMethods
	}

	g.init()

	for _, opt := range opts {
		opt(g)
	}

	g.setPaths()
	g.mkComponents()

	return g, nil
}

// Generator instance.
type Generator struct {
	methods       Methods
	messages      Messages
	schemas       map[string]struct{}
	responses     map[string]struct{}
	requestBodies map[string]struct{}
	parameters    map[string]struct{}
	spec          *ogen.Spec
}

// YAML returns OpenAPI specification bytes.
func (g *Generator) YAML() ([]byte, error) {
	return yaml.Marshal(g.spec)
}

// JSON returns OpenAPI specification bytes.
func (g *Generator) JSON() ([]byte, error) {
	return json.Marshal(g.spec)
}

func (g *Generator) init() {
	g.schemas = make(map[string]struct{})
	g.responses = make(map[string]struct{})
	g.requestBodies = make(map[string]struct{})
	g.parameters = make(map[string]struct{})
	g.spec = ogen.NewSpec()
}

func (g *Generator) setPaths() {
	for _, method := range g.methods {
		g.addPath(method)
	}
}

func (g *Generator) addPath(m *Method) {
	g.responses[m.Response.Name.String()] = struct{}{}

	switch m.HTTPRule.Method {
	case http.MethodGet:
		g.setGetOp(m)

	case http.MethodPut:
		g.setPutOp(m)

	case http.MethodPost:
		g.setPostOp(m)

	case http.MethodDelete:
		g.setDeleteOp(m)

	case http.MethodPatch:
		g.setPatchOp(m)
	}
}

func (g *Generator) setGetOp(m *Method) {
	if g.spec.Paths[m.Path()] == nil {
		g.spec.AddPathItem(m.Path(), ogen.NewPathItem().SetGet(m.Op()))
	} else {
		g.spec.Paths[m.Path()].SetGet(m.Op())
	}
}

func (g *Generator) setPutOp(m *Method) {
	reqBody := g.mkReqBody(m.Path(), m.Request())

	op := m.Op().SetRequestBody(reqBody)

	if g.spec.Paths[m.Path()] == nil {
		g.spec.AddPathItem(m.Path(), ogen.NewPathItem().SetPut(op))
	} else {
		g.spec.Paths[m.Path()].SetPut(op)
	}
}

func (g *Generator) setPostOp(m *Method) {
	reqBody := g.mkReqBody(m.Path(), m.Request())

	op := m.Op().SetRequestBody(reqBody)

	if g.spec.Paths[m.HTTPRule.Path] == nil {
		g.spec.AddPathItem(m.HTTPRule.Path, ogen.NewPathItem().SetPost(op))
	} else {
		g.spec.Paths[m.HTTPRule.Path].SetPost(op)
	}
}

func (g *Generator) setDeleteOp(m *Method) {
	if g.spec.Paths[m.Path()] == nil {
		g.spec.AddPathItem(m.Path(), ogen.NewPathItem().SetDelete(m.Op()))
	} else {
		g.spec.Paths[m.Path()].SetDelete(m.Op())
	}
}

func (g *Generator) setPatchOp(m *Method) {
	reqBody := g.mkReqBody(m.Path(), m.Request())

	op := m.Op().SetRequestBody(reqBody)

	if g.spec.Paths[m.Path()] == nil {
		g.spec.AddPathItem(m.Path(), ogen.NewPathItem().SetPatch(op))
	} else {
		g.spec.Paths[m.Path()].SetPatch(op)
	}
}

func (g *Generator) mkReqBody(path string, m *Message) *ogen.RequestBody {
	ref := reqBodyRef(m.Name.CamelCase())
	g.spec.AddRequestBody(m.Name.CamelCase(), ogen.NewRequestBody().SetContent(g.mkReqBodyContent(path, m)))
	return ogen.NewRequestBody().SetRef(ref)
}

func (g *Generator) mkReqBodyContent(path string, m *Message) map[string]ogen.Media {
	if len(m.Fields) == 0 {
		return map[string]ogen.Media{
			"application/json": {},
		}
	}

	curlyBracketsWords := curlyBracketsWords(path)
	isPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return isPathParam
	}

	properties := make(ogen.Properties, 0, len(m.Fields))
	r := make([]string, 0)

	for _, f := range m.Fields {
		if isPathParam(f.Name.String()) {
			continue
		}
		property := g.property(f)
		properties = append(properties, property)
		if f.Options.IsRequired {
			r = append(r, f.Name.LowerCamelCase())
		}
	}

	return map[string]ogen.Media{
		"application/json": {
			Schema: ogen.NewSchema().SetProperties(&properties).SetRequired(r),
		},
	}
}

func (g *Generator) mkComponents() {
	g.mkResponses()
}

func (g *Generator) mkResponses() {
	for _, message := range g.messages {
		g.mkResponse(message)
	}
}

func (g *Generator) mkResponse(m *Message) {
	if _, ok := g.responses[m.Name.String()]; !ok {
		return
	}

	schema := ogen.NewSchema()
	properties := make(ogen.Properties, 0, len(m.Fields))
	r := make([]string, 0)

	for _, f := range m.Fields {
		property := g.property(f)
		properties = append(properties, property)
		if f.Options.IsRequired {
			r = append(r, f.Name.LowerCamelCase())
		}
	}
	schema.SetProperties(&properties).SetRequired(r)
	g.spec.AddResponse(m.Name.String(), ogen.NewResponse().
		SetDescription(m.Name.String()).
		SetContent(map[string]ogen.Media{
			"application/json": {
				Schema: schema,
			},
		}),
	)
}

func (g *Generator) property(f *Field) ogen.Property {
	schema := g.propertySchema(f)

	return ogen.Property{
		Name:   f.Name.LowerCamelCase(),
		Schema: schema,
	}
}

func (g *Generator) propertySchema(f *Field) *ogen.Schema {
	s := ogen.NewSchema()

	switch f.Cardinality {
	case CardinalityOptional:
		if f.Type.HasEnum() {
			g.spec.AddSchema(f.Name.CamelCase(), f.Type.Schema())
			s.SetRef(schemaRef(f.Name.CamelCase()))
		} else {
			s = f.Type.Schema()
		}

	case CardinalityRepeated:
		n := LastAfterDots(f.Type.Type)
		if resp, ok := g.spec.Components.Responses[n]; ok {
			if c, ok := resp.Content["application/json"]; ok {
				g.spec.AddSchema(n, c.Schema)
				s.SetType("array").SetItems(ogen.NewSchema().SetRef(schemaRef(n)))
			}
		}
	}

	return s
}

func curlyBracketsWords(path string) map[string]struct{} {
	words := strings.Split(path, "/")
	curlyBracketsWords := make(map[string]struct{})
	for _, word := range words {
		if len(word) < 2 {
			continue
		}

		if word[0] == '{' && word[len(word)-1] == '}' {
			curlyBracketsWord := word[1 : len(word)-1]
			curlyBracketsWords[curlyBracketsWord] = struct{}{}
		}
	}
	return curlyBracketsWords
}

func schemaRef(s string) string {
	return fmt.Sprintf("#/components/schemas/%s", s)
}

func respRef(s string) string {
	resp := LastAfterDots(s)
	return fmt.Sprintf("#/components/responses/%s", resp)
}

func reqBodyRef(s string) string {
	return fmt.Sprintf("#/components/requestBodies/%s", s)
}
