# TASK-260830-219okr results — implement-dual-stack-major-negotiation

Role: developer. Status: ready for review (candidate UNCOMMITTED in the Story worktree).

This is the rework-rev4 run (RUN-260917-8fb798). Revision 3 was reviewed by
RUN-260917-92b548 (claude-opus-5 max) and routed CHANGES REQUESTED: three P2
and three P3, no P1, all evidence and documentation — the reviewer states
plainly that the production selection logic is right and no production
selection logic needed to change for F1 or F2. This run answers every finding
explicitly; production `negotiate.go` selection logic is unchanged (doc
comments only), base `888ae3d` is current `origin/main` (no refresh needed).

## Finding-by-finding table (rev3 review → this rework)

| Finding | Change | Test or plant that fails without it | Evidence path |
| --- | --- | --- | --- |
| F1 (P2): exposure tokens pinned circularly (expectations derived from `meshneg.DirectoryUnsupportedCode` / `meshneg.BackendEvidenceUnsupportedCode`; reviewer swaps R19/R20 SURVIVED) | `TestPeerExposure` and `TestRefusalClassesResolveFromTheRegistry` now assert the LITERAL tokens `directory_mesh_unsupported` / `terminal_backend_unavailable` at the `Negotiate` entry; no committed test references the production constants | New plants `exposure-directory-code-swap` (→ `continuation_route_unavailable`, same 1.2.0/exit 6) and `exposure-backend-code-swap` (→ `terminal_backend_capability_unproven`, same 1.3.0/exit 6): both KILLED by `TestPeerExposure/2.0.0` (backend swap also by `/3.0.0`) | `mutations-meshneg/exposure-directory-code-swap/test.log`, `mutations-meshneg/exposure-backend-code-swap/test.log`, `mutations-meshneg/results.json` |
| F2 (P2): Section 6.6 unknown-generation refusal pinned only at the `LocalMajors` helper; reviewer plant R12 (Negotiate-site fallback to `[2 3 4]`) SURVIVED | New `TestNegotiateUnknownGeneration`: 8 unknown generations (`""`, `"5.0.0"`, `"9.9.9"`, `"4.0.1"`, `"3.0"`, `"v4.0.0"`, `"4"`, `"1.0.0 "`) × all 4 pinned frames through `Negotiate`, asserting `invalid_config`/3, `ReasonUnknownConfig`, `ErrUnknownConfig`, and a zero `Decision{}` | New plant `negotiate-unknown-falls-back` (R12 shape at the Negotiate site): KILLED by `TestNegotiateUnknownGeneration` (all 32 subtests fail) | `mutations-meshneg/negotiate-unknown-falls-back/test.log`, `mutations-meshneg/results.json` |
| F3 (P2): docs named `config.Configuration.SchemaVersion`, which the config owner normalizes to CurrentVersion for every v1/v2/v3 source | `LocalMajors` and `Negotiate` doc comments now name `config.LoadedConfiguration.SourceVersion` with the normalization warning | New `TestLocalMajorsTracksConfigVocabulary`: pins the mapping against `config.Version1/Version2/CurrentVersion/Version4`, decodes minimal v1/v2/v3 docs proving `SourceVersion` carries the generation while `Value.SchemaVersion` normalizes, and proves the wiring trap (SchemaVersion input yields the wrong majors for v1/v2) | `leaf/meshneg-verbose.log` (`TestLocalMajorsTracksConfigVocabulary` PASS) |
| F4 (P3): `PeerOffer` re-validates keys + `rpc` only; comment over-claimed full re-validation | Chose the stated-bound option: doc comment, package README, and TRACEABILITY now state that non-`rpc` array values are validated by rpcwire decode and trusted here, and direct callers must pass decode-produced hellos; `directHello` comment fixed | New `TestPeerOfferTrustsDecodedNonRPCArrays` pins both halves: tampered v3/v4/v5 `session_event` admitted by `PeerOffer` but refused by rpcwire encode (`ErrHello`); v2 `lease=["9.9.9"]` admitted by BOTH (valid 1-16 range by design — the reviewer's v2 probe half describes valid input, not a hole; verified by probe, see results prose) | `leaf/meshneg-verbose.log` (`TestPeerOfferTrustsDecodedNonRPCArrays` PASS) |
| F5 (P3): frame gate measured only at 2.x/3.x edges; reviewer plant R8 (admit `5.1.0`) SURVIVED; unparseable-version class differs from 15.1 | `TestPeerOfferRefusals/frame-versions` gains `4.1.0`, `5.0.1`, `5.1.0` (plus `garbage`) rows; unparseable-class difference (`incompatible_protocol`/6 here vs 15.1 `transport_failure`, unreachable via decode) stated in the doc comment, README, and TRACEABILITY | New plant `frame-admits-5.1`: KILLED by `TestPeerOfferRefusals/frame-versions` | `mutations-meshneg/frame-admits-5.1/test.log`, `mutations-meshneg/results.json` |
| F6 (P3): TRACEABILITY 17.1 wording credited minor selection to `Select` | Minor clause dropped from the covered 17.1 row (within-major minor selection remains a stated bound in the same file) | Wording only; row now reads "No major selected by coercion" | `internal/meshneg/TRACEABILITY.md` diff |

## Deliverable

`internal/meshneg` (`negotiate.go`, selection logic unchanged since rev3): pure-data
major negotiation for Mesh RPC.

- `LocalMajors(configVersion)` — generation to offered majors: Config-1 → `[2]`,
  Config-2 → `[2 3]`, Config-3 → `[2 3 4]`, Config-4 → exactly `[5]`; unknown
  generations refuse `invalid_config`/exit 3 (Section 6.6: unknown configuration is
  a refusal, never legacy selection). Generation input documented as
  `config.LoadedConfiguration.SourceVersion`.
- `PeerOffer(frameVersion, hello)` — validates the decoded peer frame: pinned
  vocabulary 2.0.0–5.0.0, exact per-major contracts keys (a v2 14-key bound inside a
  v3+ frame refuses), single-major strictly sorted-unique semver rpc array agreeing
  with the framing major. Any mixing refuses `incompatible_protocol`/exit 6.
  Non-`rpc` array values are validated by rpcwire decode and trusted here (stated
  bound).
- `Select(local, peer)` — highest common major; inputs outside 2/3/4/5 or disjoint
  pairs refuse `incompatible_protocol`/exit 6.
- `Negotiate(configVersion, frameVersion, hello)` — production entry returning
  `Decision` (major, protocol version, exact contracts map, namespace vocabulary,
  statically bound Structured Error version, peer exposure) or `*Refusal` (pinned
  Section 15 class + reason + offer facts). Frames nothing: Sections 11.2/15.1/11.10.1
  close without a peer frame on unsupported majors; the initiator maps the class
  to its own local error.
- Core-only preservation: `Decision.Peer` reports directory activation as
  `directory_mesh_unsupported` and backend-evidence activation as
  `terminal_backend_unavailable` (registered exit-6 codes; exits resolved via
  `axerror.ExitCodeFor`), never as zero inventory. No count/inventory member exists
  in the view (structurally pinned by test); the literal tokens are asserted at the
  `Negotiate` entry.

## AC coverage: 11 of 11 rows driven

See `TASK-260830-219okr_conformance-matrix.md` rev3 for the row-by-row call sites and
named tests. Every refusal gate has negative tests plus at least one narrowing mutant
(31 narrowing probes, all killed).

## Files changed (UNCOMMITTED, Story worktree only)

- `internal/meshneg/negotiate.go` (doc comments only since rev3: SourceVersion
  input, F4/F5 stated bounds; zero selection-logic change)
- `internal/meshneg/negotiate_test.go`, `internal/meshneg/conformance_test.go`
  (literal token pins, `TestNegotiateUnknownGeneration`,
  `TestLocalMajorsTracksConfigVocabulary`, `TestPeerOfferTrustsDecodedNonRPCArrays`,
  5.x frame edges)
- `internal/meshneg/mutations.py` (4 new narrowing plants)
- `internal/meshneg/README.md`, `internal/meshneg/TRACEABILITY.md` (bounds, counts, credits)
- `README.md` (narrowing row count 27 → 31; paragraph + tool rows otherwise
  unchanged; measured-coverage subsection untouched), `LOGBOOK.md` (newest-first
  REV4 line, additive)
- `internal/traceability` untouched (final leaf owns registry bindings + re-pin)
- `task-board.config.json` == HEAD (base current, no refresh performed)

## Test evidence (all exits real, logs in the evidence tar)

Prior runs (rev1–rev3, RUN-260917-b15a5b / 244f06 / 27bea1): 15/15 package tests,
87.8% coverage, 27 narrowing killed, full 27-command suite green on `888ae3d`
(race gate 38 ok, sessquery 515.663s, no DATA RACE). See the rev3 results
attachment for the full history.

This run (RUN-260917-8fb798) — leaf evidence:

- `go test ./internal/meshneg -count=1 -v` → ok, 18/18 top-level tests pass
  (184 `--- PASS` lines, 0 FAIL; 3 new: `TestNegotiateUnknownGeneration`,
  `TestLocalMajorsTracksConfigVocabulary`, `TestPeerOfferTrustsDecodedNonRPCArrays`)
- `go test ./internal/meshneg -race -count=1 -cover` → ok, 87.8% statements
- Mutants: `PYTHONDONTWRITEBYTECODE=1 python3 internal/meshneg/mutations.py
  --output .temp/TASK-260830-219okr/mutations-meshneg` → exit 0; 31 narrowing
  probes killed by named tests, neutral passed, harmless `detail-text` control
  survived as designed; per-plant raw logs + exits + table archived; no `__pycache__`
- `test -z "$(gofmt -l ...)"` → clean; `go build ./...` → ok; `go vet ./...` → exit 0
- Owner packages `go test ./internal/rpcwire/ ./internal/axerror/ ./internal/hostchannel/ ./internal/config/ -count=1` → ok
- F4 probe note: a scratch test (deleted before handoff) confirmed the v3-tampered
  `session_event` map is admitted by `PeerOffer` (returns 3) but refused by
  `rpcwire.EncodeRequest` (`incompatible RPC hello structure`), while v2
  `lease=["9.9.9"]` is admitted by `PeerOffer` AND the full rpcwire
  encode→decode→Hello path — i.e. valid v2 range input, not a hole. The committed
  `TestPeerOfferTrustsDecodedNonRPCArrays` pins exactly this shape.

This run — full configured 27-command suite (commands 0–26, sequential, logs in
this tar under `validation/cmdN.log` + `cmdN.exit`):

- cmd0 gofmt clean; cmd1 `go build ./...` ok; cmd2 `go vet ./...` exit 0.
- cmd3 `go test ./... -count=1 -v` → exit 0, 38 ok, 21707 `--- PASS` lines,
  0 top-level FAIL.
- cmd4 `go test ./... -race -count=1 -timeout 25m` → exit 0, 38 ok, no DATA
  RACE, no timeout (sessquery 370.955s, meshneg 1.876s).
- cmd5 `go test ./... -cover -count=1` → exit 0, 38 ok, meshneg 87.8%.
- cmds 6–19 (14 fuzz smokes, 100x) → 14 ok, exit 0.
- cmd20 tracecheck → exit 0; cmd21 cataloggen `-adopted -check` → exit 0 with
  the tree untouched (`git status internal/catalog/` clean).
- cmd22/23 linux+windows builds ok; cmd24 JSON census ok; cmd25
  `task-board validate` exit 0 (187 pre-existing MISSING_ACTIVITY warnings, none
  on this task/story); cmd26 `git diff --check` clean.
- Base `888ae3d` == `origin/main` at handoff time: no refresh performed,
  `task-board.config.json` == HEAD, `git status` shows only the 8 candidate paths.
- SPEC v0.7.0 has NOT landed (`internal/specdoc` holds v0.6.0 only), so the
  v0.6.0 citations stand; `internal/traceability` untouched for the final leaf.

## Binding choices and bounds

- RPC-4 exposure equivalent is the registered `terminal_backend_unavailable`
  (exit 6): Section 11.9 requires reporting evidence unsupported rather than empty,
  Section 17.4 requires reporting activation unavailable, and the 1.3.0 exit-6 row
  carries exactly that code. No code minted. (Reviewer-verified defensible; the
  literal token is now asserted at the production entry.)
- Refusal/exposure exits resolve under each code's introducing Error version
  (1.0.0/1.2.0/1.3.0); cross-version exit stability is asserted by test.
- v2 1–16 rpc minor ranges select the major only; within-major minor selection is
  future work (pinned by `admitted-ranges`).
- Non-`rpc` hello array values: validated by rpcwire decode, trusted by `PeerOffer`
  (F4 stated bound, pinned by `TestPeerOfferTrustsDecodedNonRPCArrays`).
- Unparseable frame versions answer `incompatible_protocol`/6; Section 15.1 assigns
  `transport_failure` — unreachable via decode, stated bound (F5).
- Handshake sequencing (hello-first, request_id echo, one framed refusal then close)
  stays with rpcwire decode + the connection responder; admission stays with
  TASK-260830-z1yxg9 (separation pinned by `TestNegotiationIsNotAdmission`).
- No durable writes: direct imports exclude os/io/net/time/rand/exec; owners
  consulted through pure functions only; statelessness pinned by
  `TestNegotiationIsDeterministic`. No crash/idempotency surface.
- No `ax` command, doctor result, or runtime capability is claimed.
- Scope kept: no changed line outside the 8 candidate paths. The CR validation
  suite reruns automatically on handoff; this run executed it in full once
  beforehand (commands 0–26, all exit 0).
