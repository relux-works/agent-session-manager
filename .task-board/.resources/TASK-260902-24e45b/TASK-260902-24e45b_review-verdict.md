# TASK-260902-24e45b review verdict — ACCEPTED

Change Request `CR-TASK-260902-24e45b-1` revision 1. Reviewer run `RUN-260902-4af3ee`.
Base `010d1143e9dc1561f8793d5d18a7dc85558b5da3`, candidate tree `149dbbaf25ec2fe6d065cd3c34ceae975b4dce80`,
commit `5e3e108`, one commit past base, author `Ivan Oparin <oparin@me.com>`, `git verify-commit` good
(ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`).

Every number below was measured in this run. Nothing was accepted from the producer's
`TASK-260902-24e45b_merge-resolution-mutants.log` without reproducing it.

## 1. Provenance of the accepted trees

The attachment hashes to the declared `sha256:c43d27c8...`. Applied to a scratch index at base
`48db30b` it writes tree `3d0f74df25b64bb182a0bec75c4691d7048d6c2a`, byte-equal to `81305c8^{tree}`.
The patch is therefore a faithful snapshot of the two accepted leaves and is a valid oracle for
byte-identity.

## 2. Byte-identity, computed from blob hashes over full trees

- The accepted leaves touch 26 paths (`git diff --name-only 48db30b 81305c8`). The reapply touches
  exactly the same 26 paths. Set difference is empty in both directions: nothing extra was touched,
  nothing the leaves changed was dropped, no file was removed.
- 18 of the 26 are byte-identical to accepted tip `81305c8` — every `internal/localstore/*` file plus
  `path_registry.v0.5.0.json`. Compared by `git rev-parse <rev>:<path>`, not by diff reading.
- 8 conflicted, all of them the predicted cross-Story lines: `README.md`, `LOGBOOK.md`, `go.mod`,
  `go.sum`, `ownership.v0.5.0.json`, `traceability.go`, `traceability_test.go`,
  `cmd/tracecheck/main_test.go`. A `git merge-file` simulation confirms all 8 genuinely conflict —
  none was resolvable mechanically.

## 3. Conflict resolutions, checked structurally rather than read

**`go.mod` / `go.sum`** — `go.sum` is the exact set union: 64 = 62 (accepted) + 14 (trunk) − 12 (base),
sorted, with zero accepted lines missing, zero trunk lines missing, and zero lines belonging to
neither. `go.mod` keeps trunk's `go-toml` alongside the accepted `x/sys` + `modernc.org/sqlite` and
all 8 indirect rows. `go mod tidy` in a scratch copy produces **zero drift** — the merged module
files are exactly what the toolchain would emit.

**`ownership.v0.5.0.json`** — machine-checked three-way over all four trees, keying acceptance cases
by `id` and ownership groups by `(kind, keys, production)`:

- acceptance cases: 43 keys, 0 mismatches, 0 changed-on-both-sides. 43 = 29 base + 5 accepted +
  9 trunk. Every one-sided change landed on the side that changed it.
- ownership groups: 55 in HEAD from 35 base / 38 accepted / 53 trunk. Exactly one entry changed on
  both sides — `section:10.1` — and it is correctly unioned
  (`…canonical-identity-refusal, core-record-identity-validation, localstore-immutable-blob-install`).
- Exactly one structural collision, addressed in §5.

**`README.md`** — the tools row is a real union: trunk's "versioned Configuration readers/current
writer" and the accepted "owner-local storage, immutable installs, and SQLite rebuild/recovery" both
survive; the scoped section list is trunk's 27 plus `3.2`, `3.3`, `18.4` = 30; both
`go test ./internal/config` and `go test ./internal/localstore` rows are present; the outputs column
carries the accepted owner-only-roots clause. Only three lines were removed anywhere in the file, all
replaced in place. The acceptance-case sentence reads 43, matching the registry.

**`main_test.go` / `traceability_test.go`** — every rename-mutant row from both sides is retained
(trunk's `2.1`–`2.4`/`5.1`, the accepted `3.2`/`3.3`/`18.4`), trunk's
`TestRunAssignedConfigPathSectionUsesScopedImplementationOwner` survives, the scalar-section list is
the union, and all three count strings read 43.

**`LOGBOOK.md`** — add/add. All 4 accepted headings and all 15 trunk headings are present; one new
entry. Section bodies are byte-identical across the join except two blank lines at the seam. No
content lost.

## 4. The derived digest was rederived, not read

`reviewedOwnershipCanonicalSHA256` is `b5882b26…`, distinct from both the accepted `b7afad1b…` and
the trunk `f126a4a0…`. I stubbed the constant to 64 zeroes in a scratch copy and read the computed
value out of the `traceability.go:286` refusal:

```
ownership registry projection digest b5882b265b29d0c286c62a0263a6a38444d972434a5c5d42569efe6fc3af0b2c
  differs from reviewed 0000…
```

Identical to the pinned constant. Copying either side in reddens (§6).

## 5. The reported deviation reproduces and is correctly reported

`internal/config/loader.go:ResolvePaths` (trunk `cc89771`) and `internal/localstore/paths.go:ResolvePaths`
(accepted leaf) are two independent production implementations of the same SPEC.md §3.2 rule.
`verifyOwnershipGroups` refuses two bindings on one key (`traceability.go`, `duplicate %s
implementation owner for %q`), so a union of the `production` field is structurally impossible —
only the `acceptance_cases` array can be unioned, and it was. The producer kept the localstore owner
as `production`, merged `AC-PATH-001` into its cases, left both implementations untouched in the tree,
and recorded the ownership question as an open follow-up in board notes and `LOGBOOK.md` §1950 rather
than deciding it. That is reported, not absorbed, and it satisfies the AC.

The load-bearing half of that rationale reproduces: `AC-PATH-001`'s own `production` is
`internal/config/loader.go:Load`, and renaming it refuses under `-section 3.2` (§6, R-E). Trunk's
§3.2 evidence survives.

## 6. Attacks — every gate driven, not read

Global `tracecheck` unless noted. Baseline green before and after each.

| # | Mutant | Expected | Observed |
| --- | --- | --- | ---: |
| R-A | digest stubbed to zeroes | refusal naming computed digest | RED, digest `b5882b26…` |
| R-B | digest copied from the trunk side `f126a4a0…` | RED | RED |
| R-C | digest copied from the accepted side `b7afad1b…` | RED | RED |
| R-D | duplicate `section:3.2` binding reintroduced, **digest resealed** | RED structurally | RED, `duplicate section_binding implementation owner for "section:3.2"` |
| R-E | `internal/config/loader.go:Load` renamed, `-section 3.2` | RED | RED, `acceptance case "AC-PATH-001" production owner` |
| R-F | `internal/localstore/paths.go:ResolvePaths` renamed, `-section 3.2` | RED | RED, `section binding "section:3.2" production owner` |
| R-N1 | closed-schema gate narrowed to admit **undeclared** `sqlite_master` objects | RED | RED, localstore suite |
| R-N2b | closed-schema gate narrowed to admit **drifted declared** objects (compile-clean) | RED | RED, 4 subtests incl. autoindex `tbl_name` drift |

R-D matters: it was run with the digest **resealed to the mutated registry**, so the kill is the
structural duplicate-owner refusal and not the digest pin. R-N1/R-N2b confirm the transplanted
integrity gates are live in the merged tree rather than merely compiling — both halves of the
complement-refusing schema check bite independently.

After the full sweep the scratch tree is byte-identical to HEAD (`diff -r`), so no mutant leaked into
any measurement above.

## 7. Gates, all run in this review, all exit 0

`gofmt -l ./internal` (0 files) · `go build ./...` · `go vet ./...` ·
`go test ./... -cover -count=1` · global `tracecheck` (`acceptance_cases=43 assigned_scopes=0`) ·
30-section scoped `tracecheck` (`assigned_scopes=30`) · `cataloggen -check` (worktree clean after) ·
all five seeded fuzz targets at `-fuzztime=100x` · `GOOS=linux` and `GOOS=windows` `go build` and
`go vet`.

## 8. Coverage — measured on all three trees, not quoted

| Package | accepted `81305c8` | trunk `010d114` | HEAD | Delta vs baseline |
| --- | ---: | ---: | ---: | --- |
| `internal/localstore` | 83.7–83.8% | — | 83.5–83.8% | none (files byte-identical; 83.5–83.8 is run-to-run noise on both trees) |
| `internal/canonicaljson` | 87.1% | 97.2% | 97.2% | none (keeps the higher trunk baseline) |
| `internal/config` | — | 94.7% | 94.7% | none |
| `internal/catalog` | 97.6% | 97.6% | 97.6% | none |
| `internal/catalog/cmd/cataloggen` | 79.3% | 79.3% | 79.3% | none |
| `internal/cataloggen` | 83.9% | 83.9% | 83.9% | none |
| `internal/scalar` | 90.1% | 90.1% | 90.1% | none |
| `internal/specpin` | 85.1% | 85.1% | 85.1% | none |
| `internal/traceability` | 85.0% | 85.0% | 85.0% | none |
| `.../cmd/tracecheck` | 87.5% | 87.5% | 87.5% | none |

No regression against either baseline.

## 9. Finding — two §3.2 evidence claims do not reproduce as framed (non-blocking)

`verifyAcceptanceCases` iterates the **entire** acceptance-case registry unconditionally, before any
section scoping and before `verifyOwnershipGroups`. Two consequences the shipped evidence gets wrong:

1. **`LOGBOOK.md` §1950 states "Scoped `tracecheck -section 3.2` now verifies both packages; before
   the merge each side verified only its own."** The "verifies both packages" property comes from
   registry membership, not from the binding merge. Probe: with `AC-PATH-001` **removed** from the
   merged §3.2 binding and the digest resealed to `18b0c324…`, renaming `internal/config/loader.go:Load`
   still refuses under `-section 3.2` with `acceptance case "AC-PATH-001" production owner:
   declaration "Load" is absent`. Mutant `m09` is therefore not attributable to the binding merge, and
   the sentence reads as a scoping claim that the mechanism does not support.
2. **Mutant `m10` ("AC-PATH-001 dropped from the merged 3.2 binding → RED") kills only through the
   digest pin.** With the digest resealed, the same mutation passes global `tracecheck` at exit 0. Any
   byte change to the registry reddens that pin, so `m10` adds nothing beyond `m01`–`m04`.

This changes no code, gate, or decision. The DECISION's operative argument — that trunk's §3.2
evidence survives through `AC-PATH-001`, whose `production` is `internal/config/loader.go:Load` — is
correct and reproduced above at R-E. The union merge itself is right; only its attribution is
overstated. The correction belongs with the already-open §3.2 double-ownership follow-up: the
`LOGBOOK` clause should say the trunk case survives as a registry acceptance case verified on every
run, and the mutant log should mark `m10` as a digest-pin kill rather than a structural one.

Not gated, and the producer says so plainly: the README tools row and acceptance-case sentence are
prose that nothing reads, and can drift again silently.

## Verdict

**ACCEPTED.** Both leaves land as one signed commit on current trunk; 18/18 untouched-by-conflict
files are byte-identical to the accepted trees and the eight conflicts are union merges verified
structurally; the derived digest was independently rederived from the refusal and matches; every gate
exits 0 in this review; coverage regresses nowhere against three measured baselines; and the one
resolution that could not be unioned is reported, not absorbed. The §9 finding is a documentation
accuracy defect on an otherwise fully reproduced transplant and is recorded for the open §3.2
follow-up.
