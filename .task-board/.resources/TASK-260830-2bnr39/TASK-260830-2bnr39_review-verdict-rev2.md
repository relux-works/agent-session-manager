# TASK-260830-2bnr39 — review verdict, CR revision 2

Verdict: **accepted**. No blocking findings remain from F1–F7.
Acceptance action: accept_cr(TASK-260830-2bnr39, revision=2,
evidence=TASK-260830-2bnr39_review-verdict-rev2.md).
Expected routing: integrating, accepted but not landed. Integration belongs to
a new tracked developer/implementer run routed by the orchestrator.

## Scope and identity

Reviewer run RUN-260907-428f33. Board goal GOAL-260907-00b1f8 revision 1,
kind/success predicate reviewer_verdict/reviewer_verdict, resolved scope only
TASK-260830-2bnr39. Parent GOAL-260907-d09cee revision 1 is unchanged.
The latest objective requires exactly one evidence-backed reviewer verdict for
this assigned leaf. Acceptance under the current CR contract is accept_cr,
which records accepted and routes integrating; it does not claim trunk delivery.

CR-TASK-260830-2bnr39-2 revision 2; repository_delta=present (23 paths).
Base/unchanged Story HEAD: 7aa151a9c31071bfab190fd8ae8259f502f2ebbc.
Candidate tree: 935f624e45088a064de04945cbe4bd86d1b37d1b.
Patch SHA-256: 922d14c837c914f93c49c4da259d1d78d1d25dbec74f8539231194629b9458c5.

All 2,924 candidate files, symlink targets and executable modes were compared
with the published tree. The final verifier reconstructs Git blobs and trees
from current filesystem bytes without using/writing the real index, checks the
tracked/nonignored path set, and requires the identical candidate tree and HEAD.
All baseline validation ran in an archive of that exact tree. New review probes
and mutations ran in separate disposable copies under this task's .temp path.
No product edits, commits, branch operations, delegation, or upstream #176/#177
work occurred. Review-only artifacts remain outside the candidate.

Normative source: agent-session-manager-spec v0.5.0, commit
28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, §§10.2–10.4 and 12.1–12.3.
Embedded SPEC.md SHA-256 matches the pinned lock:
562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a.
The complete produced-string audit was checked against §10.4's embedded types.
Required-filter count 0..64 is a normative array bound; it is not a 64-character
bound on a filter name. Native paths and supplementary delta OIDs are internal
metadata, not additional normative serialized wire members.

## AC and production coverage

**8 of 8 AC rows driven**, checked before code inspection and matched against
actual executed tests in all-tests.log; see ac-driver-audit.json. These are
candidate-resident tests, awaiting the producer's managed signed checkpoint.
They are not hand-committed by the reviewer. Capture is the production internal
library entry. No other production package yet imports gitsnap; CLI integration,
full workspace closure and materialization are not claimed by this leaf.

| AC row | Production call site | Executed named drivers |
| --- | --- | --- |
| Repository identity | Capture → readRemotes → deriveIdentity | TestCaptureLiveBranchRepository; TestRemoteFetchPushMapping; TestCaptureRemoteNameBounds |
| HEAD/ref | Capture → readHead → recheckConsistency | TestCaptureLiveHeadModes; TestReviewHeadRefRace; TestCaptureSameOIDHeadModeRace; TestCaptureRefGrammarAtEachRead; TestCaptureSuccessfulEmptyHeadReadRefuses |
| Worktree metadata | Capture → readRepository | TestCaptureRootAndSubdirectoryAgree; TestCaptureLinkedWorktreeSubdirectory |
| Index stages | Capture → parseIndexEntries | TestCaptureLiveConflictStages; TestCompatConflictStages; TestRefuseIndexStageFour |
| Index flags | Capture → parseIndexRecord/debugFlags | TestCaptureLiveIndexFlags; TestCaptureLiveBranchRepository; TestCompatFSMonitorBit |
| Staged deltas | Capture → parseDeltas (cached stream) | TestCaptureLiveBranchRepository; TestReviewBinaryRenameControl |
| Unstaged deltas | Capture → parseDeltas (worktree stream) | TestCaptureLiveBranchRepository; TestCaptureLiveMissingTrackedPath; TestReviewLockOverrideMutatesIndex |
| File modes | Capture → statIndexPaths/statDeltaPaths | TestCaptureLiveExecBitAndSymlinkKind; TestRefuseSpecialWorktreeFile; TestCaptureRootAndSubdirectoryAgree |

