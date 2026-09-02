## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- TASK-260830-17suox

## Blocks
- TASK-260830-2890sd
- TASK-260830-2xdt8t
- TASK-260830-2ciy0s
- TASK-260830-3kle7h
- TASK-260830-2u34k1

## Checklist
- [x] Production entry points implement the scoped deliverable: Prove backup, migration, unknown-field refusal, bounds, duplicate backend rejection, and read-only downgrade behavior
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
FRONT-LOADED DoD, distilled from ten rounds on leaf 1 and five on leaf 2 of this Story, plus the parallel record-schema and object-store leaves. These are the criteria the reviewer will apply.

Scope note carried forward from leaf 2: durable atomic migration and downgrade mutation was deliberately deferred to THIS task. Leaf 2 covers in-memory v1/v2 to v3 translation and the current-only writer; the durability, atomicity and rollback story is yours.

1. NEVER INVENT A CONSTRAINT. Any quote you place in a Pinned SPEC declaration column must literally appear in SPEC.md at 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c. Two rounds were burned on this across sibling leaves. Where the spec is silent, be permissive and say so. Do not claim a spec section the suite does not exercise.

2. DERIVE COMPLETENESS. Version sets, member sets and refusal-site inventories must be derived mechanically - leaf 2 landed an AST-derived refusal-site gate; reuse it rather than hand-listing clauses. Verify the gate itself by injecting a synthetic never-exercised refusal site and confirming it reddens.

3. EVERY CLAUSE MUST DIE. Run the clause sweep yourself and report zero unexplained survivors. Leaf 2 finished at 109/109. Where a clause is subsumed, name the subsuming check in the artifact and pin that naming.

4. WIDENING, NOT ONLY DELETION. Vocabularies and grammars must die to a mutant that WIDENS them. Bounds proven in both directions at the production entry, accept-at-limit and refuse-past-limit, Unicode characters not bytes, asserted against the SPEC literal and never against the implementation constant.

5. A NEGATIVE CASE MUST ISOLATE ITS CLAUSE. Violate exactly one thing, so the case cannot pass on an earlier disjunct. This exact defect survived four rounds on leaf 2.

6. PROVE BOTH BRANCHES. Any check that behaves differently on a create path versus an existing path must be proven on both. For migration this extends to crash and interruption: prove the on-disk state after an interrupted migration is either the old document or the new one, never a torn one, and drive it through the real production entry.

7. DOWNGRADE MUST FAIL CLOSED. A newer document read by an older reader must refuse with a typed error rather than silently dropping members. Prove no silent rewrite path exists.

8. DETERMINISM. No network, no terminal dependency, no ambient environment. The suite must pass under env -i with an empty HOME.

Operational note: use a task-scoped GOCACHE. Shared-cache failures disrupted sibling runs on this machine.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:97a03f5c6378c00c3c16903d1bd83dfac31eeecb8d234a0b749b016e8ac47eb6 rationale="Following rank-1 recommendation: final leaf of the configuration Story, stacked on accepted checkpoint c0d42491, with the fifteen-round DoD front-loaded."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260901-7ab441, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260901-7ab441)
Implemented explicit config.Migrate/MigrateOS durability and read-only AssessCompatibility. Important findings: real OS loading used os.Stat and could follow a symlink despite the regular-file contract, so OSInputs now uses os.Lstat with a real production-entry regression; retry must fsync an already-existing exact backup because it may follow an interrupted run; traceability gained reviewed Section 6.4/6.5/17.4 owners and two acceptance cases without advertising CLI/doctor/backend capability. Full evidence and resolved red gates are attached as TASK-260830-1qf777_results.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260901-7ab441, pid=42946, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:819114cc7f1d9457c51f6d0fa29fa21a958980fe41620c157909136984c2a721 rationale="Rank-1 under the claude-exclusive policy; final leaf of the configuration Story, and cross-provider independence holds because the producer ran on Codex."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-d4aa34, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-d4aa34)
Review verdict CR-TASK-260830-1qf777-1 rev1: CHANGES REQUESTED (RUN-260901-d4aa34). Build/vet/full test suite green; every migration and downgrade refusal behaves correctly when driven through the real Migrate/AssessCompatibility entry points (probe transcript attached). Rejected on evidence, not behavior: six delete/narrow mutants of new production refusal gates in internal/config/migration.go leave go test ./internal/config fully green. (1) existing-backup integrity check at migration.go:281-287 (content + owner-only perms) - the headline prove-backup gate, deletable silently; needs stale-bytes and 0644-perms negative tests asserting ErrMigrationBackup, unchanged source, unchanged pre-existing backup, no .ax-config-* leak. (2) ErrMigrationTarget at migration.go:182-184 never produced by any test; the TargetVersion==Version1 clause is free-floating. (3) ErrMigrationSourceAbsent and the non-regular-source TOCTOU recheck at migration.go:302-304 unexercised. (4) disclosure-choice gate tested only for empty; invalid non-empty falls through to an encode error instead of ErrMigrationChoiceRequired. (5) ErrMigrationRecovery - the rollback-failed crash branch - has no test; the existing faultMigrationFileSystem is one injection point away from it plus write/publish backup and write/replace source. (6) refusal_inventory_test.go walks configError call sites only, so the ~13 unexercised &MigrationError{} sites report green, making the README:180-181 refusal-inventory guarantee over-broad; preferred fix is extending the AST walk to MigrationError composite literals. Positive controls confirm the harness can fail. Not stop-the-line: design is sound, scaffolding exists. See TASK-260830-1qf777_review-verdict.md and TASK-260830-1qf777_review-mutant-evidence.log.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-d4aa34, pid=29372, exit=0)
CROSS-LANE FINDING - read this first. All three m0 leaves in flight hit the same meta-defect in the same review round: a derived-inventory gate scans ONE form of a construct and is blind to an equivalent duplicate, so clauses pass the gate while their real behaviour is unpinned. On the object-store leaf the inventory counted any call including a direct unit call rather than production reachability. On the record-schema leaf it scans requireExactMembers and misses requireMemberSet, a byte-identical duplicate, which is why a live defect survived. On this leaf it walks configError call sites only, so roughly thirteen MigrationError composite literal sites report green and the README:180-181 refusal-inventory guarantee is over-broad.

THE PROPERTY, and it now applies to every derived gate you write or touch: the gate must prove its own coverage. It is not enough to derive an inventory; the derivation must assert that the set of constructs it scans is COMPLETE for the package, and that assertion must redden when an equivalent construct is added outside it. Extending the AST walk to MigrationError composite literals is the right first step, but do not stop at naming that one form - make the scan derive its target set and assert completeness so a third refusal form reddens the suite. Count a site as covered only when reached through a real production entry point rather than a direct helper call. A gate that cannot fail when someone adds a second copy of a check is decoration.

The review confirms behaviour is correct at the real Migrate and AssessCompatibility entry points; this is evidence work. Six delete/narrow mutants of production refusal gates leave go test ./internal/config green:

1. migration.go:281-287 existing-backup integrity check, content plus owner-only permissions. This is the headline prove-the-backup gate and it is silently deletable. Add stale-bytes and 0644-permission negative cases asserting ErrMigrationBackup, an unchanged source, an unchanged pre-existing backup, and no .ax-config-* leak.
2. migration.go:182-184 ErrMigrationTarget is never produced by any test; the TargetVersion==Version1 clause is free-floating.
3. migration.go:302-304 ErrMigrationSourceAbsent and the non-regular-source TOCTOU recheck are unexercised.
4. The disclosure-choice gate is tested only for empty; an invalid non-empty value falls through to an encode error instead of ErrMigrationChoiceRequired.
5. ErrMigrationRecovery, the rollback-failed crash branch, has no test. The existing faultMigrationFileSystem is one injection point away - add write/publish backup and write/replace source.

