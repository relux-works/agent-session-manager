# TASK-260830-21gygk recovery outcome (RUN-260910-217ee3)

Recovery developer handoff. No implementation changes made. Predecessor
RUN-260909-b4446d rev3 rework candidate preserved uncommitted in the managed
Story worktree. Scope pinned AX v0.6.0: actual lease-record identity,
union/head derivation, required summary facts, truthful mutation evidence.

## Verified (this run, observed)

- Candidate present and uncommitted (`git status --short`, excluding
  `.task-board`): `M LOGBOOK.md, README.md,
  internal/provhost/identity_test.go, internal/sessrepo/sessrepo_test.go,
  internal/sessrepo/store.go` plus untracked `internal/sessquery/` and
  `internal/sessrepo/lease.go`. No commit, no staging, no reset performed.
- Candidate markers present: `AttestLeaseRecord` in
  `internal/sessrepo/lease.go`; `compareLeaseTuple`/`checkUnionCopy` in
  `internal/sessquery/`; summary required-fact markers
  (`owner_host_name`, `process_present`) in `internal/sessquery/summary.go`.
- Board evidence present via `task-board resource get`: outcome
  `TASK-260830-21gygk_results-rev3.md` (rev3 rework: real lease records,
  greatest-winner union, closed summaries, 43 KILLED), outcome
  `TASK-260830-21gygk_results-rev3-addendum.md` (Notes heading typo
  correction), precondition `TASK-260830-21gygk_rework-rev3.md` (rev3 rework
  brief). Old immutable CR3 not reviewed or accepted.
- Mutant battery manifest re-derived from file (not rerun):
  `.temp/TASK-260830-21gygk/mutants-rev3/mutants.json` = 47 entries:
  43 KILLED, 2 CONTROL (exit 0), 1 NOT_APPLIED, 1 COMPILE_OR_HARNESS_FAILURE.
  Matches outcome claim 40N+3B=43 ALL KILLED with controls distinguished.
- Review evidence tar present:
  `.temp/TASK-260830-21gygk/TASK-260830-21gygk_review-evidence-rev3.tar.gz`
  lists verdict, regression logs, mutation-check logs, candidate manifest.
- Cheap live checks (delta-appropriate, not a battery rerun):
  `gofmt -l internal/` clean (exit 0);
  `go vet ./internal/sessquery/ ./internal/sessrepo/` clean (exit 0);
  `go test ./internal/sessquery/ -count=1 -run TestRev3RealLeaseRecordIdentity`
  PASS.

## Reused (not rerun)

- Prior full validation from rev3 outcome: `go test ./... -count=1` PASS all
  packages; cover 87.9%/87.4%/86.0% (sessquery/sessrepo/provhost); sessquery
  verbose 55 top-level PASS; full 40N+3B narrowing battery with control
  before/after green. Reused as attached evidence; full suite and mutation
  battery intentionally not repeated merely to exit.
- Prior AC/negative-gate coverage claims in `results-rev3.md` reused as
  producer evidence for the reviewer; this recovery asserts only presence and
  the spot checks above.

## Handoff

Ready for review. Candidate left UNCOMMITTED for managed-runtime Change
Request construction. Reviewer will be a separate Codex gpt-6-astra medium
run against the new CR identity/tree, never old CR3.
