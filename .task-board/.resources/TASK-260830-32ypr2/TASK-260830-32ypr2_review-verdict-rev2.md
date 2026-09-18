# TASK-260830-32ypr2 — Review verdict, Change Request revision 2

Reviewer run: RUN-260918-a4ffef (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-32ypr2-2` revision 2 (story_final), base
`c3aae73df906df1b5aa962c7bc3c16437dd920fa` (= `origin/main`, re-fetched at
review start, unmoved), candidate tree
`5832072adaf91325edff9ee636a8b93aa2cc2d99` (65 paths; patch sha256
`1562a1c2b703d19124e760aa14eb022a498208854861d72bfabb594366bf4402`, verified
byte-equal to `git diff c3aae73 5832072`). The live Story worktree's
temp-index tree OID (`git add -A` into a scratch index) equalled `5832072…`
before this review; `git status` stayed ` M LOGBOOK.md`, ` M README.md`,
` M internal/clonesnap/{TRACEABILITY.md,project.go}`,
` M internal/traceability/{6 files}`, `?? internal/cloneproject/` throughout.
Worktree, index, branch and HEAD were never touched. Authority:
`internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (10331–10546; the fidelity
paragraph at 10388–10394), §10.2, §7.8, §1.6 (line 324).

Method: isolated immutable copies of the exact candidate tree
(`git archive 5832072…` → `.temp/TASK-260830-32ypr2/rev2-review/{candidate,
pristine,battery,fixprobe,plantprobe}`; the tracked `.task-board` dropped
from the copies and re-materialized only for the `specpin` board-scope
test); `diff -rq` clean between `battery` and `candidate` after every
harness run; `PYTHONDONTWRITEBYTECODE=1` on every harness, zero
`__pycache__`/`.pyc` after the runs. Every probe named below was executed on
the candidate copy; logs are in `TASK-260830-32ypr2_review-evidence-rev2.tar.gz`.

## Verdict: CHANGES REQUESTED → `to-dev`

One P1, one P2, a P3 list. Every rev1 finding is closed at the vector it
named and the producer's evidence reproduces on the exact tree (Section 0).
What fails is the CLASS the rev1 verdict and the cr2 brief both told this
leaf to close: the projection still reads the record envelope through a
second, lenient decoder after the strict gate, and that decoder diverges
from the admitted member map on case-folded member names. Through that
divergence a foreign instruction reaches the effective snapshot with
`authority=high`, an unknown record is coerced into `user_message`, a
foreign encrypted `reasoning/summary` is promoted to `reasoning_summary`,
sealed content and the evidence identity come from a member the record does
not claim, and a record is re-attributed to `main` — all stamped
`capture_status: exact`, all on the pristine candidate, all admitted by the
committed suite. The tree, the TRACEABILITY, the LOGBOOK and the results
state in so many words that this cannot happen.

## Section 0 — rev1 rework verification (RUN-260918-320878 findings)

| Finding | Grade | Evidence |
|---|---|---|
| P1-a strict-decode fork (lone surrogate / duplicate member rewritten, stamped `exact`) | CLOSED AT THE VECTORS, NOT THE CLASS — see P1-b | `strictMembers` delegates to `environ.DecodeStrictObject` at both frame sites (`native.go:71-80`, `:94`, `:254`); rows `lone_surrogate_text`, `lone_surrogate_envelope`, `duplicate_envelope_member`, `duplicate_body_member` refuse with the literal fault (`pkg-cloneproject-v.log`); narrowings `N-strict-*` KILLED 2/2 in my runs. The envelope is then re-read by `json.NewDecoder(...).Decode(&envelope)` (`native.go:108-111`) — a second decoder whose case-insensitive field matching the strict map does not share (P1-b, `logs/probe-case-alias.log`). |
| P2-a inherited advisories dropped; false prose in `clonesnap/project.go:41-43`, `clonesnap/TRACEABILITY.md:132-147` | CLOSED | One decision line per group (1, 2, 3, 5, 6, 7b deferred with owners; 4 closed as a stated bound; 7a closed by the strict rows; 7c bound) in results + matrix D; both prose sites rewritten to what the tree does and every name they cite exists (`fidelityProfiles` `operations.go:535`, `TestValueVocabulariesMatchSpec`, `TestProjectAdmitsStableEveryProfile`, `TestProjectRefusesUnknownProfile`, `excludedClass`, `alwaysExcludedClass`, `measurePost` unconditional at `capture.go:314`). |
| P2-b `7.8#1` discharged by a test constructing no frame | CLOSED | Registry `7.8#1` → `session-adapter-frame-bound` (production `sessadapter/protocol.go DecodeRequestFrame`, `MaxFrameBytes = 8<<20` at :34; tests `TestDecodeRequestFrameRefusesOversizeFrame` builds a > 8 MiB frame and asserts `exceeds the 8 MiB bound`, `TestFrameBoundEdges` witnesses exact/one-over on request, success and failure gates — `envelope_test.go:127,144`); `clone-projection-fidelity` removed from the 7.8 binding; `--section 7.8` admitted (`logs/section-7.8.log`). |
| P2-c `bodyUint53` fork admitting `"5"` | CLOSED | `environ.CheckUint53Bounds(members[name], 0, maxUint53)` at `native.go:302`; envelope version through the same gate at `:112`; rows `string_usage_number`, `string_envelope_version`; my PB3 sweep (`1.0`, `1e0`, `-0`, `01`, `true`, `[1]`) refuses every literal but `1` (`logs/probe-value-shapes.log`). |
| P3-a..h, i, j, k, l | CLOSED (a, b, d, e, f, g, h rows + narrowings; i and l pinned as bounds; j union of 11 cases with `10.2#2` gaining `localstore-immutable-blob-install`; k relabelled) | Rows present and PASS; `N-actors-sort`, `N-reasoning-encrypted-signed`, `N-reasoning-signed-encrypted`, `N-trailing-admitted`, `N-protected-tools-skip`, `N-result-status-pending`, `N-size-verify`, `N-unknown-protected-body` KILLED 2/2. P3-c stated as a bound (measured again below: `R2-native-type-width-513` SURVIVED 2/2, within the stated bound). |
| Battery / tracecheck / digest / suite | REPRODUCED | 65-row battery: 64 KILLED + `C-doc-comment` SURVIVED, 2/2, one raw log per plant per run (`mutants/producer-run{1,2}/`, `producer-run{1,2}-battery.log`); `tracecheck` byte-equal to the quoted lines; digest re-derived by zeroing the pin → `4bd01143702d17bc05f187a14312cb868b85276b0f0b6d015c351929eae9bfc0` = pinned (`logs/digest-rederivation.log`); package suite 80 PASS / 0 FAIL (`logs/pkg-cloneproject-v.log`). |

## Findings

### P1 — the strict gate admits, a second decoder guesses, and `exact` is stamped

**P1-b `parseNativeLine` re-reads the envelope through `encoding/json`
struct decoding after the strict map admitted it; case-folded member
aliases change the projected kind, origin, authority, content, evidence
identity and attribution.** `environ.DecodeStrictObject` compares member
names exactly (`Origin` and `origin` are distinct keys, so no duplicate
fault), but the struct decode at `internal/cloneproject/native.go:108-111`
matches field names case-insensitively and takes the LAST match in document
order. The comment above it (`:104-107`, "the struct read cannot diverge
from the admitted members") and the `strictMembers` comment (`:66-70`,
"later reads of the admitted bytes are exact because the rewriting and
guessing shapes never reach them") are false. Pristine candidate, real
`clonesnap.Capture` → `Normalize` path, every row ADMITTED with
`capture_status=exact` (`logs/probe-case-alias.log`):

| Probe | Record (one extra envelope member) | Projected |
|---|---|---|
| PA1 | `"native_type":"frobnicate","NATIVE_TYPE":"message/user"` | `user_message`, `public`, evidence `native_type="message/user"` — an unknown record coerced into a known kind (AC 1: "never guessed, coerced into a neighbouring kind") |
| PA2 | same alias placed BEFORE `native_type` | `opaque_event` — the outcome depends on member order, i.e. on a member the projection does not claim |
| PA3 | `instruction/snapshot`, `"origin":"foreign","Origin":"native"`, body `authority:high` | `Instruction={Found:true Authority:high Directives:[DROP-EVERYTHING]}`, payload `authority:high` — the AC-named gate ("an instruction carried in a foreign record cannot change the projected session's effective instruction snapshot or authority") defeated |
| PA4 | `"body":{"text":"declared"},"Body":{"text":"smuggled"}` | content block `smuggled` — sealed content from a member the record does not claim |
| PA5 | `"native_event_id":"evt-a5","Native_Event_Id":"evt-forged"` | evidence `native_event_id="evt-forged"` — the evidence identity rewritten |
| PA6 | `reasoning/summary`, `"origin":"foreign","protection":"encrypted","Protection":"none"`, plaintext body | `reasoning_summary`, `internal`, content projected, no reason code — a record whose `protection` member says `encrypted` is promoted to `reasoning_summary` (AC 2: "never ... promoted to reasoning_summary") |
| PA7 | `"actor":"subagent:ghost","Actor":"main"` | attributed to `main` (pristine without the alias refuses `subagent:ghost` as unmapped) |
| PA8 | control: `"extra":true` | admitted as an unclaimed extra — the stated "envelope extras unclaimed" bound covers this, not PA1–PA7 |

Why P1, not P2: (a) it is the same class the Story has been rejected for
three times (decode-and-continue that diverges from the gate — numbers,
extension values, text) and that the cr2 brief told this leaf to close "at
the entry, not the two witnessed vectors"; the rework closed the two
vectors and left the second decoder in place, now documented as unable to
diverge. (b) It defeats gates the AC names by sentence (unknown → opaque;
foreign instruction low-authority; foreign encrypted reasoning never
promoted) and rewrites the evidence identity and content while stamping
`exact`, which is exactly the P1-a shape. (c) Shipped prose asserts the
property in five places: `native.go:66-70,104-107`,
`internal/cloneproject/TRACEABILITY.md:34` ("before any member is read"),
`:65` ("never dropped, guessed, or re-typed"), `:96` ("never rewritten"),
README:2458 ("low-authority history that never changes the effective
instruction snapshot"), LOGBOOK REV2 line ("the later struct read is
exact because the rewriting/guessing shapes never pass the gate"), results
finding table row 1 (same sentence). The stated bound "envelope extras
unclaimed" does not cover an extra that overrides a claimed member's value.

The fix is not a forced fit. In a throwaway copy
(`logs/fixprobe-strict-map-envelope.log`) I replaced the struct decode with
reads of the six claimed members from the strict map (`json.Unmarshal(
members["native_event_id"], &envelope.NativeEventID)` … and
`envelope.Body = members["body"]`; 12 lines, no second decoder). The
committed suite stays green (80/80), and under the fix PA1 → `opaque_event
frobnicate`, PA3 → `Found:false`, payload `authority:low`, PA4 → content
`declared`, PA5 → `evt-a5`, PA6 → refused (`body carries unknown member
"text"`), PA7 → refused (`subagent:ghost` unmapped), PA8 still admitted.
Ship: the strict-map read (or `environ.CheckStringBounds(members[...],
1, 512)` for the bounded members, which also removes the `StringLength`
double-call), one committed row per alias axis that matters to the AC
(native_type → kind; origin → effective snapshot and authority; protection
→ promotion; body → content; native_event_id → evidence identity; actor →
attribution), and a narrowing that re-introduces the case-folded read for
exactly one member. Then correct the five prose sites.

### P2 — an emitted kind with no committed row

**P2-d `reasoning_summary` (unprotected `reasoning/summary`) has no positive
test; a kind/visibility swap survives the whole suite.**
`R5-reasoning-summary-routed-public` (`dispatch.go:180-182`: `default:`
arm routed to `user_message`/`public`) SURVIVED 2/2 against `-run
TestNormalize` (`mutants/reviewer-run{1,2}/R5-*.log`). `grep
'reasoning/summary\|reasoning_summary' *_test.go` finds only the
promotion-negative assertion and the protected-plaintext refusal row: no
test drives an unprotected `reasoning/summary` record. TRACEABILITY's
fact-registry table (`:47`) and the results' "registries cover exactly the
ten emitted kinds" therefore describe a row nothing executes, and a
reasoning summary sealed with `public` visibility would pass review. One
positive row (kind `reasoning_summary`, visibility `internal`, content
block, `exact`) and the swap as its narrowing.

### P3 — unpinned members and labels (production correct in each case; each plant SURVIVED 2/2 against the whole committed suite, `mutants/reviewer-run{1,2}/`)

- **P3-m capture status is pinned on the opaque row only.**
  `R4-capture-status-partial-messages` (`dispatch.go:389`: `partial` for
  exactly `message/user`) SURVIVED 2/2; only
  `TestNormalizeUnknownRecordBecomesOpaqueEvent` asserts `exact`. Assert the
  status on at least one row per emitted kind, or on every event in
  `TestNormalizeSealsSessionShape`.
- **P3-n fold order is a doc claim.** `R6-first-native-instruction-wins`
  (`tools.go:171`) SURVIVED 2/2; no test carries two native instructions.
  Either pin last-wins with two native records or drop the sentence.
- **P3-o external actor kind unpinned.** `R7-external-actor-kind-subagent`
  (`dispatch.go:417`) SURVIVED 2/2; `TestNormalizeActorOrderIsDeterministic`
  references `external` but asserts nothing about its sealed kind.
- **P3-p required-member gate pinned by message, on a known type only.**
  `R1-required-member-body-dropped` (`native.go:98`, `body` removed from the
  list) KILLED 2/2 — but only because `missing_envelope_member` (a
  `message/user` fixture) then refuses with a different fragment at the body
  decode. Under the plant an unknown type without a body is ADMITTED as
  `opaque_event` (`logs/under-mutant-R1.log`). Add the unknown-type row.
- **P3-q `"text":null` seals as empty inline content, `exact`** (PB1,
  `logs/probe-value-shapes.log`): `json.Unmarshal(null, &string)` leaves the
  string empty. `directives:null` was stated as a bound; `text:null` was not.
  State it or refuse it (`ciphertext:null` is harmless: never read).
- **P3-r `N-pending-aborted` is an add-arm plant on an unreachable arm.**
  `Normalize` passes `nil` live work orders (`normalize.go:176`), so the
  `continue` at `tools.go:136` cannot be measured from the entry; the plant
  appends `call-7` explicitly. Matrix rows 13/17 call it a "killer"
  narrowing. Label it (the matrix D bound already says nothing captured can
  pend). Same for the results' "No plant covers two sites": the five
  `N-strict-*` rows patch the shared `strictMembers` body used by both frame
  sites (one gate, two call sites); say so.
- **P3-s `native_type` width edge** (`R2-native-type-width-513`, `native.go:118`)
  SURVIVED 2/2 — inside the stated per-edge bound; listed for completeness
  because the bound's text names directives, call IDs and tool names but not
  `native_type`.
- **P3-t LOGBOOK layering.** The leaf's block still opens with rev1 figures
  (25 tests, 49 mutants, pin `69d156d9…`, cases=144) and only the REV2 lines
  carry the shipped state (80 PASS lines, 65 rows, pin `4bd01143…`, 145);
  acceptable as history, but the REV2 line repeats the P1-b sentence and
  must be corrected with it.

## What is clean (verified, not read)

1. **Invariant 1** (unknown → opaque, almost-known, unknown member, raw refs
   resolve, stable twice): rows 1–5 reproduce; `N-unknown-kind`,
   `N-unknown-kind-wholly`, `N-unknown-member` KILLED 2/2; PB4 (empty
   unknown-class member) seals a zero-length whole-member `opaque_event`;
   PB8 (unknown type with a non-object body) stays opaque, body never read.
2. **Invariant 2** at the claimed members: rows 6–10 reproduce; the
   forbidden-member sweeps hold; `N-protection-encrypted/-signed`,
   `N-foreign-instruction`, `N-foreign-payload-authority`, `N-usage-target`
   KILLED 2/2; `R3-reasons-unsorted` KILLED 2/2 (order pinned).
3. **Invariant 3**: the five-row matrix plus `TestNormalizeResultAfterBoundaryCompletes`,
   `TestNormalizeProtectedToolCallStaysOpaqueHistory`,
   `TestNormalizeLiveSurfaceStaysEmpty` reproduce; live surface empty in
   every row; `N-tool-incomplete`, `N-tool-orphan`, `N-callable`,
   `N-callable-false`, `N-live-followup`, `N-result-status-pending` KILLED 2/2.
4. **Invariant 4**: main-null parent, contiguous ordinals, predecessor
   parents, single head; 65536 inline / 65537 blob-referenced and byte-exact;
   `N-overflow-bound`, `N-overflow-truncate`, `N-main-parent`,
   `N-parents-chain` KILLED 2/2; PB3 version literals and PB9 (escaped-key
   duplicate) refuse; PB5 (CRLF) keeps the CR inside the byte range.
5. **Determinism / durability**: `TestNormalizeIsDeterministic`,
   `TestNormalizeActorOrderIsDeterministic`, idempotent shared-sink replay and
   no-partial-bundle reproduce; `N-actors-sort` KILLED 2/2.
6. **Story-close**: edited registry is the adopted `ownership.v0.7.0.json`;
   the diff is additive plus the two binding upgrades (trunk's mesh-RPC cases
   intact; the only deleted lines are the two superseded `unevidenced`
   bindings and figure rewrites); digest re-derived = pin; `tracecheck`,
   `--section 7.8`, `--section 10.2` green with the quoted lines byte-equal;
   `traceability` and `cmd/tracecheck` suites PASS including
   `TestREADMEMeasuredCoverageMatchesTracecheckReport` and
   `TestV070RegistryRederivesFromTrunkV060Registry`
   (`logs/pkg-traceability-v.log`); README fenced line byte-equal, prose
   figures consistent with the report; `13.14.1` deliberately unbound with
   the reason stated; 13.14.2–13.14.5 non-ownership stated in matrix D.
7. **Hygiene**: CR paths ⊆ {LOGBOOK, README, clonebundle, cloneproject,
   clonesnap, traceability}; `task-board.config.json` byte-identical to trunk
   (30 commands); the 8 trunk-moved paths that intersect the CR carry only
   additive/figure edits (no reverts); LOGBOOK purely additive and
   newest-first; zero `__pycache__`/`.pyc`; gofmt 0 files, `go build`,
   `go vet`, linux and windows builds, `cataloggen -adopted -check` exit 0.
8. **Suite**: `go test ./... -count=1` 40/41 ok in the copy, the 41st
   (`specpin`) failing only because the copy lacked the tracked
   `.task-board`; re-run with it materialized → ok (`logs/cmd04-*.log`).
   `-race` on cloneproject, clonebundle, clonesnap, traceability/... ok, no
   DATA RACE (`logs/cmd05-race-story-packages.log`); cloneproject coverage
   90.1%. The other 37 packages' race lanes, the fuzz gates and
   `task-board validate` are accepted from the CR construction log
   (`required=30 green=30 failed=0`) and the producer's `cmds/cmd05a-g`
   (41 ok, 0 DATA RACE) — not re-run by me.

## Coverage ratio I measured

38 of 38 matrix rows are driven through `Normalize` by a named committed
test over `clonesnap.Capture` output (all 80 PASS lines re-run). Measured
clean at the class level: 34 of 38 — rows 5 (never guessed / re-typed), 8
(never promoted), 9 (foreign instruction low-authority) and 16 (attribution)
hold for the witnessed vectors and fail on the case-alias axis (P1-b).
Outside the matrix, one of the ten emitted kinds (`reasoning_summary`) has
no positive row (P2-d). Reviewer plants: 8 rows against the whole suite,
2/2 each — R1 KILLED (by message), R3 KILLED, R2/R4/R5/R6/R7 SURVIVED,
`RC-control-comment` SURVIVED (`mutants/reviewer-run{1,2}-battery.log`).
Producer battery: 64 KILLED + 1 control SURVIVED, 2/2.

## Rework scope (for the producer)

1. P1-b: read the six claimed envelope members from the strict map (no
   second decoder; the 12-line shape in `logs/fixprobe-strict-map-envelope.log`
   is proven green), add one committed row per AC-relevant alias axis
   (kind, origin/authority, protection/promotion, body/content, evidence
   identity, actor) plus a narrowing that restores the case-folded read for
   exactly one member, and correct the five prose sites (`native.go`,
   TRACEABILITY :34/:65/:96, README, LOGBOOK REV2, results).
2. P2-d: one positive `reasoning_summary` row (kind, `internal`, content,
   `exact`) with the routing swap as its narrowing.
3. P3-m..t: capture-status assertions per kind; pin or drop last-wins; pin
   the external actor kind; unknown-type-without-body row; state or refuse
   `text:null`; relabel `N-pending-aborted` and the shared-site `N-strict-*`
   rows; fix the LOGBOOK REV2 sentence.
4. Re-run the battery twice on the final tree, re-run tracecheck, report the
   measured ratio; keep the candidate UNCOMMITTED.
