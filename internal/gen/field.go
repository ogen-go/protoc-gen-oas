package gen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/ogen-go/ogen"
)

// NewField returns Field instance.
func NewField(f *protogen.Field) *Field {
	return &Field{
		Name:        NewName(string(f.Desc.Name())),
		Cardinality: NewCardinality(f.Desc.Cardinality().String()),
		Type:        NewFieldType(f.Desc),
		Options:     NewFieldOptions(f.Desc.Options()),
	}
}

// NewFields returns Fields instance.
func NewFields(fs []*protogen.Field) Fields {
	fields := make(Fields, 0, len(fs))

	for _, f := range fs {
		fields = append(fields, NewField(f))
	}

	return fields
}

// Field instance.
type Field struct {
	Name        Name
	Cardinality Cardinality
	Type        *FieldType
	Options     *FieldOptions
}

// AsParameter creates a new Parameter for this field.
func (f *Field) AsParameter(in string) *ogen.Parameter {
	return ogen.NewParameter().
		SetIn(in).
		SetName(f.Name.String()).
		SetSchema(f.Type.Schema()).
		SetRequired(f.Options.IsRequired || in == "path")
}

// Fields is Field slice instance.
type Fields []*Field
