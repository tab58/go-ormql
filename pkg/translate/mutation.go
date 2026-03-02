package translate

import (
	"fmt"
	"strings"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// translateMutation handles a mutation operation.
// Each root mutation field (e.g., "createMovies", "updateMovies", "deleteMovies")
// becomes a CALL subquery. Combined into RETURN {f1: __f1, ...} AS data.
func (t *Translator) translateMutation(op *ast.OperationDefinition, scope *paramScope) (string, error) {
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

		var callBlock, alias string
		var err error

		name := field.Name
		switch {
		case strings.HasPrefix(name, "create"):
			callBlock, alias, err = t.translateCreateField(field, scope)
		case strings.HasPrefix(name, "update"):
			callBlock, alias, err = t.translateUpdateField(field, scope)
		case strings.HasPrefix(name, "delete"):
			callBlock, alias, err = t.translateDeleteField(field, scope)
		default:
			return "", fmt.Errorf("unknown mutation field %q", name)
		}
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

// extractNodeName extracts the node type name from a mutation field name.
// e.g., "createMovies" → "Movie", "updateActors" → "Actor".
func (t *Translator) extractNodeName(fieldName, prefix string) (schema.NodeDefinition, bool) {
	plural := strings.TrimPrefix(fieldName, prefix)
	if len(plural) == 0 {
		return schema.NodeDefinition{}, false
	}
	return t.findNodeByPluralName(strings.ToLower(plural[:1]) + plural[1:])
}

// translateCreateField translates a createNodes mutation field.
// Produces: CALL { UNWIND $input AS item CREATE (n:Label) SET n.id = randomUUID(), n.prop = item.prop, ...
//
//	WITH n [nested ops] RETURN collect(projection) AS __alias }
func (t *Translator) translateCreateField(field *ast.Field, scope *paramScope) (string, string, error) {
	alias := "__" + field.Name

	node, ok := t.extractNodeName(field.Name, "create")
	if !ok {
		return "", "", fmt.Errorf("unknown type for mutation field %q", field.Name)
	}

	// Get the input argument
	inputArg := findArgument(field.Arguments, "input")
	if inputArg == nil {
		return "", "", fmt.Errorf("missing 'input' argument for %q", field.Name)
	}

	// Convert input to Go value and register as parameter
	inputParam := scope.add(resolveValue(inputArg.Value, scope.variables))

	// Get the relationships for this node for nested ops
	rels := t.model.RelationshipsForNode(node.Name)

	// Find the "movies" (or equivalent) response field for projection
	var projSelSet ast.SelectionSet
	for _, sel := range field.SelectionSet {
		f, ok := sel.(*ast.Field)
		if !ok {
			continue
		}
		// The response field is typically named after the plural node type (e.g., "movies")
		if len(f.SelectionSet) > 0 {
			projSelSet = f.SelectionSet
			break
		}
	}

	// Build SET clauses from node fields
	var setParts []string
	for _, f := range node.Fields {
		if f.IsID {
			setParts = append(setParts, fmt.Sprintf("n.%s = randomUUID()", f.Name))
		} else {
			setParts = append(setParts, fmt.Sprintf("n.%s = item.%s", f.Name, f.Name))
		}
	}

	var sb strings.Builder
	sb.WriteString("CALL { ")
	sb.WriteString(fmt.Sprintf("UNWIND %s AS item CREATE (n:%s)", inputParam, node.Labels[0]))
	sb.WriteString(fmt.Sprintf(" SET %s", strings.Join(setParts, ", ")))

	// Build nested mutation ops (create, connect) from input items
	nestedOps := t.buildNestedCreateOps(inputArg.Value, "n", rels, node, scope)
	if nestedOps != "" {
		sb.WriteString(fmt.Sprintf(" WITH n, item %s", nestedOps))
	}

	// Build projection for return
	if projSelSet != nil {
		fc := fieldContext{node: node, variable: "n", depth: 0}
		proj, subqueries, err := t.buildProjection(projSelSet, fc, scope)
		if err != nil {
			return "", "", fmt.Errorf("create projection: %w", err)
		}
		for _, sq := range subqueries {
			sb.WriteString(" ")
			sb.WriteString(sq)
		}
		sb.WriteString(fmt.Sprintf(" RETURN collect(%s) AS %s", proj, alias))
	} else {
		sb.WriteString(fmt.Sprintf(" RETURN collect(n {}) AS %s", alias))
	}
	sb.WriteString(" }")

	return sb.String(), alias, nil
}

// translateUpdateField translates an updateNodes mutation field.
// Produces: CALL { MATCH (n:Label) WHERE ... SET n.prop = $val, ...
//
//	[nested ops] RETURN collect(projection) AS __alias }
func (t *Translator) translateUpdateField(field *ast.Field, scope *paramScope) (string, string, error) {
	alias := "__" + field.Name

	node, ok := t.extractNodeName(field.Name, "update")
	if !ok {
		return "", "", fmt.Errorf("unknown type for mutation field %q", field.Name)
	}

	// Get WHERE argument
	whereArg := findArgument(field.Arguments, "where")
	updateArg := findArgument(field.Arguments, "update")

	// Build WHERE clause
	var whereClause string
	if whereArg != nil {
		whereClause = t.buildWhereClause(whereArg.Value, "n", node, scope)
	}

	// Get relationships for nested ops
	rels := t.model.RelationshipsForNode(node.Name)

	// Build SET clauses from update argument (scalar fields only)
	var setParts []string
	var nestedOps string
	if updateArg != nil {
		for _, child := range updateArg.Value.Children {
			// Check if this is a relationship field (has nested ops)
			isRelField := false
			for _, rel := range rels {
				if child.Name == rel.FieldName {
					isRelField = true
					ops := t.buildNestedUpdateOps("n", rel, child.Value, scope)
					if ops != "" {
						nestedOps += " " + ops
					}
					break
				}
			}
			if !isRelField {
				param := scope.add(resolveValue(child.Value, scope.variables))
				setParts = append(setParts, fmt.Sprintf("n.%s = %s", child.Name, param))
			}
		}
	}

	// Find response field for projection
	var projSelSet ast.SelectionSet
	for _, sel := range field.SelectionSet {
		f, ok := sel.(*ast.Field)
		if !ok {
			continue
		}
		if len(f.SelectionSet) > 0 {
			projSelSet = f.SelectionSet
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("CALL { ")
	sb.WriteString(fmt.Sprintf("MATCH (n:%s)", node.Labels[0]))
	if whereClause != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
	}
	if len(setParts) > 0 {
		sb.WriteString(fmt.Sprintf(" SET %s", strings.Join(setParts, ", ")))
	}

	// Nested ops require WITH clause
	if nestedOps != "" {
		sb.WriteString(fmt.Sprintf(" WITH n%s", nestedOps))
	}

	// Build projection for return
	if projSelSet != nil {
		fc := fieldContext{node: node, variable: "n", depth: 0}
		proj, subqueries, err := t.buildProjection(projSelSet, fc, scope)
		if err != nil {
			return "", "", fmt.Errorf("update projection: %w", err)
		}
		for _, sq := range subqueries {
			sb.WriteString(" ")
			sb.WriteString(sq)
		}
		sb.WriteString(fmt.Sprintf(" RETURN collect(%s) AS %s", proj, alias))
	} else {
		sb.WriteString(fmt.Sprintf(" RETURN collect(n {}) AS %s", alias))
	}
	sb.WriteString(" }")

	return sb.String(), alias, nil
}

// translateDeleteField translates a deleteNodes mutation field.
// Produces: CALL { MATCH (n:Label) WHERE ... WITH count(n) AS cnt DETACH DELETE n RETURN cnt AS __alias }
func (t *Translator) translateDeleteField(field *ast.Field, scope *paramScope) (string, string, error) {
	alias := "__" + field.Name

	node, ok := t.extractNodeName(field.Name, "delete")
	if !ok {
		return "", "", fmt.Errorf("unknown type for mutation field %q", field.Name)
	}

	// Get WHERE argument
	whereArg := findArgument(field.Arguments, "where")

	// Build WHERE clause
	var whereClause string
	if whereArg != nil {
		whereClause = t.buildWhereClause(whereArg.Value, "n", node, scope)
	}

	var sb strings.Builder
	sb.WriteString("CALL { ")
	sb.WriteString(fmt.Sprintf("MATCH (n:%s)", node.Labels[0]))
	if whereClause != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", whereClause))
	}
	// Count before deleting, then DETACH DELETE
	sb.WriteString(" WITH n, count(n) AS cnt DETACH DELETE n")
	sb.WriteString(fmt.Sprintf(" RETURN cnt AS %s", alias))
	sb.WriteString(" }")

	return sb.String(), alias, nil
}
