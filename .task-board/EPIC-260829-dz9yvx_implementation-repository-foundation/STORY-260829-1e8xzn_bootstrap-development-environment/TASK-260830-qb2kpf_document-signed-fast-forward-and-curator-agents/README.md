# TASK-260830-qb2kpf: document-signed-fast-forward-and-curator-agents

## Description
Document and execute automatic Ivan-signed PR delivery, permit explicit agent co-authorship, and keep project-management global.

## Scope
AGENTS.md, README.md, Skillfile.json, Curator-managed adapters, task metadata, removal of the tracked transient board journal, signed commits, PR record, and exact fast-forward synchronization to main.

## Acceptance Criteria
AGENTS.md requires automatic author-signed commits, PR review, exact signed fast-forward landing, fresh local main before new tasks, and permits explicit non-fabricated agent co-authorship; Skillfile.json targets claude_code and codex_cli and declares only go-testing-tools; local project-management adapters and shims are absent while the global skill and task-board remain usable; README matches; validation passes; the exact Ivan-signed commits are on local and remote main with a clean worktree.
