package z

import (
	"regexp"
	"testing"
)

func TestMinMaxLengthChecksDirect(t *testing.T) {
	p := AcquirePayload("hi")
	defer ReleasePayload(p)
	MinLength(5).Fn(p)
	if len(p.Issues) != 1 || p.Issues[0].Code != IssueTooSmall {
		t.Fatalf("min: %+v", p.Issues)
	}
	if p.Issues[0].Origin != "string" || p.Issues[0].Inclusive != true {
		t.Fatalf("fields: %+v", p.Issues[0])
	}

	p2 := AcquirePayload("hello!")
	defer ReleasePayload(p2)
	MaxLength(3).Fn(p2)
	if len(p2.Issues) != 1 || p2.Issues[0].Code != IssueTooBig {
		t.Fatalf("max: %+v", p2.Issues)
	}

	p3 := AcquirePayload("abcd")
	defer ReleasePayload(p3)
	LengthEquals(2).Fn(p3)
	if !p3.Issues[0].Exact || p3.Issues[0].Code != IssueTooBig {
		t.Fatalf("length: %+v", p3.Issues[0])
	}
}

func TestOverwriteMutations(t *testing.T) {
	p := AcquirePayload("  Hi ")
	defer ReleasePayload(p)
	Trim().Fn(p)
	if p.Value != "Hi" {
		t.Fatalf("trim=%v", p.Value)
	}
	ToLowerCase().Fn(p)
	if p.Value != "hi" {
		t.Fatalf("lower=%v", p.Value)
	}
	ToUpperCase().Fn(p)
	if p.Value != "HI" {
		t.Fatalf("upper=%v", p.Value)
	}
}

func TestRegexIncludesFormatFields(t *testing.T) {
	p := AcquirePayload("x")
	defer ReleasePayload(p)
	Regex(regexp.MustCompile(`^a+$`)).Fn(p)
	if p.Issues[0].Format != "regex" || p.Issues[0].Pattern != "/^a+$/" {
		t.Fatalf("%+v", p.Issues[0])
	}

	p2 := AcquirePayload("hello")
	defer ReleasePayload(p2)
	Includes("zz").Fn(p2)
	if p2.Issues[0].Format != "includes" || p2.Issues[0].Includes != "zz" {
		t.Fatalf("%+v", p2.Issues[0])
	}

	p3 := AcquirePayload("hello")
	defer ReleasePayload(p3)
	StartsWith("x").Fn(p3)
	if p3.Issues[0].Prefix != "x" {
		t.Fatalf("%+v", p3.Issues[0])
	}
	EndsWith("x").Fn(p3)
	if p3.Issues[len(p3.Issues)-1].Suffix != "x" {
		t.Fatalf("%+v", p3.Issues)
	}
}

func TestUpperLowerCaseChecks(t *testing.T) {
	p := AcquirePayload("Ab")
	defer ReleasePayload(p)
	UpperCase().Fn(p)
	if p.Issues[0].Format != "uppercase" {
		t.Fatalf("%+v", p.Issues[0])
	}
	p2 := AcquirePayload("Ab")
	defer ReleasePayload(p2)
	LowerCase().Fn(p2)
	if p2.Issues[0].Format != "lowercase" {
		t.Fatalf("%+v", p2.Issues[0])
	}
}
