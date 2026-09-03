package axerror

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func baseSpec() Spec {
	return Spec{
		Version: Version130,
		Code:    "not_found",
		Message: "no such session",
		IDs:     NoIDs(),
		Details: Details{},
	}
}

// TestNewRefusesShapeAndBoundViolations narrows every Section 15.1 bound in
// turn. A validator that dropped any one of these would still admit the valid
// object above and every other case here, so each row proves its own bound.
func TestNewRefusesShapeAndBoundViolations(test *testing.T) {
	if _, err := New(baseSpec()); err != nil {
		test.Fatalf("the valid base object was refused: %v", err)
	}
	cases := []struct {
		name    string
		mutate  func(Spec) Spec
		wantErr error
	}{
		{
			name:    "empty message",
			mutate:  func(spec Spec) Spec { spec.Message = ""; return spec },
			wantErr: ErrInvalidStructuredError,
		},
		{
			name:    "message one character past the bound",
			mutate:  func(spec Spec) Spec { spec.Message = strings.Repeat("a", 4097); return spec },
			wantErr: ErrInvalidStructuredError,
		},
		{
			name:    "message counted in bytes rather than characters",
			mutate:  func(spec Spec) Spec { spec.Message = strings.Repeat("é", 4097); return spec },
			wantErr: ErrInvalidStructuredError,
		},
		{
			name:    "invalid UTF-8 message",
			mutate:  func(spec Spec) Spec { spec.Message = string([]byte{0xff, 0xfe}); return spec },
			wantErr: ErrInvalidStructuredError,
		},
		{
			name:    "null details",
			mutate:  func(spec Spec) Spec { spec.Details = nil; return spec },
			wantErr: ErrInvalidDetails,
		},
		{
			name:    "sixty-five detail keys",
			mutate:  func(spec Spec) Spec { spec.Details = countedDetails(65); return spec },
			wantErr: ErrInvalidDetails,
		},
		{
			name:    "detail key outside the grammar",
			mutate:  func(spec Spec) Spec { spec.Details = Details{"Expected": "value"}; return spec },
			wantErr: ErrInvalidDetails,
		},
		{
			name: "detail key one byte past the grammar length",
			mutate: func(spec Spec) Spec {
				spec.Details = Details{"a" + strings.Repeat("b", 64): "value"}
				return spec
			},
			wantErr: ErrInvalidDetails,
		},
		{
			name:    "detail value nested one container too deep",
			mutate:  func(spec Spec) Spec { spec.Details = Details{"nested": nestedValue(5)}; return spec },
			wantErr: ErrInvalidDetails,
		},
		{
			name: "detail map past sixteen kibibytes",
			mutate: func(spec Spec) Spec {
				spec.Details = Details{"blob": strings.Repeat("x", 16*1024)}
				return spec
			},
			wantErr: ErrInvalidDetails,
		},
		{
			name:    "detail value of an unsupported Go type",
			mutate:  func(spec Spec) Spec { spec.Details = Details{"count": 7}; return spec },
			wantErr: ErrInvalidDetails,
		},
		{
			name:    "unregistered code",
			mutate:  func(spec Spec) Spec { spec.Code = "terminal_realm_unavailable"; return spec },
			wantErr: ErrUnregisteredCode,
		},
		{
			name: "code registered only by a later version",
			mutate: func(spec Spec) Spec {
				spec.Version = Version100
				spec.Code = "terminal_backend_untrusted"
				return spec
			},
			wantErr: ErrUnregisteredCode,
		},
		{
			name:    "unregistered version",
			mutate:  func(spec Spec) Spec { spec.Version = "1.4.0"; return spec },
			wantErr: ErrUnsupportedVersion,
		},
	}
	for _, item := range cases {
		test.Run(item.name, func(test *testing.T) {
			if _, err := New(item.mutate(baseSpec())); !errors.Is(err, item.wantErr) {
				test.Fatalf("error = %v, want %v", err, item.wantErr)
			}
		})
	}
}

