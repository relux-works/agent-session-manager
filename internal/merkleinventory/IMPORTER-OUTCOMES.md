# Direct importer outcome grid

This document preserves the Story fork-base `0ca3e4c` to an earlier rev3
candidate grid as historical evidence. The task-scoped section at the end is
the authoritative final comparison for this handoff: it uses that same fork
base and the exact uncommitted candidate at checkpoint `86a6188a0e35fd04a24eba62a01b71a54d6faa5d`
plus the final leaf. It includes the later trunk additions visible in the
candidate; it does not substitute a Story checkpoint for the fork base.

Baseline package graph: `.temp/TASK-260830-nxqqaw/importer/go-list-base-0ca3e4c.json` (fresh `git archive` of checkpoint `0ca3e4c26e2b275212796657f785b9b450f6174e`, 44 paths).
Candidate graph: `.temp/TASK-260830-nxqqaw/importer/go-list-candidate-final.json` (rev3 candidate on the Story worktree, 45 paths; regenerated after the final tests were added).
Both graphs were generated with Go 1.25.5 on darwin/arm64. The JSON streams contain one `go list -json ./...` object per package; importer edges include production, internal-test, and external-test imports. Rev3 adds no direct import edge beyond the established candidate graph. The unchanged base checkpoint and exact rev3 candidate importer set were tested serialized as `go test -json -p=1 -parallel=1 -count=1`: all 26/26 base paths and 27/27 candidate paths passed, with 21,792 and 22,854 passing test events respectively (14 skipped on each side). Raw logs are `.temp/TASK-260830-nxqqaw/validation/rev3/importer-base.json` and `importer-candidate-final.json`; their metadata records command, working directory, package count, and exit 0.

The first overlapping base/candidate invocation exited 1 on both sides after nested Go subprocesses emitted `Test I/O incomplete` / `WaitDelay expired before I/O complete`. The serialized reruns above are the accepted importer comparison. The failed raw runs remain under `.temp/TASK-260830-nxqqaw/validation/importer-base-tests-0ca3e4c.json` and `importer-candidate-tests-0ca3e4c.json` as diagnostics.

## Historical importer graph delta at earlier rev3

| Imported package | Base → candidate paths | Added path | Existing direct importers |
| --- | ---: | --- | --- |
| `internal/canonicaljson` | 22 → 23 | merkleinventory | axerror, axpane, cliresult, clonebundle, cloneproject, clonesnap, config, crashgate, dirnode, environ, fencing, matjournal, merkleinventory, provhost, rpcwire, sessadapter, sessckpt, sessprofile, sessquery, sessrepo, sessstate, termbind, terminalbackend |
| `internal/rpcwire` | 3 → 4 | merkleinventory | hostchannel, merkleinventory, meshneg, rpcwire |
| `internal/provhost` | 5 → 5 | none | axpane, environ, resumesmoke, sessprofile, sessquery |

## Runtime outcomes on both source identities

The baseline runtime grid is the 26-package direct-importer test process from
the clean archive of checkpoint `0ca3e4c26e2b275212796657f785b9b450f6174e`;
all 26 package results and 21,792 passing test/subtest events are green. The
rev3 candidate runtime grid is the 27-package direct-importer process; all 27
package results and 22,854 passing test/subtest events are green. Neither run
has a failing package. The package keys are in
`.temp/TASK-260830-nxqqaw/importer/importer-packages-{base,candidate}-rev3.txt`.
The exact run artifacts are the base and candidate JSON streams named above;
the candidate stream was rerun after the final rev3 wire-envelope tests were
added. The `(imported package, importer entry, input class)` rows below record the
observed base → candidate behavior; every moved class is named, and all other
rows remain unchanged. A separate full configured candidate suite is rerun
for the handoff and its exit codes are recorded in the result artifact.

## Per-importer entry and input outcomes

Each row is keyed by package, imported entry, and input class. `prod`, `test`, and `xtest` show the source channel. API references are enumerated from candidate Go files, rather than inferred from package names.

