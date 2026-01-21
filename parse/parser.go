package parse

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wbrown/ebnf"
)

// Parser uses an EBNF grammar to parse input text
type Parser struct {
	grammar  *ebnf.Grammar
	input    string
	pos      int
	line     int
	col      int
	Debug    bool            // Enable debug output
	depth    int             // Track recursion depth for debug indentation
	debugLog strings.Builder // Capture debug output

	// Focused debugging
	focusPos   int // Position to focus on (-1 = disabled)
	focusRange int // Range around focus position

	// Regex cache to avoid recompiling the same patterns
	regexCache map[string]*regexp.Regexp

	// Expression rules that should be flattened (cached to avoid map allocation)
	exprRules map[string]bool

	// Track furthest position reached for better error messages
	furthestPos   int    // Furthest position where parsing was attempted
	furthestLine  int    // Line at furthest position
	furthestCol   int    // Column at furthest position
	furthestError error  // Error at furthest position
	furthestRule  string // Rule being attempted at furthest position

	// Case-insensitive matching (global default)
	caseInsensitive bool
}

// Option is a function that configures a Parser
type Option func(*Parser)

// WithCaseInsensitive sets the global default for case-insensitive matching.
// By default, terminals are case-sensitive. When set to true, all terminals
// will match case-insensitively using strings.EqualFold (faster than regex).
// Per-terminal 'i' suffix (e.g., 'hello'i) can be used for selective case-insensitivity.
func WithCaseInsensitive(ci bool) Option {
	return func(p *Parser) {
		p.caseInsensitive = ci
	}
}

