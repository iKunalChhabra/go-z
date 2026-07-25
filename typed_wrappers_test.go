package zod

import (
	"testing"
	"time"
)

// The typed edge must survive wrappers: these assignments only compile if
// Parse returns the concrete type rather than any.

func TestTypedOptionalKeepsInnerType(t *testing.T) {
	got, err := String().Min(2).Optional().Parse("abc")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != "abc" {
		t.Fatalf("got %v", got)
	}
	var s string = *got
	_ = s

	absent, err := String().Optional().Parse(Missing)
	if err != nil || absent != nil {
		t.Fatalf("absent = %v, %v", absent, err)
	}
	if String().Optional().SafeParse(nil).Success {
		t.Fatal("optional must still reject null")
	}
}

func TestTypedNullableAndNullish(t *testing.T) {
	n, err := Number().Nullable().Parse(nil)
	if err != nil || n != nil {
		t.Fatalf("null = %v, %v", n, err)
	}
	n, err = Number().Nullable().Parse(1.5)
	if err != nil || n == nil || *n != 1.5 {
		t.Fatalf("value = %v, %v", n, err)
	}

	// Nullish stays *T instead of collapsing into **T.
	both := Bool().Nullish()
	for _, in := range []any{nil, Missing} {
		got, err := both.Parse(in)
		if err != nil || got != nil {
			t.Fatalf("nullish(%v) = %v, %v", in, got, err)
		}
	}
	got, err := both.Parse(true)
	if err != nil || got == nil || *got != true {
		t.Fatalf("nullish(true) = %v, %v", got, err)
	}
}

func TestTypedDefaultPrefaultCatch(t *testing.T) {
	if got, err := String().Default("d").Parse(Missing); err != nil || got != "d" {
		t.Fatalf("default = %q, %v", got, err)
	}
	if got, err := Number().Prefault(3.0).Parse(Missing); err != nil || got != 3.0 {
		t.Fatalf("prefault = %v, %v", got, err)
	}
	if got, err := String().Catch("fallback").Parse(123); err != nil || got != "fallback" {
		t.Fatalf("catch = %q, %v", got, err)
	}

	stamp := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if got, err := Time().Default(stamp).Parse(Missing); err != nil || !got.Equal(stamp) {
		t.Fatalf("time default = %v, %v", got, err)
	}

	obj := Object(Shape{"a": String()}).Default(map[string]any{"a": "x"})
	m, err := obj.Parse(Missing)
	if err != nil || m["a"] != "x" {
		t.Fatalf("object default = %#v, %v", m, err)
	}
}

func TestTypedNonOptionalAndReadonly(t *testing.T) {
	if got, err := String().Optional().NonOptional().Parse("hi"); err != nil || got != "hi" {
		t.Fatalf("nonoptional = %q, %v", got, err)
	}
	if NonOptionalOf(String().Optional()).SafeParse(Missing).Success {
		t.Fatal("nonoptional must reject Missing")
	}
	if got, err := ReadonlyOf(String()).Parse("ro"); err != nil || got != "ro" {
		t.Fatalf("readonly = %q, %v", got, err)
	}
}

func TestTypedWrapperChain(t *testing.T) {
	// Default over Optional keeps the string edge through two wrappers.
	schema := DefaultOf[string](String().Optional().NonOptional(), "fallback")
	got, err := schema.Parse(Missing)
	if err != nil || got != "fallback" {
		t.Fatalf("chain = %q, %v", got, err)
	}
}

