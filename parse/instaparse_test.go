package parse

import (
	"testing"

	"github.com/wbrown/ebnf"
)

func TestInstaparseParserFeatures(t *testing.T) {
	tests := []struct {
		name        string
		grammar     string
		input       string
		startRule   string
		wantErr     bool
		checkResult func(t *testing.T, tree *ParseTree)
	}{
		{
			name: "ordered choice - first match wins",
			grammar: `
			expr = number / identifier ;
			number = #"[0-9]+" ;
			identifier = #"[a-zA-Z0-9]+" ;
			`,
			input:     "123",
			startRule: "expr",
			checkResult: func(t *testing.T, tree *ParseTree) {
				if tree.Root.Rule != "expr" {
					t.Errorf("expected rule 'expr', got %q", tree.Root.Rule)
				}
				if len(tree.Root.Children) != 1 {
					t.Fatalf("expected 1 child, got %d", len(tree.Root.Children))
				}
				// Should match as number, not identifier
				if tree.Root.Children[0].Rule != "number" {
					t.Errorf("expected 'number' rule, got %q", tree.Root.Children[0].Rule)
				}
			},
		},
		{
			name: "ordered choice - order matters",
			grammar: `
			expr = identifier / number ;
			number = #"[0-9]+" ;
			identifier = #"[a-zA-Z0-9]+" ;
			`,
			input:     "123",
			startRule: "expr",
			checkResult: func(t *testing.T, tree *ParseTree) {
				// Should match as identifier since it comes first and can match digits
				if tree.Root.Children[0].Rule != "identifier" {
					t.Errorf("expected 'identifier' rule, got %q", tree.Root.Children[0].Rule)
				}
			},
		},
		{
			name: "hidden non-terminal",
			grammar: `
			statement = keyword <ws> identifier ;
			keyword = "let" / "var" ;
			identifier = #"[a-zA-Z]+" ;
			ws = #"\s+" ;
			`,
			input:     "let   foo",
			startRule: "statement",
			checkResult: func(t *testing.T, tree *ParseTree) {
				if tree.Root.Rule != "statement" {
					t.Errorf("expected rule 'statement', got %q", tree.Root.Rule)
				}
				// Should have 2 children (ws is hidden)
				if len(tree.Root.Children) != 2 {
					t.Fatalf("expected 2 children, got %d", len(tree.Root.Children))
				}
				// First child should be keyword
				if tree.Root.Children[0].Rule != "keyword" {
					t.Errorf("expected first child to be 'keyword', got %q", tree.Root.Children[0].Rule)
				}
				// Second child should be identifier
				if tree.Root.Children[1].Rule != "identifier" {
					t.Errorf("expected second child to be 'identifier', got %q", tree.Root.Children[1].Rule)
				}
			},
		},
		{
			name: "hidden terminals in list",
			grammar: `
			list = <"["> items <"]"> ;
			items = item (<","> item)* ;
			item = #"[a-z]+" ;
			`,
			input:     "[apple,banana,cherry]",
			startRule: "list",
			checkResult: func(t *testing.T, tree *ParseTree) {
				if tree.Root.Rule != "list" {
					t.Errorf("expected rule 'list', got %q", tree.Root.Rule)
				}
				// Should have 1 child (items), brackets are hidden
				if len(tree.Root.Children) != 1 {
					t.Fatalf("expected 1 child, got %d", len(tree.Root.Children))
				}
				items := tree.Root.Children[0]
				if items.Rule != "items" {
					t.Errorf("expected 'items' rule, got %q", items.Rule)
				}
				// items should have 3 children (commas are hidden)
				if len(items.Children) != 3 {
					t.Errorf("expected 3 items, got %d", len(items.Children))
				}
			},
		},
		{
			name: "mixed ordered and unordered choice",
			grammar: `
			value = literal / expr | error ;
			literal = number / string ;
			number = #"[0-9]+" ;
			string = #"\"[^\"]*\"" ;
			expr = "(" #"[^)]+" ")" ;
			error = "ERROR" ;
			`,
			input:     "42",
			startRule: "value",
			checkResult: func(t *testing.T, tree *ParseTree) {
				// Should match through literal -> number path
				if tree.Root.Rule != "value" {
					t.Errorf("expected rule 'value', got %q", tree.Root.Rule)
				}
			},
		},
		{
			name: "positive lookahead - must start with letter",
			grammar: `
			identifier = &letter #"[a-zA-Z0-9]+" ;
			letter = #"[a-zA-Z]" ;
			`,
			input:     "abc123",
			startRule: "identifier",
			checkResult: func(t *testing.T, tree *ParseTree) {
				if tree.Root.Rule != "identifier" {
					t.Errorf("expected rule 'identifier', got %q", tree.Root.Rule)
				}
				// Should have one child (the regex match), lookahead produces no nodes
				if len(tree.Root.Children) != 1 {
					t.Fatalf("expected 1 child, got %d", len(tree.Root.Children))
				}
				if tree.Root.Children[0].Value != "abc123" {
					t.Errorf("expected value 'abc123', got %q", tree.Root.Children[0].Value)
				}
			},
		},
		{
			name: "positive lookahead failure - starts with digit",
			grammar: `
			identifier = &letter #"[a-zA-Z0-9]+" ;
			letter = #"[a-zA-Z]" ;
			`,
			input:     "123abc",
			startRule: "identifier",
			wantErr:   true,
		},
		{
			name: "negative and positive lookahead combination",
			grammar: `
			notKeyword = !keyword &letter #"[a-zA-Z]+" ;
			keyword = "if" | "else" | "for" ;
			letter = #"[a-zA-Z]" ;
			`,
			input:     "identifier",
			startRule: "notKeyword",
			checkResult: func(t *testing.T, tree *ParseTree) {
				if tree.Root.Rule != "notKeyword" {
					t.Errorf("expected rule 'notKeyword', got %q", tree.Root.Rule)
				}
				// Should have one child (the regex match), lookaheads produce no nodes
				if len(tree.Root.Children) != 1 {
					t.Fatalf("expected 1 child, got %d", len(tree.Root.Children))
				}
				if tree.Root.Children[0].Value != "identifier" {
					t.Errorf("expected value 'identifier', got %q", tree.Root.Children[0].Value)
				}
			},
		},
		{
			name: "lookahead failure - matches keyword",
			grammar: `
			notKeyword = !keyword &letter #"[a-zA-Z]+" ;
			keyword = "if" | "else" | "for" ;
			letter = #"[a-zA-Z]" ;
			`,
			input:     "if",
			startRule: "notKeyword",
			wantErr:   true,
		},
		{
			name: "curly braces repetition",
			grammar: `
			list = {item} ;
			item = #"[a-z]+" <#"\s*"> ;
			`,
			input:     "abc def ghi ",
			startRule: "list",
			checkResult: func(t *testing.T, tree *ParseTree) {
				if tree.Root.Rule != "list" {
					t.Errorf("expected rule 'list', got %q", tree.Root.Rule)
				}
				// Should have 3 item children
				if len(tree.Root.Children) != 3 {
					t.Fatalf("expected 3 children, got %d", len(tree.Root.Children))
				}
				for i, child := range tree.Root.Children {
					if child.Rule != "item" {
						t.Errorf("child %d: expected rule 'item', got %q", i, child.Rule)
					}
				}
			},
		},
		{
			name: "empty curly braces repetition",
			grammar: `
			list = {item} ;
			item = #"[a-z]+" ;
			`,
			input:     "",
			startRule: "list",
			checkResult: func(t *testing.T, tree *ParseTree) {
				if tree.Root.Rule != "list" {
					t.Errorf("expected rule 'list', got %q", tree.Root.Rule)
				}
				// Should have 0 children (zero or more)
				if len(tree.Root.Children) != 0 {
					t.Fatalf("expected 0 children, got %d", len(tree.Root.Children))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the grammar
			grammarParser := ebnf.NewParser(tt.grammar)
			grammar, err := grammarParser.ParseGrammar()
			if err != nil {
				t.Fatalf("failed to parse grammar: %v", err)
			}

			// Create parser and parse input
			p := New(grammar)
			tree, err := p.Parse(tt.input, tt.startRule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.checkResult != nil {
				tt.checkResult(t, tree)
			}
		})
	}
}

func TestOrderedVsUnorderedChoice(t *testing.T) {
	// Test that demonstrates the difference between / and |
	grammar := `
	ordered = "a" "b" / "a" ;
	unordered = "a" "b" | "a" ;
	`

	grammarParser := ebnf.NewParser(grammar)
	g, err := grammarParser.ParseGrammar()
	if err != nil {
		t.Fatalf("failed to parse grammar: %v", err)
	}

	p := New(g)

	// For ordered choice, "a" alone should succeed with our backtracking parser
	// The parser tries "a" "b" first, fails on the missing "b", then backtracks and tries "a"
	_, err = p.Parse("a", "ordered")
	if err != nil {
		t.Errorf("ordered choice should succeed on 'a' with backtracking: %v", err)
	}

	// For unordered choice, implementation might vary
	// Our current implementation still tries alternatives in order,
	// but this is where ambiguity detection could happen
	tree, err := p.Parse("a", "unordered")
	if err != nil {
		t.Errorf("unordered choice should succeed on 'a': %v", err)
	}
	if tree == nil {
		t.Error("expected non-nil tree")
	}
}
