package parse

import (
	"testing"

	"github.com/wbrown/ebnf"
)

func TestCaseInsensitiveMatching(t *testing.T) {
	tests := []struct {
		name    string
		grammar string
		input   string
		rule    string
		wantErr bool
	}{
		// Per-terminal case-insensitive matching with 'i' suffix
		{
			name:    "case-insensitive matches uppercase",
			grammar: `keyword = 'select'i ;`,
			input:   "SELECT",
			rule:    "keyword",
			wantErr: false,
		},
		{
			name:    "case-insensitive matches lowercase",
			grammar: `keyword = 'SELECT'i ;`,
			input:   "select",
			rule:    "keyword",
			wantErr: false,
		},
		{
			name:    "case-insensitive matches mixed case",
			grammar: `keyword = 'select'i ;`,
			input:   "SeLeCt",
			rule:    "keyword",
			wantErr: false,
		},
		{
			name:    "case-sensitive fails on wrong case",
			grammar: `keyword = 'select' ;`,
			input:   "SELECT",
			rule:    "keyword",
			wantErr: true,
		},
		{
			name:    "case-sensitive matches exact case",
			grammar: `keyword = 'select' ;`,
			input:   "select",
			rule:    "keyword",
			wantErr: false,
		},
		// Sequence with mixed case sensitivity
		{
			name:    "SQL-like query with case-insensitive keywords",
			grammar: `query = 'SELECT'i ' ' column ' ' 'FROM'i ' ' table ; column = 'name' ; table = 'users' ;`,
			input:   "select name from users",
			rule:    "query",
			wantErr: false,
		},
		{
			name:    "SQL-like query uppercase",
			grammar: `query = 'SELECT'i ' ' column ' ' 'FROM'i ' ' table ; column = 'name' ; table = 'users' ;`,
			input:   "SELECT name FROM users",
			rule:    "query",
			wantErr: false,
		},
		// Unicode case folding
		{
			name:    "unicode case folding",
			grammar: `word = "café"i ;`,
			input:   "CAFÉ",
			rule:    "word",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grammar, err := ebnf.ParseString(tt.grammar)
			if err != nil {
				t.Fatalf("Failed to parse grammar: %v", err)
			}

			p := New(grammar)
			_, err = p.Parse(tt.input, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestGlobalCaseInsensitiveOption(t *testing.T) {
	grammar, err := ebnf.ParseString(`keyword = 'select' ; table = 'users' ;`)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	// Without global option - should fail
	t.Run("without global option fails on wrong case", func(t *testing.T) {
		p := New(grammar)
		_, err := p.Parse("SELECT", "keyword")
		if err == nil {
			t.Error("Expected error for case mismatch without global option")
		}
	})

	// With global option - should succeed
	t.Run("with global option matches any case", func(t *testing.T) {
		p := New(grammar, WithCaseInsensitive(true))
		_, err := p.Parse("SELECT", "keyword")
		if err != nil {
			t.Errorf("Unexpected error with global case-insensitive: %v", err)
		}
	})

	t.Run("global option matches mixed case", func(t *testing.T) {
		p := New(grammar, WithCaseInsensitive(true))
		_, err := p.Parse("SeLeCt", "keyword")
		if err != nil {
			t.Errorf("Unexpected error with global case-insensitive: %v", err)
		}
	})
}

func TestCaseInsensitivePreservesOriginalCase(t *testing.T) {
	grammar, err := ebnf.ParseString(`keyword = 'select'i ;`)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"SELECT", "SELECT"},
		{"select", "select"},
		{"SeLeCt", "SeLeCt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(grammar)
			tree, err := p.Parse(tt.input, "keyword")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tree.Root == nil {
				t.Fatal("Expected non-nil root")
			}

			// The root is the rule node, the terminal value is in the first child
			if len(tree.Root.Children) == 0 {
				t.Fatal("Expected at least one child node")
			}

			// The matched value should preserve the original case from input
			matchedValue := tree.Root.Children[0].Value
			if matchedValue != tt.expected {
				t.Errorf("Expected matched value %q, got %q", tt.expected, matchedValue)
			}
		})
	}
}

func TestPerTerminalOverridesGlobal(t *testing.T) {
	// Grammar where 'strict' is explicitly case-sensitive (no i suffix)
	// Even with global case-insensitive, the explicit terminal should be case-sensitive
	// Note: current implementation has per-terminal OR global, so explicit per-terminal
	// case-insensitive works, but we can't explicitly mark something as case-sensitive
	// when global is on. This test documents current behavior.

	grammar, err := ebnf.ParseString(`query = 'SELECT'i ' ' identifier ; identifier = 'users' ;`)
	if err != nil {
		t.Fatalf("Failed to parse grammar: %v", err)
	}

	t.Run("case-insensitive keyword with case-sensitive identifier", func(t *testing.T) {
		p := New(grammar) // No global option
		_, err := p.Parse("select users", "query")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("case-sensitive identifier fails", func(t *testing.T) {
		p := New(grammar) // No global option
		_, err := p.Parse("select USERS", "query")
		if err == nil {
			t.Error("Expected error for case-sensitive identifier mismatch")
		}
	})
}
