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

// Execute filters rows down to the given table, applies the parsed filter
// expression (if non-empty), and truncates to limit. Original insertion
// order is preserved throughout -- since every source table is already
// deterministically ordered, no extra sort is needed to satisfy the
// "deterministic ordering" requirement.
func Execute(rows []map[string]any, table string, expression string, limit int) ([]map[string]any, error) {
	var node Node
	if expression != "" {
		parsed, err := NewParser(expression).Parse()
		if err != nil {
			return nil, fmt.Errorf("parsing query: %w", err)
		}
		node = parsed
	}

	var results []map[string]any
	for _, row := range rows {
		if row["_table"] != table {
			continue
		}
		if node != nil {
			match, err := Evaluate(node, row)
			if err != nil {
				return nil, err
			}
			if !match {
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
