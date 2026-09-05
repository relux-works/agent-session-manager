## Round-3 rework (H1 blocking + logbook correction + H2)

### H1 — legacyForward closed-set pin
- New white-box `TestLegacyForwardIsExactlyTheImmutablePair` in `internal/terminalbackend/internal_pin_test.go`: holds the live `legacyForward` map to exactly `{tmux, conpty}` in both directions plus the size (the `replicationMembers` idiom), and pins injectivity (distinct canonical values) behind `ProjectToLegacy` documented determinism.
- `TestHistoricalTranslationMapsOnlyTheImmutablePair` comment restated to its half (sampling bound, closure cited at the pin); refused samples gain `"screen"`; reverse test gains `ProjectToLegacy("vendor.screen")`.
- Mutants, each run repeatedly per the round-3 lesson: L3 (`screen: vendor.screen`) 12/12 RED; C20 (`screen: BuiltinTmux`) 12/12 RED; new pin alone RED on C20 with the size message. Tree verified identical before/after.

### Logbook correction
- Round-1/round-2 entries falsely claimed C20 killed and P11 dropped (reviewer single-run red; actual 5 red / 7 green over 12 runs on map iteration). Correction appended to LOGBOOK.md with the lesson: map-iteration mutants must be run repeatedly.

### H2 — plain-error alias gate (closed cheaply)
- New `rejectPlainErrorAliases` in `refusal_arm_inventory_test.go`: any `errors.New`/`fmt.Errorf`-kin reference outside direct-call position fails the suite in every function. Alias mutant (`var newPlain = errors.New` + call) 3/3 RED. Bespoke-type bound stands, now recorded in the tree.

### Gates (all exit 0, run directly)
- `go test ./... -count=1`: 14/14 ok
- `go test ./internal/terminalbackend/ -race`: ok
- `go vet ./...`: clean; `gofmt -l`: clean
- `tracecheck`: acceptance_cases=76, clauses_discharged=17/403

No production change, no arm-table change, 193/193 undisturbed. Rework left uncommitted per the brief.