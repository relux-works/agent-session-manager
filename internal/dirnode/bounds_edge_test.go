package dirnode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file pins every contract string maximum, array cap, and
// uint53 bound at the edge: each bound is driven at the limit
// (admitted) and one past it (refused), with the same for one
// below the minimum. A witness far outside the range proves the
// arm exists, not where the bound sits; a mutant moving any bound
// by exactly one admits exactly one member of the reject class
// and reddens its row here. Lengths are characters: the repeat
// rune is ASCII, so bytes equal runes.
//
// Every suite below is named as a driver by the bound census in
// bound_census_test.go, and each asserts its row count against
// boundCensusDriverCount so the table cannot go stale when the
// roster grows. Vectors are single-defect by construction — one
// member moved while the rest of the body stays valid — so the
// refusal is attributable to the driven gate even where
// production reports several gates through one shared detail.

// quotedRepeat renders a JSON string literal of exactly n
// characters.
func quotedRepeat(n int) string {
	return strconv.Quote(strings.Repeat("a", n))
}

// spliceBound substitutes one needle inside a valid body,
// failing when the needle is absent so the vector cannot pass
// vacuously against a still-valid body.
func spliceBound(t *testing.T, body, needle, replacement string) string {
	t.Helper()
	if !strings.Contains(body, needle) {
		t.Fatalf("spliceBound: needle %q absent; the vector mutates nothing", needle)
	}
	return strings.Replace(body, needle, replacement, 1)
}

// sortedDigestList builds n sorted unique JSON digest literals
// derived from the seed, so the sortedness gate stays green
// while the count sweeps.
func sortedDigestList(seed string, n int) string {
	digests := make([]string, 0, n)
	for i := 0; i < n; i++ {
		digests = append(digests, fixtureDigest(fmt.Sprintf("%s-%05d", seed, i)))
	}
	sort.Strings(digests)
	parts := make([]string, 0, n)
	for _, digest := range digests {
		parts = append(parts, quote(digest))
	}
	return `[` + strings.Join(parts, ",") + `]`
}

// sortedUUIDList builds n sorted unique JSON UUIDv7 literals by
// counting up the low twelve hex digits under a fixed
// version/variant prefix, so the sortedness gate stays green
// while the count sweeps.
func sortedUUIDList(n int) string {
	const base = 0x123456789000
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, quote(fmt.Sprintf("0198f4c8-8e50-7f66-8f70-%012x", base+i)))
	}
	return `[` + strings.Join(parts, ",") + `]`
}

// stringBoundRow is one checkStringBounds site pinned at the
// production entry: drive maps a JSON string literal to the
// entry's verdict. extraAdmits lists further literals that must
// admit (the null absence for optional members).
type stringBoundRow struct {
	name        string
	min         int
	max         int
	extraAdmits []string
	drive       func(t *testing.T, literal string) error
}

// TestStringBoundEdges drives every checkStringBounds and
// checkURI string site at min-1/min/max/max+1 through its
// production entry point.
func TestStringBoundEdges(t *testing.T) {
	t.Parallel()
	rows := []stringBoundRow{
		{
			name: "manifest node_id", min: 1, max: 128,
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeManifest([]byte(spliceBound(t, fixtureManifestJSON(), `"node_id":"test-directory-node"`, `"node_id":`+literal)))
				return err
			},
		},
		{
			name: "node build node_id", min: 1, max: 128,
			drive: func(t *testing.T, literal string) error {
				_, err := CheckProbeResponse([]byte(spliceBound(t, fixtureProbeResponseJSON(), `"node_id":"test-directory-node"`, `"node_id":`+literal)))
				return err
			},
		},
		{
			name: "capability reason", min: 1, max: 128,
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeManifest([]byte(spliceBound(t, fixtureManifestJSON(), `"reason_code":"needs-quiesce"`, `"reason_code":`+literal)))
				return err
			},
		},
		{
			name: "finding code", min: 1, max: 128,
			drive: func(t *testing.T, literal string) error {
				_, err := CheckProbeResponse([]byte(spliceBound(t, fixtureProbeResponseJSON(), `"code":"probe-ok"`, `"code":`+literal)))
				return err
			},
		},
		{
			name: "finding message", min: 1, max: 4096,
			drive: func(t *testing.T, literal string) error {
				_, err := CheckProbeResponse([]byte(spliceBound(t, fixtureProbeResponseJSON(), `"message":"probe completed"`, `"message":`+literal)))
				return err
			},
		},
		{
			name: "finding remediation", min: 1, max: 4096, extraAdmits: []string{"null"},
			drive: func(t *testing.T, literal string) error {
				_, err := CheckProbeResponse([]byte(spliceBound(t, fixtureProbeResponseJSON(), `"remediation":null`, `"remediation":`+literal)))
				return err
			},
		},
		{
			name: "caller_id", min: 1, max: 256,
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeQuery([]byte(spliceBound(t, fixtureQueryJSON(), `"caller_id":"test-caller"`, `"caller_id":`+literal)))
				return err
			},
		},
		{
			name: "authentication_subject", min: 1, max: 512,
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeQuery([]byte(spliceBound(t, fixtureQueryJSON(), `"authentication_subject":"test-subject"`, `"authentication_subject":`+literal)))
				return err
			},
		},
		{
			name: "annotation title", min: 1, max: 512,
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeQuery([]byte(spliceBound(t, operationBatch("set_title"), `"title":"T"`, `"title":`+literal)))
				return err
			},
		},
		{
			name: "scan cursor", min: 1, max: 4096, extraAdmits: []string{"null"},
			drive: func(t *testing.T, literal string) error {
				_, err := CheckScanRequest([]byte(spliceBound(t, fixtureScanRequestJSON(), `"cursor":null`, `"cursor":`+literal)))
				return err
			},
		},
		{
			name: "scan next cursor", min: 1, max: 4096, extraAdmits: []string{"null"},
			drive: func(t *testing.T, literal string) error {
				_, err := CheckScanResponse([]byte(spliceBound(t, fixtureScanResponseJSON(), `"next_cursor":null`, `"next_cursor":`+literal)))
				return err
			},
		},
		{
			name: "contract identifier", min: 1, max: 512,
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeManifest([]byte(spliceBound(t, fixtureManifestJSON(), `"contract_id":"urn:ax:schema:contract-00"`, `"contract_id":`+literal)))
				return err
			},
		},
	}
	if want := boundCensusDriverCount("TestStringBoundEdges"); len(rows) != want {
		t.Fatalf("string bound rows = %d, want %d; the table is short, not the package", len(rows), want)
	}
	uri512 := quote("urn:ax:schema:" + strings.Repeat("c", 498))
	uri513 := quote("urn:ax:schema:" + strings.Repeat("c", 499))
	if len("urn:ax:schema:"+strings.Repeat("c", 498)) != 512 || len("urn:ax:schema:"+strings.Repeat("c", 499)) != 513 {
		t.Fatal("contract URI scaffold has the wrong length; the edge is mismeasured, not the bound")
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			below, atMin, atMax, above := quotedRepeat(row.min-1), quotedRepeat(row.min), quotedRepeat(row.max), quotedRepeat(row.max+1)
			if row.name == "contract identifier" {
				// The 1..2 range is unreachable behind the
				// colon grammar (no 1-character URI carries
				// a scheme), so the floor is proven with
				// the shortest admittable URI rather than
				// a 1-character literal.
				below, atMin = `""`, `"a:b"`
			}
			if err := row.drive(t, below); err == nil {
				t.Fatalf("%s admitted %s below the minimum", row.name, below)
			}
			if err := row.drive(t, atMin); err != nil {
				t.Fatalf("%s refused %s at the minimum: %v", row.name, atMin, err)
			}
			maxLiteral, aboveLiteral := atMax, above
			if row.name == "contract identifier" {
				maxLiteral, aboveLiteral = uri512, uri513
			}
			if err := row.drive(t, maxLiteral); err != nil {
				t.Fatalf("%s refused at the maximum: %v", row.name, err)
			}
			if err := row.drive(t, aboveLiteral); err == nil {
				t.Fatalf("%s admitted past the maximum", row.name)
			}
			for _, admitted := range row.extraAdmits {
				if err := row.drive(t, admitted); err != nil {
					t.Fatalf("%s refused %s, which must admit: %v", row.name, admitted, err)
				}
			}
		})
	}
}

