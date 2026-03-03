package falkordb

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/driver"
)

// === FK-3: FalkorDB vector query rewrite tests ===

// Test: rewriteVectorQuery rewrites Neo4j vector procedure call to FalkorDB syntax.
// Input: CALL db.index.vector.queryNodes($p0, $p1, $p2) with params {p0: "movie_embeddings", p1: 5, p2: [0.1, 0.2]}
// VectorIndexes: {"movie_embeddings": {Label: "Movie", Property: "embedding"}}
// Expected: rewritten to CALL db.idx.vector.queryNodes($rw0, $rw1, $rw2, $rw3)
//   with params rw0="Movie", rw1="embedding", rw2=5, rw3=[0.1, 0.2]
func TestRewriteVectorQuery_ValidIndex(t *testing.T) {
	indexes := map[string]driver.VectorIndex{
		"movie_embeddings": {Label: "Movie", Property: "embedding"},
	}
	query := "CALL db.index.vector.queryNodes($p0, $p1, $p2) YIELD node AS n, score"
	params := map[string]any{
		"p0": "movie_embeddings",
		"p1": int64(5),
		"p2": []any{0.1, 0.2, 0.3},
	}

	rewritten, newParams := rewriteVectorQuery(query, params, indexes)

	// Query should use db.idx instead of db.index
	if rewritten == query {
		t.Error("query should be rewritten, but was unchanged")
	}
	if len(rewritten) == 0 {
		t.Fatal("rewritten query is empty")
	}

	// Should contain FalkorDB procedure name
	if !strings.Contains(rewritten, "db.idx.vector.queryNodes") {
		t.Errorf("rewritten query should contain 'db.idx.vector.queryNodes', got: %s", rewritten)
	}

	// New params should have 4 entries (label, property, k, vector)
	if len(newParams) < 4 {
		t.Errorf("expected at least 4 params after rewrite, got %d", len(newParams))
	}

	// Verify the param values contain the label and property
	hasLabel := false
	hasProperty := false
	for _, v := range newParams {
		if s, ok := v.(string); ok {
			if s == "Movie" {
				hasLabel = true
			}
			if s == "embedding" {
				hasProperty = true
			}
		}
	}
	if !hasLabel {
		t.Error("rewritten params should contain label 'Movie'")
	}
	if !hasProperty {
		t.Error("rewritten params should contain property 'embedding'")
	}
}

// Test: Non-vector query passes through unchanged.
// Expected: query and params are returned unchanged.
func TestRewriteVectorQuery_NonVectorPassthrough(t *testing.T) {
	indexes := map[string]driver.VectorIndex{
		"movie_embeddings": {Label: "Movie", Property: "embedding"},
	}
	query := "MATCH (n:Movie) WHERE n.title = $title RETURN n"
	params := map[string]any{"title": "Matrix"}

	rewritten, newParams := rewriteVectorQuery(query, params, indexes)

	if rewritten != query {
		t.Errorf("non-vector query should pass through unchanged, got: %s", rewritten)
	}
	if len(newParams) != len(params) {
		t.Errorf("params should be unchanged, got %d (want %d)", len(newParams), len(params))
	}
}

// Test: Nil VectorIndexes causes no rewrite.
// Expected: query and params are returned unchanged.
func TestRewriteVectorQuery_NilVectorIndexes(t *testing.T) {
	query := "CALL db.index.vector.queryNodes($p0, $p1, $p2) YIELD node AS n, score"
	params := map[string]any{
		"p0": "movie_embeddings",
		"p1": int64(5),
		"p2": []any{0.1, 0.2},
	}

	rewritten, newParams := rewriteVectorQuery(query, params, nil)

	if rewritten != query {
		t.Errorf("query with nil VectorIndexes should pass through unchanged, got: %s", rewritten)
	}
	if len(newParams) != len(params) {
		t.Errorf("params should be unchanged")
	}
}

// Test: Unknown index name causes no rewrite (let FalkorDB error).
// Expected: query and params are returned unchanged.
func TestRewriteVectorQuery_UnknownIndex(t *testing.T) {
	indexes := map[string]driver.VectorIndex{
		"article_vectors": {Label: "Article", Property: "vector"},
	}
	query := "CALL db.index.vector.queryNodes($p0, $p1, $p2) YIELD node AS n, score"
	params := map[string]any{
		"p0": "movie_embeddings", // not in indexes
		"p1": int64(5),
		"p2": []any{0.1, 0.2},
	}

	rewritten, newParams := rewriteVectorQuery(query, params, indexes)

	if rewritten != query {
		t.Errorf("query with unknown index should pass through unchanged, got: %s", rewritten)
	}
	if len(newParams) != len(params) {
		t.Errorf("params should be unchanged")
	}
}

