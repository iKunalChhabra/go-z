package z

import (
	"fmt"
	"regexp"
)

// missingType is the internal sentinel for absent values (`undefined`),
// distinct from nil (JSON null). Object parsing passes the sentinel to field
// schemas when a key is absent; Optional/Default/Prefault/Catch react to it.
type missingType struct{}

// missingSentinel is the unique absent-value token used by the library.
// Callers should prefer IsMissing; the exported Missing var aliases this
// value for ergonomics but is not read by internals (so reassigning Missing
// cannot corrupt object/optional parsing).
var missingSentinel any = missingType{}

// Missing is the "absent value" sentinel (undefined) for callers.
// Prefer IsMissing(v) for checks. Do not reassign — internals ignore the var.
var Missing any = missingSentinel

// IsMissing reports whether v is the absent-value sentinel.
func IsMissing(v any) bool {
	_, ok := v.(missingType)
	return ok
}

// ParseFn is a schema's bare type parser: it validates
// and normalizes p.Value, appending issues on failure. It must not run checks.
type ParseFn func(p *Payload, ctx *ParseCtx)

// Def is the common, serializable part of a schema definition.
// Type-specific definition fields live on the concrete schema structs.
type Def struct {
	// Type is the schema kind: "string", "number", "object", ...
	Type string
	// Error is the schema-level error map, attached to issues this schema's
	// own parse produces.
	Error ErrorMap
	// Checks are the attached checks, run in order after parse.
	Checks []*Check
	// Coerce enables input coercion for primitives.
	Coerce bool
}

// Internals is the untyped core of every schema. The typed Schema[T] surface
// wraps it.
//
// Power-user API: Def and Bag are shared mutable pointers. Schemas are
// immutable by convention after construction — do not mutate Internals of a
// shared schema. Prefer fluent builders.
type Internals struct {
	Def *Def
	// Parse is the bare type parser (no checks).
	Parse ParseFn
	// Run executes parse + checks. For zero-check schemas Run == Parse
	// (deferred fast path).
	Run ParseFn

	// Values is the exhaustive set of literal values that pass validation
	// (defined on literal/enum/nil…, propagated through wrappers). Used for
	// discriminated-union dispatch.
	Values map[any]struct{}
	// PropValues maps object property names to their literal value sets;
	// discriminated unions build their dispatch table from it.
	PropValues map[string]map[any]struct{}
	// OptIn/OptOut mark input/output optionality (optin/optout);
	// object parsing skips absent keys whose field schema has OptIn.
	OptIn  bool
	OptOut bool
	// Pattern, when non-nil, is a regexp equivalent of this schema's
	// validation; template literals compose it.
	Pattern *regexp.Regexp
	// Bag is scratch metadata written by check OnAttach hooks (min/max/format
	// hints).
	Bag map[string]any

	// deferred, when set, resolves the internals that actually carry this
	// schema's traits. Only Lazy sets it: its own OptIn/OptOut/Values are
	// unknown until the getter runs, and mutating them later would race with
	// concurrent parses. Set once at construction, never written again.
	deferred func() *Internals
}

// traits returns the internals carrying optionality and value metadata,
// resolving a deferred (lazy) schema on first use. Parse-time readers must go
// through it; construction-time readers must not, because a recursive Lazy is
// still being defined at that point.
func (in *Internals) traits() *Internals {
	if in == nil || in.deferred == nil {
		return in
	}
	return in.deferred()
}

// buildInternals wires Def + ParseFn into Internals, porting the trait
// initialization in core.Type: run OnAttach hooks, then either fast-path
// Run to Parse (no checks) or compose parse → runChecks.
func buildInternals(def *Def, parse ParseFn) *Internals {
	in := &Internals{Def: def, Parse: parse}
	for _, ch := range def.Checks {
		for _, hook := range ch.OnAttach {
			hook(in)
		}
	}
	if len(def.Checks) == 0 {
		in.Run = parse
	} else {
		checks := def.Checks
		in.Run = func(p *Payload, ctx *ParseCtx) {
			parse(p, ctx)
			if ctx != nil && ctx.skipChecks {
				return
			}
			runChecks(p, checks)
		}
	}
	return in
}

// SchemaIssue prepares a raw issue produced by a schema's own parse step:
// wires the schema-level error map; continue stays unset (type errors abort,
// as in).
func (in *Internals) SchemaIssue(iss Issue) Issue {
	iss.errMap = in.Def.Error
	return iss
}

// FailInvalidType appends the canonical invalid_type issue for this schema.
func (in *Internals) FailInvalidType(p *Payload, expected string) {
	p.AddIssue(in.SchemaIssue(Issue{Code: IssueInvalidType, Expected: expected, Input: p.Value}))
}

