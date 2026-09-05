# TASK-260830-1snnef review verdict — round 4 (final)

**Verdict: accepted.** No blocking findings. One new bound to record, one
correction to my own round-3 numbers, and one observation.

CR `CR-TASK-260830-1snnef-4` rev 4, base `57afcc6`, candidate tree
`b1940de7413515dd332f10d662ef3c88d37408ec`. I verified the working tree equals
that OID before the first probe and again after the last of **271 mutants /
467 test executions**, including the non-Go `ownership.v0.5.0.json` path.

H1 is closed for real — I killed it with my own mutants rather than reading the
rework note. The whole Story re-measures with **zero resurrections** in all
three inherited batteries, and every gate is green.

---

## G-A — the pin is exact, isolated, and not self-derived

`TestLegacyForwardIsExactlyTheImmutablePair` was the round-3 ask. It holds.

### It fails in both directions

Eleven mutants over `legacyForward`, each run 3–5×. **11 of 11 RED**, and
`TestLegacyForwardIsExactlyTheImmutablePair` is among the failing tests in
every one.

| # | Shape | Round 3 | Round 4 |
| --- | --- | --- | ---: |
| A1 | third row → fresh canonical `vendor.screen` (**the H1 survivor, L3**) | GREEN 5/5 | **RED 5/5** |
| A2 | third row → `BuiltinTmux` (**C20, the coin flip**) | 5 RED / 7 GREEN of 12 | **RED 5/5** |
| A3 | third row → `vendor.term` (L4) | RED (incidental) | **RED 3/3** |
| A4 | third row → `BuiltinConpty` (L1) | 3 RED / 3 GREEN of 6 | **RED 5/5** |
| A5 | third row `""` → `BuiltinTmux` | — | **RED 3/3** |
| A6 | lose the `conpty` row (L2 control) | RED | **RED 3/3** |
| A7 | lose the `tmux` row | — | **RED 3/3** |
| A8 | both rows → `BuiltinConpty` (injectivity broken, size still 2) | — | **RED 5/5** |
| A9 | `conpty` repointed to `vendor.conpty` (size still 2) | — | **RED 3/3** |
| A10 | key renamed `tmux`→`screen` (size still 2, canonicals intact) | — | **RED 3/3** |
| A11 | the two canonicals swapped | — | **RED 3/3** |

Every previously flaky or surviving mutant in this family is now deterministic.
Three shapes (A9, A10, A11) preserve the map's size, so they can only be caught
by the membership halves — they are.

### The pin does the killing, not an incidental collision

Round 3's L4 was red only because it happened to collide with a string
`TestLegacyReverseProjectionExistsOnlyForThePair` names. So I re-ran six shapes
under `-run TestLegacyForwardIsExactlyTheImmutablePair` alone:

| # | Shape | Pin alone |
| --- | --- | ---: |
| B1 | third row → `vendor.screen` | **RED 5/5** |
| B2 | third row → `BuiltinTmux` | **RED 5/5** |
| B3 | lose the `conpty` row | **RED 3/3** |
| B4 | `conpty` → `vendor.conpty` | **RED 3/3** |
| B5 | key rename, size 2 | **RED 3/3** |
| B6 | both rows → one canonical | **RED 5/5** |

The pin is sufficient on its own for every shape. No dependence on a sibling
test naming the right string.

### `want` is not a number compared against itself

This Story hit that shape three times, so I attacked the shared source.

`want` is built from the production constants `BuiltinTmux`/`BuiltinConpty`, not
from `legacyForward`. A mutant that touches only the map cannot move `want` —
that is what A1–A11 measure. The residue is a simultaneous move of the
constants, which **the pin does not see** (D2: `BuiltinTmux → "ax.evil"`, pin
alone GREEN 3/3; D8: `→ "ax.tmuxx"`, pin alone GREEN 3/3).

That residue is covered elsewhere, and I measured it rather than assuming it:

| # | Shape | Package suite | Whole repo |
| --- | --- | ---: | ---: |
| D1 | `BuiltinTmux → "ax.evil"` | **RED 3/3** (8 tests) | **RED** (D5) |
| D6 | `BuiltinTmux → "vendor.tmux"` (valid, non-reserved) | **RED 3/3** | — |
| D7 | `BuiltinTmux → "ax.tmuxx"` (valid, still reserved) | **RED 3/3** | **RED** (D9) |

Every canonical-literal mutation reddens. The pin's source is independently
pinned by the ID grammar, reserved-namespace, and registry-admission tests. This
is a two-source comparison, not a self-comparison.

### Injectivity landed and fires

Two independent assertions, and I confirmed each fires on its own:

- The top-of-test guard `BuiltinTmux == BuiltinConpty` — **D4: pin alone RED 5/5**
  when the two constants are made equal.
