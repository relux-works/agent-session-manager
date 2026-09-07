# TASK-260906-v8heil review rev3 — VERDICT: ACCEPTED

Reviewer run `RUN-260907-39f8d1`. Change Request `CR-TASK-260906-v8heil-3` revision 3,
22 paths, base `5da63ad31990179fcd907f6dc3786c7db81379ed`,
candidate tree `316d093bffbb1f2ba187025a12ac91f0d5d7f22e`.
repeat-of: none (accepted).

Every finding below was produced by executing a plant against the candidate tree in
this worktree and reverting it, not by reading the producer's report. The tree OID was
recomputed to `316d093b…` before the first plant and again after the last revert.

---

## Round-3 blocking gates

### G-A — did the shadow residual move? PASS

| Check | Measured here |
|---|---|
| Residual recomputed independently | 210 derived construction sites; 24 shared `(code, detail)` clauses covering 88 sites; 26 of those are bound-exempt → **62 of 184 witnessed arms** share a clause. Matches the producer's figure exactly. |
| Largest clause | `CodeMismatch` / `"document member type"` — 12 sites (11 witnessed + 1 bound), matching the reported ×11. |
| Audit-only kill declared as such | Battery row `S-tb-shadow`: `class="shadow"`, census SURVIVED(2), behav SURVIVED(185), audit KILLED(657), `killers=["audit"]`. Reported as its own class, not folded in with narrowing/deletion. |

**Class generalizes beyond the one site round 2 fixed.** I planted shadowing in a
*different* clause — the 12-way `document member type`, not `CheckEntrypoint` —
by widening `manifest.go` `parseClaim`'s `object["value"].(bool)` guard to also
require `generation_variable`, killing the following arm by construction:

| Mask | Result |
|---|---|
| behavioural (`-run TestEveryDeclaredArmRefusesAtItsEntry`, 184 witnesses) | **ok** (exit 0) |
| census/bijection (`-run TestDerivedSiteLinesAreExactlyRowed\|TestDeclaredRefusalArmsAreAllDerived\|…`) | **PASS** ×3 |
| full package run | **FAIL** — `refusal-site audit: derived refusal sites without an exercised path (shadowed dead or unwitnessed): manifest.go:766` |

Audit-only kill, naming the exact dead site, in a clause the producer never touched.
The gap round 2 opened is closed as a class, not as one site.

### G-B — F3's corrected bound: PASS, with one false justification clause (F-B1)

| Claim in the corrected bound | Verified |
|---|---|
| the strconv-delegated shape exists at `provhost/opdecode.go` `rawUint53` | **TRUE** — `strconv.ParseUint(literal, 10, 64)` + error branch + `parsed > maxUint53` at `opdecode.go:81-83` |
| what covers it | **TRUE** — narrowing `parsed > maxUint53` → `> maxUint53+1` reddens exactly `TestDecodeQuiesceRefusals/count_overflow`; the cross-package digit census stays **ok**. Reproduces battery row `B-provhost-strconv-bound` (census SURVIVED 1 / behav KILLED 20) byte-for-byte in verdict. |
| code-point spellings ENUMERATED | **TRUE** — `digitGuardDigitRune` admits `token.INT` 48..57 in every base, pinned by `TestDigitGuardDigitRuneAdmitsCodePointSpellings` with adjacents (47/58/`0x2F`/`0x3A`) rejected. |
| named-rune constants OUTSIDE the classifier | **TRUE** as a bound. |

**F-B1 (non-blocking finding).** The named-rune bullet contradicts itself and its second
half does not reproduce. It says the chain "prunes" (honest) and then that a
named-constant gate "would classify as other and **fail as unclassifiable, not pass
silently**" (false), and uses that second claim to *re-verify* the absence.

Control plant, additive, no existing row touched — a pure named-const digit gate
appended to `internal/provhost/protocol.go`:

```go
const digitZeroConst = '0'
const digitNineConst = '9'
func namedConstDigitGate(text string) bool {
	for i := 0; i < len(text); i++ {
		if c := text[i]; c < digitZeroConst || c > digitNineConst { return false }
	}
	return true
}
```

→ `internal/terminalbackend` **ok**, `internal/provhost` **ok**. It passes silently.

The absence it justifies is nevertheless independently TRUE: an AST scan of both
packages for digit-valued named constants and comparisons against them returns
`0` and `0`. And *rewriting an existing rowed gate* to named constants **is** caught
(I rewrote `protocol.go:379` and got `orphan row provhost|protocol.go|parseMajor|char|…`).
So the bound's substance holds; one sentence claims a fail-closed property the
classifier does not have. Delete the re-verification clause, keep "the chain prunes".

### G-D — the battery: PASS

19 KILLED / 19 applied, 0 SURVIVED, with `NOT_APPLIED` and `COMPILE_FAIL` as two
distinct control rows. Classes separate: narrowing 11, arm-deletion 3, census-only 3,
token-preserving 1, shadow/audit-only 1. Every mask ran ≥ 1 test (minimum 1; no empty
mask). Denominator production-derived and re-derived by me: **210** terminalbackend
(own AST re-derivation) + **167** provhost (log line `refusal arm coverage domain: 167
derived arms across 15 production files`) + **18** provider (own count of the four
`failXxx` call sites in `provider.go`) = **395**.

Two rows reproduced independently: `B-provhost-strconv-bound` exactly, `S-tb-shadow`
in a different clause (above). `C-tb-dead-arm` is honestly demoted to `census-only`
with both killers named — it is no longer counted as dead-arm evidence.

### G-E — provenance and frozen ground: PASS, with a stale OID in the doc (F-E1)

| Check | Result |
|---|---|
| Candidate tree = CR record | `316d093bffbb1f2ba187025a12ac91f0d5d7f22e` recomputed via detached `GIT_INDEX_FILE` + `read-tree HEAD` + `add -A` + `write-tree`, **before** the first plant and **after** the last revert. Equal both times. |
| `ls-tree` shows the files | `internal/invcore/{invcore,invcore_test,must}.go`, `internal/terminalbackend/{refusal_arm_witnesses_test,refusal_site_audit_test}.go` present as blobs. |
| leaves 1 and 2 untouched from `5da63ad` | No `internal/provhost` or `internal/provider` **production** file modified. Only their `_test.go` files, which is this leaf's port. |
| floors | provhost `refusalArmCensusFloor` 166 = 166; tb declared rows 209 = 209, derived 210 = 210; provider sites 18 = 18. **No package regresses.** |

**F-E1 (minor finding).** The outcome doc and the logbook entry both name candidate tree
`9d7eaa4d2c4e075e235ef708ea3b175f0ecc4eed`, not the record's `316d093b…`.
`git diff --stat 9d7eaa4d 316d093b` = `LOGBOOK.md | 9 +++++` — the delta is exactly the
logbook append written after the doc, i.e. the doc records its own pre-logbook tree.
Immaterial to code and fully accounted for, but DoD row 9 asks for equality; the
self-referential ordering (writing the OID into a file inside the tree) needs a
convention, not a re-run.

**On G-E's "no production file outside `internal/invcore` modified":** stale relative to
the accepted design. The round-2 fix *is* a terminalbackend production seam, disclosed
under F6. I verified it mechanically instead:

- 89 single-line hunks are **byte-identical to base modulo the `refuse(...)` wrapper**
  (programmatic unwrap-and-compare over `git diff -U0`; 89/89 matched, 0 unexplained).
- The only other production hunks are the `recordRefusal` hook var + the `refuse` funnel
  + `//go:noinline` on `mismatchf`/`integrityFailure` and their hook calls, all of which
  return the same value they built.
- `recordRefusal` has **no non-test assignment** anywhere in the repo; it is nil in
  production, so construction stays a pure allocation.

