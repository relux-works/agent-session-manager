# TASK-260909-2ez769 — typed write-path census rev12

## Gate

The production mutation boundary is audited by
`internal/config/writepath_census_types_test.go` through
`TestWritePathCensusGate` (`censusViolations` dispatches to
`censusTypedViolations`; the syntax walker in `writepath_census_test.go`
is a historical reference only). The census loads the active Go build
context with `golang.org/x/tools/go/packages`, resolves calls through
`go/types`, and fails closed for unresolved calls, non-`*types.Func`
targets, unlisted object identities, and unlisted call sites.

An interface or type-parameter edge is admitted only when all three
hold: (1) the resolved `*types.Func` selection object is the exact
inventoried method object for the enclosing declaration (package path,
receiver, method); (2) the call's source file/line/column equals an
inventoried site for that method in that declaration; (3) for the four
pair-writer declarations, a top-level fail-fast hold guard
(`requireHold`/`requireHoldForConfig`/`requireHoldForRoot` in an
`if`-with-`return` statement, no `else`) lexically precedes the
statement containing the call. A new call, a moved call, or an escaped
method value of an already-used interface method is a new edge and is
refused until it is reviewed into the inventory. Method spelling alone
never admits an interface edge.

Name-keyed maps (`censusOSMutations`, `censusRawMutations`,
`censusPairHelpers`, `censusTrustHelpers`, `censusCustodyHelpers`,
`censusGatedIdents`, `censusBackendMethods`) are used only for
fail-closed DETECTION of concrete, statically resolved calls and for
routing them to definition-identity checks
(`censusAllowedDefinition`: same package path, same declaration name,
same receiver); the former `censusIsMutationFunctionName`-style
interface admission predicate is removed. Concretely: `os.Rename`
matches exactly the `os`-package symbol; a method named `Rename` on
any other receiver is flagged unless the enclosing declaration is the
censused writer for that receiver. No name-keyed map admits an
interface or type-parameter edge.

Scope bound: this census executes for the active Go build context.
Windows custody has native build-tagged tests and cross-compilation
evidence, but no Windows runtime census execution is claimed.

## Admitted interface call sites (119 sites, 90 groups)

`HOLD` marks the four pair-writer declarations whose admitted sites
must additionally be dominated by the top-level hold guard. Positions
are `file:line:column` of the call expression in the rev12 candidate.

### internal/config

