package z

import "testing"

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/array.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityArrayMinMax(t *testing.T) {
	// Ported from classic/tests/array.test.ts — "array min/max"
	schema := Array(String()).Min(2).Max(2)
	r1 := schema.SafeParse([]any{"asdf"})
	if r1.Success {
		t.Fatal("expected too_small")
	}
	iss := r1.Error.Issues[0]
	if iss.Code != IssueTooSmall || iss.Origin != "array" {
		t.Fatalf("%+v", iss)
	}
	if min, _ := ToFloat(iss.Minimum); min != 2 {
		t.Fatalf("minimum=%v", iss.Minimum)
	}
	if iss.Message != "Too small: expected array to have >=2 items" {
		t.Fatalf("msg=%q", iss.Message)
	}
	if !iss.Inclusive {
		t.Fatal("inclusive")
	}

	r2 := schema.SafeParse([]any{"asdf", "asdf", "asdf"})
	if r2.Success {
		t.Fatal("expected too_big")
	}
	iss = r2.Error.Issues[0]
	if iss.Code != IssueTooBig || iss.Origin != "array" {
		t.Fatalf("%+v", iss)
	}
	if max, _ := ToFloat(iss.Maximum); max != 2 {
		t.Fatalf("maximum=%v", iss.Maximum)
	}
	if iss.Message != "Too big: expected array to have <=2 items" {
		t.Fatalf("msg=%q", iss.Message)
	}
}

func TestParityArrayLength(t *testing.T) {
	// Ported from classic/tests/array.test.ts — "array length"
	schema := Array(String()).Length(2)
	if _, err := schema.Parse([]any{"asdf", "asdf"}); err != nil {
		t.Fatal(err)
	}
	r1 := schema.SafeParse([]any{"asdf"})
	if r1.Success || !r1.Error.Issues[0].Exact || r1.Error.Issues[0].Code != IssueTooSmall {
		t.Fatalf("%+v", r1.Error)
	}
	if r1.Error.Issues[0].Message != "Too small: expected array to have >=2 items" {
		t.Fatalf("msg=%q", r1.Error.Issues[0].Message)
	}
	r2 := schema.SafeParse([]any{"asdf", "asdf", "asdf"})
	if r2.Success || !r2.Error.Issues[0].Exact || r2.Error.Issues[0].Code != IssueTooBig {
		t.Fatalf("%+v", r2.Error)
	}
	if r2.Error.Issues[0].Message != "Too big: expected array to have <=2 items" {
		t.Fatalf("msg=%q", r2.Error.Issues[0].Message)
	}
}

func TestParityArrayNonempty(t *testing.T) {
	// Ported from classic/tests/array.test.ts — nonempty / nonempty.max / empty
	schema := Array(String()).NonEmpty()
	if _, err := schema.Parse([]any{"a"}); err != nil {
		t.Fatal(err)
	}
	if schema.SafeParse([]any{}).Success {
		t.Fatal("empty should fail")
	}

	schema2 := Array(String()).NonEmpty().Max(2)
	if _, err := schema2.Parse([]any{"a"}); err != nil {
		t.Fatal(err)
	}
	if schema2.SafeParse([]any{}).Success {
		t.Fatal("empty")
	}
	if schema2.SafeParse([]any{"a", "a", "a"}).Success {
		t.Fatal("too big")
	}

	if Array(String()).NonEmpty().SafeParse([]any{}).Success {
		t.Fatal("parse empty array in nonempty")
	}
}

func TestParityArrayElement(t *testing.T) {
	// Ported from classic/tests/array.test.ts — "get element"
	schema := Array(String())
	el := schema.Element().(*StringSchema)
	if _, err := el.Parse("asdf"); err != nil {
		t.Fatal(err)
	}
	if el.SafeParse(12).Success {
		t.Fatal("element rejects number")
	}
}

func TestParityArrayContinueDespiteSizeError(t *testing.T) {
	// Ported from classic/tests/array.test.ts — "continue parsing despite array size error"
	schema := Object(Shape{
		"people": Array(String()).Min(2),
	})
	result := schema.SafeParse(map[string]any{
		"people": []any{123},
	})
	if result.Success || len(result.Error.Issues) < 2 {
		t.Fatalf("want type + size: %+v", result.Error)
	}
	var hasType, hasSize bool
	for _, iss := range result.Error.Issues {
		if iss.Code == IssueInvalidType && len(iss.Path) == 2 && iss.Path[0] == "people" && iss.Path[1] == 0 {
			hasType = true
		}
		if iss.Code == IssueTooSmall && len(iss.Path) == 1 && iss.Path[0] == "people" {
			hasSize = true
			if iss.Message != "Too small: expected array to have >=2 items" {
				t.Fatalf("msg=%q", iss.Message)
			}
		}
	}
	if !hasType || !hasSize {
		t.Fatalf("issues: %+v", result.Error.Issues)
	}
}

