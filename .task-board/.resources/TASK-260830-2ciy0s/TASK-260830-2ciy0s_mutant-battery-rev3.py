#!/usr/bin/env python3
"""Round-2 narrowing/deletion mutant battery for TASK-260830-2ciy0s.

Usage: python3 /tmp/mutant_battery.py N1 N2 ...   (runs those mutants)
Each mutant: backup file, assert exact one-occurrence replacement,
run the named test, record KILLED (exit != 0 with the named test failing)
or SURVIVED (exit 0), restore the file byte-identical.
"""
import subprocess
import sys

MUTANTS = [
    # ---------------- NARROWING: boundary moves admitting exactly one member
    dict(id="N1", file="internal/secprim/path.go",
         old='return strings.Contains(unwrapPercentEncoding(member), "%2e")',
         new='return strings.Contains(strings.ToLower(member), "%2e")',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="double-encoded_dot",
         narrows="admits exactly doubly-encoded dots (%252e)"),
    dict(id="N2", file="internal/secprim/path.go",
         old='return strings.Contains(unwrapPercentEncoding(member), "%c0%af")',
         new='return strings.Contains(strings.ToLower(member), "%c0%af")',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="double-encoded_overlong",
         narrows="admits exactly doubly-encoded overlong separators"),
    dict(id="N3", file="internal/secprim/path.go",
         old='if strings.Contains(segment, ":") {',
         new='if strings.Contains(segment, "::") {',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="windows_ads",
         narrows="admits exactly single-colon alternate streams"),
    dict(id="N4", file="internal/secprim/path.go",
         old='if strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {',
         new='if strings.HasSuffix(segment, ".") {',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="windows_trailing_space",
         narrows="admits exactly trailing-space segments"),
    dict(id="N5", file="internal/secprim/path.go",
         old='for _, segment := range strings.Split(member, "/") {',
         new='for _, segment := range strings.Split(member, "/")[:1] {',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="windows_lpt",
         narrows="admits exactly non-first-segment violations (nested device names)"),
    dict(id="N6", file="internal/secprim/path.go",
         old='if guard.checkManaged && !guard.managed[member] {',
         new='if guard.checkManaged && !guard.managed[strings.ToLower(member)] {',
         pkg="./internal/secprim/", run="TestGuardManagedSet",
         kill="TestGuardManagedSet",
         narrows="admits exactly case variants of managed members"),
    dict(id="N7", file="internal/secprim/path.go",
         old='''	case mode&os.ModeDevice != 0,
		mode&os.ModeNamedPipe != 0,''',
         new='''	case mode&os.ModeDevice != 0,''',
         pkg="./internal/secprim/", run="TestCheckRegularTarget",
         kill="TestCheckRegularTarget",
         narrows="admits exactly FIFOs (one disjunct of the special-file arm)"),
    dict(id="N7b", file="internal/secprim/path.go",
         old='''	case mode&os.ModeDevice != 0,
		mode&os.ModeNamedPipe != 0,
		mode&os.ModeSocket != 0,
		mode&os.ModeCharDevice != 0,
		!mode.IsRegular():''',
         new='''	case mode&os.ModeDevice != 0,
		mode&os.ModeSocket != 0,
		mode&os.ModeCharDevice != 0:''',
         pkg="./internal/secprim/", run="TestCheckRegularTarget",
         kill="TestCheckRegularTarget",
         narrows="narrowing, two-arm: drops NamedPipe disjunct plus the irregular catch-all, admits exactly FIFOs"),
    dict(id="N8", file="internal/secprim/render.go",
         old='case character >= 0x202a && character <= 0x202e,',
         new='case character >= 0x202b && character <= 0x202e,',
         pkg="./internal/secprim/", run="TestEscapeForTerminal",
         kill="TestEscapeForTerminalOutputBytes",
         narrows="admits exactly U+202A"),
    dict(id="N9", file="internal/secprim/render.go",
         old='character >= 0x2066 && character <= 0x2069,',
         new='character >= 0x2066 && character <= 0x2068,',
         pkg="./internal/secprim/", run="TestEscapeForTerminal",
         kill="TestEscapeForTerminalOutputBytes",
         narrows="admits exactly U+2069"),
    dict(id="N10", file="internal/secprim/render.go",
         old='case character < 0x20 || character == 0x7f || character >= 0x80 && character <= 0x9f:',
         new='case character < 0x20 || character == 0x7f || character >= 0x80 && character <= 0x9e:',
         pkg="./internal/secprim/", run="TestEscapeForTerminalOutputBytes",
         kill="TestEscapeForTerminalOutputBytes",
         narrows="admits exactly U+009F"),
    dict(id="N11", file="internal/secprim/render.go",
         old='case character < 0x20 || character == 0x7f || character >= 0x80 && character <= 0x9f:',
         new='case character < 0x1f || character == 0x7f || character >= 0x80 && character <= 0x9f:',
         pkg="./internal/secprim/", run="TestEscapeForTerminalOutputBytes",
         kill="TestEscapeForTerminalOutputBytes",
         narrows="admits exactly U+001F"),
    dict(id="N12", file="internal/secprim/argv.go",
         old='''		case strings.ContainsRune(element, 0):
			return failArgv("argv NUL", position)''',
         new='''		case index == 0 && strings.ContainsRune(element, 0):
			return failArgv("argv NUL", position)''',
         pkg="./internal/secprim/", run="TestCheckArgvRefuses",
         kill="nul_argument",
         narrows="admits exactly NUL bytes past argv[0]"),
    dict(id="N13", file="internal/secprim/argv.go",
         old='case !utf8.ValidString(element):',
         new='case index == 0 && !utf8.ValidString(element):',
         pkg="./internal/secprim/", run="TestCheckArgvRefuses",
         kill="invalid_utf8",
         narrows="admits exactly invalid UTF-8 past argv[0]"),
    dict(id="N14", file="internal/secprim/argv.go",
         old='case element == "":',
         new='case index == 0 && element == "":',
         pkg="./internal/secprim/", run="TestCheckArgvRefuses",
         kill="empty_element",
         narrows="admits exactly empty elements past argv[0]"),
    dict(id="N15", file="internal/secprim/env.go",
         old='if len(name) < 1 || len(name) > 128 {',
         new='if len(name) < 1 || len(name) > 129 {',
         pkg="./internal/secprim/", run="TestIsEnvName",
         kill="TestIsEnvName",
         narrows="admits exactly 129-rune names"),
    dict(id="N16", file="internal/secprim/env.go",
         old="index > 0 && char >= '0' && char <= '9'",
         new="char >= '0' && char <= '9'",
         pkg="./internal/secprim/", run="TestIsEnvName",
         kill="TestIsEnvName",
         narrows="admits exactly leading-digit names (9LIVES)"),
    dict(id="N17", file="internal/secprim/redact.go",
         old='minSecretRunes        = 8',
         new='minSecretRunes        = 9',
         pkg="./internal/secprim/", run="TestRedactCorpusValues",
         kill="TestRedactCorpusValues",
         narrows="admits exactly 8-rune caller-known secrets"),
    dict(id="N18", file="internal/secprim/redact.go",
         old='privateKeyBeginSuffix = "PRIVATE KEY-----"',
         new='privateKeyBeginSuffix = "-----"',
         pkg="./internal/secprim/", run="TestRedactPrivateKeyBlocks",
         kill="TestRedactPrivateKeyBlocks",
         narrows="WIDENING: redacts every BEGIN block, certificates included"),
    dict(id="N19", file="internal/cliresult/output.go",
         old='return status, writeLine(emitter.streams.Stdout, terminalLine(outcome.Rendered))',
         new='return status, writeLine(emitter.streams.Stdout, outcome.Rendered)',
         pkg="./internal/cliresult/", run="TestTextSuccessEmissionNeutralizes",
         kill="TestTextSuccessEmissionNeutralizes",
         narrows="unwires exactly the success-Emit stdout site (R27)"),
    dict(id="N20", file="internal/cliresult/output.go",
         old='if err := writeLine(emitter.streams.Stderr, terminalLine(line)); err != nil {',
         new='if err := writeLine(emitter.streams.Stderr, line); err != nil {',
         pkg="./internal/cliresult/", run="TestProgressEmissionNeutralizes",
         kill="TestProgressEmissionNeutralizes",
         narrows="unwires exactly the Progress site (R23)"),
    dict(id="N21", file="internal/cliresult/output.go",
         old='return fmt.Errorf("%w: %q", ErrPromptForbidden, terminalLine(line))',
         new='return fmt.Errorf("%w: %q", ErrPromptForbidden, line)',
         pkg="./internal/cliresult/", run="TestPromptRefusalCarriesNeutralizedText",
         kill="TestPromptRefusalCarriesNeutralizedText",
         narrows="unwires exactly the Prompt-refusal text (R24)"),
    dict(id="N22", file="internal/provhost/runner.go",
         old='if runner.Env != nil {',
         new='if len(runner.Env) > 0 {',
         pkg="./internal/provhost/", run="TestExecRunnerEmptyEnvIsDenyAll",
         kill="TestExecRunnerEmptyEnvIsDenyAll",
         narrows="restores full inheritance for exactly the empty allowlist (R26)"),
    dict(id="N23", file="internal/secprim/open_member_unix.go",
         old='''	if errors.Is(err, unix.ENOTDIR) {
		var status unix.Stat_t
		if fstatErr := unix.Fstatat(parent, segment, &status, unix.AT_SYMLINK_NOFOLLOW); fstatErr == nil {
			if status.Mode&unix.S_IFMT == unix.S_IFLNK {
				return errCommitSymlink
			}
		}
	}''',
         new='',
         pkg="./internal/secprim/", run="TestGuardOpenRefusesSymlinkedParent",
         kill="TestGuardOpenRefusesSymlinkedParent",
         narrows="ELOOP-only classification: Darwin ENOTDIR escapes misreport as open-failed"),
    dict(id="N24", file="internal/secprim/path.go",
         old='''\tcase info.IsDir():\n\t\treturn failPath("target directory", info.Name())\n''',
         new='',
         pkg="./internal/secprim/", run="TestCheckRegularTarget",
         kill="TestCheckRegularTarget",
         narrows="drops the directory arm alone: directories fall to the irregular catch-all and refuse as target special file, so the exact-message test kills"),
    dict(id="N25", file="internal/secprim/argv.go",
         old='if len(argv) == 0 {',
         new='if len(argv) < 0 {',
         pkg="./internal/secprim/", run="TestCheckArgvRefuses",
         kill="empty_vector",
         narrows="admits exactly empty vectors"),
    dict(id="N26", file="internal/secprim/redact.go",
         old='for _, sep := range []string{"=", ":"} {',
         new='for _, sep := range []string{"="} {',
         pkg="./internal/secprim/", run="TestRedactSensitivePairs",
         kill="colon",
         narrows="admits exactly colon-shaped sensitive pairs"),
    dict(id="N24b", file="internal/secprim/path.go",
         old='''	case info.IsDir():
		return failPath("target directory", info.Name())
	case mode&os.ModeDevice != 0,
		mode&os.ModeNamedPipe != 0,
		mode&os.ModeSocket != 0,
		mode&os.ModeCharDevice != 0,
		!mode.IsRegular():''',
         new='''	case mode&os.ModeDevice != 0,
		mode&os.ModeNamedPipe != 0,
		mode&os.ModeSocket != 0,
		mode&os.ModeCharDevice != 0:''',
         pkg="./internal/secprim/", run="TestCheckRegularTarget",
         kill="TestCheckRegularTarget",
         narrows="narrowing, two-arm: drops directory arm plus the irregular catch-all, admits exactly directories"),
    dict(id="N27", file="internal/secprim/render.go",
         old='case character < 0x20 || character == 0x7f || character >= 0x80 && character <= 0x9f:',
         new='case character < 0x20 || character >= 0x80 && character <= 0x9f:',
         pkg="./internal/secprim/", run="TestEscapeForTerminal",
         kill="del",
         narrows="admits exactly DEL"),
    dict(id="N28", file="internal/secprim/nofollow.go",
	old='return os.SameFile(handleInfo, pathInfo)',
	new='return handleInfo.IsDir() && pathInfo.IsDir()',
	pkg="./internal/secprim/", run="TestGuardOpenRefusesForeignRootHandle",
	kill="TestGuardOpenRefusesForeignRootHandle",
	narrows="admits exactly foreign directory handles (binding checks kind, not identity)"),
    dict(id="N29", file="internal/secprim/open_member_unix.go",
	old='''		if err != nil {
			classified := classifyCommitErr(current, segment, err)
			if previous >= 0 {
				_ = unix.Close(previous)
			}
			return nil, classified
		}''',
	new='''		if err != nil {
			if previous >= 0 {
				_ = unix.Close(previous)
			}
			return nil, classifyCommitErr(current, segment, err)
		}''',
	pkg="./internal/secprim/", run="TestGuardOpenRefusesDeepSymlinkedParent",
	kill="TestGuardOpenRefusesDeepSymlinkedParent",
	narrows="classify-after-close at the loop site: admits exactly the misreported rule for non-first intermediates"),
    dict(id="N30", file="internal/secprim/env.go",
	old='''		if strings.ContainsRune(value, 0) {
			return nil, failEnv("env value NUL", name)
		}''',
	new='''		if strings.IndexRune(value, 0) == 0 {
			return nil, failEnv("env value NUL", name)
		}''',
	pkg="./internal/secprim/", run="TestBuildEnvRefuses",
	kill="inherited_NUL",
	narrows="admits exactly NUL past index 0 in allowlist values"),
    dict(id="N31", file="internal/secprim/env.go",
	old='''		if strings.ContainsRune(value, 0) {
			return nil, failEnv("env value NUL", key)
		}''',
	new='''		if strings.IndexRune(value, 0) == 0 {
			return nil, failEnv("env value NUL", key)
		}''',
	pkg="./internal/secprim/", run="TestBuildEnvRefuses",
	kill="literal_NUL",
	narrows="admits exactly NUL past index 0 in literal values"),
    dict(id="N32", file="internal/secprim/env.go",
	old='''		if !utf8.ValidString(value) {
			return nil, failEnv("env value encoding", name)
		}''',
	new='''		if !utf8.ValidString(value) && len(value) > 4 {
			return nil, failEnv("env value encoding", name)
		}''',
	pkg="./internal/secprim/", run="TestBuildEnvRefuses",
	kill="inherited_encoding",
	narrows="admits exactly short invalid-UTF-8 allowlist values"),
    dict(id="N33", file="internal/secprim/env.go",
	old='''		if !utf8.ValidString(value) {
			return nil, failEnv("env value encoding", key)
		}''',
	new='''		if !utf8.ValidString(value) && len(value) > 4 {
			return nil, failEnv("env value encoding", key)
		}''',
	pkg="./internal/secprim/", run="TestBuildEnvRefuses",
	kill="literal_encoding",
	narrows="admits exactly short invalid-UTF-8 literal values"),
    dict(id="N34", file="internal/secprim/env.go",
	old='if seen[name] {',
	new='if seen[name] && len(name) > 3 {',
	pkg="./internal/secprim/", run="TestBuildEnvRefuses",
	kill="duplicate_allowed",
	narrows="admits exactly short duplicate allowlist names"),
    dict(id="N35", file="internal/secprim/env.go",
	old='if seen[key] {',
	new='if seen[key] && len(key) > 3 {',
	pkg="./internal/secprim/", run="TestBuildEnvRefuses",
	kill="literal_collides",
	narrows="admits exactly short colliding literal keys"),
    dict(id="N36", file="internal/scalar/path.go",
         old='if segment == "" || segment == "." || segment == ".." {',
         new='if segment == "" || segment == "." {',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="nested_parent",
         narrows="admits exactly parent segments through the shared grammar"),
    dict(id="N38", file="internal/secprim/path.go",
         old='if strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {',
         new='if strings.HasSuffix(segment, " ") {',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="windows_trailing_dot",
         narrows="admits exactly trailing-dot segments"),
    dict(id="N39", file="internal/secprim/argv.go",
         old='case element == "":',
         new='case index != 0 && element == "":',
         pkg="./internal/secprim/", run="TestCheckArgvRefuses",
         kill="empty_executable",
         narrows="admits exactly empty executables"),
    dict(id="D21", file="internal/secprim/path.go",
         old='if _, err := scalar.ParseAbsolutePath(platform, root); err != nil {',
         new='if _, err := scalar.ParseAbsolutePath(platform, root); err != nil && false {',
         pkg="./internal/secprim/", run="TestGuardResolve",
         kill="TestGuardResolve",
         narrows="DELETION-shaped: deadens the guard-root absoluteness delegation (relative roots admitted)"),
    dict(id="N37", file="internal/secprim/path.go",
         old='case mode&os.ModeSymlink != 0:',
         new='case mode == os.ModeSymlink:',
         pkg="./internal/secprim/", run="TestCheckRegularTarget",
         kill="TestCheckRegularTarget",
         narrows="exact-equality symlink match: admits symlink-combined modes to the later arms"),
    # ---------------- DELETION: existence probes, one whole arm or flag
    dict(id="D1", file="internal/secprim/env.go",
         old='if seen[name] {',
         new='if seen[name] && false {',
         pkg="./internal/secprim/", run="TestBuildEnvRefuses",
         kill="duplicate_allowed",
         narrows="DELETION: drops the duplicate-allowlist arm"),
    dict(id="D2", file="internal/secprim/env.go",
         old='if seen[key] {',
         new='if seen[key] && false {',
         pkg="./internal/secprim/", run="TestBuildEnvRefuses",
         kill="literal_collides",
         narrows="DELETION: drops the literal-collision arm"),
    dict(id="D3", file="internal/secprim/redact.go",
         old='"oauth_token": true, "passphrase": true, "password": true,',
         new='"oauth_token": true, "passphrase": true,',
         pkg="./internal/secprim/", run="TestSensitiveKeyListIsReviewed",
         kill="TestSensitiveKeyListIsReviewed",
         narrows="DELETION: drops one scrubber-table entry"),
    dict(id="D5", file="internal/secprim/path.go",
         old='if guard.checkManaged && !guard.managed[member] {',
         new='if false && guard.checkManaged && !guard.managed[member] {',
         pkg="./internal/secprim/", run="TestGuardManagedSet",
         kill="TestGuardManagedSet",
         narrows="DELETION: drops the managed-set arm"),
    dict(id="D6", file="internal/secprim/path.go",
         old='if containsEncodedDot(member) {',
         new='if containsEncodedDot(member) && false {',
         pkg="./internal/secprim/", run="TestCheckMemberPathRefuses",
         kill="encoded_dot",
         narrows="DELETION: drops the encoded-dot arm"),
    dict(id="D10", file="internal/cliresult/output.go",
         old='''func (emitter *Emitter) Log(line string) error {
	return writeLine(emitter.streams.Stderr, terminalLine(line))''',
         new='''func (emitter *Emitter) Log(line string) error {
	return writeLine(emitter.streams.Stderr, line)''',
         pkg="./internal/cliresult/", run="TestTextEmissionNeutralizesHostileContent",
         kill="TestTextEmissionNeutralizesHostileContent",
         narrows="DELETION: drops the Log-only wire"),
    dict(id="D11", file="internal/provhost/runner.go",
         old='''	if runner.Env != nil {
		command.Env = runner.Env
	}''',
         new='',
         pkg="./internal/provhost/", run="TestExecRunnerDeliversExactlyTheBuiltEnv",
         kill="TestExecRunnerDeliversExactlyTheBuiltEnv",
         narrows="DELETION: ignores the runner allowlist (silent inheritance)"),
    dict(id="D12", file="internal/secprim/open_member_unix.go",
         old='unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)',
         new='unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)',
         pkg="./internal/secprim/", run="TestGuardOpenRefusesCommitShapes",
         kill="final_symlink",
         narrows="DELETION: drops the final-component O_NOFOLLOW flag"),
    dict(id="D13", file="internal/secprim/open_member_unix.go",
         old='unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)',
         new='unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)',
         pkg="./internal/secprim/", run="TestGuardOpenRefusesSymlinkedParent",
         kill="TestGuardOpenRefusesSymlinkedParent",
         narrows="DELETION: drops the intermediate O_NOFOLLOW flag"),
    dict(id="D14", file="internal/provhost/runner.go",
         old='''	if err := secprim.CheckArgv([]string{executable}); err != nil {
		return Result{}, err
	}''',
         new='',
         pkg="./internal/provhost/", run="TestExecRunnerRefusesUnusableExecutable",
         kill="TestExecRunnerRefusesUnusableExecutable",
         narrows="DELETION: drops the runner argv gate"),
    dict(id="D14b", file="internal/provhost/runner.go",
         old='if err := secprim.CheckArgv([]string{executable}); err != nil {',
         new='if err := secprim.CheckArgv([]string{executable}); err != nil && false {',
         pkg="./internal/provhost/", run="TestExecRunnerRefusesUnusableExecutable",
         kill="TestExecRunnerRefusesUnusableExecutable",
         narrows="DELETION: deadens the runner argv gate while keeping the import (scripted D14 alone is a compile-fail on the unused import)"),
    dict(id="D15", file="internal/secprim/redact.go",
         old='''		rendered.WriteString(rest[:scheme+3])
		rendered.WriteString(redactedMarker)
		rest = rest[at:]''',
         new='''		rendered.WriteString(rest[:scheme+3])
		rendered.WriteString(rest[scheme+3 : at])
		rest = rest[at:]''',
         pkg="./internal/secprim/", run="TestRedactURLUserinfo",
         kill="TestRedactURLUserinfo",
         narrows="DELETION: keeps URL userinfo credentials"),
    dict(id="D16", file="internal/secprim/redact.go",
         old='redacted = redactCorpusValues(redacted, secrets)',
         new='',
         pkg="./internal/secprim/", run="TestRedactCorpusValues",
         kill="TestRedactCorpusValues",
         narrows="DELETION: drops the corpus arm"),
    dict(id="D17", file="internal/secprim/path.go",
         old='''	if info == nil {
		return failPath("target stat missing", "nil file info")
	}''',
         new='''	if info == nil && false {
		return failPath("target stat missing", "nil file info")
	}''',
         pkg="./internal/secprim/", run="TestOpenStatFailureSeam",
         kill="TestOpenStatFailureSeam",
         narrows="DELETION: drops the nil-stat arm (killed via panic)"),
    dict(id="D18", file="internal/secprim/path.go",
         old='if _, err := scalar.ParseAbsolutePath(platform, root); err != nil {',
         new='if _, err := scalar.ParseAbsolutePath(platform, root); err != nil && false {',
         pkg="./internal/secprim/", run="TestGuardResolve",
         kill="TestGuardResolve",
         narrows="DELETION: drops the guard-root delegation"),
    dict(id="D8", file="internal/secprim/census_test.go",
         old='"secret", "credential", "passwd", "password", "token",',
         new='"credential", "passwd", "password", "token",',
         pkg="./internal/secprim/", run="TestSecretSiteRosterIsComplete",
         kill="TestSecretSiteRosterIsComplete",
         narrows="INSTRUMENT: drops one census token, rows orphan"),
    dict(id="D9", file="internal/secprim/classes_test.go",
         old='class: "Provider credentials",',
         new='class: "Provider credentials (renamed)",',
         pkg="./internal/secprim/", run="TestExclusionClassRosterIsComplete",
         kill="TestExclusionClassRosterIsComplete",
         narrows="INSTRUMENT: renames one class row, derivation orphans it"),
    dict(id="D19", file="internal/secprim/nofollow.go",
         old='''	file, err := openNoFollow(path)
	if err != nil {
		return nil, failPath("open failed", memberErrorTarget(path)).WithCause(err)
	}''',
         new='''	file, err := openNoFollow(path)
	if err != nil && false {
		return nil, failPath("open failed", memberErrorTarget(path)).WithCause(err)
	}''',
         pkg="./internal/secprim/", run="TestOpenNoFollowFile",
         kill="TestOpenNoFollowFile",
         narrows="DELETION: ignores the kernel open error (killed via panic)"),
    dict(id="D20", file="internal/secprim/nofollow.go",
	old='''	if !guardRootMatches(root, guard.root) {
		return nil, failPath("guard root mismatch", memberErrorTarget(member))
	}''',
	new='''	if !guardRootMatches(root, guard.root) && false {
		return nil, failPath("guard root mismatch", memberErrorTarget(member))
	}''',
	pkg="./internal/secprim/", run="TestGuardOpenRefusesForeignRootHandle",
	kill="TestGuardOpenRefusesForeignRootHandle",
	narrows="DELETION: drops the handle-to-root binding arm"),
]

