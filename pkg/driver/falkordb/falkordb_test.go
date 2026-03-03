package falkordb

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
)

// --- Mock types ---

// mockResultIterator implements resultIterator for unit testing.
type mockResultIterator struct {
	records []map[string]any
	index   int
}

func (m *mockResultIterator) Next() bool {
	if m.index < len(m.records) {
		m.index++
		return true
	}
	return false
}

func (m *mockResultIterator) Record() map[string]any {
	return m.records[m.index-1]
}

// mockGraphRunner implements graphRunner for unit testing.
type mockGraphRunner struct {
	queryFn   func(query string, params map[string]any) (resultIterator, error)
	roQueryFn func(query string, params map[string]any) (resultIterator, error)
}

func (m *mockGraphRunner) Query(query string, params map[string]any) (resultIterator, error) {
	if m.queryFn != nil {
		return m.queryFn(query, params)
	}
	return &mockResultIterator{}, nil
}

func (m *mockGraphRunner) ROQuery(query string, params map[string]any) (resultIterator, error) {
	if m.roQueryFn != nil {
		return m.roQueryFn(query, params)
	}
	return &mockResultIterator{}, nil
}

// mockFalkorDB implements falkordbDB for unit testing.
type mockFalkorDB struct {
	graph   *mockGraphRunner
	closeFn func() error
}

func (m *mockFalkorDB) SelectGraph(_ string) graphRunner {
	return m.graph
}

func (m *mockFalkorDB) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

// === FK-1: FalkorDB driver core tests ===

// --- Interface satisfaction ---

// Test: FalkorDBDriver implements driver.Driver interface.
// Expected: compile-time assertion succeeds.
func TestFalkorDBDriverSatisfiesInterface(t *testing.T) {
	var _ driver.Driver = &FalkorDBDriver{}
}

// --- Execute tests ---

