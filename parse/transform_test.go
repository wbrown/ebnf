package parse

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestTransform_SimpleArithmetic(t *testing.T) {
	// Create a simple parse tree: add(number("2"), number("3"))
	tree := &ParseTree{
		Root: &Node{
			Rule: "add",
			Children: []*Node{
				{Rule: "number", Value: "2"},
				{Rule: "number", Value: "3"},
			},
		},
	}

	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		"add": func(a, b float64) float64 {
			return a + b
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 5.0 {
		t.Errorf("Expected 5.0, got %v", result)
	}
}

func TestTransform_NestedArithmetic(t *testing.T) {
	// Create tree: mul(add(number("2"), number("3")), number("4"))
	// Should evaluate to (2+3)*4 = 20
	tree := &ParseTree{
		Root: &Node{
			Rule: "mul",
			Children: []*Node{
				{
					Rule: "add",
					Children: []*Node{
						{Rule: "number", Value: "2"},
						{Rule: "number", Value: "3"},
					},
				},
				{Rule: "number", Value: "4"},
			},
		},
	}

	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		"add": func(a, b float64) float64 {
			return a + b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 20.0 {
		t.Errorf("Expected 20.0, got %v", result)
	}
}

func TestTransform_StringConcat(t *testing.T) {
	// Test string concatenation
	tree := &ParseTree{
		Root: &Node{
			Rule: "sentence",
			Children: []*Node{
				{Rule: "word", Value: "hello"},
				{Rule: "word", Value: "world"},
			},
		},
	}

	transforms := TransformMap{
		"word": Identity,
		"sentence": func(args ...interface{}) string {
			var words []string
			for _, arg := range args {
				words = append(words, arg.(string))
			}
			return strings.Join(words, " ")
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != "hello world" {
		t.Errorf("Expected 'hello world', got %v", result)
	}
}

func TestTransform_VariadicFunction(t *testing.T) {
	// Test variadic function
	tree := &ParseTree{
		Root: &Node{
			Rule: "sum",
			Children: []*Node{
				{Rule: "number", Value: "1"},
				{Rule: "number", Value: "2"},
				{Rule: "number", Value: "3"},
				{Rule: "number", Value: "4"},
			},
		},
	}

	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		"sum": func(nums ...float64) float64 {
			sum := 0.0
			for _, n := range nums {
				sum += n
			}
			return sum
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 10.0 {
		t.Errorf("Expected 10.0, got %v", result)
	}
}

func TestTransform_UntransformedRules(t *testing.T) {
	// Nodes without transforms should pass through children
	tree := &ParseTree{
		Root: &Node{
			Rule: "passthrough",
			Children: []*Node{
				{Rule: "number", Value: "42"},
			},
		},
	}

	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		// No transform for "passthrough"
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Should return the single transformed child directly
	if result != 42.0 {
		t.Errorf("Expected 42.0, got %v", result)
	}
}

func TestTransform_ReturnError(t *testing.T) {
	// Test functions that return errors
	tree := &ParseTree{
		Root: &Node{
			Rule: "div",
			Children: []*Node{
				{Rule: "number", Value: "10"},
				{Rule: "number", Value: "0"},
			},
		},
	}

	transforms := TransformMap{
		"number": func(s string) (float64, error) {
			return strconv.ParseFloat(s, 64)
		},
		"div": func(a, b float64) (float64, error) {
			if b == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return a / b, nil
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error for division by zero")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Errorf("Expected 'division by zero' error, got: %v", err)
	}
}

func TestTransform_TerminalOnly(t *testing.T) {
	// Transform a terminal node
	tree := &ParseTree{
		Root: &Node{
			Rule:  "number",
			Value: "123",
		},
	}

	transforms := TransformMap{
		"number": func(s string) int {
			i, _ := strconv.Atoi(s)
			return i
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 123 {
		t.Errorf("Expected 123, got %v", result)
	}
}

func TestTransform_InterfaceArgs(t *testing.T) {
	// Test with interface{} arguments
	tree := &ParseTree{
		Root: &Node{
			Rule: "format",
			Children: []*Node{
				{Rule: "name", Value: "Alice"},
				{Rule: "age", Value: "30"},
			},
		},
	}

	transforms := TransformMap{
		"name": Identity,
		"age": func(s string) int {
			i, _ := strconv.Atoi(s)
			return i
		},
		"format": func(name interface{}, age interface{}) string {
			return fmt.Sprintf("%s is %d years old", name, age)
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	expected := "Alice is 30 years old"
	if result != expected {
		t.Errorf("Expected %q, got %v", expected, result)
	}
}

func TestTransform_HelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		helper   TransformFunc
		args     []interface{}
		expected interface{}
	}{
		{
			name:     "Identity",
			helper:   Identity,
			args:     []interface{}{"test"},
			expected: "test",
		},
		{
			name:     "First",
			helper:   First,
			args:     []interface{}{"first", "second", "third"},
			expected: "first",
		},
		{
			name:     "Last",
			helper:   Last,
			args:     []interface{}{"first", "second", "third"},
			expected: "third",
		},
		{
			name:     "Concat",
			helper:   Concat,
			args:     []interface{}{"hello", " ", "world"},
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := callTransform(tt.helper, tt.args)
			if err != nil {
				t.Fatalf("callTransform failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransform_TypeConversion(t *testing.T) {
	// Test automatic type conversion from string to number
	tree := &ParseTree{
		Root: &Node{
			Rule: "add",
			Children: []*Node{
				{Rule: "num", Value: "2.5"},
				{Rule: "num", Value: "3.7"},
			},
		},
	}

	transforms := TransformMap{
		// No transform for "num" - it stays as string
		"add": func(a, b float64) float64 {
			return a + b
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Should automatically convert strings to float64
	expected := 6.2
	if diff := result.(float64) - expected; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestTransform_DeeplyNested(t *testing.T) {
	// Test deeply nested tree structure
	// expr(term(factor(number("5"))))
	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Children: []*Node{
				{
					Rule: "term",
					Children: []*Node{
						{
							Rule: "factor",
							Children: []*Node{
								{Rule: "number", Value: "5"},
							},
						},
					},
				},
			},
		},
	}

	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		// Other rules have no transforms - should pass through
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 5.0 {
		t.Errorf("Expected 5.0, got %v", result)
	}
}

func TestTransform_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		tree        *ParseTree
		transforms  TransformMap
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "NilTree",
			tree:        nil,
			transforms:  TransformMap{},
			shouldError: true,
			errorMsg:    "nil tree",
		},
		{
			name: "WrongArgumentCount",
			tree: &ParseTree{
				Root: &Node{
					Rule: "test",
					Children: []*Node{
						{Rule: "a", Value: "1"},
					},
				},
			},
			transforms: TransformMap{
				"test": func(a, b int) int { return a + b },
			},
			shouldError: true,
			errorMsg:    "expects 2 arguments",
		},
		{
			name: "NotAFunction",
			tree: &ParseTree{
				Root: &Node{Rule: "test", Value: "value"},
			},
			transforms: TransformMap{
				"test": "not a function",
			},
			shouldError: true,
			errorMsg:    "must be a function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Transform(tt.tree, tt.transforms)
			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}
