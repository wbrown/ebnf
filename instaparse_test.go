package ebnf

import (
	"testing"
)

func TestInstaparseFeatures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkAST func(t *testing.T, grammar *Grammar)
	}{
		{
			name: "ordered choice with slash",
			input: `
			expr = number / identifier / string ;
			number = #"[0-9]+" ;
			identifier = #"[a-zA-Z]+" ;
			string = #"\"[^\"]*\"" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("expr")
				if rule == nil {
					t.Fatal("rule 'expr' not found")
				}
				choice, ok := rule.Expression.(*OrderedChoice)
				if !ok {
					t.Fatalf("expected OrderedChoice, got %T", rule.Expression)
				}
				if len(choice.Alternatives) != 3 {
					t.Errorf("expected 3 alternatives, got %d", len(choice.Alternatives))
				}
			},
		},
		{
			name: "mixed ordered and unordered choice",
			input: `
			expr = primary / secondary | fallback ;
			primary = "a" ;
			secondary = "b" ;
			fallback = "c" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("expr")
				if rule == nil {
					t.Fatal("rule 'expr' not found")
				}
				// Should be unordered choice at top level
				choice, ok := rule.Expression.(*Choice)
				if !ok {
					t.Fatalf("expected Choice, got %T", rule.Expression)
				}
				if len(choice.Alternatives) != 2 {
					t.Errorf("expected 2 alternatives, got %d", len(choice.Alternatives))
				}
				// First alternative should be ordered choice
				orderedChoice, ok := choice.Alternatives[0].(*OrderedChoice)
				if !ok {
					t.Fatalf("expected first alternative to be OrderedChoice, got %T", choice.Alternatives[0])
				}
				if len(orderedChoice.Alternatives) != 2 {
					t.Errorf("expected 2 ordered alternatives, got %d", len(orderedChoice.Alternatives))
				}
			},
		},
		{
			name: "hidden non-terminal",
			input: `
			statement = keyword <ws> identifier ;
			keyword = "let" | "var" ;
			identifier = #"[a-zA-Z]+" ;
			ws = #"\s+" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("statement")
				if rule == nil {
					t.Fatal("rule 'statement' not found")
				}
				seq, ok := rule.Expression.(*Sequence)
				if !ok {
					t.Fatalf("expected Sequence, got %T", rule.Expression)
				}
				if len(seq.Elements) != 3 {
					t.Errorf("expected 3 elements, got %d", len(seq.Elements))
				}
				// Check middle element is hidden non-terminal
				nt, ok := seq.Elements[1].(*NonTerminal)
				if !ok {
					t.Fatalf("expected NonTerminal at position 1, got %T", seq.Elements[1])
				}
				if nt.Name != "ws" {
					t.Errorf("expected non-terminal name 'ws', got %q", nt.Name)
				}
				if !nt.Hidden {
					t.Error("expected non-terminal to be hidden")
				}
			},
		},
		{
			name: "hidden terminal",
			input: `
			list = <"["> items <"]"> ;
			items = item ("," item)* ;
			item = #"[a-z]+" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("list")
				if rule == nil {
					t.Fatal("rule 'list' not found")
				}
				seq, ok := rule.Expression.(*Sequence)
				if !ok {
					t.Fatalf("expected Sequence, got %T", rule.Expression)
				}
				// Check first element is hidden terminal
				term1, ok := seq.Elements[0].(*Terminal)
				if !ok {
					t.Fatalf("expected Terminal at position 0, got %T", seq.Elements[0])
				}
				if !term1.Hidden {
					t.Error("expected first terminal to be hidden")
				}
				// Check last element is hidden terminal
				term3, ok := seq.Elements[2].(*Terminal)
				if !ok {
					t.Fatalf("expected Terminal at position 2, got %T", seq.Elements[2])
				}
				if !term3.Hidden {
					t.Error("expected last terminal to be hidden")
				}
			},
		},
		{
			name: "hidden regex",
			input: `
			word = letter+ <#"\s*"> ;
			letter = #"[a-zA-Z]" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("word")
				if rule == nil {
					t.Fatal("rule 'word' not found")
				}
				seq, ok := rule.Expression.(*Sequence)
				if !ok {
					t.Fatalf("expected Sequence, got %T", rule.Expression)
				}
				// Check second element is hidden regex
				regex, ok := seq.Elements[1].(*Regex)
				if !ok {
					t.Fatalf("expected Regex at position 1, got %T", seq.Elements[1])
				}
				if !regex.Hidden {
					t.Error("expected regex to be hidden")
				}
			},
		},
		{
			name: "positive lookahead",
			input: `
			identifier = &letter #"[a-zA-Z0-9]+" ;
			letter = #"[a-zA-Z]" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("identifier")
				if rule == nil {
					t.Fatal("rule 'identifier' not found")
				}
				seq, ok := rule.Expression.(*Sequence)
				if !ok {
					t.Fatalf("expected Sequence, got %T", rule.Expression)
				}
				// Check first element is positive lookahead
				pos, ok := seq.Elements[0].(*PositiveLookahead)
				if !ok {
					t.Fatalf("expected PositiveLookahead at position 0, got %T", seq.Elements[0])
				}
				// Check the lookahead expression
				nt, ok := pos.Expr.(*NonTerminal)
				if !ok {
					t.Fatalf("expected NonTerminal in lookahead, got %T", pos.Expr)
				}
				if nt.Name != "letter" {
					t.Errorf("expected lookahead for 'letter', got %q", nt.Name)
				}
			},
		},
		{
			name: "negative lookahead with positive lookahead",
			input: `
			notKeyword = !keyword &letter #"[a-zA-Z]+" ;
			keyword = "if" | "else" | "for" ;
			letter = #"[a-zA-Z]" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("notKeyword")
				if rule == nil {
					t.Fatal("rule 'notKeyword' not found")
				}
				seq, ok := rule.Expression.(*Sequence)
				if !ok {
					t.Fatalf("expected Sequence, got %T", rule.Expression)
				}
				// Check first element is negative lookahead
				neg, ok := seq.Elements[0].(*Predicate)
				if !ok {
					t.Fatalf("expected Predicate at position 0, got %T", seq.Elements[0])
				}
				// Check second element is positive lookahead
				pos, ok := seq.Elements[1].(*PositiveLookahead)
				if !ok {
					t.Fatalf("expected PositiveLookahead at position 1, got %T", seq.Elements[1])
				}
				// Check expressions
				if nt, ok := neg.Expr.(*NonTerminal); !ok || nt.Name != "keyword" {
					t.Error("expected negative lookahead for 'keyword'")
				}
				if nt, ok := pos.Expr.(*NonTerminal); !ok || nt.Name != "letter" {
					t.Error("expected positive lookahead for 'letter'")
				}
			},
		},
		{
			name: "alternative rule separators",
			input: `
			rule1 = "one" ;
			rule2 : "two" ;
			rule3 := "three" ;
			rule4 ::= "four" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				// All rules should parse successfully
				rules := []string{"rule1", "rule2", "rule3", "rule4"}
				values := []string{"one", "two", "three", "four"}
				
				for i, ruleName := range rules {
					rule := g.GetRule(ruleName)
					if rule == nil {
						t.Fatalf("rule %q not found", ruleName)
					}
					term, ok := rule.Expression.(*Terminal)
					if !ok {
						t.Fatalf("expected Terminal for %s, got %T", ruleName, rule.Expression)
					}
					if term.Value != values[i] {
						t.Errorf("expected value %q for %s, got %q", values[i], ruleName, term.Value)
					}
				}
			},
		},
		{
			name: "curly braces for repetition",
			input: `
			list1 = item* ;
			list2 = {item} ;
			item = #"[a-z]+" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				// Both list1 and list2 should have the same structure
				rule1 := g.GetRule("list1")
				rule2 := g.GetRule("list2")
				if rule1 == nil || rule2 == nil {
					t.Fatal("rules not found")
				}
				
				// Both should be Repetition
				rep1, ok1 := rule1.Expression.(*Repetition)
				rep2, ok2 := rule2.Expression.(*Repetition)
				if !ok1 || !ok2 {
					t.Fatalf("expected Repetition, got %T and %T", rule1.Expression, rule2.Expression)
				}
				
				// Both should repeat NonTerminal "item"
				nt1, ok1 := rep1.Expr.(*NonTerminal)
				nt2, ok2 := rep2.Expr.(*NonTerminal)
				if !ok1 || !ok2 {
					t.Fatalf("expected NonTerminal in repetitions")
				}
				if nt1.Name != "item" || nt2.Name != "item" {
					t.Errorf("expected 'item', got %q and %q", nt1.Name, nt2.Name)
				}
			},
		},
		{
			name: "curly braces in complex expression",
			input: `
			rule = "start" {middle} "end" ;
			middle = "m" ;
			`,
			checkAST: func(t *testing.T, g *Grammar) {
				rule := g.GetRule("rule")
				if rule == nil {
					t.Fatal("rule not found")
				}
				
				seq, ok := rule.Expression.(*Sequence)
				if !ok {
					t.Fatalf("expected Sequence, got %T", rule.Expression)
				}
				if len(seq.Elements) != 3 {
					t.Fatalf("expected 3 elements, got %d", len(seq.Elements))
				}
				
				// Middle element should be Repetition
				rep, ok := seq.Elements[1].(*Repetition)
				if !ok {
					t.Fatalf("expected Repetition at position 1, got %T", seq.Elements[1])
				}
				
				// Should repeat NonTerminal "middle"
				nt, ok := rep.Expr.(*NonTerminal)
				if !ok || nt.Name != "middle" {
					t.Error("expected repetition of 'middle'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			grammar, err := p.ParseGrammar()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGrammar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tt.checkAST != nil {
				tt.checkAST(t, grammar)
			}
		})
	}
}

func TestSlashTokenLexing(t *testing.T) {
	input := `rule = a / b | c ;`
	lexer := NewLexer(input)
	
	expectedTokens := []struct {
		typ   TokenType
		value string
	}{
		{TokenIdent, "rule"},
		{TokenEquals, "="},
		{TokenIdent, "a"},
		{TokenSlash, "/"},
		{TokenIdent, "b"},
		{TokenPipe, "|"},
		{TokenIdent, "c"},
		{TokenSemi, ";"},
		{TokenEOF, ""},
	}
	
	for i, expected := range expectedTokens {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if tok.Type != expected.typ {
			t.Errorf("token %d: expected type %v, got %v", i, expected.typ, tok.Type)
		}
		if tok.Value != expected.value {
			t.Errorf("token %d: expected value %q, got %q", i, expected.value, tok.Value)
		}
	}
}