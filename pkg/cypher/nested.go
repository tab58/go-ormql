package cypher

import "fmt"

// RelDisconnect produces a statement that MATCHes a relationship between two nodes
// and DELETEs only the relationship, keeping both nodes intact.
// Pattern: MATCH (a:FromLabel)-[r:TYPE]->(b:ToLabel) WHERE ... DELETE r
// Empty WhereClause on either side omits the corresponding conditions.
func RelDisconnect(fromLabel string, fromWhere WhereClause, relType string, toLabel string, toWhere WhereClause) Statement {
	b := New().addClause("MATCH", fmt.Sprintf("(%s:%s)-[%s:%s]->(%s:%s)", "a", fromLabel, "r", relType, "b", toLabel))
	b = addWhereFromClausePair(b, "a", fromWhere, "from_", "b", toWhere, "to_")
	b = b.Delete("r", false)
	return b.Build()
}

// NestedUpdate produces a statement that MATCHes a relationship pattern and SETs
// properties on both the target node and the relationship edge.
// Pattern: MATCH (a:FromLabel)-[r:TYPE]->(b:ToLabel) WHERE ... SET b.x=$v, r.y=$v RETURN b, r
// nil nodeSet skips node SET, nil edgeSet skips edge SET, both nil produces no SET.
// Node SET params use "set_" prefix, edge SET params use "edge_" prefix.
func NestedUpdate(fromLabel string, fromWhere WhereClause, relType string, toLabel string, toWhere WhereClause, nodeSet map[string]any, edgeSet map[string]any) Statement {
	b := New().addClause("MATCH", fmt.Sprintf("(%s:%s)-[%s:%s]->(%s:%s)", "a", fromLabel, "r", relType, "b", toLabel))
	b = addWhereFromClausePair(b, "a", fromWhere, "from_", "b", toWhere, "to_")
	if len(nodeSet) > 0 {
		b = addSetClause(b, "b", nodeSet, "set_")
	}
	if len(edgeSet) > 0 {
		b = addSetClause(b, "r", edgeSet, "edge_")
	}
	b = b.Return("b", "r")
	return b.Build()
}

// NestedDelete produces a statement that MATCHes a relationship pattern and
// DETACH DELETEs the target node (removing the node and all its relationships).
// Pattern: MATCH (a:FromLabel)-[r:TYPE]->(b:ToLabel) WHERE ... DETACH DELETE b
func NestedDelete(fromLabel string, fromWhere WhereClause, relType string, toLabel string, toWhere WhereClause) Statement {
	b := New().addClause("MATCH", fmt.Sprintf("(%s:%s)-[%s:%s]->(%s:%s)", "a", fromLabel, "r", relType, "b", toLabel))
	b = addWhereFromClausePair(b, "a", fromWhere, "from_", "b", toWhere, "to_")
	b = b.Delete("b", true)
	return b.Build()
}
