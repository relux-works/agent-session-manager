# TASK-260830-ljkj8r round 3 — evidence findings C1–C4 closed

Round 2 left four blocking findings, all evidence findings: every gate
refused correctly, but no test would fail if it stopped. This round
replaces the hand-written tables behind those witnesses with
production-derived gates and drives every named hole. Production code
is untouched (byte-identical to the round-2 baseline; hashes below).

## C1 — arm-slide sweep closed with a derived reachability gate

The three misdirected unknown-member vectors are fixed to append at
the true top level, verified by execution to reach the named arm:

- `manifest_test.go` unknown member: was trailing-data-after-object
  (hoisted limits members), now `...,"extensions":{},"extra":1}` at
  the top level → `node manifest carries unknown member`.
- `query_test.go` unknown member: was the operation arm
  (`extensions:{}}` matched `operations[0].parameters` first), now
  `],"extra":1,"caller":` → `directory query carries unknown member`.
- `probe_test.go` unknown member: was the node-build arm (matched
  `node_build` extensions first), now appends to the top-level
  extensions → `probe response carries unknown member`.

The stated "one member per vector" claim in `arm_identity_test.go`
is removed (it was the failure mode) and replaced by
`internal/dirnode/arm_reachability_test.go`:
`TestEveryDerivedArmFiresItsVector`. Every row pairs one
production-derived arm key with a vector driven through a production
entry; the test computes the reached arm from the full refusal
rendering and fails when that is not the named one. Expected texts
are never retyped: literal detail, fault-conduit prefix, and
context suffix come from the derived arm keys, and the wire code
plus message wrapper come from a strict AST derivation over the
eight constructor bodies (`deriveCtorEnvelopes`; any non
`Code:<lit>, Message:<lit>+detail` shape is a violation). A row with
no match, an ambiguous match, or a match outside the
literal/conduit/context/frame shapes fails, never passes. The roster
fails closed both ways: an arm without a row fails unless it carries
a `defensiveArms` rationale, and a row naming an underived arm fails
as orphaned. `TestReachabilityContextsAreClosed` derives the
`checkResponseIdentity` context set (`success`, `failure`) so the
context matcher cannot miss an instantiation.

The gate caught four of my own vectors while being built (neither/
nor renames sliding to unknown-member, a doubled comma, a
filters-anchored extensions needle, a suite-level `Message()` vs
full-rendering mismatch) — each failed its row until the vector
reached the named arm.

Arm battery (production-derived denominator: every `if` whose body
builds `failX`, 144 sites, full-deletion mutant
`if false && (cond)`): **142/144 killed, 0 NOT_APPLIED,
0 COMPILE_FAIL**. The 2 survivors are the two `defensiveArms`
marshal sites (`protocol.go:377` request not encodable,
`scan.go:365` journal not exportable) — behavior-preserving by
construction, rationale stated in the census. The 13 sites C1 named
(3 envelope unknown-member, 5 missingMember, 2 checkExtensions at
`probe.go:392`/`query.go:467`, 3 empty-frame) all die, each with its
reachability row failing. The remaining battery-4 groups (8 strict
conduits, 6 member reads, 2 marshal/parse) die by message
discrimination in the same gate. Full per-site table attached
(`TASK-260830-ljkj8r_arms_table.md`).

One merged key surfaced: `request names an operation outside the
closed registry` is emitted at two sites (`EncodeRequest:338`,
`DecodeRequestFrame:496`) under one arm key. The roster now derives
literal call-site counts and requires one row per emitting site;
each site was flipped alone and fails exactly its own row
(decode-row vs `|via EncodeRequest`-row).

## C2 — obligations derived, driven, resolved

Denominator (`deriveQueryObligations`): every false-returning site
in `query.go` — bare `return false` plus the same-shaped
`return <ZeroLit>, false` — keyed `query.go|<func>|obligation-<n>`.
That is **112 sites**: the reviewer's 89 plus the 11 `checkCaller`
and 12 `checkQueryOperation` tuple sites, which hide behind the
caller arm and the operation arm in exactly the condemned N-into-1
shape. Boolean helpers in other files funnel into message-distinct
arms pinned by the reachability gate, so `query.go` — the file whose
validators collapse N obligations into one message — carries its own
roster (`TestQueryObligationRosterIsComplete`, fail-closed both
ways) and behavioral test (`TestEveryQueryObligationIsDriven`, one
narrow vector per site through `DecodeQuery`).

Obligation battery (one mutant per site, `return false→true`):
**111/112 behaviorally killed**. On the reviewer's 89-subset that is
88/89; all 31 round-2 survivors now die, each under its named census
subtest. Sole survivor is `checkQueryParameters obligation-1` (line
609): unreachable through any public entry while the kind registry
and the parameter-member table agree (the kind lookup precedes it),
so removal is behavior-preserving. Bound, not a hole:
`TestQueryParameterMembersMatchRegistry` fails when the tables
diverge — exactly when the site becomes reachable — and the site is
listed in `defensiveObligations` with that rationale. Its only
battery signal is the roster renumbering artifact, counted
separately below, never as a kill.

