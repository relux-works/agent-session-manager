# CR11 review verdict: changes requested

Task TASK-260909-2ez769, CR-TASK-260909-2ez769-11 revision 11.
Reviewer RUN-260917-6165a9. Base 305875134f8d344eb86ef926c7fbca3fed85c49d;
candidate tree 11a860063bb9802361f7d1883dff8ad64681ba94.
Patch SHA-256 75bc1326120283632cf2afd0129598e84bc3be69d9d59e0e11dff310494ed9a0.
All probes ran in an archive of this immutable tree under
`.temp/TASK-260909-2ez769/review11/candidate`. No live Story source, index,
branch, HEAD, checkpoint or integration was changed.

## F1 — P2: interface exceptions still authorize new call sites

Repeat-of CR10 F1 / CR9 F1 / CR8 F1 / CR7 F1 / CR6 F3, narrowed to the
explicit call-site requirement that CR11 still does not implement.
The original Execute/name heuristic is repaired for standalone interface and
type-parameter call boundaries. All 17 independent standalone plants are now
killed by the census assertion, with all 17 actual-file runtime witnesses passing.
This includes both CR10 direct Execute boundaries and two new shapes: a generic
method value named Perform and a closure-returned function called through an
interface. No compilation failure is counted as a kill.

However `writepath_census_types_test.go:490` builds a set of method objects per
enclosing function, and `:681` authorizes every call of that object anywhere
inside that function. The policy does not identify a call site or enforce its
position relative to the hold gate. `:775` similarly admits its method values.
The source still contains a method-spelling predicate at `:783/:991`; that is
not the counterexample here, but the report's blanket no-heuristic claim should
be narrowed or the remaining admission path removed.

Independent compiling plant: insert a conditional `filesystem.Rename` at the
beginning of production `replaceDurably`, before `requireHoldForConfig`, using
its existing filesystem interface. The planted branch is selected only by the
reviewer witness. A zero hold and zero pair successfully rename a real file.
`TestReviewerUnlistedSiteExecutes` PASS and `TestWritePathCensusGate` PASS;
subprocess exit **0**. The original hold check and existing method-object tokens
remain intact. This is a prevention-instrument counterexample, not a claim that
the unmodified product currently contains that branch. See `callsite_probe.py`,
`reviewer_callsite_test.go`, `callsite-plant.txt`, and `callsite.log`.

Required repair: enumerate and check the exact required indirect edges AND
call sites, with a production justification for each. A new call or escaped
method value cannot inherit authorization solely from being inside a writer
that already uses that interface method. Add this executable regression and a
narrowing mutant that preserves method identity while broadening call-site
admission. Do not merely add this one sentinel to the detector.

## F2 — P2: a refused rebind durably claims a second configuration

New failure introduced by the CR11 bidirectional binding repair; related to,
but not a survival of, CR10 F2's foreign-store write.
`hosttrust.ensureConfigBindingLocked` publishes the authoritative target record
at `binding.go:254` before checking the existing store-side association at
`:263–269`. A legitimate sequence using only production APIs is:

1. Resolve (config A, state S) and (config B, state S).
2. Open both Stores while S is unbound.
3. Store A successfully calls EnsureConfigBinding(A,S).
4. Already-open Store B calls EnsureConfigBinding(B,S).

Step 4 returns ErrExclusiveHoldResourceMismatch, but only after it has durably
written B's target sidecar naming S. The store-side index still names A. A later
legitimate attempt to associate B with a different state root is now refused.
The immutable one-store/one-config association is left contradictory by a
refused request, with no crash or sidecar forgery involved.

`TestReviewerRefusedRebindLeavesTargetUnclaimed` fails, actual exit **1**, after
observing the persistent B sidecar and the subsequent legitimate-bind refusal.
See `reviewer_binding_test.go`, `rebind-final.log`, and `rebind-final.exit`.
Required repair: validate the existing store association under its exclusive
lock before publishing any target association. Cover two handles opened before
binding, sequential and competing binding requests, and failed requests leaving
both existing and previously unclaimed target associations intact. Retain the
explicit crash-repair path for a genuinely interrupted first binding.

The old F2 counterexample IS repaired: a second genuine resolver-issued pair
(target A, foreign state F) cannot bind A once A's authoritative target record
names S, even while S's lock is held. `TestReviewerSecondResolvedPairRefused`
and committed HeldExclusive controls pass, exit 0. ResolvedPaths has private
fields; its getters return copies, and no constructor other than ResolvePaths
mints its populated map. Public pair APIs now consume this value, or production
Inputs/Overrides resolved through config.Load; pair writers consume the value.
Root-string helpers remain for trust-file plumbing behind requireHoldForRoot;
this is not falsely reported as 'no string path arguments anywhere'.

## Evidence and repository hygiene

