package zod

// DefaultSchema substitutes a default when input is Missing (Zod undefined)
// without re-parsing the default through the inner schema. OptIn is set so
// objects may omit the key; OptOut is false (output is always present).
// Ports $ZodDefault.
type DefaultSchema struct {
	schemaBase[any]
	def    *Def
	inner  AnySchemaLike
	getDef func() any
}

// Default wraps inner with a static default value for Missing input.
//
// Maps and slices are cloned on each use so handlers cannot corrupt later
// parses by mutating the returned default. Prefer DefaultFunc for custom
// reference types that need a fresh allocation each time.
func Default(inner AnySchemaLike, defVal any) *DefaultSchema {
	return newDefault(inner, func() any { return cloneDefaultValue(defVal) })
}

// DefaultFunc wraps inner with a function-produced default (called each time
// the default is needed).
func DefaultFunc(inner AnySchemaLike, fn func() any) *DefaultSchema {
	if fn == nil {
		fn = func() any { return nil }
	}
	return newDefault(inner, fn)
}

func newDefault(inner AnySchemaLike, getDef func() any) *DefaultSchema {
	def := &Def{Type: "default"}
	s := &DefaultSchema{def: def, inner: inner, getDef: getDef}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		if IsMissing(p.Value) {
			p.Value = getDef()
			return
		}
		RunSelf(innerIn, p, ctx)
		if IsMissing(p.Value) {
			p.Value = getDef()
		}
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = true
	// OptOut stays false: output is required.
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *DefaultSchema) Unwrap() AnySchemaLike { return s.inner }

// PrefaultSchema substitutes a default for Missing input, then parses the
// (possibly defaulted) value through the inner schema. Ports $ZodPrefault.
type PrefaultSchema struct {
	schemaBase[any]
	def    *Def
	inner  AnySchemaLike
	getDef func() any
}

// Prefault wraps inner with a static prefault value for Missing input.
// Maps/slices are cloned on each use (same as Default).
func Prefault(inner AnySchemaLike, defVal any) *PrefaultSchema {
	return newPrefault(inner, func() any { return cloneDefaultValue(defVal) })
}

// PrefaultFunc wraps inner with a function-produced prefault.
func PrefaultFunc(inner AnySchemaLike, fn func() any) *PrefaultSchema {
	if fn == nil {
		fn = func() any { return nil }
	}
	return newPrefault(inner, fn)
}

func newPrefault(inner AnySchemaLike, getDef func() any) *PrefaultSchema {
	def := &Def{Type: "prefault"}
	s := &PrefaultSchema{def: def, inner: inner, getDef: getDef}
	innerIn := inner.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		if IsMissing(p.Value) {
			p.Value = getDef()
		}
		RunSelf(innerIn, p, ctx)
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.in.OptIn = true
	propagateWrapperMeta(s.in, innerIn)
	return s
}

// Unwrap returns the inner schema.
func (s *PrefaultSchema) Unwrap() AnySchemaLike { return s.inner }
