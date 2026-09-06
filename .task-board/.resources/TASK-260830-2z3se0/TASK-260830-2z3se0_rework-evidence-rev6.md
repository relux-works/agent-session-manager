# TASK-260830-2z3se0 — round-6 rework evidence (review rev5 F1/F2/F3)

Round 5 returned changes requested. All three findings were in the new
instruments, not in shipped behaviour. This round reworks the instruments
only: **zero production bytes changed** (all edits are in three
`*_test.go` files under `internal/sessadapter/`).

Files changed (working tree, uncommitted, per the no-commit shape):

- `internal/sessadapter/identity_census_test.go` — F1
- `internal/sessadapter/bound_census_test.go` — F2
- `internal/sessadapter/probe_test.go` — F3

## F1 — identity census now derives both spellings (was NEQ-only)

`identityGatesInSyntax` derives every `!=` **and** `==` domain
comparison with identical exclusions (nil/err/EOF/rune/len). 37
`==` sites rostered: 31 driven, 6 exempt with upstream-guard
rationales verified against the code (validObjectMode,
validValidateMode, validTupleEntryStatus, checkStringBounds
1..4096, TestResumePlanIdentityComplement; the two caller-param
dispatches state their deterministic branch).

- `discovery.go:105,112` (`== ""` zero refusals) are now rows
  (`discover` domain, driver `TestDiscoverRefusals`).
- 20 vocabulary-table membership positives are rows driven by the
  new `TestVocabularyMembershipRefusesEmpty`, which refuses `""`
  and `"bogus"` at every loop plus `validRole`, `validCandidateKind`,
  and `operationSkipsRequestContext` (tables stay pinned by
  `TestClosedVocabularyTablesAreRegistered`).
- `decode.go|rawUint53|literal == ""` is the single defensive
  exemption: unreachable because `json.Number.String` never yields
  an empty literal and every non-number is refused at the decode
  arms above.
