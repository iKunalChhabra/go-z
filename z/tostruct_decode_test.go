package z_test

import (
	"strings"
	"testing"
	"time"

	"github.com/iKunalChhabra/go-z/z"
)

// The assignment paths behind ToStruct — slices, pointers, times, numeric
// conversions — carry every zgin handler that binds into a struct, and were the
// least covered code in the package.
func TestToStructAssignsEveryFieldShape(t *testing.T) {
	type Address struct {
		City string `json:"city"`
		Zip  string `json:"zip"`
	}
	type Profile struct {
		Name      string            `json:"name"`
		Nickname  *string           `json:"nickname"`
		Age       int               `json:"age"`
		Height    float32           `json:"height"`
		Port      uint16            `json:"port"`
		Big       int64             `json:"big"`
		Active    bool              `json:"active"`
		Tags      []string          `json:"tags"`
		Scores    []float64         `json:"scores"`
		Addresses []Address         `json:"addresses"`
		Primary   *Address          `json:"primary"`
		Meta      map[string]any    `json:"meta"`
		Extra     any               `json:"extra"`
		Labels    map[string]string `json:"labels"`
		Joined    time.Time         `json:"joined"`
		Seen      *time.Time        `json:"seen"`
		Skipped   string            `json:"-"`
		Omitted   string            `json:"omitted,omitempty"`
	}

	joined := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	input := map[string]any{
		"name":     "Ada",
		"nickname": "A",
		"age":      36.0, // JSON numbers arrive as float64
		"height":   1.75,
		"port":     8080.0,
		"big":      int64(9007199254740993),
		"active":   true,
		"tags":     []any{"x", "y"},
		"scores":   []any{1.0, 2.5},
		"addresses": []any{
			map[string]any{"city": "Paris", "zip": "75001"},
			map[string]any{"city": "Lyon", "zip": "69001"},
		},
		"primary": map[string]any{"city": "Paris", "zip": "75001"},
		"meta":    map[string]any{"k": "v"},
		"extra":   []any{1.0, "two"},
		"labels":  map[string]any{"env": "prod"},
		"joined":  joined.Format(time.RFC3339),
		"seen":    joined,
		"omitted": "present",
	}

	got, err := z.ToStruct[Profile](z.Any()).Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "Ada" || got.Nickname == nil || *got.Nickname != "A" {
		t.Errorf("strings/pointers: %+v", got)
	}
	if got.Age != 36 || got.Height != 1.75 || got.Port != 8080 {
		t.Errorf("numeric conversions: age=%d height=%v port=%d", got.Age, got.Height, got.Port)
	}
	if got.Big != 9007199254740993 {
		t.Errorf("int64 lost precision: %d", got.Big)
	}
	if !got.Active {
		t.Error("bool not assigned")
	}
	if len(got.Tags) != 2 || got.Tags[1] != "y" {
		t.Errorf("string slice: %#v", got.Tags)
	}
	if len(got.Scores) != 2 || got.Scores[1] != 2.5 {
		t.Errorf("float slice: %#v", got.Scores)
	}
	if len(got.Addresses) != 2 || got.Addresses[1].City != "Lyon" {
		t.Errorf("struct slice: %#v", got.Addresses)
	}
	if got.Primary == nil || got.Primary.Zip != "75001" {
		t.Errorf("struct pointer: %#v", got.Primary)
	}
	if got.Meta["k"] != "v" || got.Labels["env"] != "prod" {
		t.Errorf("maps: %#v %#v", got.Meta, got.Labels)
	}
	if extra, ok := got.Extra.([]any); !ok || len(extra) != 2 {
		t.Errorf("any field: %#v", got.Extra)
	}
	if !got.Joined.Equal(joined) {
		t.Errorf("time from string: %v", got.Joined)
	}
	if got.Seen == nil || !got.Seen.Equal(joined) {
		t.Errorf("time pointer: %v", got.Seen)
	}
	if got.Skipped != "" {
		t.Error(`a json:"-" field must stay untouched`)
	}
	if got.Omitted != "present" {
		t.Errorf("omitempty affects encoding, not decoding: %q", got.Omitted)
	}
}

// Absent and null values leave a field at its zero value rather than failing.
func TestToStructMissingAndNullFields(t *testing.T) {
	type Row struct {
		Name  string     `json:"name"`
		Count *int       `json:"count"`
		When  *time.Time `json:"when"`
		Tags  []string   `json:"tags"`
	}
	got, err := z.ToStruct[Row](z.Any()).Parse(map[string]any{
		"name":  "only",
		"count": nil,
		"when":  nil,
		"tags":  nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "only" || got.Count != nil || got.When != nil || got.Tags != nil {
		t.Fatalf("got %+v", got)
	}
}

// A value the target field cannot hold is a decode error, not a silent zero.
func TestToStructReportsUnassignableValues(t *testing.T) {
	type Row struct {
		Count int      `json:"count"`
		Tags  []string `json:"tags"`
		Inner struct {
			A string `json:"a"`
		} `json:"inner"`
	}
	cases := map[string]map[string]any{
		"scalar type":  {"count": "not a number"},
		"slice type":   {"tags": map[string]any{"not": "a slice"}},
		"nested type":  {"inner": "not an object"},
		"slice member": {"tags": []any{map[string]any{}}},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := z.ToStruct[Row](z.Any()).Parse(input); err == nil {
				t.Fatalf("expected a decode error for %#v", input)
			}
		})
	}
}

// The wrapper reports decode failures as issues, so a handler renders them the
// same way as validation failures.
func TestToStructDecodeFailureIsAnIssue(t *testing.T) {
	type Row struct {
		Count int `json:"count"`
	}
	res := z.ToStruct[Row](z.Object(z.Shape{"count": z.Any()})).SafeParse(map[string]any{"count": "x"})
	if res.Success {
		t.Fatal("expected failure")
	}
	if len(res.Error.Issues) == 0 || res.Error.Issues[0].Code == "" {
		t.Fatalf("issues = %+v", res.Error.Issues)
	}
	if !strings.Contains(z.Prettify(res.Error), "count") {
		t.Errorf("the message should name the field: %s", z.Prettify(res.Error))
	}
}
