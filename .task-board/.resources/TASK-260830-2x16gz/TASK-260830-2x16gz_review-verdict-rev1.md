# Review verdict — CR-TASK-260830-2x16gz-1 revision 1 (ACCEPT)

Reviewer run: RUN-260917-f36c51 (reviewer, muse-spark xhigh).
Candidate tree `1d03fbc7fe3ecd73347eb424bc5de1ce94df7370` over trunk base
`e4e3e8834675cf3814effd3b2673b4931dfbf311` — 119 changed paths (whole-Story
delta: four replayed checkpoints plus this test-only leaf).
Normative authority: pinned `internal/specdoc/SPEC.v0.6.0.md` §§6.6, 11.10
(11.10.1–11.10.5, AC-HOST-001), 11.1, 16.1, 15.1/15.3.

Method: isolated immutable probes only. Live Story worktree
(`.temp/STORY-260830-1kiyj6/worktree`, HEAD `c3de159`) was used only to
execute tests; no product edit, commit, checkpoint, or integration by this
run. Mutants ran under `go -overlay` (sources verified byte-identical
after each plant) or in `/tmp` scratch. `PYTHONDONTWRITEBYTECODE=1`
throughout. Producer brief `TASK-260830-2x16gz_producer.md`, republish
brief `..._republish-rev1.md`, outcomes `..._results.md` /
`..._results-rev2.md`, matrix `..._conformance-matrix-rev2.md`, predecessor
verdicts (z1yxg9 rev1, 2ez769 rev12), and the CR patch bytes were read.

## Verdict: ACCEPT (no P1, no P2; two P3 follow-ups in §9, non-blocking)

## 1. Whole-Story diff — verified faithful

- Replayed checkpoints `14d1176` (2u34k1), `4677a87` (1tvg8e), `a14f718`
  (2ez769), `c3de159` (z1yxg9): all `git verify-commit` Good
  (`oparin@me.com`); parent chain reaches `e4e3e88` (`merge-base
  --is-ancestor` ok); `git log e4e3e88..c3de159` shows exactly the four
  Story commits in order.
- Replay fidelity: `git diff <original> <replayed> --name-only` for all
  four pairs equals exactly the 28-file trunk delta `62d4463..e4e3e88`
  (diffed both directions, empty) — matches the producer's rev2 claim.
- Stale-copy audit (rerun myself): worktree `git status` shows exactly the
  13 candidate paths (11 modified + 2 new `hostile_test.go`,
  `hostile_carrier_test.go`); no `.task-board`, catalog, or
  `task-board.config.json` deltas. `task-board.config.json` candidate diff
  vs HEAD is empty; HEAD vs trunk is exactly the one z1yxg9 rpcwire fuzz
  gate line (`FuzzUntrustedEnvelopes 100x`).
- Trunk-content preservation: candidate `LOGBOOK.md` keeps trunk's
  `TASK-260917-3lt2xv` entry (diff vs HEAD = 11 insertions, 0 deletions);
  `README.md` keeps trunk's `-adopted` regions (only figure rewrites +
  suite docs added). `internal/catalog/*` untouched by candidate.
- Leaf is test-only plus traceability/README/LOGBOOK: modified paths are
  `LOGBOOK.md`, `README.md`, `hostchannel/README.md`, `gates_test.go`,
  `mutations.py`, `refusal_test.go`, `verify_test.go`, traceability
  registry + pins + report tests; new files are the two hostile suites.
  `traceability.go` diff is the one-line digest re-pin only. No production
  file (`channel.go`, `server.go`, `client.go`, `hosttrust`, `config`,
  `rpcwire`, `sshtransport`) changed.

## 2. AC-HOST-001 — 14 of 15 fully driven, B8 partial with measured bound

Gate text (SPEC line 15761, extracted myself): "Drive the real Config-4
launch and RPC-5 dispatch on supported SSH/platform lanes: full mutual
TLS, exact enrolled certificate/UUID/hello, no preauthentication effects,
migration/downgrade refusals, fresh-key rotation, revocation of idle/live
streams within one second, expiry and concurrent mutation-generation
fencing. Exercise valid and invalid actual certificates, both role
failures, stale/failed reads and recovery bypasses."