Resolution (no two sibling obligations share one witness): the
killer-set mapping shows every witness failing under exactly its own
mutant, plus only the structurally forced extras — the framing gate
above its validator (`checkQueryOperation` obligations 7–12 share
their validator's representative witness; the battery shows the AND
under both flips) and layered delegates (5 filters witnesses over
`checkEnumSubset`/`checkUUIDv7DigestSubset`, each naming both
sites). No witness fails under an unrelated mutant. The diagonal
also caught and fixed a real misassignment I introduced (projection
ob-1/ob-2 rows targeted the wrong sites; the ob-2 site had no firing
census vector until the 135-field registry-cycled vector was added).
Full 112-row killer table attached
(`TASK-260830-ljkj8r_obligations_table.md`).

Census-only kills, counted separately: `TestQueryObligationRosterIsComplete`
fails on every obligation mutant by ordinal renumbering (by
construction). It is excluded from every behavioral verdict above;
no finding claims it as a kill.

## C3 — uniqueness half pinned at all eight scans

`TestSortedUniqueDuplicatesRefuse` feeds a duplicate (sorted, so only
the uniqueness half can fire) at every `>=` scan, each paired with
its deduplicated control admitting: manifest versions, scan
installation digests, hosts UUIDs, contract-assertion encodings,
set-tags, execute-plan confirmations, lineage anchors, journal keys.
Narrowing battery (`>=`→`>` at the eight sites): **8/8 killed, each
1:1 by its duplicate subtest**; unsorted vectors still refuse, so the
sortedness half is unaffected. Census shape 10's claim is corrected
to name both halves and the test that pins each.

## C4 — deadline ceiling driven at the encode entry

`TestUint53BoundEdges/request_deadline_ms` no longer chains the
builder behind a decode refusal. The row drives both entries
independently: in range both must admit; out of range each must
refuse on its own, so an entry admitting out-of-range fails the row
even when the other refuses. The row comment now describes exactly
that. `TestEncodeRequestRefusals` gains the direct ceiling refusal
(`3600001` → `invalid_config`), and the reachability gate carries an
encode-ceiling row asserting the full derived message. Widening
battery at `protocol.go:352` (ceiling+1, dropped ceiling disjunct):
**2/2 killed** by the bound row, the direct encode refusal, and the
reachability row together.

## Production identity and suite

- Production (`internal/dirnode/*.go` excluding tests) is
  byte-identical to the round-2 baseline: decode.go `99ac4a1b…`,
  manifest.go `c7eeac01…`, probe.go (unchanged, hash in manifest),
  protocol.go `a438b041…`, query.go `c0aad9f7…`, scan.go `93f378a0…`,
  bootstrap.go/doc.go unchanged. Worktree HEAD `7c25ae0`; all
  changes are test files inside the untracked `internal/dirnode/`
  package directory (plus the two제도 comment-only corrections the
  findings required). No behavior change; nothing to justify beyond
  the findings.
- `go build ./...`, `go vet ./...`, `gofmt` clean. Full repo suite:
  18 packages green, exit 0. `internal/dirnode` coverage 87.6%
  (was 83.3%).
- Resurrections: none. Spot-verified round-2 kills still die under
  correctly targeted witnesses: manifest conduit `:261`, manifest
  unknown `:268`, probe unknown `:338`, query unknown `:415`,
  params-unknown obligation `:621`.
- Hand-written content, stated: vector construction (a derivation
  cannot invent inputs), the two `defensiveArms` rationales plus the
  one `defensiveObligations` rationale with its tripwire, and the
  shared-witness documentation for framing gates. Everything else —
  arm keys, obligation sites, expected messages, codes, wrappers,
  frame vocabulary, contexts, site counts — is derived in-test and
  fails closed on drift.

## Coverage against the acceptance criteria

- Directory Node major bootstrap: 9 bootstrap reachability rows
  (transport/empty-frame, version, exit-status, downgrade,
  no-lower-major, guards) + `TestBootstrapTerminalFailuresNeverDowngrade`.
- Manifest/probe/scan/query framing: 146/147 arm rows (2
  defensive) + 111/112 obligation rows (1 defensive + tripwire).
- Host checks: 3 observed-binding rows + 5 node-build drift rows.
- Structured errors: envelope derivation + full-rendering message
  assertions on every row; no code-only witness carries a gate.

## Attached

- `TASK-260830-ljkj8r_arms_table.md`: 144 arm mutants × verdict ×
  reachability-row killer.
- `TASK-260830-ljkj8r_obligations_table.md`: 112 obligation mutants
  × behavioral killers (roster noise excluded).
