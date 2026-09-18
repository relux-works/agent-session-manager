// Package meshneg selects the single Mesh RPC major spoken on one peer
// connection from the local installation's configuration generation and the
// peer's decoded hello. It frames nothing, opens no connection, performs no
// I/O, keeps no state, and admits no peer: identity equality, allowlist
// membership, and trust generation remain the hostchannel admission boundary
// (TASK-260830-z1yxg9), which negotiation never substitutes.
//
// The package owns the Section 11.2-11.3/11.8/11.9/11.10.1 and Section 17
// selection rules only: highest common major among the locally offered and
// peer offered majors, core-only preservation with explicit unsupported
// directory/backend activation exposure, and no-common-major refusal with the
// pinned Section 15 class. Wire shapes (hello maps, namespaces, operation
// registry), Structured Error bindings, and configuration parsing stay with
// their landed owners (rpcwire, axerror, config); this package derives from
// them and never restates them.
package meshneg

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

// Pinned RPC framing versions this negotiator can select. A frame carrying
// any other version is not a lower minor of a supported major; Sections
// 11.2-11.10 pin exactly these four, and Section 17.2 rule 1 requires a
// reader to reject an unsupported major rather than coerce it.
var pinnedFrameVersions = map[string]int{
	"2.0.0": 2,
	"3.0.0": 3,
	"4.0.0": 4,
	"5.0.0": 5,
}

// directoryFeatureMajor is the first RPC major that negotiates directory
// replication (Section 11.8). backendEvidenceFeatureMajor is the first RPC
// major that negotiates TerminalBackend evidence replication (Section 11.9);
// RPC 5 retains the exact RPC-4 shapes (Section 11.10.1).
const (
	directoryFeatureMajor       = 3
	backendEvidenceFeatureMajor = 4
)

// Exposure codes for above-core activation on a negotiated peer. Both are
// registered Section 15.3 codes; neither is minted here.
const (
	// DirectoryUnsupportedCode is the literal Section 11.8 exposure token:
	// a node negotiating core-only with a peer reports that peer as
	// directory_mesh_unsupported, never as zero inventory.
	DirectoryUnsupportedCode = axerror.Code("directory_mesh_unsupported")
	// BackendEvidenceUnsupportedCode is the RPC-4 equivalent: Section 11.9
	// requires reporting TerminalBackend evidence unsupported rather than
	// empty, and Section 17.4 requires reporting activation unavailable
	// rather than omitting the session. terminal_backend_unavailable is
	// the registered exit-6 code that states exactly that.
	BackendEvidenceUnsupportedCode = axerror.Code("terminal_backend_unavailable")
)

// RefusalReason names why negotiation selected no major. Automation branches
// on Reason (and Code); Detail is human text only.
type RefusalReason string

const (
	// ReasonUnknownConfig reports a local configuration generation outside
	// 1.0.0-4.0.0. Section 6.6 refuses unknown configuration instead of
	// selecting legacy.
	ReasonUnknownConfig RefusalReason = "unknown_config"
	// ReasonUnsupportedFrame reports a peer frame version outside the
	// pinned 2.0.0-5.0.0 vocabulary.
	ReasonUnsupportedFrame RefusalReason = "unsupported_frame"
	// ReasonContractShape reports a peer hello whose contracts map is not
	// exactly the framing major's pinned profile (missing, extra, or
	// extension keys, or a wrong rpc array shape).
	ReasonContractShape RefusalReason = "contract_shape"
	// ReasonMixedMajor reports a peer rpc offer that mixes majors or
	// disagrees with its framing major. Section 17.1 forbids selecting a
	// major by coercion.
	ReasonMixedMajor RefusalReason = "mixed_major"
	// ReasonNoCommonMajor reports disjoint local and peer major sets.
	ReasonNoCommonMajor RefusalReason = "no_common_major"
)

