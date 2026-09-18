# TASK-260830-2056mm — independent review verdict, CR-TASK-260830-2056mm-3 revision 3

**Verdict: ACCEPTED (`accept_cr`, routed `integrating`).** No P1, no P2. Five
P3 advisories for a Story follow-up (none blocks the story_final Change
Request; every one is a measurement or prose gap around a gate that is
correct in production on my own instruments).

Reviewer: RUN-260918-20eca1 (claude-opus-5 max). Reviewed bytes: base
`c3aae73df906df1b5aa962c7bc3c16437dd920fa` (= `origin/main` by
`git ls-remote` at review time, so no refresh was owed), candidate tree
`747e23a25dc59866f38d04326aac0c26f7738086` (recomputed from the live Story
worktree through a temporary untracked-aware index: equal), patch sha256
`3a29aafc7dc85fcd33204bde5004579d90976c835691f94b4381a96a9ae1c578` (equal to
the CR record), 90 changed paths (`logs/00-identity.log`). Every probe ran in
isolated `git archive` copies of the exact tree under
`.temp/TASK-260830-2056mm/review-rev3/` (`cand/` tree-exact checks, `cand-h/`
shipped harnesses, `cand-m/` reviewer mutants, `cand-p/` probes; the
untested 1.0 GiB `.task-board/.resources` payload was stripped from the
copies, the board element files that `internal/specpin` reads were kept).
`cand-h` and `cand-m` are byte-identical to the candidate after every harness
and mutant pass (sha256 manifests before/after). The live Story worktree,
index, branch and HEAD were never touched. `PYTHONDONTWRITEBYTECODE=1`
everywhere; 0 `__pycache__`/`.pyc` in the tree or any copy. Every instrument,
raw log and subprocess exit is in `TASK-260830-2056mm_review-evidence-rev3.tar.gz`
(paths below are relative to that archive).

## 0. Rework verification (rev2 → rev3, graded first)

The rev2 → rev3 delta is 14 files (`git diff --stat f8adb161..747e23a2`):
`identity.go` (+202/−), `binding_arms_test.go` (new, 247 lines), `emit.go`
(+9 doc), `resolve.go` (+9 doc), `identity_test.go`, `recover_test.go`,
`emit_test.go`, `mutant_harness.py` (+6 rows), `TRACEABILITY.md`, README,
LOGBOOK, the adopted registry, its rederivation mirror and the pin.

