package cloneproject

// This file carries the 64 KiB message-like content rule: inline
// text at or under the bound seals inline; anything larger becomes a
// Blob Descriptor reference to an installed overflow blob. Oversized
// content is never truncated, summarized, or refused: the bytes move
// to the blob store and the reference resolves back to them exactly.

import (
	"bytes"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonesnap"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// maxInlineContentBytes is the Section 13.14.1 inline content bound:
// inline content is at most 64 KiB, measured in bytes exactly as the
// landed content-block gate measures it.
const maxInlineContentBytes = 65536

// overflowMediaType labels overflow text blobs.
const overflowMediaType = "text/plain"

// OverflowBlob is one installed overflow blob with its verified
// identities.
type OverflowBlob struct {
	// BlobID is the SHA-256 of the content bytes.
	BlobID scalar.Digest
	// DescriptorID is the omit-self identity of the descriptor the
	// content block references.
	DescriptorID scalar.Digest
	// Size is the content length.
	Size uint64
}

// installOverflow installs oversized text through the landed
// discipline: the descriptor seals through
// clonesnap.BuildBlobDescriptor, the payload installs through
// clonebundle.InstallRawBlob (identity plus blob-ID/size agreement),
// and the descriptor bytes install under their content digest so the
// reference resolves. Both installs are content-addressed and
// idempotent: a replay verifies and reuses.
func installOverflow(sink *localstore.ObjectStore, text string) (OverflowBlob, error) {
	payload := []byte(text)
	descriptor, err := clonesnap.BuildBlobDescriptor(payload)
	if err != nil {
		return OverflowBlob{}, invalid("content overflow cannot seal its blob descriptor: %v", err)
	}
	if _, err := clonebundle.InstallRawBlob(sink, descriptor.Descriptor, bytes.NewReader(payload)); err != nil {
		return OverflowBlob{}, invalid("content overflow cannot install its payload: %v", err)
	}
	contentID := scalar.SHA256Digest(descriptor.Descriptor)
	if _, err := sink.PutBlob(contentID, uint64(len(descriptor.Descriptor)), bytes.NewReader(descriptor.Descriptor)); err != nil {
		return OverflowBlob{}, invalid("content overflow cannot install its descriptor: %v", err)
	}
	return OverflowBlob{BlobID: descriptor.BlobID, DescriptorID: descriptor.DescriptorID, Size: descriptor.Size}, nil
}

// contentBlockForText builds one text content block: inline content
// at or under 64 KiB, a Blob Descriptor reference above it.
func contentBlockForText(text string, sink *localstore.ObjectStore) (map[string]any, []OverflowBlob, error) {
	if len(text) > maxInlineContentBytes {
		installed, err := installOverflow(sink, text)
		if err != nil {
			return nil, nil, err
		}
		return map[string]any{
			"type":               "text",
			"blob_descriptor_id": installed.DescriptorID.String(),
			"media_type":         overflowMediaType,
		}, []OverflowBlob{installed}, nil
	}
	return map[string]any{
		"type":    "text",
		"content": text,
	}, nil, nil
}
