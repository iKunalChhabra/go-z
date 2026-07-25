package zod

import (
	"math"
	"math/big"
	"reflect"
	"regexp"
	"testing"
	"time"
)

// Parity ports from:
//   literal.test.ts, enum.test.ts, bigint.test.ts, date.test.ts (Time),
//   anyunknown.test.ts, primitive.test.ts, intersection.test.ts,
//   lazy.test.ts / recursive-types.test.ts, registries.test.ts,
//   description.test.ts / describe-meta-checks.test.ts, global-config.test.ts

// --- literal.test.ts ---

func TestParityLiteralPassingFailing(t *testing.T) {
	tuna := Literal("tuna")
	fortyTwo := Literal(42.0)
	litTrue := Literal(true)
	if _, err := tuna.Parse("tuna"); err != nil {
		t.Fatal(err)
	}
	if _, err := fortyTwo.Parse(42.0); err != nil {
		t.Fatal(err)
	}
	if _, err := litTrue.Parse(true); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		s  *LiteralSchema
		in any
	}{
		{tuna, "shark"},
		{fortyTwo, 43.0},
		{litTrue, false},
	} {
		if tc.s.SafeParse(tc.in).Success {
			t.Fatalf("%v should fail", tc.in)
		}
	}
}

func TestParityLiteralIssueShape(t *testing.T) {
	res := Literal("tuna").SafeParse("shark")
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidValue {
		t.Fatalf("code = %s", iss.Code)
	}
	if iss.Message != `Invalid input: expected "tuna"` {
		t.Fatalf("message = %q", iss.Message)
	}
	if !reflect.DeepEqual(iss.Values, []any{"tuna"}) {
		t.Fatalf("values = %#v", iss.Values)
	}
}

func TestParityLiteralCustomMessage(t *testing.T) {
	res := Literal("tuna", "That's not a tuna").SafeParse("shark")
	if res.Success || res.Error.Issues[0].Message != "That's not a tuna" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityLiteralBigInt(t *testing.T) {
	res := Literal(big.NewInt(12)).SafeParse(big.NewInt(13))
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Error.Issues[0].Code != IssueInvalidValue {
		t.Fatalf("%+v", res.Error.Issues[0])
	}
}

func TestParityLiteralValueGetter(t *testing.T) {
	if Literal("tuna").Value() != "tuna" {
		t.Fatal("value getter")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("multi-value Literal.Value should panic")
		}
	}()
	_ = Literal(1.0, 2.0, 3.0).Value()
}

func TestParityLiteralTemplate(t *testing.T) {
	hello := TemplateLiteral([]any{"hello"})
	if got, err := hello.Parse("hello"); err != nil || got != "hello" {
		t.Fatalf("%v %v", got, err)
	}
	if hello.SafeParse("world").Success {
		t.Fatal("expected failure")
	}

	url := TemplateLiteral([]any{
		"https://",
		String().Regex(regexp.MustCompile(`\w+`)),
		".",
		Enum("com", "net"),
	})
	if _, err := url.Parse("https://example.com"); err != nil {
		t.Fatal(err)
	}
	if url.SafeParse("https://example.org").Success {
		t.Fatal("expected enum failure")
	}
}

// --- enum.test.ts ---

func TestParityEnumBasic(t *testing.T) {
	schema := Enum("Red", "Green", "Blue")
	if _, err := schema.Parse("Red"); err != nil {
		t.Fatal(err)
	}
	res := schema.SafeParse("Yellow")
	if res.Success || res.Error.Issues[0].Code != IssueInvalidValue {
		t.Fatalf("%+v", res.Error)
	}
	opts := schema.Options()
	if !reflect.DeepEqual(opts, []string{"Red", "Green", "Blue"}) {
		t.Fatalf("options = %v", opts)
	}
}

func TestParityNativeEnum(t *testing.T) {
	schema := NativeEnum(map[string]string{"A": "a", "B": "b"})
	if _, err := schema.Parse("a"); err != nil {
		t.Fatal(err)
	}
	if schema.SafeParse("A").Success {
		t.Fatal("keys are not accepted — values are")
	}
	m := schema.EnumMap()
	if m["A"] != "a" || m["B"] != "b" {
		t.Fatalf("%#v", m)
	}
}

func TestParityEnumValuesProp(t *testing.T) {
	vals := Enum("x", "y").Internals().Values
	if _, ok := vals["x"]; !ok {
		t.Fatal("missing x")
	}
	native := NativeEnum(map[string]string{"a": "A", "b": "B"}).Internals().Values
	if _, ok := native["A"]; !ok {
		t.Fatal("native values are map values")
	}
}

