package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
	"github.com/tab58/gql-orm/pkg/internal/strutil"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// directExecute handles query execution when no gqlgen ExecutableSchema is
// configured. It parses the GraphQL query, maps operations to Cypher, and
// executes them against the driver directly.
func (c *Client) directExecute(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	src := &ast.Source{Name: "query", Input: query}
	doc, err := parser.ParseQuery(src)
	if err != nil {
		return nil, fmt.Errorf("query parse error: %v", err)
	}

	if len(doc.Operations) == 0 {
		return map[string]any{}, nil
	}

	op := doc.Operations[0]
	result := map[string]any{}

	for _, sel := range op.SelectionSet {
		field, ok := sel.(*ast.Field)
		if !ok {
			continue
		}

		var fieldResult any
		var execErr error

		if op.Operation == ast.Mutation {
			fieldResult, execErr = c.executeMutationField(ctx, field, variables)
		} else {
			fieldResult, execErr = c.executeQueryField(ctx, field, variables)
		}
		if execErr != nil {
			return nil, execErr
		}
		result[field.Name] = fieldResult
	}

	return result, nil
}

// executeQueryField handles a query field like `movies(where: {...})`.
func (c *Client) executeQueryField(ctx context.Context, field *ast.Field, variables map[string]any) (any, error) {
	label := singularize(strutil.Capitalize(field.Name))
	where := resolveArgMap(field, "where", variables)

	stmt := cypher.NodeMatch(label, cypher.EqualityWhere(where), nil)
	res, err := c.drv.Execute(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", field.Name, err)
	}

	return recordsToSlice(res.Records), nil
}

// executeMutationField handles a mutation field like `createMovies(input: [...])`.
func (c *Client) executeMutationField(ctx context.Context, field *ast.Field, variables map[string]any) (any, error) {
	name := field.Name

	if strings.HasPrefix(name, "create") {
		return c.executeCreateMutation(ctx, name, field, variables)
	}
	if strings.HasPrefix(name, "update") {
		return c.executeUpdateMutation(ctx, name, field, variables)
	}
	if strings.HasPrefix(name, "delete") {
		return c.executeDeleteMutation(ctx, name, field, variables)
	}

	return map[string]any{}, nil
}

// executeCreateMutation handles `createMovies(input: [...])`.
func (c *Client) executeCreateMutation(ctx context.Context, name string, field *ast.Field, variables map[string]any) (any, error) {
	label := singularize(strings.TrimPrefix(name, "create"))
	input := resolveArgList(field, "input", variables)

	var created []any
	for _, item := range input {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Extract relationship fields before creating the node.
		nodeProps, relFields := splitNodeAndRelFields(itemMap)

		stmt := cypher.NodeCreate(label, nodeProps)
		res, err := c.drv.ExecuteWrite(ctx, stmt)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", label, err)
		}

		// Process nested relationship fields.
		for relName, relVal := range relFields {
			relMap, ok := relVal.(map[string]any)
			if !ok {
				continue
			}
			if err := c.processNestedRelField(ctx, label, relName, relMap, variables); err != nil {
				return nil, err
			}
		}

		created = append(created, recordsToSlice(res.Records)...)
	}

	// Wrap in response type shape.
	return map[string]any{pluralFieldName(label): created}, nil
}

// executeUpdateMutation handles `updateMovies(where: {...}, update: {...})`.
func (c *Client) executeUpdateMutation(ctx context.Context, name string, field *ast.Field, variables map[string]any) (any, error) {
	label := singularize(strings.TrimPrefix(name, "update"))
	where := resolveArgMap(field, "where", variables)
	update := resolveArgMap(field, "update", variables)

	stmt := cypher.NodeUpdate(label, cypher.EqualityWhere(where), update)
	res, err := c.drv.ExecuteWrite(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("update %s: %w", label, err)
	}

	return map[string]any{pluralFieldName(label): recordsToSlice(res.Records)}, nil
}

// executeDeleteMutation handles `deleteMovies(where: {...})`.
func (c *Client) executeDeleteMutation(ctx context.Context, name string, field *ast.Field, variables map[string]any) (any, error) {
	label := singularize(strings.TrimPrefix(name, "delete"))
	where := resolveArgMap(field, "where", variables)

	stmt := cypher.NodeDelete(label, cypher.EqualityWhere(where))
	_, err := c.drv.ExecuteWrite(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("delete %s: %w", label, err)
	}

	return map[string]any{"nodesDeleted": 1, "relationshipsDeleted": 0}, nil
}

