package codegen

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/tab58/gql-orm/pkg/schema"
)

// GenerateGraphModelRegistry produces Go source code that declares package-level
// variables for the GraphModel and augmented schema SDL.
//
// Output contains:
//   - var GraphModel = schema.GraphModel{...} — full serialization of all nodes,
//     relationships (including Properties), enums, and cypher fields.
//   - var AugmentedSchemaSDL = `...` — the full augmented schema as a raw string literal.
func GenerateGraphModelRegistry(model schema.GraphModel, augSchemaSDL string, packageName string) ([]byte, error) {
	if packageName == "" {
		return nil, fmt.Errorf("packageName must not be empty")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	sb.WriteString("import \"github.com/tab58/gql-orm/pkg/schema\"\n\n")

	// GraphModel variable
	sb.WriteString("var GraphModel = schema.GraphModel{\n")
	registryWriteNodes(&sb, model.Nodes)
	registryWriteRelationships(&sb, model.Relationships)
	registryWriteEnums(&sb, model.Enums)
	sb.WriteString("}\n\n")

	// AugmentedSchemaSDL variable
	registryWriteSDL(&sb, augSchemaSDL)

	src := sb.String()
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return []byte(src), nil
	}
	return formatted, nil
}

// registryWriteNodes serializes the Nodes slice.
func registryWriteNodes(sb *strings.Builder, nodes []schema.NodeDefinition) {
	sb.WriteString("\tNodes: []schema.NodeDefinition{\n")
	for _, n := range nodes {
		sb.WriteString("\t\t{\n")
		sb.WriteString(fmt.Sprintf("\t\t\tName: %q,\n", n.Name))
		sb.WriteString(fmt.Sprintf("\t\t\tLabels: []string{%s},\n", registryQuoteSlice(n.Labels)))
		registryWriteFields(sb, n.Fields, "\t\t\t")
		registryWriteCypherFields(sb, n.CypherFields, "\t\t\t")
		registryWriteVectorField(sb, n.VectorField, "\t\t\t")
		sb.WriteString("\t\t},\n")
	}
	sb.WriteString("\t},\n")
}

// registryWriteFields serializes a Fields slice.
func registryWriteFields(sb *strings.Builder, fields []schema.FieldDefinition, indent string) {
	if len(fields) == 0 {
		return
	}
	sb.WriteString(fmt.Sprintf("%sFields: []schema.FieldDefinition{\n", indent))
	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("%s\t{Name: %q, GraphQLType: %q, GoType: %q, CypherType: %q", indent, f.Name, f.GraphQLType, f.GoType, f.CypherType))
		if f.Nullable {
			sb.WriteString(", Nullable: true")
		}
		if f.IsList {
			sb.WriteString(", IsList: true")
		}
		if f.IsID {
			sb.WriteString(", IsID: true")
		}
		sb.WriteString("},\n")
	}
	sb.WriteString(fmt.Sprintf("%s},\n", indent))
}

// registryWriteCypherFields serializes a CypherFields slice.
func registryWriteCypherFields(sb *strings.Builder, fields []schema.CypherFieldDefinition, indent string) {
	if len(fields) == 0 {
		return
	}
	sb.WriteString(fmt.Sprintf("%sCypherFields: []schema.CypherFieldDefinition{\n", indent))
	for _, cf := range fields {
		sb.WriteString(fmt.Sprintf("%s\t{\n", indent))
		sb.WriteString(fmt.Sprintf("%s\t\tName: %q,\n", indent, cf.Name))
		sb.WriteString(fmt.Sprintf("%s\t\tGraphQLType: %q,\n", indent, cf.GraphQLType))
		sb.WriteString(fmt.Sprintf("%s\t\tGoType: %q,\n", indent, cf.GoType))
		sb.WriteString(fmt.Sprintf("%s\t\tStatement: %q,\n", indent, cf.Statement))
		if cf.IsList {
			sb.WriteString(fmt.Sprintf("%s\t\tIsList: true,\n", indent))
		}
		if cf.Nullable {
			sb.WriteString(fmt.Sprintf("%s\t\tNullable: true,\n", indent))
		}
		if len(cf.Arguments) > 0 {
			sb.WriteString(fmt.Sprintf("%s\t\tArguments: []schema.ArgumentDefinition{\n", indent))
			for _, arg := range cf.Arguments {
				sb.WriteString(fmt.Sprintf("%s\t\t\t{Name: %q, GraphQLType: %q, GoType: %q},\n", indent, arg.Name, arg.GraphQLType, arg.GoType))
			}
			sb.WriteString(fmt.Sprintf("%s\t\t},\n", indent))
		}
		sb.WriteString(fmt.Sprintf("%s\t},\n", indent))
	}
	sb.WriteString(fmt.Sprintf("%s},\n", indent))
}

