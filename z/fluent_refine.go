package z

import (
	"math/big"
	"time"
)

//////////////////////////////////////////////////////////////////////////////
// Mid-chain Refine / SuperRefine — return the SAME concrete schema type so
// further fluent checks still chain:
//   String().Min(5).Refine(...).Email()
//
// RefineOf / SuperRefineOf / OverwriteOf in refine.go do the same job for any
// schema type (they return a *CheckedSchema[T], which ends the type-specific
// chain), so a type missing from this file is still refinable.
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

// --- NumericSchema ---

func (s *NumericSchema[T]) Refine(pred func(T) bool, params ...any) *NumericSchema[T] {
	ch := refineCheck(func(v any) bool {
		n, ok := v.(T)
		return ok && pred(n)
	}, params...)
	return s.clone(ch)
}

func (s *NumericSchema[T]) SuperRefine(fn func(T, *RefinementCtx), params ...any) *NumericSchema[T] {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		n, _ := v.(T)
		fn(n, ctx)
	}, params...)
	return s.clone(ch)
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

// --- TupleSchema ---

func (s *TupleSchema) Refine(pred func([]any) bool, params ...any) *TupleSchema {
	ch := refineCheck(func(v any) bool {
		a, ok := v.([]any)
		return ok && pred(a)
	}, params...)
	return newTuple(s.def.withChecks(ch), append([]AnySchemaLike(nil), s.items...), s.rest)
}

func (s *TupleSchema) SuperRefine(fn func([]any, *RefinementCtx), params ...any) *TupleSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		a, _ := v.([]any)
		fn(a, ctx)
	}, params...)
	return newTuple(s.def.withChecks(ch), append([]AnySchemaLike(nil), s.items...), s.rest)
}

// --- RecordSchema ---

func (s *RecordSchema) Refine(pred func(map[string]any) bool, params ...any) *RecordSchema {
	ch := refineCheck(func(v any) bool {
		m, ok := v.(map[string]any)
		return ok && pred(m)
	}, params...)
	return newRecord(s.def.withChecks(ch), s.keySchema, s.valueSchema, s.loose)
}

func (s *RecordSchema) SuperRefine(fn func(map[string]any, *RefinementCtx), params ...any) *RecordSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		m, _ := v.(map[string]any)
		fn(m, ctx)
	}, params...)
	return newRecord(s.def.withChecks(ch), s.keySchema, s.valueSchema, s.loose)
}

// --- MapSchema ---

func (s *MapSchema) Refine(pred func(map[any]any) bool, params ...any) *MapSchema {
	ch := refineCheck(func(v any) bool {
		m, ok := v.(map[any]any)
		return ok && pred(m)
	}, params...)
	return newMap(s.def.withChecks(ch), s.keySchema, s.valueSchema)
}

func (s *MapSchema) SuperRefine(fn func(map[any]any, *RefinementCtx), params ...any) *MapSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		m, _ := v.(map[any]any)
		fn(m, ctx)
	}, params...)
	return newMap(s.def.withChecks(ch), s.keySchema, s.valueSchema)
}

// --- SetSchema ---

func (s *SetSchema) Refine(pred func([]any) bool, params ...any) *SetSchema {
	ch := refineCheck(func(v any) bool {
		a, ok := v.([]any)
		return ok && pred(a)
	}, params...)
	return newSet(s.def.withChecks(ch), s.valueSchema)
}

func (s *SetSchema) SuperRefine(fn func([]any, *RefinementCtx), params ...any) *SetSchema {
	ch := superRefineCheck(func(v any, ctx *RefinementCtx) {
		a, _ := v.([]any)
		fn(a, ctx)
	}, params...)
	return newSet(s.def.withChecks(ch), s.valueSchema)
}