| rev2 finding | What the tree does now | Graded |
| --- | --- | --- |
| P2-1 `protocol_version` major 1 unenforced | `ParseTerminalBinding` refuses unless `environ.CheckSemver && isProtocolMajorOne` (`identity.go:189`); `isProtocolMajorOne` = prefix `"1."` over an already-validated semver; `TestBindingProtocolMajorAgreesWithLandedTuple` agrees verdict-for-verdict with `terminalbackend.CheckVersionTuple` on 14 spellings. My independent rows `1`, `01.0.0`, `1.0.0 ` (trailing space), `10.0.0`, `11.0.0`, `1.x.0` all refuse at the protocol arm; `1.0.0`, `1.2.3`, `1.0.0-alpha`, `1.0.0+build.1` admit. My widening mutant `RV3-M3` (prefix `"1"`, admits 10/11/2^64-scale majors) KILLED ×2 by the committed suite | Closed |
| P2-1 `extensions == {}` unenforced | `isEmptyExtensionsObject` decodes the member through the landed strict frame and requires zero members (`identity.go:228,429`); string/number/array/null/one-key object/nested-empty object/duplicate-key object all refuse at the arm (`logs/12`); a whitespace-padded `{ }` admits and the identity recompute still matches — verdict agreement with the landed `ParseManifest` on the same bytes (`logs/14`) | Closed |
| P2-1 forked identity rule laundering numbers/nested duplicates | `bindingIdentity` now runs `decodeIdentityDocument` (size cap, UTF-8, lone-surrogate, `UseNumber`, per-depth duplicate refusal, depth cap, trailing data, non-object) and `refuseIdentityNumbers` BEFORE the omit-self marshal and JCS transform (`identity.go:265-411`). Numbers at depth 1–3, inside nested arrays, `0e0`, `1`, `2^64`; duplicates at depth 1–3 all refuse (`TestBindingIdentityRefusesHostileBytes` 12 rows + my `TestRV3_BindingIdentityDeepHostiles` 5 rows). The doc comment's false ordering claim is gone. **It is a verbatim mirror of the landed `terminalbackend` decode+walk+digest, not a reuse** — see P3-1 | Closed (P3-1 advisory) |
| P2-1 arms executed 7 of 30 | Coverprofile census over `identity.go` (`logs/84`): the parser's 32 refusal/passthrough return sites are executed 29 of 32 by the committed suite; the three unexecuted sites are the renamed-member half of the member-set arm (`:107`, subsumed by the per-member decode arms — my renamed-member probe refuses at `binding member set` anyway), trailing data inside one member value (`:118`, unreachable through the landed strict frame) and the identity-error passthrough (`:233`, unreachable once every member is a string/null/`{}`). Whole file: 38 of 52 sites; the unexecuted identity-decode sites are the UTF-8/surrogate/token-error arms the frame refuses first. My own 45-row independent enumeration of the §4.B table (wrong literals, nulls, arrays, objects, bools, uppercase/URN/v4 UUIDs, uppercase and 129-char backend IDs, leading-`v`/two-part/leading-zero semvers, empty/257-multibyte/lone-surrogate generation, empty/513-multibyte native reference, no-millis/offset/date-only timestamps, UUID/empty/bool/object/uppercase supersedes) refuses every row with a `*terminalbackend.Error`; the 256-two-byte-character generation and the 512-four-byte-character native reference ADMIT (character bounds, not bytes) | Closed |
| P2-1 landed-agreement with a numeric extension | `TestBindingIdentityVerdictAgreesWithLandedAdmission` runs a numeric extension value and a nested duplicate through both `ParseManifest` and `bindingIdentity`; my escapable-string/non-ASCII manifest variant digests identically on both sides | Closed |
| P3-1 pre-scan notes / LOGBOOK F2 / README bootstrap sentence | Harness notes now say the kill measures the Go error type the documented-redundant pre-scan adds; LOGBOOK rev3 entry carries an explicit "CORRECTION to the rev1 entry's F2 (appended, history preserved)" paragraph — 0 removed LOGBOOK lines; README says "the operation bodies (a documented redundancy — the landed parsers refuse the same code) … a plain refusal that persists nothing on the bootstrap binding". My `TestRV3_LandedBodyParsersRefuseEightForms`: 16/16 forms refused by the landed `ParseMutationContext`/`ParseStatusBody` with the pinned code | Closed |
| P3-2 `EmitResumed` binds neither checkpoint nor pair | Bound stated in `emit.go:68-75` and TRACEABILITY, pinned by `TestEmitResumedCheckpointBindingIsCallerBound` (fabricated checkpoint appends, fold reports it newest) — option B of the rework scope | Closed (as a stated bound) |
| P3-3 generation bound before the status read | `TestRecoverGenerationBoundBeforeStatusRead` committed, row `N-recover-generation` KILLED ×2 in my reruns | Closed |
| P3-4a row 57 / 5.2#18 | Row 57 relabelled STATED BOUND (payload census); registry 5.2 gap text names the driven half and the bound half | Closed |
| P3-4b 7.A citation | `TestAdmitDescriptorRefusesForbiddenIdentity` added to `terminal-instance-identity-exact` (registry + mirror) | Closed |
| P3-4c `resolve.go` "every failure" | "Every resolution failure …; the three wire-shape pre-check arms … return uncoded errors instead: defence in depth" | Closed |
| P3-5 deferral owner / `HandoffFailed` | Results, LOGBOOK and TRACEABILITY record the Story follow-up owner (no board task yet) and the `HandoffFailed` non-applicability | Closed |

All six rework items are done; construction reached `ready` on the first
attempt of this revision (validation trailer `required=30 green=30 failed=0
missing=0`).

