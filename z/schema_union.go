package z

// UnionSchema is the schema. Output type is any
// because Go cannot express a heterogeneous union as a type parameter.
type UnionSchema struct {
	schemaBase[any]
	def     *Def
	Options []AnySchemaLike
}

// Union returns a union schema over options (z.union([...])). Optional params
// may be a string message, ErrorMap, or Params.
func Union(options []AnySchemaLike, params ...any) *UnionSchema {
	p := normalizeParams(params)
	// Copy so callers cannot mutate the schema's option list later.
	opts := make([]AnySchemaLike, len(options))
	copy(opts, options)
	def := &Def{Type: "union", Error: p.Error}
	return newUnion(def, opts)
}

// UnionOf is a variadic convenience wrapper around Union.
func UnionOf(options ...AnySchemaLike) *UnionSchema {
	return Union(options)
}

func newUnion(def *Def, options []AnySchemaLike) *UnionSchema {
	s := &UnionSchema{def: def, Options: options}
	parse := makeUnionParse(def, options)
	in := buildInternals(def, parse)
	applyUnionTraits(in, options)
	s.schemaBase = newBase[any](in)
	return s
}

func applyUnionTraits(in *Internals, options []AnySchemaLike) {
	if len(options) == 0 {
		return
	}
	allValues := true
	merged := map[any]struct{}{}
	optIn, optOut := false, false
	for _, o := range options {
		oin := o.Internals()
		if oin.OptIn {
			optIn = true
		}
		if oin.OptOut {
			optOut = true
		}
		if oin.Values == nil {
			allValues = false
			continue
		}
		for v := range oin.Values {
			merged[v] = struct{}{}
		}
	}
	in.OptIn = optIn
	in.OptOut = optOut
	if allValues {
		in.Values = merged
	}
}

func makeUnionParse(def *Def, options []AnySchemaLike) ParseFn {
	// Single-option fast path: delegate directly (`first` shortcut).
	if len(options) == 1 {
		child := options[0].Internals()
		return func(p *Payload, ctx *ParseCtx) {
			child.Run(p, ctx)
		}
	}
	return func(p *Payload, ctx *ParseCtx) {
		handleUnionParse(p, ctx, def, options)
	}
}

func handleUnionParse(p *Payload, ctx *ParseCtx, def *Def, options []AnySchemaLike) {
	if len(options) == 0 {
		p.AddIssue(Issue{
			Code:   IssueInvalidUnion,
			Errors: [][]Issue{},
			Input:  p.Value,
			errMap: def.Error,
		})
		return
	}

	results := make([]*Payload, 0, len(options))
	for _, opt := range options {
		cp := AcquirePayload(p.Value)
		opt.Internals().Run(cp, ctx)
		if len(cp.Issues) == 0 {
			p.Value = cp.Value
			// Release any prior failed payloads, then the success payload.
			for _, r := range results {
				ReleasePayload(r)
			}
			ReleasePayload(cp)
			return
		}
		results = append(results, cp)
	}

	//: if exactly one non-aborted result remains, surface its issues
	// directly (continuable / refine failures) instead of wrapping.
	nonAborted := make([]*Payload, 0, len(results))
	for _, r := range results {
		if !r.aborted(0) {
			nonAborted = append(nonAborted, r)
		}
	}
	if len(nonAborted) == 1 {
		only := nonAborted[0]
		p.Value = only.Value
		p.Issues = append(p.Issues, only.Issues...)
		for _, r := range results {
			ReleasePayload(r)
		}
		return
	}

	cfg := GetConfig()
	errors := make([][]Issue, len(results))
	for i, r := range results {
		inner := make([]Issue, len(r.Issues))
		for j, iss := range r.Issues {
			inner[j] = FinalizeIssue(iss, ctx, cfg)
		}
		errors[i] = inner
	}
	p.AddIssue(Issue{
		Code:   IssueInvalidUnion,
		Errors: errors,
		Input:  p.Value,
		errMap: def.Error,
	})
	for _, r := range results {
		ReleasePayload(r)
	}
}

// Check attaches raw checks (immutable clone).
func (s *UnionSchema) Check(checks ...*Check) *UnionSchema {
	return newUnion(s.def.withChecks(checks...), s.Options)
}
