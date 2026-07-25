package z

// OptionalSchema wraps an inner schema so Missing (undefined) is accepted
// and OptIn/OptOut are set — objects may omit the key. Null (nil) is NOT
// accepted; use Nullable or Nullish for that.
//
// T is the inner schema's output type, so Parse returns *T: nil means the
// value was absent. The type-erased constructor Optional yields
// *OptionalSchema[any]; OptionalOf and the fluent .Optional() keep the inner
// type, so String().Optional().Parse(v) returns (*string, error).
type OptionalSchema[T any] struct {
	ptrBase[T]
	def   *Def
	inner AnySchemaLike
}

// Optional returns a schema that accepts the inner type or Missing. The output
// edge is type-erased; use OptionalOf when the inner type is known statically.
func Optional(inner AnySchemaLike, params ...any) *OptionalSchema[any] {
	return newOptional[any](optionalDef(params), inner)
}

// OptionalOf is Optional with a typed edge: Parse returns *T (nil when absent).
func OptionalOf[T any](inner Schema[T], params ...any) *OptionalSchema[T] {
	return newOptional[T](optionalDef(params), inner)
}

func optionalDef(params []any) *Def {
	p := normalizeParams(params)
	return &Def{Type: "optional", Error: p.Error}
}

func newOptional[T any](def *Def, inner AnySchemaLike) *OptionalSchema[T] {
	s := &OptionalSchema[T]{def: def, inner: inner}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		// Nested optional inners: always run them so their own logic applies,
		// then restore Missing when the original input was absent.
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
	s.ptrBase = newPtrBase[T](buildInternals(def, parse))
	s.in.OptIn = true
	s.in.OptOut = true
	propagateWrapperMeta(s.in, innerIn)
	if s.in.Values != nil {
		s.in.Values[missingSentinel] = struct{}{}
	}
	return s
}

// Unwrap returns the inner schema.
func (s *OptionalSchema[T]) Unwrap() AnySchemaLike { return s.inner }

// NullableSchema accepts nil (JSON null) in addition to the inner type.
// OptIn/OptOut are inherited from the inner schema.
//
// Parse returns *T: nil means the value was null.
type NullableSchema[T any] struct {
	ptrBase[T]
	def   *Def
	inner AnySchemaLike
}

// Nullable returns a schema that accepts the inner type or nil.
func Nullable(inner AnySchemaLike, params ...any) *NullableSchema[any] {
	return newNullable[any](nullableDef(params), inner)
}

// NullableOf is Nullable with a typed edge: Parse returns *T (nil for null).
func NullableOf[T any](inner Schema[T], params ...any) *NullableSchema[T] {
	return newNullable[T](nullableDef(params), inner)
}

func nullableDef(params []any) *Def {
	p := normalizeParams(params)
	return &Def{Type: "nullable", Error: p.Error}
}

func newNullable[T any](def *Def, inner AnySchemaLike) *NullableSchema[T] {
	s := &NullableSchema[T]{def: def, inner: inner}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		if p.Value == nil {
			return
		}
		RunSelf(innerIn, p, ctx)
	}
	s.ptrBase = newPtrBase[T](buildInternals(def, parse))
	s.in.OptIn = innerIn.OptIn
	s.in.OptOut = innerIn.OptOut
	propagateWrapperMeta(s.in, innerIn)
	if s.in.Values != nil {
		s.in.Values[nil] = struct{}{}
	}
	return s
}

// Unwrap returns the inner schema.
func (s *NullableSchema[T]) Unwrap() AnySchemaLike { return s.inner }

// Nullish is Optional(Nullable(inner)) — accepts Missing, nil, or the inner type.
func Nullish(inner AnySchemaLike, params ...any) *OptionalSchema[any] {
	return Optional(Nullable(inner), params...)
}

// NullishOf is Nullish with a typed edge. Both null and absent parse to a nil
// *T, so the type parameter stays the inner type rather than becoming **T.
func NullishOf[T any](inner Schema[T], params ...any) *OptionalSchema[T] {
	return newOptional[T](optionalDef(params), NullableOf(inner))
}

// NonOptionalSchema rejects Missing after running the inner schema, emitting
// invalid_type with expected "nonoptional".
type NonOptionalSchema[T any] struct {
	schemaBase[T]
	def   *Def
	inner AnySchemaLike
}

// NonOptional returns a schema that fails when the value is Missing.
func NonOptional(inner AnySchemaLike, params ...any) *NonOptionalSchema[any] {
	return newNonOptional[any](nonOptionalDef(params), inner)
}

// NonOptionalOf is NonOptional with a typed edge.
func NonOptionalOf[T any](inner Schema[T], params ...any) *NonOptionalSchema[T] {
	return newNonOptional[T](nonOptionalDef(params), inner)
}

func nonOptionalDef(params []any) *Def {
	p := normalizeParams(params)
	return &Def{Type: "nonoptional", Error: p.Error}
}

func newNonOptional[T any](def *Def, inner AnySchemaLike) *NonOptionalSchema[T] {
	s := &NonOptionalSchema[T]{def: def, inner: inner}
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
	s.schemaBase = newBase[T](buildInternals(def, parse))
	propagateWrapperMeta(s.in, innerIn)
	if s.in.Values != nil {
		delete(s.in.Values, missingSentinel)
	}
	return s
}

// Unwrap returns the inner schema.
func (s *NonOptionalSchema[T]) Unwrap() AnySchemaLike { return s.inner }

// ReadonlySchema is a documented no-op wrapper for API parity with's
// Readonly (Object.freeze has no useful equivalent for map[string]any).
type ReadonlySchema[T any] struct {
	schemaBase[T]
	def   *Def
	inner AnySchemaLike
}

// Readonly wraps inner without changing parse behavior.
func Readonly(inner AnySchemaLike, params ...any) *ReadonlySchema[any] {
	return newReadonly[any](readonlyDef(params), inner)
}

// ReadonlyOf is Readonly with a typed edge.
func ReadonlyOf[T any](inner Schema[T], params ...any) *ReadonlySchema[T] {
	return newReadonly[T](readonlyDef(params), inner)
}

func readonlyDef(params []any) *Def {
	p := normalizeParams(params)
	return &Def{Type: "readonly", Error: p.Error}
}

func newReadonly[T any](def *Def, inner AnySchemaLike) *ReadonlySchema[T] {
	s := &ReadonlySchema[T]{def: def, inner: inner}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		RunSelf(innerIn, p, ctx)
	}
	s.schemaBase = newBase[T](buildInternals(def, parse))
	s.in.OptIn = innerIn.OptIn
	s.in.OptOut = innerIn.OptOut
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *ReadonlySchema[T]) Unwrap() AnySchemaLike { return s.inner }

// propagateWrapperMeta copies Values/PropValues/Pattern from inner onto the
// wrapper internals (lazy-equivalent of defineLazy copies).
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
