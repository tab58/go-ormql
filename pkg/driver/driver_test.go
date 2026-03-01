package driver

import (
	"context"
	"errors"
	"testing"

	"github.com/tab58/gql-orm/pkg/cypher"
)

// --- Mock Transaction ---

// mockTransaction is a test double that implements the Transaction interface.
// Each method delegates to a configurable function field.
type mockTransaction struct {
	executeFn  func(ctx context.Context, stmt cypher.Statement) (Result, error)
	commitFn   func(ctx context.Context) error
	rollbackFn func(ctx context.Context) error
	committed  bool
	rolledBack bool
}

func (t *mockTransaction) Execute(ctx context.Context, stmt cypher.Statement) (Result, error) {
	if t.committed {
		return Result{}, errors.New("transaction already committed")
	}
	if t.rolledBack {
		return Result{}, errors.New("transaction already rolled back")
	}
	if t.executeFn != nil {
		return t.executeFn(ctx, stmt)
	}
	return Result{}, nil
}

func (t *mockTransaction) Commit(ctx context.Context) error {
	if t.committed {
		return errors.New("transaction already committed")
	}
	t.committed = true
	if t.commitFn != nil {
		return t.commitFn(ctx)
	}
	return nil
}

func (t *mockTransaction) Rollback(ctx context.Context) error {
	if t.committed {
		return nil // no-op after commit
	}
	t.rolledBack = true
	if t.rollbackFn != nil {
		return t.rollbackFn(ctx)
	}
	return nil
}

// --- Mock driver ---

// mockDriver is a test double that implements the Driver interface.
// Each method delegates to a configurable function field.
type mockDriver struct {
	executeFn      func(ctx context.Context, stmt cypher.Statement) (Result, error)
	executeWriteFn func(ctx context.Context, stmt cypher.Statement) (Result, error)
	beginTxFn      func(ctx context.Context) (Transaction, error)
	closeFn        func(ctx context.Context) error
	closed         bool
}

func (m *mockDriver) Execute(ctx context.Context, stmt cypher.Statement) (Result, error) {
	if m.closed {
		return Result{}, errors.New("driver is closed")
	}
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if m.executeFn != nil {
		return m.executeFn(ctx, stmt)
	}
	return Result{}, nil
}

func (m *mockDriver) ExecuteWrite(ctx context.Context, stmt cypher.Statement) (Result, error) {
	if m.closed {
		return Result{}, errors.New("driver is closed")
	}
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if m.executeWriteFn != nil {
		return m.executeWriteFn(ctx, stmt)
	}
	return Result{}, nil
}

func (m *mockDriver) BeginTx(ctx context.Context) (Transaction, error) {
	if m.closed {
		return nil, errors.New("driver is closed")
	}
	if m.beginTxFn != nil {
		return m.beginTxFn(ctx)
	}
	return &mockTransaction{}, nil
}

func (m *mockDriver) Close(ctx context.Context) error {
	m.closed = true
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

// --- Interface satisfaction test ---

// TestMockSatisfiesInterface verifies that mockDriver implements Driver at compile time.
// This is a compile-time check — if mockDriver doesn't implement Driver, this won't compile.
func TestMockSatisfiesInterface(t *testing.T) {
	var _ Driver = &mockDriver{}
}

// --- Execute tests ---

// TestExecute_ReturnsExpectedResult verifies that Execute returns the Result from the driver.
// Expected: caller receives the exact records the mock produces.
func TestExecute_ReturnsExpectedResult(t *testing.T) {
	mock := &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{
				Records: []Record{
					{Values: map[string]any{"title": "Matrix", "released": int64(1999)}},
					{Values: map[string]any{"title": "John Wick", "released": int64(2014)}},
				},
			}, nil
		},
	}

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n", Params: nil}
	result, err := mock.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("len(Records) = %d, want 2", len(result.Records))
	}
	if result.Records[0].Values["title"] != "Matrix" {
		t.Errorf("Records[0][\"title\"] = %v, want %q", result.Records[0].Values["title"], "Matrix")
	}
}

