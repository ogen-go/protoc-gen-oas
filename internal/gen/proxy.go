package gen

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/go-faster/errors"
	"golang.org/x/exp/maps"
	goimports "golang.org/x/tools/imports"
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/ogen-go/ogen/gen"
	"github.com/ogen-go/ogen/gen/ir"
	"github.com/ogen-go/ogen/jsonschema"
)

type protocFS struct {
	mux        sync.Mutex
	plugin     *protogen.Plugin
	importPath protogen.GoImportPath
	dir        string
}

func (fs *protocFS) WriteFile(baseName string, source []byte) error {
	fs.mux.Lock()
	defer fs.mux.Unlock()

	// TODO(tdakkota): make output configurable
	file := fs.plugin.NewGeneratedFile(path.Join(fs.dir, baseName), fs.importPath)
	_, err := file.Write(source)
	return err
}

// WriteProxy runs ogen and generates ogen -> gRPC proxy.
func (g *Generator) WriteProxy(plugin *protogen.Plugin) error {
	var out packageImport
	for _, f := range plugin.Files {
		if f.Generate {
			out = packageImport{
				Name: f.GoPackageName,
				Path: f.GoImportPath,
			}
			break
		}
	}

	imports := map[protogen.GoImportPath]packageImport{}
	for _, f := range plugin.Files {
		importPath := f.GoImportPath
		if out.Path != importPath {
			imports[importPath] = packageImport{
				Name: f.GoPackageName,
				Path: importPath,
			}
		}
	}

	og, err := gen.NewGenerator(g.spec, gen.Options{})
	if err != nil {
		return err
	}

	// TODO(tdakkota): make output configurable
	oasPath := out.Path + "/oas"
	fs := &protocFS{
		plugin:     plugin,
		importPath: oasPath,
		dir:        "oas",
	}
	if err := og.WriteSource(fs, "oas"); err != nil {
		return errors.Wrap(err, "write ogen files")
	}
	imports[oasPath] = packageImport{
		Name: protogen.GoPackageName(g.pkgName),
		Path: oasPath,
	}

	mapping, err := g.mapSpec(og)
	if err != nil {
		return errors.Wrap(err, "map spec")
	}

	tmpl := vendoredTemplates()
	type generatedFile struct {
		tmpl, name string
	}
	for _, file := range []generatedFile{
		{"messages", "messages.gen.go"},
		{"handler", "handler.gen.go"},
	} {
		data := TemplateConfig{
			PackageName: out.Name,
			Imports:     maps.Values(imports),
			Mapping:     mapping,
		}

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, file.tmpl, data); err != nil {
			return errors.Wrap(err, "execute template")
		}

		formatted, err := goimports.Process(file.name, buf.Bytes(), nil)
		if err != nil {
			_ = os.WriteFile(file.name+".dump", buf.Bytes(), 0o600)
			return errors.Wrap(err, "format mapping")
		}

		f := plugin.NewGeneratedFile(file.name, out.Path)
		if _, err := f.Write(formatted); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) mapSpec(og *gen.Generator) (mapping Mapping, _ error) {
	refName := func(ref jsonschema.Ref) (string, error) {
		idx := strings.LastIndexByte(ref.Ptr, '/')
		if idx < 0 {
			return "", errors.Errorf("unexpected ref: %q", ref)
		}
		return ref.Ptr[idx+1:], nil
	}

	for _, typ := range og.Types() {
		s := typ.Schema
		if s == nil || s.Ref.IsZero() {
			continue
		}

		name, err := refName(s.Ref)
		if err != nil {
			return mapping, errors.Wrapf(err, "map type %q", typ.Name)
		}
		if m, ok := g.messages[name]; ok {
			mapping.Messages = append(mapping.Messages, g.mapMessage(typ, m))
			continue
		}
		if e, ok := g.enums[name]; ok {
			mapping.Enums = append(mapping.Enums, g.mapEnum(typ, e))
			continue
		}
	}

	services := map[*protogen.Service][]MethodMapping{}
	for _, op := range og.Operations() {
		if err := func() error {
			tmpl := op.Spec.Path.String()
			ms, ok := g.ops[tmpl]
			if !ok {
				return errors.Errorf("unknown path %q", tmpl)
			}

			httpMethod := strings.ToUpper(op.Spec.HTTPMethod)
			rule, ok := ms.Methods[httpMethod]
			if !ok {
				return errors.Errorf("can't find gRPC method for %s %s", httpMethod, op.Spec.Path)
			}

			services[rule.Service] = append(services[rule.Service], g.mapMethod(op, rule))
			return nil
		}(); err != nil {
			return mapping, errors.Wrapf(err, "map operation %s", op.PrettyOperationID())
		}
	}

	for s, m := range services {
		slices.SortStableFunc(m, func(a, b MethodMapping) int {
			return cmp.Compare(a.ProtoName, b.ProtoName)
		})
		mapping.Services = append(mapping.Services, ServiceMapping{
			ProtoName:   s.GoName,
			ProtoServer: s.GoName + "Server",
			Methods:     m,
		})
	}

	// Ensure output is stable.
	slices.SortStableFunc(mapping.Messages, func(a, b MessageMapping) int {
		return cmp.Compare(a.ProtoType, b.ProtoType)
	})
	slices.SortStableFunc(mapping.Enums, func(a, b EnumMapping) int {
		return cmp.Compare(a.ProtoType, b.ProtoType)
	})
	slices.SortStableFunc(mapping.Services, func(a, b ServiceMapping) int {
		return cmp.Compare(a.ProtoName, b.ProtoName)
	})
	return mapping, nil
}

