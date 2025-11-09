package ebnf_test

import (
	"testing"

	"github.com/wbrown/ebnf"
)

func TestArithmeticGrammar(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("examples/arithmetic.ebnf")
	if err != nil {
		t.Fatalf("Failed to load arithmetic grammar: %v", err)
	}

	if len(grammar.Rules) == 0 {
		t.Fatal("Expected rules in arithmetic grammar")
	}

	// Check key rules exist (S-expression-like structure)
	keyRules := []string{"expr", "add", "sub", "mul", "div", "neg", "number"}
	for _, name := range keyRules {
		if grammar.GetRule(name) == nil {
			t.Errorf("Missing expected rule: %s", name)
		}
	}

	t.Logf("Loaded arithmetic grammar with %d rules", len(grammar.Rules))
}

func TestJSONGrammar(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("examples/json.ebnf")
	if err != nil {
		t.Fatalf("Failed to load JSON grammar: %v", err)
	}

	// Check key rules
	keyRules := []string{"json", "value", "object", "array", "string", "number"}
	for _, name := range keyRules {
		rule := grammar.GetRule(name)
		if rule == nil {
			t.Errorf("Missing expected rule: %s", name)
		}
	}

	t.Logf("Loaded JSON grammar with %d rules", len(grammar.Rules))
}

func TestRegexDemo(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("examples/regex_demo.ebnf")
	if err != nil {
		t.Fatalf("Failed to load regex demo: %v", err)
	}

	// Check for identifier rule with regex
	identifier := grammar.GetRule("identifier")
	if identifier == nil {
		t.Fatal("Missing identifier rule")
	}

	// Verify it's a regex pattern
	if _, ok := identifier.Expression.(*ebnf.Regex); !ok {
		t.Errorf("Expected identifier to be a Regex, got %T", identifier.Expression)
	}

	t.Logf("Loaded regex demo with %d rules", len(grammar.Rules))
}

func TestInstaparseDemo(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("examples/instaparse_demo.ebnf")
	if err != nil {
		t.Fatalf("Failed to load instaparse demo: %v", err)
	}

	// Check for expr rule with ordered choice
	expr := grammar.GetRule("expr")
	if expr == nil {
		t.Fatal("Missing expr rule")
	}

	// Verify it uses ordered choice
	if _, ok := expr.Expression.(*ebnf.OrderedChoice); !ok {
		t.Errorf("Expected expr to use OrderedChoice, got %T", expr.Expression)
	}

	t.Logf("Loaded instaparse demo with %d rules", len(grammar.Rules))
}

func ExampleLoadGrammar() {
	// Load a grammar from a file
	grammar, err := ebnf.LoadGrammar("examples/arithmetic.ebnf")
	if err != nil {
		panic(err)
	}

	// Access a specific rule
	exprRule := grammar.GetRule("expr")
	if exprRule != nil {
		// Process the rule...
		_ = exprRule
	}
}

func ExampleParseString() {
	// Define a simple grammar inline
	grammarText := `
		greeting = "Hello" <ws> name "!" ;
		name = #"[A-Z][a-z]+" ;
		ws = #"\s+" ;
	`

	grammar, err := ebnf.ParseString(grammarText)
	if err != nil {
		panic(err)
	}

	// Use the grammar...
	_ = grammar
}
