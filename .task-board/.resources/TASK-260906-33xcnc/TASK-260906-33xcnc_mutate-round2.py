#!/usr/bin/env python3
"""Leaf-5 mutant battery: converge-and-harden residue over internal/environ + siblings.

Every mutant applies exact-match replacements to production files, runs the
named package suite(s), and requires a NAMED test in the failure output.
Restoration is copy-back verified by checksum. Statuses:

  KILLED        suite red with the named test failing (narrowing proven)
  KILLED_OTHER  suite red without the named test (investigated, never folded in)
  SURVIVED      suite green (a survivor states its bound; expected-survive
                mutants invert: SURVIVED is the prediction holding)
  NOT_APPLIED   applier pattern count != 1 (harness fails closed, not silent)
  COMPILE_FAIL  mutant broke the build (distinct from Survived: unassessable)

Kill classes: narrowing (gate stays, admits exactly one rejected member),
arm-deletion (arm removed), census-only (behavioral green, census red),
audit-only (behavioral green, audit red), harness (category validation).
"""
import json
import subprocess
import sys
import hashlib

ROOT = "/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260905-3t31e9/worktree"

MUTANTS = []
def M(mid, files, pkgs, kill, klass, narrows, expect="kill", steps=None, appends=None):
    MUTANTS.append({"id": mid, "files": files, "pkgs": pkgs, "kill": kill,
                    "class": klass, "narrows": narrows, "expect": expect,
                    "steps": steps or [], "appends": appends or []})

ENV = ["./internal/environ/"]

# --- Round-2 regression: the 27 narrowing mutants, re-run after this leaf ---
M("N1_lowsurrogate", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/bare_low_escape", "narrowing",
  "admits lone low escapes only; highs still refused",
  steps=[("\t\t\t\tcase unit >= 0xdc00 && unit <= 0xdfff:\n\t\t\t\t\treturn true",
          "\t\t\t\tcase unit >= 0xdc00 && unit <= 0xdfff:\n\t\t\t\t\treturn false")])
M("N2_bytelength", ["internal/environ/decode.go"], ENV,
  "TestStringMeasureCountsRunes/environ_helper", "narrowing",
  "StringLength counts bytes; 128 wide chars refused",
  steps=[("func StringLength(value string) int {\n\treturn utf8.RuneCountInString(value)\n}",
          "func StringLength(value string) int {\n\treturn len(value)\n}")])
M("N3_tuplextensions", ["internal/environ/tuple.go"], ENV,
  "TestTupleAgreementAcrossFacades/extensions_refused", "narrowing",
  "tuple admits exactly the extensions member",
  steps=[('\tif name, unknown := unknownMember(members, tupleMembers); unknown {',
          '\tif name, unknown := unknownMember(members, tupleMembers); unknown && name != "extensions" {')])
M("N4_caplength", ["internal/environ/observation.go"], ENV,
  "TestDecodeEnvironmentObservation/ninth_capability_refused", "narrowing",
  "capabilities admit exactly nine keys",
  steps=[("\tif len(members) != len(capabilityOrder) {",
          "\tif len(members) != len(capabilityOrder) && len(members) != 9 {")])
M("N6_envunderscore", ["internal/environ/tuple.go"], ENV,
  "TestSharedGrammarsAreOneLanguage", "narrowing",
  "env-id admits underscore (literal drift caught by one-language)",
  steps=[("`^[a-z][a-z0-9.-]{0,63}$`", "`^[a-z][a-z0-9._-]{0,63}$`")])
# T1 rebirths the original admit hole with a token-preserving change:
# the arm looks ahead without consuming the backslash pair, so every
# `\u` byte pair reads as an escape regardless of string context.
# Quoted `\\ud800` text misreads as a high surrogate (over-strict
# refuse of the run-parity rows) and pairs with a following real
# escape (admitting the real lone escape encoding/json rewrites to
# U+FFFD: the composed rows). Well-formed escapes behave
# identically; every name, text, and structure stays.
M("T1_rawscan", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/escaped_backslash_high_run2", "narrowing",
  "raw entry scan: quoted text misread as escape (both directions)",
  steps=[("\t\t\tcase '\\\\':\n\t\t\t\tindex++\n\t\t\t\tif index >= len(data) || data[index] != 'u' {\n\t\t\t\t\tcontinue\n\t\t\t\t}\n\t\t\t\tunit, end, ok := readUTF16Escape(data, index-1)",
          "\t\t\tcase '\\\\':\n\t\t\t\tif index+1 >= len(data) || data[index+1] != 'u' {\n\t\t\t\t\tcontinue\n\t\t\t\t}\n\t\t\t\tunit, end, ok := readUTF16Escape(data, index)")])
