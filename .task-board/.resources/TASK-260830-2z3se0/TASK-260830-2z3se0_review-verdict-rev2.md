# TASK-260830-2z3se0 — reviewer verdict, CR-TASK-260830-2z3se0-2 revision 2

- Verdict: **changes requested** → `to-dev`
- repeat-of: `rev1/F1` and `rev1/F2` **by class, not by site** — both rev1 findings are themselves closed; two new findings reproduce their shape at different checks. Detail under *Routing*.
- Reviewed tree: candidate `4291b80f56f30f399ace7b79a7b189e139ff0582` (base `1cb6b93585749df4ef4755ce93e486bcde9e3a6b`), `repository_delta=present`, 29 paths.
- Working-tree OID recomputed against the candidate before the first mutant and after every batch; unchanged at the end (`TREE-OID-OK`, `git status --short` identical to the handoff).

## Baseline reproduced by this run

| Check | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./internal/sessadapter/` | exit 0 |
| `gofmt -l internal/sessadapter/` | empty |
| `go test ./... -count=1` | 17/17 packages ok |
| `go test ./internal/sessadapter/ -cover` | 81.4% of statements |

## G-D — traversal ratio

Gates were discovered by traversing production source, not by following tests:
every `range` loop over a member list, every package-level vocabulary composite
(61, enumerated by parsing production independently of the census), every
equality chain against a host fact, every bound guard. Each gate was attacked by
keeping it present and weakening it to admit exactly one member of the class it
must reject; deletions were used only where the reject class has one member.
Every mutant was grep-confirmed present and `go vet`-clean before its verdict.

**121 narrowing mutants over 93 production gate sites: 110 killed, 11 survived (91%).**
Round 1 was 63 of 87 over 45 sites (72%). The two are not the same measurement —
the traversal widened as well as improved, chiefly because the 61 vocabulary
tables are now enumerated exhaustively rather than sampled.

Reported separately, outside the ratio: 11 census shape probes (below), 3
controls (all behaved as required), 2 open-class mutants that survive by the
recorded F8 bound, 1 equivalent mutant excluded (`secondLength < 0` is subsumed
by the following range test), and 2 behavioural bypass probes.

**Round-1 survivors: 22 of 24 closed. Zero resurrections.** The two remaining are
the `unknownMember`/duplicate-member open-class pair, which the inventory header
now records as an explicit stated bound — correct, and not a finding. Every
round-1 kill I re-measured stayed dead (12 of 12; the 61-table battery re-covers
every table round 1 sampled).

| rev1 finding | Probe this round | Result |
| --- | --- | --- |
| F1 census blind to non-table vocabularies | named-type + inline-chain shapes, and all 61 tables widened | closed (S2, S3, 57/61 killed) |
| F2 `var x = ctor` alias hole | 3 alias spellings + struct-field + slice-literal + method-value | closed, 6 of 6 |
| F3 surrogate pairing bound | both edges, both directions, plus the first-surrogate top edge | closed, 5 of 5 |
| F4 set-valued gates on a sample | 16 obligation-loop mutants, grow and shrink | closed, 16 of 16 |
| F5 same-length context forgery | length-only and 32-byte-prefix narrowings | closed, 2 of 2 |
| F6 8 MiB edge | both directions on all three entry points | closed, 6 of 6 |
| F7 zero-value host facts | the four named sites | closed, 4 of 4 — but see N4 |
| F8 open-class member gates | 2 exemption mutants | survives by design; bound recorded in the inventory header |

## G-A — obligation sets are derived, and guarded in both directions

The question was whether the obligation set is *derived* or merely enumerated
more fully. Answer: **derived where it can be, and fail-closed where it is
enumerated.** 16 of 16 mutants killed, including the "fifth capability tomorrow"
case the brief named.

| Gate | Denominator from production | Obligations driven | Grow probe | Shrink probe |
| --- | ---: | ---: | --- | --- |
| `CheckDoctorHealthy` usable loop | `DoctorRequiredCapabilities` × 2 writer variants = 4 + 4 | 8 of 8 | A1 KILLED (count pin + `TestDoctorRequiredCapabilities`) | A2 KILLED |
| `CheckDoctorHealthy` name loop | same | 8 of 8 | — | A3 KILLED |
| `checkValidateNullability` | inline `targets`, 4 × 2 branches | 8 of 8 | A4 KILLED (archive positive case) | A5, A6 KILLED |
| `checkValidateResult` applicable | inline, 3 | 3 of 3 | — | A7 KILLED |
| `checkValidateResult` nullable-bool | inline, 3 | 3 of 3 | — | A8 KILLED |
| `CheckTargetWriteGates` adapter | inline, 3 + the 2-member pair | 5 of 5 | A10 KILLED (`TestTargetWriteComplementDerivesNonRequired`) | A9 KILLED |
| `CheckTargetWriteGates` provider | inline, 2 | 2 of 2 (× 2 failure modes each) | — | A11 KILLED |
| `decodeCapabilities` | `capabilityOrder`, 15 | 15 of 15 | — | A12 KILLED |
| `missingMember` | per-body `*Required` | — | — | A13 KILLED |
| `requestMemberSet` / `successMemberSet` | `requestBodyMembers` / `successBodyMembers` | — | — | A14, A15 KILLED |
| `validOperation` | `operationOrder` | — | — | A16 KILLED |

**Obligations driven over obligations that exist: 52 of 52.** Where the list is
written inline rather than read from a helper (`targets`, the two validate
triples, the two target-write lists), the addition direction is still closed —
by the archive positive case and by the derived non-required complement — so a
fifth member added tomorrow reddens rather than shipping unguarded. That is the
substantive answer to G-A, and it is a real improvement over "four of four
hand-written".

## G-B — the census after F1: 5 of 11 vocabulary shapes seen

A four-member closed vocabulary was planted and consulted live from
`CapabilityUsable` (behaviour-neutral: the planted set equals `capabilityStatuses`,
which the value is already validated against upstream), once per shape. The
denominator was built by enumerating how a closed set can be written in Go,
independently of the census.

| Shape | Census |
| --- | --- |
| package-level slice composite (control) | **seen** |
| named slice type composite | **seen** — F1's C2 closed |
| inline `\|\|` comparison chain | **seen** — F1's C3 closed |
| package-level `map[string]struct{}` | **seen** |
| tagged switch on a non-`operation` identifier | **seen** |
| **tagless switch over the same members** | blind |
| **function-local slice literal** | blind |
| **`make()` + `init()` population** | blind (named in the stated bound) |
| **`const` string + `strings.Contains`** | blind |
| **`regexp.MustCompile` alternation** | blind |
| **multi-name package-level `var a, b = []string{…}, []string{…}`** | blind |

**Vocabularies found over vocabularies that exist: 5 of 11 shapes.** The extension
is real and closes both shapes F1 named. Two of the six blind spots contradict
the census's own text rather than its stated bound, which is finding N5.

## Blocking findings

### B1 (blocking) — an archive-mode validate success is unconditionally refused at the production entry point

`checkValidateResult` (operations.go:1093) calls `checkValidateNullability` on the
success body. The pinned §7.8 validate row gives the success body
`{context, mode, valid, structural_valid, semantic_marker_valid, identity_valid,
workspace_binding_valid, resume_surface_valid, findings, evidence_digest,
extensions}` — no target members at all — while `checkValidateNullability` requires
`projection_plan_id`, `projected_object_manifest_id`,
`read_back_evidence_manifest_id`, `expected_target_native_session_id` to be `null`
under `mode == "archive"`. `isNull(nil)` is `false` (decode.go:317: `"" != "null"`),
so absent is not null and every archive success refuses.

Driven at the production entry with the package's own fixtures — an archive
request the package itself admits, then its success:

```
CheckSuccessBody(OpValidate, success, SuccessFacts{ValidateMode: "archive"})
  → session_adapter_protocol_error: validate archive mode carries a target member
    (member = projection_plan_id)