## 1. What holds (verified myself on the exact tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt (empty list), `go build ./...`, `go vet ./...`, `GOOS=linux`/`GOOS=windows` builds, `GOOS=windows go vet` of the three Story packages, JSON validity of every tracked `internal/**/*.json`, `git diff --check base..candidate` | all exit 0 | `logs/02-static.log` |
| `go test ./... -count=1` on the exact tree | 41/41 `ok`, 3m01s | `logs/04-go-test-all.log` |
| `go test -race -count=1`: the five Story packages (termbind/axpane/terminstance/terminalbackend/traceability…) + the other 35 packages in a second bounded call | 41/41 `ok`, 0 `DATA RACE` | `logs/70-race-story.log`, `logs/71-race-rest.log` |
| `go test ./internal/termbind -count=1 -v -coverprofile` | 60 top-level PASS / 0 FAIL / 0 SKIP (265 PASS lines incl. subtests), 83.7% statements | `logs/83-termbind-verbose.log`, `logs/termbind.coverprofile` |
| The 17 bounded fuzz gates (commands 7–23, `-fuzztime=100x`) | 17/17 exit 0 | `logs/72-fuzz-gates.log` |
| `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` (command 25) | exit 0 | `logs/70-race-story.log` |
| CR construction suite (30 commands) | runtime-recorded `required=30 green=30 failed=0 missing=0` on the exact tree (accepted from the attached bounded `_rev3-validation.log`; commands 4 `-v`, 6 `-cover` and 29 `task-board validate` were not rerun by me — 4 and 6 are the same tests I ran non-verbose and under `-race`) | resource |
| Payloads vs the pinned text (`SPEC.v0.7.0.md` 1956–1985): `terminal.created` = terminal_binding_id, terminal_backend_id, implementation_version, protocol_version, evidence_ids[1..256] sorted unique; `session.resumed` = checkpoint_id, execution_profile standard\|yolo, profile_source_event_id\|null + the same five | identical to `axpane.TerminalCreatedPayload`/`ResumedPayload` and to the landed `canonicaljson` 4.0.0 registry (`core_records.go:574-575`); `"4.0.0": mergePayloadShapes(base, clone, directory)` is the same constructor expression as `"3.0.0"` with exactly the two overrides, and `validateSessionEventPayloadShapeCompleteness` panics on any drift against the catalog — `internal/canonicaljson` is untouched by this Story | source read |
| Append-boundary bounds through `axpane.Emit` (bypassing `ResolveEvidence`): 1 and 256 admit; 0, 257, unsorted, duplicate, non-digest refuse; a v4-shaped payload under `3.0.0` and a v3-shaped payload under `4.0.0` refuse, while the v3 shapes still admit under `3.0.0`; a v4 `terminal.created` carrying a v1 `terminal_id` or `backend` member refuses | holds | `TestRV3_AppendBoundaryEvidenceBounds`, `TestRV3_VersionSelectionAtAppendBoundary`, `TestRV3_V4EventCannotSmuggleV1Members` (`logs/12`) |
| Evidence resolution: 18 hostile resolution targets — the Binding object itself, a Provider Protocol 3 descriptor, an attach receipt, a raw generation string, a PID, a socket path, a named pipe, an endpoint object, a token, a credential, ANSI terminal output, a live-process fact, a lease record, a session event, a manifest with a `native_reference` extra, a manifest-schema object with a generation member, a probe schema inside an array, empty bytes — every one refuses `terminal_backend_manifest_probe_mismatch`; the Binding object substituted under the manifest's own ID refuses; the emitted event carries exactly the five members and neither the instance ID, the native reference nor either raw generation | holds | `TestRV3_EvidenceNeverResolvesToHostLocalOrSensitiveDocuments` (`logs/12`) |
| Identity: the eight forms refuse at `CheckInstanceIdentity`, `ParseTerminalBinding`, the landed `AdmitProviderDescriptor`, both landed body parsers, `axpane.Store.Bind` and `AttachStore.Attach` (committed rows 2, 4, 10–14, 48 + my `TestRV3_LandedBodyParsersRefuseEightForms`); whitespace-padded, newline-, NUL-, brace-wrapped and upper-case UUIDv7 spellings refuse on the story gate, the landed descriptor and the parser alike | holds | `logs/12` |
| Recovery: real SIGKILL between the durable bind + committed effect and the result → the ONE child (`TestRecoverCreateAfterKillRecoversChild`, PASS in my run); duplicate retry same verdict; changed op in the fold-derived open window → `idempotency_mismatch` with zero status reads; outside the window proceeds; a changed op in a closed window with a live session instance → `unavailable`/`status_first` carrying no binding; an unanchored session in the open window proceeds with exactly one read; the session-scoped read carries no instance/generation and the bound read carries exactly the recorded pair; an `active` report for another instance is refused by the landed `CheckStatusResult` (no verdict, no absence); a lying `identity_match` is refused by the engine; nil deps and a malformed deadline fail closed before any read | holds | `TestRV3_RecoveryEdges` (`logs/12`) |
| Authority: emission under a REMOTE winner (`not_owner`), under unverified ownership and under a lapsed grant refuses with zero appends; a takeover emits `terminal.created` under epoch 2 with a new digest and the superseded epoch-1 lease can no longer author; attach changes neither the event count nor the lease list and its receipt carries no lease/epoch/fencing/owner/checkpoint/event member; the relay transport refuses by the landed policy | holds | `TestRV3_EmitAuthorityRefusals`, `TestRV3_AttachIsInertToAuthority` (`logs/12`) |
| Story close: the ADOPTED `ownership.v0.7.0.json` is the one edited (trunk rows merged; the four upgraded bindings 4.B/4.D/5.2/7.A and the new `section:4.1`); all 54 test declarations named in the registry delta exist as committed tests (0 missing); `reviewedOwnershipCanonicalSHA256` RE-DERIVED by me through the production decoder: `b09b5b9f881a68969dc67dff54e1a31c3a82b7320e25fdc28ef9fce96b65bed5` (canonical bytes 167094) = the pin, ≠ the trunk pin `b4ae9091…` and ≠ the rev2 pin `89944b8c…`; `tracecheck` green with exactly `contracts=64 normative_sections=36 acceptance_cases=147 fixtures=33 compatibility_contracts=55 assigned_scopes=0` / `bindings=69 full=2 partial=9 sliver=9 unevidenced=45 unmeasured=4 unowned=7 clauses_discharged=63/574`; section runs 4.1 → 1/5 sliver, 4.B → 1/12 sliver, 4.D → 1/3 sliver, 5.2 → 3/18 sliver, 7.A → 1/2 partial (each exit 1 by design, texts quoted in `logs/24`); `TestREADMEMeasuredCoverageMatchesTracecheckReport` and `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` executed and PASS; README figures equal the printed report; README removes only the superseded coverage-figure lines (11 deletions, all in the measured-coverage prose) and LOGBOOK removes nothing (85 additions, newest-first) | holds | `logs/25`, `26`, `27`, `24` |
| Bounds and hygiene: `task-board.config.json` byte-identical to the base; the only trunk-moved paths (2fc6d50..c3aae73) inside the Story delta are README/LOGBOOK (additive) and the six traceability files (edited on top of c3aae73); every other path that differs from 2fc6d50 is trunk's own landing and equals c3aae73 in the candidate; 0 `__pycache__`/`.pyc`; the producer archive is a real gzip, task-scoped, 36 raw per-plant logs × 2 passes with subprocess exits, 4 race chunks with 0 `DATA RACE`, no foreign-task artefacts | holds | `logs/00`, `logs/80` |
| Inherited advisories (final leaf): the eight closure witnesses from 1geqhj rev4 and kkh1an rev4 execute and PASS on the exact tree | holds | `logs/60-leafclose-witnesses.log` |
| Shipped harnesses in the isolated pristine copy, verbose per-plant output with subprocess exits: termbind 36/36 (34 narrowing + 1 labelled arm-delete KILLED, `C-control` SURVIVED) ×2 identical verdict lists; axpane 56/56 ×2 identical; terminstance 122/122 ×2 identical; `cand-h` byte-identical afterwards | holds | `logs/harness/*-pass{1,2}-verdicts.log`, `*-verbose.log`, `cand-h-{before,after}.sha` |

