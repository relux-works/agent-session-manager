package cloneplanning

import (
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/cloneproject"
)

// This file bridges decoded Canonical Events to planning items.
// Classification is by closed vocabularies only: the event kind
// plus the completed/aborted tool resolution and the
// encrypted/signed opaque protection. Every other payload text is
// ignored (the sweep proves adversarial payload text cannot change
// the item), and every unknown fact value refuses. The fact names
// (resolution, protection) follow cloneproject's landed registry;
// the completed/aborted spellings reference its exported statuses
// so a rename breaks the build, and the protection spellings are
// pinned against its source by the composition test.

// Item is one plannable source class: exactly one of Kind (a
// Canonical Event kind) or Block (a content-block type) is set.
// Resolution carries the folded tool resolution for tool_call and
// tool_result only; Protection carries the opaque protection for
// opaque_reasoning (required) and opaque_event (admitted, the
// unknown-native reason does not vary by it). Content blocks carry
// neither fact: blocks have no protection member.
type Item struct {
	Kind       string
	Block      string
	Resolution string
	Protection string
}

// Protection spellings, quoted from cloneproject's landed
// none|encrypted|signed envelope vocabulary.
const (
	protectionEncrypted = "encrypted"
	protectionSigned    = "signed"
)

// toolKinds admit the folded resolution fact.
var toolKinds = map[string]bool{
	"tool_call":   true,
	"tool_result": true,
}

// validResolution reports whether the value is an admitted
// resolution: absent or one of cloneproject's folded statuses.
func validResolution(value string) bool {
	return value == "" || value == cloneproject.StatusCompleted || value == cloneproject.StatusAborted
}

// validProtection reports whether the value is an admitted
// protection: absent, encrypted, or signed.
func validProtection(value string) bool {
	return value == "" || value == protectionEncrypted || value == protectionSigned
}

// requireValidItemText refuses an item carrying invalid UTF-8 in
// any text member it consumes: kind, block, resolution, or
// protection. Both planning entries apply it before the
// vocabulary gates; the vocabulary gates below then decide among
// valid strings only.
func requireValidItemText(item Item) error {
	if !validText(item.Kind) {
		return invalid("projection item kind is not valid UTF-8")
	}
	if !validText(item.Block) {
		return invalid("projection item block is not valid UTF-8")
	}
	if !validText(item.Resolution) {
		return invalid("projection item resolution is not valid UTF-8")
	}
	if !validText(item.Protection) {
		return invalid("projection item protection is not valid UTF-8")
	}
	return nil
}

// checkItem validates one caller-supplied item: exactly one class
// member set, closed kind/block vocabularies, and kind-gated fact
// admission with required facts present.
func checkItem(item Item) error {
	if (item.Kind == "") == (item.Block == "") {
		return invalid("projection item sets kind and block together or neither; exactly one class member is required")
	}
	if item.Kind != "" {
		if !clonebundle.ValidEventKind(item.Kind) {
			return invalid("projection item kind %q is not a Canonical Event kind", item.Kind)
		}
	} else {
		if !clonebundle.ValidContentBlockType(item.Block) {
			return invalid("projection item block %q is not a content-block type", item.Block)
		}
	}
	if !validResolution(item.Resolution) {
		return invalid("projection item resolution %q is outside completed|aborted", item.Resolution)
	}
	if !validProtection(item.Protection) {
		return invalid("projection item protection %q is outside encrypted|signed", item.Protection)
	}
	if item.Block != "" {
		if item.Resolution != "" {
			return invalid("projection item block carries a tool resolution; blocks carry no resolution fact")
		}
		if item.Protection != "" {
			return invalid("projection item block carries a protection fact; blocks carry no protection fact")
		}
		return nil
	}
	if toolKinds[item.Kind] {
		if item.Resolution == "" {
			return invalid("projection item kind %q requires its completed|aborted resolution", item.Kind)
		}
	} else if item.Resolution != "" {
		return invalid("projection item kind %q carries a tool resolution; only tool_call and tool_result admit one", item.Kind)
	}
	if item.Kind == "opaque_reasoning" {
		if item.Protection == "" {
			return invalid("projection item kind opaque_reasoning requires its encrypted|signed protection")
		}
	} else if item.Kind == "opaque_event" {
		return nil
	} else if item.Protection != "" {
		return invalid("projection item kind %q carries a protection fact; only opaque kinds admit one", item.Kind)
	}
	return nil
}

// ClassifyItem bridges one decoded Canonical Event to its planning
// item: the kind plus the closed resolution/protection facts when
// the landed payload carries them. Unknown fact values refuse;
// every other payload member is ignored by type (the item has no
// text field to carry it).
func ClassifyItem(event clonebundle.CanonicalEvent) (Item, error) {
	if !validText(event.Kind) {
		return Item{}, invalid("projection item kind is not valid UTF-8")
	}
	if !clonebundle.ValidEventKind(event.Kind) {
		return Item{}, invalid("projection item kind %q is not a Canonical Event kind", event.Kind)
	}
	resolution, err := closedFact(event.Payload, "resolution")
	if err != nil {
		return Item{}, err
	}
	protection, err := closedFact(event.Payload, "protection")
	if err != nil {
		return Item{}, err
	}
	item := Item{Kind: event.Kind, Resolution: resolution, Protection: protection}
	if err := requireValidItemText(item); err != nil {
		return Item{}, err
	}
	if err := checkItem(item); err != nil {
		return Item{}, err
	}
	return item, nil
}

// closedFact reads one optional closed-vocabulary payload fact. An
// absent member maps to absent; a present member must be a string,
// and its admission is decided by the caller's vocabulary gate.
func closedFact(payload map[string]any, name string) (string, error) {
	raw, present := payload[name]
	if !present || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", invalid("projection item fact %q is not a string", name)
	}
	return value, nil
}
