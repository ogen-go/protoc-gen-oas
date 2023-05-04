package gen

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

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

	for _, file := range protoFiles {
		if isSkip := !file.Generate; isSkip {
			continue
		}

		for _, service := range file.Services {
			g.methods = append(g.methods, service.Methods...)
		}

		g.messages = append(g.messages, file.Messages...)
		g.enums = append(g.enums, file.Enums...)
	}

	if len(g.methods) == 0 {
		return nil, ErrNoMethods
	}

	g.init()

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

func (g *Generator) init() {
	g.responses = make(map[string]struct{})
	g.requestBodies = make(map[string]struct{})
	g.parameters = make(map[string]struct{})
	g.spec = ogen.NewSpec()
}

func (g *Generator) mkPaths() {
	for _, method := range g.methods {
		g.mkPath(method)
	}
}

func (g *Generator) mkPath(method *protogen.Method) {
	ext := proto.GetExtension(method.Desc.Options(), annotations.E_Http)
	httpRule, ok := ext.(*annotations.HttpRule)
	if !ok || httpRule == nil {
		return
	}

	response := string(method.Output.Desc.Name())
	g.responses[response] = struct{}{}

	switch path := httpRule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		g.mkGetOp(path.Get, method)

	case *annotations.HttpRule_Put:
		g.mkPutOp(path.Put, method)

	case *annotations.HttpRule_Post:
		g.mkPostOp(path.Post, method)

	case *annotations.HttpRule_Delete:
		g.mkDeleteOp(path.Delete, method)

	case *annotations.HttpRule_Patch:
		g.mkPatchOp(path.Patch, method)
	}
}

func (g *Generator) mkGetOp(path string, method *protogen.Method) {
	pathParams := g.mkPathParams(path, method.Input.Desc)
	queryParams := g.mkQueryParams(path, method.Input.Desc)
	op := g.mkOp(method)
	op.AddParameters(pathParams...)
	op.AddParameters(queryParams...)
	g.spec.AddPathItem(path, ogen.NewPathItem().SetGet(op))
}

func (g *Generator) mkPutOp(path string, method *protogen.Method) {
	pathParams := g.mkPathParams(path, method.Input.Desc)
	reqBody := g.mkReqBody(path, method.Input.Desc)
	op := g.mkOp(method)
	op.AddParameters(pathParams...)
	op.SetRequestBody(reqBody)
	g.spec.AddPathItem(path, ogen.NewPathItem().SetPut(op))
}

func (g *Generator) mkPostOp(path string, method *protogen.Method) {
	reqBody := g.mkReqBody(path, method.Input.Desc)
	op := g.mkOp(method)
	op.SetRequestBody(reqBody)
	g.spec.AddPathItem(path, ogen.NewPathItem().SetPost(op))
}

func (g *Generator) mkDeleteOp(path string, method *protogen.Method) {
	pathParams := g.mkPathParams(path, method.Input.Desc)
	op := g.mkOp(method)
	op.SetParameters(pathParams)
	g.spec.AddPathItem(path, ogen.NewPathItem().SetDelete(op))
}

func (g *Generator) mkPatchOp(path string, method *protogen.Method) {
	pathParams := g.mkPathParams(path, method.Input.Desc)
	reqBody := g.mkReqBody(path, method.Input.Desc)
	op := g.mkOp(method)
	op.AddParameters(pathParams...)
	op.SetRequestBody(reqBody)
	g.spec.AddPathItem(path, ogen.NewPathItem().SetPatch(op))
}

func (g *Generator) mkOp(method *protogen.Method) *ogen.Operation {
	opID := mkOpID(method.Desc)
	respName := string(method.Output.Desc.Name())
	ref := respRef(respName)
	return ogen.NewOperation().
		SetOperationID(opID).
		SetResponses(ogen.Responses{
			"200": ogen.NewResponse().SetRef(ref),
		})
}

func (g *Generator) mkReqBody(path string, md protoreflect.MessageDescriptor) *ogen.RequestBody {
	name := naming.CamelCase(string(md.Name()))
	ref := reqBodyRef(name)
	g.spec.AddRequestBody(name, ogen.NewRequestBody().SetContent(g.mkReqBodyContent(path, md)))
	return ogen.NewRequestBody().SetRef(ref)
}

