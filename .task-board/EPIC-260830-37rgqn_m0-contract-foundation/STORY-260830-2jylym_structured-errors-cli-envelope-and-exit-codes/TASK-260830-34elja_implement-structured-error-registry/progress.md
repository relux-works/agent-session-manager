## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- TASK-260830-8x76g1
- TASK-260830-uqnwmi

## Blocks
- TASK-260830-1bipsa

## Checklist
- [x] Production entry points implement the scoped deliverable: Implement Structured Error versions, typed details, stable codes, retryability, and causal redaction
- [x] Relevant positive, negative, compatibility, and recovery tests pass with logs attached
- [x] README/doctor/capability evidence and specification traceability are updated without unsupported claims
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
LEAF 1 OF 3 — implement-structured-error-registry. Strictly sequential Story: leaf 2 (cli-result-envelopes) and leaf 3 (compatibility) depend on this one, and four other m0 Stories depend on the Story. Build the foundation others will bind to.

SCOPE OF THIS LEAF
Structured Error versions, typed details, stable codes, retryability, and causal redaction, per §14.2 and §17.2 of the pinned AX v0.5.0 SPEC.

THIS STORY HAS AN OBJECTIVE ACCEPTANCE SIGNAL THAT MOST DO NOT
The ownership gate landed in PR #28 has already measured this exact scope and found it empty. Today:
  tracecheck -section 15.2 -> exit 1. §15.2 is the nineteen-row normative exit-code registry at SPEC.md:11073-11095, bound to internal/catalog/catalog.go:ForRelease - an unrelated symbol - and nothing in the tree implements it. The only os.Exit calls are the exit(1) failure paths of cataloggen and tracecheck.
  section:17.2's single clause is an unknown-event READER obligation while its binding names the Configuration WRITER EncodeCurrent.
When this Story is complete those bindings should become honest. §15 belongs mostly to leaf 2 (exit-code mapping); §17.2 is in scope here.

HOW THE GATE WILL JUDGE A BINDING - READ THIS BEFORE WRITING ONE
Each clause a binding claims must index the measured inventory, declare its exact SPEC.md line, quote it verbatim beginning on that line, and name an acceptance case the binding itself owns. The declared coverage level must equal the measured bucket. `unmeasured` is admitted nowhere.
DO NOT re-bind a section to a friendlier symbol to make the gate green. That failure has appeared three times on this board and is reviewed for specifically. If a section cannot honestly be claimed, say so and leave it.

WHAT THIS BOARD HAS PAID TO LEARN - THESE APPLY TO YOUR WORK, NOT JUST TO REVIEW
- Never invent a constraint the pinned document does not declare. Six were found and removed this session. Quote the pinned SPEC verbatim with file and line; internal/specdoc embeds it byte-exact under the pinned digest and is the tool for checking your own citations. Use it rather than trusting a line number from any report.
- A gate reports coverage as a measured ratio, not as prose. Prose about what a check does is always true of what it sees and silent about what it does not.
- A pin over a documented claim must exercise the claim's SUBJECT, not only its parameter. Matching a number and then testing the dimension you already believed in confirms the digit and leaves the noun unverified.
- 'Nothing is there' is not 'nothing this checker sees'. If you cannot check a class, name it as a stated bound rather than reporting the class as empty.
- An exported doc comment is a contract. If it promises a refusal, something must fail when the refusal stops happening.
- Negative tests for every gate, refusal, validation and authorization path. A positive test proves reachability, not correctness. Prove a bound by NARROWING, not only by deleting.
- Confirm every mutant is PRESENT in the file before believing a green or a red.

DELIVERY DISCIPLINE FOR THIS STORY
- This is a non-final leaf: it will be checkpointed, not integrated. Leave the branch exactly one commit past the checkpoint and the worktree clean.
- Do not land anything on trunk while this Story is in flight, and expect the same courtesy - a Story stranded by exactly that took a day to unpick and its work had to be re-derived by others.
- Disclose anything you find but do not fix, including in code you did not touch.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; foundation leaf of the phase-1 m0 Story that four other Stories depend on."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-cc90a1, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-cc90a1)
Scope confirmed: Structured Error registry only (CLI Result envelopes and exit-code wiring belong to sibling TASK-260830-1bipsa; compatibility proofs to TASK-260830-33sfxc). Implementing internal/axerror: closed versioned object 1.0.0-1.3.0, catalog-derived code/exit registry with per-version admission, static containing-contract binding table, quoted retryability refusal policy, typed details with declared bounds, and causal redaction that keeps the Go cause chain structurally off the wire.
DEVELOPER HANDOFF — internal/axerror, one signed commit 4c9efc0, exactly one past checkpoint d1d3ece, worktree clean.

DELIVERED: internal/axerror implements Structured Error 1.0.0-1.3.0 — closed versioned object, catalog-projected code-to-exit registry with per-version admission, closed 19-row Section 15.2 exit table, static containing-contract binding table (15 majors, no negotiation path), quoted retryability refusal policy, typed details for the two code-to-detail bindings Section 15.3 names, causal redaction, and LocalFromUntrusted for the bootstrap boundaries.

THREE DEFECTS FOUND WHILE WRITING THE EVIDENCE, all fixed and all in LOGBOOK.md 0742:
1. My first detail scanner matched key SUBSTRINGS against the whole Section 16.2 table — the same invented-constraint shape BUG-260902-2faftr removed from hasForbiddenConfigName. It refused token_count / socket_timeout_ms / credential_profile while admitting a secret under any innocuous name. Now exact-match only, scoped to Section 15.1 four declared classes. Caught by reading the logbook, not by any test: 17 mutants were green over the invented rule.
2. That scanner test ranged over the production map, so a mutant deleting one key deleted its own test case and SURVIVED. Registry now pinned as a reviewed literal table compared against production.
3. exit_code as *json.Number admitted the JSON string "9". Now decoded from json.RawMessage with a leading-quote refusal.

