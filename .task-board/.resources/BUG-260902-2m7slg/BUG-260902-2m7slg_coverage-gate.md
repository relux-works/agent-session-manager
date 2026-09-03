# BUG-260902-2m7slg — Ownership bindings claim their coverage

Revised for the shipped tree at leaf `2d0962c` (tree `163d498`), parent
`fa06e6d` = `origin/main`. Every command output and exit code below was re-run
on this tree by the rework-round-3 run; nothing is carried forward from the
revision-1 description, which was false about the shipped gate in seven ways
and is fully superseded by this document.

## What was wrong

`sourceChecker.verify` proved a Go symbol exists. `verifyOwnershipGroups` proved
an acceptance case is linked. Neither compared the implementation against the
section it claims, so a sliver implementation satisfied a whole-section binding
and the gate reported it as owned. Three slivers had reached three independently
produced branches without anyone noticing.

Revision 1 of this gate then reproduced the same defect through a second door.
Its obligation scanner matched uppercase RFC 2119 keywords only, so a section
that states its obligations as a table measured zero clauses, was bucketed
`declarative`, and `declarative` was *admitted* with no clause evidence at all.
`tracecheck -section 15.2` exited 0 while nothing in the tree implements the
nineteen-row normative exit-code registry. Keyword absence had been read as a
coverage claim. That is closed here.

## What the gate does now

Every `section_binding` declares a `coverage` level. The gate recomputes it
instead of believing it:

* The **denominator** is measured from `internal/specdoc` — the hash-verified
  pinned `SPEC.md`. It is every RFC 2119 clause line (`MUST`, `MUST NOT`,
  `SHALL`, `SHALL NOT`, `REQUIRED`) under the section's own heading *and its
  subheadings*, so a parent heading is measured over its children. `MAY` and
  `SHOULD` are excluded: they create no obligation.
* The **numerator** is the `clauses` the binding enumerates. Each clause must
  index that measured inventory, declare the exact `SPEC.md` line it occupies,
  quote that clause verbatim beginning on that line, carry an RFC 2119
  obligation, and name at least one acceptance case that the binding itself
  owns and that is registered with an executable test.
* The declared level must equal the measured bucket.

| Level | Measured meaning | Admitted as an assigned scope |
| --- | --- | --- |
| `full` | every normative clause of the section is enumerated and discharged | **yes** |
| `partial` | at least half are | no |
| `sliver` | fewer than half are | no |
| `unevidenced` | none are; the registry makes no clause-level claim | no |
| `unmeasured` | the scanner finds no clause line under the section at all | no |

Assigned-scope admission (`tracecheck -section ...`, the Story-scope gate)
requires **`full` and nothing else**. `unmeasured` is deliberately not admitted,
and the alternative — admitting it on a reviewed justification sentence — was
considered and rejected, because the gate cannot verify such a sentence and
unlocking admission with one would be self-minted evidence for exactly the class
this gate exists to refuse.

* Every level below `full`, **`unmeasured` included**, must name a gap. The gap
  must name its section as a whole identifier — a sentence about 6.55 is not a
  gap about 6.5 — and must name the production declaration the binding is
  registered to.
* `unowned_sections` records a section this repository does not implement, with
  a gap and evidence. It may not cover a section the generated catalog requires
  an owner for, so it is a disclosure, never an exemption.
* Every new field is inside the reviewed canonical projection digest
  (`reviewedOwnershipCanonicalSHA256`), so a coverage claim, a gap or an unowned
  entry cannot be self-minted without an explicit re-review of that digest.

## Scope: what the gate can and cannot decide

Decides:

* a claimed clause is a real obligation of the claimed section, at the line it
  claims, quoted verbatim from the pinned document;
* the discharging acceptance case is registered, has an executable test owner,
  and is owned by that binding;
* the declared coverage level equals the measured ratio;
* a disclosed unowned section is real, is not also owned, and does not cover a
  catalog-required binding;
* a gap names its own section as a whole identifier and names the production
  declaration its binding is registered to.

Does **not** decide — the residual class, stated the same way in `README.md`:

* **That the named acceptance case exercises the clause's meaning.** A binding
  could enumerate every clause of a section and point all of them at one weak
  test, and the gate would admit it. Complete enumeration discharged by
  inadequate tests is out of scope and is not claimed. See "The residual is not
  hypothetical" below: the gate's one admitted binding is already an instance.
* **That a section carries no obligation.** The scanner sees uppercase RFC 2119
  keyword lines only. When it measures zero the honest report is `unmeasured` —
  *no clause this checker can see*, which is a different fact from *no clause*.
  Nineteen of the 157 pinned headings are in that blind spot: 7.3, 10.8.1, 13.5,
  13.12, 13.14.1–13.14.5, 14.3, 14.6, 15.2, 16.6, 16.7, 18.2, 19.4 and
  Appendices A, B and C.
  `TestUnmeasuredCoverageIsAScannerBlindSpotNotAnAbsenceOfObligation` re-derives
  that set from the pinned document, pins the heading inventory at 157, and
  measures that every one of the nineteen carries at least eight non-blank body
  lines, so not one is a heading with nothing to discharge.
  Because `unmeasured` is refused rather than admitted, the cost is real and is
  stated: a section that genuinely had nothing to discharge could not be
  admitted either. No such section was found among the nineteen.
