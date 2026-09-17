# TASK-260830-wbpf1v outcome: session repository and event chain (rev4)

## Result

`internal/sessrepo` implements durable Session Records and append-only
per-session event chains with source sequence continuity (AX v0.5.0
§2.3 local steps, §5.1–5.2, §5.7/§14.4 stored facts). Handbook handoff:
ready for review. Candidate tree OID (detached-index `read-tree HEAD` +
`git add -A` + `write-tree` on a scratch index, worktree untouched):

`7ab337c849fb1aa532927327d07329ea8f97825b`

This is revision 4, reworking revision 3 after the reviewer verdict
`changes_requested` (`TASK-260830-wbpf1v_review-verdict-rev3.md`): F1
equality census allegedly blind to a new-file plant and to a
differently-spelled comparison (repeat-of rev2/B1), F2 three stale
published numbers (tree OID, path count, 36-vs-35 refusal denominator,
the last shipped in `README.md`), F3 false `sessionDir` traversal claim
(non-blocking). The previous round-4 run died mid-work leaving two red
tests; this round resumes from that worktree state. Every finding below
was re-driven, not re-read.

## Changed paths (exactly 11, matches the CR)

- `LOGBOOK.md` (round-4 entry)
- `README.md` (sessrepo section: 35-site census with per-file split,
  equality census, 33/33 battery)
- `internal/provhost/identity.go` (attestation-bound comment, carried)
- `internal/provhost/identity_test.go` (allowlist gate, carried)
- `internal/sessrepo/sessrepo.go` (parked channel, carried)
- `internal/sessrepo/store.go` (per-session parked listing, carried)
- `internal/sessrepo/chain.go` (digest-arm regression fix, this round)
- `internal/sessrepo/doc.go` (parked decision record, carried)
- `internal/sessrepo/sessrepo_test.go` (N2 wording fix, carried vectors)
- `internal/sessrepo/crash_test.go` (parked-entry crash assertions, carried)
- `internal/sessrepo/census_test.go` (TestMain static/ runtime split,
  live-coupled plant tests, this round)

No untracked ungitignored scratch file is present (battery harness and
working logs live under gitignored `.temp/TASK-260830-wbpf1v/`; one
stray empty `internal/sessrepo/relative/` directory left by an earlier
run was removed; the battery log and this outcome are attached to the
board task).

## Reviewer findings → fix → witness

| ID | Finding | Fix (production path) | Witness |
|---|---|---|---|
| F1 | Equality census allegedly blind to P2 (new-file `bytes.Equal`) and P3 (`string==` spelling); named test did not fire | Two-part. (a) Diagnosis: the instrument was never blind — the two red tests below masked it, because TestMain skipped the whole audit unless the suite was green. TestMain now runs the static content-equality audit on every full unfiltered run, green or red; only the exercised-vs-derived refusal runtime audit stays gated on green (`census_test.go` TestMain). (b) The three plant tests that diffed synthetic sources against hard-coded ledger copies are now coupled to the live derived set and live ledger, so drift fails the sanity half first. | P2 live plant → exit 1 `sessrepo content-equality site without a registered vector: plantextra.go:6`; P3 live plant → exit 1 `... store.go:17` (both full unfiltered runs, logs quoted below). Battery C3/C4/A3 rows KILLED behaviourally-green + gate-red |
| F2a | Outcome named tree `93d4a106`, record is `08c8ceb6` | Re-derived by detached-index `write-tree` after the last edit; this outcome names `7ab337c8` (recompute, never re-type) | `git read-tree HEAD && git add -A && git write-tree` on scratch index `= 7ab337c849fb1aa532927327d07329ea8f97825b`; `ls-tree` shows all 7 new sessrepo files |
| F2b | Outcome enumerated 8 of 11 CR paths | Enumerated exactly above: README + LOGBOOK + 2 provhost + 7 sessrepo = 11 | `git diff --name-only HEAD` lists exactly these 11 |
| F2c | Refusal denominator published as 36 (outcome ×3, README ×1); derived is 35 | Fixed in place to 35 with the per-file split chain.go 25 / sessrepo.go 5 / store.go 5 | In-package probe: `DERIVED_SITES=35 per-file=map[chain.go:25 sessrepo.go:5 store.go:5] sentinels=14` |
| F3 | `sessionDir` comment claimed UUIDv7 grammar bars traversal; typo `traversaliedy` | Already corrected in the worktree (comment states the join is unguarded and the escape refuses at the record/session binding) with `TestReadEntriesRefuseTraversalSessionID` driving `../outside` through `GetRecord`/`GetEvent`/`ListEvents` | `TestReadEntriesRefuseTraversalSessionID` passes; healthy session still listable after |
| REG | Two tests red on the clean tree (`TestLoadRefusesSwappedEventBlobs`, `TestLoadRefusesSubstitutedFirstEventBlob`), no plants applied | Production regression, not tests-ahead-of-production: the round-4 edit had rewritten the digest arm as `digest != indexed && field != SelfEventID`, dead code since the field is proven two lines above. Rev3 shipped the bare digest comparison (verified in the rev3 patch); restored. | Both tests pass; N8 narrowing (`+ && field != SelfEventID`, the exact regression shape) KILLED by the swap test |
| N2 wording | Outcome/test said the canonical owner refuses "one arm earlier" | Fixed in `sessrepo_test.go`: member arms run before Verify, so the owner refuses one arm later | Comment only; mechanism re-verified by reviewer probe P4, restated correctly here |

