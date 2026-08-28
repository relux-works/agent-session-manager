# Agent Session Manager (`ax`)

Agent Session Manager is the implementation repository for the local-first,
cross-host `ax` coding-agent session orchestrator.

## Project Status

Repository foundation is in progress. The Curator-managed agent environment is
bootstrapped, but no Go production implementation has been added yet.

Product behavior is defined by the
[normative AX specification](https://github.com/relux-works/agent-session-manager-spec/blob/main/SPEC.md).
When this repository and the specification disagree, the specification is
authoritative.

## Bootstrap

The machine used for repository bootstrap runs Curator
`v0.14.0-rc.3-106-g665dd13f`, built from `relux-works/curator` main revision
`665dd13f7b6ab0408300070ef68d895f2d03b79f`.

From a fresh checkout, verify Curator, install the exact skill revisions pinned
in `Skillfile.json`, and check the managed state:

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

## Managed Skills

`Skillfile.json` pins these repositories to exact revisions:

| Skill | Revision |
| --- | --- |
| `relux-works/skill-project-management` | `b491ce1b867e4248471290dd168e069b0ecd90ea` |
| `relux-works/skill-go-testing-tools` | `90c1515239eed9321068f3bafbeb5d0a0c2aa26a` |

Curator owns `.agents/`, `.claude/skills/`, and `.codex/skills/`. Do not edit
their generated contents directly; change `Skillfile.json` and rerun Curator.

## Tools and Outputs

| Tool | Purpose | Command or entry point | Outputs |
| --- | --- | --- | --- |
| Curator | Pin, install, and validate project skills | `curator install`; `curator status --check` | `.agents/`, `.claude/skills/`, `.codex/skills/` |
| `task-board` | Track scope, lifecycle, checklists, and evidence | `.agents/bin/task-board` or a validated `task-board` shim | `.task-board/`; task outcome resources |
| Go toolchain | Build, test, and measure future Go implementation work | `go test ./... -v`; `go test ./... -cover` | Go build cache; test output captured under `.temp/<TASK-ID>/` when needed |
| Git | Branch, diff, and create signed commits/tags | `git status`; `git diff --check`; `git commit -S`; `git tag -s` | Git objects and refs under `.git/` |
| GitHub CLI | Inspect and open pull requests after bootstrap | `gh pr create`; `gh pr checks` | Pull requests and checks on GitHub |

Temporary validation logs and worktrees belong under `.temp/<TASK-ID>/` and
must not be committed. Diagrams, when implementation work introduces them,
belong under `diagrams/`.

## Contribution Baseline

All non-bootstrap changes land through pull requests with signed commits.
Before creating a branch or task-scoped worktree, fetch `origin/main` and
fast-forward local `main`. Track the work in `task-board`, use the managed
`go-testing-tools` skill for Go changes, run relevant validation, and attach
task-scoped evidence before review handoff. See [AGENTS.md](AGENTS.md) for the
full contract.
