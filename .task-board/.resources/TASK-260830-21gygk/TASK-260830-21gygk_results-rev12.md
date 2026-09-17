# TASK-260830-21gygk rev12 handoff evidence

## Candidate and authority

- Task: `TASK-260830-21gygk`, role `developer`.
- Candidate base: `e119aaeede327580a470238fdb1ccf340a785b39`.
- Authority: `relux-works/agent-session-manager-spec` v0.6.0,
  `0cbdf100dbf84df50c64f792b1f940e3a67859a6`.
- Retained compatibility authority: v0.5.0,
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Candidate is intentionally uncommitted in the managed Story worktree.
  Pre-existing `.task-board` checkout changes were preserved and are not
  part of the product candidate.

## Delivered rework

The shared selector/plan/summary implementation now has one semantic
checkpoint admission path. `admitCheckpoint` validates session binding,
owning lease tuple, creator-holder, persistence variant, and event-head
closure, and returns the non-raw `admittedCheckpoint` capability. Profile
derivation accepts only that capability. Referenced checkpoints consumed by a
resume additionally satisfy temporal authority: the owning lease is at or
before the consuming resume lease, and event heads are in the resume's
predecessor closure. A later consumer may use an earlier checkpoint.

The old source-text census is replaced by a package-wide `go/types` gate.
Alternate raw paths in a new file, method receiver, closure factory, and
inferred loader are rejected by the same instrument. A small AST census stays
only as call/order precision evidence; it is not the semantic gate.

The implementation retains the existing UUID/name/qualified selector
grammar, no-fallback source semantics, ambiguity and collision handling,
canonical Lease Record identity, complete winning ancestry, lagging-copy
behavior, parked-source refusal, authority-union validation, seven
capability names, summary observations, stable bytewise sorting, immutable
SelectionPlan binding, old-plan revalidation, and recovery/read idempotency.
No CLI capability or unsupported observation is advertised.

## Acceptance accounting

**8 of 8 AC rows are driven through the shared-library production entries.**
The entries are `Reader.Resolve`, `Reader.List`, `Reader.Status`,
`Reader.AuthoritativeStatus`, `Reader.AuthoritativeList`,
`Reader.BuildPlan`, and `Reader.Revalidate`, with shared resolver/admission
helpers below them. **0 of 8 rows are delivered as an `ax` CLI surface**;
this is the explicit caller-integration bound for this task because no `ax`
session command exists in this tree. CLI wire exit mapping and lifecycle
effect authorization remain caller-owned.

## Validation evidence

All commands below ran against the final current source after the rev12
changes. Exit status is recorded from the command wrapper; the raw logs are
in this task directory.

| Check | Command | Exit | Evidence |
| --- | --- | ---: | --- |
| Temporal regression after fix | `go test ./internal/sessquery -run '^TestReview11ReferencedTemporalAuthority$' -count=1 -v` | 0 | `temporal-green-rev12.log`; 20 direct/task_board, ancestor, control, future-owner, and epoch-mismatch subtests passed. |
| Later consumer control | `go test ./internal/sessquery -run '^TestRev12ReferencedCheckpointLaterLease$' -count=1 -v` | 0 | `temporal-later-rev12-2.log`; direct and task_board controls passed. |
| Package-wide raw boundary | `go test ./internal/sessquery -run '^(TestRev11RecordConsumptionCensus|TestRev12RecordConsumptionCensusRejectsAlternatePaths|TestRev12AdmissionPrecisionCensus)$' -count=1 -v` | 0 | `census-rev12-green-2.log`; current source and all five alternate-path plants passed the expected census behavior. |
| Focused rev12 mutant harness | `bash .temp/TASK-260830-21gygk/mutation-rev12.sh` | 0 | `mutation-rev12/results.md` plus three mutant logs. Two qualifying narrowing mutants were behaviorally killed; harmless comment survivor passed. |
| sessquery package | `go test ./internal/sessquery -count=1 -v` | 0 | `sessquery-rev12-final.log`; full package passed. |
| sessrepo package | `go test ./internal/sessrepo -count=1 -v` | 0 | `sessrepo-rev12-final.log`; full package passed. |
| Repository tests | `go test ./... -v` | 0 | `go-test-all-rev12.log`; all packages passed. |
| Repository coverage | `go test ./... -cover` | 0 | `go-test-cover-rev12.log`; sessquery 86.5%, sessrepo 87.2%, sessstate 92.2%; all packages passed. |
| Build | `go build ./...` | 0 | `go-build-rev12.log`. |
| Vet | `go vet ./...` | 0 | `go-vet-rev12.log`. |
| Formatting | `gofmt -d internal/sessquery/*.go internal/sessrepo/checkpoint.go internal/sessrepo/lease.go internal/sessrepo/store.go internal/sessrepo/sessrepo_test.go internal/provhost/identity_test.go` | 0 | `gofmt-rev12.log`, empty diff. |
| Diff whitespace | `git diff --check` | 0 | `git-diff-check-rev12.log`. |

