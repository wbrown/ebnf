package parse

// TransformResult wraps a transformed value with metadata.
// Transform functions can optionally return TransformResult instead of plain values
// to attach metadata (source positions, debugging info, etc.) to transformed values.
//
// Example:
//
//	transforms := TransformMap{
//	    "number": func(node *Node, s string) *TransformResult {
//	        val, _ := strconv.Atoi(s)
//	        return &TransformResult{
//	            Value: val,
//	            Metadata: map[string]interface{}{
//	                "source_pos": node.Start,
//	                "line": node.Line,
//	            },
//	            Node: node,
//	        }
//	    },
//	}
type TransformResult struct {
	Value    interface{}            // The transformed value
	Metadata map[string]interface{} // Optional metadata (source positions, types, etc.)
	Node     *Node                  // Reference to original parse tree node
}

// GetMetadata retrieves a metadata value by key
func (tr *TransformResult) GetMetadata(key string) interface{} {
	if tr.Metadata == nil {
		return nil
	}
	return tr.Metadata[key]
}

// GetMetadataString retrieves a metadata value as a string
func (tr *TransformResult) GetMetadataString(key string, defaultValue string) string {
	if tr.Metadata == nil {
		return defaultValue
	}
	val, ok := tr.Metadata[key]
	if !ok {
		return defaultValue
	}
	if str, ok := val.(string); ok {
		return str
	}
	return defaultValue
}

// GetMetadataInt retrieves a metadata value as an int
func (tr *TransformResult) GetMetadataInt(key string, defaultValue int) int {
	if tr.Metadata == nil {
		return defaultValue
	}
	val, ok := tr.Metadata[key]
	if !ok {
		return defaultValue
	}
	if i, ok := val.(int); ok {
		return i
	}
	return defaultValue
}

// HasMetadata checks if a metadata key exists
func (tr *TransformResult) HasMetadata(key string) bool {
	if tr.Metadata == nil {
		return false
	}
	_, ok := tr.Metadata[key]
	return ok
}
