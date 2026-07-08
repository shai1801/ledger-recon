package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/shai1801/ledger-recon/internal/model"
)

var ledgerRequiredColumns = []string{
	"entry_id", "account_id", "txn_id", "source_system", "direction",
	"amount_cents", "currency", "event_time", "posted_at", "shard_id", "description",
}

// This function will ingest rows from ledger_entries.csv

//I am not applying the validation on the AmountCent and the output will be in raw form
//The AmountCent will be converted to signed values in the Normalization stage.
//Here I am only checking if the rows are well formed and all the required colums exists

func LoadLedgerEntries(r io.Reader, sourceFile string) ([]model.NormalizedEntry, []model.Exceptions, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: reading header: %w", sourceFile, err)
	}

	colIndex, err := buildColumnIndex(header, ledgerRequiredColumns)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", sourceFile, err)
	}

	var entries []model.NormalizedEntry
	var exceptions []model.Exceptions

	lineNum := 1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		lineNum++
		if err != nil {
			exceptions = append(exceptions, model.Exceptions{
				SourceFile:    sourceFile,
				LineNumber:    lineNum,
				ExceptionType: "malformed_row",
				Severity:      "error",
				Reason:        err.Error(),
			})
			continue
		}
		if len(row) != len(header) {
			exceptions = append(exceptions, model.Exceptions{
				SourceFile:    sourceFile,
				LineNumber:    lineNum,
				ExceptionType: "malformed_row",
				Reason:        fmt.Sprintf("expected %d columns, got %d", len(header), len(row)),
			})
			continue

		}

		entry, exc := parseLedgerRow(row, colIndex, sourceFile, lineNum)

		if exc != nil {
			exceptions = append(exceptions, *exc)
			continue
		}
		entries = append(entries, entry)
	}
	return entries, exceptions, nil
}

func parseLedgerRow(row []string, col map[string]int, sourceFile string, lineNum int) (model.NormalizedEntry, *model.Exceptions) {

	get := func(name string) string {
		return strings.TrimSpace(row[col[name]])
	}

	entryID := get("entry_id")
	if entryID == "" {
		return model.NormalizedEntry{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        "entry id is empty",
		}
	}

	direction := model.Direction(get("direction"))
	if direction != model.Credit && direction != model.Debit {
		return model.NormalizedEntry{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      entryID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid direction %q: must be CREDIT or DEBIT", get("direction")),
		}
	}

	amountCents, err := strconv.ParseInt(get("amount_cents"), 10, 64)
	if err != nil {
		return model.NormalizedEntry{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      entryID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid amount_cents %q: %v", get("amount_cents"), err),
		}
	}
	if amountCents <= 0 {
		return model.NormalizedEntry{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      entryID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("amount_cents must be positive, got %d", amountCents),
		}
	}

	eventTime, err := time.Parse(time.RFC3339, get("event_time"))
	if err != nil {
		return model.NormalizedEntry{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      entryID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid event_time %q: %v", get("event_time"), err),
		}
	}

	postedAt, err := time.Parse(time.RFC3339, get("posted_at"))
	if err != nil {
		return model.NormalizedEntry{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      entryID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid posted_at %q: %v", get("posted_at"), err),
		}
	}

	return model.NormalizedEntry{
		EntryID:      entryID,
		AccountID:    get("account_id"),
		TxnID:        get("txn_id"),
		SourceSystem: get("source_system"),
		Direction:    direction,
		AmountCents:  amountCents, // raw positive value; sign applied in normalize
		Currency:     get("currency"),
		EventTime:    eventTime,
		PostedAt:     postedAt,
		ShardID:      get("shard_id"),
		Description:  get("description"),
	}, nil
}
