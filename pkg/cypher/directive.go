package cypher

import "fmt"

// CypherDirective produces a statement for a @cypher directive field.
// Pattern: MATCH (this:Label) WHERE ... CALL { WITH this <statement> AS __cypher_result } RETURN __cypher_result
// The user's Cypher statement is wrapped in a CALL subquery with the parent node
// bound as `this`. Field arguments are merged into params alongside the parent WHERE params.
// nil args means no additional parameters beyond parent WHERE.
func CypherDirective(parentLabel string, parentWhere WhereClause, statement string, args map[string]any) Statement {
	b := New().Match("this", parentLabel)
	b = addWhereFromClause(b, "this", parentWhere, "")
	callBody := fmt.Sprintf("{ WITH this %s AS __cypher_result }", statement)
	b = b.addClause("CALL", callBody)
	b = b.Return("__cypher_result")
	if len(args) > 0 {
		b = b.addParams(args)
	}
	return b.Build()
}
