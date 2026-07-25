package z

import (
	"reflect"
	"testing"
)

// Parity ports from:
//   packages/zod/src/v4/classic/tests/record.test.ts
//   packages/zod/src/v4/classic/tests/map.test.ts
//   packages/zod/src/v4/classic/tests/set.test.ts

func TestParityRecordBasic(t *testing.T) {
	schema := Record(String(), String())
	got, err := schema.Parse(map[string]any{"a": "1", "b": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "1" || got["b"] != "2" {
		t.Fatalf("%#v", got)
	}
}

func TestParityRecordValueTypeError(t *testing.T) {
	schema := Record(String(), Number())
	cases := []struct {
		name string
		in   any
		ok   bool
	}{
		{"good", map[string]any{"x": 1.0}, true},
		{"bad value", map[string]any{"x": "no"}, false},
		{"non object", []any{1}, false},
		{"string", "nope", false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := schema.SafeParse(tc.in)
			if res.Success != tc.ok {
				t.Fatalf("success=%v err=%v", res.Success, res.Error)
			}
		})
	}
}

func TestParityRecordEnumKeys(t *testing.T) {
	// Port: enum key exhaustiveness
	schema := Record(Enum("ok", "yes"), String())
	got, err := schema.Parse(map[string]any{"ok": "a", "yes": "b"})
	if err != nil {
		t.Fatal(err)
	}
	if got["ok"] != "a" || got["yes"] != "b" {
		t.Fatalf("%#v", got)
	}
	res := schema.SafeParse(map[string]any{"ok": "a", "no": "x"})
	if res.Success {
		t.Fatal("expected failure for extra key")
	}
	foundUnrecognized := false
	for _, iss := range res.Error.Issues {
		if iss.Code == IssueUnrecognizedKeys {
			foundUnrecognized = true
		}
	}
	if !foundUnrecognized {
		t.Fatalf("want unrecognized_keys among issues, got %+v", res.Error.Issues)
	}
	res = schema.SafeParse(map[string]any{"ok": "a"})
	if res.Success {
		t.Fatal("missing enum key yes should fail")
	}
}

func TestParityRecordOpenKeys(t *testing.T) {
	schema := Record(String(), Number())
	got, err := schema.Parse(map[string]any{"any": 1.0, "key": 2.0})
	if err != nil {
		t.Fatal(err)
	}
	if got["any"] != 1.0 || got["key"] != 2.0 {
		t.Fatalf("%#v", got)
	}
}

func TestParityRecordNested(t *testing.T) {
	schema := Record(String(), Record(String(), Number()))
	got, err := schema.Parse(map[string]any{
		"outer": map[string]any{"inner": 3.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	inner := got["outer"].(map[string]any)
	if inner["inner"] != 3.0 {
		t.Fatalf("%#v", got)
	}
}

func TestParityMapValidParse(t *testing.T) {
	schema := Map(String(), String())
	cases := []any{
		map[any]any{"first": "foo", "second": "bar"},
		map[string]any{"a": "b"},
	}
	for _, in := range cases {
		got, err := schema.Parse(in)
		if err != nil {
			t.Fatalf("Parse(%T): %v", in, err)
		}
		if len(got) == 0 {
			t.Fatalf("empty result for %T", in)
		}
	}
}

func TestParityMapSizeMethods(t *testing.T) {
	minTwo := Map(String(), String()).Min(2)
	maxTwo := Map(String(), String()).Max(2)
	justTwo := Map(String(), String()).Size(2)
	nonEmpty := Map(String(), String()).NonEmpty()

	two := map[any]any{"a": "b", "c": "d"}
	one := map[any]any{"a": "b"}
	three := map[any]any{"a": "b", "c": "d", "e": "f"}

	if _, err := minTwo.Parse(two); err != nil {
		t.Fatal(err)
	}
	if minTwo.SafeParse(one).Success {
		t.Fatal("min")
	}
	if maxTwo.SafeParse(three).Success {
		t.Fatal("max")
	}
	if _, err := justTwo.Parse(two); err != nil {
		t.Fatal(err)
	}
	if nonEmpty.SafeParse(map[any]any{}).Success {
		t.Fatal("nonempty")
	}
	iss := minTwo.SafeParse(one).Error.Issues[0]
	if iss.Origin != "map" || iss.Code != IssueTooSmall {
		t.Fatalf("%#v", iss)
	}
}

func TestParityMapInvalidKeyAndValue(t *testing.T) {
	schema := Map(String(), String())
	res := schema.SafeParse(map[any]any{1: "foo"})
	if res.Success {
		t.Fatal("bad key")
	}
	res = schema.SafeParse(map[any]any{"ok": 12})
	if res.Success {
		t.Fatal("bad value")
	}
	if schema.SafeParse([]any{}).Success || schema.SafeParse("nope").Success {
		t.Fatal("non-map should fail")
	}
}

func TestParityMapKeySchema(t *testing.T) {
	schema := Map(Number(), String())
	got, err := schema.Parse(map[any]any{1.0: "a", 2.0: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if got[1.0] != "a" {
		t.Fatalf("%#v", got)
	}
	if schema.SafeParse(map[any]any{"x": "a"}).Success {
		t.Fatal("string key should fail number key schema")
	}
}

func TestParitySetValidParse(t *testing.T) {
	schema := Set(String())
	got, err := schema.Parse([]any{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []any{"first", "second"}) {
		t.Fatalf("%#v", got)
	}
}

func TestParitySetUniqueness(t *testing.T) {
	schema := Set(String())
	got, err := schema.Parse([]any{"a", "a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Fatalf("uniqueness: %#v", got)
	}
}

func TestParitySetSizeMethods(t *testing.T) {
	minTwo := Set(String()).Min(2)
	maxTwo := Set(String()).Max(2)
	justTwo := Set(String()).Size(2)
	nonEmpty := Set(String()).NonEmpty()

	if _, err := minTwo.Parse([]any{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if minTwo.SafeParse([]any{"just_one"}).Success {
		t.Fatal("min")
	}
	if maxTwo.SafeParse([]any{"one", "two", "three"}).Success {
		t.Fatal("max")
	}
	if _, err := justTwo.Parse([]any{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if nonEmpty.SafeParse([]any{}).Success {
		t.Fatal("nonempty")
	}
	iss := minTwo.SafeParse([]any{"x"}).Error.Issues[0]
	if iss.Origin != "set" || iss.Code != IssueTooSmall {
		t.Fatalf("%#v", iss)
	}
}

func TestParitySetInvalidElement(t *testing.T) {
	schema := Set(String())
	res := schema.SafeParse([]any{"ok", 12})
	if res.Success {
		t.Fatal("expected failure")
	}
	if len(res.Error.Issues[0].Path) != 0 {
		t.Fatalf("set element path should be empty: %#v", res.Error.Issues[0].Path)
	}
}

func TestParitySetFromMapKeys(t *testing.T) {
	schema := Set(String())
	got, err := schema.Parse(map[string]struct{}{"a": {}, "b": {}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
}

func TestParitySetTable(t *testing.T) {
	schema := Set(Number())
	cases := []struct {
		name string
		in   any
		ok   bool
		len  int
	}{
		{"ok", []any{1.0, 2.0}, true, 2},
		{"dedupe", []any{1.0, 1.0, 2.0}, true, 2},
		{"bad elem", []any{1.0, "x"}, false, 0},
		{"string", "no", false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := schema.SafeParse(tc.in)
			if res.Success != tc.ok {
				t.Fatalf("success=%v err=%v", res.Success, res.Error)
			}
			if tc.ok && len(res.Data) != tc.len {
				t.Fatalf("len=%d want %d data=%#v", len(res.Data), tc.len, res.Data)
			}
		})
	}
}

func TestParityRecordMapSetBrandUnsupported(t *testing.T) {
	t.Skip("brand not supported in go-z")
}
