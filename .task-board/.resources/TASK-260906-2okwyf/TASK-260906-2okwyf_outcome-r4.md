# TASK-260906-2okwyf round 4 — F3 stray file + N1–N4 instrument hardening: outcome

Status: ready for review (handed off to review; review/integration are orchestrator steps).

Candidate tree: `8bfa6a51702ad1856c5f2f68b69b6dd2a262f065`
(verified: detached index — `GIT_INDEX_FILE` copy + `read-tree HEAD` +
`add -A` + `write-tree` — worktree index untouched, HEAD stays at the
`114a056` checkpoint, work uncommitted. Derived AFTER the last tracked
write (the LOGBOOK append). `git ls-tree -r` contains zero `.bak`
paths. The two untracked test files' tree blobs are byte-equal to the
worktree files by `git hash-object`. The Change Request record is the
orchestrator's integration step; this OID is the candidate it must carry.)

## 0. Scope — exact changed-path enumeration (13 paths, read back both directions)

`git diff-tree --name-only -r HEAD <candidate>` and
`git status --porcelain` agree exactly; every path below appears in both,
and neither contains any other path. This enumeration IS the comparison
the F3 fix requires — not a summary of it.

Modified (11):

- `LOGBOOK.md` — round-4 entry (newest first); N4 fixes inside (59
  derived = 10 pair + 5 hostile-only + 44 free; round-2 entry cites rev1).
- `internal/provhost/probe.go` — round-1 A9 change, untouched since.
- `internal/provhost/probe_test.go` — round-1 A9 change, untouched since.
- `internal/provhost/protocol.go` — round-1 A9 change, untouched since.
- `internal/provhost/protocol_test.go` — round-1 A9 change, untouched since.
- `internal/provider/provider.go` — round-1 A7 change, untouched since.
- `internal/terminalbackend/digit_guard_census_test.go` — round-1
  sentence correction, untouched since.
- `internal/terminalbackend/manifest.go` — round-3 Reconcile entry gate,
  untouched since (both round-4 plants reverted byte-identical; `grep`
  for PLANT across `internal/` returns nothing).
- `internal/terminalbackend/refusal_arm_inventory_test.go` — ROUND 4:
  `refusalArm.backendID` AST-key flag, `compositeLitNamesBackendID`,
  `deriveRefusalArmsIn` dir parameter, occurrence-key zeroing.
- `internal/terminalbackend/terminalbackend.go` — round-2
  CheckVersionTuple gate, untouched since.
- `internal/terminalbackend/terminalbackend_test.go` — round-2 hostile
  test, untouched since.

New (2):

- `internal/provhost/profile_agreement_test.go` — round-1 A5 pin,
  untouched since.
- `internal/terminalbackend/backend_id_entry_census_test.go` — round-3
  instrument + ROUND 4: AST-keyed site filter, fail-closed collision
  gate, two synthetic-dir derivation tests, N3 bound.

Round-4 behavioral delta: NO production file changed. The only
production-adjacent edit is test-instrument derivation. `F3` moved
`probe_f2_main.go.bak` (F2 probe, `package main`, repo root) to
`.temp/TASK-260906-2okwyf/` (gitignored: `git status` clean of it,
`ls-tree` carries zero `.bak` paths).

## 1. F3 — stray file out of the tree (fixed)

`probe_f2_main.go.bak` no longer exists at the repository root; it
lives under the task temp dir the repository instructions name for
validation artifacts. Verified three ways: `ls` at root (absent),
`git status --porcelain` (no `??` row for it), and the candidate
`ls-tree` read-back above (0 `.bak` paths in the full listing).

## 2. N1 — site filter is AST-keyed (fixed, with a test that fails before)

`deriveBackendIDSites` filtered on `strings.Contains(line, "BackendID:")`
against the arm's production line. A `refuse(&Error{...})` arm written
as a multi-line composite literal keeps `BackendID:` off that line, so
the site never entered the denominator while four sibling instruments
stayed green. Reproduced pre-fix on this tree: a dead multi-line arm
planted in `checkProbeIdentity` left `TestBackendIDEntryCensus` GREEN
(exit 0).

Fix: `deriveRefusalArms` records `backendID` from the refuse literal's
AST keys at derivation (`compositeLitNamesBackendID`); the filter reads
the flag. Completeness notes: positional literals cannot reach the flag
(`errorLiteral` fails the derivation without static Code/Detail keys);
`refuse` takes exactly one `&Error` literal, never a variable (enforced
in the same walk); `mismatchf`/`integrityFailure` build
`&Error{Code, Detail}` with no BackendID key (flag `false`, commented
at the construction site). Occurrence counting zeroes the flag with the
line so `(file, function, code, detail)` numbering is unchanged.

Fail-before / fail-after, both executed:

- Pre-fix tree + P3 plant (dead multi-line arm): census GREEN, exit 0.
- Post-fix tree + same plant: `--- FAIL: TestBackendIDEntryCensus`,
  `derived BackendID site ...|plant multiline identity#1 ... has no
  census pair and no exemption`.
- Committed `TestBackendIDSiteFilterSeesMultilineSpelling` stages a
  synthetic package copy (all production sources byte-identical + one
  multi-line-arm file) under `t.TempDir()` and requires the synthetic
  site in the denominator plus every live site still derived. It fails
  on the retired line filter (proven by mutant N-tb-sitefilter-linegated
  below, which restores exactly that spelling and is killed by this
  test alone).

## 3. N2 — entry enumeration fails closed on collision (fixed, with a test that fails before)

`deriveExportedEntries` deduped by bare name and silently absorbed a
second identity. Reproduced pre-fix: an exported method
`(*Registry).Reconcile` reaching the same arms left the census GREEN.

Fix: `collectExportedEntries` reports every second introduction of one
bare name as a collision (`deriveExportedEntries` turns it into
`t.Fatalf` naming both qualified identities and the file). Any repeat
is a collision — duplicate declarations do not compile, so a repeat
always means two entry paths behind one row, including through
unparseable (`unknown`) receivers.

Fail-before / fail-after, both executed:

- Pre-fix tree + P2 plant: census GREEN, exit 0.
- Post-fix tree + same plant: `--- FAIL: TestBackendIDEntryCensus`,
  `derived exported entries collide on one bare name, so two entry
  paths would share one census row: Reconcile: Reconcile in manifest.go
  against Registry.Reconcile in manifest.go`.
- Committed `TestExportedEntryDerivationFailsClosedOnCollision` stages
  a synthetic package copy with a function/method bare-name collision
  and requires the collision report, plus requires the live tree to
  stay collision-free (zero repeats among today's 59 bare names, so the
  gate is vacuous, never red). It fails on the retired dedupe (proven
  by mutant N-tb-entryindex-samekind below, killed by this test alone).

Both plants reverted byte-identical (`manifest.go` diff holds only the
round-3 gate; full suite green after).

## 4. N3 — free-entry reasons recorded as an open bound (deliberately left open)

The 44 `backendIDFreeEntries` reasons are unchecked prose: the mapping
check requires a non-empty reason resolving to a derived entry but
never checks content, so a free entry reaching a paired site with
unvalidated input would pass. Stated as a bound in the file's Stated
bounds (not inferred): a `go/ast` call-graph sweep from all 44 free
entries reaches zero BackendID-naming refusals (the two raw hits,
Binding and TranslateLegacyBackend, are success-value BackendID fields,
never refusals — the reviewer's own sweep, reproduced here as the
bound's evidence). Re-verified count: 44 rows in-tree.

## 5. N4 — entry arithmetic reconciled (fixed)

59 derived exported entries = 10 pair entries + 5 hostile-only + 44
free (verified by row counts in-tree; the census log line still reads
`41/41 pairs validated, 1 site exemptions plus 1 pair exemptions
pinned`). The round-3 outcome's "58 / 43" is superseded by these
numbers. LOGBOOK round-2 entry cited "rev3" where it means rev1 —
corrected in-tree.

## 6. Mutation battery (round 4 on this tree + continuity re-runs)

Harness: same verdict semantics (`KILLED` = mask exit != 0;
byte-for-byte restore verified by the harness). Full bodies in
`TASK-260906-2okwyf_battery-r4.json` + `TASK-260906-2okwyf_mutants-r4.json`;
continuity re-runs in `TASK-260906-2okwyf_battery-continuity-r4.json`.
Masks non-empty and verified by ran counts below. (One metadata-only
correction after the run: the D-tb-sitefilter-flag `expect` text
claimed the inventory mask fails too — it SURVIVES, derivation
untouched. Verdict-affecting content unchanged; the battery was re-run
on the final mutant file and the table below matches that run.)

| mutant | class | what it narrows the gate to | named test that fails | verdict / killers |
|---|---|---|---|---|
| N-tb-sitefilter-linegated: filter keyed on the arm's source line again (the retired spelling) | narrowing | admits exactly the multi-line-composite-literal subclass | `TestBackendIDSiteFilterSeesMultilineSpelling` (census 15 SURVIVED) | KILLED / behav (32 ran) |
| D-tb-sitefilter-flag: backendID flag read deleted (denominator always empty) | arm-deletion | gate absent; the filter fails itself | `TestBackendIDSiteFilterSeesMultilineSpelling` + `TestBackendIDEntryCensus` (census 15 SURVIVED — derivation untouched) | KILLED / behav |
| N-tb-entryindex-samekind: collisions reported only for receiverless pairs | narrowing | admits exactly the func-vs-method and method-vs-method subclasses | `TestExportedEntryDerivationFailsClosedOnCollision` (census 15 SURVIVED) | KILLED / behav (32 ran) |
| D-tb-entryindex-enforce: fail-closed enforcement dropped (collisions collected, ignored) | arm-deletion | gate absent; nothing observes it on this tree | — (no named failing test) | SURVIVED — bound: 0 repeats among today's 59 bare names; the P2 plant run stays GREEN under this shape and goes RED under the fixed gate (§3) |
| C-tb-multiline-deadarm: dead multi-line BackendID arm in Reconcile (condition no input satisfies) | census-only | — (no behavior change) | inventory `TestDerivedRefusalArmsAreAllDeclared` + `TestDeclaredRefusalArmsAreAllDerived` + entry-census mapping inside behav | KILLED / census (15 ran) + behav |
| X-control-notapplied | control | — | — | NOT_APPLIED (anchor occurs 0 times) |
| X-control-compilefail | control | — | — | COMPILE_FAIL (`go vet` syntax error, provider.go) |

Round-4 applied denominator: 5 applied, 4 killed, 1 survived with the
plant-evidenced bound above. NOT_APPLIED and COMPILE_FAIL are distinct
control rows outside the denominator.

Continuity re-runs on THIS tree (all anchors still apply, all still
kill — no regression from the round-4 instrument changes):

- Round 1: 9/9 killed, 0 survived (narrowing 4 incl. M-xprov-reorder
  killed by `TestBuiltinsEqualProfileProviders` alone; arm-deletion 2;
  census-only 1; audit-only 1 killed by the audit mask).
- Round 2: 5/5 killed, 0 survived (narrowing 2 — one filed narrowing in
  r2, correctly `ordering` since r3; arm-deletion 1; census-only 1;
  audit-only 1).
- Round 3: 6/6 killed, 0 survived (narrowing 1 with the len27-fail /
  len221-pass split; arm-deletion 2; ordering 1; census-only 1;
  audit-only 1 killed by the audit mask alone, 697 ran).

Cumulative production-derived denominator: 25 applied (9+5+6+5), 24
killed, 1 survived with the stated bound. Classes separate:
narrowing 9, arm-deletion 7, census-only 3, audit-only 3, ordering 1,
agreement 1 (r1's cross-package row, its own class), survivor 1.
NOT_APPLIED ×4 and COMPILE_FAIL ×4 are distinct control rows, one pair
per battery file.

## 7. Gates (real exit codes, standalone processes)

- `go test ./... -count=1`: exit 0 — 23 ok, 0 FAIL.
- `go test -race ./internal/terminalbackend/ -count=1`: exit 0.
- `go test -cover` tb/ph/pv: exit 0 (95.4% / 86.0% / 97.8%).
- `go vet ./...`: exit 0. `GOOS=windows go vet` (tb/ph/pv): exit 0.
- `go build ./...` + `GOOS=windows go build ./...`: exit 0.
- `gofmt -l internal/`: clean.
- Pre-fix plant runs (P2+P3, §2–§3): census GREEN exit 0 (the holes).
- Post-fix plant runs: census FAIL exit 1 with the two messages quoted
  in §2–§3; plants reverted byte-identical.

## 8. AC coverage

4 of 4 rows driven, each with a named test (round-1/2 rows re-verified
green on this tree; the r1–r3 continuity batteries above re-prove them
by kill, not by reading):

| AC row | production call site | driving test |
|---|---|---|
| two literals equal directly, either side alone reddens, SPEC-independent | `provider.Builtins()` / `provhost.profileProviders` | `TestBuiltinsEqualProfileProviders` (M-xprov-reorder re-killed: reorder/rename/removal each killed by this test alone per rev3; reorder re-run here) |
| §6.5 vs §7.1 asymmetry cited at both sites, or decision recorded | `terminalbackend.DigestFile`, `provider.trustCandidate` | citation pair, unchanged since rev 1, verified verbatim by the reviewer against spec@28bf96d7 |
| `ParseID` prints no unbounded refused input | `terminalbackend.ParseID` + every ID entry | `TestParseIDGrammarRefusalEchoesNothing` + census hostile batteries (re-killed by N-tb-parseid-uppercase this round) |
| every closed nit has a fail-before test; every open nit a stated bound | `Reconcile` entry, `decodeValidatedProbe`, `parseMajor`, census derivation | P2/P3 plant runs (§2–§3), `TestDecodeValidatedProbeHandsUsableMembersToRequireCapability`, `TestParseMajorLeadingZeroIsClassifiedAsForeign`; open bounds: parseMajor looseness, named-rune blind spot, N3 free-entry reasons |

Round-1/2/3 outcome and battery resources stand for the unchanged
regions; all named pins re-ran green on this tree (see §7).
