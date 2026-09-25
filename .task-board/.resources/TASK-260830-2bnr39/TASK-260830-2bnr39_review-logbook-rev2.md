# Review logbook — TASK-260830-2bnr39 CR revision 2

RUN-260907-428f33 independently reviewed tree 935f624e45088a064de04945cbe4bd86d1b37d1b.
F1–F7 are resolved for the internal repository/index capture leaf. All 8 AC rows
have executed production drivers. Full repository tests and coverage pass;
35 mutation/control vectors compile and execute: 33 behavioral kills, 2 neutral
passes; 17/17 registered gates have narrowing witnesses. Native macOS Git tests
preserve original index bytes, identity and mtime under normal and alternate
split indexes, stale stat cache and actual content edits. Timeout cleanup passes.

Non-obvious bound: real Git refreshes sharedindex.* auxiliary timestamps on read.
The reviewer set an old timestamp and measured the refresh; shared-index bytes
remain unchanged. The accepted claim is original-index byte preservation with
disposable diff indexes, not zero filesystem metadata writes. Do not reuse the
broad idempotency_test.go preamble as evidence of the stronger claim. Hard-kill
OS-temp remnants, trusted external filters, native Windows/Linux behavior,
full wire assembly and content/blob closure remain explicitly outside this leaf.

The reviewer leaves LOGBOOK.md untouched to preserve the published candidate.
This task-scoped note is the persisted logbook handoff, not an implementation edit.
Acceptance routes integrating; the orchestrator must route the bound producer
for managed checkpoint/integration.
