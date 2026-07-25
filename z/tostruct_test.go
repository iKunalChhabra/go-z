package z_test

import (
	"testing"

	"github.com/iKunalChhabra/go-z/z"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func TestToStructRoundTrip(t *testing.T) {
	obj := z.Object(z.Shape{
		"name":  z.String().Min(2),
		"email": z.String().Email(),
		"age":   z.Int().Gte(0),
	})
	schema := z.ToStruct[User](obj)

	in := map[string]any{
		"name":  "Ada",
		"email": "ada@example.com",
		"age":   float64(36), // JSON numbers are float64
	}
	got, err := schema.Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Name != "Ada" || got.Email != "ada@example.com" || got.Age != 36 {
		t.Fatalf("got %+v", got)
	}
}

func TestToStructValidationFailure(t *testing.T) {
	obj := z.Object(z.Shape{
		"name":  z.String().Min(2),
		"email": z.String().Email(),
		"age":   z.Int().Gte(0),
	})
	schema := z.ToStruct[User](obj)

	_, err := schema.Parse(map[string]any{
		"name":  "A",
		"email": "not-an-email",
		"age":   float64(-1),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	zerr, ok := err.(*z.Error)
	if !ok {
		t.Fatalf("want *Error, got %T", err)
	}
	if len(zerr.Issues) == 0 {
		t.Fatal("expected issues")
	}
}

func TestDecodeStructCached(t *testing.T) {
	data := map[string]any{
		"name":  "Grace",
		"email": "grace@example.com",
		"age":   float64(40),
	}
	u1, err := z.DecodeStruct[User](data)
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	u2, err := z.DecodeStruct[User](data)
	if err != nil {
		t.Fatalf("DecodeStruct 2: %v", err)
	}
	if u1 != u2 {
		t.Fatalf("u1=%+v u2=%+v", u1, u2)
	}
	if u1.Name != "Grace" || u1.Age != 40 {
		t.Fatalf("got %+v", u1)
	}
}

func TestDecodeStructNested(t *testing.T) {
	type Address struct {
		City string `json:"city"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}
	got, err := z.DecodeStruct[Person](map[string]any{
		"name": "Lin",
		"address": map[string]any{
			"city": "SF",
		},
	})
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if got.Name != "Lin" || got.Address.City != "SF" {
		t.Fatalf("got %+v", got)
	}
}

func TestDecodeStructJSONTagSkip(t *testing.T) {
	type Row struct {
		Keep string `json:"keep"`
		Skip string `json:"-"`
	}
	got, err := z.DecodeStruct[Row](map[string]any{
		"keep": "yes",
		"-":    "no",
		"Skip": "no",
	})
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if got.Keep != "yes" || got.Skip != "" {
		t.Fatalf("got %+v", got)
	}
}
