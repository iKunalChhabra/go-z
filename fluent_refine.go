package zod

import (
	"math/big"
	"time"
)

//////////////////////////////////////////////////////////////////////////////
// Mid-chain Refine / SuperRefine — return the SAME concrete schema type so
// further fluent checks still chain:
//   String().Min(5).Refine(...).Email()
//////////////////////////////////////////////////////////////////////////////

// --- StringSchema ---

func (s *StringSchema) Refine(pred func(string) bool, params ...any) *StringSchema {
	ch := refineCheck(func(v any) bool {
		str, ok := v.(string)
		return ok && pred(str)
	}, params...)
	return newString(s.def.withChecks(ch))
}

func (s *StringSchema) SuperRefine(fn func(string, *RefinementCtx), params ...any) *StringSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		str, _ := v.(string)
		fn(str, ctx)
	}, params...)
	return newString(s.def.withChecks(ch))
}

// --- NumberSchema ---

func (s *NumberSchema) Refine(pred func(float64) bool, params ...any) *NumberSchema {
	ch := refineCheck(func(v any) bool {
		n, ok := ToFloat(v)
		return ok && pred(n)
	}, params...)
	return newNumber(s.def.withChecks(ch))
}

func (s *NumberSchema) SuperRefine(fn func(float64, *RefinementCtx), params ...any) *NumberSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		n, _ := ToFloat(v)
		fn(n, ctx)
	}, params...)
	return newNumber(s.def.withChecks(ch))
}

// --- Int64Schema ---

func (s *Int64Schema) Refine(pred func(int64) bool, params ...any) *Int64Schema {
	ch := refineCheck(func(v any) bool {
		n, ok := v.(int64)
		return ok && pred(n)
	}, params...)
	return newInt64(s.def.withChecks(ch))
}

func (s *Int64Schema) SuperRefine(fn func(int64, *RefinementCtx), params ...any) *Int64Schema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		n, _ := v.(int64)
		fn(n, ctx)
	}, params...)
	return newInt64(s.def.withChecks(ch))
}

// --- BoolSchema ---

func (s *BoolSchema) Refine(pred func(bool) bool, params ...any) *BoolSchema {
	ch := refineCheck(func(v any) bool {
		b, ok := v.(bool)
		return ok && pred(b)
	}, params...)
	return newBool(s.def.withChecks(ch))
}

func (s *BoolSchema) SuperRefine(fn func(bool, *RefinementCtx), params ...any) *BoolSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		b, _ := v.(bool)
		fn(b, ctx)
	}, params...)
	return newBool(s.def.withChecks(ch))
}

// --- TimeSchema ---

func (s *TimeSchema) Refine(pred func(time.Time) bool, params ...any) *TimeSchema {
	ch := refineCheck(func(v any) bool {
		t, ok := v.(time.Time)
		return ok && pred(t)
	}, params...)
	return newTime(s.def.withChecks(ch))
}

func (s *TimeSchema) SuperRefine(fn func(time.Time, *RefinementCtx), params ...any) *TimeSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		t, _ := v.(time.Time)
		fn(t, ctx)
	}, params...)
	return newTime(s.def.withChecks(ch))
}

// --- EnumSchema ---

func (s *EnumSchema) Refine(pred func(string) bool, params ...any) *EnumSchema {
	ch := refineCheck(func(v any) bool {
		str, ok := v.(string)
		return ok && pred(str)
	}, params...)
	return newEnum(s.def.withChecks(ch), s.options, s.entries)
}

func (s *EnumSchema) SuperRefine(fn func(string, *RefinementCtx), params ...any) *EnumSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		str, _ := v.(string)
		fn(str, ctx)
	}, params...)
	return newEnum(s.def.withChecks(ch), s.options, s.entries)
}

// --- BigIntSchema ---

func (s *BigIntSchema) Refine(pred func(*big.Int) bool, params ...any) *BigIntSchema {
	ch := refineCheck(func(v any) bool {
		bi, ok := asBigInt(v)
		return ok && pred(bi)
	}, params...)
	return newBigInt(s.def.withChecks(ch))
}

func (s *BigIntSchema) SuperRefine(fn func(*big.Int, *RefinementCtx), params ...any) *BigIntSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		bi, _ := asBigInt(v)
		fn(bi, ctx)
	}, params...)
	return newBigInt(s.def.withChecks(ch))
}

// --- ObjectSchema ---

func (s *ObjectSchema) Refine(pred func(map[string]any) bool, params ...any) *ObjectSchema {
	ch := refineCheck(func(v any) bool {
		m, ok := v.(map[string]any)
		return ok && pred(m)
	}, params...)
	return newObject(s.def.withChecks(ch), cloneShape(s.shape), s.fieldOrder, s.mode, s.catchall)
}

func (s *ObjectSchema) SuperRefine(fn func(map[string]any, *RefinementCtx), params ...any) *ObjectSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		m, _ := v.(map[string]any)
		fn(m, ctx)
	}, params...)
	return newObject(s.def.withChecks(ch), cloneShape(s.shape), s.fieldOrder, s.mode, s.catchall)
}

// --- ArraySchema ---

func (s *ArraySchema) Refine(pred func([]any) bool, params ...any) *ArraySchema {
	ch := refineCheck(func(v any) bool {
		a, ok := v.([]any)
		return ok && pred(a)
	}, params...)
	return newArray(s.def.withChecks(ch), s.element)
}

func (s *ArraySchema) SuperRefine(fn func([]any, *RefinementCtx), params ...any) *ArraySchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		a, _ := v.([]any)
		fn(a, ctx)
	}, params...)
	return newArray(s.def.withChecks(ch), s.element)
}