## 2. Coverage statement

Producer claim: 65 of 66 rows driven, row 57 a stated bound. Re-derived
(`logs/82-ratio-check.log`): 66 numbered rows (1..66, contiguous, unique); 60
distinct cited top-level tests = the package's whole `go test -list`, no
uncited test, every one PASS in my verbose run. Rows labelled with bound
wording: 15 (drives `cliresult.VersionForCommand`, a tripwire), 57 (payload
census, not a reader drive), 66 (drives `EmitResumed`, pins the caller
boundary). **I measure 65 of 66 driven at the named entry**, matching the
claim. Rows 11 and 12 remain driven at wrapper entries with no production
caller, now honestly labelled a documented redundancy.

## 3. Findings (all P3; none blocks acceptance)

### P3-1 — `bindingIdentity` mirrors the landed identity rule instead of reusing it (architecture; rework brief item 1 said "preferably" reuse)

`decodeIdentityDocument`/`decodeIdentityValue`/`refuseIdentityNumbers`/
`bindingIdentity` (`identity.go:265-411`) are line-for-line copies of
`terminalbackend.decodeStrictObject`/`decodeCappedValue`/
`refuseIdentityNumbers`/`objectIdentity` (`manifest.go:387-690`) differing
only in the refusal code (`protocol_error` vs `mismatch`). The producer's
reason ("the terminalbackend identity rule is unexported") is true but
`internal/terminalbackend` is inside this Story's own change set, so an
exported entry was available. The copy is agreement-pinned (digest
equality on manifest-shaped documents incl. my escapable-string variant;
verdict agreement on a numeric extension and a nested duplicate; the
`{ }` whitespace case agrees with `ParseManifest`) and the hostile battery
pins the walk directly, so no laundering is reachable today — but two
copies of one repository-wide rule is exactly the class the epic keeps
paying for. Follow-up: export one identity entry from `terminalbackend`
and delete the copy.

