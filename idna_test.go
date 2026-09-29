package idna

import "testing"

func TestToASCII(t *testing.T) {
	cases := []struct {
		host            string
		allowUnderscore bool
		want            string
		ok              bool
	}{
		{"example.com", false, "example.com", true},
		{"公司.cn", false, "xn--55qx5d.cn", true},                  // CJK -> punycode
		{"faß.de", false, "xn--fa-hia.de", true},                 // non-transitional (browser/registry behaviour)
		{"münchen.de", false, "xn--mnchen-3ya.de", true},         // umlaut
		{"_dmarc.example.com", false, "", false},                 // underscore rejected under STD3
		{"_dmarc.example.com", true, "_dmarc.example.com", true}, // ...allowed under loose
		{"", false, "", true},                                    // empty is a valid (empty) name
	}
	for _, c := range cases {
		got, ok := ToASCII(c.host, c.allowUnderscore)
		if ok != c.ok || got != c.want {
			t.Errorf("ToASCII(%q, %v) = (%q, %v), want (%q, %v)", c.host, c.allowUnderscore, got, ok, c.want, c.ok)
		}
	}
}

func TestToASCIIErr(t *testing.T) {
	if a, err := ToASCIIErr("公司.cn"); err != nil || a != "xn--55qx5d.cn" {
		t.Errorf("ToASCIIErr(公司.cn) = (%q, %v), want (xn--55qx5d.cn, nil)", a, err)
	}
	if _, err := ToASCIIErr("a\x00b.com"); err == nil { // NUL is not a valid host byte
		t.Error("ToASCIIErr(NUL) = nil error, want error")
	}
}

// A lone illegal UTF-8 byte must not silently encode to a plausible-looking
// A-label: the vendored profile substitutes U+FFFD and reports no error for it
// internally (the same code path unassigned codepoints take), so ToASCII/
// ToASCIIErr must catch it themselves rather than forward that silent success.
func TestToASCIIRejectsInvalidUTF8(t *testing.T) {
	host := "exa" + string([]byte{0x80}) + "mple.com"
	if a, ok := ToASCII(host, false); ok {
		t.Errorf("ToASCII(invalid UTF-8) = (%q, true), want ok=false", a)
	}
	if a, err := ToASCIIErr(host); err == nil {
		t.Errorf("ToASCIIErr(invalid UTF-8) = (%q, nil), want an error", a)
	}
	// control: a real, validly-encoded U+FFFD must still be rejected too
	// (proves the guard doesn't just special-case the replacement rune).
	if _, ok := ToASCII("exa�mple.com", false); ok {
		t.Error("ToASCII(real U+FFFD) = ok=true, want ok=false")
	}
}

// ToUnicode shares ToASCII's UTF-8 guard.
func TestToUnicodeRejectsInvalidUTF8(t *testing.T) {
	host := "exa" + string([]byte{0x80}) + "mple.com"
	if u, ok := ToUnicode(host); ok {
		t.Errorf("ToUnicode(invalid UTF-8) = (%q, true), want ok=false", u)
	}
}

func TestToUnicodeRoundTrip(t *testing.T) {
	// A-label -> U-label, the form UTS-39 skeletoning consumes.
	if u, ok := ToUnicode("xn--55qx5d.cn"); !ok || u != "公司.cn" {
		t.Errorf("ToUnicode(xn--55qx5d.cn) = (%q, %v), want (公司.cn, true)", u, ok)
	}
}

func TestUnicodePin(t *testing.T) {
	if Unicode() != "15.0.0" {
		t.Errorf("Unicode() = %q, want 15.0.0 (bump in lockstep with a re-vendor)", Unicode())
	}
}
