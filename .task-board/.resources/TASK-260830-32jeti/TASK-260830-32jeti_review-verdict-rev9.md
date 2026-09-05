# Round-11 reviewer verdict — CR-TASK-260830-32jeti rev 9

**Verdict: ACCEPTED** → `accept_cr` → `integrating`

Reviewed at candidate tree `b63faae75f0e85b54f2544645e2ba05841fa79f0`, base
`2512f2087bcea43481f8541ee780f11daeececd4`.

Provenance established before anything was measured, not assumed:

| claim | how it was established | result |
| --- | --- | ---: |
| live worktree == the snapshotted candidate | content rebuilt into a scratch index, `git write-tree` | `b63faae…` exact |
| rev-9 patch resource is the declared payload | `shasum -a 256` vs the CR's declared digest | `e4c70c56…` match |
| rev-9 patch reconstructs the candidate | applied to a scratch index seeded at `2512f20` | `b63faae…` exact |
| rev-8 tree, for the delta | board's rev-8 patch applied to `2512f20` | `39853bf…`, matches round-10 |
| rev-7 tree, for the carry | board's rev-7 patch (`c675711de70a…`) applied to `57afcc6` | `3817cef…`, matches round-9 |

Every tree named below was reconstructed from a board resource in this run.

---

## F1 from round 10 is closed

`GOOS=windows go vet ./...` was exit 1 on rev 8 against
`terminalbackend_test.go:972:20: undefined: syscall.Mkfifo`. On rev 9:

| gate | rev 8 | rev 9 |
| --- | ---: | ---: |
| `go build ./...` | 0 | **0** |
| `go vet ./...` | 0 | **0** |
| `GOOS=linux go vet ./...` | 0 | **0** |
| **`GOOS=windows go vet ./...`** | **1** | **0** |
| `gofmt -l internal/` | clean | **clean** |
| `go test ./... -count=1` | 16 ok | **16 ok, 0 FAIL, 0 "no test files"** |
| `tracecheck -root .` | 0 | **0**, figures byte-identical |

The fix follows `internal/localstore/projection_unix_test.go`: the FIFO case
moves into a new `internal/terminalbackend/terminalbackend_unix_test.go` under
`//go:build unix`, the shared file drops its `syscall` import and keeps a
pointer comment.

## G-A — the fix did not buy green by disabling coverage

The failure mode for this class is a test that stops executing. Three separate
checks, none of them an exit code:

**1. It runs, and it drives the production entry point.** `go test -list` on
this host resolves 99 of 99 tests in the package including
`TestDigestFileRefusesFIFO`; `-run TestDigestFile -v` shows it `=== RUN` and
`--- PASS`. The mutant stack below names
`terminalbackend_unix_test.go:28 → terminalbackend.DigestFile` at
`terminalbackend.go:628`, so the call site is production, not a helper.

**2. It still asserts what it asserted — NARROWING mutant, not a deletion.**

| id | mutant | `TestDigestFile` (dir case) | `TestDigestFileRefusesFIFO` |
| --- | --- | ---: | ---: |
| M-A1 | `terminalbackend.go:625` `if !info.Mode().IsRegular()` → `if info.Mode().IsDir()` — guard kept, refusal string kept, narrowed to admit exactly FIFOs | **PASS** | **KILLED** |

The guard is present and still refuses directories after the mutation, which is
the point: the directory case cannot distinguish a narrowed guard from an intact
one (`os.ReadFile` rejects directories with EISDIR on its own), and it stays
green. Only the moved FIFO test notices. It notices by blocking in
`os.ReadFile` on the FIFO — precisely the README-documented hazard the guard
exists to prevent — and the package FAILs at `-timeout`. The mutant preserves
the searched-for token `not a regular file`, so a source-text check would not
have seen it either.

Restored and verified: `terminalbackend.go` back to
`8ba5e7e5b0e38766…`, live tree back to `b63faae…`.

**3. Nothing else left the unix build.**

| check | rev 8 | rev 9 |
| --- | ---: | ---: |
| test/fuzz/bench funcs declared in `internal/terminalbackend` | 98 | **99** (+1, −0) |
| file-set delta in that package | — | **+1 file, 0 removed** |
| `//go:build` lines added anywhere in the tree | — | **exactly one**, the new file |
| `XTestGoFiles` darwin / linux / windows | — | 4 / 4 / **3** |

The only thing excluded on Windows is the FIFO test itself, which is correct —
Windows has no `mkfifo`. Declared as the stated bound, not silently taken.

