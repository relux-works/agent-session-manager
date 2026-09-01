# TASK-260830-8x76g1 — developer rework revision 6 logbook

## Findings and decisions

1. The revision-4 bypass was broader than one malformed fixture: the old
   validator selected Blob and Transfer Manifest specially, then admitted every
   other registered identity after extension validation alone. Catalog
   recognition was incorrectly acting as shape attestation.
2. The clean fail-closed architecture is an explicit schema/version validator
   registry checked against the generated self-identity catalog. Implemented
   closed shapes validate; unavailable complete shapes refuse explicitly.
   There is no default validator and no extension-only fallback.
3. Session Record `1.0.0` is the Section 10.1 record whose complete normative
   structure is present in the pinned scope. Its common envelope and all nested
   closed/tagged shapes were implemented. Other Section 10.1 schemas retain
   common-envelope scalar validation but refuse before identity attestation
   because their full schema-specific structure is outside this leaf.
4. The completeness test obtains expected identities from
   `catalog.Current().SelfIdentities` and mutates the validator map. This is
   intentionally non-circular: adding a catalog row without a validator causes
   production initialization and the test to fail.
5. New Session Record refusal cases are committed to the closed-shape fuzz
   corpus and the target remains wired into the configured bounded suite at
   `100x`, `parallel=1`.
6. Traceability now names `validateImmutableObjectShape` as the Section 10.1
   production owner and declares the totality/common-envelope attacks. The
   reviewed registry digest was changed only after an expected-red digest
   refusal proved the pin was live.
7. The configured `gofmt -l .` command crossed the `.temp/` boundary and failed
   on reviewer probe sources. Rewriting historical evidence would be the wrong
   fix. The gate now enumerates Git tracked and non-ignored Go files, which
   includes new production sources and excludes ignored scratch artifacts.

## Operational anomalies

- The project-local `go-testing-tools` adapter was initially absent;
  `curator status --check` returned exit 1. `curator install` installed pinned
  revision `90c1515239eed9321068f3bafbeb5d0a0c2aa26a`, after which
  `curator status --check` returned exit 0. The pinned skill was then read in
  full before the final gate cycle.
- `task-board validate` returned exit 0 but reported 262 inherited
  `MISSING_ACTIVITY` diagnostics outside this task. They remain explicitly
  outside this implementation scope.
- A pre-existing untracked `.DS_Store` remains untouched.

## Scope boundary

No publication or atomic migration-reference behavior was added. External
facts that require referenced blobs, child manifests, isolated Git object
databases, or capture evidence remain classified as external in the pinned
constraint enumeration rather than guessed from one identity candidate.

