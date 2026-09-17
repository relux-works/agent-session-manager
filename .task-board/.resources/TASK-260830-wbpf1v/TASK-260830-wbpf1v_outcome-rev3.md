# TASK-260830-wbpf1v outcome: session repository and event chain (rev3)

## Result

`internal/sessrepo` implements durable Session Records and append-only
per-session event chains with source sequence continuity (AX v0.5.0
§2.3 local steps, §5.1–5.2, §5.7/§14.4 stored facts). Handbook handoff:
ready for review. Candidate tree OID (detached-index `read-tree HEAD` +
`add` of the 8 listed paths + `write-tree`):
`93d4a106fd15cdb335539c2612d3ed84d70f919a`
(`ls-tree` shows `README.md` plus the 7 `sessrepo` files in the tree).

This is revision 3, reworking revision 2 after the reviewer verdict
`changes_requested` (`TASK-260830-wbpf1v_review-verdict-rev2.md`): B1
missing same-length vector and narrowing row for `chain.go:426`
(repeat-of rev1 F2), B2 parked-state contract asserted not decided with
no operator remedy (repeat-of rev1 F3); N1 position-0 digest vector and
N2 subsumed-arm attribution closed while here. Every finding below was
re-driven, not re-read.

## Changed paths (round-3 deltas, 6)

- `README.md` (sessrepo section: per-session parked channel + retry
  table + operator remedy; equality census; 30/30 battery)
- `internal/sessrepo/sessrepo.go` (`SessionSummary` gains the parked
  channel: `Parked`, `BlockingReason`, `RetryHint` + two retry-hint
  constants; no behavior change)
- `internal/sessrepo/store.go` (per-session parked listing; `Resolve`
  routes healthy sessions only; `parkedSummary` + torn-store hint)
- `internal/sessrepo/doc.go` (per-session parked decision record with
  SPEC citations, retry table, operator remedy, downstream consumption)
- `internal/sessrepo/sessrepo_test.go` (B1 same-length vector + equality
  census; B2 parked listing/resolve tests; N1 first-blob vector; N2
  attribution bounds)
- `internal/sessrepo/crash_test.go` (session-dir crash asserts the
  parked entry instead of the whole-repository error)

Unchanged leaf content, still CR scope: `internal/sessrepo/chain.go`,
`internal/sessrepo/census_test.go` (no production or census change in
round 3; the battery restores them by checksum after every mutant).

No untracked ungitignored scratch file is present (battery harness and
working log live under gitignored `.temp/TASK-260830-wbpf1v/`; the
battery log and this outcome are attached to the board task).

## Reviewer findings → fix → witness

| ID | Finding | Fix (production path) | Witness |
|---|---|---|---|
| B1 | `installEventBlob` byte-equality (`chain.go:426`) ships no same-length vector and no narrowing row; `!bytes.Equal` → `len != len` survives the suite; both witnesses plant 15-byte `{"forged":true}` | Same-length (1-bit-flip) disagreeing plant driven through production `AppendEvent`; narrowing row N13; class census test pinning exactly the two `bytes.Equal` sites | `TestAppendEventRefusesSameLengthDisagreeingBytes` (fails under the mutant with `error = <nil>` — the false-success scenario; old 15-byte tests stay green under it, proving the blindness); `TestContentEqualityComparisonsAreCensused` |
| B2 | One parked session bricks `ListSessions` (0 sessions + `ErrChainCorrupt`) and `Resolve` for the whole repo; only the byte-identical retry heals; no delete/prune entry, no documented remedy; §13.13/`SPEC.md:9082` and §14.4 need a per-session channel | Implementation change (not a doc-only record): `ListSessions` returns healthy sessions plus a parked entry per torn session (`Parked` + `recoverable_parked_state` blocking reason + same-operation retry hint); `Resolve` routes healthy sessions only; retry table + operator remedy documented in `doc.go` and README | `TestListSessionsReportsParkedSessionPerSession`, `TestListSessionsKeepsHealthySessionsBesideParked` (reviewer probe P5 committed: healthy sibling listable/resolvable, differing retry refused, identical retry heals), `TestResolveSkipsParkedSessions`; N14 (skip-parked) and N15 (parked-admitted-to-Resolve) narrowing rows |
| N1 | Digest arm (`chain.go:293`) witnessed only at position ≥ 1; `&& position > 0` and `&& LeaseSequence != 1` survive via the two-event swap | Single-event substituted-first-blob vector (both blobs verify with `event_id`; only the digest arm can refuse) | `TestLoadRefusesSubstitutedFirstEventBlob` (fails under `&& position > 0` while the swap stays green); N16 narrowing row with precision control |
| N2 | Six bad-member vectors mutate after the digest, so `canonicaljson` refuses one arm earlier with the same sentinel; member-gate attribution unkillable | Attribution bound stated, not patched: test comments + outcome state the subsumption; vectors kept as sentinel-level regression at the production entry | Textual bound (§Measured bounds); probe-equivalent reasoning recorded |

