# TASK-260830-2z3se0 — reviewer verdict, CR revision 5

- Verdict: **changes requested** → `to-dev`
- repeat-of: `rev4/N8` (F2 — same prose-ahead-of-the-check shape, new instruments).
  F1 and F3 are `none`.
- Candidate tree `c0174e8761518432e55ed5bdcf915959da9bd9f4`, base
  `1cb6b93585749df4ef4755ce93e486bcde9e3a6b`, verified equal to the worktree
  before and after every probe (`git read-tree HEAD && git add -A && git write-tree`).

## Gate results

| Gate | Result |
| --- | --- |
| G-A — bound class over the production-derived denominator | **passes** |
| G-A — zero/absent-fact class over the production-derived denominator | **passes** |
| G-B — resurrections, and the 32 → 34 path delta | **passes** |
| G-C — everything left open is stated | **fails** (F1, F2, F3) |

Baseline before any mutant: `go build ./...` exit 0, `go vet ./...` exit 0,
`gofmt -l internal/` clean, `go test ./... -count=1` 17/17 packages ok (29.7s).

## What I measured myself

262 narrowing mutants, each applied by byte-offset edit into production,
PRESENT-asserted, driven through the **whole package suite**, restored with a
sha256 comparison. Killers are classified: a failure whose only failing tests
are the source-text censuses is recorded as `CENSUS_ONLY`, never as a kill.
That distinction matters here — every identity mutant of the form
`a != b` → `a != b && a != ""` adds a derived row and reddens
`TestIdentityGatesAreCensused` on its own.

| Battery | applied | behavioural kills | survived | census-only | compile-fail |
| --- | ---: | ---: | ---: | ---: | ---: |
| `require*` both edges, all 19 call sites | 35 | 33 | 2 | 0 | 0 |
| `check*` every literal edge (68 edges) | 68 | 60 | 8 | 0 | 0 |
| `len` guards in `if` conditions + tuple/env rows | 22 | 19 | 1 | 2 | 0 |
| identity NEQ, both sides of all 62 derived rows | 119 | 73 | 0 | 34 | 12 |
| `==`-spelled refusal gates (outside both censuses) | 13 | 10 | 0 | 3 | 0 |
| typed re-runs of the 5 above that needed real types | 5 | 3 | 2 | 0 | 0 |
| **total** | **262** | **198** | **13** | **39** | **12** |

### G-A, bound class — closed

The brief's specific worry was that `requireStringBounds` callers might be
pinned individually while the wrapper stays unmeasured. Both readings are
answered: **all 19 `require*` call sites are driven at both edges**, and every
finite edge dies.

- 16 minimum edges (`1` → `0`) — 16 killed.
- 17 finite maximum edges (`N` → `N+1`) — 17 killed.
- 2 `maxUint53` maxima (`maxUint53` → `maxUint53+1`) — survived, and **correctly
  stated**: `parseUint53Literal` (decode.go:225) refuses any literal above
  `maxUint53` during parsing, so no input reaches the widened band. The registry
  rows say exactly this and `TestUint53RepresentabilityCeiling` proves the
  mechanism at `rawUint53` and `checkUint53Bounds`.

Across the 68 `check*` literal edges plus the 19 `len` guards, 79 of 90 died.
All 11 survivors trace to a committed row, and I verified each rationale rather
than accepting it:

- 3 × `manifest.go:185` platforms (element min, element max, count max) —
  registry says "unreachable behind `scalar.ParsePlatform` and the four-name
  closed set". True: `manifest.go:192-201` runs every element through
  `scalar.ParsePlatform`, which refuses everything outside
  `macos|linux|wsl2|windows`, `""` included, and a fifth distinct member does
  not exist. The row's wording names element 16 and count 4; the element
  *minimum* is equally unreachable but is not spelled out.
- 5 × `65536` count ceilings (`canonical_event_ids`,
  `canonical_event_candidate_ids`, `raw_reference_ids`,
  `required_source_object_ids`, `created_resource_keys`) — stated bound,
  65537-element body.
- 1 × `decode.go:441` `len(name) < 3` → `< 2` — registry says no 2-character key
  satisfies the reverse-DNS grammar. Correct.
- 1 × `protocol.go:527` `len(trimmed) == 0` — rostered plumbing, a trim helper
  with no refusal. Correct.
