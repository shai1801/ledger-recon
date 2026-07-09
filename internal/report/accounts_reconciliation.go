package report

import "github.com/shai1801/ledger-recon/internal/model"

// BuildAccountsReconciliation produces one record per account: opening
// balance from the account registry, delta_cents from summing every
// post-adjustment ledger entry belonging to that account, and the
// resulting closing balance. Every account appears, even with zero
// activity (delta_cents = 0) -- omitting inactive accounts would make
// filters like "delta_cents!=0" meaningless.
func BuildAccountsReconciliation(accounts []model.Account, entries []model.NormalizedEntry) []model.AccountReconciliation {
	deltaByAccount := make(map[string]int64, len(accounts))
	for _, e := range entries {
		deltaByAccount[e.AccountID] += e.AmountCents
	}

	records := make([]model.AccountReconciliation, 0, len(accounts))
	for _, a := range accounts {
		delta := deltaByAccount[a.AccountID]
		records = append(records, model.AccountReconciliation{
			AccountID:           a.AccountID,
			CustomerID:          a.CustomerID,
			AccountType:         a.AccountType,
			Currency:            a.Currency,
			Status:              a.Status,
			OpeningBalanceCents: a.OpeningBalanceCents,
			DeltaCents:          delta,
			ClosingBalanceCents: a.OpeningBalanceCents + delta,
		})
	}

	return records
}
