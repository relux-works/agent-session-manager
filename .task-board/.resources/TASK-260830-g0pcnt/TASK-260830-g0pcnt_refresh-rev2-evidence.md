# TASK-260830-g0pcnt refresh rev2 evidence

## Managed refresh

The canonical task-board worktree refresh command was run against the authoritative board. The initial shell wrapper yielded after 30 seconds without exposing its session identifier; the process later exited and the authoritative worktree status record confirms the completed transition. No numeric exit code is claimed for that invocation.

- Selected/current base: 360c8bd79cc2325ab661a9ba591b6bfb82ebe0ea / 360c8bd79cc2325ab661a9ba591b6bfb82ebe0ea
- Story checkpoint and branch tip: 5443b3b98e8b7b66a139369d48c79879e41d4b92 / 5443b3b98e8b7b66a139369d48c79879e41d4b92
- Replayed checkpoints: 05d89f5, 602510f, and 5443b3b; git verify-commit passed for all three (exit 0).
- git merge-base --is-ancestor 360c8bd HEAD: exit 0.
- Run lease remains held by RUN-260923-c58037.
- Candidate tree from a scratch GIT_INDEX_FILE: 83a4313600b18400a9195731e6a265ff8424e45c. Candidate remains uncommitted.

The first status mutation from the worktree directory refused before writing because this installed CLI does not accept task-board.config.json's commit_time_policy shape (exit 1). Retrying the mutation from /tmp with --board-dir /Users/iv/Developer/ReluxWorks/agent-session-manager/.task-board succeeded (exit 0), setting the task to development.

## Tree integrity

A scratch-index comparison found 64 paths in c9233ce..360c8bd. All 57 trunk-only paths are blob/mode-equal to 360c8bd; the seven shared paths remain in the Story candidate. task-board.config.json is byte-equal to the current trunk blob.

The prior backup tree d8c9cde9 contains 24 paths relative to checkpoint 8d944c7. After removing the required new refresh entry from the LOGBOOK comparison, 22/24 paths match. Two backup-only differences are intentionally not carried forward:

- internal/tmuxserver/bodies.go: the backup changes the exact four-member terminate body gate to accept one unnamed extra member. In an isolated copy, go test ./internal/tmuxserver -run '^TestParseTerminateBodyViolations$' -count=1 exited 1 as expected; TestParseTerminateBodyViolations failed with “unknown member admitted”. The pinned closed request shape and this committed refusal test remain authoritative.
- internal/tmuxserver/__pycache__/mutant_harness.cpython-314.pyc: generated Python bytecode, omitted from the source candidate.

This is not reported as 24/24 backup-byte equality. The tree audit log records 22/24, the two paths, and 57/57 trunk-only equality.

## Refresh verification

| Command | Exit |
| --- | ---: |
| go run ./internal/traceability/cmd/tracecheck | 0 |
| go test ./internal/traceability/cmd/tracecheck -run '^TestREADMEMeasuredCoverageMatchesTracecheckReport$' -count=1 | 0 |
| go build ./... | 0 |
| go vet ./... | 0 |
| GOOS=windows GOARCH=amd64 go vet ./... | 0 |
| go test ./internal/tmuxserver ./internal/termbind -count=1 | 0 |
| configured gofmt gate | 0 |
| git diff --check | 0 |
| backup-only body widening probe above (expected red) | 1 |

Fresh tracecheck output:

    traceability ok: contracts=64 normative_sections=36 acceptance_cases=160 fixtures=33 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=70 full=4 partial=10 sliver=11 unevidenced=41 unmeasured=4 unowned=7 clauses_discharged=81/585

The README measured-coverage pin test passed. The registry itself did not change in this base move; the existing rev2 digest remains 3664ab2fb166189545fea34a068a318af4f251689f29c92915fa185fefdedeea.

The previously attached rev2 results and conformance matrix retain the measured 4 of 4 task acceptance rows at named production entries, the gate/axis census, and B44/B46 bounds. The full suite and importer grid were not rerun during this refresh because the refresh continuation limited reruns to checks affected by the base move; their earlier task-scoped evidence remains attached.

Fresh logs and the scratch-tree report are attached in TASK-260830-g0pcnt_refresh-rev2-logs.tar.gz.
