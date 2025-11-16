package parse

import (
	"testing"

	"github.com/wbrown/ebnf"
)

func TestArithmeticGrammar(t *testing.T) {
	// Load the arithmetic grammar
	grammar, err := ebnf.LoadGrammar("../examples/arithmetic.ebnf")
	if err != nil {
		t.Fatalf("Failed to load arithmetic grammar: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple number", "42", false},
		{"decimal number", "3.14", false},
		{"addition", "1 + 2", false},
		{"subtraction", "10 - 5", false},
		{"multiplication", "3 * 4", false},
		{"division", "8 / 2", false},
		{"precedence", "2 + 3 * 4", false},
		{"parentheses", "(2 + 3) * 4", false},
		{"nested parentheses", "((1 + 2) * (3 + 4))", false},
		{"complex expression", "1.5 + 2.3 * (4 - 1) / 2", false},
		{"whitespace variations", "1+2", false},
		{"extra whitespace", "  1  +  2  ", false}, // Grammar now handles leading/trailing whitespace
		{"invalid operator", "1 % 2", true},
		{"missing operand", "1 +", true},
		{"unmatched paren", "(1 + 2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(grammar)
			tree, err := p.Parse(tt.input, "expr")
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && tree.Root == nil {
				t.Error("Parse returned nil root")
			}
		})
	}
}

func TestJSONGrammar(t *testing.T) {
	// Load the JSON grammar
	grammar, err := ebnf.LoadGrammar("../examples/json.ebnf")
	if err != nil {
		t.Fatalf("Failed to load JSON grammar: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"null", "null", false},
		{"true", "true", false},
		{"false", "false", false},
		{"integer", "42", false},
		{"negative integer", "-42", false},
		{"decimal", "3.14", false},
		{"exponent", "1e10", false},
		{"negative exponent", "2.5e-3", false},
		{"simple string", `"hello"`, false},
		{"empty string", `""`, false},
		{"string with escapes", `"hello\nworld"`, false},
		{"string with unicode", `"hello\u0041world"`, false},
		{"empty array", "[]", false},
		{"simple array", "[1, 2, 3]", false},
		{"mixed array", `[1, "two", true, null]`, false},
		{"nested array", "[[1, 2], [3, 4]]", false},
		{"empty object", "{}", false},
		{"simple object", `{"name": "John"}`, false},
		{"object with multiple keys", `{"name": "John", "age": 30}`, false},
		{"nested object", `{"person": {"name": "John", "age": 30}}`, false},
		{"complex json", `{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`, false},
		{"whitespace", `  {  "key"  :  "value"  }  `, false},
		{"invalid - missing quote", `{name: "John"}`, true},
		{"invalid - trailing comma", `[1, 2,]`, true},
		{"invalid - missing colon", `{"name" "John"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(grammar)
			tree, err := p.Parse(tt.input, "json")
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && tree.Root == nil {
				t.Error("Parse returned nil root")
			}
		})
	}
}

func TestInstaparseDemoGrammar(t *testing.T) {
	// Load the instaparse demo grammar
	grammar, err := ebnf.LoadGrammar("../examples/instaparse_demo.ebnf")
	if err != nil {
		t.Fatalf("Failed to load instaparse demo grammar: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		rule    string
		wantErr bool
	}{
		{"number literal", "42", "expr", false},
		{"string literal", `"hello"`, "expr", false},
		{"boolean true", "true", "expr", false},
		{"boolean false", "false", "expr", false},
		{"variable", "myVar", "expr", false},
		{"parenthesized expr", "(42)", "expr", false},
		{"assignment statement", "x = 42", "statement", false},
		{"program with statements", "x = 10; y = 20;", "program", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(grammar)
			tree, err := p.Parse(tt.input, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && tree.Root == nil {
				t.Error("Parse returned nil root")
			}
		})
	}
}

func TestRegexDemoGrammar(t *testing.T) {
	// Load the regex demo grammar
	grammar, err := ebnf.LoadGrammar("../examples/regex_demo.ebnf")
	if err != nil {
		t.Fatalf("Failed to load regex demo grammar: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		rule    string
		wantErr bool
	}{
		{"identifier", "myVariable123", "identifier", false},
		{"number", "3.14", "number", false},
		{"string", `"hello world"`, "string", false},
		{"email", "user@example.com", "email", false},
		{"url http", "http://example.com", "url", false},
		{"url https", "https://example.com/path", "url", false},
		{"text content", "Hello world!", "text_content", false},
		{"comment", "// this is a comment", "comment", false},
		{"statement", "x = 42;", "statement", false},
		{"program", "x = 10; y = 20;", "program", false},
		{"invalid email", "not-an-email", "email", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(grammar)
			tree, err := p.Parse(tt.input, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && tree.Root == nil {
				t.Error("Parse returned nil root")
			}
		})
	}
}
