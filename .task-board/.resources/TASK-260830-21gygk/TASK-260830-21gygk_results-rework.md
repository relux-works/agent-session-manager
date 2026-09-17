# TASK-260830-21gygk results (rework after CR rev2 review)

Scope: v0.6.0 selector contract (\u00a7\u00a714.7/14.7.1, shared SelectionPlan
construction/revalidation \u00a714.7.2, refining \u00a72.3, authoritative summaries
\u00a7\u00a75.7/14.7.3). Authority: spec v0.6.0 commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6` (STORY-260908-18woqo landed).
Branch `task-board/story/STORY-260830-3tq4ns` at
`83640d191f78f0cd685e5f709027819799efe1cf`, descending from
`origin/main` `8cf4aaaa190e6a11dff2661aa6806ce476128653`. Candidate left
UNCOMMITTED in the managed Story worktree for the handoff snapshot:
tracked delta `LOGBOOK.md`, `README.md`,
`internal/sessrepo/store.go`, `internal/sessrepo/sessrepo_test.go`
(predecessor bytes preserved); untracked `internal/sessquery/`
(16 files: 6 implementation, 8 tests, TRACEABILITY, mutate battery).
No `ax` session surface exists in this tree; CLI delivery, wire exits,
lifecycle effects, and the \u00a714.7.4 recovery flow are caller/recovery-leaf
owned. Independent Astra medium review and orchestrator integration own
the remaining path.

## 1. Rev2 findings disposition (all four addressed in the shared layer)

- FINDING 1 \u2014 revalidation ignored conflicting authority outside the
  selected source. `Revalidate` and `bindPlan` now run
  `validateAuthorityUnion` (`internal/sessquery/revalidate.go`) over
  the local and every allowlisted source for the pinned UUID after the
  plan-source checks: a disagreeing non-parked record digest is
  `integrity_failure` (same class/message as `Resolve`); a
  contradictory observed winning lease or fresh tombstone evidence is
  `selector_plan_stale`; zero-lease copies skip only the lease check;
  parked copies and absences never contradict; unreadable required
  sources keep their read-failure class. Names are never re-resolved
  and no selection is substituted: a same-name gain for another UUID
  stays current, an agreeing replica stays current, and
  explicit-source no-fallback is preserved. Reviewer probe
  `TestReviewerOtherSourceDivergenceMustRefuse` carried verbatim into
  `review_test.go` and green, plus seven new union tests.
- FINDING 2 \u2014 16-member plan with lease authority. `SelectionPlan`
  gains `lease_record_id` (struct, wire, canonical digest, shapes).
  This stack persists no standalone `urn:ax:schema:lease` objects
  (sessrepo stores Session Records and Session Events only), so the
  member binds the locally attested canonical digest of the validated
  winning envelope lease, recomputed from validated facts on every
  build and revalidation \u2014 stated exactly in `plan.go`, TRACEABILITY
  bounds, and README, never a fabricated object digest. Record-only
  builds refuse `selector_bootstrap_incomplete`; local builds without
  a known local host refuse `invalid_config`; shapes require a bound
  source UUIDv7, the lease attestation digest, and the full positive
  triple. The plan-source path recompares the attestation, closing
  hand-built digest forgery. `TestReviewerPlanRequiresLeaseRecord`
  carried verbatim and green.
- FINDING 3 \u2014 authoritative summary refusal. `List`/`Status` refuse
  record-only bootstraps with `selector_bootstrap_incomplete`, whole
  list on any unrepresentable record (mixed listings included, nil
  rows, nothing omitted); `InspectLocal` keeps raw
  recovery-diagnostics access by design and is not a public summary
  entry. New closed `AuthoritativeStatus`/`AuthoritativeList`
  (`summary.go`) bind owner names only from validated host metadata,
  roles only with a known local host, checkpoint timestamps only from
  validated observations, and pass optional liveness/capability/
  workspace observations through only when supplied; missing required
  facts refuse `selector_observation_unavailable` and nothing is
  inferred. `TestReviewerBootstrapSummaryMustRefuse` carried verbatim
  and green, plus eight new summary tests.
- FINDING 4 \u2014 narrowing evidence. All seven `&& false` clause-disables
  replaced with single-member admissions (revocation/hostA,
  config/hostC-ws, binding/hostB-ws, index/idB, parked chain-mismatch
  record, leaseB triple+attestation, two-event chain), plus new
  narrowings for both cross-copy agreement call sites, union
  record/lease/tombstone, plan bootstrap/host, summary bootstrap, and
  observation host: 33 narrowing + 3 ordering mutants, every one
  KILLED by a named behavioral test through the delivered harness
  (battery exit 0; CONTROL/NOT_APPLIED/COMPILE_OR_HARNESS_FAILURE
  distinct, never kills). Notable diagnosis: the "record replacement"
  fixture presents as a parked chain-mismatch with an empty digest,
  not a renamed live record \u2014 the first N-rev-record plant (admitting
  name "renamed") SURVIVED and exposed it; the corrected plant admits
  the parked mismatch and is killed with degradation to the lease
  reason. TRACEABILITY claims corrected (no "8 of 8 in full", no
  "every bound fact", per-gate mutant rows).

## 2. Validation re-derived on the final candidate (real exits)

| Command | Exit |
| --- | --- |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l internal/` (empty) | 0 |
| `git diff --check -- internal/ LOGBOOK.md README.md` | 0 |
| `go test ./... -count=1` (all 26 packages ok) | 0 |
| `go test ./internal/sessquery/ ./internal/sessrepo/ ./internal/sessstate/ -cover` (90.9% / 87.6% / 92.2%) | 0 |
| `go test ./internal/sessquery/ ./internal/sessrepo/ ./internal/sessstate/ -race -count=1` | 0 |
| tracked-JSON gate (`git ls-files -z '*.json' \| xargs -0 -n1 python3 -c ...`) | 0 |
| mutant battery `python3 internal/sessquery/testdata/mutate.py` (33 N + 3 B KILLED, 0 survivors) | 0 |

Raw logs: `TASK-260830-21gygk_validation-rework.log`,
`TASK-260830-21gygk_mutants-rework.json` (attached); per-mutant logs
in worktree scratch `.temp/TASK-260830-21gygk/mutants-rework/logs/`.

## 3. Acceptance: 8 of 8 AC rows driven at the shared library boundary

Per `internal/sessquery/TRACEABILITY.md` (production call site +
named tests per row): bare UUID/name selectors, qualified selectors,
ambiguity, list summaries, status summaries, stable deterministic
sorting, SelectionPlan build/revalidate, negative/refusal cases. 0 of
8 rows delivered as a public CLI surface: stated caller-integration
bound, not a gap in assigned shared behavior. No unsupported
capability advertised.

Ready for review (producer handoff; independent Astra review and
orchestrator integration own the remaining path).
