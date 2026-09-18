package clonebundle

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Negative tests for the Canonical Session, Canonical Event, identity,
// generation, and blob-agreement gates. Every row drives a production
// entry point.

// unsortedPair returns two distinct fixture digests in
// descending order: feeding them where ascending order is required
// always exercises the unsorted refusal.
func unsortedPair(seedA, seedB string) (string, string) {
	first, second := fixtureDigest(seedA), fixtureDigest(seedB)
	if first < second {
		first, second = second, first
	}
	return first, second
}

// manyDigests returns count distinct fixture digests: every scale
// far-edge row feeds values past the bound while keeping every
// other admission property valid, so a widened bound admits the
// input instead of refusing it one gate later.
func manyDigests(seed string, count int) []string {
	out := make([]string, count)
	for index := range out {
		out[index] = fixtureDigest(fmt.Sprintf("%s-%d", seed, index))
	}
	return out
}

// manyActors returns count valid actors (one main, the rest unique
// subagents parented on main) with synthesized UUIDv7 IDs: only the
// count is out of bound.
func manyActors(count int) []ActorInput {
	actors := make([]ActorInput, count)
	actors[0] = ActorInput{ActorID: fixtureActorMain, Kind: "main"}
	for index := 1; index < count; index++ {
		actors[index] = ActorInput{
			ActorID:       fmt.Sprintf("0198f4c8-8e50-7f66-8f70-%012x", index),
			Kind:          "subagent",
			ParentActorID: strptr(fixtureActorMain),
		}
	}
	return actors
}

func mustSessionBytes(t *testing.T) []byte {
	t.Helper()
	built, err := BuildCanonicalSession(validSessionInput())
	if err != nil {
		t.Fatal(err)
	}
	return built
}

func mustEventBytes(t *testing.T, status string) []byte {
	t.Helper()
	built, err := BuildCanonicalEvent(validEventInput(status))
	if err != nil {
		t.Fatal(err)
	}
	return built
}

