package falkordb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
)

// errTransactionCommitted is returned when Execute is called on a committed transaction.
var errTransactionCommitted = errors.New("transaction already committed")

// errSingleStatementOnly is returned when a second Execute is called on a FalkorDB transaction.
var errSingleStatementOnly = errors.New("falkordb transactions support only a single statement")

// errDriverClosed is returned when an operation is attempted on a closed driver.
var errDriverClosed = errors.New("driver is closed")

// falkordbDB abstracts the FalkorDB connection for testing.
type falkordbDB interface {
	SelectGraph(name string) graphRunner
	Close() error
}

// graphRunner abstracts FalkorDB graph operations.
type graphRunner interface {
	Query(query string, params map[string]any) (resultIterator, error)
	ROQuery(query string, params map[string]any) (resultIterator, error)
}

// resultIterator abstracts FalkorDB's iterator-based results.
type resultIterator interface {
	Next() bool
	Record() map[string]any
}

// validFalkorDBSchemes lists the schemes accepted by the FalkorDB driver.
var validFalkorDBSchemes = map[string]bool{
	"redis": true, "rediss": true,
}

// FalkorDBDriver wraps a FalkorDB connection to implement driver.Driver.
type FalkorDBDriver struct {
	mu            sync.Mutex
	db            falkordbDB
	graph         graphRunner
	graphName     string
	logger        *slog.Logger
	vectorIndexes map[string]driver.VectorIndex
	closed        bool
}

// NewFalkorDBDriver creates a FalkorDB driver instance.
func NewFalkorDBDriver(cfg driver.Config) (driver.Driver, error) {
	if err := validateFalkorDBConfig(cfg); err != nil {
		return nil, err
	}
	drv, err := newRealFalkorDBDriver(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FalkorDB at %s:%d: %w", cfg.Host, cfg.Port, err)
	}
	return drv, nil
}

// newFromGraph creates a FalkorDBDriver from an already-initialized graphRunner (for testing).
func newFromGraph(db falkordbDB, graph graphRunner, graphName string) *FalkorDBDriver {
	return &FalkorDBDriver{db: db, graph: graph, graphName: graphName}
}

// newFromGraphWithLogger creates a FalkorDBDriver with a logger (for testing).
func newFromGraphWithLogger(db falkordbDB, graph graphRunner, graphName string, logger *slog.Logger, vectorIndexes map[string]driver.VectorIndex) *FalkorDBDriver {
	return &FalkorDBDriver{db: db, graph: graph, graphName: graphName, logger: logger, vectorIndexes: vectorIndexes}
}

// checkClosed returns an error if the driver has been closed.
func (d *FalkorDBDriver) checkClosed() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return errDriverClosed
	}
	return nil
}

// Execute runs a read-only query using ROQuery.
func (d *FalkorDBDriver) Execute(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	return d.executeQuery(stmt, d.graph.ROQuery, "execute read")
}

// ExecuteWrite runs a read-write query using Query.
func (d *FalkorDBDriver) ExecuteWrite(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	return d.executeQuery(stmt, d.graph.Query, "execute write")
}

// executeQuery runs a Cypher statement through the given query function.
// Handles closed-check, vector rewrite, logging, record collection, and error wrapping.
func (d *FalkorDBDriver) executeQuery(stmt cypher.Statement, queryFn func(string, map[string]any) (resultIterator, error), errPrefix string) (driver.Result, error) {
	if err := d.checkClosed(); err != nil {
		return driver.Result{}, err
	}

	query, params := rewriteVectorQuery(stmt.Query, stmt.Params, d.vectorIndexes)

	if d.logger != nil {
		d.logger.Debug("cypher.execute", "query", query, "params", params)
	}

	iter, err := queryFn(query, params)
	if err != nil {
		return driver.Result{}, fmt.Errorf("%s: %w", errPrefix, err)
	}

	rows := collectFalkorDBRecords(iter)
	return driver.FlattenRows(rows), nil
}

// BeginTx opens a single-query buffered transaction.
func (d *FalkorDBDriver) BeginTx(ctx context.Context) (driver.Transaction, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}
	return &falkordbTransaction{graph: d.graph, logger: d.logger}, nil
}

// Close releases the FalkorDB connection. Idempotent.
func (d *FalkorDBDriver) Close(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	return d.db.Close()
}

// falkordbTransaction implements a single-query buffered transaction for FalkorDB.
type falkordbTransaction struct {
	mu        sync.Mutex
	graph     graphRunner
	logger    *slog.Logger
	committed bool
	stmt      *cypher.Statement
	executed  bool
}

// isCommitted reports whether the transaction has been committed.
func (t *falkordbTransaction) isCommitted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.committed
}

