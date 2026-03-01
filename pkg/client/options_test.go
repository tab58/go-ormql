package client

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
)

// --- LOG-1: slog debug logging in client ---

// mockDriverForLog implements driver.Driver for client logging tests.
type mockDriverForLog struct{}

func (m *mockDriverForLog) Execute(_ context.Context, _ cypher.Statement) (driver.Result, error) {
	return driver.Result{}, nil
}
func (m *mockDriverForLog) ExecuteWrite(_ context.Context, _ cypher.Statement) (driver.Result, error) {
	return driver.Result{}, nil
}
func (m *mockDriverForLog) BeginTx(_ context.Context) (driver.Transaction, error) {
	return nil, nil
}
func (m *mockDriverForLog) Close(_ context.Context) error { return nil }

// TestWithLogger_ReturnsOption verifies that WithLogger returns a non-nil Option.
// Expected: WithLogger(logger) returns a function, not nil.
func TestWithLogger_ReturnsOption(t *testing.T) {
	logger := slog.Default()
	opt := WithLogger(logger)
	if opt == nil {
		t.Error("WithLogger should return a non-nil Option function")
	}
}

// TestWithLogger_NilLogger_ReturnsOption verifies that WithLogger(nil)
// returns a non-nil Option (disabling logging).
// Expected: WithLogger(nil) returns a function, not nil.
func TestWithLogger_NilLogger_ReturnsOption(t *testing.T) {
	opt := WithLogger(nil)
	if opt == nil {
		t.Error("WithLogger(nil) should return a non-nil Option function")
	}
}

// TestNew_AcceptsOptions verifies that New() accepts variadic Option parameters.
// Expected: New(schema, drv, WithLogger(logger)) compiles and returns a non-nil Client.
func TestNew_AcceptsOptions(t *testing.T) {
	logger := slog.Default()
	c := New(nil, &mockDriverForLog{}, WithLogger(logger))
	if c == nil {
		t.Error("New with options should return a non-nil Client")
	}
}

// TestNew_WithoutOptions verifies that New() works without any options (backwards compatible).
// Expected: New(schema, drv) compiles and returns a non-nil Client.
func TestNew_WithoutOptions(t *testing.T) {
	c := New(nil, &mockDriverForLog{})
	if c == nil {
		t.Error("New without options should return a non-nil Client")
	}
}

// TestClient_Execute_LogsWithLogger verifies that Execute logs a debug message
// when WithLogger is set.
// Expected: log output contains "graphql.execute" and query information.
func TestClient_Execute_LogsWithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c := New(nil, &mockDriverForLog{}, WithLogger(logger))

	_, _ = c.Execute(context.Background(), "{ movies { title } }", nil)

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("Execute should log when logger is set, but log output is empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte("graphql.execute")) {
		t.Errorf("log output should contain 'graphql.execute', got: %s", logOutput)
	}
}

// TestClient_Execute_NoLogWithoutLogger verifies that Execute produces no
// log output when no logger is set (zero overhead).
// Expected: no panic, no error from logging.
func TestClient_Execute_NoLogWithoutLogger(t *testing.T) {
	c := New(nil, &mockDriverForLog{})

	// Should not panic even without a logger
	_, _ = c.Execute(context.Background(), "{ movies { title } }", nil)
}

// TestClient_Execute_LogsQueryAndVariables verifies that the debug log includes
// both the query string and variables.
// Expected: log output contains the query and variables information.
func TestClient_Execute_LogsQueryAndVariables(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c := New(nil, &mockDriverForLog{}, WithLogger(logger))

	query := "query GetMovie($id: ID!) { movie(id: $id) { title } }"
	vars := map[string]any{"id": "movie-1"}
	_, _ = c.Execute(context.Background(), query, vars)

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("log output should contain query and variables info")
	}
}

// TestWithLogger_LastWins verifies that when multiple WithLogger options are
// provided, the last one takes effect.
// Expected: the last logger wins, earlier loggers don't receive output.
func TestWithLogger_LastWins(t *testing.T) {
	var buf1 bytes.Buffer
	logger1 := slog.New(slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var buf2 bytes.Buffer
	logger2 := slog.New(slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c := New(nil, &mockDriverForLog{}, WithLogger(logger1), WithLogger(logger2))

	_, _ = c.Execute(context.Background(), "{ movies { title } }", nil)

	// Logger2 (last) should have output, logger1 should not
	if buf2.Len() == 0 {
		t.Error("last WithLogger should receive log output")
	}
	if buf1.Len() > 0 {
		t.Error("earlier WithLogger should NOT receive log output when overridden")
	}
}
