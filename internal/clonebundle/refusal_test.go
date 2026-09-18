package clonebundle

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Negative tests for the raw manifest, capture items, source basis,
// and capture boundary gates. Every row drives a production entry
// point and asserts the refusal; rows that share a verdict shape
// assert a message fragment so the firing gate is identified.

func mustRawBytes(t *testing.T) []byte {
	t.Helper()
	identity := fixtureNativeIdentity()
	digest, err := IdentityDigest(identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	built, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
		"native-session-alpha", digest.String(), fixtureDigest("test-capture-plan"), validEntryInputs(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	return built
}

func tamper(t *testing.T, data []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	mutate(object)
	out, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func requireRefusal(t *testing.T, err error, fragment string) {
	t.Helper()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
	if fragment != "" && !strings.Contains(err.Error(), fragment) {
		t.Fatalf("error = %v, want fragment %q", err, fragment)
	}
}

func TestRawManifestRefusals(t *testing.T) {
	identity := fixtureNativeIdentity()
	identityDigest, err := IdentityDigest(identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	build := func(inputs []EntryInput) error {
		_, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
			"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil)
		return err
	}
	t.Run("build", func(t *testing.T) {
		base := validEntryInputs(t)
		clone := func() []EntryInput {
			out := make([]EntryInput, len(base))
			copy(out, base)
			return out
		}
		cases := []struct {
			name     string
			mutate   func([]EntryInput) []EntryInput
			fragment string
		}{
			{"unknown class", func(in []EntryInput) []EntryInput { in[0].Class = "mystery"; return in }, "outside the raw subset"},
			{"credential forbidden", func(in []EntryInput) []EntryInput { in[0].Class = "credential"; return in }, "outside the raw subset"},
			{"machine_auth forbidden", func(in []EntryInput) []EntryInput { in[0].Class = "machine_auth"; return in }, "outside the raw subset"},
			{"runtime_state forbidden", func(in []EntryInput) []EntryInput { in[0].Class = "runtime_state"; return in }, "outside the raw subset"},
			{"transient_lock forbidden", func(in []EntryInput) []EntryInput { in[0].Class = "transient_lock"; return in }, "outside the raw subset"},
			{"unsorted entries", func(in []EntryInput) []EntryInput { in[0], in[1] = in[1], in[0]; return in }, "sorted unique"},
			{"duplicate entries", func(in []EntryInput) []EntryInput { in[1].NativeItemKey = in[0].NativeItemKey; return in }, "sorted unique"},
			{"absolute key", func(in []EntryInput) []EntryInput { in[0].NativeItemKey = "/etc/passwd"; return in }, "absolute source path"},
			{"uuid key", func(in []EntryInput) []EntryInput { in[0].NativeItemKey = fixtureSessionID; return in }, "fabricated AX"},
			{"control key", func(in []EntryInput) []EntryInput { in[0].NativeItemKey = "a\x00b"; return in }, "control characters"},
			{"credential-url key", func(in []EntryInput) []EntryInput { in[0].NativeItemKey = "https://user:pass@host/x"; return in }, "embedded credentials"},
			{"ax-marker key", func(in []EntryInput) []EntryInput { in[0].NativeItemKey = "urn:ax:session:x"; return in }, "AX identity marker"},
			{"blob disagreement", func(in []EntryInput) []EntryInput { in[0].BlobID = fixtureDigest("other-blob"); return in }, "disagrees"},
			{"size disagreement", func(in []EntryInput) []EntryInput { in[0].ByteCount++; return in }, "disagrees"},
			{"descriptor id mismatch", func(in []EntryInput) []EntryInput {
				in[0].DescriptorID = fixtureDigest("some-other-descriptor")
				return in
			}, "disagrees with entry claim"},
			{"descriptor tampered", func(in []EntryInput) []EntryInput {
				in[0].Descriptor = append([]byte(nil), in[0].Descriptor...)
				in[0].Descriptor[10] ^= 0xff
				return in
			}, "identity fails"},
			{"bad blob digest", func(in []EntryInput) []EntryInput { in[0].BlobID = "nope"; return in }, "not a digest"},
			{"byte_count overflow", func(in []EntryInput) []EntryInput { in[0].ByteCount = maxUint53 + 1; return in }, "exceeds uint53"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				requireRefusal(t, build(cloneMutate(clone(), tc.mutate)), tc.fragment)
			})
		}
	})
	t.Run("decode", func(t *testing.T) {
		valid := mustRawBytes(t)
		cases := []struct {
			name     string
			body     func(t *testing.T) []byte
			fragment string
		}{
			{"unknown member", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["target"] = "x" })
			}, "unknown member"},
			{"missing member", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { delete(o, "entries") })
			}, "misses"},
			{"bad schema", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["schema"] = "urn:ax:schema:blob" })
			}, "not the clone raw object manifest"},
			{"bad version", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["schema_version"] = "2.0.0" })
			}, "not 1.0.0"},
			{"total mismatch", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["total_bytes"] = float64(1) })
			}, "entries sum to"},
			{"excluded class entry", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					entries := o["entries"].([]any)
					entries[0].(map[string]any)["class"] = "credential"
				})
			}, "outside the raw subset"},
			{"unsorted entries", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					entries := o["entries"].([]any)
					entries[0], entries[1] = entries[1], entries[0]
				})
			}, "sorted unique"},
			{"duplicate entries", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					entries := o["entries"].([]any)
					entries[1].(map[string]any)["native_item_key"] = entries[0].(map[string]any)["native_item_key"]
				})
			}, "sorted unique"},
			{"entries not array", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["entries"] = "nope" })
			}, "not an array"},
			{"too many entries", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["entries"] = make([]any, 65537) })
			}, "maximum is 65536"},
			{"unsanitized source id", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["source_native_session_id"] = "/abs/path" })
			}, "absolute source path"},
			{"bad extensions", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["extensions"] = map[string]any{"no-dots": 1} })
			}, "reverse-DNS"},
			{"self digest mismatch", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["raw_object_manifest_id"] = fixtureDigest("forged") })
			}, "omit-self digest"},
			{"not an object", func(t *testing.T) []byte { return []byte(`[1,2]`) }, ""},
			{"duplicate member", func(t *testing.T) []byte {
				return []byte(`{"schema":"urn:ax:schema:clone-raw-object-manifest","schema":"urn:ax:schema:blob"}`)
			}, ""},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeRawObjectManifest(tc.body(t))
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
	t.Run("extension values", func(t *testing.T) {
		build := func(extensions map[string]any) error {
			_, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
				"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), validEntryInputs(t), extensions)
			return err
		}
		cases := []struct {
			name       string
			extensions map[string]any
			fragment   string
		}{
			{"float refused", map[string]any{"com.example.f": 1.5}, "safe-integer"},
			{"exactly 2^53 refused", map[string]any{"com.example.n": uint64(1) << 53}, "safe-integer"},
			{"2^60 refused", map[string]any{"com.example.big": uint64(1) << 60}, "safe-integer"},
			{"exponent refused", map[string]any{"com.example.e": 1e21}, "safe-integer"},
			{"nested float refused", map[string]any{"com.example.o": map[string]any{"n": 0.5}}, "safe-integer"},
			{"nested unsafe refused", map[string]any{"com.example.a": []any{uint64(1) << 53}}, "safe-integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				requireRefusal(t, build(tc.extensions), tc.fragment)
			})
		}
		// Safe integers (edges included), nested objects, and arrays admit.
		admitted := map[string]any{
			"com.example.max":  uint64(9007199254740991),
			"com.example.min":  int64(-9007199254740991),
			"com.example.zero": 0,
			"com.example.o":    map[string]any{"s": "x", "b": true, "n": nil, "a": []any{1, "two"}},
		}
		if err := build(admitted); err != nil {
			t.Fatalf("BuildRawObjectManifest(safe extensions) error = %v", err)
		}
		// Decode refuses unsafe literals and nested duplicates without resealing.
		valid := mustRawBytes(t)
		for _, tc := range []struct {
			name     string
			body     string
			fragment string
		}{
			{"decode float", `"extensions":{"com.example.n":1.5}`, "safe-integer"},
			{"decode 2^53", `"extensions":{"com.example.n":9007199254740992}`, "safe-integer"},
			{"decode 2^53 plus one", `"extensions":{"com.example.n":9007199254740993}`, "safe-integer"},
			{"decode negative 2^53", `"extensions":{"com.example.n":-9007199254740992}`, "safe-integer"},
			{"decode exponent", `"extensions":{"com.example.n":1e2}`, "safe-integer"},
			{"decode nested duplicate", `"extensions":{"com.example.x":{"a":1,"a":2}}`, "duplicate nested member"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				body := strings.Replace(string(valid), `"extensions":{}`, tc.body, 1)
				_, err := DecodeRawObjectManifest([]byte(body))
				requireRefusal(t, err, tc.fragment)
			})
		}
		// A nested duplicate resealed over the collapsed form is still
		// refused: the value gate fires before the identity check, so no
		// reseal can smuggle the collapsed bytes past decoding. The
		// resealed bytes keep the duplicate text with the collapsed
		// form's digest, which decoding would verify without the gate.
		collapsed := strings.Replace(string(valid), `"extensions":{}`, `"extensions":{"com.example.x":{"a":1,"a":2}}`, 1)
		members, fault := decodeStrictObject([]byte(collapsed))
		if fault != nil {
			t.Fatalf("decodeStrictObject() fault = %+v", fault)
		}
		sealed, err := omitSelfDigest(members, rawManifestSelf)
		if err != nil {
			t.Fatalf("omitSelfDigest() error = %v", err)
		}
		var original map[string]any
		if err := json.Unmarshal(valid, &original); err != nil {
			t.Fatal(err)
		}
		originalID, ok := original[rawManifestSelf].(string)
		if !ok {
			t.Fatal("sealed manifest carries no string self id")
		}
		resealed := strings.Replace(collapsed, originalID, sealed.String(), 1)
		_, err = DecodeRawObjectManifest([]byte(resealed))
		requireRefusal(t, err, "duplicate nested member")
	})
	t.Run("operation and tuple", func(t *testing.T) {
		inputs := validEntryInputs(t)
		if _, err := BuildRawObjectManifest("not-a-uuid", []byte(fixtureTupleJSON()),
			"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad operation_id error = %v, want ErrInvalid", err)
		}
		if _, err := BuildRawObjectManifest(fixtureOperationID, []byte(`{"environment_id":"x"}`),
			"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad tuple error = %v, want ErrInvalid", err)
		}
		if _, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
			"native-session-alpha", "nope", fixtureDigest("test-capture-plan"), inputs, nil); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad identity digest error = %v, want ErrInvalid", err)
		}
		if _, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
			"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs,
			map[string]any{"no-dots": 1}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad extensions error = %v, want ErrInvalid", err)
		}
	})
}

