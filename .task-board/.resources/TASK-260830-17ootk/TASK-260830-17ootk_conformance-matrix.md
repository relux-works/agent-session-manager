# TASK-260830-17ootk conformance matrix (board deliverable)

Task: TASK-260830-17ootk fault-test-local-crash-boundaries (story-final leaf of
STORY-260830-2rqigd). What follows is the shipped
`internal/crashgate/TRACEABILITY.md` verbatim, plus the evidence index below
it. The registry JSON bindings (`section:13.12`, `section:13.13`), the two new
acceptance cases, and the re-pinned `reviewedOwnershipCanonicalSHA256` are the
reviewed registry counterparts of this matrix.

---
# `internal/crashgate` traceability (TASK-260830-17ootk)

Authority: `relux-works/agent-session-manager-spec@v0.6.0`
(`internal/specpin/v0.6.0.lock.json`, embedded
`internal/specdoc/SPEC.v0.6.0.md`). The task scope cites the same
section numbers of v0.5.0; the headings are retained in v0.6.0.

Production entry points: none of its own. The harness drives the
landed owners only: `sessckpt.Store.Capture` (checkpoint-capture
boundaries) and `matjournal.Store.Create` with its journal updates
plus `matjournal.Store.Recover` as the recovery evaluator (journal
boundaries). The package's own production file, `registry.go`, is
pure classification data (boundary IDs, per-path reachability,
drivers, owners) with lookup accessors; it performs no I/O and
reads no specification text. No CLI surface is offered.

## AC coverage (4 of 4 rows driven through the production entries)

| # | AC row | Production call site | Named tests |
| --- | --- | --- | --- |
| 1 | Crash after every durable boundary; prove exactly `safe_retry`, `explicit_rollback`, or `recoverable_parked_state` | `Capture` (blob/receipt seams) + `Create`/updates/`Recover` (journal seams) | `TestCrashGateConformance` (72 rows over 48 reachable paths), `TestRealSIGKILLSeams` (2 seams), `TestCrashGateMutualExclusion` (exclusivity/exhaustiveness) |
| 2 | Exact contract fixtures and negative/refusal cases | `Recover` rejection gates | `TestCrashGateRejections` (10 rows: two live authorities, unfenced continuation x2, substitution x4, moved lease x3), `TestCrashGateMutualExclusion` (12 rows: contradiction x2, missing x2, marker x2, unknown floors x2, torn, safe_retry writes nothing, terminal replay/park) |
| 3 | Crash/idempotency evidence for durable mutations | `Capture` receipt replay + `Create` prepare-receipt replay + `Recover` evidence | `TestCrashGateConformance` (every row writes a machine-readable record), `TestCrashGateSection1312` (8 rows), `TestRealSIGKILLSeams` |
| 4 | No unsupported capability advertised | no `cmd/` surface, no provider/task-board I/O, probes stay modeled | Stated bound (see below); `gofmt`/`go vet`/`GOOS=windows go vet` clean |

## Pinned-clause map

### Section 13.13 Crash/restart outcome gate (9 of 11 clauses discharged, `partial`)

| Clause | Content | Evidence |
| --- | --- | --- |
| `13.13#1` | inject a crash and a clean restart at every boundary for each applicable path | conformance table injects at every reachable path (checkpoint/journal-observable); NOT APPLICABLE paths name owners below |
| `13.13#2` | the evaluator classifies into exactly one outcome | `Recover` via `TestCrashGateConformance` + `TestCrashGateMutualExclusion`; `mustOutcome` enforces the three-outcome enum at every call |
| `13.13#3` | `safe_retry` meaning (same IDs, byte-identical input, unchanged lease/identity, reconcile-before-reissue, no second allocation) | same-ID/byte-identical replay in every row; unchanged lease in every record; uncertain-effect parks; no-second-manager proof in `owner_resume_lost_finalizes_same_ids` and the rejection rows |
| `13.13#4` | `explicit_rollback` meaning | NOT discharged: journal-durable halves proved (`rollback_failed`, `import_fails_before_lease_rolls_back`), event/CLI/status/audit visibility belongs to unlanded owners |
| `13.13#5` | `recoverable_parked_state` meaning | NOT discharged: fail-closed halves proved (frozen phase, durable reason, no second allocation), lifecycle-projection and status/doctor surfaces belong to unlanded owners |
| `13.13#6` | missing/stale/contradictory/unreachable/ambiguous evidence selects parked | contradiction, missing, unknown-floor, and torn rows in `TestCrashGateMutualExclusion` |
| `13.13#7` | no fourth outcome, no unclassified successful restart | `mustOutcome` enum at every call; `safe_retry_writes_no_new_evidence`; terminal replay/park rows |
| `13.13#8` | conformance record contents | `mustRecordComplete` on all 72 rows + 2 SIGKILL records; records ship in the evidence tarball |
| `13.13#9` | reject two live authorities | `TestCrashGateRejections/two_live_authorities` |
| `13.13#10` | losing/unfenced continuation fenced or parked | `unfenced_continuation`, `unfenced_state_liveness` (parked; fencing execution unlanded) |
| `13.13#11` | reject substitution (new launch, fresh handle, blank, realm) | `new_session_launch`, `fresh_native_handle`, `blank_relabel`, `different_realm` |

