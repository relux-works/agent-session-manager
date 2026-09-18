# TASK-260830-32ypr2 Results (rev5) — test-unknown-encrypted-and-incomplete-events

Status: ready for review. Candidate left UNCOMMITTED in the Story worktree
(`task-board/story/STORY-260830-1cyj0q`); no commit on the Story branch.
Branch tip: `5cc9449` (the `TASK-260830-2g5be6` checkpoint over trunk
`42d5a95`); `origin/main` is `42d5a95` at handoff — trunk did NOT move
during this run, so no refresh ran and the rev4 registry MERGE the
reviewer verified row by row stands untouched.

## Inherited work vs this run

Inherited the rev4 candidate tree verbatim (uncommitted working tree:
the complete `internal/cloneproject` package, the merged registry +
re-derived pin, README/LOGBOOK/traceability edits) plus the rev4
review packet (verdict RUN-260918-492d65, evidence tar). No partial
rev5 edits existed. This run changed nothing the verdict grades
CLOSED or CONFIRMED: no production file changed (test-plus-matrix
only; the one production-tree edit is a doc comment — the CRLF
sentence on `splitRecordLines`, P3-ff). Nothing is accepted from
prior runs: every exit quoted here was observed on the final tree
in this run.

## Finding-by-finding table (rev4 verdict → change → test → evidence)

| Finding | Change | Test that fails without it | Evidence path |
|---|---|---|---|
| P2-f AC-named scale bounds unpinned at far edges (parents 65 build/decode, heads 1025 decode, event IDs 1000001 decode, actors 1025 build SURVIVED 2/2; heads/event-IDs build died by low-edge message literal only) | Far-edge refusal rows in `internal/clonebundle/refusal_session_event_test.go`, each asserting code (`ErrInvalid`) + message: parents 65 build (`carry 65 IDs, want [0..64]`) + decode (`parents are not sorted unique digest[0..64]`); heads 1025 build (`carry 1025 IDs, want [1..1024]`) + decode (`head_event_ids are not sorted unique digest[1..1024]`); actors 1025 build (`carries 1025 actors, want [1..1024]`; decode row pre-existed); event IDs 1000001 build + decode (`carry 1000001 IDs, want [1..1000000]`). Build-side inputs are otherwise valid (65/1025/1025 sorted-unique digests, 1025 valid actors, 1000001 distinct digests with head inside), so each widening kills by admission; decode-side by the self-digest/per-element message. 8 widening narrowings (one per site, incl. the decode-actors narrowing the suite lacked). Matrix section D + `internal/clonebundle/TRACEABILITY.md` cite rows by name instead of "owned by the accepted suite" | The 7 new rows (+ pre-existing decode `too many actors` for its new narrowing) | `mutants/sibling-clonebundle-run1/N-{parents,heads,actors,eventids}-max-{build,decode}.log` + `run2` (8 narrowings, each widening exactly one bound by one); `pkg/pkg-siblings.log`; decode-event-IDs cost measured, not bounded: 0.21 s per run (no explicit cost bound needed) |
| P3-bb unterminated last record of a multi-line member unwitnessed (`RV-unterminated-tail-dropped-multiline` SURVIVED 2/2) | `TestNormalizeUnterminatedTailResolvesWithOffset`: two-record member without a trailing newline; asserts 2 events with offsets `0`, `len(l1)+1`, lengths, and per-line resolution | The new test | `mutants/run1/N-unterminated-tail-drop.log` + `run2` (labelled behaviour drop, killer); `pkg/pkg-cloneproject-v.log` |
| P3-cc per-type body registries pinned only through the shared loop (`RV-allowed-call-live-followup` SURVIVED 2/2) | Rows shipped (not the bound): `call_live_followup_member` (call smuggling the result-only optional refuses `body carries unknown member "live_followup"`) + `N-allowed-call-live-followup`; `result_live_attestation_member` + `N-allowed-result-live-attestation`. Seven-site census below | The new rows | `mutants/run1/N-allowed-call-live-followup.log`, `N-allowed-result-live-attestation.log` (+ `run2`) |
| P3-dd usage fold pinned along the input axis only (`RV-usage-output-arm-widened` SURVIVED 2/2) | `usage_ledger_overflow_output`: output totals 2^53-1 then 1; refuses `source output token total exceeds uint53` + `N-usage-overflow-output` | The new row | `mutants/run1/N-usage-overflow-output.log` (+ `run2`) |
| P3-ee decode-side `raw_refs` sortedness pins the duplicate but not two distinct unsorted refs (`RV-rawrefs-unsorted-decode` SURVIVED 2/2) | `TestCanonicalEventRefusals/decode/evidence_raw_refs_unsorted`: two distinct refs in descending canonical order refuse `raw_refs are not sorted unique` + `N-rawrefs-unsorted-decode` (complements `N-rawrefs-dup-decode`) | The new row | `mutants/sibling-clonebundle-run1/N-rawrefs-unsorted-decode.log` (+ `run2`) |
| P3-ff CRLF shape unstated (trailing CR inside the record range) | STATED beside the "one trailing newline is the terminator" sentence (`splitRecordLines` doc comment) AND pinned: `TestNormalizeCRLFResolvesWithCarriageReturn` (ranges cover line plus CR, offsets `0`, `len(l1)+2`, resolve exactly) + matrix §D + TRACEABILITY bound entries | The new test (admitted shape, no gate — same disclosure class as rows 38/45/46/47) | `pkg/pkg-cloneproject-v.log` |
| P3-gg prose | Matrix §D `live_followup:null` row names `TestNormalizeNullOptionalBoolsDecodeAsFalse` (was `...AsEmpty`); `"forty-seven"` dropped from `coverageNumberWords` in `readme_coverage_pin_test.go` (the map is lookup-only; the pin test still passes — `pkg/pkg-traceability.log`) | n/a (prose) | File diffs |
| P3-hh `subagent:` with an empty name is a message-only arm (`RV-actor-empty-subagent-name` SURVIVED 2/2) | Row shipped (not the message-only statement): `empty_subagent_name` refuses `actor "subagent:" is outside main\|external\|subagent:<name>` + `N-actor-empty-subagent-name`, honestly labelled a message pin (the plant still refuses one step later at the actor mapping) | The new row | `mutants/run1/N-actor-empty-subagent-name.log` (+ `run2`) |

