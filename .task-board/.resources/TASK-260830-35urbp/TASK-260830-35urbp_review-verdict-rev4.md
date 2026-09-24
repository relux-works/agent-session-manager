# TASK-260830-35urbp — review verdict, Change Request revision 4

**Verdict: CHANGES REQUESTED** (routed `to-dev`). No P1. Two P2 classes, two P3. Every rev3
finding (P2-A/B/C, P3-A..E, the four observations) is closed with driven evidence and stays
closed; production behaves correctly at every point this review drove. What remains is one
narrowing survivor on the classification gate this revision introduced (the second half of the
rev3 P2-B split is unmeasured because the test oracle looks through the `capability_unavailable`
wrapper), one free input that re-opens the §3.2 path per call at the entry, one platform-axis pin
gap, and one censusable sentence that is false. The rework is small and enumerated at the end.

- Reviewer run: RUN-260921-30d48f (claude-opus-5 max), reviewing `CR-TASK-260830-35urbp-4`
  revision 4 published by RUN-260921-1ba150 (muse-spark max) after the rev3 rework brief.
- Reviewed bytes: base `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` after a fresh
  `git fetch origin main`; `git diff --name-only 799c338 origin/main` empty — no trunk drift),
  candidate tree `c978fd701b1d50651db8d64efbd1b011504d75d7`, patch resource sha256
  `ecc3f693a05ca97532253b8b37cd3fca9ca030dba662a7b54c4d4ad155a0df0b` (recomputed over the
  materialized resource, equal to the CR record). The live Story worktree's uncommitted tree was
  recomputed through a temporary index copy before and after this review and equals the CR tree
  both times; the live index (mtime 19:26, before this run started) was not written. Every
  measurement below was taken on three `git archive` copies of that tree
  (`.temp/TASK-260830-35urbp/review-rev4/{candidate,mutant-copy,probe-copy}`), each hashed to
  `c978fd70` through a scratch index before use and after every battery pass. No product edit,
  commit, checkpoint or integration was performed; `git status` in the live worktree is unchanged
  (`M LOGBOOK.md`, `M README.md`, `?? internal/tmuxserver/`).
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §3.2 (806-813), §4.2 (1403-1449),
  §4.3 (1458), §4.4 (1478-1488), with §4.C/§4.D/§4.E as context. The task record's v0.5.0 Scope is
  stale; the candidate cites the pinned file only (no `0.5.0` anywhere in the package).
- Note on the rev4 reviewer brief: its templated line "revisions 1..3 carry no review verdict" is
  wrong for this task — rev1 (RUN-260921-0245e5), rev2 (RUN-260921-5a9d4b) and rev3
  (RUN-260921-2369f7) were all reviewed and routed CHANGES REQUESTED. Rev4 was reviewed on its
  own merits AND against closure of every rev3 finding.

## What was reproduced (all on tree c978fd70; PATH without any tmux binary, TMUX/TMUX_TMPDIR unset, no tmux process on the host; umask 022, uid 502, darwin/arm64, go1.25.5)

