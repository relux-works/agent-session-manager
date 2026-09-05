# TASK-260830-1zvmw7 evidence: terminal manifest / probe / capability admission

Scope: relux-works/agent-session-manager-spec@v0.5.0, §4.B (manifest/probe schemas, generation digest, reconciliation rules 1-6, omit-self identity), §4.D (16-row capability registry, claim rows, evidence schema, fact mapping, attestation), §6.5 (trust before probe; registry-bound admission), §7.A (generation-bound binding). Built on leaf-1 checkpoint `9481a20`; leaf-1 gate logic untouched (only a package-doc line and 3 added wire codes in leaf-1 files).

## Production deliverable (working tree, uncommitted per Change-Request rule)

- `internal/terminalbackend/manifest.go` (new, ~1850 lines):
  - Closed parsers `ParseManifest` / `ParseProbe` / `ParseEvidence` over a strict decoder (duplicate-key, invalid-UTF-8, depth-cap, trailing-data refusal). Unknown/missing members, non-empty `extensions`, vocabulary/bound/ordering/registry-row deviations, and ID-recompute mismatches all fail with `terminal_backend_manifest_probe_mismatch`. Error details are static clauses; no local data echoed.
  - 16-value capability registry transcribed from §4.D; claim `generation_variable`/`dependent_operations`/`evidence_requirements` must equal the row at parse time.
  - `GenerationDigest` (SHA-256 over `ax-terminal-backend-generation-v1\0` + raw generation); empty/over-long/invalid generation fails closed with `terminal_backend_stale_generation`.
  - `Reconcile` enforces rules 1-6 in deterministic order (identity equality with executable-substitution→untrusted mapping; protocol/platform membership; generation binding→stale; keyed claim relation; per-object evidence usability (tuple binding, `observed_at <= now < expires_at` liveness, `SignatureVerifier` attestation→integrity failure); per-claim requirement coverage via the 5-entry fact mapping; exact `evidence_ids` set equality), returning the sorted `Admitted` set.
  - `CapabilitiesForOperation` / `CheckOperation` gate the capability-confers-operation half of §4.C dependencies (`terminal_backend_capability_unproven`). Conditional §4.C dependencies (interactivity, credential need, attach overlap, provider observation) are explicitly owned by the lifecycle call site, not this file.
  - `(Registry).AdmitProbe` is the production entry point: manifest → `Resolve` (unknown identity is not-found, never default) → member-for-member record binding (drift vs executable-substitution→untrusted) → probe/evidence parse → `Reconcile`.
  - New wire codes (all in the pinned catalog vocabulary): `terminal_backend_manifest_probe_mismatch`, `terminal_backend_capability_unproven`, `terminal_backend_integrity_failure`, with `IsMismatch`/`IsCapabilityUnproven`/`IsIntegrityFailure` predicates; `TestWireCodesAreCatalogued` extended.
- `internal/terminalbackend/manifest_test.go` (new, black-box): fixtures built by an independent test-side transcription of the JCS identity and attestation-byte recipes with real RSA signing; positives pinning every member; ~30 parse refusals; ~20 reconciliation refusals each asserting the exact wire code; operation-gate table for all 10 operations; registry admission incl. external trust + credential-realm evidence + substitution refusal; empty-tuple, availability-non-gate, and no-aliasing pins.
- `internal/terminalbackend/manifest_pin_test.go` (new, white-box): drifted-override, drifted-addition, same-ID-conflict, and executable-substitution arms that no valid document can deliver (parse refuses first / ID is a hash), each mutated post-parse at the guard.
- `LOGBOOK.md`: 2340 entry recording the mis-targeted-test catch, the RSA-dedup finding, and the nil-vs-empty production bug.
- No CLI/doctor/README surface changed: no user-facing surface was added (admission is library-level; `ax terminal ... doctor` belongs to a later task), and traceability contract keys for the three schemas already exist, so no registry edit was needed. No unsupported capability is advertised: the vocabulary is closed at parse and `Admitted` only ever contains row-validated true claims.
- Crash/idempotency: admission performs no durable mutation (pure validation over input bytes; the catalog marks manifest/probe operations `EffectNoDurableMutation`), so no crash/idempotency evidence beyond this statement is owed.

## Measured verification (all run directly, exit codes observed)

- `go test ./... -count=1`: exit 0, no FAIL lines.
- `go vet ./...`: exit 0.
- `gofmt -l internal/ cmd/`: clean.
- `go run ./internal/traceability/cmd/tracecheck`: exit 0 (`traceability ok`, 74 cases).
- Package coverage `internal/terminalbackend`: 87.1% of statements; 208 passing test/subtest lines observed in-package.
- `go doc -all ./internal/terminalbackend` re-audited: 14 new exported decls; no exported path reaches `Registry.records` or any interior array (AdmitProbe resolves through the cloning `Resolve`; `Admitted` slices are freshly allocated; pinned by `TestAdmittedSharesNoBackingArrays`).

## Mutant protocol: 12/12 killed (12 killed of 12 applied)

Each mutant was grep-confirmed present in the guarded location, compiled, turned its target test red, and was reverted with `diff` proving byte-identical restoration. Invalid formulations were discarded, not counted (non-compiling expiry form; tautological-true expiry form that exposed the mis-targeted fixture, fixed and re-run; non-matching separator anchor caught by grep count 0).

| # | Mutation | Target test | Result |
|---|----------|-------------|--------|
| M1 | delete attestation verification | wrong_key_signature (integrity) | killed |
| M2 | neuter liveness upper bound | expired_evidence (mismatch) | killed (after fixture arm-isolation fix) |
| M3 | generation domain separator v1→v9 | golden digest | killed |
| M4 | evidence_ids exact→lower-bound-only | superset | killed |
| M4B | evidence_ids exact→listed-subset-only | unreferenced object | killed |
| M5 | allow override of generation-stable claim | override_of_stable | killed |
| M6 | drop claim ordering | unsorted_claims | killed |
| M7 | swallow executable substitution | substitution (white-box, untrusted) | killed |
| M8 | skip generation binding | stale_generation | killed |
| M9 | operation gate admits all | attach-unproven + empty-set status | killed (2 tests red) |
| M10 | allow non-empty extensions | non-empty_extensions | killed |
| M11 | admit empty raw generation | generation bounds | killed |

## Stated bounds (what this work does NOT claim)

- Split-facts evidence (requirement coverage spread across objects, none sufficient alone) is refused; each object must independently satisfy its claim (rule-5 singular reading, documented in `Reconcile`).
- `manifest`/`probe` operations confer through no capability: the gate returns unproven for them by design; they carry no capability dependency.
- Availability (`available|conditional|unavailable|unknown`) is observed, not admission-gating.
- Signature trust roots (release/host-incarnation key registry) are the caller-supplied `SignatureVerifier`; a nil verifier admits nothing. Real key wiring is a later task.
- Byte-identical duplicate evidence dedups to one ID; same-ID differing-bytes is refused.
