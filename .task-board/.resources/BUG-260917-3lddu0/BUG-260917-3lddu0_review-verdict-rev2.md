# BUG-260917-3lddu0 — independent review of CR revision 2

Candidate TREE `afb776578e26979c7c2da474116dfc1c95f8eafa`; base/trunk `799c338e401fca0b24859c0b870cd665927209f2`; Story checkpoint/HEAD `964fa472c97569fb209e5bdf831d42e2484ef078`. Reviewer run `RUN-260922-5edb17`. Review is read-only: all instrumentation and mutants ran under task-scoped `.temp/` copies exported from immutable Git objects. The 28 reviewed live paths still equal the candidate; no live index, branch, commit or product file was changed.

## Verdict: CHANGES REQUESTED

The production fix is correct for the assigned stale/same-epoch append scope. **2 of 2 task AC rows are driven**, under the expressly permitted route (b): the profile-source protection is provided by append admission, not by an independent lease-aware derivation owner. The remaining blocking issues are the evidence/registry delivery, not a request to redesign or widen production scope. No new P1 production defect was found.

### P2-1 — the newly documented §2.4 clause binding does not exist

`README.md` says `story-260917-losing-lease-profile-source` adds its proof to clause `2.4#2`. The candidate registry defines that acceptance case with the correct `AppendEvent` production owner, but it occurs in **zero ownership-group or clause acceptance-case lists**. Clause `2.4#2` still lists only `sessprofile-derive`. Thus the new source proof is declaration-checked globally but not connected to the clause the new README text claims it discharges.

Evidence: `logs/registry-binding-census.json` independently decodes the candidate registry and records one definition and zero references; compare `ownership.v0.7.0.json` around the new case at line 2880 and clause at line 4464, and README around line 3596. Renaming the test makes tracecheck fail, proving the global declaration check; it does not create the absent clause edge. The independently re-derived digest is correct, so this is a semantic missing binding, not stale hash bytes.

Required: connect the actual append-gate case to the intended section/clause, keep route (b)'s scope explicit, re-derive the digest and its exact tests, and align README/tracecheck evidence. Do not claim independent derivation-side authority or close unknown-higher-epoch ambiguity by prose.

### P2-2 — route (b)'s mandatory named owner element is still missing

The rework brief explicitly permits route (b) only with independent-owner work recorded as a bound **with a named owner element**. Results, matrix, README and TRACEABILITY name `internal/sessprofile/profile.go:Derive`, which is a function, not a tracked owner element. No owner task/bug ID is supplied for that deferred work. The sibling no-grant decision and the phantom/empty-store behaviors are accurately disclosed; the missing element here is specifically the required future lease-aware profile-source owner.

Evidence: `results.md` route-(b) paragraph and bounds section; `conformance-matrix.md` AC row 2; the rework brief item P1-1(b). My independent narrowing admits only stale `profile.changed`; the shipped `TestLosingLeaseProfileEventIgnored` fails at the append assertion, while the separate downstream probe derives `yolo` with the bypass mapping. This confirms why a function name alone does not resolve the deferred requirement.

`repeat-of: BUG-260917-3lddu0_review-verdict-rev1.md / P1-1`, limited to the incomplete bound/ownership disposition. The production-owner correction from `Derive` to `AppendEvent` is accepted. Required: identify an existing tracked owner element or create the minimum focused follow-up and link the exact bound to it. This review does not require implementing that out-of-scope follow-up here.

### P3 — distinguish test-status grids from runtime outcome grids

Producer `importer-outcomes-rev2/summary.txt` has 2,018/2,030 keys because it counts `(package, test name) -> pass/fail/skip`, including nine package rows: 2,009/2,021 actual test rows. The added entries are test names. It is not an outcome-keyed admission grid, despite LOGBOOK and results using that label. The newly added four-test semantic table is useful and resolves the old unargued-fixture-change issue; it does not change what the grid measures.

I independently instrumented actual `AppendEvent` return outcomes over the mechanically derived complete nine-package importer set and attach those grids, alongside the exact instrumentation patches. Use that evidence or an equivalent runtime grid and relabel the old counts as test-status counts. Do not claim zero admission-arm changes from unchanged PASS statuses. No additional production change is requested by this note.

