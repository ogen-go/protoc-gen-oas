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

func (g *Generator) mkPaths() {
	for _, method := range g.methods {
		ext := proto.GetExtension(method.Desc.Options(), annotations.E_Http)
		httpRule, ok := ext.(*annotations.HttpRule)
		if !ok {
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
	parameter := string(method.Input.Desc.Name())
	g.parameters[parameter] = struct{}{}

	opID := g.mkOpID(method.Desc)
	name := string(method.Output.Desc.Name())
	ref := g.respRef(name)
	op := ogen.NewOperation().
		SetOperationID(opID).
		SetResponses(ogen.Responses{
			"200": ogen.NewResponse().SetRef(ref),
		})
	g.spec.AddPathItem(path, ogen.NewPathItem().SetGet(op))
}

func (g *Generator) mkOpID(methodDescriptor protoreflect.MethodDescriptor) string {
	name := string(methodDescriptor.Name())
	return naming.LowerCamelCase(name)
}

func (g *Generator) respRef(s string) string {
	resp := naming.LastAfterDots(s)
	return fmt.Sprintf("#/components/responses/%s", resp)
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
	schema := ogen.NewSchema()
	properties := make(ogen.Properties, 0, len(message.Fields))
	for _, field := range message.Fields {
		properties = append(properties, g.mkProperty(field.Desc))
	}
	schema.SetProperties(&properties)
	name := string(message.Desc.Name())
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
