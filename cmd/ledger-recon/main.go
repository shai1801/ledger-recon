package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/shai1801/ledger-recon/internal/adjust"
	"github.com/shai1801/ledger-recon/internal/config"
	"github.com/shai1801/ledger-recon/internal/ingest"
	"github.com/shai1801/ledger-recon/internal/match"
	"github.com/shai1801/ledger-recon/internal/model"
	"github.com/shai1801/ledger-recon/internal/normalize"
	"github.com/shai1801/ledger-recon/internal/query"
	"github.com/shai1801/ledger-recon/internal/report"
	"github.com/shai1801/ledger-recon/internal/source"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// filterFlag implements flag.Value, letting --filter be passed multiple
// times (e.g. --filter currency_eq=USD --filter amount_cents_min=1000).
type filterFlag struct {
	values map[string]string
}

func (f *filterFlag) String() string { return "" }

func (f *filterFlag) Set(s string) error {
	key, value, ok := strings.Cut(s, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return fmt.Errorf("--filter expects key=value, got %q", s)
	}
	f.values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	return nil
}

func run(args []string) error {
	fs := flag.NewFlagSet("ledger-recon", flag.ContinueOnError)

	reconcileMode := fs.Bool("reconcile", false, "run reconciliation and write reports")
	input := fs.String("input", ".", "input data folder or zip file")
	output := fs.String("output", "sample-output", "output report directory")
	configPath := fs.String("config", "", "configuration.json path override")
	storePath := fs.String("store", "", "query_store file path override")
	table := fs.String("table", "", "query table name")
	queryExpr := fs.String("query", "", "structured filter expression")

	toleranceCents := fs.Int64("tolerance-cents", 0, "amount matching tolerance in cents")
	matchingWindowDays := fs.Int("matching-window-days", 0, "settlement matching window in days")
	outputFormat := fs.String("output-format", "", "report format: jsonl or json")
	partitionBy := fs.String("partition-by", "", "query-store partition field (reserved)")
	includePending := fs.Bool("include-pending-settlements", false, "allow pending/returned settlements to match")
	queryLimit := fs.Int("query-limit", 0, "maximum query rows")

	filterValues := make(map[string]string)
	fs.Var(&filterFlag{filterValues}, "filter", "dynamic filter field_operator=value (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cli := config.CLIOverrides{Filters: filterValues}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "tolerance-cents":
			cli.ToleranceCents = toleranceCents
		case "matching-window-days":
			cli.MatchingWindowDays = matchingWindowDays
		case "output-format":
			cli.OutputFormat = outputFormat
		case "partition-by":
			cli.PartitionBy = partitionBy
		case "include-pending-settlements":
			cli.IncludePendingSettlements = includePending
		case "query-limit":
			cli.QueryLimit = queryLimit
		}
	})

	if *reconcileMode && (*table != "" || *queryExpr != "" || len(filterValues) > 0) {
		return errors.New("--reconcile cannot be combined with --table, --query, or --filter")
	}
	if !*reconcileMode && *table == "" {
		return errors.New("choose --reconcile, or --table (with --query and/or --filter)")
	}

	if *reconcileMode {
		return runReconcile(*input, *output, *configPath, cli)
	}
	return runQuery(*output, *storePath, *configPath, *input, *table, *queryExpr, cli)
}

