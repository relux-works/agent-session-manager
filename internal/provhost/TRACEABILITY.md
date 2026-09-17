# TASK-260830-kp4zpu: implementation evidence

Authority: relux-works/agent-session-manager-spec v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`, adopted through
STORY-260908-18woqo. Primary scope is Sections 5.5, 7.5, 7.6, 7.7
and 8 (Sections 8.1-8.4 plus the Appendix B version gates), with
Section 2.4 carried as a sibling-owned bound below. Historical scope
is retained without weakening: v0.5.0 (commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`), same sections; the
pinned Section 8 and NativeDiscoveryProof rows are byte-identical in
both revisions. This file records implementation evidence; it changes
no normative ownership and claims no public CLI delivery.

## Acceptance rows

9 of 10 AC rows are driven through the production entries named
below; the tenth (crash evidence) is a stated bound, not a gap: the
delivered entries are pure functions that mutate no durable state, so
there is no crash-recovery surface to terminate. No `ax` session
command exists in this tree; every row is delivered at the
shared-library entries, which is the assigned surface for this leaf.

| AC row | Production call site | Named test(s) | Evidence and bound |
| --- | --- | --- | --- |
| Create exact native session IDs | `CreateIdentity` (`identity_create.go`) | `TestCreateIdentityEmitsAttestedRecord`, `TestCreateIdentityAntigravityRecord`, `TestCreatedIdentityRoundTripsIdentifyCall` | Driven: the exact param string becomes the record `native_session_id` with no normalization; created records pass `CheckIdentity`, the owner shape entry, and binding attestation, and travel one `Host.Call` identify-session round trip. A hostile params value repeating the digest placeholder still attests (`TestCreateIdentityPlaceholderDigestInParams`). |
| Validate store roots | `StoreRootFor`, `VerifyIdentityDiscovery` → `checkBindStoreRoot` (`identity_bind.go`) | `TestStoreRootForTable`, `TestStoreRootsArePairwiseDistinct`, `TestStoreRootForRefusals`, `TestVerifyIdentityDiscoveryPositive`, `TestResumeProvidersAreDerivedFromSpec` | Driven: the six Section 8.2 documented roots resolve (Muse XDG-aware, Antigravity backend-only); proof roots bind at-or-under the resolved root with a separator-aware boundary; foreign and sibling-prefix roots refuse. |
| Validate backend realms | `CreateIdentity`, `VerifyIdentityDiscovery` → `checkBindRealm` | `TestCreateIdentityRefusals` (realm rows), `TestVerifyIdentityDiscoveryPositive/backend_realm`, `TestVerifyIdentityDiscoveryRefusals` (realm rows) | Driven: digest-or-empty shape, the Antigravity backend-kind requirement, expected-realm equality, and claimed-but-unresolved refusal. |
| Validate build tuples | `CheckResumeTuple`, `VerifyIdentityBuild` (`identity_bind.go`) | `TestCheckResumeTuplePassesUsableRows`, `TestCheckResumeTupleRefusals`, `TestVerifyIdentityBuildPositive`, `TestVerifyIdentityBuildRefusals` | Driven: exact probed (provider, version, platform, arch) tuples gate resume; version drift between record and tuple refuses; the version-range member stays opaque (stated bound: no decidable range grammar in the sections). |
| Validate discovery evidence | `DecodeNativeDiscovery`, `VerifyIdentityDiscovery` | `TestDecodeNativeDiscoveryPositive`, `TestDecodeNativeDiscoveryRefusals`, `TestDiscoveryMembersAreDerivedFromSpec`, `TestVerifyIdentityDiscoveryPositive`, `TestVerifyIdentityDiscoveryRefusals` | Driven: the closed Section 7.5 proof shape decodes on the tuple platform; exact native-ID equality, reported discovery, and realm coverage bind before use. |
| Exact contract fixtures pass | `CreateIdentity`, `DecodeIdentifyResult`, `Host.Call` | `TestCreatedIdentityRoundTripsIdentifyCall`, `TestSpecIdentityExampleDecodes` (retained), `TestIdentifyThroughCall` (retained) | Driven: created records satisfy the same identify-session contract the retained Section 5.5 example satisfies, through the same transport entry. |
| Negative/refusal cases pass | All six entries | Every `Test*Refusals` table above plus the 50 witnesses in `declaredOperationWitnessesIdentityCreate` | Driven: 217/217 derived arms witnessed (was 167); every constructor site carries an exercised negative path in the full run. |
| Crash evidence | None (stated bound) | None | Bound: creation and binding are pure and stateless; `provhost` mutates no durable state of its own, so no crash/restart probe applies. Idempotency below is the applicable durability-adjacent evidence. |
| Idempotency evidence | `CreateIdentity` | `TestCreateIdentityIsByteIdentical` | Driven: identical params emit byte-identical records across 25 calls despite map-order randomization. |
| No unsupported capability advertised | `CheckResumeTuple`, `RequireCapability` (retained) | `TestCheckResumeTupleRefusals`, `TestCheckResumeTuplePassesUsableRows`, `TestCapabilityGatePrecedesCall` (retained) | Driven: unknown/unsupported/unverified-version tuples refuse; passing the matrix gate never reports usable, and conditional rows still require probe acceptance at the retained use-site gate. |