* **That a gap is substantive.** `verifyGapDiscloses` requires 32 characters,
  the section as a whole identifier, and the binding's production declaration by
  name. That is a tightening, not a proof: a sentence that satisfies all three
  and still says nothing useful is admitted, and the gate cannot decide
  otherwise.
* An `unevidenced` level means the registry makes no clause-level claim. It is
  not proof that the implementation covers nothing.

## The three named slivers

* `section:2.2` — **recorded unowned.** Was bound to `validateSessionRecordCommon`.
* `section:18.4` — **recorded unowned.** Was bound to `OpenProjection`.
* `section:6.5` — **binding kept, `unevidenced` 0/3, gap named.** `translateV3`
  is the legacy terminal-table translator, so the binding is in the right area,
  but Section 6.5 requires `required_capabilities` to default to the platform
  lane minimum (`SPEC.md:2585`) while `internal/config/validation.go` accepts
  only an empty default, and the sanitized `ax doctor` / `ax terminal backends`
  inspection surface Section 6.5 requires does not exist.

None was silently re-bound to a different symbol. All three are refused for
assigned-scope admission; each refusal is reproduced verbatim in "Reproduced
refusals" below.

## The five further gaps the gate surfaced

Two are the same shape as the three named slivers:

* `section:17.2` — its single clause is an unknown-event reader obligation
  ("MAY retain an unknown event as inert history but MUST NOT derive state from
  it") while the binding names `config.EncodeCurrent`, the Configuration writer.
* `section:2.1` — its single clause is a replica runtime obligation (no run or
  resume without takeover or fork). No code here implements it.

Three came out of closing the keyword-absence bypass. All three are catalog-
required — `expectedCatalogSectionBindings` requires an owner for each — so
`unowned_sections` was not available to them. Each keeps its binding, carries a
mandatory gap naming why the scanner measures zero *and* what is missing, and is
routed out of the admitted set by the full-only rule. None was re-bound to a
friendlier symbol:

* `section:15.2` — the nineteen-row normative exit-code registry
  (`SPEC.md:11073-11095`). Implemented nowhere; the only `os.Exit` calls in the
  tree are the `exit(1)` failure paths of `cataloggen` and `tracecheck`.
* `section:7.3` — the closed Provider Manifest (`SPEC.md:2745-2796`). Its sole
  trace is a catalog row naming the URN.
* `section:13.14.5` — survives with a justification rather than a re-binding.
  `validateSessionEventV2` is topically the subject of the section; the gap says
  so and says that the gate cannot measure how much of it the binding discharges.

All are disclosed with the gap named rather than re-bound.

## The residual is not hypothetical: the one admitted binding is an instance

This was measured on this tree by the rework-round-3 run and is reported, not
fixed; it is out of this Bug's scope and belongs on its own board item.

`section:6.2` is the gate's only `full` binding, 1/1. Its clause is
`SPEC.md:2417` — on native Windows the terminal backend "MUST be
<code>conpty</code>". It is discharged by the acceptance case
`config-versioned-readers`, which registers thirteen executable tests against
the production entry point `internal/config/loader.go:Load`.

The review brief characterised that case as positive-path only. That is not
accurate on this tree, and the correction is worth stating: the case's
`TestLoadRefusesUnknownClosedMembersUnsupportedVersionsAndMalformedReads`
carries a real negative arm ("built-in backend unsupported on platform") that
refuses `ax.tmux` on native Windows through `Decode`, Section 6.2's own
production declaration.

What the case does **not** cover is the rest of the clause. Driving the
production loader directly:

```text
PROBE control: backend_id=ax.conpty on native Windows ADMITTED
PROBE backend_id=ax.tmux on native Windows: err=configuration decode selected file for config-file from environment failed isValidation=true
PROBE backend_id=com.example.term on native Windows: ADMITTED backend="com.example.term"
```

A trusted external backend is admitted as the selected backend on native
Windows. `internal/config/validation.go:682-687` refuses only the two built-in
mismatches (`ax.tmux` on Windows, `ax.conpty` off Windows); an `external_trust`
entry with `enabled = true` joins the registered set and is selectable, so
"MUST be `conpty`" is enforced against built-ins and not against external
backends.

The gate admits `section:6.2` at 1/1 anyway, correctly and by design: the clause
is enumerated, quoted verbatim at its line, and discharged by a registered case
the binding owns. Whether that case exercises the clause's *meaning* is exactly
and only the residual declared out of scope above. One admitted binding out of
forty-eight is a thin positive arm, and this is what its thinness costs. Probe
evidence: `BUG-260902-2m7slg_section-6.2-residual-probe.log`. The probe file was
temporary and was deleted; the tree is unmodified (`git write-tree` = `163d498`
= `HEAD^{tree}`).

## Measured coverage

Verbatim output of `go run ./internal/traceability/cmd/tracecheck` on this tree,
exit 0:

```text
traceability ok: contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=48 full=1 partial=0 sliver=1 unevidenced=43 unmeasured=3 unowned=2 clauses_discharged=2/394
```

48 section bindings discharge **2 of the 394** normative clauses their sections
carry. Assigned-scope admission succeeds today for `-section 6.2` and **nothing
else**; every other assignment is refused with its measured ratio and its gap.
That is a disclosure of the shipped state, not a target that was met.

### Every binding, with its measured ratio and gap

Regenerated from the shipped `ownership.v0.5.0.json` and the measured clause
inventory on this tree, in registry order.

| Binding | Coverage | Discharged | Production owner | Gap |
| --- | --- | ---: | --- | --- |
| `section:4.B` | `unevidenced` | 0/12 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 4.B; no clause of 4.B is enumerated against an acceptance case, so the registry makes no clause-level claim about the 12 obligations it carries. |
| `section:1.6` | `unevidenced` | 0/31 | `internal/scalar/scalar.go:ErrInvalidScalar` | ErrInvalidScalar is the internal/scalar refusal funnel for the common wire scalars Section 1.6 declares, but its CBOR decoder clauses, its UTF-16 ordering derivation clause and its wall-clock-ownership clause have no implementation, and no clause of Section 1.6 is enumerated against an acceptance case. |
| `section:2.1` | `unevidenced` | 0/1 | `internal/canonicaljson/closed_shapes.go:validateSessionRecordCommon` | The single Section 2.1 clause is a replica runtime obligation - no run or resume of the logical session without takeover or fork - and validateSessionRecordCommon only validates Session Record shape, so nothing in this binding discharges it. |
| `section:2.3` | `unevidenced` | 0/7 | `internal/canonicaljson/closed_shapes.go:validateSessionRecordCommon` | validateSessionRecordCommon validates the immutable Section 2.3 name grammar only; local/peer lookup, ASCII-fold collision handling, route choice and the associated runtime errors are not implemented, and no clause of Section 2.3 is enumerated. |
| `section:2.4` | `unevidenced` | 0/4 | `internal/canonicaljson/closed_shapes.go:validateSessionRecordCommon` | validateSessionRecordCommon validates the Section 2.4 execution-profile enum value only; profile selection, first-use confirmation and downgrade behaviour are not implemented, and no clause of Section 2.4 is enumerated. |
| `section:6.1` | `unevidenced` | 0/2 | `internal/config/loader.go:Load` | Load resolves and reads the Section 6.1 configuration roots, but the clause requiring those resolved roots across the CLI, service, RPC server and provider host has no implementation, and no clause of Section 6.1 is enumerated. |
| `section:6.2` | `full` | 1/1 | `internal/config/schema.go:Decode` | — |
| `section:6.3` | `unevidenced` | 0/11 | `internal/config/validation.go:validateConfiguration` | validateConfiguration enforces many Section 6.3 field constraints, but no clause of Section 6.3 is enumerated against an acceptance case, so the registry makes no clause-level claim about its obligations. |
| `section:6.4` | `unevidenced` | 0/2 | `internal/config/migration.go:Migrate` | Migrate writes the Section 6.4 durable migration with an owner-only backup and atomic replacement, but no clause of Section 6.4 is enumerated against an acceptance case. |
| `section:6.5` | `unevidenced` | 0/3 | `internal/config/validation.go:translateV3` | translateV3 maps the legacy terminal table into the Section 6.5 terminal configuration, but the Section 6.5 required_capabilities default is the platform lane minimum while internal/config/validation.go accepts only an empty default, and the sanitized ax doctor and terminal-backends inspection surface Section 6.5 requires does not exist; no clause of Section 6.5 is enumerated. |
| `section:17.1` | `unevidenced` | 0/6 | `internal/config/schema.go:Decode` | Decode reads every pinned Section 17.1 document version, but no clause of Section 17.1 is enumerated against an acceptance case. |
| `section:17.2` | `unevidenced` | 0/1 | `internal/config/writer.go:EncodeCurrent` | The single Section 17.2 clause is an unknown-event reader obligation - retain an unknown event as inert history but never derive state from it - and EncodeCurrent writes Configuration documents, so this binding does not discharge Section 17.2 at all. |
| `section:17.4` | `unevidenced` | 0/4 | `internal/config/migration.go:AssessCompatibility` | AssessCompatibility implements the Section 17.4 read-only downgrade assessment, but no clause of Section 17.4 is enumerated against an acceptance case. |
| `section:4.C` | `unevidenced` | 0/7 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 4.C; no clause of 4.C is enumerated against an acceptance case, so the registry makes no clause-level claim about the 7 obligations it carries. |
| `section:4.D` | `unevidenced` | 0/3 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 4.D; no clause of 4.D is enumerated against an acceptance case, so the registry makes no clause-level claim about the 3 obligations it carries. |
| `section:5.1` | `unevidenced` | 0/9 | `internal/canonicaljson/closed_shapes.go:validateSessionRecordWithDerivation` | validateSessionRecordWithDerivation validates the Section 5.1 Session Record shape; the derivation and lifecycle behaviour Section 5.1 requires is not implemented and no clause of Section 5.1 is enumerated. |
| `section:5.2` | `unevidenced` | 0/18 | `internal/canonicaljson/core_records.go:validateSessionEvent` | validateSessionEvent validates the Section 5.2 Session Event shape; the event authorship, ordering and epoch behaviour Section 5.2 requires is not implemented and no clause of Section 5.2 is enumerated. |
| `section:5.3` | `unevidenced` | 0/8 | `internal/canonicaljson/core_records.go:validateLeaseRecord` | validateLeaseRecord validates the Section 5.3 Lease Record shape; lease acquisition, renewal, expiry and fencing are not implemented and no clause of Section 5.3 is enumerated. |
| `section:5.4` | `unevidenced` | 0/9 | `internal/canonicaljson/core_records.go:validateCheckpointRecord` | validateCheckpointRecord validates the Section 5.4 Checkpoint Record shape; checkpoint creation and restore behaviour is not implemented and no clause of Section 5.4 is enumerated. |
| `section:5.5` | `unevidenced` | 0/3 | `internal/canonicaljson/core_records.go:validateProviderIdentityRecord` | validateProviderIdentityRecord validates the Section 5.5 provider identity shape; the destination-adapter obligations of Section 5.5 are not implemented and no clause of it is enumerated. |
| `section:5.6` | `unevidenced` | 0/9 | `internal/canonicaljson/core_records.go:validateWorkspaceGroupRecord` | validateWorkspaceGroupRecord validates the Section 5.6 workspace group shape; workspace group membership and union behaviour is not implemented and no clause of Section 5.6 is enumerated. |
| `section:7.2` | `unevidenced` | 0/7 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 7.2; no clause of 7.2 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 7 obligations it carries. |
| `section:7.3` | `unmeasured` | 0/0 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 7.3; the RFC 2119 scanner measures no clause line under 7.3, because the closed Provider Manifest states its obligations as a required-member table, a 16-entry operation registry and exactly 7 capability names rather than in uppercase keywords, so the gate cannot measure this section - and no provider manifest is implemented anywhere in this repository. |
| `section:3.2` | `unevidenced` | 0/13 | `internal/localstore/paths.go:ResolvePaths` | ResolvePaths implements the owner-only Section 3.2 path registry, but no clause of Section 3.2 is enumerated against an acceptance case, so the registry makes no clause-level claim about its obligations. |
| `section:3.3` | `unevidenced` | 0/4 | `internal/localstore/projection.go:OpenProjection` | OpenProjection rebuilds and recovers the Section 3.3 SQLite projection, but the replication-exclusion clause and the wire-compatibility-signal clause have no implementation and no clause of Section 3.3 is enumerated. |
| `section:7.4` | `unevidenced` | 0/1 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 7.4; no clause of 7.4 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 1 obligations it carries. |
| `section:7.5` | `unevidenced` | 0/53 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 7.5; no clause of 7.5 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 53 obligations it carries. |
| `section:7.A` | `unevidenced` | 0/2 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 7.A; no clause of 7.A is enumerated against an acceptance case, so the registry makes no clause-level claim about the 2 obligations it carries. |
| `section:7.8` | `unevidenced` | 0/2 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 7.8; no clause of 7.8 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 2 obligations it carries. |
| `section:7.9` | `unevidenced` | 0/8 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 7.9; no clause of 7.9 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 8 obligations it carries. |
| `section:9.2` | `unevidenced` | 0/35 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 9.2; no clause of 9.2 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 35 obligations it carries. |
| `section:10.8.5` | `unevidenced` | 0/4 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 10.8.5; no clause of 10.8.5 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 4 obligations it carries. |
| `section:10.1` | `unevidenced` | 0/3 | `internal/canonicaljson/closed_shapes.go:validateImmutableObjectShape` | validateImmutableObjectShape enforces the Section 10.1 common envelope, but the digest-derived storage path clause and the quarantine-and-stop clause are implemented in internal/localstore rather than by this binding, and no clause of Section 10.1 is enumerated. |
| `section:10.2` | `unevidenced` | 0/5 | `internal/canonicaljson/closed_shapes.go:validateBlobDescriptor` | validateBlobDescriptor validates the Section 10.2 Blob Descriptor shape, but no clause of Section 10.2 is enumerated against an acceptance case, so the registry makes no clause-level claim about its obligations. |
| `section:10.3` | `sliver` | 1/3 | `internal/canonicaljson/closed_shapes.go:validateBlobDescriptor` | validateBlobDescriptor enforces the Section 10.3 chunk offset invariant, but the two receiver clauses - validate every chunk before marking it present, and validate the complete blob after assembly - have no implementation because no transfer receiver exists. |
| `section:10.4` | `unevidenced` | 0/25 | `internal/canonicaljson/closed_shapes.go:validateTransferManifest` | validateTransferManifest validates the Section 10.4 Transfer Manifest shape; the transfer, staging and conflict behaviour Section 10.4 requires is not implemented and no clause of Section 10.4 is enumerated. |
| `section:11.2` | `unevidenced` | 0/5 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 11.2; no clause of 11.2 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 5 obligations it carries. |
| `section:11.3` | `unevidenced` | 0/27 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 11.3; no clause of 11.3 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 27 obligations it carries. |
| `section:11.8` | `unevidenced` | 0/4 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 11.8; no clause of 11.8 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 4 obligations it carries. |
| `section:11.9` | `unevidenced` | 0/2 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 11.9; no clause of 11.9 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 2 obligations it carries. |
| `section:13.14.5` | `unmeasured` | 0/0 | `internal/canonicaljson/core_records.go:validateSessionEventV2` | validateSessionEventV2 validates the Section 13.14.5 v2 Session Event shape, which is topically the subject of the section; the RFC 2119 scanner measures no clause line under 13.14.5, because the section states its obligations as required-member and variant tables rather than in uppercase keywords, so the gate cannot measure how much of the section this binding actually discharges. |
| `section:13.15` | `unevidenced` | 0/6 | `internal/canonicaljson/core_records.go:validateSessionEventV3` | validateSessionEventV3 validates the Section 13.15 event payload shape, but no clause of Section 13.15 is enumerated against an acceptance case. |
| `section:15.1` | `unevidenced` | 0/7 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 15.1; no clause of 15.1 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 7 obligations it carries. |
| `section:15.2` | `unmeasured` | 0/0 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 15.2; the RFC 2119 scanner measures no clause line under 15.2, because its 19-row exit-code registry is a normative table stated without uppercase keywords, so the gate cannot measure this section - and no exit-code mapping is implemented, the only os.Exit calls in the tree being the exit(1) failure paths of cataloggen and tracecheck. |
| `section:15.3` | `unevidenced` | 0/3 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference 15.3; no clause of 15.3 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 3 obligations it carries. |
| `section:17.3` | `unevidenced` | 0/3 | `internal/canonicaljson/closed_shapes.go:validateMigrationProvenance` | validateMigrationProvenance validates the Section 17.3 provenance contribution to object identity; the in-place-edit prohibition and the backup/atomic-write migration clauses are implemented in internal/config, and no clause of Section 17.3 is enumerated. |
| `section:18.1` | `unevidenced` | 0/5 | `internal/canonicaljson/core_records.go:ValidateObservationEvent` | ValidateObservationEvent validates the Section 18.1 Observation Event shape; the observation stream and scheduled enrichment behaviour is not implemented and no clause of Section 18.1 is enumerated. |
| `section:appendix-d` | `unevidenced` | 0/16 | `internal/catalog/catalog.go:ForRelease` | ForRelease only returns the reviewed typed catalog rows that reference Appendix D; no clause of Appendix D is enumerated against an acceptance case, so the registry makes no clause-level claim about the 16 obligations it carries. |

Counts over that table: `full` 1, `partial` 0, `sliver` 1, `unevidenced` 43,
`unmeasured` 3 — 48 bindings, matching the command output above. No binding
carries the gap placeholder `—` except `section:6.2`, which is `full` and owes
no gap.

### Disclosed unowned sections

| Section | Gap | Evidence |
| --- | --- | --- |
| `section:2.2` | Section 2.2 states the unconditional lease, replica, replication, tombstone and capability invariants of the product, and none of them is implemented here. The binding this entry replaces named validateSessionRecordCommon, which validates Session Record shape and the execution-profile enum value only. | The repository contains no lease, epoch, replica, takeover, tombstone or replication implementation: internal/canonicaljson validates record shape, internal/localstore stores immutable objects and rebuilds a local index, and the two acceptance cases the replaced binding named are canonical-identity cases. |
| `section:18.4` | Section 18.4 requires audit retention - the exact Section 5.2 event types for lease change, force confirmation, managed replacement, task-board launch and adoption and tombstone issuance and resolution, and retention of the event and every referenced immutable object for as long as the Session Record. No retention, expiry or garbage-collection rule is implemented. | internal/localstore/projection.go opens, rebuilds and recovers the SQLite projection and installs immutable blobs; it declares no retention, expiry or garbage-collection behaviour, and the binding this entry replaces named OpenProjection with the same two storage acceptance cases already used by the Section 3.3 binding. |

## Reproduced refusals

Verbatim, from `.temp/BUG-260902-2m7slg/r3/tracecheck.log`, re-run on this tree.
Each block is the real stderr and the real exit code. Exit 1 is the expected red
this gate exists to produce.

```text
$ tracecheck -section 6.2
traceability ok: contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55 assigned_scopes=1
section coverage: bindings=48 full=1 partial=0 sliver=1 unevidenced=43 unmeasured=3 unowned=2 clauses_discharged=2/394
  exit=0
$ tracecheck -section 6.2 -section 13.14.5
spec-to-code traceability check failed: assigned section "13.14.5" binding "section:13.14.5" discharges 0/0 normative clauses, which is unmeasured coverage; assigned-scope admission requires full: validateSessionEventV2 validates the Section 13.14.5 v2 Session Event shape, which is topically the subject of the section; the RFC 2119 scanner measures no clause line under 13.14.5, because the section states its obligations as required-member and variant tables rather than in uppercase keywords, so the gate cannot measure how much of the section this binding actually discharges.
exit status 1
  exit=1
$ tracecheck -section 13.14.5
spec-to-code traceability check failed: assigned section "13.14.5" binding "section:13.14.5" discharges 0/0 normative clauses, which is unmeasured coverage; assigned-scope admission requires full: validateSessionEventV2 validates the Section 13.14.5 v2 Session Event shape, which is topically the subject of the section; the RFC 2119 scanner measures no clause line under 13.14.5, because the section states its obligations as required-member and variant tables rather than in uppercase keywords, so the gate cannot measure how much of the section this binding actually discharges.
exit status 1
  exit=1
$ tracecheck -section 7.3
spec-to-code traceability check failed: assigned section "7.3" binding "section:7.3" discharges 0/0 normative clauses, which is unmeasured coverage; assigned-scope admission requires full: ForRelease only returns the reviewed typed catalog rows that reference 7.3; the RFC 2119 scanner measures no clause line under 7.3, because the closed Provider Manifest states its obligations as a required-member table, a 16-entry operation registry and exactly 7 capability names rather than in uppercase keywords, so the gate cannot measure this section - and no provider manifest is implemented anywhere in this repository.
exit status 1
  exit=1
$ tracecheck -section 15.2
spec-to-code traceability check failed: assigned section "15.2" binding "section:15.2" discharges 0/0 normative clauses, which is unmeasured coverage; assigned-scope admission requires full: ForRelease only returns the reviewed typed catalog rows that reference 15.2; the RFC 2119 scanner measures no clause line under 15.2, because its 19-row exit-code registry is a normative table stated without uppercase keywords, so the gate cannot measure this section - and no exit-code mapping is implemented, the only os.Exit calls in the tree being the exit(1) failure paths of cataloggen and tracecheck.
exit status 1
  exit=1
$ tracecheck -section 2.2
spec-to-code traceability check failed: assigned section "2.2" binding "section:2.2" is recorded unowned: Section 2.2 states the unconditional lease, replica, replication, tombstone and capability invariants of the product, and none of them is implemented here. The binding this entry replaces named validateSessionRecordCommon, which validates Session Record shape and the execution-profile enum value only.
exit status 1
  exit=1
$ tracecheck -section 18.4
spec-to-code traceability check failed: assigned section "18.4" binding "section:18.4" is recorded unowned: Section 18.4 requires audit retention - the exact Section 5.2 event types for lease change, force confirmation, managed replacement, task-board launch and adoption and tombstone issuance and resolution, and retention of the event and every referenced immutable object for as long as the Session Record. No retention, expiry or garbage-collection rule is implemented.
exit status 1
  exit=1
$ tracecheck -section 10.3
spec-to-code traceability check failed: assigned section "10.3" binding "section:10.3" discharges 1/3 normative clauses, which is sliver coverage; assigned-scope admission requires full: validateBlobDescriptor enforces the Section 10.3 chunk offset invariant, but the two receiver clauses - validate every chunk before marking it present, and validate the complete blob after assembly - have no implementation because no transfer receiver exists.
exit status 1
  exit=1
$ tracecheck -section 6.5
spec-to-code traceability check failed: assigned section "6.5" binding "section:6.5" discharges 0/3 normative clauses, which is unevidenced coverage; assigned-scope admission requires full: translateV3 maps the legacy terminal table into the Section 6.5 terminal configuration, but the Section 6.5 required_capabilities default is the platform lane minimum while internal/config/validation.go accepts only an empty default, and the sanitized ax doctor and terminal-backends inspection surface Section 6.5 requires does not exist; no clause of Section 6.5 is enumerated.
exit status 1
  exit=1
$ tracecheck -section 9.2
spec-to-code traceability check failed: assigned section "9.2" binding "section:9.2" discharges 0/35 normative clauses, which is unevidenced coverage; assigned-scope admission requires full: ForRelease only returns the reviewed typed catalog rows that reference 9.2; no clause of 9.2 is enumerated against an acceptance case, so the registry makes no clause-level claim about the 35 obligations it carries.
exit status 1
  exit=1
```

`-section 6.2 -section 13.14.5` exits **1**. Admission is per assignment and the
pair is refused as a whole, with no success output; that is pinned by
`TestRunReportsExactCoverageAndFailsClosed`.

## Anti-vacuity

### Planted mutants on the shipped tree

Counted by running each test with `-v` and counting its subtests on this tree,
not carried forward. Every mutant **narrows** the gate rather than deleting it:
each keeps the binding, its symbol and its acceptance case intact and breaks one
thing.

| Test | Mutants | Driven through |
| --- | ---: | --- |
| `TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted` | 13 | `VerifyRepository` |
| `TestPlantedSliverRedensTheProductionEntryPoints` | 15 | `VerifyRepository` **and** `VerifyAssignedSections` |
| `TestUnmeasuredBindingWithAnHonestGapIsStillAccepted` | 4 | `VerifyRepository` |

The thirteen clause mutants each keep the `full` claim and break one thing: one
clause instead of eleven, six instead of eleven, none instead of eleven, an
invented clause id, an invented quote, a real quote from another section, prose
with no obligation, a clause moved to a line it does not occupy, no acceptance
case, an unregistered acceptance case, a borrowed case the binding does not own,
a duplicated clause, and an omitted level.

The fifteen registry mutants add the buckets and the disclosure surface: a
sliver or unevidenced binding claiming the whole section, an `unmeasured`
binding dropping its now-mandatory gap, padding it, relabelling itself
`unevidenced`, relabelling itself `full` to reach admission, a gap naming a
neighbouring section identifier instead of its own, an unowned entry padding its
gap with a neighbouring identifier, a contract owner claiming section coverage,
a section registered as both owned and unowned, an unowned entry covering a
catalog-required owner, an unowned entry with no gap, one with no evidence, and
one self-minted for a section that does not exist.

### The load-bearing mutant, re-measured by this run

The one mutant that reproduces the reported defect, planted and measured on this
tree by the rework-round-3 run, then restored from a file backup (never
`git checkout`). Full log: `BUG-260902-2m7slg_rework-r3-mutant-readmit.log`.

Mutation — `internal/traceability/traceability.go:402`, admission re-admits the
`unmeasured` bucket, narrowing rather than deleting:

```diff
-			if measured.Level != coverageFull {
+			if measured.Level != coverageFull && measured.Level != coverageUnmeasured {
```

Confirmed **present** in the file by `grep` before measuring, and confirmed
**absent** after restore; `git write-tree` afterwards returned `163d498`, byte
for byte `HEAD^{tree}`.

Measured with the mutant present:

| Command | Shipped | Mutant |
| --- | ---: | ---: |
| `tracecheck -section 15.2` | exit 1 | **exit 0** |
| `tracecheck -section 7.3` | exit 1 | **exit 0** |
| `tracecheck -section 13.14.5` | exit 1 | **exit 0** |
| `tracecheck -section 6.2` | exit 0 | exit 0 |
| `go test ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1` | exit 0 | **exit 1** |

`-section 15.2` exiting 0 while nothing implements the exit-code registry is
this Bug's own consequence paragraph, verbatim. Five test functions redden and
none of them is a delete-detector: `TestVerifyAssignedSectionsBindsGranularScopeToOwnersAndExecutableCases`,
`TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers` (subtests 15.2,
7.3 and 13.14.5), `TestRunReportsExactCoverageAndFailsClosed`, and
`TestRunRefusesEveryAssignedSectionThatOnlySlivers` — the last of which catches
the success output the mutant emits:

```text
run(-section 15.2) emitted success output "traceability ok: ... assigned_scopes=1\nsection coverage: bindings=48 full=1 ... clauses_discharged=2/394\n"
```

Both production entry points — `VerifyAssignedSections` in the library and
`main` → `run` → `VerifyAssignedSections` in the command — are covered.

### A genuinely adequate binding is still admitted

* `tracecheck -section 6.2` exits **0** against the unmodified shipped registry,
  as reproduced above. This is the only real admit the shipped data supports,
  which is disclosed here and in `README.md` as a thin positive arm rather than
  hidden.
* `TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted` additionally
  builds an eleven-of-eleven Section 6.3 binding from the pinned document and
  requires it to be admitted, so the positive arm is not carried by one shipped
  row alone.
* `TestUnmeasuredBindingWithAnHonestGapIsStillAccepted` is the anti-vacuity arm
  for the tightened `unmeasured` bucket: requiring a gap must not make the bucket
  impossible to declare honestly. A binding that discloses honestly is measured
  `unmeasured` 0/0 and accepted by the coverage check; it is refused only at
  assigned-scope admission, which is a separate decision.

## The embedded document must not reach a product binary

`internal/specdoc` is now imported by non-test code, so the earlier "only test
binaries import it" claim became false and was corrected in `README.md` and in
the package comment. `TestEmbeddedDocumentNeverReachesAProductBinary` reads the
module import graph from source with `go/parser` and refuses any `main` package
outside `tracecheck` that can reach `specdoc`, proving its own detector by
planting a `cmd/ax` that imports `internal/traceability`.
`TestModuleHasNoProductCommandYet` fails when `ax` lands, so the README sentence
cannot quietly go stale. No `ax` main package exists today, so the 883 KiB embed
reaches no shipped binary.

## Files changed at leaf `2d0962c`

```text
 LOGBOOK.md                                        |   26 +
 README.md                                         |  193 ++-
 internal/specdoc/embed_reach_test.go              |  223 +++
 internal/specdoc/specdoc.go                       |   13 +-
 internal/traceability/cmd/tracecheck/main.go      |   20 +-
 internal/traceability/cmd/tracecheck/main_test.go |  143 +-
 internal/traceability/ownership.v0.5.0.json       | 1896 +++++++++++++++++----
 internal/traceability/traceability.go             |  531 +++++-
 internal/traceability/traceability_test.go        |  718 +++++++-
 9 files changed, 3304 insertions(+), 459 deletions(-)
```

No existing assertion was weakened or deleted. Three assertions changed
direction and none was removed: `section:13.14.5` moved from the admitted arm to
the refusal table in both the library and the command tests, and a new assertion
pins that the pair `{6.2, 13.14.5}` is refused as a whole with no success
output. `TestVerifyRepositoryRejectsNarrowedOwnership`,
`TestMainRejectsRenamedScalarSectionOwnerDeclarations`,
`TestMainRejectsOneNarrowedAssignedSectionBinding`,
`TestMainRejectsDetachedScopeSpecificAcceptanceCase` and
`TestMainRejectsMissingScopeSpecificProductionDeclaration` all still hold. Two
rows left the rename table because `section:2.2` and `section:18.4` no longer
have production owners to rename; the same declarations stay covered by the
`section:2.1` and `section:3.3` rows.

## Validation

Every row below was run on this tree as a standalone process by the
rework-round-3 run and reports its real exit code. Full log:
`BUG-260902-2m7slg_rework-r3-validation.log`.

| Command | Exit | Result |
| --- | ---: | --- |
| `gofmt -l .` | 0 | no output |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `go test ./... -count=1` | 0 | 11/11 packages ok |
| `go test ./... -count=1 -cover` | 0 | `specdoc` 100.0%, `traceability` 86.4%, `tracecheck` 88.5% |
| `go generate ./internal/catalog` | 0 | |
| `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 | no generated-catalog drift |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | ratio above |
| `go run ./internal/traceability/cmd/tracecheck -section 6.2` | 0 | the only admitted assignment |
| `go run ./internal/traceability/cmd/tracecheck -section 6.2 -section 13.14.5` | **1** | expected red: the pair is refused as a whole |
| `go run ./internal/traceability/cmd/tracecheck -section 13.14.5` | **1** | expected red: `unmeasured` 0/0 |
| `go run ./internal/traceability/cmd/tracecheck -section 7.3` | **1** | expected red: `unmeasured` 0/0 |
| `go run ./internal/traceability/cmd/tracecheck -section 15.2` | **1** | expected red: `unmeasured` 0/0 |
| `go run ./internal/traceability/cmd/tracecheck -section 2.2` | **1** | expected red: recorded unowned |
| `go run ./internal/traceability/cmd/tracecheck -section 18.4` | **1** | expected red: recorded unowned |
| `go run ./internal/traceability/cmd/tracecheck -section 10.3` | **1** | expected red: `sliver` 1/3 |
| `go run ./internal/traceability/cmd/tracecheck -section 6.5` | **1** | expected red: `unevidenced` 0/3 |
| `go run ./internal/traceability/cmd/tracecheck -section 9.2` | **1** | expected red: `unevidenced` 0/35 |

The full suite was re-run after the load-bearing mutant was restored and exited
0 on a tree whose `git write-tree` equals `HEAD^{tree}`.

## Reported, not fixed here

* **Section 6.2's clause is not fully enforced.** A trusted external terminal
  backend is admitted on native Windows while `SPEC.md:2417` requires `conpty`.
  Measured above; a Configuration defect, and its own board item.
* **Section 6.5's production defect.** `SPEC.md:2585` requires
  `required_capabilities` to default to the platform lane minimum;
  `internal/config/validation.go` accepts only an empty default. Named in the
  binding's gap and in `README.md`; a Configuration defect, and its own board
  item.
* **Allowed-but-not-required sections** are still neither forced owned nor
  disclosed unowned. Pre-existing; the gate does not decide it.
* **Gap substance.** The gap rule can tell a gap about 6.5 from one about 6.55
  and requires the production declaration to be named, but it cannot tell a real
  gap from a well-formed empty one. Stated as a bound, not closed.

## Revision history of this artifact

Revision 1 of this document described a model that the revision-1 review
rejected and revision 2 removed, and was contradicted by the shipped binary in
seven ways: it named a `declarative` level that no longer exists and glossed it
as "the section carries no obligation of its own", which was the false belief
the rework removed; it said admission requires `full` or `declarative` when it
requires `full` alone; it claimed four sections were admitted with no clause
evidence when no such admission exists; it claimed admission succeeds for
`-section 13.14.5`, `-section 7.3` and `-section 15.2`, all of which exit 1; its
validation table showed `-section 6.2 -section 13.14.5` exiting 0 when it exits
1; it presented a fenced block as verbatim `tracecheck` output containing
`declarative=3`, a token no build of the command emits; and its per-binding rows
showed §7.3, §13.14.5 and §15.2 as `declarative` with an empty gap when the
registry has `unmeasured` with a mandatory non-empty gap on each.

That falsehood was this Bug's own consequence paragraph: a reader taking it at
face value would conclude §15.2, §7.3 and §13.14.5 are owned and that a Story
assigned one of them could do nothing. This revision replaces it in place; no
stale copy is left beside it.
