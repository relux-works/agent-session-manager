# TASK-260830-3uzfyn conformance matrix — profile resolution and persistence

Pinned source: `agent-session-manager-spec` v0.6.0, Sections 2.4, 5.1-5.2,
5.4, 5.5+8 (tuple authority), 7.7, 9.3. Sections 2.4 and 7.7 are
textually identical in v0.5.0 (verified by diff; see results).

Coverage: **16 of 16 AC rows driven** through the named production
entry by the named committed test. Two stated bounds ride inside
driven rows (R2 absent default, R5 env); they are declared below,
not silently skipped.

| Row | Requirement | Production call site | Named test(s) | Verdict |
|-----|-------------|----------------------|---------------|---------|
| R1 | Named profiles resolve: the vocabulary is exactly `standard\|yolo` and creation values admit only it | `sessprofile.ResolveCreationProfile` | `TestResolveCreationProfile` (both pass; `""`, case, padding, foreign refuse) | driven |
| R2 | CLI override `ax start --profile` maps to the Session Record creation profile | `sessprofile.ResolveCreationProfile` (CLI leaf passes the explicit flag) | `TestResolveCreationProfile`; bound: no spec'd absent default, so empty refuses here and defaulting belongs to the CLI-surface leaf | driven + stated bound |
| R3 | CLI `ax session set-profile` appends a change under the current lease, reporting previous/new/event | `sessprofile.Transactor.SetProfile` | `TestSetProfileAppendsFirstChangeOnEmptyChain`, `TestSetProfileAppendsUnderChainHead`, `TestSetProfileDerivesFromFromEffectivePair` | driven |
| R4 | Task-board profiles: launch `profile:` member, bridge resume `--profile`, `task_board.launched` pair, bundle pair | `sessprofile.CheckBridgeProfile`, `CheckLaunchPair`, `BundlePair`/`CheckBundlePair` | `TestCheckBridgeProfile`, `TestCheckLaunchPair`, `TestBundlePairProjection`, `TestFixtureProfileTBTakeover`, `TestFixtureProfileTBResume`, `TestFixtureProfileTBFork` | driven |
| R5 | Provider argv resolves from the mapping; env is profile-independent | `provhost.ResolveMapping`, `provhost.ProjectLaunchArgv` | `TestResolveMappingTable`, `TestProjectLaunchArgvYOLO`, `TestProjectLaunchArgvStandardOmitsEveryFlag`, `TestProjectedArgvBoundsAgreeWithSpawnPlan`; bound: the projection assembles argv only and never reads or rewrites `env_names`/`env_literals` | driven + stated bound |
| R6 | Effective non-secret profile state persists: creation member, `profile.changed` chain, derived pairs | `sessprofile.Transactor.SetProfile`, `Derive`, `MintChangeEvent` | set-profile append tests, `TestMintChangeEventEmitsAttestedBytes`, derivation tests; non-secret pinned by the reviewed `provhost\|decl\|unrestrictedTokens` argv-words census row | driven |
| R7 | Session-head derivation: creation while unchanged, newest change wins, change-back keeps its source, pair events stay inert | `sessprofile.Derive` | `TestDeriveEmptyChainIsCreationPair`, `TestDeriveNewestChangeWins`, `TestDeriveSecondChangeSupersedesFirst`, `TestDeriveChangeBackRestoresCreationValueWithSource`, `TestDeriveLaunchResumeForkPairsAreInert`, `TestDeriveAcrossSuccessorLease`, `TestProjectDerivesSessionHeadOverRepository` | driven |
| R8 | Checkpoint-closure derivation pins C1: later changes excluded, pre-change closure is creation, post-closure corruption ignored | `sessprofile.DeriveForHeads` | `TestDeriveForHeadsExcludesLaterChange`, `TestDeriveForHeadsWithoutChangeIsCreation`, `TestDeriveForHeadsIgnoresPostClosureCorruption` | driven |
| R9 | Losing, ambiguous, and corrupt inputs refuse: stale/divergent lease, repeat/skip sequence, bad predecessor, foreign session, bad version, malformed change, unknown/dangling heads | `sessprofile.Derive`, `DeriveForHeads`, `DecodeRecord`, `DecodeEvent` | `TestDeriveRefusesLosingLeaseAndAmbiguity`, `TestDeriveRefusesFirstEventViolations`, `TestDeriveRefusesMalformedChangePayload`, `TestDeriveRefusesBadRecord`, `TestDeriveForHeadsRefusals`, `TestDecodeRecordRefusals`, `TestDecodeEventRefusals` | driven |
| R10 | Operator confirmation required exactly when changing to `yolo` | `sessprofile.MintChangeEvent` (via `Transactor.SetProfile`) | `TestMintChangeEventRefusals/unconfirmed_yolo`, `TestMintChangeEventAdmitsUnconfirmedStandard`, `TestSetProfileRefusals/unconfirmed_yolo`, `TestDeriveUnconfirmedChangeIsAuthoritative` (publication owns it) | driven |
| R11 | A change from a profile to itself refuses | `sessprofile.MintChangeEvent` (via `Transactor.SetProfile` + replay check) | `TestMintChangeEventRefusals/from_equals_to`, `TestSetProfileRefusesNoOpChange` (no source, foreign author, mismatched instant) | driven |
| R12 | Retry is idempotent; crash windows behave: prepare fault admits nothing and the retry appends, commit fault counts as committed and the retry replays | `sessprofile.Transactor.SetProfile` through the `sessrepo` append path | `TestSetProfileRetryReplaysCommittedChange`, `TestSetProfileCrashBeforeDurableWriteAdmitsNothing`, `TestSetProfileCrashAfterDurableWriteReplaysOnRetry` | driven |
| R13 | Mapping resolves against the exact probed build; unmappable provider/version refuses `profile_mapping_unavailable` (exit 6, provider/version/profile details); Pi pins 0.73.1; Pi equivalence disclosed for both profiles | `provhost.ResolveMapping` | `TestResolveMappingTable`, `TestResolveMappingRefusals` (code, exit, details asserted) | driven |
| R14 | The seven Section 2.4 fixtures carry exact P0/E1/C1 pairs | derivation + projection entries per fixture | `TestFixtureProfileDirectTakeover`, `TestFixtureProfileDirectResume`, `TestFixtureProfileDirectFork`, `TestFixtureProfileTBTakeover`, `TestFixtureProfileTBResume`, `TestFixtureProfileTBFork`, `TestFixtureProfilePIEqualMapping` | driven |
| R15 | Integrity failures refuse before activation: P0 after C1, missing E1, inconsistent bundle | `sessprofile.CheckLaunchPair`, `CheckResumedPair`, `CheckBundlePair`, `CheckFinalizeParams` | `TestFixtureIntegrityFailures` plus the N1/N2/N3 bundle cases in `TestBundlePairProjection` | driven |
| R16 | No unsupported capability advertised: no `ax` surface, no doctor result, no runtime claim | README negative-claim section; `internal/cigate` gate | `TestRealREADMECarriesNoPositiveClaim` (full suite green), new section ends with the explicit no-surface sentence | driven |

