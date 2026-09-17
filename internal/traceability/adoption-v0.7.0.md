# v0.7.0 adoption ownership

Authority: `agent-session-manager-spec@v0.7.0`, commit
`32b3f2ba7c377248a53cd42389abbd2f1c321834`, as verified by
`internal/specpin/v0.7.0.lock.json`. The pinned v0.7.0 section inventory is
identifier-equal to the v0.6.0 one (170 headings, same digest): the revision
adds no numbered section, only appendix subsection A.12 and the unnumbered
`Caller launch-plan code` heading, both excluded by the inventory rule. The
new normative content therefore lands inside already-inventoried sections, and
the new bindings below own clause areas, not new headings. Catalogue census is
not implemented runtime support. Every binding below is refused by
`traceability.VerifyAssignedSections`; story IDs in registry gaps identify
pending delivery owners and do not discharge clauses.

| Sections | Primary pending owner | Responsibility and boundary |
| --- | --- | --- |
| 14.1, 13.1 | STORY-260916-3fjjtn | Launch Plan request document parsing, Section 5.1 validation, planning-role launch step, recorded-argv determinism, persisted-extensions bound, and the launch_plan_invalid / capability_unavailable / invalid_arguments / secret_policy_violation refusals; `-` reads the plan document from ax's own stdin, never the child's |
| 5.1 | STORY-260916-2q85nu | Resolved plan embedded under the ax.launch-plan-request extension key; the four works.relux.curator.* caller keys copied verbatim and treated as opaque; Launch Stdin member and fragment-digest rule |
| 7.5 | STORY-260916-1kp1lx | SpawnPlan.stdin member (null or the closed encoding/bytes object) and the stdin_resume_replay resume semantics; a resume replays no stdin payload by default |
| 7.3, 7.4 | STORY-260916-1ea7od | caller_launch_plan plugin capability: nine-name 1.1.0 manifest/probe registry, verbatim plan translation onto the SpawnPlan, plugin-side profile-flag refusal, base-flag collision rule; 1.0.0 readers keep accepting seven names |
| 13.10 | STORY-260916-3aukw3 | Resume/fork environment drift: profile-pin comparison like with like, refuse by default with details.reason environment_drift when system-modules is true, report otherwise |
| 13.1 (naming) | STORY-260916-3ustb3 | Same-second duplicate session-name refusal at start with a Structured Error; the existing session record stays byte-identical; curator-run default naming is launcher-side |
| Appendix D, fixtures | STORY-260916-vucwn0 | Launch Plan request conformance vectors and per-plugin SpawnPlan goldens bound to fixtures/launch_plan_request_conformance.json, plus the expected-red gate mutants |

The shared Section 13.1 binding names two pending owners with an explicit
split: STORY-260916-3fjjtn owns the caller-plan planning-role step and its
refusals; STORY-260916-3ustb3 owns the name-unused precondition and nothing
else. This split is clause attribution, not shared implementation. The
Section 14.1 binding likewise attributes the ax-side grammar, validation and
refusals to STORY-260916-3fjjtn and the plugin-side profile-flag refusal to
STORY-260916-1ea7od. The Section 7.3/7.4 nine-name registry shape belongs to
STORY-260916-1ea7od while the stdin_resume_replay replay semantics belong to
STORY-260916-1kp1lx. Every action and boundary in the source table remains an
acceptance obligation; later command owners must use these shared APIs.

Refined historical clauses remain in scope: Section 15.3 launch_plan_invalid
prohibition text belongs to STORY-260916-3fjjtn; the Section 7.6/7.7 provhost
operation conformance stays under its existing bindings. STORY-260916-3fjjtn,
STORY-260916-2q85nu, STORY-260916-1kp1lx, STORY-260916-1ea7od,
STORY-260916-3aukw3, STORY-260916-3ustb3 and STORY-260916-vucwn0 are linked to
EPIC-260916-18uelk. No owner is unblocked by this adoption candidate. Resume
follows reviewed, signed Story delivery through the orchestrator.

`catalog.Current` is the 64-contract v0.7.0 census, including Provider 2.1/3.1
vocabulary. Session Record 3.1.0, Error 1.4 and 1.5, Provider manifest/probe
1.1.0 and the Launch Plan request contract are census-only: no new identity
shape, error-code runtime, capability names, readers, or launch-plan runtime
are advertised. Historical
V060/V050/V043 catalogue projections, specpin/specdoc historical readers, the
old ownership registries, and localstore's explicit CurrentV050 binding remain
valid. The 31 acceptance cases and 9 section bindings the five landed v0.6.0
stories added (Sections 2.2, 2.4, 5.3/5.4/5.5, 6.6, 7.7, 8, 10.6, 11.10.1-4,
13.12/13.13, 16.1) are carried with re-measured clause lines and unchanged
owners; the 49 discharged clauses they establish remain 49. The new
denominator is 569.
