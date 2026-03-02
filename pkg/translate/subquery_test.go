package translate

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// actedInRel returns the ACTED_IN relationship from the test model.
func actedInRel() schema.RelationshipDefinition {
	return schema.RelationshipDefinition{
		FieldName: "actors",
		RelType:   "ACTED_IN",
		Direction: schema.DirectionIN,
		FromNode:  "Movie",
		ToNode:    "Actor",
		Properties: &schema.PropertiesDefinition{
			TypeName: "ActedInProperties",
			Fields: []schema.FieldDefinition{
				{Name: "role", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
			},
		},
	}
}

// averageRatingCypherField returns a scalar @cypher field definition.
func averageRatingCypherField() schema.CypherFieldDefinition {
	return schema.CypherFieldDefinition{
		Name:        "averageRating",
		GraphQLType: "Float",
		GoType:      "*float64",
		Statement:   "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
		Nullable:    true,
		IsList:      false,
	}
}

// similarMoviesCypherField returns a list @cypher field definition.
func similarMoviesCypherField() schema.CypherFieldDefinition {
	return schema.CypherFieldDefinition{
		Name:        "similarMovies",
		GraphQLType: "[Movie!]!",
		GoType:      "[]*Movie",
		Statement:   "MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) RETURN rec",
		IsList:      true,
	}
}

// --- TR-4: Nested relationship subquery tests ---

// Test: buildSubquery produces a CALL subquery with WITH parent, MATCH, and collect().
func TestBuildSubquery_ProducesCALLWithCollect(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actors", ast.SelectionSet{
		&ast.Field{Name: "name"},
		&ast.Field{Name: "born"},
	})

	result, alias, err := tr.buildSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty subquery, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(result, "CALL") {
		t.Errorf("expected CALL in subquery, got %q", result)
	}
	if !strings.Contains(result, "WITH n") {
		t.Errorf("expected 'WITH n' (parent variable) in subquery, got %q", result)
	}
	if !strings.Contains(result, "MATCH") {
		t.Errorf("expected MATCH in subquery, got %q", result)
	}
	if !strings.Contains(result, "ACTED_IN") {
		t.Errorf("expected ACTED_IN relationship type in subquery, got %q", result)
	}
	if !strings.Contains(result, "Actor") {
		t.Errorf("expected Actor label in subquery, got %q", result)
	}
	if !strings.Contains(result, "collect(") {
		t.Errorf("expected collect() in subquery, got %q", result)
	}
}

// Test: buildSubquery returns alias starting with "__sub".
func TestBuildSubquery_ReturnsSubAlias(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actors", ast.SelectionSet{
		&ast.Field{Name: "name"},
	})

	_, alias, err := tr.buildSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(alias, "__sub") {
		t.Errorf("expected alias to start with '__sub', got %q", alias)
	}
}

// Test: buildSubquery uses correct direction arrow pattern for IN direction.
// IN direction: (parent)<-[:TYPE]-(child)
func TestBuildSubquery_INDirection(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("actors", ast.SelectionSet{
		&ast.Field{Name: "name"},
	})

	result, _, err := tr.buildSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// IN direction means the arrow points into the parent: (n)<-[:ACTED_IN]-(a:Actor)
	if !strings.Contains(result, "<-[") {
		t.Errorf("expected '<-[' arrow for IN direction, got %q", result)
	}
}

// Test: buildSubquery uses correct direction arrow pattern for OUT direction.
// OUT direction: (parent)-[:TYPE]->(child)
func TestBuildSubquery_OUTDirection(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	outRel := schema.RelationshipDefinition{
		FieldName: "genres",
		RelType:   "IN_GENRE",
		Direction: schema.DirectionOUT,
		FromNode:  "Movie",
		ToNode:    "Genre",
	}

	field := makeField("genres", ast.SelectionSet{
		&ast.Field{Name: "name"},
	})

	result, _, err := tr.buildSubquery(field, outRel, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// OUT direction means arrow points away: (n)-[:IN_GENRE]->(g:Genre)
	if !strings.Contains(result, "]->(") {
		t.Errorf("expected ']->(' arrow for OUT direction, got %q", result)
	}
}

// Test: buildSubquery with filter on nested field scopes WHERE to subquery.
func TestBuildSubquery_WithFilter(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"name": strVal("Keanu"),
	}))
	field := makeField("actors", ast.SelectionSet{
		&ast.Field{Name: "name"},
	}, whereArg)

	result, _, err := tr.buildSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "WHERE") {
		t.Errorf("expected WHERE in nested subquery, got %q", result)
	}
}

