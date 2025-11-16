package parse

import (
	"fmt"
	"strings"
	"testing"
)

func TestTransformError_API_ConvenienceMethods(t *testing.T) {
	te := &TransformError{
		Rule:   "number",
		Line:   5,
		Column: 10,
		Start:  42,
		End:    49,
		Input:  "test input",
		Err:    fmt.Errorf("test error"),
	}

	if !te.HasPosition() {
		t.Error("HasPosition should return true")
	}

	pos := te.Position()
	if pos != "line 5, column 10" {
		t.Errorf("Expected 'line 5, column 10', got %q", pos)
	}

	if te.IsPanic() {
		t.Error("IsPanic should return false for non-panic error")
	}

	te.PanicValue = "test panic"
	if !te.IsPanic() {
		t.Error("IsPanic should return true for panic error")
	}
}

func TestTransformError_API_HelperFunctions(t *testing.T) {
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

	// Test AsTransformError
	te, ok := AsTransformError(err)
	if !ok {
		t.Fatal("AsTransformError should return true")
	}
	if te == nil {
		t.Fatal("AsTransformError should return non-nil TransformError")
	}

	// Test IsTransformError
	if !IsTransformError(err) {
		t.Error("IsTransformError should return true")
	}

	// Test GetTransformError
	te2 := GetTransformError(err)
	if te2 == nil {
		t.Fatal("GetTransformError should return non-nil")
	}
	if te2.Rule != "number" {
		t.Errorf("Expected rule 'number', got %q", te2.Rule)
	}
}

func TestTransformError_API_UsageExample(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "invalid",
			Line: 5,
			Column: 10,
			Start: 20,
			End: 27,
		},
		Input: "some text with invalid number here",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("invalid number: %q", s)
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	// Example usage pattern
	if te, ok := AsTransformError(err); ok {
		// Check position
		if te.HasPosition() {
			pos := te.Position()
			if pos == "" {
				t.Error("Position should not be empty")
			}
		}

		// Get source snippet
		snippet := te.GetSourceSnippet()
		if snippet != "" {
			// Snippet should contain the error location
			_ = snippet
		}

		// Format full error
		formatted := te.FormatError()
		if !strings.Contains(formatted, "Rule:") {
			t.Error("Formatted error should contain rule information")
		}
	}
}

