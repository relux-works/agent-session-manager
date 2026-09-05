# TASK-260830-1zvmw7 — Round-1 rework evidence (rev2)

Base: leaf-1 checkpoint `9481a20` + uncommitted round-1 CR candidate.
Working tree only — nothing committed, per the Change Request shape.

## Production changes (`internal/terminalbackend/manifest.go`)

- F1: `CheckOperation` admits `manifest`/`probe` for any admitted set
  (empty included) — §4.D gives both `Capability dependencies: none`.
  Doc comment reconciled with `CapabilitiesForOperation`.
- F8: removed the dead `checkProbeIdentity` inner branch (backend IDs are
  already equal where it stood — substitution is always untrusted now),
  removed the `parsePlatformList` duplicate check subsumed by `>=`,
  corrected the `checkEvidenceSet` doc (whole-reconciliation refusal).

## Test changes

- Structural root: every refusal assertion in `manifest_test.go` and
  `manifest_pin_test.go` now names `*Error.Detail` via
  `requireRefusal`/`requirePinRefusal` (was 0/29).
- F2: narrowed 4 fixtures until only the named rule rejects —
  omission/static-echo withdraw the orphaned evidence, protocol moves the
  evidence with the probe, false-claim lists the new evidence ID.
  `dangling evidence id` flips the withdrawn claim false so it reaches
  the ID-set arm (a bare deletion lands on coverage first).
- F3: substitution fixture tracks the foreign digest in the probe, so only
  the manifest↔record gate can refuse.
- F4: `evidence expires at admission instant` row (`expires_at == now`).
- F5: `ParseProbe`/`ParseEvidence` gain schema/schema_version/
  backend-identity/kind/digest cases; all three parsers gain an
  `unknown implementation kind` case.
- F6: adjacent-duplicate cases (sorted position, holds the `>=` halves),
  `unprefixed valid base64 signature`, `null realm members on realm claim`.
- F7: `claim list bound` (17), `evidence list bound` (257),
  `empty platforms`, `overlong os version` (257).
- G-A: `TestParsedClaimSlicesShareNoRegistryBacking` (white-box) poisons
  every parsed claim slice and re-reads the interior.
- F1: `TestCheckOperationAdmitsDependencyFreeOperations`.
- F9: README `15.3#3` re-scoped, `terminal-manifest-probe-admission`
  acceptance case registered, projection digest re-pinned, README and
  golden figures 74→75 acceptance cases.

## Verification (all run directly, exit codes observed)

- Targeted fix mutants: 24/24 KILLED (`/tmp/rev/verify-fixes.py`,
  exit 0). Every mutant grep-confirmed present (anchor count == 1),
  `go vet` clean, reverted (tree equals pre-run bytes).
- Deletion-mechanism probe: all 6 narrowed fixtures fail with
  `error = <nil>` under gate deletion — the gate admits what it must
  reject; no lower arm catches the fixture.
- Adjacent-vs-descending control: adjacent cases KILLED by M41/M60;
  the old descending cases pass under the narrowed gate (caught by `>`),
  proving the new cases carry the dup halves.
- Reviewer battery-1 re-run: 46 KILLED / 0 SURVIVED. 12 COMPILE-FAIL are
  battery-1 formulation artifacts (deleted code leaves an unused binding);
  battery-2's compilable variants of all 12 are KILLED. M37/M58 anchors
  changed by the F1/F8 edits — re-anchored equivalents both KILLED.
- Reviewer battery-2 re-run: 43 KILLED / 4 SURVIVED, exactly the expected
  remainder: M71/M72 (JCS unprovable — closed schemas use ASCII-only keys
  with no numbers, so Go marshalling and RFC 8785 agree byte for byte;
  reported as bound, not coverage) and M92/M95 (near-equivalents).
- G-A positive control (parseClaim returns row slices): the new pin
  FAILS while the full shipped suite previously passed with it live.
- Full suite: `go test ./... -count=1` exit 0, 14/14 packages ok
  (`/tmp/rev/full-suite.log`). Package coverage 88.6% (was 87.1%).
  `go vet ./...` exit 0. `gofmt -l .` empty.
- Traceability: `internal/traceability` + `cmd/tracecheck` green;
  projection digest re-pinned to
  `892bdb6a…bf806a035`; section coverage unchanged
  (49 bindings, 17/403 clauses).

## Stated bounds inherited

JCS canonicalization unprovable with these fixtures (M71/M72 survive as a
property of the fixtures, not a defect). M92/M95 near-equivalent. No new
bounds introduced.
