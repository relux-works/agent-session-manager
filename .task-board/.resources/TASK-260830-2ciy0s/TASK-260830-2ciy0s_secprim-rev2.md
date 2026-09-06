# TASK-260830-2ciy0s — safe path/argv/environment primitives (outcome, rev2)

Round 2. Fixes all eight blocking findings of `TASK-260830-2ciy0s_review-verdict-rev1.md`
(B1–B8), including the two live vulnerabilities (B1 containment escape, B2 FIFO hang).
Work left UNCOMMITTED in the story worktree on top of trunk `157a54c`; no commits made.

## What changed since rev1

- `internal/secprim/nofollow.go` — `OpenNoFollowFile` clears `O_NONBLOCK` after
  admission; new `Guard.Open(root, member)` commit entry: validation (member
  grammar + managed set) and handle-relative commit in one entry point.
- `internal/secprim/open_member_unix.go` (new) — `openat` walk with `O_NOFOLLOW`
  per component, `ELOOP`/`ENOTDIR`-plus-`fstatat` escape classification, `O_NONBLOCK`
  final open, `clearNonblock` hook.
- `internal/secprim/open_member_windows.go` (new) — best-effort Lstat component
  walk with the stated check-then-open bound; refuses `ModeIrregular` reparse
  points, not just `ModeSymlink`.
- `internal/secprim/open_nofollow_unix.go` — `O_NONBLOCK` on the file open.
- `internal/secprim/open_nofollow_windows.go` — refuse `ModeIrregular` alongside
  `ModeSymlink`, with the Go-runtime-source reparse mapping stated.
- `internal/secprim/errors.go` — the single-construction-site rule now names the
  full shape space (no aliases, literals, `new(Error)`, factories).
- `internal/secprim/inventory_test.go` — derivation rewritten: alias fixpoint,
  `Error`-literal / `new(Error)` / factory-call / factory-decl / value-escape
  strays, plus `TestRefusalInventoryClosesBypassShapes` (control + 7 plants).
- `internal/secprim/classes_test.go` — recursive emission scan (110 files,
  fail-closed on parse errors); five `nopath` rows restated as archive-format
  bounds; census vocabulary bound stated.
- `internal/secprim/open_unix_test.go` (new, unix-only) — B1 escape fixture
  (lexical `Resolve` admits, commit `Open` refuses), honest-member read,
  six commit shapes, FIFO-through-opener with hang timeout.
- `internal/secprim/render_test.go` — 16-rune corpus fed from the same slice it
  asserts; per-rune rows for all previously unwitnessed code points.
- `internal/secprim/path_test.go` — managed-set exactness (`A/B` refused),
  empty-directory refusal, double-encoded overlong case, seam-driven
  `Guard.Open` stat-failure arm.
- `internal/secprim/redact_test.go` — compact-colon case (first-pass `:` arm's
  unique shape).
- `internal/cliresult/render_wiring_test.go` — success-`Emit`, `Progress`, and
  `Prompt`-refusal hostile-content tests (all six text paths now driven).
- `internal/provhost/runner_env_test.go` — `TestExecRunnerEmptyEnvIsDenyAll`
  through `/usr/bin/env` (nil vs empty as distinct obligations).
- `README.md` — claims corrected to the proven bounds (see below).

## B1 — containment now composes (was: live escape)

Vector: `NewGuard(stage)` + `stage/sub -> outside` symlink + `Resolve("sub/loot")`
returned `stage/sub/loot` and `OpenNoFollowFile` read `ESCAPED` outside the root.
`O_NOFOLLOW` constrains only the final component; `Resolve` answers the lexical
question only.

Fix: `Guard.Open(root *os.File, member string)` validates (grammar + managed set)
then walks `openat` per component from the `OpenNoFollowDir` handle with
`O_NOFOLLOW` on every component — no path string is re-resolved. A symlinked
intermediate fails the walk and refuses as `member symlink escape` (commit half
of §16.3); a re-validated string is not used anywhere on this path. Darwin
reports `ENOTDIR` (not `ELOOP`) for `O_NOFOLLOW|O_DIRECTORY` meeting a symlink,
so either errno re-examines the component with `fstatat(AT_SYMLINK_NOFOLLOW)`,
still descriptor-relative; any swap between the failed open and the check fails
closed (every outcome refuses, only the rule name differs).

Driven by: `TestGuardOpenRefusesSymlinkedParent` (pins `Resolve` admitting the
lexical join AND `Open` refusing it), `TestGuardOpenAdmitsHonestMembers`,
`TestGuardOpenRefusesCommitShapes` (final symlink, missing member/intermediate,
unmanaged, grammar, nil/file root handles), N23/N24b/D12/D13 mutants below.

