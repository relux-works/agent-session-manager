REWORK for TASK-260830-2ya5le, CR rev2 → rev3. Read `TASK-260830-2ya5le_review-verdict-rev2.md` first. The `byte_counts` stated bound is ACCEPTED, so keep it exactly as it is. The base is trunk `0ca3e4c`. If trunk moves before your handoff, run `refresh-candidate` first.

The finding is `unmeasured-build-utf8-sites`. `TestBuildUTF8` covers four string members, but not `source_class`, `target_locator` or `forbid_reasons`. Close the CLASS, not the three sites.

INVARIANT: every string-valued member of every shape this leaf builds or decodes refuses invalid UTF-8, at every entry (`BuildFidelityReport`, `DecodeFidelityReport` and any other public entry), with `ErrInvalid` and a member-specific literal detail.

DELIVER:
1. **Enumerate by reflection.** The test must walk the Go types by reflection (`FidelityDispositionRecord`, the report struct, nested maps/arrays of strings, reason codes, extension keys and values), so that the set of string members is DERIVED from the types and not typed by hand. A new string field is then covered automatically. Assert that the derived set is non-empty, and list it in the matrix.
2. For every derived member × every invalid-UTF-8 class (lone continuation byte 0x80, `ff fe`, overlong encoding `c0 af`, lone surrogate `ed a0 80`, sequence truncated at the end) × both entries: assert the refusal and its literal detail.
3. **Narrowings.** Ship an admitting narrowing for the three reviewer sites and one for a member of each shape, each KILLED by the reflection test run ALONE. Keep the neutral control.
4. Remove the extra blank line at the end of `mutant_harness.py` (`git diff --check`). Update the gate × entry census, TRACEABILITY and the measured AC ratio. Run `GOOS=windows GOARCH=amd64 go vet ./...` and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260830-2ya5le --role developer`.

SCRATCH RULE: no scratch under /tmp. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
