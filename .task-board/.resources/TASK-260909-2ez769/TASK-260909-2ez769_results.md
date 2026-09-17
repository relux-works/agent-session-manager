# TASK-260909-2ez769 \u2014 producer outcome rev11

Status: ready for independent review. The managed Story worktree remains
uncommitted; this outcome does not claim review acceptance or main integration.

## Scope and provenance

- Task: `TASK-260909-2ez769`, Host Trust Store 1.0.0, credential lifecycle,
  authorization generations, and explicit Configuration 4.0.0 migration for
  `relux-works/agent-session-manager-spec@v0.6.0`, commit
  `0cbdf100dbf84df50c64f792b1f940e3a67859a6` (sections 6.6, 11.10.2,
  11.10.3).
- Candidate `HEAD=305875134f8d344eb86ef926c7fbca3fed85c49d`; protected
  `origin/main=8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`.
- Checkpointed SSH/peer work is retained. The RPC control backup is 15/15
  byte-identical and the parked RPC delta is 11/11 byte-identical, with no RPC
  implementation path absorbed by this candidate.

## Implementation delivered

- `internal/hosttrust` implements closed Trust Store 1.0.0 reading, owner-only
  custody, Profile-1 issuance/verification, explicit out-of-band enrollment,
  unique credential-to-host mapping, fresh-key bounded rotation/retirement,
  immediate revocation tombstones, current-generation mutation authorization,
  atomic lock/trust writes, crash recovery, and replication exclusion.
- Binding is explicit. The store-side `config-binding.json` is a derived index;
  the target-side `.ax-config-binding.json` record is authoritative and is
  written atomically under the store hold plus target bootstrap lock. Missing,
  malformed, unreadable, rebound, partial, downgraded, or foreign-root state
  refuses closed. A derived binding without its target-side record is refused
  on reopen.
- The only production path resolver is `localstore.ResolvePaths`. Pair-bearing
  entry points receive the opaque `localstore.ResolvedPaths` value with its
  unexported fields: `hosttrust.Open`, binding and hold validation,
  `JointCommit`, `LoadCoherent`, Config4 preview/apply/rollback, legacy
  migration, and every pair writer. Tests construct fixture pairs through the
  resolver with explicit request overrides; no caller-supplied free-string
  config/state tuple remains.
- Config4 remains an exact closed document: the normative `mesh.host_channel`
  table is unchanged, with no store-root field. Migration is explicit preview,
  exact-preview confirmed, complete-peer enrolled, durable apply, and explicit
  backup rollback. Config1/2/3 readers and accepted legacy behavior remain
  covered; direct legacy-to-v4 migration and implicit fallback refuse.
- Joint compensation restores and verifies source bytes under the same
  exclusive hold and preserves the marker when safe recovery cannot be proven.
  No private key or trust/lock/binding/backup material is replicated; only the
  explicitly exported public enrollment tuple is intended for OOB exchange.

## Acceptance matrix

The enumerated production acceptance coverage is **19 of 21 rows driven**.
Rows 15 and 21 are explicit bounds, not passing claims.

| # | Acceptance row | Production call / named test evidence | Result |
|---:|---|---|---|
| 1 | Closed Config4 | `config.Decode`; `TestDecodeConfiguration4Refusals` | pass |
| 2 | Historical compatibility + explicit migration | `config.Migrate`; `TestMigrateRefusesV4Target`, `TestMigrateRefusesV4Downgrade`, legacy stale/recovery tests | pass |
| 3 | Closed trust document | `hosttrust.DecodeTrust`; `TestDecodeTrustRefusals` | pass |
| 4 | Missing/corrupt/unreadable trust | `hosttrust.ReadSnapshot`; missing/corrupt/intervened-marker refusal tests | pass |
| 5 | Fresh issuance | `hosttrust.IssueCredential`; `TestIssueCredentialProfile`, `TestIssueSelfEnrollsActive` | pass |
| 6 | Exact profile + time bounds | `hosttrust.VerifyProfile`; profile, EKU/SAN, wrong-key, expiry tests | pass |
| 7 | Explicit tuple enrollment | `hosttrust.Store.Enroll`; `TestEnrollRefusals`, `TestEnrollBoundsPerHost`; OOB approval is an operator act | pass within bound |
| 8 | Mapping uniqueness | `DecodeTrust` and enrollment admission; duplicate credential/root/key/host tests | pass |
| 9 | Rotation/retirement bounds | `Rotate`, `MarkRetiring`; overlap and old-leaf-expiry tests | pass |
| 10 | Revocation tombstones | `Revoke`, enrollment; tombstone, reenrollment, and authorization-close tests | pass |
| 11 | Old-generation mutation refusal | `WithMutationAuthorization`; stale-generation and post-convergence tests | pass |
| 12 | Shared authorization serialization | `Open`, lock, mutation authorization, `Revoke`; first-open and concurrent serialization tests | pass |
| 13 | Coherent config+trust snapshot | `config.LoadCoherent`; surviving-reader, barrier, and revalidation tests | pass |
| 14 | Unix owner custody | `hosttrust.Open`, `ValidateCustody`; custody mode and directory-binding tests | pass on Darwin |
| 15 | Windows equivalent ACLs | Windows custody implementation and native tests | bound: no Windows runtime runner |
| 16 | Complete peers + selected credential | `config.PreviewV4`; `TestPreviewV4PeerEnrollment`, `TestPreviewV4CredentialGates` | pass |
| 17 | Exact preview + confirmation | `config.ApplyV4`; preview mismatch and confirm-with-drops tests | pass |
| 18 | Current source/generation apply | Config4 apply + `hosttrust.JointCommit`; stale source/generation and marker tests | pass |
| 19 | Crash-durable config/trust pair | `JointCommit`, `Recover`, convergence; apply/rollback crash and compensation tests | pass |
| 20 | Explicit backup rollback | `config.RollbackV4`; rollback and backup-classification/exclusion tests | pass |
| 21 | Secret exclusion across export surfaces | `MatchExcludedFromReplication`, `ExcludedConfigDirName`; matcher tests | bound: downstream consumers/diagnostics are unowned |

