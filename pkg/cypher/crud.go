package cypher

import (
	"fmt"
	"strings"
)

// NodeCreate produces a CREATE (n:Label {props}) RETURN n statement.
// Empty props creates a label-only node. Nil values become Cypher null parameters.
func NodeCreate(label string, props map[string]any) Statement {
	b := New().Create("n", label, props).Return("n")
	return b.Build()
}

// NodeMatch produces a MATCH (n:Label) WHERE ... RETURN n [ORDER BY ...] statement.
// Empty WhereClause omits the WHERE clause entirely.
// Non-nil orderBy appends ORDER BY. Nil orderBy produces no ORDER BY.
func NodeMatch(label string, where WhereClause, orderBy []SortField) Statement {
	b := New().Match("n", label)
	b = addWhereFromClause(b, "n", where, "")
	b = b.Return("n")
	if len(orderBy) > 0 {
		b = b.OrderBy(prefixSortFields("n", orderBy)...)
	}
	return b.Build()
}

// NodeUpdate produces a MATCH (n:Label) WHERE ... SET ... RETURN n statement.
// Empty WhereClause omits WHERE. Empty set map produces no SET clause.
func NodeUpdate(label string, where WhereClause, set map[string]any) Statement {
	b := New().Match("n", label)
	b = addWhereFromClause(b, "n", where, "where_")
	b = addSetClause(b, "n", set, "set_")
	b = b.Return("n")
	return b.Build()
}

// NodeDelete produces a MATCH (n:Label) WHERE ... DETACH DELETE n statement.
// Empty WhereClause omits WHERE.
func NodeDelete(label string, where WhereClause) Statement {
	b := New().Match("n", label)
	b = addWhereFromClause(b, "n", where, "")
	b = b.Delete("n", true)
	return b.Build()
}

// RelCreate produces a statement that MATCHes both endpoint nodes and CREATEs a relationship.
// Pattern: MATCH (a:FromLabel), (b:ToLabel) WHERE ... CREATE (a)-[r:TYPE {props}]->(b) RETURN r
// Empty props creates a relationship with no properties.
func RelCreate(fromLabel string, fromWhere WhereClause, relType string, toLabel string, toWhere WhereClause, props map[string]any) Statement {
	b := New().addClause("MATCH", fmt.Sprintf("(%s:%s), (%s:%s)", "a", fromLabel, "b", toLabel))
	b = addWhereFromClausePair(b, "a", fromWhere, "from_", "b", toWhere, "to_")
	b = b.RelationshipCreate("a", "r", relType, "b", props)
	b = b.Return("r")
	return b.Build()
}

// RelUpdate produces a statement that MATCHes a relationship and SETs properties.
// Pattern: MATCH (a:FromLabel)-[r:TYPE]->(b:ToLabel) WHERE ... SET r.prop = $param RETURN r
func RelUpdate(fromLabel string, relType string, toLabel string, where WhereClause, set map[string]any) Statement {
	b := New().addClause("MATCH", fmt.Sprintf("(%s:%s)-[%s:%s]->(%s:%s)", "a", fromLabel, "r", relType, "b", toLabel))
	b = addWhereFromClause(b, "a", where, "")
	b = addSetClause(b, "r", set, "set_")
	b = b.Return("r")
	return b.Build()
}

// RelDelete produces a statement that MATCHes a relationship and DELETEs it.
// Pattern: MATCH (a:FromLabel)-[r:TYPE]->(b:ToLabel) WHERE ... DELETE r
func RelDelete(fromLabel string, relType string, toLabel string, where WhereClause) Statement {
	b := New().addClause("MATCH", fmt.Sprintf("(%s:%s)-[%s:%s]->(%s:%s)", "a", fromLabel, "r", relType, "b", toLabel))
	b = addWhereFromClause(b, "a", where, "")
	b = b.Delete("r", false)
	return b.Build()
}

// addWhereFromClause adds a WHERE clause from a WhereClause's equality predicates.
// Uses field-name-based param keys with an optional prefix to avoid collisions.
func addWhereFromClause(b Builder, variable string, wc WhereClause, prefix string) Builder {
	if wc.IsEmpty() {
		return b
	}
	var conditions []string
	params := map[string]any{}
	for _, p := range wc.Predicates {
		paramName := prefix + p.Field
		conditions = append(conditions, fmt.Sprintf("%s.%s = $%s", variable, p.Field, paramName))
		params[paramName] = p.Value
	}
	return b.Where(strings.Join(conditions, " AND "), params)
}

// addWhereFromClausePair combines WHERE conditions from two WhereClause values
// for two different variables (e.g., from and to nodes).
func addWhereFromClausePair(b Builder, varA string, wcA WhereClause, prefixA string, varB string, wcB WhereClause, prefixB string) Builder {
	var conditions []string
	params := map[string]any{}
	for _, p := range wcA.Predicates {
		paramName := prefixA + p.Field
		conditions = append(conditions, fmt.Sprintf("%s.%s = $%s", varA, p.Field, paramName))
		params[paramName] = p.Value
	}
	for _, p := range wcB.Predicates {
		paramName := prefixB + p.Field
		conditions = append(conditions, fmt.Sprintf("%s.%s = $%s", varB, p.Field, paramName))
		params[paramName] = p.Value
	}
	if len(conditions) == 0 {
		return b
	}
	return b.Where(strings.Join(conditions, " AND "), params)
}

// addSetClause adds a SET clause from a map of field=value updates.
// prefix is prepended to parameter names to avoid collisions (e.g. "set_").
func addSetClause(b Builder, variable string, set map[string]any, prefix string) Builder {
	nb := b.clone()
	var parts []string
	for _, k := range sortedKeys(set) {
		paramName := prefix + k
		parts = append(parts, fmt.Sprintf("%s.%s = $%s", variable, k, paramName))
		nb.params[paramName] = set[k]
	}
	nb.clauses = append(nb.clauses, clause{keyword: "SET", body: strings.Join(parts, ", ")})
	return nb
}

// prefixSortFields prepends a variable name to each SortField's Field.
// e.g. prefixSortFields("n", [{Field:"title", Direction:SortASC}]) → [{Field:"n.title", Direction:SortASC}]
func prefixSortFields(variable string, fields []SortField) []SortField {
	result := make([]SortField, len(fields))
	for i, f := range fields {
		result[i] = SortField{Field: variable + "." + f.Field, Direction: f.Direction}
	}
	return result
}
