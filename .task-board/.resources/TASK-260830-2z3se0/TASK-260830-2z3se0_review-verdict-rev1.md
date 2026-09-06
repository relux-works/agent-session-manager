# TASK-260830-2z3se0 — reviewer verdict, CR-TASK-260830-2z3se0-1 revision 1

- Verdict: **changes requested** → `to-dev`
- repeat-of: `none` (first revision of this element)
- Reviewed tree: `560edb5038899378e1280358248f2513054ea9df` (base `1cb6b93585749df4ef4755ce93e486bcde9e3a6b`), `repository_delta=present`, 29 paths.
- Working-tree OID verified equal to the candidate tree before and after every mutation batch; the mutation harness restored `internal/sessadapter`, `internal/traceability`, `README.md`, `LOGBOOK.md` from a pre-run copy after each run and the OID is unchanged.

## Baseline facts reproduced by this run

| Check | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./internal/sessadapter/` | exit 0 |
| `go test ./... -count=1` | 17/17 packages ok |
| `go test ./internal/sessadapter/ -cover` | 81.1% of statements |
| `go test ./internal/sessadapter/ -v` | 848 tests, 848 PASS |
| refusal-arm inventory | 323 derived arms across 9 production files, 323/323 witnessed |
| vocabulary census | 55 tables registered across 9 files |
| member-set census | 20 closed bodies derived |
| switch census | 4 classified switches |
| operation table | 14 operations derived |
| lone-surrogate sweep | 6144 vectors refused |
| tracecheck | `acceptance_cases=88`, coverage line unchanged |

## Traversal method and ratio

Gates were discovered by traversing production source (`grep` over comparison
sites, membership functions, member loops, and the eight-fact equality chains),
not by following the test files. Each gate was attacked with a mutant that keeps
the gate present and weakens it to admit one member of the class it must reject;
delete-only mutants were used only where the reject class has a single member.
Compile failures were re-anchored and re-measured, never counted as kills.

**87 mutants applied and measured across 45 gate sites: 63 killed, 24 survived
(72%).** One mutant (`CheckFreshSink` exempting `MaxObjects == 0`) is excluded as
equivalent: `DecodeObjectAuthority` already refuses that combination, so no
decoded authority can reach it.

**Stated bound of this method:** it measures the gates reachable from this
package's exported API. It says nothing about how a future caller sequences
`Discover` → `CheckProbe` → `CheckTupleAdmission` → `CheckCallBinding`, because
no in-repo caller exists yet — nothing outside `internal/sessadapter` imports the
package. It also does not attack `internal/canonicaljson`, `internal/scalar`, or
`internal/axerror`, which several gates delegate to.

## Findings

### F1 (blocking, G-A) — the closed-vocabulary census cannot see non-table vocabularies, and four production vocabularies already live in the blind spot

`isVocabularyShape` (census_test.go:238) matches only an `*ast.CompositeLit`
whose `Type` is `*ast.MapType` or `*ast.ArrayType`. Everything else is invisible.
The file's own claim — "The census finds the membership surface by parsing
production source and fails closed on anything with no registered derivation, so
the surface cannot grow something unwitnessed" — does not hold.

Shape probes (a two-member vocabulary consulted from production, three ways):

| Probe | Shape | Census |
| --- | --- | --- |
| C1 | `var t = []string{...}` + membership loop | **KILLED** (`TestClosedVocabularyTablesAreRegistered`) |
| C2 | `type s []string; var t = s{...}` + same loop | **SURVIVED** |
| C3 | `if v == "a" \|\| v == "b"` | **SURVIVED** |

C3's shape is already occupied. Four closed unions are checked inline and appear
in neither `vocabularyRegistrations` nor `TestValueVocabulariesMatchSpec`:

| Site | Union | Spec | Widening mutant |
| --- | --- | --- | --- |
| `context.go:410` | `mode ∈ {read, fresh_sink}` | `<code>mode:read\|fresh_sink</code>` (same ObjectAuthority sentence the census already parses for `purpose`) | `+reuse_sink` **SURVIVED** |
| `tuple.go:1062` | entry `status ∈ {accepted, revoked}` | §13.14 entry sentence | `+expired` **SURVIVED** |
| `tuple.go:526,622` | `result = pass` (fixture and resume-smoke) | `<code>result=pass</code>` | `+waived` **SURVIVED** |
| `probe.go:592` | `direction ∈ {source_read, target_write}` | doctor row | `+hybrid` **SURVIVED** |
| `probe.go:605` | `registry_entry_status ∈ {accepted, revoked, absent}` | doctor row, same `member:a\|b\|c` form as `fidelity_profile` and validate `mode`, both of which the census does pin via `rowInlineUnion` | `+pending` **KILLED**, but only because the witness happens to feed the literal `"pending"` — the union itself is unpinned |

Control: widening a *table* vocabulary (`capabilityStatuses` + `"deprecated"`) is
**KILLED** by `TestValueVocabulariesMatchSpec`. So the spec pin works; it just
does not reach these five.

The `mode` survivor is not only a census gap. A widened `mode` produces an
`ObjectAuthority` that passes `DecodeObjectAuthority`, then silently skips
`CheckFreshSink` (`context.go:526` returns nil for any mode ≠ `fresh_sink`) *and*
the `mode == "fresh_sink" && (maxObjects == 0 || maxTotalBytes == 0)` coherence
rule at `context.go:433`. A third mode name disables both halves of the fresh-sink
authority with a green suite.

**Failure scenario:** a later leaf adds `mode:read|fresh_sink|reuse_sink` (or any
inline union) to production. Every test stays green, the census reports 55/55
registered, `TestValueVocabulariesMatchSpec` reports nothing, and an authority
mode the spec never defined bypasses the sink-emptiness and limits gates.

### F2 (blocking, G-A) — the refusal-arm inventory's stated alias closure is false for `var x = ctor`

inventory_test.go documents: "Constructor references outside direct-call position
(aliases, variables holding a constructor) fail the derivation outright: an
aliased constructor hides every arm it builds." `collectFileArms` skips any
constructor `*ast.Ident` whose parent is an `*ast.ValueSpec` — which is true of
the *definition's* Name **and** of an alias's Value.

| Probe | Form | Result |
| --- | --- | --- |
| inv01 (control) | new arm through a direct `failInvalid(...)` call | **KILLED** — `TestDerivedRefusalArmsAreAllWitnessed` |
| inv02 | `var plantedFail = failInvalid` at package level, arm built through it | **SURVIVED** |
| inv03 | `var localFail = failInvalid` inside the function, arm built through it | **SURVIVED** |
| inv04 | `localFail := failInvalid` | **KILLED** (parent is `AssignStmt`, so the alias gate does fire) |

`var` is the form the package itself uses for these constructors, so it is the
form a future producer would reach for. The 323/323 figure is therefore a count
over the arms the derivation happens to see, and the closure claim behind it does
not hold.

**Failure scenario:** a producer writes `var failShape = failProtocol` and builds
three refusal arms through it. The derivation returns 323 arms (still ≥ the
census floor), all witnessed, both directions green — and three refusal arms ship
with no witness and no assertion of their identity.

### F3 (blocking, G-B) — the surrogate *pairing* bound is proven on neither edge

`TestLoneSurrogateSweep` generates 6144 vectors and kills every mutant on the
*first* surrogate: `first >= 0xD800` → `> 0xD800` KILLED, `<= 0xDBFF` → `<= 0xDBFE`
KILLED, lone-low `<= 0xDFFF` → `<= 0xDFFE` KILLED. The *second* (low) surrogate in
`hasLoneSurrogateEscape` has only three hand-written pair vectors
(`😀`, `\uD83DA`, and truncations), and both its edges survive:

| Mutant (decode.go:107) | Admits | Result |
| --- | --- | --- |
| `second > 0xDFFF` -> `second > 0xDFFF+1` | `"A\uD800\uE000B"` -- a lone high surrogate followed by U+E000 | **SURVIVED** |
| `second < 0xDC00` -> `second < 0xDBFF` | `"A\uD800\uDBFFB"` -- a high surrogate followed by another high surrogate | **SURVIVED** |

Both admitted forms are lone surrogates that `encoding/json` then rewrites to
U+FFFD — the exact silent byte-rewriting the gate exists to prevent, and the
defect the pinned §1.6 rule is there to stop. This is the same shape the brief
names from the prior Story ("a hand-written vector corpus that cannot tell a
correct bound from an off-by-one"), moved one function inward.

### F4 (blocking, G-B) — set-valued gates are proven on a sample, and one refusal arm hides N obligations

Where a gate loops over a fixed member list, all members share one refusal-arm
identity, so the arm inventory reads green at 323/323 while only the member the
fixture happens to exercise is driven. Loop-truncation mutants:

| Gate | Truncation | Result |
| --- | --- | --- |
| `CheckDoctorHealthy` required-capability loop (probe.go:671) | `required[:1]` | **SURVIVED** — `healthy=true` with `native_read_back`, `native_resume_plan`, `workspace_binding` all unusable is admitted; 1 of 4 proven |
| `checkValidateNullability` archive branch (operations.go:607) | `targets[:3]` | **SURVIVED** — archive mode carrying a non-null `expected_target_native_session_id` is admitted |
| `checkValidateNullability` staged/live branch (operations.go:618) | `targets[:3]` | **SURVIVED** — staged/live with a null `expected_target_native_session_id` is admitted |
| `checkValidateResult` applicable-check loop (operations.go:1107) | drop `resume_surface_valid` | **SURVIVED** — `valid=true` with `resume_surface_valid=false` is admitted |
| `checkValidateResult` nullable-bool loop (operations.go:1081) | drop `resume_surface_valid` | **SURVIVED** |
| `CheckTargetWriteGates` adapter loop (probe.go:483) | drop `workspace_binding` | KILLED |
| `CheckTargetWriteGates` provider loop (probe.go:492) | drop `native_resume` | KILLED |
| `CheckDoctorHealthy` name-validation loop (probe.go:652) | `required[:1]` | KILLED |

`CheckTargetWriteGates` shows the right discipline (`TestCheckTargetWriteGates`
drives each member); `CheckDoctorHealthy`'s required set and the validate
nullability rule do not.

### F5 — the byte-for-byte context echo, this package's stated idempotency binding, is not proven against a same-length rebuilt context

`CheckContextEcho` (context.go:217) narrowed to
`!bytes.Equal(received.raw, sent.raw) && len(received.raw) != len(sent.raw)`
**SURVIVED**. Any echoed context that differs in bytes but not in length — one
flipped hex digit in `operation_id`, `request_digest`, or either digest — is
admitted. `doc.go` names the byte-for-byte echo as one of the two properties that
*are* this package's idempotency, and `session-adapter-idempotency` is one of the
seven acceptance cases registered in `ownership.v0.5.0.json`.

### F6 — the 8 MiB frame bound is unproven at its edge, in both directions

`protocol.go` guards three frames with `len(frame) > MaxFrameBytes`. The fixture
pads with `MaxFrameBytes` characters *inside* an envelope, so it lands ~200 bytes
past the edge.

| Mutant | Result |
| --- | --- |
| `> MaxFrameBytes` → `> MaxFrameBytes+1` (admits a frame of exactly 8 MiB + 1) | **SURVIVED** |
| `> MaxFrameBytes` → `>= MaxFrameBytes` (refuses a frame of exactly 8 MiB, which §7.2 admits) | **SURVIVED** |

Neither side of the stated limit is pinned.

### F7 (non-blocking, name as a bound or fix) — equality gates over host-supplied facts are unproven for the zero-value fact

`ProbeHostFacts`, `SuccessFacts`, and `TrustedCandidate` are plain structs with no
constructor. `Discover` validates its candidate; `CheckProbe`, `CheckProbeRequest`,
and `checkValidateResult` do not check that the host fact they compare against is
populated. Baseline fails closed by accident of string comparison, but nothing
pins it:

- `probe.ProviderID != facts.ExpectedProviderID` → `&& facts.ExpectedProviderID != ""` **SURVIVED**
- `probe.Environment.EnvironmentID != facts.Manifest.EnvironmentID` → `&& … != ""` **SURVIVED**
- `expectedKind != candidate.Kind` → `&& expectedKind != ""` **SURVIVED**
- `mode != facts.ValidateMode` → `&& facts.ValidateMode != ""` **SURVIVED**

A caller that forgets one field currently gets a refusal; one test per gate would
keep it that way.

### F8 (bound, not a demand) — open-class member gates are witnessed with one hand-picked name

`unknownMember` exempting one name (`&& name != "debug"`) and the duplicate-member
gate exempting one name (`&& key != "extensions"`) both **SURVIVED**. The reject
class is unbounded, so no single narrowing mutant is decisive here; recording it
as a stated bound of the arm inventory is enough.

## Evidence-quality finding

The attached `verification-evidence.log` reports `killed-or-green=8 unexpected=0`
from five mutants (M1 `capabilityStatuses` widened, M2 `findingSeverities`
widened, M3 `requestMembers` widened, M4/M4b `validCapabilityStatus` return-true,
M5 unregistered vocabulary table). None are delete-only — that part is right —
but all five land on the *table*-shaped surface that is already spec-pinned, and
the run reports no killed/measured ratio over a gate traversal. The DoD row
"Every gate ships at least one NARROWING mutant" is checked against evidence
covering 5 of ~45 gate sites. The independent traversal above found 24 survivors,
six of which are behavioral.

## What is correct and should not be re-litigated

- **G-C traceability is sound.** The 81 → 88 re-pin is computed, not copied:
  editing a single acceptance-case declaration reddens with an exact digest
  mismatch (`ownership registry projection digest %s differs from reviewed %s`),
  and renaming a production declaration is refused by
  `TestVerifyRepositoryRejectsForgedOwnershipAndCapabilityClaims`. All seven new
  rows name declarations that exist. The README §7.8 paragraph is true of the
  tree: `section:7.8` is still bound to `catalog.ForRelease` with
  `coverage: "unevidenced"`, the coverage line is byte-identical before and after,
  and the README says so rather than claiming clause coverage the registry does
  not carry.
- **The operation-body derivation is the right shape.** Probe points come from the
  pinned document and verdicts from production;
  `TestOperationBodiesRefuseDerivedMutations` drops every derived member in turn.
  Dropping a member from `requestBodyMembers` or `successBodyMembers` is killed
  from three directions.
- **Bounds are character-counted and two-sided.** `checkStringBounds` and
  `checkUint53Bounds` die on `maximum+1` and on `minimum-1`, and
  `TestDecodeManifestValueRules` carries both an ASCII and a multibyte 129-character
  fixture — the byte-vs-character defect from the prior Story is closed.
- **The eight-fact binding equality is complete.** Deleting each of the eight facts
  in `CheckBindingEquality` is killed 8/8 by `TestBindingEqualityFlipsEveryFact`.
- **Refusal arms assert their identity.** Every `valid*` membership function
  neutralised by a token-preserving `return true` is killed 14/14, and the failing
  witness names the arm (code + distinguishing detail), not merely "something
  refused". The 0-of-29 defect from the prior Story is closed.
- Registry order, sorted-unique strictness, capability count, tuple admission
  (status, validity upper edge, `archive_only` exactly-one, missing smoke),
  capability coherence, validate structural/semantic and error-finding rules,
  resource-limit cross rules, UTF-8 and trailing-data gates: all killed.

## Requested changes

1. **F1** — pin `mode`, `registry_entry_status`, entry `status`, `direction`, and
   `result=pass` against their pinned-document spans (all five are in the
   `<code>member:a|b|c</code>` or `<code>member=value</code>` form the census
   already parses elsewhere), and close the census shape gap so a vocabulary
   declared through a named type or an inline comparison chain cannot be added
   unregistered. Correct the census doc comment to match what the scanner sees.
2. **F2** — treat a constructor identifier appearing as a `ValueSpec` *Value* as an
   alias and fail the derivation, keeping the definition's *Name* exempt. Re-run
   the inv02/inv03 probes as regression witnesses.
3. **F3** — generate the surrogate *pair* corpus the way the lone corpus is
   generated: for each high surrogate, sweep the second escape across the low
   range and both adjacent code points, so `\uD800\uE000` and `\uD800\uDBFF`
   refuse by construction.
4. **F4** — drive every member of each set-valued gate, or split the arm so each
   member carries its own detail. `CheckDoctorHealthy` (all four required
   capabilities) and `checkValidateNullability` (all four targets, both branches)
   are the load-bearing ones.
5. **F5** — add a same-length, different-bytes context echo case.
6. **F6** — pin the frame bound at exactly `MaxFrameBytes` (admitted) and
   `MaxFrameBytes + 1` (refused).
7. **F7** — either add the zero-value host-fact cases or record the bound
   explicitly.
8. **Evidence** — report the mutant battery as a killed/measured ratio over a
   traversal of production gate sites, and name what the method cannot see.

No stop-the-line boundary was found: every finding is ordinary rework inside this
package's own scope, with no external platform, product, or ownership decision
required.
