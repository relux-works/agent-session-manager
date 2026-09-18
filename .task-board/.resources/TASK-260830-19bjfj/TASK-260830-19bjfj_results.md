# TASK-260830-19bjfj results — fuzz-rpc-operations-and-version-skew (final leaf, rev2)

- Task: TASK-260830-19bjfj, STORY-260830-4qojoz (mesh-rpc-framing-and-negotiation)
- Authority: `internal/specdoc/SPEC.v0.7.0.md` (§11.2-11.3, §17.1-17.4).
- Candidate: UNCOMMITTED working tree on `task-board/story/STORY-260830-4qojoz`
  over base `2fc6d50` (trunk v0.7.0 adoption) + replayed checkpoint `544c389`
  (predecessor TASK-260830-219okr, `refresh_advanced` with a checkpoint-bound
  LOGBOOK replay). No commit made.
- Character: TEST-ONLY. No production behavior file changed; the only
  production-file edit is the traceability pin constant (re-derived, below).
- Rework scope: rev1 (base `888ae3d`) was reviewed CHANGES REQUESTED by
  RUN-260917-81d7ae. The reviewer reproduced the whole technical body (N1-N3
  closure, every mutant, all fuzz controls, the 23/29 AC ratio) and blocked
  only on the story-close half (stale base, v0.6.0 registry, one invented
  binding) plus eight P3 test gaps. This revision fixes exactly that; the
  rev1 technical body is carried, not rewritten.

## AC coverage: 23 of 29 rows driven, 6 stated bounds

Matrix: `TASK-260830-19bjfj_conformance-matrix.md`. Every DRIVEN row names a
committed test through the production entry; every BOUND names the missing
entry and its owner. No row claims an unparsed operation, an unmeasured gate,
or a capability the tree does not have. The ratio is measured on the
refreshed tree, not carried from rev1.

## Finding-by-finding rework table

