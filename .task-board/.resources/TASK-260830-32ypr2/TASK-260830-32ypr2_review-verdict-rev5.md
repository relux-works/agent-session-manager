# TASK-260830-32ypr2 — Review verdict, Change Request revision 5

Reviewer run: RUN-260918-b4e4d8 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-32ypr2-5` revision 5 (story_final), base
`42d5a959aa523572d2745d44f828d41a0b2d72b3` (= `origin/main`, re-fetched at
review start, unmoved), candidate tree
`c9ee6688973eac5251371594a154cc7f8a354608` (66 paths; patch sha256
`55e3348a0b8dbb89e9b27696d4272d3e91520ef6f87b38f530dc8734e3342f8d`, equal to
the attached resource and to `git diff base candidate` computed by me). The
live Story worktree's temp-index tree OID equalled `c9ee6688…` before and after
this review; `git status` stayed the same 14 lines (13 ` M`, `?? internal/cloneproject/`);
HEAD `5cc9449`, branch `task-board/story/STORY-260830-1cyj0q`; worktree, index,
branch and HEAD were never touched. Authority: `internal/specdoc/SPEC.v0.7.0.md`
§13.14.1 with §7.8 and §10.2.

Method: isolated immutable copies of the exact candidate tree (`git archive
c9ee6688…` → `.temp/TASK-260830-32ypr2/rev5-review/{candidate,battery,battery-A..D,plant}`;
`.task-board` kept only in `candidate` for the board-scoped test); `diff -rq`
clean between `candidate` and every battery/plant copy after every harness and
plant run; `PYTHONDONTWRITEBYTECODE=1` on every harness; zero `__pycache__`/`.pyc`
in the CR tree (`git ls-tree`), the live worktree, or the copies (the two
`__pycache__` dirs that appeared under `candidate/` and `battery/` were my own
`importlib` listings of the harness rows, deleted; they never existed in the CR
tree). Every probe, plant and command below was executed by me on this tree;
logs with subprocess exits are in `TASK-260830-32ypr2_review-evidence-rev5.tar.gz`.

## Verdict: ACCEPTED → `accept_cr(TASK-260830-32ypr2, revision=5)`

No P1, no P2. The rev4 rework (P2-f and P3-bb..hh) is closed at the class:
all eight AC-named scale-bound widenings now die BEHAVIOURALLY against the
whole `internal/clonebundle` package, every P3 re-plant dies, the registry
merge and digest are unchanged and re-verified, the suite and all three
batteries reproduce. What remains is three P3 advisories (two survivors of my
own plants on gates outside this leaf's AC-named set, one stale line number),
recorded below for a follow-up; none of them is a property this leaf's AC or
matrix claims as measured.

## Section 0 — rev4 rework verification (rev5 delta: 10 files, +372/−9, test-plus-matrix; the only production edit is the CRLF doc comment on `splitRecordLines`)

### P2-f — the eight far-edge widenings, re-run by me on the rev5 tree, whole `internal/clonebundle` package as the killer (`-run .`), twice (`reviewer-plants/run{1,2}/RV-*.log`)

| Plant (bound widened by one) | Site | rev4 | rev5 | Kill mechanism observed |
|---|---|---|---|---|
| `RV-parents-65-build` | `event.go:426` | SURVIVED | KILLED 2/2 | `build_envelope/too_many_parents`: `error = <nil>, want ErrInvalid` — 65 sorted-unique parents ADMITTED |
| `RV-parents-65-decode` | `event.go:519` | SURVIVED | KILLED 2/2 | `decode/too_many_parents`: 65 parents cross the gate, refused downstream by the omit-self digest; the row's asserted message vanishes |
| `RV-heads-1025-build` | `session.go:335` | census only | KILLED 2/2 | `build_envelope/too_many_heads`: `error = <nil>` — 1025 heads ADMITTED (the low-edge `empty_heads` row also fails by its `[1..1024]` literal, as before) |
| `RV-heads-1025-decode` | `session.go:446` | SURVIVED | KILLED 2/2 | `decode/too_many_heads`: 1025 valid heads cross the gate, refused at the omit-self digest |
| `RV-eventids-1000001-build` | `session.go:331` | census only | KILLED 2/2 | `build_envelope/too_many_events`: `error = <nil>` — 1,000,001 distinct digests ADMITTED (3.9 s) |
| `RV-eventids-1000001-decode` | `session.go:442` | SURVIVED | KILLED 2/2 | `decode/too_many_events`: the count gate is crossed, refused at `event_ids[0] is not a digest` (0.2 s) |
| `RV-actors-1025-build` | `session.go:308` | SURVIVED | KILLED 2/2 | `build_actors/too_many_actors`: `error = <nil>` — 1025 valid actors ADMITTED |
| `RV-actors-1025-decode` | `session.go:493` | KILLED | KILLED 2/2 | `decode/too_many_actors`: count gate crossed, refused at `actor[0] not a JSON object` |

Every build-side kill is an admission (`error = <nil>`), every decode-side
kill is the far-edge input crossing the widened gate and being refused by a
different, later gate — the row's asserted refusal message disappears, so the
row distinguishes "gate refuses N+1" from "gate admits N+1". No kill in this
table comes from a low-edge message literal alone. The producer's nine new
harness rows (`N-{parents,heads,actors,eventids}-max-{build,decode}`,
`N-rawrefs-unsorted-decode`) are all APPLIED (no `patch anchors` error) and
KILLED 2/2 with their `-run` masks selecting exactly the named subtest
(`mutants/sibling-clonebundle-run{1,2}/`, subprocess exit 1 each). Each new
row asserts the code (`errors.Is(err, ErrInvalid)`) and the message. Matrix
section D and `internal/clonebundle/TRACEABILITY.md:56-58` cite the rows by
name; the "owned by the accepted suite" delegation is gone. Production edges
re-probed on this tree (`logs/probe-clonebundle-bounds-rev5.log`): 64/1024/1024/1,000,000
admitted, 65/1025/1025/1,000,001 refused; the million-event session seals in
3.3 s and decodes in 3.3 s.

### P3 re-plants (`reviewer-plants/run{1,2}/`, `-run TestNormalize` unless noted)

| rev4 finding | Plant | rev4 | rev5 | Closed by |
|---|---|---|---|---|
| P3-bb unterminated multi-line tail | `RV-unterminated-tail-dropped-multiline` | SURVIVED | KILLED 2/2 | `TestNormalizeUnterminatedTailResolvesWithOffset` (`len(DecodedEvents) = 1, want 2`; offsets `0`, `len(l1)+1` asserted); producer `N-unterminated-tail-drop` KILLED 2/2, honestly labelled behaviour drop |
| P3-cc per-type registries | `RV-allowed-call-live-followup` | SURVIVED | KILLED 2/2 | `TestNormalizeRefusalTable/call_live_followup_member` (admission); `N-allowed-call-live-followup`, `N-allowed-result-live-attestation` KILLED 2/2; the seven-site census is correct — `grep decodeBodyObject(` finds seven call sites in `native.go` (the rev4 verdict said six; seven is right) |
| P3-dd usage output arm | `RV-usage-output-arm-widened` | SURVIVED | KILLED 2/2 | `TestNormalizeRefusalTable/usage_ledger_overflow_output` (2^53−1 + 1 admitted under the plant); `N-usage-overflow-output` KILLED 2/2 |
| P3-ee decode-side unsorted `raw_refs` | `RV-rawrefs-unsorted-decode` (whole clonebundle) | SURVIVED | KILLED 2/2 | `TestCanonicalEventRefusals/decode/evidence_raw_refs_unsorted` (two distinct refs in descending canonical order; the `== 0` plant admits them into the self-digest refusal) |
| P3-hh empty `subagent:` | `RV-actor-empty-subagent-name` | SURVIVED | KILLED 2/2 | `TestNormalizeRefusalTable/empty_subagent_name`; honestly labelled message pin (`N-actor-empty-subagent-name` kills by the actor-mapping message) |
| P3-ff CRLF | — | — | CLOSED | `splitRecordLines` doc comment states the CR-inside-range behaviour; `TestNormalizeCRLFResolvesWithCarriageReturn` pins offsets `0`, `len(l1)+2`, lengths `len+1`, resolved bytes `line+"\r"`; matrix D + TRACEABILITY bound entries |
| P3-gg prose | — | — | CLOSED | matrix D `live_followup:null` row names `TestNormalizeNullOptionalBoolsDecodeAsFalse` (the three remaining `…DecodeAsEmpty` hits are the real `TestNormalizeNullDirectivesDecodeAsEmpty`); `"forty-seven"` dropped from `coverageNumberWords`, pin test still PASS |
| class control | `RV-allowed-message-escalate` | KILLED | KILLED 2/2 | unchanged |
| harness control | `RC-control-comment` | SURVIVED | SURVIVED 2/2 | the runner reports survivors |

### Story-close, unchanged since rev4 and re-verified

- Registry (`internal/traceability/ownership.v0.7.0.json`, adopted): byte-identical
  to the rev4 candidate (`git diff 6c670903 c9ee6688 -- internal/traceability/` is
  the one-line pin-test deletion only). Against trunk `42d5a95`: all 147 trunk
  acceptance cases present and unchanged + 5 Story cases appended; 72 of 74
  trunk ownership rows unchanged including the ptxkqe landing's `section:4.1`
  row and its upgraded `4.B`/`4.D`/`5.2`/`7.A` rows (equal to trunk, verified by
  key); the only two rows that differ are the superseded `unevidenced` stubs for
  `section:7.8` and `section:10.2`, replaced by the `full` bindings;
  `unowned_sections` and `source` unchanged (`logs/ownership-trunk.json`).
- Digest: zeroing `reviewedOwnershipCanonicalSHA256` in a throwaway copy makes
  the gate print `614eb55d70b4b83f1ff668b8bfe9830d0655189a3a76f65edefb5159c6710ecb`
  = the pinned value (`logs/digest-rederivation.log`, exit 1 as designed).
- `tracecheck` on the candidate (`logs/cmd24-tracecheck.log`, exit 0), verbatim:
  `traceability ok: contracts=64 normative_sections=36 acceptance_cases=152 fixtures=33 compatibility_contracts=55 assigned_scopes=0`
  `section coverage: bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574`
  `--section 7.8` and `--section 10.2` admitted (`assigned_scopes=1`, exit 0);
  `--section 13.14.1` refused `no scoped implementation owner` (exit 1, as the
  matrix states — the section carries no RFC 2119 line for the scanner).
- README pin: `TestREADMEMeasuredCoverageMatchesTracecheckReport` PASS on the
  candidate; planting `Two bindings are `full`` into the README copy reddens it
  (`logs/readme-pin-plant.log`: `matches "(Four) bindings are `full`" 0 times, want exactly 1`, exit 1).
  All five README figure sites agree with the report (`152`, `70/574`, Four,
  forty-three, twelve). README +125/−18 vs trunk (the 18 deletions are the
  coverage-figure rewrites and the now-false sessadapter "7.8 stays unevidenced"
  sentence); LOGBOOK +40/−0, the leaf block newest-first under `## 2026-09-18`
  with REV5 + REV5 EVIDENCE lines appended inside it; `task-board.config.json`
  byte-identical to trunk (30 commands, read from the candidate config).

## Independent review of the whole revision (spots I picked)

1. **Invariants 1–5 re-driven through `clonesnap.Capture` → `Normalize`** on the
   plant copy with the rev4 reviewer's 17-row Section-0 probe and 8 rev4 probes
   (all reproduce: PA1/PA2 → `opaque_event frobnicate`, PA3 → `Found:false`,
   PA6/PA7 refused, PR4-1..8 as in rev4, `logs/probe-rev5.log`) plus my own six
   rev5 probes (`probes/zz_reviewer_rev5_test.go`, `logs/probe-rev5-own.log`,
   all PASS): PR5-1 a CRLF member cut before its final LF keeps the tail with the
   CR inside its range (offset `len(l1)+2`, length `len(l2)+1`, `opaque_event`);
   PR5-2 a CRLF blank line refuses naming line 2 (never dropped); PR5-3 four
   non-array `directives` shapes refuse `not an array of strings`; PR5-4 foreign
   encrypted and signed reasoning project `opaque_reasoning`/`opaque` with the
   two stable reasons, the raw ref resolves byte-exact (multibyte + astral
   ciphertext), the sealed event carries no `ciphertext`/`content_blocks`/`text`/
   `summary`, a foreign HIGH instruction after a native LOW one leaves the
   snapshot `low/[be brief]` and projects with `authority=low`, foreign usage
   lands in the source ledger with target totals zero; PR5-5 one member with a
   definition, a completed call, an incomplete call, an orphan result and a
   foreign nested `subagent:worker` call: live surface empty, `c2`/`c3`/`c4`
   aborted with `unsafe_pending_action`, `internal` visibility, the nested call
   attributed to the mapped subagent actor (kind `subagent`, parented), and a
   second projection into a fresh sink byte-identical (session + 6 events);
   PR5-6 a 128-character subagent name passes the shape gate (refuses only as
   unmapped) and 129 refuses `past 128 characters`.
2. **Suite**: `go test ./... -count=1 -v` on the candidate copy with the board
   materialized: 44/44 packages ok, 2796 top-level PASS, 0 FAIL, the 2
   pre-existing environment SKIPs (`logs/cmd04-full-suite-v.log`, exit 0);
   Story packages verbose: 197 top-level + 301 subtests PASS, `TestNormalizeRefusalTable`
   54 subtests, 40 `TestNormalize*` top-level (`logs/pkg-story-v.log`);
   `-race -count=1 -timeout 25m` on cloneproject, clonebundle, clonesnap,
   traceability/..., localstore: all ok, 0 `DATA RACE` (`logs/cmd05-race-story-packages.log`).
   gofmt on the CR's Go files 0, `go build`, `go vet`, linux and windows builds,
   `cataloggen -adopted … -check`, JSON sweep (34 tracked files, 0 bad),
   `git diff --check` — all exit 0 (`logs/cmd*.log`). Not re-run by me and
   accepted from the CR construction summary (`required=30 green=30 failed=0 missing=0`)
   and the producer's archived logs (real gzip, 424-entry `MANIFEST.sha256`
   with no self-entry, `shasum -c` all OK, zero `__pycache__`): the other 38
   packages' race lanes (`cmd05a-g`: 44 ok, 0 DATA RACE), the 17 fuzz gates
   (`cmd07-23`, all exit 0) and `task-board validate` (`cmd29`, exit 0).
