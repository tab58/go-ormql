package client

import (
	"strings"
	"testing"
)

// --- CH-1: WithBatchSize option + defaultBatchSize ---

// Test: defaultBatchSize constant equals 50.
// Expected: defaultBatchSize == 50.
func TestDefaultBatchSize_Is50(t *testing.T) {
	if defaultBatchSize != 50 {
		t.Errorf("defaultBatchSize should be 50, got %d", defaultBatchSize)
	}
}

// Test: WithBatchSize returns a non-nil Option.
// Expected: non-nil function returned.
func TestWithBatchSize_ReturnsOption(t *testing.T) {
	opt := WithBatchSize(100)
	if opt == nil {
		t.Error("WithBatchSize should return a non-nil Option function")
	}
}

// Test: WithBatchSize(0) panics — batch size must be > 0.
// Expected: panic with message about batch size.
func TestWithBatchSize_Zero_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("WithBatchSize(0) should panic, but did not")
		}
		msg := ""
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(strings.ToLower(msg), "batch") {
			t.Errorf("panic message should mention 'batch', got: %q", msg)
		}
	}()
	WithBatchSize(0)
}

// Test: WithBatchSize(-1) panics — batch size must be > 0.
// Expected: panic.
func TestWithBatchSize_Negative_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("WithBatchSize(-1) should panic, but did not")
		}
	}()
	WithBatchSize(-1)
}

// Test: New() without WithBatchSize uses defaultBatchSize.
// Expected: client.batchSize == 50.
func TestNew_DefaultBatchSize(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{})
	if c.batchSize != defaultBatchSize {
		t.Errorf("expected default batchSize %d, got %d", defaultBatchSize, c.batchSize)
	}
}

// Test: New() with WithBatchSize(100) sets batchSize to 100.
// Expected: client.batchSize == 100.
func TestNew_WithBatchSize_SetsValue(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{}, WithBatchSize(100))
	if c.batchSize != 100 {
		t.Errorf("expected batchSize 100, got %d", c.batchSize)
	}
}

// Test: Multiple WithBatchSize calls — last one wins.
// Expected: client.batchSize == 25 (last value).
func TestWithBatchSize_LastWins(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{}, WithBatchSize(100), WithBatchSize(25))
	if c.batchSize != 25 {
		t.Errorf("expected batchSize 25 (last wins), got %d", c.batchSize)
	}
}

// Test: WithBatchSize(1) is valid — minimum allowed value.
// Expected: no panic, client.batchSize == 1.
func TestWithBatchSize_One_IsValid(t *testing.T) {
	c := New(testModel(), testAugSchemaSDL, &mockDriver{}, WithBatchSize(1))
	if c.batchSize != 1 {
		t.Errorf("expected batchSize 1, got %d", c.batchSize)
	}
}
