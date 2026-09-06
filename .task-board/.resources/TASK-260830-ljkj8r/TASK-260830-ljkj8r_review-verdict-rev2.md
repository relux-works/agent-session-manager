# TASK-260830-ljkj8r — reviewer verdict, CR rev2

**Verdict: CHANGES REQUESTED → `to-dev`.**
**repeat-of: rev1-B3 (C1 below) and rev1-B2 (C2 below).** Two consecutive
same-class findings: the next step is a derived gate, not another hand audit.
Evidence: `TASK-260830-ljkj8r_review-mutant-log-rev2.md`.

Candidate tree `bd3cb8c05f421365e1e20f9e18ad53cb13d33d18` verified
byte-identical before and after all 433 mutant applications.
`repository_delta=present`, 27 paths; production code is byte-identical to
rev1 (`git diff 4d8a599 bd3cb8c` touches only test files and `LOGBOOK.md`).
Delta 23 → 27 = `arm_identity_test.go`, `bound_census_test.go`,
`bounds_edge_test.go`, `query_operations_test.go`; four already-present test
files carry the vector surgery, `LOGBOOK.md` the round entry. Fully accounted.

Suite re-run on my own machine: `go test ./... -count=1` 18 packages ok,
`go vet ./...` exit 0, `gofmt -l internal/` clean,
`go test ./internal/dirnode -cover` 83.3%. All three producer claims reproduce.

---

## G-A — the instrument transferred. B1 is CLOSED.

**The shape space is this package's own, not leaf 1's list.** It enumerates ten
shapes against `dirnode` constructs that do not exist in `sessadapter` —
`checkURI`, `CheckCursorReuse`'s character measure, `Journal.Import`'s ordering
gate, `skip > 1000000`, `index > 63`, `version != 1` — and marks six derived,
four stated, with a live-instance claim per shape. Three synthetic suites prove
the derivation halves (indirection, non-`if` contexts, length variables), 11
canaries fail the census closed if the scanner goes blind, and an empty scan
fails. 101 sites rostered, 72 driven, 29 exempt with rationales.

**Ratio over the identical production-derived denominator: 68 of 94 killed
(was 19 of 94).** 0 NOT_APPLIED, 0 COMPILE_FAIL, **0 resurrections** — all 19
rev1 kills are still killed. The `checkStringBounds max +1` mutant that
survived on every maximum in rev1 now dies on every one.

All 26 survivors map 1:1 onto declared `equiv-ok` rows, and I verified each
masking gate in production source rather than accepting the rationale:
`environmentIDPattern` `{0,63}`, `providerIDPattern` `{0,31}`, `checkSemver`'s
5-character floor, the 8/5/4/7/3-member closed tables, the 27-field registry,
and the joint count/index gate at 65 (both tightenings die). No false
equivalence.

**The wrapper shape is closed here.** `checkURI` is this package's
`requireStringBounds` — a forwarder carrying its own `1..512` literals,
exempted as "mechanism". Its ceiling dies in both directions and its floor dies
at `<4`. Leaf 1's 25-of-46 survivor shape does not recur.

I extended the battery to the 54 sites rev1 never touched (len guards, wrapper
internals, shape-7/8/10 stated bounds): 42 killed. The findings below come from
there and from two further production-derived sweeps.

---

## C1 (blocking) — the arm-slide sweep was reported as covering a class it did not cover. **repeat-of rev1-B3.**

The B3 vector itself is fixed and pinned by construction, and the three
rename→delete missing vectors are pinned by message. Verified: all four arms die
under deletion. But the audit note in `arm_identity_test.go` states the rest of
the class clean —

> "every 'unknown member' vector adds exactly one member, so those reach the
> arm they name"

— and that is false for three of them. Driven through the production entry,
unmutated:

| vector | arm it names | arm it actually reaches |
|---|---|---|
| `manifest_test.go:62` "unknown member" | `unknownMember` (`manifest.go:268`) | `node manifest trailing data after the object` — the replacement `"limits":{` → `"limits":{},"extra":1,` hoists every limits member to the top level and leaves malformed JSON |
| `query_test.go:75` "unknown member" | `unknownMember` (`query.go:415`) | `directory query operation is not a closed QueryOperation` — the `"extensions":{}}` needle matches `operations[0].parameters` first, never the envelope |
| `probe_test.go:142` "unknown member" | `unknownMember` (`probe.go:338`) | `probe response node build is not the closed DirectoryNodeBuild` — the needle matches `node_build` first |

Independently: a 144-arm reachability sweep (every production `if` whose body
builds a `failX`, derived by AST; 115 killed / 144, 0 unmeasured) finds those
three envelope-level `unknownMember` arms deletable with a green suite, plus:

- **5 `missingMember` arms deletable** — `probe.go:141`, `protocol.go:428`,
  `protocol.go:559`, `scan.go:78`, `scan.go:193`. These vectors *do* reach the
  right arm today (I confirmed all four report `misses a required member`), but
  their assertions check code + detail member only, and a deleted member also
  refuses downstream. The three vectors round 2 fixed die only because round 2
  added a message assertion; the same assertion was not extended to the rest of
  the class.
- **2 gates with zero negative coverage** — `checkExtensions` on the probe
  response (`probe.go:392`) and on the query envelope (`query.go:467`). The
  gates work (a `"NOTDNS"` key gives `probe response extensions are not
  reverse-DNS keyed`), but deleting either keeps the suite green. Their four
  siblings — manifest, probe request, scan request, scan response — are all
  covered, so this is a class closed at four of six.
- **3 empty-frame arms deletable** — `bootstrap.go:144`, `protocol.go:399`,
  `protocol.go:530`. All three are census rows marked *driven*;
  `TestFrameBoundEdges` asserts only `err != nil`, and an empty frame also fails
  the strict decoder, so the named driver cannot fail.

What to change: derive the vector-reachability gate instead of auditing it by
hand. Every fixture-surgery vector that names an arm should assert that arm's
message, the way `TestMissingVectorsDriveTheMissingArm` does for three — and
the arm roster should fail closed when an arm has no vector that fires it.

---

## C2 (blocking) — B2 closed the admissions, not the obligations. **repeat-of rev1-B2.**

The 17-operation *admission* set is genuinely derived: the union is built from
the production `readOperations` + `mutationOperations` tables and fails closed
when an operation has no builder. 17 of 17 admit. The three named narrowing
probes — `"root"`, `"pwned"`, `"secret"` — are rows and die. That half is real.

The *refusal* set is a 45-row hand table with no production-derived
denominator. Measured against one — every `return false` in `query.go`, one
mutant per site, `return false` → `return true`:

**58 of 89 obligations driven; 31 removable with a green suite.**

| validator | obligations | driven | undriven |
|---|---:|---:|---:|
| `checkPlanContinueParameters` | 9 | 3 | 6 |
| `checkQuerySort` | 8 | 3 | 5 |
| `checkFilters` | 16 | 13 | 3 |
| `checkQueryParameters` | 5 | 2 | 3 |
| `checkExecutePlanParameters` | 5 | 2 | 3 |
| `checkSubject` | 4 | 1 | 3 |
| `checkUUIDv7DigestSubset` | 5 | 3 | 2 |
| `checkQueryFlags` | 3 | 1 | 2 |
| `checkTagsParameter`, `checkLineageParameters`, `checkDistinctParameters`, `isAnnotationMutation` | 8 | 4 | 4 |
| 11 validators at full coverage | 26 | 26 | 0 |
| **total** | **89** | **58** | **31** |

`plan_continue` is the sharpest instance: it admits positively and its three
closed vocabularies refuse, but all six member-shape checks
(`source_instance_id` digest, `to_host_id` UUIDv7, `to_installation_id` digest,
and the three `rawString` type reads) can be deleted with a green suite. That
is positive-path-only evidence for one of the seventeen operations.

