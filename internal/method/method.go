// Package method implements the Methodology Configuration domain (method::).
//
// The methodology domain owns methodology definitions, the master tool catalog,
// and tool activation mappings that determine which learning tools are available
// to each family. It is the platform's cross-cutting configuration domain.
//
// Key rules:
//   - method_ tables are global config — NO RLS, NO FamilyScope on queries [ARCH §5.1]
//   - No methodology name branching anywhere — all behavior is config-driven [CODING §5.1]
//   - method:: validates methodology IDs; iam:: persists them (bounded context) [02-method §11.2]
//
// See specs/domains/02-method.md for full specification.
package method
