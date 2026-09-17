package rpcwire

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"slices"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var semver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// Hello is a structural claim, including the untrusted HostID and limits.
// Received nonce randomness and freshness cannot be established by a codec.
type Hello struct {
	HostID         string              `json:"host_id"`
	Platform       string              `json:"platform"`
	AXVersion      string              `json:"ax_version"`
	Nonce          string              `json:"nonce"`
	NonceEcho      string              `json:"nonce_echo,omitempty"`
	Contracts      map[string][]string `json:"contracts"`
	MaxLineBytes   scalar.Uint53       `json:"max_line_bytes"`
	MaxObjectBytes scalar.Uint53       `json:"max_object_bytes"`
}

// NewNonce produces 256 random bits without accepting caller-supplied entropy.
func NewNonce() string {
	var nonce [32]byte
	_, _ = rand.Read(nonce[:]) // crypto/rand.Read is infallible in Go 1.25.
	return base64.RawURLEncoding.EncodeToString(nonce[:])
}

// ContractProfile describes the pinned wire shape, not installed capabilities.
// Returning a fresh map prevents one caller from changing another's validator.
func ContractProfile(version string) (map[string][]string, error) {
	n, err := major(version)
	if err != nil {
		return nil, err
	}
	m := map[string][]string{}
	for _, key := range []string{"rpc", "session_record", "session_event", "lease", "checkpoint", "workspace_group", "provider_identity", "blob", "transfer_manifest", "chunk", "materialization_plan", "tombstone", "tombstone_ack", "task_board_bundle"} {
		m[key] = []string{"1.0.0"}
	}
	m["rpc"] = []string{version}
	if n >= 3 {
		m["session_record"] = []string{"1.0.0", "2.0.0", "3.0.0"}
		m["session_event"] = []string{"1.0.0", "2.0.0", "3.0.0"}
		m["materialization_plan"] = []string{"1.0.0", "2.0.0"}
		for _, key := range []string{"environment_observation", "native_session_observation", "session_inventory_batch", "conversation_lineage_link", "session_annotation", "session_enrichment_profile", "session_enrichment_job_request", "session_enrichment_job_receipt", "session_continuation_plan", "session_directory_operation_receipt"} {
			m[key] = []string{"1.0.0"}
		}
	}
	if n >= 4 {
		m["session_event"] = append(m["session_event"], "4.0.0")
		m["terminal_backend_evidence"] = []string{"1.0.0"}
	}
	return m, nil
}

func validNonce(s string) bool {
	b, err := base64.RawURLEncoding.Strict().DecodeString(s)
	return err == nil && len(b) >= 16 && base64.RawURLEncoding.EncodeToString(b) == s
}
func decodeHello(version string, data []byte, response bool) (Hello, error) {
	m, err := object(data)
	if err != nil {
		return Hello{}, ErrHello
	}
	keys := []string{"host_id", "platform", "ax_version", "nonce", "contracts", "max_line_bytes", "max_object_bytes"}
	if response {
		keys = append(keys, "nonce_echo")
	}
	if !exact(m, keys...) {
		return Hello{}, ErrHello
	}
	var h Hello
	if json.Unmarshal(data, &h) != nil {
		return Hello{}, ErrHello
	}
	if _, err = scalar.ParseUUIDv7(h.HostID); err != nil {
		return Hello{}, ErrHello
	}
	if _, err = scalar.ParsePlatform(h.Platform); err != nil {
		return Hello{}, ErrHello
	}
	if !semver.MatchString(h.AXVersion) || !validNonce(h.Nonce) || (response && !validNonce(h.NonceEcho)) {
		return Hello{}, ErrHello
	}
	if h.MaxLineBytes.Uint64() < MaxLineBytes || h.MaxObjectBytes.Uint64() < MinObjectBytes {
		return Hello{}, ErrHello
	}
	profile, _ := ContractProfile(version)
	if len(h.Contracts) != len(profile) {
		return Hello{}, ErrHello
	}
	for key, expected := range profile {
		versions := h.Contracts[key]
		if len(versions) < 1 || len(versions) > 16 {
			return Hello{}, ErrHello
		}
		for i, v := range versions {
			if !semver.MatchString(v) || (i > 0 && versions[i-1] >= v) {
				return Hello{}, ErrHello
			}
		}
		// v2 permits 1..16 supported versions for every required key; later
		// pinned maps specify exact arrays. Selecting a common minor is separate.
		if version != "2.0.0" && !slices.Equal(versions, expected) {
			return Hello{}, ErrHello
		}
	}
	return h, nil
}

func (r Request) Hello() (Hello, error) {
	if r.operation != "hello" {
		return Hello{}, ErrHello
	}
	return decodeHello(r.version, r.body, false)
}
func (r Response) Hello() (Hello, error) {
	if !r.OK() || r.request.operation != "hello" {
		return Hello{}, ErrHello
	}
	return decodeHello(r.request.version, r.body, true)
}

// Limits is only the smaller pair of structural offers. It is not a negotiated
// or authenticated connection. The transport's fixed 8 MiB cap still applies.
type Limits struct{ LineBytes, ObjectBytes uint64 }

func OfferedLimits(request Request, response Response) (Limits, error) {
	a, err := request.Hello()
	if err != nil {
		return Limits{}, err
	}
	b, err := response.Hello()
	if err != nil {
		return Limits{}, err
	}
	if response.request.id != request.id || response.request.version != request.version || a.Nonce != b.NonceEcho {
		return Limits{}, ErrCorrelation
	}
	return Limits{min(a.MaxLineBytes.Uint64(), b.MaxLineBytes.Uint64()), min(a.MaxObjectBytes.Uint64(), b.MaxObjectBytes.Uint64())}, nil
}