M("M1_uint53floor", ["internal/environ/decode.go"], ENV,
  "TestCheckUint53BoundsEdges", "narrowing",
  "admits below-minimum values; ceiling still holds",
  steps=[("\tif value < minimum || value > maximum {", "\tif value > maximum {")])
M("M2_uint53magnitude", ["internal/environ/decode.go"], ENV,
  "TestCheckUint53BoundsEdges", "narrowing",
  "admits exactly 2^53; 2^53+1 still refused",
  steps=[("\t\tif value > maxUint53 {", "\t\tif value > maxUint53+1 {")])
M("M4_uint53trailing", ["internal/environ/decode.go"], ENV,
  "TestCheckUint53BoundsEdges", "arm-deletion",
  "trailing-data arm removed; `12a` reads as 12",
  steps=[("\t// Decode reads one value and stops: without the trailing\n\t// check a hostile slice like `12a` would read as 12. Member\n\t// slices arriving through DecodeStrictObject can never carry\n\t// trailing data, but this entry takes raw slices and must\n\t// not trust them.\n\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn 0, false\n\t}\n\tnumber, ok := value.(json.Number)",
          "\tnumber, ok := value.(json.Number)")])
M("M5_stringsdup", ["internal/environ/decode.go"], ENV,
  "TestCheckSortedUniqueStringsHalves/sorted_duplicated_refuses", "narrowing",
  "admits duplicates; unsorted still refused",
  steps=[("\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1] >= values[index] {\n\t\t\treturn nil, false\n\t\t}\n\t}\n\treturn values, true\n}\n\n// CheckSortedUniqueDigests",
          "\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1] > values[index] {\n\t\t\treturn nil, false\n\t\t}\n\t}\n\treturn values, true\n}\n\n// CheckSortedUniqueDigests")])
M("M6_stringsorder", ["internal/environ/decode.go"], ENV,
  "TestCheckSortedUniqueStringsHalves/unsorted_unique_refuses", "narrowing",
  "admits unsorted-but-unique; duplicates still refused",
  steps=[("\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1] >= values[index] {\n\t\t\treturn nil, false\n\t\t}\n\t}\n\treturn values, true\n}\n\n// CheckSortedUniqueDigests",
          "\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1] == values[index] {\n\t\t\treturn nil, false\n\t\t}\n\t}\n\treturn values, true\n}\n\n// CheckSortedUniqueDigests")])
M("M7_digestbridge", ["internal/environ/decode.go"], ENV,
  "TestTupleAgreementAcrossFacades/bad_fingerprint", "arm-deletion",
  "digest bridge removed; non-digests admitted",
  steps=[("\tdigest, err := scalar.ParseDigest(value)\n\tif err != nil {\n\t\treturn scalar.Digest{}, false\n\t}\n\treturn digest, true",
          "\tdigest, _ := scalar.ParseDigest(value)\n\treturn digest, true")])
M("M8_timestampbridge", ["internal/environ/decode.go"], ENV,
  "TestDecodeEnvironmentObservation/bad_timestamp", "arm-deletion",
  "timestamp bridge removed; non-timestamps admitted",
  steps=[("\tinstant, err := scalar.ParseTimestamp(value)\n\tif err != nil {\n\t\treturn scalar.Timestamp{}, false\n\t}\n\treturn instant, true",
          "\tinstant, _ := scalar.ParseTimestamp(value)\n\treturn instant, true")])
M("M9_tuplearch", ["internal/environ/tuple.go"], ENV,
  "TestTupleAgreementAcrossFacades/x86_refused", "narrowing",
  "admits exactly x86 alongside the closed pair",
  steps=[("\t\tif architecture == allowed {", "\t\tif architecture == allowed || architecture == \"x86\" {")])
M("M10_tuplesemver", ["internal/environ/tuple.go"], ENV,
  "TestTupleAgreementAcrossFacades/short_version", "narrowing",
  "admits short versions; also drifts the grammar literal (census must fire too)",
  steps=[("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)",
          "^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\.(0|[1-9][0-9]*))?")])
M("M11_reasonavailable", ["internal/environ/observation.go"], ENV,
  "TestDecodeEnvironmentObservation/available_with_reason_refused", "arm-deletion",
  "available-with-reason arm removed",
  steps=[('\tif status == "available" && hasReason {\n\t\treturn CapabilityResult{}, false\n\t}\n', "")])
M("M12_reasonconditional", ["internal/environ/observation.go"], ENV,
  "TestDecodeEnvironmentObservation/conditional_without_reason_refused", "arm-deletion",
  "conditional-without-reason arm removed",
  steps=[('\tif status != "available" && !hasReason {\n\t\treturn CapabilityResult{}, false\n\t}\n', "")])
