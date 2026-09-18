package terminstance

import (
	"strings"
	"testing"
	"time"
)

// TestParseAXAuthorizationAdmitsClosedFixture drives the closed
// authorization object through the production parse entry and asserts
// every member value literally.
func TestParseAXAuthorizationAdmitsClosedFixture(t *testing.T) {
	auth, err := ParseAXAuthorization([]byte(authDoc("1", nil)))
	if err != nil {
		t.Fatalf("ParseAXAuthorization() error = %v", err)
	}
	requireLiteral(t, "lease_id", auth.LeaseID, "f47ac10b-58cc-4372-a567-0e02b2c3d479")
	if auth.LeaseEpoch != 1 {
		t.Errorf("lease_epoch = %d, want 1", auth.LeaseEpoch)
	}
	requireLiteral(t, "holder_host_id", auth.HolderHostID, "0198f4c8-8e50-7f66-8f70-aaaaaaaaaaa1")
	requireLiteral(t, "authorization_kind", string(auth.Kind), "control")
	requireLiteral(t, "issued_at", auth.IssuedAt.String(), "2026-09-01T00:00:00.000Z")
	requireLiteral(t, "expires_at", auth.ExpiresAt.String(), "2026-09-02T00:00:00.000Z")
	requireLiteral(t, "authorization_evidence_id", auth.EvidenceID, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
}

// TestParseAXAuthorizationEpochBound witnesses the uint53[1..9007199254740991]
// bound at 1, 9007199254740991 and both adjacent out-of-range values,
// through the production entry.
func TestParseAXAuthorizationEpochBound(t *testing.T) {
	for _, epoch := range []string{"1", "9007199254740991"} {
		if _, err := ParseAXAuthorization([]byte(authDoc(epoch, nil))); err != nil {
			t.Errorf("ParseAXAuthorization(epoch %s) error = %v, want admission", epoch, err)
		}
	}
	for _, epoch := range []string{"0", "9007199254740992"} {
		_, err := ParseAXAuthorization([]byte(authDoc(epoch, nil)))
		requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization epoch")
	}
}

// TestParseAXAuthorizationEpochNumberModel proves the AX number model at
// the epoch member: fractions, exponents, signs and magnitudes beyond
// the scalar interval are refused, never rounded-and-continued.
func TestParseAXAuthorizationEpochNumberModel(t *testing.T) {
	for _, epoch := range []string{"1.0", "1e3", "-1", `"1"`, "true", "null", "18446744073709551615", "9007199254740993"} {
		_, err := ParseAXAuthorization([]byte(authDoc(epoch, nil)))
		requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization epoch")
	}
	// A signed literal is not JSON at all: the frame refuses before any
	// member arm runs.
	_, err := ParseAXAuthorization([]byte(authDoc("+1", nil)))
	requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization frame")
}

// TestParseAXAuthorizationKindClosed pins the four-member kind vocabulary
// at the production entry.
func TestParseAXAuthorizationKindClosed(t *testing.T) {
	for _, kind := range []string{"create", "control", "force_stale", "restore"} {
		raw := authDoc("1", map[string]*string{"authorization_kind": strptr(`"` + kind + `"`)})
		if _, err := ParseAXAuthorization([]byte(raw)); err != nil {
			t.Errorf("ParseAXAuthorization(kind %s) error = %v, want admission", kind, err)
		}
	}
	// The corpus carries the sibling vocabularies alongside the
	// landed attach/none names: the retry dispositions and the
	// provider proof kinds from the same section.
	for _, kind := range []string{"admin", "CREATE", "", "control-plane", "attach", "none", "replay_same", "provider_quiescence"} {
		raw := authDoc("1", map[string]*string{"authorization_kind": strptr(`"` + kind + `"`)})
		_, err := ParseAXAuthorization([]byte(raw))
		requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization kind")
	}
}

// TestParseAXAuthorizationMemberTypes drives every member grammar through
// the production entry: UUIDv4 lease, UUIDv7 holder, timestamps and
// digest evidence.
func TestParseAXAuthorizationMemberTypes(t *testing.T) {
	cases := []struct {
		name   string
		member string
		raw    string
		detail string
	}{
		{"lease v7 is not v4", "lease_id", `"0198f4c8-8e50-7f66-8f70-aaaaaaaaaaa1"`, "ax authorization lease"},
		{"lease garbage", "lease_id", `"not-a-uuid"`, "ax authorization lease"},
		{"lease number", "lease_id", `1`, "ax authorization lease"},
		{"holder v4 is not v7", "holder_host_id", `"f47ac10b-58cc-4372-a567-0e02b2c3d479"`, "ax authorization holder"},
		{"holder garbage", "holder_host_id", `"nope"`, "ax authorization holder"},
		{"issued garbage", "issued_at", `"yesterday"`, "ax authorization timestamp"},
		{"issued no millis", "issued_at", `"2026-09-01T00:00:00Z"`, "ax authorization timestamp"},
		{"expires garbage", "expires_at", `"tomorrow"`, "ax authorization timestamp"},
		{"evidence not digest", "authorization_evidence_id", `"raw-bytes"`, "ax authorization evidence"},
		{"evidence number", "authorization_evidence_id", `7`, "ax authorization evidence"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAXAuthorization([]byte(authDoc("1", map[string]*string{tc.member: strptr(tc.raw)})))
			requireRefusal(t, err, "terminal_backend_protocol_error", tc.detail)
		})
	}
}

