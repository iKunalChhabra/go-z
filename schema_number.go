package zod

import (
	"math"
	"math/big"
	"strconv"
)

// NumberSchema is the Go port of ZodNumber / $ZodNumber (Schema[float64]).
type NumberSchema struct {
	schemaBase[float64]
	def *Def
}

// Number returns a number schema (z.number()). Accepts float64/float32/int*/uint*
// and normalizes to float64. Rejects NaN and ±Inf (Zod v4).
func Number(params ...any) *NumberSchema {
	p := normalizeParams(params)
	def := &Def{Type: "number", Error: p.Error, Coerce: p.Coerce}
	return newNumber(def)
}

// Int returns a number schema that requires a safe integer (z.int() =
// number + safeint format check). Output type is float64 (JSON number model).
// Prefer Int64 when you want a typed Go int64 edge.
func Int(params ...any) *NumberSchema {
	p := normalizeParams(params)
	def := &Def{Type: "number", Error: p.Error, Coerce: p.Coerce}
	return newNumber(def.withChecks(NumberFormat("safeint", params...)))
}

// Int32 / Uint32 / Float32 / Float64 are number format constructors.
func Int32(params ...any) *NumberSchema {
	p := normalizeParams(params)
	def := &Def{Type: "number", Error: p.Error, Coerce: p.Coerce}
	return newNumber(def.withChecks(NumberFormat("int32", params...)))
}
func Uint32(params ...any) *NumberSchema {
	p := normalizeParams(params)
	def := &Def{Type: "number", Error: p.Error, Coerce: p.Coerce}
	return newNumber(def.withChecks(NumberFormat("uint32", params...)))
}
func Float32(params ...any) *NumberSchema {
	p := normalizeParams(params)
	def := &Def{Type: "number", Error: p.Error, Coerce: p.Coerce}
	return newNumber(def.withChecks(NumberFormat("float32", params...)))
}
func Float64(params ...any) *NumberSchema {
	p := normalizeParams(params)
	def := &Def{Type: "number", Error: p.Error, Coerce: p.Coerce}
	return newNumber(def.withChecks(NumberFormat("float64", params...)))
}

func newNumber(def *Def) *NumberSchema {
	s := &NumberSchema{def: def}
	s.schemaBase = newBase[float64](buildInternals(def, makeNumberParse(def)))
	return s
}

func makeNumberParse(def *Def) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		if def.Coerce {
			if c, ok := coerceToNumber(p.Value); ok {
				p.Value = c
			}
		}
		f, ok := ToFloat(p.Value)
		if ok && !math.IsNaN(f) && !math.IsInf(f, 0) {
			if f == 0 {
				f = 0 // normalize -0
			}
			p.Value = f
			return
		}
		p.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "number",
			Input:    p.Value,
			errMap:   def.Error,
		})
	}
}

// coerceToNumber ports limited Zod number coercion: strings via ParseFloat,
// bools 0/1. Returns ok=false when coercion is not attempted / fails.
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
		return 0, false
	}
}

//////////////////////////////////////////////////////////////////////////////
// Fluent checks
//////////////////////////////////////////////////////////////////////////////

func (s *NumberSchema) Gt(value float64, params ...any) *NumberSchema {
	return newNumber(s.def.withChecks(Gt(value, params...)))
}
func (s *NumberSchema) Gte(value float64, params ...any) *NumberSchema {
	return newNumber(s.def.withChecks(Gte(value, params...)))
}
func (s *NumberSchema) Min(value float64, params ...any) *NumberSchema {
	return s.Gte(value, params...)
}

func (s *NumberSchema) Lt(value float64, params ...any) *NumberSchema {
	return newNumber(s.def.withChecks(Lt(value, params...)))
}
func (s *NumberSchema) Lte(value float64, params ...any) *NumberSchema {
	return newNumber(s.def.withChecks(Lte(value, params...)))
}
func (s *NumberSchema) Max(value float64, params ...any) *NumberSchema {
	return s.Lte(value, params...)
}

func (s *NumberSchema) Positive(params ...any) *NumberSchema    { return s.Gt(0, params...) }
func (s *NumberSchema) Negative(params ...any) *NumberSchema    { return s.Lt(0, params...) }
func (s *NumberSchema) NonPositive(params ...any) *NumberSchema { return s.Lte(0, params...) }
func (s *NumberSchema) NonNegative(params ...any) *NumberSchema { return s.Gte(0, params...) }

func (s *NumberSchema) MultipleOf(value float64, params ...any) *NumberSchema {
	return newNumber(s.def.withChecks(MultipleOf(value, params...)))
}

