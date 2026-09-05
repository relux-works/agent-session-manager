# TASK-260830-1snnef review verdict — round 5

**Verdict: changes requested.** Three findings, all in test/doc evidence, none
in production behavior. The four defects this round was convened for (B1, B2,
B3, and B4's half) are **all genuinely closed** and I killed each of them with
my own mutants rather than reading the rework note. What blocks acceptance is
narrower: two refusal arms ship with no test that fails when they admit what
they must reject, and the README states a guarantee the inventory it describes
explicitly disclaims.

The fixes are three fixtures and one sentence. Everything else in this leaf is
verified below so the next round is a re-check, not another full sweep.

CR `CR-TASK-260830-1snnef-5` rev 5, base `57afcc6`, candidate tree
`78cb3b2c5dcf4b32454770617a6545d534680a0d`. I verified the working tree equals
that OID before the first probe and again after the last of **264 mutants /
382 measured test executions** (the shortfall is mutants that never reach a
test run: 12 compile-fail, 6 anchor-bad, 1 build-error, 1 not-applied). `gofmt -l` clean, `go vet ./...` clean, `go test ./...`
green.

---

## Findings

### F1 (blocking) — `checkGeneration`'s UTF-8 validity arm has no negative test

`terminalbackend.go:606` refuses on three conjuncts. Drop the first one:

```go
if utf8.RuneCountInString(generation) < 1 || utf8.RuneCountInString(generation) > maxGenerationRunes {
```

**S11: GREEN 3/3.** The whole package suite passes with the validity arm gone.

That is not an equivalent mutant. `CheckProviderDescriptor` takes Go
`InstanceBinding` structs from a caller, not JSON-decoded values, so an
invalid-UTF-8 generation is representable and reachable. Without the conjunct,
`CheckProviderDescriptor` admits a descriptor whose `backend_generation` is
`"gen\xff"` on both sides: `utf8.RuneCountInString` counts the invalid byte as
one U+FFFD rune, so the length bound passes and the equality compare passes.
SPEC.md:321 bounds UTF-8 *characters* over a string; invalid UTF-8 is not one.

The asymmetry is the tell, and it is the same shape this round already fixed
for the surrogate gate. The identical conjunct in `GenerationDigest`
(`manifest.go:1410`) **is** covered — `S12: RED 3/3`, killed by
`TestGenerationDigestBounds`, whose refused set carries `"ok\xffbad"`. The copy
in `checkGeneration` has no such fixture:
`TestCheckProviderDescriptorGenerationBounds` refuses only `""`, 257 bytes and
257 two-byte runes, all of them valid UTF-8.

**Fix:** one entry in that test's `refused` table, e.g.
`{"invalid utf-8", "gen\xff"}`, expecting `terminal_backend_stale_generation`
at `backend_generation bound`.

### F2 (blocking) — the §7.A binding-digest **shape** arm dies only to the AST inventory

The fourth dimension landed with two arms sharing one detail clause:

| arm | site | mutant | result |
| --- | --- | --- | --- |
| #1 shape | `terminalbackend.go:587` `scalar.ParseDigest` | C1 delete | RED — **inventory only** |
| #2 equality | `terminalbackend.go:590` | C2 delete | RED — behavioral |
| #2 equality | — | C3 narrow to empty-only | RED — behavioral |
| #2 equality | — | C4 narrow to 7-char prefix | RED — behavioral |
| both | — | C5 delete the dimension | RED — behavioral |

C1's failing-test list is exactly `TestDerivedRefusalArmsAreAllDeclared` and
`TestDeclaredRefusalArmsAreAllDerived`. `TestCheckProviderDescriptorBindingDigestMismatch`
**stays green** with arm #1 deleted, because its three cases mutate only the
descriptor side against a valid `binding.TerminalBindingID`, so arm #2 catches
all three with the same detail string.

`refusal_arm_inventory_test.go` states this exact bound in its own header:

> Kill inflation: both inventory directions fail on ANY added or deleted
> production arm by construction, so an arm-deleting mutant elsewhere reads as
> killed by the inventory alone with no behavioral change. ... review the
> failing-test list, not just the exit code.

and

> a wrongly attributed row that still resolves is a review finding, not a
> green light.

I applied that rule rather than the exit code. The test's own comment — "the
shape case and the mismatch case below hit arm #1 and arm #2 respectively" —
is true of the unmutated control flow but is not something the suite can
detect; C1 is the proof.

The arm is not decorative. I wrote the distinguishing probe and ran it both
ways:

```go
binding.TerminalBindingID = "not-a-digest"   // same malformed value on BOTH sides
terminalbackend.CheckProviderDescriptor(binding, binding)
```

| probe | production | result |
| --- | --- | ---: |
| C6-control | unmutated | **GREEN 3/3** — refused, arm #1 does real work |
| C6 | arm #1 deleted | **RED 3/3** — admitted |

So the behavior is correct and unwitnessed. `terminal-provider-descriptor-7a`
in `ownership.v0.5.0.json` names that test as the acceptance evidence for
`CheckProviderDescriptor`; for arm #1 the named evidence does not discriminate.

**Fix:** one case in `TestCheckProviderDescriptorBindingDigestMismatch` that
sets the malformed digest on the **binding** side as well as the descriptor
side, so the equality arm cannot fire.

### F3 (blocking, one sentence) — README claims a guarantee the inventory disclaims

`README.md` (new text this round):

> an AST-derived refusal-arm inventory (`refusal_arm_inventory_test.go`) that
> requires every production refusal arm to resolve to a named asserting test
> in both directions, **so a refusal that ships unwitnessed fails the suite**

The clause before the comma is accurate. The clause after it is not what the
inventory guarantees, and the inventory says so itself: resolution is
*textual*, and "a wrongly attributed row that still resolves is a review
finding, not a green light". F2 is the live counterexample — arm #1 ships
resolved and unwitnessed, and the suite is green.

**Fix:** state what the gate measures — every arm is *declared* with a
resolving named test — and keep the witness claim out, matching the file's own
stated bound.

---

## What I verified and accept

### G-A — the units fix is real, and the four-site set is complete

**Site-local, one site at a time.** The killer the brief named
(`RuneCountInString` → `len`) dies independently at each of the four converted
sites, each on a multibyte fixture:

| # | site | mutant | result | killed by |
| --- | --- | --- | ---: | --- |
| A1 | `terminalbackend.go:606` `checkGeneration` | → `len` | **RED 3/3** | `…GenerationBounds/256_two-byte_runes` |
| A2 | `manifest.go:1410` `GenerationDigest` | → `len` | **RED 3/3** | `TestGenerationDigestBounds` |
| A3 | `manifest.go:465` `boundedStringMember` | → `len` | **RED 3/3** | `TestMultibyteStringBounds` |
| A4 | `manifest.go:2043` provider_build | → `len` | **RED 3/3** | `TestMultibyteStringBounds` |

Both sides are multibyte where it discriminates. `GenerationDigest` accepts 256
× `é` and refuses 257 × `é`; `os_version` and `provider_build` do the same
through `ParseProbe`/`ParseEvidence` with the identity re-sealed after mutation
so only the bound can decide; `CheckProviderDescriptor` admits `256 two-byte
runes` and refuses `257 two-byte runes` with the same generation on both sides
so the mismatch comparison cannot fire.

**Bounds narrowed and widened, not only deleted** — nine more, all site-local
rather than through the shared constant, so each site answers for itself:

| # | shape | result |
| --- | --- | ---: |
| S1 | `boundedStringMember` upper bound deleted (M45 re-anchored) | **RED 3/3** |
| S2 | `boundedStringMember` upper widened 256 → 257 | **RED 3/3** |
| S3 | `boundedStringMember` upper narrowed 256 → 255 | **RED 3/3** |
| S4 | `boundedStringMember` lower bound deleted | **RED 3/3** |
| S5 | provider_build upper widened 256 → 257 | **RED 3/3** |
| S6 | provider_build upper narrowed 256 → 255 | **RED 3/3** |
| S7 | provider_build bound removed entirely | **RED 3/3** |
| S8 | `GenerationDigest` upper widened (site-local) | **RED 3/3** |
| S9 | `checkGeneration` upper widened (site-local) | **RED 3/3** |

Plus the re-anchored inherited generation-bound mutants, which had lost their
`maxGenerationBytes` / `len(generation)` anchors to this round's rename:
**N03r, N04r, N05r, R06r, R07r — all RED**, restoring leaf 1 to its round-4
score exactly.

**The set of four is complete — derived, not trusted.** I enumerated the
`string[n..m]` members of every schema this package parses from SPEC.md rather
than from the report's list:

- Manifest 1.0.0 (SPEC.md:900-912): no `string[n..m]` member.
- Probe 1.0.0 (SPEC.md:919-935): `os_version` only → `boundedStringMember`.
- Capability Evidence 1.0.0 (SPEC.md:1226-1250): `os_version` and
  `provider_build` → `boundedStringMember` and `manifest.go:2043`.
- `TerminalBackendCapabilityClaim` / `CapabilityEvidenceFact` (SPEC.md:1192,
  1221): closed enums and booleans, no bounded string.
- Terminal Instance Binding 1.0.0 (SPEC.md:1031): `backend_generation` →
  `checkGeneration` and `GenerationDigest`.
- `terminal_backend_id` stays in bytes at `terminalbackend.go:131`, correct
  per SPEC.md:890 ("1–128 ASCII bytes"), and the comment at
  `terminalbackend.go:68` now says so explicitly.

**Two stated bounds, both disclosed rather than filed as findings:**

- `maxNativeReference = 512` (`manifest.go:259`) has no production reference.
  The Binding's `native_reference` (SPEC.md:1032) is outside the §7.A matched
  subset, which `InstanceBinding`'s doc comment scopes correctly, so this is a
  dead constant and not an unenforced bound. Worth deleting when the Binding
  document itself lands; not a defect here.
- **S10: GREEN 3/3** — dropping `utf8.ValidString` from `boundedStringMember`
  survives. This one *is* an equivalent mutant: every caller reaches it through
  `decodeStrictObject`, which checks `utf8.Valid(raw)` on the whole document
  first and then decodes through `encoding/json`, which cannot emit invalid
  UTF-8. Unreachable, so unkillable. That is exactly why F1 is a different
  answer for the same conjunct at a different site.

### G-B — the local surrogate gate is pinned to its original, in both directions, and it runs first

`TestSurrogateGateAgreesWithCanonicalJSON` (`internal_pin_test.go:282`) drives
`hasLoneSurrogateEscape` **and** `canonicaljson.Canonicalize` over ten shared
vectors and asserts both sides agree on **accept and reject**: four accept
vectors (plain, multibyte, non-surrogate escape, escaped backslash before `u`,
valid pair) and five reject vectors (lone high, lone low, high+text,
high+non-low escape, low-then-high). It states its bound: malformed escapes and
control characters belong to both sides' syntax arms and are outside the pin.
That is a pin, not a second implementation with one witness.

| # | shape | result |
| --- | --- | ---: |
| B1 | gate call deleted from `decodeStrictObject` | **RED 3/3** |
| B2 | narrowed: lone LOW surrogate admitted | **RED 3/3** |
| B3 | narrowed: unpaired HIGH surrogate admitted | **RED 3/3** |
| B4 | widened: a VALID surrogate pair now rejected | **RED 3/3** |

B2/B3 are narrowings and B4 is a widening, so the pin is bounded on both
edges, not merely non-empty.

**Ordering — both halves of the MUST.** The gate is the second statement of
`decodeStrictObject` (`manifest.go:354`), before `json.NewDecoder`, and all four
raw-byte entry points (`ParseManifest`, `ParseProbe`, `ParseEvidence`,
`ParseAttachAuthorization`) route through it; `ImportLedger` is the only other
byte entry and is not JSON. `jcs.Transform` is reached only later, from
`checkIdentity`.

The ordering is *witnessed*, not merely structural. Under B1 the injected
documents are refused at `document identity binding` instead of
`document surrogate escape` — i.e. without the gate the document reaches
canonicalization, which is precisely the violation SPEC.md:289 forbids. So
`TestDocumentSurrogateEscapeRefused` asserting the detail string is the
ordering pin.

**B5: GREEN 3/3, expected and recorded as a bound.** Moving the gate to after
`decodeCappedValue` (still inside `decodeStrictObject`, still before any
canonicalization) is green, correctly: it does not violate the MUST, so nothing
should redden. I ran it to confirm the suite is not accidentally pinning a
stronger ordering than the spec requires. It is not.

### G-C — the fourth dimension, its registration, and the doc comment

The dimension is present: `InstanceBinding.TerminalBindingID`
(`terminalbackend.go:566`), checked at `terminalbackend.go:587` and `:590`, and
C2/C3/C4/C5 all die behaviorally. F2 above is the one gap.

**The doc comment now states all four.** SPEC.md:3473 reads "binding digest,
backend ID/version, or generation"; the comment at `terminalbackend.go:569`
reads "binding digest, backend ID, versions, or generation ... before launching
or observing a provider process". Nothing elided.

**B4's half is closed.** `terminal-provider-descriptor-7a` is a registered
acceptance case in `ownership.v0.5.0.json` naming `CheckProviderDescriptor`
with four tests; the count moves 74 → 77 and the README, `Report`, and
`tracecheck` golden all move with it. The `section:7.A` binding still reads
`unevidenced` with "no clause of 7.A is enumerated", which is honest and
uniform — §4.B, §4.C and §4.D carry the identical shape, and the sentence is a
claim about clause enumeration, not about code existence.

`reviewedOwnershipCanonicalSHA256` moved `c70c86d2…` → `8281d96b…`, and it
fails closed. Three hand-applied attacks on the registry, none re-pinning the
digest:

| # | attack | result |
| --- | --- | ---: |
| T01r | extra unreviewed test declaration added (additive; the harness cannot apply it, so by hand) | **KILLED** |
| T02r | `terminal-provider-descriptor-7a` production declaration swapped to `CheckVersionTuple` | **KILLED** |
| T03r | the binding-digest negative test silently swapped out of the acceptance case | **KILLED** |

`ownership.v0.5.0.json` restored byte-identical after each.

### G-D — resurrections: none

Every inherited battery re-run against tree `78cb3b2c`, tree OID re-verified
after each.

| battery | scope | round 4 | round 5 |
| --- | --- | --- | --- |
| conformance (bat1+2+3) | 48 mutants ×3 | 46 killed / 1 survived / 1 unmeasurable | **identical** |
| leaf 2 bat1 | 60 mutants | 46 killed / 12 compile-fail / 2 anchor-bad | 44 killed / 12 / 4 → **46 after re-anchoring M14r + S1** |
| leaf 2 bat2 | 47 mutants | 47 killed | **47 killed** |
| leaf 1 | 71 mutants | 60 killed / 8 survived / 2 n-a / 1 build-error | 55 / 8 / 7 / 1 → **60 after re-anchor** |
| round-5 new | 29 mutants ×3 + 3 by hand | — | 26 RED, 2 intended-GREEN controls (B5, C6-control), 1 equivalent mutant (S10), 1 finding (S11) |

**Zero resurrections. Zero regressions. Zero flaky mutants** — no mutant
produced a mixed verdict across its three repeats anywhere.

The seven verdict changes are all anchor drift from this round's own edits, and
every one of them is killed once re-anchored:

- leaf 1 N03/N04/N05/R06/R07 — anchored on `maxGenerationBytes` and
  `len(generation)`, renamed by the B2 fix. Re-anchored as N03r–R07r:
  **all RED**.
- leaf 2 M14/M45 — anchored on `len(rawGeneration)` / `len(value)`, same
  cause. Re-anchored as M14r and S1: **both RED**.

Survivor sets reproduce id-for-id and none is new:
leaf 1 `N09, N13, N14, N15, X01, X02, X04, X10`; conformance `C45`
(`allowedOperationErrors` widened with a non-catalogued code — the honest
unbounded-domain bound, disclosed in-test); leaf 2 none.
Unmeasurable reproduce too: leaf 1 `M32` (build error), `N12`/`T01` (not
applicable to the harness; T01 applied by hand above and killed), conformance
`C08` (non-compiling; `C08b` is the compiling variant and is RED).

### Also checked

- `internal/config/validation.go` now delegates to `terminalbackend.ParseID`
  for both `terminal.backend_id` and each `external_trust` entry, plus a
  separate `ax.` bar on trust entries. `ParseID` admits `ax.tmux`/`ax.conpty`
  and refuses every other `ax.` ID, so the built-ins still configure. M25, M27,
  N16, X08, X09, X11 all RED.
- Advisories from `.temp/fable-consolidated-review.md` (dual identity
  pipelines, the unowned Provider Protocol 3.0.0 envelope, the hand-rolled AST
  walkers, the `provider.trustCandidate` asymmetry) are tracked as
  `STORY-260905-3t31e9`. I did not expand into them, and **none of them turned
  out to be blocking** for this leaf.

---

## Routing

`to-dev`. F1 and F2 are one test fixture each and F3 is one sentence; nothing
in production changes. On the next round I only need to re-run S11, C1 and C6
plus the package suite — the sweep above does not need repeating unless
production moves.

## Artifacts

`.temp/TASK-260830-1snnef-r5/` — batteries, harness, and per-mutant JSON:
`r5bat.py` / `r5bat-results.json`, `r5bat2.py` / `r5bat2-results.json`,
`bat{1,2,3}-r5.json`, `leaf1-all-r5.json`, `leaf1-reanchor-out.json`,
`leaf2-bat{1,2}-out.json`, `probe.py`, `harness3.py`.