| Finding | Change | Test that fails without it | Evidence |
| --- | --- | --- | --- |
| P1-1 stale base / v0.6.0 registry | `refresh-candidate` onto `2fc6d50` (`refresh_advanced`); all 5 overlap paths reconciled on trunk bytes; registry work re-created in `ownership.v0.7.0.json` (5 cases, 3 bindings); clause lines re-measured through the production inventory (11.2: 7364/7408/7420; 17.1: 15567/15589/15602; 17.4: 15661); digest re-derived `b4ae9091…`; README + LOGBOOK recomputed | `TestVerifyRepositoryAcceptsExactOwnership` (140/56 figures), `TestV070RegistryRederivesFromTrunkV060Registry` (upgrade pins), `TestREADMEMeasuredCoverageMatchesTracecheckReport` (fenced line + 8 prose figures) — each fails on the stale v0.6.0 figures | `evidence/tracecheck.log`, `evidence/tracecheck-section-*.log`, `evidence/digest-derivation.log` |
| P1-1 README pin map | Closed number-word map deliberately extended with `eight`→8 and `forty-nine`→49; `partial`/`sliver`/`unevidenced` literals re-pinned | `TestREADMEMeasuredCoverageMatchesTracecheckReport` (fails on any unmapped spelling or stale figure) | `evidence/test-all.log` (tracecheck package ok) |
| P2-1 invented 11.3#18 binding | Binding dropped; 11.3 stays the carried unevidenced stub; `rpcwire-inventory-arrays` not ported; headline `clauses_discharged` carried through README + LOGBOOK as 56/569 | `TestV070RegistryRederivesFromTrunkV060Registry` fails if the binding is re-added (carried-stub equality arm) | `evidence/tracecheck-section-11.3.log` (0/27 unevidenced) |
| P3-1 opaque-body object gate | `TestOpaqueBodyMustBeObject` (new): `[]`/`null`/`"x"`/`42`/`{"a":1,"a":2}` over opaque `health.get` → `ErrFrame` through `DecodeRequest`; `FuzzClosedOperationBodies` feeds raw bodies with a new I-JSON object oracle + 5 non-object seeds; probe `opaque-admits-nonobject-body` added | Probe killed by `TestOpaqueBodyMustBeObject/array…` (exit 1); fuzz target FAILS under the plant (exit 1, `admitted request body is not a JSON object`) | `evidence/mutants-rpcwire/opaque-admits-nonobject-body/`, `evidence/p3-controls/p3-1-bodies-object-oracle.log` |
| P3-2 changed body after commit | `TestRPCChangedBodyAfterCommitRefusesMismatch` (new): `Create(moved)` after `setupPhase(PhaseCommitted)` → literal `idempotency_mismatch`, single retained journal, phase stays committed | One-off J1 overlay (both replay sites exempt at `PhaseCommitted`): new test FAILS (exit 1), staging `TestCreateReplayAndConflict` still passes (exit 0) | `evidence/p3-controls/j1/` (planted.go, kill.log, staging-still-passes.log) |
| P3-3 member-name-only oracle + missing seed | Oracle bound stated explicitly in the target doc comment and README (identity + object gate + round-trip + member sets; bounds/vocabularies owned by the unit suite); hello-body-extra seed added to `FuzzUnknownFields` | `FuzzUnknownFields` FAILS under the hello-extra plant on the new seed (exit 1, `admitted hello body is not the exact member set`) | `evidence/p3-controls/p3-3-unknownfields-hello-body-seed.log` |
| P3-4 encode edge | `encode-refuses-8388609` rebuilt as an exact edge witness: pad-1 frames a measured 8388608-byte line (admitted), pad frames 8388609 (refused) | The subtest asserts the literal 8388608 length; any off-by-one fails it | `evidence/test-all.log` (`TestFramingBounds/encode-refuses-8388609` PASS) |
| P3-5 refusal class | Three `TestErrorSchemaNotNegotiated` lanes assert `errors.Is(err, axerror.ErrVersionMismatch)` | Probe `failure-binding-v2-admits-13` still killed (exit 1) incl. the class lanes | `evidence/mutants-rpcwire/failure-binding-v2-admits-13/` |
| P3-6 production-derived expectations | `wantLocal` replaced by literal `[2 3 4]` / `[5]` in `TestPeerOfferKeyMembershipPerKey` and `TestNegotiateOfferRefusalCarriesLocal` | Config-table probes (`config1-admits-3` …) now fail through the literal comparison (exit 1) | `evidence/mutants-meshneg/config{1,2,3,4}-*/` |
| P3-7 fixture-level alphabet plant | `nonce-admits-standard-alphabet` widened to a class narrowing (second `RawStdEncoding` arm: any canonical unpadded `+/` spelling of 16+ bytes) | Killed by `…/nonce-standard-alphabet-refused` alone — padded/length lanes still pass under the plant | `evidence/mutants-rpcwire/nonce-admits-standard-alphabet/` |
| P3-8 evidence hygiene | Raw logs attached for gofmt, build, vet, cataloggen `-check`, both cross-builds, JSON census (each with its exit); matrix row 22 cites `hello_half_frame_close` | — (docs + logs) | `evidence/validation/`, matrix row 22 |

Finding from the rework itself: the naive P3-1 plant shape
(`err != nil && op != "health.get"`) is vacuous — the assigned `err` still
propagates through the later `if err != nil` and the plant admits nothing
(verified: SURVIVED with exit 0). The committed probe restructures the gate
so the exemption genuinely admits, and is killed.

## What was delivered (rev2 delta on top of the carried rev1 body)

1. Story-close rebase: 5 overlap paths reconciled on trunk bytes; registry
   ported to v0.7.0 with re-measured lines; digest `b4ae9091…`; README
   subsection equals the printed report; re-derivation test extended with
   reviewed-upgrade pins; LOGBOOK recomputed.
2. `TestOpaqueBodyMustBeObject` + `opaque-admits-nonobject-body` probe
   (rpcwire harness now 41 killed / neutral passed / `error-text`
   SURVIVED, exit 0).
3. `FuzzClosedOperationBodies`: raw-body feed, I-JSON object oracle,
   5 non-object seeds (38 in-code seeds), stated oracle bound.
