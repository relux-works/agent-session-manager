package sessquery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// TestRev14CapabilitySeal proves that the profile derivation boundary does
// not treat a zero or field-assembled admittedCheckpoint as authority. The
// product entry points are intentionally exercised directly here because a
// missing seal must fail before any repository or event read can occur.
func TestRev14CapabilitySeal(t *testing.T) {
	recordBytes := record(t, idA, "alpha")
	forged := admittedCheckpoint{
		seal: &checkpointSeal{
			digest:              "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			sessionID:           idA,
			leaseEpoch:          1,
			leaseID:             lease,
			creatorHostID:       hostA,
			hasProviderManifest: true,
			eventHeads:          []string{zeroDigest},
		},
	}
	entries := map[string]func(admittedCheckpoint) error{
		"checkCheckpointProfileAuthority": func(checkpoint admittedCheckpoint) error {
			return checkCheckpointProfileAuthority(checkpoint, nil, nil)
		},
		"sessionCreationProfile": func(checkpoint admittedCheckpoint) error {
			_, _, err := sessionCreationProfile(checkpoint, recordBytes)
			return err
		},
		"profileClosure": func(checkpoint admittedCheckpoint) error {
			_, err := profileClosure(checkpoint, zeroDigest, nil)
			return err
		},
		"eventProfilePair": func(checkpoint admittedCheckpoint) error {
			_, err := eventProfilePair(checkpoint, nil, sessrepo.EventSummary{})
			return err
		},
		"eventProfileTarget": func(checkpoint admittedCheckpoint) error {
			_, err := eventProfileTarget(checkpoint, nil, sessrepo.EventSummary{})
			return err
		},
		"eventPayloadMembers": func(checkpoint admittedCheckpoint) error {
			_, err := eventPayloadMembers(checkpoint, nil, sessrepo.EventSummary{})
			return err
		},
		"checkClosureProfilePairs": func(checkpoint admittedCheckpoint) error {
			return checkClosureProfilePairs(checkpoint, "standard", zeroDigest, nil, nil, nil, nil)
		},
		"referencedCheckpointProfile": func(checkpoint admittedCheckpoint) error {
			_, _, _, _, err := referencedCheckpointProfile(checkpoint, sessrepo.EventSummary{}, "standard", zeroDigest, nil, nil, nil)
			return err
		},
		"checkProfilePairSource": func(checkpoint admittedCheckpoint) error {
			return checkProfilePairSource(checkpoint, sessrepo.EventSummary{}, profilePair{}, "standard", "", false, nil, nil)
		},
	}
	for name, entry := range entries {
		for label, checkpoint := range map[string]admittedCheckpoint{
			"zero":   {},
			"forged": forged,
		} {
			t.Run(fmt.Sprintf("%s/%s", name, label), func(t *testing.T) {
				err := entry(checkpoint)
				if err == nil {
					t.Fatalf("%s accepted an unsealed checkpoint", label)
				}
				if !errors.Is(err, ErrObservationUnavailable) {
					t.Fatalf("%s refusal = %v, want observation_unavailable", label, err)
				}
			})
		}
	}
}