// withChecks returns a copy of the def with extra checks appended — the
// canonical clone step behind every fluent method (immutable API).
func (d *Def) withChecks(checks ...*Check) *Def {
	nd := *d
	nd.Checks = make([]*Check, 0, len(d.Checks)+len(checks))
	nd.Checks = append(nd.Checks, d.Checks...)
	nd.Checks = append(nd.Checks, checks...)
	return &nd
}

// Schema is the generic typed surface. All schemas implement Schema[T] for
// their output type T; the runtime core stays untyped, exactly like.
type Schema[T any] interface {
	Parse(data any) (T, error)
	ParseCtx(data any, ctx *ParseCtx) (T, error)
	MustParse(data any) T
	SafeParse(data any) SafeParseResult[T]
	Internals() *Internals
}

// AnySchemaLike is the type-erased view used by container schemas (object,
// array…) that hold heterogeneous children.
type AnySchemaLike interface {
	Internals() *Internals
}

// Unwrapper is implemented by every wrapper schema that holds exactly one
// inner schema (optional, nullable, default, prefault, catch, non-optional,
// readonly, refined, transform, preprocess, lazy). Tools that walk a schema
// tree — ToJSONSchema, TemplateLiteral — use it instead of type switching on
// each wrapper type, so new wrappers work without touching the walkers.
type Unwrapper interface {
	AnySchemaLike
	Unwrap() AnySchemaLike
}

// schemaBase provides the typed Parse surface over Internals; concrete schema
// types embed it.
type schemaBase[T any] struct {
	in *Internals
}

func newBase[T any](in *Internals) schemaBase[T] { return schemaBase[T]{in: in} }

// Internals exposes the untyped core.
func (b *schemaBase[T]) Internals() *Internals { return b.in }

// Parse validates data and returns the typed output or a *Error.
func (b *schemaBase[T]) Parse(data any) (T, error) { return parseTyped[T](b.in, data, nil) }

// ParseCtx validates data with per-parse options.
func (b *schemaBase[T]) ParseCtx(data any, ctx *ParseCtx) (T, error) {
	return parseTyped[T](b.in, data, ctx)
}

// MustParse is Parse but panics with *Error on failure.
func (b *schemaBase[T]) MustParse(data any) T {
	v, err := b.Parse(data)
	if err != nil {
		panic(err)
	}
	return v
}

// SafeParse never returns an error value; it mirrors safeParse result.
func (b *schemaBase[T]) SafeParse(data any) SafeParseResult[T] {
	v, err := b.Parse(data)
	if err != nil {
		zerr, _ := err.(*Error)
		return SafeParseResult[T]{Success: false, Error: zerr}
	}
	return SafeParseResult[T]{Success: true, Data: v}
}

// parseTyped is the single typed entry point (core/parse.ts _parse).
func parseTyped[T any](in *Internals, data any, ctx *ParseCtx) (T, error) {
	p := AcquirePayload(data)
	p.parseCtx = ctx
	in.Run(p, ctx)
	if len(p.Issues) > 0 {
		err := newError(p.Issues, ctx)
		ReleasePayload(p)
		var zero T
		return zero, err
	}
	value := p.Value
	out, ok := p.Value.(T)
	ReleasePayload(p)
	if !ok {
		var zero T
		if value == nil || IsMissing(value) {
			// JSON null or an absent value: the zero T is the answer.
			return zero, nil
		}
		// A schema produced a value of a type its own edge does not declare.
		// That is a bug in the schema, not invalid input — surface it instead
		// of silently handing back a zero value.
		return zero, &Error{Issues: []Issue{{
			Code:    IssueCustom,
			Path:    []any{},
			Message: internalTypeMismatch(value, zero),
		}}}
	}
	return out, nil
}

func internalTypeMismatch(got, want any) string {
	return fmt.Sprintf("go-z: internal error: schema produced %T but its typed edge is %T "+
		"(please report at github.com/iKunalChhabra/go-z/issues)", got, want)
}

// ptrBase is the typed surface for wrappers whose output domain is "T or
// nothing" — Optional (T or Missing) and Nullable (T or null). Go's honest
// encoding of `T | undefined` is *T, so Parse returns a nil pointer when
// the value is absent or null. Use ParseAny when you need the raw JSON model
// and must tell Missing apart from null.
type ptrBase[T any] struct {
	in *Internals
}

func newPtrBase[T any](in *Internals) ptrBase[T] { return ptrBase[T]{in: in} }

