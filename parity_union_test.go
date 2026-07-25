package zod

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Parity ports from:
//   packages/zod/src/v4/classic/tests/union.test.ts
//   packages/zod/src/v4/classic/tests/discriminated-unions.test.ts

func TestParityUnionFunctionParsing(t *testing.T) {
	// Port: "function parsing"
	schema := Union([]AnySchemaLike{
		Refine(String(), func(any) bool { return false }, "no"),
		Refine(Number(), func(any) bool { return false }, "no"),
	})
	if schema.SafeParse("asdf").Success {
		t.Fatal("expected failure")
	}
}

func TestParityUnion2(t *testing.T) {
	// Port: "union 2"
	schema := Union([]AnySchemaLike{
		Number(),
		Refine(String(), func(any) bool { return false }, "no"),
	})
	if schema.SafeParse("a").Success {
		t.Fatal("expected failure")
	}
}

func TestParityUnionReturnsValidOverInvalid(t *testing.T) {
	// Port: "return valid over invalid"
	schema := Union([]AnySchemaLike{
		Object(Shape{"email": String().Email()}),
		String(),
	})
	cases := []struct {
		in   any
		want any
	}{
		{"asdf", "asdf"},
		{map[string]any{"email": "asdlkjf@lkajsdf.com"}, map[string]any{"email": "asdlkjf@lkajsdf.com"}},
	}
	for _, tc := range cases {
		got, err := schema.Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%v): %v", tc.in, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("Parse(%v) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

func TestParityUnionErrorsFromBothArms(t *testing.T) {
	// Port: "return errors from both union arms"
	res := Union([]AnySchemaLike{Number(), Bool()}).SafeParse("a")
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion || iss.Message != "Invalid input" {
		t.Fatalf("got %+v", iss)
	}
	if len(iss.Errors) != 2 {
		t.Fatalf("want 2 arms, got %d", len(iss.Errors))
	}
	if iss.Errors[0][0].Expected != "number" || iss.Errors[1][0].Expected != "boolean" {
		t.Fatalf("arms = %+v", iss.Errors)
	}
}

func TestParityUnionOptionsGetter(t *testing.T) {
	// Port: "options getter"
	u := Union([]AnySchemaLike{String(), Number()})
	if len(u.Options) != 2 {
		t.Fatalf("len = %d", len(u.Options))
	}
	if _, err := u.Options[0].(*StringSchema).Parse("asdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Options[1].(*NumberSchema).Parse(1234.0); err != nil {
		t.Fatal(err)
	}
}

func TestParityUnionReadonly(t *testing.T) {
	// Port: "readonly union" (Go: fixed options slice)
	options := []AnySchemaLike{String(), Number()}
	u := Union(options)
	if _, err := u.Parse("asdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Parse(12.0); err != nil {
		t.Fatal(err)
	}
}

func TestParityUnionValues(t *testing.T) {
	// Port: "union values"
	schema := Union([]AnySchemaLike{Literal("a"), Literal("b"), Literal("c")})
	vals := schema.Internals().Values
	for _, v := range []any{"a", "b", "c"} {
		if _, ok := vals[v]; !ok {
			t.Fatalf("missing %v in %v", v, vals)
		}
	}
}

func TestParityUnionNonAbortedErrors(t *testing.T) {
	// Port: "non-aborted errors"
	arm0 := Object(Shape{
		"date":      Number(),
		"startDate": Optional(Nil()),
		"endDate":   Optional(Nil()),
	})
	arm1 := Refine(
		Object(Shape{
			"date":      Optional(Nil()),
			"startDate": Number(),
			"endDate":   Number(),
		}),
		func(v any) bool {
			m := v.(map[string]any)
			return m["startDate"] != m["endDate"]
		},
		RefineOpts{
			Error: MessageFromString("startDate and endDate must be different"),
			Path:  []any{"endDate"},
		},
	)
	schema := Union([]AnySchemaLike{arm0, arm1})
	res := schema.SafeParse(map[string]any{
		"date": nil, "startDate": 1.0, "endDate": 1.0,
	})
	if res.Success || len(res.Error.Issues) != 1 {
		t.Fatalf("got %+v", res.Error)
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueCustom || iss.Message != "startDate and endDate must be different" {
		t.Fatalf("got %+v", iss)
	}
	if !reflect.DeepEqual(iss.Path, []any{"endDate"}) {
		t.Fatalf("path = %v", iss.Path)
	}
}

func TestParityUnionSurfaceContinuableErrors(t *testing.T) {
	// Port: "surface continuable errors only if they exist"
	schema := Union([]AnySchemaLike{Bool(), String().UUID(), String().JWT()})
	res := schema.SafeParse("asdf")
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion || len(iss.Errors) != 3 {
		t.Fatalf("got %+v", iss)
	}
	if iss.Errors[1][0].Format != "uuid" || iss.Errors[2][0].Format != "jwt" {
		t.Fatalf("formats = %+v / %+v", iss.Errors[1][0], iss.Errors[2][0])
	}
}

func TestParityXor(t *testing.T) {
	// Port: "z.xor()" — exactly one option must succeed.
	schema := XorOf(String().Min(1), Number().Gte(0))
	if got, err := schema.Parse("hi"); err != nil || got != "hi" {
		t.Fatalf(`Parse("hi") = %v, %v`, got, err)
	}
	if got, err := schema.Parse(5.0); err != nil || got != 5.0 {
		t.Fatalf("Parse(5) = %v, %v", got, err)
	}
	if schema.SafeParse("").Success {
		t.Fatal(`"" should fail (zero matches)`)
	}
	if schema.SafeParse(-1.0).Success {
		t.Fatal("-1 should fail (zero matches)")
	}

	// Overlapping options: both String() and String().Min(1) accept "ab".
	overlap := XorOf(String(), String().Min(1))
	res := overlap.SafeParse("ab")
	if res.Success {
		t.Fatal("overlapping success should fail exclusive xor")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion {
		t.Fatalf("code = %s", iss.Code)
	}
	if len(iss.Errors) != 0 {
		t.Fatalf("multi-match Errors should be empty, got %+v", iss.Errors)
	}
	if iss.Inclusive {
		t.Fatal("multi-match Inclusive should be false")
	}
}

func TestParityUnionEmpty(t *testing.T) {
	// Port: "z.union([]) constructs and rejects all input"
	res := Union(nil).SafeParse("anything")
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion || len(iss.Errors) != 0 {
		t.Fatalf("got %+v", iss)
	}
}

func TestParityUnionTableDriven(t *testing.T) {
	schema := UnionOf(String().Min(2), Number().Gte(10))
	cases := []struct {
		name    string
		in      any
		wantOK  bool
		wantOut any
	}{
		{"string ok", "ab", true, "ab"},
		{"string short", "a", false, nil},
		{"number ok", 10.0, true, 10.0},
		{"number small", 9.0, false, nil},
		{"bool", true, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := schema.SafeParse(tc.in)
			if res.Success != tc.wantOK {
				t.Fatalf("success=%v err=%v", res.Success, res.Error)
			}
			if tc.wantOK && !reflect.DeepEqual(res.Data, tc.wantOut) {
				t.Fatalf("data=%#v want %#v", res.Data, tc.wantOut)
			}
		})
	}
}

// --- discriminated-unions.test.ts ---

func TestParityDiscUnionValuesProp(t *testing.T) {
	// Port: "_values" (subset of wrappers that propagate Values)
	if String().Internals().Values != nil {
		t.Fatal("string Values should be nil")
	}
	if _, ok := Enum("a", "b").Internals().Values["a"]; !ok {
		t.Fatal("enum missing a")
	}
	if _, ok := Literal("test").Internals().Values["test"]; !ok {
		t.Fatal("literal missing test")
	}
	if _, ok := Nil().Internals().Values[nil]; !ok {
		t.Fatal("null Values")
	}
	opt := Optional(Literal("test"))
	if _, ok := opt.Internals().Values["test"]; !ok {
		t.Fatal("optional keeps literal")
	}
	if _, ok := opt.Internals().Values[Missing]; !ok {
		// Optional may use Missing or a distinct undefined sentinel in Values.
		foundUndef := false
		for k := range opt.Internals().Values {
			if IsMissing(k) {
				foundUndef = true
			}
		}
		if !foundUndef {
			t.Fatalf("optional Values = %v", opt.Internals().Values)
		}
	}
}

func TestParityDiscUnionValidParse(t *testing.T) {
	// Port: "valid parse - object" / "valid - include discriminator key"
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	got, err := schema.Parse(map[string]any{"type": "a", "a": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string]any{"type": "a", "a": "abc"}) {
		t.Fatalf("got %#v", got)
	}
}

func TestParityDiscUnionOptionalDiscriminator(t *testing.T) {
	// Port: "valid - optional discriminator (object)"
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Optional(Literal("a")), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	cases := []map[string]any{
		{"type": "a", "a": "abc"},
		{"a": "abc"},
	}
	for _, in := range cases {
		if _, err := schema.Parse(in); err != nil {
			t.Fatalf("Parse(%v): %v", in, err)
		}
	}
}

func TestParityDiscUnionVariousPrimitives(t *testing.T) {
	// Port: "valid - discriminator value of various primitive types"
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("1"), "val": String()}),
		Object(Shape{"type": Literal(1.0), "val": String()}),
		Object(Shape{"type": Literal(true), "val": String()}),
		Object(Shape{"type": Literal("null"), "val": String()}),
		Object(Shape{"type": Nil(), "val": String()}),
	})
	ok := []any{"1", 1.0, true, "null", nil}
	for _, disc := range ok {
		t.Run(fmt.Sprintf("%v", disc), func(t *testing.T) {
			got, err := schema.Parse(map[string]any{"type": disc, "val": "val"})
			if err != nil {
				t.Fatal(err)
			}
			if got.(map[string]any)["val"] != "val" {
				t.Fatalf("%#v", got)
			}
		})
	}
	fail := schema.SafeParse(map[string]any{"type": "not_a_key", "val": "val"})
	if fail.Success {
		t.Fatal("expected failure")
	}
}

