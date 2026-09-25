# TASK-260924-2zboyz — read-back manifest + validation report schemas — results (rev4)

## Outcome

Rework of CR rev3 answering the rev3 verdict's single finding
(`report-sibling-provenance-unbound`, `repeat-of: none`;
severity `bypass`). Package `internal/clonereadback` (new files
only, no landed file touched): Clone Read-Back Evidence Manifest
1.0.0 and Clone Validation Report 1.0.0 as closed schemas with
JCS identities, pinned to `internal/specdoc/SPEC.v0.7.0.md`
§13.14.2 (lines 10737-10777, prose 10580-10582, registry 170-171,
ReadAuthority purpose 3796-3797). By construction ("parse, don't
validate") the report entries accept ONLY sealed
`ValidatedReadBack` siblings minted by the read-back validator:
unexported fields, the two read-back entries as the only
constructors, and the zero value refused at both report entries
before any claim comparison. The generated
`testdata/__pycache__` blob is REMOVED and a no-bytecode hygiene
test pins its absence; the harness runs with `python3 -B`.
Candidate is the uncommitted tree on story tip `5e40826` (=
trunk `6d3bff9` + three signed story checkpoints). No registry
edit (`task_delta`).

## Verdict answers (finding → fix → regression → narrowing)

| Finding | Fix | Named regression tests | Narrowings (killed alone) |
|---|---|---|---|
| report-sibling-provenance-unbound | new sealed `ValidatedReadBack` (unexported fields; minted only by `Build/DecodeReadBackEvidenceManifest`); report entries take ONLY sealed siblings and refuse zero first via shared `checkSiblingsSealed` | `TestBuildReportRefusesUnsealedSiblings`, `TestDecodeReportRefusesUnsealedSiblings` (zero/zero, zero/sealed, sealed/zero × entry), `TestReportEntriesTakeOnlySealedSiblings` (AST: no struct/digest sibling param), `TestValidatedReadBackSealedConstruction` (AST+reflection: no exported fields/ctors + zero refused), `TestBuild/DecodeReportSiblingFromAnotherPlanIsBound` (cross-plan admission bound) | N-seal-build, N-seal-decode |
| rev3 notes | `testdata/__pycache__/*.pyc` removed | `TestNoGeneratedBytecode` | — (hygiene, not a gate) |

## AC coverage: 6 of 6 rows driven through production entries

| AC row | Production call sites | Named tests |
|---|---|---|
| Both schemas closed (reflection census, every member missing/extra/duplicated/miscased/wrong-type refused with literal code at every entry) | `DecodeReadBackEvidenceManifest`, `DecodeValidationReport` (+ `Build*` render exactness) | `TestMemberCensusExactSets`, `TestMemberCensusUnknownMissing`, `TestMemberCensusMiscased`, `TestMemberCensusWrongTypes`, `TestMemberCensusDuplicated`, `TestMemberCensusNoLiteralList`, `TestCensusShapeInventory` (16/5/23) |
| Ids recomputed independently | `Build*` seal, `Decode*` verify | `TestReadBackIdentityIndependent`, `TestReportIdentityIndependent` (JCS recompute via `github.com/gowebpki/jcs` in-test) |
| Staged/live modes cannot be relabeled | both read-back entries via `modeForAuthorityPurpose` + `checkModeAuthority` | `TestModeAuthorityGrid`, 4 relabel regressions, `TestModeRelabelRefused`, `TestModeBindsIdentity` |
| Every applicable check must pass for valid=true (whole-domain oracle) | `BuildValidationReport`, `DecodeValidationReport` via shared `decideValid` | `TestValidWholeDomainOracle` (128 × 2 × 6 = 1536 combos through both entries) |
| Every entry calls the owner validators (AST census) | all four entries → `sessadapter`/`clonebundle`/`environ`/`scalar`/`canonicaljson`; fidelity/plan row census empty; report siblings sealed-only | `TestOwnerRowCensusIsEmpty`, `TestOwnerReachability` (+ `checkSiblingsSealed` both-entries pin), `TestNoOwnerImports`, `TestExportedAPIInventory`, `TestNoForkedLiterals`, `TestNoForkedSelectors`, `TestErrorSeverityDecidedOnDecodedField`, `TestOwnerDelegation`, `TestReportEntriesTakeOnlySealedSiblings`, `TestValidatedReadBackSealedConstruction` |
| Each gate has a narrowing killed alone | 43 narrowings + 1 neutral control + 9 structural plants | harness `testdata/mutant_harness.py` (53 rows), battery × 2 runs, one raw log per plant per run |

Measured narrowing ratio: 18 of 18 counted gates carry ≥1
narrowing killed alone (rev3's 17 + A20 sibling seal); 2 gates
(owner-inner rules, extension value model) are justified
delegated bounds excluded from the ratio, with wiring pinned
behaviorally + structurally.

## Gate verdicts (all green, exit codes observed)

| Gate | Command | Exit | Evidence |
|---|---|---|---|
| package suite | `go test -count=1 ./internal/clonereadback/` | 0 (56 PASS + 18 subtests, 0 FAIL) | `package-verbose.log` |
| coverage | `go test -count=1 -cover ./internal/clonereadback/` | 0, 95.6% | `cover.log` |
| importer set | owner suites `environ scalar canonicaljson clonebundle sessadapter` | 0 (all five `ok`) | `importer.log` |
| vet | `go vet ./...` | 0 | `vet.log` |
| windows vet | `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `vet-windows.log` |
| gofmt | `gofmt -l internal/` | 0, clean | `gofmt.log` |
| determinism | `go test -count=3 ./internal/clonereadback/` | 0 | `determinism.log` |
| mutant battery run1 | `mutant_harness.py run run1` (53 rows) | 0, failures=0 | `run1/*.log` |
| mutant battery run2 | `mutant_harness.py run run2` (53 rows) | 0, failures=0 | `run2/*.log` |
| full suite shard1 (47 pkgs excl. traceability) | `go test -count=1 <47 pkgs>` | 1 (45 ok; `resumesmoke` + `tmuxserver` hit the 10m `go test` timeout under contention, zero import edge to this leaf) | `full-shard1.log` |
| contention reruns | each package alone | 0 (`resumesmoke` 164.8s, `tmuxserver` 386.0s) | `rerun-resumesmoke.log`, `rerun-tmuxserver.log` |
| full suite shard2 (traceability) | `go test -count=1` on both pkgs | 0 (2 ok, tracecheck 381.5s) | `full-shard2.log` |

Full-suite total: 49 packages, all green (45 in-shard +
2 contention-timeout packages green on alone rerun +
2 traceability). The two timeouts are `panic: test timed out
after 10m0s` under 47-way contention with no dependency path to
`internal/clonereadback` (`go list -deps` shows zero edge),
the same contention shape as the rev1 tracecheck episode; both
pass alone with no code change.

## Mutant summary (narrowing vs arm-delete per row)

All 43 N-rows are narrowings (single-site; the gate stays
present and admits exactly one forbidden member past the
weakened site); there are no arm-deletes. Conjunct-drop rows
(N-valid-drop-*, N-mode-auth-*, N-report-refs-*, N-seal-*)
drop one conjunct at one call site; the N-seal kill is the
fall-through to the downstream distinctness/slot refusal with a
changed literal (the documented N-closed precedent: proves the
seal gate's presence, first position, and literal). Killers ran
ALONE (harness `-run`).

| Mutant | Admits exactly | Killer (alone) | R1 | R2 |
|---|---|---|---|---|
| N-mode-archived | `archived` mode | `TestModeVocabularyOracle` | K | K |
| N-kind-native | `native` kind | `TestEvidenceKindVocabularyOracle` | K | K |
| N-identity-len | same-length-differing IDs | `TestNativeIdentityEqualityOracle` | K | K |
| N-valid-drop-staged | staged-false + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-live | live-false + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-marker | marker-false + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-identity | identity-false + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-binding | binding-false + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-resume | resume-false + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-generation | generation-false + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-ids | ID-mismatch + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-valid-drop-error | error finding + valid=true | `TestValidWholeDomainOracle` | K | K |
| N-heads-dup | duplicated heads (Build) | `TestParsedHeadIDsSweep` | K | K |
| N-evidence-dup | duplicated (kind, blob) | `TestEvidenceOrderSweep` | K | K |
| N-evidence-count | 65537 rows, both entries | `TestEvidenceCountEdges` | K | K |
| N-findings-count-build | 4097 findings at Build | `TestFindingsCountEdges` | K | K |
| N-findings-count-decode | 4097 findings at Decode | `TestFindingsCountEdges` | K | K |
| N-closed-readback | `zz_extra_member` in shape | `TestMemberCensusUnknownMissing` | K | K |
| N-closed-evidence | `zz_extra_member` in row | `TestMemberCensusUnknownMissing` | K | K |
| N-closed-report | `zz_extra_member` in shape | `TestMemberCensusUnknownMissing` | K | K |
| N-schema-neighbor | sibling URN as schema | `TestEnvelopeLiterals` | K | K |
| N-version-neighbor | `2.0.0` as version | `TestEnvelopeLiterals` | K | K |
| N-relabel-admit | docs valid under the other mode | `TestModeRelabelRefused` | K | K |
| N-parse-count-edge | 2^53 count at Build | `TestScalarGates` | K | K |
| N-mode-auth-build | cross-mode claims at Build | `TestBuildStagedAuthorityRefusesLiveClaim` | K | K |
| N-mode-auth-decode | resealed relabels at Decode | `TestDecodeStagedAuthorityRefusesResealedLiveClaim` | K | K |
| N-purpose-source | source_native as staged | `TestModeAuthorityGrid` | K | K |
| N-report-refs-build | equal/swapped refs at Build | `TestBuildReportRefusesEqualReadRefs` | K | K |
| N-report-refs-decode | equal/swapped refs at Decode | `TestDecodeReportRefusesEqualReadRefs` | K | K |
| N-seal-build | unsealed siblings past seal at Build | `TestBuildReportRefusesUnsealedSiblings` | K | K |
| N-seal-decode | unsealed siblings past seal at Decode | `TestDecodeReportRefusesUnsealedSiblings` | K | K |
| N-envelope-alias | `Schema` normalized before closure | `TestMemberCensusMiscased` | K | K |
| N-utf8-lone | lone-invalid bytes at Build | `TestBuildUTF8Reflection` | K | K |
| N-media-bound-build | 129-char media at Build | `TestMediaTypeStringEdges` | K | K |
| N-media-bound-decode | 129-char media at Decode | `TestMediaTypeStringEdges` | K | K |
| N-heads-item-build | 513-char heads at Build | `TestParsedHeadIDsSweep` | K | K |
| N-heads-count-build | 1025 heads at Build | `TestParsedHeadIDsSweep` | K | K |
| N-heads-item-decode | 513-char heads at Decode | `TestParsedHeadIDsSweep` | K | K |
| N-heads-count-decode | 1025 heads at Decode | `TestParsedHeadIDsSweep` | K | K |
| N-scalar-digest-build | `not-a-digest` at Build | `TestScalarGates` | K | K |
| N-scalar-digest-decode | `not-a-digest` at Decode | `TestDecodeScalarStrings` | K | K |
| N-uuid-build | `not-a-uuid` at Build | `TestScalarGates` | K | K |
| N-uuid-decode | `not-a-uuid` at Decode | `TestDecodeScalarStrings` | K | K |
| N-control-neutral | nothing (comment-only) | full suite | S | S |
| P-census-entry | ninth export carrying a row | `TestExportedAPIInventory` | R | R |
| P-owner-call-removed | removed owner call | `TestOwnerReachability` | R | R |
| P-forked-literal | re-added owner literal | `TestNoForkedLiterals` | R | R |
| P-severity-second | second Severity matcher | `TestErrorSeverityDecidedOnDecodedField` | R | R |
| P-guard-literal-list | 10-name literal list | `TestMemberCensusNoLiteralList` | R | R |
| P-seal-param-struct | struct sibling param | `TestReportEntriesTakeOnlySealedSiblings` | R | R |
| P-seal-param-digest | digest sibling param | `TestReportEntriesTakeOnlySealedSiblings` | R | R |
| P-seal-ctor | third seal constructor | `TestValidatedReadBackSealedConstruction` | R | R |
| P-seal-exported-field | exported seal field | `TestValidatedReadBackSealedConstruction` | R | R |

K = KILLED, S = SURVIVED (control, proving the harness
reports survivors), R = RED (structural plant). Survivor bounds:
`N-control-neutral` is behavior-neutral by construction.
`N-session-bound` (rev2) and `N-uint53-decode` (rev3) stay
WITHDRAWN after surviving: single-member widenings masked by the
sibling bound under the equality constraint and by the owner
uint53 type ceiling respectively; both logs ride the tar under
`withdrawn/`, edges stay proven directly, and both bounds are
excluded from the measured ratio.

## Stated bounds (out-of-contract rows for the handoff)

Brief gap first: the brief carries no surface table, so the
coverage-map precondition row has no surface rows to map; the
gate × entry table in TRACEABILITY covers every implemented gate
instead.

1. Report "matching tuple" needs the read manifests' observed
   tuples as cross-inputs — reconciliation, final leaf 1esv6u.
2. Plan/projected/provider/fidelity pairing and report/operation
   coherence — reconciliation, final leaf 1esv6u. (Staged/live
   read identity/slot binding AND the sibling seal ARE gated
   here; a validated cross-plan sibling admits, pinned by
   `TestBuild/DecodeReportSiblingFromAnotherPlanIsBound`.)
3. `parsed_event_count` / `structural_digest` carry no stated
   derivation — any uint53/digest admits (spec-silent, pinned by
   `TestParsedCountAndStructuralDigestAreStatedBounds`).
4. Blob descriptor agreement needs an object store; these entries
   take no fetch — unverified by construction.
5. Authority freshness/expiry lives where the caller chain mints
   the authority; entries consume Purpose only (composition
   bound).
6. Owner-inner rules and the extension value model are delegated
   to their owner suites (wiring pinned); single-member native-ID
   bound narrowings are masked under the equality constraint;
   decode-side uint53 ceiling is the owner type check — all
   excluded from the measured ratio with behavior pinned.
7. Native-ID member names in the read-back manifest follow the
   report table + adapter body spelling (spec states the rule
   without member names); census fails on drift.
8. The sibling seal assumes Go memory safety: `unsafe`/linkname
   forgery of `ValidatedReadBack` is out of scope by
   construction.

## Files (all new, under `internal/clonereadback/`)

`doc.go errors.go decode.go vocab.go authority.go sealed.go
readback.go report.go TRACEABILITY.md fixtures_test.go
members_test.go identity_test.go oracle_test.go gates_test.go
ownercensus_test.go structural_test.go utf8_test.go
authority_test.go readrefs_test.go sealed_test.go
testdata/mutant_harness.py`
