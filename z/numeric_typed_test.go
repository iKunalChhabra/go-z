package z

import (
	"math"
	"testing"
)

// Each numeric constructor produces the Go type its name promises.
func TestNumericConstructorOutputTypes(t *testing.T) {
	if got, err := Int().Parse(3.0); err != nil || got != 3 {
		t.Fatalf("Int(): %#v %v", got, err)
	}
	if got, err := Int32().Parse(7); err != nil || got != int32(7) {
		t.Fatalf("Int32(): %#v %v", got, err)
	}
	if got, err := Uint32().Parse(7); err != nil || got != uint32(7) {
		t.Fatalf("Uint32(): %#v %v", got, err)
	}
	if got, err := Int64().Parse(7); err != nil || got != int64(7) {
		t.Fatalf("Int64(): %#v %v", got, err)
	}
	if got, err := Float32().Parse(1.5); err != nil || got != float32(1.5) {
		t.Fatalf("Float32(): %#v %v", got, err)
	}
	if got, err := Float64().Parse(1.5); err != nil || got != 1.5 {
		t.Fatalf("Float64(): %#v %v", got, err)
	}
	if got, err := Number().Parse(1.5); err != nil || got != 1.5 {
		t.Fatalf("Number(): %#v %v", got, err)
	}
}

// A value the output type cannot hold is reported, not truncated to fit. The
// issue is the same one the number format check has always produced.
func TestNumericOutOfRangeIsReported(t *testing.T) {
	cases := []struct {
		name string
		err  *Error
		code IssueCode
	}{
		{"Int 3.7", firstError(t, Int().SafeParse(3.7)), IssueInvalidType},
		{"Int32 overflow", firstError(t, Int32().SafeParse(math.MaxInt32+1)), IssueTooBig},
		{"Uint32 negative", firstError(t, Uint32().SafeParse(-1)), IssueTooSmall},
		{"Float32 overflow", firstError(t, Float32().SafeParse(MaxFloat32Exact*2)), IssueTooBig},
	}
	for _, c := range cases {
		if got := c.err.Issues[0].Code; got != c.code {
			t.Fatalf("%s: code=%s want %s", c.name, got, c.code)
		}
	}
	if msg := firstError(t, Int().SafeParse(3.7)).Issues[0].Message; msg != "Invalid input: expected int, received number" {
		t.Fatalf("message=%q", msg)
	}
}

func firstError[T Numeric](t *testing.T, r SafeParseResult[T]) *Error {
	t.Helper()
	if r.Success {
		t.Fatal("expected failure")
	}
	return r.Error
}

// Int64 keeps the full 64-bit range: it never round-trips through float64.
func TestInt64FullRange(t *testing.T) {
	for _, in := range []int64{math.MaxInt64, math.MinInt64, 9007199254740993} {
		got, err := Int64().Parse(in)
		if err != nil || got != in {
			t.Fatalf("Int64().Parse(%d) = %d, %v", in, got, err)
		}
	}
	// Non-integers are a type error for Int64: no JSON format describes a
	// 64-bit integer.
	res := Int64().SafeParse(3.7)
	if res.Success || res.Error.Issues[0].Code != IssueInvalidType {
		t.Fatalf("got %+v", res)
	}
}

// Ints above the JSON safe range are rejected by Int(), which models a JSON
// number, but accepted by Int64().
func TestIntSafeIntegerBoundary(t *testing.T) {
	safe := int64(MaxSafeInteger)
	if got, err := Int().Parse(safe); err != nil || int64(got) != safe {
		t.Fatalf("safe integer must parse: %#v %v", got, err)
	}
	if Int().SafeParse(safe + 1).Success {
		t.Fatal("Int() must reject values outside the safe-integer range")
	}
	if _, err := Int64().Parse(safe + 1); err != nil {
		t.Fatalf("Int64() must accept it: %v", err)
	}
}

// Number().Integer() is the check form: a float64 schema restricted to whole
// numbers, output type unchanged.
func TestNumberInteger(t *testing.T) {
	got, err := Number().Integer().Parse(4)
	if err != nil || got != 4.0 {
		t.Fatalf("got %#v %v", got, err)
	}
	if Number().Integer().SafeParse(3.14).Success {
		t.Fatal("3.14 is not an integer")
	}
	if Number().Safe().SafeParse(3.14).Success {
		t.Fatal("Safe is an alias of Integer")
	}
}