## Previous findings and independent attacks

- **F1 — original index preservation:** Runner-owned GIT_OPTIONAL_LOCKS=0
  overrides inherited =1. TestReviewRunnerLockOverride exercises the actual
  environment; TestReviewLockOverrideMutatesIndex exercises real Git after
  replacing a tracked file with identical bytes, and requires correct deltas
  plus unchanged original bytes. N28 weakens environment precedence; N29 keeps
  private indexes for cached diff but lets unstaged diff bypass them. The real
  producer tests also exercise alternate-index precedence and success/refusal
  cleanup. Independent TestIndependentSplitIndexRefresh runs four combinations
  (normal/alternate × identical/changed content), repeats each Capture, and
  compares original index bytes, inode identity, mtime, shared-index bytes,
  logical deltas and temporary-directory cleanup. All four pass. Native
  nanosecond index-copy mtime fidelity passes; an explicit microsecond fixture
  also passes. TestIndependentPrivateIndexTimeoutCleanup executes a real wrapper
  through ExecGitRunner, times out after private-index creation, and observes no
  temporary leak. Private copies are a real-Git implementation, not a fake of
  impossible no-write porcelain semantics.
- **F2 — state consistency:** Candidate tests refuse same-OID branch switching,
  mode-only detach via update-ref --no-deref, v2→v4 rewriting and identical-byte
  index replacement. Independent TestIndependentReverseIndexVersionRace also
  refuses a real v4→v2 rewrite and then succeeds on stable retry. These checks
  run from Capture → recheckConsistency → readHead/readIndexState/readIndexVersion,
  not directly against a standalone guard. N25/N26/N27 isolate logical-index,
  same-OID HEAD-state and actual-index-format weakenings.
- **F3 — subdirectory capture:** Root/sub snapshots agree except requested cwd,
  include all repository-relative index paths, and stat paths from the root.
  Linked-worktree capture drives its own Git/common-directory metadata.
  N32 retains the searched root token while restoring the caller-directory
  behavior and runs the full behavioral suite.
- **F4 — failed reads versus absence:** Capture drives nine fallback-bearing
  readers across fatal, transport, partial-absence and diagnostic-absence shapes
  (36 subcases), plus closing HEAD failures, after-census filter disappearance,
  absent-index config failure and successful empty HEAD output. All relevant
  named tests execute in the full baseline run. Upstream absence now comes from
  a successful for-each-ref field read; configured tracking with a missing remote
  target remains a valid non-null upstream ref. N20 admits one quiet fatal
  exit-128 member of the rejected class and is tested through Capture.
- **F5 — feature facts:** Real Git accepts default symlink/filemode behavior,
  explicit false overrides, dotted required-filter names, true/yes/on/numeric
  spellings, false/no/off/zero and last-value precedence. TestMain and fixture
  subprocesses isolate Git config/routing/identity from the host. N30/N31 attack
  the default and Git boolean-parser path. Features are documented as effective
  Git configuration, not newly measured filesystem capabilities.
- **F6 — name/count bounds:** Real Capture accepts remote-name lengths 128 and
  refuses 129 for ASCII and multibyte names. N23 widens exactly that boundary.
  The added required-filter gate has 0/64/65 candidate drivers and N24. Independent
  TestIndependentFilterLiveBounds confirms real Git at 64, refusal at 65, and
  recovery after the 65th filter is configured false.
