package adjust

import (
	"testing"
	"time"

	"github.com/shai1801/ledger-recon/internal/model"
)

func mkEntry(id, accountID string, direction model.Direction, amount int64, currency string) model.NormalizedEntry {
	return model.NormalizedEntry{
		EntryID:             id,
		AccountID:           accountID,
		TxnID:               "TXN-" + id,
		Direction:           direction,
		AmountCents:         amount,
		OriginalAmountCents: amount,
		Currency:            currency,
		PostedAt:            time.Date(2026, 1, 5, 9, 5, 0, 0, time.UTC),
	}
}

func ptr[T any](v T) *T { return &v }

func TestApply_MiniFixtureReversal(t *testing.T) {
	entries := []model.NormalizedEntry{
		mkEntry("LE-M4", "ACC-M2", model.Credit, 2000, "USD"),
	}
	adjustments := []model.Adjustment{
		{
			AdjustmentID:     "ADJ-M1",
			AdjustmentType:   model.Reversal,
			ReferenceEntryID: "LE-M4",
			EventTime:        time.Date(2026, 1, 5, 13, 0, 0, 0, time.UTC),
			Reason:           "mini reversal",
		},
	}

	result, exceptions := Apply(entries, adjustments)

	if len(exceptions) != 0 {
		t.Fatalf("expected 0 exceptions, got %d: %+v", len(exceptions), exceptions)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries (1 original + 1 synthetic), got %d", len(result))
	}

	var original, synthetic *model.NormalizedEntry
	for i := range result {
		switch result[i].EntryID {
		case "LE-M4":
			original = &result[i]
		case "ADJ-M1":
			synthetic = &result[i]
		}
	}

	if original == nil {
		t.Fatal("original entry LE-M4 not found in result")
	}
	if len(original.AppliedAdjustmentIDs) != 1 || original.AppliedAdjustmentIDs[0] != "ADJ-M1" {
		t.Errorf("original.AppliedAdjustmentIDs = %v, want [ADJ-M1]", original.AppliedAdjustmentIDs)
	}

	if synthetic == nil {
		t.Fatal("synthetic reversal entry ADJ-M1 not found in result")
	}
	if synthetic.AmountCents != -2000 {
		t.Errorf("synthetic.AmountCents = %d, want -2000", synthetic.AmountCents)
	}
	if synthetic.Direction != model.Debit {
		t.Errorf("synthetic.Direction = %q, want DEBIT", synthetic.Direction)
	}
	if synthetic.TxnID != "TXN-LE-M4" {
		t.Errorf("synthetic.TxnID = %q, want TXN-LE-M4 (inherited from referenced entry)", synthetic.TxnID)
	}
	if synthetic.AccountID != "ACC-M2" {
		t.Errorf("synthetic.AccountID = %q, want ACC-M2", synthetic.AccountID)
	}
}

func TestApply_CorrectionHappyPath(t *testing.T) {
	entries := []model.NormalizedEntry{
		mkEntry("LE-1", "ACC-1", model.Debit, -500, "USD"),
	}
	adjustments := []model.Adjustment{
		{
			AdjustmentID:     "ADJ-C-1",
			AdjustmentType:   model.Correction,
			ReferenceEntryID: "LE-1",
			AmountCents:      ptr(int64(700)), // magnitude, per spec
			Currency:         ptr("USD"),
		},
	}

	result, exceptions := Apply(entries, adjustments)

	if len(exceptions) != 0 {
		t.Fatalf("expected 0 exceptions, got %d: %+v", len(exceptions), exceptions)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	// LE-1's direction was DEBIT, so the corrected magnitude (700) should
	// become negative -- "retaining its original direction".
	if result[0].AmountCents != -700 {
		t.Errorf("AmountCents = %d, want -700", result[0].AmountCents)
	}
}

func TestApply_CorrectionMissingAmount(t *testing.T) {
	entries := []model.NormalizedEntry{mkEntry("LE-1", "ACC-1", model.Credit, 1000, "USD")}
	adjustments := []model.Adjustment{
		{AdjustmentID: "ADJ-1", AdjustmentType: model.Correction, ReferenceEntryID: "LE-1"},
	}

	result, exceptions := Apply(entries, adjustments)

	if len(exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
	}
	if exceptions[0].ExceptionType != "invalid_adjustment" {
		t.Errorf("exception type = %q, want invalid_adjustment", exceptions[0].ExceptionType)
	}
	// The original entry should be untouched -- correction was rejected.
	if result[0].AmountCents != 1000 {
		t.Errorf("AmountCents = %d, want unchanged 1000", result[0].AmountCents)
	}
}

func TestApply_CorrectionCurrencyMismatch(t *testing.T) {
	entries := []model.NormalizedEntry{mkEntry("LE-1", "ACC-1", model.Credit, 1000, "GBP")}
	adjustments := []model.Adjustment{
		{
			AdjustmentID:     "ADJ-1",
			AdjustmentType:   model.Correction,
			ReferenceEntryID: "LE-1",
			AmountCents:      ptr(int64(500)),
			Currency:         ptr("JPY"), // mismatched
		},
	}

	_, exceptions := Apply(entries, adjustments)

	if len(exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
	}
	if exceptions[0].ExceptionType != "currency_mismatch" {
		t.Errorf("exception type = %q, want currency_mismatch", exceptions[0].ExceptionType)
	}
}

func TestApply_MissingReference(t *testing.T) {
	entries := []model.NormalizedEntry{mkEntry("LE-1", "ACC-1", model.Credit, 1000, "USD")}
	adjustments := []model.Adjustment{
		{AdjustmentID: "ADJ-1", AdjustmentType: model.Reversal, ReferenceEntryID: "LE-DOES-NOT-EXIST"},
	}

	result, exceptions := Apply(entries, adjustments)

	if len(exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
	}
	if exceptions[0].ExceptionType != "missing_reference" {
		t.Errorf("exception type = %q, want missing_reference", exceptions[0].ExceptionType)
	}
	if len(result) != 1 {
		t.Errorf("expected result unchanged (1 entry, no synthesis), got %d", len(result))
	}
}
