package report

import (
	"encoding/json"
	"fmt"

	"github.com/shai1801/ledger-recon/internal/match"
	"github.com/shai1801/ledger-recon/internal/model"
)

// BuildQueryStore flattens every report table into a single ordered slice
// of generic rows, each tagged with "_table" so query.Execute can filter
// by --table before applying the query expression. Table names match the
// two worked examples in the assessment PDF (accounts, exceptions).
func BuildQueryStore(
	accounts []model.Account,
	entries []model.NormalizedEntry,
	matched []match.MatchedPair,
	unmatchedLedger []model.NormalizedEntry,
	unmatchedSettlements []model.BankSettlement,
	exceptions []model.Exceptions,
) ([]map[string]any, error) {
	var rows []map[string]any

	accountRows, err := toQueryRows("accounts", BuildAccountsReconciliation(accounts, entries))
	if err != nil {
		return nil, fmt.Errorf("flattening accounts: %w", err)
	}
	rows = append(rows, accountRows...)

	matchedRows, err := toQueryRows("matched_transactions", BuildMatchedTransactions(matched))
	if err != nil {
		return nil, fmt.Errorf("flattening matched_transactions: %w", err)
	}
	rows = append(rows, matchedRows...)

	unmatchedLedgerRows, err := toQueryRows("unmatched_ledger", unmatchedLedger)
	if err != nil {
		return nil, fmt.Errorf("flattening unmatched_ledger: %w", err)
	}
	rows = append(rows, unmatchedLedgerRows...)

	unmatchedSettlementRows, err := toQueryRows("unmatched_settlements", unmatchedSettlements)
	if err != nil {
		return nil, fmt.Errorf("flattening unmatched_settlements: %w", err)
	}
	rows = append(rows, unmatchedSettlementRows...)

	exceptionRows, err := toQueryRows("exceptions", exceptions)
	if err != nil {
		return nil, fmt.Errorf("flattening exceptions: %w", err)
	}
	rows = append(rows, exceptionRows...)

	return rows, nil
}

// toQueryRows converts a typed slice into generic map[string]any rows via
// a JSON round-trip -- reusing each struct's existing json tags instead of
// duplicating field lists -- and stamps every row with "_table".
func toQueryRows[T any](table string, items []T) ([]map[string]any, error) {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		var row map[string]any
		if err := json.Unmarshal(data, &row); err != nil {
			return nil, err
		}
		row["_table"] = table
		rows = append(rows, row)
	}
	return rows, nil
}
