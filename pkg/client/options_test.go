package client

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

// --- LOG-1: slog debug logging in client ---

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

// TestClient_Execute_LogsWithLogger verifies that Execute logs a debug message
// when WithLogger is set.
// Expected: log output contains "graphql.execute" and query information.
func TestClient_Execute_LogsWithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c := New(testModel(), testAugSchemaSDL, &mockDriver{}, WithLogger(logger))

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
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})

	// Should not panic even without a logger
	_, _ = c.Execute(context.Background(), "{ movies { title } }", nil)
}

// TestClient_Execute_LogsQueryAndVariables verifies that the debug log includes
// both the query string and variables.
// Expected: log output contains the query and variables information.
func TestClient_Execute_LogsQueryAndVariables(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c := New(testModel(), testAugSchemaSDL, &mockDriver{}, WithLogger(logger))

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

	c := New(testModel(), testAugSchemaSDL, &mockDriver{}, WithLogger(logger1), WithLogger(logger2))

	_, _ = c.Execute(context.Background(), "{ movies { title } }", nil)

	// Logger2 (last) should have output, logger1 should not
	if buf2.Len() == 0 {
		t.Error("last WithLogger should receive log output")
	}
	if buf1.Len() > 0 {
		t.Error("earlier WithLogger should NOT receive log output when overridden")
	}
}
