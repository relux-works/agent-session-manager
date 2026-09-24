# TASK-260830-35urbp — review verdict, Change Request revision 1

**Verdict: CHANGES REQUESTED** (routed `to-dev`). Two P1 classes, four P2, four P3.

- Reviewer run: RUN-260921-0245e5 (claude-opus-5 max), reviewing CR-TASK-260830-35urbp-1 rev 1
  published by RUN-260921-e93883 (muse-spark max).
- Reviewed bytes: base `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` at review
  time, no trunk drift), candidate tree `e0d9a81b236c9add75805d94df3bc31dbc256562`, patch
  sha256 `650ea3b60b58d799255882a38daffa5118f3e6839b3f715207c0941e35716ba2` (recomputed from the
  tree pair, equal). The live Story worktree's uncommitted tree was recomputed through a
  temporary index and equals the CR tree. Every measurement below was taken on detached,
  immutable copies of that tree (`candidate`, `mutant-copy`, `probe-copy`, all cut from a
  dangling review commit of the CR tree); the live worktree, its index, branch and HEAD were
  not touched, no product edit, commit, checkpoint or integration was performed.
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §4.2 (1403-1449) with §4.C,
  §4.D, §4.E; the task record's v0.5.0 Scope is stale. The candidate cites the pinned file
  only (checked: no v0.5.0 line number, every quoted sentence is present in the pinned file).

## What was reproduced (all on tree e0d9a81b, PATH without any tmux binary, TMUX/TMUX_TMPDIR unset, no tmux process on the host)

