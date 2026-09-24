# TASK-260830-35urbp — review verdict, Change Request revision 2

**Verdict: CHANGES REQUESTED** (routed `to-dev`). One P2 class, five P3. Every rev1 P1/P2/P3
finding is closed with driven evidence; what remains is one entry-reachable one-sided pin on
the verify custody path plus evidence-hygiene items. The rework is small and enumerated at the end.

- Reviewer run: RUN-260921-5a9d4b (claude-opus-5 max), reviewing CR-TASK-260830-35urbp-2 rev 2
  published by RUN-260921-6e7be5 (muse-spark max) after the rev1 rework brief.
- Reviewed bytes: base `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` at review
  time, `git diff --name-only 799c338 origin/main` empty — no trunk drift), candidate tree
  `19820613a48f6a3ea239e97d878d936804bf01a8`, patch sha256
  `54f83ebec9f3f9ba8867c4cf53c8f131bbc0f54712440ef8c1c3a658a3548cca` (recomputed from the tree
  pair with `git diff base tree | sha256`, equal). The live Story worktree's uncommitted tree was
  recomputed through a temporary index and equals the CR tree. Every measurement below was taken
  on three `git archive` copies of that tree (`candidate`, `mutant-copy`, `probe-copy`, each
  re-hashed to `19820613` through a temporary index before and after use); the live worktree,
  its index, branch and HEAD were not touched; no product edit, commit, checkpoint or
  integration was performed. `git status` in the live worktree is unchanged
  (`M LOGBOOK.md`, `M README.md`, `?? internal/tmuxserver/`).
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §4.2 (1403-1449), §4.3 (1458),
  §4.4, with §4.C/§4.D/§4.E; the task record's v0.5.0 Scope is stale. The candidate cites the
  pinned file only (grep: no `0.5.0` anywhere in the package; every quoted sentence is present
  in the pinned file).
- Note on the rev2 reviewer brief: its templated sentence "revisions 1..1 carry no review
  verdict" is wrong for this task — rev1 was reviewed by RUN-260921-0245e5 and routed CHANGES
  REQUESTED (`TASK-260830-35urbp_review-verdict-rev1.md`). Rev2 was reviewed on its own merits
  AND against closure of every rev1 finding.

## What was reproduced (all on tree 19820613; PATH without any tmux binary, TMUX/TMUX_TMPDIR unset, no tmux process on the host)

| Check | Result | Log (evidence archive) |
| --- | --- | --- |
| Configured suite, 30 commands read from `spawn.worktree_isolation.validation.commands` | **30/30 green, every command rerun by this reviewer**: 1 gofmt (729 tracked .go files), 2 build, 3 vet, 26/27 linux+windows build (`static-gates.log`); 4 `go test ./... -count=1 -v`: 45/45 ok, 0 FAIL, 14 pre-existing SKIP outside this package, 19 indented `--- FAIL` lines are mutant-kill excerpts inside the passing resumesmoke `TestSmokeMutantsAreKilled` (`cmd04-test-v.full.log`); 5 race: 45 ok, 0 DATA RACE, 7m02s (`cmd05-race.log`); 6 `go test ./... -cover -count=1`: 45/45 ok, tmuxserver 91.6% (`cmd06-cover.log`); 7-23 fuzz seeds all exit 0, 24 tracecheck exit 0 (70/574), 25 cataloggen -check exit 0, 28 JSON parse (141 files) exit 0 (`cmd07-25-28.log`); 29 `task-board validate` exit 0 with the board's pre-existing 180 issues (`cmd29-validate.log`); 30 `git diff --check` exit 0 (`cmd30-diffcheck.log`) | `logs/` |
| Hosted CI gate `GOOS=windows GOARCH=amd64 go vet ./...` (test files included) — the rev1 P1-A | **exit 0**; also `GOOS=windows go test -c ./internal/tmuxserver` exit 0 and `GOOS=linux go vet` exit 0 | `static-gates.log` |
| Package suite, tmux unresolvable | `go test ./internal/tmuxserver -count=1 -v`: 59 top-level tests, 113 PASS lines, **0 FAIL, 0 SKIP**. Determinism holds: no row's witness skips without tmux | `pkg-tests-notmux-v.log` |
| Package coverage | 91.6% (matches the results). Uncovered: the mutant-only fallback arm acquire.go:188, the constructor-failure arm acquire.go:260 (driven by my probe P18: invalid-UTF-8/NUL remediation → `tmux_invalid_arguments realm details`, fail-closed), and error arms with no seam (see P3-E) | `tmuxserver-cover.out` |
| Producer harness, two passes | **42/42 both passes** (40 narrowing + D-background-fallback KILLED exit 1, C-control SURVIVED exit 0), verdict lists identical to each other and to the producer's pass 1; mutant copy re-hashed to `19820613` after each pass; zero `__pycache__`/`.pyc` | `harness-pass1/`, `harness-pass2/`, `logs/harness-pass*-verdicts.log` |
| Reviewer battery, 16 rows × 2 passes | identical verdicts both passes; see §M | `reviewer-battery-pass1/`, `reviewer-battery-pass2/` |
| Reviewer probes, 27 tests | all behave as predicted; see §P | `probes-rev2.log`, `zz_review_probe_test.go`, `zz_review_crash_probe_test.go` |
| Boundary / hygiene | `internal/traceability` untouched (diff-stat over that path is empty); no v0.5.0 citation; no `__pycache__`/`.pyc` in the CR tree or any copy; gofmt/vet clean; producer evidence archive is a real gzip, MANIFEST has 111 entries, no self-entry, all digests verify | `static-gates.log` |

