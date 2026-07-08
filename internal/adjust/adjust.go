package adjust

import (
	"fmt"

	"github.com/shai1801/ledger-recon/internal/model"
)

func Apply(entries []model.NormalizedEntry, adjustments []model.Adjustment) ([]model.NormalizedEntry, []model.Exceptions) {
	byEntryID := make(map[string]int, len(entries))
	result := make([]model.NormalizedEntry, len(entries))
	copy(result, entries)
	for i, e := range result {
		byEntryID[e.EntryID] = i
	}

	var exceptions []model.Exceptions
	var synthesized []model.NormalizedEntry

	for _, adj := range adjustments {
		idx, ok := byEntryID[adj.ReferenceEntryID]
		if !ok {
			exceptions = append(exceptions, model.Exceptions{
				SourceFile:    "adjustments.jsonl",
				RecordID:      adj.AdjustmentID,
				ExceptionType: "missing_reference",
				Severity:      "error",
				Reason:        fmt.Sprintf("reference_entry_id %q not found among valid ledger entries", adj.ReferenceEntryID),
			})
			continue
		}
		ref := result[idx]

		switch adj.AdjustmentType {
		case model.Reversal:
			rev := buildReversal(ref, adj)
			synthesized = append(synthesized, rev)
			result[idx].AppliedAdjustmentIDs = append(result[idx].AppliedAdjustmentIDs, adj.AdjustmentID)

		case model.Correction:
			if adj.AmountCents == nil {
				exceptions = append(exceptions, model.Exceptions{
					SourceFile:    "adjustments.jsonl",
					RecordID:      adj.AdjustmentID,
					ExceptionType: "invalid_adjustment",
					Severity:      "error",
					Reason:        "CORRECTION is missing amount_cents",
				})
				continue
			}

			if adj.Currency == nil || *adj.Currency != ref.Currency {
				got := "(missing)"
				if adj.Currency != nil {
					got = *adj.Currency
				}
				exceptions = append(exceptions, model.Exceptions{
					SourceFile:    "adjustments.jsonl",
					RecordID:      adj.AdjustmentID,
					ExceptionType: "currency_mismatch",
					Severity:      "error",
					Reason:        fmt.Sprintf("CORRECTION currency %q does not match referenced entry currency %q", got, ref.Currency),
				})
				continue
			}

			signed := *adj.AmountCents
			if ref.Direction == model.Debit {
				signed = -signed
			}
			result[idx].AmountCents = signed
			result[idx].AppliedAdjustmentIDs = append(result[idx].AppliedAdjustmentIDs, adj.AdjustmentID)

		}

	}
	return append(result, synthesized...), exceptions
}

func buildReversal(ref model.NormalizedEntry, adj model.Adjustment) model.NormalizedEntry {
	amount := -ref.AmountCents
	direction := model.Credit

	if amount < 0 {
		direction = model.Debit
	}

	return model.NormalizedEntry{

		EntryID:              adj.AdjustmentID,
		AccountID:            ref.AccountID,
		TxnID:                ref.TxnID,
		SourceSystem:         ref.SourceSystem,
		Direction:            direction,
		AmountCents:          amount,
		Currency:             ref.Currency,
		EventTime:            adj.EventTime,
		PostedAt:             adj.EventTime,
		ShardID:              ref.ShardID,
		Description:          fmt.Sprintf("Reversal of %s: %s", ref.EntryID, adj.Reason),
		OriginalAmountCents:  amount,
		AppliedAdjustmentIDs: []string{adj.AdjustmentID},
	}
}