REVIEWER: the load-bearing rows are the ADMITTED ones in TestDetailsRefuseExcludedClasses. Please check every refusal against the pinned line that declares it, per the BUG-260902-2faftr review rule.

NOT CLAIMED, deliberately: clause 15.1#5 (RPC hello) and 15.3#3 (hello key / TerminalBackend capability) — no builder exists here, both are declared gaps. Section 17.2 NOT rebound: its single scanner-visible clause is an unknown-EVENT reader obligation and an unknown error CODE does not discharge it; its wrong binding to config EncodeCurrent is left in place and left disclosed. Section 15.2 stays unmeasured; the process exit status belongs to leaf 2. LocalFromUntrusted refuses the Directory Node surface because Section 15.3 leaves its code as-applicable.

MEASURED: 316/316 catalog code-version pairs resolve; 19/19 exit rows; 15/15 binding majors; 8/8 bootstrap rows; 27/27 reviewed scanner keys over 4/4 classes; 99.7% statement coverage; 20/20 mutants killed. Repo gate: acceptance_cases 43->53, clauses_discharged 2/394 -> 10/394, 15.1 partial 6/7 and 15.3 partial 2/3 (both still refused by assigned-scope admission, both added to the refusal disclosure tables with exact ratios).

EXIT CODES, all real, all 0: go test ./internal/axerror -count=1 -v; go test ./internal/axerror -cover -count=1; go test ./... -count=1; go vet ./...; go build ./...; go generate ./internal/catalog + git diff --exit-code on catalog_gen.go; go run ./internal/traceability/cmd/tracecheck; gofmt -l (no files). Logs attached as task-scoped outcome resources.

NO DURABLE STATE: the package holds none, opens no file, starts no process, so no crash/idempotency evidence is applicable. It advertises no capability: no enabled/available/supported/status field exists in it.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-cc90a1, pid=39987, exit=0)
REVIEWER CONTEXT — LEAF 1 OF 3, THE FOUNDATION FOUR OTHER m0 STORIES DEPEND ON

WHICH COMMIT. HEAD 4c9efc0. Verify reachability first; refuse as void if unreachable. The CR reports repository_delta=empty — that is a snapshot artifact, the real delta is the commit itself (19 files, +3666).

WHAT WAS BUILT. internal/axerror, a new package: 1449 production lines, 1810 test lines. Structured Error versions, typed details, stable codes, retryability, causal redaction.

THE MEASURED RESULT, WHICH IS THIS STORY'S UNUSUAL ADVANTAGE
The ownership gate moved from 2/394 to 10/394 normative clauses discharged. The eight new ones are section:15.1 at 6 of 7 and section:15.3 at 2 of 3, both declared `partial` with exact ratios. acceptance_cases 43 -> 53.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED — EXTEND, DO NOT REPEAT
- The leaf edits the gate that judges it, so I checked that first. The only changes to internal/traceability/traceability.go and its tests are: the reviewedOwnershipCanonicalSHA256 digest pin, which MUST change when the registry changes and is the mechanism that stops a coverage claim being self-minted; and test expectation strings updated to the new measured numbers. No gate logic was weakened. Confirm that independently.
- section:17.2 was NOT re-bound to something friendlier. It stays unevidenced with a gap naming exactly why: EncodeCurrent writes Configuration documents and the section's clause is an unknown-event reader obligation. That was the easy way to look green and it was refused.
- section:14.2 still has no scoped implementation owner. Judge whether that is honest for this leaf's scope or an under-claim.

THE HIGHEST-VALUE REVIEW WORK ON THIS CHANGE
The ownership gate's own stated residual is that it CANNOT decide whether a named acceptance case exercises the clause's meaning — a complete enumeration pointed at one weak test is admitted. That is precisely the gap a human reviewer exists to fill, and it is where eight freshly claimed clauses are most likely to be thin.
For each of the eight, check by hand: does the named acceptance case actually exercise that clause's obligation, through a production entry point, with a negative arm? A clause discharged by a positive-path test is the shape this board refuses everywhere else, and the gate will not catch it.

ALSO ATTACK
1. NEGATIVE COVERAGE IN THE NEW PACKAGE. 1810 test lines is not evidence by itself. Every refusal, validation and redaction path needs a test that fails when the gate admits what it must reject. Prove a bound by narrowing, not only by deleting.
2. CAUSAL REDACTION is the part most likely to leak. Try to get an unredacted cause through the exported surface.
3. STABLE CODES AND RETRYABILITY are contracts other Stories will bind to. If a code can change or a retryability flag can be derived inconsistently, four downstream Stories inherit it.
4. The digest pin: confirm it is recomputed from the registry rather than declared, so a registry edit without a matching pin reddens.

STANDING REVIEW BAR
- Positive-path-only evidence is refused.
- Confirm every mutant is PRESENT in the file before believing a green or a red.
- A gate reports coverage as a measured ratio, not as prose; a pin over a documented claim must exercise its SUBJECT, not only its parameter; 'nothing is there' is not 'nothing this checker sees'.
- Do not accept a claim in a comment, README or LOGBOOK you have not independently tested.
- This is a NON-FINAL leaf: it will be checkpointed, not integrated. Report anything you find but do not require fixed.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; review of eight freshly claimed clauses where the gate's own residual is exactly what a reviewer must check."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-737126, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-737126)
REVIEWER VERDICT — CHANGES REQUESTED (CR-TASK-260830-34elja-1 rev1, run RUN-260903-737126). Full evidence: TASK-260830-34elja_review-verdict.md. Reviewer mutants: TASK-260830-34elja_review-mutants.sh.

