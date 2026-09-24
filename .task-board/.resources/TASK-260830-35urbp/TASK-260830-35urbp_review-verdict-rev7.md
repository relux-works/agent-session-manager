# TASK-260830-35urbp — review verdict, Change Request revision 7

**Verdict: ACCEPTED** (`accept_cr(TASK-260830-35urbp, revision=7)` → `integrating`). No P1, no P2.
Every rev6 finding (P2-A commit-side `O_DIRECTORY` arm, P3-A WSL2 miss member, P3-B catalog-derived
decoy set, observations 2-5) is closed with driven evidence, and each of the three rev6 surviving
plants is now KILLED twice by the committed suite with the admit direction, the no-hang property
and the member-level attribution the rework asked for. The revision ships no production change
(`git diff 11259068 5e58cf6b` touches only test files, the harness, TRACEABILITY, README and
LOGBOOK), so everything that held under attack in rev6 still holds by construction; I re-attacked
the reworked cells and four gates the producer harness does not row. One residue is recorded
below as a **P3 (reviewed decision, not blocking)**: a fresh-decoy narrowing at the FOREGROUND
attach wiring survives — the exact rev6 P3-B shape one entry over, on a leaf-internal rule whose
attach semantics are the final leaf's (g0pcnt) scope. Production is correct at every point this
review drove.

- Reviewer run: RUN-260921-d1007e (claude-opus-5 max), reviewing `CR-TASK-260830-35urbp-7`
  revision 7 published by RUN-260921-9158bf (muse-spark max) after the rev6 rework brief.
- Reviewed bytes: base `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` after a fresh
  `git fetch origin main`; `git diff --name-only 799c338 origin/main` empty — no trunk drift, no
  stale-copy audit applies), candidate tree `5e58cf6b5cde39294fb45767c854d5bbc2085dfa`, patch
  resource sha256 `d67e4367b5471b276d1ea0fe76c17f1c10b53c6e4c2bf282f1b635f850abb7da` (recomputed
  over the materialized resource, equal to the CR record). The live Story worktree's uncommitted
  tree was recomputed through a temporary index file (`GIT_INDEX_FILE`, live index untouched) and
  equals the CR tree (`5e58cf6b`). Every measurement below was taken on three `git archive` copies
  of that tree (`.temp/TASK-260830-35urbp/review-rev7/{candidate,mutant-copy,probe-copy}`; the
  candidate copy re-hashes to tree `5e58cf6b` through a scratch `GIT_DIR`; `internal/` of the two
  lean copies re-diffed equal to the candidate copy after every battery pass). No product edit,
  commit, checkpoint or integration was performed; the live worktree, index, branch and HEAD are
  unchanged (`HEAD` = `799c338`, branch `task-board/story/STORY-260830-2t4g7i`).
- Rev6 → rev7 delta (`git diff --stat 11259068 5e58cf6b`): 9 files, +258/−53 —
  `acquire_test.go`, `runtime_test.go`, `readiness_test.go`, `tmuxserver_test.go`,
  `crash_unix_test.go`, `mutant_harness.py`, `TRACEABILITY.md`, `README.md`, `LOGBOOK.md`.
  No `.go` production file changed.
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §3.2 (765-813; the socket
  clause at 806-812), §4.2 (1403-1449), §4.3 (1450; Windows clause 1458), §4.4 (1468), with
  §4.C (1082) / §4.D (1220) / §4.E (1309) as context. Every line citation in the package
  (`SPEC 806-807`, `806-808`, `810-812`, `1458`) matches the pinned file; no `0.5.0` citation
  exists in the package (grep empty); the task record's v0.5.0 Scope is stale, as in every prior
  round. The derived path is `<Root>/tmux/ax.sock` with `Request.Root` the Runtime IPC root
  (unchanged since rev4).
- Note on the rev7 reviewer brief: its templated line "revisions 1..6 … carry no review verdict
  and no prior review findings" is wrong for this task — rev1..rev6 were each reviewed and routed
  CHANGES REQUESTED (six verdict resources attached). Rev7 was reviewed on its own merits AND
  against closure of every rev6 finding.