| Function | Edge | Sites | Hold | Production justification |
|---|---|---|---|---|
| Load | io/fs.FileInfo.Mode | loader.go 260:6 | no | Mode check on the loaded configuration file metadata. |
| loadAbsentConfig | io/fs.FileInfo.IsDir | loader.go 326:6 | no | Directory probe while synthesizing an absent configuration. |
| validateRootKinds | io/fs.FileInfo.IsDir | loader.go 518:7 | no | Root-kind directory validation for configured roots. |
| replaceDurably | migrationFileSystem.Stat | migration.go 406:15, 423:20 | yes | Pre/post-replace existence and identity stats for the atomic swap. |
| replaceDurably | io/fs.FileInfo.Mode | migration.go 410:6, 424:78, 438:100, 450:104 | yes | Mode checks on the files being replaced and staged. |
| replaceDurably | migrationFileSystem.Link | migration.go 417:12 | yes | Hard-link backup step of the durable replacement. |
| replaceDurably | migrationFileSystem.ReadFile | migration.go 422:24 | yes | Byte read used to verify staged content before the final rename. |
| replaceDurably | migrationFileSystem.Remove | migration.go 419:8, 425:8, 429:12, 443:7, 459:9 | yes | Cleanup of staging/rollback temporaries on each outcome path. |
| replaceDurably | migrationFileSystem.Rename | migration.go 442:12, 452:18 | yes | Atomic staging rename and final replacement rename. |
| syncDirectory | migrationFileSystem.OpenDirectory | migration.go 509:17 | no | Open the configuration directory for the post-replace fsync. |
| syncDirectory | migrationDirectory.Sync | migration.go 513:12 | no | Directory fsync that makes the rename durable. |
| syncDirectory | migrationDirectory.Close | migration.go 514:7, 517:9 | no | Directory handle close on success and error paths. |
| writeAll | io.Writer.Write | migration.go 496:19 | no | Byte sink for staged configuration content. |
| writeTempFile | migrationFileSystem.CreateTemp | migration.go 468:15 | no | Owner-only staged temporary file creation. |
| writeTempFile | migrationFile.Name | migration.go 472:9 | no | Staged file name for the subsequent atomic rename. |
| writeTempFile | migrationFile.Close | migration.go 474:7, 487:12 | no | Staged handle close on error and success paths. |
| writeTempFile | migrationFileSystem.Remove | migration.go 475:7, 488:7 | no | Staged temporary cleanup on error paths. |
| writeTempFile | migrationFile.Chmod | migration.go 478:12 | no | Owner-only mode enforcement on the staged file. |
| writeTempFile | migrationFile.Sync | migration.go 484:12 | no | File fsync before the atomic rename. |
| rollbackV4 | migrationFileSystem.ReadFile | migration_v4_apply.go 212:22 | no | Read of backup/source bytes during explicit rollback. |
| writeTempReplace | migrationFileSystem.Rename | migration_v4_helpers.go 137:12 | yes | Atomic install of the staged replacement bytes. |
| writeTempReplace | migrationFileSystem.Remove | migration_v4_helpers.go 138:7 | yes | Staging cleanup when the install does not proceed. |
| restorePresentEmptyMaps | reflect.Type.Kind | schema.go 558:38 | no | Closed-shape map restoration; pure reflection, no I/O. |
| restorePresentEmptyMaps | reflect.Type.Key | schema.go 558:38, 567:65 | no | Closed-shape map restoration; pure reflection, no I/O. |
| restorePresentEmptyMaps | reflect.Type.Field | schema.go 562:17 | no | Closed-shape map restoration; pure reflection, no I/O. |
| validateTerminal | BackendSettingsValidator.ValidateBackendSettings | validation.go 849:13 | no | Registered backend-settings validator callback. |

### internal/hosttrust

