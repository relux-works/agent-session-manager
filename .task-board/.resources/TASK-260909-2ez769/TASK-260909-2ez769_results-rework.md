# TASK-260909-2ez769 rework results \u2014 CR3 findings repaired

Producer rework on managed base a89328c (v0.6.0 adoption trunk in place; no base move needed).
Worktree left UNCOMMITTED on task-board/story/STORY-260830-1kiyj6 for handoff snapshot; no producer commit, no main push.
Producer: Muse (muse-spark), 2026-09-10. Prior RED: all three reviewer counterexamples reproduced FAIL on pre-fix source, then repaired.

## Preservation (re-verified this run)

- 4 tracked RPC paths CLEAN vs HEAD (README.md, internal/traceability/ownership.v0.5.0.json, internal/traceability/traceability.go, task-board.config.json); no UNRESOLVED_QUESTIONS.md or internal/rpcwire remnants; parked delta 11/11 files intact at .temp/TASK-260909-2ez769/parked-rpc-delta-260910/. Nothing from the RPC delta consumed; none of those paths in the candidate diff.
- Checkpointed SSH/peer work untouched (candidate diff touches only LOGBOOK.md, 10 prior-run config/secprim files, and new hosttrust/Config4/peer-interop files).

## CR3 finding resolutions (production mechanism, not assertions)

1. P1 stale mutation \u2014 `WithMutationAuthorization` (`internal/hosttrust/authorize.go`) no longer overwrites the binding with current generation; `request.SnapshotGeneration != current.Generation` refuses `ErrStaleGeneration`. Test: `TestWithMutationAuthorizationRefusesStaleGeneration` (enroll-third-host generation move; stale refused, fresh rebind admitted).
2. P1 boundary/revoke serialization \u2014 locks are per-acquisition descriptors/handles (unix `flock`, Windows `LockFileEx`), so same-process goroutines serialize exactly like separate processes; lock-file creation under a process mutex; `Initialize` reads without re-acquiring (nested-lock discipline documented on every primitive). Test `TestMutationAuthorizationSerializesWithRevocation` hardened to the deterministic 250ms contention shape (the old nonblocking select is gone).
3. P1 crash atomicity \u2014 new `internal/hosttrust/joint.go`: closed `PendingCommit` marker (duplicate-key audit, unknown-field refusal, bare-number uint53), `JointCommit` (exclusive hold across revalidation, marker stage, config replace, generation bump, marker removal), `Recover` (replacement-present completes forward incl. already-bumped idempotence; source-present aborts; anything else refuses keeping the marker), auto-run by `Open` (refused recovery fails the open). `CommitConfigChange` removed. Apply/rollback revalidate preview/source/generation/custody inside the hold (`revalidateApplyV4`, `revalidateRollbackV4`); post-bump failures restore source then `Recover` (`ErrJointReplacementDurable`, helper `restoreSourceAfterJointFailure`). Tests: real child-process crash `TestApplyV4CrashConvergesGeneration` (exit 77 at the rename seam; restart converges to 4.0.0/gen+1), plus forward/abort/intervention/already-bumped/exhaustion/marker-closed-decode unit tests. Power-loss vs process-crash stated honestly: process-crash atomic via stage+fsync+rename+dirsync; power loss additionally depends on disk honoring fsync.
4. P1 Windows DACL \u2014 `verifyOwner` enforces owner SID equality plus a full DACL audit (every allow entry must name the owner, incl. inherited; null/empty/deny-only/unknown-type fail closed; reparse points refuse); creation paths install protected owner-only DACLs via `secureStaged`; Unix keeps exact 0700/0600 through `platformModeOK`. Pure `ownerOnlyGrants` evaluator tested on darwin (9 cases). BOUND (no invented pass): no Windows runner exists here; evidence is GOOS=windows build + vet clean plus unit tests. Config-dir backups (*.bak.*, .pre-rollback.*) are 0600-staged and covered by exclusion; explicit owner-only DACLs there remain a stated bound (user-profile ACLs govern).
5. P2 exclusion \u2014 `ExcludedFromReplication` gains pending-commit.json; new `MatchExcludedFromReplication` (staging prefixes, fail-closed escapes) and `ExcludedConfigDirName`; tests walk the whole custody layout (every file excluded), pin published backup names, and keep explicit enrollment export exchangeable. No in-repo replication/snapshot/export-of-state production surface exists (sessadapter/localstore/dirnode verified not to touch host-channel): HANDOFF to future export surfaces \u2014 every replication, snapshot, clone, Session Directory or diagnostic-bundle writer MUST consult `MatchExcludedFromReplication`/`ExcludedConfigDirName` and drop matches, with a negative test per surface that fails when a custody path is admitted. Mutation claim corrected: config kills are 4 semantic narrowing + 1 label-precision (plant text marked LABEL-PRECISION in-table).

## AC coverage: 21 of 21 rows driven through production entry points