### Section 13.12 Failure matrix (7 of 19 rows driven; unmeasured by the scanner)

| Failure row | Disposition |
| --- | --- |
| SSH disconnect during transfer | NOT APPLICABLE: sync transfer flow (Section 13.3; unlanded) |
| Invalid digest/schema | Driven: `torn_journal_parks` quarantines torn bytes; capture-side torn refusal stays owned by `sessckpt` |
| Workspace divergence | NOT APPLICABLE: workspace divergence handling (unlanded) |
| Provider quiesce cannot prove idle | NOT APPLICABLE: provider quiescence observation (live proof unlanded) |
| Provider exits but handles open | NOT APPLICABLE: provider process runtime (unlanded) |
| Destination disk full | Driven: `disk_full_enospc_at_seam` (real ENOSPC at the write seam) + `disk_full_readonly_data` (genuine filesystem fault); both park with the free-space remediation |
| Atomic rename blocked | Driven: `rename_blocked_probe_parks` + `rename_blocked_real_fault` (refused write leaves bytes unchanged; retry converges) |
| Plugin crash/invalid stdout | NOT APPLICABLE: provider host with `doctor` (unlanded surfaces) |
| Epoch-1 launch fails before first checkpoint | Stated bound: the capture seam is proved (`CR-STOP-02`, `CR-GRACE-04/08`), but the epoch-1 bootstrap-abort classification belongs to the lifecycle story |
| Task-board bridge missing/incompatible | NOT APPLICABLE: task-board bridge runtime (Section 9; unlanded) |
| Import/open/adopt fails before new lease | Driven: `import_fails_before_lease_rolls_back` (failed + token-expired) and `CR-MAT-05/rollback_failed`; old owner stays authoritative (lease unchanged) |
| Adopt fails after new lease | Driven: `adopt_fails_after_lease_stays_stopped_owner` (safe retry of adopt/resume, no rollback past activation) |
| Owner-resume response lost | Driven: `owner_resume_lost_finalizes_same_ids` (prepared journal preserved, no second manager, same IDs) |
| Antigravity UUID unresolvable | NOT APPLICABLE: provider native identity (unlanded) |
| Muse background unfenceable | NOT APPLICABLE: provider adapter (unlanded) |
| Concurrent force takeovers | NOT APPLICABLE: lease arbitration (unlanded) |
| Old owner reconnects after force | NOT APPLICABLE: lease arbitration (unlanded) |
| SQLite index corrupt | NOT APPLICABLE: localstore projection with lifecycle rebuild (restore flow unlanded) |
| Operator interrupt | Driven: `operator_interrupt_preserves_journal` (journal preserved, authority unchanged, same operation retries) |

## Boundary matrix (31 of 94 IDs reachable; 48 of 169 paths driven)

Generated from `Registry()` by `cmd/matrixdump` (removed after the
run); the derivation test re-verifies the ID set against the pinned
text on every run. `driven (driver)` rows name the conformance
driver; `NOT APPLICABLE` rows name the exact owner.

