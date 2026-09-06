# TASK-260830-2z3se0 — reviewer verdict, CR-TASK-260830-2z3se0-3 revision 3

- Verdict: **changes requested** → `to-dev`
- repeat-of: `rev1/F7, rev2/N4` (B4) and `rev1/F2, rev2/B3` (B6). Both are the
  **third consecutive round** on their class. Per the routing rule the next step
  on each is a gate, not a fourth site-by-site pass; the requested changes are
  written that way.
- Reviewed tree: candidate `9b1c084a1b0a970bc2eed187d5155f4191528c51` (base
  `1cb6b93585749df4ef4755ce93e486bcde9e3a6b`), `repository_delta=present`, 29 paths.
- Working-tree OID recomputed against the candidate before the first mutant and
  after every batch, including the three scratch-test probes; unchanged at the
  end (`9b1c084a…`), `git status --short` identical to the handoff.

## Baseline reproduced by this run

| Check | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./internal/sessadapter/`, `GOOS=windows go vet` | exit 0 / exit 0 |
| `gofmt -l internal/` | empty |
| `go test ./... -count=1` | 17/17 packages ok |
| `go test ./internal/sessadapter/ -cover` | 81.5% of statements |
| `go run ./internal/traceability/cmd/tracecheck` | ok, acceptance_cases=88, exit 0 |

Every gate the rework note claims is reproduced. The producer's numbers are honest.

## G-A — B3 is answered: the census does the killing, not the floor

This was the whole point of the round, and the answer is unambiguous. Both
additive shapes were planted with the arm count held **at** 323, so the
inventory floor (`len(arms) < 323`) cannot fire.

| Probe | Arms derived | Full-suite failures | Verdict |
| --- | ---: | --- | --- |
| M-A1 — 8th package-level ctor `failQuota` + additive arm, direct call | 323 (unchanged) | **exactly one**: `TestRefusalConstructorsMatchProduction` | KILLED by the census |
| M-A2 — inline `axerror.New(...)` refusal + additive arm, no constructor | 323 (unchanged) | **exactly one**: `TestRefusalConstructorsMatchProduction` | KILLED by the census |

M-A1's message is `axerror.New inside unregistered "failQuota" at protocol.go:160:9`;
`TestDerivedRefusalArmsAreAllWitnessed` and `TestWitnessedArmsAreAllDerived` both
logged `323 derived arms across 9 production files` and passed. The count floor
was green in both rows. **The kill comes from `TestRefusalConstructorsMatchProduction`
and from nothing adjacent.** The rev2 kill-inflation shape is genuinely gone, and
the 323/323 figure is no longer propped up by a replaced-arm artefact.

That closes the question as asked. What it does not close is B6 below.

## G-B — N4 and N5 are closed, and closed at the right level

### N4 — `CheckCallBinding` zero-fact exclusions: 5 of 5 killed

Each mutant keeps the gate and exempts exactly the zero value of the fact it
compares. Each is killed by its own named subtest, driven through
`DecodeManifest` → `Discover` → `CheckRequestBody` → `DecodeTuple` →
`CheckCallBinding`, i.e. the real entry points.

| Mutant | Killed by |
| --- | --- |
| `role != binding.Role && binding.Role != ""` | `…ZeroFactsRefuse/zero_binding_role` |
| `… && binding.ProviderID != ""` | `…/zero_binding_provider` |
| `… && binding.AdapterManifestDigest != ""` | `…/zero_binding_manifest_digest` |
| `… && binding.ExecutableSHA256 != ""` | `…/zero_binding_executable_digest` |
| `… && admitted != (Tuple{})` | `…/zero_admitted_tuple` |

### N5 — the two false census sentences are now true

| Mutant | Result |
| --- | --- |
| paired package-level `var a, b = []string{…}, []string{…}`, consulted from production | KILLED by `TestClosedVocabularyTablesAreRegistered` |
| tagless `switch { case … }` planted in `validCapabilityStatus` | KILLED by `TestAllProductionSwitchesAreClassified` |

`classifySwitch` confines the exemption to `readUTF16Escape` **by function name**,
and `enclosingFunc` returns `""` for a func literal, so a tagless switch inside a
package-level func literal also fails closed. That is stronger than the sentence
promised.

### The 11-shape census denominator, rebuilt independently

Reported outside the gate ratio (these are behaviour-neutral plants, not gate
narrowings). Denominator built by enumerating how a closed set can be written in
Go, not by reading the census.

| Shape | rev2 | rev3 |
| --- | --- | --- |
| package-level slice composite | seen | seen |
| named slice type composite | seen | seen |
| inline `\|\|` comparison chain | seen | seen |
| package-level `map[string]struct{}` | seen | seen |
| tagged switch on a non-`operation` identifier | seen | seen |
| tagless switch over the same members | **blind** | **seen** |
| multi-name package-level `var a, b = …, …` | **blind** | **seen** |
| function-local slice literal | blind | blind — **named in the stated bound** |
| `make()` + `init()` population | blind | blind — **named in the stated bound** |
| `const` string + `strings.Contains` | blind | blind — **named in the stated bound** |
| `regexp.MustCompile` alternation | blind | blind — **named in the stated bound** |

**Vocabularies found over vocabularies that exist: 7 of 11 shapes, and all four
blind spots are declared.** A declared blind spot is a bound, not a defect. N5 is
closed and should not be re-litigated.

## G-C — the traversal, the survivors, and resurrections

Gates were found by traversing production source. Every registered vocabulary
table was widened by one planted member; every obligation loop was grown and
shrunk; every numeric bound was moved by one in both directions; every envelope
echo gate was narrowed to exempt the zero value. Each mutant keeps the gate
present. Every mutant was `go vet`-clean, and production was restored and the
tree OID re-verified after every batch.

**96 narrowing mutants killed of 131 applied (73%); 35 survivors.**

| Class | Applied | Killed | Survived |
| --- | ---: | ---: | ---: |
| Vocabulary / member tables widened by one (all 63 registered) | 63 | **63** | 0 |
| Obligation loops and capability gates | 13 | 10 | **3** |
| Numeric and string bound guards, both edges | 32 | 12 | **20** |
| Envelope echo gates, zero-value exemption | 10 | 0 | **10** |
| B3 additive-arm probes | 2 | 2 | 0 |
| B3 import-alias probes | 2 | 0 | **2** |
| N4 zero-fact exclusions | 5 | 5 | 0 |
| N5 census-sentence probes | 2 | 2 | 0 |
| B1 defect restored / B2 map widened (verification) | 2 | 2 | 0 |

The ratio is **lower** than rev2's 91% and that is not a regression: this
traversal reached two classes rev2's did not measure (see B5). Where the two
overlap, this round is strictly better.

### Round-2 survivors: 11 of 11 closed. Zero resurrections.

| rev2 survivor | Probe this round | Result |
| --- | --- | --- |
| `successMembers` unpinned | widened by one name | KILLED (`TestClosedMemberSetsAreDerivedFromSpec`) |
| `failureMembers` unpinned | widened by one name | KILLED (same) |
| `manifestMembers` unpinned (+ local shadow) | widened by one name | KILLED (same) |
| `doctorResultMembers` unpinned | widened by one name | KILLED (same) |
| 5 × `CheckCallBinding` zero-fact | zero-value exemptions | KILLED, 5 of 5 |
| 8th constructor, inline `axerror.New` | additive arms, floor held at 323 | KILLED, 2 of 2 |

**Resurrection sweep: none.** All 63 tables, every obligation loop, all three
8 MiB frame edges, and the six alias spellings are still dead. The area that
changed production behaviour is clean in both directions:

- restoring the B1 defect (calling the request rule on the success body) reddens
  **`TestValidateSuccessAdmitsEveryMode/archive`** — the archive success is the
  driving case, exactly as asked;
- `checkValidateResult`'s applicable triple and nullable-bool triple both still
  die when shrunk (`…DrivesEveryApplicableCheck/resume_surface_valid`,
  `…/non_boolean_check_member`).

B1 was fixed at the right level and B2 is genuinely pinned, 4 of 4.

## Blocking findings

### B4 (blocking) — the zero-value / absent-fact class is open at 13 more production gates, including every envelope echo gate and the target-write authorization gate

`repeat-of: rev1/F7 (4 sites, closed), rev2/N4 (5 sites, closed this round)`.
Same class, third round, new sites — and this time on the package's outermost
gate and on the gate its own comment describes as "unknown targets never write".

**Ten envelope echo gates admit the empty string.** protocol.go lines 278, 285,
384, 391, 398, 405, 474, 481, 488, 495. Narrowing any one of them to
`(x != want && x != "")` survives the full package suite — **0 of 10 killed.**

Driven at the production entry, not inferred. Baseline vs the protocol.go:278
narrowing, on the package's own valid doctor frame with `"protocol"` replaced by `""`:

```
baseline : DecodeRequestFrame -> session_adapter_protocol_error:
           request protocol is not the session adapter
