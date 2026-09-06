# TASK-260830-2ciy0s — safe path/argv/environment primitives (outcome, rev3)

Round 3. Closes all four blocking findings of
`TASK-260830-2ciy0s_review-verdict-rev2.md` (C1–C4). Work left
UNCOMMITTED in the story worktree on top of trunk `157a54c`; no commits
made, no board elements created.

Round-2 substance is preserved, not re-litigated: the openat walk stays
relative at every unix step (no `filepath.Join` in
`open_member_unix.go`), the 15 grammar shapes refuse, the TOCTOU swap
refuses from the commit step, the FIFO refuses through both openers,
all six `cliresult` sites die to narrowing, `Env` nil-versus-empty dies
both ways, the escape corpus is fed from the slice it asserts, the
archive scan is recursive with a derived count and measured fail-closed
arms. What changed is below.

Companion files (attached alongside this document): the rev3 mutant
battery script, the battery result log, the full-suite log, and the
race log.

## C1 — the guard root is bound to the handle (was: discarded)

`openCommitRelative` took `guard.root` and threw it away; a foreign
directory handle was accepted and read through (`guard.Open` admitted
and returned `FOREIGN` bytes in the reviewer's probe), and the two
platforms disagreed (Windows descends from `rootPath`, unix from the
handle alone).

Fix (`internal/secprim/nofollow.go`): `Guard.Open` binds before
descending. `guardRootMatches` fstats the handle and compares device
and inode against a stat of `guard.root` via `os.SameFile` (dev/ino on
unix, volume/file index on Windows); any stat failure is not a match,
so an unverifiable root fails closed. Mismatch refuses as the new
`guard root mismatch` rule. The check lives in shared code before
either walk runs, so unix and Windows answer identically by
construction; past it the unix walk descends from the verified handle
and the Windows walk from the verified root path, which name the same
directory (`open_member_unix.go` and `open_member_windows.go` document
this; the unix parameter is now named instead of discarded).

Driven by (both run on every platform — no build tag):

- `TestGuardOpenRefusesForeignRootHandle`
  (`internal/secprim/guard_root_test.go`): guard on `stage`, handle on
  `elsewhere` holding `secret`=`FOREIGN` → refuses with the full
  string `secprim unsafe path: guard root mismatch: secret`, and the
  test fails outright if any bytes are admitted.
- `TestGuardOpenAdmitsMatchedRootHandle`: the handle opened on the
  guard root still commits and reads the staged bytes (the binding is
  not too tight).

Live verification this session: with the binding removed the foreign
test fails with `foreign handle admitted; read "FOREIGN" outside the
guard root`; with it, the test passes (removal and restore were
byte-identical via `cmp`).

## C2 — classify before close, depth-3/4 vectors (was: use-after-close)

From the second walk iteration onward `previous == current`, so the
error branch closed the parent descriptor and then `classifyCommitErr`
fstat'd the recycled number: `l1/hop/loot` and `a/b/hop/loot` refused
as `open failed: ... not a directory` instead of `member symlink
escape` (confirmed on this host, where `O_NOFOLLOW|O_DIRECTORY` on a
symlink yields `ENOTDIR`, so the `fstatat` re-examination is the only
thing naming the shape).

Fix (`internal/secprim/open_member_unix.go`): both error sites (loop
and final) classify first and close after; the ordering invariant is
documented on the function.

Driven by `TestGuardOpenRefusesDeepSymlinkedParent`
(`internal/secprim/open_unix_test.go`): symlinks on non-first
intermediates in `l1/hop/loot` (3 segments) and `a/b/hop/loot` (4
segments), asserting the FULL refusal strings
`secprim unsafe path: member symlink escape: <member>` — a
misclassified rule reddens even though every branch still refuses.

Live verification this session: with the loop site reverted to
close-before-classify, the new test fails with
`refusal = "secprim unsafe path: open failed: l1/hop/loot: not a
directory", want "secprim unsafe path: member symlink escape:
l1/hop/loot"` — the reviewer's C2 table reproduced exactly; with the
fix, it passes (byte-identical restore).

The README sentence is now true at every depth (see README diff), and
`OpenNoFollowFile`'s doc no longer recommends the uncontained
`Resolve`+`OpenNoFollowFile` composition — it points at `Guard.Open`.

## C3 — the identifier-spelling bound is stated (was: four spellings
claimed as the class)

