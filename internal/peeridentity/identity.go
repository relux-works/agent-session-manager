// Package peeridentity resolves explicitly configured mesh identities. It does
// not discover hosts, authenticate SSH connections, or read key material.
package peeridentity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var (
	ErrConfigurationRequired = errors.New("peer identity requires an existing valid configuration")
	ErrNotAllowlisted        = errors.New("peer_not_allowlisted")
	ErrAmbiguousAlias        = errors.New("peer alias is ambiguous with a configured host ID")
	ErrHostIdentityMismatch  = errors.New("host_identity_mismatch")
	ErrDisclosureDenied      = errors.New("peer metadata disclosure refused")
	ErrSSHArgvTooLarge       = errors.New("peer SSH argv exceeds configuration byte bound")
)

// Directory is an immutable configuration snapshot. Its zero value admits no
// peers. Construct it with Load or FromSnapshot, never from a discovered host.
type Directory struct {
	local         Host
	peers         []config.Peer
	sourceVersion string
	defaultPolicy string
	summaryChoice string
	disclosure    []config.DirectoryPeerDisclosure
}

// Host is stable identity plus a display alias, not authentication evidence.
type Host struct {
	ID       string
	Name     string
	Platform scalar.Platform
}

// Load uses the canonical configuration loader, including path precedence and
// failed-read handling. Missing configuration never creates a host identity.
func Load(inputs config.Inputs, overrides config.Overrides) (Directory, error) {
	snapshot, err := config.Load(inputs, overrides)
	if err != nil {
		return Directory{}, err
	}
	return FromSnapshot(snapshot)
}

// FromSnapshot reuses the process-lifetime configuration decision. Snapshot's
// private state prevents an unvalidated Configuration from becoming an allowlist.
func FromSnapshot(snapshot config.Snapshot) (Directory, error) {
	loaded, ok := snapshot.Configuration()
	if !ok {
		return Directory{}, ErrConfigurationRequired
	}
	c := loaded.Value
	return Directory{
		local: Host{c.HostID, c.HostName, c.Platform}, peers: c.Mesh.Peers,
		sourceVersion: loaded.SourceVersion, defaultPolicy: c.Directory.DefaultMetadataPolicy,
		summaryChoice: c.Directory.GeneratedSummaryUpgradeChoice, disclosure: c.DirectoryPeerDisclosure,
	}, nil
}

func (d Directory) Local() Host { return d.local }

// Formatting a directory must not expose its retained SSH execution inputs.
func (d Directory) String() string                    { return "configured peer directory" }
func (d Directory) GoString() string                  { return d.String() }
func (d Directory) Format(state fmt.State, verb rune) { _, _ = state.Write([]byte(d.String())) }

// Resolve selects an exact configured host ID or exact display alias. Aliases
// are not case-folded or normalized. A name equal to another peer's ID is
// ambiguous, even though the two individually unique registries are valid.
func (d Directory) Resolve(selector string) (Target, error) {
	match := -1
	for i, peer := range d.peers {
		if peer.HostID != selector && peer.Name != selector {
			continue
		}
		if match >= 0 {
			return Target{}, ErrAmbiguousAlias
		}
		match = i
	}
	if match < 0 {
		return Target{}, ErrNotAllowlisted
	}
	peer := d.peers[match]
	return Target{host: Host{peer.HostID, peer.Name, peer.Platform}, endpoint: peer.Endpoint,
		args: append([]string(nil), peer.SSHArgs...)}, nil
}

// Target can only be obtained from a Directory. It is a connection plan, never
// proof that SSH authenticated a host or that a peer's protocol assertion is true.
// Endpoint and identity-file selectors are machine-local execution inputs; the
// formatting methods deliberately exclude them from logs and diagnostics.
type Target struct {
	host     Host
	endpoint string
	args     []string
}

func (t Target) Host() Host                        { return t.host }
func (t Target) String() string                    { return "configured SSH target" }
func (t Target) GoString() string                  { return t.String() }
func (t Target) Format(state fmt.State, verb rune) { _, _ = state.Write([]byte(t.String())) }