| Boundary | Phase | Path | Disposition | Driver / owner |
| --- | --- | --- | --- | --- |
| `CR-LAUNCH-D-01` | session/lease/event persistence | direct | NOT APPLICABLE | sessrepo session/lease/event persistence (Section 13.1 bootstrap; out of checkpoint/journal story boundary) |
| `CR-LAUNCH-D-02` | terminal creation | direct | NOT APPLICABLE | terminal lifecycle runtime (terminalbackend manifest/probe/conformance landed; creation runtime unlanded) |
| `CR-LAUNCH-D-03` | provider process start | direct | NOT APPLICABLE | provider plugin/process runtime (Section 7/13.1; process start unlanded) |
| `CR-LAUNCH-D-04` | Provider Identity persistence | direct | NOT APPLICABLE | provider identity persistence (canonical shape landed; durable persistence runtime unlanded) |
| `CR-LAUNCH-D-05` | capture/object validation | direct | NOT APPLICABLE | direct bootstrap validation (Section 13.1; validation gate unlanded) |
| `CR-LAUNCH-TB-01` | session/lease/event persistence | task_board | NOT APPLICABLE | sessrepo session/lease/event persistence (Section 13.2 bootstrap; out of checkpoint/journal story boundary) |
| `CR-LAUNCH-TB-02` | bridge launch may have returned a manager reference | task_board | NOT APPLICABLE | task-board bridge runtime (Section 9/13.2; launch execution unlanded) |
| `CR-LAUNCH-TB-03` | bridge export may have returned a bundle/proof | task_board | NOT APPLICABLE | task-board bridge runtime (Section 9/13.2; export execution unlanded) |
| `CR-SYNC-01` | inventory exchange | direct | NOT APPLICABLE | sync transfer flow (Section 13.3; inventory exchange unlanded) |
| `CR-SYNC-01` | inventory exchange | task_board | NOT APPLICABLE | sync transfer flow (Section 13.3; inventory exchange unlanded) |
| `CR-SYNC-02` | immutable union | direct | NOT APPLICABLE | sync transfer flow (Section 13.3; immutable union unlanded) |
| `CR-SYNC-02` | immutable union | task_board | NOT APPLICABLE | sync transfer flow (Section 13.3; immutable union unlanded) |
| `CR-SYNC-03` | missing-object selection | direct | NOT APPLICABLE | sync transfer flow (Section 13.3; missing-object selection unlanded) |
| `CR-SYNC-03` | missing-object selection | task_board | NOT APPLICABLE | sync transfer flow (Section 13.3; missing-object selection unlanded) |
| `CR-SYNC-04` | staged chunk progress | direct | NOT APPLICABLE | sync transfer staging (Section 13.3; staged bytes unlanded) |
| `CR-SYNC-04` | staged chunk progress | task_board | NOT APPLICABLE | sync transfer staging (Section 13.3; staged bytes unlanded) |
| `CR-SYNC-05` | immutable-object commit | direct | NOT APPLICABLE | sync transfer flow (Section 13.3; object commit unlanded) |
| `CR-SYNC-05` | immutable-object commit | task_board | NOT APPLICABLE | sync transfer flow (Section 13.3; object commit unlanded) |
| `CR-SYNC-06` | passive materialize.prepare/commit | direct | NOT APPLICABLE | passive sync is a dormant-replica (task-board staging) plan; no direct plan reaches this phase |
| `CR-SYNC-06` | passive materialize.prepare/commit | task_board | driven (journal-dormant) | dormant prepare/commit through the journal-create seam, evaluated with the passive_sync path |
| `CR-SYNC-07` | dormant finalize | direct | NOT APPLICABLE | passive sync is a dormant-replica (task-board staging) plan; no direct plan reaches this phase |
| `CR-SYNC-07` | dormant finalize | task_board | driven (journal-dormant) | dormant finalize through the journal-finalize seam, evaluated with the passive_sync path |
| `CR-MAT-01` | before journal creation | direct | driven (journal-create) | crash before the first durable byte; identical retry creates exactly one journal |
| `CR-MAT-01` | before journal creation | task_board | driven (journal-create) | crash before the first durable byte; identical retry creates exactly one journal |
| `CR-MAT-02` | after journal/prepare receipt | direct | driven (journal-create) | crash between journal install and prepare receipt, and after the receipt; replay converges |
| `CR-MAT-02` | after journal/prepare receipt | task_board | driven (journal-create) | crash between journal install and prepare receipt, and after the receipt; replay converges |
| `CR-MAT-03` | after transfer | direct | driven (journal-transfer) | staged chunk facts survive; resume requests only absent indexes under the same IDs |
| `CR-MAT-03` | after transfer | task_board | driven (journal-transfer) | staged chunk facts survive; resume requests only absent indexes under the same IDs |
| `CR-MAT-04` | after validation | direct | driven (journal-validate) | validation-phase facts survive the restart |
| `CR-MAT-04` | after validation | task_board | driven (journal-validate) | validation-phase facts survive the restart |
| `CR-MAT-05` | after provider prepare or bridge import | direct | NOT APPLICABLE | workspace-only direct plan carries no provider transaction or bridge; the prepare-op phase is skipped |
| `CR-MAT-05` | after provider prepare or bridge import | task_board | driven (journal-prepare-op) | provider half (prepared token durable) and bridge half (imported state durable); failed-import rollback variant included |
| `CR-MAT-06` | after bridge open or host commit enters prepared | direct | driven (journal-to-prepared) | host half: validating-to-prepared transition durable under a workspace-only closure |
| `CR-MAT-06` | after bridge open or host commit enters prepared | task_board | driven (journal-to-prepared) | bridge half (opened state durable) and host half (prepared journal durable) |
| `CR-MAT-07` | after owner activation may have occurred | direct | NOT APPLICABLE | workspace-only direct plan carries no provider activation or bridge adopt; the activation phase is skipped |
| `CR-MAT-07` | after owner activation may have occurred | task_board | driven (journal-activation) | adopt recorded with agreeing status continues; unreachable status parks; never guesses |
| `CR-MAT-08` | after finalize/cleanup may have occurred | direct | driven (journal-finalize) | direct finalize has no external effect in the plan: resume under the same IDs |
| `CR-MAT-08` | after finalize/cleanup may have occurred | task_board | driven (journal-finalize) | resumed state durable completes the commit; unproven finalize parks with the blocking reason durable |
| `CR-GRACE-01` | after prepare | direct | driven (journal-create) | takeover prepare through the journal-create seam, evaluated with the graceful_takeover path |
| `CR-GRACE-01` | after prepare | task_board | driven (journal-create) | takeover prepare through the journal-create seam, evaluated with the graceful_takeover path |
| `CR-GRACE-02` | after input block | direct | NOT APPLICABLE | input gating runtime (provider/terminal input block; Section 13.6 flow unlanded) |
| `CR-GRACE-02` | after input block | task_board | NOT APPLICABLE | input gating runtime (provider/terminal input block; Section 13.6 flow unlanded) |
| `CR-GRACE-03` | after quiescence proof | direct | NOT APPLICABLE | provider quiescence observation (provhost proof decode landed; live idle observation unlanded) |
| `CR-GRACE-03` | after quiescence proof | task_board | NOT APPLICABLE | provider quiescence observation (provhost proof decode landed; live idle observation unlanded) |
| `CR-GRACE-04` | after checkpoint capture | direct | driven (checkpoint-capture) | takeover checkpoint through the capture blob/receipt seam |
| `CR-GRACE-04` | after checkpoint capture | task_board | driven (checkpoint-capture) | takeover checkpoint through the capture blob/receipt seam |
| `CR-GRACE-05` | after final sync | direct | NOT APPLICABLE | sync transfer flow (Section 13.3/13.6; final sync unlanded) |
| `CR-GRACE-05` | after final sync | task_board | NOT APPLICABLE | sync transfer flow (Section 13.3/13.6; final sync unlanded) |
| `CR-GRACE-06` | after destination validation | direct | NOT APPLICABLE | lifecycle orchestration (Section 13.6 destination validation; flow unlanded) |
| `CR-GRACE-06` | after destination validation | task_board | NOT APPLICABLE | lifecycle orchestration (Section 13.6 destination validation; flow unlanded) |
| `CR-GRACE-07` | after source stop | direct | NOT APPLICABLE | provider/bridge stop runtime (Section 13.6/13.9; stop execution unlanded) |
| `CR-GRACE-07` | after source stop | task_board | NOT APPLICABLE | provider/bridge stop runtime (Section 13.6/13.9; stop execution unlanded) |
| `CR-GRACE-08` | after closure-delta checkpoint | direct | driven (checkpoint-capture) | closure-delta checkpoint through the capture blob/receipt seam |
| `CR-GRACE-08` | after closure-delta checkpoint | task_board | driven (checkpoint-capture) | closure-delta checkpoint through the capture blob/receipt seam |
| `CR-GRACE-09` | after destination prepared materialization | direct | driven (journal-to-prepared) | destination prepared journal, evaluated with the graceful_takeover path |
| `CR-GRACE-09` | after destination prepared materialization | task_board | driven (journal-to-prepared) | destination prepared journal, evaluated with the graceful_takeover path |
| `CR-GRACE-10` | after destination lease persistence/union | direct | NOT APPLICABLE | lease arbitration beyond sessrepo persistence (lease persistence/union unlanded) |
| `CR-GRACE-10` | after destination lease persistence/union | task_board | NOT APPLICABLE | lease arbitration beyond sessrepo persistence (lease persistence/union unlanded) |
| `CR-GRACE-11` | after direct resume or task-board activation | direct | NOT APPLICABLE | provider resume runtime (Section 13.6 direct resume; execution unlanded) |
| `CR-GRACE-11` | after direct resume or task-board activation | task_board | driven (journal-activation) | task-board activation through the journal-activation seam, evaluated with the graceful_takeover path |
| `CR-GRACE-12` | after finalize | direct | driven (journal-finalize) | direct finalize has no external effect in the plan: resume under the same IDs |
| `CR-GRACE-12` | after finalize | task_board | driven (journal-finalize) | task-board finalize through the journal-finalize seam, evaluated with the graceful_takeover path |
| `CR-GRACE-13` | after source park/resume-event publication | direct | NOT APPLICABLE | lifecycle orchestration with sessrepo event publication (Section 13.6; flow unlanded, out of story boundary) |
| `CR-GRACE-13` | after source park/resume-event publication | task_board | NOT APPLICABLE | lifecycle orchestration with sessrepo event publication (Section 13.6; flow unlanded, out of story boundary) |
| `CR-FORCE-01` | after refresh/pinned-cohort selection | direct | NOT APPLICABLE | lifecycle orchestration (Section 13.7 refresh/pinned-cohort selection; flow unlanded) |
| `CR-FORCE-01` | after refresh/pinned-cohort selection | task_board | NOT APPLICABLE | lifecycle orchestration (Section 13.7 refresh/pinned-cohort selection; flow unlanded) |
| `CR-FORCE-02` | after divergence preservation | direct | NOT APPLICABLE | workspace divergence handling (Section 13.7/13.12; preservation runtime unlanded) |
| `CR-FORCE-02` | after divergence preservation | task_board | NOT APPLICABLE | workspace divergence handling (Section 13.7/13.12; preservation runtime unlanded) |
| `CR-FORCE-03` | after inert staging | direct | NOT APPLICABLE | force staging runtime (Section 13.7; filesystem staging is coordinator work, unlanded) |
| `CR-FORCE-03` | after inert staging | task_board | NOT APPLICABLE | force staging runtime (Section 13.7; filesystem staging is coordinator work, unlanded) |
| `CR-FORCE-04` | after final preflight | direct | NOT APPLICABLE | lifecycle orchestration (Section 13.7 final preflight; flow unlanded) |
| `CR-FORCE-04` | after final preflight | task_board | NOT APPLICABLE | lifecycle orchestration (Section 13.7 final preflight; flow unlanded) |
| `CR-FORCE-05` | after confirmation receipt | direct | NOT APPLICABLE | operator confirmation runtime (Section 13.7; confirmation receipt persistence unlanded) |
| `CR-FORCE-05` | after confirmation receipt | task_board | NOT APPLICABLE | operator confirmation runtime (Section 13.7; confirmation receipt persistence unlanded) |
| `CR-FORCE-06` | after new lease persistence | direct | NOT APPLICABLE | lease arbitration beyond sessrepo persistence (new lease persistence unlanded) |
| `CR-FORCE-06` | after new lease persistence | task_board | NOT APPLICABLE | lease arbitration beyond sessrepo persistence (new lease persistence unlanded) |
| `CR-FORCE-07` | after winning-lease recomputation/events | direct | NOT APPLICABLE | lease arbitration with event publication (Section 13.7; recomputation unlanded) |
| `CR-FORCE-07` | after winning-lease recomputation/events | task_board | NOT APPLICABLE | lease arbitration with event publication (Section 13.7; recomputation unlanded) |
| `CR-FORCE-D-01` | after prepare | direct | driven (journal-create) | direct activation prepare through the journal-create seam, evaluated with the force_takeover path |
| `CR-FORCE-D-02` | after prepared commit/provider materialization | direct | driven (journal-to-prepared) | provider materialization prepared under a direct closure with a provider branch, evaluated with the force_takeover path |
| `CR-FORCE-D-03` | after native discovery | direct | NOT APPLICABLE | provider native discovery runtime (Section 13.7; discovery unlanded) |
| `CR-FORCE-D-04` | after plugin resume/process start | direct | NOT APPLICABLE | provider plugin/process runtime (Section 13.7; resume and process start unlanded) |
| `CR-FORCE-D-05` | after finalize may have occurred | direct | driven (journal-finalize) | direct finalize has no external effect in the plan: resume under the same IDs, evaluated with the force_takeover path |
| `CR-FORCE-TB-01` | after prepare/journal creation | task_board | driven (journal-create) | task-board activation prepare through the journal-create seam, evaluated with the force_takeover path |
| `CR-FORCE-TB-02` | after import/open/commit prepared | task_board | driven (journal-to-prepared) | import/open/commit through the journal-to-prepared seam, evaluated with the force_takeover path |
| `CR-FORCE-TB-03` | after adopt may have occurred | task_board | driven (journal-activation) | adopt through the journal-activation seam, evaluated with the force_takeover path |
| `CR-FORCE-TB-04` | after resume/finalize may have occurred | task_board | driven (journal-finalize) | resume/finalize through the journal-finalize seam, evaluated with the force_takeover path |
| `CR-FORK-01` | after ID allocation | direct | NOT APPLICABLE | fork planning (Section 13.8; ID allocation flow unlanded; scalar grammar landed) |
| `CR-FORK-01` | after ID allocation | task_board | NOT APPLICABLE | fork planning (Section 13.8; ID allocation flow unlanded; scalar grammar landed) |
| `CR-FORK-02` | after topology/manifest projection persistence | direct | NOT APPLICABLE | transfer manifest projection (manifest owner; out of checkpoint/journal story boundary) |
| `CR-FORK-02` | after topology/manifest projection persistence | task_board | NOT APPLICABLE | transfer manifest projection (manifest owner; out of checkpoint/journal story boundary) |
| `CR-FORK-03` | after new Session Record/lease persistence | direct | NOT APPLICABLE | sessrepo record persistence with lease arbitration (Section 13.8; out of checkpoint/journal story boundary) |
| `CR-FORK-03` | after new Session Record/lease persistence | task_board | NOT APPLICABLE | sessrepo record persistence with lease arbitration (Section 13.8; out of checkpoint/journal story boundary) |
| `CR-FORK-04` | after provider fork plan or bundle selection | direct | NOT APPLICABLE | fork planning (Section 13.8; provider fork plan and bundle selection unlanded) |
| `CR-FORK-04` | after provider fork plan or bundle selection | task_board | NOT APPLICABLE | fork planning (Section 13.8; provider fork plan and bundle selection unlanded) |
| `CR-FORK-05` | after prepared materialization | direct | driven (journal-to-prepared) | fork prepared journal under a workspace-only closure, evaluated with the fork path |
| `CR-FORK-05` | after prepared materialization | task_board | driven (journal-to-prepared) | fork prepared journal, evaluated with the fork path |
| `CR-FORK-06` | after direct identity/activation | direct | NOT APPLICABLE | provider identity and activation runtime (Section 13.8; identity persistence and activation unlanded) |
| `CR-FORK-07` | after task-board adopt/activation | task_board | driven (journal-activation) | fork adopt through the journal-activation seam, evaluated with the fork path |
| `CR-FORK-08` | after finalize before fork/resume events | direct | driven (journal-finalize) | direct finalize has no external effect in the plan: resume under the same IDs, evaluated with the fork path |
| `CR-FORK-08` | after finalize before fork/resume events | task_board | driven (journal-finalize) | fork finalize through the journal-finalize seam, evaluated with the fork path |
| `CR-STOP-01` | after input block/quiesce | direct | NOT APPLICABLE | input gating with quiescence observation (Section 13.9; block/quiesce runtime unlanded) |
| `CR-STOP-01` | after input block/quiesce | task_board | NOT APPLICABLE | input gating with quiescence observation (Section 13.9; block/quiesce runtime unlanded) |
| `CR-STOP-02` | after checkpoint/export capture | direct | driven (checkpoint-capture) | stop checkpoint through the capture blob/receipt seam |
| `CR-STOP-02` | after checkpoint/export capture | task_board | driven (checkpoint-capture) | stop checkpoint through the capture blob/receipt seam (task-board export closure) |
| `CR-STOP-03` | after process/manager stop may have occurred | direct | NOT APPLICABLE | provider/bridge stop runtime (Section 13.9; stop execution unlanded) |
| `CR-STOP-03` | after process/manager stop may have occurred | task_board | NOT APPLICABLE | provider/bridge stop runtime (Section 13.9; stop execution unlanded) |
| `CR-STOP-04` | after closure verification | direct | NOT APPLICABLE | lifecycle orchestration (Section 13.9 closure verification; flow unlanded) |
| `CR-STOP-04` | after closure verification | task_board | NOT APPLICABLE | lifecycle orchestration (Section 13.9 closure verification; flow unlanded) |
| `CR-STOP-05` | after stopped event/result persistence | direct | NOT APPLICABLE | sessrepo event with cliresult result persistence (Section 13.9; out of checkpoint/journal story boundary) |
| `CR-STOP-05` | after stopped event/result persistence | task_board | NOT APPLICABLE | sessrepo event with cliresult result persistence (Section 13.9; out of checkpoint/journal story boundary) |
| `CR-RESUME-01` | after owner/checkpoint/profile validation | direct | NOT APPLICABLE | resume validation (sessquery admission and provhost profile libraries landed; Section 13.10 flow orchestration unlanded) |
| `CR-RESUME-01` | after owner/checkpoint/profile validation | task_board | NOT APPLICABLE | resume validation (sessquery admission and provhost profile libraries landed; Section 13.10 flow orchestration unlanded) |
| `CR-RESUME-02` | after prepare | direct | driven (journal-create) | resume prepare through the journal-create seam, evaluated with the owner_resume path |
| `CR-RESUME-02` | after prepare | task_board | driven (journal-create) | resume prepare through the journal-create seam, evaluated with the owner_resume path |
| `CR-RESUME-03` | after prepared commit | direct | driven (journal-to-prepared) | resume prepared commit under a workspace-only closure, evaluated with the owner_resume path |
| `CR-RESUME-03` | after prepared commit | task_board | driven (journal-to-prepared) | resume prepared commit, evaluated with the owner_resume path |
| `CR-RESUME-04` | after exact native discovery or dormant manager reconciliation | direct | NOT APPLICABLE | provider native discovery runtime (Section 13.10; discovery unlanded) |
| `CR-RESUME-04` | after exact native discovery or dormant manager reconciliation | task_board | NOT APPLICABLE | task-board bridge runtime (Section 13.10 dormant manager reconciliation; execution unlanded) |
| `CR-RESUME-05` | after provider/bridge activation may have occurred | direct | NOT APPLICABLE | provider activation runtime (Section 13.10 direct activation; execution unlanded) and a workspace-only plan with no journal activation phase |
| `CR-RESUME-05` | after provider/bridge activation may have occurred | task_board | driven (journal-activation) | resume activation through the journal-activation seam, evaluated with the owner_resume path |
| `CR-RESUME-06` | after finalize may have occurred before session.resumed | direct | driven (journal-finalize) | direct finalize has no external effect in the plan: resume under the same IDs, evaluated with the owner_resume path |
| `CR-RESUME-06` | after finalize may have occurred before session.resumed | task_board | driven (journal-finalize) | resume finalize through the journal-finalize seam, evaluated with the owner_resume path |
| `CR-RESTORE-01` | after index validation | direct | NOT APPLICABLE | localstore SQLite projection (library landed; Section 13.11 restore flow unlanded) |
| `CR-RESTORE-01` | after index validation | task_board | NOT APPLICABLE | localstore SQLite projection (library landed; Section 13.11 restore flow unlanded) |
| `CR-RESTORE-02` | after prior-live-session enumeration | direct | NOT APPLICABLE | sessquery read library (landed; Section 13.11 restore flow orchestration unlanded) |
| `CR-RESTORE-02` | after prior-live-session enumeration | task_board | NOT APPLICABLE | sessquery read library (landed; Section 13.11 restore flow orchestration unlanded) |
| `CR-RESTORE-03` | after lease refresh | direct | NOT APPLICABLE | lease arbitration beyond sessrepo persistence (lease refresh unlanded) |
| `CR-RESTORE-03` | after lease refresh | task_board | NOT APPLICABLE | lease arbitration beyond sessrepo persistence (lease refresh unlanded) |
| `CR-RESTORE-04` | after stale-owner marking | direct | NOT APPLICABLE | lease/state arbitration (Section 13.11; stale-owner marking unlanded) |
| `CR-RESTORE-04` | after stale-owner marking | task_board | NOT APPLICABLE | lease/state arbitration (Section 13.11; stale-owner marking unlanded) |
| `CR-RESTORE-05` | after checkpoint/profile validation | direct | NOT APPLICABLE | restore validation (sessquery admission and provhost profile libraries landed; Section 13.11 flow orchestration unlanded) |
| `CR-RESTORE-05` | after checkpoint/profile validation | task_board | NOT APPLICABLE | restore validation (sessquery admission and provhost profile libraries landed; Section 13.11 flow orchestration unlanded) |
| `CR-RESTORE-06` | after wrapper recreation | direct | NOT APPLICABLE | terminal wrapper runtime (Section 13.11; recreation unlanded) |
| `CR-RESTORE-06` | after wrapper recreation | task_board | NOT APPLICABLE | terminal wrapper runtime (Section 13.11; recreation unlanded) |
| `CR-RESTORE-07` | after auto-resume may have occurred | direct | NOT APPLICABLE | auto-resume runtime with status/event reconciliation (Section 13.10/13.11 lifecycle; unlanded) |
| `CR-RESTORE-07` | after auto-resume may have occurred | task_board | NOT APPLICABLE | auto-resume runtime with status/event reconciliation (Section 13.10/13.11 lifecycle; unlanded) |
| `CR-CLONE-01` | after resolve | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-01` | after resolve | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-02` | after probe | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-02` | after probe | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-03` | after inspect | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-03` | after inspect | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-04` | after snapshot | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-04` | after snapshot | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-05` | after capture plan | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-05` | after capture plan | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-06` | after raw capture | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-06` | after raw capture | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-07` | after normalize | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-07` | after normalize | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-08` | after canonical validation | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-08` | after canonical validation | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-09` | after projection/checkpoint plan | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-09` | after projection/checkpoint plan | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-10` | after policy gate | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-10` | after policy gate | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-11` | after prepare | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-11` | after prepare | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-12` | after staged read-back | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-12` | after staged read-back | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-13` | after source/collision recheck | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-13` | after source/collision recheck | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-14` | after publish/live read-back | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-14` | after publish/live read-back | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-15` | after finalize/lineage | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-15` | after finalize/lineage | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-16` | after resume-plan/optional open | direct | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |
| `CR-CLONE-16` | after resume-plan/optional open | task_board | NOT APPLICABLE | clone flow (Section 13.14; unlanded) |

