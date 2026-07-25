package z

import (
	"strings"
	"testing"
)

// The fluent wrappers are one-line delegations, which is exactly the shape of
// code where a copy-paste mistake — Optional() built from the wrong constructor,
// Nullable() returning the optional form — compiles and ships. These assert the
// observable behaviour of each one rather than its coverage.
func TestFluentWrapperMethodsOnPrimitives(t *testing.T) {
	// Optional accepts Missing and reports nil, not the zero value.
	if v, err := String().Optional().Parse(Missing); err != nil || v != nil {
		t.Errorf("String().Optional(): %#v %v", v, err)
	}
	if v, err := String().Optional().Parse("x"); err != nil || v == nil || *v != "x" {
		t.Errorf("String().Optional() value: %#v %v", v, err)
	}
	if String().Optional().SafeParse(nil).Success {
		t.Error("Optional must not accept null")
	}

	// Nullable accepts null but not Missing.
	if v, err := String().Nullable().Parse(nil); err != nil || v != nil {
		t.Errorf("String().Nullable(): %#v %v", v, err)
	}
	if String().Nullable().SafeParse(Missing).Success {
		t.Error("Nullable must not accept Missing")
	}

	// Nullish accepts both.
	for _, in := range []any{nil, Missing} {
		if v, err := String().Nullish().Parse(in); err != nil || v != nil {
			t.Errorf("String().Nullish()(%#v): %#v %v", in, v, err)
		}
	}

	// Default and Prefault produce a value; Catch replaces a failure.
	if v, err := String().Default("d").Parse(Missing); err != nil || v != "d" {
		t.Errorf("Default: %#v %v", v, err)
	}
	if v, err := String().Prefault("p").Parse(Missing); err != nil || v != "p" {
		t.Errorf("Prefault: %#v %v", v, err)
	}
	if v, err := String().Catch("c").Parse(42); err != nil || v != "c" {
		t.Errorf("Catch: %#v %v", v, err)
	}

	// NonOptional rejects an absent value.
	if String().Optional().NonOptional().SafeParse(Missing).Success {
		t.Error("NonOptional must reject Missing")
	}
	if v, err := String().Optional().NonOptional().Parse("x"); err != nil || v != "x" {
		t.Errorf("NonOptional value: %#v %v", v, err)
	}
}

