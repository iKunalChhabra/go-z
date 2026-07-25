package z

// DefaultSchema substitutes a default when input is Missing (undefined)
// without re-parsing the default through the inner schema. OptIn is set so
// objects may omit the key; OptOut is false (output is always present).
//
// T is the inner schema's output type: a default always produces a value, so
// Parse returns T rather than *T.
type DefaultSchema[T any] struct {
	schemaBase[T]
	def    *Def
	inner  AnySchemaLike
	getDef func() any
}

// Default wraps inner with a static default value for Missing input.
//
// Maps and slices are cloned on each use so handlers cannot corrupt later
// parses by mutating the returned default. Prefer DefaultFunc for custom
// reference types that need a fresh allocation each time.
func Default(inner AnySchemaLike, defVal any) *DefaultSchema[any] {
	return newDefault[any](inner, func() any { return cloneDefaultValue(defVal) })
}

// DefaultOf is Default with a typed edge: the default must be a T and Parse
// returns T.
func DefaultOf[T any](inner Schema[T], defVal T) *DefaultSchema[T] {
	return newDefault[T](inner, func() any { return cloneDefaultValue(defVal) })
}

// DefaultFunc wraps inner with a function-produced default (called each time
// the default is needed).
func DefaultFunc(inner AnySchemaLike, fn func() any) *DefaultSchema[any] {
	if fn == nil {
		fn = func() any { return nil }
	}
	return newDefault[any](inner, fn)
}

// DefaultFuncOf is DefaultFunc with a typed edge.
func DefaultFuncOf[T any](inner Schema[T], fn func() T) *DefaultSchema[T] {
	if fn == nil {
		return newDefault[T](inner, func() any { var zero T; return zero })
	}
	return newDefault[T](inner, func() any { return fn() })
}

func newDefault[T any](inner AnySchemaLike, getDef func() any) *DefaultSchema[T] {
	def := &Def{Type: "default"}
	s := &DefaultSchema[T]{def: def, inner: inner, getDef: getDef}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		// Defaults apply only in the forward (decode) direction.
		if ctx.IsEncode() {
			RunSelf(innerIn, p, ctx)
			return
		}
		if IsMissing(p.Value) {
			p.Value = getDef()
			return
		}
		RunSelf(innerIn, p, ctx)
		if IsMissing(p.Value) {
			p.Value = getDef()
		}
	}
	s.schemaBase = newBase[T](buildInternals(def, parse))
	s.in.OptIn = true
	// OptOut stays false: output is required.
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *DefaultSchema[T]) Unwrap() AnySchemaLike { return s.inner }

// PrefaultSchema substitutes a default for Missing input, then parses the
// (possibly defaulted) value through the inner schema.
type PrefaultSchema[T any] struct {
	schemaBase[T]
	def    *Def
	inner  AnySchemaLike
	getDef func() any
}

// Prefault wraps inner with a static prefault value for Missing input.
// Maps/slices are cloned on each use (same as Default).
func Prefault(inner AnySchemaLike, defVal any) *PrefaultSchema[any] {
	return newPrefault[any](inner, func() any { return cloneDefaultValue(defVal) })
}

// PrefaultOf is Prefault with a typed edge.
func PrefaultOf[T any](inner Schema[T], defVal T) *PrefaultSchema[T] {
	return newPrefault[T](inner, func() any { return cloneDefaultValue(defVal) })
}

// PrefaultFunc wraps inner with a function-produced prefault.
func PrefaultFunc(inner AnySchemaLike, fn func() any) *PrefaultSchema[any] {
	if fn == nil {
		fn = func() any { return nil }
	}
	return newPrefault[any](inner, fn)
}

// PrefaultFuncOf is PrefaultFunc with a typed edge.
func PrefaultFuncOf[T any](inner Schema[T], fn func() T) *PrefaultSchema[T] {
	if fn == nil {
		return newPrefault[T](inner, func() any { var zero T; return zero })
	}
	return newPrefault[T](inner, func() any { return fn() })
}

func newPrefault[T any](inner AnySchemaLike, getDef func() any) *PrefaultSchema[T] {
	def := &Def{Type: "prefault"}
	s := &PrefaultSchema[T]{def: def, inner: inner, getDef: getDef}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		if !ctx.IsEncode() && IsMissing(p.Value) {
			p.Value = getDef()
		}
		RunSelf(innerIn, p, ctx)
	}
	s.schemaBase = newBase[T](buildInternals(def, parse))
	s.in.OptIn = true
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *PrefaultSchema[T]) Unwrap() AnySchemaLike { return s.inner }
