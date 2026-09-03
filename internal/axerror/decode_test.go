package axerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const conformingDocument = `{
	"schema": "urn:ax:schema:error",
	"schema_version": "1.2.0",
	"code": "observation_gap",
	"message": "the inventory batch skipped a sequence",
	"exit_code": 9,
	"retryable": false,
	"operation_id": "0198f4c8-b180-7299-9273-1234567890ab",
	"session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
	"details": {"observed_sequence": "17"}
}`

// TestDecodeRetainsUnknownCodeExitClassWithoutSuccess is the Section 15.3
// reader rule: "an unknown code retains the envelope's exit class and MUST NOT
// be interpreted as success". A code added by a compatible minor is admitted,
// keeps its exit class, and is reported as unrecognized rather than silently
// treated as a known one.
func TestDecodeRetainsUnknownCodeExitClassWithoutSuccess(test *testing.T) {
	document := strings.Replace(conformingDocument, `"observation_gap"`, `"observation_horizon_lost"`, 1)
	failure, err := Decode(Version120, []byte(document))
	if err != nil {
		test.Fatalf("a code added by a compatible minor was refused: %v", err)
	}
	if failure.Code() != "observation_horizon_lost" {
		test.Fatalf("code was rewritten to %q", failure.Code())
	}
	if failure.ExitCode() != 9 {
		test.Fatalf("exit class was not retained: %d", failure.ExitCode())
	}
	if failure.CodeRegistered() {
		test.Fatal("an unregistered code reported as registered")
	}
	if failure.ExitCode() == 0 {
		test.Fatal("an unknown code produced the success exit status")
	}

	// The exit class still has to be a registered failure status, so an unknown
	// code cannot smuggle success or an invented class in with it.
	for _, status := range []string{"0", "1", "18", "99"} {
		smuggled := strings.Replace(document, `"exit_code": 9`, `"exit_code": `+status, 1)
		if smuggled == document {
			test.Fatal("the exit status substitution did not apply")
		}
		if _, err := Decode(Version120, []byte(smuggled)); !errors.Is(err, ErrUnregisteredExit) {
			test.Fatalf("exit status %s was admitted for an unknown code: %v", status, err)
		}
	}

	// An unknown code on an authorization, refusal, or interrupt class still
	// cannot claim a safe retry.
	for _, item := range []struct{ status, name string }{{"7", "authorization"}, {"16", "refusal"}, {"130", "interrupt"}} {
		forged := strings.Replace(document, `"exit_code": 9`, `"exit_code": `+item.status, 1)
		forged = strings.Replace(forged, `"retryable": false`, `"retryable": true`, 1)
		if _, err := Decode(Version120, []byte(forged)); !errors.Is(err, ErrInvalidStructuredError) {
			test.Fatalf("an unknown code forged a retry claim on the %s class: %v", item.name, err)
		}
	}
}

// TestDecodeRefusesClosedShapeViolations narrows the reader. Every case is one
// mutation of a document the reader otherwise accepts.
func TestDecodeRefusesClosedShapeViolations(test *testing.T) {
	if _, err := Decode(Version120, []byte(conformingDocument)); err != nil {
		test.Fatalf("the conforming document was refused: %v", err)
	}
	cases := []struct {
		name     string
		document string
		wantErr  error
	}{
		{
			name:     "foreign schema",
			document: strings.Replace(conformingDocument, "urn:ax:schema:error", "urn:ax:schema:cli-result", 1),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name:     "unsupported major",
			document: strings.Replace(conformingDocument, `"1.2.0"`, `"2.0.0"`, 1),
			wantErr:  ErrUnsupportedMajor,
		},
		{
			name:     "unregistered minor",
			document: strings.Replace(conformingDocument, `"1.2.0"`, `"1.9.0"`, 1),
			wantErr:  ErrUnsupportedVersion,
		},
		{
			name:     "supported but unbound minor",
			document: strings.Replace(conformingDocument, `"1.2.0"`, `"1.3.0"`, 1),
			wantErr:  ErrVersionMismatch,
		},
		{
			name:     "unparseable version",
			document: strings.Replace(conformingDocument, `"1.2.0"`, `"1.2"`, 1),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name: "tenth top-level member",
			document: strings.Replace(
				conformingDocument, `"details":`, `"authority": "granted", "details":`, 1),
			wantErr: ErrInvalidStructuredError,
		},
		{
			name: "extensions member borrowed from another contract",
			document: strings.Replace(
				conformingDocument, `"details":`, `"extensions": {}, "details":`, 1),
			wantErr: ErrInvalidStructuredError,
		},
		{
			name:     "missing details",
			document: strings.Replace(conformingDocument, ",\n\t\"details\": {\"observed_sequence\": \"17\"}", "", 1),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name:     "null details",
			document: strings.Replace(conformingDocument, `{"observed_sequence": "17"}`, "null", 1),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name:     "exit status that contradicts a registered code",
			document: strings.Replace(conformingDocument, `"exit_code": 9`, `"exit_code": 5`, 1),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name:     "code outside the grammar",
			document: strings.Replace(conformingDocument, `"observation_gap"`, `"Observation-Gap"`, 1),
			wantErr:  ErrUnregisteredCode,
		},
		{
			name:     "operation identifier that is not a UUIDv7",
			document: strings.Replace(conformingDocument, "0198f4c8-b180-7299-9273-1234567890ab", "not-a-uuid", 1),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name: "operation identifier of the wrong UUID version",
			document: strings.Replace(
				conformingDocument, "0198f4c8-b180-7299-9273-1234567890ab", "0198f4c8-b180-4299-9273-1234567890ab", 1),
			wantErr: ErrInvalidStructuredError,
		},
		{
			name:     "trailing content after the object",
			document: conformingDocument + `{"schema":"urn:ax:schema:error"}`,
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name:     "detail key naming an excluded class",
			document: strings.Replace(conformingDocument, `"observed_sequence"`, `"access_token"`, 1),
			wantErr:  ErrInvalidDetails,
		},
		{
			name:     "empty message",
			document: strings.Replace(conformingDocument, `"the inventory batch skipped a sequence"`, `""`, 1),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name:     "forged retry claim on a refusal class",
			document: retryClaimOn(`"code": "policy_refused"`, `"exit_code": 16`),
			wantErr:  ErrInvalidStructuredError,
		},
		{
			name:     "forged retry claim on an uncertain operation",
			document: retryClaimOn(`"code": "operation_uncertain"`, `"exit_code": 12`),
			wantErr:  ErrInvalidStructuredError,
		},
	}
	for _, item := range cases {
		test.Run(item.name, func(test *testing.T) {
			if item.document == conformingDocument {
				test.Fatal("the mutation did not apply; the case proves nothing")
			}
			if _, err := Decode(Version120, []byte(item.document)); !errors.Is(err, item.wantErr) {
				test.Fatalf("error = %v, want %v", err, item.wantErr)
			}
		})
	}
}