func TestCanonicalSessionRefusals(t *testing.T) {
	t.Run("build actors", func(t *testing.T) {
		build := func(actors []ActorInput) error {
			input := validSessionInput()
			input.Actors = actors
			_, err := BuildCanonicalSession(input)
			return err
		}
		cases := []struct {
			name     string
			actors   []ActorInput
			fragment string
		}{
			{"no actors", nil, "[1..1024]"},
			{"too many actors", manyActors(1025), "carries 1025 actors, want [1..1024]"},
			{"no main", []ActorInput{{ActorID: fixtureActorSub, Kind: "subagent", ParentActorID: strptr(fixtureActorMain)}}, "exactly one"},
			{"two mains", []ActorInput{
				{ActorID: fixtureActorMain, Kind: "main"},
				{ActorID: fixtureActorSub, Kind: "main"},
			}, "exactly one"},
			{"main with parent", []ActorInput{{ActorID: fixtureActorMain, Kind: "main", ParentActorID: strptr(fixtureActorSub)}}, "carries a parent"},
			{"non-main null parent", []ActorInput{
				{ActorID: fixtureActorMain, Kind: "main"},
				{ActorID: fixtureActorSub, Kind: "external"},
			}, "requires a parent"},
			{"duplicate actor id", []ActorInput{
				{ActorID: fixtureActorMain, Kind: "main"},
				{ActorID: fixtureActorMain, Kind: "subagent", ParentActorID: strptr(fixtureActorMain)},
			}, "unique by actor ID"},
			{"bad kind", []ActorInput{{ActorID: fixtureActorMain, Kind: "ghost"}}, "main|subagent|external"},
			{"bad actor id", []ActorInput{{ActorID: "nope", Kind: "main"}}, "not a UUIDv7"},
			{"unsanitized source id", []ActorInput{{ActorID: fixtureActorMain, Kind: "main", SourceNativeID: strptr("/abs/x")}}, "absolute source path"},
			{"empty name", []ActorInput{{ActorID: fixtureActorMain, Kind: "main", Name: strptr("")}}, "string[1..512]"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				requireRefusal(t, build(tc.actors), tc.fragment)
			})
		}
	})
	t.Run("build envelope", func(t *testing.T) {
		cases := []struct {
			name     string
			mutate   func(*CanonicalSessionInput)
			fragment string
		}{
			{"bad logical", func(in *CanonicalSessionInput) { in.LogicalSessionID = "nope" }, "not a UUIDv7"},
			{"bad tuple", func(in *CanonicalSessionInput) { in.SourceEnvironment = []byte(`{}`) }, "Environment Tuple"},
			{"unsanitized native id", func(in *CanonicalSessionInput) { in.SourceNativeSessionID = fixtureSessionID }, "fabricated AX"},
			{"empty title", func(in *CanonicalSessionInput) { in.Title = strptr("") }, "string[1..4096]"},
			{"bad workspace", func(in *CanonicalSessionInput) { in.Workspace = []byte(`{}`) }, "workspace"},
			{"empty events", func(in *CanonicalSessionInput) { in.EventIDs = nil }, "[1..1000000]"},
			{"too many events", func(in *CanonicalSessionInput) {
				in.EventIDs = manyDigests("far-event", 1000001)
				in.HeadEventIDs = []string{in.EventIDs[0]}
			}, "carry 1000001 IDs, want [1..1000000]"},
			{"dup events", func(in *CanonicalSessionInput) { in.EventIDs = []string{fixtureDigest("x"), fixtureDigest("x")} }, "not unique"},
			{"bad event digest", func(in *CanonicalSessionInput) { in.EventIDs = []string{"nope"} }, "not a digest"},
			{"empty heads", func(in *CanonicalSessionInput) { in.HeadEventIDs = nil }, "[1..1024]"},
			{"too many heads", func(in *CanonicalSessionInput) {
				in.EventIDs = manyDigests("far-head", 1025)
				in.HeadEventIDs = slices.Clone(in.EventIDs)
				slices.Sort(in.HeadEventIDs)
			}, "carry 1025 IDs, want [1..1024]"},
			{"head outside events", func(in *CanonicalSessionInput) { in.HeadEventIDs = []string{fixtureDigest("elsewhere")} }, "outside event_ids"},
			{"heads unsorted", func(in *CanonicalSessionInput) {
				high, low := unsortedPair("unsorted-head-a", "unsorted-head-b")
				in.EventIDs = []string{low, high}
				in.HeadEventIDs = []string{high, low}
			}, "sorted unique"},
			{"bad created_at", func(in *CanonicalSessionInput) { in.CreatedAt = strptr("yesterday") }, "not a timestamp"},
			{"bad extensions", func(in *CanonicalSessionInput) { in.Extensions = map[string]any{"no-dots": 1} }, "reverse-DNS"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				input := validSessionInput()
				tc.mutate(&input)
				_, err := BuildCanonicalSession(input)
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
	t.Run("decode", func(t *testing.T) {
		valid := mustSessionBytes(t)
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
			}, "not the canonical session"},
			{"two mains", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					actors := o["actors"].([]any)
					actors[1].(map[string]any)["kind"] = "main"
					actors[1].(map[string]any)["parent_actor_id"] = nil
				})
			}, "exactly one"},
			{"dup actors", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					actors := o["actors"].([]any)
					actors[1].(map[string]any)["actor_id"] = actors[0].(map[string]any)["actor_id"]
				})
			}, "unique by actor ID"},
			{"dup events", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					events := o["event_ids"].([]any)
					o["event_ids"] = []any{events[0], events[0]}
				})
			}, "not unique"},
			{"too many events", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["event_ids"] = make([]any, 1000001) })
			}, "carry 1000001 IDs, want [1..1000000]"},
			{"heads unsorted", func(t *testing.T) []byte {
				high, low := unsortedPair("unsorted-head-a", "unsorted-head-b")
				return tamper(t, valid, func(o map[string]any) {
					o["event_ids"] = []any{low, high}
					o["head_event_ids"] = []any{high, low}
				})
			}, "sorted unique"},
			{"head outside", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["head_event_ids"] = []any{fixtureDigest("elsewhere")} })
			}, "outside event_ids"},
			{"main with parent", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					actors := o["actors"].([]any)
					actors[0].(map[string]any)["parent_actor_id"] = fixtureActorSub
				})
			}, "carries a parent"},
			{"subagent null parent", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					actors := o["actors"].([]any)
					actors[1].(map[string]any)["parent_actor_id"] = nil
				})
			}, "requires a parent"},
			{"external null parent", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					actors := o["actors"].([]any)
					actors[1].(map[string]any)["kind"] = "external"
					actors[1].(map[string]any)["parent_actor_id"] = nil
				})
			}, "requires a parent"},
			{"bad version", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["schema_version"] = "2.0.0" })
			}, "not 1.0.0"},
			{"actors not array", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["actors"] = "nope" })
			}, "not an array"},
			{"too many actors", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["actors"] = make([]any, 1025) })
			}, "want [1..1024]"},
			{"actor kind ghost", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					actors := o["actors"].([]any)
					actors[1].(map[string]any)["kind"] = "ghost"
				})
			}, "main|subagent|external"},
			{"heads duplicate", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					heads := o["head_event_ids"].([]any)
					o["head_event_ids"] = []any{heads[0], heads[0]}
				})
			}, "sorted unique"},
			{"too many heads", func(t *testing.T) []byte {
				ids := manyDigests("far-head-decode", 1025)
				events := make([]any, len(ids))
				for index, id := range ids {
					events[index] = id
				}
				sorted := slices.Clone(ids)
				slices.Sort(sorted)
				heads := make([]any, len(sorted))
				for index, id := range sorted {
					heads[index] = id
				}
				return tamper(t, valid, func(o map[string]any) {
					o["event_ids"] = events
					o["head_event_ids"] = heads
				})
			}, "head_event_ids are not sorted unique digest[1..1024]"},
			{"title empty", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["title"] = "" })
			}, "string[1..4096]"},
			{"self mismatch", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["canonical_session_id"] = fixtureDigest("forged") })
			}, "omit-self digest"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeCanonicalSession(tc.body(t))
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
}