```

The refusal names a member the success body is structurally forbidden to carry.
One of fourteen operations has one of its three modes unconditionally refused,
and the AC row is "Exact contract fixtures and negative/refusal cases pass".

The producer found this during F4, preserved behaviour, recorded it in
`LOGBOOK.md`, and referred it for a product decision with three options. Finding
it and not papering over it is the right call, and the honesty is noted. But it
is **not** a product decision: the pinned document's success-body member list —
the same list the member-set census already derives from that row — resolves it.
Option (a), scope the rule to request bodies, is the only reading consistent with
the pinned member table; (b) would break staged/live successes and (c) contradicts
the row, which states `mode:staged|live|archive` on the success side too. This is
rework inside the package, with no human input required.

Whatever the fix, the success side needs a driving test: today no committed test
constructs an archive-mode validate success, which is why 904 tests stay green
over an operation mode that cannot succeed.

### B2 (blocking) — four unknown-member gate maps carry no content pin, and one is a confirmed bypass

`TestClosedMemberSetsAreDerivedFromSpec` compares both the ordered `*Required`
list and the `*Members` map for its 17 `ordered` rows. The four bodies handled by
the special-cased tail compare **only** `*Required`:

| Map | Site | Widened by one name |
| --- | --- | --- |
| `successMembers` | protocol.go:370 | **SURVIVED** |
| `failureMembers` | protocol.go:460 | **SURVIVED** |
| `manifestMembers` | manifest.go:119 | **SURVIVED** |
| `doctorResultMembers` | probe.go:584 | **SURVIVED** |

`manifestMembers` additionally *looks* pinned: census_test.go:892 declares
`manifestMembers := deriveManifestTableMembers(t, text)`, a local that **shadows the
package-level map**, and then compares it against `manifestRequired`. The
production map is never read in that function.

No cardinality guard backstops any of the four — `len(members) != …` appears only
in `decodeCapabilities` and one tuple site. Confirmed behaviourally, not inferred:
with `successMembers` widened by one name, a frame carrying `ax.planted.member` is
**ADMITTED** by `CheckSuccessEnvelope`; the unwidened control refuses it. The other
three share the identical construction (`unknownMember(members, X)` with no
cardinality guard); I confirmed one of the four behaviourally and report the other
three as unpinned by identical shape rather than claiming four bypass probes.

**16 of 20 unknown-member gate maps pinned.** The census reports 63 registrations
and green, because a registration row proves the table is *known*, not that its
contents are *pinned*. This is F1's class — a closed set the census reports as
registered while nothing pins its content — at a different check and on the
outermost gate in the package.

Failure scenario: a later leaf adds a member to `successMembers` (or the other
three) rather than to `successRequired`. Every test stays green, the census
reports 63/63, the member-set census reports 20 closed bodies derived, and the
success envelope admits a member the contract never defined.

Two adjacent reporting defects at the same site: `t.Logf("member-set census: %d
closed bodies derived", len(ordered)+3)` reports 20 while 21 derivations run
(17 ordered + success + failure + manifest + doctor); and one `ordered` row
(`SupportedContractVersions`) carries `members: nil` by design. Neither number
distinguishes 16-of-20 from 20-of-20 — the count needs to be the count of
*member-map comparisons*, not of bodies.

### B3 (blocking) — the inventory's constructor-closure sentence is false

inventory_test.go:81 states: *"refusalConstructors is the closed constructor set
the derivation observes. A production refusal built any other way is invisible to
the inventory **and fails the alias gate**."* The second clause does not hold.

| Probe | Result |
| --- | --- |
| direct-call arm through `failInvalid`, **replacing** an existing arm (control) | KILLED |
| additive arm through `failInvalid` (control) | KILLED |
| package-level `var alias = failInvalid` | KILLED |
| function-local `var alias = failInvalid` | KILLED |
| `alias := failInvalid` | KILLED |
| constructor in a struct field | KILLED |
| constructor in a slice literal | KILLED |
| method value calling `failInvalid` with non-literal args | KILLED |
| **8th constructor `failQuota` + additive arm, direct call** | **SURVIVED** |
| **inline `axerror.New(...)` refusal + additive arm, no constructor** | **SURVIVED** |

The alias closure itself is now genuinely complete — all three spellings F2 named
plus three more, and the header sentence describing it matches the walk exactly
("exempts only the definition's own Name in its ValueSpec"). That half of G-C is
closed and should not be re-litigated.

What is not closed is the layer above it. `refusalConstructors` is a hand-written
list of seven with no production-derived denominator. Production currently imports
`axerror` only in protocol.go with exactly seven `axerror.New` call sites — so the
list happens to be complete today, and nothing enforces that. Both survivors ship
an unwitnessed refusal arm through the full suite. The earlier probes were killed
only because they *replaced* an existing arm and tripped the 323 floor; the
additive form is the shape a producer would actually write.

This is F2's class — a documented completeness guarantee ahead of the walk — in
the same file, one sentence below the one that was fixed, and it undermines the
323/323 figure that is this package's headline evidence.

## Non-blocking findings

### N4 — F7's class is unclosed at `CheckCallBinding`

The four sites rev1 named are closed (F7a–F7d killed). The same shape at
`CheckCallBinding` (discovery.go:196–229) is open: exempting the zero value of the
compared binding fact survives on all five gates.

| Gate | Zero-fact exemption |
| --- | --- |
| `role != binding.Role` | **SURVIVED** |
| `context.ProviderID != binding.ProviderID` | **SURVIVED** |
| `context.ManifestDigest != binding.AdapterManifestDigest` | **SURVIVED** |
| `context.ExecutableSHA256 != binding.ExecutableSHA256` | **SURVIVED** |
| `context.Environment != admitted` | **SURVIVED** |

Baseline fails closed by accident of comparison against a zero struct, exactly as
rev1 wrote of the other four. Either add the cases or record the bound explicitly;
fixing four named instances of a class and leaving the fifth site unnamed is how
the class comes back.

### N5 — the census's stated scope is still ahead of its walk in two places

The stated bound is honest about runtime-built sets and correctly names `init()`.
Two other sentences are not bounds but claims, and both are false:

- *"scans every package-level var shaped like a vocabulary"* — `vocabularyTablesInFile`
  skips any `ValueSpec` with `len(value.Values) != 1`, so
  `var a, b = []string{…}, []string{…}` is a package-level composite the scan never
  sees.
- *"Any other switch is a membership gate hiding spot and fails"* — the switch census
  classifies **every** tagless switch as OK (`if statement.Tag == nil { classified++ }`).
  The comment justifies this by naming the UTF-16 hex classifier; the code admits
  the whole shape. Production has exactly one tagless switch today, so nothing sits
  in it yet — but a tagless `switch { case v == "a": … }` is invisible to the table
  scan, the chain scan and the switch scan at once.

Two further blind spots are not named by the bound at all: a function-local
composite (a shape production already writes at five sites — `targets`, the two
validate triples, the two target-write lists) and a compile-time union encoded in a
`const` string or a `regexp` alternation. Neither is "built at runtime".

None of these is a live bypass today, which is why this is not blocking: every
function-local set production currently holds is behaviourally pinned in both
directions by the G-A battery. It is the claim, not the coverage, that is wrong.

## What is correct and should not be re-litigated

- **F1 through F7 are genuinely closed**, each verified by a mutant of my own
  construction rather than by reading the rework note. 22 of 24 rev1 survivors dead,
  zero resurrections, 72% → 91% on a wider traversal.
- **F4's obligation sets are derived and two-sided** — the doctor set is read from
  `DoctorRequiredCapabilities` with a count pin, and the target-write set has a
  derived non-required complement. This is the right discipline and it answers G-A.
- **The alias closure is complete and its sentence matches the walk.** Six spellings
  attacked, six killed.
- **F3's pair sweep is generated, not hand-written.** Both edges, both directions,
  plus the first-surrogate top edge, all killed.
- **F6 pins the bound on both sides at all three entry points.**
- **F8 is correctly recorded as a stated bound** rather than faked with a
  hand-picked witness.
- **The durable-state bound is true, not asserted**: the package imports no `os`,
  `net`, `exec`, or `time.Now` — the "no crash/idempotency evidence applicable" AC
  row is verified, not taken on trust.
- **The anomaly was surfaced, not buried.** B1 exists because the producer
  reproduced it, wrote it down in the rework note and `LOGBOOK.md`, and refused to
  redesign behaviour under a review finding. That is the correct instinct; only the
  routing is wrong.

## Evidence-quality note

The rev2 battery (26 killed + 1 control-green / 27) is honest about what it
measured and its method bounds are accurate. It is **finding-scoped**, not
traversal-scoped: 27 mutants aimed at F1–F7. That is why 11 survivors sit outside
it — 4 unknown-member maps, 5 `CheckCallBinding` facts, 2 constructor-closure
holes, none of which any rev1 finding pointed at. The DoD row asks for a
killed/measured ratio **over a traversal of production gate sites**; rev2 reports
"Survivors: none", which is true of its 27 and not of the package.

## Routing

`repeat-of: rev1/F1, rev1/F2 — by class, at different sites; both rev1 findings closed.`

B2 reproduces F1's class (a closed set the census reports as registered while
nothing pins its content) at `TestClosedMemberSetsAreDerivedFromSpec` rather than
at the table scan. B3 reproduces F2's class (a documented completeness guarantee
ahead of the walk) one sentence below the one that was fixed. B1 is new.

