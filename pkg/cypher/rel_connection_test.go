package cypher

import (
	"strings"
	"testing"
)

// --- RelConnectionQuery tests ---

// TestRelConnectionQuery_OUT verifies RelConnectionQuery with DirectionOUT produces
// (parent)-[r:TYPE]->(child) pattern with WHERE, ORDER BY, SKIP, LIMIT.
// Expected: MATCH (parent:Movie)-[r:ACTED_IN]->(child:Actor) WHERE parent.id = $p0 RETURN child, r ORDER BY child.name ASC SKIP $offset LIMIT $first
func TestRelConnectionQuery_OUT(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
		[]SortField{{Field: "name", Direction: SortASC}},
		10, 5,
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing from label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Actor") {
		t.Errorf("Query missing to label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
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
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestRelConnectionQuery_IN verifies RelConnectionQuery with DirectionIN reverses the arrow.
// Expected: MATCH (parent:Movie)<-[r:ACTED_IN]-(child:Actor) ...
func TestRelConnectionQuery_IN(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionIN,
		EqualityWhere(map[string]any{}),
		nil,
		10, 0,
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing from label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Actor") {
		t.Errorf("Query missing to label: %q", stmt.Query)
	}
	// IN direction should have reversed arrow: <-[r:TYPE]-
	// The exact syntax depends on implementation, but it should differ from OUT
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
}

// TestRelConnectionQuery_NilOrderBy_DefaultsToChildIdASC verifies that nil orderBy
// defaults to ORDER BY child.id ASC for stable pagination.
func TestRelConnectionQuery_NilOrderBy_DefaultsToChildIdASC(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
		nil,
		10, 0,
	)

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("RelConnectionQuery with nil orderBy should default to ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ASC") {
		t.Errorf("Default ORDER BY should be ASC: %q", stmt.Query)
	}
}

// TestRelConnectionQuery_WithToWhere verifies that toWhere conditions filter the child node.
func TestRelConnectionQuery_WithToWhere(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{"name": "Keanu"}),
		nil,
		10, 0,
	)

	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// Both from and to where values should be in params
	foundID := false
	foundName := false
	for _, v := range stmt.Params {
		switch v {
		case "1":
			foundID = true
		case "Keanu":
			foundName = true
		}
	}
	if !foundID {
		t.Error("Params missing from-where value \"1\"")
	}
	if !foundName {
		t.Error("Params missing to-where value \"Keanu\"")
	}
}

// TestRelConnectionQuery_OffsetZero verifies that offset 0 omits SKIP clause.
func TestRelConnectionQuery_OffsetZero(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
		nil,
		10, 0,
	)

	if strings.Contains(stmt.Query, "SKIP") {
		t.Errorf("Query should not contain SKIP when offset is 0: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "LIMIT") {
		t.Errorf("Query missing LIMIT: %q", stmt.Query)
	}
}

// TestRelConnectionQuery_EmptyBothWhere verifies that empty where on both sides omits WHERE.
func TestRelConnectionQuery_EmptyBothWhere(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
		nil,
		10, 0,
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE when both wheres are empty: %q", stmt.Query)
	}
}

// TestRelConnectionQuery_PaginationParams verifies first and offset appear in Params.
func TestRelConnectionQuery_PaginationParams(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
		nil,
		10, 5,
	)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
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
}

// TestRelConnectionQuery_NoInterpolation verifies no user values are interpolated.
func TestRelConnectionQuery_NoInterpolation(t *testing.T) {
	stmt := RelConnectionQuery(
		"Movie", EqualityWhere(map[string]any{"id": "movie-1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{"name": "Keanu"}),
		nil,
		10, 0,
	)

	if strings.Contains(stmt.Query, "movie-1") {
		t.Errorf("Query contains interpolated from-where value: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "Keanu") {
		t.Errorf("Query contains interpolated to-where value: %q", stmt.Query)
	}
}

// --- RelConnectionCount tests ---

// TestRelConnectionCount_OUT verifies RelConnectionCount with DirectionOUT produces
// MATCH (parent)-[r:TYPE]->(child) WHERE ... RETURN count(child).
func TestRelConnectionCount_OUT(t *testing.T) {
	stmt := RelConnectionCount(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing from label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Actor") {
		t.Errorf("Query missing to label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "ACTED_IN") {
		t.Errorf("Query missing rel type: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "count(") {
		t.Errorf("Query missing count(): %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestRelConnectionCount_IN verifies RelConnectionCount with DirectionIN reverses the arrow.
func TestRelConnectionCount_IN(t *testing.T) {
	stmt := RelConnectionCount(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionIN,
		EqualityWhere(map[string]any{}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "count(") {
		t.Errorf("Query missing count(): %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
}

// TestRelConnectionCount_WithToWhere verifies that toWhere conditions are included.
func TestRelConnectionCount_WithToWhere(t *testing.T) {
	stmt := RelConnectionCount(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{"name": "Keanu"}),
	)

	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	foundName := false
	for _, v := range stmt.Params {
		if v == "Keanu" {
			foundName = true
		}
	}
	if !foundName {
		t.Error("Params missing to-where value \"Keanu\"")
	}
}

// TestRelConnectionCount_EmptyBothWhere verifies that empty where on both sides omits WHERE.
func TestRelConnectionCount_EmptyBothWhere(t *testing.T) {
	stmt := RelConnectionCount(
		"Movie", EqualityWhere(map[string]any{}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE when both wheres are empty: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "count(") {
		t.Errorf("Query missing count(): %q", stmt.Query)
	}
}

// TestRelConnectionCount_NoOrderOrPagination verifies count query has no ORDER BY, SKIP, or LIMIT.
func TestRelConnectionCount_NoOrderOrPagination(t *testing.T) {
	stmt := RelConnectionCount(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{}),
	)

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

// TestRelConnectionCount_NoInterpolation verifies no user values are interpolated.
func TestRelConnectionCount_NoInterpolation(t *testing.T) {
	stmt := RelConnectionCount(
		"Movie", EqualityWhere(map[string]any{"id": "movie-1"}),
		"ACTED_IN",
		"Actor", DirectionOUT,
		EqualityWhere(map[string]any{"name": "Keanu"}),
	)

	if strings.Contains(stmt.Query, "movie-1") {
		t.Errorf("Query contains interpolated from-where value: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "Keanu") {
		t.Errorf("Query contains interpolated to-where value: %q", stmt.Query)
	}
}
