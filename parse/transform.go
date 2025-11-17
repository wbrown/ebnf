package parse

import (
	"fmt"
	"reflect"
)

// TransformFunc is a function that transforms a node's children into a new value.
// It receives the children as separate arguments and returns a transformed value.
type TransformFunc interface{}

// TransformMap maps rule names to transformation functions.
// Functions can have one of several signatures:
//  1. Old style: func(args...interface{}) interface{}
//     or any specific typed version like: func(float64, float64) float64
//  2. Node-aware: func(node *Node, args...interface{}) interface{}
//     The node parameter provides access to source position information
//     (Line, Column, Start, End) and the original parse tree node.
//  3. Context-aware: func(ctx *TransformContext, args...interface{}) interface{}
//     The context provides access to tree, node, parent, siblings, and state.
//  4. Combined: func(ctx *TransformContext, node *Node, args...interface{}) interface{}
//     Both context and node (context must come first).
//
// All styles can be mixed in the same TransformMap. The system automatically
// detects which style is used based on the function's parameter types.
type TransformMap map[string]TransformFunc

// Transform applies transformations to a parse tree in a bottom-up manner.
// Each rule in the transformMap will have its corresponding function applied
// to the node's children, and the result replaces the node in the tree.
//
// Example (old style, without node access):
//   result := parse.Transform(tree, parse.TransformMap{
//       "add": func(a, b float64) float64 { return a + b },
//       "number": strconv.ParseFloat,
//   })
//
// Example (node-aware, with position access):
//   result := parse.Transform(tree, parse.TransformMap{
//       "number": func(node *Node, s string) (*Number, error) {
//           val, _ := strconv.Atoi(s)
//           return &Number{Value: val, Line: node.Line}, nil
//       },
//   })
//
// Example (context-aware, with tree, parent, siblings access):
//   result := parse.Transform(tree, parse.TransformMap{
//       "number": func(ctx *TransformContext, s string) (*Number, error) {
//           val, _ := strconv.Atoi(s)
//           return &Number{Value: val, Line: ctx.Node.Line}, nil
//       },
//   })
func Transform(tree *ParseTree, transforms TransformMap) (interface{}, error) {
	if tree == nil || tree.Root == nil {
		return nil, fmt.Errorf("cannot transform nil tree")
	}
	return transformNodeWithContext(tree.Root, nil, nil, -1, tree, transforms, false)
}

// TransformNode applies transformations to a single node and its descendants.
func TransformNode(node *Node, transforms TransformMap) (interface{}, error) {
	// Create a minimal tree wrapper for context
	tree := &ParseTree{Root: node, Input: ""}
	return transformNodeWithContext(node, nil, nil, -1, tree, transforms, false)
}

// TransformPreserveStructure applies transformations while preserving tree structure.
// Unlike Transform(), nodes without transforms are preserved as Node objects instead
// of being flattened to their children. This is useful for multi-pass transformations.
func TransformPreserveStructure(tree *ParseTree, transforms TransformMap) (interface{}, error) {
	return transformPreserveStructureWithPass(tree, transforms, 0)
}

// transformPreserveStructureWithPass applies transformations while preserving tree structure,
// including pass number in error context for multi-pass transformations.
func transformPreserveStructureWithPass(tree *ParseTree, transforms TransformMap, passNumber int) (interface{}, error) {
	if tree == nil || tree.Root == nil {
		return nil, fmt.Errorf("cannot transform nil tree")
	}
	return transformNodeWithContextAndPass(tree.Root, nil, nil, -1, tree, transforms, true, passNumber)
}

// transformNodeWithContext recursively transforms a node and its children with context.
// It tracks parent, siblings, and index information for context-aware transforms.
// If preserveStructure is true, nodes without transforms are preserved as Node objects
// instead of being flattened to their children.
func transformNodeWithContext(node *Node, parent *Node, siblings []*Node, index int, tree *ParseTree, transforms TransformMap, preserveStructure bool) (interface{}, error) {
	return transformNodeWithContextAndPass(node, parent, siblings, index, tree, transforms, preserveStructure, 0)
}

