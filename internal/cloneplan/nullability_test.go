package cloneplan_test

import (
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// TestResourceNullabilityGrid sweeps the full kind x mode-presence x
// blob-presence grid for ExpectedTargetResource through both
// entries against the independent branch oracle: blob rows carry a
// non-null expected_blob_id, directory rows carry null, and mode
// keeps its nullable uint32[0..4095] type on both branches.
func TestResourceNullabilityGrid(t *testing.T) {
	kinds := []string{"blob", "directory"}
	modes := []struct {
		label string
		value *uint64
	}{
		{"null", nil},
		{"zero", u64ptr(0)},
		{"max", u64ptr(4095)},
	}
	blobs := []struct {
		label   string
		present bool
	}{
		{"null", false},
		{"digest", true},
	}
	oracle := func(kind string, blobPresent bool) bool {
		if kind == "blob" {
			return blobPresent
		}
		return !blobPresent
	}
	admitted := 0
	for _, kind := range kinds {
		for _, mode := range modes {
			for _, blob := range blobs {
				want := oracle(kind, blob.present)
				input := validPlanInput()
				resource := validResourceInput(1, "grid")
				resource.Kind = kind
				resource.Mode = mode.value
				if blob.present {
					resource.ExpectedBlobID = strptr(fixtureDigest("grid-blob"))
				} else {
					resource.ExpectedBlobID = nil
				}
				input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
				sealed, buildErr := cloneplan.BuildProjectionPlan(input)
				if (buildErr == nil) != want {
					t.Errorf("kind=%s mode=%s blob=%s: build admits = %v, oracle = %v (err=%v)",
						kind, mode.label, blob.label, buildErr == nil, want, buildErr)
				}
				if want {
					// Admit cells decode the fresh seal: the
					// branch gate and the digest both pass.
					plan, decodeErr := cloneplan.DecodeProjectionPlan(sealed)
					if decodeErr != nil {
						t.Errorf("kind=%s mode=%s blob=%s: fresh seal refused: %v",
							kind, mode.label, blob.label, decodeErr)
					} else if len(plan.ExpectedResources) != 1 {
						t.Errorf("kind=%s mode=%s blob=%s: resources = %d, want 1",
							kind, mode.label, blob.label, len(plan.ExpectedResources))
					}
					admitted++
					continue
				}
				// Refuse cells mutate a sealed shape-valid
				// baseline: the branch gate fires before the
				// self-digest, so the branch literal proves
				// the gate measured.
				base := validPlanInput()
				baseResource := validResourceInput(1, "grid")
				if kind == "directory" {
					baseResource = validDirectoryResourceInput(1, "grid")
				}
				base.ExpectedResources = []cloneplan.ExpectedResourceInput{baseResource}
				baseline := mustBuildPlan(t, base)
				document := decodeDocument(t, baseline)
				row := document["expected_resources"].([]any)[0].(map[string]any)
				if mode.value == nil {
					row["mode"] = nil
				} else {
					row["mode"] = float64(*mode.value)
				}
				if blob.present {
					row["expected_blob_id"] = fixtureDigest("grid-blob")
				} else {
					row["expected_blob_id"] = nil
				}
				_, decodeErr := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
				if decodeErr == nil {
					t.Errorf("kind=%s mode=%s blob=%s: decode admits, oracle refuses",
						kind, mode.label, blob.label)
				} else {
					requireRefusal(t, decodeErr, "expected target resource[0]", "expected_blob_id")
				}
			}
		}
	}
	// 2 kinds x 3 modes x 2 blob states = 12 cells; 6 admit.
	if admitted != 6 {
		t.Fatalf("admitted cells = %d, want 6", admitted)
	}
}

// TestResourceBranchLiterals pins the branch refusal literals on
// both entries.
func TestResourceBranchLiterals(t *testing.T) {
	input := validPlanInput()
	blob := validResourceInput(1, "branch-blob")
	blob.ExpectedBlobID = nil
	input.ExpectedResources = []cloneplan.ExpectedResourceInput{blob}
	_, err := cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "expected target resource[0]", "expected_blob_id is null for blob")

	input = validPlanInput()
	directory := validDirectoryResourceInput(1, "branch-dir")
	directory.ExpectedBlobID = strptr(fixtureDigest("branch-blob"))
	input.ExpectedResources = []cloneplan.ExpectedResourceInput{directory}
	_, err = cloneplan.BuildProjectionPlan(input)
	requireRefusal(t, err, "expected target resource[0]", "expected_blob_id is not null for directory")

	sealed := mustBuildPlan(t, validPlanInput())
	document := decodeDocument(t, sealed)
	document["expected_resources"].([]any)[0].(map[string]any)["expected_blob_id"] = nil
	_, err = cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
	requireRefusal(t, err, "expected target resource[0]", "expected_blob_id is null for blob")
}