M("M13_capabsent", ["internal/environ/observation.go"], ENV,
  "TestDecodeEnvironmentObservation/unknown_capability_key", "arm-deletion",
  "capability presence arm removed; swapped key admitted",
  steps=[("\t\tmember, present := members[name]\n\t\tif !present {\n\t\t\treturn nil, false\n\t\t}",
          "\t\tmember, present := members[name]\n\t\tif !present {\n\t\t\tcontinue\n\t\t}\n\t\t_ = present")])
M("M14a_evidencedup", ["internal/environ/decode.go"], ENV,
  "TestDecodeEnvironmentObservation/duplicated_evidence_refused", "narrowing",
  "evidence admits duplicates; unsorted still refused",
  steps=[("\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1].String() >= values[index].String() {",
          "\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1].String() > values[index].String() {")])
M("M14b_evidenceorder", ["internal/environ/decode.go"], ENV,
  "TestDecodeEnvironmentObservation/unsorted_evidence_refused", "narrowing",
  "evidence admits unsorted; duplicates still refused",
  steps=[("\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1].String() >= values[index].String() {",
          "\tfor index := 1; index < len(values); index++ {\n\t\tif values[index-1].String() == values[index].String() {")])
M("M15_framedup", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/duplicate_member", "arm-deletion",
  "duplicate-member arm removed",
  steps=[('\t\tif _, duplicate := members[key]; duplicate {\n\t\t\treturn nil, &Fault{Detail: FaultDuplicate, Member: key}\n\t\t}\n', "")])
M("M16_frametrailing", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/trailing_data", "arm-deletion",
  "trailing-data arm removed",
  steps=[("\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn nil, &Fault{Detail: FaultTrailing}\n\t}\n", "")])
M("M17_frameutf8", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/non-utf8_bytes", "arm-deletion",
  "UTF-8 arm removed",
  steps=[("\tif !utf8.Valid(data) {\n\t\treturn nil, &Fault{Detail: FaultNotUTF8}\n\t}\n", "")])
M("M18_highmispair", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/high_followed_by_non-low", "narrowing",
  "admits high-followed-by-non-low; lone lows still refused",
  steps=[("\t\t\t\tcase unit >= 0xd800 && unit <= 0xdbff:\n\t\t\t\t\tsecond, _, ok := readUTF16Escape(data, end)\n\t\t\t\t\tif !ok || second < 0xdc00 || second > 0xdfff {\n\t\t\t\t\t\treturn true\n\t\t\t\t\t}\n\t\t\t\t\tindex = end + 6 - 1",
          "\t\t\t\tcase unit >= 0xd800 && unit <= 0xdbff:\n\t\t\t\t\t_, _, _ = readUTF16Escape(data, end)\n\t\t\t\t\tindex = end + 6 - 1")])
M("M19_extensions", ["internal/environ/decode.go"], ENV,
  "TestDecodeEnvironmentObservation/bad_extensions_key", "narrowing",
  "admits exactly the nodots key; other keys still refused",
  steps=[('\tfor name := range members {\n\t\tif len(name) < 3 || len(name) > 253 || !reverseDNSPattern.MatchString(name) {',
          '\tfor name := range members {\n\t\tif name != "nodots" && (len(name) < 3 || len(name) > 253 || !reverseDNSPattern.MatchString(name)) {')])
M("M20_tupleunknown", ["internal/environ/tuple.go"], ENV,
  "TestTupleAgreementAcrossFacades/unknown_member", "arm-deletion",
  "unknown-member arm removed",
  steps=[('\tif name, unknown := unknownMember(members, tupleMembers); unknown {', '\tif name, unknown := unknownMember(members, tupleMembers); unknown && false {\n\t\t_ = name')])
M("M21_obsschema", ["internal/environ/observation.go"], ENV,
  "TestDecodeEnvironmentObservation/wrong_schema", "narrowing",
  "admits exactly the session-adapter-manifest schema",
  steps=[('\tif schema, ok := rawString(members["schema"]); !ok || schema != observationSchema {',
          '\tif schema, ok := rawString(members["schema"]); !ok || (schema != observationSchema && schema != "urn:ax:schema:session-adapter-manifest") {')])
# --- Residue attacks: F2 exits with no narrowing mutant, first narrowed here ---
M("R_missingfirst", ["internal/environ/decode.go"], ENV,
  "TestTupleAgreementAcrossFacades/missing_environment_id", "narrowing",
  "missingMember skips the first required name; missing environment_id admitted",
  steps=[("\tfor _, name := range required {", "\tfor _, name := range required[1:] {")])
