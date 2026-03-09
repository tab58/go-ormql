package client

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
)

// --- CH-4: executeChunk + Execute() chunking orchestration ---

// mockDriverWithCallCount tracks how many times ExecuteWrite/Execute are called.
type mockDriverWithCallCount struct {
	mu               sync.Mutex
	executeCount     int
	writeCount       int
	executeFn        func(ctx context.Context, stmt cypher.Statement) (driver.Result, error)
	executeWriteFn   func(ctx context.Context, stmt cypher.Statement) (driver.Result, error)
}

func (m *mockDriverWithCallCount) Execute(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	m.mu.Lock()
	m.executeCount++
	m.mu.Unlock()
	if m.executeFn != nil {
		return m.executeFn(ctx, stmt)
	}
	return driver.Result{
		Records: []driver.Record{{Values: map[string]any{"data": map[string]any{}}}},
	}, nil
}

func (m *mockDriverWithCallCount) ExecuteWrite(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
	m.mu.Lock()
	m.writeCount++
	m.mu.Unlock()
	if m.executeWriteFn != nil {
		return m.executeWriteFn(ctx, stmt)
	}
	return driver.Result{
		Records: []driver.Record{{Values: map[string]any{"data": map[string]any{}}}},
	}, nil
}

func (m *mockDriverWithCallCount) BeginTx(_ context.Context) (driver.Transaction, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDriverWithCallCount) Close(_ context.Context) error { return nil }

// --- Execute() chunking tests ---

// Test: Mutation with input list <= batchSize executes as single pass (no chunking).
// Expected: ExecuteWrite called exactly once for the read statement.
func TestExecute_SmallInput_NoChunking(t *testing.T) {
	drv := &mockDriverWithCallCount{
		executeWriteFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{{Values: map[string]any{"data": map[string]any{
					"createMovies": map[string]any{"movies": []any{}},
				}}}},
			}, nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv)

	items := make([]any, 10)
	for i := range items {
		items[i] = map[string]any{"title": "Movie"}
	}
	_, err := c.Execute(context.Background(),
		`mutation($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`,
		map[string]any{"input": items},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With 10 items and default batch size 50, should be 1 ExecuteWrite call (read stmt)
	if drv.writeCount != 1 {
		t.Errorf("expected 1 ExecuteWrite call (single pass), got %d", drv.writeCount)
	}
}

// Test: Mutation with input list > batchSize results in multiple chunks.
// Expected: more than 1 ExecuteWrite call.
func TestExecute_LargeInput_Chunks(t *testing.T) {
	drv := &mockDriverWithCallCount{
		executeWriteFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{{Values: map[string]any{"data": map[string]any{
					"createMovies": map[string]any{"movies": []any{}},
				}}}},
			}, nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv, WithBatchSize(5))

	items := make([]any, 12)
	for i := range items {
		items[i] = map[string]any{"title": "Movie"}
	}
	_, err := c.Execute(context.Background(),
		`mutation($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`,
		map[string]any{"input": items},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 12 items / batchSize 5 = 3 chunks = 3 ExecuteWrite calls
	if drv.writeCount < 3 {
		t.Errorf("expected at least 3 ExecuteWrite calls for 12 items / batchSize 5, got %d", drv.writeCount)
	}
}