var (
	// ErrUnknownConfig is the cause of an unknown-configuration refusal.
	ErrUnknownConfig = errors.New("unknown configuration generation")
	// ErrUnsupportedFrame is the cause of an unsupported-frame refusal.
	ErrUnsupportedFrame = errors.New("unsupported RPC frame version")
	// ErrContractShape is the cause of a contract-shape refusal.
	ErrContractShape = errors.New("hello contracts differ from the pinned profile")
	// ErrMixedMajor is the cause of a mixed-major refusal.
	ErrMixedMajor = errors.New("peer rpc offer mixes majors")
	// ErrNoCommonMajor is the cause of a no-common-major refusal.
	ErrNoCommonMajor = errors.New("no common RPC major")
)

// Refusal is the pure-data no-selection outcome. It frames no wire bytes:
// Sections 11.2 and 15.1 close without a peer error frame on an unsupported
// major, and Section 11.10.1 closes without a frame on other RPC majors, so
// the only peer-visible effect is the close itself. The initiator maps Code
// and ExitCode to its own local Structured Error; this value is that mapping
// input, never a forged peer response.
type Refusal struct {
	// Code and ExitCode are the pinned Section 15 class, resolved from the
	// axerror registry rather than retyped.
	Code     axerror.Code
	ExitCode int
	// Reason names the refusal shape; Detail is human text.
	Reason RefusalReason
	Detail string
	// Local carries the majors the local installation offered (empty when
	// the local generation itself is unknown). Peer carries the peer's
	// single offered major (zero when the peer offer was unusable).
	Local []int
	Peer  int
	// Cause is the Go sentinel for errors.Is; it never reaches the wire.
	Cause error
}

// Error renders the refusal without exposing anything beyond the refusal
// facts. Message text is for humans; automation branches on Code/Reason.
func (refusal *Refusal) Error() string {
	if refusal == nil {
		return "meshneg refusal"
	}
	return fmt.Sprintf("mesh RPC negotiation refused (%s): %s", refusal.Reason, refusal.Code)
}

// Unwrap exposes the cause sentinel for errors.Is.
func (refusal *Refusal) Unwrap() error {
	if refusal == nil {
		return nil
	}
	return refusal.Cause
}

// classExit resolves the pinned exit status for a refusal or exposure code
// under the Structured Error version that introduced it. An unregistered
// code or version fails closed: the caller gets an internal error, never a
// guessed exit.
func classExit(version axerror.Version, code axerror.Code) (int, error) {
	return axerror.ExitCodeFor(version, code)
}

// refuse builds a peer-side refusal (incompatible_protocol, exit 6). It is
// used only when incompatibility is actually known from a decoded hello and
// a known local generation, per Section 11.10.1; unframed input never
// reaches this package.
func refuse(reason RefusalReason, cause error, detail string, local []int, peer int) (*Refusal, error) {
	exit, err := classExit(axerror.Version100, axerror.Code("incompatible_protocol"))
	if err != nil {
		return nil, err
	}
	return &Refusal{
		Code:     axerror.Code("incompatible_protocol"),
		ExitCode: exit,
		Reason:   reason,
		Detail:   detail,
		Local:    append([]int(nil), local...),
		Peer:     peer,
		Cause:    cause,
	}, nil
}

