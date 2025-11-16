package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/wbrown/ebnf"
	"github.com/wbrown/ebnf/parse"
)

func main() {
	// Load the arithmetic grammar
	grammar, err := ebnf.LoadGrammar("../arithmetic.ebnf")
	if err != nil {
		log.Fatal(err)
	}

	// Create a parser
	parser := parse.New(grammar)

	// Define transformation rules - each operation is a simple function!
	// This is much cleaner than the ~150 lines of manual tree walking code
	// that would be needed without the Transform API.
	transforms := parse.TransformMap{
		// Parse numbers from their string representation
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},

		// Binary operations - just plain math!
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 {
			if b == 0 {
				panic("division by zero")
			}
			return a / b
		},

		// Unary negation
		"neg": func(a float64) float64 { return -a },
	}

	// Parse and evaluate expressions
	expressions := []string{
		"5",
		"-5",
		"2 + 3",
		"2 + 3 * 4",
		"(2 + 3) * 4",
		"10 / 2 - 3",
		"1.5 + 2.5 * 2",
		"-5 + 3",
		"10 / -2",
		"2 * -3 + 4",
		"-(2 + 3)",
	}

	for _, expr := range expressions {
		fmt.Printf("Expression: %s\n", expr)

		// Parse the expression
		tree, err := parser.Parse(expr, "expr")
		if err != nil {
			fmt.Printf("  Parse error: %v\n\n", err)
			continue
		}

		// Optionally show the parse tree
		if len(os.Args) > 1 && os.Args[1] == "-tree" {
			fmt.Println("  Parse tree:")
			parse.PrintAST(os.Stdout, tree)
			fmt.Println()
		}

		// Transform/evaluate the expression in one step!
		result, err := parse.Transform(tree, transforms)
		if err != nil {
			fmt.Printf("  Eval error: %v\n\n", err)
			continue
		}

		fmt.Printf("  Result: %.2f\n\n", result)
	}
}
