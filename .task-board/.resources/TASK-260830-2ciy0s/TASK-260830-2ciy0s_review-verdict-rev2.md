# TASK-260830-2ciy0s — reviewer verdict, CR revision 2

- Run: `RUN-260906-26a2e6` (reviewer/reviewer)
- Change Request: `CR-TASK-260830-2ciy0s-2` rev 2, base `157a54c`,
  candidate tree `703ee895c6b5f8bc2e6a97cb4aff7e538ce79c90`,
  `repository_delta=present`, 33 changed paths.
- Verdict: **changes requested** → `to-dev`.
- `repeat-of:` **rev1/B5** (finding C3 only; C1, C2, C4 are new).
- Worktree verified byte-identical to the candidate tree before and after
  every probe (`git read-tree HEAD` into a temp index + `git add -A` +
  `write-tree` → `703ee895c6b5f8bc2e6a97cb4aff7e538ce79c90`). Every mutant,
  plant and probe file reverted; no `zz_*` or `.bak` residue in the tree.

## What I re-ran myself (not accepted from the producer)

| Gate | Command | Result |
| --- | --- | ---: |
| Full suite | `go test ./... -count=1` | exit 0, 20 packages ok |
| Focused suite (post-revert) | `go test ./internal/{secprim,cliresult,provhost,scalar}` | exit 0 |
| Race | `go test -race ./internal/{secprim,cliresult}` | exit 0 |
| Format | `gofmt -l internal/` | clean |
| Vet | `go vet ./...` | exit 0 |
| Windows vet | `GOOS=windows go vet ./...` | exit 0 |
| Windows build | `GOOS=windows go build ./...` | exit 0 |

Accepted from the producer without re-running: `-cover`, `tracecheck`.
Everything below I executed in this session.

---

## Blocking findings

### C1 — `Guard.Open` never binds the walk to `guard.root`; any directory handle becomes the containment root

Driven through the production entry, no mocks:

```
NewGuard("<tmp>/stage", PlatformLinux, nil)              -> ok
OpenNoFollowDir("<tmp>/elsewhere")                       -> handle
guard.Open(thatHandle, "secret")                         -> ADMITTED, read "FOREIGN"
```

`openCommitRelative(root *os.File, _ string, member string, segments []string)`
takes the guard root path and **discards it** (`_ string`). The unix walk starts
at `root.Fd()` and never checks that the handle is the guard's root, so
`Guard.Open` enforces "relative to *a* directory", not "inside *this* root". The
only handle check is `root.Stat()` → `IsDir()`.

This is not a caller-contract nit, because the two platforms disagree:
`open_member_windows.go` uses `rootPath` as the descent base
(`filepath.Join(current, segment)`, `filepath.Join(rootPath, …)`), so on Windows
the same call refuses (the member is looked up under the guard root and is not
there) while on unix it admits and reads. One production entry, two containment
semantics, and no test on either platform drives a handle that does not match
`guard.root`.

`Guard`'s own doc calls the type "a containment boundary … a validated absolute
root", and `Resolve` enforces that root. `Open` — the half that is supposed to be
the commit conjunct of §16.3 containment — does not.

Named fix: bind the handle to the root. Either `fstat` the handle and compare
`Dev`/`Ino` against a `Stat` of `guard.root` (refusing on mismatch with a named
rule), or remove the choice by exposing `guard.OpenRoot() (*os.File, error)` and
taking the handle only from there. Drive the mismatched-handle refusal as a
negative test on both platforms, and make the unix and Windows walks agree on
what the root is.

### C2 — a symlinked intermediate at depth ≥ 2 is misclassified, because `classifyCommitErr` fstats a closed descriptor

Measured through `Guard.Open` against a real filesystem:

| Member | Symlinked component | Refusal |
| --- | --- | --- |
| `hop1/loot` | 1st intermediate | `member symlink escape` |
| `l1/hop/loot` | 2nd intermediate | **`open failed: … not a directory`** |
| `a/b/hop/loot` | 3rd intermediate | **`open failed: … not a directory`** |

