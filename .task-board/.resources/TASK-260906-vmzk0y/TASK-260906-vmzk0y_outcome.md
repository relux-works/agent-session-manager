# TASK-260906-vmzk0y — own-provider-protocol-v3-envelope-and-error-lift (round 2)

Status: ready for review (handed off to review; board status `to-review`).

Candidate tree: `be5b2844b8996b4764e1cd7a0dfccf1facf478d2`
(verified by `git ls-tree -r`: all 15 changed paths below are in-tree;
blob hashes of `lift.go`, `lift_test.go`, `descriptor.go`,
`descriptor_test.go`, `terminalbackend.go`, `protocol.go`, and
`identity.go` equal the worktree bytes exactly. Built via a detached
index — `GIT_INDEX_FILE` copy + `read-tree HEAD` + `add -A` +
`write-tree` — so the worktree index is untouched. Note: `git stash
create -u` trees omit untracked files; the recorded OID above is the
detached-index tree, which holds them.)

Round-1 findings F1–F5 (blocking) are closed below; F6–F7 notes are
answered. Everything round 1 established and the review marked verified
(v2+v3 pinned both directions, `observedMajor` refactor, inventory
bijection, `tracecheck` closedness, provenance) is preserved: no gate
was weakened — `protocol.go`'s version gate, refusal sites, and literals
are unchanged, and the provhost parse-arm census derives the identical
set (saturation adds no `return 0, false` branch).

## F1 (blocking, closed) — `descriptorGeometry` no longer wraps

`internal/terminalbackend/descriptor.go` refuses on the first digit that
would exceed the bound (`value > 100` → bound arm before the
multiply), so the accumulator never holds more than 100 and cannot wrap
on any platform. `18446744073709552116` (500), `2^64+1` (1),
`2^64+1000` (1000), and the twice-overflowing `55340232221128655348`
(500) are all refused with `descriptor geometry bound`; downstream
never receives a fabricated in-range number. The early refusal is a
second enforcement point of the same rule, so the arm census gains the
`descriptor geometry bound #2` row (witnessed by the overflow test);
bijection holds at 210/210.

## F2 (blocking, closed) — `semverMajor`/`parseMajor` saturate, never wrap

Both helpers saturate to `math.MaxInt` on would-overflow instead of
accumulating past it, so `18446744073709551617.0.0` never reads as
major 1 and `18446744073709551618.0.0` never reads as major 2.
Saturation preserves every comparison against the small constants the
call sites use, in the correct direction. Huge numeric majors stay
*recognized* in provhost and take the `incompatible_protocol`/exit-6
mismatch arm (not the exit-13 unusable-frame arm, never admission).
In-leaf sweep of every other digit-to-number site (same shape, not just
same site): `internal/provhost/opdecode.go:82` (`strconv.ParseUint`,
err → refuse), `internal/axerror/decode.go:246` (`strconv.Atoi`, err →
unsupported major), `internal/axerror/decode.go:351`
(`strconv.ParseInt` 32-bit, err → refuse), `internal/canonicaljson`
`ParseInt`/`ParseUint` sites (err → refuse) all fail closed through
`strconv` by construction. The length-prechecked
`internal/config/endpoint.go` port read (≤5 digits) and the
check-before-continue `parseUint53Literal` shapes (sessadapter,
environ, dirnode) are out of this leaf and were not touched.

## F3 (blocking, closed) — bound rewritten around measured truth

Measured, not inferred: `VerifyObjectIdentity` on the spec's own §5.5
example returns the claimed `record_id`
(`sha256:c879d7…e220c2`) with nil error — the example IS the true
omit-self digest, so the old "merely illustrative" justification is
withdrawn in the bound text itself. Separately, an AST scan over the
module (125 production files, comments and tests excluded) finds zero
production call sites of `VerifyObjectIdentity` repo-wide. The rewritten
bound states both facts, names what is deferred and why (conjoining
Verify here would refuse records this gate documents as admitted —
every boundary admission mutates content while keeping the example
`record_id` — so the mismatched-live-binding ruling belongs to the
identity owner; no board element exists yet — wiring it is the
coordinator's step after this story's final leaf closes), and the two
new tests pin it so a future false justification reddens instead of
shipping.

## F4 (blocking, closed) — `Lift` default arm fails closed both ways

