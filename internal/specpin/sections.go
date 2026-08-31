package specpin

import "strings"

const (
	// SectionInventorySHA256 pins the newline-delimited section identifiers
	// extracted from the exact v0.5.0 SPEC.md identified by DocumentSHA256.
	SectionInventorySHA256 = "a71f012f8c8a39baed38e70ee0175abd16bef712a1c3389ad0242f7b5e53ebeb"
	sectionInventoryV050   = `1
1.1
1.2
1.3
1.4
1.5
1.6
2
2.1
2.2
2.3
2.4
3
3.1
3.2
3.3
4
4.A
4.B
4.C
4.D
4.E
4.1
4.2
4.3
4.4
5
5.1
5.2
5.3
5.4
5.5
5.6
5.7
6
6.1
6.2
6.3
6.4
6.5
7
7.1
7.2
7.3
7.4
7.5
7.6
7.7
7.A
7.8
7.9
8
8.1
8.2
8.3
8.4
9
9.1
9.2
9.3
9.4
9.5
10
10.1
10.2
10.3
10.4
10.5
10.6
10.7
10.8
10.8.1
10.8.2
10.8.3
10.8.4
10.8.5
11
11.1
11.2
11.3
11.4
11.5
11.6
11.7
11.8
11.9
12
12.1
12.2
12.3
12.4
12.5
12.6
13
13.1
13.2
13.3
13.4
13.5
13.6
13.7
13.8
13.9
13.10
13.11
13.12
13.13
13.14
13.14.1
13.14.2
13.14.3
13.14.4
13.14.5
13.15
14
14.1
14.2
14.3
14.4
14.5
14.6
15
15.1
15.2
15.3
16
16.1
16.2
16.3
16.4
16.5
16.6
16.7
17
17.1
17.2
17.3
17.4
17.5
18
18.1
18.2
18.3
18.4
19
19.1
19.2
19.3
19.4
19.5
20
20.1
20.2
appendix-a
appendix-b
appendix-c
appendix-d
`
)

var sectionInventorySetV050 = func() map[string]struct{} {
	sections := strings.Fields(sectionInventoryV050)
	result := make(map[string]struct{}, len(sections))
	for _, section := range sections {
		result[section] = struct{}{}
	}
	return result
}()

// SectionInventoryV050 returns an isolated ordered copy of every real section
// identifier extracted from the pinned v0.5.0 document headings.
func SectionInventoryV050() []string {
	return strings.Fields(sectionInventoryV050)
}

// IsSectionV050 reports whether identifier is a real heading identifier in the
// exact pinned v0.5.0 document. Callers must canonicalize user input first.
func IsSectionV050(identifier string) bool {
	_, ok := sectionInventorySetV050[identifier]
	return ok
}