3. **Mutation**: producer cloneproject battery reproduced twice on four
   battery copies (86/86 `ok:` each run — 85 KILLED + `C-doc-comment` SURVIVED,
   verdicts identical across runs, subprocess exits 85×1 + 1×0 in the per-plant
   logs, `mutants/producer-run{1,2}/`, copies byte-identical after each run);
   clonebundle 76/76 twice (75 KILLED + control, `mutants/sibling-clonebundle-run{1,2}/`);
   clonesnap 44/44 once (43 KILLED + control; package byte-identical to rev4).
   Reviewer plants: 26 rows per run, 2 runs, every verdict identical across runs.
   Section-0 re-plants: 14 KILLED + 1 SURVIVED control (table above). My own 11
   rows on gates the producer did not mutate (`reviewer-plants/plants-rev5-own.json`):
   KILLED 5 — `RV5-subagent-129` (bound +1; dies by the actor-mapping message,
   i.e. `long_subagent_name` is a message pin of the same honest class as
   `empty_subagent_name`), `RV5-heads-unsorted-build-helper` (`<=`→`==` in
   `parseSortedUniqueDigestStrings`: both `heads_unsorted` and `parents_unsorted`
   build rows admit), `RV5-heads-subset-decode-dropped` (decode `head_outside`),
   `RV5-session-selfdigest-skipped` (pre-admission revalidation skipped:
   `decode/self_mismatch` admits, `error = <nil>`), `RV5-inline-65537-builder`
   (`block_inline_oversize` admits); SURVIVED 5 + 1 control —
   `RV5-native-type-513` (matrix D states this edge as a bound: honest),
   `RV5-included-no-raw-entry-dropped` (defensive arm unreachable through
   sealed, self-digested manifests; denominator only), and the three P3
   advisories below; `RC5-control-comment-clonebundle` SURVIVED.
