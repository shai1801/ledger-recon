package ingest

import (
	"os"
	"testing"
)

func TestLoadLedgerEntries_MiniFixture(t *testing.T) {
	f, err := os.Open("../../testdata/mini/mini_ledger_entries.csv")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

	entries, exception, err := LoadLedgerEntries(f, "mini_ledger_entries.csv")

	if err != nil {
		t.Fatalf("LoadLedgerEntries returned unexpected error: %v", err)
	}

	// I am counting the duplicate entries because deduplication is part of normalization
	if len(entries) != 4 {
		t.Fatalf("Expected 4 entires, found %d", len(entries))
	}

	if len(exception) != 1 {
		t.Fatalf("Expected 1 exception, found %d", len(exception))
	}

	if exception[0].RecordID != "LE-M3" {
		t.Fatalf("Expected RecordID=%s, found=%s", "LE-M3", exception[0].RecordID)
	}
}
