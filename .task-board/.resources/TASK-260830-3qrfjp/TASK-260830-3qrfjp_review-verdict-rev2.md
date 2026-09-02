# TASK-260830-3qrfjp — Reviewer verdict (round 2): ACCEPTED

Change Request `CR-TASK-260830-3qrfjp-2` revision 2, base `c0673656`, candidate
tree `292c8cba`, `repository_delta=present`, 8 changed paths.

Worktree verified byte-identical to the candidate before review began and
byte-identical again after every mutant was reverted: each of the 8 paths was
compared with `git hash-object` against `git rev-parse 292c8cba:<path>`.

## What was attacked, not read

Nine mutants were applied to the production files in this worktree and reverted
by file copy (never `git checkout`). Seven killed; two were neutered by the
physics of the fixture and are reported as such rather than counted.

| # | Mutant | Production site | Result |
| --- | --- | --- | --- |
| A1 | Delete the staged-length clause entirely | `object_store.go:230` | KILLED — both subtests, `error = <nil>`; a **truncated file was installed into the immutable namespace** |
| A2 | Narrow `uint64(info.Size()) != actualSize` to `< actualSize` | `object_store.go:230` | KILLED — short case still refuses, over-long case admits: the class-coverage proof a delete-only mutant cannot give |
| A3 | Restore the pre-change merged condition (short write → quarantine) | `object_store.go:230` | KILLED — refusal returns as `ErrIntegrityMismatch` with a quarantined fragment; the durability-vs-mismatch reclassification is pinned |
| B1 | Narrow `!info.Mode().IsRegular() \|\| symlink` to symlink-only | `inspectExisting`, `object_store.go:315` | KILLED — `PutBlob` **hung**; failed at the 20s bound with "PutBlob did not return within 20s". The availability claim in the README reproduces exactly |
| C2 | `ensureOwnerDirectory` uses `os.Stat` instead of `os.Lstat` | `paths.go:487` | KILLED — both subtests `error = <nil>`; the immutable object is **installed outside the AX data root** through the symlinked shard |
| D2 | Route a failed rebuild through the corruption quarantine path | `openProjection`, `projection.go:240` | KILLED — "index-recovery holds 1 directories: the index was quarantined for a fault that is not corruption" |
| F1 | Remove the deferred `os.Remove(stagedPath)` | `object_store.go:187` | KILLED at 11 assertion sites, 5 of them added by this candidate |
| F2 | Classify a genuine absence as an unsafe entry | `inspectExisting`, `object_store.go:309` | KILLED by 18 subtests, including six of this candidate's **recovery halves** at `object_store_fault_test.go:247/291/366` and `object_store_fault_unix_test.go:109/147`. This is the round-1 finding closing: the retry halves are load-bearing, not decoration |
| C1 | Remove the `ModeSymlink` term from `verifyOwnerDirectoryInfo` | `paths.go:529` | NOT KILLED — and correctly so: `os.Lstat` already reports `IsDir()==false` for a symlink, so the term is redundant, not uncovered. Not a test gap |
| D1 | Commit instead of roll back on rebuild failure | `rebuildProjection` | NEUTERED — on an exhausted volume the commit itself raises `SQLITE_FULL`, so the mutant cannot express itself. Reported, not counted |
| E1 | Run migration DDL outside the transaction | `migrateProjection` | Fails (deadlocks on the single-connection pool) but is not a clean mutant. Reported, not counted |

## Producer claims independently reproduced

- The pre-change defect is real and the fix is the right one. A1 shows the
  pre-existing merged condition was the only thing between a truncated file and
  the immutable namespace, and A3 shows the old classification attached a
  full-source digest to a partial artifact in quarantine.
- The FIFO-at-0600 isolation argument holds. `mkfifoFixture` already routes
  through `requireIsolatedShapeFixture`, which asserts `verifyOwnerFileInfo`
  passes before production is driven, so the new cases inherit the package's
  anti-vacuity guard.
- `PRAGMA max_page_count` produces a genuine engine `SQLITE_FULL` and the limit
  is connection-scoped, so the recovery half of each projection case runs
  unconstrained. The helper fails loudly if the pragma does not apply, so the
  test cannot pass vacuously.
