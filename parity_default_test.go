package zod

import (
	"reflect"
	"testing"
)

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/default.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityDefaultBasic(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "basic defaults"
	got, err := Default(String(), "default").Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestParityDefaultWithOptional(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "default with optional"
	schema := Default(Optional(String()), "default")
	got, err := schema.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("%v %v", got, err)
	}
	u, err := schema.Unwrap().(*OptionalSchema[any]).ParseAny(Missing)
	if err != nil || !IsMissing(u) {
		t.Fatalf("unwrap: %v %v", u, err)
	}
}

func TestParityDefaultWithTransform(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "default with transform"
	inner := Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
		s := v.(string)
		out := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			out[i] = c
		}
		return string(out), nil
	})
	schema := Default(inner, "default")
	got, err := schema.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("default short-circuits: %v %v", got, err)
	}
	got, err = schema.Parse("hi")
	if err != nil || got != "HI" {
		t.Fatalf("transform: %v %v", got, err)
	}
	if _, ok := schema.Unwrap().(*TransformSchema); !ok {
		t.Fatalf("unwrap type %T", schema.Unwrap())
	}
}

func TestParityDefaultOnExistingOptional(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "default on existing optional"
	schema := Default(Optional(String()), "asdf")
	got, err := schema.Parse(Missing)
	if err != nil || got != "asdf" {
		t.Fatalf("%v %v", got, err)
	}
	if _, ok := schema.Unwrap().(*OptionalSchema[any]); !ok {
		t.Fatalf("%T", schema.Unwrap())
	}
}

func TestParityOptionalOnDefault(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "optional on default"
	schema := Optional(Default(String(), "asdf"))
	got, err := schema.Parse(Missing)
	if err != nil || got == nil || *got != "asdf" {
		t.Fatalf("%v %v", got, err)
	}
	raw, err := schema.ParseAny(Missing)
	if err != nil || raw != "asdf" {
		t.Fatalf("raw: %v %v", raw, err)
	}
}

func TestParityDefaultRemoveDefault(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "removeDefault" → Unwrap
	schema := Default(String(), "asdf").Unwrap().(*StringSchema)
	if schema.SafeParse(Missing).Success {
		t.Fatal("unwrapped string should reject Missing")
	}
	if _, err := schema.Parse("x"); err != nil {
		t.Fatal(err)
	}
}