Per the front-loaded DoD each negative case must isolate its clause so it cannot pass on an earlier branch, and must redden when only its target clause is weakened.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:2634d51f21c2e3dc8fe5e00ba0ea646563892e7aaf37a62c49e784cbb46cf33e rationale="Rank-1 under the claude-exclusive policy: close the six unpinned migration gates and the self-coverage property the refusal inventory lacks."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260901-335354, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260901-335354)
Rework RUN-260901-335354 closes CR-TASK-260830-1qf777-1 rev1. All six unpinned migration gates now have isolated negative tests driven through the real Migrate entry, and the derived refusal inventory now proves its own coverage rather than naming one more form. Production change: every refusal is constructed through exactly one instrumented package-level constructor per error type (configError/DocumentError pre-existing, new loaderError/Error over 19 rewritten literal sites, new migrationError/MigrationError over 18). deriveRefusalInventory derives the error-type set from Error() string declarations, requires exactly one constructor per type, reports any composite literal built outside its constructor, and fails a full-package run on an unexercised site - so a duplicated constructor, a bypass literal, a fourth error type, or a never-reached site all redden. Also removed a dead guard rather than marking it subsumed: the two absent-source refusals in migrate collapsed to one reachable guard. Four subsumed sites each name their subsuming check and are ratcheted by TestRefusalSubsumptionInventoryIsPinned; two of them are now honestly covered by the new TestResolvePathsPopulatesEveryOverrideRegistryClass and TestLoadRefusesAbsentConfigWhoseParentIsNotResolvable. Findings: os.Getwd does not fail on darwin when the cwd is unlinked underneath the process, so loader.go capture-working-directory has no reachable test on a supported host and stays a pinned fail-closed guard; the remove-backup-staging fault is the one durable failure whose staging file necessarily survives, asserted owner-only instead of absent. Mutant sweep 22 applied, 22 died, none by compile error (M01/M02/M04 were rewritten after their first form died on a Go build failure), including M19-M22 self-coverage mutants. Gates all exit 0 as standalone processes: gofmt -l, go build ./..., go vet ./..., go test ./..., go test ./... -cover (internal/config 91.8%, was 83.9%), tracecheck, go mod verify, git diff --check, and go test ./internal/config/ under env -i with an empty HOME. Ownership digest re-pinned to 22d0ced7. See TASK-260830-1qf777_rework-evidence.md, TASK-260830-1qf777_mutant-sweep.log, TASK-260830-1qf777_mutants.py, TASK-260830-1qf777_logbook.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-335354, pid=53157, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:819114cc7f1d9457c51f6d0fa29fa21a958980fe41620c157909136984c2a721 rationale="Rank-1 under the claude-exclusive policy: verify the six migration gates now redden and the refusal inventory proves its own coverage."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-de6b47, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-de6b47)
Review verdict CR-TASK-260830-1qf777-2 rev2: CHANGES REQUESTED (RUN-260901-de6b47). Verified against the exact candidate tree (git write-tree re-derived a9b9e5fa). All six rev1 findings are closed and hold under an independent attack: I wrote 40 of my own mutants against the production gates, 38 died and the 2 survivors are provably subsumed rather than bypasses. Bounds die in both directions (max_parallel_chunks widened past 32 AND below 1), vocabularies die to narrowing, the backup integrity gate dies clause-by-clause (content, owner-only perms, whole gate), DisallowUnknownFields and both duplicate backend_id rejections die, a torn non-atomic replacement dies, backing up the new document instead of the original dies. I tried to game TestRefusalSubsumptionInventoryIsPinned - it compares only the COUNT of config-refusal-subsumed markers, never their positions - by MOVING a marker so the count stayed 7; it reddened on the unexercised-site audit, so the ratchet holds and that is not a finding. Both traceability gates fail closed: a fabricated section:99.9 binding is rejected as self-minted, an ownership edit without re-pinning is rejected on the projection digest. Every repo gate green including go test ./... -cover (internal/config 91.8%) and go test ./internal/config/ under env -i with an empty HOME.

Rejected on three evidence gaps, not on migration behavior.

F1. loader.go:187 swaps os.Stat for os.Lstat, and Load uses that ONE seam for two different questions - the regular-file contract on ConfigFile, and the directory-kind check in validateRootKinds and loadAbsentConfig. Only the first needs Lstat. Driving the real LoadOS entry: a symlinked DataRoot now fails ErrRootNotDirectory and an absent config under a symlinked parent fails ErrConfigParentNotDirectory. A symlink to a directory IS a usable directory and symlinked XDG/Application Support roots are a mainstream dotfiles layout, so this refuses a working setup with an error class that says the path is not a directory when it is one. Nothing asserts the directory outcome in either direction, the README does not mention it, and loader.go:187 carries no comment; my revert-to-os.Stat mutant dies only on the config-FILE symlink test, which confirms the directory consequence is entirely unpinned. I could NOT establish the spec position - SPEC.md is not in this repo and the specpin lock carries no prose - reporting that as unknown rather than inferring it. Resolve explicitly either way: cite the Section 6 clause and add negative tests for a symlinked root and symlinked parent plus a README sentence, OR split the seam so Lstat guards only the ConfigFile regular-file contract and add the positive test that a symlinked root loads.

F2. MigrateOS has 0.0 percent coverage. The README this CR writes names it an explicit Section 6.4/6.5 durable migration entry, and LoadOS - the sibling OS entry - carries four production-entry tests. The composition of OSInputs with Migrate, including that OSInputs now hands it os.Lstat, is never driven. Add one MigrateOS test that migrates a real on-disk file through explicit Overrides and asserts backup plus replacement, and one refusal lane.

F3. (*MigrationError).Error() has 0.0 percent coverage, so the migration.go:50 claim that it renders no machine-local path or document content is unproven while replaceDurably joins raw os errors carrying absolute paths into the wrapped chain. The equivalent loader claim IS proven by TestLoadErrorsDoNotEchoSelectedPathValues and TestErrorFormattingNeverEchoesWrappedDetails; extend one to *MigrationError.

Not stop-the-line: no external blocker, no human-only decision. See TASK-260830-1qf777_review-verdict-rev2.md, TASK-260830-1qf777_reviewer-mutant-evidence-rev2.log, TASK-260830-1qf777_reviewer-mutants-rev2.py.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-de6b47, pid=90070, exit=0)
Round-2 review: all six rev1 findings closed, verified independently with 40 reviewer-written mutants of which 38 died and both survivors are provably subsumed. Three findings remain, and F1 is a behaviour change nobody declared.

F1. os.Stat was changed to os.Lstat, so a symlinked directory is now silently refused. That is a real behavioural change in the production path and it is untested in BOTH directions and undocumented - loader.go:187 carries no comment and the reviewer R16 mutant reverting it stays green. Decide deliberately: if refusing a symlinked directory is intended, say so in a comment, pin it with a positive case for a regular directory and a negative case for a symlinked one, and check the pinned SPEC actually requires it. If it is not intended, revert it. Do not leave an undeclared behaviour change in a migration path.

F2. MigrateOS is the OS production entry and has 0.0 percent coverage. The DoD requires cases to drive real production entry points; a migration Story that never executes its own OS entry has not proven the thing it delivers. Cover it end to end with a real temporary directory.

