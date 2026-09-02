# BUG-260902-3dn8jd — Review verdict, round 2 (CR rev 2)

**Verdict: CHANGES REQUESTED → `to-dev`.**

Reviewed commit `c3cd51c`, tree `158bc0bc`, base `422786c` (`HEAD^`). Reachability
checked first as the round-2 brief required: `git rev-parse HEAD^^{}` =
`422786cc…`, `git rev-parse HEAD^{tree}` = `158bc0bc…` (equals the CR candidate
tree), and `git merge-base --is-ancestor 5b8423a HEAD` exits 1. The reviewed OID
is the branch head; this review is not void.

Worktree was never modified. Every mutant ran in `/tmp/axrev2`, a `git archive
HEAD` copy, and each mutant was grepped for in the file before the measurement
was believed. `git status --short` was empty at start and at end.

---

## What the change gets right, verified by attack

| Property | Evidence I produced myself |
| --- | --- |
| Digest gate is the anchor | `shasum -a 256 internal/specdoc/SPEC.md` = `562546d2…` = `specpin.DocumentSHA256`. `internal/specpin/pin.go` is **not** in the diff, so the document could not be minted to fit a digest this change chose. |
| Clause pin fails closed | Deleted `"GitFeatures": {"10.4"}` from `constraintRowSpecSections` (mutant confirmed absent by `grep -c` = 0). Both `TestEveryConstraintEnumerationShapePinsItsSpecificationClause` and `TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification` FAIL, naming the shape and all ten of its rows. Not vacuous. |
| Existing derivation preserved | `TestConstraintEnumerationMatchesRequireExactMembers` is untouched; the only inventory-test edit is `splitTableCells` + a larger scanner buffer, both forced by escaped pipes in the new cells. |
| specdoc never ships | `go list -deps` over all 11 packages: no non-test package reaches `internal/specdoc`. |
| Suite state | `gofmt -l .` empty, `go vet ./...` 0, `go test ./... -count=1` 0 across 11 packages. |

### The judgement call the brief asked me to decide independently: the producer is right

Round 1 argued the seven `extensions` rows should have stayed dropped on fidelity
grounds. I tested the enforcement claim instead of arguing it. In the `/tmp` copy
I narrowed `validateSessionBoardGoal` (`closed_shapes.go:459`) to call
`requireObject` without `validateExtensionsObject` — a narrowing, not a deletion;
the member stays required and typed, only the reverse-DNS key rule goes. Mutant
confirmed present by reading back lines 459–481.

`TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries/Session_Record_Board_Goal_extensions`
FAILS on all five invalid key shapes: `CalculateObjectIdentity(...) error = <nil>,
want identity refusal`. So the row the round-1 reviewer would have dropped is
what kills a real production narrowing on a real entry point.

The producer's separation is correct and should be recorded as answered: **the
artifact row records what the document says; the gate row records what production
enforces.** Conflating those two axes is what caused F1 in the first place. Keep
the eleven rows and the per-family count pin.

---

## BLOCKING — F4. The rewrite introduced seven misattributed rows, three of them materially false

The residual limit the producer documented — "the anchor is a clause, not a
shape" — is not merely theoretical. **The shipped artifact instantiates it seven
times**, all inside the Git embedded-types table at `SPEC.md:4787-4793`, where
one clause (10.4) declares six sibling types one per line.

I scanned all 347 rows for the class rather than sampling: for every cited line
that is a type/member table row, compare the identifier in its first cell to the
row's shape and member. Exactly these seven disagree; no other shape family in
the artifact has an instance.

| Artifact row | Cites | That line actually declares | The member's real declaration | Damage |
| --- | --- | --- | --- | --- |
| `GitIndex.format` | L4789 “`format:git_pack_v2`” | **GitObjectPack** | L4790 `format:git_index` | **material** — the column now asserts the pinned SPEC types this member `git_pack_v2`. It does not. `validateGitIndex` (`closed_shapes.go:1454`) enforces `git_index`. The artifact makes production look like it contradicts the spec. |
| `GitIndexEntry.mode` | L4788 “`mode:branch\|detached\|unborn`” | **GitHead** | L4791 `mode:uint32` | **material** — an enum attributed to a `uint32` member. `closed_shapes.go:1514` enforces `requireUint(object, "mode", 1<<32-1)`. |
| `GitIndexEntry.oid` | L4788 “`oid:git-oid\|null`” | **GitHead** | L4791 `oid:git-oid` | **material** — invents nullability. `closed_shapes.go:1518` uses `requireGitOID`, which does not admit null. |
| `GitIndex.blob_id` | L4789 | GitObjectPack | L4790 | wrong line/shape; quoted text coincides |
| `GitIndex.blob_descriptor_id` | L4789 | GitObjectPack | L4790 | wrong line/shape; text coincides |
| `GitFeatures.object_format` | L4789 | GitObjectPack | L4793 | wrong line/shape; text coincides |
| `GitSubmodule.path` | L4791 | GitIndexEntry | L4792 | wrong line/shape; text coincides |

### Why this blocks rather than being an accepted known limit

