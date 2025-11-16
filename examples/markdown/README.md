# Markdown to HTML Transformer

This example demonstrates how to use the `ebnf` library's multi-pass transform system to convert Markdown syntax to HTML.

## Overview

The transformer uses a two-pass approach:

1. **Pass 1**: Transform inline elements (bold, italic, code, links)
2. **Pass 2**: Transform block elements (headings, paragraphs, lists, code blocks)

This separation allows inline elements to be processed first and then embedded within block elements.

## Features

The transformer supports:

- **Headings**: `# Heading 1` through `###### Heading 6`
- **Paragraphs**: Plain text blocks
- **Bold text**: `**bold**`
- **Italic text**: `_italic_`
- **Inline code**: `` `code` ``
- **Links**: `[text](url)`
- **Lists**: `- Item 1`
- **Code blocks**: ` ```language\ncode\n``` `

## Usage

### Command Line

```bash
go run markdown_to_html.go example.md
```

### Programmatic

```go
import (
    "github.com/wbrown/ebnf"
    "github.com/wbrown/ebnf/parse"
)

// Load grammar
grammar, err := ebnf.LoadGrammar("markdown.ebnf")
if err != nil {
    log.Fatal(err)
}

// Parse input
parser := parse.New(grammar)
tree, err := parser.Parse(markdownText, "document")
if err != nil {
    log.Fatal(err)
}

// Transform to HTML
html, err := MarkdownToHTML(tree)
if err != nil {
    log.Fatal(err)
}

fmt.Println(html)
```

## Architecture

### Grammar (`markdown.ebnf`)

The grammar defines the structure of Markdown documents using EBNF notation. Hidden tokens (like `<"**">`) are used for structural elements that shouldn't appear in the transformed output.

### Transformation (`markdown_to_html.go`)

The transformation uses `TransformMultiPass` to apply two sequential transform passes:

- **Pass 1** (`inlineTransforms`): Processes inline elements and returns HTML strings
- **Pass 2** (`blockTransforms`): Processes block elements, embedding the already-transformed inline content

The `extractString` helper function correctly handles values that may be raw strings or `*parse.Node` objects (due to `TransformPreserveStructure`).

## Example

Input Markdown:
```markdown
# Hello World

This is a **bold** paragraph with a [link](https://example.com).

- Item 1
- Item 2
```

Output HTML:
```html
<h1>Hello World</h1>
<p>This is a <strong>bold</strong> paragraph with a <a href="https://example.com">link</a>.</p>
<ul>
<li>Item 1</li>
<li>Item 2</li>
</ul>
```

## Testing

Run the tests:

```bash
go test -v
```

The test suite covers:
- Individual element types (headings, paragraphs, bold, italic, code, links, lists, code blocks)
- Complex documents with multiple element types
- Edge cases and formatting variations

## Limitations

This is a simplified Markdown parser for demonstration purposes. It does not support:

- Nested lists
- Ordered lists (numbered)
- Blockquotes
- Tables
- Images
- Reference-style links
- HTML entities
- Escaping

For production use, consider a full Markdown parser like [goldmark](https://github.com/yuin/goldmark) or [blackfriday](https://github.com/russross/blackfriday).

## Purpose

This example serves as a reference implementation demonstrating:

1. **Multi-pass transforms**: How to structure complex transformations across multiple passes
2. **Type preservation**: How `TransformedValue` maintains type information between passes
3. **Structure preservation**: How `TransformPreserveStructure` maintains tree structure
4. **Context-aware transforms**: How to use `TransformContext` for position and parent information
5. **Real-world usage**: A practical, non-trivial example of the transform system