// entryBranchOracle is the independent branch rule retyped from the
// spec: blob entries carry byte_count, blob_id, and
// blob_descriptor_id; directory entries carry none of the three.
func entryBranchOracle(kind string, hasCount, hasBlob, hasDescriptor bool) bool {
	if kind == "blob" {
		return hasCount && hasBlob && hasDescriptor
	}
	return !hasCount && !hasBlob && !hasDescriptor
}

// entryGridModes sweeps mode null and non-null on both branches: the
// null cell plus the uint32[0..4095] edges.
func entryGridModes() []struct {
	label string
	value *uint64
} {
	return []struct {
		label string
		value *uint64
	}{
		{"null", nil},
		{"zero", u64ptr(0)},
		{"max", u64ptr(4095)},
	}
}

// gridEntryInput builds the entry candidate for one grid cell: the
// kind with exactly the flagged blob-only members present.
func gridEntryInput(kind string, mode *uint64, hasCount, hasBlob, hasDescriptor bool) cloneplan.ProjectedEntryInput {
	entry := validBlobEntryInput(1, "grid")
	entry.Kind = kind
	entry.Mode = mode
	if !hasCount {
		entry.ByteCount = nil
	}
	if !hasBlob {
		entry.BlobID = nil
	}
	if !hasDescriptor {
		entry.BlobDescriptorID = nil
	}
	return entry
}

// buildGridCell runs one build cell under a panic guard: it returns
// the sealed bytes, the production error, and whether the entry
// panicked.
func buildGridCell(entry cloneplan.ProjectedEntryInput) (sealed []byte, err error, panicked any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			sealed, err, panicked = nil, nil, recovered
		}
	}()
	manifest := validManifestInput()
	manifest.Entries = []cloneplan.ProjectedEntryInput{entry}
	sealed, err = cloneplan.BuildProjectedObjectManifest(manifest)
	return sealed, err, nil
}

// decodeGridCell runs one decode cell under a panic guard.
func decodeGridCell(raw []byte) (manifest cloneplan.ProjectedManifest, err error, panicked any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			manifest, err, panicked = cloneplan.ProjectedManifest{}, nil, recovered
		}
	}()
	manifest, err = cloneplan.DecodeProjectedObjectManifest(raw)
	return manifest, err, nil
}

// mutateEntryRow transplants one grid cell into the sealed baseline:
// the blob row (entries[1]) loses each absent blob-only member, the
// directory row (entries[0]) gains each present one, and mode takes
// the cell value. Entry unknown/missing fires before the self-digest
// check, so every refuse cell asserts a structural branch literal,
// never a digest mismatch.
func mutateEntryRow(t *testing.T, kind string, mode *uint64, hasCount, hasBlob, hasDescriptor bool) []byte {
	t.Helper()
	sealed := mustBuildManifest(t, validManifestInput())
	document := decodeDocument(t, sealed)
	rows := document["entries"].([]any)
	if kind == "blob" {
		row := rows[1].(map[string]any)
		if !hasCount {
			delete(row, "byte_count")
		}
		if !hasBlob {
			delete(row, "blob_id")
		}
		if !hasDescriptor {
			delete(row, "blob_descriptor_id")
		}
	} else {
		row := rows[0].(map[string]any)
		if hasCount {
			row["byte_count"] = float64(8)
		}
		if hasBlob {
			row["blob_id"] = fixtureDigest("stray-blob-id")
		}
		if hasDescriptor {
			row["blob_descriptor_id"] = fixtureDigest("stray-descriptor-id")
		}
	}
	row := rows[1].(map[string]any)
	if kind == "directory" {
		row = rows[0].(map[string]any)
	}
	if mode == nil {
		row["mode"] = nil
	} else {
		row["mode"] = float64(*mode)
	}
	return marshalDocument(t, document)
}