| Function | Edge | Sites | Hold | Production justification |
|---|---|---|---|---|
| (AuthRequest).Format | fmt.State.Write | authorize.go 35:9 | no | Redacted `%v` formatting; no filesystem I/O. |
| (Snapshot).Format | fmt.State.Write | authorize.go 42:9 | no | Redacted `%v` formatting; no filesystem I/O. |
| readCommitted | FileSystem.Lstat | authorize.go 89:15 | no | Existence probe for the committed trust document. |
| readCommitted | FileSystem.ReadFile | authorize.go 96:19 | no | Owner-only read of the committed trust document. |
| (Store).commitConfigBinding | FileSystem.CreateTemp | binding.go 319:17 | yes | Staged authoritative target-side binding record. |
| (Store).commitConfigBinding | StagedFile.Name | binding.go 323:10 | yes | Staged binding name for the atomic rename. |
| (Store).commitConfigBinding | StagedFile.Sync | binding.go 329:12 | yes | Binding fsync before the atomic rename. |
| (Store).commitConfigBinding | StagedFile.Close | binding.go 330:7, 333:12 | yes | Staged handle close on error and success paths. |
| (Store).commitConfigBinding | FileSystem.Chmod | binding.go 336:12 | yes | Owner-only mode enforcement on the staged binding. |
| (Store).commitConfigBinding | FileSystem.Rename | binding.go 342:12 | yes | Atomic publish of the authoritative binding record. |
| loadBindingAt | FileSystem.Lstat | binding.go 146:15 | no | Existence probe for a binding document. |
| loadBindingAt | FileSystem.ReadFile | binding.go 156:19 | no | Owner-only read of a binding document. |
| ensureOwnerDir | FileSystem.Lstat | custody.go 269:15, 292:14 | no | Directory existence/kind probes during custody setup. |
| ensureOwnerDir | FileSystem.MkdirAll | custody.go 278:12 | no | Owner-only directory creation during store setup. |
| ensureOwnerDir | FileSystem.Chmod | custody.go 283:12 | no | Owner-only mode enforcement on the store directory. |
| verifyLockFile | FileSystem.Lstat | custody.go 334:16 | no | Lock-file probe during custody verification. |
| verifyLockFile | io/fs.FileInfo.Mode | custody.go 338:7, 338:34 | no | Lock-file mode verification; no writes. |
| verifyOwnerDir | io/fs.FileInfo.IsDir | custody.go 300:6 | no | Directory-kind verification; no writes. |
| verifyOwnerDir | io/fs.FileInfo.Mode | custody.go 303:5 | no | Directory-mode verification; no writes. |
| verifyOwnerFile | io/fs.FileInfo.Mode | custody.go 351:6, 354:5 | no | File-mode verification; no writes. |
| platformModeOK | io/fs.FileInfo.Mode | custody_unix.go 15:9 | no | Unix mode comparison; no writes. |
| verifyOwner | io/fs.FileInfo.Sys | custody_unix.go 26:14 | no | Owner-identity extraction; no writes. |
| cleanupCredentialDirectory | FileSystem.Remove | enroll.go 206:6, 207:6, 208:6, 209:6 | no | Removal of partially staged credential files after a failed enroll. |
| readOwnerFile | FileSystem.Lstat | enroll.go 272:15 | no | Credential-file probe. |
| readOwnerFile | FileSystem.ReadFile | enroll.go 279:19 | no | Owner-only credential-file read. |
| writeCustodyFile | FileSystem.CreateTemp | enroll.go 162:17 | no | Staged credential-file creation. |
| writeCustodyFile | StagedFile.Name | enroll.go 166:13 | no | Staged credential name for the atomic rename. |
| writeCustodyFile | StagedFile.Close | enroll.go 170:7, 177:7, 181:7, 184:12 | no | Staged handle close on each outcome path. |
| writeCustodyFile | FileSystem.Remove | enroll.go 171:7 | no | Staged credential cleanup on the error path. |
| writeCustodyFile | StagedFile.Sync | enroll.go 180:12 | no | Credential fsync before the atomic rename. |
| writeCustodyFile | FileSystem.Rename | enroll.go 188:12 | no | Atomic install of the credential file. |
| writeCustodyFile | FileSystem.Lstat | enroll.go 191:15 | no | Post-install existence probe. |
| (EnrollmentMaterial).Format | fmt.State.Write | export.go 27:9 | no | Redacted `%v` formatting; no filesystem I/O. |
| (Store).JointCommit | FileSystem.Lstat | joint.go 254:15 | no | Committed-state probe inside the joint commit. |
| compensateLocked | FileSystem.ReadFile | joint.go 356:24, 365:20 | no | Marker/config reads during in-hold compensation. |
| markerPresent | FileSystem.Lstat | joint.go 532:12 | no | Pending-commit marker existence probe. |
| readCompensationMarker | FileSystem.Lstat | joint.go 401:19 | no | Marker probe before the compensation read. |
| readCompensationMarker | FileSystem.ReadFile | joint.go 411:19 | no | Owner-only marker read. |
| recoverLocked | FileSystem.Lstat | joint.go 467:15, 473:15 | no | Marker/trust probes during recovery. |
| recoverLocked | FileSystem.ReadFile | joint.go 480:19, 493:24 | no | Marker/trust reads during recovery. |
| removeMarkerLocked | FileSystem.Remove | joint.go 580:12 | no | Marker removal under the exclusive hold. |
| resolveFailedReplace | FileSystem.ReadFile | joint.go 301:24 | no | Post-failure configuration read for pin verification. |
| openLock | FileSystem.Lstat | lock_unix.go 41:12, 68:15 | no | Lock-file probes during atomic lock init. |
| openLock | FileSystem.CreateExclusive | lock_unix.go 51:12 | no | Non-replacing exclusive lock-file creation. |
| openLock | FileSystem.Chmod | lock_unix.go 62:12 | no | Owner-only mode enforcement on the lock file. |
| checkP256Point | elliptic.Curve.IsOnCurve | profile_checks.go 45:6 | no | P-256 point validation; no I/O. |
| (Store).Initialize | FileSystem.Lstat | transact.go 168:15 | no | Store existence probe during initialize. |
| cleanupStagedFile | FileSystem.Remove | transact.go 313:7 | no | Staged-file cleanup on the failure path. |
| commitDocument | FileSystem.CreateTemp | transact.go 276:17 | yes | Staged trust-document creation. |
| commitDocument | StagedFile.Name | transact.go 280:10 | yes | Staged trust name for the atomic rename. |
| commitDocument | StagedFile.Sync | transact.go 286:12 | yes | Trust-document fsync before the atomic rename. |
| commitDocument | StagedFile.Close | transact.go 287:7, 290:12 | yes | Staged handle close on error and success paths. |
| commitDocument | FileSystem.Chmod | transact.go 293:12 | yes | Owner-only mode enforcement on the staged trust. |
| commitDocument | FileSystem.Rename | transact.go 301:12 | yes | Atomic publish of the trust document. |
| loadCommitted | FileSystem.Lstat | transact.go 246:15 | no | Committed-trust probe. |
| loadCommitted | FileSystem.ReadFile | transact.go 256:19 | no | Owner-only committed-trust read. |
| readSnapshotLocked | FileSystem.Lstat | transact.go 133:15 | no | Snapshot-file probe. |
| readSnapshotLocked | FileSystem.ReadFile | transact.go 143:19 | no | Owner-only snapshot read. |
| syncDir | FileSystem.OpenDirectory | transact.go 332:17 | no | Open the store directory for the post-commit fsync. |
| syncDir | SyncedDirectory.Sync | transact.go 336:12 | no | Directory fsync that makes the commit durable. |
| syncDir | SyncedDirectory.Close | transact.go 337:7, 340:9 | no | Directory handle close on each outcome path. |
| writeAll | interface{Write}.Write | transact.go 319:19 | no | Byte sink for staged trust content. |
| (CredentialEntry).Format | fmt.State.Write | trust.go 67:9 | no | Redacted `%v` formatting; no filesystem I/O. |
| (TrustStore).Format | fmt.State.Write | trust.go 62:63 | no | Redacted `%v` formatting; no filesystem I/O. |

