# TASK-260830-3qrfjp — storage crash and filesystem fault boundaries

Scope: `relux-works/agent-session-manager-spec@v0.5.0` (`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`),
§3, §10.1–10.2, §18.4. Work confined to the
`STORY-260902-2ni6i3 / object-store-fault-boundaries` boundary.

Branch: `task-board/story/STORY-260902-2ni6i3`, worktree
`.temp/STORY-260902-2ni6i3/worktree`. Changes are left uncommitted for the
Story checkpoint step; no rebase, merge or trunk landing was performed.

## What changed in production code

### 1. An incomplete staged write is a durability failure, not a proven mismatch

`internal/localstore/object_store.go` — `PutBlob` previously folded the
staged-length comparison into the same condition as the two source-identity
comparisons:

```go
if info.Size() < 0 || uint64(info.Size()) != actualSize || actualSize != expectedSize || actualDigest != expected {
    // quarantine the candidate, return ErrIntegrityMismatch
}
```

The length term and the identity terms answer different questions. The identity
terms compare what the *source* produced against what the caller *declared*; a
disagreement there is the §3.2 quarantine trigger. The length term compares what
the *copy accepted* against what the *filesystem kept*; a disagreement there
means the write did not complete and proves nothing about the source bytes at
all. On a volume with delayed allocation (ext4, XFS, APFS) that is precisely how
ENOSPC surfaces: the accepting writes succeed and the shortfall appears at
writeback.

Folding them together had two consequences:

- a partially written fragment was installed into quarantine and reported with
  `PutResult.Digest`/`Size` describing bytes that no artifact on disk actually
  held — the digest of the full source, attached to a truncated file;
- a full volume caused the store to write *more* data, into quarantine, on the
  volume whose exhaustion caused the fault.

