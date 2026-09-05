# TASK-260830-2890sd — review round 3 verdict

**Verdict: changes requested → `to-dev`.**

Reviewed: worktree `.temp/STORY-260830-3jqsx1/worktree`, branch
`task-board/story/STORY-260830-3jqsx1`, HEAD `44a4699` (round-2 rework;
`d8fc669` beneath it). Repo-wide `go test ./...` green, `go build ./...` clean,
`go vet ./internal/provider` clean. Working tree restored and verified clean
after every mutant (`git status --short` empty).

## 0. The empty `repository_delta`

`CR-TASK-260830-2890sd-4` reports `repository_delta=empty`: base OID
`44a4699` == candidate tree `4427790` == `HEAD^{tree}`. This is a
CR-construction artifact, not a producer that changed nothing. The round-2
rework **is** `44a4699` (+510 lines: `owner_root_test.go`,
`digest_sweep_test.go`, `source_inventory_test.go`, `discovery_test.go`,
`LOGBOOK.md`); the CR base was snapshotted at that commit instead of at the
pre-rework checkpoint `d8fc669`, so the reviewable delta collapsed to nothing.
I reviewed `git diff d8fc669..44a4699` plus the whole of `internal/provider`
rather than the empty patch. **Verdict note for the orchestrator: the
checkpoint that produced this CR is stale by one commit — the same
`checkpoint_oid`-behind-leaf shape this story has hit before. Fix the
checkpoint before the next CR; the review itself was not blocked by it.**

## 1. Round-2 findings F6–F10: confirmed closed

Confirmed cheaply with my own mutants, per the brief. All four are genuinely
closed, not re-declared.

| Finding | Mutant | Result |
| --- | --- | --- |
| F6 uid 0 admitted | `if uid == 0 { return true }` in `OwnerPolicy.Approves` | killed — `TestOwnerPolicyTreatsRootWithoutException` |
| F7 abs-path gate at index 0 only | gate wrapped in `&& index == 0` (both `plugin_dirs` and `PATH`) | killed — `TestDiscoverRefusesRelativePluginDir`, `TestDiscoverRefusesRelativePATHDir` |
| F8 two LOGBOOK entries unlanded | `git show 44a4699 --name-only` | landed, 21 lines |
| F9 ratio compared a literal to itself | fourth production source site added to `Discover` | killed — `TestTrustGateSourceInventoryIsDerivedFromDiscover` |
| F10 digest gate on a prefix | `sum[:1]` and `sum[31:]` comparisons | killed — `TestVerifyRefusesDigestChangeAtEveryByteIndex` (32/32 sweep) |

## 2. G-A — does the derivation fail closed on an empty domain?

**For `source_inventory_test.go`: yes.** Five mutants, five kills.

| # | Mutant on the derivation / production | Result | Killed by |
| --- | --- | --- | --- |
| A1 | `deriveDiscoverSourceSites` returns `discoverSourceSites{}` | killed | `inlineBuiltins = 0, want exactly 1` |
| A2 | derivation truncates `collectClasses` to 1 of 2 real sites | killed | `1 collectDirectory sites [plugin_dirs], want exactly 2` |
| A3 | production bypass source: direct `ReadDir` + `externalID` + inline `KindExternal` in `Discover` | killed | `reads 1 directories outside collectDirectory` |
| A4 | inline `KindExternal` candidate only (no `ReadDir`) | killed | `builds 1 external candidates outside trustCandidate` + 6 behavioural tests |
| A5 | direct `externalID` call in `Discover` only | killed | `parses 1 executable names outside collectDirectory` |

The exact-count assertions (`inlineBuiltins != 1`, `len(collectClasses) != 2`,
the sorted-label equality) are the floor: an empty or short derivation cannot
pass. Each of the three zero-expected counters (`directReads`,
`directNameParses`, `inlineExternals`) is individually live — A3/A4/A5 each
redden a different one. An unreadable or unparseable `provider.go`, and a
missing `func Discover`, are `t.Fatalf`/`t.Fatal`. **G-A is answered for the
inventory it was asked about.**

**But the same question, asked of the package's other derived domains, finds
one that answers no — see F11.**

## 3. G-B — forced traversal of every gate in `internal/provider`

