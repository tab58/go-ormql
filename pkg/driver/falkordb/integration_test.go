//go:build integration

package falkordb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// === FK-INT-1: FalkorDB integration tests with testcontainers ===

// startFalkorDBContainer starts a FalkorDB testcontainer and returns the driver.Config
// with connection details. The container is terminated when the test ends.
func startFalkorDBContainer(t *testing.T) driver.Config {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "falkordb/falkordb:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start FalkorDB container: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := container.Terminate(ctx); cleanupErr != nil {
			t.Logf("failed to terminate FalkorDB container: %v", cleanupErr)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get FalkorDB container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get FalkorDB container port: %v", err)
	}

	return driver.Config{
		Host:     host,
		Port:     port.Int(),
		Scheme:   "redis",
		Database: "test_graph",
	}
}

// createTestFalkorDBDriver creates a real FalkorDBDriver connected to the test container.
// Fails the test if driver creation fails.
func createTestFalkorDBDriver(t *testing.T, cfg driver.Config) driver.Driver {
	t.Helper()
	drv, err := NewFalkorDBDriver(cfg)
	if err != nil {
		t.Fatalf("NewFalkorDBDriver failed: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		drv.Close(ctx)
	})
	return drv
}

// --- Connection ---

// Test: NewFalkorDBDriver connects to a real FalkorDB instance.
// Expected: non-nil driver, nil error.
func TestIntegration_NewFalkorDBDriver_Connects(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv, err := NewFalkorDBDriver(cfg)
	if err != nil {
		t.Fatalf("NewFalkorDBDriver should connect to FalkorDB container: %v", err)
	}
	if drv == nil {
		t.Fatal("NewFalkorDBDriver returned nil driver")
	}
	defer drv.Close(context.Background())
}

// --- CRUD operations ---

// Test: Create a node in FalkorDB via ExecuteWrite.
// Expected: ExecuteWrite with NodeCreate returns a result.
func TestIntegration_CreateNode(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv := createTestFalkorDBDriver(t, cfg)
	ctx := context.Background()

	stmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "The Matrix",
		"released": 1999,
	})
	result, err := drv.ExecuteWrite(ctx, stmt)
	if err != nil {
		t.Fatalf("create node failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("create node returned no records, want at least 1")
	}
}

// Test: Read a created node back from FalkorDB via Execute.
// Expected: Execute with NodeMatch returns the previously created node.
func TestIntegration_ReadNode(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv := createTestFalkorDBDriver(t, cfg)
	ctx := context.Background()

	// Create a node
	createStmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "Inception",
		"released": 2010,
	})
	_, err := drv.ExecuteWrite(ctx, createStmt)
	if err != nil {
		t.Fatalf("create node failed: %v", err)
	}

	// Read it back
	matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": "Inception"}), nil)
	result, err := drv.Execute(ctx, matchStmt)
	if err != nil {
		t.Fatalf("read node failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("read node returned no records, expected the created node")
	}
}

// Test: Delete a created node from FalkorDB.
// Expected: ExecuteWrite with NodeDelete succeeds; subsequent match returns 0 records.
func TestIntegration_DeleteNode(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv := createTestFalkorDBDriver(t, cfg)
	ctx := context.Background()

	// Create a node
	createStmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "Tenet",
		"released": 2020,
	})
	_, err := drv.ExecuteWrite(ctx, createStmt)
	if err != nil {
		t.Fatalf("create node failed: %v", err)
	}

	// Delete it
	deleteStmt := cypher.NodeDelete("Movie", cypher.EqualityWhere(map[string]any{"title": "Tenet"}))
	_, err = drv.ExecuteWrite(ctx, deleteStmt)
	if err != nil {
		t.Fatalf("delete node failed: %v", err)
	}

	// Verify it's gone
	matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": "Tenet"}), nil)
	result, err := drv.Execute(ctx, matchStmt)
	if err != nil {
		t.Fatalf("read after delete failed: %v", err)
	}
	if len(result.Records) != 0 {
		t.Errorf("expected 0 records after delete, got %d", len(result.Records))
	}
}

// --- Single-query transaction ---

