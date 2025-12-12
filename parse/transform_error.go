package parse

import (
	"fmt"
	"runtime/debug"
)

// TransformError provides detailed context about transform failures
type TransformError struct {
	// Original error from transform function
	Err error

	// Node information
	Rule   string // Grammar rule name
	Line   int    // Source line number
	Column int    // Source column number
	Start  int    // Start position in input
	End    int    // End position in input

	// Source text information
	Input         string // Original input text (for extracting snippets)
	SourceSnippet string // Extracted source text snippet (computed on demand)

	// Transform context
	TransformRule string // Name of transform function (usually same as Rule)
	PassNumber    int    // Pass number for multi-pass (0 = single pass)

	// Context path (parent chain for debugging)
	ContextPath []string // Path from root to this node

	// Panic information (if error came from panic)
	PanicValue interface{}
	PanicStack []byte
}

// Error returns a formatted error message including rule name, position, and source snippet.
func (e *TransformError) Error() string {
	msg := fmt.Sprintf("transform error in rule %q", e.Rule)

	if e.TransformRule != "" && e.TransformRule != e.Rule {
		msg += fmt.Sprintf(" (transform: %q)", e.TransformRule)
	}

	if e.Line > 0 {
		msg += fmt.Sprintf(" at line %d, column %d", e.Line, e.Column)
	}

	if e.PassNumber > 0 {
		msg += fmt.Sprintf(" (pass %d)", e.PassNumber)
	}

	// Include source snippet if available
	snippet := e.getSourceSnippet()
	if snippet != "" {
		msg += fmt.Sprintf("\n  source: %q", snippet)
	}

	if e.Err != nil {
		msg += "\n  error: " + e.Err.Error()
	} else if e.PanicValue != nil {
		msg += fmt.Sprintf("\n  panic: %v", e.PanicValue)
	}

	return msg
}

// Unwrap returns the underlying error
func (e *TransformError) Unwrap() error {
	return e.Err
}

// GetSourceSnippet returns the source text snippet with highlighting around the error location.
func (e *TransformError) GetSourceSnippet() string {
	return e.getSourceSnippet()
}

// HasPosition returns true if this error has position information
func (e *TransformError) HasPosition() bool {
	return e.Line > 0 && e.Column > 0
}

// Position returns the position as a formatted string
func (e *TransformError) Position() string {
	if !e.HasPosition() {
		return ""
	}
	return fmt.Sprintf("line %d, column %d", e.Line, e.Column)
}

// IsPanic returns true if this error came from a panic
func (e *TransformError) IsPanic() bool {
	return e.PanicValue != nil
}

// FormatError creates a formatted error message with full context
func (e *TransformError) FormatError() string {
	var buf string

	if len(e.ContextPath) > 0 {
		buf += fmt.Sprintf("Context: %v\n", e.ContextPath)
	}

	buf += fmt.Sprintf("Rule: %q\n", e.Rule)

	if e.Line > 0 {
		buf += fmt.Sprintf("Position: line %d, column %d (offset %d-%d)\n",
			e.Line, e.Column, e.Start, e.End)
	}

	if e.PassNumber > 0 {
		buf += fmt.Sprintf("Pass: %d\n", e.PassNumber)
	}

	// Include source snippet with highlighting
	snippet := e.getSourceSnippet()
	if snippet != "" {
		buf += fmt.Sprintf("Source:\n%s\n", snippet)
	}

	if e.Err != nil {
		buf += fmt.Sprintf("Error: %v\n", e.Err)
	} else if e.PanicValue != nil {
		buf += fmt.Sprintf("Panic: %v\n", e.PanicValue)
		if len(e.PanicStack) > 0 {
			buf += fmt.Sprintf("Stack:\n%s\n", string(e.PanicStack))
		}
	}

	return buf
}

