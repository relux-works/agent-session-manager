# TASK-260830-2z3se0 — round-7 rework evidence (review rev6 G-B/F1)

Round 6 returned changes requested with G-A passing, G-C passing, and G-B
failing: the brief asked, for each extended census, what shape comes after
the one just closed, and the rework answered with silence. Three shapes
survived the whole repository suite unrostered (bytes.Equal identity gate,
package-level func-literal scope in both censuses, length bound to a
variable), with a fourth named-but-unplanted (direct cap/RuneCount bounds).

This round does what the verdict routed: for each census, enumerate the
shape space (which AST shapes can carry the guarded thing, which are
derived, which are bounded), extend the two derivations to close the three
demonstrated shapes, and write the residue down as stated bounds in the
census headers. The two unqualified sentences are qualified.

Files changed (working tree, uncommitted, per the no-commit shape) —
**test files only, zero production bytes changed**:

- `internal/sessadapter/identity_census_test.go` — bytes.Equal derivation,
  package-var scope, +1 live row, shape space + bounds, 1 new synthetic test
- `internal/sessadapter/bound_census_test.go` — length-variable rule,
  stringLength source, package-var scope, +4 live mechanism rows, shape
  space + bounds (incl. the 21 accept-direction sites), 1 new synthetic test
- `internal/sessadapter/inventory_test.go` — constructor shape space +
  stated bound (no derivation change; was already complete), 4 new
  synthetic subtests

## G-B answer, per census

### Identity census — 8 shapes: 7 derived, 1 stated bound

| # | shape | disposition |
|---|---|---|
| 1 | `==`/`!=` BinaryExpr in FuncDecl bodies (incl. nested literals) | DERIVED, both spellings |
| 2 | `==`/`!=` in package-level var initializers (incl. func literals) | DERIVED (new; zero live rows, synthetic proof) |
| 3 | `bytes.Equal` call in either scope | DERIVED (new; 1 live row: `context.go\|CheckContextEcho\|bytes.Equal(received.raw, sent.raw)\|0`, driver `TestCheckContextEcho`, domain `context-echo`) |
| 4 | tagged switch statement | statement classified by switch census; arms are arm-inventory obligations |
| 5 | tagless switch statement | classified by switch census (only readUTF16Escape) |
| 6 | map-presence default | presenceRegistrations + fail-closed proof |
| 7 | inline comparison chain | forbidden by chain census |
| 8 | other stdlib equality predicates (reflect.DeepEqual, slices.Equal, maps.Equal, strings.EqualFold, bytes.Compare) | STATED BOUND — none in production; S3 plant proves the bound is real |

Reviewer denominator mapping: their shapes 1-5 stay covered; "stdlib
predicate" splits into shape 3 (closed, live instance rostered) and shape 8
(named bound); "package-level func-literal scope" is shape 2 (closed, zero
live rows — the seven refusal literals hold no comparisons). Census now
rosters 100 gates (86 driven, 14 exempt), was 99.

### Bound census — 10 shapes: 6 derived, 2 stated-pinned, 2 stated bounds

| # | shape | disposition |
|---|---|---|
| 1 | direct call to one of the six helpers | DERIVED |
| 2 | helper through func-value indirection (any position) | DERIVED as violation |
| 3 | `len(...)` comparison, any statement context | DERIVED |
| 4 | comparison over a length variable bound from `len(...)`/`stringLength(...)` | DERIVED (new; 2 live mechanism enforcements = 4 half-rows in checkStringBounds, checkSortedUniqueStrings) |
| 5 | comparison holding a direct `stringLength(...)` call | DERIVED (new; no live instance) |
| 6 | bound inside a package-level var initializer | DERIVED (new; zero live rows, synthetic proof) |
| 7 | cross-field relational gates | STATED-pinned by TestDecodeResourceLimitsBounds |
| 8 | maxUint53 maxima | STATED-pinned by TestUint53RepresentabilityCeiling |
| 9 | decoded-value / caller-supplied number comparison, no length source | STATED BOUND — live: checkUint53Bounds decode.go:269; enforced at every caller row |
| 10 | direct `cap(...)` / `utf8.RuneCountInString(...)` outside stringLength | STATED BOUND — no live instance; S2 plant proves it |

