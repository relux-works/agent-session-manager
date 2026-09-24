# TASK-260830-35urbp — review verdict, Change Request revision 3

**Verdict: CHANGES REQUESTED** (routed `to-dev`). No P1. Three P2 classes, five P3. Every rev2
finding (P2-A, P3-A..E, the three observations) is closed with driven evidence and stays closed;
what remains is two spec-text divergences the earlier rounds did not examine (the socket path and
the background return code for an absent runtime directory), one entry-level pin gap on the
foreground "running server never spawns" rule, and prose/bound hygiene. The rework is enumerated
at the end.

- Reviewer run: RUN-260921-2369f7 (claude-opus-5 max), reviewing `CR-TASK-260830-35urbp-3`
  revision 3 published by RUN-260921-ae96f3 (muse-spark max) after the rev2 rework brief.
- Reviewed bytes: base `799c338e401fca0b24859c0b870cd665927209f2` (= `origin/main` after a fresh
  `git fetch origin main`; `git diff --name-only 799c338 origin/main` empty — no trunk drift),
  candidate tree `eb7b660c80a01f2ec0e2c4a728751d775fa263c2`, patch resource sha256
  `c78d6635f5d77f995c418d3d450b5ee45e72340b7f4397cc2a40e18465a77132` (recomputed over the
  materialized resource, equal to the CR record). The live Story worktree's uncommitted tree was
  recomputed through a temporary index copy and equals the CR tree before and after this review.
  Every measurement below was taken on three `git archive` copies of that tree
  (`.temp/TASK-260830-35urbp/review-rev3/{candidate,mutant-copy,probe-copy}`), each re-hashed to
  `eb7b660c` through a temporary index before and after use. The live worktree, its index, branch
  and HEAD were not touched; no product edit, commit, checkpoint or integration was performed.
  `git status` in the live worktree is unchanged (`M LOGBOOK.md`, `M README.md`,
  `?? internal/tmuxserver/`).
- Normative authority: pinned `internal/specdoc/SPEC.v0.7.0.md` §4.2 (1403-1449), §4.3 (1458),
  §4.4 (1468-1488), with §4.C/§4.D/§4.E as context, **and** §3.2 (765-813) where it names the
  dedicated socket path and the runtime-directory custody rule this leaf implements. The task
  record's v0.5.0 Scope is stale; the candidate cites the pinned file only (no `0.5.0` anywhere in
  the package).
- Note on the rev3 reviewer brief: its templated sentence "revisions 1..2 carry no review verdict"
  is wrong for this task — rev1 (RUN-260921-0245e5) and rev2 (RUN-260921-5a9d4b) were both reviewed
  and routed CHANGES REQUESTED. Rev3 was reviewed on its own merits AND against closure of every
  rev2 finding.

## What was reproduced (all on tree eb7b660c; PATH without any tmux binary, TMUX/TMUX_TMPDIR unset, no tmux process on the host; umask 022, uid 502)

