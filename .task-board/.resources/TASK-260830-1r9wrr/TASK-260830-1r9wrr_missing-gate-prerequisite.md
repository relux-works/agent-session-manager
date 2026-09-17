# Missing-gate prerequisite evidence (before broader rework)

R2-F1 / repeat-of rev1 F3: `TestEventTypeUsesAreOwned` derives every production
selector named Type through invcore and requires a bijection with four
AST-context ownership rows. New files and package bindings are scanned;
unregistered, orphan, duplicate and unclassifiable uses fail closed. This
controls copies/helper arguments/address-taking at their originating field read,
without enumerating new event spellings. Existing dispatch census is composed
with this ownership prerequisite.

`go test ./internal/sessstate -count=1 -run '^TestCensusLiveEventOwnershipPlants$' -v`
exited 0. Every nested command and real exit code is in live-ownership-02.log.
The test copies the package and links dependencies read-only; managed sources
are never planted in place. Exactly anchored selectors derive from test AST:
52 behavior tests, 8 census tests, 1 static audit test, pairwise disjoint.
TestMain runtime refusal coverage is reserved for unfiltered package runs.

| Mutant | Restriction weakened / change | Build | Behavior | Census | Audit | Named failing test / survivor bound |
| --- | --- | --- | --- | --- | --- | --- |
| PA | new direct selector handler | 0 | 0 | 1 | 0 | TestEventHandlingIsCensused; TestEventTypeUsesAreOwned |
| PB | copied kind switched in helper | 0 | 0 | 1 | 0 | TestEventTypeUsesAreOwned |
| PC | raw type passed into string helper | 0 | 0 | 1 | 0 | TestEventTypeUsesAreOwned |
| PD | pointer to raw type read in helper | 0 | 0 | 1 | 0 | TestEventTypeUsesAreOwned |
| XN | identity transform of tail ID | 0 | 0 | 0 | 0 | SURVIVED: behavior-neutral identity only |
| XK | unknown default moves to idle | 0 | 1 | 0 | 0 | TestReduceTreatsUnknownV1TypeAsInert |
| G-N | ownership gate admits only the kind := event.Type assignment shape | 0 | composed test exit 1 | PB census goes green as planted | unchanged | TestCensusLiveEventOwnershipPlants/PB |

4 of 4 dispatch plants killed, all census-only; 0 of 4 behavioral kills.
After each delivered gate ran, an independent probe was generated from the
plant's unknown literal and drove Reduce: TestPlantUnknownV1Effect failed
with idle for all four plants (exit 1, expected red). These are control-effect
witnesses, NOT retroactively counted as delivered behavioral kills. PB/PC/PD
preserve all former searched tokens and registered event cases. XK also keeps
the event selector and cases intact while the real behavioral suite reddens.
G-N retains the ownership gate and admits one alias shape; the composed test
fails specifically at live PB (expected red exit 1). Harness command exit 0
means the expected red was observed, never that the weakened gate passed.

Residual bound: this is direct-field-access ownership, not whole-program taint
analysis. It does not prove reflective/unsafe/serialization access to whole
Event values or later interpretation of permitted diagnostic strings. Unrelated
selectors named Type also require classification (conservative false positives).
The gate makes no exhaustive arbitrary-Go-dispatch claim. Parse-only controls
are separate from compiling live controls.

No broader F7/delivery claims were revised before recording this prerequisite.
