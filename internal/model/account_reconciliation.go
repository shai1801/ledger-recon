package model

// AccountReconciliation is one row of the accounts_reconciliation report --
// each account's opening balance plus the net effect of all its
// post-adjustment ledger entries.
type AccountReconciliation struct {
	AccountID           string `json:"account_id"`
	CustomerID          string `json:"customer_id"`
	AccountType         string `json:"account_type"`
	Currency            string `json:"currency"`
	Status              string `json:"status"`
	OpeningBalanceCents int64  `json:"opening_balance_cents"`
	DeltaCents          int64  `json:"delta_cents"`
	ClosingBalanceCents int64  `json:"closing_balance_cents"`
}