| Check | Result | Log (evidence archive) |
| --- | --- | --- |
| Configured suite, 30 commands read from `spawn.worktree_isolation.validation.commands` (count 30, verified by read) | **30/30 green, every command rerun by this reviewer**: 1 gofmt (exact configured form in the live worktree, exit 0; plus `gofmt -l` over every tracked `.go` in the CR tree, empty), 2 build, 3 vet, 26 linux build, 27 windows build (`static-gates.log`); 4 `go test ./... -count=1 -v`: 45/45 ok, 0 package FAIL, 0 top-level `--- FAIL`, 14 pre-existing SKIP outside this package, 21 indented `--- FAIL` lines are mutant-kill excerpts inside the passing resumesmoke `TestSmokeMutantsAreKilled` (`cmd04-test-v.full.log`, 2m34s); 5 race: 45 ok, 0 `DATA RACE`, 6m43s in one call (`cmd05-race.log`); 6 cover: 45/45 ok, tmuxserver **91.6%** (`cmd06-cover.log`); 7-23 seventeen fuzz seeds exit 0, 24 tracecheck exit 0 (`clauses_discharged=70/574`), 25 cataloggen -check exit 0, 28 JSON parse over the CR tree's 141 tracked JSON files (0 bad), 30 `git diff --check` in the live worktree exit 0 (`cmd07-25-28-30.log`); 29 `task-board validate` exit 0 with the board's 180 pre-existing issues (`cmd29-validate.log`) | `logs/` |
| Hosted-CI-shaped gate `GOOS=windows GOARCH=amd64 go vet ./...` (test files included) — the rev1 P1-A | **exit 0**; also `GOOS=windows go test -c ./internal/tmuxserver` exit 0, `GOOS=linux go vet ./internal/tmuxserver` exit 0 | `static-gates.log` |
| Package suite, tmux unresolvable | `go test ./internal/tmuxserver -count=1 -v`: 126 RUN / 126 PASS lines, **64 top-level PASS, 0 FAIL, 0 SKIP**. Determinism holds: no row's witness skips without tmux; no real-tmux witness exists (bound B1, honest) | `pkg-tests-notmux-v.log` |
| Producer harness, two passes | **48/48 both passes** (46 narrowing + D-background-fallback KILLED exit 1, C-control SURVIVED exit 0); verdict lists identical to each other and to the producer's attached pass 1; mutant copy re-hashed to `eb7b660c` after each pass; zero `__pycache__`/`.pyc` anywhere | `harness-pass1/`, `harness-pass2/`, `logs/harness-pass*-verdicts.log` |
| Producer evidence archive | real gzip; MANIFEST 121 entries, no self-entry, **121/121 sha256 verify**; one raw log per plant per pass with `exit:` line; producer pass1 == pass2 | (checked in place) |
| Reviewer battery, 11 rows × 2 passes | identical verdicts both passes; see §M | `reviewer-battery-pass1/`, `reviewer-battery-pass2/`, `reviewer_battery.py` |
| Reviewer probes, 12 tests | all behave as described in §P | `logs/probes-rev3.log`, `zz_review_probe_test.go` |
| Boundary / hygiene | `internal/traceability` untouched (`git diff --stat base tree -- internal/traceability` empty); no `0.5.0` citation; no `__pycache__`/`.pyc` in the CR tree or any copy; no `Getenv`/`Environ`/`os/exec` in any production file (socket.go imports `path/filepath` only); README section carries no `ax` command, `doctor` or capability claim; every one of the 58 test names in TRACEABILITY.md exists in the committed suite; row count 55 verified | `static-gates.log` |

## Closure of the rev2 findings (verified by driving, not by reading the results table)

| rev2 | Closed? | How verified |
| --- | --- | --- |
| P2-A verify-side non-directory leaf unpinned | **yes** | `TestVerifyRuntimeDirRefusesNonDirectoryLeaf` (regular 0700 file + FIFO under a 5s deadline) and `TestAcquireBackgroundRefusesNonDirectoryLeaf` are committed; shipped row `N-containment-odirectory-verify` drops `O_DIRECTORY` on the verify `Openat` only and is KILLED in both of my harness passes with the predicted signature (err nil on both entries, 5s FIFO stall); row 5 names both entries; the no-`O_NONBLOCK` decision is stated in row 5 and in the test comment |
| P3-A both-empty generation arms | **yes** | both-empty rows present in both tables; `N-realm-empty-want` and `N-broker-empty-generation` KILLED ×2 |
| P3-B "exactly 0700" one-sided | **yes** | `TestEnsureRuntimeDirRefusesNarrowerMode` (0600, 0500; chmod-set fixtures) on both entries; `N-mode-narrower-create`/`-verify` KILLED ×2; prose kept and now true |
| P3-C byte-equality collision | **yes** | `checkAmbientDisjoint` cleans the candidate; `/./` and `//` rows at `ResolveSocket` and through `Acquire`; `N-collision-unclean` KILLED ×2; symlink alias stated as B16 |
| P3-D LOGBOOK rev1 block | **yes** | the rev1 block is a SUPERSEDED pointer; no landed entry describes `CheckLiveAttestation`, four vectors, the tripwire, 29 mutants, 54/54 or 87.5% |
| P3-E prose-only stat arms | **yes** | B15 stated as error-arm rows |
| Obs-1/2/3 | **yes** | B9 import-census sentence, B17 (hosttrust precedent), B18 (`sun_path`, owner 1c28dz) present |

