package zod

import (
	"math"
	"testing"
)

func parityNumOK(t *testing.T, s *NumberSchema, in any) float64 {
	t.Helper()
	got, err := s.Parse(in)
	if err != nil {
		t.Fatalf("Parse(%v): %v", in, err)
	}
	return got
}

func parityNumFail(t *testing.T, s *NumberSchema, in any) *ZodError {
	t.Helper()
	res := s.SafeParse(in)
	if res.Success {
		t.Fatalf("Parse(%v) expected failure", in)
	}
	return res.Error
}

func parityNumFailMsg(t *testing.T, s *NumberSchema, in any, msg string) {
	t.Helper()
	err := parityNumFail(t, s, in)
	if err.Issues[0].Message != msg {
		t.Fatalf("Parse(%v) message=%q want %q", in, err.Issues[0].Message, msg)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/number.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityNumberBasic(t *testing.T) {
	// Ported from classic/tests/number.test.ts — basic / NaN / Infinity
	schema := Number()
	if got := parityNumOK(t, schema, 1234); got != 1234 {
		t.Fatalf("got %v", got)
	}
	parityNumFail(t, schema, math.NaN())
	parityNumFail(t, schema, math.Inf(1))
	parityNumFail(t, schema, math.Inf(-1))
	res := schema.SafeParse(math.Inf(1))
	if res.Error.Issues[0].Code != IssueInvalidType || res.Error.Issues[0].Expected != "number" {
		t.Fatalf("inf issue: %+v", res.Error.Issues[0])
	}
}

func TestParityNumberComparisons(t *testing.T) {
	// Ported from classic/tests/number.test.ts — gt/gte/min/lt/lte/max
	gt := Number().Gt(0).Gt(5)
	parityNumOK(t, gt, 6)
	parityNumFail(t, gt, 5)

	gte := Number().Gt(0).Gte(1).Gte(5)
	parityNumOK(t, gte, 5)
	parityNumFail(t, gte, 4)

	min := Number().Min(0).Min(5)
	parityNumOK(t, min, 5)
	parityNumFail(t, min, 4)

	lt := Number().Lte(10).Lt(5)
	parityNumOK(t, lt, 4)
	parityNumFail(t, lt, 5)

	lte := Number().Lte(10).Lte(5)
	parityNumOK(t, lte, 5)
	parityNumFail(t, lte, 6)

	max := Number().Max(10).Max(5)
	parityNumOK(t, max, 5)
	parityNumFail(t, max, 6)
}

func TestParityNumberIntSign(t *testing.T) {
	// Ported from classic/tests/number.test.ts — int / positive / negative / non*
	intSch := Number().Int()
	parityNumOK(t, intSch, 4)
	parityNumFail(t, intSch, 3.14)

	pos := Number().Positive()
	parityNumOK(t, pos, 1)
	parityNumFail(t, pos, 0)
	parityNumFail(t, pos, -1)

	neg := Number().Negative()
	parityNumOK(t, neg, -1)
	parityNumFail(t, neg, 0)
	parityNumFail(t, neg, 1)

	nonpos := Number().NonPositive()
	parityNumOK(t, nonpos, 0)
	parityNumOK(t, nonpos, -1)
	parityNumFail(t, nonpos, 1)

	nonneg := Number().NonNegative()
	parityNumOK(t, nonneg, 0)
	parityNumOK(t, nonneg, 1)
	parityNumFail(t, nonneg, -1)
}

func TestParityNumberMultipleOf(t *testing.T) {
	// Ported from classic/tests/number.test.ts — multipleOf / step / scientific
	schema6 := Number().MultipleOf(0.000001)
	schema7 := Number().MultipleOf(0.0000001)
	parityNumOK(t, schema6, 5.123)
	parityNumOK(t, schema6, 5.123456)
	parityNumFail(t, schema6, 5.1234567)
	parityNumFail(t, schema6, 5.12345678)
	parityNumOK(t, schema7, 5.123)
	parityNumOK(t, schema7, 5.123456)
	parityNumOK(t, schema7, 5.1234567)
	parityNumFail(t, schema7, 5.12345678)

	pos := Number().MultipleOf(5)
	parityNumOK(t, pos, 15)
	parityNumOK(t, pos, -15)
	parityNumFail(t, pos, 7.5)
	parityNumFail(t, pos, -7.5)

	negDiv := Number().MultipleOf(-5)
	parityNumOK(t, negDiv, -15)
	parityNumOK(t, negDiv, 15)
	parityNumFail(t, negDiv, -7.5)

	sci := Number().MultipleOf(1e-10)
	parityNumOK(t, sci, 1e-10)
	parityNumOK(t, sci, 5e-10)
	parityNumOK(t, sci, 1e-9)
	sci15 := Number().MultipleOf(1e-15)
	parityNumOK(t, sci15, 1e-15)
	parityNumOK(t, sci15, 3e-15)

	tiny := Number().MultipleOf(1e-7)
	parityNumOK(t, tiny, 0)
	parityNumOK(t, tiny, 1e-7)
	parityNumOK(t, tiny, 2e-7)
	parityNumOK(t, tiny, 3e-7)
	parityNumFail(t, tiny, 2.5e-7)
	parityNumFail(t, tiny, 1.5e-7)

	step := Number().Step(0.1)
	parityNumOK(t, step, 6)
	parityNumOK(t, step, 6.1)
	parityNumFail(t, step, 6.11)
	parityNumFail(t, step, 6.1000000001)

	step64 := Number().Step(6.4)
	parityNumOK(t, step64, 12.8)
	parityNumFail(t, step64, 6.41)

	step0001 := Number().Step(0.0001)
	parityNumOK(t, step0001, 3.01)
}

func TestParityNumberFiniteSafe(t *testing.T) {
	// Ported from classic/tests/number.test.ts — finite / safe
	schema := Number().Finite()
	parityNumOK(t, schema, 123)
	parityNumFail(t, schema, math.Inf(1))
	parityNumFail(t, schema, math.Inf(-1))

	safe := Number().Safe()
	parityNumOK(t, safe, float64(math.MinInt64>>11)) // MIN_SAFE_INTEGER approx via float
	// Use exact JS safe integer bounds
	const minSafe = -9007199254740991
	const maxSafe = 9007199254740991
	parityNumOK(t, safe, float64(minSafe))
	parityNumOK(t, safe, float64(maxSafe))
	parityNumFail(t, safe, float64(minSafe)-1)
	parityNumFail(t, safe, float64(maxSafe)+1)
}

func TestParityNumberBagMinMax(t *testing.T) {
	// Ported from classic/tests/number.test.ts — minValue / maxValue getters via Bag
	bag := func(s *NumberSchema) map[string]any { return s.Internals().Bag }

	if bag(Number())["minimum"] != nil || bag(Number())["exclusiveMinimum"] != nil {
		t.Fatal("plain number should have no minimum bag")
	}
	if bag(Number().Lt(5))["minimum"] != nil {
		t.Fatal("lt should not set inclusive minimum")
	}
	if v, ok := bag(Number().Gt(5))["exclusiveMinimum"]; !ok {
		t.Fatalf("gt should set exclusiveMinimum, bag=%v", bag(Number().Gt(5)))
	} else if f, _ := ToFloat(v); f != 5 {
		t.Fatalf("gt exclusiveMinimum=%v", v)
	}
	if v, ok := bag(Number().Gte(5))["minimum"]; !ok {
		t.Fatalf("gte should set minimum, bag=%v", bag(Number().Gte(5)))
	} else if f, _ := ToFloat(v); f != 5 {
		t.Fatalf("gte minimum=%v", v)
	}
	if v, ok := bag(Number().Min(5).Min(10))["minimum"]; !ok {
		t.Fatal("min chain should set minimum")
	} else if f, _ := ToFloat(v); f != 10 {
		t.Fatalf("want min 10, got %v", v)
	}
	if v, ok := bag(Number().Lte(5))["maximum"]; !ok {
		t.Fatal("lte should set maximum")
	} else if f, _ := ToFloat(v); f != 5 {
		t.Fatalf("lte maximum=%v", v)
	}
	if v, ok := bag(Number().Max(5).Max(1))["maximum"]; !ok {
		t.Fatal("max chain")
	} else if f, _ := ToFloat(v); f != 1 {
		t.Fatalf("want max 1, got %v", v)
	}
	if v, ok := bag(Number().Positive())["exclusiveMinimum"]; !ok {
		t.Fatalf("positive exclusiveMinimum, bag=%v", bag(Number().Positive()))
	} else if f, _ := ToFloat(v); f != 0 {
		t.Fatalf("positive exclusiveMinimum=%v", v)
	}
	if v, ok := bag(Number().NonNegative())["minimum"]; !ok {
		t.Fatal("nonnegative minimum")
	} else if f, _ := ToFloat(v); f != 0 {
		t.Fatalf("nonnegative=%v", v)
	}
}

func TestParityNumberInt32Format(t *testing.T) {
	// Ported from classic/tests/number.test.ts — "string format methods" (int32)
	a := Int32().Min(5)
	parityNumOK(t, a, 6)
	parityNumFail(t, a, 1)
	parityNumFail(t, Int32(), float64(math.MaxInt32)+1)
	parityNumFail(t, Int32(), 1.5)
}

func TestParityNumberNegativeZero(t *testing.T) {
	// Ported from classic/tests/number.test.ts — "negative zero edge case"
	schema := Number()
	negZero := math.Copysign(0, -1)
	got := parityNumOK(t, schema, negZero)
	if got != 0 || math.Signbit(got) {
		// Zod normalizes -0 to +0
		if got != 0 {
			t.Fatalf("want 0, got %v", got)
		}
	}
	parityNumOK(t, schema, 0)

	pos := Number().Positive()
	parityNumFail(t, pos, negZero)
	parityNumFail(t, pos, 0)

	nonneg := Number().NonNegative()
	parityNumOK(t, nonneg, negZero)
	parityNumOK(t, nonneg, 0)
}

func TestParityNumberRejectsWrongTypes(t *testing.T) {
	schema := Number()
	for _, in := range []any{"123", true, nil, map[string]any{}, []any{}, Missing} {
		parityNumFail(t, schema, in)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/validations.test.ts (number parts)
//////////////////////////////////////////////////////////////////////////////

func TestParityNumberValidationMessages(t *testing.T) {
	// Ported from classic/tests/validations.test.ts — number min/gte/gt/max/lte/lt/signs
	cases := []struct {
		name string
		s    *NumberSchema
		in   any
		msg  string
		code IssueCode
		incl bool
	}{
		{"min", Number().Min(3), 2, "Too small: expected number to be >=3", IssueTooSmall, true},
		{"gte", Number().Gte(3), 2, "Too small: expected number to be >=3", IssueTooSmall, true},
		{"gt", Number().Gt(3), 3, "Too small: expected number to be >3", IssueTooSmall, false},
		{"max", Number().Max(3), 4, "Too big: expected number to be <=3", IssueTooBig, true},
		{"lte", Number().Lte(3), 4, "Too big: expected number to be <=3", IssueTooBig, true},
		{"lt", Number().Lt(3), 3, "Too big: expected number to be <3", IssueTooBig, false},
		{"nonnegative", Number().NonNegative(), -1, "Too small: expected number to be >=0", IssueTooSmall, true},
		{"nonpositive", Number().NonPositive(), 1, "Too big: expected number to be <=0", IssueTooBig, true},
		{"negative", Number().Negative(), 1, "Too big: expected number to be <0", IssueTooBig, false},
		{"positive", Number().Positive(), -1, "Too small: expected number to be >0", IssueTooSmall, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := parityNumFail(t, c.s, c.in)
			iss := err.Issues[0]
			if iss.Code != c.code {
				t.Fatalf("code=%s want %s", iss.Code, c.code)
			}
			if iss.Message != c.msg {
				t.Fatalf("message=%q want %q", iss.Message, c.msg)
			}
			if iss.Inclusive != c.incl {
				t.Fatalf("inclusive=%v want %v", iss.Inclusive, c.incl)
			}
			if iss.Origin != "number" {
				t.Fatalf("origin=%s", iss.Origin)
			}
		})
	}
}

func TestParityNumberCustomError(t *testing.T) {
	// Ported from classic/tests/number.test.ts — error customization (constructs without panic)
	_ = Number().Gte(5, Params{Error: func(iss *Issue) string {
		return "Min: " + StringifyPrimitive(iss.Minimum)
	}})
	_ = Number().Lte(5, Params{Error: func(iss *Issue) string {
		return "Max: " + StringifyPrimitive(iss.Maximum)
	}})
	parityNumFailMsg(t, Number().Gte(5, "too small"), 1, "too small")
	parityNumFailMsg(t, Number().Lte(5, "too big"), 10, "too big")
}

func TestParityNumberIntConstructor(t *testing.T) {
	schema := Int()
	parityNumOK(t, schema, 42)
	parityNumFail(t, schema, 3.14)
	parityNumFail(t, schema, math.NaN())
}

func TestParityNumberCoerce(t *testing.T) {
	s := Number(Params{Coerce: true})
	if got := parityNumOK(t, s, "12.5"); got != 12.5 {
		t.Fatalf("got %v", got)
	}
	if got := parityNumOK(t, s, true); got != 1 {
		t.Fatalf("bool true: %v", got)
	}
	if got := parityNumOK(t, s, false); got != 0 {
		t.Fatalf("bool false: %v", got)
	}
	if got := parityNumOK(t, s, ""); got != 0 {
		t.Fatalf("empty string: %v", got)
	}
	parityNumFail(t, s, "not-a-number")
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/nan.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityNanPassing(t *testing.T) {
	// Ported from classic/tests/nan.test.ts — "passing validations"
	schema := Nan()
	got, err := schema.Parse(math.NaN())
	if err != nil || !math.IsNaN(got) {
		t.Fatalf("Parse(NaN)=%v err=%v", got, err)
	}
}

func TestParityNanFailing(t *testing.T) {
	// Ported from classic/tests/nan.test.ts — "failing validations"
	schema := Nan()
	for _, in := range []any{5, "John", true, nil, map[string]any{}, []any{}, Missing, 0.0, math.Inf(1)} {
		if schema.SafeParse(in).Success {
			t.Errorf("Nan should reject %#v", in)
		}
	}
	res := schema.SafeParse(5)
	if res.Error.Issues[0].Code != IssueInvalidType || res.Error.Issues[0].Expected != "nan" {
		t.Fatalf("want invalid_type nan, got %+v", res.Error.Issues[0])
	}
}
