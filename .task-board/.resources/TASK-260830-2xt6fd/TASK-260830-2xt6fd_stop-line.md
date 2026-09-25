# TASK-260830-2xt6fd — quiescence ownership stop-line

Run RUN-260908-992351. Goal GOAL-260908-325064 revision 1. Date 2026-09-08.
Task: test-git-race-and-closure-validation, in STORY-260830-35dbcs complete-git-workspace-capture.

## Disposition

Implementation is blocked at a concrete ownership prerequisite, before product-code changes. This is not a review handoff, an accepted capture implementation, or a claim that passing decoder tests prove runtime quiescence. No checklist item is checked by this investigation. The full final-leaf deliverable remains outstanding; the scope is not reduced to metadata/content-only capture.

The active goal's exact returned objective and three-item scope are preserved in goal.txt. Its normal predicate requires scoped acceptance criteria, checklists and outcomes through role handoff; it also expressly permits the repository Stop-The-Line boundary when evidenced. This packet invokes that boundary, not the normal handoff predicate. No parent/primary goal mutation has been performed.

## Constraint and primary evidence

Pinned AX v0.5.0 commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, internal/specdoc/SPEC.md SHA-256 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a, verified directly from bytes. Section 12.3 (line 7992 onward) requires capture to quiesce agent input and filesystem-mutating provider work, then detect changes in HEAD, index and included-file digests. A consistent reread is not proof of that quiescence.

The assignment requires an existing quiescence owner or a clean integration contract and forbids inventing an attestation boolean to hide a missing owner. It simultaneously excludes full provider runtime/UI work and confines implementation to this Git capture Story.

Current repository facts, tied to source-inventory.json:

- internal/gitsnap/snapshot.go Capture accepts Runner, directory and optional ContentOptions. It calls capture, which reads Git state and optionally content, then rechecks Git consistency. It has no runtime session binding or call to a provider/terminal quiescence owner.
- internal/gitsnap/content.go ContentOptions owns store, include/exclusion policy and local roots. contentState.capture uses the accepted scanner, guarded content reads, submodule recursion and digest rereads. These are valid lower-level operations, not runtime quiescence.
- internal/provhost/runner.go Host.Call transports one operation to an executable under a deadline; it holds no cross-call state. The caller must already supply authority. internal/provhost/quiesce.go DecodeQuiesceProof validates reported facts and returns the observed safe bit; it does not stop input or provider work.
- internal/provhost/conformance_test.go TestQuiesceThroughCall uses scriptRunner with canned safe/unsafe proof bodies. It proves host transport/decoding only. Rerunning it is not real-provider evidence.
- internal/provhost/doc.go explicitly limits the host to Provider Protocol major 2 and assigns v3 binding validation to terminalbackend. Fresh tests confirm well-formed v3 envelopes are refused. This is an additional wiring bound, not a claim that v2 itself is inherently unusable.
- internal/terminalbackend/terminalbackend.go explicitly assigns lifecycle operations to sibling tasks. conformance.go CheckTransition computes allowed states/effects; it does not execute quiesce-input. Its ledger is a conformance helper, not an input-control runtime.
- The complete go list import graph contains no production consumer of provhost and no runtime capture coordinator. This structural evidence is combined with the relevant source inspection and live owner tasks; a text-search miss alone is not treated as exhaustive platform evidence.
- The authoritative board identifies TASK-260830-1c28dz implement-tmux-lifecycle-operations as to-dev, blocked by TASK-260830-35urbp, with no outcomes. TASK-260830-kkh1an implement-terminal-instance-state-machine is to-dev, blocked by TASK-260830-1geqhj, with no outcomes. These are outside the assigned three-leaf scope. No running handle is asserted for either task.

## Accepted predecessor and checkout evidence

workspace.json reports both predecessor CR2 records checkpointed:

- TASK-260830-2bnr39 implement-git-repository-and-index-snapshot: CR2 tree 935f624e45088a064de04945cbe4bd86d1b37d1b, independent reviewer RUN-260907-428f33; checklist 18/18 in the scoped board read.
- TASK-260830-3m7m7w implement-untracked-ignored-symlink-submodule-capture: CR2 tree 4f1407cb98f19424db527d313a3d89d34e2fd231, reviewer RUN-260908-2e4871; checklist 19/19. Read the exact reviewer verdict and checkpoint resource through resource get. Original checkpoint e81724a4e6e338e19864518a9f6888d976864301 was subsequently replayed by the managed owner when refreshing the base.
- Current managed checkpoint/tip d73a28570c221a9c490f98f8a9848546d4c13068, base 7654d7cadb2c226bfa3db5f23ac355a730285eea; dirty=false; lease RUN-260908-992351. No manual commit, rebase, switch, reset or integration was performed.

The earlier review's native macOS-only, auxiliary sharedindex timestamp, external-filter and change-and-revert bounds are retained. Neither prior acceptance nor this investigation claims general platform quiescence.

## Failed assumptions and evaluated routes

