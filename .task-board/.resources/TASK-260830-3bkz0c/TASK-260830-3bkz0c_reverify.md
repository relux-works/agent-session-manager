# TASK-260830-3bkz0c — independent re-verification (RUN-260906-a83bc9)

Successor run on the committed candidate `1296ecc` (tree
`292bc4242de751821d59ee1e3e68c9f1670b4634`, byte-identical to the
snapshotted tree; worktree clean; frozen `sessadapter`/`dirnode`/
`provider`/`provhost` production untouched per empty
`git diff d5ad5f6..HEAD` on those paths; `git verify-commit HEAD`
good; PR #35 head == `1296ecc`, OPEN). No production change by
this run — verification only. No new commit made, deliberately:
the tree stays byte-identical to the snapshotted candidate.

## Gates run directly by this run (no pipes, real exit codes)

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `gofmt -l internal/` (empty) | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `go test ./... -count=1 -p 2` (19 packages ok) | 0 |
| `go test ./internal/environ/ -race -count=1` | 0 |
| `go test ./internal/environ/ -cover -count=1` (76.1%) | 0 |
| `go run ./internal/traceability/cmd/tracecheck` (contracts=60 sections=36 acceptance_cases=94) | 0 |

## CR rev1 validation failure: root-caused, not fixed by code change

The CR suite stopped at command 4/18 (`go test ./... -count=1 -v`,
exit 1). This run re-executed that exact command and captured the
full log: the SOLE failure in the entire run is
`TestTransferManifestMaximumEntryGateIsLinear`
(`internal/canonicaljson/performance_test.go:36`):
"65,536-entry Transfer Manifest calculation gate took 3.51s,
want less than 2s". Every other package passes, including
`internal/environ` (7.3s) and both frozen leaves.

Why this is environmental, not a regression (four independent legs):

1. Same-tree green exists: the prior recovery run recorded
   `go test ./... -count=1` exit 0 (19 packages ok) on this exact
   tree, and this run recorded exit 0 at `-p 2`. Same code,
   different outcomes across runs = load-dependent.
2. Isolation passes with margin: the test alone takes 1.14s
   against the 2s budget; under full-suite parallel load it takes
   3.5–4.6s (3–4x slowdown = CPU contention signature).
3. Causation is impossible from this leaf: the test exercises
   only canonicaljson production plus stdlib; no file outside
   `internal/environ` imports environ (grep-verified empty), and
   the diff to the story base touches canonicaljson not at all.
   Per-package test binaries cannot be slowed by an unimported
   package — only by machine load.
4. The gate is a hard wall-clock budget (`elapsed >= 2s` fails)
   with no load normalization, in a package owned by another
   story. Loosening its budget or touching its code from this
   leaf would be gate-weakening to fit the box, not a fix.

A second timing-gated canonicaljson test
(`TestEveryProductionRefusalGuardIsExecuted`, spawns the shipped
suite under a child coverage profile) failed once under `-p 4`
and passes alone (62.8s) and in-package (112s) — same class.

Recommendation: re-run the CR validation suite; no leaf code
change is indicated. If the box stays saturated, `-p 2` passes
deterministically in this run's observation.

## Mutant battery re-executed by this run (all 7, harness `.temp/TASK-260830-3bkz0c/mut.sh`)

Baseline `/tmp/env-baseline` verified byte-identical to the tree
before the battery; tree verified restored-identical after.
Gate per mutant: `go test ./internal/environ/ -count=1`.

| Mutant | Narrowing | Named failing test | Result |
|---|---|---|---|
| N1 drop low-surrogate arm | lone lows | TestFrameAgreementAcrossFacades/bare_low_escape | KILLED |
| N2 StringLength counts bytes | 128-char/256-byte values | TestStringMeasureCountsRunes/environ_helper | KILLED |
| N3 tuple admits extensions | one unknown member | TestTupleAgreementAcrossFacades/extensions_refused | KILLED |
| N4 capabilities `<` instead of `!=` | ninth capability | TestDecodeEnvironmentObservation/ninth_capability_refused | KILLED |
| N6 env-id grammar admits `_` | underscore ids | TestSharedGrammarsAreOneLanguage | KILLED |
| T1 raw-scan surrogate gate, token preserved | escaped-backslash text | TestFrameAgreementAcrossFacades/escaped_backslash_{high,low,pair}_run2,_in_member_name,_nested (5 rows) | KILLED |
| D1 delete surrogate-gate call (existence only) | gate exists | TestFrameAgreementAcrossFacades/bare_high,low,high-followed-by-non-low,run3 | KILLED |

Applied 7, killed 7. Narrowing kills 6 of 6. Census-only kills 0.
Over-applied 0. Survivors: none — every survival bound is vacuous.

## Standing evidence (prior runs, unchanged, not re-claimed as mine)

- `TASK-260830-3bkz0c_boundary.md`: 7/7 AC rows driven through
  named production call sites; 4 live divergences ledgered.
- `TASK-260830-3bkz0c_mutation-log.md`: original battery log.
- `TASK-260830-3bkz0c_verify.md`: tree-mismatch repair + PR sync.
- Full-repo `-race` beyond `internal/environ` not run (exceeds
  one bounded call; same bound as the prior run).
