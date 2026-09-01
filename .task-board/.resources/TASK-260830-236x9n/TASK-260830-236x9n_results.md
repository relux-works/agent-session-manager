# TASK-260830-236x9n — developer handoff

## Outcome

Implemented `internal/canonicaljson` as the production RFC 8785 and AX
omit-self identity boundary.

- `Canonicalize` strictly validates UTF-8, UTF-16 surrogate escapes, duplicate
  object names, complete JSON syntax, and the I-JSON number surface before RFC
  8785 transformation.
- `CalculateObjectIdentity` rejects floating-point numbers and unsafe integers,
  discovers exactly one top-level self field from the closed 18-field Section
  1.6 registry, omits it, canonicalizes the remaining object, and returns its
  SHA-256 digest.
- `VerifyObjectIdentity` refuses malformed or mismatched claims, including a
  digest calculated over a full object that still contains its self field.
- `chunk_id` is deliberately excluded because Section 10.3 defines it over raw
  chunk bytes.
- RFC 8785 Section 3 string/primitive/property-order vectors, every finite
  Appendix B number sample, AX safe-integer and UTF-16-order vectors, and the
  normative Blob Descriptor are exercised through production entry points.
- README and reviewed traceability ownership now describe and bind the feature
  without advertising an `ax`, doctor, migration-transaction, or capability
  surface.

The package is read-only. It makes no durable state mutation, so crash recovery
is not applicable. Repeated calculation/canonicalization is covered as
deterministic and idempotent.

## Candidate scope

The temporary alternate-index snapshot reports exactly nine paths relative to
Story HEAD:

- `README.md`
- `go.mod`
- `go.sum`
- `internal/canonicaljson/canonical.go`
- `internal/canonicaljson/canonical_test.go`
- `internal/traceability/cmd/tracecheck/main_test.go`
- `internal/traceability/ownership.v0.5.0.json`
- `internal/traceability/traceability.go`
- `internal/traceability/traceability_test.go`

The inherited `internal/scalar` index state is not part of this candidate;
alternate-index comparison confirms those files equal Story HEAD.

## Final green validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/canonicaljson -count=1 -v` | 0 | `go-test-canonical-02.log` |
| `go test ./internal/canonicaljson -cover -count=1` | 0 | `go-cover-canonical-02.log`; 82.6% |
| `go test ./internal/scalar -count=1` | 0 | Direct focused run; prior-task ownership stayed green |
| `go test ./internal/traceability -count=1 -v` | 0 | Direct focused run |
| `go test ./internal/traceability/cmd/tracecheck -count=1 -v` | 0 | Direct focused run; includes renamed production-entry refusal |
| `go run ./internal/traceability/cmd/tracecheck -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4` | 0 | `tracecheck-sections-02.log`; 5 assigned scopes |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck-default-02.log` |
| `go generate ./internal/catalog` | 0 | `go-generate-01.log`; generated file unchanged |
| `go test ./... -v -count=1` | 0 | `go-test-all-02.log` |
| `go test ./... -cover -count=1` | 0 | `go-cover-all-02.log` |
| `go test -race ./internal/canonicaljson ./internal/scalar -count=1` | 0 | `go-race-focused-02.log` |
| `go vet ./...` | 0 | `go-vet-02.log` |
| `go build ./...` | 0 | `go-build-02.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `go-build-linux-amd64-01.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `go-build-windows-amd64-01.log` |
| `go mod verify` | 0 | `go-mod-verify-01.log` |
| `task-board validate` | 0 | `task-board-validate-01.log` |
| `gofmt -d internal/canonicaljson internal/traceability` plus empty-output assertion | 0 / 0 | `gofmt-diff-01.log` |
| Alternate-index `git diff --cached --check HEAD` | 0 | Candidate diagnostic output |

## Expected-red and exploratory non-zero commands

These are failures by design and are not reported as passing gates.

| Command / mutation | Exit | Expected reason |
| --- | ---: | --- |
| `go run ./internal/traceability/cmd/tracecheck -section 10.999` | 1 | Unknown v0.5.0 section refused; `tracecheck-10.999-expected-red-01.log` |
| Narrow production self-field registry from 18 to 17, then run `TestCalculateObjectIdentityOmitsEveryNormativeSelfField -count=1` | 1 | Test reported `17 entries, want exact normative 18`; source was restored and the same named test then exited 0 |
| First traceability run after reviewed registry edit | 1 | Controlled projection re-pin required; reported new digest `72a763e...`, then passed after explicit reviewed pin update |
| First renamed-entry mutant test iteration | 1 | Assertion expected the wrong diagnostic substring; corrected to the actual production-owner refusal and rerun green |
| `go doc encoding/json/jsontext.Canonicalize` | 1 | API is behind `GOEXPERIMENT=jsonv2`; production does not use the experimental surface |

## Sources and decisions

- AX v0.5.0 source commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- RFC 8785: <https://www.rfc-editor.org/rfc/rfc8785.html>.
- RFC Appendix G lists the Go implementation lineage used here; the pinned
  module is `github.com/gowebpki/jcs v1.0.1`. Repository-owned validation wraps
  it to close duplicate-name, malformed UTF-8, and surrogate-pair gaps before
  transformation.
- Detailed research and readiness records are in `research-jcs.md` and
  `tool-readiness-01.log`.
