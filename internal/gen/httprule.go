package gen

import (
	"net/http"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/go-faster/errors"
)

// ErrNotImplHTTPRule reports that options not implements *annotations.HttpRule or nil.
var ErrNotImplHTTPRule = errors.New("not implements *annotations.HttpRule or nil")

// NewHTTPRules returns HTTPRule slice.
func NewHTTPRules(opts protoreflect.ProtoMessage) ([]*HTTPRule, error) {
	httpRule, err := httpRule(opts)
	if err != nil {
		return nil, err
	}

	var (
		rules     []*HTTPRule
		walkRules func(rule *annotations.HttpRule, additional bool)
	)
	walkRules = func(rule *annotations.HttpRule, additional bool) {
		if rule == nil {
			return
		}
		rules = append(rules, &HTTPRule{
			Method:     method(rule),
			Path:       path(rule),
			Body:       rule.Body,
			Additional: additional,
		})
		for _, binding := range rule.AdditionalBindings {
			walkRules(binding, true)
		}
	}
	walkRules(httpRule, false)

	return rules, nil
}

// HTTPRule instance.
type HTTPRule struct {
	Method     string
	Path       string
	Body       string
	Additional bool
}

func httpRule(opts protoreflect.ProtoMessage) (*annotations.HttpRule, error) {
	ext := proto.GetExtension(opts, annotations.E_Http)
	httpRule, ok := ext.(*annotations.HttpRule)
	if !ok || httpRule == nil {
		return nil, ErrNotImplHTTPRule
	}

	return httpRule, nil
}

func method(httpRule *annotations.HttpRule) string {
	switch httpRule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet

	case *annotations.HttpRule_Put:
		return http.MethodPut

	case *annotations.HttpRule_Post:
		return http.MethodPost

	case *annotations.HttpRule_Delete:
		return http.MethodDelete

	case *annotations.HttpRule_Patch:
		return http.MethodPatch

	default:
		return ""
	}
}

func path(httpRule *annotations.HttpRule) string {
	switch pattern := httpRule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		return pattern.Get

	case *annotations.HttpRule_Put:
		return pattern.Put

	case *annotations.HttpRule_Post:
		return pattern.Post

	case *annotations.HttpRule_Delete:
		return pattern.Delete

	case *annotations.HttpRule_Patch:
		return pattern.Patch

	default:
		return ""
	}
}
