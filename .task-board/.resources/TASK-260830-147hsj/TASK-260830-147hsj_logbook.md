# TASK-260830-147hsj Logbook

## 2026-09-24 — Durable anti-entropy union handoff

- Implemented validated durable JSON union and discovery/fetch by digest on top
  of the nxqqaw namespace inventory checkpoint. Identical adds survive
  close/reopen idempotently; same-digest/different-byte conflicts quarantine
  and abort; Tombstones and Acknowledgements remain immutable union members.
- Lease heads are derived from the post-union tuple set, with the pinned
  `(epoch, lease_id)` ordering. Arrival-order tests enumerate all 720
  permutations of six additions under two opposing timestamp profiles (1,440
  projection rebuilds) and compare against an independent literal oracle.
- The four rev3 carry-overs are pinned by production-entry tests and isolated
  narrowing mutants: `fetch-dup-ids`, `b64-roundtrip`, `walk-quarantine`, and
  `walk-crossns`. For cross-namespace recursive walking, the malformed
  internal arm is tested; valid ingestion cannot construct it without a
  SHA-256 collision, so public-ingest reachability is recorded as a bound.
- Final-source mutation harness (`mutations-final-cleanup`): 70 narrowing mutants killed, the
  token-preserving identity bypass killed by behavior while source census
  passed, and the applied comment-only neutral control survived. Expected-red
  focused diagnostics and their corrected reruns are itemized in the results
  artifact; they are not counted as final evidence.
- Validation on checkpoint `b81258e3c5bc0f6321ae8f6bea81ec24cc298d70`:
  package/importer tests, native and Windows vet, the full configured Go suite,
  formatting, diff checks, and `golangci-lint run --new ./...` exited 0. The
  optional default `golangci-lint run ./...` exits 1 on the repository's
  existing findings; this checkout has no golangci-lint project config. Its
  first pass found two new `errcheck` issues in this leaf, both corrected;
  the final delta-only lint is clean. Logs and raw mutant outputs are in the
  task-scoped producer evidence archive.
- Boundaries: fixed `sessrepo` authority snapshot plus immutable union; no
  arbitrary event-DAG reconstruction, physical power-loss guarantee, native
  Windows crash execution, blob staging/commit, public `ax sync`, Host Channel
  transport, network anti-entropy, or capability publication is claimed.
- Candidate remains uncommitted in the managed Story worktree for the
  developer-to-review handoff.

## 2026-09-24 — Change Request validation recovery

- CR revision 1 configured validation failed at command 5/30: the uncached full
  suite exited 1 when temporary directories returned `no space left on device`.
  The generated importer baseline tree and tar under this task's scratch used
  2.8 GB; after removing those disposable copies, free disk increased from 3.7
  GB to 14 GB. Importer scan outputs and base/candidate logs remain in the
  attached producer evidence archive.
- A diagnostic full-suite retry with `TMPDIR` nested inside the checkout exited
  1. The config source-census fixture rejected its synthetic `clean.go`, and
  the sessstate census's nested baseline run timed out while other task runs
  were executing tests. That attempt is explicitly not counted as green.
- Isolated reruns with the default system TMPDIR passed:
  `TestWritePathGateFlagsUnlockedCaller` (exit 0) and
  `TestCensusLiveEventOwnershipPlants` (exit 0). Their logs, along with the
  failed diagnostic full-suite log, are included in the updated evidence
  archive. Candidate source and test files remain unchanged from the earlier
  full-suite green evidence.
- `task-board handoff TASK-260830-147hsj --role developer` exited 0 and reported
  status `to-review` with 18/18 checklist items. The candidate remains
  uncommitted at the Story checkpoint.