## AC coverage: 10 of 10 rows driven through production entries

| # | AC row | Production call site | Named driving test(s) |
|---|---|---|---|
| 1 | Persist Session Records | `(*Repository).CreateSession` | `TestCreateSessionPersistsSpecRecordFixture`, `TestSessionRecordReloadsByteIdenticallyAcrossProcessBoundary` |
| 2 | Validate Session Records | `(*Repository).CreateSession` | `TestCreateSessionRefusesMalformedFrame`, `...ForeignSchema`, `...MissingSessionID`, `...DigestMismatch` |
| 3 | Appends succeed in order | `(*Repository).AppendEvent` | `TestAppendEventChainInOrder` |
| 4 | No rewrite/delete path | `CreateSession`/`AppendEvent` (no update/delete entry) + `writeExclusive` O_EXCL | `TestCreateSessionRefusesDuplicateAndRewrite`, `TestAppendEventRefusesSequenceRepeat`, `TestAppendEventRefusesDisagreeingBytesAcrossHandles` (fresh handle), `TestAppendEventRefusesSameLengthDisagreeingBytes` |
| 5 | Gap refused (named arm, at edge) | `checkAppend` via `AppendEvent` | `TestAppendEventRefusesSequenceGap` |
| 6 | Repeat refused (named arm) | `checkAppend` via `AppendEvent` | `TestAppendEventRefusesSequenceRepeat` |
| 7 | Exact contract fixtures pass | `CreateSession`/`AppendEvent` | SPEC §5.1 record, task_board record, SPEC §5.2 stopped event; SPEC §5.5 provider-identity pinned and refused on the load path |
| 8 | Negative/refusal cases pass; parked sessions reportable per session | all entries; `ListSessions`/`Resolve` | 35-site refusal census, every site boundary-driven; load rows have one plant each (tampered, provider-identity, swapped, substituted-first); equality census 2 sites = 2 ledger rows; parked channel: `TestListSessionsReportsParkedSessionPerSession`, `...KeepsHealthySessionsBesideParked`, `TestResolveSkipsParkedSessions` |
| 9 | Crash/idempotency evidence | `BeforeWrite`/`AfterCommit` + `AfterCreateStep` hooks | 8 crash tests: 4 outer + 4 interior create steps, each with point, outcome, fired-prefix, heal/terminal assertions |
| 10 | No unsupported capability | API surface (only `Repository` entries; no command/doctor/result wiring) | stated bound with surface evidence; README claims only what exists |

