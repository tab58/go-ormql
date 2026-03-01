package client

import (
	"strconv"

	"github.com/vektah/gqlparser/v2/ast"
)

// resolveArgMap extracts a map argument from a field, resolving variable references.
func resolveArgMap(field *ast.Field, argName string, variables map[string]any) map[string]any {
	for _, arg := range field.Arguments {
		if arg.Name == argName {
			return valueToMap(arg.Value, variables)
		}
	}
	return map[string]any{}
}

// resolveArgList extracts a list argument from a field, resolving variable references.
func resolveArgList(field *ast.Field, argName string, variables map[string]any) []any {
	for _, arg := range field.Arguments {
		if arg.Name == argName {
			return valueToList(arg.Value, variables)
		}
	}
	return nil
}

// valueToMap converts an AST value to map[string]any, resolving variables.
func valueToMap(val *ast.Value, variables map[string]any) map[string]any {
	if val == nil {
		return map[string]any{}
	}
	if val.Kind == ast.Variable {
		if v, ok := variables[val.Raw]; ok {
			if m, ok := v.(map[string]any); ok {
				return m
			}
		}
		return map[string]any{}
	}
	if val.Kind == ast.ObjectValue {
		result := map[string]any{}
		for _, child := range val.Children {
			result[child.Name] = valueToAny(child.Value, variables)
		}
		return result
	}
	return map[string]any{}
}

// valueToList converts an AST value to []any, resolving variables.
func valueToList(val *ast.Value, variables map[string]any) []any {
	if val == nil {
		return nil
	}
	if val.Kind == ast.Variable {
		if v, ok := variables[val.Raw]; ok {
			if l, ok := v.([]any); ok {
				return l
			}
			if l, ok := v.([]map[string]any); ok {
				result := make([]any, len(l))
				for i, m := range l {
					result[i] = m
				}
				return result
			}
		}
		return nil
	}
	if val.Kind == ast.ListValue {
		result := make([]any, len(val.Children))
		for i, child := range val.Children {
			result[i] = valueToAny(child.Value, variables)
		}
		return result
	}
	return nil
}

// valueToAny converts an AST value to a Go value.
func valueToAny(val *ast.Value, variables map[string]any) any {
	if val == nil {
		return nil
	}
	switch val.Kind {
	case ast.Variable:
		if v, ok := variables[val.Raw]; ok {
			return v
		}
		return nil
	case ast.IntValue:
		n, _ := strconv.Atoi(val.Raw)
		return n
	case ast.FloatValue:
		f, _ := strconv.ParseFloat(val.Raw, 64)
		return f
	case ast.StringValue:
		return val.Raw
	case ast.BooleanValue:
		return val.Raw == "true"
	case ast.ObjectValue:
		return valueToMap(val, variables)
	case ast.ListValue:
		return valueToList(val, variables)
	default:
		return val.Raw
	}
}
