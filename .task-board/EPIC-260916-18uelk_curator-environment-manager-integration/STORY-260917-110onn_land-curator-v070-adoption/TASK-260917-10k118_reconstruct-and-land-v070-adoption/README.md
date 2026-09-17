# TASK-260917-10k118: reconstruct-and-land-v070-adoption

## Description
Reconstruct the reviewed v0.7.0 adoption delta on a fresh worktree from current trunk (apply the exported pin patch and registry patch plus the three untracked files, re-derive what must be re-derived against the final trunk registry, re-run every gate) and publish one story_final Change Request.

## Scope
As the Story scope; plus a registry re-derivation verification test and a README measured-coverage pin test.

## Acceptance Criteria
As the Story AC; the candidate is the union of the two accepted revisions plus the two verification tests; task-board.config.json equals HEAD.
