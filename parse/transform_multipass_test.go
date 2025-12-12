package parse

import (
	"fmt"
	"strconv"
	"testing"
)

func TestTransform_MultiPass_Basic(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "document",
			Children: []*Node{
				{Rule: "number", Value: "5"},
			},
		},
		Input: "5",
	}

	// Pass 1: Convert to int
	pass1 := TransformMap{
		"number": func(s string) int {
			val, _ := strconv.Atoi(s)
			return val
		},
	}

	// Pass 2: Multiply by 2
	pass2 := TransformMap{
		"_transformed": func(val interface{}) int {
			if i, ok := val.(int); ok {
				return i * 2
			}
			return 0
		},
	}

	result, err := TransformMultiPass(tree, []TransformMap{pass1, pass2})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 10 {
		t.Errorf("Expected 10, got %v", result)
	}
}

func TestTransform_MultiPass_Grouping(t *testing.T) {
	// Test grouping if/elseif/else chains across passes
	tree := &ParseTree{
		Root: &Node{
			Rule: "if_chain",
			Children: []*Node{
				{Rule: "if", Value: "x > 0"},
				{Rule: "elseif", Value: "x < 0"},
				{Rule: "else", Value: ""},
			},
		},
		Input: "if elseif else",
	}

	// Pass 1: Group into Conditional
	pass1 := TransformMap{
		"if": func(ctx *TransformContext, cond string) map[string]interface{} {
			return map[string]interface{}{"type": "if", "cond": cond}
		},
		"elseif": func(ctx *TransformContext, cond string) map[string]interface{} {
			return map[string]interface{}{"type": "elseif", "cond": cond}
		},
		"else": func(ctx *TransformContext) map[string]interface{} {
			return map[string]interface{}{"type": "else"}
		},
		"if_chain": func(ctx *TransformContext, branches ...interface{}) []interface{} {
			return branches
		},
	}

	// Pass 2: Process grouped branches
	pass2 := TransformMap{
		"_transformed": func(val interface{}) interface{} {
			return val
		},
	}

	result, err := TransformMultiPass(tree, []TransformMap{pass1, pass2})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	branches := result.([]interface{})
	if len(branches) != 3 {
		t.Fatalf("Expected 3 branches, got %d", len(branches))
	}
}

func TestTransform_MultiPass_TypePreservation(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule:  "number",
			Value: "42",
		},
		Input: "42",
	}

	// Pass 1: Convert to int
	pass1 := TransformMap{
		"number": func(s string) int {
			val, _ := strconv.Atoi(s)
			return val
		},
	}

	// Pass 2: Should receive int, not string
	pass2 := TransformMap{
		"_transformed": func(val interface{}) int {
			if i, ok := val.(int); ok {
				return i * 2
			}
			t.Errorf("Expected int, got %T", val)
			return 0
		},
	}

	result, err := TransformMultiPass(tree, []TransformMap{pass1, pass2})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result != 84 {
		t.Errorf("Expected 84, got %v", result)
	}
}

func TestTransform_MultiPass_ThreePasses(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Children: []*Node{
				{Rule: "number", Value: "5"},
			},
		},
		Input: "5",
	}

	// Pass 1: String to int
	pass1 := TransformMap{
		"number": func(s string) int {
			val, _ := strconv.Atoi(s)
			return val
		},
	}

	// Pass 2: Int to float
	pass2 := TransformMap{
		"_transformed": func(val interface{}) float64 {
			if i, ok := val.(int); ok {
				return float64(i)
			}
			return 0
		},
	}

	// Pass 3: Float to string
	pass3 := TransformMap{
		"_transformed": func(val interface{}) string {
			if f, ok := val.(float64); ok {
				return strconv.FormatFloat(f, 'f', 1, 64)
			}
			return "0"
		},
	}

	result, err := TransformMultiPass(tree, []TransformMap{pass1, pass2, pass3})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// TransformMultiPass should unwrap _transformed nodes
	resultStr := ""
	if node, ok := result.(*Node); ok && node.Rule == "_transformed" {
		if node.TransformedValue != nil {
			resultStr = fmt.Sprintf("%v", node.TransformedValue)
		} else {
			resultStr = node.Value
		}
	} else {
		resultStr = fmt.Sprintf("%v", result)
	}

	if resultStr != "5.0" {
		t.Errorf("Expected '5.0', got %v (result: %v)", resultStr, result)
	}
}
