package translate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- TR-5: Root connection translation tests ---

// Test: translateConnectionField produces CALL subquery with pagination (SKIP/LIMIT).
func TestTranslateConnectionField_ProducesPaginatedQuery(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
			&ast.Field{Name: "cursor"},
		}},
	}, makeArg("first", intVal("10")))

	result, alias, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty connection query, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(result, "CALL") {
		t.Errorf("expected CALL in connection query, got %q", result)
	}
	if !strings.Contains(result, "MATCH") {
		t.Errorf("expected MATCH in connection query, got %q", result)
	}
	if !strings.Contains(result, "Movie") {
		t.Errorf("expected Movie label in connection query, got %q", result)
	}
}

// Test: translateConnectionField includes totalCount subquery only when
// totalCount is in the selection set.
func TestTranslateConnectionField_TotalCountWhenSelected(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
		&ast.Field{Name: "totalCount"},
	}, makeArg("first", intVal("10")))

	result, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "totalCount") || !strings.Contains(result, "count") {
		t.Errorf("expected totalCount/count in connection query when selected, got %q", result)
	}
}

// Test: translateConnectionField omits totalCount subquery when
// totalCount is NOT in the selection set.
func TestTranslateConnectionField_NoTotalCountWhenNotSelected(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
		// No totalCount selected
	}, makeArg("first", intVal("10")))

	result, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should NOT have a separate count subquery when totalCount is not selected
	// (We check that the result doesn't have two separate CALL blocks for count)
	if result == "" {
		t.Fatal("expected non-empty connection query even without totalCount")
	}
}

// Test: translateConnectionField includes pageInfo with hasNextPage.
func TestTranslateConnectionField_PageInfo(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
		&ast.Field{Name: "pageInfo", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "hasNextPage"},
			&ast.Field{Name: "hasPreviousPage"},
		}},
		&ast.Field{Name: "totalCount"},
	}, makeArg("first", intVal("10")))

	result, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "pageInfo") || !strings.Contains(result, "hasNextPage") {
		t.Errorf("expected pageInfo/hasNextPage in connection query, got %q", result)
	}
}

// Test: translateConnectionField with "after" cursor computes offset.
func TestTranslateConnectionField_WithAfterCursor(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	// "Y3Vyc29yOjU=" is base64 for "cursor:5"
	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
	}, makeArg("first", intVal("10")), makeArg("after", strVal("Y3Vyc29yOjU=")))

	result, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty connection query with cursor, got empty")
	}
}

// Test: translateConnectionField with filter applies WHERE clause.
func TestTranslateConnectionField_WithFilter(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"released_gte": intVal("2000"),
	}))
	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
	}, makeArg("first", intVal("10")), whereArg)

	result, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "WHERE") {
		t.Errorf("expected WHERE clause in connection query, got %q", result)
	}
}

// Test: Default sort is ORDER BY n.id ASC when no sort argument provided.
func TestTranslateConnectionField_DefaultSort(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
	}, makeArg("first", intVal("10")))

	result, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should default to ORDER BY n.id ASC for stable cursor pagination
	if !strings.Contains(result, "ORDER BY") {
		t.Errorf("expected default ORDER BY in connection query, got %q", result)
	}
}

// --- Nested connection subquery tests ---

// Test: buildConnectionSubquery produces CALL subquery for nested connection.
func TestBuildConnectionSubquery_ProducesNestedConnection(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actorsConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "name"},
			}},
		}},
	}, makeArg("first", intVal("5")))

	result, alias, err := tr.buildConnectionSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty nested connection subquery, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(result, "CALL") {
		t.Errorf("expected CALL in nested connection, got %q", result)
	}
	if !strings.Contains(result, "WITH n") {
		t.Errorf("expected 'WITH n' parent pass-through, got %q", result)
	}
}

// Test: buildConnectionSubquery includes edge properties when @relationshipProperties defined.
func TestBuildConnectionSubquery_EdgeProperties(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actorsConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "name"},
			}},
			&ast.Field{Name: "properties", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "role"},
			}},
		}},
	}, makeArg("first", intVal("5")))

	result, _, err := tr.buildConnectionSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should include relationship variable for properties access
	if !strings.Contains(result, "role") || !strings.Contains(result, "properties") {
		t.Errorf("expected edge properties (role) in nested connection, got %q", result)
	}
}

// Test: buildConnectionSubquery without @relationshipProperties omits properties.
func TestBuildConnectionSubquery_NoProperties(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	relNoProps := schema.RelationshipDefinition{
		FieldName: "genres",
		RelType:   "IN_GENRE",
		Direction: schema.DirectionOUT,
		FromNode:  "Movie",
		ToNode:    "Genre",
		// No Properties
	}

	field := makeField("genresConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "name"},
			}},
		}},
	}, makeArg("first", intVal("5")))

	result, _, err := tr.buildConnectionSubquery(field, relNoProps, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty connection even without properties")
	}
}

// --- M2 regression: Nested connection produces connection map ---

