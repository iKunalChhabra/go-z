package z

import "testing"

// Ported from v4/classic/tests/catch.test.ts (subset).

func TestCatchBasic(t *testing.T) {
	schema := Catch(String(), "default")
	got, err := schema.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestCatchFnNotCalledOnSuccess(t *testing.T) {
	called := false
	schema := CatchFunc(String(), func(CatchCtx) any {
		called = true
		return "asdf"
	})
	if got, err := schema.Parse("test"); err != nil || got != "test" {
		t.Fatalf("got %v, %v", got, err)
	}
	if called {
		t.Fatal("catch fn must not run on success")
	}
}

func TestCatchWrongTypes(t *testing.T) {
	schema := Catch(String(), "default")
	for _, v := range []any{true, 15, []any{}, map[string]any{}} {
		got, err := schema.Parse(v)
		if err != nil || got != "default" {
			t.Fatalf("Parse(%v) = %v, %v", v, got, err)
		}
	}
}

func TestCatchWithTransform(t *testing.T) {
	inner := Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
		return v.(string) + "!", nil
	})
	schema := Catch(inner, "default")
	if got, err := schema.Parse(Missing); err != nil || got != "default" {
		t.Fatalf("Missing: %v, %v", got, err)
	}
	if got, err := schema.Parse(15); err != nil || got != "default" {
		t.Fatalf("number: %v, %v", got, err)
	}
	if got, err := schema.Parse("hi"); err != nil || got != "hi!" {
		t.Fatalf("string: %v, %v", got, err)
	}
}

func TestCatchOnOptional(t *testing.T) {
	schema := Catch(Optional(String()), "asdf")
	got, err := schema.Parse(Missing)
	if err != nil || !IsMissing(got) {
		t.Fatalf("optional catch keeps Missing: %v, %v", got, err)
	}
	got, err = schema.Parse(15)
	if err != nil || got != "asdf" {
		t.Fatalf("bad type uses catch: %v, %v", got, err)
	}
}

func TestChainedCatch(t *testing.T) {
	schema := Catch(Catch(String(), "inner"), "outer")
	got, err := schema.Parse(Missing)
	if err != nil || got != "inner" {
		t.Fatalf("inner catch wins: %v, %v", got, err)
	}
	got, err = schema.Parse(5)
	if err != nil || got != "inner" {
		t.Fatalf("inner catch on number: %v, %v", got, err)
	}
}

func TestCatchFuncReceivesIssues(t *testing.T) {
	var saw int
	schema := CatchFunc(String(), func(ctx CatchCtx) any {
		saw = len(ctx.Issues)
		return "fb"
	})
	got, err := schema.Parse(123)
	if err != nil || got != "fb" {
		t.Fatalf("got %v, %v", got, err)
	}
	if saw == 0 {
		t.Fatal("catch ctx should include issues")
	}
}

func TestCatchUnwrap(t *testing.T) {
	inner := String()
	if Catch(inner, "x").Unwrap() != inner {
		t.Fatal("Unwrap")
	}
}
