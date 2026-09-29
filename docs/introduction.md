# Meet idna — one spelling, pinned for good

*A domain has exactly one A-label here, and it never quietly changes underneath you.*

Two callers in the netstar toolkit need to turn a Unicode host into the punycode
form that goes in a lookup key: `sanitize`, rectifying hosts on ingest, and
`normie`, canonicalizing URLs through its `Options.IDNA` seam. If each pinned its
own copy of `golang.org/x/net/idna`, nothing would stop the two copies from
drifting — a dependency bump on one side, a Go toolchain upgrade on the other —
and the same domain would silently resolve to two different A-labels. Because the
A-label *is* the lookup key, that isn't a crash or an error; it's a stale key that
just stops matching, discovered (if ever) as a mysterious gap in coverage. idna
exists to make that disagreement structurally impossible: one vendored, pinned
implementation, imported by both.

## What it actually is

A thin, owned policy layer — `ToASCII`, `ToASCIIErr`, `ToUnicode`, `Unicode` — over
an in-tree, pruned copy of `golang.org/x/net/idna` and the `golang.org/x/text`
packages it needs, relocated under `internal/x/` and stripped of every Go-version
build selector so a toolchain upgrade can no longer swap the active Unicode table
out from under it. The module has zero external dependencies. It is pinned to
Unicode **15.0.0**, on purpose, independent of whatever Go version compiles it.

## The channel nobody else has

Every other IDNA implementation available to this toolkit ties its Unicode
version to *either* a dependency version *or* a Go build tag — often both at
once, invisibly. This one ties it to neither: the pin lives in exactly one place
(`xidna.UnicodeVersion`, generated from the vendored table file and reported
verbatim by `idna.Unicode()`), advances only when someone deliberately re-vendors,
and is queryable at runtime via `Unicode()` so every stored A-label can carry the
pin it was produced under.

## The honest scope of control

idna answers one question — *what A-label does this Unicode host canonicalize
to, and back* — and nothing else. It does not resolve DNS, does not judge whether
a host is malicious, and does not protect against **coverage** drift: a pinned
table necessarily rejects codepoints Unicode assigns *after* the pin, and no
amount of vendoring changes that. What it does guarantee is **consistency** — every
caller on this pin agrees with every other caller on this pin, always — which is
the property a lookup key actually needs.

*Read next:* [executive-summary.md](executive-summary.md) ·
[architecture.md](architecture.md) · [userguide.md](userguide.md) ·
[../internal/x/README.md](../internal/x/README.md)
