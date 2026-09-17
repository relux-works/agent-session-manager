# TASK-260830-21gygk rev4 rework outcome (RUN current)

## Verdict request
Ready for review. Candidate left UNCOMMITTED in STORY-260830-3tq4ns worktree
(checkpoint `83640d191f78f0cd685e5f709027819799efe1cf` preserved; no commit,
no branch operation).

## Scope
Close the three CR4 P1/P2 findings (review-verdict-rev4) at the shared
`internal/sessquery` layer under pinned AX v0.6.0
(`0cbdf100dbf84df50c64f792b1f940e3a67859a6`):
1. semantically validated winning lease (predecessor legality, epoch+1,
   session/checkpoint/predecessor-lease binding);
2. required parked same-session authority refuses instead of skipped;
3. closed Section 7.3 seven-name capability vocabulary at Status and List.
Closed predecessor crash/reducer/enum conclusions untouched; the only
predecessor-surface change is the provhost attestation-site bound 4→5,
which that test's own text authorizes for a justified new site.

## What changed (production)
- `internal/sessrepo/checkpoint.go` (new): `AttestCheckpointRecord`
  owns canonical Checkpoint Record attestation through the
  canonicaljson owner, mirroring `AttestLeaseRecord`, so query
  admission never attests itself.
- `internal/provhost/identity_test.go`: attestation-site bound 4→5
  with the checkpoint justification; no non-leaf production call
  site attests (second half of the bound still holds).
- `internal/sessquery/lease.go`: `validatedLease` carries
  `checkpoint_id`; new `validatedCheckpoint` admission via
  `parseCheckpointRecord`; `winningLeaseFor` now validates the full
  succession after greatest-tuple selection: `checkLeaseChain`
  (epoch+1 ancestry to an epoch-1 null root; self/cycle/skip/
  contradictory/duplicate links are integrity_failure, missing
  names/records stay observation_unavailable) and
  `checkWinnerCheckpoint` (successors must resolve an admitted
  checkpoint for their session bound to the predecessor lease;
  epoch-1 roots may carry none, or one bound to themselves;
  unresolvable references are observation_unavailable).
- `internal/sessquery/query.go`: new `Reader.CheckpointRecords`
  input with ownership documentation.
- `internal/sessquery/revalidate.go`: `checkUnionCopy` fails a
  parked pinned-session copy closed with
  selector_observation_unavailable (never evidence, never absence);
  union/heads comments updated to the fail-closed order.
- `internal/sessquery/summary.go`: `checkCapabilities` admits
  exactly the provhost-owned Section 7.3 registry
  (`provhost.Capabilities()`); any other name refuses
  invalid_config through both authoritative entries. Not the
  15-name session-adapter or 8-name directory registries.

## What changed (tests, all durable in-candidate)
- `rev4_regression_test.go`: the four failing reviewer probes
  carried verbatim (RED baseline 4 FAIL / 2 PASS controls logged
  to `rev4-baseline-red.log`), now green after the fix.
- `succession_test.go` (new): valid-successor positive,
  cycle, skipped-epoch, epoch-1-with-predecessor,
  self-with-checkpoint isolation, dangling predecessor,
  wrong-session/wrong-predecessor checkpoints, malformed
  checkpoint bytes, epoch-1 self-bound positive.
- `union_test.go`: `TestRevalidateUnionParkedCopyIgnored`
  replaced by `TestRevalidateUnionParkedCopyRefuses` (divergent
  and agreeing record arms × build/revalidate); agreeing arm
  isolates the parked gate from record agreement.
  `TestBuildWithLaggingLeaseCopy` now carries complete validated
  succession (epoch-1 record + predecessor-bound checkpoint).
- `summary_test.go`: exec/net fixtures replaced with registry
  names; bad-status/enabled cases now use valid names;
  `unknown_capability_name` input case and
  `TestAuthoritativeAdmitsFullCapabilityRegistry` (all seven
  names through both entries) added.
- `fixtures_test.go`: `checkpointRecordBytes`,
  `checkpointDigestOf`, `leaseSuccessorBytes`, `withSuccessor`,
  `withCheckpoints` helpers (fixtures identified through the
  canonicaljson owner before use).
- `testdata/mutate.py` (shipped runner): four new narrowing
  plants (N-chain-self, N-checkpoint-placeholder,
  N-parked-union-refusal, N-capability-name) plus the
  C-harmless-comment same-instrument SURVIVED control.

