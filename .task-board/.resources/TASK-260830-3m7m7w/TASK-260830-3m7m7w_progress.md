# TASK-260830-3m7m7w development evidence

Run RUN-260907-d14c2f; active GOAL-260907-c70ce0 revision 1, role_handoff,
resolved scope TASK-260830-2bnr39 and TASK-260830-3m7m7w.
Predecessor independently accepted CR2 is checkpointed at
 d888cd576f5962eb258e40ca487f7711b2215610; current board reads 18/18 checklist
and integrating. This producer preserves that checkpoint and private-index code.

Current implementation extends Capture via ContentOptions for working bytes,
policy-selected ignored files, symlink targets, recursive submodule state,
large/chunked blobs, sparse/linked worktrees and exclusions. It reuses secprim,
canonicaljson and localstore. Candidate remains uncommitted in the managed Story.

Personally executed repository-wide go test ./... -v and go test ./... -cover
on the latest Go candidate: both exit 0. Coverage is gitsnap 83.4%,
canonicaljson 97.1%. Build, vet, formatting, tracecheck and generated catalog
validation exit 0. Windows build/vet exit 0 (compile only). Relevant race tests
passed before the last lockfile predicate correction; latest gitsnap race rerun
is live and not yet claimed. Prior mutation run has 11 named kills and one
measured neutral survivor; latest replay is live. Failed early attempts remain
in the log packet and are not presented as green.

The scope map records 7 of 7 content AC rows driven at the internal Capture
entry. Exact pinned pack/index corpus creates separate real parent/child object
databases for capture fixtures; no pack production or materialization is claimed.
Object-pack production, raw-index blob delivery, workspace-root/child manifest
assembly, quiescence and full closure/race strengthening remain separate work;
this internal content result does not advertise a CLI or doctor capability.

This is progress evidence, not review handoff. Full command logs, source digests,
measured mutant table and any survivor bounds will be attached before handoff.
