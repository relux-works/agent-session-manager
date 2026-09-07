# TASK-260906-2okwyf round 3 — F2 instrument: outcome

Status: ready for review (handed off to review; review/integration are orchestrator steps).

Candidate tree: `34b88fb9bb6e9fe050e400d6b51685e9cbd6547b`
(verified: detached index — `GIT_INDEX_FILE` copy + `read-tree HEAD` +
`add -A` + `write-tree` — worktree index untouched, HEAD stays at the
`114a056` checkpoint, change uncommitted. All 5 key paths in-tree with
`git hash-object` byte-equal to the tree blob, 5/5 MATCH, including the
two untracked test files. Computed AFTER every tracked write, including
the LOGBOOK append. The Change Request record is the orchestrator's
integration step; this OID is the candidate it must carry.)

Round-3 scope (3 code/test paths + LOGBOOK; round-1/round-2 files untouched):

- `internal/terminalbackend/manifest.go` — `Reconcile` validates both
  `manifest.TerminalBackendID` (manifest.go:1696) and
  `probe.TerminalBackendID` (manifest.go:1699) through `ParseID` before
  any check runs.
- `internal/terminalbackend/terminalbackend.go` — `ParseID` comment
  rewritten (false package-wide universal replaced by a pointer to the
  instrument; terminalbackend.go:163).
- `internal/terminalbackend/backend_id_entry_census_test.go` — NEW: the
  (site, reaching exported entry) census instrument (TestBackendIDEntryCensus).
- `LOGBOOK.md` — this round's entry (newest first).

## 1. F2 — Reconcile entry gate (fixed, not named)

Pre-fix reproduction, driven through the production entry point
`terminalbackend.Reconcile` with `200x"A" + ESC[31m + ../../etc/passwd`
(221 bytes, grammar- and bound-refused) as equal hostile IDs:

- differing digests → `manifest.go` substitution refusal, 305 bytes,
  `contains(hostile)==true`, `BackendID` carries the full string;
- matching digests, foreign generation digest → generation refusal,
  313 bytes, `contains(hostile)==true`.

Both are the defect A9 named at `ParseID`, live at the sibling that
never calls it. Fix: entry validation through `ParseID` for both IDs
(the pattern `CheckVersionTuple` and `CheckProviderDescriptor` already
use). A refused ID returns the `ParseID` refusal — whose bound and
grammar arms carry no `BackendID` — and never reaches the naming arms.

- Red-before (unfixed code): new census `reconcile` battery FAILs —
  both hostile shapes at both arms (305/313B echoes), plus the short
  shape at 111/119B.
- Green-after: `TestBackendIDEntryCensus` PASS, exit 0, log line
  `backend-id entry census: 41/41 pairs validated, 1 site exemptions
  plus 1 pair exemptions pinned`.

## 2. F2 — the instrument (what closes the class)

`TestBackendIDEntryCensus` (internal package: reuses `deriveRefusalArms`
filtered to `BackendID:` constructions, `invcore.MustScanProduction`
for entry coverage, and the shared runtime refusal recorder for fire
attribution — no new file walker):

- Denominator derived, not listed: 31 BackendID sites (30 refused
  through pairs + 1 exempt) and every exported entry (58 derived:
  10 pair entries + 5 hostile-only + 43 free with reasons).
- 41 (site, entry) pairs. A site reachable through two entries appears
  twice — collapsing those rows was the bug. Every pair states its
  guard; every pair is proved twice: hostile battery through its entry
  (2 shapes, `BackendID == ""` and `!Contains(Error(), hostile)`) and
  a validated fire (12 own fires with recorder line-attribution + 29
  inventoried-witness citations resolved mechanically against
  `DeclaredArmIdentities`: row present, same entry, not bound).
- Exemptions executed, not asserted: digest-null site (direct
  `validate()` pin in-file) and the shadowed digest-parse/Admit pair
  (malformed digest refuses at the parse arm — the shadow proof).
- Ratio produced by the test (see log line above), both directions
  checked (unregistered site, orphan row, unlisted entry all fail).

Control plant C1 (reviewer's ready-made vector: ParseID guard deleted
from `CheckProviderDescriptor`): the new test FAILs with the exact
308-byte echo while the rest of the package suite stays green
(`--- FAIL: TestBackendIDEntryCensus` is the only failure) —
reproducing the finding that the old suite holds none of this class.

## 3. Non-blocking notes closed

- N1 (anchors never re-derived): all anchors above re-derived on the
  final tree after the last artifact write — Reconcile guards
  manifest.go:1696/1699, naming sites :1741/:1777/:2099/:2102,
  descriptor guard terminalbackend.go:646, ParseID census note :163.
