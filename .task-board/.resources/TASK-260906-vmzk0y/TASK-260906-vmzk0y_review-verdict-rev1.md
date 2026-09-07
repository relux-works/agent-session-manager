# TASK-260906-vmzk0y — review verdict, CR revision 1

Verdict: **changes requested** → `to-dev`.
repeat-of: `none` (first revision of this element; no prior finding IDs).

Reviewer run: RUN-260906-205198. Every claim below was reproduced by this run in
an isolated byte-identical copy of the candidate
(`.temp/STORY-260905-3t31e9/review-probe`, rsync of the worktree minus `.git`,
`.temp`, `.task-board`). The candidate worktree was never written to: after every
plant the mutated file was restored from a backup and `diff` confirmed byte
equality, and the final `git status --short` in the worktree is exactly the 9
modified + 4 untracked paths of the Change Request.

## Provenance (G-E) — clean

| Check | Result |
|---|---|
| CR tree `c027830980c32c1a90e823ebf1e101ba096af821` exists and is a tree | yes |
| `ls-tree` contains the four new files (`provider/lift.go`, `provider/lift_test.go`, `terminalbackend/descriptor.go`, `terminalbackend/descriptor_test.go`) | yes — they are real blobs in the tree, not dropped-untracked |
| all 13 changed paths: `git hash-object` of the worktree byte-for-byte equals the CR tree blob | 13 of 13 OK |
| base tree of `1d97474` | `764fabe…` — which is also what a bare `git write-tree` returns here, confirming the index was never staged and the recorded OID came from the detached-index build |
| leaf-1 packages `internal/canonicaljson`, `internal/secprim`, `internal/secconftest`, `internal/environ` vs `1d97474` | empty diff, no untracked files |

## Gates re-run by this reviewer (not accepted from the outcome)

| Gate | Result |
|---|---|
| `go test ./... -count=1` | exit 0 except `internal/specpin`, which fails only because the probe copy excludes `.task-board`; re-run in the real worktree: `ok internal/specpin` |
| `gofmt -l internal/` | clean |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, acceptance_cases=98 |

## What holds up

- **Provhost v2-only refusal is pinned in both directions.** Refuse: the three
  new tests. Admit: pre-existing `TestDecodeResponseAcceptsSuccess` and
  `TestDecodeResponseReturnsChildFailure`. Verified by mutant, not by reading —
  widening the version gate to admit `3.0.0` (both the peek and the strict gate)
  reddens `TestDecodeResponseWellFormedV3SuccessIsMismatch`,
  `…V3FailureWithValid130ErrorIsMismatch`, `…RecognizableMajorMismatch`,
  `…ForeignMajorPrecedesMemberRules`, `TestEveryArmWitnessRefusesAtTheProductionEntry`,
  `TestCallRefusesUnusableStdout`.
- **The `observedMajor` refactor is not a weakening and is pinned.** The gate at
  `protocol.go:482` is exact string equality against `"2.0.0"`, so
  `parseMajor` is total and always yields 2 on the reachable path. Mutating the
  line to `Major: observedMajor + 1` reddens four tests including
  `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100`. `{provider, 3} →
  Version130` already exists in `axerror.staticBindings`, so the doc's claim
  that 1.3.0 is "named here and implemented in axerror" is true.
- **The terminalbackend refusal-arm inventory is a real bijection with a derived
  denominator.** Control-planted both directions: a new arm in `descriptor.go`
  with no row → `derived arm has no declaring row … descriptor planted arm` plus
  `derived 210 arms but declared 209 rows`; a row with no arm → `declared row …
  resolves to 0 derived arms`.
- **The traceability registry fails closed on a fabricated owner.** Control-planted
  both a production path that does not declare the symbol
  (`declaration "AdmitProviderDescriptor" is absent from
  "internal/terminalbackend/conformance.go"`) and a test declaration that does
  not exist (`declaration "TestAdmitProviderDescriptorGhostBinding" is absent`).
  Ownership of `section:7.A` is recorded in the registry, not only in prose.
- **The lift's four narrowing mutants I reran all die.** Geometry bound
  `1000→1001` and `1→0`; `message = failure.Error()` on one arm; details
  carrying `failure.detail`. Each reddens a named test.
- **The F3 statement's factual half about the terminal side is correct.**
  `terminalbackend.checkIdentity` really does recompute (`manifest.go:692`,
  called at 1049/1231/1387 for `manifest_id`, `probe_id`, `evidence_id`), and
  `provhost.CheckIdentity` really does only `isDigest(digest)` at
  `identity.go:156`.
- **No deferring task ID is a brief constraint, not a producer omission.** The
  developer's spawn brief says verbatim: "Do not create board elements under
  STORY-260905-3t31e9 while its final leaf is open. Say what you need in your
  outcome instead." The proposed task content is in the outcome. Not held
  against this revision; it is an orchestrator follow-up.

