package naming

import (
	"strings"
	"unicode"
)

// LowerCamelCase returns string as lowerCamelCase.
func LowerCamelCase(s string) string {
	if s == "" {
		return ""
	}
	first := rune(s[0])
	if unicode.IsLower(first) {
		return s
	}
	var lowerCamelCase strings.Builder
	lowerCamelCase.WriteRune(unicode.ToLower(first))
	lowerCamelCase.WriteString(s[1:])
	return lowerCamelCase.String()
}
