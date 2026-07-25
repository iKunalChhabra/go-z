package zod

import (
	"math/big"
	"time"
)

//////////////////////////////////////////////////////////////////////////////
// Fluent terminators — wrap package-level Optional/Default/etc. so Zod-style
// chaining works: String().Min(5).Email().Optional()
//////////////////////////////////////////////////////////////////////////////

// --- StringSchema ---

func (s *StringSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *StringSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *StringSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *StringSchema) Default(v string) *DefaultSchema        { return Default(s, v) }
func (s *StringSchema) Prefault(v string) *PrefaultSchema      { return Prefault(s, v) }
func (s *StringSchema) Catch(v string) *CatchSchema            { return Catch(s, v) }
func (s *StringSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- NumberSchema ---

func (s *NumberSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *NumberSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *NumberSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *NumberSchema) Default(v float64) *DefaultSchema       { return Default(s, v) }
func (s *NumberSchema) Prefault(v float64) *PrefaultSchema     { return Prefault(s, v) }
func (s *NumberSchema) Catch(v float64) *CatchSchema           { return Catch(s, v) }
func (s *NumberSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- BoolSchema ---

func (s *BoolSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *BoolSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *BoolSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *BoolSchema) Default(v bool) *DefaultSchema          { return Default(s, v) }
func (s *BoolSchema) Prefault(v bool) *PrefaultSchema        { return Prefault(s, v) }
func (s *BoolSchema) Catch(v bool) *CatchSchema              { return Catch(s, v) }
func (s *BoolSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- TimeSchema ---

func (s *TimeSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *TimeSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *TimeSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *TimeSchema) Default(v time.Time) *DefaultSchema     { return Default(s, v) }
func (s *TimeSchema) Prefault(v time.Time) *PrefaultSchema   { return Prefault(s, v) }
func (s *TimeSchema) Catch(v time.Time) *CatchSchema         { return Catch(s, v) }
func (s *TimeSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- EnumSchema ---

func (s *EnumSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *EnumSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *EnumSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *EnumSchema) Default(v string) *DefaultSchema        { return Default(s, v) }
func (s *EnumSchema) Prefault(v string) *PrefaultSchema      { return Prefault(s, v) }
func (s *EnumSchema) Catch(v string) *CatchSchema            { return Catch(s, v) }
func (s *EnumSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- BigIntSchema ---

func (s *BigIntSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *BigIntSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *BigIntSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *BigIntSchema) Default(v *big.Int) *DefaultSchema      { return Default(s, v) }
func (s *BigIntSchema) Prefault(v *big.Int) *PrefaultSchema    { return Prefault(s, v) }
func (s *BigIntSchema) Catch(v *big.Int) *CatchSchema          { return Catch(s, v) }
func (s *BigIntSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- ObjectSchema ---

func (s *ObjectSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *ObjectSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *ObjectSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *ObjectSchema) Default(v map[string]any) *DefaultSchema {
	return Default(s, v)
}
func (s *ObjectSchema) Prefault(v map[string]any) *PrefaultSchema {
	return Prefault(s, v)
}
func (s *ObjectSchema) Catch(v map[string]any) *CatchSchema { return Catch(s, v) }
func (s *ObjectSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- ArraySchema ---

func (s *ArraySchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *ArraySchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *ArraySchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *ArraySchema) Default(v []any) *DefaultSchema         { return Default(s, v) }
func (s *ArraySchema) Prefault(v []any) *PrefaultSchema       { return Prefault(s, v) }
func (s *ArraySchema) Catch(v []any) *CatchSchema             { return Catch(s, v) }
func (s *ArraySchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- UnionSchema ---

func (s *UnionSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *UnionSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *UnionSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *UnionSchema) Default(v any) *DefaultSchema           { return Default(s, v) }
func (s *UnionSchema) Prefault(v any) *PrefaultSchema         { return Prefault(s, v) }
func (s *UnionSchema) Catch(v any) *CatchSchema               { return Catch(s, v) }
func (s *UnionSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}

// --- LiteralSchema ---

func (s *LiteralSchema) Optional(params ...any) *OptionalSchema { return Optional(s, params...) }
func (s *LiteralSchema) Nullable(params ...any) *NullableSchema { return Nullable(s, params...) }
func (s *LiteralSchema) Nullish(params ...any) *OptionalSchema  { return Nullish(s, params...) }
func (s *LiteralSchema) Default(v any) *DefaultSchema           { return Default(s, v) }
func (s *LiteralSchema) Prefault(v any) *PrefaultSchema         { return Prefault(s, v) }
func (s *LiteralSchema) Catch(v any) *CatchSchema               { return Catch(s, v) }
func (s *LiteralSchema) NonOptional(params ...any) *NonOptionalSchema {
	return NonOptional(s, params...)
}
