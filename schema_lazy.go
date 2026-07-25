package zod

import "sync"

// LazySchema is the Go port of $ZodLazy: a deferred schema getter used for
// recursive and mutually-recursive types. The getter is memoized on first use.
type LazySchema struct {
	schemaBase[any]
	def   *Def
	state *lazyState
	// syncMeta guards the one-time copy of the inner schema's traits onto
	// this schema's Internals. Without it every Inner/Unwrap call wrote to
	// shared state, which races with a concurrent Parse of an object that has
	// this Lazy as a field (object parsing reads child.OptIn/OptOut).
	syncMeta sync.Once
}

type lazyState struct {
	fn    func() AnySchemaLike
	once  sync.Once
	inner AnySchemaLike
}

func (st *lazyState) get() AnySchemaLike {
	st.once.Do(func() {
		st.inner = st.fn()
		if st.inner == nil {
			panic("Lazy getter returned nil schema")
		}
	})
	return st.inner
}

// Lazy returns a schema whose definition is produced by fn on first use
// (z.lazy(() => ...)). fn is memoized; subsequent uses reuse the same inner
// schema instance (important for recursive identity).
func Lazy(fn func() AnySchemaLike) *LazySchema {
	if fn == nil {
		panic("Lazy: getter must not be nil")
	}
	def := &Def{Type: "lazy"}
	return newLazy(def, &lazyState{fn: fn})
}

func newLazy(def *Def, state *lazyState) *LazySchema {
	s := &LazySchema{def: def, state: state}
	s.schemaBase = newBase[any](buildInternals(def, func(p *Payload, ctx *ParseCtx) {
		RunSelf(state.get().Internals(), p, ctx)
	}))
	// Let parse-time trait lookups resolve through the memoized getter instead
	// of reading fields that Inner() would have to write after construction.
	s.in.deferred = func() *Internals { return state.get().Internals() }
	return s
}

// Inner returns the memoized inner schema, resolving the getter if needed.
// The first call also copies OptIn/OptOut/Values/PropValues onto this Lazy's
// Internals so container schemas that inspect Internals() see the inner traits
// (Zod's defineLazy getters on _zod).
//
// The copy happens exactly once. Resolve a Lazy before sharing it across
// goroutines — call Inner (or build the enclosing schema, which does) during
// setup — because that first copy writes to Internals that a concurrent Parse
// may be reading.
func (s *LazySchema) Inner() AnySchemaLike {
	inner := s.state.get()
	s.syncMeta.Do(func() {
		iin := inner.Internals()
		s.in.OptIn = iin.OptIn
		s.in.OptOut = iin.OptOut
		s.in.Values = iin.Values
		s.in.PropValues = iin.PropValues
		s.in.Pattern = iin.Pattern
	})
	return inner
}

func (s *LazySchema) innerType() AnySchemaLike { return s.Inner() }

// Unwrap resolves the inner schema, satisfying Unwrapper for schema walkers.
func (s *LazySchema) Unwrap() AnySchemaLike { return s.Inner() }

// Check attaches raw checks (immutable clone). Clones share the memoized
// inner schema via the same lazyState (Zod caches on the shared def).
func (s *LazySchema) Check(checks ...*Check) *LazySchema {
	return newLazy(s.def.withChecks(checks...), s.state)
}
