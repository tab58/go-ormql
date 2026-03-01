package cypher

import (
	"fmt"
	"strings"
)

// Op represents a Cypher comparison operator used in WHERE clauses.
type Op string

const (
	OpEq         Op = "="
	OpNEq        Op = "<>"
	OpGT         Op = ">"
	OpGTE        Op = ">="
	OpLT         Op = "<"
	OpLTE        Op = "<="
	OpContains   Op = "CONTAINS"
	OpStartsWith Op = "STARTS WITH"
	OpEndsWith   Op = "ENDS WITH"
	OpRegex      Op = "=~"
	OpIn         Op = "IN"
	OpNotIn      Op = "NOT_IN"
	OpIsNull     Op = "IS NULL"
	OpIsNotNull  Op = "IS NOT NULL"
)

// Predicate represents a single field-level filter condition.
// Value is nil for OpIsNull/OpIsNotNull (value-less operators).
type Predicate struct {
	Field string
	Op    Op
	Value any
}

// WhereClause is a composable filter tree for building Cypher WHERE expressions.
// Top-level Predicates are AND-joined by default.
// AND sub-clauses are AND-joined within parentheses.
// OR sub-clauses are OR-joined within parentheses.
// NOT wraps its contents with NOT (...).
// An empty WhereClause produces no WHERE clause.
type WhereClause struct {
	Predicates []Predicate
	AND        []WhereClause
	OR         []WhereClause
	NOT        *WhereClause
}

// IsEmpty returns true if the WhereClause has no predicates and no sub-clauses.
func (w WhereClause) IsEmpty() bool {
	return len(w.Predicates) == 0 && len(w.AND) == 0 && len(w.OR) == 0 && w.NOT == nil
}

// EqualityWhere builds a WhereClause from a map of field=value pairs.
// Each entry becomes a Predicate with OpEq.
func EqualityWhere(fields map[string]any) WhereClause {
	if len(fields) == 0 {
		return WhereClause{}
	}
	preds := make([]Predicate, 0, len(fields))
	for _, k := range sortedKeys(fields) {
		preds = append(preds, Predicate{Field: k, Op: OpEq, Value: fields[k]})
	}
	return WhereClause{Predicates: preds}
}

// whereRenderer renders a WhereClause tree into a condition string with auto-incrementing params.
type whereRenderer struct {
	variable string
	counter  int
	params   map[string]any
}

// render produces a condition string for the given WhereClause, recursively handling
// AND, OR, and NOT sub-clauses with correct parenthesization.
func (r *whereRenderer) render(clause WhereClause) string {
	var parts []string

	for _, p := range clause.Predicates {
		parts = append(parts, r.renderPredicate(p))
	}

	if s := r.renderSubClauses(clause.AND, " AND "); s != "" {
		parts = append(parts, s)
	}
	if s := r.renderSubClauses(clause.OR, " OR "); s != "" {
		parts = append(parts, s)
	}

	if clause.NOT != nil {
		notRendered := r.render(*clause.NOT)
		if notRendered != "" {
			parts = append(parts, "NOT ("+notRendered+")")
		}
	}

	return strings.Join(parts, " AND ")
}

// renderSubClauses renders a list of sub-clauses, joining them with the given
// operator (e.g. " AND " or " OR ") and wrapping in parentheses.
// Returns empty string if no sub-clauses produce output.
func (r *whereRenderer) renderSubClauses(clauses []WhereClause, joiner string) string {
	if len(clauses) == 0 {
		return ""
	}
	var rendered []string
	for _, sub := range clauses {
		s := r.render(sub)
		if s != "" {
			rendered = append(rendered, s)
		}
	}
	if len(rendered) == 0 {
		return ""
	}
	return "(" + strings.Join(rendered, joiner) + ")"
}

func (r *whereRenderer) renderPredicate(p Predicate) string {
	field := fmt.Sprintf("%s.%s", r.variable, p.Field)

	switch p.Op {
	case OpIsNull:
		return field + " IS NULL"
	case OpIsNotNull:
		return field + " IS NOT NULL"
	case OpNotIn:
		paramName := r.nextParam(p.Value)
		return fmt.Sprintf("NOT %s IN $%s", field, paramName)
	default:
		paramName := r.nextParam(p.Value)
		return fmt.Sprintf("%s %s $%s", field, string(p.Op), paramName)
	}
}

func (r *whereRenderer) nextParam(value any) string {
	name := fmt.Sprintf("p%d", r.counter)
	r.counter++
	r.params[name] = value
	return name
}
