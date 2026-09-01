package canonicaljson

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestObjectIdentityMatchesCrossPlatformGoldenRepresentations(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/cross-platform-identity.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Digest          string   `json:"digest"`
		Representations []string `json:"representations"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Representations) < 2 {
		t.Fatal("cross-platform golden requires at least two representations")
	}

	for index, representation := range fixture.Representations {
		calculated, field, err := CalculateObjectIdentity([]byte(representation))
		if err != nil || calculated.String() != fixture.Digest || field != SelfRecordID {
			t.Fatalf("representation %d identity = %q/%q, %v; want %q/%q", index, calculated, field, err, fixture.Digest, SelfRecordID)
		}
		claimed := strings.Replace(representation, zeroDigest, fixture.Digest, 1)
		verified, verifiedField, err := VerifyObjectIdentity([]byte(claimed))
		if err != nil || verified.String() != fixture.Digest || verifiedField != SelfRecordID {
			t.Fatalf("representation %d verification = %q/%q, %v", index, verified, verifiedField, err)
		}
	}
}