func TestCanonicalEventRefusals(t *testing.T) {
	t.Run("build envelope", func(t *testing.T) {
		cases := []struct {
			name     string
			mutate   func(*CanonicalEventInput)
			fragment string
		}{
			{"bad kind", func(in *CanonicalEventInput) { in.Kind = "chat" }, "26-kind"},
			{"bad visibility", func(in *CanonicalEventInput) { in.Visibility = "secret" }, "public|projection|internal|opaque"},
			{"bad logical", func(in *CanonicalEventInput) { in.LogicalSessionID = "nope" }, "not a UUIDv7"},
			{"bad actor", func(in *CanonicalEventInput) { in.ActorID = "nope" }, "not a UUIDv7"},
			{"bad turn", func(in *CanonicalEventInput) { in.TurnID = strptr("nope") }, "turn_id"},
			{"bad timestamp", func(in *CanonicalEventInput) { in.Timestamp = strptr("yesterday") }, "not a timestamp"},
			{"parents unsorted", func(in *CanonicalEventInput) {
				high, low := unsortedPair("unsorted-parent-a", "unsorted-parent-b")
				in.Parents = []string{high, low}
			}, "sorted unique"},
			{"parents dup", func(in *CanonicalEventInput) {
				in.Parents = []string{fixtureDigest("a"), fixtureDigest("a")}
			}, "sorted unique"},
			{"too many parents", func(in *CanonicalEventInput) {
				in.Parents = manyDigests("far-parent", 65)
				slices.Sort(in.Parents)
			}, "carry 65 IDs, want [0..64]"},
			{"payload not object", func(in *CanonicalEventInput) { in.Payload = []byte(`[1]`) }, "payload"},
			{"message missing blocks", func(in *CanonicalEventInput) { in.Payload = []byte(`{"extensions":{}}`) }, "content_blocks"},
			{"message extra member", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[],"extensions":{},"transcript":"x"}`)
			}, "unknown member"},
			{"block bad type", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"video"}],"extensions":{}}`)
			}, "eight-type"},
			{"block missing type", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"content":"x"}],"extensions":{}}`)
			}, "misses its type"},
			{"block inline oversize", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"text","content":"` + strings.Repeat("x", 65537) + `"}],"extensions":{}}`)
			}, "64 KiB"},
			{"block bad descriptor", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"image","blob_descriptor_id":"nope"}],"extensions":{}}`)
			}, "not a digest"},
			{"block neither content nor descriptor", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"text"}],"extensions":{}}`)
			}, "not exclusive"},
			{"block both content and descriptor", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"text","content":"x","blob_descriptor_id":"` + fixtureDigest("d") + `"}],"extensions":{}}`)
			}, "not exclusive"},
			{"block extra member", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"text","blob_id":"x"}],"extensions":{}}`)
			}, "unknown member"},
			{"block null content", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"text","content":null}],"extensions":{}}`)
			}, "not exclusive"},
			{"block null descriptor", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"text","blob_descriptor_id":null}],"extensions":{}}`)
			}, "not exclusive"},
			{"payload top-level duplicate", func(in *CanonicalEventInput) {
				in.Payload = []byte(`{"content_blocks":[{"type":"text","content":"x"}],"extensions":{},"extensions":{}}`)
			}, "duplicate member"},
			{"ordinal overflow", func(in *CanonicalEventInput) { in.Ordinal = maxUint53 + 1 }, "exceeds uint53"},
			{"bad extensions", func(in *CanonicalEventInput) { in.Extensions = map[string]any{"no-dots": 1} }, "reverse-DNS"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				input := validEventInput("exact")
				tc.mutate(&input)
				_, err := BuildCanonicalEvent(input)
				requireRefusal(t, err, tc.fragment)
			})
		}
		// Inline content at exactly 64 KiB is admitted.
		input := validEventInput("exact")
		input.Payload = []byte(`{"content_blocks":[{"type":"text","content":"` + strings.Repeat("y", 65536) + `"}],"extensions":{}}`)
		if _, err := BuildCanonicalEvent(input); err != nil {
			t.Fatalf("BuildCanonicalEvent(64 KiB inline) error = %v", err)
		}
	})
	t.Run("decode", func(t *testing.T) {
		valid := mustEventBytes(t, "exact")
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
			}, "not the canonical event"},
			{"bad kind", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["kind"] = "chat" })
			}, "26-kind"},
			{"ordinal float", func(t *testing.T) []byte {
				raw := strings.Replace(string(valid), `"ordinal":0`, `"ordinal":1.5`, 1)
				return []byte(raw)
			}, "not a uint53"},
			{"visibility secret", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["visibility"] = "secret" })
			}, "public|projection|internal|opaque"},
			{"parents duplicate", func(t *testing.T) []byte {
				digest := fixtureDigest("test-event-1")
				return tamper(t, valid, func(o map[string]any) { o["parents"] = []any{digest, digest} })
			}, "sorted unique"},
			{"parents unsorted", func(t *testing.T) []byte {
				high, low := unsortedPair("unsorted-parent-a", "unsorted-parent-b")
				return tamper(t, valid, func(o map[string]any) { o["parents"] = []any{high, low} })
			}, "sorted unique"},
			{"too many parents", func(t *testing.T) []byte {
				ids := manyDigests("far-parent-decode", 65)
				slices.Sort(ids)
				parents := make([]any, len(ids))
				for index, id := range ids {
					parents[index] = id
				}
				return tamper(t, valid, func(o map[string]any) { o["parents"] = parents })
			}, "parents are not sorted unique digest[0..64]"},
			{"evidence status maybe", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					o["source_evidence"].(map[string]any)["capture_status"] = "maybe"
				})
			}, "exact|partial|synthesized|unavailable"},
			{"evidence raw_refs duplicate", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					evidence := o["source_evidence"].(map[string]any)
					refs := evidence["raw_refs"].([]any)
					evidence["raw_refs"] = []any{refs[0], refs[0]}
				})
			}, "sorted unique"},
			{"evidence raw_refs unsorted", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					evidence := o["source_evidence"].(map[string]any)
					refs := evidence["raw_refs"].([]any)
					first := map[string]any{}
					for key, value := range refs[0].(map[string]any) {
						first[key] = value
					}
					first["offset"] = float64(1)
					evidence["raw_refs"] = []any{first, refs[0]}
				})
			}, "raw_refs are not sorted unique"},
			{"evidence reason_codes duplicate", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					o["source_evidence"].(map[string]any)["reason_codes"] = []any{"a", "a"}
				})
			}, "sorted unique"},
			{"exact with core operation", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					o["source_evidence"].(map[string]any)["core_operation_id"] = fixtureOperationID
				})
			}, "carries a core operation"},
			{"bad version", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["schema_version"] = "2.0.0" })
			}, "not 1.0.0"},
			{"raw_refs not array", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					o["source_evidence"].(map[string]any)["raw_refs"] = "nope"
				})
			}, "not an array"},
			{"too many raw_refs", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					o["source_evidence"].(map[string]any)["raw_refs"] = make([]any, 65537)
				})
			}, "maximum is 65536"},
			{"too many reason codes", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) {
					codes := make([]any, 129)
					for i := range codes {
						codes[i] = fmt.Sprintf("code-%03d", i)
					}
					o["source_evidence"].(map[string]any)["reason_codes"] = codes
				})
			}, "reason_codes"},
			{"self mismatch", func(t *testing.T) []byte {
				return tamper(t, valid, func(o map[string]any) { o["event_id"] = fixtureDigest("forged") })
			}, "omit-self digest"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeCanonicalEvent(tc.body(t))
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
	t.Run("evidence", func(t *testing.T) {
		build := func(mutate func(*EvidenceInput)) error {
			input := validEventInput("exact")
			mutate(&input.Evidence)
			_, err := BuildCanonicalEvent(input)
			return err
		}
		cases := []struct {
			name     string
			mutate   func(*EvidenceInput)
			fragment string
		}{
			{"bad status", func(ev *EvidenceInput) { ev.CaptureStatus = "maybe" }, "exact|partial|synthesized|unavailable"},
			{"synthesized with native event", func(ev *EvidenceInput) {
				ev.CaptureStatus = "synthesized"
				ev.CoreOperationID = strptr(fixtureOperationID)
			}, "carries a native event ID"},
			{"synthesized missing operation", func(ev *EvidenceInput) {
				ev.CaptureStatus = "synthesized"
				ev.NativeEventID = nil
			}, "requires a core operation"},
			{"exact with operation", func(ev *EvidenceInput) { ev.CoreOperationID = strptr(fixtureOperationID) }, "carries a core operation"},
			{"bad tuple", func(ev *EvidenceInput) { ev.Environment = []byte(`{}`) }, "Environment Tuple"},
			{"unsanitized session", func(ev *EvidenceInput) { ev.NativeSessionID = "/abs" }, "absolute source path"},
			{"unsanitized event", func(ev *EvidenceInput) { ev.NativeEventID = strptr("https://u:p@h/x") }, "embedded credentials"},
			{"refs unsorted", func(ev *EvidenceInput) {
				high, low := unsortedPair("unsorted-ref-a", "unsorted-ref-b")
				ev.RawRefs = []RawRefInput{
					{ManifestID: high, BlobDescriptorID: fixtureDigest("d"), Offset: 0, Length: 1},
					{ManifestID: low, BlobDescriptorID: fixtureDigest("d"), Offset: 0, Length: 1},
				}
			}, "sorted unique"},
			{"refs dup", func(ev *EvidenceInput) {
				ev.RawRefs = []RawRefInput{
					{ManifestID: fixtureDigest("a"), BlobDescriptorID: fixtureDigest("d"), Offset: 0, Length: 1},
					{ManifestID: fixtureDigest("a"), BlobDescriptorID: fixtureDigest("d"), Offset: 0, Length: 1},
				}
			}, "sorted unique"},
			{"ref bad manifest", func(ev *EvidenceInput) { ev.RawRefs[0].ManifestID = "nope" }, "not a digest"},
			{"ref offset overflow", func(ev *EvidenceInput) { ev.RawRefs[0].Offset = maxUint53 + 1 }, "exceeds uint53"},
			{"ref length overflow", func(ev *EvidenceInput) { ev.RawRefs[0].Length = maxUint53 + 1 }, "exceeds uint53"},
			{"reasons unsorted", func(ev *EvidenceInput) { ev.ReasonCodes = []string{"b", "a"} }, "sorted unique"},
			{"reasons duplicate", func(ev *EvidenceInput) { ev.ReasonCodes = []string{"a", "a"} }, "sorted unique"},
			{"reason empty", func(ev *EvidenceInput) { ev.ReasonCodes = []string{""} }, "string[1..128]"},
			{"bad operation", func(ev *EvidenceInput) {
				ev.CaptureStatus = "synthesized"
				ev.NativeEventID = nil
				ev.CoreOperationID = strptr("nope")
			}, "not a UUIDv7"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				requireRefusal(t, build(tc.mutate), tc.fragment)
			})
		}
		// Decode-side evidence coupling.
		valid := mustEventBytes(t, "exact")
		tampered := tamper(t, valid, func(o map[string]any) {
			evidence := o["source_evidence"].(map[string]any)
			evidence["capture_status"] = "synthesized"
			evidence["core_operation_id"] = nil
			evidence["native_event_id"] = nil
		})
		if _, err := DecodeCanonicalEvent(tampered); !errors.Is(err, ErrInvalid) {
			t.Fatalf("synthesized-without-operation decode error = %v, want ErrInvalid", err)
		}
	})
}

