REPUBLISH CONTINUATION for TASK-260830-nxqqaw. The previous producer run finished the work and ran the handoff, but CHANGE REQUEST CONSTRUCTION failed at 16:45Z on 2026-09-23. At that moment the host disk had 1-2 GB free: reviewer copies under /tmp and a go-build cache had filled it. The refusal was a git snapshot/tree-identity failure. It was not a defect in your candidate. The orchestrator has freed the disk (33 GB free now). DO NOT redo the implementation.

DO EXACTLY THIS:
1. Verify the uncommitted candidate in `.temp/STORY-260830-ub60id/worktree` is intact. Compare it through a scratch GIT_INDEX_FILE, and check that its changed paths match your last results/conformance matrix. Current trunk is `0ca3e4c`. If the Story branch tip does not descend from it, run `task-board worktree refresh-candidate TASK-260830-nxqqaw`, because CR construction requires the checkpoint to descend from current trunk. If the tip already is `0ca3e4c` or descends from it, do not refresh.
2. Rerun only the fast checks that prove the tree is whole: `go build ./...`, `go vet ./...`, `GOOS=windows GOARCH=amd64 go vet ./...`, and the touched package tests. Construction reruns the full configured suite itself.
3. Attach a fresh short outcome, `TASK-260830-nxqqaw_republish-evidence.md`: the tree OID, the base, the checks and their exits. The handoff needs fresh task-scoped evidence. Keep the checklist current.
4. Run `task-board handoff TASK-260830-nxqqaw --role developer` and exit.

SCRATCH RULE: any scratch goes under `.temp/STORY-260830-ub60id/worktree/.temp/TASK-260830-nxqqaw/` or `/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/review-scratch/$TASK_BOARD_RUN_ID/`, never under `/tmp`. LIVE-INDEX RULE applies. Canonical CLI: /Users/iv/.curator/global/bin/task-board.