## P2 — must fix or record as a reviewed decision

### P2-A. The derived path is not the path the specification mandates

Pinned SPEC.v0.7.0.md §3.2 lines 806-813: "On macOS the tmux backend MUST use a dedicated AX
server selected by `tmux -S <runtime>/tmux/ax.sock`. The `<runtime>/tmux` parent MUST be
AX-created, owned by the current user, mode 0700, and verified component-by-component without
following a symlink." `<runtime>` is the Runtime IPC root of the §3.2 table (macOS: per-user
temporary directory; Linux/WSL2: `$XDG_RUNTIME_DIR/ax`), which the landed
`internal/localstore` already resolves as `PathRuntime` (`paths.go:332-366`).

The candidate hard-codes `RuntimeDirName = "ax-tmux"` (runtime.go:14) and
`SocketName = "ax-tmux.sock"` (socket.go:10), so `SocketPath(filepath.Join(root, RuntimeDirName))`
derives `<root>/ax-tmux/ax-tmux.sock` (probe P01: `/runtime/ax-tmux/ax-tmux.sock` against the
spec's `/runtime/tmux/ax.sock`). `Request.Root` is documented as "the AX state root"
(acquire.go:82-83, and "under the AX state root" at runtime.go:11-13), which names the durable
`PathState` row of the §3.2 table rather than the Runtime IPC row the spec places the socket
under — the socket "is runtime IPC, never durable identity" (SPEC 810) and a per-user temporary
root is what makes the post-reboot absence case in P2-B the normal cold state.
`TestSocketPathDerivesOnlyFromRuntimeDir` (tmuxserver_test.go:158-163) pins the literal
`"ax-tmux.sock"`, so the suite currently protects the divergence instead of detecting it
("check refusals against the normative source"). No planning note, story record or spec text
records `ax-tmux` as a decision; TRACEABILITY cites no §3.2 clause and states no bound.

§3.2 is outside the sections the producer brief named as authority, which is why this was not
raised in rev1/rev2; it is inside the spec text that governs exactly this leaf's deliverable, and
AGENTS.md makes the whole pinned specification the authority for product behavior. Landing
`ax-tmux.sock` now means the lifecycle leaf (1c28dz) builds its adapter on the wrong path and the
final leaf's registry re-pin binds §3.2 to code that contradicts it.

Fix (small): `RuntimeDirName = "tmux"`, `SocketName = "ax.sock"`; assert the literals `"tmux"`
and `"ax.sock"` in `TestSocketPathDerivesOnlyFromRuntimeDir` (and the joined spec form
`<root>/tmux/ax.sock` once); document `Request.Root` as the Runtime IPC root
(`localstore.PathRuntime`), not the state root; add a §3.2 row (path + custody clause) to
TRACEABILITY and the conformance matrix; update README/doc.go. If the orchestrator decides the
divergence is deliberate, that decision must be recorded as a stated bound naming the spec
clause — today it is neither implemented nor bounded.

### P2-B. A background caller with an absent runtime directory receives a custody code, not the `capability_unavailable` the spec mandates

SPEC §4.2 lines 1419-1423: a background CLI, SSH RPC process, daemon or restore worker "MAY
contact an already-running authenticated Aqua broker and its attested AX tmux server; if neither
exists it MUST return `capability_unavailable` with typed realm/readiness details and MUST NOT
fall back to direct server creation."

`acquireBackground` calls `VerifyRuntimeDir` before probing the broker (acquire.go:168) and
returns its refusal verbatim. When the runtime directory does not exist — no server, therefore
"neither exists" — the entry returns `tmux_unsafe_runtime_dir at runtime containment` and never
probes the broker (probe P02: `broker probed=false spawns=0`). With the Runtime IPC root being a
per-user temporary directory on macOS, this is the ordinary post-reboot state of every background
restore worker, not an edge. The committed test `TestAcquireBackgroundVerifiesWithoutCreating`
(acquire_test.go:277-295) pins that local code, so the suite defends the divergence; README
(§ "Private tmux server management") says background "otherwise returns the typed
`capability_unavailable` refusal", which P02 disproves; TRACEABILITY row 13/52 state "refuses
absence" without saying with which code.

What the rev2 review verified — verify-never-create on the authorizing path, no durable write,
no broker probe past a custody refusal — must stay. Only the classification of ABSENCE changes:
a missing leaf is a readiness fact (nothing to contact) and must surface as the typed
`capability_unavailable` with the request's realm/readiness details; a leaf that exists but fails
mode/ownership/kind/symlink custody is unsafe (SPEC 810-812) and keeps the custody code. That
requires the verify side to distinguish `ENOENT` from `ENOTDIR`/`ELOOP`/other on its `Openat`
(today all map to "runtime containment", see P3-B), so this finding and P3-B are one change.

Fix: on the background path map "leaf absent" to `backgroundUnavailable(req, verr)` (no create,
no probe needed — but probing the broker first and refusing the same way is also acceptable as
long as no spawn and no directory write occur); keep every other custody refusal as is; update
`TestAcquireBackgroundVerifiesWithoutCreating` to assert `capability_unavailable` + typed details
+ no directory + zero spawns, add a sibling test that a widened/foreign/non-directory leaf on the
background path still returns the custody code (the existing
`TestAcquireBackgroundStagesVerifyCustodyRefusal`/`...NonDirectoryLeaf` cover this), ship a
narrowing row (e.g. the absence arm returning the custody code again) and fix README/row 13/52.

### P2-C. The foreground "running server → attach-or-refuse, never spawn" rule is pinned at the entry only for running servers that carry some admission data

`acquireForeground` (acquire.go:210-215) attaches or refuses when `report.Running`, and spawns
otherwise. Every committed test that reports `Running: true`
(`TestAcquireForegroundAttachesToRunningAttested`, `...RefusesRunningUnattested` (decoy row),
`...RefusesWrongGenerationAdmission`, `TestAcquireForegroundSpawnIsIdempotentAcrossRetry`) carries
a non-empty `Admitted.Capabilities` AND a non-empty `RawGeneration`. The realistic post-spawn
state — the server runs, no evidence has been reconciled yet, `RealmAdmission{}` — is driven only
at the helper (`TestCheckServerAttestedRefusesEachMissingConjunct/no_admission_rows`), never
through `Acquire` (rejection pattern b). Two reviewer narrowings on the entry SURVIVED the
committed suite in both passes and are KILLED by my probes:

- `R01-running-empty-admission`: `if report.Running` → `if report.Running &&
  len(report.Admission.Admitted.Capabilities) > 0` — a running server with zero admission rows is
  treated as absent and `Acquire` spawns a second `new-session` into it (P07 baseline: production
  refuses `tmux_readiness_not_authorizing at server attestation`, spawn 0).
- `R02-running-empty-generation`: `if report.Running` → `if report.Running &&
  report.Admission.RawGeneration != ""` — same effect for a running server whose admission carries
  no generation (P08 baseline: production refuses `server generation`, spawn 0).

Production is correct today (P07/P08 pass on the candidate). The DoD line "refusing behavior
covered by negative tests that fail when the gate admits what it must reject, with the production
call site named" is not met for these two members at the entry the matrix names for rows 41/42.
Fix: commit the two entry-level tests (P07/P08 are templates: `Running: true` with
`RealmAdmission{}` → `server attestation`, spawn 0; `Running: true` with the realm row and empty
`RawGeneration` → `server generation`, spawn 0), ship `N-running-empty-admission` and
`N-running-empty-generation` as the R01/R02 narrowings, and name them under rows 41/42.

## P3 — fix in the same rework

- **P3-A. The TMUX-env collision member is measured on a shape tmux never produces.** tmux sets
  `TMUX=<socket>,<pid>,<index>`; the hostile fixture uses exactly that encoding
  (`/tmp/tmux-501/default,12345,0`), but the collision rows feed the bare derived path into
  `TMUXEnv`. A `TMUX` value naming the derived socket in the real encoding walks past
  `checkAmbientDisjoint` and `Acquire` spawns (probe P06: `via=spawned spawns=1`). Row 23's
  "colliding with the derived socket refuses on every member" therefore holds for the TMUX member
  only in an unrealistic spelling. Either compare the socket component (the substring before the
  first comma) for the `TMUXEnv` member with a test in the real encoding, or state the bound
  under B16 and reword row 23. Spec rule unaffected (the socket is derived, never selected).
- **P3-B. Absence and a failure to read are the same fact on the verify side.** `verifyRuntimeDir`
  maps `ENOENT` (absent leaf), `ENOTDIR`/`ELOOP` (unsafe leaf) and a missing root all to
  `runtime containment` (runtime_unix.go:97-108), while the commit side maps a missing root to
  `tmux_invalid_arguments at runtime root` (runtime_unix.go:28-29). Probe P03: verify-absent-leaf
  == verify-symlink-leaf byte for byte; missing root differs across Ensure/Verify. Distinguish
  absence (its own detail, consumed by P2-B) and align the missing-root mapping across the two
  entries (rows 10/13 then name the same code).
- **P3-C. Root custody is not gated and not bounded.** A world-writable (0777) root admits the
  leaf on `EnsureRuntimeDir`, `VerifyRuntimeDir` and the background broker path (probes P04, P11).
  SPEC 810-812 requires rejecting "a socket path, parent, or ancestor whose ... ownership, or
  permissions are unsafe" before bind/connect. B11 covers symlinked intermediates only; nothing
  states who verifies the root's own mode/ownership (the `localstore` layout owner, the lifecycle
  bind step, or this leaf). Likewise "AX-created" (SPEC 808) is not distinguishable after the
  fact — any same-user 0700 directory under any admitted `Name` is accepted as the runtime
  directory (probe P05 admits a pre-existing `tmux-501`). State both as explicit bounds with the
  named owner, or gate the root's mode/ownership on the pinned root handle.