1. Assumption: the refreshed base already contains a callable quiescence lifecycle owner. Source, import graph and live scoped owner records contradict this.
2. Assumption: Host.Call plus DecodeQuiesceProof by itself supplies the missing owner. It supplies transport and validation, while the provider runtime, terminal binding, input gate lifetime and authority must already exist. A canned proof or user-supplied safe=true would conceal the missing integration.
3. Assumption: before/after Git/file hashes establish a safe boundary. They can detect observed races but cannot hold agent input closed or prove provider writers stopped. They also do not detect an arbitrary change-and-revert interval; that accepted bound must not be upgraded by prose.

No compensating flags, stub quiescer, alternate scanner, per-path exception, forged attestation or mock-only runtime capability was implemented. Pack/index/manifest construction remains technically implementable once the capture integration boundary is settled; this is an ownership/scope blocker, not a claim that Git packs are impossible.

## Options, tradeoffs and exact decision

Recommended: deliver the existing runtime prerequisites first, then provide this leaf a production quiescence owner/entry point and real evidence tying its held boundary to the selected runtime session, terminal generation and capture lifetime. Preserve this Story's complete Git assembly scope and accepted policy owners. This changes delivery ordering and requires work outside the assigned scope, but avoids competing runtime ownership.

Alternative: explicitly expand this Story's authorized scope to implement the missing quiescence coordinator and the runtime prerequisites/evidence it needs. This is substantially more than API plumbing: input-control lifetime, provider write exclusion, authority binding and release/recovery must have real owners and tests. It overlaps the named sibling tasks and conflicts with the current runtime exclusion unless the owner changes it.

Not acceptable: return closed-looking transfer manifests and call capture complete on the strength of a caller-supplied boolean, canned proof, or stable filesystem rereads. A documented future interface without a production implementation is not satisfaction of Section 12.3.

Exact requested owner decision/input: deliver the runtime prerequisites and name the executable quiescence boundary this capture must use, or explicitly authorize and assign the overlapping runtime work to this Story. A route question was sent through the session input tool. No elapsed-time approval is inferred.

## Measured final-leaf coverage and unimplemented scope

**0 of 6 final-leaf AC rows newly driven to satisfaction in this run.** This is an explicit requirement audit, not a claim that inherited tests cover nothing. The predecessor review's 8/8 repository and 7/7 content driver ratios are prior accepted, narrower evidence; they cannot stand in for these rows.

| Required row | Intended production integration | Current evidence/result |
| --- | --- | --- |
| Held input/provider quiescence through capture | Runtime owner -> Capture | Missing runtime owner; blocked |
| Complete source-change refusal through capture interval | Capture -> existing Git/content readers | Earlier partial race witnesses retained; final interval closure not implemented |
| Repository-local packs, exact inventories, raw/logical index agreement | Capture -> per-repository Git pack/index assembly -> immutable store | Outstanding; not tested by decoder reruns |
| Workspace root/child, blob, cwd/config and safe path closure | Capture -> exact transfer manifest assembly/validation | Outstanding; no root emitted |
| Actual byte identity, recursive dirty state and unsupported cases | Capture -> accepted content owner + complete assembly validation | Existing content evidence retained; complete-output validation outstanding |
| Truthful capability, mutation/recovery and publication evidence | Production capture/store plus docs, tests and managed CR | No new claim, capability or candidate published |

No new gate is shipped and no mutation experiment was run. Therefore there is no new killed/surviving-mutant claim. Prior tables remain attributed to their predecessor reviews; they are not fresh final-leaf mutation evidence. Final-leaf narrowing mutants, neutral/bad controls, relevant full tests/coverage, lint/build and configured publication validation have NOT been run. They remain required before handoff after the boundary is resolved. No check item tied to those commands is marked green.

## Fresh command exits and limits

| Standalone command | Actual exit | What it establishes |
| --- | ---: | --- |
| go test ./internal/provhost -run '^(TestQuiesceThroughCall|TestDecodeQuiesceSafeLies|TestDecodeResponseWellFormedV3SuccessIsMismatch|TestDecodeResponseV3FailureWithValid130ErrorIsMismatch)$' -v -count=1 -timeout=2m | 0 | Four named host/decoder drivers pass; no provider runtime exercised |
| go test ./internal/terminalbackend -run '^TestOnlyQuiesceInputEntersQuiescing$' -v -count=1 -timeout=2m | 0 | Transition rule passes; no input gate executed |
| go list -f '{{.ImportPath}} {{join .Imports ","}}' ./... | 0 | Complete module package import graph |

Logs are attached in the companion archive. Go 1.25.5 darwin/arm64; Git 2.50.1 (Apple Git-155). All launched commands are terminal; no background job is left to produce later evidence. There was no product-code change, so no build/full-suite claim is manufactured for a blocked implementation.

Initial environment anomaly: the shell had no inherited TASK_BOARD_DIR or TASK_BOARD_RUN_ID. The first required status mutation therefore reached the stale checkout board and was refused (exit 1), changing no state. Reissued with explicit authoritative TASK_BOARD_DIR, it exited 0 and set this task to development. All subsequent board operations use the explicit control board. One initial query used unknown operation task (exit 1); it was corrected to get. Missing checkout skill links were resolved by reading the Curator-managed skill at the supplied control-root path. These are recovered operational errors, not the blocker.
