# Review verdict — TASK-260830-3uzfyn rev1: ACCEPT

Reviewer run `RUN-260917-085c1d` (muse-spark xhigh), independent review of
Change Request `CR-TASK-260830-3uzfyn-1` revision 1, published by
`RUN-260917-0605fd`. Normative authority: pinned
`internal/specdoc/SPEC.v0.6.0.md` §§2.4, 7.7 (verified textually identical
to the task-cited v0.5.0 for these sections — the producer's diff claim
reproduces: `diff` of the two section ranges is empty).

**Verdict: ACCEPT.** All AC rows independently driven (16 of 16), every
gate attacked with narrowing mutants (producer 20/20 KILLED in two
executions each, plus 4/4 reviewer-owned kills on unmutated arms with a
surviving control), the witness instrument attacked three ways (all RED),
crash/idempotency reproduced, full suite + race + gates green, hygiene
clean. No change request. Method: isolated copies only — the live Story
worktree was never edited by this run (all plants/probes ran under
`.temp/TASK-260830-3uzfyn/rev-*`); `go test` binaries only read it.

## 1. AC coverage — 16 of 16 rows driven (ratio, with call sites)

Each row's production entry and named committed test were confirmed present
and passing by independent runs this session (targeted `-run` masks plus
the full package suites):