The condition is now split. A staged-length disagreement returns `ErrDurability`
with no quarantine and no reported identity; the fragment is discarded by the
existing deferred cleanup. This is the exact mirror of the rule
`verifyBlobContent` already owns for reads ("an operation that did not complete
is never a proven mismatch"), applied to the write side.

### 2. Test-only projection hook: `projectionHooks.afterDatabaseOpen`

`internal/localstore/projection.go` — one field added to the existing test-only
`projectionHooks` struct, invoked in `openProjectionDatabase` after the
connection is configured and verified. It exists so a test can constrain that
exact connection and make SQLite raise a genuine `SQLITE_FULL` from inside the
migration and rebuild transactions. `OpenProjection`, the production entry
point, passes a zero `projectionHooks` and never reaches it.

## What changed in tests

New files, all driving the real production entry points `PutBlob` and
`OpenProjection`. No existing test was weakened or removed; the only edit to an
existing test file is the `openProjectionDatabase` call-site signature in
`projection_test.go`:

| File | Cases |
| --- | ---: |
| `internal/localstore/object_store_fault_test.go` | 6 tests, 12 cases |
| `internal/localstore/object_store_fault_unix_test.go` (`darwin \|\| linux`) | 2 tests |
| `internal/localstore/projection_fault_test.go` | 2 tests |

Coverage per fault dimension named in the task description:

| Dimension | Case |
| --- | --- |
| object writes | source stops producing mid-copy; volume refuses the accepting write; volume discovers exhaustion at writeback |
| short writes | filesystem kept less than the copy accepted; filesystem kept more than the copy accepted |
| full disk | all of the above, plus quarantine moves that cannot be written, plus `SQLITE_FULL` in migration and in rebuild |
| SQLite rebuilds | real engine `SQLITE_FULL` inside the rebuild transaction; prior index keeps its inode, rows and metadata count; no `index-recovery` directory; next open converges on all 151 objects |
| SQLite migrations | real engine `SQLITE_FULL` inside the migration transaction; `user_version` stays 0, `sqlite_master` stays empty, next open migrates and rebuilds cleanly |
| permissions | staged file mode drifts to `0644` between flush and install; object shard root becomes group-accessible before staging |
| symlinks | object shard root and object shard replaced by a symlink to an external directory, refused without following it and with nothing written outside the data root. The pre-existing `TestPutBlobRefusesUnsafeDigestPathWithoutFollowingOrQuarantiningSymlink` covers the leaf, which `inspectExisting` classifies; these cover the directory the staged file would be created in, which the owner-only child-tree walk decides much earlier |
| special files | FIFO at the digest path; FIFO at the object shard |

Every write-side fault case asserts the same three recovery properties: the
immutable digest path stays absent, the shard directory keeps no staged residue,
and a retry after the fault clears installs the exact declared bytes.

### Seam honesty

Write-side faults are driven through the pre-existing
`storeOperations.syncFile` seam rather than a new one. That is where a real
ENOSPC is discovered on a delayed-allocation filesystem, so a failure raised
there — and a length that changed by the time the staged file is stat'd — are
the two shapes the condition actually takes in production. No new production
seam was added to the object store.

The projection full-volume cases use `PRAGMA max_page_count` pinned to the
database's current page count. SQLite then returns result code 13,
`database or disk is full`, from inside the transaction: the same code and the
same message a genuinely exhausted filesystem produces. The limit lives on the
connection, not in the file, so it disappears when the failed open closes it and
the recovery half of each case runs unconstrained.

### Special-file cases bound availability, not only safety

A directory at the digest path already reached the regularity clause in
`inspectExisting`, but did not isolate it: with the clause removed, `os.Open`
succeeds and the read fails with `EISDIR`, so the unreadable verdict still
refuses. A FIFO at exactly mode `0600` isolates it — not a symlink, owned by the
effective user, already owner-only, so every preceding check passes — and with
the clause removed `os.Open` blocks forever on a FIFO with no writer. Both new
FIFO cases therefore drive `PutBlob` under a 20 s bound and fail on a hang, the
same pattern `projection_unix_test.go` established.

## Negative-evidence proofs

Nine mutants applied to production source, each run against the specific test
that must catch it. Every one was killed. Full transcript:
`TASK-260830-3qrfjp_mutation-evidence.log`.

| # | Mutant | Production site | Killed by | Failure observed |
| --- | --- | --- | --- | --- |
| M1 | drop the staged-length term entirely | `PutBlob` | short-write test, both cases | `error = <nil>`: a truncated file was **installed** into the immutable namespace |
| M2 | narrow the term from `!=` to `<` | `PutBlob` | short-write test, "kept more" case only | short case still passes, over-long case admits — the class-coverage proof |
| M3 | re-merge the length term into the mismatch branch | `PutBlob` | short-write test | refusal returns as `ErrIntegrityMismatch` with a quarantined fragment |
| M4 | drop the digest-path regularity clause | `inspectExisting` | FIFO digest-path test | **hang**: `PutBlob did not return within 20s` |
| M5 | drop the staged owner-only check | `PutBlob` | staged mode-drift test | group-readable candidate installed |
| M6 | replace the owner-only shard gate with plain `MkdirAll` | `PutBlob` | shard permission test + FIFO shard test + symlinked shard test | `error = <nil>` on all three: the write proceeds beneath a group-accessible directory, and lands **outside the AX data root** through the symlinked shard |
| M7 | wrap rebuild insert errors as `ErrProjectionCorrupt` | `rebuildProjection` | full-volume rebuild test | full volume reported as corruption |
| M8 | route a failed rebuild through the corruption quarantine path | `openProjection` | full-volume rebuild test | `index-recovery holds 1 directories` — a valid index quarantined on a full volume |
| M9 | drop the migration transaction rollback | `migrateProjection` | full-volume migration test | next open fails `SQLITE_BUSY`: the leaked transaction blocks recovery |

M1 and M2 together are the narrowing proof the length comparison needs: deleting
it proves only that something refuses, while narrowing it to short writes alone
leaves the over-long direction admitted.

## Validation

All 18 commands declared in
`task-board.config.json -> spawn.worktree_isolation.validation.commands` were run
directly as standalone processes. Every one exited 0. Transcript:
`TASK-260830-3qrfjp_validation.log`.

| Command | Exit |
| --- | ---: |
| `gofmt -l` over tracked and untracked Go files (empty) | 0 |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1 -v` | 0 |
| `tracecheck -section 3.2 3.3 10.1 10.2 18.4` (assigned scope) | 0 |
| `go test ./... -race -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| 5 × fuzz targets (`scalar`, `canonicaljson`), `-fuzztime=100x` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `cataloggen -check` | 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| every tracked `*.json` parsed | 0 |
| `task-board validate` | 0 |
| `git diff --check` | 0 |

Assigned-scope traceability reports
`traceability ok: contracts=60 normative_sections=36 acceptance_cases=43
fixtures=30 compatibility_contracts=55 assigned_scopes=5`.

`task-board validate` exits 0. Its 250 reported `MISSING_ACTIVITY` warnings are
pre-existing and touch no element of this Story or Task.

| Metric | Before | After | Delta |
| --- | ---: | ---: | ---: |
| `internal/localstore` statement coverage | 85.6% | 86.6% | +1.0pp |
| `internal/localstore` passing tests | 80 | 90 | +10 |

The baseline was measured by removing the three new test files and restoring
both production files to `HEAD` in the same worktree, then restoring the work.

## README

`README.md` was corrected and extended in the storage section:

- the mismatch sentence now says "size or digest mismatches **of the source
  bytes**", because an incomplete write is no longer routed there;
- a new paragraph states the write-side rule and that the length comparison is
  pinned in both directions;
- a new paragraph enumerates the full/failing-volume boundaries and the three
  recovery properties asserted at each;
- a new paragraph states that the special-file case bounds availability, and why
  a FIFO fixture and not a directory is what isolates the clause;
- the projection paragraph now states that a full volume is driven as a real
  `SQLITE_FULL` and is classified as a migration or rebuild failure, never as
  corruption.

No capability, `ax doctor` result or runtime availability claim was added.

## Not done, and why

- **Real filesystem exhaustion** (a volume actually filled to ENOSPC) is not
  driven. Creating one needs a RAM disk or a loopback image, which is a
  machine-mutating, non-deterministic side effect and is refused by the
  repository's deterministic-test rule. The seam and the SQLite page limit
  reproduce the same error values and the same code paths.
- **Windows** fault boundaries are skipped along with the rest of the package;
  the native owner-DACL implementation is a later platform task and the existing
  suite skips on `runtime.GOOS == "windows"` for the same reason.