// Every schema type carries the same wrapper surface. Anything missing here is a
// type whose users would have to fall back to the package-level constructors.
func TestFluentWrapperCoverageAcrossSchemaTypes(t *testing.T) {
	t.Run("number", func(t *testing.T) {
		if v, err := Number().Default(1.5).Parse(Missing); err != nil || v != 1.5 {
			t.Errorf("%#v %v", v, err)
		}
		if v, err := Int().Optional().Parse(7); err != nil || v == nil || *v != 7 {
			t.Errorf("%#v %v", v, err)
		}
		if v, err := Int64().Catch(9).Parse("x"); err != nil || v != 9 {
			t.Errorf("%#v %v", v, err)
		}
	})
	t.Run("bool", func(t *testing.T) {
		if v, err := Bool().Default(true).Parse(Missing); err != nil || v != true {
			t.Errorf("%#v %v", v, err)
		}
		if v, err := Bool().Nullable().Parse(nil); err != nil || v != nil {
			t.Errorf("%#v %v", v, err)
		}
	})
	t.Run("object", func(t *testing.T) {
		obj := Object(Shape{"a": String()})
		if v, err := obj.Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("%#v %v", v, err)
		}
		if v, err := obj.Default(map[string]any{"a": "d"}).Parse(Missing); err != nil || v["a"] != "d" {
			t.Errorf("%#v %v", v, err)
		}
	})
	t.Run("array", func(t *testing.T) {
		arr := Array(String())
		if v, err := arr.Default([]any{"x"}).Parse(Missing); err != nil || len(v) != 1 {
			t.Errorf("%#v %v", v, err)
		}
		if v, err := arr.Nullish().Parse(nil); err != nil || v != nil {
			t.Errorf("%#v %v", v, err)
		}
	})
	t.Run("record", func(t *testing.T) {
		rec := Record(String(), Number())
		if v, err := rec.Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("%#v %v", v, err)
		}
		if v, err := rec.Catch(map[string]any{"k": 1.0}).Parse(42); err != nil || v["k"] != 1.0 {
			t.Errorf("%#v %v", v, err)
		}
	})
	t.Run("map and set", func(t *testing.T) {
		if v, err := Map(String(), Number()).Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("map: %#v %v", v, err)
		}
		if v, err := Set(String()).Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("set: %#v %v", v, err)
		}
	})
	t.Run("tuple", func(t *testing.T) {
		tup := Tuple([]AnySchemaLike{String()})
		if v, err := tup.Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("%#v %v", v, err)
		}
		if v, err := tup.Default([]any{"d"}).Parse(Missing); err != nil || len(v) != 1 {
			t.Errorf("%#v %v", v, err)
		}
	})
	t.Run("time and bigint", func(t *testing.T) {
		if v, err := Time().Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("time: %#v %v", v, err)
		}
		if v, err := BigInt().Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("bigint: %#v %v", v, err)
		}
	})
	t.Run("enum and literal", func(t *testing.T) {
		if v, err := Enum("a", "b").Default("a").Parse(Missing); err != nil || v != "a" {
			t.Errorf("enum: %#v %v", v, err)
		}
		if v, err := Literal("x").Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("literal: %#v %v", v, err)
		}
	})
	t.Run("union and lazy", func(t *testing.T) {
		u := UnionOf(String(), Number())
		if v, err := u.Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("union: %#v %v", v, err)
		}
		lazy := Lazy(func() AnySchemaLike { return String() })
		if v, err := lazy.Optional().Parse(Missing); err != nil || v != nil {
			t.Errorf("lazy: %#v %v", v, err)
		}
	})
}

// Wrappers compose, and each rung keeps the behaviour of the one below.
func TestFluentWrappersCompose(t *testing.T) {
	schema := String().Min(2).Optional().Default("fallback")
	if v, err := schema.Parse(Missing); err != nil || v != "fallback" {
		t.Errorf("Default over Optional: %#v %v", v, err)
	}
	if v, err := schema.Parse("ok"); err != nil || v != "ok" {
		t.Errorf("value passes through: %#v %v", v, err)
	}
	if schema.SafeParse("x").Success {
		t.Error("the inner Min still applies")
	}

	// A wrapper without a given sugar method is not a missing capability: the
	// generic helpers work on any schema — as long as Optional stays outermost,
	// because the *T edge only exists at Parse.
	refined := OptionalOf(RefineOf[string](String(), func(s string) bool {
		return len(s) > 1
	}, "too short"))
	if v, err := refined.Parse(Missing); err != nil || v != nil {
		t.Errorf("absent value passes the refinement: %#v %v", v, err)
	}
	if v, err := refined.Parse("ok"); err != nil || v == nil || *v != "ok" {
		t.Errorf("value passes: %#v %v", v, err)
	}
	if res := refined.SafeParse("x"); res.Success || res.Error.Issues[0].Message != "too short" {
		t.Errorf("refinement applies: %+v", res)
	}

	if v, err := ReadonlyOf(RefineOf[string](String(), func(string) bool { return true })).Parse("x"); err != nil || v != "x" {
		t.Errorf("ReadonlyOf over a checked schema: %#v %v", v, err)
	}
}

// Instantiating a generic wrapper over an Optional is a type error the compiler
// cannot catch — OptionalSchema[T] satisfies Schema[*T] — so the failure has to
// explain itself rather than reading as an internal bug.
func TestGenericWrapperOverOptionalExplainsItself(t *testing.T) {
	_, err := ReadonlyOf[*string](String().Optional()).Parse("x")
	if err == nil {
		t.Fatal("expected the mismatch to be reported")
	}
	msg := err.Error()
	for _, want := range []string{"Optional", "Apply Optional or Nullable last"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should mention %q: %s", want, msg)
		}
	}
	if strings.Contains(msg, "please report") {
		t.Error("this is a usage error, not a bug to report")
	}
}
