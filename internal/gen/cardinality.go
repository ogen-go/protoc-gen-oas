//go:generate go run golang.org/x/tools/cmd/stringer -type=Cardinality -linecomment

package gen

// Cardinality is field cardinality.
type Cardinality uint

const (
	_                   Cardinality = iota
	CardinalityOptional             // optional
	CardinalityRequired             // required
	CardinalityRepeated             // repeated
)

// NewCardinality returns Cardinality by string.
func NewCardinality(c string) Cardinality {
	switch c {
	case CardinalityOptional.String():
		return CardinalityOptional

	case CardinalityRequired.String():
		return CardinalityRequired

	case CardinalityRepeated.String():
		return CardinalityRepeated

	default:
		return 0
	}
}