ON repository_delta=empty: snapshot artifact, NOT an empty delivery. The CR base OID IS the leaf commit 4c9efc0; base tree and HEAD tree are both 42868f86. The checkpoint advanced onto the leaf before the candidate was snapshotted, so the reviewable window closed over already-committed work. Reviewed the commit directly, per the orchestrator brief.

TWO BLOCKING DEFECTS, both reproduced.

F1 — VALIDATED DETAILS ARE ALIASED; THE WRITER EMITS AN OBJECT ITS OWN READER REFUSES. cloneDetails (axerror.go:321) is a SHALLOW copy and Detail (axerror.go:249) hands back the live nested container. After New succeeds, writing into the map returned by failure.Detail("context") put "password":"hunter2" and "access_token":"ghp_secret" into the MarshalJSON output — a class Section 15.1 forbids outright. The same accessor emitted 33036 bytes (bound 16 KiB), a value 7 containers deep (bound 4), and "count":7, a Go int validateDetailValue refuses by type. axerror.Decode then REJECTED the package own output: details["context"] exceeds maximum nesting depth 4. A caller retaining the nested map it passed to New gets the same without touching the accessor; only the top-level map is protected.
This is a bypass path around the check. ValidateDetails is the named production entry point for acceptance case structured-error-detail-redaction, which discharges clause 15.1#2 (Details MUST be redacted and schema-valid); the same aliasing defeats the bounds clause 15.1#3 claims. NO test covers post-construction mutation, so nothing fails when the refusal stops happening. It also contradicts the Error type own doc comment ("the closed Section 15.1 object is the only shape this type can produce"), RedactionBound does not disclose it, and internal/config already returns an isolated snapshot for exactly this reason. This is the foundation four m0 Stories bind to; leaf 2 will marshal these objects.
FIX: deep-copy in and out (or expose an immutable projection), with a test on BOTH arms — retained alias and Detail() — asserting the encoded bytes are unchanged and still decode.

F2 — CLAUSE 15.1#6 IS CLAIMED AGAINST AN OBLIGATION THAT BINDS THE PROVIDER PLUGIN. The gate counts a clause as one line matching MUST/MUST NOT/SHALL/REQUIRED (traceability.go:970). SPEC.md:11047 carries exactly one such keyword: "The plugin MUST NOT masquerade a different-major object as v2." That binds the CHILD PROCESS. This repository builds no provider plugin — its only binaries are cataloggen and tracecheck. LocalFromUntrusted is the HOST emitter, and the host half of that row carries no RFC 2119 keyword at all, so it is not what the gate measured. This is the same ground on which 15.1#5 was correctly declared undischarged; the 15.1 gap text discloses only #5.
EFFECT: 15.1 should read 5/7 not 6/7, clauses_discharged 9/394 not 10/394. Coverage stays partial and the gate still exits 0, so no gate outcome changes — but the inflated ratio is published in README.md, in the tracecheck expectation strings in main_test.go and traceability_test.go, and inside the reviewed digest pin.
FIX: drop 15.1#6, extend the 15.1 gap to name it beside #5 on the same reasoning, recompute reviewedOwnershipCanonicalSHA256, update the three expectation strings and the README ratios.

NON-BLOCKING: N1 the leading-quote guard in decodeExitStatus:213 is redundant (ParseInt already refuses "9") and deleting it leaves the suite green — behavior still correct, the real fix was *json.RawMessage. N2 several producer mutant-presence markers assert an UNRELATED surviving string (e.g. checking for the exit-16 row after deleting the exit-7 row) and cannot distinguish applied from no-op; no false KILL resulted. N3 the target_auth_missing / capability_unavailable typed-detail asymmetry is CORRECT (15.3 names those five details only for the realm-evidence case) but deserves a sentence so it reads as deliberate.

WHAT HELD, reproduced not accepted: go build / go vet / gofmt clean; go test ./... exit 0 across 12 packages; tracecheck exit 0; the producer 20-mutant harness re-run from the resource killed 20/20; my own 12 narrowing mutants killed 11/12 (survivor = N1). All 19 Section 15.2 exit rows and all 15 binding rows verbatim against the pinned document; all eight claimed excerpts occupy the exact lines declared. 17.2 was NOT rebound to a friendlier symbol — its clause is an unknown-EVENT reader obligation an unknown error CODE does not discharge, and the wrong EncodeCurrent binding was left in place and disclosed; that was the easy way to look green and it was refused. 14.2 correctly left unclaimed. The digest pin IS recomputed, not declared: a semantically inert gap-string edit reddens the gate with digest cb585032 differs from reviewed 5a4f769a, so coverage cannot be self-minted. No gate logic weakened in internal/traceability. Causal redaction is structurally sound — cause is unexported, MarshalJSON cannot reach it, the whole Unwrap chain including errors.Join arms is scanned, and I could not get an unredacted cause onto the wire.

STATED BOUND: internal/axerror has no production caller anywhere in the repo — no .go file outside the package imports it. Expected and disclosed for this leaf, but every behavior is proven at the package boundary and none end-to-end through a command. Reported as a bound, not a finding. I did not re-run the fuzz targets or -cover; the 99.7% figure is accepted from the attached log.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-737126, pid=83512, exit=0)
REWORK BRIEF — TWO BLOCKING FINDINGS. The package is otherwise sound; the reviewer attacked it hard and reports what held. Do not redesign.

