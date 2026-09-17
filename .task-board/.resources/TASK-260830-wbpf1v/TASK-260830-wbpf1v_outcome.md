# TASK-260830-wbpf1v outcome: session repository and event chain (rev2)

## Result

`internal/sessrepo` implements durable Session Records and append-only
per-session event chains with source sequence continuity (AX v0.5.0
§2.3 local steps, §5.1–5.2, §5.7/§14.4 stored facts). Handbook handoff:
ready for review. Candidate tree OID (detached-index `read-tree HEAD` +
`add -A` + `write-tree`): `468d27367eb88d5a9da4d3ac540ff9c5a0831700`
(`ls-tree` shows the 7 `sessrepo` files in the tree, not HEAD's).

This is revision 2, reworking revision 1 after the reviewer verdict
`changes_requested` (`TASK-260830-wbpf1v_review-verdict-rev1.md`: F1
false ordering claim, F2 five surviving load/recovery mutants, F3
interrupted-create brick; F4 allowlist anchoring fixed while here; F5
acknowledged, see §Reuse). Every finding below was re-driven, not
re-read; §Reviewer findings names the fix and the witness for each.

## Changed paths (exactly the CR content, 11)

- `README.md` (sessrepo section: O_EXCL-first install, load-arm division, four-step resume, 36-site census, 26/26 battery)
- `LOGBOOK.md` (round-2 entry superseding the false bound; round-1 entry kept as history)
- `internal/provhost/identity.go` (bound comment restated around the measured truth, no behavior change)
- `internal/provhost/identity_test.go` (scanner extracted, allowlist anchored to `<root>/internal/sessrepo/`, site count pinned at 3, synthetic-tree adversarial test)
- `internal/sessrepo/doc.go` (authority split states the load-path division)
- `internal/sessrepo/sessrepo.go` (four interior crash steps, two-state resume)
- `internal/sessrepo/chain.go` (three load verify rows, O_EXCL-first blob install)
- `internal/sessrepo/store.go` (unchanged in rev2, still CR content)
- `internal/sessrepo/sessrepo_test.go` (8 new behavioral tests, sharpened fold vector)
- `internal/sessrepo/crash_test.go` (4 interior crash tests + step-arm helpers)
- `internal/sessrepo/census_test.go` (unchanged in rev2, still CR content)

`git status --porcelain --untracked-files=all` shows only these 11: 4
modified files plus the new `internal/sessrepo/` deliverable (7 files).
No untracked ungitignored scratch file is present (battery harness and
log live under gitignored `.temp/TASK-260830-wbpf1v/`; the log is
attached to the board task).

## Reviewer findings → fix → witness

| ID | Finding | Fix (production path) | Witness |
|---|---|---|---|
| F1 | Ordering claim false: `loadSessionLocked` Verify has no schema gate; provider-identity digest recomputed and matched, refused only by `field != SelfEventID` | Bound restated (entry decodes gate schema; load re-verifies and refuses at field/digest arms) in `identity.go`, `identity_test.go`, `sessrepo/doc.go`, README, LOGBOOK, outcome; verify split into three named rows (nil-error diagnostic gone) | `TestLoadRefusesProviderIdentityBlobAtEventPath` (+ pin `TestProviderIdentityPlantVerifiesUnderRecordID`) |
| F2 | Battery 12/12 measured the write path only; 5 survived (M1a field, M1b digest, M2 O_EXCL, M6 resume equality, M7 session binding) + M8 census-only | Denominator extended to load/recovery; O_EXCL-first install routes the disagree case through the exclusive create | Battery 26/26 below; each reviewer mutant maps to N7/N8/N9/N10/N11 (+N12 for M8) |
| F3 | Crash after `MkdirAll(dir)`, before `record.json`: retry `ErrSessionExists`, reads corrupt, listing fails for the whole repo | `resumeCreateLocked` heals the bare-directory state (retry bytes become the record when no chain exists); `AfterCreateStep` fires after each of the 4 steps | 4 step-crash tests, each asserting fired-step prefix + arm consumption + heal |
| F4 | Allowlist admitted any path segment named `sessrepo` (`internal/provhost/sessrepo/` plant: 4 sites, real count 3) | Anchored on `<root>/internal/sessrepo/`, exact count pinned at 3 | `TestAttestationAllowlistAnchorsOwningPath` (synthetic tree: provhost/sessrepo + xsessrepo refused, leaf admitted) |
| F5 | "No new path policy" claim wrong: `syncDirectory`/`writeAtomic`/`writeExclusive`/`installEventBlob` duplicate `localstore` (no consumer yet) | Claim corrected, not rewired: §Reuse states the duplication with the reason | Textual bound; rewiring through `localstore` is out of leaf scope |

Gating the load path on schema (the alternative F1 offered) was measured
and rejected: `VerifyObjectIdentity` resolves the self field from the same
schema member the gate would check, so any blob reaching the field arm with
a nil error would carry `event_id` by construction — the arm would become
unreachable and its narrowing mutant unkillable, recreating F2.

## AC coverage: 10 of 10 rows driven through production entries

| # | AC row | Production call site | Named driving test(s) |
|---|---|---|---|
| 1 | Persist Session Records | `(*Repository).CreateSession` | `TestCreateSessionPersistsSpecRecordFixture`, `TestSessionRecordReloadsByteIdenticallyAcrossProcessBoundary` |
| 2 | Validate Session Records | `(*Repository).CreateSession` | `TestCreateSessionRefusesMalformedFrame`, `...ForeignSchema`, `...MissingSessionID`, `...DigestMismatch` |
| 3 | Appends succeed in order | `(*Repository).AppendEvent` | `TestAppendEventChainInOrder` |
| 4 | No rewrite/delete path | `CreateSession`/`AppendEvent` (no update/delete entry exists) + `writeExclusive` O_EXCL | `TestCreateSessionRefusesDuplicateAndRewrite`, `TestAppendEventRefusesSequenceRepeat`, `TestAppendEventRefusesDisagreeingBytesAcrossHandles` (fresh handle: filesystem-only refusal) |
| 5 | Gap refused (named arm, at edge) | `checkAppend` via `AppendEvent` | `TestAppendEventRefusesSequenceGap` (skip, first-past-one edge, successor-past-one, SPEC stopped fixture) |
| 6 | Repeat refused (named arm) | `checkAppend` via `AppendEvent` | `TestAppendEventRefusesSequenceRepeat` (two-events-at-one-sequence) |
| 7 | Exact contract fixtures pass | `CreateSession`/`AppendEvent` | SPEC §5.1 direct record (`d617…`), task_board record (`0acd…`), SPEC §5.2 stopped event (verifies, refused as gap); SPEC §5.5 provider-identity pinned (`c879…` under `record_id`) and refused on the load path |
| 8 | Negative/refusal cases pass | all entries | 36-site census, every site boundary-driven (see §Census); load rows have one plant each (tampered blob, provider-identity blob, swapped blobs) |
| 9 | Crash/idempotency evidence | `BeforeWrite`/`AfterCommit` + `AfterCreateStep` hooks | 8 crash tests: 4 outer (prepare/commit × create/append) + 4 interior create steps, each with point, outcome, fired-prefix, and heal/terminal assertions (see §Crash) |
| 10 | No unsupported capability | API surface (only `Repository` entries; no command/doctor/result wiring) | stated bound with surface evidence; README claims only what exists and passes `cigate` claims gates |

## Gates actually run (exit codes observed in this session, rev2 tree)

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `GOOS=windows go build ./...` | 0 |
| `gofmt -l internal/` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | 0 (contracts=60, acceptance=98) |
| `go test ./... -count=1` | 0, all 24 packages ok |
| `go test ./internal/sessrepo/ ./internal/provhost/ -count=1 -cover` | 0 (sessrepo 86.7%, provhost 86.0%) |
| `go test -race` over all 24 packages (4 sequential groups) | 0 |
| catalog freshness (`go generate ./internal/catalog/` + `git diff --exit-code`) | 0, current |
| `cigate` contract/targets/claims suites (run inside `go test ./...`) | 0 |
| fuzz target count (`-list '^Fuzz'` = 13) | 0, no drift |

Not run: full fuzz smoke (`-fuzztime=100x` over 13 targets). Rationale:
this change adds no fuzz target and modifies no fuzzed package
(`secconftest`, `canonicaljson`, `scalar` untouched); the smoke is a
CI-lane activity on unchanged surface.

## Census (production-derived denominator, invcore)

- Derived `refuse(` sites: **36** (sessrepo.go 5, chain.go 25, store.go 6).
  The load re-verification holds three separate rows — verify failure,
  non-`event_id` self field, index-disagreeing digest — each with its own
  boundary-driven plant (`...TamperedEventBlob`,
  `...ProviderIdentityBlobAtEventPath`, `...SwappedEventBlobs`).
- Exercised beneath a boundary entry: **36 of 36**; exercised-outside: 0.
- Sentinel roster derived from `Err*` vars; observed codes match exactly.
- Alias audit (`refuse` + 8 watched owner delegations) clean; unrouted
  `fmt.Errorf`-with-sentinel audit clean.
- Control plants, all asserting fail-closed: DiffSets unregistered site,
  orphan row, zero-site derivation; alias audit with import-alias
  direct-call (clean), import-alias var binding (fails), local var binding
  (fails), dot import (fails), shadowed name (fails); unparseable plant
  (harness fails).

## Mutation battery (production-derived denominator: 36 sites + mechanism targets)

26 applied, **26 killed**, 0 survivors. Census-only and audit-only kills
are separate classes, never folded into behavioral kills. Full log:
`TASK-260830-wbpf1v_mutation-battery.log` on the board task; harness at
`.temp/TASK-260830-wbpf1v/mutation_battery.sh` (gitignored scratch) with
exact-count application (any miscount aborts the battery) and checksum
restore after every mutant.

| Mutant | Class | What it narrows/admits | Named killing test |
|---|---|---|---|
| N1 `> want` → `> want+1` | narrowing | gap arm admits gap-of-1 | `TestAppendEventRefusesSequenceGap` |
| N2 `< want` → `+1 < want` | narrowing | repeat arm admits exact repeat | `TestAppendEventRefusesSequenceRepeat` |
| N3 `!= 1` → `< 1` | narrowing | first-link admits extra predecessors | `TestAppendEventRefusesFirstPredecessorMismatch` |
| N4 contains-tail → non-empty | narrowing | predecessor arm admits wrong link | `TestAppendEventRefusesLaterPredecessorMismatch` |
| N5 record gate admits session-event | narrowing | record gate admits one foreign class member | `TestCreateSessionRefusesForeignSchema` |
| N6 fold A–Z → A–T | narrowing | ambiguity gate admits U–Z collisions (vector carries Y) | `TestResolveRefusesCaseFoldCollision` |
| N7 field arm exempts `record_id` (M1a) | narrowing | load admits provider-identity blobs | `TestLoadRefusesProviderIdentityBlobAtEventPath` |
| N8 digest arm exempts well-formed events (M1b) | narrowing | load admits blob substitution | `TestLoadRefusesSwappedEventBlobs` |
| N9 `O_EXCL` → `O_TRUNC` (M2) | narrowing | install admits disagreeing bytes | `...DisagreeingDigestPathBytes` + `...AcrossHandles` |
| N10 byte equality → length equality (M6) | narrowing | resume admits same-length collision | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| N11 binding drops session half (M7) | narrowing | load admits cross-session record | `TestLoadRefusesMismatchedSessionBinding` |
| N12 binding drops digest half (M8) | narrowing | load admits detached index | `...DetachedChainIndexWithReboundPredecessors` |
| D1 delete gap arm | arm-deletion | — | `TestAppendEventRefusesSequenceGap` |
| D2 delete repeat arm | arm-deletion | — | `TestAppendEventRefusesSequenceRepeat` |
| D3 delete divergent arm | arm-deletion | — | `TestAppendEventPreservesDivergentBranchWithoutApplying` |
| D4 delete idempotent branch | arm-deletion | — | `TestAppendEventIdempotentRetry` |
| D5 delete resume chain-present guard | arm-deletion | — | `TestCreateSessionRefusesDuplicateAndRewrite` |
| D6 delete load self-field arm | arm-deletion | — | `TestLoadRefusesProviderIdentityBlobAtEventPath` |
| D7 delete load digest arm | arm-deletion | — | `TestLoadRefusesSwappedEventBlobs` |
| D8 delete record/chain binding check | arm-deletion | — | binding tests (M7 + M8 vectors) |
| D9 delete resume byte-equality guard | arm-deletion | — | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| D10 delete resume torn-state guard | arm-deletion | — | `TestCreateSessionRefusesResumeOfChainWithoutRecord` |
| C1 funnel → direct `fmt.Errorf` on write path (token preserved, behavior identical) | census-only | — | full-package run FAILs on `refusals raised outside the refuse funnel` while behavioral tests PASS |
| C2 funnel → direct `fmt.Errorf` on load path (token preserved, behavior identical) | census-only | — | full-package run FAILs on `refusals raised outside the refuse funnel` while behavioral tests PASS |
| A1 unreachable extra `refuse` site | audit-only | — | full-package run FAILs on `refusal site without an exercised negative path` while behavioral tests PASS |
| A2 var-bound funnel (`deny := refuse`) | audit-only | — | full-package run FAILs on the alias audit (`outside direct-call position`) while behavioral tests PASS |
| X1 absent pattern | NOT_APPLIED | harness control | — (pattern count 0, checksums unchanged) |
| X2 broken syntax | COMPILE_FAIL | harness control | — (build fails distinctly from killed/survived) |

Narrowing controls (sibling arms hold under the mutant, proving the
mutant is a narrowing and not a deletion): under N7 the swap and
tampered tests stay green; under N8 the tampered test stays green;
under N9 idempotent retry and in-order chain stay green; under N12 the
older `TestLoadRefusesDetachedChainIndex` stays green (it passes via the
downstream continuity arm, which is why it never witnessed the binding
clause — the rebound test does).

Token-preserving-behavior-change mutants (C1, C2 keep their sentinel
tokens while bypassing the funnel; A2 keeps behavior through the alias)
are executed through the behavioral suite (green) plus the full-package
gate (red), never through the static checker alone.

Measured bound (no deletion mutant, by measurement, not by omission):
the load verify-error arm is behaviorally subsumed by the self-field
arm — `VerifyObjectIdentity` returns field `""` on every error path, so
deleting the error arm changes no observable outcome (the tampered plant
is then refused one line down). The row stays for diagnostic attribution
(a verify failure names its cause instead of misreporting a field
mismatch) and is pinned by the census, which requires the tampered-blob
plant to exercise exactly that line.

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

- After session directory: `*Fault(recoverable_parked_state)`; reads fail
  closed (`GetRecord` and whole-repository `ListSessions` name the parked
  state); identical retry resumes the create (record reads back,
  listing shows the one session).
- After record install: parked record-without-chain; retry completes the
  index through the resume path; the chain continues at position 0.
- After events directory: same parked shape; retry completes it.
- After chain index: crashed commit counts as committed; record reads
  back; retry names the terminal state (`ErrSessionExists`).

Resume path (production retry, crash-driven above plus hand plants):
record-without-chain completes on retry, then refuses as existing;
bare directory completes with the retry bytes; colliding bytes and
chain-without-record refuse without mutating. Reload across a second
`Open` handle is byte-identical and the chain continues there.

## Reuse (no owner duplicates, F5 correction)

Validation via `canonicaljson.VerifyObjectIdentity`; frame/member rules via
`environ.DecodeStrictObject`/`CheckUUIDv7`/`CheckUint53Bounds`;
grammars via `scalar.ParseDigest`/`ParseUUIDv7`/`ParseUUIDv4`;
crash vocabulary via `secconftest` points/outcomes/`Fault`;
census machinery via `invcore`. No new decoder, bound check, AST walker,
or journal. Correction to the rev1 claim: `syncDirectory`,
`writeAtomic`, `writeExclusive`, and `installEventBlob` DO duplicate
`localstore` path policy (staged write + dir-fsync, `O_EXCL` install,
content-addressed idempotent put). `localstore` has no production
consumer yet, so this leaf is not rewired through it — the duplication
is stated here with that reason instead of claimed away. Session IDs
(UUIDv7 grammar) need no traversal guard beyond the scalar owner.

## Stated bounds for downstream leaves

Single authoritative branch per session (same-epoch second lease is
preserved, never applied; arbitration is the leases story). No
`SessionState` derivation (reducer leaf `1r9wrr`). No peer names or
interactive choice (name-resolution leaf `21gygk`). No list rendering
(CLI leaf). Chain index is a verified cache; blobs are truth.
Concurrent racing creates serialize on `O_EXCL` plus the resume guards
plus the atomic chain index — witnessed deterministically through the
blob path and the fresh-handle test, not through a timing race (a
flaky timing test would prove less than the deterministic mechanism
witness). The `AfterCreateStep` hook is test surface (nil in normal
operation), like `BeforeWrite`/`AfterCommit` before it.

