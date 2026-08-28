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
- The repository bootstrap is the only direct-bootstrap exception. Every later
  change lands on `main` through a pull request; never push feature work
  directly to `main`.
- Before creating a branch or worktree, fetch `origin`, switch to local `main`,
  and fast-forward it to `origin/main`. Never branch from stale `main`.

  ```bash
  git fetch origin main
  git switch main
  git merge --ff-only origin/main
  git switch -c <branch-name>
  ```

- For isolated work, create task-scoped worktrees under
  `.temp/<TASK-ID>/worktree/`; do not place temporary worktrees beside the
  repository.
- Keep changes focused, review the diff, and run relevant tests and validation
  before handing a task to review.

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

Every commit and tag must be signed. Do not add AI attribution or
`Co-Authored-By` trailers.

## Curator-Managed Skills

- `Skillfile.json` is the source of truth for project skills.
- Use Curator to install or update skills. Never edit files under `.agents/`,
  `.claude/skills/`, or `.codex/skills/` by hand; Curator owns those paths.
- Resolve project commands from `.agents/bin/` first. A validated bare command
  may be used when the project shim is unavailable.
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
