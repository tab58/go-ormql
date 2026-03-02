package translate

import (
	"fmt"
	"strings"

	"github.com/tab58/gql-orm/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// subqueryCounter tracks unique alias numbering across subqueries in a translation.
var subqueryVarNames = []string{"a", "b", "c", "d", "e", "f", "g", "h"}

// childVariable returns a unique Cypher variable for a nested subquery at the given depth.
func childVariable(depth int) string {
	if depth < len(subqueryVarNames) {
		return subqueryVarNames[depth]
	}
	return fmt.Sprintf("v%d", depth)
}

// buildSubquery builds a CALL subquery for a nested relationship field.
// Returns: CALL { WITH parent MATCH (parent)<-[:TYPE]-(child:Label) WHERE ... ORDER BY ...
//
//	RETURN collect(childProjection) AS __subN }
func (t *Translator) buildSubquery(field *ast.Field, rel schema.RelationshipDefinition, fc fieldContext, scope *paramScope) (string, string, error) {
	childVar := childVariable(fc.depth)
	alias := fmt.Sprintf("__sub%d", fc.depth)

	// Find the target node
	targetNode, ok := t.model.NodeByName(rel.ToNode)
	if !ok {
		targetNode = schema.NodeDefinition{Name: rel.ToNode, Labels: []string{rel.ToNode}}
	}

	childFC := fieldContext{
		node:     targetNode,
		variable: childVar,
		depth:    fc.depth + 1,
	}

	// Build the relationship pattern based on direction
	matchPattern := buildRelPattern(fc.variable, "", rel.RelType, childVar+":"+targetNode.Labels[0], rel.Direction)

	// Build WHERE clause from "where" argument
	var whereClause string
	if whereArg := findArgument(field.Arguments, "where"); whereArg != nil {
		whereClause = t.buildWhereClause(whereArg.Value, childVar, targetNode, scope)
	}

	// Build ORDER BY from "sort" argument
	var orderBy string
	if sortArg := findArgument(field.Arguments, "sort"); sortArg != nil {
		orderBy = t.buildOrderBy(sortArg.Value, childVar, scope)
	}

	// Build child projection
	proj, childSubqueries, err := t.buildProjection(field.SelectionSet, childFC, scope)
	if err != nil {
		return "", "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CALL { WITH %s MATCH %s", fc.variable, matchPattern))

	if whereClause != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
	}

	// Nested subqueries (for deeply nested relationships)
	for _, sq := range childSubqueries {
		sb.WriteString(" ")
		sb.WriteString(sq)
	}

	if orderBy != "" {
		sb.WriteString(fmt.Sprintf(" WITH %s ORDER BY %s", childVar, orderBy))
	}

	sb.WriteString(fmt.Sprintf(" RETURN collect(%s) AS %s }", proj, alias))

	return sb.String(), alias, nil
}