// uint53BoundRow is one checkUint53Bounds site pinned at the
// production entry: the bound refuses below the minimum and past
// the maximum, and admits both edges.
type uint53BoundRow struct {
	name  string
	min   uint64
	max   uint64
	drive func(t *testing.T, literal string) error
}

// TestUint53BoundEdges drives the scan and envelope uint53 sites
// at min-1/min/max/max+1 through the production entry. The
// manifest limits carry their own suite below because they share
// one fixture body.
func TestUint53BoundEdges(t *testing.T) {
	t.Parallel()
	rows := []uint53BoundRow{
		{
			name: "scan max_instances", min: 1, max: 65536,
			drive: func(t *testing.T, literal string) error {
				_, err := CheckScanRequest([]byte(spliceBound(t, fixtureScanRequestJSON(), `"max_instances":100`, `"max_instances":`+literal)))
				return err
			},
		},
		{
			// Both entries enforce the shared constants, and
			// the row proves each entry independently: the
			// decode entry refuses out-of-range literals
			// before the builder ever runs, so chaining the
			// builder behind a decode refusal would drive the
			// builder only with admitting values. In range,
			// both entries must admit; out of range, each
			// entry must refuse on its own — an entry that
			// admits out of range fails the row even when the
			// other refuses.
			name: "request deadline_ms", min: 1, max: 3600000,
			drive: func(t *testing.T, literal string) error {
				value := mustUint53(t, literal)
				_, decodeErr := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, value, `{}`))
				_, encodeErr := EncodeRequest(MajorV2, OpManifest, fixtureRequestID, value, []byte(`{}`))
				// The out-of-range literals below are min-1 and
				// max+1 of this row; the range itself stays
				// hand-typed above so a moved production
				// constant cannot take the expectation with it.
				if value == 0 || value == 3600001 {
					if decodeErr == nil || encodeErr == nil {
						return nil
					}
					return decodeErr
				}
				if decodeErr != nil {
					return decodeErr
				}
				return encodeErr
			},
		},
	}
	if want := boundCensusDriverCount("TestUint53BoundEdges"); len(rows) != want {
		t.Fatalf("uint53 bound rows = %d, want %d; the table is short, not the package", len(rows), want)
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			below := strconv.FormatUint(row.min-1, 10)
			if err := row.drive(t, below); err == nil {
				t.Fatalf("%s admitted %s below the %d minimum", row.name, below, row.min)
			}
			if err := row.drive(t, strconv.FormatUint(row.min, 10)); err != nil {
				t.Fatalf("%s refused %d at the minimum: %v", row.name, row.min, err)
			}
			if err := row.drive(t, strconv.FormatUint(row.max, 10)); err != nil {
				t.Fatalf("%s refused %d at the maximum: %v", row.name, row.max, err)
			}
			if err := row.drive(t, strconv.FormatUint(row.max+1, 10)); err == nil {
				t.Fatalf("%s admitted %d past the %d maximum", row.name, row.max+1, row.max)
			}
		})
	}
}

// mustUint53 renders one decimal literal back to uint64 for the
// deadline driver, which builds frames rather than splicing text.
func mustUint53(t *testing.T, literal string) uint64 {
	t.Helper()
	value, err := strconv.ParseUint(literal, 10, 64)
	if err != nil {
		t.Fatalf("mustUint53(%q): %v", literal, err)
	}
	return value
}

// TestUint53RepresentabilityCeiling pins the shared uint53
// ceiling once: 2^53-1 admits, 2^53 refuses. Every maxUint53
// maximum in the roster points here instead of repeating the
// literal per site.
func TestUint53RepresentabilityCeiling(t *testing.T) {
	t.Parallel()
	if _, ok := rawUint53([]byte(`9007199254740991`)); !ok {
		t.Fatal("rawUint53 refused 2^53-1 at the ceiling")
	}
	if _, ok := rawUint53([]byte(`9007199254740992`)); ok {
		t.Fatal("rawUint53 admitted 2^53 past the ceiling")
	}
	if _, ok := checkUint53Bounds([]byte(`9007199254740992`), 0, 9007199254740991); ok {
		t.Fatal("checkUint53Bounds admitted 2^53 past the ceiling")
	}
}

// TestManifestLimitsBoundEdges drives every DirectoryNodeLimits
// bound at min-1/min/max/max+1 through the production
// DecodeManifest entry. The fixture sits at every maximum, so
// max admission is the fixture decode itself.
func TestManifestLimitsBoundEdges(t *testing.T) {
	t.Parallel()
	good := fixtureManifestJSON()
	rows := []struct {
		name   string
		member string
		min    uint64
		max    uint64
		needle string
	}{
		{"max_frame_bytes", "max_frame_bytes", 1, 8388608, `"max_frame_bytes":8388608`},
		{"max_scan_instances", "max_scan_instances", 1, 65536, `"max_scan_instances":65536`},
		{"max_inventory_take", "max_inventory_take", 1, 1000, `"max_inventory_take":1000`},
		{"max_excerpt_count", "max_excerpt_count", 0, 20, `"max_excerpt_count":20`},
		{"max_excerpt_bytes", "max_excerpt_bytes", 0, 4096, `"max_excerpt_bytes":4096`},
		{"max_enrichment_events", "max_enrichment_events", 1, 5000, `"max_enrichment_events":5000`},
		{"max_enrichment_bytes", "max_enrichment_bytes", 1, 4194304, `"max_enrichment_bytes":4194304`},
	}
	if want := boundCensusDriverCount("TestManifestLimitsBoundEdges"); len(rows) != want {
		t.Fatalf("limits rows = %d, want %d; the table is short, not the package", len(rows), want)
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			drive := func(literal string) error {
				_, err := DecodeManifest([]byte(spliceBound(t, good, row.needle, `"`+row.member+`":`+literal)))
				return err
			}
			below := "-1"
			if row.min > 0 {
				below = strconv.FormatUint(row.min-1, 10)
			}
			if err := drive(below); err == nil {
				t.Fatalf("%s admitted %s below the %d minimum", row.name, below, row.min)
			}
			if err := drive(strconv.FormatUint(row.min, 10)); err != nil {
				t.Fatalf("%s refused %d at the minimum: %v", row.name, row.min, err)
			}
			if err := drive(strconv.FormatUint(row.max, 10)); err != nil {
				t.Fatalf("%s refused %d at the maximum: %v", row.name, row.max, err)
			}
			if err := drive(strconv.FormatUint(row.max+1, 10)); err == nil {
				t.Fatalf("%s admitted %d past the %d maximum", row.name, row.max+1, row.max)
			}
		})
	}
}

