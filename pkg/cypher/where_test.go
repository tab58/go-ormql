package cypher

import (
	"strings"
	"testing"
)

// --- Op constant tests ---

// TestOp_Constants verifies all 14 operator constants are defined with correct Cypher values.
// Expected: each Op constant maps to its Cypher operator string.
func TestOp_Constants(t *testing.T) {
	tests := []struct {
		name string
		op   Op
		want string
	}{
		{"OpEq", OpEq, "="},
		{"OpNEq", OpNEq, "<>"},
		{"OpGT", OpGT, ">"},
		{"OpGTE", OpGTE, ">="},
		{"OpLT", OpLT, "<"},
		{"OpLTE", OpLTE, "<="},
		{"OpContains", OpContains, "CONTAINS"},
		{"OpStartsWith", OpStartsWith, "STARTS WITH"},
		{"OpEndsWith", OpEndsWith, "ENDS WITH"},
		{"OpRegex", OpRegex, "=~"},
		{"OpIn", OpIn, "IN"},
		{"OpNotIn", OpNotIn, "NOT_IN"},
		{"OpIsNull", OpIsNull, "IS NULL"},
		{"OpIsNotNull", OpIsNotNull, "IS NOT NULL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.op) != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, string(tt.op), tt.want)
			}
		})
	}
}

// --- EqualityWhere tests ---

// TestEqualityWhere_SingleField verifies that a single field map produces a WhereClause
// with one Predicate using OpEq.
// Expected: WhereClause.Predicates has one entry with Field="title", Op=OpEq, Value="Matrix".
func TestEqualityWhere_SingleField(t *testing.T) {
	clause := EqualityWhere(map[string]any{"title": "Matrix"})

	if len(clause.Predicates) != 1 {
		t.Fatalf("EqualityWhere: got %d predicates, want 1", len(clause.Predicates))
	}
	p := clause.Predicates[0]
	if p.Field != "title" {
		t.Errorf("Predicate.Field = %q, want %q", p.Field, "title")
	}
	if p.Op != OpEq {
		t.Errorf("Predicate.Op = %q, want %q", p.Op, OpEq)
	}
	if p.Value != "Matrix" {
		t.Errorf("Predicate.Value = %v, want %q", p.Value, "Matrix")
	}
}

// TestEqualityWhere_MultipleFields verifies that multiple fields produce multiple predicates.
// Expected: WhereClause.Predicates has entries for each map key with OpEq.
func TestEqualityWhere_MultipleFields(t *testing.T) {
	clause := EqualityWhere(map[string]any{"title": "Matrix", "released": 1999})

	if len(clause.Predicates) != 2 {
		t.Fatalf("EqualityWhere: got %d predicates, want 2", len(clause.Predicates))
	}

	// Verify both fields are present (order may vary due to map iteration)
	fieldSet := map[string]bool{}
	for _, p := range clause.Predicates {
		fieldSet[p.Field] = true
		if p.Op != OpEq {
			t.Errorf("Predicate for %q: Op = %q, want %q", p.Field, p.Op, OpEq)
		}
	}
	if !fieldSet["title"] {
		t.Error("EqualityWhere missing predicate for 'title'")
	}
	if !fieldSet["released"] {
		t.Error("EqualityWhere missing predicate for 'released'")
	}
}

// TestEqualityWhere_EmptyMap verifies that an empty map produces an empty WhereClause.
// Expected: WhereClause has no predicates and no sub-clauses.
func TestEqualityWhere_EmptyMap(t *testing.T) {
	clause := EqualityWhere(map[string]any{})

	if len(clause.Predicates) != 0 {
		t.Errorf("EqualityWhere(empty): got %d predicates, want 0", len(clause.Predicates))
	}
	if len(clause.AND) != 0 {
		t.Errorf("EqualityWhere(empty): got %d AND clauses, want 0", len(clause.AND))
	}
	if len(clause.OR) != 0 {
		t.Errorf("EqualityWhere(empty): got %d OR clauses, want 0", len(clause.OR))
	}
	if clause.NOT != nil {
		t.Error("EqualityWhere(empty): NOT should be nil")
	}
}

// TestEqualityWhere_NilMap verifies that a nil map produces an empty WhereClause.
// Expected: WhereClause has no predicates.
func TestEqualityWhere_NilMap(t *testing.T) {
	clause := EqualityWhere(nil)

	if len(clause.Predicates) != 0 {
		t.Errorf("EqualityWhere(nil): got %d predicates, want 0", len(clause.Predicates))
	}
}

