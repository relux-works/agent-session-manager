# TASK-260830-kp4zpu conformance matrix

Authority: relux-works/agent-session-manager-spec v0.6.0
(`0cbdf100dbf84df50c64f792b1f940e3a67859a6`); section numbers below
are the retained v0.6.0 headings. Historical v0.5.0
(`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`) behavior is unchanged:
the Section 8 and NativeDiscoveryProof rows cited here are
byte-identical in both revisions, and every retained entry below
keeps its existing tests green.

Legend: owner THIS LEAF = new production in `internal/provhost`
(`identity_create.go`, `identity_bind.go`); RETAINED = landed code
reused unchanged; SIBLING = TASK-260830-3uzfyn (profile
resolution/persistence, runs after this leaf). Every THIS LEAF row
names its production call site and its named committed tests.

## 2.4 Execution profiles

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Persisted enum standard/yolo; Session Record creation value; `profile.changed` derivation; losing-lease exclusion | SIBLING | None here (identity records carry no profile member; verifiable in `identityMembers`) |
| Resume fails `profile_mapping_unavailable` when the adapter cannot map the stored profile | SIBLING | None here |
| P0/E1/C1 end-to-end fixtures (direct/takeover/resume/fork, task-board, Pi equal-mapping) | SIBLING | None here |
| Pi equal-mapping disclosure (both profiles, one tool set) | RETAINED (`ProfileMapping`) + SIBLING flows | `TestProfileYOLOMappingIsDerivedFromSpec` (retained) |

## 5.5 Provider Identity Record

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Closed 16-member `urn:ax:schema:provider-identity` 1.0.0 shape | RETAINED + THIS LEAF (`CreateIdentity` emits) | `TestCreateIdentityEmitsAttestedRecord` (passes `CheckIdentity`, owner shape, binding attestation); `TestIdentityMembersAreDerivedFromSpec` (retained) |
| `subject_id` equals `session_id` | THIS LEAF (single `SessionID` param fans out to both) | `TestCreateIdentityEmitsAttestedRecord`; `TestCreateIdentityRefusals/session_id` |
| `provider_version` 1..128, `provider_version_range` 1..256 (opaque), `native_session_id` 1..512 exact, UUIDv7 members, timestamp member | THIS LEAF (`CreateIdentity`) | `TestCreateIdentityRefusals` (one row per arm) |
| `identity_kind` closed enum (5) | THIS LEAF + RETAINED (`isIdentityKind`) | `TestCreateIdentityRefusals/kind`; N-create-kind killed |
| Key grammar `[a-z][a-z0-9_.-]{0,63}`, map 0..32, string values 1..1024 | THIS LEAF (`checkCreateOpaque`) | `TestCreateIdentityRefusals` (opaque rows) |
| Absolute-path prefix refusal (decidable half) | THIS LEAF (`checkCreateOpaque`) | `TestCreateIdentityRefusals/opaque_absolute_prefix`; N-create-opaque-abs killed (token-preserving `/` swap) |
| Antigravity `backend_conversation_uuid` requires non-null realm | THIS LEAF (`CreateIdentity`) | `TestCreateIdentityAntigravityRecord` (positive); `TestCreateIdentityRefusals/realm_required`; N-create-realm-required killed |
| Extensions reverse-DNS keys, 0..64 | THIS LEAF (`checkCreateExtensions`) + owner backstop | `TestCreateIdentityRefusals` (extensions rows); N-create-backstop killed |
| Owner conjunction (drifted copy refused, not emitted) | THIS LEAF (`CreateIdentity` backstop) | `TestCreateIdentityRefusals/owner_backstop`; N-create-backstop killed |
| Normative negative fixtures: unknown kind; absent opaque; object value; absolute path; Antigravity null realm | THIS LEAF + RETAINED | kind/path/null-realm rows above (both dialects); absent-opaque and object-value are unrepresentable in the typed params (stated bound) and stay covered by the retained `CheckIdentity` witnesses |
| Normative example record | RETAINED | `TestSpecIdentityExampleDecodes`, `TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest` (retained) |

