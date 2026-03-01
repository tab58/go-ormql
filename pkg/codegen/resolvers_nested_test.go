package codegen

import (
	"strings"
	"testing"
)

// --- CG-17: Nested disconnect/update/delete resolver templates ---

// TestGenerateResolvers_UpdateMutation_UsesBeginTx verifies that the update
// mutation resolver for a node with relationships uses BeginTx for transactional
// nested mutation processing (disconnect/update/delete ops require a transaction).
// Expected: generated source for UpdateMovies contains "BeginTx".
func TestGenerateResolvers_UpdateMutation_UsesBeginTx(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// Find the UpdateMovies function and check it uses BeginTx
	idx := strings.Index(s, "UpdateMovies")
	if idx == -1 {
		t.Fatal("generated resolvers missing UpdateMovies function")
	}
	updateSection := s[idx:]
	// Limit to next top-level function (look for next "func (r *")
	nextFunc := strings.Index(updateSection[1:], "\nfunc ")
	if nextFunc > 0 {
		updateSection = updateSection[:nextFunc+1]
	}
	if !strings.Contains(updateSection, "BeginTx") {
		t.Errorf("UpdateMovies resolver should use BeginTx for nested ops:\n%s", updateSection)
	}
}

// TestGenerateResolvers_UpdateMutation_CallsRelDisconnect verifies that the
// update mutation resolver calls cypher.RelDisconnect for disconnect operations.
// Expected: generated source contains "cypher.RelDisconnect".
func TestGenerateResolvers_UpdateMutation_CallsRelDisconnect(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.RelDisconnect") && !strings.Contains(s, "RelDisconnect") {
		t.Errorf("update resolver should call cypher.RelDisconnect for disconnect ops:\n%s", s)
	}
}

// TestGenerateResolvers_UpdateMutation_CallsNestedUpdate verifies that the
// update mutation resolver calls cypher.NestedUpdate for nested update operations.
// Expected: generated source contains "cypher.NestedUpdate".
func TestGenerateResolvers_UpdateMutation_CallsNestedUpdate(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.NestedUpdate") && !strings.Contains(s, "NestedUpdate") {
		t.Errorf("update resolver should call cypher.NestedUpdate for nested update ops:\n%s", s)
	}
}

// TestGenerateResolvers_UpdateMutation_CallsNestedDelete verifies that the
// update mutation resolver calls cypher.NestedDelete for nested delete operations.
// Expected: generated source contains "cypher.NestedDelete".
func TestGenerateResolvers_UpdateMutation_CallsNestedDelete(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.NestedDelete") && !strings.Contains(s, "NestedDelete") {
		t.Errorf("update resolver should call cypher.NestedDelete for nested delete ops:\n%s", s)
	}
}

// TestGenerateResolvers_UpdateMutation_HandlesDisconnectField verifies that
// the update mutation resolver accesses the disconnect field from the
// UpdateFieldInput (e.g., update.Actors.Disconnect).
// Expected: generated source references ".Disconnect" on the relationship field.
func TestGenerateResolvers_UpdateMutation_HandlesDisconnectField(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, ".Disconnect") {
		t.Errorf("update resolver should access .Disconnect field on UpdateFieldInput:\n%s", s)
	}
}

// TestGenerateResolvers_UpdateMutation_HandlesUpdateField verifies that
// the update mutation resolver accesses the update field from the
// UpdateFieldInput (e.g., update.Actors.Update).
// Expected: generated source references ".Update" on the relationship field.
func TestGenerateResolvers_UpdateMutation_HandlesUpdateField(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The update resolver should iterate over .Update entries
	if !strings.Contains(s, ".Update") {
		t.Errorf("update resolver should access .Update field on UpdateFieldInput:\n%s", s)
	}
}

// TestGenerateResolvers_UpdateMutation_HandlesDeleteField verifies that
// the update mutation resolver accesses the delete field from the
// UpdateFieldInput (e.g., update.Actors.Delete).
// Expected: generated source references ".Delete" on the relationship field.
func TestGenerateResolvers_UpdateMutation_HandlesDeleteField(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, ".Delete") {
		t.Errorf("update resolver should access .Delete field on UpdateFieldInput:\n%s", s)
	}
}

