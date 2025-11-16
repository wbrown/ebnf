# Transform System Guide

The transform system allows you to elegantly process parse trees by mapping rule names to transformation functions. This guide covers everything you need to know.

## Quick Start

```go
import "github.com/wbrown/ebnf/parse"

// Parse your input
tree, err := parser.Parse(input, "start_rule")

// Define transforms - map rule names to functions
transforms := parse.TransformMap{
    "number": func(s string) float64 {
        f, _ := strconv.ParseFloat(s, 64)
        return f
    },
    "add": func(a, b float64) float64 {
        return a + b
    },
}

// Transform!
result, err := parse.Transform(tree, transforms)
if err != nil {
    log.Fatal(err)  // Error includes position + source snippet!
}
```

See [transform_quick_start.md](./transform_quick_start.md) for more quick examples.

## Table of Contents

1. [Basic Usage](#basic-usage)
2. [Function Signatures](#function-signatures)
3. [Context-Aware Transforms](#context-aware-transforms)
4. [Multi-Pass Transforms](#multi-pass-transforms)
5. [Top-Down Transforms](#top-down-transforms)
6. [Error Handling](#error-handling)
7. [Common Patterns](#common-patterns)
8. [Examples](#examples)

## Basic Usage

### Simple Transform

The simplest transform just maps rule names to functions:

```go
transforms := parse.TransformMap{
    "number": func(s string) int {
        i, _ := strconv.Atoi(s)
        return i
    },
    "add": func(a, b int) int {
        return a + b
    },
}

result, err := parse.Transform(tree, transforms)
```

### How It Works

1. **Bottom-up**: Transforms are applied recursively from leaves to root
2. **Rule-based**: Each rule in `TransformMap` gets its function called on matching nodes
3. **Type-flexible**: Functions can have any signature; automatic type conversion is attempted
4. **Pass-through**: Rules without transforms pass their children up automatically

## Function Signatures

The transform system supports four function signature styles, automatically detected:

### Style 1: Simple (Old Style)
```go
"number": func(s string) float64 {
    f, _ := strconv.ParseFloat(s, 64)
    return f
}
```

### Style 2: Node-Aware
Access source position information:
```go
"number": func(node *parse.Node, s string) (*Number, error) {
    val, _ := strconv.Atoi(s)
    return &Number{
        Value: val,
        Line:  node.Line,   // Source position
        Start: node.Start,  // Character offset
    }, nil
}
```

### Style 3: Context-Aware
Access parent, siblings, and tree context:
```go
"item": func(ctx *parse.TransformContext, text string) (*Item, error) {
    // ctx.Node - current node
    // ctx.Parent - parent node
    // ctx.Siblings - sibling nodes
    // ctx.Tree - full parse tree
    // ctx.IsFirst(), ctx.IsLast() - position checks
    return &Item{Text: text}, nil
}
```

### Style 4: Combined
Both context and node:
```go
"number": func(ctx *parse.TransformContext, node *parse.Node, s string) (*Number, error) {
    // Both context and node available
    return &Number{Value: val, Line: node.Line}, nil
}
```

**Key Points:**
- You can mix all styles in the same `TransformMap`
- The system automatically detects which style you're using
- Error messages show which signature was detected

## Context-Aware Transforms

The `TransformContext` provides rich information about the current transformation:

```go
type TransformContext struct {
    Tree     *ParseTree  // Full parse tree
    Node     *Node       // Current node being transformed
    Parent   *Node       // Parent node (nil for root)
    Siblings []*Node     // Sibling nodes
    Index    int         // Index in parent's children
    Input    string      // Original input text
    State    map[string]interface{}  // Per-node state storage
}
```

### Helper Methods

```go
ctx.IsFirst()        // Is this the first sibling?
ctx.IsLast()         // Is this the last sibling?
ctx.NextSibling()     // Get next sibling (or nil)
ctx.PrevSibling()     // Get previous sibling (or nil)
ctx.SiblingCount()   // Total number of siblings
```

### Example: Grouping Related Nodes

```go
transforms := parse.TransformMap{
    "if_chain": func(ctx *parse.TransformContext, branches ...interface{}) (*Conditional, error) {
        // Access siblings to group if/elseif/else
        if ctx.Parent != nil {
            // Check parent context
        }
        return &Conditional{Branches: branches}, nil
    },
}
```

### Sharing State Across Nodes

**Important:** `TransformContext.State` is per-node and not shared. For cross-node communication (e.g., symbol tables, parent metadata, scoping), use closures:

```go
// Create shared state outside transforms
var symbolTable = make(map[string]interface{})
var currentScope *Scope

transforms := parse.TransformMap{
    "function": func(ctx *parse.TransformContext, name string, body interface{}) (*Function, error) {
        // Store in shared state for children to access
        symbolTable[name] = &Function{Name: name}
        return symbolTable[name].(*Function), nil
    },
    "variable": func(ctx *parse.TransformContext, name string) (*Variable, error) {
        // Access parent's data from shared state
        if fn, exists := symbolTable[name]; exists {
            return &Variable{Name: name, Ref: fn}, nil
        }
        return nil, fmt.Errorf("undefined: %s", name)
    },
}
```

**Why closures are needed:**
- **Bottom-up transforms**: Parent is transformed *after* children (parent state not available)
- **Top-down transforms**: Parent state is in its own `Context.State` (not accessible to children)
- **Multi-pass**: State doesn't persist across passes

Closures provide a simple, flexible way to share state across the transform tree without architectural changes.

## Multi-Pass Transforms

Some transformations require multiple passes over the tree. Use `TransformMultiPass`:

```go
// Pass 1: Group related nodes
pass1 := parse.TransformMap{
    "if_chain": func(ctx *parse.TransformContext, branches ...interface{}) (*Conditional, error) {
        return &Conditional{Branches: branches}, nil
    },
}

// Pass 2: Transform values
pass2 := parse.TransformMap{
    "number": func(s string) int {
        i, _ := strconv.Atoi(s)
        return i
    },
}

result, err := parse.TransformMultiPass(tree, []parse.TransformMap{pass1, pass2})
```

**Key Features:**
- Structure is preserved between passes
- Types are preserved via `TransformedValue`
- Each pass can build on previous results

## Top-Down Transforms

For scoping and context setup, use `TransformTopDown`:

```go
transforms := parse.TransformMap{
    "scope": func(ctx *parse.TransformContext, children ...*parse.Node) (*Scope, error) {
        // Process parent before children
        scope := &Scope{Parent: ctx.Parent}
        // Children will be processed after this returns
        return scope, nil
    },
}

result, err := parse.TransformTopDown(tree, transforms)
```

**Use Cases:**
- Symbol table building
- Scoping analysis
- Context setup before child processing

## Error Handling

Transform errors include rich context information:

```go
result, err := parse.Transform(tree, transforms)
if err != nil {
    // Basic error message
    fmt.Printf("Error: %v\n", err)
    
    // Get detailed error info
    if te, ok := parse.AsTransformError(err); ok {
        fmt.Printf("Rule: %s\n", te.Rule)
        fmt.Printf("Position: %s\n", te.Position())  // "line 5, column 10"
        fmt.Printf("Source snippet:\n%s\n", te.GetSourceSnippet())
        fmt.Printf("Pass: %d\n", te.PassNumber)
    }
}
```

**Error Information Includes:**
- Rule name where error occurred
- Line and column position
- Source code snippet with highlighting
- Pass number (for multi-pass)
- Context path (rule chain)
- Original error message

**Panic Recovery:**
Panics in transform functions are automatically caught and converted to `TransformError` with stack traces.

## Common Patterns

### Pattern 1: Building Structured Objects

```go
type Number struct {
    Value int
    Line  int
}

transforms := parse.TransformMap{
    "number": func(node *parse.Node, s string) (*Number, error) {
        val, _ := strconv.Atoi(s)
        return &Number{
            Value: val,
            Line:  node.Line,
        }, nil
    },
}
```

### Pattern 2: Using Shared State (Closures)

```go
var sharedState = make(map[string]interface{})

transforms := parse.TransformMap{
    "parent": func(ctx *parse.TransformContext, children ...interface{}) interface{} {
        sharedState["parent_id"] = ctx.Node.Rule
        return children
    },
    "child": func(ctx *parse.TransformContext, text string) interface{} {
        parentID := sharedState["parent_id"]
        return &Child{Text: text, Parent: parentID}
    },
}
```

### Pattern 3: Conditional Processing

```go
transforms := parse.TransformMap{
    "item": func(ctx *parse.TransformContext, text string) (*Item, error) {
        item := &Item{Text: text}
        
        if ctx.IsFirst() {
            item.IsFirst = true
        }
        if ctx.IsLast() {
            item.IsLast = true
        }
        
        return item, nil
    },
}
```

### Pattern 4: Metadata Attachment

```go
transforms := parse.TransformMap{
    "number": func(node *parse.Node, s string) *parse.TransformResult {
        val, _ := strconv.Atoi(s)
        return &parse.TransformResult{
            Value: val,
            Metadata: map[string]interface{}{
                "source_pos": node.Start,
                "line": node.Line,
            },
        }
    },
}
```

## Examples

### Complete Example: Markdown to HTML

See `examples/markdown/` for a complete Markdown-to-HTML transformer demonstrating:
- Multi-pass transforms
- Context-aware processing
- Error handling
- Type preservation

### Example: Calculator

```go
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

tree, _ := parser.Parse("(2 + 3) * 4", "expr")
result, _ := parse.Transform(tree, transforms)
fmt.Printf("Result: %.2f\n", result)  // Result: 20.00
```

## API Reference

### Main Functions

- `Transform(tree *ParseTree, transforms TransformMap) (interface{}, error)`
  - Basic bottom-up transformation

- `TransformNode(node *Node, transforms TransformMap) (interface{}, error)`
  - Transform a single node and its descendants

- `TransformPreserveStructure(tree *ParseTree, transforms TransformMap) (interface{}, error)`
  - Transform while preserving tree structure (for multi-pass)

- `TransformMultiPass(tree *ParseTree, passes []TransformMap) (interface{}, error)`
  - Apply multiple transformation passes sequentially

- `TransformTopDown(tree *ParseTree, transforms TransformMap) (interface{}, error)`
  - Transform parent before children (top-down)

### Error Helpers

- `AsTransformError(err error) (*TransformError, bool)`
  - Extract `TransformError` from error

- `IsTransformError(err error) bool`
  - Check if error is a `TransformError`

- `GetTransformError(err error) *TransformError`
  - Get `TransformError` (panics if not)

### Context Helpers

- `ctx.IsFirst() bool`
- `ctx.IsLast() bool`
- `ctx.NextSibling() *Node`
- `ctx.PrevSibling() *Node`
- `ctx.SiblingCount() int`

## Tips

1. **Start simple**: Use old style first, add features as needed
2. **Mix styles**: You can use different styles in the same `TransformMap`
3. **Use closures**: For shared state across nodes
4. **Read errors**: They tell you exactly where the problem is
5. **Test incrementally**: Add one transform at a time

## See Also

- [transform_quick_start.md](./transform_quick_start.md) - Quick reference
- [examples/markdown/](../../examples/markdown/) - Complete Markdown example
- [transform_test.go](./transform_test.go) - Test examples
- [transform_node_aware_test.go](./transform_node_aware_test.go) - Node-aware examples
- [transform_context_test.go](./transform_context_test.go) - Context examples