func TestIdentityRefusals(t *testing.T) {
	t.Run("native identity", func(t *testing.T) {
		encode := func(mutate func(map[string]any)) []byte {
			identity := fixtureNativeIdentity()
			generation, err := EncodeNativeIdentity(identity, nil)
			if err != nil {
				t.Fatal(err)
			}
			return tamper(t, generation, mutate)
		}
		cases := []struct {
			name     string
			body     []byte
			fragment string
		}{
			{"bad kind", encode(func(o map[string]any) { o["identity_kind"] = "imported" }), "provider_native|official_import|continuation_context"},
			{"absolute id", encode(func(o map[string]any) { o["native_session_id"] = "/abs/x" }), "absolute source path"},
			{"uuid id", encode(func(o map[string]any) { o["native_session_id"] = fixtureSessionID }), "fabricated AX"},
			{"control id", encode(func(o map[string]any) { o["native_session_id"] = "a\x7fb" }), "control characters"},
			{"credential id", encode(func(o map[string]any) { o["native_session_id"] = "https://u:p@h/x" }), "embedded credentials"},
			{"ax marker id", encode(func(o map[string]any) { o["native_session_id"] = "urn:ax:evil" }), "AX identity marker"},
			{"bad workspace", encode(func(o map[string]any) { o["logical_workspace_id"] = "nope" }), "not a UUIDv7"},
			{"bad fingerprint", encode(func(o map[string]any) { o["backend_realm_fingerprint"] = "nope" }), "not a digest"},
			{"unsanitized opaque", encode(func(o map[string]any) { o["opaque_identity"] = "/abs/opaque" }), "absolute source path"},
			{"unknown member", encode(func(o map[string]any) { o["token"] = "x" }), "unknown member"},
			{"missing member", encode(func(o map[string]any) { delete(o, "identity_kind") }), "misses"},
			{"bad extensions", encode(func(o map[string]any) { o["extensions"] = map[string]any{"x": 1} }), "reverse-DNS"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeNativeIdentity(tc.body)
				requireRefusal(t, err, tc.fragment)
			})
		}
		// Non-null fingerprint and opaque identity admit.
		admitted := encode(func(o map[string]any) {
			o["backend_realm_fingerprint"] = fixtureDigest("realm")
			o["opaque_identity"] = "opaque-native-1"
		})
		if _, err := DecodeNativeIdentity(admitted); err != nil {
			t.Fatalf("DecodeNativeIdentity(full) error = %v", err)
		}
	})
	t.Run("workspace binding", func(t *testing.T) {
		encode := func(mutate func(map[string]any)) []byte {
			return tamper(t, []byte(fixtureWorkspaceJSON()), mutate)
		}
		cases := []struct {
			name     string
			body     []byte
			fragment string
		}{
			{"absolute cwd", encode(func(o map[string]any) { o["cwd_relative"] = "/abs/x" }), "beneath the workspace root"},
			{"escape cwd", encode(func(o map[string]any) { o["cwd_relative"] = "../escape" }), "beneath the workspace root"},
			{"dot cwd", encode(func(o map[string]any) { o["cwd_relative"] = "a/./b" }), "beneath the workspace root"},
			{"backslash cwd", encode(func(o map[string]any) { o["cwd_relative"] = `a\b` }), "beneath the workspace root"},
			{"empty cwd", encode(func(o map[string]any) { o["cwd_relative"] = "" }), "string[1..4096]"},
			{"bad workspace id", encode(func(o map[string]any) { o["logical_workspace_id"] = "nope" }), "not a UUIDv7"},
			{"prints unsorted", func() []byte {
				high, low := unsortedPair("unsorted-print-a", "unsorted-print-b")
				return encode(func(o map[string]any) {
					o["repository_remote_fingerprints"] = []any{high, low}
				})
			}(), "sorted unique"},
			{"prints duplicate", encode(func(o map[string]any) {
				print := fixtureDigest("test-remote")
				o["repository_remote_fingerprints"] = []any{print, print}
			}), "sorted unique"},
			{"print bad digest", encode(func(o map[string]any) {
				o["repository_remote_fingerprints"] = []any{"nope"}
			}), "sorted unique"},
			{"empty branch", encode(func(o map[string]any) { o["branch"] = "" }), "string[1..1024]"},
			{"bad head", encode(func(o map[string]any) { o["head_digest"] = "nope" }), "not a digest"},
			{"unknown member", encode(func(o map[string]any) { o["root"] = "/" }), "unknown member"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := DecodeWorkspaceBinding(tc.body)
				requireRefusal(t, err, tc.fragment)
			})
		}
		// The workspace root itself admits.
		if _, err := DecodeWorkspaceBinding(encode(func(o map[string]any) { o["cwd_relative"] = "." })); err != nil {
			t.Fatalf("DecodeWorkspaceBinding(root) error = %v", err)
		}
	})
	t.Run("sanitizer", func(t *testing.T) {
		for _, key := range []string{"", "a\x00b", "/abs", `\\host\share`, "C:\\win", "C:/win",
			fixtureSessionID, "urn:ax:schema:blob", "https://u:p@h/x", "a\x1fb",
			"a\x7fb", "a\u0085b", "a\u2028b", "a\u200bb", "a\u2029b", "a\xffb", "URN:AX:session", "user:secret@example.com/path"} {
			if err := SanitizeNativeKey(key); !errors.Is(err, ErrInvalid) {
				t.Fatalf("SanitizeNativeKey(%q) error = %v, want ErrInvalid", key, err)
			}
		}
		for _, key := range []string{"native-1", "store/blob-a", "tmux:0", "a@b", "https://host/x", "user@host"} {
			if err := SanitizeNativeKey(key); err != nil {
				t.Fatalf("SanitizeNativeKey(%q) error = %v", key, err)
			}
		}
	})
}

