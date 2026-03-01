package cypher

// SortDirection represents a sort ordering direction.
type SortDirection string

const (
	SortASC  SortDirection = "ASC"
	SortDESC SortDirection = "DESC"
)

// SortField represents a single ORDER BY field with direction.
type SortField struct {
	Field     string
	Direction SortDirection
}
