REWORK for TASK-260830-2xt6fd, CR rev2 → rev3. Read `TASK-260830-2xt6fd_review-verdict-rev2.md` first. All four rev1 findings are FIXED. The remaining findings come from ONE cause. The rev2 brief asked for the gate list to be derived MECHANICALLY from the refusal sites. Instead `captureAdmissionGateKind` returns `""` for unlisted conditions, and the census skips them. It covers 4 of about 15 `captureUnavailable(...)` sites, and three narrowings survive because of that gap.

DELIVER EXACTLY:
1. **Mechanical census.** A committed test parses `capture_git_workspace.go` with go/ast and collects EVERY `captureUnavailable(...)` call inside `CaptureGitWorkspace` and every function it calls in this package. For each call it extracts the literal detail string. It builds the set {detail literal → matrix test name}, and it FAILS on any site with no matrix row, on any row with no site, and on any site whose detail is not a literal. Unknown sites must never be skipped. Control-plant it: add a new refusal site, and the census must redden. Also plant a site whose detail is non-literal.
2. **A matrix row for every site.** Each row uses the valid-for-all fixture, breaks only that site's condition, and asserts the literal code AND detail. The reviewer's three survivors are included:
   - the closing incarnation change (line 159);
   - the boundary receipt from another incarnation (line 136);
   - a stopped capture carrying a boundary body (line 67).

   The rest follow: identity mismatch, missing incarnation record, owner safe-boundary proof, the closing status mismatch, and every other site the census finds.
3. **Mutants.** Each site gets an admitting narrowing, not only an arm-delete. Where a condition compares values, narrow it to admit exactly one wrong value. Each must be KILLED by its row run ALONE.
4. Nothing else changes; the rev1 fixes hold. Keep the story-final registry current if test names change: re-derive the digest, run tracecheck, check the README pin. Run `GOOS=windows GOARCH=amd64 go vet ./...`, exact-tree hygiene through a scratch index, the harness and the full configured suite. Keep the checklist current, then run `task-board handoff TASK-260830-2xt6fd --role developer`. If trunk moves, run `refresh-candidate` exactly once, as the last step before the handoff.

SCRATCH RULE: no /tmp. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