// TestExecute_PropagatesError verifies that an error from Execute is returned to the caller.
func TestExecute_PropagatesError(t *testing.T) {
	mock := &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{}, errors.New("connection lost")
		},
	}

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := mock.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute returned nil error, want error")
	}
	if err.Error() != "connection lost" {
		t.Errorf("error = %q, want %q", err.Error(), "connection lost")
	}
}

// --- ExecuteWrite tests ---

// TestExecuteWrite_ReturnsExpectedResult verifies that ExecuteWrite returns the Result from the driver.
func TestExecuteWrite_ReturnsExpectedResult(t *testing.T) {
	mock := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{
				Records: []Record{
					{Values: map[string]any{"title": "New Movie"}},
				},
			}, nil
		},
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title}) RETURN n", Params: map[string]any{"title": "New Movie"}}
	result, err := mock.ExecuteWrite(context.Background(), stmt)
	if err != nil {
		t.Fatalf("ExecuteWrite returned error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(result.Records))
	}
}

// TestExecuteWrite_PropagatesError verifies that an error from ExecuteWrite is returned to the caller.
func TestExecuteWrite_PropagatesError(t *testing.T) {
	mock := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{}, errors.New("write failed")
		},
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title}) RETURN n"}
	_, err := mock.ExecuteWrite(context.Background(), stmt)
	if err == nil {
		t.Fatal("ExecuteWrite returned nil error, want error")
	}
	if err.Error() != "write failed" {
		t.Errorf("error = %q, want %q", err.Error(), "write failed")
	}
}

// --- Close tests ---

// TestClose_Succeeds verifies that Close returns nil error on success.
func TestClose_Succeeds(t *testing.T) {
	mock := &mockDriver{}

	err := mock.Close(context.Background())
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

// TestClose_PropagatesError verifies that a Close error is returned to the caller.
func TestClose_PropagatesError(t *testing.T) {
	mock := &mockDriver{
		closeFn: func(_ context.Context) error {
			return errors.New("close failed")
		},
	}

	err := mock.Close(context.Background())
	if err == nil {
		t.Fatal("Close returned nil error, want error")
	}
	if err.Error() != "close failed" {
		t.Errorf("error = %q, want %q", err.Error(), "close failed")
	}
}

// TestClose_Idempotent verifies that calling Close twice does not panic.
func TestClose_Idempotent(t *testing.T) {
	mock := &mockDriver{}

	if err := mock.Close(context.Background()); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	// Second close should not panic
	if err := mock.Close(context.Background()); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

// --- Context tests ---

// TestExecute_CancelledContext verifies that a cancelled context produces an error.
func TestExecute_CancelledContext(t *testing.T) {
	mock := &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{Records: []Record{{Values: map[string]any{"n": 1}}}}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := mock.Execute(ctx, stmt)
	if err == nil {
		t.Fatal("Execute with cancelled context returned nil error, want error")
	}
}

// --- Execute after Close test ---

// TestExecute_AfterClose verifies that calling Execute after Close returns a clear error.
func TestExecute_AfterClose(t *testing.T) {
	mock := &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{Records: []Record{{Values: map[string]any{"n": 1}}}}, nil
		},
	}

	mock.Close(context.Background())

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := mock.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute after Close returned nil error, want error")
	}
}

// === DR-3: Transaction interface and BeginTx tests ===

// TestMockTransactionSatisfiesInterface verifies at compile time that
// mockTransaction implements the Transaction interface.
func TestMockTransactionSatisfiesInterface(t *testing.T) {
	var _ Transaction = &mockTransaction{}
}

// TestBeginTx_ReturnsUsableTransaction verifies that BeginTx returns
// a non-nil Transaction that can Execute, Commit, and Rollback.
// Expected: transaction is non-nil and Execute returns a result.
func TestBeginTx_ReturnsUsableTransaction(t *testing.T) {
	mock := &mockDriver{}

	tx, err := mock.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}
	if tx == nil {
		t.Fatal("BeginTx returned nil Transaction")
	}

	// Should be able to execute a statement
	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title})", Params: map[string]any{"title": "Test"}}
	_, err = tx.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("tx.Execute returned error: %v", err)
	}
}

