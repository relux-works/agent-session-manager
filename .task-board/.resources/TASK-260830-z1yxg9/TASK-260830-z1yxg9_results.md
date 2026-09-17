# TASK-260830-z1yxg9 results — RPC envelope/hello + Host Channel 1.0.0

Role: developer. Branch work left UNCOMMITTED in the Story worktree for handoff snapshot.

## Deliverable

- `internal/rpcwire` completed for RPC 5.0.0: exact RPC-4 envelope/body shapes,
  25-key hello maps with `contracts.rpc` `["5.0.0"]`, 8 namespaces,
  namespace/cardinality validation, offered-limit floors, historical
  Structured Error framing (statically 1.3.0 for RPC majors 4 and 5 via a
  new `axerror` binding row), 8 MiB line bound, exact fixtures, refusal
  table, fuzz target.
- `internal/hostchannel` (new): Host Channel 1.0.0 over RPC 5.0.0 —
  mutual TLS 1.3 (`ax-host/1`, enrolled roots only, no tickets/cache/
  resumption), enrolled-identity supplement, authorized hello both roles,
  current-generation dispatch and mutation boundaries, specified Error
  1.3.0 refusal framing, Configuration-4 launch selection.
- Traceability gaps (11.2/11.3/11.8/11.9/11.10/15.1/15.3), README,
  LOGBOOK, and the `task-board.config.json` fuzz gate updated.

## AC accounting: 7 of 7 rows driven through production entries

| # | Row | Production call sites | Named tests |
| --- | --- | --- | --- |
| 1 | Request/response correlation | `hostchannel.Dial` + `Call` + `Serve` | TestFullHandshakeHelloAndOperation, TestHelloResponseCorrelation |
| 2 | Hello maps and nonce echo | `Serve`/`Dial` hello exchange | TestFullHandshakeHelloAndOperation, TestRefusesInvalidHello/* |
| 3 | Inventory namespace/cardinality | `Call("inventory.roots")` dispatch | TestInventoryRootsThroughChannel |
| 4 | Offered limits | hello validation both roles | TestFullHandshakeHelloAndOperation, TestOversizeHelloLine |
| 5 | Historical Structured Error framing | `Serve` refusal frames, `Dial` mapping | TestRefusesNonHelloBeforeHello, TestHelloFailureMapping/* |
| 6 | Bilateral admission before dispatch | `AuthorizeDispatch` at hello both roles | TestRefusesHelloUUIDMismatch, TestRefusesServerHelloUUIDMismatch, TestRefusesNotAllowlisted, TestRefusesDestinationMismatch |
| 7 | Deadlines and generation recovery | bounded reads/writes, stale gates, `WatchGeneration` | TestHelloDeadline/*, TestHelloWriteDeadline, TestDispatchTimeout, TestStaleGenerationAtDispatch/*, TestStaleGenerationAtMutation, TestDispatchExpiryClosesWithoutFrame, TestWatchGeneration |

Crash/idempotency: the channel writes no durable state (static + runtime
censuses), so no crash-recovery behavior is attested; trust commits stay
atomic in hosttrust. No unsupported capability is advertised: no `ax`
command, SSH launch, provider plugin, or object exchange exists.

## Validation (this run, worktree HEAD ae5a4cc)

- `go test ./... -count=1`: all packages ok, exit 0.
- `go test ./internal/rpcwire ./internal/hostchannel -race -count=1`: ok.
- Coverage: hostchannel 81.6%, rpcwire 97.8%.
- `go build ./...`, `go vet ./...`, gofmt check: clean.
- Fuzz gate `FuzzUntrustedEnvelopes` 100x: PASS.
- Traceability suite + README figures pin: green with reviewed digest
  `5102d3bce23e6f27264232332a300dbb89237dc7792673efe9ec039b1219ce08`.
- Mutants: hostchannel 1 neutral passed + 22 narrowing killed + 1
  documented harmless SURVIVED (client-cache); rpcwire 1 neutral passed
  + 34 narrowing killed (incl. v5-exact). Token-preserving
  census-evasion mutant passes the static census and is killed by the
  runtime census with the full suite running. Tables: see outcome tar.

## Conformance matrix

See `TASK-260830-z1yxg9_conformance-matrix.md`: every HC-* gate with its
positive/negative vectors, production entries, and killing mutants.

## Restoration

See `TASK-260830-z1yxg9_restoration.md`: 10 of 11 untracked packet files
restored verbatim with matching sha256; UNRESOLVED_QUESTIONS dropped as
superseded; 4 tracked deltas re-applied by hand onto the v0.6.0 tree.

## Stated bounds

- SSH-carrier parity by construction (abstract Stream, no SSH-impl
  branch); no live SSH carrier test.
- Async one-second invalidation close is the caller's `WatchGeneration`
  poll; boundaries (dispatch/send/mutation) are enforced in-package.
- TLS alert bytes owned by crypto/tls (asserted shape: alert-or-close,
  never JSON).
- 1 MiB handshake cap proven at the exact unit boundary; Go's 256 KiB
  message cap fires first end to end (documented, integration-bounded).
- Traceability: gap texts updated; coverage levels and clause
  enumerations unchanged, no new clause discharge claimed.
- Pre-existing worktree `.task-board/` checkout noise (34 files) predates
  this run and was not touched.

## Handoff

Ready for review. Candidate paths: `internal/rpcwire/`,
`internal/hostchannel/`, `internal/axerror/binding.go` (+test),
`internal/traceability/`, `README.md`, `LOGBOOK.md`,
`task-board.config.json`. No commit was made on the Story branch.
