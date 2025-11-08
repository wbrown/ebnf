package ebnf

import (
	"testing"
)

func TestParseSimpleRule(t *testing.T) {
	input := `
(* Simple test grammar *)
number = digit+ ;
digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' ;
`
	
	grammar, err := ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	if len(grammar.Rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(grammar.Rules))
	}
	
	// Check first rule
	if grammar.Rules[0].Name != "number" {
		t.Errorf("Expected first rule name 'number', got '%s'", grammar.Rules[0].Name)
	}
	
	// Check it's a OneOrMore
	oneOrMore, ok := grammar.Rules[0].Expression.(*OneOrMore)
	if !ok {
		t.Errorf("Expected OneOrMore expression, got %T", grammar.Rules[0].Expression)
	}
	
	// Check the inner expression is NonTerminal "digit"
	nonTerm, ok := oneOrMore.Expr.(*NonTerminal)
	if !ok {
		t.Errorf("Expected NonTerminal inside OneOrMore, got %T", oneOrMore.Expr)
	}
	if nonTerm.Name != "digit" {
		t.Errorf("Expected NonTerminal name 'digit', got '%s'", nonTerm.Name)
	}
}

func TestParseChoice(t *testing.T) {
	input := `bool = 'true' | 'false' ;`
	
	grammar, err := ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	if len(grammar.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(grammar.Rules))
	}
	
	// Check it's a Choice
	choice, ok := grammar.Rules[0].Expression.(*Choice)
	if !ok {
		t.Errorf("Expected Choice expression, got %T", grammar.Rules[0].Expression)
	}
	
	if len(choice.Alternatives) != 2 {
		t.Errorf("Expected 2 alternatives, got %d", len(choice.Alternatives))
	}
}

func TestParseHiddenTerminal(t *testing.T) {
	input := `ws = <' '> | <'\t'> ;`
	
	grammar, err := ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	// Check it's a Choice with hidden terminals
	choice, ok := grammar.Rules[0].Expression.(*Choice)
	if !ok {
		t.Fatalf("Expected Choice expression, got %T", grammar.Rules[0].Expression)
	}
	
	// Check first alternative is hidden terminal
	term1, ok := choice.Alternatives[0].(*Terminal)
	if !ok {
		t.Errorf("Expected Terminal, got %T", choice.Alternatives[0])
	}
	if !term1.Hidden {
		t.Errorf("Expected hidden terminal")
	}
}

func TestParseSequence(t *testing.T) {
	input := `command = '*' name args ;`
	
	grammar, err := ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	// Check it's a Sequence
	seq, ok := grammar.Rules[0].Expression.(*Sequence)
	if !ok {
		t.Errorf("Expected Sequence expression, got %T", grammar.Rules[0].Expression)
	}
	
	if len(seq.Elements) != 3 {
		t.Errorf("Expected 3 elements in sequence, got %d", len(seq.Elements))
	}
}

func TestParseOptional(t *testing.T) {
	input := `line = content? newline ;`
	
	grammar, err := ParseString(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	// Check it's a Sequence with optional first element
	seq, ok := grammar.Rules[0].Expression.(*Sequence)
	if !ok {
		t.Fatalf("Expected Sequence expression, got %T", grammar.Rules[0].Expression)
	}
	
	// First element should be Optional
	opt, ok := seq.Elements[0].(*Optional)
	if !ok {
		t.Errorf("Expected Optional as first element, got %T", seq.Elements[0])
	}
	
	// Inside should be NonTerminal "content"
	nonTerm, ok := opt.Expr.(*NonTerminal)
	if !ok {
		t.Errorf("Expected NonTerminal inside Optional, got %T", opt.Expr)
	}
	if nonTerm.Name != "content" {
		t.Errorf("Expected 'content', got '%s'", nonTerm.Name)
	}
}