Decision: a registered code with no lift arm must do what an
unregistered code does — produce no wire object. `Lift(Error{code:
not_found})` now returns `(nil, failure)`: the registry check still
refuses unregistered codes with `ErrUnregisteredCode`, and a registered
code outside the three lift arms hands the local failure back as the
error, so the caller keeps the diagnostic without minting a wire object
under a generic message it never reviewed. The doc paragraph now
describes both sub-arms, and "never produces a wire object" is true
again. No new refusal site was minted — returning the input adds no
`Error` literal and no `errors.New`/`fmt.Errorf`/`panic`, which the
provider refusal-inventory audit forbids outside `os_unix.go`/
`os_windows.go`; the full-package audit passes unchanged.

## F5 (blocking, closed) — derivation covers the package and every form

`derivedProviderCodes` now loads from every production file of the
package (const and var, grouped or alone, typed or not) and fails
closed — returns an error that fails the test — on any code-prefixed
declaration it cannot classify (identifier/call value, inherited empty
value list, shared value list, code-prefixed type alias). Control
plants, each a named test over fixture dirs (production untouched):
var binding, second-file const, grouped consts, aliased type +
identifier value (both must error). Control proof the plants are real:
the round-1 derivation run against the same fixtures yields only
`map[held_code:true]`, missing both controls.

## F6 (note, answered) — future caller recorded

`AdmitProviderDescriptor` still has no production caller; the
descriptor doc now records what must call it and when (v3
launch/observe path of the dual-stack follow-up: parse, match, then
launch, no caching across generation changes; wiring it after spawn or
reusing an admitted value across generations would admit a stale
binding), so the next leaf inherits the question.

## F7 (note, answered) — counts reconciled, every row names a test

The round-1 "12 paths" omitted `LOGBOOK.md` (the record carries every
worktree path, 13). This document lists all 15 changed paths below —
11 modified + 4 new — and the candidate tree above is verified to hold
exactly them. AC coverage is 15 of 15 rows driven, each with its
production call site and named test; row 9 (F3) now names two tests
instead of prose.

## AC row coverage: 15 of 15 driven

