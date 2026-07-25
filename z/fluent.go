package z

import (
	"math/big"
	"time"
)

//////////////////////////////////////////////////////////////////////////////
// Fluent terminators — wrap the generic OptionalOf/DefaultOf/… constructors so
// fluent chaining works and the typed edge survives the wrapper:
//
//	String().Min(5).Email().Optional().Parse(v)  // (*string, error)
//	String().Default("x").Parse(v)               // (string, error)
//
// These methods are sugar. The generic constructors they delegate to —
// OptionalOf, NullableOf, NullishOf, DefaultOf, PrefaultOf, CatchOf,
// NonOptionalOf, ReadonlyOf, RefineOf, SuperRefineOf, OverwriteOf — work with
// *every* schema type, including any added later, so a missing method here is
// a convenience gap and never a capability gap.
//////////////////////////////////////////////////////////////////////////////

// --- StringSchema ---

func (s *StringSchema) Optional(params ...any) *OptionalSchema[string] {
	return OptionalOf(s, params...)
}
func (s *StringSchema) Nullable(params ...any) *NullableSchema[string] {
	return NullableOf(s, params...)
}
func (s *StringSchema) Nullish(params ...any) *OptionalSchema[string] {
	return NullishOf(s, params...)
}
func (s *StringSchema) Default(v string) *DefaultSchema[string]   { return DefaultOf(s, v) }
func (s *StringSchema) Prefault(v string) *PrefaultSchema[string] { return PrefaultOf(s, v) }
func (s *StringSchema) Catch(v string) *CatchSchema[string]       { return CatchOf(s, v) }
func (s *StringSchema) NonOptional(params ...any) *NonOptionalSchema[string] {
	return NonOptionalOf(s, params...)
}

// --- NumberSchema ---

func (s *NumberSchema) Optional(params ...any) *OptionalSchema[float64] {
	return OptionalOf(s, params...)
}
func (s *NumberSchema) Nullable(params ...any) *NullableSchema[float64] {
	return NullableOf(s, params...)
}
func (s *NumberSchema) Nullish(params ...any) *OptionalSchema[float64] {
	return NullishOf(s, params...)
}
func (s *NumberSchema) Default(v float64) *DefaultSchema[float64]   { return DefaultOf(s, v) }
func (s *NumberSchema) Prefault(v float64) *PrefaultSchema[float64] { return PrefaultOf(s, v) }
func (s *NumberSchema) Catch(v float64) *CatchSchema[float64]       { return CatchOf(s, v) }
func (s *NumberSchema) NonOptional(params ...any) *NonOptionalSchema[float64] {
	return NonOptionalOf(s, params...)
}

// --- Int64Schema ---

func (s *Int64Schema) Optional(params ...any) *OptionalSchema[int64] {
	return OptionalOf(s, params...)
}
func (s *Int64Schema) Nullable(params ...any) *NullableSchema[int64] {
	return NullableOf(s, params...)
}
func (s *Int64Schema) Nullish(params ...any) *OptionalSchema[int64] {
	return NullishOf(s, params...)
}
func (s *Int64Schema) Default(v int64) *DefaultSchema[int64]   { return DefaultOf(s, v) }
func (s *Int64Schema) Prefault(v int64) *PrefaultSchema[int64] { return PrefaultOf(s, v) }
func (s *Int64Schema) Catch(v int64) *CatchSchema[int64]       { return CatchOf(s, v) }
func (s *Int64Schema) NonOptional(params ...any) *NonOptionalSchema[int64] {
	return NonOptionalOf(s, params...)
}

// --- BoolSchema ---

func (s *BoolSchema) Optional(params ...any) *OptionalSchema[bool] {
	return OptionalOf(s, params...)
}
func (s *BoolSchema) Nullable(params ...any) *NullableSchema[bool] {
	return NullableOf(s, params...)
}
func (s *BoolSchema) Nullish(params ...any) *OptionalSchema[bool] {
	return NullishOf(s, params...)
}
func (s *BoolSchema) Default(v bool) *DefaultSchema[bool]   { return DefaultOf(s, v) }
func (s *BoolSchema) Prefault(v bool) *PrefaultSchema[bool] { return PrefaultOf(s, v) }
func (s *BoolSchema) Catch(v bool) *CatchSchema[bool]       { return CatchOf(s, v) }
func (s *BoolSchema) NonOptional(params ...any) *NonOptionalSchema[bool] {
	return NonOptionalOf(s, params...)
}

