# TASK-260830-35urbp conformance matrix — implement-private-tmux-server-management (rev7)

Authority: pinned `internal/specdoc/SPEC.v0.7.0.md`. Every row names the
spec clause in the spec's own words, the production call site, and the
named committed test that drives it through that entry. 58 of 58 rows
driven (row 55 extends the rows 19-22 override class with the fifth
ambient member; row 56 binds the Section 3.2 path and runtime-root
clause; rows 57-58 pin the commit-failure and whitespace-override
arms). Bounds B1-B21 are stated in
`internal/tmuxserver/TRACEABILITY.md`. Changes from rev6: row 5 names
the commit-side 0700/FIFO fixtures, the foreground-entry
non-directory test, and the shipped
`N-containment-odirectory-commit` row; row 29 names the `{linux,
wsl2}` miss table; row 31 names the 24-test both-spy census and both
miss tables as the N/D kill paths; rows 27/35 name the
catalog-derived decoy tables at the helper and the background entry;
the §4.C duplicate row number 46 is merged into the §4.2 row 46; the
mutant map carries the new commit-side row and the extended
`N-realm-membership` mask; the census cells for `commit × ENS/AFG`,
`creation × ABG`, `server × ABG/SRV`, and `argv × AFG` name the new
rows, tests, and bounds (cell counts unchanged: 50 measured, 4
bounds, 210 unreachable).

Attestation authority: sentinel/smoke sufficiency, signature
verification, liveness, and expiry are the landed
`internal/terminalbackend` Reconcile/ResolveEvidence admission verdict,
consumed here as an `Admitted` set composed only with the generation
equality `internal/axpane` checkRealm enforces. This leaf decides
membership plus generation binding and re-decides nothing.

## §3.2 socket path and runtime root

| # | Clause (spec words) | Call site | Test |
|---|---------------------|-----------|------|
| 56 | Dedicated socket is `<runtime>/tmux/ax.sock` under the Runtime IPC root (macOS: per-user temporary directory; Linux/WSL2: `$XDG_RUNTIME_DIR/ax`), resolved by the landed localstore layout as PathRuntime and supplied by the caller; the socket is runtime IPC, never durable identity. The runtime leaf is not an Acquire input: the entry always passes RuntimeDirName, so the outcome socket is `<Root>/tmux/ax.sock` on both caller paths | `SocketPath` (socket.go), `Acquire` (acquire.go) | TestSocketPathDerivesOnlyFromRuntimeDir, TestAcquireOutcomeSocketEqualsSpecPath (foreground, background) |

## §4.2 dedicated -S server; no ambient/default discovery or reuse

