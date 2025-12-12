package parse

import (
	"testing"
)

// TopDownStatement is a test type for top-down transforms
type TopDownStatement struct {
	Type     string
	Children []interface{}
}

func TestTransform_TopDown_Basic(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "scope",
			Children: []*Node{
				{Rule: "statement", Value: "x = 1"},
				{Rule: "statement", Value: "y = 2"},
			},
		},
		Input: "x = 1\ny = 2",
	}

	transforms := TransformMap{
		"scope": func(ctx *TransformContext, children ...*Node) (*TopDownStatement, error) {
			// Process parent before children
			// In top-down, children are passed as *Node initially
			scope := &TopDownStatement{
				Type:     "scope",
				Children: make([]interface{}, len(children)),
			}
			// Return the node so children will be processed
			return scope, nil
		},
		"statement": func(ctx *TransformContext) (string, error) {
			// Terminal node - receive value directly
			return ctx.Node.Value, nil
		},
	}

	result, err := TransformTopDown(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	scope := result.(*TopDownStatement)
	if scope.Type != "scope" {
		t.Errorf("Expected type 'scope', got %q", scope.Type)
	}
}

func TestTransform_TopDown_Scoping(t *testing.T) {
	// Use closure to share scope state
	var currentScope *TopDownStatement

	tree := &ParseTree{
		Root: &Node{
			Rule: "function",
			Children: []*Node{
				{Rule: "name", Value: "foo"},
				{
					Rule: "body",
					Children: []*Node{
						{Rule: "statement", Value: "x = 1"},
					},
				},
			},
		},
		Input: "function foo { x = 1 }",
	}

	transforms := TransformMap{
		"name": func(ctx *TransformContext) string {
			return ctx.Node.Value
		},
		"function": func(ctx *TransformContext, children ...*Node) (*TopDownStatement, error) {
			// Set up scope before processing body
			// In top-down, we receive nodes initially
			currentScope = &TopDownStatement{
				Type: "function",
			}
			return currentScope, nil
		},
		"body": func(ctx *TransformContext, statements ...*Node) []interface{} {
			// Access current scope
			if currentScope == nil {
				t.Error("Expected currentScope to be set")
			}
			// Convert nodes to strings
			result := make([]interface{}, len(statements))
			for i, stmt := range statements {
				result[i] = stmt.Value
			}
			return result
		},
		"statement": func(ctx *TransformContext) string {
			// Verify scope is available
			if currentScope == nil {
				t.Error("Expected currentScope to be set in statement")
			}
			return ctx.Node.Value
		},
	}

	result, err := TransformTopDown(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	_ = result
}

func TestTransform_TopDown_ParentBeforeChildren(t *testing.T) {
	var processingOrder []string

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
		"parent": func(ctx *TransformContext, children ...*Node) (interface{}, error) {
			processingOrder = append(processingOrder, "parent")
			return "parent_result", nil
		},
		"child": func(ctx *TransformContext) (string, error) {
			processingOrder = append(processingOrder, "child")
			return ctx.Node.Value, nil
		},
	}

	_, err := TransformTopDown(tree, transforms)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Verify parent processed before child
	if len(processingOrder) < 2 {
		t.Fatalf("Expected at least 2 processing steps, got %d", len(processingOrder))
	}
	if processingOrder[0] != "parent" {
		t.Errorf("Expected parent first, got %q", processingOrder[0])
	}
}
