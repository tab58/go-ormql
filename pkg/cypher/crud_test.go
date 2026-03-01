package cypher

import (
	"strings"
	"testing"
)

// --- NodeCreate tests ---

// TestNodeCreate_WithProps verifies NodeCreate produces CREATE with parameterized properties.
// Expected: CREATE (n:Movie {title: $title, released: $released}) RETURN n
// Params: {title: "Matrix", released: 1999}
func TestNodeCreate_WithProps(t *testing.T) {
	stmt := NodeCreate("Movie", map[string]any{"title": "Matrix", "released": 1999})

	if !strings.Contains(stmt.Query, "CREATE") {
		t.Errorf("Query missing CREATE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	// Verify parameters are set, not interpolated
	if strings.Contains(stmt.Query, "Matrix") {
		t.Errorf("Query contains interpolated value 'Matrix' — must use $param: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	if stmt.Params["title"] != "Matrix" {
		t.Errorf("Params[\"title\"] = %v, want %q", stmt.Params["title"], "Matrix")
	}
	if stmt.Params["released"] != 1999 {
		t.Errorf("Params[\"released\"] = %v, want %d", stmt.Params["released"], 1999)
	}
}

// TestNodeCreate_EmptyProps verifies NodeCreate with empty props creates a label-only node.
// Expected: CREATE (n:Movie) RETURN n — no property braces.
func TestNodeCreate_EmptyProps(t *testing.T) {
	stmt := NodeCreate("Movie", map[string]any{})

	if !strings.Contains(stmt.Query, "CREATE") {
		t.Errorf("Query missing CREATE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
}

// TestNodeCreate_NilValueInProps verifies that nil values in props are preserved as Cypher null.
// Expected: Params contains the key with nil value.
func TestNodeCreate_NilValueInProps(t *testing.T) {
	stmt := NodeCreate("Movie", map[string]any{"title": "Matrix", "tagline": nil})

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	if _, exists := stmt.Params["tagline"]; !exists {
		t.Error("Params missing key \"tagline\" — nil values should be included for Cypher null")
	}
}

// --- NodeMatch tests ---

// TestNodeMatch_WithWhere verifies NodeMatch produces MATCH with WHERE and parameterized condition.
// Expected: MATCH (n:Movie) WHERE n.title = $title RETURN n
// Params: {title: "Matrix"}
func TestNodeMatch_WithWhere(t *testing.T) {
	stmt := NodeMatch("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), nil)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "Matrix") {
		t.Errorf("Query contains interpolated value — must use $param: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	if stmt.Params["title"] != "Matrix" {
		t.Errorf("Params[\"title\"] = %v, want %q", stmt.Params["title"], "Matrix")
	}
}

// TestNodeMatch_EmptyWhere verifies NodeMatch with empty where map omits WHERE clause.
// Expected: MATCH (n:Movie) RETURN n — no WHERE.
func TestNodeMatch_EmptyWhere(t *testing.T) {
	stmt := NodeMatch("Movie", EqualityWhere(map[string]any{}), nil)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE for empty where map: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
}

// TestNodeMatch_MultipleWhere verifies that multiple where keys produce AND-joined conditions.
// Expected: WHERE n.title = $title AND n.released = $released (or similar)
func TestNodeMatch_MultipleWhere(t *testing.T) {
	stmt := NodeMatch("Movie", EqualityWhere(map[string]any{"title": "Matrix", "released": 1999}), nil)

	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	if stmt.Params["title"] != "Matrix" {
		t.Errorf("Params[\"title\"] = %v, want %q", stmt.Params["title"], "Matrix")
	}
	if stmt.Params["released"] != 1999 {
		t.Errorf("Params[\"released\"] = %v, want %d", stmt.Params["released"], 1999)
	}
}

// --- NodeUpdate tests ---

// TestNodeUpdate_WithWhereAndSet verifies NodeUpdate produces MATCH + WHERE + SET + RETURN.
// Expected: MATCH (n:Movie) WHERE n.title = $where_title SET n.released = $set_released RETURN n
func TestNodeUpdate_WithWhereAndSet(t *testing.T) {
	stmt := NodeUpdate("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), map[string]any{"released": 2000})

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
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
	if strings.Contains(stmt.Query, "Matrix") {
		t.Errorf("Query contains interpolated value: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestNodeUpdate_EmptyWhere verifies NodeUpdate with empty where omits WHERE.
// Expected: MATCH (n:Movie) SET n.released = $set_released RETURN n
func TestNodeUpdate_EmptyWhere(t *testing.T) {
	stmt := NodeUpdate("Movie", EqualityWhere(map[string]any{}), map[string]any{"released": 2000})

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE for empty where map: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query missing SET: %q", stmt.Query)
	}
}

// TestNodeUpdate_ParamNoCollision verifies that where and set params don't collide.
// When both where and set reference the same field name (e.g. "title"),
// they must use prefixed param names to avoid collision.
func TestNodeUpdate_ParamNoCollision(t *testing.T) {
	stmt := NodeUpdate("Movie",
		EqualityWhere(map[string]any{"title": "Matrix"}),
		map[string]any{"title": "The Matrix"},
	)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// There should be two distinct params for "title" — one for where, one for set.
	// The exact naming scheme (where_title, set_title) is up to the implementation,
	// but both values must be present in Params.
	foundOld := false
	foundNew := false
	for _, v := range stmt.Params {
		if v == "Matrix" {
			foundOld = true
		}
		if v == "The Matrix" {
			foundNew = true
		}
	}
	if !foundOld {
		t.Error("Params missing where value \"Matrix\"")
	}
	if !foundNew {
		t.Error("Params missing set value \"The Matrix\"")
	}
}

// --- NodeDelete tests ---

// TestNodeDelete_WithWhere verifies NodeDelete produces MATCH + WHERE + DETACH DELETE.
// Expected: MATCH (n:Movie) WHERE n.title = $title DETACH DELETE n
func TestNodeDelete_WithWhere(t *testing.T) {
	stmt := NodeDelete("Movie", EqualityWhere(map[string]any{"title": "Matrix"}))

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DETACH DELETE") {
		t.Errorf("Query missing DETACH DELETE: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "Matrix") {
		t.Errorf("Query contains interpolated value: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	if stmt.Params["title"] != "Matrix" {
		t.Errorf("Params[\"title\"] = %v, want %q", stmt.Params["title"], "Matrix")
	}
}

// TestNodeDelete_EmptyWhere verifies NodeDelete with empty where omits WHERE.
// Expected: MATCH (n:Movie) DETACH DELETE n
func TestNodeDelete_EmptyWhere(t *testing.T) {
	stmt := NodeDelete("Movie", EqualityWhere(map[string]any{}))

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DETACH DELETE") {
		t.Errorf("Query missing DETACH DELETE: %q", stmt.Query)
	}
}

// --- CY-7: Sort behavior on NodeMatch ---

// TestNodeMatch_NilOrderBy verifies NodeMatch with nil orderBy produces no ORDER BY clause.
// Expected: MATCH (n:Movie) WHERE ... RETURN n — no ORDER BY.
func TestNodeMatch_NilOrderBy(t *testing.T) {
	stmt := NodeMatch("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), nil)

	if strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("NodeMatch with nil orderBy should not have ORDER BY: %q", stmt.Query)
	}
}

// TestNodeMatch_WithOrderBy verifies NodeMatch with sort fields produces ORDER BY.
// Expected: MATCH (n:Movie) WHERE ... RETURN n ORDER BY n.title ASC
func TestNodeMatch_WithOrderBy(t *testing.T) {
	stmt := NodeMatch("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), []SortField{
		{Field: "title", Direction: SortASC},
	})

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("NodeMatch with orderBy should have ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ASC") {
		t.Errorf("ORDER BY should include ASC: %q", stmt.Query)
	}
}

// TestNodeMatch_MultiFieldOrderBy verifies NodeMatch with multiple sort fields.
// Expected: ORDER BY n.title ASC, n.released DESC
func TestNodeMatch_MultiFieldOrderBy(t *testing.T) {
	stmt := NodeMatch("Movie", EqualityWhere(map[string]any{}), []SortField{
		{Field: "title", Direction: SortASC},
		{Field: "released", Direction: SortDESC},
	})

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("NodeMatch with orderBy should have ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ASC") {
		t.Errorf("ORDER BY should include ASC: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DESC") {
		t.Errorf("ORDER BY should include DESC: %q", stmt.Query)
	}
}

// --- Parameterization safety test ---

// TestNodeCRUD_NoInterpolation verifies that none of the CRUD helpers interpolate
// user values into the Query string. All values must appear only as $param references.
func TestNodeCRUD_NoInterpolation(t *testing.T) {
	tests := []struct {
		name string
		stmt Statement
	}{
		{"NodeCreate", NodeCreate("Movie", map[string]any{"title": "Matrix"})},
		{"NodeMatch", NodeMatch("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), nil)},
		{"NodeUpdate", NodeUpdate("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), map[string]any{"released": 2000})},
		{"NodeDelete", NodeDelete("Movie", EqualityWhere(map[string]any{"title": "Matrix"}))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.stmt.Query, "Matrix") {
				t.Errorf("%s: Query contains interpolated value 'Matrix': %q", tt.name, tt.stmt.Query)
			}
			if strings.Contains(tt.stmt.Query, "2000") {
				t.Errorf("%s: Query contains interpolated value '2000': %q", tt.name, tt.stmt.Query)
			}
		})
	}
}