// TestGenerateResolvers_UpdateMutation_AllFiveOpsInTransaction verifies that
// all 5 nested ops (create, connect, disconnect, update, delete) execute
// within a single transaction in the update mutation resolver.
// Expected: BeginTx → create/connect/disconnect/update/delete → Commit all in same function.
func TestGenerateResolvers_UpdateMutation_AllFiveOpsInTransaction(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// Find the UpdateMovies function section
	idx := strings.Index(s, "UpdateMovies")
	if idx == -1 {
		t.Fatal("generated resolvers missing UpdateMovies function")
	}
	updateSection := s[idx:]
	nextFunc := strings.Index(updateSection[1:], "\nfunc ")
	if nextFunc > 0 {
		updateSection = updateSection[:nextFunc+1]
	}

	// All 5 ops should be referenced within the same function
	for _, keyword := range []string{".Create", ".Connect", ".Disconnect", ".Update", ".Delete"} {
		if !strings.Contains(updateSection, keyword) {
			t.Errorf("UpdateMovies resolver missing %s operation in transaction:\n%s", keyword, updateSection)
		}
	}
}

// TestGenerateResolvers_UpdateMutation_CommitsTransaction verifies that
// the update mutation resolver commits the transaction after all nested ops.
// Expected: generated source for UpdateMovies contains "Commit".
func TestGenerateResolvers_UpdateMutation_CommitsTransaction(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	idx := strings.Index(s, "UpdateMovies")
	if idx == -1 {
		t.Fatal("generated resolvers missing UpdateMovies function")
	}
	updateSection := s[idx:]
	nextFunc := strings.Index(updateSection[1:], "\nfunc ")
	if nextFunc > 0 {
		updateSection = updateSection[:nextFunc+1]
	}
	if !strings.Contains(updateSection, "Commit") {
		t.Errorf("UpdateMovies resolver should commit transaction:\n%s", updateSection)
	}
}

// TestGenerateResolvers_UpdateMutation_DefersRollback verifies that
// the update mutation resolver defers rollback for cleanup on error.
// Expected: generated source for UpdateMovies contains "defer" and "Rollback".
func TestGenerateResolvers_UpdateMutation_DefersRollback(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	idx := strings.Index(s, "UpdateMovies")
	if idx == -1 {
		t.Fatal("generated resolvers missing UpdateMovies function")
	}
	updateSection := s[idx:]
	nextFunc := strings.Index(updateSection[1:], "\nfunc ")
	if nextFunc > 0 {
		updateSection = updateSection[:nextFunc+1]
	}
	if !strings.Contains(updateSection, "defer") || !strings.Contains(updateSection, "Rollback") {
		t.Errorf("UpdateMovies resolver should defer Rollback:\n%s", updateSection)
	}
}

// TestGenerateResolvers_UpdateMutation_WithoutRels_NoNestedOps verifies that
// the update mutation resolver for a node WITHOUT relationships does NOT
// generate nested disconnect/update/delete operations.
// Expected: no cypher.RelDisconnect/NestedUpdate/NestedDelete in output.
func TestGenerateResolvers_UpdateMutation_WithoutRels_NoNestedOps(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if strings.Contains(s, "RelDisconnect") {
		t.Errorf("update resolver for node without relationships should NOT call RelDisconnect:\n%s", s)
	}
	if strings.Contains(s, "NestedUpdate") {
		t.Errorf("update resolver for node without relationships should NOT call NestedUpdate:\n%s", s)
	}
	if strings.Contains(s, "NestedDelete") {
		t.Errorf("update resolver for node without relationships should NOT call NestedDelete:\n%s", s)
	}
}

// TestGenerateResolvers_UpdateMutation_EdgeProperties_HandledInNestedUpdate verifies
// that when @relationshipProperties exists, the nested update operation handles
// both node properties and edge properties.
// Expected: generated source references edge property handling (e.g., "edgeSet" or "Edge").
func TestGenerateResolvers_UpdateMutation_EdgeProperties_HandledInNestedUpdate(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// NestedUpdate takes both nodeSet and edgeSet — the template should handle edge properties
	if !strings.Contains(s, "Edge") && !strings.Contains(s, "edge") {
		t.Errorf("update resolver with @relationshipProperties should handle edge properties:\n%s", s)
	}
}
