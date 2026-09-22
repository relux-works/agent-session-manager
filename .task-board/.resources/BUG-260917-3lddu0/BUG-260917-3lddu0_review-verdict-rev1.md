# BUG-260917-3lddu0 — independent review verdict, Change Request revision 1

- Reviewer run: RUN-260921-f25edb (claude-opus-5 max), role `reviewer`/`reviewer`.
- Change Request: `CR-BUG-260917-3lddu0-1` (story_final for STORY-260917-kf9g1h). Base OID
  `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` at review time, verified with
  `git ls-remote`; trunk did NOT move, so no refresh/merge question arises). Checkpointed
  predecessor `964fa472c97569fb209e5bdf831d42e2484ef078` (BUG-260917-2fwf8e rev3). Candidate
  **tree OID `8042bcc231a9fe7ddca50969cc69c7866797996a`**, which I re-derived from the live
  working tree through a temporary index (`GIT_INDEX_FILE`) and again from my `git archive`
  extraction; patch resource sha256 `bce02417…` as recorded on the CR.
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §2.4 line 666 ("Losing-lease or
  ambiguous events MUST NOT change it."), §5.2 lines 1832–1841 ("The winning owner MUST serialize
  state-changing events … Losing-lease branches remain preserved but are never applied"), §5.3
  lines 2046–2047 ("Events under the losing same-epoch lease and all lower epochs MUST be preserved
  in a divergent branch and MUST NOT affect authoritative state.").
- Every instrument ran on immutable `git archive` copies under
  `/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/BUG-260917-3lddu0/review-rev1/`
  (`cand/`, `trunk/`, `ckpt/`, plus per-purpose copies `cand-weak/`, `cand-digest/`,
  `cand-oldtests/`, `cand-suite/`, `trunk-probe/`, `plants/plants-source/`, `harness-run{1,2}/`).
  The Story worktree, its index, branch and HEAD were not mutated (`git status --short` = the same
  17 entries before and after; the only git commands against the live worktree were read-only:
  `ls-remote`, `diff --check`, `ls-files`, `archive`, `rev-parse`). Host load 8–24 throughout (the
  other Story was producing concurrently); wall-clock and load are stamped at the top of every log.
- Evidence archive: `BUG-260917-3lddu0_review-evidence-rev1.tar.gz` (real gzip; `REVIEW-MANIFEST.txt`
  carries sha256 of every member). Test masks are stated beside every row count below.

## Verdict: CHANGES REQUESTED → `to-dev`

The production change is small, correctly placed and, for the class the task names, correct: on the
candidate every entry that reaches `Repository.AppendEvent` refuses a superseded-lease append while
the tail still sits on that lease, the losing bytes are preserved, the chain is untouched, and the
composing writers keep working under the winner. I reproduced both defects on trunk and both
refusals on the candidate myself. What is not acceptable at story_final is the evidence record:
the second half of the delivery (the §2.4 profile-source property independent of the append gate)
does not exist in the tree yet is claimed in the registry, four landed tests were rewritten to pass
under the new admission with no argument and with results that assert "0 changed"/"no moved
inputs", the shipped "narrowing" mutants are arm-deletes in disguise, one of the two preservation
arms is unmeasured, and the producer evidence archive was never attached to the board. Two P1, three
P2, five P3. Coverage as I measured it, not as restated: task AC row 1 driven at **4 of 4** entries;
task AC row 2 driven **only through row 1's gate** (0 of 1 independent owner, 0 of 1 narrowing
mutant); task-specific DoD rows **6 of 6 driven in the tree, 2 of 6 clean of findings**.

## What I reproduced (probes 9 and 15; `logs/10-probes-trunk.log`, `logs/11-probes-cand.log`)

Adapted from `TASK-260830-1geqhj_review-evidence-rev1.tar.gz` `evidence/probes/zz_review_probe_test.go`
(lines 413–449 and 686–716) to the current `Emit`/`EmitParked` API (the archived file uses
`remoteTakeover` and pre-fencing `EmitParams`; the current entries require `Presented` +
`Observation`, and the observation is a caller snapshot, so a stale-but-well-formed one reaches
`AppendEvent`). Probe bodies: `probes/zz_review_probe_test.go`.

| Probe | Entry | trunk 799c338 | candidate 8042bcc |
| --- | --- | --- | --- |
| 9 | `Repository.AppendEvent`, A/1 seq 2 after B/2 won, tail at A | admitted, `ref.Position=1`, chain 1→2 | `ErrStaleLease` "event lease epoch 1 precedes winning lease epoch 2", chain stays 1 |
| 9 | `EmitParked` (stale observation) | admitted seq 3 | `ErrStaleLease` |
| 9 | `Emit` profile.changed (stale observation) | admitted, `sha256:38ede56b…` (the archived probe-15 event) | `ErrStaleLease` |
| 9 | `Transactor.SetProfile` to=yolo under A/1 | admitted, `NewProfile="yolo"` | `ErrStaleLease` |
| 15 | append + `Projector.Project` → `Derive` (session head, the entry the producer's own baseline probe used) | admitted; pair `{yolo, sha256:38ede56b…, HasSource:true}` — the yolo pair that `provhost.ProfileMapping("codex", yolo)` maps to `--dangerously-bypass-approvals-and-sandbox` (`baseline-probe-15-direct.log`) | refused; pair `{standard, no source}` |
| 15 | `Run` under A2/2 | `parked`, empty profile (my `runRequest` world; the archived rev1 launch is not reproducible through today's `Run`, which parks here on both trees) | `parked` |

Before state named: the losing-lease `profile.changed` drives the session-head effective profile to
`yolo` with source = the losing event, i.e. the `--dangerously-bypass-approvals-and-sandbox` mapping.
The producer's `baseline-probe-9-direct.log` / `baseline-probe-15-direct.log` agree.

## Findings

### P1-1 The §2.4 profile-source property has no owner independent of the append gate; the registry binds one anyway

The producer brief required two properties with two sets of tests ("make the §2.4 property hold
independently of the append gate — a losing-lease profile.changed must never be the effective
profile source even if some other path admits the event … do not let one of them stand in for the
other"). The candidate changes no production line outside `internal/sessrepo`; `sessprofile.Derive`
is byte-identical to trunk and takes `(record, events)` only — it cannot see the lease store, and a
losing event that extends its own lease's tail is a perfectly continuous chain to it.

Measured (`logs/12-property2-under-disabled-gate.log`, copy `cand-weak/` with `checkWinningLease`
returning nil as an instrument, not as evidence):

- Probe 15 under the disabled gate: `Projector.Project` → `{Profile:yolo Source:sha256:38ede56b…
  HasSource:true}`. The property does not hold at the session-head derivation entry (`LoadProfile`/
  `Projector.Project`/`SetProfile`'s from-end, and `axpane.deriveProfile` whenever no closure heads
  apply) once anything admits the event.
- `TestLosingLeaseProfileEventIgnored` fails at `rework_test.go:662` — the `AppendEvent(...) !=
  ErrStaleLease` assertion — and never reaches its `Derive`/`Run` assertions. Its kill is the append
  gate's kill. One set of tests covers both properties, which the reviewer brief told me to record
  as a finding.
- The shipped harness never runs `TestLosingLeaseProfileEventIgnored` (its `-run` masks are the four
  append tests per arm), so task AC row 2 ("pinned by tests and by a narrowing mutant") has no
  mutant at all.
- `internal/traceability/ownership.v0.7.0.json` binds acceptance case
  `story-260917-losing-lease-profile-source` with production owner
  `internal/sessprofile/profile.go:Derive` to §2.4 clause line 666. `Derive` does not implement that
  clause with respect to the lease store; the behaviour the test observes is produced by
  `sessrepo.checkWinningLease`. tracecheck cannot see this (it pins declarations, which I confirmed
  fail closed when renamed — `logs/33-registry-binding-spotcheck.log`); it is a semantic
  misbinding a reviewer has to catch. (The README sentence for this case — "adds the production
  append-admission refusal/source proof to clause 2.4#2" — describes the test honestly; it is the
  registry's production owner that is wrong.)
- Only `Run`'s §13.10 closure derivation (`DeriveForHeads` over the resumed checkpoint's heads,
  landed by TASK-260830-1geqhj) is independent of the gate, and only on the paths that derive from a
  closure. That is why trunk's `TestLosingLeaseProfileEventIgnored` passed with the event admitted.

The producer brief also said: "If the fix appears to need a change outside that boundary, STOP and
say so in the results rather than widening." Neither happened — the results claim the property is
proven. Rework may take either of two honest routes (see the scope section).

### P1-2 Four landed tests were rewritten to pass under the new admission, unargued, and the results assert the opposite

`internal/axpane/rework_test.go:TestRunPostWindowSupersedes`, `rev3_test.go:TestRunSupersededPairIdenticalRetry`,
`rev3_test.go:TestRunCreateFromStoppedPostWindow` had `successorLease(...)` and
`publishCheckpoint(...)` swapped; `rework_test.go:TestLosingLeaseProfileEventIgnored` (the landed
probe-15 regression test) was rewritten from "admitted event never drives `Run`" to "append is
refused". Running the checkpoint's (`964fa47`) versions of those two test files against the
candidate production (`logs/41-old-tests-vs-candidate-production.log`, copy `cand-oldtests/`):

```
rework_test.go:274: AppendEvent(session.idle) error = event lease epoch precedes the chain head: event lease epoch 1 precedes winning lease epoch 2
rev3_test.go:684:   AppendEvent(session.idle) error = … epoch 1 precedes winning lease epoch 2
rev3_test.go:526:   AppendEvent(session.idle) error = … epoch 1 precedes winning lease epoch 2
rework_test.go:518: AppendEvent(profile.changed) error = … epoch 1 precedes winning lease epoch 2
--- FAIL ×4
```

Four landed inputs moved from admitted to `ErrStaleLease`: the class is "a checkpoint publication
(`session.idle`/`session.stopped`) or `profile.changed` authored under the superseded lease A after
the successor lease already won". The producer's own first importer run saw exactly these failures
(spawn log lines 34977–34980) and then edited the tests. On the merits the fixture correction is
right — the old order (successor CAS first, old owner's stop events after) is the bug pattern itself;
a graceful takeover publishes the stop with its checkpoint before the successor lease names that
checkpoint as its handoff base — and the tests' assertions about `Run` are unchanged in meaning. But
nothing in `BUG-260917-3lddu0_results.md`, the conformance matrix, `LOGBOOK.md` or either
TRACEABILITY file mentions the four edits, gives the argument, or cites §5.2/§5.3 for it, while the
results say "0 changed existing outcomes", the matrix says "No existing importer input moved from PASS
to FAIL or changed arms" and lists the moved inputs as "the seven additions". The reviewer brief
grades a landed regression test rewritten without an argument as P1, and an inflated claim as a
finding; both apply. The name-keyed outcome grid (mine reproduces theirs: 2009→2016 test rows, +7, 0
changed, 0 removed; trunk→candidate 1979→2016, +37, 0 changed) is exactly the instrument that
cannot see this, which the brief warned about.

### P2-1 The shipped "narrowing" mutants are arm-deletes in disguise

`internal/sessrepo/testdata/mutate_append_admission.py` plants
`if event.leaseEpoch < winner.Epoch && event.leaseID == "never-a-real-lease"` and the same suffix on
the same-epoch arm. No decodable event can carry that lease ID (decode requires a UUIDv4), so each
plant admits the entire rejected class, i.e. `&& false`. The matrix, README, LOGBOOK and TRACEABILITY
all say "narrowing mutants, not deletes" / "admit exactly one member". Reran twice in isolation
(`harness-run{1,2}/`, `logs/20-harness-passes.log`): both N rows KILLED ×2 (exit 1) with identical
kill lists, comment control SURVIVED, NOT_APPLIED and COMPILE_OR_HARNESS_FAILURE classified, control
before/after exit 0, zero `__pycache__`/`.pyc` — the harness is sound; the rows just do not measure
what they claim. My own true narrowings (below) show the gate IS pinnable by narrowing, so this is
an evidence defect, not a code defect.

### P2-2 Preservation of same-epoch losers is unmeasured; lower-epoch preservation is pinned at 1 of 4 entries

Reviewer plant `R6-no-preserve-on-same-epoch` (drop `ErrDivergentBranch` from the new refusal
block's preservation condition) **SURVIVED ×2** against the full `sessrepo`+`sessprofile`+`axpane`
suites (`plants/pass{1,2}-R6-*.log`, exit 0). `R5-no-preserve-on-stale` is KILLED ×2 by exactly one
test, `TestAppendEventRefusesSupersededLeaseWhileTailStillMatches` (direct entry; the SetProfile/
Emit/EmitParked cells never read the preserved blob). Production does preserve both (my idempotency
probe `logs/15-idempotency-cand.log` stats the same-epoch losing blob on disk, and a retried refusal
returns the same sentinel with the chain untouched), but "preserved as an immutable blob" is a
censusable claim repeated in results/LOGBOOK/TRACEABILITY/README and half of it has no test. §5.3
line 2047 is the clause the Story bound `story-260917-append-winning-lease-admission` to.

### P2-3 The producer evidence archive is not on the board

`BUG-260917-3lddu0_producer-evidence.tar.gz` (706,575 bytes) exists only at
`.temp/STORY-260917-kf9g1h/worktree/.temp/BUG-260917-3lddu0/` — an ephemeral worktree path. The board
outcome list holds results, matrix, patch, validation log and spawn logs; no evidence archive, so the
per-plant raw logs, subprocess exits, importer JSONL and validation logs the brief required are not
durable and the "verify what you attached by reading it back" step was not done for it. I evaluated
the on-disk copy; the finding is the missing attachment.

### P3-1 Same-epoch arm pinned along one direction per entry

`R4a` (admit same-epoch IDs bytewise smaller than the winner) is KILLED only by the two axpane cells
(`…0005`<`…0006`, `…0007`<`…0008`); `R4b` (admit greater IDs) only by the sessrepo/sessprofile
cells (`C`>`B`). Both halves are pinned somewhere, but each census cell pins one half. Worth one
extra member per side, not a rework on its own.

### P3-2 A never-minted higher-epoch lease is admitted and becomes the profile source (pre-existing)

`profile.changed` under `(epoch 3, dddddddd-…)` with the lease store at `(2, B)`: admitted by
`AppendEvent`, `Projector.Project` → `{yolo, HasSource:true}`, on trunk and candidate alike
(`logs/13-probes2-cand.log`, `logs/14-probes2-trunk.log`). `checkWinningLease` admits any epoch above
the winner. The producer's TRACEABILITY states this as a bound but names no owner, which the DoD
requires. Since a successor lease is always CAS-installed before its first event, "the new event
must be authored under the winner exactly" would refuse this class without touching any legitimate
flow; at minimum the bound needs an owner and the §2.4 word "ambiguous" needs to be addressed.

### P3-3 Census omits the `Run → EmitParked` entry

`Run` reaches `EmitParked` → `Emit` → `AppendEvent`; through `Run` a losing lease cannot reach the
gate except by an interleaving between `Observe` and `AppendEvent` (the durable gate is what closes
that window). A cell "unreachable except by interleaving; covered by the durable gate" belongs in the
table.

### P3-4 Empty-lease-store bound stated without an owner

With no lease record the gate is skipped and `AppendEvent` admits `(7, B)` at sequence 1
(`TestReviewNoLeaseStoreBound`). Stated in TRACEABILITY as a bound; needs an owner.

### P3-5 Carry-forward dispositions: 4 of 4 recorded (none silently dropped)

(1) `decide.go` "lapsed grants refuse" qualifier: deferred with reason (no production axpane line
touched; the note said fold it in when axpane production is next touched) — acceptable. (2)
`observeRemoteWinner` redundancy: recorded only — as asked. (3) no-grant §4.2-step-4: recorded as
unresolved, not decided — as asked. (4) evidence hygiene: candidate TREE OID recorded (both
`276c515c…` pre-logbook and `8042bcc…` final) and masks stated beside counts — closed; the new
hygiene miss is P2-3.

## Gate × entry census, verified and attacked

Production callers of `AppendEvent` (grep, non-test): `internal/axpane/events.go:224` (`Emit`, and
`EmitParked` through it) and `internal/sessprofile/setprofile.go:100` (`SetProfile`); plus the direct
entry. The producer's four columns are the complete direct-caller set; `Run` is P3-3. Importer set of
`internal/sessrepo` derived with `go list` (direct and transitive, test imports included): exactly the
producer's nine packages. The gate is one-sided (one implementation, four wrappers), so the "symmetry
line" is across arms, 4/4 — correct as stated.

Per-entry kill attribution measured with my plants (full three-package suite, no `-run` mask,
`plants/plants.json`, two passes, control SURVIVED ×2, every plant restored and diffed):

| Plant | Kind | Pass 1 / Pass 2 | Killed at entries |
| --- | --- | --- | --- |
| R1 gate compares against `leases[0]` (oldest) instead of the winner | narrowing, call site | KILLED / KILLED (10 rows) | direct, SetProfile ×2 arms, Emit ×2, EmitParked ×2, profile test |
| R2 `leaseEpoch+1 < winner.Epoch` (admit one-epoch-behind) | narrowing | KILLED / KILLED (6 rows) | direct, SetProfile lower_epoch, Emit, EmitParked, profile test |
| R3 gate wired only when `len(leases) > 2` | narrowing, call site | KILLED / KILLED (10 rows) | all four entries, both arms |
| R4a same-epoch admit smaller ID | narrowing | KILLED / KILLED (2 rows) | Emit, EmitParked only |
| R4b same-epoch admit greater ID | narrowing | KILLED / KILLED (3 rows) | direct, SetProfile only |
| R5 no preservation on lower-epoch refusal | narrowing, preservation arm | KILLED / KILLED (1 row) | direct only |
| R6 no preservation on same-epoch refusal | narrowing, preservation arm | **SURVIVED / SURVIVED** | — |
| R7 SetProfile mints `LeaseEpoch+1` | wrong-arm at a composing entry | KILLED / KILLED | SetProfile (literal `ErrStaleLease` asserted; landed crash-replay test also reddens) |
| R8 lower-epoch arm reports `ErrDivergentBranch` | same-exit code swap | KILLED / KILLED | direct, SetProfile, Emit, EmitParked (tests assert the literal sentinel, not "an error") |
| C harmless comment | control | SURVIVED / SURVIVED | — |

## Composing writers driven (`logs/13-probes2-cand.log`, `TestReviewComposingWritersUnderTheWinner`)

Under the durable winner on the candidate: `Emit` (successor, seq 1) admitted; `SetProfile` to yolo
admitted with `PreviousProfile=standard`, and the session-head pair becomes `{yolo, <that event>}`;
identical retry replays the same event ID; `EmitParked` under the epoch-1 winner admitted at seq 3.
Same outcomes on trunk (`logs/14-probes2-trunk.log`). Refusal retries are idempotent and leave the
chain untouched (`logs/15-idempotency-cand.log`).

## Story-close items (all green on the exact tree)

- Registry: `internal/traceability/ownership.v0.7.0.json` (adopted). Diff against the checkpoint is
  purely additive (two acceptance cases, five list insertions); trunk did not move, so every trunk row
  is present by construction. Semantic misbinding: P1-1.
- Digest: re-derived through the production decode + `json.Marshal` projection from the candidate
  bytes (`logs/30-registry-digest.log`): `ed6f016f73e390071743b09aded229e781d95408b0b9eeb3f134fcb87bb9a893`
  = pinned `reviewedOwnershipCanonicalSHA256`; raw file sha256 `335ca254…`, 229,889 bytes, 154 cases,
  74 groups.
- tracecheck on tree 8042bcc (`logs/31-tracecheck-exact-tree.log`), verbatim:
  `traceability ok: contracts=64 normative_sections=36 acceptance_cases=154 fixtures=33 compatibility_contracts=55 assigned_scopes=0`
  `section coverage: bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574`;
  `-section 2.4` exit 0 (`assigned_scopes=1`); `-section 5.3` exit 1 with the 7/8 partial refusal.
- README "Measured coverage of this repository": fenced line byte-equal to tracecheck's second line;
  `readme_coverage_pin_test.go` is exercised — planting `71/574` and `Five bindings` into a README copy
  reddens it both times, control green (`logs/32-readme-pin-plant.log`).
- Binding spot-checks: renaming `TestLosingLeaseProfileEventIgnored`,
  `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches`,
  `TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches` or `Derive` each fails tracecheck
  closed with the exact declaration named (`logs/33-registry-binding-spotcheck.log`).

## Hygiene (`logs/60-hygiene.log`)

gofmt clean; `go vet ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; zero
`__pycache__`/`.pyc` in the candidate tree and after both harness passes; `git diff --check` clean;
`CLAUDE.md → AGENTS.md` symlink intact; `git diff --name-only 799c338 origin/main` empty (no stale
pre-refresh copies possible).

## Full configured suite on the exact tree (30 commands read from the candidate `task-board.config.json`)

Commands 2–27 and the extra Windows vet ran in `cand-suite/` — a fresh, probe-free `git archive` of
tree `8042bcc` (re-hashed to the same OID before use, `.task-board` included). Commands 1, 28, 30 ran
read-only against the live worktree; 29 against the authoritative board (`TASK_BOARD_DIR`). Per-command
logs and exit files: `suite/cmd-NN.log` / `suite/cmd-NN.exit`; wall-clock and load per command in
`suite/summary.log`. Started 23:15:16 UTC, done 23:30:11 UTC; load 9–27 (the other Story's race gate
overlapped mine).

| # | Command | rc | Elapsed | Note |
| --- | --- | ---: | ---: | --- |
| 1 | gofmt -l over tracked+untracked Go files (live worktree) | 0 | 1s | empty output |
| 2 | go build ./... | 0 | 2s | |
| 3 | go vet ./... | 0 | 1s | |
| 4 | go test ./... -count=1 -v | 0 | 257s | 44 ok, 0 FAIL, 0 `--- FAIL` |
| 5 | go test ./... -race -count=1 -timeout 25m | 0 | 442s | 44 ok, 0 FAIL, 0 DATA RACE; load 14→27 |
| 6 | go test ./... -cover -count=1 | 0 | 162s | 44 ok; sessrepo 86.6%, sessprofile 92.6%, axpane 81.3%, fencing 99.3%, terminstance 86.6% |
| 7–23 | 17 fuzz lanes, 100x, parallel=1 | 0 each | 0–3s each | |
| 24 | tracecheck | 0 | 0s | `acceptance_cases=154 … clauses_discharged=70/574` (verbatim above) |
| 25 | cataloggen -adopted -check | 0 | 1s | silent (0 bytes) |
| 26 | GOOS=linux GOARCH=amd64 go build ./... | 0 | 1s | |
| 27 | GOOS=windows GOARCH=amd64 go build ./... | 0 | 1s | |
| 28 | JSON parse of tracked *.json (live worktree) | 0 | 4s | |
| 29 | task-board validate (authoritative board) | 0 | 6s | standing 179 `MISSING_ACTIVITY` ledger notices, 2812/2812 rows mirrored; none names this task or Story |
| 30 | git diff --check (live worktree) | 0 | 0s | |
| extra | GOOS=windows GOARCH=amd64 go vet ./... | 0 | 1s | |

30 of 30 configured commands exit 0 on the exact tree, plus the Windows vet.

## Measured coverage against the task-specific DoD rows

| DoD row | Call sites | Driven? | Finding |
| --- | --- | --- | --- |
| 1 superseded-lease append refused while tail matches; narrowing mutant | `AppendEvent`→`checkWinningLease`; `SetProfile`, `Emit`, `EmitParked` | driven 4 of 4 entries, both arms | P2-1 (shipped mutants are deletes); pinned by my narrowings R1–R4 |
| 2 losing-lease profile.changed never the effective source; probes 9/15 before/after with yolo named | `Derive` via `LoadProfile`/`Projector.Project`; `Run` | probes reproduced both ways; property holds only through row 1's gate | P1-1 |
| 3 gate × entry census in matrix + TRACEABILITY | — | 8 of 8 cells present; symmetry line present | P3-3, P3-2/P3-4 bounds without owner |
| 4 composing writers unchanged, OUTCOME-keyed over the importer set, moved inputs named | 9 packages | grid reproduced (+7/0/0); 4 moved landed inputs unnamed | P1-2 |
| 5 registry bindings, digest re-derived, tracecheck verbatim, README equal | `internal/traceability` | all verified green | (P1-1 misbinding) |
| 6 four P3 dispositions | — | 4 of 4 recorded | P2-3 new hygiene miss |

Task AC: row 1 **4 of 4** entries driven; row 2 **driven only via row 1** (independent owner 0 of 1,
narrowing mutant 0 of 1). DoD: **6 of 6 driven, 2 of 6 clean**.

## Rework scope (for the producer)

1. **P1-1, choose one and say which.** (a) Give the §2.4 property an owner independent of the append
   gate at the session-head derivation entries (`LoadProfile`/`Projector.Project`/`SetProfile` from-end
   and `deriveProfile`'s head fallback): e.g. derive over events that are in the winning lease's
   handoff-base closure or authored under the winner, refusing/ignoring an older-lease event outside
   that closure; ship its own test whose kill comes from the profile assertion and a true narrowing
   mutant; this touches `sessprofile`/`axpane`, so state the scope widening explicitly. Or (b) keep
   the tree as is and record the truth: the property is discharged only by `sessrepo.AppendEvent`'s
   gate; re-bind `story-260917-losing-lease-profile-source` to the real owner (or fold it into
   `story-260917-append-winning-lease-admission`) with the clause still named; restructure
   `TestLosingLeaseProfileEventIgnored` so the append outcome is logged, not fatal, and the
   `Derive`/`Run` assertions are what kills a gate narrowing; add that test to the harness mask; state
   in results, matrix, TRACEABILITY and README that independence is a BOUND with an owner (the leaf
   that gives the derivation a winner). Either way the registry's production owner for the case must be the
   declaration that actually refuses.
2. **P1-2.** Name the moved class in results and matrix with before/after (admitted → `ErrStaleLease`,
   four named landed tests, the sequence that moved), argue the fixture correction against §5.2
   1832–1836 / §5.3 2046–2047, and replace "0 changed / no moved inputs" with the true statement:
   0 changed under the edited names, 4 landed inputs moved and were re-sequenced. Add the
   old-tests-vs-new-production run as evidence.
3. **P2-1.** Replace the two `== "never-a-real-lease"` rows with narrowings that admit exactly one
   member (one-epoch-behind; same-epoch smaller-ID and greater-ID), keep the delete-shaped rows only
   if labelled as deletes, and add `TestLosingLeaseProfileEventIgnored` to the profile row's mask.
4. **P2-2.** Read the preserved blob in the same-epoch tests (direct at least, ideally one composing
   entry) so R6 dies; the R5/R6 plants are in my archive and can be reused verbatim.
5. **P2-3.** Attach `BUG-260917-3lddu0_producer-evidence.tar.gz` from an absolute path and read it
   back off the board.
6. **P3s.** One extra same-epoch member per side (P3-1); an owner for the higher-epoch and empty-store
   bounds, or the exact-winner gate (P3-2/P3-4); the `Run → EmitParked` cell (P3-3).
