# TASK-260830-wbpf1v review verdict — CR-TASK-260830-wbpf1v-4 (rev 4)

**Verdict: ACCEPTED** (`accept_cr`, revision 4). Reviewer run `RUN-260907-fd2b95`.
**repeat-of: none** — the three rev3 findings are closed, each re-measured
rather than re-read. Three non-blocking notes are carried below; none of
them admits anything, breaks a gate, or contradicts an AC row.

All five blocking gates from the round-4 brief pass under my own
measurement. I applied 18 independent mutants/plants against this tree; the
tree OID was re-derived after every one and is unchanged.

## Candidate integrity

| Fact | Recorded | Measured by me |
|---|---|---|
| Candidate tree OID | `7ab337c849fb1aa532927327d07329ea8f97825b` | detached-index `read-tree HEAD` + `add -A` + `write-tree` = **same**, re-checked after every plant, final = same |
| Base OID | `7aa151a9…` | `git rev-parse HEAD` = same (not a self-diff: candidate tree ≠ HEAD tree) |
| Changed paths | 11 | `git status --porcelain -uall` = exactly 11, matching the CR list |
| Untracked ungitignored scratch | none | `git ls-files --others --exclude-standard` = the 7 new `internal/sessrepo` files only; my scratch is under gitignored `.temp/` |

## G-A (blocking) — is the census derived, or better-hardcoded? PASS

`auditSessrepoEquality` derives over `invcore.ScanProduction(".")` and
resolves `bytes.Equal` by import path. I attacked every shape the brief
named plus two more. **It passed none of them.** Each plant was applied to
the real production tree, run as a full unfiltered package run, then
restored (checksums verified):

| Plant | Shape | Result |
|---|---|---|
| A1 | package-level `var equalBytes = bytes.Equal`, called from production | **RED** — `chain.go:407:24: constructor "bytes.Equal" referenced outside direct-call position` + orphan row `chain.go:426`. The `var x = ctor` shape that has bypassed other gates in this repo does **not** bypass here. |
| A2 | func-table binding `map[string]func(...)bool{"bytes": bytes.Equal}` then `table["bytes"](a,b)` | **RED** — same alias failure + orphan row |
| A3 | aliased wrapper in a **new** production file (`import beq "bytes"`; `beq.Equal`) | **RED** — `content-equality site without a registered vector: helperplant.go:7` + orphan `chain.go:426` |
| A4 | `bytes.Compare(existing, data) != 0` | **RED** — `chain.go:426: content-equality spelling outside the classified set … widen the census or remove it`. Refuses; does not pass silently. |
| A5 | `bytes.EqualFold(existing, data)` | **RED** — same unclassifiable failure |
| A6 | hand-rolled length-only comparison as a **new** site, every registered site left intact | **GREEN — the declared blind spot, reproducing exactly as declared.** The bound is honest, not an overclaim. |

Method values / forwarding wrappers reduce to A3: the inner `bytes.Equal`
derives at its own line and lands unregistered.

Derived denominators re-derived in-package by me, not read from the
outcome: **refuse sites = 35** (`chain.go` 25, `sessrepo.go` 5, `store.go`
5), **sentinels = 14**, **equality sites = 2** (`sessrepo.go:258`,
`chain.go:426`) = 2 ledger rows.

I also drove the two shapes the census exists to catch:

- **Token-preserving funnel bypass** — `refuse(ErrUnknownSession, …)` →
  `fmt.Errorf("%w: …", ErrUnknownSession)` at `chain.go:259`. Behaviourally
  **green** (zero test failures), gate **red** on three counts: unrouted
  funnel bypass, a missing derived site, and the closed sentinel roster.
- **Check present but uncalled from production** — a new production
  `refuse` site reachable only by a direct helper call, with a test calling
  the helper. Suite `PASS`, gate red: *"refusal sites never reached beneath
  a production boundary entry (a direct helper call does not prove the
  production effect): probeplant.go:7"*.

## G-B (blocking) — the swapped-blob regression. PASS, green for the right reason

**What changed in production between the abandoned tree and this one.** I
extracted rev3's `chain.go` from `TASK-260830-wbpf1v_change-request_rev3.patch`
and diffed it against the candidate: the **only** delta is the three-line
`sessionDir` comment (G-D). The digest arm is byte-identical to rev3's, and
both tests were already present in rev3. So this is a restore, not a fix
shipped behind a test — the test was never ahead of its production.

Full production delta rev3 → rev4 is exactly two hunks: that comment, and a
4-line `parkedSummary` fix in `store.go` (see note N3).

