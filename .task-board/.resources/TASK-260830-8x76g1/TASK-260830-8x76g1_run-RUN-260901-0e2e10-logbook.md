# TASK-260830-8x76g1 logbook — RUN-260901-0e2e10

## Recovery decision

Both supplied archives passed their declared SHA-256 checks. They were safely
extracted under `.temp/TASK-260830-8x76g1/` for comparison only. The active
worktree was already the exact reviewed rev15 candidate tree
`1617e6afc56110589f6f0f524ceb94de192b43b5` and was materially newer than the
carry-forward archive, so restoring either archive into the project root would
have regressed reviewed rework. No root restoration was performed.

## Review finding closure

The rev15 universal claim about every `requireString` caller was false for one
caller: symlink `target`. The corrected evidence enumerates the actual fact:
the shared empty-string guard is the sole lower-bound enforcer for that member,
and a dedicated public-entry test now kills its disablement. Production was
already correct and stayed untouched.

## Harness and environment notes

The attached harnesses do not have shebangs. The `python` alias was absent
(readiness exit 127), while `/opt/homebrew/bin/python3` 3.14.7 passed readiness
and ran the unmodified scripts. The filesystem had 56 GiB free before the
sweeps, so the rev15 reviewer’s disk-exhaustion anomaly did not recur.

Each long sweep was split into bounded foreground halves and fully awaited.
No process was backgrounded across lifecycle boundaries. Production source was
restored and SHA-256 verified after every mutant and after each half.

## Board validation anomaly

`task-board validate` exited 0 while reporting the same 262 inherited
`MISSING_ACTIVITY` diagnostics seen by prior runs. The current task was not in
that successfully-read set; the anomaly is inherited and outside this test-only
rework scope.
