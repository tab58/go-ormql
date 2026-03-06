package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// === FE-3: translateMutation write/read separation ===
//
// translateMutation must be updated to return ([]string, string, error) where:
// - []string are write queries (FOREACH writes from merge fields)
// - string is the read query (CALL blocks + RETURN map)
// Non-merge fields (create, update, delete, connect) remain as CALL blocks in the read query only.
// Merge fields contribute both a write query AND a read CALL block.

// Test: Mutation with only create has empty write queries.
// Expected: writeQueries is empty, readQuery contains CREATE CALL block.
func TestMutationSplit_CreateOnly_EmptyWrites(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title":    strVal("New Movie"),
			"released": intVal("2024"),
		}),
	)

	op := makeMutationOp(
		makeField("createMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", inputVal)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 0 {
		t.Errorf("expected empty writes for create-only mutation, got %d", len(writes))
	}
	if read == "" {
		t.Fatal("expected non-empty read query for create mutation")
	}
	if !strings.Contains(read, "CALL") {
		t.Errorf("read should contain CALL block, got %q", read)
	}
	if !strings.Contains(read, "AS data") {
		t.Errorf("read should contain 'AS data' RETURN, got %q", read)
	}
}

// Test: Mutation with only merge has one write query and a read query.
// Expected: writeQueries has 1 entry with FOREACH, readQuery has MATCH CALL block.
// FAILS RED: translateMutationSplit does not exist yet (stub returns empty).
func TestMutationSplit_MergeOnly_OneWriteOneRead(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", inputVal)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 1 {
		t.Errorf("expected 1 write query for merge-only mutation, got %d", len(writes))
	}
	if len(writes) > 0 && !strings.Contains(writes[0], "FOREACH") {
		t.Errorf("write[0] should contain FOREACH, got %q", writes[0])
	}
	if read == "" {
		t.Fatal("expected non-empty read query for merge mutation")
	}
	if !strings.Contains(read, "MATCH") {
		t.Errorf("read should contain MATCH for merge projection, got %q", read)
	}
	if strings.Contains(read, "MERGE") {
		t.Errorf("read should NOT contain MERGE (writes are separate), got %q", read)
	}
}

// Test: Mixed create+merge mutation has one write query and a read with both CALL blocks.
// Expected: writeQueries has 1 (from merge), readQuery has CREATE CALL + MATCH CALL.
// FAILS RED: translateMutationSplit does not exist yet (stub returns empty).
func TestMutationSplit_CreatePlusMerge_OneWriteMixedRead(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	createInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title":    strVal("New Movie"),
			"released": intVal("2024"),
		}),
	)
	mergeInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("createMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", createInput)),
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", mergeInput)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 1 {
		t.Errorf("expected 1 write query (from merge), got %d", len(writes))
	}
	if read == "" {
		t.Fatal("expected non-empty read query for mixed mutation")
	}
	// Read should have both a CREATE CALL and a MATCH CALL for merge projection
	if !strings.Contains(read, "CREATE") {
		t.Errorf("read should contain CREATE CALL block, got %q", read)
	}
	if !strings.Contains(read, "MATCH") {
		t.Errorf("read should contain MATCH CALL block for merge read, got %q", read)
	}
	if !strings.Contains(read, "AS data") {
		t.Errorf("read should contain 'AS data' RETURN, got %q", read)
	}
}

// Test: Mutation with delete only has empty write queries.
// Expected: writeQueries is empty, readQuery contains DELETE CALL block.
func TestMutationSplit_DeleteOnly_EmptyWrites(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereVal := makeWhereValue(map[string]*ast.Value{
		"title": strVal("Old Movie"),
	})

	op := makeMutationOp(
		makeField("deleteMovies", ast.SelectionSet{
			&ast.Field{Name: "nodesDeleted"},
		}, makeArg("where", whereVal)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 0 {
		t.Errorf("expected empty writes for delete-only mutation, got %d", len(writes))
	}
	if read == "" {
		t.Fatal("expected non-empty read query for delete mutation")
	}
}

// Test: Empty mutation (no fields) returns empty writes and minimal read.
// Expected: writeQueries empty, readQuery is "RETURN {} AS data".
func TestMutationSplit_Empty_ReturnsMinimalRead(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	op := makeMutationOp() // no fields

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 0 {
		t.Errorf("expected empty writes for empty mutation, got %d", len(writes))
	}
	if read != "RETURN {} AS data" {
		t.Errorf("expected 'RETURN {} AS data' for empty mutation, got %q", read)
	}
}
