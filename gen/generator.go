package gen

import (
	"fmt"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"

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

	g.mkPaths()

	return g, nil
}

// Generator instance.
type Generator struct {
	methods  []*protogen.Method
	messages []*protogen.Message
	spec     *ogen.Spec
}

func (g *Generator) mkPaths() {
	for _, method := range g.methods {
		ext := proto.GetExtension(method.Desc.Options(), annotations.E_Http)
		httpRule, ok := ext.(*annotations.HttpRule)
		if !ok {
			continue
		}

		switch path := httpRule.Pattern.(type) {
		case *annotations.HttpRule_Get:
			g.mkGetOp(path.Get, method)

		case *annotations.HttpRule_Put:
		case *annotations.HttpRule_Post:
		case *annotations.HttpRule_Delete:
		case *annotations.HttpRule_Patch:
		}
	}
}

func (g *Generator) mkGetOp(path string, method *protogen.Method) {
	opID := string(method.Desc.Name())
	op := ogen.NewOperation().SetOperationID(opID)
	op.SetResponses(ogen.Responses{
		"200": ogen.NewResponse().
			SetDescription("Success response").
			SetRef(fmt.Sprintf("#/components/responses/%s", method.Output.Desc.Name())),
	})
	g.spec.AddPathItem(path, ogen.NewPathItem().SetGet(op))
}