F1 (BLOCKING) — VALIDATED DETAILS ARE ALIASED: THE WRITER EMITS AN OBJECT ITS OWN READER REFUSES
cloneDetails (internal/axerror/axerror.go:321) is a SHALLOW copy, and Detail (axerror.go:249) hands the caller the live nested container. ValidateDetails therefore checks a graph the constructed *Error does not own, so every Section 15.1 detail bound can be violated AFTER construction through the package's own exported surface.
Reproduced through the package's own accessor:
  v, _ := failure.Detail('context'); m := v.(map[string]any); m['password'] = 'hunter2'
json.Marshal then emits details containing a class Section 15.1 forbids outright. The same accessor defeats every other declared bound: one probe run emitted 33,036 bytes against a declared 16 KiB.
This is not a hardening nicety. Validation performed at construction that does not survive the constructor's own accessor is validation that reports a property the object does not have — the exact class this board has been closing all session, in a new place. Four downstream Stories will bind to this package and inherit whatever it actually guarantees, not what it validates.
Required: make the guarantee real. Deep-copy on the way in and on the way out, or return an immutable view, or document the aliasing precisely and pin it — but a doc that says 'validated' while an accessor hands out mutable interior state is not an option. Whatever you choose, prove it: a mutation through Detail must not be able to change what MarshalJSON emits, and a test must fail if that stops being true.

F2 (BLOCKING) — CLAUSE 15.1#6 IS CLAIMED AGAINST AN OBLIGATION THAT BINDS THE PROVIDER PLUGIN
ownership.v0.5.0.json claims clause 15.1#6 at SPEC.md:11047, discharged by structured-error-local-untrusted (local.go:LocalFromUntrusted). Line 11047's ONLY RFC 2119 keyword is 'The plugin MUST NOT masquerade a different-major object as v2.' That binds the provider plugin, a child process this repository does not build — its only binaries are cataloggen and tracecheck. LocalFromUntrusted is the HOST emitter, and the host half of that table row carries no RFC 2119 keyword at all, so it is not what the gate measured.
Note you already applied this exact reasoning correctly once: 15.1#5 was honestly declared undischarged because 'this repository contains no RPC hello builder at all.' The same is true of the provider plugin. The 15.1 gap discloses only #5.
Effect: section 15.1 is 5/7, not 6/7; clauses_discharged is 9/394, not 10/394. No gate outcome changes, but the inflated ratio is now published in README.md, in the tracecheck expectation strings in main_test.go and traceability_test.go, and inside the reviewed ownership digest pin. This board's rule is that a gate reports a MEASURED ratio; one of the eight newly claimed clauses is measured against an actor this repository does not implement.
Required: drop 15.1#6, extend the 15.1 gap to name it alongside #5 with the same reasoning, recompute reviewedOwnershipCanonicalSHA256, and correct the three expectation strings and the README ratios to 5/7 and 9/394.

