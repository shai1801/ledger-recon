package ingest

import (
	"os"
	"strings"
	"testing"
)

func TestLoadAccounts(t *testing.T) {
	f, err := os.Open("../../testdata/mini/mini_accounts.csv")
	if err != nil {
		t.Fatalf("file opening failure %v", err)
	}

	defer f.Close()

	accounts, exceptions, err := LoadAccounts(f, "mini_accounts.csv")

	if err != nil {
		t.Fatalf("LoadAccounts returned unexpected error: %v", err)
	}

	if len(exceptions) != 0 {
		t.Errorf("expected 0 exceptions, got %d: %+v", len(exceptions), exceptions)
	}

	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	want := map[string]struct {
		Currency            string
		OpeningBalanceCents int64
	}{
		"ACC-M1": {Currency: "USD", OpeningBalanceCents: 10000},
		"ACC-M2": {Currency: "USD", OpeningBalanceCents: 5000},
	}

	for _, acc := range accounts {
		w, ok := want[acc.AccountID]

		if !ok {
			t.Errorf("unexpected account_id %s", acc.AccountID)
			continue
		}

		if acc.Currency != w.Currency {
			t.Errorf("%s has currency %s, expected currency is %s", acc.AccountID, acc.Currency, w.Currency)
		}

		if acc.OpeningBalanceCent != w.OpeningBalanceCents {
			t.Errorf("%s: Opening Balance = %d, want = %d", acc.AccountID, acc.OpeningBalanceCent, w.OpeningBalanceCents)
		}
	}
}

func TestLoadAccounts_MalformedRows(t *testing.T) {
	header := "account_id, customer_id, account_type, currency, opening_balance_cents, status, created_at\n"
	tests := []struct {
		name        string
		csvContent  string
		wantExcType string
	}{
		{
			name:        "empty account_id",
			csvContent:  header + ",CUST-1,CHECKING,USD,1000,ACTIVE,2026-01-01T00:00:00Z\n",
			wantExcType: "malformed_row",
		},
		{
			name:        "non-numeric opening_balance_cents",
			csvContent:  header + "ACC-1,CUST-1,CHECKING,USD,not-a-number,ACTIVE,2026-01-01T00:00:00Z\n",
			wantExcType: "malformed_row",
		},
		{
			name:        "invalid created_at",
			csvContent:  header + "ACC-1,CUST-1,CHECKING,USD,1000,ACTIVE,not-a-date\n",
			wantExcType: "malformed_row",
		},
		{
			name:        "wrong column count",
			csvContent:  header + "ACC-1,CUST-1,CHECKING,USD,1000\n",
			wantExcType: "malformed_row",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			accounts, exceptions, err := LoadAccounts(strings.NewReader(tc.csvContent), "testCsv")
			if err != nil {
				t.Fatalf("unexpected structural error: %v", err)
			}

			if len(accounts) != 0 {
				t.Errorf("expected 0 accounts, got %d", len(accounts))
			}

			if len(exceptions) != 1 {
				t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
			}

			if exceptions[0].ExceptionType != tc.wantExcType {
				t.Errorf("SourceFile=%q, want=%q", exceptions[0].SourceFile, "test.csv")
			}

			if exceptions[0].LineNumber != 2 {
				t.Errorf("LineNumber=%d, want 2", exceptions[0].LineNumber)
			}
		})
	}
}

func TestLoadAccounts_MissingRequiredColumn(t *testing.T) {
	csvContent := "account_id,customer_id,currency,opening_balance_cents,status,created_at\n" +
		"ACC-1, CUST-1, USD, 1000, ACTIVE, 2026-01-01T00:00:00Z\n"

	_, _, err := LoadAccounts(strings.NewReader(csvContent), "test.csv")
	if err == nil {
		t.Fatal("expected structural error for missing required column, got nil")
	}
}