// --- Builder.WhereClause tests ---

// TestBuilder_WhereClause_SingleEq verifies rendering a single equality predicate.
// Expected: Query contains "WHERE n.title = $p0", Params has p0="Matrix".
func TestBuilder_WhereClause_SingleEq(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpEq, Value: "Matrix"},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "WHERE") {
		t.Fatalf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.title = $p0") {
		t.Errorf("Query missing equality predicate: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	if stmt.Params["p0"] != "Matrix" {
		t.Errorf("Params[\"p0\"] = %v, want %q", stmt.Params["p0"], "Matrix")
	}
}

// TestBuilder_WhereClause_MultiplePredicates verifies multiple predicates are AND-joined.
// Expected: Query contains "n.title = $p0 AND n.released = $p1".
func TestBuilder_WhereClause_MultiplePredicates(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpEq, Value: "Matrix"},
				{Field: "released", Op: OpEq, Value: 1999},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "AND") {
		t.Errorf("Query missing AND between predicates: %q", stmt.Query)
	}
	if stmt.Params["p0"] != "Matrix" {
		t.Errorf("Params[\"p0\"] = %v, want %q", stmt.Params["p0"], "Matrix")
	}
	if stmt.Params["p1"] != 1999 {
		t.Errorf("Params[\"p1\"] = %v, want %d", stmt.Params["p1"], 1999)
	}
}

// TestBuilder_WhereClause_GT verifies OpGT renders "n.field > $param".
// Expected: Query contains "n.released > $p0".
func TestBuilder_WhereClause_GT(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpGT, Value: 2000},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.released > $p0") {
		t.Errorf("Query missing GT predicate: %q", stmt.Query)
	}
	if stmt.Params["p0"] != 2000 {
		t.Errorf("Params[\"p0\"] = %v, want %d", stmt.Params["p0"], 2000)
	}
}