## Gates actually run (exit codes observed in this session, rev4 tree)

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `GOOS=windows go build ./...` | 0 |
| `gofmt -l` over tracked+untracked Go files | clean (exit 0) |
| `go run ./internal/traceability/cmd/tracecheck` | 0 (contracts=60, acceptance=98) |
| `go test ./... -count=1` | 0, all 24 packages ok |
| `go test ./internal/sessrepo/ -cover -count=1` | 0, 87.6% of statements |
| `go test -race` full repository in 4 groups (6+8+6+5 dirs, sessrepo in two) | 0 in all 4 groups, 24 unique packages ok |
| mutation battery rev4 `all` | 0 (33 applied, 33 killed, 0 survivors; controls distinct) |

Not run: full fuzz smoke over 13 targets and the catalog-freshness
generate check. Rationale: this change adds no fuzz target and modifies
no fuzzed package (`secconftest`, `canonicaljson`, `scalar` untouched)
and no catalog surface; both stay CI-lane activities on unchanged
surface.

## Census (production-derived denominators, invcore)

- Derived `refuse(` sites: **35** (sessrepo.go 5, chain.go 25, store.go 5).
  The load re-verification holds three separate rows — verify failure,
  non-`event_id` self field, index-disagreeing digest — each with its own
  boundary-driven plant (`...TamperedEventBlob`,
  `...ProviderIdentityBlobAtEventPath`, `...SwappedEventBlobs`, plus
  `...SubstitutedFirstEventBlob` for position 0).
- Exercised beneath a boundary entry: **35 of 35**; exercised-outside: 0.
- Sentinel roster derived from `Err*` vars (14/14); observed codes match.
- Alias audit (`refuse` + 8 watched owner delegations) clean; unrouted
  `fmt.Errorf`-with-sentinel audit clean.
- Refusal control plants: DiffSets unregistered site, orphan row,
  zero-site derivation; alias audit with import-alias direct-call
  (clean), import-alias var binding (fails), local var binding (fails),
  dot import (fails), shadowed name (fails); unparseable plant (fails).
- Content-equality census: **2 derived sites = 2 ledger rows**
  (`sessrepo.go:258` resume, `chain.go:426` install; E1 qualified
  `bytes.Equal` resolved by import path, E2 `string(...) ==` conversion
  form). Static audit runs on every full package run, green or red.
  Red-before, full unfiltered run, real exit codes:
  - P2 `bytes.Equal` in new file `internal/sessrepo/plantextra.go` →
    exit 1, `sessrepo content-equality site without a registered
    vector: plantextra.go:6`
  - P3 `string(left) == string(right)` in `store.go` → exit 1,
    `sessrepo content-equality site without a registered vector:
    store.go:17`
  - Named tests `TestEqualityCensusFailsClosedOnNewFileSite` /
    `...OnStringEqualitySpelling` / `...OnOrphanRow` couple to the live
    derived set and live ledger (sanity half fails on drift before the
    plant half runs); var-binding, dot-import, and unclassifiable-spelling
    (incl. import-alias `be.Compare`, `reflect.DeepEqual`) plants fail the
    alias/unclassifiable audits. Stated blind spot, unchanged: a
    hand-rolled comparison loop carries no classified call and this census
    would not see it.

## Mutation battery (production-derived denominator: 35 sites + mechanism targets)

33 applied, **33 killed**, 0 survivors (16 narrowing, 10 arm-deletion,
4 census-only, 3 audit-only; NOT_APPLIED and COMPILE_FAIL separate).
Harness: `.temp/TASK-260830-wbpf1v/mutation_battery_rev4.sh` (gitignored
scratch) with exact-count application (any miscount aborts the battery)
and checksum restore after every mutant. Full log:
`TASK-260830-wbpf1v_mutation-battery-rev4.log` on the board task.