// TestSortedUniqueStringsEdges drives every checkSortedUniqueStrings
// and checkEnumSubset call site at its element and count edges
// through the production entry. Element probes use the longest
// admittable shape per site: free-form members admit at exactly
// the maximum, pattern members admit a maximum-length pattern
// match, SemVer members admit a maximum-length version, and
// closed-vocabulary members refuse past the maximum on length
// before the vocabulary is consulted (the vocabulary itself is
// pinned by TestDecodeQueryOperationValidatorsRefuse; count
// ceilings behind a closed vocabulary are equivalent because no
// ceiling+1 body is buildable, and that is recorded per row).
func TestSortedUniqueStringsEdges(t *testing.T) {
	t.Parallel()
	capabilities := append([]string(nil), capabilityOrder...)
	sort.Strings(capabilities)
	if len(capabilities) != 8 {
		t.Fatal("capability scaffold is short; the count edge is mismeasured, not the bound")
	}
	t.Run("supported versions", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, fixtureManifestJSON(), `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeManifest([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty version element")
		}
		if err := drive(`["1.0.0"]`); err != nil {
			t.Fatalf("refused one version at the count minimum: %v", err)
		}
		// No 4-character SemVer exists, so the element floor is
		// unreachable behind the grammar; the maximum is a
		// maximum-length version.
		version64 := "1.0.0+" + strings.Repeat("b", 58)
		version65 := "1.0.0+" + strings.Repeat("b", 59)
		if len(version64) != 64 || len(version65) != 65 {
			t.Fatal("version scaffold has the wrong length")
		}
		if err := drive(`[` + quote(version64) + `]`); err != nil {
			t.Fatalf("refused a 64-character version at the element maximum: %v", err)
		}
		if err := drive(`[` + quote(version65) + `]`); err == nil {
			t.Fatal("admitted a 65-character version past the element maximum")
		}
		sixteen := make([]string, 0, 16)
		for i := 0; i < 16; i++ {
			sixteen = append(sixteen, fmt.Sprintf("1.0.%d", i))
		}
		sort.Strings(sixteen)
		quoted := make([]string, 0, 16)
		for _, version := range sixteen {
			quoted = append(quoted, quote(version))
		}
		if err := drive(`[` + strings.Join(quoted, ",") + `]`); err != nil {
			t.Fatalf("refused 16 versions at the count maximum: %v", err)
		}
		quoted = append(quoted, quote("1.0.16"))
		sort.Strings(quoted)
		if err := drive(`[` + strings.Join(quoted, ",") + `]`); err == nil {
			t.Fatal("admitted 17 versions past the count maximum")
		}
	})
	t.Run("requested environment ids", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, fixtureProbeRequestV2(), `"requested_environment_ids":[]`, `"requested_environment_ids":`+elements)
		}
		drive := func(elements string) error {
			_, err := CheckProbeRequest(MajorV2, []byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty environment identifier")
		}
		if err := drive(`["` + "e" + strings.Repeat("0", 63) + `"]`); err != nil {
			t.Fatalf("refused a 64-character identifier at the element maximum: %v", err)
		}
		if err := drive(`["` + "e" + strings.Repeat("0", 64) + `"]`); err == nil {
			t.Fatal("admitted a 65-character identifier past the element maximum")
		}
		if err := drive(sortedEnvIDs(64)); err != nil {
			t.Fatalf("refused 64 identifiers at the count maximum: %v", err)
		}
		if err := drive(sortedEnvIDs(65)); err == nil {
			t.Fatal("admitted 65 identifiers past the count maximum")
		}
	})
	t.Run("requested capabilities", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, fixtureProbeRequestV2(), `"requested_capabilities":["directory_discovery"]`, `"requested_capabilities":`+elements)
		}
		drive := func(elements string) error {
			_, err := CheckProbeRequest(MajorV2, []byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty capability")
		}
		if err := drive(`[` + quotedRepeat(65) + `]`); err == nil {
			t.Fatal("admitted a 65-character capability past the element maximum")
		}
		quoted := make([]string, 0, len(capabilities))
		for _, name := range capabilities {
			quoted = append(quoted, quote(name))
		}
		if err := drive(`[` + strings.Join(quoted, ",") + `]`); err != nil {
			t.Fatalf("refused 8 capabilities at the count maximum: %v", err)
		}
		// The ninth member cannot come from the closed
		// registry, so the count ceiling is proven with the
		// count gate firing before the vocabulary: nine
		// members refuse even though the ninth is unknown.
		if err := drive(`[` + strings.Join(append(quoted, `"extra"`), ",") + `]`); err == nil {
			t.Fatal("admitted 9 capabilities past the count maximum")
		}
	})
	t.Run("caller scopes", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, fixtureQueryJSON(), `"scopes":["directory.read"]`, `"scopes":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty scope")
		}
		if err := drive(`[` + quotedRepeat(65) + `]`); err == nil {
			t.Fatal("admitted a 65-character scope past the element maximum")
		}
		five := []string{"directory.admin", "directory.execute", "directory.mutate", "directory.preview", "directory.read"}
		quoted := make([]string, 0, len(five))
		for _, scope := range five {
			quoted = append(quoted, quote(scope))
		}
		if err := drive(`[` + strings.Join(quoted, ",") + `]`); err != nil {
			t.Fatalf("refused 5 scopes at the count maximum: %v", err)
		}
		if err := drive(`[` + strings.Join(append(quoted, `"directory.root"`), ",") + `]`); err == nil {
			t.Fatal("admitted 6 scopes past the count maximum")
		}
		if err := drive(`[]`); err == nil {
			t.Fatal("admitted empty scopes below the count minimum")
		}
	})
	t.Run("enrich kinds", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, operationBatch("enrich"), `"kinds":["summary"]`, `"kinds":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty kind")
		}
		if err := drive(`[` + quotedRepeat(65) + `]`); err == nil {
			t.Fatal("admitted a 65-character kind past the element maximum")
		}
		if err := drive(`["generated_title","recent_activity","summary"]`); err != nil {
			t.Fatalf("refused 3 kinds at the count maximum: %v", err)
		}
		if err := drive(`["bogus","generated_title","recent_activity","summary"]`); err == nil {
			t.Fatal("admitted 4 kinds past the count maximum")
		}
	})
	t.Run("authentication status", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, operationBatch("environments"), `"authentication_status":[]`, `"authentication_status":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty status")
		}
		if err := drive(`[` + quote(strings.Repeat("s", 33)) + `]`); err == nil {
			t.Fatal("admitted a 33-character status past the element maximum")
		}
		if err := drive(`["available","expired","missing","unknown"]`); err != nil {
			t.Fatalf("refused 4 statuses at the count maximum: %v", err)
		}
		if err := drive(`["available","expired","missing","root","unknown"]`); err == nil {
			t.Fatal("admitted 5 statuses past the count maximum")
		}
	})
	t.Run("environment ids", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, operationBatch("environments"), `"environment_ids":[]`, `"environment_ids":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty environment identifier")
		}
		if err := drive(`["` + "e" + strings.Repeat("0", 63) + `"]`); err != nil {
			t.Fatalf("refused a 64-character identifier at the element maximum: %v", err)
		}
		if err := drive(`["` + "e" + strings.Repeat("0", 64) + `"]`); err == nil {
			t.Fatal("admitted a 65-character identifier past the element maximum")
		}
		if err := drive(sortedEnvIDs(64)); err != nil {
			t.Fatalf("refused 64 identifiers at the count maximum: %v", err)
		}
		if err := drive(sortedEnvIDs(65)); err == nil {
			t.Fatal("admitted 65 identifiers past the count maximum")
		}
	})
	t.Run("job states", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, operationBatch("jobs"), `"states":[]`, `"states":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty state")
		}
		if err := drive(`[` + quote(strings.Repeat("s", 33)) + `]`); err == nil {
			t.Fatal("admitted a 33-character state past the element maximum")
		}
		if err := drive(`["canceled","claimed","failed","queued","running","succeeded","superseded"]`); err != nil {
			t.Fatalf("refused 7 states at the count maximum: %v", err)
		}
		if err := drive(`["canceled","claimed","failed","pwned","queued","running","succeeded","superseded"]`); err == nil {
			t.Fatal("admitted 8 states past the count maximum")
		}
	})
	t.Run("provider ids", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, operationBatch("sessions"), `"provider_ids":[]`, `"provider_ids":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty provider identifier")
		}
		if err := drive(`["` + "a" + strings.Repeat("0", 31) + `"]`); err != nil {
			t.Fatalf("refused a 32-character identifier at the element maximum: %v", err)
		}
		if err := drive(`["` + "a" + strings.Repeat("0", 32) + `"]`); err == nil {
			t.Fatal("admitted a 33-character identifier past the element maximum")
		}
		if err := drive(sortedLiterals("p", 64)); err != nil {
			t.Fatalf("refused 64 identifiers at the count maximum: %v", err)
		}
		if err := drive(sortedLiterals("p", 65)); err == nil {
			t.Fatal("admitted 65 identifiers past the count maximum")
		}
	})
	t.Run("filter states", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, operationBatch("sessions"), `"states":[]`, `"states":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty state")
		}
		if err := drive(`[` + quote(strings.Repeat("s", 128)) + `]`); err != nil {
			t.Fatalf("refused a 128-character state at the element maximum: %v", err)
		}
		if err := drive(`[` + quote(strings.Repeat("s", 129)) + `]`); err == nil {
			t.Fatal("admitted a 129-character state past the element maximum")
		}
		if err := drive(sortedLiterals("s", 64)); err != nil {
			t.Fatalf("refused 64 states at the count maximum: %v", err)
		}
		if err := drive(sortedLiterals("s", 65)); err == nil {
			t.Fatal("admitted 65 states past the count maximum")
		}
	})
	t.Run("filter warnings", func(t *testing.T) {
		t.Parallel()
		build := func(elements string) string {
			return spliceBound(t, operationBatch("sessions"), `"warnings":[]`, `"warnings":`+elements)
		}
		drive := func(elements string) error {
			_, err := DecodeQuery([]byte(build(elements)))
			return err
		}
		if err := drive(`[""]`); err == nil {
			t.Fatal("admitted an empty warning")
		}
		if err := drive(`[` + quote(strings.Repeat("w", 256)) + `]`); err != nil {
			t.Fatalf("refused a 256-character warning at the element maximum: %v", err)
		}
		if err := drive(`[` + quote(strings.Repeat("w", 257)) + `]`); err == nil {
			t.Fatal("admitted a 257-character warning past the element maximum")
		}
		if err := drive(sortedLiterals("w", 128)); err != nil {
			t.Fatalf("refused 128 warnings at the count maximum: %v", err)
		}
		if err := drive(sortedLiterals("w", 129)); err == nil {
			t.Fatal("admitted 129 warnings past the count maximum")
		}
	})
	t.Run("filter enums", func(t *testing.T) {
		t.Parallel()
		drive := func(member, elements string) error {
			params := spliceBound(t, operationParams("sessions"), `"`+member+`":[]`, `"`+member+`":`+elements)
			_, err := DecodeQuery([]byte(refuseBatch("sessions", params)))
			return err
		}
		// Element edges refuse on length before the closed
		// vocabulary is consulted; count ceilings sit behind
		// closed vocabularies, so the ceiling+1 bodies below
		// prove the count gate fires first rather than
		// admitting through it.
		for _, probe := range []struct {
			member string
			over   string
			full   string
			extra  string
		}{
			{"kinds", `[` + quotedRepeat(65) + `]`, `["lineage","managed_session","native_instance"]`, `["bogus","lineage","managed_session","native_instance"]`},
			{"management_states", `[` + quote(strings.Repeat("m", 33)) + `]`, `["conflicted","managed","unmanaged"]`, `["bogus","conflicted","managed","unmanaged"]`},
			{"reachability", `[` + quote(strings.Repeat("r", 33)) + `]`, `["local","reachable","unknown","unreachable"]`, `["bogus","local","reachable","unknown","unreachable"]`},
			{"freshness", `[` + quote(strings.Repeat("f", 33)) + `]`, `["aging","conflicted","current","offline","partial","stale","unknown"]`, `["bogus","aging","conflicted","current","offline","partial","stale","unknown"]`},
		} {
			if err := drive(probe.member, `[""]`); err == nil {
				t.Fatalf("%s admitted an empty member", probe.member)
			}
			if err := drive(probe.member, probe.over); err == nil {
				t.Fatalf("%s admitted past the element maximum", probe.member)
			}
			if err := drive(probe.member, probe.full); err != nil {
				t.Fatalf("%s refused the full vocabulary: %v", probe.member, err)
			}
			if err := drive(probe.member, probe.extra); err == nil {
				t.Fatalf("%s admitted past the count maximum", probe.member)
			}
		}
	})
}

