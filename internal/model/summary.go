package model

// Summary matches the schema in samples/mini_expected/expected_summary.json
// exactly -- field names and JSON tags are ground truth, not inferred.
type Summary struct {
	InputAccounts              int    `json:"input_accounts"`
	InputLedgerRows            int    `json:"input_ledger_rows"`
	InputSettlements           int    `json:"input_settlements"`
	InputAdjustments           int    `json:"input_adjustments"`
	ValidLedgerRowsAfterDedupe int    `json:"valid_ledger_rows_after_dedupe"`
	DuplicateLedgerRows        int    `json:"duplicate_ledger_rows"`
	InvalidLedgerRows          int    `json:"invalid_ledger_rows"`
	AdjustmentsApplied         int    `json:"adjustments_applied"`
	NormalizedLedgerEntries    int    `json:"normalized_ledger_entries"`
	MatchedTransactions        int    `json:"matched_transactions"`
	UnmatchedLedger            int    `json:"unmatched_ledger"`
	UnmatchedSettlements       int    `json:"unmatched_settlements"`
	Notes                      string `json:"notes,omitempty"`
}