// getSourceSnippet extracts the source text snippet for this error
func (e *TransformError) getSourceSnippet() string {
	if e.SourceSnippet != "" {
		return e.SourceSnippet
	}

	if e.Input == "" || e.Start < 0 || e.End < e.Start {
		return ""
	}

	// Extract snippet with context (show surrounding lines if possible)
	start := e.Start
	end := e.End

	// Ensure valid bounds
	if start < 0 {
		start = 0
	}
	if end > len(e.Input) {
		end = len(e.Input)
	}
	if start >= end {
		return ""
	}

	// Try to include some context (up to 20 chars before/after)
	contextBefore := 20
	contextAfter := 20

	if start > contextBefore {
		start -= contextBefore
	} else {
		start = 0
	}

	if end+contextAfter < len(e.Input) {
		end += contextAfter
	} else {
		end = len(e.Input)
	}

	// Final bounds check
	if start >= len(e.Input) {
		return ""
	}
	if end > len(e.Input) {
		end = len(e.Input)
	}
	if start >= end {
		return ""
	}

	snippet := e.Input[start:end]

	// If snippet is long, truncate and add ellipsis
	maxLen := 100
	if len(snippet) > maxLen {
		// Try to center the error in the snippet
		errorLen := e.End - e.Start
		if errorLen < maxLen {
			// Center the error
			halfContext := (maxLen - errorLen) / 2
			errorStart := e.Start - start
			newStart := errorStart - halfContext
			if newStart < 0 {
				newStart = 0
			}
			newEnd := newStart + maxLen
			if newEnd > len(e.Input[start:end]) {
				newEnd = len(e.Input[start:end])
				newStart = newEnd - maxLen
				if newStart < 0 {
					newStart = 0
				}
			}
			snippet = e.Input[start+newStart : start+newEnd]
			if newStart > 0 {
				snippet = "..." + snippet
			}
			if start+newEnd < len(e.Input) {
				snippet = snippet + "..."
			}
		} else {
			snippet = snippet[:maxLen] + "..."
		}
	}

	// Highlight the error portion with carets if it's visible
	errorStartInSnippet := e.Start - start
	errorEndInSnippet := e.End - start

	if errorStartInSnippet >= 0 && errorEndInSnippet <= len(snippet) {
		// Add visual indicator
		before := snippet[:errorStartInSnippet]
		errorText := snippet[errorStartInSnippet:errorEndInSnippet]
		after := snippet[errorEndInSnippet:]

		// Create underline with carets
		underline := ""
		for i := 0; i < len(before); i++ {
			underline += " "
		}
		for i := 0; i < len(errorText); i++ {
			underline += "^"
		}

		snippet = fmt.Sprintf("%s%s%s\n%s", before, errorText, after, underline)
	}

	e.SourceSnippet = snippet
	return snippet
}

// wrapTransformError wraps an error with transform context
func wrapTransformError(err error, ctx *TransformContext, ruleName string, passNumber int) *TransformError {
	te := &TransformError{
		Err:           err,
		Rule:          ruleName,
		TransformRule: ruleName,
		PassNumber:    passNumber,
	}

	if ctx != nil {
		if ctx.Node != nil {
			te.Rule = ctx.Node.Rule
			te.Line = ctx.Node.Line
			te.Column = ctx.Node.Column
			te.Start = ctx.Node.Start
			te.End = ctx.Node.End
		}
		if ctx.Tree != nil {
			te.Input = ctx.Tree.Input
		}
		// Build context path
		if ctx.Parent != nil {
			te.ContextPath = buildContextPath(ctx.Parent)
		}
	}

	return te
}

// wrapPanic wraps a panic value with transform context
func wrapPanic(panicValue interface{}, ctx *TransformContext, ruleName string, passNumber int) *TransformError {
	te := &TransformError{
		PanicValue:    panicValue,
		PanicStack:    debug.Stack(),
		Rule:          ruleName,
		TransformRule: ruleName,
		PassNumber:    passNumber,
	}

	if ctx != nil {
		if ctx.Node != nil {
			te.Rule = ctx.Node.Rule
			te.Line = ctx.Node.Line
			te.Column = ctx.Node.Column
			te.Start = ctx.Node.Start
			te.End = ctx.Node.End
		}
		if ctx.Tree != nil {
			te.Input = ctx.Tree.Input
		}
		// Build context path
		if ctx.Parent != nil {
			te.ContextPath = buildContextPath(ctx.Parent)
		}
	}

	return te
}

// buildContextPath builds a path from root to the given node
func buildContextPath(node *Node) []string {
	var path []string
	current := node
	for current != nil {
		path = append([]string{current.Rule}, path...)
		// We don't have parent links in Node, so we can only get the current node
		// For a full path, we'd need to track it during traversal
		break
	}
	return path
}

// AsTransformError extracts a TransformError from an error chain
func AsTransformError(err error) (*TransformError, bool) {
	if err == nil {
		return nil, false
	}
	te, ok := err.(*TransformError)
	return te, ok
}

// IsTransformError checks if an error is a TransformError
func IsTransformError(err error) bool {
	_, ok := AsTransformError(err)
	return ok
}

// GetTransformError extracts a TransformError from an error chain (panics if not found)
func GetTransformError(err error) *TransformError {
	if te, ok := AsTransformError(err); ok {
		return te
	}
	panic("error is not a TransformError")
}
