# TASK-260917-10k118 conformance matrix — 30 of 30 AC rows driven

Scope: reconstruction of the reviewed v0.7.0 adoption (pin leaf rev1 + registry
leaf rev2-as-rederived-in-rev3) on trunk `7efe385`, plus this Story's two
verification tests. Rows P1–P12 are the accepted pin-leaf rows (verdict
`TASK-260916-2yzf5d_review-verdict-rev1.md`); rows R1–R16 are the accepted
registry-leaf rows (verdict `TASK-260916-n9r71p_review-verdict-rev2.md`,
rev3 figures 135/68/49/569); rows N1–N2 are this Story's new verification
tests. Every row is driven on the exact candidate tree through the named
production entry point. No new production behavior was added in the
reconstruction, so the row set is the union of the two accepted sets plus N1–N2.

## Pin rows (12 of 12)

| # | AC row | Probe (this session) → production call site | Result |
| - | ------ | -------------------------------------------- | ------ |
| P1 | Signed v0.7.0 tag verified from a fresh clone | `git tag -v v0.7.0` Good (tag `d4abe46f…795b3`, peeled `32b3f2ba…321834`, ECDSA `SHA256:V6JiK…`) in fresh `/tmp/speccheck` | PASS |
| P2 | SPEC.md digest + byte-identity | Tagged blob `c4903e60…1ddd86`, 1185291 bytes, 0 CR, sha256 `c6b2fe64…89ddcf` == LF-normalized; `cmp` identical to `SPEC.v0.7.0.md` | PASS |
| P3 | 64-row lock reproduces the §1.5 table | Independent extractor: 64 rows order-identical, name/urn identical, version-sets identical; 1 known order normalization (Provider protocol sorted, gate-required) → `specpin.VerifyV070` | PASS |
| P4 | Launch Plan row position + minors | Lock index 20 between Session record and Session event; v0.6.0→v0.7.0 diff: +1 ID, 0 removals, 5 minor bumps | PASS |
| P5 | 170-identifier inventory, digest-equal to v0.6.0 | `SectionInventoryV070` element-wise proof via committed `sections070_test.go` + `TestSectionIDResolvesV070Clauses` → `specpin.SectionInventoryV070` | PASS |
| P6 | History projections row-equal | v0.7.0→v0.6.0/v0.5.0/v0.4.3 `reflect.DeepEqual` to real locks (committed pin070 tests) → `Manifest.ContractsForRelease` | PASS |
| P7 | Stale pins cannot authorize v0.7.0 | `TestStalePinsCannotAuthorizeV070` + M4 mutant KILLED → `Manifest.ContractsForRelease` | PASS |
| P8 | Refusal census (12 arms) | Committed pin070/specdoc070 refusal tests through `specpin.VerifyV070`, `specdoc.ParseV070` (wrong tag/stale commit/wrong digest/forged inventory/trimmed delta/dropped row/forged namespace/fixture drift/capability claim/partial reads/cross-version/unknown release) | PASS |
| P9 | Narrowing mutants M1–M5 | Re-run on this tree: each KILLED by exactly its named subtest, full suites otherwise green → gate entries in P8 | PASS |
| P10 | Token-preserving mutant M2 | Delta-row Versions drop preserving tokens; full behavioral suite under plant; KILLED | PASS |
| P11 | Harmless control M0 | Operand-swap SURVIVED, suite green | PASS |
| P12 | No consumer re-pointed by the pin alone; no runtime claim | `Current`/`Verify`/`Load` still serve v0.5.0 (code read + historical suites green); README non-claim paragraph; `unsupported_capability_claim` refused by the gate → `specpin.VerifyV070` | PASS |

## Registry rows (16 of 16, rev3 figures)

