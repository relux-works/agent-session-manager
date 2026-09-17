# Review verdict — TASK-260830-2zvo8m rev1 (implement-native-resume-smoke-framework)

Reviewer: muse-spark xhigh. CR `CR-TASK-260830-2zvo8m-1` rev1,
candidate tree `f6bde3c2abb82ad86fe525428c4bf54127b66830`
against trunk `e4e3e8834675cf3814effd3b2673b4931dfbf311`.
Patch sha256 `6f49e6d21229878b3977b5349df3628e3d343e1e16befffa6cede010cef92584` confirmed.

## Verdict: CHANGES REQUESTED → `to-dev`

Product code is fully verified (see evidence below). The candidate is
refused for two P1 Story-close regressions: it deletes landed trunk
content. Repair is mechanical (restore two hunks from `HEAD`), no
product-code change needed.

### P1-1 — LOGBOOK trunk entry deleted

Worktree `LOGBOOK.md` lost the entry
`### TASK-260917-3lt2xv — release-agnostic cataloggen check via -adopted`
(header + CONTRACT/GATE/TESTS/DOCS bullets), present at line 8 in both
`HEAD` (54ed1b3) and the trunk base. Entry count is 125 in both versions
because the new `2zvo8m` entry replaced it 1-for-1. The review brief
requires trunk's 3lt2xv entry preserved; the producer results claim
"LOGBOOK order conflicts resolved trunk-first, content preserved" —
that claim is false for this entry.

Repair: re-insert the 3lt2xv block from
`git show HEAD:LOGBOOK.md` between the new `2zvo8m` entry and the
`3uzfyn` entry (newest-first order).

### P1-2 — README `-adopted` docs deleted

The leaf removed trunk's 3lt2xv README docs in two places, both present
in `HEAD` and the trunk base, absent in the worktree:

1. Regenerate block: `cataloggen -adopted -output
   internal/catalog/catalog_gen.go -check` plus the paragraph "The
   `-adopted` form derives the metadata and lock inputs …", replaced by
   the explicit `-metadata`/`-contracts` form only.
2. Validation toolchain table row: the `-adopted … (-metadata/-contracts
   select the same inputs explicitly)` command replaced by the explicit form.

`-adopted` is live production behavior (`run()` in
`internal/catalog/cmd/cataloggen`, covered by `main_test.go`
`TestRunAdopted*`); both forms exit 0 on this tree (observed). The
validated command 21 is the `-adopted` form, which the README no longer
documents.

Repair: restore both `-adopted` hunks from `git show HEAD:README.md`.

No P2. No Stop-The-Line (rework is autonomous, no external decision needed).

## What was verified green (own instruments, live tree read-only)

- Whole-Story delta: 55 paths trunk→candidate confirmed; worktree tracked
  files equal candidate (`git diff --quiet f6bde3c --` excluding the 15
  new-file paths that are untracked-in-worktree/tracked-in-candidate:
  exit 0); leaf paths beyond checkpoints exactly the 23 stated paths;
  `task-board.config.json` byte-identical to trunk and HEAD;
  checkpoints `a6dfaf2`, `54ed1b3` signatures Good (oparin@me.com).
- `go test ./internal/resumesmoke/ -count=1`: exit 0 (26.2 s full suite,
  includes shipped mutant harness + real-kill test).
- `go test ./internal/provhost/ ./internal/sessprofile/ -count=1`: exit 0.
- `go test ./internal/sessprofile/ ./internal/traceability/... -count=1`:
  exit 0 (tracecheck suite 70.5 s).
- Shipped `TestSmokeMutantsAreKilled`: green twice (full suite + focused
  rerun, 23.2 s): 10 narrowing mutants KILLED, harmless control SURVIVED.
- Own narrowing mutants (private /tmp module copies, checkout never
  written; raw log `review-mutants-raw.log`):
  m1 matrix admits gemini-wsl2 C as A → KILLED by
  `TestSmokeRowVerdicts/gemini-0.54.4-wsl2-amd64` (rc=1);
  m2b record digest admits gated verdict → KILLED by
  `TestSmokeTamperedRecordRefuses` (rc=1);
  m3 confidence admits strong for codex → KILLED by
  `TestSmokeStrongConfidenceFails` (rc=1);
  m4 profile vocabulary admits turbo → KILLED by
  `TestSmokeParamsRefusals/bad_profile` (rc=1);
  c0 harmless comment control → SURVIVED as expected.
  (One rejected probe design disclosed: m2 digest-prefix-compare
  SURVIVED because single-bit flips avalanche SHA-256; superseded by m2b.)
