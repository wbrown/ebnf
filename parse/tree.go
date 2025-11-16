package parse

// Node represents a node in the parse tree
type Node struct {
	Rule            string      // The grammar rule that produced this node
	Value           string      // For terminal nodes, the actual text (string representation)
	TransformedValue interface{} // Optional: Stores the actual typed transformed value (preserves type information)
	Children        []*Node     // For non-terminal nodes, the child nodes
	Line            int         // Line number in source
	Column          int         // Column number in source
	Start           int         // Start position in input string
	End             int         // End position in input string
}

// IsTerminal returns true if this is a terminal node (has a value but no children)
func (n *Node) IsTerminal() bool {
	return n.Value != "" && len(n.Children) == 0
}

// ParseTree represents the root of a parsed document
type ParseTree struct {
	Root  *Node
	Input string // The original input text
}