M("R_laddernondigit", ["internal/environ/decode.go"], ENV,
  "TestParseUint53LiteralRefusesNonDigits", "narrowing",
  "ladder admits exactly the letter a; other non-digits still refused",
  steps=[("\t\tif digit < '0' || digit > '9' {", "\t\tif digit < '0' || digit > '9' && digit != 'a' {")])
M("R_strfloor", ["internal/environ/decode.go"], ENV,
  "TestDecodeEnvironmentObservation/empty_version", "narrowing",
  "CheckStringBounds drops the floor; empty strings admitted",
  steps=[("\tif length < minimum || length > maximum {", "\tif length > maximum {")])
M("R_rawstringerr", ["internal/environ/decode.go"], ENV,
  "TestEveryRefusalSiteIsExercised/tuple_platform_string", "arm-deletion",
  "rawString error arm removed; the direct-rawString arms (platform/arch/version/provider) admit non-strings as empty and refuse at the wrong arm",
  expect="kill",
  steps=[("func rawString(raw json.RawMessage) (string, bool) {\n\tvar value string\n\tif err := json.Unmarshal(raw, &value); err != nil {\n\t\treturn \"\", false\n\t}\n\treturn value, true\n}",
          "func rawString(raw json.RawMessage) (string, bool) {\n\tvar value string\n\t_ = json.Unmarshal(raw, &value)\n\treturn value, true\n}")])
M("R_decodearraytrailing", ["internal/environ/decode.go"], ENV,
  "TestDecodeArrayRefusesTrailingData", "arm-deletion",
  "decodeArray trailing arm removed",
  steps=[("\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn nil, false\n\t}\n\treturn elements, true",
          "\treturn elements, true")])
M("R_digestnonstr", ["internal/environ/decode.go"], ENV,
  "", "arm-deletion",
  "CheckDigest type arm removed; EXPECTED SURVIVOR: ParseDigest refuses the zero value downstream",
  expect="survive",
  steps=[("func CheckDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\tvalue, ok := rawString(raw)\n\tif !ok {\n\t\treturn scalar.Digest{}, false\n\t}",
          "func CheckDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\tvalue, _ := rawString(raw)")])
# --- Sibling texture: trailing-data arm per package, deletion + narrowing ---
SESS = ["./internal/sessadapter/"]
M("S1_sesstrailingdel", ["internal/sessadapter/decode.go"], SESS,
  "TestRawUint53RefusesTrailingData", "arm-deletion",
  "sessadapter trailing arm removed; `12a` reads as 12",
  steps=[("\t// Decode reads one value and stops: without the trailing\n\t// check a hostile slice like `12a` would read as 12. Member\n\t// slices arriving through decodeStrictObject can never carry\n\t// trailing data, but this entry takes raw slices and must\n\t// not trust them.\n\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn 0, false\n\t}\n\tnumber, ok := value.(json.Number)",
          "\tnumber, ok := value.(json.Number)")])
M("S2_sesstrailingnarrow", ["internal/sessadapter/decode.go"], SESS,
  "TestRawUint53RefusesTrailingData", "narrowing",
  "refuses only error trailing; `12 13` admitted as 12",
  steps=[("\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn 0, false\n\t}",
          "\tif _, err := decoder.Token(); err != nil && err != io.EOF {\n\t\treturn 0, false\n\t}")])
DIRN = ["./internal/dirnode/"]
M("S3_dirtrailingdel", ["internal/dirnode/decode.go"], DIRN,
  "TestRawUint53RefusesTrailingData", "arm-deletion",
  "dirnode trailing arm removed; `12a` reads as 12",
  steps=[("\t// Decode reads one value and stops: without the trailing\n\t// check a hostile slice like `12a` would read as 12. Member\n\t// slices arriving through decodeStrictObject can never carry\n\t// trailing data, but this entry takes raw slices and must\n\t// not trust them.\n\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn 0, false\n\t}\n\tnumber, ok := value.(json.Number)",
          "\tnumber, ok := value.(json.Number)")])
M("S4_dirtrailingnarrow", ["internal/dirnode/decode.go"], DIRN,
  "TestRawUint53RefusesTrailingData", "narrowing",
  "refuses only error trailing; `12 13` admitted as 12",
  steps=[("\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn 0, false\n\t}",
          "\tif _, err := decoder.Token(); err != nil && err != io.EOF {\n\t\t\treturn 0, false\n\t}")])
