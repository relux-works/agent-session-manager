# TASK-260830-wbpf1v review verdict — CR-TASK-260830-wbpf1v-1 rev 1

**Verdict: changes_requested → `to-dev`.**
**repeat-of: none** (first revision of this leaf).

Reviewer: RUN-260907-121d05. Everything below was executed by this run in the
story worktree at candidate tree `93942f4c8f07a865c546516c1be50b82f9c4e4e6`;
the tree was restored to that exact OID after every mutation (verified by
detached-index `write-tree` after the last experiment).

## What is right, and it is a lot

The architecture, the owner reuse, and the census machinery are correct and I
re-drove them rather than reading them.

| Check | How I drove it | Result |
|---|---|---|
| Provenance (G-F) | detached-index `read-tree HEAD` + `add -A` + `write-tree` | `93942f4c…` = record; `ls-tree` shows the 7 new `sessrepo` files really in the tree, not HEAD's |
| Changed paths | `git status --porcelain --untracked-files=all` | exactly the 11 CR paths, nothing else, no ungitignored scratch |
| Census denominator (G-D) | re-derived `refuse(` sites per file | 34 = 34 (sessrepo.go 5, chain.go 23, store.go 6) |
| Census fails closed, unregistered site | planted `unreachableProbe()` calling `refuse` in store.go | FAIL `refusal site without an exercised negative path: store.go:172` |
| Census fails closed, funnel bypass w/ sentinel | `refuse(ErrPredecessorLink,…)` → `fmt.Errorf("%w: …", ErrPredecessorLink)` | FAIL `refusals raised outside the refuse funnel: chain.go:192` |
| Census fails closed, funnel bypass w/o sentinel | `fmt.Errorf("invalid session record: …")` | KILLED behaviorally by `TestAppendEventRefusesLaterPredecessorMismatch` |
| Alias audit against **live production** | rewrote chain.go's import to `cjson` + direct calls | clean (correct: alias direct-call is allowed) |
| Alias audit, live var binding | `verify := canonicaljson.VerifyObjectIdentity` in chain.go | FAIL `chain.go:73:26: constructor … referenced outside direct-call position` |
| G-A substring plant | new pkg `internal/xsessrepo/` calling Verify | FAIL, names `internal/xsessrepo/plant.go:6` — segment match, not substring ✓ |
| G-A stale allowlist | moved `internal/sessrepo/` out of the tree | FAIL `sessrepo attestation exception is stale: no allowlisted call site remains` ✓ |
| G-A unparseable | planted a broken `.go` mentioning the identifier | FAIL `an unparseable derivation proves nothing` ✓ |
| G-A dot-import bare call | `import . canonicaljson` + `VerifyObjectIdentity(raw)` | FAIL, named ✓ |
| G-A ordering claim, decode sites | genuine §5.5 provider-identity record → `CreateSession` | refused `session record schema "urn:ax:schema:provider-identity", want "urn:ax:schema:session-record"` — the schema gate, named ✓ |
| Sequence boundary (G-B) | `> want` → `> want+1` and `> want+2` | both KILLED by `TestAppendEventRefusesSequenceGap` ✓ (want-1 killed by the repeat test) |
| G-E fuzz skip | enumerated targets: canonicaljson, scalar, secconftest, traceability | none touched by this CR; the disclosure is honest and correct ✓ |
| Repo gates | `go build ./…`, `go vet ./…`, `GOOS=windows go vet ./…`, `GOOS=windows go build ./…`, `gofmt -l internal/`, `go test ./… -count=1` (24 ok), `go test -race` on sessrepo/provhost/invcore/canonicaljson, `tracecheck` (contracts=60 acceptance=98), `go test ./internal/cigate` | all exit 0 / clean ✓ |

`cmd/` does not exist, so the outcome's `gofmt -l internal/` is the whole surface.

## F1 (blocking, G-A) — the ordering claim the allowlist rests on is false, measured

`identity.go` now states, and `doc.go` / `LOGBOOK.md` / the outcome repeat, that
the sessrepo Verify call sites "attest session-record and session-event bindings
**after a schema gate that refuses any other schema first**, so no transport gate
and **no persistence path recomputes a provider-identity digest today**."

There are three production call sites, not two:

```
internal/sessrepo/chain.go:73   decodeSessionRecord   — schema gate above it ✓
internal/sessrepo/chain.go:100  decodeSessionEvent    — schema gate above it ✓
internal/sessrepo/chain.go:286  loadSessionLocked     — NO schema gate; bytes come off disk
```

Driven, not reasoned. I planted the verbatim §5.5 provider-identity record as an
event blob named by a session's chain index and called the production entry
`GetRecord`:

```
GetRecord(planted provider-identity blob)
  -> session event chain is corrupt: re-verify chained event 0 for session …: <nil>
```

