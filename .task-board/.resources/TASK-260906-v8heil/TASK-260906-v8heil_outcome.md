# TASK-260906-v8heil — unify inventory machinery on executed witnesses: outcome

Status: ready for review.

Candidate tree: `dd555ca42507259365151126fa042f113aef2441`
(verified by `git ls-tree -r`: `internal/invcore/invcore.go`,
`internal/invcore/must.go`, `internal/invcore/invcore_test.go`, and
`internal/terminalbackend/refusal_arm_witnesses_test.go` are in-tree, and
`git hash-object` of those files plus
`internal/provider/refusal_inventory_test.go` byte-for-byte equals the
tree blobs. Built via a detached index — `GIT_INDEX_FILE` copy +
`read-tree HEAD` + `add -A` + `write-tree` — so the worktree index is
untouched. HEAD stays at the `5da63ad` checkpoint; the change is
uncommitted. The Change Request record is the orchestrator's
integration step; this OID is the candidate it must carry.)

Scope: 12 modified `*_test.go` files plus the logbook entry (zero
production edits — every `M` path ends in `_test.go` or is `LOGBOOK.md`),
one new test-support package
`internal/invcore` (`invcore.go`, `must.go`, `invcore_test.go`), one new
witness file `internal/terminalbackend/refusal_arm_witnesses_test.go`,
plus a committed pin `TestQualifiedWatchesAdmitsPrefixSpellings` in the
core suite. The review's "20 walkers" count predates this base (42 test
files import `go/parser` at `5da63ad`); the in-scope twenty are the
refusal-arm inventory walkers of terminalbackend, provhost, and
provider — every inventory with arms, witnesses, and floors. No
production file is touched, so production-derived positions are
identical before and after by construction, and the census below
confirms it.

## 1. Shared core owns selection, parsing, harness

`internal/invcore` carries the production-file glob
(`ScanProduction`/`MustScanProduction`: non-test `.go`, sorted,
fail-closed on zero files and on any unparseable file), fail-closed
single-parse (`ParseBytes`/`ParseSource`), the both-direction harness
(`DiffSets`: unregistered site and orphan row fail together), the
alias-bypass union (`ConstructorSpec` with import-path qualified
matching plus the `Errorf*` prefix stem, `MustAuditConstructorReferences`
over direct-call positions, `WatchedCallPositions` attribution), and the
runtime direction (`SiteRecorder` with `AuditSites`, including the
closed code-set pin). Every refusal-arm derivation in the three
packages runs through it; the two documented exceptions keep their
selection local and route parsing through `ParseBytes`: the provhost
cross-tree attestation (recursion spans `internal`+`cmd`, the core
selects single packages) and the digit-guard leaf-test-name collector
(the core selects production files only).

## 2. terminalbackend on executed witnesses

`TestEveryDeclaredArmRefusesAtItsEntry` executes every declared arm at
its production entry point (constructor, `Registry.Resolve`,
`ParseProviderDescriptor`, manifest/conformance/descriptor gates) and
requires the refusal to come from that arm:
`refusal-arm witnesses: 184/210 derived arms executed`. The remaining
26 are stated-bound rows (unreachable vocabularies, decoder-contract
sites, canonical-plumbing totals, defensive-reparse bounds, shadowed
lookups), each pinned by a named passing test
(`TestDefensiveBoundsAreExactlyThese`,
`TestCheckTransitionOperationVocabularyIsUnreachable`,
`TestDecodeCappedValueDefensiveSitesAreUnreachable`,
`TestShadowedLookupsHaveNoInput`,
`TestCanonicalPlumbingIsTotalOnDecodedMaps`). Textual resolution is
gone: no row passes by mentioning a detail string.

## 3. Alias-bypass union preserved, per shape