| # | Clause (spec words) | Call site | Test |
|---|---------------------|-----------|------|
| 16 | AX MUST use its dedicated `-S` server, as `<root>/tmux/ax.sock` with both leaves asserted as spec literals; the Acquire outcome socket equals `<Root>/tmux/ax.sock` on both caller paths | `SocketPath` (socket.go), `Acquire` (acquire.go) | TestSocketPathDerivesOnlyFromRuntimeDir, TestAcquireOutcomeSocketEqualsSpecPath (foreground, background) |
| 17 | MUST NOT discover or reuse the default server by ambient socket name, under hostile ambient on all five members | `ResolveSocket` (socket.go) | TestResolveSocketDerivesIgnoringAmbient |
| 18 | Ambient facts observed without use | `ObserveAmbient` (socket.go) | TestObserveAmbientRecordsWithoutUsing, TestObserveAmbientNilLookupKeepsOutOfBand |
| 19 | Environment-variable vector refuses on both callers with no spawn and no directory | `Acquire` via `ResolveSocket` | TestAcquireRefusesEveryAmbientOverrideVector/environment_variable (foreground, background) |
| 20 | Inherited-socket vector refuses on both callers with no spawn and no directory | `Acquire` via `ResolveSocket` | TestAcquireRefusesEveryAmbientOverrideVector/inherited_socket (foreground, background) |
| 21 | Default-path vector refuses on both callers with no spawn and no directory | `Acquire` via `ResolveSocket` | TestAcquireRefusesEveryAmbientOverrideVector/default_path (foreground, background) |
| 22 | Conventional-name vector refuses on both callers with no spawn and no directory | `Acquire` via `ResolveSocket` | TestAcquireRefusesEveryAmbientOverrideVector/conventional_name (foreground, background) |
| 55 | TMUX_TMPDIR vector refuses on both callers with no spawn and no directory | `Acquire` via `ResolveSocket` | TestAcquireRefusesEveryAmbientOverrideVector/tmux_tmpdir (foreground, background) |
| 58 | Whitespace-only override refuses on both callers with no spawn and no directory | `Acquire` via `ResolveSocket` | TestAcquireRefusesEveryAmbientOverrideVector/whitespace_only (foreground, background), TestResolveSocketRefusesEveryOverrideVector/whitespace_only |
| 23 | Ambient value colliding with the derived socket refuses on every member, byte-identical and unclean (`/./`, `//`) spellings alike, and the TMUX member in the real `<socket>,<pid>,<index>` encoding, on both callers. Only TMUXEnv and InheritedSocket can name the derived socket in real tmux semantics; the other three members pin defence-in-depth shapes. Roots containing a comma are outside the TMUX member's encoding (bound B16) | `Acquire` via `ResolveSocket` | TestAcquireRefusesAmbientCollision (5 members × foreground/background), TestResolveSocketRefusesAmbientCollision (5 subtests), TestResolveSocketRefusesAmbientCollisionUncleanSpellings (2 subtests), TestAcquireRefusesAmbientCollisionUncleanSpelling (foreground, background), TestResolveSocketRefusesTMUXEnvRealEncoding, TestAcquireRefusesTMUXEnvCollisionRealEncoding (foreground, background) |
| 40 | Absent server spawns dedicated with exact -S argv disjoint from every ambient vector, pinned element-by-element through the entry | `Acquire` via `BuildArgv` | TestAcquireForegroundSpawnsDedicatedServer |
| 41 | Running attested dedicated server attaches with no second spawn | `Acquire` | TestAcquireForegroundAttachesToRunningAttested |
| 46 | Spawn vector carries the dedicated -S socket with the `ax pane <logical-session-id>` tail (§4.C cited) | `BuildArgv` (argv.go), `Acquire` | TestBuildArgvAddressesOnlyTheDerivedSocket, TestAcquireForegroundSpawnsDedicatedServer (tail assertion) |
| 47 | Ambient sockets refuse at the spawn vector on all five members | `BuildArgv` (argv.go) | TestBuildArgvRefusesAmbientSocket (5 subtests) |

## §4.2 credential-sensitive creation; background broker-or-refuse

