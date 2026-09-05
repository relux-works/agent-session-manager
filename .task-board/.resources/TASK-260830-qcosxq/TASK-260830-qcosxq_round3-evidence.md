# TASK-260830-qcosxq round-3 rework evidence

## Production files touched
None. Test-only round: `internal/provhost/refusal_arm_inventory_test.go` (new),
`protocol_test.go` (+4 F30 rows, comment), `status_test.go` (+1 X19 row),
`runner_test.go` (+2 F31 tests, group-kill comment correction), `README.md`
(pin paragraph, no behavior claim change). `inventory_test.go` byte-untouched.
No commit made; rework left in the working tree.

## G-A mechanism: derived refusal-arm inventory
`internal/provhost/refusal_arm_inventory_test.go` derives the arm set from
production AST (same technique as `canonicaljson/grammar_inventory_test.go`):
- `frame|<detail>|<member source>` per `&frameFault{...}` literal
- `integrity|<detail>` per literal `integrity(...)` call; the single
  `"status body is "+fault.detail` wrapper expands over the
  decodeStrictObject fault-detail set only (checkResponseMembers arms can
  never flow through it — expanding over all literals over-generated 4
  phantom arms, fixed during development)
- `parse|<enclosing condition>` per `return 0, false` in parseMajor (5 keys)
- `ctor|<constructor>|<detail>` per literal refusal-constructor call
- Non-literal shapes land as `expr:<source>` obligations, never silent
- Fail-closed: zero files / zero arms / zero parse branches / unparseable
  source all fail; census floor 60 (a silently truncated scan fails even
  when every surviving arm is witnessed)
- Both directions: `TestDerivedRefusalArmsAreAllWitnessed` (forward) and
  `TestWitnessedArmsAreAllDerived` (reverse — orphaned witnesses fail, so a
  truncated derivation cannot pass vacuously)
- `TestEveryArmWitnessRefusesAtTheProductionEntry` drives every witness
  through DecodeResponse / EncodeRequest / Host.Call / DecodeStatusOutcome
  with attributed detail, member, exit code, and status_state.

Result: refusal arm coverage 60/60 derived arms witnessed.

## Mutant verification (each: sha-recorded, cp backup, exact-once plant,
## presence grep, targeted run, cp restore, byte-verified restore)

| Mutant | Target | Result |
| --- | --- | --- |
| M23 major-digit `\|\|`→`&&` (F30) | new `a.0.0` row | RED: `incompatible_protocol` (exit 6) vs want `provider_protocol_error` — the exact F30 flip |
| M23 | arm witness + both direction tests | RED |
| M22 rest-digit `\|\|`→`&&` | both direction tests | RED (key rewrite orphans witness + adds key) |
| S4 stdoutCap `+2`→`+1` (F31 narrow) | pin + maximal-frame test | RED: maximal-frame test fails with `want a failure, got nil` — host accepts the forbidden frame |
| S4b stdoutCap `+2`→`*4` (F31 wide) | pin test | RED (behavioral stays green, as disclosed) |
| G1 new frameFault arm via existing site | forward test | RED: unwitnessed `frame\|member is too long\|protocol_version` |
| G3 new integrity arm via single site | forward test | RED: unwitnessed `integrity\|status state is too long` |
| X19 opid-string gate `&& false` | new row + witness | RED |

9 red-as-required, 0 problems. All mutants restored byte-identical
(sha256/cmp verified); production grep confirms no mutant text lingers.

## Small items
- X19: `non-string operation id` row added (status_test.go) + inventory
  witness asserting `status_state == ""`.
- AS6: `status_state` now asserted on all 20 integrity witnesses via new
  `requireIntegrityRefusal` (code + detail + status_state + exit 9 +
  non-retryable). Deviation: separate helper, not folded into
  `requireLocalRefusal`, because that helper also serves codes with no
  status_state detail (invalid_config, process, timeout).
- Group-kill comment (runner_test.go): corrected to what the timing
  measures; U2 beyond Setpgid consistency recorded as unknown on macOS.
- S11/S12/S13/S4b accepted as stated bounds (unchanged).

## Gates (exit codes observed directly, no pipes)
- `go build ./...` → 0
- `go vet ./...` → 0; `GOOS=windows go vet ./...` → 0
- `gofmt -l` (repo) → clean
- `go test ./... -count=1` → 0, all 15 packages ok
- `go test ./internal/provhost/ -cover` → 88.4%; `-race` → 0
- `tracecheck` → ok, 17/403 clauses, 74 acceptance cases (unchanged)

## Bounds (unchanged, restated)
- `provhost` does not import `internal/provider` (comment-only mention at
  runner.go:44); `provhost` has no importers; no `cmd/`. Every Host.Call
  claim is made from a test.
- Merged-identity bound: arms sharing one (constructor, detail) identity
  (two timeout sites, two empty-response sites, six identical
  not-a-JSON-object sites) merge to one obligation; deleting one of a
  merged pair is a behavioral mutant for the entry-point tests, not this
  inventory. The `failMismatch` arm is keyed by its observed detail.
