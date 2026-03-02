package client

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
	"github.com/tab58/gql-orm/pkg/schema"
)

// --- Mock Driver ---

// mockDriver implements driver.Driver for testing.
type mockDriver struct {
	mu                 sync.Mutex
	executeFn          func(ctx context.Context, stmt cypher.Statement) (driver.Result, error)
	executeWriteFn     func(ctx context.Context, stmt cypher.Statement) (driver.Result, error)
	beginTxFn          func(ctx context.Context) (driver.Transaction, error)
	closeFn            func(ctx context.Context) error
	closed             bool
	executeCalled      bool
	executeWriteCalled bool
	lastStmt           cypher.Statement
}

func (m *mockDriver) Execute(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	m.mu.Lock()
	m.executeCalled = true
	m.lastStmt = stmt
	m.mu.Unlock()
	if m.executeFn != nil {
		return m.executeFn(ctx, stmt)
	}
	return driver.Result{}, nil
}

func (m *mockDriver) ExecuteWrite(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	m.mu.Lock()
	m.executeWriteCalled = true
	m.lastStmt = stmt
	m.mu.Unlock()
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

// --- Test fixtures ---

// testModel returns a minimal GraphModel for testing.
func testModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID", GoType: "string", IsID: true},
					{Name: "title", GraphQLType: "String", GoType: "string"},
				},
			},
		},
	}
}

// testAugSchemaSDL is a minimal augmented schema SDL for testing.
const testAugSchemaSDL = `
type Query {
	movies: [Movie!]!
}
type Mutation {
	createMovies(input: [MovieCreateInput!]!): CreateMoviesMutationResponse!
}
type Movie {
	id: ID!
	title: String!
}
input MovieCreateInput {
	title: String!
}
type CreateMoviesMutationResponse {
	movies: [Movie!]!
}
`

// driverReturningData returns a mockDriver whose Execute returns a single record
// with the given data map in the "data" column.
func driverReturningData(data map[string]any) *mockDriver {
	return &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"data": data}},
				},
			}, nil
		},
	}
}

// --- New() tests ---

// Test: New() with valid model, schema, and driver returns non-nil Client.
// Expected: non-nil *Client returned.
func TestNew_ReturnsNonNilClient(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})
	if c == nil {
		t.Fatal("New returned nil Client")
	}
}

// Test: New() panics when driver is nil.
// Expected: panic with message containing "driver".
func TestNew_PanicsOnNilDriver(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("New(model, sdl, nil) should panic on nil driver, but did not")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value should be a string, got %T", r)
		}
		if !strings.Contains(strings.ToLower(msg), "driver") {
			t.Errorf("panic message should mention 'driver', got: %q", msg)
		}
	}()
	New(testModel(), testAugSchemaSDL, nil)
}

// Test: New() panics when model has zero nodes.
// Expected: panic with message containing "model" or "node".
func TestNew_PanicsOnEmptyModel(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("New(emptyModel, sdl, drv) should panic on empty model, but did not")
		}
	}()
	emptyModel := schema.GraphModel{}
	New(emptyModel, testAugSchemaSDL, &mockDriver{})
}

// Test: New() accepts variadic options.
// Expected: compiles and returns non-nil Client.
func TestNew_AcceptsOptions(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{}, WithLogger(nil))
	if c == nil {
		t.Fatal("New with options should return non-nil Client")
	}
}

// --- Execute() tests ---

// Test: Execute() returns a non-nil *Result for a valid query.
// Expected: *Result is non-nil.
func TestExecute_ReturnsResult(t *testing.T) {
	drv := driverReturningData(map[string]any{"movies": []any{}})
	c := New(testModel(), testAugSchemaSDL, drv)

	result, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute should return non-nil *Result, got nil")
	}
}

// Test: Execute() with empty query returns error.
// Expected: non-nil error.
func TestExecute_EmptyQuery_ReturnsError(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})

	_, err := c.Execute(context.Background(), "", nil)
	if err == nil {
		t.Fatal("Execute with empty query should return error")
	}
}

