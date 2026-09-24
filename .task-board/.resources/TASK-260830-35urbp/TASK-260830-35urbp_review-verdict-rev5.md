# TASK-260830-35urbp — review verdict, Change Request revision 5

**Verdict: CHANGES REQUESTED** (routed `to-dev`). No P1. One P2 class (four measured members),
two P3. Every rev4 finding (P2-A oracle + admit-direction narrowing, P2-B `Request.Name`, P3-A
platform axis, P3-B spy census, the five observation sentences) is closed with driven evidence
and stays closed. Production behaves correctly at every point this review drove — every
surviving plant below is killed by a probe that passes on the untouched candidate. What remains
is the same shape the rev3 and rev4 rounds closed on the foreground and custody paths, now on
the background broker path: the "attested AX tmux server" conjunct and the no-fallback proof are
pinned at the entry only for the fixture shapes, and three realistic server-admission shapes plus
one realistic fallback guard walk through the committed suite. The rework is small and enumerated
at the end; nothing in "What held under attack" needs to change.

- Reviewer run: RUN-260921-60d15f (claude-opus-5 max), reviewing `CR-TASK-260830-35urbp-5`
  revision 5 published by RUN-260921-615903 (muse-spark max) after the rev4 rework brief.
- Reviewed bytes: base `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` after a fresh
  `git fetch origin main`; `git diff --name-only 799c338 origin/main` empty — no trunk drift),
  candidate tree `df686ae3d850b33c260910de7abb028e534c5994`, patch resource sha256
  `16bdce220ae542e0cc619dbe72269e851ef2054028e2e8b23c1a9eabcf4168c5` (recomputed over the
  materialized resource, equal to the CR record). The live Story worktree's uncommitted tree was
  recomputed through a temporary index file before and after this review and equals the CR tree
  both times (`df686ae3`); the live index was not written by any command of mine (only the initial
  `git status` stat-cache refresh could have touched it; content unchanged). Every measurement
  below was taken on three `git archive` copies of that tree
  (`.temp/TASK-260830-35urbp/review-rev5/{candidate,mutant-copy,probe-copy}`; the candidate copy
  re-hashes to tree `df686ae3` through its own scratch repository, the two lean copies are
  byte-identical to it under `internal/`, verified by `diff -r` after every battery pass). No
  product edit, commit, checkpoint or integration was performed; `git status` in the live worktree
  is unchanged (`M LOGBOOK.md`, `M README.md`, `?? internal/tmuxserver/`).
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §3.2 (806-813), §4.2
  (1403-1449), §4.3 (1458), §4.4 (1478-1488), with §4.C/§4.D/§4.E as context. Every line citation
  in the package (`SPEC 806-807`, `806-808`, `810-812`, `1458`) was re-read against the pinned file
  and matches; no `0.5.0` citation exists in the package. The task record's v0.5.0 Scope is stale.
- Note on the rev5 reviewer brief: its templated line "revisions 1..4 … carry no review verdict
  and no prior review findings" is wrong for this task — rev1 (RUN-260921-0245e5), rev2
  (RUN-260921-5a9d4b), rev3 (RUN-260921-2369f7) and rev4 (RUN-260921-30d48f) were all reviewed and
  routed CHANGES REQUESTED, and their verdicts are attached to the task. Rev5 was reviewed on its
  own merits AND against closure of every rev4 finding.

## What was reproduced (all on tree df686ae3; PATH without any tmux binary, `TMUX`/`TMUX_TMPDIR` unset, no tmux process on the host; umask 022, uid 502, darwin/arm64, go1.25.5)