// registryWriteVectorField serializes a VectorFieldDefinition pointer.
// Emits nothing when vf is nil.
func registryWriteVectorField(sb *strings.Builder, vf *schema.VectorFieldDefinition, indent string) {
	if vf == nil {
		return
	}
	sb.WriteString(fmt.Sprintf("%sVectorField: &schema.VectorFieldDefinition{\n", indent))
	sb.WriteString(fmt.Sprintf("%s\tName: %q,\n", indent, vf.Name))
	sb.WriteString(fmt.Sprintf("%s\tIndexName: %q,\n", indent, vf.IndexName))
	sb.WriteString(fmt.Sprintf("%s\tDimensions: %d,\n", indent, vf.Dimensions))
	sb.WriteString(fmt.Sprintf("%s\tSimilarity: %q,\n", indent, vf.Similarity))
	sb.WriteString(fmt.Sprintf("%s},\n", indent))
}

// registryWriteRelationships serializes the Relationships slice.
func registryWriteRelationships(sb *strings.Builder, rels []schema.RelationshipDefinition) {
	if len(rels) == 0 {
		return
	}
	sb.WriteString("\tRelationships: []schema.RelationshipDefinition{\n")
	for _, r := range rels {
		sb.WriteString("\t\t{\n")
		sb.WriteString(fmt.Sprintf("\t\t\tFieldName: %q,\n", r.FieldName))
		sb.WriteString(fmt.Sprintf("\t\t\tRelType: %q,\n", r.RelType))
		sb.WriteString(fmt.Sprintf("\t\t\tDirection: schema.Direction%s,\n", string(r.Direction)))
		sb.WriteString(fmt.Sprintf("\t\t\tFromNode: %q,\n", r.FromNode))
		sb.WriteString(fmt.Sprintf("\t\t\tToNode: %q,\n", r.ToNode))
		if r.Properties != nil {
			sb.WriteString("\t\t\tProperties: &schema.PropertiesDefinition{\n")
			sb.WriteString(fmt.Sprintf("\t\t\t\tTypeName: %q,\n", r.Properties.TypeName))
			registryWriteFields(sb, r.Properties.Fields, "\t\t\t\t")
			sb.WriteString("\t\t\t},\n")
		}
		sb.WriteString("\t\t},\n")
	}
	sb.WriteString("\t},\n")
}

// registryWriteEnums serializes the Enums slice.
func registryWriteEnums(sb *strings.Builder, enums []schema.EnumDefinition) {
	if len(enums) == 0 {
		return
	}
	sb.WriteString("\tEnums: []schema.EnumDefinition{\n")
	for _, e := range enums {
		sb.WriteString(fmt.Sprintf("\t\t{Name: %q, Values: []string{%s}},\n", e.Name, registryQuoteSlice(e.Values)))
	}
	sb.WriteString("\t},\n")
}

// registryWriteSDL writes the AugmentedSchemaSDL variable.
// Uses raw string literal unless SDL contains backticks, then uses quoted string.
func registryWriteSDL(sb *strings.Builder, sdl string) {
	if strings.Contains(sdl, "`") {
		// Use quoted string literal with proper escaping
		sb.WriteString(fmt.Sprintf("var AugmentedSchemaSDL = %q\n", sdl))
	} else {
		sb.WriteString("var AugmentedSchemaSDL = `")
		sb.WriteString(sdl)
		sb.WriteString("`\n")
	}
}

// registryQuoteSlice formats a string slice as quoted comma-separated values.
func registryQuoteSlice(vals []string) string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(quoted, ", ")
}