| # | AC / DoD row | Production call site | Driving test |
|---|---|---|---|
| 1 | v3 envelope has exactly one named owner, recorded in provhost stated bounds and the ownership registry | `internal/provhost/doc.go` (bounds prose), `ownership.v0.5.0.json` `section:7.A` | `tracecheck` (production gate over the registry): ok (60 contracts, 98 cases) |
| 2 | working v3 admission or explicit stated bound naming the deferral | `terminalbackend.AdmitProviderDescriptor` (admission); `provhost/doc.go` (transport deferral + follow-up) | `TestAdmitProviderDescriptorAdmitsMatchingBinding`, `TestAdmitProviderDescriptorRefusesMismatch` |
| 3 | v3 envelope reaching provhost → documented outcome, both directions | `provhost.DecodeResponse` (refuse); `terminalbackend.AdmitProviderDescriptor` (admit) | `TestDecodeResponseWellFormedV3SuccessIsMismatch`, `TestDecodeResponseV3FailureWithValid130ErrorIsMismatch`, admit tests above |
| 4 | Error 1.3.0 binding implemented-or-deferred; no silent major-2 binding for v3 | `provhost.DecodeResponse` (contract derived from observed major), `axerror.BindingFor`/`DecodeBound` (table) | `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100`; the v3 test first decodes its embedded error through the production `(provider, 3)` → 1.3.0 binding, proving refusal of a genuinely well-formed foreign error |
| 5 | F1: geometry bound decided on digits, overflow refused, nothing fabricated | `terminalbackend.descriptorGeometry` (both bound sites) | `TestParseProviderDescriptorGeometryOverflowVectors` (1000 admits; 1001, 2^63, 2^64, 2^64+1, wrap-to-500, 2^64+1000, twice-overflow, 100-digit all refuse; rows spots the shared site) |
| 6 | F2: majors never wrap; wrap-to-native refused at both layers | `terminalbackend.semverMajor`, `provhost.parseMajor` + `DecodeResponse` | `TestParseProviderDescriptorProtocolVersionOverflowVectors`, `TestParseMajorNeverWraps` (parse lie + exit-6 mismatch for wrap-to-2) |
| 7 | F4: registered-but-unlifted code produces no wire object | `provider.Lift` (default arm) | `TestLiftRefusesRegisteredCodeWithoutLiftArm` (not_found: nil wire, no `ErrUnregisteredCode`, input handed back); `TestLiftRefusesUnregisteredCode` keeps the other sub-arm |
| 8 | discovery failure lifts into an axerror Structured Error through an accepted production path | `provider.Lift` (`internal/provider/lift.go`) | `TestLiftCarriesDiscoveryFailuresToTheWire` (4 real-failure vectors) |
| 9 | causal-leak shape refused (negative) | `axerror.New` (production gate) | `TestNaiveLiftShapeIsRefusedByTheWireGate` (message + details directions) |
| 10 | no machine-local path, provider ID, or owner identity on any wire message | `provider.Lift`, `terminalbackend` parse arms | wire-absence asserts in all 4 lift vectors (secrets first proven present locally); `TestParseProviderDescriptorRefusesWithoutEchoingLocalData` |
| 11 | CheckProviderDescriptor completed; registry owns 7.A | `terminalbackend.ParseProviderDescriptor` + `AdmitProviderDescriptor`; registry | 12 inventory arms + rows (bijection 210/210); `tracecheck` ok |
| 12 | F3: bound rests on true facts, both halves pinned | `provhost/identity.go` header (bound prose) | `TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest` (example IS the true digest), `TestNoProductionPathAttestsProviderIdentityBinding` (0 production call sites over 125 files) |
| 13 | F5: census derives the whole package, every form, fail-closed | `loadProviderCodes` (`internal/provider/lift_test.go`) | `TestLiftCoversTheClosedCodeSet` + `TestDerivedCodesIncludesVarBinding`, `TestDerivedCodesScansEveryPackageFile`, `TestDerivedCodesIncludesGroupedConsts`, `TestDerivedCodesFailsClosedOnUnclassifiableBinding` |
| 14 | F6: future caller recorded | `terminalbackend/descriptor.go` (future-caller paragraph) | prose by nature (names the path, order, and two miswirings); the gate it will guard is driven by row 2 |
| 15 | mutation battery with narrowing / arm-deletion / census-only split | battery scripts `/tmp/mutA.py`, `/tmp/mutB.py` | table below; 26 applied, 26 killed, 0 survivors |

## Production changes (15 paths)

- `LOGBOOK.md`: round-2 entry 2251 (findings, gates, handoff).
- `internal/terminalbackend/descriptor.go` (round-1 new): F1 early
  bound refusal in `descriptorGeometry` (second `descriptor geometry
  bound` site); F6 future-caller paragraph in the doc comment.
- `internal/terminalbackend/descriptor_test.go` (round-1 new): F1
  overflow vectors, F2 protocol-version overflow vectors.
- `internal/terminalbackend/terminalbackend.go`: F2 saturating
  `semverMajor` (+ `math` import); M16 target (generation `!=`
  intact).
- `internal/terminalbackend/refusal_arm_inventory_test.go`: F1
  `descriptor geometry bound #2` declared row (+ overflow witnesses
  on both bound rows); bijection 210/210.
- `internal/provider/lift.go` (round-1 new): F4 fail-closed default
  arm (registry check + hand-back; generic-message wire object
  removed) and matching doc.
- `internal/provider/lift_test.go` (round-1 new): F4
  `TestLiftRefusesRegisteredCodeWithoutLiftArm`; F5
  `loadProviderCodes` rewrite + 4 plant tests + fixture helper.
- `internal/provider/doc.go`: unchanged since round 1 (names `Lift`
  as the single production call site).
- `internal/provhost/protocol.go`: F2 saturating `parseMajor` (+
  `math` import); version gate, refusal sites, and literals
  unchanged (parse-arm census derives the identical set).
- `internal/provhost/protocol_test.go`: F2 `TestParseMajorNeverWraps`;
  v3-failure test now proves its embedded 1.3.0 error through the
  production `(provider, 3)` binding before asserting refusal.
- `internal/provhost/doc.go`: unchanged since round 1 (names
  terminalbackend as the §7.A owner, the 1.3.0 binding, the
  dual-stack deferral).
