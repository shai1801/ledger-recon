package normalize

import (
	"testing"
	"time"

	"github.com/shai1801/ledger-recon/internal/model"
)

func mkEntry(id, accountID string, direction model.Direction, amount int64, currency string) model.NormalizedEntry {
	return model.NormalizedEntry{
		EntryID:     id,
		AccountID:   accountID,
		TxnID:       "TXN-" + id,
		Direction:   direction,
		AmountCents: amount, // raw positive value, as ingest would produce
		Currency:    currency,
		EventTime:   time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Date(2026, 1, 5, 9, 5, 0, 0, time.UTC),
	}
}

func mkAccount(id, currency string) model.Account {
	return model.Account{AccountID: id, Currency: currency, Status: "ACTIVE"}
}

func TestNormalize_MiniFixtureTrace(t *testing.T) {
	accounts := []model.Account{mkAccount("ACC-M1", "USD"), mkAccount("ACC-M2", "USD")}
	entries := []model.NormalizedEntry{
		mkEntry("LE-M1", "ACC-M1", model.Credit, 1000, "USD"),
		mkEntry("LE-M2", "ACC-M1", model.Debit, 500, "USD"),
		mkEntry("LE-M2", "ACC-M1", model.Debit, 500, "USD"), // duplicate
		mkEntry("LE-M4", "ACC-M2", model.Credit, 2000, "USD"),
	}

	normalized, exceptions := Normalize(entries, accounts)

	if len(normalized) != 3 {
		t.Fatalf("expected 3 normalized entries, got %d", len(normalized))
	}
	if len(exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
	}
	if exceptions[0].ExceptionType != "duplicate_entry" {
		t.Errorf("exception type = %q, want duplicate_entry", exceptions[0].ExceptionType)
	}
	if exceptions[0].RecordID != "LE-M2" {
		t.Errorf("exception RecordID = %q, want LE-M2", exceptions[0].RecordID)
	}

	want := map[string]int64{
		"LE-M1": 1000, // CREDIT stays positive
		"LE-M2": -500, // DEBIT becomes negative
		"LE-M4": 2000, // CREDIT stays positive
	}
	for _, e := range normalized {
		w, ok := want[e.EntryID]
		if !ok {
			t.Errorf("unexpected entry_id %q in normalized output", e.EntryID)
			continue
		}
		if e.AmountCents != w {
			t.Errorf("%s: AmountCents = %d, want %d", e.EntryID, e.AmountCents, w)
		}
		if e.OriginalAmountCents != w {
			t.Errorf("%s: OriginalAmountCents = %d, want %d", e.EntryID, e.OriginalAmountCents, w)
		}
	}
}

func TestNormalize_CurrencyMismatch(t *testing.T) {
	accounts := []model.Account{mkAccount("ACC-1", "GBP")}
	entries := []model.NormalizedEntry{
		mkEntry("LE-1", "ACC-1", model.Credit, 1000, "JPY"), // mismatched currency
	}

	normalized, exceptions := Normalize(entries, accounts)

	if len(normalized) != 0 {
		t.Errorf("expected 0 normalized entries, got %d", len(normalized))
	}
	if len(exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
	}
	if exceptions[0].ExceptionType != "currency_mismatch" {
		t.Errorf("exception type = %q, want currency_mismatch", exceptions[0].ExceptionType)
	}
}

func TestNormalize_UnknownAccount(t *testing.T) {
	accounts := []model.Account{mkAccount("ACC-1", "USD")}
	entries := []model.NormalizedEntry{
		mkEntry("LE-1", "ACC-DOES-NOT-EXIST", model.Credit, 1000, "USD"),
	}

	normalized, exceptions := Normalize(entries, accounts)

	if len(normalized) != 0 {
		t.Errorf("expected 0 normalized entries, got %d", len(normalized))
	}
	if len(exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
	}
	if exceptions[0].ExceptionType != "unknown_account" {
		t.Errorf("exception type = %q, want unknown_account", exceptions[0].ExceptionType)
	}
}