| Check | Result | Log |
| --- | --- | --- |
| Configured suite, 30 commands (`spawn.worktree_isolation.validation.commands`) | 30/30 green, every command rerun by this reviewer (1-3, 26-28, 30: `static-gates.log`; 4: `cmd04-06-test-cover.log` + `cmd04-test-v.full.log`, 45 pkgs, 2847 PASS / 0 FAIL / 2 pre-existing SKIP outside this package; 5 race: `cmd05-race.log`, 45 ok, 6m56s; 6 cover: 45 ok; 7-25: `cmd07-25-fuzz-tracecheck-catalog.log`; 29: `cmd29-task-board-validate.log`, exit 0 with the board's pre-existing 180 issues) | `logs/` |
| Package suite, tmux unresolvable | `go test ./internal/tmuxserver -count=1 -v`: 51 top-level tests, 89 PASS lines, 0 FAIL, **0 SKIP**. The determinism claim holds: no row's witness skips without tmux | `pkg-tests-notmux-v.log` |
| Package coverage | 87.5% (matches the results); uncovered production blocks listed in the findings | `tmuxserver-cover.out` |
| Producer harness, two passes | 31/31 both passes (30 KILLED exit 1, C-control SURVIVED exit 0), verdict lists identical, tree clean after each pass, zero `__pycache__`/`.pyc` | `harness-pass1/`, `harness-pass2/` (one raw log per row per pass) |
| Reviewer battery, 14 rows × 2 passes | identical verdicts both passes; see §M | `reviewer-battery-pass1/`, `reviewer-battery-pass2/` |
| Reviewer probes, 16 tests | all behave as predicted; see §P | `probes-rev1.log`, `probe-copy/.../zz_review_probe_test.go` (in the evidence archive) |
| **Hosted CI gate `GOOS=windows go vet ./...`** | **FAILS** — see P1-A | `static-gates.log`, `windows-test-compile.log` |
| Trunk drift / boundary / hygiene | `git diff --name-only 799c338 origin/main` is empty; `internal/traceability` untouched; no `__pycache__`/`.pyc` in the CR tree or any copy; gofmt/vet clean; `GOOS=linux`/`GOOS=windows go build` green | `static-gates.log` |

## P1 — must fix

### P1-A. The candidate breaks the repository's hosted CI: the Windows test binary does not compile

`GOOS=windows GOARCH=amd64 go vet ./...` (and `go test -c`) fail on the candidate:

```
internal/tmuxserver/runtime_test.go:103:14: undefined: effectiveUID
internal/tmuxserver/runtime_test.go:104:2: undefined: effectiveUID
internal/tmuxserver/runtime_test.go:105:17: undefined: effectiveUID
```

`runtime_test.go` carries no build constraint but reassigns the `effectiveUID` seam that only
`ownership_unix.go` (`//go:build !windows`) declares. `.github/workflows/ci.yml:85` ("Vet
Windows build-tagged sources", Linters job) and `ci.yml:354` ("Vet Windows build (includes
test files)", Windows compile-only job) both run exactly `GOOS=windows go vet ./...`; the
README's CI table (README.md:805) documents that gate as "test files included". Trunk is
clean under it; the candidate is the only failing package. The local 30-command suite runs
`GOOS=windows go build` (which compiles no test files), which is why the CR construction
stayed green — the DoD line "build/validation commands run after changes and build not
broken" is not met against the repository's documented gate set.

The results' claim "Row 15's Windows-only tripwire is compile-verified (`GOOS=windows go
build` green) and executes only on Windows" is false in both halves: `go build` never
compiles `windows_bound_test.go`, and the package's test binary cannot be built for Windows
at all, so `TestWindowsCustodyBoundIsStated` can never execute. Row 15 is unmeasured.

Fix: constrain the seam-using tests (`//go:build !windows`, or move the `effectiveUID`
reassignment into a unix-only test file) and prove it with `GOOS=windows go vet ./...` in the
results. See also P2-B for what the Windows arm should be.

### P1-B. The attestation/readiness model forks a landed gate with a weaker copy, and the weaker copy admits what the spec forbids

`readiness.go` re-models the §4.2 credential-realm decision as five loose fields
(`LiveAttestation{SentinelPassed, SmokePassed, ServerGeneration, ProviderBuild, MacOSVersion}`,
readiness.go:30-36) decided by `CheckLiveAttestation` (readiness.go:46-57) and a
`BrokerStatus{Authenticated, Attested, Attestation}` (readiness.go:64-68). The repository
already owns exactly this decision:

- `internal/terminalbackend.Reconcile` / `Registry.AdmitProbe` (manifest.go:1683, :2059) admit
  the `credential_capable_execution_realm` Capability Evidence row with signature
  verification, liveness (`checkEvidenceLiveness`, manifest.go:1923), generation binding
  (`checkProbeGeneration`, manifest.go:1771, `CodeStaleGeneration`), `os_version`,
  `provider_build`, `sentinel_result`, `provider_auth_smoke_result` — the §4.D object whose
  members are the §4.2 sentence "Evidence MUST bind the exact tmux server generation,
  provider build, and macOS version" (SPEC 1428-1429; §4.D 1300: "wrong-generation evidence
  disables the claim").
- `internal/axpane.checkRealm` (decide.go:913-928) applies it to the §4.2 background-caller
  rule: `admitted.Has(credential_capable_execution_realm)`, `Realm.ServerGeneration ==
  Backend.RawGeneration` (stale generation refuses), host-binding/provider-build cross-bind,
  and the same `axerror.NewRealmEvidenceUnavailable` composition the candidate re-declares in
  `backgroundUnavailable` (acquire.go:237-252). Its `Realm` type documents the landed design
  decision the candidate reverses (decide.go:97-118): "There is deliberately no member for a
  cached sentinel result, a managername observation, or any attested-server boolean: none
  authorizes resume, so none is an input ... the bare boolean previously carried here never
  authorizes and is removed." `internal/axpane/TRACEABILITY.md:96-97` already claims the
  §4.2 rows "Background caller without attested server" and "Cached evidence never
  authorizes" against the landed reconcile path.
- The predecessor this leaf was told to sit under, `internal/termbind.ResolveEvidence`
  (resolve.go:77), returns a `terminalbackend.Admitted` — the type this leaf should have
  consumed as its "attested" fact.

The candidate reuses two string constants from axpane and forks the gate. The fork is weaker
in the one dimension this leaf's own rows claim: the "binding" arm (readiness.go:53) checks
that three strings are non-empty, never that they bind to anything. Reproduced through the
production entry `Acquire` on both paths (`probes-rev1.log`):

- `TestReviewHoleForegroundAttachAdmitsWrongGeneration`: request generation `generation-7`,
  probe reports a running server attested for `generation-6` / yesterday's build / yesterday's
  macOS → `attached-running`, admitted.
- `TestReviewHoleBackgroundBrokerAdmitsWrongGeneration`: same on the broker path → `broker`,
  admitted (§4.4 SPEC 1485 also requires broker readiness to be same-user and
  generation-bound; `BrokerStatus` carries neither fact).
- `TestReviewHoleCachedObservationReplayedAsLiveAuthorizes`: a `CachedObservation` copied
  field-by-field into `LiveAttestation` authorizes attach. The "distinct unconvertible type"
  (readiness.go:14-19, doc.go, README) is a struct literal away from conversion; nothing in
  the package can tell a cached observation from a live one, which is precisely what
  reconcile's liveness + generation digest exist for (SPEC 1445-1449).

Consequences for the evidence table: rows 24, 34, 38 drive the fork's predicate, not the
spec's; row 37 ("Generation-, build-, and macOS-unbound observations refuse") is driven only
for empty strings — a wrong generation is admitted, so the row is not driven as stated; rows
32-33 (hint and cache "authorize nothing") are vacuous: `Request.Hint` and `Request.Cached`
(acquire.go:101-105) are never read by any production statement (grep: zero reads outside
the struct), so the tests prove non-use of two fields the package itself added and nothing
about the spec's cached-sentinel rule, which lives on the reconcile/generation path. README
("bound to the exact server generation"), doc.go ("the live-attestation authorization gate
over which cached sentinel and managername observations carry no authority") and
TRACEABILITY rows 34-39 are prose that outruns the tests (rejection pattern f).

Fix (design, not a patch): drop `LiveAttestation`/`CachedObservation`/`ManagerNameHint`/
`BrokerStatus.Attested` and take the landed admission as input — a
`terminalbackend.Admitted` (or the `termbind.ResolvedEvidence` that carries it) plus the raw
generation, deciding "attested" as `Admitted.Has("credential_capable_execution_realm")`
composed with the generation equality axpane already enforces, or call into axpane's realm
predicate if it is exported for this purpose. The broker "authenticated" fact is new and may
stay, but it must be typed and generation-bound per §4.4, not a bare boolean. Re-derive rows
24-39 from the composed gate and state as a BOUND whatever the leaf still cannot prove
without a live broker.

## P2 — must fix or state as a bound

### P2-A. Nine narrowing mutants survive the committed suite on gates the matrix counts closed

Reviewer battery (`reviewer-battery-pass{1,2}/verdicts.log`, raw per-row logs with exits).
Each row was run twice per pass: against the committed suite as shipped (full package, no
`-run` mask) and again with the reviewer's probe file added, so every SURVIVED below is
proven to be a real narrowing that one test kills — not an unmeasurable arm.

| Row | Gate narrowed (site) | Committed suite | With probe | Matrix row it exposes |
| --- | --- | --- | --- | --- |
| R-acquire-swallow-commit-refusal | `Acquire` admits every commit-phase custody refusal (`err != nil && dir == ""`, acquire.go:137) | SURVIVED | KILLED | 4-9 are pinned at `EnsureRuntimeDir` only; `acquire.go:137-139` is uncovered — no Acquire test stages any runtime-dir refusal |
| R-verify-symlink-fallback | verify side follows a symlink leaf (runtime_unix.go:107) | SURVIVED | KILLED | row 6 covers the commit side only; the producer's N-containment-symlink mutates commit only |
| R-override-tmpdir | override gate admits the `TMUX_TMPDIR` value (socket.go:86) | SURVIVED | KILLED | rows 19-22 pin four literal values of one `override != ""` gate; the fifth member — the one tmux uses to relocate its default socket directory — is unpinned |
| R-collision-inherited / -tmuxenv / -tmpdir / -conventional | collision gate admits one member each (socket.go:108) | SURVIVED ×4 | KILLED ×4 | row 23 is pinned on `DefaultPath` only (N-collision) |
| R-argv-env-value | argv gate admits the TMUX environment value (argv.go:17) | SURVIVED | KILLED | row 47 "4 subtests" pins four of five members; reach is direct `BuildArgv` only |
| R-fchmod-arm-unmeasured | post-mkdirat `Fchmod` skipped for non-root (runtime_unix.go:55) | SURVIVED | KILLED | row 1's "enforced explicitly because mkdir honors umask" is unobserved: the suite runs under umask 022, where mkdirat alone yields 0700 |

Three additive fallback plants on the background miss path (re-enter foreground creation and
return `spawned`; create when `!NeedsCredentials`; create when `Platform != macos`) were all
KILLED by the committed spies — the no-fallback rule is genuinely pinned along the caller,
credential and platform axes. One delete-only row, R-ambient-vocabulary-dead-arm, SURVIVED
both runs: `ObserveAmbient`'s "ambient vocabulary" refusal (socket.go:52-54, uncovered) guards
two package constants that are valid env names and is unreachable; row 18's sentence
"Unknown variable names refuse rather than being silently skipped" describes no behaviour.

### P2-B. The Windows arm implements a tmux runtime directory where the spec forbids claiming tmux, with weaker gates

§4.3 (SPEC 1458): "Native Windows MUST NOT claim tmux or tmux resurrection." The unix commit
refuses `PlatformWindows` on a unix host, but `runtime_windows.go:23-42` creates an
`ax-tmux` runtime directory on a Windows host with `os.MkdirAll` (builds parents — the
unix arm's comment says the entry "never builds parents"), no mode gate, no ownership gate,
and a stated "custody subset" bound (B2) where the right behaviour is a typed refusal of
`PlatformWindows` on every host. `_ = filepath.Clean` (runtime_windows.go:40) is an
import-keeping hack. Combined with P1-A the whole Windows story of this leaf is untested and
unrunnable. Fix: refuse `PlatformWindows` in `lexicalRuntimePath` (or the platform gate) with
`tmux_invalid_arguments`, delete the Windows commit/verify arms and the tripwire, and keep the
package compiling and vetting for `GOOS=windows`.

### P2-C. The AC's "create dedicated tmux -S servers" has no production execution path, and the socket write has no crash evidence — neither is stated as a bound

`Dependencies` (acquire.go:59-66) has no production implementation anywhere in the tree; the
comment "Production wires live tmux and broker probes; tests wire fakes" (acquire.go:57)
describes code that does not exist. `BuildArgv` + `Acquire` decide; nothing execs. The brief
named the server socket a durable write requiring crash/idempotency evidence or an explicit
purity bound; B1 states determinism, not the missing adapter. This is a defensible split
with the lifecycle leaf, but it must be written down: state as a BOUND that this leaf ships
the decision core and argv only, that the exec/probe adapters and the socket's crash
evidence are owned by TASK-260830-1c28dz, and remove the "production wires" sentence.

### P2-D. Dead inputs, dead arms and a doc comment that contradicts the code

- `Request.NeedsCredentials` (acquire.go:90) is never read; `Hint` and `Cached` (see P1-B)
  are never read. Three request members whose only purpose is to be proven unused are API
  surface, not evidence.
- `acquireForeground`'s `checkCreationAllowed` call (acquire.go:201-203, uncovered) can never
  refuse — the caller was already classified foreground.
- `VerifyRuntimeDir` has zero production callers; its doc (runtime.go:50-54) says "Acquisition
  calls this instead of Ensure: ... a missing directory refuses rather than being silently
  created on the authorizing path", while `Acquire` (acquire.go:136) calls `EnsureRuntimeDir`
  on every path — including the background path, so a background caller performs the durable
  directory write. Its refusal arms for platform-host, root-open, handle-mismatch and lexical
  failures (runtime_unix.go:96-106, runtime.go:56-58) are all uncovered. Either wire Verify
  into the authorizing path as the comment promises, or delete the comment and the entry.

## P3 — fix in the same rework

- P3-A. The results state "the `task-board.config.json` in this worktree carries no
  `validation.commands` section (count 0, verified by read), so there is no 30-command suite
  to run here". The 30 commands are at `spawn.worktree_isolation.validation.commands`; the
  producer did not run the configured suite; the CR construction did (attached log footer
  `required=30 green=30`), and this review reran all 30. Report what was read accurately.
- P3-B. Crash convergence is umask-conditional and unstated: a leaf left between `Mkdirat`
  and `Fchmod` under an owner-bit-stripping umask (probe
  `TestReviewObserveCrashRetryUnderRestrictiveUmaskNeverConverges`: mode 0600) refuses
  `runtime mode` on every retry and never converges, because existing leaves are verified,
  never repaired. State the bound (umask ⊆ 0077) or narrow the repair to
  owner-only-and-narrower-than-0700 leaves owned by the caller.
- P3-C. A symlinked intermediate component inside the root path is followed
  (`TestReviewObserveSymlinkedIntermediateInRootIsFollowed`); only the root's last component
  is refused (row 9). This matches secprim's own root open and is acceptable if stated as the
  same bound; today TRACEABILITY says nothing.
- P3-D. `BuildArgv`'s `secprim.CheckArgv` refusal (argv.go:32-34) is uncovered and, on this
  filesystem, reachable only by a direct call with an invalid-UTF-8 runtime dir (APFS refuses
  such names, so not through `Acquire`); state it or drop the arm.

## Coverage ratio measured against the 54-row table

51 of 54 rows are driven as stated; rows 15 (Windows tripwire cannot compile, P1-A), 18
(refusal half unreachable) and 37 (measures emptiness, not binding) are not. Of the 51, rows
24, 32, 33, 34 and 38 drive the forked boolean predicate rejected in P1-B rather than the spec
clause they cite. Nine gates counted closed are pinned on one member or one side only (P2-A).
The producer's "54 of 54" is therefore inflated by three and rests on a fork for five more.

## Determinism, forks, boundary — what held

- Determinism held: 0 SKIP with tmux unresolvable; every witness is deterministic; no
  real-tmux witness exists (B1, honest).
- Custody gates held under attack: FIFO leaf and FIFO root refuse without blocking
  (`O_DIRECTORY` refuses at lookup); symlink leaf refuses on both entries; widened mode,
  foreign UID and handle mismatch refuse; the real-SIGKILL crash test is genuine (child runs
  the production entry and kills itself in `AfterMkdir`).
- The no-fallback rule held against three fallback shapes beyond the producer's D row.
- Path grammar, argv shape, UUIDv7 session and the `capability_unavailable` typed refusal are
  composed from scalar/secprim/axerror, not forked; `internal/traceability` untouched;
  README section carries no CLI/capability claim; LOGBOOK entry is newest-first.

## Rework scope (for the producer)

1. Make the package compile and vet for `GOOS=windows` including test files
   (`GOOS=windows go vet ./...` green) and refuse `PlatformWindows` on every host per §4.3;
   delete the Windows commit/verify arms and the tripwire (P1-A, P2-B).
2. Replace `LiveAttestation`/`BrokerStatus.Attested`/`CachedObservation`/`ManagerNameHint`
   with the landed admission (`terminalbackend.Admitted` from
   `Reconcile`/`termbind.ResolveEvidence` + raw generation equality as in
   `axpane.checkRealm`); remove the `Hint`/`Cached`/`NeedsCredentials` request members;
   rewrite rows 24-39 against the composed gate, with a wrong-generation negative and a
   replayed-cached-evidence negative driven through `Acquire`; fix README/doc.go/TRACEABILITY
   prose to match (P1-B, P2-D).
3. Add committed killers for the nine surviving narrowings (an `Acquire` test that stages a
   commit-phase custody refusal; a verify-side symlink leaf; `TMUX_TMPDIR` as override; a
   collision on each of the five members through `Acquire`; the fifth member at the argv gate;
   a umask-0177 create) and ship the corresponding rows in `mutant_harness.py` (P2-A).
4. State the bounds: decision-core-only delivery with the exec/probe adapters and socket
   crash evidence owned by the lifecycle leaf; umask-conditional crash convergence;
   symlinked-intermediate root; the `CheckArgv` arm (P2-C, P3-B..D). Delete or reach the
   dead arms (P2-D).
5. Correct the results: the configured suite exists (30 commands) and must be reported as run;
   report the measured ratio with the three undriven rows named (P3-A).
