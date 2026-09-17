# CR7 review verdict: changes requested

TASK-260909-2ez769 / CR-TASK-260909-2ez769-7 revision 7.
Reviewer RUN-260916-17a623. Base a89328ca85a613abde39660f4be749a8fcf57903;
immutable candidate tree fca43ac48ee7fea12e7a8429322b2d9ce1a58cf8.
Read-only product review; all probes ran in an extracted candidate under task scratch.
No live product/index/branch/HEAD changes, commits, checkpoints, integration or commit_ack.

## F1 — P2: the write-path census still admits executable unlocked writers

Repeat-of: CR6 F3 (instrument completeness), within the CR3–CR6 config/trust
transaction bypass family. This is a failed prevention gate, not a claim that
the planted functions already ship in the candidate.

`internal/config/writepath_census_test.go:434` resolves the spelling `os`, not
the imported package identity. Function-value escape detection recognizes only
bare helper identifiers, excluding selector members. Package initializers are
walked recursively only when the initializer itself is a FuncLit; an immediately
invoked factory's returned closure is missed. These are ordinary Go constructs,
not reflection, unsafe, external disk sabotage or invalid/uncompilable plants.

Independently executed **1 of 5 detected**. Every plant is a new non-test source
file in the immutable candidate copy. Each invocation runs both
TestWritePathCensusGate and TestReviewerRoguePlantExecutes with -count=1.
The witness calls the planted production-position function, replacing real file
bytes without acquiring an authorization hold. Every witness PASSED.

| Plant | Census | Process exit |
|---|---|---:|
| Direct os.WriteFile (positive detection control) | rejected | 1 |
| os.WriteFile stored in a package map and called through map lookup | MISSED | 0 |
| `import fs "os"`; fs.WriteFile | MISSED | 0 |
| Package initializer invokes factory returning closure using os.WriteFile | MISSED | 0 |
| osMigrationFileSystem{}.Rename stored as a method value, called indirectly | MISSED | 0 |

Sources and raw logs are in census-plants/; probe-census.py reproduces the five
runs. The direct-control exit 1 is a real census assertion failure, not a build
failure; the four green executions prove the escapes. In particular the backend
method-value plant bypasses the token by invoking the ordinary production backend.

Required rework: enforce the claimed domain using package/symbol-aware mutation
and function-value handling, or explicit fail-closed restrictions for unresolved
flows. Walk all executable initializer descendants. Apply the rule to backend
method references as well as calls; do not treat unreadable/unresolved forms as
safe. Keep these executable controls, the original six controls and the
preserve-token behavioral mutant. A list of these specific new names is not a
class-level repair. Until then the census's statement that out-of-protocol
writers cannot exist is false, and its exemptions are not a completeness proof.

## F2 — P2: a live capability from a different store authorizes the target writer

Repeat-of: CR6 F3's prevention instrument; newly introduced capability weakness.
`internal/hosttrust/hold.go:26-33` stores only an atomic released bit. Neither
hosttrust.requireHold nor config.requireHold (`migration.go:303`) relates that
bit to a lock, store root or mutation target. The hold proves that SOME lock is
held, not the authorization lock protecting the pair being replaced.

TestReviewerForeignHoldCannotWriteLockedPair, run with -race, fails (exit 1):

1. Open a target fixture and an unrelated store at a distinct state root.
2. A separate goroutine holds the target store's exclusive authorization lock,
   waiting on a channel. It never lends its hold to the writer.
3. A planted production-position reviewerForeignWrite obtains the unrelated
   store's genuine token through WithExclusiveHold and passes it to the existing
   writeTempReplace for the target config.
4. The replacement succeeds while the target holder is still blocked; a disk
   read verifies the replacement bytes. TestWritePathCensusGate PASSES on this
   same plant because it is syntactically inside WithExclusiveHold.

See foreign-hold-production-final.log and probes/config/. No zero token,
reflection, unsafe or unlock race is involved. The corresponding same-store
serialization, zero-token and released-token tests pass; they do not cover this
wrong-resource case. Both advertised enforcement layers admit the new writer.

Required rework: bind the capability and each protected mutation to the same
coordination resource, including the config/trust pairing, and reject a live
capability from an unrelated store. Add this production-position negative and a
narrowing plant that admits a wrong-resource token while retaining zero/released
refusals. Keep same-resource handles usable according to the actual lock model.
The current shipped callers were not found to select a foreign token; severity
is P2 for the broken mandatory guard/instrument, not a newly demonstrated P1
in the existing ApplyV4/RollbackV4 call chain.

## CR6 F1 and F2 are repaired in the exercised production paths

- Legacy migrate now opens the coordinating store and performs source
  revalidation plus replacement under WithExclusiveHold. The exact original
  synchronous TestReviewerLegacyCannotOverwriteCommittedV4 now times out at
  5 seconds (exit 1), blocked in exclusiveHold. This is serialization evidence,
  NOT a passing test. TestLegacyMigrateCannotOverwriteCommittedV4 passes and
  checks final Config4/generation+1 after refusing the delayed legacy source.
  Source inspection confirms both version and exact-byte revalidation. Opening
  an absent store creates its owner-only skeleton, as documented; it does not
  silently create trust.json.