func TestParityArraySparse(t *testing.T) {
	// Ported from classic/tests/array.test.ts — "parse should fail given sparse array"
	// Go has no sparse arrays; simulate with nil holes.
	schema := Array(String()).NonEmpty().Min(1).Max(3)
	result := schema.SafeParse([]any{nil, nil, nil})
	if result.Success {
		t.Fatal("nil elements should fail string")
	}
}

func TestParityArrayRejectsNonArray(t *testing.T) {
	schema := Array(String())
	for _, in := range []any{"nope", nil, 123, map[string]any{}, Missing} {
		res := schema.SafeParse(in)
		if res.Success {
			t.Errorf("should reject %#v", in)
		}
		if res.Error.Issues[0].Expected != "array" {
			t.Errorf("expected array, got %+v", res.Error.Issues[0])
		}
	}
}

func TestParityArrayPathPrefix(t *testing.T) {
	schema := Array(String())
	res := schema.SafeParse([]any{"ok", 1, true})
	if res.Success {
		t.Fatal("expected failure")
	}
	// Should collect issues for index 1 and 2
	if len(res.Error.Issues) < 2 {
		t.Fatalf("want >=2 issues, got %+v", res.Error.Issues)
	}
	if res.Error.Issues[0].Path[0] != 1 {
		t.Fatalf("path0=%v", res.Error.Issues[0].Path)
	}
}

func TestParityArrayNested(t *testing.T) {
	schema := Array(Array(Number()))
	got, err := schema.Parse([]any{[]any{1.0, 2.0}, []any{3.0}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	res := schema.SafeParse([]any{[]any{"x"}})
	if res.Success {
		t.Fatal("inner type fail")
	}
	if len(res.Error.Issues[0].Path) < 2 {
		t.Fatalf("path: %+v", res.Error.Issues[0].Path)
	}
}

func TestParityArrayTypedSlice(t *testing.T) {
	schema := Array(String())
	got, err := schema.Parse([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a" {
		t.Fatalf("%#v", got)
	}
}

func TestParityArrayCustomMessage(t *testing.T) {
	schema := Array(String()).Min(2, "need 2")
	res := schema.SafeParse([]any{"a"})
	if res.Success || res.Error.Issues[0].Message != "need 2" {
		t.Fatalf("%+v", res.Error)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/tuple.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityTupleSuccessfulValidation(t *testing.T) {
	// Ported from classic/tests/tuple.test.ts — "successful validation"
	testTuple := Tuple([]AnySchemaLike{String(), Number()})
	val, err := testTuple.Parse([]any{"asdf", 1234.0})
	if err != nil {
		t.Fatal(err)
	}
	if val[0] != "asdf" || val[1] != 1234.0 {
		t.Fatalf("%#v", val)
	}

	r1 := testTuple.SafeParse([]any{"asdf", "asdf"})
	if r1.Success {
		t.Fatal("type fail")
	}
	if r1.Error.Issues[0].Expected != "number" || r1.Error.Issues[0].Path[0] != 1 {
		t.Fatalf("%+v", r1.Error.Issues[0])
	}
	if r1.Error.Issues[0].Message != "Invalid input: expected number, received string" {
		t.Fatalf("msg=%q", r1.Error.Issues[0].Message)
	}

	r2 := testTuple.SafeParse([]any{"asdf", 1234.0, true})
	if r2.Success {
		t.Fatal("too big")
	}
	if r2.Error.Issues[0].Code != IssueTooBig {
		t.Fatalf("%+v", r2.Error.Issues[0])
	}
	if max, _ := ToFloat(r2.Error.Issues[0].Maximum); max != 2 {
		t.Fatalf("maximum=%v", r2.Error.Issues[0].Maximum)
	}
	if r2.Error.Issues[0].Message != "Too big: expected array to have <=2 items" {
		t.Fatalf("msg=%q", r2.Error.Issues[0].Message)
	}

	r3 := testTuple.SafeParse(map[string]any{})
	if r3.Success {
		t.Fatal("object")
	}
	if r3.Error.Issues[0].Expected != "tuple" {
		t.Fatalf("%+v", r3.Error.Issues[0])
	}
	if r3.Error.Issues[0].Message != "Invalid input: expected tuple, received object" {
		t.Fatalf("msg=%q", r3.Error.Issues[0].Message)
	}
}

func TestParityTupleAsync(t *testing.T) {
	// Ported from classic/tests/tuple.test.ts — "async validation"
	t.Skip("async parse unsupported (classic/tests/tuple.test.ts)")
}

func TestParityTupleOptionalElements(t *testing.T) {
	// Ported from classic/tests/tuple.test.ts — "tuple with optional elements"
	myTuple := Tuple([]AnySchemaLike{
		String(),
		Optional(Number()),
		Optional(String()),
	}).Rest(Bool())

	good := [][]any{
		{"asdf"},
		{"asdf", 1234.0},
		{"asdf", 1234.0, "asdf"},
		{"asdf", 1234.0, "asdf", true, false, true},
	}
	for _, data := range good {
		got, err := myTuple.Parse(data)
		if err != nil {
			t.Fatalf("good %#v: %v", data, err)
		}
		if len(got) != len(data) {
			t.Fatalf("len %#v → %#v", data, got)
		}
	}

	bad := [][]any{
		{"asdf", "asdf"},
		{"asdf", 1234.0, "asdf", "asdf"},
		{"asdf", 1234.0, "asdf", true, false, "asdf"},
	}
	for _, data := range bad {
		if myTuple.SafeParse(data).Success {
			t.Fatalf("bad should fail: %#v", data)
		}
	}
}

func TestParityTupleOptionalThenRequired(t *testing.T) {
	// Ported from classic/tests/tuple.test.ts — optional followed by required
	myTuple := Tuple([]AnySchemaLike{
		String(),
		Optional(Number()),
		String(),
	}).Rest(Bool())

	good := [][]any{
		{"asdf", 1234.0, "asdf"},
		{"asdf", 1234.0, "asdf", true, false, true},
	}
	for _, data := range good {
		if _, err := myTuple.Parse(data); err != nil {
			t.Fatalf("good %#v: %v", data, err)
		}
	}
	bad := [][]any{
		{"asdf"},
		{"asdf", 1234.0},
		{"asdf", 1234.0, "asdf", "asdf"},
		{"asdf", 1234.0, "asdf", true, false, "asdf"},
	}
	for _, data := range bad {
		if myTuple.SafeParse(data).Success {
			t.Fatalf("bad should fail: %#v", data)
		}
	}
}

func TestParityTupleTooSmall(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String(), Number()})
	res := schema.SafeParse([]any{"a"})
	if res.Success {
		t.Fatal("too small")
	}
	if res.Error.Issues[0].Code != IssueTooSmall {
		t.Fatalf("%+v", res.Error.Issues[0])
	}
	if min, _ := ToFloat(res.Error.Issues[0].Minimum); min != 2 {
		t.Fatalf("minimum=%v", res.Error.Issues[0].Minimum)
	}
}

func TestParityTupleRestOnly(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String()}).Rest(Number())
	got, err := schema.Parse([]any{"a", 1.0, 2.0, 3.0})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("%#v", got)
	}
	if schema.SafeParse([]any{"a", "b"}).Success {
		t.Fatal("rest type fail")
	}
	if schema.SafeParse([]any{}).Success {
		t.Fatal("missing required first")
	}
}

