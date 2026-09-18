# TASK-260830-32ypr2 — Review verdict, Change Request revision 1

Reviewer run: RUN-260918-320878 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-32ypr2-1` revision 1 (story_final), base
`c3aae73df906df1b5aa962c7bc3c16437dd920fa` (= `origin/main`), candidate tree
`471e8282d6a8a31a7532ee7f0b5a9294b5dc15ca` (65 paths; patch sha256
`566deda598876d35fb4ea5aa699966723a6f2fee3c1efb1141615c51f03ccba4`, verified
byte-equal to `git diff c3aae73 471e828`). The live Story worktree's
temp-index tree OID (`git add -A` into a scratch index copy) equalled
`471e828…` before this review and `git status` stayed ` M LOGBOOK.md`,
` M README.md`, ` M internal/traceability/{6 files}`, `?? internal/cloneproject/`
throughout; worktree, index, branch and HEAD were never touched. Authority:
`internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (10331–10546), §10.2 (4851–4903),
§7.8 (3738–3924), §1.6 common data model (line 324: "Decoders MUST reject
lone surrogate code points before canonicalization").

Method: isolated immutable copies of the exact candidate tree (`git archive
471e828…` → `.temp/TASK-260830-32ypr2/review-rev1/{candidate,pristine}` plus
`trunk` from `c3aae73`, the tracked `.task-board/.resources` and `.activity`
dropped from all copies); `diff -rq` clean against `pristine` after every
probe and battery; `PYTHONDONTWRITEBYTECODE=1` on every harness, zero
`__pycache__`/`.pyc` after the runs. Every probe in this verdict was executed
on the candidate copy; the logs are in
`TASK-260830-32ypr2_review-evidence-rev1.tar.gz`.

## Verdict: CHANGES REQUESTED → `to-dev`

One P1, three P2, and a P3 list. The technical body is mostly right and the
producer's evidence reproduces: the 25-test package suite, the 49-row battery
(49 KILLED + control SURVIVED, 2/2 in my runs), tracecheck at the quoted
ratios, the re-derived registry digest, and the whole 41-package suite are
all green on the exact candidate tree. What fails is (a) a live admit hole at
this leaf's own entry — the fixture-native decoder forks the landed strict
JSON gate and REWRITES source text while stamping `exact`, which is the P3-ζ
class the predecessor's verdict deferred to this leaf by name; (b) the seven
inherited-advisory groups are not mentioned anywhere in the results, matrix,
TRACEABILITY or LOGBOOK (silently dropped, including production prose on the
story_final patch that now states the final leaf did something it did not);
(c) a §7.8 registry binding that discharges the 8 MiB adapter-frame clause
with a test that constructs no frame, published in the README as
"Section 7.8 at 2/2, bound to internal/cloneproject".

## Section 0 — Inherited advisories from TASK-260830-2g5be6 rev3 (the accepted predecessor verdict)

The predecessor verdict carried seven advisory groups forward "for the
Story's FINAL leaf TASK-260830-32ypr2 or a follow-up on internal/clonesnap"
and asked for a recorded decision on each. The leaf's own delta
(`git diff --stat 3005d78 471e828`) touches no file under
`internal/clonesnap` or `internal/clonebundle`, and
`grep -i -E "P3|advis|AfterWalk|chmod|BuildAdmissionReceipt|empty-cwd|lone.surrogate|EncodeNativeIdentity|swaps-pre-post|quiet"`
over `TASK-260830-32ypr2_results.md` and `_conformance-matrix.md` returns
nothing.

| # | Advisory (2g5be6 rev3) | Grade on this CR |
|---|---|---|
| 1 | P3-a independent oracle for archive digests (`R3-archive-swaps-pre-post`) | SILENTLY DROPPED |
| 2 | P3-b quiet-store row under `RaceArchive`+`OperatorExplicit` | SILENTLY DROPPED |
| 3 | P3-c/d chmod-000 rows or restate the `walk.go` "never an absence" comment as a bound | SILENTLY DROPPED |
| 4 | P3-e state the AfterWalk-seam bound in TRACEABILITY and results | SILENTLY DROPPED |
| 5 | P3-f direct `BuildAdmissionReceipt(…, "ultra")` row or state the bound | SILENTLY DROPPED |
| 6 | P3-g empty-cwd row; early-pass "nothing installed" row or reword | SILENTLY DROPPED |
| 7a | P3-ζ lone surrogate at the text admission site — the tree itself says "Deferred to the Final leaf, which owns projection fidelity over session/event text" (`internal/clonesnap/TRACEABILITY.md:145-148`) | NOT CLOSED and LIVE at this leaf's entry: see P1-a |
| 7b | P3-η remainder (`EncodeNativeIdentity` standalone contract) | SILENTLY DROPPED |
| 7c | P3-ζ′ fidelity vocabulary spelled twice — the tree says "The FINAL leaf unifies or exports one owner" (`internal/clonesnap/project.go:41-43`, `TRACEABILITY.md:132-135`) | SILENTLY DROPPED; the production comment is now false prose on the story_final patch |

An explicit deferral with an owner (a follow-up on `internal/clonesnap`) is
an acceptable decision for 1–6 and 7b; 7a must be closed here (P1-a); 7c needs
either the unification or a rewrite of the two sentences that promise it.

## Findings

### P1 — production admits and rewrites what it must refuse, at this leaf's entry, and stamps `exact`

**P1-a `parseNativeLine`/`decodeBodyObject` fork the landed strict decoder;
lone-surrogate escapes are rewritten to U+FFFD and duplicate members are
guessed last-wins, both sealed with `capture_status: exact`.**
`internal/cloneproject/native.go:73` and `:233` decode with plain
`encoding/json` instead of the landed `environ.DecodeStrictObject`
(`internal/environ/decode.go:54`, the single JSON entry the epic already
converged sessadapter and dirnode onto; it refuses non-UTF-8, lone surrogate
escapes, duplicate members and trailing data). Pristine candidate, real
`clonesnap.Capture` → `Normalize` path (`logs/probes-pristine.log`):

- PA: `{"text":"\ud800"}` → ADMITTED `user_message`, sealed content bytes
  `ef bf bd` (U+FFFD), `capture_status=exact`; same for `\udc00`,
  `A\ud800B` (→ `A�B`), `\ud800\ud800`. The raw_ref resolves to the source
  line carrying `\ud800`; the canonical event carries different text.
- PB: `"native_event_id":"evt-\ud800"` → evidence `native_event_id="evt-�"`
  (the evidence identity itself is rewritten); `"native_type":"frob\ud800"`
  → `opaque_event` with `native_type="frob�"`.
- PC: `{"text":"A","text":"B"}` → content `"B"`, `exact`;
  `"native_type":"frobnicate","native_type":"message/user"` → `user_message`
  (an ambiguous record coerced into a known kind);
  `"protection":"encrypted","protection":"none"` on `reasoning/summary` →
  `reasoning_summary` with the body text projected; `"origin":"foreign",
  "origin":"native"` on an instruction → reaches the effective snapshot with
  `authority=high`.

Why P1: §1.6 line 324 is a MUST for decoders feeding canonicalization; the
AC says opaque records are "never dropped, guessed"; the leaf's own
TRACEABILITY row (`internal/cloneproject/TRACEABILITY.md:91`, "1.6 text is
valid UTF-8, never rewritten") is false; and this is the exact class the
predecessor deferred to this leaf (Section 0, 7a). It is also rejection
shape (c) of the producer brief (a fork of a landed gate that diverges).
The fix is not a forced fit — I applied it in a throwaway copy
(`logs/fixprobe-strict-decode.log`): replacing both decodes with
`environ.DecodeStrictObject` turns every PA/PB/PC admit into a refusal
naming the fault (`lone surrogate escape`, `duplicate member in member
text`), and the committed suite stays green except for the
`malformed_json_line` fragment (keep "is not a JSON object" in the message).
Ship: the delegation, one lone-surrogate row (text member and envelope
member), one duplicate-member row per envelope/body site, and narrowing
mutants that admit exactly one escape / exactly one duplicated member.

### P2 — evidence, traceability and a second fork

**P2-a Inherited advisories silently dropped (Section 0).** Nothing in the
results, matrix, LOGBOOK or either TRACEABILITY records a decision for
groups 1–6, 7b, 7c; 7a is live (P1-a). The story_final patch ships two
production/traceability sentences promising work this leaf did not do
(`internal/clonesnap/project.go:41-43`, `internal/clonesnap/TRACEABILITY.md:132-147`).
Record a decision per group in the results (closed → the test; deferred →
owner + reason), and either unify the fidelity vocabulary or rewrite the two
sentences as a bound with the follow-up owner named.

**P2-b `section:7.8` bound `full 2/2` to `internal/cloneproject` with a
clause no test drives at the named site.** Registry
`internal/traceability/ownership.v0.7.0.json:4735-4764`: production owner
changed to `internal/cloneproject/normalize.go Normalize`; clause `7.8#1`
("Large data is referenced by manifest or Blob Descriptor ID and MUST NOT be
embedded in the 8 MiB frame", spec 3785) is discharged by
`clone-projection-fidelity` → `TestNormalizeInlineContentBoundary`, a test
that constructs no adapter frame and never touches the 8 MiB bound; the
64 KiB inline rule it does prove is §13.14.1's. The leaf's own matrix (D)
and TRACEABILITY state "§7.8 operation frames … NOT APPLICABLE here: wiring
stays `sessadapter`", and README:3258-3260 now publishes "Section 7.8 at
2/2, bound to internal/cloneproject". The honest binding exists on trunk:
`internal/sessadapter/protocol.go:34` (`MaxFrameBytes = 8 << 20`) with
`TestDecodeRequestFrameRefusesOversizeFrame` and `TestFrameBoundEdges`
(`internal/sessadapter/envelope_test.go:127,144`). Bind `7.8#1` to a
sessadapter acceptance case over those tests with the production owner in
sessadapter (or leave 7.8 `partial 1/2` with a gap), keep `7.8#2` on
`session-adapter-call-binding` (that one is real: `CheckCallBinding`,
`discovery_test.go:103,165`), re-derive the digest, update the pin tests and
the README prose. `--section 7.8` admission must not be claimed by a
projection test.

**P2-c `bodyUint53` forks `environ.CheckUint53Bounds` and admits
string-typed numbers.** `internal/cloneproject/native.go:278-296` decodes
into `json.Number` and checks digits; a JSON string `"5"` is accepted by
`encoding/json` into `json.Number`, so `{"input_tokens":"5","output_tokens":0}`
→ ADMITTED, ledger `SourceInputTokens:5` (PD, `logs/probes-pristine.log`).
The doc comment says "never a fraction, exponent, or string" — false. The
landed `rawUint53` refuses a string (`value.(json.Number)` on a string
fails). Same class: `"v":"1"` is admitted as envelope version 1. Delegate to
`environ.CheckUint53Bounds(raw, 0, maxUint53)` and add the string row plus a
narrowing.

### P3 — sub-members, axes and gates with no committed row (production correct in each case; each plant SURVIVED the whole committed suite 2/2 with its admitted behaviour executed under the plant, `logs/under-mutant/*.log`)

- **P3-a Determinism is measured along one axis.** `R1-actors-unsorted`
  (drop `sort.Strings(selectors)`, `dispatch.go:414`) SURVIVED 2/2; under it
  a session with three extra actors sealed 3 distinct byte strings over 12
  projections (`under-mutant/R1-actors-unsorted.log`); pristine 1. Matrix
  row 22 says "a proven property, not a gate … no plant required" — the sort
  IS the gate. One determinism row with ≥2 extra actors.
- **P3-b type/protection agreement arm unrowed.**
  `R2-reasoning-encrypted-admits-signed` (`dispatch.go:106-109`) SURVIVED
  2/2: `reasoning/encrypted` declared `protection:signed` projects
  `opaque_reasoning` with `foreign_signature_unverifiable` under the plant;
  pristine refuses. One refusal row per arm.
- **P3-c width edges unwitnessed.** `R3-directive-width-4097`
  (`native.go` directive `> 4096`) and `R3b-call-id-width-513` (call_id
  `> 512`) SURVIVED 2/2; the `N-string-bound` row is called "representative
  for the shared character-width rule" but pins only `native_event_id`.
  Same shape for `tool_name` (definition and call). Either witness each edge
  or route all of them through one shared helper and pin that.
- **P3-d trailing-bytes gate unrowed.** `R4-trailing-bytes-admitted`
  (`native.go:79`) SURVIVED 2/2; under the plant `{…}x` projects
  `user_message` with a range covering the `x`.
- **P3-e protected non-reasoning tool records reach opaque only via the
  fold-skip, which has no row.** `R5-protected-tools-folded`
  (`normalize.go:351`) SURVIVED 2/2; under the plant a protected `tool/call`
  REFUSES ("unknown member ciphertext") instead of projecting `opaque_event`
  (pristine: `opaque_event`, the unprotected result becomes an aborted
  orphan). `TestNormalizeProtectedMessageBecomesOpaqueEvent` pins the
  message shape only.
- **P3-f result status vocabulary unrowed.** `R6-result-status-pending`
  (`native.go:476`) SURVIVED 2/2; under it `status:"pending"` is admitted
  and the call resolves `completed` with `ResultStatus:pending`.
- **P3-g size gate shadowed by a shared fragment.** `R7-size-check-skipped`
  (`normalize.go:300`) SURVIVED 2/2: `tampered_blob` flips a byte at equal
  size and asserts the fragment "disagrees", which both the size arm and the
  digest arm emit; a one-extra-byte blob reaches the size arm on pristine
  (`blob size 133 disagrees with raw count 132`) and the digest arm under the
  plant. Assert the arm's own message and add the size-drift row.
- **P3-h unknown+protected body refusal is pinned in neither direction.**
  `R8-body-decode-unknown-protected-skipped` (`dispatch.go:102`) SURVIVED
  2/2: pristine REFUSES `frobnicate`/`encrypted`/`{"mystery":true}` ("body
  carries unknown member") — i.e. a wholly unknown record is refused because
  its body is not `{ciphertext}` — while under the plant it projects
  `opaque_event` with both reasons. That contradicts the doc's "type-level
  unknowns become opaque_event". Decide (I would not decode the body of an
  unknown type at all) and pin it.
- **P3-i non-JSONL included members refuse the whole projection.** PI:
  a 3-byte binary `durable_index_required` member and a text
  `derived_cache_optional` member both refuse `Normalize` ("is not a JSON
  object"). Every included non-`unknown` member is parsed as the JSONL log;
  the doc's bound names only "included durable capture members". State the
  bound explicitly (or parse `durable_payload` only and route the rest).
- **P3-j `section:10.2` binding dropped eight landed acceptance-case
  attributions** (`scalar-digest-validation`, …,
  `localstore-immutable-blob-install`, `localstore-sqlite-projection`) when
  it was rewritten; `localstore-immutable-blob-install` is the actual
  fsync/verify/atomic-install owner of `10.2#2`. Union, do not replace.
- **P3-k harness labels.** `N-empty-projection` (`< 0`) and
  `N-overflow-truncate` (`if false {` around the install arm plus a
  truncation) are not narrowings; the results say "No row is an arm-delete".
  Label them (single-member class; behaviour swap) — both still measure what
  the AC asks.
- **P3-l `"directives":null` is admitted as zero directives** (PF); nit.

## What is clean (verified, not read)

1. **Invariant 1 (unknown → opaque):** rows 1–5 reproduce; `N-unknown-kind`,
   `N-unknown-kind-wholly`, `N-unknown-member` KILLED 2/2 in my runs; raw
   refs resolve through the real store layout; twice-projected bytes equal.
2. **Invariant 2 (foreign reasoning opaque, instructions low, usage
   source-only):** rows 6–10 reproduce, the forbidden-member sweeps hold,
   `N-protection-encrypted`/`-signed`, `N-foreign-instruction`,
   `N-foreign-payload-authority`, `N-usage-target` KILLED 2/2; a foreign
   unprotected `reasoning/summary` projects `reasoning_summary`/`internal`
   (allowed: only encrypted/signed must be opaque) and a protected known
   `tool/call` projects `opaque_event` (PH).
3. **Invariant 3 (tools inert):** the five-row matrix reproduces; live
   surface empty in every row; `N-tool-incomplete`, `N-tool-orphan`,
   `N-pending-aborted`, `N-pending-completed`, `N-callable`,
   `N-callable-false`, `N-live-followup` KILLED 2/2.
4. **Invariant 4 (shapes):** main-null parent, contiguous ordinals, parent
   chain, single head; 65536 inline / 65537 blob-referenced and byte-exact;
   multibyte bound measured in bytes on both sides (`clonebundle/event.go:736`
   uses `len(text)` too; PG: 65538 bytes/21846 chars → blob, 65536
   bytes/32768 chars → inline); `N-overflow-bound`, `N-overflow-truncate`,
   `N-main-parent`, `N-parents-chain` KILLED 2/2.
5. **Determinism/idempotency:** `TestNormalizeIsDeterministic` and the
   shared-sink replay reproduce (blob count stable).
6. **Story-close:** the edited registry is the adopted
   `ownership.v0.7.0.json`; `git diff c3aae73 471e828` on it is additive plus
   the two binding upgrades (trunk's 5 mesh-RPC cases and 3 upgraded bindings
   intact); digest re-derived by zeroing the pin in a copy →
   `69d156d99a3b13ae598c7de3152225eafe4576c8f33286e8e36f68f22235052a`
   (`logs/digest-rederivation.log`), equal to the pin; `tracecheck` and
   `--section 7.8`/`10.2` green with the quoted lines byte-equal
   (`logs/tracecheck.log`, `section-*.log`); README fenced line byte-equal;
   `TestREADMEMeasuredCoverageMatchesTracecheckReport` executed and PASS
   (`logs/pkg-traceability-v.log`); LOGBOOK entry newest-first and purely
   additive (`git diff c3aae73 471e828 -- LOGBOOK.md` has no deletions);
   README deletions are only the rewritten coverage figures.
7. **Hygiene:** CR paths ⊆ {LOGBOOK, README, clonebundle, cloneproject,
   clonesnap, traceability}; `task-board.config.json` byte-identical to trunk
   (30 commands); the 8 trunk-moved paths that intersect the CR carry only
   additive/expected edits; gofmt 0 files, `go build`, `go vet`, linux and
   windows builds, `cataloggen -adopted -check` all exit 0 on the copy.
8. **Suite:** `go test ./... -count=1` 41/41 ok (2m31s,
   `logs/cmd04-go-test-all.log`); `-race` on cloneproject, clonebundle,
   clonesnap, traceability/... ok with no DATA RACE
   (`logs/cmd05-race-story-packages.log`); cloneproject coverage 88.9%. The
   remaining 37 packages' race lanes and fuzz gates are accepted from the CR
   construction log (`required=30 green=30 failed=0`) and the producer's
   `cmds/cmd05a-g` (41 ok, 0 DATA RACE) — not re-run by me.

## Coverage ratio I measured

24 of 24 producer matrix rows are driven through `Normalize` by a named
committed test over `clonesnap.Capture` output (I re-ran all 62 PASS lines,
`logs/pkg-cloneproject-v.log`). Of those, 21 are measured clean at the
member level; row 5 ("never dropped, guessed") is contradicted on the
duplicate-member axis (P1-a), row 22 (determinism) is measured on one axis
only (P3-a), and rows 18/19/21 rest on stated clonebundle bounds for shape
internals (acceptable, stated). Nine reviewer narrowings on unrowed gates
SURVIVED 2/2 (P3-a..h) plus an applied control (`RC-control-comment`,
SURVIVED); the producer's 49 rows KILLED 2/2 in my runs with the
`C-doc-comment` control SURVIVED (`mutants/producer-run{1,2}-battery.log`,
per-plant logs under `mutants/producer-run{1,2}/`).

## Rework scope (for the producer)

1. P1-a: delegate the envelope and body decodes to
   `environ.DecodeStrictObject` (proven clean in `logs/fixprobe-strict-decode.log`);
   add lone-surrogate rows (text member, envelope member) and duplicate-member
   rows (envelope, body) with one narrowing each; fix TRACEABILITY row
   "never rewritten" to name the gate that now enforces it.
2. P2-a: a decision line per inherited advisory group (1–6, 7b, 7c) in the
   results; rewrite `clonesnap/project.go:41-43` and
   `clonesnap/TRACEABILITY.md:132-147` so they state what the tree does.
3. P2-b: re-bind `7.8#1` to the sessadapter frame-bound tests with a
   sessadapter production owner (or `partial 1/2` with a gap); re-derive the
   digest, update `traceability_test.go`/`main_test.go`/README pins and prose.
4. P2-c: `bodyUint53` → `environ.CheckUint53Bounds`; string row + narrowing;
   fix the doc comment.
5. P3: one row each for a–h (a, b, e, f, g are cheap and close real gaps);
   state the P3-i bound; union the 10.2 acceptance cases (P3-j); relabel the
   two non-narrowing rows (P3-k).
6. Re-run the battery twice on the final tree, re-run tracecheck, and report
   the measured ratio; keep the candidate UNCOMMITTED.