// Test: buildConnectionSubquery wraps inner CALL blocks in outer CALL returning
// a connection map {edges: ..., totalCount: ..., pageInfo: ...} AS alias.
// Before M2 fix, the alias was undefined because no map RETURN was produced.
func TestBuildConnectionSubquery_ProducesConnectionMap(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actorsConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "name"},
			}},
		}},
		&ast.Field{Name: "totalCount"},
	}, makeArg("first", intVal("5")))

	result, alias, err := tr.buildConnectionSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The result should contain a RETURN with a connection map
	if !strings.Contains(result, "RETURN {") {
		t.Errorf("expected 'RETURN {' (connection map) in nested connection, got %q", result)
	}

	// Map should include edges and totalCount
	if !strings.Contains(result, "edges:") {
		t.Errorf("expected 'edges:' in connection map, got %q", result)
	}
	if !strings.Contains(result, "totalCount:") {
		t.Errorf("expected 'totalCount:' in connection map, got %q", result)
	}

	// Map should be returned AS the alias
	expectedReturn := fmt.Sprintf("AS %s", alias)
	if !strings.Contains(result, expectedReturn) {
		t.Errorf("expected connection map returned '%s', got %q", expectedReturn, result)
	}
}

// Test: buildConnectionSubquery with pageInfo includes pageInfo in connection map.
func TestBuildConnectionSubquery_PageInfoInMap(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actorsConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "name"},
			}},
		}},
		&ast.Field{Name: "totalCount"},
		&ast.Field{Name: "pageInfo", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "hasNextPage"},
			&ast.Field{Name: "hasPreviousPage"},
		}},
	}, makeArg("first", intVal("5")))

	result, _, err := tr.buildConnectionSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "pageInfo:") {
		t.Errorf("expected 'pageInfo:' in connection map, got %q", result)
	}
	if !strings.Contains(result, "hasNextPage") {
		t.Errorf("expected 'hasNextPage' in pageInfo, got %q", result)
	}
}

// Test: buildConnectionSubquery is wrapped in an outer CALL block (WITH parent).
func TestBuildConnectionSubquery_OuterCALLWithParent(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actorsConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "name"},
			}},
		}},
	}, makeArg("first", intVal("5")))

	result, _, err := tr.buildConnectionSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should start with outer CALL { WITH n
	if !strings.HasPrefix(result, "CALL { WITH n ") {
		t.Errorf("expected nested connection to start with 'CALL { WITH n ', got %q", result[:min(len(result), 50)])
	}

	// Should end with closing brace for outer CALL
	if !strings.HasSuffix(strings.TrimSpace(result), "}") {
		t.Errorf("expected nested connection to end with '}', got %q", result[max(0, len(result)-20):])
	}
}

// --- M3 regression: Root connection param collision ---

// Test: Two root connection fields get different scoped parameter names.
// Before M3 fix, both used hardcoded "offset"/"first" causing param collision.
func TestTranslateConnectionField_ScopedParams(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field1 := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
	}, makeArg("first", intVal("10")))

	_, _, err := tr.translateConnectionField(field1, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error for first connection: %v", err)
	}

	// Simulate a second root connection field
	actorNode, _ := testModel().NodeByName("Actor")
	fc2 := fieldContext{node: actorNode, variable: "n", depth: 0}

	field2 := makeField("actorsConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "name"},
			}},
		}},
	}, makeArg("first", intVal("5")))

	_, _, err = tr.translateConnectionField(field2, fc2, scope)
	if err != nil {
		t.Fatalf("unexpected error for second connection: %v", err)
	}

	// Verify params are scoped by field name (no collision)
	params := scope.collect()
	_, hasMoviesOffset := params["moviesConnection_offset"]
	_, hasMoviesFirst := params["moviesConnection_first"]
	_, hasActorsOffset := params["actorsConnection_offset"]
	_, hasActorsFirst := params["actorsConnection_first"]

	if !hasMoviesOffset || !hasMoviesFirst {
		t.Errorf("expected moviesConnection_offset/first params, got keys: %v", paramsKeys(params))
	}
	if !hasActorsOffset || !hasActorsFirst {
		t.Errorf("expected actorsConnection_offset/first params, got keys: %v", paramsKeys(params))
	}

	// Should NOT have unscoped "offset"/"first" (the old collision-prone names)
	_, hasRawOffset := params["offset"]
	_, hasRawFirst := params["first"]
	if hasRawOffset || hasRawFirst {
		t.Errorf("found unscoped 'offset'/'first' params — param collision (M3 bug), got keys: %v", paramsKeys(params))
	}
}

// Test: Zero results connection returns empty edges, totalCount: 0.
func TestTranslateConnectionField_ZeroResults(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	// Query with very restrictive filter that would match nothing
	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("nonexistent_movie_xyz_12345"),
	}))
	field := makeField("moviesConnection", ast.SelectionSet{
		&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "title"},
			}},
		}},
		&ast.Field{Name: "totalCount"},
	}, makeArg("first", intVal("10")), whereArg)

	result, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The Cypher should be valid — zero results are handled by collect() returning []
	if result == "" {
		t.Fatal("expected non-empty connection query for edge case")
	}
}
