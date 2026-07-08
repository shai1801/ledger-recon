package model

import "time"

type AdjustmentType string

const (
	Reversal   AdjustmentType = "REVERSAL"
	Correction AdjustmentType = "CORRECTION"
)

//This model describe single line from the adjustment.jsonl file
//I have kept the Currency and AmountCent as pointer because the REVERSAL
// adjustment have empty values for these two fields.
type Adjustment struct {
	AdjustmentID     string
	AdjustmentType   AdjustmentType
	ReferenceEntryID string
	EventTime        time.Time
	Reason           string
	AmountCents      *int64
	Currency         *string
}
