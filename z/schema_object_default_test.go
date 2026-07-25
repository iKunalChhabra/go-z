package z

import "testing"

// Reproduces: Default on an object field must apply when the key is absent.
func TestObjectDefaultAppliesWhenKeyAbsent(t *testing.T) {
	user := Object(Shape{
		"name":  String().Min(1),
		"email": Default(String().Email(), "hello@example.com"),
		"age":   Optional(Int().Gte(0)),
	})

	out, err := user.Parse(map[string]any{
		"name": "Ada",
		"age":  36,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["email"] != "hello@example.com" {
		t.Fatalf("expected default email in output, got %#v", out)
	}
	if out["name"] != "Ada" {
		t.Fatalf("name: %#v", out["name"])
	}
	if out["age"] != float64(36) && out["age"] != 36 {
		// Int normalizes to float64 via Number path — accept either
		if age, ok := ToFloat(out["age"]); !ok || age != 36 {
			t.Fatalf("age: %#v", out["age"])
		}
	}
}

func TestObjectDefaultDoesNotOverrideProvided(t *testing.T) {
	user := Object(Shape{
		"email": Default(String().Email(), "hello@example.com"),
	})
	out, err := user.Parse(map[string]any{"email": "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if out["email"] != "ada@example.com" {
		t.Fatalf("got %#v", out["email"])
	}
}
