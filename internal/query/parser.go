package query

import (
	"fmt"
	"strconv"
)

// Parser performs recursive-descent parsing of a filter expression string
// into an AST, using a Lexer for tokenization. It looks one token ahead
// (peekToken) to decide how to proceed at each step.
type Parser struct {
	lexer     *Lexer
	curToken  Token
	peekToken Token
}

func NewParser(input string) *Parser {
	p := &Parser{lexer: NewLexer(input)}
	p.nextToken()
	p.nextToken() // prime both curToken and peekToken
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// Parse parses the full input and returns its AST root. An error is
// returned for any malformed expression, or if input remains after a
// complete expression was parsed (e.g. a stray trailing token).
func (p *Parser) Parse() (Node, error) {
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.curToken.Type != EOF {
		return nil, fmt.Errorf("unexpected token %q at position %d", p.curToken.Literal, p.curToken.Pos)
	}
	return node, nil
}

// parseOr handles the lowest-precedence level: OR binds loosest, so
// "a AND b OR c AND d" parses as "(a AND b) OR (c AND d)".
func (p *Parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.curToken.Type == OR {
		p.nextToken()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = OrNode{Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAnd() (Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.curToken.Type == AND {
		p.nextToken()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = AndNode{Left: left, Right: right}
	}
	return left, nil
}

// parseComparison handles the highest-precedence level: either a
// parenthesized sub-expression, or a single field/operator/value leaf.
func (p *Parser) parseComparison() (Node, error) {
	if p.curToken.Type == LPAREN {
		p.nextToken()
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.curToken.Type != RPAREN {
			return nil, fmt.Errorf("expected ')' at position %d, got %q", p.curToken.Pos, p.curToken.Literal)
		}
		p.nextToken()
		return node, nil
	}

	if p.curToken.Type != IDENT {
		return nil, fmt.Errorf("expected field name at position %d, got %q", p.curToken.Pos, p.curToken.Literal)
	}
	field := p.curToken.Literal
	p.nextToken()

	operator, err := p.parseOperator()
	if err != nil {
		return nil, err
	}

	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	return ComparisonNode{Field: field, Operator: operator, Value: value}, nil
}

func (p *Parser) parseOperator() (TokenType, error) {
	switch p.curToken.Type {
	case EQ, NEQ, LT, LTE, GT, GTE:
		t := p.curToken.Type
		p.nextToken()
		return t, nil
	default:
		return ILLEGAL, fmt.Errorf("expected comparison operator at position %d, got %q", p.curToken.Pos, p.curToken.Literal)
	}
}

func (p *Parser) parseValue() (any, error) {
	switch p.curToken.Type {
	case STRING:
		v := p.curToken.Literal
		p.nextToken()
		return v, nil
	case NUMBER:
		n, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q at position %d: %v", p.curToken.Literal, p.curToken.Pos, err)
		}
		p.nextToken()
		return n, nil
	default:
		return nil, fmt.Errorf("expected string or number at position %d, got %q", p.curToken.Pos, p.curToken.Literal)
	}
}
