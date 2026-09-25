package cloneplanning

import (
	"encoding/json"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file projects checkpoint visible text: typed text fields
// are escaped per field and joined, and the projection authority
// is one constant. Authority is decided by type, never by text:
// ProjectVisibleText assigns Authority exactly once from
// authorityUserContext with no condition on the text, kind,
// ordinal, or event IDs, and the structural test pins that shape
// with control plants. Escaped output round-trips through
// encoding/json to exactly the input fields, so no instruction,
// reply, control token, or authorization is added or interpreted.

// authorityUserContext is the only visible-projection authority
// Section 13.14.2 admits: VisibleMigrationProjection carries
// exactly authority=user_context, never an assistant reply, system
// instruction, or authorization.
const authorityUserContext = "user_context"

// maxEscapedText is the escaped_text character bound Section
// 13.14.2 states: string[1..65536], measured in characters.
const maxEscapedText = 65536

// maxVisibleEventIDs bounds the target event IDs Section 13.14.2
// states: sorted unique digest[1..64].
const maxVisibleEventIDs = 64

// VisibleInput is one checkpoint visible-projection candidate: the
// carrying event kind, the event ordinal, the extracted typed text
// fields, and the shown target event IDs. Per-kind text-field
// extraction is caller scope; this entry takes the extracted texts.
type VisibleInput struct {
	Kind     string
	Ordinal  uint64
	Texts    []string
	EventIDs []string
}

// VisibleProjection is one projected checkpoint visible text: its
// low authority, the event ordinal, the per-field escaped text,
// and the shown target event IDs.
type VisibleProjection struct {
	Authority   string
	Ordinal     uint64
	EscapedText string
	EventIDs    []string
}

// EscapeVisibleText escapes one typed text field as a JSON string
// literal. The field must be valid UTF-8 first: encoding/json
// would otherwise rewrite invalid bytes as U+FFFD and succeed,
// returning a value whose text differs from its input. The output
// parses through encoding/json to exactly the input; control
// characters, quotes, and markup delimiters never cross raw.
func EscapeVisibleText(field string) (string, error) {
	if !validText(field) {
		return "", invalid("visible text field is not valid UTF-8")
	}
	sealed, err := json.Marshal(field)
	if err != nil {
		return "", invalid("visible text field does not escape: %v", err)
	}
	return string(sealed), nil
}

// ProjectVisibleText projects one checkpoint visible text. The
// kind must be a Canonical Event kind, the ordinal a uint53, every
// text field valid UTF-8, the joined escaped text within
// string[1..65536] characters, and the event IDs sorted unique
// digests[1..64]. The authority is always user_context.
func ProjectVisibleText(input VisibleInput) (VisibleProjection, error) {
	if !validText(input.Kind) {
		return VisibleProjection{}, invalid("visible projection kind is not valid UTF-8")
	}
	if !clonebundle.ValidEventKind(input.Kind) {
		return VisibleProjection{}, invalid("visible projection kind %q is not a Canonical Event kind", input.Kind)
	}
	if _, err := scalar.NewUint53(input.Ordinal); err != nil {
		return VisibleProjection{}, invalid("visible projection ordinal %d is not a uint53: %v", input.Ordinal, err)
	}
	for index, field := range input.Texts {
		if !validText(field) {
			return VisibleProjection{}, invalid("visible projection text field[%d] is not valid UTF-8", index)
		}
	}
	escaped := make([]string, 0, len(input.Texts))
	for _, field := range input.Texts {
		quoted, err := EscapeVisibleText(field)
		if err != nil {
			return VisibleProjection{}, err
		}
		escaped = append(escaped, quoted)
	}
	joined := strings.Join(escaped, "\n")
	if length := environ.StringLength(joined); length < 1 || length > maxEscapedText {
		return VisibleProjection{}, invalid("visible projection escaped_text carries %d characters, the checkpoint requires string[1..65536]", length)
	}
	ids, err := checkVisibleEventIDs(input.EventIDs)
	if err != nil {
		return VisibleProjection{}, err
	}
	return VisibleProjection{
		Authority:   authorityUserContext,
		Ordinal:     input.Ordinal,
		EscapedText: joined,
		EventIDs:    ids,
	}, nil
}

// checkVisibleEventIDs validates the shown target event IDs:
// sorted unique digests[1..64]. Digest grammar delegates to the
// scalar owner; order and count are this entry's composition.
func checkVisibleEventIDs(ids []string) ([]string, error) {
	if len(ids) < 1 || len(ids) > maxVisibleEventIDs {
		return nil, invalid("visible projection carries %d target event IDs, the checkpoint requires sorted unique digest[1..64]", len(ids))
	}
	checked := make([]string, 0, len(ids))
	for index, id := range ids {
		if !validText(id) {
			return nil, invalid("visible projection target event ID[%d] is not valid UTF-8", index)
		}
		digest, err := scalar.ParseDigest(id)
		if err != nil {
			return nil, invalid("visible projection target event ID[%d] is not a digest: %v", index, err)
		}
		checked = append(checked, digest.String())
	}
	for index := 1; index < len(checked); index++ {
		if checked[index-1] >= checked[index] {
			return nil, invalid("visible projection target event IDs are not sorted unique")
		}
	}
	return checked, nil
}
