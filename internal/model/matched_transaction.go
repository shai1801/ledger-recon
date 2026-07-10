package model

// defining the flat matched transaction structure for reporting

type MatchedTransaction struct {
	EntryID               string `json:"entry_id"`
	SettlementID          string `json:"settlement_id"`
	AccountID             string `json:"account_id"`
	Currency              string `json:"currency"`
	EntryAmountCents      int64  `json:"entry_amount_cents"`
	SettlementAmountCents int64  `json:"settlement_amount_cents"`
	AmountDiffCents       int64  `json:"amount_diff_cents"`
}