## Refusal and recovery coverage

Every `N-` row is a genuine narrowing mutant: the gate stays present
and is weakened to admit exactly one member of the class it must
reject, and the named behavioral test fails through the delivered
harness `internal/provhost/testdata/mutate_identity.py`, executed on
the exact final source. Whole-clause disables are not accepted as
narrowing. `T-` rows preserve the searched-for token while changing
behavior. Controls are reported separately and never counted as
kills: 24/24 applied N plants killed, the harmless control
`SURVIVED`, `NOT_APPLIED` and `COMPILE_OR_HARNESS_FAILURE` reported
outside the numerator.

| Gate | Real entry test | Narrowing attack (all killed) |
| --- | --- | --- |
| Creation native bounds | `TestCreateIdentityRefusals/native_empty` at `CreateIdentity` | N-create-native-empty admits exactly the empty ID (`< 1` to `< 0`). |
| Creation opaque prefix (T) | `TestCreateIdentityRefusals/opaque_absolute_prefix` at `CreateIdentity` | N-create-opaque-abs keeps the `/` token but checks the wrong end (suffix); UNC and drive forms still refuse. |
| Creation realm requirement | `TestCreateIdentityRefusals/realm_required` at `CreateIdentity` | N-create-realm-required admits exactly the fixture native ID; the owner backstop degrades the refusal. |
| Creation owner backstop | `TestCreateIdentityRefusals/owner_backstop` at `CreateIdentity` | N-create-backstop admits exactly the fixture session past `CalculateObjectIdentity`; the record then claims an empty digest. |
| Creation kind registry | `TestCreateIdentityRefusals/kind` at `CreateIdentity` | N-create-kind admits exactly `window_handle`; the owner backstop degrades the refusal. |
| Creation session shape | `TestCreateIdentityRefusals/session_id` at `CreateIdentity` | N-create-session admits exactly `not-a-uuid`; the owner backstop degrades the refusal. |
| Discovery root rule | `TestDecodeNativeDiscoveryRefusals/root_relative_admitted-token` at `DecodeNativeDiscovery` | N-discovery-root admits exactly `sessions/admitted`; every other relative root still refuses. |
| Discovery platform registry | `TestDecodeNativeDiscoveryRefusals/platform_registry` at `DecodeNativeDiscovery` | N-discovery-platform admits exactly `plan9`; the body then decodes. |
| Store registry | `TestStoreRootForRefusals/unknown_provider` at `StoreRootFor` | N-store-unknown admits exactly `futuredesk`, silently resolving the pi root. |
| Store home rule | `TestStoreRootForRefusals/home_relative` at `StoreRootFor` | N-store-home admits exactly `Users/iv`; a relative root is then resolved. |
| Store Qwen refusal | `TestStoreRootForRefusals/qwen` at `StoreRootFor` | N-store-qwen admits exactly the fixture home; the registry arm degrades the refusal. |
| Resume Qwen refusal | `TestCheckResumeTupleRefusals/qwen_direct` at `CheckResumeTuple` | N-resume-qwen admits exactly the linux tuple; the registry arm degrades the refusal. |
| Resume registry | `TestCheckResumeTupleRefusals/unknown_provider` at `CheckResumeTuple` | N-resume-unknown admits exactly `futuredesk`; the tuple then passes. |
| Resume muse pin (T) | `TestCheckResumeTupleRefusals/muse_version_drift` at `CheckResumeTuple` | N-resume-muse-version keeps the pinned `0.1.0` token while admitting exactly `0.2.1`; every other version still refuses. |
| Build drift | `TestVerifyIdentityBuildRefusals/version_drift` at `VerifyIdentityBuild` | N-build-drift admits exactly the `0.148.0` drifted build. |
| Discovery native equality | `TestVerifyIdentityDiscoveryRefusals/native_id` at `VerifyIdentityDiscovery` | N-bind-native admits exactly the foreign fixture ID; the proof then binds. |
| Discovery presence | `TestVerifyIdentityDiscoveryRefusals/evidence_absent` at `VerifyIdentityDiscovery` | N-bind-absent admits exactly an undiscovered proof that still carries a root. |
| Root containment | `TestVerifyIdentityDiscoveryRefusals/root_sibling_prefix` at `VerifyIdentityDiscovery` | N-bind-contains admits exactly the sibling-prefix root. |
| Realm resolution | `TestVerifyIdentityDiscoveryRefusals/realm_unresolved` at `VerifyIdentityDiscovery` | N-bind-unresolved admits exactly the fixture realm; the proof then binds. |
| Override provider rule | `TestVerifyIdentityDiscoveryRefusals/override_pi-only` at `VerifyIdentityDiscovery` | N-bind-override-pi admits exactly codex; the containment arm degrades the refusal. |
| Override absolute rule | `TestVerifyIdentityDiscoveryRefusals/override_absolute` at `VerifyIdentityDiscovery` | N-bind-override-absolute admits exactly `custom/relative`; the containment arm degrades the refusal. |
| Root presence | `TestVerifyIdentityDiscoveryRefusals/root_missing` at `VerifyIdentityDiscovery` | N-bind-root-missing admits exactly a discovered proof with a null root; the containment arm degrades the refusal. |
| Realm shape | `TestVerifyIdentityDiscoveryRefusals/realm_shape` at `VerifyIdentityDiscovery` | N-bind-realm-shape admits exactly `nope`; the equality arm degrades the refusal. |
| Realm equality | `TestVerifyIdentityDiscoveryRefusals/realm_mismatch` at `VerifyIdentityDiscovery` | N-bind-realm-mismatch admits exactly the drifted fixture realm; the unresolved arm degrades the refusal. |

