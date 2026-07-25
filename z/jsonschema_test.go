package z

import (
	"testing"
)

func TestToJSONSchemaPrimitives(t *testing.T) {
	cases := []struct {
		name string
		sch  AnySchemaLike
		typ  string
	}{
		{"string", String(), "string"},
		{"number", Number(), "number"},
		{"boolean", Bool(), "boolean"},
		{"null", Null(), "null"},
		{"never", Never(), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			js, err := ToJSONSchema(c.sch)
			if err != nil {
				t.Fatal(err)
			}
			if js["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
				t.Fatalf("$schema = %v", js["$schema"])
			}
			if c.name == "never" {
				if _, ok := js["not"]; !ok {
					t.Fatalf("%#v", js)
				}
				return
			}
			if js["type"] != c.typ {
				t.Fatalf("type = %v want %v (%#v)", js["type"], c.typ, js)
			}
		})
	}
}

func TestToJSONSchemaObject(t *testing.T) {
	sch := Object(Shape{
		"name": String(),
		"age":  Number().Optional(),
	}).Strict()
	js, err := ToJSONSchema(sch)
	if err != nil {
		t.Fatal(err)
	}
	props, _ := js["properties"].(map[string]any)
	if props == nil || props["name"] == nil {
		t.Fatalf("%#v", js)
	}
	req, _ := js["required"].([]string)
	hasName := false
	for _, r := range req {
		if r == "name" {
			hasName = true
		}
		if r == "age" {
			t.Fatal("age should not be required")
		}
	}
	if !hasName {
		t.Fatalf("required = %v", req)
	}
	if js["additionalProperties"] != false {
		t.Fatalf("additionalProperties = %v", js["additionalProperties"])
	}
}

func TestToJSONSchemaArrayMinMax(t *testing.T) {
	sch := Array(String()).Min(1).Max(3)
	js, err := ToJSONSchema(sch)
	if err != nil {
		t.Fatal(err)
	}
	if js["minItems"] != 1 || js["maxItems"] != 3 {
		t.Fatalf("%#v", js)
	}
}

func TestToJSONSchemaXorOneOf(t *testing.T) {
	sch := XorOf(String(), Number())
	js, err := ToJSONSchema(sch)
	if err != nil {
		t.Fatal(err)
	}
	if js["oneOf"] == nil {
		t.Fatalf("expected oneOf: %#v", js)
	}
}

func TestToJSONSchemaUnrepresentable(t *testing.T) {
	_, err := ToJSONSchema(BigInt())
	if err == nil {
		t.Fatal("expected error")
	}
	js, err := ToJSONSchema(BigInt(), ToJSONSchemaOpts{Unrepresentable: "any"})
	if err != nil {
		t.Fatal(err)
	}
	if js["$schema"] == nil {
		t.Fatalf("%#v", js)
	}
}

func TestToJSONSchemaPipeIO(t *testing.T) {
	p := Pipe(String(), Number())
	inJS, err := ToJSONSchema(p, ToJSONSchemaOpts{IO: "input"})
	if err != nil {
		t.Fatal(err)
	}
	if inJS["type"] != "string" {
		t.Fatalf("input io: %#v", inJS)
	}
	outJS, err := ToJSONSchema(p, ToJSONSchemaOpts{IO: "output"})
	if err != nil {
		t.Fatal(err)
	}
	if outJS["type"] != "number" {
		t.Fatalf("output io: %#v", outJS)
	}
}
