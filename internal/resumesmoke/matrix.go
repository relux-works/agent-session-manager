package resumesmoke

// This file reads the Section 8.4 native-resume direction for one
// exact probed tuple: the first cell of the provider/platform row,
// refined by the Appendix B version gates. The matrix is evidence
// labels, never availability promises: the fail-closed direction is
// that only CellAvailable may run a resume plan, and the row-complete
// derivation test pins every value against the pinned specification
// text rather than trusting this table.
//
// The version refinements come from the rows and gates themselves:
// the Muse macOS arm64 row accepts only the probed 0.1.0, and
// Appendix B leaves every newer Muse behavior unsettled with
// enabled false, so a non-0.1.0 build on that tuple is unknown,
// refused outright like the row's own unknown Windows cell. Qwen
// direct is unsupported on every platform — the row names task-board
// prompt mode only — and any provider or platform outside the
// Section 8.4 rows is unknown, disabled until a tuple-specific probe
// and acceptance record exist. Unknown is never rewritten as
// unsupported: the two refuse through distinct cells and verdicts.

// Cell is one Section 8.1 matrix label for the native-resume
// direction: available, conditional, unsupported, or unknown.
type Cell string

// The four Section 8.1 labels.
const (
	CellAvailable   Cell = "A"
	CellConditional Cell = "C"
	CellUnsupported Cell = "U"
	CellUnknown     Cell = "?"
)

// musePinnedVersion is the exact Muse build the Section 8.4 macOS
// arm64 row accepts for native resume. Every other Muse version on
// that tuple is Appendix B unsettled behavior and reads unknown.
const musePinnedVersion = "0.1.0"

// ResumeCell reads the Section 8.4 native-resume cell for one exact
// tuple and cites the row or gate that decided it. The citation is a
// stable section-8.4 row reference, never a capability claim.
func ResumeCell(providerID, providerVersion, platform, architecture string) (Cell, string) {
	switch providerID {
	case "codex":
		return platformCell(platform, "codex", CellAvailable, CellAvailable, CellAvailable, CellAvailable)
	case "claude":
		return platformCell(platform, "claude", CellAvailable, CellAvailable, CellAvailable, CellConditional)
	case "gemini":
		return platformCell(platform, "gemini", CellAvailable, CellAvailable, CellConditional, CellAvailable)
	case "muse":
		return museCell(providerVersion, platform, architecture)
	case "antigravity":
		return platformCell(platform, "antigravity", CellConditional, CellConditional, CellConditional, CellConditional)
	case "pi":
		return platformCell(platform, "pi", CellAvailable, CellConditional, CellConditional, CellConditional)
	case "qwen":
		// The Qwen row covers any target platform: direct native
		// resume is unsupported everywhere, and only task-board
		// prompt mode may be available where its own probe says so.
		return CellUnsupported, "section-8.4:qwen-direct"
	default:
		// Future plugins and unknown providers start disabled: every
		// cell is unknown until a tuple-specific probe and
		// acceptance record exist.
		return CellUnknown, "section-8.4:future-plugin"
	}
}

// platformCell reads one provider row in Section 8.4 platform order:
// macOS, Linux, WSL2, native Windows. WSL2 and native Windows are
// distinct rows; one never stands for the other. A platform outside
// the registry is unknown, never a default into a named row.
func platformCell(platform, provider string, macos, linux, wsl2, windows Cell) (Cell, string) {
	switch platform {
	case "macos":
		return macos, "section-8.4:" + provider + "/macos"
	case "linux":
		return linux, "section-8.4:" + provider + "/linux"
	case "wsl2":
		return wsl2, "section-8.4:" + provider + "/wsl2"
	case "windows":
		return windows, "section-8.4:" + provider + "/windows"
	default:
		return CellUnknown, "section-8.4:platform-unknown"
	}
}

// museCell reads the five Muse rows: the macOS arm64 row accepts
// only the pinned probe version, the macOS amd64, Linux, and WSL2
// rows are conditional, and native Windows is unknown. A non-pinned
// version on macOS arm64 is unknown rather than conditional,
// because Appendix B leaves newer Muse resume behavior unsettled
// with enabled false instead of naming an acceptance gate the smoke
// could await.
func museCell(version, platform, architecture string) (Cell, string) {
	switch platform {
	case "macos":
		switch architecture {
		case "arm64":
			if version == musePinnedVersion {
				return CellAvailable, "section-8.4:muse/macos-arm64"
			}
			return CellUnknown, "section-8.4:muse/macos-arm64+appendix-B"
		case "amd64":
			return CellConditional, "section-8.4:muse/macos-amd64"
		default:
			return CellUnknown, "section-8.4:platform-unknown"
		}
	case "linux":
		return CellConditional, "section-8.4:muse/linux"
	case "wsl2":
		return CellConditional, "section-8.4:muse/wsl2"
	case "windows":
		return CellUnknown, "section-8.4:muse/windows"
	default:
		return CellUnknown, "section-8.4:platform-unknown"
	}
}
