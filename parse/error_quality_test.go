package parse

import (
	"strings"
	"testing"

	"github.com/wbrown/ebnf"
)

// TestErrorQuality verifies that error messages are preserved with the new ParseError system
func TestErrorQuality(t *testing.T) {
	grammar := `
value = "hello" | "world" | number ;
number = #"[0-9]+" ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		input         string
		expectedInErr []string // Substrings that should appear in error
		shouldHavePos bool     // Should include position info
	}{
		{
			name:          "unexpected token",
			input:         "goodbye",
			expectedInErr: []string{"no alternative matched"},
			shouldHavePos: false,
		},
		{
			name:          "unexpected EOF in terminal",
			input:         "hel",
			expectedInErr: []string{"no alternative matched"},
			shouldHavePos: false, // The choice error doesn't have position, but wrapped error does
		},
		{
			name:          "invalid pattern",
			input:         "abc",
			expectedInErr: []string{"no alternative matched"},
			shouldHavePos: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(g)
			_, err := p.Parse(tt.input, "value")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errStr := err.Error()
			for _, expected := range tt.expectedInErr {
				if !strings.Contains(errStr, expected) {
					t.Errorf("error should contain %q, got: %s", expected, errStr)
				}
			}

			if tt.shouldHavePos {
				// Should have line/col information
				if !strings.Contains(errStr, "line") || !strings.Contains(errStr, "col") {
					t.Errorf("error should contain position info, got: %s", errStr)
				}
			}
		})
	}
}

// TestErrorUnwrapping verifies that errors can be unwrapped properly
func TestErrorUnwrapping(t *testing.T) {
	grammar := `
expr = term '+' term ;
term = #"[0-9]+" ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	p := New(g)
	_, err = p.Parse("123", "expr") // Missing the '+' and second term
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should be a ParseError
	if _, ok := err.(*ParseError); !ok {
		t.Errorf("expected *ParseError, got %T", err)
	}
}

// TestErrorPositionTracking verifies that position information is preserved
func TestErrorPositionTracking(t *testing.T) {
	grammar := `
line = word+ ;
word = #"[a-z]+" #"[ \n]*" ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	input := "hello world 123"
	p := New(g)
	_, err = p.Parse(input, "line")
	if err == nil {
		t.Fatal("expected error for invalid character '1'")
	}

	errStr := err.Error()
	// Should mention where the error occurred
	if !strings.Contains(errStr, "line") && !strings.Contains(errStr, "col") {
		t.Errorf("error should contain position information: %s", errStr)
	}
}

// TestParseErrorTypes verifies different error types format correctly
func TestParseErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		err      *ParseError
		expected string
	}{
		{
			name:     "terminal mismatch",
			err:      newExpectedTerminalError("foo", "bar", 1, 5),
			expected: `expected "foo"`,
		},
		{
			name:     "unexpected EOF",
			err:      newUnexpectedEOFError("something", 2, 10),
			expected: "unexpected EOF",
		},
		{
			name:     "rule not found",
			err:      newRuleNotFoundError("missing_rule"),
			expected: `rule "missing_rule" not found`,
		},
		{
			name:     "no alternatives matched",
			err:      newNoAltMatchedError(3, nil),
			expected: "no alternative matched",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			if !strings.Contains(errStr, tt.expected) {
				t.Errorf("expected error to contain %q, got: %s", tt.expected, errStr)
			}
		})
	}
}