## Negative-evidence proofs

Shipped harness: `internal/crashgate/mutant_harness.py`. Run:

    PYTHONDONTWRITEBYTECODE=1 python3 internal/crashgate/mutant_harness.py --log-dir .temp/TASK-260830-17ootk/mutants

Result on the delivered source: **9 of 9 narrowing mutants killed**,
**1 of 1 harmless SURVIVED controls applied**, 0 NOT_APPLIED, 0
anomalies, tree restored. Per-plant raw logs (commands, return
codes, full output) live under the log dir with `summary.json`; the
same logs ship in `TASK-260830-17ootk_producer-evidence.tar.gz`.
Every plant runs the behavioral suite (`-run TestCrashGate`), not
only the static derivation.

| Plant | Weakening (admits exactly the named member) | Killer |
| --- | --- | --- |
| `N-registry-grace11` | registry flips `CR-GRACE-11`/task_board to NOT APPLICABLE with the `CR-GRACE-11` token preserved | `TestCrashGateConformance` coverage + `TestCrashGateNotApplicable` fail; static derivation still passes |
| `N-recover-lease-backward` | lease-moved gate admits a backward epoch move (`!=` to `>`) | `TestCrashGateRejections/lease_moved_backward` |
| `N-recover-native-blank` | substitution gate admits the relabelled blank state (empty native-after) | `TestCrashGateRejections/blank_relabel` |
| `N-recover-two-auth-manager` | two-live-authorities gate additionally requires `ManagerMatches` (admits the manager-mismatched member) | `TestCrashGateRejections/two_live_authorities` |
| `N-recover-unfenced-live` | unfenced gate requires the explicit `Live` flag (admits state-derived liveness) | `TestCrashGateRejections/unfenced_state_liveness` |
| `N-recover-host-rename` | host gate keeps `disk_full`, admits exactly `rename_blocked` | `TestCrashGateSection1312/rename_blocked_probe_parks` |
| `N-recover-marker-mismatch` | marker gate keeps `invalid`, admits exactly `mismatch` | `TestCrashGateMutualExclusion/marker_mismatch_parks` |
| `N-recover-provider-floor` | unrecorded-prepare floor raised `CR-MAT-05` to `CR-MAT-06` (admits unknown provider status at `CR-MAT-05`) | `TestCrashGateMutualExclusion/unknown_unrecorded_prepare_parks` |
| `N-recover-bridge-floor` | unrecorded-import floor raised `CR-MAT-05` to `CR-MAT-06` (admits unknown bridge status at `CR-MAT-05`) | `TestCrashGateMutualExclusion/unknown_unrecorded_import_parks` |
| `C-harmless-comment` | comment-only edit | full behavioral suite still passes (SURVIVED) |

