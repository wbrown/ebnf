package parse

import (
	"fmt"
	"testing"
)

// TestTransformError_Demo demonstrates the error output format
func TestTransformError_Demo(t *testing.T) {
	tree := &ParseTree{
		Root: &Node{
			Rule: "number",
			Value: "invalid_number",
			Line: 5,
			Column: 10,
			Start: 42,
			End: 57,
		},
		Input: "some text before invalid_number and some text after",
	}

	transforms := TransformMap{
		"number": func(s string) (int, error) {
			return 0, fmt.Errorf("invalid number format: %q", s)
		},
	}

	_, err := Transform(tree, transforms)
	if err == nil {
		t.Fatal("Expected error")
	}

	// Basic error message
	fmt.Printf("Basic error: %v\n", err)

	// Detailed error info
	if te, ok := AsTransformError(err); ok {
		fmt.Printf("\nDetailed error:\n")
		fmt.Printf("Rule: %s\n", te.Rule)
		fmt.Printf("Position: %s\n", te.Position())
		fmt.Printf("Source snippet:\n%s\n", te.GetSourceSnippet())
		fmt.Printf("\nFormatted error:\n%s\n", te.FormatError())
	}
}

