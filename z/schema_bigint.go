package z

import (
	"math"
	"math/big"
)

// BigIntSchema is the arbitrary-precision integer schema (Schema[*big.Int]).
type BigIntSchema struct {
	schemaBase[*big.Int]
	def *Def
}

// BigInt returns a bigint schema (z.bigint()). Accepts *big.Int and any Go
// integer. The parsed value is a fresh *big.Int, so a caller that keeps and
// mutates the input cannot change what this schema validated.
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
			// Copy: the caller keeps a pointer to their *big.Int, and mutating it
			// afterwards would change a value this schema already validated.
			p.Value = new(big.Int).Set(bi)
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
