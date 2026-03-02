package translate

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/tab58/gql-orm/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// translateQuery handles a query operation.
// Each root query field (e.g., "movies", "moviesConnection") becomes a CALL subquery.
// Returns: CALL { ... } CALL { ... } RETURN {field1: __f1, field2: __f2} AS data
func (t *Translator) translateQuery(op *ast.OperationDefinition, scope *paramScope) (string, error) {
	if len(op.SelectionSet) == 0 {
		return "RETURN {} AS data", nil
	}

	var callBlocks []string
	var returnParts []string

	for _, sel := range op.SelectionSet {
		field, ok := sel.(*ast.Field)
		if !ok {
			continue
		}

		callBlock, alias, err := t.translateRootField(field, scope)
		if err != nil {
			return "", err
		}
		callBlocks = append(callBlocks, callBlock)
		returnParts = append(returnParts, fmt.Sprintf("%s: %s", field.Alias, alias))
	}

	var sb strings.Builder
	for _, block := range callBlocks {
		sb.WriteString(block)
		sb.WriteString(" ")
	}
	sb.WriteString(fmt.Sprintf("RETURN {%s} AS data", strings.Join(returnParts, ", ")))

	return sb.String(), nil
}

// translateRootField translates a single root query field (list or connection).
// Returns a CALL subquery string and the alias used in the RETURN clause.
func (t *Translator) translateRootField(field *ast.Field, scope *paramScope) (string, string, error) {
	fieldName := field.Name
	alias := "__" + fieldName

	// Check if this is a connection field
	if strings.HasSuffix(fieldName, "Connection") {
		baseName := strings.TrimSuffix(fieldName, "Connection")
		node, ok := t.findNodeByPluralName(baseName)
		if !ok {
			return "", "", fmt.Errorf("unknown type for connection field %q", fieldName)
		}
		fc := fieldContext{node: node, variable: "n", depth: 0}
		return t.translateConnectionField(field, fc, scope)
	}

	// Regular list field — find the node type from the plural field name
	node, ok := t.findNodeByPluralName(fieldName)
	if !ok {
		return "", "", fmt.Errorf("unknown type for field %q", fieldName)
	}

	fc := fieldContext{node: node, variable: "n", depth: 0}

	// Build WHERE clause from "where" argument
	var whereClause string
	if whereArg := findArgument(field.Arguments, "where"); whereArg != nil {
		whereClause = t.buildWhereClause(whereArg.Value, fc.variable, node, scope)
	}

	// Build ORDER BY from "sort" argument
	var orderBy string
	if sortArg := findArgument(field.Arguments, "sort"); sortArg != nil {
		orderBy = t.buildOrderBy(sortArg.Value, fc.variable)
	}

	// Build projection
	proj, subqueries, err := t.buildProjection(field.SelectionSet, fc, scope)
	if err != nil {
		return "", "", err
	}

	var sb strings.Builder
	sb.WriteString("CALL { ")

	// MATCH
	sb.WriteString(fmt.Sprintf("MATCH (%s:%s)", fc.variable, node.Labels[0]))

	// WHERE
	if whereClause != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
	}

	// Subqueries (for nested relationships, @cypher fields)
	for _, sq := range subqueries {
		sb.WriteString(" ")
		sb.WriteString(sq)
	}

	// ORDER BY requires WITH clause
	if orderBy != "" {
		sb.WriteString(fmt.Sprintf(" WITH %s ORDER BY %s", fc.variable, orderBy))
	}

	// RETURN collect(projection) AS alias
	sb.WriteString(fmt.Sprintf(" RETURN collect(%s) AS %s", proj, alias))
	sb.WriteString(" }")

	return sb.String(), alias, nil
}

// findNodeByPluralName looks up a node definition by its plural field name.
// e.g., "movies" → Movie node, "actors" → Actor node.
func (t *Translator) findNodeByPluralName(pluralName string) (schema.NodeDefinition, bool) {
	for _, n := range t.model.Nodes {
		if len(n.Name) == 0 {
			continue
		}
		plural := strings.ToLower(n.Name[:1]) + n.Name[1:] + "s"
		if plural == pluralName {
			return n, true
		}
	}
	return schema.NodeDefinition{}, false
}

