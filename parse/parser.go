package parse

import (
	"fmt"
	"github.com/wbrown/ebnf"
	"regexp"
	"strings"
)

// Parser uses an EBNF grammar to parse input text
type Parser struct {
	grammar *ebnf.Grammar
	input   string
	pos     int
	line    int
	col     int
	Debug   bool // Enable debug output
	depth   int  // Track recursion depth for debug indentation
	debugLog strings.Builder // Capture debug output

	// Focused debugging
	focusPos   int  // Position to focus on (-1 = disabled)
	focusRange int  // Range around focus position
}

// New creates a new parser with the given EBNF grammar
func New(grammar *ebnf.Grammar) *Parser {
	return &Parser{
		grammar:  grammar,
		focusPos: -1,
	}
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
		return nil, fmt.Errorf("start rule %q not found in grammar", startRule)
	}

	// Initialize parser state - always reset for each parse
	p.input = input
	p.pos = 0
	p.line = 1
	p.col = 1

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
		return nil, fmt.Errorf("unexpected input at line %d, col %d (pos %d/%d): %q",
			p.line, p.col, p.pos, len(p.input), remaining)
	}

	return &ParseTree{Root: node, Input: input}, nil
}

// parseRule parses input according to a named rule
func (p *Parser) parseRule(ruleName string) (*Node, error) {
	rule := p.grammar.GetRule(ruleName)
	if rule == nil {
		return nil, fmt.Errorf("rule %q not found", ruleName)
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
		return nil, fmt.Errorf("error parsing rule %s: %w", ruleName, err)
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
	// Expression rules that just pass through when they have a single non-terminal child
	exprRules := map[string]bool{
		"or_expr":     true,
		"and_expr":    true,
		"eq_expr":     true,
		"rel_expr":     true,
		"concat_expr": true,
		"add_expr":    true,
		"mult_expr":   true,
		"fair_expr":   true,
		"exp_expr":    true,
		"unary_expr":  true,
		"primary_expr": true,
	}

	// Only flatten if:
	// 1. It's an expression rule
	// 2. It has exactly one child
	// 3. That child is a non-terminal (has a Rule name)
	return exprRules[ruleName] && len(children) == 1 && children[0].Rule != ""
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
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

// parseCharClass matches a single character from a character class
func (p *Parser) parseCharClass(cc *ebnf.CharClass) ([]*Node, error) {
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected EOF at line %d col %d, expected character from class", p.line, p.col)
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
		return nil, fmt.Errorf("character %q at line %d col %d does not match character class", ch, p.line, p.col)
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
		return nil, fmt.Errorf("unexpected EOF at line %d col %d, expected %q", p.line, p.col, term.Value)
	}

	// The terminal value from EBNF already has escape sequences properly interpreted
	// (e.g., "\n" is already a newline character, not the two chars '\' and 'n')
	termValue := term.Value

	// Check if the terminal matches at current position
	if !strings.HasPrefix(p.input[p.pos:], termValue) {
		// For better error messages, show what we got
		preview := p.input[p.pos:]
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		return nil, fmt.Errorf("expected %q at line %d col %d, got %q",
			termValue, p.line, p.col, preview)
	}

	// Create node for this terminal
	node := &Node{
		Value:  termValue,
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
		return nil, fmt.Errorf("unexpected EOF at line %d col %d, expected pattern %q", p.line, p.col, regex.Pattern)
	}

	// Compile the regex with anchoring to match from the start
	re, err := regexp.Compile("^" + regex.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %v", regex.Pattern, err)
	}

	// Find match at current position
	loc := re.FindStringIndex(p.input[p.pos:])
	if loc == nil || loc[0] != 0 {
		// No match at current position
		preview := p.input[p.pos:]
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		return nil, fmt.Errorf("expected pattern %q at line %d col %d, got %q",
			regex.Pattern, p.line, p.col, preview)
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
		return nil, fmt.Errorf("negative lookahead failed at line %d col %d", p.line, p.col)
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
		return nil, fmt.Errorf("positive lookahead failed at line %d col %d: %v", p.line, p.col, err)
	}

	// Expression matched, positive lookahead succeeds
	return []*Node{}, nil
}

// parseNonTerminal parses a reference to another rule
func (p *Parser) parseNonTerminal(nt *ebnf.NonTerminal) ([]*Node, error) {
	node, err := p.parseRule(nt.Name)
	if err != nil {
		return nil, err
	}

	// If the rule returned nil, it means it matched but produced no node
	// (e.g., the rule's expression was entirely hidden)
	if node == nil {
		return []*Node{}, nil
	}

	// Check if this non-terminal reference is hidden
	if nt.Hidden {
		return []*Node{}, nil
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
	var errors []error

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
		errors = append(errors, fmt.Errorf("alt[%d]: %w", i, err))
		p.restorePosition(savedPos, savedLine, savedCol)
	}
	p.depth--

	// If we have multiple errors and they're all similar, just return the last one
	if len(errors) > 3 {
		return nil, fmt.Errorf("no alternative matched (tried %d): %w", len(errors), lastErr)
	}

	// Otherwise show all errors for debugging
	return nil, fmt.Errorf("no alternative matched: %v", errors)
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
	return nil, fmt.Errorf("no ordered alternative matched: %w", lastErr)
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
	// Special handling for repetitions that should be consolidated
	// This includes hidden expressions and character classes
	if p.isHiddenExpression(rep.Expr) || p.isCharacterClassExpression(rep.Expr) {
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

// parseConsolidatedRepetition handles repetitions that should be consolidated into a single text node
// This includes hidden expressions and character classes
func (p *Parser) parseConsolidatedRepetition(expr ebnf.Expression, requireOne bool) ([]*Node, error) {
	startPos := p.pos
	startLine := p.line
	startCol := p.col
	count := 0

	// fmt.Printf("parseHiddenRepetition: starting at pos=%d\n", p.pos)

	for {
		savedPos, savedLine, savedCol := p.savePosition()

		_, err := p.parseExpression(expr)
		if err != nil {
			// fmt.Printf("  iteration %d failed: %v\n", count, err)
			p.restorePosition(savedPos, savedLine, savedCol)
			break
		}

		// Check if we made progress
		if p.pos == savedPos {
			// fmt.Printf("  no progress at iteration %d\n", count)
			break
		}
		count++
		// fmt.Printf("  iteration %d succeeded, pos=%d->%d\n", count, savedPos, p.pos)
	}

	// fmt.Printf("parseHiddenRepetition: count=%d, requireOne=%v\n", count, requireOne)

	if requireOne && count == 0 {
		return nil, fmt.Errorf("expected at least one occurrence")
	}

	// Create a single text node with the matched content
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
	// Special handling for repetitions that should be consolidated
	// This includes hidden expressions and character classes
	if p.isHiddenExpression(rep.Expr) || p.isCharacterClassExpression(rep.Expr) {
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
		return nil, fmt.Errorf("expected at least one occurrence")
	}

	return result, nil
}
