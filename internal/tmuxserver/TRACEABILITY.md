# TRACEABILITY — internal/tmuxserver (AX v0.7.0)

Normative scope: AX v0.7.0 Section 3.2 (platform paths: the
dedicated socket `<runtime>/tmux/ax.sock` under the Runtime IPC root;
the socket is runtime IPC, never durable identity), Section 4.2 (tmux
backend: dedicated -S server, no ambient/default discovery or reuse,
credential-sensitive creation, background broker-or-refuse with typed
capability_unavailable and no fallback to direct creation, launchctl
managername as diagnostic hint only, cached observations never
authorize resume), Section 4.C (operation vocabulary cited for the
wrapper entrypoint shape only), Section 4.D (capability evidence cited
landed), and Section 4.E (sensitive runtime state is owner-only and
non-replicable). Story: production-tmux-backend. The first leaf drives rows 1-58;
TASK-260830-1c28dz drives rows 59-92 (the §4.2 production adapters,
B18/B19/B20 bind step, and §4.C lifecycle operations); TASK-260922-vcx6yo
composes overlap admission on the same attach entry; TASK-260830-g0pcnt
drives story-final rows 93-97 for lost-response replay, reconnect/multi-attach
composition, socket substitution custody, ownership-neutral attach, and the
fresh-decoy foreground path.

Rule: every row names its production entry point. A row is driven only
when the named committed test executes that entry. Prose in place of
the ratio is not evidence. 97 of 97 AC rows driven: rows 1-58 first
leaf (row 55 extends the rows 19-22 override class with the fifth
ambient member, TMUX_TMPDIR; row 56 binds the Section 3.2 path and
runtime-root clause; rows 57-58 pin the commit-failure and
whitespace-override arms); rows 59-92 second leaf (adapters, bind
step, eight lifecycle operations, receipt discipline, state memory,
after-restore composition); rows 93-97 story-final leaf (lost-response
replay, overlap/reconnect composition, socket substitution custody,
ownership-neutral attach, and fresh-decoy foreground admission).
No row is a bound — the stated bounds below cover deliberately
unclaimed surfaces, not undriven rows.

No test in this package starts a tmux process: every broker, server,
and spawner interaction runs through injected dependencies, and the
suite passes with no tmux installed. No row has a real-tmux witness;
that is the determinism design, not a gap (bound B1).

Attestation authority: the sentinel/smoke sufficiency, signature
verification, liveness, and expiry verdict is the landed
internal/terminalbackend Reconcile/ResolveEvidence admission,
consumed here as an Admitted set composed only with the generation
equality the wrapper owner (internal/axpane checkRealm) already
enforces. This leaf decides membership plus generation binding and
re-decides nothing the admission proves.

