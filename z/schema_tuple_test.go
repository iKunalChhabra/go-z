package z

import "testing"

// Ported from v4/classic/tests/tuple.test.ts

func TestTupleSuccess(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String(), String()})
	got, err := schema.Parse([]any{"asdf", "1234"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "asdf" || got[1] != "1234" {
		t.Fatalf("%#v", got)
	}
}

func TestTupleElementTypeError(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String(), String()})
	r1 := schema.SafeParse([]any{"asdf", 1234})
	if r1.Success {
		t.Fatal("expected failure")
	}
	iss := r1.Error.Issues[0]
	if iss.Code != IssueInvalidType || len(iss.Path) != 1 || iss.Path[0] != 1 {
		t.Fatalf("%#v", iss)
	}
}

func TestTupleTooBig(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String(), String()})
	r2 := schema.SafeParse([]any{"asdf", "1234", true})
	if r2.Success {
		t.Fatal("expected too_big")
	}
	iss := r2.Error.Issues[0]
	if iss.Code != IssueTooBig || iss.Origin != "array" || iss.Maximum != 2 {
		t.Fatalf("%#v", iss)
	}
}

func TestTupleTooSmall(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String(), String()})
	r := schema.SafeParse([]any{"asdf"})
	if r.Success {
		t.Fatal("expected too_small")
	}
	iss := r.Error.Issues[0]
	if iss.Code != IssueTooSmall || iss.Minimum != 2 {
		t.Fatalf("%#v", iss)
	}
}

func TestTupleRejectsNonArray(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String()})
	r3 := schema.SafeParse(map[string]any{})
	if r3.Success {
		t.Fatal("object should fail")
	}
	if r3.Error.Issues[0].Expected != "tuple" {
		t.Fatalf("%#v", r3.Error.Issues[0])
	}
}

func TestTupleRest(t *testing.T) {
	schema := Tuple([]AnySchemaLike{String()}).Rest(String())
	got, err := schema.Parse([]any{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("%#v", got)
	}
	res := schema.SafeParse([]any{"a", 1})
	if res.Success {
		t.Fatal("rest type error")
	}
	if res.Error.Issues[0].Path[0] != 1 {
		t.Fatalf("path: %#v", res.Error.Issues[0].Path)
	}
}

func TestTupleOptionalElements(t *testing.T) {
	schema := Tuple([]AnySchemaLike{
		String(),
		Optional(String()),
		Optional(String()),
	}).Rest(String())

	good := [][]any{
		{"asdf"},
		{"asdf", "1234"},
		{"asdf", "1234", "asdf"},
		{"asdf", "1234", "asdf", "true", "false"},
	}
	for _, data := range good {
		if _, err := schema.Parse(data); err != nil {
			t.Fatalf("good %#v: %v", data, err)
		}
	}

	bad := [][]any{
		{"asdf", 1234},              // bad type at 1
		{"asdf", "1234", "asdf", 9}, // bad rest
	}
	for _, data := range bad {
		if schema.SafeParse(data).Success {
			t.Fatalf("bad should fail: %#v", data)
		}
	}
}

func TestTupleOptionalThenRequiredLength(t *testing.T) {
	// Trailing required after optional: optinStart includes the required.
	schema := Tuple([]AnySchemaLike{
		String(),
		Optional(String()),
		String(),
	})
	if schema.SafeParse([]any{"a"}).Success {
		t.Fatal("too short — required third element")
	}
	if _, err := schema.Parse([]any{"a", "b", "c"}); err != nil {
		t.Fatal(err)
	}
}
