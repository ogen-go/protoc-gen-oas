package gen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parsePathTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		output  pathTemplate
		wantErr string
	}{
		{"", pathTemplate{}, "at 0: path template must start with '/'"},
		{"f", pathTemplate{}, "at 0: path template must start with '/'"},

		{"/*", pathTemplate{}, "at 1: wildcard patterns are unsupported"},
		{"/foo/*", pathTemplate{}, "at 5: wildcard patterns are unsupported"},
		{"/foo/**", pathTemplate{}, "at 5: wildcard patterns are unsupported"},
		{"/api/v1/{repo}/*", pathTemplate{}, "at 15: wildcard patterns are unsupported"},

		{"/api/v1/{repo=*}", pathTemplate{}, "at 13: subsegments are unsupported"},
		{"/api/v1/{repo=**}", pathTemplate{}, "at 13: subsegments are unsupported"},
		{"/api/v1/{repo=/**}", pathTemplate{}, "at 13: subsegments are unsupported"},
		{"/api/v1/{repo=/issues}", pathTemplate{}, "at 13: subsegments are unsupported"},

		{"/api/v1/{repo", pathTemplate{}, "at 13: missing '}'"},

		{"/api/v1/{repo}/{repo}/issues", pathTemplate{}, `at 16: parameter "repo" mapped second time`},

		{
			"/",
			pathTemplate{
				Path: nil,
			},
			"",
		},
		{
			"/{repo}",
			pathTemplate{
				Path: []PathSegment{
					{Param: "repo"},
				},
			},
			"",
		},
		{
			"/api/v1/{repo}/{owner}/issues",
			pathTemplate{
				Path: []PathSegment{
					{Raw: "api/v1/"},
					{Param: "repo"},
					{Raw: "/"},
					{Param: "owner"},
					{Raw: "/issues"},
				},
			},
			"",
		},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			a := require.New(t)

			gotP, err := parsePathTemplate(tt.input)
			if e := tt.wantErr; e != "" {
				a.EqualError(err, e, tt.wantErr)
				return
			}
			a.Equal(tt.output, gotP)
		})
	}
}

func FuzzParsePathTemplate(f *testing.F) {
	for _, s := range []string{
		"",
		"/",
		"/{}",
	} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, tmpl string) {
		_, _ = parsePathTemplate(tmpl)
	})
}
