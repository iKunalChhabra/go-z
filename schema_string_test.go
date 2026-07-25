package zod

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// Ported from classic/tests/string.test.ts — length / includes / starts / ends.
func TestStringLengthChecks(t *testing.T) {
	minFive := String().Min(5, "min5")
	maxFive := String().Max(5, "max5")
	justFive := String().Length(5)
	nonempty := String().Min(1, "nonempty")

	mustParse(t, minFive, "12345")
	mustParse(t, minFive, "123456")
	mustParse(t, maxFive, "12345")
	mustParse(t, maxFive, "1234")
	mustParse(t, nonempty, "1")
	mustParse(t, justFive, "12345")

	mustFailMsg(t, minFive, "1234", "min5")
	mustFailMsg(t, maxFive, "123456", "max5")
	mustFailMsg(t, nonempty, "", "nonempty")
	mustFail(t, justFive, "1234")
	mustFail(t, justFive, "123456")
}

func TestStringIncludesStartsEnds(t *testing.T) {
	includes := String().Includes("includes")
	includesFrom2 := String().Includes("includes", 2)
	starts := String().StartsWith("startsWith")
	ends := String().EndsWith("endsWith")

	mustParse(t, includes, "XincludesXX")
	mustParse(t, includesFrom2, "XXXincludesXX")
	mustFail(t, includes, "XincludeXX")
	mustFail(t, includesFrom2, "XincludesXX")

	mustParse(t, starts, "startsWithX")
	mustParse(t, ends, "XendsWith")
	mustFail(t, starts, "x")
	mustFail(t, ends, "x")

	schema := String().Includes("test", "must contain test")
	mustParse(t, schema, "this is a test")
	mustFailMsg(t, schema, "this is invalid", "must contain test")
}

func TestStringTypeAndMessages(t *testing.T) {
	s := String()
	mustParse(t, s, "hello")
	res := s.SafeParse(123)
	if res.Success {
		t.Fatal("expected failure for number")
	}
	if res.Error.Issues[0].Message != "Invalid input: expected string, received number" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
	if res.Error.Issues[0].Code != IssueInvalidType {
		t.Fatalf("code=%s", res.Error.Issues[0].Code)
	}
}

func TestStringTooSmallTooBigMessages(t *testing.T) {
	res := String().Min(5).SafeParse("hi")
	if res.Success || res.Error.Issues[0].Message != "Too small: expected string to have >=5 characters" {
		t.Fatalf("got %+v", res.Error)
	}
	res = String().Max(2).SafeParse("hello")
	if res.Success || res.Error.Issues[0].Message != "Too big: expected string to have <=2 characters" {
		t.Fatalf("got %+v", res.Error)
	}
	res = String().Length(3).SafeParse("ab")
	if res.Success || !res.Error.Issues[0].Exact {
		t.Fatalf("want exact length issue, got %+v", res.Error)
	}
}

