package termbind

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func openAttachStore(t *testing.T) *AttachStore {
	t.Helper()
	store, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func attachClient(t *testing.T, store *AttachStore, clientID string) (AttachReceipt, bool) {
	t.Helper()
	receipt, replayed, err := store.Attach(fixtureSession, fixtureInstance, clientID, "local_only", true, validAttachAuth(t, "local_only", true), fixtureNow())
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	return receipt, replayed
}

func TestAttachRoundTrip(t *testing.T) {
	t.Parallel()
	store := openAttachStore(t)
	receipt, replayed := attachClient(t, store, fixtureClient)
	if replayed {
		t.Fatal("first Attach() replays, want a fresh receipt")
	}
	if receipt.SessionID != fixtureSession || receipt.TerminalInstanceID != fixtureInstance || receipt.ClientID != fixtureClient {
		t.Fatalf("receipt identity = %+v", receipt)
	}
	if receipt.Transport != "local_only" || !receipt.InputAuthorized || receipt.CreatedAt == "" {
		t.Fatalf("receipt = %+v", receipt)
	}
	lookedUp, found, err := store.Lookup(fixtureSession, fixtureInstance, fixtureClient)
	if err != nil || !found {
		t.Fatalf("Lookup() = %+v found %v err %v, want the receipt", lookedUp, found, err)
	}
	if lookedUp != receipt {
		t.Fatalf("Lookup() = %+v, want %+v", lookedUp, receipt)
	}
}

func TestAttachIdenticalRetryReplays(t *testing.T) {
	t.Parallel()
	store := openAttachStore(t)
	first, _ := attachClient(t, store, fixtureClient)
	second, replayed := attachClient(t, store, fixtureClient)
	if !replayed {
		t.Fatal("identical retry installs fresh, want replay")
	}
	if second != first {
		t.Fatalf("replay = %+v, want %+v", second, first)
	}
}

func TestAttachSecondClientRecordsAlongside(t *testing.T) {
	t.Parallel()
	store := openAttachStore(t)
	first, _ := attachClient(t, store, fixtureClient)
	second, replayed := attachClient(t, store, fixtureClientB)
	if replayed {
		t.Fatal("second client replays, want its own receipt")
	}
	if second.ClientID != fixtureClientB {
		t.Fatalf("second receipt = %+v", second)
	}
	kept, found, err := store.Lookup(fixtureSession, fixtureInstance, fixtureClient)
	if err != nil || !found || kept != first {
		t.Fatalf("first receipt after second attach = %+v found %v err %v", kept, found, err)
	}
}

func TestAttachConflictRefusesMismatch(t *testing.T) {
	t.Parallel()
	store := openAttachStore(t)
	attachClient(t, store, fixtureClient)
	_, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "trusted_private_mesh", true, validAttachAuth(t, "trusted_private_mesh", true), fixtureNow())
	requireCode(t, err, "idempotency_mismatch")
	_, _, err = store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", false, validAttachAuth(t, "local_only", false), fixtureNow())
	requireCode(t, err, "idempotency_mismatch")
}

func TestAttachRefusesForbiddenIdentity(t *testing.T) {
	t.Parallel()
	for _, forbidden := range forbiddenIdentities {
		t.Run("instance/"+forbidden.form, func(t *testing.T) {
			t.Parallel()
			store := openAttachStore(t)
			_, _, err := store.Attach(fixtureSession, forbidden.value, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), fixtureNow())
			requireCode(t, err, "terminal_backend_protocol_error")
		})
		t.Run("client/"+forbidden.form, func(t *testing.T) {
			t.Parallel()
			store := openAttachStore(t)
			_, _, err := store.Attach(fixtureSession, fixtureInstance, forbidden.value, "local_only", true, validAttachAuth(t, "local_only", true), fixtureNow())
			requireCode(t, err, "terminal_backend_protocol_error")
		})
	}
}

