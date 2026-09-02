# TASK-260830-17suox rework result (CR revision 4 candidate)

## Delivered behavior

- Exact closed TOML readers for Configuration 1.0.0, 2.0.0, and 3.0.0 are driven through production `Load`/`Decode`; v1/v2 translate in memory to the Configuration 3.0.0 model and `EncodeCurrent` emits only 3.0.0.
- Present empty closed objects are distinct from absent required members for both inline `extensions = {}` and multiline `[entry.extensions]` TOML. A schema-driven reflection walk restores only map members proven present in the generic decoded document shape.
- The empty-object round-trip test derives the complete `map[string]any` member inventory from `rawV3`; adding a new closed map member without a writer/read-back case fails the test.
- Every pinned Configuration reader has positive native-Windows and WSL2 production-entry cases. Legacy `conpty` translation, Windows/current defaults, peer-platform path grammar, every nested peer workspace-root refusal, scan-authority minimum, logical-root grammar, shortest reverse-DNS namespace, and extension safe-integer bounds are pinned.
- Full package runs derive every production `configError` call site from Go AST and fail if a refusal site has no exercised negative path. The one defensive secondary parse site is explicitly marked as subsumed by the already-successful envelope TOML parse.
- The reverse-DNS grammar is the named subsuming check for the three-byte extension namespace minimum; redundant length logic was removed while accept-at-limit/refuse-past-limit behavior remains proven.
- Existing SSH bypass, trust-enable, terminal capability, closed shape, exact version, bounds, and legacy-vocabulary refusals remain intact.
- README and pinned ownership traceability now describe and own the new derived/platform/refusal evidence without claiming durable migration, downgrade mutation, doctor output, or runtime capability availability.

## Reviewer class resolution

1. Fixture-shaped empty-map coverage is replaced by a derived raw-schema inventory covering all four current members: installation/profile/disclosure `extensions` and backend `settings`.
2. The go-toml empty-table ambiguity is repaired in the production reader without weakening required-member validation; absent members still refuse.
3. Unpinned enforcement is replaced by an executable AST-derived refusal inventory plus production-entry boundary/platform cases.
4. Section 6.2 now has positive Windows and WSL2 cases for every pinned reader version, and the ownership registry explicitly references those declarations.
5. All cited narrowing classes are killed, including peer-vs-host platform binding, count-only nested-root validation, missing scan-authority minimum, legacy/current Windows backend defaults, legacy conpty mapping, widened logical-root/extension grammars, unsafe int64 acceptance, and map restoration narrowed away from settings.

## Verification summary

- Focused Configuration coverage: 93.2%, exit 0.
- Full tests: all 9 packages pass with `-count=1`, exit 0.
- Full coverage: all 9 packages pass; Configuration 93.2%, exit 0.
- Mutation attack: 109 total mutants, 109 killed with inner `go test` exit 1, 0 survivors, 0 invalid; outer sweep exit 0.
- Global traceability: `acceptance_cases=32`, `assigned_scopes=0`, exit 0.
- Scoped Sections 3.2, 6.1-6.5, 17.1-17.2 traceability: `assigned_scopes=8`, exit 0.
- `go vet ./...`, `go build ./...`, `go mod verify`, generated catalog check, `gofmt -l internal`, and `git diff --check`: exit 0.
- Durable write/crash recovery is not applicable to this read/in-memory-write task. `Load` remains read-only/idempotent; durable backup/fsync/atomic migration and downgrade diagnostics remain sibling `TASK-260830-1qf777` scope.

See `TASK-260830-17suox_rework-rev4-validation.md` and `TASK-260830-17suox_clause-mutation-sweep-rev4.log` for exact run evidence.

## CR revision 5 rework

- Added four isolated `Load` refusal cases for the complete reflection-derived
  closed-map inventory. Each fixture starts from a valid current wire document
  and omits only its target `extensions` or `settings` member, then requires the
  exact raw required-member clause.
- Replaced the self-referential extension byte fixture with the normative
  `65_536` literal from pinned SPEC v0.5.0 Section 6. The fixture now proves both
  accept-at-limit and refuse-past-limit independently of
  `maxConfigExtensionBytes`.
- No production behavior changed in this rework; it closes the rev4 evidence
  class where tests passed for an earlier or self-derived reason.
- Gate attacks killed all six reviewer-shaped mutants: unconditional restoration
  of absent maps, each of the four individual presence-disjunct removals, and
  widening the canonical extension bound to `655_360`. Every expected-red run
  exited 1, and source restoration was verified byte-for-byte.
- Final focused Configuration coverage remained 93.2%. Full tests, full
  coverage, vet, build, module verification, scoped traceability, formatting,
  and diff checks exited 0.

See `TASK-260830-17suox_rework-rev5-validation.md` and
`TASK-260830-17suox_rework-rev5-evidence.tar.gz` for exact run evidence.
