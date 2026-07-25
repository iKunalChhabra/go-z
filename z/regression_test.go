package z

import (
	"math"
	"regexp"
	"strings"
	"testing"
)

// Each test here pins a bug that shipped once. They are grouped in one file so
// the history is legible: if one of these fails, a fix was reverted.

// RegressionBase is exported because reflect cannot allocate an embedded pointer
// to an unexported type — the same limitation encoding/json has.
type RegressionBase struct {
	Kind string `json:"kind"`
}

type unexportedBase struct {
	Kind string `json:"kind"`
}

// RegressionOuter embeds a *pointer*, which is the case reflect.FieldByIndex
// panics on when the pointer is nil.
type RegressionOuter struct {
	*RegressionBase
	Name string `json:"name"`
}

// ToStruct promoted fields through an embedded pointer but then walked the index
// path with FieldByIndex, which panics on a nil pointer — from a request body,
// through zgin.ValidateToStruct.
func TestRegressionEmbeddedPointerIsAllocated(t *testing.T) {
	out, err := ToStruct[RegressionOuter](Any()).Parse(map[string]any{"kind": "k", "name": "n"})
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.RegressionBase == nil {
		t.Fatal("the embedded pointer should have been allocated")
	}
	if out.Kind != "k" || out.Name != "n" {
		t.Fatalf("got %+v", out)
	}

	// Only the promoted field present: the pointer still has to be allocated.
	out, err = ToStruct[RegressionOuter](Any()).Parse(map[string]any{"kind": "only"})
	if err != nil || out.RegressionBase == nil || out.Kind != "only" {
		t.Fatalf("got %+v %v", out, err)
	}

	// A value embed keeps working — the first attempt at this fix broke it.
	type valueOuter struct {
		RegressionBase
		Name string `json:"name"`
	}
	v, err := ToStruct[valueOuter](Any()).Parse(map[string]any{"kind": "k", "name": "n"})
	if err != nil || v.Kind != "k" || v.Name != "n" {
		t.Fatalf("value embed: %+v %v", v, err)
	}

	// An embedded pointer to an unexported type cannot be allocated. That is a
	// reported error, not a panic — which is what it used to be.
	type unexportedOuter struct {
		*unexportedBase
		Name string `json:"name"`
	}
	res := ToStruct[unexportedOuter](Any()).SafeParse(map[string]any{"kind": "k"})
	if res.Success {
		t.Error("expected a decode error for an unexported embedded pointer")
	} else if !strings.Contains(res.Error.Issues[0].Message, "unexported") {
		t.Errorf("message should explain the limitation: %q", res.Error.Issues[0].Message)
	}
}

// Unsigned bounds were converted through int64, so a bound above MaxInt64 wrapped
// negative and the check accepted everything (or rejected everything).
func TestRegressionUnsignedBoundsDoNotWrap(t *testing.T) {
	const big = uint64(1)<<63 + 5

	if NumericOf[uint64]().Gte(big).SafeParse(uint64(7)).Success {
		t.Error("Gte(2^63+5) must reject 7")
	}
	if _, err := NumericOf[uint64]().Gte(big).Parse(big + 1); err != nil {
		t.Errorf("Gte(2^63+5) must accept 2^63+6: %v", err)
	}
	if _, err := NumericOf[uint64]().Lte(uint64(math.MaxUint64)).Parse(uint64(7)); err != nil {
		t.Errorf("Lte(MaxUint64) must accept 7: %v", err)
	}
	if NumericOf[uint64]().Lte(uint64(7)).SafeParse(big).Success {
		t.Error("Lte(7) must reject 2^63+5")
	}

	// MultipleOf needs the unsigned remainder for the same reason.
	if NumericOf[uint64]().MultipleOf(uint64(1) << 63).SafeParse(uint64(3)).Success {
		t.Error("MultipleOf(2^63) must reject 3")
	}
	if _, err := NumericOf[uint64]().MultipleOf(uint64(1) << 62).Parse(uint64(1) << 63); err != nil {
		t.Errorf("2^63 is a multiple of 2^62: %v", err)
	}

	// A bound beyond the safe-integer range reports exactly, not as a float.
	res := NumericOf[uint64]().Gte(big).SafeParse(uint64(1))
	if got, ok := res.Error.Issues[0].Minimum.(uint64); !ok || got != big {
		t.Errorf("Minimum = %#v, want uint64(%d)", res.Error.Issues[0].Minimum, big)
	}
}

