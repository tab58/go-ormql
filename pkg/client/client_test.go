package client

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- Mock ExecutableSchema ---

// mockExecutableSchema implements graphql.ExecutableSchema for testing.
type mockExecutableSchema struct {
	schemaFn     func() *ast.Schema
	complexityFn func(ctx context.Context, typeName, fieldName string, childComplexity int, args map[string]any) (int, bool)
	execFn       func(ctx context.Context) graphql.ResponseHandler
}

func (m *mockExecutableSchema) Schema() *ast.Schema {
	if m.schemaFn != nil {
		return m.schemaFn()
	}
	return &ast.Schema{}
}

func (m *mockExecutableSchema) Complexity(ctx context.Context, typeName, fieldName string, childComplexity int, args map[string]any) (int, bool) {
	if m.complexityFn != nil {
		return m.complexityFn(ctx, typeName, fieldName, childComplexity, args)
	}
	return 0, false
}

func (m *mockExecutableSchema) Exec(ctx context.Context) graphql.ResponseHandler {
	if m.execFn != nil {
		return m.execFn(ctx)
	}
	return nil
}

// --- Mock Driver ---

// mockDriver implements driver.Driver for testing.
type mockDriver struct {
	executeFn      func(ctx context.Context, stmt cypher.Statement) (driver.Result, error)
	executeWriteFn func(ctx context.Context, stmt cypher.Statement) (driver.Result, error)
	beginTxFn      func(ctx context.Context) (driver.Transaction, error)
	closeFn        func(ctx context.Context) error
	closed         bool
}

func (m *mockDriver) Execute(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, stmt)
	}
	return driver.Result{}, nil
}

func (m *mockDriver) ExecuteWrite(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	if m.executeWriteFn != nil {
		return m.executeWriteFn(ctx, stmt)
	}
	return driver.Result{}, nil
}

func (m *mockDriver) BeginTx(ctx context.Context) (driver.Transaction, error) {
	if m.beginTxFn != nil {
		return m.beginTxFn(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockDriver) Close(ctx context.Context) error {
	m.closed = true
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

// --- Tests ---

// TestNew_ReturnsNonNilClient verifies that New returns a non-nil *Client
// when given valid schema and driver.
// Expected: non-nil Client.
func TestNew_ReturnsNonNilClient(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{}

	c := New(es, drv)
	if c == nil {
		t.Fatal("New returned nil Client")
	}
}

// TestExecute_ValidQuery_ReturnsData verifies that Execute with a valid query
// returns response data as map[string]any.
// Expected: non-nil map with query results.
func TestExecute_ValidQuery_ReturnsData(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{}
	c := New(es, drv)

	result, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute returned nil result, want non-nil map[string]any")
	}
}

// TestExecute_EmptyQuery_ReturnsError verifies that Execute with an empty
// query string returns a parse error.
// Expected: non-nil error.
func TestExecute_EmptyQuery_ReturnsError(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{}
	c := New(es, drv)

	_, err := c.Execute(context.Background(), "", nil)
	if err == nil {
		t.Fatal("Execute with empty query should return error")
	}
}

// TestExecute_NilVariables_NoPanic verifies that Execute with nil variables
// does not panic and is treated as empty variables.
// Expected: no panic; returns result or error (not panic).
func TestExecute_NilVariables_NoPanic(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{}
	c := New(es, drv)

	// Should not panic with nil variables
	result, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	// Either result or error is fine, as long as no panic
	_ = result
	_ = err
}

// TestExecute_WithVariables_PassedToResolvers verifies that variables
// are passed through to the execution engine.
// Expected: Execute accepts variables without error.
func TestExecute_WithVariables_PassedToResolvers(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{}
	c := New(es, drv)

	vars := map[string]any{
		"input": []map[string]any{
			{"title": "The Matrix", "released": 1999},
		},
	}
	result, err := c.Execute(context.Background(), `mutation CreateMovie($input: [MovieCreateInput!]!) {
		createMovies(input: $input) { movies { title } }
	}`, vars)
	if err != nil {
		t.Fatalf("Execute with variables returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute with variables returned nil result")
	}
}

// TestClose_DelegatesToDriver verifies that Close calls driver.Close.
// Expected: driver.Close is called.
func TestClose_DelegatesToDriver(t *testing.T) {
	es := &mockExecutableSchema{}
	closeCalled := false
	drv := &mockDriver{
		closeFn: func(_ context.Context) error {
			closeCalled = true
			return nil
		},
	}
	c := New(es, drv)

	err := c.Close(context.Background())
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !closeCalled {
		t.Error("Close did not delegate to driver.Close")
	}
}

// TestClose_PropagatesDriverError verifies that Close propagates errors
// from the driver.
// Expected: error from driver is returned.
func TestClose_PropagatesDriverError(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{
		closeFn: func(_ context.Context) error {
			return errors.New("driver close failed")
		},
	}
	c := New(es, drv)

	err := c.Close(context.Background())
	if err == nil {
		t.Fatal("Close should propagate driver error")
	}
}

// TestClose_Idempotent verifies that calling Close multiple times does not
// panic and the second call is handled gracefully.
// Expected: no panic on multiple Close calls.
func TestClose_Idempotent(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{}
	c := New(es, drv)

	err1 := c.Close(context.Background())
	if err1 != nil {
		t.Fatalf("first Close returned error: %v", err1)
	}

	// Second close should not panic
	err2 := c.Close(context.Background())
	_ = err2 // May or may not error — key is no panic
}

// TestExecute_AfterClose_ReturnsError verifies that Execute after Close
// returns an error (from the driver being closed).
// Expected: non-nil error.
func TestExecute_AfterClose_ReturnsError(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{
		closeFn: func(_ context.Context) error {
			return nil
		},
	}
	c := New(es, drv)

	c.Close(context.Background())

	_, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err == nil {
		t.Fatal("Execute after Close should return error")
	}
}

// TestExecute_ConcurrentCalls_AreSafe verifies that multiple concurrent
// Execute calls do not race or panic.
// Expected: no data race, no panic.
func TestExecute_ConcurrentCalls_AreSafe(t *testing.T) {
	es := &mockExecutableSchema{}
	drv := &mockDriver{}
	c := New(es, drv)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Execute(context.Background(), `query { movies { title } }`, nil)
		}()
	}
	wg.Wait()
	// If we reach here without a race or panic, the test passes.
}
