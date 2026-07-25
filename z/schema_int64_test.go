package z

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"
)

// Int64 exists to carry integers float64 cannot represent, so its bounds have
// to compare exactly. Routing them through float64 rounds MaxInt64 up to
// 9223372036854775808, which made the neighbourhood of the bound unusable.
func TestInt64BoundsAreExactAtTheEdge(t *testing.T) {
	const max = int64(math.MaxInt64)
	const min = int64(math.MinInt64)

	if _, err := Int64().Lt(max).Parse(max - 1); err != nil {
		t.Fatalf("MaxInt64-1 must satisfy Lt(MaxInt64): %v", err)
	}
	if Int64().Lt(max).SafeParse(max).Success {
		t.Fatal("MaxInt64 must not satisfy Lt(MaxInt64)")
	}
	if _, err := Int64().Lte(max).Parse(max); err != nil {
		t.Fatalf("MaxInt64 must satisfy Lte(MaxInt64): %v", err)
	}
	if _, err := Int64().Gt(min).Parse(min + 1); err != nil {
		t.Fatalf("MinInt64+1 must satisfy Gt(MinInt64): %v", err)
	}
	if Int64().Gt(min).SafeParse(min).Success {
		t.Fatal("MinInt64 must not satisfy Gt(MinInt64)")
	}
	if _, err := Int64().Gte(min).Parse(min); err != nil {
		t.Fatalf("MinInt64 must satisfy Gte(MinInt64): %v", err)
	}

	// Just above the safe-integer range, where float64 starts skipping values.
	const near = int64(1)<<53 + 1
	if _, err := Int64().Gte(near).Parse(near); err != nil {
		t.Fatalf("2^53+1 must satisfy Gte(2^53+1): %v", err)
	}
	if Int64().Gte(near).SafeParse(near - 1).Success {
		t.Fatal("2^53 must not satisfy Gte(2^53+1)")
	}
	if Int64().Lte(near).SafeParse(near + 1).Success {
		t.Fatal("2^53+2 must not satisfy Lte(2^53+1)")
	}
}

// The bound reported on the issue stays exact too, so an error message about a
// large bound does not name a different number than the schema was built with.
func TestInt64IssueBoundIsExact(t *testing.T) {
	const bound = int64(1)<<62 + 1
	res := Int64().Gte(bound).SafeParse(int64(1))
	if res.Success {
		t.Fatal("expected too_small")
	}
	iss := res.Error.Issues[0]
	if got, ok := iss.Minimum.(int64); !ok || got != bound {
		t.Fatalf("Minimum = %#v, want int64(%d)", iss.Minimum, bound)
	}

	// Small bounds keep reporting as float64, the JSON number model.
	res = Int64().Gte(5).SafeParse(int64(1))
	if res.Error.Issues[0].Minimum != float64(5) {
		t.Fatalf("Minimum = %#v, want float64(5)", res.Error.Issues[0].Minimum)
	}
	if msg := res.Error.Issues[0].Message; msg != "Too small: expected number to be >=5" {
		t.Fatalf("message = %q", msg)
	}
}

func TestInt64ConstraintMethods(t *testing.T) {
	if _, err := Int64().Gt(5).Parse(6); err != nil {
		t.Fatal(err)
	}
	if Int64().Gt(5).SafeParse(5).Success {
		t.Fatal("Gt is exclusive")
	}
	if _, err := Int64().Min(5).Parse(5); err != nil {
		t.Fatal(err)
	}
	if _, err := Int64().Max(5).Parse(5); err != nil {
		t.Fatal(err)
	}
	if Int64().Max(5).SafeParse(6).Success {
		t.Fatal("Max is inclusive")
	}
	if _, err := Int64().Positive().Parse(1); err != nil {
		t.Fatal(err)
	}
	if Int64().Positive().SafeParse(0).Success {
		t.Fatal("0 is not positive")
	}
	if _, err := Int64().Negative().Parse(-1); err != nil {
		t.Fatal(err)
	}
	if _, err := Int64().NonNegative().Parse(0); err != nil {
		t.Fatal(err)
	}
	if _, err := Int64().NonPositive().Parse(0); err != nil {
		t.Fatal(err)
	}
	if _, err := Int64().Gte(1).Lte(10).Parse(5); err != nil {
		t.Fatal(err)
	}
}

// MultipleOf uses integer remainder for integer schemas, so it stays exact
// where floatSafeRemainder's tolerance would call anything close enough.
func TestInt64MultipleOfIsExact(t *testing.T) {
	if _, err := Int64().MultipleOf(3).Parse(9); err != nil {
		t.Fatal(err)
	}
	res := Int64().MultipleOf(3).SafeParse(10)
	if res.Success || res.Error.Issues[0].Code != IssueNotMultipleOf {
		t.Fatalf("got %+v", res)
	}

	const big = int64(1) << 60
	if _, err := Int64().MultipleOf(big).Parse(big * 4); err != nil {
		t.Fatalf("exact multiple above 2^53: %v", err)
	}
	if Int64().MultipleOf(big).SafeParse(big*4 + 1).Success {
		t.Fatal("2^60*4+1 is not a multiple of 2^60")
	}
}

func TestInt64AcceptedInputs(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{int64(7), 7},
		{int(7), 7},
		{int32(7), 7},
		{uint8(7), 7},
		{uint64(7), 7},
		{float64(7), 7},
		{float32(7), 7},
		{big.NewInt(7), 7},
		{json.Number("7"), 7},
		{json.Number("9007199254740993"), 9007199254740993},
	}
	for _, c := range cases {
		got, err := Int64().Parse(c.in)
		if err != nil || got != c.want {
			t.Fatalf("Parse(%#v) = %d, %v", c.in, got, err)
		}
	}
	for _, in := range []any{"7", true, nil, 7.5, math.NaN(), math.Inf(1), uint64(math.MaxUint64)} {
		if res := Int64().SafeParse(in); res.Success {
			t.Fatalf("Parse(%#v) should fail", in)
		}
	}
}

// A named type behaves like its underlying kind: the bound, the range and the
// output are all resolved from the kind, not from a type switch on the value.
type testPort uint16

func TestNumericOfNamedType(t *testing.T) {
	schema := NumericOf[testPort]().Gte(1)
	got, err := schema.Parse(8080)
	if err != nil || got != testPort(8080) {
		t.Fatalf("got %#v %v", got, err)
	}
	if schema.SafeParse(0).Success {
		t.Fatal("0 is below the bound")
	}
	if NumericOf[testPort]().SafeParse(65536).Success {
		t.Fatal("65536 does not fit in uint16")
	}
	if NumericOf[testPort]().SafeParse(-1).Success {
		t.Fatal("-1 does not fit in uint16")
	}
	res := NumericOf[testPort]().SafeParse("nope")
	if res.Success || res.Error.Issues[0].Expected != "uint16" {
		t.Fatalf("got %+v", res)
	}

	if _, err := NumericOf[uint64]().Parse(uint64(math.MaxUint64)); err != nil {
		t.Fatalf("uint64 max must fit a uint64 schema: %v", err)
	}
}