## Closure of the rev1 findings (each verified by driving, not by reading the results table)

| rev1 | Closed? | How verified |
| --- | --- | --- |
| P1-A Windows test binary | **yes** | `GOOS=windows GOARCH=amd64 go vet ./...` exit 0 on the candidate copy; `runtime_test.go` carries `//go:build !windows`; `ownership_windows.go` stub compiles the shared files |
| P1-B forked attestation gate | **yes** | `LiveAttestation`/`CachedObservation`/`ManagerNameHint`/`BrokerStatus`/`Request.Hint`/`Cached`/`NeedsCredentials` have zero occurrences in the package or README. `RealmAdmission` = `terminalbackend.Admitted` + `RawGeneration`; `CheckServerAttested` = `Admitted.Has("credential_capable_execution_realm") && RawGeneration == Generation`, the same two arms `axpane.checkRealm` (decide.go:913-928) applies; the host-binding/provider-build cross-bind is stated as B13. The three rev1 holes now refuse through `Acquire`: wrong-generation foreground attach (`TestAcquireForegroundRefusesWrongGenerationAdmission`), wrong-generation broker (`.../stale_server_admission`), and the replayed-cached shape is no longer expressible — a stale admission refuses on the generation arm. Broker principal is typed (`UID` + `Generation`), both arms pinned (N-broker-uid, N-broker-generation, my R06 at the entry wiring) |
| P2-A nine surviving narrowings | **yes** | Each has a committed killer and a shipped row; all KILLED in both of my harness passes: N-acquire-commit-refusal, N-containment-symlink-verify, N-override-tmpdir, N-collision-{inherited,tmuxenv,tmpdir,conventional}, N-argv-tmuxenv, N-fchmod-umask. The unreachable `ObserveAmbient` vocabulary arm is deleted (`secprim` import gone from socket.go) |
| P2-B Windows arm | **yes** | `lexicalRuntimePath` refuses `PlatformWindows` on every host (`TestEnsureRuntimeDirRefusesWindowsOnUnixHost`, `TestAcquireRefusesWindowsPlatform`, N-platform-windows); Windows commit/verify arms are refusal stubs; tripwire and `_ = filepath.Clean` gone |
| P2-C adapter bound | **yes** | B9 stated; the "Production wires live tmux and broker probes" sentence is gone (acquire.go:57-64 now says no production probe/spawner exists) |
| P2-D dead inputs / Verify contradiction | **yes** | dead members deleted; foreground `checkCreationAllowed` call deleted; background calls `VerifyRuntimeDir` (`TestAcquireBackgroundVerifiesWithoutCreating` asserts no directory and no probe; my R25 re-plants Ensure on that path → KILLED; my R07 reorders probe-before-verify → KILLED) |
| P3-A suite key | **yes** | results read the right key; all 30 reported as run firsthand, consistent with the attached logs |
| P3-B umask bound | **yes** | B10; my probe P19 confirms a stranded 0600 leaf refuses `runtime mode` on every retry |
| P3-C symlinked root intermediate | **yes** | B11; my probe P07 confirms the intermediate is followed and the leaf lands under the target |
| P3-D CheckArgv arm | **yes** | `TestBuildArgvRefusesInvalidArgvShape` + B12 + N-argv-shape |

