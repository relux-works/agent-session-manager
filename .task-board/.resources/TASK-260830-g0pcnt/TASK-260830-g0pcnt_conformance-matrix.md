# TASK-260830-g0pcnt Conformance Matrix

## Scope and measured task coverage

Authority: internal/specdoc/SPEC.v0.7.0.md v0.7.0. Refreshed Story checkpoint: 5a64077facb9f93357b9bbc8ebdba93069105715, based on trunk 0ca3e4c26e2b275212796657f785b9b450f6174e; the adopted registry includes the cpkajd row from 14d636e. Candidate is uncommitted.

Task acceptance coverage is **4 of 4 rows**:

| AC row | Production call site | Positive/recovery test | Negative/narrowing evidence | Verdict |
|---|---|---|---|---|
| Lost-response idempotency | Lifecycle.Execute → executeAttach | TestExecuteAttachLostResponseReplaysRecordedOutcome | N-attach-lost-response-replay bypasses replay only for the fixture client; test fails on a second stage/install or receipt change. | Driven |
| Reconnect/multi-attach policy | Lifecycle.Execute(attach) → executeAttach/checkAttachOverlap | TestAttachSameClientRetryWithPeerPresent; TestAttachOverlapRequiresMultiAttach; TestAttachOverlapRequiresMultipleInputClients; TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances | N-attach-overlap-multi; N-attach-overlap-input; N-attach-overlap-replay-unproven; N-attach-postlock-generation. Each is run alone. | Driven |
| Socket substitution refusal | ServerProber.Probe; ServerSpawner.Spawn; all eight Lifecycle.Execute operation dispatches | TestServerProberRefusesSocketSubstitutionBeforeConnect; TestServerSpawnerRefusesSocketSubstitutionBeforeUnlinkOrSpawn; TestExecuteEveryOperationRefusesSocketSubstitutionBeforeDispatch | N-socket-identity-swap; N-socket-symlink; N-socket-ownership; N-socket-permissions. Each preserves other custody guards. | Driven |
| Ownership-neutral attach | Lifecycle.Execute(attach) → executeAttach | TestExecuteAttachDoesNotReadOrChangeSessionLease | N-attach-ownership-neutral performs a successor lease CAS if attach refreshes ownership; test fails. | Driven |

Additional inherited tests: TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable with N-attach-readonly-to-writable; TestForegroundAcquireComposesProductionProbeWithAttach; TestAcquireForegroundRefusesEveryCatalogDecoyRunning with N-foreground-fresh-decoy-wiring. The latter admits only a fresh headless_creation decoy. All are production-entry tests.

## Socket custody outcomes

| Entry | Substitution/member tested | Observable refusal boundary |
|---|---|---|
| ServerProber.Probe | Changed inode at derived path | Refuses before Dial; zero Dial/admission calls |
| ServerProber.Probe | Symlink at derived path | Refuses before Dial |
| ServerProber.Probe | Foreign owner or group/world access | Refuses before Dial |
| ServerSpawner.Spawn | Changed inode at derived path | Refuses before unlink and Runner/spawn |
| ServerSpawner.Spawn | Symlink, foreign owner, or permissive mode | Refuses before unlink and Runner/spawn |
| Lifecycle.Execute: create, attach, status, quiesce, safe-boundary, stop, terminate, restore | Changed inode, foreign owner, or permissive mode | Refuses before operation dispatch |
| Foreground Acquire → Production.ProbeServer | Valid private active socket | Composition reaches attach on the real production probe path |
| Background Acquire | Derived local socket connection | Unreachable: background admission verifies runtime custody then contacts broker; it does not connect/unlink/spawn the derived socket |
| All entries | Replacement after final socket lstat and before OS connect | Bound B46: Unix has no portable path-relative connect identity pin |

Revision 6 supersedes the rev5 fixture-depth bound for walk length. The physical-depth test generates a runtime-root pathname of `unix.PathMax-1` bytes, exercises the deepest and middle ancestors through the shared production predicate, and statically proves Probe, Spawn, and Execute reach that predicate. The transitive guard derives the custody call graph and checks every in-module callee for path-length/depth branches and counters. Rev5's finite mode/kind/owner measurements remain unchanged: 270,336/270,336 mode classifications, 330/330 kind cases, and 54/54 owner cases. N2 remains the explicit ancestor-owner bound.

## Attach state and effect table

| Case | Production entry | Expected state/effect | Named evidence |
|---|---|---|---|
| Same client retries after lost response | Lifecycle.Execute(attach) | Original vector, receipt, effect evidence, and timestamp replay; no second effect | TestExecuteAttachLostResponseReplaysRecordedOutcome |
| Same client reconnects while peer receipt exists | Lifecycle.Execute(attach) | Validated receipt replay is exempt from new-client overlap admission | TestAttachSameClientRetryWithPeerPresent |
| Distinct read-only client overlaps without multi_attach | Lifecycle.Execute(attach) | Refusal; no second durable receipt/vector | TestAttachOverlapRequiresMultiAttach |
| Distinct input client overlaps without required multi-input policy | Lifecycle.Execute(attach) | Refusal; no second writable receipt/vector | TestAttachOverlapRequiresMultipleInputClients; TestAttachConcurrentInputRequiresAXPolicy |
| Two Life instances race from empty census | Lifecycle.Execute(attach) | Persistent per-instance admission lock serializes census through receipt commit | TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances |
| Generation rotates during lock wait | Lifecycle.Execute(attach) | Post-lock generation check refuses stale generation | TestAttachRefusesGenerationRotatedDuringLockWait |
| Read-only receipt is retried with InputAuthorized=true | Lifecycle.Execute(attach) | Literal idempotency_mismatch; no vector or receipt mutation | TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable |
| Attach attempts to refresh/transfer lease | Lifecycle.Execute(attach) | No lease read/refresh/event write; before/after state identical | TestExecuteAttachDoesNotReadOrChangeSessionLease |
| Receipt exists after disconnect/detach | Lifecycle.Execute(attach) | UNKNOWN liveness remains a possible peer; may conservatively block a new attach | B44, owned by a future authoritative receipt/protocol owner |

## Importer outcomes

The outcome grid was rerun against exact trunk 0ca3e4c26e2b275212796657f785b9b450f6174e and the current Story candidate. It compares (package, entry, input) keys over the complete importer set: 192 base keys / 35 packages, 223 candidate keys / 36 packages, 192 shared, 0 base-only, 31 candidate-only, 0 moved. No shared runtime outcome changed.

Candidate-only keys: termbind/Peers/{bad-identity, corrupt, empty, found}; tmuxserver/BuildArgv/{empty-session, empty-socket, valid}; CheckSocketLength/{long, short}; ClassifyStatus/{absent, live-tri, negative, parked, quiescing-memory}; DirectivesFor/{attach, bogus, create, manifest, probe, quiesce-input, request-stop, restore, status, terminate-stale, wait-safe-boundary}; ResolveSocket/{ambient, conventional}; ServerProber.Probe/{derived-path-symlink, owner-execute-0700, permissive-0644}; SocketPath/{fixed}. These have no matching base package/entry key.

## Mutation attribution

The full shipped-harness results and raw per-plant logs are in the task
evidence bundle. Historical rev4 contained 349 rows; rev5 expanded that set to
355 and ran it twice in complete bounded passes: 355 unique rows, 353 KILLED,
two expected survivors, zero NOT_APPLIED/MISMATCH/ERROR, and 24/24
candidate-file hashes restored in each pass. The current rev6 harness has five
added walk-length narrowings (360 total): each new plant was run alone twice
with raw subprocess logs; the full 360-row harness was not rerun. The survivor
bounds and named tests are in the results and mutation-table artifacts.

