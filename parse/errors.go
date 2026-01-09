package parse

import (
	"fmt"
)

// ErrorType represents the type of parse error
type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorExpectedTerminal
	ErrorExpectedEOF
	ErrorUnexpectedEOF
	ErrorNoAltMatched
	ErrorRuleNotFound
	ErrorCharClassMismatch
	ErrorRegexNoMatch
	ErrorNegativeLookaheadFailed
	ErrorPositiveLookaheadFailed
	ErrorInvalidRegex
	ErrorUnknownExpression
	ErrorExpectedAtLeastOne
)

// ParseError is a lightweight error type that defers string formatting
// until Error() is called. This avoids allocating error strings during
// backtracking when errors are created but immediately discarded.
type ParseError struct {
	Type     ErrorType
	Pos      int
	Line     int
	Col      int
	Expected string // Not formatted - raw value
	Got      string // Not formatted - raw value
	RuleName string
	Pattern  string // For regex errors
	Details  string // Additional context
	Wrapped  error  // Can be nil or another error
	AltCount int    // For choice errors
}

// Error implements the error interface with lazy formatting
func (e *ParseError) Error() string {
	switch e.Type {
	case ErrorExpectedTerminal:
		return fmt.Sprintf("expected %q at line %d col %d, got %q",
			e.Expected, e.Line, e.Col, e.Got)

	case ErrorExpectedEOF:
		if e.Details != "" {
			return fmt.Sprintf("expected EOF at line %d col %d: %s", e.Line, e.Col, e.Details)
		}
		return fmt.Sprintf("expected EOF at line %d col %d", e.Line, e.Col)

	case ErrorUnexpectedEOF:
		if e.Expected != "" {
			return fmt.Sprintf("unexpected EOF at line %d col %d, expected %s",
				e.Line, e.Col, e.Expected)
		}
		return fmt.Sprintf("unexpected EOF at line %d col %d", e.Line, e.Col)

	case ErrorNoAltMatched:
		if e.AltCount > 3 {
			return fmt.Sprintf("no alternative matched (tried %d): %v",
				e.AltCount, e.Wrapped)
		}
		if e.Wrapped != nil {
			return fmt.Sprintf("no alternative matched: %v", e.Wrapped)
		}
		return fmt.Sprintf("no alternative matched (tried %d alternatives)", e.AltCount)

	case ErrorRuleNotFound:
		return fmt.Sprintf("rule %q not found", e.RuleName)

	case ErrorCharClassMismatch:
		return fmt.Sprintf("character %q at line %d col %d does not match character class",
			e.Got, e.Line, e.Col)

	case ErrorRegexNoMatch:
		if e.Expected != "" {
			return fmt.Sprintf("regex pattern %q did not match at line %d col %d, got %q",
				e.Pattern, e.Line, e.Col, e.Expected)
		}
		return fmt.Sprintf("regex pattern %q did not match at line %d col %d",
			e.Pattern, e.Line, e.Col)

	case ErrorNegativeLookaheadFailed:
		return fmt.Sprintf("negative lookahead failed at line %d col %d", e.Line, e.Col)

	case ErrorPositiveLookaheadFailed:
		if e.Wrapped != nil {
			return fmt.Sprintf("positive lookahead failed at line %d col %d: %v",
				e.Line, e.Col, e.Wrapped)
		}
		return fmt.Sprintf("positive lookahead failed at line %d col %d", e.Line, e.Col)

	case ErrorInvalidRegex:
		return fmt.Sprintf("invalid regex pattern %q: %v", e.Pattern, e.Wrapped)

	case ErrorUnknownExpression:
		return fmt.Sprintf("unknown expression type: %s", e.Details)

	case ErrorExpectedAtLeastOne:
		return "expected at least one occurrence"

	default:
		if e.Details != "" {
			return e.Details
		}
		if e.Wrapped != nil {
			return e.Wrapped.Error()
		}
		return "parse error"
	}
}

// Unwrap implements error unwrapping for errors.Is/As
func (e *ParseError) Unwrap() error {
	return e.Wrapped
}

// Helper constructors for common error types

func newExpectedTerminalError(expected, got string, line, col int) *ParseError {
	return &ParseError{
		Type:     ErrorExpectedTerminal,
		Expected: expected,
		Got:      got,
		Line:     line,
		Col:      col,
	}
}

func newUnexpectedEOFError(expected string, line, col int) *ParseError {
	return &ParseError{
		Type:     ErrorUnexpectedEOF,
		Expected: expected,
		Line:     line,
		Col:      col,
	}
}

func newRuleNotFoundError(ruleName string) *ParseError {
	return &ParseError{
		Type:     ErrorRuleNotFound,
		RuleName: ruleName,
	}
}

func newCharClassMismatchError(got string, line, col int) *ParseError {
	return &ParseError{
		Type: ErrorCharClassMismatch,
		Got:  got,
		Line: line,
		Col:  col,
	}
}

func newRegexNoMatchError(pattern, got string, line, col int) *ParseError {
	return &ParseError{
		Type:     ErrorRegexNoMatch,
		Pattern:  pattern,
		Expected: got,
		Line:     line,
		Col:      col,
	}
}

func newNoAltMatchedError(altCount int, lastErr error) *ParseError {
	return &ParseError{
		Type:     ErrorNoAltMatched,
		AltCount: altCount,
		Wrapped:  lastErr,
	}
}

func newInvalidRegexError(pattern string, err error) *ParseError {
	return &ParseError{
		Type:    ErrorInvalidRegex,
		Pattern: pattern,
		Wrapped: err,
	}
}

func newUnknownExpressionError(typeName string) *ParseError {
	return &ParseError{
		Type:    ErrorUnknownExpression,
		Details: typeName,
	}
}

func newNegativeLookaheadError(line, col int) *ParseError {
	return &ParseError{
		Type: ErrorNegativeLookaheadFailed,
		Line: line,
		Col:  col,
	}
}

func newPositiveLookaheadError(line, col int, err error) *ParseError {
	return &ParseError{
		Type:    ErrorPositiveLookaheadFailed,
		Line:    line,
		Col:     col,
		Wrapped: err,
	}
}

func newExpectedAtLeastOneError() *ParseError {
	return &ParseError{
		Type: ErrorExpectedAtLeastOne,
	}
}

// wrapRuleError wraps an error with rule context
func wrapRuleError(ruleName string, err error) error {
	// If it's already a ParseError, we can avoid wrapping with fmt.Errorf
	if pe, ok := err.(*ParseError); ok {
		// Just update the rule name if not already set
		if pe.RuleName == "" {
			pe.RuleName = ruleName
		}
		return pe
	}
	// For non-ParseError, wrap with minimal allocation
	return &ParseError{
		Type:     ErrorUnknown,
		RuleName: ruleName,
		Wrapped:  err,
		Details:  fmt.Sprintf("error parsing rule %s", ruleName),
	}
}