## P2 — must fix or state as a bound

### P2-A. The verify-side "non-directory leaf" refusal is unpinned: the background production path admits a regular file as the runtime directory and hangs on a FIFO under a one-flag narrowing

Row 5 ("Non-directory leaf refuses") names `EnsureRuntimeDir` only. The verify side implements
the same refusal through the `O_DIRECTORY` flag on `verifyRuntimeDir`'s `Openat`
(runtime_unix.go:105-106), and that side is what `Acquire` uses on the background path
(acquire.go:168). No committed test stages a non-directory leaf against `VerifyRuntimeDir` or
the background `Acquire`, and no harness row narrows that flag. Reviewer plant
`R01-verify-odirectory` (drop `O_DIRECTORY` from the verify `Openat`, keep `O_NOFOLLOW`):

- against the committed suite as shipped (full package, no `-run` mask): **SURVIVED** in both
  passes (`reviewer-battery-pass{1,2}/R01-verify-odirectory.committed.log`, exit 0);
- with my probes present: **KILLED** — `TestReviewProbeVerifyRegularFileLeafRefuses` (a
  regular 0700 file at `root/ax-tmux` → `VerifyRuntimeDir` returns nil) and
  `TestReviewProbeFIFOLeafVerifyRefusesWithoutBlocking` (a FIFO at the leaf → `VerifyRuntimeDir`
  **blocks for the full 5s deadline**, because the verify `Openat` carries no `O_NONBLOCK`; the
  landed `secprim.openNoFollow` comment documents exactly this hang class)
  (`R01-verify-odirectory.with-probe.log`).

The commit side is pinned by reroute (`TestEnsureRuntimeDirRefusesNonDirectoryLeaf` asserts the
`runtime containment` detail; dropping `O_DIRECTORY` there reroutes to `runtime mode`). This is
the rev1 P2-A shape "the verify side follows a symlink leaf" recurring one member over: the
symlink member got its verify-side test and row (`TestVerifyRuntimeDirRefusesSymlinkLeaf`,
N-containment-symlink-verify), the non-directory member did not. Production is correct today
(my probes P04-P06 show FIFO leaf/root refuse on both entries without blocking), but the DoD
line "refusing behavior covered by negative tests that fail when the gate admits what it must
reject" is not met on the entry the background path uses.

Fix: a committed `TestVerifyRuntimeDirRefusesNonDirectoryLeaf` (regular file, and a FIFO under
a deadline so the no-hang property is pinned too), the same shape driven through the background
`Acquire` (my `TestReviewProbeBackgroundRegularFileLeafRefusesThroughAcquire` is the template),
and a harness row `N-containment-odirectory-verify` that drops the flag on the verify `Openat`
only. Update row 5 to name both entries.

## P3 — fix in the same rework

- **P3-A. Both-empty generation arms are unmeasured on two exported entries.**
  `CheckServerAttested`'s `wantGeneration == ""` disjunct (readiness.go:43) and
  `CheckBrokerContact`'s `report.Principal.Generation == ""` disjunct (readiness.go:83) survive
  deletion against the committed suite (`R02-attested-empty-generation`,
  `R03-broker-empty-generation`: SURVIVED ×2, exit 0). Every committed row keeps one side
  non-empty, so `"" != "generation-7"` refuses with or without the arm; the arm is load-bearing
  only for the both-empty pair, which my probes P10/P11 drive (`RealmAdmission{Admitted:
  realm}` with want `""` → admitted under the mutant). Through `Acquire` the pair is unreachable
  (the typed-details check refuses an empty `Generation` first, pinned by R20a), so this is
  confined to direct callers of the exported helpers. One both-empty row in each table closes
  it, or state the arm as entry-unreachable.