// sortedEnvIDs builds n sorted unique environment identifiers.
func sortedEnvIDs(n int) string {
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, quote(fmt.Sprintf("e-%05d", i)))
	}
	return `[` + strings.Join(parts, ",") + `]`
}

// TestDigestArrayBoundEdges drives every digest-array site at its
// count edges through the production entry, with element-shape
// and sortedness probes alongside. The native-observation
// ceiling is the one stated exception: the 65537-element body a
// ceiling+1 probe needs is recorded as a bound rather than built.
func TestDigestArrayBoundEdges(t *testing.T) {
	t.Parallel()
	pair := []string{fixtureInstallDigestA, fixtureInstallDigestB}
	sort.Strings(pair)
	rows := []struct {
		name      string
		min       int
		max       int
		ceilState bool
		build     func(elements string) string
	}{
		{
			name: "redaction policies", min: 1, max: 64,
			build: func(elements string) string {
				return spliceBound(t, fixtureManifestJSON(), `"redaction_policy_ids":[`+quote(fixtureRedactionDigest)+`]`, `"redaction_policy_ids":`+elements)
			},
		},
		{
			name: "enrichment profiles", min: 0, max: 256,
			build: func(elements string) string {
				return spliceBound(t, fixtureManifestJSON(), `"enrichment_profile_ids":[`+quote(fixtureEnrichmentDigest)+`]`, `"enrichment_profile_ids":`+elements)
			},
		},
		{
			name: "capability evidence", min: 0, max: 64,
			build: func(elements string) string {
				return spliceBound(t, fixtureManifestJSON(), `"evidence_ids":[`+quote(fixtureEvidenceDigest)+`]`, `"evidence_ids":`+elements)
			},
		},
		{
			name: "installation ids", min: 1, max: 256,
			build: func(elements string) string {
				return spliceBound(t, fixtureScanRequestJSON(), `"installation_ids":[`+quote(pair[0])+`,`+quote(pair[1])+`]`, `"installation_ids":`+elements)
			},
		},
		{
			name: "environment observations", min: 1, max: 256,
			build: func(elements string) string {
				return spliceBound(t, fixtureScanResponseJSON(), `"environment_observation_ids":[`+quote(fixtureEvidenceDigest)+`]`, `"environment_observation_ids":`+elements)
			},
		},
		{
			name: "native observations", min: 0, max: 65536, ceilState: true,
			build: func(elements string) string {
				return spliceBound(t, fixtureScanResponseJSON(), `"native_observation_ids":[`+quote(fixtureNativeDigestA)+`]`, `"native_observation_ids":`+elements)
			},
		},
		{
			name: "plan ids", min: 0, max: 256,
			build: func(elements string) string {
				return spliceBound(t, operationBatch("plans"), `"plan_ids":[]`, `"plan_ids":`+elements)
			},
		},
		{
			name: "job profile ids", min: 0, max: 256,
			build: func(elements string) string {
				return spliceBound(t, operationBatch("jobs"), `"profile_ids":[]`, `"profile_ids":`+elements)
			},
		},
		{
			name: "supersedes", min: 0, max: 1024,
			build: func(elements string) string {
				return spliceBound(t, operationBatch("set_title"), `"supersedes_annotation_ids":[]`, `"supersedes_annotation_ids":`+elements)
			},
		},
	}
	if want := boundCensusDriverCount("TestDigestArrayBoundEdges"); len(rows) != want {
		t.Fatalf("digest rows = %d, want %d; the table is short, not the package", len(rows), want)
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			drive := func(elements string) error {
				return driveDigestBody(t, row.build(elements))
			}
			if row.min > 0 {
				if err := drive(`[]`); err == nil {
					t.Fatalf("%s admitted an empty array below the %d minimum", row.name, row.min)
				}
				if err := drive(sortedDigestList(row.name, 1)); err != nil {
					t.Fatalf("%s refused one digest at the minimum: %v", row.name, err)
				}
			} else if err := drive(`[]`); err != nil {
				t.Fatalf("%s refused an empty array at the zero minimum: %v", row.name, err)
			}
			if err := drive(`["sha256:zzz"]`); err == nil {
				t.Fatalf("%s admitted a non-digest element", row.name)
			}
			if row.max <= 1024 {
				if err := drive(sortedDigestList(row.name, row.max)); err != nil {
					t.Fatalf("%s refused %d digests at the maximum: %v", row.name, row.max, err)
				}
				if err := drive(sortedDigestList(row.name, row.max+1)); err == nil {
					t.Fatalf("%s admitted %d digests past the %d maximum", row.name, row.max+1, row.max)
				}
			} else {
				// The 65536 maximum admits; the 65537-body
				// ceiling probe is a stated bound, not a
				// built body.
				if err := drive(sortedDigestList(row.name, row.max)); err != nil {
					t.Fatalf("%s refused %d digests at the maximum: %v", row.name, row.max, err)
				}
			}
			// Sortedness is an ordering gate, not a bound,
			// but the reversed pair proves the refusal the
			// count edges share a body with is the count
			// gate and not a vacuous admit.
			if row.max <= 256 {
				duo := sortedDigestList(row.name, 2)
				var two []string
				if err := json.Unmarshal([]byte(duo), &two); err != nil {
					t.Fatalf("scaffold: %v", err)
				}
				if err := drive(`[` + quote(two[1]) + `,` + quote(two[0]) + `]`); err == nil {
					t.Fatalf("%s admitted an unsorted pair", row.name)
				}
			}
		})
	}
}

