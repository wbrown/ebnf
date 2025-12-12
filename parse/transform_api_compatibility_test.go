package parse

import (
	"strconv"
	"testing"
)

// TestTransform_API_Compatibility tests that all function signature styles work together
func TestTransform_API_Compatibility(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Children: []*Node{
				{Rule: "number", Value: "5", Line: 1, Column: 1, Start: 0, End: 1},
				{Rule: "string", Value: "hello", Line: 1, Column: 3, Start: 2, End: 7},
				{Rule: "bool", Value: "true", Line: 1, Column: 9, Start: 8, End: 12},
			},
		},
		Input: "5 hello true",
	}

	// Mix all signature styles in the same TransformMap
	transforms := TransformMap{
		// Old style
		"string": func(s string) string {
			return s
		},
		// Node-aware
		"number": func(node *Node, s string) (int, error) {
			val, _ := strconv.Atoi(s)
			if node.Line != 1 {
				t.Errorf("Expected line 1, got %d", node.Line)
			}
			return val, nil
		},
		// Context-aware
		"bool": func(ctx *TransformContext, s string) bool {
			if ctx.Node == nil {
				t.Error("ctx.Node should not be nil")
			}
			return s == "true"
		},
		// Combined
		"expr": func(ctx *TransformContext, node *Node, args ...interface{}) ([]interface{}, error) {
			if ctx.Node == nil {
				t.Error("ctx.Node should not be nil")
			}
			if node == nil {
				t.Error("node should not be nil")
			}
			return args, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	results := result.([]interface{})
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	if results[0].(int) != 5 {
		t.Errorf("Expected 5, got %v", results[0])
	}
	if results[1].(string) != "hello" {
		t.Errorf("Expected 'hello', got %v", results[1])
	}
	if !results[2].(bool) {
		t.Errorf("Expected true, got %v", results[2])
	}
}

func TestTransform_API_BackwardCompatibility(t *testing.T) {
	// Test that old code still works
	tree := &ParseTree{
		Root: &Node{
			Rule: "add",
			Children: []*Node{
				{Rule: "number", Value: "2"},
				{Rule: "number", Value: "3"},
			},
		},
		Input: "2 + 3",
	}

	// Old-style transforms (no node, no context)
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