// Test: Single-query transaction commit persists the operation.
// Expected: node created inside tx is readable after Commit.
func TestIntegration_Tx_CommitPersists(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv := createTestFalkorDBDriver(t, cfg)
	ctx := context.Background()

	tx, err := drv.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	createStmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "FKTxCommitMovie",
		"released": 2025,
	})
	_, err = tx.Execute(ctx, createStmt)
	if err != nil {
		t.Fatalf("tx.Execute (create) failed: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("tx.Commit failed: %v", err)
	}

	// Verify node persisted
	matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": "FKTxCommitMovie"}), nil)
	result, err := drv.Execute(ctx, matchStmt)
	if err != nil {
		t.Fatalf("read after commit failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("committed node not found, expected 1 record")
	}
}

// Test: Single-query transaction rollback discards the operation.
// Expected: node created inside tx is NOT readable after Rollback.
func TestIntegration_Tx_RollbackDiscards(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv := createTestFalkorDBDriver(t, cfg)
	ctx := context.Background()

	tx, err := drv.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	createStmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "FKTxRollbackMovie",
		"released": 2025,
	})
	_, err = tx.Execute(ctx, createStmt)
	if err != nil {
		t.Fatalf("tx.Execute (create) failed: %v", err)
	}

	err = tx.Rollback(ctx)
	if err != nil {
		t.Fatalf("tx.Rollback failed: %v", err)
	}

	// Verify node was NOT persisted (rollback = discard buffered statement)
	matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": "FKTxRollbackMovie"}), nil)
	result, err := drv.Execute(ctx, matchStmt)
	if err != nil {
		t.Fatalf("read after rollback failed: %v", err)
	}
	if len(result.Records) != 0 {
		t.Errorf("rolled-back node should not exist, got %d records", len(result.Records))
	}
}

// --- Close ---

// Test: Close is idempotent — second call does not panic.
// Expected: both calls succeed without panic.
func TestIntegration_Close_Idempotent(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv, err := NewFalkorDBDriver(cfg)
	if err != nil {
		t.Fatalf("NewFalkorDBDriver failed: %v", err)
	}
	ctx := context.Background()

	err = drv.Close(ctx)
	if err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	// Second close should not panic
	err = drv.Close(ctx)
	_ = err // may or may not error
}

// --- Error handling ---

// Test: Invalid query returns a clear error.
// Expected: ExecuteWrite with invalid Cypher returns non-nil error.
func TestIntegration_InvalidQuery(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	drv := createTestFalkorDBDriver(t, cfg)
	ctx := context.Background()

	stmt := cypher.Statement{Query: "THIS IS NOT VALID CYPHER"}
	_, err := drv.ExecuteWrite(ctx, stmt)
	if err == nil {
		t.Fatal("invalid Cypher should return error")
	}
}

// Test: NewFalkorDBDriver with invalid host returns error.
// Expected: non-nil error, error message contains host.
func TestIntegration_NewFalkorDBDriver_InvalidHost(t *testing.T) {
	cfg := driver.Config{
		Host:     "invalid-falkordb-host",
		Port:     0,
		Scheme:   "redis",
		Database: "test_graph",
	}

	_, err := NewFalkorDBDriver(cfg)
	if err == nil {
		t.Fatal("NewFalkorDBDriver should return error for unreachable host")
	}

	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Error("error message should not be empty")
	}
	// Verify error includes host for debuggability
	if !containsStr(errMsg, "invalid-falkordb-host") {
		t.Errorf("error message should contain host, got: %s", errMsg)
	}
}

