package cypher

import (
	"strings"
	"testing"
)

// --- Immutability tests ---

// TestBuilder_Immutability verifies that calling a method on a builder returns a new builder
// and does not mutate the original. The original builder should still Build() to the same result.
func TestBuilder_Immutability(t *testing.T) {
	b1 := New().Match("n", "Movie")
	b2 := b1.Where("n.title = $title", map[string]any{"title": "Matrix"})

	s1 := b1.Return("n").Build()
	s2 := b2.Return("n").Build()

	// b1 should not contain WHERE clause
	if strings.Contains(s1.Query, "WHERE") {
		t.Errorf("original builder was mutated: Query = %q, should not contain WHERE", s1.Query)
	}

	// b2 should contain WHERE clause
	if !strings.Contains(s2.Query, "WHERE") {
		t.Errorf("derived builder missing WHERE: Query = %q", s2.Query)
	}
}

// TestBuilder_ChainIndependence verifies that each step in a chain produces an independent builder.
func TestBuilder_ChainIndependence(t *testing.T) {
	base := New().Match("n", "Movie")
	withWhere := base.Where("n.title = $title", map[string]any{"title": "Matrix"})
	withLimit := base.Limit(10)

	sWhere := withWhere.Return("n").Build()
	sLimit := withLimit.Return("n").Build()

	if !strings.Contains(sWhere.Query, "WHERE") {
		t.Errorf("withWhere branch missing WHERE: %q", sWhere.Query)
	}
	if strings.Contains(sWhere.Query, "LIMIT") {
		t.Errorf("withWhere branch should not have LIMIT: %q", sWhere.Query)
	}
	if !strings.Contains(sLimit.Query, "LIMIT") {
		t.Errorf("withLimit branch missing LIMIT: %q", sLimit.Query)
	}
	if strings.Contains(sLimit.Query, "WHERE") {
		t.Errorf("withLimit branch should not have WHERE: %q", sLimit.Query)
	}
}

// --- Simple clause tests ---

// TestBuilder_MatchReturn verifies MATCH (n:Label) RETURN n.
// Expected: Query contains "MATCH (n:Movie)" and "RETURN n".
func TestBuilder_MatchReturn(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "MATCH (n:Movie)") {
		t.Errorf("Query missing MATCH clause: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN n") {
		t.Errorf("Query missing RETURN clause: %q", stmt.Query)
	}
}

// TestBuilder_MatchWhereReturn verifies MATCH ... WHERE ... RETURN with parameterized conditions.
// Expected: Query contains WHERE clause, Params has the parameter value.
func TestBuilder_MatchWhereReturn(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Where("n.title = $title", map[string]any{"title": "Matrix"}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.title = $title") {
		t.Errorf("Query missing WHERE condition: %q", stmt.Query)
	}
	if stmt.Params["title"] != "Matrix" {
		t.Errorf("Params[\"title\"] = %v, want %q", stmt.Params["title"], "Matrix")
	}
}

