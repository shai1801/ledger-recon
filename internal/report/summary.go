package report

import (
	"github.com/shai1801/ledger-recon/internal/match"
	"github.com/shai1801/ledger-recon/internal/model"
)

// SummaryInputs collects everything needed to compute the summary record,
// gathered from each pipeline stage. Grouped into a struct (rather than a
// long positional parameter list) so call sites are self-documenting.
type SummaryInputs struct {
	Accounts               []model.Account
	IngestedLedgerEntries  []model.NormalizedEntry // valid rows from ingest, pre-normalize
	IngestLedgerExceptions []model.Exceptions      // malformed-row exceptions from ledger ingest only
	Settlements            []model.BankSettlement
	Adjustments            []model.Adjustment
	NormalizeExceptions    []model.Exceptions      // duplicates, currency mismatches, unknown accounts
	NormalizedEntries      []model.NormalizedEntry // normalize.Normalize's output
	AdjustExceptions       []model.Exceptions      // from adjust.Apply
	FinalEntries           []model.NormalizedEntry // adjust.Apply's output (post-adjustment, incl. synthesized)
	Matched                []match.MatchedPair
	UnmatchedLedger        []model.NormalizedEntry
	UnmatchedSettlements   []model.BankSettlement
	Notes                  string
}

// BuildSummary derives every count in model.Summary from the pipeline's
// intermediate results. See the field-by-field trace against
// samples/mini_expected/expected_summary.json in project notes for why
// each formula is shaped the way it is.
func BuildSummary(in SummaryInputs) model.Summary {
	return model.Summary{
		InputAccounts:              len(in.Accounts),
		InputLedgerRows:            len(in.IngestedLedgerEntries) + len(in.IngestLedgerExceptions),
		InputSettlements:           len(in.Settlements),
		InputAdjustments:           len(in.Adjustments),
		ValidLedgerRowsAfterDedupe: len(in.NormalizedEntries),
		DuplicateLedgerRows:        countByType(in.NormalizeExceptions, "duplicate_entry"),
		InvalidLedgerRows:          len(in.IngestLedgerExceptions),
		AdjustmentsApplied:         len(in.Adjustments) - len(in.AdjustExceptions),
		NormalizedLedgerEntries:    len(in.FinalEntries),
		MatchedTransactions:        len(in.Matched),
		UnmatchedLedger:            len(in.UnmatchedLedger),
		UnmatchedSettlements:       len(in.UnmatchedSettlements),
		Notes:                      in.Notes,
		UnknownAccountLedgerRows:   countByType(in.NormalizeExceptions, "unknown_account"),
		CurrencyMismatchLedgerRows: countByType(in.NormalizeExceptions, "currency_mismatch"),
	}
}

func countByType(exceptions []model.Exceptions, exceptionType string) int {
	count := 0
	for _, e := range exceptions {
		if e.ExceptionType == exceptionType {
			count++
		}
	}
	return count
}
