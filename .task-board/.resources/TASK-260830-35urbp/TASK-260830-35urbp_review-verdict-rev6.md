# TASK-260830-35urbp — review verdict, Change Request revision 6

**Verdict: CHANGES REQUESTED** (routed `to-dev`). No P1. One P2 (a gate arm on one of its two
entries with no narrowing row, an unmeasured admit direction and an unmeasured no-hang property —
found by planting exactly the FIFO the brief named), two P3 (a platform member and a vocabulary
member each pinned one-of-N), plus observations. Every rev5 finding (P2-A four members, P3-A,
P3-B, observations 1-6) is closed with driven evidence and stays closed. **Production behaves
correctly at every point this review drove** — every surviving plant below is killed by a probe
that passes on the untouched candidate — so the rework is tests, rows and three doc sentences,
with no production change. The gate × entry census the rev6 brief asked for exists, its counts
reproduce mechanically (24 × 11 = 264; 50 M / 4 B / 210 U), and its per-entry kill attribution
holds for all 50 M cells; the P2 is the one arm the census did not row.

- Reviewer run: RUN-260921-a7163e (claude-opus-5 max), reviewing `CR-TASK-260830-35urbp-6`
  revision 6 published by RUN-260921-a765d6 (muse-spark max) after the rev5 rework brief.
- Reviewed bytes: base `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` after a fresh
  `git fetch origin main`; `git diff --name-only 799c338 origin/main` empty — no trunk drift, so
  no stale-copy audit applies), candidate tree `112590680342722341d49713d53744759dd69874`, patch
  resource sha256 `c4f4cab59e4704e582028db13928b12cf1b57cbc6709df82148a6d2530b6685c` (recomputed
  over the materialized resource, equal to the CR record). The live Story worktree's uncommitted
  tree was recomputed through a temporary index file (`GIT_INDEX_FILE`, live index untouched) and
  equals the CR tree (`11259068`); `internal/` in the live worktree is byte-identical to the
  archive copy (`diff -rq`). Every measurement below was taken on three `git archive` copies of
  that tree (`.temp/TASK-260830-35urbp/review-rev6/{candidate,mutant-copy,probe-copy}`; the
  candidate copy re-hashes to tree `11259068` through its own scratch repository; the two lean
  copies are re-diffed against it after every battery pass). No product edit, commit, checkpoint
  or integration was performed; the live worktree, index, branch and HEAD are unchanged.
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §3.2 (806-813), §4.2
  (1403-1449), §4.3 (1458), §4.4 (1478-1490), with §4.C/§4.D/§4.E as context. Every line citation
  in the package (`SPEC 806-807`, `806-808`, `810-812`, `1458`) matches the pinned file; no
  `0.5.0` citation exists in the package; the task record's v0.5.0 Scope is stale. The derived
  path is `<Root>/tmux/ax.sock` with `Request.Root` documented and consumed as the Runtime IPC
  root (localstore `PathRuntime` = `<TMPDIR>/ax` on macOS, `$XDG_RUNTIME_DIR/ax` on Linux/WSL2),
  never the durable state root; no committed test pins an old literal.
- Note on the rev6 reviewer brief: its templated line "revisions 1..5 … carry no review verdict
  and no prior review findings" is wrong for this task — rev1..rev5 were each reviewed and routed
  CHANGES REQUESTED (verdicts attached). Rev6 was reviewed on its own merits AND against closure
  of every rev5 finding.

## What was reproduced (all on tree 11259068; PATH = `{go,gofmt,python3 symlinks}:/usr/bin:/bin:/usr/sbin:/sbin` with no tmux binary resolvable, `TMUX`/`TMUX_TMPDIR` unset, `pgrep tmux` empty; umask 022, uid 502, darwin/arm64, go1.25.5, host load 8-14 throughout)

