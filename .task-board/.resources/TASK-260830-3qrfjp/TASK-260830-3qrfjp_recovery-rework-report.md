# TASK-260830-3qrfjp — recovery-half rework (review round 2)

Scope of this round: the ONE blocking defect the reviewer named, and nothing else.
No production file was touched. `object_store.go`, `projection.go`,
`projection_test.go` and `projection_fault_test.go` are byte-identical to the
reviewed candidate; the only production diff in the worktree remains the
accepted 17-line staged-length split from round 1.

## The defect

`README.md` claimed seven fault boundaries each carry four properties — digest
path absent, no staged residue, nothing manufactured into quarantine, and a
retry after the fault clears installing the declared bytes. Only three did.
Worse, for the existing-quarantine-denied case two of the four are false BY
DESIGN and the tests correctly assert the opposite (the corrupt object stays at
the digest path; the candidate IS preserved), so the section contradicted itself
across two adjacent sentences.

Same root cause as the miss against front-loaded DoD item 4: both
quarantine-ENOSPC subtests are full-disk cases that asserted only the failure
state. Anything can fail. A durability suite has to establish what survives.

## Fix — the stronger one, not the prose narrowing

Recovery halves added to six cases (five named by the reviewer, plus the
symlinked shard for uniformity so the rewritten README list has no member
missing a property):

| Case | Recovery now asserted |
| --- | --- |
| `TestPutBlobRefusesStagedModeDriftBeforeInstallingIt` | drift clears, retry installs the declared bytes |
| `...RefusesGroupAccessibleObjectShardBeforeStagingAnything` | quarantine proven empty; shard repaired, retry installs |
| `...RefusesSymlinkedObjectShardWithoutFollowingIt` (both subtests) | symlink removed, retry installs |
| `...RefusesSpecialFileAtObjectShardBeforeStagingAnything` | FIFO removed, retry installs |
| `...RefusesSpecialFileAtDigestPathWithoutOpeningOrMovingIt` | FIFO removed, retry installs |
| `...ReportsQuarantineFailures...` / candidate subtest | volume regains room, the SAME mismatch quarantines normally and still does not install |
| `...ReportsQuarantineFailures...` / existing subtest | a later run reads the still-corrupt object, proves the disagreement, quarantines BOTH artifacts, frees the digest path; a third run then installs |

The last two carry recovery in a different SHAPE, and each case now says so at
the case rather than leaving the reader to infer a uniform one. A candidate that
was never a valid object cannot install on retry. A corrupt existing object is
deliberately left at the digest path, so its recovery is what the next run does
with it still there — a path nothing pinned before.

README quantifier rewritten: the six recoverable boundaries keep the four-property
sentence, and the quarantine-move-denied outcome is stated separately with what is
actually asserted for each of its two artifacts.

## Negative evidence — 5 mutants, 5 killed

Full transcript: `TASK-260830-3qrfjp_recovery-mutation-evidence.log`. Each mutant is
applied to the production file, the test is run SCOPED with `-run` so only that
test can kill it, then the file is restored from a byte-for-byte backup and the
restore is verified by sha256 (all five report `restored ok: yes`).

Three of the five are the ones that matter: each leaves the FAULTED first half of
its case PASSING and dies only at a line added in this round.

- **M5** — the existing artifact IS moved but `result.ExistingQuarantinePath` is
  never set. First half passes (it errors before that line is reached). Killed at
  `object_store_fault_test.go:490`, `want both artifacts preserved`.
- **M6** — the mismatched candidate IS quarantined but its path is dropped from
  the result. First half passes (quarantine is denied there, so the line is
  unreachable). Killed at `object_store_fault_test.go:434`,
  `want the mismatched candidate preserved`.
- **M4** — a genuine ABSENCE classified as an unsafe entry, the absence-versus-
  failure-to-read distinction seen from the recovery side. Killed by ALL SIX
  retry halves, every one at a new line: `:247`, `:291`, `:366` (both symlink
  subtests), `object_store_fault_unix_test.go:109`, `:147`.
- **M1** — existing quarantine path reported without moving the object. Killed,
  but at pre-existing line `:463`; a whole-test bind, not a recovery bind.
- **M2** — mismatch gate narrowed to size only. Killed at pre-existing `:410`.

Reported honestly: the `assertQuarantineIsEmpty` added to the group-accessible
shard case is a state assertion backing a README claim, NOT a gate proof. That
refusal returns before anything is staged, so no natural mutant can put something
in quarantine there.

## Validation — every command a standalone process, real exit code

| Command | Exit |
| --- | ---: |
| `test -z "$(gofmt -l $(git ls-files ... '*.go'))"` | 0 |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | 0 |
| `go test ./... -race -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `cataloggen ... -check` | 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| `git diff --check` | 0 |

`internal/localstore` unchanged at **86.6%**, **90** top-level tests, **130**
subtests — this round added assertions to existing cases, not new cases.

NOT RERUN this round, and stated rather than implied: the five seeded fuzz
targets, `git ls-files '*.json'` JSON parse, and `task-board validate`. This
round changed two `_test.go` files in `internal/localstore`, `README.md` and
`LOGBOOK.md` only; none of those three gates reads any of them. The reviewer's
own instruction was that nothing outside the localstore suite and coverage is
affected. Their round-1 results stand.

Windows remains skipped with the rest of the package pending the owner-DACL
platform task. Real filesystem exhaustion is still deliberately not driven, for
the reason recorded in the LOGBOOK.
