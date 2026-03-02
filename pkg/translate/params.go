package translate

import "fmt"

// paramScope tracks parameter naming within a single translation.
// prefix is a namespace (e.g., "" for root, "sub0_" for first nested subquery).
// next is an auto-incrementing counter. params collects all parameters for
// the final Statement. Parameters are globally unique across the entire query
// via namespacing.
type paramScope struct {
	prefix   string
	next     int
	params   map[string]any
	children []*paramScope
}

// newParamScope creates a root parameter scope.
func newParamScope() *paramScope {
	return &paramScope{
		prefix: "",
		next:   0,
		params: make(map[string]any),
	}
}

// sub creates a child scope with a namespaced prefix.
// e.g., scope.sub("sub0") creates prefix "sub0_" for nested parameters.
func (s *paramScope) sub(name string) *paramScope {
	child := &paramScope{
		prefix: s.prefix + name + "_",
		next:   0,
		params: make(map[string]any),
	}
	s.children = append(s.children, child)
	return child
}

// add registers a parameter value and returns its placeholder name (e.g., "$p0", "$sub0_p1").
func (s *paramScope) add(value any) string {
	key := fmt.Sprintf("%sp%d", s.prefix, s.next)
	s.next++
	s.params[key] = value
	return "$" + key
}

// addNamed registers a parameter with a specific name (e.g., "$set_title", "$offset").
func (s *paramScope) addNamed(name string, value any) string {
	key := s.prefix + name
	s.params[key] = value
	return "$" + key
}

// collect returns all parameters from this scope and all child scopes.
func (s *paramScope) collect() map[string]any {
	result := make(map[string]any, len(s.params))
	for k, v := range s.params {
		result[k] = v
	}
	for _, child := range s.children {
		for k, v := range child.collect() {
			result[k] = v
		}
	}
	return result
}
