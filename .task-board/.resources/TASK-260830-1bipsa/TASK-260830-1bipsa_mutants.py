#!/usr/bin/env python3
"""Mutation harness for internal/cliresult.

Each mutant edits exactly one gate in production code, verifies the edit was
really applied (the removed text is gone and the written text is present), runs
the package suite, and requires it to fail. A mutant that survives is a gate the
tests do not actually check.
"""
import subprocess, sys, os

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
P = lambda name: os.path.join(ROOT, "internal", "cliresult", name)

# (id, file, old, new)  --  new == "" means deletion.
MUTANTS = [
 ("closed-extra-member", "value.go",
  '''	for _, key := range sortedKeys(object) {
		if _, ok := declared[key]; !ok {
			return failf("%s carries unknown member %q", where, key)
		}
	}
''', ""),
 ("closed-missing-member", "value.go",
  '''		if _, present := object[member]; !present {
			return failf("%s is missing required member %q", where, member)
		}
''', ""),
 ("sorted-order", "value.go",
  '''			if key < previous {
				return failf("%s.%s is not sorted bytewise: %q follows %q", where, name, key, previous)
			}
''', ""),
 ("sorted-unique", "value.go",
  '''			if key == previous {
				return failf("%s.%s repeats %q", where, name, key)
			}
''', ""),
 ("array-lower-bound", "value.go",
  '''	if len(members) < minimum || len(members) > maximum {
		return failf("%s.%s has %d members, the bound is %d..%d", where, name, len(members), minimum, maximum)
	}
	previous := ""''',
  '''	if len(members) > maximum {
		return failf("%s.%s has %d members, the bound is %d..%d", where, name, len(members), minimum, maximum)
	}
	previous := ""'''),
 ("string-upper-bound", "value.go",
  '	if count < minimum || count > maximum {',
  '	if count < minimum || count > maximum+1 {'),
 ("uint53-floor", "value.go",
  '''	if value < floor {''', '''	if value < 0 {'''),
 ("stop-null-checkpoint-state", "bodies.go",
  '		if graceful || resumable || !bootstrapAborted || state != "failed" {',
  '		if graceful || resumable || !bootstrapAborted {'),
 ("stop-null-checkpoint-aborted", "bodies.go",
  '		if graceful || resumable || !bootstrapAborted || state != "failed" {',
  '		if graceful || resumable || state != "failed" {'),
 ("stop-process-closed", "bodies.go",
  '''	if err := requireTrue(body, "body", "process_closed"); err != nil {
		return err
	}
''', ""),
 ("stop-stopped-resumable", "bodies.go",
  '	if state == "stopped" && (!resumable || bootstrapAborted) {',
  '	if state == "stopped" && (!resumable && bootstrapAborted) {'),
 ("takeover-direct-adopted", "bodies.go",
  '''		if adopted {
			return failf("a direct takeover reports adopted true; Section 14.2 requires false")
		}
''', ""),
 ("takeover-taskboard-order", "bodies.go",
  '\t\tif resumed && !adopted {',
  '\t\tif false && resumed && !adopted {'),
 ("logs-emitter-binding", "bodies.go",
  '\t\tif hostID != emitter {',
  '\t\tif false && hostID != emitter {'),
 ("contract-name-vocabulary", "bodies.go",
  '''		if _, known := helloContractNames[key]; !known {
			return failf("%s.%s carries %q, which is not a contract name", where, name, key)
		}
''', ""),
 ("materialize-preserved-checkpoint", "types.go",
  '\tif replacement != preserved {',
  '\tif false && replacement != preserved {'),
 ("materialize-unmanaged-nonempty", "types.go",
  '''	if classification == "unmanaged_nonempty" {
		return failf("%s reports destination_classification unmanaged_nonempty in a success object", where)
	}
''', ""),
 ("materialize-committed", "types.go",
  '''	if err := requireTrue(object, where, "committed"); err != nil {
		return err
	}
''', ""),
 ("materialize-ownership-changed", "types.go",
  '''	if err := requireFalse(object, where, "ownership_changed"); err != nil {
		return err
	}
''', ""),
 ("capability-vocabulary", "types.go",
  '''		if _, known := providerCapabilityNames[name]; !known {
			return failf("%s.capabilities carries %q, which is not a capability name", where, name)
		}
''', ""),
 ("capability-bound", "types.go",
  '	if len(capabilities) > maxSessionCapabilities {',
  '	if len(capabilities) > maxSessionCapabilities+1 {'),
 ("summary-stopped-checkpoint", "types.go",
  '\tif state == "stopped" && !checkpointPresent {',
  '\tif false && state == "stopped" && !checkpointPresent {'),
 ("absolute-path-any-platform", "types.go",
  '''	return failf("%s.%s is not an absolute path on any supported platform", where, name)''',
  '''	return nil'''),
 ("id-forbidden", "command.go",
  '''	case idForbidden:
		if present {
			return failf("%s is non-null, which command %q forbids", name, command)
		}
''', ""),
 ("id-required", "command.go",
  '''	case idRequired:
		if !present {
			return failf("%s is null, which command %q forbids", name, command)
		}
''', ""),
 ("session-scope-nested", "command.go",
  '\t\tif nested.String() != session.String() {\n\t\t\treturn failf(\n\t\t\t\t"body.session.session_id %s differs from the top-level session_id %s",',
  '\t\tif false && nested.String() != session.String() {\n\t\t\treturn failf(\n\t\t\t\t"body.session.session_id %s differs from the top-level session_id %s",'),
 ("session-scope-member", "command.go",
  '\t\tif nested.String() != session.String() {\n\t\t\treturn failf(\n\t\t\t\t"body.%s %s differs from the top-level session_id %s",',
  '\t\tif false && nested.String() != session.String() {\n\t\t\treturn failf(\n\t\t\t\t"body.%s %s differs from the top-level session_id %s",'),
 ("unimplemented-body-admitted", "command.go",
  '''	if entry.validate == nil {
		return fmt.Errorf(
			"%w: %q selects CLI Result %s, whose body this repository does not build",
			ErrUnimplementedVersion, command, entry.version)
	}
	return entry.validate(body)''',
  '''	if entry.validate == nil {
		return nil
	}
	return entry.validate(body)'''),
 ("decode-ok-false", "decode.go",
  '''	if !ok {
		return nil, failf("ok is false; a failure is one Structured Error object, not a CLI Result")
	}
''', ""),
 ("decode-unknown-top-level", "decode.go",
  '''	for _, key := range names {
		if _, ok := declared[key]; !ok {
			return nil, failf("document carries unknown top-level member %q", key)
		}
	}
''', ""),
 ("decode-major-mismatch", "decode.go",
  '\tif candidateMajor != expectedMajor {',
  '\tif false && candidateMajor != expectedMajor {'),
 ("decode-tag-version-agreement", "decode.go",
  '\tif selected != version {',
  '\tif false && selected != version {'),
 ("decode-required-nullable-present", "decode.go",
  '''	if len(raw) == 0 {
		return scalar.UUIDv7{}, false, failf("%s is required and must be present as a UUIDv7 or null", name)
	}
''', ""),
 ("extensions-depth", "decode.go",
  '''		if depth == maxExtensionDepth {
			return failf("%s exceeds the maximum nesting depth %d", where, maxExtensionDepth)
		}
		for index, member := range typed {''',
  '''		if depth == maxExtensionDepth+1 {
			return failf("%s exceeds the maximum nesting depth %d", where, maxExtensionDepth)
		}
		for index, member := range typed {'''),
 ("extensions-key-grammar", "decode.go",
  '''		if len(key) < 3 || len(key) > 253 || !reverseDNSPattern.MatchString(key) {
			return failf("extensions key %q is not a 3..253 character lowercase reverse-DNS name", key)
		}
''', ""),
 ("input-utf8", "cliresult.go",
  '''		if !utf8.ValidString(typed) {
			return nil, failf("%s is not valid UTF-8", where)
		}
		return typed, nil''',
  '''		return typed, nil'''),
 ("input-type-vocabulary", "cliresult.go",
  '''	default:
		return nil, failf("%s has type %T, which the common data model does not admit", where, value)
	}
}''',
  '''	default:
		return value, nil
	}
}'''),
 ("session-kind-required", "cliresult.go",
  '''	case takesKind && kind == "":
		return failf("command %q requires a session kind: Section 14.2 fixes its adoption rule per kind", command)''',
  '''	case takesKind && kind == "":
		return nil'''),
 ("flags-yes-rejection", "flags.go",
  '''		if !entry.documentedConfirmation {
			return mustInvalidArguments(fmt.Sprintf(
				"%s has no documented confirmation and rejects --yes", invocation.surface))
		}
''', ""),
 ("flags-json-rejection", "flags.go",
  '''		if !entry.acceptsJSON {
			return mustInvalidArguments(fmt.Sprintf(
				"%s rejects --json: it is an RPC protocol endpoint, not a CLI Result producer",
				invocation.surface))
		}
''', ""),
 ("confirmation-expectation-flags", "output.go",
  '''	if len(missing) > 0 {
		return Confirmation{}, mustInvalidArguments(fmt.Sprintf(
			"%s requires %s; --yes does not bypass an expected owner, epoch, or checkpoint check",
			operation, strings.Join(missing, " and ")))
	}
''', ""),
 ("confirmation-requires-yes", "output.go",
  '''	if !invocation.Yes() {
		return Confirmation{}, mustConfirmationRequired(fmt.Sprintf(
			"%s requires --yes in non-interactive mode", operation))
	}
''', ""),
 ("confirmation-interactive-prompt", "output.go",
  '''	if !invocation.NonInteractive() {
		return Confirmation{PromptRequired: true}, nil
	}
''', ""),
 ("emit-one-document", "output.go",
  '''	if emitter.emitted {
		return status, fmt.Errorf("%w: stdout already carries this command's one document", ErrStreamDiscipline)
	}
''', ""),
 ("emit-json-no-human-text", "output.go",
  '''		if outcome.Rendered != "" {
			return status, fmt.Errorf(
				"%w: JSON mode carries exactly one JSON document, so human text is not emitted",
				ErrStreamDiscipline)
		}
''', ""),
 ("exit-status-failure", "output.go",
  '''	case outcome.Failure != nil:
		return outcome.Failure.ExitCode(), nil''',
  '''	case outcome.Failure != nil:
		return SuccessExitStatus, nil'''),
 ("exit-status-ambiguous", "output.go",
  '''	case outcome.Result != nil && outcome.Failure != nil:
		return 0, fmt.Errorf("%w: an outcome is a success or a failure, never both", ErrStreamDiscipline)
''', ""),
 ("text-mode-failure-stream", "output.go",
  '''	if outcome.Failure != nil {
		return status, writeLine(emitter.streams.Stderr, outcome.Rendered)
	}
''', ""),
 ("progress-tty-only", "output.go",
  '''	if !emitter.streams.StderrIsTTY {
		return false, nil
	}
''', ""),
 ("prompt-non-interactive", "output.go",
  '''	if emitter.nonInteractive {
		return fmt.Errorf("%w: %q", ErrPromptForbidden, line)
	}
''', ""),
]

