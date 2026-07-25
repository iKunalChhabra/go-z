package zod

import (
	"regexp"
	"strings"
	"testing"
)

func parityStrOK(t *testing.T, s *StringSchema, in string) {
	t.Helper()
	if _, err := s.Parse(in); err != nil {
		t.Fatalf("Parse(%q): %v", in, err)
	}
}

func parityStrFail(t *testing.T, s *StringSchema, in string) {
	t.Helper()
	if s.SafeParse(in).Success {
		t.Fatalf("Parse(%q) expected failure", in)
	}
}

func parityStrFailMsg(t *testing.T, s *StringSchema, in, msg string) {
	t.Helper()
	res := s.SafeParse(in)
	if res.Success {
		t.Fatalf("Parse(%q) expected failure", in)
	}
	if res.Error.Issues[0].Message != msg {
		t.Fatalf("Parse(%q) message=%q want %q", in, res.Error.Issues[0].Message, msg)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/string.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityStringLengthChecks(t *testing.T) {
	// Ported from classic/tests/string.test.ts — "length checks"
	minFive := String().Min(5, "min5")
	maxFive := String().Max(5, "max5")
	justFive := String().Length(5)
	nonempty := String().Min(1, "nonempty")

	for _, s := range []string{"12345", "123456"} {
		parityStrOK(t, minFive, s)
	}
	for _, s := range []string{"12345", "1234"} {
		parityStrOK(t, maxFive, s)
	}
	parityStrOK(t, nonempty, "1")
	parityStrOK(t, justFive, "12345")

	parityStrFailMsg(t, minFive, "1234", "min5")
	parityStrFailMsg(t, maxFive, "123456", "max5")
	parityStrFailMsg(t, nonempty, "", "nonempty")
	parityStrFail(t, justFive, "1234")
	parityStrFail(t, justFive, "123456")
}

func TestParityStringIncludesStartsEnds(t *testing.T) {
	// Ported from classic/tests/string.test.ts — includes / startswith / endswith
	includes := String().Includes("includes")
	includesFrom2 := String().Includes("includes", 2)
	starts := String().StartsWith("startsWith")
	ends := String().EndsWith("endsWith")

	parityStrOK(t, includes, "XincludesXX")
	parityStrOK(t, includesFrom2, "XXXincludesXX")
	parityStrFail(t, includes, "XincludeXX")
	parityStrFail(t, includesFrom2, "XincludesXX")

	parityStrOK(t, starts, "startsWithX")
	parityStrOK(t, ends, "XendsWith")
	parityStrFail(t, starts, "x")
	parityStrFail(t, ends, "x")

	schema := String().Includes("test", "must contain test")
	parityStrOK(t, schema, "this is a test")
	parityStrFailMsg(t, schema, "this is invalid", "must contain test")
}

func TestParityStringEmailValidations(t *testing.T) {
	// Ported from classic/tests/string.test.ts — "email validations"
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
		"disposable.style.email.with+symbol@example.com",
		"other.email-with-hyphen@example.com",
		"fully-qualified-domain@example.com",
		"user.name+tag+sorting@example.com",
		"x@example.com",
		"mojojojo@asdf.example.com",
		"example-indeed@strange-example.com",
		"example@s.example",
		"user-@example.org",
		"user@my-example.com",
		"a@b.cd",
		"work+user@mail.com",
		"tom@test.te-st.com",
		"something@subdomain.domain-with-hyphens.tld",
		"common'name@domain.com",
		"francois@etu.inp-n7.fr",
	}
	invalid := []string{
		"francois@@etu.inp-n7.fr",
		`"email"@domain.com`,
		`"e asdf sadf ?<>ail"@domain.com`,
		`" "@example.org`,
		"a,b@domain.com",
		"email@123.123.123.123",
		"email@[123.123.123.123]",
		"plainaddress",
		"#@%^%#$@#$@#.com",
		"@domain.com",
		"email.domain.com",
		"email@domain@domain.com",
		".email@domain.com",
		"email.@domain.com",
		"email..email@domain.com",
		"あいうえお@domain.com",
		"email@domain.com (Joe Smith)",
		"email@domain",
		"email@-domain.com",
		"email@111.222.333.44444",
		"email@domain..com",
		"Abc.example.com",
		"A@b@c@example.com",
		"colin..hacks@domain.com",
		"just\"not\"right@example.com",
		"i_like_underscore@but_its_not_allowed_in_this_part.example.com",
		"invalid@-start.com",
		"invalid@end.com-",
		"a.b@c.d",
		"double..point@test.com",
		"asdad@test..com",
		"test@.com",
		"aaaaaaaaaaaaaaalongemailthatcausesregexDoSvulnerability@test.c",
	}
	for _, e := range valid {
		if _, err := email.Parse(e); err != nil {
			t.Errorf("valid email rejected: %q (%v)", e, err)
		}
	}
	for _, e := range invalid {
		if email.SafeParse(e).Success {
			t.Errorf("invalid email accepted: %q", e)
		}
	}
	parityStrFailMsg(t, email, "not-an-email", "Invalid email address")
}

func TestParityStringBase64(t *testing.T) {
	// Ported from classic/tests/string.test.ts — "base64 validations"
	schema := String().Base64()
	valid := []string{
		"SGVsbG8gV29ybGQ=",
		"VGhpcyBpcyBhbiBlbmNvZGVkIHN0cmluZw==",
		"TWFueSBoYW5kcyBtYWtlIGxpZ2h0IHdvcms=",
		"UGF0aWVuY2UgaXMgdGhlIGtleSB0byBzdWNjZXNz",
		"QmFzZTY0IGVuY29kaW5nIGlzIGZ1bg==",
		"MTIzNDU2Nzg5MA==",
		"YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=",
		"QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo=",
		"ISIkJSMmJyonKCk=",
		"",
	}
	invalid := []string{
		"12345",
		"SGVsbG8gV29ybGQ",
		"!UGF0aWVuY2UgaXMgdGhlIGtleSB0byBzdWNjZXNz",
		"?QmFzZTY0IGVuY29kaW5nIGlzIGZ1bg==",
		"123 ",
		"SGVsbG8gV29ybGQ= ",
		" SGVsbG8gV29ybGQ=",
		"SGVs bG8gV29ybGQ=",
	}
	for _, s := range valid {
		parityStrOK(t, schema, s)
	}
	for _, s := range invalid {
		parityStrFail(t, schema, s)
	}
}

func TestParityStringBase64URL(t *testing.T) {
	// Ported from classic/tests/string.test.ts — "base64url validations"
	schema := String().Base64URL()
	valid := []string{
		"SGVsbG8gV29ybGQ",
		"VGhpcyBpcyBhbiBlbmNvZGVkIHN0cmluZw",
		"UGF0aWVuY2UgaXMgdGhlIGtleSB0byBzdWNjZXNz",
		"",
		"w7_Dv8O-w74K",
		"123456",
	}
	invalid := []string{
		"w7/Dv8O+w74K",
		"12345",
		"!UGF0aWVuY2UgaXMgdGhlIGtleSB0byBzdWNjZXNz",
		"SGVsbG8gV29ybGQ=",
		"VGhpcyBpcyBhbiBlbmNvZGVkIHN0cmluZw==",
	}
	for _, s := range valid {
		parityStrOK(t, schema, s)
	}
	for _, s := range invalid {
		parityStrFail(t, schema, s)
	}
}

func TestParityStringJWT(t *testing.T) {
	// Ported from classic/tests/string.test.ts — "jwt token"
	jwt := String().JWT()
	for _, bad := range []string{"invalid", "invalid.invalid", "invalid.invalid.invalid"} {
		parityStrFail(t, jwt, bad)
	}
	d1 := makeJWT(t, map[string]any{"typ": "JWT", "alg": "ES256"}, map[string]any{})
	parityStrOK(t, jwt, d1)
	parityStrOK(t, String().JWT(JWTOpts{Alg: "ES256"}), d1)

	d2 := makeJWT(t, map[string]any{}, map[string]any{})
	parityStrFail(t, jwt, d2)

	d3 := makeJWT(t, map[string]any{"typ": "JWT", "alg": "RS256"}, map[string]any{})
	parityStrFail(t, String().JWT(JWTOpts{Alg: "ES256"}), d3)

	d4 := makeJWT(t, map[string]any{"alg": "HS256"}, map[string]any{})
	parityStrOK(t, jwt, d4)

	d5 := makeJWT(t, map[string]any{"typ": "SUP", "alg": "HS256"}, map[string]any{"foo": "bar"})
	parityStrFail(t, jwt, d5)
}

func TestParityStringURL(t *testing.T) {
	// Ported from classic/tests/string.test.ts — url validations / preserve / trim / normalize
	url := String().URL()
	ok := []string{
		"http://google.com",
		"https://google.com/asdf?asdf=ljk3lk4&asdf=234#asdf",
		"https://anonymous:flabada@developer.mozilla.org/en-US/docs/Web/API/URL/password",
		"https://localhost",
		"https://my.local",
		"http://aslkfjdalsdfkjaf",
		"http://localhost",
		"c:",
	}
	for _, u := range ok {
		parityStrOK(t, url, u)
	}
	for _, u := range []string{"asdf", "https:/", "asdfj@lkjsdf.com", "https://"} {
		parityStrFail(t, url, u)
	}

	in := "https://example.com?key=NUXOmHqWNVTapJkJJHw8BfD155AuqhH_qju_5fNmQ4ZHV7u8"
	got, err := url.Parse(in)
	if err != nil || got != in {
		t.Fatalf("preserve: got %q err=%v", got, err)
	}
	for _, u := range []string{
		"https://example.com?foo=bar",
		"http://example.com?test=123",
		"https://example.com/",
		"https://example.com/path/",
		"https://example.com/path?query=param",
	} {
		got, err := url.Parse(u)
		if err != nil || got != u {
			t.Fatalf("preserve %q → %q err=%v", u, got, err)
		}
	}

	got, err = url.Parse("  https://example.com  ")
	if err != nil || got != "https://example.com" {
		t.Fatalf("trim: %q %v", got, err)
	}
	got, err = url.Parse("\t\nhttps://example.com\t\n")
	if err != nil || got != "https://example.com" {
		t.Fatalf("trim whitespace: %q %v", got, err)
	}

	norm := String().URL(URLOpts{Normalize: true})
	got, err = norm.Parse("https://example.com?key=value")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "example.com") {
		t.Fatalf("normalize got %q", got)
	}

	parityStrFailMsg(t, url, "https", "Invalid URL")
	parityStrFailMsg(t, String().URL("badurl"), "https", "badurl")
	parityStrFailMsg(t, String().URL(Params{Error: MessageFromString("badurl")}), "https", "badurl")
}

