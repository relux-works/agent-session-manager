# TASK-260830-8x76g1 — developer rework revision 5

## Outcome

The revision-4 implementation was recovered after its Change Request failed on
stale base authority and a subsequent Story rebase removed the uncommitted
candidate. Recovery was bound to exact evidence:

- CR revision 3 was reconstructed at reviewed tree
  `93e6d212a49ab7320e061254d9507a567ccd3852`.
- The successful revision-4 `apply_patch` and formatting events were replayed
  in their original order from Codex session
  `01a0591b-1f95-7943-a69e-e6707d41e923`.
- The resulting implementation was applied over current Story HEAD
  `828d63321a74bb6be1b322eebc5ea64794d7d914`, which descends from current
  trunk `ad7275181ca82fc3fa29544e3893923a92d7b9d5`.

The composed production path remains:

`CalculateObjectIdentity|VerifyObjectIdentity -> prepareObjectIdentity -> validateImmutableObjectShape`

It validates the candidate-local closed shapes and recursive rules derived
from pinned SPEC.md Sections 1.6 and 10.1-10.4 before calculating or attesting
identity. Section 17.3 ownership remains limited to the immutable
`works.relux.ax.migrated-from` identity contribution.

## Revision-5 precision audit

An independent literal-type audit found three revision-4 over-refusals:

1. bare `string` elements in `excluded_classes` and
   `required_filter_names` had acquired an undeclared non-empty bound;
2. Section 17.3 `schema_id:string` had the same undeclared bound; and
3. absolute sanitized Git URLs were required to contain a path even though
   Section 5.6 requires a remote scheme/host and exclusions, not a path bound.

Production validation now distinguishes bare UTF-8 `string` from bounded
`string[n..m]`, and sanitized HTTPS/SSH/git URLs accept host-only absolute
forms while still refusing password/token, query, fragment, local/file, and
unsupported forms. New positive tests drive both public identity entries so
the gate proves precision in both directions rather than only refusal.

## Evidence ownership

The earlier board-owned revision-4 archive contains the pre-fix expected-red
run (exit 1, 21 malformed nested candidates attested). This run did not claim
to rerun that historical pre-fix state; it independently reconstructed the
candidate and directly ran every current green gate below. The new
over-refusal regressions are positive boundary tests and are reported as such,
not mislabelled as a negative refusal gate.

## Validation run directly in revision 5

Every command was a standalone foreground process. Exit codes below are the
actual command results.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Focused scalar/canonical production entries | 0 | `rev5-focused-production-entry.log` |
| New precision/over-refusal regressions | 0 | `rev5-overrefusal-regressions.log`, `rev5-precision-positive.log` |
| Cross-platform identity golden | 0 | `rev5-golden-identity.log` |
| `go test ./... -count=1` | 0 | `rev5-go-test-all.log` |
| `go test ./... -v -count=1` | 0 | `rev5-go-test-all-verbose.log` |
| `go test ./... -cover -count=1` | 0 | `rev5-go-test-all-cover.log` |
| `go test ./... -race -count=1` | 0 | `rev5-go-test-all-race.log` |
| Focused scalar/canonical race | 0 | `rev5-go-test-race.log` |
| Focused coverage | 0 | scalar 90.1%; canonicaljson 81.4% |
| `FuzzScalarProductionEntries`, fixed `100x`, `parallel=1` | 0 | 33 seed entries loaded |
| `FuzzCanonicalizeRoundTrip`, fixed `100x`, `parallel=1` | 0 | 19 seed entries loaded |
| `FuzzObjectIdentityRepresentationInvariant`, fixed `100x`, `parallel=1` | 0 | 16 seed entries loaded |
| `FuzzClosedIdentityShapeRefusal`, fixed `100x`, `parallel=1` | 0 | 49 seed entries loaded |
| Config-to-AST fuzz target wiring gate | 0 | `rev5-fuzz-config-gate.log` |
| Scoped tracecheck 1.6, 10.1-10.4, 17.3 | 0 | `assigned_scopes=6` |
| Catalog generated-output check | 0 | `rev5-catalog-check.log` |
| `go build ./...` / `go vet ./...` | 0 / 0 | build and vet logs |
| Linux/Windows canonicaljson test cross-compiles | 0 / 0 | cross-test logs |
| Linux/Windows full builds | 0 / 0 | cross-build logs |
| `gofmt -l internal` plus empty-output assertion | 0 / 0 | empty format log |
| `git diff --check` | 0 | `rev5-git-diff-check.log` |
| `task-board validate` | 0 | 262 inherited `MISSING_ACTIVITY` diagnostics; this task is absent |

## Scope boundary

This package is read-only and deterministic, so durable-state crash recovery
is not applicable. No `ax doctor` result, migration publication, atomic
reference advancement, rollback retention, command, or runtime capability is
claimed. Rules requiring referenced Blob Descriptor bytes, child manifests,
raw Git pack/index bytes, isolated object databases, or filesystem resolution
remain outside this identity-candidate boundary.

