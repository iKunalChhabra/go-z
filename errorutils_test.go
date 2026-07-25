package zod

import (
	"reflect"
	"strings"
	"testing"
)

// Ported shapes from classic/tests/error-utils.test.ts (flatten/format/treeify/prettify/toDotPath).

func TestFlattenFormAndFieldErrors(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueInvalidType, Expected: "number", Path: []any{"f1"}, Message: "Invalid input: expected number, received undefined"},
		{Code: IssueInvalidType, Expected: "string", Path: []any{"f3"}, Message: "Invalid input: expected string, received undefined"},
		{Code: IssueInvalidType, Expected: "array", Path: []any{"f4"}, Message: "Invalid input: expected array, received undefined"},
	}}
	got := Flatten(err)
	if len(got.FormErrors) != 0 {
		t.Fatalf("formErrors: %v", got.FormErrors)
	}
	if !reflect.DeepEqual(got.FieldErrors["f1"], []string{"Invalid input: expected number, received undefined"}) {
		t.Fatalf("f1: %v", got.FieldErrors["f1"])
	}
	if !reflect.DeepEqual(got.FieldErrors["f3"], []string{"Invalid input: expected string, received undefined"}) {
		t.Fatalf("f3: %v", got.FieldErrors["f3"])
	}
	if !reflect.DeepEqual(got.FieldErrors["f4"], []string{"Invalid input: expected array, received undefined"}) {
		t.Fatalf("f4: %v", got.FieldErrors["f4"])
	}
}

func TestFlattenFormErrors(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueCustom, Message: "Must be equal", Path: []any{}},
	}}
	got := Flatten(err)
	if !reflect.DeepEqual(got.FormErrors, []string{"Must be equal"}) {
		t.Fatalf("formErrors: %v", got.FormErrors)
	}
	if len(got.FieldErrors) != 0 {
		t.Fatalf("fieldErrors: %v", got.FieldErrors)
	}
}

func TestFlattenMap(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueInvalidType, Path: []any{"a"}, Message: "hello"},
	}}
	got := FlattenMap(err, func(iss Issue) int { return len(iss.Message) })
	if !reflect.DeepEqual(got.FieldErrors["a"], []int{5}) {
		t.Fatalf("mapped: %v", got.FieldErrors["a"])
	}
}

func TestFormatNested(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueInvalidType, Path: []any{"username"}, Message: "Invalid input: expected string, received number"},
		{Code: IssueInvalidType, Path: []any{"favoriteNumbers", 1}, Message: "Invalid input: expected number, received string"},
		{Code: IssueInvalidType, Path: []any{"nesting", "a"}, Message: "Invalid input: expected string, received number"},
		{Code: IssueUnrecognizedKeys, Keys: []string{"extra"}, Path: []any{}, Message: `Unrecognized key: "extra"`},
	}}
	got := Format(err)
	rootErrs, _ := got["_errors"].([]string)
	if len(rootErrs) != 1 || rootErrs[0] != `Unrecognized key: "extra"` {
		t.Fatalf("_errors: %#v", got["_errors"])
	}
	user, ok := got["username"].(map[string]any)
	if !ok {
		t.Fatalf("username missing: %#v", got)
	}
	if msgs, _ := user["_errors"].([]string); !reflect.DeepEqual(msgs, []string{"Invalid input: expected string, received number"}) {
		t.Fatalf("username._errors: %#v", user["_errors"])
	}
	fav, ok := got["favoriteNumbers"].(map[string]any)
	if !ok {
		t.Fatalf("favoriteNumbers missing")
	}
	idx1, ok := fav["1"].(map[string]any)
	if !ok {
		t.Fatalf("favoriteNumbers[1] missing: %#v", fav)
	}
	if msgs, _ := idx1["_errors"].([]string); !reflect.DeepEqual(msgs, []string{"Invalid input: expected number, received string"}) {
		t.Fatalf("fav[1]: %#v", idx1["_errors"])
	}
}

func TestFormatInvalidUnionNested(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{
			Code: IssueInvalidUnion,
			Path: []any{"logLevel"},
			Errors: [][]Issue{
				{{Code: IssueInvalidType, Expected: "string", Path: []any{}, Message: "Invalid input: expected string, received boolean"}},
				{{Code: IssueInvalidType, Expected: "number", Path: []any{}, Message: "Invalid input: expected number, received boolean"}},
			},
			Message: "Invalid input",
		},
	}}
	got := Format(err)
	node, ok := got["logLevel"].(map[string]any)
	if !ok {
		t.Fatalf("logLevel missing: %#v", got)
	}
	msgs, _ := node["_errors"].([]string)
	if !reflect.DeepEqual(msgs, []string{
		"Invalid input: expected string, received boolean",
		"Invalid input: expected number, received boolean",
	}) {
		t.Fatalf("union format: %#v", msgs)
	}
}

