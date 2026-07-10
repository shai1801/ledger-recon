package query

// TokenType identifies the kind of a single lexed token in a filter
// string like "currency='USD' AND delta_cents!=0".
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF

	IDENT  // currency, delta_cents, exception_type
	STRING // 'USD'
	NUMBER // 1000, -5

	EQ  // =
	NEQ // !=
	LT  //
	LTE // <=
	GT  // >
	GTE // >=

	AND
	OR

	LPAREN
	RPAREN
)

// Token is one lexed unit: its type, the literal text it came from, and
// its byte offset in the original input (for error messages).
type Token struct {
	Type    TokenType
	Literal string
	Pos     int
}
