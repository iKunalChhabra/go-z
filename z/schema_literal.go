package z

import (
	"math"
	"math/big"
)

// LiteralSchema is the Go port of Literal / Literal (Schema[any]).
type LiteralSchema struct {
	schemaBase[any]
	def    *Def
	values []any
}

// Literal returns a literal schema (z.literal(...)). Multi-value OK.
// A trailing string / ErrorMap / Params is treated as schema params, not a value.
func Literal(values ...any) *LiteralSchema {
	vals, p := splitLiteralArgs(values)
	if len(vals) == 0 {
		panic("go-z: Literal requires at least one value")
	}
	// also accepts a single array argument as the value list.
	if len(vals) == 1 {
		switch x := vals[0].(type) {
		case []any:
			vals = append([]any(nil), x...)
		case []string:
			vals = make([]any, len(x))
			for i, s := range x {
				vals[i] = s
			}
		}
		if len(vals) == 0 {
			panic("go-z: Literal requires at least one value")
		}
	}
	def := &Def{Type: "literal", Error: p.Error}
	return newLiteral(def, vals)
}

func splitLiteralArgs(args []any) (values []any, p Params) {
	values = args
	for len(values) > 0 {
		last := values[len(values)-1]
		switch last.(type) {
		case Params, *Params, ErrorMap, string:
			// Only treat as params if there is at least one preceding value,
			// OR the sole arg is Params/*Params/ErrorMap (not a string literal).
			if len(values) == 1 {
				if _, ok := last.(string); ok {
					return values, Params{}
				}
			}
			p = normalizeParams([]any{last})
			values = values[:len(values)-1]
			// Merge any further trailing params.
			for len(values) > 0 {
				last = values[len(values)-1]
				switch last.(type) {
				case Params, *Params, ErrorMap:
					np := normalizeParams([]any{last})
					mergeParams(&p, &np)
					values = values[:len(values)-1]
				default:
					return values, p
				}
			}
			return values, p
		default:
			return values, Params{}
		}
	}
	return values, Params{}
}

func newLiteral(def *Def, values []any) *LiteralSchema {
	s := &LiteralSchema{def: def, values: values}
	in := buildInternals(def, makeLiteralParse(def, values))
	in.Values = make(map[any]struct{}, len(values))
	for _, v := range values {
		in.Values[v] = struct{}{}
		// Also index float64 form of integral numbers for discriminant support
		// (JSON numbers arrive as float64).
		if f, ok := ToFloat(v); ok && IsIntegral(f) {
			in.Values[f] = struct{}{}
		}
	}
	s.schemaBase = newBase[any](in)
	return s
}

func makeLiteralParse(def *Def, values []any) ParseFn {
	return func(p *Payload, _ *ParseCtx) {
		for _, v := range values {
			if literalEqual(p.Value, v) {
				p.Value = v
				return
			}
		}
		p.AddIssue(Issue{
			Code:   IssueInvalidValue,
			Values: append([]any(nil), values...),
			Input:  p.Value,
			errMap: def.Error,
		})
	}
}

func literalEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == b {
		return true
	}
	// *big.Int
	if ai, ok := a.(*big.Int); ok {
		if bi, ok := b.(*big.Int); ok {
			if ai == nil || bi == nil {
				return ai == bi
			}
			return ai.Cmp(bi) == 0
		}
	}
	// Cross-type numeric equality (int ↔ float64 from JSON).
	fa, oka := ToFloat(a)
	fb, okb := ToFloat(b)
	if oka && okb {
		if math.IsNaN(fa) && math.IsNaN(fb) {
			return true
		}
		return fa == fb
	}
	return false
}

// Values returns the accepted literal values.
func (s *LiteralSchema) Values() []any { return append([]any(nil), s.values...) }

// Value returns the sole literal value; panics if multiple.
func (s *LiteralSchema) Value() any {
	if len(s.values) != 1 {
		panic("go-z: LiteralSchema.Value: multiple values; use Values()")
	}
	return s.values[0]
}

// Check attaches raw checks.
func (s *LiteralSchema) Check(checks ...*Check) *LiteralSchema {
	return newLiteral(s.def.withChecks(checks...), s.values)
}