| Gate | Narrowing | What it admits or changes | Named behavioral killer |
|---|---|---|---|
| Lost-response retry | N-attach-lost-response-replay | Reinstalls the same receipt for the fixture key | TestExecuteAttachLostResponseReplaysRecordedOutcome |
| Store idempotency | N-attach-readonly-to-writable | Drops the input-authorization mismatch refusal | TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable |
| Ownership neutrality | N-attach-ownership-neutral | Refreshes ownership during attach | TestExecuteAttachDoesNotReadOrChangeSessionLease |
| Same-client serialization | N-attach-same-client-concurrent-winner | Bypasses the per-instance lock for the concurrent fixture | TestExecuteSerializesSameClientAttachBeforeConcurrentWinnerArm |
| Socket combination | N-socket-combination-admit | Admits invalid mode 0642 | TestCustodyModeOracleAtProductionEntries |
| Runtime-dir combination | N-runtime-dir-combination-admit | Admits invalid mode 0750 | TestCustodyModeOracleAtProductionEntries |
| Root mode | N-root-admit-0750 | Admits rev2 survivor mode 0750 | TestCustodyModeOracleAtProductionEntries |
| Ancestor sticky exception | N-ancestor-setgid-as-sticky | Treats setgid as sticky and admits 02777 | TestCustodyModeOracleAtProductionEntries |
| Ancestor sticky exception | N-ancestor-setuid-as-sticky | Treats setuid as sticky | TestCustodyModeOracleAtProductionEntries |
| Ancestor sticky exception | N-ancestor-sticky-ignored | Refuses sticky writable ancestors | TestCustodyModeOracleAtProductionEntries |
| Harness control | C-control | Harmless line-count-preserving comment edit | Expected survivor; harness control |

A complete shipped-harness pass ran in seven bounded groups, all exit 0: 347 KILLED, two explicit SURVIVED, and zero mismatches. The first pass attempt had three NOT_APPLIED source anchors (`N-bind-ancestor-writable`, `N-mode-0777-verify`, and `N-socket-ownership`). The first two were corrected in bounded group reruns; `N-socket-ownership` was corrected and run alone. The per-row audit verifies two isolated raw kill/survivor observations for every table row across the repaired first evidence set and the complete pass. The raw-log audit found zero missing exits or mismatches. The complete table records
the named failing test(s) for every KILLED row: see
`TASK-260830-g0pcnt_mutation-table.md`. The two survivors are:

| Survivor | What it narrows / does | Why it survives (bound) |
|---|---|---|
| D-attach-entry-deadline-shadowed | Admits the exact on-deadline instant past the entry arm | The earlier wait bound refuses the same instant before admission; owner `Lifecycle.Execute` wait/admission boundary, measured by `N-attach-postlock-deadline` and `N-attach-wait-deadline`. This supplementary shadowed plant is not counted as a gate narrowing. |
| C-control | Harmless comment-only, line-count-preserving control | Expected to survive; confirms the harness can report a no-behavior-change control. |

## Story-final registry convergence

The adopted registry was merged from trunk at 14d636e and preserved through
the config-only trunk moves to 360c8bd and 0ca3e4c. The decoded registry carries
all four Story leaves (35urbp, 1c28dz, vcx6yo, g0pcnt) and the g0pcnt cases
in their clause acceptance lists. The rev4 edge audit checked the six required
Story/trunk cases, their production owners, and every named test reference.
The cpkajd case story-260922-derivation-side-profile-source remains in clause
2.4#2. The re-derived canonical digest is
3664ab2fb166189545fea34a068a318af4f251689f29c92915fa185fefdedeea.

Current trunk is 0ca3e4c26e2b275212796657f785b9b450f6174e. The move from
360c8bd changed only task-board.config.json; that file is byte/blob-equal to
trunk in the candidate. Default exact-tree tracecheck exits 0 with its measured
coverage pinned in README. The candidate remains uncommitted for the Story
snapshot.

## Exact census mirrored in the implementation traceability file

The following table sections are copied from internal/tmuxserver/TRACEABILITY.md so every cell has its named test plus isolated narrowing, bound with owner, or unreachable reason:


### Inherited overlap gate axes

### TASK-260922-vcx6yo review rework — gate axes

The rev1 verdict found that the lock and concurrent-input gate had only one
axis measured. The following table enumerates each owned gate across client
class, peer count, sorted peer position, replay versus new client, and
concurrent versus sequential calls. Every cell is either test-and-narrowing
evidence or an explicit bound with an owner.

| Gate | Client class | Peer count | Peer position in client-ID order | Same-client replay vs new client | Concurrent vs sequential | Server generation |
|---|---|---|---|---|---|
| `admissionlock` | `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` / `N-attach-admission-lock` (input); `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances` / `N-M2-attach-admission-lock-input-only` (read-only) | Both paused tests race from an empty census through the first receipt install; after release the contender re-censuses one peer. `N-M2` kills the read-only narrowing. | BOUND: the key is the terminal instance and acquisition precedes `Peers` sorting. Owner: `AttachStore.AcquireAdmission`. | BOUND: the peer-present replay test is sequential; no concurrent duplicate-replay test isolates lock behavior. Replay/new share the same unconditional acquisition. Owner: `executeAttach`. | Both paused tests drive two independent Lifecycle values; `N-attach-admission-lock` and the reviewer-shaped `N-M2` run as isolated narrowing rows.  `TestAttachRefusesGenerationRotatedDuringLockWait` / `N-attach-postlock-generation` refuses a generation rotated while waiting before any receipt commits. |
| `overlapmulti` | `TestAttachOverlapRequiresMultiAttach` / `N-attach-overlap-multi` (read-only); `TestAttachOverlapRequiresMultiAttachForInputClient` / `N-attach-overlap-multi-input-client` (input-authorized) | One-peer refusal and a later two-peer refusal are driven by `TestAttachOverlapRequiresMultiAttach`; `N-attach-overlap-multi` admits the one-peer class. | BOUND: the decision reads only whether `len(peers)>0`, not ordering or receipt fields. Owner: `checkAttachOverlap`. | `TestAttachSameClientRetryWithPeerPresent` covers read-only and input replay; `N-attach-overlap-replay-unproven` admits a receiptless new client named as a peer. | BOUND: no second multi-attach-only concurrency mutant; the paused read-only empty-census interleaving is assigned to `admissionlock`. Both decisions execute under that same lock. Owner: `executeAttach` / `checkAttachOverlap`.  BOUND: `executeAttach` owns generation admission and post-lock recheck; the multi-attach decision consumes only the receipt census. No generation-specific capability narrowing. Owner: `executeAttach`. |
| `overlapinput` | `TestAttachOverlapInputRequiresMultipleInputClients` proves read-only observation remains allowed and new input refuses; `N-attach-overlap-input` weakens the gate. | One-peer input, two-peer mixed census, and three-peer census are exercised by `TestAttachOverlapInputRequiresMultipleInputClients` and `TestAttachOverlapInputGateChecksEveryPeer`; `N-M5-attach-overlap-input-first-peer-only` admits only the narrowed mixed-peer class. | `TestAttachOverlapInputGateChecksEveryPeer` sorts a read-only peer first, then one or two input-authorized peers; isolated `N-M5` kills both subtests. | `TestAttachSameClientRetryWithPeerPresent` covers validated read-only/input replay; `N-attach-overlap-replay-unproven` catches the receiptless new-client exemption, while `N-M5` catches a new input client with later input peers. | `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` races input-authorized clients with `multiple_input_clients` absent for the contender; `N-attach-admission-lock` admits from the stale empty census. Sequential decisions are killed by `N-attach-overlap-input` and `N-M5`.  BOUND: `executeAttach` owns generation admission and post-lock recheck; the input-overlap decision consumes receipt flags and AX input policy. No generation-specific capability narrowing. Owner: `executeAttach`. |
| `attachaxpolicy` | `TestAttachConcurrentInputRequiresAXPolicy` refuses input without matching AX authorization using literal `terminal_backend_unauthorized`; `N-attach-ax-input-policy` admits that input class. | BOUND: the shared request gate consumes no peer count. Owner: `terminalbackend.CheckAttachRequest`. | BOUND: neither CheckAttachRequest call receives the peer slice. Owner: `terminalbackend.CheckAttachRequest`. | `TestAttachSameClientRetryWithPeerPresent` covers valid policy on read-only and input replay. BOUND: invalid-policy replay has no separate Lifecycle narrowing row; the store rechecks the same binding. Owner: `AttachStore.Attach`. | BOUND: no simultaneous AX-policy narrowing is claimed. The request is checked at entry and again in the receipt commit after the wait. Owner: `CheckAttachRequest` / `AttachStore.Attach`.  BOUND: `CheckAttachRequest` does not consume generation; `executeAttach` admits generation around the attach gate. Owner: `executeAttach`. |
| `overlapreplay` | `TestAttachSameClientRetryWithPeerPresent` covers read-only and input-authorized validated replay; `N-attach-overlap-replay-unproven` admits a receiptless caller. | BOUND: the replay cases include one other peer; more than one distinct peer is not separately driven. Owner: `AttachStore.Lookup`. | BOUND: requester proof uses `Lookup(session, instance, client)`; peer ordering does not prove replay. Owner: `checkAttachOverlap`. | The named replay test contrasts the Lookup-proven caller with a new client; isolated `N-attach-overlap-replay-unproven` kills bare-ID exemption. | BOUND: no concurrent duplicate replay is claimed; all requests acquire the same per-instance lock before Lookup. Owner: `executeAttach`.  BOUND: generation is admitted by `executeAttach`; receipt lookup only proves same-client replay. Owner: `executeAttach`. |
| `livenessunknown` | `TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer` covers read-only and input-authorized receipts whose returned vectors have not been executed; `N-attach-liveness-unknown` retires the receipt and is killed. | BOUND: a single peer is the direct liveness witness; the protocol has no status or retirement signal, so all receipts returned by `Peers` remain possible. Owner: future authoritative receipt/protocol owner (B44). | BOUND: liveness reads no position or client-live field; `Peers` sorts but does not discard valid receipts. Owner: `AttachStore.Peers`. | BOUND: replay exclusion is receipt/key validation, not liveness; no positive evidence can retire another client's receipt. Owner: `checkAttachOverlap` / `AttachStore.Peers`. | BOUND: there is no mutable liveness signal to race; the read-only peer census is serialized by `admissionlock`. Owner: `AttachStore.Peers`.  BOUND: generation does not establish client liveness; no positive generation-aware retirement signal exists. Owner: future authoritative receipt/protocol owner (B44). |