| # | AC row | Probe (this session) → production call site | Result |
| - | ------ | -------------------------------------------- | ------ |
| R1 | 63 carried rows at v0.7.0 versions + launch-plan 1.0.0 | Lock diff (§P4) + `TestCurrentMatchesReviewedV070Catalog` → `catalog.Current()` | PASS |
| R2 | `catalog_gen.go` regenerated, `-check` green | `-adopted -check` exit 0; adopted==explicit==committed (`cmp` clean ×2); stale v0.6.0 refused exit 1 + `TestGenerateMatchesCommittedTypedCatalog` → `cataloggen.Generate`, `run()` | PASS |
| R3 | Validation command 22 is the `-adopted` string resolving to the v0.7.0 pair | Config position 22 + `TestConfiguredCRCatalogGateConsumesAdoptedAuthority` (accepts v0.7.0, refuses v0.6.0 lock) → `run()` | PASS |
| R4 | Current = V070 lock/doc/inventory/sections + ReleaseV070 | Baseline green + tracecheck exit 0 → `VerifyRepository`, tracecheck `run()` | PASS |
| R5 | Two legacy projections (v0.6.0 + v0.5.0) verified | `TestV060ProjectionMatchesHistoricalLock` (asserts launch-plan absent) + `TestV050ProjectionMatchesHistoricalLock` → `catalog.ForRelease` + legacy loop | PASS |
| R6 | 7 clause areas bound to real EPIC stories, unevidenced + honest gaps | Live board query 7/7 (names + backlog status) + `TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission` (6/6 + 2 splits) + 8 admission refusals → `VerifyRepository` + `VerifyAssignedSections` | PASS |
| R7 | New bindings refuse self-minted implementation claims | `TestAdoptedBindingsRefuseSelfMintedImplementationClaims` (3 declaration swaps + detached acceptance + test-code placement) → `VerifyRepository` | PASS |
| R8 | Non-Story owner refused | Digest-layer refusal + live board query (stated bound: prose-level, digest-enforced; same as rev2) | PASS |
| R9 | Projection digest re-pinned through review | Independent recompute (producer script + own Go mirror → `c4cd46bf…53473f6`) == `reviewedOwnershipCanonicalSHA256`; `TestVerifyRepositoryRefusesOwnershipGapDriftAtDigestLayer` → `VerifyRepository` | PASS |
| R10 | Historical registries byte-identical | 9/9 `git diff --exit-code` vs trunk (ownership/catalog/locks × v0.5.0/v0.6.0, both SPEC docs, `task-board.config.json`) (method, not unit test) | PASS |
| R11 | cigate v0.7.0/v0.4.3 roots, drift refused | `PinnedReleases==[v0.7.0 v0.4.3]` probe + `VerifyContractPreservation` green + drift table + `ForRelease("v0.7.1")` refused → `cigate.VerifyContractPreservation` | PASS |
| R12 | localstore stays on CurrentV050 | Unchanged package (not in candidate; full suite green) | PASS |
| R13 | pin.go/specdoc.go consumer comments true | Present-tense wording verified by read (prose, stated bound; reviewed bytes) | PASS |
| R14 | README/LOGBOOK truthful, no runtime claim for launch-plan/stdin/caller_launch_plan/environment_drift/curator-run | Text read (explicit disclaimers) + admission refusals for adopted sections → `VerifyAssignedSections` arms; `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` + new N2 pin the figures | PASS |
| R15 | tracecheck 49/569 with v0.6.0 + v0.5.0 histories | tracecheck exit 0 `contracts=64 … bindings=68 … clauses_discharged=49/569`; denominator 535→569 (+34 doc delta); numerator carried 49 → tracecheck `run()` | PASS |
| R16 | Historical + trunk tests unchanged and green | Full 27-command suite green incl. trunk's adopted tests; 19 files `cmp`-identical to accepted rev3 bytes; 5 merged files diff-reviewed (no trunk test lost) | PASS |

## New verification-test rows (2 of 2, this Story)

| # | AC row | Probe (this session) → production call site | Result |
| - | ------ | -------------------------------------------- | ------ |
| N1 | Registry re-derivation test: fails on any missing binding, unshifted clause line, or self-minted claim | `TestV070RegistryRederivesFromTrunkV060Registry`: 65/65 bindings + 132/132 cases carried, 49/49 clauses re-measured via production `sectionClauseInventory` + `quoteBeginsAtLine`, new sets exact → `specdoc.LoadV060/LoadV070`, `decodeOwnershipRegistry` | PASS |
| N2 | README measured-coverage pin test: subsection equals tool output byte-for-byte | `TestREADMEMeasuredCoverageMatchesTracecheckReport`: fenced line byte-equal to production `run()` output; 8 headline prose figures re-derived from `VerifyRepository` + reviewed registry | PASS |

Negative-evidence rows (gates attacked, not read): P8 (12 refusal arms),
P9–P10, R3 (stale-lock arms), R5 (legacy absence assertion), R6 (8 admission
refusals), R7 (5 self-mint refusals), R9 (gap-drift digest refusal), R11
(drift table + v0.7.1 plant), N1 (5 narrowing plants), N2 (2 narrowing + 1
token-preserving plant). Narrowing mutants: pin M1–M5 KILLED + M0 SURVIVED;
registry M1–M8 KILLED + M0 SURVIVED (both batteries re-run on this tree with
raw logs); new-test N1–N5 + R1–R2 KILLED, C0 + C0R SURVIVED. Token-preserving
mutants: pin M2, registry M7 (both re-run), new-test R3 (README intact,
production report changed, pin test + full behavioral suites fail).

Stated bounds (unchanged from the accepted revisions, plus one new finding):
owner Story-vs-Task enforcement is digest-level prose with the live board
query as the real check (R8); pin.go/specdoc.go comment truth is
prose-verified (R13); historical byte-identity is a method check, not a unit
test (R10); per-section README ratios are pinned by
TestCatalogSectionBindingCoverageIsExact… rather than re-derived in N2;
FINDING F1 — the reviewed v0.7.0 source normative-section group keeps
production `VerifyV060` while its cases moved to the v0.7.0 pair (v0.5.0→v0.6.0
precedent moved `Verify`→`VerifyV060`); green because the gate checks
declaration existence only; reviewed bytes are not rewritten in a
reconstruction, and N1 pins the carried reference literally.