// TestEntryNullabilityGrid sweeps the FULL kind x blob-only
// presence-combination x mode grid for ProjectedObjectEntry through
// both manifest entries against the independent branch oracle: kind
// (blob, directory) by all 2^3 presence combinations of byte_count,
// blob_id, and blob_descriptor_id, by mode (null, 0, 4095). Every
// invalid cell refuses STRUCTURALLY with a literal code on both
// entries; no cell may panic.
func TestEntryNullabilityGrid(t *testing.T) {
	admitted := 0
	cells := 0
	for _, kind := range []string{"blob", "directory"} {
		for _, hasCount := range []bool{false, true} {
			for _, hasBlob := range []bool{false, true} {
				for _, hasDescriptor := range []bool{false, true} {
					for _, mode := range entryGridModes() {
						cells++
						want := entryBranchOracle(kind, hasCount, hasBlob, hasDescriptor)
						cell := gridEntryInput(kind, mode.value, hasCount, hasBlob, hasDescriptor)
						sealed, buildErr, panicked := buildGridCell(cell)
						if panicked != nil {
							t.Errorf("kind=%s count=%v blob=%v descriptor=%v mode=%s: build panicked: %v",
								kind, hasCount, hasBlob, hasDescriptor, mode.label, panicked)
							continue
						}
						if (buildErr == nil) != want {
							t.Errorf("kind=%s count=%v blob=%v descriptor=%v mode=%s: build admits = %v, oracle = %v (err=%v)",
								kind, hasCount, hasBlob, hasDescriptor, mode.label, buildErr == nil, want, buildErr)
							continue
						}
						if !want {
							direction := "for blob"
							if kind == "directory" {
								direction = "for directory"
							}
							requireRefusal(t, buildErr, "projected object entry[0]", direction)
							mutated := mutateEntryRow(t, kind, mode.value, hasCount, hasBlob, hasDescriptor)
							_, decodeErr, decodePanicked := decodeGridCell(mutated)
							if decodePanicked != nil {
								t.Errorf("kind=%s count=%v blob=%v descriptor=%v mode=%s: decode panicked: %v",
									kind, hasCount, hasBlob, hasDescriptor, mode.label, decodePanicked)
								continue
							}
							if decodeErr == nil {
								t.Errorf("kind=%s count=%v blob=%v descriptor=%v mode=%s: decode admits, oracle refuses",
									kind, hasCount, hasBlob, hasDescriptor, mode.label)
								continue
							}
							if kind == "blob" {
								tokens := []string{"projected object entry[1]", "misses a required member"}
								if missing := missingBlobOnly(hasCount, hasBlob, hasDescriptor); len(missing) == 1 {
									tokens = append(tokens, missing[0])
								}
								requireRefusal(t, decodeErr, tokens...)
							} else {
								// Several stray members refuse
								// through one unknown-member
								// gate whose reported name is
								// map-ordered: only the
								// single-stray cell pins it.
								tokens := []string{"projected object entry[0]", "carries unknown member"}
								if stray := presentBlobOnly(hasCount, hasBlob, hasDescriptor); len(stray) == 1 {
									tokens = append(tokens, stray[0])
								}
								requireRefusal(t, decodeErr, tokens...)
							}
							continue
						}
						admitted++
						decoded, decodeErr, decodePanicked := decodeGridCell(sealed)
						if decodePanicked != nil {
							t.Errorf("kind=%s mode=%s: fresh seal decode panicked: %v", kind, mode.label, decodePanicked)
							continue
						}
						if decodeErr != nil {
							t.Errorf("kind=%s mode=%s: fresh seal refused: %v", kind, mode.label, decodeErr)
							continue
						}
						if len(decoded.Entries) != 1 {
							t.Errorf("kind=%s mode=%s: entries = %d, want 1", kind, mode.label, len(decoded.Entries))
						}
					}
				}
			}
		}
	}
	// 2 kinds x 2^3 presence combinations x 3 modes = 48 cells; only
	// the two branch-exact combinations admit on each of 3 modes.
	if cells != 48 {
		t.Fatalf("grid cells = %d, want 48", cells)
	}
	if admitted != 6 {
		t.Fatalf("admitted cells = %d, want 6", admitted)
	}
}

