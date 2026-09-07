package terminalbackend

import (
	"errors"
	"math"
	"testing"
)

// TestSemverMajorSaturationEdge proves the semverMajor saturation guard
// at its exact arithmetic edge, through the production entry points
// ParseProviderDescriptor, CheckVersionTuple, and New. math.MaxInt is
// 9223372036854775807 and MaxInt/10 is 922337203685477580. The guard
// `major > (math.MaxInt-digit)/10` must carry 9223372036854775807
// exactly (the largest value that fits) and saturate 9223372036854775808
// (MaxInt+1) to MaxInt. The weakened guard `major > math.MaxInt/10`
// differs only when the accumulator sits at exactly MaxInt/10 and the
// next digit is 8 or 9: it computes major*10+digit in int64, which
// wraps. The aliasing witness 922337203685477580801.0.0 walks the
// accumulator to MaxInt/10, wraps to -2^63 on the `8`, resets to 0 on
// the following `0` (0 mod 2^64), and lands on 1 — a foreign major
// aliasing native major 1, verbatim the invariant the semverMajor doc
// comment asserts. Its one-step neighbours 800 (wraps to 0) and 802
// (wraps to 2) pin that the fix is at the guard, not a literal
// blocklist of one string. Moving the guard literal by one term must
// redden here: every row requires semverMajor == MaxInt (not merely
// refusal, which the neighbours keep under the mutant) and refusal at
// each entry with the named arm — `descriptor protocol version` at
// ParseProviderDescriptor, `protocol_version major 1` at
// CheckVersionTuple, `protocol_versions major 1` at New.
func TestSemverMajorSaturationEdge(t *testing.T) {
	t.Parallel()

	versions := []string{
		"9223372036854775807.0.0",
		"9223372036854775808.0.0",
		"922337203685477580800.0.0",
		"922337203685477580801.0.0",
		"922337203685477580802.0.0",
	}
	for _, version := range versions {
		if major := semverMajor(version); major != math.MaxInt {
			t.Fatalf("semverMajor(%q) = %d, want saturation to math.MaxInt", version, major)
		}
		if major := semverMajor(version); major == 1 {
			t.Fatalf("semverMajor(%q) aliased to native major 1", version)
		}
		_, err := ParseProviderDescriptor(semverEdgeDescriptorDoc(version))
		if !IsProtocolError(err) {
			t.Fatalf("ParseProviderDescriptor(%q) error = %v, want terminal_backend_protocol_error", version, err)
		}
		var refusal *Error
		if !errors.As(err, &refusal) || refusal.Detail != "descriptor protocol version" {
			t.Fatalf("ParseProviderDescriptor(%q) detail = %v, want descriptor protocol version", version, err)
		}
		err = CheckVersionTuple("com.example.term", "1.2.3", version, []string{version})
		if !IsDrift(err) {
			t.Fatalf("CheckVersionTuple(%q) error = %v, want terminal_backend_implementation_drift", version, err)
		}
		if !errors.As(err, &refusal) || refusal.Detail != "protocol_version major 1" {
			t.Fatalf("CheckVersionTuple(%q) detail = %v, want protocol_version major 1", version, err)
		}
	}
	// The aliasing witness through the third semverMajor != 1 site: New
	// must refuse the foreign major before either built-in is admitted.
	// A weakened guard admits the witness here (and at RegisterExternal
	// through the shared validateProtocolVersions), so one row pins the
	// constructor while the loop above pins the shared gate per version.
	if _, err := New("1.2.3", []string{"922337203685477580801.0.0"}); !IsDrift(err) {
		t.Fatalf("New(aliasing major) error = %v, want terminal_backend_implementation_drift", err)
	} else {
		var refusal *Error
		if !errors.As(err, &refusal) || refusal.Detail != "protocol_versions major 1" {
			t.Fatalf("New(aliasing major) detail = %v, want protocol_versions major 1", err)
		}
	}

	// Controls: small native majors still read exactly, and the old
	// corpus witness still saturates (it saturates under the weakened
	// guard too, which is why it never distinguished the guards).
	if major := semverMajor("1.0.0"); major != 1 {
		t.Fatalf("semverMajor(1.0.0) = %d, want 1", major)
	}
	if major := semverMajor("2.0.0"); major != 2 {
		t.Fatalf("semverMajor(2.0.0) = %d, want 2", major)
	}
	if major := semverMajor("18446744073709551617.0.0"); major != math.MaxInt {
		t.Fatalf("semverMajor(18446744073709551617.0.0) = %d, want saturation to math.MaxInt", major)
	}
}

// semverEdgeDescriptorDoc builds one §7.A descriptor document carrying
// the given protocol version. Every other member is the valid
// descriptor's value, so only the protocol-version arm can refuse.
func semverEdgeDescriptorDoc(protocolVersion string) []byte {
	return []byte(`{` +
		`"terminal_binding_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",` +
		`"terminal_instance_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",` +
		`"terminal_backend_id": "com.example.term",` +
		`"implementation_version": "1.2.3",` +
		`"protocol_version": "` + protocolVersion + `",` +
		`"backend_generation": "generation-1",` +
		`"interactive": true,` +
		`"columns": 80,` +
		`"rows": 24` +
		`}`)
}