// float64(math.MinInt64) is exactly -2^63 and does fit an int64; the range test
// used <= and rejected it. Only the MaxInt64 side needs the exclusive form,
// because float64(MaxInt64) rounds up to 2^63.
func TestRegressionMinInt64FromFloat(t *testing.T) {
	got, err := Int64().Parse(float64(math.MinInt64))
	if err != nil || got != math.MinInt64 {
		t.Fatalf("got %d, %v", got, err)
	}
	if Int64().SafeParse(float64(math.MaxInt64)).Success {
		t.Error("float64(MaxInt64) is 2^63, which no int64 holds")
	}
}

// JSON Schema rendered every bound through float64, rounding the large integers
// Int64 exists to carry.
func TestRegressionJSONSchemaKeepsIntegerBoundsExact(t *testing.T) {
	const bound = int64(math.MaxInt64) - 1
	doc, err := ToJSONSchema(Int64().Gte(bound))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := doc["minimum"].(int64)
	if !ok || got != bound {
		t.Fatalf("minimum = %#v (%T), want int64(%d)", doc["minimum"], doc["minimum"], bound)
	}

	// Small bounds still render as float64, the JSON number model.
	doc, err = ToJSONSchema(Number().Gte(5))
	if err != nil {
		t.Fatal(err)
	}
	if doc["minimum"] != float64(5) {
		t.Fatalf("minimum = %#v, want float64(5)", doc["minimum"])
	}
}

// OpenAPI 3.0 has one slot per side, so rewriting an exclusive bound must not
// discard a tighter inclusive one that came from another check.
func TestRegressionOpenAPIBoundMergeKeepsTheTighterBound(t *testing.T) {
	cases := []struct {
		name              string
		schema            *NumberSchema
		minimum, maximum  any
		exclMin, exclMax  any
		expectExclMinBool bool
	}{
		{name: "inclusive then exclusive minimum", schema: Number().Gte(0).Gt(5), minimum: float64(5), exclMin: true},
		{name: "exclusive then inclusive minimum", schema: Number().Gt(5).Gte(0), minimum: float64(5), exclMin: true},
		{name: "inclusive minimum is tighter", schema: Number().Gt(0).Gte(5), minimum: float64(5), exclMin: nil},
		{name: "maximum side", schema: Number().Lte(100).Lt(50), maximum: float64(50), exclMax: true},
		{name: "inclusive maximum is tighter", schema: Number().Lt(100).Lte(50), maximum: float64(50), exclMax: nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc, err := ToJSONSchema(c.schema, ToJSONSchemaOpts{Target: JSONSchemaOpenAPI30})
			if err != nil {
				t.Fatal(err)
			}
			if c.minimum != nil && doc["minimum"] != c.minimum {
				t.Errorf("minimum = %#v, want %#v", doc["minimum"], c.minimum)
			}
			if c.maximum != nil && doc["maximum"] != c.maximum {
				t.Errorf("maximum = %#v, want %#v", doc["maximum"], c.maximum)
			}
			if doc["exclusiveMinimum"] != c.exclMin {
				t.Errorf("exclusiveMinimum = %#v, want %#v", doc["exclusiveMinimum"], c.exclMin)
			}
			if doc["exclusiveMaximum"] != c.exclMax {
				t.Errorf("exclusiveMaximum = %#v, want %#v", doc["exclusiveMaximum"], c.exclMax)
			}
		})
	}
}

