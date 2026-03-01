package cypher

import "fmt"

// ConnectionQuery produces a Relay-style paginated query.
// Pattern: MATCH (n:Label) [WHERE ...] RETURN n [ORDER BY ...] [SKIP $offset] LIMIT $first
// When orderBy is nil/empty, defaults to ORDER BY n.id ASC for stable cursor pagination.
// Offset 0 omits the SKIP clause. Empty WhereClause omits WHERE.
func ConnectionQuery(label string, where WhereClause, orderBy []SortField, first int, offset int) Statement {
	b := New().Match("n", label)
	b = addWhereFromClause(b, "n", where, "")
	b = b.Return("n")
	if len(orderBy) == 0 {
		b = b.OrderBy(SortField{Field: "n.id", Direction: SortASC})
	} else {
		b = b.OrderBy(prefixSortFields("n", orderBy)...)
	}
	if offset > 0 {
		b = b.Skip(offset)
	}
	b = b.Limit(first)
	return b.Build()
}

// ConnectionCount produces a count query for Relay totalCount.
// Pattern: MATCH (n:Label) [WHERE ...] RETURN count(n)
// Empty WhereClause omits WHERE.
func ConnectionCount(label string, where WhereClause) Statement {
	b := New().Match("n", label)
	b = addWhereFromClause(b, "n", where, "")
	b = b.Return("count(n)")
	return b.Build()
}

// RelConnectionQuery produces a relationship-level Relay-style paginated query.
// Pattern (OUT): MATCH (parent:FromLabel)-[r:TYPE]->(child:ToLabel) WHERE ... RETURN child, r ORDER BY ... SKIP $offset LIMIT $first
// Pattern (IN):  MATCH (parent:FromLabel)<-[r:TYPE]-(child:ToLabel) WHERE ... RETURN child, r ORDER BY ... SKIP $offset LIMIT $first
// When orderBy is nil/empty, defaults to ORDER BY child.id ASC for stable cursor pagination.
// Offset 0 omits the SKIP clause. Empty WhereClause omits WHERE conditions for that side.
func RelConnectionQuery(fromLabel string, fromWhere WhereClause, relType string, toLabel string, dir Direction, toWhere WhereClause, orderBy []SortField, first int, offset int) Statement {
	b := New().addClause("MATCH", relMatchPattern(fromLabel, relType, toLabel, dir))
	b = addWhereFromClausePair(b, "parent", fromWhere, "from_", "child", toWhere, "to_")
	b = b.Return("child", "r")
	if len(orderBy) == 0 {
		b = b.OrderBy(SortField{Field: "child.id", Direction: SortASC})
	} else {
		b = b.OrderBy(prefixSortFields("child", orderBy)...)
	}
	if offset > 0 {
		b = b.Skip(offset)
	}
	b = b.Limit(first)
	return b.Build()
}

// RelConnectionCount produces a relationship-level count query for Relay totalCount.
// Pattern (OUT): MATCH (parent:FromLabel)-[r:TYPE]->(child:ToLabel) WHERE ... RETURN count(child)
// Pattern (IN):  MATCH (parent:FromLabel)<-[r:TYPE]-(child:ToLabel) WHERE ... RETURN count(child)
// Empty WhereClause omits WHERE conditions for that side.
func RelConnectionCount(fromLabel string, fromWhere WhereClause, relType string, toLabel string, dir Direction, toWhere WhereClause) Statement {
	b := New().addClause("MATCH", relMatchPattern(fromLabel, relType, toLabel, dir))
	b = addWhereFromClausePair(b, "parent", fromWhere, "from_", "child", toWhere, "to_")
	b = b.Return("count(child)")
	return b.Build()
}

// relMatchPattern builds a Cypher MATCH pattern for a relationship traversal.
// DirectionOUT: (parent:From)-[r:TYPE]->(child:To)
// DirectionIN:  (parent:From)<-[r:TYPE]-(child:To)
func relMatchPattern(fromLabel, relType, toLabel string, dir Direction) string {
	if dir == DirectionOUT {
		return fmt.Sprintf("(%s:%s)-[%s:%s]->(%s:%s)", "parent", fromLabel, "r", relType, "child", toLabel)
	}
	return fmt.Sprintf("(%s:%s)<-[%s:%s]-(%s:%s)", "parent", fromLabel, "r", relType, "child", toLabel)
}
