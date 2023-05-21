package gen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/ogen-go/ogen/gen/ir"
)

type packageImport struct {
	Name protogen.GoPackageName
	Path protogen.GoImportPath
}

// TemplateConfig is a mapping template config.
type TemplateConfig struct {
	Imports     []packageImport
	PackageName protogen.GoPackageName
	Mapping
}

// Mapping defines gRPC <-> OpenAPI mapping.
type Mapping struct {
	Services []ServiceMapping
	Messages []MessageMapping
	Enums    []EnumMapping
}

// ServiceMapping defines gRPC service <-> OpenAPI operations mapping.
type ServiceMapping struct {
	ProtoName   string
	ProtoServer string
	Methods     []MethodMapping
}

// MethodMapping defines gRPC method <-> OpenAPI operation mapping.
type MethodMapping struct {
	ProtoName   string
	OgenName    string
	OperationID string
	ParamsType  string
	Input       InputMapping
	Output      OutputMapping
}

// InputMapping defines method input mapping.
type InputMapping struct {
	ProtoType string
	Path      []FieldMapping
	Query     []FieldMapping
	Body      *BodyMapping
}

// HasParams returns true if operation has any parameters.
func (m InputMapping) HasParams() bool {
	return len(m.Path)+len(m.Query) > 0
}

// AllInBody returns true if whole message is inside the body.
func (m InputMapping) AllInBody() bool {
	b := m.Body
	return b != nil && b.Field == nil && !m.HasParams()
}

// BodyMapping defines body mapping
type BodyMapping struct {
	OgenType string
	Ogen     *ir.Request
	Proto    *protogen.Message
	Field    *FieldMapping
}

// PassByPointer returns true if request is passed by pointer.
func (m BodyMapping) PassByPointer() bool {
	return m.Ogen.Type.DoPassByPointer()
}

// OutputMapping defines method output mapping.
type OutputMapping struct {
	ProtoType string
	OgenType  string
	Ogen      *ir.Type
	Proto     *protogen.Message
	Field     *FieldMapping
}

// AllInBody returns true if whole message is inside the body.
func (m OutputMapping) AllInBody() bool {
	return m.Field == nil
}

// PassByPointer returns true if request is passed by pointer.
func (m OutputMapping) PassByPointer() bool {
	return m.Ogen.DoPassByPointer()
}

// FieldMapping defines mapping between protobuf and ogen type fields.
type FieldMapping struct {
	OgenName string
	OgenType *ir.Type
	Proto    *protogen.Field
}

// Elem returns elem for this field.
func (m FieldMapping) Elem() FieldElem {
	ogenSel := "om"
	if n := m.OgenName; n != "" {
		ogenSel += "." + n
	}
	return FieldElem{
		Ogen:     m.OgenType,
		Proto:    m.Proto,
		OgenSel:  ogenSel,
		ProtoSel: "pm." + m.Proto.GoName,
	}
}

// MessageMapping defines message mapping.
type MessageMapping struct {
	ProtoType string
	OgenType  string
	Fields    []FieldMapping
}

// EnumMapping defines Enum mapping.
type EnumMapping struct {
	ProtoType    string
	OgenType     string
	EnumNameMap  string
	EnumValueMap string
}
