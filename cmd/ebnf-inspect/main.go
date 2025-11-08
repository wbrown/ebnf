// Command ebnf-inspect loads and displays information about EBNF grammar files.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/wbrown/ebnf"
)

func main() {
	var (
		rule   = flag.String("rule", "", "Show details for a specific rule")
		list   = flag.Bool("list", false, "List all rules")
		stats  = flag.Bool("stats", false, "Show grammar statistics")
		deps   = flag.Bool("deps", false, "Show rule dependencies")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <grammar.ebnf>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s grammar.ebnf              # Show summary\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list grammar.ebnf        # List all rules\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -rule expr grammar.ebnf   # Show specific rule\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -stats grammar.ebnf       # Show statistics\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	filename := flag.Arg(0)
	grammar, err := ebnf.LoadGrammar(filename)
	if err != nil {
		log.Fatalf("Failed to load grammar: %v", err)
	}

	// If specific rule requested
	if *rule != "" {
		showRule(grammar, *rule)
		return
	}

	// If list requested
	if *list {
		listRules(grammar)
		return
	}

	// If stats requested
	if *stats {
		showStats(grammar)
		return
	}

	// If deps requested
	if *deps {
		showDependencies(grammar)
		return
	}

	// Default: show summary
	showSummary(grammar, filename)
}

func showSummary(g *ebnf.Grammar, filename string) {
	fmt.Printf("Grammar: %s\n", filename)
	fmt.Printf("Rules: %d\n\n", len(g.Rules))

	fmt.Println("First 10 rules:")
	for i, rule := range g.Rules {
		if i >= 10 {
			fmt.Printf("... and %d more\n", len(g.Rules)-10)
			break
		}
		fmt.Printf("  %-20s %s\n", rule.Name, ebnf.ExpressionType(rule.Expression))
	}

	fmt.Println("\nUse -list to see all rules, -rule <name> to inspect a specific rule")
}

func listRules(g *ebnf.Grammar) {
	fmt.Printf("Rules (%d):\n", len(g.Rules))
	for _, rule := range g.Rules {
		fmt.Printf("  %-30s %s\n", rule.Name, ebnf.ExpressionType(rule.Expression))
	}
}

func showRule(g *ebnf.Grammar, name string) {
	rule := g.GetRule(name)
	if rule == nil {
		log.Fatalf("Rule '%s' not found", name)
	}

	fmt.Printf("Rule: %s\n", rule.Name)
	fmt.Printf("Type: %s\n\n", ebnf.ExpressionType(rule.Expression))
	fmt.Printf("Definition:\n")
	printExpression(rule.Expression, "  ")
}

func showStats(g *ebnf.Grammar) {
	stats := make(map[string]int)

	for _, rule := range g.Rules {
		countTypes(rule.Expression, stats)
	}

	fmt.Printf("Grammar Statistics\n")
	fmt.Printf("==================\n")
	fmt.Printf("Total rules: %d\n\n", len(g.Rules))
	fmt.Printf("Expression types:\n")

	for typ, count := range stats {
		fmt.Printf("  %-20s %d\n", typ, count)
	}
}

func showDependencies(g *ebnf.Grammar) {
	deps := make(map[string][]string)

	for _, rule := range g.Rules {
		deps[rule.Name] = findDependencies(rule.Expression)
	}

	fmt.Printf("Rule Dependencies\n")
	fmt.Printf("=================\n\n")

	for _, rule := range g.Rules {
		if len(deps[rule.Name]) > 0 {
			fmt.Printf("%s → %s\n", rule.Name, strings.Join(deps[rule.Name], ", "))
		}
	}
}

func printExpression(expr ebnf.Expression, indent string) {
	switch e := expr.(type) {
	case *ebnf.Terminal:
		if e.Hidden {
			fmt.Printf("%s<\"%s\">\n", indent, e.Value)
		} else {
			fmt.Printf("%s\"%s\"\n", indent, e.Value)
		}
	case *ebnf.NonTerminal:
		if e.Hidden {
			fmt.Printf("%s<%s>\n", indent, e.Name)
		} else {
			fmt.Printf("%s%s\n", indent, e.Name)
		}
	case *ebnf.Regex:
		fmt.Printf("%s#\"%s\"\n", indent, e.Pattern)
	case *ebnf.Sequence:
		fmt.Printf("%sSequence:\n", indent)
		for _, elem := range e.Elements {
			printExpression(elem, indent+"  ")
		}
	case *ebnf.Choice:
		fmt.Printf("%sChoice (|):\n", indent)
		for _, alt := range e.Alternatives {
			printExpression(alt, indent+"  ")
		}
	case *ebnf.OrderedChoice:
		fmt.Printf("%sOrderedChoice (/):\n", indent)
		for _, alt := range e.Alternatives {
			printExpression(alt, indent+"  ")
		}
	case *ebnf.Optional:
		fmt.Printf("%sOptional:\n", indent)
		printExpression(e.Expr, indent+"  ")
	case *ebnf.Repetition:
		fmt.Printf("%sRepetition (*):\n", indent)
		printExpression(e.Expr, indent+"  ")
	case *ebnf.OneOrMore:
		fmt.Printf("%sOneOrMore (+):\n", indent)
		printExpression(e.Expr, indent+"  ")
	case *ebnf.CharClass:
		fmt.Printf("%sCharClass: %s\n", indent, e.String())
	default:
		fmt.Printf("%s%T\n", indent, e)
	}
}

func countTypes(expr ebnf.Expression, stats map[string]int) {
	switch e := expr.(type) {
	case *ebnf.Terminal:
		stats["Terminal"]++
	case *ebnf.NonTerminal:
		stats["NonTerminal"]++
	case *ebnf.Regex:
		stats["Regex"]++
	case *ebnf.Sequence:
		stats["Sequence"]++
		for _, elem := range e.Elements {
			countTypes(elem, stats)
		}
	case *ebnf.Choice:
		stats["Choice"]++
		for _, alt := range e.Alternatives {
			countTypes(alt, stats)
		}
	case *ebnf.OrderedChoice:
		stats["OrderedChoice"]++
		for _, alt := range e.Alternatives {
			countTypes(alt, stats)
		}
	case *ebnf.Optional:
		stats["Optional"]++
		countTypes(e.Expr, stats)
	case *ebnf.Repetition:
		stats["Repetition"]++
		countTypes(e.Expr, stats)
	case *ebnf.OneOrMore:
		stats["OneOrMore"]++
		countTypes(e.Expr, stats)
	case *ebnf.CharClass:
		stats["CharClass"]++
	}
}

func findDependencies(expr ebnf.Expression) []string {
	deps := make(map[string]bool)
	collectDeps(expr, deps)

	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}
	return result
}

func collectDeps(expr ebnf.Expression, deps map[string]bool) {
	switch e := expr.(type) {
	case *ebnf.NonTerminal:
		deps[e.Name] = true
	case *ebnf.Sequence:
		for _, elem := range e.Elements {
			collectDeps(elem, deps)
		}
	case *ebnf.Choice:
		for _, alt := range e.Alternatives {
			collectDeps(alt, deps)
		}
	case *ebnf.OrderedChoice:
		for _, alt := range e.Alternatives {
			collectDeps(alt, deps)
		}
	case *ebnf.Optional:
		collectDeps(e.Expr, deps)
	case *ebnf.Repetition:
		collectDeps(e.Expr, deps)
	case *ebnf.OneOrMore:
		collectDeps(e.Expr, deps)
	case *ebnf.Group:
		collectDeps(e.Expr, deps)
	case *ebnf.Hidden:
		collectDeps(e.Expr, deps)
	}
}