func runReconcile(inputPath, outputDir, configPath string, cli config.CLIOverrides) error {
	pkg, err := source.Open(inputPath)
	if err != nil {
		return err
	}
	defer pkg.Close()

	cfg, err := loadConfig(configPath, pkg)
	if err != nil {
		return err
	}
	resolver := config.NewResolver(cli, cfg)

	accounts, accountExceptions, err := loadAndIngest(pkg, "accounts.csv", ingest.LoadAccounts)
	if err != nil {
		return err
	}
	ledgerEntries, ledgerExceptions, err := loadAndIngest(pkg, "ledger_entries.csv", ingest.LoadLedgerEntries)
	if err != nil {
		return err
	}
	settlements, settlementExceptions, err := loadAndIngest(pkg, "bank_settlements.csv", ingest.LoadSettlements)
	if err != nil {
		return err
	}
	adjustments, adjustmentExceptions, err := loadAndIngest(pkg, "adjustments.jsonl", ingest.LoadAdjustments)
	if err != nil {
		return err
	}

	normalizedEntries, normalizeExceptions := normalize.Normalize(ledgerEntries, accounts)
	finalEntries, adjustExceptions := adjust.Apply(normalizedEntries, adjustments)
	matched, unmatchedLedger, unmatchedSettlements := match.Match(
		finalEntries,
		settlements,
		resolver.ToleranceCents(),
		resolver.MatchingWindowDays(),
		resolver.IncludePendingSettlements(),
	)

	allExceptions := appendExceptions(
		accountExceptions, ledgerExceptions, settlementExceptions,
		adjustmentExceptions, normalizeExceptions, adjustExceptions,
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

	accountsReconciliation := report.BuildAccountsReconciliation(accounts, finalEntries)
	matchedTransactions := report.BuildMatchedTransactions(matched)

	queryStore, err := report.BuildQueryStore(accounts, finalEntries, matched, unmatchedLedger, unmatchedSettlements, allExceptions)
	if err != nil {
		return err
	}

	format := resolver.OutputFormat()
	writes := []struct {
		name string
		fn   func() error
	}{
		{"summary", func() error { return report.WriteRecords(outputDir, "summary", format, []model.Summary{summary}) }},
		{"accounts_reconciliation", func() error {
			return report.WriteRecords(outputDir, "accounts_reconciliation", format, accountsReconciliation)
		}},
		{"matched_transactions", func() error {
			return report.WriteRecords(outputDir, "matched_transactions", format, matchedTransactions)
		}},
		{"unmatched_ledger", func() error {
			return report.WriteRecords(outputDir, "unmatched_ledger", format, unmatchedLedger)
		}},
		{"unmatched_settlements", func() error {
			return report.WriteRecords(outputDir, "unmatched_settlements", format, unmatchedSettlements)
		}},
		{"exceptions", func() error { return report.WriteRecords(outputDir, "exceptions", format, allExceptions) }},
		{"query_store", func() error {
			// query_store is always JSONL, one record per line, regardless
			// of output_format -- query.LoadStore expects that shape.
			return report.WriteRecords(outputDir, "query_store", "jsonl", queryStore)
		}},
	}
	for _, w := range writes {
		if err := w.fn(); err != nil {
			return fmt.Errorf("writing %s: %w", w.name, err)
		}
	}

	fmt.Printf("wrote reconciliation reports to %s\n", outputDir)
	return nil
}

func runQuery(outputDir, storePath, configPath, inputPath, table, expr string, cli config.CLIOverrides) error {
	if storePath == "" {
		storePath = outputDir + "/query_store.jsonl"
	}

	var cfg = config.DefaultConfig()
	if pkg, err := source.Open(inputPath); err == nil {
		defer pkg.Close()
		if c, err := loadConfig(configPath, pkg); err == nil {
			cfg = c
		}
	} else if configPath != "" {
		c, err := config.LoadConfig(configPath)
		if err != nil {
			return err
		}
		cfg = c
	}
	resolver := config.NewResolver(cli, cfg)

	rows, err := query.LoadStore(storePath)
	if err != nil {
		return err
	}

	results, err := query.Execute(rows, table, expr, resolver.Filters(), resolver.QueryLimit())
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// loadAndIngest opens name from pkg and runs it through the given ingest
// loader function, closing the reader afterward regardless of outcome.
func loadAndIngest[T any](
	pkg *source.Package,
	name string,
	loader func(r io.Reader, sourceFile string) ([]T, []model.Exceptions, error),
) ([]T, []model.Exceptions, error) {
	r, err := pkg.Reader(name)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	return loader(r, name)
}

func loadConfig(configPath string, pkg *source.Package) (config.FileConfig, error) {
	if configPath != "" {
		return config.LoadConfig(configPath)
	}
	if pkg != nil {
		if data, ok := pkg.Bytes("configuration.json"); ok {
			return config.ParseConfig(data)
		}
	}
	if _, err := os.Stat("configuration.json"); err == nil {
		return config.LoadConfig("configuration.json")
	}
	return config.DefaultConfig(), nil
}

func appendExceptions(groups ...[]model.Exceptions) []model.Exceptions {
	var all []model.Exceptions
	for _, g := range groups {
		all = append(all, g...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.SourceFile != b.SourceFile {
			return a.SourceFile < b.SourceFile
		}
		if a.LineNumber != b.LineNumber {
			return a.LineNumber < b.LineNumber
		}
		return a.RecordID < b.RecordID
	})
	return all
}
