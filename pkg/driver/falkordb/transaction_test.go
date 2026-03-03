package falkordb

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
)

// === FK-2: FalkorDB single-query transaction tests ===

// --- Interface satisfaction ---

// Test: falkordbTransaction implements driver.Transaction interface.
// Expected: compile-time assertion succeeds.
func TestFalkordbTransactionSatisfiesInterface(t *testing.T) {
	var _ driver.Transaction = &falkordbTransaction{}
}

// --- BeginTx tests ---

// Test: BeginTx returns a non-nil Transaction.
// Expected: non-nil Transaction, nil error.
func TestBeginTx_ReturnsTransaction(t *testing.T) {
	graph := &mockGraphRunner{}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}
	if tx == nil {
		t.Fatal("BeginTx returned nil Transaction")
	}
}

// Test: BeginTx after Close returns error.
// Expected: non-nil error.
func TestBeginTx_AfterClose(t *testing.T) {
	graph := &mockGraphRunner{}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	drv.Close(context.Background())

	_, err := drv.BeginTx(context.Background())
	if err == nil {
		t.Fatal("BeginTx after Close should return error")
	}
}

// --- Single Execute (buffer) ---

// Test: First Execute on transaction buffers the statement and returns empty Result.
// The graph.Query should NOT be called yet (deferred to Commit).
// Expected: nil error, empty Result, statement buffered.
func TestTx_FirstExecute_BuffersStatement(t *testing.T) {
	queryCalled := false
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			queryCalled = true
			return &mockResultIterator{}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title})", Params: map[string]any{"title": "Matrix"}}
	result, err := tx.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("first tx.Execute returned error: %v", err)
	}
	// Result should be empty (deferred execution)
	if len(result.Records) != 0 {
		t.Errorf("first Execute should return empty Result, got %d records", len(result.Records))
	}
	// graph.Query should NOT have been called yet
	if queryCalled {
		t.Error("graph.Query should not be called on first Execute (deferred to Commit)")
	}
}

// --- Double Execute (error) ---

// Test: Second Execute on transaction returns errSingleStatementOnly.
// Expected: error on second Execute call.
func TestTx_SecondExecute_ReturnsError(t *testing.T) {
	graph := &mockGraphRunner{}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie)"}
	_, err = tx.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("first Execute returned error: %v", err)
	}

	// Second Execute should return error
	stmt2 := cypher.Statement{Query: "CREATE (n:Actor)"}
	_, err = tx.Execute(context.Background(), stmt2)
	if err == nil {
		t.Fatal("second Execute should return errSingleStatementOnly")
	}
	if !strings.Contains(err.Error(), "single") && !strings.Contains(err.Error(), "one statement") {
		t.Errorf("error should mention single statement, got: %v", err)
	}
}

// --- Execute after Commit (error) ---

// Test: Execute after Commit returns errTransactionCommitted.
// Expected: error containing "committed".
func TestTx_ExecuteAfterCommit_ReturnsError(t *testing.T) {
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{records: []map[string]any{{"n": 1}}}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie)"}
	tx.Execute(context.Background(), stmt)
	tx.Commit(context.Background())

	_, err = tx.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute after Commit should return error")
	}
	if !strings.Contains(err.Error(), "committed") {
		t.Errorf("error should mention 'committed', got: %v", err)
	}
}

// --- Commit executes buffered statement ---

// Test: Commit executes the buffered statement via graph.Query.
// Expected: graph.Query called with the buffered query string on Commit.
func TestTx_Commit_ExecutesBuffered(t *testing.T) {
	var executedQuery string
	graph := &mockGraphRunner{
		queryFn: func(query string, _ map[string]any) (resultIterator, error) {
			executedQuery = query
			return &mockResultIterator{records: []map[string]any{{"n": 1}}}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: 'Matrix'})"}
	tx.Execute(context.Background(), stmt)

	err = tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit returned error: %v", err)
	}
	if executedQuery != stmt.Query {
		t.Errorf("Commit executed query %q, want %q", executedQuery, stmt.Query)
	}
}

// Test: Commit propagates errors from graph.Query.
// Expected: non-nil error from Commit.
func TestTx_Commit_PropagatesError(t *testing.T) {
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return nil, errors.New("query execution failed")
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie)"}
	tx.Execute(context.Background(), stmt)

	err = tx.Commit(context.Background())
	if err == nil {
		t.Fatal("Commit should propagate error from graph.Query")
	}
}

// --- Commit with no buffered statement ---

// Test: Commit on empty transaction (no Execute called) is a no-op.
// Expected: nil error, graph.Query NOT called.
func TestTx_Commit_EmptyNoOp(t *testing.T) {
	queryCalled := false
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			queryCalled = true
			return &mockResultIterator{}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit on empty tx returned error: %v", err)
	}
	if queryCalled {
		t.Error("Commit on empty tx should NOT call graph.Query")
	}
}

// --- Rollback tests ---

// Test: Rollback discards buffered statement without executing.
// Expected: nil error, graph.Query NOT called.
func TestTx_Rollback_DiscardsBuffered(t *testing.T) {
	queryCalled := false
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			queryCalled = true
			return &mockResultIterator{}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie)"}
	tx.Execute(context.Background(), stmt)

	err = tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Rollback returned error: %v", err)
	}
	if queryCalled {
		t.Error("Rollback should NOT execute the buffered statement")
	}
}

// Test: Rollback after Commit is a no-op.
// Expected: nil error.
func TestTx_RollbackAfterCommit_NoOp(t *testing.T) {
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie)"}
	tx.Execute(context.Background(), stmt)
	tx.Commit(context.Background())

	err = tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Rollback after Commit returned error: %v (want nil, no-op)", err)
	}
}

// --- Concurrency tests ---

// Test: Concurrent Execute and Commit calls do not race.
// Expected: no data race detected by -race flag.
func TestTx_ConcurrentExecuteAndCommit(t *testing.T) {
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{records: []map[string]any{{"n": 1}}}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie)"}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tx.Execute(context.Background(), stmt)
		}()
		go func() {
			defer wg.Done()
			tx.Commit(context.Background())
		}()
	}
	wg.Wait()
}

// Test: Concurrent Commit and Rollback calls do not race.
// Expected: no data race detected by -race flag.
func TestTx_ConcurrentCommitAndRollback(t *testing.T) {
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tx.Commit(context.Background())
		}()
		go func() {
			defer wg.Done()
			tx.Rollback(context.Background())
		}()
	}
	wg.Wait()
}