## Independently measured acceptance and census

| AC | Production call site | Candidate test and independent observation | Result |
| --- | --- | --- | --- |
| Superseded append refused while tail still matches | `Repository.AppendEvent -> checkWinningLease`; composed by `Transactor.SetProfile`, `Emit`, `EmitParked` | `TestAppendEventRefusesSupersededLeaseWhileTailStillMatches`, `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches`, `TestEmitReachesAppendAdmissionGateAfterStaleObservation`, `TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation`; probe 9 drives all four | 1/1 driven; all four admitted on trunk and return stale refusal on candidate |
| Losing-lease profile change cannot become effective through admission | `Repository.AppendEvent`, then `Projector.Project -> Derive` / `LoadProfile -> Derive`; `Run` observes the remaining authority | `TestLosingLeaseProfileEventIgnored`; adapted probe 15 and independent profile-only narrowing | 1/1 driven under route (b); no independent derivation guarantee claimed |

**2 of 2 AC rows driven. 8 of 10 gate-entry cells measured; 2 of 10 bounded.** Lower-epoch side: 4/4 reachable entries. Same-epoch side: 4/4 reachable entries. `Run -> EmitParked` is bounded on both arms for the observe/append interleaving. Production caller grep has only `sessprofile/setprofile.go` and `axpane/events.go` directly calling AppendEvent; `EmitParked` delegates to Emit and `Run` delegates to EmitParked. No omitted public CLI entry was found.

Both same-epoch ID directions are killed separately at all four entries. The shipped larger-ID narrowing and my smaller-ID narrowing have per-entry failing tests. My `R13-stale-parked-admitted` admits only `session.parked` lower-epoch input and is killed specifically by `TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation`; direct profile-change tests remain irrelevant to that kill.

## Before/after and composition

Original probes 9 and 15 were read from `TASK-260830-1geqhj_review-evidence-rev1.tar.gz`. Current-API adaptations are included, with the necessary fencing observation/presented-token fields. On trunk, direct append, SetProfile, Emit, and EmitParked accept the superseded lease while the tail is still on it. On candidate all four refuse. Probe 15 on trunk derives `{Profile:yolo, HasSource:true}` from the losing event and `provhost.ProfileMapping("codex", derived.Profile)` returns `--dangerously-bypass-approvals-and-sandbox`; candidate derives standard/no source and an empty standard mapping.

Important bound: the historical archive showed an actual Run launch with that mapping. On current trunk 799c338, my adapted `Run` probe parks, while session-head profile derivation is still yolo. I do not report a current-trunk launch I did not observe. The security-relevant derived-profile defect and provider mapping reproduce independently through production functions.

The successor-writer probe drives successful Emit, confirmed SetProfile, profile derivation, identical SetProfile retry, and EmitParked under the winner on both trees. The refusal-idempotency probe verifies the two stale refusals leave one authoritative event and exactly two blobs, including byte-identical preserved stale bytes; same-epoch retry also preserves its blob. The newly added direct and axpane tests catch removal of same-epoch preservation.

The three checkpoint fixture reorderings correctly publish old-owner `session.idle`/`session.stopped` history before the successor CAS; the old order is precisely the newly forbidden input. The fourth landed test now asserts append refusal for losing `profile.changed`. The results, LOGBOOK and TRACEABILITY now record the four changes and §5.2/§5.3/§2.4 arguments. No unrelated production writer was changed by this leaf.

## Mutation evidence

Shipped `mutate_append_admission.py` ran twice from the exact candidate source. Each run: 2/2 real narrowings KILLED (subprocess exit 1); comment control SURVIVED (0); missing-token control NOT_APPLIED (not a kill); syntax control COMPILE_OR_HARNESS_FAILURE (1, not a kill); full three-package controls before/after exited 0. Both batteries exited 0.

Reviewer plants are isolated and restored byte-for-byte. Final corrected instruments: **8/8 behavioral plants KILLED twice**, plus one applied harmless control SURVIVED twice:

