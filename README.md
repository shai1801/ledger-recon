# ledger-recon

A CLI tool that reconciles ledger entries against bank settlements, applies
adjustments, produces reconciliation reports, and supports ad-hoc querying
of the results without standing up a database.

Built for the Viridien take-home assessment (ledger reconciliation, Go).

---

## 1. Architecture

The core of the system is a linear pipeline, with two entry modes on top of it:

```
                                          ┌──────────────┐
Load & Validate → Normalize → Adjust → Match → Persist ──┤ --reconcile  │
                                                          └──────────────┘
                                          ┌──────────────┐
                          query_store.jsonl ┤ --query      │
                                          └──────────────┘
```

Each stage has one narrow responsibility and passes clean data to the next:

1. **Ingest** (`internal/ingest`) — reads `accounts.csv`, `ledger_entries.csv`,
   `bank_settlements.csv`, `adjustments.jsonl`. Validates the *shape* of each
   row in isolation (required fields present, correct types, valid enums).
   Malformed rows become `Exception` records; they never crash the run.
2. **Normalize** (`internal/normalize`) — applies ledger-wide business rules
   that need more than one row at a time: sign conversion (CREDIT positive,
   DEBIT negative), deduplication by `entry_id`, and a currency cross-check
   against the account registry.
3. **Adjust** (`internal/adjust`) — applies `adjustments.jsonl`. `REVERSAL`
   synthesizes a new compensating entry; `CORRECTION` mutates the referenced
   entry's amount in place, keeping its original direction.
4. **Match** (`internal/match`) — pairs normalized (post-adjustment) ledger
   entries against eligible bank settlements, one-to-one, using the tie-break
   rules described below.
5. **Report** (`internal/report`) — builds and atomically writes all 7 output
   files (temp file + rename, so a crash mid-write never corrupts a
   previously-successful output).
6. **Query** (`internal/query`) — a hand-written lexer → recursive-descent
   parser → AST evaluator for the `--query` filter language, plus a loader
   for `query_store.jsonl`. Runs entirely against persisted output; does not
   require re-running reconciliation.

`internal/source` abstracts `--input` being either a plain directory or a
`.zip` archive, so every other package just asks for a file by name.

`internal/config` resolves every tunable parameter using
**CLI > configuration.json > hardcoded default** precedence.

---

## 2. Codebase hierarchy

```
ledger-recon/
├── cmd/ledger-recon/main.go   # CLI entrypoint: flag parsing, mode dispatch, wiring
├── internal/
│   ├── model/                 # plain structs shared by every package
│   ├── source/                # zip-or-folder input abstraction
│   ├── ingest/                # one loader per input file, shape validation only
│   ├── normalize/             # sign conversion, dedup, currency cross-check
│   ├── adjust/                # REVERSAL / CORRECTION application
│   ├── match/                 # ledger-to-settlement matching + tie-break
│   ├── config/                # configuration.json parsing + precedence resolver
│   ├── report/                # atomic writers for all 7 output files
│   └── query/                 # lexer, parser, AST, evaluator, query_store I/O
├── testdata/mini/              # bundled mini fixture + expected_summary.json
└── sample-output/              # output of a real --reconcile run (committed)
```

---

## 3. Data model

| File | Purpose | Key fields |
|---|---|---|
| `accounts.csv` | Account registry | `account_id` (unique), `currency`, `opening_balance_cents`, `status` |
| `ledger_entries.csv` | Raw ledger movements | `entry_id` (should be unique — is not, by design, see §6), `direction`, `amount_cents` (always positive on disk), `posted_at` |
| `bank_settlements.csv` | Bank-side settlement records | `settlement_id` (unique), `bank_ref`, `amount_cents` (signed), `status` (`SETTLED`/`PENDING`/`RETURNED`) |
| `adjustments.jsonl` | REVERSAL/CORRECTION instructions | `adjustment_type`, `reference_entry_id`, `amount_cents`/`currency` (strings, empty for REVERSAL) |

Money is always `int64` cents, never a float. Timestamps are parsed to
`time.Time` at ingest time; nothing downstream touches a raw date string.

---

## 4. Matching algorithm

Per the assessment spec, a match is one-to-one and requires **all** of:

- `bank_ref == entry.txn_id`
- `entry.account_id == settlement.account_id`
- `entry.currency == settlement.currency`
- `abs(entry.amount_cents - settlement.amount_cents) <= tolerance_cents`
- `settlement.settled_at` is on or after `entry.posted_at`, and no later than
  `entry.posted_at + matching_window_days` (**forward-only** — a settlement
  dated before posting does not qualify)

When more than one ledger row qualifies for a given settlement, the winner is
chosen by: **smallest amount difference → earliest `posted_at` → smallest
`entry_id`** (all three deterministic, so re-running on the same input always
produces the same output).

`SETTLED` settlements are always eligible. `PENDING`/`RETURNED` are only
eligible if `include_pending_settlements` is true; otherwise they go straight
to `unmatched_settlements` without an attempt to match.

**Why `bank_ref` alone is not sufficient**: the real dataset contains 6 pairs
of settlement rows sharing an identical `bank_ref` but differing in
`account_id` and `currency` — almost certainly coincidental collisions between
independent bank reference-numbering schemes, not the same transaction. The
full 4-condition check resolves every one of these correctly; requiring only
`bank_ref` would not.

---

## 5. Parameter precedence

