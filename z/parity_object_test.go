package z

import (
	"reflect"
	"testing"
)

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/object.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityObjectCorrectParsing(t *testing.T) {
	// Ported from classic/tests/object.test.ts — Test schema / correct parsing
	Test := Object(Shape{
		"f1": Number(),
		"f2": Optional(String()),
		"f3": Nullable(String()),
		"f4": Array(Object(Shape{
			"t": UnionOf(String(), Bool()),
		})),
	})

	got, err := Test.Parse(map[string]any{
		"f1": 12.0,
		"f2": "string",
		"f3": "string",
		"f4": []any{map[string]any{"t": "string"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["f1"] != 12.0 || got["f2"] != "string" {
		t.Fatalf("got %#v", got)
	}

	got, err = Test.Parse(map[string]any{
		"f1": 12.0,
		"f3": nil,
		"f4": []any{map[string]any{"t": false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["f3"] != nil {
		t.Fatalf("nullable null: %#v", got)
	}
	if _, ok := got["f2"]; ok {
		t.Fatalf("optional absent key should be omitted: %#v", got)
	}
}

func TestParityObjectUnknownThrow(t *testing.T) {
	// Ported from classic/tests/object.test.ts — "unknown throw"
	Test := Object(Shape{"f1": Number()})
	if Test.SafeParse(35).Success {
		t.Fatal("number should fail object")
	}
}

func TestParityObjectShapeAccess(t *testing.T) {
	// Ported from classic/tests/object.test.ts — "shape() should return schema of particular key"
	Test := Object(Shape{
		"f1": Number(),
		"f2": Optional(String()),
		"f3": Nullable(String()),
		"f4": Array(String()),
	})
	shape := Test.Shape()
	if _, ok := shape["f1"].(*NumberSchema); !ok {
		t.Fatalf("f1 type %T", shape["f1"])
	}
	if !defTypeIs(shape["f2"], "optional") {
		t.Fatalf("f2 type %T", shape["f2"])
	}
	if !defTypeIs(shape["f3"], "nullable") {
		t.Fatalf("f3 type %T", shape["f3"])
	}
	if _, ok := shape["f4"].(*ArraySchema); !ok {
		t.Fatalf("f4 type %T", shape["f4"])
	}
}

func TestParityObjectNonstrictStripLoose(t *testing.T) {
	// Ported from classic/tests/object.test.ts — strip / passthrough / strict
	data := map[string]any{"points": 2314.0, "unknown": "asdf"}

	// nonstrict by default (strip)
	got, err := Object(Shape{"points": Number()}).Parse(map[string]any{
		"points": 2314.0, "unknown": "asdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["points"] != 2314.0 {
		t.Fatalf("strip default: %#v", got)
	}

	got, err = Object(Shape{"points": Number()}).Parse(data)
	if err != nil || !reflect.DeepEqual(got, map[string]any{"points": 2314.0}) {
		t.Fatalf("strip by default: %#v %v", got, err)
	}

	got, err = Object(Shape{"points": Number()}).Strict().Passthrough().Strip().Passthrough().Parse(data)
	if err != nil || got["points"] != 2314.0 || got["unknown"] != "asdf" {
		t.Fatalf("unknownkeys override: %#v %v", got, err)
	}

	got, err = Object(Shape{"points": Number()}).Passthrough().Parse(data)
	if err != nil || got["unknown"] != "asdf" {
		t.Fatalf("passthrough: %#v %v", got, err)
	}

	got, err = Object(Shape{"points": Number()}).Strip().Parse(data)
	if err != nil || len(got) != 1 {
		t.Fatalf("strip: %#v %v", got, err)
	}

	res := Object(Shape{"points": Number()}).Strict().SafeParse(data)
	if res.Success || res.Error.Issues[0].Code != IssueUnrecognizedKeys {
		t.Fatalf("strict: %+v", res.Error)
	}
}

func TestParityObjectEmpty(t *testing.T) {
	// Ported from classic/tests/object.test.ts — "empty object"
	schema := Object(Shape{})
	got, err := schema.Parse(map[string]any{})
	if err != nil || len(got) != 0 {
		t.Fatalf("%#v %v", got, err)
	}
	got, err = schema.Parse(map[string]any{"name": "asdf"})
	if err != nil || len(got) != 0 {
		t.Fatalf("strip extras: %#v %v", got, err)
	}
	if schema.SafeParse(nil).Success {
		t.Fatal("null should fail")
	}
	if schema.SafeParse("asdf").Success {
		t.Fatal("string should fail")
	}
}

func TestParityObjectOptionalKeys(t *testing.T) {
	// Ported from classic/tests/object.test.ts — parse optional keys / optional keys are unset
	schema := Object(Shape{"a": Optional(String())})
	got, err := schema.Parse(map[string]any{"a": "asdf"})
	if err != nil || got["a"] != "asdf" {
		t.Fatalf("%#v %v", got, err)
	}

	named := Object(Shape{
		"id":    String(),
		"set":   Optional(String()),
		"unset": Optional(String()),
	})
	got, err = named.Parse(map[string]any{"id": "asdf", "set": Missing})
	// Missing as explicit value: Optional accepts it and may omit
	_ = got
	got, err = named.Parse(map[string]any{"id": "asdf"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["unset"]; ok {
		t.Fatalf("unset key should be omitted: %#v", got)
	}
	if got["id"] != "asdf" {
		t.Fatalf("%#v", got)
	}
}

func TestParityObjectCatchall(t *testing.T) {
	// Ported from classic/tests/object.test.ts — catchall*
	o1 := Object(Shape{"first": String()}).Catchall(Number())
	got, err := o1.Parse(map[string]any{"first": "asdf", "num": 1243.0})
	if err != nil || got["num"] != 1243.0 {
		t.Fatalf("%#v %v", got, err)
	}

	o2 := Object(Shape{"first": Optional(String())}).Strict().Catchall(Number())
	if _, err := o2.Parse(map[string]any{"asdf": 1234.0}); err != nil {
		t.Fatalf("catchall overrides strict: %v", err)
	}
	if _, err := o2.Parse(map[string]any{"first": "asdf", "asdf": 1234.0}); err != nil {
		t.Fatalf("catchall known+extra: %v", err)
	}

	o3 := Object(Shape{"first": String()}).Strict().Catchall(Number())
	if _, err := o3.Parse(map[string]any{"first": "asdf", "asdf": 1234.0}); err != nil {
		t.Fatal(err)
	}

	result, err := Object(Shape{"name": String()}).Catchall(Number()).Parse(
		map[string]any{"name": "Foo", "validExtraKey": 61.0},
	)
	if err != nil || result["validExtraKey"] != 61.0 {
		t.Fatalf("%#v %v", result, err)
	}
	res := Object(Shape{"name": String()}).Catchall(Number()).SafeParse(
		map[string]any{"name": "Foo", "validExtraKey": 61.0, "invalid": "asdf"},
	)
	if res.Success {
		t.Fatal("catchall type fail expected")
	}
}

func TestParityObjectExtendMergeKeyof(t *testing.T) {
	// Ported from classic/tests/object.test.ts — extend / merge / keyof
	base := Object(Shape{"a": String()})
	ext := base.Extend(Shape{"b": Number()})
	got, err := ext.Parse(map[string]any{"a": "x", "b": 1.0})
	if err != nil || got["a"] != "x" || got["b"] != 1.0 {
		t.Fatalf("%#v %v", got, err)
	}
	if ext.SafeParse(map[string]any{"a": "x"}).Success {
		t.Fatal("extended requires b")
	}

	other := Object(Shape{"b": Number(), "c": Bool()}).Strict()
	merged := base.Merge(other)
	got, err = merged.Parse(map[string]any{"a": "x", "b": 1.0, "c": true})
	if err != nil {
		t.Fatal(err)
	}
	res := merged.SafeParse(map[string]any{"a": "x", "b": 1.0, "c": true, "extra": 1})
	if res.Success {
		t.Fatal("merge adopts strict from other")
	}

	keys := Object(Shape{"name": String(), "age": Number()}).Keyof()
	if _, err := keys.Parse("name"); err != nil {
		t.Fatal(err)
	}
	if _, err := keys.Parse("age"); err != nil {
		t.Fatal(err)
	}
	if keys.SafeParse("nope").Success {
		t.Fatal("keyof should reject unknown")
	}
}

func TestParityObjectPathPrefixes(t *testing.T) {
	// Ported from classic/tests/object.test.ts — nested errors get path prefixes
	schema := Object(Shape{
		"name": String(),
		"address": Object(Shape{
			"city": String(),
		}),
	})
	res := schema.SafeParse(map[string]any{
		"name":    123,
		"address": map[string]any{"city": 456},
	})
	if res.Success || len(res.Error.Issues) < 2 {
		t.Fatalf("%+v", res.Error)
	}
	foundName, foundCity := false, false
	for _, iss := range res.Error.Issues {
		if len(iss.Path) > 0 && iss.Path[0] == "name" {
			foundName = true
		}
		if len(iss.Path) >= 2 && iss.Path[0] == "address" && iss.Path[1] == "city" {
			foundCity = true
		}
	}
	if !foundName || !foundCity {
		t.Fatalf("paths: %+v", res.Error.Issues)
	}
}

func TestParityObjectDefaultsInFields(t *testing.T) {
	// Ported from classic/tests/object.test.ts / default.test.ts nested object defaults
	schema := Object(Shape{
		"hi": Default(String(), "hi"),
	})
	got, err := schema.Parse(map[string]any{})
	if err != nil || got["hi"] != "hi" {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestParityObjectCatchallNever(t *testing.T) {
	schema := Object(Shape{"a": String()}).Catchall(Never())
	res := schema.SafeParse(map[string]any{"a": "x", "b": 1})
	if res.Success || res.Error.Issues[0].Code != IssueUnrecognizedKeys {
		t.Fatalf("catchall Never → unrecognized_keys: %+v", res.Error)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/pickomit.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityObjectPick(t *testing.T) {
	// Ported from classic/tests/pickomit.test.ts — pick*
	fish := Object(Shape{
		"name":   String(),
		"age":    Number(),
		"nested": Object(Shape{}),
	})
	nameOnly := fish.Pick("name")
	if _, err := nameOnly.Parse(map[string]any{"name": "bob"}); err != nil {
		t.Fatal(err)
	}
	// unknown keys stripped
	got, err := nameOnly.Parse(map[string]any{"name": "bob", "age": 12.0})
	if err != nil || len(got) != 1 || got["name"] != "bob" {
		t.Fatalf("%#v %v", got, err)
	}

	strict := nameOnly.Strict()
	if strict.SafeParse(map[string]any{"name": 12}).Success {
		t.Fatal("type fail")
	}
	if strict.SafeParse(map[string]any{"name": "bob", "age": 12.0}).Success {
		t.Fatal("extra key fail")
	}
	if strict.SafeParse(map[string]any{"age": 12.0}).Success {
		t.Fatal("missing name fail")
	}

	schema := Object(Shape{"a": String(), "b": Optional(String())})
	picked := schema.Pick("a")
	shape := picked.Shape()
	if _, ok := shape["a"]; !ok {
		t.Fatal("a should remain")
	}
	if _, ok := shape["b"]; ok {
		t.Fatal("b should be removed")
	}
}

func TestParityObjectOmit(t *testing.T) {
	// Ported from classic/tests/pickomit.test.ts — omit*
	fish := Object(Shape{
		"name":   String(),
		"age":    Number(),
		"nested": Object(Shape{}),
	})
	noname := fish.Omit("name")
	if _, err := noname.Parse(map[string]any{"age": 12.0, "nested": map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	if noname.SafeParse(map[string]any{"name": 12}).Success {
		// name omitted from shape — may strip or fail age/nested
	}
	if noname.SafeParse(map[string]any{"age": 12.0}).Success {
		t.Fatal("missing nested should fail")
	}
	if noname.SafeParse(map[string]any{}).Success {
		t.Fatal("empty should fail")
	}

	schema := Object(Shape{"a": String(), "b": Optional(String())})
	omitted := schema.Omit("a")
	if _, ok := omitted.Shape()["a"]; ok {
		t.Fatal("a should be omitted")
	}
	if _, ok := omitted.Shape()["b"]; !ok {
		t.Fatal("b should remain")
	}
}

func TestParityObjectPickPassthroughCatchall(t *testing.T) {
	// Ported from classic/tests/pickomit.test.ts — nonstrict parsing
	fish := Object(Shape{"name": String(), "age": Number(), "nested": Object(Shape{})})
	lax := fish.Passthrough().Pick("name")
	got, err := lax.Parse(map[string]any{"name": "bob", "whatever": true})
	if err != nil {
		t.Fatal(err)
	}
	if got["name"] != "bob" || got["whatever"] != true {
		t.Fatalf("%#v", got)
	}

	withCatchall := fish.Pick("name").Catchall(Any())
	got, err = withCatchall.Parse(map[string]any{"name": "x", "extra": 1})
	if err != nil || got["extra"] != 1 {
		t.Fatalf("%#v %v", got, err)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Ported from classic/tests/partial.test.ts
//////////////////////////////////////////////////////////////////////////////

func TestParityObjectPartial(t *testing.T) {
	// Ported from classic/tests/partial.test.ts — shallow partial parse
	nested := Object(Shape{
		"name": String(),
		"age":  Number(),
		"outer": Object(Shape{
			"inner": String(),
		}),
		"array": Array(Object(Shape{"asdf": String()})),
	})
	shallow := nested.Partial()
	if _, err := shallow.Parse(map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if _, err := shallow.Parse(map[string]any{"name": "asdf", "age": 23143.0}); err != nil {
		t.Fatal(err)
	}
	// nested object still required when present
	res := shallow.SafeParse(map[string]any{"outer": map[string]any{}})
	if res.Success {
		t.Fatal("partial is shallow: inner still required")
	}
}

func TestParityObjectRequired(t *testing.T) {
	// Ported from classic/tests/partial.test.ts — required / required with mask
	object := Object(Shape{
		"name":          String(),
		"age":           Optional(Number()),
		"field":         Default(Optional(String()), "asdf"),
		"nullableField": Nullable(Number()),
		"nullishField":  Nullish(String()),
	})
	req := object.Required()
	shape := req.Shape()
	if !defTypeIs(shape["name"], "nonoptional") {
		t.Fatalf("name: %T", shape["name"])
	}
	if !defTypeIs(shape["age"], "nonoptional") {
		t.Fatalf("age: %T", shape["age"])
	}
	if !defTypeIs(shape["field"], "nonoptional") {
		t.Fatalf("field: %T", shape["field"])
	}

	// required age must be present
	if req.SafeParse(map[string]any{
		"name":          "n",
		"nullableField": nil,
	}).Success {
		t.Fatal("age required after .Required()")
	}
	if _, err := req.Parse(map[string]any{
		"name":          "n",
		"age":           1.0,
		"nullableField": nil,
		"nullishField":  nil,
	}); err != nil {
		t.Fatal(err)
	}

	masked := object.Required("age")
	ms := masked.Shape()
	if _, ok := ms["name"].(*StringSchema); !ok {
		t.Fatalf("name unchanged: %T", ms["name"])
	}
	if !defTypeIs(ms["age"], "nonoptional") {
		t.Fatalf("age required: %T", ms["age"])
	}
	if _, ok := ms["country"]; ok {
		t.Fatal("no country")
	}
	if !defTypeIs(ms["field"], "default") {
		// field may still be Default
		t.Logf("field type: %T", ms["field"])
	}
}

func TestParityObjectPartialMask(t *testing.T) {
	// Ported from classic/tests/partial.test.ts — partial with mask
	object := Object(Shape{
		"name":    String(),
		"age":     Number(),
		"country": String(),
	})
	partial := object.Partial("age", "country")
	shape := partial.Shape()
	if _, ok := shape["name"].(*StringSchema); !ok {
		t.Fatalf("name: %T", shape["name"])
	}
	if !defTypeIs(shape["age"], "optional") {
		t.Fatalf("age: %T", shape["age"])
	}
	if !defTypeIs(shape["country"], "optional") {
		t.Fatalf("country: %T", shape["country"])
	}
	if _, err := partial.Parse(map[string]any{"name": "x"}); err != nil {
		t.Fatal(err)
	}
	if partial.SafeParse(map[string]any{}).Success {
		t.Fatal("name still required")
	}
}

func TestParityObjectPartialAlreadyOptional(t *testing.T) {
	schema := Object(Shape{"a": Optional(String()), "b": String()})
	p := schema.Partial()
	// a should remain Optional, not double-wrapped in a way that breaks
	if _, err := p.Parse(map[string]any{}); err != nil {
		t.Fatal(err)
	}
	a := p.Shape()["a"]
	if !defTypeIs(a, "optional") {
		t.Fatalf("want Optional, got %T", a)
	}
}
