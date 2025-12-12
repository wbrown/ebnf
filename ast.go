package ebnf

// AST nodes for EBNF grammar

type Grammar struct {
	Rules []*Rule
}

type Rule struct {
	Name       string
	Expression Expression
	Hidden     bool // true if rule name is wrapped in < >
}

// Expression is the interface for all EBNF expressions
type Expression interface {
	exprNode()
}

// Terminal represents a literal string or character
type Terminal struct {
	Value  string
	Hidden bool // true if wrapped in < >
}

// NonTerminal represents a reference to another rule
type NonTerminal struct {
	Name   string
	Hidden bool // true if wrapped in < >
}

// Sequence represents concatenation of expressions
type Sequence struct {
	Elements []Expression
}

// Choice represents alternatives (|)
type Choice struct {
	Alternatives []Expression
}

// OrderedChoice represents ordered alternatives (/)
type OrderedChoice struct {
	Alternatives []Expression
}

// Optional represents an optional expression (?)
type Optional struct {
	Expr Expression
}

// Repetition represents zero or more (*)
type Repetition struct {
	Expr Expression
}

// OneOrMore represents one or more (+)
type OneOrMore struct {
	Expr Expression
}

// Group represents a grouped expression in parentheses
type Group struct {
	Expr Expression
}

// Predicate represents a negative lookahead (!expr)
type Predicate struct {
	Expr Expression
}

// PositiveLookahead represents a positive lookahead (&expr)
type PositiveLookahead struct {
	Expr Expression
}

// Hidden represents a hidden expression <expr>
type Hidden struct {
	Expr Expression
}

func (*Hidden) exprNode() {}

// CharClass represents character classes like [a-zA-Z]
type CharClass struct {
	Chars   []rune
	Ranges  []CharRange
	Negated bool
}

type CharRange struct {
	From rune
	To   rune
}

// Empty represents an empty expression
type Empty struct{}

// Regex represents a regular expression pattern #"..."
type Regex struct {
	Pattern string
	Hidden  bool // true if wrapped in < >
}

// Implement exprNode() for all expression types
func (Terminal) exprNode()          {}
func (NonTerminal) exprNode()       {}
func (Sequence) exprNode()          {}
func (Choice) exprNode()            {}
func (OrderedChoice) exprNode()     {}
func (Optional) exprNode()          {}
func (Repetition) exprNode()        {}
func (OneOrMore) exprNode()         {}
func (Group) exprNode()             {}
func (Predicate) exprNode()         {}
func (PositiveLookahead) exprNode() {}
func (CharClass) exprNode()         {}
func (Empty) exprNode()             {}
func (Regex) exprNode()             {}
