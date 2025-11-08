package ebnf

import (
	"fmt"
	"strings"
)

// parseCharClass parses character classes like [a-zA-Z0-9_]
// This is called when we see a '[' and need to determine if it's a char class or choice
func (p *Parser) parseCharClass() (Expression, error) {
	startPos := p.current.Line
	startCol := p.current.Col
	
	if err := p.advanceSkipComments(); err != nil { // skip [
		return nil, err
	}
	
	// Peek ahead to determine if this is a character class or a choice
	// Character classes contain:
	// - Single characters: [abc]
	// - Ranges: [a-z]
	// - Escaped characters: [\n\t]
	// - Negation: [^abc]
	// Choices contain:
	// - Identifiers
	// - Quoted strings
	// - Complex expressions
	
	if p.current.Type == TokenEOF {
		return nil, fmt.Errorf("unexpected EOF in bracket expression at line %d, col %d", startPos, startCol)
	}
	
	// Check for negation
	negated := false
	if p.current.Type == TokenCaret {
		negated = true
		if err := p.advanceSkipComments(); err != nil {
			return nil, err
		}
	}
	
	// Try to parse as character class first
	chars := []rune{}
	ranges := []CharRange{}
	
	for p.current.Type != TokenRBracket && p.current.Type != TokenEOF {
		// In a character class, we expect:
		// - Single quoted characters
		// - Identifiers (for single char identifiers)
		// - Escaped sequences
		
		if p.current.Type == TokenChar {
			// Single character
			if len(p.current.Value) != 1 {
				// Multi-char string, this is a choice not a char class
				return p.parseChoiceFromBracket(startPos, startCol)
			}
			
			char := rune(p.current.Value[0])
			if err := p.advanceSkipComments(); err != nil {
				return nil, err
			}
			
			// Check for range
			if p.current.Type == TokenMinus {
				if err := p.advanceSkipComments(); err != nil {
					return nil, err
				}
				if p.current.Type != TokenChar || len(p.current.Value) != 1 {
					return nil, fmt.Errorf("invalid character range at line %d, col %d", 
						p.current.Line, p.current.Col)
				}
				endChar := rune(p.current.Value[0])
				ranges = append(ranges, CharRange{From: char, To: endChar})
				if err := p.advanceSkipComments(); err != nil {
					return nil, err
				}
			} else {
				chars = append(chars, char)
			}
		} else if p.current.Type == TokenIdent {
			// Single character identifier can be part of a char class
			if len(p.current.Value) == 1 {
				char := rune(p.current.Value[0])
				if err := p.advanceSkipComments(); err != nil {
					return nil, err
				}
				
				// Check for range
				if p.current.Type == TokenMinus {
					if err := p.advanceSkipComments(); err != nil {
						return nil, err
					}
					if (p.current.Type != TokenChar && p.current.Type != TokenIdent) || len(p.current.Value) != 1 {
						return nil, fmt.Errorf("invalid character range at line %d, col %d", 
							p.current.Line, p.current.Col)
					}
					endChar := rune(p.current.Value[0])
					ranges = append(ranges, CharRange{From: char, To: endChar})
					if err := p.advanceSkipComments(); err != nil {
						return nil, err
					}
				} else {
					chars = append(chars, char)
				}
			} else {
				// Multi-char identifier - this is a choice, not a char class
				return p.parseChoiceFromBracket(startPos, startCol)
			}
		} else if p.current.Type == TokenString {
			// Definitely a choice
			return p.parseChoiceFromBracket(startPos, startCol)
		} else if p.current.Type == TokenLAngle {
			// <...> inside brackets - definitely a choice
			return p.parseChoiceFromBracket(startPos, startCol)
		} else if p.current.Type == TokenComment {
			// Comment inside brackets - definitely a choice (char classes don't have comments)
			return p.parseChoiceFromBracket(startPos, startCol)
		} else if p.current.Type == TokenPipe {
			// Pipe inside brackets - definitely a choice
			return p.parseChoiceFromBracket(startPos, startCol)
		} else if p.current.Type == TokenLParen {
			// Parentheses inside brackets - definitely a choice
			return p.parseChoiceFromBracket(startPos, startCol)
		} else {
			// Unexpected token
			return nil, fmt.Errorf("unexpected token %v in bracket expression at line %d, col %d",
				p.current.Type, p.current.Line, p.current.Col)
		}
	}
	
	if err := p.expect(TokenRBracket); err != nil {
		return nil, err
	}
	
	return &CharClass{
		Chars:   chars,
		Ranges:  ranges,
		Negated: negated,
	}, nil
}

// parseChoiceFromBracket continues parsing a bracketed expression as a choice
// We've already consumed the '[' 
func (p *Parser) parseChoiceFromBracket(startLine, startCol int) (Expression, error) {
	// We need to reset and re-parse as a choice
	// For now, we'll parse it similar to how we parse regular choices
	
	var alternatives []Expression
	for p.current.Type != TokenRBracket && p.current.Type != TokenEOF {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, expr)
		
		if p.current.Type == TokenPipe {
			if err := p.advanceSkipComments(); err != nil {
				return nil, err
			}
		} else if p.current.Type != TokenRBracket {
			// No pipe and not closing bracket - error
			return nil, fmt.Errorf("expected | or ] in choice at line %d, col %d",
				p.current.Line, p.current.Col)
		}
	}
	
	if err := p.expect(TokenRBracket); err != nil {
		return nil, err
	}
	
	if len(alternatives) == 0 {
		return &Empty{}, nil
	}
	if len(alternatives) == 1 {
		return alternatives[0], nil
	}
	return &Choice{Alternatives: alternatives}, nil
}

// String representation for debugging
func (c *CharClass) String() string {
	var parts []string
	
	for _, ch := range c.Chars {
		parts = append(parts, fmt.Sprintf("%c", ch))
	}
	
	for _, r := range c.Ranges {
		parts = append(parts, fmt.Sprintf("%c-%c", r.From, r.To))
	}
	
	result := "[" + strings.Join(parts, "") + "]"
	if c.Negated {
		result = "[^" + result[1:]
	}
	return result
}