## Per-type body-registry census (P3-cc; the verdict says six — the tree carries seven)

`decodeBodyObject` call sites in `internal/cloneproject/native.go`:

| Site (line) | Type | `allowed` map | Narrowing |
|---|---|---|---|
| 337 | message / reasoning-summary | `{"text"}` | shared loop via `N-body-member` (`escalate` on a message body) |
| 357 | protected reasoning | `{"ciphertext"}` | shared loop via `N-body-member` |
| 378 | instruction | `{"authority", "directives"}` | shared loop via `N-body-member` |
| 418 | tool definition | `{"tool_name", "definition_digest", "live_attestation"}` | shared loop via `N-body-member` |
| 451 | tool call | `{"call_id", "tool_name"}` | `N-allowed-call-live-followup` (this run) |
| 486 | tool result | `{"call_id", "status", "live_followup"}` | `N-allowed-result-live-attestation` (this run) |
| 521 | usage | `{"input_tokens", "output_tokens"}` | shared loop via `N-body-member` |

The shared exact-shape loop refuses any unregistered member on any
type (`N-body-member` narrows the loop itself); the call and result
registries — the two sites a foreign optional can plausibly land
on — carry their own registry narrowings. The remaining five maps
are covered by the shared-loop narrowing, stated here, not waived.

## AC coverage: 47 of 47 rows driven and measured

