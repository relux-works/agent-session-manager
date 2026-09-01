# TASK-260830-1pbx0c review verdict — CR revision 6

## Verdict

Accepted. CR `CR-TASK-260830-1pbx0c-6` revision 6 satisfies the scoped common-scalar deliverable and the revision-5 Windows-path rework. No review finding remains.

The exact reviewed delta is base commit `c9e5290b1506275f5417b26070fad0391a09c50a` to candidate tree `f223be70d0075bea00cd11ab22a51b49ad35339e`. Reconstructing the candidate through a temporary alternate index produced that exact tree OID. The board patch SHA-256 is `0551b6925f12a4c5d6206e50f704cee0ae42b9baeabeb11b5ebabb7256d78af7`, matching the Change Request record.

## Contract and architecture fit

- The pinned `SPEC.md` was read from commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`; its SHA-256 is `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`, matching `internal/specpin/v0.5.0.lock.json`.
- `internal/scalar` implements canonical UUIDv4/UUIDv7, real UTC RFC3339 timestamps, SHA-256 digests, platform/provider values, relative and platform-bound absolute paths, safe/bounded integers, `decimal_uint64`, and closed enums behind validated constructors and decode boundaries.
- The package is read-only value validation. Section 17.3 durable migration/crash/idempotency behavior is not applicable, and the delta adds no `ax doctor` result or runtime capability claim.
- Five carry-forward files (`integer.go`, `names.go`, `scalar.go`, `time_digest.go`, and `uuid.go`) remain byte-identical to the supplied archive. Only `path.go` and `scalar_test.go` have the expected revision-6 Windows rework within `internal/scalar`.
- README and the project tool/output documentation accurately describe the delivered validation surface and explicitly disclaim unsupported runtime capability.

## Gate attacks

- A reviewer production probe exercised all 22 required DOS device names in bare, lowercase-with-extension, and multi-extension forms, on both drive-qualified and UNC paths. `ParseAbsolutePath` and `DecodeAbsolutePathJSON` refused them all.
- The same probe refused each required Win32 punctuation class, accepted non-reserved lookalikes (`CONSOLE`, `xCON`, `COM0`, `COM10`, `COM1x`, `LPT0`, `LPT10`, `file.CON`, `.CON`), and preserved POSIX acceptance of the same punctuation/name components.
- The probe accepted the published `1990-12-31T23:59:60` leap second through parse, JSON, and text boundaries, including lowercase `t/z` and `+00:00`, while refusing ordinary-minute and unpublished-date `:60` values.
- An independently created narrowed mutant changed DOS matching from “reserved base with any extension” to bare-name-only. The named production-boundary test failed at parse, JSON decode, and forged marshal for `con.txt`, `PRN.json`, `NUL.txt`, `com9.any`, `lpt9.any`, and UNC extension forms. Mutant exit was 1 as expected; the exact candidate was not modified.
- The five ownership declaration rename mutants for Sections 1.6 and 10.1–10.4 all passed as expected-red checks, proving the assigned-scope gate fails when its real owner declaration is renamed.

## Independent validation

| Check | Result |
| --- | ---: |
| Exact candidate snapshot reconstruction and patch digest | PASS |
| Reviewer boundary probe | PASS |
| Narrowed bare-name-only mutant | expected FAIL (exit 1) |
| `go test ./internal/scalar -count=1 -v` | PASS |
| `go test -race ./internal/scalar -count=1` | PASS |
| `go test ./internal/scalar -cover -count=1` | PASS, 89.6% |
| `go test ./... -count=1` | PASS |
| Five scoped-owner rename mutants | PASS |
| Assigned tracecheck for 1.6 and 10.1–10.4 | PASS, `assigned_scopes=5` |
| Global tracecheck | PASS |
| `go vet ./...` and `go build ./...` | PASS |
| Linux/amd64 and Windows/amd64 cross-builds | PASS |
| `gofmt -l` on changed Go files and exact-delta `git diff --check` | PASS |

Raw reviewer logs and the probe source are attached as `TASK-260830-1pbx0c_reviewer-validation-rev6.tar.gz`.
