# TASK-260830-2zvo8m results — native-resume smoke framework

Final leaf of STORY-260830-315721 (provider-identity-profile-and-resume).
Pinned authority: v0.6.0 (`internal/specdoc/SPEC.v0.6.0.md`), Sections
2.4, 5.5, 7.5-7.8, 8, Appendix B, Section 19.3.

## Deliverable

New package `internal/resumesmoke`:

- `Run` (`smoke.go`) drives probe, identify-session, discovery-bind,
  quiescence-precondition, and resume-plan through
  `provhost.Host.Call` and records one closed Native Resume Smoke
  record (`urn:ax:schema:native-resume-smoke 1.0.0`).
- `ResumeCell` (`matrix.go`) reads the Section 8.4 native-resume
  direction with Appendix B version gates.
- `VerifyRecord` (`record.go`) enforces closed shape, omit-self
  digest, and verdict consistency.
- `Store`/`Load` (`store.go`) install content-addressed, no-replace,
  fsynced evidence.
- Two zero-refusal-site provhost replay accessors (`ProbeBuild`,
  `SplitIdentifyResult`) consume validated members, never the body.

Verdicts: `pass` (A cell, all checks pass), `fail` (a check failed),
`gated` (C cell, resume plan skipped with the Section 19.3 citation),
`unsupported` (U cell), `unknown` (? cell). U/? refuse before any
adapter call. No smoke API promotes a gated cell; passing smoke
advertises no capability (no capability map, no doctor surface).

## AC coverage: 16 of 16 rows driven through production entries

| # | Behavior | Production call site | Named test |
|---|----------|----------------------|------------|
| 1 | Exact-version probe check | `Run.probe` via `Host.Call(OpProbe)` + `ProbeBuild` | `TestSmokeRowVerdicts`, `TestSmokeProbeMismatchFails` |
| 2 | Identify-session at exact confidence | `Run.identify` via `Host.Call(OpIdentifySession)` + `SplitIdentifyResult` | `TestSmokeRowVerdicts`, `TestSmokeStrongConfidenceFails` |
| 3 | Discover/read over declared native roots | `Run.bind` via `StoreRootFor` + `VerifyIdentityDiscovery` | `TestSmokeRowVerdicts`, `TestSmokeCrossTupleProofFails` |
| 4 | Resume plan over identity, workspaces, profile, terminal, lease | `Run.resume` via `ResolveMapping` + `Host.Call(OpResume)` + `DecodeSpawnPlan` | `TestSmokeRowVerdicts`, `TestSmokePiUnmappedVersionFails` |
| 5 | Quiescence as read-only precondition input | `Run.quiescence` via `DecodeQuiesceProof` | `TestSmokeSafeQuiescencePasses`, `TestSmokeUnsafeQuiescenceFails` |
| 6 | A-cell positives | `Run` + `ResumeCell` | `TestSmokeRowVerdicts` (A subtests) |
| 7 | C-cell gated outcomes, never promoted | `Run.resume` cell gate | `TestSmokeRowVerdicts` (C subtests), `TestSmokePromotionAttemptRefuses` |
| 8 | U-cell unsupported refusals, zero adapter calls | `Run` cell gate | `TestSmokeRowVerdicts` (qwen subtests) |
| 9 | ?-cell unknown refusals, zero adapter calls | `Run` cell gate | `TestSmokeRowVerdicts` (future/muse-windows subtests), `TestSmokeMuseOnePatchOffPinRefuses` |
| 10 | Probe/version mismatch fails closed | `Run.probe` drift arm | `TestSmokeProbeMismatchFails` |
| 11 | Unbound discovery proof refuses | `Run.bind` | `TestSmokeCrossTupleProofFails` |
| 12 | Non-A resume plan refuses | `Run.resume` cell arm | `TestSmokeRowVerdicts` (C subtests assert skipped + gated) |
| 13 | Passing smoke advertises no capability | Record shape (closed, no capability surface) | `TestSmokeRecordCarriesNoCapabilityClaim` |
| 14 | Appendix B gates (Muse pin, Antigravity realm) | `ResumeCell` + realm rule via `VerifyIdentityDiscovery` | `TestSmokeMuseOnePatchOffPinRefuses`, `TestSmokeAntigravityUnresolvedRealmFails` |
| 15 | Durable evidence: no-replace + fsync, crash, byte-identical replay | `Store`/`Load` | `TestStoreDisagreementQuarantines`, `TestStoreRealKillBetweenWriteAndSync`, `TestSmokeIdenticalInputsReplayIdenticalBytes`, `TestStoreReplayIsIdentical` |
| 16 | Tamper-evidence and verdict consistency | `VerifyRecord` | `TestSmokeTamperedRecordRefuses`, `TestSmokePromotionAttemptRefuses`, `TestSmokeInconsistentFailRefuses`, `TestSmokeWhitespaceVariantRefuses` |

