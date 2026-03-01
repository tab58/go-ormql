package cypher

import (
	"strings"
	"testing"
)

// --- SortDirection constant tests ---

// TestSortDirection_Constants verifies ASC and DESC constants.
// Expected: SortASC="ASC", SortDESC="DESC".
func TestSortDirection_Constants(t *testing.T) {
	if string(SortASC) != "ASC" {
		t.Errorf("SortASC = %q, want %q", string(SortASC), "ASC")
	}
	if string(SortDESC) != "DESC" {
		t.Errorf("SortDESC = %q, want %q", string(SortDESC), "DESC")
	}
}

// --- Builder.OrderBy with SortField tests ---

// TestBuilder_OrderBySortField_Single verifies OrderBy with a single SortField.
// Expected: Query contains "ORDER BY n.title ASC".
func TestBuilder_OrderBySortField_Single(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Return("n").
		OrderBy(SortField{Field: "n.title", Direction: SortASC}).
		Build()

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("Query missing ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.title ASC") {
		t.Errorf("Query missing 'n.title ASC': %q", stmt.Query)
	}
}

// TestBuilder_OrderBySortField_MultiField verifies OrderBy with multiple SortFields.
// Expected: Query contains "ORDER BY n.title ASC, n.released DESC" as a single clause.
func TestBuilder_OrderBySortField_MultiField(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Return("n").
		OrderBy(
			SortField{Field: "n.title", Direction: SortASC},
			SortField{Field: "n.released", Direction: SortDESC},
		).
		Build()

	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("Query missing ORDER BY: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.title ASC") {
		t.Errorf("Query missing 'n.title ASC': %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.released DESC") {
		t.Errorf("Query missing 'n.released DESC': %q", stmt.Query)
	}
	// Should be comma-separated in a single ORDER BY clause
	if !strings.Contains(stmt.Query, ", ") {
		t.Errorf("Multi-field ORDER BY should be comma-separated: %q", stmt.Query)
	}
}

// TestBuilder_OrderBySortField_Empty verifies OrderBy with no SortFields produces no ORDER BY.
// Expected: Query does NOT contain "ORDER BY".
func TestBuilder_OrderBySortField_Empty(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Return("n").
		OrderBy().
		Build()

	if strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("Empty OrderBy should produce no ORDER BY: %q", stmt.Query)
	}
}

// TestBuilder_OrderBySortField_Immutability verifies builder immutability is preserved.
// Expected: original builder without OrderBy is not affected.
func TestBuilder_OrderBySortField_Immutability(t *testing.T) {
	b1 := New().Match("n", "Movie").Return("n")
	b2 := b1.OrderBy(SortField{Field: "n.title", Direction: SortASC})

	s1 := b1.Build()
	s2 := b2.Build()

	if strings.Contains(s1.Query, "ORDER BY") {
		t.Errorf("Original builder was mutated: %q", s1.Query)
	}
	if !strings.Contains(s2.Query, "ORDER BY") {
		t.Errorf("Derived builder missing ORDER BY: %q", s2.Query)
	}
}

// TestBuilder_OrderBySortField_ThreeFields verifies three-field sorting.
// Expected: Query contains all three fields in correct order.
func TestBuilder_OrderBySortField_ThreeFields(t *testing.T) {
	stmt := New().
		Match("n", "Movie").
		Return("n").
		OrderBy(
			SortField{Field: "n.title", Direction: SortASC},
			SortField{Field: "n.released", Direction: SortDESC},
			SortField{Field: "n.rating", Direction: SortASC},
		).
		Build()

	if !strings.Contains(stmt.Query, "n.title ASC") {
		t.Errorf("Query missing first sort field: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.released DESC") {
		t.Errorf("Query missing second sort field: %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.rating ASC") {
		t.Errorf("Query missing third sort field: %q", stmt.Query)
	}
}