No non-callee interface method value was found in the production scan:
every interface-typed selector in `internal/config` and
`internal/hosttrust` production sources is the callee of an
inventoried call above.

## Other admitted edges

Function-valued parameters and locals (`censusIndirectPolicies`):
`applyV4`/`rollbackV4` `openStore` (test seam that opens the trust
store), `validateRootKinds` `stat` (filesystem probe seam),
`sshFlagValueRule` `permits` and `admitSSHArguments`/`admitSSHOption`
`rule` (pure SSH validators), `writeTempFile` `clean` (local staging
cleanup), hosttrust `fn`/`boundary`/`revalidate`/`replace`/`restore`
(lock/hold/joint-commit callbacks that run inside the caller's hold),
`unlock`/`unlockBootstrap`/`exclusiveUnlock` (local unlock closures),
`accept` (string-decode acceptor), and `failed` (compensation
outcome). Each is admitted only as the exact resolved declaration
object inside its owning function.

Named immutable values (`censusAllowedFunctionValues`,
`censusAllowedExternalFunctionValues`): `tomlDecode`,
`migrationError`, `configError`, `loaderError`,
`terminalCapabilitySet`, `trustError` (error constructors and the
immutable terminal set), and `elliptic.P256` (curve selector). The
single immediately invoked package initializer is the
`terminalCapabilitySet` construction
(`censusAllowedFunctionInitializers`). Function-valued struct fields
(`censusAllowedFunctionFields`): `sshOptionRule.permits` (pure SSH
validator), `Inputs.LookupEnv`/`Stat`/`ReadFile` (production input
seams).

Filesystem backend definitions (`censusBackendMethods`): the bodies of
the exact `(osFileSystem, ...)` and `(osMigrationFileSystem, ...)`
method pairs implement the seams, so direct `os` calls inside them are
the definition of the interface, not new write paths. A new method on
a backend receiver is scanned as a new write path. Gated pair helpers
(`replaceDurably`, `writeTempReplace`) may only be called inside a
`JointCommit`/`WithExclusiveHold`/`WithExclusiveHoldForConfig`
closure; trust/custody helpers (`commitDocument`,
`commitConfigBinding`, `removeMarkerLocked`, `writeCustodyFile`,
`writeTempFile`, `ensureOwnerDir`, `secureStaged`) only inside their
censused writer definitions.

## Negative evidence

- `TestWritePathCensusRejectsUnlistedInterfaceCallSite` (committed
  port of the CR11 `reviewer_callsite_test.go` probe) copies the
  module, plants real `filesystem.Rename` calls before AND after
  `requireHoldForConfig` in `replaceDurably`, and runs the production
  census plus a valid-hold witness in a child `go test`. The child
  witness must PASS (both plants rename real files through the
  production entry) while the child census FAILs with a
  `write-path census violation` for both unlisted sites.
- `writepath-interface-callsite-skip` preserves the allowlisted
  interface-method object identity while admitting the same method at
  every call site and before hold verification; it propagates its
  overlay into the child module and is killed by the regression above
  (child exit 0 under the mutant, outer test exit 1).
- `writepath-method-value-skip` preserves the raw-mutation symbol
  check while admitting backend method values; killed by
  `TestWritePathGateFlagsExecutableRogues/backend_method_value`.
- The `TestWritePathGateFlagsExecutableRogues` suite (28 subtests:
  direct/package-var/alias/returned-closure controls, aliased and
  dot-imported writers, map/slice/struct-field/parameter function
  values, promoted embedded methods, backend receiver/method-value
  rogues, interface/generic/type-parameter/channel/IIFE shapes, and
  real-file witnesses) plus
  `TestWritePathCensusFailsClosedOnUnresolvedCallee` and the five
  `TestWritePathGateFlagsUnlockedCaller` controls all pass; the
  reviewer's 17 standalone census plants are retained as killed.
- Full batteries (below) run the behavioral package suites, not only
  the static checker.

## Production call-site map

| Boundary | Production call site | Negative production evidence |
|---|---|---|
| Config4/legacy pair writers | `config.Migrate`, `PreviewV4`, `ApplyV4`, `RollbackV4`, `writeTempReplace`, `replaceDurably` | zero/released/foreign/wrong-path/symlink/foreign-root and stale-source tests |
| Pair resolution | `config.Load`/`ResolvePaths` → `localstore.ResolvePaths` | invalid class/context, empty override, platform-default and environment-bound tests |
| Trust store custody | `hosttrust.Open`, `ValidateCustody`, `Store.Initialize` | missing, malformed, unsafe mode, wrong kind, unreadable and binding tests |
| Exclusive pair hold | `WithExclusiveHoldForConfig`, `HeldExclusive.ValidateConfigPaths` | zero/released/foreign pair and foreign-state-root tests |
| Joint durable commit | `hosttrust.Store.JointCommit`, `Recover`, `convergeLocked` | marker tamper, source/replacement divergence, compensation and crash tests |
| Authorization generation | `AuthorizeDispatch`, `WithMutationAuthorization` | stale older/newer, revocation and intervention tests |

The source audit records exactly one non-test
`localstore.ResolvePaths` call in `internal/config/loader.go`; no
legacy free-string config/state tuple signatures remain in
`internal/`.
