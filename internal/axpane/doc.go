// Package axpane implements the validation core of the
// `ax pane SESSION_ID` enforcement wrapper: the decision every managed
// pane runs before launching, reattaching, offering, parking, or
// refusing a provider.
//
// Normative scope: AX v0.7.0 Section 4.B (Terminal Backend
// Manifest/Probe identity through the landed terminalbackend registry
// and contract), Section 4.C/4.1/4.2 (the wrapper rule: validate
// configuration, load the logical session, synchronize known lease
// records when possible, compare the local fencing token before
// launching a provider, park safely when ownership is remote,
// ambiguous, or unverified; the after-restore sequence 1-5; terminal
// creation and the wrapper's first child start idempotent on
// (session_id, bootstrap_operation_id) with one receipt per pair;
// no credential-dependent tmux server creation from a background
// caller; a cached sentinel or managername observation never
// authorizes resume), Section 4.D (closed capabilities and
// evidence: only the exact admitted capability rows authorize an
// action), Section 5.2 Terminal Events (session.parked with its
// reason vocabulary, terminal.created and session.resumed v4
// terminal binding/evidence payloads), Section 5.7 (the newest
// published checkpoint through the chain fold), Section 7.A
// (Provider Protocol 3.0.0 Terminal Instance binding descriptor),
// and Section 13.1 Direct-session launch steps 3-4 (durable terminal
// entry with the bootstrap operation ID, then provider launch with
// its validated argv/env plan).
//
// Division of authority (composed, not forked): every check is
// composed from a landed gate. Owner epoch and fencing come from
// internal/fencing (AuthorizeActivation, AuthorizeRestore for the
// decision, AuthorizeMutation for every event emission, the sealed
// LeaseToken projection, the parked vocabulary); materialization
// validity from internal/matjournal (Store.Get with its grammar and
// cross-reference validation plus the committed-phase and
// fold-newest-source rules this package states) and checkpoint
// admission from internal/sessckpt (Get, which re-attests) with
// internal/sessrepo attestation (AttestCheckpointRecord) plus the
// fold-newest and session bindings this package states, and the
// one successor-lease implication (a checkpoint-carrying winner
// means the fold published a newest) asserted after fencing; the
// effective persisted profile from internal/sessprofile
// (DecodeRecord, DecodeEvent, Derive, DeriveForHeads) with Section
// 7.7 mapping refusals from internal/provhost (ResolveMapping,
// profile_mapping_unavailable); provider identity from
// internal/provhost (CheckResumeTuple, VerifyIdentityBuild,
// VerifyIdentityDiscovery) with internal/resumesmoke evidence
// (VerifyRecord) as a precondition input; backend identity from
// internal/terminalbackend (ParseManifest, ParseProbe,
// ParseEvidence, GenerationDigest, Reconcile,
// CapabilitiesForOperation via CheckOperation, CheckEntrypoint,
// AdmitProviderDescriptor); lifecycle foldability from
// internal/sessstate (DecodeRecord, DecodeEvent, Reduce over the
// §5.7 table for lifecycle foldability and the newest published
// checkpoint); configuration validity from internal/config (Load);
// session, lease, and event durability from internal/sessrepo
// (GetRecord, ListEvents, fencing.Observe over ListLeases,
// AppendEvent); refusal shapes from internal/axerror
// (NewRealmEvidenceUnavailable for the credential-conditional
// capability_unavailable with typed realm/readiness details,
// NewTargetAuthMissing for the smoke precondition).
// ParkedPayload and the evidence-ID pre-checks restate closed-shape
// predicates the canonicaljson owner re-enforces at AppendEvent:
// fail-fast input guards, not a second shape model.
//
// The pure core is Decide: exactly one of
// launch|reattach|attach_remote|takeover_offer|parked(reason)|
// refused(class). Arm order is precedence and is pinned by the
// decision table: configuration, session, bootstrap idempotency
// with the window predicate, backend identity and capability
// admission with the headless conditional, fencing (activation or
// restore entry), the successor-lease null-fold arm,
// materialization with the fold-newest source binding, checkpoint
// bound to the fold newest and session, provider identity,
// provider-auth smoke with the target binding, realm evidence,
// profile derivation from the checkpoint closure on resume and
// mapping, entrypoint, provider descriptor, then the outcome
// selection (reattach on the recorded pair, remote attach/takeover
// offer for interactive remote owners, parked in all other remote
// cases, launch).
//
// Durable state: the bootstrap binding store (binding.go) binds
// (session_id, bootstrap_operation_id) to the one recorded
// wrapper/child before any launch side effect under the landed
// no-replace + fsync discipline with crash hooks: the session's
// first binding anchors the in-window contention, and every
// recorded pair keeps its own receipt, so post-window pairs
// record alongside — never over — their superseded receipts. It
// never persists or replicates a PID: identity is the
// AX-allocated UUIDv7 terminal instance ID plus the binding
// digest. Parked decisions with a locally held, verified winner
// and a foldable lifecycle author through sessrepo.AppendEvent
// under the winning lease (events.go); remote, unverifiable, or
// unfolderable parks carry no chain event rather than
// manufacturing a collision.
//
// Bounds: no `ax` CLI surface, no tmux/ConPTY process control, no
// provider process supervision, no mesh transport, and no GUI/Keychain
// broker live here. Those are modeled callers (their facts arrive as
// Decide inputs) and later leaves. The mesh lease refresh of
// after-restore step 2 is a caller-reported fact (Verified), not a
// network call. Signature verification is caller-supplied through
// terminalbackend.SignatureVerifier.
package axpane
