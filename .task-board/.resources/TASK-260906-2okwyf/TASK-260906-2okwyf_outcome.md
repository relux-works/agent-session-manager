# TASK-260906-2okwyf — close-cross-package-agreement-and-trust-asymmetry: outcome

Status: ready for review.

Candidate tree: `2a2707266e0c50f080f3297d56e0a3a4e4b912b3`
(verified by `git ls-tree -r`: all 10 touched paths are in-tree, and
`git hash-object` of each working file byte-for-byte equals the tree
blob, 10/10 MATCH. Built via a detached index — `GIT_INDEX_FILE` copy
+ `read-tree HEAD` + `add -A` + `write-tree` — so the worktree index is
untouched. HEAD stays at the `114a056` checkpoint; the change is
uncommitted. Computed AFTER every tracked write, including the LOGBOOK
append. The Change Request record is the orchestrator's integration
step; this OID is the candidate it must carry.)

Scope (9 code/test paths + LOGBOOK, all uncommitted on the story
branch; no leaf 1–3 behavioral logic reworked — advisory-scoped edits
only):

- `internal/terminalbackend/terminalbackend.go` — ParseID grammar arm
  omits BackendID; DigestFile §6.5/§7.1 asymmetry note.
- `internal/terminalbackend/terminalbackend_test.go` — new
  `TestParseIDGrammarRefusalEchoesNothing`.
- `internal/terminalbackend/digit_guard_census_test.go` — F-B1 sentence
  corrected (comment only).
- `internal/provider/provider.go` — trustCandidate §7.1/§6.5 mirror
  note (comment only).
- `internal/provhost/probe.go` — `decodeValidatedProbe` split;
  `RequireCapability` consumes validated members (body decoded once).
- `internal/provhost/probe_test.go` — new
  `TestDecodeValidatedProbeHandsUsableMembersToRequireCapability`.
- `internal/provhost/protocol.go` — parseMajor leading-zero stated
  bound (comment only).
- `internal/provhost/protocol_test.go` — new
  `TestParseMajorLeadingZeroIsClassifiedAsForeign`.
- `internal/provhost/profile_agreement_test.go` — new (order-sensitive
  cross-package equality).
- `LOGBOOK.md` — this leaf's entry (newest first).

## 1. A5 — six-provider agreement (order pin added)

Premise narrowed during the work: a DIRECT set-equality pin already
exists — `TestSixProviderSetMatchesDiscoveryRegistry` (story-2 base,
`profile_test.go`) balances `provider.Builtins()` against
`profileProviders` with no SPEC.md read. The live hole was ORDER
agreement: that pin is order-insensitive, and provhost's census scopes
`profileProviders` out, so a reorder on either side alone left the
whole suite green.

`TestBuiltinsEqualProfileProviders` (new, `profile_agreement_test.go`,
package provhost so it reads the unexported registry with zero new
exports) asserts `reflect.DeepEqual(provider.Builtins(),
profileProviders)` — ordered, SPEC-independent (imports no specdoc).

One-sided proofs (mutants applied, then reverted byte-for-byte):

- provhost-only reorder (`muse`/`antigravity` swap): new test FAILS;
  `TestProfileYOLOMappingIsDerivedFromSpec` PASSES (sorted compare),
  provider suite PASSES — the gap was real.
- provider-only swap of the same pair: new test FAILS; restored green.

Both orders stay independently spec-pinned (§7.1 listed order for
discovery, §7.7 table order for profiles); a future bump that
legitimately reorders one table must revisit this test consciously.

## 2. A7 — trust asymmetry cited at both sites (no third path)

The spec is NOT silent, so clauses are cited, not decided:

- §7.1: "the target MUST be a regular file owned by the operator or
  an administrator-approved identity" — the owner dimension EXISTS.
- §6.5: "Each external-trust entry contains exactly backend ID,
  absolute executable path, executable digest, and `enabled`" — a
  closed entry with NO owner member.