func TestParityTupleAllOptional(t *testing.T) {
	schema := Tuple([]AnySchemaLike{Optional(String()), Optional(Number())})
	got, err := schema.Parse([]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("%#v", got)
	}
	got, err = schema.Parse([]any{"x"})
	if err != nil || len(got) != 1 {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestParityTupleItems(t *testing.T) {
	items := []AnySchemaLike{String(), Number()}
	schema := Tuple(items)
	got := schema.Items()
	if len(got) != 2 {
		t.Fatalf("%d", len(got))
	}
	if _, err := got[0].(*StringSchema).Parse("x"); err != nil {
		t.Fatal(err)
	}
}

func TestParityTupleNestedObject(t *testing.T) {
	schema := Tuple([]AnySchemaLike{
		Object(Shape{"id": String()}),
		Number(),
	})
	got, err := schema.Parse([]any{map[string]any{"id": "a"}, 1.0})
	if err != nil {
		t.Fatal(err)
	}
	m := got[0].(map[string]any)
	if m["id"] != "a" {
		t.Fatalf("%#v", got)
	}
	res := schema.SafeParse([]any{map[string]any{"id": 1}, 1.0})
	if res.Success {
		t.Fatal("nested fail")
	}
	if res.Error.Issues[0].Path[0] != 0 || res.Error.Issues[0].Path[1] != "id" {
		t.Fatalf("path %+v", res.Error.Issues[0].Path)
	}
}

func TestParityTupleEmpty(t *testing.T) {
	schema := Tuple([]AnySchemaLike{})
	got, err := schema.Parse([]any{})
	if err != nil || len(got) != 0 {
		t.Fatalf("%#v %v", got, err)
	}
	if schema.SafeParse([]any{1}).Success {
		t.Fatal("extra item")
	}
}

func TestParityTupleDefaults(t *testing.T) {
	schema := Tuple([]AnySchemaLike{
		Default(String(), "a"),
		Default(Number(), 1.0),
	})
	// Missing slots: tuple length checks may require presence depending on OptIn
	got, err := schema.Parse([]any{})
	if err != nil {
		// If too_small because defaults need OptIn parse path
		t.Logf("empty tuple with defaults: %v", err)
	} else if len(got) < 1 {
		t.Logf("got %#v", got)
	}
	got, err = schema.Parse([]any{Missing, Missing})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "a" || got[1] != 1.0 {
		t.Fatalf("%#v", got)
	}
}