| Check | Result | Log (evidence archive) |
| --- | --- | --- |
| Configured suite, 30 commands read from `spawn.worktree_isolation.validation.commands` of the candidate config (count 30, verified by read) | **30/30 exit 0, every command rerun by this reviewer** (`suite/suite-exits.txt`, start/end tree `11259068`, `git status` clean after cmd 25 `-check`): 1 gofmt, 2 build, 3 vet; 4 `go test ./... -count=1 -v` (163s): 45/45 ok, 0 package FAIL, 0 top-level `--- FAIL`, 14 pre-existing nested `--- SKIP` all outside this package, **0 SKIP in `internal/tmuxserver`**; the 21 indented `--- FAIL` lines are mutant-kill excerpts inside the passing `TestSmokeMutantsAreKilled` (11) and `TestCensusLiveEventOwnershipPlants` (10); 5 race gate **run in 3 bounded groups of 15 packages with the identical flags** (97s + 60s + 517s = 674s; one call would have exceeded the 10-minute shell bound at this host load): 45/45 ok, 0 `DATA RACE`; 6 cover (254s): 45/45 ok, tmuxserver **92.7%**; 7-23 seventeen fuzz seeds exit 0; 24 tracecheck exit 0 (`clauses_discharged=70/574`); 25 cataloggen -check exit 0; 26 linux build; 27 windows build; 28 JSON parse exit 0; 29 `task-board validate` exit 0 as configured (180 pre-existing issues against the authoritative board; a second read-only run with `--board-dir` on the copy's committed snapshot also exit 0, 672 issues — neither is this leaf's scope); 30 `git diff --check` exit 0 | `suite/cmd01.log` … `suite/cmd30.log`, `suite/cmd05-race-grp-{aa,ab,ac}.log`, `suite/suite-exits.txt` |
| Hosted-CI-shaped gates (test files included) | `gofmt -l` over every tracked `.go` empty; `go vet ./internal/tmuxserver` exit 0; `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; `GOOS=windows GOARCH=amd64 go test -c ./internal/tmuxserver` exit 0; `GOOS=linux go vet` + `go test -c` exit 0 | `logs/static-gates.log` |
| Package suite, tmux unresolvable | `go test ./internal/tmuxserver -count=1 -v -coverprofile`: **190 RUN, 79 top-level PASS, 111 nested PASS, 0 FAIL, 0 SKIP**, 92.7% statements; env proof in `logs/pkg-tests-notmux-env.txt`. Determinism holds; no real-tmux witness exists (bound B1, honest; no row degrades into a mock-only shape — the production entries `Acquire`/`EnsureRuntimeDir`/`VerifyRuntimeDir`/`ResolveSocket`/`BuildArgv`/`CheckBrokerContact`/`CheckServerAttested` are what every test drives; only the exec/probe adapters are injected, and those are the lifecycle leaf's by B9). The 12 uncovered blocks are exactly the B15 error-arm class (handleBoundToPath ×2, verifyOwnership `!ok`, Fchmod, Fsync, commit- and verify-side leaf Stat, NewGuard, guard.Resolve), the `AfterMkdir` hook (exercised in the crash child, outside the coverprofile), the under-mutant-only `background fallback` arm (acquire.go:207), and the `BuildArgv` error arm through `acquireForeground` (acquire.go:243-245), which is B12 and holds: the root is `scalar.ParseAbsolutePath`-valid UTF-8 with no NUL, the leaf is `tmux`, the session is pre-parsed, so `secprim.CheckArgv` cannot refuse through `Acquire` | `logs/pkg-tests-notmux-v.log`, `logs/tmuxserver-cover.out` |
| Producer harness, two passes on the mutant copy | **75/75 both passes** (72 narrowing + `D-background-fallback` + `D-background-fallback-foreground` KILLED exit 1, `C-control` SURVIVED exit 0); verdict lists identical to each other and, modulo the `[exit N]` suffix, to the producer's attached pass 1 and pass 2; every raw log carries its `exit:` line; package files byte-identical to the candidate after each pass; zero `__pycache__`/`.pyc` anywhere. Kill lines spot-checked against the row notes (`N-broker-zero-admission`: `zero_server_admission` subtests in BOTH tables; `N-absence-remap-ownership`: `err top-level type = *axerror.Error … want *tmuxserver.Error`; `N-session-entry`: `server was probed before a malformed session refused`; `D-background-fallback-foreground`: every miss subtest `err = nil, want capability_unavailable`; `N-wiring-ensure`: foreground leg `tmux_ambient_server_reuse at spawn socket`; `N-wiring-verify`: background leg `capability_unavailable`; `N-argv-detached`: element-by-element vector mismatch at BOTH entries). One row note is imprecise: `N-details-constructor` kills with `err top-level type = *axerror.Error (<nil>)` — a typed-nil `*axerror.Error` inside a non-nil error interface, not the "zero outcome with nil error" the note describes; killed either way (observation 5) | `harness-pass1/`, `harness-pass2/`, `logs/harness-pass*-verdicts.log` |
| Producer evidence archive | real gzip; MANIFEST 193 entries, no self-entry, **193/193 sha256 verify**, no unlisted file; one raw log per plant per pass with `exit:` (75+75); pass1 == pass2 == reviewer rerun; `static-gates.log` now records every exit (rev5 obs-6 closed); `spy-census.log` 23/23; `pkg-tests-notmux-v.log` 190 RUN / 0 SKIP with the env proof; race groups 8/8/8/8/8/5 = 45, 0 `DATA RACE` | (checked in place) |
| Gate × entry census, verified mechanically | Table parsed from `TRACEABILITY.md`: 24 gates × 11 entries = **264 cells; 50 M, 4 B, 77 U1 + 133 U2 = 210 U** — the stated 50/4/210 reproduces. For all 50 M cells the named narrowing rows exist in the harness and their pass-1 raw logs redden the named entry-level test (per-entry attribution verified from `--- FAIL` lines, incl. the "same rows" cells: `N-details-*` redden the background subtest, `N-platform-wiring/-windows` redden ENS+VER+AFG+ABG, `N-root-wiring`/`N-root-missing-classify` redden ENS+VER+AFG/ABG). Every one of the 78 test names in TRACEABILITY/matrix exists in the committed suite (79 exist; `TestBuildArgvRefusesInvalidArgvShape` is the one not named — observation 3); every named row exists in the harness; row numbers 1..58, no gaps. The one committed test that is not a top-level `Test` name census: mechanical spy census reproduced **23/23** background-path `Acquire` tests arm both `spawnSpy` and `probeServerSpy`; effect census (a guarded `ProbeServer` call planted at the top of `acquireBackground`) reddens **16 top-level tests** — every background test that enters the function (the remaining 7 refuse pre-dispatch, correctly) | `battery/pass1/R28-effect-census-probe-at-background-top*.log` |
| Reviewer battery, 17 rows × 2 passes + 4 extra rows | identical verdicts both passes; see §M | `battery/pass1/`, `battery/pass2/`, `battery/pass1-probe/`, `battery/reviewer_battery.py` |
| Reviewer probes, 7 tests | all pass on the untouched candidate; K01/K01b/K01c/K07/K16 kill their rows; P11 and P-FIFO witness bounds/rows as stated; see §P | `logs/probes-rev6-baseline.log`, `battery/zz_review_probe_test.go` |
| Boundary / hygiene | `internal/traceability` untouched (`git diff base tree -- internal/traceability` empty); no `0.5.0` citation; no `__pycache__`/`.pyc` in the CR tree or any copy; no `os/exec`/`Getenv`/`Environ`/`localstore` import in any production file (comment mentions only); production imports are exactly `errors`, `io/fs`, `os`, `path/filepath`, `strings`, `syscall`, `x/sys/unix`, `axerror`, `axpane`, `scalar`, `secprim`, `terminalbackend` — no `utf8`/`json`/`sort`/`regexp`/`strconv`/`time`/`crypto`, i.e. no re-implementation of a landed owner; `checkRuntimeName`'s slash rule is genuinely package-specific (`scalar.ParseRelativePath` admits `a/b`); `handleBoundToPath` is the disclosed seam-mirror of unexported `secprim.guardRootMatches`; README section carries no `ax` command, `doctor` or capability claim; LOGBOOK newest-first with the rev6 block; crash evidence real (`TestEnsureCrashChildSelfTerminates` SIGKILLs the child inside `AfterMkdir` and proves retry convergence) and idempotency evidence present (`TestEnsureRuntimeDirIsIdempotent`, `TestAcquireCreatesRuntimeDirIdempotently`, `TestAcquireForegroundSpawnIsIdempotentAcrossRetry`); the server socket's own crash evidence is B9 (lifecycle leaf) | `logs/static-gates.log` |

## Closure of the rev5 findings (verified by driving, not by reading the results table)

| rev5 | Closed? | How verified |
| --- | --- | --- |
| P2-A four members: zero admission (R09), admitted-without-generation (R10), decoy-only (R19), fallback via `acquireForeground` guarded on `ProbeServer` (R18) | **yes** | R09 shape shipped as `N-broker-zero-admission`, KILLED ×2 by the `zero_server_admission` subtests in both `TestAcquireBackgroundMissReturnsTypedUnavailable` and `TestCheckBrokerContactRefusesEachMissingFact`; the R10 and R19 shapes re-planted by this reviewer as R02/R03 against the FULL suite → KILLED ×2 by `admitted_server_without_generation` / `decoy-only_server_admission` in both tables (the producer's anchor-share replay claim reproduces); R18 shipped as `D-background-fallback-foreground`, KILLED ×2 by every miss subtest plus the Linux miss; `ProbeServer` spy armed in 23/23 (mechanical, reproduced) and the effect census reddens all 16 tests that enter the function; row 31 and README now say both foreground dependencies |
| P3-A ownership member of the absence class | **yes** | `TestAcquireBackgroundStagesOwnershipRefusal` (unix file, `effectiveUID` seam) exists and passes; `N-absence-remap-ownership` KILLED ×2 with the top-level-type kill line; sibling rows `N-absence-remap-containment`/`-invalid` KILLED ×2 |
| P3-B false "before any side effect" | **yes** | `acquireForeground` parses the session before `EnsureRuntimeDir` and the probe; `TestAcquireForegroundRefusesMalformedSession` asserts no probe, no spawn, no directory; `N-session-entry` KILLED ×2 by `server was probed before a malformed session refused`; the acquire.go:142-147 sentence now enumerates the members that are true and I could not find a member it overstates (root grammar, override, collision, details, dependencies, session all refuse before the directory on the foreground path; reviewer rows R11/R12 confirm the verify-before-probe and ensure-before-probe orderings are pinned) |
| Obs 1-6 | **yes** | row 23 comma-root text; B16 "symlinked or relative alias"; B9 `Dependencies.Spawn` inert; B15 names the commit-side leaf-Stat and NewGuard arms; `TestAcquireBackgroundInvalidGenerationRefusesAfterProbe` + `N-details-constructor` KILLED ×2; `static-gates.log` carries exits |

## P2 — must fix or record as a reviewed decision

### P2-A. The commit-side non-directory-leaf gate (`O_DIRECTORY` on the commit `Openat`, runtime_unix.go:58-59) has no narrowing row, its admit direction is unmeasured, and its no-hang property is unmeasured — the census rowed the verify side only

SPEC §3.2 810-812: before bind AX MUST reject a socket path, parent or ancestor "whose file kind
… [is] unsafe"; row 5 claims "Non-directory leaf refuses on commit and verify … FIFO refusal
relies on O_DIRECTORY failing fast, pinned by the deadline test". The candidate implements the
gate correctly on both sides (same flags on both `Openat` calls), and production refuses a 0700
regular file and a FIFO on the commit side without blocking (probes K01/K01b/K01c pass on the
untouched candidate). What is not true is the measurement claim:

| Row | Plant (runtime_unix.go) | Committed suite | With probe | What the committed suite does not see |
| --- | --- | --- | --- | --- |
| R01 | drop `unix.O_DIRECTORY` from the COMMIT `Openat` (the exact twin of the producer's `N-containment-odirectory-verify`, which drops it from the verify `Openat`) | KILLED ×2 **by reroute only**: `TestEnsureRuntimeDirRefusesNonDirectoryLeaf` fails with `err = tmux_unsafe_runtime_dir at runtime mode, want … at runtime containment` — its fixture is a 0600 file, so the mode gate catches what the containment gate no longer does; `TestEnsureRuntimeDirRefusesSymlinkLeaf`, the crash test and every other commit test stay green | K01 kills (`EnsureRuntimeDir` on a 0700 regular file: `err = nil`); K01c kills (foreground `Acquire` on the same leaf: `err = nil` — the entry admits the leaf, probes the server and spawns into `<root>/tmux/ax.sock` where `tmux` is a regular file); K01b kills (`EnsureRuntimeDir` on a FIFO leaf **BLOCKED for 5s** — the writer-less `O_RDONLY` open stalls) | a 0700 non-directory leaf is ADMITTED on `EnsureRuntimeDir` and through the foreground entry; a FIFO leaf HANGS the creating path |

Why this is a P2 and not a P3: the verify side was given exactly this treatment in rev3 (the
`TestVerifyRuntimeDirRefusesNonDirectoryLeaf` comment says the regular file "carries 0700 so a
weakened open admits it outright instead of rerouting to the mode gate", and the FIFO subtest runs
"under a deadline so the no-hang property is pinned too"), and the shipped row's own note asserts
"the commit side still refuses" without a fixture that could tell. The rule is implemented twice
(commit and verify), rejection pattern (b) as stated in every brief of this task requires a
narrowing per side, and the harness has 3 verify-side containment rows (`-symlink-verify`,
`-odirectory-verify`, `N-root-symlink-verify`) against 2 commit-side (`-symlink`,
`N-root-symlink-commit`) — the O_DIRECTORY commit row is the missing one. The census cell
`commit (runtime_unix.go:35) × ENS` lists ten rows and none of them touches this arm, while the
`verify × VER` cell lists `N-containment-odirectory-verify` explicitly: the instrument that was
this round's deliverable shows the asymmetry and did not flag it. The DoD line "Every gate ships
at least one NARROWING mutant … and a named test must fail" is unmet for this arm (a test fails,
but for the mode gate's reason, and the DoD item is unchecked below for that reason). Production
is correct today; the fix is tests plus a row:

1. give the commit side the verify side's fixtures — a 0700 regular-file leaf on
   `EnsureRuntimeDir` and through foreground `Acquire` (K01/K01c are the templates; assert no
   probe and no spawn), and a FIFO leaf on `EnsureRuntimeDir` under the existing
   `verifyWithDeadline` (K01b);
2. ship `N-containment-odirectory-commit` (the R01 shape), name it under row 5 and in the
   `commit × ENS` and `commit × AFG` census cells, and reword the row-5 FIFO clause to name both
   deadline tests;
3. correct the `N-containment-odirectory-verify` note ("the commit side still refuses") to point
   at the new commit row rather than asserting it.

## P3 — fix in the same rework

- **P3-A. The no-fallback rule's platform axis is pinned at the miss arm for macOS and Linux only.**
  Reviewer row R16 (a fallback `Spawn` on the background MISS path guarded on
  `req.Platform == scalar.PlatformWSL2`) **SURVIVED ×2** the full suite: the miss tables drive
  macOS (8 shapes) and Linux (1 shape, row 29), and the only WSL2 background test
  (`TestAcquireBackgroundVerifiesWithoutCreating/wsl2`) refuses on the absent leaf before the miss
  arm, so no committed test reaches a WSL2 miss. The sibling narrowings are pinned — R15 (the
  same guard on Linux) KILLED ×2 by `TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback`,
  R17 (WSL2 background callers dispatched to the foreground path) KILLED ×2 by
  `…VerifiesWithoutCreating/wsl2`, R27 (absence remap withheld on WSL2) KILLED ×2 — so this is one
  member, not a class. Probe K16 (WSL2 miss with a live leaf, spy spawner + spy probe →
  `capability_unavailable`, spawn 0) passes on the candidate and kills R16 (`spawn calls = 1, want
  0 on a WSL2 miss`). This is the same shape as rev4 P3-A (absence remap pinned on macOS only),
  one arm over. Fix: make the Linux miss test a platform table `{linux, wsl2}` (or add a WSL2
  subtest to the miss table), add it to `BG_MISS_LINUX` so `N-creation-literal` and both `D-` rows
  kill through all three tmux platforms, and name it under rows 29/31.
- **P3-B. The attestation membership arm is pinned for one decoy of the sixteen-value §4.D
  vocabulary.** Reviewer row R07 (`CheckServerAttested` admits `Has("local_attach")` alongside
  the realm row) **SURVIVED ×2**: every committed decoy is `headless_creation`
  (`fixtureDecoyCapacity`), so a narrowing keyed on any of the other fourteen non-realm
  capabilities walks through the helper, the broker contact and both `Acquire` paths. Probe K07
  (`local_attach`-only admission at the helper and through the background miss →
  `server attestation` / `capability_unavailable`, spawn 0) passes on the candidate and kills R07.
  The independent source exists in the tree — `catalog.Current().Capabilities` is the exact
  sixteen-value Section 4.D table — so the completeness shape from `negative-evidence.md` applies:
  derive the decoy set from the catalog, assert every non-realm capability alone (and all of them
  together without the realm row) refuses at the helper and through the background entry, and let
  the existing `N-realm-membership` row keep its `headless_creation` member while the new test
  closes the other fourteen. Production is correct (`Has` is exact-match); this is measurement.

## Observations (not findings; a sentence each if the producer agrees)

1. Bound B9 witnessed as stated: R06 (a direct `os/exec` `tmux -S <socket> new-session -d` call
   planted on the background miss path, error discarded) **SURVIVED ×2** with tmux unresolvable —
   the no-fallback proof covers `Dependencies.Spawn` and the `acquireForeground` re-entry and
   excludes a direct exec only by the import census, exactly as B9 says. Honest; no change asked.
   (Do not run that plant on a host with tmux on PATH: it would create a real server in the test's
   temp root.)
2. `TestEnsureCrashChildSelfTerminates` flaked once in ~35 full-suite runs under host load 12-14
   (`child status = exit status 1, want killed by SIGKILL`, in the first R28 run; 25 isolated
   runs, 6 full-suite runs on the untouched candidate and 3 R28 reruns all PASS). The parent
   discards the child's stdout/stderr (`child.Run()` with neither set), so the exit-1 cause is
   undiagnosable from the log. Capturing the child's combined output into the failure message
   costs two lines and would make the next occurrence attributable.
3. `TestBuildArgvRefusesInvalidArgvShape` is the one committed top-level test not named in
   TRACEABILITY.md (B12 describes the arm without naming its witness); the conformance matrix
   lists row number 46 twice (two clauses under one number).
4. The `argv × AFG` census cell names the socket and session arms as U3 but not the
   `secprim.CheckArgv` shape arm, which through `Acquire` is B12 (unreachable: root is
   `ParseAbsolutePath`-valid, leaf fixed, session pre-parsed); worth naming B12 in the cell.
5. `N-details-constructor`'s note says the mutant "reports success with a zero outcome and no
   error"; the raw log shows it returns a typed-nil `*axerror.Error` inside a non-nil `error`
   (`err top-level type = *axerror.Error (<nil>)`). Killed either way; the note is wrong about the
   mechanism.
6. B11 witnessed as stated (P11): a symlinked INTERMEDIATE component of the runtime root is
   followed on both entries. This is the same bound as the landed `secprim.OpenNoFollowDir` root
   open and is design-necessary on macOS, where `localstore` resolves the Runtime IPC root under
   `$TMPDIR` (`/var/folders/…`, and `/var` is itself a symlink), so a strict no-follow walk from
   `/` could never admit the platform default. Not a finding.

## §M — reviewer battery (17 rows + 4 extra rows, two passes, `battery/reviewer_battery.py`)

Every row: anchor count verified (all applied, none NOT_APPLIED, none COMPILE_FAIL), plant applied
to the mutant copy from the pristine candidate bytes, the committed suite run as shipped (full
package, `-count=1 -v`, **no `-run` mask** — unlike the producer harness, which scopes each row),
exit + raw output logged with an `exit:` header, file restored from the pristine copy and
re-compared, package directory hash equal before/after (`e0314ddb4b26dfda`), `internal/` re-diffed
against the candidate copy after each pass. Rows predicted SURVIVED were additionally run with
`zz_review_probe_test.go` present and `-run TestReview…` (`battery/pass1-probe/`).

| Row | Kind | Gate (site) | Committed suite | With probe | Killer(s) |
| --- | --- | --- | --- | --- | --- |
| R01-commit-odirectory-drop | narrowing (flag) | commit-side non-directory leaf (runtime_unix.go:59) | KILLED **by reroute only** (`runtime mode` ≠ `runtime containment`) | KILLED | K01, K01b (5s BLOCK), K01c (P2-A) |
| R02-broker-admitted-without-generation | narrowing | broker-side server wiring (readiness.go:86), rev5 R10 shape | KILLED | — | `…MissReturnsTypedUnavailable/admitted_server_without_generation`, `…BrokerContactRefusesEachMissingFact/admitted_server_without_generation` |
| R03-broker-decoy-only-admits | narrowing | same site, rev5 R19 shape | KILLED | — | the two `decoy-only_server_admission` subtests |
| R04-unavailable-code-swap-same-exit | narrowing (token) | `backgroundUnavailable` emits `terminal_backend_capability_unproven` (exit 6, registered 1.3.0) with the same typed details (acquire.go:297) | KILLED | — | every miss subtest + Linux miss + `…VerifiesWithoutCreating` ×3: `code = "terminal_backend_capability_unproven", want "capability_unavailable"` |
| R05-detail-swap-generation | narrowing (detail) | `TmuxServerGeneration` carries `BrokerState` (acquire.go:294) | KILLED | — | every unavailable-asserting test: `detail "tmux_server_generation" = absent, want "generation-7"` |
| R06-exec-fallback-B9-witness | additive (bound) | direct `os/exec` spawn on the background miss (acquire.go:204) | **SURVIVED** | — | (stated bound B9; observation 1) |
| R07-membership-fresh-decoy-local-attach | narrowing | membership arm admits `local_attach` (readiness.go:40) | **SURVIVED** | KILLED | K07 (P3-B) |
| R08-verify-odirectory-drop-replay | replay | producer `N-containment-odirectory-verify` against the full suite | KILLED | — | `TestVerifyRuntimeDirRefusesNonDirectoryLeaf/{regular_file,fifo_without_blocking}` (5s BLOCK), `TestAcquireBackgroundRefusesNonDirectoryLeaf` |
| R11-background-probe-before-verify | order | broker probed before custody (acquire.go:183-193) | KILLED (8 top-level tests) | — | `…VerifiesWithoutCreating` ×3 ("broker was probed for an absent runtime directory"), every background custody test ("broker was probed past …") |
| R12-foreground-probe-before-ensure | order | server probed before ensure (acquire.go:228-235) | KILLED | — | `…StagesCommitPhaseCustodyRefusal`, `…RefusesBadRoot/missing` |
| R15-linux-only-fallback-spawn | additive (platform-narrowed) | fallback spawn on a Linux miss (acquire.go:204) | KILLED | — | `TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback` |
| R16-wsl2-only-fallback-spawn | additive (platform-narrowed) | fallback spawn on a WSL2 miss (acquire.go:204) | **SURVIVED** | KILLED | K16 (P3-A) |
| R17-dispatch-wsl2-background-to-foreground | narrowing | dispatch (acquire.go:163) routes WSL2 background callers foreground | KILLED | — | `…VerifiesWithoutCreating/wsl2` (directory created, spawn 1) |
| R22-collision-skipped-for-background | narrowing (caller axis) | collision gate bypassed for background at the entry (acquire.go:159) | KILLED | — | `TestAcquireRefusesAmbientCollision/*/background` ×5, unclean + TMUX-encoding background legs |
| R23-override-skipped-for-background | narrowing (caller axis) | override gate bypassed for background at the entry (acquire.go:159) | KILLED | — | `TestAcquireRefusesEveryAmbientOverrideVector/*/background` ×6 |
| C-reviewer-control | control | comment-only edit in acquire.go | SURVIVED | — | (must survive; proves the battery observes outcomes) |
| R25-realm-constant-swap-headless | narrowing (token) | `realmCapability` swapped to another registered §4.D value | KILLED | — | every unavailable test (`detail "capability"` literal), `…ForegroundRefusesRunningUnattested` (decoy attaches), `…BackgroundContactsBroker` |
| R26-principal-generation-admits-when-server-bound | narrowing | principal-generation arm waived for an attested server (readiness.go:83) | KILLED | — | `generation-unbound_principal` in both tables, `empty_principal_generation`, `both_generations_empty` |
| R27-absence-remap-withheld-on-wsl2 | narrowing | absence remap (acquire.go:184) skipped for WSL2 | KILLED | — | `…VerifiesWithoutCreating/wsl2` |
| R28-effect-census-probe-at-background-top | effect census | guarded `ProbeServer` call at the top of `acquireBackground` | KILLED (16 top-level tests; +1 crash-test flake in run 1 only, observation 2) | — | every background test that enters the function |

Denominator: 21 applied rows = 12 narrowing + 3 additive + 1 replay + 2 order + 1 effect census
+ 1 control (+ R28 counted once). Of the 12 narrowings, 10 KILLED by the committed suite, 2
SURVIVED (R07 outright; R01 killed only by reroute, admit direction and no-hang unmeasured) —
each proven real by a probe kill (production correct, suite blind). Of the 3 additives, R15 KILLED,
R16 SURVIVED (probe kill), R06 SURVIVED as the stated bound. Replay, both orderings and the effect
census KILLED. Every KILLED row was killed by exactly the tests named above (kill lines in the raw
logs). Passes 1 and 2 identical.

## §P — reviewer probes (7, `logs/probes-rev6-baseline.log`, all PASS on the untouched candidate)

K01 `EnsureRuntimeDir` on a 0700 regular-file leaf → `runtime containment` (baseline for R01);
K01b `EnsureRuntimeDir` on a FIFO leaf under a 5s deadline → `runtime containment` immediately;
K01c foreground `Acquire` on the 0700 regular-file leaf → `runtime containment`, no probe, no
spawn; K07 `local_attach`-only admission → `server attestation` at the helper and
`capability_unavailable` through the background miss, spawn 0 (baseline for R07); K16 WSL2
background miss with a present leaf → `capability_unavailable`, spawn 0, probe silent (baseline
for R16); P11 symlinked intermediate root component followed on Ensure and Verify (B11 witness);
P-FIFO FIFO leaf through the background entry → `runtime containment` fast, broker not probed.

## What held under attack (keep)

- Dedicated-server rule per vector: override refused on all five members plus whitespace on both
  callers through `Acquire`, and the caller axis of both the override and the collision gate is
  pinned at the entry (R22/R23 KILLED); the socket has exactly one constructor (`SocketPath`,
  called from `ResolveSocket` and `BuildArgv`), one resolution site (`Acquire` → `ResolveSocket`)
  and one spawn site; `Acquire` never reads the environment or execs; the §3.2 path is pinned at
  the entry on both callers and at all three `RuntimeDirName` sites; the argv vector is pinned
  element-by-element at the helper and through the entry.
- Background rules: broker-or-refuse with the literal `capability_unavailable`, exit 6 and the
  five typed details asserted as literals — a same-exit registered code swap (R04), a detail swap
  (R05) and a realm-constant swap (R25) all redden; the broker-side attested-server conjunct is now
  pinned for five admission shapes at the helper AND through the background entry (R02/R03
  KILLED); the principal arms are pinned (R26); the absent-leaf cold state returns the typed
  refusal on macOS, Linux and WSL2 (R27); the fallback is killed through `Spawn`, through
  `acquireForeground` re-entry, on Linux (R15) and at the WSL2 dispatch (R17); custody precedes
  the broker probe (R11) and ensure precedes the server probe (R12). Residue: P3-A.
- Attestation / cached observations: no input exists for a managername hint or a cached sentinel
  (`Request` and `RealmAdmission` have no such member); a replayed observation is a
  stale-generation admission and refuses on both paths; the generation-binding and membership
  arms are pinned at the helper, the broker contact and both entries (the producer's
  `N-realm-membership`/`N-realm-generation` kill through four tables). Residue: P3-B.
- Custody: mode (exact 0700, both sides, both directions), ownership (both sides, and the
  absence classifier's ownership member at the background entry), containment on the verify side
  (FIFO fast-fail, 0700 regular file, symlink leaf, symlink root, handle mismatch) and the
  commit-side symlink leaf/root and mismatch all pinned; the top-level custody oracle sees the
  wrapper; the real-SIGKILL crash test converges. Residue: P2-A (commit-side O_DIRECTORY arm).
- Determinism: 0 SKIP with tmux unresolvable, 190 RUN; no real-tmux witness (B1); no mock-only
  degradation.
- Boundary: `internal/traceability` untouched; README section carries no CLI/capability claim; no
  trunk drift; producer evidence archive complete and digest-verified; LOGBOOK newest-first.

## Coverage ratio measured against the 58-row table

**58 of 58 rows are driven as their statements are written**; the ratio is not inflated. The gaps
sit beside the statements, at the member level: row 5 (the commit-side non-directory clause is
driven for a 0600 file, i.e. by the mode gate, and its FIFO clause names one deadline test on one
side — P2-A), rows 29/31 (the miss arm and the no-fallback census cover two of the three tmux
platforms — P3-A), rows 35/27 (one decoy of fourteen — P3-B). Census: 50 of 264 cells measured
as claimed; the `commit × ENS/AFG` cells are the ones whose row lists omit a live arm.

## Rework scope (for the producer)

1. **P2-A (tests + row + three sentences, no production change)**: mirror the verify-side
   fixtures on the commit side — 0700 regular-file leaf on `EnsureRuntimeDir` and through
   foreground `Acquire` (assert no probe, no spawn), FIFO leaf on `EnsureRuntimeDir` under
   `verifyWithDeadline` (K01/K01b/K01c in the reviewer probe file are the templates); ship
   `N-containment-odirectory-commit` (drop `O_DIRECTORY` from the commit `Openat`), name it under
   row 5 and in the `commit × ENS` and `commit × AFG` census cells; reword row 5's FIFO clause to
   both deadline tests; fix the `N-containment-odirectory-verify` note.
2. **P3-A**: WSL2 member of the miss table (platform table on the Linux miss test or a WSL2
   subtest), added to `BG_MISS_LINUX` so `N-creation-literal` and both `D-` rows kill on all three
   platforms; name under rows 29/31.
3. **P3-B**: a catalog-derived decoy completeness test (every non-realm `catalog.Current().Capabilities`
   member alone, and all together, refuses at `CheckServerAttested` and through the background
   miss); name under rows 27/35.
4. Optional: capture the crash child's combined output in the failure message; name
   `TestBuildArgvRefusesInvalidArgvShape` under B12; de-duplicate matrix row 46; B12 in the
   `argv × AFG` cell; correct the `N-details-constructor` note.
5. Rerun the package suite (tmux hidden), `GOOS=windows GOARCH=amd64 go vet ./...`, the harness
   twice, and the 30-command suite; refresh results/matrix/archive; hand off as revision 7.
   Everything in "What held under attack" needs no change — do not re-litigate it.
