package specpin_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specpin"
)

var (
	boardSectionReference  = regexp.MustCompile(`§(20|1[0-9]|[1-9])(?:\.[0-9A-Za-z]+)*(?:-(20|1[0-9]|[1-9])(?:\.[0-9A-Za-z]+)*)?`)
	boardAppendixReference = regexp.MustCompile(`(?i)\bAppend(?:ix|ices)\s+([A-D])(?:-([A-D]))?`)
)

func TestPinnedNormativeScopeEqualsBoardStoryUnion(t *testing.T) {
	manifest, err := specpin.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	boardRoot := filepath.Join("..", "..", ".task-board")
	got, storyCount := claimedNormativeUnion(t, boardRoot)
	if storyCount != 66 {
		t.Fatalf("board normative-scope survey covered %d Stories, want 66; update the survey assertion and pin for an intentional board change", storyCount)
	}
	want := []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"appendix-a", "appendix-b", "appendix-c", "appendix-d",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("board normative-scope union = %v, want verified 24-entry union %v", got, want)
	}
	if !reflect.DeepEqual(manifest.Source.NormativeScope, got) {
		t.Fatalf("pinned normative scope = %v, want board-derived union %v", manifest.Source.NormativeScope, got)
	}
}

func TestSectionInventoryMatchesPinnedV050HeadingDigest(t *testing.T) {
	inventory := specpin.SectionInventoryV050()
	if len(inventory) != 157 {
		t.Fatalf("v0.5.0 section inventory has %d identifiers, want 157", len(inventory))
	}
	digest := sha256.Sum256([]byte(strings.Join(inventory, "\n") + "\n"))
	if got := hex.EncodeToString(digest[:]); got != specpin.SectionInventorySHA256 {
		t.Fatalf("v0.5.0 section inventory digest = %s, want %s", got, specpin.SectionInventorySHA256)
	}
	if !specpin.IsSectionV050("10.1") {
		t.Fatal("real v0.5.0 Section 10.1 is absent from the immutable inventory")
	}
	if specpin.IsSectionV050("10.999") {
		t.Fatal("nonexistent Section 10.999 is present in the immutable inventory")
	}

	inventory[0] = "forged"
	if !specpin.IsSectionV050("1") {
		t.Fatal("caller mutation leaked into the immutable section inventory")
	}
}

func claimedNormativeUnion(t *testing.T, boardRoot string) ([]string, int) {
	t.Helper()

	claimed := make(map[string]struct{})
	storyCount := 0
	err := filepath.WalkDir(boardRoot, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "README.md" || !strings.HasPrefix(filepath.Base(filepath.Dir(filename)), "STORY-") {
			return nil
		}

		value, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		lines := strings.Split(string(value), "\n")
		matchedLines := 0
		for _, line := range lines {
			if !strings.HasPrefix(line, "Normative scope:") {
				continue
			}
			matchedLines++
			if err := addClaimedScopes(claimed, line); err != nil {
				return fmt.Errorf("%s: %w", filename, err)
			}
		}
		if matchedLines > 1 {
			return fmt.Errorf("%s has %d Normative scope lines, want at most one", filename, matchedLines)
		}
		if matchedLines == 1 {
			storyCount++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("survey board normative scopes: %v", err)
	}

	result := make([]string, 0, len(claimed))
	for scope := range claimed {
		result = append(result, scope)
	}
	sort.Slice(result, func(left, right int) bool {
		leftNumber, leftErr := strconv.Atoi(result[left])
		rightNumber, rightErr := strconv.Atoi(result[right])
		if leftErr == nil && rightErr == nil {
			return leftNumber < rightNumber
		}
		if leftErr == nil {
			return true
		}
		if rightErr == nil {
			return false
		}
		return result[left] < result[right]
	})
	return result, storyCount
}

func addClaimedScopes(claimed map[string]struct{}, line string) error {
	for _, match := range boardSectionReference.FindAllStringSubmatch(line, -1) {
		start, _ := strconv.Atoi(match[1])
		end := start
		if match[2] != "" {
			end, _ = strconv.Atoi(match[2])
		}
		if end < start {
			return fmt.Errorf("descending section range %q", match[0])
		}
		for section := start; section <= end; section++ {
			claimed[strconv.Itoa(section)] = struct{}{}
		}
	}
	for _, match := range boardAppendixReference.FindAllStringSubmatch(line, -1) {
		start := strings.ToLower(match[1])[0]
		end := start
		if match[2] != "" {
			end = strings.ToLower(match[2])[0]
		}
		if end < start {
			return fmt.Errorf("descending appendix range %q", match[0])
		}
		for appendix := start; appendix <= end; appendix++ {
			claimed["appendix-"+string(appendix)] = struct{}{}
		}
	}
	return nil
}