| # | Row | Production call / named test | Result |
|---|---|---|---|
| 1 | Closed Config4 | config.Decode / TestDecodeConfiguration4Refusals | pass |
| 2 | Historical compat + explicit migration | config.Migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade | pass |
| 3 | Closed trust document | hosttrust.DecodeTrust / TestDecodeTrustRefusals | pass |
| 4 | Missing/corrupt trust refuses | hosttrust ReadSnapshot/Open / TestReadSnapshotMissingStore, TestCorruptTrustRefusedNeverEmpty | pass |
| 5 | Fresh issuance | hosttrust Issue/IssueCredential / TestIssueCredentialProfile | pass |
| 6 | Exact profile + time bounds | hosttrust VerifyProfile / TestVerifyProfileRefusals, TestVerifyProfileRefusesJustPastExpiry | pass |
| 7 | Explicit tuple enrollment | hosttrust Store.Enroll / TestEnrollRefusals | pass; OOB human verification remains operator act |
| 8 | Mapping uniqueness | hosttrust DecodeTrust+admitEntry / duplicate-root/key/digest rows | pass |
| 9 | Rotation/retirement bounds | hosttrust Rotate, MarkRetiring / TestRotateBoundedWindow, TestMarkRetiringCapsAtLeafExpiry | pass |
| 10 | Revocation tombstones | hosttrust Revoke, Enroll / TestRevokeTombstone, TestRevokedLeafNeverReenrolls | pass |
| 11 | Old-generation mutation refusal | hosttrust WithMutationAuthorization / TestWithMutationAuthorizationRefusesStaleGeneration | pass (was CR3 FAIL) |
| 12 | Boundary/revoke serialization | hosttrust WithMutationAuthorization + Revoke / TestMutationAuthorizationSerializesWithRevocation (deterministic) | pass (was CR3 FAIL) |
| 13 | Coherent config+trust snapshot | hosttrust WithSharedSnapshot + config LoadCoherent / TestLoadCoherentPairsConfigWithTrust, TestLoadCoherentSerializesWithJointCommit | pass (new production reader) |
| 14 | Unix owner custody | hosttrust Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | hosttrust verifyOwner/installOwnerOnlyACL/secureStaged / TestOwnerOnlyGrants (pure evaluator) + windows cross-build/vet | implemented; runtime Windows execution is a stated bound (no runner) |
| 16 | Complete peers + selected credential | config PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | config ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | config applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate* tests | pass, incl. in-lock revalidation |
| 19 | Crash-durable config/trust pair | config applyV4 + hosttrust JointCommit/Recover/Open / TestApplyV4CrashConvergesGeneration (+ unit matrix) | pass (was CR3 FAIL) |
| 20 | Explicit backup rollback | config RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded | pass; stopping channels stays RPC/operator boundary |
| 21 | Secret exclusion across export surfaces | hosttrust MatchExcludedFromReplication/ExcludedConfigDirName / walk + name + matcher tests | enforced at custody surface; downstream surfaces get explicit handoff above |

Stated bounds (not production-driven here): live TLS 1.3 launch/handshake/hello/dispatch and one-second stream close-down belong to the RPC transport task; peer-side distribution of enrollment/rotation/revocation is operator duty per spec; Windows runtime execution (no runner); power-loss durability beyond fsync; config-dir backup DACLs on Windows (profile ACLs + exclusion).

## Negative + mutation evidence (rerun this run, exact-source)

- hosttrust battery `.temp/TASK-260909-2ez769/mutations-hosttrust-rework/`: 18 narrowing mutants KILLED by named tests (incl. new stale-mutation-refresh, joint-marker-stale, recover-diverged; custody-0644 re-aimed at platformModeOK; preview sites unaffected) + 1 neutral SURVIVED control. results.json + table.md + manifest.json (29 source digests, head a89328c).
- config battery `.temp/TASK-260909-2ez769/mutations-config-rework/`: 4 semantic narrowing KILLED (transport-ssh, preview-length now on the in-lock revalidation site, confirm-drops, apply-gen-direction) + 1 label-precision KILLED (v4-explicit-label, marked as such) + 1 neutral passed. results.json + table.md + manifest.json (8 source digests, head a89328c).
- No gate searches source text for a token (all gates operate on decoded values/DER/parsed TOML/file bytes/ACEs); the preserve-token behavioral-mutant clause is vacuous here, as previously stated.

## Gates rerun this run (all exit 0)

gofmt clean; `go vet ./...`; `go test ./... -count=1` all ok (281 top-level PASS across hosttrust/config/peeridentity/secprim); `-cover` hosttrust 77.8% config 93.1% peeridentity 97.5% secprim 94.4%; `-race` hosttrust+config ok; config refusal-inventory audit green; secprim census green; tracecheck ok; catalog v0.6.0 -check ok; GOOS=linux/windows amd64 builds ok; `git diff --check` clean. `task-board validate` reports 231 pre-existing board-wide ledger-mirror issues, none referencing this task. Fuzz seeds untouched (no changed-candidate reason; prior evidence stands).