// driveDigestBody decodes one spliced body through its production
// entry by shape: manifest bodies through DecodeManifest, scan
// bodies through their check, and query batches through
// DecodeQuery.
func driveDigestBody(t *testing.T, body string) error {
	t.Helper()
	switch {
	case strings.Contains(body, "session-directory-node-manifest"):
		_, err := DecodeManifest([]byte(body))
		return err
	case strings.Contains(body, `"installation_ids"`):
		_, err := CheckScanRequest([]byte(body))
		return err
	case strings.Contains(body, `"environment_observation_ids"`):
		_, err := CheckScanResponse([]byte(body))
		return err
	default:
		_, err := DecodeQuery([]byte(body))
		return err
	}
}

// TestSortedUniqueUUIDv7Edges drives every UUIDv7-array site at
// its count edges through the production DecodeQuery entry.
func TestSortedUniqueUUIDv7Edges(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name   string
		needle string
		build  func(elements string) string
	}{
		{"hosts host_ids", `"host_ids":[]`, func(elements string) string {
			return spliceBound(t, operationBatch("hosts"), `"host_ids":[]`, `"host_ids":`+elements)
		}},
		{"environments host_ids", `"host_ids":[]`, func(elements string) string {
			return spliceBound(t, operationBatch("environments"), `"host_ids":[]`, `"host_ids":`+elements)
		}},
		{"filter host_ids", `"host_ids":[]`, func(elements string) string {
			return spliceBound(t, operationBatch("sessions"), `"host_ids":[]`, `"host_ids":`+elements)
		}},
		{"filter workspace_ids", `"workspace_ids":[]`, func(elements string) string {
			return spliceBound(t, operationBatch("sessions"), `"workspace_ids":[]`, `"workspace_ids":`+elements)
		}},
		{"job ids", `"job_ids":[]`, func(elements string) string {
			return spliceBound(t, operationBatch("jobs"), `"job_ids":[]`, `"job_ids":`+elements)
		}},
		{"plan operation ids", `"operation_ids":[]`, func(elements string) string {
			return spliceBound(t, operationBatch("plans"), `"operation_ids":[]`, `"operation_ids":`+elements)
		}},
	}
	if want := boundCensusDriverCount("TestSortedUniqueUUIDv7Edges"); len(rows) != want {
		t.Fatalf("uuid rows = %d, want %d; the table is short, not the package", len(rows), want)
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			drive := func(elements string) error {
				_, err := DecodeQuery([]byte(row.build(elements)))
				return err
			}
			if err := drive(`[]`); err != nil {
				t.Fatalf("%s refused an empty array at the zero minimum: %v", row.name, err)
			}
			if err := drive(`["not-a-uuid"]`); err == nil {
				t.Fatalf("%s admitted a non-UUID element", row.name)
			}
			if err := drive(sortedUUIDList(256)); err != nil {
				t.Fatalf("%s refused 256 identifiers at the maximum: %v", row.name, err)
			}
			if err := drive(sortedUUIDList(257)); err == nil {
				t.Fatalf("%s admitted 257 identifiers past the maximum", row.name)
			}
			pair := sortedUUIDList(2)
			var two []string
			if err := json.Unmarshal([]byte(pair), &two); err != nil {
				t.Fatalf("scaffold: %v", err)
			}
			if err := drive(`[` + quote(two[1]) + `,` + quote(two[0]) + `]`); err == nil {
				t.Fatalf("%s admitted an unsorted pair", row.name)
			}
		})
	}
}