## AC coverage: 10 of 10 rows driven through production entries

| # | AC row | Production call site | Named driving test(s) |
|---|---|---|---|
| 1 | Persist Session Records | `(*Repository).CreateSession` | `TestCreateSessionPersistsSpecRecordFixture`, `TestSessionRecordReloadsByteIdenticallyAcrossProcessBoundary` |
| 2 | Validate Session Records | `(*Repository).CreateSession` | `TestCreateSessionRefusesMalformedFrame`, `...ForeignSchema`, `...MissingSessionID`, `...DigestMismatch` |
| 3 | Appends succeed in order | `(*Repository).AppendEvent` | `TestAppendEventChainInOrder` |
| 4 | No rewrite/delete path | `CreateSession`/`AppendEvent` (no update/delete entry exists) + `writeExclusive` O_EXCL | `TestCreateSessionRefusesDuplicateAndRewrite`, `TestAppendEventRefusesSequenceRepeat`, `TestAppendEventRefusesDisagreeingBytesAcrossHandles` (fresh handle: filesystem-only refusal), `TestAppendEventRefusesSameLengthDisagreeingBytes` (planted bytes survive) |
| 5 | Gap refused (named arm, at edge) | `checkAppend` via `AppendEvent` | `TestAppendEventRefusesSequenceGap` (skip, first-past-one edge, successor-past-one, SPEC stopped fixture) |
| 6 | Repeat refused (named arm) | `checkAppend` via `AppendEvent` | `TestAppendEventRefusesSequenceRepeat` (two-events-at-one-sequence) |
| 7 | Exact contract fixtures pass | `CreateSession`/`AppendEvent` | SPEC §5.1 direct record (`d617…`), task_board record (`0acd…`), SPEC §5.2 stopped event (verifies, refused as gap); SPEC §5.5 provider-identity pinned (`c879…` under `record_id`) and refused on the load path |
| 8 | Negative/refusal cases pass; parked sessions reportable per session | all entries; `ListSessions`/`Resolve` | 36-site census, every site boundary-driven; load rows have one plant each (tampered, provider-identity, swapped, substituted-first); parked channel: `TestListSessionsReportsParkedSessionPerSession`, `...KeepsHealthySessionsBesideParked`, `TestResolveSkipsParkedSessions` |
| 9 | Crash/idempotency evidence | `BeforeWrite`/`AfterCommit` + `AfterCreateStep` hooks | 8 crash tests: 4 outer (prepare/commit × create/append) + 4 interior create steps, each with point, outcome, fired-prefix, and heal/terminal assertions; session-dir crash now asserts the parked listing entry plus identical-retry heal |
| 10 | No unsupported capability | API surface (only `Repository` entries; no command/doctor/result wiring) | stated bound with surface evidence; README claims only what exists |

## Gates actually run (exit codes observed in this session, rev3 tree)

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `GOOS=windows go build ./...` | 0 |
| `gofmt -l internal/` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | 0 (contracts=60, acceptance=98) |
| `go test ./... -count=1` | 0, all 24 packages ok |
| `go test ./internal/sessrepo/ -cover -count=1` | 0, 87.2% of statements |
| `go test -race` full repository (sessrepo; 18 fast pkgs; canonicaljson+provhost; localstore+catalog-cmd+tracecheck-cmd) | 0 in all 4 groups |
| mutation battery `all` | 0 (30 applied, 30 killed, 0 survivors; controls distinct) |