---

## F1 (blocking) — the §7.A geometry bound admits values outside it, and rewrites them

`internal/terminalbackend/descriptor.go`, `descriptorGeometry`.

`value = value*10 + int(digit-'0')` accumulates into `int`. A JSON literal past
2^63 wraps, and the wrapped residue is then checked against `1..1000`. The gate
does not merely admit an out-of-range value — it fabricates an in-range one the
document never carried, and hands it to the caller as a validated `uint16`.

Reproduced through the production entry `ParseProviderDescriptor`:

| `"columns"` literal | Outcome |
|---|---|
| `18446744073709552116` (2^64+500) | **ADMITTED**, `Columns = 500` |
| `18446744073709551617` (2^64+1) | **ADMITTED**, `Columns = 1` |
| `18446744073709552616` (2^64+1000) | **ADMITTED**, `Columns = 1000` |
| `36893488147419103732` (2·2^64+500) | **ADMITTED**, `Columns = 500` |
| `1000000000000000000000000000500` | refused, `descriptor geometry bound` |

This is not layered behind anything: geometry is explicitly "carried context,
matched to nothing" (`Binding()` does not project it), so `AdmitProviderDescriptor`
admits it too. The descriptor is an untrusted document from a provider process.

Why the suite did not catch it: `TestParseProviderDescriptorValueRefusals` tests
`0`, `1001`, `65535` and `TestParseProviderDescriptorGeometryBounds` tests `1`
and `1000` — all adjacent to the edge. The declared class is "outside
`uint16[1..1000]`", and the far half of that class is unwitnessed. The M2 mutant
(`1000→1001`) is an adjacent-edge narrowing and cannot see this. The file comment
claims "the Go type holds the uint16 half and the parse entry holds the 1..1000
half"; the parse entry does not hold it.

Required: refuse on accumulator overflow (or bound the digit count before
accumulating), and add the far-out-of-range literal to the refusal table so the
class is closed rather than its edge.

## F2 (blocking) — the same overflow in `semverMajor`, newly consumed by the §7.A gate

`internal/terminalbackend/terminalbackend.go:245` (`semverMajor`), consumed by
the new `descriptor.go` check `semverMajor(protocolVersion) != 1`.

`semverPattern` is anchored but admits an arbitrarily long major
(`[1-9][0-9]*`), and `semverMajor` accumulates into `int` with the same wrap.

| `"protocol_version"` literal | Outcome |
|---|---|
| `18446744073709551617.0.0` (2^64+1) | **ADMITTED by `ParseProviderDescriptor`** as major 1 |
| `18446744073709551618.0.0` | refused, `descriptor protocol version` |

Less severe than F1 because `AdmitProviderDescriptor` then requires byte equality
with the host binding's `ProtocolVersion`, so the full gate still refuses. But
`ParseProviderDescriptor` is itself a declared production entry — it is the
`entry` column of eleven rows in `refusal_arm_inventory_test.go` — and its
major-1 rule is defeated. `semverMajor` is pre-existing code; this CR is the
first caller that uses it as an admission rule on an untrusted document, so it
is in scope here.

## F3 (blocking) — the new stated bound in `identity.go` justifies itself with a false fact

`internal/provhost/identity.go:30-47`. The bound says the gate cannot verify the
`record_id` binding because it "must admit the pinned Section 5.5 example itself,
whose `record_id` is illustrative rather than the true omit-self digest."

Driven:

```
canonicaljson.VerifyObjectIdentity(specIdentityExample)
  = "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2", err = <nil>
```

The example's `record_id` **is** the true omit-self digest. The stated reason does
not hold. Adding `VerifyObjectIdentity` after every member arm in `CheckIdentity`
leaves the shipped §5.5 fixture admitted; the only two failures are hand-derived
positive fixtures in `identity_test.go` (the 128-character `provider_version`
variant and the `codex session_uuid` null-realm variant) whose `record_id` was
never recomputed after substitution. Those are test-fixture debt, not a spec
constraint.

Second half of the same paragraph: "Binding attestation belongs to the identity
owner at the persistence layer, through `VerifyObjectIdentity`." Grep over every
non-test file in the repository:

```
internal/canonicaljson/canonical.go   — the declaration and two comments
internal/provhost/identity.go         — these two comment lines
```

`VerifyObjectIdentity` has **zero production call sites**. The bound points the
reader at a layer that does not perform the attestation either, so the reader is
told the binding is checked somewhere when it is checked nowhere. This is the
"check present but uncalled from production" shape.

