package z

import (
	"reflect"
	"strings"
	"testing"
)

// Ported shapes from classic/tests/error-utils.test.ts (flatten/format/treeify/prettify/toDotPath).

func TestFlattenFormAndFieldErrors(t *testing.T) {
	err := &Error{Issues: []Issue{
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
	err := &Error{Issues: []Issue{
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
	err := &Error{Issues: []Issue{
		{Code: IssueInvalidType, Path: []any{"a"}, Message: "hello"},
	}}
	got := FlattenMap(err, func(iss Issue) int { return len(iss.Message) })
	if !reflect.DeepEqual(got.FieldErrors["a"], []int{5}) {
		t.Fatalf("mapped: %v", got.FieldErrors["a"])
	}
}

func TestFormatNested(t *testing.T) {
	err := &Error{Issues: []Issue{
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
	err := &Error{Issues: []Issue{
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
	err := &Error{Issues: []Issue{
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
	err := &Error{Issues: []Issue{
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
	err := &Error{Issues: []Issue{
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
	err := &Error{Issues: []Issue{
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

// A path index is a slot in a slice, so reaching a huge one used to allocate one
// pointer per skipped index. An index that cannot be dense — there are not enough
// issues to fill it — is recorded as a property instead.
func TestTreeifyBoundsSparseIndexAllocation(t *testing.T) {
	err := &Error{Issues: []Issue{{
		Code:    IssueCustom,
		Path:    []any{"items", 1 << 40},
		Message: "far out",
	}}}

	tree := Treeify(err)
	items := tree.Properties["items"]
	if items == nil {
		t.Fatal("missing items node")
	}
	if len(items.Items) != 0 {
		t.Fatalf("Items was grown to %d entries", len(items.Items))
	}
	node := items.Properties["1099511627776"]
	if node == nil || len(node.Errors) != 1 || node.Errors[0] != "far out" {
		t.Fatalf("issue was lost: %#v", items.Properties)
	}
}

// Indices that can be dense still land in Items.
func TestTreeifyKeepsRealisticIndicesInItems(t *testing.T) {
	err := &Error{Issues: []Issue{
		{Code: IssueCustom, Path: []any{"tags", 0}, Message: "first"},
		{Code: IssueCustom, Path: []any{"tags", 3}, Message: "fourth"},
	}}
	tree := Treeify(err)
	tags := tree.Properties["tags"]
	if tags == nil || len(tags.Items) != 4 {
		t.Fatalf("Items = %#v", tags)
	}
	if tags.Items[0].Errors[0] != "first" || tags.Items[3].Errors[0] != "fourth" {
		t.Fatalf("wrong placement: %#v", tags.Items)
	}
	if tags.Items[1] != nil || tags.Items[2] != nil {
		t.Error("gaps should stay nil")
	}
}

// TreeifyMap shares its implementation with Treeify, so a mapper sees the same
// structure with its own leaf type.
func TestTreeifyMapStructureMatchesTreeify(t *testing.T) {
	res := Object(Shape{
		"name": String().Min(3),
		"tags": Array(String()).Min(2),
	}).SafeParse(map[string]any{"name": "ab", "tags": []any{"x"}})
	if res.Success {
		t.Fatal("expected failures")
	}

	codes := TreeifyMap(res.Error, func(iss Issue) IssueCode { return iss.Code })
	strings := Treeify(res.Error)

	if len(codes.Properties) != len(strings.Properties) {
		t.Fatalf("shape differs: %d vs %d", len(codes.Properties), len(strings.Properties))
	}
	if got := codes.Properties["name"].Errors; len(got) != 1 || got[0] != IssueTooSmall {
		t.Fatalf("name errors = %#v", got)
	}
	if got := strings.Properties["name"].Errors; len(got) != 1 || got[0] == "" {
		t.Fatalf("name messages = %#v", got)
	}
}