`DigestFile` now notes trust is §6.5 external_trust (digest + enabled),
the symlink/regular-file steps mirror §7.1's mechanics, and the missing
owner dimension is contractual — the closed entry leaves no member to
record an owner in — not drift. `trustCandidate` mirrors with the §7.1
owner sentence, the OwnerPolicy naming pattern (leaf 2's
`OwnerIdentity` `uid:%d` form), and a pointer back to `DigestFile`.
The recorded decision already lives in `internal/secprim/doc.go`
("Trust for external executables is deliberately not implemented
here… the difference is contractual, not accidental"); both sites now
cite the clauses. No new trust path added. Behavior unchanged, so no
behavioral mutant is claimed here; the cited owner gate is defended by
battery row N-pv-trust-unapproved (§7).

## 3. A9 — three nits

| Nit | Close | Test (fail-before evidence) |
|---|---|---|
| ParseID grammar arm echoes raw refused input in `Error.BackendID`, printed by `Error()` | FIXED: arm omits BackendID, like the bound arm. Reserved arm keeps echoing (grammar-valid identity, like every other arm). All `ParseID` consumers discard the identity (`_, err :=`), and witnesses assert code/detail only — no collateral change. | `TestParseIDGrammarRefusalEchoesNothing`: 7 grammar-refused inputs (uppercase, space, traversal, CJK, NUL, 128×`A`, 64×`ë`) assert empty BackendID + no echo in `Error()`; bound-arm and reserved-arm controls pin the boundary. Red before (14 failures), green after. |
| `parseMajor` accepts leading zeros (`"03.0.0"` → major 3) | LEFT OPEN with stated bound on the function: classification-only — the gate admits exactly `"2.0.0"` by string equality and parseMajor only chooses mismatch (exit 6) vs unusable (exit 13); strict SemVer lives at admission (manifest `plugin_version`, shared semver grammar). Strictness here is not free: it spells `== '0'`, unclassifiable to the digit-guard census (battery row N-ph-parsemajor-strict kills the digit census AND the provhost inventory). | `TestParseMajorLeadingZeroIsClassifiedAsForeign`: `"03.0.0"` → (3,true) + `incompatible_protocol`/exit 6; `"02.0.0"` → (2,true) + `provider_protocol_error`/exit 13 with `unsupported protocol version`, through `DecodeResponse`. Pins the bound (passes by design on current behavior). |
| `RequireCapability` re-decodes the probe body after `DecodeProbe` | FIXED structurally: new `decodeValidatedProbe` holds every check and returns the members; `DecodeProbe` is its boolean form; `RequireCapability` consumes the members — the body is decoded exactly once. Sub-object replays keep `_` with an explicit nil-by-construction note (no new constructor calls, so the derived arm set is byte-identical; the site audit stays green because every moved site is still exercised by the same tests). Coupling documented at BOTH sites. | `TestDecodeValidatedProbeHandsUsableMembersToRequireCapability`: names the helper (fails to COMPILE on pre-fix code — verified via `git stash` of `probe.go`: `undefined: decodeValidatedProbe`), plus validation-first ordering rows (malformed body + known AND unknown capability → `provider_protocol_error`, never `invalid_config`). |

## 4. F-B1 — digit-guard sentence corrected (one false sentence)

The paragraph claimed a named-constant gate "would classify as other
and fail as unclassifiable, not pass silently". Control plants on the
real tree:

- PURE plant (`const plantZero/plantNine`, `c < plantZero || c >
  plantNine` in `provhost/protocol.go`): BOTH full packages green —
  the sentence is false for the pure case (chain prunes: every leaf
  `other`).
- MIXED plant (`c < plantZeroM || c > '9'`): census FAILS
  (`TestDigitCensusCoversEveryLeafGuard`, non-empty mask):
  `unclassifiable guard in protocol.go (plantMixedDigitGate) ...
  (mixes digit comparisons with unrelated comparisons)`.

  (Process note: the first mixed probe passed vacuously — the `-run`
  mask matched only the unit test, the census never ran. Masks are
  verified non-empty everywhere in this leaf; battery records carry
  ran-counts.)

Plants reverted surgically (no whole-file checkout after the
parseMajor doc landed — an early `git checkout --` wiped it once;
re-applied and re-verified). The sentence now states the precise
split; the absence stays a stated bound, re-verified by AST scan
(`/tmp/digitscan/main.go`, throwaway): 0 digit-valued named constants,
0 comparisons against them, across both packages' production files.

## 5. Mutation battery: 9/9 killed over 9 applied

Harness `.temp/TASK-260906-2okwyf/mutate_2okwyf.py` (leaf-3 runner,
unchanged), mutants `.temp/TASK-260906-2okwyf/muts_2okwyf.json`, merged
records `.temp/TASK-260906-2okwyf/battery_2okwyf.json`. Every mutant
anchors to a derived production position (refusal-arm inventories,
digit census, the two registry literals); every mask below ran ≥1 test
(ran-counts in the record). Convention, as in leaf 3: classes are
decided by the census/behav mask split; the audit mask runs only where
the class needs it.

| Mutant | Class | What it narrows the gate to | Census (ran) | Behav (ran) | Named killer |
|---|---|---|---|---|---|
| N-tb-parseid-uppercase | narrowing | pattern admits uppercase (`[a-zA-Z]` throughout): `AX.TMUX`, 128×`A` admitted | SURVIVED (15) | KILLED (229) | `TestParseIDRefusesWidenedGrammar` + `TestParseIDGrammarRefusalEchoesNothing` (first attempt with first-char-only widening admitted nothing in the corpus and SURVIVED both — reported, then strengthened) |
| D-tb-parseid-grammar | arm-deletion | grammar arm deleted: refused inputs admitted or slide to reserved | KILLED (15) | KILLED (229) | behav: widened-grammar + echo tests; census: `TestDeclaredRefusalArmsAreAllDerived` (orphaned row) |
| C-tb-parseid-dead-arm | census-only | unreachable new arm (`"terminal_backend_id shadow"` behind a space, grammar refuses first) | KILLED (15) | SURVIVED (229) | `TestDerivedRefusalArmsAreAllDeclared` |
| S-tb-shadow | audit-only | earlier guard widened to swallow the later arm's inputs, same pair (leaf-3 design, re-verified on this tree) | SURVIVED (15) | SURVIVED (229) | full-run audit (658 ran) naming `conformance.go:713, :716` |
| N-ph-requirecap-novalidation | narrowing | validation dropped: raw re-decode discarding faults; malformed slides to `invalid_config` | SURVIVED (26) | KILLED (73) | `TestDecodeValidatedProbe…` (ordering rows) + `TestRequireCapabilityRefusesUnprovenSurfaces` |
| D-ph-requirecap-known | arm-deletion | registry-membership arm deleted: unknown name slides detail | KILLED (26) | KILLED (73) | behav: `…UnprovenSurfaces` (`not a registry member`); census: `TestWitnessedArmsAreAllDerived` (orphan) |
| N-ph-parsemajor-strict | narrowing | leading-zero rejection (`len>1 && [0]=='0'`): `"03.0.0"` unrecognized | KILLED (1, digit census) | KILLED (73) | behav: `TestParseMajorLeadingZero…`; census: `TestDigitCensusCoversEveryLeafGuard` (unclassifiable equality) + provhost `TestDerivedRefusalArmsAreAllWitnessed` (new parse arm, verified in a separate run) |
| N-pv-trust-unapproved | narrowing | owner gate admits exactly UID 2000 | KILLED (102, full-run TestMain audit) | KILLED (4) | `TestDiscoverRefusesUnapprovedOwners` (+ `trust_sources_test` co-kills in the full run) |
| M-xprov-reorder | agreement | one-sided reorder of `profileProviders` | SURVIVED (27) | KILLED (2) | `TestBuiltinsEqualProfileProviders` only (`TestSixProviderSetMatchesDiscoveryRegistry` stays green — the delta this leaf closes) |
| X-control-notapplied | control | — (anchor occurs 0 times) | NOT_APPLIED | | |
| X-control-compilefail | control | — (`func Builtins() []string ` — vet rejects before tests) | COMPILE_FAIL | | |

Summary rows: KILLED 9, SURVIVED 0, NOT_APPLIED 1, COMPILE_FAIL 1.
Applied 9, killed 9 → **9/9 killed over applied** on the
production-derived denominator above. No surviving mutant, so no
survival bound is owed.

Bounds on the battery: no shipped gate inspects source text (the only
source-text classifiers in scope — refusal inventories, digit census —
are behaviorally untouched; F-B1 and the parseMajor bound are comments
plus pins), so the token-preserving-mutant clause has no target in this
diff — stated, not silently skipped. The pre-existing digit-boundary
pins (`TestParseMajorDigitBoundariesRefuseAtEntry`, `/` `:` admit-one
attacks) remain the narrowing evidence for the digit gate itself.

## 6. AC coverage ratio: 4/4 rows driven

| AC row | Production call site | Named committed test |
|---|---|---|
| six-provider literals equal directly, fails either side alone, SPEC-independent | `provider.Builtins()` + `profileProviders` (`internal/provhost/profile_agreement_test.go`) | `TestBuiltinsEqualProfileProviders` (one-sided mutants both sides) |
| 6.5/7.1 asymmetry cited at both sites or decision recorded | `terminalbackend.DigestFile` + `provider.trustCandidate` (comments); decision in `internal/secprim/doc.go` | citation (no behavior change); cited gates driven by `TestDigestFile` and `TestDiscoverRefusesUnapprovedOwners` |
| ParseID prints no unbounded refused input | `terminalbackend.ParseID` | `TestParseIDGrammarRefusalEchoesNothing` (red→green) |
| every closed nit has a fail-before test; every open nit a stated bound | `provhost.RequireCapability` / `decodeValidatedProbe`; `parseMajor` bound | structural fail-before (compile) + ordering rows; bound pin `TestParseMajorLeadingZero…` |

Gating/refusing behavior ships negative tests that fail when the gate
admits what it must reject (ParseID widened-grammar + echo,
RequireCapability malformed→protocol-error, owner-gate foreign UID,
agreement reorder); every shipped gate has a narrowing mutant above.

## 7. Gates (CI parity, this turn; every command standalone with real exit)

- `go test ./... -count=1`: exit 0, 23 ok, 0 FAIL
- `go test -race ./... -count=1`: exit 0, 23 ok, 0 FAIL
- `go test ./... -cover -count=1`: exit 0 (provhost 86.0%, provider
  97.8%, terminalbackend 95.0%)
- `go vet ./...`: exit 0; `GOOS=windows go vet ./...`: exit 0
- `gofmt -l internal/`: exit 0, empty
- `go build ./...`: exit 0; `GOOS=windows go build ./...`: exit 0
- `go run ./internal/traceability/cmd/tracecheck`: exit 0
- `go generate ./internal/catalog` + `git diff --exit-code`: clean
- cigate contracts 6/6 PASS; cigate claims 6/6 PASS
- fuzz smoke 13/13 (`fuzz: elapsed` observed per target)
- No stubs or mock-only behavior added; no platform assumption
  concealed (Windows vet + Windows build both green).

## 8. What the review should know

- The A5 premise in the brief ("no test asserts directly") was already
  half-closed by `TestSixProviderSetMatchesDiscoveryRegistry`; this
  leaf closes the remaining order half. The outcome says so plainly —
  the set test is credited, not overwritten.
- The brief's "do not edit leaves 1–3 production code" is read as
  "no behavioral rework of their logic": this leaf's production edits
  are the advisory-authorized minimum (one arm's field, one function
  split with identical checks, two comments, one doc paragraph).
  Anything beyond that stopped here would have been a block; nothing
  beyond it was needed.
- No board elements were created under STORY-260905-3t31e9; needs are
  stated here instead: none pending — ready for review.
