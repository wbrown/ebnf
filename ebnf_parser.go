package ebnf

import (
	"fmt"
)

// Parser for EBNF grammars
type Parser struct {
	lexer   *Lexer
	current Token
	peeked  *Token
}

func NewParser(input string) *Parser {
	return &Parser{
		lexer: NewLexer(input),
	}
}

func (p *Parser) advance() error {
	if p.peeked != nil {
		p.current = *p.peeked
		p.peeked = nil
	} else {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return err
		}
		p.current = tok
	}
	return nil
}

func (p *Parser) peek() (Token, error) {
	if p.peeked != nil {
		return *p.peeked, nil
	}
	tok, err := p.lexer.NextToken()
	if err != nil {
		return Token{}, err
	}
	p.peeked = &tok
	return tok, nil
}

// advanceSkipComments advances to the next token, skipping any comments
func (p *Parser) advanceSkipComments() error {
	if err := p.advance(); err != nil {
		return err
	}
	return p.skipComments()
}

// skipComments skips any comment tokens at the current position
func (p *Parser) skipComments() error {
	for p.current.Type == TokenComment {
		if err := p.advance(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) expect(typ TokenType) error {
	if p.current.Type != typ {
		return fmt.Errorf("expected %v, got %v at line %d, col %d",
			typ, p.current.Type, p.current.Line, p.current.Col)
	}
	return p.advanceSkipComments()
}

// ParseGrammar parses the entire EBNF grammar
func (p *Parser) ParseGrammar() (*Grammar, error) {
	// Initialize by reading first token
	if err := p.advanceSkipComments(); err != nil {
		return nil, err
	}

	grammar := &Grammar{}

	for p.current.Type != TokenEOF {
		// Comments are now automatically skipped

		rule, err := p.parseRule()
		if err != nil {
			return nil, err
		}
		if rule != nil {
			grammar.Rules = append(grammar.Rules, rule)
		}
	}

	return grammar, nil
}

// parseRule parses a single rule: name = expression ;
func (p *Parser) parseRule() (*Rule, error) {
	// Handle empty semicolons (from double semicolon)
	if p.current.Type == TokenSemi {
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		// Try to parse the next rule
		if p.current.Type == TokenEOF {
			return nil, nil // Skip this "empty rule"
		}
		return p.parseRule()
	}

	// Check for hidden rule: <rulename> = ...
	hidden := false
	if p.current.Type == TokenLAngle {
		hidden = true
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
	}

	if p.current.Type != TokenIdent {
		return nil, fmt.Errorf("expected rule name at line %d, col %d",
			p.current.Line, p.current.Col)
	}

	rule := &Rule{
		Name:   p.current.Value,
		Hidden: hidden,
	}

	if err := p.advanceSkipComments(); err != nil {
		return nil, err
	}

	// If rule was hidden, expect closing >
	if hidden {
		if err := p.expect(TokenRAngle); err != nil {
			return nil, err
		}
	}

	if err := p.expect(TokenEquals); err != nil {
		return nil, err
	}

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	rule.Expression = expr

	if err := p.expect(TokenSemi); err != nil {
		return nil, err
	}

	return rule, nil
}

// parseExpression parses a complete expression (handles both | and / choices)
func (p *Parser) parseExpression() (Expression, error) {
	// Parse ordered choice first (higher precedence)
	left, err := p.parseOrderedChoice()
	if err != nil {
		return nil, err
	}

	// Check for unordered choice (|)
	var alternatives []Expression
	for p.current.Type == TokenPipe {
		if alternatives == nil {
			alternatives = []Expression{left}
		}

		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}

		right, err := p.parseOrderedChoice()
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, right)
	}

	if alternatives != nil {
		return &Choice{Alternatives: alternatives}, nil
	}

	return left, nil
}