| Check | Result | Log (evidence archive) |
| --- | --- | --- |
| Configured suite, 30 commands read from `spawn.worktree_isolation.validation.commands` (count 30, verified by read) | **30/30 exit 0, every command rerun by this reviewer in one sequential pass** (`suite-exits.txt` carries every exit and duration): 1 gofmt, 2 build, 3 vet; 4 `go test ./... -count=1 -v` (167s): 45/45 ok, 0 package FAIL, 0 top-level `--- FAIL`, 14 pre-existing SKIP outside this package, 0 SKIP in `internal/tmuxserver`; the 21 indented `--- FAIL` lines are mutant-kill excerpts inside the passing `TestSmokeMutantsAreKilled` / `TestCensusLiveEventOwnershipPlants` (attributed per enclosing test in the log); 5 race in **one call** (396s, no bounded groups): 45/45 ok, 0 `DATA RACE`; 6 cover: 45/45 ok, tmuxserver **92.6%**; 7-23 seventeen fuzz seeds exit 0; 24 tracecheck exit 0 (`clauses_discharged=70/574`); 25 cataloggen -check exit 0; 26 linux build; 27 windows build; 28 JSON parse exit 0; 29 `task-board validate` exit 0 with the board's 180 pre-existing issues; 30 `git diff --check` exit 0 | `logs/cmd01.log` … `logs/cmd30.log`, `logs/suite-exits.txt` |
| Hosted-CI-shaped gates (test files included) | `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; `GOOS=windows go test -c ./internal/tmuxserver` exit 0; `GOOS=linux go vet ./internal/tmuxserver` exit 0; `GOOS=linux go test -c` exit 0; `gofmt -l` over every tracked `.go` empty | `logs/static-gates.log` |
| Package suite, tmux unresolvable | `go test ./internal/tmuxserver -count=1 -v`: 134 RUN, **70 top-level PASS, 64 subtest PASS, 0 FAIL, 0 SKIP**. Determinism holds; no real-tmux witness exists (bound B1, honest) | `logs/pkg-tests-notmux-v.log` |
| Producer harness, two passes on the mutant copy | **56/56 both passes** (54 narrowing + D-background-fallback KILLED exit 1, C-control SURVIVED exit 0); verdict lists identical to each other and to the producer's attached pass 1; every raw log carries its `exit:` line; mutant copy re-hashed to `c978fd70` after each pass; zero `__pycache__`/`.pyc` anywhere | `harness-pass1/`, `harness-pass2/`, `logs/harness-pass*-verdicts.log` |
| Producer evidence archive | real gzip; MANIFEST 137 entries, no self-entry, **137/137 sha256 verify**, no unlisted file; one raw log per plant per pass with `exit:`; producer pass1 == pass2; race groups 8/8/8/8/8/5 = 45 packages, 0 `DATA RACE` | (checked in place) |
| Reviewer battery, 15 rows × 2 passes | identical verdicts both passes; see §M | `reviewer-battery-pass1/`, `reviewer-battery-pass2/`, `reviewer_battery.py` |
| Reviewer probes, 12 tests | all behave as described in §P | `logs/probes-rev4.log`, `zz_review_probe_test.go` |
| Boundary / hygiene | `internal/traceability` untouched (`git diff base tree -- internal/traceability` empty); no `0.5.0` citation; no `__pycache__`/`.pyc` in the CR tree or any copy; no `os/exec`/`Getenv`/`Environ` in any production file (one comment mention in socket.go); no `localstore` import (doc references only — see P2-B); README section carries no `ax` command, `doctor` or capability claim; every one of the 68 test names in TRACEABILITY.md exists in the committed suite; row count 58 verified (1..58, no gaps, no duplicates) | `logs/static-gates.log` |

## Closure of the rev3 findings (verified by driving, not by reading the results table)

| rev3 | Closed? | How verified |
| --- | --- | --- |
| P2-A derived path ≠ spec path | **yes, as briefed** | `RuntimeDirName = "tmux"`, `SocketName = "ax.sock"`; `TestSocketPathDerivesOnlyFromRuntimeDir` asserts both spec literals and the joined `<root>/tmux/ax.sock`; `Request.Root` documented as the Runtime IPC root (`localstore.PathRuntime`, which resolves to `<TemporaryDir>/ax` on macOS and `$XDG_RUNTIME_DIR/ax` on Linux/WSL2 — read at paths.go:329-353, matching §3.2); §3.2 row 56 added; probe P08 commits and verifies under the real `/var/folders/…/T` shape. The entry-level residue is P2-B below (`Request.Name`) |
| P2-B absent leaf → custody code | **yes** | `acquireBackground` maps the verify-side `runtime absent` detail to the typed `capability_unavailable` with no probe, no spawn, no directory (`TestAcquireBackgroundVerifiesWithoutCreating` rewritten, asserts all four; probe P06 confirms the broker is never probed even when it would admit); `N-verify-absence-detail` / `N-background-absence-remap` KILLED ×2 with the predicted signatures; reviewer additive R01 (create + spawn on the absence path) and R02 (probe-and-contact on absence) both KILLED by that test. The custody half of the split is P2-A below |
| P2-C running arm pinned at the entry | **yes** | `TestAcquireForegroundRefusesRunningWithEmptyAdmission` / `...WithEmptyGeneration` committed (spawn 0 asserted); `N-running-empty-admission` / `N-running-empty-generation` KILLED ×2 (`err = nil, want …` — a second `new-session` spawns under the mutant); reviewer R13 re-runs the rev3 R01 shape against the rev4 suite: KILLED |
| P3-A TMUX-env encoding | **yes** | `tmuxEnvSocket` splits at the first comma; `TestResolveSocketRefusesTMUXEnvRealEncoding` + `TestAcquireRefusesTMUXEnvCollisionRealEncoding` committed; `N-collision-tmuxenv-encoding` KILLED ×2 at both levels; reviewer R06 (split at the LAST comma) KILLED by both |
| P3-B absence vs read failure | **yes** | verify side: `ENOENT` → `runtime absent`, everything else → `runtime containment`; `classifyRootOpenError` shared by commit and verify; `TestVerifyRuntimeDirRefusesAbsence`, `TestVerifyRuntimeDirRefusesMissingRoot`, `TestEnsureRuntimeDirRefusesBadRoots/missing` all name `tmux_invalid_arguments at runtime root`; `N-root-missing-classify` KILLED ×2 on both entries; probe P02 shows a dangling symlink leaf is NOT absence (containment, broker not probed); probe P07 shows a 0000 root reports containment on both entries |
| P3-C root custody / "AX-created" | **yes (bounds)** | B19 names the `localstore` layout owner for root creation custody (verified: `InitializeLayout` → `ensureOwnerDirectory` creates 0700 and verifies owner + exact mode, paths.go:544-627) and the lifecycle bind step TASK-260830-1c28dz for the SPEC 810-812 before-bind rejection; B20 states provenance unclaimed; probe P03 restates the bound (0777 root admitted on Ensure, Verify and the broker path) |
| P3-D socket.go prose / vestigial return | **yes** | comment rewritten to what the code does; `ObserveAmbient` returns `Ambient` only; both observation tests updated |
| P3-E read-only root / whitespace override | **yes** | `TestEnsureRuntimeDirRefusesReadOnlyRoot` (`runtime commit`, B14) + `N-commit-eacces-as-exist` KILLED ×2 by reroute; whitespace-only subtests at both levels + `N-override-whitespace` KILLED ×2; reviewer R07 (the realistic `TrimSpace` spelling) KILLED by both |
| Obs-1..4 | **yes** | B21 (zero `BrokerReport`, generation-only socket binding), B15 gains the unreachable guard arm, `AfterMkdir` comment says created-or-present |

## P2 — must fix or record as a reviewed decision

### P2-A. The classification gate this revision introduced has no narrowing that admits a member of the class it must reject; the custody-code oracle looks through the `capability_unavailable` wrapper

`runtimeAbsent` (acquire.go:246-252) is the new gate that splits the rev3 P2-B rule: only the
verify-side `runtime absent` detail is remapped to the typed `capability_unavailable`; every
unsafe-leaf refusal must keep its custody code (SPEC 810-812; TRACEABILITY row 52 "an unsafe leaf
keeps its custody code"; README "an existing-but-unsafe directory keeps its custody refusal"). The
shipped row `N-background-absence-remap` weakens the gate to REJECT a member it must accept (the
canonical-name absence) — that pins the absence direction. The DoD-shaped narrowing — weaken the
gate to ADMIT one member it must reject — is not shipped, and it survives:

- Reviewer row `R05-runtimeAbsent-admits-mode`: `local.Detail == "runtime absent"` →
  `(local.Detail == "runtime absent" || local.Detail == "runtime mode")`. A widened 0755 leaf on
  the background path then returns `capability_unavailable` (exit 6) with the custody `*Error` as
  its hidden cause. **SURVIVED the full committed suite in both passes** (`go test
  ./internal/tmuxserver -count=1`, exit 0).
- Why: `requireLocalError` (tmuxserver_test.go:79-91) uses `errors.As`, and `axerror.Error.Unwrap`
  (axerror.go:291) returns the cause, so `backgroundUnavailable(req, verr)` wrapping a custody
  `*Error` satisfies `requireLocalError(t, err, "tmux_unsafe_runtime_dir", "runtime mode")`. The
  three tests that are supposed to pin the custody half — `TestAcquireBackgroundStagesVerifyCustodyRefusal`,
  `TestAcquireBackgroundRefusesNonDirectoryLeaf`, `TestAcquireBackgroundMissingRootRefusesInvalid`
  — all assert through that helper, so none observes the top-level code the caller renders (the
  cause is redacted off the wire by design, so the observable refusal IS the wrapper).
- Probe K05 (assert the top-level dynamic type is `*tmuxserver.Error` and that no `*axerror.Error`
  is reachable) passes on the candidate — production is correct today — and KILLS R05
  (`top-level err = *axerror.Error (capability_unavailable: …), want *tmuxserver.Error`,
  `logs/R05-runtimeAbsent-admits-mode.with-probe.log`).

DoD line "Every gate ships at least one NARROWING mutant — the gate stays present and is weakened
to admit exactly one member of the class it must reject, and a named test must fail" is not met
for `runtimeAbsent`, and row 52's custody half is prose by test. Fix: make `requireLocalError`
assert the top-level dynamic type (`err.(*Error)`, or additionally `!errors.As(err, &axerr)`)
so every custody assertion pins what the caller sees; ship `N-absence-remap-widen` in the R05
shape; keep `N-background-absence-remap` for the other direction.

### P2-B. `Request.Name` is a caller-selectable leaf that re-opens the §3.2 path divergence per call; the spec path is pinned at the constants, not at the entry

SPEC §3.2 806-807: "MUST use a dedicated AX server selected by `tmux -S <runtime>/tmux/ax.sock`".
Rev3 P2-A was closed at the constants and `TestSocketPathDerivesOnlyFromRuntimeDir` pins them.
But through the production entry `Acquire`, the derived socket is `<Root>/<Name>/ax.sock`, and
`Name` is a free grammar-checked segment "(usually RuntimeDirName)" (acquire.go:90-91):

- Probe P01: `Acquire` with `Name` = `ax-tmux`, `default`, `tmux-502` returns `via=spawned`,
  spawns once, at `<root>/ax-tmux/ax.sock`, `<root>/default/ax.sock`, `<root>/tmux-502/ax.sock`
  — no gate refuses, and `default` is the very conventional name row 22 refuses as an override.
- Reviewer row `R11-name-ignored-dead-input`: replacing `req.Name` with `RuntimeDirName` at the
  `Acquire` lexical step **SURVIVED** both passes — every committed `Acquire` test passes the
  constant, so the field's variability is unmeasured through the entry (rejection pattern b: rows
  16/56 are pinned at `SocketPath` and the constants, not at `Acquire`).

There is no production reason for the field to vary: the exported `EnsureRuntimeDir`/
`VerifyRuntimeDir` need a `name` parameter for the grammar rows (row 11), the acquisition entry
does not. A future caller passing `"ax-tmux"` re-creates the rev3 divergence and nothing reddens.
Fix (either): drop `Request.Name` and have `Acquire` pass `RuntimeDirName` (the constant the spec
fixes), or refuse `req.Name != RuntimeDirName` at `Acquire` with an entry test and a narrowing
row; either way add the entry-level assertion that `Acquire`'s outcome socket equals
`<Root>/tmux/ax.sock` for both callers and name it under rows 16/56.

## P3 — fix in the same rework

- **P3-A. The absence remap is pinned on the macOS axis only.** Reviewer row
  `R14-background-absence-only-macos` (`runtimeAbsent(verr) && req.Platform ==
  scalar.PlatformMacOS`) **SURVIVED** both passes: `validRequest` defaults to `PlatformMacOS` and
  no committed absence test uses Linux/WSL2, so a Linux background restore worker with an absent
  leaf (the same cold state — `$XDG_RUNTIME_DIR` is tmpfs) would get the custody code. Row 29
  pins the Linux BROKER miss, not the absence arm. Probe K14 (Linux background absence →
  `capability_unavailable`, spawn 0) passes on the candidate and kills R14. Add a Linux (and
  WSL2) subtest to `TestAcquireBackgroundVerifiesWithoutCreating` and name it under row 52.
- **P3-B. A censusable "every" that is false.** README ("every background test arms a spy
  spawner that fails the run if it is ever invoked") and acquire.go:135-137 (same sentence).
  Census of the 11 background-path `Acquire` tests: 7 arm `spawnSpy`, 4 do not —
  `TestAcquireBackgroundMissingRootRefusesInvalid`, `TestAcquireBackgroundRefusesNonDirectoryLeaf`,
  the background half of `TestAcquireRefusesMissingDependencies`, the background iteration of
  `TestAcquireRefusesWindowsPlatform`. With `Spawn` nil a guarded plant (`if req.Deps.Spawn != nil`,
  the exact D-background-fallback shape) is invisible to those four. Either arm the spy in all
  four or reword to "every test that reaches the broker-or-refuse path" (rejection pattern f).

## Observations (not findings; a sentence each in TRACEABILITY if the producer agrees)

- `R03-absence-widen-eloop` (ELOOP classified as absent on the verify side) SURVIVED ×2 and is a
  **darwin-equivalent mutant**, not a suite gap: on this host `openat(O_NOFOLLOW|O_DIRECTORY)` on
  a symlink leaf returns `ENOTDIR` (R04's ENOTDIR widening is killed by
  `TestVerifyRuntimeDirRefusesSymlinkLeaf`, and probe P02's dangling symlink reports containment),
  so the ELOOP arm is unreachable here. On Linux the same open returns `ELOOP`, which the
  production code maps to containment — reasoned, not run; the configured suite is darwin-only.
  Worth one sentence under row 6/13.
- `R12-classify-eacces-as-missing` SURVIVED ×2: an unopenable (0000) root reports
  `runtime root` instead of `runtime containment` under the mutant; no committed test stages a
  0000 root (probe P07 is the baseline). Refusal-to-refusal; report under B15 as an error-arm row.
- The `Fchmod`, `Fsync` and leaf `Stat` failure arms in `commitRuntimeDir`/`verifyRuntimeDir` are
  the same unstageable class as B15's listed arms and are not listed there.
- Probe P04: an `ax` invocation from inside its own AX pane (`TMUX=<derived>,<pid>,<idx>`, the
  encoding P3-A now decodes) refuses `ambient collision` when the caller populates `Ambient` from
  the process environment. Over-strict, not over-permissive, and the caller owns what it observes
  — but the lifecycle leaf should decide it deliberately (a sentence under row 23 or B16).
- Of the five collision members, only `TMUXEnv` and `InheritedSocket` can name the derived socket
  in real tmux semantics (`TMUX_TMPDIR` is a directory, `DefaultPath`/`ConventionalName` end in
  `default`, never `ax.sock`); rows for the other three pin a shape tmux never produces. Harmless
  defence in depth; worth stating so row 23 does not read as five live vectors.
- Probe P10: on the broker path the outcome carries the derived socket while nothing checks the
  socket exists; the broker's word alone admits (B21 already says the binding is generation-only).
- Row 56's "resolved by the landed localstore layout as PathRuntime and supplied by the caller" is
  a contract statement about the caller, not a driven fact (no test composes `localstore` with
  `Acquire`); acceptable, but state it as such or under B19.
- The rev4 producer harness runs each row under a `-run` mask; the reviewer battery runs the full
  package suite as shipped. Every producer KILLED row reproduced under the mask; nothing here
  depends on the difference.

## §M — reviewer battery (15 rows, two passes, `reviewer_battery.py` in the evidence archive)

Every row: anchor count verified (all 15 applied, none NOT_APPLIED, none COMPILE_FAIL), plant
applied to the mutant copy, the committed suite run as shipped (full package, no `-run` mask),
exit + raw output logged, file restored from a byte copy and re-compared, package content hash
equal before/after (`e501498c8473843f`), tree re-hashed to `c978fd70` after each pass. Rows
predicted SURVIVED were additionally run with `zz_review_probe_test.go` present and the named
killer selected (`logs/*.with-probe.log`).

| Row | Kind | Gate (site) | Committed suite | With probe | Killer |
| --- | --- | --- | --- | --- | --- |
| R01-absence-additive-create-and-spawn | additive | create + spawn on the absence path (acquire.go:179-181) | KILLED | — | TestAcquireBackgroundVerifiesWithoutCreating |
| R02-absence-probe-and-contact | narrowing | absence path trusts a live broker (acquire.go:179-181) | KILLED | — | TestAcquireBackgroundVerifiesWithoutCreating |
| R03-absence-widen-eloop | narrowing | verify `ENOENT` arm gains `ELOOP` (runtime_unix.go:120) | **SURVIVED** | (darwin-equivalent: ELOOP unreachable, see observations) | — |
| R04-absence-widen-enotdir | narrowing | verify `ENOENT` arm gains `ENOTDIR` (runtime_unix.go:120) | KILLED | — | TestVerifyRuntimeDirRefusesNonDirectoryLeaf, TestAcquireBackgroundRefusesNonDirectoryLeaf, TestVerifyRuntimeDirRefusesSymlinkLeaf |
| R05-runtimeAbsent-admits-mode | narrowing | `runtimeAbsent` admits `runtime mode` (acquire.go:251) | **SURVIVED** | KILLED | K05 (P2-A) |
| R06-tmuxenv-split-last-comma | narrowing | `IndexByte` → `LastIndexByte` (socket.go:126) | KILLED | — | TestResolveSocketRefusesTMUXEnvRealEncoding, TestAcquireRefusesTMUXEnvCollisionRealEncoding |
| R07-override-trimspace | narrowing | `override != ""` → `TrimSpace(override) != ""` (socket.go:85) | KILLED | — | TestAcquireRefusesEveryAmbientOverrideVector, TestResolveSocketRefusesEveryOverrideVector |
| R08-verify-nofollow-drop | narrowing | drop `O_NOFOLLOW` on the verify Openat (runtime_unix.go:118) | KILLED | — | TestVerifyRuntimeDirRefusesSymlinkLeaf |
| R09-commit-nofollow-drop | narrowing | drop `O_NOFOLLOW` on the commit Openat (runtime_unix.go:59) | KILLED | — | TestEnsureRuntimeDirRefusesSymlinkLeaf |
| R10-broker-uid-selfcompare | narrowing | same-user arm compares the principal against itself (acquire.go:189) | KILLED | — | TestAcquireBackgroundMissReturnsTypedUnavailable |
| R11-name-ignored-dead-input | dead-input witness | `req.Name` → `RuntimeDirName` at the Acquire lexical step (acquire.go:150) | **SURVIVED** | (no killer exists: the field's only tested value is the constant) | — (P2-B) |
| R12-classify-eacces-as-missing | error-arm | `classifyRootOpenError` gains `EACCES` (runtime_unix.go:29) | **SURVIVED** | (refusal-to-refusal; P07 baseline) | — |
| R13-running-empty-admission-entry | narrowing | rev3 R01 shape re-run (acquire.go:224) | KILLED | — | TestAcquireForegroundRefusesRunningWithEmptyAdmission |
| R14-background-absence-only-macos | narrowing | absence remap limited to macOS (acquire.go:179) | **SURVIVED** | KILLED | K14 (P3-A) |
| C-reviewer-control | control | comment-only edit in readiness.go | SURVIVED | — | (must survive; proves the battery observes outcomes) |

Denominator: 15 applied rows = 11 narrowing + 1 additive + 1 error-arm + 1 dead-input witness +
1 control. Of the 11 narrowings, 8 KILLED by the committed suite, 3 SURVIVED: R05 and R14 are
proven real by a probe kill (production correct, suite blind); R03 is darwin-equivalent. Every
KILLED row was killed by exactly the tests named above (kill lines in the raw logs).

## §P — reviewer probes (12, `logs/probes-rev4.log`)

P01 free `Name` spawns at `<root>/<Name>/ax.sock` for `ax-tmux`/`default`/`tmux-502` (P2-B);
P02 dangling symlink leaf → `runtime containment`, broker not probed (not absence); P03 0777 root
admitted on Ensure/Verify/broker path (B19); P04 nested invocation (`TMUX=<derived>,4242,0`)
refuses `ambient collision`, spawn 0; P05 trailing-slash, `/./` and `//` root spellings all refuse
`runtime root` at the lexical gate; P06 absent leaf + live broker → `runtime absent` mapped to
unavailable, broker never probed; P07 0000 root → `runtime containment` on both entries; P08 real
`/var/folders/…/T` root shape commits and verifies (the `/var` symlink is an intermediate, B11);
P09 setgid+sticky on a 0700 leaf verifies clean (B17); P10 broker path admits with no socket file
present and `ProbeServer` never called (B21); K05 top-level custody `*Error` on a widened leaf
through the background entry (baseline for R05); K14 Linux background absence →
`capability_unavailable` (baseline for R14).

## What held under attack (keep)

- Dedicated-server rule per vector: override refused on all five members plus whitespace through
  `Acquire` with no spawn and no directory; byte-identical, cleaned and real-encoding collisions
  refused; argv gate refuses all five; `Acquire` never reads the environment or execs; root
  spellings that could alias the path refuse at the lexical gate (P05).
- Background rules: broker-or-refuse with the literal `capability_unavailable`, exit 6 and the
  five typed details asserted as literals; the absent-leaf cold state now returns the typed
  refusal with no probe, no spawn and no directory (R01/R02 KILLED); no fallback along the caller
  (N-creation-literal), platform (Linux miss), spawner (D-background-fallback) and directory-write
  axes; same-user generation-bound broker principal at the entry (R10 KILLED).
- Attestation: landed `terminalbackend.Admitted` consumed through `Has` composed with the
  generation equality; the running arm is pinned at the entry for empty admissions (R13 KILLED);
  stale, decoy and both-empty shapes refuse.
- Custody: FIFO and regular-file leaves refuse on both entries without blocking (R04 KILLED);
  symlink leaf refuses on both entries (R08/R09 KILLED); absence is its own detail and a dangling
  symlink is not absence (P02); missing root aligned across entries; widened/narrower modes,
  foreign UID, handle mismatch, read-only root refuse; the explicit `Fchmod` is load-bearing under
  umask 0177; the real-SIGKILL crash test converges.
- Determinism: 0 SKIP with tmux unresolvable; no real-tmux witness (B1).
- Boundary: `internal/traceability` untouched; README section carries no CLI/capability claim; no
  trunk drift; producer evidence archive complete and digest-verified; LOGBOOK newest-first with
  the rev1 block superseded.

## Coverage ratio measured against the 58-row table

**58 of 58 rows are driven as their statements are written**; the ratio is not inflated. The gaps
above sit beside the statements: row 52's "an unsafe leaf keeps its custody code" is measured in
the absence→custody direction only (P2-A) and on the macOS axis only (P3-A); rows 16/56 pin the
spec path at the constants while `Acquire` derives `<Root>/<Name>/ax.sock` (P2-B); the README
"every background test" sentence outruns 4 of 11 tests (P3-B).

## Rework scope (for the producer)

1. **P2-A**: tighten `requireLocalError` to the top-level dynamic type (`err.(*Error)`; or add
   `!errors.As(err, &*axerror.Error)`) so the three background custody tests pin the code the
   caller renders; ship `N-absence-remap-widen` (`runtimeAbsent` admits `runtime mode`) and keep
   `N-background-absence-remap`; name both under row 52.
2. **P2-B**: remove `Request.Name` from the acquisition surface (`Acquire` passes
   `RuntimeDirName`; `EnsureRuntimeDir`/`VerifyRuntimeDir` keep `name` for the grammar rows) or
   refuse `req.Name != RuntimeDirName` at `Acquire` with an entry test and a narrowing row; add an
   entry-level assertion that the outcome socket equals `<Root>/tmux/ax.sock` on both callers;
   update rows 16/56, `doc.go` and README ("Name is the runtime leaf (usually RuntimeDirName)"
   must go).
3. **P3-A**: Linux and WSL2 subtests for the background absence remap under row 52 (K14 is the
   template).
4. **P3-B**: arm `spawnSpy` in the four background tests that lack it, or reword README and
   acquire.go:135-137.
5. Optional sentences: R03 darwin-equivalence under row 6/13; the unopenable-root arm and the
   `Fchmod`/`Fsync`/leaf-`Stat` arms under B15; the nested-invocation refusal and the three
   vacuous collision members under row 23/B16; row 56's caller-contract half under B19.
6. Rerun the package suite (tmux hidden), `GOOS=windows GOARCH=amd64 go vet ./...`, the harness
   twice, and the 30-command suite; refresh results/matrix/archive; hand off as revision 5.
   Everything in "What held under attack" needs no change — do not re-litigate it.
