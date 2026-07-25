package zod

import "testing"

// Ported from v4/classic/tests/array.test.ts

func TestArrayMinMax(t *testing.T) {
	schema := Array(String()).Min(2).Max(2)
	r1 := schema.SafeParse([]any{"asdf"})
	if r1.Success || r1.Error.Issues[0].Code != IssueTooSmall || r1.Error.Issues[0].Origin != "array" {
		t.Fatalf("min: %#v", r1.Error)
	}
	if r1.Error.Issues[0].Message != "Too small: expected array to have >=2 items" {
		t.Fatalf("msg: %q", r1.Error.Issues[0].Message)
	}

	r2 := schema.SafeParse([]any{"a", "b", "c"})
	if r2.Success || r2.Error.Issues[0].Code != IssueTooBig || r2.Error.Issues[0].Origin != "array" {
		t.Fatalf("max: %#v", r2.Error)
	}
}

func TestArrayLength(t *testing.T) {
	schema := Array(String()).Length(2)
	if _, err := schema.Parse([]any{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	r1 := schema.SafeParse([]any{"a"})
	if r1.Success || !r1.Error.Issues[0].Exact {
		t.Fatalf("%#v", r1.Error)
	}
	r2 := schema.SafeParse([]any{"a", "b", "c"})
	if r2.Success || r2.Error.Issues[0].Code != IssueTooBig {
		t.Fatalf("%#v", r2.Error)
	}
}

func TestArrayNonEmpty(t *testing.T) {
	schema := Array(String()).NonEmpty()
	if _, err := schema.Parse([]any{"a"}); err != nil {
		t.Fatal(err)
	}
	if schema.SafeParse([]any{}).Success {
		t.Fatal("empty should fail nonempty")
	}
}

func TestArrayElement(t *testing.T) {
	schema := Array(String())
	el := schema.Element().(*StringSchema)
	if _, err := el.Parse("asdf"); err != nil {
		t.Fatal(err)
	}
	if el.SafeParse(12).Success {
		t.Fatal("element should reject number")
	}
}

func TestArrayContinueDespiteSizeError(t *testing.T) {
	// Port: "continue parsing despite array size error"
	schema := Object(Shape{
		"people": Array(String()).Min(2),
	})
	result := schema.SafeParse(map[string]any{"people": []any{123}})
	if result.Success {
		t.Fatal("expected failure")
	}
	if len(result.Error.Issues) < 2 {
		t.Fatalf("want type + size issues, got %#v", result.Error.Issues)
	}
}

func TestArrayPathPrefix(t *testing.T) {
	schema := Array(String())
	res := schema.SafeParse([]any{"ok", 1})
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if len(iss.Path) != 1 || iss.Path[0] != 1 {
		t.Fatalf("path: %#v", iss.Path)
	}
}

func TestArrayRejectsNonArray(t *testing.T) {
	if Array(String()).SafeParse("nope").Success {
		t.Fatal("string should fail")
	}
	if Array(String()).SafeParse(nil).Success {
		t.Fatal("nil should fail")
	}
}