Not run: full fuzz smoke (`-fuzztime=100x` over 13 targets) and the
catalog-freshness generate check. Rationale: this change adds no fuzz
target and modifies no fuzzed package (`secconftest`, `canonicaljson`,
`scalar` untouched) and no catalog surface; both stay CI-lane
activities on unchanged surface.

## Census (production-derived denominator, invcore)

- Derived `refuse(` sites: **36** (sessrepo.go 5, chain.go 25, store.go 6;
  round 3 adds no refusal — parked state is reported as data).
  The load re-verification holds three separate rows — verify failure,
  non-`event_id` self field, index-disagreeing digest — each with its own
  boundary-driven plant (`...TamperedEventBlob`,
  `...ProviderIdentityBlobAtEventPath`, `...SwappedEventBlobs`, plus the
  N1 `...SubstitutedFirstEventBlob` for position 0).
- Exercised beneath a boundary entry: **36 of 36**; exercised-outside: 0.
- Sentinel roster derived from `Err*` vars; observed codes match exactly.
- Alias audit (`refuse` + 8 watched owner delegations) clean; unrouted
  `fmt.Errorf`-with-sentinel audit clean.
- Control plants, all asserting fail-closed: DiffSets unregistered site,
  orphan row, zero-site derivation; alias audit with import-alias
  direct-call (clean), import-alias var binding (fails), local var binding
  (fails), dot import (fails), shadowed name (fails); unparseable plant
  (harness fails).

## Content-equality census (B1 class table)

| Site | Same-length vector | Narrowing mutant | Verdict |
|---|---|---|---|
| `sessrepo.go` `resumeCreateLocked` (`!bytes.Equal(existing, recordJSON)`) | `TestCreateSessionRefusesResumeWithDifferingBytes` (length asserted equal first) | N10 length-only | KILLED |
| `chain.go` `installEventBlob` (`!bytes.Equal(existing, data)`) | `TestAppendEventRefusesSameLengthDisagreeingBytes` (1-bit flip, length asserted equal) | N13 length-only | KILLED (new test fails with `error = <nil>`; the 15-byte vectors stay green under it) |

`TestContentEqualityComparisonsAreCensused` pins the class at two sites
(`sessrepo.go:1`, `chain.go:1`, `store.go:0`); a third `bytes.Equal`
fails the suite instead of shipping unwitnessed.

## Mutation battery (production-derived denominator: 36 sites + mechanism targets)

30 applied, **30 killed**, 0 survivors (16 narrowing, 10 arm-deletion,
2 census-only, 2 audit-only; NOT_APPLIED and COMPILE_FAIL separate).
Full log: `TASK-260830-wbpf1v_mutation-battery-rev3.log` on the board
task; harness at `.temp/TASK-260830-wbpf1v/mutation_battery.sh`
(gitignored scratch) with exact-count application (any miscount aborts
the battery) and checksum restore after every mutant.