| Bypass shape | Package direction | Negative test | Direction-drop mutant | Killed by |
|---|---|---|---|---|
| Local var binding (`refuse := mismatchf`) | tb outright refusal (core audit) | `TestAuditRejectsLocalAlias` | `N-core-local-alias` | census (1 ran) |
| Import alias (`errs.New` binding) | provider runtime + core path resolution | `TestAuditRejectsImportAlias` | `N-core-import-alias` | census (1 ran) |
| Aliased direct call (`errs.New(…)`) | allowed, attributed | `TestAuditAttributesAliasedDirectCall` | — (positive pin, stays green under all core mutants) | — |
| Dot import of watched path | fail closed | `TestAuditFailsClosedOnDotImport` | `N-core-dot-import` | census (1 ran) |
| `Errorf*` prefix stem | tb watch list union | `TestQualifiedWatchesAdmitsPrefixSpellings` (new) | `N-core-errorf-prefix` | census (1 ran) |
| Constructor var binding (provider `failXxx`) | runtime `SiteRecorder` | `TestAuditRejectsVarBinding` + `TestRecorderAttributesProductionFrame` | `N-pv-ctor` | full-suite audit |
| Constructor as argument | fail closed | `TestAuditRejectsConstructorAsArgument` | covered by audit branch | core suite green |

Every direction-drop mutant is killed by its shape's negative while the
downstream behav mask (2 tb both-direction tests) stays green.

## 4. Fail-closed controls (all committed, all green)

`TestDiffSetsFailsBothDirections`, `TestDiffSetsFailsClosedOnEmptyDerivation`,
`TestScanProductionFailsClosedOnZeroFiles`,
`TestScanProductionFailsClosedOnUnparseableFile`,
`TestScanProductionSkipsTestFiles`, `TestRecorderAuditFailsClosedOnBlindScan`,
`TestRecorderAuditPinsTheCodeSet` — unregistered site, orphan row, and
unclassifiable site each fail, control-planted including an import alias
and a var binding.

## 5. Arm-count census before/after (probes on `5da63ad` shadow vs worktree)

| Package | Before (base) | After (worktree) | Delta |
|---|---|---|---|
| terminalbackend derived arms | 210 | 210 | 0 |
| terminalbackend executed at entry | — (textual) | 184 + 26 stated-bound | ported, total unchanged |
| provhost derived arms / files | 167 / 15 | 167 / 15 | 0 |
| provhost witnessed | 167/167 | 167/167 | 0 |
| provider scanned files / sites / strays / raw | 6 / 18 / 0 / 0 | 6 / 18 / 0 / 0 | 0 |
| provider closed code set | 3 | 3 | 0 |

No package regresses its floor. (The review's "floor 162" for provhost
is stale — the base already derives 167; both trees agree.)

## 6. Digit-guard N-R1 closed

Code-point spellings are ENUMERATED: `digitGuardDigitRune` admits
`token.INT` 48..57 in every base, pinned by
`TestDigitGuardDigitRuneAdmitsCodePointSpellings` (12 admitted
spellings incl. hex/octal/legacy-octal/underscore/char/hex-char/unicode
forms; 9 rejected adjacents and non-literals). Named-rune constants and
`strconv`-delegated admission are explicitly OUTSIDE the classifier (an
identifier is not a literal; a call is not a comparison), with no such
site in either package and a fail-closed rule for everything the
classifier can see. Stated in the census header bound block.

## 7. Mutation battery: 15/15 killed over applied

Harness `.temp/TASK-260906-v8heil/mutate2.py`, mutants
`muts_v8heil.json`, raw records `battery_v8heil.json`. Denominator is
production-derived: the 395-site derived domain (210 tb + 167 ph + 18
pv) plus the core gate branches. Each mutant runs a census mask and a
behavioural mask separately (both `-v`, ran-counts recorded; every mask
below ran ≥1 test).

