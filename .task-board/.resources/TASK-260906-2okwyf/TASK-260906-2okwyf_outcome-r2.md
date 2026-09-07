# TASK-260906-2okwyf round 2 — F1 sibling echo closed: outcome

Status: ready for review (handed off to review; review/integration are orchestrator steps).

Candidate tree: `abec4211afb6b401b6614f619365d7ccf07c00e9`
(verified: detached index — `GIT_INDEX_FILE` copy + `read-tree HEAD` +
`add -A` + `write-tree` — worktree index untouched, HEAD stays at the
`114a056` checkpoint, change uncommitted. All 10 touched paths in-tree
with `git hash-object` byte-equal to the tree blob, 10/10 MATCH.
Computed AFTER every tracked write, including the LOGBOOK append. The
Change Request record is the orchestrator's integration step; this OID
is the candidate it must carry.)

Round-2 scope (2 code/test paths + LOGBOOK; round-1 files untouched):

- `internal/terminalbackend/terminalbackend.go` — `CheckVersionTuple`
  validates `backendID` through `ParseID` before any version arm runs;
  `ParseID` comment rewritten (false invariant replaced).
- `internal/terminalbackend/terminalbackend_test.go` — new
  `TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho`.
- `LOGBOOK.md` — this round's entry (newest first).

Round-1 outcome (`TASK-260906-2okwyf_outcome.md`) and battery
(`battery_2okwyf.json`, 9 applied / 9 killed / 0 survived) stand for the
unchanged regions: all 9 round-1 mutant anchors re-grepped on this tree
and each still occurs exactly once (9/9 OK), and the full suite is green
below.

## 1. F1 — hostile backendID at CheckVersionTuple (closed)

Pre-fix reproduction, driven through the production entry point
`CheckVersionTuple` with `200x"A" + ESC[31m + ../../etc/passwd`: all
three arms (`implementation_version semver`, `protocol_version major 1`,
`protocol_version membership` at terminalbackend.go:585/588/595)
rendered a 322-byte refusal containing the whole string — `BackendID`
echo plus `Error()` render. Same class A9 named at `ParseID`, live at
the sibling that never calls it.

Fix: entry validation through `ParseID` (the pattern
`RequireRestoreBinding` and `CheckProviderDescriptor` already use). A
refused ID returns the `ParseID` refusal — whose bound and grammar arms
carry no `BackendID` — and never reaches the version arms. No new refuse
line is added, so the refusal-arm derivation is byte-identical
(census-neutral by construction; verified by the battery's census
SURVIVED rows).

Test: `TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho` drives
the production entry with 3 arms x 2 hostile shapes (221-byte
over-bound, 27-byte grammar-hostile) asserting `BackendID == ""` and
`!strings.Contains(Error(), hostile)`, plus two controls: a well-formed
ID still reaches the version arms with attribution intact, and `ax.evil`
still names its validated identity through the reserved arm.

- Red-before (unfixed code): 6/6 hostile subtests FAIL (both the
  `BackendID` and the `Error()` assertions).
- Green-after: 7/7 PASS, exit 0.

## 2. Comment now describes the code (F1 second demand)

The old `ParseID` comment claimed the reserved arm names "a validated
identity like every other BackendID in the package" while 3 of 31 sites
were unvalidated. The rewritten comment enumerates the mechanism
per site instead of asserting a blanket invariant: `mustParseID` at
`Registration.validate` / `TrustEntry.validate`; `ParseID` at the
`Resolve` / `RequireRestoreBinding` / `CheckVersionTuple` /
`CheckProviderDescriptor` entries and at `ParseManifest` / `ParseProbe` /
`ParseEvidence` document admission; constants and already-validated
records at `New` / `RegisterExternal`.

Census (mechanical `grep BackendID:` over non-test package sources,
each site read for its guard):

| sites | guard | status |
|---|---|---|
| terminalbackend.go:176 reserved arm | input already passed grammar | validated |
| terminalbackend.go:247,253,257,260 record arms | `mustParseID(record.ID)` at :240 | validated |
| terminalbackend.go:269,273,276 protocol arms | callers pass validated ID / `BuiltinTmux` const | validated |
| terminalbackend.go:398,401,404 trust arms | `mustParseID(entry.BackendID)` at :394 | validated |
| terminalbackend.go:479,482,485,491 register arms | entry validated :469, observed :481 | validated |
| terminalbackend.go:504,506 duplicate/drift | `observed.validate()` at :481 | validated |
| terminalbackend.go:524 resolve | `ParseID` at :516 | validated |
| terminalbackend.go:579 restore binding | `ParseID` at :570/:574 | validated |
| terminalbackend.go:602,605,612 version tuple | `ParseID` at :598 (THIS FIX) | validated |
| terminalbackend.go:643,646,649,653,656 descriptor | `ParseID` at :641 | validated |
| manifest.go:1726,1762 probe arms | `ParseID` at `ParseProbe` admission (:1150) | validated |
| manifest.go:2084,2087 manifest arms | `ParseID` at `ParseManifest` admission (:1004) + `Resolve` re-validation before the check | validated |

27 + 4 = 31/31 validated, 0 unvalidated. (descriptor.go:80 and
conformance.go:968 are struct constructions, not refusals.)

## 3. Sibling census (F1 "ask the same question") — measured, not changed

Each driven through its production entry point with the same hostile
material (200xA + ESC + traversal, JSON-escaped where the entry takes
JSON). Sibling production code is UNCHANGED: sibling suites pin their
refusal text (e.g. secprim `env_test.go` asserts
`"secprim unsafe environment: env name: 9X"` verbatim), so reshaping
their refusals is another leaf's contract, not this diff's.