`TASK-260830-32ypr2_conformance-matrix.md` (attached, rev5):
row 4 extended (unterminated-tail + CRLF tests, `N-unterminated-tail-drop`
labelled killer); row 10 extended (`N-usage-overflow-output`);
row 23 now 54 subtests with the 4 new narrowings; section C cites
the new tests/subtests; section D rewritten (far-edge rows cited by
name, `live_followup:null` name fixed, CRLF + empty-subagent +
per-type-registry entries); section E counts updated (54+3,
5 labelled rows).

Measured, not planned: every gate row carries a directly-reddening
narrowing or labelled row (80 narrowings + 5 labelled rows, battery
green twice). Disclosed non-narrowed members: row 15 post-boundary
(gate-level only); rows 18/19/21 shape internals (clonebundle suite,
now with the far-edge rows cited by name); row 24 (failure-injection
proofs); row 37 (strict-decode rows); rows 38/45/46/47 + the CRLF
shape (admitted shapes, no gate). No row is prose-only.

## Mutants

`internal/cloneproject/testdata/mutant_harness.py`: 80 narrowing
rows + 5 labelled non-narrowing rows (`N-empty-projection`
single-member class, `N-overflow-truncate` behaviour swap,
`N-pending-aborted` add-arm, `N-offset-newline` behaviour swap,
`N-unterminated-tail-drop` behaviour drop) + harmless
`C-doc-comment` SURVIVED control. Battery executed TWICE on the
final tree, exit 0 every chunk both runs, 86/86 `ok:` lines each
(85 KILLED + control SURVIVED), one raw log per plant per run
carrying the mutant identity, patch, subprocess exit, and FAIL line
(`mutants/run1/`, `mutants/run2/`, `run1-battery-chunk*.log`,
`run2-battery-chunk*.log` in the archive). Sibling harnesses on the
final tree: clonebundle 76/76 `ok:` TWICE (75 KILLED + control;
`mutants/sibling-clonebundle-run1/`, `run2/`), clonesnap 44/44
`ok:` once (43 KILLED + control; package byte-identical to rev4).
`PYTHONDONTWRITEBYTECODE=1`, zero `__pycache__`/`.pyc` (verified
after the runs). Kill taxonomy: every build-side far-edge plant
kills by admission (`error = <nil>, want ErrInvalid`); every
decode-side far-edge plant kills by the self-digest/per-element
message; `N-actor-empty-subagent-name` kills by the actor-mapping
message (honest message pin, stated in the row note, matrix, and
TRACEABILITY).

## Story-close (rev5 deltas: none to the merge)

- Registry: UNTOUCHED — the rev4 merge (152 cases / 74 rows) stands
  exactly as the reviewer verified it row by row. No binding added,
  removed, or reworded.
- Pin: UNCHANGED and re-proven. Zeroing the pin on the final tree
  makes the gate print the derived digest (`digest-derivation.log`,
  exit 1):

  ```
  $ go run ./internal/traceability/cmd/tracecheck -root . [pin zeroed]
  spec-to-code traceability check failed: ownership registry projection digest 614eb55d70b4b83f1ff668b8bfe9830d0655189a3a76f65edefb5159c6710ecb differs from reviewed 0000000000000000000000000000000000000000000000000000000000000000
  exit status 1
  ```

  The printed digest equals the pinned value; the pin file was
  restored byte-identical (`cmp` clean).
- `tracecheck` green, quoted verbatim:

  ```
  traceability ok: contracts=64 normative_sections=36 acceptance_cases=152 fixtures=33 compatibility_contracts=55 assigned_scopes=0
  section coverage: bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574
  ```

  Section runs (both admitted, `assigned_scopes=1`, same coverage
  line): `--section 7.8` exit 0, `--section 10.2` exit 0 (logs
  `sections/section-7.8.log`, `sections/section-10.2.log`).
- README: untouched (no new package; figures already equal what
  tracecheck prints; pin test green after the `forty-seven` drop).
- LOGBOOK: REV5 + REV5 EVIDENCE lines appended to the leaf block
  (newest-first preserved; history not rewritten).
- `task-board.config.json` byte-identical to `origin/main`
  (verified with `cmp`; 30 commands, read from the candidate config).