My own narrowing mutants on the digest arm (`chain.go:293`), each killed by
the **named** test:

| Mutant | Killed by |
|---|---|
| `+ && field != canonicaljson.SelfEventID` (the exact regression shape, = N8) | `TestLoadRefusesSwappedEventBlobs`, `TestLoadRefusesSubstitutedFirstEventBlob` |
| `position > 0 &&` (exempt position 0, = N16) | `TestLoadRefusesSubstitutedFirstEventBlob` |
| `len(digest.String()) != len(indexed.EventID)` (length-only) | both of the above |

**Same-length substitution exists and is load-bearing.**
`TestAppendEventRefusesSameLengthDisagreeingBytes` plants a single-bit flip
at the digest path with an asserted-equal byte length, and asserts the
planted bytes survive. I confirmed it independently with a
**token-preserving** narrowing at `chain.go:426` —
`!bytes.Equal(existing[:0], data[:0])`, which keeps the searched-for token
so the census stays green and only behaviour moves: **KILLED** by
`…SameLengthDisagreeingBytes`, `…DisagreeingDigestPathBytes`,
`…DisagreeingBytesAcrossHandles`. The same shape at `sessrepo.go:258`:
**KILLED** by `TestCreateSessionRefusesResumeWithDifferingBytes`.

## G-C (blocking) — the three numbers. PASS

| Number | Published | Measured |
|---|---|---|
| Candidate tree | `7ab337c8` in the outcome | `7ab337c849fb…` — equal |
| Changed paths | 11, enumerated including `LOGBOOK.md` and both `internal/provhost` files | 11, exact match |
| Refusal denominator | **35** in the outcome ×3 **and in `README.md`** | in-package derivation = **35** (25/5/5). The shipped README figure is what the instrument produces. |

`README.md` also states `33 applied, 33 killed` — matches the battery log.

## G-D — F3 carried. PASS

`chain.go:245` now names what actually refuses (the record/session binding),
and a test drives it. I proved the named mechanism is the load-bearing one
by mutating each half of `chain.go:277` separately:

- drop the session half → **`TestReadEntriesRefuseTraversalSessionID`** and
  `TestLoadRefusesMismatchedSessionBinding` fail
- drop the digest half → `TestLoadRefusesDetachedChainIndexWithReboundPredecessors` fails

The traversal test stages a fully valid session **outside** the root so the
load reaches the binding rather than stopping at a missing directory — the
refusal is `ErrChainCorrupt` at `:277`, exactly as the comment now claims.
Unreachability of an escape via the write path is structural: `CreateSession`
joins the *validated* record's UUIDv7, and `AppendEvent` refuses before the
join on `event.sessionID != sessionID`.

## G-E — gates and provenance. PASS (all re-run by me at this tree)

| Gate | Result |
|---|---|
| `go build ./...`, `go vet ./...` | exit 0 |
| `gofmt -l` over tracked + untracked Go files | clean |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0 (contracts=60, acceptance=98) |
| `go test ./... -count=1` | exit 0, 24/24 packages, zero failures |
| `go test -race -count=1`, **3 groups of 8 covering all 24 packages exactly once** (axerror…config / dirnode…secconftest / secprim…tracecheck) | exit 0 in every group |
| `go test ./internal/sessrepo -cover` | 87.6% of statements |

Battery log `TASK-260830-wbpf1v_mutation-battery-rev4.log` audited, not
trusted: 33 applied / 33 killed / 0 survivors, classes separated 16
narrowing / 10 arm-deletion / 4 census-only / 3 audit-only, `X1
NOT_APPLIED` and `X2 COMPILE_FAIL` as **distinct rows excluded from the
applied denominator**, and the census/audit rows quote real gate messages
whose file:line I verified against this tree (`chain.go:187`, `:288`,
`:426`, `store.go:152/155`). The census-only and audit-only rows are
behaviourally green + full-run red — the correct shape.

"Not run" rationale checked, not accepted on assertion: fuzz targets live in
`canonicaljson`, `scalar`, `secconftest`, `cigate`, `traceability` — none in
the 11-path delta; no catalog/spec surface is touched.

## My independent boundary and provenance mutants (beyond the brief)

| Mutant | Killed by |
|---|---|
| gap arm `> want` → `> want+1` (admits gap-of-1) | `TestAppendEventRefusesSequenceGap` |
| repeat arm `< want` → `+1 < want` (admits the exact repeat) | `TestAppendEventRefusesSequenceRepeat` |
| later-predecessor arm → non-empty check | `TestAppendEventRefusesLaterPredecessorMismatch` |
| first-link arm → `len < 1` | `TestAppendEventRefusesFirstPredecessorMismatch` |

