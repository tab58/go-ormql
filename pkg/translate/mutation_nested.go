package translate

import (
	"fmt"
	"strings"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// buildNestedCreateOps builds CALL subqueries for nested create and connect operations
// within the input value. Scans input children for relationship field names, then generates
// the appropriate Cypher for each nested operation type.
func (t *Translator) buildNestedCreateOps(inputVal *ast.Value, parentVar string, rels []schema.RelationshipDefinition, parentNode schema.NodeDefinition, scope *paramScope) string {
	if inputVal == nil || len(inputVal.Children) == 0 {
		return ""
	}

	// For list inputs, process the first item to find nested op structure
	// (all items in the list share the same structure)
	var items []*ast.Value
	if inputVal.Kind == ast.ListValue {
		for _, child := range inputVal.Children {
			items = append(items, child.Value)
		}
	} else {
		items = []*ast.Value{inputVal}
	}

	var nestedBlocks []string

	for _, item := range items {
		if item == nil || len(item.Children) == 0 {
			continue
		}
		for _, child := range item.Children {
			// Check if this child name matches a relationship field
			for _, rel := range rels {
				if child.Name != rel.FieldName {
					continue
				}
				// This is a relationship field with nested ops
				if child.Value == nil {
					continue
				}
				for _, opChild := range child.Value.Children {
					switch opChild.Name {
					case "create":
						block := t.buildNestedCreate(parentVar, rel, opChild.Value, scope)
						if block != "" {
							nestedBlocks = append(nestedBlocks, block)
						}
					case "connect":
						block := t.buildNestedConnect(parentVar, rel, opChild.Value, scope)
						if block != "" {
							nestedBlocks = append(nestedBlocks, block)
						}
					}
				}
			}
		}
	}

	return strings.Join(nestedBlocks, " ")
}

// buildNestedCreate generates a CALL subquery for nested CREATE operations.
// Input format: [{node: {name: "..."}, edge: {role: "..."}}, ...]
func (t *Translator) buildNestedCreate(parentVar string, rel schema.RelationshipDefinition, createList *ast.Value, scope *paramScope) string {
	if createList == nil || len(createList.Children) == 0 {
		return ""
	}

	targetNode, ok := t.model.NodeByName(rel.ToNode)
	if !ok {
		targetNode = schema.NodeDefinition{Name: rel.ToNode, Labels: []string{rel.ToNode}}
	}

	var blocks []string
	for _, createItem := range createList.Children {
		item := createItem.Value
		if item == nil {
			continue
		}

		// Extract node and edge data
		nodeData := findASTChild(item, "node")
		edgeData := findASTChild(item, "edge")

		// Build SET for the new node
		var setParts []string
		for _, f := range targetNode.Fields {
			if f.IsID {
				setParts = append(setParts, fmt.Sprintf("child.%s = randomUUID()", f.Name))
			}
		}
		setParts = append(setParts, buildFieldAssignments(nodeData, "child", scope)...)

		// Build relationship pattern
		relPattern := buildRelPattern(parentVar, "r", rel.RelType, "child:"+targetNode.Labels[0], rel.Direction)

		// Edge properties
		relSetParts := buildFieldAssignments(edgeData, "r", scope)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("CALL { WITH %s CREATE %s", parentVar, relPattern))
		if len(setParts) > 0 || len(relSetParts) > 0 {
			allSets := append(setParts, relSetParts...)
			sb.WriteString(fmt.Sprintf(" SET %s", strings.Join(allSets, ", ")))
		}
		sb.WriteString(" }")

		blocks = append(blocks, sb.String())
	}

	return strings.Join(blocks, " ")
}