- 1 × `tuple.go` caller-side `environment.EnvironmentID` — the entry side dies
  (`TestTupleAdmissionZeroFactsRefuse`); the caller side is the documented
  hand-built-caller-param bound.

**No unstated bound survivor.**

### G-A, zero/absent-fact class — closed

Over the 62 derived rows I generated 119 mutants (both operands of every
reachable row) and completed the 4 rows my generator could not type
(`count != 1`, `occurrences != 1`, `context.Environment != admitted`,
`entry.Key.Environment != environment`) with hand-typed equivalents.

**54 of 54 driven rows produce at least one behavioural kill.** Zero survivors.
Zero of the 8 exempt rows was killed by my battery, so no row is exempted that
a test would in fact catch.

The 34 `CENSUS_ONLY` results are all one of: an exempt row behaving as declared;
a side that is a package constant (`ProtocolID`, `probeSchemaVersion`,
`"accepted"`, `"archive_only"`, `capabilityOrder[index]`) where the added clause
can never fire; a side already refused by an earlier gate in the same function;
or the stated hand-built-caller-param bound. None is an unmeasured fact side.

### G-B — resurrections and the 32 → 34 delta

Reconstructing the rev4 tree from its committed patch
(`1469c6348e064454dd31a5a727e25486ed467b1b`) and diffing it against the rev5
candidate:

```
git diff 1469c634 c0174e87 -- '*.go' ':(exclude)*_test.go'   # empty
```

**Zero production bytes changed between rev4 and rev5.** Resurrection is
impossible by construction, and my 262 mutants re-measure the same production
the rev4 battery measured. The delta is `LOGBOOK.md` (+9), the two new census
files (+2072), `inventory_test.go` (+117 for N8), and the two rewired tripwires
(±4 each). 32 → 34 = `bound_census_test.go` + `identity_census_test.go`.

The two tripwire rewires strengthen rather than weaken: `TestStringBoundEdges`
and `TestEnvelopeEchoEmptyStringRefuses` now take their expected row count from
the registry instead of a frozen literal, so a new registry row forces a new
driver row.

N8 is genuinely fixed. Planting `var zzNewFault = axerror.New` reddens
`TestRefusalConstructorsMatchProduction` with the indirection sentence at the
planted position.

## Findings

### F1 — the identity census derives only `!=`, and two live zero-fact gates are spelled `==`

`scanIdentityGateIDs` (identity_census_test.go:66) matches `token.NEQ` only. The
file's own sentence is unqualified: *"A new identity gate with no row fails the
census."* It does not.

`discovery.go:105` and `discovery.go:112` are zero-fact refusals in the exact
class this leaf has reworked for four rounds, written the other way:

```go
if candidate.ExecutablePath == "" { ... failInvalid("discovery candidate carries no executable path") }
if candidate.OwnerIdentity  == "" { ... failInvalid("discovery candidate carries no owner identity") }
```

Neither is in the 62. Both are behaviourally covered today —
`TestDiscoverRefusals` kills a narrowing of each — so nothing is currently
uncovered. The gate is what fails: an added production file containing

```go
func zzProbeEqualityRefusal(observed, expected string) error {
    if observed == expected { return nil }
    return json.Unmarshal(nil, nil)
}
```

passes the entire package suite unrostered. The control plant confirms the
probe is fair: the same file written as `if observed != expected` reddens
`TestIdentityGatesAreCensused` naming the planted site.

Either extend the denominator to `==`-spelled refusals — starting with the two
live `discovery.go` rows and their existing drivers — or narrow the sentence and
declare the spelling as a stated bound naming those two sites.

### F2 — the bound census has the N8 blind spot that was fixed for the constructor census in this same commit

`repeat-of: rev4/N8.`

`scanBoundSiteIDs` (bound_census_test.go:126-131) takes `call.Fun.(*ast.Ident)`
and returns early on anything else, and its `len`-guard pass walks `*ast.IfStmt`
conditions only. Its sentence is likewise unqualified: *"A new bound with no row
fails the census."*

Three token-preserving plants, each added to production and run through the full
suite, are invisible:

| plant | suite |
| --- | --- |
| `var zzprobeIndirect = requireStringBounds` + `zzprobeIndirect(members, "zz_probe_member", 1, 4096)` | green |
| `return len(value) >= 1 && len(value) <= 512` (bound in a `return`, not an `if`) | green |
| `for i := 0; len(values) > 128 && i < 1; i++` (bound in a `for` condition) | green |

