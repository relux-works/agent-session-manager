# Host credential exclusion handoff (AC row 21) — from TASK-260909-2ez769

Producer: TASK-260909-2ez769 (host credentials + Config4 migration), STORY-260830-1kiyj6.
Status: the matcher API below is implemented and tested in this leaf; no
downstream consumer calls it yet. Row 21 is NOT consumer-driven until each
named owner below ships its obligation. This handoff persists the exact
per-surface contract so exclusion cannot be lost between leaves.

## Producer API (shipped, do not reimplement)

Package `internal/hosttrust` (`export.go`), tested by
`TestMatchExcludedFromReplication`, `TestExcludedConfigDirName`,
`TestCustodyLayoutFullyExcluded`, `TestEnrollmentExportNotExcluded`,
`TestApplyV4BackupIsClassifiedConfigCopy`:

- `MatchExcludedFromReplication(relative string) bool` — reports whether a
  STATE_DIR-relative path must stay out of replication, snapshots, cloning,
  Session Directory, logs and exported diagnostic bundles. Covers
  `host-channel/trust.json`, `host-channel/pending-commit.json`,
  `host-channel/credentials/`, `host-channel/lock`, plus transient staging
  files (`.trust-stage-*`, `.custody-stage-*`, `lock-stage-*`). Absolute
  paths and `..` escapes fail closed as excluded.
- `ExcludedConfigDirName(base string) bool` — reports whether a
  configuration-directory file name carries trust-adjacent material:
  `.bak.<version>` backups, `.pre-rollback.<version>` copies,
  `.ax-config-*` staging files.
- `ExcludedFromReplication() []string` — the listed control paths above.
- Explicitly NOT excluded: `ExportEnrollment` material. Explicitly selected
  public bytes (certificate/root, fingerprints) are the only operator
  exchange outside replication; never route private-key.pem, trust
  internals, backups or authorization caches through it.

## Per-surface obligations

| # | Surface | Owning task | Must call | Test obligation |
|---|---|---|---|---|
| 1 | Directory metadata replication writer | TASK-260830-13bbo0 implement-directory-record-namespace-and-merkle | `MatchExcludedFromReplication` on every record path before write; `ExcludedConfigDirName` for config-dir members | Negative test enumerating every excluded fixture path (trust.json, credentials/&lt;hex&gt;/private-key.pem, pending-commit.json, lock, staging names, .bak./.pre-rollback. names) that FAILS when any is admitted; plus a narrowing variant admitting exactly one excluded member |
| 2 | Metadata disclosure policy | TASK-260830-1ybn3u implement-metadata-disclosure-policies | Encode the exclusion as a deny rule evaluated before any allow rule | Policy test refusing each excluded class and admitting a neighboring allowed path |
| 3 | Clone prepare/stage/publish | TASK-260830-1kj7ae implement-clone-prepare-stage-and-publish | Both matchers at stage enumeration and at publish | Per-phase negative test failing when a custody path is staged or published |
| 4 | Clone bundle member types | TASK-260830-24z2b3 implement-clone-bundle-and-canonical-session-types | Bundle member allowlist must reject excluded paths at construction | Constructor negative test per excluded class |
| 5 | Chunked transfer manifest | TASK-260830-355og8 implement-chunked-transfer-and-resume | Manifest builder drops excluded entries before chunking | Manifest negative test failing when an excluded entry is chunked |
| 6 | Task-board bundle capture/restore | TASK-260830-1ifuz2 implement-task-board-bundle-capture-restore | Capture walker drops excluded paths; restore refuses archives containing them | Negative tests in both directions |
| 7 | Session Directory detail/preview panes | TASK-260830-1b162e implement-detail-preview-jobs-and-provenance-panes | Panes must not resolve, preview or render excluded paths or secret bytes; redact with a static placeholder | Pane test asserting the placeholder (never the bytes) for each excluded path |
| 8 | Diagnostic bundle writer (`ax doctor` export) | UNOWNED — no board task exists (searched 2026-09-16) | When its task is created it MUST consult both matchers and drop matches | Same negative-test shape as row 1 before the surface ships |

## Acceptance note for reviewers

Until every row above ships, AC row 21 counts as implemented-matcher-only,
not consumer-driven. This handoff plus the appended notes on the seven owned
tasks are the persisted obligation; the diagnostics surface is explicitly
flagged as unowned rather than waived.
