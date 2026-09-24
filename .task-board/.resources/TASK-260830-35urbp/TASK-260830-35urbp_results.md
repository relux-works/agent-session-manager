# TASK-260830-35urbp results — implement-private-tmux-server-management (rev7)

Rework of revision 6 after review-verdict-rev6 (CHANGES REQUESTED: no
P1, one P2, two P3). First leaf of STORY-260830-2t4g7i
(production-tmux-backend, EPIC M1). The reviewer states every rev5
finding is closed with driven evidence and stays closed, production
behaves correctly at every point the review drove, the census counts
reproduce mechanically (24 × 11 = 264; 50 measured / 4 bounds / 210
unreachable), and per-entry kill attribution holds for all 50 measured
cells; this revision keeps all of that, ships no production change,
and closes only what the verdict grades open: one harness row, the
commit-side fixtures, the WSL2 miss member, the catalog-derived decoy
tables, the census-cell namings, the symmetry check, and the five
optional observations.

Candidate: UNCOMMITTED working tree on branch
`task-board/story/STORY-260830-2t4g7i` atop trunk `799c338` (trunk
unmoved since rev6; no refresh, no rebase, no commit, no merge by this
run). `git status` shows only `internal/tmuxserver/`, `README.md`,
`LOGBOOK.md`; `internal/traceability` untouched.

Normative source: the pinned `internal/specdoc/SPEC.v0.7.0.md`
§3.2 platform paths (lines 765-813: the dedicated socket
`<runtime>/tmux/ax.sock` under the Runtime IPC root) and §4.2 tmux
backend (lines 1403-1449), with §4.C (1082), §4.D (1220), and
§4.E (1309) as context, plus §4.3 (1458) and §4.4 for the Windows
and broker rules. All clause citations name the pinned file, never
the network, never v0.5.0.

## Finding-by-finding table (verdict -> change -> test that fails without it -> evidence)

| Finding | Change | Test that fails without it | Evidence path |
|---|---|---|---|
| P2-A commit-side O_DIRECTORY arm (runtime_unix.go:58-59) has no narrowing row: the R01 twin plant kills by reroute only (0600 fixture trips the mode gate), while a 0700 leaf is admitted and a FIFO hangs the creating path | Commit side gets the verify side's fixtures: 0700 regular-file leaf on `EnsureRuntimeDir` and through foreground `Acquire` (asserting no probe, no spawn), FIFO leaf on `EnsureRuntimeDir` under the shared `verifyWithDeadline` (now entry-named); shipped `N-containment-odirectory-commit`, named under row 5 and in the `commit × ENS` / `commit × AFG` census cells; row-5 FIFO clause reworded to both deadline tests; `N-containment-odirectory-verify` note corrected to point at the commit row | `TestEnsureRuntimeDirRefusesNonDirectoryLeaf/regular_file_0700` (`err = nil`), `.../fifo_without_blocking` (5s BLOCK), `TestAcquireForegroundRefusesNonDirectoryLeaf` (`err = nil`; a scratch probe proves the admitted entry probes once and spawns once into `<root>/tmux/ax.sock`) | `harness-pass1/N-containment-odirectory-commit.log` (+ pass 2), `logs/pkg-tests-notmux-v.log` |
| P3-A no-fallback miss arm pinned for macOS/Linux only: the R16 WSL2-guarded fallback spawn SURVIVES (no committed test reaches a WSL2 miss) | The Linux miss test is a platform table `{linux, wsl2}`; it already sits in `BG_MISS_LINUX`, so `N-creation-literal` and both D rows kill through all three tmux platforms; named under rows 29/31 | `TestAcquireBackgroundMissOnLinuxAlsoRefusesWithoutFallback/wsl2` (`spawn calls = 1, want 0`; the linux member stays green — member-level attribution) | `harness-pass1/D-background-fallback.log`, `D-background-fallback-foreground.log`, `N-creation-literal.log` (each fails the wsl2 subtest; + pass 2) |
| P3-B attestation membership pinned for one decoy of sixteen: the R07 `local_attach` narrowing SURVIVES (every committed decoy is `headless_creation`) | Decoy set derived from `catalog.Current().Capabilities` (Section 4.D, sixteen-value table asserted, realm row matched by spec literal and asserted present); every non-realm member alone and all fifteen together refuse at `CheckServerAttested` and through the background miss; `N-realm-membership` keeps its `headless_creation` member with its mask extended to the new tables; named under rows 27/35 | `TestCheckServerAttestedRefusesEveryCatalogDecoy` + `TestAcquireBackgroundRefusesEveryCatalogDecoy` (the R07 `local_attach` shape fails exactly the `local_attach` + `all_decoys_together` subtests at both levels; a second `durable_disconnect` variant replays the same attribution) | `harness-pass1/N-realm-membership.log` (kills on the `headless_creation` member + all-together at both new tables; + pass 2), `logs/pkg-tests-notmux-v.log` |
| Obs-2 crash-child exit-1 flake undiagnosable (stdout/stderr discarded) | Parent captures the child's combined output into both failure messages | — (diagnosability; the crash test still passes) | `crash_unix_test.go`, `logs/pkg-tests-notmux-v.log` |
| Obs-3 `TestBuildArgvRefusesInvalidArgvShape` unnamed in TRACEABILITY; matrix row 46 listed twice | B12 names the witness; the §4.C duplicate row merged into the §4.2 row 46 (matrix now 58 entries, numbers 1-58 unique, census mirrors TRACEABILITY byte-identically) | — (traceability hygiene; verified by script) | `TRACEABILITY.md` B12, matrix §4.C |
| Obs-4 `argv × AFG` cell omits the CheckArgv shape arm | Cell names the arm as B12 (unreachable through `Acquire`) | — (census accuracy) | `TRACEABILITY.md` + matrix census |
| Obs-5 `N-details-constructor` note misstates the mechanism ("zero outcome with no error") | Note corrected to the typed-nil `*axerror.Error` inside a non-nil error interface, with the top-level-assertion kill path; TRACEABILITY battery paragraph matches | The row still KILLED ×2 with the corrected note | `harness-pass1/N-details-constructor.log` (+ pass 2) |

