package parse

import (
	"fmt"
	"strings"
	"testing"
)

func TestTransform_ErrorHandling_ErrorPropagation(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "div",
			Children: []*Node{
				{Rule: "number", Value: "10"},
				{Rule: "number", Value: "0"},
			},
		},
		Input: "10 / 0",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("parse error")
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	// Check if it's a TransformError
	if te, ok := AsTransformError(err); ok {
		if te.Rule != "number" {
			t.Errorf("Expected rule 'number', got %q", te.Rule)
		}
	} else {
		t.Error("Expected TransformError")
	}
}

func TestTransform_ErrorHandling_Position(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "invalid",
			Line: 5,
			Column: 10,
			Start: 42,
			End: 49,
		},
		Input: "some text with invalid number here",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("invalid number")
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	if te, ok := AsTransformError(err); ok {
		if te.Line != 5 {
			t.Errorf("Expected line 5, got %d", te.Line)
		}
		if te.Column != 10 {
			t.Errorf("Expected column 10, got %d", te.Column)
		}
		if !te.HasPosition() {
			t.Error("Expected HasPosition() to return true")
		}
		pos := te.Position()
		if !strings.Contains(pos, "line 5") {
			t.Errorf("Expected position to contain 'line 5', got %q", pos)
		}
	}
}

func TestTransform_ErrorHandling_PanicRecovery(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "42",
			Line: 1,
			Column: 1,
		},
		Input: "42",
	}

	transforms := TransformMap{
		"number": func(s string) int {
			panic("test panic")
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error from panic")
	}

	if te, ok := AsTransformError(err); ok {
		if !te.IsPanic() {
			t.Error("Expected IsPanic() to return true")
		}
		if te.PanicValue == nil {
			t.Error("Expected PanicValue to be set")
		}
		if len(te.PanicStack) == 0 {
			t.Error("Expected PanicStack to be set")
		}
	}
}

func TestTransform_ErrorHandling_MultiPass(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "invalid",
		},
		Input: "invalid",
	}

	pass1 := TransformMap{
		"number": func(s string) int {
			return 0
		},
	}

	pass2 := TransformMap{
		"_transformed": func(val interface{}) int {
			panic("panic in pass 2")
		},
	}

	_, err := TransformMultiPass(tree, []TransformMap{pass1, pass2})
	if err == nil {
		t.Fatal("Expected error")
	}

	if te, ok := AsTransformError(err); ok {
		if te.PassNumber != 2 {
			t.Errorf("Expected pass 2, got %d", te.PassNumber)
		}
	}
}

func TestTransform_ErrorHandling_NestedErrors(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "expr",
			Children: []*Node{
				{Rule: "number", Value: "5"},
			},
		},
		Input: "5",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("inner error")
		},
		"expr": func(ctx *TransformContext, num interface{}) (int, error) {
			return 0, fmt.Errorf("outer error: %w", fmt.Errorf("wrapped"))
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	// Should be able to unwrap
	original := err
	for original != nil {
		if te, ok := AsTransformError(original); ok {
			original = te.Unwrap()
		} else {
			break
		}
	}
}

func TestTransform_ErrorHandling_Unwrap(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "invalid",
		},
		Input: "invalid",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("original error")
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	if te, ok := AsTransformError(err); ok {
		original := te.Unwrap()
		if original == nil {
			t.Error("Expected Unwrap() to return original error")
		}
		if original.Error() != "original error" {
			t.Errorf("Expected 'original error', got %q", original.Error())
		}
	}
}

func TestTransform_ErrorHandling_FormattedOutput(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "invalid",
			Line: 5,
			Column: 10,
			Start: 42,
			End: 49,
		},
		Input: "some text with invalid number here",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("invalid number format")
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	if te, ok := AsTransformError(err); ok {
		formatted := te.FormatError()
		if !strings.Contains(formatted, "Rule:") {
			t.Error("Formatted error should contain 'Rule:'")
		}
		if !strings.Contains(formatted, "Position:") {
			t.Error("Formatted error should contain 'Position:'")
		}
		if !strings.Contains(formatted, "invalid number format") {
			t.Error("Formatted error should contain original error message")
		}
	}
}

func TestTransform_ErrorHandling_HelperFunctions(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "invalid",
		},
		Input: "invalid",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("test error")
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	if !IsTransformError(err) {
		t.Error("IsTransformError should return true")
	}

	te := GetTransformError(err)
	if te == nil {
		t.Fatal("GetTransformError should not return nil")
	}
	if te.Rule != "number" {
		t.Errorf("Expected rule 'number', got %q", te.Rule)
	}
}