Rerun by reviewer (`go test ./internal/hostchannel -run TestHostile
-count=1 -v`, exit 0, 54 PASS): every matrix row B1–B15 except B8 fully
drives its production call site (`hostchannel.Dial`/`Serve`/`Call`,
`hosttrust` Enroll/Rotate/Revoke/generation, `config.Load` +
`LaunchForConfig`, OpenSSH carrier). B-row spot checks: B1
`TestHostileRealCarrierOpenSSH` PASS (executed lane); B3 ALPN
same-family row present and passing; B6 `TestHostileV4WithoutHostChannel
Binding` PASS; B7 rotation/enrollment halves PASS; B14 corrupt/unreadable
trio PASS (`invalid_config`/3); B15 bypass trio PASS.

B8 decision (the brief's explicit question). Pinned §11.10.3 text: "A
trust/config commit invalidates all streams of the prior generation;
every process, including daemonless RPC children, MUST close them within
one second of the commit using cancellation plus a bounded watchdog,
even when idle." Observed this run: `next_dispatch_refused_fast`
revoke-to-fence 10.3 ms (< 1 s, asserted); `watch_reports_commit_fast`
revoke-to-watch 11.4 ms (< 1 s, asserted); `idle_observation` logs "idle
stream still open 1.2 s after revocation without traffic" (observed, not
asserted — `serveLoop` in `server.go` fences at the next dispatch and
carries no background watchdog; confirmed by code read). Decision: the
MUST binds *processes*; this tree ships no `ax` process or CLI
(`LaunchArgv` computes argv only; the carrier test execs `Serve` via the
test binary). The library enforces the authority core — no later dispatch
or mutation from a stale generation (fenced in milliseconds, asserted) —
and exposes the sub-second cancellation signal (`WatchGeneration`,
asserted). A future production process wires watch→close to meet the
idle clause; nothing in the library prevents it. The producer's partial
with measured log evidence therefore does not claim what it does not
prove and does not contradict the MUST at this layer — no P2. The
process-level watchdog wiring is a P3 follow-up for the future ax-process
owner (§9), consistent with the z1yxg9 precedent that accepted
`WatchGeneration` for deadlines/generation. A minimal RED-first
in-package watchdog is *not* required of this leaf before story_final.

## 3. Hostile matrix — 7 of 7 rerun green through production entries

All seven rows pass through `Dial`/`Serve`/`Call` with exact §15
class/exit plus handler-effect-zero and state-census assertions (rerun
log in evidence tarball): unknown-peer both roles
(`authentication_failed`/7), key-change re-issued both roles plus
enrollment-gated rotation and no-second-rotation, spoofed hello-ID and
valid-but-unexpected peer (`host_identity_mismatch`/7,
`authentication_failed`/7), 7 disconnect phases
(`transport_failure`/8 or unframeable→8, census clean), TLS-flight +
encrypted-hello replay refused with inert re-hello redispatch, oversize
lines (over-cap → `transport_failure`/8; below-floor →
`incompatible_protocol`/6; intake bound documented as Go record framing
firing first with the 1 MiB limiter as backstop), 4 disclosure vectors
(`incompatible_protocol`/6). Timeout/race (silent peer, mid-request
stall, concurrent connections), copied-key halves (authenticates until
revoked on both sides), stale-generation at dispatch and mutation
boundary, Config-4 legacy / RPC-4 hello / plaintext-preface refusals,
both role failures, integrity-failure stale reads, and
operation-ID/body recovery bypasses all PASS. The single SKIP is
`TestHostileSSHHelperResponder`, the forced-command child guard (by
design, skips unless invoked as the helper).

## 4. OpenSSH loopback lane — executed, not skipped

`TestHostileRealCarrierOpenSSH` PASS (0.28–0.35 s across runs): starts an
unprivileged `sshd` on 127.0.0.1 with temp host key and forced exact
command, observes `ServeArgv` server-side, asserts TLS-record-first
bytes, no PTY (`-T` first), and bounded stderr. Native Tailscale SSH is
recorded as not executed in matrix §D, README, and ownership gaps —
never as passed. No SSH-implementation branch exists in the package
(HC-PARITY by construction, confirmed by the abstract `Stream` surface).