And the answer to the question as asked — **yes, one arm still carries N of
them**: all 89 obligations still report through the single
`checkQueryOperation` arm. Coverage moved (10 functions off 0.0%, package
83.3%); the obligation ratio is 65%.

---

## C3 (blocking) — the uniqueness half of the sorted-unique class is closed at one of eight sites.

`>=` → `>` in a pairwise ordering scan keeps the gate present and weakens it to
admit exactly one member of the class it must reject: a duplicate. Production
refuses duplicates today. Seven of eight survive:

| site | member class | `>=`→`>` |
|---|---|---|
| `decode.go:377` `checkSortedUniqueStrings` | every sorted-unique string array | **KILLED** |
| `decode.go:403` `checkSortedUniqueDigests` | every digest array | SURVIVED |
| `decode.go:429` `checkSortedUniqueUUIDv7` | every UUIDv7 array | SURVIVED |
| `manifest.go:525` `checkContractAssertions` | contract encodings | SURVIVED |
| `query.go:825` `checkTagsParameter` | `set_tags` tags | SURVIVED |
| `query.go:927` `checkExecutePlanParameters` | confirmations | SURVIVED |
| `query.go:1042` `checkUUIDv7DigestSubset` | `lineage_anchors` | SURVIVED |
| `scan.go:438` `Journal.Import` | journal keys | SURVIVED |

The census states this class as covered — shape 10, *"ordering gates, not
domain bounds, pinned by the sortedness refusal tests"*. The sortedness refusal
tests feed unsorted vectors (`["b","a"]`); none feeds a duplicate. This is
leaf 1's "class closed at one helper leaves the wrappers open" recurring
exactly, one abstraction level up.

---

## C4 (blocking) — `EncodeRequest`'s deadline ceiling has no driver, and its driver's comment says it does.

`protocol.go:352` is a gate on the production host-side frame builder.
Widening it survives under two spellings (`> MaxDeadlineMS+1`, `> 4000000`);
the floor control (`< MinDeadlineMS-1`) dies, so the harness is live.

The census names this a shape-7 stated bound "pinned behaviourally by its named
driver". The driver, `TestUint53BoundEdges/request_deadline_ms`, is:

```go
if _, err := DecodeRequestFrame(...); err != nil {
        return err          // <- returns here for 0 and for 3600001
}
_, err := EncodeRequest(...)
return err
```

Both out-of-range literals are refused by the decode arm and return before
`EncodeRequest` is ever called, so the builder is only ever driven with the two
admitting values. The row's own comment claims the opposite: *"a refusal from
either entry fails the row."* A reported fact that measurement contradicts is a
finding on its own.

---

## Verified good — do not re-litigate

- Bound census: package-specific shape space, fails closed on unregistered
  sites, orphan rows, helper aliases, empty scan; 11 canaries; three synthetic
  derivation proofs. 68/94 on the rev1 denominator, 0 resurrections, every
  survivor a verified equivalence.
- `checkURI` wrapper closed in both directions — leaf 1's `requireStringBounds`
  shape does not recur.
- B3's own vector fixed and pinned by construction; the three rename→delete
  conversions fixed and pinned by message.
- 17/17 operation admissions derived from the production registry, failing
  closed on a missing builder; `root`/`pwned`/`secret` all die.
- 115 of 144 refusal arms die under deletion; the refusal-arm census itself,
  the traceability re-pin, `Journal.Import`'s build-then-assign, and the README
  claims were verified in rev1 and are unchanged.
- Suite, vet, gofmt, and the 83.3% coverage figure all reproduce.

## Scope note

C1–C4 are evidence findings, not behaviour bugs: every gate named above refuses
correctly in the shipped code. What is missing is the test that would fail if it
stopped. Given two consecutive same-class rounds on C1 and C2, the next revision
should add a derived gate for each class — arm-to-vector reachability, and an
obligation roster for the operation validators — rather than more rows.
