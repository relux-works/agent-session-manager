# BUG-260917-3lddu0 — independent review, CR revision 3

Reviewer: RUN-260922-573ca5. Candidate TREE: `3306cab83d2eece38ffc3ac43c87445e57993211`. CR base and independently advertised remote main: `799c338e401fca0b24859c0b870cd665927209f2`. Live Story HEAD remains sibling checkpoint `964fa472c97569fb209e5bdf831d42e2484ef078`. The CR patch is byte-identical to `git diff base candidate`, SHA-256 `8f6102b79dfe5765774099438a804d0315100537ae10edfaa9f74e77d982e465`. All archived candidate blobs and all live candidate files matched the immutable tree. No Story files, index, branch, or HEAD were changed by this review.

## Verdict

**ACCEPTED — record with `accept_cr(BUG-260917-3lddu0, revision=3)` and route to integrating.** No P1 or P2 remains. This is review acceptance, not integration or task completion. The independent derivation-side bound remains assigned as described below.

## Acceptance and scope

Normative authority: pinned SPEC.v0.7.0 §2.4, especially line 666, and §5.2/§5.3. The final rework expressly allowed route (b), which closes this leaf through append admission and assigns independent derivation authority elsewhere. This review does not turn that limited result into an independent Derive guarantee.

The acceptance table actually contains **2 rows; 2 of 2 AC rows driven**:

| Row | Production path | Named candidate test and independent measurement |
| --- | --- | --- |
| Refuse a superseded lease even while the tail matches it | `sessrepo.Repository.AppendEvent → checkWinningLease` | `TestAppendEventRefusesSupersededLeaseWhileTailStillMatches`; adapted original probe 9 directly and through SetProfile, Emit, EmitParked; stale narrowing killed at all four entries |
| Losing profile event cannot become effective through this admission path | `AppendEvent`, then `sessprofile.Projector.Project → Derive`, and `axpane.Run → LoadProfile` | `TestLosingLeaseProfileEventIgnored`; adapted probe 15; profile-only admission narrowing restores the forbidden effective source and kills the test |

**8 of 10 gate × entry cells measured, 2 of 10 explicitly bounded.** The five columns are direct AppendEvent, Transactor.SetProfile, Emit, EmitParked, Run→EmitParked. Each of the first four has both lower-epoch and same-epoch-loser witnesses. Shipped mutant logs attribute kills to each named test, not only the package exit. Row-count symmetry: lower-epoch **4/4**, same-epoch-loser **4/4**; Run→EmitParked is bounded on both sides for the Observe/Append interleaving. Mechanical production caller search finds SetProfile and Emit directly calling AppendEvent; EmitParked composes Emit and Run composes EmitParked. No additional production caller was omitted. Historical replay intentionally does not invoke a new-event gate.

The existing sibling production change was inspected and its accepted rev3 evidence read. Its source remains checkpoint-identical; the whole Story candidate is covered by this review's full suite. Its earlier broad mutation battery is accepted as attached sibling evidence, not claimed as rerun here.

## Before, after, and independence

The original probe archive was read and verified as gzip (SHA-256 `9f473449f33fda2be591f23396da2b9b9b995ba18708b02afdff7a550e840ef4`). Its probe 9 predates the current Emit observation API. The attached adapted probes preserve the losing-writer vector and provide the required Presented/Observation inputs; they are not described as verbatim legacy-source runs.

On trunk, probe 9 admits the stale event through all four entries. Probe 15 admits losing A after successor A2 won, and Projector derives `Profile:yolo`, `HasSource:true`, with `--dangerously-bypass-approvals-and-sandbox`. Current Run returns parked on this fixture, so I do **not** claim that this current trunk run launched a process; the effective-profile and provider-mapping escalation was directly observed. On the candidate, the append returns ErrStaleLease, the chain remains unchanged, and Projector derives standard/no source. The dedicated candidate probe-9/15 run exits 0. The broad diagnostic run exits 1 because it intentionally also detects the disclosed higher-epoch bound; that exit is not relabeled green.

Two independent plants were executed twice:

- Narrow only `checkWinningLease` for `profile.changed`: the candidate's losing-profile test fails at its append assertion, and the downstream reviewer probe derives yolo/bypass again. There is no independent durable-lease-aware derivation defense in this leaf.
- Leave append admission intact and narrow `sessprofile.checkChain`'s stale refusal only for `profile.changed`: `TestDeriveRefusesLosingLeaseAndAmbiguity/stale_epoch` fails with nil error; the task's append-gated profile test still passes. The pre-existing derivation guard covers chain-relative stale inputs, a different property.

Both the never-minted higher-epoch profile event and empty-lease-store class remain admitted on trunk and candidate. The current results/matrix/README/TRACEABILITY explicitly assign independent authority, including those classes, to **STORY-260922-cpkajd / TASK-260922-31qyyi**; both IDs were read from the board and exist in backlog. The old reused diagnostic's “no owner named” suffix is obsolete; it is not this review's finding.

