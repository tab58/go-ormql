package translate

import (
	"fmt"
	"strings"

	"github.com/tab58/go-ormql/pkg/internal/strutil"
	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// translateMergeField translates a mergeNodes mutation field into a CALL subquery
// with UNWIND + MERGE + ON CREATE SET + ON MATCH SET.
//
// Produces:
//
//	CALL { UNWIND $input AS item
//	  MERGE (n:Label {matchKey1: item.match.matchKey1, ...})
//	  ON CREATE SET n.id = randomUUID(), n.field1 = item.onCreate.field1, ...
//	  ON MATCH SET n.field1 = COALESCE(item.onMatch.field1, n.field1), ...
//	  RETURN collect(projection) AS __alias }
func (t *Translator) translateMergeField(field *ast.Field, scope *paramScope) (string, string, error) {
	alias := "__" + field.Name

	node, ok := t.extractNodeName(field.Name, "merge")
	if !ok {
		return "", "", fmt.Errorf("unknown type for mutation field %q", field.Name)
	}

	_, inputParam, err := resolveInputParam(field, scope)
	if err != nil {
		return "", "", err
	}

	// Build match keys: all non-id, non-vector scalar fields
	matchKeys := mergeMatchKeys(node)

	// Build MERGE pattern with match keys
	var matchParts []string
	for _, f := range matchKeys {
		matchParts = append(matchParts, fmt.Sprintf("%s: item.match.%s", f.Name, f.Name))
	}

	onCreateParts := buildOnCreateSet(node)
	onMatchParts := buildOnMatchSet(node)

	// Find the response field for projection
	projSelSet := findResponseSelectionSet(field.SelectionSet)

	var sb strings.Builder
	sb.WriteString("CALL { ")
	fmt.Fprintf(&sb, "UNWIND %s AS item", inputParam)
	fmt.Fprintf(&sb, " MERGE (n:%s {%s})", node.Labels[0], strings.Join(matchParts, ", "))

	if len(onCreateParts) > 0 {
		fmt.Fprintf(&sb, " ON CREATE SET %s", strings.Join(onCreateParts, ", "))
	}
	if len(onMatchParts) > 0 {
		fmt.Fprintf(&sb, " ON MATCH SET %s", strings.Join(onMatchParts, ", "))
	}

	// Build projection for return and close CALL block
	if err := t.appendProjectionReturn(&sb, projSelSet, node, alias, scope, "merge"); err != nil {
		return "", "", err
	}

	return sb.String(), alias, nil
}

// mergeMatchKeys returns the fields that serve as MERGE match keys:
// all non-id, non-vector scalar fields.
func mergeMatchKeys(node schema.NodeDefinition) []schema.FieldDefinition {
	var keys []schema.FieldDefinition
	for _, f := range node.Fields {
		if f.IsID {
			continue
		}
		if node.VectorField != nil && f.Name == node.VectorField.Name {
			continue
		}
		keys = append(keys, f)
	}
	return keys
}

// buildOnCreateSet builds ON CREATE SET parts for a merge mutation.
// ID fields get randomUUID(), all other fields get item.onCreate.fieldName.
func buildOnCreateSet(node schema.NodeDefinition) []string {
	var parts []string
	for _, f := range node.Fields {
		if f.IsID {
			parts = append(parts, fmt.Sprintf("n.%s = randomUUID()", f.Name))
		} else {
			parts = append(parts, fmt.Sprintf("n.%s = item.onCreate.%s", f.Name, f.Name))
		}
	}
	return parts
}

// buildOnMatchSet builds ON MATCH SET parts for a merge mutation.
// Non-ID fields use COALESCE to only update when the input value is non-null.
func buildOnMatchSet(node schema.NodeDefinition) []string {
	var parts []string
	for _, f := range node.Fields {
		if f.IsID {
			continue
		}
		parts = append(parts, fmt.Sprintf("n.%s = COALESCE(item.onMatch.%s, n.%s)", f.Name, f.Name, f.Name))
	}
	return parts
}

// translateConnectField translates a connectSourceField mutation field into a
// CALL subquery with UNWIND + double MATCH + MERGE for standalone edge creation.
//
// Produces:
//
//	CALL { UNWIND $input AS item
//	  MATCH (from:SourceLabel) WHERE from.field = item.from.field
//	  MATCH (to:TargetLabel) WHERE to.field = item.to.field
//	  MERGE (from)-[r:REL_TYPE]->(to)  -- direction from GraphModel
//	  [SET r.prop = item.edge.prop]     -- optional edge properties
//	  RETURN size($input) AS __count }
func (t *Translator) translateConnectField(field *ast.Field, scope *paramScope) (string, string, error) {
	alias := "__" + field.Name

	// Parse field name to find the relationship: connectMovieActors → Movie + actors
	rel, ok := t.findRelationshipByConnectName(field.Name)
	if !ok {
		return "", "", fmt.Errorf("unknown relationship for connect field %q", field.Name)
	}

	fromNode, _ := t.model.NodeByName(rel.FromNode)
	toNode, _ := t.model.NodeByName(rel.ToNode)

	inputArg, inputParam, err := resolveInputParam(field, scope)
	if err != nil {
		return "", "", err
	}

	var sb strings.Builder
	sb.WriteString("CALL { ")
	fmt.Fprintf(&sb, "UNWIND %s AS item", inputParam)

	// Extract the AST structure for from/to fields from the first input item.
	// The input is a list; the first item's children tell us which fields the user provided.
	var fromInputVal, toInputVal *ast.Value
	if inputArg.Value != nil && len(inputArg.Value.Children) > 0 {
		firstItem := inputArg.Value.Children[0].Value
		fromInputVal = findASTChild(firstItem, "from")
		toInputVal = findASTChild(firstItem, "to")
	}

	// Build FROM and TO MATCH with WHERE from user-provided input fields only
	writeConnectMatch(&sb, "from", fromNode, fromInputVal, "item.from")
	writeConnectMatch(&sb, "to", toNode, toInputVal, "item.to")

	// Build MERGE with correct direction
	mergePattern := buildRelPattern("from", "r", rel.RelType, "to", rel.Direction)
	fmt.Fprintf(&sb, " MERGE %s", mergePattern)

	// Optional edge properties SET
	if rel.Properties != nil && len(rel.Properties.Fields) > 0 {
		var setParts []string
		for _, f := range rel.Properties.Fields {
			setParts = append(setParts, fmt.Sprintf("r.%s = item.edge.%s", f.Name, f.Name))
		}
		fmt.Fprintf(&sb, " SET %s", strings.Join(setParts, ", "))
	}

	fmt.Fprintf(&sb, " RETURN size(%s) AS %s", inputParam, alias)
	sb.WriteString(" }")

	return sb.String(), alias, nil
}

// findRelationshipByConnectName parses a connect field name like "connectMovieActors"
// and finds the matching RelationshipDefinition.
func (t *Translator) findRelationshipByConnectName(fieldName string) (schema.RelationshipDefinition, bool) {
	name := strings.TrimPrefix(fieldName, "connect")
	if name == fieldName {
		return schema.RelationshipDefinition{}, false
	}

	// Try each relationship: check if name starts with FromNode and rest matches capitalized FieldName
	for _, rel := range t.model.Relationships {
		if strings.HasPrefix(name, rel.FromNode) {
			rest := strings.TrimPrefix(name, rel.FromNode)
			capField := strutil.Capitalize(rel.FieldName)
			if rest == capField {
				return rel, true
			}
		}
	}
	return schema.RelationshipDefinition{}, false
}

// writeConnectMatch writes a MATCH + WHERE clause for a connect mutation endpoint.
// variable is the Cypher variable name (e.g., "from"), node provides the label,
// inputVal provides the AST children whose field names drive the WHERE predicates,
// and itemPath is the input path prefix (e.g., "item.from").
//
// Only fields present in the input are included in WHERE — omitted fields are not
// matched, avoiding null comparisons that would always fail in Cypher.
func writeConnectMatch(sb *strings.Builder, variable string, node schema.NodeDefinition, inputVal *ast.Value, itemPath string) {
	fmt.Fprintf(sb, " MATCH (%s:%s)", variable, node.Labels[0])
	if inputVal == nil || len(inputVal.Children) == 0 {
		return
	}
	sb.WriteString(" WHERE")
	var preds []string
	for _, child := range inputVal.Children {
		preds = append(preds, fmt.Sprintf(" %s.%s = %s.%s", variable, child.Name, itemPath, child.Name))
	}
	sb.WriteString(strings.Join(preds, " AND"))
}
