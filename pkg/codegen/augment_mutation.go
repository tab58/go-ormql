package codegen

import (
	"fmt"
	"strings"

	"github.com/tab58/go-ormql/pkg/internal/strutil"
	"github.com/tab58/go-ormql/pkg/schema"
)

// writeMutationType writes the root Mutation type with CRUD, merge, and connect mutations for all nodes.
func writeMutationType(b *strings.Builder, nodes []schema.NodeDefinition, rels []schema.RelationshipDefinition) {
	fmt.Fprintln(b, "type Mutation {")
	for _, node := range nodes {
		pc := pluralCapitalized(node.Name)
		fmt.Fprintf(b, "  create%s(input: [%sCreateInput!]!): Create%sMutationResponse!\n", pc, node.Name, pc)
		fmt.Fprintf(b, "  update%s(where: %sWhere, update: %sUpdateInput): Update%sMutationResponse!\n", pc, node.Name, node.Name, pc)
		fmt.Fprintf(b, "  delete%s(where: %sWhere): DeleteInfo!\n", pc, node.Name)
		fmt.Fprintf(b, "  merge%s(input: [%sMergeInput!]!): Merge%sMutationResponse!\n", pc, node.Name, pc)
	}
	for _, rel := range rels {
		connectName := "connect" + rel.FromNode + strutil.Capitalize(rel.FieldName)
		inputName := "Connect" + rel.FromNode + strutil.Capitalize(rel.FieldName) + "Input"
		fmt.Fprintf(b, "  %s(input: [%s!]!): ConnectInfo!\n", connectName, inputName)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeMatchInput writes the {Node}MatchInput type with all scalar fields except id and vector, all optional.
func writeMatchInput(b *strings.Builder, node schema.NodeDefinition) {
	fmt.Fprintf(b, "input %sMatchInput {\n", node.Name)
	hasField := false
	for _, f := range node.Fields {
		if f.IsID {
			continue
		}
		if node.VectorField != nil && f.Name == node.VectorField.Name {
			continue
		}
		hasField = true
		gqlType := stripNonNull(f.GraphQLType)
		fmt.Fprintf(b, "  %s: %s\n", f.Name, gqlType)
	}
	if !hasField {
		fmt.Fprintf(b, "  %s: String\n", emptyFieldPlaceholder)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeMergeInput writes the {Node}MergeInput type with match (required), onCreate (optional), onMatch (optional).
func writeMergeInput(b *strings.Builder, node schema.NodeDefinition) {
	fmt.Fprintf(b, "input %sMergeInput {\n", node.Name)
	fmt.Fprintf(b, "  match: %sMatchInput!\n", node.Name)
	fmt.Fprintf(b, "  onCreate: %sCreateInput\n", node.Name)
	fmt.Fprintf(b, "  onMatch: %sUpdateInput\n", node.Name)
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeMergeMutationResponse writes the Merge{Nodes}MutationResponse type.
func writeMergeMutationResponse(b *strings.Builder, node schema.NodeDefinition) {
	writeMutationResponse(b, node, "Merge")
}

// writeConnectInput writes the Connect{Source}{Field}Input type for a relationship.
// Contains from (required), to (required), and optional edge (when @relationshipProperties).
func writeConnectInput(b *strings.Builder, rel schema.RelationshipDefinition) {
	inputName := "Connect" + rel.FromNode + strutil.Capitalize(rel.FieldName) + "Input"
	fmt.Fprintf(b, "input %s {\n", inputName)
	fmt.Fprintf(b, "  from: %sWhere!\n", rel.FromNode)
	fmt.Fprintf(b, "  to: %sWhere!\n", rel.ToNode)
	if rel.Properties != nil {
		fmt.Fprintf(b, "  edge: %sCreateInput\n", rel.Properties.TypeName)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeConnectInfo writes the shared ConnectInfo type (generated once).
func writeConnectInfo(b *strings.Builder) {
	fmt.Fprintln(b, "type ConnectInfo {")
	fmt.Fprintln(b, "  relationshipsCreated: Int!")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}