1. **It is a regression this change introduced.** The pre-change cells were
   correct. `git show 422786cc:…/constraint-enumeration.md`:
   - `GitIndex.format` → “exact git_index” ✅
   - `GitIndexEntry.mode` → “uint32” ✅
   - `GitIndexEntry.oid` → “git-oid; matches object format” ✅

   The rewrite replaced three accurate prose declarations with verbatim quotes
   of a **different schema's** row. That is the identical failure this Bug
   exists to remove — "the quoted word `digest` reached the Pinned SPEC
   declaration column although no such text exists" — reproduced by the fix,
   one schema over. Under the task's own disclosure duty these are findings, and
   they were neither disclosed nor caught.

2. **The whole suite stays green with them shipped.** `go test ./... -count=1`
   exits 0 at `c3cd51c`. Positive-path-only: the gate proves the quotes are
   verbatim, correctly located and contain the member substring — and all seven
   satisfy every one of those, because `strings.Contains("mode:branch|detached|unborn", "mode")`
   is true and L4788 resolves to clause 10.4.

3. **The fix is cheap and does not need new gate design.** Every correct
   citation already exists and is uniquely locatable at the right line — I
   verified each with `specdoc.QuoteLines`:
   `format:git_index`→[4790], `mode:uint32`→[4791], `oid:git-oid`→[4791],
   `blob_id:digest`⊇4790, `path:path`⊇4792, `object_format:sha1|sha256`⊇[4789,**4793**].

### Required

1. Correct all seven rows to cite their own type's line, with that type's text.
2. Re-run the same class scan over all 347 rows and report the ratio, not prose —
   the scan is ~30 lines and catches the whole class, not the seven I found.
3. Consider whether the clause anchor can be tightened for the multi-type tables
   specifically: in `10.4` the six Git types are one table row each, so a
   *declaring-line* pin per shape is a table lookup, not new judgement. If you
   judge it disproportionate, say so concretely — but the rows must be correct
   either way.

---

## F5 — recorded, not blocking. The blank-line fix has a within-block hole, and the docs overclaim

`specdoc.Normalize`'s block separator stops a quote crossing a *blank* line. It
does not stop a quote crossing the boundary between two **adjacent table rows**,
which is not a blank line.

Planted into a `/tmp` copy of the shipped artifact (mutant confirmed present,
`grep -c` = 1) — `Lease Record.holder_host_id`:

```
L1906 “\| <code>holder_host_id</code> \| UUIDv7 \| Proposed owner \| \| <code>predecessor_lease_id</code> \| UUIDv4 or null \| Null only at epoch 1 \|”
```

`go test ./internal/canonicaljson/ -count=1` → **ok, 18.251s**. Admitted. It
begins at the declared line, names the member, and sits in its own clause, so
every rule passes while the row imports the *next* member's constraint.
`QuoteLines` also admits `“| <code>created_at</code> | timestamp | Diagnostic only | | <code>extensions</code> | object | Reverse-DNS extension keys only |”` at [1912 2091 4703 6064].

The claim to fix is the wording, in both places:

- `README.md`: "forgives the document's hard line wrapping and table indentation
  **and nothing else**"
- `constraint-enumeration.md` rule 1: same "and nothing else"

It also forgives the row boundary inside a table. `specdoc.go`'s own comment
("hard line wrapping and table indentation *inside one block* are still
forgiven") is the honest phrasing; the two docs are not. Either make a
row-boundary non-collapsible too, or state the case — the same choice round 1
offered for blank lines. I do not require the code change; I require the
documented rule to be true.

Note this is the mechanism behind F4 generalised: within one block, both the
clause anchor and the member anchor are satisfiable by a sibling's text.

---

## Answers to the remaining round-2 questions

- **Within-clause retarget still admitted?** Confirmed, and worse than stated —
  seven shipped rows already are one (F4).
- **`environment_id` gap honest?** Yes. Both cited lines (3609, 3626-3630) are in
  7.8, so the clause anchor cannot catch it. The stated limit is accurate, not
  understated. Leaving production unchanged and quoting both lines is the right
  call; it needs its own Bug, not a silent relaxation here.
- **`TestSessionRecordGrammarClassificationIgnoresPinnedSpecProse` the real F1
  mutant?** It is the right mutant but is now near-tautological: classification
  reads only shape+member, so scrubbing the column cannot move it. That is
  correct as a regression guard against re-introducing prose keying. Worth
  keeping, worth not counting as evidence of anything else.
- **`provider-id` shape-awareness.** Real defect, correct fix: the
  `Session Record 2.0.0 and 3.0.0` provider_id row was being driven through
  `validSessionRecordV1Object()`; it now selects `validSessionRecordV3Object`
  for the non-1.0.0 shape.
- **Clause pin vacuously satisfiable?** No — proven by the GitFeatures deletion
  mutant above, which reddens in both directions.
- **`closed-vocabularies.md` out of scope?** Agreed. Different artifact,
  different column contract. Own board item; `specdoc` now makes it cheap.

---

## Route

`to-dev`. F4 is a small, mechanical correction of seven cells plus a re-scan;
F5 is a wording fix in two files. Nothing in the specdoc design, the clause
anchor, the F1 re-keying, or the anti-vacuity suite is in question — all of it
survived attack and should not be regressed while fixing F4.
