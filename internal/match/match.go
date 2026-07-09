package match

import (
	"sort"
	"time"

	"github.com/shai1801/ledger-recon/internal/model"
)

type MatchedPair struct {
	Entry           model.NormalizedEntry
	Settlement      model.BankSettlement
	AmountDiffCents int64
}

func Match(
	entries []model.NormalizedEntry,
	settlements []model.BankSettlement,
	tolerenceCents int64,
	windowDays int,
	inclludePendingSettlements bool,
) (matched []MatchedPair, unmatchedLedger []model.NormalizedEntry, unmatchedSettlements []model.BankSettlement) {
	eligible, ineligible := splitByEligibility(settlements, inclludePendingSettlements)
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].SettlementID < eligible[j].SettlementID })

	usedEntry := make(map[string]bool, len(entries))

	for _, s := range eligible {
		candidates := findCandidates(entries, s, tolerenceCents, windowDays, usedEntry)

		if len(candidates) == 0 {
			unmatchedSettlements = append(unmatchedSettlements, s)
			continue
		}

		best := pickBest(candidates)
		usedEntry[best.entry.EntryID] = true
		matched = append(matched, MatchedPair{
			Entry:           best.entry,
			Settlement:      s,
			AmountDiffCents: best.diff,
		})
	}

	//Ineligible settlement (PENDING/RETURNED, when not opted in ) never
	//attempt matching - they're reported as unmatched directly..

	unmatchedSettlements = append(unmatchedSettlements, ineligible...)

	for _, e := range entries {
		if !usedEntry[e.EntryID] {
			unmatchedLedger = append(unmatchedLedger, e)

		}
	}

	return matched, unmatchedLedger, unmatchedSettlements
}

func splitByEligibility(settlements []model.BankSettlement, includePending bool) (eligible, ineligible []model.BankSettlement) {
	for _, s := range settlements {
		if s.Status == "SETTLED" || includePending {
			eligible = append(eligible, s)
		} else {
			ineligible = append(ineligible, s)
		}

	}
	return eligible, ineligible
}

type candidate struct {
	entry model.NormalizedEntry
	diff  int64
}

func findCandidates(
	entries []model.NormalizedEntry,
	s model.BankSettlement,
	toleranceCents int64,
	windowDays int,
	usedEntry map[string]bool,

) []candidate {
	var candidates []candidate
	for _, e := range entries {
		if usedEntry[e.EntryID] {
			continue
		}

		if e.TxnID != s.BankRef {
			continue
		}

		if e.AccountID != s.AccountID {
			continue
		}
		if e.Currency != s.Currency {
			continue
		}

		diff := e.AmountCents - s.AmountCents
		if diff < 0 {
			diff = -diff
		}
		if diff > toleranceCents {
			continue
		}

		if !withinWindow(e.PostedAt, s.SettledAt, windowDays) {
			continue

		}

		candidates = append(candidates, candidate{entry: e, diff: diff})
	}
	return candidates
}

// I am implementing the withinWindow logic - "settled_at within matching_window_days after posted_at "
//i.e. settled_at is on posted_at or after posted_days and no later than posted_date+windowDays

func withinWindow(postedAt, settledAt time.Time, windowDays int) bool {
	if settledAt.Before(postedAt) {
		return false
	}

	deadline := postedAt.AddDate(0, 0, windowDays)
	return !settledAt.After(deadline)
}

//I am writing the logic to pick the best match based on the tie breaker rule provided in the assessment document i.e
//smallest amount difference, then earliest posted_at, then entry_id.

func pickBest(candidates []candidate) candidate {
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.diff != b.diff {
			return a.diff < b.diff
		}

		if !a.entry.PostedAt.Equal(b.entry.PostedAt) {
			return a.entry.PostedAt.Before(b.entry.PostedAt)
		}

		return a.entry.EntryID < b.entry.EntryID
	})
	return candidates[0]
}