// LocalMajors maps the local installation's configuration generation
// (config.LoadedConfiguration.SourceVersion, the source document's
// generation) to the exact RPC majors that installation offers, in ascending
// order. It must NOT be fed config.Configuration.SchemaVersion: the config
// owner normalizes Value to the current in-memory model (SchemaVersion is
// CurrentVersion "3.0.0" for every v1/v2/v3 source), so wiring SchemaVersion
// here would make a Config-1/2 installation offer RPC 3/4 it has no
// directory or backend tables for:
//
//   - 1.0.0 offers 2: the base configuration predates the directory and
//     TerminalBackend extensions, so only the Section 11.2-11.3 core.
//   - 2.0.0 offers 2 and 3: the Section 6.4 directory extension pairs with
//     RPC 3 directory replication, and Section 11.8 requires v2 dual-stack.
//   - 3.0.0 offers 2, 3, and 4: the Section 6.5 TerminalBackend extension
//     pairs with RPC 4 evidence replication, and Section 11.9 requires v3
//     dual-stack with preserved v2 interop.
//   - 4.0.0 offers exactly 5: Section 6.6 requires Host Channel 1 and RPC 5
//     for every peer and states that legacy dual-stack obligations never
//     require a Config-4 endpoint to accept legacy.
//
// Any other generation is refused with the invalid_config class (exit 3):
// Section 6.6 makes unknown configuration a refusal, never legacy
// selection. The result is a fresh slice on every call.
func LocalMajors(configVersion string) ([]int, error) {
	switch configVersion {
	case "1.0.0":
		return []int{2}, nil
	case "2.0.0":
		return []int{2, 3}, nil
	case "3.0.0":
		return []int{2, 3, 4}, nil
	case "4.0.0":
		return []int{5}, nil
	default:
		exit, err := classExit(axerror.Version100, axerror.Code("invalid_config"))
		if err != nil {
			return nil, err
		}
		return nil, &Refusal{
			Code:     axerror.Code("invalid_config"),
			ExitCode: exit,
			Reason:   ReasonUnknownConfig,
			Detail:   fmt.Sprintf("configuration generation %q offers no RPC major", configVersion),
			Cause:    ErrUnknownConfig,
		}
	}
}

// semverMajor parses a strict semantic version and returns its major
// component. The grammar is the same full semver rpcwire enforces on hello
// contract arrays (major.minor.patch with optional prerelease and build
// metadata, no leading zeros, no "v" prefix); agreement with rpcwire decode
// is measured by test, not asserted by comment.
var semverMajor = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

func majorOf(version string) (int, error) {
	match := semverMajor.FindStringSubmatch(version)
	if match == nil {
		return 0, fmt.Errorf("%w: version %q is not strict semver", ErrMixedMajor, version)
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("%w: version %q has no numeric major", ErrMixedMajor, version)
	}
	return major, nil
}

// PeerOffer validates the peer's side of the negotiation: the decoded frame
// version plus the decoded hello's contracts map. It returns the peer's
// single offered major. Identity, platform, version, nonce, echo, and limit
// members of the hello are never consulted: negotiation selects a major, it
// does not authenticate, and consulting them here would substitute for the
// admission boundary.
//
// The offer is refused when the frame version is outside the pinned
// vocabulary, when the contracts keys differ from the framing major's exact
// pinned profile (a v2 14-key bound inside a v3+ frame, an extension key, or
// any missing key), when the rpc array is empty or not strictly
// sorted-unique, when any rpc version is not strict semver, or when any rpc
// major differs from the framing major. A v2 frame may carry a 1-16 version
// minor range (Section 11.2); selecting within that range stays a stated
// bound, since this package negotiates majors only.
//
// Two stated bounds. First, values of non-rpc arrays are validated by rpcwire
// decode (exact per-key arrays for v3+, 1-16 sorted-unique semver per key for
// v2) and trusted here: PeerOffer re-validates the key set and the rpc array
// only, so direct callers must pass hellos produced by rpcwire decode
// (Request.Hello or Response.Hello), never hand-built structs. Second, an
// unparseable frame version ("", "garbage", "v2.0.0") answers
// incompatible_protocol/exit 6 here while Section 15.1 assigns
// transport_failure to an unrecognizable version; the input is unreachable
// through rpcwire decode, which only the four pinned strings pass.
func PeerOffer(frameVersion string, hello rpcwire.Hello) (int, error) {
	framing, pinned := pinnedFrameVersions[frameVersion]
	if !pinned {
		refusal, err := refuse(ReasonUnsupportedFrame, ErrUnsupportedFrame,
			fmt.Sprintf("frame version %q is outside the pinned 2.0.0-5.0.0 vocabulary", frameVersion), nil, 0)
		if err != nil {
			return 0, err
		}
		return 0, refusal
	}
	profile, err := rpcwire.ContractProfile(frameVersion)
	if err != nil {
		return 0, err
	}
	if len(hello.Contracts) != len(profile) {
		return 0, shapeRefusal(framing)
	}
	for key := range profile {
		if _, ok := hello.Contracts[key]; !ok {
			return 0, shapeRefusal(framing)
		}
	}
	offered := hello.Contracts["rpc"]
	if len(offered) < 1 || len(offered) > 16 {
		return 0, shapeRefusal(framing)
	}
	for index, version := range offered {
		major, err := majorOf(version)
		if err != nil {
			return 0, mixedRefusal(fmt.Sprintf("rpc version %q is not strict semver", version))
		}
		if major != framing {
			return 0, mixedRefusal(fmt.Sprintf("rpc version %q mixes major %d into a major-%d frame", version, major, framing))
		}
		if index > 0 && offered[index-1] >= version {
			return 0, mixedRefusal("rpc versions are not strictly sorted unique")
		}
	}
	if frameVersion != "2.0.0" && !slices.Equal(offered, profile["rpc"]) {
		return 0, shapeRefusal(framing)
	}
	return framing, nil
}

