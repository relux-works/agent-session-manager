package provider

import (
	"errors"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// fakeFile is the scripted target behind one canonical path.
type fakeFile struct {
	content []byte
	uid     uint32
	regular bool
}

// fakeSystem scripts every System seam. Absent table entries are read
// failures, never silent absences: Inspect and ReadFile fail on unknown
// canonical paths, and Canonicalize fails on scripted paths.
type fakeSystem struct {
	entries    map[string][]string
	readDirErr map[string]error
	canon      map[string]string
	canonErr   map[string]error
	files      map[string]fakeFile
	inspectErr map[string]error
	contentErr map[string]error
	pathDirs   []string
}

func newFakeSystem() *fakeSystem {
	return &fakeSystem{
		entries:    map[string][]string{},
		readDirErr: map[string]error{},
		canon:      map[string]string{},
		canonErr:   map[string]error{},
		files:      map[string]fakeFile{},
		inspectErr: map[string]error{},
		contentErr: map[string]error{},
	}
}

// addFile registers dir/name as an executable target. Without linkTo, the
// source path is its own canonical target.
func (fake *fakeSystem) addFile(dir, name string, content []byte, uid uint32) string {
	return fake.addLinkedFile(dir, name, "", content, uid, true)
}

// addLinkedFile registers dir/name resolving to linkTo (empty means
// identity). The target is registered as a regular file unless regular is
// false.
func (fake *fakeSystem) addLinkedFile(dir, name, linkTo string, content []byte, uid uint32, regular bool) string {
	source := dir + "/" + name
	target := linkTo
	if target == "" {
		target = source
	}
	fake.entries[dir] = append(fake.entries[dir], name)
	fake.canon[source] = target
	fake.files[target] = fakeFile{content: content, uid: uid, regular: regular}
	return source
}

func (fake *fakeSystem) ReadDir(dir string) ([]string, error) {
	if err, ok := fake.readDirErr[dir]; ok {
		return nil, err
	}
	names, ok := fake.entries[dir]
	if !ok {
		return nil, errors.New("fake: unknown directory " + dir)
	}
	// Reverse the insertion order so tests prove Discover sorts rather
	// than inheriting filesystem order.
	reversed := append([]string(nil), names...)
	sort.Sort(sort.Reverse(sort.StringSlice(reversed)))
	return reversed, nil
}

func (fake *fakeSystem) Canonicalize(path string) (string, error) {
	if err, ok := fake.canonErr[path]; ok {
		return "", err
	}
	if target, ok := fake.canon[path]; ok {
		return target, nil
	}
	return "", errors.New("fake: unresolvable path " + path)
}

func (fake *fakeSystem) Inspect(canonical string) (FileInfo, error) {
	if err, ok := fake.inspectErr[canonical]; ok {
		return FileInfo{}, err
	}
	file, ok := fake.files[canonical]
	if !ok {
		return FileInfo{}, errors.New("fake: unknown target " + canonical)
	}
	return FileInfo{IsRegular: file.regular, UID: file.uid}, nil
}

func (fake *fakeSystem) ReadFile(canonical string) ([]byte, error) {
	if err, ok := fake.contentErr[canonical]; ok {
		return nil, err
	}
	file, ok := fake.files[canonical]
	if !ok {
		return nil, errors.New("fake: unknown target " + canonical)
	}
	return append([]byte(nil), file.content...), nil
}

func (fake *fakeSystem) PathDirs() []string {
	return append([]string(nil), fake.pathDirs...)
}

const (
	fakeUID      uint32 = 1000
	foreignUID   uint32 = 2000
	adminUID     uint32 = 7
	fakePlatform        = scalar.PlatformLinux
)

func fakeOwner() OwnerPolicy {
	return OwnerPolicy{OperatorUID: fakeUID}
}

func fakeConfig(dirs ...string) Config {
	return Config{Platform: fakePlatform, PluginDirs: dirs}
}

func mustTimestamp(test testHelper, value string) scalar.Timestamp {
	test.Helper()
	parsed, err := scalar.ParseTimestamp(value)
	if err != nil {
		test.Fatalf("ParseTimestamp(%q): %v", value, err)
	}
	return parsed
}

// errorCode extracts the package Code from err, failing the test when err
// is not a provider refusal.
func errorCode(test testHelper, err error) string {
	test.Helper()
	var refusal Error
	if !errors.As(err, &refusal) {
		test.Fatalf("error %v is not a provider Error", err)
	}
	return refusal.Code()
}

type testHelper interface {
	Helper()
	Fatalf(format string, args ...any)
}
