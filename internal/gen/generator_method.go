package gen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/go-faster/errors"
	"github.com/ogen-go/ogen"
)

// NewMethod returns Method instance.
func NewMethod(m *protogen.Method) (*Method, error) {
	httpRule, err := NewHTTPRule(m.Desc.Options())
	if err != nil {
		return nil, err
	}

	n := string(m.Desc.Name())

	req, err := NewMessage(m.Input)
	if err != nil {
		return nil, err
	}

	resp, err := NewMessage(m.Output)
	if err != nil {
		return nil, err
	}

	return &Method{
		Name:     NewName(n),
		HTTPRule: httpRule,
		Request:  req,
		Response: resp,
	}, nil
}

// NewMethods returns Methods instance.
func NewMethods(ms []*protogen.Method) (Methods, error) {
	methods := make(Methods, 0, len(ms))

	for _, m := range ms {
		switch method, err := NewMethod(m); {
		case err == nil: // if NO error
			methods = append(methods, method)

		case errors.Is(err, ErrNotImplHTTPRule):
			// skip

		default:
			return nil, err
		}
	}

	return methods, nil
}

// Method instance.
type Method struct {
	Name     Name
	HTTPRule *HTTPRule
	Request  *Message
	Response *Message
}

// Path returns HTTPRule.Path.
func (m *Method) Path() string { return m.HTTPRule.Path }

// PathParams returns parameters with ref only.
func (m *Method) PathParams() []*ogen.Parameter {
	pathParams := make([]*ogen.Parameter, 0, len(m.PathParamsFields()))

	for _, field := range m.PathParamsFields() {
		ref := paramRef(field.Name.CamelCase())
		pathParams = append(pathParams, ogen.NewParameter().SetRef(ref))
	}

	return pathParams
}

// PathParamsFields returns path params fields.
func (m *Method) PathParamsFields() Fields {
	curlyBracketsWords := curlyBracketsWords(m.Path())

	isNotPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return !isPathParam
	}

	fields := make(Fields, 0, len(m.Request.Fields))

	for _, field := range m.Request.Fields {
		if isNotPathParam(field.Name.String()) {
			continue
		}

		fields = append(fields, field)
	}

	return fields
}

// Op returns *ogen.Operation.
func (m *Method) Op() *ogen.Operation {
	respName := m.Response.Name.String()
	ref := respRef(respName)

	return ogen.NewOperation().
		SetOperationID(m.Name.LowerCamelCase()).
		SetResponses(ogen.Responses{
			"200": ogen.NewResponse().SetRef(ref),
		})
}

// Methods is Method slice instance.
type Methods []*Method