// parseOrderedChoice parses ordered choices (/)
func (p *Parser) parseOrderedChoice() (Expression, error) {
	left, err := p.parseSequence()
	if err != nil {
		return nil, err
	}

	// Check for ordered choice (/)
	var alternatives []Expression
	for p.current.Type == TokenSlash {
		if alternatives == nil {
			alternatives = []Expression{left}
		}

		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}

		right, err := p.parseSequence()
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, right)
	}

	if alternatives != nil {
		return &OrderedChoice{Alternatives: alternatives}, nil
	}

	return left, nil
}

// parseSequence parses a sequence of terms
func (p *Parser) parseSequence() (Expression, error) {
	var elements []Expression

	for {
		// Check for end of sequence
		switch p.current.Type {
		case TokenPipe, TokenSlash, TokenRParen, TokenRBracket, TokenRAngle, TokenSemi, TokenEOF:
			goto done
		}

		term, err := p.parseTerm()
		if err != nil {
			return nil, err
		}

		if term != nil {
			elements = append(elements, term)
		}
	}

done:
	if len(elements) == 0 {
		return &Empty{}, nil
	}
	if len(elements) == 1 {
		return elements[0], nil
	}
	return &Sequence{Elements: elements}, nil
}

// parseTerm parses a single term with optional suffixes (*, +, ?)
func (p *Parser) parseTerm() (Expression, error) {
	base, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	// Check for suffixes
	switch p.current.Type {
	case TokenStar:
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		return &Repetition{Expr: base}, nil
	case TokenPlus:
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		return &OneOrMore{Expr: base}, nil
	case TokenQuestion:
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		return &Optional{Expr: base}, nil
	}

	return base, nil
}

// parseFactor parses a base expression
func (p *Parser) parseFactor() (Expression, error) {
	switch p.current.Type {
	case TokenIdent:
		name := p.current.Value
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		return &NonTerminal{Name: name}, nil

	case TokenString:
		value := p.current.Value
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		return &Terminal{Value: value}, nil

	case TokenChar:
		value := p.current.Value
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		return &Terminal{Value: value}, nil

	case TokenRegex:
		pattern := p.current.Value
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		return &Regex{Pattern: pattern}, nil

	case TokenLParen:
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return &Group{Expr: expr}, nil

	case TokenLBracket:
		return p.parseCharClass()

	case TokenLBrace:
		// {expr} means zero or more repetitions of expr
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		// Parse just a single factor inside braces
		expr, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRBrace); err != nil {
			return nil, err
		}
		return &Repetition{Expr: expr}, nil

	case TokenLAngle:
		// Hidden expression
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}

		// Parse any expression inside < >
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if err := p.expect(TokenRAngle); err != nil {
			return nil, err
		}

		// Special handling for terminals, regex, non-terminals, and character classes to mark them as hidden
		switch e := expr.(type) {
		case *Terminal:
			e.Hidden = true
			return e, nil
		case *Regex:
			e.Hidden = true
			return e, nil
		case *NonTerminal:
			e.Hidden = true
			return e, nil
		case *CharClass:
			// Wrap character class in Hidden node
			return &Hidden{Expr: e}, nil
		default:
			// For other expressions, wrap in Hidden node
			return &Hidden{Expr: expr}, nil
		}

	case TokenExclam:
		// Negative lookahead
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		expr, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &Predicate{Expr: expr}, nil

	case TokenAmpersand:
		// Positive lookahead
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
		expr, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &PositiveLookahead{Expr: expr}, nil

	default:
		return nil, fmt.Errorf("unexpected token %v at line %d, col %d",
			p.current.Type, p.current.Line, p.current.Col)
	}
}

// ParseFile parses an EBNF grammar from a file
func ParseFile(filename string) (*Grammar, error) {
	// This would read the file and parse it
	// For now, we'll leave this as a stub
	return nil, fmt.Errorf("ParseFile not implemented")
}

// ParseString parses an EBNF grammar from a string
func ParseString(input string) (*Grammar, error) {
	parser := NewParser(input)
	return parser.ParseGrammar()
}