Discovered by walking the production source (`provider.go`, `os.go`,
`os_unix.go`, `os_windows.go`), not by recalling earlier rounds. Every row was
attacked with at least one mutant; narrowing mutants were preferred over
deletion where the gate has a class to narrow.

**Buckets:** `full` = a narrowing mutant reddens; `partial` = narrowing
reddens but deletion or a neighbouring form survives; `sliver` = the gate is
tested on one point of its domain only; `unevidenced` = no mutant on it
reddens anything.

### Discover / configuration

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 1 | `plugin_dirs[i]` absolute-path refusal | narrowed to `index == 0`; deleted | killed | full |
| 2 | `PATH[i]` absolute-path refusal | narrowed to `index == 0` | killed | full |
| 3 | `allow_path_plugins` gate | `if true` (PATH always scanned) | killed | full |
| 4 | duplicate-ID refusal | narrowed to same-source; byte-identical override added | killed | full |
| 5 | source enumeration order | builtins hoisted above `plugin_dirs` | killed | full |
| 6 | builtin registry contents / order | dropped `pi`; swapped `codex`/`claude` | killed | full |
| 7 | bytewise entry sort within a directory | `sort.Strings` removed | killed | full |
| 8 | `Discover` returns no partial set with a refusal | `return nil, err` → `return out, err` | **SURVIVED** | **unevidenced** |
| 9 | `Builtins()` returns a copy | returns `builtinOrder` directly | **SURVIVED** | **unevidenced** |

### Executable name grammar

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 10 | `ExecutablePrefix` constant value | loosened to `"ax-"` | killed | full |
| 11 | prefix is an **anchor**, not a substring | `strings.CutPrefix` → `strings.Cut` | **SURVIVED** | **sliver** |
| 12 | provider-id grammar refusal | malformed name skipped; malformed name accepted | killed | full |

### `trustCandidate`

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 13 | `ReadDir` failure → `local_precondition_failed` | error swallowed as empty directory | killed | full |
| 14 | `Canonicalize` failure → precondition | narrowed to `/opt` prefix | killed | partial |
| 14b | same, deletion form | `canon = path` on error | **SURVIVED** | (see F13) |
| 15 | `Inspect` failure → precondition | narrowed to `/zzz` prefix | killed | partial |
| 15b | same, deletion form | fabricate `{IsRegular:true, UID:operator}` | **SURVIVED** | (see F13) |
| 16 | regular-file target | gate deleted | killed | full |
| 17 | approved owner, on every source | deleted; narrowed to `source != "path"` | killed | full |
| 18 | `ReadFile` failure → precondition | error swallowed, empty content digested | killed | full |

### `OwnerPolicy`

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 19 | operator-UID approval | branch deleted | killed | full |
| 20 | administrator-UID approval | branch deleted; loosened to "any uid if the set is non-empty" | killed | full |
| 21 | no superuser exception | `if uid == 0 { return true }` | killed | full |
| 22 | `OwnerIdentity` carries the uid | returns constant `"uid:approved"` | killed | full |

### `Trust`

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 23 | builtin refusal | gate deleted | killed | full |
| 24 | records approving owner | `owner: ""` | killed | full |
| 25 | records undisguised source path | `sourcePath: candidate.canon` | killed | full |
| 26 | records trust instant | `trustedAt: scalar.Timestamp{}` | killed | full |

### `Verify`

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 27 | re-resolve failure → `integrity_failure` | narrowed to `/zzz`; deleted | killed | full |
| 28 | canonical-target retarget | gate deleted | killed | full |
| 29 | re-inspect failure → integrity | narrowed to `/zzz` | killed | partial |
| 29b | same, deletion form | fabricate `{IsRegular:true, UID:operator}` | **SURVIVED** | (see F13) |
| 30 | regular-file → integrity | gate deleted | killed | full |
| 31 | owner-**approval** half of the owner gate | `!Approves` clause dropped | killed | full |
| 32 | **recorded**-owner half of the owner gate | `identity != record.owner` clause dropped | killed | full |
| 33 | owner gate holds for a zero-value receipt | skip when `record.owner == ""` | **SURVIVED** | **sliver** |
| 34 | re-read failure → integrity | narrowed to `/zzz`; `return nil` | killed | full |
| 35 | digest comparison over all 32 bytes | compare `sum[:1]`; compare `sum[31:]` | killed | full |
| 36 | `digestBytes` length guard | `< size` + zero-fill fallback | survived — **equivalent** (both forms compare unequal) |
| 37 | `digestBytes` non-hex guard | guard removed | survived — **equivalent** (garbage bytes still compare unequal) |
| 38 | `unhex` lowercase-only | uppercase accepted | survived — **equivalent** (`scalar.Digest.Hex()` is lowercase by construction) |

