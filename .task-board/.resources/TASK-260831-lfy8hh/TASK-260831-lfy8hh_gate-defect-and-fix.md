# Catalog freshness gate: false-refusal defect and fix

## The defect

Configured command (before):

```
go generate ./internal/catalog && git diff --exit-code -- internal/catalog/catalog_gen.go
```

`git diff --exit-code` compares the working tree against the **git index**. In CI
that is equivalent to "is the committed generated file current", because CI
checks out a clean tree. A managed Story worktree is different by construction:
producer changes are legitimately unstaged until the Change Request snapshots
the tree. The gate therefore refuses any tree in which a correct, regenerated
`catalog_gen.go` is simply not staged.

Measured on the `STORY-260830-2wdm5e` worktree:

```
regeneration changes the file?   no  (generated output already matches sources)
configured gate                  FAIL
what it compared against         the index: 44 unstaged insertions
```

It blocked two consecutive producer runs — `RUN-260831-e2c291` and
`RUN-260831-c66658` — whose generated output was provably current, and a third,
`RUN-260831-f1f4ca`, was halted by the Orchestrator once the cause was
identified. Earlier tasks passed only because their producers happened to stage
the regenerated file.

## The fix

```
tmp=$(mktemp); cp internal/catalog/catalog_gen.go "$tmp"; go generate ./internal/catalog && diff -q "$tmp" internal/catalog/catalog_gen.go; rc=$?; rm -f "$tmp"; exit $rc
```

This tests the gate's actual intent — regeneration is a no-op, so the generated
file matches its sources — independently of git staging state, and it leaves the
tree as it found it.

## Verification

| Case | Old gate | New gate |
| --- | --- | --- |
| Correct tree, generated file unstaged | **FAIL** (false refusal) | PASS |
| Generated file genuinely drifted from sources | FAIL | FAIL |
| Tree restored after the drift probe | — | PASS |

The drift case was produced by appending a line to `catalog_gen.go`, so
regeneration provably rewrites it. Full 13-command suite: 13/13 exit 0.

## Note

This gate was authored by the Orchestrator in `TASK-260831-3e99as`. It is the
same false-refusal shape that reviews on this board have been rejecting in
production code: a check that admits or refuses on a proxy signal rather than on
the property it claims to verify.