// New creates a new parser with the given EBNF grammar
func New(grammar *ebnf.Grammar, opts ...Option) *Parser {
	p := &Parser{
		grammar:    grammar,
		focusPos:   -1,
		regexCache: make(map[string]*regexp.Regexp),
		exprRules: map[string]bool{
			"or_expr":      true,
			"and_expr":     true,
			"eq_expr":      true,
			"rel_expr":     true,
			"concat_expr":  true,
			"add_expr":     true,
			"mult_expr":    true,
			"fair_expr":    true,
			"exp_expr":     true,
			"unary_expr":   true,
			"primary_expr": true,
		},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// SetFocusedDebug enables detailed debugging around a specific position
func (p *Parser) SetFocusedDebug(pos, rangeSize int) {
	p.focusPos = pos
	p.focusRange = rangeSize
}

// debugf prints debug output if Debug is enabled or if we're in focus range
func (p *Parser) debugf(format string, args ...interface{}) {
	shouldDebug := p.Debug

	// Check if we're in focused debug range
	if p.focusPos >= 0 && !shouldDebug {
		if p.pos >= p.focusPos-p.focusRange && p.pos <= p.focusPos+p.focusRange {
			shouldDebug = true
		}
	}

	if shouldDebug {
		indent := strings.Repeat("  ", p.depth)
		posInfo := fmt.Sprintf("[pos=%d,line=%d,col=%d]", p.pos, p.line, p.col)
		msg := fmt.Sprintf("%s%s %s\n", indent, posInfo, fmt.Sprintf(format, args...))
		fmt.Print(msg)
		p.debugLog.WriteString(msg)
	}
}

// GetDebugLog returns the captured debug log
func (p *Parser) GetDebugLog() string {
	return p.debugLog.String()
}

// Parse parses the input according to the grammar, starting with the given rule
func (p *Parser) Parse(input string, startRule string) (*ParseTree, error) {
	// Get the start rule from grammar
	rule := p.grammar.GetRule(startRule)
	if rule == nil {
		return nil, newRuleNotFoundError(startRule)
	}

	// Initialize parser state - always reset for each parse
	p.input = input
	p.pos = 0
	p.line = 1
	p.col = 1

	// Reset furthest position tracking
	p.furthestPos = 0
	p.furthestLine = 1
	p.furthestCol = 1
	p.furthestError = nil
	p.furthestRule = ""

	// Parse starting from the rule
	node, err := p.parseRule(startRule)
	if err != nil {
		return nil, err
	}

	// Ensure we consumed all input
	if p.pos < len(p.input) {
		remaining := p.input[p.pos:]
		if len(remaining) > 20 {
			remaining = remaining[:20] + "..."
		}

		// Build detailed error message including furthest position info
		details := fmt.Sprintf("unexpected input at line %d, col %d (pos %d/%d): %q", p.line, p.col, p.pos, len(p.input), remaining)

		// If we have info about why parsing stopped further ahead, include it
		if p.furthestPos > p.pos && p.furthestError != nil {
			details += fmt.Sprintf("\n  furthest parse attempt at line %d, col %d (pos %d) in rule %q: %v",
				p.furthestLine, p.furthestCol, p.furthestPos, p.furthestRule, p.furthestError)
		} else if p.furthestError != nil && p.furthestRule != "" {
			details += fmt.Sprintf("\n  last failed rule %q at line %d: %v",
				p.furthestRule, p.furthestLine, p.furthestError)
		}

		return nil, &ParseError{
			Type:    ErrorExpectedEOF,
			Pos:     p.pos,
			Line:    p.line,
			Col:     p.col,
			Details: details,
		}
	}

	return &ParseTree{Root: node, Input: input}, nil
}

// recordFurthestError records an error if it's at or past the furthest position seen
func (p *Parser) recordFurthestError(ruleName string, err error) {
	if p.pos >= p.furthestPos {
		p.furthestPos = p.pos
		p.furthestLine = p.line
		p.furthestCol = p.col
		p.furthestError = err
		p.furthestRule = ruleName
	}
}

// parseRule parses input according to a named rule
func (p *Parser) parseRule(ruleName string) (*Node, error) {
	rule := p.grammar.GetRule(ruleName)
	if rule == nil {
		return nil, newRuleNotFoundError(ruleName)
	}

	// Save position for this rule
	line := p.line
	col := p.col
	start := p.pos

	// Debug output
	preview := ""
	if p.pos < len(p.input) {
		end := p.pos + 20
		if end > len(p.input) {
			end = len(p.input)
		}
		preview = strings.ReplaceAll(p.input[p.pos:end], "\n", "\\n")
		if end < len(p.input) {
			preview += "..."
		}
	}
	p.debugf("Trying rule %s at pos %d: %q", ruleName, p.pos, preview)
	p.depth++

	// Parse the rule's expression first
	children, err := p.parseExpression(rule.Expression)
	p.depth--
	if err != nil {
		p.debugf("Rule %s failed: %v", ruleName, err)
		p.recordFurthestError(ruleName, err)
		return nil, wrapRuleError(ruleName, err)
	}
	p.debugf("Rule %s succeeded", ruleName)

	// Check if this is an expression pass-through node with a single non-terminal child
	if p.shouldFlatten(ruleName, children) {
		return children[0], nil
	}

	// Check if this rule's expression is entirely hidden
	// In that case, we should not create a node for the rule itself
	if _, isHidden := rule.Expression.(*ebnf.Hidden); isHidden && len(children) == 0 {
		// Return nil to indicate this rule matched but produced no node
		// This is different from an error - we successfully parsed, just no AST node
		return nil, nil
	}

	// Otherwise create the node normally
	node := &Node{
		Rule:     ruleName,
		Line:     line,
		Column:   col,
		Start:    start,
		End:      p.pos,
		Children: children,
	}

	return node, nil
}

// shouldFlatten returns true if this rule should be flattened to its child
func (p *Parser) shouldFlatten(ruleName string, children []*Node) bool {
	// Only flatten if:
	// 1. It's an expression rule
	// 2. It has exactly one child
	// 3. That child is a non-terminal (has a Rule name)
	return p.exprRules[ruleName] && len(children) == 1 && children[0].Rule != ""
}

// parseExpression parses an EBNF expression and returns the resulting nodes
func (p *Parser) parseExpression(expr ebnf.Expression) ([]*Node, error) {
	switch e := expr.(type) {
	case *ebnf.Terminal:
		return p.parseTerminal(e)
	case *ebnf.NonTerminal:
		return p.parseNonTerminal(e)
	case *ebnf.Sequence:
		return p.parseSequence(e)
	case *ebnf.Choice:
		return p.parseChoice(e)
	case *ebnf.OrderedChoice:
		return p.parseOrderedChoice(e)
	case *ebnf.Optional:
		return p.parseOptional(e)
	case *ebnf.Repetition:
		return p.parseRepetition(e)
	case *ebnf.Group:
		return p.parseExpression(e.Expr)
	case *ebnf.CharClass:
		return p.parseCharClass(e)
	case *ebnf.OneOrMore:
		return p.parseOneOrMore(e)
	case *ebnf.Hidden:
		// Parse the hidden expression but don't return any nodes
		_, err := p.parseExpression(e.Expr)
		return []*Node{}, err
	case *ebnf.Regex:
		return p.parseRegex(e)
	case *ebnf.Predicate:
		return p.parsePredicate(e)
	case *ebnf.PositiveLookahead:
		return p.parsePositiveLookahead(e)
	default:
		return nil, newUnknownExpressionError(fmt.Sprintf("%T", expr))
	}
}

// parseCharClass matches a single character from a character class
func (p *Parser) parseCharClass(cc *ebnf.CharClass) ([]*Node, error) {
	if p.pos >= len(p.input) {
		return nil, newUnexpectedEOFError("character from class", p.line, p.col)
	}

	ch := rune(p.input[p.pos])
	matched := false

	// Check single characters
	for _, c := range cc.Chars {
		if ch == c {
			matched = true
			break
		}
	}

	// Check ranges
	if !matched {
		for _, r := range cc.Ranges {
			if ch >= r.From && ch <= r.To {
				matched = true
				break
			}
		}
	}

	// Apply negation
	if cc.Negated {
		matched = !matched
	}

	if !matched {
		return nil, newCharClassMismatchError(string(ch), p.line, p.col)
	}

	// Create node
	node := &Node{
		Value:  string(ch),
		Line:   p.line,
		Column: p.col,
		Start:  p.pos,
		End:    p.pos + 1,
	}

	// Advance position
	if ch == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	p.pos++

	return []*Node{node}, nil
}

// parseTerminal matches a terminal string
func (p *Parser) parseTerminal(term *ebnf.Terminal) ([]*Node, error) {
	if p.pos >= len(p.input) {
		return nil, newUnexpectedEOFError(fmt.Sprintf("%q", term.Value), p.line, p.col)
	}

	// The terminal value from EBNF already has escape sequences properly interpreted
	// (e.g., "\n" is already a newline character, not the two chars '\' and 'n')
	termValue := term.Value

	// Determine if case-insensitive matching should be used
	// Per-terminal 'i' suffix takes precedence, otherwise use global setting
	caseInsensitive := term.CaseInsensitive || p.caseInsensitive

	// Check if the terminal matches at current position
	var matched bool
	if caseInsensitive {
		// Case-insensitive matching using EqualFold (faster than regex)
		if len(p.input)-p.pos >= len(termValue) {
			matched = strings.EqualFold(p.input[p.pos:p.pos+len(termValue)], termValue)
		}
	} else {
		// Case-sensitive matching
		matched = strings.HasPrefix(p.input[p.pos:], termValue)
	}

	if !matched {
		// For better error messages, show what we got
		preview := p.input[p.pos:]
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		return nil, newExpectedTerminalError(termValue, preview, p.line, p.col)
	}

	// Get the actual matched text from input (preserves original case)
	matchedValue := p.input[p.pos : p.pos+len(termValue)]

	// Create node for this terminal
	node := &Node{
		Value:  matchedValue,
		Line:   p.line,
		Column: p.col,
		Start:  p.pos,
		End:    p.pos + len(termValue),
	}

	// Advance position
	for i := 0; i < len(termValue); i++ {
		if p.input[p.pos] == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
		p.pos++
	}

	// Only return node if not hidden
	if term.Hidden {
		return []*Node{}, nil
	}
	return []*Node{node}, nil
}

// parseRegex matches input against a regular expression pattern
func (p *Parser) parseRegex(regex *ebnf.Regex) ([]*Node, error) {
	if p.pos >= len(p.input) {
		return nil, newUnexpectedEOFError(fmt.Sprintf("pattern %q", regex.Pattern), p.line, p.col)
	}

	// Get or compile the regex (with caching)
	cacheKey := "^" + regex.Pattern
	re, ok := p.regexCache[cacheKey]
	if !ok {
		// Compile the regex with anchoring to match from the start
		var err error
		re, err = regexp.Compile(cacheKey)
		if err != nil {
			return nil, newInvalidRegexError(regex.Pattern, err)
		}
		// Cache the compiled regex
		p.regexCache[cacheKey] = re
	}

	// Find match at current position
	loc := re.FindStringIndex(p.input[p.pos:])
	if loc == nil || loc[0] != 0 {
		// No match at current position
		preview := p.input[p.pos:]
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		return nil, newRegexNoMatchError(regex.Pattern, preview, p.line, p.col)
	}

	// Extract the matched text
	matchedText := p.input[p.pos : p.pos+loc[1]]

	// Create node for this match
	node := &Node{
		Value:  matchedText,
		Line:   p.line,
		Column: p.col,
		Start:  p.pos,
		End:    p.pos + len(matchedText),
	}

	// Advance position, tracking line and column
	for i := 0; i < len(matchedText); i++ {
		if p.input[p.pos] == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
		p.pos++
	}

	// Only return node if not hidden
	if regex.Hidden {
		return []*Node{}, nil
	}
	return []*Node{node}, nil
}

// parsePredicate parses a negative lookahead (!expr)
func (p *Parser) parsePredicate(pred *ebnf.Predicate) ([]*Node, error) {
	// Save current position
	savedPos, savedLine, savedCol := p.savePosition()

	// Try to match the expression
	_, err := p.parseExpression(pred.Expr)

	// Restore position regardless of result
	p.restorePosition(savedPos, savedLine, savedCol)

	if err == nil {
		// If the expression matched, the negative lookahead fails
		return nil, newNegativeLookaheadError(p.line, p.col)
	}

	// Expression didn't match, negative lookahead succeeds
	return []*Node{}, nil
}

// parsePositiveLookahead parses a positive lookahead (&expr)
func (p *Parser) parsePositiveLookahead(pos *ebnf.PositiveLookahead) ([]*Node, error) {
	// Save current position
	savedPos, savedLine, savedCol := p.savePosition()

	// Try to match the expression
	_, err := p.parseExpression(pos.Expr)

	// Restore position regardless of result
	p.restorePosition(savedPos, savedLine, savedCol)

	if err != nil {
		// If the expression didn't match, the positive lookahead fails
		return nil, newPositiveLookaheadError(p.line, p.col, err)
	}

	// Expression matched, positive lookahead succeeds
	return []*Node{}, nil
}

// parseNonTerminal parses a reference to another rule
func (p *Parser) parseNonTerminal(nt *ebnf.NonTerminal) ([]*Node, error) {
	rule := p.grammar.GetRule(nt.Name)
	if rule == nil {
		return nil, newRuleNotFoundError(nt.Name)
	}

	node, err := p.parseRule(nt.Name)
	if err != nil {
		return nil, err
	}

	// If the rule returned nil, it means it matched but produced no node
	// (e.g., the rule's expression was entirely hidden)
	if node == nil {
		return []*Node{}, nil
	}

	// Check if this non-terminal reference is hidden (like <digit>)
	if nt.Hidden {
		return []*Node{}, nil
	}

	// Check if the rule definition itself is hidden (like <digit> = ...)
	// In that case, return the children directly without the rule wrapper
	if rule.Hidden {
		return node.Children, nil
	}

	return []*Node{node}, nil
}

// parseSequence parses a sequence of expressions
func (p *Parser) parseSequence(seq *ebnf.Sequence) ([]*Node, error) {
	var result []*Node

	for _, elem := range seq.Elements {
		nodes, err := p.parseExpression(elem)
		if err != nil {
			return nil, err
		}
		result = append(result, nodes...)
	}

	return result, nil
}

// savePosition saves the current parser position
func (p *Parser) savePosition() (int, int, int) {
	p.debugf("SAVE POSITION")
	return p.pos, p.line, p.col
}

// restorePosition restores a saved parser position
func (p *Parser) restorePosition(pos, line, col int) {
	p.debugf("RESTORE POSITION from [pos=%d,line=%d,col=%d]", pos, line, col)
	p.pos = pos
	p.line = line
	p.col = col
}

// parseChoice tries each alternative until one succeeds
func (p *Parser) parseChoice(choice *ebnf.Choice) ([]*Node, error) {
	var lastErr error

	// Save current position for backtracking
	savedPos, savedLine, savedCol := p.savePosition()

	p.debugf("Trying %d alternatives", len(choice.Alternatives))
	p.depth++

	for i, alt := range choice.Alternatives {
		p.debugf("Alternative %d", i+1)
		p.depth++
		// Try this alternative
		nodes, err := p.parseExpression(alt)
		p.depth--
		if err == nil {
			p.depth--
			p.debugf("Alternative %d succeeded", i+1)
			return nodes, nil
		}

		// Failed, restore position and try next
		p.debugf("Alternative %d failed: %v", i+1, err)
		lastErr = err
		// Don't allocate error wrapping - just track the last error
		p.restorePosition(savedPos, savedLine, savedCol)
	}
	p.depth--

	// Return error only after all alternatives fail
	return nil, newNoAltMatchedError(len(choice.Alternatives), lastErr)
}

// parseOrderedChoice tries each alternative in order and returns the first that succeeds
// This is the PEG-style ordered choice (/) - no ambiguity, first match wins
func (p *Parser) parseOrderedChoice(choice *ebnf.OrderedChoice) ([]*Node, error) {
	var lastErr error

	// Save current position for backtracking
	savedPos, savedLine, savedCol := p.savePosition()

	for _, alt := range choice.Alternatives {
		// Try this alternative
		nodes, err := p.parseExpression(alt)
		if err == nil {
			return nodes, nil
		}

		// Failed, restore position and try next
		lastErr = err
		p.restorePosition(savedPos, savedLine, savedCol)
	}

	// Return the last error (simpler for ordered choice)
	return nil, newNoAltMatchedError(len(choice.Alternatives), lastErr)
}

// parseOptional tries to parse the expression, returns empty if it fails
func (p *Parser) parseOptional(opt *ebnf.Optional) ([]*Node, error) {
	// Save current position
	savedPos, savedLine, savedCol := p.savePosition()

	nodes, err := p.parseExpression(opt.Expr)
	if err == nil {
		return nodes, nil
	}

	// Failed, restore position and return empty
	p.restorePosition(savedPos, savedLine, savedCol)
	return []*Node{}, nil
}

// parseRepetition parses zero or more (*)
func (p *Parser) parseRepetition(rep *ebnf.Repetition) ([]*Node, error) {
	// Special handling for character class repetitions that should be consolidated
	if p.isCharacterClassExpression(rep.Expr) {
		return p.parseConsolidatedRepetition(rep.Expr, false)
	}

	var result []*Node

	for {
		// Save position before trying
		savedPos, savedLine, savedCol := p.savePosition()

		nodes, err := p.parseExpression(rep.Expr)
		if err != nil {
			// Restore position
			p.restorePosition(savedPos, savedLine, savedCol)
			break
		}

		// Check if we actually consumed input even if no nodes were returned
		// This handles the case of hidden expressions that consume input
		// but don't produce nodes
		if p.pos == savedPos {
			// No progress made, stop to avoid infinite loop
			break
		}

		result = append(result, nodes...)
	}

	return result, nil
}

// isHiddenExpression checks if an expression is hidden or leads to hidden content
func (p *Parser) isHiddenExpression(expr ebnf.Expression) bool {
	switch e := expr.(type) {
	case *ebnf.Hidden:
		return true
	case *ebnf.NonTerminal:
		// Check if the non-terminal reference itself is hidden
		if e.Hidden {
			return true
		}
		// Check if the non-terminal rule has a hidden expression
		if rule := p.grammar.GetRule(e.Name); rule != nil {
			// Recursively check if the rule's expression is hidden
			return p.isHiddenExpression(rule.Expression)
		}
	}
	return false
}

// isCharacterClassExpression checks if an expression is a character class
func (p *Parser) isCharacterClassExpression(expr ebnf.Expression) bool {
	switch e := expr.(type) {
	case *ebnf.CharClass:
		return true
	case *ebnf.NonTerminal:
		// Check if the non-terminal rule resolves to a character class
		if rule := p.grammar.GetRule(e.Name); rule != nil {
			return p.isCharacterClassExpression(rule.Expression)
		}
	}
	return false
}

// parseConsolidatedRepetition handles character class repetitions
// Creates a single consolidated value node from multiple matches
func (p *Parser) parseConsolidatedRepetition(expr ebnf.Expression, requireOne bool) ([]*Node, error) {
	startPos := p.pos
	startLine := p.line
	startCol := p.col
	count := 0

	for {
		savedPos, savedLine, savedCol := p.savePosition()

		_, err := p.parseExpression(expr)
		if err != nil {
			p.restorePosition(savedPos, savedLine, savedCol)
			break
		}

		// Check if we made progress
		if p.pos == savedPos {
			break
		}
		count++
	}

	if requireOne && count == 0 {
		return nil, newExpectedAtLeastOneError()
	}

	// Create a single consolidated value node from the matched text
	if p.pos > startPos {
		node := &Node{
			Value:  p.input[startPos:p.pos],
			Line:   startLine,
			Column: startCol,
			Start:  startPos,
			End:    p.pos,
		}
		return []*Node{node}, nil
	}

	return []*Node{}, nil
}

// parseOneOrMore parses one or more (+)
func (p *Parser) parseOneOrMore(rep *ebnf.OneOrMore) ([]*Node, error) {
	// Special handling for character class repetitions that should be consolidated
	if p.isCharacterClassExpression(rep.Expr) {
		return p.parseConsolidatedRepetition(rep.Expr, true)
	}

	var result []*Node
	count := 0

	for {
		// Save position before trying
		savedPos, savedLine, savedCol := p.savePosition()

		nodes, err := p.parseExpression(rep.Expr)
		if err != nil {
			// Restore position
			p.restorePosition(savedPos, savedLine, savedCol)
			break
		}

		// Check if we actually consumed input even if no nodes were returned
		// This handles the case of hidden expressions
		if p.pos > savedPos {
			count++
		}

		result = append(result, nodes...)
	}

	// Check minimum count
	if count == 0 {
		return nil, newExpectedAtLeastOneError()
	}

	return result, nil
}
