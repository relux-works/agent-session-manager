# TASK-260830-8x76g1 — recovery rerun logbook

## Finding

CR revision 5 failed before product validation because its first command was
the older base `gofmt -l .` gate. Four ignored reviewer probes under `.temp/`
were unformatted; tracked and non-ignored production Go sources were already
formatted. The current candidate config already narrows the gate to tracked
plus non-ignored Go files, so ignored task evidence cannot create this false
red in later candidate suites.

## Recovery action

The four ignored probe sources were formatted mechanically. The exact old gate
and the candidate gate then returned exit 0. No repository source, test,
fixture, documentation, or configuration file was changed by this recovery
successor.

## Evidence boundary

The accepted-leaves and carry-forward archives were verified against their
provided SHA-256 digests and extracted only into a task-scoped comparison
overlay. All unchanged archive files match. Nine files differ solely because
the active candidate contains the later revision-4 through revision-6 work.
Overwriting those files from the older carry-forward would have destroyed the
review-driven total-registry fix, so no such destructive restoration was done.

The global Curator command was used only after the project-local
`.agents/bin/curator` readiness check returned exit 1 because that adapter is
absent. Global `curator --version` and `curator status --check` both ran
successfully. Board validation still reports 262 inherited
`MISSING_ACTIVITY` diagnostics outside this task.
