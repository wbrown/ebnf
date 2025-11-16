package parse

import (
	"strconv"
	"strings"
	"testing"
)

// Number is a test type for node-aware transforms
type Number struct {
	Value int
	Line  int
	Start int
	End   int
}

func TestTransform_NodeAware_BackwardCompatible(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "add",
			Children: []*Node{
				{Rule: "number", Value: "2", Line: 1, Column: 1, Start: 0, End: 1},
				{Rule: "number", Value: "3", Line: 1, Column: 3, Start: 2, End: 3},
			},
		},
		Input: "2 + 3",
	}

	// Old signature (no node parameter) should still work
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

func TestTransform_NodeAware_PositionAccess(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Line: 1,
			Column: 1,
			Start: 0,
			End: 5,
			Children: []*Node{
				{
					Rule: "number",
					Value: "42",
					Line: 1,
					Column: 1,
					Start: 0,
					End: 2,
				},
			},
		},
		Input: "42",
	}

	transforms := TransformMap{
		"number": func(node *Node, s string) (*Number, error) {
			// Verify we can access position info
			if node.Line != 1 {
				t.Errorf("Expected line 1, got %d", node.Line)
			}
			if node.Column != 1 {
				t.Errorf("Expected column 1, got %d", node.Column)
			}
			if node.Start != 0 || node.End != 2 {
				t.Errorf("Expected start=0, end=2, got start=%d, end=%d", node.Start, node.End)
			}

			val, _ := strconv.Atoi(s)
			return &Number{Value: val, Line: node.Line, Start: node.Start, End: node.End}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	num := result.(*Number)
	if num.Line != 1 {
		t.Errorf("Expected line 1 in result, got %d", num.Line)
	}
	if num.Value != 42 {
		t.Errorf("Expected value 42, got %d", num.Value)
	}
}

func TestTransform_NodeAware_SourceTextExtraction(t *testing.T) {
	input := "x != y"
	tree := &ParseTree{
		Root: &Node{
			Rule: "eq_expr",
			Line: 1,
			Column: 1,
			Start: 0,
			End: 6,
			Children: []*Node{
				{Rule: "variable", Value: "x", Start: 0, End: 1},
				{Rule: "variable", Value: "y", Start: 5, End: 6},
			},
		},
		Input: input,
	}

	transforms := TransformMap{
		"variable": func(node *Node, name string) string {
			return name
		},
		"eq_expr": func(node *Node, left, right string) (string, error) {
			// Extract operator from source text between left and right
			leftNode := node.Children[0]
			rightNode := node.Children[1]
			operatorText := input[leftNode.End:rightNode.Start]

			// Determine operator
			op := "="
			if strings.Contains(operatorText, "!") {
				op = "!="
			}

			return op, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != "!=" {
		t.Errorf("Expected '!=', got %v", result)
	}
}

func TestTransform_NodeAware_IndentationDetection(t *testing.T) {
	input := "- Item 1\n  - Nested 1.1\n  - Nested 1.2\n- Item 2"
	tree := &ParseTree{
		Root: &Node{
			Rule: "list",
			Children: []*Node{
				{
					Rule: "item",
					Value: "Item 1",
					Line: 1,
					Start: 2,
					End: 8,
					Children: []*Node{
						{
							Rule: "item",
							Value: "Nested 1.1",
							Line: 2,
							Start: 13,
							End: 24,
						},
						{
							Rule: "item",
							Value: "Nested 1.2",
							Line: 3,
							Start: 27,
							End: 38,
						},
					},
				},
				{
					Rule: "item",
					Value: "Item 2",
					Line: 4,
					Start: 40,
					End: 46,
				},
			},
		},
		Input: input,
	}

	transforms := TransformMap{
		"item": func(node *Node, args ...interface{}) (map[string]interface{}, error) {
			// For items with children, extract text from first child or use Value
			text := node.Value
			if text == "" && len(node.Children) > 0 {
				text = node.Children[0].Value
			}

			// Calculate indentation from source position
			lineStart := 0
			for i := node.Start - 1; i >= 0; i-- {
				if input[i] == '\n' {
					lineStart = i + 1
					break
				}
			}
			indent := node.Start - lineStart

			return map[string]interface{}{
				"text":   text,
				"indent": indent,
				"line":   node.Line,
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 2 {
		t.Fatalf("Expected 2 top-level items, got %d", len(items))
	}

	firstItem := items[0].(map[string]interface{})
	if firstItem["indent"].(int) != 2 {
		t.Errorf("Expected indent 2 for first item, got %d", firstItem["indent"])
	}
}

func TestTransform_NodeAware_WrongArgCount(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "test",
			Children: []*Node{
				{Rule: "a", Value: "1"},
			},
		},
		Input: "1",
	}

	transforms := TransformMap{
		"test": func(node *Node, a, b int) int {
			return a + b
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error for wrong argument count")
	}

	if !strings.Contains(err.Error(), "expects") || !strings.Contains(err.Error(), "arguments") {
		t.Errorf("Error should mention argument count, got: %v", err)
	}
}

func TestTransform_NodeAware_MixedSignatures(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Children: []*Node{
				{Rule: "number", Value: "5", Line: 1, Column: 1, Start: 0, End: 1},
				{Rule: "string", Value: "hello", Line: 1, Column: 3, Start: 2, End: 7},
			},
		},
		Input: "5 hello",
	}

	transforms := TransformMap{
		"number": func(node *Node, s string) int {
			val, _ := strconv.Atoi(s)
			return val
		},
		"string": func(s string) string {
			return s
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	results := result.([]interface{})
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].(int) != 5 {
		t.Errorf("Expected 5, got %v", results[0])
	}
	if results[1].(string) != "hello" {
		t.Errorf("Expected 'hello', got %v", results[1])
	}
}

func TestTransform_NodeAware_TerminalNode(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "123",
			Line: 1,
			Column: 1,
			Start: 0,
			End: 3,
		},
		Input: "123",
	}

	transforms := TransformMap{
		"number": func(node *Node, s string) (*Number, error) {
			val, _ := strconv.Atoi(s)
			return &Number{
				Value: val,
				Line:  node.Line,
				Start: node.Start,
				End:   node.End,
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	num := result.(*Number)
	if num.Value != 123 {
		t.Errorf("Expected 123, got %d", num.Value)
	}
	if num.Line != 1 {
		t.Errorf("Expected line 1, got %d", num.Line)
	}
}

