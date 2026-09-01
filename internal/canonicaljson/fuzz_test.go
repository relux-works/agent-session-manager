package canonicaljson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestCanonicalizeRejectsMalformedRecursiveObjectShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"nested duplicate member", `{"outer":{"a":1,"\u0061":2}}`},
		{"array nested duplicate member", `{"outer":[{"a":1,"a":2}]}`},
		{"nested non-string member name", `{"outer":{1:"value"}}`},
		{"nested lone surrogate member name", `{"outer":{"\ud800":1}}`},
		{"nested lone surrogate value", `{"outer":[{"value":"\udc00"}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Canonicalize([]byte(test.input)); err == nil || !errors.Is(err, ErrInvalidJSON) {
				t.Fatalf("Canonicalize(%s) error = %v, want recursive JSON-shape refusal", test.name, err)
			}
		})
	}
}

// FuzzCanonicalizeRoundTrip proves that the Canonicalize production entry is
// stable under read-back and insignificant outer whitespace. Invalid inputs
// are refusals, never panics or partially accepted values.
func FuzzCanonicalizeRoundTrip(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"b":2,"a":{"δ":4,"c":3}}`),
		[]byte(`{"\uE000":1,"\uD800\uDC00":2}`),
		[]byte(`{"outer":{"a":1,"\u0061":2}}`),
		[]byte(`{"outer":{1:"value"}}`),
		[]byte(`{"value":"\ud800"}`),
		[]byte{'{', '"', 'v', '"', ':', '"', 0xff, '"', '}'},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 64*1024 {
			return
		}
		canonical, err := Canonicalize(input)
		if err != nil {
			return
		}
		again, err := Canonicalize(canonical)
		if err != nil {
			t.Fatalf("Canonicalize accepted %q but refused canonical read-back %q: %v", input, canonical, err)
		}
		if !bytes.Equal(again, canonical) {
			t.Fatalf("canonicalization is not idempotent: first %q, second %q", canonical, again)
		}
		padded := append([]byte(" \r\n\t"), canonical...)
		padded = append(padded, '\n', ' ')
		withWhitespace, err := Canonicalize(padded)
		if err != nil || !bytes.Equal(withWhitespace, canonical) {
			t.Fatalf("insignificant whitespace changed canonical value: got %q, %v; want %q", withWhitespace, err, canonical)
		}
	})
}

