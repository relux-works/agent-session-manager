package termbind

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Peers replays every recorded presentation-client receipt for one
// terminal instance except the requesting client itself: the stored
// peer census the attach overlap gate fails closed on. A valid receipt
// is positive evidence of an admitted client claim, not evidence that
// its tmux client is still connected. This protocol has no positive
// identity join or detach signal, so an unproven liveness state stays
// UNKNOWN and remains a possible overlap. The requesting client is
// excluded by identity: a retry replays its own receipt rather than
// adding a client.
//
// An unrecorded instance has no peers. A read or validation failure
// is an error, never an empty census: unknown is not a verdict.
// Staging entries (attach-*) and non-receipt names (anything without
// the .json suffix) are skipped; every other entry sits inside the
// receipt namespace and admits only as a valid peer receipt for its
// own durable key. The admission is positive, through the keyed-read
// owner: the filename stem must parse as a client identity and the
// Lookup of that key must return the receipt. Anything else in the
// receipt namespace — a directory at a receipt-shaped name, valid
// bytes filed under another client's name, an unparseable stem —
// is corruption and errors, never a skip, which would read a
// corrupt census as no peers. Every decoded receipt must name this
// session and instance, like Lookup. Peers arrive sorted by client
// ID.
func (store *AttachStore) Peers(sessionID, instanceID, clientID string) ([]AttachReceipt, error) {
	if _, err := scalar.ParseUUIDv7(sessionID); err != nil {
		return nil, fmt.Errorf("attach peers session %q is not a UUIDv7: %w", sessionID, err)
	}
	if err := CheckInstanceIdentity(instanceID); err != nil {
		return nil, err
	}
	if err := CheckInstanceIdentity(clientID); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(store.instanceDir(instanceID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read attach peers: %w", err)
	}
	var peers []AttachReceipt
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "attach-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		if entry.IsDir() {
			return nil, fmt.Errorf("attach peer %q is not a receipt file", name)
		}
		stem := strings.TrimSuffix(name, ".json")
		if err := CheckInstanceIdentity(stem); err != nil {
			return nil, fmt.Errorf("attach peer %q is not a receipt file", name)
		}
		raw, err := os.ReadFile(filepath.Join(store.instanceDir(instanceID), name))
		if err != nil {
			return nil, fmt.Errorf("read attach peer: %w", err)
		}
		receipt, err := decodeAttachReceipt(raw)
		if err != nil {
			return nil, err
		}
		if receipt.SessionID != sessionID {
			return nil, errors.New("attach receipt names another session")
		}
		if receipt.TerminalInstanceID != instanceID {
			return nil, errors.New("attach receipt names another client pair")
		}
		// The filename binding delegates to the keyed-read
		// owner: the entry admits only when the receipt for
		// its own durable key reads back. Valid bytes filed
		// under another client's name refuse here, before
		// the requesting-client exclusion can erase them
		// from the census.
		keyed, found, err := store.Lookup(sessionID, instanceID, stem)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("attach peer %q is not a recorded receipt", name)
		}
		if keyed.ClientID == clientID {
			continue
		}
		peers = append(peers, keyed)
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].ClientID < peers[j].ClientID })
	return peers, nil
}