// transformNodeWithContextAndPass recursively transforms a node with context and pass number.
// The pass number is included in error context for multi-pass transformations.
func transformNodeWithContextAndPass(node *Node, parent *Node, siblings []*Node, index int, tree *ParseTree, transforms TransformMap, preserveStructure bool, passNumber int) (interface{}, error) {
	// Base case: terminal node (leaf)
	if node.IsTerminal() {
		// If there's a transform for this rule, apply it
		if fn, ok := transforms[node.Rule]; ok {
			ctx := &TransformContext{
				Tree:     tree,
				Node:     node,
				Parent:   parent,
				Siblings: siblings,
				Index:    index,
				Input:    tree.Input,
				State:    make(map[string]interface{}),
			}
			// For _transformed nodes, prefer TransformedValue (preserves type), fall back to Value (string)
			var value interface{} = node.Value
			if node.Rule == "_transformed" && node.TransformedValue != nil {
				value = node.TransformedValue
			}
			return callTransformWithPass(fn, ctx, node, []interface{}{value}, passNumber)
		}
		// Otherwise return the terminal value as-is (or node if preserving structure)
		if preserveStructure {
			return node, nil
		}
		// For _transformed nodes, prefer TransformedValue (preserves type), fall back to Value (string)
		if node.Rule == "_transformed" && node.TransformedValue != nil {
			return node.TransformedValue, nil
		}
		return node.Value, nil
	}

	// Recursively transform all children first (bottom-up)
	transformedChildren := make([]interface{}, len(node.Children))
	for i, child := range node.Children {
		transformed, err := transformNodeWithContextAndPass(child, node, node.Children, i, tree, transforms, preserveStructure, passNumber)
		if err != nil {
			// Error already wrapped with context from child
			return nil, err
		}
		transformedChildren[i] = transformed
	}

	// If there's a transform function for this rule, apply it
	if fn, ok := transforms[node.Rule]; ok {
		ctx := &TransformContext{
			Tree:     tree,
			Node:     node,
			Parent:   parent,
			Siblings: siblings,
			Index:    index,
			Input:    tree.Input,
			State:    make(map[string]interface{}),
		}
		// If preserving structure, unwrap _transformed nodes to get actual values
		// before passing to transform function
		unwrappedChildren := transformedChildren
		if preserveStructure {
			unwrappedChildren = make([]interface{}, len(transformedChildren))
			for i, tc := range transformedChildren {
				if trNode, ok := tc.(*Node); ok && trNode.Rule == "_transformed" {
					// Unwrap _transformed node to get the actual value
					// Prefer TransformedValue (preserves type), fall back to Value (string)
					if trNode.TransformedValue != nil {
						unwrappedChildren[i] = trNode.TransformedValue
					} else {
						unwrappedChildren[i] = trNode.Value
					}
				} else {
					unwrappedChildren[i] = tc
				}
			}
		}
		return callTransformWithPass(fn, ctx, node, unwrappedChildren, passNumber)
	}

	// No transform for this rule
	if preserveStructure {
		// Preserve structure: return a Node with transformed children
		preservedNode := &Node{
			Rule:     node.Rule,
			Value:    node.Value,
			Line:     node.Line,
			Column:   node.Column,
			Start:    node.Start,
			End:      node.End,
			Children: make([]*Node, len(transformedChildren)),
		}
		// Convert transformed children back to Nodes if possible
		for i, tc := range transformedChildren {
			if childNode, ok := tc.(*Node); ok {
				preservedNode.Children[i] = childNode
			} else {
				// Create a synthetic node for the transformed value
				// Store both string representation (for compatibility) and actual typed value
				preservedNode.Children[i] = &Node{
					Rule:            "_transformed",
					Value:           fmt.Sprintf("%v", tc), // String representation
					TransformedValue: tc,                    // Actual typed value (preserves type)
				}
			}
		}
		return preservedNode, nil
	}

	// Default behavior: return children as-is (flatten)
	// If only one child, return it directly (flatten single-child nodes)
	if len(transformedChildren) == 1 {
		return transformedChildren[0], nil
	}
	return transformedChildren, nil
}

// callTransform invokes a transformation function with the given arguments.
// It handles various function signatures using reflection.
// Supports: old style, node-aware, context-aware, and combined signatures.
// Errors are wrapped with context information including node position and rule name.
func callTransform(fn TransformFunc, ctx *TransformContext, node *Node, args []interface{}) (interface{}, error) {
	return callTransformWithPass(fn, ctx, node, args, 0)
}