// Test: Execute calls ROQuery and returns flattened records.
// Expected: driver.Result with records matching the mock ROQuery response.
func TestExecute_ReturnsResult(t *testing.T) {
	graph := &mockGraphRunner{
		roQueryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{
				records: []map[string]any{
					{"title": "Matrix", "released": int64(1999)},
				},
			}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	stmt := cypher.Statement{
		Query:  "MATCH (n:Movie) RETURN n",
		Params: map[string]any{},
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

// Test: Execute propagates errors from ROQuery.
// Expected: non-nil error from Execute.
func TestExecute_PropagatesError(t *testing.T) {
	graph := &mockGraphRunner{
		roQueryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return nil, errors.New("connection lost")
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := drv.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute should propagate error from ROQuery")
	}
}

// Test: Execute after Close returns a clear error.
// Expected: non-nil error mentioning "closed".
func TestExecute_AfterClose(t *testing.T) {
	graph := &mockGraphRunner{}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	drv.Close(context.Background())

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	_, err := drv.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("Execute after Close should return error")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("error should mention 'closed', got: %v", err)
	}
}

// --- ExecuteWrite tests ---

// Test: ExecuteWrite calls Query (read-write) and returns flattened records.
// Expected: driver.Result with records matching the mock Query response.
func TestExecuteWrite_ReturnsResult(t *testing.T) {
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{
				records: []map[string]any{
					{"title": "New Movie"},
				},
			}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

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

// Test: ExecuteWrite propagates errors from Query.
// Expected: non-nil error from ExecuteWrite.
func TestExecuteWrite_PropagatesError(t *testing.T) {
	graph := &mockGraphRunner{
		queryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return nil, errors.New("write failed")
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title})"}
	_, err := drv.ExecuteWrite(context.Background(), stmt)
	if err == nil {
		t.Fatal("ExecuteWrite should propagate error from Query")
	}
}

// Test: ExecuteWrite after Close returns a clear error.
// Expected: non-nil error.
func TestExecuteWrite_AfterClose(t *testing.T) {
	graph := &mockGraphRunner{}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

	drv.Close(context.Background())

	stmt := cypher.Statement{Query: "CREATE (n:Movie {title: $title})"}
	_, err := drv.ExecuteWrite(context.Background(), stmt)
	if err == nil {
		t.Fatal("ExecuteWrite after Close should return error")
	}
}

// --- Close tests ---

// Test: Close succeeds and calls underlying db.Close.
// Expected: nil error, db.Close called.
func TestClose_Succeeds(t *testing.T) {
	closed := false
	db := &mockFalkorDB{
		graph:   &mockGraphRunner{},
		closeFn: func() error { closed = true; return nil },
	}
	drv := newFromGraph(db, &mockGraphRunner{}, "test")

	err := drv.Close(context.Background())
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
	if !closed {
		t.Error("Close should call underlying db.Close()")
	}
}

// Test: Close is idempotent — second call does not error or panic.
// Expected: both calls return nil error.
func TestClose_Idempotent(t *testing.T) {
	db := &mockFalkorDB{graph: &mockGraphRunner{}}
	drv := newFromGraph(db, &mockGraphRunner{}, "test")

	if err := drv.Close(context.Background()); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	if err := drv.Close(context.Background()); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

// Test: Close propagates errors from underlying db.Close.
// Expected: non-nil error.
func TestClose_PropagatesError(t *testing.T) {
	db := &mockFalkorDB{
		graph:   &mockGraphRunner{},
		closeFn: func() error { return errors.New("close failed") },
	}
	drv := newFromGraph(db, &mockGraphRunner{}, "test")

	err := drv.Close(context.Background())
	if err == nil {
		t.Fatal("Close should propagate error from db.Close")
	}
}

// --- Scheme validation tests ---

// Test: NewFalkorDBDriver validates scheme is "redis" or "rediss".
// Expected: valid schemes succeed, invalid schemes return error.
func TestNewFalkorDBDriver_SchemeValidation(t *testing.T) {
	tests := []struct {
		scheme  string
		wantErr bool
	}{
		{"redis", false},
		{"rediss", false},
		{"bolt", true},
		{"neo4j", true},
		{"http", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			cfg := driver.Config{
				Host:     "localhost",
				Port:     6379,
				Scheme:   tt.scheme,
				Database: "testgraph",
			}
			err := validateFalkorDBConfig(cfg)
			if tt.wantErr && err == nil {
				t.Errorf("validateFalkorDBConfig(%q) should return error", tt.scheme)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateFalkorDBConfig(%q) returned unexpected error: %v", tt.scheme, err)
			}
		})
	}
}

// Test: NewFalkorDBDriver requires non-empty Database (graph name).
// Expected: error mentioning "database" or "graph".
func TestNewFalkorDBDriver_EmptyDatabase(t *testing.T) {
	cfg := driver.Config{
		Host:     "localhost",
		Port:     6379,
		Scheme:   "redis",
		Database: "",
	}
	err := validateFalkorDBConfig(cfg)
	if err == nil {
		t.Fatal("validateFalkorDBConfig with empty Database should return error")
	}
	errLower := strings.ToLower(err.Error())
	if !strings.Contains(errLower, "database") && !strings.Contains(errLower, "graph") {
		t.Errorf("error should mention 'database' or 'graph', got: %v", err)
	}
}

// --- Debug logging tests ---

// Test: Execute logs debug message when logger is set.
// Expected: Execute succeeds, no panic with logger.
func TestExecute_WithLogger(t *testing.T) {
	graph := &mockGraphRunner{
		roQueryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{
				records: []map[string]any{{"title": "Matrix"}},
			}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	logger := slog.Default()
	drv := newFromGraphWithLogger(db, graph, "test", logger, nil)

	stmt := cypher.Statement{Query: "MATCH (n:Movie) RETURN n"}
	result, err := drv.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("Execute with logger returned error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(result.Records))
	}
}

// --- Concurrency tests ---

// Test: Concurrent Close and Execute do not race.
// Expected: no data race detected by -race flag.
func TestConcurrentCloseAndExecute(t *testing.T) {
	graph := &mockGraphRunner{
		roQueryFn: func(_ string, _ map[string]any) (resultIterator, error) {
			return &mockResultIterator{records: []map[string]any{{"n": 1}}}, nil
		},
	}
	db := &mockFalkorDB{graph: graph}
	drv := newFromGraph(db, graph, "test")

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
}

// --- Result flattening ---

// Test: collectFalkorDBRecords flattens iterator results into []map[string]any.
// Expected: empty iterator returns empty slice, iterator with records returns matching slice.
func TestCollectFalkorDBRecords(t *testing.T) {
	tests := []struct {
		name    string
		records []map[string]any
		wantLen int
	}{
		{"empty", nil, 0},
		{"single", []map[string]any{{"title": "Matrix"}}, 1},
		{"multiple", []map[string]any{{"a": 1}, {"b": 2}, {"c": 3}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := &mockResultIterator{records: tt.records}
			rows := collectFalkorDBRecords(iter)
			if len(rows) != tt.wantLen {
				t.Errorf("len(rows) = %d, want %d", len(rows), tt.wantLen)
			}
		})
	}
}