// buildCypherSubquery builds a CALL subquery for a @cypher field.
// Returns: CALL { WITH parent AS this <statement> RETURN result AS __cypherN LIMIT 1 }
// LIMIT 1 is added for scalar return types. List return types omit LIMIT.
func (t *Translator) buildCypherSubquery(field *ast.Field, cf schema.CypherFieldDefinition, fc fieldContext, scope *paramScope) (string, string, error) {
	alias := fmt.Sprintf("__cypher%d", fc.depth)

	// Pass field arguments as parameters using the original argument name,
	// matching the $argName references in the user's Cypher statement.
	for _, arg := range field.Arguments {
		for _, cfArg := range cf.Arguments {
			if arg.Name == cfArg.Name {
				scope.addNamed(arg.Name, resolveValue(arg.Value, scope.variables))
				break
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CALL { WITH %s AS this %s", fc.variable, cf.Statement))

	// For scalar fields, add LIMIT 1
	if !cf.IsList {
		// The user's statement should end with a RETURN — we add LIMIT 1 after
		sb.WriteString(" LIMIT 1")
	}

	sb.WriteString(" }")

	return sb.String(), alias, nil
}

// buildConnectionSubquery builds a CALL subquery for a nested connection field.
// Returns: CALL { WITH parent <inner CALL blocks> RETURN {edges: ..., totalCount: ..., pageInfo: ...} AS __connN }
// Inner CALL blocks handle edge pagination and optional totalCount, wrapped in an
// outer CALL that assembles the connection map — matching translateConnectionField's pattern.
func (t *Translator) buildConnectionSubquery(field *ast.Field, rel schema.RelationshipDefinition, fc fieldContext, scope *paramScope) (string, string, error) {
	alias := fmt.Sprintf("__conn%d", fc.depth)
	childVar := childVariable(fc.depth)

	// Find the target node
	targetNode, ok := t.model.NodeByName(rel.ToNode)
	if !ok {
		targetNode = schema.NodeDefinition{Name: rel.ToNode, Labels: []string{rel.ToNode}}
	}

	childFC := fieldContext{
		node:     targetNode,
		variable: childVar,
		depth:    fc.depth + 1,
	}

	// Build the relationship pattern based on direction
	relVar := fmt.Sprintf("r%d", fc.depth)
	matchPattern := buildRelPattern(fc.variable, relVar, rel.RelType, childVar+":"+targetNode.Labels[0], rel.Direction)

	// Parse pagination params
	first, offset := parsePagination(field.Arguments, scope)

	// Build WHERE clause
	var whereClause string
	if whereArg := findArgument(field.Arguments, "where"); whereArg != nil {
		whereClause = t.buildWhereClause(whereArg.Value, childVar, targetNode, scope)
	}

	// Build ORDER BY (default to child.id ASC)
	orderBy := fmt.Sprintf("%s.id ASC", childVar)
	if sortArg := findArgument(field.Arguments, "sort"); sortArg != nil {
		if custom := t.buildOrderBy(sortArg.Value, childVar, scope); custom != "" {
			orderBy = custom
		}
	}

	// Check what's selected
	cs := detectConnectionSelections(field.SelectionSet)

	// Register pagination params
	offsetParam := scope.addNamed(fmt.Sprintf("conn%d_offset", fc.depth), offset)
	firstParam := scope.addNamed(fmt.Sprintf("conn%d_first", fc.depth), first)

	// Build inner CALL blocks (edge subquery + optional totalCount subquery)
	var inner strings.Builder

	// Edge subquery
	inner.WriteString(fmt.Sprintf("CALL { WITH %s ", fc.variable))
	inner.WriteString(fmt.Sprintf("MATCH %s", matchPattern))
	if whereClause != "" {
		inner.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
	}
	inner.WriteString(fmt.Sprintf(" WITH %s, %s ORDER BY %s", childVar, relVar, orderBy))
	inner.WriteString(fmt.Sprintf(" SKIP %s LIMIT %s", offsetParam, firstParam))

	// Build edge projection
	if cs.wantsEdges && cs.edgesField != nil {
		nodeProj := fmt.Sprintf("%s {}", childVar)
		var hasProperties bool
		var propFields []string
		for _, sel := range cs.edgesField.SelectionSet {
			f, ok := sel.(*ast.Field)
			if !ok {
				continue
			}
			if f.Name == "node" && len(f.SelectionSet) > 0 {
				proj, _, err := t.buildProjection(f.SelectionSet, childFC, scope)
				if err != nil {
					return "", "", fmt.Errorf("connection edge node projection: %w", err)
				}
				nodeProj = proj
			}
			if f.Name == "properties" && len(f.SelectionSet) > 0 {
				hasProperties = true
				for _, pSel := range f.SelectionSet {
					pf, ok := pSel.(*ast.Field)
					if !ok {
						continue
					}
					propFields = append(propFields, fmt.Sprintf("%s: %s.%s", pf.Name, relVar, pf.Name))
				}
			}
		}

		edgeParts := []string{fmt.Sprintf("node: %s", nodeProj)}
		edgeParts = append(edgeParts, fmt.Sprintf("cursor: toString(%s)", offsetParam))
		if hasProperties && len(propFields) > 0 {
			edgeParts = append(edgeParts, fmt.Sprintf("properties: {%s}", strings.Join(propFields, ", ")))
		}
		inner.WriteString(fmt.Sprintf(" RETURN collect({%s}) AS %s_edges", strings.Join(edgeParts, ", "), alias))
	} else {
		inner.WriteString(fmt.Sprintf(" RETURN collect(%s {}) AS %s_edges", childVar, alias))
	}
	inner.WriteString(" }")

	// TotalCount subquery (needed for totalCount or pageInfo)
	if cs.wantsTotalCount || cs.wantsPageInfo {
		inner.WriteString(fmt.Sprintf(" CALL { WITH %s ", fc.variable))
		inner.WriteString(fmt.Sprintf("MATCH %s", matchPattern))
		if whereClause != "" {
			inner.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
		}
		inner.WriteString(fmt.Sprintf(" RETURN count(%s) AS %s_totalCount", childVar, alias))
		inner.WriteString(" }")
	}

	// Wrap everything in an outer CALL block that returns the connection map
	returnMap := buildConnectionReturnMap(alias, offsetParam, firstParam, cs)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CALL { WITH %s ", fc.variable))
	sb.WriteString(inner.String())
	sb.WriteString(fmt.Sprintf(" RETURN %s AS %s }", returnMap, alias))

	return sb.String(), alias, nil
}
