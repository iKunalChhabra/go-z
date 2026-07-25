package z

import (
	"encoding/json"
	"regexp"
	"testing"
)

// These tests execute the examples printed in docs/content/api/{xor,
// template-literal, json-schema, codec}.md so the documentation cannot drift
// from the implementation without a test failure. Update them together.

func TestDocClaimsXor(t *testing.T) {
	ov := XorOf(String(), String().Min(1))
	res := ov.SafeParse("hello")
	if res.Success {
		t.Fatal("overlapping should fail")
	}
	if len(res.Error.Issues[0].Errors) != 0 {
		t.Fatalf("multi-match Errors = %d, want 0", len(res.Error.Issues[0].Errors))
	}
	if got, err := ov.Parse(""); err != nil || got != "" {
		t.Fatalf("empty string should match only option 1: %v %v", got, err)
	}
}

func TestDocClaimsTemplateLiteral(t *testing.T) {
	schema := TemplateLiteral([]any{
		"https://",
		String().Regex(regexp.MustCompile(`\w+`)),
		".",
		Enum("com", "net"),
	})
	if _, err := schema.Parse("https://example.com"); err != nil {
		t.Fatal(err)
	}
	if schema.SafeParse("https://example.org").Success || schema.SafeParse("http://example.com").Success {
		t.Fatal("expected failures")
	}

	version := TemplateLiteral([]any{"v", Number(), Literal("-beta").Optional()})
	if _, err := version.Parse("v1"); err != nil {
		t.Fatalf("v1: %v", err)
	}
	if _, err := version.Parse("v1-beta"); err != nil {
		t.Fatalf("v1-beta: %v", err)
	}

	if len(schema.Parts()) != 4 {
		t.Fatalf("parts = %d", len(schema.Parts()))
	}
	if schema.Internals().Pattern == nil {
		t.Fatal("Pattern should be published on Internals")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("unsupported part should panic at definition time")
			}
		}()
		_ = TemplateLiteral([]any{Object(Shape{"a": String()})})
	}()

	coerced := Pipe(Coerce.String(), TemplateLiteral([]any{"id-", Number()}))
	if got, err := coerced.Parse("id-42"); err != nil || got != "id-42" {
		t.Fatalf("coerced: %v %v", got, err)
	}
}

func TestDocClaimsJSONSchema(t *testing.T) {
	js, err := ToJSONSchema(Object(Shape{
		"name":  String().Min(2),
		"email": String().Email(),
		"age":   Optional(Int().Gte(0)),
	}))
	if err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(js)
	t.Logf("object schema: %s", out)

	if js["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("$schema = %v", js["$schema"])
	}
	props := js["properties"].(map[string]any)
	name := props["name"].(map[string]any)
	if name["type"] != "string" {
		t.Fatalf("name = %#v", name)
	}
	if _, ok := name["minLength"]; !ok {
		t.Errorf("documented minLength missing: %#v", name)
	}
	email := props["email"].(map[string]any)
	if email["format"] != "email" {
		t.Errorf("email format = %#v", email)
	}
	age := props["age"].(map[string]any)
	if _, ok := age["minimum"]; !ok {
		t.Errorf("documented minimum missing: %#v", age)
	}
	req, _ := js["required"].([]string)
	if len(req) != 2 {
		t.Errorf("required = %v", req)
	}

	// Unrepresentable handling
	if _, err := ToJSONSchema(BigInt()); err == nil {
		t.Error("default should throw on unrepresentable")
	}
	if _, err := ToJSONSchema(BigInt(), ToJSONSchemaOpts{Unrepresentable: "any"}); err != nil {
		t.Errorf(`Unrepresentable:"any": %v`, err)
	}

	// IO selection
	p := Pipe(String(), Number())
	in, _ := ToJSONSchema(p, ToJSONSchemaOpts{IO: "input"})
	outJS, _ := ToJSONSchema(p, ToJSONSchemaOpts{IO: "output"})
	if in["type"] != "string" || outJS["type"] != "number" {
		t.Errorf("io: in=%v out=%v", in["type"], outJS["type"])
	}

	// Metadata
	described := Describe(String().Email(), "Primary contact address")
	djs, _ := ToJSONSchema(described)
	if djs["description"] != "Primary contact address" {
		t.Errorf("description = %v", djs["description"])
	}

	// Nullable and template literal shapes
	njs, _ := ToJSONSchema(Nullable(String()))
	if _, ok := njs["type"].([]any); !ok {
		t.Errorf("nullable type = %#v", njs["type"])
	}
	tjs, err := ToJSONSchema(TemplateLiteral([]any{"a", Number()}))
	if err != nil || tjs["type"] != "string" || tjs["pattern"] == nil {
		t.Errorf("template literal = %#v (%v)", tjs, err)
	}

	// Recursive schemas terminate
	var node *ObjectSchema
	node = Object(Shape{"name": String(), "child": Optional(Lazy(func() AnySchemaLike { return node }))})
	if _, err := ToJSONSchema(node); err != nil {
		t.Errorf("recursive: %v", err)
	}
}

func TestDocClaimsCodec(t *testing.T) {
	c := Codec(String(), Number(), CodecTx{
		Decode: func(v any, _ *RefinementCtx) (any, error) { return float64(len(v.(string))), nil },
		Encode: func(v any, _ *RefinementCtx) (any, error) { return "encoded", nil },
	})
	if got, err := Decode(c, "abc"); err != nil || got != 3.0 {
		t.Fatalf("decode: %v %v", got, err)
	}
	if got, err := Encode(c, 3.0); err != nil || got != "encoded" {
		t.Fatalf("encode: %v %v", got, err)
	}
	if !SafeDecode(c, "ab").Success || SafeDecode(c, 1).Success {
		t.Fatal("safe decode")
	}
	inv := InvertCodec(c)
	if got, err := Decode(inv, 3.0); err != nil || got != "encoded" {
		t.Fatalf("invert: %v %v", got, err)
	}
	if c.In() == nil || c.Out() == nil {
		t.Fatal("accessors")
	}

	// Codec nested in an object works in both directions.
	obj := Object(Shape{"n": c})
	dec, err := Decode(obj, map[string]any{"n": "abcd"})
	if err != nil || dec.(map[string]any)["n"] != 4.0 {
		t.Fatalf("nested decode: %#v %v", dec, err)
	}
	enc, err := Encode(obj, map[string]any{"n": 4.0})
	if err != nil || enc.(map[string]any)["n"] != "encoded" {
		t.Fatalf("nested encode: %#v %v", enc, err)
	}

	// Defaults/catch are skipped on encode; transform panics.
	if SafeEncode(Default(String(), "d"), Missing).Success {
		t.Error("default must not fabricate a value on encode")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("encoding through a Transform should panic")
			}
		}()
		_, _ = Encode(Transform(String(), func(v any, _ *RefinementCtx) (any, error) { return v, nil }), "x")
	}()

	// JSON string codec
	j := JSONStringCodec(Object(Shape{"retries": Int().Gte(0)}))
	cfg, err := Decode(j, `{"retries":3}`)
	if err != nil || cfg.(map[string]any)["retries"] != 3 {
		t.Fatalf("json decode: %#v %v", cfg, err)
	}
	raw, err := Encode(j, map[string]any{"retries": 3})
	if err != nil || raw != `{"retries":3}` {
		t.Fatalf("json encode: %#v %v", raw, err)
	}
	if j.SafeParse("not json").Success {
		t.Error("invalid json should fail")
	}
}