- Every property the README paragraphs claim was checked case by case against
  the assertions. All four properties (digest path absent, no staged residue,
  nothing manufactured into quarantine, retry installs the declared bytes) are
  asserted for each case in the list, and the two quarantine-move-denied
  outcomes are now stated separately with the opposite properties the tests
  actually assert. The round-1 self-contradiction is gone.

## Validation rerun by this reviewer at candidate tree 292c8cba

All commands run in this worktree after full mutant revert.

| Command | Result |
| --- | --- |
| `gofmt -l .` | no output |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | all 10 packages ok |
| `go test ./internal/localstore -race -count=1` | ok, 13.961s |
| `go test ./internal/localstore -cover -count=1` | ok, **86.6%** — matches the claim |
| `go test ./internal/localstore -count=3` (new cases) | ok — stable, no flake |
| `go run ./internal/traceability/cmd/tracecheck` | ok: contracts=60 sections=36 acceptance_cases=43 |
| `tracecheck -section 3.2 -section 3.3 -section 10.1 -section 10.2 -section 18.4` | ok, assigned_scopes=5 |
| `cataloggen ... -check` | exit 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | ok |
| `GOOS=windows GOARCH=amd64 go build ./...` | ok |
| `git diff --check` | exit 0 |

No lint configuration exists in the repository; lint is `gofmt` + `go vet`, both clean.

## Non-blocking observations

1. **Traceability registration.** The three new test files register none of their
   10 top-level tests in `internal/traceability/ownership.v0.5.0.json`. I
   initially read this as a DoD gap, because the immediately preceding leaf on
   the same acceptance case (`localstore-immutable-blob-install`, commit
   `327f0a9`) registered all five of its new tests. Measuring the package
   instead of inferring from one precedent shows registration is selective, not
   exhaustive: `projection_unix_test.go` is entirely unregistered (13 tests), as
   are 13 of 36 in `projection_test.go`, 3 of 14 in `paths_test.go`, and 1 in
   `object_store_test.go`. `tracecheck` verifies that registered declarations
   exist, not that every test is registered. Not a gap against the established
   convention. Worth registering opportunistically in a later leaf.
2. **README wording.** "Full and failing volumes are driven at every boundary of
   the durable write" then lists the group-accessible, symlinked and non-
   directory shard boundaries, which are not volume faults. The properties
   claimed for them are all true and asserted; only the framing sentence
   over-reaches. Also the SQLite paragraph now wraps mid-sentence ("...as if the
   refused one had never run. Connections use / WAL journal mode"). Cosmetic.
3. **`projectionHooks.afterDatabaseOpen`.** A new test-only production seam. It
   is unexported, nil on the `OpenProjection` production path, joins an existing
   hook family (`openBlob`, `beforeRebuildCommit`, `afterRebuildCommit`), and
   opens no bypass around any gate. The 2204 LOGBOOK claim "no new production
   seam was added to the object store" is accurate and correctly scoped — the
   seam is on the projection. Justified: it is the only way to make SQLite
   itself raise `SQLITE_FULL` from inside the transaction rather than
   substituting an error value.
4. **LOGBOOK mutant numbering** collides across the 2204 and 2237 entries — M4
   and M6 name different mutants in each. Each entry is internally consistent;
   only cross-entry reading is confusing.

## Verdict

Accepted. The production change is correct and pinned in both directions, every
gate this candidate touches was attacked and held, every README capability claim
reproduces, and the round-1 finding (recovery halves missing, README quantifier
contradicting the tests) is closed with assertions that demonstrably kill an
independent mutant. Repository-wide validation is green at the candidate tree.

Recorded with `accept_cr(TASK-260830-3qrfjp, revision=2, ...)`. No `commit_ack`
supplied: the orchestrator owns the commit and the final `done` transition.

## Resource-naming note

This round's verdict was first written over `TASK-260830-3qrfjp_review-verdict.md`
before I checked the board's naming convention, which is `_review-verdict-revN.md`
for later rounds. Round 1's verdict text is therefore no longer readable under
that name; board resource payloads are not git-tracked, so it is not recoverable.
Round 1's findings survive in full elsewhere and nothing about this task's history
is lost: they are quoted verbatim in LOGBOOK entry `2026-09-02 2237` and answered
point by point in `TASK-260830-3qrfjp_recovery-rework-report.md`. This file is the
round-2 verdict of record.
