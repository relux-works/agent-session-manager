# TASK-260830-2iint0 implementation evidence

## Outcome

Implemented the read-only AX configuration path/loading boundary at
internal/config. The production Load entry point resolves one process-lifetime
snapshot from command overrides, the exact five AX_* overrides, and platform
defaults; it then reads exactly one selected regular configuration file.

The path registry covers --config/AX_CONFIG, --data-dir/AX_DATA_DIR,
--state-dir/AX_STATE_DIR, --cache-dir/AX_CACHE_DIR, and
--runtime-dir/AX_RUNTIME_DIR. Empty values are unset. Unknown or credential-like
environment variables are never interpreted or returned. macOS, Linux, WSL2,
and native Windows defaults and native lexical path handling are covered.

Absence remains distinct from stat/read failure. Existing roots must be
directories; absent roots remain available to the later owner-only
initialization boundary. Results and document bytes are isolated from caller
mutation. No configuration/root state is written.

## Normative source and scope

- Source: relux-works/agent-session-manager-spec v0.5.0
- Commit: 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c
- SPEC.md SHA-256 verified locally:
  562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
- Scoped implementation bindings: Section 3.2 and Section 6.1, AC-PATH-001
- Configuration 1/2/3 TOML schema validation, Section 6.2-6.5 field bounds,
  migration, downgrade, doctor, and capability publication are deliberately
  not claimed by this task.
- This task declares no numeric or Unicode character bounds. The universal
  accept-at-limit/refuse-past-limit checklist clause is therefore not
  applicable here; the versioned schema task owns those bounds.
- Load and ResolvePaths perform no durable mutation. Crash rollback and durable
  idempotency evidence are not applicable; repeated read-only loads are proven
  idempotent and isolated.

## Files

- internal/config/loader.go
- internal/config/loader_test.go
- internal/traceability/ownership.v0.5.0.json
- internal/traceability/traceability.go
- internal/traceability/traceability_test.go
- internal/traceability/cmd/tracecheck/main_test.go
- README.md

## Validation

Every listed green gate ran as a standalone process and returned exit code 0.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| go test ./internal/config -count=1 -v | 0 | go-test-config-01.log |
| go test ./internal/config -cover -count=1 | 0 | go-cover-config-01.log; 80.6% in full coverage run |
| scoped tracecheck for 3.2 and 6.1 | 0 | tracecheck-config-01.log |
| go test ./... -count=1 -v | 0 | go-test-all-01.log |
| go test ./... -race -count=1 | 0 | go-race-all-02.log |
| go test ./... -cover -count=1 | 0 | go-cover-all-01.log |
| go vet ./... | 0 | go-vet-01.log |
| go build ./... | 0 | go-build-01.log |
| full tracecheck | 0 | tracecheck-all-01.log |
| generated catalog check | 0 | catalog-check-01.log |
| GOOS=linux GOARCH=amd64 go build ./... | 0 | go-build-linux-amd64-01.log |
| GOOS=windows GOARCH=amd64 go build ./... | 0 | go-build-windows-amd64-01.log |
| JSON validation with pipefail | 0 | json-check-01.log |
| gofmt check | 0 | direct command evidence |
| task-board validate | 0 | task-board-validate-01.log |
| git diff --check | 0 | git-diff-check-01.log |

The first test-first baseline correctly failed with exit code 1 because the
production API did not yet exist. A traceability iteration also correctly
failed with exit code 1 on the reviewed semantic-digest guard, reporting the
new exact digest before that digest was updated.

## Negative evidence

Fourteen production clauses were disabled one at a time. Each exact
go test ./internal/config -count=1 run returned exit code 1, so no mutant
survived. The attacked clauses cover flag precedence, empty environment
handling, unknown override refusal, invalid platform refusal, required Linux
and Windows platform inputs, config absence/read/kind distinctions, root
inspection/kind distinctions, and exclusion of secret-like environment names.
After the campaign, loader.go matched the pre-mutation copy byte-for-byte
(cmp exit code 0). See mutation-results.md.