## Negative and narrowing-mutant evidence

Every production gate has a current executable negative path. Delete-only
mutants are not counted.

- Config4 current battery: one neutral control passed (exit 0); 13 unique
  narrowing/precision mutants were killed (exit 1 with the named expected test):
  `transport-ssh`, `preview-length`, `confirm-drops`, `apply-gen-direction`,
  `v4-explicit-label`, `compensate-source-swap`,
  `compensate-rollback-source-swap`, `replace-require-skip`,
  `legacy-revalidate-skip`, `replace-restore-skip`,
  `writepath-method-value-skip`, `writepath-interface-name-heuristic`, and
  `config-association-root-skip`; the last probe was rerun in `part4b` and is
  counted once. `config-association-root-skip` is killed by
  `TestWriteTempReplaceRefusesForeignStoreWithTargetBinding`.
  Raw per-plant JSON/logs are in `mutations-rev11-full-a` through `-d` and
  `mutations-rev11-part4b` under this task's `.temp` directory. The duplicate
  neutral control in `-d` is not counted as a second probe.
- Host Trust Store current battery: one neutral control passed (exit 0); 34
  unique narrowing/precision mutants were killed (exit 1 with named expected
  tests), covering custody, mapping, generation, expiry, revocation, marker,
  lock-init, convergence, compensation, pair association, and authorization
  gates. Raw per-plant JSON/logs are in
  `mutations-hosttrust-rev11-a` through `-e`. The first `hold-state-root-skip`
  attempt was an instrument compile failure because its weakened replacement
  left `stateRoot` unused; it is retained as an anomaly but excluded from the
  count. The corrected source-preserving narrowing mutant killed
  `TestHeldExclusiveRejectsForeignConfigStateRoot` in `-e`.
- Write-path census: `TestWritePathCensusGate`, unresolved-callee refusal,
  unlocked-caller controls, and executable rogue witnesses all pass. Exact
  `*types.Func`/interface-method object identity is required; unlisted
  interface/type-parameter calls, parenthesized callees, method values, and
  function-typed escapes refuse. The token-preserving
  `writepath-interface-name-heuristic` plant runs the behavioral suite and is
  killed, not merely rejected by source text.

## Validation evidence

All commands below used the task-scoped cache
`.temp/TASK-260909-2ez769/go-cache` where applicable.

| Check | Result | Evidence |
|---|---|---|
| Focused Config4 package | exit 0 | `config-all-rev11j.json` |
| Focused Host Trust package | exit 0 | `hosttrust-all-rev11k.json` |
| Focused localstore package | exit 0 | `localstore-all-rev11a.json` |
| Repository tests | `go test ./... -v`, exit 0 | `go-test-all-v-rev11.log` |
| Repository coverage | `go test ./... -cover`, exit 0 | `go-test-all-cover-rev11.log` |
| Touched-package race | config, hosttrust, localstore, peeridentity, secprim; exit 0 | `go-race-touched-rev11.log` |
| Static analysis | `go vet ./...`, exit 0 | `go-vet-rev11.log` |
| Native build | `go build ./...`, exit 0 | `go-build-rev11.log` |
| Cross build | darwin/amd64, linux/amd64, windows/amd64, all exit 0 | `go-build-cross-*-rev11.log` |
| Cross test compilation | `go test -exec /usr/bin/true ./... -run '^$'`, all 3 targets exit 0 | `go-cross-exec-*-rev11.log` |
| Module hygiene | `go mod tidy -diff`, exit 0 | `go-mod-tidy-rev11.log` |
| Formatting/diff | gofmt and `git diff --check`, exit 0 | `gofmt-rev11.log` and final check |
| Census/refusal inventory | exit 0 | `config-census-rev11.log` |

The ordinary cross `go test ./... -run '^$'` lane was also attempted. It
returned exit 1 only because the Darwin host tried to execute foreign test
binaries (`bad CPU type` / `exec format error`); those logs are retained as
`go-cross-*-rev11.log`. The succeeding `-exec /usr/bin/true` lane compiles the
same test packages without executing incompatible binaries.

Coverage highlights from the complete run: config 92.9%, hosttrust 76.2%,
localstore 83.7%; all package rows passed.

## Explicit bounds and handoff

- Windows ACL/custody runtime is not claimed without a Windows runner; cross
  build and test compilation are evidence only for compileability.
- Downstream replication consumers and diagnostics export are outside this
  repository/task and remain unowned. RPC transport, TLS, hello, admission,
  dispatch, and stream lifecycle remain with their owning task.
- Evidence covers the implemented fsync/atomic-replacement and crash/intervention
  protocol, not physical power-loss behavior beyond those tests.
- Candidate is intentionally uncommitted for the managed Story snapshot. No
  commit, rebase, branch switch, main push, or integration was performed.