// TestTransaction_Execute_ReturnsResult verifies that Transaction.Execute
// returns the expected Result.
// Expected: caller receives the exact records the mock produces.
func TestTransaction_Execute_ReturnsResult(t *testing.T) {
	tx := &mockTransaction{
		executeFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{
				Records: []Record{
					{Values: map[string]any{"title": "Matrix"}},
				},
			}, nil
		},
	}

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	result, err := tx.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("tx.Execute returned error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(result.Records))
	}
	if result.Records[0].Values["title"] != "Matrix" {
		t.Errorf("Records[0][\"title\"] = %v, want %q", result.Records[0].Values["title"], "Matrix")
	}
}

// TestTransaction_Execute_PropagatesError verifies that an error from
// Transaction.Execute is returned to the caller.
func TestTransaction_Execute_PropagatesError(t *testing.T) {
	tx := &mockTransaction{
		executeFn: func(_ context.Context, _ cypher.Statement) (Result, error) {
			return Result{}, errors.New("tx execute failed")
		},
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title})"}
	_, err := tx.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("tx.Execute should propagate error")
	}
	if err.Error() != "tx execute failed" {
		t.Errorf("error = %q, want %q", err.Error(), "tx execute failed")
	}
}

// TestTransaction_Commit_Succeeds verifies that Commit returns nil on success.
func TestTransaction_Commit_Succeeds(t *testing.T) {
	tx := &mockTransaction{}

	err := tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit returned error: %v", err)
	}
}

// TestTransaction_Commit_PropagatesError verifies that a Commit error is propagated.
func TestTransaction_Commit_PropagatesError(t *testing.T) {
	tx := &mockTransaction{
		commitFn: func(_ context.Context) error {
			return errors.New("commit failed")
		},
	}

	err := tx.Commit(context.Background())
	if err == nil {
		t.Fatal("Commit should propagate error")
	}
	if err.Error() != "commit failed" {
		t.Errorf("error = %q, want %q", err.Error(), "commit failed")
	}
}

// TestTransaction_Rollback_Succeeds verifies that Rollback returns nil on success.
func TestTransaction_Rollback_Succeeds(t *testing.T) {
	tx := &mockTransaction{}

	err := tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Rollback returned error: %v", err)
	}
}

// TestTransaction_RollbackAfterCommit_IsNoOp verifies that calling Rollback
// after Commit is a no-op and returns nil error.
// This is the critical pattern: defer tx.Rollback(ctx) is always safe.
func TestTransaction_RollbackAfterCommit_IsNoOp(t *testing.T) {
	tx := &mockTransaction{}

	err := tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit returned error: %v", err)
	}

	// Rollback after Commit should be a no-op
	err = tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Rollback after Commit returned error: %v (want nil, no-op)", err)
	}
}

// TestTransaction_ExecuteAfterCommit_ReturnsError verifies that calling
// Execute after Commit returns an error.
func TestTransaction_ExecuteAfterCommit_ReturnsError(t *testing.T) {
	tx := &mockTransaction{}

	tx.Commit(context.Background())

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := tx.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute after Commit should return error")
	}
}

// TestTransaction_ExecuteAfterRollback_ReturnsError verifies that calling
// Execute after Rollback returns an error.
func TestTransaction_ExecuteAfterRollback_ReturnsError(t *testing.T) {
	tx := &mockTransaction{}

	tx.Rollback(context.Background())

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := tx.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute after Rollback should return error")
	}
}

// TestTransaction_CommitAfterCommit_ReturnsError verifies that calling
// Commit on an already-committed transaction returns an error.
func TestTransaction_CommitAfterCommit_ReturnsError(t *testing.T) {
	tx := &mockTransaction{}

	tx.Commit(context.Background())

	err := tx.Commit(context.Background())
	if err == nil {
		t.Fatal("Commit after Commit should return error")
	}
}

// TestBeginTx_AfterClose_ReturnsError verifies that calling BeginTx
// after Close returns a clear error.
func TestBeginTx_AfterClose_ReturnsError(t *testing.T) {
	mock := &mockDriver{}
	mock.Close(context.Background())

	_, err := mock.BeginTx(context.Background())
	if err == nil {
		t.Fatal("BeginTx after Close should return error")
	}
}
