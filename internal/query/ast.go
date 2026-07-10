package query

// Node is any node in a parsed filter expression's AST. It has no
// behavior of its own -- the marker method just restricts which types
// satisfy the interface, so a stray unrelated type can't be passed where
// a Node is expected.
type Node interface {
	node()
}

// ComparisonNode is a leaf: field <operator> value, e.g. currency='USD'.
// Value holds either a string or an int64 -- whichever the parser read
// from a STRING or NUMBER token.
type ComparisonNode struct {
	Field    string
	Operator TokenType // one of EQ, NEQ, LT, LTE, GT, GTE
	Value    any
}

func (ComparisonNode) node() {}

// AndNode and OrNode are internal (non-leaf) nodes combining two
// sub-expressions. Parentheses in the input are not represented directly
// in the AST -- they only affect how parsing groups nodes together.
type AndNode struct {
	Left, Right Node
}

func (AndNode) node() {}

type OrNode struct {
	Left, Right Node
}

func (OrNode) node() {}
