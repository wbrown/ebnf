# Transform API Quick Start Guide

## The Simplest Example

```go
import "github.com/wbrown/ebnf/parse"

// Parse your input (you already have a ParseTree)
tree := parser.Parse(input, "start_rule")

// Define transforms - just map rule names to functions
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

// result is your transformed value
fmt.Println(result)  // e.g., 5.0
```

That's it! You're done.

## Common Patterns

### Pattern 1: Simple Value Transformation
```go
transforms := parse.TransformMap{
    "number": func(s string) int {
        i, _ := strconv.Atoi(s)
        return i
    },
    "string": func(s string) string {
        return s  // Identity transform
    },
}
```

### Pattern 2: Building Structured Objects
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
            Line:  node.Line,  // Access position
        }, nil
    },
}
```

### Pattern 3: Using Context (Parent/Siblings)
```go
transforms := parse.TransformMap{
    "list": func(ctx *parse.TransformContext, items ...interface{}) ([]interface{}, error) {
        // ctx.Node - current node
        // ctx.Parent - parent node
        // ctx.Siblings - sibling nodes
        // ctx.IsFirst(), ctx.IsLast() - position checks
        return items, nil
    },
}
```

### Pattern 4: Multi-Pass Transformation
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

## Function Signature Cheat Sheet

| Need | Signature |
|------|-----------|
| Nothing special | `func(args...) result` |
| Position info | `func(node *Node, args...) result` |
| Parent/siblings | `func(ctx *TransformContext, args...) result` |
| Both | `func(ctx *TransformContext, node *Node, args...) result` |

## Error Handling

```go
result, err := parse.Transform(tree, transforms)
if err != nil {
    // Error automatically includes:
    // - Rule name
    // - Line/column position
    // - Source snippet with highlighting
    // - Original error message
    fmt.Printf("Error: %v\n", err)
    
    // Or get detailed info:
    if te, ok := parse.AsTransformError(err); ok {
        fmt.Printf("Rule: %s\n", te.Rule)
        fmt.Printf("Position: %s\n", te.Position())
        fmt.Printf("Source:\n%s\n", te.GetSourceSnippet())
    }
}
```

## Tips

1. **Start simple**: Use old style first, add features as needed
2. **Mix styles**: You can use different styles in the same TransformMap
3. **Use closures**: For shared state across nodes
4. **Read errors**: They tell you exactly where the problem is

## Next Steps

- See `transform_test.go` for more examples
- See `transform.md` for complete guide
- See `error_api_usage.md` for error handling details

