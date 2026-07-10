package report

import (
	"github.com/shai1801/ledger-recon/internal/match"
	"github.com/shai1801/ledger-recon/internal/model"
)

// BuildMatchedTransactions flattens match.MatchedPair results into
// report-ready records.
func BuildMatchedTransactions(matched []match.MatchedPair) []model.MatchedTransaction {
	records := make([]model.MatchedTransaction, 0, len(matched))
	for _, m := range matched {
		records = append(records, model.MatchedTransaction{
			EntryID:               m.Entry.EntryID,
			SettlementID:          m.Settlement.SettlementID,
			AccountID:             m.Entry.AccountID,
			Currency:              m.Entry.Currency,
			EntryAmountCents:      m.Entry.AmountCents,
			SettlementAmountCents: m.Settlement.AmountCents,
			AmountDiffCents:       m.AmountDiffCents,
		})
	}
	return records
}