// Internals exposes the untyped core.
func (b *ptrBase[T]) Internals() *Internals { return b.in }

// Parse validates data and returns nil when the value is absent or null.
func (b *ptrBase[T]) Parse(data any) (*T, error) { return parsePtr[T](b.in, data, nil) }

// ParseCtx validates data with per-parse options.
func (b *ptrBase[T]) ParseCtx(data any, ctx *ParseCtx) (*T, error) {
	return parsePtr[T](b.in, data, ctx)
}

// MustParse is Parse but panics with *Error on failure.
func (b *ptrBase[T]) MustParse(data any) *T {
	v, err := b.Parse(data)
	if err != nil {
		panic(err)
	}
	return v
}

// SafeParse never returns an error value; it mirrors safeParse result.
func (b *ptrBase[T]) SafeParse(data any) SafeParseResult[*T] {
	v, err := b.Parse(data)
	if err != nil {
		zerr, _ := err.(*Error)
		return SafeParseResult[*T]{Success: false, Error: zerr}
	}
	return SafeParseResult[*T]{Success: true, Data: v}
}

// ParseAny returns the raw JSON-model value (Missing stays distinguishable
// from nil) instead of a typed pointer.
func (b *ptrBase[T]) ParseAny(data any) (any, error) { return parseTyped[any](b.in, data, nil) }

// SafeParseAny is ParseAny with a SafeParseResult.
func (b *ptrBase[T]) SafeParseAny(data any) SafeParseResult[any] {
	v, err := b.ParseAny(data)
	if err != nil {
		zerr, _ := err.(*Error)
		return SafeParseResult[any]{Success: false, Error: zerr}
	}
	return SafeParseResult[any]{Success: true, Data: v}
}

// parsePtr runs the untyped core and folds Missing/null into a nil pointer.
func parsePtr[T any](in *Internals, data any, ctx *ParseCtx) (*T, error) {
	v, err := parseTyped[any](in, data, ctx)
	if err != nil {
		return nil, err
	}
	if v == nil || IsMissing(v) {
		return nil, nil
	}
	tv, ok := v.(T)
	if !ok {
		// Inner schema produced a different dynamic type than the declared
		// edge (only reachable through the type-erased constructors).
		return nil, nil
	}
	return new(tv), nil
}

// RunChild parses value with a child schema, scoping any child issues under
// seg on the parent payload (containers' path-prefix merge). It returns
// the child's output value and whether the child parse succeeded.
func RunChild(child *Internals, parent *Payload, value any, ctx *ParseCtx, seg any) (any, bool) {
	cp := AcquirePayload(value)
	if ctx == nil && parent != nil {
		ctx = parent.parseCtx
	}
	cp.parseCtx = ctx
	child.Run(cp, ctx)
	ok := len(cp.Issues) == 0
	if !ok {
		start := len(parent.Issues)
		parent.Issues = append(parent.Issues, cp.Issues...)
		if seg != nil {
			parent.PrependPath(start, seg)
		}
	}
	out := cp.Value
	ReleasePayload(cp)
	return out, ok
}

// RunSelf runs a child schema on the parent payload without path scoping
// (wrappers like Optional/Default/Pipe).
func RunSelf(child *Internals, p *Payload, ctx *ParseCtx) {
	child.Run(p, ctx)
}

//////////////////////////////////////////////////////////////////////////////
// Reference implementations: Any / Unknown (Any / Unknown).
// These double as the canonical pattern for wave-1 schema authors.
//////////////////////////////////////////////////////////////////////////////

// AnySchema accepts any input (z.any()).
type AnySchema struct {
	schemaBase[any]
	def *Def
}

// Any returns a schema that accepts anything.
func Any() *AnySchema { return newAny(&Def{Type: "any"}) }

func newAny(def *Def) *AnySchema {
	s := &AnySchema{def: def}
	s.schemaBase = newBase[any](buildInternals(def, func(p *Payload, ctx *ParseCtx) {}))
	return s
}

// Check attaches raw checks (the low-level composition primitive; fluent
// helpers in later work packages build on this pattern).
func (s *AnySchema) Check(checks ...*Check) *AnySchema {
	return newAny(s.def.withChecks(checks...))
}

// UnknownSchema accepts any input (z.unknown()); identical to Any at runtime.
type UnknownSchema struct {
	schemaBase[any]
	def *Def
}

// Unknown returns a schema that accepts anything.
func Unknown() *UnknownSchema {
	def := &Def{Type: "unknown"}
	s := &UnknownSchema{def: def}
	s.schemaBase = newBase[any](buildInternals(def, func(p *Payload, ctx *ParseCtx) {}))
	return s
}
