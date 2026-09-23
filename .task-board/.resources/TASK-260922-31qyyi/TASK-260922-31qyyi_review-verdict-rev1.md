# TASK-260922-31qyyi review verdict — CR rev1: CHANGES REQUESTED

Reviewer: claude-opus-5-5. Candidate tree d8fb644c on base c9233ce, reviewed in a git-archive copy (live index untouched).

## What holds
- (1) With `checkWinningLease` replaced by `return nil` in a copy, all 8 axpane witnesses PASS: probe 15 yields `{standard, HasSource:false}` at LoadProfile, Projector.Project, SetProfile from-end, and deriveProfile (ev/instrumented.log).
- The design is sound. Every successor lease carries a checkpoint (sessrepo mints `checkpoint_id` at epoch 2 and above). An earlier lease's event counts only inside the winner's attested handoff closure. Never-minted epochs and an empty lease store fall back to the record. An unreadable closure refuses instead of promoting an event.
- Registry: clause 2.4#2 lists `story-260922-derivation-side-profile-source`, and its owner is `internal/sessprofile/authority.go:Derive`. That was decoded, not assumed.
- Build, vet, `GOOS=windows GOARCH=amd64 go vet ./...` and gofmt are clean. `task-board.config.json` is byte-identical to the base. The sessprofile, axpane and sessckpt packages are green.

## P1-1: the derivation-side property has no live witness in the committed suite
In the committed tree, all 8 derivation witnesses call `t.Skip`: 4 losing-lease and 4 ambiguous (ev/pkg-tests.log). My narrowing M1 (`authorizedSource` returns true for any known lease, which admits every losing-lease tail) produced **zero failures in the committed suite**. Only the instrumented copy killed it (ev/mutants.log). M2 (knownLease matches on epoch only) behaves the same way. So CI would accept a regression that brings probe 15 back.

The class does not need the append gate disabled to be exercised. Reviewer probe `ev/zz_review_probe_test.go`: lease A captures a checkpoint C at `session.created`, then appends a `profile.changed` while A is still the winner (the real gate admits it), then B takes over naming C. The candidate correctly returns `{standard, no source}`. Under M1 the probe returns yolo and FAILS. The DoD requires "own tests, own narrowing mutant ... a named test must fail"; in CI no test fails. The registry acceptance case also lists those 8 tests, and all of them skip.

## P2-1: a landed mutant row lost its anchor
`internal/axpane/mutant_harness.py` row `N-profile-closure` still matches `sessprofile.DeriveForHeads(input.ProfileRecord, input.ProfileEvents, heads)`, which the CR removed. The rerun prints `ERROR: N-profile-closure: patch anchors 0, want 1`. That row is now unmeasured.

## Notes
- M3 (isWinningLease ignores LeaseID) survives everywhere. It is equivalent because `knownLease` is checked first and the store holds one lease per epoch. No action needed.
- My own-mutant control was NOT_APPLIED (bad regex). The shipped axpane harness `C-doc-comment` control did apply and SURVIVED.
- Not rerun by me, because the verdict is already changes-requested: digest re-derivation, the full `go test ./...`, the README plant, and compare_importers.py.

## Measured ratio
Derivation entries with a committed test that is not skipped and fails under M1: **0 of 4**. With the append gate instrumented off: 4 of 4.

## Rework scope (for the producer)
1. For each derivation entry (LoadProfile+Derive, Projector.Project and ProjectForHeads, SetProfile from-end, axpane.deriveProfile), add a committed test that is NOT skipped. Build the losing tail with the real gate on, using the probe pattern above (a known-lease event outside the winner's handoff closure). Name these tests in the registry case alongside or instead of the tests that skip.
2. Make sure M1 and M2 are killed by the plain `go test ./...`, and add both as rows in the shipped harness.
3. Re-anchor `N-profile-closure` to `input.ProfileData.DeriveForHeads(heads)` and confirm it reports KILLED.
4. Re-derive the registry digest, then rerun tracecheck and the full suite.