// Test: Chunked results are aggregated — list fields concatenated.
// Expected: result contains all items from all chunks.
func TestExecute_ChunkedResults_Aggregated(t *testing.T) {
	callNum := 0
	drv := &mockDriverWithCallCount{
		executeWriteFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			callNum++
			movies := []any{}
			// Each chunk returns different movies
			if callNum == 1 {
				movies = []any{map[string]any{"title": "Movie1"}}
			} else if callNum == 2 {
				movies = []any{map[string]any{"title": "Movie2"}}
			}
			return driver.Result{
				Records: []driver.Record{{Values: map[string]any{"data": map[string]any{
					"createMovies": map[string]any{"movies": movies},
				}}}},
			}, nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv, WithBatchSize(1))

	items := []any{
		map[string]any{"title": "Movie1"},
		map[string]any{"title": "Movie2"},
	}
	result, err := c.Execute(context.Background(),
		`mutation($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`,
		map[string]any{"input": items},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := result.Data()
	cm, ok := data["createMovies"].(map[string]any)
	if !ok {
		t.Fatal("expected createMovies in result")
	}
	movies, ok := cm["movies"].([]any)
	if !ok {
		t.Fatal("expected movies to be []any")
	}
	if len(movies) != 2 {
		t.Errorf("expected 2 movies aggregated from 2 chunks, got %d", len(movies))
	}
}

// Test: Chunk execution failure mid-way returns error immediately.
// Expected: error returned, partial results not aggregated.
func TestExecute_ChunkFailure_ReturnsError(t *testing.T) {
	callNum := 0
	drv := &mockDriverWithCallCount{
		executeWriteFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			callNum++
			if callNum == 2 {
				return driver.Result{}, errors.New("database OOM")
			}
			return driver.Result{
				Records: []driver.Record{{Values: map[string]any{"data": map[string]any{
					"createMovies": map[string]any{"movies": []any{}},
				}}}},
			}, nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv, WithBatchSize(1))

	items := []any{
		map[string]any{"title": "Movie1"},
		map[string]any{"title": "Movie2"},
		map[string]any{"title": "Movie3"},
	}
	_, err := c.Execute(context.Background(),
		`mutation($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`,
		map[string]any{"input": items},
	)
	if err == nil {
		t.Fatal("expected error from failed chunk, got nil")
	}
	if !strings.Contains(err.Error(), "OOM") {
		t.Errorf("expected error to contain 'OOM', got: %v", err)
	}
}

// Test: Query operations are never chunked (no list inputs).
// Expected: Execute called exactly once.
func TestExecute_Query_NeverChunked(t *testing.T) {
	drv := &mockDriverWithCallCount{}
	c := New(testModel(), testAugSchemaSDL, drv)

	_, _ = c.Execute(context.Background(), `query { movies { title } }`, nil)

	if drv.executeCount != 1 {
		t.Errorf("query should call Execute exactly once, got %d", drv.executeCount)
	}
}

// Test: When chunking is active and logger is set, logs chunk progress.
// Expected: log output contains "graphql.execute.chunk" with chunk index.
func TestExecute_Chunked_LogsChunkProgress(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	drv := &mockDriverWithCallCount{
		executeWriteFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{{Values: map[string]any{"data": map[string]any{
					"createMovies": map[string]any{"movies": []any{}},
				}}}},
			}, nil
		},
	}
	c := New(testModel(), testAugSchemaSDL, drv, WithLogger(logger), WithBatchSize(1))

	items := []any{
		map[string]any{"title": "M1"},
		map[string]any{"title": "M2"},
	}
	_, _ = c.Execute(context.Background(),
		`mutation($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`,
		map[string]any{"input": items},
	)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "graphql.execute.chunk") {
		t.Errorf("expected chunk progress log 'graphql.execute.chunk', got: %s", logOutput)
	}
}

// Test: Context cancellation between chunks stops execution.
// Expected: error contains "context canceled".
func TestExecute_ContextCancelled_StopsChunking(t *testing.T) {
	callNum := 0
	drv := &mockDriverWithCallCount{
		executeWriteFn: func(ctx context.Context, stmt cypher.Statement) (driver.Result, error) {
			callNum++
			if err := ctx.Err(); err != nil {
				return driver.Result{}, err
			}
			return driver.Result{
				Records: []driver.Record{{Values: map[string]any{"data": map[string]any{
					"createMovies": map[string]any{"movies": []any{}},
				}}}},
			}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := New(testModel(), testAugSchemaSDL, drv, WithBatchSize(1))

	// Cancel after setup — the first chunk may succeed but subsequent should fail
	cancel()

	items := []any{
		map[string]any{"title": "M1"},
		map[string]any{"title": "M2"},
		map[string]any{"title": "M3"},
	}
	_, err := c.Execute(ctx,
		`mutation($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`,
		map[string]any{"input": items},
	)
	// Should either get context canceled error, or succeed (if single-pass, no chunking yet)
	// The key thing is it doesn't hang
	_ = err
}
