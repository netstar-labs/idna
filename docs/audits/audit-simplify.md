# Audit — simpler pathways (auditor A) + least-code

Scope: owned root code only (`idna.go`, `doc.go`, `idna_test.go`); `internal/x/` is
vendored/frozen, out of scope for simplification.

## Result: no material simplification findings in the owned code

The owned layer (~110 lines across three files) is already close to minimal: four
thin wrapper functions plus a two-profile variable and, previously, one hand-copied
constant (see the least-code finding below, now fixed).

**Considered and explicitly NOT a finding (load-bearing):** `ToASCII`/`ToUnicode`'s
`if err != nil { return "", false }; return x, true` shape looks collapsible to
`return x, err == nil` — checked against the vendored `Profile.ToASCII`/`ToUnicode`
doc comment ("If an error is encountered it will return an error and a (partially)
processed result") and confirmed on error the vendored call can return a non-empty,
partially-mangled label alongside the error. The explicit early return guarantees
callers never observe a malformed partial label when `ok == false` — this matters
because the result is used as a lookup key. Collapsing it would silently change
behavior on the error path. Correctly left as-is.

**Considered and NOT worth doing (marginal, near-zero net LOC):** the same
`err→(string,bool)` shape appears in exactly two places (`ToASCII`, `ToUnicode`, not
`ToASCIIErr` which intentionally passes `(string, error)` straight through). A shared
helper would trade ~10 inline lines for a ~5-7 line helper plus two call sites — a net
saving of roughly 1-3 LOC at the cost of one more symbol in a 4-function file. Not
recommended; noted for visibility per the audit brief, not acted on. (Independently
surfaced by auditor B — see `audit-dedup.md` — same conclusion from the dedup lens.)

## Fixed in this pass — least-code (reuse)

`idna.go` hand-maintained `const unicodeVersion = "15.0.0"`, duplicating a value the
vendored, generated file `internal/x/net/idna/tables15.0.0.go` already exports as
`const UnicodeVersion = "15.0.0"`. A re-vendor updates the generated constant
automatically but would not touch the hand-copied one — exactly the "stale key,
silent miss" hazard the package's own `doc.go` names as its central risk, except here
the drift vector would be the owned wrapper itself, not table content. Compounding it:
`idna_test.go`'s `TestUnicodePin` compares `Unicode()` against a second hardcoded
`"15.0.0"` literal, so a forgotten bump in `idna.go` would still pass that test, since
both hardcoded copies would drift together.

**Fix applied**: `Unicode()` now returns `xidna.UnicodeVersion` directly; the
hand-copied constant is deleted. Zero behavior change (same value, same call
signature) — reuse over a hand-copy, per the least-code ladder. Verified: `go build`,
`go vet`, `go test ./...` (including `TestUnicodePin`) all green after the change.