Axis count: **16 of 36 cells** are test-and-narrowing witnessed; **20 of 36**
are explicit bounds above. The added generation axis has one measured cell and
five owner-named bounds. The gate × lifecycle-entry census contains six
gates × eight `Lifecycle.Execute` operation entries: six attach cells are
measured and 42 non-attach cells are unreachable because dispatch selects a
different handler. Symmetry: `attachaxpolicy` has 2 `CheckAttachRequest`
call sites / 1 shared gate / 1 lifecycle entry / 1 shared narrowing;
`admissionlock` has 2 OS lock implementations (`flock`, `LockFileEx`) / 1
shared API / 1 lifecycle call site / 1 Unix interleaving witness. Windows is
cross-built and vetted; this run has no Windows-host runtime witness.

Ratio: 270 measured of 5632 (50+10+210 M across Tables A/B/D;
Table C all U); 191 bound; 5171 unreachable with reasons. Every
driven refusal direction is measured except the bound cells: 27
driven but rowless (B22 singletons/forks, B23 normalized details,
B32 propagation, B36 singleton deps) and
160 named undriven gaps (B24/B25/B26/B28/B31 cells
and the 153 B39 reachable-undriven cells), each with its
owner below; B29/B40/B41/B42/B43/B44/B45 are non-cell bounds stated below.
(Table A's 4 B15 guard-name arms are bound outside this split.)

Table E: after-restore outcome grid (ExecuteWrapperRestore,
wrapper.go:97). Each row is one decided outcome with its evidence
condition, its backend effect, and the committed test plus the
narrowing row that kills a routing/order mutant through the
wrapper entry (test names drop the Test prefix):

| Outcome | Evidence condition | Backend effect | Test / row |
|---|---|---|---|
| launch | verified local win + valid materialization | executes backend restore exactly once | WrapperRestoreLocalWinResumes; reorder covered by N-wrapper-reorder-refresh |
| reattach | verified local win + recorded bootstrap pair | executes backend restore exactly once | WrapperRestoreReattachReplaysBackend; reorder covered by N-wrapper-reorder-refresh |
| attach_remote | refreshed remote winner + lapsed local grant + interactive | none (no backend op, no exec, no provider) | WrapperRestoreLapsedGrantRemoteOffer/attach; N-wrapper-reorder-refresh |
| takeover_offer | refreshed remote winner + lapsed local grant + non-capable attach | none (no backend op, no exec, no provider) | WrapperRestoreLapsedGrantRemoteOffer/takeover; N-wrapper-reorder-refresh |
| parked (remote-noninteractive) | refreshed remote winner + non-interactive terminal | none | WrapperRestoreParksWithoutEffects/remote-noninteractive; N-wrapper-reorder-refresh |
| parked (refresh-failed) | refresh error → unverified local knowledge | none | WrapperRestoreParksWithoutEffects/refresh-failed; N-wrapper-reorder-refresh |
| parked (no-adapter) | missing refresh adapter → unverified local knowledge | none | WrapperRestoreParksWithoutEffects/no-adapter; N-wrapper-reorder-refresh |
| parked (no-known-lease) | no lease known at all → zero winner | none | WrapperRestoreParksWithoutEffects/no-known-lease; N-wrapper-reorder-refresh |
| parked (invalid-material) | required materialization not admitted | none (routing mutant would execute) | WrapperRestoreParksWithoutEffects/invalid-material; N-wrapper-route-material |
| parked (refresh-timeout) | refresh blocks → cancelled at the bound, unverified | none | WrapperRestoreRefreshBoundMeasured/configured-50ms; N-refresh-bound-default (default leg) |
| parked (expired refresh) | refresh answers after the deadline with a nil error → rejected, unverified | none | WrapperRestoreExpiredRefreshParksUnverified; expiry covered by N-wrapper-refresh-expired |
| parked (late-answer race) | both-ready selection → the late answer never verifies | none | WrapperRestoreRefreshRaceRejectsLateAnswer; N-wrapper-refresh-expired |
| parked (non-cooperative adapter) | adapter ignores cancellation → the entry returns at the bound | none | WrapperRestoreRefreshBoundEnforced; N-wrapper-refresh-deadline |
| refuse (divergent winner) | refreshed winner ≠ carried restore authorization | none, failed local precondition | WrapperRestoreRefusesDivergentWinner(+Lease,+Epoch); N-wrapper-winner-lease, N-wrapper-winner-epoch |
| refuse (divergent session) | decision session ≠ restore session | none, failed local precondition | WrapperRestoreRefusesDivergentSession; N-wrapper-restore-session |
| refuse (divergent bootstrap) | decision bootstrap ≠ restore bootstrap | none, failed local precondition | ReviewWrapperBootstrapMustBindExecutedRestore; N-wrapper-restore-bootstrap |
| refuse (divergent instance) | admitted descriptor instance ≠ restore instance | none, failed local precondition | ReviewWrapperInstanceMustBindExecutedRestore; N-wrapper-restore-instance |
| refuse (divergent backend) | admitted descriptor backend ≠ restore backend | none, failed local precondition | WrapperRestoreRefusesDivergentBackend; N-wrapper-restore-backend |
| refuse (divergent generation) | admitted descriptor generation ≠ restore generation | none, failed local precondition | WrapperRestoreRefusesDivergentGeneration; N-wrapper-restore-generation |
| refuse (mode) | Decide.Mode != restore (incl. garbage) | none, invalid-arguments | WrapperRestoreRefusesMalformed; N-wrapper-mode |
| refuse (operation) | Restore.Operation != restore (incl. garbage) | none, invalid-arguments | WrapperRestoreRefusesMalformed; N-wrapper-operation |
| refuse (deps) | nil lifecycle or nil dependency | none, protocol-error | WrapperRestoreRefusesMalformed; B36 singleton (no narrowing row) |

The refresh runs before the decision on every row (N-wrapper-
reorder-refresh admits exactly the fixture session past the
refresh-success arm into stale local knowledge and is killed by
the offer/resume/park tests together), so the offer path never
presents the lapsed grant to backend restore authorization. The
resume rows execute only bound to the decided session, bootstrap,
instance, backend, generation, and winner as one composed
authorization (N-wrapper-restore-session, N-wrapper-restore-
bootstrap, N-wrapper-restore-instance, N-wrapper-restore-backend,
N-wrapper-restore-generation, N-wrapper-winner-lease,
N-wrapper-winner-epoch); late answers never verify whether they
arrive after the deadline (N-wrapper-refresh-expired) or the
adapter ignores cancellation (N-wrapper-refresh-deadline). No
production mesh transport backs the adapter (B37). Union-rival
ambiguity is unmodeled (B38). The parked wire result
of the backend restore entry itself (RestoredParked) stays a
backend-row concern, not a Table E row.

### Story-final leaf census

### TASK-260830-g0pcnt story-final conformance

Pinned scope is `internal/specdoc/SPEC.v0.7.0.md` v0.7.0: §3.2#8,
§4.2#4-#7/#9, and §4.C#3-#7. The adopted ownership registry carries the
four leaf cases plus the socket-custody case; the decoded clause-edge test
requires each case in the clause's `acceptance_cases` list. The canonical
registry digest and measured README subsection are verified by the task logs.

| Row | Production call site | Executed test | Narrowing evidence / bound |
|---|---|---|---|
| 93 | `Lifecycle.Execute` → `executeAttach` | `TestExecuteAttachLostResponseReplaysRecordedOutcome` | `N-attach-lost-response-replay` retries a lost response for the same key; the test compares the original durable receipt bytes/timestamp and proves one stage/install, one receipt, and no retry exec. |
| 94 | `Lifecycle.Execute` → `executeAttach`; composed after foreground `Acquire` | `TestAttachOverlapRequiresMultiAttach`, `TestAttachOverlapRequiresMultipleInputClients`, `TestAttachSameClientRetryWithPeerPresent`, `TestForegroundAcquireComposesProductionProbeWithAttach` | `N-attach-overlap-multi`, `N-attach-overlap-input`, and `N-attach-overlap-replay-unproven` are each run alone. Existing vcx6yo peer census/lock remains the authority; this leaf adds the real Production probe → Acquire → Execute attach composition. |
| 95 | `ServerProber.Probe`, `ServerSpawner.Spawn`, all eight `Lifecycle.Execute` operation dispatches | `TestServerProberRefusesSocketSubstitutionBeforeConnect`, `TestServerProberRefusesSocketSymlinkBeforeConnect`, `TestServerProberRefusesForeignSocketBeforeConnect`, `TestServerProberRefusesPermissiveSocketBeforeConnect`, `TestServerSpawnerRefusesSocketSubstitutionBeforeUnlinkOrSpawn`, `TestServerSpawnerRefusesSymlinkSocketBeforeMutation`, `TestServerSpawnerRefusesForeignOrPermissiveSocketBeforeMutation`, `TestExecuteEveryOperationRefusesSocketSubstitutionBeforeDispatch`, `TestExecuteEveryOperationRefusesForeignOrPermissiveSocket` | `N-socket-identity-swap`, `N-socket-symlink`, `N-socket-ownership`, and `N-socket-permissions` each narrow one custody member and run the named caller-entry tests. Refusal occurs before Dial, lifecycle dispatch, unlink, or spawn. A swap after final lstat and before OS connect remains B46. |
| 96 | `Lifecycle.Execute` → `executeAttach` | `TestExecuteAttachDoesNotReadOrChangeSessionLease` | `N-attach-ownership-neutral` injects a real successor-lease CAS through the attach dependency; the test fails if lease refresh is called and compares event/lease state before and after. |
| 97 | foreground `Acquire` running-server gate | `TestAcquireForegroundRefusesEveryCatalogDecoyRunning` | `N-foreground-fresh-decoy-wiring` admits exactly a fresh `headless_creation` decoy; the production Acquire test fails on that member while other decoys and stale generations continue to refuse. |

#### Gate × production-entry census

`U` means unreachable at that entry with the reason stated; `B` is an explicit
bound with its owner; `M` names a production-entry test and the isolated
narrowing that kills it. The attach overlap gates stay owned by vcx6yo and are
listed in the six-axis table above; this table records this leaf's gates and
all entries they can reach.

| Gate | Production entry | Cell |
|---|---|---|
| Fresh foreground decoy | Foreground `Acquire` | M `TestAcquireForegroundRefusesEveryCatalogDecoyRunning` / `N-foreground-fresh-decoy-wiring` |
| Fresh foreground decoy | Background `Acquire` | U Background routes only to broker-or-refuse; it never evaluates the running-server admission (`acquireBackground`). |
| Lost-response replay | `Lifecycle.Execute(attach)` | M `TestExecuteAttachLostResponseReplaysRecordedOutcome` / `N-attach-lost-response-replay` |
| Lost-response replay | Other seven `Lifecycle.Execute` operations | U Dispatch selects a different operation handler; none reads the attach receipt store. Owner: `Lifecycle.Execute`. |
| Read-only→writable replay | `Lifecycle.Execute(attach)` | M `TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable` / `N-attach-readonly-to-writable`; literal `idempotency_mismatch`, no vector or receipt mutation. |
| Read-only→writable replay | Other seven `Lifecycle.Execute` operations | U Dispatch selects a different operation handler; no attach receipt can be replayed. Owner: `Lifecycle.Execute`. |
| Ownership-neutral attach | `Lifecycle.Execute(attach)` | M `TestExecuteAttachDoesNotReadOrChangeSessionLease` / `N-attach-ownership-neutral`; no lease read, refresh, event write, or lease change. |
| Ownership-neutral attach | Other seven `Lifecycle.Execute` operations | U This property is scoped to the attach handler; other operation handlers have their own lease/fencing contracts. Owner: `Lifecycle.Execute`. |
| Socket kind, owner, mode, identity custody | `ServerProber.Probe` | M Prober substitution/symlink/foreign-owner/0644 tests above / `N-socket-identity-swap`, `N-socket-symlink`, `N-socket-ownership`, `N-socket-permissions`; zero Dial calls on refusal. |
| Socket kind, owner, mode, identity custody | `ServerSpawner.Spawn` | M Spawner substitution/symlink/foreign-owner/0644 tests above / the same four isolated narrowings; no unlink or Runner call on refusal. |
| Socket kind, owner, mode, identity custody | `Lifecycle.Execute` — create, attach, status, quiesce, safe-boundary, stop, terminate, restore | M `TestExecuteEveryOperationRefusesSocketSubstitutionBeforeDispatch` and `TestExecuteEveryOperationRefusesForeignOrPermissiveSocket` / all four custody narrowings; the tests cover all eight dispatch values. |
| Socket kind, owner, mode, identity custody | Foreground `Acquire` → `Production.ProbeServer` | M `TestForegroundAcquireComposesProductionProbeWithAttach` covers the real probe and subsequent attach on an active private socket; substitution itself is exercised at the `ServerProber.Probe` production adapter before `Dial`. |
| Socket kind, owner, mode, identity custody | Background `Acquire` | U Background verifies runtime custody then contacts the broker; the derived local tmux socket is not connected, unlinked, or spawned. Owner: `acquireBackground`. |
| Attach receipt gates | `Acquire` foreground/background | U Acquire has no attach receipt input and does not dispatch lifecycle attach. Owner: `Acquire` / lifecycle caller. |

#### Six axes for this leaf's gates

Each cell is a measured test-and-narrowing (`M`), an explicit bound (`B`), or
unreachable/not applicable (`U`) because that production gate does not consume
the axis. `N-*` rows are run individually; the attach overlap generation axis
is added to the predecessor's six-gate matrix above.

| Gate | Client class | Peer count | Peer position | Same-client replay vs new client | Concurrent vs sequential | Server generation |
|---|---|---|---|---|---|---|
| Lost-response replay | M `TestExecuteAttachLostResponseReplaysRecordedOutcome` / `N-attach-lost-response-replay` (input-authorized client) | B fixture has zero peers; peer policy is owned by `checkAttachOverlap`. | U no peer exists to order; owner `AttachStore.Peers`. | M first install followed by same-client retry under the same request key / `N-attach-lost-response-replay`. | B the lost response is retried sequentially; cross-instance serialization is owned by `admissionlock`. | B fixture generation is stable; rotated generation is independently measured by `TestAttachRefusesGenerationRotatedDuringLockWait` / `N-attach-postlock-generation`. Owner `executeAttach`. |
| Read-only→writable replay | M `TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable` / `N-attach-readonly-to-writable` (read-only receipt, input-authorized retry) | B fixture has zero peers; overlap gates own peer policy. | U no peer exists to order; owner `AttachStore.Peers`. | M same-client durable receipt replay with changed `InputAuthorized` / `N-attach-readonly-to-writable`. | B sequential retry only; concurrent duplicate-key arbitration is owned by `admissionlock`. | B generation is held constant; generation recheck is owned by `executeAttach`. |
| Ownership-neutral attach | M `TestExecuteAttachDoesNotReadOrChangeSessionLease` / `N-attach-ownership-neutral` (local-only authorized attach) | B one attach with no receipt peer; peer census remains the sole overlap authority. | U no peer position is present; owner `AttachStore.Peers`. | B one new-client attach is measured; a replay-specific lease-neutrality assertion is not separate. Owner `executeAttach`. | B no concurrent lease writer is raced; current test proves no attach-triggered lease refresh/CAS. Owner `executeAttach` / `sessrepo`. | B fixed generation; a generation rotation does not enter this ownership-neutrality test. Owner `executeAttach`. |
| Socket custody | U path-level check runs before attach client identity is consumed; client class does not enter `CheckSocketCustody`. | U no peer census exists at path custody. | U no ordered peer list exists at path custody. | U path custody precedes any attach receipt lookup. | B deterministic substitution is placed between socket lstat checks; a later concurrent swap before OS connect is B46. Owner `CheckSocketCustody`. | U socket custody does not consume a generation; attestation follows a successful connect. Owner `CheckSocketCustody`. |
| Foreground fresh-decoy gate | M `TestAcquireForegroundRefusesEveryCatalogDecoyRunning` / `N-foreground-fresh-decoy-wiring` (fresh `headless_creation` decoy) | U Acquire has no client receipt census. | U no client peers are supplied to Acquire. | U Acquire does not inspect attach replay keys. | U Acquire decoy admission is a single foreground server probe, not an attach admission race. | B generation is validated separately by `CheckServerAttested`; decoy gate only narrows fresh capability membership. Owner `Acquire`. |

Axis count for this leaf: **6 of 30 cells** are directly measured by named
tests plus narrowing mutants; the other 24 cells are explicitly bounded or
unreachable above. No tmux process runs in these tests.

#### Bounds dispositioned by this leaf

- **B44 remains open:** an attach receipt proves an admitted client claim, not
  a live tmux client. This protocol has no positive client identity/detach
  signal. Unknown liveness remains a possible peer and can conservatively block
  a later attach after detach. Owner: future authoritative receipt/protocol
  owner; overlap policy remains in `internal/termbind` and `executeAttach`.
- **B46:** after the second socket `lstat` confirms the same inode, a pathname
  replacement can still race before the OS `connect` resolves the pathname.
  Unix has no portable path-relative connect identity pin. Tests prove the
  injected swap during custody validation is refused before `Dial`; this
  narrower post-check interval remains owner-named and is not claimed closed.
- **Section 4.E replication:** no new replication prohibition claim is made;
  the runtime package has no replication transport to exercise in this leaf.

### Revision 4 baseline — superseded by Revision 5 below

This section preserves the revision 4 measurements for audit history; its counts
are superseded by the revision 5 section below. Pinned authority is SPEC v0.7.0
§3.2, lines 806–813. The revision 4 candidate compared its four custody
predicates with an independent oracle for all 4096
Unix low-12-bit modes at all three production entries: 86,016 of 86,016
component × mode × entry inputs matched. The 21 named position-entry cells cover the three fixed positions and every
actual ancestor depth at ServerProber.Probe, ServerSpawner.Spawn, and
Lifecycle.Execute. The complete path-position table follows.

| Gate | Component class / exact depth | Entry | Named subtest | Narrowing killed by the exhaustive oracle |
|---|---|---|---|---|
| checkCustodySocket | socket_leaf, depth 00 | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/socket_leaf/Probe/depth_00_socket_leaf | group/other triplets, six single-bit mask drops, combination-admit |
| checkCustodySocket | socket_leaf, depth 00 | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/socket_leaf/Spawn/depth_00_socket_leaf | group/other triplets, six single-bit mask drops, combination-admit |
| checkCustodySocket | socket_leaf, depth 00 | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/socket_leaf/Execute/depth_00_socket_leaf | group/other triplets, six single-bit mask drops, combination-admit |
| ownerOnlyRuntimeDirMode | runtime_tmux_dir, depth 01 | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Probe/depth_01_runtime_tmux_dir | group/other triplets, six single-bit mask drops, combination-admit |
| ownerOnlyRuntimeDirMode | runtime_tmux_dir, depth 01 | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Spawn/depth_01_runtime_tmux_dir | group/other triplets, six single-bit mask drops, combination-admit |
| ownerOnlyRuntimeDirMode | runtime_tmux_dir, depth 01 | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Execute/depth_01_runtime_tmux_dir | group/other triplets, six single-bit mask drops, combination-admit |
| ownerOnlyCustodyRootMode | root, depth 02 | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/root/Probe/depth_02_runtime_root | group/other triplets, six single-bit mask drops, N-root-admit-0750 |
| ownerOnlyCustodyRootMode | root, depth 02 | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/root/Spawn/depth_02_runtime_root | group/other triplets, six single-bit mask drops, N-root-admit-0750 |
| ownerOnlyCustodyRootMode | root, depth 02 | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/root/Execute/depth_02_runtime_root | group/other triplets, six single-bit mask drops, N-root-admit-0750 |
| writableByOthers | ancestor, depth 03 | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_03_ancestor | N-ancestor-walk-skips-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 03 | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_03_ancestor | N-ancestor-walk-skips-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 03 | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_03_ancestor | N-ancestor-walk-skips-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 04 | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_04_ancestor | N-ancestor-walk-stops-after-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 04 | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_04_ancestor | N-ancestor-walk-stops-after-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 04 | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_04_ancestor | N-ancestor-walk-stops-after-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 05 | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_05_ancestor | N-ancestor-walk-skips-last-below-root; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 05 | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_05_ancestor | N-ancestor-walk-skips-last-below-root; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 05 | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_05_ancestor | N-ancestor-walk-skips-last-below-root; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 06 | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_06_ancestor | N-ancestor-walk-stops-after-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 06 | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_06_ancestor | N-ancestor-walk-stops-after-first; group/other-write mask drops; setuid/setgid/sticky narrowings |
| writableByOthers | ancestor, depth 06 | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_06_ancestor | N-ancestor-walk-stops-after-first; group/other-write mask drops; setuid/setgid/sticky narrowings |

The oracle rules are independent of the production predicates. Socket leaves
admit owner read/write with optional owner execute only; runtime tmux directory
and root admit exactly 0700; ancestors admit modes with no group/other write or
with sticky, and neither setuid nor setgid substitutes for sticky. The test
uses a synthetic socket FileInfo and fake effect adapters; every admitted mode
reaches a fake next-stage effect, every refusal asserts the literal typed code
and detail with zero effect count, and no tmux process or real socket is needed.
The runtime directory path preserves the established literal
tmux_unsafe_runtime_dir at runtime mode; socket leaf/root/ancestor refusals use
tmux_unsafe_socket_path and their specific details.

The required rev2 survivors were re-planted: N-ancestor-setgid-as-sticky
admits 02777, and N-root-admit-0750 admits root 0750; the oracle kills both.
Additional narrowings admit socket 0642 and runtime-directory 0750. The
setuid-as-sticky and sticky-ignored ancestor mutations are also killed.

N1 is demonstrated by TestExecuteSerializesSameClientAttachBeforeConcurrentWinnerArm.
A first Execute is paused after staging, while a same-client request through a
second Lifecycle/store reaches the shared per-instance admission lock. The
second request cannot stage until the first commits, then it replays the same
receipt; N-attach-same-client-concurrent-winner bypasses that lock for the
fixture and the test fails.

| Gate | Client class | Peer count | Peer position | Replay vs new | Concurrent vs sequential | Server generation |
|---|---|---|---|---|---|---|
| Socket leaf | Unreachable before client identity; owner Lifecycle.Execute | Unreachable before AttachStore.Peers | Unreachable before peer ordering; owner AttachStore.Peers | Unreachable before attach receipt lookup; owner executeAttach | Sequential full-mode enumeration; B46 remains after final lstat/before connect; owner CheckSocketCustody | Unreachable before server attestation; owner CheckServerAttested |
| Runtime tmux directory | Unreachable before client identity; owner Lifecycle.Execute | Unreachable before AttachStore.Peers | Unreachable before peer ordering; owner AttachStore.Peers | Unreachable before attach receipt lookup; owner executeAttach | Sequential full-mode enumeration; later path replacement is B46 | Unreachable before server attestation; owner CheckServerAttested |
| Runtime root | Unreachable before client identity; owner Lifecycle.Execute | Unreachable before AttachStore.Peers | Unreachable before peer ordering; owner AttachStore.Peers | Unreachable before attach receipt lookup; owner executeAttach | Sequential full-mode enumeration; later path replacement is B46 | Unreachable before server attestation; owner CheckServerAttested |
| Ancestor | Unreachable before client identity; owner Lifecycle.Execute | Unreachable before AttachStore.Peers | Unreachable before peer ordering; owner AttachStore.Peers | Unreachable before attach receipt lookup; owner executeAttach | Sequential full-mode enumeration; later path replacement is B46 | Unreachable before server attestation; owner CheckServerAttested |

### Bounds

- B44 remains open: the receipt store cannot prove that a disconnected client
  has stopped. A future authoritative receipt/protocol owner must add positive
  liveness or retirement evidence.
- N2 ancestor ownership is a stated bound: checkCustodyAncestors checks kind
  and mode but not the owner above the AX runtime root. A foreign non-root
  owner could rename an otherwise protected ancestor. SPEC §3.2 lines 810–812
  says unsafe ownership must be refused but does not define which system
  ancestor owners are trusted. Owner: future tmux path-custody owner at
  checkCustodyAncestors.
- B46 remains open: substitution after final lstat and before OS connect is not
  portably pinned on Unix. Owner: CheckSocketCustody.

No acceptance-criteria row is declared out of contract: all 4 of 4 task AC rows
are driven above. The explicit bounds are carried within the applicable rows:
B44 belongs to AC row “Reconnect/multi-attach policy”; N2 and B46 belong to AC
row “Socket substitution refusal”. Their owners and evidence limits are listed
above; they are not claimed as closed.

### Importer comparison

On base 0ca3e4c26e2b275212796657f785b9b450f6174e versus the current candidate:
192 base rows across 35 packages; 223 candidate rows across 36 packages; 192
shared keys; 0 base-only; 31 candidate-only; 0 moved. Candidate-only keys are
listed in the Importer outcomes section above. No shared class changed.

Mutation raw logs and the full measured mutation table are attached in the
results/evidence resources. The current full-harness summary is 349/349 rows,
347 KILLED, two explicitly bounded survivors, and zero mismatch/error rows.
Candidate tree is UNCOMMITTED at refreshed Story checkpoint
5a64077facb9f93357b9bbc8ebdba93069105715.

### Revision 4 — custody input-axis inventory (recorded before implementation)

The rev3 finding showed that a mode sweep at one ancestor does not measure the
whole custody input. The fixture path is an ordered component list from the
synthetic socket leaf through `tmux`, the runtime root, every ancestor, and the
filesystem root. The test asserts at least three ancestor positions and
name every actual depth dynamically. The following is the complete measured
axis inventory; a bound is explicit where the contract does not define a
trusted owner set.

| Gate / component class | Axis | Enumerated values / positions | Test or bound |
|---|---|---|---|
| `checkCustodySocket` / socket leaf | Component class | Socket at `<root>/tmux/ax.sock` | `TestCustodyModeOracleAtProductionEntries` and `TestCustodyPathKindOwnerOracleAtProductionEntries` |
| `checkCustodySocket` / socket leaf | Position / depth | Depth 0, the leaf itself | Both named tests |
| `checkCustodySocket` / socket leaf | Mode | All 4096 values of the low 12 Unix mode bits, with valid socket kind and current owner | `TestCustodyModeOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `checkCustodySocket` / socket leaf | Kind | Socket, directory, regular file, symlink, FIFO; only socket admits | `TestCustodyPathKindOwnerOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `checkCustodySocket` / socket leaf | Owner | Current user admits; another non-root owner and root refuse | `TestCustodyPathKindOwnerOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `checkCustodySocket` / socket leaf | Entry | `ServerProber.Probe`, `ServerSpawner.Spawn`, `Lifecycle.Execute` | Both named tests; admission means fake Dial, spawn, or dispatch effect, refusal means zero effects |
| `ownerOnlyRuntimeDirMode` / runtime tmux directory | Component class | Directory at `<root>/tmux` | Both named tests |
| `ownerOnlyRuntimeDirMode` / runtime tmux directory | Position / depth | Depth 1 above socket leaf | Both named tests |
| `ownerOnlyRuntimeDirMode` / runtime tmux directory | Mode | All 4096 values of the low 12 Unix mode bits, with valid directory kind and current owner | `TestCustodyModeOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `ownerOnlyRuntimeDirMode` / runtime tmux directory | Kind | Directory, regular file, symlink, FIFO, socket; only directory admits | `TestCustodyPathKindOwnerOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `ownerOnlyRuntimeDirMode` / runtime tmux directory | Owner | Current user admits; another non-root owner and root refuse | `TestCustodyPathKindOwnerOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `ownerOnlyRuntimeDirMode` / runtime tmux directory | Entry | `ServerProber.Probe`, `ServerSpawner.Spawn`, `Lifecycle.Execute` | Both named tests; refusal must precede the fake next-stage effect |
| `ownerOnlyCustodyRootMode` / runtime root | Component class | Directory at `<root>` | Both named tests |
| `ownerOnlyCustodyRootMode` / runtime root | Position / depth | Depth 2 above socket leaf | Both named tests |
| `ownerOnlyCustodyRootMode` / runtime root | Mode | All 4096 values of the low 12 Unix mode bits, with valid directory kind and current owner | `TestCustodyModeOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `ownerOnlyCustodyRootMode` / runtime root | Kind | Directory, regular file, symlink, FIFO, socket; only directory admits | `TestCustodyPathKindOwnerOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `ownerOnlyCustodyRootMode` / runtime root | Owner | Current user admits; another non-root owner and root refuse | `TestCustodyPathKindOwnerOracleAtProductionEntries`, each Probe / Spawn / Execute subtest |
| `ownerOnlyCustodyRootMode` / runtime root | Entry | `ServerProber.Probe`, `ServerSpawner.Spawn`, `Lifecycle.Execute` | Both named tests; refusal must precede the fake next-stage effect |
| `writableByOthers` / each ancestor | Component class | Every actual ancestor directory above runtime root, including filesystem root | Both named tests; fixture depth is enumerated from the concrete path |
| `writableByOthers` / each ancestor | Position / depth | Depths 3 through N, parent of runtime root through filesystem root, with N >= 3 | Both named tests, one named subtest per actual depth |
| `writableByOthers` / each ancestor | Mode | All 4096 values at each depth, with valid directory kind; includes admitted `01777` at every depth | `TestCustodyModeOracleAtProductionEntries`, each Probe / Spawn / Execute and depth subtest |
| `writableByOthers` / each ancestor | Kind | Directory, regular file, symlink, FIFO, socket; only directory admits | `TestCustodyPathKindOwnerOracleAtProductionEntries`, each Probe / Spawn / Execute and depth subtest |
| `writableByOthers` / each ancestor | Owner | N2 bound: ancestors are not owner-checked because §3.2 lines 810–812 does not define trusted system-ancestor owners; a foreign non-root owner could rename one. Owner: future tmux path-custody owner at `checkCustodyAncestors`. | Explicit bound, retained from rev3; no owner value is claimed measured |
| `writableByOthers` / each ancestor | Entry | `ServerProber.Probe`, `ServerSpawner.Spawn`, `Lifecycle.Execute` | Both named tests; refusal must precede the fake next-stage effect |

All rows use an independent §3.2-derived oracle. Metadata projections are
path-specific fixture seams; the production call site and each entry's
observed effect remain under test. No tmux process or real socket is needed.


### Refresh invariants and handoff scope

Managed refresh ran task-board worktree refresh-candidate TASK-260830-g0pcnt.
git merge-base HEAD 0ca3e4c26e2b275212796657f785b9b450f6174e equals that
trunk OID. The replayed Story commits ef3b4e5bb8fdb5663a6c4e60c0b1aaa38f95f0bf,
018790a2a3470b53de7981a46b487db2e16d9241, and
5a64077facb9f93357b9bbc8ebdba93069105715 each pass git verify-commit.
task-board.config.json is byte/blob-equal to 0ca3e4c.

The full trunk delta from c9233ce to 0ca3e4c contains 64 paths. All 57
paths not changed by this Story match 0ca3e4c; the prior c9233ce..14d636e
delta contains 63 paths, and all 56 paths in that set not changed by the
Story match 14d636e. No mismatch was found. The path audit is
rev4-refresh-path-invariant.log.

No explicit catalog-derived surface table was included in the producer brief;
that is recorded as a brief gap. The accompanying coverage map is a supplemental
position × entry map, not a claim that the missing brief table was supplied.

### Revision 5 — generated depth and unbounded custody-axis inventory

Pinned authority: SPEC v0.7.0 §3.2, lines 806–813. The custody path is an
ordered sequence from the socket leaf to the filesystem root. All three
production entries are exercised: `ServerProber.Probe`,
`ServerSpawner.Spawn`, and `Lifecycle.Execute`.

| Gate | Axis | Enumerated range / named test | Structural argument or bound |
|---|---|---|---|
| `checkCustodySocket` | Component class, position, mode | Socket leaf at depth 0 in both current and extra-depth-08 fixtures; all 4096 low modes × each fixture × three entries in `TestCustodyModeOracleAtProductionEntries`. | Finite 12-bit mode domain is exhaustive. |
| `checkCustodySocket` | Kind, owner | Five kinds at the socket position; current, other non-root, and root owner identities in `TestCustodyPathKindOwnerOracleAtProductionEntries`. | Explicitly enumerated. |
| `ownerOnlyRuntimeDirMode` | Component class, position, mode | Runtime `tmux` directory at depth 1 in both fixtures; all 4096 low modes × three entries in `TestCustodyModeOracleAtProductionEntries`. | Finite 12-bit mode domain is exhaustive. |
| `ownerOnlyRuntimeDirMode` | Kind, owner | Five kinds and three owner identities at depth 1, both fixtures, all entries in `TestCustodyPathKindOwnerOracleAtProductionEntries`. | Explicitly enumerated. |
| `ownerOnlyCustodyRootMode` | Component class, position, mode | Runtime root at depth 2 in both fixtures; all 4096 low modes × three entries in `TestCustodyModeOracleAtProductionEntries`. | Finite 12-bit mode domain is exhaustive. |
| `ownerOnlyCustodyRootMode` | Kind, owner | Five kinds and three owner identities at depth 2, both fixtures, all entries in `TestCustodyPathKindOwnerOracleAtProductionEntries`. | Explicitly enumerated. |
| `writableByOthers` | Walk length and position | Extra levels 1..16 × nearest/middle/deepest non-root × Probe/Spawn/Execute = 144 named refusals in `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries`. Full mode/kind sweeps also use current depth (four ancestors) and extra-depth-08 (twelve ancestors). | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` requires an unconditional parent walk with only root exits, no counter/cap/early nil return. K4, K6, K10, deepest-skip, and structural controls all die alone. |
| `writableByOthers` | Ancestor class, position, mode, kind | All ancestors through filesystem root in both oracle fixtures; every mode and five kinds through all entries in `TestCustodyModeOracleAtProductionEntries` and `TestCustodyPathKindOwnerOracleAtProductionEntries`. | Finite mode/kind domains are exhaustive. Ancestor ownership remains N2. |
| All four gates | Component-name byte length | Runtime-root and ancestor components are exercised at every legal byte length, and the next length refuses at the production entry, in `TestCustodyPathComponentByteLengthsAtProductionEntries`. Max legal lengths on this host: Probe/Spawn root 46, ancestor 48; Execute root 50, ancestor 52. | Complete socket path length is bounded by `CheckSocketLength`; no longer value reaches custody. |
| All four gates | Component-name content | `runtime`, `.hidden`, `...`, `two..dots`, spaces, `café`, `猫`, punctuation at runtime-root and ancestor positions through each entry in `TestCustodyPathComponentNameContentAtProductionEntries`. | No lexical path-component allowlist exists in the custody predicates; they compare cleaned paths and no-follow filesystem metadata. |
| All four gates | Symlink-chain length | Lengths 1..16 at socket leaf, runtime tmux directory, runtime root, and ancestor through Probe/Spawn/Execute in `TestCustodySymlinkChainLengthAtProductionEntries`; every case refuses before effects. | Each gate stops at the first symlink via lstat/no-follow open; it does not follow remaining links. |
| `writableByOthers` | Ancestor owner | No foreign-owner sweep is claimed above the AX root. | N2: §3.2 lines 810–813 does not define the trusted owner set for system ancestors; a foreign non-root owner could rename one. Owner: future tmux path-custody owner at `checkCustodyAncestors`. |

Current exhaustive oracle counts are 22 positions × 4096 modes × 3 entries =
270,336/270,336; kind 330/330; owner 54/54. The generated unsafe-ancestor
case is 144/144. Every refusal asserts literal `tmux_unsafe_socket_path at
socket ancestor` and zero fake effects; each admit reaches a fake next stage.
There is no tmux process in these witnesses.

Isolated rev5 narrowing attribution (each plant ran alone twice with raw
subprocess logs):

| Mutant | Class admitted by weakening | Named killer | Outcome |
|---|---|---|---|
| `N-ancestor-walk-depth-cap-4` | Unsafe mode after four checked ancestors | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice |
| `N-ancestor-walk-depth-cap-6` | Unsafe mode after six checked ancestors | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice |
| `N-ancestor-walk-depth-cap-10` | Unsafe mode after ten checked ancestors | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice |
| `N-ancestor-walk-skips-deepest-generated-ancestor` | Unsafe mode at the deepest non-root position | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice |
| `N-ancestor-walk-depth-cap-ast-control` | Adds a counter/cap | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | KILLED twice |
| `N-ancestor-walk-early-return-ast-control` | Adds `return nil` outside the root exit | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | KILLED twice |
| `C-control` | Harmless line-count-preserving comment | None | SURVIVED twice as expected |

The harness's `--help` flag is unsupported and returned
`unknown mutants: --help` (exit 2); the supported invocation is documented
in its source. This invocation failure does not affect the measured named runs.


Current rev5 full-harness audit: both complete passes each contain 355/355
unique rows, 353 KILLED and two SURVIVED, with 355 per-plant raw logs,
zero NOT_APPLIED/MISMATCH/ERROR, and 24/24 candidate-source hashes restored.
Complete group logs and audits are included in the producer-evidence archive.
An initial pass-1 300:350 slice was interrupted at exit 130 and excluded; the
complete bounded replacement slices all exited 0.

### Revision 6 — physical walk depth and transitive callee guard

Pinned authority remains SPEC v0.7.0 §3.2, lines 806–813. The walk-length
axis now reaches the host's `unix.PathMax` physical bound. On Darwin,
`unix.PathMax` is 1024; the test creates 452 one-byte nested directories so
the runtime-root path is 1023 bytes and its deepest non-root ancestor is 1021
bytes. `TestCustodyAncestorWalkGeneratedDepthToPathMax` sets a real non-sticky
0777 mode at that deepest ancestor and at a middle ancestor, then invokes the
shared `checkCustodyAncestors` predicate. Both refusals assert literal
`tmux_unsafe_socket_path at socket ancestor`; all generated directories are
removed by reverse-order `t.Cleanup`.

The deep derived socket exceeds Darwin `sun_path`, so Probe, Spawn, and Execute
cannot physically reach this walk at that fixture depth. Instead,
`TestCustodyAncestorWalkReachableFromProductionEntries` parses the production
call graph and proves ServerProber.Probe, ServerSpawner.Spawn, and
Lifecycle.Execute each reach CheckSocketCustody and the exact shared
`checkCustodyAncestors` predicate.

The AST structural test discovers the transitive call graph mechanically
from `checkCustodyAncestors` across active tmuxserver/secprim production files.
The observed graph includes `custodyModeForPath`, `writableByOthers`,
`isFilesystemRoot`, `secprim.OpenNoFollowDir`, and its local callees. It
rejects path-derived length/depth comparisons, counter-bearing loops, and
counter updates. `secprim.memberErrorTarget` is discovered as a helper called
only under `failPath` detail construction, so its existing display truncation
cannot change the custody decision.

| Gate / axis | Generator and test | Structural argument or bound |
|---|---|---|
| `writableByOthers` / walk length | Path to `unix.PathMax-1`; deepest and middle non-root ancestors are each set to mode 0777. `TestCustodyAncestorWalkGeneratedDepthToPathMax`. | Unix pathname APIs cannot name a path longer than `PathMax-1`; the transitive AST guard rejects path-derived length/depth branches and counters in reachable in-module decision callees. |
| All entries / shared predicate wiring | `TestCustodyAncestorWalkReachableFromProductionEntries` checks Probe, Spawn, and Execute reach CheckSocketCustody and the shared ancestor predicate. | The `sun_path` bound prevents a physical-depth socket path at entries; the shared predicate is driven directly, with each production entry's wiring pinned. |

| Plant | Narrowing | Named failing test | Outcome |
|---|---|---|---|
| `N-ancestor-walk-callee-depth-cap-24` | `custodyModeForPath` clears write bits after 24 separators, admitting an unsafe ancestor outside rev5's generator | `TestCustodyAncestorWalkGeneratedDepthToPathMax` | KILLED twice, run alone; raw Go test exit 1 |
| `N-ancestor-walk-filesystem-root-depth-cap` | `isFilesystemRoot` stops ancestor checks after 24 separators | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | KILLED twice, run alone; raw Go test exit 1 |
| `C-control` | Harmless line-count-preserving comment | None expected | SURVIVED; harness control |

These are five rev6 plants, making 360 harness rows total. The complete
355-row rev5 harness passed twice; the full 360-row set was not rerun in rev6.
All five rev6 rows have two isolated raw runs each, and the neutral control
was applied and survived.

### Rev6 continuation — path-derived length guard

The transitive guard also carries path provenance through `strings.Split`,
range indices over path components, numeric component counts, and
`[]byte(path)`. This is exercised by three additional narrowing mutants so a
counter hidden behind a transformed path value is not accepted as covered.

| Plant | Narrowing | Named failing test | Result |
|---|---|---|---|
| N-ancestor-walk-split-component-count-cap | `len(strings.Split(path, "/")) > 24` admits the unsafe ancestor | TestCustodyAncestorWalkGeneratedDepthToPathMax and TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice alone; raw test exit 1 |
| N-ancestor-walk-range-index-cap | Range index over split components greater than 24 admits the unsafe ancestor | TestCustodyAncestorWalkGeneratedDepthToPathMax and TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice alone; raw test exit 1 |
| N-ancestor-walk-byte-length-cap | `len([]byte(path)) > 24` admits the unsafe ancestor | TestCustodyAncestorWalkGeneratedDepthToPathMax and TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice alone; raw test exit 1 |

The range-index first attempt failed at link time because the filesystem ran
out of space. It is excluded; two later isolated raw runs killed the narrowing.
The `callee24` and filesystem-root cap plants were rerun twice after the final
AST change. C-control was applied and survived. The complete 360-row harness
was not rerun.

Current revalidation note: all 45 configured verbose tests and coverage tests
passed in bounded package groups. Race tests passed in current groups for 44
packages; `sessquery` is reused from the prior exact rev6 evidence because no
source/test/dependency for that package changed. Its current rerun was
interrupted at the shell limit with exit 1 and no test assertion output. All
17 configured fuzz targets, `-count=3` custody tests, Windows vet, builds,
tracecheck, and diff-check passed; see the updated task results and attached
current-run evidence index for real exits.