| Mutant | What it narrows/admits | Named killing test |
|---|---|---|
| N1 `> want` → `> want+1` | gap arm admits gap-of-1 | `TestAppendEventRefusesSequenceGap` |
| N2 `< want` → `+1 < want` | repeat arm admits exact repeat | `TestAppendEventRefusesSequenceRepeat` |
| N3 `!= 1` → `< 1` | first-link admits extra predecessors | `TestAppendEventRefusesFirstPredecessorMismatch` |
| N4 contains-tail → non-empty | predecessor arm admits wrong link | `TestAppendEventRefusesLaterPredecessorMismatch` |
| N5 record gate admits session-event | record gate admits one foreign class member | `TestCreateSessionRefusesForeignSchema` |
| N6 fold A–Z → A–T | ambiguity gate admits U–Z collisions (vector carries Y) | `TestResolveRefusesCaseFoldCollision` |
| N7 field arm exempts `record_id` (M1a) | load admits provider-identity blobs | `TestLoadRefusesProviderIdentityBlobAtEventPath` |
| N8 digest arm exempts well-formed events (M1b) | load admits blob substitution | `TestLoadRefusesSwappedEventBlobs` |
| N9 `O_EXCL` → `O_TRUNC` (M2) | install admits disagreeing bytes | `...DisagreeingDigestPathBytes` + `...AcrossHandles` |
| N10 byte equality → length equality (M6) | resume admits same-length collision | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| N11 binding drops session half (M7) | load admits cross-session record | `TestLoadRefusesMismatchedSessionBinding` |
| N12 binding drops digest half (M8) | load admits detached index | `...DetachedChainIndexWithReboundPredecessors` |
| N13 blob equality → length equality (B1) | install admits same-length forgery | `TestAppendEventRefusesSameLengthDisagreeingBytes` |
| N14 parked entry dropped (skip) | listing hides the parked session | parked listing tests (`...ReportsParkedSessionPerSession`, `...KeepsHealthySessionsBesideParked`) |
| N15 `!Parked` → `true` | Resolve admits parked siblings (collision) | `TestResolveSkipsParkedSessions` |
| N16 digest arm exempts position 0 (N1) | load admits first-blob substitution | `TestLoadRefusesSubstitutedFirstEventBlob` |
| D1 delete gap arm | — | `TestAppendEventRefusesSequenceGap` |
| D2 delete repeat arm | — | `TestAppendEventRefusesSequenceRepeat` |
| D3 delete divergent arm | — | `TestAppendEventPreservesDivergentBranchWithoutApplying` |
| D4 delete idempotent branch | — | `TestAppendEventIdempotentRetry` |
| D5 delete resume chain-present guard | — | `TestCreateSessionRefusesDuplicateAndRewrite` |
| D6 delete load self-field arm | — | `TestLoadRefusesProviderIdentityBlobAtEventPath` |
| D7 delete load digest arm | — | `TestLoadRefusesSwappedEventBlobs` |
| D8 delete record/chain binding check | — | binding tests (M7 + M8 vectors) |
| D9 delete resume byte-equality guard | — | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| D10 delete resume torn-state guard | — | `TestCreateSessionRefusesResumeOfChainWithoutRecord` |
| C1 funnel → direct `fmt.Errorf` on write path (token preserved, behavior identical) | census-only | full-package run FAILs on `refusals raised outside the refuse funnel` while behavioral tests PASS |
| C2 funnel → direct `fmt.Errorf` on load path (token preserved, behavior identical) | census-only | full-package run FAILs on `refusals raised outside the refuse funnel` while behavioral tests PASS |
| A1 unreachable extra `refuse` site | audit-only | full-package run FAILs on `refusal site without an exercised negative path` while behavioral tests PASS |
| A2 var-bound funnel (`deny := refuse`) | audit-only | full-package run FAILs on the alias audit (`outside direct-call position`) while behavioral tests PASS |
| X1 absent pattern | NOT_APPLIED (harness control) | — (pattern count 0, checksums unchanged) |
| X2 broken syntax | COMPILE_FAIL (harness control) | — (build fails distinctly from killed/survived) |

Narrowing controls (sibling arms hold under the mutant, proving a
narrowing and not a deletion): N7 swap+tampered green; N8 tampered
green; N9 idempotent+in-order green; N12 old detached test green (passes
via the downstream continuity arm — the rebound test names the clause);
N13 different-length plant green (length still refuses it); N16 swap
green (refuses at position 1).

Token-preserving-behavior-change mutants (C1, C2 keep their sentinel
tokens while bypassing the funnel; A2 keeps behavior through the alias)
are executed through the behavioral suite (green) plus the full-package
gate (red), never through the static checker alone.

Measured bounds (no deletion mutant, by measurement, not by omission):

- The load verify-error arm is behaviorally subsumed by the self-field
  arm — `VerifyObjectIdentity` returns field `""` on every error path, so
  deleting the error arm changes no observable outcome (the tampered plant
  is then refused one line down). The row stays for diagnostic attribution
  and is pinned by the census, which requires the tampered-blob plant to
  exercise exactly that line.