// --- TimeSchema ---

func (s *TimeSchema) Optional(params ...any) *OptionalSchema[time.Time] {
	return OptionalOf(s, params...)
}
func (s *TimeSchema) Nullable(params ...any) *NullableSchema[time.Time] {
	return NullableOf(s, params...)
}
func (s *TimeSchema) Nullish(params ...any) *OptionalSchema[time.Time] {
	return NullishOf(s, params...)
}
func (s *TimeSchema) Default(v time.Time) *DefaultSchema[time.Time]   { return DefaultOf(s, v) }
func (s *TimeSchema) Prefault(v time.Time) *PrefaultSchema[time.Time] { return PrefaultOf(s, v) }
func (s *TimeSchema) Catch(v time.Time) *CatchSchema[time.Time]       { return CatchOf(s, v) }
func (s *TimeSchema) NonOptional(params ...any) *NonOptionalSchema[time.Time] {
	return NonOptionalOf(s, params...)
}

// --- EnumSchema ---

func (s *EnumSchema) Optional(params ...any) *OptionalSchema[string] {
	return OptionalOf(s, params...)
}
func (s *EnumSchema) Nullable(params ...any) *NullableSchema[string] {
	return NullableOf(s, params...)
}
func (s *EnumSchema) Nullish(params ...any) *OptionalSchema[string] {
	return NullishOf(s, params...)
}
func (s *EnumSchema) Default(v string) *DefaultSchema[string]   { return DefaultOf(s, v) }
func (s *EnumSchema) Prefault(v string) *PrefaultSchema[string] { return PrefaultOf(s, v) }
func (s *EnumSchema) Catch(v string) *CatchSchema[string]       { return CatchOf(s, v) }
func (s *EnumSchema) NonOptional(params ...any) *NonOptionalSchema[string] {
	return NonOptionalOf(s, params...)
}

// --- BigIntSchema ---

func (s *BigIntSchema) Optional(params ...any) *OptionalSchema[*big.Int] {
	return OptionalOf(s, params...)
}
func (s *BigIntSchema) Nullable(params ...any) *NullableSchema[*big.Int] {
	return NullableOf(s, params...)
}
func (s *BigIntSchema) Nullish(params ...any) *OptionalSchema[*big.Int] {
	return NullishOf(s, params...)
}
func (s *BigIntSchema) Default(v *big.Int) *DefaultSchema[*big.Int]   { return DefaultOf(s, v) }
func (s *BigIntSchema) Prefault(v *big.Int) *PrefaultSchema[*big.Int] { return PrefaultOf(s, v) }
func (s *BigIntSchema) Catch(v *big.Int) *CatchSchema[*big.Int]       { return CatchOf(s, v) }
func (s *BigIntSchema) NonOptional(params ...any) *NonOptionalSchema[*big.Int] {
	return NonOptionalOf(s, params...)
}

// --- ObjectSchema ---

func (s *ObjectSchema) Optional(params ...any) *OptionalSchema[map[string]any] {
	return OptionalOf(s, params...)
}
func (s *ObjectSchema) Nullable(params ...any) *NullableSchema[map[string]any] {
	return NullableOf(s, params...)
}
func (s *ObjectSchema) Nullish(params ...any) *OptionalSchema[map[string]any] {
	return NullishOf(s, params...)
}
func (s *ObjectSchema) Default(v map[string]any) *DefaultSchema[map[string]any] {
	return DefaultOf(s, v)
}
func (s *ObjectSchema) Prefault(v map[string]any) *PrefaultSchema[map[string]any] {
	return PrefaultOf(s, v)
}
func (s *ObjectSchema) Catch(v map[string]any) *CatchSchema[map[string]any] {
	return CatchOf(s, v)
}
func (s *ObjectSchema) NonOptional(params ...any) *NonOptionalSchema[map[string]any] {
	return NonOptionalOf(s, params...)
}

