package zod

// CatchCtx is passed to catch fallback functions (Zod's $ZodCatchCtx).
type CatchCtx struct {
	Issues []Issue
	Input  any
}

// CatchSchema replaces any parse failure with a fallback value and clears
// issues (always succeeds). OptIn is set so Missing input can be caught.
// Ports $ZodCatch.
type CatchSchema struct {
	schemaBase[any]
	def        *Def
	inner      AnySchemaLike
	catchValue func(CatchCtx) any
}

// Catch wraps inner with a static fallback on failure.
// Maps/slices are cloned on each use so fallbacks are not shared across parses.
func Catch(inner AnySchemaLike, fallback any) *CatchSchema {
	return newCatch(inner, func(CatchCtx) any { return cloneDefaultValue(fallback) })
}

// CatchFunc wraps inner with a function-produced fallback on failure.
func CatchFunc(inner AnySchemaLike, fn func(CatchCtx) any) *CatchSchema {
	if fn == nil {
		fn = func(CatchCtx) any { return nil }
	}
	return newCatch(inner, fn)
}

func newCatch(inner AnySchemaLike, catchValue func(CatchCtx) any) *CatchSchema {
	def := &Def{Type: "catch"}
	s := &CatchSchema{def: def, inner: inner, catchValue: catchValue}
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
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = true
	s.in.OptOut = innerIn.OptOut
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *CatchSchema) Unwrap() AnySchemaLike { return s.inner }
