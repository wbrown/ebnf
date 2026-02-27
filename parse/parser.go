package parse

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/wbrown/ebnf"
)

// parseState holds mutable per-parse state. A new parseState is created
// for each Parse() call, making Parser safe for concurrent use.
type parseState struct {
	input         string
	pos           int
	line          int
	col           int
	depth         int
	debugLog      strings.Builder
	furthestPos   int
	furthestLine  int
	furthestCol   int
	furthestError error
	furthestRule  string
}

// Parser uses an EBNF grammar to parse input text.
// After construction, Parser is safe for concurrent use from multiple goroutines.
// Configure Debug, focusPos, and focusRange before concurrent use.
type Parser struct {
	grammar *ebnf.Grammar
	Debug   bool // Enable debug output

	// Focused debugging
	focusPos   int // Position to focus on (-1 = disabled)
	focusRange int // Range around focus position

	// Regex cache to avoid recompiling the same patterns
	regexCache sync.Map // string -> *regexp.Regexp

	// Expression rules that should be flattened (cached to avoid map allocation)
	exprRules map[string]bool

	// Case-insensitive matching (global default)
	caseInsensitive bool

	// Last debug log — captured at end of each Parse() call
	lastDebugMu  sync.Mutex
	lastDebugLog string
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
		grammar:  grammar,
		focusPos: -1,
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
func (p *Parser) debugf(s *parseState, format string, args ...interface{}) {
	shouldDebug := p.Debug

	// Check if we're in focused debug range
	if p.focusPos >= 0 && !shouldDebug {
		if s.pos >= p.focusPos-p.focusRange && s.pos <= p.focusPos+p.focusRange {
			shouldDebug = true
		}
	}

	if shouldDebug {
		indent := strings.Repeat("  ", s.depth)
		posInfo := fmt.Sprintf("[pos=%d,line=%d,col=%d]", s.pos, s.line, s.col)
		msg := fmt.Sprintf("%s%s %s\n", indent, posInfo, fmt.Sprintf(format, args...))
		fmt.Print(msg)
		s.debugLog.WriteString(msg)
	}
}

// GetDebugLog returns the captured debug log from the most recent Parse() call.
// Under concurrent use, returns whichever parse completed last.
func (p *Parser) GetDebugLog() string {
	p.lastDebugMu.Lock()
	log := p.lastDebugLog
	p.lastDebugMu.Unlock()
	return log
}

// getCachedRegex returns a compiled regex from the cache, compiling and caching
// it on first access. Safe for concurrent use via sync.Map.
func (p *Parser) getCachedRegex(pattern string) (*regexp.Regexp, error) {
	if v, ok := p.regexCache.Load(pattern); ok {
		return v.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	actual, _ := p.regexCache.LoadOrStore(pattern, re)
	return actual.(*regexp.Regexp), nil
}

// Parse parses the input according to the grammar, starting with the given rule
func (p *Parser) Parse(input string, startRule string) (*ParseTree, error) {
	// Get the start rule from grammar
	rule := p.grammar.GetRule(startRule)
	if rule == nil {
		return nil, newRuleNotFoundError(startRule)
	}

	// Create per-parse mutable state
	s := &parseState{
		input:        input,
		line:         1,
		col:          1,
		furthestLine: 1,
		furthestCol:  1,
	}

	// Capture debug log at end of parse, regardless of outcome
	defer func() {
		p.lastDebugMu.Lock()
		p.lastDebugLog = s.debugLog.String()
		p.lastDebugMu.Unlock()
	}()

	// Parse starting from the rule
	node, err := p.parseRule(s, startRule)
	if err != nil {
		return nil, err
	}

	// Ensure we consumed all input
	if s.pos < len(s.input) {
		remaining := s.input[s.pos:]
		if len(remaining) > 20 {
			remaining = remaining[:20] + "..."
		}

		// Build detailed error message including furthest position info
		details := fmt.Sprintf("unexpected input at line %d, col %d (pos %d/%d): %q", s.line, s.col, s.pos, len(s.input), remaining)

		// If we have info about why parsing stopped further ahead, include it
		if s.furthestPos > s.pos && s.furthestError != nil {
			details += fmt.Sprintf("\n  furthest parse attempt at line %d, col %d (pos %d) in rule %q: %v",
				s.furthestLine, s.furthestCol, s.furthestPos, s.furthestRule, s.furthestError)
		} else if s.furthestError != nil && s.furthestRule != "" {
			details += fmt.Sprintf("\n  last failed rule %q at line %d: %v",
				s.furthestRule, s.furthestLine, s.furthestError)
		}

		return nil, &ParseError{
			Type:    ErrorExpectedEOF,
			Pos:     s.pos,
			Line:    s.line,
			Col:     s.col,
			Details: details,
		}
	}

	return &ParseTree{Root: node, Input: input}, nil
}

// recordFurthestError records an error if it's at or past the furthest position seen
func (p *Parser) recordFurthestError(s *parseState, ruleName string, err error) {
	if s.pos >= s.furthestPos {
		s.furthestPos = s.pos
		s.furthestLine = s.line
		s.furthestCol = s.col
		s.furthestError = err
		s.furthestRule = ruleName
	}
}

// parseRule parses input according to a named rule
func (p *Parser) parseRule(s *parseState, ruleName string) (*Node, error) {
	rule := p.grammar.GetRule(ruleName)
	if rule == nil {
		return nil, newRuleNotFoundError(ruleName)
	}

	// Save position for this rule
	line := s.line
	col := s.col
	start := s.pos

	// Debug output
	preview := ""
	if s.pos < len(s.input) {
		end := s.pos + 20
		if end > len(s.input) {
			end = len(s.input)
		}
		preview = strings.ReplaceAll(s.input[s.pos:end], "\n", "\\n")
		if end < len(s.input) {
			preview += "..."
		}
	}
	p.debugf(s, "Trying rule %s at pos %d: %q", ruleName, s.pos, preview)
	s.depth++

	// Parse the rule's expression first
	children, err := p.parseExpression(s, rule.Expression)
	s.depth--
	if err != nil {
		p.debugf(s, "Rule %s failed: %v", ruleName, err)
		p.recordFurthestError(s, ruleName, err)
		return nil, wrapRuleError(ruleName, err)
	}
	p.debugf(s, "Rule %s succeeded", ruleName)

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
		End:      s.pos,
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
func (p *Parser) parseExpression(s *parseState, expr ebnf.Expression) ([]*Node, error) {
	switch e := expr.(type) {
	case *ebnf.Terminal:
		return p.parseTerminal(s, e)
	case *ebnf.NonTerminal:
		return p.parseNonTerminal(s, e)
	case *ebnf.Sequence:
		return p.parseSequence(s, e)
	case *ebnf.Choice:
		return p.parseChoice(s, e)
	case *ebnf.OrderedChoice:
		return p.parseOrderedChoice(s, e)
	case *ebnf.Optional:
		return p.parseOptional(s, e)
	case *ebnf.Repetition:
		return p.parseRepetition(s, e)
	case *ebnf.Group:
		return p.parseExpression(s, e.Expr)
	case *ebnf.CharClass:
		return p.parseCharClass(s, e)
	case *ebnf.OneOrMore:
		return p.parseOneOrMore(s, e)
	case *ebnf.Hidden:
		// Parse the hidden expression but don't return any nodes
		_, err := p.parseExpression(s, e.Expr)
		return []*Node{}, err
	case *ebnf.Regex:
		return p.parseRegex(s, e)
	case *ebnf.Predicate:
		return p.parsePredicate(s, e)
	case *ebnf.PositiveLookahead:
		return p.parsePositiveLookahead(s, e)
	default:
		return nil, newUnknownExpressionError(fmt.Sprintf("%T", expr))
	}
}

// parseCharClass matches a single character from a character class
func (p *Parser) parseCharClass(s *parseState, cc *ebnf.CharClass) ([]*Node, error) {
	if s.pos >= len(s.input) {
		return nil, newUnexpectedEOFError("character from class", s.line, s.col)
	}

	ch := rune(s.input[s.pos])
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
		return nil, newCharClassMismatchError(string(ch), s.line, s.col)
	}

	// Create node
	node := &Node{
		Value:  string(ch),
		Line:   s.line,
		Column: s.col,
		Start:  s.pos,
		End:    s.pos + 1,
	}

	// Advance position
	if ch == '\n' {
		s.line++
		s.col = 1
	} else {
		s.col++
	}
	s.pos++

	return []*Node{node}, nil
}

// parseTerminal matches a terminal string
func (p *Parser) parseTerminal(s *parseState, term *ebnf.Terminal) ([]*Node, error) {
	if s.pos >= len(s.input) {
		return nil, newUnexpectedEOFError(fmt.Sprintf("%q", term.Value), s.line, s.col)
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
		if len(s.input)-s.pos >= len(termValue) {
			matched = strings.EqualFold(s.input[s.pos:s.pos+len(termValue)], termValue)
		}
	} else {
		// Case-sensitive matching
		matched = strings.HasPrefix(s.input[s.pos:], termValue)
	}

	if !matched {
		// For better error messages, show what we got
		preview := s.input[s.pos:]
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		return nil, newExpectedTerminalError(termValue, preview, s.line, s.col)
	}

	// Get the actual matched text from input (preserves original case)
	matchedValue := s.input[s.pos : s.pos+len(termValue)]

	// Create node for this terminal
	node := &Node{
		Value:  matchedValue,
		Line:   s.line,
		Column: s.col,
		Start:  s.pos,
		End:    s.pos + len(termValue),
	}

	// Advance position
	for i := 0; i < len(termValue); i++ {
		if s.input[s.pos] == '\n' {
			s.line++
			s.col = 1
		} else {
			s.col++
		}
		s.pos++
	}

	// Only return node if not hidden
	if term.Hidden {
		return []*Node{}, nil
	}
	return []*Node{node}, nil
}

// parseRegex matches input against a regular expression pattern
func (p *Parser) parseRegex(s *parseState, regex *ebnf.Regex) ([]*Node, error) {
	if s.pos >= len(s.input) {
		return nil, newUnexpectedEOFError(fmt.Sprintf("pattern %q", regex.Pattern), s.line, s.col)
	}

	// Get or compile the regex (with thread-safe caching)
	cacheKey := "^" + regex.Pattern
	re, err := p.getCachedRegex(cacheKey)
	if err != nil {
		return nil, newInvalidRegexError(regex.Pattern, err)
	}

	// Find match at current position
	loc := re.FindStringIndex(s.input[s.pos:])
	if loc == nil || loc[0] != 0 {
		// No match at current position
		preview := s.input[s.pos:]
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		return nil, newRegexNoMatchError(regex.Pattern, preview, s.line, s.col)
	}

	// Extract the matched text
	matchedText := s.input[s.pos : s.pos+loc[1]]

	// Create node for this match
	node := &Node{
		Value:  matchedText,
		Line:   s.line,
		Column: s.col,
		Start:  s.pos,
		End:    s.pos + len(matchedText),
	}

	// Advance position, tracking line and column
	for i := 0; i < len(matchedText); i++ {
		if s.input[s.pos] == '\n' {
			s.line++
			s.col = 1
		} else {
			s.col++
		}
		s.pos++
	}

	// Only return node if not hidden
	if regex.Hidden {
		return []*Node{}, nil
	}
	return []*Node{node}, nil
}

// parsePredicate parses a negative lookahead (!expr)
func (p *Parser) parsePredicate(s *parseState, pred *ebnf.Predicate) ([]*Node, error) {
	// Save current position
	savedPos, savedLine, savedCol := p.savePosition(s)

	// Try to match the expression
	_, err := p.parseExpression(s, pred.Expr)

	// Restore position regardless of result
	p.restorePosition(s, savedPos, savedLine, savedCol)

	if err == nil {
		// If the expression matched, the negative lookahead fails
		return nil, newNegativeLookaheadError(s.line, s.col)
	}

	// Expression didn't match, negative lookahead succeeds
	return []*Node{}, nil
}

// parsePositiveLookahead parses a positive lookahead (&expr)
func (p *Parser) parsePositiveLookahead(s *parseState, pos *ebnf.PositiveLookahead) ([]*Node, error) {
	// Save current position
	savedPos, savedLine, savedCol := p.savePosition(s)

	// Try to match the expression
	_, err := p.parseExpression(s, pos.Expr)

	// Restore position regardless of result
	p.restorePosition(s, savedPos, savedLine, savedCol)

	if err != nil {
		// If the expression didn't match, the positive lookahead fails
		return nil, newPositiveLookaheadError(s.line, s.col, err)
	}

	// Expression matched, positive lookahead succeeds
	return []*Node{}, nil
}

// parseNonTerminal parses a reference to another rule
func (p *Parser) parseNonTerminal(s *parseState, nt *ebnf.NonTerminal) ([]*Node, error) {
	rule := p.grammar.GetRule(nt.Name)
	if rule == nil {
		return nil, newRuleNotFoundError(nt.Name)
	}

	node, err := p.parseRule(s, nt.Name)
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
func (p *Parser) parseSequence(s *parseState, seq *ebnf.Sequence) ([]*Node, error) {
	var result []*Node

	for _, elem := range seq.Elements {
		nodes, err := p.parseExpression(s, elem)
		if err != nil {
			return nil, err
		}
		result = append(result, nodes...)
	}

	return result, nil
}

// savePosition saves the current parser position
func (p *Parser) savePosition(s *parseState) (int, int, int) {
	p.debugf(s, "SAVE POSITION")
	return s.pos, s.line, s.col
}

// restorePosition restores a saved parser position
func (p *Parser) restorePosition(s *parseState, pos, line, col int) {
	p.debugf(s, "RESTORE POSITION from [pos=%d,line=%d,col=%d]", pos, line, col)
	s.pos = pos
	s.line = line
	s.col = col
}

// parseChoice tries each alternative until one succeeds
func (p *Parser) parseChoice(s *parseState, choice *ebnf.Choice) ([]*Node, error) {
	var lastErr error

	// Save current position for backtracking
	savedPos, savedLine, savedCol := p.savePosition(s)

	p.debugf(s, "Trying %d alternatives", len(choice.Alternatives))
	s.depth++

	for i, alt := range choice.Alternatives {
		p.debugf(s, "Alternative %d", i+1)
		s.depth++
		// Try this alternative
		nodes, err := p.parseExpression(s, alt)
		s.depth--
		if err == nil {
			s.depth--
			p.debugf(s, "Alternative %d succeeded", i+1)
			return nodes, nil
		}

		// Failed, restore position and try next
		p.debugf(s, "Alternative %d failed: %v", i+1, err)
		lastErr = err
		// Don't allocate error wrapping - just track the last error
		p.restorePosition(s, savedPos, savedLine, savedCol)
	}
	s.depth--

	// Return error only after all alternatives fail
	return nil, newNoAltMatchedError(len(choice.Alternatives), lastErr)
}

// parseOrderedChoice tries each alternative in order and returns the first that succeeds
// This is the PEG-style ordered choice (/) - no ambiguity, first match wins
func (p *Parser) parseOrderedChoice(s *parseState, choice *ebnf.OrderedChoice) ([]*Node, error) {
	var lastErr error

	// Save current position for backtracking
	savedPos, savedLine, savedCol := p.savePosition(s)

	for _, alt := range choice.Alternatives {
		// Try this alternative
		nodes, err := p.parseExpression(s, alt)
		if err == nil {
			return nodes, nil
		}

		// Failed, restore position and try next
		lastErr = err
		p.restorePosition(s, savedPos, savedLine, savedCol)
	}

	// Return the last error (simpler for ordered choice)
	return nil, newNoAltMatchedError(len(choice.Alternatives), lastErr)
}

// parseOptional tries to parse the expression, returns empty if it fails
func (p *Parser) parseOptional(s *parseState, opt *ebnf.Optional) ([]*Node, error) {
	// Save current position
	savedPos, savedLine, savedCol := p.savePosition(s)

	nodes, err := p.parseExpression(s, opt.Expr)
	if err == nil {
		return nodes, nil
	}

	// Failed, restore position and return empty
	p.restorePosition(s, savedPos, savedLine, savedCol)
	return []*Node{}, nil
}

// parseRepetition parses zero or more (*)
func (p *Parser) parseRepetition(s *parseState, rep *ebnf.Repetition) ([]*Node, error) {
	// Special handling for character class repetitions that should be consolidated
	if p.isCharacterClassExpression(rep.Expr) {
		return p.parseConsolidatedRepetition(s, rep.Expr, false)
	}

	var result []*Node

	for {
		// Save position before trying
		savedPos, savedLine, savedCol := p.savePosition(s)

		nodes, err := p.parseExpression(s, rep.Expr)
		if err != nil {
			// Restore position
			p.restorePosition(s, savedPos, savedLine, savedCol)
			break
		}

		// Check if we actually consumed input even if no nodes were returned
		// This handles the case of hidden expressions that consume input
		// but don't produce nodes
		if s.pos == savedPos {
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
func (p *Parser) parseConsolidatedRepetition(s *parseState, expr ebnf.Expression, requireOne bool) ([]*Node, error) {
	startPos := s.pos
	startLine := s.line
	startCol := s.col
	count := 0

	for {
		savedPos, savedLine, savedCol := p.savePosition(s)

		_, err := p.parseExpression(s, expr)
		if err != nil {
			p.restorePosition(s, savedPos, savedLine, savedCol)
			break
		}

		// Check if we made progress
		if s.pos == savedPos {
			break
		}
		count++
	}

	if requireOne && count == 0 {
		return nil, newExpectedAtLeastOneError()
	}

	// Create a single consolidated value node from the matched text
	if s.pos > startPos {
		node := &Node{
			Value:  s.input[startPos:s.pos],
			Line:   startLine,
			Column: startCol,
			Start:  startPos,
			End:    s.pos,
		}
		return []*Node{node}, nil
	}

	return []*Node{}, nil
}

// parseOneOrMore parses one or more (+)
func (p *Parser) parseOneOrMore(s *parseState, rep *ebnf.OneOrMore) ([]*Node, error) {
	// Special handling for character class repetitions that should be consolidated
	if p.isCharacterClassExpression(rep.Expr) {
		return p.parseConsolidatedRepetition(s, rep.Expr, true)
	}

	var result []*Node
	count := 0

	for {
		// Save position before trying
		savedPos, savedLine, savedCol := p.savePosition(s)

		nodes, err := p.parseExpression(s, rep.Expr)
		if err != nil {
			// Restore position
			p.restorePosition(s, savedPos, savedLine, savedCol)
			break
		}

		// Check if we actually consumed input even if no nodes were returned
		// This handles the case of hidden expressions
		if s.pos > savedPos {
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
