# TASK-260909-2ez769 \u2014 typed write-path census rev11

## Gate

The production mutation boundary is audited by
`internal/config/writepath_census_types_test.go` through
`TestWritePathCensusGate`. The census loads the active Go build context with
`golang.org/x/tools/go/packages`, resolves calls through `go/types`, and fails
closed for unresolved calls, non-`*types.Func` targets, and unlisted object
identities. It does not infer safety from method spelling.

Interface and type-parameter method calls are refused unless the exact
interface-method `*types.Func` object is present in the per-production-edge
allowlist. Allowed indirect edges are enumerated by exact package path,
enclosing declaration, receiver/method or function object, and written
justification. Parenthesized callees, method values, function-typed fields,
parameters/locals, callbacks, immediately invoked literals, and unresolved
writer-shaped calls are exercised by executable rogue plants; witnesses mutate
real temporary files.

The direct standard-library mutation rule remains restricted to the censused
filesystem backends. Runtime `HeldExclusive` validation independently rejects
zero, released, foreign-store, wrong-config, symlink, and foreign-state-root
capabilities before pair mutation.

Scope bound: this census is executed for the active Go build context. Windows
custody has native build-tagged tests and cross-compilation evidence, but no
Windows runtime census execution is claimed.

## Current executable controls

The current `TestWritePathGateFlagsExecutableRogues` suite includes the earlier
function-value and callback controls plus:

1. interface `Execute` implementation;
2. parenthesized interface callee;
3. generic/type-parameter method call;
4. type-parameter method value;
5. channel callback;
6. immediate function literal;
7. function-typed field/parameter/local and package-level writer values;
8. dot-imported and aliased writers;
9. promoted embedded backend methods;
10. runtime witnesses for function-value and interface-method writes.

`TestWritePathCensusFailsClosedOnUnresolvedCallee` proves unresolved writer
shapes do not silently enter the allowlist. Same-named impostor receivers and
unlisted method values are not admitted by the exact object-identity policy.

## Current negative evidence

- Config4: one neutral control passed; 13 unique narrowing/precision mutants
  killed with exit 1 and named behavioral failures. The set includes both
  `writepath-method-value-skip` and the token-preserving
  `writepath-interface-name-heuristic`, plus the foreign resolved-pair probe
  `config-association-root-skip` (rerun once in `part4b`, counted once).
- Host Trust Store: one neutral control passed; 34 unique narrowing/precision
  mutants killed with exit 1 and named behavioral failures. This includes
  convergence, compensation, marker, hold, custody, generation, expiry,
  revocation, and pair-association gates. The corrected
  `hold-state-root-skip` plant killed
  `TestHeldExclusiveRejectsForeignConfigStateRoot`; an earlier compile-only
  harness defect is excluded from the count.
- Raw subprocess outputs, source overlays, exit codes, and Markdown tables are
  under `.temp/TASK-260909-2ez769/mutations-rev11-full-*`,
  `.temp/TASK-260909-2ez769/mutations-rev11-part4b`, and
  `.temp/TASK-260909-2ez769/mutations-hosttrust-rev11-*`.

## Production call-site map

| Boundary | Production call site | Negative production evidence |
|---|---|---|
| Config4/legacy pair writers | `config.Migrate`, `PreviewV4`, `ApplyV4`, `RollbackV4`, `writeTempReplace`, `replaceDurably` | zero/released/foreign/wrong-path/symlink/foreign-root and stale-source tests |
| Pair resolution | `config.Load`/`ResolvePaths` \u2192 `localstore.ResolvePaths` | invalid class/context, empty override, platform-default and environment-bound tests |
| Trust store custody | `hosttrust.Open`, `ValidateCustody`, `Store.Initialize` | missing, malformed, unsafe mode, wrong kind, unreadable and binding tests |
| Exclusive pair hold | `WithExclusiveHoldForConfig`, `HeldExclusive.ValidateConfigPaths` | zero/released/foreign pair and foreign-state-root tests |
| Joint durable commit | `hosttrust.Store.JointCommit`, `Recover`, `convergeLocked` | marker tamper, source/replacement divergence, compensation and crash tests |
| Authorization generation | `AuthorizeDispatch`, `WithMutationAuthorization` | stale older/newer, revocation and intervention tests |

The current source audit records exactly one non-test
`localstore.ResolvePaths` call in `internal/config/loader.go`; no legacy
free-string config/state tuple signatures remain in `internal/`.