### `Candidate` absence accessors

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 39 | `SourcePath`/`CanonicalPath`/`Digest`/`Owner` report absence for builtins | each `kind != KindExternal` guard neutered | killed (4/4) | full |

### `OSSystem` — the production filesystem seam

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 40 | `ReadDir` error propagation | `return nil, nil` (failure → absence) | **SURVIVED** | **unevidenced** |
| 41 | `Canonicalize` resolves symlinks | `EvalSymlinks` skipped | killed | full |
| 42 | `Canonicalize` `filepath.Abs` error propagation | error swallowed | survived — **equivalent** (`Abs` only fails when `Getwd` does) |
| 43 | `Inspect` stat error propagation | fabricate `{IsRegular:true, UID:geteuid}` | **SURVIVED** | **unevidenced** |
| 44 | `Inspect` owner-attestation error propagation | `uid = 0` on attestation failure | **SURVIVED** | **unevidenced** |
| 45 | `Inspect` stats target, not link | `os.Stat` → `os.Lstat` | survived — **equivalent** (input is already canonical) |
| 46 | `PathDirs` reads `PATH` | returns nil for a non-empty `PATH` | killed | full |
| 47 | `PathDirs` skips empty entries | empty entries kept | **SURVIVED** | **unevidenced** |
| 48 | `CurrentOperatorPolicy` seeds no administrator | seeds `AdministratorUIDs: {0}` | **SURVIVED** | **unevidenced** |
| 49 | `CurrentOperatorPolicy` uses effective uid | `Geteuid` → `Getuid` | **SURVIVED** | **unevidenced** |
| 50 | `fileOwnerUID` (unix) refuses on unavailable metadata | returns `0, nil` | **SURVIVED** | **unevidenced** |
| 51 | `fileOwnerUID` (windows) refuses | returns `0, nil` | **SURVIVED** | **unevidenced (build-tag bound)** |

### Meta-gates — the derived inventories

| # | Gate | Mutant | Result | Bucket |
| --- | --- | --- | --- | --- |
| 52 | closed refusal-code set | production emits a fourth code `not_supported` | killed | full |
| 53 | `Discover` source-inventory bijection | A1–A5 above | killed (5/5) | full |
| 54 | "no `os/exec` import" scan | scan finds no files | killed — floor present (`the check is blind`) | full |
| 55 | "no capability member" scan | derives an empty member map | killed — floor present (`the scan is blind`) | full |
| 56 | "no SDK-advertising exported symbol" scan | derives an empty symbol list | **SURVIVED** — no floor | **sliver** |
| 57 | refusal inventory: every site exercised | new unexercised production refusal site | killed | partial |
| 58 | refusal inventory: no stray `Error` literal | `Error{...}` minted outside the constructors | killed | partial |
| 59 | refusal inventory: no raw `fmt.Errorf` / `panic` | both minted in `provider.go` | killed | partial |
| 60 | refusal inventory **domain floor** | B1 empty domain / B2 1-of-N domain / H4 skip `provider.go` / B5 wrong-cwd read + planted stray | **SURVIVED (4/4)** | **unevidenced** |

## 4. Measured ratio

**59 of 84 mutants killed. 25 survived; 5 of those are behaviourally
equivalent (rows 36, 37, 38, 42, 45), leaving 20 substantive survivors across
9 distinct defects.**

By gate row (54 rows, excluding the 5 equivalent-mutant rows):

| Bucket | Rows |
| --- | ---: |
| full | 35 |
| partial | 6 |
| sliver | 3 |
| unevidenced | 10 |