func TestParityStringURLHostnameProtocol(t *testing.T) {
	// Ported from classic/tests/url.test.ts + string.test.ts "httpurl" / mini string URL opts
	hostDots := String().URL(URLOpts{Hostname: regexp.MustCompile(`\.+`)})
	parityStrOK(t, hostDots, "http://example.com")
	parityStrOK(t, hostDots, "https://sub.example.com")
	parityStrFail(t, hostDots, "http://localhost") // no dot

	httpsOnly := String().URL(URLOpts{Protocol: regexp.MustCompile(`^https:$`)})
	parityStrOK(t, httpsOnly, "https://example.com")
	parityStrFail(t, httpsOnly, "http://example.com")

	both := String().URL(URLOpts{
		Hostname: regexp.MustCompile(`example\.com$`),
		Protocol: regexp.MustCompile(`^https$`),
	})
	parityStrOK(t, both, "https://example.com")
	parityStrOK(t, both, "https://sub.example.com")
	parityStrFail(t, both, "http://example.com")
	parityStrFail(t, both, "https://example.org")

	httpURL := String().HttpURL()
	parityStrOK(t, httpURL, "https://x.com")
	parityStrOK(t, httpURL, "http://example.com")
	parityStrOK(t, httpURL, "https://sub.example.com/path?q=1#f")
	parityStrFail(t, httpURL, "ftp://example.com")
	parityStrFail(t, httpURL, "mailto:asdf@lckj.com")
	parityStrFail(t, httpURL, "http://localhost")
	parityStrFail(t, httpURL, "http:example.com")
}