| Mutant | What it narrows/admits | Named killing test |
|---|---|---|
| N1 `> want` → `> want+1` | gap arm admits gap-of-1 | `TestAppendEventRefusesSequenceGap` |
| N2 `< want` → `+1 < want` | repeat arm admits exact repeat | `TestAppendEventRefusesSequenceRepeat` |
| N3 `!= 1` → `< 1` | first-link admits extra predecessors | `TestAppendEventRefusesFirstPredecessorMismatch` |
| N4 contains-tail → non-empty | predecessor arm admits wrong link | `TestAppendEventRefusesLaterPredecessorMismatch` |
| N5 record gate admits session-event | record gate admits one foreign class member | `TestCreateSessionRefusesForeignSchema` |
| N6 fold A–Z → A–T | ambiguity gate admits U–Z collisions (vector carries Y) | `TestResolveRefusesCaseFoldCollision` |
| N7 field arm exempts `record_id` | load admits provider-identity blobs | `TestLoadRefusesProviderIdentityBlobAtEventPath` |
| N8 digest arm `+ && field != SelfEventID` | load admits blob substitution (exact round-4 regression shape) | `TestLoadRefusesSwappedEventBlobs` |
| N9 `O_EXCL` → `O_TRUNC` | install admits disagreeing bytes | `...DisagreeingDigestPathBytes` + `...AcrossHandles` |
| N10 byte equality → length equality | resume admits same-length collision | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| N11 binding drops session half | load admits cross-session record | `TestLoadRefusesMismatchedSessionBinding` |
| N12 binding drops digest half | load admits detached index | `...DetachedChainIndexWithReboundPredecessors` |
| N13 blob equality → length equality | install admits same-length forgery | `TestAppendEventRefusesSameLengthDisagreeingBytes` |
| N14 parked entry dropped (skip) | listing hides the parked session | parked listing tests |
| N15 `!Parked` → `true` | Resolve admits parked siblings | `TestResolveSkipsParkedSessions` |
| N16 digest arm exempts position 0 | load admits first-blob substitution | `TestLoadRefusesSubstitutedFirstEventBlob` |
| D1 delete gap arm | — | `TestAppendEventRefusesSequenceGap` |
| D2 delete repeat arm | — | `TestAppendEventRefusesSequenceRepeat` |
| D3 delete divergent arm | — | `TestAppendEventPreservesDivergentBranchWithoutApplying` |
| D4 delete idempotent-retry branch | — | `TestAppendEventIdempotentRetry` |
| D5 delete resume chain-present guard | — | `TestCreateSessionRefusesDuplicateAndRewrite` |
| D6 delete load self-field arm | — | `TestLoadRefusesProviderIdentityBlobAtEventPath` |
| D7 delete load digest arm | — | `TestLoadRefusesSwappedEventBlobs` |
| D8 delete record/chain binding | — | binding tests |
| D9 delete resume byte-equality effect | — | `TestCreateSessionRefusesResumeWithDifferingBytes` |
| D10 delete torn-state guard | — | `TestCreateSessionRefusesResumeOfChainWithoutRecord` |
| C1 funnel bypass, token preserved (write path) | behavioural green, gate red | full run fails on unrouted funnel bypass |
| C2 funnel bypass, token preserved (load path) | behavioural green, gate red | full run fails on unrouted funnel bypass |
| C3 new-file `bytes.Equal` (reviewer P2 shape) | behavioural green, gate red | full run fails on unregistered plantextra.go site |
| C4 `string==` spelling (reviewer P3 shape) | behavioural green, gate red | full run fails on unregistered string-== site |
| A1 unreachable extra refuse site | behavioural green, gate red | full run fails on unregistered site |
| A2 var-bound funnel | behavioural green, gate red | full run fails on alias audit |
| A3 var-bound `bytes.Equal` | behavioural green, gate red | full run fails on equality alias audit |
| X1 NOT_APPLIED | harness control: absent pattern, checksums unchanged | — (distinct row) |
| X2 COMPILE_FAIL | harness control: broken tree fails the build | — (distinct row) |

Survivors: none. Stated subsumption bounds (no named failing test by
construction, carried from rev3 and re-verified by the reviewer): the
load verify-error arm is behaviourally subsumed by the self-field arm
(Verify returns field `""` on every error path); the six bad-member
vectors are refused by the canonical owner one arm later with the same
sentinel (member arms run before Verify).
