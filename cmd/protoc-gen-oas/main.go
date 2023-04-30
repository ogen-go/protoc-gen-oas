package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/go-faster/errors"
)

func run() error {
	r := bufio.NewReader(os.Stdin)

	input, err := io.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "read stdin")
	}

	req := new(pluginpb.CodeGeneratorRequest)

	if err := proto.Unmarshal(input, req); err != nil {
		return errors.Wrap(err, "unmarshal code generator request")
	}

	// TODO(sashamelentyev): add stdout or file writer.

	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
