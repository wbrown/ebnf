package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/wbrown/ebnf"
	"github.com/wbrown/ebnf/parse"
)

// Evaluator walks a parse tree and evaluates arithmetic expressions
type Evaluator struct{}

// Eval evaluates a parse tree node and returns the result
func (e *Evaluator) Eval(node *parse.Node) (float64, error) {
	// Handle leaf nodes (numbers)
	if node.Rule == "" && node.Value != "" {
		// This is a terminal value node
		return strconv.ParseFloat(node.Value, 64)
	}

	switch node.Rule {
	case "expr":
		return e.evalExpr(node)
	case "term":
		return e.evalTerm(node)
	case "factor":
		return e.evalFactor(node)
	case "number":
		return e.evalNumber(node)
	default:
		return 0, fmt.Errorf("unknown rule: %s", node.Rule)
	}
}

// evalExpr evaluates: term ( addop term )*
func (e *Evaluator) evalExpr(node *parse.Node) (float64, error) {
	if len(node.Children) == 0 {
		return 0, fmt.Errorf("expr has no children")
	}

	// First child is always a term (or terminal in flattened tree)
	result, err := e.Eval(node.Children[0])
	if err != nil {
		return 0, err
	}

	// Process remaining children: operator, operand, operator, operand, ...
	for i := 1; i < len(node.Children); i += 2 {
		if i+1 >= len(node.Children) {
			break
		}

		// Operator might be a rule node (addop/mulop) or a terminal
		op := node.Children[i]
		var opValue string
		if op.Value != "" {
			opValue = op.Value
		} else if len(op.Children) > 0 && op.Children[0].Value != "" {
			// Operator is a rule node with a terminal child
			opValue = op.Children[0].Value
		} else {
			return 0, fmt.Errorf("cannot determine operator at position %d", i)
		}

		// Next operand
		operand := node.Children[i+1]
		operandValue, err := e.Eval(operand)
		if err != nil {
			return 0, err
		}

		switch opValue {
		case "+":
			result += operandValue
		case "-":
			result -= operandValue
		default:
			return 0, fmt.Errorf("unknown operator: %s", opValue)
		}
	}

	return result, nil
}

// evalTerm evaluates: factor ( mulop factor )*
func (e *Evaluator) evalTerm(node *parse.Node) (float64, error) {
	if len(node.Children) == 0 {
		return 0, fmt.Errorf("term has no children")
	}

	// First child is always a factor (or terminal in flattened tree)
	result, err := e.Eval(node.Children[0])
	if err != nil {
		return 0, err
	}

	// Process remaining children: operator, operand, operator, operand, ...
	for i := 1; i < len(node.Children); i += 2 {
		if i+1 >= len(node.Children) {
			break
		}

		// Operator might be a rule node (addop/mulop) or a terminal
		op := node.Children[i]
		var opValue string
		if op.Value != "" {
			opValue = op.Value
		} else if len(op.Children) > 0 && op.Children[0].Value != "" {
			// Operator is a rule node with a terminal child
			opValue = op.Children[0].Value
		} else {
			return 0, fmt.Errorf("cannot determine operator at position %d", i)
		}

		// Next operand
		operand := node.Children[i+1]
		operandValue, err := e.Eval(operand)
		if err != nil {
			return 0, err
		}

		switch opValue {
		case "*":
			result *= operandValue
		case "/":
			if operandValue == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			result /= operandValue
		default:
			return 0, fmt.Errorf("unknown operator: %s", opValue)
		}
	}

	return result, nil
}

// evalFactor evaluates: number | "(" expr ")"
func (e *Evaluator) evalFactor(node *parse.Node) (float64, error) {
	if len(node.Children) == 0 {
		// Might be a terminal number value directly
		if node.Value != "" {
			return strconv.ParseFloat(node.Value, 64)
		}
		return 0, fmt.Errorf("factor has no children and no value")
	}

	// Factor is either a number or a parenthesized expression
	return e.Eval(node.Children[0])
}

// evalNumber extracts the number value
func (e *Evaluator) evalNumber(node *parse.Node) (float64, error) {
	if len(node.Children) == 0 {
		// Might be a terminal value
		if node.Value != "" {
			return strconv.ParseFloat(node.Value, 64)
		}
		return 0, fmt.Errorf("number has no children and no value")
	}

	// The number rule has a single child which is the regex match
	return e.Eval(node.Children[0])
}

func main() {
	// Load the arithmetic grammar
	grammar, err := ebnf.LoadGrammar("../arithmetic.ebnf")
	if err != nil {
		log.Fatal(err)
	}

	// Create a parser
	parser := parse.New(grammar)

	// Parse some expressions
	expressions := []string{
		"2 + 3",
		"2 + 3 * 4",
		"(2 + 3) * 4",
		"10 / 2 - 3",
		"1.5 + 2.5 * 2",
	}

	evaluator := &Evaluator{}

	for _, expr := range expressions {
		fmt.Printf("Expression: %s\n", expr)

		// Parse the expression
		tree, err := parser.Parse(expr, "expr")
		if err != nil {
			fmt.Printf("  Parse error: %v\n\n", err)
			continue
		}

		// Optionally show the parse tree
		if len(os.Args) > 1 && os.Args[1] == "-tree" {
			fmt.Println("  Parse tree:")
			parse.PrintAST(os.Stdout, tree)
			fmt.Println()
		}

		// Evaluate the expression
		result, err := evaluator.Eval(tree.Root)
		if err != nil {
			fmt.Printf("  Eval error: %v\n\n", err)
			continue
		}

		fmt.Printf("  Result: %.2f\n\n", result)
	}
}
