# TASK-260830-2ciy0s — reviewer verdict, CR revision 3

**Verdict: ACCEPTED** (`accept_cr`, revision 3).

repeat-of: none (no blocking finding).

Reviewed at candidate tree `5eb6d8a547b6efa392cdfe0c66dc2132fed96ce5`
against base `157a54cfe4e952a2638b574230967989178e9a05`, 35 changed paths.
Working tree re-derived independently and confirmed byte-identical to the
candidate tree. Companion: `TASK-260830-2ciy0s_review-probe-log-rev3.txt`
(every probe, mutant, plant and gate below, with output).

All four round-2 blocking findings (C1–C4) are closed, and all five gates
the round-3 brief marked blocking are satisfied. I found **no production
defect** on any vector I could construct. Eight advisory findings are
recorded at the end; none of them is a defect in shipped behavior, and
each carries the exact reproduction.

---

## G-A (blocking) — the root check is a real identity check: PASS

`guardRootMatches` (`internal/secprim/nofollow.go:29-39`) fstats the
handle (`root.Stat()`), stats the path (`os.Stat(rootPath)`), and compares
with `os.SameFile` — device+inode on unix, volume+file index on Windows.
It is not a path-string or name comparison. Either stat failing returns
`false`, so an unverifiable root refuses.

Driven through the production entry `Guard.Open`:

| vector | result |
| --- | --- |
| directory **replaced at the same path** (path strings identical, inode differs) | `REFUSED — guard root mismatch`, no bytes admitted |
| guard root **renamed away** under the open handle | `REFUSED — guard root mismatch` (fail-closed on the stat error) |
| foreign handle | `REFUSED — guard root mismatch` |
| foreign handle whose **basename equals** the root's (`base/a/stage` vs `base/b/stage`) | `REFUSED — guard root mismatch` |
| root path replaced by a **symlink to the handle's own directory** | admitted — `os.Stat` follows the link, so it is the same directory; an operator-configured root, not an escape |

The same-path/different-inode vector is the one that discriminates: a
path-string check admits it, an identity check refuses it. It refuses.

**One semantics or two?** The *binding* is one: it lives in shared
`Guard.Open` with no build tag, runs before either walk, and any
`SameFile` error fails closed — so a mismatched handle is answered
identically on both platforms by construction, not by coincidence. The
*descent* is two: unix descends from the verified handle, Windows from the
verified `rootPath` string. Both name the same directory at bind time. A
root swap *after* the bind would diverge on Windows; that is the
check-then-open race already stated as a bound in
`open_member_windows.go`, and Windows is compile-and-vet only, never
executed. The failure direction there is closed (a `SameFile` error
refuses a legitimate handle rather than admitting a foreign one).

## G-B (blocking) — the fd lifetime is fixed as a lifetime, not a message: PASS

Read directly at both sites (`open_member_unix.go:41-76`): the loop error
arm and the final error arm both bind `classified := classifyCommitErr(...)`
**before** `unix.Close(previous)`; the success arms close `previous` only
after the next `openat` has already consumed `current`. No descriptor is
read after close on any arm, error arms included.

Driven, beyond the committed depth-3/4 vectors:

| member | result |
| --- | --- |
| `a/b/c/d/file` (depth 5, honest) | ADMITTED, read `inside5` |
| `a/b/c/hop/loot` (symlinked intermediate at depth 4) | `REFUSED — member symlink escape` |
| `a/b/c/d/hop2/loot` (symlinked intermediate at depth 5) | `REFUSED — member symlink escape` |
| `a/b/link` (**final component** symlink at depth 3) | `REFUSED — member symlink escape` |
| `a/b/c/d/link5` (**final component** symlink at depth 5) | `REFUSED — member symlink escape` |

Each names the right arm, not "some refusal arrived". 400 repeats of a
depth-5 escape yield exactly one distinct refusal string (no recycled-fd
drift), and 6000 walks across the success and error arms exhaust no
descriptors.

## G-C (blocking) — which C3 answer was taken: STATED BOUND, correctly: PASS

The producer took the second option: state the identifier-spelling bound
and stop claiming the class. `go/types` was not introduced, and there is
no third syntactic round.

- The bound is stated in `errors.go:59-68`, in the `deriveRefusals` doc
  (`inventory_test.go:202-212`), and in the README, and it **names what
  escapes it**: `type X = Error`, `type X Error` (even converted back to
  `*Error`), and an alias-spelled factory result.
- It is pinned executably: `TestRefusalInventorySpellingBoundIsStated`
  feeds plants H/I/J through the **production** `deriveRefusals` and
  requires silence — no site, no stray.
- Nothing still claims the class. README: "outside the witness, stated as
  a bound". `TestRefusalInventoryClosesBypassShapes`'s doc now says
  explicitly that spellings outside the `Error` identifier are not members
  of its space.

