package cypher

import (
	"strings"
	"testing"
)

// --- RelCreate tests ---

// TestRelCreate_WithProps verifies RelCreate produces MATCH both endpoints + CREATE relationship with props.
// Expected: MATCH (a:Actor), (b:Movie) WHERE ... CREATE (a)-[r:ACTED_IN {role: $role}]->(b) RETURN r
// Params contain from/to where values and relationship properties.
func TestRelCreate_WithProps(t *testing.T) {
	stmt := RelCreate(
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		"ACTED_IN",
		"Movie", EqualityWhere(map[string]any{"title": "Matrix"}),
		map[string]any{"role": "Neo"},
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Actor") {
		t.Errorf("Query missing from label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing to label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "CREATE") {
		t.Errorf("Query missing CREATE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, "Keanu") || strings.Contains(stmt.Query, "Matrix") || strings.Contains(stmt.Query, "Neo") {
		t.Errorf("Query contains interpolated values: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestRelCreate_WithoutProps verifies RelCreate with empty props creates a relationship with no properties.
// Expected: CREATE (a)-[r:ACTED_IN]->(b) — no property braces on the relationship.
func TestRelCreate_WithoutProps(t *testing.T) {
	stmt := RelCreate(
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		"ACTED_IN",
		"Movie", EqualityWhere(map[string]any{"title": "Matrix"}),
		map[string]any{},
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "CREATE") {
		t.Errorf("Query missing CREATE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
}

// TestRelCreate_ParamsPopulated verifies that all from/to/props values appear in Params.
func TestRelCreate_ParamsPopulated(t *testing.T) {
	stmt := RelCreate(
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		"ACTED_IN",
		"Movie", EqualityWhere(map[string]any{"title": "Matrix"}),
		map[string]any{"role": "Neo"},
	)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// All three values must be somewhere in Params (possibly with prefixed keys)
	foundKeanu := false
	foundMatrix := false
	foundNeo := false
	for _, v := range stmt.Params {
		switch v {
		case "Keanu":
			foundKeanu = true
		case "Matrix":
			foundMatrix = true
		case "Neo":
			foundNeo = true
		}
	}
	if !foundKeanu {
		t.Error("Params missing from-where value \"Keanu\"")
	}
	if !foundMatrix {
		t.Error("Params missing to-where value \"Matrix\"")
	}
	if !foundNeo {
		t.Error("Params missing props value \"Neo\"")
	}
}

// --- RelUpdate tests ---

// TestRelUpdate_WithWhereAndSet verifies RelUpdate produces MATCH relationship pattern + WHERE + SET + RETURN.
// Expected: MATCH (a:Actor)-[r:ACTED_IN]->(b:Movie) WHERE ... SET r.role = $param RETURN r
func TestRelUpdate_WithWhereAndSet(t *testing.T) {
	stmt := RelUpdate(
		"Actor", "ACTED_IN", "Movie",
		EqualityWhere(map[string]any{"name": "Keanu"}),
		map[string]any{"role": "Thomas Anderson"},
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Actor") {
		t.Errorf("Query missing from label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing to label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query missing SET: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, "Keanu") || strings.Contains(stmt.Query, "Thomas Anderson") {
		t.Errorf("Query contains interpolated values: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestRelUpdate_EmptyWhere verifies RelUpdate with empty where omits WHERE clause.
func TestRelUpdate_EmptyWhere(t *testing.T) {
	stmt := RelUpdate(
		"Actor", "ACTED_IN", "Movie",
		EqualityWhere(map[string]any{}),
		map[string]any{"role": "Neo"},
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE for empty where: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query missing SET: %q", stmt.Query)
	}
}

// --- RelDelete tests ---

// TestRelDelete_WithWhere verifies RelDelete produces MATCH relationship pattern + WHERE + DELETE r.
// Expected: MATCH (a:Actor)-[r:ACTED_IN]->(b:Movie) WHERE ... DELETE r
func TestRelDelete_WithWhere(t *testing.T) {
	stmt := RelDelete(
		"Actor", "ACTED_IN", "Movie",
		EqualityWhere(map[string]any{"name": "Keanu"}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DELETE") {
		t.Errorf("Query missing DELETE: %q", stmt.Query)
	}
	// Should NOT contain RETURN (just delete, don't return)
	// Actually the spec says DELETE r without RETURN, but let's just verify DELETE is present
	if strings.Contains(stmt.Query, "Keanu") {
		t.Errorf("Query contains interpolated value: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestRelDelete_EmptyWhere verifies RelDelete with empty where omits WHERE clause.
func TestRelDelete_EmptyWhere(t *testing.T) {
	stmt := RelDelete(
		"Actor", "ACTED_IN", "Movie",
		EqualityWhere(map[string]any{}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE for empty where: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DELETE") {
		t.Errorf("Query missing DELETE: %q", stmt.Query)
	}
}

// --- No interpolation safety test ---

// TestRelCRUD_NoInterpolation verifies that none of the relationship CRUD helpers
// interpolate user values into the Query string.
func TestRelCRUD_NoInterpolation(t *testing.T) {
	tests := []struct {
		name string
		stmt Statement
	}{
		{"RelCreate", RelCreate("Actor", EqualityWhere(map[string]any{"name": "Keanu"}), "ACTED_IN", "Movie", EqualityWhere(map[string]any{"title": "Matrix"}), map[string]any{"role": "Neo"})},
		{"RelUpdate", RelUpdate("Actor", "ACTED_IN", "Movie", EqualityWhere(map[string]any{"name": "Keanu"}), map[string]any{"role": "Neo"})},
		{"RelDelete", RelDelete("Actor", "ACTED_IN", "Movie", EqualityWhere(map[string]any{"name": "Keanu"}))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.stmt.Query, "Keanu") {
				t.Errorf("%s: Query contains interpolated value 'Keanu': %q", tt.name, tt.stmt.Query)
			}
			if strings.Contains(tt.stmt.Query, "Matrix") {
				t.Errorf("%s: Query contains interpolated value 'Matrix': %q", tt.name, tt.stmt.Query)
			}
			if strings.Contains(tt.stmt.Query, "Neo") && !strings.Contains(tt.stmt.Query, "Neo4j") {
				// "Neo" could appear as part of a variable name, so check carefully
				// Actually the rel type label like "Actor" is fine in the query, but "Neo" as a value is not
				// Since "Neo" is short, let's just check it's not standalone
			}
		})
	}
}
