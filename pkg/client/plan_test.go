package client

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
)

// === FE-4: Client.Execute() multi-statement handling ===
//
// Execute() must handle TranslationPlan:
// 1. Execute each WriteStatement sequentially via driver.ExecuteWrite
// 2. If any write fails, return error immediately (short-circuit)
// 3. Execute ReadStatement via Execute (queries) or ExecuteWrite (mutations)
// 4. Extract response from read result

// recordingDriver tracks all Execute and ExecuteWrite calls in order.
type recordingDriver struct {
	mu          sync.Mutex
	calls       []recordedCall
	executeFn   func(ctx context.Context, stmt cypher.Statement) (driver.Result, error)
	writeFailAt int // if >= 0, fail the Nth ExecuteWrite call
}

type recordedCall struct {
	method string // "Execute" or "ExecuteWrite"
	query  string
}

func (d *recordingDriver) Execute(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	d.mu.Lock()
	d.calls = append(d.calls, recordedCall{method: "Execute", query: stmt.Query})
	d.mu.Unlock()
	if d.executeFn != nil {
		return d.executeFn(ctx, stmt)
	}
	return driver.Result{
		Records: []driver.Record{{Values: map[string]any{"data": map[string]any{}}}},
	}, nil
}

func (d *recordingDriver) ExecuteWrite(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	d.mu.Lock()
	idx := len(d.calls)
	d.calls = append(d.calls, recordedCall{method: "ExecuteWrite", query: stmt.Query})
	d.mu.Unlock()
	if d.writeFailAt >= 0 && idx == d.writeFailAt {
		return driver.Result{}, errors.New("write failed")
	}
	return driver.Result{
		Records: []driver.Record{{Values: map[string]any{"data": map[string]any{}}}},
	}, nil
}

func (d *recordingDriver) BeginTx(ctx context.Context) (driver.Transaction, error) {
	return nil, errors.New("not implemented")
}

func (d *recordingDriver) Close(ctx context.Context) error {
	return nil
}

// augSchemaWithMerge extends the test schema to include merge mutation types.
const augSchemaWithMerge = `
type Query {
	movies: [Movie!]!
}
type Mutation {
	createMovies(input: [MovieCreateInput!]!): CreateMoviesMutationResponse!
	mergeMovies(input: [MovieMergeInput!]!): MergeMoviesMutationResponse!
}
type Movie {
	id: ID!
	title: String!
}
input MovieCreateInput {
	title: String!
}
input MovieMergeInput {
	match: MovieMatchInput!
	onCreate: MovieCreateInput
	onMatch: MovieCreateInput
}
input MovieMatchInput {
	title: String
}
type CreateMoviesMutationResponse {
	movies: [Movie!]!
}
type MergeMoviesMutationResponse {
	movies: [Movie!]!
}
`

// Test: Query operation skips WriteStatements phase entirely.
// Expected: only Execute is called, never ExecuteWrite.
func TestExecutePlan_QuerySkipsWrites(t *testing.T) {
	drv := &recordingDriver{writeFailAt: -1}
	c := New(testModel(), testAugSchemaSDL, drv)

	_, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drv.mu.Lock()
	defer drv.mu.Unlock()
	for _, call := range drv.calls {
		if call.method == "ExecuteWrite" {
			t.Error("query should NOT call ExecuteWrite, but it did")
		}
	}
	if len(drv.calls) == 0 {
		t.Error("expected at least one driver call for query")
	}
}

// Test: Mutation with WriteStatements executes writes before read.
// Expected: WriteStatements executed via ExecuteWrite BEFORE the ReadStatement.
// FAILS RED: current Execute() doesn't handle WriteStatements from TranslationPlan.
func TestExecutePlan_MergeWritesBeforeRead(t *testing.T) {
	drv := &recordingDriver{writeFailAt: -1}
	c := New(testModel(), augSchemaWithMerge, drv)

	_, err := c.Execute(context.Background(),
		`mutation { mergeMovies(input: [{match: {title: "X"}}]) { movies { title } } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drv.mu.Lock()
	defer drv.mu.Unlock()

	// We expect at least 2 calls: write(s) then read
	if len(drv.calls) < 2 {
		t.Fatalf("expected at least 2 driver calls for merge mutation, got %d", len(drv.calls))
	}

	// First call(s) should be ExecuteWrite for FOREACH
	foundForeachWrite := false
	for _, call := range drv.calls[:len(drv.calls)-1] {
		if call.method == "ExecuteWrite" && strings.Contains(call.query, "FOREACH") {
			foundForeachWrite = true
		}
	}
	if !foundForeachWrite {
		t.Errorf("expected ExecuteWrite with FOREACH before read, calls: %+v", drv.calls)
	}

	// Last call should be ExecuteWrite for the read query (mutations use ExecuteWrite)
	lastCall := drv.calls[len(drv.calls)-1]
	if lastCall.method != "ExecuteWrite" {
		t.Errorf("expected final call to be ExecuteWrite for mutation read, got %s", lastCall.method)
	}
}

// Test: Write failure short-circuits — ReadStatement is never executed.
// Expected: error returned, read call never happens after the failing FOREACH write.
// FAILS RED: current Execute() doesn't execute WriteStatements separately.
func TestExecutePlan_WriteFailureShortCircuits(t *testing.T) {
	drv := &recordingDriver{writeFailAt: 0} // fail the first ExecuteWrite
	c := New(testModel(), augSchemaWithMerge, drv)

	_, err := c.Execute(context.Background(),
		`mutation { mergeMovies(input: [{match: {title: "X"}}]) { movies { title } } }`, nil)

	drv.mu.Lock()
	calls := append([]recordedCall{}, drv.calls...)
	drv.mu.Unlock()

	// Once the FOREACH write phase is implemented, the first ExecuteWrite should
	// be the FOREACH write (which fails), and the read ExecuteWrite should never happen.
	// Count how many calls contain FOREACH — should be exactly 1 (the failing write).
	foreachCalls := 0
	for _, call := range calls {
		if strings.Contains(call.query, "FOREACH") {
			foreachCalls++
		}
	}

	// FAILS RED: current code never produces FOREACH writes, so foreachCalls == 0
	if foreachCalls == 0 {
		t.Error("expected at least one FOREACH write call (which should fail), got 0 — WriteStatements not being executed")
	}

	// When properly implemented, the FOREACH write fails and error is returned
	if foreachCalls > 0 && err == nil {
		t.Error("expected error when FOREACH write fails, got nil")
	}
}

// Test: Create-only mutation does not produce any write phase calls.
// Expected: only one ExecuteWrite for the read (mutation), no FOREACH writes.
func TestExecutePlan_CreateMutationNoWritePhase(t *testing.T) {
	drv := &recordingDriver{writeFailAt: -1}
	c := New(testModel(), testAugSchemaSDL, drv)

	_, err := c.Execute(context.Background(),
		`mutation { createMovies(input: [{title: "X"}]) { movies { title } } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drv.mu.Lock()
	defer drv.mu.Unlock()

	// Should be exactly one call (ExecuteWrite for the mutation)
	if len(drv.calls) != 1 {
		t.Fatalf("expected exactly 1 driver call for create mutation, got %d: %+v", len(drv.calls), drv.calls)
	}
	if drv.calls[0].method != "ExecuteWrite" {
		t.Errorf("expected ExecuteWrite for create mutation, got %s", drv.calls[0].method)
	}
}