func qualifiedOgenType(pkg, typ string) string {
	var prefix string
loop:
	for i, c := range []byte(typ) {
		switch c {
		case '*', '[', ']':
		default:
			prefix = typ[:i]
			typ = typ[i:]
			break loop
		}
	}
	return prefix + pkg + "." + typ
}

func (g *Generator) mapMethod(ogenOp *ir.Operation, mr methodRule) MethodMapping {
	input := g.mapInput(mr.Rule.Body, ogenOp, mr.Method.Input)
	output := OutputMapping{
		ProtoType: mr.Method.Output.GoIdent.GoName,
		OgenType:  qualifiedOgenType(g.pkgName, ogenOp.Responses.GoType()),
		Ogen:      ogenOp.Responses.Type,
		Proto:     mr.Method.Output,
	}
	if b := mr.Rule.ResponseBody; b != "" && b != "*" {
		output.Field = g.mapSelector(b, ogenOp.Responses.Type, mr.Method.Output)
	}
	m := MethodMapping{
		ProtoName:   mr.Method.GoName,
		OgenName:    ogenOp.Name,
		OperationID: ogenOp.Spec.OperationID,
		ParamsType:  qualifiedOgenType(g.pkgName, ogenOp.Name+"Params"),
		Input:       input,
		Output:      output,
	}
	return m
}

func (g *Generator) mapInput(bodySel string, ogenOp *ir.Operation, msg *protogen.Message) (input InputMapping) {
	input.ProtoType = msg.GoIdent.GoName

	pathParams := map[string]*ir.Parameter{}
	for _, p := range ogenOp.Params {
		if p.Spec.In.Path() {
			pathParams[p.Spec.Name] = p
			continue
		}
	}

	for _, f := range msg.Fields {
		jsonName := f.Desc.JSONName()
		p, ok := pathParams[jsonName]
		if !ok {
			continue
		}
		input.Path = append(input.Path, FieldMapping{
			OgenName: p.Name,
			OgenType: p.Type,
			Proto:    f,
		})
	}

	if req := ogenOp.Request; req != nil {
		body := &BodyMapping{
			OgenType: qualifiedOgenType(g.pkgName, req.GoType()),
			Proto:    msg,
			Ogen:     req,
		}
		if b := bodySel; b != "" && b != "*" {
			body.Field = g.mapSelector(b, ogenOp.Request.Type, msg)
		}
		input.Body = body
	}

	return input
}

func (g *Generator) mapSelector(sel string, ogenType *ir.Type, protoType *protogen.Message) *FieldMapping {
	idx := slices.IndexFunc(protoType.Fields, func(f *protogen.Field) bool {
		return f.Desc.TextName() == sel
	})
	f := protoType.Fields[idx]
	return &FieldMapping{
		OgenType: ogenType,
		Proto:    f,
	}
}

func (g *Generator) mapMessage(ogenType *ir.Type, protoType *protogen.Message) MessageMapping {
	m := MessageMapping{
		ProtoType: protoType.GoIdent.GoName,
		OgenType:  qualifiedOgenType(g.pkgName, ogenType.Go()),
	}

	ogenFields := make(map[string]*ir.Field, len(ogenType.Fields))
	for _, f := range ogenType.Fields {
		ogenFields[f.Tag.JSON] = f
	}

	for _, protoField := range protoType.Fields {
		if o := protoField.Desc.ContainingOneof(); o != nil && !o.IsSynthetic() {
			// FIXME(tdakkota): Skip oneOfs: we don't support them yet.
			continue
		}
		jsonName := protoField.Desc.JSONName()

		ogenField, ok := ogenFields[jsonName]
		if !ok {
			panic(fmt.Sprintf("unknown JSON field %q (%s)", jsonName, protoField.Desc.FullName()))
		}

		m.Fields = append(m.Fields, FieldMapping{
			OgenName: ogenField.Name,
			OgenType: ogenField.Type,
			Proto:    protoField,
		})
	}
	return m
}

func (g *Generator) mapEnum(o *ir.Type, p *protogen.Enum) EnumMapping {
	protoType := p.GoIdent.GoName
	return EnumMapping{
		ProtoType:    protoType,
		OgenType:     qualifiedOgenType(g.pkgName, o.Go()),
		EnumNameMap:  protoType + "_name",
		EnumValueMap: protoType + "_value",
	}
}
