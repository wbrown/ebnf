package parse

import (
	"fmt"
	"reflect"
)

// TransformFunc is a function that transforms a node's children into a new value.
// It receives the children as separate arguments and returns a transformed value.
type TransformFunc interface{}

// TransformMap maps rule names to transformation functions.
// Functions should have the signature: func(args...interface{}) interface{}
// or any specific typed version like: func(float64, float64) float64
type TransformMap map[string]TransformFunc

// Transform applies transformations to a parse tree in a bottom-up manner.
// Each rule in the transformMap will have its corresponding function applied
// to the node's children, and the result replaces the node in the tree.
//
// Example:
//   result := parse.Transform(tree, parse.TransformMap{
//       "add": func(a, b float64) float64 { return a + b },
//       "number": strconv.ParseFloat,
//   })
func Transform(tree *ParseTree, transforms TransformMap) (interface{}, error) {
	if tree == nil || tree.Root == nil {
		return nil, fmt.Errorf("cannot transform nil tree")
	}
	return transformNode(tree.Root, transforms)
}

// TransformNode applies transformations to a single node and its descendants.
func TransformNode(node *Node, transforms TransformMap) (interface{}, error) {
	return transformNode(node, transforms)
}

// transformNode recursively transforms a node and its children
func transformNode(node *Node, transforms TransformMap) (interface{}, error) {
	// Base case: terminal node (leaf)
	if node.IsTerminal() {
		// If there's a transform for this rule, apply it
		if fn, ok := transforms[node.Rule]; ok {
			return callTransform(fn, []interface{}{node.Value})
		}
		// Otherwise return the terminal value as-is
		return node.Value, nil
	}

	// Recursively transform all children first (bottom-up)
	transformedChildren := make([]interface{}, len(node.Children))
	for i, child := range node.Children {
		transformed, err := transformNode(child, transforms)
		if err != nil {
			return nil, err
		}
		transformedChildren[i] = transformed
	}

	// If there's a transform function for this rule, apply it
	if fn, ok := transforms[node.Rule]; ok {
		return callTransform(fn, transformedChildren)
	}

	// No transform for this rule - return children as-is
	// If only one child, return it directly (flatten single-child nodes)
	if len(transformedChildren) == 1 {
		return transformedChildren[0], nil
	}
	return transformedChildren, nil
}

// callTransform invokes a transformation function with the given arguments.
// It handles various function signatures using reflection.
func callTransform(fn TransformFunc, args []interface{}) (interface{}, error) {
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()

	// Verify it's a function
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("transform must be a function, got %T", fn)
	}

	// Handle variadic functions
	isVariadic := fnType.IsVariadic()
	numIn := fnType.NumIn()

	// Convert args to reflect.Value slice
	argVals := make([]reflect.Value, len(args))
	for i, arg := range args {
		argVals[i] = reflect.ValueOf(arg)
	}

	// Check argument count
	if !isVariadic {
		if len(args) != numIn {
			return nil, fmt.Errorf("function expects %d arguments, got %d", numIn, len(args))
		}
	} else {
		// For variadic, we need at least numIn-1 args
		if len(args) < numIn-1 {
			return nil, fmt.Errorf("variadic function expects at least %d arguments, got %d", numIn-1, len(args))
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
			return nil, fmt.Errorf("argument %d: %w", i, err)
		}
		convertedArgs[i] = converted
	}

	// Call the function
	results := fnVal.Call(convertedArgs)

	// Handle return values
	if len(results) == 0 {
		return nil, nil
	}
	if len(results) == 1 {
		return results[0].Interface(), nil
	}
	// Multiple return values - check if last is error
	if len(results) == 2 && results[1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}
		return results[0].Interface(), nil
	}

	// Return all results as slice
	returnVals := make([]interface{}, len(results))
	for i, r := range results {
		returnVals[i] = r.Interface()
	}
	return returnVals, nil
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
