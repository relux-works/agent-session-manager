# TASK-260830-2z3se0 rework evidence (round 5, answers rev4)

Round 4 returned changes requested on B4/B5 (closed as lists, not
classes) and N8 (constructor-census indirection). This round plants
two production-derived censuses modelled on
TestClosedVocabularyTablesAreRegistered, closes the classes by the
derived denominators, widens the constructor match, and re-proves
the battery. No production Go file changed: the delta is test
files only (three new/extended census files, two rewired
tripwires), so there is nothing to resurrect by construction;
the full suite is green on identical production.

Working tree left uncommitted for the board CR, as instructed.

## Changed files (all `*_test.go`; zero production changes)

- `internal/sessadapter/bound_census_test.go` (new): B8 bound
  census — scanner, 97-row registry, census test, 12 behavioral
  drivers, shared uint53-ceiling pin.
- `internal/sessadapter/identity_census_test.go` (new): B7
  identity census — 62-gate registry, presence-default census
  with structural proof, zero-row drivers.
- `internal/sessadapter/inventory_test.go`: N8 indirection
  detection (`isAxerrorNew` + `parentNode`), widened census
  sentence, synthetic proof
  `TestConstructorSitesReportFuncValueIndirection`.
- `internal/sessadapter/bounds_edge_test.go`,
  `internal/sessadapter/zero_absent_test.go`: fixed tripwires
  (`!= 17`, `!= 10`) replaced with census counts so the tables
  cannot go stale.

## B8 — bound census (rev4 B5 closed as a class)

Denominator derived from production: every call to the six
helpers with its member, every `len(...)` guard in an `if`
condition (numeric literals normalized out so values are pinned
behaviorally, not statically), and every `DecodeFindings` cap
instantiation. Derived total: **97 sites**.

- Driven at the production entry, both edges: **82 of 97**.
- Exempt with stated rationale: **15 of 97** — 3 shared
  mechanisms (wrapper forwarders, generic count guards, generic
  findings guard; values pinned at caller/instantiation rows),
  3 byte-scanner plumbing guards (surrogate agreement suite),
  1 trim plumbing, 2 equivalent (platform element/count behind
  `scalar.ParsePlatform` + the four-name set), 1 equivalent
  (extension-key floor coincides with the grammar floor —
  proven by applying the mutant, see below), 5 stated-bound
  65536-count rows (canonical_event_ids, the three success
  digests, created_resource_keys count: each needs a
  65537-element body).
- The mechanism note from the brief is rostered, not repeated:
  both `require*` wrappers are mechanism rows; their 19 caller
  sites carry the edge rows (cursor, cwd_relative, the four
  expected_target_native_session_id ordinals, candidate_count
  among them).
- `requireUint53Bounds` finite maxima and the three
  `checkUint53Bounds` 1..maxUint53 minima are driven; every
  maxUint53 maximum points at the shared
  `TestUint53RepresentabilityCeiling` instead of repeating the
  literal per site.

Battery over the driven bound sites: **89 narrowing kills, 0
survived, 0 not-applied**. Every driven site has at least one
in-session kill (both edges for the headline sites: cursor,
cwd_relative, next_cursor, forbid element+count, warnings
count). Each mutant was applied by exact-string replacement,
grep-confirmed PRESENT, driven through its named test, then
restored with restoration verified.

## B7 — identity-gate census (rev4 B4 closed as a class)

Denominator derived from production: every `X != Y` outside the
mechanical classes (error/fault, nil, io.EOF, rune delimiters,
length comparisons — the last rostered by the bound census).
Derived total: **62 gates**. Map-presence defaults live in a
separate 5-row presence census with a structural fail-closed
proof (`TestPresenceDefaultsAreFailClosed`) plus behavioral
drivers.

