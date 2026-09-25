# TASK-260830-3m7m7w publication lifecycle evidence

Directive RUN-260908-5dde57:nudge:319558 reports reviewer launch refused
change_request_candidate_drift: current CR1 tree f219517ac93cf446bd6ae70c7ebd3a86eb2c85fa
versus recomputed current tree 4f1407cb98f19424db527d313a3d89d34e2fd231.
No reviewer child launched. The source manifest and original index remain intact.

Installed task-board version reports dev and resolves via the global Curator
wrapper. Root, spawn and worktree help expose no publish/republish command.
`task-board --board-dir /Users/iv/Developer/ReluxWorks/agent-session-manager/.task-board q 'schema(mutation=publish_cr)'`
exits 1: unknown mutation. Schema recovery finds no publication mutation.
`handoff ... --role developer --format json` exits 0, confirms to-review 19/19
and current outcomes, but neither changes CR1 nor emits publication validation.

Global project-management references/tracked-background-spawn.md lines 381-389:
“Construction runs for every producer completion in a managed workspace” and
“It runs after the spawn completion artifact guard”. The same reference states
that managed publication runs configured validation and binds it to a candidate.
The global CLI source cmd/spawn_runtime.go:813-816 explicitly documents that
Change Request validation runs after the child exits and before the run becomes
terminal. Its hidden spawn-runner command executes a complete manifest; it is
not a republish endpoint. Starting a second runner is not a safe substitute.

Supported operational route: let this already-handed-off producer return; observe
RUN-260908-5dde57 until its managed post-child publication/validation is terminal;
then read its new CR revision/tree and launch independent review against that
revision. All 26 configured commands have already run green in this session and
task-scoped sources/logs/AC/mutant evidence is attached. Runtime revalidation
remains independently owned by the existing publication lifecycle.

The alternative requires an existing supported in-session publication entry or
a separate task-board product change, neither of which is exposed by this
installed CLI or authorized as part of CR1 content rework. No registry edits,
manual commits, duplicate runners, rollback, or fabricated CR identity.

This is an operational ordering constraint, not an AX content implementation
constraint. No forced-fit product workaround has been attempted. Coordinator
was notified through the authorized task notes; stale CR1 is not claimed as a
published revision of the rework.