- **F7 — evidence instrument:** The supplied final archive's 35 vector results
  were checked against complete compile and behavior logs, executed test names,
  failures, and actual exit markers. Both neutral rows actually execute full
  suites. The known-bad inverted-sort control reaches TestAcceptBaselineSnapshot.
  N20/N21/N22 provide the formerly missing GateNotRepository, GateHeadRef and
  GateUpstreamRef narrowing witnesses. M10/M11/N32 preserve searched tokens but
  change behavior and execute full behavioral suites. The independent replay
  results, not source-registration linkage alone, determine the final ratios.

## Platform and contract bounds

Git 2.50.1 (Apple Git-155), Go 1.25.5, macOS arm64. Live Git fixtures contact no
remote. Scripted fsmonitor-valid and SHA-256 tests do not establish native runtime
support. Linux/Windows execution, every Git extension, external filter sandboxing,
change-and-revert races and all interior refusal clauses remain unproven bounds.

**Auxiliary-file observation:** TestIndependentSplitAuxiliaryTimestampBound sets
an old sharedindex.* mtime and observes Git refresh it during Capture. Its bytes
remain preserved in the independent split-index matrix. This confirms the
producer's stated auxiliary-timestamp bound: the accepted public contract is
original-index byte preservation with disposable diff indexes, not zero writes
to every filesystem metadata field. The idempotency_test.go preamble's broad
"mutates no durable state" wording is not used as evidence for that stronger
claim. A hard process kill can leave disposable OS-temp files, as documented;
no durable snapshot publication or recovery transaction is implemented here.
No platform/product decision is needed to accept this bounded read/capture leaf.

README, doc.go, acceptance map and string-domain audit accurately delimit the
internal capture result and do not advertise a CLI/doctor capture capability.
Existing normative ownership remains 17/428 discharged clauses globally; the
leaf's 8/8 driver count does not change that measure. Object packs, raw-index
blob delivery, untracked/ignored bytes, symlink targets, recursive submodule
capture, manifests, full included-file digests, quiescence and wire assembly
remain sibling/integration responsibilities. This review has not expanded them.


## Executed validation and mutation measurement

Reviewer reran the following on the immutable candidate, with full stdout,
stderr and actual subprocess exits retained. Commands were bounded (Go package
timeouts 3–4 minutes; command wrapper 8 minutes; mutation commands individually
4 minutes). No command was left running for the end of this review. One catalog
completion overlapped only the read-only formatting check by less than a second;
all Go test, coverage, probe and mutation runs completed before the next began.

| Reviewer command | Exit | Result/evidence |
| --- | ---: | --- |
| go test ./... -v -count=1 -timeout=4m | 0 | 24/24 packages pass; complete baseline and 8/8 AC drivers; all-tests.log |
| go test ./... -cover -count=1 -timeout=4m | 0 | 24/24 packages pass; gitsnap 86.5% statements; coverage.log |
| go test ./internal/gitsnap -run ^TestIndependent -v -count=1 -timeout=3m | 0 | Initial four independent probe functions pass; independent-probes.log |
| go test ./internal/gitsnap -run ^TestIndependent -v -count=1 -timeout=3m | 0 | Seven probe functions pass; four split-index combinations; native timestamp and timeout checks; independent-probes-final.log |
| python3 .scripts/gitsnap-mutations.py --out ../mutations | 0 | 35/35 applied and compiled; 33 behavioral kills, 2 measured neutral passes; mutations.log |
| go vet ./... | 0 | Repository-wide vet clean; vet.log |
| go build ./... | 0 | Repository-wide build succeeds; build.log |
| go run ./internal/traceability/cmd/tracecheck | 0 | Existing 17/428 global clause coverage retained; traceability.log |
| go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check | 0 | Pinned generated catalog current; catalog.log |
| gofmt -l internal/gitsnap | 0 | No filenames output; gofmt.log |
| git diff --check | 0 | No whitespace errors; diff-check.log |