## Runtime custody (Sections 4.2, 4.E)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 1 | Ensure creates the owner-only 0700 directory, explicitly under a restrictive umask | EnsureRuntimeDir (runtime.go, runtime_unix.go) | TestEnsureRuntimeDirCreatesOwnerOnly, TestEnsureRuntimeDirEnforcesModeUnderRestrictiveUmask |
| 2 | Re-ensure is idempotent and verifies mode | EnsureRuntimeDir (runtime.go, runtime_unix.go) | TestEnsureRuntimeDirIsIdempotent |
| 3 | Compliant directory verifies | VerifyRuntimeDir (runtime.go, runtime_unix.go) | TestVerifyRuntimeDirAdmitsCompliant |
| 4 | Widened (0755) and narrower (0600/0500) modes refuse on ensure and verify, never silently repaired | EnsureRuntimeDir, VerifyRuntimeDir (runtime_unix.go) | TestEnsureRuntimeDirRefusesWidenedMode, TestEnsureRuntimeDirRefusesNarrowerMode |
| 5 | Non-directory leaf refuses on commit and verify and through both Acquire entries; neither open carries O_NONBLOCK by decision — FIFO refusal relies on O_DIRECTORY failing fast, pinned by the commit-side and verify-side deadline tests | EnsureRuntimeDir, VerifyRuntimeDir (runtime_unix.go), Acquire (acquire.go, foreground + background) | TestEnsureRuntimeDirRefusesNonDirectoryLeaf (0600, 0700, FIFO under deadline), TestVerifyRuntimeDirRefusesNonDirectoryLeaf (regular file, FIFO under deadline), TestAcquireForegroundRefusesNonDirectoryLeaf, TestAcquireBackgroundRefusesNonDirectoryLeaf, TestAcquireBackgroundRefusesSymlinkLeaf, TestAcquireBackgroundRefusesHandleMismatch |
| 6 | Symlink leaf refuses on commit and verify. On darwin the verify-side ELOOP arm is unreachable — openat(O_NOFOLLOW&#124;O_DIRECTORY) on a symlink leaf returns ENOTDIR — while on Linux the same open returns ELOOP, mapped to containment; reasoned, not run, since the configured suite is darwin-only | EnsureRuntimeDir, VerifyRuntimeDir (runtime_unix.go) | TestEnsureRuntimeDirRefusesSymlinkLeaf, TestVerifyRuntimeDirRefusesSymlinkLeaf |
| 7 | Foreign ownership refuses on ensure and verify | EnsureRuntimeDir, VerifyRuntimeDir (ownership_unix.go) | TestEnsureRuntimeDirRefusesForeignOwnership |
| 8 | Handle/path mismatch refuses | EnsureRuntimeDir (runtime.go) | TestEnsureRuntimeDirRefusesHandleMismatch |
| 9 | Symlinked root refuses on commit, verify, and through the background entry | EnsureRuntimeDir, VerifyRuntimeDir (runtime_unix.go), Acquire (acquire.go, background) | TestEnsureRuntimeDirRefusesSymlinkRoot, TestVerifyRuntimeDirRefusesSymlinkRoot, TestAcquireBackgroundRefusesSymlinkRoot |
| 10 | Relative, empty, parent, and missing roots refuse invalid (runtime root), the same code verify reports (row 13); the foreground entry refuses a bad root before any side effect | EnsureRuntimeDir (runtime.go, runtime_unix.go), Acquire (acquire.go, foreground) | TestEnsureRuntimeDirRefusesBadRoots (4 subtests), TestAcquireForegroundRefusesBadRoot (relative, missing) |
| 11 | Empty, dot, parent, nested, absolute, and NUL names refuse | EnsureRuntimeDir (runtime.go) | TestEnsureRuntimeDirRefusesBadNames (6 subtests) |
| 12 | Unknown platform refuses on commit, verify, and through Acquire on both callers | EnsureRuntimeDir, VerifyRuntimeDir (runtime.go), Acquire (acquire.go) | TestEnsureRuntimeDirRefusesBadPlatform, TestVerifyRuntimeDirRefusesBadLexical (unknown platform), TestAcquireRefusesUnknownPlatform (foreground, background) |
| 13 | Verify refuses absence (runtime absent) and a missing root (runtime root, aligned with the commit side via the shared classifier) without creating | VerifyRuntimeDir (runtime.go, runtime_unix.go) | TestVerifyRuntimeDirRefusesAbsence, TestVerifyRuntimeDirRefusesBadLexical, TestVerifyRuntimeDirRefusesMissingRoot |
| 14 | Linux and WSL2 share the unix custody discipline | EnsureRuntimeDir (runtime_unix.go) | TestLinuxAndWSL2ShareUnixCustody |
| 15 | Windows platform refuses on every host on commit, verify, and through Acquire on both callers; no tmux directory exists on Windows | EnsureRuntimeDir, VerifyRuntimeDir (runtime.go), Acquire (acquire.go) | TestEnsureRuntimeDirRefusesWindowsOnUnixHost (commit and verify legs), TestAcquireRefusesWindowsPlatform (foreground, background) |
| 57 | Commit-phase filesystem failure (unwritable root) refuses the commit detail instead of being mistaken for an existing leaf | EnsureRuntimeDir (runtime_unix.go) | TestEnsureRuntimeDirRefusesReadOnlyRoot |

## Socket path and runtime root (Section 3.2)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 56 | Dedicated socket is `<runtime>/tmux/ax.sock` under the Runtime IPC root (macOS: per-user temporary directory; Linux/WSL2: `$XDG_RUNTIME_DIR/ax`), resolved by the landed localstore layout as PathRuntime and supplied by the caller; the socket is runtime IPC, never durable identity. The runtime leaf is not an Acquire input — the entry always passes RuntimeDirName, so the outcome socket is `<Root>/tmux/ax.sock` on both caller paths. The parent's verifiable custody (owned, 0700, component-verified without following a symlink) is rows 1-15; root creation custody is the localstore layout owner's (B19) and provenance is unclaimed (B20) | SocketPath (socket.go), Acquire (acquire.go) | TestSocketPathDerivesOnlyFromRuntimeDir, TestAcquireOutcomeSocketEqualsSpecPath (foreground, background) |

## Dedicated socket and ambient non-use (Section 4.2)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 16 | Socket derives only from the runtime directory, as `<root>/tmux/ax.sock` with both leaves asserted as spec literals; the Acquire outcome socket equals `<Root>/tmux/ax.sock` on both caller paths | SocketPath (socket.go), Acquire (acquire.go) | TestSocketPathDerivesOnlyFromRuntimeDir, TestAcquireOutcomeSocketEqualsSpecPath (foreground, background) |
| 17 | Resolution returns the derived socket under hostile ambient on all five members | ResolveSocket (socket.go) | TestResolveSocketDerivesIgnoringAmbient |
| 18 | Ambient observation records without using, including nil lookup | ObserveAmbient (socket.go) | TestObserveAmbientRecordsWithoutUsing, TestObserveAmbientNilLookupKeepsOutOfBand |
| 19 | TMUX environment value as override refuses through Acquire on both callers with no spawn and no directory | Acquire (acquire.go) via ResolveSocket (socket.go) | TestAcquireRefusesEveryAmbientOverrideVector/environment_variable (foreground, background) |
| 20 | Inherited socket as override refuses through Acquire on both callers with no spawn and no directory | Acquire (acquire.go) via ResolveSocket (socket.go) | TestAcquireRefusesEveryAmbientOverrideVector/inherited_socket (foreground, background) |
| 21 | Default path as override refuses through Acquire on both callers with no spawn and no directory | Acquire (acquire.go) via ResolveSocket (socket.go) | TestAcquireRefusesEveryAmbientOverrideVector/default_path (foreground, background) |
| 22 | Conventional name as override refuses through Acquire on both callers with no spawn and no directory | Acquire (acquire.go) via ResolveSocket (socket.go) | TestAcquireRefusesEveryAmbientOverrideVector/conventional_name (foreground, background) |
| 23 | Ambient value colliding with the derived socket refuses on every member, byte-identical and unclean (`/./`, `//`) spellings alike, and the TMUX member in the real `<socket>,<pid>,<index>` encoding, through Acquire on both callers and at ResolveSocket directly. Only TMUXEnv and InheritedSocket can name the derived socket in real tmux semantics (TMUX_TMPDIR is a directory; DefaultPath/ConventionalName end in `default`, never `ax.sock`) — the other three members pin defence-in-depth shapes, not live vectors. Roots containing a comma are outside the TMUX member's encoding: the first-comma split is faithful to tmux(1) itself, so a comma-root socket component walks past the gate (bound B16) | Acquire (acquire.go) via ResolveSocket (socket.go) | TestAcquireRefusesAmbientCollision (5 members × foreground/background), TestResolveSocketRefusesAmbientCollision (5 subtests), TestResolveSocketRefusesAmbientCollisionUncleanSpellings (2 subtests), TestAcquireRefusesAmbientCollisionUncleanSpelling (foreground, background), TestResolveSocketRefusesTMUXEnvRealEncoding, TestAcquireRefusesTMUXEnvCollisionRealEncoding (foreground, background) |
| 55 | TMUX_TMPDIR value as override refuses through Acquire on both callers with no spawn and no directory | Acquire (acquire.go) via ResolveSocket (socket.go) | TestAcquireRefusesEveryAmbientOverrideVector/tmux_tmpdir (foreground, background) |
| 58 | Whitespace-only override refuses through Acquire on both callers with no spawn and no directory | Acquire (acquire.go) via ResolveSocket (socket.go) | TestAcquireRefusesEveryAmbientOverrideVector/whitespace_only (foreground, background), TestResolveSocketRefusesEveryOverrideVector/whitespace_only |

## Background broker-or-refuse (Section 4.2)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 24 | Authenticated same-user broker plus generation-bound attested server contacts on macOS, with no spawn and no argv | Acquire (acquire.go) via CheckBrokerContact (readiness.go) | TestAcquireBackgroundContactsBroker |
| 25 | No broker returns the typed capability_unavailable with no spawn | Acquire (acquire.go) | TestAcquireBackgroundMissReturnsTypedUnavailable/no_broker |
| 26 | Foreign-user broker returns the typed capability_unavailable with no spawn | Acquire (acquire.go) | TestAcquireBackgroundMissReturnsTypedUnavailable/foreign-user_broker |
| 27 | Unattested server returns the typed capability_unavailable with no spawn, including the zero-admission (nothing-reconciled) and decoy-only members, and every catalog-derived decoy alone and together | Acquire (acquire.go) | TestAcquireBackgroundMissReturnsTypedUnavailable/unattested_server, .../zero_server_admission, .../decoy-only_server_admission, TestAcquireBackgroundRefusesEveryCatalogDecoy (15 members + all together) |
| 28 | Stale-generation admission returns the typed capability_unavailable with no spawn, including the admitted-without-generation member | Acquire (acquire.go) | TestAcquireBackgroundMissReturnsTypedUnavailable/stale_server_admission, .../admitted_server_without_generation, TestAcquireBackgroundMissReturnsTypedUnavailable/generation-unbound_principal |
| 29 | Linux and WSL2 misses also return the typed capability_unavailable with no spawn | Acquire (acquire.go) | TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback (linux, wsl2) |
| 30 | Broker probe failure passes through as unknown, never unavailable, with no spawn | Acquire (acquire.go) | TestAcquireBackgroundProbeFailurePassesThrough |
| 31 | Creation gate refuses the background caller; the miss path enforces it by call on all three tmux platforms (macOS, Linux, WSL2). Every background-path Acquire test (24/24 by mechanical census) arms both a spy spawner and a spy server probe, so a fallback through either foreground dependency is visible — including a fallback into acquireForeground under the shared-Dependencies shape (Dependencies.Spawn is inert on the background path: a nil spawner with a live broker still contacts, which is what makes the shared struct realistic) | checkCreationAllowed via Acquire (acquire.go) | TestCheckCreationAllowedRefusesBackground, TestAcquireBackgroundMissReturnsTypedUnavailable + TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback (kill paths of N-creation-literal, D-background-fallback, D-background-fallback-foreground) |

## Readiness non-authorization (Sections 4.2, 4.D)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 32 | A stale admission replayed on a broker miss still returns the typed capability_unavailable: cached proof authorizes nothing | Acquire (acquire.go) | TestAcquireBackgroundMissReturnsTypedUnavailable/stale_server_admission |
| 33 | A stale admission fakes no running attested server on the foreground path | Acquire (acquire.go) | TestAcquireForegroundRefusesWrongGenerationAdmission |
| 34 | Bound admitted row admits | CheckServerAttested (readiness.go) | TestCheckServerAttestedAdmitsBound |
| 35 | Missing realm row refuses, including a decoy capability, at the helper, at the broker contact, and through both Acquire paths; every non-realm Section 4.D member derived from the pinned catalog refuses alone and together at the helper and through the background entry | CheckServerAttested (readiness.go), CheckBrokerContact (readiness.go), Acquire (acquire.go) | TestCheckServerAttestedRefusesEachMissingConjunct (no admission rows, decoy capability only), TestCheckBrokerContactRefusesEachMissingFact (zero server admission, decoy-only server admission), TestAcquireForegroundRefusesRunningUnattested, TestAcquireBackgroundMissReturnsTypedUnavailable/zero_server_admission, .../decoy-only_server_admission, TestCheckServerAttestedRefusesEveryCatalogDecoy + TestAcquireBackgroundRefusesEveryCatalogDecoy (15 members + all together) |
| 36 | Wrong-generation admission refuses, including the both-empty pair, at the helper, at the broker contact, and through both Acquire paths | CheckServerAttested (readiness.go), CheckBrokerContact (readiness.go), Acquire (acquire.go) | TestCheckServerAttestedRefusesEachMissingConjunct (stale generation admission, both generations empty), TestCheckBrokerContactRefusesEachMissingFact (stale server admission, admitted server without generation), TestAcquireForegroundRefusesWrongGenerationAdmission, TestAcquireBackgroundMissReturnsTypedUnavailable/stale_server_admission, .../admitted_server_without_generation |
| 37 | Foreign-user broker principal refuses | CheckBrokerContact (readiness.go), Acquire (acquire.go) | TestCheckBrokerContactRefusesEachMissingFact (no broker, foreign-user broker), TestAcquireBackgroundMissReturnsTypedUnavailable/foreign-user_broker |
| 38 | Generation-unbound broker principal refuses, including the both-empty pair | CheckBrokerContact (readiness.go), Acquire (acquire.go) | TestCheckBrokerContactRefusesEachMissingFact (generation-unbound, empty and both-empty principal), TestAcquireBackgroundMissReturnsTypedUnavailable/generation-unbound_principal |
| 39 | Complete broker report admits | CheckBrokerContact (readiness.go) | TestCheckBrokerContactAdmitsLiveReport |

## Foreground acquisition (Section 4.2)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 40 | Absent server spawns dedicated with exact -S argv disjoint from every ambient vector and the ax pane tail, pinned element-by-element through the entry | Acquire (acquire.go) via BuildArgv (argv.go) | TestAcquireForegroundSpawnsDedicatedServer |
| 41 | Running attested server attaches with no spawn | Acquire (acquire.go) | TestAcquireForegroundAttachesToRunningAttested |
| 42 | Running unattested server refuses with no spawn and no repair, including a running server with zero admission rows and one with an empty generation | Acquire (acquire.go) | TestAcquireForegroundRefusesRunningUnattested, TestAcquireForegroundRefusesRunningWithEmptyAdmission, TestAcquireForegroundRefusesRunningWithEmptyGeneration |
| 43 | Server probe failure passes through as unknown with no spawn | Acquire (acquire.go) | TestAcquireForegroundProbeFailurePassesThrough |
| 44 | Spawn failure reports with exactly one attempt and no fallback | Acquire (acquire.go) | TestAcquireForegroundSpawnFailureReportsWithoutFallback |
| 45 | Malformed session refuses at the foreground entry before any side effect: no directory, no probe, no spawn (BuildArgv re-validates at the spawn site) | Acquire (acquire.go), BuildArgv (argv.go) | TestAcquireForegroundRefusesMalformedSession |

## Spawn argv (Sections 4.2, 4.C)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 46 | Spawn vector is exactly tmux -S socket new-session -d -s session ax pane session | BuildArgv (argv.go) | TestBuildArgvAddressesOnlyTheDerivedSocket |
| 47 | All five ambient sockets refuse | BuildArgv (argv.go) | TestBuildArgvRefusesAmbientSocket (5 subtests) |
| 48 | Malformed sessions refuse the landed UUIDv7 wiring | BuildArgv (argv.go) | TestBuildArgvRefusesMalformedSession |

## Input validation (Sections 4.2, 15.3)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 49 | Unknown caller refuses, with the foreground post-admission spy set armed (an admitted unknown caller always dispatches foreground) | Acquire (acquire.go) | TestAcquireRefusesUnknownCaller |
| 50 | Missing generation, broker state, and remediation each refuse on both callers; a present-but-invalid detail (invalid-UTF-8 generation) refuses after the broker probe | Acquire (acquire.go) | TestAcquireRefusesMissingRealmDetails (3 members × foreground/background), TestAcquireBackgroundInvalidGenerationRefusesAfterProbe |
| 51 | Missing dependencies refuse on every nil shape | Acquire (acquire.go) | TestAcquireRefusesMissingDependencies, TestAcquireBackgroundNilBrokerWithSpawnRefuses, TestAcquireForegroundNilSpawnWithProbeRefuses |
| 52 | Foreground acquisition ensures the runtime directory idempotently; background verifies without creating — an absent leaf on macOS, Linux, and WSL2 returns the typed capability_unavailable, a missing root refuses invalid, an unsafe leaf keeps its custody code (top-level *Error, never the unavailable wrapper) on the mode, ownership, and containment members alike | Acquire (acquire.go) via EnsureRuntimeDir/VerifyRuntimeDir (runtime.go) | TestAcquireCreatesRuntimeDirIdempotently, TestAcquireBackgroundVerifiesWithoutCreating (macos, linux, wsl2), TestAcquireBackgroundMissingRootRefusesInvalid, TestAcquireBackgroundStagesVerifyCustodyRefusal (mode), TestAcquireBackgroundStagesOwnershipRefusal (ownership), TestAcquireBackgroundRefusesNonDirectoryLeaf (containment), TestAcquireBackgroundRefusesSymlinkLeaf, TestAcquireBackgroundRefusesSymlinkRoot, TestAcquireBackgroundRefusesHandleMismatch |

## Crash and idempotency (Sections 4.2, 4.E)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 53 | Real SIGKILL between mkdir and fsync leaves a retry that converges to the one verified directory | EnsureRuntimeDir (runtime_unix.go) | TestEnsureCrashChildSelfTerminates |
| 54 | Spawn-then-retry attaches to the one server with exactly one spawn | Acquire (acquire.go) | TestAcquireForegroundSpawnIsIdempotentAcrossRetry |

## Lifecycle adapters (Section 4.2, second leaf)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 59 | The exec adapter runs -S vectors with AX-owned argv only: exits and both streams report as data, TMUX/TMUX_TMPDIR scrub from the child, malformed argv refuses, transport failures and signal deaths pass through as errors and prove nothing, and NUL-containing arguments refuse before any process starts. | OSRunner (exec.go) | TestOSRunnerReportsExitAsData, TestOSRunnerCapturesStderr, TestOSRunnerSignalDeathProvesNothing, TestOSRunnerScrubsAmbientInChild, TestOSRunnerRefusesBadArgv, TestOSRunnerRefusesNulArgument, TestOSRunnerTransportErrorProvesNothing, TestScrubTmuxEnv |
| 60 | The primitive vocabulary is exactly the ten tmux directives and the per-operation map covers exactly the eight lifecycle operations; unknown primitives and non-lifecycle operations refuse. | ParseDirective, DirectivesFor | TestParseDirectiveAdmitsTenPrimitives, TestParseDirectiveRefusesUnknown, TestDirectivesForMapsEightOperations, TestDirectivesForRefusesNonLifecycle |
| 61 | Every built vector is the fixed -S shape for its directive: the socket is pinned to the runtime leaf (ambient sockets refuse), targets parse as UUIDv7, the quiescence channel parses as UUIDv7, the attach vector states its input authorization (read-only emits -r, unstated refuses), the boundary vector is the bare blocking wait, and detach-client addresses the session. | BuildCommand | TestBuildCommandVectors, TestBuildCommandAttachReadOnlyVector, TestBuildCommandAttachRequiresInputFlag, TestBuildCommandRefusesAmbientSocket, TestBuildCommandRefusesBadIdentity |
| 62 | Dialing the socket reports live, stale, or unknown: refused and missing sockets are stale, a missing directory and every other failure is unknown, never absent. | UnixDialer.Dial | TestUnixDialerLiveStaleUnknown, TestUnixDialerMissingSocketIsStale, TestUnixDialerMissingDirectoryIsUnknown |
| 63 | The prober enforces dependencies, length, and custody before dialing: stale reports not-running, unknown errors, live attests through the admission, and admission failures pass through. A writable root refuses at Probe like at the spawner. | ServerProber.Probe | TestServerProberRefusesWithoutDependencies, TestServerProberMapsDialOutcomes, TestServerProberAdmissionFailurePassesThrough, TestServerProberEnforcesLengthAndCustody, TestServerProberRefusesWritableRoot |
| 64 | The broker prober reports the miss, admission reconciliation fails closed without a verifier, and the Production constructor wires the prober, spawner, dialer, and admission into the Dependencies. | probe.go | TestBrokerProberReportsMiss, TestReconcileAdmissionFailsClosedWithoutVerifier, TestProductionWiresDependencies |
| 65 | The spawner enforces length, custody, and the exec-site socket pin before spawning: a stale socket unlinks (hook-observed), a non-socket refuses, and exit/transport outcomes map to the spawn verdict, and a world-writable root refuses before any spawn vector runs. | ServerSpawner.Spawn | TestServerSpawnerRefusesWithoutRunner, TestServerSpawnerPinAndCustody, TestServerSpawnerUnlinksStaleSocket, TestServerSpawnerRefusesNonSocketFile, TestServerSpawnerMapsRunnerOutcome, TestServerSpawnerRefusesAbsentLeaf, TestServerSpawnerRefusesWritableRoot |

## Socket bind step (Sections 3.2, 4.2; B18, B19, B20)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 66 | B18: the socket path refuses at or beyond the platform sun_path limit (104/108), counted in bytes. | CheckSocketLength | TestCheckSocketLengthBounds, TestCheckSocketLengthCountsBytes |
| 67 | B19/B20: the socket verifies at exactly the runtime leaf, and the leaf custody delegates to the landed Verify with its refusal propagated. | CheckSocketCustody | TestCheckSocketCustodyAdmitsCompliant, TestCheckSocketCustodyRefusesMisplacedSocket, TestCheckSocketCustodyPropagatesLeafRefusal |
| 68 | B20: the runtime root verifies as a caller-owned directory unwritable by others through the landed no-follow open, with the open handle bound back to its path. | checkCustodyRoot | TestCheckSocketCustodyRefusesUnsafeRoot, TestCheckSocketCustodyRefusesForeignRoot, TestCheckSocketCustodyIdentityMismatchRefuses, TestCheckSocketCustodyNoCacheAfterChange |
| 69 | B20: every ancestor verifies as a non-symlink directory unwritable by others up the lexical chain; sticky directories admit, symlinks and files refuse through the no-follow opens. | checkCustodyAncestors | TestCheckSocketCustodyRefusesWritableAncestor, TestCheckSocketCustodyAdmitsStickyAncestor, TestCheckSocketAncestorKindArm, TestCheckSocketCustodyRefusesIntermediateSymlink, TestSocketCustodySymlinkRefusesAtProbeAndSpawn |
| 70 | B19: the socket path itself is bindable when absent and unlinkable when a socket; any other kind refuses so the stale-unlink step can never remove a non-socket. | checkCustodySocket | TestCheckSocketCustodyAdmitsCompliant, TestCheckSocketCustodyRefusesNonSocketFile |

## Lifecycle bodies (Section 4.C)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 71 | Create bodies are the closed six-member shape with the operation context, binding, bootstrap, entrypoint, transport, and interactive members. | parseCreateBody | TestParseCreateBodyValid, TestParseCreateBodyMemberSet, TestParseCreateBodyMembers |
| 72 | Attach bodies are the closed eleven-member shape: identity (session, instance, backend, versions, generation), client, transport, input flag, deadline, and the ownership-neutral authorization, which carries no lease member. | parseAttachBody | TestParseAttachBodyValid, TestParseAttachBodyMemberSet, TestParseAttachBodyMembers |
| 73 | Terminate bodies are the closed four-member shape with the stale lease, stale epoch, and evidence members. | parseTerminateBody | TestParseTerminateBodyValid, TestParseTerminateBodyViolations |
| 74 | Restore bodies are the closed four-member shape with the prior binding, checkpoint, and bootstrap members. | parseRestoreBody | TestParseRestoreBodyValid, TestParseRestoreBodyViolations |

## Lifecycle dispatch (Sections 4, 4.D)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 75 | Execute admits all eight lifecycle operations through the dependency, vocabulary, scope, length, and custody pre-gates; each pre-gate refuses before dispatch on every operation, with per-operation custody twins for root, ancestor, socket-kind, and leaf-mode arms. | Lifecycle.Execute | TestExecuteRefusesMissingDependencies, TestExecuteRefusesNonLifecycleScope, TestExecuteRefusesLongSocket, TestExecuteRefusesUnsafeSocket, TestExecuteCustodyEntryRoots, TestExecuteCreateRefusesWritableRoot, TestExecuteAttachRefusesWritableRootValidBody, TestExecuteEachOperationRefusesWritableRoot, TestExecuteEachOperationRefusesWritableAncestor, TestExecuteEachOperationRefusesNonSocketKind, TestExecuteEachOperationRefusesBadLeafMode, TestExecuteCustodyNarrownessTwins |
| 76 | Every operation verifies the ax.tmux backend identity before dispatch; foreign backends refuse on all eight operations without exec. | checkLifecycleBackend | TestExecuteRefusesForeignBackend |
| 77 | Out-of-row error codes normalize to the protocol error; in-row codes propagate with their report, on all eight rows. | rowError | TestRowErrorNormalizesOutsideSet, TestRowErrorCoversEightRows, TestLifecycleEmittedCodesAreRowAllowed |

## Lifecycle operations (Section 4.C)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 78 | Create runs the closed-shape admission, binding agreement, and relay refusal through the engine: interactive returns the bound descriptor, headless returns none, and identical requests replay one spawn. A fresh create rotates the incarnation, superseding prior closures; replays never rotate, and a failed rotation fails the operation closed. The wrapper is the path's only command: a rotation during its round trip changes nothing further (single-command probe). | executeCreate | TestExecuteCreateInteractive, TestExecuteCreateHeadless, TestExecuteCreateIdempotent, TestExecuteCreateRefusals, TestExecuteCreateBindingAgreement, TestExecuteCreateReplayPreservesBarrier, TestExecuteCreateIncarnationWriteFailureFailsClosed, TestEffectBoundaryAuditCreateIssuesSingleCommand |
| 79 | Attach returns the caller-executed vector with its descriptor and never execs an interactive attach. Admission checks closed shape, deadline, matching-transport capability, AX input authorization, the quiescing input barrier, live attestation, and the authoritative barrier recheck; generation and deadline are rechecked after waits before a durable receipt commit. Identical requests replay; read-only authorization emits a read-only vector and cannot reopen quiesced input. The peer census is fail-closed: corrupt scans, unreadable entries, receipt-shaped directories, misfiled receipts, and malformed receipt names refuse, while staging and non-receipt names skip. A receipt proves an admitted client claim, not a connected tmux client; without positive identity or detach evidence its liveness is UNKNOWN and it remains a possible peer. A persistent per-instance OS lock serializes separate Lifecycle instances/processes from before census through durable receipt commit. New overlapping clients require multi_attach; concurrent input additionally requires multiple_input_clients and valid AX policy. Receipt-validated same-client replays with a peer are exempt; conflicting retries refuse idempotency_mismatch. Lost state-record writes fail closed; lost outcome-report writes remain B43. The barrier verdict remains incarnation-scoped; stale memory, caller-carried source state, and failure results cannot reopen it. Admission and barrier waits honor deadline and cancellation with no late commit. | executeAttach | TestExecuteAttach, TestExecuteAttachRefusals, TestExecuteAttachIdempotent, TestAttachDescriptorNeverPersisted, TestExecuteAttachRefusesMeshWithoutCapability, TestExecuteAttachRefusesDeadline, TestExecuteAttachRefusesInputWhenQuiescing, TestExecuteReadOnlyAttachIsReadOnly, TestExecuteReadOnlyAttachPreservesQuiescedInput, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteAttachErrorsOnCorruptBarrierScan, TestExecuteAttachErrorsOnEmptyBarrierTime, TestReviewExistingActiveMemoryCannotReopenQuiescedInput, TestReviewFailedStopCannotEraseQuiescence, TestExecuteFailedStopPreservesQuiescingMemory, TestExecuteAdvisoryMemoryNeverDecidesAdmission, TestExecuteAttachErrorsOnEmptyBarrierKey, TestExecuteAttachErrorsOnStateReadFailure, TestAttachRefusesAfterQuiescenceWhenAdmittedLate, TestAttachQuiesceOrderingOverlapQuiesce, TestAttachQuiesceOrderingOverlapBoundary, TestExecuteAttachErrorsOnIncarnationReadFailure, TestExecuteAttachErrorsOnOutcomesDirReadFailure, TestExecuteAttachErrorsOnOutcomeFileReadFailure, TestAttachRefusesGenerationRotatedDuringLockWait, TestAttachRefusesDeadlineExpiredDuringAdmission, TestAttachRefusesDeadlineOnTheInstantDuringAdmission, TestAttachAuthorizationMembersRefuseWithoutReceipt, TestAttachOverlapRequiresMultiAttach, TestAttachOverlapInputRequiresMultipleInputClients, TestAttachSameClientRetryIgnoresOverlap, TestAttachOverlapPeerReadFailureFailsClosed, TestAttachPeerDirectoryFailsClosed, TestAttachSameClientRetryWithPeerPresent, TestAttachPeerFilenameMismatchFailsClosed, TestAttachWaitBoundedByOperationDeadline, TestAttachWaitCancelledReturnsPromptly, TestAttachConcurrentInputRequiresAXPolicy, TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer, TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances |
| 80 | Status classifies present, absent, and unknown observations against the AX-side memory: absence needs its stderr marker (unmarked, refused, and killed probes read unavailable), contradictions read unavailable, malformations refuse, and the provider triple evidences only when requested, capable, and observed. Status observes the recorded binding tuple: drifted queries non-match member by member, unbound sessions refuse unknown, tampered documents refuse integrity, negative backend counts refuse, and nonzero attached exits, foreign attached rows, unmarked panes exits, and malformed counts with plausible output read unavailable. The live probes run under the query deadline in-flight: a probe still blocked at the bound concludes unknown with no report, never absent. | executeStatus, ObserveStatus | TestExecuteStatusPresent, TestExecuteStatusAbsent, TestExecuteStatusPresentIncludesIdentity, TestExecuteStatusUnknownProbeIsUnavailable, TestExecuteStatusProviderConditional, TestExecuteStatusContradictions, TestClassifyStatusAbsentProbe, TestClassifyStatusPresentProbe, TestClassifyAbsenceMarkers, TestObserveStatusPresentParked, TestObserveStatusAbsentProbe, TestObserveStatusMemoryReconciles, TestObserveStatusUnknowns, TestProviderTripleRule, TestCutRowShapes, TestExecuteStatusNonMatchEachMember, TestExecuteStatusExactUnboundSessionRefusesUnknown, TestExecuteStatusBindingTamperRefusesIntegrity, TestObserveStatusSessionScopedObservesRecordedTuple, TestObserveStatusExactBackendDriftNonMatches, TestObserveStatusExactProtocolDriftNonMatches, TestExecuteStatusRefusesNegativeAttachedCount, TestExecuteStatusAttachedExitIsUnknown, TestExecuteStatusForeignAttachedRowIsUnknown, TestExecuteStatusUnknownPanesExitIsUnknown, TestExecuteStatusMalformedAttachedCountIsUnknown, TestExecuteStatusProbeHonorsOperationDeadline |
| 81 | Quiesce, boundary, and stop run through the engine with their outcome members: quiesce locks then detaches with no provider input, the boundary waits blocking with the proof kind bound into its evidence and replays its recorded time, and stop proves closure only from marked absence; identical requests replay. Corrupt, forged-key, and empty-time outcome reports refuse integrity instead of becoming fresh evidence; a failed barrier-state write fails the operation instead of reporting closure, and the retry heals the barrier with the original time. The stop escalation revalidates deadline, authorization, and generation after its poll before the kill issues — a graceful timeout escalates, any other stale fact refuses — the quiesce detach revalidates between the commands, the boundary tail is read-only, and the barrier waits honor the deadline on all three paths. The stop poll itself honors the operation deadline in-flight: a probe still blocked at the bound concludes unknown with no kill and no closure, never absence. The post-escalation re-confirmation observes under the operation deadline, so a cancellation-honouring executor still observes the closure after the graceful wait expires. | executeEngineOp | TestExecuteQuiesce, TestExecuteQuiesceIdempotent, TestExecuteQuiesceBarrierSequence, TestExecuteQuiesceEmitsNoProviderInput, TestExecuteQuiesceRefusesFailedClose, TestExecuteQuiesceReplayKeepsTimestamps, TestExecuteBoundary, TestExecuteBoundaryIdempotent, TestExecuteBoundaryWaitsForSignal, TestExecuteBoundaryBindsProofKind, TestExecuteBoundaryWaitTimeout, TestExecuteBoundaryPastDeadline, TestExecuteBoundaryRefusesFailedWait, TestExecuteBoundaryReplayKeepsProofAndTime, TestExecuteStop, TestExecuteStopIdempotent, TestExecuteStopUnknownProbeIsNotClosure, TestExecuteStopUnmarkedExitIsNotClosure, TestExecuteStopPollHonorsOperationDeadline, TestBoundWaitTakesLesser, TestLesserDeadlineTakesLesser, TestLifecycleIgnoresAmbientEnvironment, TestObserveBoundaryBindsProviderRows, TestOutcomeRecordCrashConverges, TestExecuteQuiesceCorruptOutcomeRefusesIntegrity, TestExecuteBoundaryCorruptOutcomeRefusesIntegrity, TestExecuteQuiesceForgedOutcomeKeyRefusesIntegrity, TestExecuteQuiesceEmptyRecordedTimeRefusesIntegrity, TestExecuteBoundaryEmptyRecordedTimeRefusesIntegrity, TestExecuteBoundaryRecordCrashRefusesIntegrity, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier, TestExecuteBoundaryBarrierSurvivesStateWriteFailure, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteStopEscalatesAfterGracefulTimeoutAlone, TestExecuteStopEscalationSucceedsWithContextHonoringRunner, TestExecuteQuiesceRefusesStaleFactsBetweenCommands, TestBackendPostWaitBoundaryRefusesWithoutRecheck, TestEffectBoundaryAuditBoundaryTailIsReadOnly, TestQuiesceBarrierWaitBoundedByOperationDeadline, TestBoundaryCommitWaitBoundedByOperationDeadline, TestQuiesceMalformedDeadlineRefusesBeforeBarrierWait |
| 82 | Terminate-stale runs the key, capability, transition, authorization, generation, fencing, target, winner, and deadline gates before the receipt-bound kill with its live-session confirm; an unproven confirm is unknown, never termination — including a confirm still blocked when the operation deadline fires, which concludes unknown in-flight with no termination evidence; the memory adopts stopped, identical requests replay, and effect failures resume through status. The re-confirm tail issues no command after its poll however the facts moved (no-post-wait-command probe). | executeTerminate | TestExecuteTerminate, TestExecuteTerminateRefusals, TestExecuteTerminateIdempotent, TestExecuteTerminateResumesAfterEffectFailure, TestExecuteTerminateUnknownConfirmIsNotTerminated, TestExecuteTerminateConfirmHonorsOperationDeadline, TestExecuteTerminateWinnerTracksCurrentLease, TestExecuteTerminateTargetLeaseIdentity, TestExecuteTerminateTargetSessionBinds, TestExecuteTerminateRefusesProviderOnlyCapability, TestExecuteTerminateRefusesOnDeadline, TestExecuteTerminateRedriveMismatch, TestExecuteTerminateRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditTerminateIssuesNoPostWaitCommand |
| 83 | Restore runs the capability, prior-binding, and receipt-bound persist-plus-wrapper sequence, then returns the required binding and the parked wire result; the after-restore branch decision (local resume, remote offer, or parked under the enforced refresh bound) is the wrapper composition (row 92, Table E). A substituted prior refuses the mismatch, a swapped document refuses the integrity failure, the memory adopts parked, identical requests replay, and rechecks resume through status. A fresh restore rotates the incarnation, superseding prior closures; replays never rotate, and a failed rotation fails the operation closed. A reboot restore validates its prior before the effects commit, then mints a successor generation persisted after the effects, with successor retries converging. The wrapper is the path's only command (single-command probe). | executeRestore | TestExecuteRestore, TestExecuteRestoreMismatch, TestExecuteRestoreIdempotent, TestExecuteRestoreReturnsBinding, TestExecuteRestoreRefusesSwappedBindingDoc, TestExecuteCreatePersistsBindingDocForRestore, TestExecuteCreateRefusesBindingDocFailure, TestExecuteResumeRefusesWhenStatusProvesOtherwise, TestExecuteRestoreResumesAfterRecheckFailure, TestExecuteRestoreRefusesWithoutRebootCapability, TestExecuteRestoreRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadlineEffect, TestExecuteRestoreRefusesGenerationDrift, TestExecuteRestoreAcrossServerGeneration, TestExecuteRestoreSuccessorRetryConverges, TestExecuteRestorePriorValidationPrecedesEffects, TestExecuteRestoreRefusesSubstitutedInstance, TestExecuteRestoreRefusesSubstitutedSession, TestExecuteRestoreRefusesSubstitutedBackend, TestExecuteQuiesceStopRestoreReopensBarrier, TestExecuteRestoreReplayPreservesBarrier, TestExecuteRestoreIncarnationWriteFailureFailsClosed, TestExecuteRestoreRefusesAuthorizationExpiredAfterReceipt, TestExecuteEntryRefusesWrongAuthorizationKind, TestEffectBoundaryAuditRestoreIssuesSingleCommand |
| 84 | The receipt discipline binds, replays, resumes, and completes under the idempotency key: tampered completions refuse member by member, corrupt images refuse, hook failures report uncertainty, and forged effect evidence fails the run pre-commit. | executeWithReceipt, resumeUncertain, runLifecycleEffects | TestExecuteTerminateReplayRefusesTamperedImage, TestExecuteTerminateReplayRefusesCorruptImage, TestExecuteTerminateBindHookFailure, TestRunLifecycleEffectsRefusesForgedEvidence |
| 85 | Deadlines are strict bounds at entry and before every effect: the on-deadline instant refuses, and the entry arm leaves the key unbound where the loop would bind it. The attach lock wait counts against the deadline: an operation that expires mid-wait refuses even while its authorization stays valid. The attach, quiesce, and boundary waits refuse at the deadline with no late commit, and cancellation returns promptly; the stop escalation answers to the operation deadline, distinct from the graceful wait. The read-only probes honor the same bound in-flight: the stop poll, the terminate confirm, and the status probes conclude unknown at the deadline, never a verdict. | ops.go, executeAttach | TestExecuteTerminateRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadline, TestExecuteRestoreRefusesOnDeadlineEffect, TestExecuteAttachRefusesDeadline, TestAttachRefusesDeadlineExpiredDuringAdmission, TestAttachRefusesDeadlineOnTheInstantDuringAdmission, TestAttachWaitBoundedByOperationDeadline, TestAttachWaitCancelledReturnsPromptly, TestQuiesceBarrierWaitBoundedByOperationDeadline, TestBoundaryCommitWaitBoundedByOperationDeadline, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteStopPollHonorsOperationDeadline, TestExecuteTerminateConfirmHonorsOperationDeadline, TestExecuteStatusProbeHonorsOperationDeadline |
| 86 | Generations validate at entry and recheck before every effect: drift between entry and effect refuses under the stale binding with nothing committed. The stop escalation and the quiesce detach revalidate generation after their waits before the next destructive command. | ops.go | TestExecuteEntryRefusesStaleGeneration, TestExecuteRestoreRefusesGenerationDrift, TestAttachRefusesGenerationRotatedDuringLockWait, TestExecuteStopRefusesStaleFactsAfterPoll, TestExecuteQuiesceRefusesStaleFactsBetweenCommands |
| 87 | The presented idempotency key re-derives from the request inputs on both receipt operations; an underived key refuses before capability, receipt, or exec. | executeTerminate, executeRestore | TestExecuteRefusesWrongIdempotencyKey |
| 88 | The AX-side state memory records and looks up per-instance states with identity and value grammars: creating never records, foreign documents refuse, failures install a first echo or the honest unknown but never overwrite an authoritative record, and the memory is advisory for admission. Operation outcome reports and binding documents persist beside the states with key, identity, and shape grammars; closure reports bind the current incarnation, a create or restore success rotates it, and the attach barrier proves exactly from current-incarnation reports. | InstanceStates | TestInstanceStatesRoundTrip, TestOpenInstanceStatesRefusesEmptyRoot, TestInstanceStatesRefusesCreating, TestInstanceStatesRefusesBadIdentity, TestInstanceStatesRefusesUnknownState, TestInstanceStatesRefusesForeignDocument, TestRecordFailureLeavesPresenceReconciliation, TestInstanceStatesOutcomeRoundTrip, TestInstanceStatesBindingDocRoundTrip, TestInstanceStatesLookupOutcomeRefusesForgedKey, TestExecuteQuiesceStateWriteFailureRefusesWiredAttach, TestExecuteQuiesceRetryAfterStateWriteFailureHealsBarrier, TestExecuteBoundaryBarrierSurvivesStateWriteFailure, TestExecuteAttachErrorsOnCorruptBarrierScan, TestExecuteAttachErrorsOnEmptyBarrierTime, TestInstanceIncarnationRoundTrip, TestQuiesceBarrierIgnoresSupersededIncarnation |
| 92 | The after-restore composition refreshes the mesh lease under the measured bound before deciding through the landed wrapper owner (Table E): local wins launch or reattach through backend restore, remote winners offer attach or takeover with no backend effects even under a lapsed local grant, and every other case parks without effects; malformed mode, operation, or dependencies refuse before any decision; the executed restore binds to the decided session, bootstrap, instance, backend, generation, and winner as one composed authorization, expired answers never verify, and the bound holds against non-cooperative adapters. | ExecuteWrapperRestore | TestWrapperRestoreLocalWinResumes, TestWrapperRestoreReattachReplaysBackend, TestWrapperRestoreLapsedGrantRemoteOffer, TestWrapperRestoreParksWithoutEffects, TestWrapperRestoreRefusesMalformed, TestWrapperRestoreRefreshBoundMeasured, TestWrapperRestoreRefusesDivergentWinner, TestWrapperRestoreRefusesDivergentWinnerLease, TestWrapperRestoreRefusesDivergentWinnerEpoch, TestReviewWrapperBootstrapMustBindExecutedRestore, TestReviewWrapperInstanceMustBindExecutedRestore, TestWrapperRestoreRefusesDivergentSession, TestWrapperRestoreRefusesDivergentGeneration, TestWrapperRestoreRefusesDivergentBackend, TestWrapperRestoreExpiredRefreshParksUnverified, TestWrapperRestoreRefreshBoundEnforced, TestWrapperRestoreRefreshRaceRejectsLateAnswer |

## Lifecycle crash and ambient discipline (Sections 4.2, 4.C; B16)

| # | Clause | Call site | Test |
|---|--------|-----------|------|
| 89 | Every mutating operation is idempotent under retry and resumes uncertainty through status: second identical requests replay without new execs, uncertain receipts reconcile before resuming, and tampered replays refuse. | Lifecycle.Execute | TestExecuteCreateIdempotent, TestExecuteAttachIdempotent, TestExecuteQuiesceIdempotent, TestExecuteBoundaryIdempotent, TestExecuteStopIdempotent, TestExecuteTerminateIdempotent, TestExecuteRestoreIdempotent, TestExecuteTerminateResumesAfterEffectFailure, TestExecuteRestoreResumesAfterRecheckFailure, TestExecuteResumeRefusesWhenStatusProvesOtherwise |
| 90 | B16: lifecycle entries take no ambient input, so a nested invocation cannot collide through them; the child environment scrubs the ambient tmux variables, and the Acquire-level over-strict nested refusal stands unchanged as first-leaf-owned. | Lifecycle.Execute, scrubTmuxEnv | TestLifecycleIgnoresAmbientEnvironment, TestScrubTmuxEnv, TestOSRunnerScrubsAmbientInChild |
| 91 | Terminate-stale decides over the landed fencing authorization with local target and winner bindings: the fencing decision authorizes exactly the keyed stale lease, and only a winner that still wins authorizes. | executeTerminate, mapFencingError | TestExecuteTerminateRefusals, TestExecuteTerminateTargetLeaseIdentity, TestExecuteTerminateTargetSessionBinds, TestExecuteTerminateWinnerTracksCurrentLease |

## Mutation battery

`PYTHONDONTWRITEBYTECODE=1 python3 internal/tmuxserver/mutant_harness.py`:
73 narrowing rows KILLED, 2 supplementary additive rows KILLED
(D-background-fallback and D-background-fallback-foreground, labeled),
1 harmless control SURVIVED, on two full passes with identical verdicts
and one raw log per plant per run. Row N-deps-foreground kills by
panic: the admitted request calls the nil spawner and the panic fails
the test. Rows N-verify-absence-detail (background leg),
N-broker-empty-generation, and N-commit-eacces-as-exist kill by reroute
to a neighboring refusal, disclosed in their row notes.
N-absence-remap-widen kills through the top-level custody oracle:
under the errors.As helper the row survives (the reviewer's R05),
under the top-level assertion it kills. N-details-constructor kills by
typed-nil: the swallowed constructor failure returns a typed-nil
*axerror.Error inside a non-nil error interface where the test expects
the invalid-arguments refusal, and the top-level type assertion
reddens. N-session-entry kills
through the no-directory/no-probe assertions while the refusal code
stays correct (the spawn-site arm still refuses the admitted member).
No token-preserving source-text mutant applies: no gate inspects source
text, and the harness executes the behavioral suite (bound B4). The
reviewers' error-arm plants (stat-failure arms flipped to fail-open,
the unreachable guard arm, the unopenable-root arm) are reported as
error-arm rows under bound B15, never counted among the narrowing
rows.

Second leaf: 218 narrowing rows KILLED net (221 added over the
73-row checkpoint harness, 3 reclassified to supplementary: 2
wiring in revision 7, 1 shadowed in revision 8), 4 supplementary
rows added (2 wiring, 1 rotation, 1 shadowed), 1 harmless control
SURVIVED (shared), on the full pass with one raw log per plant —
301 rows total: 294 narrowing KILLED, 5 supplementary KILLED, 1
supplementary SURVIVED (shadowed), 1 control SURVIVED.
Rows N-lifecycle-entry-deadline-terminate and -restore kill by the
no-bind assertion (the loop recheck would refuse the same verdict);
row N-entry-generation patches two sites and kills through both
subtests, attribution verified per subtest from the raw log. Rows
N-bind-mismatch-map, N-replay-corrupt-image, N-stop-escalation-kill,
N-bodies-entrypoint-string, and N-bind-hook-found kill by reroute to
a neighboring verdict, disclosed in their row notes. Multi-entry rows
(cmdvector-attach, vector-detached, the status contradiction rows,
directive-unknown, both scrub rows, length-boundary, present-stopped,
boundary-timeout, verify-absence-detail, boundary-blocking,
readonly-vector, quiesce-detach, unmarked-exit, alias-skip,
boundexit, wrapper-reorder-refresh, the EachOperation custody
rows, the three shared within-effect arms, the shared wait bound,
the shared probe bound)
kill through every claimed entry, attribution verified
per test from the raw logs and, for the shared arms, from isolated
single-test runs. The six custody-entry rows share one
killer test but kill through one subtest each (single-entry rows);
the attach row additionally kills through its valid-body test. The
shared within-effect arm rows kill through the stop and quiesce
killers while the other-fact subtests stay green; the shared wait
row kills through the attach, quiesce, and boundary deadline tests
while the cancellation companion stays green; the shared probe
row kills through the stop, terminate-confirm, and status
deadline tests; the per-site skip
rows kill through their own entry while the other members stay
green. Row D-attach-entry-deadline-shadowed survives by reroute:
the wait bound refuses the admitted instant with the same verdict
before any admission runs, so the entry arm is fail-fast
redundancy pinned behaviorally, not a narrowing. Singleton refusal
arms (relay transport, nil guards, stop-timeout, failed-kill
self-confirm, unconfigured post-wait revalidation) ship no row:
admitting the singleton is deleting the gate (bound B22); the
outcome-lookup empty-key arm left this set by narrowing
(N-outcome-key-forged). The engine normalizes backend status error
details, so the cutrow and empty-panes narrowings measure at the
helper while the dispatch legs pin the code (bound B23).

## Gate × entry census

Every gate in the package against every production entry that could
reach it. Columns: AFG = Acquire foreground, ABG = Acquire
background, ENS = EnsureRuntimeDir, VER = VerifyRuntimeDir, ARG =
BuildArgv, SOCK = SocketPath, RES = ResolveSocket, OBS =
ObserveAmbient, BRO = CheckBrokerContact, SRV = CheckServerAttested,
CRE = checkCreationAllowed called directly. Each cell states exactly
one of: M (measured — a named committed test drives the gate through
that entry and a shipped admitting narrowing row kills through that
test; multi-entry rows list every entry their killers cover, verified
per entry from the raw logs), U1 (entry takes no input this gate
decides over), U2 (entry never reaches this code, with the structural
reason), U3 (shadowed — an upstream arm refuses every such input
through this entry first), B (stated bound with the reason). 50 of
264 cells measured, 4 bounds, 210 unreachable with reasons; every
reachable non-bound cell is measured (50 of 54).

| Gate (site) | AFG | ABG | ENS | VER | ARG | SOCK | RES | OBS | BRO | SRV | CRE |
|---|---|---|---|---|---|---|---|---|---|---|---|
| caller (acquire.go:149) | M TestAcquireRefusesUnknownCaller / N-caller-vocabulary | U2 unknown callers refuse pre-dispatch; admission always dispatches foreground | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U2 vocabulary enforced upstream in Acquire; direct callers pass post-gate values |
| details (acquire.go:152) | M TestAcquireRefusesMissingRealmDetails foreground / N-details-generation, N-details-brokerstate, N-details-remediation | M TestAcquireRefusesMissingRealmDetails background / same 3 rows | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| render (acquire.go:289) | U2 foreground never renders the unavailable refusal | M TestAcquireBackgroundInvalidGenerationRefusesAfterProbe / N-details-constructor | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| dispatch (acquire.go:163) | M TestAcquireForegroundSpawnsDedicatedServer / N-dispatch-foreground | M TestAcquireBackgroundMissReturnsTypedUnavailable / N-dispatch-background | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| plat (runtime.go:76) | M TestAcquireRefusesUnknownPlatform + TestAcquireRefusesWindowsPlatform foreground / N-platform-wiring + N-platform-windows | M same tests background / same rows | M TestEnsureRuntimeDirRefusesBadPlatform + TestEnsureRuntimeDirRefusesWindowsOnUnixHost / same rows | M TestVerifyRuntimeDirRefusesBadLexical + WindowsOnUnixHost verify leg / same rows | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| root (runtime.go:87) | M TestAcquireForegroundRefusesBadRoot / N-root-wiring (relative member; missing member asserted here, row-pinned via ABG/ENS/VER) | M TestAcquireBackgroundMissingRootRefusesInvalid / N-root-missing-classify (missing member; relative member row-pinned via AFG/ENS/VER) | M TestEnsureRuntimeDirRefusesBadRoots / N-root-wiring + N-root-missing-classify | M TestVerifyRuntimeDirRefusesBadLexical + TestVerifyRuntimeDirRefusesMissingRoot / same rows | U1 applies no root grammar to its directory input | U1 | U1 | U1 | U1 | U1 | U1 |
| name (runtime.go:110) | U2 name is not an Acquire input; entry always passes RuntimeDirName (see wiring) | U2 name is not an Acquire input; entry always passes RuntimeDirName (see wiring) | M TestEnsureRuntimeDirRefusesBadNames / N-containment-slash (nested member; 5 other members test-pinned) | M TestVerifyRuntimeDirRefusesBadLexical nested / N-containment-slash | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| guard (runtime.go:93) | B B15 guard refusal arm unreachable through the name grammar; fail-open plant survives by construction | B B15 same | B B15 same | B B15 same | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| wiring (acquire.go:155,183,228) | M TestAcquireOutcomeSocketEqualsSpecPath foreground / N-wiring-lexical + N-wiring-ensure (verify site not reached from AFG) | M TestAcquireOutcomeSocketEqualsSpecPath background / N-wiring-lexical + N-wiring-verify | U2 leaf is a parameter; fixed-leaf choice lives in Acquire | U2 leaf is a parameter; fixed-leaf choice lives in Acquire | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| override (socket.go:85) | M TestAcquireRefusesEveryAmbientOverrideVector foreground ×6 / N-override-env, -inherited, -default, -conventional, -tmpdir, -whitespace | M same background ×6 / same 6 rows | U1 | U1 | U1 | U1 | M TestResolveSocketRefusesEveryOverrideVector ×6 / same 6 rows | U1 | U1 | U1 | U1 |
| collision (socket.go:106) | M TestAcquireRefusesAmbientCollision foreground ×5 + UncleanSpelling + TMUXEnvRealEncoding foreground / N-collision ×5 + N-collision-unclean + N-collision-tmuxenv-encoding | M same background / same 7 rows | U1 | U1 | U2 compares the spawn socket under G-argv, not ambient members | U1 | M direct collision tables / same 7 rows | U1 | U1 | U1 | U1 |
| deps (acquire.go:180,222) | M TestAcquireRefusesMissingDependencies FG + TestAcquireForegroundNilSpawnWithProbeRefuses / N-deps-foreground | M TestAcquireRefusesMissingDependencies BG + TestAcquireBackgroundNilBrokerWithSpawnRefuses / N-deps-background | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| session (acquire.go:225, argv.go:20) | M entry arm TestAcquireForegroundRefusesMalformedSession / N-session-entry; spawn-site arm U3 shadowed (entry refuses first; measured at ARG) | U2 background never reads SessionID | U1 | U1 | M spawn-site arm TestBuildArgvRefusesMalformedSession / N-argv-session | U1 | U1 | U1 | U1 | U1 | U1 |
| commit (runtime_unix.go:35) | M wiring + widened member TestAcquireForegroundStagesCommitPhaseCustodyRefusal + TestAcquireCreatesRuntimeDirIdempotently / N-acquire-commit-refusal + non-directory leaf TestAcquireForegroundRefusesNonDirectoryLeaf / N-containment-odirectory-commit (remaining members pinned at ENS over shared commit code) | U2 background verifies, never ensures | M commit tests / N-mode-create, N-mode-narrower-create, N-fchmod-umask, N-ownership-uid, N-containment-mismatch, N-containment-symlink, N-containment-odirectory-commit, N-root-symlink-commit, N-root-wiring, N-root-missing-classify, N-commit-eacces-as-exist | U2 Verify never commits | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| verify (runtime_unix.go:106) | U2 foreground ensures; Ensure never calls Verify | M BG custody tests (mode, ownership, non-dir, symlink leaf/root, mismatch, absence, missing root) / N-acquire-verify-refusal, N-absence-remap-ownership, N-containment-odirectory-verify, N-containment-symlink-verify, N-containment-mismatch, N-root-symlink-verify, N-verify-absence-detail, N-background-absence-remap, N-root-missing-classify | U2 Ensure never calls Verify | M verify tests / N-mode-verify, N-mode-narrower-verify, N-ownership-uid, N-containment-mismatch, N-containment-symlink-verify, N-containment-odirectory-verify, N-root-symlink-verify, N-verify-absence-detail, N-root-missing-classify | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| absence (acquire.go:184,258) | U2 Ensure creates; absence never surfaces on the creating path | M VerifiesWithoutCreating ×3 / N-background-absence-remap + N-verify-absence-detail; mode / N-absence-remap-widen; ownership / N-absence-remap-ownership; containment / N-absence-remap-containment; missing root / N-absence-remap-invalid | U2 | M TestVerifyRuntimeDirRefusesAbsence / N-verify-absence-detail (detail production; remap lives in Acquire) | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| creation (acquire.go:277) | U2 foreground creation allowed; path never consults the gate (deliberate, no dead call) | M miss tables 8 shapes + linux + wsl2 / N-creation-literal + D-background-fallback + D-background-fallback-foreground (24-test spy census arms both foreground dependencies) | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 | M TestCheckCreationAllowedRefusesBackground / N-creation-literal |
| broker (readiness.go:80) | U2 foreground decides the server admission directly, never a broker principal | M miss-table no-broker, foreign-user, generation-unbound / N-broker-uid + N-broker-generation (both-empty pair U3: details gate refuses empty Generation first) | U2 | U2 | U2 | U2 | U2 | U2 | M TestCheckBrokerContactRefusesEachMissingFact / N-broker-uid + N-broker-generation + N-broker-empty-generation | U2 decides the server admission only, no principal | U2 |
| broker-server (readiness.go:86) | U2 foreground never decides through a broker report | M miss-table server subtests ×5 / N-broker-zero-admission (R10/R19 members test-pinned, anchor shared — replay-verified) | U2 | U2 | U2 | U2 | U2 | U2 | M broker-contact server subtests ×5 / N-broker-zero-admission | U2 wiring lives in CheckBrokerContact; CheckServerAttested is downstream | U2 |
| server (readiness.go:40) | M running tests ×4 / N-realm-membership + N-realm-generation + N-unattested-foreground (both-empty U3: details gate precedes) | M miss-table server subtests ×5 + catalog-decoy table / N-realm-membership + N-realm-generation + N-broker-zero-admission (admitted-without-gen test-pinned, R10 anchor shared — replay-verified; the 15 catalog decoys are test-pinned, row N-realm-membership keeps its headless_creation member) | U2 | U2 | U2 | U2 | U2 | U2 | M broker-contact server subtests ×5 / N-realm-membership + N-realm-generation (both-empty server pair U3: principal arm precedes) | M TestCheckServerAttestedRefusesEachMissingConjunct ×6 + catalog-decoy table / N-realm-membership + N-realm-generation + N-realm-empty-want | U2 |
| running (acquire.go:236) | M running tests + probe-failure / N-running-empty-admission + N-running-empty-generation + N-unattested-foreground | U2 background decides the broker report, never a Running flag | U2 | U2 | U2 | U2 | U2 | U2 | U2 BRO/SRV decide admissions without a Running flag | U2 | U2 |
| probe (acquire.go:190,232) | M TestAcquireForegroundProbeFailurePassesThrough / N-probe-swallow-foreground | M TestAcquireBackgroundProbeFailurePassesThrough / N-probe-swallow-background | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| spawn (acquire.go:246) | M TestAcquireForegroundSpawnFailureReportsWithoutFallback + TestAcquireForegroundSpawnIsIdempotentAcrossRetry / N-spawn-swallow | U2 no spawn call on the background path (structural); no-fallback proof is the spy census + D rows at creation × ABG | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 | U2 |
| argv (argv.go:17,24) | M TestAcquireForegroundSpawnsDedicatedServer full element compare / N-argv-detached + N-argv-tail (socket/session arms U3: socket always derived, session pre-validated; CheckArgv shape arm B12: unreachable through Acquire) | U2 background never builds spawn argv | U2 | U2 | M exact-vector + ambient-socket + invalid-shape + malformed-session tests / N-argv-socket + N-argv-tmuxenv + N-argv-shape + N-argv-session + N-argv-detached + N-argv-tail | U2 | U2 | U2 | U2 | U2 | U2 |

SOCK (SocketPath) is a pure join and OBS (ObserveAmbient) records
without deciding: neither enforces any gate, so their columns read U
throughout by construction — the wiring gate pins what callers pass
to the join. CRE decides only G-creation; every other CRE cell is U
because the direct entry answers only "may this caller create" for
post-gate caller values.

## Second-leaf census

New columns: EXC/EXA/EXS/EXE/EXT/EXR = Execute create, attach,
status, engine ops (quiesce/boundary/stop), terminate-stale, restore;
PRB = ServerProber.Probe, SPW = ServerSpawner.Spawn, DIAL =
UnixDialer.Dial, CMD = BuildCommand, DIRV = DirectivesFor, PDIR =
ParseDirective, LEN = CheckSocketLength, CUS = CheckSocketCustody,
REC/LOOK = InstanceStates Record/Lookup (including the RecordOutcome/RecordBindingDoc/RecordIncarnation writes and the LookupOutcome/LookupBindingDoc/LookupIncarnation reads), CLS = ClassifyStatus, OBSP =
ObserveStatus called directly, PFX = PerformEffect called directly,
OSR = OSRunner called directly, UNT = unexported internals called
directly in unit tests (body parsers, rowError, providerTriple,
cutRow, ReconcileAdmission). Cell codes M/U1/U2/U3/B as above, with
U2 reasons tagged: U2a = entry never calls this code (the retired
U2b, reachable-but-undriven, is bound B39 now: those cells state
what the committed tests drive, with the narrowed direction
measured at the listed entry); U2c = reachable only through
caller-injected
Production Dependencies with no committed composition test (bound
B27).

Table B: old gates × new entries (24 × 21 = 504 cells; 10 M, 494
U2a). Only the verify side (via custody delegation) and the server
attestation (via attach) are reachable from new entries; every other
old gate reads U2a throughout because no new code calls it (new code
never ensures, never lexically gates, never reads ambient, never
remaps, never consults creation, and never calls BuildArgv,
CheckBrokerContact, or the Acquire call sites).

| Gate (site) | EXC | EXA | EXS | EXE | EXT | EXR | PRB | SPW | DIAL | CMD | DIRV | PDIR | LEN | CUS | REC | LOOK | CLS | OBSP | PFX | OSR | UNT |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| caller (acquire.go:149) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| details (acquire.go:152) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| render (acquire.go:289) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| dispatch (acquire.go:163) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| plat (runtime.go:76) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| root (runtime.go:87) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| name (runtime.go:110) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| guard (runtime.go:93) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| wiring (acquire.go:155,183,228) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| override (socket.go:85) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| collision (socket.go:106) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| deps (acquire.go:180,222) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| session (acquire.go:225, argv.go:20) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| commit (runtime_unix.go:35) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| verify (runtime_unix.go:106) | M UnsafeSocket/create / N-verify-absence-detail | M UnsafeSocket/attach / same row | M UnsafeSocket/status / same row | M UnsafeSocket/quiesce-input,boundary,stop / same row | M UnsafeSocket/terminate-stale / same row | M UnsafeSocket/restore / same row | M EnforcesLengthAndCustody / same row | M RefusesAbsentLeaf / same row | U2a | U2a | U2a | U2a | U2a | M PropagatesLeafRefusal / same row | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| absence (acquire.go:184,258) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| creation (acquire.go:277) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| broker (readiness.go:80) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| broker-server (readiness.go:86) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| server (readiness.go:40) | U2a | M AttachRefusals fresh-decoy,stale-admission / N-attach-attestation-decoy | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| running (acquire.go:236) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| probe (acquire.go:190,232) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| spawn (acquire.go:246) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |
| argv (argv.go:17,24) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a |

Table C: new gates × old entries (152 × 11 = 1672 cells; 1654 U2a,
18 U2c). Table C derives its gate set from Table D: the eight
revision-5 gates (barrierincarnation, barrierkey, attachhealth,
failurerecord, createincarnation, restoreincarnation,
incarnationrecord, incarnationlookup), the three revision-6
gates (recheck, ordering, barrierread), the four revision-8
gates (effectrecheck, overlapmulti, overlapinput, waitbound), and
the four revision-9 gates (peershape, overlapreplay,
confirmbound, statusbound) and the one revision-10 gate
(escalationbound) add 220 U2a cells, because old code
calls none of them. The three TASK-260922-vcx6yo gates
(attachaxpolicy, admissionlock, livenessunknown) add 33 U2a cells.
Old code
predates new code and calls
none of it, except
through caller-injected Dependencies: AFG reaches the adapter and
bind gates below only when the caller injects the Production
implementations, and no committed test composes Production into
Acquire (bound B27), so those 18 cells read U2c. ABG reaches no new
gate (background never probes, spawns, or binds). The U2c cells are
AFG × {dialstale, probedeps, probeunknown, spawnpin, spawnshape,
spawnexit, spawnnil, unlinkkind, reconcilenil, sunpath, placement,
leafdelegate, leafwiring, rootwrite, rootowner, ancestorwrite,
ancestorkind, socketkind}; every other Table C cell is U2a.

Table D: new gates × new entries (152 × 21 = 3192 cells; 210 M, 187
B, 2795 U; counts verified by script over the table). Test names
drop the common Test prefix; "same N rows" means the row list in the
gate's first M cell.

| Gate (site) | EXC | EXA | EXS | EXE | EXT | EXR | PRB | SPW | DIAL | CMD | DIRV | PDIR | LEN | CUS | REC | LOOK | CLS | OBSP | PFX | OSR | UNT |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| directive (exec.go:104) | B39 post-gate vectors; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | B39 post-gate; M at PDIR,CMD | U1 | U1 | U1 | M RefusesUnknownDirective / N-exec-directive-unknown | U1 | M ParseDirectiveRefusesUnknown / same row | U1 | U1 | U1 | U1 | U1 | U1 | B39 engine effects valid; M at PDIR,CMD | U1 | U1 |
| cmdsocket (exec.go:158) | B39 derived sockets; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | B39 derived; M at CMD | U1 | U1 | U1 | M RefusesAmbientSocket / N-exec-ambient-socket | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 derived; M at CMD | U1 | U1 |
| cmdident (exec.go:200,213,226) | B39 parsed identities; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | B39 parsed; M at CMD | U1 | U1 | U1 | M RefusesBadIdentity / N-exec-instance-identity, N-exec-session-identity, N-exec-quiescence-identity | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 parsed; M at CMD | U1 | U1 |
| scrub (exec.go:69) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M ScrubsAmbientInChild / N-exec-scrub-tmux, N-exec-scrub-tmpdir | M ScrubTmuxEnv / same 2 rows |
| cmdvector (exec.go:158) | M CreateInteractive / N-exec-vector-detached | M Attach / N-cmdvector-attach | U2a status builds no vectors | M BoundaryWaitsForSignal / N-boundary-blocking-vector; quiesce/stop subcommand-only (M at CMD) | B39 kill/has subcommands; M at CMD | M Restore / N-exec-vector-detached | U1 | U1 | U1 | M Vectors / N-exec-vector-detached, N-cmdvector-attach, N-boundary-blocking-vector, N-cmdvector-detach | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 subcommand-only; M at CMD | U1 | U1 |
| osrun (exec.go:40) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M SignalDeathProvesNothing / N-osrunner-signal-death; other mappings B30 total | U1 |
| directives (backend.go:30) | M RefusesNonLifecycleScope / N-exec-directives-manifest | M same test+row; pre-gate op-independent | M same test+row; pre-gate | M same test+row; pre-gate | M same test+row; pre-gate | M same test+row; pre-gate | U1 | U1 | U1 | U1 | M DirectivesForRefusesNonLifecycle / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| effectvocab (backend.go:100) | B39 landed effects; M at PFX | B39 landed; M at PFX | U1 | B39 landed; M at PFX | B39 landed; M at PFX | B39 landed; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesUnknownVocabulary / N-backend-effect-vocabulary | U1 | U1 |
| exitrefused (backend.go:261) | B39 execs succeed; M at PFX | U1 attach never execs | U1 | B39 execs succeed; M at PFX | B39 kill confirms inline; M at PFX | B39 execs succeed; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesFailedExec / N-backend-exit-refused | U1 | U1 |
| transport (backend.go:261) | B39 runners scripted; M at PFX | U1 | U1 | B39 valid ctx; M at PFX | B39 scripted; M at PFX | B39 scripted; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M TransportProvesNothing / N-backend-transport-unknown | U1 | U1 |
| reconfirm (backend.go:224) | U1 | U1 | U1 | U1 | B39 single-effect failures; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M TerminateLiveRefuses, ReconfirmRefusesFailedKill / N-backend-reconfirm-failed-kill | U1 | U1 |
| boundarywait (backend.go:286) | U1 | U1 | U1 | M BoundaryWaitTimeout / N-backend-boundary-timeout | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BoundaryTimeout / same row | U1 | U1 |
| escalation (backend.go:224) | U1 | U1 | U1 | B39 clean closes; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M EscalationKillFailure / N-stop-escalation-kill | U1 | U1 |
| persistmap (backend.go:132) | B39 bindings succeed; M at PFX | U1 | U1 | U1 | U1 | B39 priors mismatch first; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BindingConflict / N-persist-conflict-map | U1 | U1 |
| attachmap (backend.go:158) | U1 | B39 receipts replay; M at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AttachClient / N-attach-conflict-map | U1 | U1 |
| selfconfirm (backend.go:195) | U1 | U1 | U1 | U1 | B22 ResumesAfterEffectFailure; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 TerminateLiveRefuses; singleton | U1 | U1 |
| stoptimeout (backend.go:224) | U1 | U1 | U1 | B39 closes clean; M nowhere (B22 at PFX) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 StopTimeout; singleton | U1 | U1 |
| cmdmap (backend.go:107) | B39 parsed identities; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | B39 parsed; B28 at PFX | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B28 untested mapping; builder rows at CMD | U1 | U1 |
| storenil (backend.go:132,158) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 NilStoresRefuse; singleton | U1 | U1 |
| staterecord (state.go:74) | B39 adopted states; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | B39 adopted; M at REC | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RoundTrip, RefusesCreating, RefusesUnknownState / N-state-creating, N-state-unknown | U1 | U1 | U1 | U1 | U1 | U1 |
| statelookup (state.go:104) | U2a | M AttachErrorsOnStateReadFailure / N-state-read-absent (value ignored; error propagates) | B39 parsed instance; M at LOOK | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesBadIdentity, RefusesForeignDocument / N-state-lookup-identity, N-state-foreign | U1 | U1 | U1 | U1 | U1 |
| stateopen (state.go:60) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 RefusesEmptyRoot; singleton | U1 | U1 | U1 | U1 | U1 | U1 |
| createcount (bodies.go:64) | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M MemberSet / N-bodies-create-count |
| attachcount (bodies.go:152) | U1 | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M MemberSet / N-bodies-attach-count |
| termcount (bodies.go:241) | U1 | U1 | U1 | U1 | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Violations / N-bodies-terminate-count |
| restorecount (bodies.go:304) | U1 | U1 | U1 | U1 | U1 | B39 valid bodies; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Violations / N-bodies-restore-count |
| transportvocab (bodies.go:340) | B39 valid transports; M at UNT | B39 valid; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Create/AttachMembers / N-bodies-transport |
| epochzero (bodies.go:265) | U1 | U1 | U1 | U1 | B39 valid epochs; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Violations/epoch-zero / N-bodies-epoch-zero |
| entrypoint (bodies.go:110) | B39 valid entrypoints; M at UNT | B39 valid; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M StringRefuses, shapes pinned / N-bodies-entrypoint-string |
| absentmemory (status.go:296) | U1 | U1 | M Contradictions/quiescing-memory / N-status-absent-quiescing | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AbsentProbe/memory-quiescing / same row | B39 live shapes; M at CLS,EXS | U1 | U1 | U1 |
| presentmemory (status.go:314) | U1 | U1 | M Contradictions/stopped-memory / N-status-present-stopped | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M PresentProbe/memory-stopped / same row | M MemoryReconciles / same row | U1 | U1 | U1 |
| attachable (status.go:334) | U1 | U1 | M Contradictions/incapable-attach / N-status-attachable | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M PresentProbe/no-attach-cap / same row | B39 quiescing never attachable; M at CLS,EXS | U1 | U1 | U1 |
| cutrow (status.go:160) | U1 | U1 | B23 count-less-row pins code; engine normalizes detail | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U2a classify takes parsed rows | M Unknowns / N-status-cutrow-count | U1 | U1 | M CutRowShapes / same row |
| emptypanes (status.go:96) | U1 | U1 | B23 empty-rows pins code; engine maps uncoded | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M Unknowns/empty-rows / N-status-empty-panes | U1 | U1 | U1 |
| triple (status.go:344) | U1 | U1 | M Contradictions/unrequested-present / N-provider-triple-unrequested | U1 | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 requested shapes; M at UNT,EXS | U1 | U1 | M TripleRule / same row |
| dialstale (probe.go:60) | U2a | U2a | U2a | U2a | U2a | U2a | B39 injected outcomes; M at DIAL | U1 | M LiveStaleUnknown, MissingSocketIsStale / N-probe-dial-refused, N-probe-dial-enoent | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| probedeps (probe.go:100) | U2a | U2a | U2a | U2a | U2a | U2a | M RefusesWithoutDependencies / N-probe-deps-dialer, N-probe-deps-admit | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| probeunknown (probe.go:112) | U2a | U2a | U2a | U2a | U2a | U2a | M MapsDialOutcomes/unknown / N-probe-unknown-passthrough | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnpin (probe.go:182) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | M PinAndCustody / N-spawn-pin | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnshape (probe.go:182) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | M PinAndCustody / N-spawn-shape | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnexit (probe.go:196) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | M MapsRunnerOutcome / N-spawn-exit | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| spawnnil (probe.go:177) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | B22 RefusesWithoutRunner; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| unlinkkind (probe.go:206) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U3 custody refuses non-sockets first | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| reconcilenil (probe.go:238) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 FailsClosedWithoutVerifier; singleton |
| sunpath (bind.go:28) | M AtLimitSocket/create / N-bind-length-boundary | M AtLimitSocket/attach / same row | M AtLimitSocket/status / same row | M AtLimitSocket/3 engine ops / same row | M AtLimitSocket/terminate-stale / same row | M AtLimitSocket/restore / same row | M EnforcesLengthAndCustody / same row | B39 socket arg short; M at LEN,EXS,PRB | U1 | U1 | U1 | U1 | M Bounds, CountsBytes / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| placement (bind.go:44) | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | U2a socket derived from RuntimeDir; entry input never reaches placement | M EnforcesLengthAndCustody / N-bind-placement | B39 pinned vectors; M at CUS,PRB | U1 | U1 | U1 | U1 | U1 | M RefusesMisplacedSocket / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| leafdelegate (bind.go:52) | B32 UnsafeSocket/create; propagation pinned | B32 UnsafeSocket/attach; pinned | B32 UnsafeSocket/status; pinned | B32 UnsafeSocket/3 ops; pinned | B32 UnsafeSocket/terminate; pinned | B32 UnsafeSocket/restore; pinned | B32 EnforcesLengthAndCustody; pinned | B32 RefusesAbsentLeaf; pinned | U1 | U1 | U1 | U1 | U1 | M PropagatesLeafRefusal / N-bind-wiring-verify | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| leafwiring (bind.go:44,52) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | M AdmitsCompliant / N-bind-wiring-placement, N-bind-wiring-verify | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| rootwrite (bind.go:96) | M CreateRefusesWritableRoot / N-custody-entry-root-create; EachOperationRefusesWritableRoot/create / N-bind-root-writable | M AttachRefusesWritableRootValidBody / N-attach-entry-root-custody; EachOperationRefusesWritableRoot/attach / N-bind-root-writable | M CustodyEntryRoots/status/root / N-custody-entry-root-status | M EachOperationRefusesWritableRoot/quiesce-input,boundary,stop / N-bind-root-writable | M EachOperationRefusesWritableRoot/terminate-stale / N-bind-root-writable | M CustodyEntryRoots/restore/root / N-custody-entry-root-restore | M ServerProberRefusesWritableRoot / N-prober-root-custody | M ServerSpawnerRefusesWritableRoot / N-spawn-root-custody | U1 | U1 | U1 | U1 | U1 | M RefusesUnsafeRoot / N-bind-root-writable | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| rootowner (bind.go:96) | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | B31 ForeignRoot stages leaf-first; arm untested+singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| ancestorwrite (bind.go:116) | M EachOperationRefusesWritableAncestor/create / N-bind-ancestor-writable | M EachOperationRefusesWritableAncestor/attach / N-bind-ancestor-writable | M CustodyEntryRoots/status/ancestor / N-custody-entry-ancestor-status | M EachOperationRefusesWritableAncestor/quiesce-input,boundary,stop / N-bind-ancestor-writable | M EachOperationRefusesWritableAncestor/terminate-stale / N-bind-ancestor-writable | M CustodyEntryRoots/restore/ancestor / N-custody-entry-ancestor-restore | B39 staged; M at CUS + 6 EX | B39 staged; M at CUS + 6 EX | U1 | U1 | U1 | U1 | U1 | M RefusesWritableAncestor / N-bind-ancestor-writable | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| ancestorkind (bind.go:116) | B39 staged compliant; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | B39 staged; M at CUS,PRB,SPW | M SymlinkRefusesAtProbeAndSpawn/probe, RefusesIntermediateSymlink / N-custody-alias-skip | M SymlinkRefusesAtProbeAndSpawn/spawn / same row | U1 | U1 | U1 | U1 | U1 | M RefusesIntermediateSymlink, AncestorKindArm / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| socketkind (bind.go:164) | M EachOperationRefusesNonSocketKind/create / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/attach / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/status / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/quiesce-input,boundary,stop / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/terminate-stale / N-bind-socket-kind | M EachOperationRefusesNonSocketKind/restore / N-bind-socket-kind | U2a staged; M at CUS + 6 EX | B39 absent sockets; M at CUS + 6 EX | U1 | U1 | U1 | U1 | U1 | M RefusesNonSocketFile / N-bind-socket-kind | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| backendid (lifecycle.go:200) | M RefusesForeignBackend/create / N-lifecycle-backend-identity | M RefusesForeignBackend/attach / same row | M RefusesForeignBackend/status / same row | M RefusesForeignBackend/3 engine ops / same row | M RefusesForeignBackend/terminate-stale / same row | M RefusesForeignBackend/restore / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| agreement (lifecycle.go:255) | M BindingAgreement / N-lifecycle-agreement-session, -instance, -backend, -generation | U2a create-only gate | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| rowvocab (lifecycle.go:270) | B39 in-row codes; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | B39 in-row; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M NormalizesOutsideSet, CoversEightRows / N-rowvocab-pass |
| enginemembers (lifecycle.go:380) | U1 | U1 | U1 | M Quiesce, Boundary, Stop / N-engineop-quiesce-generation, N-engineop-boundary-evidence, N-engineop-stop-process, N-engineop-stop-store | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| waitlesser (lifecycle.go:410) | U1 | U1 | U1 | B39 scripted waits; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BoundWaitTakesLesser / N-wait-bound-lesser |
| stoplesser (lifecycle.go:421) | U1 | U1 | U1 | B39 staged deadlines; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 staged; M at UNT | U1 | M LesserDeadlineTakesLesser / N-stop-deadline-lesser |
| relay (lifecycle.go:300) | B22 CreateRefusals/relay; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| descpresence (lifecycle.go:330) | B22 Interactive+Headless; boolean fork | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachcap (ops.go:100) | U1 | M wrong-transport-capability, MeshWithoutCapability / N-attach-capability-local, N-attach-capability-mesh | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachdeadline (ops.go:72) | U1 | M RefusesDeadline / D-attach-entry-deadline-shadowed survived-supplementary (entry fail-fast; enforced boundary at post-lock + wait); AttachRefusesDeadlineExpiredDuringAdmission, AttachRefusesDeadlineOnTheInstantDuringAdmission / N-attach-postlock-deadline; AttachWaitBoundedByOperationDeadline, AttachWaitCancelledReturnsPromptly / N-attach-wait-deadline shared | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attestation (ops.go:113) | U1 | M fresh-decoy / N-attach-attestation-decoy; stale member pinned | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachrecord (ops.go:140) | U1 | M Attach / N-attach-record | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachdesc (ops.go:46) | U1 | M Attach / N-attach-descriptor-order | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| createdesc (ops.go:56) | M CreateInteractive / N-create-descriptor-order | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| serveadmitnil (ops.go:108) | U1 | B22 no-admission; singleton | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attacheffect (ops.go:119) | U1 | B26 unstaged failure mapping | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachbuild (ops.go:128) | U1 | U3 body parse precedes; parsed identities never refuse | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| authkind (ops.go:196,352,520) | U1 | U1 | U1 | U1 | M ExecuteEntryRefusesWrongAuthorizationKind/terminate / N-entry-authkind-terminate; ExecuteTerminateRefusesAuthorizationExpiredAfterReceipt / N-lifecycle-loop-authorization shared; D-lifecycle-terminate-authkind supplementary wiring | M ExecuteEntryRefusesWrongAuthorizationKind/restore / N-entry-authkind-restore; ExecuteRestoreRefusesAuthorizationExpiredAfterReceipt / N-lifecycle-loop-authorization shared; D-lifecycle-restore-authkind supplementary wiring | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| generation (ops.go:205,361,524) | U1 | M AttachRefusesGenerationRotatedDuringLockWait / N-attach-postlock-generation | U1 | U1 | M EntryRefusesStaleGeneration/terminate / N-entry-generation | M EntryRefusesStaleGeneration/restore, RefusesGenerationDrift / N-entry-generation, N-lifecycle-loop-generation | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| deadline (ops.go:72,226,374,510) | U1 | M RefusesDeadline/boundary / D-attach-entry-deadline-shadowed survived-supplementary (entry fail-fast; enforced boundary at post-lock + wait); AttachRefusesDeadlineExpiredDuringAdmission, AttachRefusesDeadlineOnTheInstantDuringAdmission / N-attach-postlock-deadline; AttachWaitBoundedByOperationDeadline, AttachWaitCancelledReturnsPromptly / N-attach-wait-deadline shared | U1 | U1 | M RefusesOnDeadline / N-lifecycle-entry-deadline-terminate | M RefusesOnDeadline, RefusesOnDeadlineEffect / N-lifecycle-entry-deadline-restore, N-lifecycle-loop-deadline | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| keymaterial (ops.go:176,338) | U1 | U1 | U1 | U1 | M WrongIdempotencyKey/terminate / N-key-material-terminate | M WrongIdempotencyKey/restore / N-key-material-restore | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| capdual (ops.go:240) | U1 | U1 | U1 | U1 | M missing-capability, RefusesProviderOnlyCapability / N-lifecycle-capability-provider, N-lifecycle-capability-stale | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| restorecap (ops.go:345) | U1 | U1 | U1 | U1 | U1 | M RefusesWithoutRebootCapability / N-lifecycle-restore-capability | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| fencingmap (ops.go:262) | U1 | U1 | U1 | U1 | M no-force, malformed-presented / N-lifecycle-fencing-force, N-fencing-invalid-reroute; other members pinned | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| target (ops.go:270) | U1 | U1 | U1 | U1 | M target-mismatch, TargetLeaseIdentity, TargetSessionBinds / N-lifecycle-target-epoch, N-lifecycle-target-lease, N-lifecycle-target-session | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| winner (ops.go:285) | U1 | U1 | U1 | U1 | M winner-rotated, winner-lease-identity / N-lifecycle-winner-epoch, N-lifecycle-winner-lease; session arm U3 fenced | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| mismatch (ops.go:368) | U1 | U1 | U1 | U1 | U1 | M RestoreMismatch/wrong-digest / N-lifecycle-restore-mismatch | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindmap (ops.go:410) | U1 | U1 | U1 | U1 | M RedriveMismatch / N-bind-mismatch-map | B39 shared code; unstaged redrive on restore; M at EXT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindfound (ops.go:424) | U1 | U1 | U1 | U1 | M BindHookFailure / N-bind-hook-found | B39 shared code; unstaged hook failure on restore; M at EXT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindmiss (ops.go:424) | U1 | U1 | U1 | U1 | B25 clean-miss unstageable; hooks fire post-commit | B25 clean-miss unstageable | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| completedread (ops.go:440) | U1 | U1 | U1 | U1 | B24 read failure unstaged; no seam | B24 read failure unstaged | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| evidence (ops.go:548) | U1 | U1 | U1 | U1 | B39 real digests; M at UNT | B39 real digests; M at UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesForgedEvidence / N-lifecycle-evidence-digest |
| record (ops.go:505,592) | U1 | U2a attach records at its own site | U1 | U1 | M Terminate / N-lifecycle-record-success; failure arm untested here | M Restore / same row; failure arm untested here | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesForgedEvidence / N-lifecycle-record-failure |
| resume (ops.go:463) | U1 | U1 | U1 | U1 | B39 source-proven resumes; M at EXR | M RefusesWhenStatusProvesOtherwise / N-lifecycle-resume-state; error arm subsumed | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| replay (ops.go:407) | U1 | U1 | U1 | U1 | M TamperedImage, CorruptImage / N-lifecycle-replay-session, N-replay-corrupt-image; other members pinned | B39 clean replays; M at EXT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| completearms (ops.go:580) | U1 | U1 | U1 | U1 | U2a Complete runs post-Bind once; never fails here | U2a Complete runs post-Bind once | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| readonly (exec.go:attach) | U1 | M ReadOnlyAttachIsReadOnly / N-attach-readonly-vector | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AttachReadOnlyVector / same row | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| inputflag (exec.go:attach) | U1 | B39 stated authorizations; M at CMD | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M AttachRequiresInputFlag / N-attach-input-flag | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| detachvec (exec.go:detach) | U2a quiesce-only vector | U2a quiesce-only | U2a quiesce-only | B39 subcommand-only; M at CMD | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | M Vectors / N-cmdvector-detach | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 subcommand-only; M at CMD | U1 | U1 |
| boundarykind (backend.go:observeBoundary) | U2a boundary-only gate | U2a boundary-only | U2a boundary-only | M BoundaryBindsProofKind / N-boundary-proof-kind | U2a boundary-only | U2a boundary-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 kind-echoed; M at EXE | U1 | U1 |
| boundexit (backend.go:observeBoundary) | U2a boundary-only gate | U2a boundary-only | U2a boundary-only | M BoundaryRefusesFailedWait / N-boundary-exit-refused | U2a boundary-only | U2a boundary-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BoundaryRefusedExit / same row | U1 | U1 |
| quiescedetach (backend.go:closeInput) | U2a quiesce-only gate | U2a quiesce-only | U2a quiesce-only | M BarrierSequence / N-quiesce-detach-step | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M InputClosed / same row | U1 | U1 |
| nosendkeys (backend.go:closeInput) | U2a quiesce-only gate | U2a quiesce-only | U2a quiesce-only | M EmitsNoProviderInput / N-quiesce-sendkeys-absent | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 shared sequence; M at EXE | U1 | U1 |
| closeexit (backend.go:closeInput) | U2a quiesce-only gate | U2a quiesce-only | U2a quiesce-only | M RefusesFailedClose / N-quiesce-close-exit | U2a quiesce-only | U2a quiesce-only | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 clean closes; M at EXE | U1 | U1 |
| quiescegate (ops.go:checkAttachQuiesced) | U1 | M RefusesInputWhenQuiescing / N-quiesce-attach-gate (scoping, key, decode, and empty arms are separate rows below) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| probemarker (backend.go:classifyAbsence) | U2a no has/list probes | U2a no probes | M StatusUnknownProbe / N-probe-unmarked-exit | M StopUnmarkedExit / same row | M TerminateUnknownConfirm / same row | U2a no probes | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 shared code; M at EXS | B39 shared code; M at EXE,EXT | U1 | U1 |
| probeneg (backend.go:classifyAbsence) | U2a no has/list probes | U2a no probes | B39 non-negative scripts; M at EXE | M StopUnknownProbe / N-probe-negative-exit | B39 non-negative scripts; M at EXE | U2a no probes | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 non-negative scripts; M at EXE | U1 | U1 |
| identity (bind.go:checkCustodyRoot) | B39 matching identity; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | B39 matching; M at CUS | U1 | U1 | U1 | U1 | U1 | M IdentityMismatchRefuses / N-custody-identity-bind | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| bindcontract (ops.go:restoreBinding) | U1 | U1 | U1 | U1 | U1 | M ReturnsBinding, Idempotent / N-restore-binding-omit | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| restoreagree (ops.go:restoreBinding) | U1 | U1 | U1 | U1 | U1 | M RefusesSwappedBindingDoc / N-restore-agreement-session | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| replaytime (lifecycle.go:report) | U1 | U1 | U1 | M QuiesceReplayKeepsTimestamps, BoundaryReplayKeepsProofAndTime (+Idempotent) / N-replay-timestamp-quiesce, N-replay-timestamp-boundary | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| outcomekey (state.go:RecordOutcome) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M OutcomeRoundTrip / N-outcome-key-record | M LookupOutcomeRefusesForgedKey / N-outcome-key-forged | U1 | U1 | U1 | U1 | U1 |
| binddocid (state.go:RecordBindingDoc) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BindingDocRoundTrip / N-binddoc-session-identity | M BindingDocRoundTrip / N-binddoc-lookup-identity | U1 | U1 | U1 | U1 | U1 |
| binddocshape (state.go:RecordBindingDoc) | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M BindingDocRoundTrip / N-binddoc-document-shape | U1 | U1 | U1 | U1 | U1 | U1 |
| docpersist (backend.go:persistBinding) | B39 doc-kept; M at EXR | U1 | U1 | U1 | U1 | M CreatePersistsBindingDocForRestore / N-create-doc-persist | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 unstaged bindingDoc; M at EXR | U1 | U1 |
| pollbound (backend.go:460) | U1 | U1 | U1 | M ExecuteStopPollHonorsOperationDeadline / N-probe-deadline shared | B39 terminate tail reaches the shared poll; M at EXE | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 direct staging unmeasured here; M at EXE | U1 | U1 |
| attachreadonly (ops.go:134) | U2a attach-only gate | M ReadOnlyAttachPreservesQuiescedInput / N-attach-readonly-record | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| quiescereport (lifecycle.go:reportInputClosedAt) | U2a quiesce-only gate | U2a quiesce-only gate | U2a quiesce-only gate | M CorruptOutcomeRefusesIntegrity, EmptyRecordedTimeRefusesIntegrity, OutcomeRecordCrashConverges / N-quiesce-report-corrupt, N-quiesce-report-empty, N-quiesce-report-record (quiesce leg) | U2a quiesce-only gate | U2a quiesce-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| boundaryreport (lifecycle.go:reportBoundaryObservedAt) | U2a boundary-only gate | U2a boundary-only gate | U2a boundary-only gate | M CorruptOutcomeRefusesIntegrity, EmptyRecordedTimeRefusesIntegrity, RecordCrashRefusesIntegrity / N-boundary-report-corrupt, N-boundary-report-empty, N-boundary-report-record (boundary leg) | U2a boundary-only gate | U2a boundary-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| statusunbound (status.go:ObserveStatus) | U2a status-only gate | U2a status-only gate | M ExactUnboundSessionRefusesUnknown / N-status-exact-unbound | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| statusdoc (status.go:statusBindingDoc) | U2a status-only gate | U2a status-only gate | M BindingTamperRefusesIntegrity (missing, corrupt, foreign-session, foreign-instance, digest-mismatch) / N-status-doc-missing, N-status-parse-admit, N-status-agree-session, N-status-agree-instance | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| statusmatch (status.go:bindingMatchesQuery) | U2a status-only gate | U2a status-only gate | M NonMatchEachMember (instance, implementation, generation) / N-status-match-instance, N-status-match-impl, N-status-match-generation | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M ExactBackendDriftNonMatches, ExactProtocolDriftNonMatches / N-status-match-backend, N-status-match-proto | U1 | U1 | U1 | U1 | U1 |
| statusnegative (status.go:probeAttached) | U2a status-only gate | U2a status-only gate | M RefusesNegativeAttachedCount / N-status-negative-attached; MalformedAttachedCountIsUnknown / N-status-attached-parse | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| prioragree (ops.go:readRestorePrior) | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | M RefusesSubstitutedInstance, RefusesSubstitutedSession, RefusesSubstitutedBackend, RefusesSwappedBindingDoc / N-restore-prior-instance, N-restore-prior-backend, N-restore-agreement-session | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| succession (ops.go:restoreBinding) | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | M AcrossServerGeneration, SuccessorRetryConverges, PriorValidationPrecedesEffects / N-restore-succession-gen, N-restore-mint-generation, N-restore-successor-persist | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| execdeps (lifecycle.go:Execute) | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | B36 nil-lifecycle pinned; singleton arms | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| quiescerecord (lifecycle.go:executeEngineOp) | U2a quiesce-only gate | U2a quiesce-only gate | U2a quiesce-only gate | M QuiesceStateWriteFailureRefusesWiredAttach / N-quiesce-state-record | U2a quiesce-only gate | U2a quiesce-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| boundaryrecord (lifecycle.go:executeEngineOp) | U2a boundary-only gate | U2a boundary-only gate | U2a boundary-only gate | M BoundaryBarrierSurvivesStateWriteFailure / N-boundary-state-record | U2a boundary-only gate | U2a boundary-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierquiesce (state.go:QuiesceBarrierProven) | U2a attach-only gate | M QuiesceStateWriteFailureRefusesWiredAttach / N-attach-barrier-quiesce-recovery | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierboundary (state.go:QuiesceBarrierProven) | U2a attach-only gate | M BoundaryBarrierSurvivesStateWriteFailure / N-attach-barrier-boundary-recovery | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierdecode (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnCorruptBarrierScan / N-attach-barrier-corrupt-scan | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierempty (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnEmptyBarrierTime/quiesce,boundary / N-attach-barrier-empty-quiesce, N-attach-barrier-empty-boundary | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierincarnation (state.go:QuiesceBarrierProven) | U2a attach-only gate | M QuiesceStopRestoreReopensBarrier / N-attach-barrier-incarnation | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierkey (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnEmptyBarrierKey / N-attach-barrier-empty-key | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachhealth (ops.go:checkAttachQuiesced) | U1 | M AttachErrorsOnStateReadFailure / N-attach-health-check | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| failurerecord (lifecycle.go:recordFailure) | B39 first-echo only; M at EXE,UNT | U1 | U1 | M FailedStopCannotEraseQuiescence / N-failure-preserves-record | B39 echo-or-unknown; M at UNT | B39 echo-or-unknown; M at EXE,UNT | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RunLifecycleEffectsRefusesForgedEvidence / N-lifecycle-record-failure |
| createincarnation (lifecycle.go:executeCreate) | M CreateIncarnationWriteFailureFailsClosed / N-create-incarnation-record, N-create-replay-skip | U2a create-only gate | U2a create-only gate | U2a create-only gate | U2a create-only gate | U2a create-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| restoreincarnation (ops.go:runLifecycleEffects) | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | U2a restore-only gate | M ExecuteTerminate / N-terminate-no-rotation | M RestoreIncarnationWriteFailureFailsClosed / N-restore-incarnation-record; replay skip B41 structural with D-restore-replay-rotation | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| incarnationrecord (state.go:RecordIncarnation) | B39 adopted rotations; M at REC | U2a | U2a | U2a | U2a | B39 adopted rotations; M at REC | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M IncarnationRoundTrip / N-incarnation-record-identity, N-incarnation-record-key | U1 | U1 | U1 | U1 | U1 | U1 |
| incarnationlookup (state.go:LookupIncarnation) | U2a | M AttachErrorsOnIncarnationReadFailure / N-incarnation-read-absent | U2a | B39 closure binds current; M at LOOK | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M IncarnationRoundTrip / N-incarnation-lookup-identity, N-incarnation-foreign, N-incarnation-empty-key | U1 | U1 | U1 | U1 | U1 |
| attachedexit (status.go:probeAttached) | U2a status-only gate | U2a status-only gate | M AttachedExitIsUnknown / N-status-attached-exit | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| foreignrow (status.go:probeAttached) | U2a status-only gate | U2a status-only gate | M ForeignAttachedRowIsUnknown / N-status-foreign-row | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| panesexit (status.go:probePanes) | U2a status-only gate | U2a status-only gate | M UnknownPanesExitIsUnknown / N-status-panes-unknown-exit | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| argvnul (exec.go:OSRunner.Run) | U2a | U2a | U2a | U2a | U2a | U2a | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | M RefusesNulArgument / N-osrunner-argv-nul | U1 |
| recheck (ops.go:executeAttach) | U2a attach-only gate | M AttachRefusesAfterQuiescenceWhenAdmittedLate, AttachQuiesceOrderingOverlapQuiesce / N-attach-barrier-recheck | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| ordering (lifecycle.go:barrierMu, ops.go:executeAttach) | U2a no barrier commit on this path | M AttachQuiesceOrderingOverlapQuiesce / N-ordering-lock-attach | U2a no barrier commit on this path | M AttachQuiesceOrderingOverlapQuiesce, AttachQuiesceOrderingOverlapBoundary / N-ordering-lock-quiesce, N-ordering-lock-boundary | U2a no barrier commit on this path | U2a no barrier commit on this path | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| barrierread (state.go:QuiesceBarrierProven) | U2a attach-only gate | M AttachErrorsOnOutcomesDirReadFailure, AttachErrorsOnOutcomeFileReadFailure / N-outcomes-dir-read-absent, N-outcome-file-read-skip | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| effectrecheck (lifecycle.go:727; backend.go:closeInput,confirmClosed) | U2a create effects are single commands with no post-wait boundary; recheck never configured | U2a attach commits via the store with no post-wait destructive command; recheck never configured | U2a status has no side effects | M ExecuteStopRefusesStaleFactsAfterPoll, ExecuteStopEscalatesAfterGracefulTimeoutAlone, ExecuteQuiesceRefusesStaleFactsBetweenCommands / N-stop-escalation-generation, N-stop-escalation-expiry, N-stop-escalation-deadline shared 3 arms + N-stop-escalation-skip, N-quiesce-midsequence-skip per-site | U2a terminate tail issues no command after its poll; recheck never configured; the no-post-wait-command probe pins it | U2a restore effects are single commands with no post-wait boundary; recheck never configured | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B22 BackendPostWaitBoundaryRefusesWithoutRecheck; singleton | U1 | U1 |
| overlapmulti (ops.go:299; termbind/overlap.go:36) | U2a attach-only gate | M TestAttachOverlapRequiresMultiAttach / N-attach-overlap-multi; TestAttachOverlapRequiresMultiAttachForInputClient / N-attach-overlap-multi-input-client | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| overlapinput (ops.go:299; termbind/overlap.go:36) | U2a attach-only gate | M TestAttachOverlapInputRequiresMultipleInputClients / N-attach-overlap-input; TestAttachOverlapInputGateChecksEveryPeer / N-M5-attach-overlap-input-first-peer-only | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| waitbound (lifecycle.go:750,776; ops.go:259) | U2a create takes no barrier and runs no admission wait | M AttachWaitBoundedByOperationDeadline/admission,barrier / N-attach-wait-deadline shared; AttachWaitCancelledReturnsPromptly stays green (cancellation companion) | U2a status takes no barrier and runs no admission wait | M QuiesceBarrierWaitBoundedByOperationDeadline, BoundaryCommitWaitBoundedByOperationDeadline / N-attach-wait-deadline shared | U2a terminate takes no barrier and runs no admission wait | U2a restore takes no barrier and runs no admission wait | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| peershape (termbind/overlap.go:36) | U2a attach-only gate | M AttachPeerDirectoryFailsClosed / N-attach-peers-dir-skip; AttachPeerFilenameMismatchFailsClosed / N-attach-peers-filename-skip | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| overlapreplay (ops.go:299) | U2a attach-only gate | M AttachSameClientRetryWithPeerPresent / N-attach-overlap-replay-unproven | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| admissionlock (ops.go:157; termbind/admission.go:22) | U2a attach-only gate | M TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances / N-attach-admission-lock; TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances / N-M2-attach-admission-lock-input-only | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| livenessunknown (ops.go:314; termbind/overlap.go:36) | U2a attach-only gate | M AttachReceiptWithoutPositiveLivenessRemainsPossiblePeer / N-attach-liveness-unknown | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| attachaxpolicy (ops.go:87; termbind/attach.go:143; terminalbackend/conformance.go:683) | U2a attach-only gate | M AttachConcurrentInputRequiresAXPolicy / N-attach-ax-input-policy | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U2a attach-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| confirmbound (backend.go:540) | U2a terminate-only gate | U2a terminate-only gate | U2a terminate-only gate | U2a terminate-only gate | M ExecuteTerminateConfirmHonorsOperationDeadline / N-probe-deadline shared | U2a terminate-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 direct staging unmeasured here; M at EXT | U1 | U1 |
| statusbound (status.go:28) | U2a status-only gate | U2a status-only gate | M ExecuteStatusProbeHonorsOperationDeadline / N-probe-deadline shared | U2a status-only gate | B39 resume reaches via status reconcile; M at EXS | B39 resume reaches via status reconcile; M at EXS | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 |
| escalationbound (backend.go:431) | U2a stop-only gate | U2a stop-only gate | U2a stop-only gate | M ExecuteStopEscalationSucceedsWithContextHonoringRunner / N-stop-escalation-budget | U2a stop-only gate | U2a stop-only gate | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | U1 | B39 direct staging unmeasured here; M at EXE | U1 | U1 |

Symmetry (every gate on two or more sides, with admitting-row
counts per side; "shared" marks one row killing through several
entries, attribution verified per entry from the raw logs):
directive CMD 1 / PDIR 1 shared; scrub OSR 2 / UNT 2 shared; cmdident
CMD 3 members; cmdvector EXC 1 / EXA 1 / EXE 1 / EXR 1 / CMD 4;
directives 6 EX + DIRV shared 1; effectvocab/exitrefused/transport
single-sided at PFX; reconfirm/escalation/persistmap/attachmap
single-sided at PFX; boundarywait EXE 1 / PFX 1 shared; staterecord
REC 2 members; statelookup LOOK 2 members; entrypoint/transportvocab
single-sided at UNT; absentmemory EXS 1 / CLS 1 shared; presentmemory
EXS 1 / CLS 1 / OBSP 1 shared; attachable EXS 1 / CLS 1 shared; cutrow
OBSP 1 / UNT 1 shared (EXS B23); triple EXS 1 / UNT 1 shared; dialstale
DIAL 2 members; probedeps PRB 2 members; sunpath LEN 1 / 6 EX 1 /
PRB 1 shared; placement CUS 1 / PRB 1 shared; leafdelegate CUS 1 (B32
propagation elsewhere); leafwiring CUS 2 sites; rootwrite CUS 1 /
EXC 1 / EXA 1 / EXS 1 / EXR 1 / SPW 1 / shared EachOperation row
(EXA, EXE, EXT);
ancestorwrite CUS 1 / EXS 1 / EXR 1 / shared EachOperation row
(EXC, EXA, EXE, EXT); socketkind CUS 1 / shared EachOperation row
(6 EX); ancestorkind CUS 1 /
PRB 1 / SPW 1 shared; backendid 6 EX shared 1; agreement EXC 4
members; enginemembers EXE 4 members; attachcap EXA 2 members;
attestation EXA 1 + pinned member; attachdeadline EXA 2 arms;
authkind EXT 2 / EXR 2 (entry 1 / shared loop 1 each; D-loop wiring
supplementary, uncounted); generation EXA 1 / EXT 1 / EXR 2; deadline
EXA 2 / EXT 1 / EXR 2; keymaterial EXT 1 / EXR
1; capdual EXT 2 members; fencingmap EXT 2 + 3 pinned; target EXT 3
members; winner EXT 2 members (session arm U3: fencing+target
precede); record EXT 1 / EXR 1 shared + UNT 1; replay EXT 2 + 6
pinned; bindmap/bindfound single-sided at EXT; readonly EXA 1 / CMD 1
shared; detachvec CMD 1; boundarykind/boundexit EXE 1 each + PFX
0/1; quiescedetach EXE 1 / PFX 1 shared; nosendkeys/closeexit EXE 1
each; probemarker EXS 1 / EXE 1 / EXT 1 shared; probeneg EXE 1;
identity CUS 1; bindcontract/restoreagree EXR 1 each; replaytime EXE
2; after-restore outcomes in Table E (launch/reattach/offer/park/
refuse + refresh audit); outcomekey REC 1 / LOOK 1; binddocid REC
1 / LOOK 1; binddocshape REC 1; docpersist EXR 1; attachreadonly
EXA 1; quiescereport/boundaryreport EXE 3 rows each
(corrupt/empty/crash legs); statusunbound/statusdoc/statusmatch/
statusnegative EXS 1 each + statusmatch LOOK 1; prioragree EXR 4
tests, 3 rows; succession EXR 3 members; execdeps 6 EX pinned
(B36); Table B verify 9 new-entry sides shared 1, server EXA 1 (its
other sides are first-leaf symmetry). Revision-4 gates:
quiescerecord/boundaryrecord single-sided at EXE;
barrierquiesce/barrierboundary/barrierdecode/barrierempty
single-sided at EXA (same entry as the quiescegate memory arm, whose
row points at them); attachedexit/foreignrow/panesexit single
measured side at EXS (EXT/EXR B39 via the resume reconcile);
argvnul single-sided at OSR (mirrors osrun: no committed Execute
composition injects OSRunner). Revision-5 gates:
barrierincarnation/barrierkey/attachhealth single-sided at EXA;
failurerecord EXE 1 / UNT 1 (different arms: never-overwrite vs
call value); createincarnation EXC 2 rows; restoreincarnation EXR
1 + structural D row / EXT 1; incarnationrecord REC 2 members;
incarnationlookup LOOK 3 members. Rotation symmetry:
createincarnation EXC 1 narrowing row / restoreincarnation EXR 1
narrowing row (symmetric); rootwrite gains PRB 1 (CUS 1 / EXC 1 /
EXS 1 / EXR 1 / PRB 1 / SPW 1 / shared EachOperation row);
statelookup gains EXA 1 alongside LOOK 2 (different arms: absent
vs grammar); statusnegative EXS 2 rows (negative + parse legs).
Revision-6 gates: recheck single-sided at EXA; ordering EXA 1 /
EXE 2 with the three-sided symmetry attach 1 / quiesce 1 /
boundary 1; barrierread EXA 2 arms with the symmetry
directory-read 1 / file-read 1; incarnationlookup gains EXA 1
alongside LOOK 3 (different arms: read-error vs grammar);
rootwrite EXA gains the entry row alongside the shared row (the
valid-body attach is served when the gate is weakened for attach
alone). Revision-8 gates: effectrecheck single-column at EXE with
the two-site symmetry stop 4 / quiesce 4 (3 shared arms + 1
per-site skip row each); overlapmulti/overlapinput single-sided at
EXA; waitbound EXA 1 / EXE 1 shared (4 killers across
attach-admission, attach-barrier, quiesce-span, and
boundary-commit; the cancellation companion stays green);
attachdeadline/deadline EXA gain the shared wait row beside the
survived-supplementary entry row. Revision-9 gates:
peershape/overlapreplay single-sided at EXA; pollbound EXE 1
(B39 at PFX and EXT: the terminate tail reaches the shared
poll); confirmbound single measured side at EXT (B39 at PFX);
statusbound single measured side at EXS (EXT/EXR B39 via the
resume reconcile); the shared probe row kills through the stop,
terminate-confirm, and status deadline tests with isolated
per-test attribution. Revision-10 gate: escalationbound single
measured side at EXE (B39 at PFX); the peershape EXA cell gains
the filename arm beside the directory arm, each narrowing killing
through its own entry test. The three TASK-260922-vcx6yo semantic gates
each have one reachable lifecycle-entry row at EXA. `attachaxpolicy`
uses one shared `CheckAttachRequest` implementation at request preflight
and durable receipt commit within that same `Lifecycle.Execute(attach)`
entry; the common-helper narrowing and one end-to-end row measure both
invocations. Symmetry: 2 helper call sites / 1 shared semantic gate / 1
reachable lifecycle entry / 1 narrowing row. No task-added gate reaches a
second lifecycle-operation entry.

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

## Stated bounds

- B1 Determinism: no row has a real-tmux witness. Every broker, server,
  and spawner interaction runs through injected dependencies; the suite
  passes with no tmux process anywhere. A live-server test would add a
  witness, not a rule, and is sibling scope.
- B2 Windows refusal: native Windows MUST NOT claim tmux (pinned SPEC
  1458). The Windows platform refuses at the lexical gate on every host
  (row 15); the Windows commit/verify arms refuse every platform and
  exist only so the package compiles under GOOS=windows.
- B3 Nil-dependency weakening on the foreground path panics instead of
  misbehaving; the narrowing row pins the refusal and discloses the
  panic kill.
- B4 No gate inspects source text, so no token-preserving mutant
  applies; the harness executes the behavioral suite.
- B5 Attach presentation vectors are sibling scope: attached and broker
  outcomes carry the socket, and only the spawned outcome carries argv.
- B6 Lifecycle operations (create, attach, status, quiesce, stop,
  terminate, restore) belong to the sibling leaves, not to acquisition.
- B7 The handle-mismatch arm stages its race through the sameDirectory
  seam; a true concurrent handle swap is not staged.
- B8 Registry bindings and the digest re-pin belong to the story's
  final leaf; this leaf documents clause coverage here and in the
  conformance matrix only.
- B9 Adapter split: this leaf ships the decision core and argv only. No
  production probe or spawner implementation exists in the tree; the
  exec/probe adapters that report Reconcile/ResolveEvidence verdicts
  and spawn through the built argv, and the server socket's crash
  evidence, are owned by the lifecycle leaf TASK-260830-1c28dz. The
  no-fallback proof covers spawns routed through Dependencies.Spawn;
  a fallback through a direct os/exec call is excluded today only by
  the import census (no os/exec import in the package).
  Dependencies.Spawn is inert on the background path: a nil spawner
  with a live broker still contacts (the broker path never calls it),
  so the field is dead weight there — which is what makes the
  shared-Dependencies fallback shape realistic, and why the spy
  census arms the foreground probe alongside the spawner.
- B10 Crash convergence assumes a umask that preserves owner rwx
  (umask & 0700 == 0). Under an owner-bit-stripping umask a leaf
  stranded between mkdirat and the explicit chmod refuses on retry
  instead of converging, because existing leaves verify, never repair.
- B11 A symlinked intermediate component inside the root path is
  followed; only the root's last component refuses (row 9). This
  matches the landed secprim root open and is the same bound.
- B12 The landed argv shape refusal (BuildArgv via secprim.CheckArgv)
  is reachable only by direct call with a non-filesystem runtime dir
  (witnessed by TestBuildArgvRefusesInvalidArgvShape); through Acquire
  the directory always names a created directory.
- B13 Sentinel/smoke sufficiency, signature verification, liveness,
  and expiry are the landed Reconcile admission's verdict, consumed
  here through Has. This leaf decides membership plus generation
  binding only; the host-binding and provider-build cross-bind the
  wrapper evaluates stays with the lifecycle owner.
- B14 The battery and the read-only-root test assume a non-root test
  user: N-fchmod-umask admits non-root creates past the explicit
  chmod, so under root it is equivalent to the original and survives,
  and TestEnsureRuntimeDirRefusesReadOnlyRoot stages a genuine EACCES
  only when permission bits bind the test user. The reviewer's
  same-shaped plant carries the identical assumption.
- B15 Unmeasured fail-closed stat arms, no seam: handleBoundToPath's
  two stat-failure arms (runtime.go) and verifyOwnership's !ok arm
  (ownership_unix.go) fail closed by code and are prose only by test —
  no test can stage a Stat failure on an open directory handle, or a
  stat without ownership metadata on unix, without a seam. The
  guard.Resolve refusal arm (runtime.go) is unreachable through the
  name grammar: every input the guard would refuse is refused earlier
  by scalar, so the fail-open plant survives by construction. The
  secprim.NewGuard refusal arm (runtime.go:94-96) is the same class:
  no committed input reaches it. The same unstageable class covers
  the commit-side Fchmod and parent-Fsync failure arms and the
  commit-side (runtime_unix.go:76-79) and verify-side leaf-Stat
  failure arms: no committed test stages those failures. An
  unopenable (0000) root reports `runtime containment` on both entries
  with no committed test staging it — classifying EACCES as missing
  would reroute it to `runtime root`, refusal-to-refusal. Reported
  as error-arm rows, never counted among the narrowing survivors.
- B16 The collision gate compares cleaned byte strings: an ambient
  value naming the derived socket through a symlinked or relative
  alias of the root walks past it (realpath equivalence is not
  applied, and a cwd-relative spelling never equals the absolute
  derived socket). Roots containing a comma are outside the TMUX
  member's encoding: the gate splits at the first comma exactly as
  tmux(1) itself does, so a comma-root socket component walks past
  it. The spec rule is unaffected — the socket is derived, never
  selected from an ambient value — so this bounds only the
  package-local defense-in-depth claim. An `ax` invocation from
  inside its own AX pane refuses `ambient collision` when the caller
  populates Ambient from the process environment
  (TMUX=`<derived>`,pid,idx in the real encoding): over-strict
  rather than over-permissive, and the caller owns what it observes
  — the lifecycle leaf TASK-260830-1c28dz decides the
  nested-invocation rule deliberately.
- B17 The mode gate reads permission bits only: setgid/sticky bits on
  a 0700 leaf are admitted and macOS ACLs are not inspected. This
  matches the landed hosttrust custody precedent
  (internal/hosttrust/custody_unix.go platformModeOK: Perm() == mode),
  so it is the same bound, not a fork.
- B18 The derived socket path is not checked against the platform
  sun_path limit (104 bytes on macOS); a deep runtime root surfaces
  as a spawn failure in the lifecycle leaf's adapter, owned by
  TASK-260830-1c28dz.
- B19 The Runtime IPC root's own mode and ownership are not gated in
  this leaf: the root arrives as the caller-supplied
  localstore-resolved PathRuntime value, its creation-time custody is
  the localstore layout owner's, and the SPEC 810-812
  before-bind/connect rejection of an unsafe path, parent, or ancestor
  is the lifecycle bind step's (TASK-260830-1c28dz), which is the step
  that binds. This leaf pins the root handle (no-follow open plus
  handle/path binding) and commits the leaf relative to it; a
  world-writable root is admitted here and refused at bind. That the
  root value is the landed localstore PathRuntime is a contract
  statement about the caller, not a driven fact: no test composes
  localstore with Acquire.
- B20 Whether the runtime leaf was created by AX is indistinguishable
  after the fact: any same-user 0700 directory under an admitted name
  is accepted. The verifiable properties — mode, ownership, kind,
  containment — are gated (rows 1-15); provenance is not claimed.
- B21 No broker is modelled as the zero BrokerReport, whose
  Principal.UID is 0: for a non-root process the absent broker
  refuses on the authentication arm, while under a root process the
  same-user arm passes and refusal comes from the generation arm.
  The broker-path outcome socket binding is the generation equality
  only — BrokerReport carries no socket of its own.
- B22 Singleton arms ship no narrowing mutant: admitting the
  singleton refused member is deleting the gate. Covered sites, each
  test-pinned in both directions where a fork exists: create relay
  transport (relay), descriptor presence fork (descpresence),
  attach admission nil (serveadmitnil), spawner runner nil
  (spawnnil), reconcile verifier nil (reconcilenil), states root
  empty (stateopen), persist/attach stores nil (storenil),
  failed-kill self-confirm (selfconfirm), live-after-escalation
  (stoptimeout), and post-wait boundaries without a configured
  revalidation (recheckPostWait nil). The outcome-lookup empty-key
  arm left this bound: it narrows now (LookupOutcomeRefusesForgedKey /
  N-outcome-key-forged). Owner: this leaf; retest if any site
  grows a second refused member.
- B23 The landed status engine normalizes backend error details to
  "status observation unknown" (coded or not), so the cutrow and
  empty-panes narrowings measure at the helper entries while the
  dispatch legs pin the code. Owner: this leaf; retest if the
  engine stops normalizing.
- B24 The idempotency-store Completed read-failure arm is unstaged:
  no seam fails receipt reads (hooks fire on write paths only).
  Owner: hardening; stage with a read-failure seam to measure.
- B25 The Bind clean-miss sub-arm is unstageable: every Bind error
  the hooks stage leaves the committed file, so Lookup always finds
  it. The clean-found sub-arm is measured (N-bind-hook-found).
  Owner: hardening.
- B26 The attach effect-failure dispatch mapping is unstaged: no
  committed failure reaches it (the store replays identical
  receipts; conflicting receipts are untested at dispatch).
  Owner: hardening.
- B27 No committed test composes the Production Dependencies into
  Acquire: AFG reaches the adapter and bind gates only through
  caller-injected implementations (the 18 Table C U2c cells). The
  injection seam is first-leaf-owned; Production wiring is
  unit-pinned (TestProductionWiresDependencies). Owner: composition
  test, sibling scope.
- B28 The backend command-error wrap is unmeasured at PFX (no
  backend test stages a refused build; the builder rows pin the
  refused shapes at CMD). Owner: this leaf; retest with a
  refused-build backend test.
- B29 The close-confirm poll's exact-instant member is unmeasured
  at every entry: the real-time in-flight bound has no stageable
  equal instant, and no committed test stages the clock arm's
  instant member (past-deadline concludes, pinned at EXE and at
  PFX behaviorally). Dispatch-level staging is measured (row
  pollbound). Owner: hardening.
- B30 The OSRunner exit/transport/argv mappings are total functions
  over uncoded inputs with no refused class to admit a member of;
  tests pin them including real-exec scrub, stderr-capture, and
  signal-death integrations. The signal-death arm (a killed child
  errors instead of reporting -1 data) is a refused class and
  carries its own narrowing row; the remaining mappings stay total.
  Owner: this leaf.
- B31 The custody root-ownership arm is untested and singleton: the
  ownership seam stages leaf-first (the leaf refuses before the
  root arm runs), and a truly foreign root needs privilege. Owner:
  hardening with a privileged stage or a root-level seam.
- B32 The leaf-delegation wiring is measured at CUS (wrong-leaf
  mutants are observable only on compliant roots); the absence
  propagation through the six dispatch paths, the prober, and the
  spawner is pinned by absence tests without a row. Owner: this
  leaf.
- B33 The wrapper safe-boundary signal step is unimplemented in the
  landed wrapper: the boundary entry waits blocking on
  `ax-boundary-<generation>` for the wrapper's `wait-for -S`, and
  the wrapper never signals today, so boundary waits time out with
  the coded quiesce_timeout (fail closed, never a faked proof)
  until the signal step lands. The wait vector, the proof-kind
  binding, the provider-row context, and the timeout are this
  leaf's and fully driven; the signal emission is the wrapper's.
  Owner: the wrapper (`ax pane`) task; retest the full
  quiesce-boundary-stop flow when the signal lands.
- B34 A crash between the mutation-receipt commit and the operation
  outcome record re-observes the report time on retry: the effect
  never re-runs and the evidence never changes, but the reported
  closure/boundary time renders from the retry clock instead of
  the original one. Sequential retries without a crash replay
  identical times. Before the retry heals the report, input can
  reopen: attach consults the outcome proof, so the missing report
  reads as no closure (bound B43 states the owner and the scoped
  contract for the structural fix). Owner: this leaf for the
  timestamp behavior; the receipt owner for the authority window.
- B35 The absence stderr markers are verified against tmux 3.6a
  (`can't find session`, `can't find window`, and the
  no-such-socket connect error); any other version's
  absence-shaped stderr fails closed to unknown. Owner: this leaf;
  re-verify the marker set when the tmux floor version moves.
- B36 The Execute dispatch and the ExecuteWrapperRestore entry pin
  their dependency singletons without per-arm narrowing rows: a
  nil lifecycle or a nil Runner, Receipts, Bindings, Attach,
  States, CurrentLease, CurrentGeneration, or Now refuses
  protocol-error before any decision or effect. Admitting one nil
  arm is deleting the check. Owner: this leaf; retest if any site
  grows a second refused member.
- B37 No production mesh lease-refresh transport exists:
  LeaseRefresh is an injected adapter (function field), and the
  after-restore composition enforces the configured bound on any
  adapter — cooperative or not — and rejects late answers even
  with a nil error. The mesh RPC transport that would back a
  production adapter is sibling scope (the mesh story); until it
  lands, callers that need verified refreshes inject their own.
  Owner: the mesh-refresh composition; retest the bound when a
  production adapter lands.
- B38 Union-rival ambiguity is unmodeled at the after-restore
  composition: the entry decides over the single refreshed or
  local winner, and a refresh adapter that observes rivals must
  surface them as refresh failure so the entry parks unverified.
  Owner: the mesh-refresh composition; retest when rivals are
  modeled.
- B39 Reachable-undriven: the entry reaches the gate but the
  committed tests carry post-gate values only, so the refused
  direction is undriven at that entry; the narrowed direction is
  measured at the listed entry ("M at ..."). Formerly claimed
  unreachable as U2b; stated as a bound now. Owner: this leaf;
  drive a direction to move its cell to M.
- B40 Lost incarnation rotation with a prior closure stays
  closed with no production-entry recovery: the failed rotation
  fails the operation closed, identical retries replay success
  without rotating (rotating on replay would erase a genuine
  newer closure), and a fresh key is unreachable because the
  landed bootstrap store binds one bootstrap per session and the
  receipt store refuses a second operation under a replayed key.
  The stuck closure is over-strict in the safe direction; the
  committed tests pin the failure, the replay skip, and the
  bootstrap-store mechanism. A lost rotation with no prior
  closure is harmless (the zero incarnation stays
  self-consistent). Owner: axpane receipt/bootstrap contracts
  for a repair entry; operator surgery clears the superseded
  reports until one lands.
- B41 The restore replay rotation skip is structural: the replay
  branch contains no rotation code, so no narrowing mutant can
  weaken it — the supplementary additive row
  D-restore-replay-rotation inserts the forbidden write to prove
  the committed replay test observes it. The create-side skip
  has a narrowing row (N-create-replay-skip) because its
  freshness condition is explicit code. Owner: this leaf; the
  asymmetry is stated, not hidden.
- B42 The input-closure ordering contract is in-process: the
  barrier mutex serializes quiesce/boundary report commits
  against attach receipt commits within one Lifecycle, but
  concurrent ax processes racing attach against quiesce observe
  only the report files, so a receipt committed in another
  process's report-write window is not excluded. No committed
  test stages two processes. Owner: this leaf; closing the
  window needs cross-process exclusion (for example a lock file
  under the state root). The vector handoff the contract covers
  ends at vector construction: a caller that execs a
  pre-quiescence vector after quiescence commits holds a
  pre-quiescence authorization, which quiesce does not revoke.
  The same in-process scope covers the attach post-lock
  generation and deadline rechecks: a rotation landing between
  the recheck and the commit — inside one process or across
  processes — is not excluded; the window holds no waits.
- B43 Closure authority has no single durable source. The engine
  commits input closure and its receipt before this leaf records
  the outcome report, and attach admission consults the outcome
  proof alone, so a lost outcome-report write after committed
  closure admits a fresh writable attach until the identical
  retry heals the report: the failure to create the proof reads
  as permission, and pending-versus-committed closure is not
  representable in the outcome store. The same window covers a
  crash between the receipt commit and the report, and a
  simultaneous loss of both the report and the state record.
  Retrying the identical quiesce heals the report and re-closes
  input; no other production entry heals it. Closing the window
  structurally needs the landed receipt owner
  (internal/terminstance ReceiptStore) to expose
  completed-closure enumeration — the completed quiesce/boundary
  receipts for one instance, distinguishing
  pending-without-completion from completed — so that attach
  admission can consult receipt completion as the authoritative
  proof. Owner: internal/terminstance for the enumeration
  contract; this leaf for the admission side once it lands.
  Until then the construction (incarnation-scoped proof,
  advisory-memory split, in-process ordering) stands, and row 79
  claims only the lost state-record write fails closed.
- B44 Attach overlap is inferred from stored receipts, not
  current tmux liveness: the receipt census proves an admitted client
  claim, but the protocol has no positive client identity or detach
  signal and the returned vector is caller-executed. A valid receipt
  without positive liveness evidence is UNKNOWN and remains a possible
  peer. It may conservatively block a later attach after detach; it
  never makes the peer census empty. A persistent per-instance OS lock
  held from census through durable receipt commit makes the overlap
  decision atomic across Lifecycle instances and processes. The gate
  requires multi_attach for any new client that may overlap, plus
  multiple_input_clients and valid AX input authorization for concurrent
  input. A validated same-client replay with a peer is exempt; unreadable
  census data fails closed. This closes B44's admission-safety gap while
  retaining a stated bound: exact live-client distinction and stale
  receipt reclamation require an authoritative retirement signal from a
  future protocol/receipt owner. Owner: this leaf for UNKNOWN-as-possible
  admission; future protocol/receipt owner for retirement evidence.
- B45 Single-shot commit commands (lock-session, detach-client,
  send-keys, new-session, kill-session) and the bind-step spawn
  are bounded by caller cancellation only — not by the operation
  deadline. A hung commit therefore hangs until the caller
  cancels; it never manufactures a verdict (a cancelled commit
  surfaces the honest uncertain: uncoded failure, status_first).
  Mid-commit cancellation cannot un-commit, so bounding commits
  would trade hangs for uncertainty without changing any verdict;
  the landed engine passes its context through to commits and
  checks the deadline between effects, and this leaf mirrors
  that contract rather than diverging from the landed owner.
  Every read-only wait and probe, by contrast, carries a
  deadline-derived context (rows waitbound, pollbound,
  confirmbound, statusbound, boundarywait). Owner: this leaf
  for the audit; internal/terminstance for any engine-wide
  commit-bound contract change.

Carried bounds (first-leaf statements dispositioned by this leaf).
B9 adapter split is discharged: the production exec, dial, probe,
and spawn adapters ship (exec.go, probe.go, backend.go) and
Production wires them into Dependencies; the residual is the
uncomposed wiring (B27). B16 nested invocation is decided: lifecycle
entries take no Ambient input, so nesting cannot collide through
them (row 90); the Acquire-level over-strict nested refusal stands
unchanged as first-leaf-owned. B18 sun_path is decided: strict-below
per platform, counted in bytes (row 66). B19/B20 bind step is built:
CheckSocketCustody enforces placement, leaf delegation, root,
ancestors, and socket kind (rows 67-70); the bind step refuses
intermediate symlinks through no-follow opens, which is stricter
than the B11 Acquire rule it stands beside. The rev7 P3-A fresh-decoy
foreground attach wiring is closed: TestAcquireForegroundRefusesEveryCatalogDecoyRunning
refuses every catalog-derived decoy alone and together on the
foreground running path with a spawn spy silent on every member, and
the attach attestation cells carry N-attach-attestation-decoy. New
bounds this revision: B33 states the unimplemented wrapper boundary
signal (fail-closed timeout until it lands), B34 the receipt/outcome
crash window, B35 the tmux-version marker set, B36 the dependency
singletons, B38 the unmodeled union rivals, and B39 the
reachable-undriven reclass. Revision 4 fills B37 (no production
mesh refresh transport; the composition enforces the bound on any
injected adapter), extends Table E with the binding, expiry, race,
and enforcement outcomes, and moves the SPW rootwrite cell from
B39 to measured. Revision 5 makes the incarnation-scoped closure
proof the sole barrier verdict (new Table D rows
barrierincarnation, barrierkey, attachhealth, failurerecord,
createincarnation, restoreincarnation, incarnationrecord,
incarnationlookup), extends Table E with the five composed-binding
refusals, moves the PRB rootwrite cell from B39 to measured, and
states B40 (stuck rotation) and B41 (structural replay skip).
Revision 6 adds the post-admission barrier recheck with the
in-process ordering lock (new Table D rows recheck and ordering;
attach 1 / quiesce 1 / boundary 1), the barrier read-error arms
(new Table D row barrierread; directory-read 1 / file-read 1),
moves the incarnationlookup EXA cell from B39 to measured, extends
the rootwrite EXA cell with the attach entry row, recounts Table C
from the Table D gate set (140 gates, 1540 cells), narrows row 79
to the lost state-record write with the outcome-report window
stated as B43 (receipt-owner enumeration contract), and states B42
(in-process ordering). Revision 7 closes the attach
admission-to-effect interval for every remaining live fact (the
post-lock generation and deadline rechecks with the full fact
enumeration; the authorization facts revalidate inside the
commit), moves the generation EXA cell from U1 to measured,
extends the deadline and authkind cells with the new rows,
reclassifies the two effect-loop authkind wirings as
supplementary D rows beside the genuine shared loop narrowing
N-lifecycle-loop-authorization, converts the entry authkind rows
to genuine no-bind narrowings, and extends B42 to the new
rechecks. Revision 8 closes the stop admission-to-effect interval
at the escalation boundary and the quiesce interval between its
commands (new Table D row effectrecheck; stop 4 / quiesce 4),
bounds every barrier and admission wait by the operation deadline
and cancellation (new Table D row waitbound; EXA 1 / EXE 1
shared), enforces the attach overlap capabilities fail-closed
over the stored peer census (new Table D rows overlapmulti and
overlapinput; liveness owned by TASK-260922-vcx6yo as B44),
reclassifies the entry-deadline instant row as
shadowed-supplementary (the wait bound refuses the same member),
extends B22 with the nil-recheck singleton, and recounts Table C
from the Table D gate set (144 gates, 1584 cells). Revision 9
refuses receipt-namespace directories in the peer census (new
Table D row peershape), exempts receipt-validated same-client
replays with a peer present (new Table D row overlapreplay),
bounds every read-only probe in-flight by the operation deadline
(new Table D rows confirmbound and statusbound; pollbound moves
from B29 to measured with the terminate tail corrected from U1
to B39), repairs the three false per-entry kill attributions
with isolated single-test evidence (stop 4 / quiesce 4 and 4
wait killers now measured, not claimed), revises B29 to the
exact-instant remainder, clarifies B44 (receipt-validated replay
here, liveness with TASK-260922-vcx6yo), states B45 (commits
bounded by caller cancellation), and recounts Table C from the
Table D gate set (148 gates, 1628 cells). Revision 11 / TASK-260922-vcx6yo adds the AX input-policy composition row, a persistent per-instance OS admission lock, and the UNKNOWN-as-possible-peer liveness rule; its three EXA gates each have one narrowing mutant row. Table C is recounted from 152 gates.


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

## Whole-path custody oracle — pinned SPEC v0.7.0 §3.2

TestCustodyModeOracleAtProductionEntries projects each low-12-bit Unix mode
onto exactly one path position while every other component stays at safe
metadata. The ordered path is socket leaf (depth 0), runtime tmux directory
(depth 1), runtime root (depth 2), and each ancestor from the runtime-root
parent through the filesystem root (depths 3..N). The current-depth fixture
observes depths 3..6 (four ancestors); a second fixture adds eight nested
directories and observes depths 3..14 (twelve ancestors). Across both paths,
every position is swept at ServerProber.Probe, ServerSpawner.Spawn, and
Lifecycle.Execute: (7 + 15) positions × 4096 modes × 3 entries = 270,336 of
270,336 mode/position/entry comparisons matched the independent §3.2 oracle.
TestCustodyAncestorWalkGeneratedDepthAtProductionEntries adds 144 named
refusal cells over extra depths 1..16, nearest/middle/deepest non-root
positions, and all three production entries. Each projects mode 0777 and
requires the literal refusal with zero effects.

TestCustodyPathKindOwnerOracleAtProductionEntries projects all five relevant
kinds (directory, regular file, symlink, FIFO, socket) at every position and
all owner classes at every owner-gated position. Only a socket admits at the
leaf; only a directory admits at every directory component. The current UID
admits at the socket, runtime tmux directory, and root; another non-root UID
and root refuse on this non-root test runner. Ancestor ownership remains the
N2 bound below because §3.2 lines 810–812 does not define an accepted set of
system-ancestor owners. Across both fixtures, the 22 positions × 5 kinds × 3
entries and six owner-gated position/fixture pairs × 3 owners × 3 entries run
as named subtests: 330/330 kind cases and 54/54 owner cases.

The independent mode rules are:

| Component class | Oracle admits | Production predicate / entry path |
|---|---|---|
| Socket leaf | Owner read/write, optional owner execute toggle, no group/other bits, no special bits | ownerOnlySocketMode via checkCustodySocket |
| Runtime tmux directory | Exactly 0700 with no special bits | ownerOnlyRuntimeDirMode via verifyRuntimeDir |
| Runtime root | Exactly 0700 with no special bits | ownerOnlyCustodyRootMode via checkCustodyRoot |
| Ancestor | No group/other write, or sticky set; setuid and setgid never replace sticky | writableByOthers via checkCustodyAncestors |

The independent kind oracle accepts socket at depth 0 and directory at all
other positions. Regular file, symlink, FIFO, and socket at a directory
position refuse. Owner projections retain the actual file identity and change
only the UID consumed by ownershipMatches; tests enumerate current UID,
another non-root UID, and UID 0. custodyModeProjection,
custodyKindProjection, and custodyOwnerProjection are test seams in bind.go;
production leaves them nil. Each path-position subtest counts projection hits
so a bypassed gate is not mistaken for a successful sweep. The leaf is a
synthetic socket FileInfo backed by a regular file. Fake dialers and runners
observe effects; no real socket is bound or connected and no tmux process is
started.

| Gate | Position / component class | Production entry | Executed subtest | Result |
|---|---|---|---|---|
| checkCustodySocket | depth 0 / socket leaf | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/socket_leaf/Probe/depth_00_socket_leaf; TestCustodyPathKindOwnerOracleAtProductionEntries/Probe/depth_00_socket_leaf | All 4096 modes and kinds/owner rows agree; unsafe inputs refuse before Dial |
| checkCustodySocket | depth 0 / socket leaf | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/socket_leaf/Spawn/depth_00_socket_leaf; TestCustodyPathKindOwnerOracleAtProductionEntries/Spawn/depth_00_socket_leaf | All 4096 modes and kinds/owner rows agree; unsafe inputs refuse before unlink or Runner |
| checkCustodySocket | depth 0 / socket leaf | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/socket_leaf/Execute/depth_00_socket_leaf; TestCustodyPathKindOwnerOracleAtProductionEntries/Execute/depth_00_socket_leaf | All 4096 modes and kinds/owner rows agree; unsafe inputs refuse before dispatch |
| ownerOnlyRuntimeDirMode | depth 1 / runtime tmux directory | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Probe/depth_01_runtime_tmux_dir; TestCustodyPathKindOwnerOracleAtProductionEntries/Probe/depth_01_runtime_tmux_dir | All 4096 modes and kinds/owner rows agree; refusals retain tmux_unsafe_runtime_dir |
| ownerOnlyRuntimeDirMode | depth 1 / runtime tmux directory | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Spawn/depth_01_runtime_tmux_dir; TestCustodyPathKindOwnerOracleAtProductionEntries/Spawn/depth_01_runtime_tmux_dir | All 4096 modes and kinds/owner rows agree; refusals retain tmux_unsafe_runtime_dir |
| ownerOnlyRuntimeDirMode | depth 1 / runtime tmux directory | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Execute/depth_01_runtime_tmux_dir; TestCustodyPathKindOwnerOracleAtProductionEntries/Execute/depth_01_runtime_tmux_dir | All 4096 modes and kinds/owner rows agree; refusals retain tmux_unsafe_runtime_dir |
| ownerOnlyCustodyRootMode | depth 2 / runtime root | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/root/Probe/depth_02_runtime_root; TestCustodyPathKindOwnerOracleAtProductionEntries/Probe/depth_02_runtime_root | All 4096 modes and kinds/owner rows agree; refusals assert literal tmux_unsafe_socket_path |
| ownerOnlyCustodyRootMode | depth 2 / runtime root | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/root/Spawn/depth_02_runtime_root; TestCustodyPathKindOwnerOracleAtProductionEntries/Spawn/depth_02_runtime_root | All 4096 modes and kinds/owner rows agree; refusals assert literal tmux_unsafe_socket_path |
| ownerOnlyCustodyRootMode | depth 2 / runtime root | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/root/Execute/depth_02_runtime_root; TestCustodyPathKindOwnerOracleAtProductionEntries/Execute/depth_02_runtime_root | All 4096 modes and kinds/owner rows agree; refusals assert literal tmux_unsafe_socket_path |
| writableByOthers | every actual ancestor, depths 3..6 current and 3..14 deep, including / | ServerProber.Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe in both fixtures; TestCustodyPathKindOwnerOracleAtProductionEntries/Probe at every projected depth | Every fixture depth sweeps 4096 modes and five kinds; generated depths 1..16 refuse mode 0777 at nearest, middle, and deepest non-root positions before Dial |
| writableByOthers | every actual ancestor, depths 3..6 current and 3..14 deep, including / | ServerSpawner.Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn in both fixtures; TestCustodyPathKindOwnerOracleAtProductionEntries/Spawn at every projected depth | Every fixture depth sweeps 4096 modes and five kinds; generated depths 1..16 refuse mode 0777 at nearest, middle, and deepest non-root positions before unlink or Runner |
| writableByOthers | every actual ancestor, depths 3..6 current and 3..14 deep, including / | Lifecycle.Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute in both fixtures; TestCustodyPathKindOwnerOracleAtProductionEntries/Execute at every projected depth | Every fixture depth sweeps 4096 modes and five kinds; generated depths 1..16 refuse mode 0777 at nearest, middle, and deepest non-root positions before dispatch |

The mode loop checks admitted 01777 at every ancestor depth and requires a
fake next-stage effect. Every refusal asserts the literal code and full message
detail; it also asserts zero Dial, spawn, or dispatch effects. Refusals at the
runtime directory retain the established literal tmux_unsafe_runtime_dir
and runtime mode; socket leaf, root, and ancestor refusals assert literal
tmux_unsafe_socket_path and their specific detail.

The required whole-path narrowings were each run ALONE against
TestCustodyModeOracleAtProductionEntries; all four were KILLED in two
independent runs with separate per-plant logs:

| Mutant | Narrowing | Named failing oracle subtest |
|---|---|---|
| N-ancestor-walk-stops-after-first | Returns after checking only the first ancestor | ancestor/Probe/depth_04_ancestor admits an unsafe mode |
| N-ancestor-walk-skips-first | Skips the runtime-root parent | ancestor/Probe/depth_03_ancestor admits an unsafe mode |
| N-ancestor-walk-skips-last-below-root | Skips the final ancestor below / | ancestor/Probe/depth_05_ancestor admits an unsafe mode |
| N-ancestor-walk-checks-only-even-depths | Checks only even ancestor depths from the runtime root | ancestor/Probe/depth_03_ancestor admits an unsafe mode |

C-control is an applied comment-only control and SURVIVED. Raw stdout/stderr,
real exit codes, and the actual failing test lines are in the rev4 per-mutant
logs. The full 349-row shipped harness is reported in the task results.

### N1 — same-client concurrent attach

TestExecuteSerializesSameClientAttachBeforeConcurrentWinnerArm pauses one
Lifecycle.Execute attach after the receipt is staged, then sends the same
client key through a second Lifecycle and independently opened AttachStore.
The second request reaches the per-instance OS admission lock and cannot stage
or commit a duplicate receipt while the first is paused. Once released, both
executions observe the same recorded receipt. N-attach-same-client-concurrent-winner
bypasses the file lock for this fixed fixture; the test alone fails when the
second stage becomes observable. Thus the AttachStore.install concurrent-winner
branch is unreachable from Lifecycle.Execute for same-client attaches while
the production per-instance admission lock is held.

### Six custody-gate axes

| Gate | Client class | Peer count | Peer position | Replay vs new | Concurrent vs sequential | Server generation |
|---|---|---|---|---|---|---|
| Socket leaf | Unreachable before client identity; owner: Lifecycle.Execute | Unreachable before Peers; owner: AttachStore.Peers | Unreachable before peer ordering; owner: AttachStore.Peers | Unreachable before receipt lookup; owner: executeAttach | Sequential whole-path mode and kind/owner projections; deterministic in-check swap refused. B46 remains open after final lstat and before connect, owner: CheckSocketCustody | Unreachable before attestation; owner: CheckServerAttested |
| Runtime tmux directory | Unreachable before client identity; owner: Lifecycle.Execute | Unreachable before Peers; owner: AttachStore.Peers | Unreachable before peer ordering; owner: AttachStore.Peers | Unreachable before receipt lookup; owner: executeAttach | Sequential whole-path mode and kind/owner projections; replacement after custody verification is B46, owner: CheckSocketCustody | Unreachable before attestation; owner: CheckServerAttested |
| Runtime root | Unreachable before client identity; owner: Lifecycle.Execute | Unreachable before Peers; owner: AttachStore.Peers | Unreachable before peer ordering; owner: AttachStore.Peers | Unreachable before receipt lookup; owner: executeAttach | Sequential whole-path mode and kind/owner projections; replacement after custody verification is B46, owner: CheckSocketCustody | Unreachable before attestation; owner: CheckServerAttested |
| Ancestor | Unreachable before client identity; owner: Lifecycle.Execute | Unreachable before Peers; owner: AttachStore.Peers | Unreachable before peer ordering; owner: AttachStore.Peers | Unreachable before receipt lookup; owner: executeAttach | Every ancestor in current and 8-extra-level fixtures sweeps all modes and kinds; generated depths 1..16 test nearest/middle/deepest 0777 refusal. AST test excludes a walk cap. Later path replacement remains B46, owner: CheckSocketCustody | Unreachable before attestation; owner: CheckServerAttested |

### Bounds retained

- B44 remains open: durable attach receipts prove prior admission, not current
  client liveness. The protocol has no positive detach or liveness signal, so
  receipts returned by Peers remain possible peers. Owner: future authoritative
  attach receipt/protocol owner.
- N2 ancestor ownership remains a stated bound: checkCustodyAncestors verifies
  kind and mode but does not require a particular owner above the AX runtime
  root. A foreign non-root owner may rename a protected ancestor. SPEC §3.2,
  lines 810–812 requires unsafe ownership to be rejected but does not define
  the accepted owner set for system ancestors. Owner: future tmux path-custody
  owner at checkCustodyAncestors.
- B46 remains open: a pathname replacement can race after the final socket
  lstat and before OS connect; Unix has no portable path-relative connect pin.
  The injected swap during validation is refused before Dial. Owner:
  CheckSocketCustody.
- Section 4.E replication: no replication prohibition claim is added; this
  package has no replication transport to exercise in this leaf.

### Revision 4 baseline — superseded by Revision 5 below

This section preserves the rev4 measurements for audit history; the rev5
section below supersedes its path-depth counts. The managed task-board worktree
refresh replayed the three Story checkpoints
onto trunk 0ca3e4c26e2b275212796657f785b9b450f6174e. The refreshed Story
checkpoint is 5a64077facb9f93357b9bbc8ebdba93069105715; its three replayed
checkpoints ef3b4e5, 018790a, and 5a64077 each pass git verify-commit. The
working candidate is uncommitted. task-board.config.json is blob-equal to trunk.

The refresh path audit found 64 paths in c9233ce..0ca3e4c; all 57 paths the
Story does not change are byte-equal to 0ca3e4c. It also rechecked the older
c9233ce..14d636e delta: all 56 non-Story paths in its 63-path list are
byte-equal to 14d636e. Neither audit found a mismatch.

The full-path mode oracle matched 86,016/86,016 classifications: seven path
positions (socket leaf, runtime tmux directory, runtime root, and four
ancestors through /) times 4096 low mode values times three production entries.
The kind oracle covers 105/105 path-position × kind × entry cases, and the
owner oracle covers 27/27 owner-gated-position × owner × entry cases.
Ancestor ownership remains the explicit N2 bound; no owner rule is inferred
for system prefixes.

#### Position × entry narrowing census

| Component position | Entry | Named oracle subtest | Narrowing killed |
|---|---|---|---|
| depth 00 / socket leaf | Probe | TestCustodyModeOracleAtProductionEntries/socket_leaf/Probe/depth_00_socket_leaf | N-socket-mask-drop-other-triplet |
| depth 00 / socket leaf | Spawn | TestCustodyModeOracleAtProductionEntries/socket_leaf/Spawn/depth_00_socket_leaf | N-socket-mask-drop-other-triplet |
| depth 00 / socket leaf | Execute | TestCustodyModeOracleAtProductionEntries/socket_leaf/Execute/depth_00_socket_leaf | N-socket-mask-drop-other-triplet |
| depth 01 / runtime tmux directory | Probe | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Probe/depth_01_runtime_tmux_dir | N-runtime-dir-combination-admit |
| depth 01 / runtime tmux directory | Spawn | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Spawn/depth_01_runtime_tmux_dir | N-runtime-dir-combination-admit |
| depth 01 / runtime tmux directory | Execute | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Execute/depth_01_runtime_tmux_dir | N-runtime-dir-combination-admit |
| depth 02 / runtime root | Probe | TestCustodyModeOracleAtProductionEntries/root/Probe/depth_02_runtime_root | N-root-admit-0750 |
| depth 02 / runtime root | Spawn | TestCustodyModeOracleAtProductionEntries/root/Spawn/depth_02_runtime_root | N-root-admit-0750 |
| depth 02 / runtime root | Execute | TestCustodyModeOracleAtProductionEntries/root/Execute/depth_02_runtime_root | N-root-admit-0750 |
| depth 03 / ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_03_ancestor | N-ancestor-walk-skips-first |
| depth 03 / ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_03_ancestor | N-ancestor-walk-skips-first |
| depth 03 / ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_03_ancestor | N-ancestor-walk-skips-first |
| depth 04 / ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_04_ancestor | N-ancestor-walk-stops-after-first |
| depth 04 / ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_04_ancestor | N-ancestor-walk-stops-after-first |
| depth 04 / ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_04_ancestor | N-ancestor-walk-stops-after-first |
| depth 05 / ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_05_ancestor | N-ancestor-walk-skips-last-below-root |
| depth 05 / ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_05_ancestor | N-ancestor-walk-skips-last-below-root |
| depth 05 / ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_05_ancestor | N-ancestor-walk-skips-last-below-root |
| depth 06 / filesystem root ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_06_ancestor | N-ancestor-walk-stops-after-first |
| depth 06 / filesystem root ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_06_ancestor | N-ancestor-walk-stops-after-first |
| depth 06 / filesystem root ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_06_ancestor | N-ancestor-walk-stops-after-first |

The importer grid was rerun against base 0ca3e4c26e2b275212796657f785b9b450f6174e:
192 base rows across 35 packages, 223 candidate rows across 36 packages,
192 shared keys, zero base-only, 31 candidate-only, zero moved. Candidate-only
classes remain listed in the task matrix; no shared outcome class moved.

The shipped harness has 349 plants. A complete pass2 ran in seven bounded
groups, all exit 0: 347 KILLED, two SURVIVED, and no mismatch or infrastructure
error. The initial pass1 attempt reported three NOT_APPLIED anchors for
N-bind-ancestor-writable, N-mode-0777-verify, and N-socket-ownership. The first
two were corrected in bounded group reruns; N-socket-ownership was corrected
and killed in a standalone run. The per-row raw-log audit confirms two
standalone exit observations for every expected kill or survivor across the
repaired pass1 evidence and complete pass2 (`rev4-mutant-artifact-audit-final.log`).
A partial optional pass3 slice was stopped with exit 130 before completion and
is not counted. The survivors are the expected line-count-preserving C-control
and supplementary D-attach-entry-deadline-shadowed; the latter is shadowed by
the preceding wait bound and is not counted as a gate narrowing. The four
whole-path narrowings (stop after first ancestor, skip first, skip last below
/, and even-depth-only) were also each run alone twice and killed by named
TestCustodyModeOracleAtProductionEntries subtests.

Registry re-derivation on the refreshed candidate remains
3664ab2fb166189545fea34a068a318af4f251689f29c92915fa185fefdedeea.
Decoded audit confirms all six required Story/trunk acceptance cases have
clause edges, declared production owners, and named tests. The
story-260922-derivation-side-profile-source case remains in clause 2.4#2.
Exact tracecheck output and section-scope diagnostics are in task evidence.

### Revision 5 — generated walk depth and unbounded path axes

The walk-length measurement in this historical section is superseded by
Revision 6 below. Other rev5 mode, kind, owner, component-name, and symlink
measurements remain in force.

Pinned authority remains SPEC v0.7.0 §3.2, lines 806–813. The rev4 fixture
depth finding is answered by generating 1–16 extra nested directories beneath
the temp base. `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries`
drives 16 depths × three positions (nearest, middle, deepest non-root) ×
Probe/Spawn/Execute = 144 named cells. Every cell projects mode 0777 onto that
ancestor, asserts the literal `tmux_unsafe_socket_path at socket ancestor`,
and observes zero Dial, spawn, or dispatch effects. The current-depth and
eight-extra-level oracle fixtures cover 22 concrete component positions in
all three entries: 270,336/270,336 mode cases, 330/330 kind cases, and 54/54
owner cases. The deep oracle walks twelve ancestors.

`TestCustodyAncestorWalkHasNoLengthDependentControlFlow` parses the production
`checkCustodyAncestors` AST. It requires one unconditional loop, no init/test/
post counter, no integer-literal depth guard, no break/continue/goto, exactly
the `parent == cleaned` and filesystem-root exits, and no `return nil` outside
those root guards. Its cap and early-return control plants both fail that test
alone. The production loop advances with `filepath.Dir` until one of those
filesystem-root conditions or an existing custody refusal; there is no finite
path-depth claim.

| Gate / unbounded axis | Generator range and named evidence | Structural argument or stated bound |
|---|---|---|
| `writableByOthers` / path depth | Extra nesting 1..16 at nearest, middle, and deepest non-root positions through all entries; all mode/kind sweeps also run on current and extra-depth-08 fixtures. `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries`, `TestCustodyModeOracleAtProductionEntries`, `TestCustodyPathKindOwnerOracleAtProductionEntries`. | The AST test proves the production loop advances only with `filepath.Dir` and exits only at the cleaned/filesystem root or on refusal. No counter or path-length-dependent branch exists. |
| `checkCustodySocket`, `ownerOnlyRuntimeDirMode`, `ownerOnlyCustodyRootMode`, `writableByOthers` / component-name byte length | All legal single-component ASCII byte lengths at runtime-root and ancestor positions through each entry, up to the path bound; the next byte is refused through the production entry. Maxima on this host: Probe and Spawn root 46 / ancestor 48; Execute root 50 / ancestor 52. `TestCustodyPathComponentByteLengthsAtProductionEntries`. | `CheckSocketLength` bounds the complete derived socket path before custody; within that bound the gates use the path component for filesystem metadata and do not branch on its length. No larger name reaches custody. |
| `checkCustodySocket`, `ownerOnlyRuntimeDirMode`, `ownerOnlyCustodyRootMode`, `writableByOthers` / component-name content | ASCII, `.hidden`, `...`, `two..dots`, spaces, `café`, `猫`, and punctuation at runtime-root and ancestor positions, all three entries. `TestCustodyPathComponentNameContentAtProductionEntries`. | The production custody checks use path cleaning and no-follow filesystem metadata; they have no basename prefix, suffix, Unicode, or dot-name allowlist. The finite representatives test distinct lexical classes; arbitrary content does not change the metadata predicate. |
| All four gates / symlink-chain length | Chain lengths 1..16 at socket leaf, runtime tmux directory, runtime root, and ancestor, through Probe/Spawn/Execute. `TestCustodySymlinkChainLengthAtProductionEntries`. | Each entry uses `os.Lstat` or `OpenNoFollowDir` component checks and refuses at the first symlink encountered; it never follows the chain. Length beyond one cannot change the refusal or effects. |
| All four gates / low mode values | All 4096 low 12-bit values at every position in both fixtures, through all entries. `TestCustodyModeOracleAtProductionEntries`. | Domain is finite and exhaustive. Ancestor ownership remains N2: SPEC §3.2 lines 810–813 does not define accepted owners for system ancestors; a foreign owner may rename one. Owner: future tmux path-custody owner at `checkCustodyAncestors`. |
| All four gates / kind and owner | Five kinds at all 22 positions; current/other non-root/root owners at six owner-gated fixture-position pairs. `TestCustodyPathKindOwnerOracleAtProductionEntries`. | Kind and owner domains are explicitly enumerated. Ancestor owner is the N2 bound above. |

The rev5 path-length, content, and chain rows add positive-entry tests; the
generated mode-0777 rows are the negative gate witnesses. Isolated harness
results are recorded in the task result and raw-log archive:

| Plant | Change admitted by the narrowing | Named test that fails | Outcome |
|---|---|---|---|
| `N-ancestor-walk-depth-cap-4` | Any unsafe ancestor after four checked ancestors | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice, each alone |
| `N-ancestor-walk-depth-cap-6` | Any unsafe ancestor after six checked ancestors | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice, each alone |
| `N-ancestor-walk-depth-cap-10` | Any unsafe ancestor after ten checked ancestors | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice, each alone |
| `N-ancestor-walk-skips-deepest-generated-ancestor` | Mode 0777 at the deepest non-root position | `TestCustodyAncestorWalkGeneratedDepthAtProductionEntries` | KILLED twice, each alone |
| `N-ancestor-walk-depth-cap-ast-control` | Adds an explicit four-ancestor counter/cap | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | KILLED twice, each alone |
| `N-ancestor-walk-early-return-ast-control` | Adds an early `return nil` outside the root exit | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | KILLED twice, each alone |
| `C-control` | Harmless comment-only line-count-preserving edit | None expected | SURVIVED twice; harness control |

The harness `--help` invocation is unsupported (it returns `unknown mutants:
--help`, exit 2); usage is documented in the harness source. Named mutant runs
and raw per-plant logs work as intended. No tmux process is required by these
tests.

### Revision 6 — physical path depth and transitive guard

Pinned authority remains SPEC v0.7.0 §3.2, lines 806–813. The walk-length
axis is generated to the platform unix.PathMax constant rather than a
fixture-selected depth. On Darwin, unix.PathMax is 1024. The generated
runtime-root path is the longest valid pathname on this host (1023 bytes);
its deepest ancestor is 1021 bytes because the runtime root child needs the
final two bytes (/r). The fixture has 460 ancestor positions and 452
one-byte nested directories. TestCustodyAncestorWalkGeneratedDepthToPathMax
sets a real non-sticky 0777 mode at the deepest non-root ancestor and then at
a middle ancestor. Both are refused by the shared checkCustodyAncestors
predicate with the literal tmux_unsafe_socket_path at socket ancestor. The
test registers reverse-order t.Cleanup removal for every generated directory.

The derived socket path for that physical-depth root is longer than Darwin's
sun_path limit, so Probe, Spawn, and Execute stop earlier at their existing
CheckSocketLength gate. TestCustodyAncestorWalkReachableFromProductionEntries
parses the production call graph and proves that all three entries reach
CheckSocketCustody, which reaches the exact shared checkCustodyAncestors
predicate. The physical-depth behavior is therefore exercised at the shared
predicate and its use is wired from every entry without a test-only bypass in
production.

TestCustodyAncestorWalkHasNoLengthDependentControlFlow now builds the
transitive in-repository call graph from checkCustodyAncestors, using the
active Go build files in internal/tmuxserver and internal/secprim. The
observed graph includes custodyModeForPath, writableByOthers,
isFilesystemRoot, secprim.OpenNoFollowDir, and its in-module callees.
Across those functions it rejects path-derived length, separator-count, or
depth comparisons; counter-bearing loops; and counter updates. The existing
secprim.memberErrorTarget helper is identified mechanically as a
string-returning helper whose every call is nested in failPath detail
construction; its existing 64-character message truncation cannot influence a
custody decision. Path provenance also flows through `strings.Split`, a range
index over path components, numeric component-count expressions, and
`[]byte(path)` conversion. No other path-derived length comparison is
exempted. `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` rejects the
split-count, range-index, and byte-length caps through the mechanically
discovered call graph.

| Gate / unbounded axis | Generator range and named evidence | Structural argument or stated bound |
|---|---|---|
| writableByOthers / walk length | unix.PathMax-bounded absolute runtime-root path; Darwin fixture root 1023 bytes, nearest ancestor 1021 bytes, 460 ancestor positions. Deepest and middle non-root positions get real 0777 modes. TestCustodyAncestorWalkGeneratedDepthToPathMax; all mode/kind/owner sweeps still run at the current path and extra-depth-08 fixture. | The transitive AST guard covers every in-module callee reached from checkCustodyAncestors and rejects count/length/depth comparisons and counters. No pathname longer than unix.PathMax-1 bytes can be named by the Unix pathname APIs. The derived socket exceeds sun_path; all three production entries are statically wired to the same predicate by TestCustodyAncestorWalkReachableFromProductionEntries. |
| checkCustodySocket, ownerOnlyRuntimeDirMode, ownerOnlyCustodyRootMode, writableByOthers / component-name byte length | All legal single-component ASCII byte lengths at runtime-root and ancestor positions through each entry, up to the socket-path bound; the next byte is refused. Maxima: Probe and Spawn root 46 / ancestor 48; Execute root 50 / ancestor 52. TestCustodyPathComponentByteLengthsAtProductionEntries. | CheckSocketLength bounds the complete derived socket path before custody. The transitive guard rejects path-derived length thresholds in the custody decision graph. Error-detail truncation is separately bounded at 64 characters by secprim.memberErrorTarget, whose callsites are structurally confined to failPath detail construction. |
| All four gates / component-name content | ASCII, .hidden, ..., two..dots, spaces, café, 猫, and punctuation at runtime-root and ancestor positions through all entries. TestCustodyPathComponentNameContentAtProductionEntries. | Production checks use path cleaning and no-follow filesystem metadata without a basename or Unicode allowlist. |
| All four gates / symlink-chain length | Chain lengths 1..16 at socket leaf, runtime tmux directory, runtime root, and ancestor through Probe, Spawn, and Execute. TestCustodySymlinkChainLengthAtProductionEntries. | Each entry uses os.Lstat or OpenNoFollowDir and refuses at the first symlink. No chain length admits a substituted path. |
| All four gates / low mode values | All 4096 low 12-bit values at every position in the current and extra-depth-08 fixtures through all entries. TestCustodyModeOracleAtProductionEntries. | The domain is finite and exhaustive. Ancestor ownership remains N2: SPEC §3.2 lines 810–813 does not define an accepted owner set for system ancestors. A foreign non-root owner could rename one; owner is the future tmux path-custody owner at checkCustodyAncestors. |
| All four gates / kind and owner | Five kinds at all 22 oracle positions; current/other non-root/root owners at six owner-gated fixture-position pairs. TestCustodyPathKindOwnerOracleAtProductionEntries. | Kind and owner domains are enumerated. Ancestor owner is the N2 bound above. |

| Plant | Change admitted by the narrowing | Named test that fails | Outcome |
|---|---|---|---|
| N-ancestor-walk-callee-depth-cap-24 | custodyModeForPath clears group/other write bits after 24 path separators, admitting a 0777 ancestor beyond the generator range | TestCustodyAncestorWalkGeneratedDepthToPathMax | KILLED twice, each alone; raw logs record test exit 1 |
| N-ancestor-walk-filesystem-root-depth-cap | isFilesystemRoot stops the walk after 24 path separators | TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice, each alone; raw logs record test exit 1 |
| N-ancestor-walk-split-component-count-cap | custodyModeForPath admits after `len(strings.Split(path, "/")) > 24` | TestCustodyAncestorWalkGeneratedDepthToPathMax and TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice alone; raw test exits 1 |
| N-ancestor-walk-range-index-cap | custodyModeForPath admits after a range index over split path components exceeds 24 | TestCustodyAncestorWalkGeneratedDepthToPathMax and TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice alone; raw test exits 1 |
| N-ancestor-walk-byte-length-cap | custodyModeForPath admits after `len([]byte(path)) > 24` | TestCustodyAncestorWalkGeneratedDepthToPathMax and TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice alone; raw test exits 1 |
| C-control | Harmless comment-only line-count-preserving edit | None expected | SURVIVED; harness control |

The named plants ran as isolated harness invocations. Every generated deep
tree is removed in test cleanup. The full shipped harness now contains 360
rows (355 rev5 rows plus five rev6 walk-length narrowings); the 360-row set was
not rerun as one complete harness pass. The five added narrowings each have
two isolated KILLED observations, and the applied neutral control survived.
The first range-index run hit a linker `no space left on device` error and is
excluded; two later isolated runs were KILLED. No tmux process or real socket
is required.
