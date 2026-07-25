package z

import "testing"

// Ported from v4/classic/tests/set.test.ts (adapted: Go accepts []any with uniqueness).

func TestSetValidParse(t *testing.T) {
	schema := Set(String())
	got, err := schema.Parse([]any{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("%#v", got)
	}
}

func TestSetUniqueness(t *testing.T) {
	schema := Set(String())
	got, err := schema.Parse([]any{"a", "a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("uniqueness: %#v", got)
	}
}

func TestSetSizeMethods(t *testing.T) {
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

func TestSetInvalidElement(t *testing.T) {
	schema := Set(String())
	res := schema.SafeParse([]any{"ok", 12})
	if res.Success {
		t.Fatal("expected failure")
	}
	// Set element issues are not path-prefixed.
	if len(res.Error.Issues[0].Path) != 0 {
		t.Fatalf("path should be empty: %#v", res.Error.Issues[0].Path)
	}
}

func TestSetFromMapKeys(t *testing.T) {
	schema := Set(String())
	got, err := schema.Parse(map[string]struct{}{"a": {}, "b": {}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
}

func TestSetRejectsNonSet(t *testing.T) {
	// Plain maps that aren't set-shaped fail (map[string]any is not a set).
	if Set(String()).SafeParse(map[string]any{"a": 1}).Success {
		t.Fatal("map[string]any should fail")
	}
	if Set(String()).SafeParse("nope").Success {
		t.Fatal("string should fail")
	}
}
