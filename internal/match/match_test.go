package match

import (
	"testing"
	"time"

	"github.com/shai1801/ledger-recon/internal/model"
)

func mkEntry(id, accountID, txnID string, direction model.Direction, amount int64, currency string, postedAt time.Time) model.NormalizedEntry {
	return model.NormalizedEntry{
		EntryID:     id,
		AccountID:   accountID,
		TxnID:       txnID,
		Direction:   direction,
		AmountCents: amount,
		Currency:    currency,
		PostedAt:    postedAt,
	}
}

func mkSettlement(id, bankRef, accountID string, amount int64, currency, status string, settledAt time.Time) model.BankSettlement {
	return model.BankSettlement{
		SettlementID: id,
		BankRef:      bankRef,
		AccountID:    accountID,
		AmountCents:  amount,
		Currency:     currency,
		Status:       status,
		SettledAt:    settledAt,
	}
}

func TestMatch_MiniFixtureTrace(t *testing.T) {
	day := func(d int, h int) time.Time { return time.Date(2026, 1, d, h, 0, 0, 0, time.UTC) }

	entries := []model.NormalizedEntry{
		mkEntry("LE-M1", "ACC-M1", "TXN-M1", model.Credit, 1000, "USD", day(5, 9)),
		mkEntry("LE-M2", "ACC-M1", "TXN-M2", model.Debit, -500, "USD", day(5, 10)),
		mkEntry("LE-M4", "ACC-M2", "TXN-M4", model.Credit, 2000, "USD", day(5, 12)),
		mkEntry("ADJ-M1", "ACC-M2", "TXN-M4", model.Debit, -2000, "USD", day(5, 13)), // synthetic reversal
	}
	settlements := []model.BankSettlement{
		mkSettlement("SET-M1", "TXN-M1", "ACC-M1", 1001, "USD", "SETTLED", day(6, 9)),
		mkSettlement("SET-M2", "TXN-M2", "ACC-M1", -500, "USD", "SETTLED", day(6, 10)),
		mkSettlement("SET-M3", "TXN-M4", "ACC-M2", 2000, "USD", "SETTLED", day(6, 12)),
		mkSettlement("SET-M4", "TXN-UNKNOWN", "ACC-M2", 750, "USD", "SETTLED", day(6, 12)),
	}

	matched, unmatchedLedger, unmatchedSettlements := Match(entries, settlements, 2, 3, false)

	if len(matched) != 3 {
		t.Fatalf("expected 3 matched pairs, got %d: %+v", len(matched), matched)
	}
	if len(unmatchedLedger) != 1 {
		t.Fatalf("expected 1 unmatched ledger entry, got %d", len(unmatchedLedger))
	}
	if unmatchedLedger[0].EntryID != "ADJ-M1" {
		t.Errorf("unmatched ledger entry = %q, want ADJ-M1 (the reversal)", unmatchedLedger[0].EntryID)
	}
	if len(unmatchedSettlements) != 1 {
		t.Fatalf("expected 1 unmatched settlement, got %d", len(unmatchedSettlements))
	}
	if unmatchedSettlements[0].SettlementID != "SET-M4" {
		t.Errorf("unmatched settlement = %q, want SET-M4", unmatchedSettlements[0].SettlementID)
	}

	wantPairs := map[string]string{ // entry_id -> settlement_id
		"LE-M1": "SET-M1",
		"LE-M2": "SET-M2",
		"LE-M4": "SET-M3",
	}
	for _, m := range matched {
		want, ok := wantPairs[m.Entry.EntryID]
		if !ok {
			t.Errorf("unexpected match for entry %q", m.Entry.EntryID)
			continue
		}
		if m.Settlement.SettlementID != want {
			t.Errorf("%s matched %q, want %q", m.Entry.EntryID, m.Settlement.SettlementID, want)
		}
	}
}

func TestMatch_ToleranceBoundary(t *testing.T) {
	day := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	entries := []model.NormalizedEntry{
		mkEntry("LE-1", "ACC-1", "TXN-1", model.Credit, 1000, "USD", day),
	}

	t.Run("exactly at tolerance matches", func(t *testing.T) {
		settlements := []model.BankSettlement{
			mkSettlement("SET-1", "TXN-1", "ACC-1", 1002, "USD", "SETTLED", day),
		}
		matched, _, _ := Match(entries, settlements, 2, 1, false)
		if len(matched) != 1 {
			t.Errorf("expected a match at exactly tolerance_cents, got %d matches", len(matched))
		}
	})

	t.Run("one cent over tolerance does not match", func(t *testing.T) {
		settlements := []model.BankSettlement{
			mkSettlement("SET-1", "TXN-1", "ACC-1", 1003, "USD", "SETTLED", day),
		}
		matched, _, unmatchedSettlements := Match(entries, settlements, 2, 1, false)
		if len(matched) != 0 {
			t.Errorf("expected no match beyond tolerance_cents, got %d matches", len(matched))
		}
		if len(unmatchedSettlements) != 1 {
			t.Errorf("expected settlement to be unmatched, got %d unmatched", len(unmatchedSettlements))
		}
	})
}

func TestMatch_WindowIsForwardOnly(t *testing.T) {
	postedAt := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	entries := []model.NormalizedEntry{
		mkEntry("LE-1", "ACC-1", "TXN-1", model.Credit, 1000, "USD", postedAt),
	}

	t.Run("settled before posted_at does not match", func(t *testing.T) {
		settledBefore := postedAt.AddDate(0, 0, -1)
		settlements := []model.BankSettlement{
			mkSettlement("SET-1", "TXN-1", "ACC-1", 1000, "USD", "SETTLED", settledBefore),
		}
		matched, _, _ := Match(entries, settlements, 0, 3, false)
		if len(matched) != 0 {
			t.Errorf("expected no match for a settlement dated before posted_at, got %d", len(matched))
		}
	})

	t.Run("settled after window does not match", func(t *testing.T) {
		tooLate := postedAt.AddDate(0, 0, 4) // window is 3 days
		settlements := []model.BankSettlement{
			mkSettlement("SET-1", "TXN-1", "ACC-1", 1000, "USD", "SETTLED", tooLate),
		}
		matched, _, _ := Match(entries, settlements, 0, 3, false)
		if len(matched) != 0 {
			t.Errorf("expected no match beyond matching_window_days, got %d", len(matched))
		}
	})
}

func TestMatch_PendingSettlementRequiresOptIn(t *testing.T) {
	day := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	entries := []model.NormalizedEntry{
		mkEntry("LE-1", "ACC-1", "TXN-1", model.Credit, 1000, "USD", day),
	}
	settlements := []model.BankSettlement{
		mkSettlement("SET-1", "TXN-1", "ACC-1", 1000, "USD", "PENDING", day),
	}

	t.Run("excluded by default", func(t *testing.T) {
		matched, _, unmatchedSettlements := Match(entries, settlements, 0, 3, false)
		if len(matched) != 0 {
			t.Errorf("expected PENDING settlement excluded by default, got %d matches", len(matched))
		}
		if len(unmatchedSettlements) != 1 {
			t.Errorf("expected PENDING settlement reported as unmatched, got %d", len(unmatchedSettlements))
		}
	})

	t.Run("included when opted in", func(t *testing.T) {
		matched, _, _ := Match(entries, settlements, 0, 3, true)
		if len(matched) != 1 {
			t.Errorf("expected PENDING settlement to match when include_pending_settlements=true, got %d matches", len(matched))
		}
	})
}
