package zod

import "testing"

// Ported from v4/classic/tests/map.test.ts (adapted to Go map[any]any / map[string]any).

func TestMapValidParse(t *testing.T) {
	schema := Map(String(), String())
	got, err := schema.Parse(map[any]any{"first": "foo", "second": "bar"})
	if err != nil {
		t.Fatal(err)
	}
	if got["first"] != "foo" || got["second"] != "bar" {
		t.Fatalf("%#v", got)
	}
	// Also accept map[string]any.
	got, err = schema.Parse(map[string]any{"a": "b"})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "b" {
		t.Fatalf("%#v", got)
	}
}

func TestMapSizeMethods(t *testing.T) {
	minTwo := Map(String(), String()).Min(2)
	maxTwo := Map(String(), String()).Max(2)
	justTwo := Map(String(), String()).Size(2)
	nonEmpty := Map(String(), String()).NonEmpty()

	if _, err := minTwo.Parse(map[any]any{"a": "b", "c": "d"}); err != nil {
		t.Fatal(err)
	}
	if minTwo.SafeParse(map[any]any{"a": "b"}).Success {
		t.Fatal("min should fail")
	}
	if maxTwo.SafeParse(map[any]any{"a": "b", "c": "d", "e": "f"}).Success {
		t.Fatal("max should fail")
	}
	if _, err := justTwo.Parse(map[any]any{"a": "b", "c": "d"}); err != nil {
		t.Fatal(err)
	}
	if nonEmpty.SafeParse(map[any]any{}).Success {
		t.Fatal("nonempty")
	}
	iss := minTwo.SafeParse(map[any]any{"a": "b"}).Error.Issues[0]
	if iss.Origin != "map" || iss.Code != IssueTooSmall {
		t.Fatalf("%#v", iss)
	}
}

func TestMapInvalidKeyAndValue(t *testing.T) {
	schema := Map(String(), String())
	res := schema.SafeParse(map[any]any{1: "foo"})
	if res.Success {
		t.Fatal("bad key")
	}
	// int key is a property key → path-prefixed issues
	found := false
	for _, iss := range res.Error.Issues {
		if len(iss.Path) > 0 && iss.Path[0] == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("%#v", res.Error.Issues)
	}

	res = schema.SafeParse(map[any]any{"ok": 12})
	if res.Success {
		t.Fatal("bad value")
	}
}

func TestMapRejectsNonMap(t *testing.T) {
	if Map(String(), String()).SafeParse([]any{}).Success {
		t.Fatal("array should fail")
	}
	if Map(String(), String()).SafeParse("nope").Success {
		t.Fatal("string should fail")
	}
}
