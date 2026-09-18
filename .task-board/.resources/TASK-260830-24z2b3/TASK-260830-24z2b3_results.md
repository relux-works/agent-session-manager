# TASK-260830-24z2b3 Results — implement-clone-bundle-and-canonical-session-types (rev5 completion)

Status: ready for review. Candidate left UNCOMMITTED in the Story
worktree (`task-board/story/STORY-260830-1cyj0q`) on base `a12d1bd`
(`origin/main` at handoff); no commit on the Story branch. This run
completes rework rev5 of review RUN-260917-ab25c8, which routed rev4
CHANGES REQUESTED with one P2 and four small P3 items and graded
every rev3 finding fixed. The first rev5 construction attempt failed
at validation command 6 on the trunk resumesmoke time-bomb
(BUG-260918-354b03, fixed on trunk as `a12d1bd`), not on the
candidate; this run refreshed onto that fix and re-executed all
evidence. Every finding is answered below with the change, the test
that fails without it, and the evidence path.

## Start-of-run recovery note

When this run started, the worktree carried the completed rev5
candidate plus one piece of debris: `M LOGBOOK.md`, `M README.md`,
`?? internal/clonebundle/`, `?? internal/resumesmoke/zz_debug_test.go`
(a scratch "deleted after use" probe, not part of any Change Request
patch — removed). Per the brief, `task-board worktree
refresh-candidate TASK-260830-24z2b3` ran FIRST and advanced the base
`2fc6d50` → `a12d1bd` (`refresh_advanced`). The audit was NOT empty:
`git diff --name-only 2fc6d50 a12d1bd` names only
`internal/resumesmoke/support_test.go` (the trunk deadline fix), and
the refresh had carried the OLD base's version of that file forward
as an uncommitted delta (reverting the fix in the worktree), so trunk
content was restored by hand (`git checkout HEAD -- <file>`,
`smokeDeadlineValue` present afterwards). No other trunk path touches
the candidate. `git status` at handoff shows only candidate paths. No
`internal/traceability` edit (FINAL leaf carries bindings + re-pin).
Authority stays `SPEC.v0.7.0.md` with the same heading numbers. The
rev5 candidate code is unchanged by this run except the LOGBOOK REV5b
refresh note; every check below was re-executed on the new base.

## Finding-by-finding table