Kept untouched per the verdict: the dedicated-server rule per vector,
the background broker-or-refuse wiring with the literal
`capability_unavailable` and typed details, the landed-admission
attestation composition, the custody gates that held under attack,
the determinism design (no real-tmux witness), and the
`internal/traceability` boundary. No production file changed in this
revision (test files, harness, TRACEABILITY, README, LOGBOOK only).

## Two-sided-gate symmetry check (the rework brief's instrument run)

For every gate implemented on two or more sides, the row sets side by
side — gate, sides, rows per side, symmetric or the named missing
row. Row existence, harness anchor counts, and pass-1 KILLED verdicts
are checked mechanically (`logs/census_check.py`, exit 0); the side
mapping is declared from the row find-sites:

| Gate | Sides (rows per side) | Verdict |
|---|---|---|
| mode exact-0700 | commit: N-mode-create, N-mode-narrower-create (2) / verify: N-mode-verify, N-mode-narrower-verify (2) | symmetric |
| O_DIRECTORY containment | commit: N-containment-odirectory-commit (1) / verify: N-containment-odirectory-verify (1) | symmetric (closed this round) |
| symlink-leaf containment | commit: N-containment-symlink (1) / verify: N-containment-symlink-verify (1) | symmetric |
| root no-follow | commit: N-root-symlink-commit (1) / verify: N-root-symlink-verify (1) | symmetric |
| session identity | entry: N-session-entry (1) / spawn-site: N-argv-session (1) | symmetric |
| caller dispatch | FG→BG: N-dispatch-foreground (1) / BG→FG: N-dispatch-background (1) | symmetric |
| nil dependencies | background: N-deps-background (1) / foreground: N-deps-foreground (1) | symmetric |
| probe passthrough | foreground: N-probe-swallow-foreground (1) / background: N-probe-swallow-background (1) | symmetric |
| custody wiring | foreground ensure-site: N-acquire-commit-refusal (1) / background verify-site: N-acquire-verify-refusal (1) | symmetric |
| leaf wiring | lexical: N-wiring-lexical (1) / ensure: N-wiring-ensure (1) / verify: N-wiring-verify (1) | symmetric |
| typed details (single site) | 3/3 members rowed (generation, brokerstate, remediation) | member-complete |
| absence classifier (single site + remap) | 5/5 members rowed (macOS remap, mode, ownership, containment, invalid) | member-complete |
| running admission (single site) | 2/2 members rowed (empty admission, empty generation) | member-complete |

