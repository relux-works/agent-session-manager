# TASK-260830-wbpf1v review verdict — CR-TASK-260830-wbpf1v-2 revision 2

**Verdict: changes_requested → `to-dev`.**

Two blocking findings (B1, B2), two non-blocking (N1, N2). Everything the
spawn brief named under G-B, G-C and G-D was re-driven by this run and
holds; the two blocking findings are new measurements on the same axes.

`repeat-of:` B1 → rev1 F2 (same class, sibling site). B2 → rev1 F3
(un-retried case unchanged; the rule is asserted, not decided).

## What this run actually executed

Independent battery: 28 mutants applied to the rev2 tree with exact-count
replacement and checksum restore, each run against the full behavioral
suite (`go test ./internal/sessrepo -count=1 -run 'Test'`, which skips the
census by design so a kill is behavioral). 13 KILLED, 16 SURVIVED, 1
COMPILE_FAIL control. Of the 16 survivors, 9 are subsumed-by-an-owner and
measured as such (below), 5 are prefix narrowings on digest equality whose
admitted member is not realistically mintable, and **2 are genuine
unwitnessed classes** (B1, N1).

Tree restored to `468d27367eb88d5a9da4d3ac540ff9c5a0831700` after every
probe; verified by detached-index `write-tree` at the end.

## B1 (blocking) — `installEventBlob` byte-equality ships no narrowing mutant, and a length-only narrowing survives the whole suite

`internal/sessrepo/chain.go:426`

```go
existing, readErr := os.ReadFile(path)
if readErr != nil || !bytes.Equal(existing, data) {
    return errDigestDisagreement
}
```

This is a gate: its own comment says "identical bytes are reused,
disagreeing bytes are reported for refusal", and it funnels into
`refuse(ErrChainCorrupt, "digest path holds disagreeing bytes for event")`
— one of the 36 census sites.

Mutant (narrowing, gate stays present):

```
chain.go:426   !bytes.Equal(existing, data)  →  len(existing) != len(data)
result         SURVIVED the full behavioral suite
```

Failure scenario: a blob of exactly the candidate event's byte length but
different content already sits at the candidate's digest path. Under the
mutant `AppendEvent` treats it as "already installed", writes the chain
entry, and returns success — after which every read of that session fails
`ErrChainCorrupt`. A false success on a store the same call just bricked.

Why the suite does not see it: both witnesses plant `{"forged":true}` (15
bytes) — `sessrepo_test.go:702` and `:1378`. There is no same-length
vector anywhere for this comparison, and the battery has no mutant on it
at all: N9 targets the `O_EXCL` flag on `chain.go:435`, a different line,
and its "narrowing control" only asserts that identical-byte installs
still succeed.

This is the round-1 F2/M6 class, closed at exactly one of the two
`bytes.Equal` comparisons in the package:

| site | witness | narrowing mutant | verdict |
|---|---|---|---|
| `sessrepo.go:234` (`resumeCreateLocked`) | `TestCreateSessionRefusesResumeWithDifferingBytes`, deliberate same-length plant asserted at `:1208` | N10 | KILLED (reviewer R4b confirms) |
| `chain.go:426` (`installEventBlob`) | `{"forged":true}`, 15 bytes, both sites | **none** | **SURVIVED** |

The rev2 test at `:1209` states the reasoning verbatim — "the retry carries
the same byte length as the parked record, so a narrowed equality that
compares lengths only would admit it" — so the class was understood and
applied to one member of it. Reviewer probe P2 confirms production is
correct here (the real code refuses the same-length plant); what is
missing is the witness and the mutant, not the behavior.

Fix shape: a same-length disagreeing plant driven through `AppendEvent`,
plus a narrowing mutant row for `chain.go:426`. Do not stop at the one
site — census the class (`bytes.Equal` occurrences) rather than patching
the named one.

## B2 (blocking) — the parked-state contract is asserted, not decided, and has no operator remedy

The rev2 doc reclassifies round 1's repository-wide `ListSessions` failure
from bug to "the documented closed-listing rule". The brief asked three
questions about that. One is satisfied; two are not.

**Driven measurement (reviewer probe P5, through production entries):**

```
ListSessions before park            = 1 session,  err = nil
park session A at CreateStepRecord  (fired steps = [session-dir record])
ListSessions after park             = 0 sessions, err = ErrChainCorrupt (session A)
Resolve("healthy-one")              = "",         err = ErrChainCorrupt (session A)
GetRecord(session B, by id)         = ok
CreateSession(different valid bytes, same id) = ErrSessionExists   ← does NOT heal
ListSessions after heal attempt     = 0 sessions, err = ErrChainCorrupt
CreateSession(identical bytes)      = ok
ListSessions after identical retry  = 2 sessions, err = nil
```

So one parked session removes **every other session** from `ListSessions`
and from name resolution, and the only production entry that clears it is
a `CreateSession` carrying the byte-identical original record. There is no
delete, prune, or quarantine entry — by design. A caller that lost those
bytes leaves the repository's listing and name resolution permanently
broken for all sessions, with no API path back.

