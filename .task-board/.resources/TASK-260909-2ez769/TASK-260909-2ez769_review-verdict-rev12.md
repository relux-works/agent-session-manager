# CR12 review verdict: accepted

Task TASK-260909-2ez769, CR-TASK-260909-2ez769-12 revision 12.
Reviewer RUN-260917-8f78c2 (reviewer/muse).
Base `5e548743ea79f169d20f4006b592f171c6961f30`;
candidate tree `83d69b0098922f7d07cfb93e48a638c6439601b0`.
Patch SHA-256 `4392305f79960cd914bb47485526003b84e437ef9bb55387bafd12cf3b36f4c3` (verified against board bytes).
All probes ran against an immutable `git archive` of the candidate tree plus a
symlink-free scratch copy; the live Story worktree, index, branch and HEAD were
never mutated (HEAD still `5e54874`, candidate uncommitted). No product edits,
commits, checkpoints or integration by this reviewer.

Provenance: `efe119a` and `5e54874` signatures verify (author key).
Spec pin: `internal/specdoc/SPEC.v0.6.0.md` sha256
`74504539fb43c28ae3450622bc1002e643f116cd3e14df4567a882231e90896b`
matches `internal/specpin/v0.6.0.lock.json` commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`. Producer evidence
`TASK-260909-2ez769_producer-evidence-rev12.tar.gz` is a real gzip archive
(verified with `file` + sha256 `6ae14a1d…e73e2de7bd`).

## F1 (CR11 P2) — repaired and verified

- CR11 plant rerun: the committed regression
  `TestWritePathCensusRejectsUnlistedInterfaceCallSite` (port of
  `reviewer_callsite_test.go`, extended to pre- AND post-hold plants) passes,
  i.e. the child witness renames real files through the production entry while
  the child census refuses both unlisted sites. Census gate passes on the clean
  tree.
- Five own plants in `replaceDurably`, each gate exit 1 with a violation at the
  planted site: (p1) second `Rename` at a later unlisted position — refused;
  (p2) method value `rn := filesystem.Rename` invoked pre-hold — refused with
  explicit "method value/site escapes"; (p3) call inside a deferred func —
  refused; (p4) edge in an `if false` branch — refused (no constant folding);
  (p5) `goto` bypass shifting lines — refused by site identity.
- p6 (line-preserving `goto` over the guard to an inventoried site): census
  exit 0, zero violations — ADMITTED by the static census, which enforces
  lexical guard-precedes-call, not CFG dominance. Behaviorally inert: all four
  HOLD writers carry the guard as the lexically-first statement, so no
  intra-function path can reach any call without executing it; a zero-hold
  witness through the p6 bypass is still refused (`require exclusive hold`).
  Recorded as a P3 hardening observation (CFG-aware dominance or goto/label
  scan), not verdict-blocking: the census document claims lexical precedence
  and the implementation matches the claim.
- Former method-spelling admission predicate is removed; remaining `Name() ==`
  uses are hold-guard recognition (`requireHold*`) and the `CreateExclusive`
  backend definition — fail-closed detectors, matching the narrowed census
  claim. The 119-site/90-group inventory carries per-group production
  justifications. Narrowing mutant `writepath-interface-callsite-skip`
  (preserves method identity, drops site/hold admission) killed by the
  regression, 2/2 runs. Rogues suite 28/28 pass.

## F2 (CR11 P2) — repaired and verified

`ensureConfigBindingLocked` validates the store-side association (:246-254)
before publishing the target sidecar (:256-280); the store index is derived
after authority (:285-293). Committed trio green:
`TestReviewerRefusedRebindLeavesTargetUnclaimed`,
`TestCompetingFirstBindingsPublishExactlyOneAssociation`,
`TestInterruptedFirstBindingIsRepairedAfterReopen`, plus the CR11
`TestConfigBindingSurvivesReopenAndRejectsForeignRoot` control.
Own probes (scratch, exact final source): target-side crash injection (fails
the authoritative rename, unlike the producer's index injection) leaves both
records absent and a retry converges; reverse-ordering consequence
(index present, sidecar deleted) is refused already at `Open` with resource
mismatch and the stranded index stays intact, never silently rebound; a second
store refused against a target bound elsewhere leaves both associations
untouched. All pass.

## Mutation batteries (exact final source, fixed harness)

Methodology note: `go -overlay` silently ignores mappings whose keys are
symlink-resolved (`/private/tmp/...`) while the build resolves the logical
(`/tmp/...`) path. My first `/tmp` runs therefore reported false survivors;
rerunning from a symlink-free copy reproduces the producer's kills, and direct
source application confirms the mechanism. Producer's 47-killed evidence is
consistent with a non-symlinked run location and stands.

- Config (`mutations_v4.py`, 14 probes): neutral passed + 13 narrowing mutants
  killed by the expected named tests, 0 survivors.
- Host Trust (`mutations.py`, 35 probes): neutral passed + 34 narrowing
  mutants killed by the expected named tests, 0 survivors — including
  `hold-state-root-skip` (killed by
  `TestHeldExclusiveRejectsForeignConfigStateRoot`) and the preserve-token
  `compensate-verify-before-restore` order swap (behavioral-suite kill).
- Three reviewer-owned narrowing mutants on unmutated arms, each killed twice
  by the expected test: `reviewer-trust-version-downgrade` (admits exactly
  `0.9.0`; `TestDecodeTrustRefusals/downgraded_version`),
  `reviewer-credential-format-leak` (emits exactly the credential ID;
  `TestTrustRedaction`), `reviewer-exclusion-prerollback-skip` (admits exactly
  pre-rollback copies; `TestExcludedConfigDirName/pre-rollback_copy`).
- Key mutants rerun twice for determinism (callsite-skip, hold-state-root-skip,
  compensate-verify-before-restore, all three own): 2/2 kills.

## Suites, hygiene, bounds

- `go test ./... -count=1`: exit 0, all packages ok. `-race` on
  `internal/config` and `internal/hosttrust`: exit 0. `go vet ./...` (native
  and `GOOS=windows`): exit 0. `gofmt -l`: clean. `go mod tidy -diff`: exit 0
  (offline). Builds: native, darwin/arm64, linux/arm64, windows/amd64 exit 0.
  Coverage: config 92.9%, hosttrust 76.4%, peeridentity 97.5%, localstore
  83.8%, secprim 94.4%. Validation log: 26/26 green.
- Changed paths: 74 (17 tracked + 57 untracked), restricted to
  `internal/` source/test/harness, `README.md`, `LOGBOOK.md`, `go.mod/go.sum`.
  No `__pycache__`/`.pyc`, no `task-board.config.json`, no stray plant files.
  README tool row describes the all-indirect-call policy and exact exceptions.
  `golang.org/x/tools` is test-only (census). Literal RPC-manifest overlap is
  `README.md` only; no RPC implementation path absorbed. `go.mod` adds only
  `x/tools` (+2 indirect).
- AC accounting: **19 of 21 rows driven**; rows 15 (Windows ACL runtime) and
  21 (downstream replication consumers) remain stated bounds, not passes. No
  post-lock-release dispatch, mesh-wide revocation, physical power-loss, or
  human-verification guarantee inferred. RPC transport/admission outside leaf.

## Verdict routing

Accept. F1/F2 counterexamples are repaired with committed regressions,
narrowing mutants, and independent probes; hygiene is clean; evidence is
attached as `TASK-260909-2ez769_review-verdict-rev12.md` (this file) and
`TASK-260909-2ez769_review-evidence-rev12.tar.gz`. No product edits, no
`commit_ack`, no integration by this reviewer.