# SUBSUMED records the mutants that are expected to survive, with the guard
# that already refuses the same input earlier. A subsumed guard is retained in
# production so the declared bound stays visible, and it is listed here rather
# than quietly tolerated: a mutant that survives without an entry is a gate the
# tests do not check.
SUBSUMED = {
 "capability-bound":
   "the capability vocabulary has exactly maxSessionCapabilities members, so a map "
   "of admitted names can never exceed the count bound; the vocabulary check refuses "
   "the eighth entry first",
 "decode-required-nullable-present":
   "decodeClosedDocument already refuses a document missing any of the eight declared "
   "top-level members, so an absent raw identifier cannot reach this guard",
}

def run(cmd):
    return subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)

def main():
    killed, survived, broken = [], [], []
    for ident, name, old, new in MUTANTS:
        path = P(name)
        original = open(path).read()
        if original.count(old) != 1:
            broken.append((ident, "anchor found %d times" % original.count(old)))
            continue
        mutated = original.replace(old, new, 1)
        open(path, "w").write(mutated)
        try:
            current = open(path).read()
            # Verify the mutant really applied: the removed text is gone, and
            # for an edit the written text is present.
            if old in current or (new and new not in current):
                broken.append((ident, "mutant did not apply"))
                continue
            compiled = run(["go", "build", "./internal/cliresult"])
            if compiled.returncode != 0:
                broken.append((ident, "mutant does not compile"))
                continue
            result = run(["go", "test", "./internal/cliresult", "-count=1"])
            if result.returncode == 0:
                survived.append(ident)
            else:
                killed.append(ident)
        finally:
            open(path, "w").write(original)

    unexpected = [i for i in survived if i not in SUBSUMED]
    wrongly_killed = [i for i in killed if i in SUBSUMED]
    print("mutants: %d  killed: %d  survived: %d (subsumed: %d)  broken: %d"
          % (len(MUTANTS), len(killed), len(survived),
             len([i for i in survived if i in SUBSUMED]), len(broken)))
    for ident in survived:
        if ident in SUBSUMED:
            print("SUBSUMED", ident, "--", SUBSUMED[ident])
        else:
            print("SURVIVED", ident)
    for ident in wrongly_killed:
        print("NOT SUBSUMED AFTER ALL", ident)
    for ident, why in broken:
        print("BROKEN  ", ident, why)
    return 0 if not unexpected and not wrongly_killed and not broken else 1

sys.exit(main())
