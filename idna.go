package idna

import (
	"errors"
	"unicode/utf8"

	xidna "github.com/netstar-labs/idna/internal/x/net/idna"
)

// errInvalidUTF8 is returned when host contains a byte sequence that isn't valid
// UTF-8. The vendored profile does not treat this as an error itself — an illegal
// leading byte and an unassigned Unicode codepoint share the same silent
// U+FFFD-substitution path internally (see internal/x/net/idna's validateAndMap),
// so a malformed host would otherwise encode to a plausible-looking A-label with
// ok=true, which silently violates the "malformed input" contract below.
var errInvalidUTF8 = errors.New("idna: host is not valid UTF-8")

// strict and loose are the shared idna profiles. Both use non-transitional
// (UTS-46) processing, so deviation characters resolve as browsers and registries
// now handle them (faß.de -> xn--fa-hia.de, not fass.de). strict enforces STD3
// ASCII rules (letters, digits, hyphen); loose relaxes STD3 so underscore labels
// (_dmarc, _sip._tcp) validate. Both are immutable and safe for concurrent use, so
// a single instance of each serves every caller.
var (
	strict = xidna.New(xidna.MapForLookup(), xidna.Transitional(false))
	loose  = xidna.New(xidna.MapForLookup(), xidna.Transitional(false), xidna.StrictDomainName(false))
)

// Unicode reports the Unicode version the vendored UTS-46 tables are pinned to
// (internal/x/net/idna/tables15.0.0.go, bumped in lockstep with a re-vendor).
// Stamp it onto every stored A-label / hash / artifact so a later re-vendor is
// detectable as skew rather than a silent miss — the A-label is a lookup key and a
// changed mapping produces a stale one (see the drift model in the package doc).
func Unicode() string { return xidna.UnicodeVersion }

// ToASCII converts host to its canonical punycode A-label. When allowUnderscore is
// true, STD3 ASCII rules are relaxed so underscore (and other non-LDH ASCII that
// UTS-46 permits) validate. ok is false when the label is not a valid idna name,
// in which case the caller should treat host as malformed.
//
// ok=true does not by itself imply an RFC 1035-conformant name: neither profile
// enforces the 63-octet label / 253-octet name length limit, and empty or
// leading/consecutive-dot labels (".", "..example.com") pass through unchanged. A
// caller that needs those invariants must check them separately.
func ToASCII(host string, allowUnderscore bool) (ascii string, ok bool) {
	if !utf8.ValidString(host) {
		return "", false
	}
	p := strict
	if allowUnderscore {
		p = loose
	}
	a, err := p.ToASCII(host)
	if err != nil {
		return "", false
	}
	return a, true
}

// ToASCIIErr is the (string, error) form of [ToASCII] under the strict (STD3)
// profile, shaped to wire directly into an injected seam such as normie's
// Options.IDNA (func(host string) (string, error)). Use [ToASCII] with
// allowUnderscore for the loose profile.
func ToASCIIErr(host string) (string, error) {
	if !utf8.ValidString(host) {
		return "", errInvalidUTF8
	}
	return strict.ToASCII(host)
}

// ToUnicode converts an A-label back to its U-label (the pre-punycode Unicode
// form) — the input UTS-39 confusable analysis (skeletoning) runs on. allowUnderscore
// mirrors [ToASCII]'s profile selection, so a host ToASCII(host, true) accepted under
// the loose profile (_dmarc, _sip._tcp) round-trips here instead of failing under
// strict STD3 rules it was never validated against. ok is false when host is not a
// valid A-label under the selected profile.
func ToUnicode(host string, allowUnderscore bool) (unicode string, ok bool) {
	if !utf8.ValidString(host) {
		return "", false
	}
	p := strict
	if allowUnderscore {
		p = loose
	}
	u, err := p.ToUnicode(host)
	if err != nil {
		return "", false
	}
	return u, true
}
