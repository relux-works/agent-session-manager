# TASK-260830-wbpf1v review verdict — CR-TASK-260830-wbpf1v-3 (rev 3)

**Verdict: `changes_requested` → `to-dev`.**

Two blocking findings (F1, F2) and one non-blocking-but-fix-with (F3).
G-B passes, G-C passes on re-measurement, G-D fails only on the recorded
document (the repository tree itself is correct).

Everything below was driven in the worktree at the candidate tree
`08c8ceb6c31530930f554d89e04b41046397a7e0`. Every production source was
checksummed before probing and restored after; the post-probe
detached-index `write-tree` is again `08c8ceb6`.

---

## F1 (BLOCKING) — the content-equality census is a hard-coded token count, not a derived class census

**repeat-of: rev2 / B1** (same class: the equality census does not measure
the class it names). Round 2 found one of two sites uncensused. Round 3
fixed both *sites* and added an *instrument* that cannot see a third.

`internal/sessrepo/sessrepo_test.go:832` `TestContentEqualityComparisonsAreCensused`:

```go
for _, name := range []string{"sessrepo.go", "chain.go", "store.go"} {
    counts[name] = strings.Count(string(raw), "bytes.Equal(")
}
if counts["sessrepo.go"] != 1 || counts["chain.go"] != 1 || counts["store.go"] != 0 {
```

The denominator is not "every content-equality comparison in the package".
It is "occurrences of the literal substring `bytes.Equal(` in three
hard-coded file names". Three control plants, each applied to production
and then reverted:

| Plant | Where | Census | Full unfiltered package run |
|---|---|---|---|
| P1 (control) `bytes.Equal(left, right)` | appended to `chain.go` — a listed file | **FAIL** (`sites = map[chain.go:2 …]`) | FAIL |
| P2 `bytes.Equal(left, right)` | new production file `internal/sessrepo/plantextra.go`, same package | **PASS** | **PASS** |
| P3 `string(left) == string(right)` | appended to `store.go` — a listed file | **PASS** | **PASS** |

P1 proves the instrument is alive. P2 and P3 are the blind spots the
brief asked me to plant: a third `bytes.Equal` **does not** redden when it
lives in a new file of the same package, and a content-equality
comparison spelled without the searched-for token does not redden at all.
That is a two-row table, and it is additionally a source-text gate with
no token-preserving attack against it.

Against the DoD row *"Every census derives its denominator from production
and fails closed on an unregistered site, an orphan row and an
unclassifiable site, each control-planted including an import alias and a
var binding"* this census derives nothing, distinguishes no orphan from no
unclassifiable site, and ships zero control plants.

The owner already exists in this very package and was bypassed for this
one census: `census_test.go:132` derives the refusal inventory over
**every** production file through `invcore.ScanProduction(".")`, then
diffs derived-vs-exercised through `invcore.DiffSets` with unregistered
and orphan rows separated and five control plants. The equality census
should be built the same way — walk the package AST, classify every
content-equality comparison (`bytes.Equal`, `string(x) == string(y)`,
`reflect.DeepEqual` over byte content), fail closed on a site with no
registered vector, and control-plant it in both directions.

**What is NOT wrong:** production is correct today. Both live sites are
witnessed and both narrowing mutants die behaviourally — I re-drove them
myself, see F-OK-2. The defect is the instrument, not the two sites.

---

## F2 (BLOCKING) — three published measured numbers are stale, one of them in a shipped repository file

The brief flagged the tree OID; re-deriving the rest found two more.

**F2a — candidate tree OID.** `TASK-260830-wbpf1v_outcome-rev3.md:10`
names `93d4a106fd15cdb335539c2612d3ed84d70f919a`. The CR record and my
independent detached-index write-tree (`read-tree HEAD` + `git add -A` +
`write-tree` on a scratch index) both give
`08c8ceb6c31530930f554d89e04b41046397a7e0`. Fourth instance of this class
in this programme.

**F2b — the outcome does not enumerate the CR's changed paths.** The same
paragraph describes the candidate as *"the 8 listed paths … `ls-tree`
shows `README.md` plus the 7 `sessrepo` files"*. The Change Request
carries **11** paths. `LOGBOOK.md`, `internal/provhost/identity.go` and
`internal/provhost/identity_test.go` are never enumerated anywhere in the
outcome. DoD row: *"the outcome enumerates exactly the CR changed paths"*.

