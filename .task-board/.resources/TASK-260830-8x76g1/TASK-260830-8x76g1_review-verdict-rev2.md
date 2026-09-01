# TASK-260830-8x76g1 review verdict — CR revision 2

## Verdict

**Changes requested.** Route the implementation to `to-dev`; do not accept
`CR-TASK-260830-8x76g1-2` revision 2.

Reviewed immutable candidate tree:
`2885010caafe4fc54651ea11232398825deada93`, base
`c9e5290b1506275f5417b26070fad0391a09c50a`. The attached patch SHA-256
reproduced the handed digest
`79726020ace08bcf193b0ef5fb142c1b3c9369db78222e3b4bcf0968e70388c3`.
Tests were executed from an archive of that exact candidate tree because the
managed checkout's index/worktree representation was not itself an unambiguous
candidate snapshot.

## Blocking findings

### F1 — invalid Transfer Manifest extension keys are attested

Severity: blocking, attestation-integrity bypass.

The normative Section 1.6 rule permits unknown fields only under an
`extensions` map whose keys are reverse-DNS names, and Section 10.4 repeats
that constraint for Transfer Manifest. The production gate in
`internal/canonicaljson/closed_shapes.go` lines 209-215 passes the whole map to
`validateMigrationExtensionObject`, but lines 475-498 inspect only the optional
`works.relux.ax.migrated-from` value and never validate any other key.

The reviewer drove both composed production entries with an otherwise valid
Transfer Manifest containing `"extensions":{"not_namespaced":true}`. Both
`CalculateObjectIdentity` and `VerifyObjectIdentity` returned nil errors, and
verification accepted the correctly computed omit-self claim. This is the
standard **bypass path around the check** shape: the closed top-level gate runs,
but an invalid nested extension namespace still reaches attestation.

Required rework:

- validate every Transfer Manifest extension key against the normative
  reverse-DNS namespace grammar before identity calculation or verification;
- add negative tests that drive both production entries with a correct
  omit-self claim; and
- add this reviewer-found shape to the committed refusal fuzz corpus.

Evidence: `production-attack-01.log`.

### F2 — a valid character-bounded symlink target is refused by a byte bound

Severity: blocking interoperability over-refusal.

Section 1.6 says `string[n..m]` bounds UTF-8 **characters**. Section 10.4 types
`ManifestEntry.target` as `string[1..4096]`. The production validator instead
uses `len(target) > 4096` at `internal/canonicaljson/closed_shapes.go` lines
294-299, which counts UTF-8 bytes.

The reviewer supplied a platform-neutral segmented target containing 3,999
characters and 5,999 UTF-8 bytes. `CalculateObjectIdentity` refused it as
exceeding 4,096 bytes even though it is inside the normative character bound.

Required rework: count Unicode characters consistently with the common scalar
contract and add a multibyte boundary test through the production identity
entry.

Evidence: `production-attack-01.log`.

### F3 — the configured validation suite omits the closed-shape fuzz target

Severity: blocking acceptance-gate gap.

`task-board.config.json` lines 119-121 configure fixed `100x` runs for scalar,
canonical round-trip, and representation-invariance fuzzing, then proceed to
tracecheck. The new `FuzzClosedIdentityShapeRefusal` target is absent even
though README documents it and the task Definition of Done explicitly requires
the refusal fuzz corpus to run in the bounded deterministic validation suite.
The candidate-bound Change Request validation log likewise contains no active
closed-shape fuzz command.

The reviewer ran the omitted target directly with
`-fuzztime=100x -parallel=1`; it passed, but a one-off reviewer run is not the
configured gate required to protect later candidates.

Required rework: add the exact fixed-count `FuzzClosedIdentityShapeRefusal`
command to `spawn.worktree_isolation.validation.commands` and ensure the next
candidate-bound validation transcript executes it.

Evidence: `configured-fuzz-gate-13.log`, `cr-rev2-validation.log`, and
`fuzz-closed-shape-08.log`.

## What passed

The revision does fix CR revision 1's original bypass: `prepareObjectIdentity`
invokes `validateImmutableObjectShape` before both calculation and verification,
and focused negative tests refuse unknown Blob/Manifest members and chunk
order/bounds/coverage violations with correctly computed claims.

Reviewer-rerun commands on the exact candidate all exited 0:

- focused canonical and scalar tests;
- all four native fuzz targets at fixed `100x` and `-parallel=1`;
- scoped tracecheck for Sections 1.6, 10.1-10.4, and 17.3;
- `go test ./... -count=1`;
- `go vet ./...` and `go build ./...`; and
- focused coverage: canonicaljson 80.7%, scalar 90.5%.

The attached producer evidence reports passing race, catalog-generation,
Linux/Windows cross-build, diff, and board-validation gates. README correctly
keeps migration publication, atomic reference advancement, `ax doctor`, and
runtime capability claims unavailable. Those positives do not override the
three blocking findings above.

