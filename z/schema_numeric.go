package z

import (
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"strconv"
)

// Numeric is the set of Go numeric types a numeric schema can produce.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// NumericSchema validates a number and produces the Go type T, so a schema's
// name and its output type agree: Int() yields int, Uint32() yields uint32,
// Float32() yields float32.
//
// Input is accepted in whatever numeric form it arrives — JSON decodes numbers
// to float64, so that is the common case — and is converted to T only when the
// conversion is exact. A value T cannot hold fails instead of being silently
// truncated: a non-integral value for an integer type is invalid_type, and an
// out-of-range value is too_small/too_big against T's bounds.
type NumericSchema[T Numeric] struct {
	schemaBase[T]
	def *Def
}

// NumberSchema is the float64 numeric schema: the JSON number model.
type NumberSchema = NumericSchema[float64]

// Int64Schema is the int64 numeric schema.
type Int64Schema = NumericSchema[int64]

// Number returns a float64 schema (z.number()). Accepts any Go numeric type and
// normalizes to float64. Rejects NaN and ±Inf.
func Number(params ...any) *NumberSchema {
	return newNumeric[float64](numericDef(params), "number", false)
}

// Float64 is Number with an explicit float64 range check.
func Float64(params ...any) *NumericSchema[float64] {
	return newNumeric[float64](numericDef(params).withChecks(NumberFormat("float64", params...)), "number", false)
}

// Float32 returns a float32 schema. Values outside float32 range fail with
// invalid_format.
func Float32(params ...any) *NumericSchema[float32] {
	return newNumeric[float32](numericDef(params).withChecks(NumberFormat("float32", params...)), "number", false)
}

// Int returns an int schema: a whole number inside the JSON safe-integer range
// (±2^53-1). Non-integral or out-of-range input fails with invalid_format.
func Int(params ...any) *NumericSchema[int] {
	return newNumeric[int](numericDef(params).withChecks(NumberFormat("safeint", params...)), "number", false)
}

// Int32 returns an int32 schema.
func Int32(params ...any) *NumericSchema[int32] {
	return newNumeric[int32](numericDef(params).withChecks(NumberFormat("int32", params...)), "number", false)
}

// Uint32 returns a uint32 schema.
func Uint32(params ...any) *NumericSchema[uint32] {
	return newNumeric[uint32](numericDef(params).withChecks(NumberFormat("uint32", params...)), "number", false)
}

// Int64 returns an int64 schema covering the full 64-bit range. Unlike the
// JSON-number constructors it reports invalid_type when the input is not an
// integer, because no JSON number format corresponds to a 64-bit integer.
func Int64(params ...any) *Int64Schema {
	p := normalizeParams(params)
	def := &Def{Type: "int64", Error: p.Error, Coerce: p.Coerce}
	return newNumeric[int64](def, "int64", true)
}

// NumericOf returns a schema for any Go numeric type, including widths without
// a named constructor and named types such as `type Port uint16`. A value that
// T cannot hold exactly is an invalid_type failure, reported with T's kind:
//
//	z.NumericOf[uint16]()      // 0 … 65535
//	z.NumericOf[Port]().Gte(1) // bounds take a Port
func NumericOf[T Numeric](params ...any) *NumericSchema[T] {
	kind := reflect.TypeFor[T]().Kind().String()
	p := normalizeParams(params)
	def := &Def{Type: kind, Error: p.Error, Coerce: p.Coerce}
	return newNumeric[T](def, kind, true)
}

func numericDef(params []any) *Def {
	p := normalizeParams(params)
	return &Def{Type: "number", Error: p.Error, Coerce: p.Coerce}
}

// newNumeric builds the schema. strictType selects what happens when the input
// is a number that T cannot hold: false keeps the value as float64 so the
// attached format check can name the reason (the JSON number model), true
// reports invalid_type immediately (Int64).
func newNumeric[T Numeric](def *Def, expected string, strictType bool) *NumericSchema[T] {
	// Converter and coercer are resolved once per schema, not per parse.
	convert, coerce := numericConverter[T](), numericCoercer[T]()
	s := &NumericSchema[T]{def: def}
	s.schemaBase = newBase[T](buildInternals(def, func(p *Payload, _ *ParseCtx) {
		v := p.Value
		if def.Coerce {
			if c, ok := coerce(v); ok {
				v = c
			}
		}
		if out, ok := convert(v); ok {
			p.Value = out
			return
		}
		if f, ok := ToFloat(v); ok && !strictType && !math.IsNaN(f) && !math.IsInf(f, 0) {
			// A number T cannot represent exactly. Leave it in place: the
			// format check attached by the constructor reports why.
			p.Value = f
			return
		}
		p.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: expected,
			Input:    p.Value,
			errMap:   def.Error,
		})
	}))
	return s
}