### P3-2 — Unmeasured arm found by narrowing: the PROBE protocol-version arm of `checkEventBinding`

`RV3-M4` (`resolve.go:220`: drop `probe.ProtocolVersion != tuple.ProtocolVersion`;
the shipped row `N-resolve-binding` drops the EVIDENCE arm) SURVIVED the
whole committed suite ×2. Witness `TestRV3W_ProbeProtocolArmWithoutEvidence`
(a coherent zero-claim Manifest/Probe pair — `static_capability_claims: []`,
`capability_claims: []`, `evidence_ids: []`, all admitted by the landed
`AdmitProbe` — with the probe at `1.0.0` under an event tuple at `1.1.0`):
PASS pristine (`evidence binding` refusal), FAIL ×2 under the plant
(`logs/rv3-mutants/RV3-M4-*.log`). The arm is subsumed only when at least
one evidence object carries the protocol; the specification admits
zero-evidence probes (`capability_claims[0..16]`, `evidence_ids[0..256]`).
Commit the witness and a harness row.

### P3-3 — Unmeasured path found by narrowing: `EmitResumed` resolution on the carried-source path

`RV3-M7` (`emit.go`: run `ResolveEvidence` only when `!hasSource`)
SURVIVED the whole committed suite ×2: `TestEmitResolvesEvidence/resumed_unresolvable`
uses a null source and `TestEmitResumedRoundTrip/profile_source` never
plants a bad set. Witness `TestRV3W_ResumedWithSourceStillResolves`
(yolo + carried source + unresolvable evidence): PASS pristine, FAIL ×2
under the plant. Commit it and a row (a path-scoped narrowing of the
`D-emit-noresolve` class).

### P3-4 — Unstated bound: no production path mints a Binding 1.0.0 document or binds `terminal_binding_id` to one

`ParseTerminalBinding` has no production caller (grep: tests only), nothing
in the repository composes Binding → digest → `axpane.Store.Bind` → v4
event, and `facts.BindingDigest`/`terminal_binding_id` is caller-supplied
and grammar-checked only (`axpane.checkBindingFacts`). That is consistent
with the Story bounds (no `ax pane` wrapper process, no CLI) but neither
`doc.go` nor the TRACEABILITY bounds list says so, and the §7.A/§5.2
phrase "digest of the validated Terminal Instance Binding 1.0.0" reads as
if a validated binding stood behind every emitted digest. State the bound
(and the three context-dependent §4.B table equalities the parser cannot
check alone: implementation_version equal to the admitted Manifest/Probe,
protocol_version equal to the Probe, `supersedes_binding_id` null only for
the first binding) next to the parser rows.

