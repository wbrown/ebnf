# Calculator Example

This is a complete example showing how to build a working calculator using the EBNF parser and tree transformation framework.

## What it demonstrates

1. **S-expression-like parse trees** - Using right recursion and strategic hiding
2. **Tree transformation** - Converting parse trees to results with simple functions
3. **Operator precedence** - Multiplication/division before addition/subtraction
4. **Negative numbers** - Unary negation support
5. **Parentheses** - Override precedence with grouping

## Running the example

```bash
# From the examples/calculator directory
go run main.go

# Show parse trees along with results
go run main.go -tree
```

## How it works

### The Grammar

The arithmetic grammar (`../arithmetic.ebnf`) uses **right recursion** and **strategic hiding** to create clean, S-expression-like parse trees:

```ebnf
(* Each operation becomes its own node *)
<add_sub> = add | sub | mul_div ;
add = mul_div <"+"> add_sub ;
sub = mul_div <"-"> add_sub ;

<mul_div> = mul | div | unary ;
mul = unary <"*"> mul_div ;
div = unary <"/"> mul_div ;

<unary> = neg | atom ;
neg = <"-"> atom ;

<atom> = number | <"("> add_sub <")"> ;
```

Hidden rules (`<add_sub>`, `<mul_div>`, etc.) are removed from the parse tree, leaving only the operation nodes.

### The Parse Tree

When parsing `2 + 3 * 4`, the grammar produces this clean tree:

```
expr
└── add
    ├── number: "2"
    └── mul
        ├── number: "3"
        └── number: "4"
```

Notice how each operation (`add`, `mul`) is its own node with exactly 2 children. This structure mirrors the mathematical expression perfectly!

Compare to `(2 + 3) * 4`:

```
expr
└── mul
    ├── add
    │   ├── number: "2"
    │   └── number: "3"
    └── number: "4"
```

### The Transformation

With the clean tree structure, evaluation is trivial - each operation is just one line:

```go
transforms := parse.TransformMap{
    "number": func(s string) float64 {
        f, _ := strconv.ParseFloat(s, 64)
        return f
    },

    // Binary operations - just plain math!
    "add": func(a, b float64) float64 { return a + b },
    "sub": func(a, b float64) float64 { return a - b },
    "mul": func(a, b float64) float64 { return a * b },
    "div": func(a, b float64) float64 { return a / b },

    // Unary negation
    "neg": func(a float64) float64 { return -a },
}

// Transform applies these functions bottom-up
result, _ := parse.Transform(tree, transforms)
```

The Transform function:
1. Recursively processes children first (bottom-up)
2. Calls the transformation function for each rule
3. Passes transformed children as arguments
4. Returns the final result

For `2 + 3 * 4`:
- `number("2")` → `2.0`
- `number("3")` → `3.0`
- `number("4")` → `4.0`
- `mul(3.0, 4.0)` → `12.0`
- `add(2.0, 12.0)` → `14.0`

### Comparison: Before vs After

**Before** (manual tree walking): ~220 lines
```go
type Evaluator struct{}

func (e *Evaluator) Eval(node *parse.Node) (float64, error) {
    switch node.Rule {
    case "expr": return e.evalExpr(node)
    case "term": return e.evalTerm(node)
    // ... 30+ lines per function
}
```

**After** (transformation): ~30 lines
```go
transforms := parse.TransformMap{
    "add": func(a, b float64) float64 { return a + b },
    "sub": func(a, b float64) float64 { return a - b },
    // ... one line per operation
}
result, _ := parse.Transform(tree, transforms)
```

## Why S-expression-like trees?

Traditional recursive descent grammars create flat trees like:
```
expr: [term, "+", term, "*", term]  // Hard to transform
```

S-expression-like trees create hierarchical nodes:
```
add(term, mul(term, term))  // Easy to transform
```

The key is using **right recursion** (works with recursive descent) and **hiding wrapper rules** to promote operations to top-level nodes.

## Note: Right Associativity

Right recursion makes operations right-associative:
- `10 - 5 - 2` evaluates as `10 - (5 - 2) = 7`, not `(10 - 5) - 2 = 3`

This is a known trade-off for using recursive descent without left-recursion support. For most math (addition, multiplication) it doesn't matter since they're associative.

To support left-associativity would require a **GLL parser** that can handle left recursion.

## Extending this example

You could extend this calculator to support:

- **Variables**: Add a symbol table and a `var` transformation
- **Functions**: Add `sin`, `cos`, `sqrt` nodes with corresponding transforms
- **More operators**: Add `pow`, `mod` rules and one-line transforms
- **Different types**: Return `interface{}` instead of `float64` to support strings, bools, etc.

The pattern is always:
1. Add the rule to the grammar (using right recursion)
2. Add a one-line transform function
3. Done!
