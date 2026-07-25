package zod

import (
	"math"
	"testing"
)

// Ported from classic/tests/number.test.ts

func TestNumberBasic(t *testing.T) {
	schema := Number()
	got, err := schema.Parse(1234)
	if err != nil || got != 1234 {
		t.Fatalf("got %v err=%v", got, err)
	}
	// Accept integer Go types, normalize to float64.
	got, err = schema.Parse(42)
	if err != nil || got != 42 {
		t.Fatalf("int: got %v err=%v", got, err)
	}
}

func TestNumberNaNAndInfinity(t *testing.T) {
	schema := Number()
	res := schema.SafeParse(math.NaN())
	if res.Success {
		t.Fatal("NaN should fail")
	}
	if res.Error.Issues[0].Message != "Invalid input: expected number, received NaN" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}

	res = schema.SafeParse(math.Inf(1))
	if res.Success {
		t.Fatal("+Inf should fail")
	}
	if res.Error.Issues[0].Code != IssueInvalidType || res.Error.Issues[0].Expected != "number" {
		t.Fatalf("got %+v", res.Error.Issues[0])
	}
	res = schema.SafeParse(math.Inf(-1))
	if res.Success {
		t.Fatal("-Inf should fail")
	}
}

func TestNumberComparisons(t *testing.T) {
	// .gt()
	schema := Number().Gt(0).Gt(5)
	mustParseNum(t, schema, 6)
	mustFailNum(t, schema, 5)

	// .gte() / .min()
	schema = Number().Gt(0).Gte(1).Gte(5)
	mustParseNum(t, schema, 5)
	mustFailNum(t, schema, 4)
	schema = Number().Min(0).Min(5)
	mustParseNum(t, schema, 5)
	mustFailNum(t, schema, 4)

	// .lt() / .lte() / .max()
	schema = Number().Lte(10).Lt(5)
	mustParseNum(t, schema, 4)
	mustFailNum(t, schema, 5)
	schema = Number().Lte(10).Lte(5)
	mustParseNum(t, schema, 5)
	mustFailNum(t, schema, 6)
	schema = Number().Max(10).Max(5)
	mustParseNum(t, schema, 5)
	mustFailNum(t, schema, 6)
}

