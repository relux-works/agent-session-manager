# BUG-260903-3nt3ah — canonicalize-doc-promises-refusal-it-does-not-perform

Finding N5 (LOW) of the second adversarial audit. Last of the thirteen
second-audit findings.

## Decision: (b) — document the RFC 8785 rounding as intended behaviour

Option (a), making `Canonicalize` refuse the literals its doc comment claimed to
refuse, is not available. It contradicts both authorities the AC names.

**RFC 8785.** Section 3.2.2.3 serializes every number through the ECMAScript
`Number.prototype.toString` algorithm applied to the IEEE 754 binary64 value.
Appendix B publishes the resulting rounded outputs as normative test data, and
two of its samples are exactly this rounding: `0x4340000000000000` →
`9007199254740992` and `0x4430000000000000` → `295147905179352830000`. Those
samples are pinned at `canonical_test.go:47-70` in
`TestCanonicalizeMatchesEveryFiniteRFC8785AppendixBNumberSample`. Refusing
`9007199254740993` in `Canonicalize` means refusing to be an RFC 8785
canonicalizer, and it means deleting that test — which the DoD forbids.

**Pinned SPEC.** The specification does not put the refusal on the
canonicalization step. It puts it on the AX decoder: "The safe-integer
restriction is part of every 1.0.0 JSON, CBOR, identity, and wire contract. A
decoder MUST reject a numeric literal at or beyond `2^53` even if its host
language can represent it ... Implementations MUST NOT round a value and
continue." (SPEC.md:291-296). The normative fixture is explicit about *where*:
`NUM-UNSAFE-ROUND` — "Reject **before identity calculation**; an implementation
that first rounds it to 9007199254740992 is nonconforming" (SPEC.md:302).
Identity calculation is `CalculateObjectIdentity` / `VerifyObjectIdentity`, and
`validateAXNumbers` runs inside `prepareObjectIdentity` before any canonical
bytes exist. The implementation already conforms; only the doc comment lied.

`Canonicalize`'s single production caller is `canonicalEncoding` in
`declared_byte_bounds.go`, which measures declared byte bounds on values that
have already passed the AX gate and needs the true RFC 8785 form. There is no
caller that wants a refusing variant.

## What changed

`internal/canonicaljson/canonical.go`

- `Canonicalize` doc comment rewritten. The retracted "rejects ... non-I-JSON
  input" claim is gone. It now enumerates what the strict decoder does refuse,
  states the rounding as intended RFC 8785 behaviour with three machine-readable
  rows (`9007199254740993 -> 9007199254740992`,
  `18446744073709551615 -> 18446744073709552000`, `1.0 -> 1`), and carries a
  DIVISION OF GUARANTEES section naming the four exported entry points that do
  enforce the AX number model.
- `CalculateObjectIdentity` and `VerifyObjectIdentity` doc comments state the
  same split from the other side — a caller reading either one is told the
  guarantee lives here and not in `Canonicalize`.
- `validateAXNumbers` documented as the AX decoder half of the split, including
  that it decides on the `json.Number` token rather than on a host double.

`README.md` — the split, the three rounded literals, and how the pin works,
alongside the existing decoder/nesting-bound paragraphs.

`internal/canonicaljson/canonical_number_contract_test.go` (new)

The doc comment is the *expectation*, not a restatement of it. The rows and
names are parsed out of the comment with `go/parser` and checked against the
package's real behaviour, so prose and code cannot drift apart in either
direction.

- `TestCanonicalizeDocumentedRoundingIsWhatCanonicalizeActuallyDoes` — every
  documented row is driven through the real `Canonicalize` and must match byte
  for byte; the three audited literals must remain documented.
- `TestCanonicalizeDocumentedNonGuaranteeIsRefusedAtEveryNamedEntryPoint` —
  negative test. Each documented literal is carried into an otherwise valid
  Session Record / Observation Event through an `extensions` member, which is
  the one place a number survives every closed-shape validator
  (`validateExtensionValue` accepts any `json.Number` unexamined). All four
  named entry points must refuse it with the typed error. A green baseline with
  a safe extension number runs first, so the refusals cannot be vacuous.
- `TestCanonicalizeDocumentsExactlyTheEntryPointsThatEnforceTheAXNumberModel` —
  derives the exported functions reaching `validateAXNumbers` from the
  production call graph and requires the doc's list to equal it exactly, with
  `Canonicalize` absent and the retracted `non-I-JSON` phrase absent.
- `TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee` — the precondition that
  makes the derivation exact rather than merely sound.

### The call graph is reused, not re-derived

The first draft of this file built its own AST call graph recording only
`call.Fun.(*ast.Ident)` edges and skipping methods. That is exactly the graph
the 2026-09-03 0210 logbook entry measured at 3 of 7 UTF-8 guards while it
looked complete, because this package dispatches through a function-value table.
Shipping a second, weaker graph next to the hardened one would have reopened a
blind spot this repository had already paid to close.

