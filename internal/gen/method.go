package gen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/go-faster/errors"
	"github.com/ogen-go/ogen"
)

// NewMethod returns Method instance.
func NewMethod(httpRule *HTTPRule, m *protogen.Method) (*Method, error) {
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
	methods := make(Methods, 0)

	for _, m := range ms {
		httpRules, err := NewHTTPRules(m.Desc.Options())
		switch {
		case errors.Is(err, ErrNotImplHTTPRule): // skip
			continue

		case err != nil:
			return nil, err
		}

		for _, httpRule := range httpRules {
			method, err := NewMethod(httpRule, m)
			if err != nil {
				return nil, err
			}
			methods = append(methods, method)
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

// Path return path with lowerCamelCased params.
func (m *Method) Path() string {
	splitPath := strings.Split(m.HTTPRule.Path[1:], "/")
	var res strings.Builder
	for _, p := range splitPath {
		res.WriteRune('/')
		if len(p) > 2 && p[0] == '{' && p[len(p)-1] == '}' {
			res.WriteString(LowerCamelCase(p))
		} else {
			res.WriteString(p)
		}
	}
	return res.String()
}

// Body returns HTTPRule.Body.
func (m *Method) Body() string { return m.HTTPRule.Body }

// Request returns request data by Body data.
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

	op := ogen.NewOperation()
	if !m.HTTPRule.Additional {
		op.SetOperationID(m.Name.LowerCamelCase())
	}
	return op.
		SetParameters(m.parameters()).
		SetResponses(ogen.Responses{
			"200": ogen.NewResponse().SetRef(ref),
		})
}

// Methods is Method slice instance.
type Methods []*Method
