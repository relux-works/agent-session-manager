## Status
to-review

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(3))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] An endpoint beginning with a hyphen is refused at the production Load entry, with a case per attack shape from the audit probe
- [x] An endpoint containing whitespace is refused, or the field is required to match a [user@]host[:port] grammar and that grammar is cited
- [x] A dangling -o with no operand is refused
- [x] The short-flag tables carry the same both-directions key-set pin the -o registry has, so a mutant permitting an extra letter reddens
- [x] Each case reddens when only its own clause is weakened, proven by mutant
- [x] Repository tests, vet, build, tracecheck and catalog check exit 0 with no coverage regression
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
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f96d39051c2687a978ffea4bcd9a3b81466b5b1dae5cd7c8a91ad18250148a4c rationale="Rank-1 recommendation: close option injection through a validated field, in internal/config so it cannot collide with the record-schema leaf working in canonicaljson."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-fc0fe2, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-fc0fe2)
Endpoint option injection closed. internal/config/endpoint.go admits mesh.peers[].endpoint against a closed [user@]host[:port] grammar, wired as one refusal site at validation.go:453 inside validateMesh, clause mesh.peers[N].endpoint <reason>. Leading hyphen and whitespace are refused first by name; user is 1-64 [A-Za-z0-9._-]; host is LDH DNS labels or a bracketed IPv6 literal (bare IPv6 refused - colons are indistinguishable from the port separator); port is 1-5 digits in 1-65535.

CORRECTION to the report: the dangling -o was NOT open. BUG-260902-beqfwr already closed it (admitSSHArguments returns sshRefusalMissingFlagValue). It is now pinned per declared value-taking letter (-i -l -p -o, alone, grouped as -q<letter>, and trailing) rather than left as two incidental cases. Reported, not claimed as a fix.

Short-flag asymmetry closed: permittedSSHFlagSamples and permittedSSHValueFlagSamples carry the both-directions key-set pin permittedSSHOptionSamples gave the -o registry, plus named refusals for -A -E -L -R -D -W -w -S -O.

Regression fixed: schema_test.go proved the 1024-character endpoint bound with a non-ASCII filler the grammar refuses; rewritten as a grammar-valid name of the same length, still proved at 1024 accepted / 1025 refused.

16 single-clause mutants, 16 RED, 0 survived - including two narrowing mutants that prove the admitted half is not vacuous. Gates all exit 0: gofmt -l ., go build ./..., go vet ./..., go test ./... -count=1, go test ./... -cover, tracecheck (global and -section 6.1..6.5), cataloggen -check. internal/config coverage 94.4% (baseline measured at checkpoint 10aaa16) -> 94.6%.

Commit 52ad979, one commit past checkpoint 10aaa16, signed and verified.

NOT DONE: no -- separator added to a launcher; no launcher exists in this repository yet. The grammar refuses %zone IPv6 scope IDs, trailing-dot FQDNs and IDN hostnames - legal SSH destinations in some deployments, none in the pinned spec example set, widening is a reviewed change.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-fc0fe2, pid=4856, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-opus-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:5c7b2b85386ee880001c8abd4df995e0cd99607f2ac881eb510e68dfeaca1776 rationale="Rank-1 recommendation: attack the endpoint admission with injection shapes the producer did not enumerate, and verify against a live ssh binary how each would actually be parsed."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260902-fed31a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260902-fed31a)
Review ACCEPTED. CR revision 1 declared repository_delta=empty, but base OID 52ad979 IS the leaf commit carrying the change (base tree == candidate tree == HEAD tree == 0ed1c6c), so the snapshot window opens after the commit instead of at checkpoint 10aaa16. Real delta 10aaa16..52ad979 = 7 files, +571/-2, reviewed in full. Gate wired at validation.go:452, the only non-test caller, on the Load path. 17 reviewer mutants applied to production files only: 16 killed, M13 (redundant empty-host guard) proven an equivalent mutant. Reproduced the disclosed short-flag asymmetry as real - the -A mutant survives against checkpoint tests and is killed by the new both-directions key-set pin. Independent 34-shape hostile-endpoint probe: all refused, incl. -ivan@peer.example which the grammar alone would admit without the leading-hyphen clause. 12 dangling value-flag spellings refused. build/vet/gofmt/go test ./.../tracecheck global+6.1-6.5/cataloggen -check all exit 0; internal/config coverage 94.4% (measured at 10aaa16 via git archive) -> 94.6%. Non-blocking: CR base construction, redundant M13 guard, whitespace-ownership assertion weaker than its comment (claim independently verified true), trailing-dot FQDN narrowing. Orchestrator owns the done transition with commit_ack=scope_committed.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-fed31a, pid=29567, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260902-3c7ovg_spawn-log_-implementer--developer--claude-_RUN-260902-fc0fe2.log](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_spawn-log_-implementer--developer--claude-_RUN-260902-fc0fe2.log) — System spawn log captured by task-board
- [BUG-260902-3c7ovg_results.md](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_results.md) — Endpoint option-injection fix: reproduction, grammar, 16-mutant matrix, gate exit codes
- [BUG-260902-3c7ovg_mutant-matrix.log](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_mutant-matrix.log) — 16 single-clause mutants, all RED, with the reddened case list per mutant
- [BUG-260902-3c7ovg_audit-probe-before.log](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_audit-probe-before.log) — Pre-fix probe at the production Load entry: six endpoint injections with err=nil
- [BUG-260902-3c7ovg_change-request_rev1.patch](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_change-request_rev1.patch) — Change Request CR-BUG-260902-3c7ovg-1 revision 1 candidate patch (repository_delta=empty, 0 changed paths)
- [BUG-260902-3c7ovg_change-request_rev1-validation.log](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_change-request_rev1-validation.log) — Change Request CR-BUG-260902-3c7ovg-1 revision 1 bounded validation log
- [BUG-260902-3c7ovg_spawn-log_-reviewer--reviewer--claude-_RUN-260902-fed31a.log](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_spawn-log_-reviewer--reviewer--claude-_RUN-260902-fed31a.log) — System spawn log captured by task-board
- [BUG-260902-3c7ovg_review-verdict.md](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_review-verdict.md) — Reviewer verdict: ACCEPTED; empty-delta explained as CR window artifact, 17 reviewer mutants, 34-shape attack probe, independent gate and coverage re-run
- [BUG-260902-3c7ovg_reviewer-gates.log](file://BUG-260902-3c7ovg/BUG-260902-3c7ovg_reviewer-gates.log) — Reviewer re-run of repository gates: go test ./..., tracecheck global and 6.1-6.5, cataloggen -check, internal/config coverage

## Created
2026-09-02T12:00:22Z

## Last Update
2026-09-02T13:01:05Z

## Assigned To
[reviewer] reviewer (claude)
