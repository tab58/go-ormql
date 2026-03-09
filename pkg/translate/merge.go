package translate

import (
	"fmt"
	"strings"

	"github.com/tab58/go-ormql/pkg/internal/strutil"
	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// translateMergeFieldSplit splits a merge mutation field into a FOREACH write query
// and a MATCH read CALL block. Returns (writeQuery, readCallBlock, alias, error).
// The write uses FOREACH for O(1) memory. The read uses MATCH to fetch merged nodes.
// Both share the same $input parameter.
func (t *Translator) translateMergeFieldSplit(field *ast.Field, scope *paramScope) (string, string, string, error) {
	alias := "__" + field.Name

	node, ok := t.extractNodeName(field.Name, "merge")
	if !ok {
		return "", "", "", fmt.Errorf("unknown type for mutation field %q", field.Name)
	}

	inputArg, inputParam, err := resolveInputParam(field, scope)
	if err != nil {
		return "", "", "", err
	}

	// Build match keys from user-provided input
	matchKeyNames := extractMergeMatchKeyNames(inputArg, scope, node)
	matchParts := buildMergeMatchParts(matchKeyNames)

	onCreateParts := buildOnCreateSet(node, matchKeyNames)
	onMatchParts := buildOnMatchSet(node)

	// --- Build FOREACH write query ---
	var writeSB strings.Builder
	fmt.Fprintf(&writeSB, "FOREACH (item IN %s | MERGE (n:%s {%s})",
		inputParam, node.Labels[0], strings.Join(matchParts, ", "))
	if len(onCreateParts) > 0 {
		fmt.Fprintf(&writeSB, " ON CREATE SET %s", strings.Join(onCreateParts, ", "))
	}
	if len(onMatchParts) > 0 {
		fmt.Fprintf(&writeSB, " ON MATCH SET %s", strings.Join(onMatchParts, ", "))
	}
	writeSB.WriteString(")")

	// --- Build MATCH read CALL block ---
	projSelSet := findResponseSelectionSet(field.SelectionSet)

	var readSB strings.Builder
	readSB.WriteString("CALL { ")
	fmt.Fprintf(&readSB, "UNWIND %s AS item", inputParam)
	fmt.Fprintf(&readSB, " MATCH (n:%s {%s})", node.Labels[0], strings.Join(matchParts, ", "))

	if err := t.appendProjectionReturn(&readSB, projSelSet, node, alias, scope, "merge"); err != nil {
		return "", "", "", err
	}

	return writeSB.String(), readSB.String(), alias, nil
}

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

	inputArg, inputParam, err := resolveInputParam(field, scope)
	if err != nil {
		return "", "", err
	}

	// Build match keys from the user-provided input (not schema defaults).
	// This prevents null property errors when the user only provides a subset
	// of fields in the match object.
	matchKeyNames := extractMergeMatchKeyNames(inputArg, scope, node)
	matchParts := buildMergeMatchParts(matchKeyNames)

	onCreateParts := buildOnCreateSet(node, matchKeyNames)
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

// extractMergeMatchKeyNames determines which fields to use in the MERGE pattern
// by inspecting the user-provided input. Checks AST children first (inline values),
// then resolved variables. Falls back to all non-ID schema fields.
func extractMergeMatchKeyNames(inputArg *ast.Argument, scope *paramScope, node schema.NodeDefinition) []string {
	// Try AST children first (inline values)
	if inputArg.Value != nil && len(inputArg.Value.Children) > 0 {
		firstItem := inputArg.Value.Children[0].Value
		matchVal := findASTChild(firstItem, "match")
		if matchVal != nil && len(matchVal.Children) > 0 {
			keys := make([]string, 0, len(matchVal.Children))
			for _, child := range matchVal.Children {
				keys = append(keys, child.Name)
			}
			return keys
		}
	}

	// Try resolved variables
	resolved := resolveValue(inputArg.Value, scope.variables)
	if items, ok := resolved.([]any); ok && len(items) > 0 {
		if first, ok := items[0].(map[string]any); ok {
			if matchMap, ok := first["match"].(map[string]any); ok {
				keys := make([]string, 0, len(matchMap))
				for k := range matchMap {
					keys = append(keys, k)
				}
				return keys
			}
		}
	}

	// Fallback: all non-ID, non-vector schema fields
	return mergeMatchKeyNames(node)
}

// mergeMatchKeyNames returns the names of all non-ID, non-vector scalar fields.
func mergeMatchKeyNames(node schema.NodeDefinition) []string {
	var keys []string
	for _, f := range node.Fields {
		if f.IsID {
			continue
		}
		if node.VectorField != nil && f.Name == node.VectorField.Name {
			continue
		}
		keys = append(keys, f.Name)
	}
	return keys
}

// buildMergeMatchParts builds the {key: item.match.key, ...} property map entries
// for a MERGE pattern from the given match key names.
func buildMergeMatchParts(matchKeyNames []string) []string {
	parts := make([]string, len(matchKeyNames))
	for i, name := range matchKeyNames {
		parts[i] = fmt.Sprintf("%s: item.match.%s", name, name)
	}
	return parts
}

// buildOnCreateSet builds ON CREATE SET parts for a merge mutation.
// ID fields get randomUUID(), match keys are skipped (already set by MERGE pattern),
// all other fields get item.onCreate.fieldName.
func buildOnCreateSet(node schema.NodeDefinition, matchKeyNames []string) []string {
	matchKeys := make(map[string]struct{}, len(matchKeyNames))
	for _, k := range matchKeyNames {
		matchKeys[k] = struct{}{}
	}

	var parts []string
	for _, f := range node.Fields {
		if f.IsID {
			parts = append(parts, fmt.Sprintf("n.%s = randomUUID()", f.Name))
			continue
		}
		if _, isMatch := matchKeys[f.Name]; isMatch {
			continue // already set by the MERGE pattern
		}
		parts = append(parts, fmt.Sprintf("n.%s = item.onCreate.%s", f.Name, f.Name))
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

// translateConnectFieldSplit splits a connect mutation field into a write query
// and a lightweight read CALL block. Returns (writeQuery, readCallBlock, alias, error).
// The write uses UNWIND + MATCH + MATCH + MERGE without RETURN (fire-and-forget),
// avoiding result-set accumulation that causes OOM in memory-constrained FalkorDB.
// The read returns only the input count via size().
func (t *Translator) translateConnectFieldSplit(field *ast.Field, scope *paramScope) (string, string, string, error) {
	alias := "__" + field.Name

	rel, ok := t.findRelationshipByConnectName(field.Name)
	if !ok {
		return "", "", "", fmt.Errorf("unknown relationship for connect field %q", field.Name)
	}

	fromNode, _ := t.model.NodeByName(rel.FromNode)
	toNode, _ := t.model.NodeByName(rel.ToNode)

	inputArg, inputParam, err := resolveInputParam(field, scope)
	if err != nil {
		return "", "", "", err
	}

	// Extract from/to field names from AST or resolved variables.
	fromFields := extractConnectFieldNames(inputArg, scope, "from")
	toFields := extractConnectFieldNames(inputArg, scope, "to")

	// --- Build write query (no CALL wrapper, no RETURN) ---
	var writeSB strings.Builder
	fmt.Fprintf(&writeSB, "UNWIND %s AS item", inputParam)
	writeConnectMatch(&writeSB, "from", fromNode, fromFields, "item.from")
	writeConnectMatch(&writeSB, "to", toNode, toFields, "item.to")

	mergePattern := buildRelPattern("from", "r", rel.RelType, "to", rel.Direction)
	fmt.Fprintf(&writeSB, " MERGE %s", mergePattern)

	if rel.Properties != nil && len(rel.Properties.Fields) > 0 {
		var setParts []string
		for _, f := range rel.Properties.Fields {
			setParts = append(setParts, fmt.Sprintf("r.%s = item.edge.%s", f.Name, f.Name))
		}
		fmt.Fprintf(&writeSB, " SET %s", strings.Join(setParts, ", "))
	}

	// --- Build read CALL block (lightweight count only) ---
	readBlock := fmt.Sprintf("CALL { RETURN size(%s) AS %s }", inputParam, alias)

	return writeSB.String(), readBlock, alias, nil
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

	// Extract from/to field names from AST or resolved variables.
	fromFields := extractConnectFieldNames(inputArg, scope, "from")
	toFields := extractConnectFieldNames(inputArg, scope, "to")

	// Build FROM and TO MATCH with WHERE from user-provided input fields only
	writeConnectMatch(&sb, "from", fromNode, fromFields, "item.from")
	writeConnectMatch(&sb, "to", toNode, toFields, "item.to")

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
// fieldNames provides the field names to use in WHERE predicates (extracted from
// AST or resolved variables), and itemPath is the input path prefix (e.g., "item.from").
//
// Only fields present in the input are included in WHERE — omitted fields are not
// matched, avoiding null comparisons that would always fail in Cypher.
func writeConnectMatch(sb *strings.Builder, variable string, node schema.NodeDefinition, fieldNames []string, itemPath string) {
	fmt.Fprintf(sb, " MATCH (%s:%s)", variable, node.Labels[0])
	if len(fieldNames) == 0 {
		return
	}
	sb.WriteString(" WHERE")
	var preds []string
	for _, name := range fieldNames {
		preds = append(preds, fmt.Sprintf(" %s.%s = %s.%s", variable, name, itemPath, name))
	}
	sb.WriteString(strings.Join(preds, " AND"))
}

// extractConnectFieldNames extracts the field names for a connect mutation's
// "from" or "to" input object. Tries AST children first (inline values), then
// falls back to resolved variables. This ensures WHERE clauses are generated
// even when input is passed as a variable reference (e.g., $input).
func extractConnectFieldNames(inputArg *ast.Argument, scope *paramScope, key string) []string {
	// Try AST children first (inline values)
	if inputArg.Value != nil && len(inputArg.Value.Children) > 0 {
		firstItem := inputArg.Value.Children[0].Value
		subVal := findASTChild(firstItem, key)
		if subVal != nil && len(subVal.Children) > 0 {
			names := make([]string, 0, len(subVal.Children))
			for _, child := range subVal.Children {
				names = append(names, child.Name)
			}
			return names
		}
	}

	// Try resolved variables
	resolved := resolveValue(inputArg.Value, scope.variables)
	if items, ok := resolved.([]any); ok && len(items) > 0 {
		if first, ok := items[0].(map[string]any); ok {
			if subMap, ok := first[key].(map[string]any); ok {
				names := make([]string, 0, len(subMap))
				for k := range subMap {
					names = append(names, k)
				}
				return names
			}
		}
	}

	return nil
}
