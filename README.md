# ebnf

A production-ready EBNF parser and transformation framework for Go. Parse text with EBNF grammars and transform parse trees into any structure—from simple calculators to full compilers and document processors.

## Features

- **Powerful Tree Transformation**: Multi-pass, top-down, context-aware transforms with rich error handling
- **Standard EBNF**: Terminals, non-terminals, sequences, choices, repetition (`*`, `+`), optionals (`?`)
- **Modern Extensions**:
  - Regex patterns: Instaparse-style `#"regex"` for complex patterns
  - Character classes: `[a-z]`, `[0-9]`, `[^abc]` with negation support
  - Case-insensitive literals: `'SELECT'i` matches "select", "SELECT", "SeLeCt"
  - PEG features: Ordered choice (`/`), lookahead (`!`, `&`)
  - Comments: C-style `(* comment *)`
  - Multiple assignment operators: `=`, `:=`, `::=`, `<-`
- **Hidden Tokens**: `<token>` syntax and `<rule>` definitions to omit structural noise from AST
- **Production Ready**: Comprehensive error messages with source positions and visual highlighting
- **Pure Go**: No external dependencies or code generation required

## Why This Library?

Most parser libraries give you a parse tree and leave you to figure out what to do with it. This library provides a complete transformation framework that makes it easy to go from text to your desired output:

```go
// Define grammar -> Parse input -> Transform to result
// All with automatic type conversion, error handling, and position tracking
```

**Perfect for:**
- **Compilers and Interpreters**: Multi-pass transforms, scoping, symbol tables
- **Document Processors**: Markdown, reStructuredText, custom markup languages  
- **Configuration Languages**: TOML, YAML-like formats, DSLs
- **Data Formats**: Custom serialization, query languages
- **Template Engines**: With variable substitution and logic

## Installation

```bash
go get github.com/wbrown/ebnf
```

## Quick Start: Calculator in ~20 Lines

```go
package main

import (
    "fmt"
    "strconv"
    "github.com/wbrown/ebnf"
    "github.com/wbrown/ebnf/parse"
)

func main() {
    // Load grammar
    grammar, _ := ebnf.LoadGrammar("examples/arithmetic.ebnf")
    parser := parse.New(grammar)
    
    // Define transformations - one function per rule
    transforms := parse.TransformMap{
        "number": func(s string) float64 {
            f, _ := strconv.ParseFloat(s, 64)
            return f
        },
        "add": func(a, b float64) float64 { return a + b },
        "sub": func(a, b float64) float64 { return a - b },
        "mul": func(a, b float64) float64 { return a * b },
        "div": func(a, b float64) float64 { return a / b },
        "neg": func(a float64) float64 { return -a },
    }
    
    // Parse and evaluate
    tree, _ := parser.Parse("(2 + 3) * 4 - -5", "expr")
    result, _ := parse.Transform(tree, transforms)
    fmt.Printf("Result: %.2f\n", result) // Result: 25.00
}
```

That's it! The transform system handles:
- ✅ Type conversions (string → float64)
- ✅ Bottom-up evaluation (leaves to root)
- ✅ Tree traversal
- ✅ Error propagation

## Understanding Transforms: A Compilation Pipeline

**Here's the key insight:** This library transforms source text into structures your code can use. Parse once, transform once, then use the result efficiently.

```
Source Text → Parse Tree → Transform → Usable Output (AST, structs, bytecode, etc.)
```

The transforms run at **parse time** (once), not runtime. You're building the compiler frontend, not the runtime:

| What You're Building | Parse | Transform To | Then Use For |
|---------------------|-------|--------------|--------------|
| JSON Parser | JSON text | Go maps/structs | Fast lookups at runtime |
| Template Engine | Template syntax | Compiled template | Fast rendering (no re-parsing) |
| SQL Query | SQL text | Query AST | Optimized execution |
| Config Processor | Config file | Validated structs | Application startup |
| Compiler | Source code | AST/Bytecode/IR | Optimization & codegen |
| Markdown Converter | Markdown | HTML/PDF | Static site generation |

**Example: The calculator isn't evaluating expressions in a hot loop.** It parses `"2 + 3 * 4"` once, transforms to the result `14.0`, and you're done. For a real application, you'd transform to an AST or bytecode that your fast runtime interpreter/VM executes.

### Real-World Pattern

```go
// Parse + transform once (at load/compile time)
configTree, _ := parser.Parse(configFile, "config")
config, _ := parse.Transform(configTree, configTransforms)
// Result: validated *Config struct

// Use many times (at runtime, fast)
for request := range requests {
    if config.EnableFeature {  // No parsing overhead
        handleRequest(request)
    }
}
```

This is a **compiler frontend tool**, not a runtime interpreter. Your production code uses the transformed output, not the parser.

### What You Can Build

- **Expression evaluators**: Calculator, spreadsheet formulas, query languages
- **Compilers**: Parse source → Transform to AST → Transform to bytecode → Transform to machine code (each pass is a transform)
- **Template engines**: Parse template → Transform with variable substitution
- **Configuration processors**: Parse config → Transform to validated structs
- **Document converters**: Parse Markdown → Transform to HTML/PDF/LaTeX
- **DSLs**: Parse domain language → Transform to actions/API calls