| Check | Result | Log (evidence archive) |
| --- | --- | --- |
| Configured suite, 30 commands read from `spawn.worktree_isolation.validation.commands` of the candidate config (count 30, verified by read) | **30/30 exit 0, every command rerun by this reviewer in one sequential pass** (`logs/suite-exits.txt` carries every exit and duration, start/end tree `df686ae3`): 1 gofmt, 2 build, 3 vet; 4 `go test ./... -count=1 -v` (261s): 45/45 ok, 0 package FAIL, 0 top-level `--- FAIL`, 2 top-level and 12 nested pre-existing `--- SKIP` outside this package, **0 SKIP in `internal/tmuxserver`**; the 19 indented `--- FAIL` lines are mutant-kill excerpts inside the passing `TestSmokeMutantsAreKilled` (9) and `TestCensusLiveEventOwnershipPlants` (10), attributed per enclosing test; 5 race in **one call** (557s, no bounded groups): 45/45 ok, 0 `DATA RACE`; 6 cover (241s): 45/45 ok, tmuxserver **92.6%**; 7-23 seventeen fuzz seeds exit 0; 24 tracecheck exit 0 (`clauses_discharged=70/574`); 25 cataloggen -check exit 0; 26 linux build; 27 windows build; 28 JSON parse exit 0; 29 `task-board validate` exit 0 (613 pre-existing issues against the committed board snapshot in the copy — not this leaf's scope); 30 `git diff --check` exit 0 | `logs/cmd01.log` … `logs/cmd30.log`, `logs/suite-exits.txt` |
| Hosted-CI-shaped gates (test files included) | `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; `GOOS=windows GOARCH=amd64 go test -c ./internal/tmuxserver` exit 0; `GOOS=linux go vet ./internal/tmuxserver` exit 0; `GOOS=linux go test -c` exit 0; `gofmt -l` over every tracked `.go` empty; `go vet ./internal/tmuxserver` exit 0 | `logs/static-gates.log` |
| Package suite, tmux unresolvable | `go test ./internal/tmuxserver -count=1 -v -coverprofile`: 140 RUN, **71 top-level PASS, 69 subtest PASS, 0 FAIL, 0 SKIP**, 92.6% statements. Determinism holds; no real-tmux witness exists (bound B1, honest). The 12 uncovered blocks are the B15 error-arm class (two of them unnamed in B15's list), the `AfterMkdir` hook (exercised in the crash child, outside the coverprofile), the under-mutant-only `background fallback` arm (acquire.go:203), and one input-reachable arm not listed in B15 (observation 5) | `logs/pkg-tests-notmux-v.log`, `logs/tmuxserver-cover.out` |
| Producer harness, two passes on the mutant copy | **57/57 both passes** (55 narrowing + D-background-fallback KILLED exit 1, C-control SURVIVED exit 0); verdict lists identical to each other and to the producer's attached pass 1 and pass 2; every raw log carries its `exit:` line; package files byte-identical to the candidate after each pass; zero `__pycache__`/`.pyc` anywhere. Kill lines spot-checked against the row notes (`N-absence-remap-widen`: `err top-level type = *axerror.Error … want *tmuxserver.Error`; `N-background-absence-remap`: macos subtest; `D-background-fallback`: `spawn calls = 1, want 0` ×5; `N-containment-odirectory-verify`: `BLOCKED for 5s on a FIFO leaf`; `N-deps-foreground`: panic) | `harness-pass1/`, `harness-pass2/`, `logs/harness-pass*-verdicts.log` |
| Producer evidence archive | real gzip; MANIFEST 143 entries, no self-entry, **143/143 sha256 verify**, no unlisted file; one raw log per plant per pass with `exit:` (57+57); producer pass1 == pass2 == reviewer rerun; race groups 8/8/8/8/8/5 = 45 packages, 0 `DATA RACE`; `pkg-tests-notmux-v.log` 140 RUN / 0 SKIP; `R14-replay.log` and `oracle-replay-old-verdict.log` show what they claim. `static-gates.log`, `cmd26/27/30` are 0-byte files: consistent with green commands that print nothing, but the archive records no exit code for the static gates — the exit claims for those live only in the results prose (observation 6; rerun and green here) | (checked in place) |
| Old-oracle replay | With `requireLocalError` reverted to the rev4 `errors.As` body and the rev4 R05 mutant applied, the full package suite **passes (exit 0)**; with the committed top-level oracle the same mutant is KILLED (R03 below). The oracle fix is load-bearing, as the producer's replay claims | `logs/oracle-replay-old-R03.log` |
| Reviewer battery, 20 rows × 2 passes | identical verdicts both passes; see §M | `battery/pass1/`, `battery/pass2/`, `battery/reviewer_battery.py` |
| Reviewer probes, 11 tests | all behave as described in §P; K01-K05 pass on the untouched candidate and kill their rows | `logs/probes-rev5.log`, `battery/zz_review_probe_test.go` |
| Boundary / hygiene | `internal/traceability` untouched (`git diff base tree -- internal/traceability` empty); no `0.5.0` citation; no `__pycache__`/`.pyc` in the CR tree or any copy; no `os/exec`/`Getenv`/`Environ`/`localstore` import in any production file (comment mentions only); README section carries no `ax` command, `doctor` or capability claim; every one of the 69 test names in TRACEABILITY.md and in the conformance matrix exists in the committed suite; row count 58 verified (1..58, no gaps, no duplicates); every mutant named in TRACEABILITY/matrix exists in the harness (57 rows: 55 `N-`, `D-background-fallback`, `C-control`); LOGBOOK newest-first with the rev1 block superseded; `hosttrust.platformModeOK` (B17) and `localstore.InitializeLayout/ensureOwnerDirectory` 0700 (B19) precedents re-read and hold | `logs/static-gates.log` |

## Closure of the rev4 findings (verified by driving, not by reading the results table)

| rev4 | Closed? | How verified |
| --- | --- | --- |
| P2-A custody oracle looks through the wrapper; no admit-direction narrowing on `runtimeAbsent` | **yes** | `requireLocalError` asserts `err.(*Error)` AND `!errors.As(err, &*axerror.Error)`; rev4 R05 replayed as R03 → KILLED ×2 by `TestAcquireBackgroundStagesVerifyCustodyRefusal` (`err top-level type = *axerror.Error … want *tmuxserver.Error`); the containment member (R02) KILLED ×2 by `TestAcquireBackgroundRefusesNonDirectoryLeaf`; the old-oracle replay proves the helper is load-bearing; `N-absence-remap-widen` shipped and KILLED ×2 in the producer harness; `N-background-absence-remap` kept. The ownership member of the same class is the residue (P3-A below) |
| P2-B `Request.Name` re-opens the §3.2 path per call | **yes** | The field is gone (`grep -n Name acquire.go` finds only `RuntimeDirName`); `Acquire` passes `RuntimeDirName` at all three sites; `TestAcquireOutcomeSocketEqualsSpecPath` asserts the literal `<root>/tmux/ax.sock` on both callers; reviewer rows R04/R05/R06 (a `"ax-tmux"` literal at the lexical, ensure and verify sites respectively) KILLED ×2 by 18/8/6 tests including that test on the relevant caller |
| P3-A absence remap pinned on macOS only | **yes** | `TestAcquireBackgroundVerifiesWithoutCreating` runs macos/linux/wsl2; rev4 R14 replayed as R07 → KILLED ×2 by the linux and wsl2 subtests exactly |
| P3-B "every background test arms a spy" false for 4 of 11 | **yes** | Mechanical census: 12 background-path `Acquire` tests, 12 arm `spawnSpy` and assert zero calls (the 13th `CallerBackground` match is the helper test `TestCheckCreationAllowedRefusesBackground`, not an `Acquire` test); effect census R08 (a guarded direct-spawn plant at the top of `acquireBackground`) reddens 11 top-level tests / 20 lines — the 12th, `TestAcquireRefusesWindowsPlatform`, refuses at the lexical gate before the function is entered, which is correct. README and acquire.go:137 now scope the sentence to background-path `Acquire` tests and it is true |
| Obs-1..5 | **yes** | row 6 ELOOP darwin-equivalence; B15 gains the unopenable-root, `Fchmod`, `Fsync` and leaf-`Stat` arms; B16 nested invocation (owner 1c28dz); row 23 live-member scoping; B19 caller-contract sentence |

## P2 — must fix or record as a reviewed decision

### P2-A. The background broker-or-refuse rule is pinned at the entry only for the fixture shapes: three realistic server-admission shapes and one realistic fallback guard walk through the committed suite

SPEC §4.2 1419-1424: a background caller "MAY contact an already-running authenticated Aqua
broker and its **attested** AX tmux server; if neither exists it MUST return
`capability_unavailable` … and MUST NOT fall back to direct server creation." The candidate
implements both halves correctly (`CheckBrokerContact` → `CheckServerAttested`; no spawn call on
the path; `checkCreationAllowed` consulted by call), and the committed background miss table
drives five report shapes. Four plants that admit exactly one realistic member each SURVIVED the
full committed suite in both passes; each is KILLED by a reviewer probe that passes on the
untouched candidate:

| Row | Plant (readiness.go / acquire.go) | Member admitted under the plant | Committed suite | Probe |
| --- | --- | --- | --- | --- |
| R09 | before `return CheckServerAttested(report.Server, wantGeneration)`: `if RawGeneration == "" && !Has(realm) { return nil }` | the zero `RealmAdmission{}` server report with a live same-user generation-bound principal — the realistic nothing-reconciled state (the exact shape rev3 P2-C closed on the foreground path) | **SURVIVED ×2**, `Acquire` returns `Via: broker` for an unattested server | K02 kills |
| R10 | same site: `if RawGeneration == "" && Has(realm) { return nil }` | rows present, generation empty, wanted generation bound | **SURVIVED ×2** | K03 kills |
| R19 | same site: `if Has("headless_creation") && !Has(realm) { return nil }` | decoy-only admission with a bound generation — the member the foreground entry pins via `TestAcquireForegroundRefusesRunningUnattested` | **SURVIVED ×2** | K05 kills |
| R18 | inside the miss arm, after `checkCreationAllowed` refuses: `if req.Deps.ProbeServer != nil && req.Deps.Spawn != nil { return acquireForeground(req, socket) }` | a background miss whose `Dependencies` carry the foreground probe — realistic, because the lifecycle adapter passes one shared `Dependencies` value to every caller | **SURVIVED ×2**: no committed background test arms `ProbeServer`, so the spy census (`Spawn`) never observes the fallback; the unguarded shape (`D-background-fallback`) is still killed by 11 tests | K04 kills |

Why this is one class: the broker path drives exactly two server-admission shapes (`{gen, no
rows}` as "unattested server" and `{rows, stale gen}` as "stale server admission") and one
dependency shape (`Spawn` armed, `ProbeServer` nil). Rejection pattern (b) as stated in every
brief of this task — "if a rule is implemented TWICE (… an exported entry and its internal
sibling), each side needs its own narrowing mutant" — applies: the "attested server" rule is
pinned at the helper (`TestCheckServerAttestedRefusesEachMissingConjunct`) and at the foreground
entry, but the broker-side wiring in `CheckBrokerContact` has no row that admits a member, and
rows 27/28/31 are measured for one fixture each. The DoD lines "negative tests that fail when the
gate admits what it must reject" and "every gate ships at least one NARROWING mutant" are not met
for the broker-side attested-server wiring; the no-fallback proof has a blind spot keyed on the
sibling dependency. Production is correct today; the fix is tests plus rows:

1. extend `TestAcquireBackgroundMissReturnsTypedUnavailable` with three subtests — zero
   `RealmAdmission{}`, `{Admitted: realm row, RawGeneration: ""}`, `{Admitted: decoy only,
   RawGeneration: bound}` — asserting the literal `capability_unavailable`, exit 6, the five typed
   details, zero spawns (K02/K03/K05 are the templates), and mirror the three rows in
   `TestCheckBrokerContactRefusesEachMissingFact`;
2. ship a narrowing row at the `CheckBrokerContact` server wiring (the R09 shape; R10/R19 may
   share the anchor) and name it under rows 27/28/36;
3. arm a fail-if-called `ProbeServer` spy in every background miss test (at least the two miss
   tables and `TestAcquireBackgroundVerifiesWithoutCreating`), so that a fallback through the
   foreground path is visible; ship it as a second labelled additive row
   (`D-background-fallback-foreground`, the R18 shape) and extend the row-31 / README sentence to
   say the census arms both foreground dependencies.

## P3 — fix in the same rework

- **P3-A. The absence classifier's reject class is pinned at the entry for the mode and
  containment members only.** Reviewer row R01 (`runtimeAbsent` widened to admit
  `runtime ownership`) **SURVIVED ×2**: no committed background `Acquire` test stages foreign
  ownership (the ownership gate is pinned at the helper by `TestEnsureRuntimeDirRefusesForeignOwnership`
  on both entries, and R13 shows the verify-side wiring is killed there), so a foreign-owned leaf
  on the background path would return `capability_unavailable` with the custody `*Error` as its
  redacted cause — refusal-to-refusal, but the misclassification row 52 forbids ("an unsafe leaf
  keeps its custody code (top-level *Error, never the unavailable wrapper)"). Probe K01 (background
  `Acquire` + the `effectiveUID` seam → top-level `tmux_unsafe_runtime_dir at runtime ownership`,
  broker not probed, spawn 0) passes on the candidate and kills R01. Add it as
  `TestAcquireBackgroundStagesOwnershipRefusal` under row 52, or state the ownership member as
  helper-pinned only.
- **P3-B. A censusable "before" that is false.** acquire.go:140-143 ("Refusals precede side
  effects: malformed input, ambient reuse, and missing dependencies all refuse before any directory
  is created") and results.md:61-62 ("refuses malformed input, ambient reuse, and missing
  dependencies before any side effect"). Probe P01: a foreground request with `SessionID =
  "12345"` **creates `<root>/tmux` and calls `ProbeServer` before** refusing `tmux_invalid_arguments
  at session identity` (spawn 0). The session identity is malformed input; `acquireForeground`
  validates it only inside `BuildArgv`, after `EnsureRuntimeDir` and the probe. Either parse the
  session (`scalar.ParseUUIDv7`) at the `Acquire` entry before any side effect and add the
  no-directory/no-probe assertions to `TestAcquireForegroundRefusesMalformedSession` (row 45), or
  reword both sentences to the members that are true (caller, realm details, root/name grammar,
  override, collision, dependencies). Rejection pattern (f).

## Observations (not findings; a sentence each in TRACEABILITY if the producer agrees)

1. Probe P02: a comma inside the runtime root (`<root>/a,b`, admitted by the POSIX grammar) makes
   `tmuxEnvSocket` split at the wrong comma, so a `TMUX` value naming the derived socket in the
   real encoding walks past the collision gate and `Acquire` spawns. tmux(1) itself splits `TMUX`
   at the first comma, so the decoding is faithful to the source of the encoding; worth a clause in
   row 23/B16 ("roots containing a comma are outside the TMUX member's encoding").
2. Probe P03: a cwd-relative spelling of the derived socket (`tmux/ax.sock` with the process cwd
   at the root) walks past the cleaned-string collision gate — the same bound class as the B16
   symlink alias (no path equivalence is applied); B16 could say "symlinked or relative alias".
3. Probe P04: `Dependencies.Spawn` is inert on the background path (nil spawner + live broker
   contacts) — consistent with the design; it is what makes P2-A's R18 realistic, since the shared
   struct carries both foreground dependencies on a background call.
4. Probe P05: a `linux` request on this darwin host commits and verifies the same unix custody on
   both callers (row 14 through `Acquire`, not only `EnsureRuntimeDir`).
5. Probe P06: an invalid-UTF-8 typed detail (`Generation = "gen\xff"`) passes the entry's
   non-empty check, is probed against the broker, and then reaches the constructor-failure arm
   `backgroundUnavailable` → `tmux_invalid_arguments at realm details` (acquire.go:289-291) — an
   input-reachable refusal arm with no committed test, not listed under B15 (whose members are all
   seam-only). A 10,000-character remediation is admitted by the landed detail bound. Pin it with
   one subtest or list it under B15 as input-reachable; note that the broker probe precedes the
   refusal. While editing B15: the coverprofile also shows the commit-side leaf-`Stat` arm
   (runtime_unix.go:76-79) and the `secprim.NewGuard` refusal arm (runtime.go:94-96) uncovered;
   both are the same unstageable/unreachable class as their listed siblings but are not named in
   the list.
6. The producer archive's `static-gates.log` is empty and no exit codes for the
   `GOOS=windows`/`GOOS=linux` vet and test-compile gates are recorded anywhere in the archive; the
   green claims live in the results prose only. Rerun here (all exit 0), so not a finding — but
   record the exits next time.
7. Coverage: the uncovered `background fallback` arm (acquire.go:203) is reachable only under a
   weakened `checkCreationAllowed`; `N-creation-literal` exercises it under mutation, which is the
   intended design (rev2 F1). Fine as is.

## §M — reviewer battery (20 rows, two passes, `battery/reviewer_battery.py` in the evidence archive)

Every row: anchor count verified (all 20 applied, none NOT_APPLIED, none COMPILE_FAIL), plant
applied to the mutant copy, the committed suite run as shipped (full package, `-count=1 -v`, no
`-run` mask), exit + raw output logged, file restored from a byte copy and re-compared, package
directory hash equal before/after (`c8eb3ed00bc20f82`), `internal/` re-diffed against the
candidate copy after each pass. Rows predicted SURVIVED were additionally run with
`zz_review_probe_test.go` present and the named killer selected (`*.with-probe.log`).

| Row | Kind | Gate (site) | Committed suite | With probe | Killer(s) |
| --- | --- | --- | --- | --- | --- |
| R01-absence-admits-ownership | narrowing | `runtimeAbsent` admits `runtime ownership` (acquire.go:252) | **SURVIVED** | KILLED | K01 (P3-A) |
| R02-absence-admits-containment | narrowing | `runtimeAbsent` admits `runtime containment` (acquire.go:252) | KILLED | — | TestAcquireBackgroundRefusesNonDirectoryLeaf |
| R03-absence-admits-mode-rev4R05 | replay | `runtimeAbsent` admits `runtime mode` (acquire.go:252) | KILLED | — | TestAcquireBackgroundStagesVerifyCustodyRefusal |
| R04-name-literal-lexical-site | narrowing | `"ax-tmux"` at the Acquire lexical step (acquire.go:151) | KILLED | — | 18 tests incl. TestAcquireOutcomeSocketEqualsSpecPath/{foreground,background} |
| R05-name-literal-ensure-site | narrowing | `"ax-tmux"` at the foreground ensure (acquire.go:217) | KILLED | — | 8 tests incl. TestAcquireOutcomeSocketEqualsSpecPath/foreground, TestAcquireForegroundSpawnsDedicatedServer |
| R06-name-literal-verify-site | narrowing | `"ax-tmux"` at the background verify (acquire.go:179) | KILLED | — | 6 tests incl. TestAcquireOutcomeSocketEqualsSpecPath/background, TestAcquireBackgroundContactsBroker |
| R07-absence-only-macos-rev4R14 | replay | absence remap limited to macOS (acquire.go:180) | KILLED | — | TestAcquireBackgroundVerifiesWithoutCreating/{linux,wsl2} |
| R08-spy-guarded-plant-top | additive | guarded `Spawn` call at the top of `acquireBackground` (acquire.go:176) | KILLED (11 top-level tests, 20 lines) | — | every background-path Acquire test that enters the function |
| R09-broker-empty-admission-admits | narrowing | broker-side server wiring admits the zero admission (readiness.go:86) | **SURVIVED** | KILLED | K02 (P2-A) |
| R10-broker-empty-generation-admits | narrowing | same site, admits rows-without-generation | **SURVIVED** | KILLED | K03 (P2-A) |
| R11-details-drop-generation | narrowing | typed-details gate drops the generation member (acquire.go:148) | KILLED | — | TestAcquireRefusesMissingRealmDetails/generation (reroute to `dependencies`) |
| R12-details-drop-brokerstate | narrowing | same, broker-state member | KILLED | — | TestAcquireRefusesMissingRealmDetails/broker_state (reroute) |
| R13-verify-ownership-wiring-drop | narrowing | verify side consults `verifyOwnership` and drops the verdict (runtime_unix.go:134) | KILLED | — | TestEnsureRuntimeDirRefusesForeignOwnership (Verify leg) |
| R14-argv-drop-detached | narrowing | `-d` removed from the spawn vector (argv.go:28) | KILLED | — | TestBuildArgvAddressesOnlyTheDerivedSocket |
| R15-argv-tail-attach | narrowing | `ax pane` → `ax attach` (argv.go:30) | KILLED | — | TestBuildArgvAddressesOnlyTheDerivedSocket, TestAcquireForegroundSpawnsDedicatedServer |
| R16-verify-mode-group-bits-only | replay | verify mode `!= 0700` → `&0o077 != 0` (runtime_unix.go:131) | KILLED | — | TestEnsureRuntimeDirRefusesNarrowerMode/{0600,0500} |
| R17-commit-nofollow-drop | replay | drop `O_NOFOLLOW` on the commit Openat (runtime_unix.go:59) | KILLED | — | TestEnsureRuntimeDirRefusesSymlinkLeaf |
| R18-fallback-via-foreground-guarded | additive | fallback into `acquireForeground` guarded on `ProbeServer` (acquire.go:200-202) | **SURVIVED** | KILLED | K04 (P2-A) |
| R19-broker-decoy-admission-admits | narrowing | broker-side server wiring admits a decoy-only admission (readiness.go:86) | **SURVIVED** | KILLED | K05 (P2-A) |
| C-reviewer-control | control | comment-only edit in readiness.go | SURVIVED | — | (must survive; proves the battery observes outcomes) |

Denominator: 20 applied rows = 13 narrowing + 4 replay + 2 additive + 1 control. Of the 13
narrowings, 9 KILLED by the committed suite, 4 SURVIVED (R01, R09, R10, R19), each proven real by
a probe kill (production correct, suite blind). Of the 2 additives, R08 KILLED and R18 SURVIVED
(probe kill). All 4 replays KILLED — the rev4 closures hold. Every KILLED row was killed by exactly
the tests named above (kill lines in the raw logs).

## §P — reviewer probes (11, `logs/probes-rev5.log`)

K01 background + foreign ownership → top-level `tmux_unsafe_runtime_dir at runtime ownership`,
broker not probed, spawn 0 (baseline for R01); K02 broker path + zero `RealmAdmission{}` →
`capability_unavailable`, spawn 0, and `CheckBrokerContact` → `server attestation` (baseline for
R09); K03 broker path + admitted-without-generation → `capability_unavailable` / `server
generation` (R10); K04 background miss with `ProbeServer` AND `Spawn` armed → typed unavailable,
neither dependency called (R18); K05 broker path + decoy-only admission → `capability_unavailable`
(R19); P01 malformed session creates the directory and probes the server before refusing (P3-B);
P02 comma-root TMUX encoding walks through (obs 1); P03 relative alias walks through (obs 2); P04
nil spawner on the background path contacts (obs 3); P05 linux request on both callers (obs 4);
P06 invalid-UTF-8 generation reaches the constructor-failure arm after the broker probe, a 10k
remediation is admitted (obs 5).

## What held under attack (keep)

- Dedicated-server rule per vector: override refused on all five members plus whitespace through
  `Acquire` with no spawn and no directory; byte-identical, cleaned and real-encoding collisions
  refused; argv gate refuses all five and the vector is pinned element-by-element (R14/R15 KILLED);
  `Acquire` never reads the environment or execs; the §3.2 path is now pinned at the entry on both
  callers and at all three `RuntimeDirName` sites (R04/R05/R06 KILLED).
- Background rules: broker-or-refuse with the literal `capability_unavailable`, exit 6 and the
  five typed details asserted as literals; the absent-leaf cold state returns the typed refusal on
  macOS, Linux and WSL2 with no probe, no spawn and no directory (R07 KILLED); the unguarded
  fallback is killed by 11 tests (R08, D-background-fallback); same-user generation-bound broker
  principal at the entry (producer N-broker-uid/-generation reproduced). The residue is P2-A.
- Attestation: landed `terminalbackend.Admitted` consumed through `Has` composed with the
  generation equality; the foreground running arm is pinned at the entry for empty admissions;
  stale, decoy and both-empty shapes refuse at the helper and on the foreground entry.
- Custody: the top-level oracle now sees the wrapper (R03 KILLED; old-oracle replay SURVIVES the
  same mutant); mode/containment members of the absence class pinned at the background entry;
  FIFO and regular-file leaves refuse on both entries without blocking; symlink leaf refuses on
  both entries (R17 KILLED); exact 0700 on both sides (R16 KILLED); ownership wiring on both sides
  (R13 KILLED); read-only root, missing root, handle mismatch refuse; the real-SIGKILL crash test
  converges.
- Determinism: 0 SKIP with tmux unresolvable, 140 RUN; no real-tmux witness (B1).
- Boundary: `internal/traceability` untouched; README section carries no CLI/capability claim; no
  trunk drift; producer evidence archive complete and digest-verified; LOGBOOK newest-first.

## Coverage ratio measured against the 58-row table

**58 of 58 rows are driven as their statements are written**; the ratio is not inflated. The
gaps sit beside the statements, at the member level: rows 27/28/36 (the broker-path "attested
server" conjunct is driven for two of five admission shapes — P2-A), row 31 (the no-fallback
census arms `Spawn` only — P2-A), row 52 (the custody-code half is measured for the mode and
containment members, not ownership — P3-A), and acquire.go:140-143 claims a "before" that row 45
does not measure (P3-B).

## Rework scope (for the producer)

1. **P2-A (tests + rows, no production change required)**: add the zero-admission,
   admitted-without-generation and decoy-only server shapes to
   `TestAcquireBackgroundMissReturnsTypedUnavailable` and `TestCheckBrokerContactRefusesEachMissingFact`
   (K02/K03/K05 in the reviewer probe file are the templates); ship a narrowing row at the
   `CheckBrokerContact` server wiring (R09 shape) named under rows 27/28/36; arm a fail-if-called
   `ProbeServer` spy in the background miss tests and `TestAcquireBackgroundVerifiesWithoutCreating`,
   ship the R18 shape as a labelled additive row, and extend the row-31/README sentence to say the
   census arms both foreground dependencies.
2. **P3-A**: `TestAcquireBackgroundStagesOwnershipRefusal` (K01 template) named under row 52, or
   state the ownership member as helper-pinned only.
3. **P3-B**: validate the session identity at the `Acquire` entry before `EnsureRuntimeDir` and
   the probe (and assert no directory / no probe in `TestAcquireForegroundRefusesMalformedSession`),
   or reword acquire.go:140-143 and the results sentence to the members that are true.
4. Optional sentences: comma-root and relative-alias scoping under row 23/B16; the
   input-reachable constructor-failure arm under B15 or a subtest; record static-gate exit codes in
   the archive.
5. Rerun the package suite (tmux hidden), `GOOS=windows GOARCH=amd64 go vet ./...`, the harness
   twice, and the 30-command suite; refresh results/matrix/archive; hand off as revision 6.
   Everything in "What held under attack" needs no change — do not re-litigate it.