F3. (*MigrationError).Error() has 0.0 percent coverage, so the migration.go:50 claim that it renders no machine-local path or document content is unproven - while replaceDurably joins raw os errors carrying absolute paths into the wrapped chain, which is exactly how such a leak happens. The equivalent loader claim is already proven by TestLoadErrorsDoNotEchoSelectedPathValues and TestErrorFormattingNeverEchoesWrappedDetails; extend one of them to *MigrationError rather than writing a new bespoke assertion.

This is the final leaf of the configuration Story. Once accepted the Story integrates and unblocks downstream work, so finish the evidence cleanly rather than narrowly.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:2634d51f21c2e3dc8fe5e00ba0ea646563892e7aaf37a62c49e784cbb46cf33e rationale="Rank-1 under the claude-exclusive policy: resolve the undeclared Lstat behaviour change and cover the two uncovered production entry points."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260901-d41916, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260901-d41916)
F1 RESOLVED BY THE ORCHESTRATOR AGAINST THE PINNED SPEC. The reviewer could not check this because SPEC.md is not in this repository; I read it at the pinned commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c. Do not re-derive this, and do not resolve F1 the other way.

DIRECTORY ROOTS - follow symlinks. The spec resolves symlinks, it does not refuse them:
  SPEC.md:258  Receivers still resolve symlinks/reparse points beneath an allowed root before mutation.
  SPEC.md:845  MUST be path-disjoint from every workspace, provider native-store, task-board staging, credential, authentication, durable-object, and runtime root AFTER symlink/reparse-point resolution.
  SPEC.md:5270 Authorities MUST be pairwise path-disjoint; neither a symlink nor a reparse point may make them overlap.
The normative model is resolve-then-check, never refuse-because-symlink. Refusing a symlinked directory root is therefore an INVENTED constraint - the same defect class that cost two rounds on the record-schema leaf - and it breaks a mainstream dotfiles layout while reporting ErrRootNotDirectory about something that is a directory.

Take the reviewer second branch: split the seam so Lstat guards only the ConfigFile regular-file contract, while validateRootKinds and loadAbsentConfig keep following symlinks. Add the positive case that a symlinked DataRoot and a symlinked config parent both LOAD, and keep the existing config-file symlink refusal test green.

ONE THING TO CHECK, do not assume. SPEC.md:738 requires the configuration file to be an existing regular file, or a not-yet-created regular-file path whose parent exists. But the spec idiom for the same question elsewhere, SPEC.md:2648 for plugin executables, is Symlinks MUST be resolved before comparison, the target MUST be a regular file. So refusing a symlink that points at a regular config file may itself be stricter than the pinned contract. Decide it explicitly with a citation: either keep the refusal and cite 738 as the stricter closed reading, or resolve-then-require-regular per the 2648 idiom. State the choice in the README paragraph that describes the loader kind checks and pin whichever way you choose in both directions. Do not leave it undeclared - that is what made F1 a finding in the first place.
agent completed: [implementer] developer (claude) (exit=143)
spawn run completed: claude (run=RUN-260901-d41916, pid=15672, exit=143)
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:2634d51f21c2e3dc8fe5e00ba0ea646563892e7aaf37a62c49e784cbb46cf33e rationale="Rank-1 under the claude-exclusive policy: respawn with the spec-backed F1 resolution so the producer does not guess a constraint the pinned SPEC does not declare."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260901-12b8e2, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260901-12b8e2)
spawn run RUN-260901-d41916 cancelled by operator; operator action required; reason: no operator reason supplied
ROUND 3 (claude/opus-5) — F1, F2, F3 closed; committed d583fe8 (signed, verified) on task-board/story/STORY-260830-jeaivu.

F1 SYMLINK STANCE — resolved as the orchestrator directed, against the pinned SPEC, and now DECLARED rather than implied. Read seam (loader.go:199 os.Stat) follows symlinks and applies the Section 3.2 value kind to the resolved target; the mutating seam (osMigrationFileSystem.Stat, os.Lstat) does not, because an atomic rename replaces the operator link rather than the document it points at and the SPEC is silent about that mutation. The one thing you asked me to decide, not assume: I took the SPEC.md:2648 resolve-then-require-regular idiom rather than the stricter reading of SPEC.md:738, so a configuration file symlinked onto a regular file LOADS. Both stances are stated in the README paragraph on loader kind checks and pinned in both directions at the real LoadOS and MigrateOS entries. Mutants M23 (widen the mutating seam) and M24 (narrow the read seam — the original F1 defect) both redden.

PRODUCTION CHANGE beyond the three findings: replaceDurably now re-inspects the selected file kind/mode BEFORE any durable write. It previously refused a symlinked source only after publishing config.toml.bak.<version>, so a refused migration still mutated the directory. Mutant M26 restores the old ordering and dies on the leaked staging file.

F2 — MigrateOS 0.0% -> 100.0%. internal/config/migration_os_test.go drives the real OS entry against a real directory tree with explicit overrides for all five path classes: v1-with-legacy-terminal to v3 with exact backup bytes/0600 mode/preserved source mode/LoadOS read-back/idempotent second run; the both-directions symlink pair; and an unknown-platform refusal.

F3 — (*MigrationError).Error 0.0% -> 100.0%. Extended TestErrorFormattingNeverEchoesWrappedDetails as instructed rather than writing a bespoke assertion, and the MigrateOS symlink subtest additionally asserts the REAL production MigrationError does not echo the temp root or the link path.

FINDING WORTH YOUR ATTENTION — one mutant SURVIVED on first run and it was a real gap, not a script artifact. M27 (MigrateOS discards its own OSInputs error and continues with empty Inputs) stayed GREEN because my test asserted only errors.Is(err, ErrInvalidContext), and ResolvePaths refuses the same platform later with the same sentinel. Gate present, reachable, and proving nothing — subsumed downstream. Fixed by asserting the refusing (*Error).Operation is "capture OS inputs" versus the downstream "validate inputs"; M27 then died. Recorded in the logbook rather than quietly patched.

TRACEABILITY — MigrateOS is now a registered production owner: new acceptance case config-durable-migration-os-entry referenced from the section:6.4 and section:6.5 bindings. The ownership projection digest gate failed closed on the change and the reviewed constant plus both count assertions (34 -> 35) were updated deliberately.

MUTATION — 29/29 dead, zero survivors, restored tree green. Honest note: M25 (delete the source-inspection guard) dies by nil-pointer panic rather than a clean assertion, because deleting the guard leaves info nil. Reported as a panic in the artifact, not dressed up as an assertion failure.

GATES, all real exit codes on the restored tree: go test ./... =0; go test ./... -cover =0 (internal/config 92.3%); go test ./internal/config -race =0; go vet ./... =0; go build ./... =0; gofmt -l . =0 empty; tracecheck full =0; tracecheck assigned 9 scopes =0; go generate ./internal/catalog + git diff --exit-code =0; go mod verify =0; git diff --check =0; mutants.py =0. Three consecutive config runs identical.

I also corrected a now-false claim in the round-1 results artifact (it credited the os.Lstat loader change as a tightening) rather than leaving a stale artifact on the board.

Artifacts: TASK-260830-1qf777_rework-r3-evidence.md, TASK-260830-1qf777_mutant-sweep-r3.log, updated _mutants.py, _logbook.md, _results.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-12b8e2, pid=26802, exit=0)
spawn autonomous recovery: run RUN-260901-12b8e2 queued successor RUN-260901-877162 (attempt 1/3, model=claude-opus-5): Change Request construction for TASK-260830-1qf777 failed: change_request_base_authority_mismatch: the STORY-260830-jeaivu candidate provenance disagrees (selected=48db30b59e5e1bbc5e0cf73ec2e0e0eec3d215d1 current=48db30b59e5e1bbc5e0cf73ec2e0e0eec3d215d1 checkpoint=c0d42491d08bc6dca6c2322b84085cfce50b1217 branch=d583fe82ccd48ad23382c6b202545c7041fda29b head=d583fe82ccd48ad23382c6b202545c7041fda29b)
spawn run started: [implementer] developer (claude) (run=RUN-260901-877162)
ROUND 4 (RUN-260901-877162, recovery successor). NO production or test source changed. This round fixed the delivery-provenance defect that made round 3 exit 0 while Change Request construction failed.

