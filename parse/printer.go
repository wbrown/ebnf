package parse

import (
	"fmt"
	"io"
	"strings"
)

// PrintAST prints the parse tree in a human-readable format
func PrintAST(w io.Writer, tree *ParseTree) {
	if tree == nil || tree.Root == nil {
		fmt.Fprintln(w, "(empty tree)")
		return
	}
	printNodeFixed(w, tree.Root, "", "")
}

// PrintNode prints a single node and its subtree
func PrintNode(w io.Writer, node *Node) {
	if node == nil {
		fmt.Fprintln(w, "(nil)")
		return
	}
	printNodeFixed(w, node, "", "")
}

func printNodeFixed(w io.Writer, node *Node, prefix string, childrenPrefix string) {
	if node == nil {
		return
	}

	// Print current line prefix + node
	fmt.Fprintf(w, "%s", prefix)

	// Print node info
	if node.Value != "" && node.Rule != "" {
		// Terminal node with rule and value
		fmt.Fprintf(w, "%s: %q", node.Rule, node.Value)
	} else if node.Value != "" {
		// Terminal node with just value (e.g., from regex)
		fmt.Fprintf(w, "%q", node.Value)
	} else {
		// Non-terminal
		fmt.Fprintf(w, "%s", node.Rule)
	}

	// Add position info if available
	if node.Line > 0 {
		fmt.Fprintf(w, " [line %d]", node.Line)
	}
	fmt.Fprintln(w)

	// Print children
	for i, child := range node.Children {
		isLast := i == len(node.Children)-1

		if isLast {
			printNodeFixed(w, child, childrenPrefix+"└── ", childrenPrefix+"    ")
		} else {
			printNodeFixed(w, child, childrenPrefix+"├── ", childrenPrefix+"│   ")
		}
	}
}

func printNodeWithPrefix(w io.Writer, node *Node, prefix string, isLast bool) {
	if node == nil {
		return
	}

	// Print the tree branch
	fmt.Fprintf(w, "%s", prefix)

	// Print the connector, unless this is the root
	if prefix != "" {
		if isLast {
			fmt.Fprintf(w, "└── ")
		} else {
			fmt.Fprintf(w, "├── ")
		}
	}

	// Print node info
	if node.Value != "" && node.Rule != "" {
		// Terminal node with rule and value
		fmt.Fprintf(w, "%s: %q", node.Rule, node.Value)
	} else if node.Value != "" {
		// Terminal node with just value (e.g., from regex)
		fmt.Fprintf(w, "%q", node.Value)
	} else if len(node.Children) == 0 {
		// Non-terminal with no children
		fmt.Fprintf(w, "%s", node.Rule)
	} else {
		// Non-terminal with children
		fmt.Fprintf(w, "%s", node.Rule)
	}

	// Add position info if available
	if node.Line > 0 {
		fmt.Fprintf(w, " [line %d]", node.Line)
	}
	fmt.Fprintln(w)

	// Calculate the extension for the prefix
	extension := ""
	if prefix == "" {
		// First level children - they need indentation
		extension = ""
	} else if isLast {
		extension = "    " // spaces under a └──
	} else {
		extension = "│   " // vertical line under a ├──
	}

	// Print children with extended prefix
	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		printNodeWithPrefix(w, child, prefix+extension, isLastChild)
	}
}

// Compatibility wrapper
func printNode(w io.Writer, node *Node, depth int, isLast bool) {
	// Build prefix from depth
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "│   "
	}
	if depth > 0 {
		// Remove the last "│   " since printNodeWithPrefix will add the connector
		prefix = prefix[:len(prefix)-4]
	}
	printNodeWithPrefix(w, node, prefix, isLast)
}

// CompactPrintAST prints the parse tree in a more compact format
func CompactPrintAST(w io.Writer, tree *ParseTree) {
	if tree == nil || tree.Root == nil {
		fmt.Fprintln(w, "(empty tree)")
		return
	}
	compactPrintNode(w, tree.Root, 0)
}

func compactPrintNode(w io.Writer, node *Node, depth int) {
	if node == nil {
		return
	}

	indent := strings.Repeat("  ", depth)

	// Skip certain nodes for compactness
	skipRules := map[string]bool{
		"ws":           true,
		"newline":      true,
		"null":         true,
		"space_indent": true,
		"tab_indent":   true,
	}

	if skipRules[node.Rule] {
		return
	}

	// Collapse single-child chains
	if len(node.Children) == 1 && node.Value == "" {
		child := node.Children[0]
		// Check if we should collapse
		if shouldCollapse(node.Rule, child.Rule) {
			compactPrintNode(w, child, depth)
			return
		}
	}

	// Print node
	if node.Value != "" {
		fmt.Fprintf(w, "%s%s: %q\n", indent, node.Rule, node.Value)
	} else {
		fmt.Fprintf(w, "%s%s\n", indent, node.Rule)
		for _, child := range node.Children {
			compactPrintNode(w, child, depth+1)
		}
	}
}

func shouldCollapse(parent, child string) bool {
	// Define rules for which parent-child combinations should be collapsed
	collapseRules := map[string]map[string]bool{
		"scene_element": {"line": true},
		"content":       {"command": true, "text_line": true},
		"text_element":  {"text_chunk": true, "var_sub": true, "expr_sub": true},
		"var_name_tail": {"var_name_char": false}, // Don't collapse these
		"text_chunk":    {"text_char": false},     // Don't collapse individual characters
	}

	if childRules, ok := collapseRules[parent]; ok {
		if shouldCollapse, found := childRules[child]; found {
			return shouldCollapse
		}
	}

	// Default: collapse single children
	return true
}

// GetNodeSummary returns a one-line summary of a node
func GetNodeSummary(node *Node) string {
	if node == nil {
		return "(nil)"
	}

	if node.Value != "" {
		return fmt.Sprintf("%s: %q", node.Rule, node.Value)
	}

	// For certain nodes, show more detail
	switch node.Rule {
	case "command":
		if len(node.Children) > 0 {
			return fmt.Sprintf("command: %s", node.Children[0].Rule)
		}
	case "text":
		return fmt.Sprintf("text (%d elements)", len(node.Children))
	case "scene":
		return fmt.Sprintf("scene (%d elements)", len(node.Children))
	}

	if len(node.Children) == 0 {
		return node.Rule
	}
	return fmt.Sprintf("%s (%d children)", node.Rule, len(node.Children))
}
