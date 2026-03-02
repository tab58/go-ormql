package neo4j

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
)

// errTransactionCommitted is returned when Execute is called on a committed transaction.
var errTransactionCommitted = errors.New("transaction already committed")

// sessionRunner abstracts the neo4j session operations we actually use.
// This narrow interface makes the adapter testable without mocking the full neo4j session.
type sessionRunner interface {
	ExecuteRead(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
	ExecuteWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
	BeginTransaction(ctx context.Context) (transactionRunner, error)
	Close(ctx context.Context) error
}

// transactionRunner abstracts the neo4j explicit transaction operations.
type transactionRunner interface {
	Run(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// neo4jDB abstracts the neo4j driver for testing.
type neo4jDB interface {
	NewSession(database string) sessionRunner
	Close(ctx context.Context) error
}

// Neo4jDriver wraps the official neo4j-go-driver to implement driver.Driver.
type Neo4jDriver struct {
	mu       sync.Mutex
	db       neo4jDB
	database string
	logger   *slog.Logger
	closed   bool
}

// NewNeo4jDriver creates a Neo4j driver instance connected to the given URI.
// Returns a clear error (without credentials) if connection fails.
func NewNeo4jDriver(cfg driver.Config) (driver.Driver, error) {
	drv, err := newRealNeo4jDriver(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Neo4j at %s: %w", cfg.URI, err)
	}
	return drv, nil
}

// newFromDB creates a Neo4jDriver from an already-initialized neo4jDB (for testing).
func newFromDB(db neo4jDB, database string) *Neo4jDriver {
	return &Neo4jDriver{db: db, database: database}
}

// newFromDBWithLogger creates a Neo4jDriver from a neo4jDB with a logger (for testing).
func newFromDBWithLogger(db neo4jDB, database string, logger *slog.Logger) *Neo4jDriver {
	return &Neo4jDriver{db: db, database: database, logger: logger}
}

// checkClosed returns an error if the driver has been closed.
func (d *Neo4jDriver) checkClosed() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return fmt.Errorf("driver is closed")
	}
	return nil
}

// Execute runs a read-only query using a read session.
func (d *Neo4jDriver) Execute(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	if err := d.checkClosed(); err != nil {
		return driver.Result{}, err
	}

	if d.logger != nil {
		d.logger.Debug("cypher.execute", "query", stmt.Query, "params", stmt.Params)
	}

	session := d.db.NewSession(d.database)
	defer session.Close(ctx)

	rows, err := session.ExecuteRead(ctx, stmt.Query, stmt.Params)
	if err != nil {
		return driver.Result{}, fmt.Errorf("execute read: %w", err)
	}

	return flattenRows(rows), nil
}

// ExecuteWrite runs a write query using a write session.
func (d *Neo4jDriver) ExecuteWrite(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	if err := d.checkClosed(); err != nil {
		return driver.Result{}, err
	}

	if d.logger != nil {
		d.logger.Debug("cypher.execute", "query", stmt.Query, "params", stmt.Params)
	}

	session := d.db.NewSession(d.database)
	defer session.Close(ctx)

	rows, err := session.ExecuteWrite(ctx, stmt.Query, stmt.Params)
	if err != nil {
		return driver.Result{}, fmt.Errorf("execute write: %w", err)
	}

	return flattenRows(rows), nil
}

// BeginTx opens an explicit transaction for multi-statement operations.
// Creates a write session and begins an explicit transaction on it.
func (d *Neo4jDriver) BeginTx(ctx context.Context) (driver.Transaction, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}

	session := d.db.NewSession(d.database)
	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		session.Close(ctx)
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	return &neo4jTransaction{tx: tx, session: session, logger: d.logger}, nil
}

// Close releases the Neo4j driver resources. Idempotent.
func (d *Neo4jDriver) Close(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	return d.db.Close(ctx)
}

// neo4jTransaction wraps a transactionRunner to implement driver.Transaction.
type neo4jTransaction struct {
	mu        sync.Mutex
	tx        transactionRunner
	session   sessionRunner
	logger    *slog.Logger
	committed bool
}

// isCommitted reports whether the transaction has been committed.
// Thread-safe: reads the committed flag under the mutex.
func (t *neo4jTransaction) isCommitted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.committed
}

// Execute runs a query within the open transaction.
// Returns an error if the transaction has already been committed.
func (t *neo4jTransaction) Execute(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	if t.isCommitted() {
		return driver.Result{}, errTransactionCommitted
	}

	if t.logger != nil {
		t.logger.Debug("cypher.execute", "query", stmt.Query, "params", stmt.Params)
	}

	rows, err := t.tx.Run(ctx, stmt.Query, stmt.Params)
	if err != nil {
		return driver.Result{}, fmt.Errorf("tx execute: %w", err)
	}
	return flattenRows(rows), nil
}

// Commit commits all operations in the transaction.
func (t *neo4jTransaction) Commit(ctx context.Context) error {
	if t.isCommitted() {
		return nil
	}

	if err := t.tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	t.mu.Lock()
	t.committed = true
	t.mu.Unlock()

	t.session.Close(ctx)
	return nil
}

// Rollback aborts the transaction. Safe to call after Commit (no-op).
func (t *neo4jTransaction) Rollback(ctx context.Context) error {
	if t.isCommitted() {
		return nil
	}

	if err := t.tx.Rollback(ctx); err != nil {
		return fmt.Errorf("tx rollback: %w", err)
	}
	t.session.Close(ctx)
	return nil
}

// flattenRows converts []map[string]any rows to driver.Result with driver.Record entries.
func flattenRows(rows []map[string]any) driver.Result {
	records := make([]driver.Record, len(rows))
	for i, row := range rows {
		records[i] = driver.Record{Values: row}
	}
	return driver.Result{Records: records}
}
