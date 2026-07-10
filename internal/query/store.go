package query

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// LoadStore reads a query_store JSONL file into memory, one record per line.
func LoadStore(path string) ([]map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening query store: %w", err)
	}
	defer f.Close()

	var rows []map[string]any
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var row map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("parsing query store line: %w", err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading query store: %w", err)
	}
	return rows, nil
}

// Execute filters rows down to the given table, combining the explicit
// query expression (if any) with resolved dynamic filters (if any) via
// AND, then truncates to limit. Original insertion order is preserved.
func Execute(rows []map[string]any, table string, expression string, filters map[string]any, limit int) ([]map[string]any, error) {
	var exprNode Node
	if expression != "" {
		parsed, err := NewParser(expression).Parse()
		if err != nil {
			return nil, fmt.Errorf("parsing query: %w", err)
		}
		exprNode = parsed
	}

	filterNode, err := BuildFilterNode(filters)
	if err != nil {
		return nil, err
	}

	node := combineAnd(exprNode, filterNode)

	var results []map[string]any
	for _, row := range rows {
		if row["_table"] != table {
			continue
		}
		if node != nil {
			ok, err := Evaluate(node, row)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
		}
		results = append(results, row)
		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

func combineAnd(a, b Node) Node {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return AndNode{Left: a, Right: b}
}