// A raw *regexp.Regexp part took a different path from a String().Regex() part
// and skipped the anchor handling, so a pattern with inner anchors compiled into
// something that silently matched nothing.
func TestRegressionRawRegexpPartsAreTreatedLikeSchemaParts(t *testing.T) {
	raw := TemplateLiteral([]any{"pet-", regexp.MustCompile(`^(?:cat|dog)$`)})
	if _, err := raw.Parse("pet-cat"); err != nil {
		t.Errorf("pet-cat should match: %v", err)
	}
	if _, err := raw.Parse("pet-dog"); err != nil {
		t.Errorf("pet-dog should match: %v", err)
	}
	if raw.SafeParse("pet-fox").Success {
		t.Error("pet-fox should not match")
	}

	// Alternation keeps its precedence: without grouping, "^pet-cat|dog$" would
	// match a bare "dog".
	alt := TemplateLiteral([]any{"pet-", regexp.MustCompile(`cat|dog`)})
	if _, err := alt.Parse("pet-dog"); err != nil {
		t.Errorf("pet-dog should match: %v", err)
	}
	if alt.SafeParse("dog").Success {
		t.Error("a bare dog should not match")
	}

	// Inner anchors are reported for a raw regexp too.
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("inner anchors should be reported at construction")
			}
			if msg, ok := r.(string); !ok || !strings.Contains(msg, "anchors in the middle") {
				t.Fatalf("unhelpful message: %v", r)
			}
		}()
		TemplateLiteral([]any{"pet-", regexp.MustCompile(`^cat$|^dog$`)})
	}()
}

// Trimming the outer "$" must not eat an escaped dollar sign, which would leave a
// dangling backslash and fail construction with a raw regexp error.
func TestRegressionEscapedDollarSurvivesAnchorTrim(t *testing.T) {
	for _, part := range []any{
		regexp.MustCompile(`^a\$`),
		String().Regex(regexp.MustCompile(`^a\$`)),
		regexp.MustCompile(`^a\$$`),
	} {
		schema := TemplateLiteral([]any{"p", part})
		if _, err := schema.Parse("pa$"); err != nil {
			t.Errorf("%T: pa$ should match: %v", part, err)
		}
		if schema.SafeParse("pa").Success {
			t.Errorf("%T: pa should not match", part)
		}
	}
}

// The discriminated-union detail was English-only: the other locales returned a
// bare "invalid input" for the same issue.
func TestRegressionDiscriminatorDetailIsTranslated(t *testing.T) {
	detailed := Issue{Code: IssueInvalidUnion, Discriminator: "type", Values: []any{"a", "b"}}
	generic := Issue{Code: IssueInvalidUnion}

	for lang, render := range localeFns {
		withValues, withoutValues := detailed, generic
		detail := render(&withValues)
		fallback := render(&withoutValues)
		if detail == fallback {
			t.Errorf("locale %s renders the discriminator detail as the generic message %q", lang, detail)
		}
		if !strings.Contains(detail, "a") || !strings.Contains(detail, "b") {
			t.Errorf("locale %s omits the expected values: %q", lang, detail)
		}
	}
}

// ParseAny replaced a per-request identity Transform in zgin; ObjectShapeOf is
// what lets a binder see an object's fields through wrappers.
func TestRegressionErasedParseAndShapeIntrospection(t *testing.T) {
	schema := Object(Shape{"a": String().Min(1)})

	out, err := ParseAny(schema, map[string]any{"a": "x"})
	if err != nil || out.(map[string]any)["a"] != "x" {
		t.Fatalf("ParseAny: %#v %v", out, err)
	}
	if _, err := ParseAny(schema, map[string]any{"a": ""}); err == nil {
		t.Error("ParseAny must still report validation failures")
	}
	if out, err := ParseAny(nil, "untouched"); err != nil || out != "untouched" {
		t.Errorf("ParseAny(nil): %#v %v", out, err)
	}

	for _, wrapped := range []AnySchemaLike{
		schema,
		Optional(schema),
		Default(schema, map[string]any{}),
		ToStruct[struct {
			A string `json:"a"`
		}](schema),
		Lazy(func() AnySchemaLike { return schema }),
	} {
		shape, ok := ObjectShapeOf(wrapped)
		if !ok || shape["a"] == nil {
			t.Errorf("ObjectShapeOf(%T) = %#v, %v", wrapped, shape, ok)
		}
	}
	if _, ok := ObjectShapeOf(String()); ok {
		t.Error("a string schema has no object shape")
	}
}