| package | entry | observation | verdict |
|---|---|---|---|
| environ | `DecodeEnvironmentObservation`, unknown wire key | 269-byte refusal `environment observation carries unknown member: AAAA…ESC…` — full hostile key in `Error()` | LIVE — stated bound (see below) |
| environ | value refusals (all members) | member NAME only (closed set), values never echo | clean |
| secprim | `BuildEnv`, hostile allowlist name | 259-byte refusal `secprim unsafe environment: env name: AAAA…` — full refused name in `Error()` | LIVE — stated bound (see below) |
| secprim | `CheckMemberPath` | 104-byte refusal, truncated by `memberErrorTarget` (64 chars + `...`), `contains(full)==false` | bounded by construction (deliberate, documented on the function) |
| secprim | `CheckArgv` | `Detail` is the position (`argv[3]`), never element bytes | clean |
| provhost | `DecodeResponse`, hostile `protocol_version` | `Error()` is static text (`contains==false`); version travels in `Details{"member"}`/`{"observed"}` | bound by architecture: `Error()` renders code+message only (axerror.go:285); details are inert data capped at 16KiB canonical (`maxDetailCanonical`, details.go:39) |
| provhost | `DecodeProbe`, hostile unknown key | `Error()` static (`contains==false`); key in `Details{"member"}` (221 chars, equals-hostile) | same bound as above |

Stated bounds (LIVE, out of this diff; recommended follow-up leaves,
not created here — creating board elements is prohibited while the
final leaf is open):

- B1 (environ): `DecodeEnvironmentObservation` unknown-member and
  duplicate-member arms echo the full wire key into `Fault.Error()`.
  Escapes: unbounded wire member names (up to the frame size).
  Owner: environ refusal-text pins in `frame_agreement_test.go` et al.
- B2 (secprim): `BuildEnv` `env name` / `allowlist duplicate` /
  `literal collision` arms echo the full refused name into `Error()`.
  Escapes: unbounded refused names (`IsEnvName` fails open-length
  input, then the input is echoed). Owner: `env_test.go:76,86` pin the
  echo verbatim.

## 4. Mutation battery (round 2, this tree)

Harness: same verdict semantics as round 1 (`KILLED` = mask exit != 0,
`SURVIVED` = exit 0; `go vet` gate first; byte-for-byte restore
verified). Masks verified non-empty by `-list` before running:
census 15 tests, behav 11 top-level (+ subtests), audit full-package.
Full bodies in `battery_2okwyf_r2.json`.

| mutant | class | what it narrows the gate to | named test that fails | verdict / killers |
|---|---|---|---|---|
| N-tb-checktuple-lengthgated: entry check runs only when `len > maxIDBytes` | narrowing | admits exactly the grammar-refused-but-bounded subclass (len27 echoes, len221 still refused) | `TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho/.../len27` (3 rows; len221 rows pass — narrowing proven) | KILLED / behav (census 15 SURVIVED) |
| D-tb-checktuple-entry: entry block deleted | arm-deletion | gate absent; all hostile IDs echo | same test, all 6 hostile rows | KILLED / behav (census SURVIVED — the call is not a refuse line) |
| R-tb-checktuple-order: entry check moved after the version arms | narrowing (order) | validation present but bypassed on every version-failure path | same test, all 6 hostile rows | KILLED / behav (census SURVIVED — same lines, moved) |
| C-tb-checktuple-deadarm: unreachable new arm (`backendID == "never.reached shadow"`, space can never pass the entry) | census-only | — (no behavior change) | census: `TestDerivedRefusalArmsAreAllDeclared`, `TestDeclaredRefusalArmsAreAllDerived`, `TestDerivedSiteLinesAreExactlyRowed` | KILLED / census (behav 243 SURVIVED, empty evidence) |
| S-tb-checkdesc-shadow: digest-arm-1 condition widened to swallow arm-2 inputs (same pair) | audit-only | — (occurrence-2 witness proves its pair through the sibling) | full-run audit: `terminalbackend.go:651` named as derived-without-exercised-path | KILLED / audit (census 15 + behav 243 SURVIVED) |
| X-control-notapplied | control | — | — | NOT_APPLIED (anchor occurs 0 times) |
| X-control-compilefail | control | — | — | COMPILE_FAIL (`go vet` syntax error, provider.go) |

Killed over applied: 5/5 killed, 0 survived. No surviving mutant, so no
survival bounds to state. NOT_APPLIED and COMPILE_FAIL are distinct
control rows, not in the denominator.

## 5. Gates (real exit codes, standalone processes, no pipes)

- `go test ./internal/terminalbackend/ -run TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho` (pre-fix): FAIL, 6/6 hostile subtests red.
- `go test ./internal/terminalbackend/` (post-fix): exit 0.
- `go build ./...`: exit 0. `GOOS=windows go build ./...`: exit 0.
- `gofmt -l internal/`: clean (no files).
- `go vet ./...`: exit 0. `GOOS=windows go vet` (tb/ph/pv): exit 0.
- `go test ./...`: exit 0 — every package ok, 0 FAIL.
- `go test -race ./internal/terminalbackend/`: exit 0.
- `go test -cover` tb/ph/pv: exit 0 (95.1% / 86.0% / 97.8%).

AC coverage this round: 3/3 F1 rows driven through production entry
points — (a) hostile ID refused without echo at all 3 version arms via
`CheckVersionTuple`; (b) comment accuracy via the 31/31 census above;
(c) sibling echo class asked and answered with measurements per package.
Round-1 AC 4/4 stands (anchors re-verified, suites green).