## Mapped seams (boundary ID to evaluated production seam)

Checkpoint rows evaluate at the capture blob/receipt seam (recovery
is the idempotent retry; capture has no evaluator, rollback, or
parked semantics). Journal rows evaluate at the named `CR-MAT`
seam with the flow's evaluation path, per the Section 13.13 rule
that the `CR-MAT` row applies independently to graceful takeover,
force takeover, passive sync, owner resume, and fork:

| Registry boundary | Evaluated seam | Evaluation path |
| --- | --- | --- |
| `CR-SYNC-06` | `CR-MAT-02` (dormant prepare) | `passive_sync` |
| `CR-SYNC-07` | `CR-MAT-08` (dormant finalize) | `passive_sync` |
| `CR-GRACE-01` | `CR-MAT-02` | `graceful_takeover` |
| `CR-GRACE-09` | `CR-MAT-06` | `graceful_takeover` |
| `CR-GRACE-11` task_board | `CR-MAT-07` | `graceful_takeover` |
| `CR-GRACE-12` | `CR-MAT-08` | `graceful_takeover` |
| `CR-FORCE-D-01` | `CR-MAT-02` | `force_takeover` |
| `CR-FORCE-D-02` | `CR-MAT-06` (direct closure with provider branch) | `force_takeover` |
| `CR-FORCE-D-05` | `CR-MAT-08` (direct resume, no effect in plan) | `force_takeover` |
| `CR-FORCE-TB-01` | `CR-MAT-02` | `force_takeover` |
| `CR-FORCE-TB-02` | `CR-MAT-06` | `force_takeover` |
| `CR-FORCE-TB-03` | `CR-MAT-07` | `force_takeover` |
| `CR-FORCE-TB-04` | `CR-MAT-08` | `force_takeover` |
| `CR-FORK-05` | `CR-MAT-06` | `fork` |
| `CR-FORK-07` | `CR-MAT-07` | `fork` |
| `CR-FORK-08` | `CR-MAT-08` | `fork` |
| `CR-RESUME-02` | `CR-MAT-02` | `owner_resume` |
| `CR-RESUME-03` | `CR-MAT-06` | `owner_resume` |
| `CR-RESUME-05` | `CR-MAT-07` | `owner_resume` |
| `CR-RESUME-06` | `CR-MAT-08` | `owner_resume` |
| `CR-MAT-01..08` | self | `owner_resume` (all five paths appear via mapped rows) |
| `CR-STOP-02`, `CR-GRACE-04`, `CR-GRACE-08` | capture blob/receipt seam | direct + task_board closures |