G-C asked whether the statement overstates. It does, in both halves. Required:
either verify the binding at this gate (recomputing the two derived fixtures), or
restate the bound with a reason that survives a one-call check and say plainly
that no production path attests the binding today.

## F4 (blocking) — `Lift`'s default arm is fail-open for a registered code, and its doc says otherwise

`internal/provider/lift.go`. The doc block claims: "An `Error` carrying any other
code is failed closed through the registry: axerror refuses the unregistered code
and Lift returns that refusal, **never a wire object**."

Driven against `Lift` directly:

| `Error.code` | `Lift` result |
|---|---|
| `invalid_argument`, `internal_error`, `unauthorized`, `conflict`, `timeout` | refused — `unregistered structured error code` |
| `not_found` | **wire object**: `{"code":"not_found","message":"provider failure carries no liftable code","exit_code":4,…}` |
| `incompatible_protocol` | **wire object**, `exit_code` 6 |
| `provider_protocol_error` | **wire object**, `exit_code` 13 |

The default arm is not a fail-closed sentinel; it is a fail-closed sentinel only
for codes that happen to be outside the axerror registry. For a registered code
it ships a real Structured Error, with a real exit status, whose message says the
code could not be lifted. `TestLiftRefusesUnregisteredCode` picks `"bogus_code"`
— the half that trivially holds — so the negative proves the reachable-and-refused
path and nothing about the reachable-and-admitted one.

Latent today (no production constructor mints those codes), but it composes with
F5: the census that is supposed to stop a new code from reaching the default arm
has holes.

## F5 (blocking) — the lift code census does not fail closed on the two shapes the DoD names

`internal/provider/lift_test.go`, `derivedProviderCodes`. It reads exactly one
file (`provider.go`) and exactly one declaration form (`general.Tok ==
token.CONST`).

Control plants, `TestLiftCoversTheClosedCodeSet` in isolation:

| Plant | Result |
|---|---|
| `codeExperimental = "experimental"` added to the `const` block in `provider.go` | **RED** — `production code "experimental" has no lift vector` |
| `var codeExperimental = "experimental"` in `provider.go` | **GREEN — census blind** |
| `const codeExperimental = "experimental"` in `os.go` (same package) | **GREEN — census blind** |

The DoD row names var-binding explicitly as a required control plant, and the
outcome's M9 row reports the census as failing closed on the strength of the
const-in-`provider.go` shape alone. Running the *full* provider suite, both blind
plants are caught — but by the pre-existing stray-literal gate, reporting
`provider refusals built outside an instrumented constructor: provider.go:512` /
`os.go:88`. That gate fires on the accompanying `Error{…}` literal, not on the
unwitnessed code, and would not fire on a code introduced through an already
instrumented constructor. Required: derive over the package rather than one file,
accept `token.VAR`, and control-plant both shapes.

## F6 (observation, not blocking) — §7.A ownership is an exported entry with no caller

`grep` over every non-test file: `AdmitProviderDescriptor` appears in its own
declaration, its own doc comment, and one prose mention in `provhost/doc.go`.
Zero callers. The round-1 finding was that `CheckProviderDescriptor` had no
possible caller; that is now one level up — `CheckProviderDescriptor` is called
by `AdmitProviderDescriptor`, which is called by nothing but its tests. The
repository has no `cmd/`, and the repo-wide convention (the `entry` column of the
refusal inventory) treats an exported package function as a production entry, so
this is a systemic condition rather than a regression introduced here, and both
`provhost/doc.go` and the registry `gap` disclose the dual-stack deferral
honestly. Recorded so the next round does not have to rediscover it; no rework
required for this item alone.

## F7 (minor) — two counts in the outcome document do not match what it describes

- "Candidate tree … holds all 12 changed paths" and "Production changes (12
  paths)"; the Change Request carries 13 (`LOGBOOK.md` is the thirteenth).
- Headline "AC row coverage: 11 of 11 driven", while row 9 (F3) names no driving
  test and says "prose by nature". The honest ratio is 10 of 11 driven plus one
  stated bound — and per F3 above that stated bound does not currently hold.

## What the next round has to produce

1. Close the numeric class in `descriptorGeometry` (F1) and in the `semverMajor`
   consumption (F2), each with a refusal test using a literal from the far half
   of the class, not the adjacent edge.
2. Fix the `identity.go` bound (F3): verify the binding at the gate, or restate
   the paragraph with a checkable reason and say that no production path attests
   the binding.
3. Make `Lift`'s default arm actually fail closed for a registered code, or
   change the doc claim to match the behavior, and add the registered-code
   negative (F4).
4. Widen `derivedProviderCodes` to the package and to `token.VAR`, with both
   control plants reported (F5).
5. Correct the two counts in the outcome (F7) and report the AC ratio as measured.