| Imported package | Direct importer | Channel | Entry | Input class | Base → candidate outcome |
| --- | --- | --- | --- | --- | --- |
| `internal/canonicaljson` | `axerror` | `prod` | `CanonicalByteLength` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `axerror` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `axerror` | `test` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `axpane` | `prod` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `axpane` | `prod` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `axpane` | `prod` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `axpane` | `prod` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `axpane` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `axpane` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `axpane` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `axpane` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `cliresult` | `prod` | `CanonicalByteLength` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `cliresult` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `cliresult` | `prod` | `ValidateObservationEvent` | Observation Event v1.0.0 schema input | unchanged |
| `internal/canonicaljson` | `cliresult` | `test` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `clonebundle` | `prod` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `clonebundle` | `prod` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonebundle` | `prod` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonebundle` | `prod` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `clonebundle` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `clonebundle` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `clonebundle` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonebundle` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonebundle` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `clonebundle` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `clonebundle` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonebundle` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonebundle` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `cloneproject` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `cloneproject` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `cloneproject` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `cloneproject` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `cloneproject` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `clonesnap` | `prod` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `clonesnap` | `prod` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonesnap` | `prod` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `clonesnap` | `prod` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `clonesnap` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `config` | `prod` | `CanonicalByteLength` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `config` | `test` | `CanonicalByteLength` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `config` | `test` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `crashgate` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `crashgate` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `crashgate` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `crashgate` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `dirnode` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `environ` | `xtest` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `environ` | `xtest` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `environ` | `xtest` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `environ` | `xtest` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `environ` | `xtest` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `fencing` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `fencing` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `fencing` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `fencing` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `fencing` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `fencing` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `fencing` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `fencing` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `matjournal` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `matjournal` | `test` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `merkleinventory` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `merkleinventory` | `prod` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `merkleinventory` | `prod` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `merkleinventory` | `prod` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `merkleinventory` | `prod` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `merkleinventory` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `merkleinventory` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `merkleinventory` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `merkleinventory` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `merkleinventory` | `test` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `merkleinventory` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `merkleinventory` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `merkleinventory` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `merkleinventory` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `provhost` | `prod` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `provhost` | `prod` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `prod` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `prod` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `provhost` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `provhost` | `prod` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `provhost` | `prod` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `prod` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `prod` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `provhost` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `provhost` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `provhost` | `test` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `provhost` | `test` | `SelfRecordID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `provhost` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `provhost` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `provhost` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `rpcwire` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `rpcwire` | `xtest` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `sessadapter` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `sessadapter` | `test` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `sessckpt` | `prod` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessckpt` | `prod` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessckpt` | `prod` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessckpt` | `prod` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessckpt` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `sessckpt` | `prod` | `ErrInvalidIdentity` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessckpt` | `prod` | `SelfCheckpointID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessckpt` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessckpt` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessckpt` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessckpt` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessckpt` | `test` | `ErrInvalidIdentity` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessckpt` | `test` | `SelfCheckpointID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessprofile` | `prod` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessprofile` | `prod` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessprofile` | `prod` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessprofile` | `prod` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessprofile` | `prod` | `SelfEventID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessprofile` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessprofile` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessprofile` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessprofile` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessprofile` | `test` | `SelfEventID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessprofile` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessprofile` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessprofile` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessprofile` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessquery` | `prod` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `sessquery` | `prod` | `SelfCheckpointID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessquery` | `prod` | `SelfRecordID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessquery` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessquery` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessquery` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessquery` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessquery` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessquery` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessquery` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessquery` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessrepo` | `prod` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessrepo` | `prod` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `prod` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `prod` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessrepo` | `prod` | `SelfEventID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessrepo` | `prod` | `SelfField` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessrepo` | `prod` | `SelfRecordID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessrepo` | `prod` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessrepo` | `prod` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `prod` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `prod` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessrepo` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessrepo` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessrepo` | `test` | `SelfRecordID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `sessrepo` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessrepo` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessrepo` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessstate` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessstate` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessstate` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessstate` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `sessstate` | `test` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `sessstate` | `test` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessstate` | `test` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `sessstate` | `test` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `termbind` | `test` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `termbind` | `test` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `termbind` | `test` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `termbind` | `test` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `CalculateObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `CalculateObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `CalculateObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `CalculateObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `Canonicalize` | general RFC 8785 canonical JSON input | unchanged |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `ErrInvalidIdentity` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `ErrInvalidJSON` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `SelfEvidenceID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `SelfManifestID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `SelfProbeID` | record-specific self-ID helper or exported error sentinel | unchanged; helper/error API only |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `VerifyObjectIdentity` | previously admitted identity schemas | unchanged; existing rules retain the same validators |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `VerifyObjectIdentity` | Tombstone@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `VerifyObjectIdentity` | Tombstone Acknowledgement@1.0.0 | fail-closed → admitted by Section 10.7 shape and self-digest validation |
| `internal/canonicaljson` | `terminalbackend` | `xtest` | `VerifyObjectIdentity` | session-record@3.1.0; materialization-plan@1.0.0/@2.0.0; task-board-bundle@1.0.0 | fail-closed → fail-closed |
| `internal/rpcwire` | `hostchannel` | `prod` | `ContractProfile` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `DecodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `DecodeResponse` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `EncodeFailure` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `EncodeRejection` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `EncodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `EncodeSuccess` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `ErrFrame` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `ErrVersion` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `Hello` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `MaxLineBytes` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `MinObjectBytes` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `NewNonce` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `Request` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `prod` | `Response` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `ContractProfile` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `DecodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `DecodeResponse` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `EncodeFailure` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `EncodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `EncodeSuccess` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `Hello` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `MaxLineBytes` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `MinObjectBytes` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `Namespaces` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `NewNonce` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `Request` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `hostchannel` | `xtest` | `Root` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `merkleinventory` | `prod` | `DecodeRequest` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `prod` | `DecodeResponse` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `prod` | `EncodeRequest` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `prod` | `EncodeSuccess` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `prod` | `ErrFrame` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `prod` | `MaxLineBytes` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `prod` | `Namespaces` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `test` | `DecodeRequest` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `test` | `DecodeResponse` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `test` | `EncodeRequest` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `test` | `ErrVersion` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `merkleinventory` | `test` | `MaxLineBytes` | RPC 2 inventory.roots / inventory.children / objects.get frames and bodies | new consumer; request/response shape validation remains in rpcwire |
| `internal/rpcwire` | `meshneg` | `prod` | `ContractProfile` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `prod` | `Hello` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `prod` | `Namespaces` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `xtest` | `ContractProfile` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `xtest` | `DecodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `xtest` | `EncodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `xtest` | `ErrHello` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `xtest` | `Hello` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `meshneg` | `xtest` | `NewNonce` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `ContractProfile` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `DecodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `DecodeResponse` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `EncodeFailure` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `EncodeRejection` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `EncodeRequest` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `EncodeSuccess` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `ErrCorrelation` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `ErrFrame` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `ErrHello` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `ErrInventory` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `ErrVersion` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `MaxLineBytes` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `Namespaces` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `NewNonce` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `OfferedLimits` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `Protocol` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `Request` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/rpcwire` | `rpcwire` | `xtest` | `Response` | existing RPC 2 request, response, hello, namespace, and frame inputs used by this importer | unchanged; rpcwire source and validators are unmodified |
| `internal/provhost` | `axpane` | `prod` | `BuildTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `prod` | `CheckResumeTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `prod` | `DiscoveryContext` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `prod` | `ResolveMapping` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `prod` | `VerifyIdentityBuild` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `prod` | `VerifyIdentityDiscovery` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `test` | `BuildTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `test` | `CreateIdentity` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `test` | `DiscoveryContext` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `axpane` | `test` | `IdentityParams` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `environ` | `xtest` | `CheckIdentity` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `environ` | `xtest` | `DecodeManifest` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `environ` | `xtest` | `DecodeSpawnPlan` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `environ` | `xtest` | `ProfileMapping` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `environ` | `xtest` | `ProfileYOLO` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `BuildTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `CheckResumeTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `DecodeQuiesceProof` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `DecodeSpawnPlan` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `DiscoveryContext` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `Host` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `IdentifyOutcome` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `OpIdentifySession` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `OpProbe` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `OpResume` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `Operation` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `ProbeBuild` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `Request` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `ResolveMapping` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `Response` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `SplitIdentifyResult` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `StoreRootFor` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `prod` | `VerifyIdentityDiscovery` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `BuildTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `CheckResumeTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `CreateIdentity` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `ExecRunner` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `Host` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `IdentityParams` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `ProfileMapping` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `ProtocolID` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `ProtocolVersion` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `ResolveMapping` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `Result` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `StoreRootFor` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `resumesmoke` | `test` | `VerifyIdentityDiscovery` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `sessprofile` | `test` | `BuildTuple` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `sessprofile` | `test` | `ResolveMapping` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |
| `internal/provhost` | `sessquery` | `prod` | `Capabilities` | existing provider identity tuple, discovery, resume, or test fixture inputs named by the entry | unchanged; no provhost production API behavior changed |