## 5. HC-* gates 10 of 10; both z1yxg9 P3s closed

`TestHCGateRegistryIsComplete` PASS; registry carries all ten families.
P3a: `TestVerifyPeerRefusals/alpn-same-family` present and passing (row
added to `verify_test.go`). P3b: `TestInvalidActivation/bad_destination`
uses 500 ms overrides with a < 2 s elapsed assertion (hook in
`refusal_test.go`), suite time 0.73 s. Reran both plus the registry green.

## 6. Mutation — producer battery rerun; 4 own narrowing mutants + control

Producer harness (`internal/hostchannel/mutations.py`, full behavioral
suite per plant under `go -overlay`, sources verified untouched):
- 9 new probes + neutral + client-cache control rerun by reviewer:
  9/9 KILLED by the named `TestHostile*`/`TestVerifyPeerRefusals` vectors
  (per-plant `test.log`/`results.json`/`table.md` in the evidence
  tarball), neutral passed, `client-cache` SURVIVED with the documented
  harmless bound (one-sided weakening: caching client vs ticketless
  server — the complementary `tickets` arm IS killed).
- 22 prior probes rerun by reviewer in two bounded batches: all KILLED
  by their named tests (including token-preserving `census-evasion`,
  killed behaviorally by `TestRuntimeWriteCensus`). Harness denominator
  on this tree: 33 probes = 1 neutral passed + 31 killed + 1 documented
  harmless SURVIVED (the producer's "26 prior" counts pre-leaf probes
  across packages; every probe in the shipped harness was rerun here).
- No new source-text-inspecting gate was added, so the DoD's
  token-preserving clause rests on the rerun `census-evasion` mutant,
  which passes the static census by design and is killed only by the
  behavioral suite — satisfying "executes the behavioral suite, not only
  the static checker."

Reviewer-owned narrowing mutants (`/tmp/rev-2x16gz/own_mutants.py` in the
tarball; overlay; full `hostchannel`+`hosttrust` behavioral suite per
plant; each KILLED run twice for stability, exits recorded):
- `own-authz-stale-plus1` (`hosttrust/authorize.go`: dispatch fence
  admits exactly +1 generation) — KILLED 2/2 by
  `TestAuthorizeDispatchRefusals/refreshed_number_on_old_snapshot`.
- `own-disconnect-misclass` (`hostchannel/server.go`: one truncated
  dispatch misclassified as `authentication_failed` instead of
  unframeable/`transport_failure`) — KILLED 2/2 by
  `TestHostileDisconnectPhases/mid_request_close` (proves the exact
  class pin, not just refusal).
