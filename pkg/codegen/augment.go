package codegen

import (
	"fmt"
	"strings"

	"github.com/tab58/gql-orm/pkg/internal/strutil"
	"github.com/tab58/gql-orm/pkg/schema"
)

// emptyFieldPlaceholder is the dummy field name used when a GraphQL input type
// has no user-defined fields (e.g., a node with only an ID field).
// GraphQL input types must have at least one field.
const emptyFieldPlaceholder = "_empty"

// AugmentSchema takes a parsed GraphModel and produces an augmented GraphQL SDL string
// containing: original node types (minus custom directives), CRUD queries, mutations,
// filter/input types, response types, Relay connection types, and nested mutation input types.
// Shared types (DeleteInfo, PageInfo) are generated once.
func AugmentSchema(model schema.GraphModel) (string, error) {
	if len(model.Nodes) == 0 {
		return "", nil
	}

	var b strings.Builder

	// Generate original node types (without custom directives).
	// Build a map of relationships per node for connection fields on parent types.
	relsByNode := map[string][]schema.RelationshipDefinition{}
	for _, rel := range model.Relationships {
		relsByNode[rel.FromNode] = append(relsByNode[rel.FromNode], rel)
	}
	for _, node := range model.Nodes {
		writeNodeType(&b, node, relsByNode[node.Name])
	}

	// Generate input types, response types, and connection types per node.
	for _, node := range model.Nodes {
		rels := relationshipsForNode(model, node.Name)
		writeWhereInput(&b, node)
		writeSortInput(&b, node)
		writeCreateInput(&b, node, rels)
		writeUpdateInput(&b, node, rels)
		writeCreateResponse(&b, node)
		writeUpdateResponse(&b, node)
		writeConnectionTypes(&b, node)
	}

	// Generate nested mutation input types and relationship connection types per relationship.
	emittedProperties := map[string]bool{}
	for _, rel := range model.Relationships {
		writeNestedInputTypes(&b, rel)
		writeUpdateFieldInput(&b, rel)
		writeDisconnectFieldInput(&b, rel)
		writeDeleteFieldInput(&b, rel)
		writeUpdateConnectionInput(&b, rel)
		writeRelConnectionTypes(&b, rel)
		if rel.Properties != nil && !emittedProperties[rel.Properties.TypeName] {
			writePropertiesCreateInput(&b, *rel.Properties)
			writePropertiesUpdateInput(&b, *rel.Properties)
			writePropertiesType(&b, *rel.Properties)
			emittedProperties[rel.Properties.TypeName] = true
		}
	}

	// Shared types (generated once).
	writeSharedTypes(&b)
	writeSortDirection(&b)

	// Query type.
	writeQueryType(&b, model.Nodes)

	// Mutation type.
	writeMutationType(&b, model.Nodes)

	return b.String(), nil
}

// plural returns a simple plural form (append "s") and lowercase first char.
func plural(name string) string {
	return strutil.PluralLower(name)
}

// pluralCapitalized returns the plural with capitalized first char.
func pluralCapitalized(name string) string {
	return name + "s"
}