- Spec-derivation sensitivity: flipping `Codex / macOS | A / C / C` to
  `C / C / C` in a private doc copy reddens
  `TestResumeMatrixCoversEverySpecRow` (FAIL observed).
- 27 Section 8.4 rows confirmed in `SPEC.v0.6.0.md` table; cell mapping
  independently matched provider by provider (codex A×4, claude A×3+C,
  gemini A/macOS+linux/windows C/wsl2, muse pin + C×3 + ?, antigravity
  C×4, pi A+C×3, qwen U, future ?); WSL2/Windows never collapse
  (`platformCell` keeps all four platforms distinct).
- Zero-call refusals: qwen×8, future-plugin×8, not-a-provider×8 subtests
  plus `TestSmokeMuseOnePatchOffPinRefuses` all PASS; refused shape
  asserts exactly one check (`resume-cell=pass`), no digests, nil calls,
  and the fake has no handler scripted so any call would fatal.
- Promotion attacks: `TestSmokePromotionAttemptRefuses` (flipped +
  resigned), `TestSmokeInconsistentFailRefuses`,
  `TestSmokeWhitespaceVariantRefuses` PASS in suite.
- Closed shape / no capability: `TestSmokeRecordCarriesNoCapabilityClaim`
  (16 members, no capab/doctor/enabled) PASS; canary asserted on every
  row; real-process run byte-identical (`TestSmokeThroughRealProviderProcess`) PASS.
- Durability: `TestStoreRealKillBetweenWriteAndSync` PASS 3× (suite + 2
  focused reruns); `TestStoreDisagreementQuarantines`,
  `TestStoreReplayIsIdentical`, crash-hook tests PASS in suite.
- Provhost accessors: `ProbeBuild`/`SplitIdentifyResult` consume
  `decodeValidated*` members only (read `probe.go`/`identity.go` diff);
  zero own refusal sites confirmed; `accessors_test.go` PASS in suite.
- Traceability: `tracecheck` exit 0, `113 cases, 29/463 clauses`
  (matches pinned test figures); 12 added acceptance cases enumerated
  (`resumesmoke-matrix/run/record/store`, `provhost-resume-tuple-gate`,
  plus 7 predecessor cases); all 12 clause excerpts for 2.4/5.5/7.7/8
  found in the pinned doc (whitespace-normalized); all 8 production
  declarations exist; self-minted binding probe refused by tracecheck.
- Hygiene: `gofmt -l internal` clean; `go vet` (incl. `GOOS=windows`
  on resumesmoke+provhost) exit 0; `-race` on focused smoke tests exit 0;
  `cataloggen -adopted -check` and explicit form both exit 0;
  native-Windows rows skip loudly
  (`windows native-store bind requires a Windows host; cell "A" is pinned
  by TestResumeMatrixCoversEverySpecRow`); no `__pycache__`/`.pyc`;
  `git status` shows only the 23 candidate paths; candidate UNCOMMITTED.

## Coverage ratio

16 of 16 producer AC rows driven through production entries — confirmed
by the green suites above (row verdicts, probe/identify/bind/quiescence/
resume negatives, Appendix B gates, durability, tamper/consistency).
Story-close: ownership bindings 2.4 full 4/4, 5.5 sliver 1/3, 7.7 partial
3/4, 8 sliver 4/12 with honest gaps — accepted; LOGBOOK/README
preservation — FAILED (P1-1, P1-2).

## Refusal-census observations (not blockers)

- `smoke.go:158` tuple-gate-fail arm has no dedicated test reaching it
  (matrix/gate agreement covers derived tuples; any matrix-admit vs
  gate-refuse divergence would need an off-matrix version probe).
- Transport-failure arms (`unframeable`/`call failed` in probe/identify/
  resume) are covered by construction (params validation + strict fake),
  not by dedicated failing-call tests.
- `deriveVerdict` unexpected-skip→fail arm is exercised only via
  C-cell gated paths, not as a standalone skip violation.

## Rework scope for producer

Restore the two hunks (P1-1, P1-2), re-run `go test
./internal/resumesmoke ./internal/provhost ./internal/sessprofile
./internal/traceability/... -count=1`, re-verify `git status` shows only
candidate paths, republish rev2. No product-code change required.