Plus the folded-in accept-direction bound: 21 rows drive the refusal side
only (rev6 battery); the header no longer claims min/max admission per site.
Binding precision, also stated: only a DIRECT binding counts
(`values := make([]T, 0, len(x))` binds no length variable — proven by the
synthetic negative); conversions/parens unwrap; arithmetic combos are out.
Census now rosters 107 sites (80 driven, 27 exempt), was 103.

### Constructor census — 7 shapes: 7 derived-or-refused (no derivation change)

Direct call, package-level constructor literal (8th ctor fails as
unregistered), inline New outside registered bodies, New selector in
non-call position (indirection), failX ident in non-call position
(alias — now proven per position: var/:=/return/argument/struct-field),
aliased error-package import (import census), &frameFault literal. Residue
stated: frameFault-through-type-alias (none exists), non-Go construction
(none used), out-of-dir/_test.go files (never ship).

Overall: every shape the reviewer named is now derived except the four
named bounds (identity-8, bound-9, bound-10, constructor residue), each with
a live-status citation and, for the three plantable ones, a surviving plant
below proving the sentence no longer over-claims.

## Sentences qualified

- Identity: "A new identity gate **in one of the derived shapes below**
  with no row fails the census" (was unqualified).
- Bound: "A new bound **in one of the derived shapes below** with no row
  fails the census" (was unqualified).

## Mutant battery (this round, 15 rows; census-only scored separately)

Rule, the reviewer's: a failure whose only failing top-level tests are
source-text censuses is CENSUS-ONLY. Instrument plants (dead code) can only
be census-only — the correct signal, not a gap. Full table in
`TASK-260830-2z3se0_mutant-table-rev7.md`. Summary:

- 6 census-only kills (P1-P6): every reviewer G-B plant shape now reddens
  its census and nothing else — bytes.Equal FuncDecl, package-literal
  gate+bound (both censuses), len-variable, struct-field helper
  indirection, returned `axerror.New`, inline `axerror.New`.
- 6 behavioural kills (P7, N1a, N1b, N2, N3, N4e): retargeted switch arm
  fails at the arm witness entry; echo inversion fails 19 behavioural
  tests with zero census fires; echo same-length narrowing fails
  `TestCheckContextEcho` ("admitted a same-length rebuilt context");
  mechanism `maximum→maximum+1` fails 5 bound suites ("admitted length
  1025 past the 1024 maximum"); capabilityMapUsable admitting conditional
  fails `TestCheckTargetWriteGatesRefusesNonAvailableEnabled` AT THE WRITE
  GATE (G-A re-pinned, not re-litigated); cursor 1024→1025 fails its edges.
- 3 stated survivors, all green as the headers promise (S1 plain-param
  bound, S2 `cap()`, S3 `reflect.DeepEqual`).

Two expected co-fires, both by design and reported, not hidden: a
length-spelled echo weakening IS a new bound site (`...|len|...` derived),
and any capability-clause edit IS a new identity arm (exact-render pin).
Neither demotes the behavioural kill.

## Gates (all exit 0, run this round)

- `go test ./... -count=1` — 17/17 ok, exit 0
- `go vet ./...` — exit 0; `GOOS=windows go vet ./internal/sessadapter/` — exit 0
- `gofmt -l internal/` — clean
- `go test ./internal/sessadapter/ ./internal/traceability/ -count=1 -race` — exit 0
- `go run ./internal/traceability/cmd/tracecheck` — exit 0, acceptance_cases=88

## No resurrections

Production untouched this round (3 test files only); test inventory
109 → 111 top-level (+2 synthetic tests, 0 removed). All battery
production edits byte-restored (`git diff` on production files empty after
each row; no `zz_*` files remain).

## Explicitly not re-litigated

G-A (reachability settled rev6; re-pinned by N3), G-C prior survivors
(byte-identical production), AC 7 of 7 (unchanged call sites and drivers).