The producer's final full race run is accepted prior evidence, **not independently
rerun** here: race-final-02.log plus its actual exit-0 record were inspected in
TASK-260830-2bnr39_rework-evidence-rev2.tar.gz. Earlier failed producer attempts
remain failed in that archive; none substitutes for the final results. The rev1
verdict and full rev1/rev2 archives were opened; their SHA-256 values and the
matching producer source manifest are recorded in this review's evidence.
The review reruns ordinary full tests and coverage independently, as required.
No native cross-platform run, fuzz budget, all-clause mutation exhaustion or
new CLI capability is claimed.

**17 of 17 registered gates have executed narrowing witnesses.** This is gate
coverage by selected behavioral weakenings, not all interior clauses. The source
census reports **77 literal refusal sites / 17 registry names**, a separate
syntactic measure. Deletion-only vectors are supporting evidence and do not count
as narrowing witnesses. There are zero NOT_APPLIED, COMPILE_FAILED, unexplained
survivors, unexecuted controls or census-only behavioral kills in the replay.

| Gate | Executed narrowing witness |
| --- | --- |
| GateNotRepository | N20-not-repository-admits-fatal-128 |
| GateHeadCorrupt | M6-unborn-admits-tags |
| GateHeadRef | N21-head-ref-admits-HEAD |
| GateHeadOIDFormat | M8b-branch-admits-40hex-in-sha256-repo |
| GateRemotesRange | M3-remotes-admits-17 |
| GateRemoteURL | N23-remote-name-admits-129 |
| GateIdentityLength | M5-identity-admits-257 |
| GateIndexVersion | M2-version-admits-5 |
| GateIndexStage | M1-stage-admits-4 |
| GateIndexEntry | M18-index-admits-8hex-oid |
| GateIndexSort | M9-sort-admits-duplicate |
| GateIndexEntriesRange | M4-entries-admits-65537 |
| GateUpstreamRef | N22-upstream-admits-HEAD |
| GateDeltaStatus | M7-delta-admits-X |
| GateWorktreeKind | M19-worktree-admits-fifo |
| GateConsistency | N25-consistency-admits-same-length-index-change |
| GateFeatures | N24-filters-admits-65 |