func cloneMutate(inputs []EntryInput, mutate func([]EntryInput) []EntryInput) []EntryInput {
	return mutate(inputs)
}

// TestMultibyteStringMeasure pins the Section 1.6 string measure in
// characters, not bytes: a 300-character / 600-byte native key is
// inside string[1..512] and admits, the 512-character edge admits,
// and 513 characters refuse.
func TestMultibyteStringMeasure(t *testing.T) {
	identity := fixtureNativeIdentity()
	identityDigest, err := IdentityDigest(identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	build := func(key string) error {
		inputs := validEntryInputs(t)
		inputs[0].NativeItemKey = key
		if len(inputs) == 2 && inputs[0].NativeItemKey > inputs[1].NativeItemKey {
			inputs[0], inputs[1] = inputs[1], inputs[0]
		}
		_, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
			"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil)
		return err
	}
	if err := build(strings.Repeat("é", 300)); err != nil {
		t.Fatalf("BuildRawObjectManifest(300 chars / 600 bytes) error = %v", err)
	}
	if err := build(strings.Repeat("é", 512)); err != nil {
		t.Fatalf("BuildRawObjectManifest(512 chars) error = %v", err)
	}
	requireRefusal(t, build(strings.Repeat("é", 513)), "string[1..512]")
}

func mustCaptureBytes(t *testing.T) []byte {
	t.Helper()
	built, err := BuildCaptureManifest(validCaptureInput(t))
	if err != nil {
		t.Fatal(err)
	}
	return built
}

func TestCaptureItemRefusals(t *testing.T) {
	count := uint64(7)
	descriptor := fixtureDigest("test-descriptor")
	reason := "credential_excluded"
	build := func(items []CaptureItemInput) error {
		input := validCaptureInput(t)
		input.Items = items
		keys := make([]string, 0, len(items))
		for _, item := range items {
			keys = append(keys, item.NativeItemKey)
		}
		input.PlanKeys = keys
		raw := []string{}
		for _, item := range items {
			if item.Disposition == "included" {
				raw = append(raw, item.NativeItemKey)
			}
		}
		input.RawKeys = raw
		_, err := BuildCaptureManifest(input)
		return err
	}
	single := func(item CaptureItemInput) []CaptureItemInput { return []CaptureItemInput{item} }
	cases := []struct {
		name     string
		items    []CaptureItemInput
		fragment string
	}{
		{"unknown class", single(CaptureItemInput{NativeItemKey: "a", Class: "mystery", Disposition: "excluded", ExclusionReason: &reason}), "nine-class"},
		{"included with reason", single(CaptureItemInput{NativeItemKey: "a", Class: "durable_payload", Disposition: "included", BlobDescriptorID: &descriptor, ByteCount: &count, ExclusionReason: &reason}), "carries an exclusion reason"},
		{"included missing descriptor", single(CaptureItemInput{NativeItemKey: "a", Class: "durable_payload", Disposition: "included", ByteCount: &count}), "non-null descriptor"},
		{"included missing count", single(CaptureItemInput{NativeItemKey: "a", Class: "durable_payload", Disposition: "included", BlobDescriptorID: &descriptor}), "non-null descriptor"},
		{"excluded with descriptor", single(CaptureItemInput{NativeItemKey: "a", Class: "credential", Disposition: "excluded", BlobDescriptorID: &descriptor, ExclusionReason: &reason}), "carries content"},
		{"excluded with count", single(CaptureItemInput{NativeItemKey: "a", Class: "credential", Disposition: "excluded", ByteCount: &count, ExclusionReason: &reason}), "carries content"},
		{"excluded missing reason", single(CaptureItemInput{NativeItemKey: "a", Class: "credential", Disposition: "excluded"}), "non-null exclusion reason"},
		{"credential included", single(CaptureItemInput{NativeItemKey: "a", Class: "credential", Disposition: "included", BlobDescriptorID: &descriptor, ByteCount: &count}), "always excluded"},
		{"machine_auth included", single(CaptureItemInput{NativeItemKey: "a", Class: "machine_auth", Disposition: "included", BlobDescriptorID: &descriptor, ByteCount: &count}), "always excluded"},
		{"runtime_state included", single(CaptureItemInput{NativeItemKey: "a", Class: "runtime_state", Disposition: "included", BlobDescriptorID: &descriptor, ByteCount: &count}), "always excluded"},
		{"transient_lock included", single(CaptureItemInput{NativeItemKey: "a", Class: "transient_lock", Disposition: "included", BlobDescriptorID: &descriptor, ByteCount: &count}), "always excluded"},
		{"bad disposition", single(CaptureItemInput{NativeItemKey: "a", Class: "unknown", Disposition: "maybe", ExclusionReason: &reason}), "included|excluded"},
		{"unsorted items", []CaptureItemInput{
			{NativeItemKey: "b", Class: "unknown", Disposition: "excluded", ExclusionReason: &reason},
			{NativeItemKey: "a", Class: "unknown", Disposition: "excluded", ExclusionReason: &reason},
		}, "sorted bytewise"},
		{"duplicate items", []CaptureItemInput{
			{NativeItemKey: "a", Class: "unknown", Disposition: "excluded", ExclusionReason: &reason},
			{NativeItemKey: "a", Class: "unknown", Disposition: "excluded", ExclusionReason: &reason},
		}, "sorted bytewise"},
		{"bad descriptor digest", single(CaptureItemInput{NativeItemKey: "a", Class: "durable_payload", Disposition: "included", BlobDescriptorID: strptr("nope"), ByteCount: &count}), "not a digest"},
		{"empty reason", single(CaptureItemInput{NativeItemKey: "a", Class: "credential", Disposition: "excluded", ExclusionReason: strptr("")}), "string[1..128]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireRefusal(t, build(tc.items), tc.fragment)
		})
	}
}

