# TASK-260922-31qyyi — conformance matrix

## Finding-by-finding review response

| Requirement / finding | Production behavior or call site | Named committed test(s) | Negative / narrowing evidence | Status |
| --- | --- | --- | --- | --- |
| Rev1 P1-1: lower-epoch profile source must not become effective at `LoadProfile` + derivation | `LoadProfile` / returned `Derivation.Derive` | `TestLoadProfileDoesNotExposeLosingLeaseAsEffectiveSource` | M1 exact-event narrowing fails this test with the append gate on and disabled as an instrument | Addressed |
| Rev1 P1-1: source selection through `Projector.Project` | `Projector.Project` | `TestProbe15DisabledAppendGateProjector` | M1 fails this test with the append gate on and disabled as an instrument | Addressed |
| Rev1 P1-1: explicit heads cannot bypass winner handoff authority | `Projector.ProjectForHeads` | `TestProjectorForHeadsDoesNotExposeLosingLeaseAsEffectiveSource` | M1 fails this test with the append gate on and disabled as an instrument | Addressed |
| Rev1 P1-1: from-end profile writing must ignore the losing source | `Transactor.SetProfile` from-end | `TestSetProfileFromEndDoesNotReplayLosingLeaseChange` | M1 fails this test with the append gate on and disabled as an instrument | Addressed |
| Rev1 P1-1: axpane derivation must reject the losing source | `axpane.deriveProfile` | `TestAxpaneDeriveProfileDoesNotUseLosingLeaseSource` | M1 fails this test with the append gate on and disabled as an instrument | Addressed |
| Rev1 P1-1: named same-epoch ambiguous tuple must not be accepted | `axpane.deriveProfile` | `TestAxpaneDeriveProfileRejectsUnmintedSameEpochLeaseTuple` | Independent M2 epoch-only tuple narrowing fails this test with the append gate disabled and enabled; the enabled full suite fails only here | Addressed |
| Rev1 P2-1: `N-profile-closure` must retain a working behavioral plant | `axpane.Decide` profile derivation during restore | `TestDecideResumeProfileUsesClosure` | `N-profile-closure` alters the branch while preserving the call token; behavioral test fails, harness classifies KILLED | Addressed |

## Census and ratios

| Measured population | Passing witnesses | Ratio |
| --- | --- | ---: |
| Task-specific acceptance rows | non-skipped tests, registry decode, importer comparison, tracecheck and mutation evidence | **5 of 5** |
| Assigned entry categories from the reviewer brief | `LoadProfile`+`Derive`, `Projector.Project`, `SetProfile` from-end, `axpane.deriveProfile` | **4 of 4** |
| Concrete derivation entry points including `Projector.ProjectForHeads` | five rows above with their own named lower-epoch test | **5 of 5** |
| M1 individual gate-disabled narrowing rows | all five concrete paths | **5 of 5 killed** |
| M1 individual gate-enabled narrowing rows | all five concrete paths | **5 of 5 killed** |
| M1 full gate-enabled suite | five named losing-event tests fail, no unrelated failure | **1 of 1 killed** |
| M2 same-epoch tuple witnesses | instrumented named test, enabled named test, enabled full suite | **3 of 3 killed** |

The mutation source logs preserve subprocess commands and exit codes. The instrumented derivation baseline exits 0 and the applied harmless-comment control survives with exit 0.

## Authority and closure edge cases

| Case | Named witness | Expected outcome |
| --- | --- | --- |
| Never-minted higher epoch through projection | `TestProjectorRejectsNeverMintedHigherEpochProfileSource` | Session Record remains authoritative |
| Never-minted higher epoch through `Derivation.DeriveForHeads` | `TestDerivationForHeadsRejectsUnmintedHigherEpochSource` | Session Record remains authoritative |
| Never-minted higher epoch through `Projector.ProjectForHeads` | `TestProjectorForHeadsRejectsUnmintedHigherEpochProfileSource` | Session Record remains authoritative |
| Never-minted higher epoch from-end | `TestSetProfileRefusesNeverMintedHigherEpochOnEmptyChain` | refusal; no append |
| Empty lease store through projection | `TestProjectorEmptyLeaseStoreKeepsRecordAuthority` | Session Record remains authoritative |
| Empty lease store through load/derive | `TestLoadProfileEmptyLeaseStoreKeepsSessionRecordAuthority` | Session Record remains authoritative |
| Empty lease store through from-end | `TestSetProfileRefusesEmptyLeaseStore` | refusal; no append |
| Valid earlier profile source included in winning handoff closure | `TestProjectorKeepsPriorLeaseSourceFromWinningHandoffClosure` | valid source remains effective |
| Missing or unreadable winning handoff closure | `TestProjectorMissingWinningHandoffStoreIsNotTreatedAsEmptyClosure` | fail closed; do not promote prior source |

## Registry and importer census

| Check | Evidence |
| --- | --- |
| Section 2.4 owner edge | Decoded Section 2.4 binding and clause `2.4#2` point to `internal/sessprofile/authority.go:Derive` |
| Acceptance case | `story-260922-derivation-side-profile-source` appears in the decoded clause list and decoded case registry |
| Digest | `reviewedOwnershipCanonicalSHA256=c173e0216d13b3225c31e48c474a23969d578ad163ab079bf571217d939bf905` |
| Tracecheck | 64 contracts; 36 normative sections; 155 acceptance cases; 33 fixtures; 55 compatibility contracts; 70/574 clauses discharged; exit 0 |
| README coverage | `TestREADMEMeasuredCoverageMatchesTracecheckReport` passes |
| Complete importer set | `internal/axpane` only, from both production and test rows of `go list -test` |
| Outcome census | append gate disabled in both isolated copies; runtime exit 0 before and after; 21 keys before and after; 18 moved |
| Every moved class | `empty_lease_store`, `post_takeover_lower_epoch`, `unminted_higher_epoch`, `unminted_same_epoch_loser` |
| Preserved outcomes | winning-handoff source and normal current-winner writer outcomes stay equal |
| Candidate tree SHA-256 | `d2a528942e16a11d2ad92a755da410e7adb05cb18eadd6c025075fdfbd235f42` |

See `outcome-grid-final-03/outcome-comparison.json` for each key and each moved before/after value.

## Full configured validation

| Validation | Result |
| --- | --- |
| gofmt gate | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1 -v` | exit 0 |
| `go test ./... -cover -count=1` | exit 0 |
| Race test all-package set | all 44 packages passed in 8 bounded groups; each command includes `-race -count=1 -timeout 25m` |
| Fuzz test all configured targets | 17 commands, each `-fuzztime=100x -parallel=1`, all exit 0 |
| Tracecheck and scoped Section 2.4 tracecheck | exit 0 |
| Catalog generator `-check` | exit 0 |
| Linux amd64 build | exit 0 |
| Windows amd64 build | exit 0 |
| Windows amd64 vet | exit 0 |
| Tracked JSON parse | exit 0 |
| `git diff --check` | exit 0 |
| Final traceability package tests | exit 0, including registry owner and README coverage tests |
| Final `N-profile-closure` run | KILLED by named behavioral test; harness exit 0 |
| `task-board validate --board-dir "$TASK_BOARD_DIR"` | exit 0; 254 existing repository-wide diagnostics; no finding for this task |
| Updated task resources | Results, matrix, evidence archive, and board-validation log were read back from the authoritative board and matched local bytes exactly |

Logs and raw mutation rows are included in `TASK-260922-31qyyi_producer-evidence.tar.gz`.