ROOT CAUSE. This leaf is now the FINAL unresolved leaf of the Story (2iint0 and 17suox are done), so its CR resolves as kind=story_final - confirmed on the live record CR-TASK-260830-1qf777-2 ("kind": "story_final"). Revisions 1 and 2 were built while siblings were open, i.e. as task_delta, which never reaches the story-final authority gate. convergeStoryFinalBaseAuthority (changerequest.go:653) admits only two shapes: (1) an UNCOMMITTED candidate with branch==head==checkpoint, or (2) an exact one-commit candidate that additionally needs checkpoint==selected. Observed state failed both: round 3 HAND-COMMITTED its work as d583fe8, so branch/head were past the checkpoint c0d42491, and shape 2 did not apply because checkpoint != selected (48db30b). Committing is the boards step - the two accepted sibling leaves reached the branch via worktree checkpoint (020d0b6, c0d42491, board-generated doubled-ID subjects) while d583fe8 carries a hand-written subject.

REPAIR. git reset --mixed c0d42491 in the managed worktree. Every byte preserved, proven by tree hash: d583fe8^{tree} and the post-reset worktree tree (add -A into an alternate index, exactly how Take() builds the candidate) are BOTH 9700e40f4bc9787d495864cd97654b2a8864121b. Branch and HEAD now equal the checkpoint; d583fe8 (signed, G oparin@me.com) stays reachable in the reflog; it touched only source paths, nothing board-owned. Candidate is 9 modified tracked files plus 5 new untracked non-ignored files. Gate re-checked after repair: shape 1 satisfied, upstream==selected==48db30b, selected tree==upstream tree, 48db30b is an ancestor of c0d42491.

ROUND 3 RE-VERIFIED, NOT TRUSTED. I re-derived its claims rather than accepting the note. Mutation harness re-run from scratch: 29 RED, 0 GREEN, 0 survivors, restored tree green, exit=0. F2 closed - migration.go:61 MigrateOS 100.0%. F3 closed - migration.go:53 (*MigrationError).Error 100.0%. F1 inspected in source: read seam loader.go:199 os.Stat follows symlinks and applies the kind check to the resolved target; mutating seam migration.go:267 os.Lstat does not; both declared in comments at both sites and README.md:218-227, pinned both directions by mutants M23 (widen) and M24 (narrow), both RED.

GATES, real exit codes, standalone processes, no tee/pipes, at the handoff state: go build ./... =0; go vet ./... =0; gofmt -l . =0 empty; go test ./... -cover -count=1 =0 (config 92.3%); go test ./internal/config -race =0; tracecheck full =0; tracecheck 8 assigned scopes =0; cataloggen -check =0; go generate ./internal/catalog + git diff --exit-code on catalog_gen.go =0; go mod verify =0; git diff --check =0; mutants.py =0; go test ./internal/config/ -count=1 under env -i with empty HOME and GOPROXY=off =0.

LOGBOOK PLACEMENT DECISION. No root LOGBOOK.md exists in this repo, and a parallel Story (CR-TASK-260830-3amrl9-5) is already introducing one. Creating a competing root file inside a story_final candidate about to integrate would collide with that in-flight work, so the round-4 finding went into the task-scoped TASK-260830-1qf777_logbook.md instead. Flagging it so the decision is reviewable rather than silent.

ONE THING FOR THE REVIEWER TO CHECK DELIBERATELY: the stale ready record CR-TASK-260830-1qf777-2 still carries base_oid=48db30b from the task_delta era. This round did not touch the CR store. Artifacts: TASK-260830-1qf777_rework-r4-evidence.md, TASK-260830-1qf777_mutant-sweep-r4.log, updated _logbook.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-877162, pid=60424, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:819114cc7f1d9457c51f6d0fa29fa21a958980fe41620c157909136984c2a721 rationale="Rank-1 under the claude-exclusive policy: verify the orchestrator-resolved symlink seam split against the pinned spec citations, plus MigrateOS and MigrationError coverage."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-8970a0, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-8970a0)
Review of CR-TASK-260830-1qf777-3 (tree 9700e40, verified by write-tree): CHANGES REQUESTED. Build/vet/gofmt clean; go test ./... all green; internal/config coverage 92.3%; tracecheck ok with sections 3.2/6.1-6.5/17.1/17.2/17.4 and assigned_scopes=9. Producer 29-mutant sweep re-confirmed all RED. Independent reviewer sweep of 20 additional mutants: 12 RED, 8 survived. Three survivors are on this tasks own AC and are blocking. F1: narrowing AssessCompatibility to source.major > reader.major+1 keeps the suite green, so AssessCompatibility(v3, reader 2.0.0) returns compatible - the exact SPEC 6.5 case A v1/v2 binary opening v3 is read-only diagnostic is untested. F2: every migration fixture is minimalValidConfigVersion, so no non-default member ever crosses a migration; deleting SafeBoundaryTimeoutSeconds or GracefulStopTimeoutSeconds from encodeVersion2, or SafeBoundaryTimeoutSeconds from currentWire, all survive - a v1 safe_boundary_timeout_seconds=42 silently becomes 300 with the suite green, against SPEC 6.4 retains all Configuration 1 members. F3: deleting file.Sync() in writeTempFile survives, so the SPEC 6.4 fsyncs it and the directory claim (also in README) has no test on the file half; the directory half is well covered and faultMigrationFileSystem already has the seam to close this. No production defect found - the shipped behavior passes every probe; the gap is missing negative/preservation evidence on a durable mutation. Full verdict: TASK-260830-1qf777_review-verdict-rev3.md; sweep log: TASK-260830-1qf777_reviewer-mutation-sweep-rev3.log.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-8970a0, pid=91488, exit=0)
Round-3 review: the symlink seam split is accepted and no production defect was found. Three blocking gaps remain, all on this task own acceptance criteria, and F2 is the serious one.

F2 - FIXTURE-SHAPED COVERAGE, and it hides silent data loss. Every migration fixture is minimalValidConfigVersion, so no non-default member ever crosses a migration. Deleting SafeBoundaryTimeoutSeconds or GracefulStopTimeoutSeconds from encodeVersion2, or SafeBoundaryTimeoutSeconds from currentWire, all survive: a v1 document carrying safe_boundary_timeout_seconds=42 silently becomes 300 and the suite stays green, against SPEC 6.4 which requires that migration retains all Configuration 1 members.

Do not add three cases for the three cited members. This is the same class that cost a round on the previous leaf: a fixture that only carries defaults cannot observe a dropped member, because the default is what a dropped member decays to. Build the preservation property instead - derive the member list from the versioned schema, populate EVERY member with a value distinguishable from its default, migrate, and assert each value survives. A member added later must then be covered without editing the test, and dropping any member from an encoder must redden.

F1 - narrowing AssessCompatibility to source.major > reader.major+1 keeps the suite green, so AssessCompatibility(v3, reader 2.0.0) returns compatible. The exact SPEC 6.5 case A, a v1 or v2 binary opening v3 being read-only diagnostic, is untested. Pin it at the production entry in both directions.

