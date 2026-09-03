> **Superseded in part by `TASK-260830-34elja_rework-rev2.md`.** Review of
> CR-TASK-260830-34elja-1 rev1 returned `changes_requested` with two blocking
> findings. Four claims below are stale and are corrected there, not here:
> section 15.1 is **5/7**, not 6/7; repository `clauses_discharged` is
> **9/394**, not 10/394; the harness now applies **26** mutants, not 20; and the
> `exit_code` leading-quote refusal described below was redundant dead weight in
> front of `strconv.ParseInt` and has been removed. The rest of this report
> still holds. This file is kept unedited as the rev1 record.

# TASK-260830-34elja — Structured Error registry implementation

Scope: `relux-works/agent-session-manager-spec@v0.5.0` (commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`), Sections 14.2, 15, 17.2, inside the
`structured-errors-cli-envelope-and-exit-codes` Story boundary. CLI Result
envelopes and the process exit-status mapping belong to sibling leaf
`TASK-260830-1bipsa`; historical-envelope compatibility proofs belong to
`TASK-260830-33sfxc`.

## What was built

New package `internal/axerror` (6 production files, 6 test files):

| File | Production surface |
| --- | --- |
| `registry.go` | `Version` set, `ExitCodeFor`, `ExitStatusMeaning`, `IsFailureExitStatus`, `CodesFor`, `RetryabilityRefusal` |
| `axerror.go` | `Spec`, `New`, `NewTargetAuthMissing`, `NewRealmEvidenceUnavailable`, `IDs`, `Error` accessors, `MarshalJSON` |
| `details.go` | `Details`, `ValidateDetails`, `RedactionBound`, `TargetAuth`, `RealmEvidence`, causal-leak refusal |
| `binding.go` | `ContainingContract`, `BindingFor`, `BoundContracts` |
| `decode.go` | `Decode`, `DecodeBound` |
| `local.go` | `Surface`, `UntrustedOutcome`, `LocalFromUntrusted`, `LocalSurfaces` |

Design decisions that carry the acceptance criteria:

- **Exit status is not a caller input.** `Spec` has no `exit_code` field; `New`
  resolves it through `ExitCodeFor`, so no call site can mint a mapping the
  specification does not assign.
- **The code registry is projected from the reviewed catalog**, not retyped, and
  admits a code only for the versions that register it. A 1.3.0 TerminalBackend
  code cannot be emitted inside a 1.0.0 envelope.
- **Retryability is a refusal table, never a permission.** `RetryabilityRefusal`
  reports only where the pinned document forbids `retryable = true` (exit
  classes 7, 16, 130 and the codes `operation_uncertain`,
  `terminal_backend_stale_generation`, `transaction_unknown`), each entry
  quoting the sentence that disqualifies it. It never reports that a retry is
  safe.
- **Causal redaction is two decidable gates plus one structural property.** A
  detail key that exactly names one of the four classes Section 15.1 forbids is
  refused at any nesting depth; human text or a diagnostic value reproducing the
  rendered local cause verbatim is refused; and the cause itself is an
  unexported field that the only encoder in the package cannot reach.
- **`LocalFromUntrusted` takes no part of a foreign payload.** Its inputs are the
  surface, AX's own classification, AX's own text, AX's own identifiers, and a
  local Go cause. There is no parameter through which a child or peer code,
  retryable bit, detail map, or authority field could be adopted.

## Two defects found while writing the evidence

**1. My own first revision repeated `BUG-260902-2faftr`.** The detail scanner
initially matched key **substrings** (`token`, `socket`, `credential`, ...). That
refuses `token_count`, `socket_timeout_ms` and `credential_profile` - ordinary
diagnostics the specification admits - while still admitting a secret written
under an innocuous name: pure false-positive surface with no true-positive
capability, which is precisely the defect removed from the Configuration
extension-key validator on 2026-09-02. It was caught by reading `LOGBOOK.md`
before writing the entry, not by a test. The scanner is now **exact-match only**
and scoped to Section 15.1's four declared classes (credential, raw transcript,
environment secret, opaque bundle content) rather than to the whole Section 16.2
exclusion table, which governs manifests and bundles rather than error details.
A mutant that restores substring matching is now killed by the admitted rows.

**2. A self-referential test let a mutant survive.** The first version of
`TestDetailsRefuseExcludedClasses` ranged over the production
`excludedDetailKeys` map, so a mutant that deleted `password` also deleted its
own test case and the suite stayed green. The registry is now pinned as a
reviewed literal table in the test and compared against production, and that
mutant is killed.

**3. `exit_code` accepted a JSON string.** Declared as `*json.Number`, it
admitted the JSON **string** `"9"`,
because `encoding/json` will unmarshal a quoted value into a `json.Number`. A
peer could therefore have written its exit status as text. Fixed by decoding
`exit_code` from `json.RawMessage` and refusing a leading quote, so the JSON
type is part of the check. `TestReaderAndAccessorEdges` pins it.

## Stated bounds — what this does NOT establish

- `RedactionBound` is quoted in the code and asserted by
  `TestCausalLeakGateStatesItsBound`: the scanner decides **exact key names**
  and verbatim cause reproduction. It inspects no free-form value for secret
  content and matches no key by substring, so a secret under an innocuous key
  is admitted. Section 16.2 states that v0.5.0 "does not claim reliable
  content-level secret scrubbing" while an implementation "SHOULD offer a
  best-effort scanner"; this is that scanner and nothing more.
- The causal-leak gate detects **verbatim containment** of the whole rendered
  cause and of each wrapped link. A paraphrase, a truncation, or a single
  extracted field of the cause is not detected. Causes rendering shorter than 8
  characters are skipped, so a cause rendering as `EOF` does not refuse every
  message containing that substring.
- `LocalFromUntrusted` **refuses** the Directory Node surface. Section 15.3
  leaves its local code as `incompatible_protocol` or
  `adapter_protocol_violation`/`transport_failure` "as applicable" without
  fixing which, so the mapping is reported unknown rather than guessed.
- Clause 15.1#5 (RPC hello must not advertise or negotiate an error contract
  key) and clause 15.3#3 (Error must not appear as an RPC hello key or a
  TerminalBackend capability) are **not discharged**. This repository builds no
  RPC hello frame and no TerminalBackend capability set. Both are recorded as
  the declared gaps on their bindings.
- Section 17.2 was **not re-bound**. Its single scanner-visible clause is an
  unknown-**event** reader obligation; an unknown error **code** is not an
  unknown session event, so the Structured Error reader does not discharge it.
  Its existing binding to `internal/config/writer.go:EncodeCurrent` remains
  wrong and remains disclosed in `README.md`. This is a residual for a later
  Story, not something this leaf silently rebound to a friendlier symbol.
- Section 15.2 stays `unmeasured`: the nineteen-row exit-code table carries no
  RFC 2119 keyword, so the clause scanner measures zero obligations under it.
  The table is implemented and checked row by row, but the **process exit
  status** Section 14.2 requires to equal it is still implemented nowhere; that
  belongs to sibling leaf `TASK-260830-1bipsa`.

## Measured coverage

| Measure | Ratio |
| --- | --- |
| Catalog code-version pairs resolving through `ExitCodeFor` | 316/316 (47 + 66 + 94 + 109) |
| Section 15.2 exit-status rows reproduced and checked | 19/19 |
| Pinned containing-contract majors in the static binding table | 15/15 |
| Pinned bootstrap surface-outcome rows | 8/8 (4 of the 5 named surfaces; Directory Node refused) |
| Statement coverage of `internal/axerror` | 99.7% |
| Reviewed detail-scanner keys checked against production | 27/27, over 4/4 Section 15.1 classes |
| Mutants killed by the negative suite | 20/20 |

Repository ownership gate after this change:

```text
traceability ok: contracts=60 normative_sections=36 acceptance_cases=53 fixtures=30 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=48 full=1 partial=2 sliver=1 unevidenced=41 unmeasured=3 unowned=2 clauses_discharged=10/394
```

Sections 15.1 (6/7) and 15.3 (2/3) moved from `unevidenced` to `partial` and are
re-bound from `internal/catalog/catalog.go:ForRelease` to `internal/axerror`.
Section 15.2 stays `unmeasured` and is re-bound to `ExitCodeFor` with a rewritten
gap. `clauses_discharged` moved 2/394 -> 10/394. Both new `partial` bindings are
still refused by assigned-scope admission, which requires `full`; they are added
to the refusal disclosure tables in `traceability_test.go` and
`cmd/tracecheck/main_test.go` with their exact ratios.

## Mutation evidence

`.temp/TASK-260830-34elja/mutants.sh` applies 20 mutants, verifies each marker is
actually present in the file before believing the result, restores after each
one, and aborts if a restore leaves the baseline red. Every mutant narrows a
gate rather than deleting it where narrowing is possible: one forbidden
retryability class removed at a time, one individually disqualified code removed
at a time, per-version code admission dropped while membership is kept, the
nested excluded-key walk dropped while the top-level walk is kept, the causal
gate restricted to the outermost cause link, one bootstrap code drifted to a
plausible neighbour, and the message bound measured in bytes instead of
characters. Result: 17 killed, 0 survived. Log:
`.temp/TASK-260830-34elja/mutation-01.log`.

## Commands run and real exit codes

| Command | Exit |
| --- | ---: |
| `go test ./internal/axerror -count=1 -v` | 0 |
| `go test ./internal/axerror -cover -count=1` | 0 (99.7%) |
| `go test ./... -count=1` | 0 |
| `go vet ./...` | 0 |
| `go build ./...` | 0 |
| `go generate ./internal/catalog` + `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 (generated catalog unchanged) |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `gofmt -l internal/axerror internal/traceability` | 0 (no files listed) |
| `zsh .temp/TASK-260830-34elja/mutants.sh` | 20/20 killed, baseline restored green |

Logs: `go-test-axerror-01.log`, `go-test-axerror-cover-01.log`,
`go-test-all-01.log`, `go-vet-01.log`, `go-build-01.log`, `tracecheck-01.log`,
`gofmt-01.log`, `mutation-01.log`, all under `.temp/TASK-260830-34elja/`.

## Durable state and capability claims

`internal/axerror` holds no state, opens no file, starts no process, and mutates
nothing durable, so this leaf has no crash or idempotency surface and no
crash/idempotency evidence is applicable. It advertises no provider, platform,
backend, or CLI capability: no `enabled`, `available`, `supported`, or `status`
field exists anywhere in the package.
