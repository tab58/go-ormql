package translate

import (
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// =============================================================================
// VAR-2: resolveValue() + toInt64()
// =============================================================================

// varVal creates an ast.Value representing a GraphQL variable reference ($varName).
func varVal(name string) *ast.Value {
	return &ast.Value{Kind: ast.Variable, Raw: name}
}

// --- resolveValue: Variable kind ---

// Test: resolveValue with ast.Variable kind returns the value from the variables map.
// Expected: resolveValue($title, {"title": "Matrix"}) → "Matrix"
// FAILS: stub returns nil.
func TestResolveValue_Variable_ReturnsFromMap(t *testing.T) {
	val := varVal("title")
	vars := map[string]any{"title": "Matrix"}

	result := resolveValue(val, vars)
	if result != "Matrix" {
		t.Errorf("expected 'Matrix', got %v", result)
	}
}

// Test: resolveValue with ast.Variable kind and missing key returns nil.
// Expected: resolveValue($unknown, {"title": "Matrix"}) → nil
// PASSES: stub returns nil (guardrail).
func TestResolveValue_Variable_MissingKey_ReturnsNil(t *testing.T) {
	val := varVal("unknown")
	vars := map[string]any{"title": "Matrix"}

	result := resolveValue(val, vars)
	if result != nil {
		t.Errorf("expected nil for missing variable, got %v", result)
	}
}

// Test: resolveValue with ast.Variable kind and nil variables map returns nil.
// Expected: resolveValue($title, nil) → nil
// PASSES: stub returns nil (guardrail).
func TestResolveValue_Variable_NilVariables_ReturnsNil(t *testing.T) {
	val := varVal("title")

	result := resolveValue(val, nil)
	if result != nil {
		t.Errorf("expected nil for nil variables map, got %v", result)
	}
}

// Test: resolveValue with ast.Variable kind returns float64 for JSON numbers.
// Expected: resolveValue($year, {"year": float64(1999)}) → float64(1999)
// FAILS: stub returns nil.
func TestResolveValue_Variable_Float64Number(t *testing.T) {
	val := varVal("year")
	vars := map[string]any{"year": float64(1999)}

	result := resolveValue(val, vars)
	f, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T: %v", result, result)
	}
	if f != 1999 {
		t.Errorf("expected 1999, got %v", f)
	}
}

// Test: resolveValue with ast.Variable kind returns bool for JSON booleans.
// Expected: resolveValue($flag, {"flag": true}) → true
// FAILS: stub returns nil.
func TestResolveValue_Variable_Bool(t *testing.T) {
	val := varVal("flag")
	vars := map[string]any{"flag": true}

	result := resolveValue(val, vars)
	b, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T: %v", result, result)
	}
	if !b {
		t.Error("expected true, got false")
	}
}

// Test: resolveValue with ast.Variable kind returns []any for JSON arrays.
// Expected: resolveValue($ids, {"ids": []any{"1","2"}}) → []any{"1","2"}
// FAILS: stub returns nil.
func TestResolveValue_Variable_List(t *testing.T) {
	val := varVal("ids")
	vars := map[string]any{"ids": []any{"1", "2"}}

	result := resolveValue(val, vars)
	list, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T: %v", result, result)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
}

// --- resolveValue: Scalar literals (same behavior as astValueToGo) ---

// Test: resolveValue with IntValue produces int64 (same as astValueToGo).
// Expected: resolveValue({Kind: IntValue, Raw: "42"}, nil) → int64(42)
// FAILS: stub returns nil.
func TestResolveValue_IntLiteral(t *testing.T) {
	val := intVal("42")

	result := resolveValue(val, nil)
	n, ok := result.(int64)
	if !ok {
		t.Fatalf("expected int64 for IntValue, got %T: %v", result, result)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

// Test: resolveValue with FloatValue produces float64.
// Expected: resolveValue({Kind: FloatValue, Raw: "3.14"}, nil) → float64(3.14)
// FAILS: stub returns nil.
func TestResolveValue_FloatLiteral(t *testing.T) {
	val := floatVal("3.14")

	result := resolveValue(val, nil)
	f, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64 for FloatValue, got %T: %v", result, result)
	}
	if f != 3.14 {
		t.Errorf("expected 3.14, got %v", f)
	}
}

// Test: resolveValue with BooleanValue produces bool.
// Expected: resolveValue({Kind: BooleanValue, Raw: "true"}, nil) → true
// FAILS: stub returns nil.
func TestResolveValue_BoolLiteral(t *testing.T) {
	val := boolVal(true)

	result := resolveValue(val, nil)
	b, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool for BooleanValue, got %T: %v", result, result)
	}
	if !b {
		t.Error("expected true, got false")
	}
}

// Test: resolveValue with StringValue produces string.
// Expected: resolveValue({Kind: StringValue, Raw: "hello"}, nil) → "hello"
// FAILS: stub returns nil.
func TestResolveValue_StringLiteral(t *testing.T) {
	val := strVal("hello")

	result := resolveValue(val, nil)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string for StringValue, got %T: %v", result, result)
	}
	if s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}
}

// Test: resolveValue with EnumValue produces string (same as StringValue).
// Expected: resolveValue({Kind: EnumValue, Raw: "DESC"}, nil) → "DESC"
// FAILS: stub returns nil.
func TestResolveValue_EnumLiteral(t *testing.T) {
	val := &ast.Value{Kind: ast.EnumValue, Raw: "DESC"}

	result := resolveValue(val, nil)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string for EnumValue, got %T: %v", result, result)
	}
	if s != "DESC" {
		t.Errorf("expected 'DESC', got %q", s)
	}
}

