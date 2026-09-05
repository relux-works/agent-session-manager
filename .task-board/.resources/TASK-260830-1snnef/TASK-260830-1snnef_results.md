# TASK-260830-1snnef: terminal-backend conformance harness — outcome evidence

Leaf 3 of STORY-260830-3m2mw8, built on accepted leaves 1–2 at `adba01f`.
Work left untracked in the worktree; no commit (board checkpoints).

## Deliverable

`internal/terminalbackend/conformance.go` (new, ~990 lines): the Section 4.B
"internal semantic interface and conformance harness" proving five properties
at production entry points:

| Property | Entry point | Refusal codes |
| --- | --- | --- |
| Lifecycle semantics (§4.C, §4.1) | `CheckTransition`, `CheckErrorAllowed`, `ParseInstanceState`, `ParseOperation`, `ParseSideEffect` | `terminal_backend_protocol_error`, `local_precondition_failed` |
| Attach ownership neutrality (§4.A, §4.C) | `ParseAttachAuthorization`, `CheckAttachRequest`, `CheckAttachResult` | `terminal_backend_manifest_probe_mismatch`, `terminal_backend_unauthorized` |
| `ax pane` enforcement (§4.A, §4.1) | `CheckEntrypoint` | `local_precondition_failed` |
| Replication exclusions (§4.E) | `ClassifyReplication`, `CheckReplicable` | `terminal_backend_protocol_error` |
| Historical translation (§4.E) | `TranslateLegacyBackend`, `ProjectToLegacy` | `incompatible_schema` |
| Crash/idempotency (§4.C, §4.1) | `IdempotencyKey`, `Ledger.Bind/Replay/Export`, `ImportLedger` | `terminal_backend_protocol_error`, `idempotency_mismatch` |
| Status lookup rules (§4.C) | `CheckStatusResult` | `terminal_backend_protocol_error`, `local_precondition_failed` |

Five new wire codes (`terminal_backend_protocol_error`,
`terminal_backend_unauthorized`, `terminal_backend_unavailable` (sets only),
`local_precondition_failed`, `idempotency_mismatch`, `incompatible_schema` —
all members of the pinned catalog vocabulary, proven by the extended
`TestWireCodesAreCatalogued`).

`internal/terminalbackend/refusal_arm_inventory_test.go` (new): AST-derived
inventory closing leaf 2's disclosed weakness. Derives 193 refusal arms from
production sources (never consults the table), checks both directions
(every derived arm declared, every declared arm derived), binds rows to
production const wire values, resolves each row to an existing test (detail
mention, or entry plus wire/verified-predicate mention), pins the one
dynamic-detail funnel exactly, allowlists plain-error construction to two
functions, and pins five defensive re-parse bounds as exactly those five.
Fail-closed on empty, short (file-globbed, reverse-checked), unparseable,
or non-static derivation.

## Gaps the derivation closed while building it

- `Registry.AdmitProbe` on a nil registry: no test drove it → new
  `TestAdmitProbeRefusesNilRegistry`.
- `ParseEvidence` on malformed `observed_at`/`expires_at`: no case drove the
  two timestamp arms → two cases added to `TestEvidenceDocumentRefusals`
  (leaf-2 table, additive; timestamp parse precedes identity recomputation).
- Seven dead-by-construction arms with stated bounds: two `evidence
  canonical bytes`, five defensive re-parses.

## Mutant battery (copy-aside, grep-confirmed present, `cmp`-clean restore)

| Mutant | Shape | Result |
| --- | --- | --- |
| M1 new arm + new detail in `CheckTransition` (reviewer's attack) | addition | forward names the arm + reverse count break, exit 1 |
| M2 existing detail reused in a new function | addition | forward fails on the function key, exit 1 |
| M3 `create` sources widened with `active` | narrowing | gate fails at `error = <nil>`, exit 1 |
| M4 `attach_descriptor` reclassified safe | narrowing | pin + gate fail, exit 1 |
| M5 relay refusal deleted | deletion | gate + both inventory directions fail, exit 1 |
| M6 legacy map `tmux`→`ax.conpty` | swap | three translation tests fail, exit 1 |
| M7 status key floor removed | narrowing | gate fails at `error = <nil>`, exit 1 |
| M9 one table row deleted | declaration | forward + bijection fail, exit 1 |

8/8 killed, 0 survivors.

## Gates (each standalone, real exits)

- `go test ./... -count=1`: exit 0, 14/14 packages ok.
- `go test ./... -count=1 -race`: exit 0, 14/14 packages ok.
- `go vet ./...`: exit 0.
- `gofmt -l internal/`: clean (no output).
- `go run ./internal/traceability/cmd/tracecheck`: exit 0,
  `acceptance_cases=76` (new `terminal-lifecycle-conformance` case registered;
  section bindings untouched, still `unevidenced` with honest gaps).
- `go test ./internal/terminalbackend/ -cover`: 90.0% of statements.

## Stated bounds

- M71/M72 (JCS) stay reported as unknown, per inheritance; M92/M95 stay
  survivors (leaf-2 bounds, untouched).
- The inventory proves every arm is declared with a resolving test, not that
  each test decides its arm: decision proof is each row's named test and the
  mutant battery above.
- `CodeUnavailable` intentionally carries no arm (allowed-error sets only).
