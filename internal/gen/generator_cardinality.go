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
	case "optional":
		return CardinalityOptional

	case "required":
		return CardinalityRequired

	case "repeated":
		return CardinalityRepeated

	default:
		return 0
	}
}
