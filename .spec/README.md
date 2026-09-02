# AX v0.5.0 Implementation Source and Board Map

The implementation board is derived from the published normative AX
specification, not from mutable prose in this repository.

## Pinned authority

- Repository: `relux-works/agent-session-manager-spec`
- Release: `v0.5.0`
- Annotated tag object: `d3da6614a6c7bf119a88c9596a86c0853c22cfb9`
- Peeled commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`
- `SPEC.md` SHA-256: `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`
- Normative source: <https://github.com/relux-works/agent-session-manager-spec/blob/v0.5.0/SPEC.md>

The upstream specification remains normative. This file records the consumed
release and implementation ownership; it does not amend or supersede the
specification. The repository vendors one byte-exact, digest-verified copy of
`SPEC.md` under `internal/specdoc` purely so repository gates can compare their
artifacts against the real text; that copy carries no authority of its own.

The implementation consumes this identity through the embedded
[`internal/specpin/v0.5.0.lock.json`](../internal/specpin/v0.5.0.lock.json)
lock. The lock records the ordered 60-row Section 1.5 registry, the exact
v0.4.3 compatibility delta, and these shipped fixture identities:

| Fixture ID | Upstream path | SHA-256 |
| --- | --- | --- |
| `ax-session-directory-conformance-v1` | `fixtures/session_directory_conformance.json` | `a6351a83e25a3a909297ed20bd1f4a75622b10f536a06b164fff3b12cb66f2ce` |
| `ax-terminal-backend-conformance-v1` | `fixtures/terminal_backend_conformance.json` | `67de0d78d76c9c445c742af5c4c14ffa5cecd620d4cb07dc5497d391b421ad37` |
| `ax-v0.4.3-roadmap-terminal-realm-v1` | `fixtures/v0_4_3_roadmap_terminal_realm.json` | `6023ec0d1562e8868b8bef3dc41cfd66ea0b4a4054fbaf13d3aec504578a7f74` |

`internal/specpin.Current` and `internal/specpin.Verify` reject partial reads,
unknown fields, substituted source identities, contract drift, fixture drift,
and any byte-different lock.

[`internal/specdoc`](../internal/specdoc) additionally vendors the byte-exact
`SPEC.md` of this release as a verification input for repository fidelity gates,
accepted only when its SHA-256 equals the digest above. The upstream
specification remains normative: that copy is compared against, never amended,
extended, or republished as authority, and only test binaries read it.

This slice is read-only: it mutates no durable state and advertises no
provider, platform, backend, or CLI capability.

## Program shape

The board contains five agent-executable implementation Epics and one separate
advisory human-validation Epic:

| Milestone | Board Epic | Stories | Tasks | Fibonacci points |
| --- | --- | ---: | ---: | ---: |
| M0 — contract foundation | `EPIC-260830-37rgqn` | 10 | 30 | 211 |
| M1 — single-host durability | `EPIC-260830-1uwj54` | 12 | 36 | 342 |
| M2 — multi-host preview | `EPIC-260830-3v9jlg` | 12 | 36 | 370 |
| M3 — daily-driver tmux | `EPIC-260830-27bix9` | 7 | 21 | 205 |
| M4 — cloning, Directory, platforms, release | `EPIC-260830-391tge` | 21 | 63 | 744 |
| Advisory human validation | `EPIC-260830-1ewjla` | 4 | 12 | 49 |

The automated implementation scope is 62 Stories, 186 Tasks, and 1,872
Fibonacci points. Every implementation Story contains three atomic leaf Tasks
with explicit scope, acceptance criteria, three checklist gates, an estimate,
and a required review policy.

These implementation counts intentionally exclude the M0 planning-only
quality-assurance Story and its two independent reviewer Tasks. Those Tasks
validate this decomposition; they do not enter the product release closure or
inflate implementation estimates.

## Machine-derived critical path

The canonical project plan derives this milestone path from real `blocked_by`
edges:

```text
EPIC-260830-37rgqn (M0)
  -> EPIC-260830-1uwj54 (M1)
  -> EPIC-260830-3v9jlg (M2)
  -> EPIC-260830-27bix9 (M3)
  -> EPIC-260830-391tge (M4)
```

The dependency closure of final release Task `TASK-260830-55kcni` contains all
186 implementation Tasks. Its machine-derived leaf critical path contains 66
Tasks and terminates at signed release evidence/publication. Use:

```bash
.agents/bin/task-board q 'plan()'
.agents/bin/task-board q 'plan(TASK-260830-55kcni, mode=related)'
```

The human-validation Epic has no incoming or outgoing hard dependency to any
M0-M4 Task. It may run in parallel or later. Human studies may create newly
triaged Bugs or product decisions, but cannot silently block, complete, or
rewrite automated implementation/conformance work.

## Normative coverage map

| SPEC.md scope | Primary implementation ownership |
| --- | --- |
| §1 conformance, registry, common rules | M0 source/catalog, common types, records, CI |
| §2 product/operator model | M0 records; M1 state/name/profile; M3 routing |
| §3 architecture/local layout/SQLite | M0 storage; M1 services |
| §4 TerminalBackend/tmux/ConPTY/services | M0 TerminalBackend; M1 pane/tmux/Aqua/services; M4 Windows |
| §5 domain records/state/ownership | M0 records; M1 state/lease/checkpoint/provider identity |
| §6 Configuration 1/2/3 | M0 configuration |
| §7 Provider, Session Adapter, Directory Node protocols | M0 provider/terminal/companion frameworks; provider stories in M4 |
| §8 provider/platform contracts | M1 Codex/Claude gates; M3 lanes; M4 Windows/Gemini/Muse/Antigravity/Pi/Qwen |
| §9 task-board integration | M1 task-board bridge; M4 provider/task-board acceptance |
| §10 immutable objects/workspace/cloning/directory | M0 storage; M1 workspace; M2 ingestion/conflicts; M4 cloning/Directory |
| §11 mesh RPC/replication | M2 peer/RPC/Merkle/transfer/ingestion; M4 directory mesh |
| §12 workspace replication | M1 Git capture/materialization/groups; M2 transfer |
| §13 lifecycle/recovery/cloning/continuation | M1 local lifecycle; M2 transfer/takeover/recovery; M3 restore; M4 cloning/planner/executor |
| §14 CLI/operator experience | M0 CLI envelopes; M1/M2/M3 CLI; M4 query/TUI |
| §15 errors/exits | M0 errors; every mutation/transport story tests mappings |
| §16 security/threat boundary | M0 security primitives; M1 credential realm; M2 mesh; M3/M4 security gates |
| §17 compatibility/migration | M0 config/errors/catalog; M2 RPC; M4 fleet compatibility |
| §18 observability/operations | M1 events; M2 mesh health; M3 recovery views; M4 operations/retention |
| §19 implementation conformance/release | M0-M4 exit/gate Stories and final release closure |
| §20 specification publication/governance | M0 pinned source/traceability; M4 release evidence/governance |
| Appendices A-D | M0 source/CI traceability; all provider/platform fixtures; M4 final conformance |

## Execution rules

- Agents execute the live task-board DAG; Markdown snapshots are recovery and
  review artifacts only.
- Implementation starts from unblocked `to-dev` Tasks and follows dependency
  waves. It does not wait for the advisory human Epic.
- Every production change uses the Curator-managed `go-testing-tools` skill,
  task-scoped Story worktrees, a producer/reviewer cycle, and a pull request.
- Closed historical wire contracts are preserved. Unknown or unproven
  platform/provider/backend capabilities remain disabled and visible.
- No implementation claim exists merely because a board Task is present.
