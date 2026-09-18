# TASK-260830-32ypr2 — Review verdict, Change Request revision 4

Reviewer run: RUN-260918-492d65 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-32ypr2-4` revision 4 (story_final), base
`42d5a959aa523572d2745d44f828d41a0b2d72b3` (= `origin/main`, re-fetched at
review start, unmoved), candidate tree
`6c6709039fb80e75a921718769622d91f060264c` (66 paths; patch sha256
`0b7c325f8fdc4cf0d0cb1d960641a19b234f2ac8a4fc0fa0e5071cc9bf5c0815`, equal to
the attached resource and to `git diff base candidate`). The live Story
worktree's temp-index tree OID (`git add -A` into a scratch index) equalled
`6c670903…` before and after this review; `git status` stayed ` M LOGBOOK.md`,
` M README.md`, ` M internal/clonesnap/{TRACEABILITY.md,project.go}`,
` M internal/traceability/{6 files}`, `?? internal/cloneproject/` throughout.
Worktree, index, branch and HEAD were never touched. Authority:
`internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (10331–10546), §10.2, §7.8.

Method: isolated immutable copies of the exact candidate tree
(`git archive 6c670903…` → `.temp/TASK-260830-32ypr2/review-rev4/{candidate,
probe,battery,plant}`; the tracked `.task-board` dropped from the copies and
re-materialized only in `candidate` for the `specpin` board-scope test);
`diff -rq` clean between `candidate` and `battery`/`plant` after every harness
and plant run; `PYTHONDONTWRITEBYTECODE=1` on every harness, zero
`__pycache__`/`.pyc` in the CR tree, the live worktree, or the copies after the
runs (the one `__pycache__` that appeared under `candidate/` was my own
`importlib` listing of the harness, deleted; it never existed in the CR tree).
Every probe named below was executed on a candidate copy; logs, per-plant raw
logs with subprocess exits, and the probe/plant sources are in
`TASK-260830-32ypr2_review-evidence-rev4.tar.gz`.

## Verdict: CHANGES REQUESTED → `to-dev`

No P1. One P2 (new, on a gate the producer did not mutate) and a P3 list. The
rev3 rework (P2-e, P3-u..aa) is closed — Section 0 below — and the refresh onto
`42d5a95` was a MERGE, verified row by row. What remains is the same shape as
P2-e one level down: the closed-shape scale bounds that this leaf's AC (4)
names — 0..64 parents, 1..1024 heads, 1..1,000,000 event IDs, 1..1024 actors —
are reported in matrix section D as "owned by the accepted `internal/clonebundle`
suite", but that suite witnesses only the low edges (and the 1025-actor decode
row): a bound widened by one survives the whole Story suite outright at five of
the eight build/decode sites, and dies at two more only because the low-edge
row's message fragment carries the bound literal — seven of eight sites have no
behavioural witness. Production enforces every far edge (probed again on
this tree); the rework is test-plus-matrix only and small.

## Section 0 — rev3 rework and refresh verification

