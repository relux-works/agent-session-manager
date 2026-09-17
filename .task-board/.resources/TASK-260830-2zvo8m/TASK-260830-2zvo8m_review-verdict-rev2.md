# Review verdict — TASK-260830-2zvo8m rev2 (implement-native-resume-smoke-framework)

Reviewer: muse-spark (reviewer). CR `CR-TASK-260830-2zvo8m-2` rev2,
candidate tree `901ef02784aabcab8ce913240f344ff5caaae96b`
against trunk `e4e3e8834675cf3814effd3b2673b4931dfbf311`.
Patch `TASK-260830-2zvo8m_change-request_rev2.patch`
sha256 `b3d6693d48719f5e8edc294f50ee22c432e60bd4e22a186c1e2de2c96794d585` confirmed.
Story tip `54ed1b3` on trunk `e4e3e88`; checkpoints `a6dfaf2`, `54ed1b3`
signatures Good (oparin@me.com). No product edits, commits, checkpoints,
or integration by reviewer; probes read-only except private /tmp copies.
`PYTHONDONTWRITEBYTECODE=1` for every harness. Candidate left UNCOMMITTED.

## Verdict: ACCEPTED → `accept_cr(revision=2)`

The two P1 Story-close regressions from rev1 are repaired exactly.
Product code is byte-identical to the fully-verified rev1 candidate
outside `LOGBOOK.md`/`README.md`, and every gate re-ran green on the rev2 tree.

## P1-1 / P1-2 restoration (exact)

- P1-1 LOGBOOK: the `### TASK-260917-3lt2xv` header + 4 bullets are
  line-identical to `git show e4e3e88:LOGBOOK.md` (header index base 7 =
  work 58, next 4 lines EQ). Order newest-first: 2zvo8m, 3lt2xv, 3uzfyn.
  `git diff HEAD --numstat -- LOGBOOK.md` = 51/0: purely additive.
- P1-2 README: all three `-adopted` needles present 2/2, 1/1, 1/1
  (base = work). `git diff HEAD --numstat -- README.md` = 87/13; the 13
  deleted lines are traceability-figure rewrites only (list in evidence
  `readme-deleted-13.txt`), zero contain `adopted`.
- Stale-copy audit: trunk `62d4463→e4e3e88` non-board paths are
  `LOGBOOK.md`, `README.md`, 4 `internal/catalog` files,
  `task-board.config.json`. Candidate touches none of the catalog paths
  (`git diff HEAD --name-only -- internal/catalog task-board.config.json`
  empty); `task-board.config.json` byte-identical trunk = HEAD = worktree.
- rev1-vs-rev2: same 55 paths; all 53 non-doc file bodies identical
  between the rev1 and rev2 patches (LOGBOOK/README excluded).
- Whole-Story delta: 55 paths; worktree union (HEAD 34 + leaf tracked 8 +
  untracked 15) diffed against the patch path list: identical.

## Verification (own instruments, rev2 tree)

1. Whole-Story diff: 55/55 paths confirmed; config identical; checkpoints
   signed. Leaf beyond checkpoints = the 23 stated paths (8 tracked +
   15 untracked files). `git status` shows only candidate paths.
2. Section 8.4 derivation: `TestResumeMatrixCoversEverySpecRow` PASS
   (27 parsed rows; table lines 3970–3996). Doc-side plant (Codex/macOS
   A→C in a private module copy) reddens: `specdoc.LoadV060` pin mismatch,
   rc=1 FAIL observed. Code-side narrowing (gemini wsl2 C→A) killed by the
   same test (own-m1, rc=1). Muse macOS-arm64 admits exactly 0.1.0
   (`TestSmokeMuseOnePatchOffPinRefuses` PASS); Antigravity unresolved realm
   fails (`TestSmokeAntigravityUnresolvedRealmFails` PASS); qwen×8 and
   future-plugin×8 row subtests PASS with the strict fake (any adapter call
   would fatal: zero-call refusal proven by construction + green); WSL2 and
   native Windows never collapse (`platformCell` keeps four distinct rows;
   spec line 3998 asserts the rule).
3. Smoke sequence through `provhost.Host.Call` (`Run` in `smoke.go`):
   A-positive (`TestSmokeRowVerdicts/codex-0.147.0-macos-arm64` PASS),
   C-gated (`.../gemini-0.54.4-wsl2-amd64` PASS, resume-plan skipped with
   the 19.3 citation), probe drift / strong confidence / cross-tuple proof /
   unsafe quiescence / unmapped Pi version — all FAIL-verdict PASS.
   Promotion attacks refused: flipped-verdict (digest), resigned promotion
   (consistency), whitespace variant — all PASS. Closed shape (extra member
   refused via `DisallowUnknownFields`), no capability surface
   (`record.go` grep for capab/doctor/enabled: no match;
   `TestSmokeRecordCarriesNoCapabilityClaim` PASS), no secret/raw reference
   (details carry digests/citations only by construction).
   Real-provider-process run byte-identical
   (`TestSmokeThroughRealProviderProcess` PASS).