Every tunable resolves as **CLI flag → `configuration.json` → hardcoded
default**, via `internal/config.Resolver`. Defaults (used only when neither
CLI nor config supplies a value):

| Parameter | Default |
|---|---|
| `tolerance_cents` | `0` |
| `matching_window_days` | `0` |
| `output_format` | `jsonl` |
| `partition_by` | `""` (no partitioning) |
| `include_pending_settlements` | `false` |
| `query_limit` | `100` |

These defaults are not specified by the assessment ("default or open bounds")
and are a documented assumption, not a certainty.

---

## 6. Assumptions

The spec leaves several implementation choices genuinely open. Each is
resolved as follows, and flagged here rather than hidden in code:

1. **Deduplication policy** (`entry_id` is not actually unique in the source
   data — 2 of 128 rows collide): **first occurrence in file order wins**;
   every later duplicate becomes a `duplicate_entry` exception and is
   excluded from downstream processing.
2. **Currency mismatch / unknown account** on a ledger entry: the entry is
   **excluded** from further processing (not silently carried through with a
   wrong currency assumption), logged as `currency_mismatch` /
   `unknown_account`.
3. **REVERSAL's synthetic entry** inherits the referenced entry's `txn_id`,
   meaning it *can* participate in matching. Its `entry_id` is set to the
   originating `adjustment_id` for traceability.
4. **Settlement processing order** during matching: sorted by `settlement_id`
   ascending. The spec's tie-break rule is defined per-settlement (which
   ledger row to choose), not across settlements — so this implementation is
   a **greedy, order-dependent** match, not a globally-optimal assignment.
   In principle a different processing order (or a true optimal bipartite
   matching) could produce different results if one ledger entry is a valid
   candidate for two different settlements; this did not occur in the
   provided dataset upon inspection, but is not guaranteed in general.
5. **Dynamic filters** (`currency_eq`, `amount_cents_min`, etc.): treated as
   a generic `field_suffix` grammar (`_eq`, `_ne`, `_min`, `_max`, `_gt`,
   `_gte`, `_lt`, `_lte`), not a fixed enum — so any field name works with any
   of these suffixes. Resolved filters are combined with the explicit
   `--query` expression via `AND`.
6. **`query_store` schema**: a single JSONL file containing every table's
   rows, each tagged with a `_table` discriminator field. `--table` filters
   by that tag before the query expression runs.
7. **Output formats**: only `json` and `jsonl` are implemented. The
   assessment's own `configuration.json` uses `jsonl`, and no CSV column
   layout is specified anywhere for these particular tables.

---

## 7. Testing methodology

- **Unit tests per `ingest` loader**: happy path against `testdata/mini`,
  plus a table-driven suite of malformed-row cases (empty required fields,
  invalid enums, non-numeric amounts, invalid timestamps, wrong column
  counts) and a missing-required-column structural-error case.
- **`normalize`/`adjust`/`match`**: validated by hand against
  `testdata/mini/mini_expected/expected_summary.json`, tracing every input
  row through every pipeline stage to confirm the final counts match exactly.
- **`query`**: lexer/parser/evaluator tested independently of file I/O.
- **Full-dataset smoke test**: `--reconcile` run against the complete
  `DataMag_LedgerReconciliation_Data` set, with every summary count
  hand-traced back to root causes (e.g. the 2 "missing" rows between
  `valid_ledger_rows_after_dedupe` and `normalized_ledger_entries` were
  traced to one `unknown_account` and one `currency_mismatch` exception,
  confirming zero silent data loss anywhere in the pipeline).

---

## 8. Usage

### Reconcile mode

```
$ ledger-recon --reconcile --input "D:\Viridien\Assessment" --output sample-output
wrote reconciliation reports to sample-output
```

`sample-output/summary.jsonl`:
```json
{"input_accounts":30,"input_ledger_rows":128,"input_settlements":104,"input_adjustments":6,"valid_ledger_rows_after_dedupe":122,"duplicate_ledger_rows":2,"invalid_ledger_rows":2,"adjustments_applied":4,"normalized_ledger_entries":124,"matched_transactions":87,"unmatched_ledger":37,"unmatched_settlements":17,"unknown_account_ledger_rows":1,"currency_mismatch_ledger_rows":1}
```

### Query mode

```
$ ledger-recon --table accounts --query "currency='USD' AND delta_cents!=0" --output sample-output
```
```json
[
  {
    "_table": "accounts",
    "account_id": "ACC-1001",
    "account_type": "SAVINGS",
    "closing_balance_cents": 211139,
    "currency": "USD",
    "customer_id": "CUST-2007",
    "delta_cents": 156389,
    "opening_balance_cents": 54750,
    "status": "ACTIVE"
  },
  {
    "_table": "accounts",
    "account_id": "ACC-1016",
    "account_type": "CHECKING",
    "closing_balance_cents": 636126,
    "currency": "USD",
    "customer_id": "CUST-2017",
    "delta_cents": 510126,
    "opening_balance_cents": 126000,
    "status": "ACTIVE"
  }
]
```

---

## 9. Known limitations

- Matching is greedy over settlements sorted by `settlement_id`, not a
  globally-optimal bipartite assignment (see assumption 4).
- Adjustment chaining (an adjustment referencing an entry synthesized by an
  earlier `REVERSAL`) is not supported — it surfaces as `missing_reference`.
- CSV output format is not implemented, only `json`/`jsonl`.