// writeNodeType writes the original node type definition with connection fields for relationships.
func writeNodeType(b *strings.Builder, node schema.NodeDefinition, rels []schema.RelationshipDefinition) {
	fmt.Fprintf(b, "type %s {\n", node.Name)
	for _, f := range node.Fields {
		fmt.Fprintf(b, "  %s: %s\n", f.Name, f.GraphQLType)
	}
	for _, cf := range node.CypherFields {
		writeCypherField(b, cf)
	}
	for _, rel := range rels {
		prefix := node.Name + strutil.Capitalize(rel.FieldName)
		fmt.Fprintf(b, "  %sConnection(first: Int, after: String, where: %sWhere, sort: [%sSort!]): %sConnection!\n",
			rel.FieldName, rel.ToNode, rel.ToNode, prefix)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeCypherField writes a @cypher field definition with optional arguments.
func writeCypherField(b *strings.Builder, cf schema.CypherFieldDefinition) {
	if len(cf.Arguments) == 0 {
		fmt.Fprintf(b, "  %s: %s\n", cf.Name, cf.GraphQLType)
		return
	}
	args := make([]string, len(cf.Arguments))
	for i, a := range cf.Arguments {
		args[i] = a.Name + ": " + a.GraphQLType
	}
	fmt.Fprintf(b, "  %s(%s): %s\n", cf.Name, strings.Join(args, ", "), cf.GraphQLType)
}

// writeWhereInput writes the where input type with equality fields plus operator-suffixed fields.
// All fields are optional (nullable) for filtering.
// Operator fields: comparison (_gt, _gte, _lt, _lte) for Int/Float/String/ID (not Boolean),
// string ops (_contains, _startsWith, _endsWith, _regex) for String/ID only,
// list ops (_in, _nin) and _not for all scalars, _isNull: Boolean for all scalars.
// Boolean composition: AND, OR, NOT.
func writeWhereInput(b *strings.Builder, node schema.NodeDefinition) {
	fmt.Fprintf(b, "input %sWhere {\n", node.Name)
	for _, f := range node.Fields {
		baseType := stripNonNull(f.GraphQLType)
		baseName := baseTypeName(f.GraphQLType)

		// Equality field (original)
		fmt.Fprintf(b, "  %s: %s\n", f.Name, baseType)

		// Comparison operators (_gt, _gte, _lt, _lte) — not for Boolean
		if baseName != "Boolean" {
			for _, suffix := range []string{"_gt", "_gte", "_lt", "_lte"} {
				fmt.Fprintf(b, "  %s%s: %s\n", f.Name, suffix, baseType)
			}
		}

		// String operators (_contains, _startsWith, _endsWith, _regex) — String and ID only
		if baseName == "String" || baseName == "ID" {
			for _, suffix := range []string{"_contains", "_startsWith", "_endsWith", "_regex"} {
				fmt.Fprintf(b, "  %s%s: %s\n", f.Name, suffix, baseType)
			}
		}

		// List operators (_in, _nin) — all scalar types
		fmt.Fprintf(b, "  %s_in: [%s!]\n", f.Name, baseName)
		fmt.Fprintf(b, "  %s_nin: [%s!]\n", f.Name, baseName)

		// Negation (_not) — all scalar types
		fmt.Fprintf(b, "  %s_not: %s\n", f.Name, baseType)

		// Null check (_isNull) — all scalar types
		fmt.Fprintf(b, "  %s_isNull: Boolean\n", f.Name)
	}
	// Boolean composition
	fmt.Fprintf(b, "  AND: [%sWhere!]\n", node.Name)
	fmt.Fprintf(b, "  OR: [%sWhere!]\n", node.Name)
	fmt.Fprintf(b, "  NOT: %sWhere\n", node.Name)
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// baseTypeName extracts the base type name from a GraphQL type (strips ! and []).
func baseTypeName(gqlType string) string {
	s := strings.TrimPrefix(gqlType, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSuffix(s, "!")
	return s
}

// writeSortDirection writes the shared SortDirection enum (ASC, DESC).
func writeSortDirection(b *strings.Builder) {
	fmt.Fprintln(b, "enum SortDirection {")
	fmt.Fprintln(b, "  ASC")
	fmt.Fprintln(b, "  DESC")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeSortInput writes the per-node Sort input type with one optional SortDirection
// field per scalar field.
func writeSortInput(b *strings.Builder, node schema.NodeDefinition) {
	fmt.Fprintf(b, "input %sSort {\n", node.Name)
	for _, f := range node.Fields {
		fmt.Fprintf(b, "  %s: SortDirection\n", f.Name)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeCreateInput writes the create input type.
// Non-nullable fields stay required, nullable fields stay optional.
// Relationship fields reference their FieldInput types.
func writeCreateInput(b *strings.Builder, node schema.NodeDefinition, rels []schema.RelationshipDefinition) {
	fmt.Fprintf(b, "input %sCreateInput {\n", node.Name)
	hasNonID := false
	for _, f := range node.Fields {
		if f.IsID {
			continue // ID is auto-generated, not in create input
		}
		hasNonID = true
		fmt.Fprintf(b, "  %s: %s\n", f.Name, f.GraphQLType)
	}
	for _, rel := range rels {
		hasNonID = true
		fieldInputName := node.Name + strutil.Capitalize(rel.FieldName) + "FieldInput"
		fmt.Fprintf(b, "  %s: %s\n", rel.FieldName, fieldInputName)
	}
	// GraphQL input types must have at least one field.
	// If only ID fields exist, include a dummy field.
	if !hasNonID {
		fmt.Fprintf(b, "  %s: Boolean\n", emptyFieldPlaceholder)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeUpdateInput writes the update input type.
// All scalar fields are optional. Relationship fields reference their UpdateFieldInput types.
func writeUpdateInput(b *strings.Builder, node schema.NodeDefinition, rels []schema.RelationshipDefinition) {
	fmt.Fprintf(b, "input %sUpdateInput {\n", node.Name)
	hasField := false
	for _, f := range node.Fields {
		if f.IsID {
			continue
		}
		hasField = true
		gqlType := stripNonNull(f.GraphQLType)
		fmt.Fprintf(b, "  %s: %s\n", f.Name, gqlType)
	}
	for _, rel := range rels {
		hasField = true
		updateFieldInputName := node.Name + strutil.Capitalize(rel.FieldName) + "UpdateFieldInput"
		fmt.Fprintf(b, "  %s: %s\n", rel.FieldName, updateFieldInputName)
	}
	// GraphQL input types must have at least one field.
	if !hasField {
		fmt.Fprintf(b, "  %s: Boolean\n", emptyFieldPlaceholder)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeCreateResponse writes the create mutation response type.
func writeCreateResponse(b *strings.Builder, node schema.NodeDefinition) {
	writeMutationResponse(b, node, "Create")
}

// writeUpdateResponse writes the update mutation response type.
func writeUpdateResponse(b *strings.Builder, node schema.NodeDefinition) {
	writeMutationResponse(b, node, "Update")
}

// writeMutationResponse writes a mutation response type with the given verb prefix.
// Pattern: type {Verb}{PluralNode}MutationResponse { {pluralNode}: [{Node}!]! }
func writeMutationResponse(b *strings.Builder, node schema.NodeDefinition, verb string) {
	p := plural(node.Name)
	fmt.Fprintf(b, "type %s%sMutationResponse {\n", verb, pluralCapitalized(node.Name))
	fmt.Fprintf(b, "  %s: [%s!]!\n", p, node.Name)
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeConnectionTypes writes Relay connection and edge types for a node.
func writeConnectionTypes(b *strings.Builder, node schema.NodeDefinition) {
	pc := pluralCapitalized(node.Name)

	// Connection type
	fmt.Fprintf(b, "type %sConnection {\n", pc)
	fmt.Fprintf(b, "  edges: [%sEdge!]!\n", node.Name)
	fmt.Fprintln(b, "  pageInfo: PageInfo!")
	fmt.Fprintln(b, "  totalCount: Int!")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)

	// Edge type
	fmt.Fprintf(b, "type %sEdge {\n", node.Name)
	fmt.Fprintf(b, "  node: %s!\n", node.Name)
	fmt.Fprintln(b, "  cursor: String!")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeSharedTypes writes DeleteInfo and PageInfo (once).
func writeSharedTypes(b *strings.Builder) {
	fmt.Fprintln(b, "type DeleteInfo {")
	fmt.Fprintln(b, "  nodesDeleted: Int!")
	fmt.Fprintln(b, "  relationshipsDeleted: Int!")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)

	fmt.Fprintln(b, "type PageInfo {")
	fmt.Fprintln(b, "  hasNextPage: Boolean!")
	fmt.Fprintln(b, "  hasPreviousPage: Boolean!")
	fmt.Fprintln(b, "  startCursor: String")
	fmt.Fprintln(b, "  endCursor: String")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeQueryType writes the root Query type with list and connection fields for all nodes.
func writeQueryType(b *strings.Builder, nodes []schema.NodeDefinition) {
	fmt.Fprintln(b, "type Query {")
	for _, node := range nodes {
		p := plural(node.Name)
		pc := pluralCapitalized(node.Name)
		fmt.Fprintf(b, "  %s(where: %sWhere, sort: [%sSort!]): [%s!]!\n", p, node.Name, node.Name, node.Name)
		fmt.Fprintf(b, "  %sConnection(first: Int, after: String, where: %sWhere, sort: [%sSort!]): %sConnection!\n", p, node.Name, node.Name, pc)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// writeMutationType writes the root Mutation type with CRUD mutations for all nodes.
func writeMutationType(b *strings.Builder, nodes []schema.NodeDefinition) {
	fmt.Fprintln(b, "type Mutation {")
	for _, node := range nodes {
		pc := pluralCapitalized(node.Name)
		fmt.Fprintf(b, "  create%s(input: [%sCreateInput!]!): Create%sMutationResponse!\n", pc, node.Name, pc)
		fmt.Fprintf(b, "  update%s(where: %sWhere, update: %sUpdateInput): Update%sMutationResponse!\n", pc, node.Name, node.Name, pc)
		fmt.Fprintf(b, "  delete%s(where: %sWhere): DeleteInfo!\n", pc, node.Name)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
}

// stripNonNull removes the trailing "!" from a GraphQL type string if present.
func stripNonNull(gqlType string) string {
	return strings.TrimSuffix(gqlType, "!")
}


// relationshipsForNode returns all relationships where FromNode matches the given node name.
func relationshipsForNode(model schema.GraphModel, nodeName string) []schema.RelationshipDefinition {
	var result []schema.RelationshipDefinition
	for _, r := range model.Relationships {
		if r.FromNode == nodeName {
			result = append(result, r)
		}
	}
	return result
}

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
