# BUG-260902-3dn8jd — revision 5 recovery run: repository work re-verified, publication blocked on stale workspace checkpoint

Run: RUN-260902-0fd594 (autonomous recovery, attempt 1/3, root RUN-260902-589898)
Worktree: `.temp/STORY-260902-2xwq38/worktree`, branch `task-board/story/STORY-260902-2xwq38`

## Summary

The repository change is finished, was ACCEPTED at review round 3, was rebased
onto current trunk in revision 4, and is re-verified independently here. It
cannot be published: Change Request construction refuses with
`change_request_base_authority_mismatch` because the managed workspace's
`checkpoint_oid` is still the pre-refresh fork point. No producer-reachable
command advances it. This run therefore delivers verification and this evidence
packet, and stops at `blocked` rather than burning recovery attempts 2/3 and 3/3
on a handoff that would fail identically.

## The blocker

The refusal that ended RUN-260902-589898:

```
change_request_base_authority_mismatch: the STORY-260902-2xwq38 candidate provenance
disagrees: checkpoint 422786cc5b4303f03d3971caa509ac12b49a00c6 does not descend from
selected authority facbd9a8ca3bcf385a0b5f9c646c81282a495758
while branch=6466943e53de001817901079cf9cd15872afd716
and head=6466943e53de001817901079cf9cd15872afd716
```

`task-board worktree status --json`, workspace `WS-10a6fe438910`:

| Field | Value | State |
| --- | --- | --- |
| `initial_base_oid` | `422786cc` | fork point |
| `checkpoint_oid` | `422786cc` | **STALE** — never advanced |
| `current_base_oid` | `facbd9a8` | refreshed |
| `selected_base_oid` | `facbd9a8` | refreshed |
| `upstream_oid` / `protected_authority.fetched_oid` | `facbd9a8` | fresh authority |
| `branch_tip_oid` | `6466943e` | correct leaf |

`checkpoint_oid == initial_base_oid` means no leaf has ever been checkpointed on
this Story, so the checkpoint IS the fork point. Revision 4 rebased the leaf onto
`facbd9a` under an explicit orchestrator brief. The workspace record's
`selected_base_oid` followed the authority; `checkpoint_oid` did not. The guard
compares `checkpoint` against `selected authority`, and `422786cc` is an ancestor
of `facbd9a8`, not a descendant.

The branch itself is in exactly the shape publication requires. The board record
is what disagrees.

## Why this is not producer rework

Attempts considered and rejected, with the reason:

1. **Advance the checkpoint from the CLI.** No such command exists.
   `task-board worktree checkpoint` and `worktree integrate` both state
   "orchestrator's command" / "orchestrator-only"; `worktree repair` only
   re-materializes a workspace whose worktree is missing; `worktree abort`
   destroys the work; `worktree transaction show STORY-260902-2xwq38` reports
   `No integration transaction is recorded`, so there is nothing to discharge or
   resolve. The mutation DSL exposes no checkpoint or workspace mutation
   (`task-board q 'schema()'`, 36 mutations, none workspace-scoped).

