package sessprofile

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

func validChangeParams() ChangeParams {
	return ChangeParams{
		SessionID:       testSessionID,
		CreatedByHostID: testHostID,
		LeaseEpoch:      1,
		LeaseID:         testLeaseID,
		Sequence:        2,
		Predecessors:    []string{zeroDigest},
		From:            ProfileStandard,
		To:              ProfileYOLO,
		Confirmed:       true,
		CreatedAt:       testChangedAt,
	}
}

func TestMintChangeEventEmitsAttestedBytes(t *testing.T) {
	minted, err := MintChangeEvent(validChangeParams())
	if err != nil {
		t.Fatalf("MintChangeEvent error = %v", err)
	}
	// Only tests attest: the minted event_id must be the true
	// omit-self digest through the owner's Verify entry.
	digest, field, err := canonicaljson.VerifyObjectIdentity(minted)
	if err != nil {
		t.Fatalf("VerifyObjectIdentity(minted) error = %v", err)
	}
	if field != canonicaljson.SelfEventID {
		t.Fatalf("VerifyObjectIdentity(minted) field = %q, want event_id", string(field))
	}
	event, err := DecodeEvent(minted)
	if err != nil {
		t.Fatalf("DecodeEvent(minted) error = %v", err)
	}
	if event.ID != digest.String() {
		t.Fatalf("minted event_id = %q, want computed %q", event.ID, digest.String())
	}
	if event.Type != "profile.changed" || event.SessionID != testSessionID {
		t.Fatalf("minted routing = %+v", event)
	}
	from, _ := payloadString(event.Payload, "from")
	to, _ := payloadString(event.Payload, "to")
	confirmed, _ := payloadBool(event.Payload, "confirmed")
	if from != ProfileStandard || to != ProfileYOLO || !confirmed {
		t.Fatalf("minted payload = %v", event.Payload)
	}
}

func TestMintChangeEventIsDeterministic(t *testing.T) {
	first, err := MintChangeEvent(validChangeParams())
	if err != nil {
		t.Fatalf("MintChangeEvent error = %v", err)
	}
	second, err := MintChangeEvent(validChangeParams())
	if err != nil {
		t.Fatalf("MintChangeEvent again error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("identical params minted different bytes")
	}
	// A different instant names a different instant and MUST differ.
	later := validChangeParams()
	later.CreatedAt = "2026-08-19T04:06:00.000Z"
	third, err := MintChangeEvent(later)
	if err != nil {
		t.Fatalf("MintChangeEvent(later) error = %v", err)
	}
	if bytes.Equal(first, third) {
		t.Fatal("differing instants minted identical bytes")
	}
	// Predecessors sort in the emitted bytes.
	multi := validChangeParams()
	high := "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	multi.Predecessors = []string{high, zeroDigest}
	out, err := MintChangeEvent(multi)
	if err != nil {
		t.Fatalf("MintChangeEvent(multi) error = %v", err)
	}
	event, err := DecodeEvent(out)
	if err != nil {
		t.Fatalf("DecodeEvent(multi) error = %v", err)
	}
	if len(event.Predecessors) != 2 || event.Predecessors[0] != zeroDigest || event.Predecessors[1] != high {
		t.Fatalf("minted predecessors = %v, want sorted", event.Predecessors)
	}
}

func TestMintChangeEventRefusals(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*ChangeParams)
		sentinel error
		fault    string
	}{
		{"bad session", func(params *ChangeParams) { params.SessionID = "nope" }, ErrInvalidEvent, "session_id"},
		{"bad author", func(params *ChangeParams) { params.CreatedByHostID = "nope" }, ErrInvalidEvent, "created_by_host_id"},
		{"zero epoch", func(params *ChangeParams) { params.LeaseEpoch = 0 }, ErrInvalidEvent, "lease_epoch"},
		{"epoch past uint53", func(params *ChangeParams) { params.LeaseEpoch = uint53Max + 1 }, ErrInvalidEvent, "lease_epoch"},
		{"bad lease", func(params *ChangeParams) { params.LeaseID = "nope" }, ErrInvalidEvent, "lease_id"},
		{"zero sequence", func(params *ChangeParams) { params.Sequence = 0 }, ErrInvalidEvent, "lease_sequence"},
		{"sequence past uint53", func(params *ChangeParams) { params.Sequence = uint53Max + 1 }, ErrInvalidEvent, "lease_sequence"},
		{"no predecessor", func(params *ChangeParams) { params.Predecessors = nil }, ErrInvalidEvent, "predecessor"},
		{"malformed predecessor", func(params *ChangeParams) { params.Predecessors = []string{"nope"} }, ErrInvalidEvent, "predecessor"},
		{"from outside vocabulary", func(params *ChangeParams) { params.From = "turbo" }, ErrInvalidProfile, "change from"},
		{"to outside vocabulary", func(params *ChangeParams) { params.To = "turbo" }, ErrInvalidProfile, "change to"},
		{"from equals to", func(params *ChangeParams) { params.From = ProfileYOLO }, ErrProfileUnchanged, "differ"},
		{"unconfirmed yolo", func(params *ChangeParams) { params.Confirmed = false }, ErrUnconfirmedYOLO, "confirmation"},
		{"bad instant", func(params *ChangeParams) { params.CreatedAt = "yesterday" }, ErrInvalidEvent, "created_at"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := validChangeParams()
			tc.mutate(&params)
			_, err := MintChangeEvent(params)
			if !errors.Is(err, tc.sentinel) {
				t.Fatalf("MintChangeEvent(%s) error = %v, want %v", tc.name, err, tc.sentinel)
			}
			if !strings.Contains(err.Error(), tc.fault) {
				t.Fatalf("MintChangeEvent(%s) error = %v, want %q named", tc.name, err, tc.fault)
			}
		})
	}
}

func TestMintChangeEventAdmitsUnconfirmedStandard(t *testing.T) {
	params := validChangeParams()
	params.From = ProfileYOLO
	params.To = ProfileStandard
	params.Confirmed = false
	minted, err := MintChangeEvent(params)
	if err != nil {
		t.Fatalf("MintChangeEvent(unconfirmed standard) error = %v", err)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(minted); err != nil {
		t.Fatalf("VerifyObjectIdentity(minted) error = %v", err)
	}
}