## What was reproduced (all on tree 5e58cf6b; PATH = `{go,gofmt,python3 symlinks}:/usr/bin:/bin:/usr/sbin:/sbin` with no tmux binary resolvable, `TMUX`/`TMUX_TMPDIR` unset, `pgrep tmux` empty; umask 022, uid 502, darwin/arm64, go1.25.5, host load 9-22 throughout — a second Story producing concurrently)

| Check | Result | Log (evidence archive) |
| --- | --- | --- |
| Configured suite, 30 commands read from `spawn.worktree_isolation.validation.commands` of the candidate config (count 30, verified by read) | **30/30 exit 0, every command rerun by this reviewer** on the candidate copy (`suite/suite-exits.txt`; start and post-suite re-added tree both `5e58cf6b`, no stray artifact in the copy — the runner's `git status --short` count of 4504 is the scratch repository's staged-add listing, it has no commit, not dirt; wall clock 22:14:24→22:32:34 UTC, 18m10s, host load 9-46): 1 gofmt, 2 build, 3 vet; 4 `go test ./... -count=1 -v` (216s): 45/45 ok, 0 package FAIL, 0 top-level `--- FAIL`, 14 pre-existing nested `--- SKIP` all outside this package, **0 SKIP in `internal/tmuxserver`**; the 21 indented `--- FAIL` lines are mutant-kill excerpts inside the passing `TestSmokeMutantsAreKilled` (11) and `TestCensusLiveEventOwnershipPlants` (10); 5 race gate **run in 6 bounded groups of 8/8/8/8/8/5 packages (= all 45 from `go list ./...`) with the identical flags** (117+59+28+22+367+73 = 666s; one call would have exceeded the shell bound at this host load): 45/45 ok, 0 `DATA RACE`; 6 cover (163s): 45/45 ok, tmuxserver **92.7%**; 7-23 seventeen fuzz seeds exit 0; 24 tracecheck exit 0 (`clauses_discharged=70/574`, unchanged — `internal/traceability` untouched); 25 cataloggen -check exit 0; 26 linux build; 27 windows build; 28 JSON parse exit 0; 29 `task-board validate` exit 0 as configured (180 pre-existing issues against the authoritative board; a second read-only run with `--board-dir` on the copy's committed snapshot also exit 0, 703 issues — neither is this leaf's scope); 30 `git diff --check` exit 0 | `suite/cmd01.log` … `suite/cmd30.log`, `suite/cmd05-race-{aa..af}.log`, `suite/suite-exits.txt` |
| Hosted-CI-shaped gates (test files included) | `gofmt -l` over every `.go` file empty; `go vet ./internal/tmuxserver` exit 0; `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; `GOOS=windows GOARCH=amd64 go test -c ./internal/tmuxserver` exit 0; `GOOS=linux go vet` + `go test -c` exit 0 | `logs/static-gates.log` |
| Package suite, tmux unresolvable | `go test ./internal/tmuxserver -count=1 -v -coverprofile`: **230 RUN, 82 top-level PASS, 148 nested PASS, 0 FAIL, 0 SKIP**, 92.7% statements (env proof in `logs/pkg-tests-notmux-env.txt`). The 12 uncovered blocks are byte-for-byte the rev6 list (B15 error arms ×8, the `AfterMkdir` hook exercised in the crash child, the under-mutant-only `background fallback` arm acquire.go:207, the B12 `BuildArgv` error arm acquire.go:243) — expected with no production change. Determinism holds; no real-tmux witness exists (bound B1); the production entries are what every test drives | `logs/pkg-tests-notmux-v.log`, `logs/tmuxserver-cover.out` |
| Producer harness, two passes on the mutant copy | **76/76 both passes** (73 narrowing + `D-background-fallback` + `D-background-fallback-foreground` KILLED exit 1, `C-control` SURVIVED exit 0); verdict lists identical to each other and, modulo the `[exit N]` suffix, to the producer's attached pass 1 and pass 2; every raw log carries its `exit:` line (76+76); package files byte-identical to the candidate after each pass; zero `__pycache__`/`.pyc` in the mutant copy (one `__pycache__` appeared in MY candidate copy from my own harness-row enumeration without `PYTHONDONTWRITEBYTECODE` — my artifact, not the candidate's; removed, recorded in `logs/pycache-found-before-cleanup.txt`, tree re-hash unaffected). The new row `N-containment-odirectory-commit` kills exactly as claimed: `TestAcquireForegroundRefusesNonDirectoryLeaf` `err = nil` (admit direction through the entry), `…/regular_file_0700` `err = nil`, `…/fifo_without_blocking` `EnsureRuntimeDir BLOCKED for 5s`, `…/regular_file_0600` by reroute (`runtime mode`) | `harness-pass1/`, `harness-pass2/`, `logs/harness-pass*-verdicts.log`, `logs/harness-timing.txt` |
| Producer evidence archive | real gzip (sha256 `25c5628649e1…`); MANIFEST 204 entries, no self-entry, **204/204 sha256 verify**, no unlisted file; one raw log per plant per pass with `exit:` (76+76); producer pass1 == pass2 == reviewer rerun; `suite-exits.txt` 30 exits all 0 with the race gate in 6 groups; `pkg-tests-notmux-v.log` 230 RUN / 0 SKIP with the env proof; `census-symmetry.log` + `census_check.py` + `spy_census.py` + `replay_rev7.py` with the R01/R07/R07b/R16 replay logs; zero `__pycache__` | (checked in place) |
| Gate × entry census + AC table, verified mechanically (`logs/census_verify.py`, my own parser) | 58 AC rows, numbers 1..58 unique; **82 test names in TRACEABILITY, 82 defined, 0 missing, 0 unnamed**; 66 row names in TRACEABILITY all exist in the 76-row harness (the 10 unnamed rows are the per-vector collision/override members named in aggregate, plus `C-control`); 24 gates × 11 entries = **264 cells; 50 M / 4 B / 77 U1 + 133 U2 = 210 U** — the stated counts reproduce; every M cell's named rows and tests exist; the matrix census section is line-identical to TRACEABILITY's; the matrix's 58 numbered rows are unique (rev6 obs-3 row-46 duplicate gone) | `logs/census-verify.log` |
| Both-spy census, reproduced mechanically (my own parser over every `Acquire(`-driving test that uses `CallerBackground`) | **24/24** arm both `spawnSpy` and `probeServerSpy` (the rev7 addition `TestAcquireBackgroundRefusesEveryCatalogDecoy` included) | `logs/spy-census-reviewer.log` |
| Reviewer battery, 13 rows × 2 passes + 1 probe row | identical verdicts both passes; see §M | `battery/pass1/`, `battery/pass2/`, `battery/pass1-probe/`, `battery/reviewer_battery.py` |
| Reviewer probes, 4 tests (16+1+1+2 subtests) | all PASS on the untouched candidate; K31 kills R31; P-FIFO-FG, P11 and PR01 witness as stated; see §P | `logs/probes-rev7-baseline.log`, `battery/zz_review_probe_test.go` |
| Boundary / hygiene | `internal/traceability` untouched (`git diff base tree -- internal/traceability` empty); no `0.5.0` citation; no `__pycache__`/`.pyc` in the CR tree; production imports unchanged (no `os/exec`, `Getenv`, `Environ`, `localstore`, `catalog` — the catalog derivation is test-only, `tmuxserver_test.go` imports `internal/catalog`); README section carries no `ax` command, `doctor` or capability claim and its two touched numbers (24/24 census, 73 narrowing rows) match the measured values; LOGBOOK rev7 block newest-first; crash evidence real (`TestEnsureCrashChildSelfTerminates` SIGKILLs the child inside `AfterMkdir`, now with the child's combined output captured into the failure message — rev6 obs-2) and idempotency evidence present (`TestEnsureRuntimeDirIsIdempotent`, `TestAcquireCreatesRuntimeDirIdempotently`, `TestAcquireForegroundSpawnIsIdempotentAcrossRetry`); the server socket's own crash evidence remains B9 (lifecycle leaf) | `logs/static-gates.log`, `logs/census-verify.log` |

## Closure of the rev6 findings (verified by driving, not by reading the results table)

| rev6 | Closed? | How verified |
| --- | --- | --- |
| P2-A commit-side `O_DIRECTORY` arm: no narrowing row, admit direction and no-hang unmeasured | **yes** | Replay R01 (drop `O_DIRECTORY` from the COMMIT `Openat`, full suite, no `-run` mask) **KILLED ×2** by `TestEnsureRuntimeDirRefusesNonDirectoryLeaf/regular_file_0700` (`err = nil, want … runtime containment` — the admit direction), `…/fifo_without_blocking` (`EnsureRuntimeDir BLOCKED for 5s on a FIFO leaf` — the no-hang property), `TestAcquireForegroundRefusesNonDirectoryLeaf` (`acquire_test.go:812: err = nil` — the foreground entry admits the 0700 leaf), and `…/regular_file_0600` by reroute as before. The shipped row `N-containment-odirectory-commit` is anchored (count 1), KILLED ×2 in the producer harness with the same four kill lines; row 5 names the commit-side fixtures, the foreground-entry test and both deadline tests; the `commit × ENS` and `commit × AFG` census cells name the row; the `N-containment-odirectory-verify` note now points at the commit row instead of asserting it. The shared `verifyWithDeadline` helper is entry-named |
| P3-A no-fallback miss arm pinned for macOS/Linux only | **yes** | Replay R16 (fallback `Spawn` guarded on WSL2) **KILLED ×2** by exactly `TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback/wsl2` (`spawn calls = 1, want 0`), the `linux` member staying green — member-level attribution holds. My R60 (foreground re-entry guarded on WSL2) KILLED ×2 by the same `wsl2` member only; R61 (guarded on Linux) KILLED ×2 by the `linux` member only. `BG_MISS_LINUX` already named the test, so `N-creation-literal` and both `D-` rows now kill through the wsl2 subtest too (verified in the producer raw logs: `D-background-fallback.log` and `N-creation-literal.log` list `…/linux` and `…/wsl2`); rows 29/31 name it |
| P3-B attestation membership pinned for one decoy of sixteen | **yes** (helper + broker contact + background entry) | Replay R07 (`local_attach` narrowing at the helper) **KILLED ×2** by exactly `TestCheckServerAttestedRefusesEveryCatalogDecoy/{local_attach,all_decoys_together}` and `TestAcquireBackgroundRefusesEveryCatalogDecoy/{local_attach,all_decoys_together}`; R07b (`reboot_restoration`) KILLED ×2 by the same two tables on its own member; R47 (fresh-decoy admission at the BROKER-CONTACT wiring, readiness.go:86) KILLED ×2 through the background entry; R62 (admit exactly the all-fifteen set) KILLED ×2 by the `all_decoys_together` subtests, so the together-fixture is live. `catalogDecoyCapabilities` derives from `catalog.Current().Capabilities` filtered on `NormativeSection == "4.D"` — I checked the generated catalog: exactly 16 rows carry `4.D`, they are the sixteen names of the pinned §4.D table (1223-1245), and the realm row is matched by the spec literal; the derivation asserts the table size and the realm row's presence, so it cannot go vacuous. Residue at the FOREGROUND wiring: §P3-A below |
| Obs 2-5 | **yes** | crash child's `CombinedOutput` in both failure messages; `TestBuildArgvRefusesInvalidArgvShape` named under B12 and the `argv × AFG` cell names B12; matrix row 46 unique; `N-details-constructor` note now states the typed-nil mechanism (matches its raw log: `err top-level type = *axerror.Error (<nil>)`) |
| Obs 1, 6 (bounds B9, B11) | unchanged, witnessed | B9 stated as before; B11 re-witnessed by probe P11 (symlinked intermediate root component followed on Ensure and Verify) |

## P3 — reviewed decision, not blocking (record for the Story; fix in the attach-semantics leaf or the next rework of this leaf)

### P3-A. The foreground attach wiring is member-pinned for the fixture decoy only — the rev6 P3-B shape one entry over

Reviewer row R31 plants, at the FOREGROUND wiring acquire.go:237 (`if report.Running { if err :=
CheckServerAttested(…); err != nil { … } }`), the narrowing `err != nil && !report.Admission.Admitted.Has("local_attach")`
— i.e. "a running server that admits `local_attach` may be attached without the realm row". It
**SURVIVED ×2** the full committed suite: the four foreground running tests use `headless_creation`
(`TestAcquireForegroundRefusesRunningUnattested`), a stale realm admission, an empty admission and
an empty generation, so no committed foreground test presents a fresh non-realm member. The same
plant keyed on `headless_creation` (R31b, my control) is KILLED, so the wiring IS pinned — for one
member of fifteen. Probe K31 (every catalog decoy alone and all together through the foreground
entry with `Running: true` → `tmux_readiness_not_authorizing at server attestation`, spawn 0)
passes on the untouched candidate and kills R31 (`local_attach` + `all_decoys_together`), so
**production is correct**; the membership arm itself (readiness.go:40, one site) is now
member-complete at the helper and through the background entry (R07/R07b/R47/R62 all KILLED).

Why this is recorded as a reviewed decision rather than a blocking finding: (1) the rev6 verdict
scoped the P3-B fix to "at the helper and through the background entry", and the producer did
exactly that — the foreground member is the rev6 verdict's own omission, not a departure from the
brief; (2) the rule at stake ("never attach to a running server whose admission lacks the realm
row") is this leaf's composition rule, not one of the §4.2 MUSTs this review is gating (dedicated
`-S`, no ambient reuse, background never creates, broker-or-refuse, no fallback, hint never
authorizes — all of which held under attack and are member-complete where they were measured);
(3) foreground attach semantics are the declared scope of the final leaf (TASK-260830-g0pcnt,
"generation and attach semantics"), which is the natural place for the foreground decoy table;
(4) the fix is one table and one mask extension, and the test is already written (probe K31,
`battery/zz_review_probe_test.go`, liftable verbatim). The DoD letter is met for this gate — a
narrowing row exists at the wiring (`N-unattested-foreground`, generation member) and negative
tests fail when the wiring admits the fixture members — and the rev6 verdict's own "held under
attack" list stated the arm as pinned at both entries with P3-B as the only residue.

Exact closure (for the orchestrator to carry into the g0pcnt brief, or into this leaf if it is
reworked for any other reason): add `TestAcquireForegroundRefusesEveryCatalogDecoyRunning` (K31's
shape: catalog-derived decoys, each alone and all together, `Running: true`, assert
`tmux_readiness_not_authorizing at server attestation` and spawn 0), extend the
`N-realm-membership` mask with it, ship a wiring row of the R31 shape (or key it on the fixture
member and let the catalog table kill fresh members), and name the table under rows 35/42
and in the `server × AFG` census cell.

## Observations (not findings)

1. The `catalogDecoyCapabilities` helper counts `NormativeSection == "4.D"` rows; the catalog's
   `Family: "terminal_backend"` would be the equivalent filter. Either is fine; the count assertion
   (16) and the realm-presence assertion make drift loud.
2. `TestEnsureCrashChildSelfTerminates` did not flake in this review (1 package run, 1 full-suite
   `-v` run, the race group, the cover run, and 26 battery rows = 30 executions, all PASS); the
   captured child output would now make a recurrence attributable.
3. The results' claim that under the commit-side plant the foreground entry "probes once and
   spawns once into `<root>/tmux/ax.sock`" comes from the producer's scratch probe; the committed
   test stops at `err = nil` before its probe/spawn assertions run. The row's kill does not depend
   on it; the sentence is accurate but not committed-test-measured.

## §M — reviewer battery (13 rows × 2 passes + 1 probe row, `battery/reviewer_battery.py`)

Every row: anchor count verified (all applied; R37's first shape `if true {` was a COMPILE_FAIL —
`created` unused — re-planted as `if created || leafFd >= 0 {`, both raw logs kept), plant applied
to the mutant copy from the pristine candidate bytes, the committed suite run as shipped (full
package, `-count=1 -v`, **no `-run` mask**), exit + raw output logged with an `exit:` header,
file restored from the pristine bytes and re-compared, package directory hash equal before/after
(`2a833808c85b9a56`), `internal/` re-diffed against the candidate copy after the run. R31 was
additionally run with `zz_review_probe_test.go` present and `-run TestReviewK31`.

| Row | Kind | Gate (site) | Committed suite | With probe | Killer(s) |
| --- | --- | --- | --- | --- | --- |
| R01-commit-odirectory-drop | replay rev6 R01 (narrowing, flag) | commit-side non-directory leaf (runtime_unix.go:59) | **KILLED** ×2 — admit direction + no-hang, not only reroute | — | `TestAcquireForegroundRefusesNonDirectoryLeaf` (err = nil), `TestEnsureRuntimeDirRefusesNonDirectoryLeaf/{regular_file_0700 (err = nil), fifo_without_blocking (BLOCKED 5s), regular_file_0600 (reroute)}` |
| R07-membership-fresh-decoy-local-attach | replay rev6 R07 (narrowing) | membership arm (readiness.go:40) | **KILLED** ×2 | — | `TestCheckServerAttestedRefusesEveryCatalogDecoy/{local_attach,all_decoys_together}`, `TestAcquireBackgroundRefusesEveryCatalogDecoy/{local_attach,all_decoys_together}` |
| R07b-membership-fresh-decoy-reboot-restoration | narrowing, second fresh member | same site | **KILLED** ×2 | — | the same two tables on `reboot_restoration` + `all_decoys_together` |
| R16-wsl2-only-fallback-spawn | replay rev6 R16 (additive, platform-narrowed) | creation gate / miss arm (acquire.go:204) | **KILLED** ×2 | — | `TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback/wsl2` only |
| R31-foreground-wiring-fresh-decoy-local-attach | narrowing (wiring) | foreground attach wiring (acquire.go:237) | **SURVIVED** ×2 | KILLED (K31: `local_attach`, `all_decoys_together`) | P3-A |
| R31b-foreground-wiring-fresh-decoy-headless | narrowing (wiring, fixture member; control for R31) | same site | KILLED ×2 | — | `TestAcquireForegroundRefusesRunningUnattested` |
| R37-repair-existing-leaf | narrowing (silent repair: chmod on every opened leaf) | mode gate, commit side (runtime_unix.go:64) | KILLED ×2 | — | `TestEnsureRuntimeDirRefusesWidenedMode`, `…RefusesNarrowerMode/{0600,0500}`, `TestAcquireForegroundStagesCommitPhaseCustodyRefusal` (all `err = nil` — the widened/narrower leaf is repaired instead of refused, through the entry too) |
| R47-broker-contact-fresh-decoy-admits | narrowing | broker-side server wiring (readiness.go:86) | KILLED ×2 | — | `TestAcquireBackgroundRefusesEveryCatalogDecoy/{local_attach,all_decoys_together}` |
| R54-verify-absence-widened-enotdir | narrowing | verify-side absence classifier (runtime_unix.go:120) | KILLED ×2 | — | `TestVerifyRuntimeDirRefusesNonDirectoryLeaf/{regular_file,fifo_without_blocking}`, `…RefusesSymlinkLeaf`, `TestAcquireBackgroundRefusesNonDirectoryLeaf`, `…RefusesSymlinkLeaf` (containment reported as absence → background maps to `capability_unavailable`) |
| R60-wsl2-foreground-reentry | additive, platform-narrowed | fallback into `acquireForeground` on a WSL2 miss (acquire.go:204) | KILLED ×2 | — | `…MissOnLinuxAlsoRefusesWithoutFallback/wsl2` only |
| R61-linux-foreground-reentry | additive, platform-narrowed | same, Linux | KILLED ×2 | — | `…MissOnLinuxAlsoRefusesWithoutFallback/linux` only |
| R62-catalog-decoy-all-together-only-admit | narrowing (`len == 15` and no realm row admitted) | membership arm (readiness.go:40) | KILLED ×2 | — | both `all_decoys_together` subtests |
| C-reviewer-control | control | comment-only edit in acquire.go | SURVIVED ×2 | — | (must survive; proves the battery observes outcomes) |

Denominator: 13 applied rows = 3 replays (R01 narrowing, R07 narrowing, R16 additive) + 7 new
narrowings (R07b, R31, R31b, R37, R47, R54, R62; R31b is the control for R31) + 2 new additives
(R60, R61) + 1 control. Of the 8 narrowings that are not controls, 7 KILLED by the
committed suite and 1 SURVIVED (R31, probe-killed — P3-A). Both additives and all three replays
KILLED. Passes 1 and 2 identical. NOT_APPLIED: none. COMPILE_FAIL: R37's first shape only (kept,
not counted; the re-planted shape measures the arm).

## §P — reviewer probes (`logs/probes-rev7-baseline.log`, all PASS on the untouched candidate)

K31 `TestReviewK31ForegroundRefusesEveryCatalogDecoyRunning` — 15 decoys + all together through
foreground `Acquire` with `Running: true` → `server attestation`, spawn 0 (baseline for R31);
P-FIFO-FG `TestReviewPFifoForegroundEntry` — FIFO leaf through foreground `Acquire` under a 5s
deadline → `runtime containment` immediately, no probe, no spawn (the brief's item 4 FIFO plant at
the creating entry; the committed suite pins it on `EnsureRuntimeDir`, which the entry calls);
P11 `TestReviewP11SymlinkedIntermediateRootFollowed` — a symlinked INTERMEDIATE component of the
runtime root is followed on Ensure and Verify (bound B11 witnessed as stated; the brief's item 4
symlinked-intermediate plant); PR01 `TestReviewPR01Members` — the 0700 and FIFO members on
`EnsureRuntimeDir` as named subtests (both refuse fast on the candidate; under R01 the 0700 member
admits and the FIFO member blocks, read from the R01 raw log).

## What held under attack (keep)

- Dedicated-server rule per vector (unchanged since rev6, no production change): override refused
  on all five members plus whitespace on both callers through `Acquire`; collision refused on all
  five members plus unclean spellings and the real `TMUX` encoding on both callers; one socket
  constructor, one resolution site, one spawn site; `Acquire` never reads the environment or execs.
- Background rules: broker-or-refuse with the literal `capability_unavailable`, exit 6 and the five
  typed details asserted as literals; the miss arm and the no-fallback proof now span macOS, Linux
  AND WSL2 (R16/R60/R61 KILLED on their own members); the fresh-decoy class is closed at the helper,
  the broker contact and the background entry (R07/R07b/R47/R62); the 24/24 both-spy census
  reproduces.
- Custody: the commit-side `O_DIRECTORY` arm is now measured in the admit direction and for
  no-hang on both `EnsureRuntimeDir` and the foreground entry (R01); silent repair of an existing
  leaf is refused (R37); the verify-side absence classifier admits only ENOENT (R54); FIFO and
  symlinked-intermediate behavior witnessed at the entries (P-FIFO-FG, P11).
- Determinism: 0 SKIP with tmux unresolvable, 230 RUN; no real-tmux witness (B1); no mock-only
  degradation.
- Boundary: `internal/traceability` untouched; no trunk drift; README/LOGBOOK/TRACEABILITY/matrix
  consistent with the measured numbers; producer evidence archive complete and digest-verified.

## Coverage ratio measured against the 58-row table

**58 of 58 rows are driven as their statements are written**; the ratio is not inflated (every
named test exists and executes its named entry; the census counts and the both-spy census
reproduce mechanically). The one member-level gap beside a statement is P3-A above: row 35's
"every non-realm Section 4.D member … at the helper and through the background entry" is exactly
true, and the foreground entry (row 42's "running unattested server refuses") is pinned for the
fixture member only. Census: 50 of 264 cells measured as claimed; the `server × AFG` cell's
rows kill, and its member depth is the P3.

## Rework scope (for the producer) — none required for acceptance

Carry-forward for the Story (g0pcnt brief, or this leaf if reworked for another reason):
1. P3-A: foreground catalog-decoy table (`TestAcquireForegroundRefusesEveryCatalogDecoyRunning`,
   K31's shape), `N-realm-membership` mask extended with it, a wiring row of the R31 shape, rows
   35/42 and the `server × AFG` cell updated.