// TestNewAdmitsEveryBoundaryValue is the complement: the bounds are refused one
// step past the limit and admitted exactly at it, so the refusals above are not
// passing because the validator refuses everything.
func TestNewAdmitsEveryBoundaryValue(test *testing.T) {
	cases := []struct {
		name   string
		mutate func(Spec) Spec
	}{
		{name: "one-character message", mutate: func(spec Spec) Spec { spec.Message = "x"; return spec }},
		{
			name:   "message exactly at the character bound",
			mutate: func(spec Spec) Spec { spec.Message = strings.Repeat("é", 4096); return spec },
		},
		{name: "sixty-four detail keys", mutate: func(spec Spec) Spec { spec.Details = countedDetails(64); return spec }},
		{
			name:   "detail key exactly at the grammar length",
			mutate: func(spec Spec) Spec { spec.Details = Details{"a" + strings.Repeat("b", 63): "v"}; return spec },
		},
		{
			name:   "detail value nested exactly four containers deep",
			mutate: func(spec Spec) Spec { spec.Details = Details{"nested": nestedValue(4)}; return spec },
		},
	}
	for _, item := range cases {
		test.Run(item.name, func(test *testing.T) {
			if _, err := New(item.mutate(baseSpec())); err != nil {
				test.Fatalf("a value at the declared bound was refused: %v", err)
			}
		})
	}
}

// TestRetryableRefusedForEveryForbiddenClass proves the retryability gate by
// narrowing it. Each forbidden class and each individually disqualified code is
// exercised separately, so a policy that kept only one of them would fail here;
// and one representative code from every other exit class is admitted, so a
// policy that refused everything would fail too.
func TestRetryableRefusedForEveryForbiddenClass(test *testing.T) {
	forbidden := []struct {
		name    string
		version Version
		code    Code
	}{
		{name: "exit 7 authentication", version: Version100, code: "authentication_failed"},
		{name: "exit 7 allowlist", version: Version100, code: "peer_not_allowlisted"},
		{name: "exit 7 directory field", version: Version120, code: "field_forbidden"},
		{name: "exit 7 terminal authorization", version: Version130, code: "terminal_backend_unauthorized"},
		{name: "exit 16 confirmation", version: Version100, code: "confirmation_required"},
		{name: "exit 16 policy", version: Version100, code: "policy_refused"},
		{name: "exit 16 secret policy", version: Version100, code: "secret_policy_violation"},
		{name: "exit 130 interrupt", version: Version100, code: "interrupted"},
		{name: "ambiguous transaction effect", version: Version110, code: "transaction_unknown"},
		{name: "uncertain operation", version: Version120, code: "operation_uncertain"},
		{name: "stale backend generation", version: Version130, code: "terminal_backend_stale_generation"},
	}
	for _, item := range forbidden {
		test.Run("refused/"+item.name, func(test *testing.T) {
			spec := baseSpec()
			spec.Version = item.version
			spec.Code = item.code
			spec.Retryable = true
			if _, err := New(spec); !errors.Is(err, ErrInvalidStructuredError) {
				test.Fatalf("retryable was admitted for %s: %v", item.code, err)
			}
			// The same object without the claim is valid, so the refusal is
			// about the retry claim and not about the code.
			spec.Retryable = false
			if _, err := New(spec); err != nil {
				test.Fatalf("%s is not constructible at all: %v", item.code, err)
			}
			if _, forbiddenNow := RetryabilityRefusal(item.code, 0); !forbiddenNow {
				status, err := ExitCodeFor(item.version, item.code)
				if err != nil {
					test.Fatalf("ExitCodeFor: %v", err)
				}
				if _, byClass := RetryabilityRefusal(item.code, status); !byClass {
					test.Fatalf("%s is refused by neither code nor exit class", item.code)
				}
			}
		})
	}

	admitted := []struct {
		version Version
		code    Code
	}{
		{version: Version100, code: "transport_failure"},
		{version: Version100, code: "owner_unreachable"},
		{version: Version100, code: "quiesce_timeout"},
		{version: Version100, code: "provider_timeout"},
		{version: Version100, code: "workspace_group_busy"},
		{version: Version120, code: "host_offline"},
	}
	for _, item := range admitted {
		test.Run("admitted/"+string(item.code), func(test *testing.T) {
			spec := baseSpec()
			spec.Version = item.version
			spec.Code = item.code
			spec.Retryable = true
			failure, err := New(spec)
			if err != nil {
				test.Fatalf("a retry claim outside every forbidden class was refused: %v", err)
			}
			if !failure.Retryable() {
				test.Fatal("the retry claim was silently dropped")
			}
		})
	}
}