4. **Bounds and hygiene**: the 66 CR paths are exactly {`LOGBOOK.md`, `README.md`,
   `internal/clonebundle` 22, `internal/cloneproject` 16, `internal/clonesnap` 20,
   `internal/traceability` 6}; nothing outside the Story's scope differs from
   trunk `42d5a95` (the CR base is trunk, so there is no stale-refresh window);
   `internal/clonebundle` and `internal/clonesnap` production byte-identical to
   rev4; the cloneproject production delta over rev4 is the three-line CRLF
   comment. Citation census (`logs/citation-census.log`): all 56 top-level test
   names and 57 subtest citations in the matrix exist; all cited names in the
   two changed TRACEABILITY files exist; all 86 cloneproject harness rows are
   cited. 13.14.2–13.14.5 non-ownership is stated in matrix D with owners.

## Coverage ratio I measured

47 of 47 matrix rows are driven through `Normalize` over `clonesnap.Capture`
output by a named committed test (every cited test, subtest and harness row
exists and passed in my run). The Story-level closed-shape claim that rev4
measured at 1 of 8 sites is now 8 of 8 build/decode sites witnessed
behaviourally at the far edge. Producer batteries: 85 + 75 + 43 KILLED, three
SURVIVED controls, identical across runs. Reviewer plants: 26 rows × 2 runs,
19 KILLED / 7 SURVIVED (2 controls, 1 stated bound, 1 defensive-arm
denominator, 3 advisories), every row with a raw log and subprocess exit per run.

