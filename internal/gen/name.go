package gen

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// NewName returns Name instance.
func NewName(s string) Name { return Name(s) }

// Name instance.
type Name string

// String implements fmt.Stringer.
func (n Name) String() string { return string(n) }

// LowerCamelCase returns lowerCamelCased Name.
func (n Name) LowerCamelCase() string { return LowerCamelCase(n.String()) }

// CamelCase returns CamelCased Name.
func (n Name) CamelCase() string { return CamelCase(n.String()) }

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

// CamelCase was copied from https://github.com/golang/protobuf/blob/v1.5.2/protoc-gen-go/generator/generator.go#L2648
// CamelCase returns the CamelCased name.
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
// There is a remote possibility of this rewrite causing a name collision,
// but it's so remote we're prepared to pretend it's nonexistent - since the
// C++ generator lowercases names, it's extremely unlikely to have two fields
// with different capitalizations.
// In short, _my_field_name_2 becomes XMyFieldName_2.
func CamelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// LowerCamelCase returns the lowerCamelCased name.
func LowerCamelCase(s string) string {
	return Decapitalize(CamelCase(s))
}

// Capitalize converts first character to upper.
//
// If the string is invalid UTF-8 or empty, it is returned as is.
func Capitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

// Decapitalize converts first character to lower.
//
// If the string is invalid UTF-8 or empty, it is returned as is.
func Decapitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToLower(r)) + s[size:]
}

// LastAfterDots returns last word from string with word split dots.
func LastAfterDots(s string) string {
	words := strings.Split(s, ".")
	return words[len(words)-1]
}
