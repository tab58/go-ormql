package codegen

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/tab58/gql-orm/pkg/internal/strutil"
	"github.com/tab58/gql-orm/pkg/schema"
)

// GenerateModels produces Go source code containing model types for the given
// GraphModel. Includes: node structs (with relationship/cypher/connection fields),
// input types (CreateInput, UpdateInput, Where, Sort), nested mutation input types,
// connection types, response types, enum types, and relationship properties types.
// All types have JSON tags matching GraphQL field names.
func GenerateModels(model schema.GraphModel, packageName string) ([]byte, error) {
	if packageName == "" {
		return nil, fmt.Errorf("packageName must not be empty")
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
			modelsWritePropertiesType(&sb, rel.Properties)
			modelsWritePropertiesCreateInput(&sb, rel.Properties)
			modelsWritePropertiesUpdateInput(&sb, rel.Properties)
			propsWritten[rel.Properties.TypeName] = true
		}
	}

	// Node types
	for _, node := range model.Nodes {
		rels := model.RelationshipsForNode(node.Name)
		modelsWriteNodeStruct(&sb, node, rels)
		modelsWriteCreateInput(&sb, node)
		modelsWriteUpdateInput(&sb, node)
		modelsWriteWhereInput(&sb, node)
		modelsWriteSortInput(&sb, node)

		// Root connection types
		modelsWriteConnectionTypes(&sb, node)

		// Mutation response types
		modelsWriteMutationResponseTypes(&sb, node)

		// Relationship-specific types
		for _, rel := range rels {
			if rel.FromNode == node.Name {
				modelsWriteRelConnectionTypes(&sb, node, rel)
				modelsWriteNestedMutationInputTypes(&sb, node, rel)
			}
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
func modelsWritePropertiesType(sb *strings.Builder, props *schema.PropertiesDefinition) {
	sb.WriteString(fmt.Sprintf("type %s struct {\n", props.TypeName))
	for _, f := range props.Fields {
		modelsWriteField(sb, f, false)
	}
	sb.WriteString("}\n\n")
}

// modelsWritePropertiesCreateInput writes a create input for relationship properties.
func modelsWritePropertiesCreateInput(sb *strings.Builder, props *schema.PropertiesDefinition) {
	sb.WriteString(fmt.Sprintf("type %sCreateInput struct {\n", props.TypeName))
	for _, f := range props.Fields {
		modelsWriteField(sb, f, false)
	}
	sb.WriteString("}\n\n")
}

// modelsWritePropertiesUpdateInput writes an update input for relationship properties (all optional).
func modelsWritePropertiesUpdateInput(sb *strings.Builder, props *schema.PropertiesDefinition) {
	sb.WriteString(fmt.Sprintf("type %sUpdateInput struct {\n", props.TypeName))
	for _, f := range props.Fields {
		modelsWriteField(sb, f, true)
	}
	sb.WriteString("}\n\n")
}

// modelsWriteNodeStruct writes a node struct with scalar, relationship, cypher, and connection fields.
func modelsWriteNodeStruct(sb *strings.Builder, node schema.NodeDefinition, rels []schema.RelationshipDefinition) {
	sb.WriteString(fmt.Sprintf("type %s struct {\n", node.Name))

	// Scalar fields
	for _, f := range node.Fields {
		modelsWriteField(sb, f, false)
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
		goType := cf.GoType
		tag := modelsJsonTag(cf.Name, cf.Nullable || strings.HasPrefix(goType, "*"))
		sb.WriteString(fmt.Sprintf("\t%s %s `json:%s`\n", fieldName, goType, tag))
	}

	sb.WriteString("}\n\n")
}

// modelsWriteCreateInput writes a CreateInput struct for a node.
func modelsWriteCreateInput(sb *strings.Builder, node schema.NodeDefinition) {
	sb.WriteString(fmt.Sprintf("type %sCreateInput struct {\n", node.Name))
	for _, f := range node.Fields {
		if f.IsID {
			continue // IDs are auto-generated
		}
		modelsWriteField(sb, f, false)
	}
	sb.WriteString("}\n\n")
}

// modelsWriteUpdateInput writes an UpdateInput struct for a node (all fields optional).
func modelsWriteUpdateInput(sb *strings.Builder, node schema.NodeDefinition) {
	sb.WriteString(fmt.Sprintf("type %sUpdateInput struct {\n", node.Name))
	for _, f := range node.Fields {
		if f.IsID {
			continue // IDs cannot be updated
		}
		modelsWriteField(sb, f, true)
	}
	sb.WriteString("}\n\n")
}

// modelsWriteWhereInput writes a Where struct with operator-suffixed fields + AND/OR/NOT.
func modelsWriteWhereInput(sb *strings.Builder, node schema.NodeDefinition) {
	name := node.Name + "Where"
	sb.WriteString(fmt.Sprintf("type %s struct {\n", name))

	for _, f := range node.Fields {
		fieldUpper := strutil.Capitalize(f.Name)

		// Equality
		sb.WriteString(fmt.Sprintf("\t%s %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+",omitempty"))
		// Not
		sb.WriteString(fmt.Sprintf("\t%sNot %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_NOT,omitempty"))
		// In / NotIn
		sb.WriteString(fmt.Sprintf("\t%sIn %s `json:%q`\n", fieldUpper, modelsSliceType(f.GoType), f.Name+"_IN,omitempty"))
		sb.WriteString(fmt.Sprintf("\t%sNotIn %s `json:%q`\n", fieldUpper, modelsSliceType(f.GoType), f.Name+"_NOT_IN,omitempty"))

		// Comparison operators for non-ID fields
		if !f.IsID {
			sb.WriteString(fmt.Sprintf("\t%sGt %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_GT,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sGte %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_GTE,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sLt %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_LT,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sLte %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_LTE,omitempty"))
		}

		// String operators for string types
		if modelsIsStringType(f.GoType) {
			sb.WriteString(fmt.Sprintf("\t%sContains %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_CONTAINS,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sStartsWith %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_STARTS_WITH,omitempty"))
			sb.WriteString(fmt.Sprintf("\t%sEndsWith %s `json:%q`\n", fieldUpper, modelsPtrType(f.GoType), f.Name+"_ENDS_WITH,omitempty"))
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

// modelsWriteField writes a single struct field with JSON tag.
func modelsWriteField(sb *strings.Builder, f schema.FieldDefinition, forceOptional bool) {
	fieldName := strutil.Capitalize(f.Name)
	goType := f.GoType
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