func TestCaptureManifestRefusals(t *testing.T) {
	t.Run("decode", func(t *testing.T) {
		valid := mustCaptureBytes(t)
		cases := []struct {
			name     string
			body     func(t *testing.T) []byte
			fragment string
		}{
			{"unknown member", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["lease"] = "x" })
			}, "unknown member"},
			{"bad schema", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["schema"] = "urn:ax:schema:blob" })
			}, "not the clone capture manifest"},
			{"bad version", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["schema_version"] = "2.0.0" })
			}, "not 1.0.0"},
			{"items not array", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["items"] = "nope" })
			}, "not an array"},
			{"too many items", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["items"] = make([]any, 65537) })
			}, "maximum is 65536"},
			{"excluded classes extra", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["excluded_classes"] = []any{"credential", "unknown"} })
			}, "exactly the excluded-row classes"},
			{"excluded classes missing", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["excluded_classes"] = []any{} })
			}, "exactly the excluded-row classes"},
			{"excluded classes unsorted", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[1].(map[string]any)["class"] = "credential"
					items[1].(map[string]any)["disposition"] = "excluded"
					items[1].(map[string]any)["blob_descriptor_id"] = nil
					items[1].(map[string]any)["byte_count"] = nil
					items[1].(map[string]any)["exclusion_reason"] = "operator_policy"
					o["excluded_classes"] = []any{"runtime_state", "credential"}
				})
			}, "sorted unique"},
			{"excluded classes unknown", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["excluded_classes"] = []any{"mystery"} })
			}, "unknown class"},
			{"raw_complete with unknown", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[0].(map[string]any)["class"] = "unknown"
					o["raw_complete"] = true
				})
			}, "unknown-class item"},
			{"item class mystery", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[0].(map[string]any)["class"] = "mystery"
				})
			}, "nine-class"},
			{"items duplicate key", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[1].(map[string]any)["native_item_key"] = items[0].(map[string]any)["native_item_key"]
				})
			}, "sorted bytewise"},
			{"items unsorted", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[0], items[2] = items[2], items[0]
				})
			}, "sorted bytewise"},
			{"excluded classes duplicate", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["excluded_classes"] = []any{"credential", "credential"} })
			}, "sorted unique"},
			{"item disposition maybe", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[0].(map[string]any)["disposition"] = "maybe"
				})
			}, "included|excluded"},
			{"included with reason", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[0].(map[string]any)["exclusion_reason"] = "r"
				})
			}, "carries an exclusion reason"},
			{"excluded with content", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					items[2].(map[string]any)["byte_count"] = float64(3)
				})
			}, "carries content"},
			{"credential included", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					items := o["items"].([]any)
					item := items[2].(map[string]any)
					item["disposition"] = "included"
					item["blob_descriptor_id"] = fixtureDigest("d")
					item["byte_count"] = float64(3)
					item["exclusion_reason"] = nil
					o["excluded_classes"] = []any{}
				})
			}, "always excluded"},
			{"raw_complete not bool", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["raw_complete"] = "yes" })
			}, "not a boolean"},
			{"source identity unsanitized", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					o["source_identity"].(map[string]any)["native_session_id"] = "/abs"
				})
			}, "absolute source path"},
			{"boundary proof unequal digests", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					o["capture_boundary"].(map[string]any)["proof"].(map[string]any)["post_capture_digest"] = fixtureDigest("zzz")
				})
			}, "differ"},
			{"bad bundle", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["bundle_id"] = "nope" })
			}, "bundle_id"},
			{"bad created_at", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["created_at"] = "yesterday" })
			}, "created_at"},
			{"self mismatch", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["capture_manifest_id"] = fixtureDigest("forged") })
			}, "omit-self digest"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeCaptureManifest(tc.body(t))
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
	t.Run("reconciliation", func(t *testing.T) {
		input := validCaptureInput(t)
		built, err := BuildCaptureManifest(input)
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := DecodeCaptureManifest(built)
		if err != nil {
			t.Fatal(err)
		}
		if err := VerifyCaptureReconciliation(manifest, []string{"store/blob-a"}, input.RawKeys); !errors.Is(err, ErrInvalid) {
			t.Fatalf("short plan error = %v, want ErrInvalid", err)
		}
		if err := VerifyCaptureReconciliation(manifest, input.PlanKeys, []string{"store/blob-a"}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("short raw error = %v, want ErrInvalid", err)
		}
		// Duplicate raw keys hiding a missing included object are
		// refused outright, not derived incomplete.
		if err := VerifyCaptureReconciliation(manifest, input.PlanKeys, []string{"store/blob-a", "store/blob-a"}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("duplicate raw keys error = %v, want ErrInvalid", err)
		} else if !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("duplicate raw keys error = %v, want the duplicate refusal", err)
		}
		if err := VerifyCaptureReconciliation(manifest, []string{"store/blob-a", "store/blob-b", "store/blob-b"}, input.RawKeys); !errors.Is(err, ErrInvalid) {
			t.Fatalf("duplicate plan keys error = %v, want ErrInvalid", err)
		}
		if deriveRawComplete(manifest.Items, input.PlanKeys, []string{"store/blob-a", "store/blob-a"}) {
			t.Fatal("deriveRawComplete(duplicate raw keys) = true, want false")
		}
		// A manifest whose flag disagrees with the derivation is refused
		// even when the key sets reconcile.
		flipped := manifest
		flipped.RawComplete = false
		if err := VerifyCaptureReconciliation(flipped, input.PlanKeys, input.RawKeys); !errors.Is(err, ErrInvalid) {
			t.Fatalf("flipped flag error = %v, want ErrInvalid", err)
		}
		incomplete := manifest
		incomplete.RawComplete = false
		if err := RefuseMaximalSafeUnlessComplete(incomplete); !errors.Is(err, ErrInvalid) {
			t.Fatalf("incomplete maximal_safe error = %v, want ErrInvalid", err)
		}
	})
	t.Run("build envelope", func(t *testing.T) {
		input := validCaptureInput(t)
		input.BundleID = "nope"
		if _, err := BuildCaptureManifest(input); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad bundle error = %v, want ErrInvalid", err)
		}
		input = validCaptureInput(t)
		input.CreatedAt = "yesterday"
		if _, err := BuildCaptureManifest(input); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad created_at error = %v, want ErrInvalid", err)
		}
		input = validCaptureInput(t)
		input.SourceBasis.Kind = "mystery"
		if _, err := BuildCaptureManifest(input); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad basis kind error = %v, want ErrInvalid", err)
		}
		input = validCaptureInput(t)
		input.Boundary.Kind = "mystery"
		if _, err := BuildCaptureManifest(input); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad boundary kind error = %v, want ErrInvalid", err)
		}
	})
	t.Run("build plan match", func(t *testing.T) {
		input := validCaptureInput(t)
		input.PlanKeys = append(input.PlanKeys, "store/zz-missing-item")
		_, err := BuildCaptureManifest(input)
		requireRefusal(t, err, "one per plan candidate")
		input = validCaptureInput(t)
		input.PlanKeys = input.PlanKeys[:2]
		_, err = BuildCaptureManifest(input)
		requireRefusal(t, err, "one per plan candidate")
		input = validCaptureInput(t)
		input.PlanKeys[2] = "store/zz-renamed"
		_, err = BuildCaptureManifest(input)
		requireRefusal(t, err, "has no item")
		// A repeated plan key hiding an item that is not a plan
		// candidate is refused outright, never sealed.
		input = validCaptureInput(t)
		input.PlanKeys = []string{"store/blob-a", "store/blob-a", "store/token-cache"}
		_, err = BuildCaptureManifest(input)
		requireRefusal(t, err, "plan keys carry duplicate")
	})
}