Row verdicts derive all 27 Section 8.4 rows from the pinned
specification text (`TestResumeMatrixCoversEverySpecRow`), not from a
hand-typed literal; `TestResumeMatrixAgreesWithTupleGate` pins
matrix/gate agreement. One available row additionally runs end to end
through a real provider child process
(`TestSmokeThroughRealProviderProcess`, byte-identical to in-process).

## Narrowing mutants: 10 killed, 1 SURVIVED control

Committed harness `TestSmokeMutantsAreKilled`; each mutant runs in a
private module copy, so the harness never touches the checkout.

| Mutant (narrowing) | Killed by |
|---|---|
| resume-gate-admits-conditional | `TestSmokeRowVerdicts/gemini-0.54.4-wsl2-amd64` |
| probe-admits-version-drift | `TestSmokeProbeMismatchFails` |
| identify-admits-strong | `TestSmokeStrongConfidenceFails` |
| bind-admits-antigravity | `TestSmokeAntigravityUnresolvedRealmFails` |
| digest-scoped-to-pass | `TestSmokeTamperedRecordRefuses` |
| consistency-admits-fail | `TestSmokeInconsistentFailRefuses` |
| disagreement-admits-same-length | `TestStoreDisagreementQuarantines` |
| quiescence-admits-unsafe-codex | `TestSmokeUnsafeQuiescenceFails` |
| mapping-admits-yolo | `TestSmokePiUnmappedVersionFails` |
| matrix-admits-muse-offpin | `TestResumeMatrixCoversEverySpecRow` |
| harmless-comment-control | SURVIVED as required |

No production source-text gate exists (the spec-derivation test reads
the pinned authority document, which `specpin` guards), so the
token-preserving-mutant row is not applicable; stated, not skipped.

## Stated bounds

- Native-Windows bind rows (codex, claude, gemini, pi) need a Windows
  host — store roots join with the host separator while the proof rule
  is platform-native — and skip loudly on POSIX. Their cells stay
  pinned by the derivation test on every host.
- Quiescence is consumed as valid-and-safe only; binding the proof's
  own provider/version to the tuple belongs to the quiescence owner.
- Record integrity detects tampering and naive/resigned promotion,
  not a forgery that recomputes both the digest and a consistent
  check list (local evidence, no adversary).

## Story-close items

- Ownership bindings: 2.4 full 4/4, 5.5 sliver 1/3, 7.7 partial 3/4,
  8 sliver 4/12; 12 new acceptance cases; pin re-pinned.
  `tracecheck`: 113 cases, 29/463 clauses. `-section 2.4` admitted
  alongside 6.2; README figures and pin tests updated deliberately.
- README: new `Native-resume smoke` section (`internal/sessprofile`
  section already existed). LOGBOOK entry added.
- `task-board.config.json` equals HEAD after the trunk refresh; the
  refresh replayed both story checkpoints with signed continuation
  (LOGBOOK order conflicts resolved trunk-first, content preserved).

## Validation (all observed, post-refresh tree unless noted)

- `gofmt -l internal`: clean (448 files)
- `go vet ./...` and `GOOS=windows go vet ./...`: pass
- `go build ./...`: pass
- `tracecheck`: pass (113 cases, 29/463)
- `cataloggen -adopted ... -check` (new command 21): pass
- `cigate` pinned gates: pass
- `go test ./... -count=1`: exit 0, 28 ok, 0 fail (pre- and post-refresh)
- `go test -race` (touched 3 + remaining 25): exit 0, all ok
  (pre-refresh run; the refreshed tree carries identical package
  content and all focused suites re-ran green post-refresh)
- `go test ./... -cover`: exit 0; resumesmoke 77.0% statements

Candidate left UNCOMMITTED in the Story worktree for handoff snapshot.
