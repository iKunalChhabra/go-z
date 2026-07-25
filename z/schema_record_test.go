package z

import "testing"

// Ported from v4/classic/tests/record.test.ts (key/value validation, enum exhaustiveness via Keyof Values).

func TestRecordBasic(t *testing.T) {
	schema := Record(String(), String())
	got, err := schema.Parse(map[string]any{"a": "1", "b": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "1" || got["b"] != "2" {
		t.Fatalf("%#v", got)
	}
}

func TestRecordValueTypeError(t *testing.T) {
	schema := Record(String(), String())
	res := schema.SafeParse(map[string]any{"a": 1})
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidType || len(iss.Path) != 1 || iss.Path[0] != "a" {
		t.Fatalf("%#v", iss)
	}
}

func TestRecordInvalidKey(t *testing.T) {
	// Key schema that only accepts "ok" (enum exhaustiveness path).
	key := Enum("ok")
	schema := Record(key, String())
	// Exhaustive mode: Values set requires "ok", rejects extras.
	got, err := schema.Parse(map[string]any{"ok": "yes"})
	if err != nil {
		t.Fatal(err)
	}
	if got["ok"] != "yes" {
		t.Fatalf("%#v", got)
	}

	res := schema.SafeParse(map[string]any{"ok": "yes", "no": "x"})
	if res.Success {
		t.Fatal("unrecognized key")
	}
	if res.Error.Issues[0].Code != IssueUnrecognizedKeys {
		t.Fatalf("%#v", res.Error.Issues[0])
	}

	// Missing required enum key → value schema sees Missing.
	res = schema.SafeParse(map[string]any{})
	if res.Success {
		t.Fatal("missing enum key")
	}
	found := false
	for _, iss := range res.Error.Issues {
		if iss.Code == IssueInvalidType && len(iss.Path) == 1 && iss.Path[0] == "ok" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%#v", res.Error.Issues)
	}
}

func TestRecordOpenKeySchema(t *testing.T) {
	// String key schema has no Values → open record.
	schema := Record(String(), String())
	got, err := schema.Parse(map[string]any{"any": "key"})
	if err != nil {
		t.Fatal(err)
	}
	if got["any"] != "key" {
		t.Fatalf("%#v", got)
	}
}

func TestRecordRejectsNonObject(t *testing.T) {
	if Record(String(), String()).SafeParse([]any{}).Success {
		t.Fatal("array should fail")
	}
}

func TestRecordNestedPath(t *testing.T) {
	schema := Record(String(), Object(Shape{"n": String()}))
	res := schema.SafeParse(map[string]any{"x": map[string]any{}})
	if res.Success {
		t.Fatal("expected failure")
	}
	iss := res.Error.Issues[0]
	if len(iss.Path) != 2 || iss.Path[0] != "x" || iss.Path[1] != "n" {
		t.Fatalf("path: %#v", iss.Path)
	}
}
