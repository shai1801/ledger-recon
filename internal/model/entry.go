package model

import "time"

type Direction string // This is direction from ledger_entries.csv

const (
	Credit Direction = "CREDIT"
	Debit  Direction = "DEBIT"
)

//NormalizedEntry is canonical model which is post-normalization representation
//of ledger_entries.csv rows.
//As per the instuction AmountCent is signed : positive for CREDIT and negative (-) for DEBIT

//AmountCent is mutable - CORRECTION Adjustments  overwrite it in place.
//OriginalAmountCent preserves the value as first normalized entry
//AdjustmentID slice holds the history of how the correction adjustments have changed the vlaue in AmountCent

type NormalizedEntry struct {
	EntryID      string
	AccountID    string
	TxnID        string
	SourceSystem string
	Direction    Direction
	AmountCents  int64 // signed integer, this is current value post correction
	Currency     string
	EventTime    time.Time
	PostedAt     time.Time
	ShardID      string
	Description  string

	OriginalAmountCents  int64
	AppliedAdjustmentIDs []string //multiple adjustments for a particular reference_entry_id can be made

}
