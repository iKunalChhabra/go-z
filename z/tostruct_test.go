package z_test

import (
	"fmt"
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

// A recursive Go type used to overflow the plan builder with a fatal stack
// error: the plan for Node needed the plan for Node. The pending-plan cache
// reuses the in-progress plan, the way z.Lazy reuses an in-progress schema.
type Node struct {
	Value int   `json:"value"`
	Next  *Node `json:"next"`
}

func TestToStructRecursiveType(t *testing.T) {
	var nodeSchema z.AnySchemaLike
	nodeSchema = z.Object(z.Shape{
		"value": z.Int(),
		"next":  z.Optional(z.Lazy(func() z.AnySchemaLike { return nodeSchema })),
	})

	got, err := z.ToStruct[Node](nodeSchema).Parse(map[string]any{
		"value": 1,
		"next": map[string]any{
			"value": 2,
			"next":  map[string]any{"value": 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != 1 || got.Next == nil || got.Next.Value != 2 ||
		got.Next.Next == nil || got.Next.Next.Value != 3 || got.Next.Next.Next != nil {
		t.Fatalf("got %+v", got)
	}

	decoded, err := z.DecodeStruct[Node](map[string]any{
		"value": 1,
		"next":  map[string]any{"value": 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Next == nil || decoded.Next.Value != 2 {
		t.Fatalf("got %+v", decoded)
	}
}

// A fixed-size array decodes in place and refuses input longer than it holds.
func TestDecodeStructFixedArray(t *testing.T) {
	type Pair struct {
		Tags [2]string `json:"tags"`
	}
	got, err := z.DecodeStruct[Pair](map[string]any{"tags": []any{"a", "b"}})
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if got.Tags != [2]string{"a", "b"} {
		t.Fatalf("got %+v", got)
	}

	short, err := z.DecodeStruct[Pair](map[string]any{"tags": []any{"only"}})
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if short.Tags != [2]string{"only", ""} {
		t.Fatalf("got %+v", short)
	}

	if _, err := z.DecodeStruct[Pair](map[string]any{"tags": []any{"a", "b", "c"}}); err == nil {
		t.Fatal("expected an error for input longer than the array")
	}
}

// Go's int→string conversion yields a rune (65 → "A"); decoding must produce
// the decimal form instead.
func TestDecodeStructIntToString(t *testing.T) {
	type Row struct {
		Label string `json:"label"`
	}
	got, err := z.DecodeStruct[Row](map[string]any{"label": 65})
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if got.Label != "65" {
		t.Fatalf("got %q, want %q", got.Label, "65")
	}
}

// Numeric fields reject values they cannot hold exactly — no silent
// truncation or wraparound.
func TestDecodeStructRejectsLossyNumbers(t *testing.T) {
	type Row struct {
		Small int8   `json:"small"`
		Count uint32 `json:"count"`
		Whole int    `json:"whole"`
	}
	cases := map[string]map[string]any{
		"int8 overflow":      {"small": 300},
		"negative to uint32": {"count": -1},
		"non-integral float": {"whole": 3.9},
		"float overflow":     {"small": 300.0},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := z.DecodeStruct[Row](input); err == nil {
				t.Fatalf("expected a decode error for %#v", input)
			}
		})
	}

	ok, err := z.DecodeStruct[Row](map[string]any{
		"small": 127,
		"count": 4000000000.0,
		"whole": 3.0,
	})
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if ok.Small != 127 || ok.Count != 4000000000 || ok.Whole != 3 {
		t.Fatalf("got %+v", ok)
	}
}

// Slice element plans are built for any element type, so nested slices decode.
func TestDecodeStructNestedSlices(t *testing.T) {
	type Grid struct {
		Rows [][]string `json:"rows"`
	}
	got, err := z.DecodeStruct[Grid](map[string]any{
		"rows": []any{[]any{"a", "b"}, []any{"c"}},
	})
	if err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if len(got.Rows) != 2 || got.Rows[0][1] != "b" || got.Rows[1][0] != "c" {
		t.Fatalf("got %#v", got.Rows)
	}
}

// A non-empty interface field must not be Set blindly: a value that does not
// implement it is a decode error, not a panic.
func TestDecodeStructNonEmptyInterface(t *testing.T) {
	type Row struct {
		S fmt.Stringer `json:"s"`
	}
	if _, err := z.DecodeStruct[Row](map[string]any{"s": "plain string"}); err == nil {
		t.Fatal("expected a decode error, not a panic or silent success")
	}
}

// an embedded struct to the outer level. It used to skip them, so an embedded
// field silently stayed at its zero value.
func TestToStructPromotesEmbeddedFields(t *testing.T) {
	type Timestamps struct {
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}
	type Identity struct {
		ID string `json:"id"`
	}
	type User struct {
		Identity
		Timestamps
		Name string `json:"name"`
	}

	schema := z.ToStruct[User](z.Object(z.Shape{
		"id":         z.String(),
		"name":       z.String(),
		"created_at": z.String(),
		"updated_at": z.String(),
	}))

	got, err := schema.Parse(map[string]any{
		"id":         "u_1",
		"name":       "Ada",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-02T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "u_1" || got.Name != "Ada" {
		t.Fatalf("got %+v", got)
	}
	if got.CreatedAt != "2026-01-01T00:00:00Z" || got.UpdatedAt != "2026-01-02T00:00:00Z" {
		t.Fatalf("embedded fields not populated: %+v", got)
	}
}

// A field declared on the outer struct wins over a promoted one, and an embedded
// struct with a json tag stays a nested object.
func TestToStructEmbeddedPrecedence(t *testing.T) {
	type Base struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	type Outer struct {
		Base
		Name string `json:"name"` // shadows Base.Name
	}
	out, err := z.ToStruct[Outer](z.Object(z.Shape{
		"name": z.String(),
		"kind": z.String(),
	})).Parse(map[string]any{"name": "outer", "kind": "k"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "outer" || out.Base.Name != "" {
		t.Fatalf("shallower field must win: %+v", out)
	}
	if out.Kind != "k" {
		t.Fatalf("other promoted fields still decode: %+v", out)
	}

	type Tagged struct {
		Base `json:"base"`
	}
	tagged, err := z.ToStruct[Tagged](z.Object(z.Shape{
		"base": z.Object(z.Shape{"name": z.String(), "kind": z.String()}),
	})).Parse(map[string]any{"base": map[string]any{"name": "n", "kind": "k"}})
	if err != nil {
		t.Fatal(err)
	}
	if tagged.Base.Name != "n" {
		t.Fatalf("a tagged embed is a nested object: %+v", tagged)
	}
}