func (g *Generator) mkReqBodyContent(path string, md protoreflect.MessageDescriptor) map[string]ogen.Media {
	if md.Fields().Len() == 0 {
		return map[string]ogen.Media{
			"application/json": {},
		}
	}

	curlyBracketsWords := curlyBracketsWords(path)
	isPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return isPathParam
	}

	props := make(ogen.Properties, 0, md.Fields().Len())
	r := make([]string, 0)

	for i := 0; i < md.Fields().Len(); i++ {
		field := md.Fields().Get(i)
		name := string(field.Name())
		if isPathParam(name) {
			continue
		}
		prop := mkProperty(field)
		props = append(props, prop)
		if !prop.Schema.Nullable {
			r = append(r, field.JSONName())
		}
	}

	return map[string]ogen.Media{
		"application/json": {
			Schema: ogen.NewSchema().SetProperties(&props).SetRequired(r),
		},
	}
}

func (g *Generator) mkPathParams(path string, md protoreflect.MessageDescriptor) []*ogen.Parameter {
	curlyBracketsWords := curlyBracketsWords(path)

	isNotPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return !isPathParam
	}

	pathParams := make([]*ogen.Parameter, 0, len(curlyBracketsWords))

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)

		pathName := field.TextName()

		if isNotPathParam(pathName) {
			continue
		}

		ref := paramRef(naming.CamelCase(pathName))
		pathParams = append(pathParams, ogen.NewParameter().SetRef(ref))

		g.mkParam("path", field)
	}

	return pathParams
}

func (g *Generator) mkQueryParams(path string, md protoreflect.MessageDescriptor) []*ogen.Parameter {
	curlyBracketsWords := curlyBracketsWords(path)

	isPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return isPathParam
	}

	queryParams := make([]*ogen.Parameter, 0)

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)

		pathName := field.TextName()

		if isPathParam(pathName) {
			continue
		}

		ref := paramRef(naming.CamelCase(pathName))
		queryParams = append(queryParams, ogen.NewParameter().SetRef(ref))

		g.mkParam("query", field)
	}

	return queryParams
}

func (g *Generator) mkParam(in string, fd protoreflect.FieldDescriptor) {
	name := fd.TextName()
	s := typ(fd)
	isRequired := !s.Nullable
	param := ogen.NewParameter().
		SetIn(in).
		SetName(name).
		SetSchema(s).
		SetRequired(isRequired)

	g.spec.AddParameter(naming.CamelCase(name), param)
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
		prop := mkProperty(field.Desc)
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

func mkOpID(methodDescriptor protoreflect.MethodDescriptor) string {
	name := string(methodDescriptor.Name())
	return naming.LowerCamelCase(name)
}

func mkProperty(fieldDescriptor protoreflect.FieldDescriptor) ogen.Property {
	name := fieldDescriptor.JSONName()
	schema := mkPropertySchema(fieldDescriptor)

	return ogen.Property{
		Name:   name,
		Schema: schema,
	}
}

func mkPropertySchema(fieldDescriptor protoreflect.FieldDescriptor) *ogen.Schema {
	s := ogen.NewSchema()

	switch fieldDescriptor.Cardinality() {
	case protoreflect.Optional:
		s = typ(fieldDescriptor)

	case protoreflect.Repeated:
		typName := typName(fieldDescriptor)
		ref := respRef(typName)
		s.SetType("array").SetItems(ogen.NewSchema().SetRef(ref))
	}

	return s
}

func typ(fieldDescriptor protoreflect.FieldDescriptor) *ogen.Schema {
	typName := typName(fieldDescriptor)
	s := ogen.NewSchema()

	switch typName {
	case "bool":
		return s.SetType("boolean")

	case "bytes":
		return s.SetType("string").SetFormat("binary")

	case "double":
		return s.SetType("number").SetFormat("double")

	case "enum":
		enum := enum(fieldDescriptor.Enum())
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

func enum(enumDescriptor protoreflect.EnumDescriptor) []json.RawMessage {
	values := enumDescriptor.Values()
	enum := make([]json.RawMessage, 0, values.Len())
	for i := 0; i < values.Len(); i++ {
		val := []byte(values.Get(i).Name())
		enum = append(enum, val)
	}
	return enum
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

func typName(fieldDescriptor protoreflect.FieldDescriptor) string {
	switch {
	case fieldDescriptor.Message() != nil:
		fullName := string(fieldDescriptor.Message().FullName())
		return fullName

	default:
		return fieldDescriptor.Kind().String()
	}
}