Both continuity arms are witnessed **at the edge**, not mid-range.

Provhost allowlist attacked in both directions: a 4th `VerifyObjectIdentity`
site inside the leaf → *"session-leaf attestation sites = 4, want exactly
3"*; a site in a foreign package → *"production call sites outside the
session leaf attest the identity binding"*. Live count in production is
exactly 3 (`chain.go:73`, `:100`, `:286`).

**Stated subsumption bound verified, not accepted.** The outcome states the
bad-member arms are subsumed by the canonical owner. I built a *freshly
valid* event with `lease_sequence = 2^53` and drove it: the owner refuses at
digest-computation time — *"integer 9007199254740992 is outside the AX
safe-integer interval"* — so no digest-valid event can carry an
out-of-range member. The subsumption is real, not cover for a hole, and the
shipped vectors sit at both edges (0 and 2^53).

**Append-only obligation, structurally checked.** Production's only
destructive syscalls are `os.Remove` of its own staged temp and `os.Rename`
onto `chain.json` (the replaceable index, by design). Records and event
blobs are only ever created through `O_WRONLY|O_CREATE|O_EXCL`. There is no
update or delete entry.

## Non-blocking notes (carry forward; none blocks this revision)

**N1 — `chain.go:417-418` and `:426`: a failed read reported as a proven
disagreement, and the comment claims the opposite.** The comment states *"A
path that exists but is not a readable file still fails the install, as an
operational error rather than a digest disagreement."* Measured: with a
directory at the blob path, `AppendEvent` returns *"session event chain is
corrupt: **digest path holds disagreeing bytes** for event"* and
`installEventBlob` returns `errDigestDisagreement` — while zero bytes were
read. `readErr != nil || !bytes.Equal(...)` conflates the two facts. It is
fail-closed and the sentinel is right, so nothing is admitted; the defect is
that the diagnostic asserts a fact production never established and the
comment states the opposite of the code. This is the same class the rev3
reviewer raised at `store.go:192` and this round fixed at `store.go:194` —
the sibling site was not swept.

**N2 — `sessrepo_test.go:298`: the byte-identity assertion for DoD row 4 is
a disjunction that admits non-byte-identical bytes.** The row claims records
reload *byte-identically*; the guard is `string(stored) != specRecordExample
&& !jsonEqual(stored, specRecordExample)`, whose second arm accepts a
semantically-equal-but-byte-different reload. Measured: narrowing the guard
to strict byte equality alone still **PASSES**, so the property genuinely
holds today and the fallback is dead — but the guard cannot fail for the
reason it names. Note also that the "process boundary" is a second `Open`
handle rather than a subprocess; that is declared in the test comment and is
sound (`Repository` carries no cache), so it is a stated bound, not a gap.

**N3 — outcome bookkeeping.** The changed-paths table labels `store.go`
*"(per-session parked listing, carried)"*, but `store.go` changed this round
— the absence-vs-read-failure fix in `parkedSummary`, driven by the new
`TestListSessionsReportsTornHintOnUnreadableRecord`. The producer used
"carried" precisely for byte-identical files elsewhere (`crash_test.go`,
`doc.go`, `sessrepo.go` are byte-identical to rev3), so this one label is
wrong rather than loose. Separately: `chain.go:78` and `chain.go:105` return
`fmt.Errorf` without a sentinel, so they sit outside both the refuse funnel
and the stated-bounds list. I measured them **unreachable** — the
schema→self-field contract in `catalog_gen.go` is 1:1 (`session-record` →
`record_id` for 1/2/3, `session-event` → `event_id` for 1/2/3/4) and the
schema gate precedes `Verify` — so the 35 denominator is honest. Named here
so a future change that makes either arm reachable does not get a free pass
from the census.

## Acceptance

Every AC row is driven through a production entry by a named committed test
(**10 of 10**; each name verified to exist exactly once in the committed
test files). Every gate ships a narrowing mutant, the source-text gate is
additionally attacked by two token-preserving mutants (funnel bypass, and
`bytes.Equal(x[:0], y[:0])`), the crash evidence runs through
`internal/secconftest`'s own points and outcomes on both sides of the
durable write, and the census fails closed on an unregistered site, an
orphan row, an unclassifiable site, an import alias, and a var binding.

Recorded with `accept_cr(TASK-260830-wbpf1v, revision=4,
evidence=TASK-260830-wbpf1v_review-verdict-rev4.md)`.
