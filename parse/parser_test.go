package parse

import (
	"fmt"
	"sync"
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

// TestConcurrentParse verifies that a single *Parser can be used concurrently
// by 100 goroutines parsing different inputs with the same grammar.
func TestConcurrentParse(t *testing.T) {
	grammarText := `
digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' ;
number = digit+ ;
`
	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	p := New(grammar)

	const goroutines = 100
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			input := fmt.Sprintf("%d", n)
			tree, err := p.Parse(input, "number")
			if err != nil {
				t.Errorf("goroutine %d: Parse(%q) failed: %v", n, input, err)
				return
			}
			if tree.Root == nil {
				t.Errorf("goroutine %d: Parse(%q) returned nil root", n, input)
				return
			}
			// Verify the parsed span matches the input
			text := input[tree.Root.Start:tree.Root.End]
			if text != input {
				t.Errorf("goroutine %d: got %q, want %q", n, text, input)
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentParseDifferentRules verifies concurrent parses using different
// start rules on the same grammar.
func TestConcurrentParseDifferentRules(t *testing.T) {
	grammarText := `
digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' ;
letter = 'a' | 'b' | 'c' | 'd' | 'e' | 'f' ;
number = digit+ ;
word = letter+ ;
`
	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	p := New(grammar)

	type testCase struct {
		input string
		rule  string
	}
	cases := []testCase{
		{"123", "number"},
		{"456", "number"},
		{"abc", "word"},
		{"def", "word"},
		{"7", "digit"},
		{"a", "letter"},
	}

	const repeats = 20
	var wg sync.WaitGroup

	for _, tc := range cases {
		for i := 0; i < repeats; i++ {
			wg.Add(1)
			go func(tc testCase) {
				defer wg.Done()
				tree, err := p.Parse(tc.input, tc.rule)
				if err != nil {
					t.Errorf("Parse(%q, %q) failed: %v", tc.input, tc.rule, err)
					return
				}
				if tree.Root == nil {
					t.Errorf("Parse(%q, %q) returned nil root", tc.input, tc.rule)
				}
			}(tc)
		}
	}

	wg.Wait()
}

// TestConcurrentParseWithRegex verifies concurrent parses that hit the
// regex cache, exercising getCachedRegex under contention.
func TestConcurrentParseWithRegex(t *testing.T) {
	grammarText := `
identifier = #"[a-zA-Z_][a-zA-Z0-9_]*" ;
number = #"[0-9]+" ;
token = identifier | number ;
`
	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	p := New(grammar)

	inputs := []string{"foo", "bar_baz", "_x", "42", "100", "hello123"}
	const repeats = 30
	var wg sync.WaitGroup

	for _, input := range inputs {
		for i := 0; i < repeats; i++ {
			wg.Add(1)
			go func(input string) {
				defer wg.Done()
				tree, err := p.Parse(input, "token")
				if err != nil {
					t.Errorf("Parse(%q) failed: %v", input, err)
					return
				}
				if tree.Root == nil {
					t.Errorf("Parse(%q) returned nil root", input)
				}
			}(input)
		}
	}

	wg.Wait()
}

// TestConcurrentParseErrors verifies concurrent parses where some inputs
// are invalid, exercising recordFurthestError and error reporting concurrently.
func TestConcurrentParseErrors(t *testing.T) {
	grammarText := `
digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' ;
number = digit+ ;
`
	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	p := New(grammar)

	type testCase struct {
		input   string
		wantErr bool
	}
	cases := []testCase{
		{"123", false},
		{"abc", true},
		{"456", false},
		{"xyz", true},
		{"7", false},
		{"", true},
		{"99", false},
		{"!!", true},
	}

	const repeats = 20
	var wg sync.WaitGroup

	for _, tc := range cases {
		for i := 0; i < repeats; i++ {
			wg.Add(1)
			go func(tc testCase) {
				defer wg.Done()
				_, err := p.Parse(tc.input, "number")
				gotErr := err != nil
				if gotErr != tc.wantErr {
					t.Errorf("Parse(%q): gotErr=%v, wantErr=%v, err=%v",
						tc.input, gotErr, tc.wantErr, err)
				}
			}(tc)
		}
	}

	wg.Wait()
}

// TestConcurrentParseDebug verifies concurrent parses with Debug=true,
// ensuring no races on the debug log paths.
func TestConcurrentParseDebug(t *testing.T) {
	grammarText := `
digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' ;
number = digit+ ;
`
	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	p := New(grammar)
	p.Debug = true

	const goroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			input := fmt.Sprintf("%d", n+10)
			_, err := p.Parse(input, "number")
			if err != nil {
				t.Errorf("goroutine %d: Parse(%q) failed: %v", n, input, err)
			}
		}(i)
	}

	wg.Wait()

	// GetDebugLog should return a non-empty string (from whichever parse finished last)
	log := p.GetDebugLog()
	if log == "" {
		t.Error("GetDebugLog() returned empty string after concurrent debug parses")
	}
}
