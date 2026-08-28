# TASK-260829-1m2pv0 Staged Diff Rework

## Change

Added a repository `.gitattributes` policy scoped to task-board spawn-log
resources:

```gitattributes
.task-board/.resources/*/*_spawn-log_*.log whitespace=-space-before-tab
```

Spawn logs preserve external command output verbatim. The policy disables only
Git's `space-before-tab` diagnostic for those generated logs; standard
whitespace checks still apply to source, documentation, other board outcomes,
and other whitespace error classes. No captured task-board evidence was
rewritten.

## Validation

| Command / gate | Exit | Result |
| --- | ---: | --- |
| Temporary-index `git diff --cached --check` before policy | 2 | Reproduced 19 `space before tab in indent` diagnostics, all in the preserved implementer spawn log. |
| Temporary-index `git diff --cached --check` after policy | 0 | Full staged repository candidate passes. |
| `git check-attr whitespace` scope check | 0 | Spawn log resolves to `-space-before-tab`; README and ordinary outcome resources remain unspecified/default. |
| `curator status --check` | 0 | Both pinned skills and all managed command artifacts are current. |
| Metadata validation | 0 | Exact Skillfile pins, canonical relative symlink, Git signing metadata/key, no-Go scope, and narrow attribute value pass. |
| `git check-ignore -q .temp/TASK-260829-1m2pv0/tool-readiness.log` | 0 | Task-local validation artifacts remain ignored. |
| `task-board validate` after attaching evidence and checking the rework item | 0 | Board is valid with no issues. |
| Final temporary-index `git diff --cached --check` after board mutations | 0 | The complete current repository candidate still passes. |

Go tests were not run because this task changes repository metadata and
documentation only; the repository has no `go.mod` and no non-ignored `.go`
files. Curator and repository metadata checks are the applicable build and
validation gates.

The existing independent reviewer verdict remains accepted with no findings.
This rework addresses the sole post-review staged-diff diagnostic without
changing reviewed content or generated evidence bytes.
