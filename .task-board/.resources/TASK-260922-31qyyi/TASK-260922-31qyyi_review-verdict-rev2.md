# TASK-260922-31qyyi review verdict — CR rev2: ACCEPTED

Reviewer: claude-opus-5-5 (low). The live worktree's tree, taken through a scratch GIT_INDEX_FILE, equals the candidate d9dad842 on base c9233ce. The review ran in a git-archive copy, and the live index was not touched.

## Rev1 findings
| Finding | Status | Evidence |
|---|---|---|
| P1-1: all 8 derivation witnesses skipped, so M1/M2 had no CI kill | FIXED | No `t.Skip` remains in sessprofile, axpane or sessckpt. Each witness builds the losing tail with the real append gate ON (`gateOnLosingProfileWorld`: A appends while it is the winner, then B takes over). |
| P2-1: N-profile-closure anchor lost | FIXED | Re-anchored to `input.ProfileData.DeriveForHeads(heads)`. The axpane harness reports KILLED. The run had no NOT_APPLIED or ERROR rows, 55 KILLED lines, and the C-doc-comment control SURVIVED. |

## My own instruments (plain `go test` over sessprofile, axpane and sessckpt, real gate on)
| Plant | Type | Result | Killers |
|---|---|---|---|
| M1 authorizedSource returns true for any known lease | narrowing | KILLED, 5 fails | the 5 entry witnesses. Each one also fails when run ALONE (exit 1). |
| M2 knownLease matches on epoch only | narrowing | KILLED | TestAxpaneDeriveProfileRejectsUnmintedSameEpochLeaseTuple |
| M5 authorizedSource drops the knownLease check | narrowing | KILLED | same as M2 |
| M6 missing-handoff-closure refusal disabled | arm-disable | KILLED | TestProjectorMissingWinningHandoffStoreIsNotTreatedAsEmptyClosure |
| M4 authorizedSource drops the explicitClosure check | narrowing | SURVIVED | Equivalent: the fold loop already `continue`s past events outside requestedClosure. Not a finding. |
| CTRL comment edit | control | SURVIVED | none. The instrument can report a survivor. |

Shipped `mutate_profile_authority.py`: instrumented CONTROL 0 fails; M1 KILLED (11); M2 KILLED (3, including the gate-on full suite); harmless control SURVIVED. My first attempt failed only because my archive excluded the tracked `.task-board`, which made specpin fail. That was my setup, not the candidate.

## Measured ratio
Derivation entries with a committed, non-skipped, gate-on test that fails under M1: **4 of 4**. The entries and their witnesses:
- LoadProfile+Derive: TestLoadProfileDoesNotExposeLosingLeaseAsEffectiveSource
- Projector.Project: TestProbe15DisabledAppendGateProjector
- Projector.ProjectForHeads: TestProjectorForHeadsDoesNotExposeLosingLeaseAsEffectiveSource
- SetProfile from-end: TestSetProfileFromEndDoesNotReplayLosingLeaseChange
- axpane.deriveProfile: TestAxpaneDeriveProfileDoesNotUseLosingLeaseSource

## Mandatory four
1. **Digest.** tracecheck exits 0 on the exact tree, so the pinned `reviewedOwnershipCanonicalSHA256` equals the projection it computes. Two registry plants made it exit 1: a duplicate test owner, and 2.4 coverage set to partial.
2. **Full `go test ./... -count=1`.** Exit 0, 44 packages ok.
3. **README plant.** Changing `clauses_discharged` from 70 to 71 made the tracecheck package tests exit 1. With the plant reverted, the README equals the tracecheck output (`clauses_discharged=70/574`).
4. **compare_importers.py.** Exit 0. The importer set, from `go list`, is `{internal/axpane}`. It produced 21/21 keys, 18 of which moved. The moved classes are empty_lease_store, post_takeover_lower_epoch, unminted_higher_epoch and unminted_same_epoch_loser, and all of them moved to `{standard, no source}`. SetProfile from-end now reads previous=standard. `winning_handoff_contains_prior_source` stays yolo, so composition holds.

## Registry, decoded
- Clause `2.4#2` acceptance_cases contains `story-260922-derivation-side-profile-source`.
- That case's production owner is `internal/sessprofile/authority.go:Derive`, and its tests are the non-skipped witnesses listed above.
- The section:2.4 binding is `full`.

## Never-minted higher epoch and empty store
Both fall back to the record, which fits "ambiguous" in SPEC §2.4. Named tests pin them: TestProjectorRejectsNeverMintedHigherEpochProfileSource, TestDerivationForHeadsRejectsUnmintedHigherEpochSource, TestProjectorForHeadsRejectsUnmintedHigherEpochProfileSource, TestProjectorEmptyLeaseStoreKeepsRecordAuthority and TestLoadProfileEmptyLeaseStoreKeepsSessionRecordAuthority. The instrumented importer grid confirms them at all 5 entries.

## Hygiene
`go vet ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` both exit 0. gofmt reports no files. `task-board.config.json` is byte-identical to the base.

## P3 (no action)
None blocking. M4 is an equivalent mutant and is documented above.
