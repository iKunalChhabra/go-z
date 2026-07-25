package zod

//////////////////////////////////////////////////////////////////////////////
// Fluent methods on the wrapper types themselves, so chains keep composing
// after the first wrapper:
//
//	String().Optional().NonOptional().Parse(v)  // (string, error)
//	Number().Default(1).Nullable().Parse(v)     // (*float64, error)
//
// Each method is written once and works for every T, which is the payoff of
// making the wrappers generic.
//////////////////////////////////////////////////////////////////////////////

// --- Wrappers whose output is *T (Optional, Nullable) ---

// NonOptional rejects Missing again, returning to a T-valued edge.
func (s *OptionalSchema[T]) NonOptional(params ...any) *NonOptionalSchema[T] {
	return newNonOptional[T](nonOptionalDef(params), s)
}

// Default substitutes v when the value is absent, so the edge becomes T.
func (s *OptionalSchema[T]) Default(v T) *DefaultSchema[T] {
	return newDefault[T](s, func() any { return cloneDefaultValue(v) })
}

// Prefault substitutes v for absent input before the inner parse.
func (s *OptionalSchema[T]) Prefault(v T) *PrefaultSchema[T] {
	return newPrefault[T](s, func() any { return cloneDefaultValue(v) })
}

// Catch replaces any failure with v.
func (s *OptionalSchema[T]) Catch(v T) *CatchSchema[T] {
	return newCatch[T](s, func(CatchCtx) any { return cloneDefaultValue(v) })
}

// NonOptional rejects Missing, returning to a T-valued edge.
func (s *NullableSchema[T]) NonOptional(params ...any) *NonOptionalSchema[T] {
	return newNonOptional[T](nonOptionalDef(params), s)
}

// Default substitutes v when the value is absent.
func (s *NullableSchema[T]) Default(v T) *DefaultSchema[T] {
	return newDefault[T](s, func() any { return cloneDefaultValue(v) })
}

// Prefault substitutes v for absent input before the inner parse.
func (s *NullableSchema[T]) Prefault(v T) *PrefaultSchema[T] {
	return newPrefault[T](s, func() any { return cloneDefaultValue(v) })
}

// Catch replaces any failure with v.
func (s *NullableSchema[T]) Catch(v T) *CatchSchema[T] {
	return newCatch[T](s, func(CatchCtx) any { return cloneDefaultValue(v) })
}

// --- Wrappers whose output is T ---

func (s *DefaultSchema[T]) Optional(params ...any) *OptionalSchema[T] {
	return OptionalOf[T](s, params...)
}
func (s *DefaultSchema[T]) Nullable(params ...any) *NullableSchema[T] {
	return NullableOf[T](s, params...)
}
func (s *DefaultSchema[T]) Nullish(params ...any) *OptionalSchema[T] {
	return NullishOf[T](s, params...)
}
func (s *DefaultSchema[T]) Refine(pred func(T) bool, params ...any) *CheckedSchema[T] {
	return RefineOf[T](s, pred, params...)
}
func (s *DefaultSchema[T]) SuperRefine(fn func(T, *RefinementCtx), params ...any) *CheckedSchema[T] {
	return SuperRefineOf[T](s, fn, params...)
}

func (s *PrefaultSchema[T]) Optional(params ...any) *OptionalSchema[T] {
	return OptionalOf[T](s, params...)
}
func (s *PrefaultSchema[T]) Nullable(params ...any) *NullableSchema[T] {
	return NullableOf[T](s, params...)
}
func (s *PrefaultSchema[T]) Nullish(params ...any) *OptionalSchema[T] {
	return NullishOf[T](s, params...)
}
func (s *PrefaultSchema[T]) Refine(pred func(T) bool, params ...any) *CheckedSchema[T] {
	return RefineOf[T](s, pred, params...)
}
func (s *PrefaultSchema[T]) SuperRefine(fn func(T, *RefinementCtx), params ...any) *CheckedSchema[T] {
	return SuperRefineOf[T](s, fn, params...)
}

func (s *CatchSchema[T]) Optional(params ...any) *OptionalSchema[T] {
	return OptionalOf[T](s, params...)
}
func (s *CatchSchema[T]) Nullable(params ...any) *NullableSchema[T] {
	return NullableOf[T](s, params...)
}
func (s *CatchSchema[T]) Nullish(params ...any) *OptionalSchema[T] {
	return NullishOf[T](s, params...)
}
func (s *CatchSchema[T]) Refine(pred func(T) bool, params ...any) *CheckedSchema[T] {
	return RefineOf[T](s, pred, params...)
}
func (s *CatchSchema[T]) SuperRefine(fn func(T, *RefinementCtx), params ...any) *CheckedSchema[T] {
	return SuperRefineOf[T](s, fn, params...)
}

func (s *NonOptionalSchema[T]) Optional(params ...any) *OptionalSchema[T] {
	return OptionalOf[T](s, params...)
}
func (s *NonOptionalSchema[T]) Nullable(params ...any) *NullableSchema[T] {
	return NullableOf[T](s, params...)
}
func (s *NonOptionalSchema[T]) Default(v T) *DefaultSchema[T] {
	return DefaultOf[T](s, v)
}
func (s *NonOptionalSchema[T]) Catch(v T) *CatchSchema[T] {
	return CatchOf[T](s, v)
}
func (s *NonOptionalSchema[T]) Refine(pred func(T) bool, params ...any) *CheckedSchema[T] {
	return RefineOf[T](s, pred, params...)
}

func (s *ReadonlySchema[T]) Optional(params ...any) *OptionalSchema[T] {
	return OptionalOf[T](s, params...)
}
func (s *ReadonlySchema[T]) Nullable(params ...any) *NullableSchema[T] {
	return NullableOf[T](s, params...)
}
func (s *ReadonlySchema[T]) Refine(pred func(T) bool, params ...any) *CheckedSchema[T] {
	return RefineOf[T](s, pred, params...)
}

func (s *CheckedSchema[T]) Optional(params ...any) *OptionalSchema[T] {
	return OptionalOf[T](s, params...)
}
func (s *CheckedSchema[T]) Nullable(params ...any) *NullableSchema[T] {
	return NullableOf[T](s, params...)
}
func (s *CheckedSchema[T]) Nullish(params ...any) *OptionalSchema[T] {
	return NullishOf[T](s, params...)
}
func (s *CheckedSchema[T]) Default(v T) *DefaultSchema[T] { return DefaultOf[T](s, v) }
func (s *CheckedSchema[T]) Catch(v T) *CatchSchema[T]     { return CatchOf[T](s, v) }
func (s *CheckedSchema[T]) Refine(pred func(T) bool, params ...any) *CheckedSchema[T] {
	return RefineOf[T](s, pred, params...)
}
func (s *CheckedSchema[T]) SuperRefine(fn func(T, *RefinementCtx), params ...any) *CheckedSchema[T] {
	return SuperRefineOf[T](s, fn, params...)
}