4. `TestRPCChangedBodyAfterCommitRefusesMismatch` (committed-phase member).
5. `FuzzUnknownFields`: hello-body-extra seed (7 in-code seeds).
6. Exact encode edge; `ErrVersionMismatch` class pins; literal offer sets;
   class-level nonce plant (meshneg harness still 37 killed / neutral
   passed / `detail-text` SURVIVED, exit 0).
7. README meshneg rows + coverage figures; TRACEABILITY closure updated
   (literal N2 sets, v0.7.0 bindings, `b4ae9091…`); one newest-first
   LOGBOOK entry.

## Explicit config statement

`task-board.config.json` is trunk (`2fc6d50`) plus exactly the 3 fuzz rows
from rev1, in the same shape (verified: `git diff` on the file shows only
the 3 added lines; trunk never touched the file). No other config byte
changed in rev2.

## Fuzz controls (every target proven able to fail, re-run on the new targets)

Each control applies a weakening plant via `go test -overlay` (source never
touched), runs the target at `-fuzztime=100x`, and expects FAIL on the seeded
must-reject input. Logs + planted sources: `evidence/fuzz-controls/<target>/`.

| Target | Plant | Result |
| --- | --- | --- |
| FuzzClosedOperationBodies | hello body admits one `extensions` member | FAIL exit 1: `admitted hello body is not the exact member set` |
| FuzzNamespaceVocabulary | inventory admits `credential` | FAIL exit 1: `invalid 2.0.0 namespaces admitted: "{"namespaces":["credential"]}"` |
| FuzzUnknownFields | envelope admits one `extra=true` member | FAIL exit 1: `admitted envelope carries 6 members, want exactly 5` |
| FuzzClosedOperationBodies (P3-1) | object gate exempts `health.get` | FAIL exit 1: `admitted request body is not a JSON object` |
| FuzzUnknownFields (P3-3) | hello body admits one `extensions` member (new seed) | FAIL exit 1: `admitted hello body is not the exact member set` |

Seed baselines measured with a fresh `GOCACHE` (no `testdata/fuzz` corpus
in the tree): bodies 38, nsvocab 22, unknownfields 7 in-code seeds.

## Digest derivation (gate-computed, not copied)

After editing the registry, with the pin zeroed, the gate computes the
reviewed projection (raw log: `evidence/digest-derivation.log`):

```text
$ go run ./internal/traceability/cmd/tracecheck
spec-to-code traceability check failed: ownership registry projection digest b4ae9091d95a5f5ece43e60f621b23ec476089f160f82ca36c4ad638c9900a34 differs from reviewed 0000000000000000000000000000000000000000000000000000000000000000
```

(The pre-pin run against the carried `c4cd46bf…` pin refused the same way,
computing the identical `b4ae9091…` digest.)
`reviewedOwnershipCanonicalSHA256` in `internal/traceability/traceability.go`
was set to `b4ae9091…9900a34` — the only production-file line changed, on
top of trunk's v0.7.0 structure. Full `tracecheck` is green:

```text
traceability ok: contracts=64 normative_sections=36 acceptance_cases=140 fixtures=33 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=68 full=2 partial=8 sliver=5 unevidenced=49 unmeasured=4 unowned=7 clauses_discharged=56/569
```

Section-scoped runs refuse admission (the gate requires `full`), quoting the
measured ratios verbatim:

- `-section 11.2`: `discharges 3/5 normative clauses, which is partial coverage`
- `-section 11.3`: `discharges 0/27 normative clauses, which is unevidenced coverage`
- `-section 17.1`: `discharges 3/6 normative clauses, which is partial coverage`
- `-section 17.4`: `discharges 1/4 normative clauses, which is sliver coverage`

(Full refusal texts with gaps: `evidence/tracecheck-section-*.log`.)

## Inherited advisories N1-N3 (all closed, carried from rev1)

- N1: sweep kills per-key exemption plants at every key (reviewer's S7/S8/S9
  class); missing-key cases pin the length arm, substituted-key cases pin the
  membership loop.
- N2: refusal `Local` asserted against the literal offer sets (`[2 3 4]`,
  `[5]`, never derived from production) on all shape/mixed sweep rows plus
  5 dedicated mixed rows; kills S12 (`negotiate-drops-local-fact`).
