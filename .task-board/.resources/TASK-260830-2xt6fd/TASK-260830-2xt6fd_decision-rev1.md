ORCHESTRATOR DECISION on `TASK-260830-2xt6fd_quiescence-blocker.md`. The answer comes from the pinned specification, so no owner or API extension is needed.

Pinned SPEC v0.7.0 §4.C, operation table (lines 1198-1215), defines the Terminal Instance state machine as CLOSED:
- `quiesce-input`: active|parked → quiescing (`input_closed`);
- `wait-safe-boundary`: quiescing → quiescing;
- `request-stop`: quiescing → stopped;
- `restore`: absent|stopped|unavailable → parked.

No transition goes from `quiescing` back to `active` in the same incarnation. A "release" operation would add an eleventh operation to a closed vocabulary and contradict the spec. Do NOT add one, do NOT unlock tmux, and do NOT add a caller flag. Your analysis was correct: those would be forced fits.

§12.3 (9076-9078) requires that capture "quiesce agent input and filesystem-mutating provider work" and fail on any HEAD/index/file-digest change. Combined with §4.C, **capture is a stop-point operation**. The capture coordinator you build:
1. requires the Terminal Instance to be either `quiescing` with a recorded safe boundary for the current quiescence generation, or `stopped`. It REFUSES with a literal typed code in every other state (active, parked, creating, stale_fenced, unavailable, absent, and quiescing without a boundary). It never releases, reopens or changes the lease;
2. composes the LANDED owners only: the lifecycle `quiesce-input` / `wait-safe-boundary` / `request-stop` (tmuxserver, terminstance, axpane). It never duplicates them;
3. runs capture inside that window and fails if HEAD, index or any included file digest changes;
4. for crash recovery: an instance left `quiescing` by a crashed capture is recovered through the owner's existing path (`request-stop`, or `terminate-stale` when stale), and capture is then retried from `stopped`. It is never recovered by a release. Prove idempotent retry after a crash at every capture phase.

Closing the held-quiescence row therefore needs NO new owner API. It needs a coordinator that holds quiescence by construction, because it can only run in those states, and that is proven:
- a whole-domain test over every Terminal Instance state × capture phase;
- a narrowing that admits one wrong state (for example parked, or quiescing without a boundary), killed by its named test run ALONE;
- a structural test that the coordinator calls no transition outside {quiesce-input, wait-safe-boundary, request-stop, status}.

Record this decision and its citations in the results and in TRACEABILITY. Non-disruptive capture that keeps the session live would be a SPECIFICATION change for the spec owner. It is not in scope, and you should state it as a bound.

Then continue your resume brief (`TASK-260830-2xt6fd_resume-rev1.md`) from STEP 4. The refresh onto `6d3bff9` is done, and the v0.5→v0.7 crosswalk is done (byte-identical clause bodies). Do the AC by construction, then the story-final registry for all three leaves, then the full validation. Set the task from `blocked` back to development yourself, keep the checklist current, and run `task-board handoff TASK-260830-2xt6fd --role developer`. If trunk moves again before the handoff, run `refresh-candidate` again. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max.