| Finding | Change | Test that fails without it | Evidence |
|---|---|---|---|
| P2-δ Build entries rewrite invalid UTF-8 to U+FFFD and seal it; `BuildCaptureManifest` + `IdentityDigest` launder a native key past the sanitizer; "native\xff" vs "native\xfe" seal byte-identical | Shared `validText` gate (`decode.go`) at every Build-side string admission — session title, actor name/model (`session.go`), evidence native_type + reason codes (`event.go`), item exclusion_reason (`captureitem.go`), generation token (`generation.go`), identity kind (`identity.go`) — each refusing with a member-naming "not valid UTF-8" detail before marshaling; `EncodeNativeIdentity` (`identity.go`) sanitizes `NativeSessionID` and `OpaqueIdentity` before marshaling so the caller's re-decode never sees rewritten bytes; `encodeExtensions` (`rawmanifest.go`) walks keys and string values at any depth (maps, slices, arrays, pointers, interfaces, exported struct fields) and refuses invalid UTF-8 before `json.Marshal`. Re-verified on the new base: both laundering entries refuse, the collision pair refuses on both sides | `TestBuildTextMustBeValidUTF8` (`text_test.go`): 22 rows (capture identity native + lone surrogate + opaque, item reason, boundary generation, capture/session/event/raw extensions incl. nested/key/struct shapes, digest opaque + native-collision-pair, session title/actor name/actor model, evidence native_type/reason, `ParseGeneration`) + the collision-pair loop; every row asserts ErrInvalid with a "not valid UTF-8" fragment AND that no returned bytes carry U+FFFD. `TestEncodeNativeIdentityRefusals` carries the direct unsanitized-identity row | `pkg-test.log`, `mutants/run1/N-text-utf8.log`, `mutants/run2/N-text-utf8.log` (narrowing plant admits exactly `ti\xff`tle past the shared gate; the `session_title` killer fails on the sealed-bytes U+FFFD assertion) |
| P3-a″ producer evidence archive contaminated (foreign TASK-260909-2ez769 artifacts + `__pycache__`) | Archive rebuilt from the task-scoped `producer-evidence-rev5b/` directory only: 27 command logs, package logs, 2×67 per-plant logs, two battery logs, the P3-b probe log, and a fresh `MANIFEST.sha256`. Verified: zero `__pycache__`/`.pyc` entries; every tar member except the manifest itself and directory entries is listed in the manifest | n/a (artifact hygiene; verified by listing the tar contents against the manifest) | `TASK-260830-24z2b3_producer-evidence.tar.gz` + `MANIFEST.sha256` inside |
| P3-b″ TRACEABILITY over-claims "`-0` and leading-`+`/leading-zero admit" | Sentence fixed: `-0` admits (sealed as `0`, the JCS form); `+1` and `01` refuse at the frame gate. First-hand re-verified on the new base (reviewer vector B8 re-executed) | Scratch probe `TestZZNumberLiteralBound` (deleted after use): `+1`/`01` refuse `not a JSON object (member extensions)`; `-0` passes the extension gate (tampered seal then fails the self-digest check with the same deterministic digest, which is the proof the gate passed) | `p3b-probe.log` |
| P3-c″ build-side reason-code duplicate unmeasured (`R4-reason-codes-dup-build` SURVIVED 2/2) | `reasons duplicate` row (`["a","a"]` at Build → `sorted unique`) in `TestCanonicalEventRefusals/evidence`, mirroring the decode-only row. Mutant anchors byte-verified (`\t\tif index > 0 && reason <= previousReason {`) | Same row; `N-reason-codes-dup-build` weakens the build comparator to `<` and the row fails | `pkg-test.log`, `mutants/run1/N-reason-codes-dup-build.log`, `mutants/run2/N-reason-codes-dup-build.log` |
| P3-d″ producer ticked reviewer-owned checklist rows | HANDOFF-GATE-EXPERIMENT (see below): with only the producer rows checked, `task-board handoff` refuses (the gate verifies every checklist item); the four verdict rows were therefore each verified true this run and checked so the handoff can construct. Producer rows re-completed by re-execution, not carried over | the named evidence for each row (matrix, TRACEABILITY ownership section, `cmd*.log`/`pkg-*.log`, `mutants/run1+2/`); the handoff refusal output for the producer-only state | board checklist state + activity log at handoff |

Also: TRACEABILITY clause map carries the §1.6 text-gate row
(`validText` + `EncodeNativeIdentity` sanitize +
`encodeExtensions` walk → `TestBuildTextMustBeValidUTF8`) and states
the walker bound (custom `json.Marshaler` outputs trusted); LOGBOOK
carries the REV5 bullet plus the REV5b refresh note (same block);
README package section unchanged (commands still exact, no capability
claim).

## P3-d handoff-gate experiment (this run, executed)

The brief asks to re-complete only the producer checklist rows. This
run executed that state literally first: items 15-18 (Implementation
matches AC; Solution fits project architecture; Tests green; Gate …
attacked, not read) unchecked, items 1-14 and 19 checked, with
task-scoped outcome resources present on the board (the rev5
attachments, so the checklist was the only missing variable; the
rev5b updates attach before the final handoff). `task-board handoff
TASK-260830-24z2b3 --role
developer` in that state refuses:

> cannot hand off TASK-260830-24z2b3: unchecked checklist items
> [15 16 17 18] (Implementation matches AC; Solution fits project
> architecture; Tests green; Gate, refusal, validation,
> authorization, and attestation behavior attacked, not read —
> positive-path-only evidence is not accepted): handoff evidence
> missing

Status stays `development` (fails closed, no transition). The
uncheck → refusal → re-check sequence is in the board activity log.
The four verdict rows were therefore each verified true this run and
re-checked so the Change Request can construct:

- 15 Implementation matches AC: 18 of 18 AC rows driven through
  production entries by named tests (conformance matrix).
- 16 Solution fits project architecture: validating owner over the
  landed owners (canonicaljson/canonicalize, sessadapter tuples,
  environ gates, scalar, hosttrust, localstore); no forked model,
  no `internal/traceability` edit (TRACEABILITY.md ownership
  section, `doc.go`).
- 17 Tests green: 27/27 validation commands exit 0 on the final
  tree plus package test/cover/race green (`cmd*.log`,
  `pkg-*.log`).