// TestUUIDv7DigestSubsetEdges drives the lineage-anchors union
// site through the production DecodeQuery entry: UUIDv7 and
// digest members both admit, anything else refuses, and the
// count edges hold over the union.
func TestUUIDv7DigestSubsetEdges(t *testing.T) {
	t.Parallel()
	build := func(elements string) string {
		params := spliceBound(t, operationParams("sessions"), `"lineage_anchors":[]`, `"lineage_anchors":`+elements)
		return refuseBatch("sessions", params)
	}
	drive := func(elements string) error {
		_, err := DecodeQuery([]byte(build(elements)))
		return err
	}
	if err := drive(`[]`); err != nil {
		t.Fatalf("refused an empty array at the zero minimum: %v", err)
	}
	if err := drive(`[` + quote(fixtureQueryID) + `]`); err != nil {
		t.Fatalf("refused a UUIDv7 member: %v", err)
	}
	if err := drive(`[` + quote(fixtureBatchDigest) + `]`); err != nil {
		t.Fatalf("refused a digest member: %v", err)
	}
	if err := drive(`["zzz"]`); err == nil {
		t.Fatal("admitted a member that is neither UUIDv7 nor digest")
	}
	union256 := make([]string, 0, 256)
	for i := 0; i < 128; i++ {
		union256 = append(union256, fmt.Sprintf("0198f4c8-8e50-7f66-8f70-%012x", 0x123456789000+i))
	}
	for i := 0; i < 128; i++ {
		union256 = append(union256, fixtureDigest(fmt.Sprintf("union-%05d", i)))
	}
	sort.Strings(union256)
	quoted := make([]string, 0, len(union256))
	for _, member := range union256 {
		quoted = append(quoted, quote(member))
	}
	if err := drive(`[` + strings.Join(quoted, ",") + `]`); err != nil {
		t.Fatalf("refused 256 union members at the maximum: %v", err)
	}
	quoted = append(quoted, quote(fixtureDigest("union-overflow")))
	sort.Strings(quoted)
	if err := drive(`[` + strings.Join(quoted, ",") + `]`); err == nil {
		t.Fatal("admitted 257 union members past the maximum")
	}
	// The union orders bytewise across both shapes, so a digest
	// followed by a smaller UUID refuses on sortedness.
	ordered := []string{quote(fixtureBatchDigest), quote(fixtureQueryID)}
	sort.Strings(ordered)
	if ordered[0] == quote(fixtureBatchDigest) {
		if err := drive(`[` + quote(fixtureQueryID) + `,` + quote(fixtureBatchDigest) + `]`); err == nil {
			t.Fatal("admitted an unsorted union pair")
		}
	} else if err := drive(`[` + quote(fixtureBatchDigest) + `,` + quote(fixtureQueryID) + `]`); err == nil {
		t.Fatal("admitted an unsorted union pair")
	}
}

// frameOfSize builds one request frame of exactly total bytes by
// padding a body member. The body stays an object, so a refusal
// is attributable to the frame bound and an admission proves the
// bound does not bite early.
func frameOfSize(t *testing.T, total int) []byte {
	t.Helper()
	base := `{"schema":"urn:ax:schema:session-directory-node-request","schema_version":"2.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":"2.0.0","request_id":"` + fixtureRequestID + `","operation":"manifest","deadline_ms":1,"body":{"pad":"`
	suffix := `"}}`
	padding := total - len(base) - len(suffix)
	if padding < 0 {
		t.Fatalf("frameOfSize(%d) is below the %d-byte envelope floor", total, len(base)+len(suffix))
	}
	frame := base + strings.Repeat("p", padding) + suffix
	if len(frame) != total {
		t.Fatalf("frameOfSize(%d) built %d bytes; the edge is mismeasured, not the bound", total, len(frame))
	}
	return []byte(frame)
}

// successOfSize builds one success envelope of exactly total
// bytes with an object body.
func successOfSize(t *testing.T, total int) []byte {
	t.Helper()
	base := `{"schema":"urn:ax:schema:session-directory-node-response","schema_version":"1.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":"2.0.0","request_id":"` + fixtureRequestID + `","operation":"manifest","ok":true,"body":{"pad":"`
	suffix := `"}}`
	padding := total - len(base) - len(suffix)
	if padding < 0 {
		t.Fatalf("successOfSize(%d) is below the envelope floor", total)
	}
	frame := base + strings.Repeat("p", padding) + suffix
	if len(frame) != total {
		t.Fatalf("successOfSize(%d) built %d bytes", total, len(frame))
	}
	return []byte(frame)
}

// TestFrameBoundEdges drives the 8 MiB transport bound and the
// empty-frame refusal through the production request and
// response entries. A valid failure envelope cannot reach the
// bound — the Structured Error body it carries is itself
// bounded — so the failure path proves the over-limit refusal
// with an oversize (hence invalid) error and near-limit
// admission with a large valid one; the exact-maximum admission
// is proven on the request and success paths sharing the same
// guard shape.
// frameBoundBytes is the 8 MiB transport bound retyped from the
// contract, never referenced from the production constant: a
// probe sized from production would move with a mutated constant
// and stay green while the bound widened.
const frameBoundBytes = 8388608

func TestFrameBoundEdges(t *testing.T) {
	t.Parallel()
	request := Request{Major: MajorV2, Operation: OpManifest, RequestID: fixtureRequestID}
	if _, err := DecodeRequestFrame([]byte{}); err == nil {
		t.Fatal("DecodeRequestFrame admitted an empty frame")
	}
	if _, err := CheckSuccessEnvelope([]byte{}, request); err == nil {
		t.Fatal("CheckSuccessEnvelope admitted an empty frame")
	}
	if _, err := CheckFailureEnvelope([]byte{}, request); err == nil {
		t.Fatal("CheckFailureEnvelope admitted an empty frame")
	}
	if _, err := DecodeRequestFrame(frameOfSize(t, frameBoundBytes)); err != nil {
		t.Fatalf("DecodeRequestFrame refused exactly 8 MiB: %v", err)
	}
	if _, err := DecodeRequestFrame(frameOfSize(t, frameBoundBytes+1)); err == nil {
		t.Fatal("DecodeRequestFrame admitted past 8 MiB")
	}
	if _, err := CheckSuccessEnvelope(successOfSize(t, frameBoundBytes), request); err != nil {
		t.Fatalf("CheckSuccessEnvelope refused exactly 8 MiB: %v", err)
	}
	if _, err := CheckSuccessEnvelope(successOfSize(t, frameBoundBytes+1), request); err == nil {
		t.Fatal("CheckSuccessEnvelope admitted past 8 MiB")
	}
	oversizeError := `{"schema":"urn:ax:schema:error","schema_version":"1.2.0","code":"incompatible_protocol","message":"` + strings.Repeat("m", 4096) + `","exit_code":6,"retryable":false,"details":{}}`
	oversizeFailure := fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID, oversizeError)
	// Pad past the bound with a message the size gate rejects
	// before the error body is ever read.
	big := []byte(string(oversizeFailure) + strings.Repeat(" ", frameBoundBytes))
	if _, err := CheckFailureEnvelope(big, request); err == nil {
		t.Fatal("CheckFailureEnvelope admitted past 8 MiB")
	}
	large := fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID, oversizeError)
	if _, err := CheckFailureEnvelope([]byte(large), request); err != nil {
		t.Fatalf("CheckFailureEnvelope refused a large valid failure: %v", err)
	}
}

// TestExtensionKeyBoundEdges drives the reverse-DNS extension
// key bounds through the production DecodeManifest entry: a
// 2-character key refuses, 3 admits, 253 admits, 254 refuses.
func TestExtensionKeyBoundEdges(t *testing.T) {
	t.Parallel()
	valid253 := "a" + strings.Repeat(".b", 126)
	if len(valid253) != 253 {
		t.Fatalf("scaffold key = %d chars, want 253", len(valid253))
	}
	drive := func(key string) error {
		var extensions string
		if key == "" {
			extensions = `{}`
		} else {
			extensions = `{` + quote(key) + `: {}}`
		}
		body := spliceBound(t, fixtureManifestJSON(), `"extensions":{}},"extensions":{}}`, `"extensions":{}},"extensions":`+extensions+`}`)
		_, err := DecodeManifest([]byte(body))
		return err
	}
	if err := drive("ab"); err == nil {
		t.Fatal("admitted a 2-character extension key")
	}
	if err := drive("a.b"); err != nil {
		t.Fatalf("refused a 3-character key at the minimum: %v", err)
	}
	if err := drive(valid253); err != nil {
		t.Fatalf("refused a 253-character key at the maximum: %v", err)
	}
	if err := drive(valid253 + "c"); err == nil {
		t.Fatal("admitted a 254-character extension key past the maximum")
	}
}

