package parse

import (
	"strconv"
	"testing"

	"github.com/wbrown/ebnf"
)

// TestTransform_ArithmeticGrammar tests transformation with the arithmetic grammar
func TestTransform_ArithmeticGrammar(t *testing.T) {
	// Load the arithmetic grammar (S-expression-like structure with right recursion)
	grammar, err := ebnf.LoadGrammar("../examples/arithmetic.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	parser := New(grammar)

	// Define transformation rules - super clean with the new grammar!
	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 { return a / b },
		"neg": func(a float64) float64 { return -a },
	}

	tests := []struct {
		input    string
		expected float64
	}{
		{"5", 5.0},
		{"-5", -5.0},
		{"2 + 3", 5.0},
		{"2 - 3", -1.0},
		{"2 * 3", 6.0},
		{"10 / 2", 5.0},
		{"2 + 3 * 4", 14.0},
		{"(2 + 3) * 4", 20.0},
		{"10 / 2 - 3", 2.0},
		{"1.5 + 2.5 * 2", 6.5},
		{"-5 + 3", -2.0},
		{"10 / -2", -5.0},
		{"2 * -3 + 4", -2.0},
		{"-(2 + 3)", -5.0},
		// Note: Right associative due to right recursion
		{"100 - 50 - 10", 60.0}, // 100 - (50 - 10) = 60, not (100 - 50) - 10 = 40
		{"2 * 3 * 4", 24.0},
		{"(1 + 2) * (3 + 4)", 21.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Parse
			tree, err := parser.Parse(tt.input, "expr")
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Transform/evaluate
			result, err := Transform(tree, transforms)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			resultFloat := result.(float64)
			if resultFloat != tt.expected {
				t.Errorf("Expected %.2f, got %.2f", tt.expected, resultFloat)
			}
		})
	}
}

// TestTransform_OperationNodes verifies that operations become their own nodes
func TestTransform_OperationNodes(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("../examples/arithmetic.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	parser := New(grammar)

	// Parse "2 + 3 * 4" - should be expr -> add(2, mul(3, 4))
	tree, err := parser.Parse("2 + 3 * 4", "expr")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Root is expr wrapper
	if tree.Root.Rule != "expr" {
		t.Errorf("Expected root to be 'expr', got %q", tree.Root.Rule)
	}

	// First child of expr should be the add operation
	if len(tree.Root.Children) < 1 {
		t.Fatal("expr node should have at least 1 child")
	}
	addNode := tree.Root.Children[0]
	if addNode.Rule != "add" {
		t.Errorf("Expected first child to be 'add', got %q", addNode.Rule)
	}

	// Add should have 2 children: number(2) and mul(...)
	if len(addNode.Children) < 2 {
		t.Fatal("add node should have 2 children")
	}
	if addNode.Children[0].Rule != "number" {
		t.Errorf("Expected add's first child to be 'number', got %q", addNode.Children[0].Rule)
	}
	if addNode.Children[1].Rule != "mul" {
		t.Errorf("Expected add's second child to be 'mul', got %q", addNode.Children[1].Rule)
	}

	// Verify the transformation works
	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		"add": func(a, b float64) float64 { return a + b },
		"mul": func(a, b float64) float64 { return a * b },
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.(float64) != 14.0 {
		t.Errorf("Expected 14.0, got %.2f", result.(float64))
	}
}

// TestTransform_UnaryNegation tests unary negation support
func TestTransform_UnaryNegation(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("../examples/arithmetic.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	parser := New(grammar)
	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		"neg": func(a float64) float64 { return -a },
		"add": func(a, b float64) float64 { return a + b },
	}

	tests := []struct {
		input    string
		expected float64
	}{
		{"-5", -5.0},
		{"-5 + 3", -2.0},
		{"-(2 + 3)", -5.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tree, err := parser.Parse(tt.input, "expr")
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			result, err := Transform(tree, transforms)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			if result.(float64) != tt.expected {
				t.Errorf("Expected %.2f, got %.2f", tt.expected, result.(float64))
			}
		})
	}
}
