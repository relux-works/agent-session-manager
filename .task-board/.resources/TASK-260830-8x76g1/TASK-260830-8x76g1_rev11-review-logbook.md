# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-09-01

### 0530 — Closing an enumerated finding list is not the same as auditing for it
- FINDING: TASK-260830-8x76g1 CR rev 11 pinned the 23 unguarded constraint bounds the rev-10 reviewer had listed by name. A systematic 66-mutant widening sweep over every bound expression in `internal/canonicaljson/closed_shapes.go` found 15 more that can be widened or dropped with the whole repository suite still printing `ok`.
- FINDING: 7 upper bounds (`BlobChunk.size` 4194304 at `closed_shapes.go:610`, `GitIndex.version` lower enum edge at `:1147`, `GitIndexEntry.mode` uint32 at `:1203`, Board Identity `logical_id` at `:404`+pattern `:32`, managed_tree `agent_project_config_paths` at `:925`, `GitSubmodule.repository_identity` at `:1257`, `GitSubmodule.agent_project_config_paths` at `:1343`) and 8 lower bounds were unbacked while their `constraint-enumeration.md` rows all asserted "Enforced exactly as declared".
- FINDING: production refuses all 15 correctly with the exact declared message; every one is reachable through `CalculateObjectIdentity` at 419–6624 bytes against a 5242880-byte cap. Claim-versus-evidence gap only, no product defect.
- DECISION: a reviewer handing back a named list invites the producer to treat that list as the spec of the rework. State the *class* and require the producer to re-run the whole sweep, and hand over the harness so the next cycle is mechanical rather than another hand-enumeration.
- NOTE: a surviving mutant is not automatically a finding. 4 survivors here are masked by a stricter production check that still enforces the declared rule exactly (`BlobChunk.index` behind `index == position`; submodule array caps behind the `*total > 256` forest guard; extension key minimum behind `reverseDNSPattern`; single-side media-type case). Separating those needs multi-site combined mutants, not single-site ones.
- SCOPE: `internal/canonicaljson/closed_shapes.go`, `internal/canonicaljson/boundary_constraints_test.go`, `internal/canonicaljson/testdata/constraint-enumeration.md`
- STATUS: pending — routed to `to-dev` for a third test/doc-only cycle. Evidence: `TASK-260830-8x76g1_review-verdict.md`, `TASK-260830-8x76g1_rev11-mutation-sweep.tar.gz`.