- **Q1 "is this the right contract?"** §5 and §13 do not settle listing
  directly, but SPEC.md:9082 (the §13.13 `recoverable_parked_state` row)
  requires that after this exact outcome "the lifecycle projection MUST be
  `parked`, `failed`, or a stopped owner … and status/doctor MUST expose
  the blocking reason and the same-operation retry". A listing entry that
  returns one error for the whole repository carries no per-session
  channel, so the §5.7 reducer and §14.4 rendering leaves that build on
  `ListSessions` cannot produce that projection from what this leaf
  exposes. §14.4 likewise requires per-session "warnings such as … divergent
  history". Either answer may be right for this leaf, but it is a design
  decision with a named downstream consequence and it is not recorded as
  one.
- **Q2 resume byte-equality** — satisfied. `resumeCreateLocked` compares
  the full record with `bytes.Equal` (`sessrepo.go:234`), not a prefix; the
  test plants a same-length collision and asserts the lengths match first;
  reviewer R4b (length-only narrowing) is KILLED by it.
- **Q3 "what heals a parked session when no retry comes?"** — the answer
  is "nothing; an operator must remove the directory", and it appears
  nowhere. Not in `doc.go`, not in the README section, not in the
  outcome's "Stated bounds for downstream leaves".

Where the rule currently lives: `store.go:78-79` states closed-listing
without the blast radius; `crash_test.go:202-205` states the blast radius
in a test comment; the outcome mentions it inside the crash evidence. None
of those is the durable place a design decision belongs, and none carries
the tradeoff, the spec reasoning, or the remedy.

Fix shape: record the decision where downstream leaves will read it
(`doc.go` stated bounds + README), with (a) why whole-listing closure was
chosen over per-session error reporting, referencing SPEC.md:9082 and
§14.4 explicitly, (b) the operator remedy for an un-retried parked
session, (c) what the §5.7/§14.4 leaves are expected to do given this
entry. If the answer is that `ListSessions` should carry a per-session
error instead, that is an implementation change, not a doc change — say
which and do it.

## N1 (non-blocking) — the load digest arm is witnessed only at position ≥ 1

`chain.go:293`. Mutants:

```
if digest.String() != indexed.EventID  →  ... && position > 0              SURVIVED
if digest.String() != indexed.EventID  →  ... && indexed.LeaseSequence != 1 SURVIVED
```

`TestLoadRefusesSwappedEventBlobs` swaps a two-event chain, so position 1
always refuses and the test passes regardless. A single-event chain whose
one blob is replaced by a different individually-valid event — the event
that links the Session Record, the highest-value substitution target — has
no driving test. Reviewer probe P3 confirms production refuses it, so this
is coverage granularity, not a defect: the gate does ship a killed
narrowing mutant (N8), which is the stated bar. Worth one vector.

## N2 (non-blocking) — six named bad-member vectors do not attribute to the arm they name

`TestAppendEventRefusesBadMember` and `TestCreateSessionRefusesMissingSessionID`
mutate the object *after* `buildEvent`/`buildRecord` computed the digest, so
identity verification refuses them one arm later with the same sentinel
(`ErrInvalidEvent` / `ErrInvalidRecord`). Removing the member bound changes
nothing the assertion can see:

```
lease_epoch lower bound 1 → 0        SURVIVED
lease_sequence lower bound 1 → 0     SURVIVED
lease_id UUIDv4 gate disabled        SURVIVED
predecessors non-empty gate disabled SURVIVED
record session_id UUIDv7 disabled    SURVIVED
record identity-field arm widened    SURVIVED
event identity-field arm widened     SURVIVED
```

Reviewer probe P4 measures *why*, and the answer is benign: the canonical
owner already refuses every one of these classes first —
`lease_epoch must be greater than zero`, `lease_sequence must be greater
than zero`, `member lease_id: UUIDv4 …`, `predecessors requires at least 1
entries`, `missing required member "session_id"`, etc. These sessrepo
bounds are unkillable-by-construction, exactly like the load verify-error
arm the outcome already states as a measured bound. The census cannot see
this (it is keyed by site, and other vectors in the same table still reach
the site), so it should be stated the way the verify-error arm is: name
the arms that are subsumed by `canonicaljson`, and stop presenting those
six vectors as witnesses of the member gate.

## Confirmed from the spawn brief — re-driven by this run, no rework needed

**G-B, F1 (the ordering claim).** The bound is **restated around what
actually refuses**, not gated. `loadSessionLocked` still has no schema gate
ahead of `VerifyObjectIdentity` (`chain.go:286`), and the comments in
`identity.go`, `identity_test.go`, `doc.go` and the README now say exactly
that, including an explicit withdrawal of the false universal. The
producer's argument for not gating (a schema gate would make the field arm
unreachable and its narrowing mutant unkillable) is correct — the self
field is resolved from the same schema member.

- Production `VerifyObjectIdentity` sites: **3**, matching the pin.
- §5.5 record through `GetRecord`: refused at the arm the bound names —
  reviewer mutant R1 (`&& field != canonicaljson.SelfRecordID`) is KILLED
  by `TestLoadRefusesProviderIdentityBlobAtEventPath`, so the refusal is
  attributable to the self-field clause and not to a downstream one.
