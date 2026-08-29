# Agent Session Manager Repository Instructions

`AGENTS.md` is the canonical instruction file for this repository. `CLAUDE.md`
must remain a relative symlink to this file so every supported agent reads the
same rules.

## Product and Specification

- The product is Agent Session Manager, exposed as the `ax` command.
- Treat the [normative AX specification](https://github.com/relux-works/agent-session-manager-spec/blob/main/SPEC.md)
  as the authority for product behavior. Do not invent behavior that conflicts
  with it.
- This repository contains the implementation. Keep specifications and
  implementation changes traceable to their task-board work.

## Change Workflow

- Track every change in `task-board` before implementation begins. Keep the
  task status, checklist, notes, and task-scoped outcome evidence current.
- An authorized repository-changing task includes automatic delivery: stage
  only its reviewed scope, create author-signed commits, publish the feature
  branch, open or update the pull request, complete review and checks, and land
  the accepted exact head in `main`. Do not stop merely to ask whether the
  completed task should be committed, pushed, or landed.
- The repository bootstrap is the only direct-bootstrap exception. Every later
  change must be reviewed through a pull request. After approval and passing
  checks, land the exact signed feature head with a plain fast-forward push to
  `main`; never land through GitHub rebase, squash, or merge-commit actions.
- Never push unreviewed work, a different commit from the reviewed PR head, or
  a forced update to `main`. GitHub records the reviewed PR as indirectly
  merged when its exact head becomes reachable from `main`.
- Before creating a branch or worktree, fetch `origin`, switch to local `main`,
  and fast-forward it to `origin/main`. Never branch from stale `main`.

  ```bash
  git fetch origin main
  git switch main
  git merge --ff-only origin/main
  test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"
  git switch -c <branch-name>
  ```

- Use this signed linear landing flow after the PR is approved and its checks
  pass:

  ```bash
  BRANCH=$(git branch --show-current)
  test "$BRANCH" != main
  git fetch origin

  # If main advanced, rewrite locally and sign every rewritten commit.
  BEFORE_REBASE=$(git rev-parse HEAD)
  git rebase -S origin/main

  # Update only the feature branch when the signed head changed.
  if [ "$(git rev-parse HEAD)" != "$BEFORE_REBASE" ]; then
    git push --force-with-lease origin HEAD:refs/heads/"$BRANCH"
  fi
  ```

  If the feature head changed, stop here until that exact head passes the
  required PR review and checks. Then land it without rewriting:

  ```bash
  BRANCH=$(git branch --show-current)
  test "$BRANCH" != main
  git fetch origin

  # Verify every commit that will be introduced and require a fast-forward.
  for sha in $(git rev-list --reverse origin/main..HEAD); do
    git verify-commit "$sha" || exit 1
  done
  git merge-base --is-ancestor origin/main HEAD

  SIGNED_HEAD=$(git rev-parse HEAD)
  test "$(git rev-parse "origin/$BRANCH")" = "$SIGNED_HEAD"
  test "$(gh pr view --json headRefOid --jq .headRefOid)" = "$SIGNED_HEAD"

  # Plain push fails safely if main advanced. Never force main.
  git push origin HEAD:refs/heads/main
  git fetch origin
  test "$(git rev-parse origin/main)" = "$SIGNED_HEAD"
  git verify-commit "$SIGNED_HEAD"
  ```

  If the plain push to `main` is rejected, refresh and repeat the signed
  rebase/review cycle; do not fall back to a GitHub merge action or force the
  protected branch.

- For isolated work, create task-scoped worktrees under
  `.temp/<TASK-ID>/worktree/`; do not place temporary worktrees beside the
  repository.
- Keep changes focused, review the diff, and run relevant tests and validation
  before opening or updating its pull request.

## Git Identity and Signing

Use the repository-local identity and SSH signing configuration below:

```bash
git config --local user.name "Ivan Oparin"
git config --local user.email "oparin@me.com"
git config --local gpg.format ssh
git config --local user.signingkey /Users/iv/.ssh/ivanopcode.pub
git config --local commit.gpgsign true
git config --local tag.gpgsign true
```

Every commit and tag must be signed. Verify commits with `git verify-commit`
before publishing them. Rebase locally with `git rebase -S` whenever history
must move; GitHub rebase and squash create different unsigned commit objects
and are not valid landing mechanisms. Agent co-authorship is allowed through an
explicit `Co-authored-by` trailer when the task or user requests it and a real,
intentionally configured agent name and email are available. Keep Ivan Oparin
as the commit author and never invent an identity or verification address.

## Curator-Managed Skills

- `Skillfile.json` is the source of truth for project-local skills and target
  agent adapters.
- The `project-management` skill and `task-board` CLI are global development
  dependencies. Do not declare, install, vendor, or shadow
  `project-management` in this repository; verify and use the global skill and
  global `task-board` command.
- Use Curator to install or update skills. Never edit files under `.agents/`,
  `.claude/skills/`, or `.codex/skills/` by hand; Curator owns those paths.
- Resolve Curator-managed project commands from `.agents/bin/` first. The
  globally managed `task-board` command is the intentional exception. Other
  bare commands require explicit validation before use.
- Run `curator install` after changing `Skillfile.json`, then require
  `curator status --check` to exit successfully.

## Go Development and Testing

- Use the Curator-managed `go-testing-tools` skill for every Go implementation,
  refactor, or test task. Read its `SKILL.md` before technical work and follow
  its closed-loop workflow.
- Add or update tests whenever behavior changes. Prefer pure reducer tests,
  component tests, and mocked integration tests in that order when the AX code
  uses Elm-style TUI architecture.
- Drive real production entry points and include negative tests for validation,
  authorization, refusal, and attestation gates.
- Run the relevant package tests while iterating, then run the repository-wide
  test and coverage commands before review:

  ```bash
  go test ./... -v
  go test ./... -cover
  ```

- Keep tests deterministic and free of network or terminal dependencies. Do
  not use stubs or mock-only behavior to conceal an invalid platform or product
  assumption.

## Documentation and Artifacts

- Write public specifications, repository documentation, and code comments in
  English.
- Store temporary logs and validation artifacts under
  `.temp/<TASK-ID>/`. Attach task deliverables to the owning board item with a
  task-scoped resource name.
- Keep `README.md` and its tool/output documentation aligned with the current
  repository state.