// containsStr checks if s contains substr.
func containsStr(s, substr string) bool {
	return fmt.Sprintf("%s", s) != "" && len(s) >= len(substr) && findStr(s, substr)
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// === FK-INT-2: FalkorDB vector query rewrite integration tests ===

// createTestFalkorDBDriverWithVectorIndexes creates a FalkorDBDriver with VectorIndexes configured.
func createTestFalkorDBDriverWithVectorIndexes(t *testing.T, cfg driver.Config, indexes map[string]driver.VectorIndex) driver.Driver {
	t.Helper()
	cfg.VectorIndexes = indexes
	drv, err := NewFalkorDBDriver(cfg)
	if err != nil {
		t.Fatalf("NewFalkorDBDriver failed: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		drv.Close(ctx)
	})
	return drv
}

// Test: Vector query rewrite integration — create vector index, insert data, query with similarity.
// Expected: similarity query executes successfully via FalkorDB's db.idx.vector.queryNodes.
func TestIntegration_VectorQueryRewrite(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	indexes := map[string]driver.VectorIndex{
		"movie_embeddings": {Label: "Movie", Property: "embedding"},
	}
	drv := createTestFalkorDBDriverWithVectorIndexes(t, cfg, indexes)
	ctx := context.Background()

	// Create vector index in FalkorDB
	createIdxStmt := cypher.Statement{
		Query: "CREATE VECTOR INDEX FOR (n:Movie) ON (n.embedding) OPTIONS {dimension: 3, similarityFunction: 'euclidean'}",
	}
	_, err := drv.ExecuteWrite(ctx, createIdxStmt)
	if err != nil {
		t.Fatalf("create vector index failed: %v", err)
	}

	// Insert a node with embedding
	createNodeStmt := cypher.Statement{
		Query:  "CREATE (n:Movie {title: $title, embedding: vecf32($vec)}) RETURN n",
		Params: map[string]any{"title": "Matrix", "vec": []any{0.1, 0.2, 0.3}},
	}
	_, err = drv.ExecuteWrite(ctx, createNodeStmt)
	if err != nil {
		t.Fatalf("create node with embedding failed: %v", err)
	}

	// Execute similarity query using Neo4j-style procedure (should be rewritten by driver)
	similarStmt := cypher.Statement{
		Query:  "CALL db.index.vector.queryNodes($p0, $p1, $p2) YIELD node AS n, score RETURN n.title AS title, score",
		Params: map[string]any{"p0": "movie_embeddings", "p1": int64(5), "p2": []any{0.1, 0.2, 0.3}},
	}
	result, err := drv.Execute(ctx, similarStmt)
	if err != nil {
		t.Fatalf("vector similarity query failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("vector similarity query returned no records, expected at least 1")
	}
}

// Test: Vector query rewrite — non-vector query passes through unchanged with VectorIndexes configured.
// Expected: standard MATCH query succeeds even when VectorIndexes are configured.
func TestIntegration_VectorRewrite_NonVectorPassthrough(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	indexes := map[string]driver.VectorIndex{
		"movie_embeddings": {Label: "Movie", Property: "embedding"},
	}
	drv := createTestFalkorDBDriverWithVectorIndexes(t, cfg, indexes)
	ctx := context.Background()

	// Create a normal node
	createStmt := cypher.NodeCreate("Movie", map[string]any{"title": "NonVectorMovie"})
	_, err := drv.ExecuteWrite(ctx, createStmt)
	if err != nil {
		t.Fatalf("create node failed: %v", err)
	}

	// Standard query should work fine even with VectorIndexes configured
	matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": "NonVectorMovie"}), nil)
	result, err := drv.Execute(ctx, matchStmt)
	if err != nil {
		t.Fatalf("non-vector query with VectorIndexes configured failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("non-vector query returned no records, expected the created node")
	}
}

// Test: Vector similarity query returns score field in results.
// Expected: result records contain a "score" field with a numeric value.
func TestIntegration_VectorRewrite_ScoreInResults(t *testing.T) {
	cfg := startFalkorDBContainer(t)
	indexes := map[string]driver.VectorIndex{
		"movie_embeddings": {Label: "Movie", Property: "embedding"},
	}
	drv := createTestFalkorDBDriverWithVectorIndexes(t, cfg, indexes)
	ctx := context.Background()

	// Create vector index
	createIdxStmt := cypher.Statement{
		Query: "CREATE VECTOR INDEX FOR (n:Movie) ON (n.embedding) OPTIONS {dimension: 3, similarityFunction: 'euclidean'}",
	}
	drv.ExecuteWrite(ctx, createIdxStmt)

	// Insert nodes with embeddings
	for i, title := range []string{"A", "B", "C"} {
		vec := []any{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3}
		stmt := cypher.Statement{
			Query:  "CREATE (n:Movie {title: $title, embedding: vecf32($vec)})",
			Params: map[string]any{"title": title, "vec": vec},
		}
		drv.ExecuteWrite(ctx, stmt)
	}

	// Similarity query — should return results with score
	similarStmt := cypher.Statement{
		Query:  "CALL db.index.vector.queryNodes($p0, $p1, $p2) YIELD node AS n, score RETURN n.title AS title, score",
		Params: map[string]any{"p0": "movie_embeddings", "p1": int64(3), "p2": []any{0.1, 0.2, 0.3}},
	}
	result, err := drv.Execute(ctx, similarStmt)
	if err != nil {
		t.Fatalf("vector similarity query failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("expected at least 1 result from similarity query")
	}

	// Verify score field exists
	firstRecord := result.Records[0]
	if _, ok := firstRecord.Values["score"]; !ok {
		t.Error("result should contain 'score' field from similarity query")
	}
}