func TestErasedWrappersStillReturnAny(t *testing.T) {
	// The type-erased constructors keep working for heterogeneous containers.
	shape := Shape{"a": Optional(String()), "b": Default(Number(), 1.0)}
	out, err := Object(shape).Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := out["a"]; present {
		t.Fatalf("absent optional key should stay absent: %#v", out)
	}
	if out["b"] != 1.0 {
		t.Fatalf("default = %#v", out["b"])
	}

	raw, err := Optional(String()).ParseAny(Missing)
	if err != nil || !IsMissing(raw) {
		t.Fatalf("erased ParseAny = %#v, %v", raw, err)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Generic helpers work for every schema type, including ones with no fluent
// method of their own.
//////////////////////////////////////////////////////////////////////////////

func TestGenericHelpersOnContainers(t *testing.T) {
	nonEmpty := RefineOf(Set(String()), func(v []any) bool { return len(v) > 0 }, "set must not be empty")
	if _, err := nonEmpty.Parse([]any{"a"}); err != nil {
		t.Fatal(err)
	}
	res := nonEmpty.SafeParse([]any{})
	if res.Success || res.Error.Issues[0].Message != "set must not be empty" {
		t.Fatalf("%+v", res.Error)
	}

	rec := RefineOf(Record(String(), Number()), func(m map[string]any) bool { return len(m) < 3 })
	if rec.SafeParse(map[string]any{"a": 1.0, "b": 2.0, "c": 3.0}).Success {
		t.Fatal("record refine should fail")
	}

	pair := RefineOf(Tuple([]AnySchemaLike{String(), Number()}), func(v []any) bool {
		return v[1].(float64) > 0
	})
	if pair.SafeParse([]any{"a", -1.0}).Success {
		t.Fatal("tuple refine should fail")
	}

	counted := 0
	sup := SuperRefineOf(Map(String(), Number()), func(m map[any]any, ctx *RefinementCtx) {
		counted = len(m)
		if len(m) == 0 {
			ctx.AddIssue(Issue{Code: IssueCustom, Message: "empty map"})
		}
	})
	if sup.SafeParse(map[any]any{}).Success || counted != 0 {
		t.Fatal("map super-refine should fail on empty input")
	}

	upper := OverwriteOf(String(), func(s string) string { return s + "!" })
	if got, err := upper.Parse("hi"); err != nil || got != "hi!" {
		t.Fatalf("overwrite = %q, %v", got, err)
	}
}

func TestFluentCoverageOnContainers(t *testing.T) {
	// Record / Map / Set / Tuple now carry the same fluent surface.
	if got, err := Record(String(), Number()).Default(map[string]any{"a": 1.0}).Parse(Missing); err != nil || got["a"] != 1.0 {
		t.Fatalf("record default = %#v, %v", got, err)
	}
	if got, err := Set(String()).Optional().Parse(Missing); err != nil || got != nil {
		t.Fatalf("set optional = %v, %v", got, err)
	}
	if got, err := Tuple([]AnySchemaLike{String()}).Catch([]any{"x"}).Parse(123); err != nil || got[0] != "x" {
		t.Fatalf("tuple catch = %#v, %v", got, err)
	}
	if got, err := Map(String(), Number()).Nullable().Parse(nil); err != nil || got != nil {
		t.Fatalf("map nullable = %v, %v", got, err)
	}

	only2 := Set(String()).Refine(func(v []any) bool { return len(v) == 2 }, "need two")
	if only2.SafeParse([]any{"a"}).Success {
		t.Fatal("set fluent refine should fail")
	}
	// Fluent refine keeps the concrete type, so chaining continues.
	if _, err := only2.Min(1).Parse([]any{"a", "b"}); err != nil {
		t.Fatal(err)
	}
}

func TestUnwrapperCoversWrappers(t *testing.T) {
	inner := String()
	wrappers := []AnySchemaLike{
		OptionalOf(inner),
		NullableOf(inner),
		DefaultOf(inner, "d"),
		PrefaultOf(inner, "p"),
		CatchOf(inner, "c"),
		NonOptionalOf(inner),
		ReadonlyOf(inner),
		CheckOf(inner),
	}
	for _, w := range wrappers {
		u, ok := w.(Unwrapper)
		if !ok {
			t.Fatalf("%T does not implement Unwrapper", w)
		}
		if u.Unwrap() != AnySchemaLike(inner) {
			t.Fatalf("%T unwrapped to %T", w, u.Unwrap())
		}
	}
}
