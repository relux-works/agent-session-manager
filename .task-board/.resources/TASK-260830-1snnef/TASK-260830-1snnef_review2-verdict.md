# TASK-260830-1snnef review verdict — round 2

**Verdict: changes requested → `to-dev`.**

CR `CR-TASK-260830-1snnef-2` rev 2, base `57afcc6`, candidate tree
`4400b106c826536d499d8e13a871c41af9764daa`. I verified the tree I measured **is**
the candidate: a scratch-index `git write-tree` returned `4400b106…` before the
first probe and again after the last one, every mutation restored.

Round 1's four findings are **closed for real** — I killed each one with my own
mutants, not by reading the rework note. The Story's four inherited batteries
re-measure identically, zero resurrections. The two closed derivation shapes stay
closed under replay.

Three findings block, all one family and all cheap: **a bounded, enumerable domain
measured by a sample and stated as a closure.** That is the class that blocked
round 1 (F2); this round it sits on three surfaces, one of which is the third
derivation shape the brief asked me to hunt.

---

## Round-1 findings: verified closed, by attack

| Round-1 finding | My mutant | Result |
| --- | --- | --- |
| F1 shape A (`var f = func() *Error`) | W4: package-level `var plantedRefusal = &Error{…}` returned from a `switch` | **RED** — both bijection directions |
| F1 shape B (`refuse := mismatchf`) | W2b/W2c: parametric wrapper `refuseFrame(detail)` → `integrityFailure(detail)` | **RED** — both directions |
| F2 (`CheckErrorAllowed` sample) | C01: admit `terminal_backend_unavailable` for all ten operations | **RED** — `TestCheckErrorAllowedRefusesEveryUnlistedCode` |
| F3 (triple equality third leg) | C02: drop the `auth.InputAuthorized` conjunct | **RED** — `TestCheckAttachResultRequiresTripleEquality` |
| F4 (expiry boundary instant) | C03: `!now.Before(expires)` → `now.After(expires)` | **RED** — `TestCheckAttachRequestBindsTransportInputAndExpiry` |

F2's rebuild is the right shape: the complement derives from
`catalog.Current().Errors`, the mirror is pinned inside the catalog, and the
zero-catalog floor fails closed. The admit direction proves mirror ⊆ production and
the complement direction proves production ⊆ mirror over the pinned vocabulary — a
real bijection, not a closed loop.

**Correction in the producer's favour:** round-1's declared survivor **P11** is no
longer a survivor. C20 (`legacyForward` gains `"screen": BuiltinTmux`) is **RED**,
killed by `TestLegacyReverseProjectionExistsOnlyForThePair`. The "declined closure"
bound for P11 in LOGBOOK 0022 is stale — the closed-set property is pinned.

---

## G-A — the ratio, and the third shape

### Denominator, built three ways independent of the derivation under test