// --- resolveValue: Compound types with mixed literal+variable children ---

// Test: resolveValue with ObjectValue containing mixed literal and variable children.
// Expected: Object with title=literal("Matrix") and year=$year resolves both.
// FAILS: stub returns nil.
func TestResolveValue_ObjectValue_MixedLiteralVariable(t *testing.T) {
	val := &ast.Value{
		Kind: ast.ObjectValue,
		Children: ast.ChildValueList{
			{Name: "title", Value: strVal("Matrix")},
			{Name: "year", Value: varVal("year")},
		},
	}
	vars := map[string]any{"year": float64(1999)}

	result := resolveValue(val, vars)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any for ObjectValue, got %T: %v", result, result)
	}
	if m["title"] != "Matrix" {
		t.Errorf("expected title='Matrix', got %v", m["title"])
	}
	if m["year"] != float64(1999) {
		t.Errorf("expected year=1999, got %v", m["year"])
	}
}

// Test: resolveValue with ListValue containing mixed literal and variable entries.
// Expected: List with literal("a") and $name resolves both.
// FAILS: stub returns nil.
func TestResolveValue_ListValue_MixedLiteralVariable(t *testing.T) {
	val := &ast.Value{
		Kind: ast.ListValue,
		Children: ast.ChildValueList{
			{Value: strVal("a")},
			{Value: varVal("name")},
		},
	}
	vars := map[string]any{"name": "b"}

	result := resolveValue(val, vars)
	list, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any for ListValue, got %T: %v", result, result)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	if list[0] != "a" {
		t.Errorf("expected list[0]='a', got %v", list[0])
	}
	if list[1] != "b" {
		t.Errorf("expected list[1]='b', got %v", list[1])
	}
}

// Test: resolveValue with pure literal ObjectValue (no variables) matches astValueToGo.
// Expected: resolveValue({title: "Matrix"}, nil) == astValueToGo({title: "Matrix"})
// FAILS: stub returns nil.
func TestResolveValue_ObjectValue_AllLiterals(t *testing.T) {
	val := makeWhereValue(map[string]*ast.Value{
		"title": strVal("Matrix"),
	})

	result := resolveValue(val, nil)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T: %v", result, result)
	}
	if m["title"] != "Matrix" {
		t.Errorf("expected title='Matrix', got %v", m["title"])
	}
}

// --- toInt64 ---

// Test: toInt64 converts float64 (JSON number) to int64.
// Expected: toInt64(float64(10)) → int64(10)
// FAILS: stub returns 0.
func TestToInt64_Float64(t *testing.T) {
	result := toInt64(float64(10))
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

// Test: toInt64 converts int64 (literal parsing) to int64 (passthrough).
// Expected: toInt64(int64(10)) → int64(10)
// FAILS: stub returns 0.
func TestToInt64_Int64(t *testing.T) {
	result := toInt64(int64(10))
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

// Test: toInt64 converts int to int64.
// Expected: toInt64(int(10)) → int64(10)
// FAILS: stub returns 0.
func TestToInt64_Int(t *testing.T) {
	result := toInt64(int(10))
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

// Test: toInt64 converts string to int64.
// Expected: toInt64("10") → int64(10)
// FAILS: stub returns 0.
func TestToInt64_String(t *testing.T) {
	result := toInt64("10")
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

// Test: toInt64 returns 0 for invalid string.
// Expected: toInt64("invalid") → int64(0)
// PASSES: stub returns 0 (guardrail).
func TestToInt64_InvalidString(t *testing.T) {
	result := toInt64("invalid")
	if result != 0 {
		t.Errorf("expected 0 for invalid string, got %d", result)
	}
}

// Test: toInt64 returns 0 for nil.
// Expected: toInt64(nil) → int64(0)
// PASSES: stub returns 0 (guardrail).
func TestToInt64_Nil(t *testing.T) {
	result := toInt64(nil)
	if result != 0 {
		t.Errorf("expected 0 for nil, got %d", result)
	}
}

// Test: toInt64 handles float64 with fractional part (truncates).
// Expected: toInt64(float64(10.9)) → int64(10)
// FAILS: stub returns 0.
func TestToInt64_Float64Fractional(t *testing.T) {
	result := toInt64(float64(10.9))
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

// --- resolveValue: NullValue ---

// Test: resolveValue with NullValue returns Go nil.
// Expected: resolveValue({Kind: NullValue, Raw: "null"}, nil) → nil
func TestResolveValue_NullLiteral(t *testing.T) {
	val := &ast.Value{Kind: ast.NullValue, Raw: "null"}

	result := resolveValue(val, nil)
	if result != nil {
		t.Errorf("expected nil for NullValue, got %v (%T)", result, result)
	}
}

// Test: resolveValue with nil val returns nil.
// Expected: resolveValue(nil, vars) → nil
func TestResolveValue_NilVal(t *testing.T) {
	vars := map[string]any{"title": "Matrix"}

	result := resolveValue(nil, vars)
	if result != nil {
		t.Errorf("expected nil for nil val, got %v", result)
	}
}

// Test: astValueToGo with NullValue returns Go nil.
// Expected: astValueToGo({Kind: NullValue}) → nil
func TestAstValueToGo_NullValue(t *testing.T) {
	val := &ast.Value{Kind: ast.NullValue, Raw: "null"}

	result := astValueToGo(val)
	if result != nil {
		t.Errorf("expected nil for NullValue, got %v (%T)", result, result)
	}
}
