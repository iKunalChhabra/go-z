package zod

import (
	"reflect"
	"strings"
	"testing"
)

// Parity ports from:
//   packages/zod/src/v4/classic/tests/error.test.ts
//   packages/zod/src/v4/classic/tests/error-utils.test.ts
// Focus: Flatten / Format / Treeify / Prettify / ToDotPath (+ custom error maps).

func TestParityErrorUtilsRegularError(t *testing.T) {
	// Port: error-utils.test.ts schema parse of {}
	schema := Object(Shape{
		"f1": Number(),
		"f2": Optional(String()),
		"f3": Nullable(String()),
		"f4": Array(Object(Shape{"t": UnionOf(String(), Bool())})),
	})
	res := schema.SafeParse(map[string]any{})
	if res.Success {
		t.Fatal("expected failure")
	}
	codes := map[string]string{}
	for _, iss := range res.Error.Issues {
		if len(iss.Path) > 0 {
			codes[pathSegString(iss.Path[0])] = iss.Expected
		}
	}
	if codes["f1"] != "number" || codes["f3"] != "string" || codes["f4"] != "array" {
		t.Fatalf("codes = %#v issues=%+v", codes, res.Error.Issues)
	}
}

func TestParityFlattenFormAndFieldErrors(t *testing.T) {
	// Port: ".flatten()"
	schema := Object(Shape{
		"f1": Number(),
		"f3": Nullable(String()),
		"f4": Array(String()),
	})
	res := schema.SafeParse(map[string]any{})
	flat := Flatten(res.Error)
	if len(flat.FormErrors) != 0 {
		t.Fatalf("formErrors: %v", flat.FormErrors)
	}
	for _, key := range []string{"f1", "f3", "f4"} {
		if len(flat.FieldErrors[key]) == 0 {
			t.Fatalf("missing fieldErrors[%s]: %#v", key, flat.FieldErrors)
		}
	}
}

func TestParityFlattenCustomMapper(t *testing.T) {
	// Port: "custom .flatten()"
	err := &ZodError{Issues: []Issue{
		{Code: IssueInvalidType, Expected: "number", Path: []any{"f1"}, Message: "bad number"},
		{Code: IssueInvalidType, Expected: "string", Path: []any{"f3"}, Message: "bad string"},
	}}
	type errT struct {
		Message string
		Code    int
	}
	got := FlattenMap(err, func(iss Issue) errT {
		return errT{Message: iss.Message, Code: 1234}
	})
	if got.FieldErrors["f1"][0] != (errT{Message: "bad number", Code: 1234}) {
		t.Fatalf("f1: %#v", got.FieldErrors["f1"])
	}
	if got.FieldErrors["f3"][0].Message != "bad string" {
		t.Fatalf("f3: %#v", got.FieldErrors["f3"])
	}
}

func TestParityFlattenFormErrors(t *testing.T) {
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

func TestParityFormatNested(t *testing.T) {
	// Port: ".format()" nested structure
	err := &ZodError{Issues: []Issue{
		{Code: IssueInvalidType, Path: []any{"username"}, Message: "Invalid input: expected string, received number"},
		{Code: IssueInvalidType, Path: []any{"favoriteNumbers", 1}, Message: "Invalid input: expected number, received string"},
		{Code: IssueInvalidType, Path: []any{"nesting", "a"}, Message: "Invalid input: expected string, received number"},
		{Code: IssueUnrecognizedKeys, Keys: []string{"extra"}, Path: []any{}, Message: `Unrecognized key: "extra"`},
	}}
	got := Format(err)
	root, _ := got["_errors"].([]string)
	if len(root) != 1 || root[0] != `Unrecognized key: "extra"` {
		t.Fatalf("_errors: %#v", got["_errors"])
	}
	user := got["username"].(map[string]any)
	if msgs, _ := user["_errors"].([]string); !reflect.DeepEqual(msgs, []string{"Invalid input: expected string, received number"}) {
		t.Fatalf("username: %#v", user["_errors"])
	}
	fav := got["favoriteNumbers"].(map[string]any)
	idx1 := fav["1"].(map[string]any)
	if msgs, _ := idx1["_errors"].([]string); !reflect.DeepEqual(msgs, []string{"Invalid input: expected number, received string"}) {
		t.Fatalf("fav[1]: %#v", idx1["_errors"])
	}
}

func TestParityFormatInvalidUnion(t *testing.T) {
	err := &ZodError{Issues: []Issue{{
		Code: IssueInvalidUnion,
		Path: []any{"logLevel"},
		Errors: [][]Issue{
			{{Code: IssueInvalidType, Expected: "string", Path: []any{}, Message: "Invalid input: expected string, received boolean"}},
			{{Code: IssueInvalidType, Expected: "number", Path: []any{}, Message: "Invalid input: expected number, received boolean"}},
		},
		Message: "Invalid input",
	}}}
	got := Format(err)
	node := got["logLevel"].(map[string]any)
	msgs, _ := node["_errors"].([]string)
	if !reflect.DeepEqual(msgs, []string{
		"Invalid input: expected string, received boolean",
		"Invalid input: expected number, received boolean",
	}) {
		t.Fatalf("union format: %#v", msgs)
	}
}

func TestParityFormatMap(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueInvalidType, Path: []any{"a"}, Message: "hello"},
	}}
	got := FormatMap(err, func(iss Issue) int { return len(iss.Message) })
	node := got["a"].(map[string]any)
	if lens, _ := node["_errors"].([]int); !reflect.DeepEqual(lens, []int{5}) {
		t.Fatalf("mapped: %#v", node["_errors"])
	}
}

