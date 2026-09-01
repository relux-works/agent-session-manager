# TASK-260830-8x76g1 — logbook (RUN-260901-2ac20b)

## 2026-09-01 — archive and workspace decision

Both supplied archives matched their declared SHA-256 digests and contained
relative repository paths. They were extracted only under task-scoped scratch.
The prior attached recovery outcome proves the current workspace already carries
the later CR revision 10 task-owned changes, so overwriting it with the older
archive snapshots would have discarded accepted rework rather than restored it.

## 2026-09-01 — authorized GitIndex exception

The spawned-run rationale explicitly authorizes option 1 from the prior
stop-the-line packet: preserve the 5 MiB public identity cap; prove 22 reachable
bounds through both public entries; prove `GitIndex.entries` at the production
validator and prove public outer-cap refusal. The focused rerun measured the
smallest valid 65,536-entry parent at 12,323,001 bytes.

The older checklist item requiring all 23 accept-at-N cases through both public
entries contradicted that authorized contract. A dry run resolved item 31 to the
exact obsolete text, then `remove_checklist_item(..., item=31)` removed it with
explicit confirmation. The newer GitIndex-exception criterion remains.

## 2026-09-01 — validation and inherited board diagnostics

All direct format, build, vet, focused, full, race, coverage, four fixed-count
fuzz, traceability, catalog, cross-platform, JSON, diff, and board commands
exited 0. `task-board validate` still prints 262 inherited
`MISSING_ACTIVITY` diagnostics while returning 0; `TASK-260830-8x76g1` is not
listed. Mutation expected-red evidence was accepted from the prior attached run
and was not regenerated against production source in this recovery.
