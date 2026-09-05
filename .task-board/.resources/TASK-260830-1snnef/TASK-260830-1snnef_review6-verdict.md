# TASK-260830-1snnef — review round 6 — ACCEPTED

CR `CR-TASK-260830-1snnef-7` revision 7. Base `57afcc6d`, candidate tree
`771f9c53`. Working tree re-derived through an isolated index at the start and
after every mutant battery: `771f9c53fb708e5b401df371899c0d316bfd8e61`, equal
to the declared candidate. `internal/traceability/ownership.v0.5.0.json`
md5 `a0855df114dab02dcdbd55f8e48a0145` before and after the hand-applied T01.

Verdict: **accepted**. Nothing blocking. One stated bound recorded below; it is
a corpus bound in a production-dead branch, not a bypass, and I measured the
production gate over its whole domain rather than inferring it.

## Gates

| Gate | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | empty |
| `go test ./... -count=1` | 14/14 packages ok |
| `go test ./internal/terminalbackend/ -cover` | 90.3% |

## G-A — the replacement corpus, judged as the corpus

**Derived, not a longer hand list.** `TestSurrogateGateAgreesWithCanonicalJSON`
(`internal_pin_test.go:297`) enumerates every code point in U+D800..U+DFFF as a
lone escape (2048 vectors) and every code point in that range as the *second*
unit after a fixed high surrogate (2048 vectors), asserting admit-iff-low. The
old 10-vector table is gone. Every dropped vector is behaviourally subsumed:
`"\ud800abc"` and `"\ud800"` traverse the same `!ok` path in
`readUTF16EscapeUnit`, and `"\udc00\ud800"` returns at the same lone-low arm as
`"\udc00"`.

**Both bounds, both directions.** I mutated each of the four numeric bounds in
the escape arm and each in both directions where meaningful. Every one is RED,
and the failure counts are the arithmetic you would predict from the width of
the hole, which is the check that the sweep is actually enumerating rather than
tripping on one lucky vector:

| Mutant | Site | Result |
| --- | --- | ---: |
| low upper `0xdfff` → `0xdc00` | `manifest.go:307` | RED, 1023 failures |
| high upper `0xdbff` → `0xd900` | `manifest.go:302` | RED, 768 |
| high lower `0xd800` → `0xd900` | `manifest.go:302` | RED, 1543 |
| pair-continuation upper `>0xdfff` → `>0xdc00` | `manifest.go:304` | RED, 1024 |
| pair-continuation lower `<0xdc00` → `<0xdd00` | `manifest.go:304` | RED, 257 |

The high-surrogate bound you had not measured is killed from both sides.

**Valid pairs still admit — the gate is not one-sided.** Driven through the
production decoder `decodeStrictObject`, not the helper: escaped `😀`,
raw UTF-8 U+1F600, both corner pairs `𐀀` and `􏿿`, CJK, and
two adjacent pairs are all admitted. The complementary direction is already
pinned in the battery: r5bat `B4` (gate widened so a valid pair is rejected too)
is RED.

**The raw WTF-8 arm is the intended one and is witnessed.** `decodeStrictObject`
(`manifest.go:374`) runs `utf8.Valid(raw)` → `document encoding` *before*
`hasLoneSurrogateEscape` → `document surrogate escape`. Raw WTF-8 is not valid
UTF-8, so `document encoding` is correct and inevitable, not a slide.
The inventory carries that row at `manifest.go / decodeStrictObject /
CodeMismatch / "document encoding" #1`, witnessed by
`TestManifestEncodingRefusals`; `TestDocumentWTF8SurrogateRefused` asserts that
exact detail string at all three production entries (`ParseProbe`,
`ParseManifest`, `ParseEvidence`), so a slide to another arm reddens it. The
surrogate arm keeps its own separate witness on the escape road
(`TestDocumentSurrogateEscapeRefused`), and r5bat `B1` (surrogate call deleted
from `decodeStrictObject`) is RED through it. The two roads are not proving each
other.

**The agreement test now catches the disagreement it missed.** Disabling
`isWTF8SurrogateAt` entirely reddens `TestSurrogateGateAgreesWithCanonicalJSON`
with 6 agreement failures — the local gate admitting what `canonicaljson`
refuses. That is the exact defect you reported, now caught.

### Stated bound (not blocking)

