package zod

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
type CheckedSchema struct {
	schemaBase[any]
	def    *Def
	inner  AnySchemaLike
	checks []*Check
}

// CheckSchema runs inner then the provided checks (composition primitive for
// schemas that lack a fluent Check method via AnySchemaLike).
func CheckSchema(inner AnySchemaLike, checks ...*Check) *CheckedSchema {
	def := &Def{Type: "check", Checks: append([]*Check(nil), checks...)}
	s := &CheckedSchema{def: def, inner: inner, checks: checks}
	innerIn := inner.Internals()
	// buildInternals will compose parse→runChecks when Checks is non-empty.
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
func (s *CheckedSchema) Unwrap() AnySchemaLike { return s.inner }

// Refine attaches a boolean predicate check after inner. Failed predicates
// produce a custom issue. Defaults to continue (non-abort) like Zod's refine.
func Refine(inner AnySchemaLike, pred func(any) bool, params ...any) *CheckedSchema {
	return CheckSchema(inner, refineCheck(pred, params...))
}

// SuperRefine attaches a contextual refinement after inner.
func SuperRefine(inner AnySchemaLike, fn func(any, *RefinementCtx), params ...any) *CheckedSchema {
	return CheckSchema(inner, superRefineCheck(fn, params...))
}

// OverwriteSchema attaches an in-place value transform check after inner.
// Named distinctly from Overwrite (the string Check helper in checks_string.go).
// Ports schema-level .overwrite() / $ZodCheckOverwrite for AnySchemaLike.
func OverwriteSchema(inner AnySchemaLike, fn func(any) any) *CheckedSchema {
	if fn == nil {
		fn = func(v any) any { return v }
	}
	ch := &Check{Name: "overwrite"}
	ch.Fn = func(p *Payload) {
		p.Value = fn(p.Value)
	}
	return CheckSchema(inner, ch)
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
