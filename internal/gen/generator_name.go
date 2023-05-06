package gen

import "github.com/ogen-go/protoc-gen-oas/internal/naming"

// NewName returns Name instance.
func NewName(s string) Name { return Name(s) }

// Name instance.
type Name string

// String implements fmt.Stringer.
func (n Name) String() string { return string(n) }

// LowerCamelCase returns lowerCamelCased Name.
func (n Name) LowerCamelCase() string { return naming.LowerCamelCase(n.String()) }

// CamelCase returns CamelCased Name.
func (n Name) CamelCase() string { return naming.CamelCase(n.String()) }
