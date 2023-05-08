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

// Parameters returns path and query parameters.
func (m *Method) Parameters() []*ogen.Parameter {
	parameters := make([]*ogen.Parameter, 0)
	parameters = append(parameters, m.PathParameters()...)
	parameters = append(parameters, m.QueryParameters()...)
	return parameters
}

// PathParameters returns path parameters with ref only.
func (m *Method) PathParameters() (params []*ogen.Parameter) {
	curlyBracketsWords := curlyBracketsWords(m.Path())

	isNotPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return !isPathParam
	}

	for _, field := range m.Request.Fields {
		if isNotPathParam(field.Name.String()) {
			continue
		}
		params = append(params, field.AsParameter("path"))
	}

	return params
}

// QueryParameters returns query parameters with ref only.
func (m *Method) QueryParameters() (params []*ogen.Parameter) {
	curlyBracketsWords := curlyBracketsWords(m.Path())

	isPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return isPathParam
	}

	for _, field := range m.Request.Fields {
		if isPathParam(field.Name.String()) {
			continue
		}
		params = append(params, field.AsParameter("query"))
	}

	return params
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
