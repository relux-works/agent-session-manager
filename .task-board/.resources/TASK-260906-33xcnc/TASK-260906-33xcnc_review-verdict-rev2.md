# TASK-260906-33xcnc rev2 — review verdict: ACCEPTED

**repeat-of:** none (no finding repeats a prior revision; F1/F2/F3 from rev1 are closed).

Reviewer run: `RUN-260907-911b99`. Candidate `CR-TASK-260906-33xcnc-2` revision 2,
base `cd8591d`, candidate tree `ca0bf6b68b174192432c4c46296cc4610609cdb9`.
Every number below was produced by a command I ran in this worktree, not read
from the producer's log.

## Provenance (G-E)

| Check | Result |
| --- | --- |
| Detached-index `write-tree` over `HEAD` + working tree | `ca0bf6b68b174192432c4c46296cc4610609cdb9` |
| Record's candidate tree OID | `ca0bf6b68b174192432c4c46296cc4610609cdb9` — **equal** |
| `git diff --name-only cd8591d <tree>` | 25 paths, exactly the CR list, both directions |
| Untracked non-ignored files | the 6 new test files only; no scratch in the worktree root or the candidate tree |
| Tree OID re-verified after every plant/restore cycle | unchanged at `ca0bf6b6` (5 checks) |

The rev1 outcome's stale `8dbf62d7`/21-path mismatch is gone: the round-2 document
names the record's tree and the 25 paths, and the enumeration matches `diff-tree`
in both directions.

**Round-2 production delta is byte-identical to rev1.** I extracted the 9
production-file hunks from `TASK-260906-33xcnc_change-request_rev1.patch` and
diffed the `+`/`-` lines against `git diff HEAD -- <same 9 files>`: 24208 bytes on
both sides, no differing line. The three paths new in rev2
(`environ/frame_agreement_test.go`, `environ/tuple_agreement_test.go`,
`provhost/protocol_test.go`) are all test files. The "test-only round" claim holds.

## G-A (blocking) — is the delegation gate load-bearing, or merely stricter?

Five decoys planted directly into `internal/sessadapter/decode.go` (and, for the
alias shape, a real `internal/environdecoy` package aliased to the name `environ`).
Every plant compiled; every plant was restored and the tree OID re-verified.

| # | Plant | Structural (`TestCheckHelpersDelegateToEnviron`) | Behavioural (`TestTupleAgreementAcrossFacades`) | Other |
| --- | --- | --- | --- | --- |
| A1 | owner call **discarded**, local answer that **agrees** on the corpus (`_, _ = environ.CheckDigest(raw)` + verbatim owner logic) | **CAUGHT** | green (it agrees — by construction) | — |
| A2 | owner call behind a **runtime-false condition** (`if len(raw) < 0`), drifted local answer | **CAUGHT** | **CAUGHT** | — |
| A3 | **import alias**: `environ "…/internal/environdecoy"`, a shim forwarding 9 of 10 members and drifting `CheckDigest` | passed | **CAUGHT** | **CAUGHT** by `TestCensusScopeHasNoAliasedSharedRules` |
| A4 | **var binding**: `var checkDigestDelegate = environ.CheckDigest`; helper returns `checkDigestDelegate(raw)` | **CAUGHT** | green | — |
| A5 | pure single delegating return with a **mutated argument**: `return environ.CheckDigest(loweredDigestRaw(raw))` | passed | **CAUGHT** | — |

**Answer to G-A: the structural half is load-bearing, not cosmetic.** A1 is the
exact shape the brief asked about — call the owner, discard the result, return a
local answer that agrees — and it is caught by the structural gate **alone**; the
behavioural corpus cannot see it. A4 is the same. `checkHelperDelegation` now
requires the body to be exactly `return environ.Twin(args)` with a twin-name
match, so mention-without-use is gone.

Two structural blind spots, both covered by an independent direction:

- **A5 (argument mutation)** — the gate does not constrain the call's arguments,
  so a drift moved into the argument expression passes it. Caught only by the
  behavioural corpus, i.e. it depends on the `uppercase fingerprint refused`
  vector existing. Bound B1 below.
- **A3 (alias to a decoy package)** — the gate is identifier-keyed on the literal
  name `environ` and never resolves the import path. Caught twice elsewhere (alias
  audit + behavioural), so the composed defence holds.

Both directions therefore carry real weight, and neither is redundant.

## G-B (blocking) — the duplicate class, every key and the retained copy

There are exactly two duplicate-member gates in production:
`environ/decode.go:84` and `provhost/protocol.go:295` (`grep` over `internal/`,
non-test).

**Owner — all 30 derived keys, not a sample.** The row set is **derived over the
member set**, not hand-extended: `duplicateMemberNames` unions
`environ.tupleRequired`, `environ.observationRequired`, `provhost.responseMembers`
and `dirnode.scanRequestRequired` from production AST, adds the generic `v`, and
`t.Fatal`s on a zero-length derivation. That yields 30 keys. I planted
`duplicate && key != "K"` for **each of the 30** and re-ran:

