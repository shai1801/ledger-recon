//go:build integration

package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/shai1801/ledger-recon/internal/adjust"
	"github.com/shai1801/ledger-recon/internal/config"
	"github.com/shai1801/ledger-recon/internal/ingest"
	"github.com/shai1801/ledger-recon/internal/match"
	"github.com/shai1801/ledger-recon/internal/model"
	"github.com/shai1801/ledger-recon/internal/normalize"
	"github.com/shai1801/ledger-recon/internal/report"
	"github.com/shai1801/ledger-recon/internal/source"
)

func i64ptr(v int64) *int64 { return &v }
func iptr(v int) *int       { return &v }

// TestReconcile_MiniFixture_MatchesExpectedSummary runs the full pipeline
// against testdata/mini and asserts every count matches
// mini_expected/expected_summary.json exactly. tolerance_cents and
// matching_window_days are supplied explicitly (2 and 3, matching the real
// configuration.json) because testdata/mini has no configuration.json of
// its own -- the expected values were computed assuming those parameters,
// not our hardcoded zero-value defaults.
func TestReconcile_MiniFixture_MatchesExpectedSummary(t *testing.T) {
	pkg, err := source.Open("../../testdata/mini")
	if err != nil {
		t.Fatalf("opening mini fixture: %v", err)
	}
	defer pkg.Close()

	accounts, _, err := loadAndIngest(pkg, "mini_accounts.csv", ingest.LoadAccounts)
	if err != nil {
		t.Fatalf("loading accounts: %v", err)
	}
	ledgerEntries, ledgerExceptions, err := loadAndIngest(pkg, "mini_ledger_entries.csv", ingest.LoadLedgerEntries)
	if err != nil {
		t.Fatalf("loading ledger entries: %v", err)
	}
	settlements, _, err := loadAndIngest(pkg, "mini_bank_settlements.csv", ingest.LoadSettlements)
	if err != nil {
		t.Fatalf("loading settlements: %v", err)
	}
	adjustments, _, err := loadAndIngest(pkg, "mini_adjustments.jsonl", ingest.LoadAdjustments)
	if err != nil {
		t.Fatalf("loading adjustments: %v", err)
	}

	cli := config.CLIOverrides{
		ToleranceCents:     i64ptr(2),
		MatchingWindowDays: iptr(3),
	}
	resolver := config.NewResolver(cli, config.DefaultConfig())

	normalizedEntries, normalizeExceptions := normalize.Normalize(ledgerEntries, accounts)
	finalEntries, adjustExceptions := adjust.Apply(normalizedEntries, adjustments)
	matched, unmatchedLedger, unmatchedSettlements := match.Match(
		finalEntries, settlements,
		resolver.ToleranceCents(), resolver.MatchingWindowDays(), resolver.IncludePendingSettlements(),
	)

	summary := report.BuildSummary(report.SummaryInputs{
		Accounts:               accounts,
		IngestedLedgerEntries:  ledgerEntries,
		IngestLedgerExceptions: ledgerExceptions,
		Settlements:            settlements,
		Adjustments:            adjustments,
		NormalizeExceptions:    normalizeExceptions,
		NormalizedEntries:      normalizedEntries,
		AdjustExceptions:       adjustExceptions,
		FinalEntries:           finalEntries,
		Matched:                matched,
		UnmatchedLedger:        unmatchedLedger,
		UnmatchedSettlements:   unmatchedSettlements,
	})

	expectedBytes, err := os.ReadFile("../../testdata/mini/mini_expected/expected_summary.json")
	if err != nil {
		t.Fatalf("reading expected_summary.json: %v", err)
	}
	var expected model.Summary
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatalf("parsing expected_summary.json: %v", err)
	}

	checks := []struct {
		name string
		got  int
		want int
	}{
		{"InputAccounts", summary.InputAccounts, expected.InputAccounts},
		{"InputLedgerRows", summary.InputLedgerRows, expected.InputLedgerRows},
		{"InputSettlements", summary.InputSettlements, expected.InputSettlements},
		{"InputAdjustments", summary.InputAdjustments, expected.InputAdjustments},
		{"ValidLedgerRowsAfterDedupe", summary.ValidLedgerRowsAfterDedupe, expected.ValidLedgerRowsAfterDedupe},
		{"DuplicateLedgerRows", summary.DuplicateLedgerRows, expected.DuplicateLedgerRows},
		{"InvalidLedgerRows", summary.InvalidLedgerRows, expected.InvalidLedgerRows},
		{"AdjustmentsApplied", summary.AdjustmentsApplied, expected.AdjustmentsApplied},
		{"NormalizedLedgerEntries", summary.NormalizedLedgerEntries, expected.NormalizedLedgerEntries},
		{"MatchedTransactions", summary.MatchedTransactions, expected.MatchedTransactions},
		{"UnmatchedLedger", summary.UnmatchedLedger, expected.UnmatchedLedger},
		{"UnmatchedSettlements", summary.UnmatchedSettlements, expected.UnmatchedSettlements},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}