| Vector | Reviewer result | Named failing tests / survivor bound |
| --- | --- | --- |
| C-neutral-comment | NEUTRAL_PASS | Comment-only change cannot affect runtime behavior; full suite must execute. |
| C-neutral-oid-swap-survives | NEUTRAL_PASS | Both scalar parsers enforce prefix/length; the supplied objectFormat is also the constructed prefix. This substitution is behavior-neutral at this call site; it does not cover cross-format acceptance elsewhere. |
| C-bad-inverted-sort | BEHAVIOR_KILL | TestAcceptBaselineSnapshot |
| M1-stage-admits-4 | BEHAVIOR_KILL | TestRefuseIndexStageFour |
| M2-version-admits-5 | BEHAVIOR_KILL | TestRefuseIndexVersionFive |
| M3-remotes-admits-17 | BEHAVIOR_KILL | TestEdgeRemotesSeventeenRefuses |
| M4-entries-admits-65537 | BEHAVIOR_KILL | TestEdgeIndexMaxPlusOneRefuses |
| M5-identity-admits-257 | BEHAVIOR_KILL | TestEdgeIdentity257CharsRefuses |
| M6-unborn-admits-tags | BEHAVIOR_KILL | TestRefuseUnbornNonBranchRef |
| M7-delta-admits-X | BEHAVIOR_KILL | TestRefuseUnknownDeltaStatus |
| M8b-branch-admits-40hex-in-sha256-repo | BEHAVIOR_KILL | TestRefuseCrossFormatOID |
| M9-sort-admits-duplicate | BEHAVIOR_KILL | TestRefuseDuplicateIndexEntry |
| M10-fetch-push-swapped | BEHAVIOR_KILL | TestRemoteFetchPushMapping |
| M11-assume-bit-swapped | BEHAVIOR_KILL | TestCaptureLiveBranchRepository; TestCaptureLiveIndexFlags |
| M17-push-credential-admission | BEHAVIOR_KILL | TestRefuseCredentialPushURL |
| M18-index-admits-8hex-oid | BEHAVIOR_KILL | TestRefuseIndexBadOID |
| M19-worktree-admits-fifo | BEHAVIOR_KILL | TestRefuseSpecialWorktreeFile |
| M12-remotes-min-disabled | BEHAVIOR_KILL | TestRefuseNoRemotes |
| M13-branch-ref-clause-disabled | BEHAVIOR_KILL | TestRefuseBranchWithLiteralHeadRef |
| M14-consistency-index-clause-disabled | BEHAVIOR_KILL | TestConsistencyRefusesIndexMutationArmedOnce |
| M15-corrupt-head-clause-disabled | BEHAVIOR_KILL | TestRefuseCorruptHead |
| M16-upstream-clause-disabled | BEHAVIOR_KILL | TestRefuseBadUpstream |
| N20-not-repository-admits-fatal-128 | BEHAVIOR_KILL | TestReviewFeatureReadFailure |
| N21-head-ref-admits-HEAD | BEHAVIOR_KILL | TestCaptureRefGrammarAtEachRead |
| N22-upstream-admits-HEAD | BEHAVIOR_KILL | TestCaptureRefGrammarAtEachRead |
| N23-remote-name-admits-129 | BEHAVIOR_KILL | TestCaptureRemoteNameBounds |
| N24-filters-admits-65 | BEHAVIOR_KILL | TestCaptureRequiredFilterCountBounds |
| N25-consistency-admits-same-length-index-change | BEHAVIOR_KILL | TestConsistencyRefusesIndexMutationArmedOnce |
| N26-consistency-admits-same-oid-branch-switch | BEHAVIOR_KILL | TestReviewHeadRefRace; TestCaptureSameOIDHeadModeRace |
| N27-consistency-admits-index-version-only-change | BEHAVIOR_KILL | TestReviewIndexVersionRace |
| N28-lock-env-inherited-one-wins | BEHAVIOR_KILL | TestReviewRunnerLockOverride |
| N29-diff-bypasses-private-index | BEHAVIOR_KILL | TestReviewLockOverrideMutatesIndex |
| N30-feature-default-symlinks-false | BEHAVIOR_KILL | TestReviewSymlinkDefault |
| N31-filter-boolean-raw-yes | BEHAVIOR_KILL | TestReviewRequiredFilterBoolean |
| N32-subdir-keeps-caller-command-scope | BEHAVIOR_KILL | TestReviewSubdirectoryCapture; TestCaptureRootAndSubdirectoryAgree; TestCaptureLinkedWorktreeSubdirectory |


## Disposition and evidence

Accept the exact revision 2 candidate above. The seven prior finding classes
are resolved at this leaf boundary; there is no changes_requested branch and
therefore no repeat-of finding to route. The measured auxiliary timestamp bound
is documented explicitly rather than represented as zero filesystem mutation.
A task-scoped review logbook handoff carries that observation while preserving
LOGBOOK.md and the candidate tree.

Evidence: TASK-260830-2bnr39_review-evidence-rev2.tar.gz contains complete reviewer
logs, command/exit records, per-vector compile/behavior/positive logs, the
mutation results and harness, independent probe source, candidate/source
provenance, AC-driver audit and before/after candidate identity. All artifacts
are attached before acceptance. The board's accept_cr response and subsequent
scoped read are the authoritative acceptance/routing evidence; this document
alone is not a substitute for that transaction. No done, checkpoint, integration
or trunk-delivery claim is made by this reviewer.