- N2 (ordering filed as narrowing): `R-tb-checktuple-order` is filed
  `ordering` in this round's battery (class list below); the rev-1/2
  mislabel is superseded, not repeated.

## 4. Mutation battery (round 3, this tree)

Harness: same verdict semantics (`KILLED` = mask exit != 0;
byte-for-byte restore verified by the harness, tree verified clean
after). Masks non-empty and verified by `-list` semantics (ran counts
below): inventory-census 15 ran, behav 273 ran (r2's 11 + the new
census test), audit full-package 695 ran. Full bodies in
`TASK-260906-2okwyf_battery-r3.json` + `TASK-260906-2okwyf_mutants-r3.json`.

| mutant | class | what it narrows the gate to | named test that fails | verdict / killers |
|---|---|---|---|---|
| N-tb-reconcile-lengthgated: entry checks run only when `len > maxIDBytes` | narrowing | admits exactly the grammar-refused-but-bounded subclass (len27 echoes, len221 still refused) | `TestBackendIDEntryCensus/reconcile/len27` (len221 rows pass — narrowing proven off the subtests) | KILLED / behav (census 15 SURVIVED) |
| D-tb-reconcile-entry: both entry blocks deleted | arm-deletion | gate absent; all hostile IDs echo at the naming arms | `TestBackendIDEntryCensus/reconcile` (both shapes, both arms) | KILLED / behav (census SURVIVED — entry calls are not refuse lines) |
| D-tb-checkdesc-entry: entry block deleted (= C1) | arm-deletion | gate absent; hostile descriptor IDs echo (308B) | `TestBackendIDEntryCensus/check-provider-descriptor` | KILLED / behav (census SURVIVED; rest of package green) |
| R-tb-checktuple-order: entry check moved after the version arms | ordering | validation present but bypassed on every version-failure path | `TestBackendIDEntryCensus/check-version-tuple` + `TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho` | KILLED / behav (census SURVIVED — same lines, moved) |
| C-tb-reconcile-deadarm: unreachable new BackendID arm in Reconcile (space can never pass the entry) | census-only | — (no behavior change) | inventory `TestDerivedRefusalArmsAreAllDeclared` + entry-census mapping inside behav | KILLED / census + behav-mapping (no behavioral probe observes it) |
| S-tb-statusresult-shadow: occurrence-1 condition widened to subsume occurrence-2 inputs (lines unchanged) | audit-only | — (occurrence-2 witness proves its pair through the sibling) | full-run audit naming `conformance.go:765` | KILLED / audit only (census 15 + behav 273 SURVIVED) |
| X-control-notapplied | control | — | — | NOT_APPLIED (anchor occurs 0 times) |
| X-control-compilefail | control | — | — | COMPILE_FAIL (`go vet` syntax error, provider.go) |

Killed over applied: 6/6 killed, 0 survived. No surviving mutant, so no
survival bounds to state. NOT_APPLIED and COMPILE_FAIL are distinct
control rows, not in the denominator.

## 5. Gates (real exit codes, standalone processes)

- `go test ./internal/terminalbackend/ -run TestBackendIDEntryCensus`
  pre-fix: FAIL (reconcile battery, 305/313B echoes).
- `go test ./... -count=1`: exit 0 — 23 ok, 0 FAIL.
- `go test -race ./internal/terminalbackend/ -count=1`: exit 0.
- `go test -cover` tb/ph/pv: exit 0 (95.4% / 86.0% / 97.8%).
- `go vet ./...`: exit 0. `gofmt -l internal/`: clean.
- `GOOS=windows go build ./...`: exit 0.

## 6. AC coverage

4 of 4 rows driven, each with a named test (round-1/2 rows re-verified
green this round; F2 is the previously-missing entry-dimension row):

| AC row | production call site | driving test |
|---|---|---|
| two literals equal directly, either side alone reddens, SPEC-independent | `provider.Builtins()` / `provhost.profileProviders` | `TestBuiltinsEqualProfileProviders` (re-ran PASS) |
| §6.5 vs §7.1 asymmetry cited at both sites, or decision recorded | `terminalbackend.DigestFile`, `provider.trustCandidate` | citation pair, unchanged since rev 1 |
| `ParseID` prints no unbounded refused input | `terminalbackend.ParseID` + every ID entry | `TestParseIDGrammarRefusalEchoesNothing` + census hostile batteries (15 entries × 2 shapes) |
| every closed nit has a fail-before test; every open nit a stated bound | `Reconcile` entry, `decodeValidatedProbe`, `parseMajor` | census (red-before above), `TestDecodeValidatedProbeHandsUsableMembersToRequireCapability`, `TestParseMajorLeadingZeroIsClassifiedAsForeign` (all re-ran PASS) |

Round-1/round-2 outcome and battery resources stand for the unchanged
regions; all named pins re-ran green on this tree (see §5).
