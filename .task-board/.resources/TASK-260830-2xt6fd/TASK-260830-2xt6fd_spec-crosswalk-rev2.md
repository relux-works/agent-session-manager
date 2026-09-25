# Specification crosswalk — TASK-260830-2xt6fd

The task started against agent-session-manager-spec v0.5.0 at
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`. The active implementation pin is
`internal/specdoc/SPEC.v0.7.0.md` (SHA-256
`c6b2fe64ee79ed697a96ed27a1679c80b8ee1eba137feb99e7738b4da289ddcf`). The
v0.5.0 source was read from the pinned commit (SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`).

The table compares each section body byte-for-byte, including its heading and
line endings. These cited clauses did not move, split, or change text between
the two spec versions; only surrounding document line numbers shifted.

| v0.5.0 section and lines | v0.7.0 section and lines | Body SHA-256 | Result |
| --- | --- | --- | --- |
| §4.C Lifecycle and operations, 1046–1183 | §4.C Lifecycle and operations, 1082–1219 | `de7598b2dd14ab86501de272e156043b91e0caec84e70408129ba9f8c860a168` | Byte-identical |
| §4.1 Terminal backend interface, 1338–1366 | §4.1 Terminal backend interface, 1374–1402 | `088e7346b19dc121297fb35258f5c207d49a5d97577f46e0c36eb8e9b8ec764b` | Byte-identical |
| §4.2 tmux backend, 1367–1413 | §4.2 tmux backend, 1403–1449 | `b2abde19c8528bec4463cda2e7b35ce776b89dfe90c6bd4baef8d4a7d4be5bff` | Byte-identical |
| §10.2 Content-addressed blobs, 4600–4653 | §10.2 Content-addressed blobs, 4851–4904 | `b778d81ae961488d3bc55f59729348ad41fab20a3078928041d1f87abe5ec052` | Byte-identical |
| §10.3 Transfer Chunk Descriptor, 4654–4681 | §10.3 Transfer Chunk Descriptor, 4905–4932 | `9074470e15affd590d5c564ff1b95ad12d4465d1e23927c7c4ee572702911111` | Byte-identical |
| §10.4 Transfer Manifest, 4682–5216 | §10.4 Transfer Manifest, 4933–5467 | `d3ddc67727369b71f346b7449418278c570327e5f53e0f6b6f2b329977bff231` | Byte-identical |
| §12.1 Workspace snapshot, 7939–7965 | §12.1 Workspace snapshot, 9020–9046 | `b8e7f024341a902ce4a0ca92f5e5bb7ed36e92e0696ae56c7f9bc48b77ddefd1` | Byte-identical |
| §12.2 Git workspace schema, 7966–7992 | §12.2 Git workspace schema, 9047–9073 | `71700c8b7507570f11684d9035ee58ac03d2141c2451e8e95b8c2a333cf49178` | Byte-identical |
| §12.3 Git capture and materialization, 7993–8027 | §12.3 Git capture and materialization, 9074–9108 | `f89db6bcc3e6bf6a3db9dde4ef837d2dfae65cc7512426796134cacc1a3bc282` | Byte-identical |

## Capture and held-input decision

- §4.C v0.7.0 lines 1082–1087 define the closed eight-state domain. Lines
  1098–1115 state that only `quiesce-input` enters `quiescing` and the lifecycle
  is closed. The operation transitions are at lines 1200–1211: `quiesce-input`
  admits `active|parked → quiescing`, `wait-safe-boundary` remains
  `quiescing → quiescing`, `request-stop` transitions `quiescing → stopped`,
  and `restore` accepts only `absent|stopped|unavailable → parked`.
- §4.1 v0.7.0 lines 1374–1388 list the semantic interface operations, including
  input closure, safe-boundary wait, and stop. §4.2 lines 1403–1449 describe the
  tmux implementation. They add no release transition. The durable state model
  and transition table are in §4.C; the interface summary is §4.1.
- §12.3 v0.7.0 lines 9076–9078 requires quiescing agent input and
  filesystem-mutating provider work, consistent Git reads, and refusal if HEAD,
  index, or included file digests change. This is the stop-point capture
  contract: there is no legal transition back to `active` in the same
  incarnation. A failed/crashed capture remains quiescing and uses the landed
  lifecycle owner's stop or stale-termination recovery path before retry.
- §12.1 lines 9022–9045 defines complete workspace-group closure and its
  required captured state. §12.2 lines 9047–9072 defines the Git member schema,
  sanitized configuration, and parser-before-destination requirements. §12.3
  lines 9080–9108 describe materialization and the staged/index versus working
  byte distinction; destination materialization is outside this leaf.
- §10.2 lines 4851–4904 define Blob Descriptor and the 128 GiB refusal; §10.3
  lines 4905–4932 define fixed 4 MiB chunks; §10.4 lines 4933–5467 define
  Transfer Manifest, Git workspace members, raw/logical indexes, child closure,
  and content path semantics. The 25 numbered §10.4 clauses are inventoried in
  `.temp/TASK-260830-2xt6fd/section-clause-inventory-01.log`; the v0.7.0
  registry records the 21 clauses this capture/assembly leaf discharges and
  declares the other four as out of contract.

The resume decision's reference to §4.2 is interpreted with the spec's exact
numbering: quiescence operations are specified in §4.C and summarized in §4.1;
§4.2 is the tmux backend subsection. All three section bodies are byte-identical
to v0.5.0, and none authorizes a release operation.
