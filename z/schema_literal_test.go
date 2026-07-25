package z

import (
	"math/big"
	"testing"
)

// Ported from classic/tests/literal.test.ts

func TestLiteralPassing(t *testing.T) {
	Literal("tuna").MustParse("tuna")
	Literal(42).MustParse(42)
	Literal(42).MustParse(float64(42)) // JSON number
	Literal(true).MustParse(true)
}

func TestLiteralFailing(t *testing.T) {
	res := Literal("tuna").SafeParse("shark")
	if res.Success {
		t.Fatal("expected fail")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidValue {
		t.Fatalf("code=%s", iss.Code)
	}
	if iss.Message != `Invalid input: expected "tuna"` {
		t.Fatalf("got %q", iss.Message)
	}
	if len(iss.Values) != 1 || iss.Values[0] != "tuna" {
		t.Fatalf("values=%v", iss.Values)
	}

	if Literal(42).SafeParse(43).Success {
		t.Fatal("expected fail")
	}
	if Literal(true).SafeParse(false).Success {
		t.Fatal("expected fail")
	}
}

func TestLiteralCustomMessage(t *testing.T) {
	s := Literal("tuna", "That's not a tuna")
	res := s.SafeParse("shark")
	if res.Success || res.Error.Issues[0].Message != "That's not a tuna" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestLiteralMultiValue(t *testing.T) {
	s := Literal("a", "b", 1)
	s.MustParse("a")
	s.MustParse("b")
	s.MustParse(1)
	res := s.SafeParse("c")
	if res.Success {
		t.Fatal("expected fail")
	}
	if res.Error.Issues[0].Message != `Invalid option: expected one of "a"|"b"|1` {
		t.Fatalf("got %q", res.Error.Issues[0].Message)
	}
	if s.Internals().Values == nil {
		t.Fatal("Values should be set for discriminant support")
	}
	if _, ok := s.Internals().Values["a"]; !ok {
		t.Fatal("missing value a in Internals.Values")
	}
}

func TestLiteralBigInt(t *testing.T) {
	s := Literal(big.NewInt(12))
	s.MustParse(big.NewInt(12))
	res := s.SafeParse(big.NewInt(13))
	if res.Success || res.Error.Issues[0].Message != "Invalid input: expected 12n" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestLiteralValueGetter(t *testing.T) {
	if Literal("tuna").Value() != "tuna" {
		t.Fatal("Value()")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for multi-value .Value()")
		}
	}()
	Literal(1, 2, 3).Value()
}

func TestLiteralSliceArg(t *testing.T) {
	s := Literal([]any{"x", "y"})
	s.MustParse("x")
	s.MustParse("y")
	if s.SafeParse("z").Success {
		t.Fatal("expected fail")
	}
}
