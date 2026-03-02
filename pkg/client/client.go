package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/tab58/go-ormql/pkg/driver"
	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/tab58/go-ormql/pkg/translate"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// errNilDriver is the panic message when New() is called with a nil driver.
const errNilDriver = "gormql: driver must not be nil"

// errEmptyModel is the panic message when New() is called with a zero-node model.
const errEmptyModel = "gormql: model must have at least one node"

// errClientClosed is returned when Execute is called on a closed client.
var errClientClosed = errors.New("client is closed")

// resultDataKey is the key used to extract the response map from driver records.
const resultDataKey = "data"

// Result wraps the GraphQL response data from a single execution.
// The data map mirrors the GraphQL JSON response shape.
type Result struct {
	data map[string]any
}

// Decode unmarshals the result data into the target struct.
// Uses JSON marshal/unmarshal — generated model structs have json tags.
func (r *Result) Decode(v any) error {
	b, err := json.Marshal(r.data)
	if err != nil {
		return fmt.Errorf("failed to marshal result data: %w", err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("failed to decode result: %w", err)
	}
	return nil
}

// Data returns the raw response map (for dynamic access without generated types).
// Returns a copy to prevent mutation.
func (r *Result) Data() map[string]any {
	if r.data == nil {
		return map[string]any{}
	}
	cp := make(map[string]any, len(r.data))
	for k, v := range r.data {
		cp[k] = v
	}
	return cp
}

// Client provides a programmatic Go API for executing GraphQL queries and
// mutations against a Cypher-backed graph database. Uses pkg/translate for
// GraphQL-to-Cypher translation and gqlparser for query validation.
type Client struct {
	translator *translate.Translator
	augSchema  *ast.Schema
	drv        driver.Driver
	logger     *slog.Logger
	mu         sync.Mutex
	closed     bool
}

// New creates a Client from a GraphModel, augmented schema SDL, and driver.
// Constructs the translator from the model. Parses the augmented schema for
// query validation via gqlparser.
// Panics if model has zero nodes or drv is nil.
func New(model schema.GraphModel, augSchemaSDL string, drv driver.Driver, opts ...Option) *Client {
	if drv == nil {
		panic(errNilDriver)
	}
	if len(model.Nodes) == 0 {
		panic(errEmptyModel)
	}

	// Apply options
	options := &clientOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Parse augmented schema for validation
	schemaDoc, parseErr := gqlparser.LoadSchema(&ast.Source{Input: augSchemaSDL})
	if parseErr != nil {
		// Schema parsing failure is a programming error — panic
		panic(fmt.Sprintf("gormql: failed to parse augmented schema: %v", parseErr))
	}

	return &Client{
		translator: translate.New(model),
		augSchema:  schemaDoc,
		drv:        drv,
		logger:     options.logger,
	}
}

// isClosed reports whether the client has been closed.
// Thread-safe: reads the closed flag under the mutex.
func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Execute runs a GraphQL query or mutation and returns a Result.
//
// Execution flow:
// 1. Check closed state.
// 2. Log the query at slog.LevelDebug if logger is configured.
// 3. Parse the query with gqlparser.
// 4. Translate to a single Cypher Statement via pkg/translate.
// 5. Execute against the driver (one database round-trip).
//    Queries use driver.Execute (read). Mutations use driver.ExecuteWrite (write).
// 6. Extract records[0].Values["data"] as the response map.
// 7. Return &Result{data: responseMap}.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]any) (*Result, error) {
	if c.isClosed() {
		return nil, errClientClosed
	}

	// Log the query
	if c.logger != nil {
		c.logger.Debug("graphql.execute", "query", query, "variables", variables)
	}

	// Parse the query
	if query == "" {
		return nil, errors.New("empty query")
	}

	doc, parseErr := gqlparser.LoadQuery(c.augSchema, query)
	if parseErr != nil {
		return nil, fmt.Errorf("query parse error: %v", parseErr)
	}

	if len(doc.Operations) == 0 {
		return nil, errors.New("no operations in query")
	}

	// Use the first operation
	op := doc.Operations[0]

	// Translate to Cypher
	stmt, err := c.translator.Translate(doc, op, variables)
	if err != nil {
		return nil, fmt.Errorf("translation error: %w", err)
	}

	// Execute against driver — queries use Execute, mutations use ExecuteWrite
	var drvResult driver.Result
	switch op.Operation {
	case ast.Mutation:
		drvResult, err = c.drv.ExecuteWrite(ctx, stmt)
	default:
		drvResult, err = c.drv.Execute(ctx, stmt)
	}
	if err != nil {
		return nil, err
	}

	// Extract records[0].Values["data"]
	var data map[string]any
	if len(drvResult.Records) > 0 {
		if d, ok := drvResult.Records[0].Values[resultDataKey]; ok {
			if m, ok := d.(map[string]any); ok {
				data = m
			}
		}
	}

	return &Result{data: data}, nil
}

// Close releases the underlying driver resources and marks the client as closed.
// Delegates to driver.Close(ctx). Idempotent.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	if c.drv != nil {
		return c.drv.Close(ctx)
	}
	return nil
}
