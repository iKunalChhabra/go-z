package zod

import "testing"

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/optional.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityOptionalBasic(t *testing.T) {
	// Ported from classic/tests/optional.test.ts — ".optional()"
	schema := Optional(String())
	got, err := schema.Parse("adsf")
	if err != nil || got != "adsf" {
		t.Fatalf("%v %v", got, err)
	}
	got, err = schema.Parse(Missing)
	if err != nil || !IsMissing(got) {
		t.Fatalf("Missing: %v %v", got, err)
	}
	if schema.SafeParse(nil).Success {
		t.Fatal("optional must reject null")
	}
	if schema.SafeParse(123).Success {
		t.Fatal("optional must reject number")
	}
}

func TestParityOptionalUnwrap(t *testing.T) {
	// Ported from classic/tests/optional.test.ts — "unwrap"
	inner := String()
	if Optional(inner).Unwrap() != inner {
		t.Fatal("Unwrap should return inner")
	}
}

func TestParityOptionalityFlags(t *testing.T) {
	// Ported from classic/tests/optional.test.ts — "optionality"
	a := String()
	if a.Internals().OptIn || a.Internals().OptOut {
		t.Fatal("string should not be optional")
	}
	b := Optional(String())
	if !b.Internals().OptIn || !b.Internals().OptOut {
		t.Fatal("optional should set OptIn and OptOut")
	}
	c := Default(String(), "asdf")
	if !c.Internals().OptIn {
		t.Fatal("default should set OptIn")
	}
	if c.Internals().OptOut {
		t.Fatal("default should not set OptOut")
	}
	d := Nullable(Optional(String()))
	if !d.Internals().OptIn || !d.Internals().OptOut {
		t.Fatal("optional().nullable() should keep optionality")
	}
	e := Nullable(Default(String(), "asdf"))
	if !e.Internals().OptIn {
		t.Fatal("default().nullable() should keep OptIn")
	}
	if e.Internals().OptOut {
		t.Fatal("default().nullable() should not set OptOut")
	}

	// z.undefined should NOT be optional — Undefined accepts only Missing;
	// Missing alone is not a schema.
}

func TestParityOptionalPipeOptionality(t *testing.T) {
	// Ported from classic/tests/optional.test.ts — "pipe optionality"
	a := Pipe(Optional(String()), String())
	if !a.Internals().OptIn {
		t.Fatal("optional.pipe(string) OptIn")
	}
	if a.Internals().OptOut {
		t.Fatal("optional.pipe(string) OptOut should be false")
	}

	b := Pipe(
		Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
			return v, nil
		}),
		Optional(String()),
	)
	if b.Internals().OptIn {
		t.Fatal("string.transform.pipe(optional) OptIn should be false")
	}
	if !b.Internals().OptOut {
		t.Fatal("string.transform.pipe(optional) OptOut")
	}

	c := Pipe(Default(String(), "asdf"), String())
	if !c.Internals().OptIn {
		t.Fatal("default.pipe(string) OptIn")
	}
	if c.Internals().OptOut {
		t.Fatal("default.pipe(string) OptOut should be false")
	}

	d := Pipe(
		Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
			return Missing, nil
		}),
		Default(String(), "asdf"),
	)
	if d.Internals().OptIn {
		t.Fatal("transform.pipe(default) OptIn should be false")
	}
	if d.Internals().OptOut {
		t.Fatal("transform.pipe(default) OptOut should be false")
	}
}