- **30 of 30 KILLED.**
- every mutant failed **exactly one** subtest (`totalfailrows=1` on all 30), and
  spot-checking `body`, `ok`, `store_schema_fingerprint` showed the failing row is
  the mutant's own key. The suite measures the class, not a witness.

**Retained copy — all 10 keys of the provhost sweep, plus 30 by agreement.** I
planted the same narrowing at `provhost/protocol.go:295`:

- the 6 derived `responseMembers` (`protocol`, `protocol_version`, `request_id`,
  `ok`, `body`, `error`) and the 4 foreign keys (`v`, `capabilities`,
  `provider_id`, `cursor`): **10 of 10 KILLED**, each on its own row.
- **the rev1 exploit is refused**: `P_dupnarrow_body` fails
  `TestDecodeResponseRefusesDuplicateOfEveryMember/body`, so at baseline
  `DecodeResponse` refuses a frame with two `body` members instead of returning the
  second.
- coverage on the retained copy is wider than its own test: narrowing it at
  `schema_version` and `installation_ids` — keys **absent** from the provhost sweep
  — is caught by `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/<key>`
  through `provhost.DecodeManifest`. The retained gate is pinned per key over the
  union of both sets.

`TestDecodeResponseRefusesDuplicateOfEveryMember` derives its members from
`responseMembers` at runtime and `t.Fatal`s when the literal table and the member
set disagree, so a seventh member fails closed rather than passing silently. The
only hand-written parts are the value literals and the 4 foreign keys.

## G-C (blocking) — the denominator

Re-derived by me in source, not carried forward:

| Component | Count | How I counted |
| --- | ---: | --- |
| direct `refuse(` call sites | **31** | `observation.go` 21 + `tuple.go` 10 |
| bool-false exits | **59** | `decode.go` 40 + `observation.go` 18 + `tuple.go` 1 |
| `&Fault{}` frame exits in `DecodeStrictObject` | **11** | `decode.go` lines 60, 63, 68, 72, 78, 82, 85, 89, 94, 96, 99 |
| **stated denominator** | **101** | 31 + 59 + 11 |
| bool-`return <expr>` arms | **4** | `tuple.go:18`, `tuple.go:28`, `decode.go:239`, `decode.go:344` — exactly the four the outcome names |
| every exit kind | **105** | 101 + 4 |

The 11 previously-excluded `&Fault{}` exits **are** included. The `100 → 101`
step is real and verified: `git diff HEAD -- internal/environ/decode.go` shows the
new `if literal == "" { return 0, false }` guard in `parseUint53Literal`, which is
the +1 bool-false exit (89+11=100 pre-guard, 90+11=101 now). The "two stories
stale" wording is withdrawn in the round-2 document, and coverage is reported
against 101 rather than inherited.

I attacked the new guard both ways: deleting it and narrowing it to
`if literal == "\x00"` both fail `TestParseUint53LiteralRefusesNonDigits`. The
new provhost trailing check is killed by a narrowing
(`err != io.EOF && err == nil`) via `TestRawUint53RefusesTrailingData`; deleting
it COMPILE_FAILs on the orphaned `io` import, exactly as the S5 row discloses.

## G-D — the gates, run by me

| Gate | Result |
| --- | --- |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `GOOS=windows go build ./...` | 0 |
| `gofmt -l internal/` / `gofmt -l .` | empty |
| `go test ./... -count=1` | **23/23 ok** (59.9s) |
| `go test ./... -race -count=1` | **23/23 ok** (2m09s) — **full repo, no package scoping** |
| skipped tests | 5, all pre-existing and outside the 25 changed paths (`TestDumpSweepSites` gated on `AX_SWEEP_DUMP`, 4 `TestCanonicalRoundTripDoesNotLaunderAMalformedMember` subtests) |

The rev1 7-package race bound is legitimately withdrawn: I ran the full repo
under `-race` myself.

## Fail-closed behaviour I attacked directly

| Plant | Result |
| --- | --- |
| unexercised `refuse()` site added to `environ/tuple.go` | `TestEveryRefusalSiteIsExercised` **fails closed**: `derived refusal sites without an exercised negative path: tuple.go:29`, and the observed-rule set comparison fires too |
| `var semverRefuse = refuse` rebinding | **fails**: `tuple.go:27:20: constructor "refuse" referenced outside direct-call position` |
| fresh-name rune measure spelled `utf8.RuneCountInString` in `sessadapter/manifest.go` | **fails**: `unregistered site with no declaring row: string-measure|sessadapter|manifest.go|measureWidth` |
| over-strict divergence in `scalar.hasLoneJSONSurrogate` (paired surrogates refused) | **caught in the admit direction**: `TestScalarSurrogateAgreementAcrossUnits/paired_surrogates_admit` and `/escaped_backslash_plus_real_pair_admits`, plus `scalar.TestJSONDecodersCannotBypassScalarValidation` |
| narrowing on `canonicaljson.validateSurrogateEscapes` (`0xdc00` → `0xdc01`, gate stays) | **caught**: `TestFrameAgreementAcrossFacades/bare_low_escape`, `/escaped_backslash_plus_real_lone_low`, plus 3 canonicaljson suites |

