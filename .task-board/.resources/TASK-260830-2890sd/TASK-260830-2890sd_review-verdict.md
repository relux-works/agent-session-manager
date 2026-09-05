# TASK-260830-2890sd — review round 4 verdict (CR rev 5)

**Verdict: accepted → `accept_cr(TASK-260830-2890sd, revision=5)`.**

Reviewed the real patch this round. Worktree `.temp/STORY-260830-3jqsx1/worktree`,
branch `task-board/story/STORY-260830-3jqsx1`, base `44a4699`, candidate tree
`3780d471`. I confirmed the working tree hashes **byte-identical** to the
candidate tree before and after every mutant
(`git read-tree HEAD; git add -A; git write-tree` → `3780d471…` both times), so
every measurement below was taken on the exact reviewable revision and the tree
was returned to it. No `git checkout`/`restore` was used; restores came from a
`.temp` copy of the package.

## 0. Baseline, independently rerun

| Check | Result |
| --- | --- |
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `GOOS=windows go vet ./internal/provider` | clean |
| `GOOS=windows go build ./...` | clean |
| `go test ./... -race` (repo-wide) | 14/14 packages **ok** |
| `go test ./internal/provider -race -count=2` | ok |
| `go test ./internal/provider -race -shuffle=on` | ok |
| `go test ./internal/provider -cover` | **97.0%** |
| `gofmt -l .` | clean |

## 1. The three findings handed to me as pre-verified — confirmed, not re-derived

| Finding | Mutant | Result |
| --- | --- | --- |
| F11 vacuous audit | `Sites[:1]` (1 of N) | **KILLED** |
| F12 / G36 | `Inspect` swallows attester error, uid 0 | **KILLED** |
| F12 / G40 | `CurrentOperatorPolicy` seeds `{0}` | **KILLED** |

## 2. G-A — the `ownerAttester` seam: **justified, with one measured residual**

The brief asked three concrete questions. Answers, each measured.

### Can any production path reassign or observe it?

No. `ownerAttester` is unexported, in `internal/`, and read at exactly one site
(`OSSystem.Inspect`, `os.go:48`). No production source assigns it
(`grep -rn "ownerAttester ="` → the declaration plus two lines in `os_test.go`).
No package outside `internal/provider` can name it. The composed production
artifact is behaviourally identical to the pre-seam one.

It is also **not a new idiom in this package**: `failDuplicate`, `failInvalid`,
`failPrecondition`, and `failIntegrity` have been production-level `var` function
values since the first revision, reassigned by `TestMain` for exactly the same
reason — to make a production-side fact observable. `ownerAttester` is the fifth
instance of a shape two prior review rounds accepted.

### Does any test run parallel with one that swaps it?

No, and this is structurally enforced, not merely current:

- `grep -rn "t.Parallel" internal/provider/` → **zero occurrences in the package.**
- Four tests in `os_test.go` call `t.Setenv`, which **panics** if the test ever
  calls `t.Parallel`. The file cannot be parallelised without a runtime failure.
- The swap is `defer`-restored.
- `go test ./internal/provider -race -count=2` and `-race -shuffle=on` are both
  green, as is the repo-wide `-race` run.

### Is there a seam-free construction, and is the `var` justified anyway?

Two alternatives exist and neither dominates:

- **Inject through `System`/`OSSystem`** (an `attester` field with a nil
  fallback). It removes the package-level mutable, but it moves the same
  weakening target into `Inspect`'s fallback branch: a mutant that swaps the
  fallback is not killed by any test either. It trades one hole for the same
  hole in a different place.
- **Cover the branch on Windows only**, where `fileOwnerUID` always fails and
  the branch is genuinely production-reachable. Seam-free, but unexecutable on
  this host — evidence would regress from executed to compiled.

The `var` is justified: it exposes a real production branch (fail-closed
attestation, live on Windows) that is unreachable on unix, and it damages
nothing in the shipped artifact.

### The residual: the seam's own stated invariant is unenforced

`os.go` says *"Production code never reassigns it."* Nothing checks that. I
measured the cost precisely:

| Mutant | Result |
| --- | ---: |
| **SEAM1** — production `init()` sets `ownerAttester` to attest every file as the operator | **SURVIVED** |
| **SEAM2** — production `init()` sets `ownerAttester` to attest uid 0 | KILLED (`TestOSSystemEndToEnd`) |
| **CTRL1** — the *same* weakening written into `fileOwnerUID` itself, no seam | KILLED (`TestFileOwnerUIDRefusesWithoutMetadata`) |

SEAM1 vs CTRL1 is the whole finding: one semantic weakening, killed at the
natural site, survivable through the new seam, because
`TestFileOwnerUIDRefusesWithoutMetadata` calls `fileOwnerUID` directly and
nothing asserts `ownerAttester == fileOwnerUID`. The seam therefore opens
exactly one bypass shape that did not exist before.

**Why this does not block:** the bypass is not reachable by any caller — it
requires adding a production statement whose only purpose is to disable
attestation, inside the package under review. There is no defeat in the
delivered artifact. The uid-0 shape that LOGBOOK 0230 records as this package's
repeat offender (SEAM2) **is** killed. Recorded as **F18** below with the
one-line closer.

## 3. G-B — the traversal, with ratios

**80 killed / 83 measured. 3 survivors: 1 environment-equivalent, 2 substantive.**
Every round-3 killed mutant I re-ran was re-killed.

### Resurrections: none

I re-ran **45** of round 3's 59 kills — every substantive gate row across
`Discover`, the name grammar, `trustCandidate`, `OwnerPolicy`, `Trust`,
`Verify`, the `Candidate` absence accessors, `OSSystem`, and the meta-gates.

**45 of 45 still killed. 0 resurrections.** Nothing was loosened to close a
survivor. Representative re-kills: `index == 0` narrowing on both absolute-path
gates, `allow_path_plugins → if true`, duplicate narrowed to same-source, the
byte-identical duplicate override, builtins hoisted, `pi` dropped,
`sort.Strings` removed, `ExecutablePrefix → "ax-"`, every `trustCandidate`
narrowing, `uid == 0 → true`, `OwnerIdentity` constant, all four `Trust` fact
mutants, every `Verify` narrowing, `sum[:1]` / `sum[31:]`, all four absence
accessors, `EvalSymlinks` skipped, `PathDirs` nil.

### Round-3 survivors: 18 of 20 substantive survivors closed

| Round-3 finding | Mutant re-run now | Result |
| --- | --- | ---: |
| **F11** refusal-inventory domain floor | B1 empty inventory | **KILLED** |
| | B2 sites truncated to 1 of N | **KILLED** |
| | H4 scan skips `provider.go` | **KILLED** |
| | B5 stray literal + wrong-cwd (empty-but-successful) read | **KILLED** |
| Row 56 `exportedSymbols` floor | derivation returns nothing | **KILLED** |
| (new) `parseProductionSources` floor | returns an empty map | **KILLED** (both callers) |
| (new) `structMembers` floor | derives an empty map | **KILLED** |
| (new) `TestDiscoveryReachesNoProcess` floor | scan finds no production files | **KILLED** |
| **F12** row 40 `ReadDir` failure→absence | `return nil, nil` | **KILLED** |
| F12 row 43 `Inspect` stat error | fabricate `{IsRegular:true, UID:euid}` | **KILLED** |
| F12 row 44 `Inspect` attestation error | `uid = 0` | **KILLED** |
| F12 row 47 `PathDirs` empty entries | empty entries kept | **KILLED** |
| F12 row 48 admin wildcard | seeds `AdministratorUIDs{0}` | **KILLED** |
| F12 row 49 `Geteuid → Getuid` | | **SURVIVED — environment-equivalent** |
| F12 row 50 unix `fileOwnerUID` | returns `0, nil` | **KILLED** |
| F12 row 51 windows `fileOwnerUID` | — | **not executable here (bound, see F20)** |
| **F13** row 14b `Canonicalize` deletion form | `canon = path` | **KILLED** (detail pin) |
| F13 row 15b `Inspect` deletion form | fabricate info | **KILLED** (detail pin) |
| F13 row 29b `Verify` inspect deletion form | fabricate info | **KILLED** (detail pin) |
| **F14** row 11 prefix anchor | `CutPrefix → Cut` | **KILLED** |
| **F15** row 8 partial set | `return out, err` at plugin_dirs collect | **KILLED** |
| | at builtin add | **KILLED** |
| | at PATH abs-path refusal | **KILLED** |
| | at PATH collect | **KILLED** |
| | **at the `plugin_dirs` abs-path refusal** | **SURVIVED — F19** |
| **F16** row 33 empty-owner receipt | owner gate skipped when `record.owner == ""` | **KILLED** |
| **F17** row 9 `Builtins()` copy | returns `builtinOrder` | **KILLED** |