NON-BLOCKING, ADDRESS WHILE THERE
- N1: deleting the text[0]==''' refusal in decodeExitStatus (decode.go:213) leaves the suite green, because strconv.ParseInt already refuses '9'. Behaviour is correct and the real fix was moving exit_code to *json.RawMessage, but the commit message presents the quote check as the fix and no mutant distinguishes it. Either give it independent coverage or stop presenting it as the fix.
- N2: several mutant-presence markers assert an unrelated string — checking 'func IsFailureExitStatus' after deleting a guard cannot distinguish 'mutation applied' from 'perl no-op'. It produced no false KILL here because an unapplied mutation reports SURVIVED, but it is weaker than 'confirm every mutant is PRESENT' intends. Make the marker the mutated text itself.
- N3: the typed-detail asymmetry between target_auth_missing and capability_unavailable is CORRECT — a blanket requirement would be an invented constraint — but unexplained. One sentence so it reads as deliberate.

METHOD
- Confirm every mutant is PRESENT in the file before believing a green or a red, and make the presence check assert the mutated text.
- Do not weaken any existing assertion to accommodate the F1 fix.
- Re-run the gate and report the corrected ratio as a measured number.

spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 and only admitted pair; rework closing an aliasing defect that defeats construction-time validation and an over-claimed clause."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260903-a8cef4, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260903-a8cef4)
REWORK OF CR-TASK-260830-34elja-1 rev1 (changes_requested). Both blocking findings fixed and reproduced before/after; three non-blocking notes addressed. Report: TASK-260830-34elja_rework-rev2.md.

F1 - validated details were aliased. cloneDetails was shallow and Detail returned the live nested container, so ValidateDetails checked a graph the *Error did not own and every Section 15.1 detail bound could be violated after construction through the package own exported accessor. Fixed with cloneDetailValue: deep copy on the way in (New) and on the way out (Detail). decodeBody no longer clones - its map is decoder-allocated and unaliased, and a clone there would have been an unkillable line; the ownership is stated at the assignment. New tests in internal/axerror/ownership_test.go attack containers at three depths and on both sides of an array, so a deep copy NARROWED to one level fails them rather than only a copy deleted outright. The reviewer probe now reports 198 bytes, no password, no int, no depth-7 nest, and Decode of our own output accepts.

F2 - clause 15.1#6 claimed against the wrong actor. SPEC.md:11047 only RFC 2119 sentence binds the provider plugin, which this repository does not build; the host half that LocalFromUntrusted implements carries no RFC 2119 keyword. Clause dropped, section gap now names both undischarged clauses, digest recomputed 5a4f769a -> b6254945, three expectation strings and the README ratios corrected. Section 15.1 6/7 -> 5/7, clauses_discharged 10/394 -> 9/394. No gate outcome moves: 15.1 stays partial and assigned-scope admission still refuses it.

N1 the redundant exit_code leading-quote guard removed (ParseInt over raw bytes already refuses it); refusal table widened from four to nine wrong JSON types and a quote-stripping mutant added. N2 run_mutant now verifies a deletion by the ABSENCE of the deleted text; nine mutants moved to absent mode. N3 the typed-detail asymmetry is now explained in New doc comment. LOGBOOK.md 0753 carries the two entries the reviewer asked for plus the N1 correction.

EVIDENCE, real exit codes: go build 0, go vet 0, gofmt -l 0 files, go test ./internal/axerror -count=1 -v 0, -cover 0 at 99.7%, go test ./... -count=1 0 across 12 packages, tracecheck 0 at 9/394, mutants.sh 26 mutants 26 KILLED 0 SURVIVED 0 MUTANT-ABSENT. Five mutants are new and target F1; three of those five are narrowings.

STATED BOUNDS: internal/axerror still has no production caller anywhere in the repository - leaf 2 owns exit-code wiring and the CLI envelope, so every behavior here is proven at the package boundary and none end-to-end through a command. The clause scanner counts RFC 2119 lines and cannot decide which actor a MUST binds, which is exactly how 15.1#6 passed on rev1; nothing added here closes that class. Redaction bound unchanged: exact key names and verbatim cause containment only. Fuzz targets not re-run.

Commit 4c9efc0 amended in place to ebc4e3197ce0f3334450fe0344a21e4542d4a457, signed, branch exactly one commit past checkpoint d1d3eceb2c42e7a094206465d128272bc81008f5, worktree clean.
STALE CHANGE-REQUEST WINDOW — READ BEFORE REVIEWING. task-board handoff exited 0 but built NO new CR revision. CR-TASK-260830-34elja-1 is still rev1 with base_oid 4c9efc09bded14e5dcdf96ae0191fb2c15e57cf7 and candidate_tree_oid 42868f86e9823b46b5b390efe1a5e60c0cca2f2f, and its attached patch is empty (sha256 e3b0c442... = zero bytes). That base commit was amended away by this rework and is no longer on the branch; it survives only in the object database.

CURRENT HEAD ebc4e3197ce0f3334450fe0344a21e4542d4a457, tree 49e6cb0d375dcb885832d64d9f2723853c7a38d4, signed (git verify-commit: Good "git" signature for oparin@me.com), exactly one commit past checkpoint d1d3eceb2c42e7a094206465d128272bc81008f5, worktree clean. candidate_tree_oid 42868f86 != HEAD tree 49e6cb0d, so the CR window does NOT cover this rework.

The reviewable content is attached directly, since the CR machinery did not snapshot it:
- TASK-260830-34elja_rework-rev2.patch — git diff 4c9efc0 -> ebc4e319, the rework delta only, 9 files, 549 lines. This is what the rev1 review verdict should be checked against.
- TASK-260830-34elja_leaf-full.patch — git diff d1d3ece -> ebc4e319, the whole leaf, 4244 lines.
Both were produced with git diff --binary and are reproducible from the object database; the rework patch applies to 4c9efc0, not to the current tree.

This is the same snapshot-artifact class the rev1 reviewer already diagnosed, one revision later and for a different reason (an amend rather than a checkpoint move). It is an orchestration/tooling condition I cannot repair from a producer run, not a defect in the delivered code. Flagged rather than left for the reviewer to discover.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-a8cef4, pid=94091, exit=0)
REVIEW ROUND 2 — LEAF 1 OF 3

WHICH COMMIT. HEAD ebc4e31. Verify reachability first; refuse as void if unreachable. repository_delta=empty is a snapshot artifact — the checkpoint advanced onto the leaf commit before the candidate was snapshotted; the real delta is the commit.

ROUND 1 BLOCKED ON TWO FINDINGS, BOTH NOW ADDRESSED
F1: cloneDetails was a shallow copy and Detail handed out the live nested container, so ValidateDetails checked a graph the *Error did not own and every Section 15.1 bound could be violated after construction through the package's own accessor.
F2: clause 15.1#6 was claimed at SPEC.md:11047, whose only RFC 2119 keyword binds the provider plugin — a child process this repository does not build.

ORCHESTRATOR VERIFICATION ALREADY PERFORMED — EXTEND, DO NOT REPEAT
- F1: I wrote my own probe from the reviewer's reproduction. Mutating through Detail no longer reaches MarshalJSON. Detail still returns map[string]any, so the fix is a real copy rather than a type change that would have dodged the defect by breaking callers.
- F2: the gate now reports clauses_discharged=9/394 with section 15.1 at partial.

WHAT TO ATTACK
1. IS THE COPY COMPLETE? A deep copy is easy to get almost right. Try nested slices, maps inside slices, maps inside maps at depth, and any type that carries a pointer. Try the INBOUND direction too — mutating the Details map the caller passed to New after construction. F1 was the outbound half; the inbound half is the same defect mirrored, and fixing one is the natural way to miss the other.
2. IS THE COPY PAID FOR ONCE OR ON EVERY ACCESS? If Detail deep-copies per call, a caller in a loop pays for it. Not a correctness issue, but say so if it is a surprise.
3. F2's REASONING GENERALISED. The scanner counts a clause by the single RFC 2119 keyword on its line, and on that line the keyword belonged to an actor this repository does not implement while the host obligation carried no keyword at all. Check the OTHER seven claimed clauses for the same shape: a keyword that binds the plugin, the operator, or any actor outside this repository. One instance was found; the reasoning was applied correctly once (15.1#5) and missed once, so a third instance is plausible.
4. N1/N2/N3 from round 1: the redundant decodeExitStatus guard presented as the fix, the mutant-presence markers asserting an unrelated string, and the unexplained typed-detail asymmetry. Confirm each is addressed or explicitly declined.
5. Confirm nothing accepted in round 1 regressed — the reviewer reported what held; do not let the F1 fix disturb it.

STANDING REVIEW BAR
- Positive-path-only evidence is refused.
- Confirm every mutant is PRESENT in the file before believing a green or a red, and make the presence check assert the MUTATED text.
- A gate reports coverage as a measured ratio; a pin must exercise its claim's SUBJECT; 'nothing is there' is not 'nothing this checker sees'.
- This is a NON-FINAL leaf: it will be checkpointed, not integrated.

spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 and only admitted pair; round-2 review probing copy completeness in both directions and a third instance of the wrong-actor clause shape."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-128-gab60e0d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260903-3b8a74, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260903-3b8a74)
REVIEWER VERDICT ROUND 2 — ACCEPTED (CR-TASK-260830-34elja-2 rev2, run RUN-260903-3b8a74). Full evidence: TASK-260830-34elja_review-verdict-rev2.md. Harness: TASK-260830-34elja_review-mutants-rev2.sh + two logs. Probe: TASK-260830-34elja_review-copy-probe-rev2.go.

ON repository_delta=empty — ACCEPTING AN EMPTY DELTA IS NOT ACCEPTING AN EMPTY DELIVERY. The CR base_oid IS the leaf commit and candidate_tree_oid 49e6cb0d equals both that commit tree and HEAD tree, so the window is zero-width over already-committed work. The delivery is the commit: git diff d1d3ece..ebc4e319 is 20 files +3950/-37, the rework alone 10 files +317/-33. Both reviewed as the real change. Signature verified Good for oparin@me.com, worktree clean, one commit past checkpoint d1d3ece. No repository change would have been the wrong outcome for this leaf and none was made — the code is there and the CR machinery failed to snapshot it twice, for two different reasons (rev1 checkpoint move, rev2 handoff-after-amend). Orchestration defect, not a producer defect, and the producer disclosed it.

BOTH BLOCKING FINDINGS CLOSED, REPRODUCED NOT ACCEPTED.
F1 deep copy: I wrote my own probe against map-in-map, map-in-slice, slice-in-slice and slice-in-map at depths 1-3. Seven inbound attacks through containers the caller retained and seven outbound attacks through Detail, plus the decoded-object arm: encoded bytes unchanged on all fourteen, Decode of our own output byte-identical. The copy is COMPLETE for the admitted set — validateDetailValue admits only nil/bool/string/json.Number/[]any/map[string]any, the four scalars are immutable, both containers recurse. I attacked the one hole a Go type switch leaves, a NAMED map type falling through cloneDetailValue default arm as a live alias: axerror.Details itself is such a type and is refused upstream. Detail deep-copies PER CALL (distinct allocations), noted as a cost surprise not a defect.
F2 wrong actor: clause dropped, gap names #5 and #6 on the same reasoning, digest b6254945, and 5/7 + 9/394 is consistent in all four published places (main_test.go:26 and :59, traceability_test.go:626/628, README:1467/1474). I re-read SPEC.md:11047 myself — its only RFC 2119 sentence binds the plugin, the host half carries no keyword.

NO THIRD WRONG-ACTOR CLAUSE. I read every claimed line and named the actor its keyword governs: 11002 emitter, 11003 emitter, 11010 readers, 11019 emitter, 11053 receivers, 11121 reader, 11133 implementations — every one an actor this repository implements. F2s shape was the lines ONLY keyword binding a foreign actor; none of the seven survivors has it.

MUTATION: 41 mutations applied and compiled, 39 KILLED, 2 survivors, preferring narrowings over deletions. Deep copy narrowed to one level / maps only / arrays only; depth 4->5; canonical 16KiB->64KiB; keys 64->128; grammar {0,63}->{0,127}; message 4096->8192 and 1->0; one credential key removed; nested excluded-key walk removed; causal scan restricted to the outermost link and blinded to detail values and to maps; major gate widened to admit major 2; version match reduced to majors; DisallowUnknownFields removed; trailing content admitted; exit/code contradiction admitted; success status 0 admitted as a failure class; exit-7, exit-16 and operation_uncertain retryability classes each removed separately; per-version code admission removed; typed-detail non-empty narrowed to presence-only; bootstrap version pin drifted; unknown outcome mapped to the fallback code — all killed. I DISCARDED one mutant (M29) as a compile failure rather than scoring it a kill; the producer harness has no build gate and would have scored it KILLED. Producer 26-mutant harness re-run by me on a clean tree: 26 KILLED 0 SURVIVED 0 MUTANT-ABSENT, two ambiguous ones hand-checked to confirm they compile.

WHAT ELSE HELD, reproduced: build/vet/gofmt clean; go test ./... exit 0 across 12 packages; coverage 99.7% MEASURED BY ME not accepted from the log; tracecheck exit 0 at 9/394. Causal redaction — could not get an unredacted cause onto the wire; cause unexported, wireError has no field for it, both Unwrap arms walked. Digest pin IS recomputed: one added full stop in a gap string reddens with be328132 differs from b6254945. AND THE GATE CHECKS THE CLAIM NOT ONLY THE PIN — I pointed clause 15.1#1 at line 11004 with the digest blanked and it refused with declares line 11004, but the pinned clause is at line 11002, so a claim cannot be relocated to a friendlier sentence even by someone who recomputes the pin. Section 15.2 table re-derived from SPEC.md:11077-11094 by parser: every row matches exactly. 17.2 NOT rebound — still EncodeCurrent with a gap saying it discharges nothing. 14.2 correctly unclaimed: its operative sentence is the process exit status, which is leaf 2, and no catalog row requires a 14.2 binding. decodeBody dropping cloneDetails is right, not a regression: the map is decoder-allocated inside decodeClosedDocument, json.RawMessage copies rather than aliasing the input, and the outbound arm is still covered. decode_test change WIDENS a refusal table 4 rows -> 9. N1/N2/N3 all addressed and verified, including a quote-stripping mutant that compiles and is killed.

TWO NON-BLOCKING FINDINGS.
R1 the causal-redaction sensitivity bound is undefended: minRedactableCause = 8 decides which causes are scanned at all, and raising it to 16 OR to 64 leaves the suite green — only 4096 reddens, because every cause in the corpus is >=64 chars. An edit widening it to 63 silently stops scanning a class of short causes. RedactionBound does not state it either. Wants one case at exactly the bound that must be refused and one just under it that must be admitted.
R2 a new prose count that contradicts the measured table: README:1248 and registry.go:64 both call Section 15.2 the nineteen-row table; SPEC.md:11077-11094 has EIGHTEEN rows (0, 2-17, 130). Implementation and its reviewed literal are both correct at 18 and compared by length and content, so nothing is broken — only the sentence. It repeats a miscount already in README:1426, traceability.go:155 and the BUG-260902-2m7slg artifacts, so fix it at source. This is the boards own rule applied to itself: a count stated as prose drifts from the count that was measured.

STATED BOUNDS. internal/axerror still has NO production caller anywhere — no .go file outside the package imports it; every behavior is proven at the package boundary and none end-to-end through a command. Expected for this leaf, disclosed not inferred away. The clause scanner counts RFC 2119 lines and cannot decide WHICH ACTOR a MUST binds; nothing here closes that class and my actor table is a human read, not a machine check. No fuzz targets exist in the package; none run. I did not re-verify the 15-row binding table row by row this round — binding.go is untouched by the rework and that is an accepted round-1 result, not a round-2 measurement. Working tree left byte-identical to HEAD; all mutations restored from a private byte backup, never git checkout.

FOR THE ORCHESTRATOR: the CR snapshot window has now failed twice on this leaf for two different reasons and both times the delta had to be reconstructed by hand — worth a board item of its own. R1 and R2 are cheap and belong to this package; R2 should be fixed at its source.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260903-3b8a74, pid=17649, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-34elja_spawn-log_-implementer--developer--claude-_RUN-260903-cc90a1.log](file://TASK-260830-34elja/TASK-260830-34elja_spawn-log_-implementer--developer--claude-_RUN-260903-cc90a1.log) — System spawn log captured by task-board
- [TASK-260830-34elja_results.md](file://TASK-260830-34elja/TASK-260830-34elja_results.md) — Rev1 Structured Error implementation report, with a header marking the four claims superseded by TASK-260830-34elja_rework-rev2.md (15.1 5/7, 9/394, 26 mutants, leading-quote guard removed)
- [TASK-260830-34elja_mutation-01.log](file://TASK-260830-34elja/TASK-260830-34elja_mutation-01.log) — 26/26 mutants killed over the Structured Error gates including five new detail-ownership mutants, three of them narrowings; every mutant verified applied before measurement
- [TASK-260830-34elja_mutants.sh](file://TASK-260830-34elja/TASK-260830-34elja_mutants.sh) — 26-mutant harness for internal/axerror: run_mutant now verifies a deletion by the absence of the deleted text and an edit by the presence of the written text; refuses a red baseline and re-verifies after each restore
- [TASK-260830-34elja_go-test-axerror-01.log](file://TASK-260830-34elja/TASK-260830-34elja_go-test-axerror-01.log) — Verbose go test ./internal/axerror -count=1 run after rework, exit 0
- [TASK-260830-34elja_go-test-all-01.log](file://TASK-260830-34elja/TASK-260830-34elja_go-test-all-01.log) — Repository-wide go test ./... -count=1 after rework, exit 0, 12 packages
- [TASK-260830-34elja_tracecheck-01.log](file://TASK-260830-34elja/TASK-260830-34elja_tracecheck-01.log) — Ownership gate after dropping clause 15.1#6: clauses_discharged 9/394, section 15.1 at 5/7, exit 0
- [TASK-260830-34elja_change-request_rev1.patch](file://TASK-260830-34elja/TASK-260830-34elja_change-request_rev1.patch) — Change Request CR-TASK-260830-34elja-1 revision 1 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260830-34elja_change-request_rev1-validation.log](file://TASK-260830-34elja/TASK-260830-34elja_change-request_rev1-validation.log) — Change Request CR-TASK-260830-34elja-1 revision 1 bounded validation log
- [TASK-260830-34elja_spawn-log_-reviewer--reviewer--claude-_RUN-260903-737126.log](file://TASK-260830-34elja/TASK-260830-34elja_spawn-log_-reviewer--reviewer--claude-_RUN-260903-737126.log) — System spawn log captured by task-board
- [TASK-260830-34elja_review-verdict.md](file://TASK-260830-34elja/TASK-260830-34elja_review-verdict.md) — Reviewer verdict for CR-TASK-260830-34elja-1 rev1: changes_requested. F1 shallow cloneDetails + Detail() alias lets a validated object emit password/access_token and blow the 16KiB/depth-4/type bounds onto the wire (reproduced; its own reader refuses the output). F2 clause 15.1#6's MUST NOT binds the provider plugin this repo does not build, inflating 15.1 to 6/7 and clauses_discharged to 10/394. 20/20 producer mutants and 11/12 reviewer mutants reproduced; digest pin verified recomputed; 17.2 correctly not rebound.
- [TASK-260830-34elja_review-mutants.sh](file://TASK-260830-34elja/TASK-260830-34elja_review-mutants.sh) — Reviewer's 12 independent narrowing mutants over internal/axerror gates the producer harness did not cover; 11 killed, the survivor is the redundant leading-quote guard in decodeExitStatus (note N1)
- [TASK-260830-34elja_spawn-log_-implementer--developer--claude-_RUN-260903-a8cef4.log](file://TASK-260830-34elja/TASK-260830-34elja_spawn-log_-implementer--developer--claude-_RUN-260903-a8cef4.log) — System spawn log captured by task-board
- [TASK-260830-34elja_rework-rev2.md](file://TASK-260830-34elja/TASK-260830-34elja_rework-rev2.md) — Rework report for CR rev1 changes_requested: F1 detail-graph ownership, F2 clause 15.1#6 dropped, N1-N3, 26/26 mutants, real exit codes and stated bounds
- [TASK-260830-34elja_probe-after-fix-01.log](file://TASK-260830-34elja/TASK-260830-34elja_probe-after-fix-01.log) — Reviewer's F1 reproduction probe re-run against the fix: 198 bytes, no password, no int, no depth-7 nest, Decode of our own output accepts
- [TASK-260830-34elja_go-test-axerror-cover-01.log](file://TASK-260830-34elja/TASK-260830-34elja_go-test-axerror-cover-01.log) — Measured coverage run after rework: 99.7% of statements, exit 0
- [TASK-260830-34elja_rework-rev2.patch](file://TASK-260830-34elja/TASK-260830-34elja_rework-rev2.patch) — REVIEWABLE DELTA of this rework: git diff 4c9efc0 (reviewed rev1 commit) -> ebc4e319 (amended head). 9 files, 549 lines. The board CR was NOT rebuilt by handoff and still points at the pre-amend tree.
- [TASK-260830-34elja_leaf-full.patch](file://TASK-260830-34elja/TASK-260830-34elja_leaf-full.patch) — Full leaf delta: git diff checkpoint d1d3ece -> ebc4e319, 4244 lines, the whole internal/axerror package plus its traceability and documentation changes
- [TASK-260830-34elja_change-request_rev2.patch](file://TASK-260830-34elja/TASK-260830-34elja_change-request_rev2.patch) — Change Request CR-TASK-260830-34elja-2 revision 2 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260830-34elja_change-request_rev2-validation.log](file://TASK-260830-34elja/TASK-260830-34elja_change-request_rev2-validation.log) — Change Request CR-TASK-260830-34elja-2 revision 2 bounded validation log
- [TASK-260830-34elja_spawn-log_-reviewer--reviewer--claude-_RUN-260903-3b8a74.log](file://TASK-260830-34elja/TASK-260830-34elja_spawn-log_-reviewer--reviewer--claude-_RUN-260903-3b8a74.log) — System spawn log captured by task-board
- [TASK-260830-34elja_review-verdict-rev2.md](file://TASK-260830-34elja/TASK-260830-34elja_review-verdict-rev2.md) — Round-2 reviewer verdict (CR rev2, run RUN-260903-3b8a74): ACCEPTED, with two non-blocking findings and stated bounds
- [TASK-260830-34elja_review-mutants-rev2.sh](file://TASK-260830-34elja/TASK-260830-34elja_review-mutants-rev2.sh) — Reviewer round-2 mutation harness: 41 mutations, build-gated, presence/absence-verified, byte-backup restore
- [TASK-260830-34elja_review-mutants-rev2-01.log](file://TASK-260830-34elja/TASK-260830-34elja_review-mutants-rev2-01.log) — Reviewer round-2 mutation batch 1 (34 mutations)
- [TASK-260830-34elja_review-mutants-rev2-02.log](file://TASK-260830-34elja/TASK-260830-34elja_review-mutants-rev2-02.log) — Reviewer round-2 mutation batch 2 (13 mutations, repairs plus new narrowings)
- [TASK-260830-34elja_review-copy-probe-rev2.go](file://TASK-260830-34elja/TASK-260830-34elja_review-copy-probe-rev2.go) — Reviewer deep-copy attack probe: inbound and outbound aliasing at four container shapes and three depths
- [TASK-260830-34elja_review-copy-probe-rev2-01.log](file://TASK-260830-34elja/TASK-260830-34elja_review-copy-probe-rev2-01.log) — Deep-copy probe output: every inbound and outbound mutation leaves the encoded object unchanged
- [TASK-260830-34elja_review-final-validation-rev2-01.log](file://TASK-260830-34elja/TASK-260830-34elja_review-final-validation-rev2-01.log) — Reviewer-run build/vet/gofmt/test/cover/tracecheck on the restored tree at ebc4e31

## Created
2026-08-29T21:59:57Z

## Last Update
2026-09-03T04:23:29Z

## Assigned To
[reviewer] reviewer (claude)