| Method | What it counts | Result |
| --- | --- | ---: |
| Whole-file `ast.Inspect`, positional exclusion of the two constructor bodies | `&Error{}` literals + direct `mismatchf`/`integrityFailure` calls | 75 + 118 = **193** |
| Constructor identifier accounting | 120 idents = 118 call-funs + 2 decl names | **residual 0** |
| Declared table + funnel (the suite's own green bijection) | 192 rows + 1 | **193** |

**193 of 193 arms derived and declared — 100%.** New this round, and the reason I
can say the two closed shapes are still *empty* rather than merely *handled*:
`outsideAnyFuncDecl = 0`, `insideVarInitializer = 0`, `residual alias refs = 0`, and
a census of single-return refusal wrappers returns **0 candidates** — no
`refuseFrame`-style funnel exists in the tree at all.

### Attacks on the arm derivation: 10 of 10 killed or fail-closed

| # | Shape | Result |
| --- | --- | --- |
| W1 | one-line wrapper `refuseFrame() → mismatchf("literal")`, called from `CheckReplicable` | RED |
| W2b | parametric wrapper → `integrityFailure(detail)`, caller passes a new literal | RED (both directions) |
| W2c | same wrapper, caller passes an **already-declared** detail | RED (function key) |
| W3 | wrapper returning `&Error{Detail: detail}` with a parameterised detail | RED |
| W4 | package-level `var` holding a prebuilt refusal, returned from a `switch` | RED |
| W6 | refusal on a method of a new type, reached via a package-level **method value** | RED |
| W7 | arm in a **new build-tagged** production file (`//go:build darwin`) | RED |
| W8 | arm in a new file **excluded from this build** (`//go:build linux`) | RED |
| W17 | arm in a nested func literal inside a `FuncDecl` (re-verify A4) | RED |
| W14b | delete an existing arm (reverse-direction control) | RED |

Note: `mismatchf` is variadic-format, so `go vet`'s printf analyzer refuses a
non-constant format string — the `mismatchf(detail)` variant dies at the compiler,
not at the inventory. Worth knowing which guard is actually holding that line.

### The third shape is on the *sibling* guard in the same file

`TestPlainErrorsLiveOnlyInTheirTwoFunctions` walks `file.Decls` → `*ast.FuncDecl`
bodies **only** — the exact walk `deriveRefusalArms` was extended past in round 1,
200 lines above it, left unextended here.

| # | Shape | Result |
| --- | --- | --- |
| W15 | **CONTROL** — `errors.New("planted plain refusal")` inside a `FuncDecl` body | **RED** |
| W5b | the **same string**, in a package-level `var errPlanted = errors.New(…)`, returned from `validateProtocolVersions` | **GREEN** |
| W9b | same, `fmt.Errorf` | **GREEN** |
| W16 | plain error inside a package-level `var f = func() error { … }` | **GREEN** |
| W12 | `errors.New` in a **new** production file's `FuncDecl` | RED |

Identical text, identical production reachability, opposite verdicts, decided purely
by declaration position. **2 of 6 plain-error attacks killed.**

The file's claim is a closure claim, in its own words:

> `TestPlainErrorsLiveOnlyInTheirTwoFunctions` closes the smuggling hole: a non-wire
> error constructed **anywhere** but validatePlatforms … or DigestFile … fails, so a
> refusal wearing a plain error's clothes **cannot ship unwitnessed**.

W5b ships one and the suite stays green, with `gofmt`, `go vet` and all 14 packages
clean. The "Stated bounds" block asserts the same property ("They live only in
validatePlatforms … and DigestFile … Both properties are asserted below").

Severity is below round-1's F1 — a plain error carries no wire code, so this is not
a forged wire refusal — but it is the failure this file exists to prevent, for the
half of it that plain errors own.

**Fix: the same ~12-line `GenDecl`/`ValueSpec` loop already written above it**, so
one walk covers both guards.

---

## G-B — the whole Story, re-measured

Production files are **byte-identical** between rev 1 and rev 2 (`manifest.go`,
`terminalbackend.go`, `conformance.go`, `internal/config/validation.go` all hash
equal), so every battery result below measures the *test* delta.

| Battery | Applied | Killed | Survived | Anchor missing | vs round 1 |
| --- | ---: | ---: | ---: | ---: | --- |
| Leaf 1 review-3 (71) | 70 | 61 | 9 | 1 (N12) | **identical** |
| Leaf 1 review-2 (60) | 59 | 54 | 5 | 1 (N12) | **identical** |
| Leaf 2 battery-1 (60) | 58 | 58 | **0** | 2 | **identical** |
| Leaf 2 battery-2 (47) | 47 | 47 | **0** | 0 | **identical** |

**Zero resurrections.** Survivor sets reproduce element for element
(N09/N13/N14/N15/T01/X01/X02/X04/X10 and N09/N13/N14/N15/T01).

**T01 re-verified directly.** I applied the self-minted-declaration mutant to
`ownership.v0.5.0.json` and ran the traceability packages: **KILLED**, eleven tests
red including
`TestVerifyRepositoryRejectsForgedOwnershipAndCapabilityClaims/valid_declaration_cannot_self-mint_ownership`.
This leaf's `+116`-line ownership delta has not weakened that gate.

Note for whoever runs these next: the battery runner restores only the four Go
files, so a T01 run leaves `ownership.v0.5.0.json` mutated. I caught it against the
tree OID and restored it; the candidate carries no residue from round 1 either.

---

## This leaf's own delta — my battery, 46 mutants

The P-series from round 1 was the reviewer's; I built my own, larger and
independent. **Every mutant preserves every refusal literal**, so the arm inventory
cannot kill any of them by construction — the kill-inflation bound is neutralised by
design and only behavioural tests can score.

**34 of 46 killed (74%). 12 survivors, in four clusters.**

Killed, worth naming because they are the AC surfaces: C01 error-mapping widening;
C02/C03/C04/C05/C06 the four attach-authorization conjuncts and the relay refusal;
C07/C08b/C09 the `ax pane` entrypoint (argv shape, argv[0], session binding);
C10/C11/C12 idempotency key shape and arity; C14/C15 status provider observation and
the canonical false form; C16 replication class; C17 ledger rebind conflict;
C18/C19 ledger image duplicate key and empty result; C20 legacy escape name;
C21/C24 transition create resolution and instance scope; C22/C23 attach policy exact
members and issue/expiry order; C25 legacy reverse projection grammar; C34–C39 six
transition-table widenings (sources, effects, target); C40b/C40d two replication
reclassifications.

### G1 (blocking) — attachability is a two-state rule measured at one excluded state

`CheckStatusResult`, `internal/terminalbackend/conformance.go:770`:

```go
if result.Attachable && !((result.State == StateParked || result.State == StateActive) && result.AttachEvidenced) {
```

The instance-state domain is a **closed eight-value enum** that `ParseInstanceState`
pins. The rule admits two. `TestCheckStatusResultEnforcesLookupRules` drives exactly
**one** of the six excluded states:

```go
stopped := matched
stopped.State = "stopped"
```

under the comment "attachable **outside parked/active** … is refused". Widening the
state set one member at a time:

| Mutant | Widened with | Result |
| --- | --- | --- |
| C30 | `StateStopped` | RED |
| C13 | `StateQuiescing` | **GREEN** |
| C26 | `StateCreating` | **GREEN** |
| C27 | `StateAbsent` | **GREEN** |
| C28 | `StateUnavailable` | **GREEN** |
| C29 | `StateStaleFenced` | **GREEN** |

**5 of 6 survive.** This is F2's shape on a domain that is not merely enumerable but
already enumerated in the same file, on the §4.C attach/status surface the AC names.
The fix is a loop over the eight states — three lines — and it closes the class
instead of adding a seventh sample.

### G2 (blocking) — the plain-error smuggling gate

W5b/W9b/W16 survive against the W15 control; detail in G-A above. Same file, same
fix already written 200 lines up.

### G3 (blocking) — the replication table has no reverse direction

`TestReplicationClassificationIsClosed` iterates its `want` map against production,
which proves **want ⊆ production**. Nothing proves production ⊆ want, and nothing
pins the table size.

| Mutant | Result |
| --- | --- |
| C40b — reclassify the existing `native_pid` from forbidden to safe evidence | RED |
| C40d — promote `attach_credential` from sensitive to safe evidence | RED |
| C40c — **add** `"ax_pane_pid"` to the safe-evidence list | **GREEN** |
| C40 — same, injected via `init()` | **GREEN** |

A member the reviewed table never listed becomes replicable and the suite stays
green. The doc comment says the test "pins the whole §4.E table"; it pins the half it
iterates. The unknown-member probe is a four-string sample
(`"", "session_record", "MANIFEST_ID", "manifest_id "`).

"Replication exclusions" is named in the AC verbatim. The opening paragraph of
`refusal_arm_inventory_test.go` diagnoses this exact failure mode for arms — "the
forward direction alone passes vacuously" — and the replication table is the same
shape without the second direction. A size pin from an internal test, or an AST
derivation in the file's own idiom, closes it.

### G4 (not blocking, but it must stop being stated as a closure)

Three vocabulary tests are named and commented as sweeps and are samples:

| Mutant | Result |
| --- | --- |
| C31 — `ParseInstanceState` admits `"running"` | RED (in the refused sample) |
| C44 — `ParseInstanceState` admits `"detached"` | **GREEN** |
| C32 — `ParseOperation` admits `"detach"` | **GREEN** |
| C43 — `ParseSideEffect` admits `"input_reopened"` | **GREEN** |
| C33 — `parseTransport` admits `"ssh_tunnel"` | **GREEN** |

C31 and C44 are the same mutation on the same test with a different string. The
domain is unbounded, so by round 1's own standard (P11) this is a **bound to
report**, not a blocker — except that it is currently stated as a closure three times
(`…AdmitsOnlyTheTenClosedOperations`; "sweeps the operation domain the same way: no
eleventh operation is admittable"), **and** the F2 rework cites one of them as its
pin:

> `conformanceOperations` … matching the ten operations ParseOperation admits
> (pinned by `TestParseOperationAdmitsOnlyTheTenClosedOperations`).

That citation does not hold. Either close it by AST derivation over the `switch` case
list — the repo's own idiom, the one `deriveRefusalArms` and
`grammar_inventory_test.go` already use — or restate all three as sampling bounds and
drop the "pinned by" clause.

### Bounds I am not blocking on, to be recorded

- **W11** — a new `errors.New` inside `DigestFile` survives. The allowlist is
  function-wide with no detail pin, unlike `validatePlatforms`, whose funneled detail
  set is derived and exact. Asymmetric; disclose or mirror.
- **W13** — a custom error type (`type plantedErr struct{}`) smuggled as a refusal
  survives: neither `*Error` nor one of the two recognised plain-error spellings.
  "A non-wire error constructed anywhere" does not cover it.
- **C45** — `allowedOperationErrors` widened with a **non-catalogued** code
  (`"terminal_backend_smuggled"`) survives. Already disclosed in-test as the
  unbounded-domain bound. **No action needed** — this is the honest version of G4.
- `mismatchf` is `func(format string, arguments ...any)`. A `mismatchf("x %s", v)`
  site would be inventoried under its format string while interpolating local data,
  which the constructor's own comment forbids. Zero such sites today (measured: all
  118 calls pass exactly one string literal); `go vet` catches the non-constant case.
  One assertion or one bound line.

---

## G-C — the bounds, and whether they reached the tree

Both round-1 bounds are **in the tree**, not only on the board:

| Bound | Where |
| --- | --- |
| Kill inflation — the inventory fails on any added/deleted arm, so post-leaf-3 scores are **not comparable** to leaf 2's 91/95; review the failing-test list, not the exit code | `refusal_arm_inventory_test.go` "Stated bounds" block, lines 34–40, **and** LOGBOOK |
| Attribution drift via the entry+code fallback, with `CheckTransition`'s dead `"operation vocabulary"` arm as the live example and the five re-parse arms as the contrast | same block, lines 25–33, **and** LOGBOOK |

Coverage figure corrected: LOGBOOK now records 89.9% on the rev-1 tree as my
predecessor measured it and 90.0% on this one. **I measure 90.0% on this tree.**
Accurate.

## Gates I ran myself

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | **exit 0**, 14/14 packages ok |
| `go test ./internal/terminalbackend/ -race -count=1` | **exit 0** |
| `go vet ./...` | exit 0 |
| `gofmt -l .` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=76`, `bindings=49 … clauses_discharged=17/403` |
| `go test ./internal/terminalbackend/ -cover` | **90.0%** |
| Candidate tree identity, before first probe and after last | `4400b106c826536d499d8e13a871c41af9764daa` ✓ |

Architecture fit is unchanged and good: the harness holds no authority, does no I/O,
leaves capability gating in `manifest.go` and trust in `terminalbackend.go`. The
stated bound from round 1 still applies — nothing outside `internal/terminalbackend`
calls these entry points, by design per §4.B, so "production entry point" here means
the harness's own exported surface.

---

## What acceptance needs

1. **G1** — derive the attachability refusal over all eight states instead of
   sampling `stopped`. Kill C13/C26/C27/C28/C29.
2. **G2** — extend `TestPlainErrorsLiveOnlyInTheirTwoFunctions` past `FuncDecl`
   bodies (the `GenDecl`/`ValueSpec` loop already in this file). Kill W5b/W9b/W16 and
   keep W15 red.
3. **G3** — give the replication table its reverse direction, or pin its size. Kill
   C40c.
4. **G4** — close the three vocabulary sweeps by derivation, or restate them as
   sampling bounds **and** drop the "pinned by
   `TestParseOperationAdmitsOnlyTheTenClosedOperations`" citation from
   `conformanceOperations`.
5. Record W11, W13 and the `mismatchf` format-verb note as bounds. Drop or restate the
   now-stale P11 "declined closure" bound — C20 is killed.
6. Report the conformance survivor set as a ratio, including anything declined.

Everything from round 1 is genuinely closed and the Story's other two leaves
re-measure clean against this tree. These four are the same family, on surfaces the
AC names, and each is a few lines. Once they land, this leaf is ready.
