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
		request:  req,
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
	request  *Message
	Response *Message
}

// Path returns HTTPRule.Path.
func (m *Method) Path() string { return m.HTTPRule.Path }

// Body returns HTTPRule.Body.
func (m *Method) Body() string { return m.HTTPRule.Body }

func (m *Method) Request() *Message {
	if m.Body() == "" || m.Body() == "*" {
		return m.request
	}

	var msg *Message

	for _, field := range m.request.Fields {
		if field.Name.String() == m.Body() {
			msg = &Message{
				Name:   NewName(m.Body()),
				Fields: []*Field{field},
			}

			break
		}
	}

	return msg
}

// parameters returns path and query parameters.
func (m *Method) parameters() []*ogen.Parameter {
	parameters := make([]*ogen.Parameter, 0)
	parameters = append(parameters, m.pathParameters()...)
	parameters = append(parameters, m.queryParameters()...)
	return parameters
}

func (m *Method) pathParameters() (params []*ogen.Parameter) {
	curlyBracketsWords := curlyBracketsWords(m.Path())

	isNotPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return !isPathParam
	}

	for _, field := range m.request.Fields {
		if isNotPathParam(field.Name.String()) {
			continue
		}
		params = append(params, field.AsParameter("path"))
	}

	return params
}

func (m *Method) queryParameters() (params []*ogen.Parameter) {
	if m.HTTPRule.Body == "*" {
		return params
	}

	curlyBracketsWords := curlyBracketsWords(m.Path())

	isPathParam := func(pathName string) bool {
		_, isPathParam := curlyBracketsWords[pathName]
		return isPathParam
	}

	for _, field := range m.request.Fields {
		isBodyParam := m.HTTPRule.Body == field.Name.String()
		if isPathParam(field.Name.String()) || isBodyParam {
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
		SetParameters(m.parameters()).
		SetResponses(ogen.Responses{
			"200": ogen.NewResponse().SetRef(ref),
		})
}

// Methods is Method slice instance.
type Methods []*Method
