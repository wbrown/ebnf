// Package ebnf provides a parser for Extended Backus-Naur Form (EBNF) grammars.
//
// This package supports standard EBNF syntax with modern extensions including
// regex patterns, PEG-style ordered choice, and hidden tokens.
//
// # Basic Usage
//
// Parse a grammar from a string:
//
//	grammar, err := ebnf.ParseString(`
//	    expr = term ( "+" term | "-" term )* ;
//	    term = number ;
//	    number = #"[0-9]+" ;
//	`)
//
// Load a grammar from a file:
//
//	grammar, err := ebnf.LoadGrammar("path/to/grammar.ebnf")
//
// Access rules:
//
//	rule := grammar.GetRule("expr")
//	if rule != nil {
//	    fmt.Printf("Rule type: %s\n", ebnf.ExpressionType(rule.Expression))
//	}
//
// # EBNF Syntax Supported
//
// Terminals:
//   - "text" or 'text' - literal strings
//   - #"regex" - regular expression patterns
//
// Non-terminals:
//   - identifier - reference to another rule
//
// Character classes:
//   - [a-z] - character ranges
//   - [abc] - character set
//   - [^abc] - negated character class
//
// Repetition:
//   - expr* - zero or more
//   - expr+ - one or more
//   - expr? - optional
//   - {expr} - zero or more (alternative syntax)
//
// Choices:
//   - a | b - unordered choice
//   - a / b - ordered choice (PEG-style, first match wins)
//
// Grouping:
//   - (expr) - group expressions
//
// Lookahead:
//   - !expr - negative lookahead
//   - &expr - positive lookahead
//
// Hidden tokens (omitted from AST):
//   - <"token"> - hidden terminal
//   - <rule> - hidden non-terminal
//   - <expr> - hidden expression
//
// Comments:
//   - (* comment *) - C-style comments
//
// Assignment operators:
//   - =, :=, ::=, <- - all supported
//
// # AST Structure
//
// The parser produces a typed AST with the following node types:
//
//   - Grammar: Container for all rules
//   - Rule: A named production rule
//   - Terminal: Literal string or character
//   - NonTerminal: Reference to another rule
//   - Sequence: Concatenation of expressions
//   - Choice: Unordered alternatives (|)
//   - OrderedChoice: Ordered alternatives (/)
//   - Optional: Optional expression (?)
//   - Repetition: Zero or more (*)
//   - OneOrMore: One or more (+)
//   - CharClass: Character class [...]
//   - Regex: Regular expression #"..."
//   - Predicate: Negative lookahead (!)
//   - PositiveLookahead: Positive lookahead (&)
//   - Hidden: Marks hidden expressions
//
// All AST node types implement the Expression interface.
//
// # Examples
//
// See the examples/ directory for complete grammar examples including:
//   - arithmetic.ebnf - Simple expression grammar
//   - json.ebnf - Complete JSON grammar
//   - regex_demo.ebnf - Regex pattern examples
//   - instaparse_demo.ebnf - PEG-style features
package ebnf
