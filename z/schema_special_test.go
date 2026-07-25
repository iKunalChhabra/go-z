package z

import (
	"math"
	"math/big"
	"testing"
)

// Ported from classic/tests/nan.test.ts (+ never/null)

func TestNever(t *testing.T) {
	res := Never().SafeParse("x")
	if res.Success {
		t.Fatal("expected fail")
	}
	if res.Error.Issues[0].Expected != "never" {
		t.Fatalf("got %+v", res.Error.Issues[0])
	}
	if res.Error.Issues[0].Message != "Invalid input: expected never, received string" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
	if Never().SafeParse(nil).Success {
		t.Fatal("nil should also fail never")
	}
}

func TestNilNull(t *testing.T) {
	for _, s := range []*NilSchema{Nil(), Null()} {
		got, err := s.Parse(nil)
		if err != nil || got != nil {
			t.Fatalf("got %v err=%v", got, err)
		}
		res := s.SafeParse("x")
		if res.Success || res.Error.Issues[0].Expected != "null" {
			t.Fatalf("got %+v", res.Error)
		}
		if res.Error.Issues[0].Message != "Invalid input: expected null, received string" {
			t.Fatalf("got %q", res.Error.Issues[0].Message)
		}
		if _, ok := s.Internals().Values[nil]; !ok {
			t.Fatal("Values should contain nil")
		}
	}
}

func TestNan(t *testing.T) {
	s := Nan()
	got, err := s.Parse(math.NaN())
	if err != nil || !math.IsNaN(got) {
		t.Fatalf("got %v err=%v", got, err)
	}
	if s.SafeParse(5).Success {
		t.Fatal("5 should fail")
	}
	if s.SafeParse("John").Success {
		t.Fatal("string should fail")
	}
	if s.SafeParse(true).Success {
		t.Fatal("bool should fail")
	}
	if s.SafeParse(nil).Success {
		t.Fatal("nil should fail")
	}
	res := s.SafeParse(1.0)
	if res.Error.Issues[0].Message != "Invalid input: expected NaN, received number" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestBigIntBasic(t *testing.T) {
	// Ported from classic/tests/bigint.test.ts
	BigInt().MustParse(big.NewInt(1))
	BigInt().MustParse(big.NewInt(0))
	BigInt().MustParse(big.NewInt(-1))
	BigInt().MustParse(int64(7))

	gtFive := BigInt().Gt(big.NewInt(5))
	gtFive.MustParse(big.NewInt(6))
	if gtFive.SafeParse(big.NewInt(5)).Success {
		t.Fatal("gt 5")
	}

	gteFive := BigInt().Gte(big.NewInt(5))
	gteFive.MustParse(big.NewInt(5))
	gteFive.MustParse(big.NewInt(6))

	ltFive := BigInt().Lt(big.NewInt(5))
	ltFive.MustParse(big.NewInt(4))
	if ltFive.SafeParse(big.NewInt(5)).Success {
		t.Fatal("lt 5")
	}

	pos := BigInt().Positive()
	pos.MustParse(big.NewInt(3))
	if pos.SafeParse(big.NewInt(0)).Success {
		t.Fatal("positive 0")
	}

	neg := BigInt().Negative()
	neg.MustParse(big.NewInt(-2))

	nn := BigInt().NonNegative()
	nn.MustParse(big.NewInt(0))
	nn.MustParse(big.NewInt(7))

	np := BigInt().NonPositive()
	np.MustParse(big.NewInt(0))
	np.MustParse(big.NewInt(-12))

	mult := BigInt().MultipleOf(big.NewInt(5))
	mult.MustParse(big.NewInt(15))
	if mult.SafeParse(big.NewInt(13)).Success {
		t.Fatal("multipleOf")
	}

	res := BigInt().SafeParse("x")
	if res.Success || res.Error.Issues[0].Message != "Invalid input: expected bigint, received string" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestBigIntCoerce(t *testing.T) {
	s := BigInt(Params{Coerce: true})
	got, err := s.Parse("5")
	if err != nil || got.Cmp(big.NewInt(5)) != 0 {
		t.Fatalf("got %v err=%v", got, err)
	}
	got, err = s.Parse(true)
	if err != nil || got.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("got %v err=%v", got, err)
	}
}

func TestUndefined(t *testing.T) {
	s := Undefined()
	got, err := s.Parse(Missing)
	if err != nil || !IsMissing(got) {
		t.Fatalf("got %v err=%v", got, err)
	}
	for _, in := range []any{nil, "undefined", 1, false, map[string]any{}} {
		res := s.SafeParse(in)
		if res.Success || res.Error.Issues[0].Expected != "undefined" {
			t.Fatalf("%v: %+v", in, res.Error)
		}
	}
	res := s.SafeParse("x")
	if res.Error.Issues[0].Message != "Invalid input: expected undefined, received string" {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
}

func TestVoid(t *testing.T) {
	s := Void()
	got, err := s.Parse(Missing)
	if err != nil || !IsMissing(got) {
		t.Fatalf("got %v err=%v", got, err)
	}
	if s.SafeParse(nil).Success {
		t.Fatal("null should fail void")
	}
	if s.SafeParse("").Success {
		t.Fatal("empty string should fail void")
	}
	res := s.SafeParse(1)
	if res.Error.Issues[0].Expected != "void" {
		t.Fatalf("got %+v", res.Error.Issues[0])
	}
}

func TestJSON(t *testing.T) {
	s := JSON()
	for _, in := range []any{
		"hello",
		123.0,
		true,
		nil,
		map[string]any{},
		map[string]any{"a": "hello"},
		[]any{1.0, 2.0, 3.0},
		[]any{map[string]any{"a": "hello"}},
	} {
		if _, err := s.Parse(in); err != nil {
			t.Fatalf("Parse(%v): %v", in, err)
		}
	}
	for _, in := range []any{
		Missing,
		map[string]any{"a": Missing},
		math.NaN(),
		math.Inf(1),
		func() {},
	} {
		if s.SafeParse(in).Success {
			t.Fatalf("expected fail for %v", in)
		}
	}
}

func TestStringBool(t *testing.T) {
	s := StringBool()
	for _, in := range []string{"true", "yes", "1", "on", "TRUE", "Y", "enabled"} {
		got, err := s.Parse(in)
		if err != nil || !got {
			t.Fatalf("%q: %v %v", in, got, err)
		}
	}
	for _, in := range []string{"false", "no", "0", "off", "FALSE", "n", "disabled"} {
		got, err := s.Parse(in)
		if err != nil || got {
			t.Fatalf("%q: %v %v", in, got, err)
		}
	}
	for _, in := range []any{"other", "", Missing, map[string]any{}, true, false} {
		if s.SafeParse(in).Success {
			t.Fatalf("expected fail for %#v", in)
		}
	}
}

func TestCoerceBigIntNS(t *testing.T) {
	s := Coerce.BigInt()
	got, err := s.Parse("42")
	if err != nil || got.Cmp(big.NewInt(42)) != 0 {
		t.Fatalf("got %v err=%v", got, err)
	}
}
