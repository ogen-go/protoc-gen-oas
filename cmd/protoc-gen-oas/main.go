package main

import (
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/go-faster/errors"

	"github.com/ogen-go/protoc-gen-oas/gen"
)

func run() error {
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	openapi := set.String("openapi", "3.0.0", "OpenAPI version")
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
		_, _ = gen.NewGenerator(
			plugin.Request,
			gen.WithSpecOpenAPI(*openapi),
			gen.WithSpecInfoTitle(*title),
			gen.WithSpecInfoDescription(*description),
			gen.WithSpecInfoVersion(*version),
		)

		// TODO(sashamelentyev): add stdout or file writer.

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