- 18 Gates attacked, not read: 66 narrowing mutants KILLED plus
  the harmless SURVIVED control, battery run twice with per-plant
  raw logs; sealed-bytes U+FFFD assertions on the text gate
  (`mutants/run1+2/`, `text_test.go`).

The reviewer resets verdict rows per P3-d″ as before; the tool gate,
not producer preference, sets the handoff-time state.

## AC coverage: 18 of 18 rows driven

See `TASK-260830-24z2b3_conformance-matrix.md` (rev5, re-executed on
`a12d1bd`): every row names its production call site and driving
test. The ratio stays **18 of 18 AC rows driven**; no matrix row
changed because the candidate code is unchanged.

## Validation reran in rev5b (this run, final tree, base `a12d1bd`)

- Commands 1-3 (`gofmt` check, `go build ./...`, `go vet ./...`):
  exit 0 (`cmd01.log`, `cmd02.log`, `cmd03.log`)
- Command 4 (`go test ./... -count=1 -v`): exit 0, 21967 passes,
  38 packages ok, zero `^FAIL` and zero `--- FAIL`
  (`cmd04.log`; the FAIL-text lines inside are nested mutant-kill
  output of the passing `TestSmokeMutantsAreKilled`)
- Command 5 (`go test ./... -race -count=1 -timeout 25m`): exit 0,
  38 packages ok, zero `DATA RACE` lines
  (`ok internal/sessquery 390.347s`, `ok internal/clonebundle
  3.172s`) (`cmd05.log`)
- Command 6 (`go test ./... -cover -count=1`): exit 0, 38 packages
  ok — the command that failed the first rev5 construction on the
  trunk time-bomb now passes with the `a12d1bd` fix
  (`ok internal/resumesmoke`, `ok internal/clonebundle 87.1%`)
  (`cmd06.log`)
- Commands 7-20 (fuzz smokes, 14 targets): exit 0
  (`cmd07.log` … `cmd20.log`)
- Command 21 (`tracecheck`): exit 0, bindings=68,
  clauses 49/569 — unchanged, `internal/traceability` untouched
  (`cmd21.log`)
- Command 22 (`cataloggen -adopted -check`): exit 0, empty output
  (`cmd22.log`)
- Commands 23-24 (`GOOS=linux/windows` builds): exit 0
  (`cmd23.log`, `cmd24.log`)
- Commands 25-27 (JSON validation, `task-board validate`,
  `git diff --check`): exit 0 (`cmd25.log`, `cmd26.log`,
  `cmd27.log`)
- Package suite: exit 0 (`pkg-test.log`), 87.1% statements
  (`pkg-cover.log`); package race exit 0 (`pkg-race.log`)
- Mutant harness: 67/67 ok (66 KILLED + 1 SURVIVED control),
  full battery run twice with per-plant raw logs carrying
  subprocess exits (`mutants/run1/*.log`,
  `mutants/run2/*.log`, `mutants-run1.log`, `mutants-run2.log`),
  `PYTHONDONTWRITEBYTECODE=1`, no `__pycache__`, tree restored
  (`git status` unchanged after both runs)

- No `internal/traceability` edit; `git status` shows only
  candidate paths (`M LOGBOOK.md`, `M README.md`,
  `?? internal/clonebundle/`)

## Bounds for siblings (not waived)

- Per-kind event payload fact registries: which facts a kind may
  carry (normalization sibling). The §1.6 value model on every
  payload member is enforced here, not bounded away.
- `maximal_safe` projection planning past
  `RefuseMaximalSafeUnlessComplete` (projection sibling).
- Target-branch (G2) admission past `RefuseUnstableForTarget`.
- Section 7.8 operation wiring (stays `sessadapter`).
- `-0` admits (JCS `0`); `+n`/leading-zero literals refuse at the
  frame gate (stated, not waived).
- Custom-`MarshalJSON` extension values: the text walk trusts the
  computed form (stated, not waived).
- `internal/traceability` bindings + re-pin (FINAL leaf).

## Attachments

- `TASK-260830-24z2b3_conformance-matrix.md` (rev5, re-executed)
- `TASK-260830-24z2b3_producer-evidence.tar.gz` (rev5b logs:
  27-command suite, package suite, 2×66+1 mutant battery with
  per-plant logs, P3-b probe log, file manifest)
