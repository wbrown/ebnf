package parse

import (
	"fmt"
)

// TransformMultiPass applies multiple transform passes sequentially.
// Each pass transforms the result of the previous pass.
//
// If a pass returns a *ParseTree, it is used as input for the next pass.
// If a pass returns a value (not a tree), it is wrapped in a synthetic tree
// for the next pass, or the transformation stops if it's the last pass.
//
// IMPORTANT: TransformPreserveStructure is used for intermediate passes to
// ensure tree structure is maintained between passes. This allows nodes without
// explicit transforms to persist across passes.
//
// Example:
//
//	pass1 := TransformMap{
//	    "if_chain": func(ctx *TransformContext, ...) (*Conditional, error) {
//	        // Group if/elseif/else chains
//	    },
//	}
//	pass2 := TransformMap{
//	    "number": func(node *Node, s string) float64 { ... },
//	    "add": func(a, b float64) float64 { return a + b },
//	}
//	result, err := TransformMultiPass(tree, []TransformMap{pass1, pass2})
func TransformMultiPass(tree *ParseTree, passes []TransformMap) (interface{}, error) {
	if tree == nil || tree.Root == nil {
		return nil, fmt.Errorf("cannot transform nil tree")
	}
	if len(passes) == 0 {
		return tree, nil
	}

	current := tree
	var lastResult interface{}

	for i, pass := range passes {
		// Transform current tree with this pass, preserving structure
		// This ensures nodes without transforms are preserved for the next pass
		result, err := transformPreserveStructureWithPass(current, pass, i+1)
		if err != nil {
			// Error already wrapped with pass number
			return nil, fmt.Errorf("pass %d failed: %w", i+1, err)
		}

		// Check if result is a tree (for next pass)
		if resultTree, ok := result.(*ParseTree); ok {
			current = resultTree
			lastResult = resultTree
		} else if resultNode, ok := result.(*Node); ok {
			// Wrap node in a tree for next pass
			current = &ParseTree{Root: resultNode, Input: current.Input}
			lastResult = resultNode
		} else {
			// Value result - wrap in synthetic tree if more passes remain
			if i < len(passes)-1 {
				current = valueToTree(result, current.Input)
				lastResult = result
			} else {
				// Last pass - return the value directly
				lastResult = result
			}
		}
	}

	// Unwrap final result if it's a synthetic _transformed node
	if node, ok := lastResult.(*Node); ok {
		if node.Rule == "_transformed" {
			// Prefer TransformedValue if available (preserves type)
			if node.TransformedValue != nil {
				return node.TransformedValue, nil
			}
			// Fall back to Value (string)
			return node.Value, nil
		}
		// If node has a single _transformed child, unwrap it
		if len(node.Children) == 1 && node.Children[0].Rule == "_transformed" {
			child := node.Children[0]
			if child.TransformedValue != nil {
				return child.TransformedValue, nil
			}
			return child.Value, nil
		}
	}

	return lastResult, nil
}

// valueToTree wraps a transformed value in a synthetic ParseTree for the next pass
func valueToTree(value interface{}, input string) *ParseTree {
	// Create a synthetic node to hold the value
	node := &Node{
		Rule:             "_transformed",
		TransformedValue: value,                    // Store the actual typed value
		Value:            fmt.Sprintf("%v", value), // String representation
	}

	return &ParseTree{
		Root:  node,
		Input: input,
	}
}