- **P3-B. "Exactly 0700" is pinned on the widened side only.** `R11a-mode-narrower-create` and
  `R11b-mode-narrower-verify` (`Perm() != 0o700` → `Perm()&0o077 != 0`, admitting owner-narrower
  modes such as 0600/0500) SURVIVED ×2 against the committed suite and are KILLED by my probes
  P19/P24 (a pre-existing 0600 leaf → both entries return nil under the mutant). Not a
  confidentiality exposure — a narrower mode exposes nothing — but doc.go, README, TRACEABILITY
  rows 1/4 and the runtime.go comment say "exact"/"exactly 0700". Either pin the narrower side
  (a pre-existing 0600 leaf refuses `runtime mode` on Ensure and Verify) or restate the claim as
  "no group/world bits" (mode ⊆ 0700) so prose and tests agree.
- **P3-C. The collision gate is byte equality, not path equality.** `checkAmbientDisjoint`
  (socket.go:103) compares strings. An ambient value naming the derived socket through an
  unclean spelling (`dir/./ax-tmux.sock`, `dir//ax-tmux.sock`) or through a symlinked alias of
  the root walks past every member (probes P01-P03: `Acquire` spawns). Row 23's "colliding with
  the derived socket refuses on every member" holds for byte-identical values only. The spec
  rule is unaffected — the socket is derived, never selected from an ambient value — so this
  is a package-local defense-in-depth claim that outruns its tests: either `filepath.Clean`
  both sides (symlink aliases remain a bound) or state the byte-equality bound in TRACEABILITY.
- **P3-D. LOGBOOK carries a rev1 entry describing code that never reached trunk.** LOGBOOK.md
  lines 14-22 (the "TASK-260830-35urbp — Private tmux server management" block under the rev2
  block) still say `CheckLiveAttestation` "admits only complete live sentinel+smoke", "all four
  ambient vectors", "Windows ownership is the creating-user ACL (tripwire-pinned)", "29
  narrowing mutants", "54 of 54 rows", "87.5% statement coverage" — every one of these
  describes the rejected rev1 tree. Neither entry has landed; landing both records a design
  that was deleted before it existed on trunk. Fold the rev1 block into the rev2 block (keep
  F1-F3 as the findings they are) or mark it superseded in place.
- **P3-E. Fail-closed error arms that are prose only.** `handleBoundToPath`'s two stat-failure
  arms ("an unreadable root fails closed into the mismatch refusal", runtime.go:127-139) flip
  to fail-open under `R10-handle-stat-fail-open` and SURVIVE — no test can stage a `Stat`
  failure on an open directory handle without a seam, so the sentence is unmeasured (rejection
  pattern f). Same class: `verifyOwnership`'s `!ok` arm ("a stat without ownership metadata
  fails closed", ownership_unix.go:29-34), unstageable on unix. State both as bounds
  (unmeasured error arms, no seam) or add the seam. Reported honestly as an error-arm row, not
  counted as a narrowing survivor.

## Observations (not findings; worth a sentence in TRACEABILITY if the producer agrees)

- The no-fallback proof covers spawns routed through `Dependencies.Spawn` (D-background-fallback
  and my R14 both KILLED by the zero-call spies). A fallback through a direct `os/exec` call is
  outside the injected model and is excluded today only by the import census (no `os/exec`
  import in the package) — a fact worth stating under B9.
- The mode gate reads permission bits only: setgid/sticky bits on a 0700 leaf are admitted
  (probe P08) and macOS ACLs are not inspected. This matches the landed `hosttrust` custody
  precedent (`custody_unix.go:15`, `Perm() == mode`), so it is the same bound, not a fork.
- The derived socket path is not checked against the platform `sun_path` limit (104 bytes on
  macOS); a deep state root will surface as a spawn failure in the lifecycle leaf's adapter.

## §M — reviewer battery (16 rows, two passes, `reviewer_battery.py` in the evidence archive)

Every row: anchor count verified (all 16 applied, none NOT_APPLIED), plant applied to the
mutant copy, the committed suite run as shipped (full package, no `-run` mask), exit + raw
output logged, file restored, tree re-hashed to `19820613` after each pass. Rows predicted
SURVIVED were additionally run with my probe file present and the named killer selected.

