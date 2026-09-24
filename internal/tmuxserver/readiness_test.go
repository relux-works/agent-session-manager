package tmuxserver

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func TestCheckServerAttestedAdmitsBound(t *testing.T) {
	if err := CheckServerAttested(boundAdmission(), fixtureGeneration); err != nil {
		t.Fatal(err)
	}
}

func TestCheckServerAttestedRefusesEachMissingConjunct(t *testing.T) {
	for _, tc := range []struct {
		name      string
		admission RealmAdmission
		want      string
		detail    string
	}{
		{"no admission rows", RealmAdmission{RawGeneration: fixtureGeneration}, fixtureGeneration, "server attestation"},
		{"decoy capability only", RealmAdmission{
			Admitted:      terminalbackend.Admitted{Capabilities: []string{fixtureDecoyCapacity}},
			RawGeneration: fixtureGeneration,
		}, fixtureGeneration, "server attestation"},
		{"stale generation admission", staleAdmission(), fixtureGeneration, "server generation"},
		{"empty raw generation", RealmAdmission{Admitted: realmAdmitted()}, fixtureGeneration, "server generation"},
		{"empty wanted generation", boundAdmission(), "", "server generation"},
		{"both generations empty", RealmAdmission{Admitted: realmAdmitted()}, "", "server generation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireLocalError(t, CheckServerAttested(tc.admission, tc.want), "tmux_readiness_not_authorizing", tc.detail)
		})
	}
}

// The membership arm refuses every non-realm member of the Section
// 4.D vocabulary — not just the one decoy the fixture tables pin: a
// narrowing keyed on any other capability must redden here. The decoy
// set is derived from the pinned catalog, each member alone and all
// of them together without the realm row.
func TestCheckServerAttestedRefusesEveryCatalogDecoy(t *testing.T) {
	decoys := catalogDecoyCapabilities(t)
	for _, decoy := range decoys {
		t.Run(decoy, func(t *testing.T) {
			admission := RealmAdmission{
				Admitted:      terminalbackend.Admitted{Capabilities: []string{decoy}},
				RawGeneration: fixtureGeneration,
			}
			requireLocalError(t, CheckServerAttested(admission, fixtureGeneration), "tmux_readiness_not_authorizing", "server attestation")
		})
	}
	t.Run("all decoys together", func(t *testing.T) {
		admission := RealmAdmission{
			Admitted:      terminalbackend.Admitted{Capabilities: append([]string(nil), decoys...)},
			RawGeneration: fixtureGeneration,
		}
		requireLocalError(t, CheckServerAttested(admission, fixtureGeneration), "tmux_readiness_not_authorizing", "server attestation")
	})
}

func TestCheckBrokerContactAdmitsLiveReport(t *testing.T) {
	if err := CheckBrokerContact(brokerReport(), fixtureGeneration, currentUID()); err != nil {
		t.Fatal(err)
	}
}

func TestCheckBrokerContactRefusesEachMissingFact(t *testing.T) {
	live := brokerReport()
	for _, tc := range []struct {
		name   string
		report BrokerReport
		want   string
		detail string
	}{
		{"no broker", BrokerReport{}, fixtureGeneration, "broker authentication"},
		{"foreign-user broker", BrokerReport{
			Principal: BrokerPrincipal{UID: foreignUID(), Generation: fixtureGeneration},
			Server:    boundAdmission(),
		}, fixtureGeneration, "broker authentication"},
		{"generation-unbound principal", BrokerReport{
			Principal: BrokerPrincipal{UID: live.Principal.UID, Generation: fixtureStaleGen},
			Server:    boundAdmission(),
		}, fixtureGeneration, "broker generation"},
		{"empty principal generation", BrokerReport{
			Principal: BrokerPrincipal{UID: live.Principal.UID},
			Server:    boundAdmission(),
		}, fixtureGeneration, "broker generation"},
		{"both generations empty", BrokerReport{
			Principal: BrokerPrincipal{UID: live.Principal.UID},
			Server:    RealmAdmission{Admitted: realmAdmitted()},
		}, "", "broker generation"},
		{"unattested server", BrokerReport{
			Principal: live.Principal,
			Server:    RealmAdmission{RawGeneration: fixtureGeneration},
		}, fixtureGeneration, "server attestation"},
		{"stale server admission", BrokerReport{
			Principal: live.Principal,
			Server:    staleAdmission(),
		}, fixtureGeneration, "server generation"},
		{"zero server admission", BrokerReport{
			Principal: live.Principal,
			Server:    RealmAdmission{},
		}, fixtureGeneration, "server attestation"},
		{"admitted server without generation", BrokerReport{
			Principal: live.Principal,
			Server:    RealmAdmission{Admitted: realmAdmitted()},
		}, fixtureGeneration, "server generation"},
		{"decoy-only server admission", BrokerReport{
			Principal: live.Principal,
			Server: RealmAdmission{
				Admitted:      terminalbackend.Admitted{Capabilities: []string{fixtureDecoyCapacity}},
				RawGeneration: fixtureGeneration,
			},
		}, fixtureGeneration, "server attestation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireLocalError(t, CheckBrokerContact(tc.report, tc.want, currentUID()), "tmux_readiness_not_authorizing", tc.detail)
		})
	}
}