func shapeRefusal(framing int) error {
	refusal, err := refuse(ReasonContractShape, ErrContractShape,
		fmt.Sprintf("hello contracts do not match the pinned major-%d profile", framing), nil, 0)
	if err != nil {
		return err
	}
	return refusal
}

func mixedRefusal(detail string) error {
	refusal, err := refuse(ReasonMixedMajor, ErrMixedMajor, detail, nil, 0)
	if err != nil {
		return err
	}
	return refusal
}

// Select returns the highest major in the intersection of the local and peer
// offer sets. Either set naming a major outside the pinned 2/3/4/5
// vocabulary, an empty set on either side, or a disjoint pair is refused as
// no-common-major: an unmentionable major can never be common, and Section
// 17.1 forbids selecting one by coercion.
func Select(local, peer []int) (int, error) {
	for _, major := range local {
		if major < 2 || major > 5 {
			refusal, err := refuse(ReasonNoCommonMajor, ErrNoCommonMajor,
				fmt.Sprintf("local offer names major %d outside the pinned 2/3/4/5 vocabulary", major), append([]int(nil), local...), 0)
			if err != nil {
				return 0, err
			}
			return 0, refusal
		}
	}
	for _, major := range peer {
		if major < 2 || major > 5 {
			refusal, err := refuse(ReasonNoCommonMajor, ErrNoCommonMajor,
				fmt.Sprintf("peer offer names major %d outside the pinned 2/3/4/5 vocabulary", major), append([]int(nil), local...), 0)
			if err != nil {
				return 0, err
			}
			return 0, refusal
		}
	}
	best := 0
	for _, candidate := range local {
		if slices.Contains(peer, candidate) && candidate > best {
			best = candidate
		}
	}
	if best == 0 {
		peerMajor := 0
		if len(peer) == 1 {
			peerMajor = peer[0]
		}
		refusal, err := refuse(ReasonNoCommonMajor, ErrNoCommonMajor,
			"local and peer offers share no major ("+joinMajors(local)+" against "+joinMajors(peer)+")",
			append([]int(nil), local...), peerMajor)
		if err != nil {
			return 0, err
		}
		return 0, refusal
	}
	return best, nil
}

