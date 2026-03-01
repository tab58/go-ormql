package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-16: whereToClause + sortToFields resolver templates ---

// TestGenerateResolvers_WhereToClause_FunctionGenerated verifies that a
// {node}WhereToClause function is generated per node.
// Expected: generated source contains "movieWhereToClause" function.
func TestGenerateResolvers_WhereToClause_FunctionGenerated(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieWhereToClause") {
		t.Errorf("generated resolvers missing 'movieWhereToClause' function:\n%s", s)
	}
}

// TestGenerateResolvers_WhereToClause_ReturnsCypherWhereClause verifies that
// the whereToClause function returns cypher.WhereClause.
// Expected: function signature contains "cypher.WhereClause".
func TestGenerateResolvers_WhereToClause_ReturnsCypherWhereClause(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "WhereToClause") || !strings.Contains(s, "cypher.WhereClause") {
		t.Errorf("whereToClause function should return cypher.WhereClause:\n%s", s)
	}
}

// TestGenerateResolvers_WhereToClause_MapsEqualityField verifies that the
// whereToClause function maps an equality field to a cypher.Predicate with OpEq.
// Expected: generated source contains "cypher.Predicate" and "cypher.OpEq".
func TestGenerateResolvers_WhereToClause_MapsEqualityField(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.Predicate") {
		t.Errorf("whereToClause should create cypher.Predicate entries:\n%s", s)
	}
	if !strings.Contains(s, "cypher.OpEq") {
		t.Errorf("whereToClause should use cypher.OpEq for equality fields:\n%s", s)
	}
}

// TestGenerateResolvers_WhereToClause_MapsComparisonOps verifies that the
// whereToClause function maps comparison operator fields (_gt, _gte, _lt, _lte)
// to the corresponding cypher.Op constants.
// Expected: generated source contains cypher.OpGT, cypher.OpGTE, cypher.OpLT, cypher.OpLTE.
func TestGenerateResolvers_WhereToClause_MapsComparisonOps(t *testing.T) {
	src, err := GenerateResolvers(filterTestResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	for _, op := range []string{"cypher.OpGT", "cypher.OpGTE", "cypher.OpLT", "cypher.OpLTE"} {
		if !strings.Contains(s, op) {
			t.Errorf("whereToClause should map to %s:\n%s", op, s)
		}
	}
}

// TestGenerateResolvers_WhereToClause_MapsStringOps verifies that the
// whereToClause function maps string operator fields (_contains, _startsWith,
// _endsWith, _regex) to the corresponding cypher.Op constants.
// Expected: generated source contains cypher.OpContains, cypher.OpStartsWith, etc.
func TestGenerateResolvers_WhereToClause_MapsStringOps(t *testing.T) {
	src, err := GenerateResolvers(filterTestResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	for _, op := range []string{"cypher.OpContains", "cypher.OpStartsWith", "cypher.OpEndsWith", "cypher.OpRegex"} {
		if !strings.Contains(s, op) {
			t.Errorf("whereToClause should map to %s:\n%s", op, s)
		}
	}
}

// TestGenerateResolvers_WhereToClause_MapsListOps verifies that the whereToClause
// function maps _in and _nin fields to cypher.OpIn and cypher.OpNotIn.
// Expected: generated source contains cypher.OpIn and cypher.OpNotIn.
func TestGenerateResolvers_WhereToClause_MapsListOps(t *testing.T) {
	src, err := GenerateResolvers(filterTestResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.OpIn") {
		t.Errorf("whereToClause should map _in to cypher.OpIn:\n%s", s)
	}
	if !strings.Contains(s, "cypher.OpNotIn") {
		t.Errorf("whereToClause should map _nin to cypher.OpNotIn:\n%s", s)
	}
}

// TestGenerateResolvers_WhereToClause_MapsNegation verifies that the whereToClause
// function maps _not fields to cypher.OpNEq.
// Expected: generated source contains cypher.OpNEq.
func TestGenerateResolvers_WhereToClause_MapsNegation(t *testing.T) {
	src, err := GenerateResolvers(filterTestResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.OpNEq") {
		t.Errorf("whereToClause should map _not to cypher.OpNEq:\n%s", s)
	}
}

// TestGenerateResolvers_WhereToClause_MapsIsNull verifies that the whereToClause
// function maps _isNull fields to cypher.OpIsNull or cypher.OpIsNotNull.
// Expected: generated source contains cypher.OpIsNull.
func TestGenerateResolvers_WhereToClause_MapsIsNull(t *testing.T) {
	src, err := GenerateResolvers(filterTestResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.OpIsNull") {
		t.Errorf("whereToClause should map _isNull to cypher.OpIsNull:\n%s", s)
	}
}

// TestGenerateResolvers_WhereToClause_BooleanComposition verifies that the
// whereToClause function handles AND, OR, NOT boolean composition fields
// with recursive calls.
// Expected: generated source contains references to AND, OR, NOT handling.
func TestGenerateResolvers_WhereToClause_BooleanComposition(t *testing.T) {
	src, err := GenerateResolvers(filterTestResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// Should handle AND/OR as arrays and NOT as single recursive call
	if !strings.Contains(s, ".AND") {
		t.Errorf("whereToClause should handle AND composition:\n%s", s)
	}
	if !strings.Contains(s, ".OR") {
		t.Errorf("whereToClause should handle OR composition:\n%s", s)
	}
	if !strings.Contains(s, ".NOT") {
		t.Errorf("whereToClause should handle NOT composition:\n%s", s)
	}
}

// TestGenerateResolvers_SortToFields_FunctionGenerated verifies that a
// {node}SortToFields function is generated per node.
// Expected: generated source contains "movieSortToFields" function.
func TestGenerateResolvers_SortToFields_FunctionGenerated(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieSortToFields") {
		t.Errorf("generated resolvers missing 'movieSortToFields' function:\n%s", s)
	}
}

// TestGenerateResolvers_SortToFields_ReturnsCypherSortField verifies that
// the sortToFields function returns []cypher.SortField.
// Expected: function signature contains "[]cypher.SortField".
func TestGenerateResolvers_SortToFields_ReturnsCypherSortField(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "SortToFields") || !strings.Contains(s, "[]cypher.SortField") {
		t.Errorf("sortToFields function should return []cypher.SortField:\n%s", s)
	}
}

// TestGenerateResolvers_SortToFields_MapsSortDirection verifies that the
// sortToFields function maps SortDirection enum values to cypher.SortASC/SortDESC.
// Expected: generated source contains cypher.SortASC and cypher.SortDESC.
func TestGenerateResolvers_SortToFields_MapsSortDirection(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.SortASC") {
		t.Errorf("sortToFields should map to cypher.SortASC:\n%s", s)
	}
	if !strings.Contains(s, "cypher.SortDESC") {
		t.Errorf("sortToFields should map to cypher.SortDESC:\n%s", s)
	}
}