mutant   : DecodeRequestFrame -> err=<nil>, Request{Operation:doctor,
           RequestID:0198f4c8-…-1234567890ab, Body:{…}}
```

A frame that does not name the session adapter protocol at all is **fully
admitted** and returns a valid `Request`. The full suite stays green.

**Three absent-fact defaults invert silently.**

| Site | Mutant | Result |
| --- | --- | --- |
| `capabilityMapUsable` probe.go:531 | absent name ⇒ `true` | **SURVIVED** |
| `CapabilityUsable` probe.go:466 | absent name ⇒ `true` | **SURVIVED** |
| `CheckDoctorHealthy` probe.go:673 | `!validCapabilityName(name) && name != ""` | **SURVIVED** |

`capabilityMapUsable` is the one that matters. Driven through
`CheckTargetWriteGates`:

```
baseline : capabilities without native_read_back -> capability_unavailable:
                target write misses a usable adapter capability
           capabilities without canonical_write AND official_import ->
                capability_unavailable: neither usable
mutant   : both -> err=<nil>
```

A caller-supplied capability map that is simply **missing** the required
surfaces is admitted for target write. The doc comment states the opposite
("An absent name is not usable… which this gate enforces by refusing every
capability it cannot see"), and the package has no in-repo importers, so no
caller contract makes the branch unreachable.

The asymmetry is visible inside one test function. `TestCheckTargetWriteGates`
ends with:

```go
// A missing provider surface is refused, never defaulted.
missing := map[string]ProviderCapability{"portable_store": …}
err = CheckTargetWriteGates(probe.Capabilities, missing)
requireRefusal(t, err, …, "provider:native_resume", …)
```

Every **adapter** case in the same function sets the capability to
`{Status: "conditional", Enabled: false}` — present but unusable. The absence
rule is proven on the provider half and never on the adapter half of the same gate.

This is the standard negative shape *absent evidence treated as satisfied*, at
an authorization gate, with the correct test one screen above it.

### B5 (blocking) — contract string and array upper bounds are not pinned at the edge: 20 of 32 bound mutants survive

Moving a bound by exactly one admits exactly one member of the reject class.
That is the narrowing shape the DoD asks for, and most of these bounds do not
survive it as evidence.

Proven at the production entry, not inferred:

```
baseline : DecodeManifest, environment_version_range = 257 chars (bound 1..256)
           -> session_adapter_protocol_error: … not a non-empty string[1..256]