| # | Clause (spec words) | Call site | Test |
|---|---------------------|-----------|------|
| 24 | A background CLI MAY contact an already-running authenticated Aqua broker and its attested AX tmux server (macOS) | `Acquire` via `CheckBrokerContact` | TestAcquireBackgroundContactsBroker |
| 25 | If neither exists it MUST return `capability_unavailable` with typed realm/readiness details: no broker | `Acquire` | TestAcquireBackgroundMissReturnsTypedUnavailable/no_broker |
| 26 | Same: foreign-user broker | `Acquire` | TestAcquireBackgroundMissReturnsTypedUnavailable/foreign-user_broker |
| 27 | Same: unattested server, including the zero-admission and decoy-only members, and every catalog-derived decoy alone and together | `Acquire` | TestAcquireBackgroundMissReturnsTypedUnavailable/unattested_server, .../zero_server_admission, .../decoy-only_server_admission, TestAcquireBackgroundRefusesEveryCatalogDecoy (15 members + all together) |
| 28 | Same: stale-generation admission, including the admitted-without-generation member | `Acquire` | TestAcquireBackgroundMissReturnsTypedUnavailable/stale_server_admission, .../admitted_server_without_generation, .../generation-unbound_principal |
| 29 | Linux and WSL2 background misses likewise refuse without fallback | `Acquire` | TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback (linux, wsl2) |
| 30 | Broker probe failure is unknown, never unavailable, with no spawn | `Acquire` | TestAcquireBackgroundProbeFailurePassesThrough |
| 31 | A Background caller MUST NOT create a credential-dependent tmux server; MUST NOT fall back to direct server creation on any of the three tmux platforms (macOS, Linux, WSL2) — every background-path Acquire test (24/24) arms both a spy spawner and a spy server probe | `checkCreationAllowed` via `Acquire` | TestCheckCreationAllowedRefusesBackground, TestAcquireBackgroundMissReturnsTypedUnavailable + TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback (kill paths of N-creation-literal, D-background-fallback, D-background-fallback-foreground) |
| 42 | Running unattested server refuses with no spawn and no repair, including zero admission rows and an empty generation | `Acquire` | TestAcquireForegroundRefusesRunningUnattested, TestAcquireForegroundRefusesRunningWithEmptyAdmission, TestAcquireForegroundRefusesRunningWithEmptyGeneration |
| 43 | Server probe failure is unknown, never absent, with no spawn | `Acquire` | TestAcquireForegroundProbeFailurePassesThrough |
| 44 | Spawn failure reports with exactly one attempt and no fallback | `Acquire` | TestAcquireForegroundSpawnFailureReportsWithoutFallback |
| 45 | Malformed session refuses at the foreground entry before any side effect (no directory, no probe, no spawn) | `Acquire`, `BuildArgv` | TestAcquireForegroundRefusesMalformedSession |
| 48 | Session identity follows the landed UUIDv7 wiring | `BuildArgv` (argv.go) | TestBuildArgvRefusesMalformedSession |
| 54 | Spawn-then-retry converges to the one server with exactly one spawn | `Acquire` | TestAcquireForegroundSpawnIsIdempotentAcrossRetry |

## §4.2 managername hint only; cached observations never authorize

| # | Clause (spec words) | Call site | Test |
|---|---------------------|-----------|------|
| 32 | A cached sentinel or prior `managername` observation MUST NOT authorize resume (background: a stale admission replayed on a miss still returns the typed unavailable) | `Acquire` | TestAcquireBackgroundMissReturnsTypedUnavailable/stale_server_admission |
| 33 | Same (foreground: a stale admission fakes no running attested server) | `Acquire` | TestAcquireForegroundRefusesWrongGenerationAdmission |
| 34 | Bound admitted row admits | `CheckServerAttested` (readiness.go) | TestCheckServerAttestedAdmitsBound |
| 35 | Missing realm row refuses, including a decoy capability, at the helper, the broker contact, and both Acquire paths; every non-realm Section 4.D member derived from the pinned catalog refuses alone and together at the helper and through the background entry | `CheckServerAttested` (readiness.go), `CheckBrokerContact` (readiness.go), `Acquire` | TestCheckServerAttestedRefusesEachMissingConjunct (no admission rows, decoy capability only), TestCheckBrokerContactRefusesEachMissingFact (zero server admission, decoy-only server admission), TestAcquireForegroundRefusesRunningUnattested, TestAcquireBackgroundMissReturnsTypedUnavailable/zero_server_admission, .../decoy-only_server_admission, TestCheckServerAttestedRefusesEveryCatalogDecoy + TestAcquireBackgroundRefusesEveryCatalogDecoy (15 members + all together) |
| 36 | Wrong-generation admission refuses, including the both-empty pair, at the helper, the broker contact, and both Acquire paths | `CheckServerAttested` (readiness.go), `CheckBrokerContact` (readiness.go), `Acquire` | TestCheckServerAttestedRefusesEachMissingConjunct (stale generation admission, both generations empty), TestCheckBrokerContactRefusesEachMissingFact (stale server admission, admitted server without generation), TestAcquireForegroundRefusesWrongGenerationAdmission, TestAcquireBackgroundMissReturnsTypedUnavailable/stale_server_admission, .../admitted_server_without_generation |
| 37 | Foreign-user broker principal refuses (`§4.4` same-user) | `CheckBrokerContact` (readiness.go), `Acquire` | TestCheckBrokerContactRefusesEachMissingFact, TestAcquireBackgroundMissReturnsTypedUnavailable/foreign-user_broker |
| 38 | Generation-unbound broker principal refuses (`§4.4` generation-bound), including the both-empty pair | `CheckBrokerContact` (readiness.go), `Acquire` | TestCheckBrokerContactRefusesEachMissingFact (incl. both generations empty), TestAcquireBackgroundMissReturnsTypedUnavailable/generation-unbound_principal |
| 39 | Complete broker report admits | `CheckBrokerContact` (readiness.go) | TestCheckBrokerContactAdmitsLiveReport |

