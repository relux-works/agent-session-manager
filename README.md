# Agent Session Manager (`ax`)

Agent Session Manager is the implementation repository for the local-first,
cross-host `ax` coding-agent session orchestrator.

## Project Status

The Curator-managed repository foundation is complete and the published AX
v0.5.0 specification has been decomposed into an implementation board. No Go
production implementation has been added yet.

Product behavior is defined by the
[normative AX specification](https://github.com/relux-works/agent-session-manager-spec/blob/main/SPEC.md).
When this repository and the specification disagree, the specification is
authoritative.

## Bootstrap

The machine used for repository bootstrap runs Curator
`v0.14.0-rc.3-106-g665dd13f`, built from `relux-works/curator` main revision
`665dd13f7b6ab0408300070ef68d895f2d03b79f`.

From a fresh checkout, verify Curator, install the exact project-local skill
revisions and agent adapters pinned in `Skillfile.json`, and check the managed
state:

```bash
curator --version
curator install
curator status --check
```

Configure the required repository-local Git identity and SSH signing policy:

```bash
git config --local user.name "Ivan Oparin"
git config --local user.email "oparin@me.com"
git config --local gpg.format ssh
git config --local user.signingkey /Users/iv/.ssh/ivanopcode.pub
git config --local commit.gpgsign true
git config --local tag.gpgsign true
```

Contributor and agent workflow rules live in [AGENTS.md](AGENTS.md).
`CLAUDE.md` is a relative symlink to that canonical file.

## Implementation Plan

The live `task-board` contains five implementation milestones (`M0` through
`M4`) with 62 Stories and 186 atomic agent-executable Tasks. The dependency
closure of the final release Task contains every implementation Task, and the
machine-derived milestone critical path is:

```text
M0 contract foundation
  -> M1 single-host durability
  -> M2 multi-host preview
  -> M3 daily-driver tmux
  -> M4 cloning, Directory, platforms, and product release
```

All optional human UX, pilot, fidelity, and product go/no-go work is isolated
in a separate advisory Epic with no hard dependency into the implementation
DAG. It cannot block autonomous agent execution.

See [.spec/README.md](.spec/README.md) for the pinned specification source,
section coverage, board IDs, counts, and execution rules. Generated planning
snapshots live in [.planning/](.planning/); the live task-board remains the
planning authority.

## Managed Skills

`Skillfile.json` targets the Curator-standard `claude_code` and `codex_cli`
agent adapters and pins this project-local skill to an exact revision:

| Skill | Revision |
| --- | --- |
| `relux-works/skill-go-testing-tools` | `90c1515239eed9321068f3bafbeb5d0a0c2aa26a` |

Project management is intentionally global. The repository does not declare or
install `project-management`; development uses the globally installed skill
and `task-board` CLI.

Curator owns `.agents/`, `.claude/skills/`, and `.codex/skills/`. Do not edit
their generated contents directly; change `Skillfile.json` and rerun Curator.

## Tools and Outputs

| Tool | Purpose | Command or entry point | Outputs |
| --- | --- | --- | --- |
| Curator | Pin, install, and validate project skills | `curator install`; `curator status --check` | `.agents/`, `.claude/skills/`, `.codex/skills/` |
| `task-board` | Track scope, lifecycle, checklists, evidence, dependency waves, and the critical path through the global `project-management` installation | `task-board q 'plan()'`; `task-board q 'plan(TASK-260830-55kcni, mode=related)'`; `task-board plan --save` | `.task-board/`; `.planning/`; task outcome resources |
| Go toolchain | Build, test, and measure future Go implementation work | `go test ./... -v`; `go test ./... -cover` | Go build cache; test output captured under `.temp/<TASK-ID>/` when needed |
| Git | Branch, diff, and create signed commits/tags | `git status`; `git diff --check`; `git commit -S`; `git tag -s` | Git objects and refs under `.git/` |
| GitHub CLI | Inspect and open pull requests after bootstrap | `gh pr create`; `gh pr checks` | Pull requests and checks on GitHub |

Temporary validation logs and worktrees belong under `.temp/<TASK-ID>/` and
must not be committed. Diagrams, when implementation work introduces them,
belong under `diagrams/`.

## Contribution Baseline

All non-bootstrap changes are delivered automatically through pull requests as
author-signed commits, then landed by fast-forwarding the exact reviewed head
into `main`. Before creating a branch or task-scoped worktree, fetch
`origin/main`, fast-forward local `main`, and verify the refs are equal. Track
the work in `task-board`, use the managed `go-testing-tools` skill for Go
changes, use the global `project-management` skill for orchestration, run
relevant validation, and attach task-scoped evidence before review. See
[AGENTS.md](AGENTS.md) for the full contract.