func TestNumberInt(t *testing.T) {
	schema := Number().Int()
	mustParseNum(t, schema, 4)
	mustFailNum(t, schema, 3.14)

	res := schema.SafeParse(3.14)
	if res.Success || res.Error.Issues[0].Expected != "int" {
		t.Fatalf("want expected=int, got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "Invalid input: expected int, received number" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}

	// Top-level Int()
	mustParseNum(t, Int(), 10)
	mustFailNum(t, Int(), 1.5)
}

func TestNumberSignChecks(t *testing.T) {
	pos := Number().Positive()
	mustParseNum(t, pos, 1)
	mustFailNum(t, pos, 0)
	mustFailNum(t, pos, -1)

	neg := Number().Negative()
	mustParseNum(t, neg, -1)
	mustFailNum(t, neg, 0)
	mustFailNum(t, neg, 1)

	np := Number().NonPositive()
	mustParseNum(t, np, 0)
	mustParseNum(t, np, -1)
	mustFailNum(t, np, 1)

	nn := Number().NonNegative()
	mustParseNum(t, nn, 0)
	mustParseNum(t, nn, 1)
	mustFailNum(t, nn, -1)
}

func TestNumberMultipleOf(t *testing.T) {
	schema := Number().MultipleOf(5)
	mustParseNum(t, schema, 15)
	mustParseNum(t, schema, -15)
	mustFailNum(t, schema, 7.5)

	schema = Number().MultipleOf(-5)
	mustParseNum(t, schema, -15)
	mustParseNum(t, schema, 15)

	// Scientific / small floats (ported from number.test.ts)
	schema = Number().MultipleOf(1e-10)
	mustParseNum(t, schema, 1e-10)
	mustParseNum(t, schema, 5e-10)
	mustParseNum(t, schema, 1e-9)

	schema = Number().MultipleOf(1e-7)
	mustParseNum(t, schema, 0)
	mustParseNum(t, schema, 1e-7)
	mustParseNum(t, schema, 2e-7)
	mustFailNum(t, schema, 2.5e-7)

	res := Number().MultipleOf(5).SafeParse(7)
	if res.Success || res.Error.Issues[0].Code != IssueNotMultipleOf {
		t.Fatalf("got %+v", res.Error)
	}
	if res.Error.Issues[0].Message != "Invalid number: must be a multiple of 5" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestNumberStep(t *testing.T) {
	schema := Number().Step(0.1)
	mustParseNum(t, schema, 6)
	mustParseNum(t, schema, 6.1)
	mustFailNum(t, schema, 6.11)
}

func TestNumberSafeAndFinite(t *testing.T) {
	schema := Number().Safe()
	mustParseNum(t, schema, MaxSafeInteger)
	mustParseNum(t, schema, MinSafeInteger)
	mustFailNum(t, schema, MaxSafeInteger+1)
	mustFailNum(t, schema, MinSafeInteger-1)

	// Finite is a no-op; Inf still fails at type level.
	res := Number().Finite().SafeParse(math.Inf(1))
	if res.Success {
		t.Fatal("Inf should fail")
	}
}

func TestNumberTooSmallTooBigMessages(t *testing.T) {
	res := Number().Gte(5).SafeParse(4)
	if res.Success || res.Error.Issues[0].Message != "Too small: expected number to be >=5" {
		t.Fatalf("got %+v", res.Error)
	}
	res = Number().Gt(5).SafeParse(5)
	if res.Success || res.Error.Issues[0].Message != "Too small: expected number to be >5" {
		t.Fatalf("got %+v", res.Error)
	}
	res = Number().Lte(5).SafeParse(6)
	if res.Success || res.Error.Issues[0].Message != "Too big: expected number to be <=5" {
		t.Fatalf("got %+v", res.Error)
	}
	res = Number().Lt(5).SafeParse(5)
	if res.Success || res.Error.Issues[0].Message != "Too big: expected number to be <5" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestNumberCoerce(t *testing.T) {
	schema := Number(Params{Coerce: true})
	got, err := schema.Parse("12")
	if err != nil || got != 12 {
		t.Fatalf("got %v err=%v", got, err)
	}
	got, err = schema.Parse(true)
	if err != nil || got != 1 {
		t.Fatalf("bool true: got %v err=%v", got, err)
	}
	got, err = schema.Parse(false)
	if err != nil || got != 0 {
		t.Fatalf("bool false: got %v err=%v", got, err)
	}
	got, err = schema.Parse("")
	if err != nil || got != 0 {
		t.Fatalf("empty string: got %v err=%v", got, err)
	}
	mustFailNum(t, schema, "NOT_A_NUMBER")
}

func TestNumberNegativeZero(t *testing.T) {
	schema := Number()
	got, err := schema.Parse(math.Copysign(0, -1))
	if err != nil || got != 0 || math.Signbit(got) {
		t.Fatalf("want +0, got %v signbit=%v err=%v", got, math.Signbit(got), err)
	}
	mustFailNum(t, Number().Positive(), 0)
	mustParseNum(t, Number().NonNegative(), 0)
}

func TestNumberTypeMessage(t *testing.T) {
	res := Number().SafeParse("x")
	if res.Success || res.Error.Issues[0].Message != "Invalid input: expected number, received string" {
		t.Fatalf("got %+v", res.Error)
	}
}

func mustParseNum(t *testing.T, s *NumberSchema, in any) {
	t.Helper()
	if _, err := s.Parse(in); err != nil {
		t.Fatalf("Parse(%v) unexpected error: %v", in, err)
	}
}

func mustFailNum(t *testing.T, s *NumberSchema, in any) {
	t.Helper()
	if res := s.SafeParse(in); res.Success {
		t.Fatalf("Parse(%v) expected failure", in)
	}
}