// --- bigint.test.ts ---

func TestParityBigIntChecks(t *testing.T) {
	five := big.NewInt(5)
	cases := []struct {
		name   string
		schema *BigIntSchema
		ok     []*big.Int
		fail   []*big.Int
	}{
		{"gt", BigInt().Gt(five), []*big.Int{big.NewInt(6)}, []*big.Int{big.NewInt(5)}},
		{"gte", BigInt().Gte(five), []*big.Int{big.NewInt(5), big.NewInt(6)}, []*big.Int{big.NewInt(4)}},
		{"lt", BigInt().Lt(five), []*big.Int{big.NewInt(4)}, []*big.Int{big.NewInt(5)}},
		{"lte", BigInt().Lte(five), []*big.Int{big.NewInt(5), big.NewInt(4)}, []*big.Int{big.NewInt(6)}},
		{"positive", BigInt().Positive(), []*big.Int{big.NewInt(3)}, []*big.Int{big.NewInt(0), big.NewInt(-2)}},
		{"negative", BigInt().Negative(), []*big.Int{big.NewInt(-2)}, []*big.Int{big.NewInt(0), big.NewInt(3)}},
		{"nonneg", BigInt().NonNegative(), []*big.Int{big.NewInt(0), big.NewInt(7)}, []*big.Int{big.NewInt(-1)}},
		{"nonpos", BigInt().NonPositive(), []*big.Int{big.NewInt(0), big.NewInt(-12)}, []*big.Int{big.NewInt(1)}},
		{"multiple", BigInt().MultipleOf(five), []*big.Int{big.NewInt(15)}, []*big.Int{big.NewInt(13)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range tc.ok {
				if _, err := tc.schema.Parse(v); err != nil {
					t.Fatalf("ok %v: %v", v, err)
				}
			}
			for _, v := range tc.fail {
				if tc.schema.SafeParse(v).Success {
					t.Fatalf("fail %v should reject", v)
				}
			}
		})
	}
	if _, err := BigInt().Parse(big.NewInt(0)); err != nil {
		t.Fatal(err)
	}
}

// --- date.test.ts (Time) ---

func TestParityTimeMinMax(t *testing.T) {
	before := time.Date(2022, 11, 4, 0, 0, 0, 0, time.UTC)
	bench := time.Date(2022, 11, 5, 0, 0, 0, 0, time.UTC)
	after := time.Date(2022, 11, 6, 0, 0, 0, 0, time.UTC)

	minCheck := Time().Min(bench)
	maxCheck := Time().Max(bench)

	if _, err := minCheck.Parse(bench); err != nil {
		t.Fatal(err)
	}
	if _, err := minCheck.Parse(after); err != nil {
		t.Fatal(err)
	}
	res := minCheck.SafeParse(before)
	if res.Success || res.Error.Issues[0].Code != IssueTooSmall || res.Error.Issues[0].Origin != "date" {
		t.Fatalf("min: %+v", res.Error)
	}

	if _, err := maxCheck.Parse(bench); err != nil {
		t.Fatal(err)
	}
	if _, err := maxCheck.Parse(before); err != nil {
		t.Fatal(err)
	}
	res = maxCheck.SafeParse(after)
	if res.Success || res.Error.Issues[0].Code != IssueTooBig || res.Error.Issues[0].Origin != "date" {
		t.Fatalf("max: %+v", res.Error)
	}
}

// --- anyunknown.test.ts ---

func TestParityAnyUnknownNever(t *testing.T) {
	if _, err := Any().Parse("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := Any().Parse(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Unknown().Parse(123); err != nil {
		t.Fatal(err)
	}
	n := Never()
	for _, in := range []any{Missing, "asdf", nil, 1} {
		if n.SafeParse(in).Success {
			t.Fatalf("never should reject %#v", in)
		}
	}
}

// --- primitive.test.ts ---