// TestEntryGridNeverPanics sweeps the same full grid through both
// manifest entries with per-cell panic recovery: any panic fails the
// cell by name. A branch-guard bypass that dereferences an absent
// member (the reviewer's branchcombo plant) dies here as well as in
// the verdict grid above.
func TestEntryGridNeverPanics(t *testing.T) {
	for _, kind := range []string{"blob", "directory"} {
		for _, hasCount := range []bool{false, true} {
			for _, hasBlob := range []bool{false, true} {
				for _, hasDescriptor := range []bool{false, true} {
					for _, mode := range entryGridModes() {
						cell := gridEntryInput(kind, mode.value, hasCount, hasBlob, hasDescriptor)
						if _, _, panicked := buildGridCell(cell); panicked != nil {
							t.Errorf("kind=%s count=%v blob=%v descriptor=%v mode=%s: build panicked: %v",
								kind, hasCount, hasBlob, hasDescriptor, mode.label, panicked)
						}
						mutated := mutateEntryRow(t, kind, mode.value, hasCount, hasBlob, hasDescriptor)
						if _, _, panicked := decodeGridCell(mutated); panicked != nil {
							t.Errorf("kind=%s count=%v blob=%v descriptor=%v mode=%s: decode panicked: %v",
								kind, hasCount, hasBlob, hasDescriptor, mode.label, panicked)
						}
					}
				}
			}
		}
	}
}

// missingBlobOnly names the absent blob-only members for one grid
// cell.
func missingBlobOnly(hasCount, hasBlob, hasDescriptor bool) []string {
	var missing []string
	if !hasCount {
		missing = append(missing, "byte_count")
	}
	if !hasBlob {
		missing = append(missing, "blob_id")
	}
	if !hasDescriptor {
		missing = append(missing, "blob_descriptor_id")
	}
	return missing
}

// presentBlobOnly names the present blob-only members for one grid
// cell.
func presentBlobOnly(hasCount, hasBlob, hasDescriptor bool) []string {
	var present []string
	if hasCount {
		present = append(present, "byte_count")
	}
	if hasBlob {
		present = append(present, "blob_id")
	}
	if hasDescriptor {
		present = append(present, "blob_descriptor_id")
	}
	return present
}

// TestEntryBranchLiterals pins the build-side branch refusal
// literals for each blob-only member on each branch.
func TestEntryBranchLiterals(t *testing.T) {
	blob := validBlobEntryInput(1, "branch")
	blob.ByteCount = nil
	manifest := validManifestInput()
	manifest.Entries = []cloneplan.ProjectedEntryInput{blob}
	_, err := cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "projected object entry[0]", "byte_count is missing for blob")

	blob = validBlobEntryInput(1, "branch")
	blob.BlobID = nil
	manifest.Entries = []cloneplan.ProjectedEntryInput{blob}
	_, err = cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "projected object entry[0]", "blob_id is missing for blob")

	blob = validBlobEntryInput(1, "branch")
	blob.BlobDescriptorID = nil
	manifest.Entries = []cloneplan.ProjectedEntryInput{blob}
	_, err = cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "projected object entry[0]", "blob_descriptor_id is missing for blob")

	directory := validDirectoryEntryInput(1, "branch")
	directory.ByteCount = u64ptr(8)
	manifest.Entries = []cloneplan.ProjectedEntryInput{directory}
	_, err = cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "projected object entry[0]", "byte_count is present for directory")

	directory = validDirectoryEntryInput(1, "branch")
	directory.BlobID = strptr(fixtureDigest("stray"))
	manifest.Entries = []cloneplan.ProjectedEntryInput{directory}
	_, err = cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "projected object entry[0]", "blob_id is present for directory")

	directory = validDirectoryEntryInput(1, "branch")
	directory.BlobDescriptorID = strptr(fixtureDigest("stray"))
	manifest.Entries = []cloneplan.ProjectedEntryInput{directory}
	_, err = cloneplan.BuildProjectedObjectManifest(manifest)
	requireRefusal(t, err, "projected object entry[0]", "blob_descriptor_id is present for directory")
}