Winning-writer probes independently drive Emit, SetProfile, identical SetProfile retry, and EmitParked successfully on both trees. They establish that legitimate composing writers still operate.

## Rework dispositions

| Prior finding | Disposition |
| --- | --- |
| P2-1 missing section/clause edge | Closed: one case definition plus real ownership-section and `2.4#2` references; production owner is AppendEvent. Renaming the registered test makes tracecheck refuse twice. |
| P2-2 no tracked independent-authority owner | Closed by the two verified IDs above and their explicit bounds in all required artifacts. No out-of-scope implementation was added. |
| P3 misleading status-grid label | Closed: counts now say test-status keys, and the runtime outcome grid is separately adopted and reproduced by this reviewer. |

Four sibling P3 dispositions: (1) axpane doc-comment qualifier is explicitly deferred/left untouched, owned by the next axpane documentation edit; it is not claimed fixed. (2) observeRemoteWinner structural redundancy is recorded only, with no simplification. (3) no-grant §4.2 step-4 remains an unresolved product question for grant-minting work; no decision is smuggled into this change. (4) results/matrix name the actual candidate TREE OID and provide test masks beside counts.

Nonblocking wording note: TRACEABILITY's introductory “must be at the winning epoch” is broader than checkWinningLease, which also admits higher epochs. Its adjacent explicit higher-epoch bound and tracked owner define the reviewed behavior; no complete higher-epoch refusal is credited here. The existing “lapsed grants refuse” axpane comments likewise still require the local-owner qualifier. These are recorded P3 wording debts, not reopened production or architecture work.

## Adversarial evidence

Shipped harness executed twice on isolated copies, with exact source restoration. Both lower-epoch and same-epoch narrowing mutants are KILLED twice (exit 1); comment control SURVIVED twice (exit 0); missing replacement is NOT_APPLIED and compilation control is COMPILE_OR_HARNESS_FAILURE, neither counted as a kill. Controls before/after exit 0. Every gate cell credited above has its own failing test in the logs.

Reviewer behavioral mask for the following table:
`go test ./internal/sessrepo ./internal/sessprofile ./internal/axpane -count=1 -v -run 'Test(AppendEventChainInOrder|AppendEventRefuses|SetProfileRefuses|EmitReaches|EmitParkedReaches|Mint|Change|Confirmed)'`.

| Reviewer plant | Production gate attacked | Result in both runs / named killer |
| --- | --- | --- |
| R9 one skipped sequence admitted | checkAppend continuity | KILLED, exit 1; TestAppendEventRefusesSequenceGap |
| R10 equal-length disagreeing blob admitted | installEventBlob integrity | KILLED, exit 1; TestAppendEventRefusesSameLengthDisagreeingBytes |
| R11 same-epoch preservation only for profile events | AppendEvent preservation arm | KILLED, exit 1; direct same-epoch test and TestEmitParkedReachesSameEpochAppendAdmissionGate |
| R12 sequence-2 unconfirmed yolo admitted | MintChangeEvent confirmation | KILLED, exit 1; TestMintChangeEventRefusals/unconfirmed_yolo |
| R13 only stale session.parked admitted | checkWinningLease through alternative EmitParked entry | KILLED, exit 1; TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation |
| Harmless comment | control | SURVIVED, exit 0 |

**5/5 reviewer narrowings killed twice, 1/1 applied harmless control survived twice.** First four attack gates not covered by the two producer admission plants; R13 attacks another entry while profile events stay protected. These weaken gates rather than delete them. Raw per-plant logs, edits, subprocess exits and restoration census are attached. An initial full-package reviewer plant attempt was deliberately interrupted (exit 130) to bound repeated work, restored its source, and was replaced by the stated focused mask. That incomplete attempt is retained and credited with no result.

The production admission gate does not inspect source text. Supplemental registry-test disabling and README numeric corruption attacks both refuse twice; the README behavioral pin executes rather than merely grepping text. The admission/derive mutants preserve the refusal tokens and are run through behavioral tests.

## Registry and story-final traceability

Re-derived from candidate registry decode + canonical JSON projection:
`174728a8a01e8429004e4f9c4687f8fa1f17406b641012236c3427a758baf31e`, equal to reviewedOwnershipCanonicalSHA256. Registry raw SHA-256: `da91e71a9fb592c434b0d87a9367cd8a043f191f89e0a1d2a031640e4ee8afb1`. No baseline acceptance case was removed; remote main is still the CR base, so there is no refresh delta to hide a foreign Story revert.

Exact-tree output, quoted verbatim:

```
traceability ok: contracts=64 normative_sections=36 acceptance_cases=154 fixtures=33 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=69 full=4 partial=9 sliver=9 unevidenced=43 unmeasured=4 unowned=7 clauses_discharged=70/574
```

