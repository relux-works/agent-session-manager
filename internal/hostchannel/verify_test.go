package hostchannel_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hostchannel"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// verifyEntry builds one enrolled entry from issued material through public
// entries only. Flip selectors corrupt a single member for isolation.
func verifyEntry(t *testing.T, issued hosttrust.IssuedCredential, state string, enrolledAt time.Time, retireAt *time.Time) hosttrust.CredentialEntry {
	t.Helper()
	leaf, err := x509.ParseCertificate(issued.LeafDER)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	stamp, err := scalar.ParseTimestamp(enrolledAt.UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	var retireStamp *scalar.Timestamp
	if retireAt != nil {
		parsed, err := scalar.ParseTimestamp(retireAt.UTC().Format("2006-01-02T15:04:05.000Z"))
		if err != nil {
			t.Fatalf("ParseTimestamp: %v", err)
		}
		retireStamp = &parsed
	}
	return hosttrust.CredentialEntry{
		HostID:       issued.HostID,
		CredentialID: scalar.SHA256Digest(issued.LeafDER),
		SPKIID:       scalar.SHA256Digest(leaf.RawSubjectPublicKeyInfo),
		RootID:       scalar.SHA256Digest(issued.RootDER),
		LeafDER:      bytes.Clone(issued.LeafDER),
		RootDER:      bytes.Clone(issued.RootDER),
		State:        state,
		EnrolledAt:   stamp,
		RetireAt:     retireStamp,
	}
}

func verifySnapshot(entries ...hosttrust.CredentialEntry) hosttrust.Snapshot {
	return hosttrust.Snapshot{
		Generation: 7,
		Trust:      hosttrust.TrustStore{Generation: 7, Entries: entries},
	}
}

func verifyState(t *testing.T, leafDER, rootDER []byte) tls.ConnectionState {
	t.Helper()
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatalf("parse root: %v", err)
	}
	return tls.ConnectionState{
		Version:            tls.VersionTLS13,
		NegotiatedProtocol: "ax-host/1",
		PeerCertificates:   []*x509.Certificate{leaf},
		VerifiedChains:     [][]*x509.Certificate{{leaf, root}},
	}
}