// Step is an alias of MultipleOf (deprecated in Zod).
func (s *NumberSchema) Step(value float64, params ...any) *NumberSchema {
	return s.MultipleOf(value, params...)
}

// Int attaches a safeint number-format check (z.number().int()).
func (s *NumberSchema) Int(params ...any) *NumberSchema {
	return newNumber(s.def.withChecks(NumberFormat("safeint", params...)))
}

// Safe is identical to Int in Zod v4.
func (s *NumberSchema) Safe(params ...any) *NumberSchema { return s.Int(params...) }

// Finite is a no-op in Zod v4 (numbers already reject Inf).
func (s *NumberSchema) Finite(params ...any) *NumberSchema { return s }

// Check attaches raw checks.
func (s *NumberSchema) Check(checks ...*Check) *NumberSchema {
	return newNumber(s.def.withChecks(checks...))
}

//////////////////////////////////////////////////////////////////////////////
// BigInt
//////////////////////////////////////////////////////////////////////////////

// BigIntSchema is the Go port of ZodBigInt (Schema[*big.Int]).
type BigIntSchema struct {
	schemaBase[*big.Int]
	def *Def
}

// BigInt returns a bigint schema (z.bigint()). Accepts *big.Int and int64.
func BigInt(params ...any) *BigIntSchema {
	p := normalizeParams(params)
	def := &Def{Type: "bigint", Error: p.Error, Coerce: p.Coerce}
	return newBigInt(def)
}

func newBigInt(def *Def) *BigIntSchema {
	s := &BigIntSchema{def: def}
	s.schemaBase = newBase[*big.Int](buildInternals(def, makeBigIntParse(def)))
	return s
}

func makeBigIntParse(def *Def) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		if def.Coerce {
			if c, ok := coerceToBigInt(p.Value); ok {
				p.Value = c
			}
		}
		if bi, ok := asBigInt(p.Value); ok {
			p.Value = bi
			return
		}
		p.AddIssue(Issue{
			Code:     IssueInvalidType,
			Expected: "bigint",
			Input:    p.Value,
			errMap:   def.Error,
		})
	}
}

func coerceToBigInt(v any) (*big.Int, bool) {
	switch x := v.(type) {
	case *big.Int:
		if x == nil {
			return nil, false
		}
		return new(big.Int).Set(x), true
	case int64:
		return big.NewInt(x), true
	case int:
		return big.NewInt(int64(x)), true
	case int32:
		return big.NewInt(int64(x)), true
	case string:
		if x == "" {
			return big.NewInt(0), true
		}
		n, ok := new(big.Int).SetString(x, 10)
		if !ok {
			return nil, false
		}
		return n, true
	case bool:
		if x {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) || x != math.Trunc(x) {
			return nil, false
		}
		n, _ := big.NewFloat(x).Int(nil)
		return n, true
	default:
		return nil, false
	}
}

func (s *BigIntSchema) Gt(value *big.Int, params ...any) *BigIntSchema {
	return newBigInt(s.def.withChecks(GreaterThan(value, false, params...)))
}
func (s *BigIntSchema) Gte(value *big.Int, params ...any) *BigIntSchema {
	return newBigInt(s.def.withChecks(GreaterThan(value, true, params...)))
}
func (s *BigIntSchema) Min(value *big.Int, params ...any) *BigIntSchema {
	return s.Gte(value, params...)
}

func (s *BigIntSchema) Lt(value *big.Int, params ...any) *BigIntSchema {
	return newBigInt(s.def.withChecks(LessThan(value, false, params...)))
}
func (s *BigIntSchema) Lte(value *big.Int, params ...any) *BigIntSchema {
	return newBigInt(s.def.withChecks(LessThan(value, true, params...)))
}
func (s *BigIntSchema) Max(value *big.Int, params ...any) *BigIntSchema {
	return s.Lte(value, params...)
}

func (s *BigIntSchema) Positive(params ...any) *BigIntSchema {
	return s.Gt(big.NewInt(0), params...)
}
func (s *BigIntSchema) Negative(params ...any) *BigIntSchema {
	return s.Lt(big.NewInt(0), params...)
}
func (s *BigIntSchema) NonPositive(params ...any) *BigIntSchema {
	return s.Lte(big.NewInt(0), params...)
}
func (s *BigIntSchema) NonNegative(params ...any) *BigIntSchema {
	return s.Gte(big.NewInt(0), params...)
}
func (s *BigIntSchema) MultipleOf(value *big.Int, params ...any) *BigIntSchema {
	return newBigInt(s.def.withChecks(MultipleOfBigInt(value, params...)))
}
func (s *BigIntSchema) Check(checks ...*Check) *BigIntSchema {
	return newBigInt(s.def.withChecks(checks...))
}