## Moved classes and interpretation

- `canonicaljson.CalculateObjectIdentity` and `canonicaljson.VerifyObjectIdentity` now admit complete Section 10.7 Tombstone and Tombstone Acknowledgement v1.0.0 records after closed-shape, coupling, and content-digest checks. Prior behavior failed closed. Existing identity schemas are unchanged; four other mapped classes remain fail-closed as listed in each identity row.
- In the earlier rev3 comparison, `internal/merkleinventory` was the only new direct importer of `canonicaljson`, `rpcwire`, and `provhost`. The final fork-base comparison adds `merkleinventory` callers for canonicaljson/rpcwire and a separate `tmuxserver` test caller for `provhost`; the exact current counts and channels are in the final section.
- `internal/rpcwire` production code and validators are unchanged. Existing `hostchannel`, `meshneg`, and package external tests retain their baseline edges and are labeled unchanged above.
- `internal/provhost` production behavior is unchanged. Its identity ownership test recognizes exactly `internal/merkleinventory/index.go` as another identity verifier call site; broadening that prefix is killed by `TestAttestationAllowlistAnchorsOwningPath`.
- Package-entry usage is source-enumerated from `go list` production/test file inventories. The grid reports changed behavior at the canonicaljson identity entry and the new importer boundary; it does not claim end-to-end session or durable-state semantics.

