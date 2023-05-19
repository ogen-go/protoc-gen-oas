package gen

import (
	"net/http"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func httpRule(opts protoreflect.ProtoMessage) (r *annotations.HttpRule, ok bool) {
	r, ok = proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
	ok = ok && r != nil
	return r, ok
}

// HTTPRule is a parsed HTTP rule annotation.
type HTTPRule struct {
	Path         string
	Method       string
	Body         string
	ResponseBody string
	Additional   bool
}

func collectRules(opts protoreflect.ProtoMessage) (rules []HTTPRule) {
	r, ok := httpRule(opts)
	if !ok {
		return nil
	}

	var walkRules func(rule *annotations.HttpRule, additional bool)
	walkRules = func(rule *annotations.HttpRule, additional bool) {
		if rule == nil {
			return
		}
		rules = append(rules, HTTPRule{
			Method:     method(rule),
			Path:       path(rule),
			Body:       rule.Body,
			Additional: additional,
		})
		for _, binding := range rule.AdditionalBindings {
			walkRules(binding, true)
		}
	}
	walkRules(r, false)

	return rules
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