Second round on the rev1/B5 class. Taken the reviewer's preferred
option: state the bound. `errors.go`, the `deriveRefusals` comment,
and the README now say the gate matches the `Error` identifier
spelling (`isErrorLiteral` on `Ident{"Error"}`, `returnsError` on
`*Error`), and that a refusal type spelled through a type alias
(`type X = Error`), a defined type (`type X Error`, even converted
back to `*Error`), or an alias-spelled factory result is outside the
witness — by construction (syntax-only derivation, no `go/types`),
not by oversight.

Executable pin: `TestRefusalInventorySpellingBoundIsStated` feeds H
(`type shadowAlias = Error` + `&shadowAlias{...}`), I
(`type shadowNamed Error` + `(*Error)(&shadowNamed{...})`), and J
(`type shadowResult = Error` + factory returning `*shadowResult`)
through the production `deriveRefusals` and requires silence (no
site, no stray) for all three. A future spelling cannot re-claim the
class without touching that test plus `errors.go` plus the README.

Live acceptance this session (each as a real production file, package
suite per plant, reverted after, no residue):

| Plant | Shape | Result |
| --- | --- | ---: |
| CONTROL | bare unexercised `failArgv` | CAUGHT, exit 1: `refusal site plant_control_rv.go:4 never fired during the suite` |
| H | `type shadowAlias = Error` + literal | BYPASSED, exit 0 (stated bound) |
| I | `type shadowNamed Error` + conversion | BYPASSED, exit 0 (stated bound) |
| J | alias-spelled factory result | BYPASSED, exit 0 (stated bound) |
| shell | token-preserving `os/exec` (`"s"+"h"`) | CAUGHT by `TestPackageLaunchesNoProcess`, exit 1: `plant_shell_probe.go imports os/exec` |

## C4 — N24 restored, D14 reported honestly, evidence as files

N24 is KILLED and back in the battery: dropping the directory arm
alone (`CheckRegularTarget`) kills `TestCheckRegularTarget`, the
refusal moving from `target directory` to `target special file`
(reproduced live this session). Count corrected below.

The scrutiny C4 demands, applied to the withdrawn spelling: the
`Size() > 0` variant of N24 (`case info.IsDir() && info.Size() > 0`)
was run and SURVIVES (exit 0) — empty directories report `Size() > 0`
on this host's filesystems, so that spelling is vacuous, not
admitting. The rev2 withdrawal confused the two spellings: the vacuous
spelling does not die, the arm-drop spelling does. Both results are in
the battery log; only the killing spelling counts as a kill.

D14 honesty: the scripted D14 (delete the runner argv-gate block) is a
COMPILE-FAIL (`secprim` imported and not used), not a test kill — it
proves the import is used, nothing about the gate. It stays in the log
as `KILLED-OTHER`/compile-fail and does NOT count as a kill. D14b
(deadens the gate with `&& false`, import kept) kills
`TestExecRunnerRefusesUnusableExecutable` with
`Run("") = *errors.errorString exec: no command, want *secprim.Error`
and counts instead.

This document and its companions are attached as FILE resources, not
inline content.

## Denominator — 42 refusal exits, exact