- The `seen` duplicate-value loop — **B6: pin alone RED 5/5** when both rows map
  to one canonical.

Worth stating precisely: given the exactness pin, the `seen` loop cannot fire
*alone* on a two-row table — any duplicate value implies a membership error too.
It is defensive redundancy that becomes load-bearing if the table ever grows.
The property `ProjectToLegacy`'s doc comment rests on is enforced; the claim is
accurate.

---

## G-B — the last full sweep, zero resurrections

### My conformance battery — 48 mutants, 144 executions, 3 runs each

| Outcome | Round 2 | Round 3 | Round 4 |
| --- | ---: | ---: | ---: |
| Killed | 34 | 46 | **46** |
| Survived | 12 | 2 | **1** (C45) |
| Non-compiling (no measurement) | — | — | **1** (C08) |

C20 moved GREEN(flaky) → **RED 3/3**. C45 (`allowedOperationErrors` widened with
a non-catalogued code) remains the honest unbounded-domain bound, disclosed
in-test. **No mutant produced a mixed verdict across its repeats** — zero
flakiness anywhere in the battery this round.

**Correction to my own round-3 numbers.** C08 (`carried.String() != session.String()`
→ self-comparison) leaves `session` unused, so it does not compile. Round 3
recorded it as a kill on exit code alone — the exact error I corrected leaf 2's
battery-1 for in the same verdict. A non-compiling mutant is not a measurement.
C08b is the compiling variant and is killed. The honest headline is **46
measured kills of 47 measurable**, not 46/48.

### Leaf 1 — TASK-260830-2xdt8t (71 mutants)

| Outcome | Round 3 | Round 4 |
| --- | ---: | ---: |
| Killed | 60 | **60** |
| Survived | 8 | **8** |
| Not applied | 2 | **2** (N12, T01) |
| Build error | 1 | **1** (M32) |

Survivors reproduce id-for-id: **N09, N13, N14, N15, X01, X02, X04, X10**.
**Zero resurrections, zero regressions.**

**T01 re-applied by hand** (its edit is additive, so the harness cannot measure
it): **KILLED, 89 tests red**, and `ownership.v0.5.0.json` restored to
sha256 `22bf4a8eed2e6de3…` — the same digest as round 3. That file is outside
the Go-file restore set of the inherited harness, so I watched it explicitly.

### Leaf 2 — TASK-260830-1zvmw7

| Battery | Killed | Survived | Non-compiling | Anchor bad |
| --- | ---: | ---: | ---: | ---: |
| battery-1 (60) | 46 | **0** | 12 | 2 |
| battery-2 (47) | 47 | **0** | 0 | 0 |

Identical to round 3. The 12 non-compiling battery-1 mutants are all re-run as
compiling variants in battery-2 and killed there, so the gates are proven; the
headline stays "46 measured kills + 12 covered by battery 2".

---

## J1 (new, recorded as a bound — not blocking) — the plain-error bound is narrower than the hole

H2's fix works. `rejectPlainErrorAliases` kills every shape I named in round 3
and several I did not:

| # | Shape | Result |
| --- | --- | ---: |
| E0 | CONTROL: direct `errors.New` in a FuncDecl | **RED 2/2** |
| E1 | `var newPlain = errors.New` + call (**round-3 V5, the finding**) | **RED 3/3** |
| E2 | function-local `newPlain := errors.New` | **RED 3/3** |
| E3 | `fmt.Errorf` alias | **RED 3/3** |
| E4 | `errors.New` passed as an argument | **RED 3/3** |
| E5 | `errors.New` stored in a struct field | **RED 3/3** |
| E6 | alias declared in a **new** production file | **RED 3/3** |

But the residual bound the tree states is not the residual bound I measured:

| # | Bypass | Result |
| --- | --- | ---: |
| E7b | `import errs "errors"` → `errs.New(…)` in an existing file | **GREEN 3/3** |
| E7c | same, in a new production file | **GREEN 3/3** |
| E9b | `errors.Join(os.ErrClosed, os.ErrInvalid)` — a third stdlib spelling | **GREEN 3/3** |
| E10 | bespoke error type (**the stated bound**) | GREEN 2/2 — correctly disclosed |

`isPlainErrorSelector` matches on the *receiver identifier* (`errors`, `fmt`),
so the identical constructor under an import alias is invisible to both the
allowlist scan and the new alias gate. The tree discloses only "custom error
types outside the two recognised spellings"; `errs.New` is not a custom type,
it is the same function. So the sentence "an aliased construction cannot ship
unattributed" is true for assignment aliases and false for import aliases.

**Why this is recorded and not blocking**, stated so the judgement is auditable:

- Severity is the lowest in the change. A plain error carries no wire code, no
  capability claim, no authorization decision. This gate is test-suite
  attribution bookkeeping.
- **It does not touch the `*Error` inventory.** `rejectConstructorAliases`
  matches package-local `*ast.Ident`s (`mismatchf`, `integrityFailure`), which
  cannot be import-aliased. The README's claim — "every production refusal arm
  resolves to a named asserting test in both directions, so a refusal that ships
  unwitnessed fails the suite" — is about wire refusals and **stands**. I checked
  rather than assumed.
- It is accurate about the tree as it stands today: **zero aliased imports and
  zero `errors.Join` sites** exist in the package. All seven plain-error sites
  are direct calls in `validatePlatforms` (3) and `DigestFile` (4).
- In round 3 I graded this family non-blocking and offered "extend the gate **or**
  record the bound". The producer extended it, exactly as specified, and killed
  more shapes than I named. The residual is two spellings I did not name and did
  not measure last round.

The fix, if it is ever wanted, is to resolve the import path rather than the
receiver identifier. Recommended action now: widen the stated bound text to name
import aliases and non-`New`/`Errorf` stdlib constructors, and carry it forward.

**Observation.** Dot import (`import . "errors"` → `New(…)`) does not survive,
but the gate does not catch it either — the package declares its own `New`
(`terminalbackend.go:374`), so the dot import collides and the package fails to
compile. Closed by the compiler, not by the check. That closure would disappear
if `New` were ever renamed, so it should not be counted as gate coverage.

---

## G-C — bounds, coverage, capability claims

| Item | Result |
| --- | --- |
| P11 (legacy escape name) | **Genuinely closed now**, and by the pin rather than the flaky mutant — B1–B6. The round-3 stale bound is resolved. |
| LOGBOOK correction | Present and correctly scoped: the new entry names the false round-1/round-2 claims, attributes the error to the reviewer, and records the repeated-run lesson. Append-only correction rather than rewritten history — the right instrument. |
| P13 (`ProjectToLegacy` on an empty ID refuses with a different wire code) | Declined closure, stands. Guard removal is caught (RED 3/3). |
| W11 (DigestFile allowlist function-wide, no detail pin) | **Still a real bound**, correctly disclosed: a new `errors.New` inside `DigestFile` survives (GREEN 3/3). |
| W13 (bespoke error types) | Still open, correctly disclosed (E10). |
| Kill-inflation bound (inventory kills every arm-deleting mutant, so post-leaf-3 scores are not comparable to leaf 2's 91/95) | Carried forward in the inventory file's stated-bounds block and LOGBOOK. Correct. |
| Coverage | **90.0% of statements**, exactly as claimed. |
| Arm table | 193/193 undisturbed; both inventory directions pass. |
| Capability registry | **16 entries**, as the README states; pinned by `manifest_pin_test.go`. |
| Acceptance cases | tracecheck reports **76**, matching the README's updated count. |

**Bounds set as it now stands:** W11, W13, C45's unbounded code domain, the
kill-inflation non-comparability warning, the attribution-drift fallback, P13 —
plus **J1** to be added. P11 is the one that leaves the list, correctly.

---

## Gates I ran myself

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | **exit 0**, 14/14 packages |
| `go test ./internal/terminalbackend/ ./internal/config/ ./internal/traceability/... -race -count=1` | **exit 0** |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/ cmd/` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=76`, `bindings=49 … clauses_discharged=17/403` |
| `go test ./internal/terminalbackend/ -cover` | **90.0%** |
| Candidate tree identity, before first and after last of 271 mutants | `b1940de7413515dd332f10d662ef3c88d37408ec` ✓ |

## Architecture and advertised capability

Unchanged and sound. The harness holds no authority, does no I/O, leaves
capability gating in `manifest.go` and trust in `terminalbackend.go`.

I re-verified the round-1 bound on this tree: the only production importer of
`internal/terminalbackend` outside the package is `internal/config/validation.go`,
and it uses **`ParseID` only** — no production caller exists for any conformance
entry point. That is disclosed, not concealed: `conformance.go`'s header quotes
§4.B verbatim ("M0 exposes only an internal semantic interface and conformance
harness … performs no I/O, launches no backend, and holds no authority"), so
"production entry point" here honestly means the harness's own exported surface.
No unsupported capability is advertised.

---

## Acceptance

The round-3 blocking finding is closed by a pin I attacked eleven ways and could
not defeat. The Story's three inherited batteries reproduce element for element
with zero resurrections. Coverage, arm count, capability count, and acceptance-case
count are all the measured values the tree claims. J1 is a bound to record, not a
defect to fix before landing.

The leaf is clean. Accepted.
