package gen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/ogen-go/ogen"
)

// NewGenerator returns new Generator instance.
func NewGenerator(protoFiles []*protogen.File, opts ...GeneratorOption) (*Generator, error) {
	g := Generator{
		protoFiles: protoFiles,
		spec:       ogen.NewSpec(),
	}

	for _, opt := range opts {
		opt(&g)
	}

	return &g, nil
}

// Generator instance.
type Generator struct {
	protoFiles []*protogen.File
	spec       *ogen.Spec
}
