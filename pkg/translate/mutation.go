package translate

import (
	"fmt"
	"strings"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// translateMutationSplit separates mutation fields into write queries (FOREACH writes
// from merge fields) and a read query (CALL blocks + RETURN map).
// Non-merge fields remain as CALL blocks in the read query only.
// Merge fields contribute both a write query AND a read CALL block.
func (t *Translator) translateMutationSplit(op *ast.OperationDefinition, scope *paramScope) ([]string, string, error) {
	if len(op.SelectionSet) == 0 {
		return nil, "RETURN {} AS data", nil
	}

	var writeQueries []string
	var callBlocks []string
	var returnParts []string

	for _, sel := range op.SelectionSet {
		field, ok := sel.(*ast.Field)
		if !ok {
			continue
		}

		name := field.Name
		switch {
		case strings.HasPrefix(name, "merge"):
			// Merge fields produce both a FOREACH write and a MATCH read CALL block
			writeQuery, readBlock, alias, err := t.translateMergeFieldSplit(field, scope)
			if err != nil {
				return nil, "", err
			}
			writeQueries = append(writeQueries, writeQuery)
			callBlocks = append(callBlocks, readBlock)
			returnParts = append(returnParts, fmt.Sprintf("%s: %s", field.Alias, alias))

		case strings.HasPrefix(name, "connect"):
			// Connect fields produce a write (UNWIND+MATCH+MERGE) and a lightweight read (count)
			writeQuery, readBlock, alias, err := t.translateConnectFieldSplit(field, scope)
			if err != nil {
				return nil, "", err
			}
			writeQueries = append(writeQueries, writeQuery)
			callBlocks = append(callBlocks, readBlock)
			returnParts = append(returnParts, fmt.Sprintf("%s: %s", field.Alias, alias))

		default:
			// Non-split fields: create, update, delete — single CALL block
			var callBlock, alias string
			var err error
			switch {
			case strings.HasPrefix(name, "create"):
				callBlock, alias, err = t.translateCreateField(field, scope)
			case strings.HasPrefix(name, "update"):
				callBlock, alias, err = t.translateUpdateField(field, scope)
			case strings.HasPrefix(name, "delete"):
				callBlock, alias, err = t.translateDeleteField(field, scope)
			default:
				return nil, "", fmt.Errorf("unknown mutation field %q", name)
			}
			if err != nil {
				return nil, "", err
			}
			callBlocks = append(callBlocks, callBlock)
			returnParts = append(returnParts, fmt.Sprintf("%s: %s", field.Alias, alias))
		}
	}

	var sb strings.Builder
	for _, block := range callBlocks {
		sb.WriteString(block)
		sb.WriteString(" ")
	}
	fmt.Fprintf(&sb, "RETURN {%s} AS data", strings.Join(returnParts, ", "))

	return writeQueries, sb.String(), nil
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
	projSelSet := findResponseSelectionSet(field.SelectionSet)

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
	fmt.Fprintf(&sb, "UNWIND %s AS item CREATE (n:%s)", inputParam, node.Labels[0])
	fmt.Fprintf(&sb, " SET %s", strings.Join(setParts, ", "))

	// Build nested mutation ops (create, connect) from input items
	nestedOps := t.buildNestedCreateOps(inputArg.Value, "n", rels, node, scope)
	if nestedOps != "" {
		fmt.Fprintf(&sb, " WITH n, item %s", nestedOps)
	}

	// Build projection for return and close CALL block
	if err := t.appendProjectionReturn(&sb, projSelSet, node, alias, scope, "create"); err != nil {
		return "", "", err
	}

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
	projSelSet := findResponseSelectionSet(field.SelectionSet)

	var sb strings.Builder
	sb.WriteString("CALL { ")
	fmt.Fprintf(&sb, "MATCH (n:%s)", node.Labels[0])
	if whereClause != "" {
		fmt.Fprintf(&sb, " WHERE %s", whereClause)
	}
	if len(setParts) > 0 {
		fmt.Fprintf(&sb, " SET %s", strings.Join(setParts, ", "))
	}

	// Nested ops require WITH clause
	if nestedOps != "" {
		fmt.Fprintf(&sb, " WITH n%s", nestedOps)
	}

	// Build projection for return and close CALL block
	if err := t.appendProjectionReturn(&sb, projSelSet, node, alias, scope, "update"); err != nil {
		return "", "", err
	}

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
	fmt.Fprintf(&sb, "MATCH (n:%s)", node.Labels[0])
	if whereClause != "" {
		fmt.Fprintf(&sb, " WHERE %s", whereClause)
	}
	// Count before deleting, then DETACH DELETE
	sb.WriteString(" WITH n, count(n) AS cnt DETACH DELETE n")
	fmt.Fprintf(&sb, " RETURN cnt AS %s", alias)
	sb.WriteString(" }")

	return sb.String(), alias, nil
}