func TestParityPrimitives(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		s := String()
		if _, err := s.Parse("foo"); err != nil {
			t.Fatal(err)
		}
		for _, in := range []any{1.2, true, nil} {
			if s.SafeParse(in).Success {
				t.Fatalf("should fail %#v", in)
			}
		}
	})
	t.Run("number", func(t *testing.T) {
		s := Number()
		if _, err := s.Parse(1.5); err != nil {
			t.Fatal(err)
		}
		for _, in := range []any{"foo", true, nil, big.NewInt(17)} {
			if s.SafeParse(in).Success {
				t.Fatalf("should fail %#v", in)
			}
		}
	})
	t.Run("bigint", func(t *testing.T) {
		s := BigInt()
		if _, err := s.Parse(big.NewInt(17)); err != nil {
			t.Fatal(err)
		}
		for _, in := range []any{"foo", 1.2, true, nil} {
			if s.SafeParse(in).Success {
				t.Fatalf("should fail %#v", in)
			}
		}
	})
	t.Run("boolean", func(t *testing.T) {
		s := Bool()
		if _, err := s.Parse(true); err != nil {
			t.Fatal(err)
		}
		for _, in := range []any{"foo", 1.2, nil} {
			if s.SafeParse(in).Success {
				t.Fatalf("should fail %#v", in)
			}
		}
	})
	t.Run("time", func(t *testing.T) {
		s := Time()
		if _, err := s.Parse(time.Now()); err != nil {
			t.Fatal(err)
		}
		for _, in := range []any{"foo", 1.2, true, nil} {
			if s.SafeParse(in).Success {
				t.Fatalf("should fail %#v", in)
			}
		}
	})
	t.Run("null", func(t *testing.T) {
		s := Nil()
		if _, err := s.Parse(nil); err != nil {
			t.Fatal(err)
		}
		if s.SafeParse("x").Success {
			t.Fatal("null rejects string")
		}
	})
	t.Run("nan", func(t *testing.T) {
		s := Nan()
		if _, err := s.Parse(math.NaN()); err != nil {
			t.Fatal(err)
		}
		if s.SafeParse(1.0).Success {
			t.Fatal("nan rejects number")
		}
	})
	t.Run("optional nullable wrappers", func(t *testing.T) {
		if _, err := Optional(String()).Parse(Missing); err != nil {
			t.Fatal(err)
		}
		if _, err := Nullable(String()).Parse(nil); err != nil {
			t.Fatal(err)
		}
		if Optional(String()).SafeParse(1).Success {
			t.Fatal("optional still typechecks")
		}
	})
	t.Run("literals", func(t *testing.T) {
		if _, err := Literal("asdf").Parse("asdf"); err != nil {
			t.Fatal(err)
		}
		if _, err := Literal(12.0).Parse(12.0); err != nil {
			t.Fatal(err)
		}
		if _, err := Literal(true).Parse(true); err != nil {
			t.Fatal(err)
		}
		if _, err := Literal(big.NewInt(42)).Parse(big.NewInt(42)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestParitySymbolFilePromiseFunctionUnsupported(t *testing.T) {
	t.Skip("symbol/file/promise/function not supported in go-zod")
}

// --- intersection.test.ts ---

func TestParityIntersectionObject(t *testing.T) {
	A := Object(Shape{"a": String()})
	B := Object(Shape{"b": String()})
	C := Intersection(A, B)
	data := map[string]any{"a": "foo", "b": "foo"}
	got, err := C.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, data) {
		t.Fatalf("%#v", got)
	}
	if C.SafeParse(map[string]any{"a": "foo"}).Success {
		t.Fatal("missing b")
	}
}

func TestParityIntersectionLoose(t *testing.T) {
	A := Object(Shape{"a": String()}).Loose()
	B := Object(Shape{"b": String()})
	C := Intersection(A, B)
	data := map[string]any{"a": "foo", "b": "foo", "c": "extra"}
	got, err := C.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["a"] != "foo" || m["b"] != "foo" {
		t.Fatalf("%#v", got)
	}
}

func TestParityIntersectionStrictStrict(t *testing.T) {
	A := Object(Shape{"a": String()}).Strict()
	B := Object(Shape{"b": String()}).Strict()
	C := Intersection(A, B)
	if _, err := C.Parse(map[string]any{"a": "foo", "b": "bar"}); err != nil {
		t.Fatal(err)
	}
	res := C.SafeParse(map[string]any{"a": "foo", "b": "bar", "c": "extra"})
	if res.Success {
		t.Fatal("extra key should fail")
	}
	found := false
	for _, iss := range res.Error.Issues {
		if iss.Code == IssueUnrecognizedKeys {
			found = true
		}
	}
	if !found {
		t.Fatalf("want unrecognized_keys, got %+v", res.Error.Issues)
	}
}

func TestParityIntersectionDeep(t *testing.T) {
	Animal := Object(Shape{
		"properties": Object(Shape{"is_animal": Bool()}),
	})
	Cat := Intersection(
		Object(Shape{"properties": Object(Shape{"jumped": Bool()})}),
		Animal,
	)
	got, err := Cat.Parse(map[string]any{
		"properties": map[string]any{"is_animal": true, "jumped": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	props := got.(map[string]any)["properties"].(map[string]any)
	if props["is_animal"] != true || props["jumped"] != true {
		t.Fatalf("%#v", got)
	}
}

// --- lazy / recursive-types ---

func TestParityLazyBasic(t *testing.T) {
	schema := Lazy(func() AnySchemaLike { return String() })
	if _, err := schema.Parse("asdf"); err != nil {
		t.Fatal(err)
	}
	if schema.SafeParse(1.0).Success {
		t.Fatal("expected failure")
	}
}

func TestParityLazyRecursiveCategory(t *testing.T) {
	var category *LazySchema
	category = Lazy(func() AnySchemaLike {
		return Object(Shape{
			"name":          String(),
			"subcategories": Array(category),
		})
	})
	data := map[string]any{
		"name": "I",
		"subcategories": []any{
			map[string]any{
				"name": "A",
				"subcategories": []any{
					map[string]any{"name": "1", "subcategories": []any{}},
				},
			},
		},
	}
	got, err := category.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["name"] != "I" {
		t.Fatalf("%#v", got)
	}
}

func TestParityLazyRecursiveUnion(t *testing.T) {
	var ll *LazySchema
	ll = Lazy(func() AnySchemaLike {
		return Union([]AnySchemaLike{
			Nil(),
			Object(Shape{"value": Number(), "next": ll}),
		})
	})
	data := map[string]any{
		"value": 1.0,
		"next": map[string]any{
			"value": 2.0,
			"next":  nil,
		},
	}
	if _, err := ll.Parse(data); err != nil {
		t.Fatal(err)
	}
	if _, err := ll.Parse(nil); err != nil {
		t.Fatal(err)
	}
}

// --- registries / description / meta ---

func TestParityRegistryCRUD(t *testing.T) {
	reg := NewRegistry[map[string]any]()
	a := String()
	reg.Add(a, map[string]any{"field": "sup"})
	if !reg.Has(a) {
		t.Fatal("has")
	}
	got, ok := reg.Get(a)
	if !ok || got["field"] != "sup" {
		t.Fatalf("get %#v", got)
	}
	reg.Remove(a)
	if reg.Has(a) {
		t.Fatal("removed")
	}
}

func TestParityRegistryTypedMeta(t *testing.T) {
	type meta struct {
		Name        string
		Description string
	}
	reg := NewRegistry[meta]()
	a := String()
	reg.Add(a, meta{Name: "hello", Description: "world"})
	got, ok := reg.Get(a)
	if !ok || got.Name != "hello" {
		t.Fatalf("%#v", got)
	}
	reg.Clear()
	if reg.Has(a) {
		t.Fatal("clear")
	}
}

func TestParityRegistryIDIndex(t *testing.T) {
	reg := NewRegistry[map[string]any]()
	a, b := String(), Number()
	reg.Add(a, map[string]any{"id": "shared"})
	if got, ok := reg.GetByID("shared"); !ok || got.Internals() != a.Internals() {
		t.Fatal("id a")
	}
	reg.Add(b, map[string]any{"id": "shared"})
	if got, ok := reg.GetByID("shared"); !ok || got.Internals() != b.Internals() {
		t.Fatal("overwrite")
	}
}

func TestParityDescribeMeta(t *testing.T) {
	GlobalRegistry.Clear()
	t.Cleanup(func() { GlobalRegistry.Clear() })

	a := String()
	Describe(a, "a description")
	if GetDescription(a) != "a description" {
		t.Fatalf("desc = %q", GetDescription(a))
	}
	Meta(a, map[string]any{"title": "T", "examples": []any{"x"}})
	m, ok := GlobalRegistry.Get(a)
	if !ok || m["title"] != "T" || m["description"] != "a description" {
		t.Fatalf("meta merge: %#v", m)
	}
}

func TestParityGlobalConfig(t *testing.T) {
	prev := Configure(Config{
		CustomError: ErrorMap(func(iss *Issue) string {
			if iss.Code == IssueInvalidType {
				return "GLOBAL"
			}
			return ""
		}),
	})
	t.Cleanup(func() { Configure(prev) })

	res := String().SafeParse(123)
	if res.Success || res.Error.Issues[0].Message != "GLOBAL" {
		t.Fatalf("got %+v", res.Error)
	}
	cfg := GetConfig()
	if cfg.CustomError == nil {
		t.Fatal("config should be set")
	}
}

func TestParityBrandUnsupported(t *testing.T) {
	t.Skip("brand not supported in go-zod")
}