func TestParityStringEmoji(t *testing.T) {
	// Ported from classic/tests/string.test.ts — "emoji validations" (subset)
	emoji := String().Emoji()
	for _, s := range []string{"👋👋👋👋", "🍺👩‍🚀🫡", "💚💙💜💛❤️", "🇹🇷"} {
		parityStrOK(t, emoji, s)
	}
	for _, s := range []string{":-)", "😀 is an emoji", "😀stuff", "stuff😀"} {
		parityStrFail(t, emoji, s)
	}
}

func TestParityStringNanoID(t *testing.T) {
	// Ported from classic/tests/string.test.ts — nanoid
	nanoid := String().NanoID("custom error")
	for _, s := range []string{
		"lfNZluvAxMkf7Q8C5H-QS",
		"mIU_4PJWikaU8fMbmkouz",
		"Hb9ZUtUa2JDm_dD-47EGv",
		"5Noocgv_8vQ9oPijj4ioQ",
		"ySh_984wpDUu7IQRrLXAp",
	} {
		parityStrOK(t, nanoid, s)
	}
	parityStrFailMsg(t, nanoid, "Xq90uDyhddC53KsoASYJGX", "custom error")
	parityStrFailMsg(t, nanoid, "invalid nanoid", "custom error")
}

func TestParityStringUUID(t *testing.T) {
	// Ported from classic/tests/string.test.ts — good/bad uuid + guid
	uuid := String().UUID("custom error")
	good := []string{
		"9491d710-3185-1e06-bea0-6a2f275345e0",
		"9491d710-3185-2e06-bea0-6a2f275345e0",
		"9491d710-3185-3e06-bea0-6a2f275345e0",
		"9491d710-3185-4e06-bea0-6a2f275345e0",
		"9491d710-3185-5e06-bea0-6a2f275345e0",
		"9491d710-3185-5e06-aea0-6a2f275345e0",
		"9491d710-3185-5e06-8ea0-6a2f275345e0",
		"9491d710-3185-5e06-9ea0-6a2f275345e0",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
	}
	for _, g := range good {
		parityStrOK(t, uuid, g)
	}
	bad := []string{
		"9491d710-3185-0e06-bea0-6a2f275345e0",
		"9491d710-3185-5e06-0ea0-6a2f275345e0",
		"d89e7b01-7598-ed11-9d7a-0022489382fd",
		"b3ce60f8-e8b9-40f5-1150-172ede56ff74",
		"92e76bf9-28b3-4730-cd7f-cb6bc51f8c09",
		"invalid uuid",
		"9491d710-3185-4e06-bea0-6a2f275345e0X",
	}
	for _, b := range bad {
		parityStrFailMsg(t, uuid, b, "custom error")
	}

	guid := String().GUID("custom error")
	for _, g := range []string{
		"9491d710-3185-4e06-bea0-6a2f275345e0",
		"d89e7b01-7598-ed11-9d7a-0022489382fd",
		"b3ce60f8-e8b9-40f5-1150-172ede56ff74",
		"92e76bf9-28b3-4730-cd7f-cb6bc51f8c09",
	} {
		parityStrOK(t, guid, g)
	}
	parityStrFailMsg(t, guid, "invalid", "custom error")

	parityStrOK(t, String().UUIDv4(), "9491d710-3185-4e06-bea0-6a2f275345e0")
	parityStrFail(t, String().UUIDv4(), "9491d710-3185-1e06-bea0-6a2f275345e0")
}