| Item | Grade | Evidence |
|---|---|---|
| P2-e raw-addressability past offset 0 | CLOSED | `TestNormalizeMultiRecordRefsResolveAtAssertedOffsets` asserts `ref.Offset` (0, `len(l1)+1`, `len(l1)+len(l2)+2`), `ref.Length`, and the resolved bytes for every event of a three-record member with a multibyte first line. Producer row `N-offset-newline` KILLED 2/2 (`raw offset = 150, want 151`). My three rev3 shapes re-anchored against the new row: `RV-offset-line-number`, `RV-evidence-offset-second-line`, `RV-offset-drops-newline` — all KILLED 2/2 (`reviewer-plants/run{1,2}/`). Matrix row 4 and `TRACEABILITY.md:62` cite the test. My own multi-member probe (`logs/probe-rev4.log` PR4-3: two JSONL members + one unknown-class member, six events) resolves every event to its own line inside its own member with the offsets 0/148/0/153/302 and per-member descriptors, and a second projection is byte-identical. |
| P3-u null optional booleans | CLOSED (stated + pinned) | `TestNormalizeNullOptionalBoolsDecodeAsFalse`; `doc.go`, TRACEABILITY, matrix §A row 46 + §D. |
| P3-v body type / required gates | CLOSED (rows shipped) | `body_text_number`, `body_live_attestation_string`, `body_missing_text` + `N-body-text-number`, `N-body-bool-string`, `N-body-required-skip-text` KILLED 2/2; the required-member row is honestly described as a message pin. |
| P3-w empty directive | CLOSED | `empty_directive` + `N-directive-empty` KILLED 2/2; bound text names the low edge. |
| P3-x "nine" | CLOSED | `doc.go:67` says ten. |
| P3-y message cosmetics | DECLINED with rationale | Accepted: the member+line shape is uniform and AC-pinned; the clonebundle helper label is sibling production with verified-correct behaviour. |
| P3-z `body:null` on unknown types | CLOSED (stated + pinned) | `TestNormalizeUnknownTypeNullBodyProjectsOpaque`; matrix §A row 47. |
| P3-aa archive hygiene | CLOSED | `MANIFEST.sha256` has 327 entries, no self-entry, `shasum -c` all OK; `cmds/cmd01..30.log` (+ `cmd05a..g`) each carry the command line and its exit; real gzip; zero `__pycache__`. |
| Refresh was a MERGE | CONFIRMED | `logs/registry-merge-check.log`: all 147 trunk acceptance cases present and unchanged, the 5 Story cases appended after them (order preserved); all 74 trunk ownership rows present, the ptxkqe landing's 4 upgraded bindings (4.B, 4.D, 5.2, 7.A) and its new `section:4.1` row intact; the only rows the candidate changes are `section:7.8` and `section:10.2`, byte-equal to the rev3 candidate's; `unowned_sections` and `source` unchanged. Registry deletions vs trunk are exactly the two superseded `unevidenced` stubs' old fields. |
| Digest re-derived | CONFIRMED | Zeroing the pin in a throwaway copy → the gate prints `614eb55d70b4b83f1ff668b8bfe9830d0655189a3a76f65edefb5159c6710ecb` = the pinned value (`logs/digest-rederivation.log`; pin file restored byte-identical). |
| tracecheck + sections | CONFIRMED | `logs/tracecheck-candidate.log` byte-equal to the two quoted lines (`acceptance_cases=152`, `full=4 … unevidenced=43 … clauses_discharged=70/574`); `--section 7.8` and `--section 10.2` admitted with `assigned_scopes=1`; `--section 13.14.1` refused as designed (no scoped owner). |
| README pin | CONFIRMED, live | `TestREADMEMeasuredCoverageMatchesTracecheckReport` PASS on the candidate (`logs/pkg-traceability-v.log`); planting "Two bindings are `full`" into the README copy reddens it (`logs/readme-pin-plant.log`, exit 1). README figures at all five sites (`152`, `70/574`, Four full, forty-three unevidenced, twelve clauses) agree with the report. |
| README / LOGBOOK / config | CONFIRMED | README +125/−18 (the 18 deletions are the coverage-figure rewrites and the sessadapter paragraph's "Seven … unevidenced" sentence, no reverts); LOGBOOK +38/−0 (leaf block newest-first at the top of 2026-09-18, predecessor blocks replayed); `task-board.config.json` byte-identical to trunk; CR paths ⊆ {LOGBOOK, README, clonebundle, cloneproject, clonesnap, traceability}; `internal/clonebundle` and `internal/clonesnap` byte-identical to the rev3 candidate; this leaf's `clonesnap/project.go` delta over the 2g5be6 checkpoint is comment-only. |

The rev2 class closure was re-driven on this tree, not re-litigated: the 17
Section-0 rows of the rev3 probe (`probes/zz_reviewer_section0_test.go`,
`logs/probe-rev4.log`) reproduce exactly — PA1/PA2 → `opaque_event
frobnicate`, PA3 → `Found:false`, PA4 → `declared`, PA5 → `evt-a5`, PA6/PA7
refused, PC1..PC9 as in rev3 — and `grep -n 'NewDecoder\|Unmarshal'` over the
non-test package still finds only the strict-map reads.

## Findings

### P2 — AC-named scale bounds are unpinned at their far edges

**P2-f The Canonical Event and Canonical Session scale bounds this leaf's AC
(4) names — parents 0..64, head IDs 1..1024, event IDs 1..1,000,000, actors
1..1024 — are enforced by the landed builder/decoder but no committed test in
the Story witnesses the far edge behaviourally at seven of the eight
build/decode sites (five survive a widening outright, two die by message
literal only), and matrix section D reports them as "owned by the accepted
`internal/clonebundle` suite".** Plants against the whole `internal/clonebundle` package, twice
(`reviewer-plants/run{1,2}/`, subprocess exit recorded per run):

| Plant (bound widened by one) | Site | Verdict |
|---|---|---|
| `RV-parents-65-build` (`0, 64` → `0, 65`, `event.go:426`) | build | SURVIVED 2/2 |
| `RV-parents-65-decode` (`event.go:519`) | decode | SURVIVED 2/2 |
| `RV-heads-1025-decode` (`session.go:446`) | decode | SURVIVED 2/2 |
| `RV-eventids-1000001-decode` (`session.go:442`) | decode | SURVIVED 2/2 |
| `RV-actors-1025-build` (`session.go:308`) | build | SURVIVED 2/2 |
| `RV-heads-1025-build` (`session.go:335`) | build | KILLED 2/2 — by the low-edge row's message fragment only (`head_event_ids carry 0 IDs, want [1..1025]` ≠ `"[1..1024]"`); no test feeds 1025 heads |
| `RV-eventids-1000001-build` (`session.go:331`) | build | KILLED 2/2 — same census kill (`want [1..1000001]` ≠ `"[1..1000000]"`); no test feeds 1,000,001 IDs |
| `RV-actors-1025-decode` (control) | decode | KILLED 2/2 — the committed `actors = make([]any, 1025)` decode row |

Production is correct on this tree: `logs/probe-clonebundle-bounds-rev4.log`
(the rev3 probe re-run) shows 64 parents / 1024 heads / 1024 actors /
1,000,000 event IDs admitted and 65 / 1025 / 1025 / 1,000,001 refused at the
builder, the million-event session sealing in 3.1 s and decoding in 3.0 s. The
finding is the measurement: the clonebundle matrix (row 16/17) pins only the
"1024-actor maximum" and the raw_refs/reason-codes maxima, states no bound for
the other far edges, and this leaf's matrix D delegates to it. A widening by one
of an AC-named bound survives the Story suite. This is the P2-e shape one level
down — an AC-named property reported as measured where the suite reaches only
the trivial side — and it is test-only to close.

Ship (test-plus-matrix, no production change): far-edge rows in
`internal/clonebundle` for parents 65 (build + decode), heads 1025 (build +
decode), actors 1025 (build; the decode row exists), and event IDs 1,000,001
(build; decode too unless its cost is stated as a bound with the measured
seconds), each asserting the refusal; one widening narrowing per site in
`internal/clonebundle/testdata/mutant_harness.py` that the new rows kill;
the sibling battery re-run twice with per-plant logs; matrix section D and
`internal/clonebundle/TRACEABILITY.md` citing the rows instead of "owned by
the accepted suite".

### P3 — unpinned shapes and prose (production correct in each case; every plant below ran twice against `-run TestNormalize` on the plant copy unless noted)

- **P3-bb Unterminated final record is unwitnessed.** Every committed JSONL
  member is built by `jsonl()` (trailing newline); the two non-`jsonl` members
  are single-line. `splitRecordLines` admits an unterminated last line
  (probe PR4-2: two records without a trailing newline project with offsets
  0 / 144 and resolve exactly), but `RV-unterminated-tail-dropped-multiline`
  (the unterminated last record of a multi-line member silently dropped)
  SURVIVED 2/2 — a "never dropped" violation on a realistic log shape that no
  test reaches. (My first shape, `RV-unterminated-tail-dropped`, was KILLED
  only by the binary-member refusal row — a single-line member — which is why
  the multi-line shape is the measured one.) Add one row with a member that
  lacks a trailing newline, asserting the last event exists with its offset,
  or state the shape as a bound.
- **P3-cc Per-type body registries are pinned only through the shared loop.**
  `N-body-member` narrows `decodeBodyObject`'s loop for `escalate` on a
  message body; the six per-type `allowed` maps are separate registry sites.
  `RV-allowed-call-live-followup` (the tool/call map admits `live_followup`,
  which the call decoder never reads) SURVIVED 2/2 while the class control
  `RV-allowed-message-escalate` (the message map admits `escalate`) KILLED
  2/2. A fact wrongly registered on one type is silently ignored — no live
  effect, but exactly the "faithful kind while ignoring a fact" rule the
  package states. Add one unknown-member row per type that carries an
  optional member (`live_followup` on a call, `live_attestation` on a
  result), or state the per-type registry as a bound.
- **P3-dd Usage fold pinned along the input axis only.** `usage_ledger_overflow`
  overflows `input_tokens`; `RV-usage-output-arm-widened` (`output > maxUint53`
  → `> maxUint53+1`, admitting a source output total of exactly 2^53)
  SURVIVED 2/2. Add the output-axis row or state it.
- **P3-ee Decode-side raw_refs sortedness (clonebundle).** The decode suite
  pins the duplicate (`refs[0], refs[0]`) but not two distinct unsorted refs:
  `RV-rawrefs-unsorted-decode` (`<= 0` → `== 0` at `event.go:198`) SURVIVED
  2/2 against the whole package, while the build side
  (`RV-rawrefs-unsorted-build`) KILLED 2/2 and `RV-rawrefs-max-decode-65537`
  KILLED 2/2. Owner `internal/clonebundle`; one decode row closes it.
- **P3-ff CRLF members are admitted with the CR inside the record range.**
  Probe PR4-1: `l1\r\nl2\r\n` projects two events whose ranges include the
  trailing `\r` (lengths 143/134, offsets 0/144) and resolve exactly; the
  strict frame gate tolerates the trailing whitespace. Consistent with
  raw-addressability, but unstated; state it beside the "one trailing newline
  is the terminator" sentence.
- **P3-gg Prose.** Matrix section D (`live_followup:null` row) names
  `TestNormalizeNullOptionalBoolsDecodeAsEmpty`; the test is
  `TestNormalizeNullOptionalBoolsDecodeAsFalse` (sections A and C are right).
  `readme_coverage_pin_test.go` adds an unused `"forty-seven"` number word
  (harmless).
- **P3-hh `subagent:` with an empty name is a message-only arm.**
  `RV-actor-empty-subagent-name` (`!ok || name == ""` → `!ok`) SURVIVED 2/2:
  the record then refuses at `checkActors` ("no mapped UUID") and a mapped
  empty name refuses at the landed builder (`name:string[1..512]`). State or
  add the row (`bad_actor_shape` uses `boss`).

## What is clean (verified, not read)

1. **Invariant 1**: rows 1–5 reproduce on this tree; `N-unknown-kind`,
   `N-unknown-kind-wholly`, `N-unknown-member` KILLED 2/2; whole-member
   opaque unit offset 0 / full length resolves (PR4-3); top-level array,
   string, number and `null` lines refuse "not a JSON object" (PR4-6); a
   known type with a non-object body (`[]`, `"text"`, `7`, `null`, `true`)
   refuses (PR4-7); a mid-member blank line refuses naming line 2 (PR4-4); a
   whitespace-only line refuses (PR4-5).
2. **Invariant 2**: rows 6–10 reproduce; `N-protection-encrypted`,
   `N-protection-signed`, `N-foreign-instruction`,
   `N-foreign-payload-authority`, `N-usage-target` KILLED 2/2; a foreign
   `tool/definition` claiming `live_attestation` refuses, foreign unprotected
   calls fold as inert history (`fc1` aborted, `fc2` completed/error), and
   foreign usage lands in the source ledger only with target totals zero
   (PR4-8).
3. **Invariant 3**: the five-row matrix plus
   `TestNormalizeResultAfterBoundaryCompletes`,
   `TestNormalizeProtectedToolCallStaysOpaqueHistory`,
   `TestNormalizeLiveSurfaceStaysEmpty` reproduce; `N-tool-incomplete`,
   `N-tool-orphan`, `N-callable`, `N-callable-false`, `N-live-followup`,
   `N-result-status-pending`, `N-pending-completed` KILLED 2/2;
   `N-pending-aborted` honestly labelled add-arm; the live surface is empty in
   every probe.
4. **Invariant 4**: 65536 inline / 65537 blob-referenced and byte-exact
   (`TestNormalizeInlineContentBoundary` compares the resolved bytes);
   `N-overflow-bound`, `N-overflow-truncate` KILLED 2/2; my
   `RV-overflow-descriptor-skipped` (descriptor bytes never installed) KILLED
   2/2 by the same test; `RV-head-first-event` and
   `RV-assistant-visibility-internal` KILLED 2/2 by
   `TestNormalizeSealsSessionShape`; parents-chain/main-parent narrowings
   KILLED 2/2; the synthesized coupling (null native ID + non-null core
   operation) is enforced and mutated in clonebundle (`N-synth-operation`).
5. **Determinism / durability**: `TestNormalizeIsDeterministic`,
   `TestNormalizeActorOrderIsDeterministic`, idempotent shared-sink replay and
   no-partial-bundle reproduce; PR4-3's second projection of a six-event,
   three-member capture is byte-identical; `N-actors-sort` KILLED 2/2;
   13.14.2–13.14.5 non-ownership stated in matrix D with owners.
6. **Story-close**: see Section 0 (merge, digest, tracecheck, README pin,
   LOGBOOK). Of the 53 distinct test names the matrix cites, 52 exist (38 in
   `cloneproject` — the package's whole top-level set — and 14 in
   `clonebundle`/`clonesnap`/`sessadapter`, the latter also checked by
   `tracecheck`'s declaration gate); the 53rd is the section-D typo of P3-gg.
   All 26 cited subtests exist, all 81 harness rows are cited and all cited
   rows exist; `TestNormalizeRefusalTable` has 50 subtests.
7. **Hygiene**: gofmt 0 files on the CR's Go files, `go vet`, `go build`,
   linux and windows builds, `cataloggen -adopted -check`, JSON parse sweep,
   `git diff --check` all exit 0 (`logs/cmd*.log`); zero `__pycache__`/`.pyc`.
8. **Suite**: `go test ./... -count=1 -v` on the candidate copy with the
   board materialized: 44/44 packages ok, 2794 top-level PASS, 0 FAIL, 2
   pre-existing environment SKIPs (`logs/cmd04-full-suite-v.log`, exit 0).
   `-race` on cloneproject, clonebundle, clonesnap, traceability/...: ok, 0
   `DATA RACE` (`logs/cmd05-race-story-packages.log`). Coverage: cloneproject
   91.7%, clonebundle 87.1%, clonesnap 87.5%. Package suite 38 top-level + 64
   subtests PASS, 0 FAIL (`logs/pkg-cloneproject-v.log`). Producer battery
   reproduced twice on the battery copy: 81/81 `ok:` (80 KILLED + `C-doc-comment`
   SURVIVED), harness exit 0 both runs, one raw log per plant per run with the
   subprocess exit (`mutants/producer-run{1,2}/`), battery copy byte-identical
   after each run. Sibling harnesses re-run once on this tree (the packages are
   byte-identical to the rev3 candidate, where they were run once more):
   clonebundle 67/67 (66 KILLED + control), clonesnap 44/44 (43 KILLED +
   control), harness exits 0, per-plant logs under `mutants/sibling-*-run1/`.
   The other 39 packages' race lanes, the 17 fuzz gates and `task-board
   validate` are accepted from the CR construction summary
   (`required=30 green=30 failed=0 missing=0`) and the producer's archived
   `cmds/cmd05a-g` (44 ok, 0 DATA RACE), `cmd07-23` (all exit 0) and `cmd29`
   (exit 0) — not re-run by me.

## Coverage ratio I measured

47 of 47 matrix rows are driven through `Normalize` by a named committed test
over `clonesnap.Capture` output (every cited test, subtest and harness row
exists and passes). Clean at the class level on this leaf's own package: 47 of
47 — row 4 is now clean at every offset. The Story-level closed-shape claim in
matrix section D is what P2-f measures: 4 of the 4 AC-named scale bounds carry
no far-edge witness at build or decode in the committed Story suite (1 of 8
sites witnessed, 2 killed by message literal only, 5 survived). Producer
battery: 80 KILLED + 1 control SURVIVED, 2/2. Reviewer plants: 27 rows per
run, 2 runs (25 distinct names — `RV-parents-65-build` was re-run against the
whole clonebundle package and `RV-overflow-install-unverified` re-planted after
a compile failure), every outcome identical across the two runs. KILLED 13:
`RV-offset-line-number`, `RV-evidence-offset-second-line`,
`RV-offset-drops-newline`, `RV-allowed-message-escalate`,
`RV-overflow-descriptor-skipped`, `RV-head-first-event`,
`RV-assistant-visibility-internal`, `RV-rawrefs-unsorted-build`,
`RV-rawrefs-max-decode-65537`, `RV-actors-1025-decode`, `RV-heads-1025-build`
and `RV-eventids-1000001-build` (both by the low-edge message literal only),
`RV-unterminated-tail-dropped` (killed by the single-line binary-member row,
i.e. for the wrong reason; superseded by the multi-line shape). SURVIVED 12:
`RV-parents-65-build`, `RV-parents-65-decode`, `RV-heads-1025-decode`,
`RV-eventids-1000001-decode`, `RV-actors-1025-build` → P2-f;
`RV-unterminated-tail-dropped-multiline` → P3-bb; `RV-allowed-call-live-followup`
→ P3-cc; `RV-usage-output-arm-widened` → P3-dd; `RV-rawrefs-unsorted-decode` →
P3-ee; `RV-actor-empty-subagent-name` → P3-hh; `RV-overflow-install-unverified`
(the descriptor/bytes agreement check before the overflow write bypassed by a
direct `PutBlob`: defence-in-depth over bytes the same call just hashed,
unmeasurable without a builder fault — denominator only); `RC-control-comment`,
the applied harmless control. COMPILE_FAIL 1 (`RV-overflow-install-unverified`
v1, unused import; re-planted as the SURVIVED row above). Denominators: 81
producer rows, 111 sibling rows, 27 reviewer rows per run, every row with a raw
log and subprocess exit per run.

## Rework scope (for the producer)

1. P2-f: far-edge rows in `internal/clonebundle` (parents 65 build+decode,
   heads 1025 build+decode, actors 1025 build, event IDs 1,000,001 build and
   decode — or a stated cost bound for the decode side with the measured
   seconds), one widening narrowing per site in the clonebundle harness, the
   sibling battery re-run twice, matrix section D and clonebundle
   TRACEABILITY citing the rows. Test-plus-matrix only; no production change.
2. P3-bb/cc/dd/ee/hh: one row each (unterminated last record with its offset;
   per-type unknown optional member on call/result; output-axis usage
   overflow; decode-side unsorted distinct raw_refs; empty subagent name) or
   the message-only / bound statement where you choose not to pin.
3. P3-ff/gg: state the CRLF shape; fix the section-D test name; drop the
   unused number word or leave it.
4. Re-run the cloneproject battery twice on the final tree, re-run tracecheck
   (no registry change expected, pin unchanged), report the measured ratio;
   keep the candidate UNCOMMITTED; if `origin/main` moves, `refresh-candidate`
   first and audit the co-owned files as in rev4.