- Allowlist re-planted in the **real** tree, not only the synthetic one:

```
internal/provhost/sessrepo/plant.go   → FAIL "production call sites outside the session leaf"
internal/xsessrepo/plant.go           → FAIL, same arm
extra Verify site inside the leaf     → FAIL "attestation sites = 4, want exactly 3"
```

  Round 1's segment-matching bypass is closed, and the count pin fires in
  both directions.

**G-C (the battery on load and recovery).** All five round-1 survivors and
M8 confirmed KILLED by independently written mutants:

| ref | reviewer mutant | verdict | killed by |
|---|---|---|---|
| M1a | field arm `&& field != SelfRecordID` | KILLED | `TestLoadRefusesProviderIdentityBlobAtEventPath` |
| M1b | digest arm `&& field != SelfEventID` (producer N8) | KILLED | `TestLoadRefusesSwappedEventBlobs` |
| M2 | `O_EXCL` → `O_TRUNC` | KILLED | `...DisagreeingDigestPathBytes`, `...AcrossHandles` |
| M6 | `bytes.Equal` → `len != len` (+ import) | KILLED | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| M7 | binding drops session half | KILLED | `TestLoadRefusesMismatchedSessionBinding` |
| M8 | binding drops record-digest half | KILLED | `TestLoadRefusesDetachedChainIndexWithReboundPredecessors` |

M8 now has a behavioural killer: the rebound-predecessor vector silences
the continuity arm so only the binding clause can refuse. The old
`TestLoadRefusesDetachedChainIndex` stays green under the mutant, which is
the correct precision control.

Also KILLED by reviewer mutants not in the producer's battery: stale-lease
arm narrowed to a two-epoch gap, divergent arm gated on lease sequence,
`Resolve` ambiguity `>1 → >2`, `AppendEvent` session-match narrowed to a
prefix, idempotent-lookup narrowed to a prefix, `Open` absolute-path gate,
resume chain-present guard, and `ListSessions` fail-closed converted to
`continue` (killed by two tests). The chain-continuity and recovery core
is solidly witnessed.

**Cross-process `O_EXCL`.** `TestAppendEventRefusesDisagreeingBytesAcrossHandles`
drives a second `Open` over the same root — a distinct `Repository` with a
distinct mutex — and asserts both the refusal and that the planted bytes
survive. `O_EXCL` is a kernel-level property, so a second handle exercises
the identical syscall path a second process would; the remaining gap is
only true OS-level concurrency, and the outcome states it is witnessed
deterministically through the mechanism rather than a timing race. That is
an honest bound, and R3 (`O_EXCL → O_TRUNC`) is killed *through the
fresh-handle test*, so the cross-handle path is load-bearing in the
evidence, not decorative.

**G-D (provenance and gates), all re-run by this review:**

| gate | result |
|---|---|
| detached-index `read-tree HEAD` + `add -A` + `write-tree` | `468d27367eb88d5a9da4d3ac540ff9c5a0831700` = record |
| `ls-tree` on that OID | the 7 new `sessrepo` files present (not HEAD's tree) |
| changed paths | exactly the 11 the CR enumerates |
| untracked ungitignored | only the 7 CR files; no scratch |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `GOOS=windows go build ./...` | 0 |
| `gofmt -l internal` | clean |
| `tracecheck` | 0 (contracts=60, acceptance=98) |
| `go test ./... -count=1` | 0, 24/24 packages ok |
| `go test -race` over all 24 packages (2 groups) | 0, 24 ok / 0 fail |

**Census and AC coverage.** 36 derived `refuse` sites, matching the README
figure; 3 production `VerifyObjectIdentity` sites, matching the pin. The
census correctly declines to run under a `-run` mask
(`fullSessrepoPackageTestRun`), so behavioral kills and census kills stay
separable — the producer's C1/C2/A1/A2 rows are honest about which gate
reddens. AC table is 10 of 10 rows with named driving tests and production
call sites; I checked each named test exists and drives the entry it
claims. The "no unsupported capability" row holds: `sessrepo` is
referenced from no other production package, adds no command, no doctor
result and no capability claim.

**Crash evidence.** The four interior create steps are real: `armCreateStep`
fires only at the targeted step, `mustFiredPrefix` requires the exact fired
prefix *and* that the armed point was consumed, so a crash point outside
its claimed window fails the test rather than passing silently. That was
round 1's finding and it is properly closed.

**F5 self-correction.** The outcome now states plainly that
`syncDirectory`/`writeAtomic`/`writeExclusive`/`installEventBlob` duplicate
`localstore` path policy and why they were not rewired, instead of claiming
no duplication. `internal/localstore` does exist with the same primitives
and has no production consumer, so the stated reason checks out.

## Routing

`to-dev`. B1 is a missing witness and a missing mutant row on an existing,
correct gate. B2 is a decision to record (or an implementation change, if
the decision goes the other way) — say which. N1 and N2 are cheap while
in there. Nothing here needs a spec change or a human decision, so this is
not a stop-the-line.