Single-implementation gates (root grammar, root classifier,
ownership, handle binding, name, platform, override, collision, realm,
broker, creation, argv shape, spawn, render) are symmetric by
construction — one site, their rows kill through every entry the
census names — and the three single-sided arms (explicit chmod,
commit EACCES, foreground unattested wiring) have no twin by
construction (verify never creates, background never attaches). No
asymmetric gate remains; the O_DIRECTORY commit side was the one this
round closed.

## Census (refreshed, counts unchanged)

24 gates × 11 entries = 264 cells: 50 measured (M), 4 bounds (B, all
in the guard gate's unreachable arm), 210 unreachable with reasons
(77 U1 + 133 U2) — verified mechanically from the table source, with
every named test existing in the committed suite (82 named, 82
defined, zero unnamed) and every named row existing in the harness
(`logs/census_check.py`, exit 0). No new M cell was needed: the
affected cells were already measured, and now name the new row and
tests (`commit × ENS/AFG`, `creation × ABG`, `server × ABG/SRV`,
`argv × AFG`). Per-entry kill attribution re-verified from the raw
logs for every row this revision touches: the commit row fails the
foreground entry test plus all three commit subtests (0700 admit,
FIFO 5s block, 0600 reroute); both D rows and `N-creation-literal`
fail the linux AND wsl2 miss subtests; `N-realm-membership` fails the
fixture tables plus both decoy tables on `headless_creation` and the
all-together set.

## Deliverable (rev7 shape)

`internal/tmuxserver`, composing the landed owners (never forked) —
production imports are exactly `errors`, `io/fs`, `os`,
`path/filepath`, `strings`, `syscall`, `x/sys/unix`, `axerror`,
`axpane`, `scalar`, `secprim`, `terminalbackend` (no `os/exec`, no
environment access, no `catalog`: the catalog derivation is
test-only):

- (1) Runtime custody: `EnsureRuntimeDir` / `VerifyRuntimeDir` with
  three independent gates (containment, exact `0700`, eUID ownership),
  the O_DIRECTORY arm now rowed on both sides with admit-direction
  and no-hang fixtures. Foreground acquisition ensures idempotently;
  background acquisition verifies the existing directory and never
  creates it.
- (2) Dedicated socket: `SocketPath` / `ResolveSocket` derive
  `<root>/tmux/ax.sock` and refuse overrides and ambient collisions
  on all five members plus whitespace, on both callers and at
  `ResolveSocket` directly (unchanged).
- (3) Acquisition: `Acquire` dispatches by caller, refuses malformed
  input before any side effect, and the background miss refuses
  without fallback on macOS, Linux, AND WSL2 — the no-fallback proof
  (creation-gate call, 24/24 both-spy census, both D rows) now spans
  all three tmux platforms.
- (4) Readiness: `CheckServerAttested` / `CheckBrokerContact` decide
  the landed admission over a typed principal; the membership arm is
  now pinned for the full §4.D vocabulary (every non-realm catalog
  member alone and together) at the helper and through the background
  entry, with `N-realm-membership` keeping its fixture member.
- (5) Spawn argv: `BuildArgv` emits exactly
  `tmux -S socket new-session -d -s session ax pane session`
  through the landed argv gate (unchanged).

Extended vs deliberately not extended (unchanged from rev6, plus the
test-only catalog derivation): `terminalbackend.Admitted`,
`axpane` caller vocabulary, `axerror` typed refusal, `scalar`
grammars, `secprim` Guard / `OpenNoFollowDir` / `CheckArgv`, the
`localstore` Runtime IPC root (consumed as `Request.Root`), the
`hosttrust` verify-never-repair precedent, the `termbind`
crash-hook pattern, and now `catalog.Current().Capabilities` as the
test-only §4.D vocabulary source. Deliberately not extended:
`hosttrust` custody itself, `terminalbackend` Reconcile evidence
admission (consumed, not re-owned), `axpane` checkRealm (unexported),
the host-binding/provider-build cross-bind (B13), `terminstance`
lifecycle execution (sibling leaf), `secprim`'s private
`guardRootMatches` (mirrored with a seam), `internal/traceability`
(untouched).

Determinism: every broker, server, and spawner interaction runs
through injected dependencies. No test starts a tmux process; the
suite passes with the tmux binary unresolvable
(`PATH={go,gofmt,python3 symlinks}:/usr/bin:/bin:/usr/sbin:/sbin`),
`TMUX`/`TMUX_TMPDIR` unset, and no tmux process on the host (230
PASS — 82 top-level, 148 nested — 0 SKIP in the tmux-hidden run,
with the env proof in the log). No AC row has a real-tmux witness —
that is the design, not a gap (bound B1).

## Coverage ratio (measured, not planned)

**58 of 58 AC rows driven** through production entries by named
committed tests — matrix in `internal/tmuxserver/TRACEABILITY.md`,
reproduced with spec-clause mapping in
`TASK-260830-35urbp_conformance-matrix.md` (rev7). Row 5 names the
commit-side fixtures, the foreground-entry test, and the new row;
row 29 names the `{linux, wsl2}` table; row 31 names the 24-test
both-spy census and both miss tables as the N/D kill paths; rows
27/35 name the catalog-decoy tables; the §4.C duplicate row 46 is
merged (58 entries, numbers 1-58 unique); the mutant map carries the
new row and the extended mask. The gate × entry census (50 of 264
cells measured, 4 bounds, 210 unreachable with reasons) is the
matrix's own table, mirrored in TRACEABILITY.md byte-identically
(verified by script).

## Mutants (shipped harness, per-plant raw logs)

`PYTHONDONTWRITEBYTECODE=1 python3 internal/tmuxserver/mutant_harness.py` —
73 narrowing + 2 supplementary additive (D-background-fallback,
D-background-fallback-foreground, labeled) + 1 harmless SURVIVED
control. Pass 1 and pass 2: **76/76 each, verdict lists identical**,
exit 0. Per-row raw logs (go output + subprocess exit) for both passes
are in the evidence tarball; production blobs byte-identical
before/after every run (`git status` shows only candidate paths). No
token-preserving source-text mutant applies: no gate inspects source
text (stated bound B4, harness executes the behavioral suite). The one
nil-dependency narrowing that kills by panic (N-deps-foreground) is
disclosed in the harness and the matrix; N-deps-background kills by
an unavailable-vs-dependencies mismatch since the admitted request
reaches Verify on the absent leaf. N-fchmod-umask and the
read-only-root test assume a non-root test user (bound B14).
N-broker-empty-generation, N-commit-eacces-as-exist, and the
background leg of N-verify-absence-detail kill by reroute to a
neighboring refusal, disclosed in their row notes; the 0600 commit
subtest under N-containment-odirectory-commit fails by reroute while
the row's attribution rests on the 0700 admit and the FIFO block.
N-details-constructor kills by typed-nil (corrected note).
N-session-entry kills through the side-effect assertions while the
refusal code stays correct. The reviewers' error-arm plants are
reported as error-arm rows under bound B15 and are never counted
among the narrowing rows.

## Crash / idempotency (test-harness touch only)

Real SIGKILL seam (unix): `TestEnsureCrashChildSelfTerminates`
(child runs the genuine `EnsureRuntimeDir` entry and kills itself in
`AfterMkdir`; parent proves the retry converges). This revision
captures the child's combined output into the failure message
(obs-2). Idempotent replay: `TestEnsureRuntimeDirIsIdempotent`,
`TestAcquireCreatesRuntimeDirIdempotently`,
`TestAcquireForegroundSpawnIsIdempotentAcrossRetry`. Crash
convergence states the umask bound B10; the server socket's crash
evidence is owned by the lifecycle leaf (bound B9).

## Validation (all run firsthand in this session)

Configured suite read from `spawn.worktree_isolation.validation.commands`
(count 30, verified by read). All 30 green, run by this run (nothing
accepted from rev6 or from construction):

- 1 gofmt clean (empty file list); 2 `go build ./...` exit 0;
  3 `go vet ./...` exit 0.
- 4 `go test ./... -count=1 -v` exit 0: 45/45 packages ok, 0
  column-0 FAIL, 21 indented excerpt FAILs inside the passing
  `TestSmokeMutantsAreKilled` / `TestCensusLiveEventOwnershipPlants`,
  14 pre-existing nested SKIP all outside this package.
- 5 race gate exit 0 with no DATA RACE: run in 6 bounded groups
  (8/8/8/8/8/5 packages = all 45 from `go list ./...`) because one
  shell call is time-bounded; same flags otherwise
  (`-race -count=1 -timeout 25m`). 45/45 ok. Host load 11-16 during
  the run (a second Story producing concurrently, as briefed); no
  gate timed out.
- 6 `go test ./... -cover -count=1` exit 0: 45/45 ok.
  `go test ./internal/tmuxserver/ -cover` 92.7% statements.
- 7-23 all 17 fuzz seeds exit 0. 24 tracecheck exit 0 (70/574).
  25 cataloggen -check exit 0. 26 linux build exit 0. 27 windows build
  exit 0. 28 JSON parse exit 0. 29 `task-board validate` exit 0 (with
  the board's pre-existing 180 issues). 30 `git diff --check` exit 0.
- Repository gates beyond the 30: `GOOS=windows GOARCH=amd64 go vet
  ./...` exit 0, `GOOS=windows go test -c ./internal/tmuxserver`
  exit 0, `GOOS=linux go vet ./internal/tmuxserver` exit 0 (+
  test-compile), `go vet ./internal/tmuxserver` exit 0, gofmt empty;
  package suite 230 RUN / 230 PASS (82 top-level) / 0 SKIP;
  tmux-hidden suite (tmux unresolvable, `TMUX`/`TMUX_TMPDIR` unset, no
  tmux process) exit 0 with 230 PASS and 0 SKIP.

Reran myself: everything above, plus the mutant battery (targeted
subsets during development + two full logged passes), the R01/R07/R16
replay plants with member-level attribution, the census/symmetry
scripts, and the 24/24 spy recount. Accepted from already-attached
evidence: nothing — rev6 evidence is superseded by this revision.

`internal/traceability` untouched; README section updated (73-row
battery count, 24/24 census sentence) with exact test commands and
no CLI/capability claim; LOGBOOK rev7 entry added newest-first. The
candidate is UNCOMMITTED in the Story worktree.

## Rejection-pattern self-check

(a) Literals asserted from spec text at entries (`capability_unavailable`
plus all five typed details, exit 6; `"tmux"`/`"ax.sock"` plus the
joined `<root>/tmux/ax.sock` at `SocketPath` AND through `Acquire` on
both callers; `ax`/`pane`/UUID session tail; the full spawn vector
element-by-element at `BuildArgv` AND through `Acquire`; `0700`
on both the widened and narrower sides; caller vocabulary;
`credential_capable_execution_realm` as the admitted member; the
sixteen-value §4.D table size and the realm literal in the decoy
derivation, matched by literal, never the production constant).
(b) Every rule driven through the entry the matrix names — the
commit-side O_DIRECTORY arm now measured at `EnsureRuntimeDir` AND
through foreground `Acquire` with its own narrowing row; the WSL2
miss member driven through `Acquire`; every decoy driven at
`CheckServerAttested` AND through the background miss; the census
cells name the new rows and tests. (c) Landed owners composed
(listed above, now including the test-only catalog derivation) —
only package-specific rules are new. (d) Inputs driven, effects
asserted (admit-direction nils, 5s FIFO blocks, probe/spawn call
counts on both dependencies, per-member subtest attribution for the
R01/R07/R16 replays); the census and symmetry tables map
measurement, they substitute for none — every M cell carries a test
that fails without the property, and every symmetry row is backed by
a KILLED verdict. (e) 73 narrowing rows + 2 labeled additives +
applied SURVIVED control, one raw log per plant per run,
narrowing-vs-additive stated per row; per-entry kill attribution
verified from the raw logs for every touched row; error-arm rows
reported separately, never counted as narrowings. (f) Every
"never/composes" sentence is backed by a failing-without test or
stated as an explicit bound (B1-B21); the universal spy sentence is a
mechanical 24/24 census; the completeness claim for the decoy set is
stated only where the catalog derivation proves it (table size +
realm presence asserted, derivation fails loudly on drift).
