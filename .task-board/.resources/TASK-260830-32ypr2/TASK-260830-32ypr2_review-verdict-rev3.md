# TASK-260830-32ypr2 — Review verdict, Change Request revision 3

Reviewer run: RUN-260918-d009f2 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-32ypr2-3` revision 3 (story_final), base
`c3aae73df906df1b5aa962c7bc3c16437dd920fa` (= `origin/main`, re-fetched at
review start, unmoved), candidate tree
`fc597b8415e4b1440aa2bc12560fdf8f5b668311` (66 paths; patch sha256
`06ace981df5e81452b2857ddf0f47436729a7e0a58d2cb9e54b59fdeda3302e8`, verified
against the attached resource). The live Story worktree's temp-index tree
OID (`git add -A` into a scratch index) equalled `fc597b84…` before this
review; `git status` stayed ` M LOGBOOK.md`, ` M README.md`,
` M internal/clonesnap/{TRACEABILITY.md,project.go}`,
` M internal/traceability/{6 files}`, `?? internal/cloneproject/` throughout.
Worktree, index, branch and HEAD were never touched. Authority:
`internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (10331–10546), §10.2, §7.8.

Method: isolated immutable copies of the exact candidate tree
(`git archive fc597b84…` → `.temp/TASK-260830-32ypr2/review-rev3/{candidate,
probe,battery,plant}`; the tracked `.task-board` dropped from the copies and
re-materialized only for the `specpin` board-scope test); `diff -rq` clean
between `candidate` and `battery`/`plant` after every harness and plant run;
`PYTHONDONTWRITEBYTECODE=1` on every harness, zero `__pycache__`/`.pyc` after
the runs. Every probe named below was executed on the candidate copy; logs,
per-plant raw logs with subprocess exits, and the probe sources are in
`TASK-260830-32ypr2_review-evidence-rev3.tar.gz`.

## Verdict: CHANGES REQUESTED → `to-dev`

No P1. One P2 (new, outside the rev2 rework scope) and a P3 list. The rev2
rework is CLOSED AT THE CLASS, not only at the witnessed vectors — Section 0
below is unqualified. What remains is one AC-named invariant that the
committed suite measures only in its degenerate case: raw-reference
byte ranges resolve to the captured bytes for single-record members
(offset 0) and are never witnessed at a non-zero offset, so the natural
one-token bug in the offset accumulation survives the whole suite while
matrix row 4 reports the property as driven. Production is correct (probed);
the rework is test-plus-matrix only and small.

## Section 0 — rev2 rework verification (RUN-260918-a4ffef findings)

