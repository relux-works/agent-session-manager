package environ

import (
	"encoding/json"
	"regexp"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// environmentIDPattern is the Section 7.8 environment-id grammar:
// [a-z][a-z0-9.-]{0,63}. One semantic native environment per
// identifier; it is never inferred by spelling.
var environmentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,63}$`)

// CheckEnvironmentID reports whether the value is an
// environment-id.
func CheckEnvironmentID(value string) bool {
	return environmentIDPattern.MatchString(value)
}

// semverPattern is the SemVer grammar adapter and environment
// versions satisfy: three dot-separated numeric parts with
// optional pre-release and build metadata.
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// CheckSemver reports whether the value is a SemVer string.
func CheckSemver(value string) bool {
	return semverPattern.MatchString(value)
}

// tupleArchitectures is the closed tuple architecture vocabulary
// Section 7.8 states. Membership is a table loop so the census
// derives it like every other closed vocabulary: an inline chain
// here would be invisible to the census, and a widened chain
// would silently admit a third architecture.
var tupleArchitectures = []string{
	"amd64",
	"arm64",
}

// validTupleArchitecture reports whether the name is one of the
// two tuple architectures.
func validTupleArchitecture(architecture string) bool {
	for _, allowed := range tupleArchitectures {
		if architecture == allowed {
			return true
		}
	}
	return false
}

// tupleMembers is the exact Environment Tuple member set Sections
// 7.8 and 13.14 require: environment_id, environment_version,
// platform, architecture, store_schema_fingerprint, and
// adapter_version. The tuple never contains executable provenance,
// and — unlike every neighbouring closed type — it carries no
// extensions member, so even a reverse-DNS extensions object is
// refused here.
var tupleMembers = map[string]bool{
	"environment_id":           true,
	"environment_version":      true,
	"platform":                 true,
	"architecture":             true,
	"store_schema_fingerprint": true,
	"adapter_version":          true,
}

// tupleRequired lists tupleMembers in a fixed order.
var tupleRequired = []string{
	"environment_id",
	"environment_version",
	"platform",
	"architecture",
	"store_schema_fingerprint",
	"adapter_version",
}

// Tuple is one validated Environment Tuple.
type Tuple struct {
	EnvironmentID    string
	Version          string
	Platform         string
	Architecture     string
	StoreFingerprint string
	AdapterVersion   string
}

// DecodeTuple validates one closed Environment Tuple: exact
// members, the environment-id grammar, environment_version as
// string[1..128] (the bound Section 10.8.1 states for the
// observation that reuses this admission model), the closed
// platform and architecture vocabularies, a digest store
// fingerprint, and a SemVer adapter version. Any executable
// provenance member is an unknown member, never a second
// identity.
func DecodeTuple(raw json.RawMessage) (Tuple, error) {
	members, fault := DecodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return Tuple{}, refuse("tuple "+fault.Detail, fault.Member)
	}
	if name, unknown := unknownMember(members, tupleMembers); unknown {
		return Tuple{}, refuse("tuple carries unknown member", name)
	}
	if name, missing := missingMember(members, tupleRequired); missing {
		return Tuple{}, refuse("tuple misses a required member", name)
	}
	environmentID, ok := rawString(members["environment_id"])
	if !ok || !CheckEnvironmentID(environmentID) {
		return Tuple{}, refuse("tuple environment identifier is not an environment-id", "environment_id")
	}
	version, ok := CheckStringBounds(members["environment_version"], 1, 128)
	if !ok {
		return Tuple{}, refuse("tuple environment version is not a string[1..128]", "environment_version")
	}
	platform, ok := rawString(members["platform"])
	if !ok {
		return Tuple{}, refuse("tuple platform is not a string", "platform")
	}
	if _, err := scalar.ParsePlatform(platform); err != nil {
		return Tuple{}, refuse("tuple platform is outside linux|macos|windows|wsl2", "platform")
	}
	architecture, ok := rawString(members["architecture"])
	if !ok || !validTupleArchitecture(architecture) {
		return Tuple{}, refuse("tuple architecture is outside amd64|arm64", "architecture")
	}
	fingerprint, ok := CheckDigest(members["store_schema_fingerprint"])
	if !ok {
		return Tuple{}, refuse("tuple store fingerprint is not a digest", "store_schema_fingerprint")
	}
	adapterVersion, ok := rawString(members["adapter_version"])
	if !ok || !CheckSemver(adapterVersion) {
		return Tuple{}, refuse("tuple adapter version is not SemVer", "adapter_version")
	}
	return Tuple{
		EnvironmentID:    environmentID,
		Version:          version,
		Platform:         platform,
		Architecture:     architecture,
		StoreFingerprint: fingerprint.String(),
		AdapterVersion:   adapterVersion,
	}, nil
}