F3 - deleting file.Sync() in writeTempFile survives, so the SPEC 6.4 fsyncs it and the directory claim, repeated in the README, has no test on the file half. The directory half is already well covered and faultMigrationFileSystem has the seam; close the file half the same way.

This is the final leaf of the configuration Story. Once accepted the Story integrates and unblocks downstream work.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:2634d51f21c2e3dc8fe5e00ba0ea646563892e7aaf37a62c49e784cbb46cf33e rationale="Rank-1 under the claude-exclusive policy: build the member-preservation property that makes silent migration data loss observable."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260901-e6ac21, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260901-e6ac21)
Round 5 closes CR-TASK-260830-1qf777 rev3. All three blocking findings resolved, plus all six non-blocking observations.

F2 (the serious one). Not three cases for the three cited members - the reviewer was explicit that would re-create the defect. Built the property in three coupled parts: wireLeafTOMLPaths derives the Configuration 1.0.0 member set from reflect.TypeOf(rawV1{}); a saturated fixture must cover every derived path at a value distinguishable from the one a dropped member decays to, where the decay oracle is a minimal v1 migrated through the real Migrate (currentWire emits every member, so that document IS the default); and the whole loaded Configuration is compared before and after on v1->2.0.0, 2.0.0->3.0.0 continuing from that document, and a fresh direct v1->3.0.0. Ten members are pinned to a single valid value by the validators and cannot be saturated - each declares the alternate it claims is refused, and assertVersion1ExemptionIsReal drives that alternate through the real Load. Mutants N18-N23 widen each validator in turn; every one reddens the exemption self-check, so the list cannot outlive its justification. Fifteen drop-a-member mutants across encodeVersion2 and currentWire, including all three rev3 survivors, are RED.

F1. TestAssessCompatibilityPinsEveryPinnedVersionPairAtTheProductionEntry drives the full cross-product at the real entry. Versions come from the pinned catalog; expected mode from a test-local major parse, not from the production compareSemver it bounds. The test fails if either direction went unexercised. Narrowing to major+1 (the survivor), widening to >=, and deleting the gate are all RED.

F3. faultMigrationFileSystem.CreateTemp now wraps its file so the per-file Sync can be counted and failed. Two injected-failure cases joined TestMigrateLeavesTheSourceIntactAtEveryDurableFailurePoint and inherit its whole invariant set. The narrowing direction is separate: a successful migration must satisfy fileSyncCalls == createTempCalls, so fsyncing only the replacement reddens.

Non-blocking, also closed: obs 4 pins the replacement mode as EQUAL to the source (both directions); obs 3 adds a zero-progress writer asserting io.ErrShortWrite; obs 2 makes the subsumption pin resolve each file:clause key to exactly one marked site instead of comparing len(); obs 1 gives the v2 re-read a named subsumption marker. Obs 5 was NOT subsumed as claimed - the read-only branch returns before knownConfigVersion runs, so AssessCompatibility(04.0.0, reader 3.0.0) succeeds under the mutant; two cases added, R07 now RED. Obs 6 left as production behavior deliberately (clearing BackupPath would also hide a backup that WAS published on the staging-remove path) with the contract now stated on the type.

HONEST SWEEP FIGURE: 73 mutants, 71 RED, 1 SURVIVOR, 1 positive control GREEN, 0 invalid. The survivor is R13 and it is reported as a survivor, not folded into the RED count. encodeVersion2 directory-collection passthrough is unreachable - Migrate reaches that encoder only with a v1 source and rawV1 declares no directory collection. TestVersion1SourcesNeverCarryDirectoryCollections pins that precondition from the derived member set and a real load; the harness declares it in EXPECT_SUBSUMED with the subsuming check named and prints SUBSUMPTION CLAIM NO LONGER NEEDED if it ever starts dying. Prior rounds 29 producer + 18 reviewer mutants were re-run against the WHOLE package rather than a per-mutant -run mask.

GATES, real exit codes, standalone processes, no tee or pipes: go build ./... =0; go vet ./... =0; gofmt -l . =0 empty; go test ./... -cover -count=1 =0 (config 92.3% -> 93.1%); go test ./internal/config -race =0; tracecheck =0; cataloggen -check =0; go generate ./internal/catalog + git diff --exit-code =0; go mod verify =0; git diff --check =0; sweep batches [0:18] [18:36] [36:55] [55:73] all =0; go test ./internal/config/ under env -i with empty HOME and GOPROXY=off =0.

TRACEABILITY: four tests joined config-durable-migration, one joined config-read-only-downgrade. reviewedOwnershipCanonicalSHA256 moved 1abb92b1 -> 8885305e; that is the review gate working as designed, not formatting drift. No new section binding, no new acceptance case ID, no CLI/doctor/backend capability advertised.

INCIDENT FOR THE REVIEWER TO SEE: while editing ownership.v0.5.0.json I ran git checkout -- on that single path to undo a formatting-churning scripted rewrite. The story-final candidate is uncommitted by contract, so that discarded ~100 lines of prior-round work on the file. Recovered byte-identically from the pre-mutation tree copy and verified equal to d583fe8:internal/traceability/ownership.v0.5.0.json; the replacement edit was made surgically as text, producing a 20-line diff. Nothing was lost, but the class is worth flagging: in an uncommitted candidate a single-path git checkout is as destructive as a repo-wide one.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-e6ac21, pid=39974, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:819114cc7f1d9457c51f6d0fa29fa21a958980fe41620c157909136984c2a721 rationale="Rank-1 under the claude-exclusive policy: verify the member-preservation property makes a dropped migration member observable, plus the 6.5 compatibility and fsync pins."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-54aa4b, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-54aa4b)
Reviewer verdict CR rev4: CHANGES REQUESTED (to-dev). Evidence: TASK-260830-1qf777_review-verdict-rev4.md, TASK-260830-1qf777_reviewer-mutation-sweep-rev4.log, TASK-260830-1qf777_reviewer-home-probe-rev4_test.go.

Verified worktree hashes to candidate tree 0ea2e47f55b3ce32227d69763c4bcc0118b5453d. Green: go build/vet clean; go test ./... -cover all 9 packages ok (internal/config 93.1%); gofmt -l internal/ clean; tracecheck global ok and -section 3.2 6.1..6.5 17.1 17.2 17.4 assigned_scopes=9. Ownership pin not taken on trust: all three new acceptance cases and their 21 named test declarations were confirmed to exist by an independent source scan.

All three rev3 blocking findings (F1 read-only one-major step, F2 member preservation, F3 temp-file fsync) are fixed and each was re-killed with the exact mutant that survived before. Independent 44-mutant sweep: 41 killed, 3 equivalent, 2 survivors. Every AC gate (backup, migration, unknown-field refusal, bounds, duplicate backend rejection, read-only downgrade) dies under a narrowing mutant, not only a delete-only one. The derived refusal inventory itself was attacked and holds.

BLOCKING B1: this change removed the OSInputs capture-user-home refusal and replaced it with a deferred homeDirError field, but the only test hand-sets that unexported field on a fixture (self-minted evidence). No test drives the home-derived platform defaults at a real entry - every LoadOS/MigrateOS test overrides all five path classes. Two mutants survive the whole suite: homeErr=nil after the capture (operator loses the cause of an unavailable default) and home="" after the capture (every home-derived platform default refuses). A verified test that kills both at the real LoadOS entry is attached as TASK-260830-1qf777_reviewer-home-probe-rev4_test.go; it passes clean and fails under each mutant. No production-code defect was found on any probe.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-54aa4b, pid=36871, exit=0)
Round-4 review: all three rev3 blocking findings fixed and each re-killed with the exact mutant that survived before. 44 mutants, 41 killed, 3 equivalent. Every acceptance gate now dies under a NARROWING mutant rather than delete-only, and the derived refusal inventory was itself attacked and held. One finding left.

