# Review verdict — TASK-260830-kp4zpu revision 1: ACCEPT

Reviewer run: independent verification of Change Request `CR-TASK-260830-kp4zpu-1`
revision 1 (base `62d4463`, candidate tree `144f2b3`, 13 changed paths).
Verdict: **ACCEPT**. No P1/P2/P3 findings. Reasoning and evidence below;
raw logs are in `TASK-260830-kp4zpu_review-evidence-rev1.tar.gz`.

Authority note: judged against the pinned `internal/specdoc/SPEC.v0.6.0.md`
(§2.4, §5.5, §7.5–7.7, §8, Appendix B), not the task text's v0.5.0
numbering. The producer's byte-identity claim was re-verified by the
reviewer: the §8.2 body rows (v0.5.0 lines 3865–3872 vs v0.6.0 lines
3931–3938) diff byte-identical, and the `NativeDiscoveryProof` type row
(v0.5.0 line 2893 vs v0.6.0 line 2959) is the identical line. The new
derivation tests pin the embedded digest-verified v0.5.0 document at the
correct lines, so they bind the same rows the v0.6.0 authority names —
not test literals.

## 1. Spec fidelity (reviewer-extracted, cell by cell)

- `discoveryMembers` = {native_session_id, discovered, discovery_root,
  backend_resolved} — exact match to the §7.5 `NativeDiscoveryProof` row;
  `TestDiscoveryMembersAreDerivedFromSpec` reads the pinned row
  (4/4 members derived, observed green).
- `resumeProviders` = [codex, claude, gemini, muse, antigravity, pi] in
  table order; lines 3871–3872 (Qwen, Future plugin) asserted excluded
  (6/6 providers derived, observed green).
- `StoreRootFor` (`identity_bind.go`): codex `~/.codex/sessions`, claude
  `~/.claude/projects`, muse `$XDG_DATA_HOME/muse/sessions` with
  `~/.local/share` default, antigravity backend-only (no root invented),
  pi default plus pi-only override, qwen/unknown refused. One disclosed
  approximation (see Notes): gemini resolves the documented base
  `~/.gemini`; the per-project hash stays adapter-owned and binding is
  at-or-under, so no invented hash and no verbatim-key copying.
- `CheckResumeTuple`: unknown providers/platforms refuse, Qwen direct
  refuses, Muse native-Windows `?` refuses, Muse macOS-arm64 admits only
  exact `0.1.0` (Appendix B `enabled = false` for the rest); WSL2 and
  native Windows are distinct rows. Passing never reports usable (separate
  `RequireCapability` plane, retained suite green).
- Created records are accepted by the landed
  `CheckIdentity`/`DecodeIdentifyResult` through `Host.Call`
  (`TestCreatedIdentityRoundTripsIdentifyCall` PASS, observed) and are
  byte-identical across repeated calls
  (`TestCreateIdentityIsByteIdentical` PASS, observed); the omit-self
  `record_id` goes through the owner `CalculateObjectIdentity`, and the
  frame substitution matches the full `"record_id":"…"` frame so a param
  repeating the placeholder digest cannot misdirect it (dedicated test
  present and green).

## 2. Refusal census and witness-instrument attacks

- 50 new arms in `declaredOperationWitnessesIdentityCreate`
  (`refusal_arm_operations_d_test.go`): 19 × `CreateIdentity`, 7 ×
  `DecodeNativeDiscovery`, 6 × `StoreRootFor`, 8 × `CheckResumeTuple`,
  1 × `VerifyIdentityBuild`, 9 × `VerifyIdentityDiscovery` — each naming
  the production entry with exact code/detail. Observed:
  `refusal arm coverage: 217/217 derived arms witnessed`
  (floor 166 → 217); `vocabulary census coverage: 24/24`.
- Witness attacks in an isolated copy (live tree never written):
  - A — swapped conditions of two adjacent `invalid_config` arms
    (session_id ↔ provider_id): suite RED via
    `TestCreateIdentityRefusals`.
  - B — one-token swallow (`== ""` → `!= ""` on the Antigravity
    realm arm): suite RED via
    `TestCreateIdentityRefusals/realm_required`.
  - C — deleted the `StoreRootFor` qwen arm: suite RED via
    `TestStoreRootForRefusals/qwen` and
    `TestDerivedRefusalArmsAreAllWitnessed`.
