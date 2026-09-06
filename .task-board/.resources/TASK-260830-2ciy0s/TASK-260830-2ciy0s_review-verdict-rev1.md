# TASK-260830-2ciy0s — reviewer verdict, CR revision 1

- Run: `RUN-260906-43ee79` (reviewer/reviewer)
- Change Request: `CR-TASK-260830-2ciy0s-1` rev 1, `kind=task_delta`,
  base `157a54c`, candidate tree `9f821f3e5602137d3c8470291394d011df1b6f05`,
  `repository_delta=present`, 30 changed paths.
- Verdict: **changes requested** → `to-dev`.
- `repeat-of:` **none** (first round of this leaf).
- Worktree verified byte-identical to the candidate tree before and after
  every probe (`git read-tree HEAD` into a temp index + `git add -A` +
  `write-tree` → `9f821f3e…`). All mutants and plants reverted.

## What I re-ran myself (not accepted from the producer)

| Gate | Command | Result |
| --- | --- | ---: |
| Full suite | `go test ./... -count=1` | exit 0, 20 packages ok |
| Format | `gofmt -l internal/` | clean |
| Vet | `go vet ./...` | exit 0 |
| Windows vet | `GOOS=windows go vet ./...` | exit 0 |
| Windows build | `GOOS=windows go build ./...` | exit 0 |

Accepted from the producer without re-running: `-race`, `-cover`
(secprim 94.5%), `tracecheck`. Everything else below I executed.

## Blocking findings

### B1 — Containment and no-follow do not compose: a symlinked intermediate component escapes the guard root

Driven through the real entry points, no mocks:

```
NewGuard("<tmp>/stage", PlatformLinux, nil)   -> ok
os.Symlink("<tmp>/outside", "<tmp>/stage/sub")
Guard.Resolve("sub/loot")                     -> "<tmp>/stage/sub/loot", err=nil
OpenNoFollowFile("<tmp>/stage/sub/loot")      -> ADMITTED, read "ESCAPED"
```

`O_NOFOLLOW` constrains the **final** component only. `Guard.Resolve` is
purely lexical. Composing them therefore does not produce containment:
any hostile intermediate directory that is a symlink is traversed by the
open, and the read leaves the root.

Section 16.3 requires refusal of "symlink/reparse escape during **both**
validation and commit" and TOCTOU prevention "by validating **through
directory handles** or equivalent safe filesystem primitives". The
package exposes no `openat`-relative API: `OpenNoFollowDir` returns an
`*os.File` that nothing in the package or the repository ever opens
through. `nofollow.go` states the mitigation as "Parent directories are
the caller's responsibility: open the staging root with `OpenNoFollowDir`
and resolve members through a `Guard`" — that composition is not
implementable with the shipped surface, because there is no entry point
that takes the directory handle.

The outcome doc lists "symlink/reparse escape at validation and commit"
and "TOCTOU dir-handle primitives (unix)" among the 13 rows **"fully
driven through production entries"**. Neither is: the property does not
hold, and no test drives a symlinked parent.

Named fix: add a handle-relative open (`unix.Openat` with
`O_NOFOLLOW|O_CLOEXEC` per component, walking `Guard.Resolve`'s segments
from an `OpenNoFollowDir` root handle), and drive the escape fixture
above as a negative test. Windows keeps its stated check-then-open bound.

### B2 — `OpenNoFollowFile` on a FIFO blocks forever instead of refusing

```
mkfifo("<tmp>/stage/pipe")
OpenNoFollowFile("<tmp>/stage/pipe")  -> *** still blocked after 3s ***
```

A read-only `open(2)` on a FIFO with no writer blocks until one appears.
`CheckRegularTarget` runs **after** the open and is never reached, so the
"device/FIFO/socket" refusal that `path.go` implements is unreachable
through the opener. A hostile staging tree containing one FIFO stalls the
reader indefinitely.

`fifo_unix_test.go` proves what it proves and no more: it builds a FIFO
and feeds the resulting `os.Lstat` **info** to `CheckRegularTarget`. It
never calls `OpenNoFollowFile` on the FIFO. So the pure function is
covered and the production opener is not — and the production opener is
where the claim is false. The outcome doc lists "device/FIFO/socket
refusal" as fully driven through production entries; it is not driven
through the opener at all.

Named fix: add `unix.O_NONBLOCK` to the unix open flags (POSIX: a
read-only `O_NONBLOCK` open of a FIFO returns immediately), keep the
post-open `CheckRegularTarget`, clear `O_NONBLOCK` on the descriptor if
the target turns out to be a regular file, and drive the FIFO through
`OpenNoFollowFile` with a test that would hang without the fix.