// processNestedRelField processes create/connect nested inputs for a relationship field.
func (c *Client) processNestedRelField(ctx context.Context, fromLabel, relName string, relMap map[string]any, _ map[string]any) error {
	toLabel := singularize(strutil.Capitalize(relName))
	relType := strings.ToUpper(relName)

	if creates, ok := relMap["create"]; ok {
		if err := c.processNestedCreate(ctx, fromLabel, toLabel, relType, creates); err != nil {
			return err
		}
	}

	if connects, ok := relMap["connect"]; ok {
		if err := c.processNestedConnect(ctx, fromLabel, toLabel, relType, connects); err != nil {
			return err
		}
	}

	return nil
}

// processNestedCreate handles "create" nested inputs: creates a related node and a relationship to it.
func (c *Client) processNestedCreate(ctx context.Context, fromLabel, toLabel, relType string, creates any) error {
	createList, _ := creates.([]any)
	for _, createItem := range createList {
		ci, ok := createItem.(map[string]any)
		if !ok {
			continue
		}
		nodeInput, _ := ci["node"].(map[string]any)

		nodeStmt := cypher.NodeCreate(toLabel, nodeInput)
		if _, err := c.drv.ExecuteWrite(ctx, nodeStmt); err != nil {
			return fmt.Errorf("nested create %s: %w", toLabel, err)
		}

		edgeProps := extractEdgeProps(ci)
		relStmt := cypher.RelCreate(fromLabel, cypher.EqualityWhere(map[string]any{}), relType, toLabel, cypher.EqualityWhere(nodeInput), edgeProps)
		if _, err := c.drv.ExecuteWrite(ctx, relStmt); err != nil {
			return fmt.Errorf("create %s rel: %w", relType, err)
		}
	}
	return nil
}

// processNestedConnect handles "connect" nested inputs: matches an existing node and creates a relationship to it.
func (c *Client) processNestedConnect(ctx context.Context, fromLabel, toLabel, relType string, connects any) error {
	connectList, _ := connects.([]any)
	for _, connectItem := range connectList {
		ci, ok := connectItem.(map[string]any)
		if !ok {
			continue
		}
		where, _ := ci["where"].(map[string]any)

		matchStmt := cypher.NodeMatch(toLabel, cypher.EqualityWhere(where), nil)
		if _, err := c.drv.Execute(ctx, matchStmt); err != nil {
			return fmt.Errorf("connect match %s: %w", toLabel, err)
		}

		edgeProps := extractEdgeProps(ci)
		relStmt := cypher.RelCreate(fromLabel, cypher.EqualityWhere(map[string]any{}), relType, toLabel, cypher.EqualityWhere(where), edgeProps)
		if _, err := c.drv.ExecuteWrite(ctx, relStmt); err != nil {
			return fmt.Errorf("connect %s rel: %w", relType, err)
		}
	}
	return nil
}

// extractEdgeProps extracts the "edge" properties map from a nested input item.
func extractEdgeProps(input map[string]any) map[string]any {
	if ep, ok := input["edge"]; ok {
		if m, ok := ep.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

// splitNodeAndRelFields separates scalar node properties from relationship fields.
// Relationship fields are maps with "create"/"connect" keys.
func splitNodeAndRelFields(input map[string]any) (nodeProps, relFields map[string]any) {
	nodeProps = map[string]any{}
	relFields = map[string]any{}
	for k, v := range input {
		if m, ok := v.(map[string]any); ok {
			if _, hasCreate := m["create"]; hasCreate {
				relFields[k] = v
				continue
			}
			if _, hasConnect := m["connect"]; hasConnect {
				relFields[k] = v
				continue
			}
		}
		nodeProps[k] = v
	}
	return nodeProps, relFields
}

// recordsToSlice converts driver records to []any (each element is map[string]any).
func recordsToSlice(records []driver.Record) []any {
	result := make([]any, len(records))
	for i, rec := range records {
		result[i] = rec.Values
	}
	return result
}

// pluralFieldName returns the lowercase-first plural form for response field keys.
// e.g. "Movie" → "movies".
func pluralFieldName(label string) string {
	return strutil.PluralLower(label)
}

// singularize removes trailing 's' for simple plural → singular conversion.
func singularize(s string) string {
	if len(s) > 1 && s[len(s)-1] == 's' {
		return s[:len(s)-1]
	}
	return s
}