README's measured-coverage fenced line is equal. Changing 70 to 71 makes TestREADMEMeasuredCoverageMatchesTracecheckReport fail twice. Section 2.4 exits 0. Section 5.3 exits **1**, explicitly expected for the existing **7/8** partial binding; it is not reported as a passing section. The global coverage number is inherited, not a claim that this leaf implemented independent lease-aware derivation.


## Complete importer runtime comparison

Mechanically derived Imports/TestImports/XTestImports mask:
`go test -json -count=1 ./internal/axpane ./internal/crashgate ./internal/fencing ./internal/sessckpt ./internal/sessprofile ./internal/sessquery ./internal/sessstate ./internal/termbind ./internal/terminstance`.
Both baseline and candidate exit 0, **9/9 package summaries PASS**. Instrumentation records actual AppendEvent returns by `(executing package, event type, epoch, sequence, relation to durable winner, outcome)`, with no test-name key. Exact instrumentation bytes/patches are attached. `no-readable-winner` deliberately does not conflate a read failure with absence; that diagnostic label covers both without asserting which occurred.

| Measurement | Trunk 799c338 | Candidate 3306cab | Delta |
| --- | ---: | ---: | ---: |
| Outcome keys | 130 | 137 | +10 / −3 |
| Observed AppendEvent calls | 2,926 | 2,955 | +29 |
| Packages passing | 9 | 9 | 0 |

Both runtime grids exactly equal the previously attached rev2 grids, which the producer adopted. All 17 count-delta keys are in logs/runtime-delta.json. Moved classes: losing lower-epoch profile.changed ADMIT→STALE; newly measured same-epoch profile.changed and session.parked return DIVERGENT; stale session.parked returns STALE. The three old-owner stop fixtures now execute session.idle/2 and session.stopped/3 before takeover (same-winner), instead of admitting lower-epoch events after takeover. Their unchanged test names are explicitly documented, with §5.2/§5.3 reasoning, not hidden by PASS statuses. New higher-epoch/no-store setup events remain admitted as the disclosed bounds; ordinary created/failed fixture counts grow. Seven other importer packages have no outcome count delta; terminstance runs its tests but emits zero AppendEvent calls in this mask.

The changed landed tests were read: TestRunPostWindowSupersedes, TestRunSupersededPairIdenticalRetry, TestRunCreateFromStoppedPostWindow now checkpoint before successor CAS; TestLosingLeaseProfileEventIgnored now asserts the refusal before checking the remaining profile authority. Their semantic arguments match the gate and specification. No unargued fixture rewrite was found.

## Validation, durability, and handoff evidence

**30 of 30 configured commands exited 0**, read from the candidate config. Raw logs and a complete command/exit/time/load table are in logs/configured-validation-summary.json. This reviewer reran all 30; no producer full-suite result is substituted. The 17 fuzz lanes use the configured 100x/parallel=1 settings. Host platform is macOS arm64, Go 1.25.5; Linux and Windows builds are explicitly configured by this Go project. Additional Windows amd64 vet exits 0.

| Gate | Exit | Elapsed | Load before / after (1 minute) |
| --- | ---: | ---: | ---: |
| `go test ./... -count=1 -v` | 0 | 431.9s | 6.7 / 18.3 |
| `go test ./... -race -count=1 -timeout 25m` | 0 | 660.5s | 12.4 / 13.5 |
| `go test ./... -cover -count=1` | 0 | 214.8s | 15.3 / 13.5 |

The race timeout stayed 25m. It actually took 660.5s; its persistent tool call was awaited through completion and never abandoned. The other full checks and mutation batches were separate bounded calls. No test timeout or race failure occurred. Board validation exits 0 with existing MISSING_ACTIVITY diagnostics; those notices are not called newly repaired. Archive equivalents enumerate immutable files for gofmt/JSON, and command 30 checks the exact base-to-candidate delta, avoiding accidental enumeration of the parent checkout. All remaining commands use the configured invocation.

The reviewer additionally repeated stale and same-epoch losing appends and checked refusal idempotency, preserved blob bytes, blob count and unchanged authoritative chain (exit 0). Existing crash suites ran in the full checks; the new refusal arm composes the existing immutable blob installer rather than inventing a new durability mechanism. No new crash-boundary claim beyond those executed paths is made.

Zero __pycache__ or .pyc in the review tree. Live candidate remains byte-identical and uncommitted at checkpoint HEAD. The complete live checklist was read, with no unfinished item. Supplemental diagnostic runs intentionally exit 1 for defeated gates/known bounds; compile and NOT_APPLIED controls and the interrupted exploratory attempt are never counted as green tests.

Evidence archive: BUG-260917-3lddu0_review-evidence-rev3.tar.gz, real gzip, with SHA-256 manifest, immutable identifiers, exact test masks, per-plant raw outputs and subprocess exits. Reused original/previous/sibling evidence is identified by references and digests, not represented as newly executed. This verdict and archive are attached and read back for byte comparison before accept_cr. No commit_ack, commit, checkpoint, integration, or done transition is performed by this reviewer.