- `own-unknown-zero-match` (`hostchannel/channel.go`: exact-match gate
  admits the zero-match member while still refusing double matches) —
  KILLED 2/2 by `TestVerifyPeerRefusals/unknown-leaf` (complements the
  producer's `match-count` probe, which admits the double-match member).
- `own-mutation-stale-plus1` (`hosttrust/authorize.go`: mutation boundary
  admits exactly +1 generation) — KILLED 2/2 by
  `TestStaleGenerationAtMutation` (plus two sibling stale-mutation
  tests failing).
- `own-control-comment` (comment-only) — SURVIVED with exit 0 as
  designed (applied harmless control). First run initially named two
  expected tests that the wrong layer killed; corrected to the actual
  killing tests above and rerun clean 2/2 (history in §10).

## 7. Story-close items — verified

- Ownership registry: `AC-HOST-001` bound to the suite; new unevidenced
  bindings 11.10.1–11.10.4 + 16.1 with honest gaps; 6.6/11.10 gaps
  refreshed; 11.10.5 unowned by design (no policy reader — the pinned
  text calls the predicates synthetic context, and no reader declaration
  exists in the tree). `tracecheck` exit 0 with 102 cases / 61 bindings /
  8 unowned / 17-of-487 clauses — exactly the README figures and the
  re-pinned `reviewedOwnershipCanonicalSHA256 e327bb08…`. (A naive
  file-bytes sha256 does NOT reproduce the pin — the pin is over the
  decoded semantic projection per `traceability.go:431`; `tracecheck`
  exit 0 is the correct oracle and was rerun here.)
- Self-mint refusal: `TestVerifyRepositoryRejectsForgedOwnershipAnd
  CapabilityClaims` (valid-declaration-cannot-self-mint,
  unknown-contract-owner, capability-field) rerun PASS; an independent
  `/tmp` probe confirmed a planted `AC-REVIEWER-FAKE-001` acceptance case
  perturbs the registry (102→103 cases) and therefore cannot match the
  pinned projection without review.
- README (`README.md` + `hostchannel/README.md`): suite commands present
  (`go test ./internal/hostchannel -count=1`), stated bounds named
  (Tailscale, non-host lanes, idle self-close, TLS alert text, no `ax`
  CLI/SSH launch/listener), no capability claim found by grep.
- LOGBOOK: candidate entry newest-first on top; trunk's `3lt2xv` entry
  preserved; every Story entry plus trunk present, zero conflict markers.

## 8. Hygiene — green

`gofmt -l` empty; `go vet` (hostchannel/rpcwire/hosttrust/traceability)
exit 0; `go build ./...` exit 0 (incl. `GOOS=linux/windows` per producer
rev2 rerun, accepted spot-checked); `git diff --check` clean; no
`__pycache__`/`.pyc`/stray plants/sshd state in the worktree;
`cataloggen -adopted -check` exit 0. Test gates rerun this run:
`TestHostile*` 54 PASS exit 0; `hostchannel`+`rpcwire`+`hosttrust`+
`traceability/...` exit 0; `-race` on the three product packages exit 0
(no warnings); full `go test ./... -count=1` exit 0, 31 packages ok
(observed tail in run notes; producer's per-package logs are the
attached reference).

## 9. P3 follow-ups (non-blocking, for later owners)

- P3: future `ax rpc serve` process owner should wire `WatchGeneration`
  → stream close and convert `TestHostileRevocationLivePeer/
  idle_observation` from observation into assertion (see §2 for why this
  does not block story_final).
- P3: the carried LOGBOOK entry's trailing NOTE still says the
  `refresh-candidate` replay "fails … worktree untouched at `244a7dc`"
  (rev1 state); rev2 refreshed to `c3de159` successfully. Consider a
  one-line annotation when the Story closes so the history reads
  correctly.

## 10. Reran-vs-accepted

Reran myself (all with observed exits; raw logs in the evidence tarball
unless noted): replay signatures ×4, ancestor/fidelity/stale-copy audits,
`TestHostile*` 3× (54 PASS each, incl. executed OpenSSH lane),
`TestHCGateRegistryIsComplete`/`TestVerifyPeerRefusals`/
`TestInvalidActivation`, `tracecheck`, `cataloggen -adopted -check`,
traceability package incl. forged-claim tests, full `mutations.py`
harness (33 probes), 4 own mutants 2× each + control 2×, relevant
packages + `-race`, full `go test ./...`, `go build`, `go vet`,
`gofmt -l`, `git diff --check`, README/LOGBOOK greps, self-mint probe.
Accepted from attached evidence (format spot-checked, not re-executed
byte-for-byte): producer rev1 per-plant logs and rev2 suite/race/cover/
fuzz/cross-build logs, CR validation log tail, predecessor outcomes.
`task-board validate`'s 191 pre-existing board-hygiene issues were not
rerun (none reference this task per rev2 record).

## 11. Handoff state

Candidate left UNCOMMITTED in the Story worktree; no commit, checkpoint,
integration, or landing by this run. `git status` shows only the 13
candidate paths. Evidence attached: `TASK-260830-2x16gz_review-
verdict-rev1.md` (this file) and `TASK-260830-2x16gz_review-
evidence-rev1.tar.gz` (real gzip: §6 mutant outputs + own harness,
§8 logs, run notes).
