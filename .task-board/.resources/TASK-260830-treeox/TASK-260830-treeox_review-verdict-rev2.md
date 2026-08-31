# TASK-260830-treeox reviewer verdict — CR revision 2

## Verdict

Changes requested for `CR-TASK-260830-treeox-2` revision 2. Route the task to
`to-dev` for focused test-evidence rework. The production implementation is
currently correct; the blocking defects are surviving narrowed-gate mutants.
This is ordinary recoverable rework, not a Stop-The-Line boundary.

- Reviewed base: `8441818417458d7e88a46470e40ee376d99eea26`
- Reviewed candidate tree: `9ce4e16f8831238a416504bccb76f9348ad7d6ac`
- Review patch SHA-256: `32bb8e36f4e906a442793ea761e6b360778cc866af1feebbcb0b07dd258d8acf`
- Reviewed paths: 15; all 15 working-tree blobs matched the candidate tree

## Blocking findings

### F1 — acceptance-case production-owner verification lacks a meaningful negative

`verifyAcceptanceCases` verifies each `acceptance.Production` reference at
`internal/traceability/traceability.go:249`, but no shipped negative isolates
that call. Revision 2 correctly added source-declaration negatives for an
acceptance **test** owner and an ownership-group **production** owner. It still
does not cover the production owner attached directly to an acceptance case.

Reviewer mutation, isolated under `.temp/`:

1. Removed only the three-line
   `checker.verify(acceptance.Production, false)` call and its diagnostic.
2. Ran the complete relevant scope with cache disabled:

   ```text
   go test ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1 -v
   ```

3. Result: exit 0; both packages and every shipped negative passed.
4. Added a reviewer-only probe that left the ownership registry unchanged,
   renamed only the unique `writeIfChanged` declaration in the repository
   fixture, and drove production `VerifyRepository`.
5. Result: `VerifyRepository` admitted the missing acceptance production owner;
   the probe passed with exit 0.

This is the same required negative shape as the revision-1 findings, but for a
third distinct owner-resolution call. A regression can stop checking every
acceptance-case production owner while the required package suite remains green.

Required rework:

1. Add a production `VerifyRepository` negative that renames/removes a unique
   acceptance-case production declaration while leaving the registry unchanged.
   `catalog-generation-idempotency-recovery -> writeIfChanged` is an isolated
   target that is not also an ownership-group production reference.
2. Require the exact owner-specific diagnostic, for example:

   ```text
   acceptance case "catalog-generation-idempotency-recovery" production owner:
   declaration "writeIfChanged" is absent from
   "internal/catalog/cmd/cataloggen/main.go"
   ```

3. Re-run the call-removal mutant with `-count=1` and retain expected-red
   evidence naming the new test.

### F2 — the real tracecheck boundary does not prove ownership-loss propagation

`internal/traceability/cmd/tracecheck/main.go:32` calls `VerifyRepository`, but
the command test only exercises a wholly missing repository. It never drives
`run` with a registered contract, normative section, acceptance case, or
fixture whose ownership is missing. The helper-level negatives therefore do
not prove the exact headless CI entry point fails for this error class.

Reviewer narrowing mutation, isolated under `.temp/`:

1. Changed only the `run` error branch to suppress errors containing
   `has no implementation owner`; all unrelated read/parse errors continued to
   fail.
2. Ran:

   ```text
   go test ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1 -v
   ```

3. Result: exit 0; the complete relevant suite passed.
4. Removed the registered Session Directory Query contract owner and launched
   the real command:

   ```text
   go run ./internal/traceability/cmd/tracecheck
   ```

5. Result: exit 0 with the false success line
   `traceability ok: contracts=0 normative_sections=0 acceptance_cases=0 fixtures=0 compatibility_contracts=0`.

The later repository test step is a useful second guard, but it is not proof
that the advertised headless CI gate itself refuses the protected class.

Required rework:

1. Add a command-level negative that drives the production `run` entry point
   against a task-local repository fixture with one registered owner removed.
2. Require `ErrTraceability`/the owner-specific diagnostic and require zero
   success output.
3. Re-run the narrowed error-propagation mutant with `-count=1` and retain the
   expected-red log.

## Revision-1 rework assessment

The prior findings are resolved in revision 2. The delta from candidate tree
`6d9a90eb92a2414ac4e72450a34f4ba4fa68ff6c` is test-only: 51 lines in
`internal/traceability/traceability_test.go`. The two new tests drive production
`VerifyRepository` and isolate:

- `checker.verify(test, true)` for acceptance test owners; and
- `checker.verify(group.Production, false)` for ownership-group production owners.

The producer's two expected-red logs were inspected and show real exit 1 under
the exact revision-1 mutants. I independently reran the resulting green tests;
I did not rerun those already-attached producer mutants because this review
instead exercised the two additional uncovered bounds above.

## Passing evidence

| Check | Reviewer result |
| --- | --- |
| CR integrity | Attached patch SHA-256 matched; all 15 worktree blobs matched candidate tree; exact diff check passed |
| Pinned authority | Local upstream annotated tag and commit signatures verified; `SPEC.md` and all three shipped fixture digests matched the lock |
| Public immutable fetch | GitHub browser fetch returned cache miss; recorded as unknown, not absence; no claim relies on it |
| Targeted six-package suite | Exit 0 with `-count=1 -v` |
| Repository suite | `go test ./... -count=1` exited 0 |
| Coverage | Exit 0; traceability 82.1%, tracecheck 80.0% |
| Race | `go test ./... -count=1 -race` exited 0 |
| Vet | `go vet ./...` exited 0 |
| Builds | Native Darwin, Linux amd64, and Windows amd64 builds exited 0 |
| Production tracecheck | Exit 0 with exact inventory `60/17/15/30/55` |
| Generated catalog | Independent cataloggen output was byte-identical to committed `catalog_gen.go` |
| Formatting and JSON | `gofmt -l` emitted no paths; all reviewed JSON parsed |
| Capability boundary | Public capability type and README remain vocabulary-only and make no runtime availability/support claim |

No reviewed repository file was modified. Candidate archives, reviewer probes,
and mutants exist only under `.temp/TASK-260830-treeox/`.