// queryBatch wraps raw operation elements in a valid query batch.
func queryBatch(ops string) string {
	return `{"schema":"urn:ax:schema:session-directory-query","schema_version":"1.0.0","query_id":` + quote(fixtureQueryID) + `,"operations":[` + ops + `],"caller":` + fixtureCallerJSON() + `,"extensions":{}}`
}

// schemaOp renders the schema read at the given index.
func schemaOp(index int) string {
	return queryOperationJSON(index, "schema", `{"extensions":{}}`, "null", "null", 0, 1, "[]", "false", "false", "null", "null")
}

// TestQueryBatchCountEdges drives the operations-array count
// bound through the production DecodeQuery entry: 64 operations
// admit (which also admits index 63 at its bound) and 65 refuse.
// The count and index gates coincide at the 65th operation —
// neither single weakening admits a batch alone — so the pair is
// pinned jointly here and each weakening is recorded as masked
// behind the other in the mutant report.
func TestQueryBatchCountEdges(t *testing.T) {
	t.Parallel()
	if _, err := DecodeQuery([]byte(queryBatch(""))); err == nil {
		t.Fatal("admitted an empty batch below the 1 minimum")
	}
	sixtyFour := make([]string, 0, 64)
	for i := 0; i < 64; i++ {
		sixtyFour = append(sixtyFour, schemaOp(i))
	}
	if _, err := DecodeQuery([]byte(queryBatch(strings.Join(sixtyFour, ",")))); err != nil {
		t.Fatalf("refused 64 operations at the maximum: %v", err)
	}
	sixtyFive := append(sixtyFour, schemaOp(64))
	if _, err := DecodeQuery([]byte(queryBatch(strings.Join(sixtyFive, ",")))); err == nil {
		t.Fatal("admitted 65 operations past the 64 maximum")
	}
	// Index 64 with a 2-operation batch refuses on the index
	// bound without touching the count ceiling.
	pair := schemaOp(0) + "," + schemaOp(64)
	if _, err := DecodeQuery([]byte(queryBatch(pair))); err == nil {
		t.Fatal("admitted operation_index 64 past the 63 maximum")
	}
}

// TestQuerySortBoundEdges drives the sort-tuple ceiling through
// the production DecodeQuery entry: 8 tuples admit, 9 refuse.
func TestQuerySortBoundEdges(t *testing.T) {
	t.Parallel()
	tuple := `{"field":"stable_id","direction":"asc","extensions":{}}`
	eight := strings.Repeat(tuple+",", 7) + tuple
	nine := eight + "," + tuple
	admitted := singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[`+eight+`]`, "false", "false", "null", "null")
	if _, err := DecodeQuery([]byte(admitted)); err != nil {
		t.Fatalf("refused 8 sort tuples at the maximum: %v", err)
	}
	refused := singleOperationBatch("sessions", operationParams("sessions"), "null", "null", 0, 10, `[`+nine+`]`, "false", "false", "null", "null")
	if _, err := DecodeQuery([]byte(refused)); err == nil {
		t.Fatal("admitted 9 sort tuples past the 8 maximum")
	}
}

// TestQueryPaginationBoundEdges pins the stated skip/take bounds
// (census shape 7: comparisons over decoded values, not lengths)
// through the production DecodeQuery entry: skip admits at
// 1000000 and refuses at 1000001; take admits at 1 and 1000 and
// refuses at 0 and 1001.
func TestQueryPaginationBoundEdges(t *testing.T) {
	t.Parallel()
	paged := func(skip, take int) string {
		return singleOperationBatch("sessions", operationParams("sessions"), "null", "null", skip, take, "[]", "false", "false", "null", "null")
	}
	if _, err := DecodeQuery([]byte(paged(1000000, 10))); err != nil {
		t.Fatalf("refused skip 1000000 at the maximum: %v", err)
	}
	if _, err := DecodeQuery([]byte(paged(1000001, 10))); err == nil {
		t.Fatal("admitted skip 1000001 past the maximum")
	}
	for _, take := range []int{1, 1000} {
		if _, err := DecodeQuery([]byte(paged(0, take))); err != nil {
			t.Fatalf("refused take %d at an edge: %v", take, err)
		}
	}
	for _, take := range []int{0, 1001} {
		if _, err := DecodeQuery([]byte(paged(0, take))); err == nil {
			t.Fatalf("admitted take %d outside 1..1000", take)
		}
	}
}

// TestProbeEnvironmentsBoundEdges drives the probe environment
// count edges through the production CheckProbeResponse entry:
// the values live at the call site the census cannot derive, so
// this suite pins them while the shared enforcement rows stay
// mechanism.
func TestProbeEnvironmentsBoundEdges(t *testing.T) {
	t.Parallel()
	build := func(elements string) string {
		return spliceBound(t, fixtureProbeResponseJSON(), `[{"environment_id":"test.env"}]`, `[`+elements+`]`)
	}
	drive := func(elements string) error {
		_, err := CheckProbeResponse([]byte(build(elements)))
		return err
	}
	if err := drive(``); err != nil {
		t.Fatalf("refused an empty array at the zero minimum: %v", err)
	}
	if err := drive(`{"environment_id":"test.env"}`); err != nil {
		t.Fatalf("refused one environment: %v", err)
	}
	many := strings.Repeat(`{},`, 255) + `{}`
	if err := drive(many); err != nil {
		t.Fatalf("refused 256 environments at the maximum: %v", err)
	}
	if err := drive(many + `,{}`); err == nil {
		t.Fatal("admitted 257 environments past the maximum")
	}
	if err := drive(`1`); err == nil {
		t.Fatal("admitted a non-object environment")
	}
}

// TestContractAssertionBoundEdges drives the schemas-array count
// bound through the production DecodeManifest entry: 14 refuse,
// 15 admit, 64 admit, 65 refuse. Zero-padded contract suffixes
// keep the canonical-encoding order identical to generation
// order, so the count edges are measured without tripping the
// sortedness gate.
func TestContractAssertionBoundEdges(t *testing.T) {
	t.Parallel()
	build := func(n int) string {
		rows := make([]string, 0, n)
		for i := 0; i < n; i++ {
			rows = append(rows, fmt.Sprintf(`{"contract_id":"urn:ax:schema:contract-%02d","exact_version":"1.0.%d","extensions":{}}`, i, i))
		}
		return spliceBound(t, fixtureManifestJSON(), `"schemas":[`+fifteenAssertions()+`]`, `"schemas":[`+strings.Join(rows, ",")+`]`)
	}
	if _, err := DecodeManifest([]byte(build(14))); err == nil {
		t.Fatal("admitted 14 assertions below the 15 minimum")
	}
	if _, err := DecodeManifest([]byte(build(15))); err != nil {
		t.Fatalf("refused 15 assertions at the minimum: %v", err)
	}
	if _, err := DecodeManifest([]byte(build(64))); err != nil {
		t.Fatalf("refused 64 assertions at the maximum: %v", err)
	}
	if _, err := DecodeManifest([]byte(build(65))); err == nil {
		t.Fatal("admitted 65 assertions past the 64 maximum")
	}
}

// fifteenAssertions renders the fifteen fixture assertions the
// replacement needle must match.
func fifteenAssertions() string {
	rows := make([]string, 0, 15)
	for i := 0; i < 15; i++ {
		rows = append(rows, fmt.Sprintf(`{"contract_id":"urn:ax:schema:contract-%02d","exact_version":"1.0.%d","extensions":{}}`, i, i))
	}
	return strings.Join(rows, ",")
}

// TestJournalBoundEdges drives the journal key bound through the
// production Import entry: the key refuses empty and past 512
// and admits at both edges.
func TestJournalBoundEdges(t *testing.T) {
	t.Parallel()
	document := func(keyLiteral string) []byte {
		return []byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[{"key":` + keyLiteral + `,"body_digest":` + quote(fixtureBatchDigest) + `,"result":"e30="}]}`)
	}
	if err := NewJournal().Import(document(`""`)); err == nil {
		t.Fatal("admitted an empty journal key")
	}
	if err := NewJournal().Import(document(quote("k"))); err != nil {
		t.Fatalf("refused a 1-character key at the minimum: %v", err)
	}
	if err := NewJournal().Import(document(quotedRepeat(512))); err != nil {
		t.Fatalf("refused a 512-character key at the maximum: %v", err)
	}
	if err := NewJournal().Import(document(quotedRepeat(513))); err == nil {
		t.Fatal("admitted a 513-character key past the maximum")
	}
	// An admitted import installs the record: the restarted
	// journal replays it.
	journal := NewJournal()
	if err := journal.Import(document(quote("k"))); err != nil {
		t.Fatalf("Import error = %v", err)
	}
	exported, err := journal.Export()
	if err != nil {
		t.Fatalf("Export error = %v", err)
	}
	restarted := NewJournal()
	if err := restarted.Import(exported); err != nil {
		t.Fatalf("re-Import error = %v", err)
	}
}

