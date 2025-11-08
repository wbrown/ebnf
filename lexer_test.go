package ebnf

import (
	"testing"
)

func TestLexerEscapeSequences(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: `'\\'`,
			expected: []Token{
				{Type: TokenChar, Value: "\\"},
			},
		},
		{
			input: `'\''`,
			expected: []Token{
				{Type: TokenChar, Value: "'"},
			},
		},
		{
			input: `"test\nline"`,
			expected: []Token{
				{Type: TokenString, Value: "test\nline"},
			},
		},
		{
			input: `<'\\'> | <']'>`,
			expected: []Token{
				{Type: TokenLAngle, Value: "<"},
				{Type: TokenChar, Value: "\\"},
				{Type: TokenRAngle, Value: ">"},
				{Type: TokenPipe, Value: "|"},
				{Type: TokenLAngle, Value: "<"},
				{Type: TokenChar, Value: "]"},
				{Type: TokenRAngle, Value: ">"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
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

func TestLexerAmpersand(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: `&expr`,
			expected: []Token{
				{Type: TokenAmpersand, Value: "&"},
				{Type: TokenIdent, Value: "expr"},
			},
		},
		{
			input: `rule = &letter #"[a-z]+" ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "rule"},
				{Type: TokenEquals, Value: "="},
				{Type: TokenAmpersand, Value: "&"},
				{Type: TokenIdent, Value: "letter"},
				{Type: TokenRegex, Value: "[a-z]+"},
				{Type: TokenSemi, Value: ";"},
			},
		},
		{
			input: `!neg &pos`,
			expected: []Token{
				{Type: TokenExclam, Value: "!"},
				{Type: TokenIdent, Value: "neg"},
				{Type: TokenAmpersand, Value: "&"},
				{Type: TokenIdent, Value: "pos"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
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

func TestLexerAlternativeRuleSeparators(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: `rule : "value" ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "rule"},
				{Type: TokenEquals, Value: ":"},
				{Type: TokenString, Value: "value"},
				{Type: TokenSemi, Value: ";"},
			},
		},
		{
			input: `rule := "value" ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "rule"},
				{Type: TokenEquals, Value: ":="},
				{Type: TokenString, Value: "value"},
				{Type: TokenSemi, Value: ";"},
			},
		},
		{
			input: `rule ::= "value" ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "rule"},
				{Type: TokenEquals, Value: "::="},
				{Type: TokenString, Value: "value"},
				{Type: TokenSemi, Value: ";"},
			},
		},
		{
			input: `a = "a" ; b : "b" ; c := "c" ; d ::= "d" ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "a"},
				{Type: TokenEquals, Value: "="},
				{Type: TokenString, Value: "a"},
				{Type: TokenSemi, Value: ";"},
				{Type: TokenIdent, Value: "b"},
				{Type: TokenEquals, Value: ":"},
				{Type: TokenString, Value: "b"},
				{Type: TokenSemi, Value: ";"},
				{Type: TokenIdent, Value: "c"},
				{Type: TokenEquals, Value: ":="},
				{Type: TokenString, Value: "c"},
				{Type: TokenSemi, Value: ";"},
				{Type: TokenIdent, Value: "d"},
				{Type: TokenEquals, Value: "::="},
				{Type: TokenString, Value: "d"},
				{Type: TokenSemi, Value: ";"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
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

func TestLexerCurlyBraces(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: `{item}`,
			expected: []Token{
				{Type: TokenLBrace, Value: "{"},
				{Type: TokenIdent, Value: "item"},
				{Type: TokenRBrace, Value: "}"},
			},
		},
		{
			input: `list = {#"[a-z]+"} ;`,
			expected: []Token{
				{Type: TokenIdent, Value: "list"},
				{Type: TokenEquals, Value: "="},
				{Type: TokenLBrace, Value: "{"},
				{Type: TokenRegex, Value: "[a-z]+"},
				{Type: TokenRBrace, Value: "}"},
				{Type: TokenSemi, Value: ";"},
			},
		},
		{
			input: `a* {b} c+`,
			expected: []Token{
				{Type: TokenIdent, Value: "a"},
				{Type: TokenStar, Value: "*"},
				{Type: TokenLBrace, Value: "{"},
				{Type: TokenIdent, Value: "b"},
				{Type: TokenRBrace, Value: "}"},
				{Type: TokenIdent, Value: "c"},
				{Type: TokenPlus, Value: "+"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
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
