# STORY-260830-14qxuc: resumable-blob-and-manifest-transfer

## Description
Implement transfer checkpoint closure and large blobs safely with resumable staging according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §10.2-10.4, §11.5. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Transfer checkpoint closure and large blobs safely with resumable staging is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