// callTransformWithPass is like callTransform but includes pass number for error context
func callTransformWithPass(fn TransformFunc, ctx *TransformContext, node *Node, args []interface{}, passNumber int) (interface{}, error) {
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()

	// Get rule name for error context
	ruleName := ""
	if node != nil {
		ruleName = node.Rule
	}
	
	// Verify it's a function
	if fnType.Kind() != reflect.Func {
		err := fmt.Errorf("transform must be a function, got %T", fn)
		return nil, wrapTransformError(err, ctx, ruleName, passNumber)
	}

	numIn := fnType.NumIn()
	
	// Check parameter types to determine function signature
	firstParamIsContext := numIn > 0 && fnType.In(0) == reflect.TypeOf((*TransformContext)(nil))
	firstParamIsNode := numIn > 0 && fnType.In(0) == reflect.TypeOf((*Node)(nil))
	secondParamIsNode := numIn > 1 && firstParamIsContext && fnType.In(1) == reflect.TypeOf((*Node)(nil))

	// Build argument list based on function signature
	callArgs := make([]interface{}, 0, len(args)+2) // Pre-allocate with room for ctx/node
	
	// Handle context parameter (first or only special param)
	if firstParamIsContext {
		callArgs = append(callArgs, ctx)
		// If second param is also Node, add it
		if secondParamIsNode {
			callArgs = append(callArgs, node)
			callArgs = append(callArgs, args...)
		} else {
			callArgs = append(callArgs, args...)
		}
	} else if firstParamIsNode {
		// Node-aware but not context-aware
		callArgs = append(callArgs, node)
		callArgs = append(callArgs, args...)
	} else {
		// Old style - no special parameters
		callArgs = args
	}

	// Handle variadic functions
	isVariadic := fnType.IsVariadic()

	// Convert callArgs to reflect.Value slice
	argVals := make([]reflect.Value, len(callArgs))
	for i, arg := range callArgs {
		if arg == nil {
			// Create a proper nil interface{} value to avoid zero Value panic
			argVals[i] = reflect.Zero(reflect.TypeOf((*interface{})(nil)).Elem())
		} else {
			argVals[i] = reflect.ValueOf(arg)
		}
	}

	// Check argument count
	if !isVariadic {
		if len(callArgs) != numIn {
			// Provide helpful error message with signature hint
			var hint string
			if firstParamIsContext {
				if secondParamIsNode {
					hint = fmt.Sprintf(" (signature: func(ctx *TransformContext, node *Node, args...) - got %d args including ctx+node)", len(callArgs))
				} else {
					hint = fmt.Sprintf(" (signature: func(ctx *TransformContext, args...) - got %d args including ctx)", len(callArgs))
				}
			} else if firstParamIsNode {
				hint = fmt.Sprintf(" (signature: func(node *Node, args...) - got %d args including node)", len(callArgs))
			} else {
				hint = fmt.Sprintf(" (signature: func(args...) - got %d args)", len(callArgs))
			}
			err := fmt.Errorf("function expects %d arguments, got %d%s. Check that your transform function signature matches the number of child nodes", numIn, len(callArgs), hint)
			return nil, wrapTransformError(err, ctx, ruleName, passNumber)
		}
	} else {
		// For variadic, we need at least numIn-1 args
		minArgs := numIn - 1
		if len(callArgs) < minArgs {
			return nil, fmt.Errorf("variadic function expects at least %d arguments, got %d", minArgs, len(callArgs))
		}
	}

	// Convert arguments to match expected types if needed
	convertedArgs := make([]reflect.Value, len(argVals))
	for i, argVal := range argVals {
		var expectedType reflect.Type
		if isVariadic && i >= numIn-1 {
			// For variadic args, use the element type of the slice
			expectedType = fnType.In(numIn - 1).Elem()
		} else {
			expectedType = fnType.In(i)
		}

		converted, err := convertArg(argVal, expectedType)
		if err != nil {
			err = fmt.Errorf("argument %d: %w", i, err)
			return nil, wrapTransformError(err, ctx, ruleName, passNumber)
		}
		convertedArgs[i] = converted
	}

	// Call the function with panic recovery
	var results []reflect.Value
	var panicErr *TransformError
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Check if it's already a TransformError (from nested call)
				if te, ok := r.(*TransformError); ok {
					panicErr = te
				} else {
					panicErr = wrapPanic(r, ctx, ruleName, passNumber)
				}
			}
		}()
		results = fnVal.Call(convertedArgs)
	}()
	
	// If panic occurred, return the error
	if panicErr != nil {
		return nil, panicErr
	}

	// Handle return values
	if len(results) == 0 {
		return nil, nil
	}
	
	// Extract the main return value
	var returnVal interface{}
	if len(results) == 1 {
		returnVal = results[0].Interface()
	} else if len(results) == 2 && results[1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		// Two return values: (value, error)
		if !results[1].IsNil() {
			err := results[1].Interface().(error)
			return nil, wrapTransformError(err, ctx, ruleName, passNumber)
		}
		returnVal = results[0].Interface()
	} else {
		// Multiple return values - return as slice
		returnVals := make([]interface{}, len(results))
		for i, r := range results {
			returnVals[i] = r.Interface()
		}
		return returnVals, nil
	}
	
	// Check if return value is TransformResult
	if tr, ok := returnVal.(*TransformResult); ok {
		// Store metadata in context for child transforms to access
		if ctx != nil && tr.Metadata != nil {
			// Merge metadata into context state with special prefix
			if ctx.State == nil {
				ctx.State = make(map[string]interface{})
			}
			for k, v := range tr.Metadata {
				ctx.State["_meta:"+k] = v
			}
			// Store reference to result node
			if tr.Node != nil {
				ctx.State["_meta:node"] = tr.Node
			}
		}
		// Return the value, not the wrapper
		return tr.Value, nil
	}
	
	// Regular return value
	return returnVal, nil
}