// --- ArraySchema ---

func (s *ArraySchema) Optional(params ...any) *OptionalSchema[[]any] {
	return OptionalOf(s, params...)
}
func (s *ArraySchema) Nullable(params ...any) *NullableSchema[[]any] {
	return NullableOf(s, params...)
}
func (s *ArraySchema) Nullish(params ...any) *OptionalSchema[[]any] {
	return NullishOf(s, params...)
}
func (s *ArraySchema) Default(v []any) *DefaultSchema[[]any]   { return DefaultOf(s, v) }
func (s *ArraySchema) Prefault(v []any) *PrefaultSchema[[]any] { return PrefaultOf(s, v) }
func (s *ArraySchema) Catch(v []any) *CatchSchema[[]any]       { return CatchOf(s, v) }
func (s *ArraySchema) NonOptional(params ...any) *NonOptionalSchema[[]any] {
	return NonOptionalOf(s, params...)
}

// --- TupleSchema ---

func (s *TupleSchema) Optional(params ...any) *OptionalSchema[[]any] {
	return OptionalOf(s, params...)
}
func (s *TupleSchema) Nullable(params ...any) *NullableSchema[[]any] {
	return NullableOf(s, params...)
}
func (s *TupleSchema) Nullish(params ...any) *OptionalSchema[[]any] {
	return NullishOf(s, params...)
}
func (s *TupleSchema) Default(v []any) *DefaultSchema[[]any]   { return DefaultOf(s, v) }
func (s *TupleSchema) Prefault(v []any) *PrefaultSchema[[]any] { return PrefaultOf(s, v) }
func (s *TupleSchema) Catch(v []any) *CatchSchema[[]any]       { return CatchOf(s, v) }
func (s *TupleSchema) NonOptional(params ...any) *NonOptionalSchema[[]any] {
	return NonOptionalOf(s, params...)
}

// --- RecordSchema ---

func (s *RecordSchema) Optional(params ...any) *OptionalSchema[map[string]any] {
	return OptionalOf(s, params...)
}
func (s *RecordSchema) Nullable(params ...any) *NullableSchema[map[string]any] {
	return NullableOf(s, params...)
}
func (s *RecordSchema) Nullish(params ...any) *OptionalSchema[map[string]any] {
	return NullishOf(s, params...)
}
func (s *RecordSchema) Default(v map[string]any) *DefaultSchema[map[string]any] {
	return DefaultOf(s, v)
}
func (s *RecordSchema) Prefault(v map[string]any) *PrefaultSchema[map[string]any] {
	return PrefaultOf(s, v)
}
func (s *RecordSchema) Catch(v map[string]any) *CatchSchema[map[string]any] {
	return CatchOf(s, v)
}
func (s *RecordSchema) NonOptional(params ...any) *NonOptionalSchema[map[string]any] {
	return NonOptionalOf(s, params...)
}

// --- MapSchema ---

func (s *MapSchema) Optional(params ...any) *OptionalSchema[map[any]any] {
	return OptionalOf(s, params...)
}
func (s *MapSchema) Nullable(params ...any) *NullableSchema[map[any]any] {
	return NullableOf(s, params...)
}
func (s *MapSchema) Nullish(params ...any) *OptionalSchema[map[any]any] {
	return NullishOf(s, params...)
}
func (s *MapSchema) Default(v map[any]any) *DefaultSchema[map[any]any]   { return DefaultOf(s, v) }
func (s *MapSchema) Prefault(v map[any]any) *PrefaultSchema[map[any]any] { return PrefaultOf(s, v) }
func (s *MapSchema) Catch(v map[any]any) *CatchSchema[map[any]any]       { return CatchOf(s, v) }
func (s *MapSchema) NonOptional(params ...any) *NonOptionalSchema[map[any]any] {
	return NonOptionalOf(s, params...)
}

// --- SetSchema ---

