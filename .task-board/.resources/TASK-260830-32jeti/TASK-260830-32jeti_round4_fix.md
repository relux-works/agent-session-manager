# TASK-260830-32jeti — Round-4 blocking finding fix (surrogate pair dimension) + Kill-comment advisory

## Scope
Round-4 `changes requested` had one blocking finding (second consecutive of the
same class: a derived sweep carrying one hand-written dimension) and two
recorded advisories. This round fixes the blocking finding and the actionable
half of the second advisory. Nothing else was touched.

## Changes (working tree only, nothing committed)
- `internal/provhost/surrogate.go` — named the gate's range constants
  (`highSurrogateMin/Max = 0xd800/0xdbff`, `lowSurrogateMin/Max = 0xdc00/0xdfff`)
  and used them in `hasLoneSurrogateEscape`. Behavior-identical refactor.
- `internal/provhost/surrogate_test.go` — the pair-second dimension is now
  derived arithmetically from those constants: `{lowMin-2, lowMin-1, lowMin,
  lowMin+1, mid, lowMax-1, lowMax, lowMax+1, lowMax+2, highMin, highMax,
  highMin-1}` (12 seconds x every high surrogate). The old five-element literal
  (`0xdc00, 0xdc01, 0xdfff, 0x0041, 0xd800`) missed exactly the two neighbors
  that separate the correct bound from an off-by-one (`0xdbff`, `0xe000`); both
  are now in the swept set by construction, and shifting a gate bound moves the
  swept neighborhood with it. Verdicts still come from the independent oracle
  (`canonicaljson.Canonicalize`), never from the constants.
- `internal/provhost/runner.go` — fixed the `Process.Kill` backstop comment:
  it named the direct child as the escaper, which the code structurally prevents
  (group leader, own `setsid` fails EPERM). The comment now names the
  grandchild-or-deeper escaper and states the direct kill's true role (racing
  exec's group `Cancel`, which may not have fired yet when the deadline is
  observed). Comment-only change.

## Mutant evidence (production entry point: `decodeStrictObject`, driven by the named test below)
Named test: `TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON`
(`internal/provhost/surrogate_test.go`), 246,798 vectors green on correct code.

| Mutant (site `surrogate.go:83`) | What it narrows the gate to | Named test that fails | Outcome |
|---|---|---|---|
| SGC `low > 0xe000` (was `low > lowSurrogateMax`) | admits lone second `0xe000` | sweep test FAILS — e.g. `{"a":"\ud807\ue000"}` canonical.refuse=true gate.refuse=false | KILLED |
| SGJ `low < 0xdbff` (was `low < lowSurrogateMin`) | admits lone second `0xdbff` | sweep test FAILS — diverges on 2048 of 246798 | KILLED |
| Over-broad `low >= lowSurrogateMax` (defense in depth) | refuses valid pair second `0xdfff` | sweep test FAILS — diverges on 1024 of 246798 | KILLED |

Survivors in this gate's pair dimension: none. Bound stated, not inferred:
the swept set contains both bound endpoints and both adjacent outsiders, so
every one-sided bound shift is killed — a narrowing admits `lowMin-1` or
`lowMax+1`, an over-broadening refuses `lowMin` or `lowMax`. Shifts of 2 are
covered directly (`+-2` swept); larger one-sided shifts admit a superset that
still includes a swept neighbor.

## Gate log (each command run directly, standalone; exit codes real)
- `go build ./...` — exit 0
- `go test ./internal/provhost/ -run TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON -v` — PASS, `gate agrees with canonicaljson on 246798 of 246798 vectors`
- SGC mutant run — test FAIL (exit 1), restored fixed file (verified `low > lowSurrogateMax` back in place)
- SGJ mutant run — test FAIL `diverge on 2048 of 246798` (exit 1), restored
- Over-broad mutant run — test FAIL `diverge on 1024 of 246798` (exit 1), restored
- `gofmt -l internal/provhost/` — clean (no output); `go vet ./internal/provhost/` — exit 0
- `go test ./internal/provhost/` — ok, 8.990s, exit 0
- `go test ./...` — all packages ok, exit 0
- `go test ./... -cover` — exit 0 (provhost 85.8%, canonicaljson 97.2%, others per log)
- No repo lint config exists; `gofmt` + `go vet` are the applied static gates.

## Carried forward, untouched (recorded advisories, not blocking)
- `RN1`/`RN2` (`writeErr` ordering half of the discarded-frame finding) remain
  unpinned, as before. Touching that ordering risks the F1/F2-pinned behavior;
  left for the owning round.
- `RN4`/`RN6`/`RN7`/`RN9` region: branch kept, comment corrected per the
  advisory ("fix the comment or the branch; either is fine"). The direct kill
  is reachable only via the Cancel race, which is timing-dependent and not
  deterministically pinnable from a unit test; stated as a bound.

## AC traceability
Round-4 scope is a subset of the story AC (fail-closed lone-surrogate tuples
through the production decoders). Coverage: the Section 1.6 gate's pair
dimension is now 12 derived seconds x all 1024 high surrogates = 12,288 pair
vectors inside the 246,798-vector agreement sweep against the independent
oracle — every AC-relevant refusal tuple in that class is driven through
`decodeStrictObject` (production call site) on every run.