That `<nil>` is the finding: `canonicaljson.VerifyObjectIdentity` returned **no
error** — it resolved the provider-identity contract, recomputed the
provider-identity omit-self digest, and matched it. The refusal came afterwards,
from `field != SelfEventID`. So a persistence path does recompute a
provider-identity digest today, with no schema gate in front of it.

The record is still refused, so this is not an admission hole. It is a false
universal written into a production bound comment, the package doc's authority
split, the repository logbook, and the CR's headline finding — and the brief is
explicit that the allowlist's safety "rests entirely on that ordering claim."
Either gate the load-path bytes on schema before Verify, or rewrite the claim
around what is true (two entry sites gated by schema; one load-path site that
re-verifies stored bytes and refuses any non-`event_id` self field) — and pin the
rewritten claim with a test, because today nothing does (see F2).

Related, non-blocking: the refusal message at chain.go:288 formats `%v` on an
`err` that is `nil` on two of its three clauses, so the diagnostic reads
`re-verify chained event 0 …: <nil>` for both the wrong-self-field and the
wrong-digest cases. Three distinct clauses share one message and one census row.

## F2 (blocking, G-B + G-D) — "12 applied, 12 killed, 0 survivors" does not survive contact

README ships that number as a repository claim. I ran 11 behavioral mutants in
the same classes the battery claims (narrowing on gates). **Five survived the
full committed package suite**, all of them on the load and recovery paths the
battery never touches:

| Mutant | Site | Narrowing | Result |
|---|---|---|---|
| M1a | chain.go:286 | drop `field != SelfEventID` only | **SURVIVED** |
| M1b | chain.go:286 | drop `digest.String() != indexed.EventID` only | **SURVIVED** |
| M2 | chain.go `writeExclusive` | `O_EXCL` → `O_TRUNC` | **SURVIVED** |
| M6 | sessrepo.go `resumeCreateLocked` | stored-vs-retried byte equality bypassed | **SURVIVED** |
| M7 | chain.go:278 | drop `record.sessionID != sessionID` clause | **SURVIVED** |
| M3 | `installEventBlob` disagreement | narrowed | KILLED (`…DisagreeingDigestPathBytes`) |
| M4 / M4b | gap arm `> want+1` / `> want+2` | narrowed | KILLED |
| M5 | `asciiFold` `Z`→`T` | narrowed | KILLED |
| M8 | chain.go:278 drop `recordID != chain.RecordID` | narrowed | KILLED **by the census only** — `TestLoadRefusesDetachedChainIndex` still passes, satisfied by a downstream `checkAppend` arm, so the test does not witness the clause it names |

Kill rate on my sample: 5 of 11. The battery's 12/12 is a true statement about
twelve mutants that were aimed at the arms which already had tests.

**M1b has a real failure scenario, and I built the witness.** Two events appended
through `AppendEvent`, then the two blob *files* swapped so each digest path
holds the other event's bytes. Both blobs individually verify and both carry
`SelfEventID`, so the only clause that can refuse is the digest comparison.

- unmutated production: `ListEvents` → `session event chain is corrupt` ✓ (the
  clause works)
- under M1b: `ListEvents(swapped blobs) -> <nil>` — **content-addressed blob
  substitution admitted** — while `go test ./internal/sessrepo/` is **ok**.

A ~40-line test driving that vector through `ListEvents` reddens M1a, M1b and M8
at once.

**M2 is the append-only obligation itself.** `doc.go` says "files are
content-addressed and installed no-replace, so the chain is append-only **by
construction**"; README says "installed no-replace". Flip `O_EXCL` to `O_TRUNC`
and nothing in the repository notices. It is load-bearing: the `sessrepo` mutex
is per-process, and the suite itself models a second process as a second `Open`
over one root — two processes calling `CreateSession` with different records for
one session ID both pass `os.Stat(directory)`, and `O_EXCL` is the only thing
that stops the second from overwriting `record.json`. AC row 4 names
`TestCreateSessionRefusesDuplicateAndRewrite` and `…RefusesSequenceRepeat`, but
those are stopped by the `Stat` and the sequence arm, never by `O_EXCL`.

This is the DoD bullet "Every gate ships at least one NARROWING mutant" failing
for two gates (the load-path corruption gate and the no-replace install gate),
and it is the DoD/G-B bullet "no production path can rewrite or delete an
existing entry, **proven by a negative test**" resting on absence-of-a-method
plus an unwitnessed `O_EXCL`.

## F3 (blocking, G-B) — an interrupted create bricks the session ID and the whole listing