B1 - SELF-MINTED EVIDENCE, the same class as a self-comparing assertion. This change removed the OSInputs capture-user-home refusal and replaced it with a deferred homeDirError field. The only test that exercises it HAND-SETS that unexported field on a fixture, so the test constructs the very state it claims to observe and can never witness whether production ever reaches it. No test drives the home-derived platform defaults at a real entry: every LoadOS and MigrateOS test overrides all five path classes.

Two mutants survive the whole suite as a result:
  homeErr = nil after the capture - the operator loses the cause of an unavailable default.
  home = "" after the capture - every home-derived platform default refuses.

The reviewer attached a verified test that kills both at the real LoadOS entry: TASK-260830-1qf777_reviewer-home-probe-rev4_test.go. It passes clean and fails under each mutant. Adopt it, or write an equivalent that reaches the state through production rather than through a struct literal - do not simply widen the hand-set fixture.

The general rule, since this is now the third shape of it on this board: evidence whose expected side, or whose precondition, is produced by the test rather than by the production path proves only that the test is self-consistent. Reach the state the way production reaches it, or the case pins nothing.

No production defect was found on any probe. This is the last open finding on the final leaf of the configuration Story; once accepted the Story integrates and unblocks downstream work.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:2634d51f21c2e3dc8fe5e00ba0ea646563892e7aaf37a62c49e784cbb46cf33e rationale="Rank-1 under the claude-exclusive policy: replace the self-minted home-capture fixture with a case that reaches the state through the real LoadOS entry."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260901-22ea2b, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260901-22ea2b)
Round 6 (rev5 candidate) closes CR-TASK-260830-1qf777 rev4. B1 was the only open finding and it is closed. No production code changed this round: internal/config/loader.go and internal/config/migration.go are byte-identical to the rev4 candidate. Only tests, the ownership registry, its reviewed digest, and one README paragraph moved.

B1 - SELF-MINTED EVIDENCE FOR A REMOVED REFUSAL. The state is now reached the way production reaches it: the cases set or clear the exact variable os.UserHomeDir reads (HOME on unix, USERPROFILE on Windows, home on plan9) and let the real OSInputs capture run. Nothing assigns Inputs.HomeDir or Inputs.homeDirError. The expected cause is the message a real os.UserHomeDir call returns in the same process environment, not a literal, because os.UserHomeDir constructs a fresh error per call and identity comparison is impossible.

TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass is the narrowing direction the reviewer asked for. overridesExcept supplies a real admissible value for every registry class except the one under test, so exactly one class falls through to the platform-default layer; without that isolation the first home-derived class masks every later one and a cause dropped at any single site is invisible. Each subtest requires ErrPlatformDefaultUnavailable, the refusing class and source to be exactly the isolated one, the captured cause to be carried, and the rendered message to not echo the raw OS cause.

TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome resolves TWO distinct real homes per lane. One home cannot distinguish the captured home reaching the defaults from some other value that happened to match; the second run pins that every home-derived class moved with the capture.

TestMigrateOSMigratesTheHomeDerivedConfigurationAtTheRealProcessEntry covers the second consumer of the same capture. Every other MigrateOS case overrides all five classes; here the captured home is the only thing selecting the file the durable mutation rewrites.

LANE COVERAGE, not host-gating. An earlier draft pinned only hostPlatform and left the four Linux/WSL2 cause-drop mutants surviving on a darwin host. homeDrivenPlatforms instead drives macOS, Linux and WSL2 through the real LoadOS entry from one unix host, because os.UserHomeDir reads $HOME on both and the rendered separator matches. All four turned RED. Windows is skipped with a stated reason: production derives no path default from the user home there, so a home-capture case has nothing to pin - a real absence in the Section 3.2 Windows layout, not missing coverage.

TestResolvePathsDefersButPreservesHomeLookupFailure was NOT widened - the reviewer ruled that out. It is kept and relabelled as the injected half only, naming the two cases that pin the real capture.

SWEEP: 16 mutants, 16 RED, 0 SURVIVOR, 0 INVALID. Whole internal/config package with -count=1, never a -run mask. Positive control green before, restored tree green after, every anchor asserted to occur exactly once. Both rev4 survivors are RED (H01 homeDirError:=nil, H02 home:=""). H03-H05 attack the capture other ways (constant home, substituted self-minted cause, corrupted home). H06-H13 drop the cause at each of the eight home-derived sites individually. H14 deletes the gate at macOS data-root. H15/H16 drift the layout.

GATES, real exit codes, standalone processes, no tee or pipes: go build ./... =0; go vet ./... =0; gofmt -l . =0 empty; go test ./... -cover -count=1 =0 (all 9 packages ok, internal/config 93.1% -> 93.7%); go test ./internal/config -race -count=1 =0; tracecheck global =0; tracecheck -section 3.2 6.1..6.5 17.1 17.2 17.4 assigned_scopes=9 =0; cataloggen -check with its real -metadata/-contracts/-output flags =0; go generate ./internal/catalog + git diff --exit-code =0; go mod verify =0; git diff --check =0; go test ./internal/config -count=1 under env -i with empty HOME =0. The empty-HOME hermetic run matters specifically here: the new cases must not depend on the developer real home, and they do not.

TRACEABILITY: two tests joined AC-PATH-001, one joined config-durable-migration-os-entry. reviewedOwnershipCanonicalSHA256 moved 8885305e -> 7badbfe8; that is the review gate working as designed. The ownership JSON was edited surgically as text - 12 added lines, no reformatting, no repeat of the round-5 checkout incident. No new section binding, no new acceptance case ID, no CLI/doctor/backend capability advertised.

README: one paragraph on why the home lookup failure is deferred rather than refused at capture time (Windows derives no default from the user home; several classes are satisfied by an XDG override), that the captured cause travels into every home-derived class refusal, and that both halves are proven at the real entries. No capability claimed.

Evidence: TASK-260830-1qf777_rework-r6-evidence.md, TASK-260830-1qf777_home-mutation-sweep-rev5.log, TASK-260830-1qf777_home-mutants-rev5.py.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-22ea2b, pid=12725, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-fable-5-1/high pair_source=explicit match=not_recommended snapshot=sha256:b5c63b2693e54f4476e75d104bf1d12d151a31c28a199437ea6b39c844b95447 rationale="Only surviving pair while the claude-opus-5 limit group is unavailable; this review also gives a first read on how claude-fable-5-1 performs in the reviewer seat."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-e9845b, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-e9845b)
spawn run RUN-260901-e9845b cancelled by operator; operator action required; reason: no operator reason supplied
agent completed: [reviewer] reviewer (claude) (exit=143)
spawn run completed: claude (run=RUN-260901-e9845b, pid=90857, exit=143)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:ff2e48be132a3e1446f7d2b8ef259d90312efc801ffd6946243540882addd00f rationale="Rank-1 recommendation: review the member-preservation rework on Opus, keeping this lane on the reviewer that has carried it."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260901-fba8aa, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260901-fba8aa)
Reviewer verdict CR-TASK-260830-1qf777-5 rev5: ACCEPTED (RUN-260901-fba8aa). Worktree tree re-derived independently as 613bb21a, equal to the declared candidate, and re-derived again after every probe.

