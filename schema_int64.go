package zod

import (
	"math"
	"math/big"
	"strconv"
)

// Int64Schema validates integers and outputs int64 (Go-native edge).
// Unlike Int()/Number() which normalize to float64 for the JSON model,
// Int64 is for APIs that want typed integers without ToStruct.
type Int64Schema struct {
	schemaBase[int64]
	def *Def
}

// Int64 returns an int64 schema. Accepts int*/uint* (in range), float64 with
// no fractional part, and numeric strings when Coerce is set.
func Int64(params ...any) *Int64Schema {
	p := normalizeParams(params)
	def := &Def{Type: "int64", Error: p.Error, Coerce: p.Coerce}
	return newInt64(def)
}

func newInt64(def *Def) *Int64Schema {
	s := &Int64Schema{def: def}
	s.schemaBase = newBase[int64](buildInternals(def, makeInt64Parse(def)))
	return s
}

func makeInt64Parse(def *Def) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		if def.Coerce {
			if c, ok := coerceToInt64(p.Value); ok {
				p.Value = c
				return
			}
		}
		if n, ok := toInt64(p.Value); ok {
			p.Value = n
			return
		}
		p.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "int64",
			Input:    p.Value,
			errMap:   def.Error,
		})
	}
}

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
		if x > float64(math.MaxInt64) || x < float64(math.MinInt64) {
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
	default:
		return 0, false
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

func (s *Int64Schema) Gt(value int64, params ...any) *Int64Schema {
	return newInt64(s.def.withChecks(Gt(float64(value), params...)))
}
func (s *Int64Schema) Gte(value int64, params ...any) *Int64Schema {
	return newInt64(s.def.withChecks(Gte(float64(value), params...)))
}
func (s *Int64Schema) Min(value int64, params ...any) *Int64Schema { return s.Gte(value, params...) }

func (s *Int64Schema) Lt(value int64, params ...any) *Int64Schema {
	return newInt64(s.def.withChecks(Lt(float64(value), params...)))
}
func (s *Int64Schema) Lte(value int64, params ...any) *Int64Schema {
	return newInt64(s.def.withChecks(Lte(float64(value), params...)))
}
func (s *Int64Schema) Max(value int64, params ...any) *Int64Schema { return s.Lte(value, params...) }

func (s *Int64Schema) Positive(params ...any) *Int64Schema    { return s.Gt(0, params...) }
func (s *Int64Schema) Negative(params ...any) *Int64Schema    { return s.Lt(0, params...) }
func (s *Int64Schema) NonPositive(params ...any) *Int64Schema { return s.Lte(0, params...) }
func (s *Int64Schema) NonNegative(params ...any) *Int64Schema { return s.Gte(0, params...) }

func (s *Int64Schema) MultipleOf(value int64, params ...any) *Int64Schema {
	return newInt64(s.def.withChecks(MultipleOf(float64(value), params...)))
}

func (s *Int64Schema) Check(checks ...*Check) *Int64Schema {
	return newInt64(s.def.withChecks(checks...))
}
