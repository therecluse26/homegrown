// Package discover implements the Discovery domain (03-discover.md).
//
// # Domain Ownership
//
// discover:: owns:
//   - Quiz definitions (questions, answer options, scoring weights per methodology)
//   - Quiz results (anonymous, stored with URL-safe share IDs)
//   - State legal guide structured data
//   - Homeschooling 101 and advocacy content metadata
//
// # Key Invariants
//
//   - All endpoints are public (no auth middleware required). [03-discover §1]
//   - disc_ tables have NO RLS policies — they contain platform-authored content only,
//     not user-generated or family-scoped data. [ARCH §5.1]
//   - Quiz answer weights are internal-only and MUST NOT be returned to API clients.
//     stripWeights() enforces this at the service boundary. [03-discover §3.1]
//   - Share IDs are 12-character base62 nanoids (URL-safe, collision-resistant). [03-discover §3.1]
//   - Draft state guides return 404 (not 403) — their existence is not secret,
//     but their content is not ready for public consumption. [03-discover §3.2]
//   - Scoring is deterministic: the same inputs always produce the same output. [03-discover §15.26]
//
// # Cross-Domain Dependency
//
// discover:: consumes method:: for methodology display names in quiz results.
// The dependency is expressed as the consumer-defined MethodologyServiceForDiscover
// interface (see ports.go) and bridged in cmd/server/main.go via NewMethodAdapter.
// This avoids importing the method package directly. [ARCH §4.2]
package discover