**4. The gate that caught F1 is still live on this tree, and it does cover test
files.** Not inferred from rev 8's failure:

| id | probe | `GOOS=windows go vet ./...` |
| --- | --- | ---: |
| M-A2 | scratch `_test.go` in `internal/terminalbackend` with an unconstrained `syscall.Mkfifo`, no build tag | **exit 1** |
| — | same tree, probe removed | **exit 0** |

Live tree re-verified `b63faae…` after removal.

**5. Delta accounting: 60 → 62 fully closed.**

`git diff --name-status 39853bf b63faae` is exactly three paths — `M LOGBOOK.md`,
`M internal/terminalbackend/terminalbackend_test.go`,
`A internal/terminalbackend/terminalbackend_unix_test.go` — 5 insertions in the
LOGBOOK, 16/-13 in the shared test file, 31 in the new one. Set difference of
the two changed-path lists: `terminalbackend_test.go` is new to the list (it was
untouched from trunk at rev 8) plus the new `_unix_test.go`; **nothing dropped
out**. Nothing outside the carried verdict's coverage moved.

## G-B — what carries, re-verified rather than assumed

**`internal/provider`, `internal/provhost`, `.github`: byte-identical to
accepted rev 7.** Re-checked after the second restoration, not carried forward
from round 10:

```
$ git diff --stat 3817cef… b63faae… -- internal/provider internal/provhost .github
(empty)
```

The round-9 battery — 177 mutant rows, zero resurrections, two declared bounds —
transfers intact. Not re-run, per the brief. The 9 AC production call sites were
re-resolved against the candidate rather than quoted: `DecodeManifest`,
`DecodeProbe`, `RequireCapability`, `ProfileMapping`, `DecodeQuiesceProof`,
`CheckIdentity`, `DecodeSpawnPlan`, `IdempotencyKeyFor`, `DecodeStatusOutcome`
all present at their round-9 line positions. **9 of 9 AC rows driven**, 173
tests across the two packages.

**`README.md`, `internal/traceability/*`, `ownership.v0.5.0.json`: unchanged
rev 8 → rev 9** (absent from the three-path delta). The round-10 measurements
carry unchanged, and `tracecheck` reproduces them byte-for-byte on this tree:

```
contracts=60 normative_sections=36 acceptance_cases=81 fixtures=30 compatibility_contracts=55
bindings=53 full=1 partial=3 sliver=1 unevidenced=45 unmeasured=3 unowned=2 clauses_discharged=17/428
```

**The new LOGBOOK entry did not disturb the sibling's blocks.** rev 8's
LOGBOOK is a **byte-exact prefix** of rev 9's (894 of 899 lines, `cmp` clean) —
a pure append, so no prior block can have moved. Three-way block comparison
against trunk `2512f20` and rev 7 confirms it independently:

| check | result |
| --- | ---: |
| `###` entry blocks: 66 → 67 | +1 |
| trunk blocks missing from candidate | 0 |
| rev-7 blocks missing from candidate | 0 |
| candidate blocks matching no source | 1 — the new rev-9 entry, expected |
| candidate order restricted to trunk == trunk order | true |
| candidate order restricted to rev 7 == rev 7 order | true |
| duplicate `## DATE` headers | none |

Four blocks reported a differing body; all four are boundary artifacts of the
splitter, inspected rather than waved through — two swallowed a following
`## DATE` header, two are a trailing blank line at a merge seam. Zero
substantive body changes.

One observation, not a finding: the new entry appends at the file tail, which
places it under `## 2026-09-01` in a reverse-chronological file. That is the
repository's existing convention for task-titled entries, not something this
Story introduced — trunk itself carries four `TASK-260830-1snnef` entries and
this Story's own round-8 entry the same way. Not this leaf's to change.

## G-C

Clean. The one blocking finding from round 10 is closed by the minimal correct
fix, the fix is proven load-bearing by a narrowing mutant rather than by its exit
code, the gate that found it still works on this tree, and the delta is exactly
the three files the carried verdict does not cover. Everything else is
byte-identical to trees already accepted.

Accepted.

---

### Gates run in this round, on the candidate tree

| gate | result |
| --- | ---: |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=linux go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | 16 packages ok, 0 FAIL |
| `go test ./internal/terminalbackend -run TestDigestFile -v` | 3/3 PASS incl. `TestDigestFileRefusesFIFO` |
| `tracecheck -root .` | exit 0 |
| M-A1 narrowing mutant | KILLED, restored, tree re-verified |
| M-A2 gate-liveness probe | exit 1 / exit 0, restored, tree re-verified |
