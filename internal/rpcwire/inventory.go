package rpcwire

import (
	"encoding/json"
	"slices"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Namespaces is the wire vocabulary for an explicitly chosen pinned version.
// It does not advertise support, compute inventory or authorize object exchange.
func Namespaces(version string) ([]string, error) {
	n, err := major(version)
	if err != nil {
		return nil, err
	}
	names := []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"}
	if n >= 3 {
		names = append(names, "directory_record")
	}
	if n >= 4 {
		names = append(names, "terminal_backend_evidence")
	}
	slices.Sort(names)
	return names, nil
}
func decodeNamespaces(version string, data []byte) ([]string, error) {
	m, err := object(data)
	if err != nil || !exact(m, "namespaces") {
		return nil, ErrInventory
	}
	var names []string
	if json.Unmarshal(m["namespaces"], &names) != nil {
		return nil, ErrInventory
	}
	allowed, _ := Namespaces(version)
	if len(names) < 1 || len(names) > len(allowed) {
		return nil, ErrInventory
	}
	for i, name := range names {
		if !slices.Contains(allowed, name) || (i > 0 && names[i-1] >= name) {
			return nil, ErrInventory
		}
	}
	return names, nil
}

// Root carries a claimed digest/count, not a verified Merkle root. Merkle
// verification and schema-to-namespace membership belong to object exchange.
type Root struct {
	Namespace string        `json:"namespace"`
	Count     scalar.Uint53 `json:"count"`
	RootID    scalar.Digest `json:"root_id"`
}

func validateRoots(version string, data, request []byte) error {
	names, err := decodeNamespaces(version, request)
	if err != nil {
		return err
	}
	m, err := object(data)
	if err != nil || !exact(m, "roots") {
		return ErrInventory
	}
	var roots []json.RawMessage
	if json.Unmarshal(m["roots"], &roots) != nil || len(roots) != len(names) {
		return ErrInventory
	}
	for i, raw := range roots {
		item, err := object(raw)
		if err != nil || !exact(item, "namespace", "count", "root_id") {
			return ErrInventory
		}
		var root Root
		if json.Unmarshal(raw, &root) != nil || root.Namespace != names[i] {
			return ErrInventory
		}
	}
	return nil
}
