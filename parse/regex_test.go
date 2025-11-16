package parse

import (
	"testing"

	"github.com/wbrown/ebnf"
)

// Helper function to extract the value from a parse tree
func getTreeValue(node *Node) string {
	if node.Value != "" {
		return node.Value
	}
	// If this node has no value, concatenate children values
	var result string
	for _, child := range node.Children {
		result += getTreeValue(child)
	}
	return result
}

func TestParserRegexSupport(t *testing.T) {
	tests := []struct {
		name     string
		grammar  string
		input    string
		start    string
		wantErr  bool
		expected string // expected value if no error
	}{
		{
			name:     "simple identifier regex",
			grammar:  `identifier = #"[a-zA-Z][a-zA-Z0-9_]*" ;`,
			input:    "myVariable123",
			start:    "identifier",
			wantErr:  false,
			expected: "myVariable123",
		},
		{
			name:     "whitespace regex",
			grammar:  `ws = #"\s+" ;`,
			input:    "   \t\n  ",
			start:    "ws",
			wantErr:  false,
			expected: "   \t\n  ",
		},
		{
			name:     "number regex",
			grammar:  `number = #"[0-9]+(\.[0-9]+)?" ;`,
			input:    "123.456",
			start:    "number",
			wantErr:  false,
			expected: "123.456",
		},
		{
			name:     "hidden regex",
			grammar:  `line = <#"\s*"> word <#"\s*"> ; word = #"\w+" ;`,
			input:    "  hello  ",
			start:    "line",
			wantErr:  false,
			expected: "hello", // whitespace should be hidden
		},
		{
			name:     "regex in choice",
			grammar:  `token = #"[0-9]+" | #"[a-zA-Z]+" ;`,
			input:    "123",
			start:    "token",
			wantErr:  false,
			expected: "123",
		},
		{
			name:     "regex in choice (second option)",
			grammar:  `token = #"[0-9]+" | #"[a-zA-Z]+" ;`,
			input:    "abc",
			start:    "token",
			wantErr:  false,
			expected: "abc",
		},
		{
			name:     "regex no match",
			grammar:  `number = #"[0-9]+" ;`,
			input:    "abc",
			start:    "number",
			wantErr:  true,
		},
		{
			name:     "complex regex",
			grammar:  `email = #"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}" ;`,
			input:    "user@example.com",
			start:    "email",
			wantErr:  false,
			expected: "user@example.com",
		},
		{
			name:     "regex with repetition",
			grammar:  `words = word+ ; word = #"\w+" #"\s+" ;`,
			input:    "hello world test ",
			start:    "words",
			wantErr:  false,
		},
		{
			name:     "regex with optional",
			grammar:  `line = #"\w+" (#"\s+" #"\w+")? ;`,
			input:    "hello world",
			start:    "line",
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse the grammar
			grammar, err := ebnf.ParseString(test.grammar)
			if err != nil {
				t.Fatalf("failed to parse grammar: %v", err)
			}

			// Create parser and parse input
			parser := New(grammar)
			tree, err := parser.Parse(test.input, test.start)

			if test.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tree == nil {
				t.Error("expected parse tree but got nil")
				return
			}

			// For simple cases, check the value
			if test.expected != "" && tree.Root != nil {
				// Get the actual value from the tree (might be nested)
				actualValue := getTreeValue(tree.Root)
				if actualValue != test.expected {
					t.Errorf("expected value %q, got %q", test.expected, actualValue)
				}
			}
		})
	}
}

func TestRegexCharacterClasses(t *testing.T) {
	// Test that we can now use regex instead of problematic character classes
	tests := []struct {
		name    string
		grammar string
		input   string
		start   string
	}{
		{
			name: "identifier with regex instead of char classes",
			grammar: `
				var_name = #"[a-zA-Z][a-zA-Z0-9_]*" ;
			`,
			input: "myVar_123",
			start: "var_name",
		},
		{
			name: "text content with negated character class as regex",
			grammar: `
				text = #"[^$@\[\r\n]+" ;
			`,
			input: "Hello world!",
			start: "text",
		},
		{
			name: "mixed alphanumeric",
			grammar: `
				alnum = #"[a-zA-Z0-9_]+" ;
			`,
			input: "test_123_ABC",
			start: "alnum",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grammar, err := ebnf.ParseString(test.grammar)
			if err != nil {
				t.Fatalf("failed to parse grammar: %v", err)
			}

			parser := New(grammar)
			tree, err := parser.Parse(test.input, test.start)
			if err != nil {
				t.Errorf("parse error: %v", err)
			}
			if tree == nil {
				t.Error("expected parse tree but got nil")
			}
		})
	}
}