Asked directly: the WTF-8 vector set inside that test is **hand-written**, not
derived — 4 positive encodings (ED A0 80, ED B0 80, ED BF BF, ED AF 93), one
negative neighbour (ED 9F BF), two backslash-adjacency cases. That is 4 of the
2048 encodings in the branch's domain. All four *bound* narrowings I threw at
`isWTF8SurrogateAt` die on those corners, but a **punch-out** mutant
(`raw[index+1] != 0xa1`, admitting the 64 encodings of U+D840..U+D87F) survives
the entire package. Same shape as the original finding, one branch over.

I did not route it as a finding, for two measured reasons rather than an
argument:

1. **The gate is right over its whole domain.** A derived reviewer sweep of all
   2048 encodings ED A0..BF 80..BF against the production predicate:
   `missed = 0 of 2048`. Against the neighbouring valid band ED 80..9F
   (U+D000..U+D7FF): `false positives = 0 of 2048`.
2. **The branch is production-dead, and I proved it rather than reading it.**
   Disabling `isWTF8SurrogateAt` fails *only* the white-box pin — no production
   test moves — because `utf8.Valid` has already refused every raw WTF-8 document
   at `document encoding`. leaf2 `M81` (delete that `utf8.Valid` arm) is RED, so
   the arm that actually closes the class is itself measured.

So the residual is: a defence-in-depth branch no production entry reaches,
whose corpus measures 0.2% of its domain while the branch itself is correct at
100%. Worth deriving the sweep the next time this file is opened; not worth a
seventh round. Logged, and flagged to the sibling Story since it carries the
same surrogate-gate copy.

## G-B — round 5's three findings

**F1 — confirmed.** `TestCheckProviderDescriptorGenerationBounds` gained
`{"invalid utf-8", "gen\xff"}`. Dropping the `!utf8.ValidString(generation)`
conjunct at `terminalbackend.go:606` now reddens
`TestCheckProviderDescriptorGenerationBounds/invalid_utf-8` and nothing else.
r5bat2 `S11` RED (was the finding), `S12` RED.

