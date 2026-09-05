# TASK-260830-32jeti round-5 rework results

Round-4 verdict: changes requested (106 mutants, 76 killed, 28 real survivors + 2 stated bounds).
This rework closes all 28 plus a MA8 bonus. Production untouched except one comment restatement.
Uncommitted per the CR shape; rework lives in the working tree (`internal/provhost`, `LOGBOOK.md`).

## F1 — 8 closed vocabularies derived from specdoc (blocking)

New `internal/provhost/closed_vocabularies_test.go`: each of the 8 enums is re-read from the
pinned SPEC.md and asserted equal to the implementation, then the exact widened member from
the verdict battery is refused through the production entry point:

| Enum | Spec source | Planted member | Derivation | Refusal row |
| --- | --- | --- | --- | --- |
| probe statuses | 7.4 sentence (SPEC.md:2869-2870) | partial | killed | killed |
| probe evidence | 7.4 sentence (:2872-2874) | assumed | killed | killed |
| probe architectures | 7.4 prose + 7.5 probe row cross-check | 386 | killed x2 | killed |
| probe platforms | 7.5 probe request row (:3073), sorted both sides | freebsd | killed | killed |
| quiesce blockers | 7.5 enum sentence (:2907-2910) | other | killed | killed |
| identity kinds | 5.5 table row (:2086) | legacy_alias | killed | killed |
| identify confidences | 7.5 identify-session row (:3075) | guess | killed | killed |
| matched_evidence | 7.5 identify-session row (:3075) | guessed | killed | killed |

## F2 — positional blind spots

- manifest capability last substituted, operations last substituted, platform first unknown (aix, still sorted-unique): all killed (MA1r, MA4r, MA14r).
- probe missing first key (native_resume) and missing last key (native_goal_binding): both killed (P3r, P3s).

## F3 — NUL injection

- argv element and env-literal value fixtures via JSON u0000 escapes: both killed (SP4, SP5).

## F4 — equal-length provider pair

- claude (6) vs gemini (6) with a length-equality guard: killed (ID14).

## F5 — ten upper bounds

- quiesce/probe provider_version 129, manifest provider-id 33 (+32 acceptance, +digit-first row killing MA8),
  opaque key 65 (+64 acceptance), opaque digit-first key, version_range 257 (+256 acceptance),
  native_session_id 513 (+512 acceptance), argv 129 elements, argv total 65537 (+65536 acceptance),
  env-name 129: all killed (Q14, P10, MA7, ID9, ID10, ID15, ID16, SP6, SP8, SP12).
- Lesson: the first SP8 draft totalled 69,632 bytes and survived; bound+1 must be exactly bound+1.

## F6 — comments and unmeasured claims

- `TestMutationOperationsReturnsACopy` + `TestCapabilitiesReturnsACopy`: both killed (K5, K6).
- quiesce safe-rule comment restated (backgroundNull folded into !background, unreachable not untested).
- `TestCapabilityGatePrecedesCall` vacuous spawned assertion removed; zero-process half restated as structural.

## Gates (all run directly, real exit codes)

- `go build ./...` exit 0; `go vet ./...` exit 0; `GOOS=windows go vet ./...` exit 0; `gofmt -l` empty.
- `go test ./... -count=1` exit 0 (15 packages); provhost `-race` exit 0.
- cover: provhost 86.5%, provider 97.0%. tracecheck exit 0, unchanged 17/403.
- Mutant replay with the reviewer harness: 29 killed, 0 survived, 0 not-applied.