## Stated bounds (not driven, declared)

1. Status probes (provider, bridge, marker, lease, native) are
   modeled inputs, as in `matjournal`. No process, manager, lease
   arbitration, or replica file is driven.
2. Destination-marker completion bytes (`MarkerBytes`) are proved
   in `matjournal` (`TestRecoverCompletesCommit/provider_workspace`);
   this harness covers marker mismatch/invalid/absent parks and the
   dormant/board completions that need no marker.
3. The `CR-CLONE` finalization injection prefix list (Provider
   commit through G4, no `CR-` IDs) belongs to the unlanded clone
   flow with the `CR-CLONE-01..16` rows.
4. Rollback/parked visibility through event, CLI/status, audit, and
   lifecycle-projection surfaces belongs to unlanded owners; the
   harness proves the journal-durable halves (terminal result,
   `last_error`, `recovery.json`, frozen phase, unchanged lease).
5. The full hook matrix (before-write/after-journal/after-commit)
   is proved at `CR-STOP-02` and `CR-MAT-01/02`; mapped rows use
   the representative post-write seam named in each record.
6. Epoch-1 bootstrap-abort classification belongs to the lifecycle
   story; the first-checkpoint capture seam it would use is proved
   here.

## Evidence index (this run)

- Conformance records: 74 JSON files under `conformance/` (72 table rows +
  2 real-SIGKILL rows), outcomes `{safe_retry: 62, recoverable_parked_state:
  11, explicit_rollback: 1}`. Every record names boundary, path, evaluated
  seam, closure, variant, driver, operation IDs, pre/post durable facts,
  external effect, status probe, lease before/after, native before/after,
  outcome, reason, and remediation.
- Mutant logs: `mutants/` (9 `N-*.log` KILLED + `C-harmless-comment.log`
  SURVIVED + `summary.json`).
- Suite logs: `crashgate-verbose.log` (evidence run with records),
  `crashgate-evidence-run.log`, `full-test.log` (29 packages ok),
  `cover.log`, `race-*.log` (4 chunks), `cigate-contracts.log`,
  `cigate-claims.log`.
- Validation commands (all exit 0; see results artifact for the table):
  `go run ./internal/traceability/cmd/tracecheck`, `go generate`
  + catalog diff, cigate contract + claim gates, `go vet ./...`,
  `GOOS=windows go vet ./...`, `gofmt -l`, `go build ./...`,
  `GOOS=windows go build ./...`, `go test ./... -count=1`,
  `go test ./... -cover -count=1`, `go test -race` (all packages),
  fixture packages, fuzz smoke (13 targets).