// TestDecodeRefusesAWrongVersionBeforeReadingItsBody is Section 15.1's
// "Receivers MUST NOT parse a different major's payload far enough to trust its
// error code, retryable bit, details, or authority fields". The document below
// is wrong in every one of those members as well as in its major. The reader
// must refuse on the major, and must return no object at all, so that nothing
// downstream can read a value the payload supplied.
func TestDecodeRefusesAWrongVersionBeforeReadingItsBody(test *testing.T) {
	hostile := `{
		"schema": "urn:ax:schema:error",
		"schema_version": "2.0.0",
		"code": "not_found",
		"message": "ok",
		"exit_code": 0,
		"retryable": true,
		"details": {"authorized": true}
	}`
	failure, err := Decode(Version100, []byte(hostile))
	if !errors.Is(err, ErrUnsupportedMajor) {
		test.Fatalf("error = %v, want %v", err, ErrUnsupportedMajor)
	}
	if failure != nil {
		test.Fatalf("a different-major payload produced a usable object: %+v", failure)
	}
	if strings.Contains(err.Error(), "not_found") || strings.Contains(err.Error(), "authorized") {
		test.Fatalf("the refusal quoted the untrusted body: %v", err)
	}
	if _, err := Decode(Version("2.0.0"), []byte(hostile)); !errors.Is(err, ErrUnsupportedVersion) {
		test.Fatal("the reader can be asked to expect an unsupported major")
	}
}

func retryClaimOn(code, exit string) string {
	document := strings.Replace(conformingDocument, `"code": "observation_gap"`, code, 1)
	document = strings.Replace(document, `"exit_code": 9`, exit, 1)
	return strings.Replace(document, `"retryable": false`, `"retryable": true`, 1)
}

