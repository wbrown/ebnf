package main

import (
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wbrown/ebnf"
	"github.com/wbrown/ebnf/parse"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <markdown-file>\n", os.Args[0])
		os.Exit(1)
	}

	// Load Markdown grammar
	grammarPath := filepath.Join(filepath.Dir(os.Args[0]), "markdown.ebnf")
	// Try current directory first
	if _, err := os.Stat("markdown.ebnf"); err != nil {
		grammarPath = "markdown.ebnf"
	}
	grammar, err := ebnf.LoadGrammar(grammarPath)
	if err != nil {
		log.Fatalf("Failed to load grammar: %v", err)
	}

	// Read Markdown input
	input, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Parse Markdown
	parser := parse.New(grammar)
	tree, err := parser.Parse(string(input), "document")
	if err != nil {
		log.Fatalf("Parse failed: %v", err)
	}

	// Transform to HTML using multi-pass approach
	html, err := MarkdownToHTML(tree)
	if err != nil {
		log.Fatalf("Transform failed: %v", err)
	}

	fmt.Print(html)
}

// MarkdownToHTML transforms a Markdown parse tree to HTML
func MarkdownToHTML(tree *parse.ParseTree) (string, error) {
	// Pass 1: Transform inline elements (bold, italic, code, links)
	pass1 := parse.TransformMap{
		"text": func(v interface{}) string {
			return html.EscapeString(extractString(v))
		},
		"space": func(v interface{}) string {
			return extractString(v) // Preserve spaces
		},
		"link_text": func(v interface{}) string {
			return extractString(v) // Pass through
		},
		"link_url": func(v interface{}) string {
			return extractString(v) // Pass through
		},
		"language": func(v interface{}) string {
			return extractString(v) // Pass through
		},
		"code_content": func(v interface{}) string {
			return extractString(v) // Pass through
		},
		"bold": func(ctx *parse.TransformContext, text interface{}) string {
			textStr := extractString(text)
			return fmt.Sprintf("<strong>%s</strong>", html.EscapeString(textStr))
		},
		"italic": func(ctx *parse.TransformContext, text interface{}) string {
			textStr := extractString(text)
			return fmt.Sprintf("<em>%s</em>", html.EscapeString(textStr))
		},
		"code": func(ctx *parse.TransformContext, text interface{}) string {
			textStr := extractString(text)
			return fmt.Sprintf("<code>%s</code>", html.EscapeString(textStr))
		},
		"link": func(ctx *parse.TransformContext, text, url interface{}) string {
			textStr := extractString(text)
			urlStr := extractString(url)
			// Validate URL (simple check)
			if urlStr == "" {
				// Error would include position info automatically!
				return fmt.Sprintf(`<a href="#">%s</a>`, html.EscapeString(textStr))
			}
			return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(urlStr), html.EscapeString(textStr))
		},
		"inline": func(ctx *parse.TransformContext, content interface{}) string {
			// Unwrap inline content (could be text, bold, italic, etc.)
			return extractString(content)
		},
		"paragraph": func(ctx *parse.TransformContext, inlines ...interface{}) string {
			// Combine inline elements (may be strings or already transformed)
			var parts []string
			for _, inline := range inlines {
				parts = append(parts, extractString(inline))
			}
			content := strings.Join(parts, "")
			// Trim leading/trailing whitespace
			content = strings.TrimSpace(content)
			if content == "" {
				return ""
			}
			return fmt.Sprintf("<p>%s</p>\n", content)
		},
	}

	// Pass 2: Transform block elements (headings, lists, code blocks)
	pass2 := parse.TransformMap{
		"heading_level": func(v interface{}) string {
			return extractString(v) // Pass through
		},
		"heading": func(ctx *parse.TransformContext, level interface{}, textParts ...interface{}) string {
			// level is the heading_level string, textParts are text/space elements
			levelStr := extractString(level)
			var textPartsStr []string
			for _, part := range textParts {
				textPartsStr = append(textPartsStr, extractString(part))
			}
			textStr := strings.Join(textPartsStr, "")
			// Count # to determine level
			levelNum := strings.Count(levelStr, "#")
			if levelNum < 1 || levelNum > 6 {
				levelNum = 1
			}
			return fmt.Sprintf("<h%d>%s</h%d>\n", levelNum, strings.TrimSpace(textStr), levelNum)
		},
		"list": func(ctx *parse.TransformContext, items ...interface{}) string {
			html := "<ul>\n"
			for _, item := range items {
				itemStr := extractString(item)
				html += fmt.Sprintf("  <li>%s</li>\n", strings.TrimSpace(itemStr))
			}
			html += "</ul>\n"
			return html
		},
		"list_item": func(ctx *parse.TransformContext, inlines ...interface{}) string {
			// Inlines already transformed in pass 1
			var parts []string
			for _, inline := range inlines {
				parts = append(parts, extractString(inline))
			}
			return strings.Join(parts, "")
		},
		"code_block": func(ctx *parse.TransformContext, args ...interface{}) string {
			// Handle optional language parameter
			var langStr, codeStr string
			if len(args) == 2 {
				langStr = extractString(args[0])
				codeStr = extractString(args[1])
			} else if len(args) == 1 {
				codeStr = extractString(args[0])
			}
			langAttr := ""
			if langStr != "" {
				langAttr = fmt.Sprintf(` class="language-%s"`, html.EscapeString(langStr))
			}
			return fmt.Sprintf("<pre><code%s>%s</code></pre>\n", langAttr, html.EscapeString(codeStr))
		},
		"horizontal_rule": func(ctx *parse.TransformContext) string {
			return "<hr>\n"
		},
		"block": func(ctx *parse.TransformContext, content interface{}) string {
			// Unwrap block content
			return extractString(content)
		},
		"document": func(ctx *parse.TransformContext, blocks ...interface{}) string {
			var parts []string
			for _, block := range blocks {
				blockStr := extractString(block)
				if blockStr != "" {
					parts = append(parts, blockStr)
				}
			}
			return strings.Join(parts, "")
		},
		// Handle synthetic nodes created by TransformPreserveStructure
		"_transformed": func(val interface{}) string {
			return extractString(val)
		},
	}

	// Apply multi-pass transformation
	result, err := parse.TransformMultiPass(tree, []parse.TransformMap{pass1, pass2})
	if err != nil {
		return "", err
	}

	html, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("expected string result, got %T", result)
	}

	return html, nil
}

// extractString extracts a string value from various types.
// This handles the fact that multi-pass transforms can return either
// strings (from previous passes) or *Node objects (from TransformPreserveStructure).
func extractString(v interface{}) string {
	if v == nil {
		return ""
	}
	// Direct string value
	if s, ok := v.(string); ok {
		return s
	}
	// Node from parse tree (terminal node with Value)
	if node, ok := v.(*parse.Node); ok {
		// Prefer TransformedValue (from previous pass) over Value (from parse)
		if node.TransformedValue != nil {
			if s, ok := node.TransformedValue.(string); ok {
				return s
			}
			return fmt.Sprintf("%v", node.TransformedValue)
		}
		if node.Value != "" {
			return node.Value
		}
	}
	// Fallback: convert to string
	return fmt.Sprintf("%v", v)
}