The three deletion-form survivors on precondition gates (14b, 15b, 29b) are
counted inside their `partial` rows, not as separate rows.

## 5. Findings

### F11 — the refusal inventory has no floor on its derived domain (blocking)

`auditRefusalInventory` derives its domain with `os.Getwd()` + `os.ReadDir` +
skip-`_test.go`, then asserts only *"every derived site was exercised"*,
*"no stray literals"*, *"no raw constructors"*. All three are vacuous on an
empty domain, and an empty domain is produced by a **successful** read of the
wrong directory — no error, no signal.

Four mutants, four survivals, suite green each time:

- **B1** `deriveRefusalInventory` returns `refusalInventory{}` → green.
- **B2** domain truncated to 1 of N sites → green.
- **H4** the scan skips `provider.go` entirely — every production refusal site
  in the package — → green.
- **B5** a stray `Error{...}` literal planted in `provider.go` **plus** a
  directory read redirected one level up (an empty-but-successful read, exactly
  what a wrong cwd produces) → **green**. The same stray literal alone (B4)
  reddens. The check is real; its domain can be silently emptied out from
  under it.

This is F9's defect, one level up, wearing a parser — the shape LOGBOOK 0230
names as the round-2 root cause. It is not a design disagreement: **the
producer already established this floor three times in this same package** —
`TestDiscoveryReachesNoProcess` has `if !found { t.Fatal("scanned no
production sources; the check is blind") }`, `TestCandidatesAdvertiseNoCapability`
has `if len(members) == 0 { t.Fatal("derived no candidate members; the scan is
blind") }`, and the source inventory has exact-count assertions. The refusal
inventory, which uses the identical file-scan idiom, has none. Fourth
occurrence of "closed the guards the review named, did not enumerate the rest."

Fix: give `deriveRefusalInventory` a floor — a minimum production file count
and a minimum site count, asserted against a derived-not-hand-listed lower
bound — and prove it with the B1/H4 mutants. Row 56 (`exportedSymbols`) needs
the same one-line floor.

### F12 — the production `OSSystem` seam has zero negative tests (blocking)

Rows 40, 43, 44, 47, 48, 49, 50, 51: eight survivors, one file. `os_test.go`
contains no `chmod`, no unreadable path, no stat failure, no attestation
failure — `grep -n "chmod\|0o000\|permission\|Inspect\|fileOwnerUID\|Geteuid"
internal/provider/os_test.go` returns one comment line. Every seam-failure
test in the package drives `fakeSystem`; the fake proves `Discover` reacts
correctly to a reported failure, and proves nothing about whether `OSSystem`
reports one.

Three of these are F6's twin, not new classes:

- **Row 44**: `OSSystem.Inspect` swallowing a `fileOwnerUID` error and
  reporting `uid = 0` survives. F6 fixed uid-0 admission in `Approves`; the
  attestation side that *produces* the uid was never attacked.
- **Row 50**: `fileOwnerUID` (unix) returning `0, nil` when
  `info.Sys()` is not a `*syscall.Stat_t` survives — against its own doc
  comment, *"The host refuses trust when attestation is unavailable rather
  than treating an unknown owner as approved."* Trivially testable: the
  function takes an `os.FileInfo`; a stub whose `Sys()` returns `nil` reaches
  it in three lines.
- **Row 48**: `CurrentOperatorPolicy` seeding `AdministratorUIDs: {0}` — a
  root wildcard at the policy-construction site — survives, against its doc
  comment *"with no additional administrator-approved identities."*

Row 40 (`ReadDir` error → `nil, nil`) is the package's headline invariant
inverted at the only place it can actually be violated: *"a failed or partial
read is reported as a failure, never as an absence"* (`doc.go`), restated in
`README.md:400`. Row 51 (native Windows refusal) is a **stated bound**, not a
fixable gap on this host: `os_windows.go` is not compiled on darwin, so no
mutant here can reach it. The README states the Windows behaviour as fact;
either add a `GOOS=windows` vet/build-tagged check or mark the claim as
unverified on this platform.

### F13 — precondition refusals are pinned by code, not by gate (non-blocking, fix with F12)

