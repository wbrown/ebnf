package ebnf

import (
	"testing"
)

func TestLexerCaseInsensitiveTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "double quoted string with i suffix",
			input: `"hello"i`,
			expected: []Token{
				{Type: TokenStringCI, Value: "hello"},
			},
		},
		{
			name:  "single quoted string with i suffix",
			input: `'world'i`,
			expected: []Token{
				{Type: TokenCharCI, Value: "world"},
			},
		},
		{
			name:  "mixed case-sensitive and case-insensitive",
			input: `"hello"i 'world' "test"i`,
			expected: []Token{
				{Type: TokenStringCI, Value: "hello"},
				{Type: TokenChar, Value: "world"},
				{Type: TokenStringCI, Value: "test"},
			},
		},
		{
			name:  "case-insensitive in rule definition",
			input: `keyword = 'SELECT'i ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "keyword"},
				{Type: TokenEquals, Value: "="},
				{Type: TokenCharCI, Value: "SELECT"},
				{Type: TokenSemi, Value: ";"},
			},
		},
		{
			name:  "regular string without i suffix",
			input: `"identifier"`,
			expected: []Token{
				{Type: TokenString, Value: "identifier"},
			},
		},
		{
			name:  "i as identifier after space",
			input: `"hello" i`,
			expected: []Token{
				{Type: TokenString, Value: "hello"},
				{Type: TokenIdent, Value: "i"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)

			for i, expected := range tt.expected {
				tok, err := lexer.NextToken()
				if err != nil {
					t.Fatalf("Token %d: unexpected error: %v", i, err)
				}

				if tok.Type != expected.Type {
					t.Errorf("Token %d: expected type %v, got %v", i, expected.Type, tok.Type)
				}
				if tok.Value != expected.Value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.Value, tok.Value)
				}
			}

			// Check for EOF
			tok, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("EOF check: unexpected error: %v", err)
			}
			if tok.Type != TokenEOF {
				t.Errorf("Expected EOF, got %v", tok.Type)
			}
		})
	}
}

func TestGrammarParserCaseInsensitive(t *testing.T) {
	tests := []struct {
		name            string
		grammar         string
		terminalValue   string
		caseInsensitive bool
	}{
		{
			name:            "case-insensitive terminal with double quotes",
			grammar:         `keyword = "SELECT"i ;`,
			terminalValue:   "SELECT",
			caseInsensitive: true,
		},
		{
			name:            "case-insensitive terminal with single quotes",
			grammar:         `keyword = 'FROM'i ;`,
			terminalValue:   "FROM",
			caseInsensitive: true,
		},
		{
			name:            "case-sensitive terminal (default)",
			grammar:         `keyword = "SELECT" ;`,
			terminalValue:   "SELECT",
			caseInsensitive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grammar, err := ParseString(tt.grammar)
			if err != nil {
				t.Fatalf("Failed to parse grammar: %v", err)
			}

			if len(grammar.Rules) != 1 {
				t.Fatalf("Expected 1 rule, got %d", len(grammar.Rules))
			}

			rule := grammar.Rules[0]
			terminal, ok := rule.Expression.(*Terminal)
			if !ok {
				t.Fatalf("Expected Terminal expression, got %T", rule.Expression)
			}

			if terminal.Value != tt.terminalValue {
				t.Errorf("Expected terminal value %q, got %q", tt.terminalValue, terminal.Value)
			}

			if terminal.CaseInsensitive != tt.caseInsensitive {
				t.Errorf("Expected CaseInsensitive=%v, got %v", tt.caseInsensitive, terminal.CaseInsensitive)
			}
		})
	}
}