// buildNestedConnect generates a CALL subquery for nested CONNECT operations.
// Input format: [{where: {name: "..."}, edge: {role: "..."}}, ...]
func (t *Translator) buildNestedConnect(parentVar string, rel schema.RelationshipDefinition, connectList *ast.Value, scope *paramScope) string {
	if connectList == nil || len(connectList.Children) == 0 {
		return ""
	}

	targetNode, ok := t.model.NodeByName(rel.ToNode)
	if !ok {
		targetNode = schema.NodeDefinition{Name: rel.ToNode, Labels: []string{rel.ToNode}}
	}

	var blocks []string
	for _, connectItem := range connectList.Children {
		item := connectItem.Value
		if item == nil {
			continue
		}

		// Extract where and edge data
		whereData := findASTChild(item, "where")
		edgeData := findASTChild(item, "edge")

		// Build WHERE for matching the target node
		whereParts := buildFieldAssignments(whereData, "target", scope)

		// Build relationship pattern
		mergePattern := buildRelPattern(parentVar, "r", rel.RelType, "target", rel.Direction)

		// Edge properties
		relSetParts := buildFieldAssignments(edgeData, "r", scope)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("CALL { WITH %s MATCH (target:%s)", parentVar, targetNode.Labels[0]))
		if len(whereParts) > 0 {
			sb.WriteString(fmt.Sprintf(" WHERE %s", strings.Join(whereParts, " AND ")))
		}
		sb.WriteString(fmt.Sprintf(" MERGE %s", mergePattern))
		if len(relSetParts) > 0 {
			sb.WriteString(fmt.Sprintf(" SET %s", strings.Join(relSetParts, ", ")))
		}
		sb.WriteString(" }")

		blocks = append(blocks, sb.String())
	}

	return strings.Join(blocks, " ")
}

// buildNestedUpdateOps builds CALL subqueries for nested update operations
// (disconnect, update, delete) within an update mutation.
func (t *Translator) buildNestedUpdateOps(parentVar string, rel schema.RelationshipDefinition, opsVal *ast.Value, scope *paramScope) string {
	if opsVal == nil || len(opsVal.Children) == 0 {
		return ""
	}

	targetNode, ok := t.model.NodeByName(rel.ToNode)
	if !ok {
		targetNode = schema.NodeDefinition{Name: rel.ToNode, Labels: []string{rel.ToNode}}
	}

	var blocks []string

	for _, opChild := range opsVal.Children {
		switch opChild.Name {
		case "disconnect":
			block := t.buildNestedDisconnect(parentVar, rel, targetNode, opChild.Value, scope)
			if block != "" {
				blocks = append(blocks, block)
			}
		case "update":
			block := t.buildNestedUpdate(parentVar, rel, targetNode, opChild.Value, scope)
			if block != "" {
				blocks = append(blocks, block)
			}
		case "delete":
			block := t.buildNestedDeleteOp(parentVar, rel, targetNode, opChild.Value, scope)
			if block != "" {
				blocks = append(blocks, block)
			}
		case "create":
			block := t.buildNestedCreate(parentVar, rel, opChild.Value, scope)
			if block != "" {
				blocks = append(blocks, block)
			}
		case "connect":
			block := t.buildNestedConnect(parentVar, rel, opChild.Value, scope)
			if block != "" {
				blocks = append(blocks, block)
			}
		}
	}

	return strings.Join(blocks, " ")
}

// buildNestedDisconnect generates a CALL subquery for nested DISCONNECT operations.
// Matches the relationship and deletes it (keeps nodes).
// Input format: [{where: {name: "..."}}, ...]
func (t *Translator) buildNestedDisconnect(parentVar string, rel schema.RelationshipDefinition, targetNode schema.NodeDefinition, disconnectList *ast.Value, scope *paramScope) string {
	if disconnectList == nil || len(disconnectList.Children) == 0 {
		return ""
	}

	var blocks []string
	for _, disconnectItem := range disconnectList.Children {
		item := disconnectItem.Value
		if item == nil {
			continue
		}

		// Extract where data
		whereData := findASTChild(item, "where")

		// Build relationship pattern
		matchPattern := buildRelPattern(parentVar, "r", rel.RelType, "target:"+targetNode.Labels[0], rel.Direction)

		// Build WHERE for matching the target
		whereParts := buildFieldAssignments(whereData, "target", scope)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("CALL { WITH %s MATCH %s", parentVar, matchPattern))
		if len(whereParts) > 0 {
			sb.WriteString(fmt.Sprintf(" WHERE %s", strings.Join(whereParts, " AND ")))
		}
		sb.WriteString(" DELETE r }")

		blocks = append(blocks, sb.String())
	}

	return strings.Join(blocks, " ")
}

