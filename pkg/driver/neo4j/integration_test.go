//go:build integration

package neo4j

import (
	"context"
	"testing"
	"time"

	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
	tcneo4j "github.com/testcontainers/testcontainers-go/modules/neo4j"
)

// startNeo4jContainer starts a Neo4j testcontainer and returns the driver.Config
// with connection details. The container is terminated when the test ends.
func startNeo4jContainer(t *testing.T) driver.Config {
	t.Helper()
	ctx := context.Background()

	container, err := tcneo4j.Run(ctx,
		"neo4j:5",
		tcneo4j.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to start Neo4j container: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := container.Terminate(ctx); cleanupErr != nil {
			t.Logf("failed to terminate Neo4j container: %v", cleanupErr)
		}
	})

	boltURL, err := container.BoltUrl(ctx)
	if err != nil {
		t.Fatalf("failed to get Neo4j bolt URL: %v", err)
	}

	return driver.Config{
		URI:      boltURL,
		Username: "neo4j",
		Password: "",
		Database: "neo4j",
	}
}

// createTestDriver creates a real Neo4jDriver connected to the test container.
// Fails the test if driver creation fails.
func createTestDriver(t *testing.T, cfg driver.Config) driver.Driver {
	t.Helper()
	drv, err := NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		drv.Close(ctx)
	})
	return drv
}

// === INT-1: Neo4j integration tests ===

// TestIntegration_NewNeo4jDriver_Connects verifies that NewNeo4jDriver
// successfully connects to a real Neo4j instance.
// Expected: non-nil driver, nil error.
func TestIntegration_NewNeo4jDriver_Connects(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver should connect to Neo4j container: %v", err)
	}
	if drv == nil {
		t.Fatal("NewNeo4jDriver returned nil driver")
	}
	defer drv.Close(context.Background())
}

// TestIntegration_CreateNode verifies that a node can be created in Neo4j.
// Expected: ExecuteWrite with NodeCreate returns a result with at least one record.
func TestIntegration_CreateNode(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv := createTestDriver(t, cfg)
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

// TestIntegration_ReadNode verifies that a created node can be read back.
// Expected: Execute with NodeMatch returns the previously created node.
func TestIntegration_ReadNode(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv := createTestDriver(t, cfg)
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

// TestIntegration_UpdateNode verifies that a created node can be updated.
// Expected: ExecuteWrite with NodeUpdate returns a result with the updated node.
func TestIntegration_UpdateNode(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv := createTestDriver(t, cfg)
	ctx := context.Background()

	// Create a node
	createStmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "Interstellar",
		"released": 2014,
	})
	_, err := drv.ExecuteWrite(ctx, createStmt)
	if err != nil {
		t.Fatalf("create node failed: %v", err)
	}

	// Update it
	updateStmt := cypher.NodeUpdate("Movie",
		cypher.EqualityWhere(map[string]any{"title": "Interstellar"}),
		map[string]any{"released": 2015},
	)
	result, err := drv.ExecuteWrite(ctx, updateStmt)
	if err != nil {
		t.Fatalf("update node failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("update node returned no records, expected the updated node")
	}
}

// TestIntegration_DeleteNode verifies that a created node can be deleted.
// Expected: ExecuteWrite with NodeDelete succeeds; subsequent match returns 0 records.
func TestIntegration_DeleteNode(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv := createTestDriver(t, cfg)
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

// TestIntegration_BeginTx_CommitPersists verifies that operations within a
// committed transaction are persisted.
// Expected: node created inside tx is readable after Commit.
func TestIntegration_BeginTx_CommitPersists(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv := createTestDriver(t, cfg)
	ctx := context.Background()

	tx, err := drv.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}
	defer tx.Rollback(ctx)

	createStmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "TxCommitMovie",
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

	// Verify node persisted outside transaction
	matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": "TxCommitMovie"}), nil)
	result, err := drv.Execute(ctx, matchStmt)
	if err != nil {
		t.Fatalf("read after commit failed: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatal("committed node not found, expected 1 record")
	}
}

// TestIntegration_BeginTx_RollbackDiscards verifies that operations within a
// rolled-back transaction are NOT persisted.
// Expected: node created inside tx is NOT readable after Rollback.
func TestIntegration_BeginTx_RollbackDiscards(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv := createTestDriver(t, cfg)
	ctx := context.Background()

	tx, err := drv.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	createStmt := cypher.NodeCreate("Movie", map[string]any{
		"title":    "TxRollbackMovie",
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

	// Verify node was NOT persisted
	matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": "TxRollbackMovie"}), nil)
	result, err := drv.Execute(ctx, matchStmt)
	if err != nil {
		t.Fatalf("read after rollback failed: %v", err)
	}
	if len(result.Records) != 0 {
		t.Errorf("rolled-back node should not exist, got %d records", len(result.Records))
	}
}

// TestIntegration_BeginTx_MultiStatement verifies that multiple operations
// within a single transaction all succeed or all fail.
// Expected: both nodes created in a single committed tx are readable.
func TestIntegration_BeginTx_MultiStatement(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv := createTestDriver(t, cfg)
	ctx := context.Background()

	tx, err := drv.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create two nodes in same transaction
	stmt1 := cypher.NodeCreate("Movie", map[string]any{"title": "TxMulti1"})
	_, err = tx.Execute(ctx, stmt1)
	if err != nil {
		t.Fatalf("tx.Execute (create 1) failed: %v", err)
	}

	stmt2 := cypher.NodeCreate("Movie", map[string]any{"title": "TxMulti2"})
	_, err = tx.Execute(ctx, stmt2)
	if err != nil {
		t.Fatalf("tx.Execute (create 2) failed: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("tx.Commit failed: %v", err)
	}

	// Verify both nodes exist
	for _, title := range []string{"TxMulti1", "TxMulti2"} {
		matchStmt := cypher.NodeMatch("Movie", cypher.EqualityWhere(map[string]any{"title": title}), nil)
		result, err := drv.Execute(ctx, matchStmt)
		if err != nil {
			t.Fatalf("read %s failed: %v", title, err)
		}
		if len(result.Records) == 0 {
			t.Errorf("node %s not found after multi-statement commit", title)
		}
	}
}

// TestIntegration_Close_Idempotent verifies that calling Close multiple times
// on a real driver does not panic.
// Expected: no panic on double close.
func TestIntegration_Close_Idempotent(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
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
