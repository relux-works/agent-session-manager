# TASK-260830-2x16gz recovery prerequisite audit

Run: RUN-260907-de0af8, autonomous recovery attempt 1 of 3.
Disposition: blocked by unchanged cross-owner prerequisite, not ready for review.

## Current evidence

The initial required status command changed blocked to development. No new
operator directive or revised ownership authorization exists. The live downstream
task TASK-260830-z1yxg9 (implement-rpc-envelope-and-hello) is to-dev and still
blockedBy TASK-260830-2x16gz, TASK-260830-treeox and TASK-260830-33sfxc.
Thus a producer retry did not resolve the previously reported dependency cycle.

Both accepted predecessor CR1 outcomes and review verdicts were downloaded and
read. The exact Git checkpoints match their accepted trees:

| Leaf | Signed checkpoint | Accepted tree |
| --- | --- | --- |
| peeridentity | 43c0e2b9a33f0b08ee9a6a6f8740cc0a68dd8792 | c775903a1aa76cb3d433aeee89ebfd24e73159dd |
| sshtransport | c1eff016dce2e55c4e2c1828d5c20f3da84118bc | f1dbf8f2dcd80ec8fdee9c0e7e1822cd921d41ac |

Transport's parent is the identity checkpoint. Both git verify-commit commands
exit 0. HEAD is the transport checkpoint, HEAD..main count is 0. This is local
ref evidence, not a claim that remote main was fetched. Worktree and index are
unchanged. No checkpoint refresh, commit, integration or publication attempted.

Pinned source: AX v0.5.0 at 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c;
internal/specdoc/SPEC.md is validated by the focused specdoc tests. Section 6.3
requires allowlisted identity plus authenticated endpoint; section 11.1 requires
both sides to check protocol host_id after SSH starts the remote process.
Section 16.1 delegates transport authentication/integrity/confidentiality to SSH.
The assignment explicitly excludes implementing downstream hello here.

Source caller inspection confirms CheckProtocolHost and DisclosurePolicy have
only test callers; neither has a runtime admission/publishing caller. Open owns
process start, not authenticated operation admission. This agrees with both
accepted reviews and existing README/traceability gaps, rather than contradicting
accepted predecessor behavior.

## Seven-case accounting

**3 of 7 AC rows driven at the existing local configuration/stream boundary** in
this rerun (unknown selectors, disconnect propagation, oversize lines).
This is not 7/7 hostile-network conformance. Four rows remain unproven at the
requested boundary; comparator/configuration tests are conditional evidence.

| Case | Actual production path and named committed driver | Control/recovery and remaining bound |
| --- | --- | --- |
| Unknown peers | Client.Open -> Directory.Resolve; TestOpenRefusesBeforeStart; TestResolveAllowlistAndAliasAmbiguity | Known configured targets start in TestOpenStructuredCommand; unknown selectors refuse before start. Inbound authenticated hello remains downstream. |
| Key changes | Client.Open -> native SSH; TestOpenNativeSSHPolicy supplies effective-config evidence only | Actual synthetic old/new-key handshake driver is still missing; fake exit255 does not count. Known-key success and fresh corrected-key recovery required. |
| Spoofed host IDs | Target.CheckProtocolHost; TestProtocolIdentityRefusal | Exact ID passes; other allowed ID/alias refuses in comparator only. Both-side runtime admission is missing and explicitly owned downstream. |
| Disconnects | Session.Receive/Send/Wait -> supervise/readLines; TestReceiveFailureAndLimits; TestReadFailureAndRecovery | Failed reads/partial lines/process exits refuse and fresh session recovers; local process/stream proof, not encrypted disconnect or transaction rollback. |
| Replay | Client.Open -> external SSH integrity | No committed cryptographic replay driver; not proven. Needs ephemeral native SSH capture/reinject fixture with ordinary traffic and reconnect controls. Application operation replay is separately owned and may legitimately be idempotent. |
| Oversized frames | Session.Send/readLines through Receive; TestDuplexAndSendBoundaries; TestReceiveFailureAndLimits | Exact 8 MiB neighbor passes; +1 refuses and terminates; fresh sessions succeed. JSON/negotiated framing remains downstream. |
| Disclosure mismatches | Directory.DisclosurePolicy; TestDisclosureClassPolicyBinding; TestDisclosurePolicyRefusals | Per-class configured allow/deny controls pass; policy output is not authenticated publication or payload sanitization. Scope interpretation requires primary resolution. |

No new tests/code or durable mutation were introduced. No crash/idempotency
claim is made. README, doctor and ownership claims were preserved because
they already state these boundaries.

## Validation performed by this recovery run

Platform inspected: Go project supporting macOS/Linux/WSL2/Windows, native
macOS arm64; no iOS target. Tools were readiness-checked; logs are in this
directory. No installations or private credentials were used.

Every command in commands.json ran directly, with redirection to a full log,
without tee/pipeline status masking. Each exited 0, including:

- go test ./internal/peeridentity ./internal/sshtransport ./internal/specdoc -count=1 -v
- go build ./...
- both git verify-commit checks
- git status --porcelain=v1, git diff --cached --exit-code HEAD, git diff --exit-code HEAD

All three selected packages reported ok with no top-level skips. Full tests,
coverage, race, lint, fuzz and configured full gates were not rerun: no
repository delta occurred and the unchanged ownership conflict stops product
work. Predecessor full-suite evidence is inherited, not this run's verification.
No new checklist claim is made for those commands.

No mutants were rerun during this recovery audit. The prior blocked outcome,
included in the evidence archive, retains the full named mutant table, controls,
real exits and one subsumed survivor. Those are prior-run observations, not new
kills; this recovery adds no mutation coverage.

Read/discovery errors: worktree-local skill path absent (exit 1), recovered by
reading the explicitly assigned main-checkout Curator skill. Initial board
queries used unknown fields resources/blockers and unknown operation resources;
they returned errors, then were repaired using installed schema and exact
outcomeResources/blockedBy fields. No failed read was treated as absence.

## Decision needed from primary

Recommended: resolve dependency ordering so the existing RPC owner can implement
real bilateral hello admission against the accepted signed peeridentity and
transport checkpoints, then return this task for all seven conformance rows.
Primary must select the supported managed Story route and preserve the valid
checkpoints; this worker cannot integrate an incomplete Story.

Alternative: explicitly transfer minimal runtime admission ownership here and
revise the downstream scope consistently. This widens current scope and risks
duplicate handshake ownership, so it requires an explicit owner decision.
Reducing this task to configuration/comparator tests does not meet the requested
seven cases and is not recommended.

Also settle whether disclosure mismatch acceptance refers to the section 6.4
configuration policy boundary or authenticated publishing. Neither interpretation
removes the concrete spoofed-host runtime prerequisite.

Native listener-free process fixtures for key-change/replay remain viable work;
they were not attempted after the ownership stop and are not claimed impossible.
Do not automatically rerun the same unchanged producer brief as a remedy.
No upstream issue #176/#177 implementation, new worker or manual board edit.