- **P3-D. Prose that outruns the code in socket.go.** Lines 28-31 say both environment names
  "are validated by the landed environment-name grammar at the observation boundary, so a
  misspelled variable cannot smuggle an ambient value past the non-use proof" — that arm was
  deleted in rev2 (no `environ`/`secprim` import remains; probe P09 shows the lookup is called
  with the two constants and nothing is validated). `ObserveAmbient`'s `error` return is always
  nil since that deletion. Fix the comment and drop the vestigial return (or state what the
  return is for). Rejection pattern (f).
- **P3-E. Two refusal arms are pinned by detail in prose only.** `R04-mkdirat-eacces-as-exist`
  (a permission failure at `Mkdirat` treated as `EEXIST`) SURVIVED ×2: no committed test stages a
  read-only root, so the `runtime commit` detail at runtime_unix.go:41 is unmeasured (my P12 kills
  it: the refusal reroutes to `runtime containment` — still fail-closed, hence P3). `R07`
  (a two-space override admitted and ignored) SURVIVED ×2 — the override gate is pinned on the
  five hostile literals only; a whitespace-only override row closes it. Add the two rows or
  report the arms as bounds; neither admits an unsafe state.

## Observations (not findings; a sentence each in TRACEABILITY if the producer agrees)

- "No broker" is modelled as the zero `BrokerReport{}`, whose `Principal.UID` is 0 — a real UID.
  Under a root process the same-user arm passes for an absent broker and refusal comes from the
  generation arm (`R21-broker-uid-zero` is KILLED only by the helper-level detail pin; through
  `Acquire` both arms collapse into the same typed unavailable). Worth stating next to B14, or
  giving the report an explicit presence fact.
