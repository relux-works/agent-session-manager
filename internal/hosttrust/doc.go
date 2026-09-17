// Package hosttrust owns the machine-local Host Trust Store 1.0.0, local
// Host Credential Profile 1 issuance and custody, explicit out-of-band
// enrollment, fresh-key bounded rotation, immediate revocation, and the
// crash-durable authorization-generation transaction shared by trust changes
// and Configuration 4.0.0 peer/credential changes.
//
// Normative source: AX v0.6.0 (agent-session-manager-spec commit
// 0cbdf100dbf84df50c64f792b1f940e3a67859a6), sections 6.6, 11.10.2 and
// 11.10.3. This package covers the enrolled-trust half of those sections:
// certificate profile checks, exact leaf/root/key mapping, enrollment,
// lifecycle, generation authorization, custody and exclusion. The TLS 1.3
// launch, handshake, hello and dispatch half (sections 11.10.1, 11.10.4
// HC-TLS/HC-HELLO/HC-DISPATCH/HC-PARITY gates and live stream lifecycle)
// belongs to the RPC transport task and is a stated bound here, not a silent
// fallback: nothing in this package opens a socket or sends a byte.
//
// The store is machine-local authority, never an immutable mesh object.
// Remote RPC cannot mutate it; only the local operator path constructed here
// commits transactions.
package hosttrust