`CreateSession` performs four durable steps (`MkdirAll` dir → `writeExclusive`
record → `MkdirAll` events → `writeChain`). Both wired crash points sit outside
that span: `BeforeWrite`/`PointPrepareEnter` fires before the first byte and
`AfterCommit`/`PointCommitApply` after the last. `resumeCreateLocked` exists for
exactly the in-between window but handles only one of its two sub-windows
(record written, chain missing) and is driven by a hand-planted directory, never
by a crash point.

The other sub-window — crash after the directory exists, before `record.json` —
is unrecoverable. Driven through production entries:

```
CreateSession(retry)  -> session already exists: session 0198f4c8-…-1234567890ab already exists
GetRecord(same id)    -> session event chain is corrupt: read chain index …: no such file or directory
ListSessions()        -> session event chain is corrupt: read chain index …: no such file or directory
```

`CreateSession` reports a session that was never created as existing, and there
is no entry that can clear it. Worse, `ListSessions` fails closed on the first
corrupt directory — deliberate and documented — so one interrupted create takes
down the listing surface for **every** session in the repository.

Both DoD crash bullets are satisfied in letter (a fault before the durable write
and a fault after it are both driven). The gap is that a multi-write operation
was wired with a two-point vocabulary that structurally cannot express its own
recovery window, and the resume logic written for that window is half-covered.

## F4 (non-blocking, G-A) — the allowlist admits any directory named `sessrepo`

`sessrepoLeaf := sep + "sessrepo" + sep` matched with `strings.Contains` is a
segment match — the substring plant is correctly refused (P1 above) — but it
admits **any** path segment named `sessrepo`, not the owning directory. Planting
`internal/provhost/sessrepo/plant.go` with a production Verify call:

```
attestation scan: 132 production files, 5 parsed, 4 session-leaf call sites, 0 elsewhere
PASS
```

Four allowlisted sites where the real leaf has three. The comment says "the one
directory whose production `VerifyObjectIdentity` calls the rewritten bound
admits". Anchor it to the module-relative path (`internal/sessrepo/`) so the
allowlist means what it says.

## F5 (non-blocking, G-C) — reuse is right for the five named owners; the "no new path policy" claim is not

Against the brief's G-C list, `sessrepo` is clean: it reuses `environ`
(`DecodeStrictObject`, `CheckUUIDv7`, `CheckUint53Bounds`), `canonicaljson`
(`VerifyObjectIdentity`), `scalar` (`ParseDigest`, `ParseUUIDv7`,
`ParseUUIDv4`), `secconftest` points/outcomes/`Fault`, and `invcore` census
machinery, with no new decoder, no new bound check, and no new AST walker.
`secprim` is genuinely not needed: session IDs pass UUIDv7 grammar before any
join, and `ListSessions` skips symlinked entries via `DirEntry.IsDir()`.

But the outcome's "No new decoder, bound check, **path policy**, AST walker, or
journal: atomicity is temp-file + fsync + rename + dir-fsync" is not accurate.
`internal/localstore` already owns that policy — "the AX owner-local path layout
and immutable SHA-256 blob sink" — and this CR re-implements it:

| sessrepo | localstore |
|---|---|
| `store.go syncDirectory` | `paths.go:535 syncDirectory` — same body |
| `writeAtomic` temp+fsync+rename+dir-fsync | `object_store.go` staged write + `syncDirectory` |
| `writeExclusive` `O_EXCL` install | `projection.go:339` `O_EXCL` + `syncDirectory` |
| `installEventBlob` content-addressed idempotent install | `ObjectStore.PutBlob` |
| `Open(dataRoot)/sessions` layout | `ResolvePaths` / `PathClass` |

The two copies already disagree: sessrepo skips the directory fsync on
`runtime.GOOS == "windows"`, localstore does not. `localstore` has no production
consumer yet, so I am not asking you to rewire this leaf through it — but say so
as a stated bound with the reason, instead of claiming no new path policy exists.

## What I did not re-run

Full fuzz smoke (`-fuzztime=100x`) — I judged the producer's skip rationale and
agree with it (no fuzz target and no fuzzed package is touched), so I did not
substitute a run for the judgement. Catalog freshness and the six `cigate`
contract selections ran only as part of the unmasked `go test ./...`, not as
separate named invocations.

## Rework scope

1. Make the F1 claim true or make it accurate, and pin whichever you choose.
2. Drive the three unwitnessed clauses at chain.go:278/286 and the `O_EXCL`
   no-replace property, then re-run the battery with the load and recovery paths
   in the denominator. Republish the README battery numbers from that run.
3. Decide what an interrupted create before the record write must do, and wire a
   crash point that can reach the window.
4. Anchor the allowlist to `internal/sessrepo/`.
5. Correct the "no new path policy" claim, or reuse `localstore`.

Items 1 and 2 are the ones that block acceptance. 3 is a product decision on
recovery semantics — if the answer is "an operator repairs it", that is a stated
bound, not a fix, but it has to be stated.
