package gen

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/go-faster/sdk/gold"
)

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	dirEntries, err := os.ReadDir("_golden")
	require.NoError(t, err)

	fileNames := make(map[string]struct{})

	for _, dirEntry := range dirEntries {
		n := strings.Split(dirEntry.Name(), ".")[0]
		fileNames[n] = struct{}{}
	}

	for fileName := range fileNames {
		fileName := fileName

		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			textproto, err := os.ReadFile(fmt.Sprintf("_golden/%s.textproto", fileName))
			require.NoError(t, err)

			req := new(pluginpb.CodeGeneratorRequest)
			err = prototext.Unmarshal(textproto, req)
			require.NoError(t, err)

			opts := protogen.Options{}
			p, err := opts.New(req)
			require.NoError(t, err)

			g, err := NewGenerator(p.Files, WithSpecOpenAPI("3.1.0"))
			require.NoError(t, err)

			yaml, err := g.YAML()
			require.NoError(t, err)

			json, err := g.JSON()
			require.NoError(t, err)

			gold.Str(t, string(yaml), fmt.Sprintf("%s.yaml", fileName))
			gold.Str(t, string(json), fmt.Sprintf("%s.json", fileName))
		})
	}
}