## P3 advisories (production correct in each case; none is an AC-named or matrix-claimed measurement; follow-up material, not a rework)

- **P3-ii `directives` array-type gate unpinned.** `RV5-directives-type-dropped`
  (`native.go:393-395`: the `json.Unmarshal(members["directives"], &directives)`
  error ignored, so `directives: 5` decodes as zero directives) SURVIVED 2/2
  against `-run TestNormalize`. Production refuses all four non-array shapes
  (PR5-3). This is the third member-type shape of the P3-v family (text→
  `body_text_number`, bool→`body_live_attestation_string` are rowed); matrix D's
  "body shapes exact" is one row short. One `TestNormalizeRefusalTable` row +
  narrowing closes it.
- **P3-jj `internal/clonebundle` string far edges at the caller literals.**
  `RV5-title-4097-build` (`session.go:300`, `4096`→`4097`) and
  `RV5-actor-name-513-build` (`session.go:158`, `512`→`513`) SURVIVED 2/2
  against the whole package; production refuses 4097/513 and admits 4096/512
  in characters (PR5-7, `probes/zz_reviewer_rev5_bounds_test.go`). The character
  measure is pinned at one site only (`TestMultibyteStringMeasure`, native key
  513); each caller carries its own literal. Not in this leaf's AC (4) list
  (actors/event IDs/heads/parents/inline content are, and are closed); either
  two far-edge rows in `refusal_session_event_test.go` or a stated bound in
  `internal/clonebundle/TRACEABILITY.md`.
- **P3-kk stale line numbers.** The results' seven-site census quotes
  `native.go:337/357/378/…`; on the candidate the sites are `340/360/381/419/452/487/522`
  (the CRLF comment added three lines above them). Cosmetic.

## Rework scope (for the producer)

None required for acceptance. If the orchestrator routes a follow-up: P3-ii
one refusal row + narrowing; P3-jj two far-edge rows or a stated bound;
P3-kk refresh the line numbers.
