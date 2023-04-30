package gen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/ogen-go/ogen"
)

// NewGenerator returns new Generator instance.
func NewGenerator(protoFiles []*protogen.File, opts ...GeneratorOption) (*Generator, error) {
	g := new(Generator)
	g.spec = ogen.NewSpec()

	for _, file := range protoFiles {
		if isSkip := !file.Generate; isSkip {
			continue
		}

		for _, service := range file.Services {
			g.methods = append(g.methods, service.Methods...)
		}

		g.messages = append(g.messages, file.Messages...)
	}

	for _, opt := range opts {
		opt(g)
	}

	return g, nil
}

// Generator instance.
type Generator struct {
	methods  []*protogen.Method
	messages []*protogen.Message
	spec     *ogen.Spec
}
