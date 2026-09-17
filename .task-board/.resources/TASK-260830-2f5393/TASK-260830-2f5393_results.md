# TASK-260830-2f5393 results — implement lease record validation and CAS

Authority: relux-works/agent-session-manager-spec v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6` (Sections 2.2, 5.3,
13.6-13.10). Historical v0.5.0 (`28bf96d`) behavior retained.

## Outcome

Ready for review. 8 of 8 AC rows driven through production entries by
named committed tests (ratio `8 of 8`; call site named per row in
`internal/sessrepo/TRACEABILITY.md`). Public CLI coverage is 0 of 8 by
stated ownership bound: no `ax` command exists in this tree, and the
shared-library entries are the deliverable.

## What changed

- `internal/sessrepo/lease_store.go` (new, ~800 lines): `CreateLease`,
  `CompareAndSwapLease`, `GetLease`, `ListLeases`, `WinningLease`,
  `VerifyFencingToken`, `CheckFencingExpiry`, `CompareLeaseTuple`, plus
  7 new refusal sentinels. Lease blobs are immutable content-addressed
  files installed stage-then-rename; no lease index exists to skew.
- `internal/sessrepo/sessrepo.go`: `AfterLeaseStage` crash-seam hook
  (nil in production).
- `internal/sessrepo/lease.go`: comment records the mint-confirm funnel.
- `internal/sessrepo/lease_store_test.go`, `lease_crash_test.go` (new):
  positive, negative, idempotency, injector-crash, and real-SIGKILL
  suites.
- `internal/sessrepo/census_test.go`: 7 new boundary markers; equality
  ledger grows to 3 rows (new lease site + line-shifted resume row).
- `internal/sessrepo/testdata/mutate.py` (new): shipped narrowing
  battery, 27 N + 1 T + 3 controls.
- `internal/sessquery/lease_store_adopt_test.go` (new): adoption +
  tuple-agreement proof (query admission unchanged).
- `internal/sessrepo/TRACEABILITY.md` (new), `internal/sessrepo/doc.go`,
  `README.md`, `LOGBOOK.md`: evidence, scope, and log updates with no
  CLI or capability claims.

## Key decisions

1. No second lease model: shape/identity stay with `canonicaljson`,
   the tuple rule stays with `sessstate.Compare` (restated across the
   import cycle, agreement-pinned), `winningLeaseFor` unchanged.
2. Expiry policy is grant-expiry, never lease-expiry (Section 5.3:
   "no time-expiring ownership lease"); pinned by an ancient-`created_at`
   test. Refusal is `fencing_grant_expired`, never `lease_expired`.
3. Mint failures are plain operational errors, never refusals — a
   weakened grammar gate fails closed at mint with a changed class,
   which is what kills its narrowing mutant.
4. CAS replays re-mint against the pre-commit basis, not the head
   (fixed a real epoch+2/token-reuse bug found by the idempotency test).
5. The provhost 5-site attestation bound is satisfied without rewriting
   it: the mint confirm routes through `AttestLeaseRecord`.

## Validation (exit codes observed)

- `gofmt` clean, `go build ./...`, `go vet ./...`, `git diff --check`: 0
- `go test ./... -count=1`: 0 (all packages; provhost bound holds 5/5)
- `go test ./... -race -count=1`: 0 (26/26 packages, log in tarball)
- `go test ./... -cover`: 0 (sessrepo coverage: 86.6% of statements)
- 13 configured fuzz smokes (100x each): all ok
- `tracecheck`: ok (contracts=63 sections=36 cases=101 fixtures=32)
- `cataloggen -check`: 0
- `GOOS=linux/windows go build ./...`: 0
- JSON sweep over tracked `*.json`: 0
- `task-board validate`: exit 0 (195 board-wide advisory issues, none
  on this task or story)
- Mutant battery `mutants-03` on final source: exit 0 — 27/27 applied
  plants KILLED (26 narrowing, one per new `refuse` site, + 1
  token-preserving); harmless control SURVIVED; NOT_APPLIED and
  COMPILE_OR_HARNESS_FAILURE controls classified separately.
  Per-plant raw logs in the evidence tarball.

## Source integrity for the battery

- `sha256 internal/sessrepo/lease_store.go` at battery time:
  `bdb89cb607268d1d06ac2eb235f4e1a22017d3da61899203be26d15153cac6c8`
- `sha256 internal/sessrepo/testdata/mutate.py` at battery time:
  `a9d795ce50ae4d17c746a9c66bc9256a37902b82f0a76d78a3e8d2ee07290e5c`
- The battery asserts the mutation copy restores every byte
  (`control-after.log` runs the full package green after restoration).

## Handoff state

- Work left UNCOMMITTED in the managed Story worktree for snapshot.
- No `ax` command, doctor result, or runtime capability added.