## Red-to-green and mutant evidence

Before the temporal fix, the same production-entry regression command failed
all four `future_owner` cases (direct/task_board × ancestor false/true), while
the valid and epoch-mismatch controls passed. Raw baseline:
`temporal-red-baseline-rev12.log`.

The final focused harness reports:

| Mutant | Result | What the applied mutant admitted | Behavioral kill |
| --- | --- | --- | --- |
| `temporal-narrowing` | `exit=1`, `KILLED` | A future-owned referenced checkpoint, while preserving the searched tokens | Named `future_owner` cases failed through `TestReview11ReferencedTemporalAuthority`; controls stayed green. |
| `record-boundary-narrowing` | `exit=1`, `KILLED` | A raw checkpoint path in `review12_new_file.go` | `TestRev12RecordConsumptionCensusRejectsAlternatePaths` failed for the admitted new-file path. |
| `harmless-survivor` | `exit=0`, `SURVIVED` | No product behavior; comment-only plant | Same instrument and controls passed, so it is correctly classified as harmless survivor. |

The previously established 63-plant battery (60 narrowing N plants and
3 ordering B plants) is reused as prior exact-source evidence. It was not
rerun in full in rev12 because the current defect scope was covered by the
new final-source targeted harness above. No compile failure or unapplied
plant is counted as a behavioral kill.

## Ownership and bounds

- `lease_record_id` remains the canonical digest of the validated winning
  Lease Record; no substitute digest or envelope fact is invented.
- Summary observation owners, CLI/publication, wire result mapping, recovery
  effects, fencing, commit, retry, remote dispatch/admission, transport
  resume, storage, and publication remain outside this leaf.
- The shared API performs no durable writes. `InspectLocal` is an internal
  diagnostic read for parked/tombstoned/recovery state, not a public summary
  fallback.
- CI evidence is local only; no hosted-CI result is claimed.
- `.agents/bin/curator` was unavailable in this worktree (`curator not found`,
  exit 127, logged in `curator-status-rev12.log`). `Skillfile.json` was not
  changed, so no Curator install was required for this candidate.

## Handoff artifacts

- `TASK-260830-21gygk_conformance-matrix-rev12.md`
- `TASK-260830-21gygk_relation-census-rev12.md`
- `mutation-rev12/results.md` and the three raw mutant logs
- all validation logs named above

The candidate is ready for independent review and remains uncommitted as
required by the managed Story handoff contract.

## Republish (rev13)

- Run: `RUN-260916-2a340e`.
- The managed base refresh left a stale `task-board.config.json` copy in the
  candidate. Its pre-restore diff contained only removal of the codex
  developer role override and the `gpt-5.6-luna` workload entries. The single
  explicit-path restore `git checkout HEAD -- task-board.config.json` was run;
  `git diff --quiet HEAD -- task-board.config.json` then returned exit `0`.
- The remaining candidate paths (`LOGBOOK.md`, `README.md`,
  `internal/provhost/identity_test.go`, `internal/sessrepo/*`, and
  `internal/sessquery/*`) remained present and unchanged by the restore.
- Republish quick gates all passed: `gofmt -l internal/` exit `0`,
  `go build ./...` exit `0`, `go vet ./...` exit `0`, and
  `go test ./internal/sessquery/ ./internal/sessrepo/ -count=1` exit `0`
  (`sessquery` 153.818s, `sessrepo` 4.757s). Raw logs are
  `gofmt-republish-rev13.log`, `go-build-republish-rev13.log`,
  `go-vet-republish-rev13.log`, and `go-test-republish-rev13.log` in the
  task scratch directory.
- No product implementation files were changed during republish. The
  candidate remains uncommitted for the managed Story handoff.