// TestReaderAndAccessorEdges covers the remaining reader branches: a
// non-integer or out-of-range exit status, both optional identifiers in their
// explicit-null and wrong-type forms, and the sorted key accessor.
func TestReaderAndAccessorEdges(test *testing.T) {
	// Every wrong JSON type is refused by the same parse over the raw document
	// bytes: the quotes, the point, the exponent and the letters are all bytes
	// strconv.ParseInt rejects. A reader that took exit_code through a
	// json.Number field, or that stripped quotes before parsing, would admit
	// the string form and let a peer write its exit status as text.
	for _, exit := range []string{"9.5", "\"9\"", "\"\"", "1e1", "4294967296", "true", "[]", "{}", "[9]"} {
		document := strings.Replace(conformingDocument, `"exit_code": 9`, `"exit_code": `+exit, 1)
		if document == conformingDocument {
			test.Fatal("the exit status substitution did not apply")
		}
		if _, err := Decode(Version120, []byte(document)); err == nil {
			test.Fatalf("exit_code %s was admitted", exit)
		}
	}

	nulled := strings.Replace(
		conformingDocument, `"operation_id": "0198f4c8-b180-7299-9273-1234567890ab"`, `"operation_id": null`, 1)
	if nulled == conformingDocument {
		test.Fatal("the identifier substitution did not apply")
	}
	failure, err := Decode(Version120, []byte(nulled))
	if err != nil {
		test.Fatalf("an explicitly null optional identifier was refused: %v", err)
	}
	if _, known := failure.IDs().Operation(); known {
		test.Fatal("an explicit null was read as a known identifier")
	}
	if _, known := failure.IDs().Session(); !known {
		test.Fatal("the known session identifier was lost")
	}

	typed := strings.Replace(
		conformingDocument, `"session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, `"session_id": 17`, 1)
	if typed == conformingDocument {
		test.Fatal("the identifier substitution did not apply")
	}
	if _, err := Decode(Version120, []byte(typed)); !errors.Is(err, ErrInvalidStructuredError) {
		test.Fatalf("a non-string session identifier was admitted: %v", err)
	}

	keys := failure.DetailKeys()
	if len(keys) != 1 || keys[0] != "observed_sequence" {
		test.Fatalf("DetailKeys() = %v, want the one diagnostic key", keys)
	}
	if _, present := failure.Detail("absent"); present {
		test.Fatal("Detail reported a key the object does not carry")
	}
}

// TestCodesForRefusesAnUnregisteredVersion keeps the measured denominator from
// being taken for a version the registry does not carry.
func TestCodesForRefusesAnUnregisteredVersion(test *testing.T) {
	for _, version := range []Version{"1.4.0", "2.0.0", "", "1.0"} {
		if _, err := CodesFor(version); !errors.Is(err, ErrUnsupportedVersion) {
			test.Fatalf("CodesFor(%q) returned a code set: %v", version, err)
		}
	}
}

// TestDecodeRequiresEveryClosedMember removes one required member at a time. A
// reader that treated an absent member as a zero value would admit each of
// these, and the resulting object would carry a code, a message, or a retry
// claim nobody wrote.
func TestDecodeRequiresEveryClosedMember(test *testing.T) {
	members := map[string]string{
		"schema":         `"schema": "urn:ax:schema:error",`,
		"schema_version": `"schema_version": "1.2.0",`,
		"code":           `"code": "observation_gap",`,
		"message":        `"message": "the inventory batch skipped a sequence",`,
		"exit_code":      `"exit_code": 9,`,
		"retryable":      `"retryable": false,`,
	}
	for name, fragment := range members {
		test.Run(name, func(test *testing.T) {
			document := strings.Replace(conformingDocument, fragment, "", 1)
			if document == conformingDocument {
				test.Fatalf("the %s removal did not apply; the case proves nothing", name)
			}
			if _, err := Decode(Version120, []byte(document)); err == nil {
				test.Fatalf("a document without %s was admitted", name)
			}
		})
	}
}

// TestDecodeBodyGuardsAnUnregisteredVersion exercises the guard that Decode's
// own entry check makes unreachable from outside the package. It is kept
// because decodeBody resolves a code against a version, and a future caller
// that reached it without the entry check must not receive an object built on
// an unresolvable registry lookup.
func TestDecodeBodyGuardsAnUnregisteredVersion(test *testing.T) {
	exitCode := json.RawMessage("9")
	schema, version, code := Schema, "1.4.0", "observation_gap"
	message, retryable := "measured", false
	details := Details{}
	document := &wireDocument{
		Schema:        &schema,
		SchemaVersion: &version,
		Code:          &code,
		Message:       &message,
		ExitCode:      &exitCode,
		Retryable:     &retryable,
		Details:       &details,
	}
	if _, err := decodeBody(Version("1.4.0"), document); !errors.Is(err, ErrUnsupportedVersion) {
		test.Fatalf("decodeBody built an object on an unregistered version: %v", err)
	}
}

// TestTypedDetailsAreCheckedOnTheReadingSideToo covers the reader's typed
// detail requirement for values that are present but not usable text.
func TestTypedDetailsAreCheckedOnTheReadingSideToo(test *testing.T) {
	base := `{"schema":"urn:ax:schema:error","schema_version":"1.3.0","code":"target_auth_missing",` +
		`"message":"provider auth smoke did not run","exit_code":7,"retryable":false,` +
		`"details":{"provider_id":"codex","provider_build":"0.48.0","macos_version":"15.4",` +
		`"tmux_server_generation":"7","remediation":%s}}`
	if _, err := Decode(Version130, []byte(fmt.Sprintf(base, `"restart the broker"`))); err != nil {
		test.Fatalf("a complete typed object was refused: %v", err)
	}
	for _, unusable := range []string{`""`, `17`, `null`, `["restart"]`} {
		if _, err := Decode(Version130, []byte(fmt.Sprintf(base, unusable))); !errors.Is(err, ErrInvalidDetails) {
			test.Fatalf("typed detail value %s was admitted: %v", unusable, err)
		}
	}
}
