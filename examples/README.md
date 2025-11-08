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

Each example can be loaded and inspected:

```go
package main

import (
    "fmt"
    "log"
    "github.com/wbrown/ebnf"
)

func main() {
    grammar, err := ebnf.LoadGrammar("examples/arithmetic.ebnf")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Loaded grammar with %d rules\n", len(grammar.Rules))

    // Inspect a specific rule
    expr := grammar.GetRule("expr")
    if expr != nil {
        fmt.Printf("Expression rule type: %s\n", ebnf.ExpressionType(expr.Expression))
    }
}
```

See `examples_test.go` for more examples of loading and working with these grammars.
