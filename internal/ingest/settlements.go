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

var settlementRequiredColumns = []string{"settlement_id", "bank_ref", "account_id", "amount_cents", "currency", "settled_at", "status", "processor"}

var validSettlementStatuses = map[string]bool{
	"SETTLED":  true,
	"PENDING":  true,
	"RETURNED": true,
}

func LoadSettlements(r io.Reader, sourceFile string) ([]model.BankSettlement, []model.Exceptions, error) {

	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()

	if err != nil {
		return nil, nil, fmt.Errorf("%s: reading header: %w", sourceFile, err)
	}

	colIndex, err := buildColumnIndex(header, settlementRequiredColumns)

	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", sourceFile, err)
	}

	var settlements []model.BankSettlement
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
				Severity:      "error",
				Reason:        fmt.Sprintf("expected %d columns, got %d", len(header), len(row)),
			})

			continue
		}

		s, exc := parseSettlementRow(row, colIndex, sourceFile, lineNum)

		if exc != nil {
			exceptions = append(exceptions, *exc)
			continue
		}

		settlements = append(settlements, s)
	}

	return settlements, exceptions, nil

}

func parseSettlementRow(row []string, col map[string]int, sourceFile string, lineNum int) (model.BankSettlement, *model.Exceptions) {

	var settlement model.BankSettlement

	get := func(name string) string {

		return strings.TrimSpace(row[col[name]])

	}

	settlementID := get("settlement_id")
	if settlementID == "" {
		return model.BankSettlement{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        "settlement_id is empty",
		}
	}

	amountCents, err := strconv.ParseInt(get("amount_cents"), 10, 64)
	if err != nil {
		return model.BankSettlement{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      settlementID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid amount_cent %q: %v", get("amount_cents"), err),
		}
	}

	settledAt, err := time.Parse(time.RFC3339, get("settled_at"))
	if err != nil {
		return model.BankSettlement{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      settlementID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid settled_at %q: %v", get("settled_at"), err),
		}
	}

	status := get("status")
	if !validSettlementStatuses[status] {
		return model.BankSettlement{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      settlementID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid status %q: must be SETTLED, PENDING or RETURNED", status),
		}
	}

	settlement.AccountID = settlementID
	settlement.AccountID = get("account_id")
	settlement.AmountCents = amountCents
	settlement.BankRef = get("bank_ref")
	settlement.Currency = get("currency")
	settlement.Processor = get("processor")
	settlement.Status = status
	settlement.SettledAt = settledAt

	return settlement, nil

}
