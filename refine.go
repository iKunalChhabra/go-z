package zod

import (
	"context"
	"fmt"
)

// RefinementCtx is the context passed to SuperRefine and Transform callbacks.
// Ports core.$RefinementCtx.
type RefinementCtx struct {
	payload *Payload
}

// Value returns the current payload value under refinement.
func (ctx *RefinementCtx) Value() any {
	if ctx == nil || ctx.payload == nil {
		return nil
	}
	return ctx.payload.Value
}

// Context returns the context.Context from ParseCtx, or context.Background()
// when none was provided. Use this for DB/HTTP work inside SuperRefine.
func (ctx *RefinementCtx) Context() context.Context {
	if ctx == nil || ctx.payload == nil || ctx.payload.parseCtx == nil || ctx.payload.parseCtx.Context == nil {
		return context.Background()
	}
	return ctx.payload.parseCtx.Context
}

// ParseContext returns the active ParseCtx, or nil.
func (ctx *RefinementCtx) ParseContext() *ParseCtx {
	if ctx == nil || ctx.payload == nil {
		return nil
	}
	return ctx.payload.parseCtx
}

// AddIssue appends a custom (or caller-specified) issue. By default issues
// continue subsequent checks (Zod superRefine default). Use Issue.WithAbort()
// or set continue explicitly for early termination.
func (ctx *RefinementCtx) AddIssue(iss Issue) {
	if ctx == nil || ctx.payload == nil {
		return
	}
	if iss.Code == "" {
		iss.Code = IssueCustom
	}
	if iss.Input == nil && ctx.payload.Value != nil {
		iss.Input = ctx.payload.Value
	}
	if iss.cont == continueUnset {
		iss.cont = continueYes
	}
	ctx.payload.AddIssue(iss)
}

// AddMessage is a string shorthand for AddIssue (Zod's ctx.addIssue("msg")).
func (ctx *RefinementCtx) AddMessage(msg string) {
	ctx.AddIssue(Issue{Code: IssueCustom, Message: msg, cont: continueYes})
}

// RefineOpts customizes Refine/SuperRefine/Custom beyond Params (path, params map).
type RefineOpts struct {
	Error  ErrorMap
	Abort  bool
	Path   []any
	Params map[string]any
}

// CheckedSchema wraps any schema and runs extra checks after the inner parse.
// T is the inner schema's output type — checks never change it.
type CheckedSchema[T any] struct {
	schemaBase[T]
	def    *Def
	inner  AnySchemaLike
	checks []*Check
}

// CheckSchema runs inner then the provided checks (composition primitive for
// schemas that lack a fluent Check method via AnySchemaLike).
func CheckSchema(inner AnySchemaLike, checks ...*Check) *CheckedSchema[any] {
	return newChecked[any](inner, checks...)
}

// CheckOf is CheckSchema with a typed edge.
func CheckOf[T any](inner Schema[T], checks ...*Check) *CheckedSchema[T] {
	return newChecked[T](inner, checks...)
}