| Plant | Weakened behavior | Killing test / entry |
| --- | --- | --- |
| R4a | Admit same-epoch loser ID smaller than winner | All four direct/SetProfile/Emit/EmitParked same-epoch entry tests |
| R5 | Skip lower-epoch losing blob preservation | `TestAppendEventRefusesSupersededLeaseWhileTailStillMatches` |
| R6 | Skip same-epoch losing blob preservation | Direct same-epoch test plus Emit and EmitParked |
| R9 | Admit exactly one skipped sequence | `TestAppendEventRefusesSequenceGap` through AppendEvent |
| R10 | Admit equal-length corrupt pre-existing blob | `TestAppendEventRefusesSameLengthDisagreeingBytes` through AppendEvent |
| R11 | Preserve same-epoch losers only for profile.changed | Direct lifecycle refusal and EmitParked preservation tests |
| R12 | Admit unconfirmed yolo only at sequence 2 | `TestMintChangeEventRefusals/unconfirmed_yolo`, `TestSetProfileRefusals/unconfirmed_yolo` |
| R13 | Admit stale session.parked only | `TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation` |

The main reviewer mask is `Test(AppendEventRefuses|AppendEventChainInOrder|SetProfileRefuses|EmitReaches|EmitParkedReaches|LosingLeaseProfileEventIgnored|MintProfileChange)` over `./internal/sessrepo ./internal/sessprofile ./internal/axpane`, `-count=1 -v`. It accidentally omits the real confirmation test names, so R12 SURVIVED twice under that mask; those logs remain. The corrected R12 mask is `Test(MintChangeEventRefusals|SetProfileRefusals)$` over the same three packages and kills twice. No survivor was silently converted into a kill. The initial full-package plant attempt was interrupted (exit -2) before completion to keep runs bounded; no kill claim uses it.

Additional attacks: stale-profile-only admission narrowing makes both the production-entry test and separate downstream probe fail (expected); wrong README coverage figure fails the landed README pin; disabling the new profile-source test fails tracecheck. The admission mutant retains the source token and the behavioral suite, rather than only a source checker, detects the breach.

## Registry, scope and carried notes

Independently decoded and re-marshaled registry digest:
`4975c39feb26d56c5ef785e993ccf6407770b6dd118926da40b5a19bd54a9a78` — matches the candidate pin. All trunk acceptance cases and ownership keys remain; two acceptance cases added, zero carried acceptance cases changed. Fresh remote-main read still equals 799c338, so the base-to-current-trunk changed-path set is empty and there is no refresh overwrite to hide. 827 archived source/test/config/document paths were compared against candidate Git blobs with zero mismatches.

Verbatim exact-tree tracecheck:

```text
traceability ok: contracts=64 normative_sections=36 acceptance_cases=154 fixtures=33 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574
```

README's measured-coverage line equals that report. `-section 2.4` exits 0; `-section 5.3` exits 1 with the explicitly expected 7/8 partial-binding refusal. That failure is not called green. The misleading new README clause-edge sentence is P2-1, separate from the numerically correct coverage subsection.

All four sibling P3 notes are dispositioned: the untouched axpane doc-comment qualifier is recorded; observeRemoteWinner redundancy is recorded, not simplified; the no-grant §4.2 step-4 product question stays unresolved, not decided; candidate TREE OID and row masks are recorded. Unknown higher epoch still becomes a profile source and empty lease store still skips the gate on both trees; these are disclosed bounds, not newly fixed behavior.

## Validation, evidence reuse and instrument failures

Validation summary and runtime grids are appended below from completed logs. Exact configured command list has 30 commands. Relevant product platform is the cross-platform Go library/application implementation, validated locally on darwin/arm64 plus the configured Linux/Windows build checks and additional Windows vet. No iOS target exists.

The sibling BUG-260917-2fwf8e accepted rev3 verdict is reused only for its prior 37-row fencing and 123-row terminstance mutation batteries and their historical outcome grids; those source paths are byte-identical to the signed checkpoint. This review independently runs the entire candidate tests, current sessrepo harness, extra plants, and new runtime outcome census. The attached producer raw logs are corroborating evidence, not substitutes for failures in my runs.

