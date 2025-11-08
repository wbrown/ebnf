package ebnf

import (
	"testing"
)

func TestRegexLexing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple regex",
			input: `#"[a-z]+"`,
			expected: []Token{
				{Type: TokenRegex, Value: "[a-z]+", Line: 1, Col: 1},
				{Type: TokenEOF},
			},
		},
		{
			name:  "regex with escaped quote",
			input: `#"\"hello\""`,
			expected: []Token{
				{Type: TokenRegex, Value: `\"hello\"`, Line: 1, Col: 1},
				{Type: TokenEOF},
			},
		},
		{
			name:  "regex with backslash",
			input: `#"\\s+"`,
			expected: []Token{
				{Type: TokenRegex, Value: `\\s+`, Line: 1, Col: 1},
				{Type: TokenEOF},
			},
		},
		{
			name:  "complex regex",
			input: `#"[a-zA-Z][a-zA-Z0-9_]*"`,
			expected: []Token{
				{Type: TokenRegex, Value: "[a-zA-Z][a-zA-Z0-9_]*", Line: 1, Col: 1},
				{Type: TokenEOF},
			},
		},
		{
			name:  "regex in rule",
			input: `identifier = #"[a-zA-Z][a-zA-Z0-9_]*" ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "identifier", Line: 1, Col: 1},
				{Type: TokenEquals, Value: "=", Line: 1, Col: 12},
				{Type: TokenRegex, Value: "[a-zA-Z][a-zA-Z0-9_]*", Line: 1, Col: 14},
				{Type: TokenSemi, Value: ";", Line: 1, Col: 38},
				{Type: TokenEOF},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			for i, expected := range test.expected {
				token, err := lexer.NextToken()
				if err != nil {
					t.Fatalf("unexpected error at token %d: %v", i, err)
				}
				if token.Type != expected.Type {
					t.Errorf("token %d: expected type %v, got %v", i, expected.Type, token.Type)
				}
				if expected.Value != "" && token.Value != expected.Value {
					t.Errorf("token %d: expected value %q, got %q", i, expected.Value, token.Value)
				}
			}
		})
	}
}

func TestRegexParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple regex rule",
			input:   `identifier = #"[a-zA-Z][a-zA-Z0-9_]*" ;`,
			wantErr: false,
		},
		{
			name:    "hidden regex",
			input:   `ws = <#"\\s+"> ;`,
			wantErr: false,
		},
		{
			name:    "regex in choice",
			input:   `token = #"[0-9]+" | #"[a-zA-Z]+" ;`,
			wantErr: false,
		},
		{
			name:    "regex with repetition",
			input:   `text = #"[^\\n]"+ ;`,
			wantErr: false,
		},
		{
			name:    "unterminated regex",
			input:   `bad = #"[a-z] ;`,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := NewParser(test.input)
			grammar, err := parser.ParseGrammar()
			if test.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if grammar == nil {
					t.Error("expected grammar but got nil")
				}
			}
		})
	}
}

func TestRegexAST(t *testing.T) {
	input := `identifier = #"[a-zA-Z][a-zA-Z0-9_]*" ;`
	parser := NewParser(input)
	grammar, err := parser.ParseGrammar()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(grammar.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(grammar.Rules))
	}

	rule := grammar.Rules[0]
	if rule.Name != "identifier" {
		t.Errorf("expected rule name 'identifier', got %q", rule.Name)
	}

	regex, ok := rule.Expression.(*Regex)
	if !ok {
		t.Fatalf("expected Regex expression, got %T", rule.Expression)
	}

	expectedPattern := "[a-zA-Z][a-zA-Z0-9_]*"
	if regex.Pattern != expectedPattern {
		t.Errorf("expected pattern %q, got %q", expectedPattern, regex.Pattern)
	}

	if regex.Hidden {
		t.Error("expected regex to not be hidden")
	}
}

func TestHiddenRegex(t *testing.T) {
	input := `ws = <#"\\s+"> ;`
	parser := NewParser(input)
	grammar, err := parser.ParseGrammar()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rule := grammar.Rules[0]
	regex, ok := rule.Expression.(*Regex)
	if !ok {
		t.Fatalf("expected Regex expression, got %T", rule.Expression)
	}

	if !regex.Hidden {
		t.Error("expected regex to be hidden")
	}

	if regex.Pattern != `\\s+` {
		t.Errorf("expected pattern %q, got %q", `\\s+`, regex.Pattern)
	}
}