**F2c — the refusal-site denominator is 35, published as 36, including in
`README.md`.** Measured through the package's own derivation
(`invcore.ScanProduction(".")` + `deriveSessrepoRefusalSites`, driven from
a temporary probe test in-package, then removed):

```
DERIVED_SITES=35  per-file=map[chain.go:25 sessrepo.go:5 store.go:5]
sentinels=14 sentinelNames=14
```

`chain.go` 25, `sessrepo.go` 5, `store.go` **5**. The rev2 patch carries
`store.go` 6 — round 3 converted the repository-wide `ListSessions`
closure into parked data and removed that refusal, so the count went
*down*, not up. The rev2 number was carried forward unchanged into:

- `TASK-260830-wbpf1v_outcome-rev3.md:93` — *"Derived `refuse(` sites: **36** (sessrepo.go 5, chain.go 25, store.go 6)"*
- `TASK-260830-wbpf1v_outcome-rev3.md:100` — *"Exercised beneath a boundary entry: **36 of 36**"*
- `TASK-260830-wbpf1v_outcome-rev3.md:66` (AC row 8) — *"36-site census"*
- **`README.md`** — *"The unfiltered package run derives the 36-site refusal inventory"*

The README one ships in the repository. DoD row: *"README/doctor/capability
evidence and specification traceability are updated without unsupported
claims"*. The gate itself is fine — it derives, so it cannot go stale; only
the prose about it did.

---

## F3 (non-blocking, fix in the same round) — false justification on `sessionDir`, and the real fail-closed property is unwitnessed

`internal/sessrepo/chain.go:245-247`:

```go
// sessionDir maps a validated session ID to its directory. The ID passed
// the scalar UUIDv7 grammar at decode time, so no traversaliedy member can
// reach the join; path policy stays with the scalar and secprim owners.
```

That universal is false. `GetRecord`, `GetEvent` and `ListEvents` take a
caller-supplied `sessionID string` and hand it straight to
`loadSessionLocked` → `sessionDir` → `filepath.Join` with **no** grammar
check. Driven through the production entries:

```
sessionDir("../outside") = /…/001/outside     (sessions root = /…/001/sessions)
ESCAPED: the join left the sessions root
GetRecord("../outside")  err = session event chain is corrupt: decode chain index …
ListEvents("../outside") err = session event chain is corrupt: decode chain index …
GetEvent("../outside")   err = session event chain is corrupt: decode chain index …
```

**Not exploitable**, and I checked why: the escaped path still has to
satisfy `record.sessionID != sessionID` at `chain.go:277`, and a canonical
UUIDv7 string can never equal a traversal string, so every escape refuses.
But that is a *different* mechanism from the one the comment claims, and
no test drives an unvalidated session ID at those three entries — the
fail-closed property is asserted, not witnessed. Same class as rev1 F1
(false universal in a production comment). Also: `traversaliedy` is a typo.

Fix: state the property that actually holds (an escaped path cannot bind
to a stored record, so the load refuses) and add one negative test driving
a traversal string through `GetRecord`/`GetEvent`/`ListEvents`.

---

## G-B — PASS. The parked channel is a contract, not a resemblance

Driven, not read:

```
--- PASS: TestListSessionsReportsParkedSessionPerSession
--- PASS: TestListSessionsKeepsHealthySessionsBesideParked
--- PASS: TestResolveSkipsParkedSessions
```

- **§14.4 per-session** — confirmed. `listSessionsLocked` (`store.go:157`)
  builds one entry per session *directory*; a load failure becomes
  `parkedSummary(...)` and `continue`, never a repository-level flag.
  `TestListSessionsKeepsHealthySessionsBesideParked` parks one session at
  `CreateStepRecord` beside a healthy sibling, asserts two entries, asserts
  the healthy one is first-class and resolvable, asserts the differing
  retry refuses with `ErrSessionExists` and the byte-identical original
  heals. That is exactly the case rev2 measured as broken repository-wide.
- **My own mutants confirm both halves are load-bearing:** dropping the
  parked entry from the listing → KILLED behaviourally by
  `TestCreateStepCrashAfterSessionDirResumesOnRetry` +
  `...ReportsParkedSessionPerSession` + `...KeepsHealthySessionsBesideParked`;
  `!summary.Parked` → `true` in `Resolve` → KILLED by
  `TestResolveSkipsParkedSessions` + `...ReportsParkedSessionPerSession`.
