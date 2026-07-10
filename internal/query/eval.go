package query

import "fmt"

// Evaluate walks node against record and returns whether the record
// satisfies the filter expression. record is a flattened row from
// query_store -- field names map directly to JSON keys.
func Evaluate(node Node, record map[string]any) (bool, error) {
	switch n := node.(type) {
	case ComparisonNode:
		return evalComparison(n, record)
	case AndNode:
		left, err := Evaluate(n.Left, record)
		if err != nil {
			return false, err
		}
		if !left {
			return false, nil // short-circuit: right side can't change the outcome
		}
		return Evaluate(n.Right, record)
	case OrNode:
		left, err := Evaluate(n.Left, record)
		if err != nil {
			return false, err
		}
		if left {
			return true, nil // short-circuit
		}
		return Evaluate(n.Right, record)
	default:
		return false, fmt.Errorf("unknown AST node type %T", node)
	}
}

func evalComparison(n ComparisonNode, record map[string]any) (bool, error) {
	fieldVal, ok := record[n.Field]
	if !ok {
		return false, nil // field absent on this record -- never matches
	}

	switch queryVal := n.Value.(type) {
	case string:
		fv, ok := fieldVal.(string)
		if !ok {
			return false, nil // type mismatch -- doesn't match, not an error
		}
		return compareStrings(fv, n.Operator, queryVal)

	case int64:
		fv, ok := toFloat64(fieldVal)
		if !ok {
			return false, nil
		}
		return compareNumbers(fv, n.Operator, float64(queryVal))

	default:
		return false, fmt.Errorf("unsupported literal type %T for field %q", n.Value, n.Field)
	}
}

func compareStrings(a string, op TokenType, b string) (bool, error) {
	switch op {
	case EQ:
		return a == b, nil
	case NEQ:
		return a != b, nil
	case LT:
		return a < b, nil
	case LTE:
		return a <= b, nil
	case GT:
		return a > b, nil
	case GTE:
		return a >= b, nil
	default:
		return false, fmt.Errorf("unsupported operator for string comparison")
	}
}

func compareNumbers(a float64, op TokenType, b float64) (bool, error) {
	switch op {
	case EQ:
		return a == b, nil
	case NEQ:
		return a != b, nil
	case LT:
		return a < b, nil
	case LTE:
		return a <= b, nil
	case GT:
		return a > b, nil
	case GTE:
		return a >= b, nil
	default:
		return false, fmt.Errorf("unsupported operator for numeric comparison")
	}
}

// toFloat64 normalizes JSON-decoded numeric values (always float64 after
// json.Unmarshal into `any`) and, defensively, native Go numeric types
// that might reach here some other way.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}
