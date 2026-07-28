// Package idna is the shared UTS-46 host canonicalization primitive for the
// netstar toolkit: map a Unicode host to its punycode A-label (the lookup key)
// and back. It is a thin, owned policy layer over a vendored, pinned copy of
// golang.org/x/net/idna (and the golang.org/x/text packages it needs), so the
// module has zero external dependencies and the mapping is frozen against Go
// toolchain and upstream drift — see internal/x/README.md.
//
// One implementation, one pin. Both consumers use it: sanitize (host rectify +
// TLD/apex) and normie (URL canonicalization, wired through its Options.IDNA
// seam). Vendoring it in one place is the point — two independently-pinned copies
// could resolve the same host to different A-labels, which is a stale-key /
// silent-miss hazard across ingest and query.
//
// # The drift model (why the pin and the stamp matter)
//
// The A-label is a lookup key, so a changed mapping produces a stale key, not a
// visible error. Two properties, kept distinct:
//
//   - Consistency (internal): ingest and query on the same pin always agree. The
//     pin delivers this outright; a consistently-pinned system does not drift.
//   - Coverage (external): a pinned table rejects codepoints assigned after the
//     pin. That is the real, ongoing risk, and it is a coverage problem, not a
//     consistency one.
//
// Stamp [Unicode] onto every stored A-label / hash so a re-vendor is detectable as
// skew during a rolling deploy, and migrate deliberately (diff the mapping tables
// for changed ranges, recompute only the affected non-ASCII rows). This is the
// UTS-46 half; UTS-39 confusable skeletoning (which is NOT a lookup key and
// updates freely) is a separate concern in a separate package.
package idna