## §4.C wrapper entrypoint shape (cited)

§4.C is cited through row 46 (spawn tail), discharged in §4.2 above —
it carries no separate row number.

## §4.D capability binding (cited landed)

| # | Clause (spec words) | Call site | Test |
|---|---------------------|-----------|------|
| 25-28 | Typed refusal names `credential_capable_execution_realm` with the exact Section 15.3 detail set | `Acquire` via `axerror.NewRealmEvidenceUnavailable` | TestAcquireBackgroundMissReturnsTypedUnavailable (detail assertions in every subtest) |

## §4.E owner-only sensitive runtime state

| # | Clause (spec words) | Call site | Test |
|---|---------------------|-----------|------|
| 1 | Runtime directory created owner-only (0700), explicitly under a restrictive umask | `EnsureRuntimeDir` | TestEnsureRuntimeDirCreatesOwnerOnly, TestEnsureRuntimeDirEnforcesModeUnderRestrictiveUmask |
| 2 | Re-ensure converges idempotently | `EnsureRuntimeDir` | TestEnsureRuntimeDirIsIdempotent |
| 3 | Compliant directory verifies | `VerifyRuntimeDir` | TestVerifyRuntimeDirAdmitsCompliant |
| 4 | Widened (0755) and narrower (0600/0500) modes refuse, never silently repaired | `EnsureRuntimeDir`, `VerifyRuntimeDir` | TestEnsureRuntimeDirRefusesWidenedMode, TestEnsureRuntimeDirRefusesNarrowerMode |
| 5 | Non-directory leaf refuses on commit and verify and through both Acquire entries (incl. symlink leaf and handle mismatch); FIFO refusal relies on O_DIRECTORY failing fast, pinned by the commit-side and verify-side deadline tests | `EnsureRuntimeDir`, `VerifyRuntimeDir`, `Acquire` (foreground + background) | TestEnsureRuntimeDirRefusesNonDirectoryLeaf (0600, 0700, FIFO under deadline), TestVerifyRuntimeDirRefusesNonDirectoryLeaf (regular file, FIFO under deadline), TestAcquireForegroundRefusesNonDirectoryLeaf, TestAcquireBackgroundRefusesNonDirectoryLeaf, TestAcquireBackgroundRefusesSymlinkLeaf, TestAcquireBackgroundRefusesHandleMismatch |
| 6 | Symlink leaf refuses on commit and verify. On darwin the verify-side ELOOP arm is unreachable (ENOTDIR); on Linux the same open returns ELOOP, mapped to containment — reasoned, not run | `EnsureRuntimeDir`, `VerifyRuntimeDir` | TestEnsureRuntimeDirRefusesSymlinkLeaf, TestVerifyRuntimeDirRefusesSymlinkLeaf |
| 7 | Foreign ownership refuses | `EnsureRuntimeDir`, `VerifyRuntimeDir` | TestEnsureRuntimeDirRefusesForeignOwnership |
| 8 | Handle/path mismatch refuses | `EnsureRuntimeDir` | TestEnsureRuntimeDirRefusesHandleMismatch |
| 9 | Symlinked root refuses on commit, verify, and through the background entry | `EnsureRuntimeDir`, `VerifyRuntimeDir`, `Acquire` (background) | TestEnsureRuntimeDirRefusesSymlinkRoot, TestVerifyRuntimeDirRefusesSymlinkRoot, TestAcquireBackgroundRefusesSymlinkRoot |
| 10 | Relative, empty, parent, and missing roots refuse invalid (runtime root), the same code verify reports (row 13); the foreground entry refuses a bad root before any side effect | `EnsureRuntimeDir`, `Acquire` (foreground) | TestEnsureRuntimeDirRefusesBadRoots (4 subtests), TestAcquireForegroundRefusesBadRoot (relative, missing) |
| 11 | Malformed names refuse | `EnsureRuntimeDir` | TestEnsureRuntimeDirRefusesBadNames (6 subtests) |
| 12 | Unknown platform refuses on commit, verify, and through Acquire on both callers | `EnsureRuntimeDir`, `VerifyRuntimeDir`, `Acquire` | TestEnsureRuntimeDirRefusesBadPlatform, TestVerifyRuntimeDirRefusesBadLexical (unknown platform), TestAcquireRefusesUnknownPlatform (foreground, background) |
| 13 | Verify refuses absence (runtime absent) and a missing root (runtime root, aligned with the commit side via the shared classifier) without creating | `VerifyRuntimeDir` | TestVerifyRuntimeDirRefusesAbsence, TestVerifyRuntimeDirRefusesBadLexical, TestVerifyRuntimeDirRefusesMissingRoot |
| 14 | Linux and WSL2 share the unix custody discipline | `EnsureRuntimeDir` | TestLinuxAndWSL2ShareUnixCustody |
| 15 | Windows platform refuses on every host on commit, verify, and through Acquire on both callers; no tmux directory exists on Windows | `EnsureRuntimeDir`, `VerifyRuntimeDir`, `Acquire` | TestEnsureRuntimeDirRefusesWindowsOnUnixHost (commit and verify legs), TestAcquireRefusesWindowsPlatform (foreground, background) |
| 57 | Commit-phase filesystem failure (unwritable root) refuses the commit detail instead of being mistaken for an existing leaf | `EnsureRuntimeDir` | TestEnsureRuntimeDirRefusesReadOnlyRoot |
| 52 | Foreground acquisition ensures the runtime directory idempotently; background verifies without creating — an absent leaf on macOS, Linux, and WSL2 returns the typed capability_unavailable, a missing root refuses invalid, an unsafe leaf keeps its custody code (top-level *Error, never the unavailable wrapper) on the mode, ownership, and containment members alike | `Acquire` via `EnsureRuntimeDir`/`VerifyRuntimeDir` | TestAcquireCreatesRuntimeDirIdempotently, TestAcquireBackgroundVerifiesWithoutCreating (macos, linux, wsl2), TestAcquireBackgroundMissingRootRefusesInvalid, TestAcquireBackgroundStagesVerifyCustodyRefusal (mode), TestAcquireBackgroundStagesOwnershipRefusal (ownership), TestAcquireBackgroundRefusesNonDirectoryLeaf (containment), TestAcquireBackgroundRefusesSymlinkLeaf, TestAcquireBackgroundRefusesSymlinkRoot, TestAcquireBackgroundRefusesHandleMismatch |
| 53 | Crash between mkdir and fsync retries to the one verified directory (real SIGKILL) | `EnsureRuntimeDir` | TestEnsureCrashChildSelfTerminates |