func newChecked[T any](inner AnySchemaLike, checks ...*Check) *CheckedSchema[T] {
	def := &Def{Type: "check", Checks: append([]*Check(nil), checks...)}
	s := &CheckedSchema[T]{def: def, inner: inner, checks: checks}
	innerIn := inner.Internals()
	// buildInternals will compose parse→runChecks when Checks is non-empty.
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
func (s *CheckedSchema[T]) Unwrap() AnySchemaLike { return s.inner }

// Refine attaches a boolean predicate check after inner. Failed predicates
// produce a custom issue. Defaults to continue (non-abort) like Zod's refine.
func Refine(inner AnySchemaLike, pred func(any) bool, params ...any) *CheckedSchema[any] {
	return CheckSchema(inner, refineCheck(pred, params...))
}

// RefineOf is Refine with a typed predicate. It works with every schema type,
// including ones without a fluent Refine method:
//
//	z.RefineOf(z.Set(z.String()), func(v []any) bool { return len(v) > 0 })
func RefineOf[T any](inner Schema[T], pred func(T) bool, params ...any) *CheckedSchema[T] {
	if pred == nil {
		pred = func(T) bool { return true }
	}
	return newChecked[T](inner, refineCheck(func(v any) bool {
		tv, ok := v.(T)
		return ok && pred(tv)
	}, params...))
}

// SuperRefine attaches a contextual refinement after inner.
func SuperRefine(inner AnySchemaLike, fn func(any, *RefinementCtx), params ...any) *CheckedSchema[any] {
	return CheckSchema(inner, superRefineCheck(fn, params...))
}

// SuperRefineOf is SuperRefine with a typed value, for every schema type.
func SuperRefineOf[T any](inner Schema[T], fn func(T, *RefinementCtx), params ...any) *CheckedSchema[T] {
	if fn == nil {
		fn = func(T, *RefinementCtx) {}
	}
	return newChecked[T](inner, superRefineCheck(func(v any, ctx *RefinementCtx) {
		tv, _ := v.(T)
		fn(tv, ctx)
	}, params...))
}

// OverwriteSchema attaches an in-place value transform check after inner.
// Named distinctly from Overwrite (the string Check helper in checks_string.go).
// Ports schema-level .overwrite() / $ZodCheckOverwrite for AnySchemaLike.
func OverwriteSchema(inner AnySchemaLike, fn func(any) any) *CheckedSchema[any] {
	if fn == nil {
		fn = func(v any) any { return v }
	}
	ch := &Check{Name: "overwrite"}
	ch.Fn = func(p *Payload) {
		p.Value = fn(p.Value)
	}
	return CheckSchema(inner, ch)
}

// OverwriteOf is OverwriteSchema with a typed value, for every schema type.
func OverwriteOf[T any](inner Schema[T], fn func(T) T) *CheckedSchema[T] {
	if fn == nil {
		return newChecked[T](inner)
	}
	ch := &Check{Name: "overwrite"}
	ch.Fn = func(p *Payload) {
		if tv, ok := p.Value.(T); ok {
			p.Value = fn(tv)
		}
	}
	return newChecked[T](inner, ch)
}

// CustomSchema is a schema that accepts any input and validates with a
// predicate check. Ports $ZodCustom.
type CustomSchema struct {
	schemaBase[any]
	def *Def
}

// Custom returns a schema that succeeds iff pred(value) is true.
// Defaults to abort:true like Zod's z.custom().
func Custom(pred func(any) bool, params ...any) *CustomSchema {
	if pred == nil {
		pred = func(any) bool { return true }
	}
	opts := normalizeRefineParams(params)
	if !opts.abortSet {
		opts.Abort = true // z.custom defaults abort:true
	}
	ch := refineCheck(pred, RefineOpts{
		Error:  opts.Error,
		Abort:  opts.Abort,
		Path:   opts.Path,
		Params: opts.Params,
	})
	def := &Def{Type: "custom", Checks: []*Check{ch}, Error: opts.Error}
	s := &CustomSchema{def: def}
	parse := func(p *Payload, ctx *ParseCtx) {}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	return s
}

type refineParamResult struct {
	Error    ErrorMap
	Abort    bool
	abortSet bool
	Path     []any
	Params   map[string]any
	When     func(p *Payload) bool
}

func normalizeRefineParams(params []any) refineParamResult {
	var out refineParamResult
	for _, p := range params {
		switch x := p.(type) {
		case nil:
		case string:
			out.Error = MessageFromString(x)
		case ErrorMap:
			out.Error = x
		case func(iss *Issue) string:
			out.Error = ErrorMap(x)
		case Params:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.Abort {
				out.Abort = true
				out.abortSet = true
			}
		case *Params:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.Abort {
					out.Abort = true
					out.abortSet = true
				}
			}
		case RefineOpts:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.Abort {
				out.Abort = true
				out.abortSet = true
			}
			if x.Path != nil {
				out.Path = x.Path
			}
			if x.Params != nil {
				out.Params = x.Params
			}
		case *RefineOpts:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.Abort {
					out.Abort = true
					out.abortSet = true
				}
				if x.Path != nil {
					out.Path = x.Path
				}
				if x.Params != nil {
					out.Params = x.Params
				}
			}
		default:
			panic(fmt.Sprintf("zod: unsupported refine params type %T (want string, ErrorMap, Params, RefineOpts, or nil)", p))
		}
	}
	return out
}

func refineCheck(pred func(any) bool, params ...any) *Check {
	opts := normalizeRefineParams(params)
	ch := &Check{
		Name:  "custom",
		Error: opts.Error,
		Abort: opts.Abort,
		When:  opts.When,
	}
	path := opts.Path
	issueParams := opts.Params
	ch.Fn = func(p *Payload) {
		if pred(p.Value) {
			return
		}
		iss := Issue{
			Code:   IssueCustom,
			Input:  p.Value,
			Path:   append([]any(nil), path...),
			Params: issueParams,
		}
		p.AddIssue(ch.Issue(iss))
	}
	return ch
}

func superRefineCheck(fn func(any, *RefinementCtx), params ...any) *Check {
	opts := normalizeRefineParams(params)
	ch := &Check{
		Name:  "custom",
		Error: opts.Error,
		Abort: opts.Abort,
		When:  opts.When,
	}
	ch.Fn = func(p *Payload) {
		rctx := &RefinementCtx{payload: p}
		fn(p.Value, rctx)
	}
	return ch
}