func joinMajors(majors []int) string {
	parts := make([]string, 0, len(majors))
	for _, major := range majors {
		parts = append(parts, strconv.Itoa(major))
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// FeatureActivation reports how one above-core feature is exposed for the
// negotiated peer. It carries no inventory counts, no object IDs, and no
// capability advertisement: Negotiated states the outcome of this selection,
// and Code names the registered exposure code only when the feature was not
// negotiated, so a core-only peer can never be mistaken for an empty one.
type FeatureActivation struct {
	// Feature is "directory" or "terminal_backend_evidence".
	Feature string
	// Negotiated is true exactly when the selected major carries the
	// feature (directory at 3+, backend evidence at 4+).
	Negotiated bool
	// Code is empty when negotiated; otherwise the registered exposure
	// code with ExitCode 6.
	Code     axerror.Code
	ExitCode int
}

// PeerView is the local node's exposure of the negotiated peer: which
// above-core activations the selection carries and which it reports
// unsupported. Core sync itself is the negotiated major, not a member here.
type PeerView struct {
	Directory       FeatureActivation
	BackendEvidence FeatureActivation
}

// Decision is the pure-data selection outcome: exactly one common major
// plus the shapes bound to it. Contracts and Namespaces are rebuilt from
// the pinned profile for the selected major, never copied from peer bytes,
// so a malformed peer array cannot propagate past selection.
type Decision struct {
	Major           int
	ProtocolVersion string
	Contracts       map[string][]string
	Namespaces      []string
	ErrorVersion    axerror.Version
	Peer            PeerView
}

func exposure(feature string, negotiated bool, code axerror.Code, introduced axerror.Version) (FeatureActivation, error) {
	activation := FeatureActivation{Feature: feature, Negotiated: negotiated}
	if negotiated {
		return activation, nil
	}
	exit, err := classExit(introduced, code)
	if err != nil {
		return FeatureActivation{}, err
	}
	activation.Code = code
	activation.ExitCode = exit
	return activation, nil
}

func decisionFor(major int) (Decision, error) {
	version := fmt.Sprintf("%d.0.0", major)
	contracts, err := rpcwire.ContractProfile(version)
	if err != nil {
		return Decision{}, err
	}
	namespaces, err := rpcwire.Namespaces(version)
	if err != nil {
		return Decision{}, err
	}
	bound, err := axerror.BindingFor(axerror.ContainingContract{ID: "urn:ax:protocol:rpc", Major: major})
	if err != nil {
		return Decision{}, err
	}
	directory, err := exposure("directory", major >= directoryFeatureMajor, DirectoryUnsupportedCode, axerror.Version120)
	if err != nil {
		return Decision{}, err
	}
	backend, err := exposure("terminal_backend_evidence", major >= backendEvidenceFeatureMajor, BackendEvidenceUnsupportedCode, axerror.Version130)
	if err != nil {
		return Decision{}, err
	}
	return Decision{
		Major:           major,
		ProtocolVersion: version,
		Contracts:       contracts,
		Namespaces:      namespaces,
		ErrorVersion:    bound,
		Peer:            PeerView{Directory: directory, BackendEvidence: backend},
	}, nil
}

// Negotiate selects exactly one common Mesh RPC major for the local
// configuration generation and the peer's decoded hello frame. configVersion
// is the local config.LoadedConfiguration.SourceVersion (never
// config.Configuration.SchemaVersion, which the config owner normalizes to
// CurrentVersion for every v1/v2/v3 source); frameVersion is the decoded
// envelope's protocol_version; hello is the decoded hello body.
//
// On success it returns the Decision for the highest common major. On
// refusal it returns a *Refusal carrying the pinned Section 15 class:
// incompatible_protocol with exit 6 when the decoded inputs prove the
// majors are disjoint or the peer offer is malformed, or invalid_config
// with exit 3 when the local generation is unknown. Any other error is an
// internal registry inconsistency, never a peer verdict.
//
// Negotiate consults only the configuration generation, the frame version,
// and the hello contracts map. It reads no environment, no argv, no raw
// frame bytes, no extension member, and no prior-handshake state, so none
// of those can select a fallback (Section 6.6). It performs no admission:
// the hello identity, allowlist, and trust generation must still pass the
// hostchannel boundary before any dispatch.
func Negotiate(configVersion, frameVersion string, hello rpcwire.Hello) (Decision, error) {
	local, err := LocalMajors(configVersion)
	if err != nil {
		return Decision{}, err
	}
	peer, err := PeerOffer(frameVersion, hello)
	if err != nil {
		if refusal, ok := err.(*Refusal); ok {
			refusal.Local = append([]int(nil), local...)
			return Decision{}, refusal
		}
		return Decision{}, err
	}
	major, err := Select(local, []int{peer})
	if err != nil {
		return Decision{}, err
	}
	return decisionFor(major)
}
