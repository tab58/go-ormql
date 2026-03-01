package cypher

import (
	"strings"
	"testing"
)

// --- RelDisconnect tests ---

// TestRelDisconnect_Basic verifies RelDisconnect produces MATCH relationship pattern + WHERE + DELETE r.
// Expected: MATCH (a:Movie)-[r:ACTED_IN]->(b:Actor) WHERE a.id = $p0 AND b.name = $p1 DELETE r
// Params: {p0: "1", p1: "Old Actor"}
func TestRelDisconnect_Basic(t *testing.T) {
	stmt := RelDisconnect(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Old Actor"}),
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
	if !strings.Contains(stmt.Query, "DELETE") {
		t.Errorf("Query missing DELETE: %q", stmt.Query)
	}
	// Should NOT contain DETACH DELETE — only deletes the relationship, not the node
	if strings.Contains(stmt.Query, "DETACH DELETE") {
		t.Errorf("RelDisconnect should use DELETE r (not DETACH DELETE): %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, "Old Actor") || strings.Contains(stmt.Query, `"1"`) {
		t.Errorf("Query contains interpolated values: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestRelDisconnect_EmptyFromWhere verifies RelDisconnect with empty fromWhere only
// filters on the toWhere conditions.
func TestRelDisconnect_EmptyFromWhere(t *testing.T) {
	stmt := RelDisconnect(
		"Movie", EqualityWhere(map[string]any{}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Old Actor"}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DELETE") {
		t.Errorf("Query missing DELETE: %q", stmt.Query)
	}
}

// TestRelDisconnect_EmptyBothWhere verifies RelDisconnect with empty where on both sides
// omits WHERE entirely.
func TestRelDisconnect_EmptyBothWhere(t *testing.T) {
	stmt := RelDisconnect(
		"Movie", EqualityWhere(map[string]any{}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE when both wheres are empty: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DELETE") {
		t.Errorf("Query missing DELETE: %q", stmt.Query)
	}
}

// TestRelDisconnect_ParamsPopulated verifies that all where values appear in Params.
func TestRelDisconnect_ParamsPopulated(t *testing.T) {
	stmt := RelDisconnect(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Old Actor"}),
	)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	foundID := false
	foundName := false
	for _, v := range stmt.Params {
		switch v {
		case "1":
			foundID = true
		case "Old Actor":
			foundName = true
		}
	}
	if !foundID {
		t.Error("Params missing from-where value \"1\"")
	}
	if !foundName {
		t.Error("Params missing to-where value \"Old Actor\"")
	}
}

// TestRelDisconnect_NoInterpolation verifies no user values are interpolated in the query.
func TestRelDisconnect_NoInterpolation(t *testing.T) {
	stmt := RelDisconnect(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
	)

	if strings.Contains(stmt.Query, "Keanu") {
		t.Errorf("Query contains interpolated value 'Keanu': %q", stmt.Query)
	}
}

// --- NestedUpdate tests ---

// TestNestedUpdate_NodeAndEdge verifies NestedUpdate with both nodeSet and edgeSet
// produces MATCH + WHERE + SET (node + edge) + RETURN b, r.
// Expected: MATCH (a:Movie)-[r:ACTED_IN]->(b:Actor) WHERE ... SET b.name = $set_name, r.role = $edge_role RETURN b, r
func TestNestedUpdate_NodeAndEdge(t *testing.T) {
	stmt := NestedUpdate(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		map[string]any{"name": "Keanu Reeves"},
		map[string]any{"role": "Neo"},
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
	if !strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query missing SET: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, "Keanu") || strings.Contains(stmt.Query, "Neo") {
		t.Errorf("Query contains interpolated values: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestNestedUpdate_NodeSetOnly verifies NestedUpdate with nil edgeSet only SETs node properties.
func TestNestedUpdate_NodeSetOnly(t *testing.T) {
	stmt := NestedUpdate(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		map[string]any{"name": "Keanu Reeves"},
		nil,
	)

	if !strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query missing SET for node update: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// Node set value should be in params
	found := false
	for _, v := range stmt.Params {
		if v == "Keanu Reeves" {
			found = true
		}
	}
	if !found {
		t.Error("Params missing node set value \"Keanu Reeves\"")
	}
}

// TestNestedUpdate_EdgeSetOnly verifies NestedUpdate with nil nodeSet only SETs edge properties.
func TestNestedUpdate_EdgeSetOnly(t *testing.T) {
	stmt := NestedUpdate(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		nil,
		map[string]any{"role": "Neo"},
	)

	if !strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query missing SET for edge update: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	// Edge set value should be in params
	found := false
	for _, v := range stmt.Params {
		if v == "Neo" {
			found = true
		}
	}
	if !found {
		t.Error("Params missing edge set value \"Neo\"")
	}
}

// TestNestedUpdate_BothNilSets verifies NestedUpdate with both nil nodeSet and edgeSet
// produces no SET clause (still valid MATCH + RETURN pattern).
func TestNestedUpdate_BothNilSets(t *testing.T) {
	stmt := NestedUpdate(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		nil,
		nil,
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "SET") {
		t.Errorf("Query should not contain SET when both nodeSet and edgeSet are nil: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "RETURN") {
		t.Errorf("Query missing RETURN: %q", stmt.Query)
	}
}

// TestNestedUpdate_ParamsPopulated verifies all where + set values appear in Params.
func TestNestedUpdate_ParamsPopulated(t *testing.T) {
	stmt := NestedUpdate(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		map[string]any{"name": "Keanu Reeves"},
		map[string]any{"role": "Neo"},
	)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	expected := []any{"1", "Keanu", "Keanu Reeves", "Neo"}
	for _, want := range expected {
		found := false
		for _, v := range stmt.Params {
			if v == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Params missing value %v", want)
		}
	}
}

// TestNestedUpdate_NoInterpolation verifies no user values are interpolated.
func TestNestedUpdate_NoInterpolation(t *testing.T) {
	stmt := NestedUpdate(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
		map[string]any{"name": "Keanu Reeves"},
		map[string]any{"role": "Neo"},
	)

	for _, val := range []string{"Keanu", "Keanu Reeves", "Neo"} {
		if strings.Contains(stmt.Query, val) {
			t.Errorf("Query contains interpolated value %q: %q", val, stmt.Query)
		}
	}
}

// --- NestedDelete tests ---

// TestNestedDelete_Basic verifies NestedDelete produces MATCH relationship pattern +
// WHERE + DETACH DELETE b (detach-deletes the target node).
// Expected: MATCH (a:Movie)-[r:ACTED_IN]->(b:Actor) WHERE ... DETACH DELETE b
func TestNestedDelete_Basic(t *testing.T) {
	stmt := NestedDelete(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Old Actor"}),
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
	if !strings.Contains(stmt.Query, "DETACH DELETE") {
		t.Errorf("Query missing DETACH DELETE: %q", stmt.Query)
	}
	// No interpolated values
	if strings.Contains(stmt.Query, "Old Actor") {
		t.Errorf("Query contains interpolated values: %q", stmt.Query)
	}
	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
}

// TestNestedDelete_EmptyBothWhere verifies NestedDelete with empty where on both sides
// omits WHERE.
func TestNestedDelete_EmptyBothWhere(t *testing.T) {
	stmt := NestedDelete(
		"Movie", EqualityWhere(map[string]any{}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{}),
	)

	if !strings.Contains(stmt.Query, "MATCH") {
		t.Errorf("Query missing MATCH: %q", stmt.Query)
	}
	if strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("Query should not contain WHERE when both wheres are empty: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "DETACH DELETE") {
		t.Errorf("Query missing DETACH DELETE: %q", stmt.Query)
	}
}

// TestNestedDelete_ParamsPopulated verifies that all where values appear in Params.
func TestNestedDelete_ParamsPopulated(t *testing.T) {
	stmt := NestedDelete(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Old Actor"}),
	)

	if stmt.Params == nil {
		t.Fatal("Params is nil")
	}
	foundID := false
	foundName := false
	for _, v := range stmt.Params {
		switch v {
		case "1":
			foundID = true
		case "Old Actor":
			foundName = true
		}
	}
	if !foundID {
		t.Error("Params missing from-where value \"1\"")
	}
	if !foundName {
		t.Error("Params missing to-where value \"Old Actor\"")
	}
}

// TestNestedDelete_NoInterpolation verifies no user values are interpolated.
func TestNestedDelete_NoInterpolation(t *testing.T) {
	stmt := NestedDelete(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
	)

	if strings.Contains(stmt.Query, "Keanu") {
		t.Errorf("Query contains interpolated value 'Keanu': %q", stmt.Query)
	}
}

// --- Cross-helper distinction test ---

// TestNested_DisconnectVsDelete verifies the key behavioral difference:
// RelDisconnect uses DELETE r (relationship only), NestedDelete uses DETACH DELETE b (node + relationships).
func TestNested_DisconnectVsDelete(t *testing.T) {
	disconnect := RelDisconnect(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
	)
	delete := NestedDelete(
		"Movie", EqualityWhere(map[string]any{"id": "1"}),
		"ACTED_IN",
		"Actor", EqualityWhere(map[string]any{"name": "Keanu"}),
	)

	// Disconnect should use plain DELETE (not DETACH DELETE)
	if strings.Contains(disconnect.Query, "DETACH") {
		t.Errorf("RelDisconnect should use DELETE r, not DETACH DELETE: %q", disconnect.Query)
	}

	// Delete should use DETACH DELETE
	if !strings.Contains(delete.Query, "DETACH DELETE") {
		t.Errorf("NestedDelete should use DETACH DELETE: %q", delete.Query)
	}
}
