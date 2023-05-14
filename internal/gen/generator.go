package gen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/go-faster/errors"
	"github.com/go-faster/yaml"

	"github.com/ogen-go/ogen"
)

// ErrNoMethods reports that service have no methods.
var ErrNoMethods = errors.New("protoc-gen-oas: service has no methods")

// NewGenerator returns new Generator instance.
func NewGenerator(files []*protogen.File, opts ...GeneratorOption) (*Generator, error) {
	g := new(Generator)
	g.init()
	for _, opt := range opts {
		opt(g)
	}

	type endpoint struct {
		Rule   HTTPRule
		Method *protogen.Method
	}

	for _, f := range files {
		if !f.Generate {
			continue
		}

		for _, m := range f.Messages {
			if err := g.mkSchema(m); err != nil {
				return nil, err
			}
		}

		for _, e := range f.Enums {
			if err := g.mkEnum(e); err != nil {
				return nil, err
			}
		}

		for _, s := range f.Services {
			for _, m := range s.Methods {
				for _, rule := range collectRules(m.Desc.Options()) {
					pi := g.spec.Paths[rule.Path]
					if pi == nil {
						pi = ogen.NewPathItem()
						g.spec.AddPathItem(rule.Path, pi)
					}

					var to **ogen.Operation
					switch rule.Method {
					case http.MethodGet:
						to = &pi.Get
					case http.MethodPut:
						to = &pi.Put
					case http.MethodPost:
						to = &pi.Post
					case http.MethodDelete:
						to = &pi.Delete
					case http.MethodPatch:
						to = &pi.Patch
					}

					if *to != nil {
						return nil, errors.Errorf("conflict on endpoint %s %s", rule.Method, rule.Path)
					}

					op, err := g.mkMethod(rule, m)
					if err != nil {
						return nil, errors.Wrapf(err, "make method %s => %s %s mapping", m.Desc.FullName(), rule.Method, rule.Path)
					}

					*to = op
				}
			}
		}
	}

	return g, nil
}