PROV = ["./internal/provhost/"]
M("S5_provtrailingdel", ["internal/provhost/opdecode.go"], PROV,
  "TestRawUint53RefusesTrailingData", "arm-deletion",
  "provhost trailing arm neutered (`&& false`: deleting the block would orphan the file's only io use and fail closed at compile time); `12a` reads as 12",
  steps=[("\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn 0, false\n\t}",
          "\tif _, err := decoder.Token(); err != io.EOF && false {\n\t\treturn 0, false\n\t}")])
M("S6_provtrailingnarrow", ["internal/provhost/opdecode.go"], PROV,
  "TestRawUint53RefusesTrailingData", "narrowing",
  "refuses only error trailing; `12 13` admitted as 12",
  steps=[("\tif _, err := decoder.Token(); err != io.EOF {\n\t\treturn 0, false\n\t}",
          "\tif _, err := decoder.Token(); err != nil && err != io.EOF {\n\t\t\treturn 0, false\n\t}")])
M("S7_sessboundforward", ["internal/sessadapter/decode.go"], SESS,
  "TestStringBoundEdges", "narrowing",
  "delegation forwards maximum+1; 129-char values admitted through sessadapter entries",
  steps=[("\treturn environ.CheckStringBounds(raw, minimum, maximum)",
          "\treturn environ.CheckStringBounds(raw, minimum, maximum+1)")])
M("S8_dirboundforward", ["internal/dirnode/decode.go"], DIRN,
  "TestStringBoundEdges", "narrowing",
  "delegation forwards maximum+1; overlong values admitted through dirnode entries",
  steps=[("\treturn environ.CheckStringBounds(raw, minimum, maximum)",
          "\treturn environ.CheckStringBounds(raw, minimum, maximum+1)")])
M("S9_sesssemversever", ["internal/sessadapter/tuple.go"], SESS + ENV,
  "TestTupleAgreementAcrossFacades/short_version", "narrowing",
  "tuple semver check admits exactly 1.2 (seam contract runs in the environ package)",
  steps=[('\tadapterVersion, ok := rawString(members["adapter_version"])\n\tif !ok || !environ.CheckSemver(adapterVersion) {',
          '\tadapterVersion, ok := rawString(members["adapter_version"])\n\tif (!ok || !environ.CheckSemver(adapterVersion)) && adapterVersion != "1.2" {')])
M("S10_dirextsever", ["internal/dirnode/decode.go"], DIRN,
  "TestExtensionKeyBoundEdges", "arm-deletion",
  "dirnode extensions delegation severed to admit-all",
  steps=[("func checkExtensions(raw json.RawMessage) bool {\n\treturn environ.CheckExtensions(raw)\n}",
          "func checkExtensions(raw json.RawMessage) bool {\n\treturn true\n}")])
# --- Scalar third spelling: narrowing over the single-string unit ---
M("C1_scalarlows", ["internal/scalar/scalar.go"], ENV,
  "TestScalarSurrogateAgreementAcrossUnits/bare_low_refused", "narrowing",
  "scalar admits lone lows; highs still refused",
  steps=[("\t\tcase code >= 0xdc00 && code <= 0xdfff:\n\t\t\treturn true",
          "\t\tcase code >= 0xdc00 && code <= 0xdfff:\n\t\t\treturn false")])
M("C2_scalarmispair", ["internal/scalar/scalar.go"], ENV,
  "TestScalarSurrogateAgreementAcrossUnits/high_followed_by_non-low_refused", "narrowing",
  "scalar admits high-followed-by-non-low",
  steps=[("\t\t\tlow, ok := parseHex16(value[index+3:])\n\t\t\tif !ok || low < 0xdc00 || low > 0xdfff {\n\t\t\t\treturn true\n\t\t\t}",
          "\t\t\tlow, _ := parseHex16(value[index+3:])\n\t\t\t_ = low")])
M("C3_scalarevenrun", ["internal/scalar/scalar.go"], ENV,
  "TestScalarSurrogateAgreementAcrossUnits/escaped_backslash_plus_real_lone_low_refused", "narrowing",
  "token-preserving: pair-skip dropped, quoted text misreads as escape and the composed lone low is admitted (the Story hole reborn)",
  steps=[("\t\tif value[index+1] != 'u' {\n\t\t\tindex++\n\t\t\tcontinue\n\t\t}",
          "\t\tif value[index+1] != 'u' {\n\t\t\tcontinue\n\t\t}")])
# --- Census-only: behavioral green, census red ---
M("G1_revivedgrammar", ["internal/sessadapter/decode.go"], ENV,
  "TestCheckHelpersDelegateToEnviron", "census-only",
  "revived semver copy with the identical literal; no entry changes verdict",
  steps=[],
  appends=[("internal/sessadapter/decode.go",
            '\nvar semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\\+[0-9A-Za-z-]+(?:\\.[0-9A-Za-z-]+)*)?$`)\n')])
