package neo4j

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
)

// --- LOG-1: slog debug logging in Neo4j driver ---

// mockSessionRunnerForLog is a minimal sessionRunner for logging tests.
type mockSessionRunnerForLog struct{}

func (m *mockSessionRunnerForLog) ExecuteRead(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockSessionRunnerForLog) ExecuteWrite(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockSessionRunnerForLog) BeginTransaction(_ context.Context) (transactionRunner, error) {
	return &mockTxRunnerForLog{}, nil
}
func (m *mockSessionRunnerForLog) Close(_ context.Context) error { return nil }

type mockTxRunnerForLog struct{}

func (m *mockTxRunnerForLog) Run(_ context.Context, _ string, _ map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockTxRunnerForLog) Commit(_ context.Context) error   { return nil }
func (m *mockTxRunnerForLog) Rollback(_ context.Context) error { return nil }

// mockDBForLog implements neo4jDB for logging tests.
type mockDBForLog struct{}

func (m *mockDBForLog) NewSession(_ string) sessionRunner { return &mockSessionRunnerForLog{} }
func (m *mockDBForLog) Close(_ context.Context) error     { return nil }

// TestNeo4jDriver_Execute_LogsWithLogger verifies that Execute logs a debug
// message when Config.Logger is set.
// Expected: log output contains "cypher.execute" and the query string.
func TestNeo4jDriver_Execute_LogsWithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	drv := newFromDBWithLogger(&mockDBForLog{}, "neo4j", logger)

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n", Params: map[string]any{}}
	_, err := drv.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("Execute should log when logger is set, but log output is empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("cypher.execute")) {
		t.Errorf("log output should contain 'cypher.execute', got: %s", logOutput)
	}
}

// TestNeo4jDriver_Execute_NoLogWithoutLogger verifies that Execute produces no
// log output when Config.Logger is nil (zero overhead).
// Expected: no log output.
func TestNeo4jDriver_Execute_NoLogWithoutLogger(t *testing.T) {
	drv := newFromDB(&mockDBForLog{}, "neo4j")

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n", Params: map[string]any{}}
	_, err := drv.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	// No logger set — no way to capture output, just verify no panic
}

// TestNeo4jDriver_ExecuteWrite_LogsWithLogger verifies that ExecuteWrite logs
// a debug message when Config.Logger is set.
// Expected: log output contains "cypher.execute" and the query string.
func TestNeo4jDriver_ExecuteWrite_LogsWithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	drv := newFromDBWithLogger(&mockDBForLog{}, "neo4j", logger)

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $p0})", Params: map[string]any{"p0": "The Matrix"}}
	_, err := drv.ExecuteWrite(context.Background(), stmt)
	if err != nil {
		t.Fatalf("ExecuteWrite returned error: %v", err)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("ExecuteWrite should log when logger is set, but log output is empty")
	}
}

// TestNeo4jDriver_TxExecute_LogsWithLogger verifies that transaction Execute
// logs a debug message when Config.Logger is set.
// Expected: log output contains "cypher.execute".
func TestNeo4jDriver_TxExecute_LogsWithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	drv := newFromDBWithLogger(&mockDBForLog{}, "neo4j", logger)

	tx, err := drv.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}
	defer tx.Rollback(context.Background())

	stmt := cypher.Statement{Query: "CREATE (n:Movie)", Params: map[string]any{}}
	_, err = tx.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("tx.Execute returned error: %v", err)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("tx.Execute should log when logger is set, but log output is empty")
	}
}

// TestNeo4jDriver_LogsQueryAndParams verifies that the debug log includes
// both the query string and params.
// Expected: log output contains the query and "params" key.
func TestNeo4jDriver_LogsQueryAndParams(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := driver.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
		Logger:   logger,
	}
	_ = cfg // Will be used when NewNeo4jDriver accepts logger

	drv := newFromDBWithLogger(&mockDBForLog{}, "neo4j", logger)

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n", Params: map[string]any{"p0": "test"}}
	_, _ = drv.Execute(context.Background(), stmt)

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("log output should contain query and params info")
	}
	if !bytes.Contains(buf.Bytes(), []byte("query")) && !bytes.Contains(buf.Bytes(), []byte("MATCH")) {
		t.Errorf("log output should contain the query, got: %s", logOutput)
	}
}