## Negative-test and narrowing-mutant census

Every gate below ships a narrowing mutant (gate present, weakened to
admit exactly one member of the rejected class) and the named test
fails. Harness: `internal/sessprofile/testdata/mutate_profile.py`
(full battery 20/20 killed, harmless control survived; raw logs in
the evidence tarball).

| Gate (production site) | Narrowing plant | Named failing test |
|---|---|---|
| `sessprofile.compareLease` stale arm | N-derive-stale admits exactly head-minus-one | `TestDeriveRefusesLosingLeaseAndAmbiguity/stale_epoch` |
| `sessprofile.checkChain` divergent arm | N-derive-divergent admits exactly the fixture lease | `.../divergent_lease` |
| `sessprofile.checkChain` sequence arms | N-derive-repeat admits exactly the tail repeat | `.../repeated_sequence` |
| `sessprofile.changeTarget` from/to arm | N-change-fromto admits exactly standard→standard | `TestDeriveRefusesMalformedChangePayload/from_equals_to` |
| `sessprofile.changeTarget` confirmed arm | N-change-confirmed admits exactly `"yes"` | `.../confirmed_not_boolean` |
| `sessprofile.MintChangeEvent` confirmation arm | N-mint-unconfirmed admits exactly standard→yolo | `TestMintChangeEventRefusals/unconfirmed_yolo` |
| `sessprofile.MintChangeEvent` from/to arm | N-mint-fromto admits exactly yolo→yolo | `.../from_equals_to` |
| `sessprofile.MintChangeEvent` enum arm | N-mint-to-enum admits exactly `turbo` | `.../to_outside_vocabulary` |
| `sessprofile.currentLeaseLink` stale arm | N-setprofile-stale admits exactly head-minus-one | `TestSetProfileRefusals/stale_epoch` |
| `sessprofile.currentLeaseLink` divergent arm | N-setprofile-divergent admits exactly the below-head lease | `.../divergent_lease_below_head` |
| `sessprofile.sameEnvelope` instant arm | N-setprofile-replay-instant admits exactly the mismatched instant as replay | `TestSetProfileRefusesNoOpChange` |
| `sessprofile.checkObservedPair` value arm | N-check-pair-profile admits exactly the stale bundle value | `TestBundlePairProjection/N1_stale_creation_value` |
| `sessprofile.checkObservedPair` presence arm | N-check-pair-source admits exactly the missing source | `TestBundlePairProjection` (message-pinned) |
| `sessprofile.CheckForkPair` provenance arm | N-fork-provenance admits exactly the empty string | `TestForkProjectionAndCheck/empty_provenance_string` |
| `sessprofile.FinalizePair` dormant arm | N-finalize-dormant carries the pair under dormant | `TestFinalizePair` |
| `provhost.ResolveMapping` row arm | N-mapping-norow admits exactly qwen | `TestResolveMappingRefusals` |
| `provhost.ResolveMapping` Pi pin | N-mapping-pi preserves `0.73.1`, admits exactly `0.74.0` | `TestResolveMappingRefusals` |
| `provhost.ProjectLaunchArgv` token gate | N-argv-standard-alias admits exactly `--yolo` | `TestProjectLaunchArgvRefusals` (both alias subtests) |
| `provhost.checkProjectedArgvBounds` count arm | N-argv-bound-128 admits exactly 129 elements | `TestProjectLaunchArgvBoundsBothDirections` |
| `provhost.ProjectLaunchArgv` resolution match | N-argv-mismatch admits exactly the codex mismatch | `TestProjectLaunchArgvRefusals` (both mismatch subtests) |

