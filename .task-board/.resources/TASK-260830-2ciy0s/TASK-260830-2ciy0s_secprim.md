# TASK-260830-2ciy0s — safe path/argv/environment primitives (outcome)

Leaf 1 of STORY-260830-1i3qu7. New shared library `internal/secprim`
(Sections 16.3, 16.4, 16.7) plus production wiring into the provider
launch path and the CLI render path. Work left UNCOMMITTED in the story
worktree on top of trunk `157a54c`; no commits made by this leaf.

## Production entry points (call sites)

- `internal/secprim/doc.go` — scope, reuse map, trust-path
  non-duplication rationale, stated bounds.
- `internal/secprim/path.go` — `CheckMemberPath`, `Guard.NewGuard` /
  `Resolve`, `DetectCaseCollision`, `CheckRegularTarget`.
- `internal/secprim/nofollow.go` (+ `open_nofollow_unix.go`) —
  `OpenNoFollowFile`, `OpenNoFollowDir` (Windows file:
  `open_nofollow_windows.go`, Lstat gate with stated TOCTOU bound).
- `internal/secprim/argv.go` — `CheckArgv`, `Command.NewCommand`.
- `internal/secprim/env.go` — `IsEnvName`, `BuildEnv`.
- `internal/secprim/render.go` — `EscapeForTerminal`,
  `RenderForTerminal`.
- `internal/secprim/redact.go` — `Redact` (corpus, PEM, key=value,
  URL userinfo).
- Wiring 1: `internal/provhost/runner.go` — `ExecRunner.Env`
  (nil inherits; non-nil replaces wholesale, built by `BuildEnv`) and
  `CheckArgv` validation of the executable at start.
- Wiring 2: `internal/provhost/spawn.go` — `validEnvName` now delegates
  to `secprim.IsEnvName` (one grammar, two call sites).
- Wiring 3: `internal/cliresult/output.go` — `Log`, `Progress`,
  `Prompt`, and text-mode `Emit` cross `terminalLine`
  (`RenderForTerminal`); JSON mode stays byte-exact.
- Extension: `internal/scalar/path.go` — exported
  `IsReservedWindowsDeviceName` for the member gate (same table the
  absolute-path gate uses).

## Reuse / non-duplication record

- Scalar relative/absolute grammars (incl. recursive encoded
  separators), environ (frames/identity/tuples), axerror detail +
  causal-leak gates, config SSH admission, terminalbackend
  `CheckEntrypoint`, localstore/config AX_* registry: all cited as
  owners in `doc.go`; none re-implemented.
- Trust: NO third path. `provider.trustCandidate` carries canonical
  path + digest + approving owner identity per Section 7.1;
  `terminalbackend.DigestFile` + trust tuple carry platform/adapter/
  manifest/probe digests with no owner fact per Section 4.B. The
  difference is contractual (spec clauses differ), not accidental.
- `IsEnvName` is the one deliberate convergence: provhost delegated to
  the new owner rather than keeping a second copy.

## Coverage — Section 16 MUST rows: 16 of 25 driven, 9 bounds/pre-existing

Fully driven through production entries (13): untrusted-member
grammar, traversal, absolute, drive/UNC, alternate streams,
encoded separators (+encoded-dot/overlong extension),
symlink/reparse escape at validation and commit, case-fold
collisions, unmanaged replacement, device/FIFO/socket refusal,
argv-array/no-shell, terminal render escaping, TOCTOU dir-handle
primitives (unix).
Partially driven — gate wired, consumer or policy bound (3):
stderr/log redaction (key/PEM arms wired into the emitter; corpus arm
tested, no in-repo secret corpus yet), launch environment
(allowlist builder + runner field wired; default policy unset —
inherits until supplied), structured launch (argv+env new; fixed
`ax pane` entry and cwd launcher pre-existing/absent).
Pre-existing owners, cited not re-driven (3): SSH fixed entry +
encoding (config), plugin trust (provider/terminalbackend), plugin
stdout validation (provhost).
Declared bounds, no in-repo path at M0 (6): tombstone deletion
scope, archive staging root + re-validation after extraction,
clone-bundle exclusions, metric labels, audit retention, cwd policy.

Section 19.4 security ACs: AC-SEC-003 and AC-DIR-TERM-001 driven at
gate level (member/symlink/case/special-file/command-injection
refusals; hostile-render inertness; argv/env admission); AC-SEC-001,
AC-DIR-SEC-001, AC-OBS-001, AC-V043-SEC-001 are bounds (no
transfer/record/metric/log implementations to exclude from);
AC-SEC-002, AC-PATH-001, AC-PATH-002 are pre-existing owners
(config, localstore).

Crash/idempotency: no surface. The package is pure functions plus
read-only opens plus terminal writes; it mutates no durable state,
keeps no cache, owns no journal.

## Mutant battery: applied 15, killed 15, NOT_APPLIED 0, COMPILE_FAIL 1*

