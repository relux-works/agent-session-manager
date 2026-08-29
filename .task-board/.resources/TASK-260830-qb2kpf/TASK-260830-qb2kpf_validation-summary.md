# Validation summary

- Curator-spec `profiles/manager.md` defines the canonical agent identifiers `claude_code` and `codex_cli`.
- `AGENTS.md` requires automatic author-signed commits, PR review, exact-head fast-forward landing, fresh `main`, and permits explicit agent co-authorship without fabricated identity.
- `Skillfile.json` declares both canonical agents and only the project-local `go-testing-tools` skill.
- Project-local `project-management` adapters, skills, and task-board shims are absent.
- Global `project-management`, `task-board`, `task-board-tui`, and `tb-sessiond` are current through Curator.
- `curator status --check .`: pass.
- `curator global status --check`: pass.
- `task-board validate`: pass, no issues.
- JSON shape, symlink, command-resolution, policy-presence, superseded-policy-absence, and `git diff --check` assertions: pass.
- The global source policy was separately landed through relux-agents-infra PR #14 at signed commit `77c96621a7ee4e510c920b215ccfc3c002d1e2b3`.
