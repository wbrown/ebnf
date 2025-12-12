package main

import (
	"strings"
	"testing"

	"github.com/wbrown/ebnf"
	"github.com/wbrown/ebnf/parse"
)

func TestMarkdownToHTML_Heading(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "# Heading 1"
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<h1>") {
		t.Errorf("Expected <h1> tag, got: %s", html)
	}
	if !strings.Contains(html, "Heading") {
		t.Errorf("Expected 'Heading' text, got: %s", html)
	}
}

func TestMarkdownToHTML_Paragraph(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "This is a paragraph."
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<p>") {
		t.Errorf("Expected <p> tag, got: %s", html)
	}
	if !strings.Contains(html, "paragraph") {
		t.Errorf("Expected paragraph text, got: %s", html)
	}
}

func TestMarkdownToHTML_Bold(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "This is **bold** text."
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<strong>") {
		t.Errorf("Expected <strong> tag, got: %s", html)
	}
	if !strings.Contains(html, "bold") {
		t.Errorf("Expected 'bold' text, got: %s", html)
	}
}

func TestMarkdownToHTML_Italic(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "This is _italic_ text."
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<em>") {
		t.Errorf("Expected <em> tag, got: %s", html)
	}
}

func TestMarkdownToHTML_Code(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "This is `code` text."
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<code>") {
		t.Errorf("Expected <code> tag, got: %s", html)
	}
}

func TestMarkdownToHTML_Link(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "This is a [link](https://example.com) text."
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<a") {
		t.Errorf("Expected <a> tag, got: %s", html)
	}
	if !strings.Contains(html, "href=") {
		t.Errorf("Expected href attribute, got: %s", html)
	}
	if !strings.Contains(html, "https://example.com") {
		t.Errorf("Expected URL, got: %s", html)
	}
}

func TestMarkdownToHTML_List(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "- Item 1\n- Item 2\n- Item 3"
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<ul>") {
		t.Errorf("Expected <ul> tag, got: %s", html)
	}
	if !strings.Contains(html, "<li>") {
		t.Errorf("Expected <li> tag, got: %s", html)
	}
}

func TestMarkdownToHTML_CodeBlock(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := "```go\nfunc main() {}\n```"
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if !strings.Contains(html, "<pre>") {
		t.Errorf("Expected <pre> tag, got: %s", html)
	}
	// Code block should have code tag (may be inside pre)
	if !strings.Contains(html, "code") {
		t.Errorf("Expected code tag, got: %s", html)
	}
}

func TestMarkdownToHTML_Complex(t *testing.T) {
	grammar, err := ebnf.LoadGrammar("markdown.ebnf")
	if err != nil {
		t.Fatalf("Failed to load grammar: %v", err)
	}

	input := `# Heading

This is a paragraph with **bold** and _italic_ text.

- List item 1
- List item 2

[Link](https://example.com)
`
	parser := parse.New(grammar)
	tree, err := parser.Parse(input, "document")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	html, err := MarkdownToHTML(tree)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Check for multiple elements
	checks := []string{"<h1>", "<p>", "<strong>", "<em>", "<ul>", "<li>", "<a"}
	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("Expected %s tag, got: %s", check, html)
		}
	}
}
