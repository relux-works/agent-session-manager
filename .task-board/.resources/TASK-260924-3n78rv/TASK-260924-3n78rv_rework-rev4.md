REWORK for TASK-260924-3n78rv, CR rev3 → rev4. Read `TASK-260924-3n78rv_review-verdict-rev3.md` first. The refresh onto current trunk succeeded (checkpoint `618d78d`). Do NOT refresh again unless trunk moves. If it does, run `refresh-candidate` once, as the last step before the handoff.

The finding is `escape-invalid-utf8-value-change`. `EscapeVisibleText` passes its input to `encoding/json.Marshal`, which silently rewrites invalid UTF-8 as U+FFFD and succeeds. Only `ProjectVisibleText` validates first. This is a gate pinned at one entry. The same class has already cost other leaves rounds.

INVARIANT: every EXPORTED entry of `internal/cloneplanning` that accepts text (a string, `[]byte`, or a struct with text fields) refuses invalid UTF-8 BEFORE any encoding step. No entry ever returns a value whose text differs from its input.

DELIVER:
1. **Entry census by AST, not by hand.** A committed test parses the package's production files and lists every exported function or method with a text-carrying parameter (string, `[]byte`, or a struct type containing either, followed transitively). It must fail if any such entry is missing from the refusal table below. Control-plant it: add a new exported text entry with no refusal row, and the test must redden.
2. **The refusal table.** For every censused entry × every invalid-UTF-8 class (lone continuation 0x80, `ff fe`, overlong `c0 af`, lone surrogate `ed a0 80`, sequence truncated at the end), the entry refuses with the literal code. It must not return a U+FFFD-substituted value.
3. **The fix, by construction.** Route every text entry through ONE validated text type or function shared with `ProjectVisibleText`. No entry may call `json.Marshal` on unvalidated input. Add a structural guard over the package: a text value may only reach `json.Marshal` after the validator.
4. **Mutants.** Ship the reviewer's narrowing (skip validation for one invalid-byte class at `EscapeVisibleText`) plus one at any other censused entry. Each must be KILLED by its named test run ALONE.
5. Keep every held row unchanged. Attach complete command evidence for every configured-suite command you rely on, as the reviewer asked. Run `GOOS=windows GOARCH=amd64 go vet ./...`, exact-tree hygiene through a scratch index, the harness and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260924-3n78rv --role developer`.

SCRATCH RULE: no /tmp. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: muse-spark max. Reviewer: gpt-6-sol medium.