func TestParityOptionalPipeInsideObjects(t *testing.T) {
	// Ported from classic/tests/optional.test.ts — "pipe optionality inside objects"
	schema := Object(Shape{
		"a": Optional(String()),
		"b": Pipe(Optional(String()), String()),
		"c": Pipe(Default(String(), "asdf"), String()),
		"d": Pipe(
			Transform(String(), func(v any, _ *RefinementCtx) (any, error) { return v, nil }),
			Optional(String()),
		),
		"e": Pipe(
			Transform(String(), func(v any, _ *RefinementCtx) (any, error) { return v, nil }),
			Default(String(), "asdf"),
		),
	})
	// a optional out; b OptIn so can omit but out required → needs present or fails;
	// parse with all present
	got, err := schema.Parse(map[string]any{
		"a": "a",
		"b": "b",
		"c": "c",
		"d": "d",
		"e": "e",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "a" || got["b"] != "b" {
		t.Fatalf("%#v", got)
	}
	// omit a (optional out) and c (default fills)
	got, err = schema.Parse(map[string]any{
		"b": "b",
		"d": "d",
		"e": "e",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["a"]; ok {
		t.Fatalf("a should be omitted: %#v", got)
	}
	if got["c"] != "asdf" {
		t.Fatalf("c default via pipe: %#v", got)
	}
}

func TestParityOptionalPropWithPipe(t *testing.T) {
	// Ported from classic/tests/optional.test.ts — "optional prop with pipe"
	inner := Pipe(
		UnionOf(Number(), Nullish(String())),
		Transform(Any(), func(val any, _ *RefinementCtx) (any, error) {
			if val == nil || IsMissing(val) {
				return val, nil
			}
			switch x := val.(type) {
			case float64:
				return x, nil
			case string:
				f, ok := coerceToNumber(x)
				if !ok {
					return nil, nil
				}
				return f, nil
			default:
				return val, nil
			}
		}),
	)
	schema := Object(Shape{
		"id": Optional(Pipe(inner, Number())),
	})
	if _, err := schema.Parse(map[string]any{}); err != nil {
		t.Fatal(err)
	}
}

func TestParityOptionalUnionOptIn(t *testing.T) {
	// Ported from classic/tests/optional.test.ts — union optionality
	g := UnionOf(String(), Optional(String()))
	if !g.Internals().OptIn || !g.Internals().OptOut {
		t.Fatal("union with optional member should be optional")
	}
	h := UnionOf(String(), Number())
	if h.Internals().OptIn || h.Internals().OptOut {
		t.Fatal("plain union should not be optional")
	}
}

func TestParityOptionalInObjectOmit(t *testing.T) {
	schema := Object(Shape{
		"req": String(),
		"opt": Optional(String()),
	})
	got, err := schema.Parse(map[string]any{"req": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["opt"]; ok {
		t.Fatalf("opt omitted: %#v", got)
	}
	res := schema.SafeParse(map[string]any{})
	if res.Success {
		t.Fatal("req missing should fail")
	}
	found := false
	for _, iss := range res.Error.Issues {
		if len(iss.Path) > 0 && iss.Path[0] == "req" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want path req: %+v", res.Error.Issues)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/nullable.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityNullableBasic(t *testing.T) {
	// Ported from classic/tests/nullable.test.ts — ".nullable()"
	nullable := Nullable(String())
	got, err := nullable.Parse(nil)
	if err != nil || got != nil {
		t.Fatalf("%v %v", got, err)
	}
	got, err = nullable.Parse("asdf")
	if err != nil || got != "asdf" {
		t.Fatalf("%v %v", got, err)
	}
	if nullable.SafeParse(123).Success {
		t.Fatal("nullable string rejects number")
	}
	if nullable.SafeParse(Missing).Success {
		t.Fatal("nullable rejects Missing (use optional/nullish)")
	}
}

func TestParityNullableUnwrap(t *testing.T) {
	// Ported from classic/tests/nullable.test.ts — unwrap
	inner := String()
	schema := Nullable(inner)
	if schema.Unwrap() != inner {
		t.Fatal("unwrap")
	}
}

func TestParityNullSchema(t *testing.T) {
	// Ported from classic/tests/nullable.test.ts — "z.null"
	n := Null()
	got, err := n.Parse(nil)
	if err != nil || got != nil {
		t.Fatalf("%v %v", got, err)
	}
	if n.SafeParse("asdf").Success {
		t.Fatal("null rejects string")
	}
	if n.SafeParse(0).Success {
		t.Fatal("null rejects number")
	}
}

func TestParityNullish(t *testing.T) {
	schema := Nullish(String())
	if !schema.Internals().OptIn || !schema.Internals().OptOut {
		t.Fatal("nullish should be optional")
	}
	if _, err := schema.Parse(Missing); err != nil {
		t.Fatal(err)
	}
	got, err := schema.Parse(nil)
	if err != nil || got != nil {
		t.Fatalf("%v %v", got, err)
	}
	got, err = schema.Parse("x")
	if err != nil || got != "x" {
		t.Fatalf("%v %v", got, err)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/nonoptional.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityNonOptionalOnString(t *testing.T) {
	// Ported from classic/tests/nonoptional.test.ts — "nonoptional"
	schema := NonOptional(String())
	if _, err := schema.Parse("hello"); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse(Missing)
	if res.Success {
		t.Fatal("expected failure")
	}
	// String fails first with invalid_type expected string (Missing → undefined)
	if res.Error.Issues[0].Code != IssueInvalidType {
		t.Fatalf("%+v", res.Error.Issues[0])
	}
}

func TestParityNonOptionalWithOptional(t *testing.T) {
	// Ported from classic/tests/nonoptional.test.ts — "nonoptional with default"
	schema := NonOptional(Optional(String()))
	if _, err := schema.Parse("hi"); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse(Missing)
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidType || iss.Expected != "nonoptional" {
		t.Fatalf("want nonoptional, got %+v", iss)
	}
	if iss.Message != "Invalid input: expected nonoptional, received undefined" {
		t.Fatalf("msg=%q", iss.Message)
	}
}

func TestParityNonOptionalInObject(t *testing.T) {
	// Ported from classic/tests/nonoptional.test.ts — "nonoptional in object"
	schema := Object(Shape{
		"hi": NonOptional(Optional(String())),
	})
	r1 := schema.SafeParse(map[string]any{"hi": "asdf"})
	if !r1.Success {
		t.Fatalf("%v", r1.Error)
	}
	r2 := schema.SafeParse(map[string]any{"hi": Missing})
	if r2.Success {
		t.Fatal("Missing value should fail")
	}
	if r2.Error.Issues[0].Expected != "nonoptional" {
		t.Fatalf("%+v", r2.Error.Issues[0])
	}
	if len(r2.Error.Issues[0].Path) == 0 || r2.Error.Issues[0].Path[0] != "hi" {
		t.Fatalf("path: %+v", r2.Error.Issues[0].Path)
	}

	r3 := schema.SafeParse(map[string]any{})
	if r3.Success {
		t.Fatal("absent key should fail")
	}
	if r3.Error.Issues[0].Expected != "nonoptional" {
		t.Fatalf("%+v", r3.Error.Issues[0])
	}
}

func TestParityNonOptionalEncode(t *testing.T) {
	// Ported from classic/tests/nonoptional.test.ts — "encoding"
	schema := NonOptional(Optional(String()))
	got, err := Encode(schema, "hello")
	if err != nil || got != "hello" {
		t.Fatalf("encode hello: %v %v", got, err)
	}
	res := SafeEncode(schema, Missing)
	if res.Success {
		t.Fatal("encode Missing should fail")
	}
	if res.Error.Issues[0].Expected != "nonoptional" {
		t.Fatalf("%+v", res.Error.Issues[0])
	}
}

func TestParityReadonly(t *testing.T) {
	schema := Readonly(String())
	got, err := schema.Parse("x")
	if err != nil || got != "x" {
		t.Fatalf("%v %v", got, err)
	}
	if schema.Internals().OptIn {
		t.Fatal("readonly should inherit non-optional")
	}
	inner := Optional(String())
	ro := Readonly(inner)
	if !ro.Internals().OptIn || !ro.Internals().OptOut {
		t.Fatal("readonly should inherit optionality")
	}
}
