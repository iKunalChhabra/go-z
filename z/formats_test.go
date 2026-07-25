package z

import (
	"regexp"
	"testing"
)

// Ported from classic/tests/string-formats.test.ts (format chaining + hex).
func TestStringFormatChaining(t *testing.T) {
	a := String().Email().Min(10)
	b := String().Email().Max(10)
	c := String().Email().Length(10)
	d := String().Email().Uppercase()
	e := String().Email().Lowercase()

	mustParse(t, a, "longemail@example.com")
	mustFail(t, a, "ort@e.co")

	mustParse(t, b, "sho@e.co")
	mustFail(t, b, "longemail@example.com")

	mustParse(t, c, "56780@e.co")
	mustFail(t, c, "shoasdfasdfrt@e.co")

	mustParse(t, d, "EMAIL@EXAMPLE.COM")
	mustFail(t, d, "email@example.com")

	mustParse(t, e, "email@example.com")
	mustFail(t, e, "EMAIL@EXAMPLE.COM")
}

func TestFormatHex(t *testing.T) {
	hex := String().Hex()
	mustParse(t, hex, "")
	mustParse(t, hex, "123abc")
	mustParse(t, hex, "DEADBEEF")
	mustParse(t, hex, "0123456789abcdefABCDEF")
	mustFail(t, hex, "xyz")
	mustFail(t, hex, "123g")
	mustFail(t, hex, "hello world")
	mustFail(t, hex, "123-abc")
}

func TestFormatFactoriesDirect(t *testing.T) {
	p := AcquirePayload("not-email")
	defer ReleasePayload(p)
	FormatEmail().Fn(p)
	if len(p.Issues) != 1 || p.Issues[0].Format != "email" {
		t.Fatalf("%+v", p.Issues)
	}
	if p.Issues[0].Pattern != patternEmail {
		t.Fatalf("pattern=%q", p.Issues[0].Pattern)
	}

	p2 := AcquirePayload("not-a-duration")
	defer ReleasePayload(p2)
	FormatISODuration().Fn(p2)
	if p2.Issues[0].Pattern != patternDuration {
		t.Fatalf("duration pattern=%q", p2.Issues[0].Pattern)
	}
}

func TestFormatAbort(t *testing.T) {
	// Email has no When gate; aborting it suppresses the following Regex check.
	s := String().Email(Params{Abort: true, Error: MessageFromString("bad email")}).Regex(regexp.MustCompile(`^x$`))
	res := s.SafeParse("x")
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("want 1 issue after abort, got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "bad email" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}