func TestBoundaryRefusals(t *testing.T) {
	t.Run("decode stable proof", func(t *testing.T) {
		proof := func(mutate func(map[string]any)) []byte {
			base := map[string]any{
				"proof_kind": "closed_store", "source_generation": "g",
				"snapshot_identity_digest": nil,
				"pre_capture_digest":       fixtureDigest("test-pre-capture"),
				"post_capture_digest":      fixtureDigest("test-pre-capture"),
				"input_blocked":            true, "foreground_idle": true, "background_idle": true,
				"extensions": map[string]any{},
			}
			mutate(base)
			out, err := json.Marshal(base)
			if err != nil {
				t.Fatal(err)
			}
			return out
		}
		cases := []struct {
			name     string
			body     []byte
			fragment string
		}{
			{"unequal digests", proof(func(o map[string]any) { o["post_capture_digest"] = fixtureDigest("other") }), "differ"},
			{"bad kind", proof(func(o map[string]any) { o["proof_kind"] = "file_size" }), "proof vocabulary"},
			{"closed_store with identity", proof(func(o map[string]any) { o["snapshot_identity_digest"] = fixtureDigest("x") }), "null snapshot identity"},
			{"closed_store idle false", proof(func(o map[string]any) { o["foreground_idle"] = false }), "requires input_blocked"},
			{"closed_store input_blocked false", proof(func(o map[string]any) { o["input_blocked"] = false }), "requires input_blocked"},
			{"snapshot null identity", proof(func(o map[string]any) {
				o["proof_kind"] = "immutable_snapshot"
			}), "non-null snapshot identity"},
			{"log prefix null identity", proof(func(o map[string]any) {
				o["proof_kind"] = "verified_log_prefix"
			}), "non-null snapshot identity"},
			{"quiescence idle false", proof(func(o map[string]any) {
				o["proof_kind"] = "provider_quiescence"
				o["background_idle"] = false
			}), "requires input_blocked"},
			{"empty generation", proof(func(o map[string]any) { o["source_generation"] = "" }), "string[1..512]"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeStableSnapshotProof(tc.body)
				requireRefusal(t, err, tc.fragment)
			})
		}
		// Non-null identity admits the snapshot kinds.
		admitted := proof(func(o map[string]any) {
			o["proof_kind"] = "immutable_snapshot"
			o["snapshot_identity_digest"] = fixtureDigest("snap")
			o["foreground_idle"] = false
		})
		if _, err := DecodeStableSnapshotProof(admitted); err != nil {
			t.Fatalf("DecodeStableSnapshotProof(snapshot) error = %v", err)
		}
	})
	t.Run("decode boundary", func(t *testing.T) {
		stableProof := `{"proof_kind":"closed_store","source_generation":"g","snapshot_identity_digest":null,` +
			`"pre_capture_digest":"` + fixtureDigest("test-pre-capture") + `","post_capture_digest":"` + fixtureDigest("test-pre-capture") +
			`","input_blocked":true,"foreground_idle":true,"background_idle":true,"extensions":{}}`
		cases := []struct {
			name     string
			body     string
			fragment string
		}{
			{"bad kind", `{"kind":"maybe","extensions":{}}`, "stable|unstable_archive"},
			{"stable missing proof", `{"kind":"stable","extensions":{}}`, "misses"},
			{"stable bad proof", `{"kind":"stable","proof":{},"extensions":{}}`, "misses"},
			{"unstable bad reason", `{"kind":"unstable_archive","source_generation":"g","pre_capture_digest":"` + fixtureDigest("a") + `","post_capture_digest":"` + fixtureDigest("b") + `","reason_code":"tired","operator_explicit":true,"target_projection_forbidden":true,"extensions":{}}`, "source_not_quiescent"},
			{"unstable operator false", `{"kind":"unstable_archive","source_generation":"g","pre_capture_digest":"` + fixtureDigest("a") + `","post_capture_digest":"` + fixtureDigest("b") + `","reason_code":"source_not_quiescent","operator_explicit":false,"target_projection_forbidden":true,"extensions":{}}`, "operator_explicit"},
			{"unstable projection false", `{"kind":"unstable_archive","source_generation":"g","pre_capture_digest":"` + fixtureDigest("a") + `","post_capture_digest":"` + fixtureDigest("b") + `","reason_code":"source_not_quiescent","operator_explicit":true,"target_projection_forbidden":false,"extensions":{}}`, "target_projection_forbidden"},
			{"unstable extra", `{"kind":"unstable_archive","source_generation":"g","pre_capture_digest":"` + fixtureDigest("a") + `","post_capture_digest":"` + fixtureDigest("b") + `","reason_code":"source_not_quiescent","operator_explicit":true,"target_projection_forbidden":true,"proof":{},"extensions":{}}`, "unknown member"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeCaptureBoundary([]byte(tc.body))
				requireRefusal(t, err, tc.fragment)
			})
		}
		if _, err := DecodeCaptureBoundary([]byte(`{"kind":"stable","proof":` + stableProof + `,"extensions":{}}`)); err != nil {
			t.Fatalf("DecodeCaptureBoundary(stable) error = %v", err)
		}
		if err := RefuseUnstableForTarget(CaptureBoundary{Kind: "mystery"}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("RefuseUnstableForTarget(mystery) error = %v, want ErrInvalid", err)
		}
	})
	t.Run("build core-only unstable", func(t *testing.T) {
		input := validCaptureInput(t)
		input.Boundary = BoundaryInput{
			Kind: "unstable_archive", Generation: "g",
			PreCaptureDigest: fixtureDigest("a"), PostCaptureDigest: fixtureDigest("b"),
		}
		_, err := BuildCaptureManifest(input)
		requireRefusal(t, err, "core-created only")
	})
}

