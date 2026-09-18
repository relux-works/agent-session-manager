package clonebundle

import (
	"encoding/json"
	"io"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file owns the Section 10.2 raw evidence descriptor path: Blob
// Descriptor identity and byte-count agreement, and no-replace blob
// installation through the landed localstore durable-write
// discipline. This package performs no durable write itself: every
// byte crosses localstore.ObjectStore.PutBlob, whose no-replace,
// fsync, and crash/idempotency evidence is cited in TRACEABILITY.md.

// BlobAgreement is the verified agreement between a manifest entry
// and its Section 10.2 Blob Descriptor.
type BlobAgreement struct {
	DescriptorID scalar.Digest
	BlobID       scalar.Digest
	Size         uint64
}

// VerifyDescriptorAgreement verifies one raw evidence descriptor:
// the bytes must be a closed valid Blob Descriptor 1.0.0 whose
// claimed identity equals the computed omit-self digest, the
// computed identity must equal the manifest entry's
// blob_descriptor_id claim, and the descriptor's blob ID and size
// must agree with the entry's claim. The identity link is what
// binds the entry to the exact descriptor bytes verified: without
// it any descriptor with an agreeing blob ID and size would satisfy
// the entry. Shape validation and digest computation run through
// the canonicaljson owner, never a fork; the claim comparisons are
// local, because binding attestation outside the session leaf is
// refused by the provhost no-attestation bound, so this package
// never calls the attesting entry.
func VerifyDescriptorAgreement(descriptor []byte, wantDescriptorID, wantBlobID scalar.Digest, wantSize uint64) error {
	calculated, claimed, members, err := calculateDescriptorIdentity(descriptor)
	if err != nil {
		return err
	}
	if calculated != claimed {
		return invalid("blob descriptor claim %q does not match omit-self digest %q", claimed.String(), calculated.String())
	}
	if calculated != wantDescriptorID {
		return invalid("blob descriptor identity %q disagrees with entry claim %q", calculated.String(), wantDescriptorID.String())
	}
	blobID, ok := checkDigest(members["blob_id"])
	if !ok {
		return invalid("blob descriptor blob_id is not a digest")
	}
	if blobID != wantBlobID {
		return invalid("blob descriptor blob_id %q disagrees with entry claim %q", blobID.String(), wantBlobID.String())
	}
	size, ok := checkUint53Bounds(members["size"], 0, maxUint53)
	if !ok {
		return invalid("blob descriptor size is not a uint53")
	}
	if size != wantSize {
		return invalid("blob descriptor size %d disagrees with entry claim %d", size, wantSize)
	}
	return nil
}

// InstallRawBlob installs one raw evidence blob through the landed
// durable-write discipline: fsync, size/digest verification, and
// atomic no-replace install owned by localstore. It returns the
// verified agreement. A digest collision or same-ID/different-bytes
// observation quarantines through the store owner; this package
// adds no second write path.
func InstallRawBlob(store *localstore.ObjectStore, descriptor []byte, source io.Reader) (BlobAgreement, error) {
	calculated, claimed, members, err := calculateDescriptorIdentity(descriptor)
	if err != nil {
		return BlobAgreement{}, err
	}
	if calculated != claimed {
		return BlobAgreement{}, invalid("blob descriptor claim %q does not match omit-self digest %q", claimed.String(), calculated.String())
	}
	blobID, ok := checkDigest(members["blob_id"])
	if !ok {
		return BlobAgreement{}, invalid("blob descriptor blob_id is not a digest")
	}
	size, ok := checkUint53Bounds(members["size"], 0, maxUint53)
	if !ok {
		return BlobAgreement{}, invalid("blob descriptor size is not a uint53")
	}
	if store == nil {
		return BlobAgreement{}, invalid("blob install requires an object store")
	}
	if _, err := store.PutBlob(blobID, size, source); err != nil {
		return BlobAgreement{}, invalid("blob install refused: %v", err)
	}
	return BlobAgreement{DescriptorID: calculated, BlobID: blobID, Size: size}, nil
}

// calculateDescriptorIdentity runs the owner's shape validation and
// omit-self computation over one descriptor and reads its claimed
// self digest. Callers compare the two; this helper never attests.
func calculateDescriptorIdentity(descriptor []byte) (scalar.Digest, scalar.Digest, map[string]json.RawMessage, error) {
	calculated, field, err := canonicaljson.CalculateObjectIdentity(descriptor)
	if err != nil {
		return scalar.Digest{}, scalar.Digest{}, nil, invalid("blob descriptor identity fails: %v", err)
	}
	if field != "descriptor_id" {
		return scalar.Digest{}, scalar.Digest{}, nil, invalid("blob descriptor self field is %q, want descriptor_id", string(field))
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(descriptor, &members); err != nil {
		return scalar.Digest{}, scalar.Digest{}, nil, invalid("blob descriptor is not an object: %v", err)
	}
	claimed, ok := checkDigest(members["descriptor_id"])
	if !ok {
		return scalar.Digest{}, scalar.Digest{}, nil, invalid("blob descriptor descriptor_id is not a digest")
	}
	return calculated, claimed, members, nil
}