## TASK-260830-147hsj current Story-checkpoint importer grid

This section supersedes the earlier direct-importer summary for the current
Story checkpoint and candidate. Baseline is a fresh `git archive` of
`b81258e3c5bc0f6321ae8f6bea81ec24cc298d70`; candidate is the uncommitted Story
worktree. Both graphs were generated with Go 1.25.5 on darwin/arm64 using
`go list -json ./...`: 45 package paths in each graph. The baseline and
candidate snapshots, scanner output, and exact test logs are in
`.temp/TASK-260830-147hsj/`.

### Direct importer edges

| Changed package | Base incoming importers | Candidate incoming importers | Edge delta |
| --- | --- | --- | --- |
| `internal/merkleinventory` | none | none | No other package imports this library package. |
| `internal/sessquery` | `sessckpt` tests; `termbind` tests | `sessckpt` tests; `termbind` tests; `merkleinventory` production | Added production edge `merkleinventory → sessquery`. |

`merkleinventory`'s direct internal imports are base `[canonicaljson,rpcwire,scalar]`
and candidate `[canonicaljson,rpcwire,scalar,sessquery,sessrepo,sessstate]`.
The new production dependency surface is limited to validated lease-head
derivation (`sessquery`), exact authoritative record/event reads (`sessrepo`),
and the existing projection reducer (`sessstate`). The other candidate-only
imports in `go list` are standard-library filesystem/persistence support.

### `(imported package, importer, entry, input)` outcome grid