- `internal/provhost/identity.go`: F3 bound rewritten (false
  justification withdrawn, true deferral named, board-element wiring
  noted as the coordinator's step).
- `internal/provhost/identity_test.go`: F3 measurement test +
  attestation tripwire scan.
- `internal/traceability/ownership.v0.5.0.json` +
  `traceability.go`: unchanged since round 1 (7.A →
  terminalbackend, digest `caaca4c9…`).

Leaf-1 production code untouched (`canonicaljson/*`, `environ/*`,
`terminalbackend/manifest.go`, and all other leaves' files
unmodified; `git status` shows only the 15 paths above).

## Mutation battery: 26 applied, 26 killed, 0 survivors

Denominator production-derived: every new arm and touched gate of
this round (geometry bound sites, both major parsers, the Lift
switch + default, the code census) plus the retained round-1
denominator. Classes reported separately; narrowing admits exactly
one member of the rejected class (delete-only where noted).
NOT_APPLIED and COMPILE_FAIL are distinct control rows, not kills.

| Mutant | Class | What it narrows / removes | Named failing test |
|---|---|---|---|
| R1 early overflow refusal deleted | arm-deletion | 2^64+1 admitted as columns 1 | `TestParseProviderDescriptorGeometryOverflowVectors` (`want refusal …, got admission`) |
| R3 saturation deleted from `semverMajor` | arm-deletion | 2^64+1 admitted as major 1 | `TestParseProviderDescriptorProtocolVersionOverflowVectors` (got admission) |
| R4 saturation deleted from `parseMajor` | arm-deletion | 2^64+2 reads as major 2 | `TestParseMajorNeverWraps` (`parseMajor(…) = 2, want other than 1 or 2`) |
| R6 registry check deleted from default | arm-deletion | unregistered codes lose `ErrUnregisteredCode` | `TestLiftRefusesUnregisteredCode` |
| M4 digest-shape arm deleted | arm-deletion | malformed digest admitted by Parse (match still refuses at Admit — layered) | `ValueRefusals` digest rows + no-echo test + inventory (co-kill) |
| M6r2 integrity arm dropped from lift switch | arm-deletion | integrity failures handed back, no wire object | `TestLiftCarriesDiscoveryFailuresToTheWire` (expects a lifted object) |
| R5 fail-open default restored | narrowing | not_found yields a wire object, exit 4, generic message | `TestLiftRefusesRegisteredCodeWithoutLiftArm` (`Lift admitted …: not_found: provider failure carries no liftable code`) |
| M1 proto major-1 rule dropped | narrowing | admits `protocol_version: 2.0.0` | `ValueRefusals/protocol_version_major_2` (got admission) |
| M2 geometry bound 1000→1001 | narrowing | admits columns 1001 | `ValueRefusals/columns_over_bound` (got admission) |
| M3 member-set count half dropped | narrowing | admits extra member | `TestParseProviderDescriptorMemberSet` + inventory bijection (co-kill) |
| M5 lift carries local detail on wire | narrowing | `details: {detail: …}` | `TestLiftCarriesDiscoveryFailuresToTheWire` (details + wire-secret asserts) |
| M7 provhost version gate admits 3.0.0 | narrowing (verdict-change) | 3.0.0 admitted as success | `TestDecodeResponseWellFormedV3SuccessIsMismatch` |
| M12 member-type gate admits null-shape | narrowing | verdict slides to a downstream arm | `TestParseProviderDescriptorMemberType` |
| M13 digest gate admits empty string | narrowing | admits exactly `""` | `ValueRefusals/binding_digest_empty` |
| M14 interactive type gate dropped | narrowing | admits `"true"`/numbers as bools | `TestParseProviderDescriptorMemberType` (got admission) |
| M15 integrity message swapped | narrowing | wrong static message per code | lift test (message exactness) |
| M16 generation match folds case | narrowing | admits `GENERATION-1` | admit-mismatch test (case-variant row) |
| M17 impl-version semver dropped | narrowing | admits exactly `"1.2"` | `ValueRefusals/implementation_version_not_semver` |
| M18 instance check → length check | narrowing | admits v4 UUIDs (36 chars) | `ValueRefusals/instance_uuid_v4` |
| M8 production detail renamed, no test change | census-only | row without arm + arm without row | `TestDerivedRefusalArmsAreAllDeclared` + reverse |
| M9 fourth code const, no lift vector | census-only | unwitnessed code | `TestLiftCoversTheClosedCodeSet` (`production code "experimental" has no lift vector`) |
| M19 code value changed, const name kept | census-only (token-preserving) | static census still passes; behavioral suite must catch | lift positive tests fail (unregistered code); totality passes — proves the harness runs behavior, not only the checker |
| R7 planted const in second file, no vector | census-only | unwitnessed code | `TestLiftCoversTheClosedCodeSet` |
| R8 planted identifier-valued binding | census-only | derivation fails closed | `TestLiftCoversTheClosedCodeSet` (derivation error) |
| R9 derivation drops var bindings | census-only (token-preserving) | `code` token kept; var codes vanish from denominator | `TestDerivedCodesIncludesVarBinding` (behavioral suite, not the checker) |
| R10 unclassifiable bindings skipped silently | census-only (token-preserving) | fail-closed becomes fail-silent | `TestDerivedCodesFailsClosedOnUnclassifiableBinding` |
| M10 absent pattern | NOT_APPLIED (control) | — | — |
| M11 unbalanced brace | COMPILE_FAIL (control) | — | `go vet` exit 1 (distinct from kills) |

Totals: narrowing 13/13 killed, arm-deletion 6/6 killed,
census-only 7/7 killed; 26 applied, 26 killed, 0 survivors.
Survivors: none — no survivor bounds to state. Every mutant file was
restored (sha256-verified) or removed; `git status` shows only the 15
paths above and the tree blobs match the worktree bytes.

## Gate evidence (exit codes observed in this turn)

- `go test ./... -count=1`: exit 0, 22 packages ok, 0 FAIL.
- `go test -race ./internal/provider/ ./internal/terminalbackend/ -count=1`: exit 0.
- `go test -race ./internal/provhost/ -count=1`: exit 0.
- `go test ./... -cover -count=1` on the three touched packages: exit 0 (provider 97.8%, provhost 85.8%, terminalbackend 94.2%).
- `go vet ./...`: exit 0. `GOOS=windows go vet ./...`: exit 0.
- `gofmt -l internal/`: clean (empty). `go build ./...` + `GOOS=windows go build ./...`: exit 0.
- `go generate ./internal/catalog` + `git diff --exit-code -- internal/catalog/`: current (no drift).
- `go run ./internal/traceability/cmd/tracecheck`: exit 0 (60 contracts, 98 acceptance cases).
- Expected-red evidence (all reproduced before fixing, then green after): F1 overflow vectors FAIL admitted-before/refused-after; `TestParseMajorNeverWraps` FAIL `parseMajor("18446744073709551617.0.0") = 1` before; `TestLiftRefusesRegisteredCodeWithoutLiftArm` FAIL `Lift admitted … not_found: provider failure carries no liftable code` before.
- Not run: fuzz smoke — targets live in `secconftest`/`canonicaljson`/`scalar`, none touched and none listed in the touched packages; CI runs it.

## Follow-up: dual-stack v3 host + identity-binding ruling (deferring content)

No board element was created (freeze while the story's final leaf is
open — wiring the ID is the coordinator's step, not this leaf's).
Deferred content, named so the element can be written without
rediscovery: (a) implement the dual-stack provider host — negotiate/
send v3 requests per plugin, carry the §7.A descriptor (built via
`ParseProviderDescriptor`) in v3 launch/resume bodies with
`AdmitProviderDescriptor` on the launch/observe path (parse, match,
then launch; see its future-caller paragraph), decode v3 failures
under Structured Error 1.3.0 through the major-derived contract in
`provhost.DecodeResponse`, and refuse non-`ax.tmux`/`ax.conpty`
backends as `incompatible_protocol` per §7.A/§4.E (no fallback);
acceptance: v3 request bytes on the wire, 1.3.0 failure decode under
test, descriptor mismatch refused before launch from the real call
path. (b) Identity owner's ruling on provider-identity binding
attestation: what a mismatched live binding must do (refuse?
re-resolve? at which layer?) — until it lands, `CheckIdentity` admits
any well-formed digest and `TestNoProductionPathAttestsProvider-
IdentityBinding` guards the deferral. Until both land, the stated
bounds above govern.