Mechanism, confirmed by moving one statement: in `openCommitRelative` the error
branch closes `previous` **before** calling `classifyCommitErr(current, …)`, and
from the second iteration onward `previous == current`. So `unix.Fstatat(parent,
…)` runs on a descriptor that was just closed, returns `EBADF`, the symlink
re-examination is skipped, and the raw `ENOTDIR` falls through to the
"open failed" refusal. Reordering classify before close makes all three depths
report `member symlink escape`:

```
classified := classifyCommitErr(current, segment, err)
if previous >= 0 { _ = unix.Close(previous) }
return nil, classified
```

I confirmed the errno premise on this host directly: `openat` with
`O_NOFOLLOW|O_DIRECTORY` on a symlink gives `ENOTDIR`, not `ELOOP`
(`ELOOP=false ENOTDIR=true`), so the `fstatat` re-examination is the *only* thing
that names the shape on darwin — and at depth ≥ 2 it never runs.

Consequences:

1. `README.md` claims `Guard.Open` "refuses a symlinked parent at commit time as
   `member symlink escape`". Falsified for every member with two or more
   intermediate components.
2. It is a use-after-close on a file descriptor. Every branch still refuses, so
   this is not an escape — but in a concurrent process the fd number can be
   reused between the close and the `Fstatat`, and the check then interrogates an
   unrelated directory. A refusal that consults a recycled descriptor is not a
   refusal anyone can reason about.
3. The suite cannot see it: every committed commit-walk case uses a one- or
   two-segment member (`sub/loot`, `final-link`, `ok/absent`, `nodir/file`), so
   `previous` is always `-1` and the closed-fd path is never taken. The same
   ordering bug exists on the final-component open (`previous` closed before
   `classifyCommitErr`); it is masked there because `ELOOP` catches a trailing
   symlink before the `fstatat`.

Named fix: classify before closing at both sites, and add a commit-walk case with
at least three segments and the symlink on a non-first intermediate.

### C3 — the inventory gate closed four spellings, not the class (`repeat-of: rev1/B5`)

Round 1: four of four bypass plants walked through an **identifier-keyed**
gate. Round 2 rebuilt the derivation around alias fixpoints, literals,
`new(Error)`, factories and value escapes — and keyed it on the *type*
identifier instead. I planted a fifth shape the gate does not name; it and two
variants walk through a fully green suite. Control proves the live instrument
works:

| Plant | Shape | Result |
| --- | --- | ---: |
| CONTROL | `failArgv("plant control rv", …)`, unexercised | **CAUGHT** — `refusal site plant_control_rv.go:4 never fired during the suite` |
| **H** | `type shadowAlias = Error` then `&shadowAlias{Kind: ErrUnsafeArgv, …}` | **BYPASSED** |
| **I** | `type shadowNamed Error` then `(*Error)(&shadowNamed{…})` | **BYPASSED** |
| **J** | `func plantJFactory(…) *shadowResult` returning `&shadowResult{…}` | **BYPASSED** |

Root cause: `isErrorLiteral` matches `*ast.Ident` name `== "Error"` and
`returnsError` matches `*ast.StarExpr` → `Ident{"Error"}`. Any other spelling of
the same type is invisible, so an alias-spelled literal is neither a literal
stray nor inside a factory, and an alias-spelled result makes the enclosing
function not a factory either. H is a direct, invisible violation of the rule
`errors.go` states, exactly as rev1's plant B was.

`README.md` says the inventory "additionally fails on any refusal built outside
the constructors (aliases, literals, factories, value escapes)". "Any" is
falsified.

This is the second consecutive round of the same class, so the recommendation is
a gate rather than a third syntactic revision:

- **Preferred: state the bound.** Keep the enumerated-spelling gate and declare
  in `errors.go`, the derivation comment and the README that the witness covers
  the `Error` identifier spelling, and that alias or named-type spellings of the
  type are outside it. That is honest, costs one paragraph, and stops the
  find-a-sixth-shape loop.
- **Or resolve types instead of syntax.** Load the package with `go/types`
  (`packages.NeedTypes`) and ask whether a composite literal's type is identical
  to `secprim.Error` and whether a result type is `*secprim.Error`. That closes
  alias and named-type spellings in one move, and is the only version of this
  gate that can claim the class.

Do not ship a third round of added identifier cases and call the class closed.

### C4 — the rev2 outcome artifact is truncated on the board; the AC ratio and the survivor/withdrawal reasoning are not there

`TASK-260830-2ciy0s_secprim-rev2.md` on the board is 12 515 bytes and ends
mid-sentence:

```
### Survivors with stated bounds — 2

- N7 (drop NamedPipe disjunct only): SURVIVES — the `!mode.IsRegular()`
  catch-all subsum
...[truncated 5836 chars]
```

Everything after that is gone from the persisted evidence: the AC-row coverage
table (the reported 16 of 25 and the production call site per row), the R12
bound, the N24 withdrawal reasoning, M13a, and the Windows bound statement. The
full text exists nowhere else — not in `.temp/`, not in another resource. The DoD
requires the coverage ratio and its named call sites as *reported* evidence, and
the spawn brief asked me to check the N7/R12/N24 reasoning rather than accept it;
that reasoning is not checkable from the board.

I re-derived what could be re-derived (below): the 41-exit denominator is exact,
N7's bound is correct, and R12's is honest. But **N24 does not hold up**: I
applied "drop the directory arm only" and it is **KILLED** by
`TestCheckRegularTarget` and `TestOpenNoFollowFile` (the refusal message moves
from `target directory` to `target special file`). Withdrawing it as "invalid"
removed a valid, killing narrowing mutant from the count — under-claiming rather
than over-claiming, but the stated reason is wrong and unverifiable as persisted.

Named fix: attach the artifact as a file (`task-board resource add`) rather than
inline `content=`, which is what truncated it, and correct the N24 row.

---

## Gate-by-gate results

### G-A — is the walk actually relative at every step? **Yes on unix.**

- **No path-string re-resolution anywhere on the unix walk.** Every step is
  `unix.Openat(current, segment, …)` from the previous descriptor;
  `filepath.Join` appears nowhere in `open_member_unix.go`. Intermediates carry
  `O_RDONLY|O_NOFOLLOW|O_DIRECTORY|O_CLOEXEC`, the final component
  `O_RDONLY|O_NOFOLLOW|O_CLOEXEC|O_NONBLOCK`. Descriptors pin what they opened.
  (Windows re-resolves the root path string; that is stated in-source as a bound
  alongside compile-and-vet-only, and it is the C1 divergence.)
- **Distinct grammar obligations, all refused through `Guard.Open`** — 15 of 15,
  each with its own rule:

  | Member | Refusal |
  | --- | --- |
  | `..`, `../x`, `d/../../etc/passwd` | `member grammar` |
  | `/etc/passwd`, `C:/x`, `d\f` | `member grammar` |
  | `d//f` (empty segment), `d/f/` (trailing) | `member grammar` |
  | `d/./f`, `./d/f`, `.` | `member grammar` |
  | `""` | `member grammar` |
  | `d/%2e%2e/f`, `d/%252e%252e/f` | `member encoded dot` |
  | `d/%c0%af/f` | `member overlong separator` |

- **Final and intermediate are both driven.** Trailing symlink →
  `member symlink escape` (committed `final symlink` case, re-run). Intermediate
  → refused at all three depths I probed, but see C2 for the rule name.
- **TOCTOU confirmed at the commit step, not by luck.** I built an honest
  `stage/sub/loot`, called `Resolve` (admitted), then `rm -rf`'d `sub` and
  replaced it with a symlink to `outside`, then called `Open`:
  `member symlink escape`. The refusal comes from the walk, after validation had
  already passed on the honest tree. No committed test drives this swap shape —
  advisory only, since descriptors pin and the property holds.
