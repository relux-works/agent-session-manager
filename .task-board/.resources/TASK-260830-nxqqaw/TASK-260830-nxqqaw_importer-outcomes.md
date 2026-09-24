# Direct importer outcome grid

Baseline package graph: `.temp/TASK-260830-nxqqaw/importer/go-list-base-0ca3e4c.json` (fresh `git archive` of checkpoint `0ca3e4c26e2b275212796657f785b9b450f6174e`, 44 paths).
Candidate graph: `.temp/TASK-260830-nxqqaw/importer/go-list-candidate-final.json` (rev3 candidate on the Story worktree, 45 paths; regenerated after the final tests were added).
Both graphs were generated with Go 1.25.5 on darwin/arm64. The JSON streams contain one `go list -json ./...` object per package; importer edges include production, internal-test, and external-test imports. Rev3 adds no direct import edge beyond the established candidate graph. The unchanged base checkpoint and exact rev3 candidate importer set were tested serialized as `go test -json -p=1 -parallel=1 -count=1`: all 26/26 base paths and 27/27 candidate paths passed, with 21,792 and 22,854 passing test events respectively (14 skipped on each side). Raw logs are `.temp/TASK-260830-nxqqaw/validation/rev3/importer-base.json` and `importer-candidate-final.json`; their metadata records command, working directory, package count, and exit 0.

The first overlapping base/candidate invocation exited 1 on both sides after nested Go subprocesses emitted `Test I/O incomplete` / `WaitDelay expired before I/O complete`. The serialized reruns above are the accepted importer comparison. The failed raw runs remain under `.temp/TASK-260830-nxqqaw/validation/importer-base-tests-0ca3e4c.json` and `importer-candidate-tests-0ca3e4c.json` as diagnostics.

## Importer graph deltas

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
- `internal/merkleinventory` is the only new direct importer of `canonicaljson`, `rpcwire`, and `provhost`. It uses `rpcwire.DecodeRequest` / `EncodeSuccess` for serving, and its production identity path is `Index.AddJSON` → `ClassifyJSON` → `canonicaljson.VerifyObjectIdentity`.
- `internal/rpcwire` production code and validators are unchanged. Existing `hostchannel`, `meshneg`, and package external tests retain their baseline edges and are labeled unchanged above.
- `internal/provhost` production behavior is unchanged. Its identity ownership test recognizes exactly `internal/merkleinventory/index.go` as another identity verifier call site; broadening that prefix is killed by `TestAttestationAllowlistAnchorsOwningPath`.
- Package-entry usage is source-enumerated from `go list` production/test file inventories. The grid reports changed behavior at the canonicaljson identity entry and the new importer boundary; it does not claim end-to-end session or durable-state semantics.