// Test: buildSubquery with sort scopes ORDER BY to subquery.
func TestBuildSubquery_WithSort(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	sortArg := makeArg("sort", makeSortValue(map[string]string{"name": "ASC"}))
	field := makeField("actors", ast.SelectionSet{
		&ast.Field{Name: "name"},
	}, sortArg)

	result, _, err := tr.buildSubquery(field, actedInRel(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "ORDER BY") {
		t.Errorf("expected ORDER BY in nested subquery, got %q", result)
	}
}

// --- TR-4: @cypher subquery tests ---

// Test: buildCypherSubquery produces CALL subquery with "WITH parent AS this".
func TestBuildCypherSubquery_ProducesCALLWithThis(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("averageRating", ast.SelectionSet{})

	result, alias, err := tr.buildCypherSubquery(field, averageRatingCypherField(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty cypher subquery, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(result, "CALL") {
		t.Errorf("expected CALL in cypher subquery, got %q", result)
	}
	if !strings.Contains(result, "AS this") {
		t.Errorf("expected 'AS this' binding in cypher subquery, got %q", result)
	}
}

// Test: buildCypherSubquery for scalar fields includes LIMIT 1.
func TestBuildCypherSubquery_ScalarIncludesLimit1(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("averageRating", ast.SelectionSet{})

	result, _, err := tr.buildCypherSubquery(field, averageRatingCypherField(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "LIMIT 1") {
		t.Errorf("expected 'LIMIT 1' for scalar @cypher field, got %q", result)
	}
}

// Test: buildCypherSubquery for list fields omits LIMIT 1.
func TestBuildCypherSubquery_ListOmitsLimit(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("similarMovies", ast.SelectionSet{
		&ast.Field{Name: "title"},
	})

	result, _, err := tr.buildCypherSubquery(field, similarMoviesCypherField(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "LIMIT 1") {
		t.Errorf("list @cypher field should NOT have 'LIMIT 1', got %q", result)
	}
}

// Test: buildCypherSubquery returns alias starting with "__cypher".
func TestBuildCypherSubquery_ReturnsCypherAlias(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("averageRating", ast.SelectionSet{})

	_, alias, err := tr.buildCypherSubquery(field, averageRatingCypherField(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(alias, "__cypher") {
		t.Errorf("expected alias to start with '__cypher', got %q", alias)
	}
}

// Test: buildCypherSubquery includes the user's Cypher statement.
func TestBuildCypherSubquery_IncludesUserStatement(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := makeField("averageRating", ast.SelectionSet{})

	result, _, err := tr.buildCypherSubquery(field, averageRatingCypherField(), fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "MATCH (this)<-[r:REVIEWED]-()") {
		t.Errorf("expected user's Cypher statement in subquery, got %q", result)
	}
}

// Test: buildCypherSubquery with arguments passes parameters.
func TestBuildCypherSubquery_WithArguments(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	cfWithArgs := schema.CypherFieldDefinition{
		Name:        "similarMovies",
		GraphQLType: "[Movie!]!",
		GoType:      "[]*Movie",
		Statement:   "MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) RETURN rec LIMIT $limit",
		IsList:      true,
		Arguments: []schema.ArgumentDefinition{
			{Name: "limit", GraphQLType: "Int!", GoType: "int"},
		},
	}

	limitArg := makeArg("limit", intVal("3"))
	field := makeField("similarMovies", ast.SelectionSet{
		&ast.Field{Name: "title"},
	}, limitArg)

	result, _, err := tr.buildCypherSubquery(field, cfWithArgs, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The subquery should contain the statement and parameter handling
	if result == "" {
		t.Fatal("expected non-empty cypher subquery with arguments, got empty")
	}
}

// --- M1 regression: @cypher argument parameter names match user's $argName ---

// Test: buildCypherSubquery registers argument params with original names (not namespaced).
// The user's Cypher statement references $limit, so the params map must have key "limit".
// Before M1 fix, params had key "cypher0_limit" causing a Neo4j parameter-not-found error.
func TestBuildCypherSubquery_ArgParamsUseOriginalName(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	cfWithArgs := schema.CypherFieldDefinition{
		Name:        "similarMovies",
		GraphQLType: "[Movie!]!",
		GoType:      "[]*Movie",
		Statement:   "MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) RETURN rec LIMIT $limit",
		IsList:      true,
		Arguments: []schema.ArgumentDefinition{
			{Name: "limit", GraphQLType: "Int!", GoType: "int"},
		},
	}

	limitArg := makeArg("limit", intVal("3"))
	field := makeField("similarMovies", ast.SelectionSet{
		&ast.Field{Name: "title"},
	}, limitArg)

	_, _, err := tr.buildCypherSubquery(field, cfWithArgs, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the parameter is registered with original name "limit", not "cypher0_limit"
	params := scope.collect()
	if _, ok := params["limit"]; !ok {
		t.Errorf("expected params to have key 'limit', got keys: %v", paramsKeys(params))
	}
	if _, ok := params["cypher0_limit"]; ok {
		t.Error("params should NOT have namespaced key 'cypher0_limit' (M1 bug)")
	}
	if params["limit"] != int64(3) {
		t.Errorf("expected params['limit']=3, got %v", params["limit"])
	}
}

// paramsKeys returns the sorted keys of a map for error messages.
func paramsKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// --- Deeply nested subqueries ---

// Test: Nested relationship 3+ levels deep produces correctly nested CALL subqueries.
func TestBuildSubquery_DeepNesting(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	// Simulating depth=1 (inside a movie → actor subquery context)
	fc := fieldContext{node: node, variable: "a", depth: 1}

	// Actors have movies via a different relationship (reverse)
	reverseRel := schema.RelationshipDefinition{
		FieldName: "movies",
		RelType:   "ACTED_IN",
		Direction: schema.DirectionOUT,
		FromNode:  "Actor",
		ToNode:    "Movie",
	}

	field := makeField("movies", ast.SelectionSet{
		&ast.Field{Name: "title"},
	})

	result, _, err := tr.buildSubquery(field, reverseRel, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty deeply nested subquery, got empty")
	}
	if !strings.Contains(result, "WITH a") {
		t.Errorf("expected 'WITH a' (parent variable at depth 1), got %q", result)
	}
}
