# TASK-260830-1qf777 — Round 4 (recovery run RUN-260901-877162)

## Why this run exists

Round 3 (RUN-260901-12b8e2) closed reviewer findings F1/F2/F3 and exited 0, but
Change Request construction then failed and the board queued this successor:

```
change_request_base_authority_mismatch: the STORY-260830-jeaivu candidate
provenance disagrees (selected=48db30b current=48db30b checkpoint=c0d42491
branch=d583fe8 head=d583fe8)
```

This was a **delivery-provenance defect, not a behavioural one**. No production
code was changed in this round.

## Root cause

`TASK-260830-1qf777` is now the final unresolved leaf of `STORY-260830-jeaivu`
(`TASK-260830-2iint0` and `TASK-260830-17suox` are both `done`), so its Change
Request resolves as `kind=story_final` — confirmed on the live record
`CR-TASK-260830-1qf777-2` (`"kind": "story_final"`). Revisions 1 and 2 were
constructed while sibling leaves were still open, i.e. as ordinary
`task_delta`, which never reaches the story-final authority gate.

`convergeStoryFinalBaseAuthority`
(`project-management/tools/board-cli/internal/spawnruntime/changerequest.go:653`)
admits exactly two candidate shapes:

1. **uncommitted candidate** — `branchOID == checkpoint && headOID == checkpoint`,
   with the leaf's work sitting in the managed worktree;
2. **exact one-commit candidate** (BUG-260831-3lb2jt) — additionally requires
   `current == selected && checkpoint == selected`.

Observed state failed both. Shape 1 failed because branch/HEAD were at
`d583fe8`, not at the checkpoint `c0d42491`. Shape 2 failed because
`checkpoint (c0d42491) != selected (48db30b)` — that shape only covers a story
whose branch is one commit past trunk with nothing yet checkpointed.

The cause is that round 3 **hand-committed** its work as `d583fe8` onto the
Story branch. Committing is the board's step, not the producer's: the two
accepted sibling leaves reached the branch through `task-board worktree
checkpoint` (`020d0b6`, `c0d42491`, both carrying the board's doubled-ID commit
subject), whereas `d583fe8` carries a hand-written single-ID subject. For a
`story_final` leaf the candidate must remain uncommitted so that `Take()` can
build the whole-story delta from trunk (`48db30b`) to the worktree.

## Repair applied

`git reset --mixed c0d42491d08bc6dca6c2322b84085cfce50b1217` in the managed
worktree — branch tip and HEAD returned to the checkpoint, every byte of the
round-3 result preserved in the worktree.

Proof nothing was lost — the worktree tree hash is identical to the tree of the
un-committed commit:

| Item | Value |
| --- | --- |
| `d583fe8^{tree}` (before) | `9700e40f4bc9787d495864cd97654b2a8864121b` |
| worktree tree after reset | `9700e40f4bc9787d495864cd97654b2a8864121b` |
| rescued commit (reflog) | `d583fe82ccd48ad23382c6b202545c7041fda29b` (signed, `G oparin@me.com`) |
| branch/HEAD after | `c0d42491d08bc6dca6c2322b84085cfce50b1217` (= checkpoint) |

`d583fe8` touched only source paths (no `.task-board`), so nothing board-owned
moved. It remains reachable in the reflog if the decision is reversed.

Resulting candidate: 9 modified tracked files + 5 new untracked files
(`internal/config/migration.go`, `migration_os_test.go`,
`migration_refusal_test.go`, `migration_test.go`, `refusal_structure_test.go`).
The untracked files are not gitignored and ARE part of the candidate — the tree
hash above is computed with `add -A`, exactly as `changerequest.Take()`
(`snapshot.go:103`) builds it.

Gate re-check against `convergeStoryFinalBaseAuthority` after the repair:
`branch == head == checkpoint` (shape 1 satisfied), `upstream == selected ==
48db30b`, selected tree == upstream tree, and `48db30b` is an ancestor of
`c0d42491`.

## Independent verification of round 3 (not taken on trust)

