package neo4j

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
)

// --- Mock types ---

// mockTransactionRunner implements transactionRunner for unit testing.
type mockTransactionRunner struct {
	runFn      func(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
	commitFn   func(ctx context.Context) error
	rollbackFn func(ctx context.Context) error
}

func (m *mockTransactionRunner) Run(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	if m.runFn != nil {
		return m.runFn(ctx, query, params)
	}
	return nil, nil
}

func (m *mockTransactionRunner) Commit(ctx context.Context) error {
	if m.commitFn != nil {
		return m.commitFn(ctx)
	}
	return nil
}

func (m *mockTransactionRunner) Rollback(ctx context.Context) error {
	if m.rollbackFn != nil {
		return m.rollbackFn(ctx)
	}
	return nil
}

// mockSessionRunner implements sessionRunner for unit testing.
type mockSessionRunner struct {
	executeReadFn      func(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
	executeWriteFn     func(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
	beginTransactionFn func(ctx context.Context) (transactionRunner, error)
	closeFn            func(ctx context.Context) error
}

func (m *mockSessionRunner) ExecuteRead(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	if m.executeReadFn != nil {
		return m.executeReadFn(ctx, query, params)
	}
	return nil, nil
}

func (m *mockSessionRunner) ExecuteWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	if m.executeWriteFn != nil {
		return m.executeWriteFn(ctx, query, params)
	}
	return nil, nil
}

func (m *mockSessionRunner) BeginTransaction(ctx context.Context) (transactionRunner, error) {
	if m.beginTransactionFn != nil {
		return m.beginTransactionFn(ctx)
	}
	return &mockTransactionRunner{}, nil
}

func (m *mockSessionRunner) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

// mockNeo4jDB implements neo4jDB for unit testing.
type mockNeo4jDB struct {
	session *mockSessionRunner
	closeFn func(ctx context.Context) error
}

func (m *mockNeo4jDB) NewSession(_ string) sessionRunner {
	return m.session
}

func (m *mockNeo4jDB) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

// --- Interface satisfaction test ---

// TestNeo4jDriverSatisfiesInterface verifies at compile time that Neo4jDriver implements driver.Driver.
func TestNeo4jDriverSatisfiesInterface(t *testing.T) {
	var _ driver.Driver = &Neo4jDriver{}
}

// --- NewNeo4jDriver tests ---

// TestNewNeo4jDriver_ReturnsError verifies that the stub constructor returns an error
// (no real Neo4j to connect to). Once implemented, this confirms connection setup.
func TestNewNeo4jDriver_ReturnsError(t *testing.T) {
	_, err := NewNeo4jDriver(driver.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	})
	if err == nil {
		t.Fatal("NewNeo4jDriver stub should return error")
	}
}

// --- Execute tests ---

// TestExecute_ReturnsResult verifies that Execute runs a read query and returns flattened records.
// Expected: driver.Result with records containing the query results.
func TestExecute_ReturnsResult(t *testing.T) {
	session := &mockSessionRunner{
		executeReadFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
			return []map[string]any{
				{"title": "Matrix", "released": int64(1999)},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	stmt := cypher.Statement{
		Query:  "MATCH (n:Movie) WHERE n.title = $title RETURN n",
		Params: map[string]any{"title": "Matrix"},
	}

	result, err := drv.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(result.Records))
	}
	if result.Records[0].Values["title"] != "Matrix" {
		t.Errorf("Records[0][\"title\"] = %v, want %q", result.Records[0].Values["title"], "Matrix")
	}
}

// TestExecute_PropagatesError verifies that errors from the session are propagated to the caller.
func TestExecute_PropagatesError(t *testing.T) {
	session := &mockSessionRunner{
		executeReadFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
			return nil, errors.New("connection lost")
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := drv.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute should propagate error from session")
	}
}

// TestExecute_AfterClose verifies that Execute after Close returns a clear error.
func TestExecute_AfterClose(t *testing.T) {
	session := &mockSessionRunner{}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	drv.Close(context.Background())

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := drv.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute after Close should return error")
	}
}

// --- ExecuteWrite tests ---