Both directions of the scalar divergence are therefore pinned, not inferred: the
over-permissive half by C1/C2/C3 in the battery, the over-strict half by my plant.
`TestEveryRefusalSiteIsExercised` observes **39** expanded rules, matching the
outcome's claim exactly.

## AC coverage: 6 of 6 rows driven, each reddened by me

| AC row | Production entry | Named test I reddened |
| --- | --- | --- |
| provhost frame decoder + surrogate gate justified | `provhost.DecodeResponse`, `provhost.DecodeManifest` | `TestDecodeResponseRefusesDuplicateOfEveryMember` (10/10), `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember` (2 cross-package keys) |
| canonicaljson frame gate justified | `canonicaljson.Canonicalize` / `CalculateObjectIdentity` | `TestFrameAgreementAcrossFacades/bare_low_escape` |
| scalar third-spelling gate justified | `scalar.DecodeClosedEnumJSON` | `TestScalarSurrogateAgreementAcrossUnits` (both directions) |
| sessadapter/dirnode per-helper copies converged | `sessadapter.DecodeTuple` | `TestCheckHelpersDelegateToEnviron/sessadapter` (A1, A2, A4), `TestTupleAgreementAcrossFacades/uppercase_fingerprint_refused` (A2, A3, A5) |
| sibling rawUint53 trailing texture hardened | `provhost.rawUint53`, sessadapter/dirnode siblings | `TestRawUint53RefusesTrailingData` present and passing in all 3 packages; killed by my `P1_trailnarrow` |
| F2 residue exits attacked or re-stated | `environ` production entries with `invcore` runtime site recording | `TestEveryRefusalSiteIsExercised` (fails closed under my unexercised-site plant), denominator re-derived at 101 |

## Bounds recorded (non-blocking)

- **B1 — the structural delegation gate does not constrain the delegated call's
  arguments.** `return environ.CheckDigest(loweredDigestRaw(raw))` is a pure
  single delegating return with a matching twin name and passes
  `checkHelperDelegation`. It is caught only by the behavioural corpus, so this
  arm's strength equals the corpus's coverage of the drifted rule. Same shape for
  an import aliased to a decoy package (caught by the alias audit and the corpus,
  not by the delegation gate).
- **B2 — the shape census's rune-measure detector is keyed to
  `utf8.RuneCountInString` / `RuneCount` only.** I planted a fresh-name
  `for range value { count++ }` measure in `sessadapter/manifest.go` (in census
  scope) and the entire repository suite stayed green; the same function spelled
  with `utf8.RuneCountInString` fails as unregistered. **No live site of the
  missed spelling exists** — I checked every range-over-string in all 7 census
  packages, and the two live ones (`dirnode.checkURI`, `scalar.invalidWindowsSegment`)
  are character-class walks, with `checkURI` correctly measuring through
  `stringLength`. Graded as a bound because no live site exists, but the ledger
  comment at `shape_census_test.go:122` ("A revived local count fails here as
  unregistered") states the closure more universally than what is measured.
- **B3 — `TestEveryRefusalSiteIsExercised`'s derived-vs-exercised comparison is a
  `TestMain` audit and only fires on a full package run.** Under a `-run` mask my
  unexercised-site plant passed silently; it failed under `go test ./internal/environ/`.
  The repository gate is `go test ./...`, so this holds in practice, but a future
  masked CI lane would not see it.

## Battery

The producer's table reports **70 killed / 71 applied** on the 101-exit
denominator, with narrowing (49), arm-deletion (17), census-only (3) and
audit-only (1) separate, `NOT_APPLIED` (`H1`) and `COMPILE_FAIL` (`H2`) as distinct
rows, one predicted survivor (`R_digestnonstr`, now property-pinned by
`TestDigestBridgeRefusesEmptyDownstream`), and an honest in-session survivor
disclosure (`P_dupnarrow_provider_id` found green, gap fixed, re-run KILLED). My
own sweep is strictly wider than the table's on the duplicate class: 30 owner keys
against their 8 `E_dupnarrow_*` rows, and 10 retained keys against their 6
`P_dupnarrow_*` rows. Nothing in my sweep contradicts the table.

## Verdict

**Accepted.** All five briefed gates are answered with executed evidence, the two
rev1 blocking classes (decoy-satisfiable delegation gate, per-witness duplicate
pin) are closed and independently re-measured by me at wider scope, the
denominator is restated on the corrected 101-exit basis and re-derived in source,
the full-repo `-race` bound is withdrawn on a run I performed, and the candidate
tree matches the record over exactly the 25 CR paths with no scratch file. The
three bounds above are recorded for the story's residue, not held against this
revision: B2 and B3 have no live production site or lane behind them, and B1's
blind spot is covered by an independent direction that I reddened.