F13's fix is the right shape: the detail pins now attribute the refusal to the
gate that fired, so a deleted gate no longer passes by falling through to the
next seam with the same code. Each of the three kills above names the *wrong*
downstream detail in its failure message, which is the proof.

### Additional attacks I added this round (not on round 3's list)

| Mutant | Result |
| --- | ---: |
| production emits a genuine fourth code (`not_supported`) | KILLED |
| unexercised production refusal site added | KILLED |
| stray `Error{}` literal in `provider.go` | KILLED |
| raw `fmt.Errorf` in `provider.go` | KILLED |
| `panic` in `provider.go` | KILLED |
| candidate digest not recorded (`scalar.Digest{}`) | KILLED |
| digest computed over the canonical path, not the bytes | KILLED |
| candidate `path` recorded as `canon` (disguised source path) | KILLED |
| `Verify` owner gate deleted outright | KILLED |
| `Verify` owner gate: `!Approves` clause dropped | KILLED |
| `Verify` owner gate: `identity != record.owner` clause dropped | KILLED |

### Equivalent survivors: two upgraded from *reasoned* to *proven*

Round 3 called five survivors equivalent on reasoning. Two are now proven by
construction, not judgement:

- **Row 36** (`digestBytes` length guard) and **row 38** (`unhex` lowercase-only)
  are equivalent **by construction**: `scalar.Digest` is only ever produced by
  `SHA256Digest` (`hex.EncodeToString`, lowercase) or `ParseDigest`, which
  rejects any non-lowercase-hex digit (`internal/scalar/time_digest.go:132-147`).
  No uppercase or non-hex digest value can exist, and
  `subtle.ConstantTimeCompare` returns 0 on a length mismatch. Row 37 follows
  from the same construction.
- **Rows 42 and 45** remain *reasoned*, not proven — see bounds.

## 4. Findings

All three are non-blocking, one-line-scale, and none is a defeat in the
delivered artifact. Recorded for the story's follow-up, not for another rework
round.

### F18 — the seam's stated invariant is unenforced (non-blocking)

`os.go:36-41` claims *"Production code never reassigns it"*; SEAM1 proves
nothing checks it, while the identical weakening at `fileOwnerUID` (CTRL1) is
killed. Closer, in the idiom this package already owns: either a four-line test
asserting `reflect.ValueOf(ownerAttester).Pointer() ==
reflect.ValueOf(fileOwnerUID).Pointer()`, or an extension of the source-derived
`auditRefusalInventory` scan refusing any production assignment to
`ownerAttester` outside its declaration.

### F19 — one of five `Discover` refusal returns is still unpinned for partial sets (non-blocking)

`return nil, …` → `return out, …` at the **`providers.plugin_dirs` absolute-path
refusal** (`provider.go:338`) survives; the other four return sites are killed.
`TestDiscoverRefusesRelativePluginDir` populates its absolute directories as
**empty** (`fake.entries[dir] = []string{}`), so `out` is nil at the refusal
regardless of the relative entry's index.

I proved the mutant is **not** equivalent rather than asserting it: with the
mutant applied plus one `ax-provider-foo` planted in `/abs/plugins`,
`Discover("/abs/plugins", "relative/plugins")` returns **1 candidate alongside
the refusal**. `doc.go` and `README.md:399` both promise the opposite. Closer:
one fixture — give the first absolute directory a real candidate in that test.

### F20 — the Windows check is never run by anything (non-blocking)

