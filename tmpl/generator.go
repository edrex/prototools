// Package tmpl implements a protobuf template-based generator.
package tmpl // import "sourcegraph.com/sourcegraph/prototools/tmpl"

import (
	"bytes"
	"fmt"
	"html/template"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// Generator is the type whose methods generate the output, stored in the associated response structure.
type Generator struct {
	// Request from protoc compiler, which should have data unmarshaled into it by
	// the user of this package.
	Request *plugin.CodeGeneratorRequest

	// The template to execute for generation, which should be set by the user of
	// this package.
	Template *template.Template

	// Extension is the extension string to name generated files with. If it is an
	// empty string, the extension of the first template file is used.
	Extension string

	// RootDir is the root directory path prefix to place onto URLs for generated
	// types.
	RootDir string

	// Response to protoc compiler.
	response *plugin.CodeGeneratorResponse
}

// Generate generates a response for g.Request (which you should unmarshal data
// into using protobuf).
//
// If any error is encountered during generation, it is returned and should be
// considered fatal to the generation process (the response will be nil).
func (g *Generator) Generate() (response *plugin.CodeGeneratorResponse, err error) {
	// Reset the response to its initial state.
	g.response.Reset()

	// Determine the extension string.
	ext := g.Extension
	if len(ext) == 0 {
		ext = findExt(g.Template)
	}

	// Generate each proto file:
	errs := new(bytes.Buffer)
	buf := new(bytes.Buffer)
	protoFile := g.Request.GetProtoFile()
	for _, f := range protoFile {
		ctx := &tmplFuncs{
			f:         f,
			ext:       ext,
			rootDir:   g.RootDir,
			protoFile: protoFile,
		}

		// Execute the template and generate a response for the input file.
		buf.Reset()
		err := g.Template.Funcs(ctx.funcMap()).Execute(buf, f)

		// If an error occured during executing the template, we pass it pack to
		// protoc via the error field in the response.
		if err != nil {
			fmt.Fprintf(errs, "%s\n", err)
			continue
		}

		// Determine the file name (relative to the output directory).
		name := stripExt(f.GetName()) + ext
		name = unixPath(name)

		// Generate the response file with the rendered template.
		bufStr := buf.String()
		g.response.File = append(g.response.File, &plugin.CodeGeneratorResponse_File{
			Name:    &name,
			Content: &bufStr,
		})
	}
	if errs.Len() > 0 {
		g.response.File = nil
		errsStr := errs.String()
		g.response.Error = &errsStr
	}
	return g.response, nil
}

// New returns a new generator for the given template.
func New() *Generator {
	return &Generator{
		Request:  &plugin.CodeGeneratorRequest{},
		response: &plugin.CodeGeneratorResponse{},
	}
}
