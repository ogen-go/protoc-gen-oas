package gen

import (
	"encoding/json"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/ogen-go/ogen"
)

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
		return &FieldType{Type: "string", Enum: enum(fd.Enum())}

	case "int32":
		return &FieldType{Type: "integer", Format: "int32"}

	case "int64":
		return &FieldType{Type: "integer", Format: "int64"}

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
	Enum   []json.RawMessage
}

// HasEnum returns true if FieldType have Enum.
func (ft *FieldType) HasEnum() bool {
	return len(ft.Enum) > 0
}

// Schema returns *ogen.Schema filled by FieldType.
func (ft *FieldType) Schema() *ogen.Schema {
	return ogen.NewSchema().
		SetType(ft.Type).
		SetFormat(ft.Format).
		SetNullable(ft.Null).
		SetEnum(ft.Enum)
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
