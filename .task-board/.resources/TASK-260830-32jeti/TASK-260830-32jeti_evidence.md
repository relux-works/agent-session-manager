# TASK-260830-32jeti evidence: provider conformance harness

## Scope

Final leaf of STORY-260830-3jqsx1 (`§7.1–7.7`, `§8`,
spec `relux-works/agent-session-manager-spec@v0.5.0`). Leaves 1 and 2
accepted; this leaf builds on them. Work is uncommitted in the story
worktree per the Change Request shape; the board checkpoints the
signed commit.

## Production added (`internal/provhost`, no new package, no new imports)

- `opdecode.go` — parse-only helpers (never refuse; no wrapper shape).
- `manifest.go` — `DecodeManifest`, `Capabilities()` (7 ordered).
- `probe.go` — `DecodeProbe`, `CapabilityUsable`, `RequireCapability`.
- `profile.go` — `ProfileMapping` (§7.7, 6 providers × 2 profiles).
- `quiesce.go` — `DecodeQuiesceProof` returning the observed safe bit.
- `spawn.go` — `DecodeSpawnPlan` (launch/resume success, §7.7 tie-in).
- `identity.go` — `CheckIdentity`, `DecodeIdentifyResult` (§5.5).
- `idempotency.go` — `MutationOperations()`, `IdempotencyKey`
  (`IdempotencyKeyFor`), the sole `(operation, operation_id)` key.
- `doc.go`, `README.md` updated; `LOGBOOK.md` entry `0510` added
  (closes leaf 2's open derived-inventory item).

No new refusal constructors: plugin-body failures reuse `failProtocol`,
caller errors reuse `failInvalid`. `provhost` still has no importers.

## Inventory

- Header line corrected (constructor non-literals are fault conduits,
  not obligations).
- Census floor 60 → 162; 102 new arms each witnessed in
  `refusal_arm_operations_{a,b,c}_test.go`, registered in all three
  inventory checks.

## Derived complements (not samples)

- 6 providers × 2 profiles; 4 statuses × 2 enabled (probe rule, pure
  gate, one key) + per-key sweep over all 7 capabilities; 21
  fail-closed capability-by-status tuples with zero processes started.
- Keyed operations derived per-row from the §7.5 table
  (capture, materialize, commit, rollback, fork); the 10 unkeyed
  operations refused in the same test.
- Registries re-derived from the §7.3 example; §7.7 rows counted and
  quoted from the pinned document.
- Exact fixtures pass: §7.3 manifest, §7.4 probe, normative resume
  plan, §5.5 identity, prepared result; boundaries pinned both sides
  (128/129, 256/257, 1024/1025, 2048/2049, 4096/4097, 65536, 31/32
  token pre-existing, 8 MiB pre-existing).

## Mutant sweep (re-runnable: `TASK-260830-32jeti_mutant-sweep.py`)

26 narrowing mutants, each presence-asserted before the run,
full-package `go test ./internal/provhost/ -count=1` per mutant,
cp-aside restore byte-verified, tree hash equal before/after.

- First run: 24 killed, 2 survived (`M01`, `M02`: order checks
  skipping index 0 — no fixture distinguished the first element).
- Added first-element-substitution fixtures; re-run: both killed.
- Final: **26/26 killed, 0 survivors, 0 errors**.

## Gates (each run directly, real exit codes)

| Command | Exit |
|---|---|
| `go test ./... -count=1` (15/15 packages) | 0 |
| `go test ./internal/provhost/ -race -count=1` | 0 |
| `go test ./... -cover -count=1` (provhost 86.4%) | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `gofmt -l internal/` (empty) | 0 |
| `go run ./internal/traceability/cmd/tracecheck` (17/403, 74 cases, unchanged) | 0 |

## Bounds restated

`provhost` has no importers; no `cmd/`; leaf-2 survivors
(S11/S12/S13/U2) and the S1 one-off stand. New: receipts live in the
transaction document/plugin (host holds no cache); §8 cell values are
evidence labels (host pins fail-closed direction per tuple);
§8.2 exclusions, commit/rollback result bodies, and remaining §7.5
request vocabularies stay opaque; literal secrecy / shell-string argv
are structural, not content-admitted; `record_id` shape-checked, not
recomputed; opaque-identity prefix rule admits embedded paths
(same decidable-half bound as the object-identity story).
