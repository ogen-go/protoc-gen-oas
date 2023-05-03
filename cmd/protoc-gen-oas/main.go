package main

import (
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/go-faster/errors"

	"github.com/ogen-go/protoc-gen-oas/internal/gen"
)

func run() error {
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	openapi := set.String("openapi", "3.1.0", "OpenAPI version")
	title := set.String("title", "", "Title")
	description := set.String("description", "", "Description")
	version := set.String("version", "", "Version")

	if err := set.Parse(os.Args[1:]); err != nil {
		return errors.Wrap(err, "parse args")
	}

	opts := protogen.Options{
		ParamFunc: set.Set,
	}

	p := func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

		g, err := gen.NewGenerator(
			plugin.Files,
			gen.WithSpecOpenAPI(*openapi),
			gen.WithSpecInfoTitle(*title),
			gen.WithSpecInfoDescription(*description),
			gen.WithSpecInfoVersion(*version),
		)
		if err != nil {
			return err
		}

		bytes, err := g.YAML()
		if err != nil {
			return err
		}

		gf := plugin.NewGeneratedFile("openapi.yaml", "")
		if _, err := gf.Write(bytes); err != nil {
			return err
		}

		return nil
	}

	opts.Run(p)

	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