// TestRev14RecordConsumptionCensusRejectsAlternatePaths expands the
// package-wide type gate with plants that preserve reviewer-visible tokens
// while changing the authority object or construction path. A plant is
// accepted only if the same production census rejects it.
func TestRev14RecordConsumptionCensusRejectsAlternatePaths(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	packageDir := filepath.Dir(file)
	plants := map[string]string{
		"interface_assertion.go": `package sessquery

func review14Interface(v any) string { return v.(validatedCheckpoint).Digest }
`,
		"package_closure.go": `package sessquery

var review14Closure = func(v validatedCheckpoint) string { return v.Digest }
`,
		"unresolvable_callee.go": `package sessquery

func review14Unresolved() { missingReview14Callee() }
`,
		"method_named_admit.go": `package sessquery

type review14Receiver struct{}

func (review14Receiver) admitCheckpoint(v validatedCheckpoint) (admittedCheckpoint, error) {
		return admittedCheckpoint{}, nil
}
`,
		"method_raw_only.go": `package sessquery

type review14RawReceiver struct{}

func (review14RawReceiver) admitCheckpoint(v validatedCheckpoint) string {
		return v.Digest
}
`,
		"field_assembly.go": `package sessquery

func review14FieldAssembly(raw validatedCheckpoint) admittedCheckpoint {
		var checkpoint admittedCheckpoint
		checkpoint.seal = &checkpointSeal{digest: raw.Digest, sessionID: raw.SessionID}
		return checkpoint
}
`,
		"embedding.go": `package sessquery

type review14Embedding struct{ admittedCheckpoint }
`,
		"raw_copy.go": `package sessquery

func review14RawCopy(raw validatedCheckpoint) admittedCheckpoint {
		var checkpoint admittedCheckpoint
		checkpoint.seal = &checkpointSeal{digest: raw.Digest, sessionID: raw.SessionID}
		return checkpoint
}
`,
		"capability_map.go": `package sessquery

var review14Capabilities = map[string]admittedCheckpoint{}

func review14Populate(raw validatedCheckpoint) {
		var checkpoint admittedCheckpoint
		checkpoint.seal = &checkpointSeal{digest: raw.Digest, sessionID: raw.SessionID}
		review14Capabilities[raw.Digest] = checkpoint
}
`,
		"mint_seal_without_composite.go": `package sessquery

func review14MintSeal() *checkpointSeal {
		v := new(checkpointSeal)
		v.token = new(checkpointSealToken)
		v.sessionID = "invented"
		return v
}
`,
		"generic_seal_constructor.go": `package sessquery

func review14GenericNew[T any]() *T { return new(T) }

func review14GenericSeal() *checkpointSeal {
		v := review14GenericNew[checkpointSeal]()
		v.token = review14GenericNew[checkpointSealToken]()
		return v
}
`,
		"token_any_copy.go": `package sessquery

func review14TokenAnyCopy(token *checkpointSealToken) any {
		var copied any = token
		return copied
}
`,
		"reflect_type_for.go": `package sessquery

import "reflect"

func review14ReflectSealType() reflect.Type { return reflect.TypeFor[checkpointSeal]() }
`,
		"unsafe_pointer.go": `package sessquery

import "unsafe"

func review14UnsafeTokenPointer() unsafe.Pointer { return unsafe.Pointer(new(checkpointSealToken)) }
`,
	}
	for name, plant := range plants {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			rev12CopyProductionSources(t, packageDir, dir)
			if err := os.WriteFile(filepath.Join(dir, name), []byte(plant), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := rev12CheckRecordConsumptionSource(dir); err == nil {
				t.Fatalf("record-consumption gate admitted %s", name)
			} else {
				t.Logf("rejected %s: %v", name, err)
			}
		})
	}
}

// TestRev14CallbackProvenanceRejectsUnknownFunctions keeps the census from
// treating an unresolved function-value callee as the profile's admission
// parameter. Both plants type-check, but neither callee is the exact parameter
// object recorded for the approved profile entry. The first callback is local
// to the approved profile flow so its rejection exercises callback provenance,
// rather than the independent package-scope sealed-value guard.
func TestRev14CallbackProvenanceRejectsUnknownFunctions(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	packageDir := filepath.Dir(file)
	plants := map[string]string{
		"unknown_callback.go": `package sessquery
`,
		"callback_field.go": `package sessquery

type review14CallbackHolder struct {
		callback checkpointAdmission
}

var review14Callback review14CallbackHolder
`,
	}
	const anchor = "ref, err := admission(checkpointID, authority.sessionID, summary)"
	replacements := map[string]string{
		"unknown_callback.go": "review14UnknownAdmission := admission\n\tref, err := review14UnknownAdmission(checkpointID, authority.sessionID, summary)",
		"callback_field.go":   "ref, err := review14Callback.callback(checkpointID, authority.sessionID, summary)",
	}
	for name, plant := range plants {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			rev12CopyProductionSources(t, packageDir, dir)
			if err := os.WriteFile(filepath.Join(dir, name), []byte(plant), 0o600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "lease.go")
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(contents), anchor) != 1 {
				t.Fatalf("anchor count = %d, want 1", strings.Count(string(contents), anchor))
			}
			contents = []byte(strings.Replace(string(contents), anchor, replacements[name], 1))
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := rev12CheckRecordConsumptionSource(dir); err == nil {
				t.Fatalf("record-consumption gate admitted %s", name)
			} else if strings.Contains(err.Error(), "type-check") {
				t.Fatalf("%s did not reach the census: %v", name, err)
			} else {
				t.Logf("rejected %s: %v", name, err)
			}
		})
	}
}