func TestAttachTransportVocabulary(t *testing.T) {
	t.Parallel()
	t.Run("unknown transport", func(t *testing.T) {
		t.Parallel()
		store := openAttachStore(t)
		_, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "smoke_signal", true, validAttachAuth(t, "local_only", true), fixtureNow())
		requireCode(t, err, "terminal_backend_protocol_error")
		requireDetail(t, err, "attach transport")
	})
	t.Run("relay refused by policy", func(t *testing.T) {
		t.Parallel()
		store := openAttachStore(t)
		_, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "third_party_relay", true, validAttachAuth(t, "third_party_relay", true), fixtureNow())
		requireCode(t, err, "terminal_backend_unauthorized")
	})
}

func TestAttachRequiresLiveAuth(t *testing.T) {
	t.Parallel()
	t.Run("expired auth", func(t *testing.T) {
		t.Parallel()
		store := openAttachStore(t)
		expired := attachAuth(t, "local_only", true, "2026-08-19T02:00:00.000Z", "2026-08-19T03:00:00.000Z")
		_, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, expired, fixtureNow())
		requireCode(t, err, "terminal_backend_unauthorized")
	})
	t.Run("replay with expired auth", func(t *testing.T) {
		t.Parallel()
		store := openAttachStore(t)
		attachClient(t, store, fixtureClient)
		expired := attachAuth(t, "local_only", true, "2026-08-19T02:00:00.000Z", "2026-08-19T03:00:00.000Z")
		_, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, expired, fixtureNow())
		requireCode(t, err, "terminal_backend_unauthorized")
	})
	t.Run("auth bound to another transport", func(t *testing.T) {
		t.Parallel()
		store := openAttachStore(t)
		_, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "trusted_private_mesh", true), fixtureNow())
		requireCode(t, err, "terminal_backend_unauthorized")
	})
	t.Run("malformed auth", func(t *testing.T) {
		t.Parallel()
		store := openAttachStore(t)
		if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, []byte(`{"transport":"local_only"}`), fixtureNow()); err == nil {
			t.Fatal("Attach(malformed auth) succeeded, want refusal")
		}
	})
}

func TestAttachEmitsNeitherEventNorStateChange(t *testing.T) {
	t.Parallel()
	repository, _ := chainFixture(t)
	snapshot := func() string {
		t.Helper()
		events, err := repository.ListEvents(fixtureSession)
		if err != nil {
			t.Fatal(err)
		}
		var builder strings.Builder
		for _, summary := range events {
			raw, err := repository.GetEvent(fixtureSession, summary.EventID)
			if err != nil {
				t.Fatal(err)
			}
			builder.Write(raw)
		}
		leases, err := repository.ListLeases(fixtureSession)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(leases)
		if err != nil {
			t.Fatal(err)
		}
		builder.Write(encoded)
		return builder.String()
	}
	before := snapshot()
	store := openAttachStore(t)
	attachClient(t, store, fixtureClient)
	attachClient(t, store, fixtureClient)
	attachClient(t, store, fixtureClientB)
	if after := snapshot(); after != before {
		t.Fatal("chain or lease bytes changed across attach activity, want zero state change")
	}
}

func TestAttachUnknownIsNotAbsent(t *testing.T) {
	t.Parallel()
	store := openAttachStore(t)
	attachClient(t, store, fixtureClient)
	receiptPath := store.receiptPath(fixtureInstance, fixtureClient)
	if err := os.WriteFile(receiptPath, []byte(`{"schema":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.Lookup(fixtureSession, fixtureInstance, fixtureClient); err == nil || found {
		t.Fatalf("Lookup(torn) = found %v err %v, want an error", found, err)
	}
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), fixtureNow()); err == nil {
		t.Fatal("Attach(torn receipt) succeeded, want an error")
	}
}

func TestAttachSessionMismatchRefuses(t *testing.T) {
	t.Parallel()
	store := openAttachStore(t)
	attachClient(t, store, fixtureClient)
	other := "0198f4c8-8e50-7f66-8f70-1234567890ab"
	if _, found, err := store.Lookup(other, fixtureInstance, fixtureClient); err == nil || found {
		t.Fatalf("Lookup(foreign session) = found %v err %v, want refusal", found, err)
	}
}
