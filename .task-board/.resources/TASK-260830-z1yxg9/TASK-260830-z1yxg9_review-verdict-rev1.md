# Review verdict — CR-TASK-260830-z1yxg9-1 revision 1 (ACCEPT)

Reviewer run: RUN-260917-00d532 (reviewer). Candidate tree
9d0c99b0669444ae77677348f54f1b4b1d791881 over base
ae5a4cc5394ef80fc2fc3ef12965acc0670cb5a4 — 30 paths, no board files.
Normative authority: spec v0.6.0 §§11.10/11.10.1–11.10.5, 11.9,
11.2/11.3, 15.1/15.3. Credential owner hosttrust accepted under
CR-TASK-260909-2ez769-12 (not re-litigated; candidate consumes it).

Method: isolated immutable probes only. Candidate bytes materialized with
`git archive <candidate-oid>` to /private/tmp scratch; mutation probes via
`go -overlay` or scratch-copy edits; the live Story worktree, index,
branch and HEAD were never mutated; no product edits, commits,
checkpoints, or integration. PYTHONDONTWRITEBYTECODE=1 for every harness.

## Verdict: ACCEPT (no P1; two P3 hardening notes in §8)

## 1. Restoration — verified, faithful

- Packet manifest: all 15 packet bytes match manifest sha256 (recomputed).
- Candidate fixtures contracts-v3/v4 and hello-request/response-v2 match
  manifest digests byte-for-byte.
- rpcwire code/README extended for RPC 5.0.0; mutations.py extended by the
  declared v5-exact probe; contracts-v5.json added and differs from v4 by
  exactly the `"rpc": ["5.0.0"]` line.
- UNRESOLVED_QUESTIONS.md dropped: faithful — it recorded the inbound
  identity association as open and v0.6.0 §11.10 resolves it (mutual TLS
  1.3 over verified enrolled certificates before hello/dispatch); the
  resolution is recorded in both READMEs, LOGBOOK and results.
- Tracked deltas: README section appended (15.1 5/7, 15.3 2/3 unchanged,
  no new discharge claimed); ownership gaps rewritten with coverage
  levels and clause enumerations unchanged; traceability digest
  5102d3bc… (tracecheck green); task-board.config.json adds exactly one
  fuzz-gate line, no other field touched.

## 2. AC accounting — 7 of 7 rows driven through production entries

All named tests exist in the candidate and pass (full suite green):

