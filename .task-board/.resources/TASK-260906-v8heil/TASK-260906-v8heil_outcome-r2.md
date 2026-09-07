# TASK-260906-v8heil round 2 — unify inventory machinery on executed witnesses: outcome

Status: ready for review (handoff to `to-review`).

Round 1 built the shared core and ported terminalbackend to executed
witnesses; review rev2 (RUN-260907-e3b260) returned CHANGES REQUESTED
with three blocking findings (F1/F2/F3) and three minor ones (F4/F5/F6)
plus a scope-reporting correction. This round fixes all six against the
same tree, with no narrowing of the in-scope set.

Candidate tree: `9d7eaa4d2c4e075e235ef708ea3b175f0ecc4eed`
(verified: detached `GIT_INDEX_FILE` + `read-tree HEAD` + `add -A` +
`write-tree`; `git ls-tree -r` carries `internal/invcore/{invcore,must,
invcore_test}.go`,
`internal/terminalbackend/{refusal_arm_witnesses_test,refusal_site_audit_test}.go`;
`git hash-object` of every new/changed file byte-for-byte equals the
tree blobs — 11/11 MATCH. HEAD stays at the `5da63ad` checkpoint; the
change is uncommitted. The Change Request record is the orchestrator's
integration step; this OID is the candidate it must carry.)

Scope (F6 corrected — round 1's "zero production edits" sentence was
false the moment this round landed, and is withdrawn, not amended):
4 production files carry the site-recording seam
(`internal/terminalbackend/{terminalbackend,manifest,conformance,
descriptor}.go`: one hook var + `refuse` funnel + hook calls in
`mismatchf`/`integrityFailure` + 89 `&Error{}` literals routed through
`refuse`, all single-line, behavior-preserving by construction and by
the full suite below), 1 new internal test file
(`refusal_site_audit_test.go`), 3 rewritten test sections (inventory
derivation lines, shadow cover analysis, digit-guard N-R1 bound),
`internal/invcore/invcore.go` minus the dead loop. By file:
12 of 42 `go/parser`-importing test files ported through `invcore`
(42 at base → 30 now, method: `grep -rl '"go/parser"' --include=
"*_test.go" internal/`); by package: the refusal-arm inventories of
3 packages are expressed through the core (terminalbackend, provhost,
provider). The other 9 packages with hand-rolled walkers
(`canonicaljson`, `cliresult`, `config`, `dirnode`, `environ`,
`localstore`, `secprim`, `sessadapter`, `specdoc`) keep them for
non-refusal censuses (grammar, vocabulary, identity) and are out of
scope: the task names exactly the three refusal-arm architectures and
asks for the union of those.

## 1. F1/F2 closed: terminalbackend has the runtime site direction

Production seam (`terminalbackend.go`, next to the `Error` type):

- `var recordRefusal func(code, detail string)` — nil in production
  (construction is then a pure allocation; zero behavior change),
  armed by the internal `TestMain` for the test-binary run.
- `func refuse(err *Error) *Error` (`//go:noinline`) — every
  production `&Error` literal routes through it; returns the wrapped
  value unchanged. `mismatchf`/`integrityFailure` (`//go:noinline`)
  invoke the hook the same way.
- The audit attributes the site through the call stack
  (`Record(code, 2)`, the provider/provhost skip convention, pinned
  empirically by `TestRefusalSiteAttributionIsExact`, not by reading).

Derivation (`refusal_arm_inventory_test.go`): every arm carries its
construction line (refuse-call line for literals, constructor-call line
for funnels); an `&Error` literal outside `refuse`, or a `refuse`
wrapping a non-literal, fails the derivation; two arms sharing one line
fail the derivation (one line attributes one arm); `refuse` joins the
alias-audit watch list. Audit (`refusal_site_audit_test.go`,
`TestMain` post-`m.Run`, full runs only): every derived non-bound line
must have fired (`210 derived lines, 26 bound-exempt, 14 wire codes`
in the log line), every fired line must be derived, observed wire codes
must equal the closed set.

P8 (the reviewer's plant, reproduced byte-for-byte: CheckEntrypoint's
first session parse widened with `|| sessionID != argv[2]`) now reads:

- census (bijection, 2 ran): SURVIVED — derived set byte-identical.
- behav (witnesses, 185 ran): SURVIVED — all three proves pass through
  the sibling.
- full-suite audit (657 ran): KILLED —
  `refusal-site audit: derived refusal sites without an exercised path
  (shadowed dead or unwitnessed): conformance.go:713,
  conformance.go:716` (battery row `S-tb-shadow`, killers `["audit"]`).

