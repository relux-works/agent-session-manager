package clonesnap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file owns the source-race record: the canonical digest over
// every enumerated store member that the pre and post capture
// measurements compare. Included-class members contribute their key,
// size, and content hash; excluded-class members contribute key and
// size only, because their bytes are never opened; directories
// contribute their keys, so an appended record is a detected
// mutation. Size equality is never proof: the comparison is digest
// equality over content hashes through the landed CheckDigestsEqual.

// sourceRecord is one canonical source listing: sorted unique lines,
// one per enumerated member.
type sourceRecord struct {
	lines []string
}

func recordFileLine(key string, size uint64, content string) string {
	return key + "\x00" + string(memberFile) + "\x00" + strconv.FormatUint(size, 10) + "\x00" + content
}

func recordDirLine(key string) string {
	return key + "\x00" + string(memberDir) + "\x00" + "0" + "\x00" + "-"
}

func recordOtherLine(key string) string {
	return key + "\x00" + string(memberOther) + "\x00" + "0" + "\x00" + "-"
}

// addFile records one regular member: content is the hex content
// hash for included members and "-" for excluded members, whose
// bytes are never read.
func (record *sourceRecord) addFile(key string, size uint64, content string) {
	record.lines = append(record.lines, recordFileLine(key, size, content))
}

func (record *sourceRecord) addDir(key string) {
	record.lines = append(record.lines, recordDirLine(key))
}

func (record *sourceRecord) addOther(key string) {
	record.lines = append(record.lines, recordOtherLine(key))
}

// digest seals the record: the lines sort and join deterministically,
// so identical sources seal identical digests.
func (record *sourceRecord) digest() scalar.Digest {
	lines := append([]string(nil), record.lines...)
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	digest, err := scalar.ParseDigest("sha256:" + hex.EncodeToString(sum[:]))
	if err != nil {
		panic(fmt.Sprintf("clonesnap: source record digest is not a digest: %v", err))
	}
	return digest
}

// contentHash is the hex content hash one record line carries for an
// included member.
func contentHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// checkSourceRace compares the pre and post source digests through
// the landed equality gate. Equal sizes with differing bytes refuse:
// the gate compares content digests, never sizes. The refusal names
// the first differing member alongside the two digests; it never
// carries source bytes.
func checkSourceRace(pre, post scalar.Digest, preLines, postLines []string) error {
	if err := clonebundle.CheckDigestsEqual(pre, post); err != nil {
		return invalid("source mutated during capture: member %q changed: %v", firstChangedMember(preLines, postLines), err)
	}
	return nil
}

// firstChangedMember names one member whose record line differs
// between the pre and post listings: an appended, removed, resized,
// retyped, or content-changed member. The listings compared equal as
// sets only when the digests agree, so a mismatch always names a
// member; the fallback names the record, never nothing.
func firstChangedMember(preLines, postLines []string) string {
	pre := make(map[string]bool, len(preLines))
	for _, line := range preLines {
		pre[line] = true
	}
	post := make(map[string]bool, len(postLines))
	for _, line := range postLines {
		post[line] = true
	}
	for _, line := range postLines {
		if !pre[line] {
			if key, _, found := strings.Cut(line, "\x00"); found {
				return key
			}
			return "source record"
		}
	}
	for _, line := range preLines {
		if !post[line] {
			if key, _, found := strings.Cut(line, "\x00"); found {
				return key
			}
			return "source record"
		}
	}
	return "source record"
}