Production `deriveRefusals` over the candidate's non-test sources (12
files): argv 5, env 9, nofollow 15 (rev2's 14 + the new `guard root
mismatch` site), path 13 — TOTAL 42, STRAYS 0. Distinct refusal rules:
29 (5 argv + 6 env + 18 path/containment, with `member unmanaged`,
`target directory`, `target symlink` shared between `Resolve`/`Open`
and the stat gate).

## AC-row coverage — 31 of 31 rows driven, call site per row

Every row is driven through the production entry point by the named
committed test; the inventory audit additionally proves every site
fired during the suite (an unfired site fails the build). Mutant
column: N = narrowing kill, D = arm-deletion kill, probe = verified by
a dedicated probe rather than a mutant, fired-only = exercised with no
admitting mutant (reason stated; these are propagation, type, or
delegation sites with no input class to admit).

| # | Rule (production call site) | Driving test (production entry) | Mutant |
| --- | --- | --- | --- |
| 1 | argv empty (`argv.go:30` `CheckArgv`) | `TestCheckArgvRefuses/empty_vector` | N25 |
| 2 | argv executable empty (`argv.go:37`) | `TestCheckArgvRefuses/empty_executable` | N39 |
| 3 | argv element empty (`argv.go:39`) | `TestCheckArgvRefuses/empty_element` | N14 |
| 4 | argv NUL (`argv.go:41`) | `TestCheckArgvRefuses/nul_argument`, `nul_executable` | N12 |
| 5 | argv encoding (`argv.go:43`) | `TestCheckArgvRefuses/invalid_utf8` | N13 |
| 6 | env lookup missing (`env.go:60` `BuildEnv`) | `TestBuildEnvRefuses/nil_lookup` | fired-only: nil reader is a programming error, no input class to admit |
| 7 | env name (`env.go:65,74`) | `TestIsEnvName`; `.../bad_allowed_name`, `bad_literal_key` | N15, N16 |
| 8 | env allowlist duplicate (`env.go:68`) | `TestBuildEnvRefuses/duplicate_allowed` | N34, D1 |
| 9 | env literal collision (`env.go:77`) | `TestBuildEnvRefuses/literal_collides` | N35, D2 |
| 10 | env value NUL, allowlist (`env.go:87`) | `TestBuildEnvRefuses/inherited_NUL` | N30 |
| 11 | env value NUL, literals (`env.go:96`) | `TestBuildEnvRefuses/literal_NUL` | N31 |
| 12 | env value encoding, allowlist (`env.go:90`) | `TestBuildEnvRefuses/inherited_encoding` | N32 |
| 13 | env value encoding, literals (`env.go:99`) | `TestBuildEnvRefuses/literal_encoding` | N33 |
| 14 | member grammar (`path.go:31` `CheckMemberPath`) | `TestCheckMemberPathRefuses` (15 shapes) | N36 |
| 15 | member encoded dot (`path.go:34`) | `.../encoded_dot`, `double-encoded_dot` | N1, D6 |
| 16 | member overlong separator (`path.go:37`) | `.../overlong_separator`, `double-encoded_overlong` | N2 |
| 17 | member alternate stream (`path.go:42`) | `.../windows_ads`, `windows_ads_nested` | N3 |
| 18 | member trailing dot or space (`path.go:45`) | `.../windows_trailing_dot`, `windows_trailing_space` | N38 (dot), N4 (space) |
| 19 | member reserved device name (`path.go:48`) | `.../windows_con`, `windows_lpt`, `...` | N5 |
| 20 | guard root (`path.go:115` `NewGuard`) | `TestGuardResolve` (relative root refused) | D18, D21 (deletion; delegation to scalar, absoluteness has no narrower admittable spelling) |
| 21 | member unmanaged (`path.go:146`, `nofollow.go:116`) | `TestGuardManagedSet`; commit-shapes `unmanaged_member` | N6, D5 |
| 22 | member containment (`path.go:150` `Resolve`) | `TestGuardPrefixCheckIsLoadBearing` | probe: defensive conjunct, unreachable through the entry — R12 double probe below, no mutant can admit through the entry |
| 23 | guard root handle (`nofollow.go:119,122` `Guard.Open`) | commit-shapes `nil_root_handle`, `file_root_handle` | fired-only: nil/non-dir handles are type errors, no admittable class |
| 24 | guard root mismatch (`nofollow.go:125`) | `TestGuardOpenRefusesForeignRootHandle` | N28, D20 |
| 25 | member symlink escape (`nofollow.go:131`) | symlinked-parent, deep-symlinked-parent, `final_symlink` | N23, N29, D12, D13 |
| 26 | open failed (`nofollow.go:60,133,167,175`) | missing member/intermediate, unreadable dir, missing file/dir | D19 (one arm); rest fired-only: errno propagation, no class boundary to move |
| 27 | open stat failed (`nofollow.go:65,138,180`) | `TestOpenStatFailureSeam` (forced fstat failure) | fired-only: seam-driven propagation; D17 covers the adjacent nil arm |
| 28 | target directory (`path.go:224`, `nofollow.go:171,188` `CheckRegularTarget`) | `TestCheckRegularTarget` (+ empty dir), `TestOpenNoFollowFile` | N24, N24b |
| 29 | target symlink (`path.go:222`, `nofollow.go:169`) | `TestCheckRegularTarget`, symlink opener tests | N37 |
| 30 | target special file (`path.go:230`) | `TestCheckRegularTarget` (fifo, null) | N7b (N7 survives with bound) |
| 31 | target stat missing (`path.go:217`) | `TestOpenStatFailureSeam` (nil result) | D17 |

Ratio: 31 of 31 AC rows driven through production entries.
Narrowing cover: 22 of 29 distinct rules carry a killing narrowing
mutant (rows 1–5, 7–19, 21, 24, 25, 28–30, counting shared-rule rows
once); row 22 is verified by the R12 double probe instead of a
mutant; rows 6, 20, 23, 26, 27, 31 are stated bounds (each reason in
the table). No row rests on prose.

Positive (admission) controls, same entries: honest-member commit
read, matched-handle commit read, `TestCheckArgvAdmits`,
`TestCheckMemberPathAdmits`, `TestBuildEnvAdmitsEmptyValues`,
nil-Env-inherits vs empty-Env-denies-all (`provhost`), success-path
rendering (`cliresult`).

## Mutant battery — 59 killed of 61 applied, kinds counted separately

Script: `TASK-260830-2ciy0s_mutant-battery-rev3.py` (rev2's 46 +
N24-restored-as-arm-drop, N28, N29, N30–N35, N36, N37, N38, N39,
D14b, D20, D21). Each mutant is applied to the working tree with an
exact one-occurrence anchor, run as a standalone process
(`go test <pkg> -count=1 -run <test>`), and restored byte-identical;
the log is `TASK-260830-2ciy0s_mutant-battery-rev3.log`.

- Narrowing, 41 applied, 40 killed: N1–N39 (39 ids, of which only
  N7 survives) plus N7b and N24b. Every kill names the admitted
  member and the failing test in the log; new in rev3 are N24
  (directory arm-drop, exact-message kill), N28 (foreign handles),
  N29 (misreported rule), N30–N35 (per-arm env narrowings), N36
  (parent segments), N37 (symlink equality), N38 (trailing dot), N39
  (empty executable).
- Arm deletion, 18 applied, 17 killed: D1–D3, D5, D6, D10–D13, D14b,
  D15–D21. Scripted D14 is the 18th: COMPILE-FAIL (`secprim` imported
  and not used), superseded by D14b — counted as applied, not as a
  kill. New in rev3 are D14b (argv gate, import kept), D20 (binding
  arm), D21 (root delegation).
- Instrument probes, 2 applied, 2 killed: D8 (census token drop),
  D9 (class-row rename).
- TOTAL: 59 killed of 61 applied. The two non-kills are N7
  (SURVIVES, bound below) and D14 (compile-fail, superseded by D14b).
  The vacuous `Size() > 0` spelling of N24 was run separately and
  SURVIVES (exit 0) — withdrawn as vacuous with that evidence, not
  counted in either column.

Survivors with stated bounds — 1:

- N7 (drop the `ModeNamedPipe` disjunct alone): SURVIVES. Bound
  correct: `CheckRegularTarget` ends in `!mode.IsRegular()`, which
  subsumes the device, named-pipe, socket, and char-device disjuncts —
  any one disjunct alone is unobservable. The two-arm N7b (drop
  NamedPipe disjunct plus the catch-all) admits FIFOs and is KILLED by
  `TestCheckRegularTarget`.

R12 (drop the slash boundary from `withinRoot`): verified by double
probe on this tree, not by battery mutant (the conjunct is defensive
and unreachable through the entry):

- shared grammar weakened (`scalar.ParseRelativePath` admits `..`),
  `withinRoot` intact → `TestGuardPrefixCheckIsLoadBearing` PASSES
  (`prefix property holds over 60 admitted of 512 generated members`):
  the prefix conjunct catches the escape — that is what defense in
  depth is for;
- grammar weakened AND `withinRoot` neutralized (`return true`) →
  FAILS (`Resolve("a/../..") escaped to "/stage"`, …): the tripwire
  reddens exactly when both conjuncts are gone;
- both reverted byte-identical (`cmp` clean), tripwire green again.

## Windows — escape-capable set established, universal withdrawn

Checked against the toolchain in use (go1.25.5,
`$GOROOT/src/os/types_windows.go`, `(*fileStat).mode()` and
`modePreGo1_23`), in exactly this shape:

| Reparse tag | Go maps to | Gate outcome |
| --- | --- | --- |
| `IO_REPARSE_TAG_SYMLINK` | `ModeSymlink` | refused |
| junction / `MOUNT_POINT`, AppExecLink, cloud placeholder (default arm) | `ModeIrregular` | refused |
| `MOUNT_POINT` under `GODEBUG=winsymlink=0` | `ModeSymlink` | refused (both bits tested) |
| `IO_REPARSE_TAG_AF_UNIX` | `ModeSocket` | passes the Lstat gate; refused later by `CheckRegularTarget` as `target special file` |
| `IO_REPARSE_TAG_DEDUP` | no type bit (regular by explicit design) | admitted |

The escape-capable set (symlinks, junctions/mount points,
AppExecLinks, cloud placeholders) is established under both mappings;
the universal "every reparse point" is not, with the two
counterexamples above — neither escape-capable. Stated this way in
`open_member_windows.go`, `open_nofollow_windows.go`, and the README.
Windows remains compile-and-vet only, never executed.

## Gates (all run in this session, standalone processes)

| Gate | Command | Result |
| --- | --- | ---: |
| Full suite | `go test ./... -count=1` | exit 0, 20 packages ok |
| Full cover | `go test ./... -cover -count=1` | exit 0 (secprim 94.4%) |
| Race | `go test -race ./internal/secprim/ ./internal/cliresult/ -count=1` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Windows vet | `GOOS=windows go vet ./...` | exit 0 |
| Windows build | `GOOS=windows go build ./...` | exit 0 |
| Format | `gofmt -l internal/` | clean (empty) |
| Tracecheck | `go run ./internal/traceability/cmd/tracecheck` | exit 0 |

## Tree integrity

Candidate OID `7cae2bfa6203408a3b83c1a4bb544c4352e1889b` (`git
read-tree HEAD` into a temp index + `git add -A` + `write-tree`),
captured before the battery and re-captured after the battery, after
every live plant/probe, and at finish: identical all four times. Every
mutant, plant, and probe file reverted (`cmp` byte-identical); no
`zz_*`, `.bak`, `.mutbak`, or `plant_*` residue (`git status` shows
only the rev2+rev3 work: 7 modified tracked files plus the untracked
additions `internal/secprim/`, `render_wiring_test.go`,
`env_agreement_test.go`, `runner_env_test.go`).

## Files changed (rev3 only; rev2 files untouched unless noted)

- `internal/secprim/nofollow.go`: `guardRootMatches` + `guard root
  mismatch` site in `Guard.Open`; binding documented; `OpenNoFollowFile`
  doc points at `Guard.Open` (advisory 1).
- `internal/secprim/open_member_unix.go`: classify-before-close at both
  sites; named `rootPath` with binding note.
- `internal/secprim/open_member_windows.go`,
  `open_nofollow_windows.go`: binding note; honest reparse mapping.
- `internal/secprim/errors.go`, `inventory_test.go`: spelling bound
  (+ `TestRefusalInventorySpellingBoundIsStated`).
- `internal/secprim/guard_root_test.go` (new): C1 refusal + positive
  control, cross-platform.
- `internal/secprim/open_unix_test.go`: C2 depth-3/4 vectors.
- `README.md`: binding, any-depth escape, 42 sites, spelling bound,
  escape-capable reparse set + counterexamples, wiring surface stated
  (advisory 2).
