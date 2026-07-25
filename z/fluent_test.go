package z

import "testing"

func TestFluentOptionalInObject(t *testing.T) {
	schema := Object(Shape{
		"email": String().Min(2).Email().Optional(),
	})
	out, err := schema.Parse(map[string]any{})
	if err != nil {
		t.Fatalf("missing key should be ok: %v", err)
	}
	if _, ok := out["email"]; ok {
		t.Fatalf("optional missing key should omit email, got %#v", out)
	}

	out, err = schema.Parse(map[string]any{"email": "ab@c.com"})
	if err != nil {
		t.Fatalf("valid email: %v", err)
	}
	if out["email"] != "ab@c.com" {
		t.Fatalf("got %#v", out["email"])
	}

	if schema.SafeParse(map[string]any{"email": "x"}).Success {
		t.Fatal("too-short non-email should fail")
	}
}

func TestFluentRefineMidChain(t *testing.T) {
	schema := String().Min(2).Refine(func(s string) bool {
		return s != "xx"
	}, "bad").Email()

	if _, err := schema.Parse("ab@c.com"); err != nil {
		t.Fatalf("valid: %v", err)
	}
	res := schema.SafeParse("xx")
	if res.Success {
		t.Fatal("refine should reject xx")
	}
	// "xx" fails Min(2)? No, len=2. Refine rejects. Email also fails format —
	// either way failure is expected. Prefer refine message when Min passes.
	res2 := schema.SafeParse("not-an-email")
	if res2.Success {
		t.Fatal("Email after Refine should still apply")
	}
}

func TestFluentDefaultInObject(t *testing.T) {
	schema := Object(Shape{
		"n": Number().Gte(0).Default(1.0),
	})
	out, err := schema.Parse(map[string]any{})
	if err != nil {
		t.Fatalf("absent key: %v", err)
	}
	if out["n"] != 1.0 {
		t.Fatalf("want default 1.0, got %#v", out["n"])
	}

	out, err = schema.Parse(map[string]any{"n": 3.0})
	if err != nil {
		t.Fatalf("present: %v", err)
	}
	if out["n"] != 3.0 {
		t.Fatalf("want 3.0, got %#v", out["n"])
	}
}

func TestFluentObjectStrictDefault(t *testing.T) {
	schema := Object(Shape{"a": String()}).Strict().Default(map[string]any{"a": "x"})
	got, err := schema.Parse(Missing)
	if err != nil {
		t.Fatalf("default object: %v", err)
	}
	// Default keeps the object's typed edge: Parse returns map[string]any.
	if got["a"] != "x" {
		t.Fatalf("got %#v", got)
	}
}
