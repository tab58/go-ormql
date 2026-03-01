package cypher

import (
	"strings"
	"testing"
)

// --- ConnectionQuery tests ---

// TestConnectionQuery_Full verifies ConnectionQuery with where, first, and offset > 0.
// Expected: MATCH (n:Movie) WHERE n.title = $title RETURN n ORDER BY n.id SKIP $offset LIMIT $first
// Params: {title: "Matrix", offset: 5, first: 10}
func TestConnectionQuery_Full(t *testing.T) {
	stmt := ConnectionQuery("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), nil, 10, 5)

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
	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("Query missing ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "SKIP") {
		t.Errorf("Query missing SKIP: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "LIMIT") {
		t.Errorf("Query missing LIMIT: %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, "Matrix") {
		t.Errorf("Query contains interpolated value 'Matrix': %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestConnectionQuery_EmptyWhere verifies ConnectionQuery with empty where omits WHERE.
// Expected: MATCH (n:Movie) RETURN n ORDER BY n.id LIMIT $first
func TestConnectionQuery_EmptyWhere(t *testing.T) {
	stmt := ConnectionQuery("Movie", EqualityWhere(map[string]any{}), nil, 10, 0)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE for empty where: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("Query missing ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "LIMIT") {
		t.Errorf("Query missing LIMIT: %q", stmt.Query)
	}
}

// TestConnectionQuery_OffsetZero verifies that offset 0 omits the SKIP clause.
// Expected: Query contains LIMIT but NOT SKIP.
func TestConnectionQuery_OffsetZero(t *testing.T) {
	stmt := ConnectionQuery("Movie", EqualityWhere(map[string]any{}), nil, 10, 0)

	if strings.Contains(stmt.Query, "SKIP") {
		t.Errorf("Query should not contain SKIP when offset is 0: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "LIMIT") {
		t.Errorf("Query missing LIMIT: %q", stmt.Query)
	}
}

// TestConnectionQuery_Params verifies that first and offset values appear in Params.
func TestConnectionQuery_Params(t *testing.T) {
	stmt := ConnectionQuery("Movie", EqualityWhere(map[string]any{"title": "Matrix"}), nil, 10, 5)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}

	// Check first/offset params exist (exact key names depend on implementation)
	foundFirst := false
	foundOffset := false
	for _, v := range stmt.Params {
		switch v {
		case 10:
			foundFirst = true
		case 5:
			foundOffset = true
		}
	}
	if !foundFirst {
		t.Error("Params missing first value (10)")
	}
	if !foundOffset {
		t.Error("Params missing offset value (5)")
	}

	// Where param should also be present
	foundTitle := false
	for _, v := range stmt.Params {
		if v == "Matrix" {
			foundTitle = true
		}
	}
	if !foundTitle {
		t.Error("Params missing where value \"Matrix\"")
	}
}

// --- CY-7: ConnectionQuery sort behavior ---

// TestConnectionQuery_NilOrderBy_DefaultsToIdASC verifies that ConnectionQuery with nil orderBy
// defaults to ORDER BY n.id ASC for stable cursor pagination.
func TestConnectionQuery_NilOrderBy_DefaultsToIdASC(t *testing.T) {
	stmt := ConnectionQuery("Movie", EqualityWhere(map[string]any{}), nil, 10, 0)

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("ConnectionQuery with nil orderBy should default to ORDER BY: %q", stmt.Query)
	}
	// Default sort is by id ASC
	if !strings.Contains(stmt.Query, "id") {
		t.Errorf("Default ORDER BY should reference 'id': %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ASC") {
		t.Errorf("Default ORDER BY should be ASC: %q", stmt.Query)
	}
}

// TestConnectionQuery_WithOrderBy verifies that ConnectionQuery with explicit sort fields uses them.
// Expected: ORDER BY n.title DESC (overriding default id ASC).
func TestConnectionQuery_WithOrderBy(t *testing.T) {
	stmt := ConnectionQuery("Movie", EqualityWhere(map[string]any{}), []SortField{
		{Field: "title", Direction: SortDESC},
	}, 10, 0)

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("ConnectionQuery with orderBy should have ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DESC") {
		t.Errorf("ORDER BY should include DESC: %q", stmt.Query)
	}
}

// TestConnectionQuery_EmptyOrderBy_DefaultsToIdASC verifies that ConnectionQuery with empty
// (non-nil but zero-length) orderBy still defaults to ORDER BY n.id ASC.
func TestConnectionQuery_EmptyOrderBy_DefaultsToIdASC(t *testing.T) {
	stmt := ConnectionQuery("Movie", EqualityWhere(map[string]any{}), []SortField{}, 10, 0)

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("ConnectionQuery with empty orderBy should default to ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ASC") {
		t.Errorf("Default ORDER BY should be ASC: %q", stmt.Query)
	}
}

// --- ConnectionCount tests ---

// TestConnectionCount_WithWhere verifies ConnectionCount with where produces count query.
// Expected: MATCH (n:Movie) WHERE n.title = $title RETURN count(n)
func TestConnectionCount_WithWhere(t *testing.T) {
	stmt := ConnectionCount("Movie", EqualityWhere(map[string]any{"title": "Matrix"}))

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "count(n)") {
		t.Errorf("Query missing count(n): %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, "Matrix") {
		t.Errorf("Query contains interpolated value 'Matrix': %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// Where param should be present
	foundTitle := false
	for _, v := range stmt.Params {
		if v == "Matrix" {
			foundTitle = true
		}
	}
	if !foundTitle {
		t.Error("Params missing where value \"Matrix\"")
	}
}

// TestConnectionCount_EmptyWhere verifies ConnectionCount with empty where omits WHERE.
// Expected: MATCH (n:Movie) RETURN count(n) — no WHERE.
func TestConnectionCount_EmptyWhere(t *testing.T) {
	stmt := ConnectionCount("Movie", EqualityWhere(map[string]any{}))

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE for empty where: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "count(n)") {
		t.Errorf("Query missing count(n): %q", stmt.Query)
	}
}

// TestConnectionCount_NoOrderOrPagination verifies ConnectionCount does NOT include
// ORDER BY, SKIP, or LIMIT — it's a simple count.
func TestConnectionCount_NoOrderOrPagination(t *testing.T) {
	stmt := ConnectionCount("Movie", EqualityWhere(map[string]any{"title": "Matrix"}))

	if strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("count query should not contain ORDER BY: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "SKIP") {
		t.Errorf("count query should not contain SKIP: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "LIMIT") {
		t.Errorf("count query should not contain LIMIT: %q", stmt.Query)
	}
}
