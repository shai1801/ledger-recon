package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/shai1801/ledger-recon/internal/model"
)

//I have created this private struct to parse the adjustments.jsonl file
//all the fields in this struct will be of type string that will match to the type in adjustments.jsonl
//file.

type rawAdjustment struct {
	AdjustmentID     string `json:"adjustment_id"`
	AdjustmentType   string `json:"adjustment_type"`
	ReferenceEntryID string `json:"reference_entry_id"`
	EventTime        string `json:"event_time"`
	Reason           string `json:"reason"`
	AmountCents      string `json:"amount_cents"`
	Currency         string `json:"currency"`
}

//I will read the adjustment.jsonl file line by line.
//This function will help in reading the file and creating the DataModel model.Adjustment.

func LoadAdjustments(r io.Reader, sourceFile string) ([]model.Adjustment, []model.Exceptions, error) {
	scanner := bufio.NewScanner(r)

	var adjustments []model.Adjustment
	var exceptions []model.Exceptions

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		adj, exc := parseAdjustmentLine(line, sourceFile, lineNum)

		if exc != nil {
			exceptions = append(exceptions, *exc)
			continue
		}
		adjustments = append(adjustments, adj)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("%s: reading lines: %w", sourceFile, err)
	}

	return adjustments, exceptions, nil
}

func parseAdjustmentLine(line, sourceFile string, lineNum int) (model.Adjustment, *model.Exceptions) {
	var raw rawAdjustment
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return model.Adjustment{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid JSON: %v", err),
		}
	}

	adjustmentID := strings.TrimSpace(raw.AdjustmentID)

	if adjustmentID == "" {
		return model.Adjustment{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        "adjustment_id is empty",
		}
	}

	adjType := model.AdjustmentType(strings.TrimSpace(raw.AdjustmentType))
	if adjType != model.Reversal && adjType != model.Correction {
		return model.Adjustment{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      adjustmentID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid adjustment_type %q: must be REVERSAL or CORRECTION", raw.AdjustmentType),
		}
	}

	referenceEntryID := strings.TrimSpace(raw.ReferenceEntryID)
	if referenceEntryID == "" {
		return model.Adjustment{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      adjustmentID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        "reference_entry_id is empty",
		}
	}

	eventTime, err := time.Parse(time.RFC3339, strings.TrimSpace(raw.EventTime))
	if err != nil {
		return model.Adjustment{}, &model.Exceptions{
			SourceFile:    sourceFile,
			LineNumber:    lineNum,
			RecordID:      adjustmentID,
			ExceptionType: "malformed_row",
			Severity:      "error",
			Reason:        fmt.Sprintf("invalid event_time %q: %v", raw.EventTime, err),
		}
	}

	var amountCents *int64
	if s := strings.TrimSpace(raw.AmountCents); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return model.Adjustment{}, &model.Exceptions{
				SourceFile:    sourceFile,
				LineNumber:    lineNum,
				RecordID:      adjustmentID,
				ExceptionType: "malformed_row",
				Severity:      "error",
				Reason:        fmt.Sprintf("invalid amount_cents %q: %v", raw.AmountCents, err),
			}
		}
		amountCents = &v
	}

	var currency *string
	if s := strings.TrimSpace(raw.Currency); s != "" {
		currency = &s
	}

	return model.Adjustment{
		AdjustmentID:     adjustmentID,
		AdjustmentType:   adjType,
		ReferenceEntryID: referenceEntryID,
		EventTime:        eventTime,
		Reason:           strings.TrimSpace(raw.Reason),
		AmountCents:      amountCents,
		Currency:         currency,
	}, nil
}
