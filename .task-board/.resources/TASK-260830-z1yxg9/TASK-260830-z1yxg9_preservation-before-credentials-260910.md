# RPC delta preservation (verified 2026-09-10, pre-credentials parking)

Blocked leaf: TASK-260830-z1yxg9 (implement-rpc-envelope-and-hello), status=blocked.
Parked by credentials producer TASK-260909-2ez769 to free candidate scope; RPC task remains pending.

## Control root
- /Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/TASK-260830-z1yxg9/preserved-before-credentials-260910/
- manifest.json: 15 files, SHA256, head c1eff016dce2e55c4e2c1828d5c20f3da84118bc, tracked/untracked classification
- tracked.patch: byte-identical to worktree git diff at park time (verified with diff -q)
- files/: full byte copies of all 15 files (verified BACKUP-ALL-MATCH against manifest SHA256)
- Worktree bytes verified ALL-MATCH against manifest before any change.
- All file modes 100644 (regular files, no executables, no symlinks).

## Enumerated delta
Tracked (restore = git checkout c1eff01 -- <path>; checkpoint blobs e662afb/README, 039a0301/ownership.json, 4ee55d7/traceability.go, 1a2b3147/task-board.config.json):
- README.md, internal/traceability/ownership.v0.5.0.json, internal/traceability/traceability.go, task-board.config.json
Untracked (restore = copy back from files/ preserving mode 0644):
- UNRESOLVED_QUESTIONS.md, internal/rpcwire/{README.md,envelope.go,envelope_test.go,hello.go,inventory.go,mutations.py,testdata/contracts-v3.json,testdata/contracts-v4.json,testdata/hello-request-v2.json,testdata/hello-response-v2.json}

## Restoration instructions
1. cd to STORY-260830-1kiyj6 worktree at checkpoint c1eff016dce2e55c4e2c1828d5c20f3da84118bc.
2. Tracked: git apply --check <control-root>/tracked.patch && git apply <control-root>/tracked.patch
   (equivalently: apply worktree diff; patch was proven identical to git diff).
3. Untracked: cp -R <control-root>/files/internal/rpcwire ./internal/rpcwire && cp <control-root>/files/UNRESOLVED_QUESTIONS.md ./ && chmod 0644 on all restored paths.
4. Re-verify: sha256 of each restored path must equal manifest.json entry; git status must show the 4 tracked modifications plus untracked UNRESOLVED_QUESTIONS.md and internal/rpcwire/.
5. No secrets are contained; nothing was printed. Backup retained after parking.
