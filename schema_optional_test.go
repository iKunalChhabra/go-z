package zod

import "testing"

// Ported from v4/classic/tests/optional.test.ts and nullable.test.ts (subset).

func TestOptionalBasic(t *testing.T) {
	schema := OptionalOf(String())
	if got, err := schema.Parse("adsf"); err != nil || got == nil || *got != "adsf" {
		t.Fatalf("Parse(string) = %v, %v", got, err)
	}
	got, err := schema.Parse(Missing)
	if err != nil {
		t.Fatalf("Parse(Missing) error: %v", err)
	}
	if got != nil {
		t.Fatalf("Parse(Missing) = %#v, want nil", got)
	}
	raw, err := schema.ParseAny(Missing)
	if err != nil || !IsMissing(raw) {
		t.Fatalf("ParseAny(Missing) = %#v, %v", raw, err)
	}
	if schema.SafeParse(nil).Success {
		t.Fatal("optional must reject null")
	}
}

func TestOptionalUnwrap(t *testing.T) {
	inner := String()
	if Optional(inner).Unwrap() != inner {
		t.Fatal("Unwrap should return inner")
	}
}

func TestOptionalityFlags(t *testing.T) {
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
}

func TestPipeOptionality(t *testing.T) {
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
}

func TestNullableBasic(t *testing.T) {
	schema := NullableOf(String())
	if got, err := schema.Parse(nil); err != nil || got != nil {
		t.Fatalf("Parse(nil) = %v, %v", got, err)
	}
	if got, err := schema.Parse("asdf"); err != nil || got == nil || *got != "asdf" {
		t.Fatalf("Parse(string) = %v, %v", got, err)
	}
	if schema.SafeParse(123).Success {
		t.Fatal("nullable string must reject number")
	}
}

func TestNullish(t *testing.T) {
	schema := NullishOf(String())
	if !schema.Internals().OptIn || !schema.Internals().OptOut {
		t.Fatal("nullish should be optional")
	}
	if got, err := schema.Parse(Missing); err != nil || got != nil {
		t.Fatalf("nullish Missing: %v, %v", got, err)
	}
	if got, err := schema.Parse(nil); err != nil || got != nil {
		t.Fatalf("nullish nil: %v, %v", got, err)
	}
	if got, err := schema.Parse("x"); err != nil || got == nil || *got != "x" {
		t.Fatalf("nullish string: %v, %v", got, err)
	}
}

func TestNonOptional(t *testing.T) {
	schema := NonOptional(Optional(String()))
	if _, err := schema.Parse("hi"); err != nil {
		t.Fatalf("Parse(string): %v", err)
	}
	res := schema.SafeParse(Missing)
	if res.Success {
		t.Fatal("nonoptional must reject Missing")
	}
	if res.Error.Issues[0].Code != IssueInvalidType || res.Error.Issues[0].Expected != "nonoptional" {
		t.Fatalf("want invalid_type nonoptional, got %+v", res.Error.Issues[0])
	}
}

func TestReadonlyNoop(t *testing.T) {
	schema := Readonly(String())
	if got, err := schema.Parse("x"); err != nil || got != "x" {
		t.Fatalf("readonly: %v, %v", got, err)
	}
	if schema.Internals().OptIn {
		t.Fatal("readonly should inherit non-optional")
	}
}