- **Operator remedy** — present in `doc.go:71-83` and in `README.md`,
  names the exact path (`<data-root>/sessions/<session-id>`), names the
  blast radius (*"discards whatever that session had parked; it never
  touches siblings"*), and names the irrecoverable case as irrecoverable
  (*"the original record bytes are lost … no `CreateSession` retry heals
  it"*), with the no-delete-entry design stated as design.

**Two observations, neither blocking, both for the next three leaves:**

1. **§13.13 consumability is rendering-grade, not reducer-grade.**
   `Parked bool` is typed and machine-consumable — that is what the §5.7
   reducer (`1r9wrr`) actually needs. `BlockingReason` is
   `"recoverable_parked_state: " + loadErr.Error()` (`store.go:190`) — a
   literal class token concatenated with free-form Go error text, so a
   reducer that wants the *class* must split on `": "`. `RetryHint` is one
   of three **unexported** constants (`retryHintForParkedRecord`,
   `retryHintForBareDirectory`, `retryHintForTornStore`), so no downstream
   package can discriminate the three retry classes by symbol at all — it
   can only pass the string through. A typed owner for the token already
   exists: `secconftest.OutcomeRecoverableParked`. The outcome does state
   the downstream contract as pass-through rendering, so this is a stated
   bound rather than a hidden gap, and I am not blocking on it — but
   `1r9wrr`/`21gygk`/`14yo67` will want an exported blocking-class and
   retry-class type before they consume this.
2. **`Resolve` reports a parked-but-existing name as `ErrNameNotFound`.**
   A name that exists but is parked is not absent. This is decided and
   documented (`store.go:107-112`, `doc.go:39-42`) and downstream can still
   tell the two apart via `ListSessions`, so I accept it as a stated bound
   — but flag it for `21gygk`, which surfaces the name-resolution UX.
3. Minor, in `parkedSummary` (`store.go:192-196`): any `os.ReadFile`
   failure on `record.json` — not only `IsNotExist` — produces the
   bare-directory retry hint. An unreadable-but-present record is then
   advertised as *"retry CreateSession with the session record for this
   session ID"*, which `resumeCreateLocked` will fail with a read error.
   Absence and failure-to-read conflated in the advice path. Advisory text
   only; fix while in there.

---

## G-C — PASS on re-measurement

**Denominator re-derived: 35, not 30 and not 36.** See F2c. The battery's
`30 applied / 30 killed` is a subset of the 35 refusal sites plus mechanism
targets, with the remainder carried as explicitly stated subsumption
bounds — which is legitimate, but the *census* denominator printed beside
it is wrong.

**Classes are properly separated** in `TASK-260830-wbpf1v_mutation-battery-rev3.log`:
16 narrowing / 10 arm-deletion / 2 census-only / 2 audit-only, with the
census-only and audit-only rows showing behavioural-green + full-run-red
and quoting the actual gate message, and `X1 NOT_APPLIED` / `X2
COMPILE_FAIL` as distinct harness controls. The C1/C2 rows are
token-preserving (funnel bypassed, sentinel token kept) and are executed
through the behavioural suite as well as the static gate — that satisfies
the DoD's token-preserving requirement for the refusal census.

**Six mutants re-driven independently by me** (own harness, exact-count
application, checksum restore, behavioural run under `-run '.*'` separated
from the full unfiltered run):

| Mutant | Applied to | Verdict | Killing test |
|---|---|---|---|
| N13 `!bytes.Equal(existing, data)` → `len != len` | `chain.go` install | KILLED (behavioural) | `TestAppendEventRefusesSameLengthDisagreeingBytes` |
| N10 `!bytes.Equal(existing, recordJSON)` → length-only, `bytes` token preserved | `sessrepo.go` resume | KILLED (behavioural) | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| N16 digest arm `&& position > 0` | `chain.go` load | KILLED (behavioural) | `TestLoadRefusesSubstitutedFirstEventBlob` |
| N14 drop parked entry from listing | `store.go` | KILLED (behavioural) | 3 tests |
| N15 `!summary.Parked` → `true` | `store.go` `Resolve` | KILLED (behavioural) | `TestResolveSkipsParkedSessions` + 1 |
| — first N10 attempt | `sessrepo.go` | COMPILE_FAIL (unused import), re-run as above | — |

Note on N13: the equality census *also* reddens under it (the token
disappears), so I confirmed the behavioural kill separately — the
`-run '.*'` run fails on `TestAppendEventRefusesSameLengthDisagreeingBytes`
itself. The same-length vector self-checks that the plant differs in
content at equal length (`sessrepo_test.go:795-800`), so the kill is real.

**N2's subsumption claim — verified, not accepted.** The claim is that
the sessrepo member arms are behaviourally subsumed by the canonical
owner. I drove three narrowings and read the mechanism:

| Mutant | Verdict |
|---|---|
| `CheckUint53Bounds(lease_epoch, 1, …)` → `0` | SURVIVED |
| `CheckUint53Bounds(lease_sequence, 1, …)` → `0` | SURVIVED |
| delete the non-empty `predecessors` arm | SURVIVED |

Surviving is the *correct* outcome here, and the reason holds
structurally: `canonicaljson.VerifyObjectIdentity` →
`prepareObjectIdentity` (`canonical.go:281`) → `validateImmutableObjectShape`,
which enforces `requirePositiveUint("lease_epoch")` (`core_records.go:481`),
`requirePositiveUint("lease_sequence")` (`:487`) and the `predecessors`
minimum on **every** input, digest-valid or not. So a well-formed,
correctly-digested event carrying `lease_epoch: 0` is refused by the owner
even with the sessrepo arm removed — the bound is an *attribution* bound,
not a hole. One wording correction for the outcome: it says canonicaljson
refuses *"one arm earlier"*; in `decodeSessionEvent` the member arms run
**before** `Verify` (`chain.go:96` then `:100`), so the owner refuses one
arm *later*. The substance is right, the ordering word is wrong.

**`-race` — I re-ran it myself across every package.** The outcome's four
groups sum to 24 and `go list ./...` gives 24 packages, all with tests, so
the split is complete by arithmetic. I re-ran it in three groups of my own
rather than trusting that:

| Group | Result |
|---|---|
| `sessrepo` + `provhost` | ok (3.3s / 29.3s) |
| the other 20 packages | ok, 20/20, exit 0 |
| `canonicaljson` + `localstore` | ok (120.6s / 37.5s) |

24 of 24 packages green under `-race`. Nothing is left unmeasured by the
split.

## Gates I re-ran at the candidate tree

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | exit 0, 24/24 ok |
| `go test ./internal/sessrepo -count=1` (unfiltered — census gate active) | ok |
| `go test -race` all 24 packages (3 groups) | exit 0 |

Not re-run by me, accepted from the attached rev3 evidence and stated as
such: `GOOS=windows` vet/build, `tracecheck`, coverage 87.2%.

---

## G-D — repository tree PASS, recorded document FAIL

| Check | Result |
|---|---|
| Candidate tree OID equals the record's | **PASS** — independent detached-index `write-tree` = `08c8ceb6c31530930f554d89e04b41046397a7e0` |
| Tree unchanged after my probes | **PASS** — re-computed `08c8ceb6…` after restore, all 7 source checksums OK |
| No untracked ungitignored scratch file | **PASS** — the only untracked entries are the 7 new `internal/sessrepo/*.go` files, all of them CR paths |
| Diff resolves and matches the 11 declared paths | **PASS** |
| Outcome names that tree | **FAIL** — F2a |
| Outcome enumerates exactly the CR changed paths | **FAIL** — F2b |

---

## What to do

1. **F1** — rebuild `TestContentEqualityComparisonsAreCensused` as a
   derived census over the package (reuse `invcore.ScanProduction(".")` /
   `invcore.DiffSets` exactly as `census_test.go` already does), classify
   every content-equality comparison rather than counting one token in
   three named files, fail closed on an unregistered site / an orphan row /
   an unclassifiable site, and ship control plants in both directions —
   including a new-file plant and a plant spelled without `bytes.Equal`.
2. **F2** — recompute and correct the tree OID, enumerate all 11 CR paths
   in the outcome, and fix the refusal-site count to 35 in the outcome
   (three places) **and in `README.md`**. Re-derive rather than re-typing:
   the number moved because round 3 removed a refusal.
3. **F3** — correct the `sessionDir` comment to the property that actually
   holds, fix the `traversaliedy` typo, and add one negative test driving
   an unvalidated session ID through `GetRecord`/`GetEvent`/`ListEvents`.
4. Optional while in there: the `parkedSummary` absence-vs-read-failure
   conflation, and the outcome's *"one arm earlier"* → *"one arm later"*.

**repeat-of: rev2 / B1** (F1). F2 and F3 are new; F2's tree-OID half is the
fourth instance of the stale-number class in this programme, which is why
the recommendation on it is "re-derive", not "re-type".
