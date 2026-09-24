# TASK-260830-g0pcnt review verdict — CR rev1: CHANGES REQUESTED

Reviewer: claude-opus-5-5 (low). Candidate tree 893980c4 (the live worktree tree matches it, checked with a scratch GIT_INDEX_FILE). Base 14d636e.

## Mandatory results (all rerun by me on the exact tree)
- `go test ./... -count=1`: 45/45 ok, exit 0.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0. Native vet exit 0. gofmt -l: clean.
- Digest: tracecheck recomputes the canonical projection and matches the pin `3664ab2f…`. Output matches README: contracts=64 … acceptance_cases=160, clauses_discharged=81/585.
- README plant: changing 81/585 to 82/585 made `TestREADMEMeasuredCoverageMatchesTracecheckReport` fail (KILLED).
- Decoded registry: tmux-private-server-management→4.2#4-7, lifecycle-operations→4.2#9 and 4.C#3,4,7, attach-overlap→4.C#6, attach-semantics→4.C#3,5,6,7, socket-custody→3.2#8. `story-260922-derivation-side-profile-source` is still in 2.4#2. The production owners are Execute, Acquire and CheckSocketCustody.
- Trunk preservation: all 63 paths in `git diff c9233ce 14d636e` that the Story does not touch are blob-equal to 14d636e. task-board.config.json is byte-identical to the base. Checkpoints 5299cba, e1e2901 and 8d944c7 all pass verify-commit.
- Determinism: 8 new or named tests at -count=3 → 24/24 PASS.
- Importer outcome comparison: NOT rerun by me. I accepted it from producer evidence (224 keys, one moved class, which is the 0644 probe refusal).

## Mutation (my plants, each run alone against the full tmuxserver+termbind packages)
| Plant | Result | Killer |
|---|---|---|
| C-control (comment only, applied) | SURVIVED (expected) | — |
| R1 leaf socket `perm&0o077` → `perm&0o070` (admits other-only bits, e.g. 0606/0602) | **SURVIVED ×2** | none |
| R2 ancestor walk skips the immediate parent | KILLED | TestCheckSocketCustodyRefusesWritableAncestor and 4 others |
| R3 escalation made one-directional (`stored.InputAuthorized && !inputAuthorized`) | KILLED | TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable (alone) |
| R4 Execute custody skipped only for `status` | KILLED | TestExecuteEveryOperationRefusesSocketSubstitutionBeforeDispatch and others |
| R5 root refuses only when world-writable (group-writable admitted) | KILLED | TestExecuteCustodyNarrownessTwins |
| Replant of the 35urbp P3-A fresh-decoy foreground narrowing | KILLED | TestAcquireForegroundRefusesEveryCatalogDecoyRunning/headless_creation |

Measured: 5 of 6 non-control narrowings killed.

## Findings
**P2 — the socket-permission gate is pinned along one axis only** (repeat-of: rule pinned along one axis). In `internal/tmuxserver/bind.go` `checkCustodySocket`, the socket leaf modes the tests witness are 0600 (admit) and 0644 (refuse). 0644 carries a group bit, so a narrowing that ignores the "other" bits (R1) leaves every test green. Under that narrowing, a current-user socket at mode 0606 or 0602 (world read/write, i.e. world-connectable) passes custody at all three production entries: Probe, Spawn and Execute. SPEC §3.2 requires an owner-only socket. The shipped `N-socket-permissions` mutant admits 0644, so it cannot reach this member.

No P1 findings. Inherited items (a), (b) and (c) hold. I did not independently audit (d), B44 liveness.

## Rework scope (for the producer)
1. Add socket-leaf mode rows for the other-only class (at least 0602 and 0604, plus 0606) and a group-only class (0620). Drive each through `ServerProber.Probe`, `ServerSpawner.Spawn` and `Lifecycle.Execute`, and assert the literal `tmux_unsafe_socket_path` / "socket permissions" with zero Dial, spawn and dispatch.
2. Ship my R1 narrowing (`perm&0o077` → `perm&0o070`) and its twin (`→ 0o007`) as harness mutants. Both must be KILLED by a test run alone.
3. Update TRACEABILITY and the conformance matrix cell for socket permissions with the mode axis enumerated.