func TestParityStringIDs(t *testing.T) {
	// Ported from classic/tests/string.test.ts — cuid / cuid2 / ulid / xid / ksuid
	parityStrOK(t, String().CUID(), "ckopqwoedu0013g5hseu82ta1")
	parityStrFailMsg(t, String().CUID(), "bad", "Invalid cuid")

	parityStrOK(t, String().CUID2(), "tz4a98xxat96iws9zmbrgj3a")
	parityStrFail(t, String().CUID2(), "tz4a98xxat96iws9zMbrgj3a")

	parityStrOK(t, String().ULID(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	parityStrFailMsg(t, String().ULID(), "bad", "Invalid ULID")

	parityStrOK(t, String().XID(), "9m4e2mr0ui3e8a215n4g")
	parityStrFailMsg(t, String().XID(), "bad", "Invalid XID")

	parityStrOK(t, String().KSUID(), "2GcR3dN8zH1KpLqWvYxTjBfMsEa")
	parityStrFailMsg(t, String().KSUID(), "bad", "Invalid KSUID")
}

func TestParityStringIPMACCIDRE164(t *testing.T) {
	// Ported from classic/tests/string.test.ts — ipv4/ipv6/mac/cidr/e164
	ipv4 := String().IPv4()
	for _, s := range []string{"192.168.0.1", "0.0.0.0", "255.255.255.255", "127.0.0.1"} {
		parityStrOK(t, ipv4, s)
	}
	for _, s := range []string{"256.0.4.4", "1.1.1", "1.1.1.1.1", "not an ip"} {
		parityStrFail(t, ipv4, s)
	}

	ipv6 := String().IPv6()
	for _, s := range []string{
		"1e5e:e6c8:daac:514b:114b:e360:d8c0:682c",
		"::1",
		"2001:db8::",
		"::ffff:192.168.0.1",
	} {
		parityStrOK(t, ipv6, s)
	}
	for _, s := range []string{"114.71.82.94", "not an ip", "zzzz::"} {
		parityStrFail(t, ipv6, s)
	}

	mac := String().MAC()
	parityStrOK(t, mac, "00:1A:2B:3C:4D:5E")
	parityStrOK(t, mac, "0a:1b:2c:3d:4e:5f")
	parityStrFail(t, mac, "00-1A-2B-3C-4D-5E")
	parityStrOK(t, String().MAC(MACOpts{Delimiter: "-"}), "00-1A-2B-3C-4D-5E")

	parityStrOK(t, String().CIDRv4(), "192.168.0.0/24")
	parityStrFail(t, String().CIDRv4(), "192.168.0.0/33")
	parityStrOK(t, String().CIDRv6(), "2001:db8::/32")
	parityStrFail(t, String().CIDRv6(), "2001:db8::")

	e164 := String().E164()
	parityStrOK(t, e164, "+1555555")
	parityStrOK(t, e164, "+155555555555555")
	parityStrFail(t, e164, "555555555")
	parityStrFail(t, e164, "+0000000")
}

func TestParityStringRegexTrimCase(t *testing.T) {
	// Ported from classic/tests/string.test.ts — regexp / trim / case / normalize
	res := String().Regex(regexp.MustCompile(`^moo+$`)).SafeParse("boooo")
	if res.Success {
		t.Fatal("expected fail")
	}
	if res.Error.Issues[0].Format != "regex" {
		t.Fatalf("format=%s", res.Error.Issues[0].Format)
	}
	parityStrFailMsg(t, String().Regex(regexp.MustCompile(`^moo+$`), "Custom error message"), "boooo", "Custom error message")

	got, err := String().Trim().Min(2).Parse(" 12 ")
	if err != nil || got != "12" {
		t.Fatalf("trim+min: %q %v", got, err)
	}
	if _, err := String().Trim().Min(2).Parse(" 1 "); err == nil {
		t.Fatal("trim then min should fail")
	}
	got, err = String().ToLowerCase().Parse("ASDF")
	if err != nil || got != "asdf" {
		t.Fatalf("tolower: %q", got)
	}
	got, err = String().ToUpperCase().Parse("asdf")
	if err != nil || got != "ASDF" {
		t.Fatalf("toupper: %q", got)
	}
	parityStrOK(t, String().Uppercase(), "ABC")
	parityStrFail(t, String().Uppercase(), "Abc")
	parityStrOK(t, String().Lowercase(), "abc")
	parityStrFail(t, String().Lowercase(), "Abc")

	if String().Normalize().MustParse("café") != "café" {
		t.Fatal("NFC identity")
	}
}

func TestParityStringTypeMessages(t *testing.T) {
	// Ported from classic/tests/string.test.ts + validations.test.ts (string parts)
	s := String()
	res := s.SafeParse(123)
	if res.Success || res.Error.Issues[0].Code != IssueInvalidType {
		t.Fatalf("want invalid_type, got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "Invalid input: expected string, received number" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}

	res = String().Min(5).SafeParse("hi")
	if res.Success || res.Error.Issues[0].Message != "Too small: expected string to have >=5 characters" {
		t.Fatalf("got %+v", res.Error)
	}
	res = String().Max(2).SafeParse("hello")
	if res.Success || res.Error.Issues[0].Message != "Too big: expected string to have <=2 characters" {
		t.Fatalf("got %+v", res.Error)
	}

	res = String().Length(4).SafeParse("asd")
	if res.Success || res.Error.Issues[0].Code != IssueTooSmall || !res.Error.Issues[0].Exact {
		t.Fatalf("length too_small exact: %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "Too small: expected string to have >=4 characters" {
		t.Fatalf("msg=%q", res.Error.Issues[0].Message)
	}
	res = String().Length(4).SafeParse("asdaa")
	if res.Success || res.Error.Issues[0].Code != IssueTooBig || !res.Error.Issues[0].Exact {
		t.Fatalf("length too_big exact: %+v", res.Error)
	}
	res = String().Min(4).SafeParse("asd")
	if res.Success || res.Error.Issues[0].Message != "Too small: expected string to have >=4 characters" {
		t.Fatalf("min: %+v", res.Error)
	}
	res = String().Max(4).SafeParse("aasdfsdfsd")
	if res.Success || res.Error.Issues[0].Message != "Too big: expected string to have <=4 characters" {
		t.Fatalf("max: %+v", res.Error)
	}

	res = String().StartsWith("ab").SafeParse("x")
	if res.Error.Issues[0].Message != `Invalid string: must start with "ab"` {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
	res = String().EndsWith("yz").SafeParse("x")
	if res.Error.Issues[0].Message != `Invalid string: must end with "yz"` {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/string-formats.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityStringFormatsChaining(t *testing.T) {
	// Ported from classic/tests/string-formats.test.ts — "string format methods"
	a := String().Email().Min(10)
	b := String().Email().Max(10)
	c := String().Email().Length(10)
	d := String().Email().Uppercase()
	e := String().Email().Lowercase()

	parityStrOK(t, a, "longemail@example.com")
	parityStrFail(t, a, "ort@e.co")
	parityStrOK(t, b, "sho@e.co")
	parityStrFail(t, b, "longemail@example.com")
	parityStrOK(t, c, "56780@e.co")
	parityStrFail(t, c, "shoasdfasdfrt@e.co")
	parityStrOK(t, d, "EMAIL@EXAMPLE.COM")
	parityStrFail(t, d, "email@example.com")
	parityStrOK(t, e, "email@example.com")
	parityStrFail(t, e, "EMAIL@EXAMPLE.COM")
}

func TestParityStringFormatCustom(t *testing.T) {
	// Ported from classic/tests/string-formats.test.ts — "z.stringFormat"
	re := regexp.MustCompile(`^foo+$`)
	s := StringFormat("my-format", re)
	parityStrOK(t, s, "foo")
	parityStrOK(t, s, "foooo")
	parityStrFail(t, s, "bar")
	res := s.SafeParse("bar")
	if res.Success || res.Error.Issues[0].Format != "my-format" {
		t.Fatalf("got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "Invalid my-format" {
		t.Fatalf("message=%q", res.Error.Issues[0].Message)
	}

	ccRegex := regexp.MustCompile(`^(?:\d{14,19}|\d{4}(?: \d{3,6}){2,4}|\d{4}(?:-\d{3,6}){2,4})$`)
	a := StringFormat("creditCard", func(val string) bool {
		return ccRegex.MatchString(val)
	}, Params{Error: MessageFromString("Invalid credit card number")}).Refine(func(string) bool {
		return false
	}, "Also bad")
	res = a.SafeParse("asdf")
	if res.Success || len(res.Error.Issues) != 2 {
		t.Fatalf("want 2 issues, got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "Invalid credit card number" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
	res = a.SafeParse("1234-5678-9012-3456")
	if res.Success || len(res.Error.Issues) != 1 || res.Error.Issues[0].Message != "Also bad" {
		t.Fatalf("got %+v", res.Error)
	}

	b := StringFormat("creditCard", ccRegex, Params{
		Abort: true,
		Error: MessageFromString("Invalid credit card number"),
	}).Refine(func(string) bool { return false }, "Also bad")
	res = b.SafeParse("asdf")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("abort: %+v", res.Error)
	}
}

func TestParityStringHex(t *testing.T) {
	// Ported from classic/tests/string-formats.test.ts — "z.hex"
	hex := String().Hex()
	for _, s := range []string{"", "123abc", "DEADBEEF", "0123456789abcdefABCDEF"} {
		parityStrOK(t, hex, s)
	}
	for _, s := range []string{"xyz", "123g", "hello world", "123-abc"} {
		parityStrFail(t, hex, s)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/datetime.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityStringDateTimeBasic(t *testing.T) {
	// Ported from classic/tests/datetime.test.ts — "basic datetime parsing"
	dt := String().ISODateTime()
	for _, s := range []string{
		"1970-01-01T00:00:00.000Z",
		"2022-10-13T09:52:31.816Z",
		"2022-10-13T09:52:31.8162314Z",
		"1970-01-01T00:00:00Z",
		"2022-10-13T09:52:31Z",
	} {
		parityStrOK(t, dt, s)
	}
	for _, s := range []string{"", "foo", "2020-10-14", "T18:45:12.123", "2020-10-14T17:42:29+00:00"} {
		parityStrFail(t, dt, s)
	}
}

func TestParityStringDateTimePrecision(t *testing.T) {
	// Ported from classic/tests/datetime.test.ts — precision -1 / 0 / 3
	pNeg1 := -1
	noMs := String().ISODateTime(ISODateTimeOpts{Precision: &pNeg1, Offset: true, Local: true})
	for _, s := range []string{"1970-01-01T00:00Z", "2022-10-13T09:52Z", "2022-10-13T09:52+02:00", "2022-10-13T09:52"} {
		parityStrOK(t, noMs, s)
	}
	for _, s := range []string{"tuna", "2022-10-13T09:52+02", "1970-01-01T00:00:00.000Z", "2022-10-13T09:52:31.816Z"} {
		parityStrFail(t, noMs, s)
	}

	p0 := 0
	prec0 := String().ISODateTime(ISODateTimeOpts{Precision: &p0})
	parityStrOK(t, prec0, "1970-01-01T00:00:00Z")
	parityStrOK(t, prec0, "2022-10-13T09:52:31Z")
	parityStrFail(t, prec0, "1970-01-01T00:00:00.000Z")
	parityStrFail(t, prec0, "2022-10-13T09:52:31.816Z")

	p3 := 3
	prec3 := String().ISODateTime(ISODateTimeOpts{Precision: &p3})
	parityStrOK(t, prec3, "1970-01-01T00:00:00.000Z")
	parityStrOK(t, prec3, "2022-10-13T09:52:31.123Z")
	parityStrFail(t, prec3, "1970-01-01T00:00:00.1Z")
	parityStrFail(t, prec3, "2022-10-13T09:52:31Z")
}

func TestParityStringDateTimeOffsetLocal(t *testing.T) {
	// Ported from classic/tests/datetime.test.ts — offset / local
	off := String().ISODateTime(ISODateTimeOpts{Offset: true})
	for _, s := range []string{
		"1970-01-01T00:00:00.000Z",
		"2022-10-13T09:52:31.816234134Z",
		"1970-01-01T00:00:00Z",
		"2022-10-13T09:52:31.4Z",
		"2020-10-14T17:42:29+00:00",
		"2020-10-14T17:42:29+03:15",
	} {
		parityStrOK(t, off, s)
	}
	for _, s := range []string{
		"2020-10-14T17:42:29+0315",
		"2020-10-14T17:42:29+03",
		"tuna",
		"2022-10-13T09:52:31.Z",
		"2020-10-14T17:42:29+24:00",
		"2020-10-14T17:42:29+00:60",
		"2020-10-14T17:42:29+1:30",
	} {
		parityStrFail(t, off, s)
	}

	local := String().ISODateTime(ISODateTimeOpts{Local: true})
	for _, s := range []string{"1970-01-01T00:00", "1970-01-01T00:00:00", "2022-10-13T09:52:31.816", "1970-01-01T00:00:00.000"} {
		parityStrOK(t, local, s)
	}
	for _, s := range []string{"1970-01-01T00", "2022-10-13T09:52:31+00:00", "2022-10-13 09:52:31", "2022-10-13T24:52:31"} {
		parityStrFail(t, local, s)
	}

	both := String().ISODateTime(ISODateTimeOpts{Local: true, Offset: true})
	for _, s := range []string{"2022-10-13T12:52:00", "2022-10-13T12:52:00Z", "2022-10-13T12:52Z", "2022-10-13T12:52", "2022-10-13T12:52+02:00"} {
		parityStrOK(t, both, s)
	}
	parityStrFail(t, both, "2022-10-13T12:52:00+02")
}

func TestParityStringISODateTimeDate(t *testing.T) {
	// Ported from classic/tests/datetime.test.ts — date / time / duration
	date := String().ISODate()
	for _, s := range []string{
		"1970-01-01", "2022-01-31", "2022-02-28", "2020-02-29",
		"2022-03-31", "2022-04-30", "2022-05-31", "2022-06-30",
		"2022-07-31", "2022-08-31", "2022-09-30", "2022-10-31",
		"2022-11-30", "2022-12-31",
	} {
		parityStrOK(t, date, s)
	}
	for _, s := range []string{"2021-02-29", "2020-1-1", "2020-01-32", "foo", ""} {
		parityStrFail(t, date, s)
	}

	time := String().ISOTime()
	parityStrOK(t, time, "12:34")
	parityStrOK(t, time, "12:34:56")
	parityStrOK(t, time, "12:34:56.789")
	parityStrFail(t, time, "24:00")
	parityStrFail(t, time, "12:60")

	dur := String().ISODuration()
	for _, s := range []string{"P1Y", "P1W", "PT1H", "P1DT2H3M4S", "P1Y2M3DT4H5M6S"} {
		parityStrOK(t, dur, s)
	}
	for _, s := range []string{"P", "P1W2D", "nope", ""} {
		parityStrFail(t, dur, s)
	}
}

func TestParityStringNonEmpty(t *testing.T) {
	// Ported from classic/tests/string.test.ts — nonempty
	parityStrOK(t, String().NonEmpty(), "x")
	parityStrFail(t, String().NonEmpty(), "")
}

func TestParityStringCoerce(t *testing.T) {
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
