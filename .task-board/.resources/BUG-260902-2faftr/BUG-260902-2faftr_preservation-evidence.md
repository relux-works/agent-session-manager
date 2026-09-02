# BUG-260902-2faftr — revision 2: pinning the nested-key preservation claim

Scope of this revision: the single non-blocking finding the accepted review
(RUN-260902-cdeacc) recorded. Nothing else was changed. The rest of the
revision remains the reapplied, independently verified BUG-260902-3ru6vw work.

## The finding, reproduced

README and the commit message both claimed that a nested extension key named
`endpoint` or `token` is "preserved as data". The tests asserted only that such
a document is **admitted**. Admission is evidence about a gate, not about what
survives it.

Reproduced on a scratch `git archive HEAD` copy of the previous revision. Mutant
applied to the `map[string]any` arm of `validateExtensionValue`
(`internal/config/validation.go`), which receives the live decoded map:

```go
for k := range typed {
    if oneOf(k, "secret", "token", "password", "credential",
        "auth", "environment", "env", "endpoint") {
        delete(typed, k)
    }
}
```

| Tree | Command | Exit | Meaning |
| --- | --- | ---: | --- |
| Previous revision (`git archive HEAD`) + mutant | `go test ./... -count=1` | **0** | Silent nested-key deletion passed the entire suite. Claim unpinned — finding confirmed. |
| This revision + mutant | `go test ./... -count=1` | **1** | `internal/config` fails. Claim pinned. |
| This revision + mutant | `go test ./internal/config -run TestExtensionValueObjectKeysArePreservedAsData -v` | **1** | All **11** preservation subtests redden. |

Representative failure line:

```
extensions { "works.relux.fixture" = { auth = "value-of-auth" } } round-tripped as
{"works.relux.fixture":{}}, want byte-identical {"works.relux.fixture":{"auth":"value-of-auth"}}
```

## The closure

`TestExtensionValueObjectKeysArePreservedAsData` in
`internal/config/extension_key_admission_test.go`. It drives the same production
entry the admission tests use — `loadConfigDocument`, reached from the exported
`Load` (`schema_test.go:622`) — and asserts byte identity of the re-encoded
extensions map rather than merely a nil error. `encoding/json` sorts object
keys, so the comparison is stable and a dropped, added, renamed, retyped or
rewritten nested key all fail.

Cases (11 subtests):

- one per previously blacklisted label — `secret`, `token`, `password`,
  `credential`, `auth`, `env`, `environment`, `endpoint` — each with a value
  distinct per label, so a mutant dropping one key cannot be masked by another
- a blacklisted label at every admitted depth (`endpoint` / `token` / `password`
  / `credential` nested through depth 4)
- non-string nested values: integer, boolean, array and nested object under
  blacklisted labels, so a retyping mutant fails too
- a root key that itself carries a blacklisted label (`works.relux.env-tools`)
  with blacklisted nested keys, so this one case fails if **either** arm of the
  removed rule returns

Preservation turned out to be genuinely guaranteed, so README and the commit
message were left stating it and are now backed; no claim needed correcting.

## Mutants (all run by me, real exit codes)

| Mutant | Exit | Failing subtests |
| --- | ---: | ---: |
| Delete nested blacklisted keys — previous revision | 0 | 0 (the finding) |
| Delete nested blacklisted keys — this revision | 1 | 11 |
| Re-added key blacklist (AC-required narrowing) | 1 | 18 |
| Re-added nested-key blacklist (AC-required narrowing) | 1 | 21 |

The two AC-required narrowing mutants redden **more** than on the previous
revision (18 vs 17, 21 vs 10): the preservation subtests add pinning, they
remove none. The tree was restored green after every mutant; all mutants ran on
scratch copies under `.temp/`, never on the worktree.

## Gates on the exact committed tree (`29d8cee`), real exit codes

| Gate | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l ./internal` | 0 (no output) |
| `go test ./... -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `tracecheck -section 6.1 6.2 6.3 6.4 6.5 17.1 17.2 17.4` | 0 |
| `cataloggen ... -check` | 0 |

Coverage, `internal/config`: **93.9% -> 94.4%**. Baseline measured from an
independent `git archive HEAD` copy of the previous revision. No regression; the
preservation tests raise it.

Not run: the `scalar` / `canonicaljson` fuzz targets — outside this delta and
unchanged by it.

## Commit

One signed commit, `29d8cee`, exactly one past checkpoint `67aed0b` on
`task-board/story/STORY-260902-2oiugz`. `git verify-commit HEAD` exits 0 (good
signature, `oparin@me.com`). The previous revision's commit `1a94718` was
amended, not followed by a second commit, per the one-commit-per-leaf contract.
Working tree clean.
