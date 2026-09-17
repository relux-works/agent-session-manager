# TASK-260830-z1yxg9 restoration record

Preserved packet: `.temp/TASK-260830-z1yxg9/preserved-before-credentials-260910/`
(manifest 15 entries). All untracked packet files were copied into the Story
worktree verbatim and verified byte-for-byte against the manifest before any
adaptation. Observed digests below are of the restored bytes at copy time.

## Untracked files: applied verbatim (10 of 11)

| Path | Expected sha256 | Observed | Disposition |
| --- | --- | --- | --- |
| internal/rpcwire/envelope.go | 0e322f0e349c57d4… | match | applied, then extended for RPC 5.0.0 |
| internal/rpcwire/hello.go | f5f9b47f7c47ad51… | match | applied, then extended for RPC 5.0.0 |
| internal/rpcwire/inventory.go | 9292376341a67da7… | match | applied, then extended for RPC 5.0.0 |
| internal/rpcwire/envelope_test.go | dfdedd01bea7a09… | match | applied, then extended to 5.0.0 vectors |
| internal/rpcwire/mutations.py | ea5cb417be7fcaec… | match | applied, v5-exact probe added |
| internal/rpcwire/README.md | c0424738e542c397… | match | applied, then updated to v0.6.0/RPC 5 |
| internal/rpcwire/testdata/contracts-v3.json | 0887f72cc115915d… | match | applied unchanged |
| internal/rpcwire/testdata/contracts-v4.json | 00802985148e84b4… | match | applied unchanged |
| internal/rpcwire/testdata/hello-request-v2.json | 3c92c8a84eac5a9b… | match | applied unchanged |
| internal/rpcwire/testdata/hello-response-v2.json | 44d6e97f3ea3b811… | match | applied unchanged |

(Full digests in the packet manifest; prefixes above are the first 16 hex
digits. Every restored file matched its manifest digest exactly.)

The restored tree passed `go test ./internal/rpcwire -count=1` unmodified
before adaptation began.

## Untracked files: dropped (1 of 11)

- `UNRESOLVED_QUESTIONS.md` (4e1b741274472a39…): DROPPED. It records the
  inbound identity association as an open decision; the v0.6.0 Section
  11.10 binding adopted by this task resolves it (mutual TLS 1.3 over
  enrolled host credentials before hello and dispatch, both hello IDs
  checked). Carrying the stale open question forward would misstate the
  decision state. The resolution is recorded here, in the rpcwire and
  hostchannel READMEs, and in LOGBOOK.

## Tracked deltas: re-applied by hand, adapted (4 of 4)

The packet's `tracked.patch` (README.md, ownership.v0.5.0.json,
traceability.go, task-board.config.json) targeted the pre-credentials
tree and does not apply mechanically to the current v0.6.0 tree. Each
delta was re-applied by hand onto the current files:

- README.md: RPC/host-channel section added at the tail; the stale "no
  RPC hello frame" prose in the 15.1/15.3 paragraph corrected. Counts
  (15.1 at 5/7, 15.3 at 2/3) unchanged: no new clause discharge claimed.
- internal/traceability/ownership.v0.6.0.json: gap texts for
  11.2/11.3/11.8/11.9/11.10/15.1/15.3 rewritten to describe the delivered
  implementation; coverage levels and clause enumerations unchanged.
- internal/traceability/traceability.go: reviewed projection digest
  updated to `5102d3bce23e6f27264232332a300dbb89237dc7792673efe9ec039b1219ce08`.
- task-board.config.json: rpcwire fuzz gate added to validation
  commands (position preserved: immediately after the cover line).

## Adaptations after restoration

- rpcwire: RPC 5.0.0 accepted in `major()`; `ContractProfile`/`Namespaces`
  extended (`n >= 4`); new `contracts-v5` fixture (v4 map with
  `contracts.rpc` `["5.0.0"]`); tests extended to 5.0.0 vectors with 6.0.0
  as the unknown-major probe; `v5-exact` mutant probe added.
- axerror (cross-owner, required by 11.10.1 "Structured Error remains
  statically bound to 1.3.0"): `{rpc,5} → Version130` row plus the pinned
  test row; the unbound-major test now probes RPC major 6.
- New package `internal/hostchannel` (channel/server/client/launch,
  five test files, mutations.py, README) implementing Host Channel 1.0.0.
