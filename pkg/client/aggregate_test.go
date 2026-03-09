package client

import (
	"reflect"
	"testing"
)

// --- CH-3: aggregateResults function ---

// Test: Single result — returns it unchanged.
// Expected: same map returned.
func TestAggregateResults_SingleResult_ReturnsUnchanged(t *testing.T) {
	data := map[string]any{"movies": []any{"a", "b"}}
	result := aggregateResults([]map[string]any{data})
	if !reflect.DeepEqual(result, data) {
		t.Errorf("single result should be returned unchanged")
	}
}

// Test: Empty results slice — returns nil.
// Expected: nil.
func TestAggregateResults_EmptySlice_ReturnsNil(t *testing.T) {
	result := aggregateResults(nil)
	if result != nil {
		t.Errorf("empty results should return nil, got %v", result)
	}
}

// Test: []any values are concatenated across chunks (preserving order).
// Expected: [a, b] + [c, d] = [a, b, c, d].
func TestAggregateResults_ConcatenatesLists(t *testing.T) {
	results := []map[string]any{
		{"createMovies": map[string]any{"movies": []any{"a", "b"}}},
		{"createMovies": map[string]any{"movies": []any{"c", "d"}}},
	}
	agg := aggregateResults(results)
	cm, ok := agg["createMovies"].(map[string]any)
	if !ok {
		t.Fatal("expected createMovies to be map[string]any")
	}
	movies, ok := cm["movies"].([]any)
	if !ok {
		t.Fatal("expected movies to be []any")
	}
	expected := []any{"a", "b", "c", "d"}
	if !reflect.DeepEqual(movies, expected) {
		t.Errorf("expected %v, got %v", expected, movies)
	}
}

// Test: int64 values are summed across chunks (count aggregation).
// Expected: 5 + 3 = 8.
func TestAggregateResults_SumsInt64(t *testing.T) {
	results := []map[string]any{
		{"connectInfo": map[string]any{"relationshipsCreated": int64(5)}},
		{"connectInfo": map[string]any{"relationshipsCreated": int64(3)}},
	}
	agg := aggregateResults(results)
	ci, ok := agg["connectInfo"].(map[string]any)
	if !ok {
		t.Fatal("expected connectInfo to be map[string]any")
	}
	count, ok := ci["relationshipsCreated"].(int64)
	if !ok {
		t.Fatalf("expected int64 for relationshipsCreated, got %T", ci["relationshipsCreated"])
	}
	if count != 8 {
		t.Errorf("expected sum 8, got %d", count)
	}
}

// Test: float64 values are summed across chunks.
// Expected: 2.5 + 3.5 = 6.0.
func TestAggregateResults_SumsFloat64(t *testing.T) {
	results := []map[string]any{
		{"score": float64(2.5)},
		{"score": float64(3.5)},
	}
	agg := aggregateResults(results)
	score, ok := agg["score"].(float64)
	if !ok {
		t.Fatalf("expected float64 for score, got %T", agg["score"])
	}
	if score != 6.0 {
		t.Errorf("expected sum 6.0, got %f", score)
	}
}

// Test: map[string]any values are recursed.
// Expected: nested list fields concatenated, nested counts summed.
func TestAggregateResults_RecursesIntoMaps(t *testing.T) {
	results := []map[string]any{
		{"createMovies": map[string]any{"movies": []any{"a"}, "count": int64(1)}},
		{"createMovies": map[string]any{"movies": []any{"b"}, "count": int64(2)}},
	}
	agg := aggregateResults(results)
	cm, ok := agg["createMovies"].(map[string]any)
	if !ok {
		t.Fatal("expected createMovies to be map[string]any")
	}
	movies, ok := cm["movies"].([]any)
	if !ok {
		t.Fatal("expected movies to be []any")
	}
	if !reflect.DeepEqual(movies, []any{"a", "b"}) {
		t.Errorf("expected [a, b], got %v", movies)
	}
	count, ok := cm["count"].(int64)
	if !ok {
		t.Fatalf("expected int64 for count, got %T", cm["count"])
	}
	if count != 3 {
		t.Errorf("expected sum 3, got %d", count)
	}
}

// Test: Other value types use last chunk's value.
// Expected: string from last chunk.
func TestAggregateResults_OtherTypes_LastWins(t *testing.T) {
	results := []map[string]any{
		{"status": "first"},
		{"status": "second"},
	}
	agg := aggregateResults(results)
	if agg["status"] != "second" {
		t.Errorf("expected last chunk's value 'second', got %v", agg["status"])
	}
}

// Test: Mixed fields — list + count + scalar in same result.
// Expected: list concatenated, count summed, scalar last-wins.
func TestAggregateResults_MixedFields(t *testing.T) {
	results := []map[string]any{
		{"movies": []any{"a"}, "total": int64(10), "status": "ok"},
		{"movies": []any{"b"}, "total": int64(20), "status": "done"},
	}
	agg := aggregateResults(results)

	movies, ok := agg["movies"].([]any)
	if !ok {
		t.Fatal("expected movies to be []any")
	}
	if !reflect.DeepEqual(movies, []any{"a", "b"}) {
		t.Errorf("expected [a, b], got %v", movies)
	}

	total, ok := agg["total"].(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", agg["total"])
	}
	if total != 30 {
		t.Errorf("expected sum 30, got %d", total)
	}

	if agg["status"] != "done" {
		t.Errorf("expected last 'done', got %v", agg["status"])
	}
}

// Test: Three chunks — verifies aggregation works for > 2 chunks.
// Expected: all three lists concatenated.
func TestAggregateResults_ThreeChunks(t *testing.T) {
	results := []map[string]any{
		{"items": []any{1, 2}},
		{"items": []any{3, 4}},
		{"items": []any{5}},
	}
	agg := aggregateResults(results)
	items, ok := agg["items"].([]any)
	if !ok {
		t.Fatal("expected items to be []any")
	}
	if !reflect.DeepEqual(items, []any{1, 2, 3, 4, 5}) {
		t.Errorf("expected [1,2,3,4,5], got %v", items)
	}
}