Behaviour-neutral by construction, not by assertion.

---

## G-C — F4 / F5 / F6 and scope: PASS

- **F4** — `TestShadowedLookupsHaveNoInput` no longer hand-lists 19 pairs. The cover
  analysis derives 11 `mismatchf("document members")` misses from the AST, joins them to
  derived arms in **both** directions, computes cover through same-function preceding
  `checkExactMembers` on the same map var and recursively through every production
  caller, and requires covered ⟺ `boundShadowedLookup` and uncovered ⟺ witnessed, plus
  `covered == boundRows`. Attacked: neutering
  `checkExactMembers(object, attachAuthorizationMembers)` in `ParseAttachAuthorization`
  reddens with `shadow miss conformance.go|ParseAttachAuthorization:623 is uncovered but
  its row claims bound "unreachable: …"` and `cover analysis covers 8 misses, want
  exactly the 9 boundShadowedLookup rows`.
- **F5** — `hasLocal` dead loop gone (0 hits repo-wide). `ParseSource` has 3 call sites.
- **F6** — disclosure accurate; verified mechanically above.
- **Scope** — `grep -rl '"go/parser"' --include="*_test.go" internal/` reproduces
  **42 at base → 30 now = 12 ported**. By package, 3 of 12; the 9 remaining are named and
  my own enumeration of base packages carrying `go/ast`|`go/parser` test walkers returns
  exactly those 12. The 50 → 42 change is a *metric* change (`go/ast ∪ go/parser` = 50,
  `go/parser` = 42); the doc gives the new reproducible method but does not reconcile it
  against the prior number. Minor.

---

## Attacks beyond the brief

### Smuggling a refusal past the terminalbackend funnel — 5 shapes, 5 caught

| Plant in production | Result |
|---|---|
| raw `&Error{…}` literal returned without `refuse` | FAIL `ParseID (terminalbackend.go:158): &Error literal outside refuse()` |
| package-level `var smuggledVarArm = &Error{…}` returned | FAIL, attributed to the var initializer |
| `var aliasedRefuse = refuse` + call through it | FAIL |
| `refuse(buildSmuggled())` (non-literal argument) | FAIL `refuse wraps a non-literal` |
| unused `var unusedRefuseAlias = refuse` | FAIL `constructor "refuse" referenced outside direct-call position` |

The last one confirms `refuse` is in the **real production** alias watch list, not only
in the synthetic `TestRefuseFunnelRejectsAliases` plants.

### Narrowing the shared core — 7 directions, 7 killed by a named test

| Mutant (gate stays present, weakened to admit one member) | Killed by |
|---|---|
| resolve import aliases by local spelling instead of import path (audit sites) | `TestAuditRejectsImportAlias` + terminalbackend downstream |
| admit a var binding of `mismatchf` specifically | `TestAuditRejectsLocalAlias`, `TestAuditRejectsConstructorAsArgument` + 4 downstream FAILs |
| drop the dot-import fail-closed | `TestAuditFailsClosedOnDotImport` |
| drop `DiffSets`' orphan-row direction | `TestDiffSetsFailsBothDirections` + 4 downstream |
| drop the zero-production-files floor | `TestScanProductionFailsClosedOnZeroFiles` |
| skip an unparseable file instead of failing | `TestScanProductionFailsClosedOnUnparseableFile` |
| drop the empty-derivation floor | `TestDiffSetsFailsClosedOnEmptyDerivation` |

