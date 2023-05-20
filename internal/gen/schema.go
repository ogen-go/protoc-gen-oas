package gen

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/go-faster/errors"

	"github.com/ogen-go/ogen"
)

func (g *Generator) mkEnum(e *protogen.Enum) error {
	s := &ogen.Schema{
		Type: "string",
		Enum: enum(e.Desc),
	}

	name := descriptorName(e.Desc)
	g.spec.AddSchema(name, s)
	return nil
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

func (g *Generator) mkSchema(msg *protogen.Message) error {
	s := ogen.NewSchema().SetType("object")

	if err := g.mkJSONFields(s, msg.Fields); err != nil {
		return err
	}

	for _, field := range msg.Fields {
		if field.Desc.HasPresence() || !field.Desc.HasPresence() && field.Message == nil {
			continue
		}
		if err := g.mkSchema(field.Message); err != nil {
			return err
		}
	}

	for _, m := range msg.Messages {
		if err := g.mkSchema(m); err != nil {
			return err
		}
	}

	for _, e := range msg.Enums {
		if err := g.mkEnum(e); err != nil {
			return err
		}
	}

	name := descriptorName(msg.Desc)
	g.spec.AddSchema(name, s)
	return nil
}

func (g *Generator) mkJSONFields(s *ogen.Schema, fields []*protogen.Field) error {
	for _, f := range fields {
		propSchema, err := g.mkFieldSchema(f.Desc)
		if err != nil {
			return errors.Wrapf(err, "make field %q", f.Desc.FullName())
		}

		prop := ogen.Property{
			Name:   f.Desc.JSONName(),
			Schema: propSchema,
		}
		if isFieldRequired(f) {
			s.AddRequiredProperties(&prop)
		} else {
			s.AddOptionalProperties(&prop)
		}
	}
	return nil
}

func (g *Generator) mkFieldSchema(fd protoreflect.FieldDescriptor) (s *ogen.Schema, rerr error) {
	defer func() {
		if rerr != nil {
			return
		}

		if fd.IsList() {
			s = ogen.NewSchema().
				SetType("array").
				SetItems(s)
		}
	}()

	switch kind := fd.Kind(); kind {
	case protoreflect.BoolKind:
		return &ogen.Schema{Type: "boolean"}, nil

	case protoreflect.Int32Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind:
		return &ogen.Schema{Type: "integer", Format: "int32"}, nil

	case protoreflect.Uint32Kind,
		protoreflect.Fixed32Kind:
		return &ogen.Schema{Type: "integer", Format: "uint32"}, nil

	case protoreflect.Int64Kind,
		protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind:
		return &ogen.Schema{Type: "integer", Format: "int64"}, nil

	case protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind:
		return &ogen.Schema{Type: "integer", Format: "uint64"}, nil

	case protoreflect.FloatKind:
		return &ogen.Schema{Type: "number", Format: "float"}, nil
	case protoreflect.DoubleKind:
		return &ogen.Schema{Type: "number", Format: "double"}, nil

	case protoreflect.StringKind:
		return &ogen.Schema{Type: "string"}, nil
	case protoreflect.BytesKind:
		// Go's protojson encodes binary data as base64 string.
		//
		//	https://github.com/protocolbuffers/protobuf-go/blob/f221882bfb484564f1714ae05f197dea2c76898d/encoding/protojson/encode.go#L287-L288
		//
		// Do the same here.
		return &ogen.Schema{Type: "string", Format: "base64"}, nil

	case protoreflect.EnumKind:
		return &ogen.Schema{Ref: descriptorRef(fd.Enum())}, nil

	case protoreflect.MessageKind:
		msg := fd.Message()

		wkt, ok, err := g.mkWellKnownPrimitive(msg)
		switch {
		case err != nil:
			// Unsupported well-known type.
			return nil, err
		case ok:
			// Well-known type.
			return wkt, nil
		default:
			if fd.IsMap() {
				if keyKind := fd.MapKey().Kind(); keyKind != protoreflect.StringKind {
					return nil, errors.Errorf("unsupported map key kind: %s", keyKind)
				}

				elem, err := g.mkFieldSchema(fd.MapValue())
				if err != nil {
					return nil, errors.Wrap(err, "make map key")
				}
				s = ogen.NewSchema().
					SetType("object")
				s.AdditionalProperties = &ogen.AdditionalProperties{
					Schema: *elem,
				}
				return s, nil
			}

			// User-defined type.
			return &ogen.Schema{Ref: descriptorRef(msg)}, nil
		}
	default: // protoreflect.GroupKind
		return nil, errors.Errorf("unsupported kind: %s", kind)
	}
}

func (g *Generator) mkWellKnownPrimitive(msg protoreflect.MessageDescriptor) (s *ogen.Schema, ok bool, _ error) {
	switch msg.FullName().Parent() {
	case "google.protobuf":
		switch msg.Name() {
		case "BoolValue":
			return &ogen.Schema{Type: "boolean", Nullable: true}, true, nil

		case "Int32Value":
			return &ogen.Schema{Type: "integer", Format: "int32", Nullable: true}, true, nil
		case "UInt32Value":
			return &ogen.Schema{Type: "integer", Format: "uint32", Nullable: true}, true, nil

		case "Int64Value":
			return &ogen.Schema{Type: "integer", Format: "int64", Nullable: true}, true, nil
		case "UInt64Value":
			return &ogen.Schema{Type: "integer", Format: "uint64", Nullable: true}, true, nil

		case "FloatValue":
			return &ogen.Schema{Type: "number", Format: "float", Nullable: true}, true, nil
		case "DoubleValue":
			return &ogen.Schema{Type: "number", Format: "double", Nullable: true}, true, nil

		case "StringValue":
			return &ogen.Schema{Type: "string", Nullable: true}, true, nil
		case "BytesValue":
			// Go's protojson encodes binary data as base64 string.
			//
			//	https://github.com/protocolbuffers/protobuf-go/blob/f221882bfb484564f1714ae05f197dea2c76898d/encoding/protojson/encode.go#L287-L288
			//
			// Do the same here.
			return &ogen.Schema{Type: "string", Format: "base64", Nullable: true}, true, nil

		case "Duration":
			return &ogen.Schema{Type: "string", Format: "duration"}, true, nil
		case "Timestamp":
			// FIXME(tdakkota): protojson uses RFC 3339
			return &ogen.Schema{Type: "string", Format: "date-time"}, true, nil
		case "Any",
			"Value",
			"NullValue",
			"ListValue",
			"Struct":
			return nil, false, errors.New("dynamic values are unsupported yet")
		}
	case "google.api":
		if msg.Name() == "HttpBody" {
			// See https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/httpbody_messages
			// for sematic details.
			return nil, false, errors.New("HttpBody is unsupported yet")
		}
	}
	return nil, false, nil
}

func isFieldRequired(f *protogen.Field) bool {
	return f.Desc.Cardinality() == protoreflect.Required
}

type descriptor interface {
	ParentFile() protoreflect.FileDescriptor
	FullName() protoreflect.FullName
}

func descriptorName[D descriptor](d D) string {
	pkgName := d.ParentFile().FullName()
	fullName := d.FullName()
	// Trim package name.
	name := strings.TrimPrefix(string(fullName), string(pkgName))
	// Trim dot between package name and type name.
	name = strings.TrimPrefix(name, ".")
	return name
}

func descriptorRef[D descriptor](d D) string {
	return schemaRef(descriptorName(d))
}

func schemaRef(s string) string {
	return fmt.Sprintf("#/components/schemas/%s", s)
}