Instrument corrections are retained: first probe-copy attempts misplaced a sessrepo test into axpane and failed compilation; corrected copies then ran. The old idempotency probe used `file.json/..` and ignored ReadDir's error, yielding a meaningless zero count; the corrected probe uses filepath.Dir, asserts the read and count, and checks exact stale blob bytes. These are reviewer-instrument errors, not product failures. Python ran without local bytecode artifacts. The reused phantom probe has an old diagnostic suffix saying no owner was named; this review instead uses the current, disclosed code-owner bound and reports only the separate missing tracked owner element in P2-2. No unknown or unexecuted status is treated as a pass.

## Completed validation on the exact candidate

**30 of 30 configured checks green; additional Windows vet green.** All 30 were rerun by this reviewer. The archive-based gofmt and JSON commands enumerate the immutable archive instead of a live index; diff-check names the exact base/candidate Git objects. Board validation explicitly inherits the authoritative TASK_BOARD_DIR; its 179 standing MISSING_ACTIVITY diagnostics remain visible and its actual exit is 0.

| Check | Exit | Elapsed | Load before / after (1m) |
| --- | ---: | ---: | ---: |
| `Configured 1 (archive-equivalent enumeration)` | 0 | 0.5s | 13.6 / 13.6 |
| `go build ./...` | 0 | 3.5s | 13.6 / 13.0 |
| `go vet ./...` | 0 | 12.5s | 13.0 / 12.7 |
| `go test ./... -count=1 -v` | 0 | 570.2s | 9.2 / 24.2 |
| `go test ./... -race -count=1 -timeout 25m` | 0 | 604.8s | 11.7 / 13.9 |
| `go test ./... -cover -count=1` | 0 | 271.2s | 25.6 / 18.3 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | 1.8s | 12.5 / 13.1 |
| `go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | 0 | 1.7s | 13.1 / 13.1 |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | 8.6s | 13.1 / 15.3 |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | 2.2s | 15.3 / 15.3 |
| `Configured 28 (archive-equivalent enumeration)` | 0 | 0.1s | 15.3 / 15.3 |
| `task-board validate` | 0 | 7.7s | 11.7 / 12.3 |
| `git diff --check 799c338e401fca0b24859c0b870cd665927209f2 afb776578e26979c7c2da474116dfc1c95f8eafa` | 0 | 0.1s | 12.3 / 12.3 |

Commands 7–23: all 17 configured fuzz lanes exit 0, each with `-fuzztime=100x -parallel=1`; individual commands, times, loads and logs are attached. The race timeout remained exactly `25m`. No command was backgrounded and abandoned.

Package coverage from the reviewer run:
- `ok   github.com/relux-works/agent-session-manager/internal/axpane 27.184s coverage: 81.3% of statements`
- `ok   github.com/relux-works/agent-session-manager/internal/fencing 10.709s coverage: 99.3% of statements`
- `ok   github.com/relux-works/agent-session-manager/internal/sessprofile 29.502s coverage: 92.6% of statements`
- `ok   github.com/relux-works/agent-session-manager/internal/sessrepo 113.095s coverage: 86.6% of statements`
- `ok   github.com/relux-works/agent-session-manager/internal/terminstance 21.538s coverage: 86.6% of statements`

## Reviewer runtime outcome comparison

Mask: `go test -json -count=1 ./internal/axpane ./internal/crashgate ./internal/fencing ./internal/sessckpt ./internal/sessprofile ./internal/sessquery ./internal/sessstate ./internal/termbind ./internal/terminstance`. Both trees exit 0, 9/9 packages PASS. Baseline is actual trunk 799c338, not just the parent checkpoint. The Imports/TestImports/XTestImports sets were derived mechanically with `go list` (direct imports, internal tests and external tests).

The instrumentation records `(executing package, event type, epoch, sequence, relation to durable winner, actual AppendEvent outcome)`, not a test name. Winner lookup occurs in a deferred read after the repository lock is released; `no-readable-winner` explicitly does not distinguish missing state from a read error. Exact patches are attached. This diagnostic instrumentation is not the source of the pristine candidate suite result.

| Importer | Before outcome keys / calls | After outcome keys / calls | Count-delta keys |
| --- | ---: | ---: | ---: |
| `axpane` | 23 / 221 | 26 / 244 | 13 |
| `crashgate` | 1 / 15 | 1 / 15 | 0 |
| `fencing` | 4 / 8 | 4 / 8 | 0 |
| `sessckpt` | 3 / 28 | 3 / 28 | 0 |
| `sessprofile` | 12 / 88 | 16 / 94 | 4 |
| `sessquery` | 54 / 2360 | 54 / 2360 | 0 |
| `sessstate` | 23 / 160 | 23 / 160 | 0 |
| `termbind` | 10 / 46 | 10 / 46 | 0 |
| `terminstance` | 0 / 0 | 0 / 0 | 0 |

Total: 130 keys / 2926 calls before; 137 keys / 2955 calls after. Set delta: 10 new keys, 3 removed keys. The suite includes changed and new fixtures, so counts are evidence about executed input classes, not proof that identical tests kept identical meanings.

| Runtime tuple (package, event, epoch, seq, winner relation, outcome) | Before count | After count |
| --- | ---: | ---: |
| `axpane, profile.changed, 1, 2, lower-epoch, ADMIT` | 1 | 0 |
| `axpane, profile.changed, 1, 2, lower-epoch, STALE` | 0 | 2 |
| `axpane, profile.changed, 2, 1, higher-epoch, ADMIT` | 0 | 2 |
| `axpane, profile.changed, 2, 2, same-epoch-loser, DIVERGENT` | 0 | 2 |
| `axpane, session.created, 1, 1, same-winner, ADMIT` | 144 | 154 |
| `axpane, session.failed, 1, 2, same-winner, ADMIT` | 9 | 12 |
| `axpane, session.failed, 2, 1, higher-epoch, ADMIT` | 0 | 2 |
| `axpane, session.idle, 1, 2, lower-epoch, ADMIT` | 3 | 0 |
| `axpane, session.idle, 1, 2, same-winner, ADMIT` | 11 | 14 |
| `axpane, session.parked, 1, 3, lower-epoch, STALE` | 0 | 1 |
| `axpane, session.parked, 2, 2, same-epoch-loser, DIVERGENT` | 0 | 2 |
| `axpane, session.stopped, 1, 3, lower-epoch, ADMIT` | 3 | 0 |
| `axpane, session.stopped, 1, 3, same-winner, ADMIT` | 10 | 13 |
| `sessprofile, profile.changed, 1, 2, lower-epoch, STALE` | 0 | 1 |
| `sessprofile, profile.changed, 2, 2, same-epoch-loser, DIVERGENT` | 0 | 2 |
| `sessprofile, session.created, 1, 1, same-winner, ADMIT` | 0 | 1 |
| `sessprofile, session.created, 2, 1, no-readable-winner, ADMIT` | 0 | 2 |

Interpretation:

- The landed losing `profile.changed` input moves ADMIT -> STALE; the additional stale Emit and SetProfile rows also return STALE. Direct AppendEvent and EmitParked moves are independently driven by probe 9.
- The three landed old-owner stop fixtures no longer execute `session.idle` seq 2 and `session.stopped` seq 3 under a lower epoch after takeover. They now execute those same events before takeover (`same-winner`), matching their recorded §5.2/§5.3 argument.
- Newly exercised same-epoch losing `profile.changed` through Emit/SetProfile and `session.parked` through EmitParked return DIVERGENT for both losing ID directions. The new parked lower-epoch row returns STALE.
- New same-epoch test setup adds higher-epoch profile/failed events and no-store epoch-2 created events; these remain admitted as the disclosed higher-epoch/no-store bounds, not newly fixed ambiguity.
- Added ordinary created/failed fixture counts remain ADMIT. The seven other importer packages have zero append-outcome count deltas. `terminstance` imports sessrepo but executes zero AppendEvent calls in this suite; its package tests do run and pass.


## Rework scope (for the producer)

1. Repair the missing profile-source section/clause reference and the matching README claim. Re-derive the registry pin and run its narrow registry/tracecheck/README checks on the resulting candidate.
2. Finish route (b)'s explicit tracked owner-element bound; link that element from results/matrix/TRACEABILITY. Do not implement independent derivation or widen the append scope in this leaf.
3. Relabel the package/test PASS grid honestly, and reference the attached runtime outcome comparison or reproduce it. Preserve the four-test semantic moved-input table.

No further research prerequisite, replacement append algorithm, broad refactor, or live Story commit is requested. Leave the candidate uncommitted and publish a new reviewable revision with updated evidence.