func TestFormatInvalidKeyAndElement(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{
			Code:   IssueInvalidKey,
			Origin: "record",
			Path:   []any{"data"},
			Issues: []Issue{
				{Code: IssueInvalidType, Expected: "string", Path: []any{"bad"}, Message: "bad key"},
			},
			Message: "Invalid key in record",
		},
		{
			Code:   IssueInvalidElement,
			Origin: "map",
			Path:   []any{"m"},
			Issues: []Issue{
				{Code: IssueInvalidType, Expected: "number", Path: []any{"x"}, Message: "bad elem"},
			},
			Message: "Invalid value in map",
		},
	}}
	got := Format(err)
	data := got["data"].(map[string]any)
	bad := data["bad"].(map[string]any)
	if msgs, _ := bad["_errors"].([]string); !reflect.DeepEqual(msgs, []string{"bad key"}) {
		t.Fatalf("invalid_key format: %#v", bad)
	}
	m := got["m"].(map[string]any)
	x := m["x"].(map[string]any)
	if msgs, _ := x["_errors"].([]string); !reflect.DeepEqual(msgs, []string{"bad elem"}) {
		t.Fatalf("invalid_element format: %#v", x)
	}
}

func TestTreeify(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueUnrecognizedKeys, Path: []any{}, Message: `Unrecognized key: "extra"`},
		{Code: IssueInvalidType, Path: []any{"username"}, Message: "Invalid input: expected string, received number"},
		{Code: IssueInvalidType, Path: []any{"favoriteNumbers", 1}, Message: "Invalid input: expected number, received string"},
		{Code: IssueInvalidType, Path: []any{"nesting", "a"}, Message: "Invalid input: expected string, received number"},
	}}
	tree := Treeify(err)
	if !reflect.DeepEqual(tree.Errors, []string{`Unrecognized key: "extra"`}) {
		t.Fatalf("root errors: %#v", tree.Errors)
	}
	if tree.Properties["username"] == nil || !reflect.DeepEqual(tree.Properties["username"].Errors, []string{"Invalid input: expected string, received number"}) {
		t.Fatalf("username: %#v", tree.Properties["username"])
	}
	fav := tree.Properties["favoriteNumbers"]
	if fav == nil || len(fav.Items) < 2 || fav.Items[1] == nil {
		t.Fatalf("favoriteNumbers items: %#v", fav)
	}
	if !reflect.DeepEqual(fav.Items[1].Errors, []string{"Invalid input: expected number, received string"}) {
		t.Fatalf("items[1]: %#v", fav.Items[1].Errors)
	}
	nest := tree.Properties["nesting"]
	if nest == nil || nest.Properties["a"] == nil {
		t.Fatalf("nesting: %#v", nest)
	}
	if !reflect.DeepEqual(nest.Properties["a"].Errors, []string{"Invalid input: expected string, received number"}) {
		t.Fatalf("nesting.a: %#v", nest.Properties["a"].Errors)
	}
}

func TestTreeifyInvalidUnion(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{
			Code: IssueInvalidUnion,
			Path: []any{"logLevel"},
			Errors: [][]Issue{
				{{Code: IssueInvalidType, Path: []any{}, Message: "Invalid input: expected string, received boolean"}},
				{{Code: IssueInvalidType, Path: []any{}, Message: "Invalid input: expected number, received boolean"}},
			},
			Message: "Invalid input",
		},
	}}
	tree := Treeify(err)
	node := tree.Properties["logLevel"]
	if node == nil {
		t.Fatal("missing logLevel")
	}
	if !reflect.DeepEqual(node.Errors, []string{
		"Invalid input: expected string, received boolean",
		"Invalid input: expected number, received boolean",
	}) {
		t.Fatalf("union tree: %#v", node.Errors)
	}
}

func TestPrettify(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueUnrecognizedKeys, Path: []any{}, Message: `Unrecognized key: "extra"`},
		{Code: IssueInvalidType, Path: []any{"username"}, Message: "Invalid input: expected string, received number"},
		{Code: IssueInvalidType, Path: []any{"favoriteNumbers", 1}, Message: "Invalid input: expected number, received string"},
		{Code: IssueInvalidType, Path: []any{"nesting", "a"}, Message: "Invalid input: expected string, received number"},
	}}
	got := Prettify(err)
	want := strings.Join([]string{
		`✖ Unrecognized key: "extra"`,
		`✖ Invalid input: expected string, received number`,
		`  → at username`,
		`✖ Invalid input: expected number, received string`,
		`  → at favoriteNumbers[1]`,
		`✖ Invalid input: expected string, received number`,
		`  → at nesting.a`,
	}, "\n")
	if got != want {
		t.Fatalf("prettify\n got: %q\nwant: %q", got, want)
	}
}

func TestToDotPath(t *testing.T) {
	cases := []struct {
		path []any
		want string
	}{
		{[]any{"a", "b", 0, "c"}, "a.b[0].c"},
		{[]any{"user.name", "first.last"}, `["user.name"]["first.last"]`},
		{[]any{"user", "$special"}, "user.$special"},
		{[]any{"search", `query("foo.bar"="abc")`}, `search["query(\"foo.bar\"=\"abc\")"]`},
		{[]any{"search", "foo\nbar"}, `search["foo\nbar"]`},
		{[]any{"", "empty"}, ".empty"},
		{[]any{"items", 0, 1, 2}, "items[0][1][2]"},
		{[]any{"users", "user.config", 0, "settings.theme"}, `users["user.config"][0]["settings.theme"]`},
		{[]any{"data[0]", "value"}, `["data[0]"].value`},
		{[]any{}, ""},
		{[]any{map[string]any{"key": "wrapped"}}, "wrapped"},
	}
	for _, tc := range cases {
		if got := ToDotPath(tc.path); got != tc.want {
			t.Errorf("ToDotPath(%v) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
