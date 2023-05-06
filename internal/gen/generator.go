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

	"github.com/ogen-go/protoc-gen-oas/internal/naming"
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

	for _, f := range fs {
		if isSkip := !f.Generate; isSkip {
			continue
		}

		for _, service := range f.Services {
			g.methods = append(g.methods, service.Methods...)
		}

		g.messages = append(g.messages, f.Mesagges...)
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

	r := m.HTTPRule

	switch r.Method {
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
	g.addPathParams(m.PathParamsFields())
	queryParams := g.mkQueryParams(m.Path(), m.Request)

	op := m.Op().
		AddParameters(m.PathParams()...).
		AddParameters(queryParams...)

	if g.spec.Paths[m.Path()] == nil {
		g.spec.AddPathItem(m.Path(), ogen.NewPathItem().SetGet(op))
	} else {
		g.spec.Paths[m.Path()].SetGet(op)
	}
}

func (g *Generator) setPutOp(m *Method) {
	g.addPathParams(m.PathParamsFields())
	reqBody := g.mkReqBody(m.Path(), m.Request)

	op := m.Op().
		AddParameters(m.PathParams()...).
		SetRequestBody(reqBody)

	if g.spec.Paths[m.Path()] == nil {
		g.spec.AddPathItem(m.Path(), ogen.NewPathItem().SetPut(op))
	} else {
		g.spec.Paths[m.Path()].SetPut(op)
	}
}

func (g *Generator) setPostOp(m *Method) {
	reqBody := g.mkReqBody(m.Path(), m.Request)

	op := m.Op().
		SetRequestBody(reqBody)

	if g.spec.Paths[m.HTTPRule.Path] == nil {
		g.spec.AddPathItem(m.HTTPRule.Path, ogen.NewPathItem().SetPost(op))
	} else {
		g.spec.Paths[m.HTTPRule.Path].SetPost(op)
	}
}

func (g *Generator) setDeleteOp(m *Method) {
	g.addPathParams(m.PathParamsFields())

	op := m.Op().
		AddParameters(m.PathParams()...)

	if g.spec.Paths[m.HTTPRule.Path] == nil {
		g.spec.AddPathItem(m.HTTPRule.Path, ogen.NewPathItem().SetDelete(op))
	} else {
		g.spec.Paths[m.HTTPRule.Path].SetDelete(op)
	}
}

func (g *Generator) setPatchOp(m *Method) {
	g.addPathParams(m.PathParamsFields())
	reqBody := g.mkReqBody(m.Path(), m.Request)

	op := m.Op().
		AddParameters(m.PathParams()...).
		SetRequestBody(reqBody)
	if g.spec.Paths[m.HTTPRule.Path] == nil {
		g.spec.AddPathItem(m.HTTPRule.Path, ogen.NewPathItem().SetPatch(op))
	} else {
		g.spec.Paths[m.HTTPRule.Path].SetPatch(op)
	}
}

func (g *Generator) mkReqBody(path string, m *Message) *ogen.RequestBody {
	ref := reqBodyRef(m.Name.CamelCase())
	g.spec.AddRequestBody(m.Name.String(), ogen.NewRequestBody().SetContent(g.mkReqBodyContent(path, m)))
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

	props := make(ogen.Properties, 0, len(m.Fields))
	r := make([]string, 0)

	for _, field := range m.Fields {
		if isPathParam(field.Name.String()) {
			continue
		}
		prop := g.mkProperty(field)
		props = append(props, prop)
		if !prop.Schema.Nullable {
			r = append(r, field.Name.LowerCamelCase())
		}
	}

	return map[string]ogen.Media{
		"application/json": {
			Schema: ogen.NewSchema().SetProperties(&props).SetRequired(r),
		},
	}
}

func (g *Generator) mkQueryParams(path string, m *Message) []*ogen.Parameter {
	curlyBracketsWords := curlyBracketsWords(path)

	isPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return isPathParam
	}

	queryParams := make([]*ogen.Parameter, 0)

	for _, field := range m.Fields {
		if isPathParam(field.Name.String()) {
			continue
		}

		ref := paramRef(field.Name.CamelCase())
		queryParams = append(queryParams, ogen.NewParameter().SetRef(ref))

		g.mkParam("query", field)
	}

	return queryParams
}

func (g *Generator) addPathParams(fs Fields) {
	g.addParams("path", fs)
}

func (g *Generator) addParam(in string, f *Field) {
	s := f.Type.Schema()
	isRequired := !s.Nullable
	param := ogen.NewParameter().
		SetIn(in).
		SetName(f.Name.String()).
		SetSchema(s).
		SetRequired(isRequired)

	g.spec.AddParameter(f.Name.CamelCase(), param)
}

func (g *Generator) addParams(in string, fs Fields) {
	for _, f := range fs {
		g.addParam(in, f)
	}
}

func (g *Generator) mkParam(in string, f *Field) {
	s := f.Type.Schema()
	isRequired := !s.Nullable
	param := ogen.NewParameter().
		SetIn(in).
		SetName(f.Name.String()).
		SetSchema(s).
		SetRequired(isRequired)

	g.spec.AddParameter(f.Name.CamelCase(), param)
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
		prop := g.mkProperty(f)
		properties = append(properties, prop)
		if !prop.Schema.Nullable {
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

func (g *Generator) mkProperty(f *Field) ogen.Property {
	schema := g.mkPropertySchema(f)

	return ogen.Property{
		Name:   f.Name.LowerCamelCase(),
		Schema: schema,
	}
}

func (g *Generator) mkPropertySchema(f *Field) *ogen.Schema {
	s := ogen.NewSchema()

	switch f.Cardinality {
	case CardinalityOptional:
		s = f.Type.Schema()

	case CardinalityRepeated:
		n := naming.LastAfterDots(f.Type.Type)
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
	resp := naming.LastAfterDots(s)
	return fmt.Sprintf("#/components/responses/%s", resp)
}

func paramRef(s string) string {
	return fmt.Sprintf("#/components/parameters/%s", s)
}

func reqBodyRef(s string) string {
	return fmt.Sprintf("#/components/requestBodies/%s", s)
}
