# EBNF Grammar Examples

This directory contains example grammars demonstrating various features of the EBNF parser.

## Getting Started

### arithmetic.ebnf
**Difficulty: Beginner**

A simple arithmetic expression grammar showing:
- Basic EBNF structure
- Operator precedence through grammar hierarchy (expr → term → factor)
- Hidden tokens for parentheses and whitespace
- Regex patterns for numbers

Perfect for understanding the basics.

### json.ebnf
**Difficulty: Intermediate**

A complete JSON grammar based on the official specification at https://www.json.org/

Demonstrates:
- Recursive structures (objects and arrays containing values)
- Character-by-character string parsing with escape sequences
- Optional parts (`?`) for number fractions and exponents
- Hidden structural tokens (braces, brackets, quotes)
- Complex choice expressions

This is a real-world example showing how to parse a well-known format.

### regex_demo.ebnf
**Difficulty: Intermediate**

Shows modern regex pattern features:
- `#"regex"` syntax for complex patterns
- Practical patterns (identifiers, emails, URLs)
- Comparison between character-by-character rules vs regex

Good for learning when to use regex patterns vs traditional EBNF.

### instaparse_demo.ebnf
**Difficulty: Advanced**

Demonstrates PEG (Parsing Expression Grammar) style features:
- Ordered choice (`/`) where first match wins
- Hidden expressions (`<>`) to simplify AST
- Expression language with proper precedence

Shows how to write grammars that are deterministic and unambiguous.

## Running the Examples

### Parsing with a Grammar

```go
package main

import (
    "fmt"
    "log"
    "os"
    "github.com/wbrown/ebnf"
    "github.com/wbrown/ebnf/parse"
)

func main() {
    // Load the arithmetic grammar
    grammar, err := ebnf.LoadGrammar("examples/arithmetic.ebnf")
    if err != nil {
        log.Fatal(err)
    }

    // Create a parser
    parser := parse.New(grammar)

    // Parse an arithmetic expression
    input := "2 + 3 * 4"
    tree, err := parser.Parse(input, "expr")
    if err != nil {
        log.Fatal(err)
    }

    // Print the parse tree
    parse.PrintAST(os.Stdout, tree)
}
```

### Using the CLI Tool

A command-line tool is provided for quick testing:

```bash
# Build the tool
go build -o ebnf-parse ./cmd/ebnf-parse

# Parse JSON
echo '{"name": "Alice"}' | ./ebnf-parse -grammar examples/json.ebnf

# Parse arithmetic with compact output
echo "2 + 3 * 4" | ./ebnf-parse -grammar examples/arithmetic.ebnf -rule expr -compact

# Enable debug output to see parsing trace
./ebnf-parse -grammar examples/json.ebnf -input data.json -debug
```

See `examples_test.go` and `parse/*_test.go` for more examples of loading and working with these grammars.