// Execute buffers the first statement. Second call returns errSingleStatementOnly.
func (t *falkordbTransaction) Execute(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
	if t.isCommitted() {
		return driver.Result{}, errTransactionCommitted
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.executed {
		return driver.Result{}, errSingleStatementOnly
	}
	t.stmt = &stmt
	t.executed = true
	return driver.Result{}, nil
}

// Commit executes the buffered statement via graph.Query.
func (t *falkordbTransaction) Commit(_ context.Context) error {
	if t.isCommitted() {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stmt != nil {
		if t.logger != nil {
			t.logger.Debug("cypher.execute", "query", t.stmt.Query, "params", t.stmt.Params)
		}
		if _, err := t.graph.Query(t.stmt.Query, t.stmt.Params); err != nil {
			return fmt.Errorf("tx commit: %w", err)
		}
	}

	t.committed = true
	return nil
}

// Rollback discards the buffered statement.
func (t *falkordbTransaction) Rollback(_ context.Context) error {
	if t.isCommitted() {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stmt = nil
	t.committed = true
	return nil
}

// validateFalkorDBConfig validates Config fields for FalkorDB connection.
func validateFalkorDBConfig(cfg driver.Config) error {
	if cfg.Host == "" {
		return fmt.Errorf("falkordb: host is required")
	}
	if !validFalkorDBSchemes[cfg.Scheme] {
		return fmt.Errorf("falkordb: unsupported scheme %q (valid: redis, rediss)", cfg.Scheme)
	}
	if cfg.Database == "" {
		return fmt.Errorf("falkordb: database (graph name) is required")
	}
	return nil
}

// collectFalkorDBRecords collects all records from a resultIterator.
func collectFalkorDBRecords(iter resultIterator) []map[string]any {
	var rows []map[string]any
	for iter.Next() {
		rows = append(rows, iter.Record())
	}
	return rows
}

// neo4jVectorProc is the Neo4j vector procedure prefix to detect.
const neo4jVectorProc = "CALL db.index.vector.queryNodes("

// falkorDBVectorProc is the FalkorDB vector procedure prefix to rewrite to.
const falkorDBVectorProc = "CALL db.idx.vector.queryNodes("

// rewriteVectorQuery rewrites Neo4j vector procedure calls to FalkorDB syntax.
// Neo4j: CALL db.index.vector.queryNodes($indexName, $k, $vector) — 3 params
// FalkorDB: CALL db.idx.vector.queryNodes($label, $property, $k, $vector) — 4 params
// Returns input unchanged for non-vector queries, nil indexes, or unknown index names.
func rewriteVectorQuery(query string, params map[string]any, indexes map[string]driver.VectorIndex) (string, map[string]any) {
	if indexes == nil {
		return query, params
	}

	idx := strings.Index(query, neo4jVectorProc)
	if idx < 0 {
		return query, params
	}

	// Extract the arguments inside the CALL parentheses
	argStart := idx + len(neo4jVectorProc)
	argEnd := strings.Index(query[argStart:], ")")
	if argEnd < 0 {
		return query, params
	}
	argEnd += argStart

	argStr := query[argStart:argEnd]
	args := strings.Split(argStr, ",")
	if len(args) != 3 {
		return query, params
	}

	// Trim whitespace and $ prefix from param names
	indexParam := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args[0]), "$"))
	kParam := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args[1]), "$"))
	vectorParam := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args[2]), "$"))

	// Look up the index name from params
	indexName, ok := params[indexParam].(string)
	if !ok {
		return query, params
	}

	// Look up the index in vectorIndexes
	vecIdx, found := indexes[indexName]
	if !found {
		return query, params
	}

	// Build new params with label, property, k, vector
	newParams := make(map[string]any, len(params)-3+4)
	for k, v := range params {
		if k != indexParam && k != kParam && k != vectorParam {
			newParams[k] = v
		}
	}

	newParams["rw0"] = vecIdx.Label
	newParams["rw1"] = vecIdx.Property
	newParams["rw2"] = params[kParam]
	newParams["rw3"] = toInterfaceSlice(params[vectorParam])

	// Rewrite the query
	newCallArgs := "$rw0, $rw1, $rw2, $rw3"
	rewritten := query[:idx] + falkorDBVectorProc + newCallArgs + query[argEnd:]

	return rewritten, newParams
}

// toInterfaceSlice converts typed slices (e.g. []float64) to []interface{}
// so that falkordb-go's ToString can serialize them.
func toInterfaceSlice(v any) any {
	switch s := v.(type) {
	case []float64:
		out := make([]any, len(s))
		for i, f := range s {
			out[i] = f
		}
		return out
	case []any:
		return v
	default:
		return v
	}
}
