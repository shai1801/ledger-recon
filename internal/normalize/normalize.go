package normalize

import (
	"fmt"

	"github.com/shai1801/ledger-recon/internal/model"
)

//I have written this function to apply the business rule to entries that already have been validated by

func Normalize(entries []model.NormalizedEntry, accounts []model.Account) ([]model.NormalizedEntry, []model.Exceptions) {
	accountsByID := buildAccountIndex(accounts)
	deduped, exceptions := dedupeByEntryID(entries)

	var normalized []model.NormalizedEntry
	for _, e := range deduped {
		acc, ok := accountsByID[e.AccountID]
		if !ok {
			exceptions = append(exceptions, model.Exceptions{
				SourceFile:    "ledger_entries.csv",
				RecordID:      e.EntryID,
				ExceptionType: "unknown_account",
				Severity:      "error",
				Reason:        fmt.Sprintf("account_id %q not found in account registry", e.AccountID),
			})
			continue
		}
		if e.Currency != acc.Currency {
			exceptions = append(exceptions, model.Exceptions{
				SourceFile:    "ledger_entries.csv",
				RecordID:      e.EntryID,
				ExceptionType: "currency_mismatch",
				Severity:      "error",
				Reason: fmt.Sprintf("entry currency %q does not match account %q currency %q",
					e.Currency, e.AccountID, acc.Currency),
			})
			continue
		}
		normalized = append(normalized, applySign(e))
	}

	return normalized, exceptions
}

// dedupByEntryID keeps the first occurrence of each entry_id and turns
// every subsequent occurrence into a duplicate_entry exception.
func dedupeByEntryID(entries []model.NormalizedEntry) ([]model.NormalizedEntry, []model.Exceptions) {
	seen := make(map[string]bool, len(entries))
	var kept []model.NormalizedEntry
	var exceptions []model.Exceptions

	for _, e := range entries {
		if seen[e.EntryID] {
			exceptions = append(exceptions, model.Exceptions{
				SourceFile:    "ledger_entries.csv",
				RecordID:      e.EntryID,
				ExceptionType: "duplicate_entry",
				Severity:      "warning",
				Reason:        fmt.Sprintf("duplicate entry_id %q; first occurrence retained, this row skipped", e.EntryID),
			})
			continue
		}
		seen[e.EntryID] = true
		kept = append(kept, e)
	}
	return kept, exceptions
}

func applySign(e model.NormalizedEntry) model.NormalizedEntry {
	signed := e.AmountCents
	if e.Direction == model.Debit {
		signed = -signed
	}
	e.AmountCents = signed
	e.OriginalAmountCents = signed

	return e

}

func buildAccountIndex(accounts []model.Account) (index map[string]model.Account) {
	idx := make(map[string]model.Account, len(accounts))
	for _, a := range accounts {
		idx[a.AccountID] = a
	}
	return idx
}
