package cypher

// Statement represents a parameterized Cypher query ready for execution.
// Immutable after construction.
type Statement struct {
	Query  string
	Params map[string]any
}