`os_windows_test.go` is a real improvement and `GOOS=windows go vet
./internal/provider` genuinely type-checks it — I verified that by planting a
type error in the file and watching vet report it at
`os_windows_test.go:32`. But `.github/workflows/ci.yml` runs only
`go test ./...`, `go vet ./...`, and `go build ./...` on `ubuntu-latest`: **no
step cross-compiles for Windows.** `os_windows.go` and `os_windows_test.go` are
invisible to CI, so breaking either leaves CI green. The compile-verification
cited in LOGBOOK is a one-off local run, not a repeatable check — the "check
present but uncalled" shape, one level up. Closer: one `GOOS=windows go vet
./...` step in `ci.yml`.

The README's Windows claim itself (`README.md:407-409`) is supported by
construction — `os_windows.go`'s `fileOwnerUID` returns an error
unconditionally — and LOGBOOK states the not-executed bound honestly. The gap
is the automation, not the claim.

## 5. Why this is accepted

- Both round-3 **blocking** findings (F11, F12) are closed, and closed in the
  right shape — F11 with a bidirectional inventory whose reverse check is
  derived from the test run rather than hand-listed, so truncation reddens even
  where the forward check passes vacuously. All four F11 mutants and six of the
  eight F12 rows redden.
- **Zero resurrections** over 45 re-run kills: nothing was loosened to close a
  survivor.
- Every gate that refuses, validates, authorizes, or attests has a **narrowing**
  negative mutant that reddens, not only a deletion form — and the three
  deletion-form survivors round 3 named are now killed by detail pins.
- The delivered artifact contains no bypass reachable by any caller. F18/F19 are
  coverage residuals against deliberate future edits, F20 is CI wiring; all
  three are one-liners and none changes shipped behaviour.
- Repo-wide `-race` green, provider coverage 97.0%, vet/build clean on darwin
  and under `GOOS=windows`.

## 6. What this method still cannot see (stated bounds, unchanged unless noted)

1. **Build-tag-invisible behaviour.** `os_windows.go` compiles but does not run
   here. Now compile-verified under `GOOS=windows go vet` — but see F20: no
   automation runs that. Behaviour still needs a Windows runner.
2. **No production caller exists.** `grep -rn '"…/internal/provider"'` outside
   the package returns nothing. "Production entry point" means the package's
   exported API, which the tests do drive through the real `OSSystem`
   (`TestOSSystemEndToEnd`, `TestOSSystemReadDirReportsFailure`). Whether the
   `ax` host wires `Discover`/`Trust`/`Verify` correctly is a later leaf and is
   **not** evidenced by this task. Unchanged from round 3.
3. **`Geteuid` vs `Getuid` is environment-equivalent here** (`euid=502
   uid=502`, probed). The test pins `OperatorUID` to the effective uid and kills
   the `{0}` wildcard; the euid/uid distinction itself is untested on this host.
   Reported as unknown, not inferred.
4. **TOCTOU.** Discovery reads owner (`Inspect`) and bytes (`ReadFile`) in two
   calls; a file swapped between them yields a digest of new bytes under an old
   attestation. Every userspace implementation has this race and the spec
   records no requirement against it — but this review did not probe it, and no
   test covers it. Unchanged from round 3.
5. **Equivalent survivors.** Rows 36/37/38 are now **proven** equivalent by
   `scalar.Digest`'s construction. Rows 42 (`filepath.Abs` fails only when
   `Getwd` does) and 45 (`Stat` vs `Lstat` on an already-canonical path) remain
   **reasoned, not proven**.
6. **Spec-quote pins read, not mutated.** Quote-anchored with `Contains` +
   section checks and `t.Fatalf` on miss, so not structurally vacuous, but I did
   not mutate `specdoc` itself. Unchanged from round 3.
7. **Mutation coverage is a lower bound.** 80/83 measured over ~54 gate rows
   says nothing about gates I did not think to write a mutant for.

## 7. Routing

`accept_cr(TASK-260830-2890sd, revision=5, evidence=TASK-260830-2890sd_review-verdict.md)`.
The element parks at `to-review`; the orchestrator checkpoints or integrates the
accepted revision and makes the `done` transition with
`commit_ack=scope_committed`. F18, F19, and F20 are handed forward as follow-up
for the story, not as rework on this leaf.
