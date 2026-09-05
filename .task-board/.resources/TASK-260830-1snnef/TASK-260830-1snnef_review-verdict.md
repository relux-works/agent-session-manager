# TASK-260830-1snnef review verdict — round 1

**Verdict: changes requested → `to-dev`.**

CR `CR-TASK-260830-1snnef-1` rev 1, base `57afcc6`, candidate tree
`f9e66a3c8ea25af155356ba24e2c0e815eef49d3`. I verified the tree I measured **is**
the candidate: after every probe below, a scratch-index `git write-tree` returned
`f9e66a3c8ea25af155356ba24e2c0e815eef49d3` exactly.

The leaf is strong. The derived inventory is real, load-bearing, and fails closed
in the ways the round-3 verdict asked for. Four findings block acceptance, all
small and all in the same class: a claim measured by a sample presented as a
closure.

---

## G-A — attacking the attacker

### The ratio, denominator established independently

The derivation under test walks `*.go` (non-test) `FuncDecl` bodies. I built the
denominator two ways that never touch it:

| Method | What it counts | Result |
| --- | --- | ---: |
| Separate Go AST program: whole-file `ast.Inspect`, position-span exclusion of the two constructors | `&Error{}` composite literals + direct `mismatchf`/`integrityFailure` calls | 75 + 118 = **193** |
| Regex scan of `Detail: "…"` and constructor string args | literal-detail sites (blind to the dynamic funnel) | 192 + 1 funnel = **193** |
| The inventory's declared table | `declaredRefusalArms` rows + funnel row | 192 + 1 = **193** |

**193 of 193 arms derived and declared — 100%**, with the denominator agreeing
across three independent methods. The same program reported
`outsideFuncDecl=0` and constructor identifier occurrences `= calls + 2
declarations`, i.e. the two blind spots below are **currently empty**, measured,
not assumed.

### Nine probes against the derivation

Copy-aside, `shasum`-verified restore after each.

| # | Shape | Expectation | Result |
| --- | --- | --- | --- |
| A1 | arm in a package-level `var f = func() *Error {…}`, called from `CheckReplicable` | should fail | **SURVIVED — blind spot** |
| A2 | `refuse := mismatchf; return refuse("planted alias arm")` | should fail | **SURVIVED — blind spot** |
| A3 | arm in a brand-new production file in the package | should fail | KILLED (both directions) |
| A4 | arm in a func literal *nested inside* a `FuncDecl` body | should fail | KILLED (both directions) |
| A5 | arm on a method of a new type | should fail | KILLED (both directions) |
| A6 | `Detail:` from a package const identifier | should fail closed | FATAL, "not a static clause" |
| A7 | delete an existing arm | reverse direction | KILLED |
| A8 | rename an existing detail | both directions | KILLED |
| A9 | arms built through a **new** helper not in `constructorFunctions` | should fail closed | FATAL, "not a static clause" |

7 of 9 killed or fail-closed. A9 is the important positive: a producer cannot
route refusals through a fresh helper to hide them — the helper's own dynamic
`Detail` fatals the derivation.

### Is the declared side a closed loop against itself?

No, and this is worth saying plainly because the sibling Story's round 2 found
exactly that failure. The table is hand-written, but it is anchored to **three**
things the table cannot edit: the derivation (bijection, both directions), the
production `Code` const values (`TestDeclaredRefusalWiresMatchProductionConsts`),
and real test bodies indexed from the test AST (`TestDeclaredRefusalTestsResolve`).
Truncating the derivation fails the reverse direction (A3, A7 confirm). The
numerator and denominator do not come from the same literal.

The residual weakness is different and narrower: when the derivation is **blind**
to a shape, both directions stay silent because neither side ever sees the arm.
A1 and A2 are that case.

### Does it cover `conformance.go`'s 989 new lines?

Yes. The glob is file-level and A3 proves a new file is scanned. Of the 193 arms,
40 live in `conformance.go` (34 `&Error{}` + 6 `mismatchf`), all declared.

---

## G-A finding (blocking)

### F1 — README states a capability two probes falsify

`README.md:1983` (new text):

