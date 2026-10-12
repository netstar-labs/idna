# idna — architecture

idna is two layers: a small, owned policy file at the root, and a vendored
UTS-46 engine underneath it. No I/O, no network, no allocation beyond what the
underlying transform requires.

## Data flow

```
host ──▶ strict/loose Profile ──▶ xidna.Profile.ToASCII/ToUnicode ──▶ A-label / U-label
              (idna.go)                (internal/x/net/idna)
                                              │
                                   internal/x/text/{transform,unicode/bidi,
                                   unicode/norm,secure/bidirule}
```

`idna.go` never touches Unicode tables directly — it configures two immutable,
concurrency-safe `xidna.Profile` values once at package init and dispatches every
call to one of them.

## The two profiles (the load-bearing design)

```go
strict = xidna.New(xidna.MapForLookup(), xidna.Transitional(false))
loose  = xidna.New(xidna.MapForLookup(), xidna.Transitional(false), xidna.StrictDomainName(false))
```

| | `strict` | `loose` |
|---|---|---|
| used by | `ToASCII(host, false)`, `ToASCIIErr` | `ToASCII(host, true)` |
| STD3 ASCII rules | enforced (letters/digits/hyphen only) | relaxed (`StrictDomainName(false)`) |
| underscore labels | rejected (`_dmarc`, `_sip._tcp` fail) | accepted |
| transitional processing | off (both) | off (both) |

Both use **non-transitional** (UTS-46) processing, so deviation characters
resolve the way current browsers and registries handle them — `faß.de` maps to
`xn--fa-hia.de`, not the transitional `fass.de`. Non-transitional is the single
processing mode this package exposes; there is no caller-facing switch for it,
because a mixed-mode caller is exactly the kind of divergence idna exists to
prevent.

## The vendored engine (`internal/x/`)

A pruned, relocated copy of `golang.org/x/net/idna` v0.61.0 and the
`golang.org/x/text` v0.43.0 packages it needs (`transform`, `unicode/bidi`,
`unicode/norm`, `secure/bidirule`). Three changes were made to the upstream source
on import:

1. **Import paths rewritten** — `golang.org/x/{net,text}/… →
   github.com/netstar-labs/idna/internal/x/{net,text}/…`.
2. **Every table variant but Unicode 17's, and every `//go:build go1.x` selector, was
   deleted** (upstream v0.61.0/v0.43.0 ship `tables15.0.0.go` under `!go1.27` and
   `tables17.0.0.go` under `go1.27`, across idna/bidi/norm). Upstream picks its Unicode table by **Go build tag**, not by
   module version — a compiler upgrade alone can silently swap tables even with
   no dependency change. Deleting every variant but the Unicode-17 one closes
   that off entirely: `grep -r '//go:build go1' internal/x --include='*.go'` returns
   nothing (scoped to `.go` files — an unscoped grep also matches this sentence), so
   every toolchain that compiles this module compiles the same table.
3. **`unicode16` pinned to `true`** in `net/idna/idna.go`. Upstream derives it from the
   toolchain's `unicode.Version`, a second, tag-free coupling to the Go release. Under Go 1.24 it
   ran the pre-16 algorithm over the 17.0.0 tables and stopped enforcing STD3. Pinned, every
   toolchain runs the same algorithm over the same table: the 2026-10-12 re-vendor checked
   byte-identical output for every code point under Go 1.24 and 1.27.

Only runtime `.go` files were vendored; upstream's own tests and code generators
(`gen*.go`, `maketables.go`, `triegen.go` — all `//go:build ignore`) were left
behind. The upstream BSD license is preserved under `internal/x/net` and
`internal/x/text`.

## The drift model

The A-label is a lookup key, not a value checked once and discarded, so a
changed mapping doesn't error — it produces a stale key. Two properties, kept
deliberately distinct:

- **Consistency** (internal, solved) — every caller on this pin resolves a given
  host identically, forever, because the mapping only changes when someone
  deliberately re-vendors.
- **Coverage** (external, open) — a pinned table rejects any codepoint Unicode
  assigns after the pin. Vendoring cannot fix this; it can only make the
  trade-off explicit and the pin visible (`Unicode()`).

A re-vendor is a deliberate migration, not an update: bump `VENDOR` and the
generated table file (`Unicode()` reads `xidna.UnicodeVersion` from it directly,
so there is no separate copy to keep in sync), diff the mapping tables for the
ranges that actually changed,
and recompute only the non-ASCII rows those ranges affect — guided by
`Result.HadNonASCII`-style flags in a consuming corpus (see `normie`'s
architecture doc for the caller-side half of this).

## Trade-offs / open items

- **No differential harness against a live, unpinned `x/net/idna`.** The test
  suite (`idna_test.go`) is a fixed table of known-good vectors, not a
  generator that would catch a table-pruning mistake at vendor time. A
  transplant bug would currently surface as a wrong A-label in production, not
  a test failure.
- **Coverage risk is structural, not a bug to fix.** Any pinned implementation
  carries it; the mitigation is the stamp-and-migrate discipline above, not a
  code change here.
- **Single Unicode version, by design.** There is no runtime option to select
  an older table — intentional, since a caller-selectable version would
  reintroduce the exact two-pin divergence this package exists to remove.