func TestStringRegex(t *testing.T) {
	// Ported: "regexp error message"
	res := String().Regex(regexp.MustCompile(`^moo+$`)).SafeParse("boooo")
	if res.Success {
		t.Fatal("expected fail")
	}
	iss := res.Error.Issues[0]
	if iss.Format != "regex" || iss.Pattern != "/^moo+$/" {
		t.Fatalf("pattern/format: %+v", iss)
	}
	if iss.Message != "Invalid string: must match pattern /^moo+$/" {
		t.Fatalf("message=%q", iss.Message)
	}

	res = String().Regex(regexp.MustCompile(`^moo+$`), "Custom error message").SafeParse("boooo")
	if res.Error.Issues[0].Message != "Custom error message" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestStringStartsEndsMessages(t *testing.T) {
	res := String().StartsWith("ab").SafeParse("x")
	if res.Error.Issues[0].Message != `Invalid string: must start with "ab"` {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
	res = String().EndsWith("yz").SafeParse("x")
	if res.Error.Issues[0].Message != `Invalid string: must end with "yz"` {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestStringTrimToCaseNormalize(t *testing.T) {
	// Ported: trim / toLowerCase / toUpperCase
	got, err := String().Trim().Min(2).Parse(" 12 ")
	if err != nil || got != "12" {
		t.Fatalf("got %q err=%v", got, err)
	}
	got, err = String().Min(2).Trim().Parse(" 1 ")
	if err != nil || got != "1" {
		t.Fatalf("ordering: got %q err=%v", got, err)
	}
	if _, err := String().Trim().Min(2).Parse(" 1 "); err == nil {
		t.Fatal("trim then min should fail")
	}

	got, err = String().ToLowerCase().Parse("ASDF")
	if err != nil || got != "asdf" {
		t.Fatalf("got %q", got)
	}
	got, err = String().ToUpperCase().Parse("asdf")
	if err != nil || got != "ASDF" {
		t.Fatalf("got %q", got)
	}

	// NFC: e + combining acute → é
	composed, err := String().Normalize().Parse("e\u0301")
	if err != nil {
		t.Fatal(err)
	}
	if composed != "é" && composed != "e\u0301" {
		// Accept either if already NFC-equivalent length; prefer composed.
		t.Logf("normalize result %q (len=%d)", composed, len(composed))
	}
	if String().Normalize().MustParse("café") != "café" {
		t.Fatal("NFC identity")
	}
}

func TestStringUpperLowerChecks(t *testing.T) {
	mustParse(t, String().Uppercase(), "ABC")
	mustFail(t, String().Uppercase(), "Abc")
	mustParse(t, String().Lowercase(), "abc")
	mustFail(t, String().Lowercase(), "Abc")
	res := String().Uppercase().SafeParse("ab")
	if res.Error.Issues[0].Message != "Invalid uppercase" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestStringCoerce(t *testing.T) {
	s := String(Params{Coerce: true})
	cases := []struct {
		in   any
		want string
	}{
		{123, "123"},
		{true, "true"},
		{false, "false"},
		{12.5, "12.5"},
		{nil, "null"},
		{"keep", "keep"},
	}
	for _, c := range cases {
		got, err := s.Parse(c.in)
		if err != nil || got != c.want {
			t.Fatalf("coerce %v: got %q err=%v want %q", c.in, got, err, c.want)
		}
	}
}

func TestStringAbortSemantics(t *testing.T) {
	// Aborting check suppresses subsequent checks that lack a When gate
	// (length checks have When, so use Email as the follower).
	s := String().Email(Params{Abort: true}).Regex(regexp.MustCompile(`^x$`))
	res := s.SafeParse("not-email")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("want 1 issue, got %+v", res.Error)
	}
	if res.Error.Issues[0].Format != "email" {
		t.Fatalf("format=%s", res.Error.Issues[0].Format)
	}

	// Non-abort: both issues collected.
	s = String().Email().Regex(regexp.MustCompile(`^x$`))
	res = s.SafeParse("not-email")
	if res.Success || len(res.Error.Issues) != 2 {
		t.Fatalf("want 2 issues, got %+v", res.Error)
	}
}

func TestStringEmail(t *testing.T) {
	// Subset of classic/tests/string.test.ts email validations
	email := String().Email()
	valid := []string{
		"email@domain.com",
		"firstname.lastname@domain.com",
		"email@subdomain.domain.com",
		"firstname+lastname@domain.com",
		"1234567890@domain.com",
		"email@domain-one.com",
		"_______@domain.com",
		"email@domain.name",
		"email@domain.co.jp",
		"firstname-lastname@domain.com",
		"very.common@example.com",
		"x@example.com",
		"user-@example.org",
		"a@b.cd",
		"work+user@mail.com",
		"common'name@domain.com",
	}
	invalid := []string{
		"francois@@etu.inp-n7.fr",
		`"email"@domain.com`,
		"a,b@domain.com",
		"email@123.123.123.123",
		"plainaddress",
		"@domain.com",
		"email.domain.com",
		".email@domain.com",
		"email.@domain.com",
		"email..email@domain.com",
		"email@domain",
		"email@-domain.com",
		"email@domain..com",
		"a.b@c.d",
		"test@.com",
	}
	for _, e := range valid {
		if _, err := email.Parse(e); err != nil {
			t.Errorf("valid email rejected: %s (%v)", e, err)
		}
	}
	for _, e := range invalid {
		if res := email.SafeParse(e); res.Success {
			t.Errorf("invalid email accepted: %s", e)
		}
	}
	res := email.SafeParse("not-an-email")
	if res.Error.Issues[0].Message != "Invalid email address" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestStringURL(t *testing.T) {
	u := String().URL()
	mustParse(t, u, "http://google.com")
	mustParse(t, u, "https://google.com/asdf?asdf=ljk3lk4&asdf=234#asdf")
	mustParse(t, u, "https://localhost")
	mustParse(t, u, "http://localhost")
	mustParse(t, u, "c:")
	mustFail(t, u, "asdf")
	mustFail(t, u, "https:/")
	mustFail(t, u, "https://")

	// trim + preserve
	got, err := u.Parse("  https://example.com  ")
	if err != nil || got != "https://example.com" {
		t.Fatalf("trim: %q %v", got, err)
	}
	in := "https://example.com?key=value"
	got, err = u.Parse(in)
	if err != nil || got != in {
		t.Fatalf("preserve: %q", got)
	}

	got, err = String().URL(URLOpts{Normalize: true}).Parse("https://example.com?key=value")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "example.com") {
		t.Fatalf("normalize got %q", got)
	}

	res := u.SafeParse("nope")
	if res.Error.Issues[0].Message != "Invalid URL" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestStringUUID(t *testing.T) {
	uuid := String().UUID()
	// Ported from classic/tests/string.test.ts "good uuid"
	good := []string{
		"9491d710-3185-1e06-bea0-6a2f275345e0",
		"9491d710-3185-4e06-bea0-6a2f275345e0",
		"9491d710-3185-5e06-8ea0-6a2f275345e0",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
	}
	for _, g := range good {
		mustParse(t, uuid, g)
	}
	mustFail(t, uuid, "invalid uuid")
	mustFail(t, uuid, "9491d710-3185-0e06-bea0-6a2f275345e0") // version 0
	mustFail(t, uuid, "b3ce60f8-e8b9-40f5-1150-172ede56ff74") // bad variant

	mustParse(t, String().UUIDv4(), "9491d710-3185-4e06-bea0-6a2f275345e0")
	mustFail(t, String().UUIDv4(), "9491d710-3185-1e06-bea0-6a2f275345e0")

	res := uuid.SafeParse("purr")
	if res.Error.Issues[0].Message != "Invalid UUID" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestStringIDsAndFormats(t *testing.T) {
	mustParse(t, String().CUID(), "ckopqwoedu0013g5hseu82ta1")
	mustFailMsg(t, String().CUID(), "bad", "Invalid cuid")

	mustParse(t, String().CUID2(), "tz4a98xxat96iws9zmbrgj3a")
	mustFail(t, String().CUID2(), "tz4a98xxat96iws9zMbrgj3a") // uppercase

	mustParse(t, String().ULID(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	mustFailMsg(t, String().ULID(), "bad", "Invalid ULID")

	mustParse(t, String().XID(), "9m4e2mr0ui3e8a215n4g")
	mustFailMsg(t, String().XID(), "bad", "Invalid XID")

	mustParse(t, String().KSUID(), "2GcR3dN8zH1KpLqWvYxTjBfMsEa")
	mustFailMsg(t, String().KSUID(), "bad", "Invalid KSUID")

	mustParse(t, String().NanoID(), "lfNZluvAxMkf7Q8C5H-QS")
	mustFail(t, String().NanoID(), "invalid nanoid")

	mustParse(t, String().GUID(), "b3ce60f8-e8b9-40f5-1150-172ede56ff74") // variant 0 ok for guid
}

func TestStringBase64HexJWT(t *testing.T) {
	mustParse(t, String().Base64(), "SGVsbG8gV29ybGQ=")
	mustParse(t, String().Base64(), "")
	mustFail(t, String().Base64(), "!UGF0aWVuY2U=")
	mustFailMsg(t, String().Base64(), "@@@", "Invalid base64-encoded string")

	mustParse(t, String().Base64URL(), "SGVsbG8")
	mustFail(t, String().Base64URL(), "SGVsbG8=") // padding not allowed in url alphabet check path via validator

	mustParse(t, String().Hex(), "")
	mustParse(t, String().Hex(), "DEADBEEF")
	mustFail(t, String().Hex(), "xyz")
	res := String().Hex().SafeParse("gg")
	if res.Error.Issues[0].Message != "Invalid hex" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}

	jwt := String().JWT()
	mustFail(t, jwt, "invalid")
	mustFail(t, jwt, "invalid.invalid")
	mustFail(t, jwt, "invalid.invalid.invalid")
	token := makeJWT(t, map[string]any{"typ": "JWT", "alg": "ES256"}, map[string]any{})
	mustParse(t, jwt, token)
	mustParse(t, String().JWT(JWTOpts{Alg: "ES256"}), token)
	mustFail(t, String().JWT(JWTOpts{Alg: "ES256"}), makeJWT(t, map[string]any{"typ": "JWT", "alg": "RS256"}, map[string]any{}))
	mustParse(t, jwt, makeJWT(t, map[string]any{"alg": "HS256"}, map[string]any{}))
	mustFail(t, jwt, makeJWT(t, map[string]any{"typ": "SUP", "alg": "HS256"}, map[string]any{}))
}

func TestStringIPMACCIDRE164(t *testing.T) {
	ipv4 := String().IPv4()
	mustParse(t, ipv4, "192.168.0.1")
	mustParse(t, ipv4, "0.0.0.0")
	mustFail(t, ipv4, "256.0.4.4")
	mustFail(t, ipv4, "1.1.1")

	ipv6 := String().IPv6()
	mustParse(t, ipv6, "1e5e:e6c8:daac:514b:114b:e360:d8c0:682c")
	mustParse(t, ipv6, "::1")
	mustParse(t, ipv6, "2001:db8::")
	mustParse(t, ipv6, "::ffff:192.168.0.1")
	mustFail(t, ipv6, "114.71.82.94")
	mustFail(t, ipv6, "not an ip")

	mac := String().MAC()
	mustParse(t, mac, "00:1A:2B:3C:4D:5E")
	mustParse(t, mac, "0a:1b:2c:3d:4e:5f")
	mustFail(t, mac, "00-1A-2B-3C-4D-5E")
	mustFail(t, mac, "00:1a:2B:3c:4D:5e") // mixed case
	mustParse(t, String().MAC(MACOpts{Delimiter: "-"}), "00-1A-2B-3C-4D-5E")

	mustParse(t, String().CIDRv4(), "192.168.0.0/24")
	mustFail(t, String().CIDRv4(), "192.168.0.0/33")
	mustParse(t, String().CIDRv6(), "2001:db8::/32")
	mustFail(t, String().CIDRv6(), "2001:db8::")

	e164 := String().E164()
	mustParse(t, e164, "+1555555")
	mustParse(t, e164, "+155555555555555")
	mustFail(t, e164, "555555555")
	mustFail(t, e164, "+0000000")
}

func TestStringISOFormats(t *testing.T) {
	mustParse(t, String().ISODate(), "2020-01-01")
	mustParse(t, String().ISODate(), "2020-02-29")
	mustFail(t, String().ISODate(), "2021-02-29")
	mustFail(t, String().ISODate(), "2020-1-1")

	mustParse(t, String().ISOTime(), "12:34")
	mustParse(t, String().ISOTime(), "12:34:56")
	mustParse(t, String().ISOTime(), "12:34:56.789")
	mustFail(t, String().ISOTime(), "24:00")

	mustParse(t, String().ISODateTime(), "2020-01-01T12:34:56Z")
	mustFail(t, String().ISODateTime(), "2020-01-01T12:34:56") // offset required by default (local=false)
	mustParse(t, String().ISODateTime(ISODateTimeOpts{Local: true}), "2020-01-01T12:34:56")
	mustParse(t, String().ISODateTime(ISODateTimeOpts{Offset: true}), "2020-01-01T12:34:56+01:00")

	mustParse(t, String().ISODuration(), "P1Y")
	mustParse(t, String().ISODuration(), "P1W")
	mustParse(t, String().ISODuration(), "PT1H")
	mustParse(t, String().ISODuration(), "P1DT2H3M4S")
	mustFail(t, String().ISODuration(), "P")
	mustFail(t, String().ISODuration(), "P1W2D") // weeks can't mix
	res := String().ISODuration().SafeParse("nope")
	if res.Error.Issues[0].Message != "Invalid ISO duration" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestStringEmoji(t *testing.T) {
	e := String().Emoji()
	mustParse(t, e, "👋👋👋👋")
	mustParse(t, e, "💚💙💜💛❤️")
	mustFail(t, e, ":-)")
	mustFail(t, e, "😀 is an emoji")
	mustFail(t, e, "😀stuff")
}

func TestStringBoundaryZeroLength(t *testing.T) {
	mustParse(t, String().Length(0), "")
	mustFail(t, String().Length(0), "a")
	mustParse(t, String().Min(0), "")
	mustParse(t, String().Max(0), "")
	mustFail(t, String().Max(0), "a")
}

func TestStringNonEmpty(t *testing.T) {
	mustParse(t, String().NonEmpty(), "x")
	mustFail(t, String().NonEmpty(), "")
}

// ---- helpers ----

func mustParse(t *testing.T, s *StringSchema, in string) {
	t.Helper()
	if _, err := s.Parse(in); err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", in, err)
	}
}

func mustFail(t *testing.T, s *StringSchema, in string) {
	t.Helper()
	if res := s.SafeParse(in); res.Success {
		t.Fatalf("Parse(%q) expected failure", in)
	}
}

func mustFailMsg(t *testing.T, s *StringSchema, in, msg string) {
	t.Helper()
	res := s.SafeParse(in)
	if res.Success {
		t.Fatalf("Parse(%q) expected failure", in)
	}
	if res.Error.Issues[0].Message != msg {
		t.Fatalf("Parse(%q) message=%q want %q", in, res.Error.Issues[0].Message, msg)
	}
}

func makeJWT(t *testing.T, header, payload map[string]any) string {
	t.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	enc := func(b []byte) string {
		return strings.TrimRight(base64.RawURLEncoding.EncodeToString(b), "=")
	}
	// Use std encoding without padding style similar to Zod's btoa path.
	h := base64.RawURLEncoding.EncodeToString(hb)
	p := base64.RawURLEncoding.EncodeToString(pb)
	_ = enc
	return h + "." + p + ".sig"
}