func TestParityDiscUnionInvalidNull(t *testing.T) {
	// Port: "invalid - null"
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	res := schema.SafeParse(nil)
	if res.Success || res.Error.Issues[0].Code != IssueInvalidType || res.Error.Issues[0].Expected != "object" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityDiscUnionInvalidDiscriminator(t *testing.T) {
	// Port: "invalid discriminator value"
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	res := schema.SafeParse(map[string]any{"type": "x", "a": "abc"})
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidUnion || iss.Discriminator != "type" {
		t.Fatalf("got %+v", iss)
	}
	if !strings.Contains(iss.Message, "Invalid discriminator value") {
		t.Fatalf("message = %q", iss.Message)
	}
}

func TestParityDiscUnionFallback(t *testing.T) {
	// Port: "invalid discriminator value - unionFallback"
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	}, DiscUnionParams{UnionFallback: true})
	res := schema.SafeParse(map[string]any{"type": "x", "a": "abc"})
	if res.Success || res.Error.Issues[0].Code != IssueInvalidUnion {
		t.Fatalf("got %+v", res.Error)
	}
	if len(res.Error.Issues[0].Errors) < 1 {
		t.Fatalf("want nested arm errors, got %+v", res.Error.Issues[0])
	}
}

func TestParityDiscUnionValidDiscInvalidData(t *testing.T) {
	// Port: "valid discriminator value, invalid data"
	schema := DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("b"), "b": String()}),
	})
	res := schema.SafeParse(map[string]any{"type": "a", "b": "abc"})
	if res.Success {
		t.Fatal("expected failure")
	}
	found := false
	for _, iss := range res.Error.Issues {
		if iss.Code == IssueInvalidType && len(iss.Path) > 0 && iss.Path[0] == "a" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues = %+v", res.Error.Issues)
	}
}

func TestParityDiscUnionEmpty(t *testing.T) {
	// Port: "z.discriminatedUnion with empty options"
	schema := DiscriminatedUnion("type", nil)
	if schema.SafeParse("nope").Error.Issues[0].Code != IssueInvalidType {
		t.Fatal("non-object")
	}
	res := schema.SafeParse(map[string]any{"type": "x"})
	if res.Success || res.Error.Issues[0].Code != IssueInvalidUnion {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityDiscUnionDuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "Duplicate discriminator") {
			t.Fatalf("recover = %v", r)
		}
	}()
	_ = DiscriminatedUnion("type", []AnySchemaLike{
		Object(Shape{"type": Literal("a"), "a": String()}),
		Object(Shape{"type": Literal("a"), "b": String()}),
	})
}