## B2 — opener refuses FIFOs (was: hangs forever)

`open(2)` read-only on a writer-less FIFO blocks in the kernel before the
post-open shape check runs. Fix: `unix.O_NONBLOCK` on the unix file open and on
the commit-walk final open; the post-open `CheckRegularTarget` refusal is now
reachable, and admitted regular files are cleared of the flag.

Driven by: `TestOpenNoFollowFileRefusesFifoWithoutBlocking` — drives the
production opener in a goroutine with a 10 s timeout, so a regression fails
instead of stalling the suite. `fifo_unix_test.go` still covers the pure
predicate; the opener behavior is covered here.

## B3 — all six cliresult sites driven (was: 3 of 6)

`Log`, interactive `Prompt`, failure `Emit` were driven. Added:
`TestTextSuccessEmissionNeutralizes` (success `Emit` → stdout, R27),
`TestProgressEmissionNeutralizes` (TTY `Progress`, R23),
`TestPromptRefusalCarriesNeutralizedText` (non-interactive refusal text with a
hand-computed neutralized expectation, R24 — an unwired site fails the exact
comparison rather than hiding behind `%q`). README now enumerates the six
instead of claiming "every text path" uncovered.

## B4 — nil vs empty allowlist (was: surviving R26)

`TestExecRunnerEmptyEnvIsDenyAll` runs `/usr/bin/env` through
`ExecRunner{Env: []string{}}` and requires a byte-empty child environment, so
no helper marker travels in the environment under test. Together with
`TestExecRunnerNilEnvInherits` the `!= nil` boundary is pinned from both sides:
under the `len(Env) > 0` mutant the parent-only marker leaks and the test fails
(N22). Unix execution check; skipped on Windows (compile-and-vet only there).

## B5 — inventory gate closed over the shape space (was: 4 of 4 bypasses walked through)

`deriveRefusals` replaces the identifier-keyed walk: constructor-alias fixpoint
(`var`, `:=`, `=`, alias-of-alias), `Error` literals (direct and `secprim.`
qualified) outside constructor bodies, `new(Error)`, calls to package `*Error`
factories (functions and methods; `WithCause` is the one allowlisted decorator),
factory declarations whose body never reaches a constructor, and constructor
names escaping as values. Aliases resolve to their underlying constructor so
exemptions keep working.

Evidence, two layers. Committed: `TestRefusalInventoryClosesBypassShapes` feeds
all eight shapes through the same `deriveRefusals` the audit uses — control
(bare call, clean site), A (var alias), B (literal), C (method factory: decl +
call + literal strays), D (local binding), E (value escape), F (`new(Error)`),
G (alias of alias). Live (this run, reverted after, tree verified identical):
control + reviewer plants A–D written as real production files, full suite run
per plant — CONTROL caught (`never fired`), A caught (`alias`), B caught
(`Error literal`), C caught (`factory`), D caught (`alias`).

## B6 — corpus feeds all 16 (was: 10 fed, 16 asserted)

`TestEscapeForTerminalOutputBytes` feeds the `forbidden` slice it asserts, so
feed and assertion cannot drift again; per-rune table rows added for U+202A–
U+202D, U+2068, U+2069, U+200E, U+061C, U+200C. R9b/R7/R7c now kill (N8/N9).

## B7 — recursive scan, honest rows (was: one level, five overstated rows)

`TestNoArchiveEmissionPath` walks all of `internal/` recursively (110 production
files, up from the one-level scan), fails closed on unreadable/unparseable
files, and names violating paths. Live probe this run: `archive/zip` planted in
`internal/catalog/cmd/cataloggen` is caught with its path (reverted after).
The five `nopath` dispositions now claim exactly the proven bound (no
archive-writer import anywhere under `internal/`, nested packages included —
not the absence of every non-archive emission shape), and the Machine
authentication row states the census vocabulary bound (13 tokens; names outside
it are outside the witness).

## B8 — denominator and honest mutant kinds (was: 6 of 35, deletions as narrowing)

Derived refusal exits: **41** (argv 5, env 9, nofollow 14 = 8 rev1 + 6 new
`Guard.Open` sites, path 13). Battery below: every row applied to the working
tree and run as a standalone process in this session; the tree was verified
byte-identical after (sha256 manifest over all touched files; only expected
delta is the compact-colon test added before the manifest… correction: the
manifest was recorded before the compact-colon insert, so that one file differs
by exactly that insert — confirmed by inspection; no `.mutbak`/`zz_*` residue).

