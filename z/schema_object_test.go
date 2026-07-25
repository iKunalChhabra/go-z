package z

import (
	"testing"
)

// Ported from v4/classic/tests/object.test.ts (strip/strict/loose/catchall,
// empty object, path prefixes, optional OptIn skipping, Pick/Omit/Extend/Merge/Keyof/Partial).

func TestObjectStripByDefault(t *testing.T) {
	schema := Object(Shape{"points": String()})
	got, err := schema.Parse(map[string]any{"points": "2314", "unknown": "asdf"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["points"] != "2314" {
		t.Fatalf("strip default: got %#v", got)
	}
}

func TestObjectStrict(t *testing.T) {
	schema := Object(Shape{"points": String()}).Strict()
	res := schema.SafeParse(map[string]any{"points": "2314", "unknown": "asdf"})
	if res.Success {
		t.Fatal("expected strict failure")
	}
	if res.Error.Issues[0].Code != IssueUnrecognizedKeys {
		t.Fatalf("want unrecognized_keys, got %#v", res.Error.Issues[0])
	}
	if len(res.Error.Issues[0].Keys) != 1 || res.Error.Issues[0].Keys[0] != "unknown" {
		t.Fatalf("keys: %#v", res.Error.Issues[0].Keys)
	}
}

func TestObjectLoose(t *testing.T) {
	data := map[string]any{"points": "2314", "unknown": "asdf"}
	got, err := Object(Shape{"points": String()}).Loose().Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got["points"] != "2314" || got["unknown"] != "asdf" {
		t.Fatalf("loose: got %#v", got)
	}
}

func TestObjectStripExplicit(t *testing.T) {
	got, err := Object(Shape{"points": String()}).Loose().Strip().Parse(
		map[string]any{"points": "2314", "unknown": "asdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["unknown"]; ok {
		t.Fatalf("strip should drop unknown: %#v", got)
	}
}

func TestObjectCatchall(t *testing.T) {
	schema := Object(Shape{"name": String()}).Catchall(String())
	got, err := schema.Parse(map[string]any{"name": "Foo", "validExtraKey": "61"})
	if err != nil {
		t.Fatal(err)
	}
	if got["name"] != "Foo" || got["validExtraKey"] != "61" {
		t.Fatalf("catchall ok: %#v", got)
	}

	res := schema.SafeParse(map[string]any{"name": "Foo", "validExtraKey": "61", "invalid": 1})
	if res.Success {
		t.Fatal("expected catchall type failure")
	}
	found := false
	for _, iss := range res.Error.Issues {
		if len(iss.Path) > 0 && iss.Path[0] == "invalid" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want path invalid, got %#v", res.Error.Issues)
	}
}

func TestObjectCatchallOverridesStrict(t *testing.T) {
	schema := Object(Shape{"first": String()}).Strict().Catchall(String())
	got, err := schema.Parse(map[string]any{"first": "asdf", "asdf": "1234"})
	if err != nil {
		t.Fatal(err)
	}
	if got["asdf"] != "1234" {
		t.Fatalf("catchall should override strict: %#v", got)
	}
}

func TestObjectEmpty(t *testing.T) {
	schema := Object(Shape{})
	got, err := schema.Parse(map[string]any{})
	if err != nil || len(got) != 0 {
		t.Fatalf("empty: %#v %v", got, err)
	}
	got, err = schema.Parse(map[string]any{"name": "asdf"})
	if err != nil || len(got) != 0 {
		t.Fatalf("empty strips: %#v %v", got, err)
	}
	if schema.SafeParse(nil).Success {
		t.Fatal("null should fail")
	}
	if schema.SafeParse("asdf").Success {
		t.Fatal("string should fail")
	}
}

func TestObjectMissingRequired(t *testing.T) {
	schema := Object(Shape{"name": String()})
	res := schema.SafeParse(map[string]any{})
	if res.Success {
		t.Fatal("expected missing key failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidType || iss.Expected != "string" {
		t.Fatalf("want invalid_type string, got %#v", iss)
	}
	if len(iss.Path) != 1 || iss.Path[0] != "name" {
		t.Fatalf("path: %#v", iss.Path)
	}
	// Message uses received undefined (Missing sentinel).
	if iss.Message != `Invalid input: expected string, received undefined` {
		t.Fatalf("message: %q", iss.Message)
	}
}

func TestObjectOptInSkip(t *testing.T) {
	schema := Object(Shape{
		"id":   String(),
		"nick": Optional(String()),
	})
	got, err := schema.Parse(map[string]any{"id": "a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["nick"]; ok {
		t.Fatalf("absent OptIn key must be omitted: %#v", got)
	}
	got, err = schema.Parse(map[string]any{"id": "a", "nick": "n"})
	if err != nil {
		t.Fatal(err)
	}
	if got["nick"] != "n" {
		t.Fatalf("present OptIn: %#v", got)
	}
}

func TestObjectPartial(t *testing.T) {
	schema := Object(Shape{"a": String(), "b": String()}).Partial()
	got, err := schema.Parse(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("partial all absent: %#v", got)
	}
	got, err = schema.Parse(map[string]any{"a": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "x" {
		t.Fatalf("%#v", got)
	}
	// Required restores.
	req := schema.Required()
	if req.SafeParse(map[string]any{"a": "x"}).Success {
		t.Fatal("Required should demand b")
	}
}

func TestObjectPickOmitExtendMergeKeyof(t *testing.T) {
	base := Object(Shape{"a": String(), "b": String(), "c": String()})
	picked, err := base.Pick("a", "c").Parse(map[string]any{"a": "1", "b": "2", "c": "3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(picked) != 2 || picked["a"] != "1" || picked["c"] != "3" {
		t.Fatalf("pick: %#v", picked)
	}

	omitted, err := base.Omit("b").Parse(map[string]any{"a": "1", "b": "2", "c": "3"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := omitted["b"]; ok || len(omitted) != 2 {
		t.Fatalf("omit: %#v", omitted)
	}

	ext, err := base.Extend(Shape{"d": String()}).Parse(map[string]any{
		"a": "1", "b": "2", "c": "3", "d": "4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ext["d"] != "4" {
		t.Fatalf("extend: %#v", ext)
	}

	other := Object(Shape{"b": String().Min(2), "e": String()}).Strict()
	merged := base.Merge(other)
	// Merge adopts other's strict mode.
	res := merged.SafeParse(map[string]any{"a": "1", "b": "xy", "c": "3", "e": "5", "x": "no"})
	if res.Success {
		t.Fatal("merged strict should reject x")
	}

	k := base.Keyof()
	if _, err := k.Parse("a"); err != nil {
		t.Fatal(err)
	}
	if k.SafeParse("z").Success {
		t.Fatal("keyof should reject z")
	}
}

func TestObjectNestedPathPrefix(t *testing.T) {
	schema := Object(Shape{
		"people": Array(String()).Min(2),
	})
	res := schema.SafeParse(map[string]any{"people": []any{123}})
	if res.Success {
		t.Fatal("expected failure")
	}
	// Element type error at people.0 and too_small at people.
	var sawPath, sawSize bool
	for _, iss := range res.Error.Issues {
		if iss.Code == IssueInvalidType && len(iss.Path) == 2 && iss.Path[0] == "people" && iss.Path[1] == 0 {
			sawPath = true
		}
		if iss.Code == IssueTooSmall && len(iss.Path) == 1 && iss.Path[0] == "people" && iss.Origin == "array" {
			sawSize = true
		}
	}
	if !sawPath || !sawSize {
		t.Fatalf("issues: %#v", res.Error.Issues)
	}
}

func TestObjectUnknownRejectsNonObject(t *testing.T) {
	schema := Object(Shape{"a": String()})
	if schema.SafeParse(35).Success {
		t.Fatal("number should fail")
	}
	if schema.SafeParse([]any{}).Success {
		t.Fatal("array should fail")
	}
}
