package driver

import (
	"context"
	"log/slog"

	"github.com/tab58/go-ormql/pkg/cypher"
)

// VectorIndex maps a named vector index to its label and property.
// Used by FalkorDB driver for vector query rewriting.
type VectorIndex struct {
	Label    string
	Property string
}

// Config holds connection configuration for a graph database driver.
// Credentials come from environment variables or explicit config, never hardcoded.
type Config struct {
	Host   string
	Port   int
	Scheme string
	Username string
	Password string
	Database string
	// VectorIndexes maps index name → VectorIndex for FalkorDB vector query rewriting.
	// Optional — only needed when using FalkorDB with @vector directives.
	VectorIndexes map[string]VectorIndex
	// Logger enables debug logging of Cypher queries.
	// When non-nil, the driver logs slog.Debug("cypher.execute", ...) before each query.
	// When nil (default), no logging overhead occurs.
	Logger *slog.Logger
}

// Record represents a single result row with named fields.
// Values are Go-native types: string, int64, float64, bool, []any, map[string]any.
type Record struct {
	Values map[string]any
}

// Result represents the result of a Cypher query execution (zero or more rows).
type Result struct {
	Records []Record
}

// Transaction represents an open database transaction.
// All operations are atomic — either all succeed (Commit) or all are rolled back (Rollback).
type Transaction interface {
	// Execute runs a read or write query within the transaction.
	Execute(ctx context.Context, stmt cypher.Statement) (Result, error)

	// Commit commits all operations in the transaction.
	// After Commit, further Execute calls return an error.
	Commit(ctx context.Context) error

	// Rollback aborts the transaction, discarding all operations.
	// Safe to call after Commit (no-op). Safe to call multiple times.
	// Typically called via defer immediately after BeginTx.
	Rollback(ctx context.Context) error
}

// FlattenRows converts []map[string]any rows to a Result with Record entries.
// Shared by Neo4j and FalkorDB driver implementations.
func FlattenRows(rows []map[string]any) Result {
	records := make([]Record, len(rows))
	for i, row := range rows {
		records[i] = Record{Values: row}
	}
	return Result{Records: records}
}

// Driver executes Cypher statements against a graph database.
type Driver interface {
	// Execute runs a read-only query.
	Execute(ctx context.Context, stmt cypher.Statement) (Result, error)

	// ExecuteWrite runs a write query.
	// In clustered Neo4j, this routes to the leader.
	ExecuteWrite(ctx context.Context, stmt cypher.Statement) (Result, error)

	// BeginTx opens an explicit transaction for multi-statement operations.
	// The caller must call Commit or Rollback on the returned Transaction.
	// Used by generated resolvers for nested mutations (create + connect).
	BeginTx(ctx context.Context) (Transaction, error)

	// Close releases all resources (connections, pools).
	// Must be idempotent — safe to call multiple times.
	Close(ctx context.Context) error
}