// Checks, wrappers and refinements all speak the output type.
func TestNumericTypedEdges(t *testing.T) {
	if _, err := Int().Gte(1).Lte(10).MultipleOf(2).Parse(4); err != nil {
		t.Fatalf("bounds: %v", err)
	}
	if Int().Gte(1).SafeParse(0).Success {
		t.Fatal("expected too_small")
	}

	def, err := Int().Default(5).Parse(Missing)
	if err != nil || def != 5 {
		t.Fatalf("Default: %#v %v", def, err)
	}
	ptr, err := Int().Optional().Parse(Missing)
	if err != nil || ptr != nil {
		t.Fatalf("Optional: %#v %v", ptr, err)
	}
	if v, err := Int().Optional().Parse(3); err != nil || v == nil || *v != 3 {
		t.Fatalf("Optional value: %#v %v", v, err)
	}
	if v, err := Uint32().Catch(9).Parse("nope"); err != nil || v != 9 {
		t.Fatalf("Catch: %#v %v", v, err)
	}

	even := Int().Refine(func(n int) bool { return n%2 == 0 }, "must be even")
	if _, err := even.Parse(4); err != nil {
		t.Fatalf("refine: %v", err)
	}
	if res := even.SafeParse(5); res.Success || res.Error.Issues[0].Message != "must be even" {
		t.Fatalf("got %+v", res)
	}

	seen := int32(0)
	Int32().SuperRefine(func(n int32, ctx *RefinementCtx) { seen = n }).Parse(11)
	if seen != 11 {
		t.Fatalf("SuperRefine saw %d", seen)
	}
}

// Object fields carry the field schema's output type into the result map.
func TestObjectCarriesNumericTypes(t *testing.T) {
	schema := Object(Shape{
		"count": Int(),
		"ratio": Number(),
		"id":    Int64(),
		"port":  Uint32(),
	})
	input := map[string]any{"count": 2.0, "ratio": 0.5, "id": 4.0, "port": 8080.0}
	m, err := schema.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if m["count"] != 2 || m["ratio"] != 0.5 || m["id"] != int64(4) || m["port"] != uint32(8080) {
		t.Fatalf("types not carried: %#v", m)
	}

	type config struct {
		Count int     `json:"count"`
		Ratio float64 `json:"ratio"`
		ID    int64   `json:"id"`
		Port  uint32  `json:"port"`
	}
	cfg, err := ToStruct[config](schema).Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Count != 2 || cfg.Ratio != 0.5 || cfg.ID != 4 || cfg.Port != 8080 {
		t.Fatalf("ToStruct: %+v", cfg)
	}
}

func TestNumericCoercion(t *testing.T) {
	if got, err := Int(Params{Coerce: true}).Parse("42"); err != nil || got != 42 {
		t.Fatalf("coerce int: %#v %v", got, err)
	}
	// A coerced number that is not an integer still fails the format check
	// rather than being truncated.
	res := Int(Params{Coerce: true}).SafeParse("33.7")
	if res.Success || res.Error.Issues[0].Expected != "int" {
		t.Fatalf("got %+v", res)
	}
	if got, err := Int64(Params{Coerce: true}).Parse("9007199254740993"); err != nil || got != 9007199254740993 {
		t.Fatalf("coerce keeps precision: %#v %v", got, err)
	}
	if got, err := Number(Params{Coerce: true}).Parse("1.5"); err != nil || got != 1.5 {
		t.Fatalf("coerce number: %#v %v", got, err)
	}
}

func TestNumericRejectsNonNumbers(t *testing.T) {
	for _, s := range []func(any) (any, error){
		func(v any) (any, error) { return Int().Parse(v) },
		func(v any) (any, error) { return Number().Parse(v) },
		func(v any) (any, error) { return Int64().Parse(v) },
	} {
		for _, in := range []any{"x", true, nil, math.NaN(), math.Inf(1)} {
			if _, err := s(in); err == nil {
				t.Fatalf("expected failure for %#v", in)
			}
		}
	}
}
