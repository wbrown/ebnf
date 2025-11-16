package parse

import (
	"testing"
)

func TestTransform_Sibling_NextSibling(t *testing.T) {
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
			next := ctx.NextSibling()
			nextText := ""
			if next != nil {
				nextText = next.Value
			}

			return map[string]interface{}{
				"text":     text,
				"nextText": nextText,
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	items := result.([]interface{})
	first := items[0].(map[string]interface{})
	if first["nextText"].(string) != "second" {
		t.Errorf("Expected nextText 'second', got %q", first["nextText"])
	}

	last := items[2].(map[string]interface{})
	if last["nextText"].(string) != "" {
		t.Errorf("Expected empty nextText for last item, got %q", last["nextText"])
	}
}

func TestTransform_Sibling_PrevSibling(t *testing.T) {
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
			prev := ctx.PrevSibling()
			prevText := ""
			if prev != nil {
				prevText = prev.Value
			}

			return map[string]interface{}{
				"text":     text,
				"prevText": prevText,
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	items := result.([]interface{})
	first := items[0].(map[string]interface{})
	if first["prevText"].(string) != "" {
		t.Errorf("Expected empty prevText for first item, got %q", first["prevText"])
	}

	last := items[2].(map[string]interface{})
	if last["prevText"].(string) != "second" {
		t.Errorf("Expected prevText 'second', got %q", last["prevText"])
	}
}

func TestTransform_Sibling_IsFirstIsLast(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "list",
			Children: []*Node{
				{Rule: "item", Value: "only"},
			},
		},
		Input: "only",
	}

	transforms := TransformMap{
		"item": func(ctx *TransformContext, text string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"text":    text,
				"isFirst": ctx.IsFirst(),
				"isLast":  ctx.IsLast(),
				"count":   ctx.SiblingCount(),
			}, nil
		},
		"list": func(ctx *TransformContext, items ...interface{}) []interface{} {
			return items
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	item := items[0].(map[string]interface{})
	if !item["isFirst"].(bool) {
		t.Error("Single item should have isFirst=true")
	}
	if !item["isLast"].(bool) {
		t.Error("Single item should have isLast=true")
	}
	if item["count"].(int) != 1 {
		t.Errorf("Expected count 1, got %d", item["count"])
	}
}

func TestTransform_Sibling_SingleNode(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "root",
			Children: []*Node{
				{Rule: "item", Value: "single"},
			},
		},
		Input: "single",
	}

	transforms := TransformMap{
		"item": func(ctx *TransformContext, text string) (map[string]interface{}, error) {
			// Root node has no siblings
			return map[string]interface{}{
				"text":    text,
				"isFirst": ctx.IsFirst(),
				"isLast":  ctx.IsLast(),
			}, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	item := result.(map[string]interface{})
	if !item["isFirst"].(bool) {
		t.Error("Single child should have isFirst=true")
	}
	if !item["isLast"].(bool) {
		t.Error("Single child should have isLast=true")
	}
}

func TestTransform_Sibling_Grouping(t *testing.T) {
	// Test grouping related nodes using sibling access
	tree := &ParseTree{
		Root: &Node{
			Rule: "if_chain",
			Children: []*Node{
				{Rule: "if", Value: "condition1"},
				{Rule: "elseif", Value: "condition2"},
				{Rule: "elseif", Value: "condition3"},
				{Rule: "else", Value: ""},
			},
		},
		Input: "if elseif elseif else",
	}

	transforms := TransformMap{
		"if": func(ctx *TransformContext, cond string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"type": "if",
				"cond": cond,
			}, nil
		},
		"elseif": func(ctx *TransformContext, cond string) (map[string]interface{}, error) {
			// Check if this is the first elseif
			isFirstElseif := ctx.PrevSibling() != nil && ctx.PrevSibling().Rule == "if"
			return map[string]interface{}{
				"type":        "elseif",
				"cond":        cond,
				"isFirst":     isFirstElseif,
			}, nil
		},
		"else": func(ctx *TransformContext) (map[string]interface{}, error) {
			return map[string]interface{}{
				"type": "else",
			}, nil
		},
		"if_chain": func(ctx *TransformContext, branches ...interface{}) ([]interface{}, error) {
			return branches, nil
		},
	}

	result, err := Transform(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	branches := result.([]interface{})
	if len(branches) != 4 {
		t.Fatalf("Expected 4 branches, got %d", len(branches))
	}

	firstElseif := branches[1].(map[string]interface{})
	if !firstElseif["isFirst"].(bool) {
		t.Error("First elseif should have isFirst=true")
	}
}
