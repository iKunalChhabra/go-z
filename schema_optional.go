package zod

// OptionalSchema wraps an inner schema so Missing (Zod undefined) is accepted
// and OptIn/OptOut are set — objects may omit the key. Null (nil) is NOT
// accepted; use Nullable or Nullish for that. Ports $ZodOptional.
type OptionalSchema struct {
	schemaBase[any]
	def   *Def
	inner AnySchemaLike
}

// Optional returns a schema that accepts the inner type or Missing.
func Optional(inner AnySchemaLike, params ...any) *OptionalSchema {
	p := normalizeParams(params)
	def := &Def{Type: "optional", Error: p.Error}
	return newOptional(def, inner)
}

func newOptional(def *Def, inner AnySchemaLike) *OptionalSchema {
	s := &OptionalSchema{def: def, inner: inner}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		// Nested optional inners: always run them so their own logic applies,
		// then restore Missing when the original input was absent (Zod's
		// handleOptionalResult).
		if innerIn.traits().OptIn {
			input := p.Value
			RunSelf(innerIn, p, ctx)
			if IsMissing(input) && len(p.Issues) > 0 {
				p.Issues = p.Issues[:0]
				p.Value = missingSentinel
			}
			return
		}
		if IsMissing(p.Value) {
			return
		}
		RunSelf(innerIn, p, ctx)
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = true
	s.in.OptOut = true
	propagateWrapperMeta(s.in, innerIn)
	if s.in.Values != nil {
		s.in.Values[missingSentinel] = struct{}{}
	}
	return s
}

// Unwrap returns the inner schema.
func (s *OptionalSchema) Unwrap() AnySchemaLike { return s.inner }

// NullableSchema accepts nil (JSON null) in addition to the inner type.
// OptIn/OptOut are inherited from the inner schema. Ports $ZodNullable.
type NullableSchema struct {
	schemaBase[any]
	def   *Def
	inner AnySchemaLike
}

// Nullable returns a schema that accepts the inner type or nil.
func Nullable(inner AnySchemaLike, params ...any) *NullableSchema {
	p := normalizeParams(params)
	def := &Def{Type: "nullable", Error: p.Error}
	return newNullable(def, inner)
}

func newNullable(def *Def, inner AnySchemaLike) *NullableSchema {
	s := &NullableSchema{def: def, inner: inner}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		if p.Value == nil {
			return
		}
		RunSelf(innerIn, p, ctx)
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = innerIn.OptIn
	s.in.OptOut = innerIn.OptOut
	propagateWrapperMeta(s.in, innerIn)
	if s.in.Values != nil {
		s.in.Values[nil] = struct{}{}
	}
	return s
}

// Unwrap returns the inner schema.
func (s *NullableSchema) Unwrap() AnySchemaLike { return s.inner }

// Nullish is Optional(Nullable(inner)) — accepts Missing, nil, or the inner type.
func Nullish(inner AnySchemaLike, params ...any) *OptionalSchema {
	return Optional(Nullable(inner), params...)
}

// NonOptionalSchema rejects Missing after running the inner schema, emitting
// invalid_type with expected "nonoptional". Ports $ZodNonOptional.
type NonOptionalSchema struct {
	schemaBase[any]
	def   *Def
	inner AnySchemaLike
}

// NonOptional returns a schema that fails when the value is Missing.
func NonOptional(inner AnySchemaLike, params ...any) *NonOptionalSchema {
	p := normalizeParams(params)
	def := &Def{Type: "nonoptional", Error: p.Error}
	return newNonOptional(def, inner)
}

func newNonOptional(def *Def, inner AnySchemaLike) *NonOptionalSchema {
	s := &NonOptionalSchema{def: def, inner: inner}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		RunSelf(innerIn, p, ctx)
		if len(p.Issues) == 0 && IsMissing(p.Value) {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "nonoptional",
				Input:    p.Value,
				errMap:   def.Error,
			})
		}
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	propagateWrapperMeta(s.in, innerIn)
	if s.in.Values != nil {
		delete(s.in.Values, missingSentinel)
	}
	return s
}

// Unwrap returns the inner schema.
func (s *NonOptionalSchema) Unwrap() AnySchemaLike { return s.inner }

// ReadonlySchema is a documented no-op wrapper for API parity with Zod's
// $ZodReadonly (Object.freeze has no useful equivalent for map[string]any).
type ReadonlySchema struct {
	schemaBase[any]
	def   *Def
	inner AnySchemaLike
}

// Readonly wraps inner without changing parse behavior.
func Readonly(inner AnySchemaLike, params ...any) *ReadonlySchema {
	p := normalizeParams(params)
	def := &Def{Type: "readonly", Error: p.Error}
	return newReadonly(def, inner)
}

func newReadonly(def *Def, inner AnySchemaLike) *ReadonlySchema {
	s := &ReadonlySchema{def: def, inner: inner}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		RunSelf(innerIn, p, ctx)
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = innerIn.OptIn
	s.in.OptOut = innerIn.OptOut
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *ReadonlySchema) Unwrap() AnySchemaLike { return s.inner }

// propagateWrapperMeta copies Values/PropValues/Pattern from inner onto the
// wrapper internals (lazy-equivalent of Zod's defineLazy copies).
func propagateWrapperMeta(dst, src *Internals) {
	if src.Values != nil {
		dst.Values = make(map[any]struct{}, len(src.Values))
		for k, v := range src.Values {
			dst.Values[k] = v
		}
	}
	if src.PropValues != nil {
		dst.PropValues = make(map[string]map[any]struct{}, len(src.PropValues))
		for k, set := range src.PropValues {
			cp := make(map[any]struct{}, len(set))
			for vk, vv := range set {
				cp[vk] = vv
			}
			dst.PropValues[k] = cp
		}
	}
	dst.Pattern = src.Pattern
}