The calculator demonstrates this pattern. Use the same approach to build any language processor.

## Transform System

The transform system provides multiple modes for different use cases.

### Basic Usage

```go
transforms := parse.TransformMap{
    "number": func(s string) float64 {
        f, _ := strconv.ParseFloat(s, 64)
        return f
    },
    "add": func(a, b float64) float64 { return a + b },
}

result, err := parse.Transform(tree, transforms)
```

### Advanced: Access Source Positions

```go
transforms := parse.TransformMap{
    "number": func(node *parse.Node, s string) (*Number, error) {
        val, err := strconv.Atoi(s)
        if err != nil {
            return nil, fmt.Errorf("invalid number at line %d", node.Line)
        }
        return &Number{Value: val, Line: node.Line}, nil
    },
}
```

### Advanced: Multi-Pass Transformations

Perfect for document processors and multi-stage compilation:

```go
// Pass 1: Transform inline elements
pass1 := parse.TransformMap{
    "bold": func(text string) string {
        return fmt.Sprintf("<strong>%s</strong>", text)
    },
    "italic": func(text string) string {
        return fmt.Sprintf("<em>%s</em>", text)
    },
}

// Pass 2: Transform block elements
pass2 := parse.TransformMap{
    "paragraph": func(inlines ...string) string {
        return fmt.Sprintf("<p>%s</p>", strings.Join(inlines, ""))
    },
    "document": func(blocks ...string) string {
        return strings.Join(blocks, "\n")
    },
}

// Apply both passes sequentially
html, err := parse.TransformMultiPass(tree, []parse.TransformMap{pass1, pass2})
```

See `examples/markdown/` for a complete Markdown-to-HTML transformer.

### Advanced: Context-Aware Transforms

Access parent nodes, siblings, and tree context:

```go
transforms := parse.TransformMap{
    "list_item": func(ctx *parse.TransformContext, text string) string {
        // Add comma unless it's the last item
        if !ctx.IsLast() {
            return text + ", "
        }
        return text
    },
}
```

### Advanced: Top-Down Transforms

Process parent before children—useful for scoping and symbol tables:

```go
// Build symbol table top-down
var symbolTable = make(map[string]*Function)

transforms := parse.TransformMap{
    "function": func(ctx *parse.TransformContext, name string, body interface{}) {
        symbolTable[name] = &Function{Name: name}
        // Children will see this in symbol table
    },
}

result, err := parse.TransformTopDown(tree, transforms)
```

### Rich Error Messages

Errors automatically include source position and visual highlighting:

```
transform error in rule "number" at line 3, column 5
  source: "abc + 123"
          ^^^
  error: strconv.Atoi: parsing "abc": invalid syntax
```

### Complete Documentation

- **[parse/transform.md](parse/transform.md)** - Complete transform guide
- **[parse/transform_quick_start.md](parse/transform_quick_start.md)** - Quick reference
- **[parse/error_api_usage.md](parse/error_api_usage.md)** - Error handling

## Real-World Examples

### Markdown to HTML

Complete working example in `examples/markdown/`:
- Multi-pass transformation (inline → block elements)
- Context-aware list handling
- Type preservation between passes
- Rich error reporting

```bash
cd examples/markdown
go run markdown_to_html.go example.md
```

### Calculator

Complete calculator with precedence and parentheses in `examples/calculator/`:

```bash
cd examples/calculator
go run main.go
```

## Parsing Input

```go
// Load grammar
grammar, err := ebnf.LoadGrammar("examples/json.ebnf")
if err != nil {
    log.Fatal(err)
}

// Create parser
parser := parse.New(grammar)

// Parse input
tree, err := parser.Parse(`{"name": "Alice", "age": 30}`, "json")
if err != nil {
    log.Fatal(err)
}

// The parse tree includes:
// - Rule names
// - Source positions (line, column, start, end)
// - Terminal values
// - Child nodes
```

## EBNF Grammar Syntax

### Basic Elements

```ebnf
(* Terminals *)
literal = "text" ;
char = 'a' ;

(* Character classes *)
letter = [a-zA-Z] ;
digit = [0-9] ;
not_quote = [^"] ;

(* Regex patterns *)
identifier = #"[a-zA-Z_][a-zA-Z0-9_]*" ;

(* Case-insensitive literals *)
keyword = 'SELECT'i ;             (* Matches select, SELECT, SeLeCt *)
sql = "FROM"i table ;

(* Repetition *)
zero_or_more = element* ;
one_or_more = element+ ;
optional = element? ;

(* Choices *)
unordered = a | b | c ;    (* Any match wins *)
ordered = a / b / c ;      (* First match wins - PEG style *)

(* Grouping *)
grouped = (a | b) c ;

(* Lookahead *)
not_keyword = !keyword identifier ;
starts_with_digit = &digit number ;
```

### Hidden Tokens

Remove structural noise from your parse tree:

