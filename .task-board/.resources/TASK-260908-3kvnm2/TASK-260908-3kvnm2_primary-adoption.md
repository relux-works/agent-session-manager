# Adopt the accepted immutable v0.6.0 normative source

Implement TASK-260908-3kvnm2 in its managed Story worktree. Source prerequisite is delivered and independently accepted: agent-session-manager-spec PR2 auth and PR3 selector landed, signed annotated git tag v0.6.0 published. Release reviewer RUN-260909-97e5d0 accepted; remaining release-side RUN095979 only acknowledges board completion, no source changes. Read the attached release verdict copy. Verify fresh remote tag object/signature and pinned commit before adoption:
- repository relux-works/agent-session-manager-spec
- tag object 40c123eb8399efa8e05cbc009110940ed861a785
- peeled commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6
- SPEC.md SHA256 74504539fb43c28ae3450622bc1002e643f116cd3e14df4567a882231e90896b (1150005 bytes).
Use immutable commit/blob URLs, not mutable main or an unproven hosted GitHub Release. Only signed git tag exists. Existing local source checkout /Users/iv/Developer/ReluxWorks/agent-session-manager-spec may be used read-only after exact object verification.

Inspect existing specpin/specdoc embedded source, manifest, tests, README and source-version dependencies. Adopt the FULL approved selector/auth/migration normative document and immutable identity with meaningful production-entry refusal tests for partial, stale, substituted or mismatched pins/document bytes. Preserve historical v0.5.0 and earlier provenance and compatibility semantics; do not relabel old constants/artifacts as new or silently alter historical contracts. Source authority does not advertise implementation support: new runtime contracts remain unimplemented until their real owner tasks satisfy them.

Sibling TASK-260908-2tkufa owns regenerated catalogue/traceability and new obligation ownership after this leaf is accepted/checkpointed. Keep coherent source-pin prerequisites here, clearly record cross-leaf dependencies; do not weaken existing gates to pretend missing implementation exists. Do not add a second sibling CR or start sibling writers. Preserve every other partial Story worktree. Fresh AX main/origin verified equal2a8db9653e476f8375371b16e9b6b82adfa23b91 before this spawn; recheck authority through task-board.

Read Curator Go testing skill, inspect target platform/toolchain, use narrow tests during iteration plus required repository-wide go test ./... -v and go test ./... -cover before review. Store full logs with explicit exits under task .temp; do not rely on capped CR logs, compile failures, unexecuted controls or empty selections as semantic evidence. LocalCIonly, no hosted triggers.

All source/docs edits stay in managed Story worktree. Update appropriate README/LOGBOOK there before final candidate/commit. Signed Ivan Oparin commits only; obey current task-board candidate/checkpoint rules, never broad reset/clean or foreign file staging. Deliver reviewed-scope candidate CR and task-scoped outcome, then producer handoff to reviewer; no unreviewed main push. Report unresolved new runtime support truthfully. Keep global config/skills and upstream176/177 out of scope.