func TestGenerationRefusals(t *testing.T) {
	if _, err := ParseGeneration(""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ParseGeneration(empty) error = %v, want ErrInvalid", err)
	}
	if _, err := ParseGeneration(strings.Repeat("g", 513)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ParseGeneration(513) error = %v, want ErrInvalid", err)
	}
	first, err := ParseGeneration("generation-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseGeneration("generation-1")
	if err != nil {
		t.Fatal(err)
	}
	third, err := ParseGeneration("generation-2")
	if err != nil {
		t.Fatal(err)
	}
	if !first.Equal(second) || first.Equal(third) {
		t.Fatal("Generation equality is wrong")
	}
	if err := CheckDigestsEqual(mustDigest(t, fixtureDigest("a")), mustDigest(t, fixtureDigest("a"))); err != nil {
		t.Fatalf("CheckDigestsEqual(equal) error = %v", err)
	}
	if err := CheckDigestsEqual(mustDigest(t, fixtureDigest("a")), mustDigest(t, fixtureDigest("b"))); !errors.Is(err, ErrInvalid) {
		t.Fatalf("CheckDigestsEqual(differ) error = %v, want ErrInvalid", err)
	}
}

// TestVerifyRawManifestDescriptorsRefusals pins the three refusal
// arms of the evidence-verification entry (fetch failure, absent
// bytes, disagreeing bytes) plus the descriptor-identity link: a
// fetch that returns the real bytes for an entry naming a different
// descriptor is refused.
func TestVerifyRawManifestDescriptorsRefusals(t *testing.T) {
	manifest, err := DecodeRawObjectManifest(mustRawBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	inputs := validEntryInputs(t)
	byID := map[string][]byte{}
	for index, input := range inputs {
		byID[manifest.Entries[index].BlobDescriptorID.String()] = input.Descriptor
	}
	requireRefusal(t, VerifyRawManifestDescriptors(manifest, func(id string) ([]byte, error) {
		return nil, fmt.Errorf("boom")
	}), "descriptor fetch failed")
	requireRefusal(t, VerifyRawManifestDescriptors(manifest, func(id string) ([]byte, error) {
		return nil, nil
	}), "descriptor is absent")
	// Wrong bytes for entry 0 only: entry 1 verifies, so the entry-0
	// arm is the one that fires.
	wrong, _, _, _ := makeDescriptor(t, []byte("unrelated"))
	requireRefusal(t, VerifyRawManifestDescriptors(manifest, func(id string) ([]byte, error) {
		if id == manifest.Entries[0].BlobDescriptorID.String() {
			return wrong, nil
		}
		return byID[id], nil
	}), "disagrees")
	// The fetched bytes are real, but the entry names a descriptor
	// that does not exist: the identity link refuses.
	renamed := manifest
	renamed.Entries = append([]RawObjectEntry(nil), manifest.Entries...)
	renamed.Entries[0].BlobDescriptorID = mustDigest(t, fixtureDigest("some-other-descriptor"))
	requireRefusal(t, VerifyRawManifestDescriptors(renamed, func(id string) ([]byte, error) {
		return byID[manifest.Entries[0].BlobDescriptorID.String()], nil
	}), "disagrees with entry claim")
}

// TestInstallRawBlobRefusals pins the install-site gates: a forged
// identity claim is refused before any byte crosses the store, and
// byte streams that disagree with the descriptor size are refused
// by the landed no-replace discipline.
func TestInstallRawBlobRefusals(t *testing.T) {
	payload := []byte("twelve-bytes")
	descriptor, _, _, _ := makeDescriptor(t, payload)
	store := openFixtureStore(t)
	resealed := tamper(t, descriptor, func(o map[string]any) { o["descriptor_id"] = fixtureDigest("forged-descriptor") })
	_, err := InstallRawBlob(store, resealed, bytes.NewReader(payload))
	requireRefusal(t, err, "omit-self digest")
	_, err = InstallRawBlob(store, descriptor, bytes.NewReader([]byte("short")))
	requireRefusal(t, err, "blob install refused")
	_, err = InstallRawBlob(store, descriptor, bytes.NewReader([]byte("twelve-bytes-and-more")))
	requireRefusal(t, err, "blob install refused")
}

// TestIdentityDigestRefusals pins the digest gates: no digest is
// sealed over an identity decoding would refuse (re-decode for the
// kind, encode-time sanitize for the native keys), and unsafe
// extensions are refused at encode.
func TestIdentityDigestRefusals(t *testing.T) {
	bad := fixtureNativeIdentity()
	bad.IdentityKind = "bogus"
	_, err := IdentityDigest(bad, nil)
	requireRefusal(t, err, "undecodable identity")
	_, err = IdentityDigest(fixtureNativeIdentity(), map[string]any{"no-dots": 1})
	requireRefusal(t, err, "reverse-DNS")
	opaque := fixtureNativeIdentity()
	opaque.OpaqueIdentity = strptr("/abs/opaque")
	_, err = IdentityDigest(opaque, nil)
	requireRefusal(t, err, "absolute source path")
	unsafe := fixtureNativeIdentity()
	unsafe.NativeSessionID = "/abs/session"
	_, err = IdentityDigest(unsafe, nil)
	requireRefusal(t, err, "absolute source path")
}

// TestEncodeNativeIdentityRefusals pins the encode-time gates:
// encoding never admits what decoding refused, so unsafe extension
// values and non-reverse-DNS keys are refused here, not merely at
// the caller's re-decode — and unsanitized native keys are refused
// before marshaling, never rewritten to U+FFFD and sealed.
func TestEncodeNativeIdentityRefusals(t *testing.T) {
	_, err := EncodeNativeIdentity(fixtureNativeIdentity(), map[string]any{"com.example.n": uint64(1) << 60})
	requireRefusal(t, err, "safe-integer")
	_, err = EncodeNativeIdentity(fixtureNativeIdentity(), map[string]any{"no-dots": 1})
	requireRefusal(t, err, "reverse-DNS")
	unsanitized := fixtureNativeIdentity()
	unsanitized.NativeSessionID = "native\xff"
	_, err = EncodeNativeIdentity(unsanitized, nil)
	requireRefusal(t, err, "not valid UTF-8")
	badKind := fixtureNativeIdentity()
	badKind.IdentityKind = "provider\xff"
	sealed, err := EncodeNativeIdentity(badKind, nil)
	requireRefusal(t, err, "not valid UTF-8")
	if sealed != nil {
		t.Fatalf("EncodeNativeIdentity(bad kind) returned %d bytes, want none", len(sealed))
	}
	encoded, err := EncodeNativeIdentity(fixtureNativeIdentity(), nil)
	if err != nil {
		t.Fatalf("EncodeNativeIdentity(valid) error = %v", err)
	}
	if _, err := DecodeNativeIdentity(encoded); err != nil {
		t.Fatalf("DecodeNativeIdentity(EncodeNativeIdentity(valid)) error = %v", err)
	}
}

func TestBlobAgreementRefusals(t *testing.T) {
	descriptor, blobID, descriptorID, size := makeDescriptor(t, []byte("evidence"))
	blob := mustDigest(t, blobID)
	identity := mustDigest(t, descriptorID)
	if err := VerifyDescriptorAgreement(descriptor, identity, blob, size); err != nil {
		t.Fatalf("VerifyDescriptorAgreement() error = %v", err)
	}
	if err := VerifyDescriptorAgreement(descriptor, mustDigest(t, fixtureDigest("some-other-descriptor")), blob, size); !errors.Is(err, ErrInvalid) {
		t.Fatalf("descriptor identity disagreement error = %v, want ErrInvalid", err)
	} else if !strings.Contains(err.Error(), "disagrees with entry claim") {
		t.Fatalf("descriptor identity disagreement error = %v, want the identity link to fire", err)
	}
	if err := VerifyDescriptorAgreement(descriptor, identity, mustDigest(t, fixtureDigest("other")), size); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blob disagreement error = %v, want ErrInvalid", err)
	}
	if err := VerifyDescriptorAgreement(descriptor, identity, blob, size+1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("size disagreement error = %v, want ErrInvalid", err)
	}
	tampered := append([]byte(nil), descriptor...)
	tampered[len(tampered)-10] ^= 0xff
	if err := VerifyDescriptorAgreement(tampered, identity, blob, size); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tampered descriptor error = %v, want ErrInvalid", err)
	}
	resealed := tamper(t, descriptor, func(o map[string]any) { o["descriptor_id"] = fixtureDigest("forged-descriptor") })
	if err := VerifyDescriptorAgreement(resealed, identity, blob, size); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong-claim descriptor error = %v, want ErrInvalid", err)
	} else if !strings.Contains(err.Error(), "omit-self digest") {
		t.Fatalf("wrong-claim descriptor error = %v, want the claim comparison to fire", err)
	}
	if err := VerifyDescriptorAgreement([]byte(`{}`), identity, blob, size); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty object error = %v, want ErrInvalid", err)
	}
	store := openFixtureStore(t)
	if _, err := InstallRawBlob(nil, descriptor, strings.NewReader("evidence")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("InstallRawBlob(nil store) error = %v, want ErrInvalid", err)
	}
	if _, err := InstallRawBlob(store, []byte(`{}`), strings.NewReader("evidence")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("InstallRawBlob(bad descriptor) error = %v, want ErrInvalid", err)
	}
}
