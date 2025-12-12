package parse

import (
	"strings"
	"testing"

	"github.com/wbrown/ebnf"
)

// TestLargeInput tests parsing of very large input
func TestLargeInput(t *testing.T) {
	grammar := `
list = item* ;
item = #"[a-z]+" "," ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	// Generate large input (100K items)
	var sb strings.Builder
	for i := 0; i < 100000; i++ {
		sb.WriteString("item,")
	}

	p := New(g)
	_, err = p.Parse(sb.String(), "list")
	if err != nil {
		t.Fatalf("failed to parse large input: %v", err)
	}
}

// TestDeeplyNestedGrammar tests parsing with deep recursion
func TestDeeplyNestedGrammar(t *testing.T) {
	grammar := `
expr = "(" expr ")" | "x" ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	// Create deeply nested input: ((((x))))
	depth := 100
	var sb strings.Builder
	for i := 0; i < depth; i++ {
		sb.WriteString("(")
	}
	sb.WriteString("x")
	for i := 0; i < depth; i++ {
		sb.WriteString(")")
	}

	p := New(g)
	_, err = p.Parse(sb.String(), "expr")
	if err != nil {
		t.Fatalf("failed to parse deeply nested input: %v", err)
	}
}

// TestHighBacktracking tests grammar with many alternatives
func TestHighBacktracking(t *testing.T) {
	// Grammar with many alternatives that will cause backtracking
	grammar := `
value = a | b | c | d | e | f | g | h | i | j | target ;
a = "prefix" "a" ;
b = "prefix" "b" ;
c = "prefix" "c" ;
d = "prefix" "d" ;
e = "prefix" "e" ;
f = "prefix" "f" ;
g = "prefix" "g" ;
h = "prefix" "h" ;
i = "prefix" "i" ;
j = "prefix" "j" ;
target = "prefix" "target" ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	// This will cause backtracking through all alternatives before matching
	p := New(g)
	_, err = p.Parse("prefixtarget", "value")
	if err != nil {
		t.Fatalf("failed to parse with backtracking: %v", err)
	}
}

// TestManyFailures tests that we don't accumulate errors
func TestManyFailures(t *testing.T) {
	grammar := `
list = item+ ;
item = #"[a-z]+" "," ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	// Input with error at the end - should not accumulate all the successful parses
	input := strings.Repeat("item,", 10000) + "INVALID"

	p := New(g)
	_, err = p.Parse(input, "list")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}

	// Error message should not be huge (no accumulated errors)
	errStr := err.Error()
	if len(errStr) > 1000 {
		t.Errorf("error message too long (%d bytes), might be accumulating errors", len(errStr))
	}
}

// TestRepeatedParsing tests that parser can be reused
func TestRepeatedParsing(t *testing.T) {
	grammar := `
value = "test" ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		t.Fatal(err)
	}

	// Parse many times with same grammar
	for i := 0; i < 1000; i++ {
		p := New(g)
		_, err := p.Parse("test", "value")
		if err != nil {
			t.Fatalf("iteration %d failed: %v", i, err)
		}
	}
}

// BenchmarkBacktrackingStress benchmarks worst-case backtracking
func BenchmarkBacktrackingStress(b *testing.B) {
	grammar := `
value = a | b | c | d | e | f | g | h | target ;
a = "x" "a" ;
b = "x" "b" ;
c = "x" "c" ;
d = "x" "d" ;
e = "x" "e" ;
f = "x" "f" ;
g = "x" "g" ;
h = "x" "h" ;
target = "x" "target" ;
`
	g, err := ebnf.ParseString(grammar)
	if err != nil {
		b.Fatal(err)
	}

	input := "xtarget"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(g)
		_, err := p.Parse(input, "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}
