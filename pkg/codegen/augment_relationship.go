package codegen

import (
	"fmt"
	"strings"

	"github.com/tab58/go-ormql/pkg/internal/strutil"
	"github.com/tab58/go-ormql/pkg/schema"
)

// writeNestedInputTypes writes the FieldInput, CreateFieldInput, and ConnectFieldInput
// types for a single relationship.
func writeNestedInputTypes(b *strings.Builder, rel schema.RelationshipDefinition) {
	prefix := rel.FromNode + strutil.Capitalize(rel.FieldName)

	// FieldInput: { create: [...], connect: [...] }
	fmt.Fprintf(b, "input %sFieldInput {\n", prefix)
	fmt.Fprintf(b, "  create: [%sCreateFieldInput!]\n", prefix)
	fmt.Fprintf(b, "  connect: [%sConnectFieldInput!]\n", prefix)
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)

	// CreateFieldInput: { node: <ToNode>CreateInput!, edge: <Props>CreateInput }
	fmt.Fprintf(b, "input %sCreateFieldInput {\n", prefix)
	fmt.Fprintf(b, "  node: %sCreateInput!\n", rel.ToNode)
	if rel.Properties != nil {
		fmt.Fprintf(b, "  edge: %sCreateInput\n", rel.Properties.TypeName)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)

	// ConnectFieldInput: { where: <ToNode>Where, edge: <Props>CreateInput }
	fmt.Fprintf(b, "input %sConnectFieldInput {\n", prefix)
	fmt.Fprintf(b, "  where: %sWhere\n", rel.ToNode)
	if rel.Properties != nil {
		fmt.Fprintf(b, "  edge: %sCreateInput\n", rel.Properties.TypeName)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeRelConnectionTypes writes the connection and edge types for a single relationship.
// {Node}{FieldCap}Connection { edges, pageInfo, totalCount }
// {Node}{FieldCap}Edge { node, cursor, properties? }
func writeRelConnectionTypes(b *strings.Builder, rel schema.RelationshipDefinition) {
	prefix := rel.FromNode + strutil.Capitalize(rel.FieldName)

	// Connection type
	fmt.Fprintf(b, "type %sConnection {\n", prefix)
	fmt.Fprintf(b, "  edges: [%sEdge!]!\n", prefix)
	fmt.Fprintln(b, "  pageInfo: PageInfo!")
	fmt.Fprintln(b, "  totalCount: Int!")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)

	// Edge type
	fmt.Fprintf(b, "type %sEdge {\n", prefix)
	fmt.Fprintf(b, "  node: %s!\n", rel.ToNode)
	fmt.Fprintln(b, "  cursor: String!")
	if rel.Properties != nil {
		fmt.Fprintf(b, "  properties: %s\n", rel.Properties.TypeName)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writePropertiesType writes the output type for a @relationshipProperties type (for reading edge properties).
func writePropertiesType(b *strings.Builder, props schema.PropertiesDefinition) {
	fmt.Fprintf(b, "type %s {\n", props.TypeName)
	for _, f := range props.Fields {
		fmt.Fprintf(b, "  %s: %s\n", f.Name, f.GraphQLType)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writePropertiesCreateInput writes the CreateInput type for a @relationshipProperties type.
func writePropertiesCreateInput(b *strings.Builder, props schema.PropertiesDefinition) {
	writePropertiesInput(b, props, "Create", false)
}

// writeUpdateFieldInput writes the UpdateFieldInput type for a relationship with all 5 ops.
// {Node}{FieldCap}UpdateFieldInput { create, connect, disconnect, update, delete }
func writeUpdateFieldInput(b *strings.Builder, rel schema.RelationshipDefinition) {
	prefix := rel.FromNode + strutil.Capitalize(rel.FieldName)

	fmt.Fprintf(b, "input %sUpdateFieldInput {\n", prefix)
	fmt.Fprintf(b, "  create: [%sCreateFieldInput!]\n", prefix)
	fmt.Fprintf(b, "  connect: [%sConnectFieldInput!]\n", prefix)
	fmt.Fprintf(b, "  disconnect: [%sDisconnectFieldInput!]\n", prefix)
	fmt.Fprintf(b, "  update: %sUpdateConnectionInput\n", prefix)
	fmt.Fprintf(b, "  delete: [%sDeleteFieldInput!]\n", prefix)
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeDisconnectFieldInput writes the DisconnectFieldInput type for a relationship.
func writeDisconnectFieldInput(b *strings.Builder, rel schema.RelationshipDefinition) {
	writeWhereOnlyInput(b, rel, "Disconnect")
}

// writeDeleteFieldInput writes the DeleteFieldInput type for a relationship.
func writeDeleteFieldInput(b *strings.Builder, rel schema.RelationshipDefinition) {
	writeWhereOnlyInput(b, rel, "Delete")
}

// writeWhereOnlyInput writes a relationship input type that contains only a where field.
// Pattern: input {Node}{FieldCap}{suffix}FieldInput { where: {TargetNode}Where }
func writeWhereOnlyInput(b *strings.Builder, rel schema.RelationshipDefinition, suffix string) {
	prefix := rel.FromNode + strutil.Capitalize(rel.FieldName)

	fmt.Fprintf(b, "input %s%sFieldInput {\n", prefix, suffix)
	fmt.Fprintf(b, "  where: %sWhere\n", rel.ToNode)
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeUpdateConnectionInput writes the UpdateConnectionInput type for a relationship.
// {Node}{FieldCap}UpdateConnectionInput { where, node, edge? }
// The edge field is only present when @relationshipProperties exists.
func writeUpdateConnectionInput(b *strings.Builder, rel schema.RelationshipDefinition) {
	prefix := rel.FromNode + strutil.Capitalize(rel.FieldName)

	fmt.Fprintf(b, "input %sUpdateConnectionInput {\n", prefix)
	fmt.Fprintf(b, "  where: %sWhere\n", rel.ToNode)
	fmt.Fprintf(b, "  node: %sUpdateInput\n", rel.ToNode)
	if rel.Properties != nil {
		fmt.Fprintf(b, "  edge: %sUpdateInput\n", rel.Properties.TypeName)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writePropertiesUpdateInput writes the UpdateInput type for a @relationshipProperties type.
// All fields are optional (non-null markers stripped).
func writePropertiesUpdateInput(b *strings.Builder, props schema.PropertiesDefinition) {
	writePropertiesInput(b, props, "Update", true)
}

// writePropertiesInput writes a properties input type with the given suffix.
// When makeOptional is true, non-null markers (!) are stripped from field types.
func writePropertiesInput(b *strings.Builder, props schema.PropertiesDefinition, suffix string, makeOptional bool) {
	fmt.Fprintf(b, "input %s%sInput {\n", props.TypeName, suffix)
	for _, f := range props.Fields {
		gqlType := f.GraphQLType
		if makeOptional {
			gqlType = stripNonNull(gqlType)
		}
		fmt.Fprintf(b, "  %s: %s\n", f.Name, gqlType)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}
