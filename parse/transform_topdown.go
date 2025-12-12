package parse

import (
	"fmt"
	"reflect"
)

// TransformTopDown applies transformations top-down (parent before children).
// This is useful for scoping, symbol table building, and context-aware transformations.
//
// Unlike Transform (which is bottom-up), TransformTopDown processes the parent node
// before processing its children. This allows transforms to:
// - Set up scopes or symbol tables before processing children
// - Inspect parent structure before transforming children
// - Modify tree structure before children are processed
//
// Transform functions receive children as *Node (not transformed values) in top-down mode,
// allowing them to inspect the original structure before transformation.
//
// Example:
//
//	transforms := TransformMap{
//	    "scope": func(ctx *TransformContext, children ...*Node) (*Scope, error) {
//	        // Process parent scope before children
//	        scope := &Scope{Name: ctx.Node.Rule}
//	        // Children will be transformed after this
//	        return scope, nil
//	    },
//	}
//	result, err := TransformTopDown(tree, transforms)
func TransformTopDown(tree *ParseTree, transforms TransformMap) (interface{}, error) {
	if tree == nil || tree.Root == nil {
		return nil, fmt.Errorf("cannot transform nil tree")
	}
	return transformNodeTopDown(tree.Root, nil, nil, -1, tree, transforms)
}

// transformNodeTopDown recursively transforms a node top-down (parent before children).
// This allows parent transforms to set up context before children are processed.
func transformNodeTopDown(node *Node, parent *Node, siblings []*Node, index int, tree *ParseTree, transforms TransformMap) (interface{}, error) {
	// Create context
	ctx := &TransformContext{
		Tree:     tree,
		Node:     node,
		Parent:   parent,
		Siblings: siblings,
		Index:    index,
		Input:    tree.Input,
		State:    make(map[string]interface{}),
	}

	// Base case: terminal node (leaf)
	if node.IsTerminal() {
		if fn, ok := transforms[node.Rule]; ok {
			// Apply transform to terminal node
			result, err := callTransformWithPass(fn, ctx, node, nil, 0)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		// No transform - return value
		if node.TransformedValue != nil {
			return node.TransformedValue, nil
		}
		return node.Value, nil
	}

	// Non-terminal node: process parent first
	if fn, ok := transforms[node.Rule]; ok {
		// Check function signature to see if it expects transformed children
		fnType := reflect.TypeOf(fn)
		numIn := fnType.NumIn()

		// Determine if function expects transformed values or raw nodes
		expectsTransformedValues := false
		if numIn > 0 {
			firstParamType := fnType.In(0)
			// If first param is TransformContext, check second param
			if firstParamType == reflect.TypeOf((*TransformContext)(nil)) {
				if fnType.IsVariadic() && numIn > 1 {
					// Check variadic element type
					variadicType := fnType.In(numIn - 1).Elem()
					// If variadic element is *Node, it expects nodes, not transformed values
					expectsTransformedValues = variadicType != reflect.TypeOf((*Node)(nil))
				} else if numIn > 1 {
					secondParamType := fnType.In(1)
					// If second param is not *Node, it expects transformed values
					expectsTransformedValues = secondParamType != reflect.TypeOf((*Node)(nil))
				}
			} else if firstParamType == reflect.TypeOf((*Node)(nil)) {
				// Node-aware but not context-aware - check if variadic with *Node
				if fnType.IsVariadic() && numIn > 1 {
					variadicType := fnType.In(numIn - 1).Elem()
					expectsTransformedValues = variadicType != reflect.TypeOf((*Node)(nil))
				}
			} else {
				// Old style - expects transformed values
				expectsTransformedValues = true
			}
		}

		// First call: pass children as *Node (not transformed)
		childNodes := make([]interface{}, len(node.Children))
		for i, child := range node.Children {
			childNodes[i] = child
		}

		result, err := callTransformWithPass(fn, ctx, node, childNodes, 0)
		if err != nil {
			return nil, err
		}

		// If function expects transformed values, we need to transform children first
		if expectsTransformedValues {
			// Transform children
			transformedChildren := make([]interface{}, len(node.Children))
			for i, child := range node.Children {
				childSiblings := node.Children
				childResult, err := transformNodeTopDown(child, node, childSiblings, i, tree, transforms)
				if err != nil {
					return nil, err
				}
				transformedChildren[i] = childResult
			}

			// Call transform again with transformed children
			result, err = callTransformWithPass(fn, ctx, node, transformedChildren, 0)
			if err != nil {
				return nil, err
			}
		} else {
			// Function expects *Node children - process them now
			transformedChildren := make([]interface{}, len(node.Children))
			for i, child := range node.Children {
				childSiblings := node.Children
				childResult, err := transformNodeTopDown(child, node, childSiblings, i, tree, transforms)
				if err != nil {
					return nil, err
				}
				transformedChildren[i] = childResult
			}

			// If result is a Node, we might need to update its children
			if resultNode, ok := result.(*Node); ok {
				resultNode.Children = node.Children // Keep original children structure
			}
		}

		return result, nil
	}

	// No transform for this rule - process children and return them
	transformedChildren := make([]interface{}, len(node.Children))
	for i, child := range node.Children {
		childSiblings := node.Children
		childResult, err := transformNodeTopDown(child, node, childSiblings, i, tree, transforms)
		if err != nil {
			return nil, err
		}
		transformedChildren[i] = childResult
	}

	// Return children (flattened)
	if len(transformedChildren) == 1 {
		return transformedChildren[0], nil
	}
	return transformedChildren, nil
}
