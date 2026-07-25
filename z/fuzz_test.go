package z

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// The hand-written matchers replaced regexes on the hot path, so they have to
// agree with those regexes on every input, not just the seeded corpora in
// matchers_test.go. Run with: go test -run '^$' -fuzz FuzzMatchers z/
func FuzzMatchersAgreeWithRegexes(f *testing.F) {
	for _, seed := range []string{
		"", "a", "a@b.co", "a..b@c.de", ".a@b.co", "a@b", "A@B.CO",
		"550e8400-e29b-41d4-a716-446655440000", "550E8400-E29B-41D4-A716-446655440000",
		"00000000-0000-0000-0000-000000000000", "not-a-uuid",
		"1.2.3.4", "255.255.255.255", "256.1.1.1", "1.2.3",
		"2026-07-25", "2026-02-29", "2024-02-29", "2026-13-01",
		"12:34", "12:34:56", "12:34:56.789", "24:00", "12:60",
		"2026-07-25T12:34:56Z", "2026-07-25T12:34:56.123Z", "2026-07-25 12:34:56",
	} {
		f.Add(seed)
	}

	// The email regex cannot express the two lookaheads isEmail enforces, so the
	// reference composes them the way the regex-based implementation did.
	emailReference := func(s string) bool {
		if s == "" || s[0] == '.' || strings.Contains(s, "..") {
			return false
		}
		return reEmailBody.MatchString(s)
	}
	pairs := []struct {
		name      string
		fn        func(string) bool
		reference func(string) bool
	}{
		{"isEmail", isEmail, emailReference},
		{"isUUID", isUUID, reUUID.MatchString},
		{"isGUID", isGUID, reGUID.MatchString},
		{"isIPv4", isIPv4, reIPv4.MatchString},
		{"isISODate", isISODate, reDate.MatchString},
		{"isISOTimeDefault", isISOTimeDefault, timeRegexp(nil).MatchString},
		{"isISODateTimeDefault", isISODateTimeDefault, datetimeRegexp(nil, false, false).MatchString},
	}

	f.Fuzz(func(t *testing.T, s string) {
		for _, p := range pairs {
			if got, want := p.fn(s), p.reference(s); got != want {
				t.Fatalf("%s(%q) = %v, reference = %v", p.name, s, got, want)
			}
		}
	})
}

// Parsing arbitrary JSON must never panic and must never report success with a
// value that violates the schema's own edge.
func FuzzParseJSONNeverPanics(f *testing.F) {
	for _, seed := range []string{
		`{}`, `null`, `[]`, `{"name":"Ada","age":36}`, `{"age":"x"}`,
		`{"name":"","tags":[1,2,3],"nested":{"id":"550e8400-e29b-41d4-a716-446655440000"}}`,
		`{"age":1e309}`, `{"age":9007199254740993}`, `{"name":null}`,
		`{"tags":{"not":"an array"}}`, `[[[[[[[[[[]]]]]]]]]]`,
	} {
		f.Add(seed)
	}

	schema := Object(Shape{
		"name": String().Min(1).Max(64),
		"age":  Optional(Int().Gte(0).Lte(150)),
		"tags": Default(Array(String().NonEmpty()).Max(8), []any{}),
		"nested": Optional(Object(Shape{
			"id":   String().UUID(),
			"when": Optional(String().ISODateTime()),
		})),
	}).Strict()

	f.Fuzz(func(t *testing.T, data string) {
		var input any
		if err := json.Unmarshal([]byte(data), &input); err != nil {
			return // not our problem: the fuzzer found invalid JSON
		}
		res := schema.SafeParse(input)
		if !res.Success {
			if len(res.Error.Issues) == 0 {
				t.Fatal("failure with no issues")
			}
			for _, iss := range res.Error.Issues {
				if iss.Code == "" {
					t.Fatalf("issue with no code: %+v", iss)
				}
				if strings.TrimSpace(iss.Message) == "" {
					t.Fatalf("issue with no message: %+v", iss)
				}
			}
			// The error renderers run on whatever the parse produced.
			_ = Prettify(res.Error)
			_ = Treeify(res.Error)
			_ = Flatten(res.Error)
			return
		}
		// On success the output must satisfy the declared edges.
		out := res.Data
		name, ok := out["name"].(string)
		if !ok || len(name) == 0 {
			t.Fatalf("name edge violated: %#v", out["name"])
		}
		if age, present := out["age"]; present && age != nil && !IsMissing(age) {
			n, ok := age.(int)
			if !ok {
				t.Fatalf("age should be an int, got %T", age)
			}
			if n < 0 || n > 150 {
				t.Fatalf("age %d passed a 0..150 schema", n)
			}
		}
		if _, ok := out["tags"].([]any); !ok {
			t.Fatalf("tags should default to a slice, got %T", out["tags"])
		}
	})
}

// A number schema must classify every float without panicking, and must never
// accept NaN or an infinity.
func FuzzNumericEdges(f *testing.F) {
	f.Add(0.0)
	f.Add(1.5)
	f.Add(-0.0)
	f.Add(math.MaxFloat64)
	f.Add(float64(1 << 53))

	f.Fuzz(func(t *testing.T, v float64) {
		if res := Number().SafeParse(v); res.Success {
			if math.IsNaN(res.Data) || math.IsInf(res.Data, 0) {
				t.Fatalf("Number accepted %v", v)
			}
		} else if !math.IsNaN(v) && !math.IsInf(v, 0) {
			t.Fatalf("Number rejected the finite %v: %v", v, res.Error)
		}

		if res := Int().SafeParse(v); res.Success {
			if float64(res.Data) != math.Trunc(v) {
				t.Fatalf("Int(%v) = %d", v, res.Data)
			}
			if float64(res.Data) > MaxSafeInteger || float64(res.Data) < MinSafeInteger {
				t.Fatalf("Int accepted %v, outside the safe range", v)
			}
		}

		if res := Int64().SafeParse(v); res.Success && float64(res.Data) != math.Trunc(v) {
			t.Fatalf("Int64(%v) = %d", v, res.Data)
		}
	})
}
