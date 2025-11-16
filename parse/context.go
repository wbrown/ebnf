package parse

// TransformContext provides context information to transform functions.
// Transform functions can optionally receive *TransformContext as their first parameter
// (or second, if they also use *Node).
//
// The context provides access to:
//   - Tree: The full parse tree being transformed
//   - Node: The current node being transformed
//   - Parent: The parent node (if any)
//   - Siblings: Sibling nodes at the same level (if accessible)
//   - Index: Index of current node among siblings
//   - Input: The original input text
//   - State: Extensible map for storing state (per-node, not shared across nodes)
//
// Note: Each node gets its own TransformContext with its own State map.
// For state that needs to persist across multiple nodes, use closures to share
// a common state object.
type TransformContext struct {
	Tree     *ParseTree
	Node     *Node
	Parent   *Node
	Siblings []*Node
	Index    int // Index of current node among siblings
	Input    string
	State    map[string]interface{} // Per-node state storage
}

// Store stores a value in the context's state map
func (ctx *TransformContext) Store(key string, value interface{}) {
	if ctx.State == nil {
		ctx.State = make(map[string]interface{})
	}
	ctx.State[key] = value
}

// Get retrieves a value from the context's state map
func (ctx *TransformContext) Get(key string) interface{} {
	if ctx.State == nil {
		return nil
	}
	return ctx.State[key]
}

// GetInt retrieves an integer value from the context's state map
func (ctx *TransformContext) GetInt(key string, defaultValue int) int {
	if ctx.State == nil {
		return defaultValue
	}
	val, ok := ctx.State[key]
	if !ok {
		return defaultValue
	}
	if intVal, ok := val.(int); ok {
		return intVal
	}
	return defaultValue
}

// Set updates a value in the context's state map (alias for Store)
func (ctx *TransformContext) Set(key string, value interface{}) {
	ctx.Store(key, value)
}

// NextSibling returns the next sibling node, or nil if this is the last sibling
func (ctx *TransformContext) NextSibling() *Node {
	if ctx.Index < 0 || ctx.Siblings == nil {
		return nil
	}
	if ctx.Index+1 >= len(ctx.Siblings) {
		return nil
	}
	return ctx.Siblings[ctx.Index+1]
}

// PrevSibling returns the previous sibling node, or nil if this is the first sibling
func (ctx *TransformContext) PrevSibling() *Node {
	if ctx.Index < 0 || ctx.Siblings == nil {
		return nil
	}
	if ctx.Index-1 < 0 {
		return nil
	}
	return ctx.Siblings[ctx.Index-1]
}

// IsFirst returns true if this is the first sibling (index 0)
func (ctx *TransformContext) IsFirst() bool {
	return ctx.Index == 0
}

// IsLast returns true if this is the last sibling
func (ctx *TransformContext) IsLast() bool {
	if ctx.Siblings == nil {
		return ctx.Index == -1 // Root node
	}
	return ctx.Index >= 0 && ctx.Index == len(ctx.Siblings)-1
}

// SiblingCount returns the total number of siblings (including this node)
func (ctx *TransformContext) SiblingCount() int {
	if ctx.Siblings == nil {
		return 1 // Root node
	}
	return len(ctx.Siblings)
}

// GetMetadata retrieves metadata from this node's transform result.
// Metadata is stored when a transform function returns a TransformResult.
func (ctx *TransformContext) GetMetadata(key string) interface{} {
	if ctx.State == nil {
		return nil
	}
	return ctx.State["_meta:"+key]
}

// GetMetadataString retrieves a metadata value as a string
func (ctx *TransformContext) GetMetadataString(key string, defaultValue string) string {
	val := ctx.GetMetadata(key)
	if val == nil {
		return defaultValue
	}
	if str, ok := val.(string); ok {
		return str
	}
	return defaultValue
}

// GetMetadataInt retrieves a metadata value as an int
func (ctx *TransformContext) GetMetadataInt(key string, defaultValue int) int {
	val := ctx.GetMetadata(key)
	if val == nil {
		return defaultValue
	}
	if i, ok := val.(int); ok {
		return i
	}
	return defaultValue
}

// HasMetadata checks if a metadata key exists
func (ctx *TransformContext) HasMetadata(key string) bool {
	if ctx.State == nil {
		return false
	}
	_, ok := ctx.State["_meta:"+key]
	return ok
}