// numericConverter returns an exact any → T conversion, reporting false when
// the input is not a number or cannot be represented in T. The output kind is
// resolved once here, so named types (type Port uint16) behave like their
// underlying kind and the parse path stays a plain closure call.
func numericConverter[T Numeric]() func(any) (T, bool) {
	var zero T
	switch kind := reflect.TypeFor[T]().Kind(); kind {
	case reflect.Float32, reflect.Float64:
		lo, hi := -math.MaxFloat64, math.MaxFloat64
		if kind == reflect.Float32 {
			lo, hi = -math.MaxFloat32, math.MaxFloat32
		}
		return func(v any) (T, bool) {
			f, ok := ToFloat(v)
			if !ok || math.IsNaN(f) || math.IsInf(f, 0) || f < lo || f > hi {
				return zero, false
			}
			if f == 0 {
				f = 0 // normalize -0
			}
			return T(f), true
		}
	default:
		lo, hi, unsigned := integerBounds[T]()
		return func(v any) (T, bool) {
			if u, ok := v.(uint64); ok && unsigned && u > math.MaxInt64 {
				// Only reachable for uint64 and uint outputs.
				if hi == math.MaxInt64 || u <= uint64(hi) {
					return T(u), true
				}
				return zero, false
			}
			n, ok := toInt64(v)
			if !ok || n < lo || n > hi {
				return zero, false
			}
			return T(n), true
		}
	}
}

// integerBounds returns the inclusive int64-domain bounds of integer type T.
// uint64 and uint are capped at MaxInt64; numericConverter handles the values
// above that separately.
func integerBounds[T Numeric]() (lo, hi int64, unsigned bool) {
	t := reflect.TypeFor[T]()
	bits := t.Bits()
	switch t.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if bits >= 64 {
			return 0, math.MaxInt64, true
		}
		return 0, int64(1)<<bits - 1, true
	default:
		if bits >= 64 {
			return math.MinInt64, math.MaxInt64, false
		}
		return -(int64(1) << (bits - 1)), int64(1)<<(bits-1) - 1, false
	}
}

// numericCoercer returns the coercion used when Coerce is set. Integer outputs
// try an exact integer parse first so large numeric strings keep every digit,
// then fall back to the float path so "33.7" still reaches the format check.
func numericCoercer[T Numeric]() func(any) (any, bool) {
	switch reflect.TypeFor[T]().Kind() {
	case reflect.Float32, reflect.Float64:
		return func(v any) (any, bool) {
			f, ok := coerceToNumber(v)
			return f, ok
		}
	default:
		return func(v any) (any, bool) {
			if n, ok := coerceToInt64(v); ok {
				return n, true
			}
			f, ok := coerceToNumber(v)
			return f, ok
		}
	}
}

// toInt64 converts any Go number to an int64 without loss.
func toInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case int32:
		return int64(x), true
	case int16:
		return int64(x), true
	case int8:
		return int64(x), true
	case uint64:
		if x > math.MaxInt64 {
			return 0, false
		}
		return int64(x), true
	case uint:
		if uint64(x) > math.MaxInt64 {
			return 0, false
		}
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint8:
		return int64(x), true
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) || x != math.Trunc(x) {
			return 0, false
		}
		// float64(MaxInt64) rounds up to 2^63, which no int64 holds, but
		// float64(MinInt64) is exactly -2^63 and does.
		if x >= math.MaxInt64 || x < math.MinInt64 {
			return 0, false
		}
		return int64(x), true
	case float32:
		return toInt64(float64(x))
	case *big.Int:
		if x == nil || !x.IsInt64() {
			return 0, false
		}
		return x.Int64(), true
	case json.Number:
		// A JSON number decoded with Decoder.UseNumber: parse it as an integer
		// so digits above 2^53 survive.
		if n, err := x.Int64(); err == nil {
			return n, true
		}
		f, err := x.Float64()
		if err != nil {
			return 0, false
		}
		return toInt64(f)
	default:
		return namedInt(v)
	}
}

