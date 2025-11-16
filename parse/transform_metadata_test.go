package parse

import (
	"strconv"
	"testing"
)

func TestTransform_Metadata_Basic(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "42",
			Line: 1,
			Column: 1,
			Start: 0,
			End: 2,
		},
		Input: "42",
	}

	transforms := TransformMap{
		"number": func(node *Node, s string) *TransformResult {
			val, _ := strconv.Atoi(s)
			return &TransformResult{
				Value: val,
				Metadata: map[string]interface{}{
					"source_pos": node.Start,
					"line":       node.Line,
				},
				Node: node,
			}
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestTransform_Metadata_AccessViaContext(t *testing.T) {
	// Use closure to share metadata across transforms
	var childMetadata map[string]interface{}

	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Children: []*Node{
				{Rule: "number", Value: "5", Line: 1, Start: 0, End: 1},
			},
		},
		Input: "5",
	}

	transforms := TransformMap{
		"number": func(node *Node, s string) *TransformResult {
			val, _ := strconv.Atoi(s)
			metadata := map[string]interface{}{
				"line": node.Line,
			}
			// Store in shared closure for parent to access
			childMetadata = metadata
			return &TransformResult{
				Value:    val,
				Metadata: metadata,
			}
		},
		"expr": func(ctx *TransformContext, num interface{}) (map[string]interface{}, error) {
			// Access metadata from child via closure
			line := 0
			if childMetadata != nil {
				if l, ok := childMetadata["line"].(int); ok {
					line = l
				}
			}
			return map[string]interface{}{
				"value": num,
				"line":  line,
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	data := result.(map[string]interface{})
	if data["line"].(int) != 1 {
		t.Errorf("Expected line 1, got %v", data["line"])
	}
}

func TestTransform_Metadata_MixedReturnTypes(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "list",
			Children: []*Node{
				{Rule: "number", Value: "1"},
				{Rule: "string", Value: "hello"},
			},
		},
		Input: "1 hello",
	}

	transforms := TransformMap{
		"number": func(node *Node, s string) *TransformResult {
			val, _ := strconv.Atoi(s)
			return &TransformResult{
				Value: val,
				Metadata: map[string]interface{}{
					"type": "number",
				},
			}
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
}

func TestTransform_Metadata_HelperMethods(t *testing.T) {
	tr := &TransformResult{
		Value: 42,
		Metadata: map[string]interface{}{
			"line":   "5",
			"column": 10,
			"type":   "number",
		},
	}

	if !tr.HasMetadata("line") {
		t.Error("HasMetadata should return true for 'line'")
	}

	if tr.HasMetadata("missing") {
		t.Error("HasMetadata should return false for 'missing'")
	}

	line := tr.GetMetadataString("line", "0")
	if line != "5" {
		t.Errorf("Expected '5', got %q", line)
	}

	column := tr.GetMetadataInt("column", 0)
	if column != 10 {
		t.Errorf("Expected 10, got %d", column)
	}
}

