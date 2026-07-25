package zod

// TransformSchema runs an inner schema, then applies a transform function.
// Issues added via RefinementCtx (or a returned error) fail the parse.
// Ports the classic pattern pipe(schema, $ZodTransform).
type TransformSchema struct {
	schemaBase[any]
	def   *Def
	inner AnySchemaLike
	fn    func(any, *RefinementCtx) (any, error)
}

// Transform runs inner then fn. Prefer TransformTo when the output type is known.
func Transform(inner AnySchemaLike, fn func(any, *RefinementCtx) (any, error)) *TransformSchema {
	if fn == nil {
		fn = func(v any, _ *RefinementCtx) (any, error) { return v, nil }
	}
	def := &Def{Type: "transform"}
	return newTransform(def, inner, fn)
}

func newTransform(def *Def, inner AnySchemaLike, fn func(any, *RefinementCtx) (any, error)) *TransformSchema {
	s := &TransformSchema{def: def, inner: inner, fn: fn}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		RunSelf(innerIn, p, ctx)
		if len(p.Issues) > 0 {
			p.Aborted = true
			return
		}
		rctx := &RefinementCtx{payload: p}
		out, err := fn(p.Value, rctx)
		if err != nil {
			p.AddIssue(Issue{
				Code:    IssueCustom,
				Message: err.Error(),
				Input:   p.Value,
			})
			return
		}
		if len(p.Issues) > 0 {
			return
		}
		p.Value = out
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = innerIn.OptIn
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *TransformSchema) Unwrap() AnySchemaLike { return s.inner }

// transformToSchema is a thin typed wrapper around Transform.
type transformToSchema[Out any] struct {
	schemaBase[Out]
	inner AnySchemaLike
}

// TransformTo is like Transform but returns Schema[Out].
func TransformTo[Out any](inner AnySchemaLike, fn func(any) (Out, error)) Schema[Out] {
	if fn == nil {
		fn = func(v any) (Out, error) {
			out, _ := v.(Out)
			return out, nil
		}
	}
	wrapped := Transform(inner, func(v any, _ *RefinementCtx) (any, error) {
		out, err := fn(v)
		if err != nil {
			return nil, err
		}
		return out, nil
	})
	s := &transformToSchema[Out]{inner: inner}
	s.schemaBase = newBase[Out](wrapped.Internals())
	return s
}

// PreprocessSchema applies fn to the input, then parses through schema.
// Ports $ZodPreprocess (pipe(transform, schema)).
type PreprocessSchema struct {
	schemaBase[any]
	def    *Def
	fn     func(any) any
	schema AnySchemaLike
}

// Preprocess returns a schema that transforms input then validates with schema.
func Preprocess(fn func(any) any, schema AnySchemaLike) *PreprocessSchema {
	if fn == nil {
		fn = func(v any) any { return v }
	}
	def := &Def{Type: "pipe"} // Zod preprocess is a pipe subtype
	return newPreprocess(def, fn, schema)
}

func newPreprocess(def *Def, fn func(any) any, schema AnySchemaLike) *PreprocessSchema {
	s := &PreprocessSchema{def: def, fn: fn, schema: schema}
	schemaIn := schema.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		p.Value = fn(p.Value)
		RunSelf(schemaIn, p, ctx)
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = schemaIn.OptIn
	s.in.OptOut = schemaIn.OptOut
	propagateWrapperMeta(s.in, schemaIn)
	return s
}
