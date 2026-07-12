package ingest

import (
	"os"
	"strings"
	"testing"
)

func TestLoadSettlements_MiniFixture(t *testing.T) {
	f, err := os.Open("../../testdata/mini/mini_bank_settlements.csv")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

	settlements, exceptions, err := LoadSettlements(f, "mini_bank_settlements.csv")
	if err != nil {
		t.Fatalf("LoadSettlements returned unexpected error: %v", err)
	}
	if len(exceptions) != 0 {
		t.Errorf("expected 0 exceptions, got %d: %+v", len(exceptions), exceptions)
	}
	if len(settlements) != 4 {
		t.Fatalf("expected 4 settlements, got %d", len(settlements))
	}

	want := map[string]struct {
		BankRef     string
		AccountID   string
		AmountCents int64
		Status      string
	}{
		"SET-M1": {BankRef: "TXN-M1", AccountID: "ACC-M1", AmountCents: 1001, Status: "SETTLED"},
		"SET-M2": {BankRef: "TXN-M2", AccountID: "ACC-M1", AmountCents: -500, Status: "SETTLED"},
		"SET-M3": {BankRef: "TXN-M4", AccountID: "ACC-M2", AmountCents: 2000, Status: "SETTLED"},
		"SET-M4": {BankRef: "TXN-UNKNOWN", AccountID: "ACC-M2", AmountCents: 750, Status: "SETTLED"},
	}

	for _, s := range settlements {
		w, ok := want[s.SettlementID]
		if !ok {
			t.Errorf("unexpected settlement_id %q", s.SettlementID)
			continue
		}
		if s.BankRef != w.BankRef {
			t.Errorf("%s: bank_ref = %q, want %q", s.SettlementID, s.BankRef, w.BankRef)
		}
		if s.AccountID != w.AccountID {
			t.Errorf("%s: account_id = %q, want %q", s.SettlementID, s.AccountID, w.AccountID)
		}
		if s.AmountCents != w.AmountCents {
			t.Errorf("%s: amount_cents = %d, want %d", s.SettlementID, s.AmountCents, w.AmountCents)
		}
		if s.Status != w.Status {
			t.Errorf("%s: status = %q, want %q", s.SettlementID, s.Status, w.Status)
		}
	}
}

func TestLoadSettlements_MalformedRows(t *testing.T) {
	header := "settlement_id,bank_ref,account_id,amount_cents,currency,settled_at,status,processor\n"

	tests := []struct {
		name       string
		csvContent string
	}{
		{
			name:       "empty settlement_id",
			csvContent: header + ",TXN-1,ACC-1,1000,USD,2026-01-06T09:00:00Z,SETTLED,BANK-A\n",
		},
		{
			name:       "non-numeric amount_cents",
			csvContent: header + "SET-1,TXN-1,ACC-1,not-a-number,USD,2026-01-06T09:00:00Z,SETTLED,BANK-A\n",
		},
		{
			name:       "invalid settled_at",
			csvContent: header + "SET-1,TXN-1,ACC-1,1000,USD,not-a-date,SETTLED,BANK-A\n",
		},
		{
			name:       "invalid status",
			csvContent: header + "SET-1,TXN-1,ACC-1,1000,USD,2026-01-06T09:00:00Z,SETLED,BANK-A\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			settlements, exceptions, err := LoadSettlements(strings.NewReader(tc.csvContent), "test.csv")
			if err != nil {
				t.Fatalf("unexpected structural error: %v", err)
			}
			if len(settlements) != 0 {
				t.Errorf("expected 0 settlements, got %d", len(settlements))
			}
			if len(exceptions) != 1 {
				t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
			}
		})
	}
}

func TestLoadSettlements_MissingRequiredColumn(t *testing.T) {
	csvContent := "settlement_id,bank_ref,account_id,amount_cents,currency,settled_at,processor\n" +
		"SET-1,TXN-1,ACC-1,1000,USD,2026-01-06T09:00:00Z,BANK-A\n"

	_, _, err := LoadSettlements(strings.NewReader(csvContent), "test.csv")
	if err == nil {
		t.Fatal("expected structural error for missing required column, got nil")
	}
}
