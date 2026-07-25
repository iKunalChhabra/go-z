package z

import (
	"testing"
)

// Ports cases from v4/classic/tests/intersection.test.ts.

// Port: "object intersection"
func TestIntersectionObject(t *testing.T) {
	a := Object(Shape{"a": String()})
	b := Object(Shape{"b": String()})
	c := Intersection(a, b)
	data := map[string]any{"a": "foo", "b": "foo"}
	got, err := c.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["a"] != "foo" || m["b"] != "foo" {
		t.Fatalf("got %#v", got)
	}
	if _, err := c.Parse(map[string]any{"a": "foo"}); err == nil {
		t.Fatal("expected failure for missing b")
	}
}

// Port: "object intersection: strict + strict"
func TestIntersectionStrictStrict(t *testing.T) {
	a := Object(Shape{"a": String()}).Strict()
	b := Object(Shape{"b": String()}).Strict()
	c := Intersection(a, b)
	got, err := c.Parse(map[string]any{"a": "foo", "b": "bar"})
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["a"] != "foo" || got.(map[string]any)["b"] != "bar" {
		t.Fatalf("got %#v", got)
	}
	res := c.SafeParse(map[string]any{"a": "foo", "b": "bar", "c": "extra"})
	if res.Success {
		t.Fatal("expected unrecognized_keys")
	}
	found := false
	for _, iss := range res.Error.Issues {
		if iss.Code == IssueUnrecognizedKeys {
			found = true
			if len(iss.Keys) != 1 || iss.Keys[0] != "c" {
				t.Fatalf("keys = %v", iss.Keys)
			}
		}
	}
	if !found {
		t.Fatalf("want unrecognized_keys, got %+v", res.Error.Issues)
	}
}

// Port: "deep intersection"
func TestIntersectionDeep(t *testing.T) {
	animal := Object(Shape{
		"properties": Object(Shape{"is_animal": Bool()}),
	})
	catExtra := Object(Shape{
		"properties": Object(Shape{"jumped": Bool()}),
	})
	cat := Intersection(catExtra, animal)
	res := cat.SafeParse(map[string]any{
		"properties": map[string]any{"is_animal": true, "jumped": true},
	})
	if !res.Success {
		t.Fatalf("err = %v", res.Error)
	}
	props := res.Data.(map[string]any)["properties"].(map[string]any)
	if props["is_animal"] != true || props["jumped"] != true {
		t.Fatalf("props = %#v", props)
	}
}

// Port: "deep intersection of arrays"
func TestIntersectionDeepArrays(t *testing.T) {
	left := Object(Shape{
		"posts": Array(Object(Shape{"post_id": Number()})),
	})
	right := Object(Shape{
		"posts": Array(Object(Shape{"title": String()})),
	})
	reg := Intersection(left, right)
	posts := []any{
		map[string]any{"post_id": 1.0, "title": "Novels"},
		map[string]any{"post_id": 2.0, "title": "Fairy tales"},
	}
	got, err := reg.Parse(map[string]any{"posts": posts})
	if err != nil {
		t.Fatal(err)
	}
	out := got.(map[string]any)["posts"].([]any)
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	p0 := out[0].(map[string]any)
	if p0["post_id"] != 1.0 || p0["title"] != "Novels" {
		t.Fatalf("p0 = %#v", p0)
	}
}

// Port: "invalid intersection types"
func TestIntersectionUnmergeable(t *testing.T) {
	left := Number()
	right := Transform(Number(), func(v any, _ *RefinementCtx) (any, error) {
		f, _ := ToFloat(v)
		return f + 1, nil
	})
	schema := Intersection(left, right)
	res := schema.SafeParse(1234.0)
	if res.Success {
		t.Fatal("expected unmergeable failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueCustom || iss.Message != "Unmergeable intersection results" {
		t.Fatalf("got %+v", iss)
	}
}

// Port: "invalid array merge (incompatible lengths)"
func TestIntersectionUnmergeableArrayLength(t *testing.T) {
	left := Array(String())
	right := Transform(Array(String()), func(v any, _ *RefinementCtx) (any, error) {
		arr := v.([]any)
		return append(append([]any{}, arr...), "asdf"), nil
	})
	schema := Intersection(left, right)
	res := schema.SafeParse([]any{"asdf", "qwer"})
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Error.Issues[0].Message != "Unmergeable intersection results" {
		t.Fatalf("got %+v", res.Error.Issues[0])
	}
}

// Port: "invalid object merge"
func TestIntersectionUnmergeableObjectKey(t *testing.T) {
	cat := Object(Shape{
		"phrase": Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
			return v.(string) + " Meow", nil
		}),
	})
	dog := Object(Shape{
		"phrase": Transform(String(), func(v any, _ *RefinementCtx) (any, error) {
			return v.(string) + " Woof", nil
		}),
	})
	schema := Intersection(cat, dog)
	res := schema.SafeParse(map[string]any{"phrase": "Hello"})
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Message != "Unmergeable intersection results" {
		t.Fatalf("got %+v", iss)
	}
	if len(iss.Path) != 1 || iss.Path[0] != "phrase" {
		t.Fatalf("path = %v", iss.Path)
	}
}

func TestIntersectionSamePrimitive(t *testing.T) {
	schema := Intersection(String(), String())
	got, err := schema.Parse("hello")
	if err != nil || got != "hello" {
		t.Fatalf("got %v, %v", got, err)
	}
}

// Each side of an intersection only knows its own shape, so a strict object
// flags the other side's keys. The intersection's recognized set is the union of
// both shapes, which keeps two strict objects usable without letting a key that
// neither side declares slip through a strict side.
func TestIntersectionUnrecognizedKeys(t *testing.T) {
	l := Object(Shape{"a": String()}).Strict()
	r := Object(Shape{"b": String()}).Strict()
	both := Intersection(l, r)

	if _, err := both.Parse(map[string]any{"a": "x", "b": "y"}); err != nil {
		t.Fatalf("keys declared by either side must be accepted: %v", err)
	}

	res := both.SafeParse(map[string]any{"a": "x", "b": "y", "c": "z"})
	if res.Success {
		t.Fatal("a key neither side declares must be rejected")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueUnrecognizedKeys || len(iss.Keys) != 1 || iss.Keys[0] != "c" {
		t.Fatalf("got %+v", iss)
	}

	// A strict side still rejects unknown keys when the other side is loose:
	// the loose side never flags them, so the "both sides flagged it" rule alone
	// would have accepted this.
	mixed := Intersection(Object(Shape{"a": String()}).Strict(), Object(Shape{"b": String()}).Loose())
	if mixed.SafeParse(map[string]any{"a": "x", "b": "y", "c": "z"}).Success {
		t.Fatal("the strict side must still reject an undeclared key")
	}
	if _, err := mixed.Parse(map[string]any{"a": "x", "b": "y"}); err != nil {
		t.Fatalf("declared keys must pass: %v", err)
	}

	// Non-object sides have no shape to union, so the fallback applies: only
	// keys both sides flag are reported.
	withUnion := Intersection(
		UnionOf(Object(Shape{"a": String()}).Strict(), Object(Shape{"a": String(), "b": String()}).Strict()),
		Object(Shape{"b": String()}).Strict(),
	)
	if _, err := withUnion.Parse(map[string]any{"a": "x", "b": "y"}); err != nil {
		t.Fatalf("union side: %v", err)
	}
}