The named board resource `TASK-260909-2ez769_producer-evidence-rev11.tar.gz` is
ASCII prose, not an archive. `TASK-260909-2ez769_results-rev11.md` is absent as a
board resource (get returned resource-not-found). The local archive DOES exist,
and SHA-256 b33b18fa31ce0df7db66f937b9c7f83cf0980f92d28f0b0e8bbd7c82b142d1ce
matches the pointer; its outcome and raw logs were available for this review.
Correct the attachment with resource update using the actual binary and attach
the outcome independently before the next handoff. Local survival must not be
mistaken for a portable evidence attachment.

Remove the candidate's generated `internal/hosttrust/__pycache__/mutations.cpython-314.pyc`.
x/tools is test-only; offline tidy and all three platform builds pass. README's
tool row still describes writer-shaped calls, not the new all-indirect-call
policy and its exact exceptions. Update it after F1 is fixed.

## Functional AC accounting and limits

**19 of 21 functional AC rows driven**, with rows 15 and 21 retained as stated
bounds. This is the established functional decomposition, not exhaustive
normative-clause coverage or discharge of F1/F2. Rechecked pinned v0.6.0 sections
6.6, 11.10.2 and 11.10.3 against the local hash-verified source. All 43 exact test
names in the table below resolve in the immutable candidate; generic suite
labels additionally refer to the package runs. The new binding failure is not
hidden by this functional ratio.

| # | Acceptance row | Production call / named test evidence | Result |
|---:|---|---|---|
| 1 | Closed Config4 | `config.Decode`; `TestDecodeConfiguration4Refusals` | pass |
| 2 | Historical compatibility + explicit migration | `config.Migrate`/legacy migration; `TestMigrateRefusesV4Target`, `TestMigrateRefusesV4Downgrade`, `TestLegacyMigrateCannotOverwriteCommittedV4`, `TestLegacyMigrateConvergesUnreplacedMarkerThenCommits`, `TestLegacyMigrateRefusesAfterReplacementLanded` | pass |
| 3 | Closed trust document | `hosttrust.DecodeTrust`; `TestDecodeTrustRefusals` | pass |
| 4 | Missing/corrupt/unreadable trust | `hosttrust.ReadSnapshot`; `TestReadSnapshotMissingStore`, `TestCorruptTrustRefusedNeverEmpty`, `TestReadSnapshotRefusesUnreadableMarker` | pass |
| 5 | Fresh issuance | `hosttrust.IssueCredential`; `TestIssueCredentialProfile`, `TestIssueSelfEnrollsActive` | pass |
| 6 | Exact profile + time bounds | `hosttrust.VerifyProfile`; `TestVerifyProfileRefusals`, `TestVerifyProfileRefusesJustPastExpiry`, wrong-key-usage/EKU/SAN tests | pass |
| 7 | Explicit tuple enrollment | `hosttrust.Store.Enroll`; `TestEnrollRefusals`, `TestEnrollBoundsPerHost`; OOB human verification remains an operator act | pass within stated OOB boundary |
| 8 | Mapping uniqueness | `hosttrust.DecodeTrust` + admission; duplicate credential/root/key/digest cases in `TestDecodeTrustRefusals`, `TestEnrollBoundsPerHost`, and lifecycle tests | pass |
| 9 | Rotation/retirement bounds | `hosttrust.Rotate`, `MarkRetiring`; `TestRotateBoundedWindow`, `TestRotateCapsAtLeafExpiry`, `TestMarkRetiringCapsAtLeafExpiry` | pass |
| 10 | Revocation tombstones | `hosttrust.Revoke`, `Enroll`; `TestRevokeTombstone`, `TestRevokedLeafNeverReenrolls`, `TestRevocationClosesAuthorization` | pass |
| 11 | Old-generation mutation refusal | `hosttrust.WithMutationAuthorization`; `TestWithMutationAuthorizationRefusesStaleGeneration`, `TestMutationAuthorizationRefusesStaleAfterConvergence` | pass |
| 12 | Shared authorization serialization | `hosttrust.Open`/lock, `WithMutationAuthorization`, `Revoke`; `TestFirstOpenPreservesHeldLock`, `TestConcurrentOpenAttachesToExistingLock`, `TestMutationAuthorizationSerializesWithRevocation`, `TestMutationAuthorizationSerializesWithSeparateStoreRevocation` | pass |
| 13 | Coherent config+trust snapshot | `config.LoadCoherent`; `TestSurvivingReaderConvergesInterruptedApply`, `TestSurvivingReaderConvergesInterruptedRollback`, `TestSurvivingReaderRefusesDivergedConfig`, barrier and revalidation tests | pass |
| 14 | Unix owner custody | `hosttrust.Open`, `ValidateCustody`; `TestCustodyRefusals`, `TestValidateCustodyBindsDirectory`, `TestOpenCreatesOwnerOnlyLayout` | pass on Darwin |
| 15 | Windows equivalent ACLs | `verifyOwner`, `installOwnerOnlyACL`, `secureStaged`; `custody_windows_test.go` seven native tests | stated bound: no Windows runtime runner; cross-build, test-compile, and vet pass |
| 16 | Complete peers + selected credential | `config.PreviewV4`; `TestPreviewV4PeerEnrollment`, `TestPreviewV4CredentialGates` | pass |
| 17 | Exact preview + confirmation | `config.ApplyV4`; `TestApplyV4PreviewMismatch`, `TestApplyV4ConfirmRequiredWithDrops` | pass |
| 18 | Current source/generation apply | Config4 apply + `hosttrust.JointCommit`; `TestApplyV4StaleSource`, `TestApplyV4StaleGeneration`, revalidation and stale-marker tests | pass |
| 19 | Crash-durable config/trust pair | Config4 apply/rollback + `hosttrust.JointCommit`, `Recover`, convergence; `TestApplyV4CrashConvergesGeneration`, surviving-reader/barrier/compensation tests | pass |
| 20 | Explicit backup rollback | `config.RollbackV4`; `TestRollbackV4*`, `TestPublishedBackupNamesStayExcluded`, `TestApplyV4BackupIsClassifiedConfigCopy` | pass |
| 21 | Secret exclusion across export surfaces | `hosttrust.MatchExcludedFromReplication`, `ExcludedConfigDirName`; matcher and persisted handoff/owner-note evidence | stated bound: repository matcher only; downstream consumers and diagnostics are unowned |

