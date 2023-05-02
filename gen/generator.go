package gen

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/go-faster/yaml"
	"github.com/ogen-go/ogen"

	"github.com/ogen-go/protoc-gen-oas/internal/naming"
)

// NewGenerator returns new Generator instance.
func NewGenerator(protoFiles []*protogen.File, opts ...GeneratorOption) (*Generator, error) {
	g := new(Generator)
	g.spec = ogen.NewSpec()
	g.responses = make(map[string]struct{})
	g.requestBodies = make(map[string]struct{})
	g.parameters = make(map[string]struct{})

	for _, file := range protoFiles {
		if has := strings.HasPrefix(file.GeneratedFilenamePrefix, "google.golang.org"); has {
			continue
		}

		for _, service := range file.Services {
			g.methods = append(g.methods, service.Methods...)
		}

		g.messages = append(g.messages, file.Messages...)
		g.enums = append(g.enums, file.Enums...)
	}

	for _, opt := range opts {
		opt(g)
	}

	g.mkPaths()
	g.mkComponents()

	return g, nil
}

// Generator instance.
type Generator struct {
	methods       []*protogen.Method
	messages      []*protogen.Message
	enums         []*protogen.Enum
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

func (g *Generator) mkPaths() {
	for _, method := range g.methods {
		ext := proto.GetExtension(method.Desc.Options(), annotations.E_Http)
		httpRule, ok := ext.(*annotations.HttpRule)
		if !ok || httpRule == nil {
			continue
		}

		response := string(method.Output.Desc.Name())
		g.responses[response] = struct{}{}

		switch path := httpRule.Pattern.(type) {
		case *annotations.HttpRule_Get:
			g.mkGetOp(path.Get, method)

		case *annotations.HttpRule_Put:
		case *annotations.HttpRule_Post:
		case *annotations.HttpRule_Delete:
		case *annotations.HttpRule_Patch:
		}
	}
}

func (g *Generator) mkGetOp(path string, method *protogen.Method) {
	g.mkParameters(path, method.Input.Desc)

	opID := g.mkOpID(method.Desc)
	parameters := make([]*ogen.Parameter, 0)
	fields := method.Input.Desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		paramName := string(fields.Get(i).Name())
		paramRef := g.paramRef(naming.CamelCase(paramName))
		parameters = append(parameters, ogen.NewParameter().SetRef(paramRef))
	}
	respName := string(method.Output.Desc.Name())
	respRef := g.respRef(respName)
	op := ogen.NewOperation().
		SetOperationID(opID).
		SetParameters(parameters).
		SetResponses(ogen.Responses{
			"200": ogen.NewResponse().SetRef(respRef),
		})
	g.spec.AddPathItem(path, ogen.NewPathItem().SetGet(op))
}

func (g *Generator) mkOpID(methodDescriptor protoreflect.MethodDescriptor) string {
	name := string(methodDescriptor.Name())
	return naming.LowerCamelCase(name)
}

func (g *Generator) mkParameters(path string, messageDescriptor protoreflect.MessageDescriptor) {
	curlyBracketsWords := g.curlyBracketsWords(path)
	isPathParam := func(name string) bool {
		_, isPathParam := curlyBracketsWords[name]
		return isPathParam
	}
	fields := messageDescriptor.Fields()
	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())
		g.mkParameter(isPathParam(name), fields.Get(i))
	}
}

func (g *Generator) mkParameter(isPathParam bool, fieldDescriptor protoreflect.FieldDescriptor) {
	name := string(fieldDescriptor.Name())
	g.parameters[name] = struct{}{}

	in := "query"
	if isPathParam {
		in = "path"
	}

	s := g.typ(fieldDescriptor)
	isRequired := !s.Nullable
	param := ogen.NewParameter().
		SetIn(in).
		SetName(name).
		SetSchema(s).
		SetRequired(isRequired)

	g.spec.AddParameter(naming.CamelCase(name), param)
}

func (g *Generator) curlyBracketsWords(path string) map[string]struct{} {
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

func (g *Generator) respRef(s string) string {
	resp := naming.LastAfterDots(s)
	return fmt.Sprintf("#/components/responses/%s", resp)
}

func (g *Generator) paramRef(s string) string {
	return fmt.Sprintf("#/components/parameters/%s", s)
}

func (g *Generator) mkComponents() {
	g.mkResponses()
}

func (g *Generator) mkResponses() {
	for _, message := range g.messages {
		g.mkResponse(message)
	}
}

func (g *Generator) mkResponse(message *protogen.Message) {
	name := string(message.Desc.Name())
	if _, ok := g.responses[name]; !ok {
		return
	}

	schema := ogen.NewSchema()
	properties := make(ogen.Properties, 0, len(message.Fields))
	r := make([]string, 0)
	for _, field := range message.Fields {
		prop := g.mkProperty(field.Desc)
		properties = append(properties, prop)
		if !prop.Schema.Nullable {
			r = append(r, field.Desc.JSONName())
		}
	}
	schema.SetProperties(&properties).SetRequired(r)
	g.spec.AddResponse(name, ogen.NewResponse().
		SetDescription(name).
		SetContent(map[string]ogen.Media{
			"application/json": {
				Schema: schema,
			},
		}),
	)
}

func (g *Generator) mkProperty(fieldDescriptor protoreflect.FieldDescriptor) ogen.Property {
	name := fieldDescriptor.JSONName()
	schema := g.mkPropertySchema(fieldDescriptor)

	return ogen.Property{
		Name:   name,
		Schema: schema,
	}
}

func (g *Generator) mkPropertySchema(fieldDescriptor protoreflect.FieldDescriptor) *ogen.Schema {
	s := ogen.NewSchema()

	switch fieldDescriptor.Cardinality() {
	case protoreflect.Optional:
		s = g.typ(fieldDescriptor)

	case protoreflect.Repeated:
		typName := g.typName(fieldDescriptor)
		ref := g.respRef(typName)
		s.SetType("array").SetItems(ogen.NewSchema().SetRef(ref))
	}

	return s
}

func (g *Generator) typName(fieldDescriptor protoreflect.FieldDescriptor) string {
	switch {
	case fieldDescriptor.Message() != nil:
		fullName := string(fieldDescriptor.Message().FullName())
		return fullName

	default:
		return fieldDescriptor.Kind().String()
	}
}

func (g *Generator) typ(fieldDescriptor protoreflect.FieldDescriptor) *ogen.Schema {
	typName := g.typName(fieldDescriptor)
	s := ogen.NewSchema()

	switch typName {
	case "bool":
		return s.SetType("boolean")

	case "bytes":
		return s.SetType("string").SetFormat("binary")

	case "double":
		return s.SetType("number").SetFormat("double")

	case "enum":
		var enum []json.RawMessage
		for i := 0; i < fieldDescriptor.Enum().Values().Len(); i++ {
			val := []byte(fieldDescriptor.Enum().Values().Get(i).Name())
			enum = append(enum, val)
		}
		return s.SetType("string").SetEnum(enum)

	case "int32":
		return s.SetType("integer").SetFormat(typName)

	case "google.protobuf.DoubleValue":
		return s.SetType("number").SetFormat("double").SetNullable(true)

	case "google.protobuf.StringValue":
		return s.SetType("string").SetNullable(true)

	case "google.protobuf.Timestamp":
		return s.SetType("string").SetFormat("date-time")

	default:
		return s.SetType(typName)
	}
}
