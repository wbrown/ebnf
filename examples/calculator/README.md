# Calculator Example

This is a complete end-to-end example showing how to use the EBNF parser to build a working calculator.

## What it demonstrates

1. **Loading a grammar** - Uses the `arithmetic.ebnf` grammar
2. **Parsing input** - Converts expression strings to parse trees
3. **Walking the tree** - Implements an evaluator that traverses the parse tree
4. **Extracting values** - Converts terminal nodes to numbers
5. **Computing results** - Performs arithmetic operations based on tree structure

## Running the example

```bash
# From the examples/calculator directory
go run main.go

# Show parse trees along with results
go run main.go -tree
```

## How it works

### The Grammar

The arithmetic grammar (`../arithmetic.ebnf`) defines expressions with proper operator precedence:

```ebnf
expr = term ( addop term )* ;        (* Addition/subtraction *)
term = factor ( mulop factor )* ;    (* Multiplication/division *)
factor = number | "(" expr ")" ;     (* Numbers or grouped expressions *)
```

This grammar structure ensures:
- Multiplication/division bind tighter than addition/subtraction
- Parentheses can override precedence
- Left-to-right evaluation within the same precedence level

### The Parse Tree

When parsing `2 + 3 * 4`, the grammar produces:

```
expr
├── term
│   └── factor
│       └── number: "2"
├── addop: "+"
└── term
    ├── factor
    │   └── number: "3"
    ├── mulop: "*"
    └── factor
        └── number: "4"
```

Notice how `3 * 4` is grouped under a single `term`, ensuring multiplication happens first.

### The Evaluator

The `Evaluator` type implements tree walking:

```go
func (e *Evaluator) Eval(node *parse.Node) (float64, error) {
    switch node.Rule {
    case "expr":
        return e.evalExpr(node)
    case "term":
        return e.evalTerm(node)
    case "factor":
        return e.evalFactor(node)
    case "number":
        return e.evalNumber(node)
    }
}
```

Each method handles one grammar rule:

- `evalExpr` - Processes addition/subtraction from left to right
- `evalTerm` - Processes multiplication/division from left to right
- `evalFactor` - Extracts numbers or evaluates parenthesized expressions
- `evalNumber` - Converts terminal values to `float64`

### Key Patterns

**Processing repetitions:**
```go
// expr = term ( addop term )*
result := evalFirst(node.Children[0])
for i := 1; i < len(node.Children); i += 2 {
    op := node.Children[i]       // operator
    term := node.Children[i+1]   // operand
    result = apply(result, op, term)
}
```

**Handling choices:**
```go
// factor = number | "(" expr ")"
// Just evaluate the child - it's either a number or expr
return e.Eval(node.Children[0])
```

**Extracting leaf values:**
```go
// Terminal nodes have empty Rule and non-empty Value
if node.Rule == "" && node.Value != "" {
    return strconv.ParseFloat(node.Value, 64)
}
```

## Extending this example

You could extend this calculator to support:

- **Variables**: Add a symbol table to store variable values
- **Functions**: Add `sin`, `cos`, `sqrt`, etc.
- **More operators**: Add exponentiation, modulo, etc.
- **Error recovery**: Continue parsing after errors to report multiple issues
- **Source positions**: Use `node.Line` and `node.Column` for better error messages

The pattern is always the same:
1. Define the grammar
2. Parse to get a tree
3. Walk the tree to compute results
