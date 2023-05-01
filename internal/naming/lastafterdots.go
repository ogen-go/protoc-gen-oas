package naming

import "strings"

// LastAfterDots returns last word from string with word split dots.
func LastAfterDots(s string) string {
	words := strings.Split(s, ".")
	return words[len(words)-1]
}