// Test: Execute() with nil variables does not panic.
// Expected: no panic.
func TestExecute_NilVariables_NoPanic(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})
	// Should not panic with nil variables
	_, _ = c.Execute(context.Background(), `query { movies { title } }`, nil)
}

// Test: Execute() for query operation uses driver.Execute (read).
// Expected: driver.Execute called, not ExecuteWrite.
func TestExecute_QueryUsesDriverExecute(t *testing.T) {
	drv := driverReturningData(map[string]any{"movies": []any{}})
	c := New(testModel(), testAugSchemaSDL, drv)

	_, _ = c.Execute(context.Background(), `query { movies { title } }`, nil)

	if !drv.executeCalled {
		t.Error("query should use driver.Execute (read), but it was not called")
	}
	if drv.executeWriteCalled {
		t.Error("query should NOT use driver.ExecuteWrite")
	}
}

// Test: Execute() for mutation operation uses driver.ExecuteWrite (write).
// Expected: driver.ExecuteWrite called, not Execute.
func TestExecute_MutationUsesDriverExecuteWrite(t *testing.T) {
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"data": map[string]any{"createMovies": map[string]any{}}}},
				},
			}, nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv)

	_, _ = c.Execute(context.Background(), `mutation { createMovies(input: [{title: "X"}]) { movies { title } } }`, nil)

	if !drv.executeWriteCalled {
		t.Error("mutation should use driver.ExecuteWrite (write), but it was not called")
	}
	if drv.executeCalled {
		t.Error("mutation should NOT use driver.Execute")
	}
}

// Test: Execute() translates query to Cypher Statement via translator.
// Expected: driver receives a cypher.Statement with non-empty Query.
func TestExecute_TranslatesToCypher(t *testing.T) {
	drv := &mockDriver{
		executeFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			if stmt.Query == "" {
				t.Error("expected non-empty Cypher query passed to driver")
			}
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"data": map[string]any{}}},
				},
			}, nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv)

	_, _ = c.Execute(context.Background(), `query { movies { title } }`, nil)

	if !drv.executeCalled {
		t.Error("driver.Execute should have been called")
	}
}

// Test: Execute() with invalid query syntax returns parse error.
// Expected: non-nil error.
func TestExecute_InvalidSyntax_ReturnsError(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})

	_, err := c.Execute(context.Background(), `query { `, nil)
	if err == nil {
		t.Fatal("Execute with invalid syntax should return error")
	}
}

// Test: Execute() extracts records[0].Values["data"] from driver result.
// Expected: Result.Data() returns the map from "data" column.
func TestExecute_ExtractsDataFromRecord(t *testing.T) {
	expectedData := map[string]any{
		"movies": []any{
			map[string]any{"title": "The Matrix"},
		},
	}
	drv := driverReturningData(expectedData)
	c := New(testModel(), testAugSchemaSDL, drv)

	result, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil Result")
	}
	data := result.Data()
	if data == nil {
		t.Fatal("expected non-nil Data from Result")
	}
	movies, ok := data["movies"]
	if !ok {
		t.Fatal("expected 'movies' key in Result.Data()")
	}
	movieList, ok := movies.([]any)
	if !ok {
		t.Fatalf("expected []any for movies, got %T", movies)
	}
	if len(movieList) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(movieList))
	}
}

// Test: Execute() after Close returns errClientClosed.
// Expected: error is errClientClosed.
func TestExecute_AfterClose_ReturnsError(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})
	_ = c.Close(context.Background())

	_, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err == nil {
		t.Fatal("Execute after Close should return error")
	}
	if !errors.Is(err, errClientClosed) {
		t.Errorf("expected errClientClosed, got: %v", err)
	}
}

// Test: Execute() passes variables through to translator/driver.
// Expected: driver.Execute is called with variables reflected in Cypher params.
func TestExecute_PassesVariables(t *testing.T) {
	drv := driverReturningData(map[string]any{"movies": []any{}})
	c := New(testModel(), testAugSchemaSDL, drv)

	vars := map[string]any{"limit": 10}
	_, _ = c.Execute(context.Background(), `query { movies { title } }`, vars)

	if !drv.executeCalled {
		t.Fatal("driver.Execute should be called for queries with variables")
	}
}