This is the second consecutive round in which an evidence instrument's own
completeness claim was found ahead of what the instrument does, at three distinct
sites. Per the routing rule, the next step on that class should be a gate rather
than a third site-by-site pass: a check that every claim of the form "every X is
seen / any other X fails" is either enforced or demoted to a stated bound —
for instance, deriving `refusalConstructors` from production (`axerror.New` call
sites outside the registered set fail), comparing every `*Members` map against its
derived list with a count of comparisons rather than of bodies, and making the
switch census name the classifier it exempts instead of exempting the shape.

## Requested changes

1. **B1** — scope `checkValidateNullability` to request bodies (or state why the
   success-side call is correct against the pinned §7.8 success member list), and add
   a committed test that drives an archive-mode validate success through
   `CheckSuccessBody`.
2. **B2** — compare `successMembers`, `failureMembers`, `manifestMembers` and
   `doctorResultMembers` against their derived member lists; remove the local
   `manifestMembers` shadow at census_test.go:892; report the member-set census as a
   count of member-map comparisons, not of bodies.
3. **B3** — derive the refusal-constructor denominator from production instead of
   listing it, so an 8th constructor or a bare `axerror.New` refusal fails the
   inventory; then make the header sentence say what the walk does.
4. **N4** — close or explicitly bound the five `CheckCallBinding` zero-fact gates.
5. **N5** — fix the two census sentences that are claims rather than bounds
   (multi-name `ValueSpec`, tagless switch), and name the function-local and
   encoded-union blind spots in the stated bound.
6. **Evidence** — report the next battery as a ratio over a traversal of production
   gate sites, not over the finding list.

No stop-the-line boundary was found. B1 looked like one and is not: the pinned
document resolves it without a human decision, and every other finding is ordinary
rework inside this package.
