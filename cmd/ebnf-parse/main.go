package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/wbrown/ebnf"
	"github.com/wbrown/ebnf/parse"
)

var (
	grammarFile = flag.String("grammar", "", "Path to EBNF grammar file (required)")
	inputFile   = flag.String("input", "", "Path to input file to parse (default: stdin)")
	startRule   = flag.String("rule", "", "Start rule name (default: first rule in grammar)")
	compact     = flag.Bool("compact", false, "Use compact output format")
	debug       = flag.Bool("debug", false, "Enable debug output")
	showInput   = flag.Bool("show-input", false, "Show input text before parsing")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	// Validate required flags
	if *grammarFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -grammar flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load grammar
	grammar, err := ebnf.LoadGrammar(*grammarFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading grammar: %v\n", err)
		os.Exit(1)
	}

	// Determine start rule - use first rule if not specified
	rule := *startRule
	if rule == "" {
		if len(grammar.Rules) == 0 {
			fmt.Fprintln(os.Stderr, "Error: grammar has no rules")
			os.Exit(1)
		}
		rule = grammar.Rules[0].Name
		if *debug {
			fmt.Fprintf(os.Stderr, "Using first rule as start rule: %q\n", rule)
		}
	}

	// Read input
	var input string
	if *inputFile == "" {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		input = string(data)
	} else {
		// Read from file
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
		input = string(data)
	}

	// Show input if requested
	if *showInput {
		fmt.Println("=== Input ===")
		fmt.Println(input)
		fmt.Println()
	}

	// Create parser
	parser := parse.New(grammar)
	parser.Debug = *debug

	// Parse input
	tree, err := parser.Parse(input, rule)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		if *debug {
			fmt.Fprintln(os.Stderr, "\n=== Debug Log ===")
			fmt.Fprintln(os.Stderr, parser.GetDebugLog())
		}
		os.Exit(1)
	}

	// Print parse tree
	if *compact {
		fmt.Println("=== Parse Tree (Compact) ===")
		parse.CompactPrintAST(os.Stdout, tree)
	} else {
		fmt.Println("=== Parse Tree ===")
		parse.PrintAST(os.Stdout, tree)
	}

	if *debug {
		fmt.Println("\n=== Debug Log ===")
		fmt.Println(parser.GetDebugLog())
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `ebnf-parse - Parse input files using EBNF grammars

Usage:
  ebnf-parse -grammar <file> [-rule <name>] [-input <file>] [options]

Description:
  Parse input text using an EBNF grammar and display the resulting parse tree.
  If no input file is specified, reads from stdin.
  If no start rule is specified, uses the first rule in the grammar.

Required Flags:
  -grammar string
        Path to EBNF grammar file

Optional Flags:
  -rule string
        Start rule name (default: first rule in grammar)
  -input string
        Path to input file to parse (default: stdin)
  -compact
        Use compact output format (collapses single-child chains)
  -debug
        Enable debug output (shows parsing trace)
  -show-input
        Display the input text before parsing

Examples:
  # Parse an arithmetic expression
  echo "2 + 3 * 4" | ebnf-parse -grammar arithmetic.ebnf -rule expr

  # Parse a JSON file
  ebnf-parse -grammar json.ebnf -rule json -input data.json

  # Parse with debug output
  ebnf-parse -grammar test.ebnf -rule program -input code.txt -debug

  # Use compact format
  ebnf-parse -grammar test.ebnf -rule expr -input expr.txt -compact

Exit Status:
  0   Parsing succeeded
  1   Parsing failed or invalid usage

`)
}
