package z

import (
	"context"
	"testing"
)

func TestEnumWithParamsNotValues(t *testing.T) {
	s := EnumWith([]string{"active", "inactive"}, "must be a status")
	if s.SafeParse("must be a status").Success {
		t.Fatal("error message must not be an accepted enum value")
	}
	if !s.SafeParse("active").Success {
		t.Fatal("active should pass")
	}
	res := s.SafeParse("nope")
	if res.Success || res.Error == nil || res.Error.Issues[0].Message != "must be a status" {
		t.Fatalf("want custom message, got %#v", res)
	}
}

func TestNormalizeParamsUnknownPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unsupported params type")
		}
	}()
	_ = normalizeParams([]any{123})
}

func TestDefaultClonesSlice(t *testing.T) {
	schema := Default(Array(String()), []any{})
	a, err := schema.Parse(Missing)
	if err != nil {
		t.Fatal(err)
	}
	sliceA, ok := a.([]any)
	if !ok {
		t.Fatalf("got %T", a)
	}
	sliceA = append(sliceA, "mutated")
	_ = sliceA

	b, err := schema.Parse(Missing)
	if err != nil {
		t.Fatal(err)
	}
	sliceB, ok := b.([]any)
	if !ok {
		t.Fatalf("got %T", b)
	}
	if len(sliceB) != 0 {
		t.Fatalf("default slice was shared/mutated: %#v", sliceB)
	}
}

func TestInt64TypedOutput(t *testing.T) {
	s := Int64().Gte(0).Lt(150)
	got, err := s.Parse(float64(42))
	if err != nil || got != 42 {
		t.Fatalf("got %v, %v", got, err)
	}
	if s.SafeParse(3.5).Success {
		t.Fatal("fractional float must fail")
	}
}

func TestObjectOrderedIssueOrder(t *testing.T) {
	schema := ObjectOrdered([]Field{
		{Name: "zebra", Schema: String().Min(2)},
		{Name: "alpha", Schema: String().Min(2)},
	})
	res := schema.SafeParse(map[string]any{"zebra": "x", "alpha": "y"})
	if res.Success || res.Error == nil || len(res.Error.Issues) < 2 {
		t.Fatalf("expected two issues, got %#v", res)
	}
	if res.Error.Issues[0].Path[0] != "zebra" {
		t.Fatalf("first issue path=%v, want zebra", res.Error.Issues[0].Path)
	}
	if res.Error.Issues[1].Path[0] != "alpha" {
		t.Fatalf("second issue path=%v, want alpha", res.Error.Issues[1].Path)
	}
}

func TestSuperRefineSeesParseContext(t *testing.T) {
	type ctxKey struct{}
	schema := String().SuperRefine(func(_ string, rctx *RefinementCtx) {
		if rctx.Context().Value(ctxKey{}) != "ok" {
			rctx.AddMessage("missing ctx")
		}
	})
	ctx := &ParseCtx{Context: context.WithValue(context.Background(), ctxKey{}, "ok")}
	if _, err := schema.ParseCtx("hello", ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMissingReassignDoesNotBreakInternals(t *testing.T) {
	prev := Missing
	Missing = nil
	defer func() { Missing = prev }()

	schema := Object(Shape{"name": String().Min(1)})
	res := schema.SafeParse(map[string]any{})
	if res.Success {
		t.Fatal("absent key should fail String, not be treated as nil")
	}
	// Optional still accepts the real sentinel via IsMissing path.
	opt := Optional(String())
	if _, err := opt.Parse(missingSentinel); err != nil {
		t.Fatalf("optional missingSentinel: %v", err)
	}
}