// Generator instance.
type Generator struct {
	spec *ogen.Spec
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
	g.spec = ogen.NewSpec()
	g.spec.Init()
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

func (g *Generator) mkMethod(rule HTTPRule, m *protogen.Method) (*ogen.Operation, error) {
	op := ogen.NewOperation()
	if !rule.Additional {
		op.SetOperationID(string(m.Desc.Name()))
	}

	if err := g.mkInput(rule, m, op); err != nil {
		return nil, errors.Wrap(err, "make input")
	}

	if err := g.mkOutput(rule, m, op); err != nil {
		return nil, errors.Wrap(err, "make output")
	}

	return op, nil
}

func (g *Generator) mkInput(rule HTTPRule, m *protogen.Method, op *ogen.Operation) error {
	var (
		fields        = collectFields(m.Input)
		hasPathParams bool
	)
	for name := range curlyBracketsWords(rule.Path) {
		hasPathParams = true

		f, ok := fields[name]
		if !ok {
			return errors.Errorf("unknown field %q", name)
		}

		p, err := g.mkParameter("path", f.Desc.JSONName(), f)
		if err != nil {
			return err
		}
		op.AddParameters(p)

		delete(fields, name)
	}

	var s *ogen.Schema
	switch body := rule.Body; body {
	case "*":
		s = ogen.NewSchema()
		// All remaining fields are inside request body.
		if !hasPathParams {
			// Special case: all message fields are inside body, generate a direct reference to schema.
			s.SetRef(descriptorRef(m.Input.Desc))
		} else {
			// Map remaining fields.
			values := maps.Values(fields)
			// Sort to make output stable.
			slices.SortStableFunc(values, func(a, b *protogen.Field) bool {
				return a.Desc.FullName() < b.Desc.FullName()
			})
			if err := g.mkJSONFields(s, values); err != nil {
				return errors.Wrap(err, "make requestBody schema")
			}
		}
	default:
		// TODO(tdakkota): generate a requestBody component.

		// This field is body, remaining fields are query parameters.
		f, ok := fields[body]
		if !ok {
			return errors.Errorf("unknown field %q", body)
		}

		fieldSch, err := g.mkFieldSchema(f.Desc)
		if err != nil {
			return errors.Wrapf(err, "make requestBody schema (field: %q)", body)
		}
		s = fieldSch

		delete(fields, body)
		fallthrough
	case "":
		// Remaining fields are query parameters.
		if err := g.mkQueryParameters(op, fields); err != nil {
			return err
		}
	}
	if s != nil {
		op.SetRequestBody(
			ogen.NewRequestBody().SetJSONContent(s),
		)
	}
	// Sort to make output stable.
	slices.SortStableFunc(op.Parameters, func(a, b *ogen.Parameter) bool {
		if a.In != b.In {
			return a.In < b.In
		}
		return a.Name < b.Name
	})
	return nil
}

func (g *Generator) mkOutput(rule HTTPRule, m *protogen.Method, op *ogen.Operation) error {
	fields := collectFields(m.Output)

	s := ogen.NewSchema()
	switch body := rule.ResponseBody; body {
	case "", "*":
		// Map all response fields.
		s.SetRef(descriptorRef(m.Output.Desc))
	default:
		// TODO(tdakkota): generate a response component.

		// This field is body, remaining fields are omitted.
		f, ok := fields[body]
		if !ok {
			return errors.Errorf("unknown field %q", body)
		}

		fieldSch, err := g.mkFieldSchema(f.Desc)
		if err != nil {
			return errors.Wrapf(err, "make response schema (field: %q)", body)
		}
		s = fieldSch
	}
	if s != nil {
		op.SetResponses(
			ogen.Responses{
				"200": ogen.NewResponse().
					SetDescription(fmt.Sprintf("%s response", m.Desc.FullName())).
					SetJSONContent(s),
			},
		)
	}
	return nil
}

func (g *Generator) mkQueryParameters(op *ogen.Operation, fields map[string]*protogen.Field) error {
	flattenFields := make(map[string]*protogen.Field, len(fields))

	// Recursively collect and flatten message type to primitive parameters.
	//
	// For example, if path template is "/v1/messages/{message_id}":
	//
	//	 message GetMessageRequest {
	//	 	message SubMessage {
	//	 	  string subfield = 1;
	//	 	}
	//	 	string message_id = 1; // Mapped to URL path.
	//	 	int64 revision = 2;    // Mapped to URL query parameter `revision`.
	//	 	SubMessage sub = 3;    // Mapped to URL query parameter `sub.subfield`.
	//	 }
	//
	// See https://cloud.google.com/service-infrastructure/docs/service-management/reference/rpc/google.api#grpc-transcoding.
	var (
		walkFields func(prefix string, fields []*protogen.Field) error
		seen       = map[*protogen.Message]struct{}{}
	)
	walkFields = func(prefix string, fields []*protogen.Field) error {
		for _, f := range fields {
			fd := f.Desc
			name := prefix + fd.TextName()

			switch kind := fd.Kind(); kind {
			case protoreflect.MessageKind:
				if fd.IsMap() {
					return errors.New("map parameters are not supported")
				}

				_, ok, err := g.mkWellKnownPrimitive(fd.Message())
				if err != nil {
					return err
				}
				if !ok {
					msg := f.Message
					if _, ok := seen[msg]; ok {
						return errors.Errorf("query parameter cannot be recursive: field %s", name)
					}
					seen[msg] = struct{}{}

					if err := walkFields(name+".", msg.Fields); err != nil {
						return err
					}
					delete(seen, msg)
					continue
				}
			case protoreflect.GroupKind:
				return errors.Errorf("unsupported kind: %s", kind)
			}

			flattenFields[name] = f
		}
		return nil
	}
	if err := walkFields("", maps.Values(fields)); err != nil {
		return err
	}

	for name, f := range flattenFields {
		p, err := g.mkParameter("query", name, f)
		if err != nil {
			return err
		}
		op.AddParameters(p)
	}

	return nil
}

func (g *Generator) mkParameter(in, name string, f *protogen.Field) (*ogen.Parameter, error) {
	s, err := g.mkFieldSchema(f.Desc)
	if err != nil {
		return nil, errors.Wrapf(err, "generate %s parameter %q", in, f.Desc.Name())
	}

	p := ogen.NewParameter().
		SetIn(in).
		SetName(name).
		SetRequired(f.Desc.Cardinality() == protoreflect.Required).
		SetSchema(s)

	switch in {
	case "path":
		p.SetRequired(true)
	case "query":
		if s.Type == "array" {
			// Explicitly set parameter style to match transcoding spec.
			p.SetStyle("form").
				SetExplode(true)
		}
	}
	return p, nil
}

func collectFields(message *protogen.Message) (fields map[string]*protogen.Field) {
	fields = make(map[string]*protogen.Field, len(message.Fields))
	for _, f := range message.Fields {
		fields[string(f.Desc.Name())] = f
	}
	return fields
}