Reviewer plant controls (real files on disk, run, removed; tree OID
re-verified identical afterwards): an unexercised `failArgv` call is
CAUGHT (`refusal site zzrv_plant_control.go:4 never fired during the
suite`); a token-preserving `os/exec` import with the name assembled from
variables is CAUGHT; a `/bin/sh -c` literal is CAUGHT. The instrument
works and is not vacuous.

## G-D — the battery arithmetic: PASS, reproduced independently

I re-derived the denominator from the candidate source myself: **42
refusal exits** (argv 5, env 9, nofollow 15, path 13) and **29 distinct
rules** — exactly as claimed, strays 0.

The log has 61 rows: 59 `KILLED` + 1 `KILLED-OTHER` (D14) + 1 `SURVIVED`
(N7). That is the missing one the brief asked me to name: **D14**, the
compile-fail, honestly excluded from the kill column.

Contested rows re-run by me through an independent overlay harness (never
touching the worktree):

| row | reviewer result | note |
| --- | --- | --- |
| N24 (directory arm drop) | **KILLED** | restoration is valid; the rev2 withdrawal reasoning was wrong and the producer says so |
| N24 `Size() > 0` spelling | SURVIVED | **genuinely vacuous**: `IsDir() && Size()>0` ≡ `IsDir()` for real directories on this host, so it admits nothing |
| N7 | SURVIVED | bound correct — `!mode.IsRegular()` subsumes any single disjunct |
| N7b (two-arm) | **KILLED** | admits FIFOs, `TestCheckRegularTarget` fails |
| N28, D20, N29, D14b, D21 | **KILLED** | all reproduce |
| D14 | COMPILE-FAIL | `secprim` imported and not used — reported honestly, not relabelled |

D14 → D14b is a real supersession, not a rename: D14 cannot compile, D14b
keeps the import and deadens the gate with `&& false`, and it kills
`TestExecRunnerRefusesUnusableExecutable`.

"22 of 29 rules narrowing-covered" does **not** mean 7 rules rest on arm
deletion. The residue is 6 stated bounds (env lookup missing, guard root,
guard root handle, open failed, open stat failed, target stat missing) plus
1 probe-verified defensive conjunct (member containment / R12). Each has
its reason in the AC table, and each reason holds: they are nil-argument
programming errors, errno propagation, seam-driven paths, or delegation to
`scalar`, none of which has an input class to admit. That is adequate.

## G-E — evidence persistence: PASS

`TASK-260830-2ciy0s_secprim-rev3.md` is a complete **18836-byte file**
resource (not truncated inline content) carrying the full 31-row AC table
with a production call site per row. The battery script, the 61-row
battery log, the full-suite log, the race log and the validation log are
all attached as files.

The Windows bound is stated in its honest form in three places
(`open_member_windows.go`, `open_nofollow_windows.go`, README): the
escape-capable set (symlinks, junctions/mount points, AppExecLinks, cloud
placeholders) is established under both the go1.25.5 mapping and
`GODEBUG=winsymlink=0`; the universal "every reparse point" claim is
withdrawn, with both counterexamples named — `IO_REPARSE_TAG_AF_UNIX`
(→ `ModeSocket`, refused later by `CheckRegularTarget`) and
`IO_REPARSE_TAG_DEDUP` (no type bit, admitted), neither escape-capable.

## Independent attack beyond the brief

I built 24 narrowing mutants of my own against the production-reachable
surfaces. **20 were killed by the committed suite**, including all six
`cliresult` wire sites, nine `EscapeForTerminal` arms, both `IsEnvName`
boundaries (through `provhost` as well as `secprim`), the
empty-`Env`-denies-all obligation, six `Redact` arms, and both `Command`
copy obligations. Four survivors are advisories A1–A5 below; the fifth
(`withinRoot` slash boundary) is the R12 conjunct the producer already
declares and double-probes.

I also ran a directed containment enumeration: a 16-token hostile alphabet
to depth 4 — **69,904 members** — driven through both `Guard.Resolve` and
`Guard.Open` against a stage carrying symlinked intermediates at depths
1/2/3 and a final-component symlink, with every admitted result checked
against the real filesystem via `EvalSymlinks` rather than lexically.
Result: 2800 lexically admitted by `Resolve`, exactly 1 committed by
`Open` (`a/b/file`, the honest member), **0 escapes**. The instrument
fails if nothing is admitted, so it is not vacuous.

Gates I ran myself: `go build ./...` 0; `gofmt -l internal/` clean;
`go vet ./...` 0; `GOOS=windows go vet ./...` 0; `GOOS=windows go build
./...` 0; `go test ./... -count=1` 0 (20 packages); `secprim -cover`
94.4%; `-race` on secprim+cliresult 0; `tracecheck` 0. 1295 test runs
across secprim/provhost/cliresult with **zero skips** on this host — no
platform skip is hiding coverage here.

