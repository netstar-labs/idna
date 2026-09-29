# idna

Shared **UTS-46** host canonicalization for the netstar toolkit — map a Unicode
host to its punycode **A-label** (the lookup key) and back — over a **vendored,
pinned** copy of `golang.org/x/net/idna`. **Zero external dependencies.**

```go
idna.ToASCII("公司.cn", false)   // ("xn--55qx5d.cn", true)
idna.ToASCII("faß.de", false)    // ("xn--fa-hia.de", true)   — non-transitional
idna.ToASCII("_dmarc.x.com", false) // ("", false)            — STD3 rejects underscore
idna.ToASCII("_dmarc.x.com", true)  // ("_dmarc.x.com", true) — loose profile

idna.ToASCIIErr("公司.cn")             // ("xn--55qx5d.cn", nil) — seam shape for normie.Options.IDNA
idna.ToUnicode("xn--55qx5d.cn", false) // ("公司.cn", true)      — U-label for UTS-39 skeletoning
idna.Unicode()                         // "15.0.0"               — the pin; stamp it on stored A-labels
```

## Documentation

- **Start here** — [docs/introduction.md](docs/introduction.md) ·
  [docs/executive-summary.md](docs/executive-summary.md)
- **Deep dive** — [docs/architecture.md](docs/architecture.md)
- **Operations** — [docs/userguide.md](docs/userguide.md) ·
  [internal/x/README.md](internal/x/README.md) (vendor + re-vendor procedure)

## Why this exists

One implementation, one pin, both consumers. `sanitize` (host rectify + TLD/apex)
and `normie` (URL canonicalization, via its `Options.IDNA` seam) both need UTS-46.
Two independently-pinned copies could resolve the same host to different A-labels —
a stale-key / silent-miss hazard across ingest and query. This repo vendors
`x/net/idna` (+ the `x/text` packages it needs) **in-tree** under
[`internal/x`](internal/x/README.md), pinned to **Unicode 15.0.0** independent of
the Go toolchain, behind a thin owned policy layer.

## The drift model

The A-label is a **lookup key**, so a changed mapping produces a stale key, not a
visible error. Keep two properties distinct:

- **Consistency** (internal) — ingest and query on the same pin always agree; the
  pin delivers this outright.
- **Coverage** (external) — a pinned table rejects codepoints assigned *after* the
  pin. This is the real, ongoing risk.

Stamp `Unicode()` onto every stored A-label / hash so a re-vendor is detectable as
skew during a rolling deploy, and migrate deliberately (diff the mapping tables for
changed ranges, recompute only the affected non-ASCII rows). This is the UTS-46
half; UTS-39 confusable skeletoning (never a lookup key, updates freely) is a
separate package.

## Layout

| Path | Purpose |
|---|---|
| [idna.go](idna.go) | the policy layer — `ToASCII` (strict/loose STD3), `ToASCIIErr`, `ToUnicode`, `Unicode` |
| [internal/x/](internal/x/README.md) | vendored, pruned `x/net/idna` + `x/text`, pinned Unicode 15.0.0 |

Go module `github.com/netstar-labs/idna`. Standard library only (the `x/*` code is
vendored, not a dependency). Build with `GOWORK=off`.