func (s *SetSchema) Optional(params ...any) *OptionalSchema[[]any] {
	return OptionalOf(s, params...)
}
func (s *SetSchema) Nullable(params ...any) *NullableSchema[[]any] {
	return NullableOf(s, params...)
}
func (s *SetSchema) Nullish(params ...any) *OptionalSchema[[]any] {
	return NullishOf(s, params...)
}
func (s *SetSchema) Default(v []any) *DefaultSchema[[]any]   { return DefaultOf(s, v) }
func (s *SetSchema) Prefault(v []any) *PrefaultSchema[[]any] { return PrefaultOf(s, v) }
func (s *SetSchema) Catch(v []any) *CatchSchema[[]any]       { return CatchOf(s, v) }
func (s *SetSchema) NonOptional(params ...any) *NonOptionalSchema[[]any] {
	return NonOptionalOf(s, params...)
}

// --- UnionSchema ---

func (s *UnionSchema) Optional(params ...any) *OptionalSchema[any] {
	return OptionalOf(s, params...)
}
func (s *UnionSchema) Nullable(params ...any) *NullableSchema[any] {
	return NullableOf(s, params...)
}
func (s *UnionSchema) Nullish(params ...any) *OptionalSchema[any] { return NullishOf(s, params...) }
func (s *UnionSchema) Default(v any) *DefaultSchema[any]          { return DefaultOf(s, v) }
func (s *UnionSchema) Prefault(v any) *PrefaultSchema[any]        { return PrefaultOf(s, v) }
func (s *UnionSchema) Catch(v any) *CatchSchema[any]              { return CatchOf(s, v) }
func (s *UnionSchema) NonOptional(params ...any) *NonOptionalSchema[any] {
	return NonOptionalOf(s, params...)
}

// --- XorSchema ---

func (s *XorSchema) Optional(params ...any) *OptionalSchema[any] { return OptionalOf(s, params...) }
func (s *XorSchema) Nullable(params ...any) *NullableSchema[any] { return NullableOf(s, params...) }
func (s *XorSchema) Nullish(params ...any) *OptionalSchema[any]  { return NullishOf(s, params...) }
func (s *XorSchema) Default(v any) *DefaultSchema[any]           { return DefaultOf(s, v) }
func (s *XorSchema) Prefault(v any) *PrefaultSchema[any]         { return PrefaultOf(s, v) }
func (s *XorSchema) Catch(v any) *CatchSchema[any]               { return CatchOf(s, v) }
func (s *XorSchema) NonOptional(params ...any) *NonOptionalSchema[any] {
	return NonOptionalOf(s, params...)
}

// --- DiscriminatedUnionSchema ---

func (s *DiscriminatedUnionSchema) Optional(params ...any) *OptionalSchema[any] {
	return OptionalOf(s, params...)
}
func (s *DiscriminatedUnionSchema) Nullable(params ...any) *NullableSchema[any] {
	return NullableOf(s, params...)
}
func (s *DiscriminatedUnionSchema) Nullish(params ...any) *OptionalSchema[any] {
	return NullishOf(s, params...)
}
func (s *DiscriminatedUnionSchema) Default(v any) *DefaultSchema[any]   { return DefaultOf(s, v) }
func (s *DiscriminatedUnionSchema) Prefault(v any) *PrefaultSchema[any] { return PrefaultOf(s, v) }
func (s *DiscriminatedUnionSchema) Catch(v any) *CatchSchema[any]       { return CatchOf(s, v) }
func (s *DiscriminatedUnionSchema) NonOptional(params ...any) *NonOptionalSchema[any] {
	return NonOptionalOf(s, params...)
}

// --- IntersectionSchema ---

func (s *IntersectionSchema) Optional(params ...any) *OptionalSchema[any] {
	return OptionalOf(s, params...)
}
func (s *IntersectionSchema) Nullable(params ...any) *NullableSchema[any] {
	return NullableOf(s, params...)
}
func (s *IntersectionSchema) Nullish(params ...any) *OptionalSchema[any] {
	return NullishOf(s, params...)
}
func (s *IntersectionSchema) Default(v any) *DefaultSchema[any]   { return DefaultOf(s, v) }
func (s *IntersectionSchema) Prefault(v any) *PrefaultSchema[any] { return PrefaultOf(s, v) }
func (s *IntersectionSchema) Catch(v any) *CatchSchema[any]       { return CatchOf(s, v) }
func (s *IntersectionSchema) NonOptional(params ...any) *NonOptionalSchema[any] {
	return NonOptionalOf(s, params...)
}

