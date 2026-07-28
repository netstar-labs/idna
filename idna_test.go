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
