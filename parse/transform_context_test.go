package parse

import (
	"strconv"
	"testing"
)

func TestTransform_Context_BackwardCompatible(t *testing.T) {
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

	// Old signature should still work
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

func TestTransform_Context_Access(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Children: []*Node{
				{Rule: "number", Value: "42"},
			},
		},
		Input: "42",
	}

	transforms := TransformMap{
		"number": func(ctx *TransformContext, s string) (int, error) {
			// Verify context access
			if ctx.Node == nil {
				t.Fatal("ctx.Node should not be nil")
			}
			if ctx.Node.Rule != "number" {
				t.Errorf("Expected rule 'number', got %q", ctx.Node.Rule)
			}
			if ctx.Tree == nil {
				t.Fatal("ctx.Tree should not be nil")
			}
			if ctx.Input != "42" {
				t.Errorf("Expected input '42', got %q", ctx.Input)
			}

			val, _ := strconv.Atoi(s)
			return val, nil
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

func TestTransform_Context_ParentAccess(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "parent",
			Children: []*Node{
				{Rule: "child", Value: "test"},
			},
		},
		Input: "test",
	}

	transforms := TransformMap{
		"child": func(ctx *TransformContext, text string) (string, error) {
			if ctx.Parent == nil {
				t.Fatal("ctx.Parent should not be nil")
			}
			if ctx.Parent.Rule != "parent" {
				t.Errorf("Expected parent rule 'parent', got %q", ctx.Parent.Rule)
			}
			return text, nil
		},
	}

	_, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
}

func TestTransform_Context_SiblingAccess(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "list",
			Children: []*Node{
				{Rule: "item", Value: "first"},
				{Rule: "item", Value: "second"},
				{Rule: "item", Value: "third"},
			},
		},
		Input: "first second third",
	}

	transforms := TransformMap{
		"item": func(ctx *TransformContext, text string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"text":    text,
				"index":   ctx.Index,
				"isFirst": ctx.IsFirst(),
				"isLast":  ctx.IsLast(),
				"count":   ctx.SiblingCount(),
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(items))
	}

	first := items[0].(map[string]interface{})
	if !first["isFirst"].(bool) {
		t.Error("First item should have isFirst=true")
	}
	if first["isLast"].(bool) {
		t.Error("First item should have isLast=false")
	}

	last := items[2].(map[string]interface{})
	if !last["isLast"].(bool) {
		t.Error("Last item should have isLast=true")
	}
	if last["isFirst"].(bool) {
		t.Error("Last item should have isFirst=false")
	}
}

func TestTransform_Context_StateStorage(t *testing.T) {
	// Use closure to share state across transforms
	var linkRefs = make(map[string]string)

	tree := &ParseTree{
		Root: &Node{
			Rule: "document",
			Children: []*Node{
				{
					Rule: "link_ref",
					Children: []*Node{
						{Rule: "link_id", Value: "example"},
						{Rule: "link_url", Value: "https://example.com"},
					},
				},
				{
					Rule: "link",
					Children: []*Node{
						{Rule: "link_text", Value: "Example"},
						{Rule: "link_id", Value: "example"},
					},
				},
			},
		},
		Input: "[Example](example)",
	}

	transforms := TransformMap{
		"link_id": func(s string) string {
			return s
		},
		"link_url": func(s string) string {
			return s
		},
		"link_text": func(s string) string {
			return s
		},
		"link_ref": func(ctx *TransformContext, id, url string) (string, error) {
			// Store in shared state
			linkRefs[id] = url
			return "", nil
		},
		"link": func(ctx *TransformContext, text, id string) (string, error) {
			// Access from shared state
			url, ok := linkRefs[id]
			if !ok {
				return "", nil
			}
			return text + " -> " + url, nil
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

	linkResult := results[1].(string)
	if linkResult != "Example -> https://example.com" {
		t.Errorf("Expected 'Example -> https://example.com', got %q", linkResult)
	}
}

func TestTransform_Context_CombinedSignature(t *testing.T) {
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
		"number": func(ctx *TransformContext, node *Node, s string) (map[string]interface{}, error) {
			// Both context and node available
			val, _ := strconv.Atoi(s)
			return map[string]interface{}{
				"value":  val,
				"line":   node.Line,
				"column": node.Column,
				"rule":   ctx.Node.Rule,
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	data := result.(map[string]interface{})
	if data["value"].(int) != 42 {
		t.Errorf("Expected value 42, got %v", data["value"])
	}
	if data["line"].(int) != 1 {
		t.Errorf("Expected line 1, got %v", data["line"])
	}
}

