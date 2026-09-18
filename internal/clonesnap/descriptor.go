package clonesnap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file builds Section 10.2 Blob Descriptors for captured store
// bytes. The descriptor shape and identity verify through the
// canonicaljson owner; the blob-ID/size agreement re-checks through
// the landed clonebundle entry. No descriptor this package seals is
// ever installed without that agreement.

const (
	// blobMediaType is the media type of captured store bytes:
	// uninterpreted payload octets.
	blobMediaType = "application/octet-stream"
	// blobChunkSize is the Section 10.2 transfer unit: every chunk
	// is 4 MiB except the last.
	blobChunkSize = 4194304
	// blobMaxChunks is the Section 10.2 descriptor chunk maximum:
	// 32768 chunks cap one blob at 128 GiB.
	blobMaxChunks = 32768
	// DefaultMaxSingleBytes is the largest single store member this
	// package captures: 32768 chunks of 4194304 bytes, exactly
	// 137438953472. A larger member fails capture with
	// capability_unavailable before any manifest is published.
	DefaultMaxSingleBytes = uint64(blobMaxChunks) * uint64(blobChunkSize)
)

// BlobDescriptor is one sealed Section 10.2 Blob Descriptor with its
// verified identities.
type BlobDescriptor struct {
	// Descriptor carries the exact sealed descriptor bytes the raw
	// entry references.
	Descriptor []byte
	// BlobID is the SHA-256 of the payload bytes.
	BlobID scalar.Digest
	// DescriptorID is the omit-self identity of Descriptor.
	DescriptorID scalar.Digest
	// Size is the payload length.
	Size uint64
}

// BuildBlobDescriptor seals one Section 10.2 Blob Descriptor over
// the payload: sorted contiguous non-overlapping chunks from offset
// zero covering exactly the payload, indexes from zero by one, an
// empty payload sealing with no chunks. Chunk digests cover raw
// chunk bytes, so a single-chunk blob's chunk digest equals the
// whole-blob digest, exactly as the Section 10.2 example.
func BuildBlobDescriptor(payload []byte) (BlobDescriptor, error) {
	size := uint64(len(payload))
	if size > DefaultMaxSingleBytes {
		return BlobDescriptor{}, invalid("capture member of %d bytes exceeds the 128 GiB blob bound: capability_unavailable", size)
	}
	blobSum := sha256.Sum256(payload)
	blobID, err := scalar.ParseDigest("sha256:" + hex.EncodeToString(blobSum[:]))
	if err != nil {
		return BlobDescriptor{}, invalid("capture blob digest is not a digest: %v", err)
	}
	chunks := make([]any, 0, 1)
	for offset := uint64(0); offset < size; {
		length := uint64(blobChunkSize)
		if remaining := size - offset; remaining < length {
			length = remaining
		}
		chunkSum := sha256.Sum256(payload[offset : offset+length])
		chunks = append(chunks, map[string]any{
			"index":    len(chunks),
			"offset":   offset,
			"size":     length,
			"chunk_id": "sha256:" + hex.EncodeToString(chunkSum[:]),
		})
		offset += length
	}
	object := map[string]any{
		"schema":         "urn:ax:schema:blob",
		"schema_version": "1.0.0",
		"descriptor_id":  "sha256:" + "0000000000000000000000000000000000000000000000000000000000000000",
		"blob_id":        blobID.String(),
		"size":           size,
		"media_type":     blobMediaType,
		"chunks":         chunks,
	}
	plain, err := json.Marshal(object)
	if err != nil {
		return BlobDescriptor{}, invalid("serialize blob descriptor: %v", err)
	}
	calculated, field, err := canonicaljson.CalculateObjectIdentity(plain)
	if err != nil {
		return BlobDescriptor{}, invalid("blob descriptor identity fails: %v", err)
	}
	if string(field) != "descriptor_id" {
		return BlobDescriptor{}, invalid("blob descriptor self field is %q, want descriptor_id", string(field))
	}
	object["descriptor_id"] = calculated.String()
	sealed, err := json.Marshal(object)
	if err != nil {
		return BlobDescriptor{}, invalid("serialize blob descriptor: %v", err)
	}
	if err := clonebundle.VerifyDescriptorAgreement(sealed, calculated, blobID, size); err != nil {
		return BlobDescriptor{}, invalid("sealed blob descriptor disagrees: %v", err)
	}
	return BlobDescriptor{Descriptor: sealed, BlobID: blobID, DescriptorID: calculated, Size: size}, nil
}
