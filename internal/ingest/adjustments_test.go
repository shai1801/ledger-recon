package ingest

import (
	"os"
	"strings"
	"testing"

	"github.com/shai1801/ledger-recon/internal/model"
)

func TestLoadAdjustments_MiniFixture(t *testing.T) {
	f, err := os.Open("../../testdata/mini/mini_adjustments.jsonl")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

	adjustments, exceptions, err := LoadAdjustments(f, "mini_adjustments.jsonl")
	if err != nil {
		t.Fatalf("LoadAdjustments returned unexpected error: %v", err)
	}
	if len(exceptions) != 0 {
		t.Errorf("expected 0 exceptions, got %d: %+v", len(exceptions), exceptions)
	}
	if len(adjustments) != 1 {
		t.Fatalf("expected 1 adjustment, got %d", len(adjustments))
	}

	adj := adjustments[0]
	if adj.AdjustmentID != "ADJ-M1" {
		t.Errorf("AdjustmentID = %q, want %q", adj.AdjustmentID, "ADJ-M1")
	}
	if adj.AdjustmentType != model.Reversal {
		t.Errorf("AdjustmentType = %q, want %q", adj.AdjustmentType, model.Reversal)
	}
	if adj.ReferenceEntryID != "LE-M4" {
		t.Errorf("ReferenceEntryID = %q, want %q", adj.ReferenceEntryID, "LE-M4")
	}
	if adj.AmountCents != nil {
		t.Errorf("AmountCents = %v, want nil (REVERSAL has no amount)", *adj.AmountCents)
	}
	if adj.Currency != nil {
		t.Errorf("Currency = %v, want nil (REVERSAL has no currency)", *adj.Currency)
	}
}

func TestLoadAdjustments_MalformedLines(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "invalid JSON",
			line: `{"adjustment_id":"ADJ-1", not valid json`,
		},
		{
			name: "empty adjustment_id",
			line: `{"adjustment_id":"","adjustment_type":"REVERSAL","reference_entry_id":"LE-1","event_time":"2026-01-05T09:00:00Z","reason":"x","amount_cents":"","currency":""}`,
		},
		{
			name: "invalid adjustment_type",
			line: `{"adjustment_id":"ADJ-1","adjustment_type":"SIDEWAYS","reference_entry_id":"LE-1","event_time":"2026-01-05T09:00:00Z","reason":"x","amount_cents":"","currency":""}`,
		},
		{
			name: "empty reference_entry_id",
			line: `{"adjustment_id":"ADJ-1","adjustment_type":"REVERSAL","reference_entry_id":"","event_time":"2026-01-05T09:00:00Z","reason":"x","amount_cents":"","currency":""}`,
		},
		{
			name: "invalid event_time",
			line: `{"adjustment_id":"ADJ-1","adjustment_type":"REVERSAL","reference_entry_id":"LE-1","event_time":"not-a-date","reason":"x","amount_cents":"","currency":""}`,
		},
		{
			name: "non-numeric amount_cents",
			line: `{"adjustment_id":"ADJ-1","adjustment_type":"CORRECTION","reference_entry_id":"LE-1","event_time":"2026-01-05T09:00:00Z","reason":"x","amount_cents":"not-a-number","currency":"USD"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adjustments, exceptions, err := LoadAdjustments(strings.NewReader(tc.line), "test.jsonl")
			if err != nil {
				t.Fatalf("unexpected structural error: %v", err)
			}
			if len(adjustments) != 0 {
				t.Errorf("expected 0 adjustments, got %d", len(adjustments))
			}
			if len(exceptions) != 1 {
				t.Fatalf("expected 1 exception, got %d: %+v", len(exceptions), exceptions)
			}
		})
	}
}

func TestLoadAdjustments_CorrectionWithAmount(t *testing.T) {
	line := `{"adjustment_id":"ADJ-C-1","adjustment_type":"CORRECTION","reference_entry_id":"LE-1","event_time":"2026-01-05T09:00:00Z","reason":"fix","amount_cents":"29533","currency":"USD"}`

	adjustments, exceptions, err := LoadAdjustments(strings.NewReader(line), "test.jsonl")
	if err != nil {
		t.Fatalf("unexpected structural error: %v", err)
	}
	if len(exceptions) != 0 {
		t.Fatalf("expected 0 exceptions, got %d: %+v", len(exceptions), exceptions)
	}
	if len(adjustments) != 1 {
		t.Fatalf("expected 1 adjustment, got %d", len(adjustments))
	}

	adj := adjustments[0]
	if adj.AmountCents == nil {
		t.Fatal("AmountCents is nil, want a value")
	}
	if *adj.AmountCents != 29533 {
		t.Errorf("AmountCents = %d, want 29533", *adj.AmountCents)
	}
	if adj.Currency == nil || *adj.Currency != "USD" {
		t.Errorf("Currency = %v, want USD", adj.Currency)
	}
}
