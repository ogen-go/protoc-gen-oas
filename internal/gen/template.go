package gen

import (
	"embed"
	"fmt"
	"sync"
	"text/template"

	"github.com/go-faster/errors"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/ogen-go/ogen/gen/ir"
)

type FieldElem struct {
	Ogen     *ir.Type
	Proto    *protogen.Field
	OgenSel  string
	ProtoSel string
}

// CheckProtoForNil whether mapper should check field for nil.
func (f FieldElem) CheckProtoForNil() bool {
	return (f.Ogen.IsGeneric() && f.Ogen.GenericOf.IsMap()) ||
		(f.Ptr() && f.Ogen.Is(ir.KindPointer, ir.KindGeneric))
}

// ProtoType returns proto message or enum name.
func (f FieldElem) ProtoType() string {
	var protoType func(f *protogen.Field) string
	protoType = func(f *protogen.Field) string {
		d := f.Desc
		switch kind := d.Kind(); kind {
		case protoreflect.MessageKind:
			m := f.Message
			if d.IsMap() {
				return protoType(m.Fields[1])
			}
			return m.GoIdent.GoName
		case protoreflect.EnumKind:
			return f.Enum.GoIdent.GoName
		default:
			panic(fmt.Sprintf("unexpected kind %q", kind))
		}
	}
	return protoType(f.Proto)
}

// Ptr whether to use pointer when setting field.
func (f FieldElem) Ptr() bool {
	fdesc := f.Proto.Desc
	return fdesc.HasPresence() &&
		fdesc.Kind() != protoreflect.BytesKind
}

//go:embed _template/*
var templates embed.FS

var _templates struct {
	sync.Once
	val *template.Template
}

// vendoredTemplates parses and returns vendored code generation templates.
func vendoredTemplates() *template.Template {
	_templates.Do(func() {
		tmpl := template.New("templates").Funcs(template.FuncMap{
			"field_elem": func(
				o *ir.Type,
				p *protogen.Field,
				ogenSel string,
				protoSel string,
			) FieldElem {
				return FieldElem{
					Ogen:     o,
					Proto:    p,
					OgenSel:  ogenSel,
					ProtoSel: protoSel,
				}
			},
			"errorf": func(format string, args ...any) (any, error) {
				return nil, errors.Errorf(format, args...)
			},
		})
		tmpl = template.Must(tmpl.ParseFS(templates,
			"_template/*.tmpl",
		))
		_templates.val = tmpl
	})
	return _templates.val
}