> "an AST-derived refusal-arm inventory (`refusal_arm_inventory_test.go`) that
> requires **every** production refusal arm to resolve to a named asserting test
> in both directions, **so a refusal that ships unwitnessed fails the suite**"

A1 and A2 each ship a reachable production refusal arm with a brand-new detail
and leave both inventory tests, `go vet`, `gofmt` and all 14 packages green. The
claim is a closure claim; the guard covers a subset. The file's own "Stated
bounds" block lists three bounds and does not name this one, while its
`deriveRefusalArms` comment advertises the file-level closure ("so a new
production file is scanned without anyone remembering it") without the
declaration-level bound beside it.

DoD item: "README/doctor/capability evidence … updated **without unsupported
claims**."

Fix is cheap and I'd prefer the real one over a caveat: walk `file.Decls` for
`GenDecl`/`ValueSpec` func literals as well (or `ast.Inspect` the whole file with
a constructor-span exclusion, as my audit program does), and fatal when
`mismatchf`/`integrityFailure` appears as an `Ident` outside call position. Both
are ~10 lines and both blind spots are empty today, so neither changes the table.

---

## G-B — the whole Story, re-run against this tree

I re-ran **all four inherited batteries** against this exact candidate.

| Battery | Applied | Killed | Survived | Anchor missing |
| --- | ---: | ---: | ---: | ---: |
| Leaf 1 review-3 (`review3-mutant-definitions.json`, 71) | 70 | 61 | 9 | 1 (N12) |
| Leaf 1 review-2 (`review2-mutant-definitions.json`, 60) | 59 | 54 | 5 | 1 (N12) |
| Leaf 2 battery-1 (60) | 58 | 58 | **0** | 2 |
| Leaf 2 battery-2 (47) | 47 | 47 | **0** | 0 |

**Zero resurrections.** Every mutant leaf 1 and leaf 2 recorded as KILLED is
still KILLED here. Two improvements: leaf 1's declared survivor **X03** is now
killed, and leaf 2's declared survivor **M92** is genuinely killed by this leaf's
new `TestAdmitProbeRefusesNilRegistry` (I verified the failing-test list, not just
the exit code).

Leaf 1's survivor set reproduces exactly: N09, N13, N14, N15, X01, X02, X04, X10.
`T01` was "not applied" in leaf 1's record; its anchor applies now, and when the
traceability packages are included it is **KILLED** by six `tracecheck` tests —
so this leaf's +57-line `ownership.v0.5.0.json` change has not weakened the
self-minted-declaration gate.

### G-B bound worth recording: the inventory inflates future kill counts

M71, M72 and M95 report as KILLED in a re-run **solely** because the two inventory
tests fail — the failing-test list contains nothing else. The inventory kills any
arm-deleting mutant by construction, so leaf-2-style mutation scores measured
after this leaf are not comparable to leaf 2's 91/95. Nothing behavioral changed
for those three. The producer's own bounds statement is honest here ("M71/M72 …
unknown; M92/M95 stay survivors") — credit — but the tree carries no warning for
the next person who re-runs a battery.

---

## G-B finding (blocking) — the harness in the permissive direction

`conformance.go` asserts things about other code. Twelve narrowing mutants on it,
copy-aside, restore-verified. Nine killed. Three survivors, all on gates the AC
names directly, none declared in the leaf's evidence:

### F2 — `CheckErrorAllowed`: the refusal set is a sample, not a complement

Mutant P20: admit `terminal_backend_unavailable` for **every** operation, ahead of
the table lookup. **Whole suite green.** The production table omits
`CodeUnavailable` for `quiesce-input`, `wait-safe-boundary`, `request-stop` and
`terminate-stale`, so the mutant makes the harness pass a backend emitting an
error code the contract forbids for that operation.

`TestCheckErrorAllowedRefusesEveryUnlistedCode` hand-picks 3–5 refused codes per
operation. The name claims "every unlisted code"; it measures a sample. The
package already pins the catalog error vocabulary, so the complement is derivable:
for each operation, assert every catalogued code **not** in its allowed set is
refused. That is the difference between a sample and a closure, and this is a
conformance harness — permissive is the direction that matters.

### F3 — `TestCheckAttachResultRequiresTripleEquality` measures a pair

Mutant P8: drop the `resultInputAuthorized != auth.InputAuthorized` conjunct.
**Whole suite green.** The test builds one `auth` with `InputAuthorized: true`
and checks `(false,true)` and `(true,false)`; the case only the third leg catches
— `(false,false)` against `auth{true}` — is never driven. The test name asserts
the property; nothing measures it. One line fixes it.

### F4 — attach expiry boundary instant unwitnessed

Mutant P16: `!now.Before(expires)` → `now.After(expires)`, i.e. admit attach at
exactly the expiry instant. **Whole suite green.** The expiry case uses
`now + 48h`. Production chose refuse-at-`expires`; nothing pins that choice, so
the boundary can flip silently. One case at `now == expires` fixes it.

### Survivors I am **not** blocking on

- **P11** — `TranslateLegacyBackend` with an extra escape name (`"screen"`)
  survives; the refusal sample is 10 names over an unbounded string domain. Not
  closable by sampling; the closed-set property (`legacyForward` has exactly two
  entries) is the right pin if anyone wants it. Reported as a bound.
- **P13** — `ProjectToLegacy` skipping `ParseID` for the empty ID still refuses,
  with a different wire code. Near-equivalent.

Killed in the same battery: P1 (drop an op from `idempotencyKeySegments` — the
empty-key permissive path), P4/P5/P7/P10/P14/P15/P17/P18/P19. P15 (`ImportLedger`
treating a malformed image as an empty table — the failed-read-as-absence shape)
is killed both behaviorally **and** by both inventory directions.

---

## G-C — bounds

| Bound | Status after this leaf |
| --- | --- |
| M71/M72 (JCS) unprovable, reported **unknown** | Unchanged behaviorally. They now *read* as killed in a re-run, by the inventory alone. See the G-B bound above. |
| M92/M95 counted as survivors | **M92 is now genuinely killed** (`TestAdmitProbeRefusesNilRegistry`). M95 unchanged behaviorally; reads as killed by the inventory alone. |
| Leaf 2: 19 of 88 executed sites share a detail (69 file-unique) | **Changed, and got worse in absolute terms.** Over all 192 literal-detail arm sites: 20 distinct details cover 82 sites; 110 sites are detail-unique. `conformance.go` adds repeats deliberately (`entrypoint session binding` ×3, `idempotency ledger image` ×4, `idempotency key shape` ×4 across two functions). The occurrence index keeps *rows* distinct; textual test attribution still cannot separate site #1 from #2 in one function, which the file discloses. |

### Does the derived inventory make leaf 2's "five vs six" drift impossible?

**For that exact statistic, yes — it is now derived, not hand-counted.** I
measured it independently: of 130 distinct details across 192 sites, **zero** have
no textual hit in any test file. That class of manual-enumeration drift is closed.

**A weaker cousin survives.** `rowResolves` falls back to entry-mention +
wire-code/predicate-mention when the detail is not mentioned, so a row can resolve
while pointing at a test that exercises a *sibling* site sharing its detail. Live
example: `CheckTransition`'s `"operation vocabulary"` arm is unreachable
(`ParseOperation` admits exactly the ten operations `transitionTable` covers), it
carries no `bound`, and it resolves through a test that drives `ParseOperation`'s
own arm with the same string. Compare the five re-parse arms, which *do* declare
`boundDefensiveReparse` and are pinned as exactly five by
`TestDefensiveBoundsAreExactlyThese`. Same dead-by-construction class, different
treatment. Not blocking; worth one row or one bound line.

---

## Gates I ran myself

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | **exit 0**, 14/14 packages ok |
| `go test ./internal/terminalbackend/ -race -count=1` | **exit 0** |
| `go vet ./...` | exit 0 |
| `gofmt -l .` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=76`, `bindings=49 … clauses_discharged=17/403` (section coverage unchanged — the ownership delta claims one acceptance case and no new section binding) |
| `go test ./internal/terminalbackend/ -cover` | **89.9%** |
| Candidate tree identity after all probes | `f9e66a3c8ea25af155356ba24e2c0e815eef49d3` ✓ |

Nit, not a finding: the leaf's evidence and LOGBOOK say "90.0% of statements"; I
measure 89.9% on this tree. Round it or restate it.

Architecture fit is good: the harness holds no authority, does no I/O, keeps
capability gating in `manifest.go` and trust in `terminalbackend.go`, and the
ownership delta adds one acceptance case with twelve real test declarations and
claims no new section binding. Stated bound: nothing outside
`internal/terminalbackend` calls these entry points — by design per §4.B, and
correct for M0, but it does mean "production entry point" here means the harness's
own exported surface, not an `ax` call site.

---

## What acceptance needs

1. **F1** — close the two derivation blind spots (preferred) or state them in the
   "Stated bounds" block **and** qualify the README sentence. The README claim as
   written is falsified by A1/A2.
2. **F2** — derive `CheckErrorAllowed`'s refusal cases as the complement of each
   operation's allowed set over the pinned catalog vocabulary, and kill P20.
3. **F3** — add `CheckAttachResult(false, false, auth{InputAuthorized: true})`
   and kill P8.
4. **F4** — add the `now == expires` attach case and kill P16.
5. Report the survivor set for `conformance.go` as a ratio, including anything you
   decline to close (P11 and P13 are fine as declared bounds).

Optional, cheap, and it would make the next re-run honest: one bound line noting
that the inventory kills every arm-deleting mutant by construction, so post-leaf-3
mutation scores are not comparable to leaf 2's 91/95.

The Story's other two leaves re-measure clean against this tree. Once these four
land, this leaf is ready.
# TASK-260830-1snnef — review round 7 (republication round) — ACCEPT

CR `CR-TASK-260830-1snnef-8` revision 8. Base `57afcc6d`, candidate tree
`771f9c53fb708e5b401df371899c0d316bfd8e61`.

Verdict: **accepted** via `accept_cr`. `repeat-of: none` — republication, not rework.

## 1. Tree unchanged from the round-6 accepted candidate

- `TASK-260830-1snnef_change-request_rev7.patch` and
  `..._rev8.patch` are byte-identical: same sha256
  `8a7e7e5ab85b99df478d5121c1a9c00c5031a9c36eab1ca0a6dabb97cd311593`
  (`cmp` clean).
- Independently re-derived: applied the rev8 patch onto base OID
  `57afcc6dc5019672780baad393a0cef4873871b9` in a scratch repo
  (`read-tree` + `apply --index` + `write-tree`) → tree
  `771f9c53fb708e5b401df371899c0d316bfd8e61`, exactly the handed
  candidate OID and verbatim the tree measured in round 6
  (`TASK-260830-1snnef_review6-verdict.md`, rev-7 candidate).
- Delta shape: `repository_delta=present`, 17 changed paths — matches the
  handoff, so no `empty`-delta question arises.

No re-measurement of an identical tree: the round-6 evidence (derived
surrogate sweep over all 2048 code points in both positions, five boundary
mutants red with refusal counts matching each hole's exact width, valid
pairs through the production decoder, WTF-8 arm by detail string, 277
mutants across ten batteries, zero resurrections, AC coverage 5/5 with
named production call sites) carries over verbatim.

## 2. Revision 8 binding is complete and genuinely produced

- Producer run `RUN-260905-14f244`: `task-board spawn status` reports
  `Status: completed`, `Completion: success`, exit 0, and agent identity
  `[implementer] developer (muse)` — i.e. role `developer`, archetype
  `implementer`. The binding is recorded by the spawn system for that run
  itself, not transcribed from anywhere; the run completed 14:02–14:10Z,
  after the 13:10:45Z tooling rebuild, through the current publisher path
  that refuses without all three fields.
- Revision 8 state is `ready` against element `TASK-260830-1snnef`.

## 3. On revision 4's null-binding acceptance (judgement asked)

Nothing unsound remains in the accepted chain, and the binding requirement
is forward-only by design. What round-6's predecessor accepted at 06:12 was
content — a tree, evidence, and a reviewer stamp — and that content is
unchanged and still the thing being accepted now. `ProducerRole`/
`ProducerArchetype` are accept-time provenance metadata enforced by the new
binary; they do not retroactively alter what was reviewed or what the tree
contains. A backfill migration for pre-upgrade records is a tooling-owner
decision and is not required for the soundness of this leaf: revision 8
now carries the complete binding going forward, and that is sufficient.
