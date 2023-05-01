package naming

import (
	"strings"
	"unicode"
)

// CamelCase returns string as CamelCase.
func CamelCase(s string) string {
	if s == "" {
		return ""
	}
	first := rune(s[0])
	if unicode.IsUpper(first) {
		return s
	}
	var camelCase strings.Builder
	camelCase.WriteRune(unicode.ToUpper(first))
	camelCase.WriteString(s[1:])
	return camelCase.String()
}

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
