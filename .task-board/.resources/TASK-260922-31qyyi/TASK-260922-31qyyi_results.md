# TASK-260922-31qyyi — handoff evidence

## Candidate

- Base commit: `c9233ce2b7d98b70707cb3aec28f919d53a4c020`.
- Final importer comparison candidate tree: `d2a528942e16a11d2ad92a755da410e7adb05cb18eadd6c025075fdfbd235f42`.
- Candidate remains uncommitted in the managed Story worktree.
- No `task-board.config.json` change.

## Review rev1 findings answered

| Finding | Revision 1 evidence | Revision 2 result |
| --- | --- | --- |
| P1-1: derivation tests skipped, so M1/M2 survived committed CI | Reviewer measured 0 of 4 entries with a live M1 killer. | Four assigned AC entry categories now have non-skipped named tests: `LoadProfile`+`Derivation.Derive`, `Projector.Project`, `Transactor.SetProfile` from-end, and `axpane.deriveProfile`. The public `Projector.ProjectForHeads` sibling has its own named test too: **4/4 AC categories and 5/5 production paths**. Fixtures append the profile change under the real winning lease, then have the successor name an earlier checkpoint, so the event is outside its attested handoff. M1 is killed independently at each path with the append gate disabled as an instrument and with it enabled. The gate-on full suite fails on exactly the five lower-epoch witnesses. M2 has its own same-epoch tuple test and is killed by its named gate-on test and full gate-on suite. |
| P2-1: `N-profile-closure` had a stale source anchor | Old text matched a removed `sessprofile.DeriveForHeads(...)` call and produced `patch anchors 0, want 1`. | Re-anchored to `input.ProfileData.DeriveForHeads(heads)`. `N-profile-closure` is KILLED by `TestDecideResumeProfileUsesClosure`; the mutant preserves the searched-for call text inside its altered branch and runs the behavioral test. |

### Production witnesses

| Production call site | Named committed test | Gate-on narrowing | Gate-disabled instrument |
| --- | --- | --- | --- |
| `LoadProfile` followed by `Derivation.Derive` | `TestLoadProfileDoesNotExposeLosingLeaseAsEffectiveSource` | M1 KILLED | M1 KILLED |
| `Projector.Project` | `TestProbe15DisabledAppendGateProjector` | M1 KILLED | M1 KILLED |
| `Projector.ProjectForHeads` | `TestProjectorForHeadsDoesNotExposeLosingLeaseAsEffectiveSource` | M1 KILLED | M1 KILLED |
| `Transactor.SetProfile` from-end | `TestSetProfileFromEndDoesNotReplayLosingLeaseChange` | M1 KILLED | M1 KILLED |
| `axpane.deriveProfile` | `TestAxpaneDeriveProfileDoesNotUseLosingLeaseSource` | M1 KILLED | M1 KILLED |

M1’s `authorizedSource` narrowing admits the exact losing event ID. Its five gate-on targeted rows and five gate-disabled targeted rows each fail only at their named test. The gate-on full `go test ./... -count=1` row fails on exactly those five tests. M2 narrows exact lease tuple matching to epoch-only matching; its gate-disabled targeted witness and gate-on targeted witness both fail at `TestAxpaneDeriveProfileRejectsUnmintedSameEpochLeaseTuple`, and the gate-on full suite fails only at that witness. The instrumented six-test baseline passes with exit 0. The applied harmless-comment control survives with exit 0. See `mutants-authority-expanded-12/mutants.json` and its per-row raw logs.

## Authority decisions and edge cases

- The current winning lease’s exact tuple may author a profile source.
- A prior lease may author a source only when the event is inside both the requested explicit closure and the winning lease’s independently re-attested handoff closure.
- A never-minted higher-epoch event is not an authority source: `TestProjectorRejectsNeverMintedHigherEpochProfileSource`, `TestDerivationForHeadsRejectsUnmintedHigherEpochSource`, `TestProjectorForHeadsRejectsUnmintedHigherEpochProfileSource`, and `TestSetProfileRefusesNeverMintedHigherEpochOnEmptyChain` cover projection, `DeriveForHeads`, `ProjectForHeads`, and from-end writing.
- With no lease rows, the Session Record remains authoritative: `TestProjectorEmptyLeaseStoreKeepsRecordAuthority`, `TestLoadProfileEmptyLeaseStoreKeepsSessionRecordAuthority`, `TestSetProfileRefusesEmptyLeaseStore`, and `TestSetProfileRefusesNeverMintedHigherEpochOnEmptyChain` pin the empty-store behavior.
- A valid prior source in the winning handoff stays effective (`TestProjectorKeepsPriorLeaseSourceFromWinningHandoffClosure`). A missing/unreadable winning handoff closure is not treated as empty (`TestProjectorMissingWinningHandoffStoreIsNotTreatedAsEmptyClosure`).
- Append admission code was not changed.