// convertArg attempts to convert a value to the expected type
func convertArg(val reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	// If types already match, no conversion needed
	if val.Type().AssignableTo(targetType) {
		return val, nil
	}

	// Handle interface{} - any type can be assigned
	if targetType.Kind() == reflect.Interface && targetType.NumMethod() == 0 {
		return val, nil
	}

	// Try direct conversion if types are convertible
	if val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType), nil
	}

	// Handle string to numeric conversions
	if val.Kind() == reflect.String {
		str := val.String()
		switch targetType.Kind() {
		case reflect.Float64, reflect.Float32:
			return convertStringToFloat(str, targetType)
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			return convertStringToInt(str, targetType)
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			return convertStringToUint(str, targetType)
		}
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %v (type %s) to %s",
		val.Interface(), val.Type(), targetType)
}

// convertStringToFloat converts a string to a float type
func convertStringToFloat(s string, targetType reflect.Type) (reflect.Value, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("cannot parse %q as float: %w", s, err)
	}

	val := reflect.ValueOf(f)
	if val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot convert float64 to %s", targetType)
}

// convertStringToInt converts a string to an int type
func convertStringToInt(s string, targetType reflect.Type) (reflect.Value, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("cannot parse %q as int: %w", s, err)
	}

	val := reflect.ValueOf(i)
	if val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot convert int64 to %s", targetType)
}

// convertStringToUint converts a string to a uint type
func convertStringToUint(s string, targetType reflect.Type) (reflect.Value, error) {
	var u uint64
	_, err := fmt.Sscanf(s, "%d", &u)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("cannot parse %q as uint: %w", s, err)
	}

	val := reflect.ValueOf(u)
	if val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot convert uint64 to %s", targetType)
}

// Helper functions for common transformation patterns

// Identity returns the first argument unchanged - useful for pass-through transformations
func Identity(args ...interface{}) interface{} {
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

// Flatten returns all arguments as a slice
func Flatten(args ...interface{}) interface{} {
	return args
}

// First returns the first argument
func First(args ...interface{}) interface{} {
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

// Last returns the last argument
func Last(args ...interface{}) interface{} {
	if len(args) == 0 {
		return nil
	}
	return args[len(args)-1]
}

// Concat concatenates all string arguments
func Concat(args ...interface{}) interface{} {
	var result string
	for _, arg := range args {
		if str, ok := arg.(string); ok {
			result += str
		}
	}
	return result
}
