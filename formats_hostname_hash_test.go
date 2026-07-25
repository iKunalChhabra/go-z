package zod

import (
	"regexp"
	"strings"
	"testing"
)

func TestHostnameFormat(t *testing.T) {
	h := String().Hostname()
	for _, s := range []string{
		"localhost",
		"example.com",
		"sub.example.com",
		"a-b-c.example.com",
		"192.168.1.1",
		"xn--d1acj3b.com",
		"www.google.com",
	} {
		if _, err := h.Parse(s); err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
	}
	for _, s := range []string{
		"",
		"example..com",
		"example-.com",
		"-example.com",
		"example_com",
		"example.com:8080",
		"http://example.com",
		"exa mple.com",
	} {
		if h.SafeParse(s).Success {
			t.Fatalf("expected fail for %q", s)
		}
	}
}

func TestHashFormat(t *testing.T) {
	md5 := String().Hash("md5")
	md5.MustParse("5d41402abc4b2a76b9719d911017c592")
	md5.MustParse("5D41402ABC4B2A76B9719D911017C592")
	if md5.SafeParse("5d41402abc4b2a76b9719d911017c59").Success {
		t.Fatal("short md5 should fail")
	}
	if md5.SafeParse("not-hex!!!!!!!!!!!!!!!!!!!!!!!!!!!!").Success {
		t.Fatal("non-hex should fail")
	}

	sha256 := String().Hash("sha256")
	sha256.MustParse("a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3")
	if sha256.SafeParse(strings.Repeat("a", 32)).Success {
		t.Fatal("wrong length")
	}

	res := String().Hash("md5", "Invalid MD5 hash").SafeParse("invalid")
	if res.Success || res.Error.Issues[0].Message != "Invalid MD5 hash" {
		t.Fatalf("%+v", res.Error)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("unknown alg should panic")
		}
	}()
	_ = FormatHash("blake2b")
}

func TestURLOptsHostnameProtocolPattern(t *testing.T) {
	// classic/tests/url.test.ts — pattern on hostname failure
	s := String().URL(URLOpts{Hostname: regexp.MustCompile(`^example\.com$`)})
	res := s.SafeParse("http://example.org/")
	if res.Success {
		t.Fatal("expected fail")
	}
	if res.Error.Issues[0].Pattern != `^example\.com$` {
		t.Fatalf("pattern=%q", res.Error.Issues[0].Pattern)
	}
}

func TestFormatHttpURLDirect(t *testing.T) {
	p := AcquirePayload("https://example.com")
	defer ReleasePayload(p)
	FormatHttpURL().Fn(p)
	if len(p.Issues) != 0 {
		t.Fatalf("%+v", p.Issues)
	}

	p2 := AcquirePayload("ftp://example.com")
	defer ReleasePayload(p2)
	FormatHttpURL().Fn(p2)
	if len(p2.Issues) == 0 || p2.Issues[0].Format != "url" {
		t.Fatalf("%+v", p2.Issues)
	}
}

func TestStringFormatRegexpAndPredicate(t *testing.T) {
	re := regexp.MustCompile(`^abc$`)
	s := StringFormat("tag", re)
	s.MustParse("abc")
	if s.SafeParse("ab").Success {
		t.Fatal("expected fail")
	}

	pred := StringFormat("even-len", func(v string) bool { return len(v)%2 == 0 })
	pred.MustParse("ab")
	if pred.SafeParse("a").Success {
		t.Fatal("expected fail")
	}
}
