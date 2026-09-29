# Audit — correctness + optimization (auditor C)

Scope: owned root code only (`idna.go`, `doc.go`, `idna_test.go`); `internal/x/` read
for root-cause tracing but is vendored/frozen and out of scope for direct edits.

## CONFIRMED

### 1. [BUG] Malformed (invalid) UTF-8 input is silently accepted, not rejected

`idna.go` — `ToASCII` / `ToASCIIErr`. Doc comment promises: *"ok is false when the
label is not a valid idna name, in which case the caller should treat host as
malformed."*

**Repro** (independently reproduced twice — original + adversarial skeptic, byte-for-byte identical):

```
ToASCII("exa\x80mple.com", false) = ("xn--example-2e14b.com", true)
ToASCIIErr("exa\x80mple.com")     = ("xn--example-2e14b.com", nil)

# control: a real, validly-encoded U+FFFD in the same position IS rejected —
# proves this is a distinct code path, not "FFFD is always allowed"
ToASCII("exa�mple.com", false) = ("", false)
ToASCIIErr(...)                      = ("xn--example-2e14b.com", "idna: disallowed rune U+FFFD")
```

Also silently accepted: lone `0x80`/`0xC0`/`0xC1`/`0xFF` at string start/end.
Correctly rejected (negative control): truncated multi-byte sequences at end of
string (lone `0xC2`, or `0xE0 0x80`) — so it is not "all malformed UTF-8 passes,"
specifically illegal leading bytes.

**Root cause** (traced in `internal/x/net/idna`, vendored, frozen): `idnaTrie.lookupString`
(`tables15.0.0.go`) classifies an illegal leading byte as `v=0, sz=1`, landing in
category `unknown` in `validateAndMap` (`idna10.0.0.go`) — the same category used for
unassigned Unicode codepoints. That branch appends U+FFFD and never sets `err`. Only
the genuinely-truncated (`sz=0`) case sets an error. The owned wrapper in `idna.go`
never independently checks `utf8.ValidString(host)`.

**Why this isn't a scope/expectations mismatch** (checked adversarially): the package's
own existing test (`idna_test.go`, `TestToASCIIErr`) already asserts a NUL byte must
produce an error ("NUL is not a valid host byte") — i.e. the authors already treat
`ToASCIIErr` as a byte-validity gate, not merely an IDNA-semantic validator. Silent
FFFD-substitution is standard Go convention at the iteration-primitive level (`range`,
`utf8.DecodeRuneInString`), but those primitives make no "ok" promise; `idna.go`'s
public API does, so the leak is real.

**Recommended fix**: add a `utf8.ValidString(host)` guard in `ToASCII`/`ToASCIIErr`
(and `ToUnicode`, for the same reason) before calling into the vendored profile. Cheap
(stdlib only), and the fix lives entirely in owned code — no vendor patch needed.

**Verdict: CONFIRMED** (independent adversarial skeptic could not refute; reproduced
twice from different scratch modules, negative controls rule out alternative
explanations). **Fixed — issue #4, commit on `audit/a1-idna`.** Regression test added
and sabotage-verified (reverting the guard makes the test fail with an attributable
message; restoring passes). This is an accepted-input behavior change on the public
API — previously-accepted malformed input is now rejected.

---

### 2. [BUG] `ToUnicode` cannot round-trip a host `ToASCII` accepted under the loose profile

`idna.go` — `ToUnicode` hardcodes the `strict` profile with no `allowUnderscore`
parameter, unlike its sibling `ToASCII`. Doc comment promises: *"ok is false when host
is not a valid A-label"* — implying `ok=true` whenever host *is* one.

**Repro** (independently reproduced twice):

```
ToASCII("_dmarc.example.com", true)  = ("_dmarc.example.com", true)   # loose accepts it
ToUnicode("_dmarc.example.com")      = ("", false)                     # strict rejects the same string

ToUnicode("example.com")             = ("example.com", true)           # plain ASCII, no underscore: succeeds
ToUnicode("xn--55qx5d.cn")           = ("公司.cn", true)                # real punycode: succeeds
```

Failure is precisely and only STD3/underscore-related — `ToUnicode` is not
"punycode-only"; it already passes through any strict-valid plain-ASCII host.
`_dmarc`/`_sip._tcp` is the package's own stated motivating case for the loose
profile (README, doc.go), so this is the one documented use case the asymmetry
breaks.

**Why this isn't a designed restriction** (checked adversarially): whitebox-tested the
package's own unexported `loose` var directly — `loose.ToUnicode("_dmarc.example.com")`
succeeds cleanly with no vendor changes needed. `(*xidna.Profile).ToUnicode` already
respects `useSTD3Rules` symmetrically with `ToASCII` upstream; nothing in the vendored
engine structurally prevents a loose `ToUnicode`. No doc anywhere (`doc.go`, README,
`docs/*.md`) scopes `ToUnicode`'s contract more narrowly than "valid A-label," and the
only existing test (`TestToUnicodeRoundTrip`) exercises only the punycode case.

**Downstream relevance**: a package documenting its input contract as "a U-label, as
`idna.ToUnicode` yields" (e.g. a confusable-skeleton package) would get a silent dead
end for exactly the host class the loose profile exists to admit. No current in-tree
caller chains loose-`ToASCII` output into `ToUnicode` today (checked), so nothing is
broken in production right now — but the gap is real and would bite the first caller
that does.

**Recommended fix**: `ToUnicode(host string, allowUnderscore bool)`, mirroring
`ToASCII`'s signature exactly, selecting `strict`/`loose` the same way.

**Verdict: CONFIRMED** (independent adversarial skeptic could not refute).
**Fixed — issue #5, commit on `audit/a1-idna`.** Regression test added and
sabotage-verified. This is a public function-signature change — breaking for every
existing caller — flagged for human sign-off per house review discipline rather than
self-merged.

## MINOR (documented, not changed, in this pass)

### 3. `VerifyDNSLength` is not enabled on either profile

A 64-octet label (over the RFC 1035 63-octet limit) round-trips with `ok=true`, no
truncation or error — verified directly (`ToASCII` on a 64-`a` label + `.com`).
Consequence of the specific option set chosen (`MapForLookup()` is opt-in on length by
default upstream); not necessarily wrong, but was undocumented. **Now documented** on
`ToASCII`'s doc comment (this pass) rather than behavior-changed, since enabling it
would itself be a behavior change on the public API needing separate confirmation.

### 4. Leading/consecutive/empty-dot labels pass through unchanged

`ToASCII("..example.com", false)` → `("..example.com", true)`; `ToASCII(".", false)` →
`(".", true)` — verified directly. Same root cause as #3 (`RemoveLeadingDots` never
passed). **Now documented** alongside #3, not behavior-changed.

## Clean checks performed (verified, not read-and-assumed)

- **Concurrency safety** of the package-level `strict`/`loose` values: traced
  `xidna.Profile` (plain struct, fields set once in `New()`, never mutated by
  `process()`); empirically hammered with 50 goroutines × 2000 iterations under
  `go run -race` — no race. The doc's concurrency claim holds.
- Empty host (`ToASCII("", false)` → `("", true)`) traced by hand through `process()` —
  matches `idna_test.go`.
- NUL byte via both `ToASCII`/`ToASCIIErr` — correctly rejected, matches
  `idna_test.go`.
- No allocation introduced by the owned wrapper layer itself; all allocation cost is
  inside the vendored `process()` (out of scope).