```ebnf
(* Hide specific references *)
expr = <"("> value <")"> ;        (* Parens hidden *)
rule = <whitespace>* term ;       (* Whitespace hidden *)

(* Hide entire rule wrapper *)
<value> = object | array | string | number ;
<whitespace> = " " | "\t" | "\n" ;
```

Hidden rule definitions (with `<>` around the name) remove the wrapper node entirely, flattening the tree.

## Example Grammars

### Arithmetic with Clean Parse Trees

[`examples/arithmetic.ebnf`](examples/arithmetic.ebnf) uses right recursion to create S-expression-like trees:

```ebnf
add = mul_div <"+"> add_sub ;
mul = unary <"*"> mul_div ;
```

This creates trees like `add(2, mul(3, 4))` instead of flat lists, making transformations trivial.

### JSON Parser

[`examples/json.ebnf`](examples/json.ebnf) demonstrates:
- Recursive structures (objects, arrays)
- String escape sequences
- Hidden structural tokens for clean trees

### Regex Patterns

[`examples/regex_demo.ebnf`](examples/regex_demo.ebnf):

```ebnf
identifier = #"[a-zA-Z][a-zA-Z0-9_]*" ;
email = #"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}" ;
```

## Use Cases

This library is suitable for:

### ✅ Production Compilers and Interpreters
- Multi-pass compilation (parse → resolve → typecheck → codegen)
- Symbol table construction with scoping
- Source position tracking for error messages
- Type-safe transformations

### ✅ Document Processors
- Markdown, reStructuredText, AsciiDoc
- Custom markup languages
- Multi-pass processing (inline → block → layout)
- Context-aware formatting

### ✅ Configuration Languages
- Custom DSLs for your application
- TOML/YAML-like formats
- Validation with rich error messages
- Type-safe configuration objects

### ✅ Data Formats and Query Languages
- Custom serialization formats
- SQL-like query languages
- Template languages
- Expression evaluators

### ✅ Development Tools
- Syntax highlighters
- Code formatters
- Linters and analyzers
- Educational parsing tools

## API Reference

### Loading Grammars

```go
// From string
grammar, err := ebnf.ParseString(grammarText)

// From file
grammar, err := ebnf.LoadGrammar("path/to/grammar.ebnf")

// Access rules
rule := grammar.GetRule("expr")
```

### Parsing

```go
// Create parser
parser := parse.New(grammar)

// Parse input
tree, err := parser.Parse(input, "start_rule")

// Print parse tree
parse.PrintAST(os.Stdout, tree)          // Full tree
parse.CompactPrintAST(os.Stdout, tree)   // Compact format
```

### Parser Options

```go
// Terminals are case-sensitive by default.
// Use WithCaseInsensitive(true) for global case-insensitive matching,
// or use the 'i' suffix on individual terminals: 'SELECT'i
parser := parse.New(grammar, parse.WithCaseInsensitive(true))
```

Per-terminal case-insensitivity uses the `i` suffix in the grammar:

```ebnf
(* Only keywords are case-insensitive, identifiers are case-sensitive *)
query = 'SELECT'i column 'FROM'i table ;
column = identifier ;
table = identifier ;
identifier = #"[a-zA-Z_]+" ;
```

### Transforming

```go
// Bottom-up (default)
result, err := parse.Transform(tree, transforms)

// Multi-pass
result, err := parse.TransformMultiPass(tree, []parse.TransformMap{pass1, pass2})

// Top-down
result, err := parse.TransformTopDown(tree, transforms)
```

## Parse Tree Structure

The parser produces `*parse.Node` with:

```go
type Node struct {
    Rule             string      // Grammar rule name
    Value            string      // Terminal text value
    TransformedValue interface{} // Typed value from transforms
    Children         []*Node     // Child nodes
    Line, Column     int         // Source position
    Start, End       int         // Character offsets
}
```

## Comparison to Other Tools

| Feature | This Library | ANTLR | Yacc/Bison | Hand-Written |
|---------|--------------|-------|------------|--------------|
| Pure Go | ✅ | ❌ (Java tool) | ❌ (C tool) | ✅ |
| No code gen | ✅ | ❌ | ❌ | ✅ |
| Transform framework | ✅ | ⚠️ (listeners) | ❌ | ❌ |
| Multi-pass | ✅ | ❌ | ❌ | ⚠️ (manual) |
| Rich errors | ✅ | ⚠️ | ❌ | ⚠️ (manual) |
| Learning curve | Low | High | High | Medium |
| Runtime deps | None | Runtime JAR | C runtime | None |

## Performance

The transform system uses reflection for function signature detection but only during initialization. Transform execution is direct function calls with no reflection overhead.

For large inputs, consider:
- Streaming parsers for multi-GB files
- This library excels at medium-sized inputs (source files, documents, configs)

## Contributing

Contributions welcome! Please submit issues or pull requests.

## License

MIT License - See LICENSE file for details

## Related Projects

- [Instaparse](https://github.com/Engelberg/instaparse) - Clojure parser library that inspired the transform system
- [ANTLR](https://www.antlr.org/) - Full-featured parser generator
- [PEG parsers](https://en.wikipedia.org/wiki/Parsing_expression_grammar) - For background on ordered choice

