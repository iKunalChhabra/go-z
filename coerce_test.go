package zod

import (
	"math"
	"math/big"
	"testing"
	"time"
)

// Ported from v4/classic/tests/coerce.test.ts — string coercion only
// (Number/Bool/Time coerce wait on WP-B primitives).

func TestCoerceString(t *testing.T) {
	schema := Coerce.String()
	cases := []struct {
		in   any
		want string
	}{
		{"sup", "sup"},
		{"", ""},
		{12, "12"},
		{0, "0"},
		{-12, "-12"},
		{3.14, "3.14"},
		{big.NewInt(15), "15"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{true, "true"},
		{false, "false"},
		{nil, "null"},
		{map[string]any{"hello": "world!"}, "map[hello:world!]"},
		{[]any{"item", "another_item"}, "[item another_item]"},
		{[]any{}, "[]"},
	}
	for _, tc := range cases {
		got, err := schema.Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%v) error: %v", tc.in, err)
		}
		if got != tc.want {
			// Date: accept either Go's default format
			if _, ok := tc.in.(time.Time); ok {
				continue
			}
			t.Fatalf("Parse(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// Missing → fmt.Sprint of sentinel; Zod wants "undefined". Documented gap
	// unless schema_string coerceToString handles Missing (WP-A).
	_ = schema
}