Wiring verified at the production call sites, not from prose: the only
production callers of `secprim` are `cliresult/output.go:20`
(`RenderForTerminal`, reached from all six text paths, JSON left
byte-exact at `output.go:176`), `provhost/runner.go:72` (`CheckArgv`) and
`provhost/spawn.go:174` (`IsEnvName`). Unwiring any one of the six render
sites reddens a named test. The README correctly says the path/containment
half has no production caller yet — no unsupported capability is claimed.

---

## Advisory findings (non-blocking; each control-planted)

Production behavior is correct in every one of these. They are unmeasured
arms — a future edit could weaken them and the suite would stay green.
Each is cheap to close and should be picked up by the next leaf that
touches this package.

**A1 — `guardRootMatches` identity axis has no witness.**
Mutant `os.SameFile(handleInfo, pathInfo)` → `handleInfo.Name() ==
pathInfo.Name()` survives the whole `secprim` suite. Control: with it
applied, a foreign handle whose basename equals the guard root's basename
is admitted and reads `FOREIGN` outside the root. N28 measures the
kind-vs-identity axis; the committed fixture (`stage` vs `elsewhere`)
happens to discriminate on name, so the identity-vs-name axis is untested.
Closing it costs one fixture rename.

**A2 — `guardRootMatches` fail-closed arm has no witness.**
Mutant: the `os.Stat` error arm `return false` → `return true` survives.
Control: with the guard root removed, a foreign handle is admitted and
reads `FOREIGN`. `nofollow.go:26-28` explicitly claims "an unreadable root
fails closed into the mismatch refusal rather than admitting an
unverifiable handle" — the claim is true in production (I drove it) but
nothing pins it. This is the "a failed read is not a legitimate absence"
shape applied to the gate's own stat.

**A3 — `isBareSensitiveKey` case-folding has no witness on the spaced
bare-key path.** Mutant `sensitiveKeyNames[strings.ToLower(core)]` →
`[core]` survives. Control: `Redact("SECRET = hunter2abcdef")` returns the
line unchanged while `secret = hunter2abcdef` still redacts. The compact
path's fold (`maskSensitiveChunk`) *is* pinned — that mutant dies. `Redact`
is wired into all six `cliresult` text paths.

**A4 — `strings.ToValidUTF8` in `EscapeForTerminal` is fully deletable
with nothing failing.** The `range` loop already yields U+FFFD per invalid
*byte*, so output stays valid UTF-8 and no raw byte reaches the terminal;
only the documented "U+FFFD **per run**" coalescing changes. No security
delta, but the documented behavior is unmeasured.

**A5 — `memberErrorTarget` bound witnessed far out of range.** `len > 64`
→ `len > 65` survives; production truncates a 65-character member
correctly (verified), but the corpus never sits on the edge.

**A6 — a test comment over-claims relative to its gate.**
`TestPackageLaunchesNoProcess` says the package "validates argv but never
launches"; the gate refuses an `os/exec` import. A planted production file
calling `os.StartProcess` with no shell token walks through both halves of
the no-shell gate. The README wording is precise ("a behavioral
no-`os/exec` import gate"); the test comment should match it.

**A7 — two provenance statements in the evidence do not hold, though both
underlying facts do.**
(i) The rev3 document names candidate OID `7cae2bfa…`, but the CR
candidate tree is `5eb6d8a5…`. I diffed them: the entire delta is
`LOGBOOK.md` +9 lines, written after the captures — **no Go source
differs**, so every battery/plant/probe result still applies to the
delivered tree. The "identical all four times" claim is nonetheless not
true of the finish tree.
(ii) The document says both `N24` spellings' results "are in the battery
log". The vacuous `Size() > 0` result is not in the 61-row log; it exists
only in the prose. I reran it myself and it does survive and is genuinely
vacuous, so the claim is right and its cited location is wrong.

**A8 — the final-arm close ordering has no behavioral witness, and cannot
have one here.** Reordering the *final* site to close-before-classify
survives everything, because a final-component symlink fails with `ELOOP`,
which `classifyCommitErr` answers without touching a descriptor; the
`ENOTDIR` re-examination is reachable only from the loop site, which N29
pins. Not a defect — same category as the R12 defensive conjunct — but the
evidence's "both error sites" phrasing implies two measured sites when one
is measured and one is defensive.

---

## Definition of Done

Every merged checklist item is verified against the artifact, not against
prose: production entry points implement the scoped deliverable; positive,
negative and recovery tests pass with logs attached; README/traceability
updated with no unsupported claim (the unwired path half is declared);
31 of 31 AC rows driven with a named production call site; gating behavior
covered by negative tests; every gate carries at least one narrowing
mutant, deletion-only evidence rejected; the source-text gate is attacked
by a token-preserving mutant and the harness runs the behavioral suite;
lint, vet, build (including `GOOS=windows`) and the full suite green;
outcome artifacts attached as task-scoped files; logbook updated.

Accepted. `accept_cr` routes `TASK-260830-2ciy0s` to `integrating`; the
bound `developer`/`implementer` producer run owns checkpoint and
integration.
