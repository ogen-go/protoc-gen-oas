package gen

import (
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/ogen-go/ogen"
)

// NewGenerator returns new Generator instance.
func NewGenerator(req *pluginpb.CodeGeneratorRequest, opts ...GeneratorOption) (*Generator, error) {
	g := Generator{
		req:  req,
		spec: ogen.NewSpec(),
	}

	for _, opt := range opts {
		opt(&g)
	}

	return &g, nil
}

// Generator instance.
type Generator struct {
	req  *pluginpb.CodeGeneratorRequest
	spec *ogen.Spec
}