func coerceToInt64(v any) (int64, bool) {
	if n, ok := toInt64(v); ok {
		return n, true
	}
	switch x := v.(type) {
	case string:
		if x == "" {
			return 0, true
		}
		n, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// coerceToNumber ports limited number coercion: strings via ParseFloat, bools
// 0/1. Returns ok=false when coercion is not attempted or fails.
func coerceToNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		f, ok := ToFloat(x)
		return f, ok
	case string:
		if x == "" {
			return 0, true // Number("") === 0 in JS
		}
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	case *big.Int:
		if x == nil {
			return 0, false
		}
		f, _ := new(big.Float).SetInt(x).Float64()
		return f, true
	default:
		return ToFloat(v)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Fluent checks — written once, valid for every T
//////////////////////////////////////////////////////////////////////////////

// clone rebuilds the schema with extra checks, preserving the output type and
// the constructor's failure mode (Def.Type is "number" only for the JSON-number
// constructors; Int64 and NumericOf name their own type).
func (s *NumericSchema[T]) clone(checks ...*Check) *NumericSchema[T] {
	if s.def.Type == "number" {
		return newNumeric[T](s.def.withChecks(checks...), "number", false)
	}
	return newNumeric[T](s.def.withChecks(checks...), s.def.Type, true)
}

// Bounds are passed with the schema's own type so the comparison stays exact:
// an int64 bound near math.MaxInt64 cannot survive a trip through float64.
func (s *NumericSchema[T]) Gt(value T, params ...any) *NumericSchema[T] {
	return s.clone(GreaterThan(numericBound(value), false, params...))
}
func (s *NumericSchema[T]) Gte(value T, params ...any) *NumericSchema[T] {
	return s.clone(GreaterThan(numericBound(value), true, params...))
}
func (s *NumericSchema[T]) Min(value T, params ...any) *NumericSchema[T] {
	return s.Gte(value, params...)
}

func (s *NumericSchema[T]) Lt(value T, params ...any) *NumericSchema[T] {
	return s.clone(LessThan(numericBound(value), false, params...))
}
func (s *NumericSchema[T]) Lte(value T, params ...any) *NumericSchema[T] {
	return s.clone(LessThan(numericBound(value), true, params...))
}
func (s *NumericSchema[T]) Max(value T, params ...any) *NumericSchema[T] {
	return s.Lte(value, params...)
}

func (s *NumericSchema[T]) Positive(params ...any) *NumericSchema[T]    { return s.Gt(0, params...) }
func (s *NumericSchema[T]) Negative(params ...any) *NumericSchema[T]    { return s.Lt(0, params...) }
func (s *NumericSchema[T]) NonPositive(params ...any) *NumericSchema[T] { return s.Lte(0, params...) }
func (s *NumericSchema[T]) NonNegative(params ...any) *NumericSchema[T] { return s.Gte(0, params...) }

func (s *NumericSchema[T]) MultipleOf(value T, params ...any) *NumericSchema[T] {
	switch {
	case numericIsUnsigned[T]():
		return s.clone(MultipleOfUint64(uint64(value), params...))
	case numericIsInteger[T]():
		return s.clone(MultipleOfInt64(int64(value), params...))
	default:
		return s.clone(MultipleOf(float64(value), params...))
	}
}

// Step is an alias of MultipleOf.
func (s *NumericSchema[T]) Step(value T, params ...any) *NumericSchema[T] {
	return s.MultipleOf(value, params...)
}

// Integer requires a whole number inside the JSON safe-integer range. It is
// meaningful on float schemas — Number().Integer() — and already implied by the
// integer constructors.
func (s *NumericSchema[T]) Integer(params ...any) *NumericSchema[T] {
	return s.clone(NumberFormat("safeint", params...))
}

// Safe is an alias of Integer.
func (s *NumericSchema[T]) Safe(params ...any) *NumericSchema[T] { return s.Integer(params...) }

// Finite is a no-op: numeric schemas already reject NaN and ±Inf.
func (s *NumericSchema[T]) Finite(params ...any) *NumericSchema[T] { return s }

// Check attaches raw checks.
func (s *NumericSchema[T]) Check(checks ...*Check) *NumericSchema[T] { return s.clone(checks...) }

// numericBound hands the check layer a bound it can compare exactly: a uint64
// for unsigned schemas (an int64 would wrap anything above MaxInt64), an int64
// for signed integers, a float64 otherwise. Named types are converted through
// their underlying kind rather than boxed as themselves.
func numericBound[T Numeric](v T) any {
	switch {
	case numericIsUnsigned[T]():
		return uint64(v)
	case numericIsInteger[T]():
		return int64(v)
	default:
		return float64(v)
	}
}

// numericIsInteger reports whether T is an integer type, without naming every
// type in the constraint: integer division truncates, float division does not.
func numericIsInteger[T Numeric]() bool {
	var three T = 3
	return three/2 == 1
}

// numericIsUnsigned reports whether T cannot hold a negative value.
func numericIsUnsigned[T Numeric]() bool {
	var zero T
	return zero-1 > 0
}
