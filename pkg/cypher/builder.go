package cypher

import (
	"fmt"
	"sort"
	"strings"
)

// Direction represents the traversal direction for a relationship pattern.
type Direction string

const (
	DirectionIN  Direction = "IN"
	DirectionOUT Direction = "OUT"
)

// clause represents a single Cypher clause (MATCH, WHERE, CREATE, etc.).
type clause struct {
	keyword string
	body    string
}

// Builder constructs parameterized Cypher statements via immutable method chaining.
// Each method returns a new Builder — the original is never modified.
type Builder struct {
	clauses  []clause
	params   map[string]any
	variable string // last matched variable name, used by WhereClause rendering
}

// New creates a new empty Builder.
func New() Builder {
	return Builder{}
}

// clone returns a deep copy of the builder to ensure immutability.
func (b Builder) clone() Builder {
	clauses := make([]clause, len(b.clauses))
	copy(clauses, b.clauses)

	params := make(map[string]any, len(b.params))
	for k, v := range b.params {
		params[k] = v
	}

	return Builder{clauses: clauses, params: params, variable: b.variable}
}

// addClause appends a clause to a cloned builder.
func (b Builder) addClause(keyword, body string) Builder {
	nb := b.clone()
	nb.clauses = append(nb.clauses, clause{keyword: keyword, body: body})
	return nb
}

// addParams merges params into a cloned builder.
func (b Builder) addParams(params map[string]any) Builder {
	nb := b.clone()
	for k, v := range params {
		nb.params[k] = v
	}
	return nb
}

// Match adds a MATCH (variable:Label) clause.
func (b Builder) Match(variable, label string) Builder {
	nb := b.addClause("MATCH", fmt.Sprintf("(%s:%s)", variable, label))
	nb.variable = variable
	return nb
}

// Where adds a WHERE clause with parameterized conditions.
func (b Builder) Where(condition string, params map[string]any) Builder {
	nb := b.addClause("WHERE", condition)
	for k, v := range params {
		nb.params[k] = v
	}
	return nb
}

// WhereClause adds a WHERE clause from a composable WhereClause tree.
// Renders predicates, AND/OR/NOT sub-clauses with correct parenthesization.
// An empty WhereClause is a no-op.
func (b Builder) WhereClause(wc WhereClause) Builder {
	if wc.IsEmpty() {
		return b
	}
	r := &whereRenderer{
		variable: b.variable,
		params:   make(map[string]any),
	}
	condition := r.render(wc)
	if condition == "" {
		return b
	}
	return b.Where(condition, r.params)
}

// Create adds a CREATE (variable:Label {props}) clause.
func (b Builder) Create(variable, label string, props map[string]any) Builder {
	if len(props) == 0 {
		return b.addClause("CREATE", fmt.Sprintf("(%s:%s)", variable, label))
	}

	nb := b.clone()
	assignments := buildPropAssignments(props)
	nb.clauses = append(nb.clauses, clause{
		keyword: "CREATE",
		body:    fmt.Sprintf("(%s:%s {%s})", variable, label, strings.Join(assignments, ", ")),
	})
	for k, v := range props {
		nb.params[k] = v
	}
	return nb
}

// Set adds a SET clause for updating properties.
func (b Builder) Set(variable string, props map[string]any) Builder {
	nb := b.clone()
	var parts []string
	for _, k := range sortedKeys(props) {
		paramName := "set_" + k
		parts = append(parts, fmt.Sprintf("%s.%s = $%s", variable, k, paramName))
		nb.params[paramName] = props[k]
	}
	nb.clauses = append(nb.clauses, clause{keyword: "SET", body: strings.Join(parts, ", ")})
	return nb
}

// Delete adds a DELETE or DETACH DELETE clause.
func (b Builder) Delete(variable string, detach bool) Builder {
	if detach {
		return b.addClause("DETACH DELETE", variable)
	}
	return b.addClause("DELETE", variable)
}

// Return adds a RETURN clause with the given expressions.
func (b Builder) Return(expressions ...string) Builder {
	return b.addClause("RETURN", strings.Join(expressions, ", "))
}

// OrderBy adds an ORDER BY clause with one or more SortField values.
// Multiple fields are comma-separated in a single ORDER BY clause.
// Empty fields list is a no-op (no ORDER BY produced).
func (b Builder) OrderBy(fields ...SortField) Builder {
	if len(fields) == 0 {
		return b
	}
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = f.Field + " " + string(f.Direction)
	}
	return b.addClause("ORDER BY", strings.Join(parts, ", "))
}

// Skip adds a SKIP clause.
func (b Builder) Skip(n int) Builder {
	nb := b.clone()
	nb.clauses = append(nb.clauses, clause{keyword: "SKIP", body: "$skip"})
	nb.params["skip"] = n
	return nb
}

// Limit adds a LIMIT clause.
func (b Builder) Limit(n int) Builder {
	nb := b.clone()
	nb.clauses = append(nb.clauses, clause{keyword: "LIMIT", body: "$limit"})
	nb.params["limit"] = n
	return nb
}

// With adds a WITH clause for query piping.
func (b Builder) With(expressions ...string) Builder {
	return b.addClause("WITH", strings.Join(expressions, ", "))
}

// RelationshipMatch adds a MATCH pattern for a relationship between two nodes.
func (b Builder) RelationshipMatch(from, rel, relType, to, toLabel string, dir Direction) Builder {
	var pattern string
	if dir == DirectionOUT {
		pattern = fmt.Sprintf("(%s)-[%s:%s]->(%s:%s)", from, rel, relType, to, toLabel)
	} else {
		pattern = fmt.Sprintf("(%s)<-[%s:%s]-(%s:%s)", from, rel, relType, to, toLabel)
	}
	// Append as a raw clause that follows the previous MATCH
	return b.addClause("MATCH", pattern)
}

// RelationshipCreate adds a CREATE pattern for a relationship between two nodes.
func (b Builder) RelationshipCreate(from, rel, relType, to string, props map[string]any) Builder {
	nb := b.clone()
	if len(props) == 0 {
		nb.clauses = append(nb.clauses, clause{
			keyword: "CREATE",
			body:    fmt.Sprintf("(%s)-[%s:%s]->(%s)", from, rel, relType, to),
		})
		return nb
	}

	assignments := buildPropAssignments(props)
	nb.clauses = append(nb.clauses, clause{
		keyword: "CREATE",
		body:    fmt.Sprintf("(%s)-[%s:%s {%s}]->(%s)", from, rel, relType, strings.Join(assignments, ", "), to),
	})
	for k, v := range props {
		nb.params[k] = v
	}
	return nb
}

// Build produces the final Statement from the builder's accumulated clauses.
func (b Builder) Build() Statement {
	if len(b.clauses) == 0 {
		return Statement{}
	}

	var parts []string
	for _, c := range b.clauses {
		parts = append(parts, c.keyword+" "+c.body)
	}

	var params map[string]any
	if len(b.params) > 0 {
		params = make(map[string]any, len(b.params))
		for k, v := range b.params {
			params[k] = v
		}
	}

	return Statement{
		Query:  strings.Join(parts, " "),
		Params: params,
	}
}

// buildPropAssignments builds sorted "key: $key" pairs for property maps.
func buildPropAssignments(props map[string]any) []string {
	var assignments []string
	for _, k := range sortedKeys(props) {
		assignments = append(assignments, fmt.Sprintf("%s: $%s", k, k))
	}
	return assignments
}

// sortedKeys returns map keys in sorted order for deterministic output.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
