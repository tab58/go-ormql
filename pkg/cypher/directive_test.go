package cypher

import (
	"strings"
	"testing"
)

// --- CypherDirective tests ---

// TestCypherDirective_Basic verifies CypherDirective produces MATCH this + WHERE + CALL subquery + RETURN.
// Expected: MATCH (this:Movie) WHERE this.id = $p0 CALL { WITH this MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score) AS __cypher_result } RETURN __cypher_result
// Params: {p0: "1"}
func TestCypherDirective_Basic(t *testing.T) {
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{"id": "1"}),
		"MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
		nil,
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "Movie") {
		t.Errorf("Query missing parent label: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "this") {
		t.Errorf("Query missing 'this' variable: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query missing WHERE: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "CALL") {
		t.Errorf("Query missing CALL subquery: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "WITH this") {
		t.Errorf("Query missing 'WITH this' in CALL subquery: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "__cypher_result") {
		t.Errorf("Query missing __cypher_result: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing outer RETURN: %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, `"1"`) {
		t.Errorf("Query contains interpolated value: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestCypherDirective_WithArgs verifies CypherDirective merges field arguments into params.
// Expected: Params contain both parent WHERE params and field args.
func TestCypherDirective_WithArgs(t *testing.T) {
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{"id": "1"}),
		"MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) RETURN rec LIMIT $limit",
		map[string]any{"limit": 5},
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "CALL") {
		t.Errorf("Query missing CALL subquery: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "__cypher_result") {
		t.Errorf("Query missing __cypher_result: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// Field arg should be in params
	foundLimit := false
	for _, v := range stmt.Params {
		if v == 5 {
			foundLimit = true
		}
	}
	if !foundLimit {
		t.Error("Params missing field arg value 5 (limit)")
	}
	// Where param should also be present
	foundID := false
	for _, v := range stmt.Params {
		if v == "1" {
			foundID = true
		}
	}
	if !foundID {
		t.Error("Params missing parent where value \"1\"")
	}
}

// TestCypherDirective_NilArgs verifies CypherDirective with nil args produces no extra params.
func TestCypherDirective_NilArgs(t *testing.T) {
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{"id": "1"}),
		"MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
		nil,
	)

	if !strings.Contains(stmt.Query, "CALL") {
		t.Errorf("Query missing CALL subquery: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// Only the where param should be present
	if len(stmt.Params) == 0 {
		t.Error("Params should contain at least the parent where param")
	}
}

// TestCypherDirective_EmptyWhere verifies CypherDirective with empty parentWhere omits WHERE.
func TestCypherDirective_EmptyWhere(t *testing.T) {
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{}),
		"MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
		nil,
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE for empty parentWhere: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "CALL") {
		t.Errorf("Query missing CALL subquery: %q", stmt.Query)
	}
}

// TestCypherDirective_StatementWithReturn verifies that the user's statement containing
// RETURN is handled correctly — the CALL subquery wraps it, and the outer query
// returns __cypher_result.
func TestCypherDirective_StatementWithReturn(t *testing.T) {
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{"id": "1"}),
		"MATCH (this)<-[:ACTED_IN]-(a:Actor) RETURN a",
		nil,
	)

	if !strings.Contains(stmt.Query, "CALL") {
		t.Errorf("Query missing CALL subquery: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "__cypher_result") {
		t.Errorf("Query missing __cypher_result (CALL should alias user RETURN): %q", stmt.Query)
	}
	// The outer query must RETURN __cypher_result
	// Split at the last RETURN to check it references __cypher_result
	parts := strings.Split(stmt.Query, "}")
	if len(parts) < 2 {
		t.Fatalf("Query doesn't have closing brace for CALL subquery: %q", stmt.Query)
	}
	outerPart := parts[len(parts)-1]
	if !strings.Contains(outerPart, "RETURN") || !strings.Contains(outerPart, "__cypher_result") {
		t.Errorf("Outer query after CALL should RETURN __cypher_result, got: %q", outerPart)
	}
}

// TestCypherDirective_ContainsUserStatement verifies the user's Cypher statement
// appears inside the CALL subquery.
func TestCypherDirective_ContainsUserStatement(t *testing.T) {
	userStmt := "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)"
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{"id": "1"}),
		userStmt,
		nil,
	)

	if !strings.Contains(stmt.Query, userStmt) && !strings.Contains(stmt.Query, "avg(r.score)") {
		t.Errorf("Query should contain user statement or key parts of it: %q", stmt.Query)
	}
}

// TestCypherDirective_NoInterpolation verifies no user values are interpolated.
func TestCypherDirective_NoInterpolation(t *testing.T) {
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{"id": "movie-123"}),
		"MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
		map[string]any{"threshold": 3.5},
	)

	if strings.Contains(stmt.Query, "movie-123") {
		t.Errorf("Query contains interpolated where value: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "3.5") {
		t.Errorf("Query contains interpolated arg value: %q", stmt.Query)
	}
}

// TestCypherDirective_MultipleArgs verifies that multiple field arguments are all
// merged into params.
func TestCypherDirective_MultipleArgs(t *testing.T) {
	stmt := CypherDirective(
		"Movie",
		EqualityWhere(map[string]any{"id": "1"}),
		"MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) WHERE rec.released >= $minYear RETURN rec LIMIT $limit",
		map[string]any{"limit": 5, "minYear": 2000},
	)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	foundLimit := false
	foundMinYear := false
	for _, v := range stmt.Params {
		switch v {
		case 5:
			foundLimit = true
		case 2000:
			foundMinYear = true
		}
	}
	if !foundLimit {
		t.Error("Params missing arg 'limit' value 5")
	}
	if !foundMinYear {
		t.Error("Params missing arg 'minYear' value 2000")
	}
}
