# Vendored idna dependencies

Pruned copies of `golang.org/x/net/idna` and the `golang.org/x/text` packages it
needs, relocated under this module's `internal/` tree and imported through
the root `idna` package. After this vendor the module has **zero external
dependencies** (`go.mod` requires only the Go version), and idna is **pinned to
Unicode 17 independent of the Go toolchain** (see below).

| Vendored path | Upstream | Version |
| --- | --- | --- |
| `internal/x/net/idna` | `golang.org/x/net/idna` | v0.61.0 |
| `internal/x/text/transform` | `golang.org/x/text/transform` | v0.43.0 |
| `internal/x/text/unicode/bidi` | `golang.org/x/text/unicode/bidi` | v0.43.0 |
| `internal/x/text/unicode/norm` | `golang.org/x/text/unicode/norm` | v0.43.0 |
| `internal/x/text/secure/bidirule` | `golang.org/x/text/secure/bidirule` | v0.43.0 |

Only runtime `.go` files were copied; `*_test.go` and the code generators
(`gen*.go`, `maketables.go`, `triegen.go`, which are `//go:build ignore` and
import ungvendored tooling) were excluded. Two edits were then applied:

1. Import paths and package import comments were rewritten
   `golang.org/x/{net,text}/… → github.com/netstar-labs/idna/internal/x/{net,text}/…`.
2. Upstream ships two table variants in idna/bidi/norm: `tables15.0.0.go`
   (`//go:build !go1.27`) and `tables17.0.0.go` (`//go:build go1.27`). The
   15.0.0 variants were deleted and the selector stripped from the 17.0.0 files
   that remain, so canonicalization no longer varies by toolchain.

The upstream BSD `LICENSE` and `PATENTS` are kept in `internal/x/net` and
`internal/x/text`.

## Why freeze idna this way

`sanitize` stores the canonical punycode A-label as a **durable key** — telemetry
is validated and canonicalized on the way in, persisted, then looked up later by
re-canonicalizing partner queries and published. That contract only holds if a
given input maps to the same A-label *indefinitely*. If canonicalization shifts
between the write and a later read, the same domain canonicalizes to a different
key and silently fails to match the stored record (records split; lookups miss).

idna canonicalization can drift from two directions, and pinning a module version
alone does not stop both:

1. **Dependency upgrades.** A routine `go get -u` of `x/net`/`x/text` can change
   mapping tables or processing. Vendoring makes the code first-party, so it
   changes only when we deliberately re-copy it.

2. **The Go toolchain's Unicode version.** `x/net/idna` selects its Unicode
   tables by *Go build tag*, not by module version — e.g. upstream `tables17.0.0.go`
   is `//go:build go1.27` — so merely upgrading the compiler can swap the active
   Unicode table and change canonicalization with no dependency change at all. To
   remove that coupling entirely, the lower-version table variants were deleted
   and the `//go:build go1.x` selectors stripped from the Unicode-17 files that
   remain. No version selector is left in the vendored stack
   (`grep -r '//go:build go1' internal/x --include='*.go'` → nothing — scoped to
   `.go` files, since an unscoped grep also matches this sentence), so **every toolchain
   compiles Unicode 17** — the mapping is pinned to Unicode 17 outright, not
   merely for Go ≥ 1.27. (Filenames like `tables17.0.0.go` are kept as-is; they
   just record the Unicode/Go era the code came from.)

Combined with non-transitional (UTS-46) processing, this makes the A-label a
stable, reproducible key for as long as this vendored copy is in place.

## Updating

Because this is a pruned snapshot (not a verbatim mirror), an update is a
re-vendor from scratch, not a diff against upstream. Treat it as a
**canonicalization migration**, not a routine bump:

1. Re-copy runtime files from the chosen `x/net`/`x/text` versions, re-apply the
   import-path rewrite, then re-prune to the target Unicode version (delete the
   lower-version table variants and strip their `//go:build go1.x` selectors).
2. Run the suite. `sanitize`'s tests pin known outputs (e.g. `faß.de →
   xn--fa-hia.de`); any diff means the canonical form of some inputs changed.
3. If outputs changed, re-canonicalize stored keys before shipping — otherwise
   old and new keys will not match.

## Re-vendor log

### 2026-10-12: Unicode 15.0.0 → 17.0.0 (`x/net` v0.40.0 → v0.61.0, `x/text` v0.25.0 → v0.43.0)

**Local patches.** None were lost. Every previously vendored `.go` file was compared with its
upstream source at the old pin, ignoring import paths and build tags, and all were identical.
Upstream has also consolidated files since then: `idna10.0.0.go`, `go118.go`, `trie13.0.0.go`
and `bidirule10.0.0.go` are gone, folded into `idna.go`, `trie.go` and `bidirule.go`.

**What changed.** Each of the 1,111,936 Unicode scalar values from U+0080 up was run through
`ToASCIIErr`, both as a bare label and after an ASCII letter, under both pins. 10,990 code
points give a different result:

| Change | Code points | Effect on stored A-labels |
|---|---|---|
| Valid under both pins, different A-label | **1: U+1E9E ẞ** (capital sharp s), which now maps to ß (`xn--zca`), not "ss" | **Re-keys.** A stored label that contained ẞ was keyed with "ss" and is now keyed with ß. Recompute the non-ASCII rows that contain U+1E9E. |
| Rejected under 15, valid under 17 | 10,684, in 108 ranges: new Unicode 16/17 characters; capitals such as U+10A0–U+10C5, U+2132 and U+04C0 now mapped; `≠ ≮ ≯` now valid | None. These inputs could not have been stored before. |
| Rejected under 15, **ignored** under 17 | Among the above: invisible format characters (U+2061–U+2063, U+206A–U+206F, U+115F, U+1160, U+3164, U+FFA0, U+180E, U+17B4, U+17B5) | None for stored keys. **But** `exam⁡ple.com` now canonicalizes to `example.com` instead of erroring, and a host made only of such characters canonicalizes to an empty label. Empty labels already pass through, as `ToASCII` documents, so consumers must keep rejecting them. |
| Rejected under both pins, different error-path output | 305 | None. |

**Consumer action.**
- **`sanitize`** pins `idna` v0.1.1. Bumping it takes the new pin.
- **Recompute only the stored A-labels whose source contains U+1E9E.** `normie`'s `HadNonASCII`
  narrows the scan.
- **Confirm that an all-invisible host is still rejected downstream** as an empty label.

`TestUnicode17Changes` pins these behaviours.
