package ebnf

import (
	"fmt"
	"testing"
)

func TestComplexChoiceScriptRules(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*Grammar) error
	}{
		{
			name: "bracketed choice expression",
			input: `
expr_option = <ws>+ <'|'> <ws>* text_content ;
`,
			validate: func(g *Grammar) error {
				// This should parse as a sequence with hidden terminals
				if len(g.Rules) != 1 {
					return fmt.Errorf("expected 1 rule, got %d", len(g.Rules))
				}
				seq, ok := g.Rules[0].Expression.(*Sequence)
				if !ok {
					return fmt.Errorf("expected Sequence, got %T", g.Rules[0].Expression)
				}
				if len(seq.Elements) != 4 {
					return fmt.Errorf("expected 4 elements, got %d", len(seq.Elements))
				}
				return nil
			},
		},
		{
			name: "character class with ranges",
			input: `
letter = 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' 
       | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z'
       | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' 
       | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' ;
`,
			validate: func(g *Grammar) error {
				if len(g.Rules) != 1 {
					return fmt.Errorf("expected 1 rule, got %d", len(g.Rules))
				}
				choice, ok := g.Rules[0].Expression.(*Choice)
				if !ok {
					return fmt.Errorf("expected Choice, got %T", g.Rules[0].Expression)
				}
				if len(choice.Alternatives) != 52 {
					return fmt.Errorf("expected 52 alternatives, got %d", len(choice.Alternatives))
				}
				return nil
			},
		},
		{
			name: "nested optional with choice",
			input: `
goto_scene_tail = <ws>+ label_ref [ goto_scene_args | null ] ;
`,
			wantErr: false, // We DO handle [ choice ] - it's just a choice!
			validate: func(g *Grammar) error {
				if len(g.Rules) != 1 {
					return fmt.Errorf("expected 1 rule, got %d", len(g.Rules))
				}
				// Should parse as sequence with a choice at the end
				seq, ok := g.Rules[0].Expression.(*Sequence)
				if !ok {
					return fmt.Errorf("expected Sequence, got %T", g.Rules[0].Expression)
				}
				if len(seq.Elements) != 3 {
					return fmt.Errorf("expected 3 elements, got %d", len(seq.Elements))
				}
				// Last element should be a choice
				choice, ok := seq.Elements[2].(*Choice)
				if !ok {
					return fmt.Errorf("expected Choice as last element, got %T", seq.Elements[2])
				}
				if len(choice.Alternatives) != 2 {
					return fmt.Errorf("expected 2 alternatives in choice, got %d", len(choice.Alternatives))
				}
				return nil
			},
		},
		{
			name: "complex expression with predicates",
			input: `
text_chunk = plain_char+ ;
plain_char = letter | digit | <' '> | <'!'> | <'"'> | <'#'> | <'%'> | <'&'> | <"'"> 
           | <'('> | <')'> | <'*'> | <'+'> | <','> | <'-'> | <'.'> | <'/'> | <':'> | <';'> 
           | <'<'> | <'='> | <'>'> | <'?'> | <'\\'> | <']'> | <'^'> | <'_'> | <'` + "`" + `'> 
           | <'{'> | <'|'> | <'}'> | <'~'> ;
`,
			validate: func(g *Grammar) error {
				if len(g.Rules) != 2 {
					return fmt.Errorf("expected 2 rules, got %d", len(g.Rules))
				}
				// Check plain_char has many alternatives
				plainChar := g.GetRule("plain_char")
				if plainChar == nil {
					return fmt.Errorf("plain_char rule not found")
				}
				choice, ok := plainChar.Expression.(*Choice)
				if !ok {
					return fmt.Errorf("expected Choice for plain_char, got %T", plainChar.Expression)
				}
				// Should have many alternatives
				if len(choice.Alternatives) < 20 {
					return fmt.Errorf("expected many alternatives, got %d", len(choice.Alternatives))
				}

				// Debug: print what we actually got
				fmt.Printf("DEBUG: plain_char has %d alternatives\n", len(choice.Alternatives))
				for i, alt := range choice.Alternatives {
					fmt.Printf("  [%d] %T\n", i, alt)
					if i > 10 {
						fmt.Println("  ...")
						break
					}
				}
				return nil
			},
		},
		{
			name: "empty rule (null production)",
			input: `
null = '' ;
paragraph_break = '' ;
`,
			validate: func(g *Grammar) error {
				if len(g.Rules) != 2 {
					return fmt.Errorf("expected 2 rules, got %d", len(g.Rules))
				}
				// Both should have empty string terminals
				for _, rule := range g.Rules {
					term, ok := rule.Expression.(*Terminal)
					if !ok {
						return fmt.Errorf("expected Terminal for %s, got %T", rule.Name, rule.Expression)
					}
					if term.Value != "" {
						return fmt.Errorf("expected empty string for %s, got %q", rule.Name, term.Value)
					}
				}
				return nil
			},
		},
		{
			name: "complex sequence with optionals",
			input: `
achievement = <'achievement'> <ws>+ achievement_name <ws>+ visibility <ws>+ points <ws>+ 
              text_content <newline> current_indent text_content 
              [ achievement_earned | null ] ;
`,
			wantErr: false, // Should parse fine - multi-line is OK and [ ] is just a choice
			validate: func(g *Grammar) error {
				if len(g.Rules) != 1 {
					return fmt.Errorf("expected 1 rule, got %d", len(g.Rules))
				}
				// Should be a long sequence
				seq, ok := g.Rules[0].Expression.(*Sequence)
				if !ok {
					return fmt.Errorf("expected Sequence, got %T", g.Rules[0].Expression)
				}
				// Should have many elements including the final choice
				if len(seq.Elements) < 10 {
					return fmt.Errorf("expected at least 10 elements, got %d", len(seq.Elements))
				}
				return nil
			},
		},
		{
			name: "special operators",
			input: `
lte = <'<='> ;
gte = <'>='> ;
not_equals = <'!='> ;
`,
			validate: func(g *Grammar) error {
				if len(g.Rules) != 3 {
					return fmt.Errorf("expected 3 rules, got %d", len(g.Rules))
				}
				// Check that <= is parsed correctly
				lte := g.GetRule("lte")
				if lte == nil {
					return fmt.Errorf("lte rule not found")
				}
				term, ok := lte.Expression.(*Terminal)
				if !ok {
					return fmt.Errorf("expected Terminal for lte, got %T", lte.Expression)
				}
				if term.Value != "<=" || !term.Hidden {
					return fmt.Errorf("expected hidden '<=', got hidden=%v value=%q", term.Hidden, term.Value)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := ParseString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				if err := tt.validate(g); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestEscapeSequences(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`rule = '\n' ;`, "\n"},
		{`rule = '\t' ;`, "\t"},
		{`rule = '\r' ;`, "\r"},
		{`rule = '\\' ;`, "\\"},
		{`rule = '` + "\\" + `'' ;`, "'"},
		{`rule = '` + "\\" + `"' ;`, "\""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			g, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			term, ok := g.Rules[0].Expression.(*Terminal)
			if !ok {
				t.Fatalf("Expected Terminal, got %T", g.Rules[0].Expression)
			}
			if term.Value != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, term.Value)
			}
		})
	}
}

func TestParserEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty input",
			input:   "",
			wantErr: false, // Should parse as empty grammar
		},
		{
			name:    "only comments",
			input:   "(* just a comment *)",
			wantErr: false,
		},
		{
			name:    "unterminated comment",
			input:   "(* unterminated",
			wantErr: true, // Our lexer DOES catch this now!
		},
		{
			name:    "missing semicolon",
			input:   "rule = 'test'",
			wantErr: true,
		},
		{
			name:    "missing equals",
			input:   "rule 'test' ;",
			wantErr: true,
		},
		{
			name:    "unclosed string",
			input:   `rule = "unclosed ;`,
			wantErr: true,
		},
		{
			name:    "unclosed group",
			input:   "rule = ( 'a' | 'b' ;",
			wantErr: true,
		},
		{
			name:    "double semicolon",
			input:   "rule = 'test' ; ;",
			wantErr: false, // Second semicolon starts empty rule
		},
		{
			name:    "nested groups",
			input:   "rule = ( ( 'a' ) ) ;",
			wantErr: false,
		},
		{
			name:    "empty choice",
			input:   "rule = 'a' | | 'b' ;",
			wantErr: false, // Middle empty is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