(The whole-file variant of the first mutant is a legitimate `COMPILE_FAIL` — unused
`imports` — and was re-anchored to the audit's two sites before being counted.)

### F-C1 (non-blocking, PRE-EXISTING, measured) — provider's runtime direction does not catch an init-time constructor alias

Plant in `internal/provider/provider.go`:

```go
return Candidate{}, aliasedFailInvalid(fmt.Sprintf("provider %q target %q is not a regular file", id, canon))
...
var aliasedFailInvalid = failInvalid
```

→ `internal/provider` **ok**. Nothing reddens. Two directions go silent together:

1. the package-level alias is initialised **before** `TestMain` swaps `failInvalid`, so it
   holds the original value and records no site;
2. the derivation keys on `refusalConstructors[ident.Name]` in direct-call position, so the
   site simultaneously leaves the derived set — and provider carries no site-count floor,
   only `len(Sites) == 0`.

- **Not a regression.** The identical plant on a clean `git archive 5da63ad` extract is
  also green. Coverage is *preserved*, which is what the AC asks.
- **provhost catches it** (its `ctor|<constructor>|<detail>` obligation disappears and the
  witness orphans → FAIL). **terminalbackend catches it** — it is the only package that
  runs `invcore.AuditConstructorReferences` over production source.
- The finding is in the *evidence*: outcome §5's row
  `Constructor var binding (provider failXxx) | runtime SiteRecorder | TestAuditRejectsVarBinding + TestRecorderAttributesProductionFrame`
  reads as provider-production coverage. Both named tests are **core** tests over
  synthetic source; neither fails when provider production carries the bypass. The task
  description's premise "provider catches aliases at runtime" is false for this shape.
- The AC's literal requirement is met — each core direction *does* fail when dropped
  (rows 1 and 2 of the table above). Closing the gap means applying
  `AuditConstructorReferences` to provider (and provhost) production; that is story-seam
  work, not this leaf's AC.

---

## Gates re-run in this review

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` / `GOOS=windows go vet ./...` | exit 0 / exit 0 |
| `gofmt -l internal/` | empty |
| `go test ./... -count=1` | exit 0, no non-`ok` package |
| `go test -race ./internal/{invcore,terminalbackend,provhost,provider} -count=1` | all ok |
| `go test … -cover` | invcore 70.2%, provhost 86.0%, provider 97.8%, terminalbackend 95.0% — exact match to the doc |
| `go run ./internal/traceability/cmd/tracecheck` | `traceability ok: contracts=60 …` exit 0 |
| `go generate ./internal/catalog` | no diff (20 status lines before and after) |

**CI actually runs the audit.** `.github/workflows/ci.yml:115` is
`go test ./... -v -count=1` with no `-run` mask, so `fullPackageTestRun()` is true and
the TestMain exercised-site audit fires. The fuzz step's `-run='^$'` correctly turns it
off; that is the disclosed and correct behaviour, not a hole.

---

## Findings carried forward (none blocking)

1. **F-B1** — `digit_guard_census_test.go`: delete the "would … fail as unclassifiable,
   not pass silently" clause from the named-rune bullet. It contradicts the same bullet's
   "the chain prunes" and does not reproduce (control plant above). The absence it
   justifies is independently true; state it as grepped, not as re-verified.
2. **F-C1** — outcome §5's provider row overstates. Either restate it as "core direction,
   synthetic" or close it by running `invcore.AuditConstructorReferences` over
   `internal/provider` and `internal/provhost` production. Pre-existing at `5da63ad`;
   belongs to the story's remaining seam list.
3. **F-E1** — outcome doc / logbook name tree `9d7eaa4d…`, record carries `316d093b…`;
   delta is the logbook append alone. Needs an ordering convention for writing an OID
   into a file that is inside the tree.

## Why this is accepted

Every gate the round-3 brief named blocking is satisfied by measurement I performed:
the shadow class is killed in a clause the producer never touched, the audit-only kill is
declared as its own category, F3's corrected bound reproduces at the named site with the
named test, the battery separates all five classes over a denominator I re-derived, and
the tree matches the record before and after every plant. The production seam is
mechanically proven behaviour-neutral, every core direction dies under a narrowing
mutant with a named test, and no package regressed a floor. The three residual findings
are documentation accuracy and one pre-existing, now-measured gap outside this leaf's AC.