- 4 EQL canaries added; census now rosters 99 gates (85 driven,
  14 exempt). The unqualified sentence ("a new identity gate with
  no row fails the census") is true again for both spellings.
- Synthetic `TestIdentityScanSeesBothSpellings` proves the split,
  including the exact rev5 probe shape (`observed == expected`).

## F2 — bound census: indirection gate + any-context len derivation

Ported the constructor-census shape (review rev4/N8 fix) to the six
bound helpers rather than adding a `var`-form case:

- `boundHelperIndirections` flags **any** use of a helper name that
  is neither a direct call (`Fun` of the enclosing call) nor one of
  the six definitions — var/:=/assignment/argument/return, package
  or function scope, uniformly. A helper alias fails as a census
  violation instead of hiding every bound built through it.
- The `len` pass walks every `BinaryExpr` in the function body, not
  just `if` conditions — covering `for`/`switch`/`return`/
  assignment contexts. Production today gains exactly the six
  reviewer-predicted loop indices (`decode.go` x4, `tuple.go` x2),
  all rostered as plumbing-exempt with the suite that pins each
  loop (`TestSortedUniqueBoundEdges`, surrogate agreement suite,
  `TestUint53RepresentabilityCeiling`).
- Census now rosters 103 sites (80 driven, 23 exempt); 2 for-context
  canaries added.
- Synthetic `TestBoundHelperIndirectionReports` (package-level and
  function-local alias report; direct call and definition exempt)
  and `TestBoundLenScanSeesNonIfContexts` (return/for/switch/assign
  derive; boundless `make` length does not) prove both halves.

## F3 — conditional+enabled settled: refused, and now pinned

Resolution: **no live hole** — the behaviour was already correct;
what was missing was the pin and the stated reasoning.

- `decodeCapabilityValue` refuses `enabled=true` with non-`available`
  status at the boundary (pinned by `TestCapabilityStatusMatrix`),
  which keeps decoded probes clean. But `CapabilityUsable` and
  `capabilityMapUsable` also take caller-built values, so their
  `== "available"` clauses independently refuse
  `conditional`+`enabled` — today, at both sites.
- New `TestCapabilityNonAvailableEnabledIsNotUsable` drives
  hand-built conditional/unsupported/unknown x enabled true/false
  through both predicates (usable only for available+enabled).
- New `TestCheckTargetWriteGatesRefusesNonAvailableEnabled` drives
  hand-built conditional+enabled through `CheckTargetWriteGates`
  on every required adapter capability and the provider side.
- The admit-conditional widening now fails at both sites (M-F3a/b
  below) — the probe the verdict asked for, failing if the decode
  gate were the only thing standing between `conditional` and a
  target write.

## Mutant battery (this round, byte-offset edits, sha256-restored)

Scored the reviewer's way: a failure whose only failing tests are
source-text censuses is `census-only`, never a behavioural kill.
Instrument plants (dead/uncalled code) can only ever be
census-only — that is the correct signal for them, not a gap.

| mutant | narrows the gate to | named failing test(s) | class |
|---|---|---|---|
| M-F1a: `ExecutablePath == ""` -> `== "\x00"` | admits the empty path | `TestDiscoverRefusals` | behavioural |
| M-F1b: `purpose == allowed` -> `... \|\| purpose == ""` | admits `""` purpose | `TestVocabularyMembershipRefusesEmpty` | behavioural |
| M-F1d: `Severity == "error"` -> `== "err"` | admits error findings as valid | `TestValidateResultRules` (+ witness) | behavioural |
| M-F1e: `strategy == "archive_only"` -> `== "archive-only"` | admits archive_only target | `TestCheckTupleAdmissionRefusals` (+ witness) | behavioural |
| M-F1c-probe: reviewer `==` plant, uncalled | new unrostered gate | `TestIdentityGatesAreCensused` only | census-only (correct: dead code) |
| M-F1c-control: reviewer `!=` control, uncalled | new unrostered gate | `TestIdentityGatesAreCensused` only | census-only |
| M-F2a: reviewer `var zzprobeIndirect` plant | hidden bound site | `TestBoundGuardsAreCensused` (indirection) only | census-only (correct: dead code) |
| M-F2b: reviewer return-`len` plant | hidden bound site | `TestBoundGuardsAreCensused` only | census-only (correct: dead code) |
| M-F2c: reviewer `for`-`len` plant | hidden bound site | `TestBoundGuardsAreCensused` only | census-only (correct: dead code) |
| M-F2d: control direct helper call | new unrostered site | `TestBoundGuardsAreCensused` only | census-only |
| M-F2e: cursor max `1024` -> `1025` | admits 1025-char cursor | `TestRequestScalarBoundEdges` | behavioural |
| M-F3a: `CapabilityUsable` admits `capabilityStatuses[1]` | conditional+enabled usable | `TestCapabilityNonAvailableEnabledIsNotUsable` | behavioural |
| M-F3b: `capabilityMapUsable` admits `capabilityStatuses[1]` | conditional+enabled write gate admits | `TestCheckTargetWriteGatesRefusesNonAvailableEnabled`, `TestCapabilityNonAvailableEnabledIsNotUsable` | behavioural |
| M-F3c: provider side admits conditional | conditional provider usable | `TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_provider` | behavioural |

Zero survivors. Every mutant restored byte-identical (snapshot
check passed, no extra files left). One harness note: the first
run of this battery restored via `git checkout`, which is a no-op
on the untracked `sessadapter/` tree, so four edits cascaded and
those verdicts were discarded; the battery was re-run with
byte-copy backup/restore and the table above is from the clean
run only.

## Gates (all exit 0, run this round)

- `go build ./...` — exit 0
- `go vet ./...` — exit 0
- `GOOS=windows go vet ./internal/sessadapter/` — exit 0
- `gofmt -l internal/` — clean
- `go test ./... -count=1` — 17/17 packages ok, zero FAIL
- `go test ./internal/sessadapter/ -count=1 -race` — exit 0
- `tracecheck` — exit 0, acceptance_cases=88

## Coverage ratios (new/changed rows this round)

- F1: 37 of 37 derived `==` rows rostered (31 driven, 6 exempt
  with code-checked rationales); every driven row names its
  production call site and driver above.
- F2: 6 of 6 newly derived `len` rows rostered (plumbing-exempt);
  indirection shape proven by synthetic gate tests.
- F3: 2 of 2 usability `==` rows driven by hand-built
  conditional+enabled vectors through `CapabilityUsable` and
  `CheckTargetWriteGates` (provider side pinned too).

AC rows from prior rounds are unchanged (10 of 10 per the
round-1..5 evidence chain); this round adds instrument closure,
no production behaviour.