- Both exact TestReviewerFailedReplacementCannotLoseIntent and
  TestReviewerFailedRollbackCannotLoseIntent pass. Failed replace now enters
  resolveFailedReplace under the continuous hold. Source-equal state aborts;
  durable replacement restores under compensation or retains the marker.
  The committed marker-kept/reopen tests pass in the package suite.
- The new TestReviewerFailedReplaceRestoreSerializesReader exercises apply AND
  rollback: replacement rename succeeds, its sync fails, the first restore fails,
  and the compensation restore pauses before succeeding. A surviving coherent
  reader attempts convergence during that pause and remains blocked; after
  release it sees the exact source bytes and unchanged generation. Reopen sees
  the same clean abort and no marker. Both subtests pass with -race. A reader
  cannot converge 'in between' through this actual locked path.
- All seven original probes and all six CR5 compensation races pass with -race.
  Child crash probes retain real process termination and cross-process locking.

The exempt credential path uses fresh digest directories, fixed filenames and
use-time custody checks; lock creation uses O_EXCL and never replaces a live
inode; config writeTempFile stages through CreateTemp without publishing;
secureStaged is ACL setup at the inspected call sites. These current callers
are consistent with the stated exemptions. Their prose does not establish that
new callers cannot reach committed state: F1/F2 defeat that prevention claim.
No native Windows runtime, physical power-loss, post-lock-release dispatch or
secret-exclusion consumer guarantee is added here.

## AC accounting

**19 of 21 functional AC rows driven**, with the existing flow tests passing
within the bounds below. The 21 rows were checked against pinned sections 6.6,
11.10.2 and 11.10.3 and their named candidate tests. This is the established
functional decomposition, not exhaustive normative-clause or all-gates mutation
coverage. The newly claimed class-level writer instrument fails F1/F2, so these
passing rows do not justify acceptance. Rows 15 and 21 retain CR5/CR6 bounds.

| # | Row | Production call / named test | Result |
|---|---|---|---|
| 1 | Closed Config4 | config.Decode / TestDecodeConfiguration4Refusals | pass |
| 2 | Historical compat + explicit migration | config.Migrate/migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade, TestLegacyMigrateCannotOverwriteCommittedV4, TestLegacyMigrateConvergesUnreplacedMarkerThenCommits, TestLegacyMigrateRefusesAfterReplacementLanded | pass (was CR6 FAIL) |
| 3 | Closed trust document | hosttrust.DecodeTrust / TestDecodeTrustRefusals | pass |
| 4 | Missing/corrupt/unreadable trust | hosttrust ReadSnapshot / TestReadSnapshotMissingStore, TestCorruptTrustRefusedNeverEmpty, TestReadSnapshotRefusesUnreadableMarker | pass |
| 5 | Fresh issuance | hosttrust IssueCredential / TestIssueCredentialProfile | pass |
| 6 | Exact profile + time bounds | hosttrust VerifyProfile / TestVerifyProfileRefusals, TestVerifyProfileRefusesJustPastExpiry | pass |
| 7 | Explicit tuple enrollment | hosttrust Store.Enroll / TestEnrollRefusals | pass; OOB human verification remains operator act |
| 8 | Mapping uniqueness | hosttrust DecodeTrust+admitEntry / duplicate-root/key/digest rows | pass |
| 9 | Rotation/retirement bounds | hosttrust Rotate, MarkRetiring / TestRotateBoundedWindow, TestMarkRetiringCapsAtLeafExpiry | pass |
| 10 | Revocation tombstones | hosttrust Revoke, Enroll / TestRevokeTombstone, TestRevokedLeafNeverReenrolls | pass |
| 11 | Old-generation mutation refusal | hosttrust WithMutationAuthorization / TestWithMutationAuthorizationRefusesStaleGeneration, TestMutationAuthorizationRefusesStaleAfterConvergence | pass |
| 12 | Shared authorization serialization | hosttrust Open/openLock, WithMutationAuthorization+Revoke / TestFirstOpenPreservesHeldLock, TestConcurrentOpenAttachesToExistingLock, TestMutationAuthorizationSerializesWithRevocation, TestMutationAuthorizationSerializesWithSeparateStoreRevocation | pass |
| 13 | Coherent config+trust snapshot | config LoadCoherent / surviving-reader crash tests, six compensation races, F1/F2 ports, marker-kept reopen tests | pass (was CR6 FAIL) |
| 14 | Unix owner custody | hosttrust Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | hosttrust verifyOwner/installOwnerOnlyACL/secureStaged / custody_windows_test.go (7 native tests) | NOT DRIVEN: code + tests present, runtime unverified (no runner) |
| 16 | Complete peers + selected credential | config PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | config ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | config applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate* tests | pass, incl. in-lock revalidation |
| 19 | Crash-durable config/trust pair | config applyV4/rollbackV4 + hosttrust JointCommit/resolveFailedReplace/compensateLocked/Recover/converge / TestApplyV4CrashConvergesGeneration (fixed), surviving-reader crash tests, barrier matrix, 6 compensation race tests, F2 ports + marker-kept tests | pass (was CR6 FAIL) |
| 20 | Explicit backup rollback | config RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded, TestApplyV4BackupIsClassifiedConfigCopy, TestRollbackV4FailedReplaceKeepsIntent, TestRollbackV4FailedReplaceKeepsMarkerWhenRestoreFails | pass (was CR6 FAIL) |
| 21 | Secret exclusion across export surfaces | hosttrust MatchExcludedFromReplication/ExcludedConfigDirName / matcher tests + persisted handoff + owner notes; diagnostics unowned | NOT DRIVEN: matcher enforced, no consumer calls it yet |