## AC coverage ratio: 8 of 8 rows driven, 8 of 8 established
Denominator is the task's existing 8-row decomposition
(TRACEABILITY.md table); 0 of 8 public CLI rows implemented
remains the accepted caller-integration bound.
- UUID/name → Reader.Resolve (precedence/union tests).
- Qualified selector → Reader.Resolve/resolveExplicit
  (single-index, never-fallback, read-failure classes).
- Ambiguity → matchName/checkRecordAgreement (ASCII-fold,
  agreement/integrity).
- List summaries → Reader.List/AuthoritativeList
  (bootstrap/mixed refusal, validated observations, closed
  capability vocabulary).
- Status summaries → Reader.AuthoritativeStatus (lease/role/
  host/observation binding, 64-bound, process boolean).
- Stable sorting → Reader.List/Resolve (bytewise order,
  peer order, tie break + B-* mutants).
- SelectionPlan → Reader.BuildPlan/Revalidate/ParsePlan
  (16 members, chain+checkpoint succession, union/heads,
  fixed compare order, tamper-evident persistence).
- Negative/refusal → every gate above through its production
  entry, each with a narrowing mutant KILLED (table below);
  no unsupported capability advertised.

## Mutation evidence (exact current source)
- Focused driver `.temp/TASK-260830-21gygk/mutants-rev4/`:
  4 narrowing KILLED + 1 harmless SURVIVED, all files
  hash-verified restored (see results.json + per-plant logs).
  - N-chain-self → TestSelfPredecessorWithCheckpointMustRefuse
    fails by building successfully (strong admission kill);
    TestRev4SelfPredecessorMustRefuse still refuses via the
    checkpoint gate (documented subsumption).
  - N-checkpoint-placeholder → TestRev4MissingCheckpointMustRefuse
    fails by building successfully. (A mere condition weakening
    without early return only shifts the refusal reason onto the
    zero value — recorded, not claimed as a kill.)
  - N-parked-empty-record → agreeing-record member admitted
    (Revalidate/BuildPlan return nil, strong kill);
    divergent-record member shifts to record-agreement
    integrity (reason change only; divergent reviewer probe
    still refuses — documented subsumption).
  - N-capability-name → TestRev4UnknownCapabilityMustRefuse
    fails by advertising invented_capability.
  - C-harmless-comment → SURVIVED exit 0 (classifier control).
- Shipped battery `python3 internal/sessquery/testdata/mutate.py`
  `.temp/TASK-260830-21gygk/mutants-shipped-final/mutants.json`
  on the exact final candidate: 44N+3B KILLED, 0 BAD, script
  exit 0; control-before/after green, C-harmless-comment
  SURVIVED, C-not-applied and C-compile-failure classified
  separately (never kills). The 40 retained plants cover
  unchanged gates byte-identically; the four new plants plus
  the harmless control exercise the changed lines.

## Validation (observed in-session)
- `go build ./...`: clean. `go vet`
  ./internal/sessquery/ ./internal/sessrepo/
  ./internal/provhost/: clean. `gofmt -l internal/ cmd/`: clean.
  `git diff --check`: clean.
- `go test ./... -count=1`: all 26 packages ok.
- Coverage: sessquery 87.7%, sessrepo 87.2%, provhost 86.0%,
  sessstate 92.2%.
- `go test -race ./internal/sessquery/ ./internal/sessrepo/
  -count=1`: clean.
- sessquery verbose: 72 top-level PASS, 0 FAIL.
- Focused driver `.temp/TASK-260830-21gygk/mutants-rev4/`:
  same 4 KILLED + 1 SURVIVED on final bytes with per-file
  sha256 before/after binding (lease.go 7c9d9b01…,
  revalidate.go 66ae8109…, summary.go 4e33b754…).

## Docs/traceability
- `internal/sessquery/TRACEABILITY.md`: AC rows (new tests,
  chain+checkpoint succession, parked fail-closed, closed
  vocabulary), four new gate rows with subsumption bounds,
  lease/heads bound bullets, runner paragraph.
- `README.md`: immutable-plans and authoritative-layer
  paragraphs (succession, parked union refusal, closed
  registry). No new capability or CLI claim.
- LOGBOOK: entry appended below.

## Evidence map (task scratch, git-ignored)
- `.temp/TASK-260830-21gygk/rev4-baseline-red.log`: ported
  probes before the fix (4 FAIL, 2 PASS controls).
- `.temp/TASK-260830-21gygk/mutants-rev4/`: focused driver
  (/tmp/sel-mutants-rev4/mutate.py copy recommended for reruns),
  results.json + per-plant logs.
- `.temp/TASK-260830-21gygk/mutants-shipped-final/`: isolated-copy
  shipped battery, mutants.json + per-plant logs +
  control-before/after logs.