| Mutant | Class | Census (ran) | Behav (ran) | Killers |
|---|---|---|---|---|
| `N-core-local-alias` | narrowing | KILLED (1) | pass (2) | census |
| `N-core-import-alias` | narrowing | KILLED (1) | pass (2) | census |
| `N-core-dot-import` | narrowing | KILLED (1) | pass (2) | census |
| `N-core-errorf-prefix` | narrowing | KILLED (1) | pass (2) | census |
| `D-tb-resolve` | arm-deletion | KILLED (2) | KILLED (185) | both |
| `N-tb-ctor` | narrowing | KILLED (2) | pass (185) | census |
| `N-digit-int-spelling` | narrowing | KILLED (1) | pass (2) | census |
| `C-tb-dead-arm` | census-only | KILLED (2) | pass (185) | census |
| `T-guard-accumulator` | token-preserving | pass (1) | KILLED (2) | behav |
| `D-ph-unknown-operation` | arm-deletion | KILLED (2) | KILLED (172) | both |
| `N-ph-ctor` | narrowing | KILLED (2) | pass (172) | census |
| `C-ph-dead-arm` | census-only | KILLED (2) | pass (172) | census |
| `D-pv-malformed-name` | arm-deletion | full-suite FAIL* (102) | KILLED (1) | behav* |
| `N-pv-ctor` | narrowing | KILLED via TestMain audit (102) | pass (49) | census |
| `C-pv-dead-arm` | census-only | KILLED via TestMain audit (102) | pass (49) | census |
| `X-control-notapplied` | control | NOT_APPLIED (anchor occurs 0 times) | | |
| `X-control-compilefail` | control | COMPILE_FAIL (vet rejects before tests run) | | |

Summary rows: KILLED 15, SURVIVED 0, NOT_APPLIED 1, COMPILE_FAIL 1.

`*D-pv` footnote (verified by a manual re-run under the mutant): the
full-suite census mask fails only through the embedded
`TestDiscoverRefusesMalformedNames` witness; the TestMain audit itself
stays green. Provider's recorder fires at construction time, so
swallowing a built refusal downstream (`return` → `continue`) is
invisible to the census — the witness is the only net for that shape.
This is the documented runtime-audit asymmetry (execution-recorded, not
outcome-recorded), not a regression: the mutant is killed, and the
split shows exactly which direction holds it.

Behavioural masks are non-empty everywhere (min 1, max 185 ran) and the
split covers all four shapes: census-only (9), behav-only (`T-guard`,
`D-pv` audit-corrected), both (`D-tb`, `D-ph`), plus live
NOT_APPLIED/COMPILE_FAIL control rows. The `T-guard` mutant preserves
every searched-for digit token (same comparisons, same site key — the
digit census passes) while shifting geometry values so the bound arm
rejects valid `1000`; the behavioural geometry tests kill it. Two
harness bugs were caught and fixed during the run (behav masks first
ran in the mutant's package with 0 tests; provider's scoped `-run`
silently skipped its TestMain audit) — the rerun records above are the
corrected ones.

## 8. Production-entry coverage ratio

Refusals named in the AC driven through production entry points by
named committed tests: tb `TestEveryDeclaredArmRefusesAtItsEntry`
(entries: `Registry.Resolve`, `Registration.validate`,
`validatePlatforms`, `CheckProviderDescriptor`,
`ParseProviderDescriptor`, manifest/conformance/descriptor gates),
provhost `TestEveryArmWitnessRefusesAtTheProductionEntry` (entries:
`IdempotencyKeyFor`, frame/parse/ctor/integrity gates),
provider TestMain audit over `Discover`/`Verify`/lift tests plus the
closed code-set pin (`codeIntegrityFailure`, `codeInvalidConfig`,
`codeLocalPrecondition`).

369 of 395 derived refusal sites driven (184 tb + 167 ph + 18 pv); the
26 undriven tb sites are stated-bound unreachable/defensive rows with
named pins (§2), not silent admits. Ratio: **369/395 driven, 26/395
stated-bound**.

## 9. Gates (CI parity, this turn)

- `go vet ./...`: clean
- `GOOS=windows go vet ./...`: clean
- `gofmt -l internal`: empty
- `go build ./...`: clean
- `go test ./... -count=1`: exit 0, 23 ok, 0 FAIL
- `go test -race ./... -count=1`: exit 0, 23 ok
- `go test ./... -cover -count=1`: exit 0 (invcore 71.0%, provhost 86.0%, provider 97.8%, terminalbackend 94.9%)
- `go run ./internal/traceability/cmd/tracecheck`: ok
- `go generate ./internal/catalog` + `git diff --exit-code`: current
- cigate pinned contract gates: pass
- curator: not applicable (no `Skillfile.json` changes, no skills installed)