| Row | Requirement | Production call site | Named test(s) — all PASS rerun |
|-----|-------------|----------------------|-------------------------------|
| R1 | vocabulary exactly `standard\|yolo` | `sessprofile.ResolveCreationProfile` | `TestResolveCreationProfile` |
| R2 | `ax start --profile` → creation profile (+ stated bound: no spec'd absent default, empty refuses) | same | same |
| R3 | `set-profile` appends under current lease | `Transactor.SetProfile` | `TestSetProfileAppendsFirstChangeOnEmptyChain`, `...UnderChainHead`, `...DerivesFromFromEffectivePair` |
| R4 | task-board profiles (launch/bridge/bundle) | `CheckBridgeProfile`, `CheckLaunchPair`, `BundlePair`/`CheckBundlePair` | `TestCheckBridgeProfile`, `TestCheckLaunchPair`, `TestFixtureProfileTBTakeover/TBResume/TBFork` |
| R5 | provider argv from mapping; env profile-independent (+ bound: argv only) | `provhost.ResolveMapping`, `ProjectLaunchArgv` | `TestResolveMappingTable`, `TestProjectLaunchArgvYOLO/Standard...`, bounds agreement |
| R6 | effective non-secret state persists | `SetProfile`, `Derive`, `MintChangeEvent` (+ `provhost\|decl\|unrestrictedTokens` census row) | append/mint/derivation tests |
| R7 | session-head derivation | `Derive` | `TestDeriveEmptyChain/NewestChangeWins/SecondChange/ChangeBack/InertPairs/AcrossSuccessorLease`, `TestProjectDerivesSessionHead...` |
| R8 | checkpoint-closure pins C1 | `DeriveForHeads` | `TestDeriveForHeadsExcludesLaterChange/WithoutChangeIsCreation/IgnoresPostClosureCorruption` |
| R9 | losing/ambiguous/corrupt inputs refuse | `Derive`, `DeriveForHeads`, `DecodeRecord`, `DecodeEvent` | `TestDeriveRefusesLosingLeaseAndAmbiguity/FirstEventViolations/MalformedChangePayload/BadRecord`, heads/decode refusals |
| R10 | confirmation exactly for yolo | `MintChangeEvent` via `SetProfile` | `.../unconfirmed_yolo`, `...AdmitsUnconfirmedStandard`, `TestSetProfileRefusals/unconfirmed_yolo` |
| R11 | from==to refuses | `MintChangeEvent` via `SetProfile` + replay check | `.../from_equals_to`, `TestSetProfileRefusesNoOpChange` |
| R12 | idempotent retry; crash windows | `SetProfile` via sessrepo append | `TestSetProfileRetryReplaysCommittedChange`, both `TestSetProfileCrash*` (each rerun 2× green this session) |
| R13 | exact-build mapping; `profile_mapping_unavailable` exit 6 + details; Pi 0.73.1 + disclosure | `ResolveMapping` | `TestResolveMappingTable`, `TestResolveMappingRefusals` (code, exit 6, provider/version/profile details all asserted) |
| R14 | seven fixtures, exact P0/E1/C1 | derivation + projection entries | all 7 `TestFixtureProfile*` PASS rerun this session |
| R15 | integrity failures before activation | `CheckLaunchPair/CheckResumedPair/CheckBundlePair/CheckFinalizeParams` | `TestFixtureIntegrityFailures` + N1/N2/N3 bundle cases |
| R16 | no unsupported capability | README section + cigate gate | `TestRealREADMECarriesNoPositiveClaim` PASS rerun |

## 2. Independent probes (reviewer-owned, isolated copy)

Five probes in `rev-probe/` (sources + raw logs in the evidence tarball),
all PASS, all through production entries:

- Changed-envelope answer: **REFUSED, never appended.** After a commit, a
  retry with a changed instant or changed author gets `ErrProfileUnchanged`
  and the chain stays length 1 (`TestReviewerChangedEnvelopeRefuses...`).
  Why: `sameEnvelope` (setprofile.go) requires identical
  to/lease/author/instant; from is deliberately excluded (derived, not
  requested). A changed member is a genuine no-op, not a replay.
- Fencing precedes replay (own seam): identical envelope under a
  greater-epoch acting lease refuses `ErrDivergentLease`, chain stays 1
  (`TestReviewerFencingPrecedesReplay`). Replay never bypasses fencing.
- Pi one patch off (reviewer-chosen **0.73.2**, distinct from the
  producer's 0.74.0): `profile_mapping_unavailable`, exit 6, all three
  details present; pinned 0.73.1 still resolves.
- Base argv already carrying the exact flag refuses; a foreign provider's
  flag refuses; clean yolo base appends exactly once.
- One past every §5.1 bound refuses: 129 elements, 4097-byte element,
  65537-byte total (reviewer-built vectors).

## 3. Mutation evidence

- Producer battery rerun **twice**: run 1 (full, with controls) 20/20
  KILLED + harmless control SURVIVED + control-before/after green; run 2
  (all 20 plants again) 20/20 KILLED. Stable denominators, per-plant raw
  logs in `rev-run1/`, `rev-run2/`.
- Reviewer-owned plants on arms the producer never mutated — **4/4 KILLED**,
  control SURVIVED (`rev-own/`, harness `rev_mutants.py`):
  - `R-closure-limit-plus-one` (closure validation limit widened by one
    event → post-closure corruption enters validation and refuses) killed
    by `TestDeriveForHeadsIgnoresPostClosureCorruption`;
  - `R-first-seq-admits-2` (first-event gate admits exactly the
    sequence-2 opener) killed by `.../sequence_is_not_1`;
  - `R-pi-equivalence-dropped` (`Equivalent: false`) killed by
    `TestResolveMappingTable/pi/{yolo,standard}`;
  - `R-mint-confirm-from-standard` (confirmation skipped for exactly the
    from-value the refusal test drives) killed by `.../unconfirmed_yolo`.
- Witness-instrument attacks — **3/3 RED** (`rev-witness/`): adjacent
  same-code detail swap, one-token `!` swallow, Pi-pin deletion. Each
  reddens `TestEveryArmWitnessRefusesAtTheProductionEntry`; deletion also
  orphans the witness both directions. Census claim holds: 12 new derived
  arms, floor 217 → 229, seventh constructor `failMappingUnavailable`.
- Token-preserving requirement: `N-mapping-pi` keeps the `0.73.1` token
  while admitting exactly `0.74.0`, and the harness executes the
  behavioral suite, never a static checker. No gate here inspects source
  text beyond the version token, which that plant covers.

## 4. Bounds, registration, hygiene, gates

- B1 verified by grep: no named profile registry in the pinned spec —
  only `set-profile NAME PROFILE` (NAME is the session) and the
  `standard|yolo` enum. B2–B8 stated bounds accepted as declared; B7's
  argv-inspection residual is correctly scoped to execve argv.
- `profile_mapping_unavailable` is a legitimate pre-registered class
  (catalog entry since v0.4.3, exit 6 per §15.3); the change mints no
  other class — the protocol.go delta is the single constructor, and the
  runtime audit pins the closed 7-code set. No traceability/catalog row
  touched (final-leaf owned).
- Attestation bound holds: production uses only
  `canonicaljson.CalculateObjectIdentity` (mint.go); `Verify` appears
  only in comments/tests.
- Hygiene: changed paths exactly the 24 CR paths (6 tracked + 18 new),
  no `__pycache__`/`.pyc`, no `task-board.config.json`, no stray plants;
  live tree unmutated by review.
- Gates observed this session (real exits): `gofmt -l internal` clean;
  `go vet` on the four packages exit 0; `go build ./...` exit 0;
  `go test ./... -count=1` exit 0 — **27 ok, 0 fail**;
  `go test -race` on sessprofile/provhost/sessrepo/secprim exit 0;
  `tracecheck` exit 0; `go generate ./internal/catalog` +
  `git diff --exit-code` clean.

## 5. Findings (no change request)

Two reviewer-brief suggestions proved unusable as stated, and were
replaced with observable equivalents (documented in `rev_mutants.py`):

1. The suggested `compareLease` `<`→`<=` plant is **vacuous**: it sits
   inside the outer `a.Epoch != b.Epoch` guard, so `<=` is
   behavior-identical to `<` (verified SURVIVED, correctly). The order
   gate is instead attacked observably by `R-first-seq-admits-2`.
2. The suggested "confirmation skipped when from is yolo" plant is
   **masked**: the earlier from==to arm still refuses yolo→yolo, so the
   suite would stay green. `R-mint-confirm-from-standard` targets the
   tested member instead. Both maskings reflect correct arm ordering in
   the implementation, not defects.

No P1/P2/P3 issues. The candidate matches the AC, fits the
reducer-plus-projection architecture (pure derivation, transaction
through the landed append path, mapping owned by provhost), and its
evidence is positive-path-complete with genuine negative depth.

## Evidence attached

- `TASK-260830-3uzfyn_review-verdict-rev1.md` (this file).
- `TASK-260830-3uzfyn_review-evidence-rev1.tar.gz`: `rev-run1/`
  (20 KILLED + controls, `mutants.json`), `rev-run2/` (20 KILLED second
  execution), `rev-own/` (4 KILLED + SURVIVED control, `rev-mutants.json`),
  `rev-witness/` (3 RED attacks, `rev-witness.json`), `rev-probe/` (5
  reviewer probes + raw logs), and both reviewer harnesses. Isolated
  source copies excluded for size; every log carries its subprocess exit.