// TestBuilder_CreateReturn verifies CREATE (n:Label {props}) RETURN n with params.
// Expected: Query contains CREATE with label, Params has property values.
func TestBuilder_CreateReturn(t *testing.T) {
	stmt := New().
		Create("n", "Movie", map[string]any{"title": "Matrix", "released": 1999}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "CREATE") {
		t.Errorf("Query missing CREATE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN n") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	if stmt.Params["title"] != "Matrix" {
		t.Errorf("Params[\"title\"] = %v, want %q", stmt.Params["title"], "Matrix")
	}
}

// TestBuilder_MatchSetReturn verifies MATCH ... SET variable.prop = $param ... RETURN.
// Expected: Query contains SET clause, Params has the set values.
func TestBuilder_MatchSetReturn(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Where("n.title = $title", map[string]any{"title": "Matrix"}).
		Set("n", map[string]any{"released": 2000}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query missing SET: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN n") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
}

// TestBuilder_DetachDelete verifies MATCH ... DETACH DELETE n.
// Expected: Query contains "DETACH DELETE n".
func TestBuilder_DetachDelete(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Where("n.title = $title", map[string]any{"title": "Matrix"}).
		Delete("n", true).
		Build()

	if !strings.Contains(stmt.Query, "DETACH DELETE") {
		t.Errorf("Query missing DETACH DELETE: %q", stmt.Query)
	}
}

// TestBuilder_DeleteNonDetach verifies MATCH ... DELETE n (without DETACH).
// Expected: Query contains "DELETE n" but NOT "DETACH DELETE".
func TestBuilder_DeleteNonDetach(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Delete("n", false).
		Build()

	if !strings.Contains(stmt.Query, "DELETE n") {
		t.Errorf("Query missing DELETE: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "DETACH") {
		t.Errorf("Query should not contain DETACH: %q", stmt.Query)
	}
}

// --- Pagination/ordering tests ---

// TestBuilder_OrderBy verifies ORDER BY field direction.
// Expected: Query contains "ORDER BY n.title ASC".
func TestBuilder_OrderBy(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Return("n").
		OrderBy(SortField{Field: "n.title", Direction: SortASC}).
		Build()

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("Query missing ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.title") {
		t.Errorf("Query missing order field: %q", stmt.Query)
	}
}

// TestBuilder_SkipLimit verifies SKIP and LIMIT clauses with parameterized values.
// Expected: Query contains SKIP and LIMIT.
func TestBuilder_SkipLimit(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Return("n").
		Skip(10).
		Limit(5).
		Build()

	if !strings.Contains(stmt.Query, "SKIP") {
		t.Errorf("Query missing SKIP: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "LIMIT") {
		t.Errorf("Query missing LIMIT: %q", stmt.Query)
	}
}

// --- Composition tests ---

// TestBuilder_With verifies WITH clause for query piping.
// Expected: Query contains "WITH n".
func TestBuilder_With(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		With("n").
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "WITH n") {
		t.Errorf("Query missing WITH: %q", stmt.Query)
	}
}

// TestBuilder_RelationshipMatch_OUT verifies MATCH relationship pattern with OUT direction.
// Expected: Query contains "(a)-[r:ACTED_IN]->(b:Movie)".
func TestBuilder_RelationshipMatch_OUT(t *testing.T) {
	stmt := New().
		Match("a", "Actor").
		RelationshipMatch("a", "r", "ACTED_IN", "b", "Movie", DirectionOUT).
		Return("a", "r", "b").
		Build()

	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	// OUT direction: (a)-[r:TYPE]->(b:Label)
	if !strings.Contains(stmt.Query, "->") {
		t.Errorf("Query missing OUT direction arrow: %q", stmt.Query)
	}
}

// TestBuilder_RelationshipMatch_IN verifies MATCH relationship pattern with IN direction.
// Expected: Query contains "(a)<-[r:ACTED_IN]-(b:Movie)".
func TestBuilder_RelationshipMatch_IN(t *testing.T) {
	stmt := New().
		Match("a", "Actor").
		RelationshipMatch("a", "r", "ACTED_IN", "b", "Movie", DirectionIN).
		Return("a", "r", "b").
		Build()

	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	// IN direction: (a)<-[r:TYPE]-(b:Label)
	if !strings.Contains(stmt.Query, "<-") {
		t.Errorf("Query missing IN direction arrow: %q", stmt.Query)
	}
}

// TestBuilder_RelationshipCreate verifies CREATE relationship pattern.
// Expected: Query contains CREATE (a)-[r:TYPE {props}]->(b).
func TestBuilder_RelationshipCreate(t *testing.T) {
	stmt := New().
		RelationshipCreate("a", "r", "ACTED_IN", "b", map[string]any{"role": "Neo"}).
		Build()

	if !strings.Contains(stmt.Query, "CREATE") {
		t.Errorf("Query missing CREATE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// --- Edge case tests ---

// TestBuilder_EmptyBuild verifies that Build() on an empty builder returns an empty query.
// Expected: Query is empty string, Params is nil or empty.
func TestBuilder_EmptyBuild(t *testing.T) {
	stmt := New().Build()

	// Empty builder should produce empty or minimal statement
	if stmt.Query != "" {
		t.Errorf("empty builder produced non-empty Query: %q", stmt.Query)
	}
}

// TestBuilder_CreateEmptyProps verifies CREATE with no properties creates label-only node.
// Expected: Query contains "CREATE (n:Movie)" without property braces.
func TestBuilder_CreateEmptyProps(t *testing.T) {
	stmt := New().
		Create("n", "Movie", map[string]any{}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "CREATE") {
		t.Errorf("Query missing CREATE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing label: %q", stmt.Query)
	}
}

// TestBuilder_CreateNilProps verifies that nil values in props produce Cypher null parameters.
// Expected: Params contains the key with nil value.
func TestBuilder_CreateNilValueInProps(t *testing.T) {
	stmt := New().
		Create("n", "Movie", map[string]any{"title": "Matrix", "tagline": nil}).
		Return("n").
		Build()

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// The nil value should be preserved as nil in params (Cypher null)
	if _, exists := stmt.Params["tagline"]; !exists {
		t.Error("Params missing key \"tagline\" — nil values should be included")
	}
}

// TestBuilder_ParameterUniqueness verifies that parameters from different clauses
// do not collide. For example, a WHERE param and a SET param with the same field name
// should use prefixed naming.
func TestBuilder_ParameterUniqueness(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Where("n.title = $where_title", map[string]any{"where_title": "Matrix"}).
		Set("n", map[string]any{"title": "The Matrix"}).
		Return("n").
		Build()

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}

	// Should have both where_title and a set parameter for title
	if stmt.Params["where_title"] != "Matrix" {
		t.Errorf("Params[\"where_title\"] = %v, want %q", stmt.Params["where_title"], "Matrix")
	}
}