## Bounds

- Section 2.4 profile authority (Session Record creation value,
  `profile.changed` derivation, the `profile_mapping_unavailable`
  resume refusal) belongs to sibling TASK-260830-3uzfyn
  (profile resolution/persistence); this leaf contributes the
  identity records those flows carry and refuses nothing on profile
  grounds.
- The `provider_version_range` member is carried opaquely and
  equality on the exact probed version is the enforced build rule:
  the sections declare no decidable range grammar to evaluate.
- Only the Muse macOS arm64 cell carries a version pin
  (`A for probed 0.1.0`); exact-version acceptance for every other
  provider stays the probe plane's decision through
  `RequireCapability`, never this gate's.
- Discovery-root binding is at-or-under with a separator-aware byte
  comparison, so project-keyed subpaths count as store discovery
  while sibling prefixes refuse. Windows case-insensitivity is out
  of scope: comparison is byte-exact and fails closed on case drift.
- Provider-identity binding attestation stays where the retained
  `TestNoProductionPathAttestsProviderIdentityBinding` bound keeps
  it: no production path outside `internal/sessrepo` calls
  `VerifyObjectIdentity`. Creation computes digests with
  `CalculateObjectIdentity`; only tests attest them.
- No CLI, doctor surface, or capability advertisement is added or
  changed: `ax` operator surfaces belong to their owning leaves.
