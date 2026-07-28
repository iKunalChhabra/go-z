package z

import (
	"math/big"
	"testing"
	"time"
)

func TestNumberFormatRanges(t *testing.T) {
	mustParseNum(t, Int32(), 2147483647)
	mustFailNum(t, Int32(), 2147483648)
	mustParseNum(t, Uint32(), 0)
	mustFailNum(t, Uint32(), -1)
	mustParseNum(t, Float32(), 1.5)
	mustFailNum(t, Float32(), MaxFloat32Exact*2)
}

func TestGreaterThanLessThanOrigins(t *testing.T) {
	// number origin
	res := Number().Gt(10).SafeParse(5)
	if res.Error.Issues[0].Origin != "number" || res.Error.Issues[0].Inclusive {
		t.Fatalf("got %+v", res.Error.Issues[0])
	}

	// date origin + millisecond bound (ported from date.test.ts)
	benchmark := time.Date(2022, 11, 5, 0, 0, 0, 0, time.UTC)
	before := time.Date(2022, 11, 4, 0, 0, 0, 0, time.UTC)
	resT := Time().Min(benchmark).SafeParse(before)
	if resT.Success {
		t.Fatal("expected fail")
	}
	iss := resT.Error.Issues[0]
	if iss.Code != IssueTooSmall || iss.Origin != "date" || !iss.Inclusive {
		t.Fatalf("got %+v", iss)
	}
	wantMin := float64(benchmark.UnixMilli())
	if iss.Minimum != wantMin {
		t.Fatalf("minimum=%v want %v", iss.Minimum, wantMin)
	}
	if iss.Message != "Too small: expected date to be >=1667606400000" {
		t.Fatalf("got %q", iss.Message)
	}

	// bigint origin
	resB := BigInt().Gt(big.NewInt(5)).SafeParse(big.NewInt(5))
	if resB.Success || resB.Error.Issues[0].Origin != "bigint" {
		t.Fatalf("got %+v", resB.Error)
	}
	if resB.Error.Issues[0].Message != "Too small: expected bigint to be >5" {
		t.Fatalf("got %q", resB.Error.Issues[0].Message)
	}
}

func TestFloatSafeRemainder(t *testing.T) {
	if floatSafeRemainder(5.123, 0.000001) != 0 {
		t.Fatal("5.123 should be multiple of 1e-6")
	}
	if floatSafeRemainder(5.1234567, 0.000001) == 0 {
		t.Fatal("5.1234567 should NOT be multiple of 1e-6")
	}
}

func TestMultipleOfZeroDivisorFloatPasses(t *testing.T) {
	// Regression: MultipleOf(0) on floats reported not_multiple_of via fmod
	// by zero, while the int64/uint64 variants treated a zero divisor as pass.
	sch := Number().Check(MultipleOf(0))
	if _, err := sch.Parse(3.7); err != nil {
		t.Fatalf("zero divisor should pass: %v", err)
	}
	if _, err := sch.Parse(0.0); err != nil {
		t.Fatalf("zero divisor should pass: %v", err)
	}
}