// TestTypedDetailsAreRequiredWhereThePinnedDocumentNamesThem checks the two
// code-to-detail bindings Section 15.3 states, on both the writing and the
// reading side.
func TestTypedDetailsAreRequiredWhereThePinnedDocumentNamesThem(test *testing.T) {
	target := TargetAuth{
		ProviderID:           "codex",
		ProviderBuild:        "0.48.0",
		MacOSVersion:         "15.4",
		TmuxServerGeneration: "7",
		Remediation:          "run the provider auth smoke in the login realm",
	}
	failure, err := NewTargetAuthMissing(Version130, "provider auth smoke did not run", NoIDs(), target, nil)
	if err != nil {
		test.Fatalf("NewTargetAuthMissing: %v", err)
	}
	if failure.ExitCode() != 7 {
		test.Fatalf("target_auth_missing carries exit %d, the pinned mapping is 7", failure.ExitCode())
	}
	for _, key := range targetAuthMissingKeys {
		if _, present := failure.Detail(key); !present {
			test.Fatalf("typed detail %q is missing", key)
		}
	}

	for _, missing := range targetAuthMissingKeys {
		test.Run("writer/"+missing, func(test *testing.T) {
			details, err := target.Details()
			if err != nil {
				test.Fatalf("Details: %v", err)
			}
			delete(details, missing)
			_, err = New(Spec{
				Version: Version130,
				Code:    "target_auth_missing",
				Message: "provider auth smoke did not run",
				IDs:     NoIDs(),
				Details: details,
			})
			if !errors.Is(err, ErrInvalidDetails) {
				test.Fatalf("target_auth_missing without %q was admitted: %v", missing, err)
			}
		})
	}

	encoded, err := json.Marshal(failure)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	if _, err := Decode(Version130, encoded); err != nil {
		test.Fatalf("Decode of a complete typed object: %v", err)
	}
	stripped := strings.Replace(
		string(encoded), `"remediation":"run the provider auth smoke in the login realm",`, "", 1)
	if stripped == string(encoded) {
		test.Fatal("the reader case did not actually remove the typed detail")
	}
	if _, err := Decode(Version130, []byte(stripped)); !errors.Is(err, ErrInvalidDetails) {
		test.Fatalf("a peer object missing a required typed detail was admitted: %v", err)
	}

	for _, empty := range []RealmEvidence{
		{CallerRealm: "aqua", BrokerState: "absent", TmuxServerGeneration: "7", Remediation: "restart the broker"},
		{Capability: "terminal_attach", BrokerState: "absent", TmuxServerGeneration: "7", Remediation: "restart the broker"},
		{Capability: "terminal_attach", CallerRealm: "aqua", TmuxServerGeneration: "7", Remediation: "restart the broker"},
		{Capability: "terminal_attach", CallerRealm: "aqua", BrokerState: "absent", Remediation: "restart the broker"},
		{Capability: "terminal_attach", CallerRealm: "aqua", BrokerState: "absent", TmuxServerGeneration: "7"},
	} {
		if _, err := NewRealmEvidenceUnavailable(Version130, "broker evidence is unsafe", NoIDs(), empty, nil); !errors.Is(err, ErrInvalidDetails) {
			test.Fatalf("an incomplete realm evidence set was admitted: %v", err)
		}
	}
	complete, err := NewRealmEvidenceUnavailable(Version130, "broker evidence is unsafe", NoIDs(), RealmEvidence{
		Capability:           "terminal_attach",
		CallerRealm:          "aqua",
		BrokerState:          "absent",
		TmuxServerGeneration: "7",
		Remediation:          "restart the broker in the login realm",
	}, nil)
	if err != nil {
		test.Fatalf("NewRealmEvidenceUnavailable: %v", err)
	}
	if complete.Code() != "capability_unavailable" || complete.ExitCode() != 6 {
		test.Fatalf("realm evidence failure is %s/%d, want capability_unavailable/6", complete.Code(), complete.ExitCode())
	}
}

func countedDetails(count int) Details {
	details := make(Details, count)
	for index := 0; index < count; index++ {
		details[detailName(index)] = "v"
	}
	return details
}

func detailName(index int) string {
	name := "k"
	for _, digit := range []byte(itoa(index)) {
		name += string(rune('a' + digit - '0'))
	}
	return name
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// nestedValue builds a value that opens exactly depth containers.
func nestedValue(depth int) any {
	var value any = "leaf"
	for index := 0; index < depth; index++ {
		value = map[string]any{"n": value}
	}
	return value
}
