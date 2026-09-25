package cloneplan_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/gowebpki/jcs"
	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// independentSelfDigest recomputes the omit-self identity without
// touching production: plain marshal, JCS transform, SHA-256. The
// canonical-bytes path is shared infrastructure (JCS itself); the
// omission choice and digest comparison are independent.
func independentSelfDigest(t *testing.T, sealed []byte, self string) string {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(sealed, &document); err != nil {
		t.Fatalf("unmarshal sealed: %v", err)
	}
	delete(document, self)
	plain, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal omitted: %v", err)
	}
	canonical, err := jcs.Transform(plain)
	if err != nil {
		t.Fatalf("JCS transform: %v", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// TestPlanIdentityIndependent pins that projection_plan_id is the
// JCS digest with only itself omitted: the independent recompute
// agrees, omitting a different member disagrees, and a tampered
// claim refuses.
func TestPlanIdentityIndependent(t *testing.T) {
	sealed := mustBuildPlan(t, validPlanInput())
	document := decodeDocument(t, sealed)
	claimed, ok := document["projection_plan_id"].(string)
	if !ok {
		t.Fatal("projection_plan_id is not a string")
	}
	if got := independentSelfDigest(t, sealed, "projection_plan_id"); got != claimed {
		t.Fatalf("independent digest = %s, sealed claim = %s", got, claimed)
	}
	if got := independentSelfDigest(t, sealed, "strategy"); got == claimed {
		t.Fatal("omitting strategy instead of the id agrees: the omission rule is not pinned")
	}
	document["projection_plan_id"] = fixtureDigest("tampered-claim")
	_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "projection_plan_id", "does not match omit-self digest")
	document = decodeDocument(t, sealed)
	document["projection_plan_id"] = "not-a-digest"
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "projection_plan_id", "is not a digest")
}

// TestManifestIdentityIndependent pins that
// projected_object_manifest_id is the JCS digest with only itself
// omitted.
func TestManifestIdentityIndependent(t *testing.T) {
	sealed := mustBuildManifest(t, validManifestInput())
	document := decodeDocument(t, sealed)
	claimed, ok := document["projected_object_manifest_id"].(string)
	if !ok {
		t.Fatal("projected_object_manifest_id is not a string")
	}
	if got := independentSelfDigest(t, sealed, "projected_object_manifest_id"); got != claimed {
		t.Fatalf("independent digest = %s, sealed claim = %s", got, claimed)
	}
	if got := independentSelfDigest(t, sealed, "total_bytes"); got == claimed {
		t.Fatal("omitting total_bytes instead of the id agrees: the omission rule is not pinned")
	}
	document["projected_object_manifest_id"] = fixtureDigest("tampered-manifest")
	_, err := cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, document))
	requireRefusal(t, err, "projected_object_manifest_id", "does not match omit-self digest")
}