| Row | Kind | Gate (site) | Committed suite | With probe | Killer |
| --- | --- | --- | --- | --- | --- |
| R01-verify-odirectory | narrowing | verify Openat drops `O_DIRECTORY` (runtime_unix.go:106) | **SURVIVED** | KILLED | P09 regular file admitted; P05 FIFO **blocks 5s** |
| R02-attested-empty-generation | narrowing | `wantGeneration == ""` arm (readiness.go:43) | **SURVIVED** | KILLED | P10 |
| R03-broker-empty-generation | narrowing | `Principal.Generation == ""` arm (readiness.go:83) | **SURVIVED** | KILLED | P11 |
| R05-unattested-as-absent | narrowing | `if report.Running` gains `&& Has(realm)` (acquire.go:210) | KILLED | — | TestAcquireForegroundRefusesRunningUnattested |
| R06-broker-uid-self | narrowing | entry passes `status.Principal.UID` as the current UID (acquire.go:175) | KILLED | — | TestAcquireBackgroundMissReturnsTypedUnavailable/foreign-user_broker |
| R07-probe-before-verify | additive | broker probed before custody verify (acquire.go:168) | KILLED | — | TestAcquireBackgroundVerifiesWithoutCreating, TestAcquireBackgroundStagesVerifyCustodyRefusal |
| R08-silent-repair | narrowing | `if created` → always chmod (runtime_unix.go:54) | KILLED | — | TestEnsureRuntimeDirRefusesWidenedMode, TestAcquireForegroundStagesCommitPhaseCustodyRefusal |
| R09-ensure-before-override | additive | foreground Ensure before the override refusal (acquire.go:147) | KILLED | — | TestAcquireRefusesEveryAmbientOverrideVector (all 5 subtests) |
| R10-handle-stat-fail-open | error-arm | stat-failure arms return true (runtime.go:133,138) | **SURVIVED** | (no killer possible without a seam) | — |
| R11a-mode-narrower-create | narrowing | create-side `Perm()&0o077 != 0` (runtime_unix.go:73) | **SURVIVED** | KILLED | P19 |
| R11b-mode-narrower-verify | narrowing | verify-side `Perm()&0o077 != 0` (runtime_unix.go:116) | **SURVIVED** | KILLED | P24 |
| R14-linux-miss-fallback | additive | non-macOS background miss spawns directly (acquire.go:185) | KILLED | — | TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback |
| R20a-details-generation | narrowing | typed-details arm drops `Generation == ""` (acquire.go:140) | KILLED | — | TestAcquireRefusesMissingRealmDetails/generation |
| R20b-details-broker | narrowing | typed-details arm drops `BrokerState == ""` | KILLED | — | TestAcquireRefusesMissingRealmDetails/broker_state |
| R25-background-ensures | narrowing | background calls Ensure instead of Verify (acquire.go:168) | KILLED | — | TestAcquireBackgroundVerifiesWithoutCreating |
| C-reviewer-control | control | comment-only edit in socket.go | SURVIVED | — | (must survive; proves the battery observes outcomes) |

Denominator: 16 applied rows (12 narrowing, 3 additive, 1 error-arm) + 1 control. Of the 12
narrowings, 7 KILLED by the committed suite, 5 SURVIVED and are each proven real by a probe
kill. Every KILLED row was killed by exactly the test named above (kill lines in the raw logs).

## §P — reviewer probes (27, `probes-rev2.log`)

P01-P03 collision unclean/doubled/symlink spellings admitted (P3-C); P04-P06 FIFO leaf (Ensure,
Verify) and FIFO root refuse `runtime containment` within the deadline — no hang; P07 B11
confirmed; P08 setgid/sticky admitted; P09/P09b regular-file leaf refuses on Verify and on the
background `Acquire` (baselines for R01); P10/P11 both-empty generation pairs refuse (baselines
for R02/R03); P12 a request generation newer than the live broker's refuses
`capability_unavailable` carrying the request's generation; P13 foreground never probes the
broker; P14 backslash name refuses; P15 trailing-slash root refuses `runtime root`; P16
whitespace override refuses; P17 background foreign-ownership refuses through `Acquire` before
the broker is probed; P18 5000/100000-char remediation still renders the typed unavailable
(axerror admits them), invalid UTF-8 and NUL reach the constructor-failure arm and refuse
`tmux_invalid_arguments realm details` (fail-closed, never nil); P19 stranded 0600 leaf never
converges (B10); P20 read-only root refuses `runtime commit` (uncovered Mkdirat arm is
fail-closed); P21 broker hit carries the derived socket and no argv; P22 a superset Admitted
admits (Has semantics); P23 umask 0077 converges; P24 narrower leaf refuses on Verify; P25
uppercase UUID refused by the landed grammar (no canonicalization gap); P26 the SIGKILL child
leaves the mkdirat-committed leaf and the retry verifies it (crash test shape genuine).

