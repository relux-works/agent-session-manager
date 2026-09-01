# TASK-260830-8x76g1 — CR revision 4 review verdict

## Verdict

**Changes requested.** Route `TASK-260830-8x76g1` to `to-dev` for another
producer/reviewer cycle. CR revision 4 MUST NOT be accepted.

Reviewed candidate: base `ad7275181ca82fc3fa29544e3893923a92d7b9d5`,
candidate tree `8a05b4881814aef152be060e364bbf34e913ad04`, patch SHA-256
`8f1c34826f8800842100c7af935eba45655a95fd5e992aec769d6926406a240d`.
All 59 candidate path contents matched the recorded candidate tree. The pinned
SPEC.md downloaded from commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`
matched the repository lock digest
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.

## Blocking finding F1 — Section 10.1 records bypass the composed closed-shape gate

Severity: blocking attestation-integrity defect.

`prepareObjectIdentity` calls `validateImmutableObjectShape` before both public
entries attest, but that validator has schema-specific cases only for Blob
Descriptor and Transfer Manifest. Every other registered immutable schema,
including `urn:ax:schema:session-record@1.0.0`, falls through to extension-only
validation.

That narrowing violates two requirements in the pinned source:

- Section 10.1 requires every record envelope to include its digest ID,
  `subject_id`, `created_by_host_id`, and `created_at` (plus optional
  namespaced extensions).
- Section 1.6 requires unknown top-level members in a major-version-1 object to
  be rejected, requires validation to use the negotiated schema/version and
  exact JSON path, and states that recomputing the self-ID does not make a
  malformed value valid.

The reviewer probe starts from the normative Session Record 1.0.0 shape and
drives both production entries. After recomputing a correct `record_id`, all of
the following malformed objects were accepted by both
`CalculateObjectIdentity` and `VerifyObjectIdentity`:

| Mutant | Calculate | Verify | Required result |
| --- | --- | --- | --- |
| impossible `created_at = 2023-02-29T12:00:00.000Z` | accepted | accepted | refuse |
| malformed `subject_id = not-a-uuid` | accepted | accepted | refuse |
| missing `created_by_host_id` | accepted | accepted | refuse |
| unknown top-level `unexpected_security_control` | accepted | accepted | refuse |

This is the standard negative-evidence shape **check present but narrowed away
from production**, and the green Section 10.1 tracecheck is also a **capability
claim that does not reproduce**. Existing tests prove Blob/Transfer Manifest
closure and extension grammar, but intentionally use minimal generic records;
they therefore leave this bypass green.

Production anchors:

- `internal/canonicaljson/canonical.go:174-199`
- `internal/canonicaljson/closed_shapes.go:30-54`

Evidence: `record-envelope-probe.log` and `record_envelope_probe.go` in the
review evidence archive.

## Required rework

1. Do not patch only the four Session Record examples. Systematically bind
   every identity schema/version exposed by the public entries to its complete
   negotiated closed-shape validator, or narrow/refuse the public identity
   surface for schemas whose full shape is not validated. An identity API must
   not attest a structurally invalid registered object.
2. At minimum, enforce every Section 10.1 common envelope member and its scalar
   grammar on all record-envelope schemas. Complete schema-specific member sets
   and nested shapes must also reject unknown or malformed data before identity
   calculation or verification, as Section 1.6 requires.
3. Add negative tests through both public entries from complete valid fixtures,
   then independently mutate a missing common member, malformed UUID,
   impossible timestamp, unknown top-level member, and unknown nested member.
   Add the record-envelope class to the closed-shape fuzz corpus and keep the
   target wired to the fixed configured budget.
4. Keep README and traceability ownership claims no broader than the production
   behavior that those two public entries reproduce.

## Validation summary

Reviewer-rerun commands all completed in the foreground:

| Gate | Result |
| --- | --- |
| normative source SHA-256 and CR patch SHA-256 | pass |
| all 59 candidate path contents vs candidate tree | pass |
| focused scalar/canonical tests | pass |
| `go test ./... -count=1` | pass |
| focused scalar/canonical race tests | pass |
| four configured fuzz targets, each `-fuzztime=100x -parallel=1` | pass |
| scoped tracecheck for 1.6, 10.1–10.4, 17.3 | pass (`assigned_scopes=6`) |
| catalog generated-output check | pass |
| `go vet ./...` | pass |
| `go build ./...` | pass |
| adversarial Section 10.1 public-entry probe | **fail: four malformed shapes attested** |

Producer-attached coverage and Linux/Windows cross-build evidence was inspected
but not independently rerun; the reviewer reran the gates listed above.

## Scope and workspace notes

- The provided accepted-leaves and audit-carryforward archives matched their
  declared SHA-256 digests. Every file included in the audit carryforward
  matches the candidate byte-for-byte; candidate-only `go.mod`/`go.sum` carry
  the new JCS dependency.
- A non-candidate untracked `.DS_Store` is present in the managed worktree. It
  is not part of the recorded candidate tree and was not modified or deleted by
  this read-only review. The next producer/orchestrator should account for it
  before publishing the successor candidate.