Rows 15/21 are NOT DRIVEN: no native Windows ACL runner, and downstream
replication/diagnostic consumers remain unowned. No post-lock-release dispatch,
mesh-wide revocation, physical power-loss, or authenticated-human verification
guarantee is inferred. RPC transport/admission remains outside this leaf.

## Validation performed by this reviewer

- Clean immutable config, hosttrust, peeridentity suites with `-count=1 -cover`:
  exit 0; coverage 92.9%, 76.2%, 97.5% respectively. Localstore/secprim suites:
  exit 0; 83.8%, 94.4%.
- 17 independent standalone census plants: all exit 1 with census assertion
  failure and passing actual-file witness. Additional unlisted-call-site plant:
  exit 0 (survivor). Rebinding regression: exit 1 (product defect). Second-pair
  foreign-root refusal and committed hold controls: exit 0.
- Independently reran config neutral, writepath-interface-name-heuristic,
  config-association-root-skip; neutral exit 0, both mutants exit 1 at expected
  named tests. The census mutant runs the behavioral package suite but its kill
  is the census expectation, not a runtime authorization rejection.
- Independently reran hosttrust neutral and hold-state-root-skip: neutral exit 0;
  mutant exit 1 at TestHeldExclusiveRejectsForeignConfigStateRoot.
- Offline `go mod tidy -diff`, and `go build ./...` for darwin/arm64,
  linux/arm64, windows/amd64: all exit 0. No native Windows runtime claim.
- Producer full-repo tests/coverage/race/vet, cross test compilation, and full
  mutation batteries were inspected from the digest-verified local archive,
  not rerun wholesale. 52 raw mutation rows, 47 unique non-neutral names;
  the initial hold-state-root compile error remains excluded, its corrected
  behavioral kill is recorded. Duplicate neutral/config-association runs are
  not counted as unique gates. See raw-mutant-audit.json for actual fail events
  and log hashes; producer-mutant-audit.json preserves command/exit metadata.

## Preservation and provenance

RPC backup hashes: 15/15; parked RPC hashes: 11/11, unchanged. There is **one
literal changed-path overlap: README.md** (already present in CR10); no RPC
implementation-path overlap. The requested literal zero-overlap claim would be
false, so it is not made. This leaf's README change describes host credentials,
Config4 and tooling; the original RPC README bytes remain preserved in backup.
All 74 candidate delta files match the immutable tree after probes were removed.
SSH and peer checkpoint directory comparisons against their prior checkpoint
remain unchanged (apart from this leaf's added v4 interoperability test).
Both ef0b63b and 3058751 signatures verify against the configured author key.
Pinned spec source document hash matches commit
0cbdf100dbf84df50c64f792b1f940e3a67859a6.

Tool-readiness/operational anomalies: project-local skill directories are absent
inside this archived/worktree layout; the Curator-managed Go skill was read from
the canonical main repository. Initial unsupported resources() query was replaced
by the documented scoped get projection. One reviewer scratch-file write used a
wrong relative path and failed; its resulting no-test run is not evidence. The
corrected probe was rerun and its actual failure retained. Directives reads
succeeded with none recorded. No unchanged handoff retry or source repair was
attempted by this reviewer.

## Verdict routing

Changes requested, route to **to-dev**, not blocked and not accepted. Reopen
checklist items 6, 11, 12 and 18: full acceptance, negative coverage, narrowing
coverage and implementation match are not true while these counterexamples
remain. Retain passed tests and explicit bounds; repair F1/F2 and the evidence
attachment/hygiene issues in the next producer candidate. No accept_cr,
commit_ack, producer checkpoint, commit, or trunk integration was performed.
