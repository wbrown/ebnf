package parse

import (
	"strings"
	"testing"
)

func TestTransformError_SourceSnippet_Basic(t *testing.T) {
	input := "This is a test with an error here"
	te := &TransformError{
		Input: input,
		Start: 20,
		End:   25,
		Line:  1,
		Column: 21,
	}

	snippet := te.GetSourceSnippet()
	if snippet == "" {
		t.Error("Expected non-empty snippet")
	}

	// Should contain the error text
	if !strings.Contains(snippet, "error") {
		t.Errorf("Snippet should contain 'error', got: %q", snippet)
	}
}

func TestTransformError_SourceSnippet_Context(t *testing.T) {
	input := "This is a test with an error here in the middle of the text"
	te := &TransformError{
		Input: input,
		Start: 20,
		End:   25,
		Line:  1,
		Column: 21,
	}

	snippet := te.GetSourceSnippet()
	
	// Should include context before and after
	if !strings.Contains(snippet, "test") || !strings.Contains(snippet, "here") {
		t.Errorf("Snippet should include context, got: %q", snippet)
	}
}

func TestTransformError_SourceSnippet_LongInput(t *testing.T) {
	// Create a long input
	input := strings.Repeat("a", 200) + "ERROR" + strings.Repeat("b", 200)
	te := &TransformError{
		Input: input,
		Start: 200,
		End:   205,
		Line:  1,
		Column: 201,
	}

	snippet := te.GetSourceSnippet()
	
	// Should be truncated but still show the error
	if len(snippet) > 200 {
		t.Errorf("Snippet should be truncated, got length %d", len(snippet))
	}
	if !strings.Contains(snippet, "ERROR") {
		t.Errorf("Snippet should contain 'ERROR', got: %q", snippet)
	}
}

func TestTransformError_SourceSnippet_InvalidBounds(t *testing.T) {
	input := "test"
	
	// Test with invalid bounds
	te := &TransformError{
		Input: input,
		Start: -1,
		End:   10,
		Line:  1,
		Column: 1,
	}

	snippet := te.GetSourceSnippet()
	// Should handle gracefully
	_ = snippet
}

func TestTransformError_SourceSnippet_EmptyInput(t *testing.T) {
	te := &TransformError{
		Input: "",
		Start: 0,
		End:   0,
	}

	snippet := te.GetSourceSnippet()
	if snippet != "" {
		t.Errorf("Expected empty snippet for empty input, got: %q", snippet)
	}
}

