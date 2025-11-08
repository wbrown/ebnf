package ebnf

import (
	"fmt"
	"os"
)

// LoadGrammar loads and parses an EBNF grammar from a file
func LoadGrammar(filename string) (*Grammar, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseString(string(content))
}

// PrintGrammarSummary prints a summary of the grammar
func PrintGrammarSummary(g *Grammar) {
	fmt.Printf("Grammar contains %d rules:\n", len(g.Rules))
	for _, rule := range g.Rules {
		fmt.Printf("  %s\n", rule.Name)
	}
}

// GetRule finds a rule by name
func (g *Grammar) GetRule(name string) *Rule {
	for _, rule := range g.Rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}

// ExpressionType returns a string describing the type of expression
func ExpressionType(expr Expression) string {
	switch e := expr.(type) {
	case *Terminal:
		if e.Hidden {
			return fmt.Sprintf("hidden(%q)", e.Value)
		}
		return fmt.Sprintf("terminal(%q)", e.Value)
	case *NonTerminal:
		return fmt.Sprintf("rule(%s)", e.Name)
	case *Sequence:
		return fmt.Sprintf("sequence[%d]", len(e.Elements))
	case *Choice:
		return fmt.Sprintf("choice[%d]", len(e.Alternatives))
	case *Optional:
		return "optional"
	case *Repetition:
		return "zero_or_more"
	case *OneOrMore:
		return "one_or_more"
	case *Group:
		return "group"
	case *Predicate:
		return "lookahead"
	case *CharClass:
		return "char_class"
	case *Empty:
		return "empty"
	default:
		return "unknown"
	}
}