// TestParseAXAuthorizationClosedShape proves the closed object: an
// unknown member and a missing member are both refused, and the four
// frame violations (non-object, trailing data, duplicate member, lone
// surrogate) never decode.
func TestParseAXAuthorizationClosedShape(t *testing.T) {
	_, err := ParseAXAuthorization([]byte(authDoc("1", map[string]*string{"lease_record": strptr(`"smuggled"`)})))
	requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization members")

	_, err = ParseAXAuthorization([]byte(authDoc("1", map[string]*string{"holder_host_id": nil})))
	requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization members")

	frames := map[string]string{
		"array top level": `[1,2]`,
		"trailing data":   authDoc("1", nil) + ` {}`,
		"duplicate":       strings.Replace(authDoc("1", nil), `"holder_host_id"`, `"lease_id":"f47ac10b-58cc-4372-a567-0e02b2c3d478","holder_host_id"`, 1),
		"lone surrogate":  strings.Replace(authDoc("1", nil), fixtureLease, `f47ac10b-58cc-4372-a567-0e02b2c3d47\ud800`, 1),
		"not utf8":        "{\"\xff\":1}",
	}
	for name, raw := range frames {
		t.Run(name, func(t *testing.T) {
			_, err := ParseAXAuthorization([]byte(raw))
			requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization frame")
		})
	}
}

// TestParseAXAuthorizationStrictExpiry proves expiry strictly after
// issue: equal and inverted instants refuse with the unauthorized class,
// mirroring the landed attach twin.
func TestParseAXAuthorizationStrictExpiry(t *testing.T) {
	equal := map[string]*string{"expires_at": strptr(`"` + fixtureIssued + `"`)}
	_, err := ParseAXAuthorization([]byte(authDoc("1", equal)))
	requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry order")

	inverted := map[string]*string{
		"issued_at":  strptr(`"` + fixtureExpires + `"`),
		"expires_at": strptr(`"` + fixtureIssued + `"`),
	}
	_, err = ParseAXAuthorization([]byte(authDoc("1", inverted)))
	requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry order")
}

// TestCheckAuthorizationAdmitsWinningLease drives the binding check at
// the production entry: the required kind, an unexpired window and the
// exact winning tuple pass.
func TestCheckAuthorizationAdmitsWinningLease(t *testing.T) {
	auth := testAuth(t)
	if err := CheckAuthorization(auth, AuthorizationControl, testLease(), fixtureNow()); err != nil {
		t.Errorf("CheckAuthorization() error = %v, want admission", err)
	}
}

// TestCheckAuthorizationRefusals drives every binding arm through the
// production entry: kind mismatch, expiry at and past the edge, lease
// identity drift and lease epoch drift, each with the unauthorized
// class and its own detail.
func TestCheckAuthorizationRefusals(t *testing.T) {
	auth := testAuth(t)
	expires, _ := auth.ExpiresAt.Time()

	t.Run("kind mismatch", func(t *testing.T) {
		err := CheckAuthorization(auth, AuthorizationCreate, testLease(), fixtureNow())
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization kind binding")
	})
	t.Run("expired", func(t *testing.T) {
		err := CheckAuthorization(auth, AuthorizationControl, testLease(), expires.Add(time.Second))
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry")
	})
	t.Run("expiry edge refuses", func(t *testing.T) {
		err := CheckAuthorization(auth, AuthorizationControl, testLease(), expires)
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization expiry")
	})
	t.Run("lease identity drift", func(t *testing.T) {
		lease := LeaseView{LeaseID: fixtureLeaseB, Epoch: 1}
		err := CheckAuthorization(auth, AuthorizationControl, lease, fixtureNow())
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization lease identity")
	})
	t.Run("lease epoch drift", func(t *testing.T) {
		lease := LeaseView{LeaseID: fixtureLease, Epoch: 2}
		err := CheckAuthorization(auth, AuthorizationControl, lease, fixtureNow())
		requireRefusal(t, err, "terminal_backend_unauthorized", "ax authorization lease epoch")
	})
}