### Narrowing — 27 applied, 27 killed (killed/applied = 27/27)

| Mutant | Admits exactly | Named failing test |
|---|---|---|
| N1 drop `%25`-unwrap on encoded-dot arm | doubly-encoded dots | TestCheckMemberPathRefuses/double-encoded_dot |
| N2 drop `%25`-unwrap on overlong arm | doubly-encoded overlong separators | TestCheckMemberPathRefuses/double-encoded_overlong |
| N3 ADS `:` → `::` | single-colon streams | .../windows_ads(+nested) |
| N4 drop trailing-space disjunct | trailing-space segments | .../windows_trailing_space |
| N5 segment loop `[:1]` | non-first-segment violations | .../windows_lpt |
| N6 managed lookup `ToLower` | case variants of managed members | TestGuardManagedSet |
| N7b drop NamedPipe disjunct + irregular catch-all (two-arm) | FIFOs | TestCheckRegularTarget |
| N8 bidi lower `0x202a`→`0x202b` | U+202A | TestEscapeForTerminalOutputBytes |
| N9 isolate upper `0x2069`→`0x2068` | U+2069 | TestEscapeForTerminalOutputBytes |
| N10 C1 upper `0x9f`→`0x9e` | U+009F | TestEscapeForTerminalOutputBytes |
| N11 C0 `<0x20`→`<0x1f` | U+001F | TestEscapeForTerminalOutputBytes |
| N12 argv NUL first-element-only | NUL past argv[0] | .../nul_argument |
| N13 argv encoding first-element-only | invalid UTF-8 past argv[0] | .../invalid_utf8 |
| N14 argv empty first-element-only | empty past argv[0] | .../empty_element |
| N15 env name length 128→129 | 129-rune names | TestIsEnvName |
| N16 env name leading digit | 9LIVES | TestIsEnvName |
| N17 minSecretRunes 8→9 | 8-rune secrets | TestRedactCorpusValues |
| N18 PEM suffix widened (WIDENING) | non-private BEGIN blocks | TestRedactPrivateKeyBlocks |
| N19 success-Emit wire identity (R27) | raw stdout | TestTextSuccessEmissionNeutralizes |
| N20 Progress wire identity (R23) | raw progress | TestProgressEmissionNeutralizes |
| N21 Prompt-refusal wire identity (R24) | raw refusal text | TestPromptRefusalCarriesNeutralizedText |
| N22 runner Env `!=nil`→`len>0` (R26) | empty allowlist inherits | TestExecRunnerEmptyEnvIsDenyAll |
| N23 ELOOP-only escape classification | Darwin misreport as open-failed | TestGuardOpenRefusesSymlinkedParent |
| N24b drop directory arm + catch-all (two-arm) | directories | TestCheckRegularTarget |
| N25 argv `len==0`→`len<0` | empty vectors | .../empty_vector |
| N26 first-pass sep `"="` only | compact colon pairs | .../compact_colon |
| N27 drop DEL from control case | DEL | .../del |

### Deletion (existence probes) — 15 applied, 15 killed

D1 duplicate-allowlist arm → duplicate_allowed; D2 literal-collision arm →
literal_collides; D3 one scrubber-table entry → TestSensitiveKeyListIsReviewed;
D5 managed-set arm → TestGuardManagedSet; D6 encoded-dot arm → encoded_dot;
D10 Log-only wire → hostile-content test; D11 runner.Env ignored → built-env
test; D12 final O_NOFOLLOW flag → final_symlink; D13 intermediate O_NOFOLLOW
flag → symlinked-parent test; D14b argv gate retirado (narrow re-application
keeping the import) → TestExecRunnerRefusesUnusableExecutable with
`Run("") = *errors.errorString exec: no command, want *secprim.Error`;
D15 userinfo arm → TestRedactURLUserinfo; D16 corpus arm →
TestRedactCorpusValues; D17 nil-stat arm → seam test (killed via panic);
D18 guard-root delegation → TestGuardResolve; D19 kernel-open-error ignore →
TestOpenNoFollowFile (killed via panic).

### Instrument probes — 2 applied, 2 killed; bypass probe — 1 applied, 1 killed

D8 census token drop → TestSecretSiteRosterIsComplete (orphans); D9 class-row
rename → TestExclusionClassRosterIsComplete (orphans); D7 token-preserving
shell (`"s"+"h"` + `os/exec`, no literal token) → TestPackageLaunchesNoProcess
(behavioral half).

### Survivors with stated bounds — 2

- N7 (drop NamedPipe disjunct only): SURVIVES — the `!mode.IsRegular()`
  catch-all subsum
...[truncated 5836 chars]