// TestExecuteWrite_ReturnsResult verifies that ExecuteWrite runs a write query and returns results.
func TestExecuteWrite_ReturnsResult(t *testing.T) {
	session := &mockSessionRunner{
		executeWriteFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
			return []map[string]any{
				{"title": "New Movie"},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	stmt := cypher.Statement{
		Query:  "CREATE (n:Movie {title: $title}) RETURN n",
		Params: map[string]any{"title": "New Movie"},
	}

	result, err := drv.ExecuteWrite(context.Background(), stmt)
	if err != nil {
		t.Fatalf("ExecuteWrite returned error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(result.Records))
	}
}

// TestExecuteWrite_PropagatesError verifies that write operation errors propagate.
func TestExecuteWrite_PropagatesError(t *testing.T) {
	session := &mockSessionRunner{
		executeWriteFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
			return nil, errors.New("write failed")
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title}) RETURN n"}
	_, err := drv.ExecuteWrite(context.Background(), stmt)
	if err == nil {
		t.Fatal("ExecuteWrite should propagate error")
	}
}

// --- Close tests ---

// TestClose_Succeeds verifies that Close calls the underlying driver close and returns nil.
func TestClose_Succeeds(t *testing.T) {
	db := &mockNeo4jDB{}
	drv := newFromDB(db, "neo4j")

	err := drv.Close(context.Background())
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

// TestClose_Idempotent verifies that calling Close twice does not panic or return error on second call.
func TestClose_Idempotent(t *testing.T) {
	db := &mockNeo4jDB{}
	drv := newFromDB(db, "neo4j")

	if err := drv.Close(context.Background()); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	if err := drv.Close(context.Background()); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

// TestClose_PropagatesError verifies that Close propagates errors from the underlying driver.
func TestClose_PropagatesError(t *testing.T) {
	db := &mockNeo4jDB{
		closeFn: func(_ context.Context) error {
			return errors.New("close failed")
		},
	}
	drv := newFromDB(db, "neo4j")

	err := drv.Close(context.Background())
	if err == nil {
		t.Fatal("Close should propagate error")
	}
}

// --- Result flattening tests ---

// TestExecute_NilValuesMapToGoNil verifies that neo4j null values become Go nil in Record.Values.
func TestExecute_NilValuesMapToGoNil(t *testing.T) {
	session := &mockSessionRunner{
		executeReadFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
			return []map[string]any{
				{"title": "Matrix", "tagline": nil},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	result, err := drv.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(result.Records))
	}
	if _, exists := result.Records[0].Values["tagline"]; !exists {
		t.Error("Record missing key \"tagline\" — nil values should be preserved")
	}
	if result.Records[0].Values["tagline"] != nil {
		t.Errorf("Record[\"tagline\"] = %v, want nil", result.Records[0].Values["tagline"])
	}
}

// === DR-4: Neo4j Transaction support tests ===

// TestNeo4jTransactionSatisfiesInterface verifies at compile time that
// neo4jTransaction implements driver.Transaction.
func TestNeo4jTransactionSatisfiesInterface(t *testing.T) {
	var _ driver.Transaction = &neo4jTransaction{}
}

// TestBeginTx_ReturnsTransaction verifies that BeginTx opens a session,
// begins a transaction, and returns a non-nil driver.Transaction.
// Expected: non-nil Transaction, nil error.
func TestBeginTx_ReturnsTransaction(t *testing.T) {
	session := &mockSessionRunner{
		beginTransactionFn: func(_ context.Context) (transactionRunner, error) {
			return &mockTransactionRunner{}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}
	if tx == nil {
		t.Fatal("BeginTx returned nil Transaction")
	}
}

// TestBeginTx_SessionBeginTransactionError verifies that if the underlying
// session.BeginTransaction fails, BeginTx propagates the error.
func TestBeginTx_SessionBeginTransactionError(t *testing.T) {
	session := &mockSessionRunner{
		beginTransactionFn: func(_ context.Context) (transactionRunner, error) {
			return nil, errors.New("begin transaction failed")
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	_, err := drv.BeginTx(context.Background())
	if err == nil {
		t.Fatal("BeginTx should propagate session.BeginTransaction error")
	}
}

// TestTx_Execute_DelegatesToUnderlyingTransaction verifies that
// neo4jTransaction.Execute delegates to the underlying transactionRunner.Run
// and returns flattened results.
// Expected: Result with records matching the mock.
func TestTx_Execute_DelegatesToUnderlyingTransaction(t *testing.T) {
	session := &mockSessionRunner{
		beginTransactionFn: func(_ context.Context) (transactionRunner, error) {
			return &mockTransactionRunner{
				runFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
					return []map[string]any{
						{"title": "Matrix"},
					}, nil
				},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{
		Query:  "CREATE (n:Movie {title: $title}) RETURN n",
		Params: map[string]any{"title": "Matrix"},
	}
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

// TestTx_Execute_PropagatesError verifies that errors from the underlying
// transaction are propagated through neo4jTransaction.Execute.
func TestTx_Execute_PropagatesError(t *testing.T) {
	session := &mockSessionRunner{
		beginTransactionFn: func(_ context.Context) (transactionRunner, error) {
			return &mockTransactionRunner{
				runFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
					return nil, errors.New("tx run failed")
				},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title})"}
	_, err = tx.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("tx.Execute should propagate underlying error")
	}
}

// TestTx_Commit_DelegatesToUnderlyingTransaction verifies that
// neo4jTransaction.Commit calls the underlying transaction's Commit.
// Expected: nil error on success.
func TestTx_Commit_DelegatesToUnderlyingTransaction(t *testing.T) {
	committed := false
	session := &mockSessionRunner{
		beginTransactionFn: func(_ context.Context) (transactionRunner, error) {
			return &mockTransactionRunner{
				commitFn: func(_ context.Context) error {
					committed = true
					return nil
				},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("tx.Commit returned error: %v", err)
	}
	if !committed {
		t.Error("tx.Commit did not call underlying transaction.Commit")
	}
}

// TestTx_Rollback_DelegatesToUnderlyingTransaction verifies that
// neo4jTransaction.Rollback calls the underlying transaction's Rollback.
// Expected: nil error on success.
func TestTx_Rollback_DelegatesToUnderlyingTransaction(t *testing.T) {
	rolledBack := false
	session := &mockSessionRunner{
		beginTransactionFn: func(_ context.Context) (transactionRunner, error) {
			return &mockTransactionRunner{
				rollbackFn: func(_ context.Context) error {
					rolledBack = true
					return nil
				},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	err = tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("tx.Rollback returned error: %v", err)
	}
	if !rolledBack {
		t.Error("tx.Rollback did not call underlying transaction.Rollback")
	}
}

// TestTx_RollbackAfterCommit_IsNoOp verifies that calling Rollback after
// Commit on neo4jTransaction is a no-op (does not call underlying Rollback).
// This is the critical pattern for: defer tx.Rollback(ctx).
func TestTx_RollbackAfterCommit_IsNoOp(t *testing.T) {
	rolledBack := false
	session := &mockSessionRunner{
		beginTransactionFn: func(_ context.Context) (transactionRunner, error) {
			return &mockTransactionRunner{
				rollbackFn: func(_ context.Context) error {
					rolledBack = true
					return nil
				},
			}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	tx.Commit(context.Background())

	err = tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Rollback after Commit returned error: %v (want nil, no-op)", err)
	}
	if rolledBack {
		t.Error("Rollback after Commit should be a no-op and NOT call underlying Rollback")
	}
}

// TestBeginTx_AfterClose_ReturnsError verifies that BeginTx after Close
// returns a clear error.
func TestBeginTx_AfterClose_ReturnsError(t *testing.T) {
	session := &mockSessionRunner{}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	drv.Close(context.Background())

	_, err := drv.BeginTx(context.Background())
	if err == nil {
		t.Fatal("BeginTx after Close should return error")
	}
}

// TestNeo4jDriver_ConcurrentCloseAndExecute verifies that concurrent Close
// and Execute calls do not race (tests the sync.Mutex fix for M2).
// Expected: no data race detected by -race flag. Some calls may error, but no panic.
func TestNeo4jDriver_ConcurrentCloseAndExecute(t *testing.T) {
	session := &mockSessionRunner{
		executeReadFn: func(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
			return []map[string]any{{"n": 1}}, nil
		},
	}
	db := &mockNeo4jDB{session: session}
	drv := newFromDB(db, "neo4j")

	stmt := cypher.Statement{Query: "MATCH (n) RETURN n"}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			drv.Execute(context.Background(), stmt)
		}()
		go func() {
			defer wg.Done()
			drv.Close(context.Background())
		}()
	}
	wg.Wait()
	// If we reach here without a data race, the test passes.
}