M("G2_freshmeasure", ["internal/dirnode/decode.go"], ENV,
  "TestSharedShapesAreLedgered", "census-only",
  "fresh-name rune counter; no entry changes verdict",
  steps=[('import (\n\t"bytes"\n\t"encoding/json"\n\t"io"\n\t"strings"',
          'import (\n\t"bytes"\n\t"encoding/json"\n\t"io"\n\t"strings"\n\t"unicode/utf8"')],
  appends=[("internal/dirnode/decode.go",
            '\nfunc measureRunes(value string) int {\n\treturn utf8.RuneCountInString(value)\n}\n')])
M("G3_freshdecoder", ["internal/sessadapter/decode.go"], ENV,
  "TestSharedShapesAreLedgered", "census-only",
  "fresh-name decoder with duplicate texture; no entry changes verdict",
  appends=[("internal/sessadapter/decode.go",
            '\nfunc parseStrictObject(data []byte) bool {\n\tdecoder := json.NewDecoder(bytes.NewReader(data))\n\tvar duplicate string\n\t_ = decoder\n\t_ = duplicate\n\treturn len(data) == 0\n}\n')])
# --- Audit-only: behavioral green, site audit red ---
M("A1_refusealias", ["internal/environ/decode.go"], ENV,
  'constructor "refuse" referenced outside direct-call position', "audit-only",
  "rebound refuse var; every entry keeps its verdict, the use site is unattributed",
  appends=[("internal/environ/decode.go", '\nvar sneakyRefuse = refuse\n')])
# --- Gate-narrowing completions: every gate admits exactly one member ---
M("M7n_digestnarrow", ["internal/environ/decode.go"], ENV,
  "TestEveryRefusalSiteIsExercised/tuple_fingerprint", "narrowing",
  "digest bridge admits exactly sha256:zzzz; other non-digests still refused",
  steps=[("\tdigest, err := scalar.ParseDigest(value)\n\tif err != nil {\n\t\treturn scalar.Digest{}, false\n\t}",
          "\tdigest, err := scalar.ParseDigest(value)\n\tif err != nil && value != \"sha256:zzzz\" {\n\t\treturn scalar.Digest{}, false\n\t}")])
M("M8n_timestampnarrow", ["internal/environ/decode.go"], ENV,
  "TestEveryRefusalSiteIsExercised/observation_timestamp", "narrowing",
  "timestamp bridge admits exactly yesterday; other non-timestamps still refused",
  steps=[("\tinstant, err := scalar.ParseTimestamp(value)\n\tif err != nil {\n\t\treturn scalar.Timestamp{}, false\n\t}",
          "\tinstant, err := scalar.ParseTimestamp(value)\n\tif err != nil && value != \"yesterday\" {\n\t\treturn scalar.Timestamp{}, false\n\t}")])
M("M13n_capabsentnarrow", ["internal/environ/observation.go"], ENV,
  "TestDecodeEnvironmentObservation/unknown_capability_key", "narrowing",
  "presence admits exactly a missing directory_discovery (skipping its decode, which would refuse the nil member as fall-through); other absences still refused",
  steps=[("\t\tmember, present := members[name]\n\t\tif !present {\n\t\t\treturn nil, false\n\t\t}",
          "\t\tmember, present := members[name]\n\t\tif !present {\n\t\t\tif name != \"directory_discovery\" {\n\t\t\t\treturn nil, false\n\t\t\t}\n\t\t\tcontinue\n\t\t}")])
M("M15n_framedupnarrow", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/duplicate_member", "narrowing",
  "duplicate arm admits exactly a duplicated v; other duplicates still refused",
  steps=[('\t\tif _, duplicate := members[key]; duplicate {',
          '\t\tif _, duplicate := members[key]; duplicate && key != "v" {')])
M("M17n_frameutf8narrow", ["internal/environ/decode.go"], ENV,
  "TestFrameAgreementAcrossFacades/non-utf8_bytes", "narrowing",
  "UTF-8 arm admits exactly bodies containing 0xff; other non-UTF-8 still refused",
  steps=[("\tif !utf8.Valid(data) {",
          "\tif !utf8.Valid(data) && !bytes.Contains(data, []byte{0xff}) {")])