func TestSourceBasisRefusals(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		fragment string
	}{
		{"bad kind", `{"kind":"mystery","extensions":{}}`, "ax_session|external_native"},
		{"missing kind", `{"extensions":{}}`, "misses"},
		{"ax missing members", `{"kind":"ax_session","source_session_id":"` + fixtureSessionID + `","extensions":{}}`, "misses"},
		{"ax bad session", `{"kind":"ax_session","source_session_id":"nope","source_session_record_id":"` + fixtureDigest("a") + `","source_checkpoint_id":"` + fixtureDigest("b") + `","source_provider_identity_record_id":"` + fixtureDigest("c") + `","extensions":{}}`, "not a UUIDv7"},
		{"ax bad record", `{"kind":"ax_session","source_session_id":"` + fixtureSessionID + `","source_session_record_id":"nope","source_checkpoint_id":"` + fixtureDigest("b") + `","source_provider_identity_record_id":"` + fixtureDigest("c") + `","extensions":{}}`, "not a digest"},
		{"ax extra", `{"kind":"ax_session","source_session_id":"` + fixtureSessionID + `","source_session_record_id":"` + fixtureDigest("a") + `","source_checkpoint_id":"` + fixtureDigest("b") + `","source_provider_identity_record_id":"` + fixtureDigest("c") + `","external_source_ref":"x","extensions":{}}`, "unknown member"},
		{"external missing ref", `{"kind":"external_native","extensions":{}}`, "misses"},
		{"external absolute ref", `{"kind":"external_native","external_source_ref":"/abs/path","extensions":{}}`, "absolute source path"},
		{"external uuid ref", `{"kind":"external_native","external_source_ref":"` + fixtureSessionID + `","extensions":{}}`, "fabricated AX"},
		{"external extra", `{"kind":"external_native","external_source_ref":"native-x","source_session_id":"` + fixtureSessionID + `","extensions":{}}`, "unknown member"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeSourceBasis([]byte(tc.body))
			requireRefusal(t, err, tc.fragment)
		})
	}
	admitted, err := DecodeSourceBasis([]byte(`{"kind":"external_native","external_source_ref":"native-x","extensions":{}}`))
	if err != nil {
		t.Fatalf("DecodeSourceBasis(external) error = %v", err)
	}
	if admitted.Kind != "external_native" || admitted.ExternalSourceRef == nil {
		t.Fatalf("external basis = %+v", admitted)
	}
	t.Run("build union", func(t *testing.T) {
		input := validCaptureInput(t)
		input.SourceBasis = SourceBasisInput{Kind: "ax_session", SourceSessionID: fixtureSessionID,
			SourceSessionRecordID: fixtureDigest("a"), SourceCheckpointID: fixtureDigest("b"),
			SourceProviderIdentityID: fixtureDigest("c"), ExternalSourceRef: "x"}
		_, err := BuildCaptureManifest(input)
		requireRefusal(t, err, "external source reference")
		input.SourceBasis = SourceBasisInput{Kind: "external_native", ExternalSourceRef: "native-x", SourceSessionID: fixtureSessionID}
		_, err = BuildCaptureManifest(input)
		requireRefusal(t, err, "ax_session members")
		input.SourceBasis = SourceBasisInput{Kind: "external_native", ExternalSourceRef: "native-x"}
		if _, err := BuildCaptureManifest(input); err != nil {
			t.Fatalf("BuildCaptureManifest(external basis) error = %v", err)
		}
	})
}