Row 15: NOT DRIVEN. Windows equivalent ACL code and seven native tests exist;
cross-build/vet is not runtime DACL evidence. Windows backup ACL inheritance
remains a bound. Row 21: NOT DRIVEN. Matcher-only evidence and persisted consumer
handoff; no consumer enforcement call yet, diagnostics still unowned. RPC
TLS/hello/dispatch/watchdog/stream integration stays with the RPC owner. No
mesh-wide revocation or externally authenticated operator action is inferred.

## Validation, provenance and limitations

- Independently ran extracted-candidate config/hosttrust/peeridentity tests
  -count=1 -cover: exit 0; 93.5% / 78.0% / 97.5%, Go 1.25.5 darwin/arm64.
  Independently ran the nine prior probes (seven original plus two CR6 partial
  replacements), six compensation races, legacy async port and new two-case
  compensation-reader interleaving with -race: exit 0. Census/foreign-hold
  attacks and legacy synchronous timeout have the real exits recorded above.
- Verified all 61 changed candidate paths byte-for-byte against the immutable
  tree; all also match live worktree bytes. Base signature verifies (exit 0).
  RPC backup 15/15 and parked 11/11 hashes match; zero overlap with the 61 paths.
  SSH transport unchanged; peeridentity delta only v4_interop_test.go. The patch
  digest is 234f5ad8c5473c35ed500259a23de37373df12fe57375730b2fea556cf6dde0f.
  Normative SPEC.v0.6.0.md matches the pin for
  0cbdf100dbf84df50c64f792b1f940e3a67859a6.
- Audited, not reran, both producer mutation batteries. All 34 hosttrust and
  39 config source hashes match the candidate. All 43 plants match exactly once
  and archived mutant source equals the expected overlay. Raw logs contain the
  expected named failures, nonempty execution and no compilation-failure kills.
  Hosttrust: 32 narrowing + 1 separately classified call-site kill, neutral exit
  0. Config: 9 semantic + 1 separately classified label-precision kill, neutral
  exit 0. Both neutrals use the same harness. The token-preserving compensation
  reorder is killed by behavior, not merely the source checker. These batteries
  do not cover the new F1/F2 counterexamples.
- The handoff validation artifact reports 26/26 command shards green, but is
  explicitly truncated (5,128,824 bytes omitted). Its visible build/vet commands
  exit 0; the aggregate is not a raw complete replay of every required job.
  The omitted commands' full outputs were not independently verified, and this
  reviewer did not rerun the full repository/fuzz/build matrix. The producer
  outcome claims those passed. Do not convert those claims or truncated logs
  into complete local-check certification. Handoff board validation reports
  366 issues with exit 0 (producer outcome said 201); neither proves clean board.
- The first attempt to set up the production foreign-token plant used the wrong
  scratch-relative path. Its log is marked invalid-setup and is not evidence
  for that plant. The earlier direct-helper foreign-token experiment held both
  locks in one goroutine and is superseded. The decisive final probe places the
  target holder in a separate goroutine, installs the new production-position
  writer and runs the gate and behavioral test together under -race.
- Canonical CLI provenance and readiness recorded; the PATH wrapper execs
  /Users/iv/.curator/global/bin/task-board. Required project-management,
  architecture-diagrams and Curator go-testing-tools instructions read. No
  diagram or installation needed. Missing worktree-local skill paths were
  resolved to the main checkout's Curator-managed go-testing-tools skill.
  Directive reads succeeded with no directives.

## Verdict and route

**Changes requested → to-dev. Do not accept CR7.** Two P2 defects in the mandatory
class-level enforcement instrument remain. No new P1 is asserted in the existing
migration paths. Repair the instrument with the executable controls above,
preserve the passing F1/F2 repairs and all foreign RPC/SSH/peer bytes, then publish
a new managed candidate. Ordinary implementable rework; no external blocker or
human decision. Review evidence and logbook are attached before routing.