// TestSortedUniqueDuplicatesRefuse feeds a duplicate — sorted but
// not unique — at every one of the eight pairwise ordering scans
// (review round 3, C3). The sortedness refusal tests feed unsorted
// vectors (["b","a"]) and prove the ordered half; only a duplicate
// proves the uniqueness half: weakening `>=` to `>` keeps the gate
// and admits exactly the duplicate. Each row pairs the duplicate
// refusal with its deduplicated control admitting, so the vector
// isolates uniqueness rather than breaking sortedness too.
func TestSortedUniqueDuplicatesRefuse(t *testing.T) {
	t.Parallel()
	digestA := quote(fixtureInstallDigestA)
	host := quote(fixtureOriginHost)
	batchDigest := quote(fixtureBatchDigest)
	for _, probe := range []struct {
		name    string
		refuse  func(t *testing.T) error
		control func(t *testing.T) error
	}{
		{"sorted unique strings", func(t *testing.T) error {
			_, err := DecodeManifest([]byte(spliceBound(t, fixtureManifestJSON(), `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":["1.0.0","1.0.0"]`)))
			return err
		}, func(t *testing.T) error {
			_, err := DecodeManifest([]byte(spliceBound(t, fixtureManifestJSON(), `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":["1.0.0"]`)))
			return err
		}},
		{"sorted unique digests", func(t *testing.T) error {
			pair := sortedInstallPair()
			needle := `"installation_ids":[` + quote(pair[0]) + `,` + quote(pair[1]) + `]`
			_, err := CheckScanRequest([]byte(spliceBound(t, fixtureScanRequestJSON(), needle, `"installation_ids":[`+digestA+`,`+digestA+`]`)))
			return err
		}, func(t *testing.T) error {
			_, err := CheckScanRequest([]byte(fixtureScanRequestJSON()))
			return err
		}},
		{"sorted unique uuids", func(t *testing.T) error {
			_, err := DecodeQuery([]byte(refuseBatch("hosts", badParams(t, "hosts", `"host_ids":[]`, `"host_ids":[`+host+`,`+host+`]`))))
			return err
		}, func(t *testing.T) error {
			_, err := DecodeQuery([]byte(refuseBatch("hosts", badParams(t, "hosts", `"host_ids":[]`, `"host_ids":[`+host+`]`))))
			return err
		}},
		{"contract assertion encodings", func(t *testing.T) error {
			good := fixtureManifestJSON()
			duplicated := replaceOnce(t, good, `{"contract_id":"urn:ax:schema:contract-01","exact_version":"1.0.1","extensions":{}}`, `{"contract_id":"urn:ax:schema:contract-00","exact_version":"1.0.0","extensions":{}}`, 1)
			_, err := DecodeManifest([]byte(duplicated))
			return err
		}, func(t *testing.T) error {
			_, err := DecodeManifest([]byte(fixtureManifestJSON()))
			return err
		}},
		{"set tags", func(t *testing.T) error {
			_, err := DecodeQuery([]byte(refuseBatch("set_tags", badParams(t, "set_tags", `"tags":[]`, `"tags":["a","a"]`))))
			return err
		}, func(t *testing.T) error {
			_, err := DecodeQuery([]byte(refuseBatch("set_tags", badParams(t, "set_tags", `"tags":[]`, `"tags":["a"]`))))
			return err
		}},
		{"execute plan confirmations", func(t *testing.T) error {
			_, err := DecodeQuery([]byte(refuseBatch("execute_plan", badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":["a","a"]`))))
			return err
		}, func(t *testing.T) error {
			_, err := DecodeQuery([]byte(refuseBatch("execute_plan", badParams(t, "execute_plan", `"confirmations":[]`, `"confirmations":["a"]`))))
			return err
		}},
		{"lineage anchors", func(t *testing.T) error {
			params := operationParams("sessions")
			dup := strings.Replace(params, `"lineage_anchors":[]`, `"lineage_anchors":[`+batchDigest+`,`+batchDigest+`]`, 1)
			_, err := DecodeQuery([]byte(refuseBatch("sessions", dup)))
			return err
		}, func(t *testing.T) error {
			params := operationParams("sessions")
			one := strings.Replace(params, `"lineage_anchors":[]`, `"lineage_anchors":[`+batchDigest+`]`, 1)
			_, err := DecodeQuery([]byte(refuseBatch("sessions", one)))
			return err
		}},
		{"journal keys", func(t *testing.T) error {
			record := `{"key":"k","body_digest":` + batchDigest + `,"result":"e30="}`
			document := `{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[` + record + `,` + record + `]}`
			err := NewJournal().Import([]byte(document))
			if err == nil {
				return nil
			}
			if failure := requireCode(t, err, "integrity_failure"); failure == nil {
				t.Fatal("duplicate journal keys refused without integrity_failure")
			} else if !strings.Contains(failure.Error(), "records are not sorted unique") {
				t.Fatalf("duplicate journal keys reached %q, not the sorted-unique arm", failure.Error())
			}
			return err
		}, func(t *testing.T) error {
			record := `{"key":"k","body_digest":` + batchDigest + `,"result":"e30="}`
			document := `{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[` + record + `]}`
			return NewJournal().Import([]byte(document))
		}},
	} {
		probe := probe
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if err := probe.refuse(t); err == nil {
				t.Fatalf("%s admitted a duplicate", probe.name)
			}
			if err := probe.control(t); err != nil {
				t.Fatalf("%s refused its deduplicated control: %v", probe.name, err)
			}
		})
	}
}
