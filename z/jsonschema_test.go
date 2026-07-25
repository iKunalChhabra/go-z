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

// Target used to change only the $schema header while the body stayed 2020-12,
// so a draft-07 consumer saw "prefixItems" (unknown, therefore ignored) plus an
// "items" schema that rejected every element of a tuple.
func TestJSONSchemaDraft07TupleForm(t *testing.T) {
	doc, err := ToJSONSchema(Tuple([]AnySchemaLike{String(), Number()}),
		ToJSONSchemaOpts{Target: JSONSchemaDraft07})
	if err != nil {
		t.Fatal(err)
	}
	if doc["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Fatalf("$schema = %v", doc["$schema"])
	}
	if _, ok := doc["prefixItems"]; ok {
		t.Error("prefixItems is not a draft-07 keyword")
	}
	items, ok := doc["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items = %#v", doc["items"])
	}
	if items[0].(map[string]any)["type"] != "string" || items[1].(map[string]any)["type"] != "number" {
		t.Fatalf("positional schemas lost: %#v", items)
	}
	if doc["additionalItems"] != false {
		t.Errorf("additionalItems = %#v, want false for a closed tuple", doc["additionalItems"])
	}

	// With a rest schema, the rest lands in additionalItems.
	withRest, err := ToJSONSchema(Tuple([]AnySchemaLike{String()}).Rest(Number()),
		ToJSONSchemaOpts{Target: JSONSchemaDraft07})
	if err != nil {
		t.Fatal(err)
	}
	rest, ok := withRest["additionalItems"].(map[string]any)
	if !ok || rest["type"] != "number" {
		t.Fatalf("additionalItems = %#v", withRest["additionalItems"])
	}
}

func TestJSONSchemaOpenAPI30Subset(t *testing.T) {
	doc, err := ToJSONSchema(Object(Shape{
		"name": String().Nullable(),
		"kind": Literal("user"),
		"age":  Number().Gt(0),
	}), ToJSONSchemaOpts{Target: JSONSchemaOpenAPI30})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["$schema"]; ok {
		t.Error("OpenAPI 3.0 schemas carry no $schema")
	}
	props := doc["properties"].(map[string]any)

	name := props["name"].(map[string]any)
	if name["type"] != "string" || name["nullable"] != true {
		t.Fatalf("nullable form wrong: %#v", name)
	}

	kind := props["kind"].(map[string]any)
	if _, ok := kind["const"]; ok {
		t.Error("const is not in the OpenAPI 3.0 subset")
	}
	if enum, ok := kind["enum"].([]any); !ok || len(enum) != 1 || enum[0] != "user" {
		t.Fatalf("const should become a single-value enum: %#v", kind)
	}

	age := props["age"].(map[string]any)
	if age["minimum"] != float64(0) || age["exclusiveMinimum"] != true {
		t.Fatalf("exclusive bound should be the boolean form: %#v", age)
	}
}

// A construct the target cannot express follows the Unrepresentable policy
// instead of emitting something invalid.
func TestJSONSchemaOpenAPI30Unrepresentable(t *testing.T) {
	tuple := Tuple([]AnySchemaLike{String()})
	if _, err := ToJSONSchema(tuple, ToJSONSchemaOpts{Target: JSONSchemaOpenAPI30}); err == nil {
		t.Fatal("a tuple should be reported as unrepresentable by default")
	}
	doc, err := ToJSONSchema(tuple, ToJSONSchemaOpts{
		Target:          JSONSchemaOpenAPI30,
		Unrepresentable: "any",
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc["type"] != "array" {
		t.Fatalf("type = %#v", doc["type"])
	}
	if _, ok := doc["prefixItems"]; ok {
		t.Error("prefixItems should be dropped")
	}

	if _, err := ToJSONSchema(Nil(), ToJSONSchemaOpts{Target: JSONSchemaOpenAPI30}); err == nil {
		t.Fatal("a bare null type should be reported as unrepresentable")
	}
}

// The rewrite must not touch a property that happens to be named after a keyword.
func TestJSONSchemaDialectIgnoresPropertyNames(t *testing.T) {
	doc, err := ToJSONSchema(Object(Shape{
		"const":       String(),
		"prefixItems": Number(),
		"type":        Bool(),
	}), ToJSONSchemaOpts{Target: JSONSchemaOpenAPI30})
	if err != nil {
		t.Fatal(err)
	}
	props := doc["properties"].(map[string]any)
	for name, want := range map[string]string{"const": "string", "prefixItems": "number", "type": "boolean"} {
		got, ok := props[name].(map[string]any)
		if !ok || got["type"] != want {
			t.Fatalf("property %q = %#v", name, props[name])
		}
	}
	if _, ok := doc["enum"]; ok {
		t.Error("a property named const must not be read as the const keyword")
	}
}