`exportedFunctionsReaching` now calls `productionCallGraph`,
`exportedProductionEntryPoints` and `reachableProductionFunctions` from
`utf8_subsumption_test.go`, so it inherits the function-value and method edges
and the residual constructions stated there. Within that model the derived set
is *exact*, not merely sound: a computed callee is resolved to the functions
whose identifier is used as a value in the package, and `validateAXNumbers` is
never used that way — which is the precondition
`TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee` holds, and which mutant
M6 below shows is load-bearing.

`canonical_test.go` is byte-for-byte unchanged (`git diff --numstat` → 0 lines).

## Mutant proofs

Each mutant was confirmed present in the source before the test was run. Log:
`.temp/BUG-260903-3nt3ah/mutants.log`.

| Mutant | Shape | Confirmed present | Test | Exit |
| --- | --- | --- | ---: |
| M1 | Gate NARROWED, not deleted: `integer > maxSafeInteger` → `maxSafeInteger+2` | `grep` showed the widened bound | `...NonGuaranteeIsRefusedAtEveryNamedEntryPoint/9007199254740993` | 1 |
| M2 | `validateAXNumbers` call removed from `prepareObjectIdentity` | call-site count in `canonical.go` dropped to 0 | same test, all three subtests | 1 |
| M3 | Doc row changed to `1.0 -> 1.0`, code untouched | `grep` matched the mutated row | `...RoundingIsWhatCanonicalizeActuallyDoes/1.0` | 1 |
| M4 | Audited row `18446744073709551615 -> ...` deleted from the doc | `grep` no longer matched | `...RoundingIsWhatCanonicalizeActuallyDoes` | 1 |
| M5 | `Canonicalize` starts calling `validateAXNumbers` | `grep` inside the function body matched | `...DocumentsExactlyTheEntryPointsThatEnforceTheAXNumberModel` | 1 |
| M6 | `validateAXNumbers` registered in a function-value table and the identity gate routed through it — behaviour unchanged | 2 references to the table; `go build ./...` exit 0 | `TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee` | 1 |

M1 is the narrowing mutant the evidence contract asks for: it leaves the gate in
place and only moves its bound, and the negative test still catches it. M5's
failure message also correctly reported `CanonicalByteLength` entering the
reachable set, which is the graph reporting a real transitive edge rather than
the doc's list.

M6 is the one that identifies which test is load-bearing. It changes no
behaviour, so the negative test stays **green** (`exit=0`, recorded in the log)
and only `TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee` reddens. That is
the precondition doing real work rather than narrating a bound: without it, a
dispatch-table registration would silently make every computed call site reach
`validateAXNumbers`, widening the derived entry-point set past anything the doc
comment tracks, with the whole suite still `ok`.

M1-M5 were run twice, before and after the call-graph swap; the log carries both
passes under a marked header.

## Validation

All commands run as standalone processes; real exit codes.

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l internal/canonicaljson` | 0, no output |
| `go test ./... -cover -count=1` | 0 (canonicaljson 97.2%, unchanged) |
| `go run ./internal/catalog/cmd/cataloggen ... -check` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x` | 0 |
| `curator status --check` | **1** — see below |

Logs: `.temp/BUG-260903-3nt3ah/repo-tests-01.log`,
`.temp/BUG-260903-3nt3ah/repo-tests-02.log` (re-run after the call-graph swap),
`.temp/BUG-260903-3nt3ah/gates-01.log`,
`.temp/BUG-260903-3nt3ah/mutants.log`.

`curator status --check` exits 1 with `worktree: go-testing-tools not-installed`.
This is a pre-existing property of the Story worktree, not of this change:
`.claude/skills/` does not exist in the worktree at all, and this diff touches
no Curator-managed path (`git diff --name-only` matches no `Skillfile.json`,
`.agents/`, `.claude/` or `.codex/` entry). Reported as failing rather than
explained away; the orchestrator owns worktree provisioning.

## Residual

Named, not closed:

1. **Prose outside the checked rows is not verified.** The pin covers the
   `literal -> canonical` rows, the required-literal set, the entry-point names,
   and the absence of the `non-I-JSON` phrase. The RFC and SPEC quotations in
   the doc comment are not checked against their sources — no test inside this
   package can reach RFC 8785 or SPEC.md text for the quoted clauses. A future
   editor could still misquote the reason while every row stays true.
2. **The call graph's residual is inherited, not removed.**
   `exportedFunctionsReaching` reuses `productionCallGraph`, so it models
   identifier calls, methods declared in this package, and function-value
   dispatch — and it still cannot see reflection, a function value handed to
   another package and invoked there, or a func-typed struct field, since
   `x.M(...)` resolves only when `M` is declared here. That residual is stated
   on the graph itself and restated on the test that uses it. The derived set is
   exact *because* a second test enforces a precondition (M6), not because the
   graph is universal.
3. **The three rounded literals are examples, not a closed class.** Every
   binary64-inexact literal rounds; the doc documents three and the pin proves
   three. The class itself is closed by the identity entries' refusal of *any*
   literal outside the safe interval, which
   `TestCalculateObjectIdentityRefusesANonAXNumberNestedInAContainer` and this
   file's negative test cover at the production entries.
