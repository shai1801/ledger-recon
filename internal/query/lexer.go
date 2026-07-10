package query

import (
	"fmt"
	"strings"
)

// Lexer tokenizes a filter string into a stream of Tokens, one at a time,
// for a recursive-descent parser to consume via repeated NextToken calls.
type Lexer struct {
	input        string
	position     int  // current byte position
	readPosition int  // next byte to read
	ch           byte // current byte under examination, 0 means EOF
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// NextToken returns the next token in the input, advancing the lexer's
// position. Returns a Token{Type: EOF} once the input is exhausted.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	startPos := l.position

	switch l.ch {
	case '=':
		l.readChar()
		return Token{Type: EQ, Literal: "=", Pos: startPos}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: NEQ, Literal: "!=", Pos: startPos}
		}
		ch := l.ch
		l.readChar()
		return Token{Type: ILLEGAL, Literal: string(ch), Pos: startPos}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: LTE, Literal: "<=", Pos: startPos}
		}
		l.readChar()
		return Token{Type: LT, Literal: "<", Pos: startPos}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: GTE, Literal: ">=", Pos: startPos}
		}
		l.readChar()
		return Token{Type: GT, Literal: ">", Pos: startPos}
	case '(':
		l.readChar()
		return Token{Type: LPAREN, Literal: "(", Pos: startPos}
	case ')':
		l.readChar()
		return Token{Type: RPAREN, Literal: ")", Pos: startPos}
	case '\'':
		s, err := l.readString()
		if err != nil {
			return Token{Type: ILLEGAL, Literal: err.Error(), Pos: startPos}
		}
		return Token{Type: STRING, Literal: s, Pos: startPos}
	case 0:
		return Token{Type: EOF, Literal: "", Pos: startPos}
	}

	if isLetter(l.ch) || l.ch == '_' {
		ident := l.readIdentifier()
		return Token{Type: lookupIdent(ident), Literal: ident, Pos: startPos}
	}
	if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
		num := l.readNumber()
		return Token{Type: NUMBER, Literal: num, Pos: startPos}
	}

	ch := l.ch
	l.readChar()
	return Token{Type: ILLEGAL, Literal: string(ch), Pos: startPos}
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readNumber() string {
	start := l.position
	if l.ch == '-' {
		l.readChar()
	}
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.position]
}

// readString reads a single-quoted string literal, e.g. 'USD'. Escaped
// quotes are not supported -- no field in this data model ever contains
// a quote character, so the grammar doesn't need to handle it.
func (l *Lexer) readString() (string, error) {
	l.readChar() // consume opening quote
	start := l.position
	for l.ch != '\'' && l.ch != 0 {
		l.readChar()
	}
	if l.ch == 0 {
		return "", fmt.Errorf("unterminated string literal")
	}
	s := l.input[start:l.position]
	l.readChar() // consume closing quote
	return s, nil
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// lookupIdent distinguishes the AND/OR keywords (matched case-insensitively,
// since the spec doesn't mandate case-sensitivity) from ordinary field
// name identifiers.
func lookupIdent(ident string) TokenType {
	switch strings.ToUpper(ident) {
	case "AND":
		return AND
	case "OR":
		return OR
	default:
		return IDENT
	}
}
