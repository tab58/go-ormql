package codegen

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/tab58/go-ormql/pkg/internal/strutil"
	"github.com/tab58/go-ormql/pkg/schema"
)

// GenerateModels produces Go source code containing model types for the given
// GraphModel. Includes: node structs (with relationship/cypher/connection fields),
// input types (CreateInput, UpdateInput, Where, Sort), nested mutation input types,
// connection types, response types, enum types, and relationship properties types.
// All types have JSON tags matching GraphQL field names.
//
// When the schema declares custom scalars (e.g., DateTime), generated struct fields
// use the scalar type alias (defined in scalars_gen.go) instead of the raw Go type.
// This eliminates the need for models_gen.go to import packages like "time" and
// provides a seam for downstream consumers.
func GenerateModels(model schema.GraphModel, packageName string) ([]byte, error) {
	if packageName == "" {
		return nil, fmt.Errorf("packageName must not be empty")
	}

	// Build custom scalar lookup set for type alias resolution.
	customScalars := make(map[string]bool, len(model.CustomScalars))
	for _, s := range model.CustomScalars {
		customScalars[s] = true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// SortDirection enum
	modelsWriteSortDirectionEnum(&sb)

	// User-defined enums
	for _, e := range model.Enums {
		modelsWriteEnum(&sb, e)
	}

	// PageInfo
	modelsWritePageInfo(&sb)

	// DeleteInfo
	modelsWriteDeleteInfo(&sb)

	// Relationship properties types
	propsWritten := map[string]bool{}
	for _, rel := range model.Relationships {
		if rel.Properties != nil && !propsWritten[rel.Properties.TypeName] {
			modelsWritePropertiesType(&sb, rel.Properties, customScalars)
			modelsWritePropertiesCreateInput(&sb, rel.Properties, customScalars)
			modelsWritePropertiesUpdateInput(&sb, rel.Properties, customScalars)
			propsWritten[rel.Properties.TypeName] = true
		}
	}

	// Node types
	for _, node := range model.Nodes {
		rels := model.RelationshipsForNode(node.Name)
		modelsWriteNodeStruct(&sb, node, rels, customScalars)
		modelsWriteCreateInput(&sb, node, customScalars)
		modelsWriteUpdateInput(&sb, node, customScalars)
		modelsWriteWhereInput(&sb, node, customScalars)
		modelsWriteSortInput(&sb, node)

		// Root connection types
		modelsWriteConnectionTypes(&sb, node)

		// Mutation response types
		modelsWriteMutationResponseTypes(&sb, node)

		// SimilarResult type for vector similarity queries
		if node.VectorField != nil {
			modelsWriteSimilarResultType(&sb, node)
		}

		// Relationship-specific types
		for _, rel := range rels {
			if rel.FromNode == node.Name {
				modelsWriteRelConnectionTypes(&sb, node, rel)
				modelsWriteNestedMutationInputTypes(&sb, node, rel)
			}
		}
	}

	// Merge types per node
	for _, node := range model.Nodes {
		modelsWriteMatchInput(&sb, node, customScalars)
		modelsWriteMergeInput(&sb, node)
		modelsWriteMergeMutationResponse(&sb, node)
	}

	// Connect types per relationship + shared ConnectInfo
	connectInfoWritten := false
	for _, rel := range model.Relationships {
		modelsWriteConnectInput(&sb, rel)
		if !connectInfoWritten {
			modelsWriteConnectInfo(&sb)
			connectInfoWritten = true
		}
	}

	src := sb.String()
	formatted, err := format.Source([]byte(src))
	if err != nil {
		// Return unformatted if gofmt fails (useful for debugging)
		return []byte(src), nil
	}
	return formatted, nil
}

// modelsWriteSortDirectionEnum writes the SortDirection type and its constants.
func modelsWriteSortDirectionEnum(sb *strings.Builder) {
	sb.WriteString("// SortDirection represents sort ordering.\n")
	sb.WriteString("type SortDirection string\n\n")
	sb.WriteString("const (\n")
	sb.WriteString("\tSortDirectionASC  SortDirection = \"ASC\"\n")
	sb.WriteString("\tSortDirectionDESC SortDirection = \"DESC\"\n")
	sb.WriteString(")\n\n")
}

// modelsWriteEnum writes a user-defined enum type and its constants.
func modelsWriteEnum(sb *strings.Builder, e schema.EnumDefinition) {
	sb.WriteString(fmt.Sprintf("type %s string\n\n", e.Name))
	sb.WriteString("const (\n")
	for _, v := range e.Values {
		sb.WriteString(fmt.Sprintf("\t%s%s %s = %q\n", e.Name, v, e.Name, v))
	}
	sb.WriteString(")\n\n")
}

// modelsWritePageInfo writes the PageInfo struct.
func modelsWritePageInfo(sb *strings.Builder) {
	sb.WriteString("type PageInfo struct {\n")
	sb.WriteString("\tHasNextPage     bool    `json:\"hasNextPage\"`\n")
	sb.WriteString("\tHasPreviousPage bool    `json:\"hasPreviousPage\"`\n")
	sb.WriteString("\tStartCursor     *string `json:\"startCursor,omitempty\"`\n")
	sb.WriteString("\tEndCursor       *string `json:\"endCursor,omitempty\"`\n")
	sb.WriteString("}\n\n")
}

// modelsWriteDeleteInfo writes the DeleteInfo struct.
func modelsWriteDeleteInfo(sb *strings.Builder) {
	sb.WriteString("type DeleteInfo struct {\n")
	sb.WriteString("\tNodesDeleted int `json:\"nodesDeleted\"`\n")
	sb.WriteString("}\n\n")
}

// modelsWritePropertiesType writes a relationship properties struct.
func modelsWritePropertiesType(sb *strings.Builder, props *schema.PropertiesDefinition, customScalars map[string]bool) {
	sb.WriteString(fmt.Sprintf("type %s struct {\n", props.TypeName))
	for _, f := range props.Fields {
		modelsWriteField(sb, f, false, customScalars)
	}
	sb.WriteString("}\n\n")
}

// modelsWritePropertiesCreateInput writes a create input for relationship properties.
func modelsWritePropertiesCreateInput(sb *strings.Builder, props *schema.PropertiesDefinition, customScalars map[string]bool) {
	sb.WriteString(fmt.Sprintf("type %sCreateInput struct {\n", props.TypeName))
	for _, f := range props.Fields {
		modelsWriteField(sb, f, false, customScalars)
	}
	sb.WriteString("}\n\n")
}

// modelsWritePropertiesUpdateInput writes an update input for relationship properties (all optional).
func modelsWritePropertiesUpdateInput(sb *strings.Builder, props *schema.PropertiesDefinition, customScalars map[string]bool) {
	sb.WriteString(fmt.Sprintf("type %sUpdateInput struct {\n", props.TypeName))
	for _, f := range props.Fields {
		modelsWriteField(sb, f, true, customScalars)
	}
	sb.WriteString("}\n\n")
}

// modelsWriteNodeStruct writes a node struct with scalar, relationship, cypher, and connection fields.
func modelsWriteNodeStruct(sb *strings.Builder, node schema.NodeDefinition, rels []schema.RelationshipDefinition, customScalars map[string]bool) {
	sb.WriteString(fmt.Sprintf("type %s struct {\n", node.Name))

	// Scalar fields
	for _, f := range node.Fields {
		modelsWriteField(sb, f, false, customScalars)
	}

	// Relationship fields
	for _, rel := range rels {
		if rel.FromNode == node.Name {
			fieldName := strutil.Capitalize(rel.FieldName)
			sb.WriteString(fmt.Sprintf("\t%s []*%s `json:%q`\n", fieldName, rel.ToNode, rel.FieldName+",omitempty"))
		}
	}

	// @cypher fields
	for _, cf := range node.CypherFields {
		fieldName := strutil.Capitalize(cf.Name)
		goType := modelsResolveGoType(cf.GraphQLType, cf.GoType, customScalars)
		tag := modelsJsonTag(cf.Name, cf.Nullable || strings.HasPrefix(goType, "*"))
		sb.WriteString(fmt.Sprintf("\t%s %s `json:%s`\n", fieldName, goType, tag))
	}

	sb.WriteString("}\n\n")
}

// modelsWriteCreateInput writes a CreateInput struct for a node.
func modelsWriteCreateInput(sb *strings.Builder, node schema.NodeDefinition, customScalars map[string]bool) {
	sb.WriteString(fmt.Sprintf("type %sCreateInput struct {\n", node.Name))
	for _, f := range node.Fields {
		if f.IsID {
			continue // IDs are auto-generated
		}
		modelsWriteField(sb, f, false, customScalars)
	}
	sb.WriteString("}\n\n")
}

// modelsWriteUpdateInput writes an UpdateInput struct for a node (all fields optional).
func modelsWriteUpdateInput(sb *strings.Builder, node schema.NodeDefinition, customScalars map[string]bool) {
	sb.WriteString(fmt.Sprintf("type %sUpdateInput struct {\n", node.Name))
	for _, f := range node.Fields {
		if f.IsID {
			continue // IDs cannot be updated
		}
		modelsWriteField(sb, f, true, customScalars)
	}
	sb.WriteString("}\n\n")
}

// modelsWriteWhereInput writes a Where struct with operator-suffixed fields + AND/OR/NOT.
func modelsWriteWhereInput(sb *strings.Builder, node schema.NodeDefinition, customScalars map[string]bool) {
	name := node.Name + "Where"
	sb.WriteString(fmt.Sprintf("type %s struct {\n", name))

	for _, f := range node.Fields {
		fieldUpper := strutil.Capitalize(f.Name)
		goType := modelsResolveGoType(f.GraphQLType, f.GoType, customScalars)

		// Equality
		sb.WriteString(fmt.Sprintf("\t%s %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+",omitempty"))
		// Not
		sb.WriteString(fmt.Sprintf("\t%sNot %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_NOT,omitempty"))
		// In / NotIn
		sb.WriteString(fmt.Sprintf("\t%sIn %s `json:%q`\n", fieldUpper, modelsSliceType(goType), f.Name+"_IN,omitempty"))
		sb.WriteString(fmt.Sprintf("\t%sNotIn %s `json:%q`\n", fieldUpper, modelsSliceType(goType), f.Name+"_NOT_IN,omitempty"))

		// Comparison operators for non-ID fields
		if !f.IsID {
			sb.WriteString(fmt.Sprintf("\t%sGt %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_GT,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sGte %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_GTE,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sLt %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_LT,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sLte %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_LTE,omitempty"))
		}

		// String operators for string types
		if modelsIsStringType(goType) {
			sb.WriteString(fmt.Sprintf("\t%sContains %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_CONTAINS,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sStartsWith %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_STARTS_WITH,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sEndsWith %s `json:%q`\n", fieldUpper, modelsPtrType(goType), f.Name+"_ENDS_WITH,omitempty"))
		}
	}

	// Boolean composition
	sb.WriteString(fmt.Sprintf("\tAND []*%s `json:\"AND,omitempty\"`\n", name))
	sb.WriteString(fmt.Sprintf("\tOR  []*%s `json:\"OR,omitempty\"`\n", name))
	sb.WriteString(fmt.Sprintf("\tNOT *%s  `json:\"NOT,omitempty\"`\n", name))
	sb.WriteString("}\n\n")
}

// modelsWriteSortInput writes a Sort struct for a node.
func modelsWriteSortInput(sb *strings.Builder, node schema.NodeDefinition) {
	sb.WriteString(fmt.Sprintf("type %sSort struct {\n", node.Name))
	for _, f := range node.Fields {
		fieldUpper := strutil.Capitalize(f.Name)
		sb.WriteString(fmt.Sprintf("\t%s *SortDirection `json:%q`\n", fieldUpper, f.Name+",omitempty"))
	}
	sb.WriteString("}\n\n")
}

// modelsWriteConnectionTypes writes root connection types for a node.
func modelsWriteConnectionTypes(sb *strings.Builder, node schema.NodeDefinition) {
	plural := node.Name + "s"

	// Connection
	sb.WriteString(fmt.Sprintf("type %sConnection struct {\n", plural))
	sb.WriteString(fmt.Sprintf("\tEdges      []*%sEdge `json:\"edges\"`\n", node.Name))
	sb.WriteString("\tTotalCount int        `json:\"totalCount\"`\n")
	sb.WriteString("\tPageInfo   PageInfo   `json:\"pageInfo\"`\n")
	sb.WriteString("}\n\n")

	// Edge (root — no properties)
	sb.WriteString(fmt.Sprintf("type %sEdge struct {\n", node.Name))
	sb.WriteString(fmt.Sprintf("\tNode   *%s    `json:\"node\"`\n", node.Name))
	sb.WriteString("\tCursor string `json:\"cursor\"`\n")
	sb.WriteString("}\n\n")
}

// modelsWriteRelConnectionTypes writes nested connection types for a relationship.
func modelsWriteRelConnectionTypes(sb *strings.Builder, node schema.NodeDefinition, rel schema.RelationshipDefinition) {
	connName := node.Name + strutil.Capitalize(rel.FieldName) + "Connection"
	edgeName := node.Name + strutil.Capitalize(rel.FieldName) + "Edge"

	sb.WriteString(fmt.Sprintf("type %s struct {\n", connName))
	sb.WriteString(fmt.Sprintf("\tEdges      []*%s `json:\"edges\"`\n", edgeName))
	sb.WriteString("\tTotalCount int         `json:\"totalCount\"`\n")
	sb.WriteString("\tPageInfo   PageInfo    `json:\"pageInfo\"`\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("type %s struct {\n", edgeName))
	sb.WriteString(fmt.Sprintf("\tNode   *%s    `json:\"node\"`\n", rel.ToNode))
	sb.WriteString("\tCursor string `json:\"cursor\"`\n")
	if rel.Properties != nil {
		sb.WriteString(fmt.Sprintf("\tProperties *%s `json:\"properties,omitempty\"`\n", rel.Properties.TypeName))
	}
	sb.WriteString("}\n\n")
}

// modelsWriteMutationResponseTypes writes Create/Update mutation response types.
func modelsWriteMutationResponseTypes(sb *strings.Builder, node schema.NodeDefinition) {
	plural := node.Name + "s"

	// Create response
	sb.WriteString(fmt.Sprintf("type Create%sMutationResponse struct {\n", plural))
	sb.WriteString(fmt.Sprintf("\t%s []*%s `json:%q`\n", plural, node.Name, strutil.PluralLower(node.Name)))
	sb.WriteString("}\n\n")

	// Update response
	sb.WriteString(fmt.Sprintf("type Update%sMutationResponse struct {\n", plural))
	sb.WriteString(fmt.Sprintf("\t%s []*%s `json:%q`\n", plural, node.Name, strutil.PluralLower(node.Name)))
	sb.WriteString("}\n\n")
}

// modelsWriteSimilarResultType writes the {Node}SimilarResult struct for vector similarity queries.
// Pattern: type {Node}SimilarResult struct { Score float64, Node *{Node} }
func modelsWriteSimilarResultType(sb *strings.Builder, node schema.NodeDefinition) {
	sb.WriteString(fmt.Sprintf("type %sSimilarResult struct {\n", node.Name))
	sb.WriteString("\tScore float64 `json:\"score\"`\n")
	sb.WriteString(fmt.Sprintf("\tNode  *%s     `json:\"node\"`\n", node.Name))
	sb.WriteString("}\n\n")
}

// modelsWriteNestedMutationInputTypes writes FieldInput, CreateFieldInput, ConnectFieldInput, etc.
func modelsWriteNestedMutationInputTypes(sb *strings.Builder, node schema.NodeDefinition, rel schema.RelationshipDefinition) {
	prefix := node.Name + strutil.Capitalize(rel.FieldName)

	// FieldInput (for initial create — contains create + connect)
	sb.WriteString(fmt.Sprintf("type %sFieldInput struct {\n", prefix))
	sb.WriteString(fmt.Sprintf("\tCreate  []*%sCreateFieldInput  `json:\"create,omitempty\"`\n", prefix))
	sb.WriteString(fmt.Sprintf("\tConnect []*%sConnectFieldInput `json:\"connect,omitempty\"`\n", prefix))
	sb.WriteString("}\n\n")

	// UpdateFieldInput
	sb.WriteString(fmt.Sprintf("type %sUpdateFieldInput struct {\n", prefix))
	sb.WriteString(fmt.Sprintf("\tCreate     []*%sCreateFieldInput     `json:\"create,omitempty\"`\n", prefix))
	sb.WriteString(fmt.Sprintf("\tConnect    []*%sConnectFieldInput    `json:\"connect,omitempty\"`\n", prefix))
	sb.WriteString(fmt.Sprintf("\tDisconnect []*%sDisconnectFieldInput `json:\"disconnect,omitempty\"`\n", prefix))
	sb.WriteString(fmt.Sprintf("\tUpdate     []*%sUpdateConnectionInput `json:\"update,omitempty\"`\n", prefix))
	sb.WriteString(fmt.Sprintf("\tDelete     []*%sDeleteFieldInput     `json:\"delete,omitempty\"`\n", prefix))
	sb.WriteString("}\n\n")

	// CreateFieldInput
	sb.WriteString(fmt.Sprintf("type %sCreateFieldInput struct {\n", prefix))
	sb.WriteString(fmt.Sprintf("\tNode %sCreateInput `json:\"node\"`\n", rel.ToNode))
	if rel.Properties != nil {
		sb.WriteString(fmt.Sprintf("\tEdge *%sCreateInput `json:\"edge,omitempty\"`\n", rel.Properties.TypeName))
	}
	sb.WriteString("}\n\n")

	// ConnectFieldInput
	sb.WriteString(fmt.Sprintf("type %sConnectFieldInput struct {\n", prefix))
	sb.WriteString(fmt.Sprintf("\tWhere %sWhere `json:\"where\"`\n", rel.ToNode))
	if rel.Properties != nil {
		sb.WriteString(fmt.Sprintf("\tEdge *%sCreateInput `json:\"edge,omitempty\"`\n", rel.Properties.TypeName))
	}
	sb.WriteString("}\n\n")

	// DisconnectFieldInput
	sb.WriteString(fmt.Sprintf("type %sDisconnectFieldInput struct {\n", prefix))
	sb.WriteString(fmt.Sprintf("\tWhere %sWhere `json:\"where\"`\n", rel.ToNode))
	sb.WriteString("}\n\n")

	// UpdateConnectionInput
	sb.WriteString(fmt.Sprintf("type %sUpdateConnectionInput struct {\n", prefix))
	sb.WriteString(fmt.Sprintf("\tWhere %sWhere       `json:\"where\"`\n", rel.ToNode))
	sb.WriteString(fmt.Sprintf("\tNode  *%sUpdateInput `json:\"node,omitempty\"`\n", rel.ToNode))
	if rel.Properties != nil {
		sb.WriteString(fmt.Sprintf("\tEdge *%sUpdateInput `json:\"edge,omitempty\"`\n", rel.Properties.TypeName))
	}
	sb.WriteString("}\n\n")

	// DeleteFieldInput
	sb.WriteString(fmt.Sprintf("type %sDeleteFieldInput struct {\n", prefix))
	sb.WriteString(fmt.Sprintf("\tWhere %sWhere `json:\"where\"`\n", rel.ToNode))
	sb.WriteString("}\n\n")
}

// modelsWriteMatchInput writes a {Node}MatchInput struct with all-pointer fields (all optional).
// Excludes id and vector fields.
func modelsWriteMatchInput(sb *strings.Builder, node schema.NodeDefinition, customScalars map[string]bool) {
	sb.WriteString(fmt.Sprintf("type %sMatchInput struct {\n", node.Name))
	for _, f := range node.Fields {
		if f.IsID {
			continue
		}
		if node.VectorField != nil && f.Name == node.VectorField.Name {
			continue
		}
		modelsWriteField(sb, f, true, customScalars) // forceOptional = true
	}
	sb.WriteString("}\n\n")
}

// modelsWriteMergeInput writes a {Node}MergeInput struct.
func modelsWriteMergeInput(sb *strings.Builder, node schema.NodeDefinition) {
	sb.WriteString(fmt.Sprintf("type %sMergeInput struct {\n", node.Name))
	sb.WriteString(fmt.Sprintf("\tMatch    *%sMatchInput  `json:\"match\"`\n", node.Name))
	sb.WriteString(fmt.Sprintf("\tOnCreate *%sCreateInput `json:\"onCreate,omitempty\"`\n", node.Name))
	sb.WriteString(fmt.Sprintf("\tOnMatch  *%sUpdateInput `json:\"onMatch,omitempty\"`\n", node.Name))
	sb.WriteString("}\n\n")
}

// modelsWriteMergeMutationResponse writes the Merge{Nodes}MutationResponse struct.
func modelsWriteMergeMutationResponse(sb *strings.Builder, node schema.NodeDefinition) {
	plural := node.Name + "s"
	sb.WriteString(fmt.Sprintf("type Merge%sMutationResponse struct {\n", plural))
	sb.WriteString(fmt.Sprintf("\t%s []*%s `json:%q`\n", plural, node.Name, strutil.PluralLower(node.Name)))
	sb.WriteString("}\n\n")
}

// modelsWriteConnectInput writes a Connect{Source}{Field}Input struct.
func modelsWriteConnectInput(sb *strings.Builder, rel schema.RelationshipDefinition) {
	inputName := "Connect" + rel.FromNode + strutil.Capitalize(rel.FieldName) + "Input"
	sb.WriteString(fmt.Sprintf("type %s struct {\n", inputName))
	sb.WriteString(fmt.Sprintf("\tFrom *%sWhere `json:\"from\"`\n", rel.FromNode))
	sb.WriteString(fmt.Sprintf("\tTo   *%sWhere `json:\"to\"`\n", rel.ToNode))
	if rel.Properties != nil {
		sb.WriteString(fmt.Sprintf("\tEdge *%sCreateInput `json:\"edge,omitempty\"`\n", rel.Properties.TypeName))
	}
	sb.WriteString("}\n\n")
}

// modelsWriteConnectInfo writes the shared ConnectInfo struct (generated once).
func modelsWriteConnectInfo(sb *strings.Builder) {
	sb.WriteString("type ConnectInfo struct {\n")
	sb.WriteString("\tRelationshipsCreated int `json:\"relationshipsCreated\"`\n")
	sb.WriteString("}\n\n")
}

// modelsWriteField writes a single struct field with JSON tag.
// When customScalars contains the field's base GraphQL type, the scalar alias
// name is used instead of the raw Go type.
func modelsWriteField(sb *strings.Builder, f schema.FieldDefinition, forceOptional bool, customScalars map[string]bool) {
	fieldName := strutil.Capitalize(f.Name)
	goType := modelsResolveGoType(f.GraphQLType, f.GoType, customScalars)
	if forceOptional && !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
		goType = "*" + goType
	}
	tag := modelsJsonTag(f.Name, f.Nullable || forceOptional || strings.HasPrefix(goType, "*"))
	sb.WriteString(fmt.Sprintf("\t%s %s `json:%s`\n", fieldName, goType, tag))
}

// modelsJsonTag builds a JSON struct tag string.
func modelsJsonTag(name string, omitempty bool) string {
	if omitempty {
		return fmt.Sprintf("%q", name+",omitempty")
	}
	return fmt.Sprintf("%q", name)
}

// modelsPtrType ensures a type is a pointer type.
func modelsPtrType(goType string) string {
	if strings.HasPrefix(goType, "*") {
		return goType
	}
	return "*" + goType
}

// modelsSliceType converts a Go type to a slice type.
func modelsSliceType(goType string) string {
	base := strings.TrimPrefix(goType, "*")
	return "[]" + base
}

// modelsIsStringType checks if a Go type is a string type.
func modelsIsStringType(goType string) bool {
	base := strings.TrimPrefix(goType, "*")
	return base == "string"
}

// modelsResolveGoType returns the Go type to use for a field in generated models.
// If the field's base GraphQL type is a custom scalar declared in the schema,
// the scalar alias name is used instead of the raw Go type (e.g., "DateTime"
// instead of "time.Time"). This ensures generated models reference the type
// aliases defined in scalars_gen.go rather than the underlying Go types.
func modelsResolveGoType(graphqlType, goType string, customScalars map[string]bool) string {
	base := modelsStripGraphQLModifiers(graphqlType)
	if !customScalars[base] {
		return goType
	}
	if modelsIsGraphQLListType(graphqlType) {
		return "[]" + base
	}
	if !strings.HasSuffix(graphqlType, "!") {
		return "*" + base
	}
	return base
}

// modelsStripGraphQLModifiers extracts the base type name from a GraphQL type string.
// "DateTime!" → "DateTime", "[DateTime!]!" → "DateTime", "[DateTime!]" → "DateTime".
func modelsStripGraphQLModifiers(graphqlType string) string {
	s := strings.TrimSuffix(graphqlType, "!")
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
		s = strings.TrimSuffix(s, "!")
	}
	return s
}

// modelsIsGraphQLListType checks if a GraphQL type string is a list type.
func modelsIsGraphQLListType(graphqlType string) bool {
	s := strings.TrimSuffix(graphqlType, "!")
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}
