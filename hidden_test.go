package ebnf

import (
	"testing"
)

func TestHiddenExpressions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "hidden terminal",
			input:   `test = <'hello'> ;`,
			wantErr: false,
		},
		{
			name:    "hidden non-terminal",
			input:   `test = <ws> ;`,
			wantErr: false,
		},
		{
			name:    "hidden character class",
			input:   `test = <[a-z]> ;`,
			wantErr: false,
		},
		{
			name:    "hidden negated character class",
			input:   `test = <[^abc]> ;`,
			wantErr: false,
		},
		{
			name:    "hidden complex expression",
			input:   `test = <('a' | 'b' | [0-9])> ;`,
			wantErr: false,
		},
		{
			name:    "hidden repetition",
			input:   `test = <[a-z]+> ;`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			grammar, err := parser.ParseGrammar()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && grammar == nil {
				t.Error("Expected grammar but got nil")
				return
			}

			// Check that the expression is properly wrapped in Hidden
			if !tt.wantErr {
				rule := grammar.Rules[0]
				switch expr := rule.Expression.(type) {
				case *Terminal:
					if !expr.Hidden {
						t.Error("Expected terminal to be hidden")
					}
				case *NonTerminal:
					if !expr.Hidden {
						t.Error("Expected non-terminal to be hidden")
					}
				case *Regex:
					if !expr.Hidden {
						t.Error("Expected regex to be hidden")
					}
				case *Hidden:
					// Good, it's wrapped in Hidden
					t.Logf("Expression wrapped in Hidden: %T", expr.Expr)
				default:
					t.Errorf("Expected Hidden or hidden expression, got %T", expr)
				}
			}
		})
	}
}