// --- LazySchema ---

func (s *LazySchema) Optional(params ...any) *OptionalSchema[any] {
	return OptionalOf(s, params...)
}
func (s *LazySchema) Nullable(params ...any) *NullableSchema[any] {
	return NullableOf(s, params...)
}
func (s *LazySchema) Nullish(params ...any) *OptionalSchema[any] { return NullishOf(s, params...) }
func (s *LazySchema) Default(v any) *DefaultSchema[any]          { return DefaultOf(s, v) }
func (s *LazySchema) Prefault(v any) *PrefaultSchema[any]        { return PrefaultOf(s, v) }
func (s *LazySchema) Catch(v any) *CatchSchema[any]              { return CatchOf(s, v) }
func (s *LazySchema) NonOptional(params ...any) *NonOptionalSchema[any] {
	return NonOptionalOf(s, params...)
}

// --- LiteralSchema ---

func (s *LiteralSchema) Optional(params ...any) *OptionalSchema[any] {
	return OptionalOf(s, params...)
}
func (s *LiteralSchema) Nullable(params ...any) *NullableSchema[any] {
	return NullableOf(s, params...)
}
func (s *LiteralSchema) Nullish(params ...any) *OptionalSchema[any] {
	return NullishOf(s, params...)
}
func (s *LiteralSchema) Default(v any) *DefaultSchema[any]   { return DefaultOf(s, v) }
func (s *LiteralSchema) Prefault(v any) *PrefaultSchema[any] { return PrefaultOf(s, v) }
func (s *LiteralSchema) Catch(v any) *CatchSchema[any]       { return CatchOf(s, v) }
func (s *LiteralSchema) NonOptional(params ...any) *NonOptionalSchema[any] {
	return NonOptionalOf(s, params...)
}

// --- CodecSchema ---

func (s *CodecSchema) Optional(params ...any) *OptionalSchema[any] {
	return OptionalOf(s, params...)
}
func (s *CodecSchema) Nullable(params ...any) *NullableSchema[any] {
	return NullableOf(s, params...)
}
func (s *CodecSchema) Nullish(params ...any) *OptionalSchema[any] { return NullishOf(s, params...) }
func (s *CodecSchema) Default(v any) *DefaultSchema[any]          { return DefaultOf(s, v) }
func (s *CodecSchema) Prefault(v any) *PrefaultSchema[any]        { return PrefaultOf(s, v) }
func (s *CodecSchema) Catch(v any) *CatchSchema[any]              { return CatchOf(s, v) }
func (s *CodecSchema) NonOptional(params ...any) *NonOptionalSchema[any] {
	return NonOptionalOf(s, params...)
}

// --- TemplateLiteralSchema ---

func (s *TemplateLiteralSchema) Optional(params ...any) *OptionalSchema[string] {
	return OptionalOf(s, params...)
}
func (s *TemplateLiteralSchema) Nullable(params ...any) *NullableSchema[string] {
	return NullableOf(s, params...)
}
func (s *TemplateLiteralSchema) Nullish(params ...any) *OptionalSchema[string] {
	return NullishOf(s, params...)
}
func (s *TemplateLiteralSchema) Default(v string) *DefaultSchema[string] { return DefaultOf(s, v) }
func (s *TemplateLiteralSchema) Prefault(v string) *PrefaultSchema[string] {
	return PrefaultOf(s, v)
}
func (s *TemplateLiteralSchema) Catch(v string) *CatchSchema[string] { return CatchOf(s, v) }
func (s *TemplateLiteralSchema) NonOptional(params ...any) *NonOptionalSchema[string] {
	return NonOptionalOf(s, params...)
}