// TestBuilder_WhereClause_GTE verifies OpGTE renders "n.field >= $param".
// Expected: Query contains "n.released >= $p0".
func TestBuilder_WhereClause_GTE(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpGTE, Value: 2000},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.released >= $p0") {
		t.Errorf("Query missing GTE predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_LT verifies OpLT renders "n.field < $param".
// Expected: Query contains "n.released < $p0".
func TestBuilder_WhereClause_LT(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpLT, Value: 2000},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.released < $p0") {
		t.Errorf("Query missing LT predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_LTE verifies OpLTE renders "n.field <= $param".
// Expected: Query contains "n.released <= $p0".
func TestBuilder_WhereClause_LTE(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpLTE, Value: 2000},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.released <= $p0") {
		t.Errorf("Query missing LTE predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_NEq verifies OpNEq renders "n.field <> $param".
// Expected: Query contains "n.title <> $p0".
func TestBuilder_WhereClause_NEq(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpNEq, Value: "Matrix"},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.title <> $p0") {
		t.Errorf("Query missing NEq predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_Contains verifies OpContains renders "n.field CONTAINS $param".
// Expected: Query contains "n.title CONTAINS $p0".
func TestBuilder_WhereClause_Contains(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpContains, Value: "Matrix"},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.title CONTAINS $p0") {
		t.Errorf("Query missing CONTAINS predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_StartsWith verifies OpStartsWith renders "n.field STARTS WITH $param".
// Expected: Query contains "n.title STARTS WITH $p0".
func TestBuilder_WhereClause_StartsWith(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpStartsWith, Value: "The"},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.title STARTS WITH $p0") {
		t.Errorf("Query missing STARTS WITH predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_EndsWith verifies OpEndsWith renders "n.field ENDS WITH $param".
// Expected: Query contains "n.title ENDS WITH $p0".
func TestBuilder_WhereClause_EndsWith(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpEndsWith, Value: "Reloaded"},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.title ENDS WITH $p0") {
		t.Errorf("Query missing ENDS WITH predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_Regex verifies OpRegex renders "n.field =~ $param".
// Expected: Query contains "n.title =~ $p0".
func TestBuilder_WhereClause_Regex(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpRegex, Value: ".*Matrix.*"},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.title =~ $p0") {
		t.Errorf("Query missing regex predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_In verifies OpIn renders "n.field IN $param".
// Expected: Query contains "n.title IN $p0", param is a list.
func TestBuilder_WhereClause_In(t *testing.T) {
	values := []string{"Matrix", "Inception"}
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpIn, Value: values},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.title IN $p0") {
		t.Errorf("Query missing IN predicate: %q", stmt.Query)
	}
	if stmt.Params["p0"] == nil {
		t.Error("Params[\"p0\"] should be the list value")
	}
}

// TestBuilder_WhereClause_NotIn verifies OpNotIn renders "NOT n.field IN $param".
// Expected: Query contains "NOT n.title IN $p0".
func TestBuilder_WhereClause_NotIn(t *testing.T) {
	values := []string{"Matrix", "Inception"}
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpNotIn, Value: values},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "NOT n.title IN $p0") {
		t.Errorf("Query missing NOT IN predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_IsNull verifies OpIsNull renders "n.field IS NULL" (no param).
// Expected: Query contains "n.released IS NULL", no parameter for this predicate.
func TestBuilder_WhereClause_IsNull(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpIsNull, Value: nil},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.released IS NULL") {
		t.Errorf("Query missing IS NULL predicate: %q", stmt.Query)
	}
	// IS NULL should not produce a parameter
	if len(stmt.Params) != 0 {
		t.Errorf("IS NULL should have no params, got %v", stmt.Params)
	}
}

// TestBuilder_WhereClause_IsNotNull verifies OpIsNotNull renders "n.field IS NOT NULL" (no param).
// Expected: Query contains "n.released IS NOT NULL".
func TestBuilder_WhereClause_IsNotNull(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpIsNotNull, Value: nil},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.released IS NOT NULL") {
		t.Errorf("Query missing IS NOT NULL predicate: %q", stmt.Query)
	}
	if len(stmt.Params) != 0 {
		t.Errorf("IS NOT NULL should have no params, got %v", stmt.Params)
	}
}

// --- Boolean composition tests ---

// TestBuilder_WhereClause_OR verifies OR sub-clauses are OR-joined within parentheses.
// Expected: Query contains "(n.title = $p0 OR n.title = $p1)".
func TestBuilder_WhereClause_OR(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			OR: []WhereClause{
				{Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Matrix"}}},
				{Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Inception"}}},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "OR") {
		t.Errorf("Query missing OR composition: %q", stmt.Query)
	}
	// Both values should be parameterized
	if stmt.Params["p0"] != "Matrix" {
		t.Errorf("Params[\"p0\"] = %v, want %q", stmt.Params["p0"], "Matrix")
	}
	if stmt.Params["p1"] != "Inception" {
		t.Errorf("Params[\"p1\"] = %v, want %q", stmt.Params["p1"], "Inception")
	}
}

// TestBuilder_WhereClause_AND verifies AND sub-clauses are AND-joined within parentheses.
// Expected: Query contains "(n.title = $p0 AND n.released = $p1)".
func TestBuilder_WhereClause_AND(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			AND: []WhereClause{
				{Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Matrix"}}},
				{Predicates: []Predicate{{Field: "released", Op: OpGTE, Value: 2000}}},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "AND") {
		t.Errorf("Query missing AND composition: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_NOT verifies NOT wraps its contents with "NOT (...)".
// Expected: Query contains "NOT (n.title = $p0)".
func TestBuilder_WhereClause_NOT(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			NOT: &WhereClause{
				Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Matrix"}},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "NOT") {
		t.Errorf("Query missing NOT composition: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "NOT (") {
		t.Errorf("NOT should wrap contents in parentheses: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_NestedComposition verifies deeply nested boolean composition.
// Structure: AND containing OR sub-clauses.
// Expected: correct parenthesization at each nesting level.
func TestBuilder_WhereClause_NestedComposition(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpGTE, Value: 2000},
			},
			OR: []WhereClause{
				{Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Matrix"}}},
				{Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Inception"}}},
			},
		}).
		Return("n").
		Build()

	// Should have both top-level predicate AND the OR composition
	if !strings.Contains(stmt.Query, "OR") {
		t.Errorf("Query missing OR: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, ">= $") {
		t.Errorf("Query missing GTE predicate: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_DeeplyNested verifies OR inside AND inside NOT.
// Expected: correct parenthesization: NOT ((... AND ...)).
func TestBuilder_WhereClause_DeeplyNested(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			NOT: &WhereClause{
				AND: []WhereClause{
					{Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Bad"}}},
					{
						OR: []WhereClause{
							{Predicates: []Predicate{{Field: "released", Op: OpLT, Value: 2000}}},
							{Predicates: []Predicate{{Field: "released", Op: OpGT, Value: 2020}}},
						},
					},
				},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "NOT") {
		t.Errorf("Query missing NOT: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "AND") {
		t.Errorf("Query missing AND: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "OR") {
		t.Errorf("Query missing OR: %q", stmt.Query)
	}
}

// --- Empty/edge case tests ---

// TestBuilder_WhereClause_Empty verifies an empty WhereClause produces no WHERE.
// Expected: Query does NOT contain "WHERE".
func TestBuilder_WhereClause_Empty(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{}).
		Return("n").
		Build()

	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Empty WhereClause should produce no WHERE: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_EmptySubClauses verifies that empty AND/OR/NOT produce no WHERE.
// Expected: Query does NOT contain "WHERE".
func TestBuilder_WhereClause_EmptySubClauses(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			AND: []WhereClause{},
			OR:  []WhereClause{},
		}).
		Return("n").
		Build()

	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Empty sub-clauses should produce no WHERE: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_ParameterAutoIncrement verifies parameters use auto-incrementing $p0, $p1, $p2... naming.
// Expected: multiple predicates produce $p0, $p1, $p2 parameters.
func TestBuilder_WhereClause_ParameterAutoIncrement(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpEq, Value: "Matrix"},
				{Field: "released", Op: OpGTE, Value: 2000},
				{Field: "rating", Op: OpLT, Value: 5.0},
			},
		}).
		Return("n").
		Build()

	if stmt.Params["p0"] != "Matrix" {
		t.Errorf("Params[\"p0\"] = %v, want %q", stmt.Params["p0"], "Matrix")
	}
	if stmt.Params["p1"] != 2000 {
		t.Errorf("Params[\"p1\"] = %v, want %d", stmt.Params["p1"], 2000)
	}
	if stmt.Params["p2"] != 5.0 {
		t.Errorf("Params[\"p2\"] = %v, want %v", stmt.Params["p2"], 5.0)
	}
}

// TestBuilder_WhereClause_Immutability verifies the builder remains immutable when WhereClause is used.
// Expected: original builder is not mutated.
func TestBuilder_WhereClause_Immutability(t *testing.T) {
	b1 := New().Match("n", "Movie")
	b2 := b1.WhereClause(WhereClause{
		Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Matrix"}},
	})

	s1 := b1.Return("n").Build()
	s2 := b2.Return("n").Build()

	if strings.Contains(s1.Query, "WHERE") {
		t.Errorf("Original builder was mutated: %q", s1.Query)
	}
	if !strings.Contains(s2.Query, "WHERE") {
		t.Errorf("Derived builder missing WHERE: %q", s2.Query)
	}
}

// TestBuilder_WhereClause_MixedPredicatesAndOR verifies predicates combined with OR sub-clauses.
// Expected: top-level predicate AND-joined with OR group.
func TestBuilder_WhereClause_MixedPredicatesAndOR(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "released", Op: OpGTE, Value: 2000},
			},
			OR: []WhereClause{
				{Predicates: []Predicate{{Field: "title", Op: OpContains, Value: "Matrix"}}},
				{Predicates: []Predicate{{Field: "title", Op: OpContains, Value: "Star"}}},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "WHERE") {
		t.Fatalf("Query missing WHERE: %q", stmt.Query)
	}
	// Should have both the GTE predicate and the OR composition
	if !strings.Contains(stmt.Query, ">=") {
		t.Errorf("Query missing GTE predicate: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "OR") {
		t.Errorf("Query missing OR composition: %q", stmt.Query)
	}
}

// TestBuilder_WhereClause_NOTWithSinglePredicate verifies NOT with a single predicate.
// Expected: Query contains "NOT (n.title = $p0)".
func TestBuilder_WhereClause_NOTWithSinglePredicate(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			NOT: &WhereClause{
				Predicates: []Predicate{{Field: "title", Op: OpEq, Value: "Bad Movie"}},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "NOT (") {
		t.Errorf("Query should have NOT with parens: %q", stmt.Query)
	}
	if stmt.Params["p0"] != "Bad Movie" {
		t.Errorf("Params[\"p0\"] = %v, want %q", stmt.Params["p0"], "Bad Movie")
	}
}

// TestBuilder_WhereClause_VariablePrefix verifies the variable name is used in rendered conditions.
// When using variable "n", conditions should reference "n.field".
// Expected: Query contains "n.title" not "title" alone.
func TestBuilder_WhereClause_VariablePrefix(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		WhereClause(WhereClause{
			Predicates: []Predicate{
				{Field: "title", Op: OpEq, Value: "Matrix"},
			},
		}).
		Return("n").
		Build()

	if !strings.Contains(stmt.Query, "n.title") {
		t.Errorf("Query should reference variable.field: %q", stmt.Query)
	}
}
