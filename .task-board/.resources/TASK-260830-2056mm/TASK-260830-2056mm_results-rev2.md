# TASK-260830-2056mm results rev2 — race-gate rework

Rework run (RUN-260918-2c22a5) after autonomous recovery: rev1 CR validation
stopped at command 5/30 (`go test ./... -race -count=1 -timeout 25m`, exit 1).
Root cause was a single test-fixture data race; the fix is test-only and the
rev1 deliverable is otherwise carried verbatim.

Candidate: UNCOMMITTED working tree on branch
`task-board/story/STORY-260830-ptxkqe` atop checkpoint ca1c1d9.
No commit, rebase, or branch operation performed by this run.

Trunk: origin/main == c3aae73 at handoff (verified `git fetch origin main`;
merge-base == origin/main), so no `refresh-candidate` was needed.
`task-board.config.json` is byte-identical to HEAD (verified
`git diff HEAD --quiet -- task-board.config.json`). No validation command added.

Normative source: SPEC.v0.7.0 (unchanged from rev1).

## Root cause (evidence-backed)

`TASK-260830-2056mm_change-request_rev1-validation.log` lines 708-853:
`WARNING: DATA RACE` — goroutine 170 wrote `universe[extraID]` at
`internal/termbind/resolve_test.go:172` while goroutine 169 read the same map
in `mapUniverse.Lookup` (`fixtures_test.go:490`) via `ResolveEvidence`
(`resolve.go:86`); a second identical race at `resolve_test.go:181`.
`TestResolveEvidenceRequiresManifestAndProbe` built ONE `mapUniverse` and
shared it across four `t.Parallel()` subtests while the two "second" cases
extended it. The detector then failed the whole `termbind` package
(fifteen tests reporting "race detected during execution of test"); every
other package in the repo was green, including commands 1-4 of the suite.

Audit of the rest of the package: every other parallel subtest already builds
its own `universeMap(t, world)`; `TestResolveEvidenceShapeBounds` shares one
map read-only (safe); `Registry.AdmitProbe` is safe for concurrent use
(`Registry.Resolve` takes `mutex.RLock` and returns a clone,
`internal/terminalbackend/terminalbackend.go:519`). No production code races:
no production byte changed.

## Fix

`internal/termbind/resolve_test.go`, `TestResolveEvidenceRequiresManifestAndProbe`:
each of the four parallel subtests now builds its own `universeMap(t, world)`
copy before resolving; the mutating cases extend only their own copy.
Same assertions, same killer names, same behavior.

Whole-tree proof: reconstructed the rev1 candidate
(`git archive c3aae73` + apply `..._change-request_rev1.patch`, which is the
whole-story patch against trunk) and diffed recursively against this worktree.
Exactly two files differ — the fix hunk plus this run's LOGBOOK entry:

- `internal/termbind/resolve_test.go` (the fix, +7/-1 lines)
- `LOGBOOK.md` (rev2 entry, purely additive)

Full diff in the evidence tarball as `rev1-to-rev2.diff`. No production file,
no other test, no registry/README/traceability byte differs from rev1.

## Verification (this run)

Reran myself (logs in `TASK-260830-2056mm_producer-evidence-rev2.tar.gz`):

- `go test ./internal/termbind/ -race -count=10` → ok, exit 0 (plus two
  further `-count=3` passes, one saved as `termbind-race-count3.log`)
- `go test ./internal/axpane/ ./internal/terminstance/ -race -count=1` → ok/ok
  (`siblings-race.log`)
- `PYTHONDONTWRITEBYTECODE=1 python3 internal/termbind/mutant_harness.py` →
  30/30 on pass 1 and pass 2, verdict lists byte-identical
  (`termbind-harness-pass1.log`, `termbind-harness-pass2.log`); targeted
  verbose row `N-resolve-manifest-count` (whose killer is the edited test)
  still KILLED (`targeted-row-verbose.log`)
- `gofmt -l` clean, `go vet ./internal/termbind/` ok, `go build ./...` ok,
  config byte-identical, trunk pinned (`gates.log`)
- No `__pycache__` under the worktree (`PYTHONDONTWRITEBYTECODE=1` throughout)

Accepted from already-attached rev1 evidence (bytes unchanged, so results
carry): commands 1-4 and 6-30 of the configured suite, per-plant verbose
mutant logs, crash/idempotency logs, tracecheck section runs, digest
derivation. The full 30-command suite reruns automatically at handoff.

## Coverage and story-close (carried from rev1, unchanged)

- 60 of 60 AC rows driven through production entries by named committed
  tests — no test added, removed, or renamed, so
  `TASK-260830-2056mm_conformance-matrix.md` (rev1) and
  `internal/termbind/TRACEABILITY.md` stand without modification.
- Mutants: 28 narrowing + 1 labeled supplementary arm-delete + 1 harmless
  SURVIVED control, 30/30 twice (re-run above against the fixed tree).
- Registry `ownership.v0.7.0.json`, re-pinned digest, tracecheck numbers,
  README section, and the seven new acceptance cases are byte-identical to
  rev1; nothing re-derived because nothing they measure changed.

## Rejection-pattern self-check (delta)

(a) No expectation touched. (b) The edited killer still drives the composed
entry (`ResolveEvidence`, production call site `resolve.go:86` lookup path)
and still kills `N-resolve-manifest-count` — verified, not assumed.
(c) No landed gate touched. (d) No census added. (e) Full harness re-run
twice with identical verdicts, plus the verbose per-plant log for the row
whose killer changed. (f) "No production byte changed" is the measured
whole-tree diff, quoted above. Reported ratio (60/60) is rev1's measured
ratio on unchanged tests, not a new claim.