2. **Reset the branch back to the pre-rebase leaf `b8a85d4` (parent `422786c`)
   so the automatic base refresh can replay it.** Rejected. The refresh replays
   through a three-way tree merge ("the three-way tree merge against the advanced
   trunk conflicted"), and this leaf's overlap with trunk `facbd9a` is exactly
   `LOGBOOK.md` and `README.md` — the two files that already required manual
   composition in revision 4, including a real merge defect (four dropped blank
   separators before interleaved `###` headings) that only a human-read
   composition caught. Replaying would either abort on that conflict or discard a
   resolution that was produced under an explicit brief and validated. Trading a
   verified composition for a probable abort is not a fix.

3. **Hand off to `to-review` anyway.** Rejected. That re-runs CR construction,
   which fails on the same guard, and consumes recovery attempts 2/3 and 3/3.
   The refusal is not a function of anything a producer can change.

## Exact orchestrator action needed

Advance workspace `WS-10a6fe438910`'s `checkpoint_oid` from
`422786cc5b4303f03d3971caa509ac12b49a00c6` to
`facbd9a8ca3bcf385a0b5f9c646c81282a495758`, then republish.

This is a bookkeeping advance, not a replay: the branch is already rooted at
`facbd9a8`, its tree is clean, and `git rev-list --count origin/main..HEAD` is
exactly 1. Once the checkpoint equals the authority, the leaf reads as exactly
one commit past the checkpoint and CR construction has nothing left to object to.

## Leaf shape — verified in this run

```
commit    6466943e53de001817901079cf9cd15872afd716
parent    facbd9a8ca3bcf385a0b5f9c646c81282a495758   (== origin/main)
signature G  Good "git" signature for oparin@me.com, ECDSA SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM
author    Ivan Oparin <oparin@me.com>
ahead 1, behind 0, single parent, git status --short empty
```

Ten changed paths against trunk, unchanged from the accepted revision-3 set plus
the two composed documentation files:

```
.spec/README.md
LOGBOOK.md
README.md
internal/canonicaljson/constraint_excerpt_test.go
internal/canonicaljson/constraint_inventory_test.go
internal/canonicaljson/session_record_versions_test.go
internal/canonicaljson/testdata/constraint-enumeration.md
internal/specdoc/SPEC.md
internal/specdoc/specdoc.go
internal/specdoc/specdoc_test.go
```

`internal/specpin` is absent from the diff, so the digest the fidelity gate
anchors on is the pre-existing pin and could not have been minted to fit. The
vendored document hashes to the pin:

```
sha256(internal/specdoc/SPEC.md) = 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
internal/specpin/pin.go:30 DocumentSHA256 = 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
```

## Validation — all 17 configured gates, real exit codes, this tree

Every command from `task-board.config.json -> spawn.worktree_isolation.validation.commands`,
run as a standalone process, exit code reported as returned. Log:
`BUG-260902-3dn8jd_rev5-validation.log`.

| # | Command | Exit |
| --- | --- | ---: |
| 1 | `test -z "$(gofmt -l $(git ls-files ... '*.go'))"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0 |
| 5 | `go test ./... -race -count=1` | 0 |
| 6 | `go test ./... -cover -count=1` | 0 |
| 7 | `go test ./internal/scalar -fuzz=FuzzScalarProductionEntries -fuzztime=100x` | 0 |
| 8 | `go test ./internal/canonicaljson -fuzz=FuzzCanonicalizeRoundTrip -fuzztime=100x` | 0 |
| 9 | `go test ./internal/canonicaljson -fuzz=FuzzObjectIdentityRepresentationInvariant -fuzztime=100x` | 0 |
| 10 | `go test ./internal/canonicaljson -fuzz=FuzzClosedIdentityShapeRefusal -fuzztime=100x` | 0 |
| 11 | `go test ./internal/canonicaljson -fuzz=FuzzObservationEventRefusal -fuzztime=100x` | 0 |
| 12 | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| 13 | `cataloggen ... -check` | 0 |
| 14 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 15 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 16 | `git ls-files -z '*.json' \| xargs -0 -n1 python3 -c 'json.load(...)'` | 0 |
| 17 | `task-board validate` | 0 |
| 18 | `git diff --check` | 0 |

## Anti-vacuity — re-proved on THIS merged tree, not inherited

Three mutants. Each was confirmed present in the file before its result was
believed. Each was reverted from a byte-backup under `.temp/BUG-260902-3dn8jd/backup/`,
never with `git checkout`, and the restored SHA-256 was compared against the
pre-mutation SHA-256 every time. `git status --short` was empty after each.

**Mutant A — the original incident defect, replayed.** Replaced the
`EnvironmentTuple.store_schema_fingerprint` excerpt with
`L3630 "<code>store_schema_fingerprint</code> is a lowercase hex digest"` — the exact
class of invented `digest` claim this Bug exists to catch. Presence confirmed
(`grep -c` = 1). `go test ./internal/canonicaljson/ -count=1` **real exit 1**,
three tests red, naming both the row and the absent text:

```
TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification
  EnvironmentTuple.store_schema_fingerprint: entry 1 (L3630) quotes text absent from
  the pinned SPEC.md: "<code>store_schema_fingerprint</code> is a lowercase hex digest"
TestArtifactQuotesAreVerbatimPinnedSpecificationText
  line 375 quotes text absent from the pinned SPEC.md: ...
TestUnmodifiedConstraintEnumerationIsAdmitted   (shipped row refused)
```

**Mutant B — a NARROWING of the document, not a deletion.** Appended one trailing
space to `SPEC.md` line 3630. That perturbation is benign under the package's own
whitespace-collapse rule, so the excerpt comparison alone would still admit the
row; only the digest gate can catch it. **Real exit 1**, refused by name and by
both digests:

```
Load() error = pinned specification document mismatch:
SHA-256 is b7ffc9ea04b0b38e38a7092bf052ad2ece47f4f7f3f8c1113f34694d0b362c2a,
want    562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
```

Every excerpt test in `canonicaljson` failed closed on the load error rather than
comparing against the substituted document. This distinguishes the two axes: a
document that would still satisfy the text comparison is still refused, so the
gate is anchored on the pin and not on the text alone.

**Mutant C — a NARROWING of a citation, not a deletion.** Kept the true quote and
moved its declared line from `L3630` to `L3631`. **Real exit 1**, refused with the
correct location rather than a generic failure:

```
entry 1 (L3631) quotes text that begins at [3630], not the declared line:
"<code>store_schema_fingerprint</code>, and <code>adapter_version</code>"
```

**Bound, not a parser that reddens on everything.** The unmodified tree passes:
gate #4 above (`go test ./... -count=1 -v`, exit 0) includes
`TestUnmodifiedConstraintEnumerationIsAdmitted`, which is red under all three
mutants and green on the shipped tree.

## Embed isolation — verified independently, not read from a comment

The 883 KiB vendored `SPEC.md` must not reach a shipped binary. A bare
`go list -deps ./... | grep internal/specdoc` returns a match because `./...`
enumerates the package itself; that is not the check. The real check:

- No package's non-test `Imports` contains `internal/specdoc` (0 matches).
- It appears only in `TestImports`/`XTestImports` (2 matches).
- Both `main` packages were walked with `go list -deps` individually:
  `internal/catalog/cmd/cataloggen` — clean; `internal/traceability/cmd/tracecheck` — clean.

## Carried forward, unchanged

Open items disclosed by earlier revisions and still open, each needing its own
board item — none of them regressed or newly introduced here:

- `validateEnvironmentTuple` enforces `[a-z][a-z0-9.-]{0,63}` on `environment_id`
  (`closed_shapes.go:733`) while the pinned `EnvironmentTuple` clause
  (`SPEC.md:3626-3630`) gives that member no type and no format. Production left
  unchanged deliberately; both lines are quoted in the row and the open question
  is named. Both lines sit in clause 7.8, so the clause anchor genuinely cannot
  catch it — the stated limit is honest.
- `testdata/closed-vocabularies.md` has the same class of unchecked declaration
  column, and its `validateSessionForkProvenance` `provider_fork_mode` row cites
  the same-provider-fork variant rather than the Fork Provenance prose its
  validator implements.
- Residual gap, stated in `README.md` and in the artifact's own
  "What this check does not prove" section: the anchor is a CLAUSE, not a SHAPE.
  Ten shapes are declared in 10.4, so a retarget within one clause is still
  admitted, and a citation landing inside a fenced JSON example is admitted
  (zero shipped rows use either).
- Pre-existing `LOGBOOK.md` ordering inversion (2026-09-02 1810 before 2026-09-02
  2240), identical in base, trunk, Story and merged. Owned by trunk; reported,
  not silently repaired.

Nothing was weakened or deleted to make anything pass in this run. No production
validator was changed by this Bug at any revision.