| Finding | Grade | Evidence |
|---|---|---|
| P1-b second lenient decoder after the strict gate | CLOSED AT THE CLASS | `grep -n 'NewDecoder\|Unmarshal' internal/cloneproject/*.go` (non-test): the only decodes are `environ.DecodeStrictObject` at `strictMembers` (`native.go:77`), five `json.Unmarshal(members[field.name], *string)` reads by exact name (`native.go:125`), `Body = members["body"]` (`:135`), `CheckUint53Bounds(members["v"], 1, 1)` (`:136`), body-member reads from the strict body map (`:293`, `:307`, `:390`) and the sealed-manifest map read by exact key (`normalize.go:222`). No struct decode of the line remains. Re-driven on the pristine candidate through `clonesnap.Capture` → `Normalize` (`logs/probe-section0.log`): PA1/PA2 → `opaque_event`, evidence `native_type="frobnicate"` in both document orders; PA3 → `Instruction.Found=false`, payload `authority=low`; PA4 → content `declared`; PA5 → evidence `evt-a5`; PA6 → refused `body carries unknown member "text"`; PA7 → refused `subagent:ghost ... no mapped UUID`; PA8 control admitted. Class, not spelling: every claimed member aliased at once with other spellings (`V`, `NATIVE_EVENT_ID`, `Native_Type`, `ORIGIN`, `PROTECTION`, `ACTOR`, `BODY`) → `opaque_event frobnicate evt-c1`; `"v":1,"V":2` admitted, `"v":2,"V":1` refused; body-level aliases (`TEXT`, `Authority`) refuse as unknown members; an escaped-key duplicate (`"native\u005ftype"` beside `"native_type"`) refuses `duplicate member`; a mapped `subagent:ghost` with an `Actor` alias attributes to the subagent UUID. |
| the case-folded narrowing DIES | CONFIRMED, and the CLASS mutants die too | Producer rows `N-alias-{native-type,origin,protection,body,native-event-id,actor}` KILLED 2/2 (`mutants/producer-run{1,2}/`). My own class plants, run twice against `-run TestNormalize` (`reviewer-plants/run{1,2}/`): `RV-class-struct-redecode` (the rev2 P1-b shape restored: `json.Unmarshal(raw, &envelope)` after the strict map) KILLED, 6 alias subtests fail; `RV-class-equalfold-lookup` (`strings.EqualFold` lookup over the strict map for every claimed string member, order-independent) KILLED, 6 subtests; `RV-class-equalfold-body` (body only) KILLED by `body_alias`. |
| five prose sites | CORRECTED | `native.go:52-57,66-76,104-110` now state the strict-map mechanism; `internal/cloneproject/TRACEABILITY.md:34` ("then read from the admitted map by exact member name with no second decoder"), `:65` (kind aliases in either order stay `opaque_event`), `:102` (envelope values from the strict map; `body_alias`/`evidence_alias`); README:2459-2462 (exact-name sentence, no capability claim); LOGBOOK "REV3 CORRECTION" line names the REV2 sentence false with the mechanism (history kept, 36 added / 0 deleted lines vs trunk); results finding table row 1. |
| P2-d `reasoning_summary` positive row | CLOSED | `TestNormalizeReasoningSummaryProjectsInternal` (kind, `internal`, text block, `exact`); `N-reasoning-summary-routed` KILLED 2/2. |
| P3-m..t | CLOSED | `TestNormalizeCaptureStatusExactForEveryKind` + `N-capture-status-partial`; `TestNormalizeLastNativeInstructionWins` + `N-instruction-fold-first`; `TestNormalizeExternalActorSealsAsExternal` + `N-external-actor-kind`; `TestNormalizeRefusalTable/unknown_type_without_body` + `N-required-member-body`; `TestNormalizeNullTextDecodesAsEmpty` + bound text; `N-pending-aborted` labelled add-arm and the five `N-strict-*` rows disclosed as one gate/two call sites in the harness docstring, TRACEABILITY, matrix and results; LOGBOOK REV3 lines. All KILLED 2/2 in my runs. |
| Battery / tracecheck / digest / suite | REPRODUCED | 76-row battery: 75 KILLED + `C-doc-comment` SURVIVED, 2/2, harness exit 0 in every chunk, one raw log per plant per run (`mutants/producer-run{1,2}/`, `logs/producer-run{1,2}-chunk-*.log`); battery copy byte-identical to the candidate after both runs. `tracecheck` byte-equal to the quoted lines; `--section 7.8` and `--section 10.2` admitted (`logs/section-*.log`). Digest re-derived by zeroing the pin in a throwaway copy → gate prints `4bd01143702d17bc05f187a14312cb868b85276b0f0b6d015c351929eae9bfc0` = pinned (`logs/digest-rederivation.log`). Package suite 95 PASS / 0 FAIL, 35 top-level tests (`logs/pkg-cloneproject-v.log`). |

## Findings

### P2 — raw-addressability is measured only at offset 0