## 7.5 Required operations

| Clause | Owner | Test(s) |
| --- | --- | --- |
| `NativeDiscoveryProof` embedded type (exact 4 members, root absolute-or-null) | THIS LEAF (`DecodeNativeDiscovery`) | `TestDecodeNativeDiscoveryPositive`, `TestDecodeNativeDiscoveryRefusals`, `TestDiscoveryMembersAreDerivedFromSpec`; N-discovery-root, N-discovery-platform killed |
| `identify-session` success body carries identity + confidence + 1..4 evidence | RETAINED (`DecodeIdentifyResult`) + THIS LEAF (created records round-trip) | `TestCreatedIdentityRoundTripsIdentifyCall` (created record through `Host.Call`); `TestIdentifyThroughCall` (retained) |
| `resume`/`launch`/`quiesce`/`stop`/`fork`/`doctor` request bodies carry the identity | RETAINED decoders + THIS LEAF (`VerifyIdentityBuild` binds record to probed tuple) | `TestVerifyIdentityBuildPositive`, `TestVerifyIdentityBuildRefusals`; N-build-drift killed |
| Prepared `materialize` result carries `native_discovery` | THIS LEAF (decodes + binds) | `TestVerifyIdentityDiscoveryPositive`, `TestVerifyIdentityDiscoveryRefusals` |
| `(operation, operation_id)` mutation idempotency | RETAINED (`idempotency.go`) | Retained idempotency suite (creation idempotency is separate: byte-identical purity, `TestCreateIdentityIsByteIdentical`) |
| Remaining request vocabularies cross opaquely | RETAINED | Unchanged `doc.go` bound |

## 7.6 Quiescence proof

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Safe-proof members and fail-closed `safe` bit; carried identity validated | RETAINED (`DecodeQuiesceProof` + `CheckIdentity`) | Retained quiescence suite; created records accepted through the same `CheckIdentity` entry (`TestCreateIdentityEmitsAttestedRecord`) |

## 7.7 Profile mapping

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Six-provider yolo table; standard omission; exact-version probe before mapping | RETAINED (`ProfileMapping`) | `TestProfileYOLOMappingIsDerivedFromSpec` (retained, untouched); version probing stays the caller's (retained gap, unchanged) |

## 8.1 Matrix notation

| Clause | Owner | Test(s) |
| --- | --- | --- |
| A/C/U/? labels; unknown never rewritten as unsupported; conditional never advertised as available | THIS LEAF + RETAINED (`RequireCapability`) | Distinct refuse details per class (`TestCheckResumeTupleRefusals`, `TestStoreRootForRefusals`); passing never reports usable (`TestCheckResumeTuplePassesUsableRows` + retained `TestCapabilityGatePrecedesCall`) |