### B3 — 3 of 6 `cliresult` wiring sites are unmeasured, including the stdout success path

Narrowing mutants I applied and ran (`terminalLine(x)` → `x` at one site):

| Site | Mutant | Result |
| --- | --- | ---: |
| `Emit` failure → stderr | R22 | KILLED (`TestTextEmissionNeutralizesHostileContent`) |
| `Log` / `Prompt` write | R28 | not applicable (shared literal); driven by the same test |
| **`Emit` success → stdout** | **R27** | **SURVIVED** |
| **`Progress`** | **R23** | **SURVIVED** |
| **`Prompt` non-interactive refusal text** | **R24** | **SURVIVED** |

`render_wiring_test.go` drives `Log`, interactive `Prompt`, and failure
`Emit`. Success `Emit` — the primary human stdout path — `Progress`, and
the `ErrPromptForbidden` message are wired but unwitnessed. The README
claims escaping is "wired into every `cliresult` text path"; the code is
right today and only half of it would redden if that stopped being true.

### B4 — `ExecRunner.Env` nil-vs-empty is unmeasured, and the deny-all allowlist is the case that breaks

Mutant R26: `if runner.Env != nil` → `if len(runner.Env) > 0`. **SURVIVED**
the full suite.

`runner.go` names this exact distinction as the design point ("when nil
the child inherits …; when non-nil the child observes exactly `Env`"),
and `env.go` makes the same argument for `Guard`'s managed set ("denying
by default is representable and distinct from not checking"). Under the
surviving mutant an operator who supplies an empty allowlist — the
deny-all case, the whole point of an allowlist — silently gets full
parent-environment inheritance. Add `TestExecRunnerDeliversExactlyTheBuiltEnv`
sibling driving `ExecRunner{Env: []string{}}` and asserting an empty
child environment.

### B5 — the refusal-inventory gate is identifier-keyed; 4 of 4 planted bypasses walk through

`auditRefusalInventory` derives sites by matching `call.Fun.(*ast.Ident)`
against the three names `failPath`/`failArgv`/`failEnv`. Every plant below
adds an **unexercised** production refusal; the control proves the
instrument works.

| Plant | Shape | Result |
| --- | --- | ---: |
| CONTROL | `failArgv("plant control", …)` | **CAUGHT** — `refusal site argv.go:84 never fired` |
| A | `var failArgvAlias = failArgv` then `failArgvAlias(…)` | BYPASSED |
| B | `return &Error{Kind: ErrUnsafeArgv, …}` | BYPASSED |
| C | `plantMaker.argv(…)` (method selector) | BYPASSED |
| D | `make := failArgv` then `make(…)` | BYPASSED |

So "every refusal exit fired during the suite" is true only of refusals
spelled with the bare constructor identifier. `errors.go` states the rule
("production code must not build `Error` values any other way") but
nothing enforces it — B is a direct, invisible violation of the stated
rule. This is the same identifier-keyed-AST-gate class the previous story
had to close; it is unrepaired here. Close it by typing on the resolved
callee (or by keying on the returned `*Error` construction) and re-run
all four plants as the acceptance evidence.

### B6 — `TestEscapeForTerminalOutputBytes` asserts on 16 code points but feeds 10; six assertions are vacuous

The `forbidden` loop enumerates `202A 202B 202C 202D 202E 2066 2067 2068
2069 200E 200F 061C 200B 200C 200D FEFF`. The corpus it escapes contains
only `202E 2066 2067 200E 200F 061C 200B 200C 200D FEFF`. The other six
are asserted absent from an output they were never present in.

Confirmed by narrowing mutants on `render.go`:

| Mutant | Narrowed to admit | Result |
| --- | --- | ---: |
| R4 `< 0x20` → `< 0x1f` | U+001F | KILLED |
| R5 C1 upper `<= 0x9f` → `<= 0x9e` | U+009F | KILLED |
| R6 C1 lower `>= 0x80` → `> 0x80` | U+0080 | KILLED |
| R8 zero-width `<= 0x200d` → `<= 0x200c` | U+200D | KILLED |
| R9 override `<= 0x202e` → `<= 0x202d` | U+202E | KILLED |
| R10 drop U+200F | U+200F | KILLED |
| R11 `0xfeff` → `0xfefe` | U+FEFF | KILLED |
| **R9b override `>= 0x202a` → `>= 0x202b`** | **U+202A** | **SURVIVED** |
| **R7 isolate `<= 0x2069` → `<= 0x2068`** | **U+2069** | **SURVIVED** |
| **R7c isolate `<= 0x2069` → `<= 0x2067`** | **U+2068, U+2069** | **SURVIVED** |

Each bidi range is witnessed at exactly one endpoint. U+202A–U+202D (four
fifths of the Trojan-Source embedding/override set) and U+2068–U+2069 are
unmeasured. Fix is one line: put all sixteen runes in the corpus.

### B7 — `TestNoArchiveEmissionPath` never scans nested production packages, and it is the sole witness for five `nopath` dispositions

The walk reads `internal/<pkg>/*.go` one level deep and `continue`s on any
`entry.IsDir()`. This repository already contains nested production
packages (`internal/catalog/cmd/cataloggen`, `internal/traceability/cmd/tracecheck`).

| Plant | Location | Result |
| --- | --- | ---: |
| CONTROL | `import "archive/zip"` in `internal/scalar/path.go` | **CAUGHT** |
| NESTED | identical import in `internal/catalog/cmd/cataloggen/main.go` | **BYPASSED** |

`classes_test.go` cites this test as the proof for the `nopath`
disposition of five of the nine Section 16.2 classes (Machine
authentication, Live process identity, IPC/runtime, Transient locking,
Mutable derived indexes). Beyond the directory blind spot, what it
actually establishes is narrower than what the rows claim: it proves that
no scanned file imports `archive/tar`, `archive/zip`, or `compress/gzip`.
Section 16.2 governs "manifest **or bundle**" emission, and a JSON
manifest needs none of those imports. Recursive-walk the tree, and state
the row as the bound it is ("no archive-format emission path") rather
than as "no in-repo emission path carries this class".

### B8 — the mutant battery is mostly deletions, and its denominator is 6 of 35

Re-derived from production with the same AST shape the inventory gate
uses: `internal/secprim` has **35 refusal exits** (argv 5, env 9,
nofollow 8, path 13). Mapping the producer's table onto them, the battery
narrows six — M1 (`member encoded dot`), M2 (`member alternate stream`),
M3 (`member reserved device name`), M4 (`argv NUL`), M5 (`env literal
collision`), M10 (`member unmanaged`) — i.e. **6 of 35 = 17%**. M6–M9 hit
the render/redact arms, M11/M12/M15 mutate **test instruments** rather
than production gates, M13b/M14 hit wiring.

Shape: twelve of the fifteen rows are arm deletions written in the
"narrows the gate to" column. "Drop the encoded-dot arm" admits the whole
encoded-dot class, not one member of it; that is a delete mutant with a
narrowing label. Only M9 (widening the PEM match) and M11
(token-preserving `"s"+"h"`) are genuinely non-deletion. Per the DoD row,
a delete-only mutant proves the gate exists and says nothing about the
class boundary.

M13a is correctly reported as a separate COMPILE_FAIL row and is not
folded into the kill count — that part is honest.

I ran 20 additional mutants against the unmeasured majority. Fourteen
were killed, confirming the bounds below are real: `IsEnvName` length at
exactly 128/129 (R1) and its leading-digit guard (R2), `minSecretRunes`
at exactly 8 (R3), the `%25`-unwrap on both the encoded-dot and overlong
arms (R13/R14), the Windows trailing-space arm independently of the
trailing-dot arm (R15), `EqualFold` vs `==` in case-collision (R16), the
argv UTF-8 and empty-element arms (R17/R18), the PEM `PRIVATE KEY` suffix
requirement (R19), URL userinfo (R20), the corpus arm (R21), the
`ExecRunner` argv gate (R25b), and the four render mutants above. Five
survived and are B3/B4/B6. One (R12, dropping the slash boundary from
`withinRoot`) survived consistently with the producer's own declared
`defensiveSites` bound — that one is not a finding.

## What passed, and passed well

- **G-D, no third trust path: confirmed.** I read both cited call sites.
  `provider.trustCandidate` does `Canonicalize` → `Inspect` → `IsRegular`
  → `owner.Approves(info.UID)` → digest; `terminalbackend.DigestFile`
  does `EvalSymlinks` → `Stat` → `IsRegular` → digest with **no owner
  fact**. The Section 7.1 / Section 4.B difference the doc cites is real
  and visible in the code, not asserted. `secprim` adds no canonicalise,
  no digest, and no owner check — there is no third path.
- **The Section 16.2 roster derivation is real and fails closed.** It
  parses the Class table out of the pinned `SPEC.md`, `t.Fatal`s on zero
  derived classes, and checks both directions against the disposition
  rows. Nine derived, nine dispositioned.
- **The secret-site census fails closed for its vocabulary.** Control
  plant `var PlantApiKeyStore = "sk-live-…"` in production →
  `derived secret sites with no disposition row: secprim|decl|PlantApiKeyStore`.
  Plants outside the 13-token vocabulary (`PlantGithubPAT`,
  `PlantBearerMaterial`, `PlantKeychainItem`) are not rostered — that is
  inherent to a token instrument, but see the note below.
- **The `Redact` shapes the spawn brief flagged are all inside stated
  bounds.** `GITHUB_TOKEN=ghp_…` and `{"api_secret":"…"}` are compound
  keys; `redact.go` states exactly that residue ("Compound keys outside
  this list (for example `db_password`) are not matched"), and the
  exact-match-not-substring rationale is sound (substring matching would
  redact `token_count`). `AKIAIOSFODNN7EXAMPLE` is a bare credential with
  no key shape — also stated, and it is a key *identifier*, not the
  secret. Section 16.2 itself disclaims reliable content-level scrubbing
  and the function claims no more. The asymmetry the brief noticed
  (`password=` caught, `GITHUB_TOKEN=` not) is explained by the table,
  not by a hole in the detector. Advisory only: since `provhost` env
  names are `[A-Za-z_][A-Za-z0-9_]{0,127}`, a suffix rule on
  `_token`/`_key`/`_secret`/`_password` would cover the dominant
  environment-secret spelling at near-zero false-positive cost.
- Suite, format, vet, Windows vet and Windows build all green under my
  own execution.

## Bounds I am recording as unknown rather than inferring

- **Windows is asserted, not executed.** `open_nofollow_windows.go` is
  proven to compile and vet under `GOOS=windows` and nothing more. No
  Windows behavior in this CR has been executed anywhere.
  `priv_windows_test.go` closes none of it — it supplies a
  `privilegedUser()` returning `false` and a `makeFifo` that skips. The
  check-then-open TOCTOU bound is stated honestly in the source.
  Separately: the Windows opener refuses only `os.ModeSymlink`, while
  Section 16.3 says "symlink/**reparse** escape". Whether Go's
  `os.Lstat` sets `ModeSymlink` for every Windows reparse point
  (junctions, mount points, AppExecLink, cloud placeholders) is **not
  established** by any evidence here and I did not infer it — it needs a
  Windows run or a cited Go guarantee.
- **The census blind spot is wider than the row that states it.**
  Residue 4 and `TestSecretScanBlindSpot` state the blind spot as
  "`auth`-substring declarations". The instrument is in fact blind to
  every name outside its 13 tokens, demonstrated above with `ghp_`, JWT,
  and keychain plants. That matters because the **Machine
  authentication** `nopath` row cites `TestSecretSiteRosterIsComplete` as
  proving "no secret-named field exists outside the dispositioned
  tables"; it proves "no field whose name contains one of 13 tokens".
  Restate the row to the bound the witness actually supports.

## AC-row coverage, as measured

The producer reports 16 of 25 Section 16 MUST rows driven. Three of the
thirteen rows claimed "fully driven through production entries" do not
survive execution: symlink/reparse escape at commit (B1), TOCTOU
dir-handle primitives (B1 — the handle is returned and never used), and
device/FIFO/socket refusal (B2 — driven through `CheckRegularTarget`,
never through the opener, and false there). Corrected: **13 of 25 driven,
3 claimed-but-falsified, 9 bounds/pre-existing.**

## Required for revision 2

1. B1 — handle-relative no-follow walk so containment and no-follow
   compose; negative test on the symlinked-parent escape fixture.
2. B2 — `O_NONBLOCK` on the unix open; FIFO driven through
   `OpenNoFollowFile`.
3. B3 — witness the three unmeasured `cliresult` sites (success `Emit`,
   `Progress`, `Prompt` refusal).
4. B4 — witness `ExecRunner{Env: []string{}}` as deny-all.
5. B5 — close the identifier-keyed inventory gate; re-run all four
   plants plus the control as acceptance evidence.
6. B6 — feed all sixteen enumerated runes into the escape corpus.
7. B7 — recurse the emission-path walk; restate the five `nopath` rows to
   the bound the witness supports.
8. B8 — report the mutant denominator against the 35 derived refusal
   exits, and replace delete-labelled-as-narrowing rows with real
   boundary moves.

Correct the outcome doc and the README to match, in particular
"fully driven through production entries" and "wired into every
`cliresult` text path".