- N3: `refuse-local-names-1` row beside a common major; reachability bound
  stated: `LocalMajors` never yields 1, so only a direct `Select` caller
  observes the arm.

## Validation (all 30 commands run by the producer on the refreshed tree)

| # | Command | Exit |
| --- | --- | --- |
| 1 | `test -z "$(gofmt -l …)"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0, 38 ok, 2357 top-level passes (the `FAIL` text inside is quoted mutant-kill output of the passing `TestSmokeMutantsAreKilled`) |
| 5 | `go test … -race -count=1 -timeout 25m` in 5 chunks (R1 sessquery alone: 312.889s) | 0, 38 distinct ok, no DATA RACE |
| 6 | `go test ./... -cover -count=1` | 0, 38 ok |
| 7-23 | 17 fuzz smokes incl. the 3 leaf targets | all 0 |
| 24 | `go run ./internal/traceability/cmd/tracecheck` | 0 (56/569) |
| 25 | `cataloggen -adopted … -check` | 0, tree unchanged |
| 26-27 | `GOOS=linux/windows go build ./...` | 0 / 0 |
| 28 | JSON census over tracked `*.json` | 0 |
| 29 | `task-board validate` | 0 (185 pre-existing MISSING_ACTIVITY warnings only) |
| 30 | `git diff --check` | 0 |

Package coverage: rpcwire 99.1%, meshneg 87.8%, matjournal 77.0%,
hostchannel 84.1%. Mutants: §2/§6 above; per-plant raw logs +
`results.json` + `table.md` in the evidence tar. Nothing was accepted from
rev1 evidence; every row above was executed in this session on the
refreshed tree.

Raw logs: `evidence/test-all.log`, `evidence/test-cover.log`,
`evidence/race-R*.log`, `evidence/fuzz-smokes/*.log`,
`evidence/tracecheck*.log`, `evidence/task-board-validate.log`,
`evidence/validation/*.log` (commands 1, 2, 3, 25, 26, 27, 28, 30).

## Incidents (no evidence depends on them)

1. Carried rev1 note: an early shell `>` typo truncated
   `internal/matjournal/journal_test.go`; rev1 restored it byte-identical.
   Verified clean in rev2 (`git status` shows no modification).
2. Rev2 note: a pin-bite check (`7364`→`7365`) proved both the rederivation
   test and the production gate refuse a drifted line; the follow-up
   `git checkout` briefly reverted the whole v0.7.0 registry, and it was
   re-ported by the deterministic script (identical bytes: tracecheck
   green on the same `b4ae9091…` digest, backup kept at `/tmp/own70.good.json`
   during the session).

## Candidate file inventory (`git status`)

Modified (13): LOGBOOK.md, README.md, internal/meshneg/TRACEABILITY.md,
internal/meshneg/mutations.py, internal/meshneg/negotiate_test.go,
internal/rpcwire/mutations.py,
internal/traceability/cmd/tracecheck/main_test.go,
internal/traceability/cmd/tracecheck/readme_coverage_pin_test.go,
internal/traceability/ownership.v0.7.0.json,
internal/traceability/registry_rederivation_test.go,
internal/traceability/traceability.go,
internal/traceability/traceability_test.go, task-board.config.json.
New (4): internal/matjournal/rpc_replay_test.go,
internal/meshneg/adversarial_test.go, internal/rpcwire/adversarial_test.go,
internal/rpcwire/fuzz_adversarial_test.go.

## Stated bounds (summary; owners in the matrix)

22/24 §11.3 operation bodies opaque (no initiator/lease parser or
pre-mutation revalidation entry; the I-JSON object gate over opaque bodies
IS driven); lease/materialization embedded-type vocabularies unwitnessed at
RPC; no minor-selection entry; envelope `request_id` correlation-only (no
responder replay cache); chunk-data base64url unwitnessed (strictness pinned
at the nonce gate); RPC/meshneg purity (no durable writes —
crash/idempotency evidence adopted from the matjournal suite plus the new
committed-phase pin). 17.2/17.3 registry bindings keep their existing
unevidenced owners.
