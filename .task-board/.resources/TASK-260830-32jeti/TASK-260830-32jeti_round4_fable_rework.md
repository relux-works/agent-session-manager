# TASK-260830-32jeti round-4: Fable consolidated-review rework (B3, B4, A2–A5/A9)

Story worktree: `.temp/STORY-260830-3jqsx1/worktree`, branch
`task-board/story/STORY-260830-3jqsx1`. Round-3 work untouched except where
this round's findings order a change. Nothing committed; all work in the
working tree for integration. `canonicaljson` not modified (coordination
constraint).

## B3 (blocking): lone-surrogate gate in `internal/provhost`

- New `internal/provhost/surrogate.go`: `hasLoneSurrogateEscape` scans raw
  frame bytes before decoding and refuses the same class
  `canonicaljson.validateSurrogateEscapes` refuses (lone high, lone low,
  high-not-followed-by-low, either case). Malformed escapes / unterminated
  strings fall through to the existing JSON-syntax arms. No `switch`
  statements: the closed-vocabulary census
  (`TestAllProductionSwitchesAreClassified`) refuses new switches, so the
  gate is written with if/else (first version used switches and reddened
  that census; rewritten, green).
- `protocol.go` `decodeStrictObject` calls the gate first with the single
  new arm `frame|lone surrogate escape|""`. It fires for every entry
  (manifest, probe, identity, identify-result, quiesce, spawn, response,
  status) since all funnel through `decodeStrictObject`.
- Census floor `refusalArmCensusFloor` 162 -> 164: one frame arm plus its
  automatic `integrity|status body is lone surrogate escape` expansion,
  each witnessed at the production entry (`DecodeResponse`,
  `DecodeStatusOutcome`).
- `surrogate_test.go`: `TestProductionEntriesRefuseLoneSurrogateEscapes`
  (8 entries refuse with code + member + detail + non-retryable) and
  `TestSurrogateGateAgreesWithCanonicalJSON` (13 shared vectors; both
  implementations must reach the same verdict — 9 refuse incl. member-name,
  nested, array, uppercase placements; 4 accept incl. valid pair and
  `\\ud800`-as-text).
- Mutants, both red as required: admit-all gate reddened
  `TestProductionEntriesRefuseLoneSurrogateEscapes`,
  `TestSurrogateGateAgreesWithCanonicalJSON`, and the executed-witness
  test (run twice: switch-form and final if/else-form). Derivation-only
  tests stayed green under the admit-all mutant, confirming the inventory
  derivation is textual while entry tests execute. A5 narrowing mutant
  (drop `pi` from `profileProviders`) reddened the new set-equality test.
- New tests re-run with `-count=5`: green 5/5 (no map-iteration verdict).

## B4 (blocking): ownership registry covers the story's code

- 4 new acceptance cases (`provider-discovery-and-trust`,
  `provhost-quiesce-proof`, `provhost-profile-mapping`,
  `provhost-capability-gate`), each bound to a real production
  declaration and 2 real executable tests.
- 4 new `section_binding` groups: `section:7.1`->`Discover`,
  `section:7.6`->`DecodeQuiesceProof`, `section:7.7`->`ProfileMapping`,
  `section:8`->`RequireCapability`, all `unevidenced` with honest gaps.
- Rewrote the now-false gap texts for `section:5.5` (destination-adapter
  work exists in `CheckIdentity`), `section:7.2`, `section:7.3` (dropped
  "no provider manifest is implemented anywhere"; now names
  `DecodeManifest`), `section:7.5`.
- Re-pinned `reviewedOwnershipCanonicalSHA256` to
  `680a78f3...cf5a`; updated `traceability_test.go` report expectations
  (bindings 49->53, acceptance 74->78, unevidenced 41->45, clauses
  403->428), `tracecheck/main_test.go` golden strings, and README figures
  (ownership paragraph + measured-coverage section).

## Advisory decisions (recorded in code, not silently skipped)

- A2 deliberate: `CheckIdentity` re-validates rather than delegating —
  provhost must mint `provider_protocol_error` with member attribution;
  canonicaljson offers no such entry and may not gain one (frozen).
  Bound recorded in `identity.go`; shape changes must land in both.
- A3 stated bound: Provider Protocol 3.0.0 has no implementation owner
  here; host speaks major 2, refuses 3.x as `incompatible_protocol`
  loudly. Recorded in `provhost/doc.go`.
- A4 bound naming the composition point: `provider.Error` detail carries
  paths/IDs and `Error()` appends the cause, so the future `cmd/ax` lift
  must rebuild the wire message (facts into Details, cause stripped) or
  `axerror.refuseCausalLeak` refuses it. Recorded in `provider/doc.go`.
- A5 closed: `TestSixProviderSetMatchesDiscoveryRegistry` asserts the
  profile registry equals `provider.Builtins()` as sets.
- A9 nit fixed with a comment: `RequireCapability` re-decodes are exact
  replays over just-validated bytes; validation must stay first.
- Out of this leaf's scope (recorded, not changed): A1/A7 (sibling
  story's packages), A6 (shared walker core), A8 logistics, A9
  `parseMajor` leading zeros (classification-only, parse-arm inventoried).

## Evidence (all exit 0 unless noted)

- `go test ./...` green (full suite, incl. traceability + tracecheck).
- `go test ./... -cover` green; touched packages: provhost 86.0%,
  provider 97.0%, traceability 86.6%.
- `go vet` clean; `GOOS=windows go vet` clean (story CI gate);
  `gofmt -l` clean.
- New/inventory tests `-count=5` green; admit-all mutant red (expected,
  restored); `pi`-drop mutant red (expected, restored).
- Ratios: surrogate vectors 13 (9 refuse / 4 accept, both
  implementations); refusal arms 164/164 witnessed; traceability report
  bindings=53 unevidenced=45 clauses 17/428; six-provider set 6/6 both
  planes.
- Bounds: CheckIdentity still skips `record_id` recomputation (prior
  round); §8 cell values remain evidence labels; v3 envelope unowned;
  provider->Structured Error lift rule documented, no call site yet.