// translateConnectionField translates a root or nested connection field.
// Produces CALL subqueries for edges (with pagination), optional totalCount, and pageInfo.
// Returns the combined CALL blocks, an alias for the connection result, and the RETURN map expression.
func (t *Translator) translateConnectionField(field *ast.Field, fc fieldContext, scope *paramScope) (string, string, error) {
	alias := "__" + field.Name
	node := fc.node

	// Parse pagination params
	first := defaultConnectionPageSize
	offset := 0
	if firstArg := findArgument(field.Arguments, "first"); firstArg != nil {
		n, _ := strconv.ParseInt(firstArg.Value.Raw, 10, 64)
		first = int(n)
	}
	if afterArg := findArgument(field.Arguments, "after"); afterArg != nil {
		offset = decodeCursor(afterArg.Value.Raw) + 1
	}

	// Build WHERE clause
	var whereClause string
	if whereArg := findArgument(field.Arguments, "where"); whereArg != nil {
		whereClause = t.buildWhereClause(whereArg.Value, fc.variable, node, scope)
	}

	// Build ORDER BY (default to n.id ASC for stable cursor pagination)
	orderBy := fmt.Sprintf("%s.id ASC", fc.variable)
	if sortArg := findArgument(field.Arguments, "sort"); sortArg != nil {
		if custom := t.buildOrderBy(sortArg.Value, fc.variable); custom != "" {
			orderBy = custom
		}
	}

	// Check what's selected
	cs := detectConnectionSelections(field.SelectionSet)

	// Register pagination params scoped by field name to avoid collision
	// when multiple root connection fields are queried simultaneously.
	offsetParam := scope.addNamed(field.Name+"_offset", offset)
	firstParam := scope.addNamed(field.Name+"_first", first)

	var sb strings.Builder

	// Edge subquery
	sb.WriteString("CALL { ")
	sb.WriteString(fmt.Sprintf("MATCH (%s:%s)", fc.variable, node.Labels[0]))
	if whereClause != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
	}
	sb.WriteString(fmt.Sprintf(" WITH %s ORDER BY %s", fc.variable, orderBy))
	sb.WriteString(fmt.Sprintf(" SKIP %s LIMIT %s", offsetParam, firstParam))

	// Build edge projection
	if cs.wantsEdges && cs.edgesField != nil {
		nodeProj := fmt.Sprintf("%s {}", fc.variable)
		for _, sel := range cs.edgesField.SelectionSet {
			f, ok := sel.(*ast.Field)
			if !ok {
				continue
			}
			if f.Name == "node" && len(f.SelectionSet) > 0 {
				proj, _, err := t.buildProjection(f.SelectionSet, fc, scope)
				if err != nil {
					return "", "", fmt.Errorf("connection edge node projection: %w", err)
				}
				nodeProj = proj
			}
		}
		sb.WriteString(fmt.Sprintf(" RETURN collect({node: %s, cursor: toString(%s)}) AS %s_edges", nodeProj, offsetParam, alias))
	} else {
		sb.WriteString(fmt.Sprintf(" RETURN collect(%s {}) AS %s_edges", fc.variable, alias))
	}
	sb.WriteString(" }")

	// TotalCount subquery (only when selected)
	if cs.wantsTotalCount {
		sb.WriteString(" CALL { ")
		sb.WriteString(fmt.Sprintf("MATCH (%s:%s)", fc.variable, node.Labels[0]))
		if whereClause != "" {
			sb.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
		}
		sb.WriteString(fmt.Sprintf(" RETURN count(%s) AS %s_totalCount", fc.variable, alias))
		sb.WriteString(" }")
	}

	// pageInfo requires totalCount computation
	if cs.wantsPageInfo && !cs.wantsTotalCount {
		sb.WriteString(" CALL { ")
		sb.WriteString(fmt.Sprintf("MATCH (%s:%s)", fc.variable, node.Labels[0]))
		if whereClause != "" {
			sb.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
		}
		sb.WriteString(fmt.Sprintf(" RETURN count(%s) AS %s_totalCount", fc.variable, alias))
		sb.WriteString(" }")
	}

	// Build return map parts
	var returnParts []string
	returnParts = append(returnParts, fmt.Sprintf("edges: %s_edges", alias))
	if cs.wantsTotalCount {
		returnParts = append(returnParts, fmt.Sprintf("totalCount: %s_totalCount", alias))
	}
	if cs.wantsPageInfo {
		pageInfoParts := []string{
			fmt.Sprintf("hasNextPage: %s_totalCount > (%s + %s)", alias, offsetParam, firstParam),
			fmt.Sprintf("hasPreviousPage: %s > 0", offsetParam),
		}
		returnParts = append(returnParts, fmt.Sprintf("pageInfo: {%s}", strings.Join(pageInfoParts, ", ")))
	}

	// Wrap everything in a final CALL block that returns the connection map
	returnMap := fmt.Sprintf("{%s}", strings.Join(returnParts, ", "))

	var result strings.Builder
	result.WriteString(sb.String())
	result.WriteString(fmt.Sprintf(" RETURN %s AS %s", returnMap, alias))

	return result.String(), alias, nil
}

// connectionSelections tracks which parts of a connection field are selected.
type connectionSelections struct {
	wantsTotalCount bool
	wantsPageInfo   bool
	wantsEdges      bool
	edgesField      *ast.Field
}

// detectConnectionSelections inspects a connection field's selection set
// to determine which parts (edges, totalCount, pageInfo) are requested.
func detectConnectionSelections(selSet ast.SelectionSet) connectionSelections {
	var cs connectionSelections
	for _, sel := range selSet {
		f, ok := sel.(*ast.Field)
		if !ok {
			continue
		}
		switch f.Name {
		case "totalCount":
			cs.wantsTotalCount = true
		case "pageInfo":
			cs.wantsPageInfo = true
		case "edges":
			cs.wantsEdges = true
			cs.edgesField = f
		}
	}
	return cs
}

// decodeCursor decodes a base64 cursor to an offset integer.
// Cursor format: base64("cursor:N") where N is the zero-based offset.
func decodeCursor(cursor string) int {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return 0
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return n
}

// findArgument finds an argument by name in an argument list.
func findArgument(args ast.ArgumentList, name string) *ast.Argument {
	for _, arg := range args {
		if arg.Name == name {
			return arg
		}
	}
	return nil
}

// findASTChild finds a named child value within an ast.Value's Children.
// Returns nil if the value is nil or the child is not found.
func findASTChild(val *ast.Value, name string) *ast.Value {
	if val == nil {
		return nil
	}
	for _, child := range val.Children {
		if child.Name == name {
			return child.Value
		}
	}
	return nil
}

// buildFieldAssignments builds "variable.field = $param" assignment strings
// from an ast.Value's children. Used for both WHERE predicates and SET clauses
// in nested mutation operations.
func buildFieldAssignments(data *ast.Value, variable string, scope *paramScope) []string {
	if data == nil {
		return nil
	}
	parts := make([]string, 0, len(data.Children))
	for _, child := range data.Children {
		param := scope.add(astValueToGo(child.Value))
		parts = append(parts, fmt.Sprintf("%s.%s = %s", variable, child.Name, param))
	}
	return parts
}