// FuzzObjectIdentityRepresentationInvariant drives CalculateObjectIdentity and
// VerifyObjectIdentity with the same nested logical value in raw and canonical
// representations. Key order and whitespace must not alter the identity.
func FuzzObjectIdentityRepresentationInvariant(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"b":2,"a":1}`),
		[]byte(" { \n \"z\" : [ { \"b\" : 2, \"a\" : 1 } ] } \r\n"),
		[]byte(`{"\uE000":1,"\uD800\uDC00":2}`),
		[]byte(`[null,true,false,"данные",9007199254740991]`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, nested []byte) {
		if len(nested) > 32*1024 {
			return
		}
		canonicalNested, err := Canonicalize(nested)
		if err != nil {
			return
		}

		first, second := identityFuzzRepresentations(nested, canonicalNested, zeroDigest)

		firstDigest, firstField, err := CalculateObjectIdentity(first)
		if err != nil {
			return // The nested value can be valid JCS but outside AX's integer-only model.
		}
		secondDigest, secondField, err := CalculateObjectIdentity(second)
		if err != nil {
			t.Fatalf("canonical/reordered representation refused after equivalent input was accepted: %v", err)
		}
		if firstDigest != secondDigest || firstField != secondField || firstField != SelfRecordID {
			t.Fatalf("identity changed across representation: first %q/%q, second %q/%q", firstDigest, firstField, secondDigest, secondField)
		}

		claimedFirst, claimedSecond := identityFuzzRepresentations(nested, canonicalNested, firstDigest.String())
		for name, claimed := range map[string][]byte{"raw": claimedFirst, "canonical": claimedSecond} {
			verified, field, err := VerifyObjectIdentity(claimed)
			if err != nil || verified != firstDigest || field != SelfRecordID {
				t.Fatalf("VerifyObjectIdentity(%s representation) = %q/%q, %v; want %q/%q", name, verified, field, err, firstDigest, SelfRecordID)
			}
			recalculated, _, err := CalculateObjectIdentity(claimed)
			if err != nil || recalculated != firstDigest {
				t.Fatalf("accepted %s identity later refused or changed: %q, %v", name, recalculated, err)
			}
		}
	})
}

// FuzzClosedIdentityShapeRefusal mutates the concrete invalid shapes that
// previously bypassed both identity production entries. Every generated case
// remains structurally invalid, so acceptance is always a gate failure rather
// than an ambiguous fuzz classification.
func FuzzClosedIdentityShapeRefusal(f *testing.F) {
	for shape, salt := range []string{
		"blob-unknown-top-level",
		"blob-unknown-nested",
		"blob-index-order",
		"blob-offset-gap",
		"blob-size-bound",
		"blob-coverage",
		"manifest-unknown-top-level",
		"manifest-entry-unknown-nested",
		"manifest-snapshot-unknown-nested",
		"manifest-member-unknown-nested",
		"migration-provenance-unknown-nested",
		"manifest-invalid-extension-key",
		"record-invalid-extension-key",
		"manifest-repository-identity-bound",
		"manifest-remote-name-bound",
		"manifest-pack-digest",
		"manifest-index-stage-four",
		"manifest-uninitialized-submodule-state",
		"manifest-managed-tree-identity-bound",
		"manifest-git-ref",
		"manifest-git-url",
		"manifest-sparse-pair",
		"manifest-hardlink-target",
		"manifest-symlink-escape",
		"manifest-index-count",
		"record-missing-created-by-host",
		"record-malformed-subject",
		"record-impossible-created-at",
		"record-unknown-top-level",
		"record-launch-plan-unknown-nested",
		"record-invalid-name",
	} {
		f.Add(uint8(shape), salt)
	}

	f.Fuzz(func(t *testing.T, shape uint8, salt string) {
		if len(salt) > 1024 {
			return
		}
		unknown := fmt.Sprintf("unexpected_%x", []byte(salt))
		var object map[string]any
		var selfField SelfField

		switch shape % 31 {
		case 0:
			object, selfField = validBlobDescriptorObject(), SelfDescriptorID
			object[unknown] = true
		case 1:
			object, selfField = validBlobDescriptorObject(), SelfDescriptorID
			firstChunk(object)[unknown] = true
		case 2:
			object, selfField = validBlobDescriptorObject(), SelfDescriptorID
			firstChunk(object)["index"] = json.Number("1")
		case 3:
			object, selfField = validBlobDescriptorObject(), SelfDescriptorID
			firstChunk(object)["offset"] = json.Number("1")
		case 4:
			object, selfField = validBlobDescriptorObject(), SelfDescriptorID
			firstChunk(object)["size"] = json.Number("0")
		case 5:
			object, selfField = validBlobDescriptorObject(), SelfDescriptorID
			object["size"] = json.Number("12")
		case 6:
			object, selfField = validTransferManifestObject("workspace_tree"), SelfManifestID
			object[unknown] = true
		case 7:
			object, selfField = validTransferManifestObject("workspace_tree"), SelfManifestID
			object["entries"] = []any{map[string]any{"path": "src", "type": "directory", "mode": json.Number("493"), unknown: true}}
		case 8:
			object, selfField = validTransferManifestObject("workspace_group"), SelfManifestID
			object["workspace_snapshot"].(map[string]any)[unknown] = true
		case 9:
			object, selfField = validTransferManifestObject("workspace_group"), SelfManifestID
			snapshot := object["workspace_snapshot"].(map[string]any)
			snapshot["members"].([]any)[0].(map[string]any)[unknown] = true
		case 10:
			object, selfField = validSessionRecordV1Object(), SelfRecordID
			object["extensions"] = map[string]any{
				"works.relux.ax.migrated-from": map[string]any{
					"schema_id":      "urn:ax:schema:session-record",
					"schema_version": "0.9.0",
					"object_id":      digestWithDigit('6'),
					unknown:          true,
				},
			}
		case 11:
			object, selfField = validTransferManifestObject("workspace_tree"), SelfManifestID
			object["extensions"].(map[string]any)[unknown] = true
		case 12:
			object, selfField = genericExtensionIdentityObject(map[string]any{unknown: true}), SelfRecordID
		case 13:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["repository_identity"] = strings.Repeat("界", 257)
		case 14:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["remotes"].([]any)[0].(map[string]any)["name"] = strings.Repeat("界", 129)
		case 15:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["object_pack"].(map[string]any)["blob_id"] = "not-a-digest"
		case 16:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["index"].(map[string]any)["entries"].([]any)[0].(map[string]any)["stage"] = json.Number("4")
		case 17:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["submodules"].([]any)[0].(map[string]any)["head"] = map[string]any{
				"mode": "detached", "oid": "sha1:" + strings.Repeat("3", 40), "ref": nil,
			}
		case 18:
			object, selfField = validTransferManifestObject("workspace_group"), SelfManifestID
			gitWorkspaceMember(object)["tree_identity"] = strings.Repeat("界", 257)
		case 19:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["upstream_ref"] = "HEAD"
		case 20:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["remotes"].([]any)[0].(map[string]any)["fetch_url"] = "https://token@example.com/repo.git"
		case 21:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			features := gitWorkspaceMember(object)["features"].(map[string]any)
			features["sparse_checkout"] = true
			features["sparse_patterns_blob_id"] = digestWithDigit('8')
		case 22:
			object, selfField = validTransferManifestObject("workspace_tree"), SelfManifestID
			object["entries"] = []any{map[string]any{"path": "hard", "type": "hardlink", "mode": json.Number("420"), "target_path": "missing"}}
		case 23:
			object, selfField = validTransferManifestObject("workspace_tree"), SelfManifestID
			object["entries"] = []any{map[string]any{"path": "link", "type": "symlink", "mode": json.Number("511"), "target": "../escape"}}
		case 24:
			object, selfField = validGitWorkspaceGroupObject(), SelfManifestID
			gitWorkspaceMember(object)["index"].(map[string]any)["entry_count"] = json.Number("99")
		case 25:
			object, selfField = validSessionRecordV1Object(), SelfRecordID
			delete(object, "created_by_host_id")
		case 26:
			object, selfField = validSessionRecordV1Object(), SelfRecordID
			object["subject_id"] = "not-a-uuid"
		case 27:
			object, selfField = validSessionRecordV1Object(), SelfRecordID
			object["created_at"] = "2023-02-29T12:00:00.000Z"
		case 28:
			object, selfField = validSessionRecordV1Object(), SelfRecordID
			object[unknown] = true
		case 29:
			object, selfField = validSessionRecordV1Object(), SelfRecordID
			object["launch_plan"].(map[string]any)[unknown] = true
		case 30:
			object, selfField = validSessionRecordV1Object(), SelfRecordID
			object["name"] = "-" + salt
		}
		assertIdentityEntriesRefuseShape(t, mustJSON(t, object), selfField)
	})
}

func identityFuzzRepresentations(nested, canonicalNested []byte, claim string) ([]byte, []byte) {
	first := []byte(fmt.Sprintf(
		`{"record_id":%q,"schema":"urn:ax:schema:session-record","schema_version":"1.0.0","subject_id":"0198f4c8-3e70-7a11-8a2b-1234567890ab","session_id":"0198f4c8-3e70-7a11-8a2b-1234567890ab","name":"payments-api","kind":"direct","created_at":"2026-08-19T04:00:00.000Z","created_by_host_id":"0198f4c8-4a10-7b22-8b3c-1234567890ab","provider_id":"codex","workspace_group_id":"0198f4c8-5b20-7c33-8c4d-1234567890ab","execution_profile":"yolo","launch_plan":{"argv":["codex"],"cwd_workspace_id":"0198f4c8-6c30-7d44-8d5e-1234567890ab","cwd_relative":"src","env_names":[],"env_literals":{},"contains_secrets":false,"extensions":{}},"task_board":null,"fork_provenance":null,"extensions":{"example.fuzz":%s}}`,
		claim,
		nested,
	))
	second := []byte(fmt.Sprintf(
		" {\r\n  \"extensions\" : { \"example.fuzz\" : %s },\r\n  \"fork_provenance\" : null,\r\n  \"task_board\" : null,\r\n  \"launch_plan\" : { \"extensions\" : {}, \"contains_secrets\" : false, \"env_literals\" : {}, \"env_names\" : [], \"cwd_relative\" : \"src\", \"cwd_workspace_id\" : \"0198f4c8-6c30-7d44-8d5e-1234567890ab\", \"argv\" : [\"codex\"] },\r\n  \"execution_profile\" : \"yolo\",\r\n  \"workspace_group_id\" : \"0198f4c8-5b20-7c33-8c4d-1234567890ab\",\r\n  \"provider_id\" : \"codex\",\r\n  \"created_by_host_id\" : \"0198f4c8-4a10-7b22-8b3c-1234567890ab\",\r\n  \"created_at\" : \"2026-08-19T04:00:00.000Z\",\r\n  \"kind\" : \"direct\",\r\n  \"name\" : \"payments-api\",\r\n  \"session_id\" : \"0198f4c8-3e70-7a11-8a2b-1234567890ab\",\r\n  \"subject_id\" : \"0198f4c8-3e70-7a11-8a2b-1234567890ab\",\r\n  \"schema_version\" : \"1.0.0\",\r\n  \"schema\" : \"urn:ax:schema:session-record\",\r\n  \"record_id\" : %q\r\n}\r\n",
		canonicalNested,
		claim,
	))
	return first, second
}