VERIFIED, NOT TRUSTED: the round-6 claim that no production code changed. Splitting the rev4 and rev5 patch resources per file and comparing hunks byte-for-byte shows loader.go, migration.go, schema.go, validation.go, writer.go, go.mod and go.sum identical. Revision 5 moves only loader_home_test.go (new), migration_os_test.go (one added test), loader_test.go (a comment block), README.md (one paragraph), ownership.v0.5.0.json (three owner entries) and the reviewed digest constant.

B1 CLOSED. Both rev4 survivors are RED: homeDirError=nil dies on TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass, home="" dies on TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome. Two further attacks on the same seam also die - a constant home, and a self-minted cause substituted for the real os.UserHomeDir error. Nothing in loader_home_test.go assigns HomeDir or homeDirError; the state is reached by setting or clearing the exact variable os.UserHomeDir consults. The surviving hand-set test is relabelled as the injected half only and names the two real-entry cases, which is the disposition the finding asked for rather than a widened fixture. MigrateOS and (*MigrationError).Error are both at 100% function coverage.

INDEPENDENT SWEEP: 36 reviewer-written mutants, one edit at a time, whole internal/config package with -count=1 and never a -run mask, tree restored and re-verified between each. 31 RED, 4 survivors, 1 positive control green. No mutant counted RED on a Go build failure - A01, B07 and B08 were rewritten into compiling forms after their first form died on a compile error, and C03s kill was reproduced by hand to name the real gate (the package-level unexercised-refusal-site audit). Every acceptance gate dies under a narrowing or widening mutant, not delete-only: backup integrity clause by clause, wire-member drops on the derived preservation property, DisallowUnknownFields, both bound directions at the production entry, both duplicate backend_id rejections, the rev3 read-only downgrade survivor, and the symlink seam split in both directions.

I attacked the derived gates themselves rather than reading them. A bypass composite literal, a second byte-identical constructor and a brand-new never-exercised site all redden. The subsumption escape hatch holds BOTH ways: deleting a declared marker reddens, and - the attack that matters - attaching a self-declared subsumption marker to a synthetic never-exercised site does NOT silence it. A producer cannot buy its way out of the inventory with a comment. Traceability fails closed on a fabricated owner; the three added ownership entries are exactly the three tests claimed and all three exist.

FOUR SURVIVORS, reported as survivors, none a finding. B19 (absent-source guard || narrowed to &&) is equivalent: from a successful Load, configPresent and a non-nil configuration are set together, so the operands are never independently true. B17/B20 add a brand-new vocabulary member that no case supplies, which no finite sampling suite can observe; every widening at a value the suite DOES supply dies (B21, C04, B23), and control B22 shows the identical arbitrary widening survives on directory.mode, a sibling vocabulary outside this delta. That is the existing package standard, not a regression here. Killing the class needs the vocabulary derived from a pinned SPEC literal with set equality asserted - recorded as an observation for a future leaf.

GATES, real exit codes at the candidate tree: go build =0; go vet =0; gofmt -l . =0 empty; go test ./... -cover -count=1 =0 (internal/config 93.7%); go test ./... -count=1 post-sweep =0; go test ./internal/config -race -count=1 =0; go mod verify =0; git diff --check =0; tracecheck global =0; tracecheck -section 3.2 6.1-6.5 17.1 17.2 17.4 =0 assigned_scopes=9; go test ./internal/config -count=1 under env -i with HOME= =0.

