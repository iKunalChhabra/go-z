package zod

import "testing"

// Ported from v4/classic/tests/default.test.ts (subset; no object nesting).

func TestDefaultBasic(t *testing.T) {
	schema := Default(String(), "default")
	got, err := schema.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("Parse(Missing) = %v, %v", got, err)
	}
	if got, err := schema.Parse("hi"); err != nil || got != "hi" {
		t.Fatalf("Parse(string) = %v, %v", got, err)
	}
}

func TestDefaultWithOptional(t *testing.T) {
	schema := Default(Optional(String()), "default")
	got, err := schema.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("got %v, %v", got, err)
	}
	unwrapped := schema.Unwrap()
	u, err := unwrapped.(*OptionalSchema).Parse(Missing)
	if err != nil || !IsMissing(u) {
		t.Fatalf("unwrap parse Missing: %v, %v", u, err)
	}
}

func TestDefaultWithTransform(t *testing.T) {
	inner := Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
		return v.(string) + "!", nil
	})
	schema := Default(inner, "default")
	got, err := schema.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("default short-circuits without transform: %v, %v", got, err)
	}
	got, err = schema.Parse("hi")
	if err != nil || got != "hi!" {
		t.Fatalf("present value transforms: %v, %v", got, err)
	}
}

func TestDefaultFunc(t *testing.T) {
	n := 0
	schema := DefaultFunc(String(), func() any {
		n++
		return "x"
	})
	schema.MustParse(Missing)
	schema.MustParse(Missing)
	if n != 2 {
		t.Fatalf("default func called %d times, want 2", n)
	}
}

func TestDefaultOnOutputUndefined(t *testing.T) {
	// transform that yields Missing → default applied after inner run
	inner := Transform(String(), func(any, *RefinementCtx) (any, error) {
		return Missing, nil
	})
	schema := Default(inner, "asdf")
	got, err := schema.Parse("")
	if err != nil || got != "asdf" {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestChainedDefaults(t *testing.T) {
	schema := Default(Default(String(), "inner"), "outer")
	got, err := schema.Parse(Missing)
	if err != nil || got != "outer" {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestPrefaultParsesDefault(t *testing.T) {
	// prefault value is parsed through inner (and subsequent refine)
	inner := Refine(String(), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'c'
	}, "must start with c")
	schema := Prefault(inner, "c")
	got, err := schema.Parse(Missing)
	if err != nil || got != "c" {
		t.Fatalf("prefault ok: %v, %v", got, err)
	}

	bad := Prefault(inner, "z")
	if bad.SafeParse(Missing).Success {
		t.Fatal("bad prefault should fail refine")
	}
}

func TestDefaultSkipsInnerValidation(t *testing.T) {
	// Zod: default is NOT re-parsed. A default that would fail the inner
	// schema still succeeds when input is Missing.
	inner := Refine(String(), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'a'
	})
	schema := Default(inner, "z")
	got, err := schema.Parse(Missing)
	if err != nil || got != "z" {
		t.Fatalf("default bypasses refine: %v, %v", got, err)
	}
}

func TestOptionalOnDefault(t *testing.T) {
	schema := Optional(Default(String(), "asdf"))
	got, err := schema.Parse(Missing)
	// Optional with OptIn inner (Default): runs default, which replaces Missing.
	if err != nil || got != "asdf" {
		t.Fatalf("got %v, %v", got, err)
	}
}
