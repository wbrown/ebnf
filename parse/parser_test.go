package parse

import (
	"testing"

	"github.com/wbrown/ebnf"
)

func TestSimpleGrammarParsing(t *testing.T) {
	// Define a simple grammar
	grammarText := `
digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' ;
number = digit+ ;
`

	// Parse the grammar
	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	// Create parser
	p := New(grammar)

	// Test cases
	tests := []struct {
		input   string
		rule    string
		wantErr bool
	}{
		{"5", "digit", false},
		{"123", "number", false},
		{"a", "digit", true},
		{"", "digit", true},
		{"12a", "number", true}, // Should fail - extra character
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.rule, func(t *testing.T) {
			tree, err := p.Parse(tt.input, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q, %q) error = %v, wantErr %v", tt.input, tt.rule, err, tt.wantErr)
			}
			if err == nil && tree.Root == nil {
				t.Error("Parse returned nil root")
			}
		})
	}
}

func TestParseTreeStructure(t *testing.T) {
	grammarText := `
ws = ' ' | '\t' | '\n' ;
space = <ws>+ ;
word = ('a' | 'b' | 'c')+ ;
phrase = word space word ;
`

	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	p := New(grammar)
	tree, err := p.Parse("abc bca", "phrase")
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	// Check tree structure
	if tree.Root.Rule != "phrase" {
		t.Errorf("Expected root rule 'phrase', got %q", tree.Root.Rule)
	}

	// Should have 3 children (word, space, word)
	if len(tree.Root.Children) != 3 {
		t.Errorf("Expected 3 children, got %d", len(tree.Root.Children))
	}

	// Check children
	if tree.Root.Children[0].Rule != "word" {
		t.Errorf("First child: expected rule 'word', got %q", tree.Root.Children[0].Rule)
	}
	if tree.Root.Children[1].Rule != "space" {
		t.Errorf("Second child: expected rule 'space', got %q", tree.Root.Children[1].Rule)
	}
	if tree.Root.Children[2].Rule != "word" {
		t.Errorf("Third child: expected rule 'word', got %q", tree.Root.Children[2].Rule)
	}
}