## 8.2 Native-store contract matrix

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Codex `~/.codex/sessions` | THIS LEAF (`StoreRootFor`) | `TestStoreRootForTable`, `TestResumeProvidersAreDerivedFromSpec` |
| Claude `~/.claude/projects` (key derived, never copied as identity) | THIS LEAF | Same + `TestVerifyIdentityDiscoveryPositive/project-keyed_subpath` (containment, not verbatim-key equality) |
| Gemini `~/.gemini/tmp/<hash>/chats` base | THIS LEAF (resolves the documented base; hash derivation stays the adapter's) | `TestStoreRootForTable` |
| Muse `$XDG_DATA_HOME/muse/sessions` defaulting below `~/.local/share` | THIS LEAF | `TestStoreRootForTable` (both XDG rows); `TestStoreRootForRefusals/xdg_relative` |
| Antigravity backend realm, no store root | THIS LEAF (backend-only, no root invented) | `TestStoreRootForTable/antigravity`, `TestVerifyIdentityDiscoveryPositive/backend_realm` |
| Pi `~/.pi/agent/sessions` or `PI_CODING_AGENT_SESSION_DIR`/`--session-dir` | THIS LEAF (default + caller-supplied pi-only override) | `TestStoreRootForTable/pi`, `TestVerifyIdentityDiscoveryPositive/pi_override`; N-bind-override-pi, N-bind-override-absolute killed |
| Qwen: no direct claim | THIS LEAF (dedicated refuse arms) | `TestStoreRootForRefusals/qwen`, `TestCheckResumeTupleRefusals/qwen_direct`; N-store-qwen, N-resume-qwen killed |
| Future plugin: every cell starts ?/disabled | THIS LEAF (unknown-provider refuse arms) | `TestStoreRootForRefusals/unknown_provider`, `TestCheckResumeTupleRefusals/unknown_provider`; N-store-unknown, N-resume-unknown killed |
| Required exclusions and materialization rules | None (stated bound: `doc.go` — exclusions have no host production) | None |

## 8.3 Capability matrix

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Per-provider capability cells; task-board/direct planes never conflated | RETAINED (`RequireCapability` direction; no cell values in code) | `TestCapabilityGatePrecedesCall` (retained, 21 tuples) |

## 8.4 Provider/platform matrix

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Codex/Claude/Gemini/Pi A and C rows pass the gate (never advertised usable) | THIS LEAF (`CheckResumeTuple`) | `TestCheckResumeTuplePassesUsableRows` (12 tuples) |
| WSL2 and native Windows never collapsed | THIS LEAF | Distinct pass rows for both (`TestCheckResumeTuplePassesUsableRows`); muse split pinned below |
| Muse/macOS-arm64 `A for probed 0.1.0` | THIS LEAF (exact pin) | Pass row 0.1.0; `TestCheckResumeTupleRefusals/muse_version_drift`; N-resume-muse-version killed (token-preserving) |
| Muse/macOS-amd64, Linux, WSL2 C rows pass (probe decides) | THIS LEAF | Pass rows for all three |
| Muse/native-Windows ? refuses | THIS LEAF | `TestCheckResumeTupleRefusals/muse_native_windows` |
| Antigravity C rows pass with realm precondition | THIS LEAF + realm binding | Pass rows; `TestVerifyIdentityDiscoveryPositive/backend_realm` |
| Qwen U direct refuses | THIS LEAF | `TestCheckResumeTupleRefusals/qwen_direct` |
| Future plugin ? refuses | THIS LEAF | `TestCheckResumeTupleRefusals/unknown_provider` |

## Appendix B Explicit provider version gates

| Clause | Owner | Test(s) |
| --- | --- | --- |
| Muse 0.2.1 store/cron/resume/import/quiesce/stop unsettled (`enabled = false`) | THIS LEAF (non-0.1.0 macOS-arm64 refuses) | `TestCheckResumeTupleRefusals/muse_version_drift`; N-resume-muse-version killed |
| Muse native-Windows behavior unsettled | THIS LEAF | `TestCheckResumeTupleRefusals/muse_native_windows` |
| Antigravity SQLite root/schema unsettled | THIS LEAF (no root invented; backend-only) | `TestStoreRootForTable/antigravity` |
| Closed-store portability C until P suites; PTY until M suites | RETAINED probe plane (`RequireCapability`) | Retained capability-gate suite |
| Every future-plugin capability ? before evidence | THIS LEAF | Unknown-provider refuse arms above |

## Coverage ratio

9 of 10 AC rows driven through the named production entries (see
`internal/provhost/TRACEABILITY.md`); the tenth is the crash-evidence
stated bound (pure entries, no durable writes). Conformance clauses:
every THIS LEAF row above carries a named committed test; RETAINED
rows keep their landed suites green; SIBLING and exclusion rows are
declared bounds, not silent gaps.