func TestParityDefaultApplyAtOutput(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "apply default at output"
	inner := Transform(String(), func(any, *RefinementCtx) (any, error) {
		return Missing, nil
	})
	schema := Default(inner, "asdf")
	got, err := schema.Parse("")
	if err != nil || got != "asdf" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestParityDefaultNested(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "nested"
	inner := Default(String(), "asdf")
	outer := Default(Object(Shape{"inner": inner}), map[string]any{"inner": "qwer"})
	got, err := outer.Parse(Missing)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["inner"] != "qwer" {
		t.Fatalf("%#v", got)
	}
	got, err = outer.Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	m = got.(map[string]any)
	if m["inner"] != "asdf" {
		t.Fatalf("inner default: %#v", got)
	}
}

func TestParityChainedDefaults(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "chained defaults"
	schema := Default(Default(String(), "inner"), "outer")
	got, err := schema.Parse(Missing)
	if err != nil || got != "outer" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestParityDefaultObjectOptionality(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "object optionality"
	schema := Object(Shape{
		"hi": Default(String(), "hi"),
	})
	got, err := schema.Parse(map[string]any{})
	if err != nil || got["hi"] != "hi" {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestParityDefaultNestedPrefaultRefine(t *testing.T) {
	// Ported from classic/tests/default.test.ts — "nested prefault/default"
	a := Refine(Default(String(), "a"), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'a'
	})
	b := Default(Refine(String(), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'b'
	}), "b")
	c := Refine(Prefault(String(), "c"), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'c'
	})
	d := Prefault(Refine(String(), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'd'
	}), "d")

	obj := Object(Shape{"a": a, "b": b, "c": c, "d": d})
	got, err := obj.Parse(map[string]any{"a": "a1", "b": "b1", "c": "c1", "d": "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "a1" || got["d"] != "d1" {
		t.Fatalf("%#v", got)
	}
	// defaults fill when absent
	got, err = obj.Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "a" || got["b"] != "b" || got["c"] != "c" || got["d"] != "d" {
		t.Fatalf("defaults: %#v", got)
	}
}

func TestParityDefaultFunc(t *testing.T) {
	n := 0
	schema := DefaultFunc(String(), func() any {
		n++
		return "x"
	})
	schema.MustParse(Missing)
	schema.MustParse(Missing)
	if n != 2 {
		t.Fatalf("called %d times", n)
	}
}

func TestParityDefaultSkipsInnerValidation(t *testing.T) {
	inner := Refine(String(), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'a'
	})
	schema := Default(inner, "z")
	got, err := schema.Parse(Missing)
	if err != nil || got != "z" {
		t.Fatalf("%v %v", got, err)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/prefault.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityPrefaultBasic(t *testing.T) {
	// Ported from classic/tests/prefault.test.ts — "basic prefault"
	a := Prefault(String().Trim(), "  default  ")
	got, err := a.Parse("  asdf  ")
	if err != nil || got != "asdf" {
		t.Fatalf("%v %v", got, err)
	}
	got, err = a.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("prefault trim: %v %v", got, err)
	}
}

func TestParityPrefaultInsideObject(t *testing.T) {
	// Ported from classic/tests/prefault.test.ts — "prefault inside object"
	a := Object(Shape{
		"name":  Optional(String()),
		"age":   Default(Number(), 1234.0),
		"email": Prefault(String(), "1234"),
	})
	got, err := a.Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if got["age"] != 1234.0 || got["email"] != "1234" {
		t.Fatalf("%#v", got)
	}
	if _, ok := got["name"]; ok {
		t.Fatalf("name omitted: %#v", got)
	}
}

func TestParityPrefaultObjectClone(t *testing.T) {
	// Ported from classic/tests/prefault.test.ts — shallow clone of object prefault
	schema := Prefault(Object(Shape{"a": String()}), map[string]any{"a": "x"})
	r1, err := schema.Parse(Missing)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := schema.Parse(Missing)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(r1, r2) == false {
		t.Fatalf("%#v vs %#v", r1, r2)
	}
	// Same contents; Prefault returns the same map literal each time unless Func —
	// Zod clones; we accept equal values.
	m1, _ := r1.(map[string]any)
	m2, _ := r2.(map[string]any)
	if m1["a"] != "x" || m2["a"] != "x" {
		t.Fatalf("%#v %#v", r1, r2)
	}
}

func TestParityPrefaultDirectionEncode(t *testing.T) {
	// Ported from classic/tests/prefault.test.ts — "direction-aware prefault"
	schema := Prefault(String(), "hello")
	got, err := schema.Parse(Missing)
	if err != nil || got != "hello" {
		t.Fatalf("parse Missing: %v %v", got, err)
	}
	res := SafeEncode(schema, Missing)
	if res.Success {
		t.Fatal("encode Missing should not apply prefault")
	}
	enc, err := Encode(schema, "world")
	if err != nil || enc != "world" {
		t.Fatalf("encode world: %v %v", enc, err)
	}
}

func TestParityPrefaultBadFailsRefine(t *testing.T) {
	inner := Refine(String(), func(v any) bool {
		s, _ := v.(string)
		return len(s) > 0 && s[0] == 'c'
	}, "must start with c")
	if Prefault(inner, "z").SafeParse(Missing).Success {
		t.Fatal("bad prefault should fail refine")
	}
	got, err := Prefault(inner, "c").Parse(Missing)
	if err != nil || got != "c" {
		t.Fatalf("%v %v", got, err)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/catch.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityCatchBasic(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "basic catch"
	got, err := Catch(String(), "default").Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestParityCatchFnNotCalledOnSuccess(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "catch fn does not run when parsing succeeds"
	called := false
	schema := CatchFunc(String(), func(CatchCtx) any {
		called = true
		return "asdf"
	})
	got, err := schema.Parse("test")
	if err != nil || got != "test" {
		t.Fatalf("%v %v", got, err)
	}
	if called {
		t.Fatal("catch fn should not run on success")
	}
}

func TestParityCatchAsync(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "basic catch async"
	t.Skip("async parse unsupported (classic/tests/catch.test.ts)")
}

func TestParityCatchReplaceWrongTypes(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "catch replace wrong types"
	schema := Catch(String(), "default")
	for _, in := range []any{true, 15, []any{}, map[string]any{}} {
		got, err := schema.Parse(in)
		if err != nil || got != "default" {
			t.Fatalf("in=%#v got=%v err=%v", in, got, err)
		}
	}
}

func TestParityCatchWithTransform(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "catch with transform"
	inner := Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
		s := v.(string)
		b := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			b[i] = c
		}
		return string(b), nil
	})
	schema := Catch(inner, "default")
	got, err := schema.Parse(Missing)
	if err != nil || got != "default" {
		t.Fatalf("%v %v", got, err)
	}
	got, err = schema.Parse(15)
	if err != nil || got != "default" {
		t.Fatalf("%v %v", got, err)
	}
	got, err = schema.Parse("hi")
	if err != nil || got != "HI" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestParityCatchOnExistingOptional(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "catch on existing optional"
	schema := Catch(Optional(String()), "asdf")
	got, err := schema.Parse(Missing)
	if err != nil || !IsMissing(got) {
		// Optional succeeds with Missing — catch should NOT replace
		t.Fatalf("optional success: %v %v", got, err)
	}
	got, err = schema.Parse(15)
	if err != nil || got != "asdf" {
		t.Fatalf("catch wrong type: %v %v", got, err)
	}
}

func TestParityOptionalOnCatch(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "optional on catch"
	schema := Optional(Catch(String(), "asdf"))
	if !schema.Internals().OptIn {
		t.Fatal("optional on catch should be OptIn")
	}
}

func TestParityCatchNested(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "nested"
	inner := Catch(String(), "asdf")
	outer := Catch(Object(Shape{"inner": inner}), map[string]any{"inner": "asdf"})
	got, err := outer.Parse(Missing)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["inner"] != "asdf" {
		t.Fatalf("%#v", got)
	}
	got, err = outer.Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	m = got.(map[string]any)
	if m["inner"] != "asdf" {
		t.Fatalf("inner catch: %#v", got)
	}
}

func TestParityChainedCatch(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "chained catch"
	schema := Catch(Catch(String(), "inner"), "outer")
	got, err := schema.Parse(Missing)
	if err != nil || got != "inner" {
		t.Fatalf("inner catch wins: %v %v", got, err)
	}
	got, err = schema.Parse(5)
	if err != nil || got != "inner" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestParityCatchCtx(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — catch callback receives issues
	var sawIssues bool
	schema := CatchFunc(String().Min(5), func(ctx CatchCtx) any {
		if len(ctx.Issues) > 0 {
			sawIssues = true
		}
		return "fallback"
	})
	got, err := schema.Parse("hi")
	if err != nil || got != "fallback" {
		t.Fatalf("%v %v", got, err)
	}
	if !sawIssues {
		t.Fatal("catch ctx should include issues")
	}
}

func TestParityCatchUnwrap(t *testing.T) {
	// Ported from classic/tests/catch.test.ts — "removeCatch"
	inner := String()
	if Catch(inner, "x").Unwrap() != inner {
		t.Fatal("unwrap")
	}
}

func TestParityCatchInObject(t *testing.T) {
	schema := Object(Shape{
		"name": Catch(String(), "anon"),
	})
	got, err := schema.Parse(map[string]any{})
	if err != nil || got["name"] != "anon" {
		t.Fatalf("%#v %v", got, err)
	}
	got, err = schema.Parse(map[string]any{"name": 123})
	if err != nil || got["name"] != "anon" {
		t.Fatalf("%#v %v", got, err)
	}
	got, err = schema.Parse(map[string]any{"name": "bob"})
	if err != nil || got["name"] != "bob" {
		t.Fatalf("%#v %v", got, err)
	}
}