- `Hooks.AfterMkdir` also fires on the `EEXIST` path (created == false); harmless for the crash
  test, but the name promises a mkdir that did not happen.
- `R03-guard-resolve-failopen` (the `guard.Resolve` refusal arm at runtime.go:94-97 flipped to
  fail-open) SURVIVED ×2 — the name grammar refuses every input before the guard can; report it as
  an unreachable/error arm alongside B15 rather than as a gate.
- On the broker path the outcome carries the derived socket while `BrokerReport` carries no
  socket; the binding between "its attested AX tmux server" and the derived path is the generation
  equality only. Acceptable under the spec's generation model; worth one sentence under B13.

## §M — reviewer battery (11 rows, two passes, `reviewer_battery.py` in the evidence archive)

Every row: anchor count verified (all 11 applied, none NOT_APPLIED), plant applied to the mutant
copy, the committed suite run as shipped (full package, no `-run` mask), exit + raw output logged,
file restored from a byte copy and re-compared, tree re-hashed to `eb7b660c` after each pass.
Rows predicted SURVIVED were additionally run with `zz_review_probe_test.go` present and the
named killer selected (`*.with-probe.log`).

| Row | Kind | Gate (site) | Committed suite | With probe | Killer |
| --- | --- | --- | --- | --- | --- |
| R01-running-empty-admission | narrowing | `if report.Running` gains `&& len(Admitted.Capabilities) > 0` (acquire.go:210) | **SURVIVED** | KILLED | P07 |
| R02-running-empty-generation | narrowing | `if report.Running` gains `&& RawGeneration != ""` (acquire.go:210) | **SURVIVED** | KILLED | P08 |
| R03-guard-resolve-failopen | error-arm | `guard.Resolve` refusal → fail-open join (runtime.go:94-97) | **SURVIVED** | (unreachable through the name grammar) | — |
| R04-mkdirat-eacces-as-exist | narrowing | `!EEXIST` gains `&& !EACCES` (runtime_unix.go:40) | **SURVIVED** | KILLED (reroute to `runtime containment`) | P12 |
| R07-override-two-spaces | narrowing | `override != ""` gains `&& override != "  "` (socket.go:81) | **SURVIVED** | KILLED | P10 |
| R15-argv-detached-flag | positive-pin | drop `"-d"` from the spawn vector (argv.go:28) | KILLED | — | TestBuildArgvAddressesOnlyTheDerivedSocket |
| R17-realm-capability-swap | constant-swap | `realmCapability` → `"headless_creation"` (acquire.go:47) | KILLED | — | literal detail/fixture assertions (rejection pattern a holds) |
| R18-via-literal-swap | constant-swap | `ViaAttached` → `"attached"` (acquire.go:35) | KILLED | — | literal outcome assertions |
| R21-broker-uid-zero | narrowing | UID arm admits principal UID 0 (readiness.go:80) | KILLED | — | TestCheckBrokerContactRefusesEachMissingFact/no_broker (helper-level detail) |
| R26-background-absent-ensures | additive | background creates the directory when verify fails (acquire.go:168) | KILLED | — | TestAcquireBackgroundVerifiesWithoutCreating |
| C-reviewer-control | control | comment-only edit in acquire.go | SURVIVED | — | (must survive; proves the battery observes outcomes) |