func TestParityTreeify(t *testing.T) {
	err := &ZodError{Issues: []Issue{
		{Code: IssueUnrecognizedKeys, Path: []any{}, Message: `Unrecognized key: "extra"`},
		{Code: IssueInvalidType, Path: []any{"username"}, Message: "Invalid input: expected string, received number"},
		{Code: IssueInvalidType, Path: []any{"favoriteNumbers", 1}, Message: "Invalid input: expected number, received string"},
		{Code: IssueInvalidType, Path: []any{"nesting", "a"}, Message: "Invalid input: expected string, received number"},
	}}
	tree := Treeify(err)
	if !reflect.DeepEqual(tree.Errors, []string{`Unrecognized key: "extra"`}) {
		t.Fatalf("root: %#v", tree.Errors)
	}
	if !reflect.DeepEqual(tree.Properties["username"].Errors, []string{"Invalid input: expected string, received number"}) {
		t.Fatalf("username: %#v", tree.Properties["username"])
	}
	fav := tree.Properties["favoriteNumbers"]
	if fav == nil || len(fav.Items) < 2 || fav.Items[1] == nil {
		t.Fatalf("favoriteNumbers: %#v", fav)
	}
	if !reflect.DeepEqual(fav.Items[1].Errors, []string{"Invalid input: expected number, received string"}) {
		t.Fatalf("items[1]: %#v", fav.Items[1].Errors)
	}
}

func TestParityTreeifyInvalidUnion(t *testing.T) {
	err := &ZodError{Issues: []Issue{{
		Code: IssueInvalidUnion,
		Path: []any{"logLevel"},
		Errors: [][]Issue{
			{{Code: IssueInvalidType, Path: []any{}, Message: "Invalid input: expected string, received boolean"}},
			{{Code: IssueInvalidType, Path: []any{}, Message: "Invalid input: expected number, received boolean"}},
		},
		Message: "Invalid input",
	}}}
	tree := Treeify(err)
	node := tree.Properties["logLevel"]
	if node == nil || !reflect.DeepEqual(node.Errors, []string{
		"Invalid input: expected string, received boolean",
		"Invalid input: expected number, received boolean",
	}) {
		t.Fatalf("union tree: %#v", node)
	}
}

func TestParityPrettify(t *testing.T) {
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

func TestParityToDotPath(t *testing.T) {
	cases := []struct {
		path []any
		want string
	}{
		{[]any{"a", "b", 0, "c"}, "a.b[0].c"},
		{[]any{"user.name", "first.last"}, `["user.name"]["first.last"]`},
		{[]any{"user", "$special"}, "user.$special"},
		{[]any{"", "empty"}, ".empty"},
		{[]any{"items", 0, 1, 2}, "items[0][1][2]"},
		{[]any{"users", "user.config", 0, "settings.theme"}, `users["user.config"][0]["settings.theme"]`},
		{[]any{"data[0]", "value"}, `["data[0]"].value`},
		{[]any{}, ""},
	}
	for _, tc := range cases {
		if got := ToDotPath(tc.path); got != tc.want {
			t.Errorf("ToDotPath(%v) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestParityErrorCustomMessageOnSchema(t *testing.T) {
	// Port: error.test.ts custom message shorthand
	schema := String("bad string!")
	res := schema.SafeParse(123)
	if res.Success || res.Error.Issues[0].Message != "bad string!" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityErrorCustomMessageOnCheck(t *testing.T) {
	schema := String().Min(5, "too short")
	res := schema.SafeParse("abc")
	if res.Success || res.Error.Issues[0].Message != "too short" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityErrorMapOverride(t *testing.T) {
	schema := String(ErrorMap(func(iss *Issue) string {
		return "mapped:" + string(iss.Code)
	}))
	res := schema.SafeParse(true)
	if res.Success || res.Error.Issues[0].Message != "mapped:invalid_type" {
		t.Fatalf("got %+v", res.Error)
	}
}

func TestParityErrorPathOnObjectField(t *testing.T) {
	schema := Object(Shape{
		"name": String(),
		"age":  Number(),
	})
	res := schema.SafeParse(map[string]any{"name": 1, "age": "x"})
	if res.Success {
		t.Fatal("expected failure")
	}
	paths := map[string]bool{}
	for _, iss := range res.Error.Issues {
		paths[ToDotPath(iss.Path)] = true
	}
	if !paths["name"] || !paths["age"] {
		t.Fatalf("paths = %#v issues=%+v", paths, res.Error.Issues)
	}
}

func TestParityErrorJSONSchemaUnsupported(t *testing.T) {
	t.Skip("json-schema not supported in go-zod")
}

func TestParityFlattenEmptyNil(t *testing.T) {
	got := Flatten(nil)
	if len(got.FormErrors) != 0 || len(got.FieldErrors) != 0 {
		t.Fatalf("nil flatten: %#v", got)
	}
	if Prettify(nil) != "" {
		t.Fatal("nil prettify")
	}
}