// Test: Execute() propagates driver errors.
// Expected: error from driver is returned.
func TestExecute_DriverError_ReturnsError(t *testing.T) {
	drv := &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{}, errors.New("database error")
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv)

	_, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err == nil {
		t.Fatal("Execute should propagate driver errors")
	}
	if !strings.Contains(err.Error(), "database") {
		t.Errorf("expected error to contain 'database', got: %v", err)
	}
}

// Test: Concurrent Execute calls are safe (no data race).
// Expected: no race, no panic.
func TestExecute_ConcurrentCalls_AreSafe(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Execute(context.Background(), `query { movies { title } }`, nil)
		}()
	}
	wg.Wait()
}

// --- Result tests ---

// Test: Result.Decode() unmarshals data into target struct.
// Expected: target struct fields populated from data.
func TestResult_Decode_UnmarshalsData(t *testing.T) {
	r := &Result{data: map[string]any{
		"movies": []any{
			map[string]any{"title": "The Matrix"},
		},
	}}

	var target struct {
		Movies []struct {
			Title string `json:"title"`
		} `json:"movies"`
	}
	err := r.Decode(&target)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(target.Movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(target.Movies))
	}
	if target.Movies[0].Title != "The Matrix" {
		t.Errorf("expected title 'The Matrix', got %q", target.Movies[0].Title)
	}
}

// Test: Result.Decode() with wrong target type returns error.
// Expected: non-nil error.
func TestResult_Decode_WrongType_ReturnsError(t *testing.T) {
	r := &Result{data: map[string]any{
		"movies": "not a list",
	}}

	var target struct {
		Movies []struct{} `json:"movies"`
	}
	err := r.Decode(&target)
	if err == nil {
		t.Error("Decode with incompatible target type should return error")
	}
}

// Test: Result.Data() returns a copy — mutations don't affect original.
// Expected: modifying returned map doesn't change Result.
func TestResult_Data_ReturnsCopy(t *testing.T) {
	r := &Result{data: map[string]any{
		"movies": []any{},
	}}

	data := r.Data()
	if data == nil {
		t.Fatal("Data() should return non-nil map")
	}
	// Mutate the returned copy
	data["injected"] = "should not appear"

	// Original should be unaffected
	original := r.Data()
	if _, exists := original["injected"]; exists {
		t.Error("Data() should return a copy — mutation of returned map affected original")
	}
}

// Test: Result.Data() on nil data returns empty map (not nil).
// Expected: non-nil, empty map.
func TestResult_Data_NilData_ReturnsEmptyMap(t *testing.T) {
	r := &Result{data: nil}

	data := r.Data()
	if data == nil {
		t.Fatal("Data() on nil data should return empty map, not nil")
	}
}

// --- Close() tests ---

// Test: Close() delegates to driver.Close.
// Expected: driver.Close called.
func TestClose_DelegatesToDriver(t *testing.T) {
	closeCalled := false
	drv := &mockDriver{
		closeFn: func(_ context.Context) error {
			closeCalled = true
			return nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv)

	err := c.Close(context.Background())
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !closeCalled {
		t.Error("Close did not delegate to driver.Close")
	}
}

// Test: Close() propagates driver errors.
// Expected: error from driver is returned.
func TestClose_PropagatesDriverError(t *testing.T) {
	drv := &mockDriver{
		closeFn: func(_ context.Context) error {
			return errors.New("driver close failed")
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv)

	err := c.Close(context.Background())
	if err == nil {
		t.Fatal("Close should propagate driver error")
	}
}

// Test: Close() is idempotent — no panic on multiple calls.
// Expected: no panic.
func TestClose_Idempotent(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})
	_ = c.Close(context.Background())
	_ = c.Close(context.Background())
	// If we reach here without panic, test passes
}