The first is literally the N8 shape. rev4 asked for it, the producer built
`isAxerrorNew` + `parentNode` indirection detection for the constructor census
this round, and then shipped a new census with the same hole. The fix already
exists in `inventory_test.go` and needs porting to the six bound helpers.

This is a durability finding, not a coverage one: I verified the current
denominator is complete for today's source. An independent any-call-shape scan
of the package finds exactly **58** calls to the six helpers, and the census
derives **58**. The only `len` comparisons outside an `if` condition are six
loop indices (`index < len(...)` in decode.go ×4, tuple.go ×2), none of them a
domain bound. Nothing is unmeasured today; the instrument that two more leaves
inherit is what fails open.

Either widen the derivation (indirection for the six helpers, `len` guards in
`switch`/`for`/`return`/assignment contexts) or narrow both sentences and state
the bound.

### F3 — unstated survivor: the capability-usability status clause

```go
// probe.go:469
func CapabilityUsable(probe Probe, name string) bool {
    value, present := probe.Capabilities[name]
    if !present { return false }
    return value.Status == "available" && value.Enabled
}
// probe.go:534 capabilityMapUsable — identical body over a raw map
```

Narrowing the status clause to also admit `capabilityStatuses[1]`
(`"conditional"`) — no new literal, no new `!=`, so no census fires — leaves the
**full package suite green** at both sites. These are authorization predicates:
`capabilityMapUsable` is the whole of `CheckTargetWriteGates` at probe.go:497
and :505, and `CapabilityUsable` gates the doctor required-capability check at
probe.go:693.

I believe it is equivalent, and I checked why rather than assuming:
`decodeCapabilityValue` refuses `enabled=true` with a non-`available` status
(probe.go:114), and `TestCapabilityStatusMatrix` drives all eight
status × enabled combinations through `DecodeProbe`. So no probe produced by the
decoder can carry `conditional`+`enabled`. But `CapabilityUsable` is exported
and takes a caller-built `Probe`, the two functions appear in the identity
census only as *presence-default* rows (`outcome: false` when absent, driver
`TestCapabilityUsableAbsentIsNotUsable`), and the equivalent list in the rework
note covers `decodeCapabilityValue`'s own `status != "available"` echo — not
this clause.

That makes it exactly the shape G-C names: a survivor with no stated bound. It
needs one row or one comment naming the upstream coherence gate, or a drive of a
hand-built `Probe{Capabilities: {"x": {Status: "conditional", Enabled: true}}}`
through `CheckTargetWriteGates`.

Lineage note, so routing is not misled: rev3/B4 named these same two functions
and closed their **absence** path (`TestCapabilityUsableAbsentIsNotUsable`, and
the rows in the presence census). This is their **status** clause, a different
class, so I record `repeat-of: none` rather than a fourth round on the
zero/absent-fact class. It is adjacency, not recurrence.

## Explicitly closed — do not re-litigate in round 6

- The bound class over its production-derived denominator. 122 mutants,
  110 behavioural kills, 12 survivors all traced to a committed row whose
  rationale I verified against the code.
- The zero/absent-fact class over its production-derived denominator.
  54 of 54 driven rows behaviourally killed; 8 of 8 exemptions consistent.
- Both censuses fail closed on the shapes they do cover (control plant reddens
  both, naming the planted sites).
- N8, the constructor-census indirection.
- Resurrections: none possible; production is byte-identical to rev4.
- README/traceability: acceptance cases 81 → 88, the 7 session-adapter cases
  name real committed tests, and the README states the Section 7.8 clause
  binding stays `unevidenced` rather than claiming clause coverage. Honest.

## Reproduction

```bash
# tree identity
git read-tree HEAD && git add -A && git write-tree     # c0174e87…

# rev4 → rev5 production delta (empty)
git diff 1469c634 c0174e87 -- '*.go' ':(exclude)*_test.go'

# F1 / F2 plants: add the probe file to internal/sessadapter, then
go test ./internal/sessadapter/ -count=1

# control plant (must redden both censuses)
#   func zzControlBound(m map[string]json.RawMessage) error {
#       return requireStringBounds(m, "zz_control_member", 1, 4096) }
#   func zzControlIdentity(a, b string) bool { if a != b { return false }; return true }

# F3
#   probe.go:469 / :534  value.Status == "available"
#     -> (value.Status == "available" || value.Status == capabilityStatuses[1])
go test ./internal/sessadapter/ -count=1     # green
```
