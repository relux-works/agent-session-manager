# Shared environment boundary (rev4: delegation + shape census)

One environment library (`internal/environ`) behind the Provider,
Session Adapter, and Directory Node facades, with shared fixtures
and identity. Round 2 closes review-verdict-rev3 (B1/B2/F1/F2) and
records the residue in TASK-260906-33xcnc.

| # | Shared rule | Canonical owner and production call sites | Agreement battery |
|---|---|---|---|
| 1 | Strict-frame decoder + string-walk surrogate semantics | `environ.DecodeStrictObject`, called from `sessadapter.decodeStrictObject` and `dirnode.decodeStrictObject` (delegating wrappers translating `environ.Fault` into the package refusal dialect; pinned by `TestDelegatingWrappersCallEnviron`). `provhost.decodeStrictObject` and `canonicaljson.decodeStrict` remain independent string-aware copies (residue). | `frame_agreement_test.go`: environ verdict plus the other four judges per row, incl. the even-run-plus-real-escape compositions (admit-hole side) and run-parity rows (over-strict side). |
| 2 | Rune string measure | `environ.StringLength`, called from `environ.CheckStringBounds`; `provhost.runeLength`, `sessadapter.stringLength`, `dirnode.stringLength` | `measure_agreement_test.go` incl. multibyte vectors |
| 3 | Byte string measure (Section 5.1 only) | `provhost.checkSpawnArgv`, `provhost.checkSpawnEnvLiterals` | `measure_agreement_test.go` byte gate + `TestByteBoundsStayBytes` |
| 4 | Environment Tuple model | `environ.DecodeTuple`; `sessadapter.DecodeTuple`, `sessadapter.DecodeTupleEntry` | `tuple_agreement_test.go` |
| 5 | Environment Observation model | `environ.DecodeEnvironmentObservation`; provider-owned member sources (dirnode `/validate/*`, provider registry) produce it | `observation_test.go` (matrix + 8-law table) |
| 6 | Shared grammars | `scalar` (env-id, SemVer, digestion); `provhost` (provider-id); extensions reverse-DNS | `identity_agreement_test.go` + grammar one-language (fresh-name copies fail) |
| 7 | Boundary census | Name layer (`census_test.go`) + shape layer (`shape_census_test.go`, 25 rows) + `TestShapeCensusCatchesControls` (14 controls) + `TestGrammarCensusCatchesFreshName` | Live-planted 7/7 shapes fail; evidence in the round-2 report |

Tombstone: the original `hasLoneSurrogateEscape` raw-scan divergence
(escaped-backslash text refused at the surrogate arm) is GONE, not
ledgered: both facades delegate. The recorded divergence it replaced —
the admit hole (`\\ud800` text + real lone low admitted while
`encoding/json` rewrote U+FFFD) — is closed and proven in both
directions by the corpus. The remaining over-strict copies
(provhost/canonicaljson/scalar gates, per-helper copies, sibling
`rawUint53` texture, 67-exit mutant residue) are unification residue in
TASK-260906-33xcnc, not accepted divergence.