- Driven at the exported entry, zero on the side the mutant
  would exempt (both sides where reachable): **54 of 62**,
  including the named survivors — DecodeProbe schema echo,
  CheckProbe x6 (+manifest-side, +facts-side rows),
  CheckProbeRequest provider both sides, doctor direction
  (caller side) and entry status, VerifyRequestDigest zero
  context, Discover both sides, CheckBindingEquality 8 facts x
  2 sides, CheckTupleAdmission direction/environment/5 key
  facts x entry+binding sides/status/strategy word,
  CheckCallBinding caller-side x4, provider-loop empty status.
- Exempt with stated rationale: **8 of 62** — mode dispatch and
  capability dispatch (upstream vocabulary guards named),
  4 first-iteration sortedness sentinels, 1 equivalent
  (capture-plan digest: both sides come from
  `requireDigestValue`, mismatch pinned by
  `TestCapturePlanDigestEquality`), 1 equivalent (capability
  `""` unreachable via `validCapabilityStatus`, coherence pinned
  by `TestCapabilityStatusMatrix`).
- Stated bound: a hand-built `Tuple{}` caller param (outside
  the decode-pinned flow); vacuous agreement of two hand-built
  empty structs stays admitted and documented, since honest
  decode never yields empty provider identifiers or digests.

Battery over the driven identity gates: **58 narrowing kills, 0
survived, 0 not-applied** (side-duplicate mutants where both
sides are reachable). Same PRESENT/restore discipline.

## N8 — constructor-census indirection

`constructorSitesInFile` now reports any `axerror.New`
selector outside call-fun position (func-value indirection).
Synthetic proof covers package-level and function-local
indirection (both report), the definition spelling (exempt),
and `axerror.Decode` (exempt). An additive production probe
(`var zzPlantedFault = axerror.New`, since removed) failed the
census with the indirection sentence at the planted position
while `TestAxerrorImportsAreUnaliased` stayed green.

## Fail-closed probes (additive, PRESENT-confirmed, removed after)

One scratch production file carried an unrostered
`requireStringBounds` call, an unrostered `a != b` gate, and the
indirection. All three censuses failed closed naming the
planted sites; the tree is green after removal.

## Equivalent mutants (6, all with stated bounds)

- platforms element/count widening: full suite green
  (unreachable behind `scalar.ParsePlatform` + closed set).
- extension-key floor `< 3` to `< 2`: full suite green (no
  2-character key satisfies the grammar).
- validate-mode body-side, capability-status, capture-digest
  zero exemptions: behavioral suite green; only the identity
  roster reddens, which is the designed fail-closed response
  to a new gate clause, not a behavior change.

## Gates (real exit codes, standalone processes)

- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `gofmt -l internal/`: clean (no output).
- `go test ./... -count=1`: exit 0, 17/17 packages ok.
- `go test ./... -cover -count=1`: exit 0 (sessadapter 82.4%).
- `go test ./internal/sessadapter/ -race -count=1`: exit 0.
- Mutant battery `/tmp/battery.py` (153 mutants, exact-string
  apply, PRESENT-confirmed, restore-verified): 147 killed, 6
  equivalent-confirmed, 0 survived, 0 not-applied. Full
  per-mutant table:
  `TASK-260830-2z3se0_mutant-table-rev5.md`.

## Coverage as ratios over derived denominators

- Bound gates: 82 of 97 driven at production entries (15
  exempt: 4 mechanism, 4 plumbing, 3 equivalent, 5
  stated-bound); 89 in-session narrowing kills, 0 survived.
- Identity gates: 54 of 62 driven at exported entries (8
  exempt: 2 dispatch, 4 sentinel, 2 equivalent) + 5 of 5
  presence defaults proven structurally and behaviorally; 58
  in-session narrowing kills, 0 survived.
- Constructor census: denominator derived both directions +
  indirection refused; synthetic proof green.
- Rev4 zero-resurrection set (76 re-measured kills) stands on
  identical production; suite green with the two censuses
  added (900+ tests in package).