Denominator: 11 applied rows = 5 narrowing + 1 additive + 1 error-arm + 1 positive-pin + 2
constant-swap + 1 control. Of the 5 narrowings, 1 KILLED by the committed suite (R21), 4 SURVIVED
and each is proven real by a probe kill (R01, R02, R04, R07). Every KILLED row was killed by
exactly the test named above (kill lines in the raw logs).

## §P — reviewer probes (12, `logs/probes-rev3.log`)

P01 derived `/runtime/ax-tmux/ax-tmux.sock` vs spec `/runtime/tmux/ax.sock` (P2-A); P02 background
+ absent runtime dir → `tmux_unsafe_runtime_dir at runtime containment`, broker not probed, 0
spawns (P2-B); P03 verify absent leaf == verify symlink leaf; missing root differs across
Ensure/Verify (P3-B); P04 0777 root admitted on Ensure and Verify (P3-C); P05 pre-existing
non-AX `tmux-501` directory admitted (P3-C bound); P06 `TMUX=<derived>,12345,0` → spawned (P3-A);
P07 running + `RealmAdmission{}` refuses `server attestation`, spawn 0 (baseline for R01); P08
running + realm row + empty `RawGeneration` refuses `server generation`, spawn 0 (baseline for
R02); P09 `ObserveAmbient` validates nothing (P3-D); P10 two-space override refuses (baseline for
R07); P11 background broker path admits under a 0777 root (P3-C); P12 read-only root refuses
`runtime commit` (baseline for R04).

