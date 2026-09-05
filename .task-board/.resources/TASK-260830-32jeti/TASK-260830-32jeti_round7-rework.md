# TASK-260830-32jeti round-7 rework (answers round-6 verdict, changes requested)

Candidate base: story branch `task-board/story/STORY-260830-3jqsx1`, uncommitted working tree.
Scope kept to F1, F2, the flake, and logbook. Production delta: `protocol.go`,
`profile.go`, `status.go`, `runner.go`. Test delta: `closed_vocabulary_census_test.go`,
`runner_test.go`. Docs: `LOGBOOK.md` (new 0935 entry + 0926 green-suite correction).

## F1 — three switch-shaped vocabularies folded into the census (blocking, closed)

- `responseMembers`, `profileNames`, `statusStates` are package-level `map[string]bool`
  consulted by membership checks (`checkResponseMembers`, `ProfileMapping`,
  `validStatusState`); the widened-switch mutants C1/C2/C8 have no switch to widen.
- New pinned-document derivations: `TestResponseMembersAreDerivedFromSpec` (§7.2
  success+failure sentences, `ok = true/false` normalized), `TestProfileNamesAreDerivedFromSpec`
  (§7.7 yolo + standard sentences, exactly-one-span guards),
  `TestStatusStatesAreDerivedFromSpec` (§7.5 ProviderTransactionStatus state span).
- Census registration 19 → 22 (both directions); three entry-point refusal rows
  adopted from the reviewer's probe (`DecodeResponse` envelope/`bogus`,
  `ProfileMapping(codex, bogus)`, bogus status state); `TestAllProductionSwitchesAreClassified`
  fails on any new/unclassified production switch; header "by construction" sentence corrected.
- Mutant evidence (cp-aside, sha256-verified restore, `go test . -count=1`, exit 0 runs
  asserted planted by grep): envelope/`bogus`, profile/`bogus`, status/`bogus` each die
  three ways — derivation test, census subtest, entry-point row. Unclassified-switch
  plant dies in `TestAllProductionSwitchesAreClassified`. One vacuous-green run caused by
  a gofmt-realigned plant anchor was caught and re-run with the corrected anchor.

## F2 — ExecRunner kept no judgeable result on stdin EPIPE (blocking, closed)

Two races fixed in `ExecRunner.Run`, per the file's own comments:
(a) `return Result{}, writeErr` before stdout was ever considered — valid correlated
frames discarded on an EPIPE race; now the frame governs and write/close/wait failures
surface only with empty stdout and no known exit code.
(b) Found in this rework by driving `Run` directly (2/20 crash-script iters returned
`read |0: file already closed`): `Wait` closes `StdoutPipe`/`StderrPipe` read ends after
process exit, racing a drain still in flight — this misclassified the stdin-draining
`exit 3` fixture, not EPIPE. Drains now run over `os.Pipe` pairs owned by `Run`.

- `TestExecRunnerKeepsValidFrameWhenPluginIgnoresStdin` (200 calls, zero discards) and
  `TestExecRunnerClassifiesCrashExitDeterministically` (50 crash calls, all
  `plugin exited without a response`) both FAIL on the pre-fix runner (2/200 discarded;
  `runner reported no result`, verified by index-version swap with sha256 restore) and
  pass post-fix through `Host.Call` over `ExecRunner`.

## Flake bound

- Post-fix `go test ./internal/provhost/ -count=1`: 30/30 green (two 15-run chunks).
- Pre-fix (reviewer): 3 red in 50. LOGBOOK 0926 "15/15 green" corrected to one observation.

## Gates (all exit 0)

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./internal/provhost/`, gofmt clean.
- `go test ./... -count=1`: 15 packages green. provhost `-race`: green.
- provhost coverage 86.5% → 86.0% (new judge/pipe branches).
- `git status` delta non-empty (this rework plus the staged leaf work, uncommitted).