| Mutant | Narrows the gate to | Named failing test |
|---|---|---|
| M1 drop encoded-dot arm | admits exactly `%2e` members | TestCheckMemberPathRefuses/encoded_dot* (4 subtests) |
| M2 drop ADS arm | admits exactly `ab:c` on Windows | …/windows_ads, …/windows_ads_nested |
| M3 drop device-name arm | admits exactly CON/LPT… | …/windows_con* (5 subtests) |
| M4 drop argv NUL arm | admits exactly NUL argv | …/nul_executable, …/nul_argument |
| M5 drop literal-collision arm | admits exactly shadowing literals | TestBuildEnvRefuses/literal_collides |
| M6 drop DEL arm | admits exactly DEL | …/del + OutputBytes sweep |
| M7 drop bidi arm | admits exactly U+202A–U+202E | …/bidi_override + OutputBytes sweep |
| M8 drop `password` scrub key | admits exactly that key | TestSensitiveKeyListIsReviewed + …/equals, json_pair, case_folded, json_spaced |
| M9 redact every BEGIN block (widening) | certificates redacted | TestRedactPrivateKeyBlocks |
| M10 drop managed-set arm | admits unmanaged replacement | TestGuardManagedSet (audit stays green — derivation follows source; recorded split) |
| M11 token-preserving shell (`"s"+"h"` + os/exec, no literal) | static scan passes | TestPackageLaunchesNoProcess (behavioral half kills it) |
| M12 drop `secret` census token | secret-named sites underived | TestSecretSiteRosterIsComplete (orphans) |
| M13a drop all terminalLine uses | — | COMPILE_FAIL (unused import; re-done as M13b) |
| M13b drop Log-only wiring | raw Log output | TestTextEmissionNeutralizesHostileContent |
| M14 ignore runner.Env | silent inheritance | TestExecRunnerDeliversExactlyTheBuiltEnv |
| M15 rename class row | orphaned row | TestExclusionClassRosterIsComplete |

\* M13a is the single COMPILE_FAIL entry: removing every use broke the
import. It was re-applied narrowly (M13b) and killed. No survivors:
every surviving-mutant question is answered by a named failing test.

## Gates (real exit codes, standalone processes)

- `go test ./... -count=1` → exit 0 (20 packages ok; log attached).
- `go test secprim+provhost+cliresult+scalar -race -count=1` → exit 0.
- `go test ./... -cover` → exit 0 (secprim 94.5%, provhost 85.8%,
  cliresult 95.5%, scalar 90.1%).
- `go vet ./...` → exit 0. `GOOS=windows go vet ./...` → exit 0
  (after adding the Windows fifo stub; unix-only failure was
  `makeFifo` undefined — fixed, re-verified).
- `gofmt -l internal/` → clean. `tracecheck` → exit 0, figures
  unchanged (94 acceptance cases; none added to the registry by this
  leaf — the conformance-CI leaf owns the gate).

## Residue (explicit bounds, not hidden)

1. Windows no-follow is check-then-open (no O_NOFOLLOW in Go);
   owner-only staging roots narrow the race; stated in code.
2. Corpus redaction needs a caller-known secret list; no in-repo
   caller holds credentials yet (no config credential fields, no
   logger/doctor). `RenderForTerminal(line, nil)` still applies
   key/PEM arms everywhere wired.
3. Compound keys outside the 27 (`db_password`), bare `key`/`auth`/
   `pwd`, non-PEM key material, combining-mark floods, emoji width:
   explicitly out of the scrubber/escaper; corpus arm is the fallback.
4. `auth`-substring declarations are outside the census instrument
   (Authority blind spot, pinned by TestSecretScanBlindSpot);
   `auth_json`/non-derived table keys are covered behaviorally by
   TestSecretAxerrorTableRefused instead.
5. Operator default allowlist policy is unset (inherit until
   supplied); provider needs cannot be derived with no provider
   implementation in-repo.
6. Guard/OpenNoFollow have no in-repo staging/extraction consumer yet
   (no archive code at M0); the storage-boundary leaf
   (TASK-260830-3qrfjp) is the designated first consumer.

## Follow-ups for the orchestrator (no board elements created)

- F1: storage-boundary leaf should consume Guard/OpenNoFollowFile
  for staging/extraction when archive code lands.
- F2: provider-conformance leaf should set the ExecRunner allowlist
  policy (which names) once provider runtime needs are known.
- F3: doctor/logging work should supply secret corpora to
  RenderForTerminal/Redact at persistence points.
- F4: conformance-CI leaf owns traceability-registry additions for
  the new AC-level gates (this leaf added none).

## Tree state

HEAD `157a54c` unchanged (no commits). Modified: README.md,
internal/cliresult/output.go, internal/provhost/{inventory_test,
runner, spawn}.go, internal/scalar/{path,scalar_test}.go. Added: internal/secprim/ (10 prod + 10 test files; 42 tests + 84 subtests green),
internal/cliresult/render_wiring_test.go,
internal/provhost/{env_agreement,runner_env}_test.go.
