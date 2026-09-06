// Package environ is the single environment library behind the
// Provider, Session Adapter, and Directory Node facades.
//
// Pinned specification v0.5.0 requires the Directory Node facade to
// expose source-local environment discovery "through the same
// parsers, identity logic, redaction rules, tuple gates, and
// fixtures as the Provider and Session Adapter facades" (Section
// 3.1), backed by "the same environment implementation as Provider
// 2 and Session Adapter 1" (Sections 7.9, 10.8, 13.14). Before this
// package each facade carried its own copy of those rules, and
// independent copies of the same rule drift in ways no per-facade
// reviewer can see. This package is the one implementation every
// facade must converge on:
//
//   - frame parsing: DecodeStrictObject is the only JSON entry
//     point, with duplicate-member rejection and the Section 1.6
//     lone-surrogate gate in canonical string-walk semantics;
//   - string measure: CheckStringBounds counts Unicode characters
//     (runes), never bytes and never UTF-16 code units;
//   - shared grammars: the environment-id, provider-id (via
//     scalar), SemVer, platform, and architecture rules;
//   - tuple gates: DecodeTuple admits the exact six-member
//     Environment Tuple of Sections 7.8 and 13.14, which never
//     carries executable provenance or extensions;
//   - observation gates: DecodeEnvironmentObservation admits the
//     Section 10.8.1 Environment Observation, reusing the tuple
//     admission model rather than creating a second one.
//
// Every refusal is a plain error whose text names the rule and the
// member: facades keep their own refusal dialects (Structured
// Error codes per protocol), so this package names rules instead
// of minting codes, and the boundary battery pins each facade's
// rendering separately.
//
// Stated bounds. Provider Identity refusal dialects stay in
// provhost and canonicaljson: the two validators must agree
// rule-for-rule and the agreement battery in this package pins
// that, but a third validator here would be another copy, not
// convergence. Byte-counted exec rules (Provider SpawnPlan argv
// and env literals, Section 5.1) stay in provhost: they bound
// bytes by specification while every character-bound member in
// this story counts runes, and the measure battery ledgers that
// split. The tuple-registry signature, sequence monotonicity, and
// publication rules of Section 13.14 fail closed at the registry
// owner, which does not exist yet. This package mutates no durable
// state and keeps no cache: every entry point is a pure function
// of its input bytes.
package environ
