package codegen

import (
	"strings"
	"testing"
)

// === CG-29: Target type + dialect-aware GenerateIndexes tests ===

// --- Target type ---

// Test: TargetNeo4j and TargetFalkorDB constants exist and have expected values.
// Expected: TargetNeo4j == "neo4j", TargetFalkorDB == "falkordb".
func TestTargetConstants(t *testing.T) {
	if TargetNeo4j != "neo4j" {
		t.Errorf("TargetNeo4j = %q, want %q", TargetNeo4j, "neo4j")
	}
	if TargetFalkorDB != "falkordb" {
		t.Errorf("TargetFalkorDB = %q, want %q", TargetFalkorDB, "falkordb")
	}
}

// --- Target validation ---

// Test: Empty target defaults to TargetNeo4j.
// Expected: validateTarget("") returns TargetNeo4j, nil.
func TestValidateTarget_EmptyDefaultsToNeo4j(t *testing.T) {
	got, err := validateTarget("")
	if err != nil {
		t.Fatalf("validateTarget(\"\") returned error: %v", err)
	}
	if got != TargetNeo4j {
		t.Errorf("validateTarget(\"\") = %q, want %q", got, TargetNeo4j)
	}
}

// Test: Valid targets pass through unchanged.
// Expected: validateTarget("neo4j") returns "neo4j", validateTarget("falkordb") returns "falkordb".
func TestValidateTarget_ValidTargets(t *testing.T) {
	tests := []struct {
		input string
		want  Target
	}{
		{"neo4j", TargetNeo4j},
		{"falkordb", TargetFalkorDB},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := validateTarget(Target(tt.input))
			if err != nil {
				t.Fatalf("validateTarget(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("validateTarget(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Test: Unknown target returns error.
// Expected: validateTarget("mysql") returns error.
func TestValidateTarget_Unknown(t *testing.T) {
	_, err := validateTarget("mysql")
	if err == nil {
		t.Fatal("validateTarget(\"mysql\") should return error for unknown target")
	}
}

// --- Neo4j DDL output (existing behavior preserved) ---

// Test: GenerateIndexes with TargetNeo4j produces existing DDL format.
// Expected: output contains "CREATE VECTOR INDEX movie_embeddings IF NOT EXISTS"
//   with "indexConfig" and "vector.dimensions".
func TestGenerateIndexes_Neo4jDDL(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated", TargetNeo4j)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil")
	}
	src := string(out)
	if !strings.Contains(src, "CREATE VECTOR INDEX") {
		t.Error("Neo4j DDL missing 'CREATE VECTOR INDEX'")
	}
	if !strings.Contains(src, "movie_embeddings") {
		t.Error("Neo4j DDL missing index name 'movie_embeddings'")
	}
	if !strings.Contains(src, "indexConfig") {
		t.Error("Neo4j DDL missing 'indexConfig' option key")
	}
	if !strings.Contains(src, "vector.dimensions") {
		t.Error("Neo4j DDL missing 'vector.dimensions'")
	}
}

// --- FalkorDB DDL output ---

// Test: GenerateIndexes with TargetFalkorDB produces FalkorDB-specific DDL.
// Expected: output contains "CREATE VECTOR INDEX FOR (n:Movie) ON (n.embedding)"
//   with "dimension" and "similarityFunction" (NOT "indexConfig" or "vector.dimensions").
//   FalkorDB DDL does NOT include an index name before FOR.
func TestGenerateIndexes_FalkorDBDDL(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated", TargetFalkorDB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil")
	}
	src := string(out)
	if !strings.Contains(src, "CREATE VECTOR INDEX FOR") {
		t.Error("FalkorDB DDL missing 'CREATE VECTOR INDEX FOR'")
	}
	if !strings.Contains(src, "dimension") {
		t.Error("FalkorDB DDL missing 'dimension' option key")
	}
	if !strings.Contains(src, "similarityFunction") {
		t.Error("FalkorDB DDL missing 'similarityFunction' option key")
	}
	// FalkorDB DDL should NOT use Neo4j-specific options
	if strings.Contains(src, "indexConfig") {
		t.Error("FalkorDB DDL should NOT contain 'indexConfig' (Neo4j-specific)")
	}
	if strings.Contains(src, "vector.dimensions") {
		t.Error("FalkorDB DDL should NOT contain 'vector.dimensions' (Neo4j-specific)")
	}
}

// --- FalkorDB VectorIndexes var ---

// Test: GenerateIndexes with TargetFalkorDB generates VectorIndexes var.
// Expected: output contains "var VectorIndexes = map[string]driver.VectorIndex{".
func TestGenerateIndexes_FalkorDBVectorIndexesVar(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated", TargetFalkorDB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil")
	}
	src := string(out)
	if !strings.Contains(src, "VectorIndexes") {
		t.Error("FalkorDB output missing 'VectorIndexes' var")
	}
	if !strings.Contains(src, "driver.VectorIndex") {
		t.Error("FalkorDB output missing 'driver.VectorIndex' type reference")
	}
}

// Test: GenerateIndexes with TargetNeo4j does NOT generate VectorIndexes var.
// Expected: output does NOT contain "VectorIndexes =".
func TestGenerateIndexes_Neo4jNoVectorIndexesVar(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated", TargetNeo4j)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil")
	}
	src := string(out)
	if strings.Contains(src, "VectorIndexes =") {
		t.Error("Neo4j output should NOT contain 'VectorIndexes =' var")
	}
}

// --- Target-specific warning ---

// Test: vectorWarningForTarget returns correct warning for each target.
// Expected: Neo4j → "Neo4j 5.11+", FalkorDB → "FalkorDB 4.2+".
func TestVectorWarningForTarget(t *testing.T) {
	neo4jWarn := vectorWarningForTarget(TargetNeo4j)
	if !strings.Contains(neo4jWarn, "Neo4j 5.11") {
		t.Errorf("Neo4j warning should mention 'Neo4j 5.11', got: %q", neo4jWarn)
	}

	fkWarn := vectorWarningForTarget(TargetFalkorDB)
	if !strings.Contains(fkWarn, "FalkorDB 4.2") {
		t.Errorf("FalkorDB warning should mention 'FalkorDB 4.2', got: %q", fkWarn)
	}
}

// --- Config.Target field ---

// Test: Config struct has Target field.
// Expected: Config{Target: TargetFalkorDB} compiles.
func TestConfig_HasTargetField(t *testing.T) {
	cfg := Config{
		SchemaFiles: []string{"schema.graphql"},
		OutputDir:   "/tmp",
		Target:      TargetFalkorDB,
	}
	if cfg.Target != TargetFalkorDB {
		t.Errorf("Target = %q, want %q", cfg.Target, TargetFalkorDB)
	}
}