Residual after the fix: the reviewer's 62-of-184 clause-shared count is
recomputed identically on this tree (20 shared clauses, largest
`CodeMismatch/"document_member_type"` x11) and the death residual is
0 of 184 — every shared-clause arm is site-attributed by the audit.
Stated remainder (disclosed, not closed): the audit binds every ROW to
a fired LINE, not every WITNESS to its row's line; a complementary
two-witness swap would stay green. No such swap exists, the shape is
perverse rather than adjacent, and provider/provhost share it by
construction (their witnesses bind codes, their audits bind lines).

The audit also caught a REAL pre-existing sibling-swallow on its first
run: the `descriptor member set` #2 witness deleted a member (shorter
document), so the count arm #1 fired first and `descriptor.go:123`
never executed anywhere in the suite. Fixed as a witness, not a bound
(rename-away keeps the count, fires the loop), with the why in the
prove comment. `C-tb-dead-arm` no longer counts as dead-arm coverage on
its own: it is kept as a bijection row with both killers named
(census + full-suite), and `S-tb-shadow` is the dead-arm evidence.

## 2. F3 corrected: the strconv site exists and is pinned

`digit_guard_census_test.go` no longer claims no strconv-delegated site
exists. It names `internal/provhost/opdecode.go` `rawUint53`
(`strconv.ParseUint(literal, 10, 64)` + error branch + `parsed >
maxUint53`) as OUTSIDE the comparison classifier but behaviorally
pinned by `TestDecodeQuiesceRefusals/count_overflow`, reproduced as
battery row `B-provhost-strconv-bound` (bound narrowed to
`maxUint53+1`, admitting exactly 2^53: census SURVIVED ran 1,
behav KILLED ran 20 naming `count_overflow`). Code-point spellings stay
ENUMERATED with their pin; named-rune constants stay OUTSIDE with the
no-such-site claim kept (a named-constant gate classifies as other and
fails as unclassifiable, not silently).

## 3. F4 closed: the shadow pin derives from the rows

