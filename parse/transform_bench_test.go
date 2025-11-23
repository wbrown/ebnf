package parse

import (
	"strconv"
	"testing"
)

// BenchmarkTransform_SimpleArithmetic measures the performance of a simple arithmetic transform
// This is the primary target for our optimization (func(float64, float64) float64)
func BenchmarkTransform_SimpleArithmetic(b *testing.B) {
	// Create a deep tree to magnify the transform overhead
	// add(add(add(..., 1), 1), 1)
	depth := 100
	var root *Node = &Node{Rule: "number", Value: "1"}

	for i := 0; i < depth; i++ {
		root = &Node{
			Rule: "add",
			Children: []*Node{
				root,
				{Rule: "number", Value: "1"},
			},
		}
	}

	tree := &ParseTree{Root: root}

	transforms := TransformMap{
		"number": func(s string) float64 {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		},
		"add": func(a, b float64) float64 {
			return a + b
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Transform(tree, transforms)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTransform_StringConcat measures string concatenation performance
// Target: func(...interface{}) string
func BenchmarkTransform_StringConcat(b *testing.B) {
	// concat(word, word, word, ...)
	count := 100
	children := make([]*Node, count)
	for i := 0; i < count; i++ {
		children[i] = &Node{Rule: "word", Value: "test"}
	}

	tree := &ParseTree{
		Root: &Node{
			Rule:     "concat",
			Children: children,
		},
	}

	transforms := TransformMap{
		"word": Identity,
		"concat": func(args ...interface{}) string {
			var res string
			for _, arg := range args {
				res += arg.(string)
			}
			return res
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Transform(tree, transforms)
		if err != nil {
			b.Fatal(err)
		}
	}
}
