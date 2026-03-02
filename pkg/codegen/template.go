package codegen

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/tab58/go-ormql/pkg/schema"
)

// templateData is the common data structure passed to all code generation templates.
type templateData struct {
	PackageName   string
	Nodes         []schema.NodeDefinition
	Relationships []schema.RelationshipDefinition
}

// executeTemplate parses and executes a Go text/template with the given model
// and package name. Returns the rendered output as a byte slice.
func executeTemplate(name, tmplText string, funcMap template.FuncMap, model schema.GraphModel, packageName string) ([]byte, error) {
	tmpl, err := template.New(name).Funcs(funcMap).Parse(tmplText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s template: %w", name, err)
	}

	data := templateData{
		PackageName:   packageName,
		Nodes:         model.Nodes,
		Relationships: model.Relationships,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute %s template: %w", name, err)
	}

	return buf.Bytes(), nil
}
