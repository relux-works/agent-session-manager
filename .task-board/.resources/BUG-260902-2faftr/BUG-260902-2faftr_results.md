# BUG-260902-2faftr — admit extension keys by grammar alone

## Provenance

The accepted, reviewed patch from the discarded element BUG-260902-3ru6vw was
reapplied, not reimplemented. It did **not** apply to current trunk unchanged:
BUG-260902-beqfwr had since landed and removed `parseSSHConfigOption`, so the
helper-removal hunk (`internal/config/validation.go`) and the README insertion
point had to be relocated by hand. The resulting code and test deltas are
identical to the accepted patch; only their line context moved.

`git apply --check` on the attached precondition patch exits 1 against this
trunk (two hunks reject); the relocated result is commit `4719092` on
`task-board/story/STORY-260902-2oiugz`, forked from trunk `67aed0b`,
`git rev-list --count HEAD..main` = 0.

## Change

- `internal/config/validation.go`: `hasForbiddenConfigName` and **both** call
  sites removed. `validateExtensions` now decides by
  `len(key) > 253 || !reverseDNSPattern.MatchString(key)` alone;
  `validateExtensionValue`'s `map[string]any` arm enforces only `depth >= 4`.
- `internal/config/refusal_test.go`: the two cases that pinned the invented rule
  ("extension forbidden root name", "extension forbidden nested name") removed
  with the rule, not left asserting it.
- `internal/config/extension_key_admission_test.go` (new): 45 subtests driving
  the production entry. `loadConfigDocument` (schema_test.go:622) constructs
  fixture inputs and calls the exported `Load` — the production call site.
- `README.md`, `LOGBOOK.md`: rationale and the failure shape recorded.

## Gates (all run on this tree, real exit codes)

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l ./internal` (no output) | 0 |
| `go test ./internal/config -count=1` | 0 |
| `go test ./internal/config -count=1 -v -run TestExtension` (45 subtests PASS) | 0 |
| `go test ./... -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `tracecheck -section 6.1 6.2 6.3 6.4 6.5 17.1 17.2 17.4` | 0 |
| `cataloggen -metadata ... -contracts ... -check` | 0 |

## Coverage

| Package | Before | After | Delta |
| --- | ---: | ---: | ---: |
| `internal/config` | 93.9% | 93.9% | 0.0pp |

Baseline measured from a `git archive HEAD` copy of the same tree extracted to
`.temp/BUG-260902-2faftr/baseline` (removed after measurement). No regression.

## Mutants — each applied singly, suite re-run, tree restored green after each

| Mutant | Exit | Subtests reddened | First failing subtest |
| --- | ---: | ---: | --- |
| Re-add key blacklist (**the AC narrowing mutant**) | 1 | 17 of the 20 admission subtests | `TestExtensionKeyAdmissionIsDecidedByTheReverseDNSGrammarAlone/admits_works.relux.env-tools` |
| Re-add nested-key blacklist | 1 | 10 | `TestExtensionValueObjectKeysAreAdmittedAsData/admits_{"works.relux.fixture" = { endpoint = ... }}` |
| Drop grammar gate (widening) | 1 | 11 | `TestExtensionKeyRefusalStillEnforcesTheReverseDNSGrammar/refuses_single_label_has_no_dot` |
| Widen 253-byte key bound to 254 | 1 | 2 | `.../refuses_a_key_longer_than_253_bytes` |
| Widen object depth bound 4 -> 5 | 1 | 2 | `.../still_refuses_object_nesting_past_depth_4` |
| Drop registered backend-settings schema call | 1 | 2 | `TestExtensionAdmissionDoesNotClaimSecretDetection/SPEC.md:2596-2597_...` |
| Drop `DisallowUnknownFields` | 1 | 5 | `TestLoadRefusesUnknownClosedMembersUnsupportedVersionsAndMalformedReads/1.0.0_unknown_root` |

Counts are `--- FAIL` subtest lines only; parent-test lines are excluded. The
17-of-20 figure reproduces the accepted review's measurement exactly. The three
surviving admission subtests are the grammar edges `a.b`, `a1.b-c` and
`z{63}.b`, which carry no blacklisted label and so are invisible to the mutant.

The first two are narrowing mutants: they re-add the removed rule rather than
delete a gate, so they prove the admitted class, not merely that a branch exists.

## Not run

Nothing in the repository gate set was skipped. The fuzz targets
(`FuzzScalarProductionEntries`, `FuzzCanonicalizeRoundTrip`,
`FuzzObjectIdentityRepresentationInvariant`, `FuzzClosedIdentityShapeRefusal`)
were not run: they cover `internal/scalar` and `internal/canonicaljson`, which
this delta does not touch.
