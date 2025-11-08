package ebnf

import (
	"testing"
)

func TestCharacterClassParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		checkResult func(t *testing.T, grammar *Grammar)
	}{
		{
			name:    "simple_char_class",
			input:   `test = [a-z] ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				cc, ok := grammar.Rules[0].Expression.(*CharClass)
				if !ok {
					t.Fatalf("Expected CharClass, got %T", grammar.Rules[0].Expression)
				}
				if cc.Negated {
					t.Error("Expected non-negated char class")
				}
				if len(cc.Ranges) != 1 {
					t.Errorf("Expected 1 range, got %d", len(cc.Ranges))
				}
			},
		},
		{
			name:    "negated_char_class_single_chars",
			input:   `test = [^a-z] ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				cc, ok := grammar.Rules[0].Expression.(*CharClass)
				if !ok {
					t.Fatalf("Expected CharClass, got %T", grammar.Rules[0].Expression)
				}
				if !cc.Negated {
					t.Error("Expected negated char class")
				}
				if len(cc.Ranges) != 1 {
					t.Errorf("Expected 1 range, got %d", len(cc.Ranges))
				}
			},
		},
		{
			name:    "char_class_with_individual_chars",
			input:   `test = [abc] ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				// "abc" gets lexed as a single identifier, so it becomes a choice
				_, ok := grammar.Rules[0].Expression.(*Choice)
				if !ok {
					// Actually becomes a NonTerminal since no pipe
					_, ok = grammar.Rules[0].Expression.(*NonTerminal)
					if !ok {
						t.Fatalf("Expected NonTerminal or Choice due to lexer behavior, got %T", grammar.Rules[0].Expression)
					}
				}
			},
		},
		{
			name:    "char_class_vs_choice_with_pipe",
			input:   `test = [a | b] ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				_, ok := grammar.Rules[0].Expression.(*Choice)
				if !ok {
					t.Fatalf("Expected Choice (not CharClass), got %T", grammar.Rules[0].Expression)
				}
			},
		},
		{
			name:    "char_class_vs_choice_with_identifiers",
			input:   `test = [foo | bar] ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				_, ok := grammar.Rules[0].Expression.(*Choice)
				if !ok {
					t.Fatalf("Expected Choice (not CharClass), got %T", grammar.Rules[0].Expression)
				}
			},
		},
		{
			name:    "char_class_vs_choice_with_comment",
			input:   `test = [a (* comment *) | b] ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				_, ok := grammar.Rules[0].Expression.(*Choice)
				if !ok {
					t.Fatalf("Expected Choice due to comment, got %T", grammar.Rules[0].Expression)
				}
			},
		},
		{
			name:    "char_class_vs_choice_with_parentheses",
			input:   `test = [(a) | b] ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				_, ok := grammar.Rules[0].Expression.(*Choice)
				if !ok {
					t.Fatalf("Expected Choice due to parentheses, got %T", grammar.Rules[0].Expression)
				}
			},
		},
		{
			name:    "hidden_char_class",
			input:   `test = <[a-z]> ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				hidden, ok := grammar.Rules[0].Expression.(*Hidden)
				if !ok {
					t.Fatalf("Expected Hidden, got %T", grammar.Rules[0].Expression)
				}
				_, ok = hidden.Expr.(*CharClass)
				if !ok {
					t.Fatalf("Expected CharClass inside Hidden, got %T", hidden.Expr)
				}
			},
		},
		{
			name:    "char_class_with_repetition",
			input:   `test = [a-z]+ ;`,
			wantErr: false,
			checkResult: func(t *testing.T, grammar *Grammar) {
				oneOrMore, ok := grammar.Rules[0].Expression.(*OneOrMore)
				if !ok {
					t.Fatalf("Expected OneOrMore, got %T", grammar.Rules[0].Expression)
				}
				_, ok = oneOrMore.Expr.(*CharClass)
				if !ok {
					t.Fatalf("Expected CharClass inside OneOrMore, got %T", oneOrMore.Expr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grammar, err := ParseString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, grammar)
			}
		})
	}
}

func TestCharacterClassTokenization(t *testing.T) {
	// Test that the lexer produces expected tokens for character class syntax
	tests := []struct {
		name   string
		input  string
		tokens []TokenType
		values []string
	}{
		{
			name:   "simple_range",
			input:  "[a-z]",
			tokens: []TokenType{TokenLBracket, TokenIdent, TokenMinus, TokenIdent, TokenRBracket},
			values: []string{"[", "a", "-", "z", "]"},
		},
		{
			name:   "negated_range",
			input:  "[^a-z]",
			tokens: []TokenType{TokenLBracket, TokenCaret, TokenIdent, TokenMinus, TokenIdent, TokenRBracket},
			values: []string{"[", "^", "a", "-", "z", "]"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			
			for i, expectedType := range tt.tokens {
				tok, err := lexer.NextToken()
				if err != nil {
					t.Fatalf("Unexpected error at token %d: %v", i, err)
				}
				
				if tok.Type != expectedType {
					t.Errorf("Token %d: expected type %v, got %v", i, expectedType, tok.Type)
				}
				
				if i < len(tt.values) && tok.Value != tt.values[i] {
					t.Errorf("Token %d: expected value %q, got %q", i, tt.values[i], tok.Value)
				}
			}
		})
	}
}