// buildNestedUpdate generates a CALL subquery for nested UPDATE operations.
// Matches the relationship + target node, then SETs both node and edge properties.
// Input format: [{where: {name: "..."}, node: {name: "..."}, edge: {role: "..."}}, ...]
func (t *Translator) buildNestedUpdate(parentVar string, rel schema.RelationshipDefinition, targetNode schema.NodeDefinition, updateList *ast.Value, scope *paramScope) string {
	if updateList == nil || len(updateList.Children) == 0 {
		return ""
	}

	var blocks []string
	for _, updateItem := range updateList.Children {
		item := updateItem.Value
		if item == nil {
			continue
		}

		// Extract where, node, and edge data
		whereData := findASTChild(item, "where")
		nodeData := findASTChild(item, "node")
		edgeData := findASTChild(item, "edge")

		// Build relationship pattern
		matchPattern := buildRelPattern(parentVar, "r", rel.RelType, "target:"+targetNode.Labels[0], rel.Direction)

		// Build WHERE, node SET, and edge SET from AST data
		whereParts := buildFieldAssignments(whereData, "target", scope)
		nodeSetParts := buildFieldAssignments(nodeData, "target", scope)
		edgeSetParts := buildFieldAssignments(edgeData, "r", scope)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("CALL { WITH %s MATCH %s", parentVar, matchPattern))
		if len(whereParts) > 0 {
			sb.WriteString(fmt.Sprintf(" WHERE %s", strings.Join(whereParts, " AND ")))
		}
		if len(nodeSetParts) > 0 {
			sb.WriteString(fmt.Sprintf(" SET %s", strings.Join(nodeSetParts, ", ")))
		}
		if len(edgeSetParts) > 0 {
			sb.WriteString(fmt.Sprintf(" SET %s", strings.Join(edgeSetParts, ", ")))
		}
		sb.WriteString(" }")

		blocks = append(blocks, sb.String())
	}

	return strings.Join(blocks, " ")
}

// buildNestedDeleteOp generates a CALL subquery for nested DELETE operations.
// Matches the relationship + target node, then DETACH DELETEs the target node.
// Input format: [{where: {name: "..."}}, ...]
func (t *Translator) buildNestedDeleteOp(parentVar string, rel schema.RelationshipDefinition, targetNode schema.NodeDefinition, deleteList *ast.Value, scope *paramScope) string {
	if deleteList == nil || len(deleteList.Children) == 0 {
		return ""
	}

	var blocks []string
	for _, deleteItem := range deleteList.Children {
		item := deleteItem.Value
		if item == nil {
			continue
		}

		// Extract where data
		whereData := findASTChild(item, "where")

		// Build relationship pattern (anonymous — no named rel variable)
		matchPattern := buildRelPattern(parentVar, "", rel.RelType, "target:"+targetNode.Labels[0], rel.Direction)

		// Build WHERE
		whereParts := buildFieldAssignments(whereData, "target", scope)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("CALL { WITH %s MATCH %s", parentVar, matchPattern))
		if len(whereParts) > 0 {
			sb.WriteString(fmt.Sprintf(" WHERE %s", strings.Join(whereParts, " AND ")))
		}
		sb.WriteString(" DETACH DELETE target }")

		blocks = append(blocks, sb.String())
	}

	return strings.Join(blocks, " ")
}