### P3-5 — Doc-comment citation

`identity.go:262-264` credits `TestBindingIdentityAgreesWithLandedAdmission`
with the verdict-agreement half that lives in
`TestBindingIdentityVerdictAgreesWithLandedAdmission`.

Notes (not findings): `RV3-M6` (drop the probe-required half of the
manifest+probe cardinality) and `RV3-M8` (admit the PID form as
`client_id` at `Attach`) SURVIVED but are equivalent: the landed
`ParseProbe(nil)` refuses the same code (`terminal_backend_manifest_probe_mismatch
at document syntax`) and `Lookup`'s own identity gate refuses the client
before any write — the targeted committed subtests
(`…/missing_probe`, `…/client/pid`) still PASS under each plant
(`logs/52-equivalence-checks.log`).

## 4. Reviewer instruments

`probes/zz_review3_probe_test.go` (10 top-level probes, 124 PASS / 0 FAIL,
two passes with identical verdicts, `logs/12-probes-pass{1,2}.log`);
`probes/zz_review3_witness_test.go` (2 witnesses, PASS pristine
`logs/13`); `probes/zz_review3_digest_probe_test.go` (`logs/25`);
`probes/review3_mutants.py` (9 narrowings + kill control K + survive
control C against the WHOLE committed termbind suite, two passes with
identical verdicts, raw per-plant logs with exits, diffs and restoration
checks in `logs/rv3-mutants/`: KILLED — K, M1 schema_version, M2 native
513, M3 major prefix, M5 evidence-identity half, M9 window anchor;
SURVIVED — M4 non-equivalent (P3-2), M6 equivalent, M7 non-equivalent
(P3-3), M8 equivalent, control C); `logs/84-parser-arm-census.log`
(coverprofile census); `logs/82-ratio-check.log`; shipped-harness reruns
(`logs/harness/`); `logs/00`–`83`; `REVIEW-MANIFEST.txt` (sha256 of every
file in the archive).

## 5. DoD walk (live merged checklist)

Production entry points implement the scoped deliverable — yes (events,
recovery, identity refusal; the Binding parser now enforces its table).
Positive/negative/compatibility/recovery tests pass with logs — yes.
README/traceability without unsupported claims — yes after the rev3 prose
corrections; P3-4 names one bound still implied rather than stated.
Uncommitted in the Story worktree — yes (tree OID equal, HEAD =
checkpoint ca1c1d9). Ratio — 65 of 66 measured = claimed. Negative tests
that fail when the gate admits — yes for every gate the rows name.
Narrowing mutant per gate — shipped 36/56/122 KILLED ×2 with working
controls; my 9 narrowings: 6 KILLED ×2 incl. the kill control, 2
non-equivalent survivors with witnesses (P3-2, P3-3), 2 equivalent. No
source-text gate exists (stated bound). Lint/build — clean. Outcome
artefacts — attached, hygienic. Logbook — newest-first, additive, the F2
correction appended. Implementation matches AC — yes. Fits the
architecture — yes, with the P3-1 copy noted. Tests green — yes, incl.
`-race` and the fuzz gates. Gates attacked, not read — yes.

## Follow-up scope (for the orchestrator; not a rework of this revision)

1. Consolidate the identity rule: export one omit-self identity entry from
   `terminalbackend` and make `termbind.bindingIdentity` call it (P3-1).
2. Commit `TestRV3W_ProbeProtocolArmWithoutEvidence` and
   `TestRV3W_ResumedWithSourceStillResolves` (or equivalents) with harness
   rows (P3-2, P3-3).
3. State the no-Binding-writer bound and the three context-dependent §4.B
   equalities in `doc.go`/TRACEABILITY (P3-4); fix the doc-comment
   citation (P3-5).
4. The restore/terminate-stale follow-up the producer recorded still has
   no board task.

Verdict recorded by RUN-260918-20eca1 through `accept_cr(TASK-260830-2056mm,
revision=3, evidence=TASK-260830-2056mm_review-verdict-rev3.md)`; no product
edits, commits, checkpoints or integration were performed; the isolated
copies are deleted after attachment. `commit_ack` is not supplied.
