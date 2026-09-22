## Reviewer runtime outcome comparison

Mask: `go test -json -count=1 ./internal/axpane ./internal/crashgate ./internal/fencing ./internal/sessckpt ./internal/sessprofile ./internal/sessquery ./internal/sessstate ./internal/termbind ./internal/terminstance`. Both trees exit 0, 9/9 packages PASS. Baseline is actual trunk 799c338, not just the parent checkpoint. The Imports/TestImports/XTestImports sets were derived mechanically with `go list` (direct imports, internal tests and external tests).

The instrumentation records `(executing package, event type, epoch, sequence, relation to durable winner, actual AppendEvent outcome)`, not a test name. Winner lookup occurs in a deferred read after the repository lock is released; `no-readable-winner` explicitly does not distinguish missing state from a read error. Exact patches are attached. This diagnostic instrumentation is not the source of the pristine candidate suite result.

| Importer | Before outcome keys / calls | After outcome keys / calls | Count-delta keys |
| --- | ---: | ---: | ---: |
| `axpane` | 23 / 221 | 26 / 244 | 13 |
| `crashgate` | 1 / 15 | 1 / 15 | 0 |
| `fencing` | 4 / 8 | 4 / 8 | 0 |
| `sessckpt` | 3 / 28 | 3 / 28 | 0 |
| `sessprofile` | 12 / 88 | 16 / 94 | 4 |
| `sessquery` | 54 / 2360 | 54 / 2360 | 0 |
| `sessstate` | 23 / 160 | 23 / 160 | 0 |
| `termbind` | 10 / 46 | 10 / 46 | 0 |
| `terminstance` | 0 / 0 | 0 / 0 | 0 |

Total: 130 keys / 2926 calls before; 137 keys / 2955 calls after. Set delta: 10 new keys, 3 removed keys. The suite includes changed and new fixtures, so counts are evidence about executed input classes, not proof that identical tests kept identical meanings.

| Runtime tuple (package, event, epoch, seq, winner relation, outcome) | Before count | After count |
| --- | ---: | ---: |
| `axpane, profile.changed, 1, 2, lower-epoch, ADMIT` | 1 | 0 |
| `axpane, profile.changed, 1, 2, lower-epoch, STALE` | 0 | 2 |
| `axpane, profile.changed, 2, 1, higher-epoch, ADMIT` | 0 | 2 |
| `axpane, profile.changed, 2, 2, same-epoch-loser, DIVERGENT` | 0 | 2 |
| `axpane, session.created, 1, 1, same-winner, ADMIT` | 144 | 154 |
| `axpane, session.failed, 1, 2, same-winner, ADMIT` | 9 | 12 |
| `axpane, session.failed, 2, 1, higher-epoch, ADMIT` | 0 | 2 |
| `axpane, session.idle, 1, 2, lower-epoch, ADMIT` | 3 | 0 |
| `axpane, session.idle, 1, 2, same-winner, ADMIT` | 11 | 14 |
| `axpane, session.parked, 1, 3, lower-epoch, STALE` | 0 | 1 |
| `axpane, session.parked, 2, 2, same-epoch-loser, DIVERGENT` | 0 | 2 |
| `axpane, session.stopped, 1, 3, lower-epoch, ADMIT` | 3 | 0 |
| `axpane, session.stopped, 1, 3, same-winner, ADMIT` | 10 | 13 |
| `sessprofile, profile.changed, 1, 2, lower-epoch, STALE` | 0 | 1 |
| `sessprofile, profile.changed, 2, 2, same-epoch-loser, DIVERGENT` | 0 | 2 |
| `sessprofile, session.created, 1, 1, same-winner, ADMIT` | 0 | 1 |
| `sessprofile, session.created, 2, 1, no-readable-winner, ADMIT` | 0 | 2 |

Interpretation:

- The landed losing `profile.changed` input moves ADMIT -> STALE; the additional stale Emit and SetProfile rows also return STALE. Direct AppendEvent and EmitParked moves are independently driven by probe 9.
- The three landed old-owner stop fixtures no longer execute `session.idle` seq 2 and `session.stopped` seq 3 under a lower epoch after takeover. They now execute those same events before takeover (`same-winner`), matching their recorded §5.2/§5.3 argument.
- Newly exercised same-epoch losing `profile.changed` through Emit/SetProfile and `session.parked` through EmitParked return DIVERGENT for both losing ID directions. The new parked lower-epoch row returns STALE.
- New same-epoch test setup adds higher-epoch profile/failed events and no-store epoch-2 created events; these remain admitted as the disclosed higher-epoch/no-store bounds, not newly fixed ambiguity.
- Added ordinary created/failed fixture counts remain ADMIT. The seven other importer packages have zero append-outcome count deltas. `terminstance` imports sessrepo but executes zero AppendEvent calls in this suite; its package tests do run and pass.
