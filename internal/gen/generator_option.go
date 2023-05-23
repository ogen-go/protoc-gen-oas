package gen

// GeneratorOption is option for Generator.
type GeneratorOption func(g *Generator)

// WithSpecOpenAPI sets openapi.
func WithSpecOpenAPI(openapi string) GeneratorOption {
	return func(g *Generator) {
		g.spec.SetOpenAPI(openapi)
	}
}

// WithSpecInfoTitle sets title.
func WithSpecInfoTitle(title string) GeneratorOption {
	return func(g *Generator) {
		g.spec.Info.SetTitle(title)
	}
}

// WithSpecInfoDescription sets description.
func WithSpecInfoDescription(description string) GeneratorOption {
	return func(g *Generator) {
		g.spec.Info.SetDescription(description)
	}
}

// WithSpecInfoVersion sets version.
func WithSpecInfoVersion(version string) GeneratorOption {
	return func(g *Generator) {
		g.spec.Info.SetVersion(version)
	}
}

// WithIndent sets indent.
func WithIndent(indent int) GeneratorOption {
	return func(g *Generator) {
		g.indent = indent
	}
}
