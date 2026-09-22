# BUG-260917-2fwf8e: fencing-authorize-checks-expiry-before-ownership-direction

## Description
Found by RUN-260917-3934e0 (claude-opus-5 max) while reviewing TASK-260830-1geqhj CR1, probe 10 in TASK-260830-1geqhj_review-evidence-rev1.tar.gz. internal/fencing Authorize evaluates grant expiry BEFORE ownership direction, so a remote interactive owner with a lapsed local grant is refused lease_conflict instead of being offered attach/takeover. The after-restore sequence step 4 (offer remote attach/takeover when another host owns the session interactively) is therefore unreachable once the local grant lapses.

## Scope
internal/fencing authorization arm ordering; no change to the lease chain or to the park vocabulary.

## Acceptance Criteria
A remote interactive owner with a lapsed local grant yields the attach/takeover offer path, not lease_conflict; the ordering is pinned by a test that fails when the two arms are swapped; the previously reported vector and one vector a step away both behave correctly.
