package codegen

import "fmt"

// Target identifies the graph database backend for dialect-aware code generation.
type Target string

const (
	// TargetNeo4j generates Neo4j-specific DDL and index format.
	TargetNeo4j Target = "neo4j"
	// TargetFalkorDB generates FalkorDB-specific DDL and VectorIndexes var.
	TargetFalkorDB Target = "falkordb"
)

// validTargets is the set of accepted target values.
var validTargets = map[Target]bool{
	TargetNeo4j:    true,
	TargetFalkorDB: true,
}

// validateTarget validates and normalizes a Target value.
// Empty string defaults to TargetNeo4j. Unknown values return an error.
func validateTarget(t Target) (Target, error) {
	if t == "" {
		return TargetNeo4j, nil
	}
	if !validTargets[t] {
		return "", fmt.Errorf("unsupported target %q (valid: neo4j, falkordb)", t)
	}
	return t, nil
}

// vectorWarningForTarget returns the target-specific @vector warning message.
func vectorWarningForTarget(t Target) string {
	switch t {
	case TargetFalkorDB:
		return "Warning: @vector directive requires FalkorDB 4.2+"
	default:
		return "Warning: @vector directive requires Neo4j 5.11+"
	}
}