| # | Row | Production call sites | Named tests (confirmed present+green) |
| --- | --- | --- | --- |
| 1 | Request/response correlation | Dial + Call + Serve | TestFullHandshakeHelloAndOperation, TestHelloResponseCorrelation |
| 2 | Hello maps and nonce echo | Serve/Dial hello exchange | TestFullHandshakeHelloAndOperation, TestRefusesInvalidHello/* |
| 3 | Namespace/cardinality | Call("inventory.roots") dispatch | TestInventoryRootsThroughChannel |
| 4 | Offered limits | hello validation both roles | TestFullHandshakeHelloAndOperation, TestOversizeHelloLine |
| 5 | Structured Error framing | Serve refusal frames, Dial mapping | TestRefusesNonHelloBeforeHello, TestHelloFailureMapping/* |
| 6 | Bilateral admission | AuthorizeDispatch at hello both roles | TestRefusesHelloUUIDMismatch, TestRefusesServerHelloUUIDMismatch, TestRefusesNotAllowlisted, TestRefusesDestinationMismatch |
| 7 | Deadlines/generation | bounded I/O, stale gates, WatchGeneration | TestHelloDeadline/*, TestHelloWriteDeadline, TestDispatchTimeout, TestStaleGenerationAtDispatch/*, TestStaleGenerationAtMutation, TestDispatchExpiryClosesWithoutFrame, TestWatchGeneration |

Crash/idempotency: the channel writes no durable state — proven by static
census (TestStaticWriteCensus) + runtime census (TestRuntimeWriteCensus:
state trees, trust bytes, temp tripwire); trust commits stay atomic in
hosttrust. No crash-recovery behavior attested, correctly so. No
unsupported capability advertised: docs state no `ax` command, no SSH
process launch, no listener, no provider plugin, no object exchange;
launch selection is a computed Config-4 argv only.

## 3. Conformance (code read in full: channel/server/client/launch)

TLS 1.3 only (shared Min=Max builder), NextProtos exactly [ax-host/1],
no tickets, nil client cache, enrolled-roots-only pools (no system pool
in non-test code), VerifyConnection supplementing (never replacing)
standard verification, UUID-SAN ServerName, 65 535-byte authority bound,
10 s handshake/hello deadlines enforced by stream close, 1 MiB handshake
cap, 8 MiB line / 5 MiB object limits, no app bytes before handshake,
hello success validated before any other op, current-generation dispatch
and WithMutationAuthorization boundaries, exporter/key-log material never
persisted. Refusal semantics exact: ≤1 encrypted Error 1.3.0 then close
for pre-hello violations (host_identity_mismatch/peer_not_allowlisted
exit 7, incompatible_protocol exit 6 only when known), silent close for
unframeable/foreign-major input, TLS-alert-only pre-handshake,
authentication_failed 7 / transport_failure 8 / invalid_config 3 on the
initiator. axerror diff adds only the {rpc,5}→1.3.0 row (unbound probe
5→6); no new enum or exit code. HC-MIGRATE takes Host Channel 1 + RPC 5
only from Config-4 and refuses legacy; §11.10.5 correctly not
implemented as a runtime policy reader. HC-PARITY is a stated bound
(abstract Stream, no SSH branch — confirmed by grep); same-profile tests
run over the agnostic Stream.

## 4. Independent mutant evidence (all rerun by reviewer, per-plant logs
in the evidence tarball)

- rpcwire harness (isolated copy): 1 neutral passed + 34 narrowing KILLED,
  incl. v5-exact and the token-preserving common-data-model probe (killed
  behaviorally by TestFrameRefusals/duplicate).
- hostchannel harness (isolated copy): 1 neutral passed + 22 narrowing
  KILLED + census-evasion KILLED behaviorally by TestRuntimeWriteCensus
  (passes the static census by design) + 1 documented SURVIVED
  (client-cache, see §5).
- Reviewer-owned narrowing mutants on arms the producer did not mutate
  (scratch copy, restored to pristine after each; raw logs attached):
  M1 ALPN admits exactly "ax-host/2" — KILLED by new scratch vector
  TestOwnRefusesAxHost2ALPN while TestVerifyPeerRefusals/alpn stays
  green on the mutant (proves narrowing; the existing suite alone would
  not catch this weakening); M2 client generation admits exactly +1 —
  KILLED by client_refuses_to_send; M3 ServerName skips UUID parse —
  KILLED by bad_destination; M4 helloRefusal maps mismatch→protocol —
  KILLED by host_identity_mismatch mapping test; harmless comment-only
  control passes (SURVIVED as expected).
- KILLED stability: hostchannel suite green 3× (initial, -race, stability
  rerun: 98 subtests pass), rpcwire green 4×+, full repo suite
  `go test ./... -count=1`: 31 packages ok, exit 0.

## 5. The client-cache SURVIVED — accepted as harmless, correctly bounded

A caching client cannot resume against a server that issues no tickets,
so no behavioral test through the public API can distinguish the mutant:
there is no cache-injection hook and nothing is ever stored. The
complementary arm (server issues tickets) IS killed by
TestRefusesResumption via the `tickets` probe. One-sided weakening with
the pair covered — accept.

## 6. Gates run green here

`go test ./internal/rpcwire ./internal/hostchannel -race -count=1`: ok.
FuzzUntrustedEnvelopes 200x: PASS (200 execs). Coverage reproduced
exactly: hostchannel 81.6%, rpcwire 97.8%. `go vet`: clean. `gofmt -l`:
empty. `go build ./...`: clean (via full test compile). tracecheck:
exit 0. cataloggen -check: exit 0. Candidate tree carries zero
__pycache__/.pyc entries.

## 7. Reran-vs-accepted

Reran myself: everything in §§1–6 (restoration sha, fixture diffs,
config/axerror diffs, both harnesses in full, 4 own mutants + control,
full suite, race, fuzz 200x, cover, vet/gofmt, tracecheck, cataloggen,
forbidden-pattern greps, 27/27 named-test existence). Accepted from
attached evidence (spot-checked, not re-executed byte-for-byte):
producer per-probe raw logs (format confirmed present in evidence zip,
70 files) and the validation log's board-health tail (pre-existing
MISSING_ACTIVITY noise, exit 0 — not candidate-related).

## 8. P3 hardening notes (non-blocking)

- P3: add an "ax-host/2" same-family ALPN case to
  TestVerifyPeerRefusals — M1 proved the committed suite only refuses the
  unrelated "rogue-alpn" token while the supplement enforces exact
  equality. (Reviewer scratch vector TestOwnRefusesAxHost2ALPN
  demonstrates the test; not added to the candidate — reviewer makes no
  product edits.)
- P3: TestInvalidActivation/bad_destination takes ~10 s (real handshake
  deadline) — consider a shortened deadline hook for that vector, as
  already done elsewhere.

## 9. Handoff state

Candidate left UNCOMMITTED in the Story worktree; no commit, checkpoint,
integration, or landing performed by this run. Live `git status` shows
only the candidate paths plus pre-existing worktree `.task-board`
checkout noise (diagnosed: the 34 board paths are absent from the
candidate tree itself — verified via `git diff base candidate --stat`:
exactly the 30 declared files).
