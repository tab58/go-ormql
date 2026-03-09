package client

import (
	"reflect"
	"testing"
)

// --- CH-2: chunkParams function ---

// Test: No list params — returns single-element slice with original params.
// Expected: []map[string]any{params}.
func TestChunkParams_NoLists_ReturnsSingleChunk(t *testing.T) {
	params := map[string]any{"where": "x", "limit": 10}
	chunks := chunkParams(params, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !reflect.DeepEqual(chunks[0], params) {
		t.Errorf("single chunk should equal original params")
	}
}

// Test: List param within batchSize — returns single chunk (no splitting).
// Expected: 1 chunk.
func TestChunkParams_ListWithinBatchSize_SingleChunk(t *testing.T) {
	items := make([]any, 50)
	for i := range items {
		items[i] = i
	}
	params := map[string]any{"p0": items}
	chunks := chunkParams(params, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for list of size 50 with batchSize 50, got %d", len(chunks))
	}
}

// Test: List param exceeds batchSize — splits into ceil(len/batchSize) chunks.
// Expected: 100 items / 50 batchSize = 2 chunks.
func TestChunkParams_ListExceedsBatchSize_SplitsIntoChunks(t *testing.T) {
	items := make([]any, 100)
	for i := range items {
		items[i] = i
	}
	params := map[string]any{"p0": items}
	chunks := chunkParams(params, 50)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks for 100 items / batchSize 50, got %d", len(chunks))
	}

	// Chunk 0: items[0:50]
	p0c0, ok := chunks[0]["p0"].([]any)
	if !ok {
		t.Fatal("chunk 0 p0 should be []any")
	}
	if len(p0c0) != 50 {
		t.Errorf("chunk 0 p0 should have 50 items, got %d", len(p0c0))
	}
	if p0c0[0] != 0 || p0c0[49] != 49 {
		t.Errorf("chunk 0 items should be 0..49")
	}

	// Chunk 1: items[50:100]
	p0c1, ok := chunks[1]["p0"].([]any)
	if !ok {
		t.Fatal("chunk 1 p0 should be []any")
	}
	if len(p0c1) != 50 {
		t.Errorf("chunk 1 p0 should have 50 items, got %d", len(p0c1))
	}
	if p0c1[0] != 50 || p0c1[49] != 99 {
		t.Errorf("chunk 1 items should be 50..99")
	}
}

// Test: Non-even split — last chunk has remainder.
// Expected: 75 items / 50 batchSize = 2 chunks (50 + 25).
func TestChunkParams_NonEvenSplit_LastChunkHasRemainder(t *testing.T) {
	items := make([]any, 75)
	for i := range items {
		items[i] = i
	}
	params := map[string]any{"p0": items}
	chunks := chunkParams(params, 50)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks for 75 items / batchSize 50, got %d", len(chunks))
	}

	p0c0, _ := chunks[0]["p0"].([]any)
	p0c1, _ := chunks[1]["p0"].([]any)
	if len(p0c0) != 50 {
		t.Errorf("chunk 0 should have 50 items, got %d", len(p0c0))
	}
	if len(p0c1) != 25 {
		t.Errorf("chunk 1 should have 25 items, got %d", len(p0c1))
	}
}

// Test: Non-list params are copied unchanged to every chunk.
// Expected: scalar params present in all chunks with same value.
func TestChunkParams_NonListParams_CopiedToAllChunks(t *testing.T) {
	items := make([]any, 100)
	for i := range items {
		items[i] = i
	}
	params := map[string]any{"p0": items, "where": "title = $x"}
	chunks := chunkParams(params, 50)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	for i, chunk := range chunks {
		w, ok := chunk["where"]
		if !ok {
			t.Errorf("chunk %d missing 'where' param", i)
		}
		if w != "title = $x" {
			t.Errorf("chunk %d 'where' should be 'title = $x', got %v", i, w)
		}
	}
}

// Test: Multiple list params with different lengths — aligned on longest.
// Shorter lists get empty slices for out-of-bounds chunks.
// Expected: 100 items and 30 items / batchSize 50 = 2 chunks.
// Chunk 0: p0[0:50], p1[0:30]. Chunk 1: p0[50:100], p1=[]any{} (empty).
func TestChunkParams_MultipleListParams_AlignedOnLongest(t *testing.T) {
	long := make([]any, 100)
	for i := range long {
		long[i] = i
	}
	short := make([]any, 30)
	for i := range short {
		short[i] = i + 1000
	}
	params := map[string]any{"p0": long, "p1": short}
	chunks := chunkParams(params, 50)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks aligned on longest list (100), got %d", len(chunks))
	}

	// Chunk 0: p1 has items[0:30]
	p1c0, ok := chunks[0]["p1"].([]any)
	if !ok {
		t.Fatal("chunk 0 p1 should be []any")
	}
	if len(p1c0) != 30 {
		t.Errorf("chunk 0 p1 should have 30 items, got %d", len(p1c0))
	}

	// Chunk 1: p1 should be empty (out of bounds for shorter list)
	p1c1, ok := chunks[1]["p1"].([]any)
	if !ok {
		t.Fatal("chunk 1 p1 should be []any (empty)")
	}
	if len(p1c1) != 0 {
		t.Errorf("chunk 1 p1 should be empty for out-of-bounds, got %d items", len(p1c1))
	}
}

// Test: Empty list param — no chunking needed.
// Expected: 1 chunk with empty list preserved.
func TestChunkParams_EmptyList_SingleChunk(t *testing.T) {
	params := map[string]any{"p0": []any{}}
	chunks := chunkParams(params, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for empty list, got %d", len(chunks))
	}
}

// Test: Single-item list — never chunked (1 <= batchSize).
// Expected: 1 chunk.
func TestChunkParams_SingleItem_SingleChunk(t *testing.T) {
	params := map[string]any{"p0": []any{"item1"}}
	chunks := chunkParams(params, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for single-item list, got %d", len(chunks))
	}
}

// Test: Nil params — returns single chunk with nil.
// Expected: 1 chunk.
func TestChunkParams_NilParams_SingleChunk(t *testing.T) {
	chunks := chunkParams(nil, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for nil params, got %d", len(chunks))
	}
}

// Test: batchSize of 1 — every item is its own chunk.
// Expected: 3 items / batchSize 1 = 3 chunks.
func TestChunkParams_BatchSizeOne_EachItemOwnChunk(t *testing.T) {
	params := map[string]any{"p0": []any{"a", "b", "c"}}
	chunks := chunkParams(params, 1)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks for 3 items with batchSize 1, got %d", len(chunks))
	}
	for i, chunk := range chunks {
		p0, ok := chunk["p0"].([]any)
		if !ok {
			t.Fatalf("chunk %d p0 should be []any", i)
		}
		if len(p0) != 1 {
			t.Errorf("chunk %d should have 1 item, got %d", i, len(p0))
		}
	}
}
