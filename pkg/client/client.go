package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/tab58/gql-orm/pkg/driver"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// Client provides a programmatic Go API for executing GraphQL queries and
// mutations against a Cypher-backed graph database. Wraps gqlgen's
// ExecutableSchema as an in-memory execution engine.
type Client struct {
	schema graphql.ExecutableSchema
	drv    driver.Driver
	logger *slog.Logger
	mu     sync.Mutex
	closed bool
}

// New creates a Client from a gqlgen ExecutableSchema and a driver.
// The schema is typically produced by generated code (NewExecutableSchema).
// The driver is used for lifecycle management (Close).
// Options (e.g., WithLogger) configure optional behavior.
func New(schema graphql.ExecutableSchema, drv driver.Driver, opts ...Option) *Client {
	var options clientOptions
	for _, o := range opts {
		o(&options)
	}
	return &Client{schema: schema, drv: drv, logger: options.logger}
}

// Execute runs a GraphQL query or mutation and returns the response data.
// The result mirrors the GraphQL JSON response shape as map[string]any.
// Errors from GraphQL execution (validation, resolver errors) are returned as error.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()

	if closed {
		return nil, fmt.Errorf("client is closed")
	}

	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is empty")
	}

	if variables == nil {
		variables = map[string]any{}
	}

	if c.logger != nil {
		c.logger.Debug("graphql.execute", "query", query, "variables", variables)
	}

	// When no gqlgen schema is configured, use direct GraphQL-to-Cypher execution.
	if c.schema == nil {
		return c.directExecute(ctx, query, variables)
	}

	// Parse the query syntax (without schema validation so mock schemas work).
	src := &ast.Source{Name: "query", Input: query}
	doc, parseErr := parser.ParseQuery(src)
	if parseErr != nil {
		return nil, fmt.Errorf("query parse error: %v", parseErr)
	}

	var op *ast.OperationDefinition
	if len(doc.Operations) > 0 {
		op = doc.Operations[0]
	}

	// Build gqlgen operation context and dispatch to schema execution.
	oc := &graphql.OperationContext{
		RawQuery:  query,
		Variables: variables,
		Doc:       doc,
		Operation: op,
	}
	ctx = graphql.WithOperationContext(ctx, oc)

	handler := c.schema.Exec(ctx)
	if handler == nil {
		return map[string]any{}, nil
	}

	resp := handler(ctx)
	if resp == nil {
		return map[string]any{}, nil
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql errors: %v", resp.Errors)
	}

	if len(resp.Data) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// Close releases the underlying driver resources.
// Delegates to driver.Close(ctx). Idempotent.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return c.drv.Close(ctx)
}