mutant   : checkStringBounds(…, 1, 257) -> err=<nil>
```

`go test ./internal/sessadapter/` stays green.

| Killed (edge is pinned) | Survived (edge is not pinned) |
| --- | --- |
| manifest.go:163 `display_name` 1..128 | manifest.go:178 `environment_version_range` 1..256 |
| probe.go:144 `detail` 0..2048 | context.go:613 `native_session_id`, :636 `opaque_source_ref` (1..512) |
| protocol.go:249/355/445 `MaxFrameBytes` (×3) | context.go:847 `code`, :855 `message`, :865 `remediation`, :971 `native_item_key` |
| operations.go:593 `>7`, :1016 `>65536`, :1046 `>9` | operations.go:933 `argv >128`, :942 word 1..4096, :593 lower edge `len==0` |
| tuple.go:790 `>1024`; lower edges at tuple.go:948, :998, operations.go:933 | tuple.go:96, :594, :737, :822, :830, :846, :981, :1155; :948 `>64`; :998 `>32` |

The pattern behind it: the arm witnesses use far-out-of-range values (a
2049-character `detail`, a several-hundred-character blob), which fire the arm
wherever the bound sits. `checkStringBounds` is the sole enforcement point for
every one of these maxima, so `+1` is a real widening with no compensating check.

**This class was inside my rev2 verdict's claimed traversal and was not actually
measured there.** The rev2 121/93 denominator over-reported on bound guards; F6
(the 8 MiB frame edge) was measured and is still closed, the `checkStringBounds`
maxima were not. Recording that so the drop from 91% to 73% is read as a wider
traversal and not as a regression in the work.

### B6 (blocking, and the reason the next step is a gate) — the constructor census is matched by the package identifier, so an import alias walks straight past it

The sentence at inventory_test.go:98 and on `TestRefusalConstructorsMatchProduction`
now reads: *"every `axerror.New` site in production must sit inside one registered
constructor body."* `constructorSitesInFile` implements that as
`receiver.Name != "axerror"`.

| Probe | Result |
| --- | --- |
| M-A3 — second aliased import `axe ".../axerror"` in protocol.go + `axe.New(...)` additive arm | **SURVIVED** the package suite |
| M-A4 — `axe ".../axerror"` in manifest.go (which imports no axerror today) + `axe.New(...)` additive arm in `DecodeManifest` | **SURVIVED `go test ./... -count=1`, exit 0** |

M-A4 is the realistic shape: a producer adding a refusal to a file that does not
yet import `axerror` writes one new import line. Under an alias the arm is
invisible to the constructor census *and* to the arm derivation at once, and the
whole repository suite passes. The same hole applies to the other direction —
`var failQuota = func(...) { return axe.New(...) }` at package level produces no
`produced` entry and no violation.

`repeat-of: rev1/F2, rev2/B3`. Round 1 closed the alias hole in the arm
derivation; round 2 closed the missing production-derived denominator; round 3
leaves the denominator matched by a spelling. **Three rounds, one sentence, each
time narrower.** That is exactly the signal the routing rule describes, and the
fix is not a fourth site patch: the package has **no import census at all** —
nothing pins that `axerror` is imported under its own name in any production
file — so no AST gate in this package that matches a package identifier can be
trusted until one exists.

## Non-blocking findings

### N6 — `constructorSitesInFile`'s own comment is ahead of its switch

The comment says *"Function-local var-held literals are never constructors, so a
New call inside one always reports."* The switch is

```go
case !initialized: violation
case !registered[best]: violation
case topLevel: produced[best] = filename
```

A function-local `var failInvalid = func(...){ axerror.New(...) }` is
`initialized`, `registered`, and not `topLevel` — it falls off the end and
reports nothing. It is defended in depth (a new arm would need a witness, and a
changed code would redden `TestEveryArmWitnessRefusesAtTheProductionEntry`), so
it is not a live hole; it is the same prose-ahead-of-the-check habit in the
sentence right below B6's.

### N7 — the round-3 battery is finding-scoped again

The rework note's 14 mutants are all killed and its method bound is accurate
("battery is scoped to this rework… does not re-traverse the rev2 93-site
surface"). Stating the bound instead of implying a traversal is the right
instinct and I am counting it in the producer's favour. But rev2's requested
change #6 asked for a ratio **over a traversal of production gate sites**, and
round 3 delivered a finding-scoped ratio for the second consecutive round. All
35 survivors above sit outside the 14. A traversal is what distinguishes 14 of
14 from 96 of 131.

## What is correct and must not be re-litigated

- **B1 is fixed at the right level and for the right reason.** The call is gone
  from `checkValidateResult`, the doc comment states why against the pinned §7.8
  success member list, and restoring the defect reddens the archive subtest.
  Resolved by the document, not by a product decision — as it should have been.
- **B2 is closed, 4 of 4.** All four tail maps now die when widened; the
  `manifestMembers` local shadow is gone; the log counts member-map comparisons.
- **B3's additive-arm question is answered.** M-A1 and M-A2 die to
  `TestRefusalConstructorsMatchProduction` alone, with the 323 floor green in
  both rows. The kill-inflation shape that made rev1 and rev2 look proven is
  gone. B6 is a different hole in the same sentence, not a re-run of the same probe.
- **N4 is closed, 5 of 5, at the real entry points.**
- **N5 is closed** and its four remaining blind spots are declared rather than
  hidden; `classifySwitch` names its exemption by function and fails closed for
  func literals.
- **All 63 vocabulary tables are content-pinned** — 63 of 63 killed when widened
  by one member. That is the strongest single piece of evidence in this package
  and it holds up completely.
- **Zero resurrections** across everything I re-measured, including the area
  where production changed.
- **Traceability and README are honest.** `acceptance_cases` 81 → 88 matches
  seven new cases, each naming a real production declaration and real tests; the
  README explicitly keeps the §7.8 clause binding `unevidenced` and states that
  the registry makes no clause-level claim. No unsupported capability is advertised.

## Requested changes

1. **B4** — close the class, do not patch thirteen sites. Add a
   zero-and-absent-fact obligation test that derives its subjects from
   production rather than listing them: for every envelope echo comparison and
   every map-presence default, drive the empty/absent value through the exported
   entry point and require a refusal. `TestZeroValueHostFactsRefuse` and
   `TestCheckCallBindingZeroFactsRefuse` are the right shape already — they just
   do not reach `DecodeRequestFrame`, `CheckSuccessEnvelope`,
   `CheckFailureEnvelope`, `CheckTargetWriteGates`, or `CheckDoctorHealthy`.
   At minimum `CheckTargetWriteGates` must gain the adapter-side twin of the
   provider-side "missing surface is refused, never defaulted" case.
2. **B5** — pin every `checkStringBounds` maximum and every array cap at the
   edge. The cheapest correct form is a derived one: a table of
   (entry point, member, min, max) driven at `max` (admit) and `max+1` (refuse),
   with the same for `min-1`. A witness that is far out of range proves the arm
   exists, not where the bound sits.
3. **B6** — add the import census the package lacks: every production file that
   imports the `axerror` path must bind it to the name `axerror`, and any other
   binding fails. Then the existing constructor census is sound as written and
   the sentence becomes true. Do not special-case aliases inside
   `constructorSitesInFile`; that is the fourth site patch this class does not need.
4. **N6** — make the switch report the function-local registered-name case, or
   change the comment to what the code does.
5. **Evidence** — report the next battery as a ratio over a traversal of
   production gate sites. The 63-table sweep in this verdict is the model: a
   denominator derived from production, applied exhaustively, with the killer
   named per row.

## Stop-the-line

None. Every finding is ordinary rework inside this package, with no external
constraint and no human product, architecture, or approval decision required.
B4 and B5 are bounded, mechanical, and have a correct example already in the
tree; B6 is one new census.

## Method and integrity

- 131 gate-narrowing mutants plus 11 census shape probes, each applied by exact
  string or exact line replacement, grep-confirmed present, `go vet`-clean before
  its verdict, and reverted immediately after.
- Three scratch test files were created under `internal/sessadapter/` to drive
  baseline-vs-mutant behaviour at the production entry, and each was deleted in
  the same call.
- Working-tree OID recomputed after every batch and at the end:
  `9b1c084a1b0a970bc2eed187d5155f4191528c51`, equal to the candidate;
  `git status --short` unchanged (6 modified, 1 untracked).
- `go test ./... -count=1` re-run on the restored tree: 17/17 packages ok.