Rows 14b, 15b, 29b. `TestDiscoverRefusesPartialReads` and
`TestVerifyTreatsReadFailureAsIntegrityFailure` assert only
`errorCode(err) == codeLocalPrecondition` / `codeIntegrityFailure`. Deleting
the `Canonicalize`-failure or `Inspect`-failure refusal lets control fall
through to the *next* seam, which also fails, with the *same* code — green.
The narrowing forms redden only because `auditRefusalInventory` notices the
site went unexercised, i.e. the meta-gate is carrying the behavioural test.
With F11 unfixed, that carrier can itself be silently emptied.

Assert the refusal `Detail()` (or the wrapped cause) so each subtest pins the
gate that fired, not merely the class of the code.

### F14 — `ax-provider-` is not proven to be an anchor (non-blocking)

Row 11. `strings.CutPrefix` → `strings.Cut` survives:
`TestDiscoverIgnoresNonCandidateEntries` uses `README.md`, `ax-provider`,
`other-tool` — no name that *contains* the prefix without starting with it.
Under the mutant, `vendor-ax-provider-evil` in a `PATH` directory is
discovered as provider `evil`. This contradicts the spec text the package's
own test quotes: *"External provider executables are named
`ax-provider-<id>`"* (`TestSection71TrustRulesArePinned`). One added fixture
name closes it.

### F15 — `Discover` may return a partial candidate set with a refusal (non-blocking)

Row 8. `return nil, err` → `return out, err` survives: every refusal test
discards the first return with `_, err := Discover(...)`. `doc.go` and
`README.md:400` both promise *"instead of yielding a partial set"* — the
"instead of" half is an unsupported claim. Assert `got == nil` in at least one
refusal test per source.

### F16 — forged-receipt coverage is one point wide (non-blocking)

Row 33. `TestVerifyRefusesForgedReceipts` forges the digest only, with every
other fact genuine. Skipping the owner gate when `record.owner == ""` — the
zero value a truncated or half-deserialized persisted receipt carries —
survives. That is *absent evidence treated as satisfied*, in the one function
whose job is refusing forged evidence. Add a zero-value/empty-field receipt to
that test's domain.

### F17 — `Builtins()` copy claim untested (non-blocking)

Row 9. Returning `builtinOrder` directly survives, against the doc comment
*"The result is a copy; the registry cannot be mutated through it."* A caller
mutating the result corrupts discovery order process-wide.

## 6. What this method cannot see

Stated bounds, not hedges:

1. **Build-tag-invisible code.** `os_windows.go` is not compiled on darwin.
   Row 51 is unreachable by any mutant here. `GOOS=windows go vet` would at
   least type-check it; behaviour needs a Windows runner.
2. **No production caller exists.** `grep -rn --include='*.go' 'internal/provider'`
   outside the package returns nothing — there is no `ax` binary wiring
   `Discover`/`Trust`/`Verify` yet. "Production entry point" here means the
   package's exported API, which the tests do drive (`TestOSSystemEndToEnd`
   runs the real `OSSystem` over a real temp tree). Whether the host calls
   them correctly is a later leaf and is *not* evidenced by this task.
3. **Concurrency.** Everything ran single-threaded; no `-race`, no TOCTOU
   attack between `Canonicalize`, `Inspect`, and `ReadFile`. A file swapped
   between the stat and the read is a real substitution vector this review did
   not probe.
4. **Equivalent mutants are my judgement.** Rows 36, 37, 38, 42, 45 are called
   equivalent on reasoning (`Digest.Hex()` is lowercase; a canonical path has
   no links; `Abs` fails only when `Getwd` does), not proven so.
5. **The spec-quote pins were read, not mutated.** They are quote-anchored
   with `Contains` + section checks and `t.Fatalf` on miss, so they are not
   structurally vacuous — but I did not mutate `specdoc` itself.
6. **The empty CR delta.** I reviewed `d8fc669..44a4699` and the package as it
   stands. If any work was intended for round 3 that never reached a commit,
   this review cannot see it.

## 7. Routing

`to-dev`. F11 and F12 are blocking; F13–F17 are cheap and should ride along.
F11 is the fourth recorded instance of the same enumeration failure and should
be landed in `LOGBOOK.md` as such — the reviewer does not modify the tree, so
the producer lands it.
