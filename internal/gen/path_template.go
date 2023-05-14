package gen

import (
	"fmt"
	"strings"
)

type pathTemplate struct {
	// Path is a OpenAPI path template.
	Path []PathSegment
}

// PathSegment is a OpenAPI path segment.
type PathSegment struct {
	Raw   string
	Param string
}

// IsParam whether is segement defines a path parameter.
func (p PathSegment) IsParam() bool {
	return p.Param != ""
}

// InvalidPathTemplateError is a path template parsing error.
type InvalidPathTemplateError struct {
	// Msg is error message.
	Msg string
	// At is byte position.
	At int
}

// Error implements error.
func (e *InvalidPathTemplateError) Error() string {
	return fmt.Sprintf("at %d: %s", e.At, e.Msg)
}

func parsePathTemplate(tmpl string) (p pathTemplate, _ error) {
	// BNF for template syntax:
	//
	// 	Template = "/" Segments [ Verb ] ;
	// 	Segments = Segment { "/" Segment } ;
	// 	Segment  = "*" | "**" | LITERAL | Variable ;
	// 	Variable = "{" FieldPath [ "=" Segments ] "}" ;
	// 	FieldPath = IDENT { "." IDENT } ;
	// 	Verb     = ":" LITERAL ;
	//
	errAt := func(i int, msg string) error {
		return &InvalidPathTemplateError{At: i, Msg: msg}
	}

	if !strings.HasPrefix(tmpl, "/") {
		return p, errAt(0, "path template must start with '/'")
	}

	// Note that iteration starts from 1.
	var (
		i     = 1
		param bool

		names = map[string]struct{}{}
	)
	for {
		if param {
			endIdx := strings.IndexAny(tmpl[i:], "=}/")
			if endIdx < 0 {
				return p, errAt(len(tmpl), "missing '}'")
			}
			endIdx += i

			// TODO(tdakkota): probably, we can handle some, e.g. /api/v1/{id=users/*}
			if tmpl[endIdx] == '=' {
				return p, errAt(endIdx, "subsegements are unsupported")
			}

			name := tmpl[i:endIdx]
			if _, ok := names[name]; ok {
				return p, errAt(i, fmt.Sprintf("parameter %q mapped second time", name))
			}
			names[name] = struct{}{}
			p.Path = append(p.Path, PathSegment{Param: name})

			// Consume '}'
			i = endIdx + 1
			param = false
		} else {
			endIdx := strings.IndexAny(tmpl[i:], "*{")
			if endIdx < 0 {
				// End of template.
				if remaining := tmpl[i:]; remaining != "" {
					p.Path = append(p.Path, PathSegment{Raw: remaining})
				}
				break
			}
			if raw := tmpl[i : i+endIdx]; raw != "" {
				p.Path = append(p.Path, PathSegment{Raw: raw})
			}
			i += endIdx

			if tmpl[i] == '*' {
				return p, errAt(i, "wildcard patterns are unsupported")
			}

			// Consume '{'
			i++
			param = true
		}
	}
	return p, nil
}
