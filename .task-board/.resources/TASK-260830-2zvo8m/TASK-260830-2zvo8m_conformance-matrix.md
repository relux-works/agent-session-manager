# TASK-260830-2zvo8m conformance matrix — native-resume smoke framework

Pinned authority: `relux-works/agent-session-manager-spec@v0.6.0`
(`internal/specdoc/SPEC.v0.6.0.md`). Each row maps one in-scope clause
to its production call site and named committed test, or to a stated
bound. "Proven" means a named test drives the production entry and
fails when the behavior is weakened (see the narrowing mutants in
`internal/resumesmoke/mutant_test.go`).

## Section 7.5 — provider operations used by the smoke

| Clause | Call site | Evidence |
|---|---|---|
| probe identifies exact build; typed failures stay opaque | `Run.probe` → `Host.Call(OpProbe)` → `ProbeBuild` | Proven: `TestSmokeRowVerdicts` (pass rows), `TestSmokeProbeMismatchFails` |
| identify-session with session, provider, observation; closed confidence | `Run.identify` → `Host.Call(OpIdentifySession)` → `SplitIdentifyResult` | Proven: `TestSmokeRowVerdicts`, `TestSmokeStrongConfidenceFails` |
| resume over identity, workspace_paths, execution_profile, terminal, lease; SpawnPlan result | `Run.resume` → `ResolveMapping` → `Host.Call(OpResume)` → `DecodeSpawnPlan` | Proven: `TestSmokeRowVerdicts`, `TestSmokePiUnmappedVersionFails`, `TestSmokeThroughRealProviderProcess` |
| Deeper request-body member semantics | Adapter-owned | Bound: params boundary validates IDs, deadline, profile, workspace absoluteness, terminal geometry, lease shape, observation bounds; the adapter refuses the rest under 7.5 |

## Section 7.6 — quiescence (read-only precondition input)

| Clause | Call site | Evidence |
|---|---|---|
| valid safe proof consumed as precondition; unsafe stops graceful work | `Run.quiescence` → `DecodeQuiesceProof` | Proven: `TestSmokeSafeQuiescencePasses`, `TestSmokeUnsafeQuiescenceFails` |
| absent proof never fails the smoke (no mutation of its own) | `Run.quiescence` skip arm | Proven: `TestSmokeRowVerdicts` (quiescence skipped on every row) |
| proof-to-tuple provider/version binding | Quiescence owner | Bound: stated in `doc.go`; the smoke consumes valid-and-safe only |

## Section 7.7 — execution profiles (smoke consumption)

| Clause | Call site | Evidence |
|---|---|---|
| stored profile maps for the probed build or resume fails `profile_mapping_unavailable` | `Run.resume` → `ResolveMapping` | Proven: `TestSmokePiUnmappedVersionFails` (detail names the failure) |

## Section 7.8 — smoke as target-write precondition

| Clause | Call site | Evidence |
|---|---|---|
| bounded native-resume smoke evidence exists in the closed record shape | `Run` → `Record` → `VerifyRecord` | Proven: `TestSmokeRowVerdicts` (every row verifies), `TestSmokeRecordCarriesNoCapabilityClaim` |
| record carries checks with pass/fail/skipped, digests, timestamps; no secrets, no raw native references | `Record` encoding + `assertNoCanary` | Proven: every row test asserts the canary-free record; `TestSmokeTamperedRecordRefuses` |
| evidence pointer for the Section 13.14 registry object | `evidence_digest` integration point | Bound: the detailed record is what a registry entry points at; the registry object itself stays owned by `internal/sessadapter` (untouched) |

## Section 8.2 — native-store inventory

| Clause | Call site | Evidence |
|---|---|---|
| discovery over the declared native roots incl. XDG default, Pi override | `Run.bind` → `StoreRootFor` | Proven: `TestSmokeRowVerdicts` (store root asserted on pass rows) |
| native Windows rows bind with native separators | `StoreRootFor` + proof root rule | Bound: needs a Windows host; 4 rows skip loudly on POSIX with cells pinned by derivation |

## Section 8.3 — capability matrix (no advertisement)

| Clause | Call site | Evidence |
|---|---|---|
| smoke advertises no capability; Section 8.3 stays the only authority | Record shape (no capability/doctor surface) | Proven: `TestSmokeRecordCarriesNoCapabilityClaim` (closed 16-member shape) |

## Section 8.4 — support matrix (native-resume direction)

| Clause | Call site | Evidence |
|---|---|---|
| all 27 rows read their native-resume cell | `ResumeCell` | Proven: `TestResumeMatrixCoversEverySpecRow` (derived from pinned text, 27-row guard) |
| matrix agrees with the landed tuple gate on every row | `ResumeCell` vs `CheckResumeTuple` | Proven: `TestResumeMatrixAgreesWithTupleGate` |
| A rows pass end to end | `Run` | Proven: `TestSmokeRowVerdicts` (A subtests) |
| C rows gate the resume plan; U/? refuse with zero calls | `Run` cell arms | Proven: `TestSmokeRowVerdicts` (C/U/? subtests + call assertions) |
| unknown never rewritten as unsupported | `Cell` + `Verdict` vocabularies | Proven: distinct `unknown`/`unsupported` verdicts asserted per row |
| WSL2 and native Windows never collapse | `platformCell` distinct rows | Proven: separate subtests per row in derivation and verdict suites |

## Appendix B — version gates

| Clause | Call site | Evidence |
|---|---|---|
| Muse macOS arm64 accepts only the probed 0.1.0; newer behavior unsettled | `museCell` pin + `?` off-pin | Proven: `TestResumeMatrixCoversEverySpecRow` (pin + off-pin), `TestSmokeMuseOnePatchOffPinRefuses` |
| Antigravity resume only when the backend realm resolves | `Run.bind` → `VerifyIdentityDiscovery` realm rule | Proven: `TestSmokeAntigravityUnresolvedRealmFails`, gated Antigravity rows with resolved realm |

## Section 19.3 — promotion rule

| Clause | Call site | Evidence |
|---|---|---|
| smoke never promotes a gated cell; promotion needs published evidence | no promotion path in `Run`; `VerifyRecord` consistency | Proven: `TestSmokePromotionAttemptRefuses` (flipped + resigned), `TestSmokeInconsistentFailRefuses`; C detail cites Section 19.3 |

## Durability and crash behavior (evidence record mutates durable state)

| Clause | Call site | Evidence |
|---|---|---|
| no-replace + fsync install; identical retry reuses; disagreement quarantines | `Store` | Proven: `TestStoreInstallsVerifiedRecord`, `TestStoreReplayIsIdentical`, `TestStoreDisagreementQuarantines` |
| crash hooks recover | `StoreHooks` | Proven: `TestStoreCrashBeforeCreateAdmitsNothing`, `TestStoreCrashAfterWriteReplays` |
| real kill between write and fsync recovers byte-identical evidence | `Store` via re-executed helper | Proven: `TestStoreRealKillBetweenWriteAndSync` |
| identical inputs replay byte-identical records | `Run` determinism | Proven: `TestSmokeIdenticalInputsReplayIdenticalBytes` |
