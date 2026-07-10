package query

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var suffixOperators = map[string]TokenType{
	"_eq":  EQ,
	"_ne":  NEQ,
	"_min": GTE,
	"_max": LTE,
	"_gt":  GT,
	"_gte": GTE,
	"_lt":  LT,
	"_lte": LTE,
}

// BuildFilterNode converts resolved "field_suffix": value entries (from
// configuration.json's filters section, merged with CLI --filter flags)
// into a single AST node, AND-combining every entry. Returns (nil, nil)
// for an empty map, meaning "no additional restriction".
func BuildFilterNode(filters map[string]any) (Node, error) {
	if len(filters) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic AST shape across runs

	var node Node
	for _, key := range keys {
		field, op, err := splitFieldSuffix(key)
		if err != nil {
			return nil, err
		}
		value, err := normalizeFilterValue(filters[key])
		if err != nil {
			return nil, err
		}
		cmp := ComparisonNode{Field: field, Operator: op, Value: value}
		if node == nil {
			node = cmp
		} else {
			node = AndNode{Left: node, Right: cmp}
		}
	}
	return node, nil
}

func splitFieldSuffix(key string) (field string, op TokenType, err error) {
	for suffix, tokenType := range suffixOperators {
		if strings.HasSuffix(key, suffix) {
			field = strings.TrimSuffix(key, suffix)
			if field == "" {
				return "", ILLEGAL, fmt.Errorf("invalid filter field %q: missing field name before suffix", key)
			}
			return field, tokenType, nil
		}
	}
	return "", ILLEGAL, fmt.Errorf("invalid filter field %q: must end in one of _eq, _ne, _min, _max, _gt, _gte, _lt, _lte", key)
}

// normalizeFilterValue handles values arriving from two different
// sources: CLI values are always raw strings; configuration.json values
// come back as float64 (JSON numbers) or string, per encoding/json's
// map[string]any decoding behavior.
func normalizeFilterValue(v any) (any, error) {
	switch val := v.(type) {
	case string:
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n, nil
		}
		return val, nil
	case float64:
		return int64(val), nil
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	default:
		return nil, fmt.Errorf("unsupported filter value type %T", v)
	}
}