- Coverage ratio: **50 of 50 new arms driven; 217 of 217 derived arms
  witnessed**, call sites named per arm in the witness table.

## 3. Own behavioral plants (isolated copy, production entries)

| Plant | Result |
| --- | --- |
| Provider `Codex` (case change) via Create/Resume/Store | REFUSED `invalid_config` ×3 |
| Proof root with trailing slash | REFUSED `provider_protocol_error` (decode rejects; fail-closed) |
| Symlinked/unresolved root | REFUSED `invalid_config` outside-store |
| Same version, different arch (codex arm64→amd64) | ADMITTED (no arch pin; version equality is the rule) |
| Discovery extra / missing / wrong-type member | REFUSED `provider_protocol_error` with member ×3 |
| Muse macos-arm64 `0.1.0` | ADMITTED |
| Muse `0.1.0-rc1`, `0.1.1`, `0.10.0` | REFUSED `invalid_config` unverified-version ×3 |
| Opaque value with trailing `/` | ADMITTED (rule is absolute-prefix only, per spec wording) |

## 4. Mutation

- Producer harness `testdata/mutate_identity.py` rerun by the reviewer on
  the exact final source, twice: **24/24 applied N plants KILLED in both
  runs, classifications identical** (deterministic); harmless control
  SURVIVED; `NOT_APPLIED` and `COMPILE_OR_HARNESS_FAILURE` outside the
  numerator. Includes the two token-preserving plants (opaque `/`
  suffix-swap, muse `0.1.0` admission).
- Four own narrowing mutants on producer-unmutated arms, same harness
  discipline with per-plant logs: opaque count `>32→>33` KILLED by
  `TestCreateIdentityRefusals/opaque_count`; resume arch admit-`mips`
  KILLED by `…/architecture_registry`; discovery native `<1→<0` KILLED
  by `…/native_bounds`; store empty-home admit-codex KILLED by
  `…/home_empty` (degraded-detail kill). Applied harmless comment
  control SURVIVED. **4 of 4 own narrowing mutants killed.**

## 5. Bounds

- `VerifyObjectIdentity` production call sites: only
  `internal/sessrepo` (checkpoint, chain ×3, lease) plus the
  `canonicaljson` definition — retained bound holds (whole-module grep).
- `reverseDNSPattern`: provhost copy differs from the config copy only
  by `(…)` vs `(?:…)` — same accepted language by construction; ledger
  row present and `TestSharedGrammarsAreOneLanguage` green. True
  one-language registration, not a diverging copy.
- §2.4: zero `profile` references in either new production file —
  sibling-owned, not partially implemented here.
- Crash evidence as stated bound is legitimate: the new entries import
  only bytes/encoding-json/regexp/sort/strings/filepath plus
  canonicaljson/scalar; grep for writes/exec/SQL shows no durable-state
  mutation. Purity is the applicable durability evidence, with
  byte-identical idempotency proven by test.

## 6. Docs, hygiene, gates (real exits, this review)

- README operation-layer sentence and LOGBOOK entry: no
  capability/parity claim (read in full).
- Changed paths: exactly the 13 candidate paths; `gofmt` clean; no
  `__pycache__`/`.pyc`; no `task-board.config.json`; no stray plants.
- `go test ./internal/provhost/ ./internal/environ/ -count=1`: exit 0.
- Same with `-race`: exit 0. `go vet`: exit 0. `tracecheck`: ok
  (contracts=63, sections=36, cases=101). `cataloggen -check`: exit 0.
- Method: all attacks/plants/mutants ran in scratch copies under
  `.temp/TASK-260830-kp4zpu/`, deleted after log capture; the live Story
  worktree was never written by the reviewer (status shows only the
  candidate paths, HEAD still `62d4463`, no commits).

## Notes (non-blocking)

- Gemini resolves the `~/.gemini` base rather than the full
  `~/.gemini/tmp/<hash>/chats` root; disclosed in the conformance
  matrix with the hash left adapter-owned. Acceptable: the alternative
  invents an underivable path component, and at-or-under containment
  keeps every proof under the documented root binding.
- AC ratio: **9 of 10 rows driven through named production entries**;
  the tenth (crash evidence) is the purity-argued stated bound above.

Acceptance: `task-board m 'accept_cr(TASK-260830-kp4zpu, revision=1,
evidence=TASK-260830-kp4zpu_review-verdict-rev1.md)'`.
