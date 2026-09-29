# idna — executive summary

**What it is.** A pure-Go, zero-external-dependency library that maps a Unicode
host to its canonical punycode A-label and back (UTS-46), vendored in-tree from
`golang.org/x/net/idna` and pinned to Unicode 15.0.0 independent of the Go
toolchain.

**Why it exists.** Two consumers in the netstar toolkit — `sanitize` (host
rectify + TLD/apex) and `normie` (URL canonicalization, via its `Options.IDNA`
seam) — both need to turn a Unicode host into the same lookup key. If each
depended on its own copy of `x/net/idna`, a routine dependency bump on one side
or a Go toolchain upgrade on the other could silently change which A-label a
given host maps to. Because the A-label is a stored, re-derived lookup key —
not a value checked once and forgotten — that divergence doesn't error, it just
stops matching: records split, lookups miss, and nothing logs it. Vendoring one
pinned implementation and making both consumers import it removes the
possibility outright.

**The core discipline — consistency vs. coverage, never conflated.** A pinned
mapping guarantees **consistency**: every caller on this pin resolves a given
host to the same A-label, unconditionally. It cannot guarantee **coverage**: a
codepoint Unicode assigns *after* the pin will be rejected until a deliberate
re-vendor. That is a real, ongoing risk, and it is a coverage problem, not a
consistency one — treating it as the latter is how a pin gets "fixed" by
silently floating, which reintroduces the exact hazard the vendoring exists to
remove.

**Boundaries.** idna canonicalizes a host string and reports the pin it did it
under; it does not resolve DNS, does not judge maliciousness, and UTS-39
confusable skeletoning (which is *not* a lookup key and is meant to update
freely) is deliberately a separate, unrelated concern.

**Shape.** `ToASCII(host, allowUnderscore) (string, bool)` — strict STD3 or a
loose profile that admits underscore labels (`_dmarc`, `_sip._tcp`).
`ToASCIIErr(host) (string, error)` is the same strict profile shaped as an
injectable seam (e.g. `normie`'s `Options.IDNA`). `ToUnicode(host) (string,
bool)` recovers the U-label. `Unicode() string` reports the pin ("15.0.0") to
stamp on every stored artifact, so a future re-vendor is detectable as skew
during a rolling deploy instead of a silent, unmarked change in meaning.