`TestShadowedLookupsHaveNoInput` no longer hand-lists 19 pairs. It
derives every `mismatchf("document members")` miss (11: file, function,
call line, map var, fixed/param lookup), every `checkExactMembers`
call (5: function, map var, position, resolved list — package var or
inline literal), and every helper call site, then classifies each miss
covered (bound) or uncovered (witnessed): same-function preceding
exact check on the same map variable with the member in the list, else
coverage through every production caller recursively (parsePlatformList
and parseRealmMembers pass through; parseRealmProvider nests two
deep). Cover is per (call site, member) lookup, not per member. Both
directions hold against the rows: covered ⟺ `boundShadowedLookup`
(9 covered), uncovered ⟺ witnessed (checkExactMembers' own two),
miss set == derived-arm set (11 == 11), covered count == bound-row
count (9 == 9). Log line: `shadowed lookups: 9 covered (bound),
2 uncovered (witnessed)`.

## 4. F5 closed: dead code gone, ParseSource used

`invcore.go`'s vestigial `spec.Qualified` loop (`_ = hasLocal`) is
deleted. `ParseSource` has three callers
(`TestRefuseFunnelRejectsAliases` × 2 shapes plus the existing
`invcore_test.go` control), so the "zero callers" clause is moot.

## 5. Alias-bypass union, per shape (round-2 delta in the last row)

| Bypass shape | Package direction | Negative test | Direction-drop mutant | Killed by |
|---|---|---|---|---|
| Local var binding (`refuse := mismatchf`) | tb outright refusal (core audit) | `TestAuditRejectsLocalAlias` (core) | `N-core-local-alias` | census (1 ran) |
| Import alias (`errs.New` binding) | provider runtime + core path resolution | `TestAuditRejectsImportAlias` (core) | `N-core-import-alias` | census (1 ran) |
| Aliased direct call (`errs.New(…)`) | allowed, attributed | `TestAuditAttributesAliasedDirectCall` | — (positive pin) | — |
| Dot import of watched path | fail closed | `TestAuditFailsClosedOnDotImport` | `N-core-dot-import` | census (1 ran) |
| `Errorf*` prefix stem | tb watch list union | `TestQualifiedWatchesAdmitsPrefixSpellings` | `N-core-errorf-prefix` | census (1 ran) |
| Constructor var binding (provider `failXxx`) | runtime `SiteRecorder` | `TestAuditRejectsVarBinding` + `TestRecorderAttributesProductionFrame` | `N-pv-ctor` | full-suite audit (102 ran) |
| Constructor as argument | fail closed | `TestAuditRejectsConstructorAsArgument` | covered by audit branch | core suite green |
| `refuse` funnel binding (NEW: `r := refuse`, P1 `var aliasedMismatch = mismatchf`, P2 `import errs "errors"` + `var mintPlain = errs.New`) | tb outright refusal (core audit over the extended spec) | `TestRefuseFunnelRejectsAliases` (tb, synthetic plants via `ParseSource`) | `N-tb-refuse` | census (1 ran), behav green (185) |

P1/P2 reviewer plants were additionally re-run by hand against the new
tree during development (both redden four tb tests through the `refuse`
and `mismatchf` spec entries); the committed synthetic equivalents
above are the durable pins.

## 6. Arm-count census before/after

| Package | Base `5da63ad` | Worktree | Delta |
|---|---|---|---|
| terminalbackend derived arms / lines | 210 | 210 / 210 lines (26 bound-exempt, 14 wire codes) | 0 |
| terminalbackend executed at entry | 184 pair-proves | 184 pair-proves + 184/184 lines exercised (audit) | attribution added, total unchanged |
| provhost derived arms / files | 167 / 15 | 167 / 15 | 0 |
| provhost witnessed | 167/167 | 167/167 | 0 |
| provider sites / strays / raw | 18 / 0 / 0 | 18 / 0 / 0 (untouched in round 2, suite green) | 0 |
| provider closed code set | 3 | 3 | 0 |

No package regresses its floor. Production-error values are unchanged
by the seam: the 184 pair-proves assert exact wire codes and details,
and the full suite (which asserts hundreds of refusal strings) is
green.

## 7. Mutation battery: 19/19 killed over applied (denominator 395)

Harness `.temp/TASK-260906-v8heil/mutate2.py` (round-2: optional
third `audit_*` full-package mask; evidence filter also keeps
`refusal-site audit` lines). Mutants `muts_{tb_r2,ph_r2,core,pv,ph,x}
.json`, raw records `battery_{tb_r2,ph_strconv,core_r2,pv_r2,ph_r2,
x_r2}.json`. Denominator is production-derived: 210 tb + 167 ph +
18 pv = 395 refusal sites plus the core gate branches. Every mask ran
>= 1 test (minimum ran 1, no empty mask).

| Mutant | Class | Census (ran) | Behav (ran) | Audit = full run (ran) | Killers |
|---|---|---|---|---|---|
| `N-core-local-alias` | narrowing | KILLED (1) | pass (2) | — | census |
| `N-core-import-alias` | narrowing | KILLED (1) | pass (2) | — | census |
| `N-core-dot-import` | narrowing | KILLED (1) | pass (2) | — | census |
| `N-core-errorf-prefix` | narrowing | KILLED (1) | pass (2) | — | census |
| `D-tb-resolve` | arm-deletion | KILLED (2) | KILLED (185) | — | both |
| `N-tb-ctor` | narrowing | KILLED (2) | pass (185) | — | census |
| `N-tb-refuse` | narrowing | KILLED (1) `TestRefuseFunnelRejectsAliases` | pass (185) | — | census |
| `N-shadow-varname` | narrowing | KILLED (1) `TestShadowedLookupsHaveNoInput` | pass (185) | — | census |
| `N-digit-int-spelling` | narrowing | KILLED (1) | pass (2) | — | census |
| `T-guard-accumulator` | token-preserving | pass (1) | KILLED (2) | — | behav |
| `C-tb-dead-arm` | census-only | KILLED (2) bijection | pass (185) | KILLED (657) full-suite | census + full |
| `S-tb-shadow` | shadow | pass (2) derivation identical | pass (185) siblings refuse | KILLED (657) naming `conformance.go:713, :716` | audit ONLY |
| `D-ph-unknown-operation` | arm-deletion | KILLED (2) | KILLED (172) | — | both |
| `N-ph-ctor` | narrowing | KILLED (2) | pass (172) | — | census |
| `C-ph-dead-arm` | census-only | KILLED (2) | pass (172) | — | census |
| `B-provhost-strconv-bound` | narrowing | pass (1) cross-package digit census | KILLED (20) `count_overflow` | — | behav |
| `D-pv-malformed-name` | arm-deletion | KILLED (102) | KILLED (1) | — | both |
| `N-pv-ctor` | narrowing | KILLED via TestMain audit (102) | pass (49) | — | census |
| `C-pv-dead-arm` | census-only | KILLED via TestMain audit (102) | pass (49) | — | census |
| `X-control-notapplied` | control | NOT_APPLIED (anchor occurs 0 times) | | | — |
| `X-control-compilefail` | control | COMPILE_FAIL (vet rejects before tests run) | | | — |

Summary rows: KILLED 19, SURVIVED 0, NOT_APPLIED 1, COMPILE_FAIL 1.
Applied (production- or gate-derived) denominator: 19 mutants over the
395-site derived domain — 19/19 killed. Classes separate: narrowing 11,
arm-deletion 3, census-only 3, token-preserving 1, shadow 1.

Per-arm behavioural-versus-census split: every row ran both masks
non-empty (census min 1, behav min 1; tb behav masks run the full 185
witnesses, ph 172, pv 49/1). Narrowing rows hold behav green while
census reddens (gate weakened, production intact — the
delete-is-not-enough direction); `T-guard-accumulator` and
`B-provhost-strconv-bound` hold census green while behav reddens (the
token-preserving/behavioural direction the static checker cannot see);
`D-` rows redden both; `S-tb-shadow` reddens neither static mask and
only the exercised-path audit. No surviving mutant: no survival bound
to state. `D-pv` footnote (round 1, still true, battery re-ran green):
the full-suite census mask fails only through the embedded
`TestDiscoverRefusesMalformedNames` witness; the TestMain audit itself
stays green — execution-recorded, not outcome-recorded (documented
asymmetry, not a regression).

## 8. AC coverage ratio

AC rows driven through production entry points by named committed
tests, 4 of 4:

- Shared core (selection/parsing/harness): `internal/invcore`
  suite (17 tests incl. alias/unclassifiable control plants) +
  all three inventories' both-direction tests. Call sites:
  `ScanProduction`, `ParseBytes`/`ParseSource`, `DiffSets`,
  `AuditConstructorReferences`/`WatchedCallPositions`,
  `SiteRecorder.Record`/`AuditSites`.
- Executed witnesses: `TestEveryDeclaredArmRefusesAtItsEntry`
  (184 subtests over `CheckEntrypoint`, `CheckTransition`,
  `Registry.Resolve`/`RegisterExternal`, `CheckProviderDescriptor`,
  `ParseProviderDescriptor`, manifest/conformance/descriptor gates) +
  the TestMain exercised-site audit + `TestDerivedSiteLinesAreExactlyRowed`.
- Alias bypass per shape: table in §5 (8 shapes, each with a named
  failing test and a direction-drop mutant).
- Arm counts: §6 (log lines `refusal-site audit domain: 210 derived
  lines…`, `refusal arm coverage domain: 167 derived arms…`).

Refusal-level ratio: **369 of 395 derived sites driven** (184 tb +
167 ph + 18 pv); the 26 undriven tb sites are stated-bound rows with
named pins (§1), not silent admits. tb's 184 are now audit-proven
exercised lines, not merely passing witnesses.

## 9. Gates (this turn, real exit codes)

- `go build ./...`: exit 0.
- `go vet ./...`: exit 0. `GOOS=windows go vet ./...`: exit 0.
- `gofmt -l internal/`: empty.
- `go test ./... -count=1`: exit 0, 23 ok, 0 FAIL (incl.
  `internal/cigate` contract gates).
- `go test -race ./... -count=1`: exit 0, 23 ok (hook-swap test is
  sequential by design; no data race).
- `go test ./... -cover -count=1` on the four pkgs: exit 0
  (invcore 70.2%, provhost 86.0%, provider 97.8%,
  terminalbackend 95.0%).
- `go run ./internal/traceability/cmd/tracecheck`: ok
  (`contracts=60 … clauses_discharged=17/428`).
- `go generate ./internal/catalog` + `git status`: no diff.
- curator: not applicable (no `Skillfile.json` changes).

No gate was expected-red; nothing is reported as passing that did not
exit 0. Battery KILLED rows are the intentional reds, each with its
real failing output in the raw records.

## 10. Files changed (worktree vs `5da63ad`)

Production (seam only, §1): `internal/terminalbackend/
terminalbackend.go` (hook + `refuse`), `manifest.go` (funnel hooks),
`conformance.go` + `descriptor.go` (89 `refuse(` wraps, same line).
Tests: `internal/invcore/invcore.go` (dead loop deleted),
`internal/terminalbackend/refusal_arm_inventory_test.go` (lines,
refuse-wrap enforcement, refuse spec entry, derived shadow cover,
true header claims), `refusal_arm_witnesses_test.go` (descriptor #2
witness fix, header), `refusal_site_audit_test.go` (new: TestMain
audit, mapping, attribution pin, refuse-alias pins),
`digit_guard_census_test.go` (F3 bound). Round-1 files otherwise
unchanged. Harness: `mutate2.py` (+audit mask, +audit evidence
lines), `muts_tb_r2.json`, `muts_ph_r2.json` (new rows).