4. Durability (`Store`/`Load` in `store.go`, no-replace + fsync):
   `TestStoreRealKillBetweenWriteAndSync` PASS twice (real SIGKILL between
   write and sync, retry reuses path, loads byte-identical); `TestStoreReplayIsIdentical`,
   `TestSmokeIdenticalInputsReplayIdenticalBytes`,
   `TestStoreDisagreementQuarantines` PASS. Torn bytes refuse at `Load`
   (digest/closed-shape fail-closed by construction; garbage-input Store
   refuses before create by `VerifyRecord` gate).
5. Refusal census + mutation: every `OutcomeFail` arm in `smoke.go` is
   reached by a named test (row verdicts + the 9 negatives above + params
   refusals); `matrix.go` cells by the derivation test; `record.go`/`store.go`
   by tamper/promotion/whitespace/disagreement tests. `ProbeBuild` /
   `SplitIdentifyResult` consume `decodeValidated*` members only (diff
   read); zero own refusal sites confirmed; accessor tests (4) PASS.
   Shipped `TestSmokeMutantsAreKilled` green twice: 10 KILLED + 1 SURVIVED
   control (27.9 s / 33.8 s). Own narrowing mutants in private copies
   (raw log `own-mutants-raw.log`, all `failed` via rc=1 + FAIL):
   own-m1 matrix gemini-wsl2 C→A killed by derivation (wsl2 amd64+arm64);
   own-m2 probe version-prefix (`[:2]` equal admits 0.147→0.148 drift)
   killed by `TestSmokeProbeMismatchFails` (unexpected identify call);
   own-m3 confidence strong-for-codex killed by
   `TestSmokeStrongConfidenceFails` (unexpected resume call);
   own-m4 digest admits-gated killed by `TestSmokeTamperedRecordRefuses`
   (tampered gemini byte 774 verifies); own-c0 comment control SURVIVED.
   Token-preserving-mutant row: N/A by construction — no production
   source-text gate (derivation reads the specpin-guarded doc; doc plant
   above fails at the pin, not the cell parse). Stated, not skipped.
6. Story-close: `tracecheck` exit 0 — 113 cases, 29/463 clauses;
   `-section 2.4` admitted, exit 0. Bindings 2.4 (4 clause lines at
   ownership 3282–3307), 5.5 sliver (gap + 5.5#3), 7.7 (gap + 3 clauses),
   8 sliver present with honest gaps. README native-resume-smoke section
   present with test commands, no capability claim (no capab/doctor
   surface in package). LOGBOOK newest-first with all Story entries +
   trunk 3lt2xv preserved.
7. Hygiene: no `__pycache__`/`.pyc`; `gofmt -l internal` clean;
   `go vet ./...` exit 0; `GOOS=windows go vet` on smoke+provhost exit 0;
   focused suites + `-race` (smoke/provhost/sessprofile) green;
   `cataloggen -adopted ... -check` exit 0; native-Windows bind rows skip
   loudly (8 SKIPs with `windows native-store bind requires a Windows
   host; cell ... is pinned by TestResumeMatrixCoversEverySpecRow`).

## AC coverage: 16 of 16 rows driven through production entries

Confirmed by the green suites above: (1) probe via `Host.Call(OpProbe)` +
`ProbeBuild`; (2) identify via `OpIdentifySession` + `SplitIdentifyResult`;
(3) bind via `StoreRootFor` + `VerifyIdentityDiscovery`; (4) resume via
`ResolveMapping` + `OpResume` + `DecodeSpawnPlan`; (5) quiescence via
`DecodeQuiesceProof` (safe pass + unsafe fail); (6) A positives; (7) C
gated + promotion refusal; (8) U zero-call; (9) ? zero-call + Muse off-pin;
(10) drift fails closed; (11) cross-tuple refuses; (12) non-A plan gated;
(13) no capability advertised; (14) Appendix B gates; (15) durable
no-replace/fsync/crash/replay; (16) tamper/verdict consistency.

## Refusal-census observations (not blockers, carried from rev1)

- `smoke.go:158` tuple-gate-fail arm has no dedicated off-matrix test
  (matrix/gate agreement covers derived tuples).
- Transport-failure arms are covered by construction (strict fake), not by
  dedicated failing-call tests.
- `deriveVerdict` unexpected-skip→fail is exercised via C-cell paths only.

## Evidence attached

- `TASK-260830-2zvo8m_review-verdict-rev2.md` (this file)
- `TASK-260830-2zvo8m_review-evidence-rev2.tar.gz` (real gzip: cmds log,
  own-mutant raw log + script, path/union/numstat/tracked/untracked/trunk
  lists, README deleted-lines list)

Checklist: live merged checklist is complete; no Stop-The-Line (no external
blocker; rework was autonomous).
