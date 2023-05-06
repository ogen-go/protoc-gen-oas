package gen

import (
	"encoding/json"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/ogen-go/ogen"
)

// NewField returns Field instance.
func NewField(f *protogen.Field) (*Field, error) {
	return &Field{
		Name:        NewName(string(f.Desc.Name())),
		Cardinality: NewCardinality(f.Desc.Cardinality().String()),
		Type:        NewFieldType(f.Desc),
	}, nil
}

// NewFields returns Fields instance.
func NewFields(fs []*protogen.Field) (Fields, error) {
	fields := make(Fields, 0, len(fs))

	for _, f := range fs {
		field, err := NewField(f)
		if err != nil {
			return nil, err
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// Field instance.
type Field struct {
	Name        Name
	Cardinality Cardinality
	Type        *FieldType
}

// Fields is Field slice instance.
type Fields []*Field

// NewFieldType returns FieldType instance.
func NewFieldType(fd protoreflect.FieldDescriptor) *FieldType {
	typ := fd.Kind().String()
	if fd.Message() != nil {
		typ = string(fd.Message().FullName())
	}

	switch typ {
	case "bool":
		return &FieldType{Type: "boolean"}

	case "bytes":
		return &FieldType{Type: "string", Format: "binary"}

	case "double":
		return &FieldType{Type: "number", Format: "double"}

	case "enum":
		return &FieldType{Type: "string", Enum: fd.Enum()}

	case "int32":
		return &FieldType{Type: "integer", Format: "int32"}

	case "google.protobuf.DoubleValue":
		return &FieldType{Type: "number", Format: "double", Null: true}

	case "google.protobuf.StringValue":
		return &FieldType{Type: "string", Null: true}

	case "google.protobuf.Timestamp":
		return &FieldType{Type: "string", Format: "date-time"}

	default:
		return &FieldType{Type: typ}
	}
}

// FieldType instance.
type FieldType struct {
	Type   string
	Format string
	Null   bool
	Enum   protoreflect.EnumDescriptor
}

// Schema returns *ogen.Schema filled by FieldType.
func (ft *FieldType) Schema() *ogen.Schema {
	return ogen.NewSchema().
		SetType(ft.Type).
		SetFormat(ft.Format).
		SetNullable(ft.Null).
		SetEnum(enum(ft.Enum))
}

func enum(ed protoreflect.EnumDescriptor) []json.RawMessage {
	if ed == nil {
		return nil
	}

	values := ed.Values()
	enum := make([]json.RawMessage, 0, values.Len())

	for i := 0; i < values.Len(); i++ {
		val := []byte(values.Get(i).Name())
		enum = append(enum, val)
	}

	return enum
}