**P2-e Every raw-reference resolution in the committed suite targets a
single-record member; no test witnesses a non-zero `offset`, so the
natural offset-accumulation bug survives the whole suite while matrix
row 4 reports "Evidence ranges resolve to the exact captured bytes
(manifest + descriptor linkage, in range)" as driven.** `grep -n
mustResolveRef *_test.go` finds seven call sites (`foreign_test.go:45,75,
93,117`, `normalize_test.go:44,79,113`) — every one on a one-line member
(line 1, offset 0) or on the whole-member opaque unit (offset 0 by
construction). Plants run twice against `-run TestNormalize`
(`reviewer-plants/run{1,2}/`, subprocess exit 0 both runs):

| Plant | Edit (`native.go` / `dispatch.go`) | Verdict |
|---|---|---|
| `RV-offset-drops-newline` | `offset += uint64(len(raw)) + 1` → `offset += uint64(len(raw))` (`splitRecordLines`; every record after line 1 references bytes shifted by its line index) | SURVIVED 2/2 |
| `RV-offset-line-number` | `recordLine{... offset: offset + uint64(number) - 1 ...}` (line 1 exact, every later record off by its index) | SURVIVED 2/2 |
| `RV-evidence-offset-second-line` | `evidenceFor`: exactly the line-2 record's `Offset` zeroed (its ref slices line 1's bytes) | SURVIVED 2/2 |
| `RV-evidence-length-off-by-one` (control for the length axis) | `Length: record.Length - 1` for exactly the unprotected unknown-type record | KILLED 2/2 (5 tests) |

Production is correct: on the pristine candidate a three-record member
(`message/user` with multibyte text, unknown `frobnicate`, foreign
`reasoning/encrypted`) projects offsets 0 / 142 / 285 with lengths equal to
each line and every `RawRefs[0]` slices back to its own line byte-exact
(`logs/probe-set2.log`, `TestReviewerProbeMultiRecordRefsResolve`). The
finding is the measurement: the AC's "keeps a Source Evidence whose raw
blob/manifest byte-range references actually resolve back to the captured
bytes" is invariant (1)'s raw-addressability, the realistic member holds
many records, and the suite proves the property only where `offset == 0`.
The length axis IS pinned (the control kills), which is why this is a P2
and not a P1: no shipped behaviour is wrong, one axis of an AC-named
property is unmeasured and reported as measured.

Ship: one committed row over a multi-record member that resolves EVERY
event's `RawRefs[0]` against its own line and asserts the offsets
(`0`, `len(l1)+1`, `len(l1)+len(l2)+2`) — the whole-member opaque unit and
a CRLF-terminated line are useful extra records; a harness row for the
offset-accumulation bug (labelled behaviour swap, like `N-overflow-truncate`,
not a narrowing) that the new row kills; matrix row 4 and
TRACEABILITY "raw-addressable" rows citing the multi-record test.

### P3 — unpinned gates and prose (production correct in each case; each plant SURVIVED 2/2 against `-run TestNormalize`, `reviewer-plants/run{1,2}/`)

- **P3-u optional-boolean null bound unstated.** `"live_followup":null`
  and `"live_attestation":null` are admitted as `false`
  (`bodyOptionalBool`: `json.Unmarshal(null, &bool)` leaves false;
  `logs/probe-set2.log` PR2). `text:null`, `directives:null` and
  `ciphertext:null` are stated; these two are not. State or refuse.
- **P3-v body-level type and required-member gates carry no negative
  row.** `body member %q is not a string` / `is not a boolean` /
  `body misses member %q` refuse correctly on the pristine tree
  (`"text":123`, `"text":{}`, `"ciphertext":123`, `"call_id":5`,
  `"live_attestation":"true"`, `{}` — PR3 in `logs/probe-set2.log`) but no
  committed test drives any of them: `RV-member-text-number` (the string
  gate admits exactly the JSON number `123` for `text`) SURVIVED 2/2;
  `RV-body-required-skip-text` (required check skipped for exactly `text`)
  SURVIVED 2/2 — under that plant `{}` still refuses, with `is not a string`
  instead of `misses member` (`logs/under-plant-RV-body-required-skip-text.log`),
  so the required-member loop is a message-only gate. Add one row per gate
  or state the message-only nature.
- **P3-w empty-directive low edge unpinned.** `[""]` and `[null]` refuse
  `directive[0] is not a string[1..4096]` (PR4) but `RV-bound-directive-empty`
  (`< 1` → `< 0`) SURVIVED 2/2; the stated per-edge bound names only the
  upper edges (4096/4097). Name the low edge in the bound or add the row.
  (`RV-bound-toolname-513` SURVIVED 2/2 inside the stated bound — listed for
  the denominator only.)
- **P3-x `doc.go:65` says "the nine fixture-native known types"** — there
  are ten (`knownNativeType`, `native.go:231-242`); results and TRACEABILITY
  say ten.
- **P3-y refusal message shapes.** Actor-selector faults are wrapped twice
  (`record store/session.jsonl line 1 cloneproject: invalid projection:
  actor "subagent:" is outside ...`, PR12); the clonebundle parents-count
  refusal names "canonical session parents" for event parents
  (`logs/probe-clonebundle-bounds.log`, owner `internal/clonebundle`).
  Cosmetic.
- **P3-z `"body":null` on an unknown type is admitted** as an opaque
  event (present-but-null satisfies the required-member rule; the body is
  never read; PR16 in `logs/probe-set3.log`) while a missing body refuses.
  Consistent with the "unknown-type bodies are never read" rule; say so in
  the `unknown_type_without_body` bound text.
- **P3-aa evidence archive hygiene.** `MANIFEST.sha256` lists itself (its
  own line fails `shasum -c`); `cmds/cmd01-03.log` are empty with the exit
  code recorded only in the results table. Cosmetic.

## What is clean (verified, not read)

1. **Invariant 1**: rows 1–5 reproduce; `N-unknown-kind`,
   `N-unknown-kind-wholly`, `N-unknown-member` KILLED 2/2; a case-variant
   type (`Message/User`) is simply unknown → `opaque_event`, never guessed;
   an unknown type with a string, null or nested-duplicate body stays
   opaque with the body never read (`logs/probe-set{2,3}.log`).
2. **Invariant 2**: rows 6–10 reproduce; `reasoning/summary` signed or
   encrypted with a ciphertext body → `opaque_reasoning`, never promoted;
   `Protection` alias cannot downgrade (PA6/PC8); `N-protection-encrypted`,
   `N-protection-signed`, `N-foreign-instruction`,
   `N-foreign-payload-authority`, `N-usage-target` KILLED 2/2; an
   unprotected foreign unknown record carries exactly
   `[unknown_native_event]`.
3. **Invariant 3**: the five-row matrix plus
   `TestNormalizeResultAfterBoundaryCompletes`,
   `TestNormalizeProtectedToolCallStaysOpaqueHistory`,
   `TestNormalizeLiveSurfaceStaysEmpty` reproduce with kind, visibility,
   resolution, reason and empty live surface asserted per row;
   `N-tool-incomplete`, `N-tool-orphan`, `N-callable`, `N-callable-false`,
   `N-live-followup`, `N-result-status-pending`, `N-pending-completed`
   KILLED 2/2; `N-pending-aborted` honestly labelled add-arm.
4. **Invariant 4**: 65536 bytes inline / 65537 blob-referenced and
   byte-exact; the bound is in bytes (32768 × `é` = 65536 bytes inline,
   32769 × `é` = 65538 bytes → blob, round trip exact); directive edges
   1024/1025 and 4096/4097, `native_type` and multibyte `native_event_id`
   512/513, subagent name 128/129 all correct; landed builder far edges
   probed directly: parents 64/65, heads 1024/1025, actors 1024/1025,
   event_ids 1,000,000/1,000,001 (build and decode, 74 MB session in 3.4 s)
   all enforced (`logs/probe-clonebundle-bounds.log`); actor UUID collisions
   through the request map refuse at the builder; title 0/4096/4097 gated;
   `N-overflow-bound`, `N-overflow-truncate`, `N-main-parent`,
   `N-parents-chain`, `N-string-bound`, `N-directive-count` KILLED 2/2; my
   `RV-reasons-unsorted-build` and `RV-heads-unsorted-decode` (unsorted
   array admitted at the landed builder/decoder) KILLED 2/2 by the
   clonebundle suite.
5. **Determinism / durability**: `TestNormalizeIsDeterministic`,
   `TestNormalizeActorOrderIsDeterministic`, idempotent shared-sink replay
   and no-partial-bundle reproduce; `N-actors-sort` KILLED 2/2; 13.14.2–13.14.5
   non-ownership stated in matrix D with owners.
6. **Story-close**: the edited registry is the adopted
   `ownership.v0.7.0.json`, byte-identical to the rev2 candidate the
   previous reviewer verified (additive cases plus the two binding
   upgrades; only the two superseded `unevidenced` bindings deleted);
   digest re-derived = pin; `tracecheck` and both section runs green with
   the quoted lines byte-equal; `traceability` and `cmd/tracecheck` suites
   PASS including `TestREADMEMeasuredCoverageMatchesTracecheckReport`
   (literal "Four"/"forty-seven"/"twelve" figures pinned against the report,
   not bypassed) and `TestV070RegistryRederivesFromTrunkV060Registry`
   (`logs/pkg-traceability-v.log`); README fenced line byte-equal, prose
   figures 63/569, four full, forty-seven unevidenced consistent; LOGBOOK
   newest-first (2026-09-18 block at the top), 36 added / 0 deleted lines vs
   trunk; `13.14.1` deliberately unbound with the reason stated.
7. **Hygiene**: CR paths ⊆ {LOGBOOK, README, clonebundle, cloneproject,
   clonesnap, traceability}; the rev2→rev3 delta touches only
   `internal/cloneproject/*` plus LOGBOOK, README and the cloneproject
   TRACEABILITY; every other CR path is byte-identical to the rev2
   candidate; `task-board.config.json` byte-identical to trunk (30
   commands); `internal/clonesnap/project.go` delta is comment-only; README
   deletions (18) are the required coverage-figure rewrites, no reverts;
   zero `__pycache__`/`.pyc` in the tree or after the runs; gofmt 0 files,
   `go vet`, `go build`, linux and windows builds, `cataloggen -adopted
   -check`, JSON parse sweep, `git diff --check` all exit 0
   (`logs/cmd*-*.log`).
8. **Suite**: `go test ./... -count=1 -v` 40/41 ok in the copy, the 41st
   (`specpin`) failing only for the missing tracked `.task-board`; re-run
   with it materialized → ok (`logs/cmd04-*.log`; 2504 + 1 top-level PASS
   lines, matching the producer's 2505). `-race` on cloneproject,
   clonebundle, clonesnap, traceability/... ok, zero `DATA RACE`
   (`logs/cmd05-race-story-packages.log`); cloneproject coverage 90.6%
   (`logs/cmd06-cover-cloneproject.log`). Sibling harnesses re-run once on
   the candidate copy: clonebundle 67/67 `ok:` (66 KILLED + control
   SURVIVED) and clonesnap 44/44 `ok:` (43 KILLED + control SURVIVED), both
   harness exits 0, per-plant logs under `mutants/sibling-*-run1/`
   (`logs/sibling-*-battery.log`). The other 37 packages' race lanes,
   the 17 fuzz gates and `task-board validate` are accepted from the CR
   construction summary (`required=30 green=30 failed=0`) and the producer's
   `cmds/cmd05a-g` (41 ok, 0 DATA RACE) — not re-run by me.

## Coverage ratio I measured

45 of 45 matrix rows are driven through `Normalize` by a named committed
test over `clonesnap.Capture` output (all 74 matrix-cited test names exist
and PASS; all 76 harness rows exist and are cited). Measured clean at the
class level: 44 of 45 — row 4 (raw references resolve, in range) holds
only where `offset == 0` (P2-e). Producer battery: 75 KILLED + 1 control
SURVIVED, 2/2. Reviewer plants: 15 rows, 2/2 each — KILLED 6
(`RV-class-struct-redecode`, `RV-class-equalfold-lookup`,
`RV-class-equalfold-body`, `RV-evidence-length-off-by-one`,
`RV-reasons-unsorted-build`, `RV-heads-unsorted-decode`); SURVIVED 9
(`RV-offset-drops-newline`, `RV-offset-line-number`,
`RV-evidence-offset-second-line` → P2-e; `RV-member-text-number`,
`RV-body-required-skip-text` → P3-v; `RV-bound-directive-empty` → P3-w;
`RV-bound-toolname-513` inside the stated bound; `RV-contiguity-skip`
structural post-check, index-derived ordinals; `RC-control-comment` the
applied harmless control). Denominators: 76 producer rows, 15 reviewer
rows, every row with a raw log and subprocess exit per run.

## Rework scope (for the producer)

1. P2-e: one committed multi-record row resolving every event's raw
   reference against its own line with the offsets asserted; a labelled
   harness row for the offset-accumulation bug that the new row kills;
   matrix row 4 / TRACEABILITY citing it. Test-only; no production change.
2. P3-u/v/w/z: state the optional-boolean null bound; add rows (or the
   message-only statement) for the body type/required gates; name the
   empty-directive edge; note `body:null` on unknown types.
3. P3-x/y/aa: "nine" → "ten" in `doc.go`; optional message tidy-ups;
   drop the self-entry from `MANIFEST.sha256`.
4. Re-run the battery twice on the final tree, re-run tracecheck (no
   registry change expected, pin unchanged), report the measured ratio;
   keep the candidate UNCOMMITTED.