- **`Resolve` + `OpenNoFollowFile` still escapes** — I drove it and read
  `ESCAPED`. That is expected (rev1's B1 vector through the composed API), but
  `OpenNoFollowFile`'s doc comment still recommends exactly that composition
  ("Parent directories are the caller's responsibility: open the staging root
  with `OpenNoFollowDir` and resolve members through a `Guard`"). Point it at
  `Guard.Open`.
- **FIFO refuses through both openers.** `OpenNoFollowFile` and `Guard.Open`
  (`d/pipe`) both answered `target special file: pipe` inside a 4 s deadline.
  B2 closed, on the commit path too.

### G-B — B7's recursive scan: **closed.**

| Plant | Location | Result |
| --- | --- | ---: |
| CONTROL | `internal/scalar/path.go` | **CAUGHT** with path |
| rev1 bypass | `internal/catalog/cmd/cataloggen/main.go` | **CAUGHT** with path |
| package added since | `internal/secprim/argv.go` | **CAUGHT** with path |
| second nested cmd | `internal/traceability/cmd/tracecheck/main.go` | **CAUGHT** with path |

Fail-closed measured on the test's **own** arms, not just via the compiler: a
plant under `internal/scalar/testdata/` (which the toolchain ignores but the walk
reads) produced `emission-path walk failed closed: parse …: import path must be a
string`, and a `chmod 000` file produced `emission-path walk failed closed: open
…: permission denied`. An `archive/zip` import under `testdata/` is also caught,
so the scan is if anything over-broad — safe direction.

The file count is **derived**, not asserted: `scanned++` per file, checked only
`!= 0`, and reported with `t.Logf` (110 on this tree). A new package cannot rot
it.

`TestNoArchiveEmissionPath` is cited by **five** `nopath` rows (Machine
authentication, Live process identity, IPC/runtime, Transient locking, Mutable
derived indexes). All five are now stated as archive-format bounds — each says
what the scan proves and explicitly what it does not — and **none is a sole
witness any more**: each row carries a second witness
(`TestSecretSiteRosterIsComplete`, `axerror.TestCauseNeverReachesTheWire`, or
`TestOpenNoFollowDir`). Honest.

### G-C — B5's shape space: **not closed.** See C3.

### G-D — B6, B3, B4: **all closed.**

- **B6 corpus.** Feed and assertion are the same `forbidden` slice, one
  declaration and two loops over it (`render_test.go:71-84` feeds,
  `:99-103` asserts). The counts are equal *by construction*; they cannot drift
  again. Both endpoints of every range are now witnessed:

  | Mutant | Narrowed to admit | Result |
  | --- | --- | ---: |
  | `>= 0x202a` → `>= 0x202b` | U+202A | KILLED (`bidi_embedding_202a`) |
  | `<= 0x202e` → `<= 0x202d` | U+202E | KILLED (`bidi_override`) |
  | `<= 0x2069` → `<= 0x2068` | U+2069 | KILLED (`bidi_isolate_2069`) |
  | `>= 0x2066` → `>= 0x2067` | U+2066 | KILLED (`bidi_isolate`) |
  | `>= 0x200b` → `>= 0x200c` | U+200B | KILLED (`zero_width_space`) |
  | drop `== 0x061c` | U+061C | KILLED (`directional_isolate_061c`) |

  U+202A–U+202D are witnessed per-rune, and the range is now pinned at both ends
  rather than at one.

- **B3 — all six `cliresult` paths driven.** Every wiring site killed by a
  narrowing mutant (`terminalLine(x)` → `x`):

  | Site | Mutant | Killed by |
  | --- | --- | --- |
  | `Emit` success → stdout | R27 | `TestTextSuccessEmissionNeutralizes` |
  | `Emit` failure → stderr | R22 | `TestTextEmissionNeutralizesHostileContent` |
  | `Progress` | R23 | `TestProgressEmissionNeutralizes` |
  | `Prompt` refusal text | R24 | `TestPromptRefusalCarriesNeutralizedText` |
  | `Log` | R28 | `TestTextEmissionNeutralizesHostileContent` |
  | `Prompt` interactive write | R28b | `TestTextEmissionNeutralizesHostileContent` |

- **B4 — nil vs empty are distinct obligations.** `Env != nil` →
  `len(Env) > 0` is **KILLED** by `TestExecRunnerEmptyEnvIsDenyAll`; dropping the
  wiring entirely is killed by two tests. I also narrowed the co-located argv
  gate two ways — stripping NUL before `CheckArgv`, and gating it on
  `executable == ""` only — and both are **KILLED** by
  `TestExecRunnerRefusesUnusableExecutable`
  (`Run("bad\x00bin") = *exec.Error …, want *secprim.Error`).

### G-E — the ratio and the survivors

**Denominator re-derived, exact.** Running the production `deriveRefusals` over
the candidate's non-test sources: **41** refusal exits — argv 5, env 9,
nofollow 14, path 13 — matching the reported figure component by component, with
zero strays. (Rev1's 35 plus the six new `Guard.Open` sites.)

**Instruments reported separately, which is the right shape** (narrowing 27,
deletion 15, instrument probes 2, bypass probe 1). Measured against the
*distinct refusal rules* rather than the exit count: **14 of 28 rules carry a
narrowing mutant**; the other 14 are existence-probe or fired-only. Several of
those are error-propagation exits with no class boundary to move
(`open failed`, `open stat failed`, `target stat missing`, `guard root handle`),
but several are real class gates covered by deletion alone. I narrowed six of
them myself:

| Mutant | Narrowed to admit | Result |
| --- | --- | ---: |
| `env value NUL`, allowlist arm → prefix-only | NUL past index 0 | KILLED |
| `env value NUL`, literals arm → prefix-only | NUL past index 0 | KILLED |
| `env value encoding`, allowlist arm → `len > 4` | short invalid UTF-8 | KILLED |
| `env value encoding`, literals arm → `len > 4` | short invalid UTF-8 | KILLED |
| `env allowlist duplicate` → `len(name) > 3` | short duplicate names | KILLED |
| `env literal collision` → `len(key) > 3` | short colliding keys | KILLED |

So the gap is in the **reporting**, not the coverage: those gates are genuinely
bounded and the battery understates itself. Note for the table — `env value NUL`
and `env value encoding` each have two arms sharing one rule name; the derivation
keys sites by `file:line`, so both arms must fire independently, and I confirmed
each arm dies to its own narrowing mutant rather than hiding behind its twin.

**Survivors, checked rather than accepted:**

- **N7 (drop the `ModeNamedPipe` disjunct alone): SURVIVES — bound correct.**
  `CheckRegularTarget`'s case list ends in `!mode.IsRegular()`, which subsumes
  the device, named-pipe, socket and char-device disjuncts. Any one of them alone
  is unobservable; the two-arm N7b is the right instrument, and it kills.
- **R12 (drop the slash boundary from `withinRoot`): SURVIVES — bound honest,
  and I verified the tripwire rather than trusting it.** I also ran the stronger
  mutant (`withinRoot` → `return true`), which likewise survives, confirming the
  `defensiveSites` claim that the conjunct is unreachable through the entry. The
  claim that matters is that a *grammar relaxation* reddens, so I weakened the
  shared grammar in `scalar.ParseRelativePath` to admit `..` segments:
  - grammar weakened, `withinRoot` intact → `TestGuardPrefixCheckIsLoadBearing`
    **passes**: the prefix conjunct catches the escape, which is what defense in
    depth is for;
  - grammar weakened **and** `withinRoot` neutralized → **FAILS**,
    `Resolve("x/../..") escaped to "/stage"`, `24 admitted members escaped the
    root`.

  The defensive site and its rationale are both load-bearing exactly as stated.
- **N24: not a legitimate withdrawal.** See C4 — the mutant kills.

### Windows — established for the tags that matter; the universal is overstated

The spawn brief asked whether the reparse mapping is now established or stated as
a bound. It is **established, and slightly overstated**. Checked against the
toolchain in use (go1.25.5, `$GOROOT/src/os/types_windows.go`,
`(*fileStat).mode()`):

| Reparse tag | Go maps to | Gate outcome |
| --- | --- | --- |
| `IO_REPARSE_TAG_SYMLINK` | `ModeSymlink` | refused ✓ |
| junction / `MOUNT_POINT`, AppExecLink, cloud placeholder (default arm) | `ModeIrregular` | refused ✓ |
| `IO_REPARSE_TAG_AF_UNIX` | **`ModeSocket`** | passes the Lstat gate |
| `IO_REPARSE_TAG_DEDUP` | **no type bit** (deliberately regular) | admitted |

So the claim "every other reparse point … surfaces as `ModeIrregular`, never as
`ModeSymlink`" has two counterexamples. Neither opens an escape: an AF_UNIX
reparse point is refused a moment later by `CheckRegularTarget` as
`target special file`, and DEDUP is a regular file by Go's explicit design
decision. There is also a `GODEBUG=winsymlink=0` path (`modePreGo1_23`) under
which a junction reports `ModeSymlink` instead — still refused, since the gate
tests both bits. The escape-capable §16.3 "reparse" obligation is therefore met
under both mappings; the sentence should enumerate the two exceptions and name
the `GODEBUG` dependency instead of quantifying over all tags. **Advisory, not
blocking.** Windows remains compile-and-vet only, never executed, which the
source states.

---

## AC-row coverage

I cannot report the ratio as measured, because the artifact that carried it is
truncated (C4) and the per-row production call sites are in the missing section.
What I can state from execution: the three rows rev1 falsified are now true —
symlink/reparse escape at commit (driven, C2 affects the rule name not the
refusal), TOCTOU dir-handle primitives (the handle is now walked, C1 affects
which root it binds to), and device/FIFO/socket refusal (driven through both
openers). Re-attach the table and I will check the ratio against it.

---

## Advisory (not blocking)

1. `OpenNoFollowFile`'s doc comment still recommends the composition that
   escapes containment; point it at `Guard.Open`.
2. The path/containment half of `secprim` has no production caller — only
   `RenderForTerminal`, `CheckArgv` and `IsEnvName` are wired
   (`cliresult/output.go`, `provhost/runner.go`, `provhost/spawn.go`).
   Consistent with a primitives leaf whose consumer is a later leaf, but the
   README's "a library behind the production launch and rendering paths" reads
   wider than the wiring. State which surface is wired today.
3. No committed test drives the validate-then-swap TOCTOU shape. The property
   holds (I drove it), so this is coverage, not correctness.
4. The Windows reparse sentence — see above.

## Required for revision 3

1. **C1** — bind `Guard.Open`'s handle to `guard.root` (fstat/dev-ino compare, or
   a `guard.OpenRoot()` that removes the choice), make the unix and Windows walks
   agree on the root, and drive the mismatched-handle refusal.
2. **C2** — classify before closing at both `classifyCommitErr` call sites, and
   add a commit-walk case with ≥ 3 segments and the symlink on a non-first
   intermediate. Correct the README sentence about the refusal rule.
3. **C3** — either state the identifier-spelling bound in `errors.go`, the
   derivation comment and the README, or resolve types with `go/types`. Do not
   add more identifier cases and re-claim the class. Re-run plants H, I and J as
   the acceptance evidence either way.
4. **C4** — re-attach the outcome artifact as a file so it is not truncated, with
   the AC-row table and call sites intact, and correct the N24 row (it kills).