# --- Round-3 load-bearing delegation: token-preserving decoys (F1) ---
# D1 replays the reviewer's decoy verbatim: a discarded environ call
# plus a regrown digest copy drifted to case-folding. The harness runs
# the structural gate AND the behavioral suite: both must fail.
M("D1_decoycheckdigest", ["internal/sessadapter/decode.go"], SESS + ENV,
  "TestCheckHelpersDelegateToEnviron/sessadapter", "narrowing",
  "token-preserving: admits exactly uppercase-hex digests; lowercase still refused",
  steps=[("func checkDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\treturn environ.CheckDigest(raw)\n}",
          "func checkDigest(raw json.RawMessage) (scalar.Digest, bool) {\n\t_, _ = environ.CheckDigest(raw)\n\tvalue, ok := rawString(raw)\n\tif !ok {\n\t\treturn scalar.Digest{}, false\n\t}\n\tdigest, err := scalar.ParseDigest(strings.ToLower(value))\n\tif err != nil {\n\t\treturn scalar.Digest{}, false\n\t}\n\treturn digest, true\n}")])
# D2 keeps the wrapper's environ call but ignores the fault: every
# malformed frame slides to a later arm. Structural and behavioral
# suites must both fail.
M("D2_decoywrapperfault", ["internal/sessadapter/decode.go"], SESS + ENV,
  "TestDelegatingWrappersCallEnviron/sessadapter", "arm-deletion",
  "token-preserving: fault check removed; malformed frames slide to later arms",
  steps=[("\tmembers, fault := environ.DecodeStrictObject(data)\n\tif fault != nil {\n\t\treturn nil, &frameFault{detail: fault.Detail, member: fault.Member}\n\t}\n\treturn members, nil",
          "\tmembers, _ := environ.DecodeStrictObject(data)\n\treturn members, nil")])
# --- Round-3 duplicate class: one narrowing mutant per swept key (F2) ---
# Owner (environ decode.go:84): the seven keys the reviewer's sweep
# survived, plus body (previously killed only by coincidence).
for _key in ("body", "ok", "protocol_version", "request_id", "error", "capabilities", "provider_id", "cursor"):
    M("E_dupnarrow_" + _key.replace(".", "_"), ["internal/environ/decode.go"], ENV,
      "TestFrameAgreementRefusesDuplicateOfEveryDerivedMember/" + _key, "narrowing",
      "duplicate arm admits exactly a duplicated " + _key + "; every other key still refused",
      steps=[('\t\tif _, duplicate := members[key]; duplicate {',
              '\t\tif _, duplicate := members[key]; duplicate && key != "' + _key + '" {')])
# Retained copy (provhost protocol.go:295): the six keys the reviewer's
# sweep survived, driven through DecodeResponse.
for _key in ("body", "ok", "protocol_version", "error", "capabilities", "provider_id"):
    M("P_dupnarrow_" + _key, ["internal/provhost/protocol.go"], PROV,
      "TestDecodeResponseRefusesDuplicateOfEveryMember/" + _key, "narrowing",
      "provhost duplicate arm admits exactly a duplicated " + _key + "; every other key still refused",
      steps=[('\t\tif _, duplicate := members[key]; duplicate {',
              '\t\tif _, duplicate := members[key]; duplicate && key != "' + _key + '" {')])
# --- Harness validation (excluded from the headline ratio) ---
M("H1_ambiguous", ["internal/environ/decode.go"], ENV,
  "", "harness", "ambiguous pattern must NOT_APPLY, never half-apply",
  expect="not_applied",
  steps=[("\treturn false", "\treturn true")])
M("H2_arity", ["internal/sessadapter/decode.go"], SESS,
  "", "harness", "arity break must COMPILE_FAIL, never pass as survivor",
  expect="compile_fail",
  steps=[("\treturn environ.CheckStringBounds(raw, minimum, maximum)",
          "\treturn environ.CheckStringBounds(raw, minimum)")])

CENSUS_TESTS = ("Census", "Censused", "Ledgered", "OneLanguage", "OneTable",
                "Delegate", "Roster", "Grammar", "Shape")
AUDIT_TESTS = ("Audit", "Alias")


def sha(path):
    with open(path, "rb") as handle:
        return hashlib.sha256(handle.read()).hexdigest()


def run_suite(pkgs):
    proc = subprocess.run(["go", "test"] + pkgs + ["-count=1"],
                          cwd=ROOT, capture_output=True, text=True, timeout=300)
    return proc.returncode, proc.stdout + proc.stderr


def classify_failures(output):
    fails = [line.split("--- FAIL: ")[1].split()[0]
             for line in output.splitlines() if "--- FAIL: " in line]
    suites = set()
    for name in fails:
        if any(key in name for key in AUDIT_TESTS):
            suites.add("audit")
        elif any(key in name for key in CENSUS_TESTS):
            suites.add("census")
        else:
            suites.add("behavioral")
    if ("refusal audit:" in output or "environ refusals:" in output or
            "referenced outside direct-call position" in output):
        suites.add("audit")
    return fails, suites