## SPEC 2.4 registry and measured coverage

The decoded `ownership.v0.7.0.json` Section 2.4 binding and clause `2.4#2` point to `internal/sessprofile/authority.go:Derive`. Both the clause acceptance-case list and decoded acceptance-case record include `story-260922-derivation-side-profile-source`; `TestSection24LosingLeaseClausePointsAtDerivationAuthority` checks the decoded objects. `TestV070RegistryRederivesFromTrunkV060Registry` covers registry re-derivation. The canonical digest was re-derived after the registry edit:

```text
reviewedOwnershipCanonicalSHA256=c173e0216d13b3225c31e48c474a23969d578ad163ab079bf571217d939bf905
```

Final tracecheck output: `contracts=64 normative_sections=36 acceptance_cases=155 fixtures=33 compatibility_contracts=55 assigned_scopes=0`; coverage is `bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574`. `README.md` measured-coverage text matches the output; `TestREADMEMeasuredCoverageMatchesTracecheckReport` passes. Global and Section 2.4 scoped tracecheck both exit 0.

## Complete importer outcome comparison

The `go list -test` importer census for `internal/sessprofile` contains only `internal/axpane` (production and test rows). In isolated base and candidate copies, `sessrepo.checkWinningLease` was disabled in both before driving the complete `TestProfileSourceOutcomeGrid`:

- Runtime exits: base 0, candidate 0.
- Outcome keys: 21 before / 21 after; 18 moved.
- Every moved input class: `empty_lease_store`, `post_takeover_lower_epoch`, `unminted_higher_epoch`, `unminted_same_epoch_loser`.
- All 18 moved entry/input pairs are listed in `outcome-grid-final-03/outcome-comparison.json`.
- Winning-handoff and current-winner writer outcomes remain equal.
- Base tree SHA-256: `cc778d7611e759fe3c213c541d7d4ad0c05744fb175f17d60ead6cefabda9d7e`.
- Candidate tree SHA-256: `d2a528942e16a11d2ad92a755da410e7adb05cb18eadd6c025075fdfbd235f42`.

The base probe-15 output `{Profile:yolo, HasSource:true}` with the append gate disabled is accepted from the earlier attached raw reproduction artifact under `.temp/TASK-260922-31qyyi/reproduction/`; it was not rerun during final validation.

## Validation evidence

The full configured validation set was run against this candidate. The all-package race command was split into eight sequential, bounded package groups; the groups cover all 44 packages exactly once and each retains `-race -count=1 -timeout 25m`.

| Check | Result |
| --- | --- |
| `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1 -v` | exit 0 |
| `go test ./... -cover -count=1` | exit 0 |
| 8 race groups, all 44 packages | each exit 0; `sessquery` 340.461s, tracecheck 68.782s |
| 17 configured `-fuzztime=100x -parallel=1` commands | all exit 0 |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0; report above |
| Section-scoped tracecheck for `2.4`, `6.2`, `7.8`, and `10.2` | each exit 0 on the final tree |
| `go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | exit 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | exit 0 |
| `GOOS=windows GOARCH=amd64 go build ./...` | exit 0 |
| `GOOS=windows GOARCH=amd64 go vet ./...` | exit 0 |
| Tracked JSON parse command from `task-board.config.json` | exit 0 |
| `git diff --check` | exit 0 |
| `task-board validate --board-dir "$TASK_BOARD_DIR"` | exit 0; 254 existing repository-wide diagnostics; none reference this task |
| `go test ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1 -v` | exit 0 on the final tree |
| `go run ./internal/traceability/cmd/tracecheck --section=2.4` | exit 0 on the final tree |
| `AX_MUTANT_VERBOSE=1 PYTHONDONTWRITEBYTECODE=1 python3 internal/axpane/mutant_harness.py N-profile-closure` | harness exit 0; mutant killed by `TestDecideResumeProfileUsesClosure` |

Raw command logs are in `.temp/TASK-260922-31qyyi/`, along with the full mutation and importer outputs.

## Checklist and handoff state

- Task-specific acceptance rows demonstrated: **5 of 5**.
- Assigned derivation entry categories with live non-skipped M1 killer: **4 of 4**.
- Concrete derivation call sites with their own live named lower-epoch witness and M1 killer: **5 of 5**.
- Updated task resources `TASK-260922-31qyyi_results.md`, `TASK-260922-31qyyi_conformance-matrix.md`, `TASK-260922-31qyyi_producer-evidence.tar.gz`, and `TASK-260922-31qyyi_board-validation.log` were downloaded from the authoritative board and compared byte-for-byte with the local artifacts.
- Review rev1 was rejected as `CHANGES REQUESTED`; its verdict and review evidence are attached and the task was routed back to `development` before this rework began. This revision is now handed off as `to-review`; no rev2 verdict is presumed.
- Requested handoff is `to-review` / ready for review, not accepted completion.
