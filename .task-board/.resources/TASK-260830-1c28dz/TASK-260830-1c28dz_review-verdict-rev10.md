# TASK-260830-1c28dz — independent review of revision 10

Verdict: **ACCEPTED**. Candidate tree `2df3d47ac01fd90ef348af86730460f6080c88d3`, base/checkpoint `d4bd91d0`. Reviewer: claude-opus-5-5 (low). Authority: pinned SPEC v0.7.0 §4.2, §3.2, §4.C. All work ran on `git archive` copies under /tmp; the live worktree is byte-equal to the candidate on all 50 changed paths (0 diffs) and HEAD stays at the checkpoint.

## Revision 9 findings

| rev9 finding | Status | Evidence |
| --- | --- | --- |
| P1: peer census drops a receipt filed under another client's name | **Closed** | `AttachStore.Peers` (`internal/termbind/overlap.go`) now admits an entry only if the stem parses as a client identity and `AttachStore.Lookup(session, instance, stem)` reads it back. Lookup enforces the session/instance/client triple, so a mismatch errors before the requesting-client exclusion runs. That is the whitelist the brief asked for. Witness: `TestAttachPeerFilenameMismatchFailsClosed` goes through `Lifecycle.Execute(attach)`. |
| P2-A: post-escalation probe inherits the exhausted graceful deadline | **Closed** | `backend.escalationDeadline()` returns `opDeadline`. Production sets it from `mctx.Deadline` at `lifecycle.go:512-513`. The recheck before the kill (`effectRecheck`, `lifecycle.go:525/727`) is still there. Witness: `TestExecuteStopEscalationSucceedsWithContextHonoringRunner`. |
| P2-B: importer outcome evidence regressed to a suite grid | **Closed** | I compiled and ran the shipped `closuregrid` driver on my own base and candidate copies. Its output is **byte-identical** to the attached `closure-base.jsonl` (193 keys) and `closure-candidate.jsonl` (215 keys). It imports 33 of the 36 closure packages plus scalar. cloneproject, crashgate and sshtransport are stated bounds with reasons. The results label the suite grid as supplementary, and say so honestly. 0 keys moved. |

## Independent narrowings (each run twice, one raw log per run)

| Plant | Kind | Result | Killer |
| --- | --- | --- | --- |
| N1: Peers uses the decoded receipt, Lookup binding removed | narrowing | KILLED ×2 | TestAttachPeerFilenameMismatchFailsClosed |
| N2: escalationDeadline returns the graceful `deadline` | narrowing (reverts the fix) | KILLED ×2 | TestExecuteStopEscalationSucceedsWithContextHonoringRunner |
| N3: unparseable receipt stem skipped instead of refused | narrowing | KILLED ×2 | TestAttachStorePeersRejectsFilenameMismatch |
| N4: Lookup mismatch ignored only when the content names the requester (the exact rev9 hole) | narrowing | KILLED ×2 | TestAttachPeerFilenameMismatchFailsClosed |
| C0: harmless comment | applied control | SURVIVED ×2 | — |

Every KILLED run fails on an assertion (`--- FAIL`), not a build error. Logs: `plant-*.log`. The rev9 reviewer independently re-established the earlier rows: the 8/8 operation table, 34/34 AC witness sets, and the census counts. The rev10 delta touches only overlap.go, the backend deadline, the new tests and the evidence. I did not re-run the full shipped mutation harness.

## Validation (firsthand, exact candidate tree)

- The tmuxserver and termbind package suites pass with `tmux` unresolvable on PATH: 1059 pass, 0 skip, 0 fail, exit 0. `pgrep -x tmux` returns 1 before and after.
- `go vet ./...` exits 0. `GOOS=windows GOARCH=amd64 go vet ./...` exits 0. `gofmt -l ./internal` lists no files.
- Full `go test ./... -count=1`: 165s, 44 packages ok. `internal/specpin` fails because it surveys `.task-board`, which my archive copy excludes. It fails the same way on the **base** copy, so this is environmental. The attached rev10 handoff validation log reports `required=30 green=30 failed=0 missing=0`, and I accept that as the configured-suite evidence.
- `internal/traceability` is not touched. `task-board.config.json` equals the base. Seven candidate paths overlap with base..origin/main: LOGBOOK, README, and five tmuxserver files. These are predecessor Story content carried at the checkpoint plus intended leaf edits, the same as rev9's audit. Trunk c9233ce changes only the config, which the candidate does not touch.
- The producer evidence archive has 0 `.pyc` / `__pycache__` entries.

## Process note

While comparing live bytes, I ran `git add -N .` in the live worktree. That added intent-to-add index entries for the 41 untracked leaf files. I removed them right away with `git rm --cached` on exactly those 41 paths. The status is back to 41 `??` + 9 ` M`, HEAD is unchanged, and all 50 paths are still byte-equal to the candidate. No file content changed. I made no commits, and I did not edit the index in any other way.

## Remaining bounds (carried, not findings)

- B44: client liveness is owned by TASK-260922-vcx6yo.
- B45: the single-shot commit wait.
- Foreground attach integration is handed to g0pcnt.
- The registry is handed to g0pcnt.

## Addendum — acceptance refused by the board, routed to-dev

`accept_cr(TASK-260830-1c28dz, revision=10, ...)` refused with: `validation_not_bound_to_tree: ... the validation evidence carries no source tree identity, so its identity is unknown and revalidation is required (candidate_tree_oid=2df3d47a..., evidence_tree_oid=2df3d47a...)`. The reviewer role has no command that rebinds or revalidates the CR's validation record. This review found **no code findings**: all three rev9 findings are closed and every narrowing is killed. The element is therefore routed `to-dev` for a procedural re-handoff only.

## Rework scope (for the producer)

1. **Make no product changes.** Re-run `task-board handoff TASK-260830-1c28dz --role developer` on the unchanged candidate, tree `2df3d47ac01fd90ef348af86730460f6080c88d3`. The goal is a validation record bound to its source-tree identity. If the handoff still omits that identity, the gap is in task-board tooling, and the orchestrator must escalate it there.
2. The next reviewer may reuse this verdict's findings once they have confirmed the new revision's candidate tree OID equals `2df3d47a...`.
