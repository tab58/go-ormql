package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// === FE-2: translateMergeField FOREACH rewrite ===
//
// translateMergeField must be rewritten to return two outputs:
// (1) A FOREACH write statement: FOREACH (item IN $input | MERGE ... ON CREATE SET ... ON MATCH SET ...)
// (2) A MATCH read CALL block: CALL { UNWIND $input AS item MATCH (n:Label {matchKeys}) RETURN collect(projection) AS __alias }
//
// The write uses O(1) memory per item. The read fetches merged nodes by match keys.
// Both share the same $input parameter.

// Test: FOREACH write statement contains FOREACH keyword (not UNWIND+MERGE in CALL).
// Expected: write query contains "FOREACH" and does NOT contain "CALL".
// FAILS RED: current translateMergeField returns a single CALL block with UNWIND+MERGE.
func TestMergeForeach_WriteContainsForeach(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	write, _, _, err := tr.translateMergeFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(write, "FOREACH") {
		t.Errorf("write should contain FOREACH, got %q", write)
	}
	if strings.Contains(write, "CALL") {
		t.Errorf("write should NOT contain CALL (that's the read), got %q", write)
	}
}

// Test: FOREACH write contains MERGE with match keys from user input.
// Expected: write contains "MERGE (n:Movie {title: item.match.title})".
// FAILS RED: translateMergeFieldSplit does not exist yet.
func TestMergeForeach_WriteContainsMergeWithMatchKeys(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	write, _, _, err := tr.translateMergeFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(write, "MERGE") {
		t.Errorf("write should contain MERGE, got %q", write)
	}
	if !strings.Contains(write, "title: item.match.title") {
		t.Errorf("write should contain match key 'title: item.match.title', got %q", write)
	}
}

// Test: FOREACH write contains ON CREATE SET with randomUUID() for id.
// Expected: write contains "ON CREATE SET" and "randomUUID()".
// FAILS RED: translateMergeFieldSplit does not exist yet.
func TestMergeForeach_WriteContainsOnCreateSet(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("New Movie"),
			}),
			"onCreate": makeWhereValue(map[string]*ast.Value{
				"released": intVal("2024"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	write, _, _, err := tr.translateMergeFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(write, "ON CREATE SET") {
		t.Errorf("write should contain ON CREATE SET, got %q", write)
	}
	if !strings.Contains(write, "randomUUID()") {
		t.Errorf("write should contain randomUUID(), got %q", write)
	}
}

// Test: FOREACH write contains ON MATCH SET with COALESCE.
// Expected: write contains "ON MATCH SET" and "COALESCE".
// FAILS RED: translateMergeFieldSplit does not exist yet.
func TestMergeForeach_WriteContainsOnMatchSet(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"onMatch": makeWhereValue(map[string]*ast.Value{
				"released": intVal("1999"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	write, _, _, err := tr.translateMergeFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(write, "ON MATCH SET") {
		t.Errorf("write should contain ON MATCH SET, got %q", write)
	}
	if !strings.Contains(write, "COALESCE") {
		t.Errorf("write should contain COALESCE, got %q", write)
	}
}

// Test: Read CALL block uses MATCH (not MERGE) to fetch merged nodes by match keys.
// Expected: read contains "MATCH (n:Movie" and uses match keys in WHERE.
// FAILS RED: translateMergeFieldSplit does not exist yet.
func TestMergeForeach_ReadContainsMatchByMatchKeys(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	_, read, _, err := tr.translateMergeFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(read, "CALL") {
		t.Errorf("read should contain CALL, got %q", read)
	}
	if !strings.Contains(read, "MATCH") {
		t.Errorf("read should contain MATCH, got %q", read)
	}
	if strings.Contains(read, "MERGE") {
		t.Errorf("read should NOT contain MERGE, got %q", read)
	}
	if !strings.Contains(read, "Movie") {
		t.Errorf("read should contain Movie label, got %q", read)
	}
}

// Test: Read CALL block contains collect() for batched result aggregation.
// Expected: read contains "collect(" for projection.
// FAILS RED: translateMergeFieldSplit does not exist yet.
func TestMergeForeach_ReadContainsCollect(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	_, read, _, err := tr.translateMergeFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(read, "collect(") {
		t.Errorf("read should contain collect() for projection, got %q", read)
	}
}

// Test: Both write and read share the same $input parameter.
// Expected: both write and read reference the same parameter placeholder.
// FAILS RED: translateMergeFieldSplit does not exist yet.
func TestMergeForeach_WriteAndReadShareInputParam(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	write, read, _, err := tr.translateMergeFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both should reference $p0 (the input parameter)
	if !strings.Contains(write, "$p0") {
		t.Errorf("write should reference $p0 input param, got %q", write)
	}
	if !strings.Contains(read, "$p0") {
		t.Errorf("read should reference $p0 input param, got %q", read)
	}
}
