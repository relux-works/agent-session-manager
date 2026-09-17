# Review verdict: TASK-260830-3k3e6m rev1 — ACCEPT

Reviewer run: RUN-260917-23e107 (reviewer, muse-spark xhigh).
Change Request: CR-TASK-260830-3k3e6m-1 rev1, candidate tree
84843140f625a05e3fb1f4239f707cffe5d600c4 over base d805720ef2d0f862573018f2f5967a1b6c1dc628.
Normative authority: internal/specdoc/SPEC.v0.6.0.md §§10.5, 10.6, 13.12, 13.13
(read from the pinned text, not from the producer's summary).

Verdict: **ACCEPT**. No P1/P2/P3 items. The candidate implements the
durable operation journal contract with independently verified phase,
token, digest, and recovery gates, attacked — not read — by 11
producer narrowing mutants (all reproduced KILLED here) plus 4
reviewer narrowing mutants on arms the producer did not mutate (all
KILLED) and 2 harmless controls (both SURVIVED).

All reviewer probes ran in an isolated copy at /tmp/rev3k3e6m/cand
(rsync of the worktree excluding .task-board/.git/.temp); the live
Story worktree, index, branch, and HEAD were never mutated by this
review (git status shows exactly the 19 candidate paths, no reviewer
files, no __pycache__/.pyc). PYTHONDONTWRITEBYTECODE=1 for every
harness run.

## 1. Section 10.6 shape and phases — verified edge by edge

Extracted from the pinned text independently: the closed 22-member
journal object, the 8-value phase enum (staging, validating,
prepared, committing, rolling_back, rolled_back, committed, failed),
the closed Provider Journal Transaction (prepared requires token,
all other states null), the 19-member Task-board Journal
Transaction with the exact null/state invariants (not_started all
null; imported the import triple only; opened the open triple only;
adopted/resumed manager+binding with retained_active;
dormant_finalized dormant ref only, no usable token, bridge
dormant, pending_expiry with cleanup_after = consumed open expiry;
rolled_back no token/reference with removed; failed at most one
valid pair), the 4 bridge operation IDs allocated once and
surviving restart, retry = same operation ID + byte-equal body.

- journal.go encodes the 22-member closed shape (journalMembers),
  all enums, all nullability tables (checkProviderTransaction,
  checkBoardNullability, requireImportTriple, requireOpenTriple,
  checkFailedPair), ID binding (checkTaskBoardDrift,
  checkTransactionAuthority), and token grammar (checkToken:
  base64url fixpoint, 32..512 bytes). Diffed member by member
  against the spec tables: no missing, no extra member.
- Phase table (legalTransitions): staging→{validating,
  rolling_back, failed}; validating→{prepared, rolling_back,
  failed}; prepared→{committing, rolling_back, failed};
  committing→{committed, rolling_back, failed};
  rolling_back→{rolled_back, failed}. Every admitted edge is
  grounded (forward edges follow the §10.6 coordinator order;
  abort edges follow "resume a valid staging transaction or roll
  it back"; committing→rolling_back is additionally gated on
  pre-activation per step 5's post-adopt rollback ban). Attacked:
  staging→committed, prepared→committed, rolled_back→anything,
  committed→rolling_back all refuse (TestTransitionRefusesIllegalEdges,
  rerun here, PASS).
- Bridge sub-table matches the import→open→adopt→resume /
  dormant-finalize order with the post-adopt rollback ban
  (adopted/resumed never reach rolled_back). Dormant expiry
  binding and failed-pair rules verified in code and tests.
- Retry semantics (store.go replayCreateLocked): byte-equal
  replays without a second effect (inode/file-set assertion in
  TestCreateReplayAndConflict, PASS here); byte-unequal refuses
  idempotency_mismatch; second prepare refuses naming the first.
  Defense in depth confirmed by attack: weakening only the
  journal-level digest compare to a prefix still refuses (receipt
  level holds) — the R-own-digest-prefix SURVIVED-then-KILLED
  sequence below proves both layers are load-bearing.

## 2. Recovery evaluator — verified with own crashes

CR-MAT-01..08 mapping confirmed: 9 TestCrash* tests name each
boundary (01, 02, 03, 04, 05 provider+bridge, 06 open+prepared,
07 adopted+ambiguous, 08) with hook crashes + reopen + Recover
asserting the exact outcome; SIGKILL child test at the
journal/receipt seam (crash_unix_test.go, asserts killed by
SIGKILL, journal durable without receipt, CR-MAT-02 safe_retry).

Own injections (own_probe_test.go, isolated copy only, 7/7 PASS):
- Exclusivity: provable commit (committing journal, converged
  authorities, exact marker) + moved lease → parked
  (lease_conflict). Rejection gates precede completions.
- Receipt-without-journal at CR-MAT-02 → ErrUnknownJournal
  (absence, never a classification).
- Garbage journal bytes → parked with quarantine evidence that
  re-reads byte-identical after clean store reopen.
- Unknown provider probe at CR-MAT-05 with provider-store
  authority and no recorded transaction → parked
  (provider-uncertainty floor).
- 13.12 disk_full at CR-MAT-06 on a prepared journal → parked
  with the row remediation; phase preserved.
- Dormant_finalized carrying a usable open token refuses;
  65537-blob chunk map refuses (these two double as mutant
  killers below).

Rejection rules confirmed in decide(): moved/unknown lease,
substitution (native changed/missing), two live authorities,
unfenced continuation, marker mismatch/invalid, unknown decisive
status, token/binding contradiction, torn progression — each parks
(TestRecoverGatesPark 14/14 subtests rerun here, PASS). Parked and
rollback evidence persist as recovery.json with lease, exact IDs,
checkpoint/native identity, and last_error; never a fourth outcome
(mustOutcome enforces the 3-outcome vocabulary in every test).
Terminal journals replay but stay frozen under live-authority
gates.

## 3. Section 13.12 rows

Journal-observable rows driven: disk full, rename blocked
(TestRecoverGatesPark), owner-resume lost, operator interrupt,
adopt-fails-after-lease (TestRecoverFailureMatrixRows),
import/open/adopt-before-lease rollback and token-expiry
(TestRecoverRollbackRequired), plus my disk-full-at-prepared
plant. Non-journal rows (SSH disconnect, invalid digest/schema,
SQLite corrupt, etc.) require no journal behavior per the matrix
and are correctly out of scope.

## 4. Receipt reconciliation — accept

§10.6 MJ-RPC-PREPARE-LOST names "the destination journal and
prepare receipt" as the durable pair; the coordinator order step 1
binds prepare IDs + canonical digest before any staging/bridge
mutation. The sessckpt receipt binds a different namespace
(capture operation, record bytes, checkpoint under §5.4). The
candidate adopts sessckpt's discipline (operation key, input-digest
compare, no-replace install after the named bytes — store.go
writeReceiptLocked/readReceiptLocked) while keeping the records
distinct, documented in TRACEABILITY.md. Sharing one shape would
merge the idempotency namespaces the spec keeps apart. The
discipline was reused, not forked: verified the no-replace +
compare logic mirrors sessckpt's install order.

## 5. Refusal census and mutation

Census: ~130 invalid() arms in journal.go, ~30 refusal sites in
store.go, ~30 parks/refusals in recover.go, ~20 in marker.go. The
producer's denominator is the gate-family count (11), not the arm
count — stated honestly in their results ("Every refusing gate",
table of 11). Arms without a dedicated producer mutant include
the dormant open-token rule, the 65536 chunk bound, the
receipt-level digest/byte compares, and the unfenced-continuation
polarity — each now covered by a reviewer mutant:

- Producer harness rerun in isolated copy: 11 KILLED + control
  SURVIVED, exit 0 (producer-harness-rerun.log).
- R-own-unfenced-flip (recover.go lease polarity flip) →
  KILLED by TestRecoverGatesPark/unfenced_continuation.
- R-own-dormant-open-token (admits usable open token in
  dormant_finalized) → KILLED by
  TestOwnDormantOpenTokenRefuses.
- R-own-chunk-bound (admits exactly 65537 blobs) → KILLED by
  TestOwnChunkBoundRefuses65537.
- R-own-digest-prefix (scheme-prefix digest compare at both
  journal and receipt sites) → KILLED by
  TestCreateReplayAndConflict. Notably SURVIVED with only one
  site weakened, proving redundant defenses.
- C-own-control (comment-only) → SURVIVED.
- No gate inspects source text (all gates are value/table/grammar
  comparisons); the table mutants preserve every literal and
  change only edge sets, satisfying the token-preserving
  requirement structurally.

## 6. Story-close items — all verified

- Forged-Admit same-epoch foreign-lease negative present in
  internal/sessckpt/refusal_test.go and RED: dropping the
  same-epoch arm from checkRawHeadBinding admits the forgery
  (test FAILs, "error = nil"); restored file passes.
- README sections for sessckpt and matjournal present with test
  commands and explicit no-capability closers ("adds no ax
  command, no doctor result, and no runtime capability claim");
  cigate suite green in the full run.
- Ownership registry: section:5.4 rebound to
  sessckpt.Capture with honest gap text; new 10.6/13.12/13.13
  bindings to real declarations (matjournal Create/Recover);
  13.12 marked "unmeasured" with an explicit gap sentence —
  judged HONEST (it declares the measurement gap rather than
  claiming coverage; matches the 13.14.5 precedent pattern, and
  the tracecheck gate deliberately refuses to admit unmeasured
  bindings to assigned scope). tracecheck reproduces
  acceptance_cases=105 bindings=59 clauses 17/489 exit 0 on the
  exact tree; the digest pin is enforced by the passing
  traceability tests. Acceptance cases
  materialization-journal-lifecycle/-recovery are registered
  (2/3 references) and bound to real production declarations —
  no self-minted claim.

## 7. Bounds and hygiene

- canonicaljson still registers materialization-journal 2.0.0
  with rejectUnsupportedImmutableObjectShape (closed_shapes.go);
  census/inventory suites pin it; no consumer treats the refusal
  as support — shapes validate inside matjournal as stated.
- last_error at Structured Error 1.2.0 justified: the catalog
  pins operation_uncertain ContractVersions to [1.2.0, 1.3.0]
  (catalog_gen.go:590), the earliest version registering every
  emitted code.
- Changed paths exactly the 19 candidate paths; no
  __pycache__/.pyc; no task-board.config.json change; no stray
  plants (live tree untouched — all reviewer probes/mutants ran
  in /tmp/rev3k3e6m/cand, deleted after).
- gofmt clean, go vet clean, -race green on matjournal+sessckpt,
  full `go test ./... -count=1` green (see evidence tarball),
  tracecheck exit 0.
- AC coverage 12/12 as claimed: spot-checked every row's
  production call site and named test; the ratio is earned, not
  prose.

## Evidence attached

- TASK-260830-3k3e6m_review-verdict-rev1.md (this file)
- TASK-260830-3k3e6m_review-evidence-rev1.tar.gz: own_probe_test.go
  (7 probes), own_mutant_harness.py, own-mutants-final.log
  (per-plant raw output + subprocess exits), own-probes.log,
  producer-harness-rerun.log, suite/race/gate logs.

Checklist: the 5 reviewer items (implementation matches AC,
solution fits architecture, tests green, gates attacked not read,
verdict evidence added) are complete; routing via accept_cr.