def apply_mutant(mut):
    # Steps are (old, new) pairs applied to the mutant's first file;
    # appends are (path, text) pairs. Every touched file is backed
    # up before any write, and a non-unique pattern aborts with
    # NOT_APPLIED before anything is written.
    backups = {}
    for path in mut["files"]:
        full = ROOT + "/" + path
        with open(full) as handle:
            backups[path] = handle.read()
    for path, _ in mut["appends"]:
        if path not in backups:
            with open(ROOT + "/" + path) as handle:
                backups[path] = handle.read()
    target = ROOT + "/" + mut["files"][0]
    with open(target) as handle:
        src = handle.read()
    for old, _ in mut["steps"]:
        if src.count(old) != 1:
            return backups, "NOT_APPLIED"
    for path, text in mut["appends"]:
        with open(ROOT + "/" + path, "a") as handle:
            handle.write(text)
    with open(target) as handle:
        src = handle.read()
    for old, new in mut["steps"]:
        src = src.replace(old, new)
    with open(target, "w") as handle:
        handle.write(src)
    return backups, None


def restore(backups):
    for path, src in backups.items():
        with open(ROOT + "/" + path, "w") as handle:
            handle.write(src)


def run_mutant(mut):
    backups, blocked = apply_mutant(mut)
    if blocked:
        return {"id": mut["id"], "status": blocked, "fails": [], "suites": []}
    code, output = run_suite(mut["pkgs"])
    restore(backups)
    if code != 0 and "build failed" in output:
        return {"id": mut["id"], "status": "COMPILE_FAIL", "fails": [], "suites": []}
    fails, suites = classify_failures(output)
    if code == 0:
        return {"id": mut["id"], "status": "SURVIVED", "fails": fails, "suites": sorted(suites)}
    if mut["kill"] and mut["kill"] in output:
        return {"id": mut["id"], "status": "KILLED", "fails": fails, "suites": sorted(suites)}
    return {"id": mut["id"], "status": "KILLED_OTHER" if fails else "SURVIVED",
            "fails": fails, "suites": sorted(suites)}


def denominator():
    import re
    refuse = 0
    for name in ("decode.go", "tuple.go", "observation.go"):
        with open(ROOT + "/internal/environ/" + name) as handle:
            for line in handle:
                if "refuse(" in line and "func refuse" not in line and "var refuse" not in line:
                    refuse += 1
    bools = 0
    for name in ("decode.go", "tuple.go", "observation.go"):
        with open(ROOT + "/internal/environ/" + name) as handle:
            for line in handle:
                if re.search(r"return\b.*\bfalse\b", line):
                    bools += 1
    faults = 0
    with open(ROOT + "/internal/environ/decode.go") as handle:
        for line in handle:
            if "return nil, &Fault{" in line:
                faults += 1
    return refuse, bools, faults


def main():
    only = sys.argv[1:] or None
    refuse, bools, faults = denominator()
    print("denominator: %d refuse sites + %d boolean-false exits + %d frame-fault exits = %d" % (refuse, bools, faults, refuse + bools + faults))
    code, out = run_suite(["./internal/environ/", "./internal/sessadapter/",
                           "./internal/dirnode/", "./internal/provhost/"])
    print("baseline exit:", code)
    if code != 0:
        print(out[-3000:])
        sys.exit("baseline red; battery invalid")
    results = []
    for mut in MUTANTS:
        if only and mut["id"] not in only:
            continue
        res = run_mutant(mut)
        res.update({key: mut[key] for key in ("kill", "class", "narrows", "expect")})
        results.append(res)
        print("%-22s %-13s expect=%-12s suites=%s" % (
            res["id"], res["status"], mut["expect"], ",".join(res["suites"])))
        if res["status"] in ("KILLED_OTHER",):
            print("   fails:", res["fails"][:6])
    with open("/tmp/mutbattery_leaf6/results.json", "w") as handle:
        json.dump({"denominator": {"refuse": refuse, "bools": bools, "faults": faults}, "results": results},
                  handle, indent=2)
    billed = [r for r in results if r["class"] != "harness"]
    applied = [r for r in billed if r["status"] in ("KILLED", "SURVIVED", "KILLED_OTHER")]
    killed = [r for r in applied if r["status"] == "KILLED"]
    print("killed/applied: %d/%d" % (len(killed), len(applied)))
    for status in ("SURVIVED", "KILLED_OTHER", "NOT_APPLIED", "COMPILE_FAIL"):
        rows = [r for r in billed if r["status"] == status]
        print("%s: %d %s" % (status, len(rows), [r["id"] for r in rows]))


if __name__ == "__main__":
    main()