## What held under attack (keep)

- Dedicated-server rule per vector: override refused on all five members through `Acquire` with
  no spawn and no directory; byte-identical and cleaned collisions refused on all five; argv gate
  refuses all five; `Acquire` never reads the environment or execs.
- Background rules: broker-or-refuse with the literal `capability_unavailable`, exit 6 and the five
  typed details asserted as literals for every miss with an existing directory; no fallback along
  the caller (N-creation-literal), platform (Linux miss), spawner (D-background-fallback) and
  directory-write (R26, R25/R07 of rev2) axes; same-user generation-bound broker principal at the
  entry.
- Attestation: landed `terminalbackend.Admitted` consumed through `Has` composed with the
  generation equality axpane enforces; stale, decoy and both-empty shapes refuse; the rev1 fork
  stays deleted; constant swaps (R17/R18) redden literal assertions.
- Custody: FIFO leaf refuses on both entries without blocking (5s deadline pinned), symlink leaf /
  root, widened and narrower modes, foreign UID and handle mismatch refuse on both entries; the
  explicit `Fchmod` is load-bearing under umask 0177; the real-SIGKILL crash test converges.
- Determinism: 0 SKIP with tmux unresolvable; no real-tmux witness (B1).
- Boundary: `internal/traceability` untouched; README section carries no CLI/capability claim; no
  trunk drift; producer evidence archive complete and digest-verified.

## Coverage ratio measured against the 55-row table

**55 of 55 rows are driven as their statements are written**; the ratio is not inflated. The gaps
above sit beside the statements: row 16 pins a socket literal the spec contradicts (P2-A), rows
13/52 state "refuses absence" without the code the spec mandates (P2-B), rows 41/42 are driven for
running servers with admission data only (P2-C), row 23 holds for the TMUX member only in a
non-tmux spelling (P3-A), and no row cites §3.2 (P2-A, P3-C).

## Rework scope (for the producer)

1. **P2-A**: `RuntimeDirName = "tmux"`, `SocketName = "ax.sock"`; assert the literals in
   `TestSocketPathDerivesOnlyFromRuntimeDir`; document `Request.Root` as the Runtime IPC root
   (`localstore.PathRuntime`); add a §3.2 row (SPEC 806-813) to TRACEABILITY and the matrix;
   update README/doc.go/LOGBOOK — or record the divergence as an explicit reviewed bound naming
   the clause (orchestrator decision).
2. **P2-B + P3-B**: distinguish `ENOENT` on the verify `Openat` (and align the missing-root code
   with the commit side); on the background path return the typed `capability_unavailable` for an
   absent leaf (no create, no spawn), keep the custody code for an unsafe leaf; rewrite
   `TestAcquireBackgroundVerifiesWithoutCreating` to assert the typed refusal + no directory + zero
   spawns; ship a narrowing row; fix README "otherwise returns" and rows 13/52.
3. **P2-C**: commit the two entry-level tests (running + `RealmAdmission{}` → `server attestation`;
   running + realm row + empty `RawGeneration` → `server generation`; spawn 0 in both) and ship
   `N-running-empty-admission` / `N-running-empty-generation`; name them under rows 41/42.
4. **P3-A**: compare the socket component of the `TMUXEnv` member (split at the first comma) with
   a real-encoding collision test, or state the bound and reword row 23.
5. **P3-C**: state root custody and "AX-created" as bounds with named owners, or gate the root's
   mode/ownership on the pinned handle.
6. **P3-D**: fix socket.go:28-31 and the vestigial `ObserveAmbient` error return.
7. **P3-E**: add a read-only-root row (`runtime commit`) and a whitespace-override row, or report
   both arms as bounds.
8. Rerun the package suite (tmux hidden), `GOOS=windows GOARCH=amd64 go vet ./...`, the harness
   twice, and the 30-command suite; refresh results/matrix/archive; hand off as revision 4.
   Everything in "What held under attack" needs no change — do not re-litigate it.