**F2 — confirmed, and the kill is behavioural, not inventory inflation.** The
new `malformed digest on both sides` subtest puts `"not-a-digest"` on the
descriptor **and** the binding, so the equality arm (#2) cannot fire on equal
values and only the `scalar.ParseDigest` shape arm (#1, `terminalbackend.go:587`)
can refuse. To keep the AST inventory out of the measurement I neutralised the
arm *without deleting it* (`err != nil && false`), leaving the arm count and
every literal intact: exactly one test fails,
`TestCheckProviderDescriptorBindingDigestMismatch/malformed_digest_on_both_sides`,
and both inventory directions stay green. Arm #1 now dies to behaviour.

**F3 — confirmed.** The README now says the inventory "requires every production
refusal arm to be **declared** with a resolving named asserting test in both
directions; behavioural proof stays with each row's named test, which the
inventory resolves textually". That is what it guarantees. The file's own header
states the same bound plus the kill-inflation warning.

## G-C — resurrections

Production moved in exactly one file since round 5 (`manifest.go`, +32 lines:
the WTF-8 branch); `terminalbackend.go` and `conformance.go` are byte-identical,
and the test deltas are additive except the pin corpus replacement analysed
above. I re-ran every inherited battery anyway.

| Battery | Round 5 | Round 6 |
| --- | --- | --- |
| leaf 2 bat1 (60, `manifest.go`) | 44 K / 0 S / 12 CF / 4 AB | **identical** |
| leaf 2 bat2 (47, `manifest.go`) | 47 K / 0 S | **47 K / 0 S** |
| leaf 1 (71, `terminalbackend.go`) | 60 K / 8 S / 3 n-a | **identical**, survivors id-for-id |
| leaf 1 re-anchor (N03r–R07r) | 5 RED | **5 RED** |
| conformance bat1+2+3 (48) | 46 K / 1 S / 1 unmeasurable | **identical** |
| round-5 new (r5bat/2/3, 29) | 26 RED, S10+B5 GREEN, S11 finding | **28 RED, S10+B5 GREEN** |
| T01 (hand, ownership registry) | KILLED | **KILLED** |

**Zero resurrections. Zero regressions.** Survivor sets reproduce id-for-id:
leaf 1 `N09, N13, N14, N15, X01, X02, X04, X10`; conformance `C45` (the
disclosed unbounded-domain bound); leaf 2 none; r5bat `S10` (equivalent mutant)
and `B5` (intended-GREEN control). Unmeasurable reproduce too: leaf 1 `M32`
(build error), `N12` and `N03/N04/N05` (anchor drift, all five re-anchored and
RED), `T01` (harness anchor; applied by hand and RED), conformance `C08`
(non-compiling; `C08b` RED).

`T01` is the one I re-ran by hand rather than trusting: it is the self-minted
acceptance-case declaration gate, and `ownership.v0.5.0.json` grew by 141 lines
in this Story. Adding an unreviewed test declaration is refused by tracecheck
across every section with *"declaration ... is absent from ..."* — a failed
lookup reported as a failure, not silently as an absence. File restored
byte-identical.

277 mutant rows measured this round across ten batteries.

## Not mine

The four structural advisories from the cross-Story review are `STORY-260905-3t31e9`.
I did not expand into them and none is blocking for this leaf.

## Routing

`accept_cr` revision 7. This leaf is clean.

---

# Routing addendum — `accept_cr` refused; element routed to `blocked`

**The technical verdict above stands unchanged: the leaf is clean and I accept
it.** The acceptance *transaction* cannot be recorded, for a reason outside this
task's scope. Recording it as a code finding would be false, so this section
states the blocker exactly.

```
task-board m 'accept_cr(TASK-260830-1snnef, revision=7, evidence=TASK-260830-1snnef_review6-verdict.md)'
→ change_request_invalid_record: Change Request CR-TASK-260830-1snnef-7 has no
  complete immutable producer role/archetype binding; legacy or unreadable
  ownership cannot authorize integration
```

## Constraint and evidence

`changerequest.Store.Accept` (`accept.go:78`) requires `ProducerRunID`,
`ProducerRole` and `ProducerArchetype` to all be non-empty. The record carries
only the first:

| Field | `rev-000007.json` |
| --- | --- |
| `producer_run_id` | `RUN-260905-f02929` |
| `producer_role` | `null` |
| `producer_archetype` | `null` |

This is a **mid-flight tooling upgrade**, not a defect in the published work:

- Revision 4 of this same CR was accepted at `06:12:48Z` by `RUN-260905-4e7ed8`
  with `producer_role` and `producer_archetype` equally `null`. The precondition
  did not exist then.
- Revisions 5, 6 and 7 were published at `07:17`, `08:27` and `08:40` by the
  binary of that time. The current binary is
  `0.24.3-246-g615a7f46`, **built `2026-09-05T13:10:45Z`** — after every one of
  them.
- The null binding is board-wide, not local to this element: every CR record I
  sampled across other elements has `producer_role = null` and
  `producer_archetype = null` too.
- The publisher itself now refuses without those fields
  (`spawnruntime/changerequest.go:119`), so no *new* record can reproduce the
  gap — only records written before the upgrade carry it.

The gate is correct and I am not arguing with it: it exists precisely so that
ownership nobody can read cannot authorize integration.

## Failed assumptions and what I did not do

- No reviewer-side repair path exists. `accept_cr` is the only CR mutation on
  the `m` path, and it has no override, no migration, and no way to supply the
  missing binding.
- Re-running from another shell or run id would not help — and the neighbouring
  `change_request_acceptance_unauthorized` guidance explicitly forbids it. This
  refusal is a different code and is about the *record*, not about me.
- **I did not hand-edit `.temp/changerequests/TASK-260830-1snnef/rev-000007.json`
  to insert a role and archetype.** That would mint, from a read-only reviewer
  run, the exact ownership binding the gate refuses to trust — the self-minted
  evidence shape this Story spent six rounds closing. Refusing to forge it is
  the point.

## Viable options

1. **Producer republish (recommended).** A new tracked producer run with a
   complete manifest role/archetype publishes revision 8 over the identical
   tree. The working tree is byte-identical to the reviewed candidate
   (`771f9c53fb708e5b401df371899c0d316bfd8e61`, re-derived through an isolated
   index after every battery), so revision 8 is the same bytes and every
   measurement in this verdict carries over verbatim — a reviewer run can accept
   it immediately without re-measuring. Cost: one producer run, no rework.
2. **Sanctioned backfill migration.** A tooling-owner migration writes the
   producer role/archetype onto records published before the upgrade, from the
   spawn manifests that already recorded them. This unblocks every pre-upgrade
   CR on the board at once, not just this one. It is a task-board decision, not
   a reviewer's, and it must come from the manifests rather than from a guess.
3. Do nothing and leave the Story's last leaf unlandable. Not viable.

## The exact input needed

Orchestrator/human decision: **republish this candidate as revision 8 through
the current binary's producer path (option 1), or authorize the tooling-side
backfill (option 2).** Either unblocks integration immediately; nothing in this
repository needs to change, and no finding in the code is outstanding.
