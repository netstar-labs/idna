# idna — user guide

## Library

```go
import "github.com/netstar-labs/idna"

a, ok := idna.ToASCII("公司.cn", false)  // ("xn--55qx5d.cn", true)
a, ok  = idna.ToASCII("faß.de", false)   // ("xn--fa-hia.de", true) — non-transitional
a, ok  = idna.ToASCII("_dmarc.x.com", false) // ("", false)          — STD3 rejects underscore
a, ok  = idna.ToASCII("_dmarc.x.com", true)  // ("_dmarc.x.com", true) — loose profile

a, err := idna.ToASCIIErr("公司.cn")  // ("xn--55qx5d.cn", nil) — seam shape for normie.Options.IDNA

u, ok := idna.ToUnicode("xn--55qx5d.cn", false)  // ("公司.cn", true) — U-label for UTS-39 skeletoning
u, ok  = idna.ToUnicode("_dmarc.example.com", true) // ("_dmarc.example.com", true) — loose round-trip

idna.Unicode()  // "15.0.0" — stamp this on every stored A-label / hash
```

### API

| Func | Meaning |
|---|---|
| `ToASCII(host string, allowUnderscore bool) (ascii string, ok bool)` | Canonical A-label. `allowUnderscore=true` relaxes STD3 so `_dmarc`/`_sip._tcp` labels validate. `ok=false` means `host` is not a valid IDNA name — treat it as malformed, not as an empty result. |
| `ToASCIIErr(host string) (string, error)` | Strict-profile `ToASCII`, `(string, error)`-shaped for wiring directly into an injected seam such as `normie`'s `Options.IDNA`. |
| `ToUnicode(host string, allowUnderscore bool) (unicode string, ok bool)` | A-label back to U-label — the pre-punycode form UTS-39 confusable analysis runs on. `allowUnderscore` mirrors `ToASCII`'s profile selection, so a host accepted under the loose profile round-trips instead of failing under STD3 rules it was never validated against. |
| `Unicode() string` | The pinned Unicode version (`"15.0.0"`). Stamp it on every stored A-label/hash so a later re-vendor is detectable as skew, not a silent miss. |

Both `strict` and `loose` profiles are immutable package-level values, safe for
concurrent use — there is no per-call setup cost and no shared mutable state to
guard.

## Wiring into normie

```go
r := normie.Canon(raw, &normie.Options{
    IDNA: idna.ToASCIIErr, // browser-agreeing UTS-46; nil percent-escapes instead (spec-literal, not browser behaviour)
})
```

## Testing

```
GOWORK=off go test ./...
```

`idna_test.go` covers the STD3/loose split (`_dmarc` rejected vs. accepted),
non-transitional deviation-character handling (`faß.de`, `münchen.de`), the
`ToASCIIErr` error path (a `NUL` byte is not a valid host byte), invalid-UTF-8
rejection (`ToASCII`/`ToASCIIErr`/`ToUnicode` all reject malformed bytes rather
than silently substituting U+FFFD), the `ToUnicode` round trip in both the
strict and loose-underscore case, and a pin assertion (`Unicode() ==
"15.0.0"`) that must be updated deliberately alongside any re-vendor.

## Operations: re-vendoring (the only day-2 procedure this repo has)

There is no runtime configuration and nothing to deploy — the one operational
event is a **deliberate** upgrade of the vendored tables, done by hand, never by
`go get -u` (there is nothing for `go get` to update; the vendored code is
first-party under `internal/x/`):

1. Re-copy the runtime `.go` files from the target `golang.org/x/net/idna` and
   `golang.org/x/text/{transform,unicode/bidi,unicode/norm,secure/bidirule}`
   versions into `internal/x/`, rewriting import paths to
   `github.com/netstar-labs/idna/internal/x/...` (see
   [internal/x/README.md](../internal/x/README.md)).
2. Delete every table variant and `//go:build go1.x` selector that isn't the
   new pinned Unicode version, the same way the original vendor did — confirm
   with `grep -r '//go:build go1' internal/x --include='*.go'` returning nothing
   (scope to `.go` files — an unscoped grep also matches this instruction's own text).
3. Bump the version table in [`VENDOR`](../VENDOR) in the same commit as the
   table swap. `Unicode()` reads the vendored `tables15.0.0.go`-generated
   constant directly (no separate copy to keep in sync in `idna.go` itself),
   but `VENDOR` and the actual tables must never disagree.
4. Diff the mapping tables for the ranges that changed and, in the consuming
   corpus, recompute only the non-ASCII rows those ranges could affect —
   `normie`'s `Result.HadNonASCII` (or equivalent) is what scopes that
   recomputation instead of a full re-key.
5. `go test ./...` must pass with the new pin, including the updated
   `TestUnicodePin` expectation.