## §15.3 typed refusal shape (input validation)

| # | Clause (spec words) | Call site | Test |
|---|---------------------|-----------|------|
| 49 | Unknown caller refuses, with the foreground post-admission spy set armed | `Acquire` | TestAcquireRefusesUnknownCaller |
| 50 | Missing typed-detail members refuse on both callers; a present-but-invalid detail refuses after the broker probe | `Acquire` | TestAcquireRefusesMissingRealmDetails (3 members × foreground/background), TestAcquireBackgroundInvalidGenerationRefusesAfterProbe |
| 51 | Missing dependencies refuse on every nil shape | `Acquire` | TestAcquireRefusesMissingDependencies, TestAcquireBackgroundNilBrokerWithSpawnRefuses, TestAcquireForegroundNilSpawnWithProbeRefuses |

## Mutant-to-gate map (all kills through the production entries above)

N-mode-create (row 4, create arm), N-mode-verify (row 4, verify arm),
N-mode-narrower-create (row 4, create arm, 0600 member),
N-mode-narrower-verify (row 4, verify arm, 0600 member),
N-fchmod-umask (row 1, explicit chmod arm),
N-ownership-uid (row 7), N-containment-slash (row 11),
N-containment-mismatch (row 8, commit + verify + background entries),
N-containment-symlink (row 6, commit arm),
N-containment-symlink-verify (row 6, verify arm + background entry),
N-containment-odirectory-verify (row 5, verify arm + background entry),
N-containment-odirectory-commit (row 5, commit arm + foreground entry),
N-root-wiring (row 10, commit + verify + foreground entries),
N-root-missing-classify (rows 10, 13, 52; shared classifier, three entries),
N-root-symlink-commit (row 9, commit arm),
N-root-symlink-verify (row 9, verify arm + background entry),
N-commit-eacces-as-exist (row 57),
N-platform-wiring (row 12, three entries),
N-platform-windows (row 15, three entries),
N-override-env/inherited/default/conventional/tmpdir (rows 19-22, 55;
Acquire both callers + direct ResolveSocket),
N-override-whitespace (row 58, both entries),
N-collision + N-collision-inherited/tmuxenv/tmpdir/conventional (row 23,
byte-identical per member, both callers + direct),
N-collision-unclean (row 23, normalization arm, both entries),
N-collision-tmuxenv-encoding (row 23, TMUX encoding arm, both entries),
N-realm-membership (rows 27, 35, 42; helper + foreground +
broker-contact + background-miss tables + both catalog-decoy tables,
killing on the headless_creation member and the all-together set),
N-realm-generation (rows 33, 36; four tables),
N-realm-empty-want (row 36, both-empty pair),
N-broker-uid (rows 26, 37), N-broker-generation (rows 28, 38),
N-broker-empty-generation (row 38, both-empty pair),
N-broker-zero-admission (rows 27, 28, 36; broker-contact + background-miss),
N-running-empty-admission (rows 41/42, Running arm, zero rows),
N-running-empty-generation (rows 41/42, Running arm, empty generation),
N-unattested-foreground (rows 33, 36; foreground wiring),
N-verify-absence-detail (rows 13, 52, verify arm + background entry),
N-background-absence-remap (row 52, background remap, macOS member),
N-absence-remap-widen (row 52, absence classifier, mode member),
N-absence-remap-ownership (row 52, absence classifier, ownership member),
N-absence-remap-containment (row 52, absence classifier, containment member),
N-absence-remap-invalid (row 52, absence classifier, invalid member),
N-creation-literal (row 31), N-caller-vocabulary (row 49),
N-details-generation/brokerstate/remediation (row 50, one row per member),
N-details-constructor (row 50, refusal-renderer arm),
N-session-entry (row 45, foreground-entry arm),
N-dispatch-foreground/background (caller→path routing, both directions),
N-wiring-lexical/ensure/verify (rows 16, 56; one row per site),
N-acquire-commit-refusal (foreground custody wiring),
N-acquire-verify-refusal (background custody wiring),
N-spawn-swallow (row 44), N-probe-swallow-foreground/background (rows 43, 30),
N-argv-socket + N-argv-tmuxenv (row 47), N-argv-shape (P3-D arm),
N-argv-session (rows 45, 48; spawn-site arm),
N-argv-detached + N-argv-tail (rows 40, 46; vector shape, helper + entry),
N-deps-background/foreground (row 51),
D-background-fallback + D-background-fallback-foreground supplementary
(rows 25-28, 31), C-control SURVIVED.
Error-arm rows (stat-failure arms flipped to fail-open, the unreachable
guard arm, the unopenable-root arm) are reported under bound B15 and never
counted among the narrowing rows.

## Gate × entry census (mirrors internal/tmuxserver/TRACEABILITY.md)

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