| Imported package | Importer package/channel | Entry | Input class | Base → candidate outcome | Validation |
| --- | --- | --- | --- | --- | --- |
| `internal/merkleinventory` | package itself / `prod` | `DurableIndex.AddJSON`, `SyncFrom`, `RebuildProjection` | schema-valid immutable JSON, common/missing IDs, byte-conflict variant, competing lease/event union | process-local inventory only → durable JSON union, quarantine/abort, repository-bound projection rebuild | Base and candidate package-test commands exit 0; exact logs below. |
| `internal/sessquery` | `internal/merkleinventory` / `prod` | `Reader.LeaseHeadsForSession` | every validated Lease Record; competing IDs under reversed `created_at` values; generated set sizes 0–32; duplicate lease UUID with conflicting bytes | no caller → complete sorted tuple set without winner selection; conflict refuses literal `integrity_failure` | `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation`, `TestLeaseHeadsForSessionCoversGeneratedCardinalityRange`, `TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID`; tuple omission, timestamp-winner, generated-size-17 omission, and same-ID conflict narrowings are each killed by their named test alone. |
| `internal/sessrepo` | `internal/merkleinventory` / `prod` | `GetRecord`, `ListEvents`, `GetEvent` | fixed authoritative record and chained event bytes, including exact byte comparison with union copies | no caller → authoritative bytes are required in the union before projection | `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority`, `TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion`; independent record and event omission narrowings are killed alone. |
| `internal/sessstate` | `internal/merkleinventory` / `prod` | `Projector.Project` | fixed repository chain plus complete post-union lease tuple set | no caller → same literal projection for all 1,440 permutation/timestamp cases; losing event bytes stay outside reducer input | `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority`; `projection-drops-union-head` killed by the named test alone. |
| `internal/sessquery` | `internal/sessckpt` / `test` | existing imported selector/session query entries | checkpoint capture fixtures and ordinary selected session state | unchanged | Base/candidate `go test ./internal/sessckpt ./internal/termbind -count=1` both exit 0; no `sessckpt` source/test row moved. |
| `internal/sessquery` | `internal/termbind` / `test` | existing imported selector/session query entries | terminal binding and session-resolve fixtures | unchanged | Same baseline/candidate importer test command exits 0; no `termbind` source/test row moved. |

The changed code files are `merkleinventory/{index.go,durable.go}` and
`sessquery/lease.go`; tests add crash/restart, union-order, and carry-over
coverage. Moved classes are: process-local JSON identities gain durable
replay/reopen; same-ID different-byte inputs enter persistent quarantine and
abort sync; Tombstone/Acknowledgement links close after union while both remain
records; lease records gain a complete tuple-set handoff; losing event objects
remain stored but are not applied to the repository-owned chain. No
schema-identity class moves, and no existing `sessckpt`/`termbind` behavior
moves.

### Source identity and run results

| Side | Source | Importer package test command | Exit | Result |
| --- | --- | --- | ---: | --- |
| Base | `b81258e3c5bc0f6321ae8f6bea81ec24cc298d70` archive | `go test -p=1 -parallel=1 ./internal/merkleinventory ./internal/sessquery -count=1` | 0 | both package tests green |
| Base | same archive | `go test ./internal/sessckpt ./internal/termbind -count=1` | 0 | both direct test importers green |
| Candidate | uncommitted Story worktree after final code/tests | `go test ./internal/merkleinventory ./internal/sessquery -count=1` | 0 | final changed-package run; `.temp/TASK-260830-147hsj/package-tests-final.log` |
| Candidate | same final source identity | `go test ./internal/sessckpt ./internal/termbind -count=1` | 0 | both direct test importers green; `.temp/TASK-260830-147hsj/importer-tests-candidate-final.log` |

The checkpoint importer packages remain unchanged; their base run is
`.temp/TASK-260830-147hsj/importer-tests-base-01.log`. The final candidate
package run includes all quarantine crash points, the unclosed and mismatched
Tombstone/Acknowledgement refusals, corrupt-active reopen, projection authority
guards, and the generated lease cardinality test. The full configured suite
will establish the repository-wide candidate identity before handoff.

## TASK-260830-2h5uv9 final Story leaf importer grid

