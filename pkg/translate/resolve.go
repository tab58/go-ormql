package translate

import (
	"strconv"

	"github.com/vektah/gqlparser/v2/ast"
)

// resolveValue resolves an AST value to a Go native type, handling variable references.
//
// If val.Kind is ast.Variable, looks up val.Raw in the variables map and returns
// the Go value directly. For compound types (ObjectValue, ListValue), recurses
// into children. For scalar literals, parses the AST raw value.
func resolveValue(val *ast.Value, variables map[string]any) any {
	if val == nil {
		return nil
	}

	// Variable reference: look up in variables map
	if val.Kind == ast.Variable {
		if variables == nil {
			return nil
		}
		return variables[val.Raw]
	}

	// Compound types: recurse into children with resolveValue
	switch val.Kind {
	case ast.ObjectValue:
		m := make(map[string]any, len(val.Children))
		for _, child := range val.Children {
			m[child.Name] = resolveValue(child.Value, variables)
		}
		return m
	case ast.ListValue:
		items := make([]any, 0, len(val.Children))
		for _, child := range val.Children {
			items = append(items, resolveValue(child.Value, variables))
		}
		return items
	}

	// Null literal: return Go nil
	if val.Kind == ast.NullValue {
		return nil
	}

	// Scalar literals: delegate to existing astValueToGo
	return astValueToGo(val)
}

// toInt64 converts a resolved value to int64, handling JSON number types.
// JSON deserializes numbers as float64, but pagination args (first) need int64.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case string:
		parsed, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

// astValueToGo converts a scalar ast.Value to a Go native type for parameterization.
// Handles IntValue, FloatValue, BooleanValue, StringValue, EnumValue, and NullValue.
// Compound types (ListValue, ObjectValue) and Variable references are handled by
// resolveValue before reaching this function.
func astValueToGo(val *ast.Value) any {
	switch val.Kind {
	case ast.IntValue:
		n, _ := strconv.ParseInt(val.Raw, 10, 64)
		return n
	case ast.FloatValue:
		f, _ := strconv.ParseFloat(val.Raw, 64)
		return f
	case ast.BooleanValue:
		b, _ := strconv.ParseBool(val.Raw)
		return b
	case ast.StringValue, ast.EnumValue:
		return val.Raw
	case ast.NullValue:
		return nil
	default:
		return val.Raw
	}
}
