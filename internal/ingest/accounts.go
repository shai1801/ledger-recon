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

var accountsRequiredColumns = []string{"account_id",
	"customer_id",
	"account_type",
	"currency",
	"opening_balance_cents",
	"status",
	"created_at",
}

func LoadAccounts(r io.Reader, sourceFile string) ([]model.Account, []model.Exceptions, error) {

	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // -1 ensure that two rows can have different number fields. This will help me in capturing the Exceptions

	header, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: reading header: %w", sourceFile, err)
	}

	colIndex, err := buildColumnIndex(header, accountsRequiredColumns)

	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", sourceFile, err)
	}

	var accounts []model.Account
	var exceptions []model.Exceptions

	lineNum := 1 // line 1 is the header of the table

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
				Reason:        fmt.Sprintf("Expected %d coulums, got %d", len(header), len(row)),
			})
			continue
		}

		acc, exc := parserAccountRow(row, colIndex, sourceFile, lineNum)

		if exc != nil {
			exceptions = append(exceptions, *exc)
			continue
		}
		accounts = append(accounts, acc)
	}

	return accounts, exceptions, nil
}

func parserAccountRow(row []string, col map[string]int, sourceFile string,
	lineNum int) (model.Account, *model.Exceptions) {
	get := func(name string) string {
		return strings.TrimSpace(row[col[name]])
	}

	accountID := get("account_id")
	if accountID == "" {
		return model.Account{}, &model.Exceptions{

			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        "account_id is empty",
		}
	}

	openingBalance, err := strconv.ParseInt(get("opening_balance_cents"), 10, 64)
	if err != nil {
		return model.Account{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("Invalid opening_balance_cents %q: %v", get("opening_balance_cents"), err),
		}
	}

	createdAt, err := time.Parse(time.RFC3339, get("created_at"))
	if err != nil {
		return model.Account{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid created_at %q %v", get("created_at"), err),
		}
	}

	return model.Account{
		AccountID:           accountID,
		CustomerID:          get("customer_id"),
		AccountType:         get("account_type"),
		Currency:            get("currency"),
		OpeningBalanceCents: openingBalance,
		Status:              get("status"),
		CreatedAt:           createdAt,
	}, nil
}

//I am writing this function to map each column with its positing in the file.
//This functions takes the mandatory fields/columns described in the guideline
//  and matches with header and return and error with all the missing columns

func buildColumnIndex(header []string, required []string) (map[string]int, error) {
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[strings.TrimSpace(name)] = i
	}

	var missing []string
	for _, col := range required {
		if _, ok := index[col]; !ok {
			missing = append(missing, col)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required column(s): %s", strings.Join(missing, ", "))
	}
	return index, nil
}