func TestVerifyPeerAdmitsEnrolled(t *testing.T) {
	now := time.Now().UTC()
	issued, err := hosttrust.IssueCredential(hostA, now, nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	snapshot := verifySnapshot(verifyEntry(t, issued, hosttrust.EntryActive, now, nil))
	verify, verified := hostchannel.VerifyPeer(snapshot, now)
	if err := verify(verifyState(t, issued.LeafDER, issued.RootDER)); err != nil {
		t.Fatalf("VerifyPeer: %v", err)
	}
	if verified.HostID != hostA {
		t.Fatalf("verified host = %q", verified.HostID)
	}
	if verified.CredentialID != scalar.SHA256Digest(issued.LeafDER).String() {
		t.Fatalf("verified credential = %q", verified.CredentialID)
	}
}

func TestVerifyPeerRefusals(t *testing.T) {
	now := time.Now().UTC()
	issued, err := hosttrust.IssueCredential(hostA, now, nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	attacker, err := hosttrust.IssueCredential(hostC, now, nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	aged, err := hosttrust.IssueCredential(hostC, now.Add(-100*24*time.Hour), nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	entry := verifyEntry(t, issued, hosttrust.EntryActive, now, nil)
	zeroSPKI := "sha256:" + string(bytes.Repeat([]byte("0"), 64))

	cases := []struct {
		name     string
		snapshot hosttrust.Snapshot
		state    func() tls.ConnectionState
		want     error
	}{
		{"alpn", verifySnapshot(entry), func() tls.ConnectionState {
			state := verifyState(t, issued.LeafDER, issued.RootDER)
			state.NegotiatedProtocol = "rogue-alpn"
			return state
		}, hostchannel.ErrALPNMismatch},
		{"alpn-same-family", verifySnapshot(entry), func() tls.ConnectionState {
			state := verifyState(t, issued.LeafDER, issued.RootDER)
			state.NegotiatedProtocol = "ax-host/2"
			return state
		}, hostchannel.ErrALPNMismatch},
		{"missing-alpn", verifySnapshot(entry), func() tls.ConnectionState {
			state := verifyState(t, issued.LeafDER, issued.RootDER)
			state.NegotiatedProtocol = ""
			return state
		}, hostchannel.ErrALPNMismatch},
		{"resumed", verifySnapshot(entry), func() tls.ConnectionState {
			state := verifyState(t, issued.LeafDER, issued.RootDER)
			state.DidResume = true
			return state
		}, hostchannel.ErrUnexpectedResume},
		{"no-chains", verifySnapshot(entry), func() tls.ConnectionState {
			state := verifyState(t, issued.LeafDER, issued.RootDER)
			state.VerifiedChains = nil
			return state
		}, hostchannel.ErrUnverifiedChain},
		{"no-certificates", verifySnapshot(entry), func() tls.ConnectionState {
			state := verifyState(t, issued.LeafDER, issued.RootDER)
			state.VerifiedChains = nil
			state.PeerCertificates = nil
			return state
		}, hostchannel.ErrUnverifiedChain},
		{"unknown-leaf", verifySnapshot(entry), func() tls.ConnectionState {
			return verifyState(t, attacker.LeafDER, attacker.RootDER)
		}, hostchannel.ErrNoEnrolledMatch},
		{"double-match", verifySnapshot(entry, func() hosttrust.CredentialEntry {
			clone := entry
			parsed, err := scalar.ParseUUIDv7(hostC)
			if err != nil {
				t.Fatalf("ParseUUIDv7: %v", err)
			}
			clone.HostID = parsed
			return clone
		}()), func() tls.ConnectionState {
			return verifyState(t, issued.LeafDER, issued.RootDER)
		}, hostchannel.ErrNoEnrolledMatch},
		{"leaf-flipped", verifySnapshot(func() hosttrust.CredentialEntry {
			clone := entry
			clone.LeafDER = bytes.Clone(entry.LeafDER)
			clone.LeafDER[len(clone.LeafDER)-1] ^= 0x01
			return clone
		}()), func() tls.ConnectionState {
			return verifyState(t, issued.LeafDER, issued.RootDER)
		}, hostchannel.ErrNoEnrolledMatch},
		{"root-flipped", verifySnapshot(func() hosttrust.CredentialEntry {
			clone := entry
			clone.RootDER = bytes.Clone(entry.RootDER)
			clone.RootDER[len(clone.RootDER)-1] ^= 0x01
			return clone
		}()), func() tls.ConnectionState {
			return verifyState(t, issued.LeafDER, issued.RootDER)
		}, hostchannel.ErrNoEnrolledMatch},
		{"spki-zero", verifySnapshot(func() hosttrust.CredentialEntry {
			clone := entry
			digest, err := scalar.ParseDigest(zeroSPKI)
			if err != nil {
				t.Fatalf("ParseDigest: %v", err)
			}
			clone.SPKIID = digest
			return clone
		}()), func() tls.ConnectionState {
			return verifyState(t, issued.LeafDER, issued.RootDER)
		}, hostchannel.ErrNoEnrolledMatch},
		{"revoked", verifySnapshot(func() hosttrust.CredentialEntry {
			clone := entry
			clone.State = hosttrust.EntryRevoked
			return clone
		}()), func() tls.ConnectionState {
			return verifyState(t, issued.LeafDER, issued.RootDER)
		}, hostchannel.ErrPeerTrustState},
		{"retiring-past", verifySnapshot(func() hosttrust.CredentialEntry {
			past := now.Add(-30 * time.Minute)
			return verifyEntry(t, issued, hosttrust.EntryRetiring, now.Add(-time.Hour), &past)
		}()), func() tls.ConnectionState {
			return verifyState(t, issued.LeafDER, issued.RootDER)
		}, hostchannel.ErrPeerTrustState},
		{"retiring-nil", verifySnapshot(func() hosttrust.CredentialEntry {
			clone := entry
			clone.State = hosttrust.EntryRetiring
			clone.RetireAt = nil
			return clone
		}()), func() tls.ConnectionState {
			return verifyState(t, issued.LeafDER, issued.RootDER)
		}, hostchannel.ErrPeerTrustState},
		{"expired", verifySnapshot(verifyEntry(t, aged, hosttrust.EntryActive, now.Add(-100*24*time.Hour), nil)), func() tls.ConnectionState {
			return verifyState(t, aged.LeafDER, aged.RootDER)
		}, hostchannel.ErrPeerProfile},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verify, _ := hostchannel.VerifyPeer(tc.snapshot, now)
			if err := verify(tc.state()); !errors.Is(err, tc.want) {
				t.Fatalf("VerifyPeer = %v, want %v", err, tc.want)
			}
		})
	}
}
