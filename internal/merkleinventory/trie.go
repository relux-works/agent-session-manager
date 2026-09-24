package merkleinventory

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const nodeDomain = "urn:ax:merkle-node:1"

// Child is one materialized radix child. Labels are one lowercase hex nibble.
type Child struct {
	Label string        `json:"label"`
	Count scalar.Uint53 `json:"count"`
	Hash  scalar.Digest `json:"hash"`
}

// Node is the exact RPC 2 inventory.children success body.
type Node struct {
	Namespace string        `json:"namespace"`
	Prefix    string        `json:"prefix"`
	Count     scalar.Uint53 `json:"count"`
	NodeHash  scalar.Digest `json:"node_hash"`
	Children  []Child       `json:"children"`
	IDs       []string      `json:"ids"`
}

type logicalChild struct {
	Count scalar.Uint53 `json:"count"`
	Hash  scalar.Digest `json:"hash"`
	Label string        `json:"label"`
}

type logicalNode struct {
	Children  []logicalChild `json:"children"`
	Count     scalar.Uint53  `json:"count"`
	Domain    string         `json:"domain"`
	IDs       []string       `json:"ids"`
	Namespace string         `json:"namespace"`
	Prefix    string         `json:"prefix"`
}

type builtNode struct {
	node Node
}

func validNamespace(namespace string) bool {
	return slices.Contains(rpc2Namespaces, namespace)
}

var rpc2Namespaces = []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"}

func validPrefix(prefix string) bool {
	if len(prefix) > 64 {
		return false
	}
	for index := 0; index < len(prefix); index++ {
		c := prefix[index]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// ComputeNode is the shared Section 11.4 node construction entry. Duplicate
// input IDs are set members at ordinary prefixes; at a fully-qualified 64
// nibble prefix, duplicate storage entries indicate an integrity failure.
func ComputeNode(namespace, prefix string, inputIDs []string) (Node, error) {
	if !validNamespace(namespace) || !validPrefix(prefix) {
		return Node{}, ErrInvalidRequest
	}
	ids := make([]string, 0, len(inputIDs))
	for _, raw := range inputIDs {
		id, err := scalar.ParseDigest(raw)
		if err != nil {
			return Node{}, fmt.Errorf("%w: invalid object ID %q", ErrInvalidObject, raw)
		}
		if strings.HasPrefix(id.Hex(), prefix) {
			ids = append(ids, id.String())
		}
	}
	return buildNode(namespace, prefix, ids)
}

func buildNode(namespace, prefix string, matching []string) (Node, error) {
	if len(prefix) == 64 && len(matching) > 1 {
		return Node{}, refusal("integrity_failure", ErrIntegrity)
	}
	ids := slices.Clone(matching)
	sort.Strings(ids)
	ids = slices.Compact(ids)
	count, err := scalar.NewUint53(uint64(len(ids)))
	if err != nil {
		return Node{}, fmt.Errorf("%w: node count: %v", ErrIntegrity, err)
	}
	children := make([]Child, 0, 16)
	leafIDs := make([]string, 0, 1)
	if len(ids) == 1 {
		leafIDs = append(leafIDs, ids[0])
	} else if len(ids) > 1 {
		if len(prefix) == 64 {
			return Node{}, refusal("integrity_failure", ErrIntegrity)
		}
		groups := make(map[byte][]string, 16)
		for _, id := range ids {
			nibble := id[len("sha256:")+len(prefix)]
			groups[nibble] = append(groups[nibble], id)
		}
		labels := make([]byte, 0, len(groups))
		for label := range groups {
			labels = append(labels, label)
		}
		sort.Slice(labels, func(a, b int) bool { return labels[a] < labels[b] })
		for _, label := range labels {
			childPrefix := prefix + string(label)
			child, childErr := buildNode(namespace, childPrefix, groups[label])
			if childErr != nil {
				return Node{}, childErr
			}
			children = append(children, Child{Label: string(label), Count: child.Count, Hash: child.NodeHash})
		}
	}
	logicalChildren := make([]logicalChild, 0, len(children))
	for _, child := range children {
		logicalChildren = append(logicalChildren, logicalChild{Count: child.Count, Hash: child.Hash, Label: child.Label})
	}
	logical := logicalNode{
		Children:  logicalChildren,
		Count:     count,
		Domain:    nodeDomain,
		IDs:       leafIDs,
		Namespace: namespace,
		Prefix:    prefix,
	}
	encoded, err := json.Marshal(logical)
	if err != nil {
		return Node{}, fmt.Errorf("%w: encode logical node: %v", ErrIntegrity, err)
	}
	canonical, err := canonicaljson.Canonicalize(encoded)
	if err != nil {
		return Node{}, fmt.Errorf("%w: canonicalize logical node: %v", ErrIntegrity, err)
	}
	return Node{
		Namespace: namespace,
		Prefix:    prefix,
		Count:     count,
		NodeHash:  scalar.SHA256Digest(canonical),
		Children:  children,
		IDs:       leafIDs,
	}, nil
}
