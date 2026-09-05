# TASK-260830-2890sd review: mutation and probe log

Reviewer run RUN-260904-4f8f04. Target commit `292fc6d` on
`task-board/story/STORY-260830-3jqsx1` (Change Request
`CR-TASK-260830-2890sd-1` rev 1, `repository_delta=empty`; the reviewable
delta is the commit, 15 files, +2492).

Method: each mutant was applied to a restored copy of the production source,
confirmed PRESENT (pattern occurrence asserted before edit, file compared
after), then `go test ./internal/provider/ -count=1` was run and the source
restored from a backup copy (never `git checkout`). Worktree verified clean
after every batch.

Baseline: `go test ./internal/provider/ -count=1` ok, 93.4% statements.
`go test ./... -count=1` ok (14 packages). `go vet ./...` clean.
`go run ./internal/traceability/cmd/tracecheck` exit 0,
`clauses_discharged=17/403`, `bindings=49` — unchanged, as the producer reported.

## Mutants CAUGHT (27)

| Mutant | Kind | Caught by |
| --- | --- | --- |
| duplicate refusal deleted | delete | TestDiscoverRefusesDuplicates + 4 |
| builtins no longer registered in `seen` | narrow | TestDiscoverRefusesDuplicates + 2 |
| duplicate refused only within one source label | narrow | TestDiscoverRefusesDuplicates + 3 |
| `sort.Strings(names)` reversed | narrow | TestDiscoverSortsEntriesWithinADirectory |
| `sort.Strings(names)` removed | delete | compile error (unused import) |
| builtins enumerated before plugin_dirs | reorder | 6 tests |
| PATH gate `if cfg.AllowPathPlugins` → `if true` | delete | TestDiscoverSkipsPATHWhenDisallowed + 2 |
| plugin_dirs absolute-path gate deleted | delete | TestDiscoverRefusesRelativePluginDir |
| discovery regular-file gate deleted | delete | TestDiscoverRefusesNonRegularTargets |
| discovery regular-file gate narrowed to NUL-containing paths | narrow | TestDiscoverRefusesNonRegularTargets |
| discovery owner gate deleted | delete | TestDiscoverRefusesUnapprovedOwners |
| discovery owner gate narrowed to `uid == 0` | narrow | TestDiscoverRefusesUnapprovedOwners |
| `OwnerPolicy.Approves` widened to `uid < 65535` | widen | TestDiscoverRefusesUnapprovedOwners, TestVerifyDetectsSubstitution |
| `AdministratorUIDs` loop deleted | narrow | TestDiscoverRefusesUnapprovedOwners |
| symlink resolution skipped (`canon = path`) | delete | 6 tests |
| malformed prefixed name silently skipped | delete | TestDiscoverRefusesMalformedNames |
| malformed-name refusal narrowed to `len(suffix) > 200` | narrow | TestDiscoverRefusesMalformedNames |
| `ReadDir` failure returned as absence (`return nil`) | absence-as-satisfied | TestDiscoverRefusesPartialReads |
| Verify canonical-path check deleted | delete | TestVerifyDetectsSubstitution |
| Verify digest check deleted | delete | compile error (unused imports) |
| Verify owner policy half deleted (identity half kept) | narrow | TestVerifyDetectsSubstitution |
| Verify `Canonicalize` failure → `return nil` | absence-as-satisfied | TestVerifyTreatsReadFailureAsIntegrityFailure |
| Verify `Inspect` failure → `return nil` | absence-as-satisfied | TestVerifyTreatsReadFailureAsIntegrityFailure |
| Verify `ReadFile` failure → `return nil` | absence-as-satisfied | TestVerifyTreatsReadFailureAsIntegrityFailure |
| Verify re-resolves `record.canon` instead of `record.sourcePath` | bypass | TestOSSystemEndToEnd, TestVerifyAcceptsUnchangedTree |
| Trust builtin gate deleted | delete | TestTrustRefusesBuiltins |
| new unexercised `failInvalid` site added | gate self-test | inventory gate: "call sites without an exercised negative path: provider.go:401" |
| stray `Error{...}` literal outside a constructor | gate self-test | inventory gate |
| raw `fmt.Errorf` in provider.go | gate self-test | TestDiscoverRefusesNonRegularTargets |

