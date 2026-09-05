# TASK-260830-32jeti round-8 rework — F1/F2/F3

Worktree holds all changes uncommitted per the CR shape. `git status`
shows a non-empty delta (my 8 files among the story work); nothing was
committed here.

## F1 — detached descendant holds `Host.Call` past its deadline

- Root cause: the `os.Pipe` rewrite fixed the Wait-closes-StdoutPipe
  race but removed the only thing that unblocked the post-Wait drain.
  `exec.WaitDelay` closes pipes exec itself owns; with caller-supplied
  `*os.File` pipes nothing closes them, so the drain waits for EOF
  until every writer is gone.
- Fix (`internal/provhost/runner.go`): `command.Wait()` runs in a
  goroutine under `select` on the call context; both drains collect
  through a new `collectDrains` under the same context, and closing
  the read ends fails blocked drains at once. A `Process.Kill`
  backstop covers a self-detached direct child, which the group-aimed
  `Cancel` misses. The round-6 EPIPE rule is untouched: with the wait
  reaped and stdout holding a frame, the frame still governs
  (`TestExecRunnerKeepsValidFrameWhenPluginIgnoresStdin` 200/200,
  `TestExecRunnerClassifiesCrashExitDeterministically` 50/50).
- Regression test
  `TestExecRunnerDetachedDescendantCannotHoldCallPastDeadline`
  (perl setsid sleeper holding stdout, 2s deadline): 3/3 red before
  the fix at the sleeper lifetime (20.3–20.7s, frame accepted), green
  after at ~2.0s with `provider_timeout`.
- Anomaly: the first probe (python3, 400ms deadline) passed vacuously
  in 0.79s — interpreter cold start can beat the kill, so no
  grandchild ever held the pipe. Rebuilt on perl with seconds of
  margin; recorded in LOGBOOK 1244.

## F2 — derived sweep replaces the hand-written corpus; raw WTF-8 closed

- `decodeStrictObject` (`internal/provhost/protocol.go`) refuses
  non-UTF-8 first, mirroring `canonicaljson.decodeStrict` order. This
  closes the live raw-WTF-8 class (`\xed\xa0\x80` accepted, silently
  U+FFFD) at every operation entry, not only on the wire
  (`DecodeResponse` already checked `utf8.Valid` first).
- `TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON` enumerates
  239,630 vectors (full BMP x hex case, surrogate x 16 case mixes,
  high x boundary/non-low seconds, sampled positions plus
  all-surrogate member names, full BMP raw with WTF-8 form for
  surrogates, malformed patterns, astral samples); every verdict is
  judged by `canonicaljson`, never by a shared hand expectation.
  Green: 239,630/239,630 agree in ~0.6s.
- Mutants, each planted and restored by cp+sha256 (no checkout):
  SG1 (low ceiling to 0xdc00) diverges on 19,482 vectors; SG5
  (uppercase F to E) on 4,411; UTF-8-gate removal reddens the new
  identity row with err=nil and the sweep on 2,059. Zero NOT_APPLIED.
- Inventory: floor 164 -> 166 — one `frame|not valid UTF-8|""` arm
  (witnessed via `DecodeManifest`) plus its `integrity|status body is
  not valid UTF-8` expansion (witnessed via `DecodeStatusOutcome`).
  166/166 both directions.
- Comments corrected: `surrogate.go` (non-UTF-8 never reaches the
  escape scanner), `identity.go` (decoder agreement pinned by the
  derived sweep, escape and raw dimensions).

## F3 — foreign-major gate precedes the v2 member rules

- Ordering fix, not a doc fix (`internal/provhost/protocol.go`): a
  `foreignMajor` peek (protocol identity plus readable foreign
  version; trusts nothing, refuses nothing) skips the v2 `ok`/member
  rules so the single existing mismatch site decides. A 3.0.0
  envelope with v3 members, without body/error, or without `ok` is
  now `incompatible_protocol`; frames without our protocol identity
  or without a readable version keep the member-rule verdicts
  (boundary rows pin this). No new refusal arm; census floor
  untouched. `doc.go`'s claim is now literally true and unchanged.
- `TestDecodeResponseForeignMajorPrecedesMemberRules`: 6 mismatch
  rows plus 3 fall-through boundary rows. Under the old ordering it
  reddens with exactly the reported misclassifications (`unknown
  member`, `missing member`).

## Gates (exit codes observed directly, no pipes)

- `go build ./...` → 0
- `go vet ./...` → 0
- `GOOS=windows go vet ./...` → 0
- `gofmt -l internal/` → clean (the one flagged file,
  `.temp/rev3-review/enum_vars.go`, is a stale prior-round artifact
  outside this task)
- `go test ./... -count=1` → 15/15 ok, exit 0
- `go test ./internal/provhost/ -race -count=1` → ok, exit 0
- coverage: provhost 85.8%, provider 97.0%