## Validation reran (this run, final tree — nothing accepted from prior runs)

All 30 configured commands (read from the candidate
`task-board.config.json`, `spawn.worktree_isolation.validation.commands`),
exits observed firsthand (logs `cmds/cmd01.log` … `cmds/cmd30.log`
in the archive; race gate as `cmd05a` … `cmd05g`; every log carries
its command line + output + exit):

- 1 gofmt check: 0. 2 build: 0. 3 vet: 0.
- 4 `go test ./... -count=1 -v`: 0 — 44 packages `ok`, 2796
  top-level PASS (+2 new tests vs rev4), zero FAIL.
- 5 race gate: 0 in 7 bounded groups (8+7+7+7+7+7+1 = 44
  packages `ok`, zero `DATA RACE`, zero FAIL; `sessquery` alone
  in group G, 393.6 s), each with `-timeout 25m`, instead of one
  long call per the headless time bound.
- 6 cover gate: 0 — 44 `ok`; `cloneproject` 92.4%, `clonebundle`
  87.2%, `clonesnap` 87.5% of statements.
- 7–23 fuzz gates (rpcwire ×4, scalar ×1, canonicaljson ×4,
  secconftest ×8, each `-fuzztime=100x -parallel=1`): all 0.
- 24 tracecheck: 0 (ratio above). 25 cataloggen `-adopted … -check`: 0.
- 26 linux build: 0. 27 windows build: 0. 28 JSON parse sweep: 0.
- 29 `task-board validate`: 0 (board-hygiene notes on unrelated
  elements only; zero mention this task or story —
  `grep -c '32ypr2\|1cyj0q'` prints 0).
- 30 `git diff --check`: 0.
- Package evidence on the final tree: `pkg-cloneproject-v.log`
  (108 PASS lines / 40 top-level, zero FAIL),
  `pkg-cloneproject-cover.log` (92.4%),
  `pkg-cloneproject-race.log` (`ok`, zero `DATA RACE`),
  `pkg-siblings.log` (clonebundle + clonesnap `ok`),
  `pkg-traceability.log` (both suites `ok`).
- Config check: `cmp` of `task-board.config.json` against
  `origin/main` bytes is silent (byte-identical); `git status`
  shows only candidate paths (LOGBOOK, README, 3 clonebundle
  files, 2 clonesnap prose files, 7 traceability files,
  untracked `internal/cloneproject/`).
- Citation census (measured, not claimed): all 56 test names the
  matrix cites exist; all cited subtests exist (clonebundle
  space-form names match the underscore-form citations — the
  same normalization `go test -run` applies, and every harness
  `run=` anchor executed); all 86 cloneproject harness rows are
  cited and all cited rows exist; the 9 new clonebundle rows are
  cited in matrix §D and `internal/clonebundle/TRACEABILITY.md`.

## Bounds (sibling scope, not waived)

Rev4 bounds stand, plus: CRLF members admitted with the CR inside
the record range (pinned, not waived); the empty-`subagent:` arm
is a message pin (plant still refuses at the actor mapping);
message/protected/instruction/definition/usage body registries are
covered by the shared-loop narrowing with the call/result
registries narrowed per site (census above); decode-side far-edge
kills arrive via the tamper-broken self digest (the same mechanics
as every pre-existing decode row, e.g. `N-main-count-decode`).
No `ax` command, `doctor` result, or runtime capability is claimed
anywhere.

## Handoff state

Candidate UNCOMMITTED in the Story worktree for the handoff
snapshot. Checklist: all producer items verified true this run
and checked (evidence above + attached). Attached:
`TASK-260830-32ypr2_results.md` (this file),
`TASK-260830-32ypr2_conformance-matrix.md`,
`TASK-260830-32ypr2_producer-evidence.tar.gz` (real gzip: 30
command logs, 7 race-group logs, package logs, sibling harness
logs, 2×86 + 2×76 + 44 per-plant logs, 10 battery-chunk logs,
section-run logs, digest-derivation log, `MANIFEST.sha256`).