The refusal-inventory gate is real: it independently caught a newly added
unexercised refusal site and a stray `Error` composite literal.

## Mutants SURVIVED (5)

| # | Mutant | Meaning |
| --- | --- | --- |
| S1 | trust checks skipped for `source == "path"` (`info.IsRegular = true; info.UID = OperatorUID`) | no test drives a trust-check failure through the PATH source |
| S2 | trust checks skipped for every `source != "plugin_dirs[0]"` | no test drives a trust-check failure through a second plugin directory |
| S3 | Verify non-regular check deleted | `TestVerifyDetectsSubstitution/replaced_with_directory` does not distinguish it |
| S4 | Verify digest compare narrowed to `sum[:1]` vs `digestBytes(...)[:1]` | digest guard proven at 1 of 32 bytes |
| S5 | Verify owner check narrowed: `identity != record.owner` half deleted | that half has no test |

### S3 mechanism, isolated

`trust_test.go:136` sets `fakeFile{content: nil, uid: fakeUID, regular: false}`
— it changes the shape AND the bytes, and the assertion is on `Code()` only,
which is `integrity_failure` for every Verify branch. Isolation experiment
(fixture changed to `content: []byte("bytes")`, i.e. shape-only):

```
fixture=shape-only-change production=prod-intact:            exit 0 PASS
fixture=shape-only-change production=regular-check-deleted:  exit 1 FAIL
  trust_test.go:138: Verify accepted a non-regular replacement, want integrity_failure
```

So the one-line fixture change makes the subtest prove what it names. The
real-filesystem twin `TestOSSystemEndToEnd/non-regular replacement` has the
same blind spot: `os.ReadFile` on a directory errors, so the digest branch
fires there too.

This is the same defect class the producer's own gate caught pre-commit for
the Verify re-read path (recorded in the outcome doc), reintroduced one
subtest away. The inventory gate cannot see it: it requires each constructor
site to be *reached*, and this fixture does reach `provider.go:456` — it just
cannot tell that a different site would have produced the same observable.
By construction the gate also cannot see a *deleted* site.

## Probe: relative PATH entries bypass the absolute-directory gate

`Discover` applies `scalar.ParseAbsolutePath` to `providers.plugin_dirs`
(provider.go:334) but not to `system.PathDirs()` (provider.go:348); both
sources then share `collectDirectory`. Probe driving the production
`OSSystem` + `CurrentOperatorPolicy`, with an operator-owned
`ax-provider-evil` planted in the process working directory:

```
PATH="."                      err=<nil>
  ADMITTED external id="evil" source="path"
  canon=".../T/ax-relpath-probe/ax-provider-evil"
plugin_dirs=["."]             err=provider invalid_config: providers.plugin_dirs[0] "." is not an absolute path
PATH="./../ax-relpath-probe"  err=<nil> admitted=1
```

`OSSystem.Canonicalize` resolves a relative entry with `filepath.Abs`, i.e.
against the process working directory. The owner check does not help: files
in a checkout the operator owns are operator-owned. No test covers a relative
PATH entry, and the outcome doc's Bounds section does not mention it.

## Facts checked and confirmed as reported

- Single hash end to end: `scalar.SHA256Digest` = `hex(sha256(content))`
  (`internal/scalar/time_digest.go:144`); `Verify` recomputes with
  `sha256.Sum256` and compares via `subtle.ConstantTimeCompare` over all 32
  bytes. The reported double-hash fix holds.
- `digestBytes` fails closed: an undecodable receipt yields `nil`, and
  `ConstantTimeCompare` returns 0 on a length mismatch.
- The producer's "ownership JSON is hash-pinned" reason is accurate:
  `internal/traceability/traceability.go:422` compares the registry
  projection against `reviewedOwnershipCanonicalSHA256` and fails otherwise.
- `TestOSSystemDuplicateNeverExecutes` is genuinely behavioural: a live shell
  script fixture, refusal with `invalid_config`, sentinel asserted absent.
- `TestDiscoveryReachesNoProcess` and the AST scans carry blind-scan guards
  ("scanned no production sources; the check is blind").
- Section 7.1 citations resolve against the digest-pinned document via
  `internal/specdoc`, quoting the line each claim begins on.
