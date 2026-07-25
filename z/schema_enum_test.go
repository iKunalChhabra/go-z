package z

import "testing"

// Ported from classic/tests/enum.test.ts

func TestEnumFromStringArgs(t *testing.T) {
	my := Enum("Red", "Green", "Blue")
	my.MustParse("Red")
	my.MustParse("Green")
	my.MustParse("Blue")
	res := my.SafeParse("Yellow")
	if res.Success {
		t.Fatal("expected fail")
	}
	iss := res.Error.Issues[0]
	if iss.Code != IssueInvalidValue {
		t.Fatalf("code=%s", iss.Code)
	}
	if iss.Message != `Invalid option: expected one of "Red"|"Green"|"Blue"` {
		t.Fatalf("got %q", iss.Message)
	}
	if len(iss.Values) != 3 {
		t.Fatalf("values=%v", iss.Values)
	}
	opts := my.Options()
	if len(opts) != 3 || opts[0] != "Red" {
		t.Fatalf("options=%v", opts)
	}
	if _, ok := my.Internals().Values["Red"]; !ok {
		t.Fatal("Internals.Values missing Red")
	}
}

func TestNativeEnum(t *testing.T) {
	fruits := map[string]string{
		"Apple":  "apple",
		"Banana": "banana",
	}
	fruitEnum := NativeEnum(fruits)
	fruitEnum.MustParse("apple")
	fruitEnum.MustParse("banana")
	fruitEnum.MustParse(fruits["Apple"])
	if fruitEnum.SafeParse("Apple").Success {
		t.Fatal("key should not be accepted, only value")
	}
	if fruitEnum.SafeParse("Cantaloupe").Success {
		t.Fatal("expected fail")
	}
	em := fruitEnum.EnumMap()
	if em["Apple"] != "apple" {
		t.Fatalf("EnumMap=%v", em)
	}
}

func TestEnumNonConst(t *testing.T) {
	foods := []string{"Pasta", "Pizza", "Tacos", "Burgers", "Salad"}
	foodEnum := Enum(foods...)
	if !foodEnum.SafeParse("Pasta").Success {
		t.Fatal("Pasta should pass")
	}
	if foodEnum.SafeParse("Cucumbers").Success {
		t.Fatal("Cucumbers should fail")
	}
}
