# TASK-260830-2xdt8t rework verification (F1–F5 close-out)

Rework of the terminal-backend registry after reviewer verdict CHANGES REQUESTED
(`TASK-260830-2xdt8t_review-verdict.md`, RUN-260904-cd73df). Both adjudicated
decisions (D1 empty `required_capabilities` default, D2 no conpty rule for
third-party IDs) were upheld and needed no code change; D1's LOGBOOK reasoning
is restated below. Worktree: `task-board/story/STORY-260830-3m2mw8` on top of
reviewed commit `b5cf5b5`.

## Production changes

`internal/terminalbackend/terminalbackend.go` only (no new refusal sites):

- `cloneStrings` / `clonePlatforms` / `cloneRecord` helpers.
- `New`: each built-in gets its own protocol-slice copy (previously one sorted
  slice shared across both built-ins).
- `RegisterExternal`: stores `cloneRecord(observed)` (previously verbatim).
- `Resolve`: returns `cloneRecord(record)` (previously shared backing arrays).

## Validation commands run (each a standalone process, real exit codes)

| Command | Exit |
| --- | --- |
| `gofmt -l internal/terminalbackend/ internal/config/ internal/traceability/` (no output) | 0 |
| `go vet ./...` (no output) | 0 |
| `go test ./... -count=1` (14 packages, incl. tracecheck) | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go run ./internal/traceability/cmd/tracecheck -section 6.2` | 0 |

Tracecheck after registering the negative arm: `acceptance_cases=74`,
`clauses_discharged=17/403` — no figure moved (a test joined an existing
acceptance case; no clause was added or removed).

## Targeted mutation re-run (harness: `/tmp/mutcheck.py`)

One mutant at a time, presence-checked (`count == 1`), sources restored from
memory backup between mutants and restoration asserted at the end.

| Mutant | Result |
| --- | --- |
| F1 `Resolve` returns stored record (no egress copy) | KILLED, exit 1 |
| M12 `equalRecord` ignores `ProtocolVersions` members | KILLED, exit 1 |
| M13 `equalRecord` ignores `Platforms` members | KILLED, exit 1 |
| M16 generation bound 256→257 | KILLED, exit 1 |
| M30 generation lower bound deleted | KILLED, exit 1 |
| M08 protocol count bound 32→33 | KILLED, exit 1 |
| M20 platforms sorted-unique `>=`→`>` | KILLED, exit 1 |
| M21 platforms non-empty bound removed | KILLED, exit 1 |
| M22 platforms upper bound 4→5 | KILLED, exit 1 (refusal clause differs) |
| M23 kind vocabulary admits any string | KILLED, exit 1 |
| M24 `DigestFile` regular-file guard deleted | KILLED, exit 1 (build error: `info` unused) |
| M28-DELETION trust-loop `ParseID` call removed | KILLED, exit 1 |
| digest-must-be-null branch removed | KILLED, exit 1 |

13 applied, 13 killed, 0 survivors.

M24 compiling-variant control: `if !info.Mode().IsRegular() && false`
(mutant confirmed present by grep) makes `TestDigestFile` go red via the
documented FIFO block (go-test 20s timeout panic), proving the FIFO subtest —
not the EISDIR-confounded directory subtest — carries the pin. Sources
restored; package green afterwards.

## Declared equivalent mutants (not covered, with reason)

- M28-revert (trust-loop `ParseID` back to the previous local pattern):
  behavior-identical. Refusal sets coincide — the old grammar refuses the same
  malformed inputs, the error clause is intentionally non-distinguishing, and
  the `ax.` difference is subsumed by the `HasPrefix` line two statements
  later. The call is pinned against deletion instead (`INVALID_ID` wiring case).
- `parseKind` vocabulary arm and digest-must-be-null arm: unreachable through
  current production entry points (external-kind gate precedes validation and
  reports `Untrusted`, pinned by
  `TestRegisterExternalRefusesUnknownKindAsUntrusted`; `New` builds only null
  digests). Measured white-box in `internal_pin_test.go`; reachability stated
  as a bound with the contract reason (reordering would mislabel
  attacker-influenced input), not as effort.

## Finding-by-finding close-out

- F1: fixed + `TestRegistryCopiesSlicesAcrossItsBoundary` (4 subtests, incl.
  the end-to-end drift-reported-as-duplicate bypass). F1 mutant killed.
- F2: same-length drift cases added; M12/M13 killed.
- F3: generation-bound test rewritten with equal generations (1 and 256
  admitted, 0 and 257 refused); mismatch arm split out. M16/M29-shape and M30
  killed (M29 256→100000 subsumed by the M16 256→257 pin — same comparison).
- F4: all domains narrowed (see table); ratio for the re-run batch 13/13
  killed. Remaining non-covered variants are the declared equivalents above.
- F5: `TestDecodeRefusesNonConptyBackendOnNativeWindows` added to the
  `config-versioned-readers` tests in `ownership.v0.5.0.json`; projection
  digest re-pinned in `traceability.go` (`e3cc40da…9438d`); README discharge
  prose names both arms and no longer claims a positive-only discharge.
  Digest coordination note: a parallel story shares this registry file; if its
  digest moves first, this pin must be recomputed in merge order.
- D1 correction: LOGBOOK entry 2147 restated as a stated bound
  (underspecified clause, empty the only non-inventing default, weaker gate
  disclosed); the SPEC.md:2604-2605 justification is withdrawn.

## Crash/idempotency evidence

Registry is process-local memory: no durable state, so no crash-recovery
evidence applies. Idempotency evidence is the duplicate/drift non-clobber
assertions (`TestDuplicateRegistrationIsRefused`, `TestDriftIsRefused`, and the
new boundary test asserting the admitted record survives every refused attempt).

## Files changed vs `b5cf5b5`

- `internal/terminalbackend/terminalbackend.go` (+48/-6): slice clones.
- `internal/terminalbackend/terminalbackend_test.go` (+300/-): boundary,
  same-length drift, rewritten generation bounds, platform violations,
  kind-order, 32/33 protocol bound, FIFO pins.
- `internal/terminalbackend/internal_pin_test.go` (new): white-box kind
  vocabulary, digest-null arm, platform bound precedence.
- `internal/config/terminal_registry_wiring_test.go` (+21): malformed trust ID.
- `internal/traceability/ownership.v0.5.0.json` (+4): negative arm registered.
- `internal/traceability/traceability.go` (1 line): digest re-pin.
- `README.md`, `LOGBOOK.md`: discharge prose, D1 restatement, rework entry.