## What held under attack (keep)

- Dedicated-server rule per vector: override refused on all five members through `Acquire`
  with no spawn and no directory; collision refused on all five (byte-identical); argv gate
  refuses all five; `Acquire` never reads the environment (no `Getenv`/`Environ`/`exec` in any
  production file; socket.go imports `path/filepath` only).
- Background rules: broker-or-refuse with the literal `capability_unavailable`, exit 6, and the
  five typed details asserted as literals; no fallback along the caller (N-creation-literal),
  platform (R14, my plant) and spawner (D-background-fallback) axes; verify-never-create on the
  authorizing path (R25, R07); same-user and generation-bound broker principal at the entry
  (R06).
- Attestation: landed `terminalbackend.Admitted` consumed through `Has`, composed with the
  generation equality axpane enforces; stale admissions refuse on both paths; decoy capability
  refuses; Windows refuses on every host.
- Custody: FIFO leaf/root and symlink leaf/root refuse on both entries without blocking; widened
  mode, foreign UID and handle mismatch refuse on both entries; the explicit `Fchmod` is
  load-bearing under umask 0177; the real-SIGKILL crash test is genuine (P26).
- No forks: `axpane` caller vocabulary, `axerror` typed refusal, `scalar` grammars, `secprim`
  Guard/OpenNoFollowDir/CheckArgv, `terminalbackend.Admitted` — all consumed; `handleBoundToPath`
  mirrors the private `secprim.guardRootMatches` and says so.
- Determinism: 0 SKIP with tmux unresolvable; no real-tmux witness (B1, honest).
- Boundary: `internal/traceability` untouched; README section carries no CLI/capability claim;
  no trunk drift.

## Coverage ratio measured against the 55-row table

**55 of 55 rows are driven as stated** — the ratio is honest, not inflated. The gaps above sit
outside the row statements: row 5 names Ensure only (P2-A), rows 36/38 keep one side non-empty
(P3-A), rows 1/4 claim "exact" while pinning widening only (P3-B), row 23 holds for byte-equal
values only (P3-C).

## Rework scope (for the producer)

1. **P2-A**: add `TestVerifyRuntimeDirRefusesNonDirectoryLeaf` (regular 0700 file; plus a FIFO
   leaf under a deadline so the no-hang property is pinned), drive the regular-file shape through
   the background `Acquire`, ship harness row `N-containment-odirectory-verify` (drop
   `O_DIRECTORY` on the verify `Openat` only, expect KILLED), and name both entries in row 5.
2. **P3-A**: one both-empty-generation row in `TestCheckServerAttestedRefusesEachMissingConjunct`
   and one in `TestCheckBrokerContactRefusesEachMissingFact`, or state the arms as
   entry-unreachable bounds.
3. **P3-B**: pin a pre-existing 0600 leaf as `runtime mode` on both entries, or restate the mode
   claim as "no group/world bits" in doc.go/README/TRACEABILITY/runtime.go.
4. **P3-C**: `filepath.Clean` both sides of `checkAmbientDisjoint` (with a test for `/./` and
   `//` spellings) or state the byte-equality bound; symlink aliases stay a bound either way.
5. **P3-D**: fold or supersede the rev1 LOGBOOK block so no landed entry describes
   `CheckLiveAttestation`, four vectors, the tripwire, 54/54, or 29 mutants.
6. **P3-E**: state the unmeasured stat-failure arms of `handleBoundToPath` and `verifyOwnership`
   as bounds (or add a seam and a row).
7. Rerun the package suite, `GOOS=windows GOARCH=amd64 go vet ./...`, the harness twice, and the
   30-command suite; refresh results/matrix/archive; hand off as revision 3. Everything in "What
   held under attack" needs no change — do not re-litigate it.