The comparison base is the Story fork commit `0ca3e4c26e2b275212796657f785b9b450f6174e`, not a Story checkpoint. The candidate is checkpoint `86a6188a0e35fd04a24eba62a01b71a54d6faa5d` plus this uncommitted leaf. Fresh Go 1.25.5 `go list -json ./...` snapshots were captured on darwin/arm64 with `GOWORK=off` from a clean `git archive` of the fork base and the exact candidate.

| Graph measure | Fork base | Exact candidate | Delta |
| --- | ---: | ---: | ---: |
| Package paths | 44 | 46 | +2 / −0 |
| `Imports` + `TestImports` + `XTestImports` references | 1,468 | 1,592 | +124 / −0 |
| In-repository internal-package references | 317 | 350 | +33 / −0 |
| Unique direct importers of `internal/canonicaljson` | 22 | 23 | +1 |
| Unique direct importers of `internal/rpcwire` | 3 | 4 | +1 |
| Unique direct importers of `internal/provhost` | 5 | 6 | +1 |
| Package union directly importing those three targets | 25 | 27 | +2 |

The only new package paths are `internal/merkleinventory` and `internal/tmuxserver`; there are no removed package paths or import edges. The new target importer edges are `merkleinventory` production and test imports of canonicaljson/rpcwire, and a `tmuxserver` test import of `provhost`. The rev5 test-only skip census also adds `go/ast`, `go/parser`, and `go/token` imports to `merkleinventory`; repository-tree hygiene adds no production dependency. The complete in-repository edge delta and package lists are in the task's `graph-diff-current.md` evidence.

### `(imported package, importer, entry, input)` outcome grid

