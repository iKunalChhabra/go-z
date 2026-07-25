package zod

// CatchCtx is passed to catch fallback functions (Zod's $ZodCatchCtx).
type CatchCtx struct {
	Issues []Issue
	Input  any
}

// CatchSchema replaces any parse failure with a fallback value and clears
// issues (always succeeds). OptIn is set so Missing input can be caught.
// Ports $ZodCatch.
// T is the inner schema's output type: catch always yields a value, so Parse
// returns T.
type CatchSchema[T any] struct {
	schemaBase[T]
	def        *Def
	inner      AnySchemaLike
	catchValue func(CatchCtx) any
}

// Catch wraps inner with a static fallback on failure.
// Maps/slices are cloned on each use so fallbacks are not shared across parses.
func Catch(inner AnySchemaLike, fallback any) *CatchSchema[any] {
	return newCatch[any](inner, func(CatchCtx) any { return cloneDefaultValue(fallback) })
}

// CatchOf is Catch with a typed edge: the fallback must be a T.
func CatchOf[T any](inner Schema[T], fallback T) *CatchSchema[T] {
	return newCatch[T](inner, func(CatchCtx) any { return cloneDefaultValue(fallback) })
}

// CatchFunc wraps inner with a function-produced fallback on failure.
func CatchFunc(inner AnySchemaLike, fn func(CatchCtx) any) *CatchSchema[any] {
	if fn == nil {
		fn = func(CatchCtx) any { return nil }
	}
	return newCatch[any](inner, fn)
}

// CatchFuncOf is CatchFunc with a typed edge.
func CatchFuncOf[T any](inner Schema[T], fn func(CatchCtx) T) *CatchSchema[T] {
	if fn == nil {
		return newCatch[T](inner, func(CatchCtx) any { var zero T; return zero })
	}
	return newCatch[T](inner, func(c CatchCtx) any { return fn(c) })
}

func newCatch[T any](inner AnySchemaLike, catchValue func(CatchCtx) any) *CatchSchema[T] {
	def := &Def{Type: "catch"}
	s := &CatchSchema[T]{def: def, inner: inner, catchValue: catchValue}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		RunSelf(innerIn, p, ctx)
		if len(p.Issues) == 0 {
			return
		}
		// Catch applies only in the forward (decode) direction.
		if ctx.IsEncode() {
			return
		}
		// Finalize issues for the catch callback (Zod passes finalized issues).
		finalized := make([]Issue, len(p.Issues))
		cfg := GetConfig()
		for i, iss := range p.Issues {
			finalized[i] = FinalizeIssue(iss, ctx, cfg)
		}
		input := p.Value
		p.Value = catchValue(CatchCtx{Issues: finalized, Input: input})
		p.Issues = p.Issues[:0]
		p.Aborted = false
	}
	s.schemaBase = newBase[T](buildInternals(def, parse))
	s.in.OptIn = true
	s.in.OptOut = innerIn.OptOut
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *CatchSchema[T]) Unwrap() AnySchemaLike { return s.inner }
