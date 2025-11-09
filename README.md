# ebnf

A pure Go parser for Extended Backus-Naur Form (EBNF) grammars with support for modern extensions including regex patterns and PEG-style ordered choice.

## Features

- **Standard EBNF**: Terminals, non-terminals, sequences, choices, repetition (`*`, `+`), optionals (`?`)
- **Character Classes**: `[a-z]`, `[0-9]`, `[^abc]` (negation supported)
- **Regex Patterns**: Instaparse-style `#"regex"` for complex patterns
- **Hidden Tokens**: `<token>` syntax to omit tokens from AST
- **PEG Features**: Ordered choice (`/`), lookahead (`!`, `&`)
- **Comments**: C-style `(* comment *)` supported
- **Multiple EBNF Variants**: Supports `=`, `:=`, `::=`, and `<-` as assignment operators

## Installation

```bash
go get github.com/wbrown/ebnf
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/wbrown/ebnf"
)

func main() {
    // Define a simple grammar
    grammarText := `
        expr = term ( <"+"> term | <"-"> term )* ;
        term = number ;
        number = #"[0-9]+" ;
    `

    // Parse the grammar
    grammar, err := ebnf.ParseString(grammarText)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Loaded %d rules\n", len(grammar.Rules))

    // Access rules
    exprRule := grammar.GetRule("expr")
    fmt.Printf("Rule 'expr': %s\n", ebnf.ExpressionType(exprRule.Expression))
}
```

## Loading from Files

```go
grammar, err := ebnf.LoadGrammar("path/to/grammar.ebnf")
if err != nil {
    log.Fatal(err)
}
```

## Parsing Input

Once you have a grammar, you can use it to parse input text:

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
    // Load a grammar
    grammar, err := ebnf.LoadGrammar("examples/json.ebnf")
    if err != nil {
        log.Fatal(err)
    }

    // Create a parser
    parser := parse.New(grammar)

    // Parse some JSON
    input := `{"name": "Alice", "age": 30}`
    tree, err := parser.Parse(input, "json")
    if err != nil {
        log.Fatal(err)
    }

    // Print the parse tree
    parse.PrintAST(os.Stdout, tree)

    // Or use compact format
    parse.CompactPrintAST(os.Stdout, tree)
}
```

The parser produces a parse tree (`*parse.Node`) with:
- `Rule` - The rule name that matched
- `Value` - Consolidated text value (for leaf nodes)
- `Children` - Child nodes
- `Line`, `Column` - Source position
- `Start`, `End` - Character positions in input

## Complete Example: Calculator

A complete working calculator is provided in [`examples/calculator/`](examples/calculator/) showing how to:
- Load and parse using the arithmetic grammar
- Walk the parse tree
- Extract and evaluate expressions
- Handle operator precedence correctly

```bash
cd examples/calculator
go run main.go
```

See the [calculator README](examples/calculator/README.md) for a detailed explanation of how it works.

## Examples

The `examples/` directory contains several demonstration grammars:

### [arithmetic.ebnf](examples/arithmetic.ebnf)
A simple expression grammar showing operator precedence:
```ebnf
expr = term ( <ws>* addop <ws>* term )* ;
term = factor ( <ws>* mulop <ws>* factor )* ;
factor = number | <"("> <ws>* expr <ws>* <")"> ;
```

### [json.ebnf](examples/json.ebnf)
Complete JSON grammar demonstrating:
- Recursive structures (objects, arrays)
- String escape sequences
- Number formats with optional parts
- Hidden structural tokens

### [regex_demo.ebnf](examples/regex_demo.ebnf)
Shows regex pattern usage:
```ebnf
identifier = #"[a-zA-Z][a-zA-Z0-9_]*" ;
email = #"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}" ;
```

### [instaparse_demo.ebnf](examples/instaparse_demo.ebnf)
Demonstrates PEG-style features:
```ebnf
(* Ordered choice - first match wins *)
expr = literal / variable / paren_expr ;
```

## AST Structure

The parser produces a typed AST with these node types:

- `Grammar` - Container for all rules
- `Rule` - Named production rule
- `Terminal` - Literal string/character
- `NonTerminal` - Reference to another rule
- `Sequence` - Concatenation of expressions
- `Choice` - Unordered alternatives (`|`)
- `OrderedChoice` - Ordered alternatives (`/`)
- `Optional` - Optional expression (`?`)
- `Repetition` - Zero or more (`*`)
- `OneOrMore` - One or more (`+`)
- `CharClass` - Character class `[...]`
- `Regex` - Regular expression `#"..."`
- `Predicate` - Negative lookahead (`!`)
- `PositiveLookahead` - Positive lookahead (`&`)
- `Hidden` - Marks hidden expressions

## Supported EBNF Syntax

### Terminals
```ebnf
literal = "text" ;        (* Double quotes *)
char = 'a' ;              (* Single quotes *)
```

### Character Classes
```ebnf
letter = [a-z] | [A-Z] ;  (* Ranges *)
digit = [0-9] ;
not_quote = [^"] ;        (* Negation *)
```

### Regex Patterns
```ebnf
identifier = #"[a-zA-Z_][a-zA-Z0-9_]*" ;
whitespace = #"\s+" ;
```

### Repetition
```ebnf
zero_or_more = element* ;
one_or_more = element+ ;
optional = element? ;
braces = {element} ;      (* Same as * *)
```

### Choices
```ebnf
unordered = a | b | c ;   (* Any order *)
ordered = a / b / c ;     (* First match wins - PEG style *)
```

### Hidden Tokens

There are two ways to hide elements from the parse tree:

#### Hidden References
```ebnf
(* Hide specific references *)
expr = <"("> value <")"> ;     (* Parens hidden *)
rule = <whitespace>* term ;    (* Whitespace hidden *)
```

#### Hidden Rule Definitions
```ebnf
(* Hide the rule wrapper itself *)
<value> = object | array | string | number ;
<digit> = [0-9] ;
```

When you define a rule as hidden (with angle brackets around the name), the rule node is removed from the parse tree and its children are promoted up. This is useful for:
- Removing unnecessary wrapper nodes (like `<value>` which is just a choice)
- Flattening the tree for easier processing
- Keeping only semantically meaningful nodes

**Example:**
```ebnf
pair = key <":"> value ;
<value> = string | number ;
```

Without hidden rule definition:
```
pair
  ├── key: "name"
  └── value           ← unnecessary wrapper
      └── string: "Alice"
```

With hidden rule definition:
```
pair
  ├── key: "name"
  └── string: "Alice"  ← direct child, cleaner
```

### Grouping and Lookahead
```ebnf
grouped = (a | b) c ;          (* Grouping *)
negative = !keyword identifier ; (* Not a keyword *)
positive = &digit number ;      (* Starts with digit *)
```

## API Reference

### Parse Functions

```go
// Parse grammar from string
func ParseString(input string) (*Grammar, error)

// Load grammar from file
func LoadGrammar(filename string) (*Grammar, error)
```

### Grammar Methods

```go
// Find a rule by name
func (g *Grammar) GetRule(name string) *Rule

// Get expression type as string
func ExpressionType(expr Expression) string
```

## Use Cases

This EBNF parser is useful for:

- Building custom language parsers
- Creating DSLs (Domain-Specific Languages)
- Parser generators
- Grammar validation and analysis
- Educational tools for teaching parsing

## License

MIT License - See LICENSE file for details

## Contributing

Contributions welcome! Please feel free to submit issues or pull requests.