- The sessrepo member arms named in N2 (event `lease_sequence`/`lease_epoch`
  lower bounds, `lease_id` UUIDv4, non-empty `predecessors`, record
  `session_id` UUIDv7, both decoder identity-field arms) are subsumed by
  the canonical owner for the post-digest-mutation classes: `canonicaljson`
  refuses every one first (`lease_sequence must be greater than zero`,
  `member lease_id: UUIDv4 …`, `predecessors requires at least 1 entries`,
  `missing required member "session_id"`, …) with the same sentinel. The
  vectors pin the sentinel at the production entry; they do not attribute
  to the member clause.

## Crash evidence (owner `secconftest` points, owner outcomes)

Outer boundaries (`BeforeWrite`/`AfterCommit`):

- `PointPrepareEnter` armed on create: `*Fault(safe_retry)`; session stays
  unknown; identical retry succeeds.
- `PointPrepareEnter` armed on append: `*Fault(safe_retry)`; chain length
  unchanged; retry appends at position 0.
- `PointCommitApply` armed on create: `*Fault(recoverable_parked_state)`;
  record reads back (crashed commit counts as committed); retry names the
  terminal state (`ErrSessionExists`).
- `PointCommitApply` armed on append: `*Fault(recoverable_parked_state)`;
  event reads back; identical retry returns the committed reference with no
  second chain entry; committed bytes survive a later refusal.

Interior create steps (`AfterCreateStep`, one armed point per boundary,
each test asserting the exact fired-step prefix and arm consumption):

- After session directory: `*Fault(recoverable_parked_state)`; `GetRecord`
  refuses per session while `ListSessions` reports the bare directory on
  the parked channel (bare-directory retry hint); identical retry resumes
  the create (record reads back, listing shows one healthy session).
- After record install: parked record-without-chain (parked-record retry
  hint, identity filled); retry completes the index through the resume
  path; the chain continues at position 0.
- After events directory: same parked shape; retry completes it.
- After chain index: crashed commit counts as committed; record reads
  back; retry names the terminal state (`ErrSessionExists`).

Resume path (production retry, crash-driven above plus hand plants):
record-without-chain completes on retry, then refuses as existing;
bare directory completes with the retry bytes; colliding bytes and
chain-without-record refuse without mutating. Reload across a second
`Open` handle is byte-identical and the chain continues there.

## Reuse (no owner duplicates, F5 correction carried)

Validation via `canonicaljson.VerifyObjectIdentity`; frame/member rules via
`environ.DecodeStrictObject`/`CheckUUIDv7`/`CheckUint53Bounds`;
grammars via `scalar.ParseDigest`/`ParseUUIDv7`/`ParseUUIDv4`;
crash vocabulary via `secconftest` points/outcomes/`Fault`;
census machinery via `invcore`. No new decoder, bound check, AST walker,
or journal. `syncDirectory`, `writeAtomic`, `writeExclusive`, and
`installEventBlob` DO duplicate `localstore` path policy (stated in rev2
with the reason: `localstore` has no production consumer yet, so this
leaf is not rewired through it). Session IDs (UUIDv7 grammar) need no
traversal guard beyond the scalar owner.

## Stated bounds for downstream leaves

Single authoritative branch per session (same-epoch second lease is
preserved, never applied; arbitration is the leases story). No
`SessionState` derivation (reducer leaf `1r9wrr`): consume
`SessionSummary.Parked` for the parked/failed projection,
`BlockingReason` for status/doctor reason and warnings, `RetryHint` for
the retry/remedy line. No peer names or interactive choice
(name-resolution leaf `21gygk`): `Resolve` routes healthy sessions only.
No list rendering (CLI leaf). Chain index is a verified cache; blobs are
truth. Concurrent racing creates serialize on `O_EXCL` plus the resume
guards plus the atomic chain index — witnessed deterministically through
the blob path and the fresh-handle test, not through a timing race. The
`AfterCreateStep` hook is test surface (nil in normal operation), like
`BeforeWrite`/`AfterCommit` before it. There is no delete/prune entry by
design; the operator remedy for an unhealable parked session is the
documented filesystem removal in `doc.go`.