## Stated bounds (declared, not silent)

- B1: no named profile registry exists in the pinned specification;
  the vocabulary is the `standard|yolo` enum (R1 pins exactly it).
- B2: `ax start --profile` has no spec'd absent default; the CLI leaf
  owns defaulting and this layer refuses empty (R2).
- B3: env is profile-independent; the projection assembles argv only
  (R5).
- B4: on an eventless chain the acting lease opens sequence 1; the
  Lease Record binding of that opening lease belongs to the
  ownership-leases story (R3).
- B5: the `provider.launched profile_mapping` member requires 1..512
  chars while standard omits every flag, so no standard spelling
  exists; fixtures carry the profile word and the pair checks never
  read the member (R7/R14).
- B6: sessquery's closure capability is unexported, so sessprofile
  defines its closure input explicitly with identical semantics
  (R8); the admission twin (`checkCheckpointProfileAuthority`) is
  the independent oracle for a future agreement test.
- B7: a machine-local alias that expands to unrestricted mode
  without spelling a table token is beyond argv inspection; the
  gate refuses exact table tokens and execve argv (secprim) is the
  structural defense (R13/R16 scope note in code).
- B8: bundle export objects, takeover/resume/fork transactions, and
  the `ax`/`doctor` surfaces stay with their owning leaves; this
  task ships the reducer-level projections those transactions must
  carry (R4/R14).