Round 3's claims were re-derived in this run rather than accepted from the
board note.

| Claim (round 3) | Verified this round | Result |
| --- | --- | --- |
| 29/29 mutants dead, zero survivors | `mutants.py` re-run from scratch | 29 RED, 0 GREEN, restored tree green, exit=0 |
| F2: `MigrateOS` 0.0% -> 100.0% | `go tool cover -func` | `migration.go:61 MigrateOS 100.0%` |
| F3: `(*MigrationError).Error` 0.0% -> 100.0% | `go tool cover -func` | `migration.go:53 Error 100.0%` |
| `internal/config` coverage 92.3% | `go test ./... -cover` | 92.3% |

F1 (symlink stance) inspected in source: the read seam
(`loader.go:199`, `os.Stat`) follows symlinks and applies the kind check to the
resolved target; the mutating seam (`migration.go:267`,
`osMigrationFileSystem.Stat`, `os.Lstat`) does not. Both stances are declared in
comments at both sites and in README.md:218-227, and pinned in both directions
by mutants M23 (widen mutating seam) and M24 (narrow read seam) — both RED.

## Gates, real exit codes, at the handoff state (uncommitted candidate)

All run as standalone processes; no `tee`, no pipes.

| Gate | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | 0 (empty) |
| `go test ./... -cover -count=1` | 0 |
| `go test ./internal/config -race` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `tracecheck -section 6.1 6.2 6.3 6.4 6.5 17.1 17.2 17.4` | 0 (assigned_scopes=8) |
| `cataloggen -check` | 0 |
| `go generate ./internal/catalog` + `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 |
| `go mod verify` | 0 |
| `git diff --check` | 0 |
| `mutants.py` (29 mutants) | 0 |
| `go test ./internal/config/ -count=1` under `env -i`, empty HOME, `GOPROXY=off` | 0 |

Coverage at handoff: canonicaljson 87.1%, catalog 97.6%, cataloggen(cmd) 79.3%,
cataloggen 83.9%, **config 92.3%**, scalar 90.1%, specpin 85.1%, traceability
85.0%, tracecheck 87.5%.

## Scope statement

No production or test source was modified in round 4. The only change is the
Git-provenance repair described above. Everything the reviewer assessed at
`d583fe8` is byte-identical in the candidate now presented.

## Post-handoff verification

Handoff recorded: `status:to-review checklist:15/15`.

Change Request construction runs at spawn-run completion, not inside `handoff`,
so the rev3 record cannot be observed from this session. What CAN be established
now, and was:

**Every condition of `convergeStoryFinalBaseAuthority` holds.**

| Condition (changerequest.go:653-731) | State | Holds |
| --- | --- | --- |
| workspace record present | `WS-e99224f092ce` | yes |
| `kind == story_final` | `story_final` | yes |
| `selected`/`current`/`checkpoint`/`branchRef` non-empty | all set | yes |
| `upstream == selected` | both `48db30b` | yes |
| shape 1: `branch == checkpoint && head == checkpoint` | both `c0d42491` | yes |
| `selectedTree == upstreamTree` | same commit | yes |
| `IsAncestorOfTrunk(selected, checkpoint)` | `48db30b` is an ancestor of `c0d42491` | yes |

**The candidate tree was independently recomputed the way `Take()` builds it** —
alternate index seeded from `48db30b^{tree}`, `add -A` over the worktree,
`.task-board` prefix forced back to base content:

| Tree | Value |
| --- | --- |
| predicted candidate | `9700e40f4bc9787d495864cd97654b2a8864121b` |
| round-3 result (`d583fe8^{tree}`) | `9700e40f4bc9787d495864cd97654b2a8864121b` |
| recorded rev2 candidate | `a9b9e5fad99897cd11906d89cc44c21251f5a4cd` |

The candidate is byte-identical to the round-3 result, and differs from rev2, so
a new revision will be constructed rather than the stale record being reused.

Not asserted: that rev3 exists. That fact belongs to the run-completion step and
is reported as unknown here rather than inferred from the preconditions.