BY_ID = {m["id"]: m for m in MUTANTS}


def run_one(mid):
    m = BY_ID[mid]
    path = m["file"]
    with open(path, encoding="utf-8") as f:
        src = f.read()
    if src.count(m["old"]) != 1:
        return (mid, "ERROR", "anchor count %d" % src.count(m["old"]), m["narrows"])
    with open(path + ".mutbak", "w", encoding="utf-8") as f:
        f.write(src)
    try:
        with open(path, "w", encoding="utf-8") as f:
            f.write(src.replace(m["old"], m["new"]))
        p = subprocess.run(["go", "test", m["pkg"], "-count=1", "-run", m["run"]],
                           capture_output=True, text=True, timeout=240)
        out = p.stdout + p.stderr
        if p.returncode == 0:
            return (mid, "SURVIVED", "exit 0", m["narrows"])
        if ("--- FAIL: " + m["kill"]) in out or ("    --- FAIL: " + m["kill"]) in out:
            return (mid, "KILLED", m["kill"], m["narrows"])
        # named-test match on subtests: check any FAIL line mentions kill stem
        fails = [l.strip() for l in out.splitlines() if "FAIL:" in l]
        if any(m["kill"] in fl for fl in fails):
            return (mid, "KILLED", m["kill"] + " (subtest)", m["narrows"])
        return (mid, "KILLED-OTHER", "; ".join(fails[:3]), m["narrows"])
    finally:
        with open(path + ".mutbak", encoding="utf-8") as f:
            orig = f.read()
        with open(path, "w", encoding="utf-8") as f:
            f.write(orig)
        subprocess.run(["rm", "-f", path + ".mutbak"])


if __name__ == "__main__":
    for mid in sys.argv[1:]:
        if mid not in BY_ID:
            print("%s ERROR unknown id" % mid)
            continue
        print(" | ".join(run_one(mid)))