// KeyProvenance reports the authority boundary, not an observed key or a
// fingerprint. The pinned contract delegates both host-key and user-key policy
// to external SSH. No AX key registry, private-key reader, or verification flag
// is invented here. Empty targets have no provenance.
func (t Target) KeyProvenance() string {
	if t.endpoint == "" {
		return ""
	}
	return "external_ssh"
}

// RPCArgv returns an isolated atomic argv for the Section 11.1 fixed command.
// The canonical endpoint grammar permits :port; OpenSSH needs that port as -p,
// not as part of the destination. Brackets protect IPv6 while parsing the
// endpoint and are removed from the OpenSSH destination. Endpoint port wins
// over ssh_args port, as it is the explicitly configured target.
// The transport owner must execute via a native argv API and authenticate SSH;
// this method starts no process and asserts no successful authentication.
func (t Target) RPCArgv() ([]string, error) {
	if t.endpoint == "" {
		return nil, ErrNotAllowlisted
	}
	destination := t.endpoint
	port := ""
	if colon := strings.LastIndexByte(destination, ':'); colon >= 0 && !strings.HasSuffix(destination, "]") {
		port, destination = destination[colon+1:], destination[:colon]
	}
	if opening := strings.IndexByte(destination, '['); opening >= 0 {
		destination = destination[:opening] + strings.TrimSuffix(destination[opening+1:], "]")
	}
	args := []string{"-T"}
	if port != "" {
		args = append(args, "-p", port)
	}
	args = append(args, t.args...)
	args = append(args, destination, "ax", "rpc", "serve", "--stdio")
	// Config admission bounds the supplied endpoint/options. Count again here
	// because the fixed RPC command and generated options are also SSH argv.
	totalBytes := 0
	for _, arg := range args {
		totalBytes += len(arg)
	}
	if totalBytes > 65_536 {
		return nil, ErrSSHArgvTooLarge
	}
	return args, nil
}

// CheckProtocolHost checks only the identity half of Section 11.1, after the
// transport owner has authenticated the configured SSH endpoint. It is not an
// authentication entry point and accepts no caller-minted "verified" flag.
// Matching the endpoint, alias, or another allowlisted ID cannot substitute for
// an exact match to this target's stable host ID.
func (t Target) CheckProtocolHost(hostID string) error {
	if t.endpoint == "" {
		return ErrNotAllowlisted
	}
	if _, err := scalar.ParseUUIDv7(hostID); err != nil {
		return ErrHostIdentityMismatch
	}
	if hostID != t.host.ID {
		return ErrHostIdentityMismatch
	}
	return nil
}

// DisclosurePolicy resolves only the Configuration 2/3 policy for the five
// metadata classes in Section 6.4. It is not a payload sanitizer or permission
// to send arbitrary bytes. The directory publisher must still validate object
// schemas, apply redaction and object policy, and authenticate the recipient.
// local_only refuses export. Unset generated-summary choice cannot authorize
// replication. A disclosure entry alone never adds a peer to the allowlist.
func (d Directory) DisclosurePolicy(selector, class string) (string, error) {
	target, err := d.Resolve(selector)
	if err != nil {
		return "", err
	}
	switch class {
	case "environment_observations", "native_observations", "manual_metadata", "generated_metadata", "job_operation_status":
	default:
		return "", ErrDisclosureDenied
	}
	if d.sourceVersion == config.Version1 {
		return "", ErrDisclosureDenied
	}
	if class == "generated_metadata" && d.summaryChoice == "unset" {
		return "", ErrDisclosureDenied
	}
	policy := d.defaultPolicy
	for _, entry := range d.disclosure {
		if entry.HostID != target.host.ID {
			continue
		}
		switch class {
		case "environment_observations":
			policy = entry.EnvironmentObservations
		case "native_observations":
			policy = entry.NativeObservations
		case "manual_metadata":
			policy = entry.ManualMetadata
		case "generated_metadata":
			policy = entry.GeneratedMetadata
		case "job_operation_status":
			policy = entry.JobOperationStatus
		}
	}
	if policy != "mesh_sanitized" && policy != "reference_only" {
		return "", ErrDisclosureDenied
	}
	return policy, nil
}