// TestGenerateResolvers_QueryResolver_CallsWhereToClause verifies that the
// list query resolver (e.g., Movies) calls whereToClause instead of the
// old WhereToMap + EqualityWhere bridge.
// Expected: generated source for Movies resolver contains "movieWhereToClause".
func TestGenerateResolvers_QueryResolver_CallsWhereToClause(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieWhereToClause") {
		t.Errorf("list query resolver should call movieWhereToClause:\n%s", s)
	}
}

// TestGenerateResolvers_QueryResolver_CallsSortToFields verifies that the
// list query resolver (e.g., Movies) calls sortToFields to pass sort
// parameters to NodeMatch.
// Expected: generated source for Movies resolver contains "movieSortToFields".
func TestGenerateResolvers_QueryResolver_CallsSortToFields(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieSortToFields") {
		t.Errorf("list query resolver should call movieSortToFields:\n%s", s)
	}
}

// TestGenerateResolvers_ConnectionResolver_CallsWhereToClause verifies that
// the connection resolver (e.g., MoviesConnection) calls whereToClause.
// Expected: MoviesConnection resolver uses movieWhereToClause.
func TestGenerateResolvers_ConnectionResolver_CallsWhereToClause(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The connection resolver should also use whereToClause
	// Count both occurrences — list query + connection query should both use it
	count := strings.Count(s, "movieWhereToClause")
	if count < 2 {
		t.Errorf("whereToClause should be called in both list and connection resolvers (found %d occurrences):\n%s", count, s)
	}
}

// TestGenerateResolvers_ConnectionResolver_CallsSortToFields verifies that
// the connection resolver (e.g., MoviesConnection) calls sortToFields.
// Expected: MoviesConnection resolver uses movieSortToFields.
func TestGenerateResolvers_ConnectionResolver_CallsSortToFields(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// Connection query should use sortToFields
	count := strings.Count(s, "movieSortToFields")
	if count < 2 {
		t.Errorf("sortToFields should be called in both list and connection resolvers (found %d occurrences):\n%s", count, s)
	}
}

// TestGenerateResolvers_MultiNode_WhereToClause verifies that whereToClause
// functions are generated for each node in a multi-node model.
// Expected: both "movieWhereToClause" and "actorWhereToClause".
func TestGenerateResolvers_MultiNode_WhereToClause(t *testing.T) {
	src, err := GenerateResolvers(multiResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieWhereToClause") {
		t.Errorf("missing movieWhereToClause for multi-node model:\n%s", s)
	}
	if !strings.Contains(s, "actorWhereToClause") {
		t.Errorf("missing actorWhereToClause for multi-node model:\n%s", s)
	}
}

// TestGenerateResolvers_MultiNode_SortToFields verifies that sortToFields
// functions are generated for each node in a multi-node model.
// Expected: both "movieSortToFields" and "actorSortToFields".
func TestGenerateResolvers_MultiNode_SortToFields(t *testing.T) {
	src, err := GenerateResolvers(multiResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieSortToFields") {
		t.Errorf("missing movieSortToFields for multi-node model:\n%s", s)
	}
	if !strings.Contains(s, "actorSortToFields") {
		t.Errorf("missing actorSortToFields for multi-node model:\n%s", s)
	}
}

// filterTestResolverModel returns a model with diverse scalar types
// for testing filter operator mapping in whereToClause.
func filterTestResolverModel() schema.GraphModel {
	return filterTestModel()
}
