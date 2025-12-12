// Package ebnf implements a parser for EBNF grammars
package ebnf

import (
	"fmt"
	"strings"
	"unicode"
)

// Token types
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString
	TokenChar
	TokenComment
	TokenEquals
	TokenPipe
	TokenSemi
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenLBrace
	TokenRBrace
	TokenLAngle
	TokenRAngle
	TokenPlus
	TokenStar
	TokenQuestion
	TokenExclam
	TokenComma
	TokenCaret
	TokenMinus
	TokenRegex     // New token type for regex literals #"..."
	TokenSlash     // New token type for ordered choice /
	TokenAmpersand // New token type for positive lookahead &
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

// Lexer for EBNF
type Lexer struct {
	input string
	pos   int
	line  int
	col   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
		line:  1,
		col:   1,
	}
}

func (l *Lexer) current() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.current())) {
		l.advance()
	}
}

func (l *Lexer) readIdent() string {
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsLetter(rune(l.current())) ||
		unicode.IsDigit(rune(l.current())) || l.current() == '_') {
		l.advance()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readString(quote byte) (string, error) {
	l.advance() // skip opening quote
	var result strings.Builder

	for l.pos < len(l.input) && l.current() != quote {
		if l.current() == '\\' {
			l.advance()
			if l.pos >= len(l.input) {
				return "", fmt.Errorf("unexpected end of input in string")
			}
			// Handle escape sequences
			switch l.current() {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '\'':
				result.WriteByte('\'')
			case '"':
				result.WriteByte('"')
			default:
				result.WriteByte(l.current())
			}
		} else {
			result.WriteByte(l.current())
		}
		l.advance()
	}

	if l.current() != quote {
		return "", fmt.Errorf("unterminated string")
	}
	l.advance() // skip closing quote

	return result.String(), nil
}

func (l *Lexer) readComment() (string, error) {
	startLine, startCol := l.line, l.col
	l.advance() // skip (
	l.advance() // skip *

	var result strings.Builder
	for l.pos < len(l.input) {
		if l.pos < len(l.input)-1 && l.current() == '*' && l.input[l.pos+1] == ')' {
			l.advance() // skip *
			l.advance() // skip )
			return result.String(), nil
		}
		result.WriteByte(l.current())
		l.advance()
	}

	// Reached end of input without closing comment
	return "", fmt.Errorf("unterminated comment starting at line %d, col %d", startLine, startCol)
}

func (l *Lexer) readRegex() (string, error) {
	startLine, startCol := l.line, l.col
	l.advance() // skip #
	l.advance() // skip "

	var result strings.Builder

	for l.pos < len(l.input) && l.current() != '"' {
		if l.current() == '\\' {
			l.advance()
			if l.pos >= len(l.input) {
				return "", fmt.Errorf("unexpected end of input in regex at line %d, col %d", startLine, startCol)
			}
			// In regex, we preserve escape sequences as-is for the regex engine
			result.WriteByte('\\')
			result.WriteByte(l.current())
		} else {
			result.WriteByte(l.current())
		}
		l.advance()
	}

	if l.current() != '"' {
		return "", fmt.Errorf("unterminated regex starting at line %d, col %d", startLine, startCol)
	}
	l.advance() // skip closing "

	return result.String(), nil
}

func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF}, nil
	}

	line, col := l.line, l.col

	// Check for three-character tokens first
	if l.pos < len(l.input)-2 {
		threeChar := l.input[l.pos : l.pos+3]
		if threeChar == "::=" {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenEquals, Value: "::=", Line: line, Col: col}, nil
		}
	}

	// Check for two-character tokens
	if l.pos < len(l.input)-1 {
		twoChar := l.input[l.pos : l.pos+2]
		switch twoChar {
		case "(*":
			comment, err := l.readComment()
			if err != nil {
				return Token{}, err
			}
			return Token{Type: TokenComment, Value: comment, Line: line, Col: col}, nil
		case "<-":
			l.advance()
			l.advance()
			return Token{Type: TokenEquals, Value: "<-", Line: line, Col: col}, nil
		case ":=":
			l.advance()
			l.advance()
			return Token{Type: TokenEquals, Value: ":=", Line: line, Col: col}, nil
		case "#\"":
			regex, err := l.readRegex()
			if err != nil {
				return Token{}, err
			}
			return Token{Type: TokenRegex, Value: regex, Line: line, Col: col}, nil
		}
	}

	// Single character tokens
	switch l.current() {
	case '=':
		l.advance()
		return Token{Type: TokenEquals, Value: "=", Line: line, Col: col}, nil
	case ':':
		l.advance()
		return Token{Type: TokenEquals, Value: ":", Line: line, Col: col}, nil
	case '|':
		l.advance()
		return Token{Type: TokenPipe, Value: "|", Line: line, Col: col}, nil
	case ';':
		l.advance()
		return Token{Type: TokenSemi, Value: ";", Line: line, Col: col}, nil
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Value: "(", Line: line, Col: col}, nil
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Value: ")", Line: line, Col: col}, nil
	case '[':
		l.advance()
		return Token{Type: TokenLBracket, Value: "[", Line: line, Col: col}, nil
	case ']':
		l.advance()
		return Token{Type: TokenRBracket, Value: "]", Line: line, Col: col}, nil
	case '{':
		l.advance()
		return Token{Type: TokenLBrace, Value: "{", Line: line, Col: col}, nil
	case '}':
		l.advance()
		return Token{Type: TokenRBrace, Value: "}", Line: line, Col: col}, nil
	case '<':
		l.advance()
		return Token{Type: TokenLAngle, Value: "<", Line: line, Col: col}, nil
	case '>':
		l.advance()
		return Token{Type: TokenRAngle, Value: ">", Line: line, Col: col}, nil
	case '+':
		l.advance()
		return Token{Type: TokenPlus, Value: "+", Line: line, Col: col}, nil
	case '*':
		l.advance()
		return Token{Type: TokenStar, Value: "*", Line: line, Col: col}, nil
	case '?':
		l.advance()
		return Token{Type: TokenQuestion, Value: "?", Line: line, Col: col}, nil
	case '!':
		l.advance()
		return Token{Type: TokenExclam, Value: "!", Line: line, Col: col}, nil
	case ',':
		l.advance()
		return Token{Type: TokenComma, Value: ",", Line: line, Col: col}, nil
	case '^':
		l.advance()
		return Token{Type: TokenCaret, Value: "^", Line: line, Col: col}, nil
	case '-':
		l.advance()
		return Token{Type: TokenMinus, Value: "-", Line: line, Col: col}, nil
	case '/':
		l.advance()
		return Token{Type: TokenSlash, Value: "/", Line: line, Col: col}, nil
	case '&':
		l.advance()
		return Token{Type: TokenAmpersand, Value: "&", Line: line, Col: col}, nil
	case '\'':
		str, err := l.readString('\'')
		if err != nil {
			return Token{}, err
		}
		return Token{Type: TokenChar, Value: str, Line: line, Col: col}, nil
	case '"':
		str, err := l.readString('"')
		if err != nil {
			return Token{}, err
		}
		return Token{Type: TokenString, Value: str, Line: line, Col: col}, nil
	}

	// Identifier or single character (including digits for character classes)
	if unicode.IsLetter(rune(l.current())) || l.current() == '_' {
		ident := l.readIdent()
		return Token{Type: TokenIdent, Value: ident, Line: line, Col: col}, nil
	}

	// Single digit (for character classes like [0-9])
	if unicode.IsDigit(rune(l.current())) {
		digit := string(l.current())
		l.advance()
		return Token{Type: TokenIdent, Value: digit, Line: line, Col: col}, nil
	}

	return Token{}, fmt.Errorf("unexpected character: %c at line %d, col %d", l.current(), line, col)
}