| Imported package | Importer package/channel | Entry | Input class | Fork base → exact candidate | Executed evidence |
| --- | --- | --- | --- | --- | --- |
| `internal/canonicaljson` | `merkleinventory` / prod | `ClassifyJSON` → `VerifyObjectIdentity` | complete canonical immutable object bytes; Record, Event, Lease, Checkpoint, Descriptor, Tombstone, and Tombstone Acknowledgement identities; unknown and owner-incomplete schemas | no M2 inventory caller → production classification delegates canonical identity and complete §10.7 shape/digest validation; incomplete mapped classes stay fail-closed | `TestRPC2SchemaMembershipTableIsTotalAndDisjoint`, `TestDurableSyncGeneratedPerturbationProduct`, `TestDurableSyncSameDigestDifferentBytesIsTypedConflict`, `TestDurableSyncConflictWithPartialOverlapPeer`; `descriptor-namespace`, unsupported-schema row plants, and four partial-overlap audit narrowings run individually |
| `internal/canonicaljson` | `merkleinventory` / test | `CalculateObjectIdentity`, `VerifyObjectIdentity`, `Canonicalize` | generated valid object fixtures, paired same-identity/different-byte candidates with an additional peer-only identity, plus typed refusal fixtures | no inventory fixtures/calls → tests construct byte-accurate fixtures using the production identity entry; no canonicaljson rule is forked | `TestDurableSyncGeneratedPerturbationProduct`, `TestDurableSyncSameDigestDifferentBytesIsTypedConflict`, `TestDurableSyncConflictWithPartialOverlapPeer`, identity mismatch and unsupported-schema tests |
| `internal/rpcwire` | `merkleinventory` / prod | `DecodeRequest`, `EncodeRequest`, `EncodeSuccess`, `DecodeResponse` | exact RPC 2 request/success/failure envelopes for `inventory.roots`, `inventory.children`, and `objects.get`; returned singleton WireObject response | no inventory client/server caller → typed request framing, strict envelope handling and object-body processing through production dispatch/fetch | `TestDispatchRejectsEveryNonExactOuterRequestEnvelopeShape`, `TestFetchObjectsRejectsEveryNonExactSuccessWireShape`, `TestFetchObjectsRefusesWellFormedRPCFailureEnvelope`; `rpcwire` remains the envelope owner |
| `internal/rpcwire` | `merkleinventory` / test | exported request/response codecs used by fixtures | canonical bodies, missing/extra/duplicate/casefolded members, wrong-kind members, response cardinality and failure envelopes | no inventory protocol fixtures → hostile and valid request/response variants exercise each `Dispatch`/`FetchObjects` route | `TestDispatchRejectsEveryNonExactOperationBodyShape`, `TestFetchObjectsRejectsEveryNonExactSuccessWireShape`, `TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality` |
| `internal/scalar` | `merkleinventory` / prod,test | `scalar.Uint53` via `strictJSON` | AX JSON integer model, exact bounds, floats and out-of-range integer refusal | no inventory decoder caller → every numeric read delegates to the landed scalar boundary; floating point and rounding are not reimplemented | `TestStrictJSONRejectsStructDestinations`, `TestDispatchRequestShapeStrictTypes`, `TestObjectsGetRefusesRequestIDCountOutsideBound`; package scalar suite is green in the exact candidate full run |
| `internal/sessquery` | `merkleinventory` / prod | `Reader.LeaseHeadsForSession` | all validated Lease tuples, losing-lease branch, same-epoch tie, both opposing `created_at` orderings, generated cardinality | no durable inventory caller → complete ordered tuple set is consumed after union without timestamp winner selection | `TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation`, `TestLeaseHeadsForSessionCoversGeneratedCardinalityRange`, `TestDurableSyncClockSkewDoesNotSelectLeaseWinner`; `union-lease-tuple-omission` and timestamp/cardinality plants are killed alone |
| `internal/sessrepo` | `merkleinventory` / prod,test | `Open`, `CreateSession`, `GetRecord`, `ListEvents`, `GetEvent` | repository-owned record and event bytes required to rebuild a projection | no durable inventory caller → repository authority is read after union; `SyncFrom` itself receives no repository mutation handle | `TestDurableSyncGeneratedPerturbationProduct`, `TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion`, `TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority`; record/event omission plants are killed alone |
| `internal/sessstate` | `merkleinventory` / prod,test | `Projector.Project` | authoritative event chain with complete competing lease tuple set and preserved losing branch | no durable inventory caller → rebuild produces the reference projection after union | independent test oracle in `TestDurableSyncGeneratedPerturbationProduct`; `projection-drops-union-head` and lease winner plants fail their named tests |
| `internal/provhost` | `tmuxserver` / test, `!windows` | `CreateIdentity` | ordinary session identity fixture: session UUID, provider/build metadata, creation timestamp | no `tmuxserver` package at fork base → a test-only fixture caller exists in candidate; no production `provhost` caller was added | Exact candidate full suite runs `internal/tmuxserver`; the `CreateIdentity` fixture is not claimed as anti-entropy behavior |

The fork-base importer test set contains 25 packages and ran in one serialized process with exit 0 (8,967 passed test events). The exact candidate configured suite is rerun for this revision; it includes all 27 direct target-importer packages. The generated production-sync product is executed in bounded N/order shards over 38,000 valid-union and 37,765 conflict cases. See `TASK-260830-2h5uv9_results.md` and `generated-property-shards.md` for exact commands/log identities and exits.

The complete base-to-candidate module graph also contains trunk changes outside this leaf: `internal/termbind` adds production `golang.org/x/sys/unix` and `sort` imports and test `bufio`, `path/filepath`, and `runtime` imports. These do not import canonicaljson, rpcwire, or provhost and do not alter the anti-entropy acceptance surface; they are listed separately in the graph evidence rather than attributed to this leaf. The importer API/input grid above makes no universal semantic-equivalence claim for inputs its named tests do not exercise.

Moved behavior classes are limited to the declared task boundary: the durable sync entry validates and unions schema-valid JSON immutable objects, fills delayed gaps, quarantines/aborts same-ID different bytes even when the peer also has missing identities or after a successful prior sync, converges partial/duplicate/order/skew perturbations, and rebuilds the reference projection without timestamp winner selection. The rev4 change adds tests and conformance evidence; it does not edit production code or alter internal importer edges. Tombstone lifecycle effects, raw-blob transfer/materialization, lease issuance/takeover, and retention remain out of contract and are not advertised.