No production defect found on any probe. Evidence: TASK-260830-1qf777_review-verdict.md, TASK-260830-1qf777_reviewer-mutation-sweep-rev5.log, TASK-260830-1qf777_reviewer-mutants-rev5.py.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260901-fba8aa, pid=72069, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-1qf777_spawn-log_-implementer--developer--codex-_RUN-260901-7ab441.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-implementer--developer--codex-_RUN-260901-7ab441.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_results.md](file://TASK-260830-1qf777/TASK-260830-1qf777_results.md) — Implementation evidence for config migration and read-only downgrade; corrected in round 3 for the symlink stance and superseded figures
- [TASK-260830-1qf777_clause-mutation-sweep.log](file://TASK-260830-1qf777/TASK-260830-1qf777_clause-mutation-sweep.log) — 113/113 configuration refusal and narrowing mutants killed; zero survivors or invalid mutants
- [TASK-260830-1qf777_change-request_rev1.patch](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev1.patch) — Change Request CR-TASK-260830-1qf777-1 revision 1 candidate patch (repository_delta=present, 18 changed paths)
- [TASK-260830-1qf777_change-request_rev1-validation.log](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev1-validation.log) — Change Request CR-TASK-260830-1qf777-1 revision 1 bounded validation log
- [TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-d4aa34.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-d4aa34.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_review-verdict.md](file://TASK-260830-1qf777/TASK-260830-1qf777_review-verdict.md) — Reviewer verdict, current revision CR rev5 (RUN-260901-fba8aa): ACCEPTED, with the 36-mutant independent sweep and gate results
- [TASK-260830-1qf777_review-mutant-evidence.log](file://TASK-260830-1qf777/TASK-260830-1qf777_review-mutant-evidence.log) — Mutation + probe transcript: baseline green, 6 surviving delete/narrow mutants, 2 positive controls, gate-behavior probes
- [TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-335354.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-335354.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_rework-evidence.md](file://TASK-260830-1qf777/TASK-260830-1qf777_rework-evidence.md) — Rework round closing CR-TASK-260830-1qf777-1: derived refusal inventory with self-coverage, six negative gates, 22/22 mutants dead, gate exit codes
- [TASK-260830-1qf777_mutant-sweep.log](file://TASK-260830-1qf777/TASK-260830-1qf777_mutant-sweep.log) — Full 22-mutant sweep transcript with per-mutant exit codes and restored-tree verification
- [TASK-260830-1qf777_mutants.py](file://TASK-260830-1qf777/TASK-260830-1qf777_mutants.py) — Mutation harness, round 3: 29 mutants including the symlink-seam widening/narrowing pair, the pre-write placement reorder, and the MigrateOS entry mutants
- [TASK-260830-1qf777_logbook.md](file://TASK-260830-1qf777/TASK-260830-1qf777_logbook.md) — Task logbook through round 6: deferred error fields are untestable via struct literals; a capture-level gate proof says nothing about its propagation sites; a host-gated survivor was an unexamined assumption
- [TASK-260830-1qf777_change-request_rev2.patch](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev2.patch) — Change Request CR-TASK-260830-1qf777-2 revision 2 candidate patch (repository_delta=present, 20 changed paths)
- [TASK-260830-1qf777_change-request_rev2-validation.log](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev2-validation.log) — Change Request CR-TASK-260830-1qf777-2 revision 2 bounded validation log
- [TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-de6b47.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-de6b47.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_review-verdict-rev2.md](file://TASK-260830-1qf777/TASK-260830-1qf777_review-verdict-rev2.md) — Reviewer verdict for CR rev2 (RUN-260901-de6b47): changes requested on three evidence gaps; rev1 findings all verified closed
- [TASK-260830-1qf777_reviewer-mutant-evidence-rev2.log](file://TASK-260830-1qf777/TASK-260830-1qf777_reviewer-mutant-evidence-rev2.log) — Independent reviewer sweep: 40 mutants, 38 died, 2 survivors proven subsumed; ratchet/traceability gate attacks; F1 probe; gate transcript
- [TASK-260830-1qf777_reviewer-mutants-rev2.py](file://TASK-260830-1qf777/TASK-260830-1qf777_reviewer-mutants-rev2.py) — Reviewer mutation harness batch 1 (26 mutants) used for the rev2 verdict
- [TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-d41916.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-d41916.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-12b8e2.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-12b8e2.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_rework-r3-evidence.md](file://TASK-260830-1qf777/TASK-260830-1qf777_rework-r3-evidence.md) — Round 3: F1 symlink stance declared and pinned both directions, F2 MigrateOS end-to-end coverage, F3 MigrationError no-leak; 29/29 mutants dead; gate exit codes including clean-environment run
- [TASK-260830-1qf777_mutant-sweep-r3.log](file://TASK-260830-1qf777/TASK-260830-1qf777_mutant-sweep-r3.log) — Round 3 full mutation transcript: 29 mutants, all RED, restored tree green, zero survivors
- [TASK-260830-1qf777_clean-env-r3.log](file://TASK-260830-1qf777/TASK-260830-1qf777_clean-env-r3.log) — Round 3 determinism gate: full suite under env -i with an empty HOME, offline module proxy, task-scoped GOCACHE
- [TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-877162.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-877162.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_rework-r4-evidence.md](file://TASK-260830-1qf777/TASK-260830-1qf777_rework-r4-evidence.md) — Round 4 recovery: story_final base-authority root cause, provenance repair with tree-hash proof, round-3 re-verification, gate exit codes, post-handoff gate/candidate-tree verification
- [TASK-260830-1qf777_mutant-sweep-r4.log](file://TASK-260830-1qf777/TASK-260830-1qf777_mutant-sweep-r4.log) — Round 4 independent re-run of the 29-mutant sweep: 29 RED, 0 survivors, restored tree green
- [TASK-260830-1qf777_change-request_rev3.patch](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev3.patch) — Change Request CR-TASK-260830-1qf777-3 revision 3 candidate patch (repository_delta=present, 21 changed paths)
- [TASK-260830-1qf777_change-request_rev3-validation.log](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev3-validation.log) — Change Request CR-TASK-260830-1qf777-3 revision 3 bounded validation log
- [TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-8970a0.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-8970a0.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_review-verdict-rev3.md](file://TASK-260830-1qf777/TASK-260830-1qf777_review-verdict-rev3.md) — Reviewer verdict for CR revision 3: changes requested, three surviving mutants on the task's own acceptance criteria
- [TASK-260830-1qf777_reviewer-mutation-sweep-rev3.log](file://TASK-260830-1qf777/TASK-260830-1qf777_reviewer-mutation-sweep-rev3.log) — Independent reviewer mutation sweep for CR rev3: 20 mutants, 8 survivors (F1/F2/F3 confirmed)
- [TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-e6ac21.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-e6ac21.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_rework-r5-evidence.md](file://TASK-260830-1qf777/TASK-260830-1qf777_rework-r5-evidence.md) — Round 5: CR rev3 F1/F2/F3 closed plus all six non-blocking observations; derived member-preservation property with falsifiable exemptions; 73-mutant sweep with one declared-subsumed survivor
- [TASK-260830-1qf777_mutant-sweep-r5.log](file://TASK-260830-1qf777/TASK-260830-1qf777_mutant-sweep-r5.log) — Round 5 sweep transcript, 4 bounded batches: 73 mutants, 71 RED, 1 declared-subsumed (R13, subsuming check named), 1 positive control GREEN, 0 unexplained survivors
- [TASK-260830-1qf777_mutants-r5.py](file://TASK-260830-1qf777/TASK-260830-1qf777_mutants-r5.py) — Round 5 mutation harness: 29 producer + 18 reviewer rev3 mutants re-run against the whole package, plus 26 new; declares EXPECT_GREEN control and EXPECT_SUBSUMED with named subsuming checks
- [TASK-260830-1qf777_change-request_rev4.patch](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev4.patch) — Change Request CR-TASK-260830-1qf777-4 revision 4 candidate patch (repository_delta=present, 22 changed paths)
- [TASK-260830-1qf777_change-request_rev4-validation.log](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev4-validation.log) — Change Request CR-TASK-260830-1qf777-4 revision 4 bounded validation log
- [TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-54aa4b.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-54aa4b.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_review-verdict-rev4.md](file://TASK-260830-1qf777/TASK-260830-1qf777_review-verdict-rev4.md) — Reviewer verdict for CR revision 4: changes requested, one blocking finding (B1, self-minted home-capture evidence)
- [TASK-260830-1qf777_reviewer-mutation-sweep-rev4.log](file://TASK-260830-1qf777/TASK-260830-1qf777_reviewer-mutation-sweep-rev4.log) — Independent 44-mutant reviewer sweep for CR revision 4 plus clean-environment determinism re-check
- [TASK-260830-1qf777_reviewer-home-probe-rev4_test.go](file://TASK-260830-1qf777/TASK-260830-1qf777_reviewer-home-probe-rev4_test.go) — Verified test closing finding B1: kills both surviving OSInputs home-wiring mutants at the real LoadOS entry; passes under env -i with empty HOME
- [TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-22ea2b.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-implementer--developer--claude-_RUN-260901-22ea2b.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_home-mutation-sweep-rev5.log](file://TASK-260830-1qf777/TASK-260830-1qf777_home-mutation-sweep-rev5.log) — Round 5 sweep transcript: 16 mutants, 16 RED, 0 survivors, 0 invalid; whole-package gate, positive control and restored-tree verification
- [TASK-260830-1qf777_home-mutants-rev5.py](file://TASK-260830-1qf777/TASK-260830-1qf777_home-mutants-rev5.py) — Round 5 mutation harness for the OSInputs home capture: capture, per-site cause drop, gate deletion, and layout drift mutants
- [TASK-260830-1qf777_rework-r6-evidence.md](file://TASK-260830-1qf777/TASK-260830-1qf777_rework-r6-evidence.md) — Round 6 (rev5 candidate) closing CR rev4 finding B1: home capture proven at the real LoadOS/MigrateOS entries across all three home-derived lanes; 16/16 mutants RED including both rev4 survivors; gate exit codes
- [TASK-260830-1qf777_change-request_rev5.patch](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev5.patch) — Change Request CR-TASK-260830-1qf777-5 revision 5 candidate patch (repository_delta=present, 23 changed paths)
- [TASK-260830-1qf777_change-request_rev5-validation.log](file://TASK-260830-1qf777/TASK-260830-1qf777_change-request_rev5-validation.log) — Change Request CR-TASK-260830-1qf777-5 revision 5 bounded validation log
- [TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-e9845b.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-e9845b.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-fba8aa.log](file://TASK-260830-1qf777/TASK-260830-1qf777_spawn-log_-reviewer--reviewer--claude-_RUN-260901-fba8aa.log) — System spawn log captured by task-board
- [TASK-260830-1qf777_reviewer-mutation-sweep-rev5.log](file://TASK-260830-1qf777/TASK-260830-1qf777_reviewer-mutation-sweep-rev5.log) — Independent reviewer mutation sweep for CR rev5: 36 mutants, 31 RED, 4 declared survivors, 1 positive control
- [TASK-260830-1qf777_reviewer-mutants-rev5.py](file://TASK-260830-1qf777/TASK-260830-1qf777_reviewer-mutants-rev5.py) — Reviewer mutation harness for CR rev5: applies one edit at a time and runs the whole internal/config package with -count=1
- [TASK-260830-1qf777_review-verdict-rev5.md](file://TASK-260830-1qf777/TASK-260830-1qf777_review-verdict-rev5.md) — Reviewer verdict for CR rev5 (RUN-260901-fba8aa): ACCEPTED, with the 36-mutant independent sweep and gate results

## Created
2026-08-29T21:59:56Z

## Last Update
2026-09-02T00:14:28Z

## Assigned To
[reviewer] reviewer (claude)
