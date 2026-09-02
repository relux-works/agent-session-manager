# BUG-260902-3dn8jd — revision 6 publication state: handed off, but no new Change Request was constructed

Run `RUN-260902-0d963e`. Read this before reviewing: the reviewer will otherwise
open a patch that is not what is on the branch.

## What the handoff actually did

`task-board handoff BUG-260902-3dn8jd --role developer` returned **exit 0** and
moved the board to `to-review` with `checklist:20/20`. It did **not** construct a
Change Request revision. Measured immediately afterwards:

| Fact | Value |
| --- | --- |
| latest Change Request | `CR-BUG-260902-3dn8jd-3`, revision 3 |
| its state | **`stale`** |
| its `base_oid` | `422786cc` |
| its `candidate_tree_oid` | `faa92b69` |
| workspace `checkpoint_oid` | `422786cc` — still the fork point |
| workspace `selected_base_oid` | `facbd9a8` |
| branch tip | `6466943e`, tree `49fa3c6a`, parent `facbd9a8` (== `origin/main`) |

No revision 4, 5 or 6 patch resource exists. This differs from revision 4's run,
which was hard-refused with `change_request_base_authority_mismatch`; here the
handoff succeeded and the construction simply produced nothing.

## What a reviewer would be looking at, and why it is wrong

`BUG-260902-3dn8jd_change-request_rev3.patch` renders tree `faa92b69`, which is
the tree of the **pre-rebase** leaf `b8a85d4`, rooted at `422786cc`. The branch
carries tree `49fa3c6a`, rooted at `facbd9a8`. The stale patch is therefore not
merely stale by base pointer; its bytes are not the branch's bytes.

Blob-level comparison of this Bug's ten paths, reviewed tree `faa92b69` against
branch tree `49fa3c6a`:

```
IDENTICAL  .spec/README.md
IDENTICAL  internal/canonicaljson/constraint_excerpt_test.go
IDENTICAL  internal/canonicaljson/constraint_inventory_test.go
IDENTICAL  internal/canonicaljson/session_record_versions_test.go
IDENTICAL  internal/canonicaljson/testdata/constraint-enumeration.md
IDENTICAL  internal/specdoc/SPEC.md
IDENTICAL  internal/specdoc/specdoc.go
IDENTICAL  internal/specdoc/specdoc_test.go
DIFFERS    LOGBOOK.md   648604db -> a45ea4a4
DIFFERS    README.md    b5b65b01 -> a390c107
```

So the accepted substance — the gate, the pinned document, the rewritten
artifact, the two dependent test files — is **byte-identical** to what review
round 3 accepted. The only divergence is the two documentation files that
revision 4 composed by hand against advanced trunk, which is the composition
whose merge defect (four dropped blank separators before interleaved `###`
headings) that run found and fixed.

Everything else in the whole-tree diff between the two trees is trunk's own
advance `422786cc..facbd9a8` absorbed by the rebase.

## The blocker is unchanged and is orchestrator-only

`checkpoint_oid` (`422786cc`) is an ancestor of, not a descendant of, the
selected authority (`facbd9a8`). Revision 4's manual rebase moved the branch and
the workspace's `selected_base_oid`; the checkpoint did not follow. Advancing it
requires `task-board worktree checkpoint`, which is orchestrator-only and itself
needs an accepted Change Request — the cycle.

## Revision 5's rejected option 2, now measured rather than reasoned

Revision 5 rejected "reset the branch to the pre-rebase leaf `b8a85d4` so the
automatic base refresh can replay it" on the grounds that the replay would
probably conflict. This run measured it instead of estimating it:

```
$ git merge-base facbd9a8 b8a85d4
422786cc5b4303f03d3971caa509ac12b49a00c6
$ git merge-tree --write-tree facbd9a8 b8a85d4        # exit 1
9aa9085e03cb049b95f70fa580ce12ed18ea3439
100644 d4365b38... 1  LOGBOOK.md
100644 1d72f199... 2  LOGBOOK.md
100644 648604db... 3  LOGBOOK.md

Auto-merging LOGBOOK.md
CONFLICT (content): Merge conflict in LOGBOOK.md
```

The refresh's three-way tree merge conflicts on `LOGBOOK.md`. Option 2 is a
measured dead end, not a risk estimate, and no other producer-reachable route
advances the checkpoint.

## Exact orchestrator action needed

Advance workspace `WS-10a6fe438910`'s `checkpoint_oid` from `422786cc` to
`facbd9a8`, then republish. The branch is already rooted at `facbd9a8`,
`git rev-list --count origin/main..HEAD` is exactly 1, the worktree is clean, and
the head is signed (`G`, `oparin@me.com`). Once the checkpoint equals the
authority the leaf reads as exactly one commit past it and construction has
nothing left to object to.

Until then, review the **branch tip `6466943e`** directly rather than
`BUG-260902-3dn8jd_change-request_rev3.patch`.

## Verification supporting this handoff

`BUG-260902-3dn8jd_rev6-verification.md` — 18/18 configured gates at real exit 0
on this exact tree, three anti-vacuity mutants each red on the real entry point
and each restored from a byte backup with SHA-256 compared, pin equality and
embed isolation verified per package. `BUG-260902-3dn8jd_rev6-validation.log` and
`BUG-260902-3dn8jd_rev6-mutants.log` carry the raw output.
