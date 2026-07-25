package z

// XorSchema is an exclusive union: exactly one option must succeed.
// Ports Xor / z.xor([...]).
type XorSchema struct {
	schemaBase[any]
	def     *Def
	Options []AnySchemaLike
}

// Xor returns an exclusive union over options. Optional params may be a string
// message, ErrorMap, or Params. Zero matches and multiple matches both yield
// invalid_union (multiple matches use Errors=[] and Inclusive=false).
func Xor(options []AnySchemaLike, params ...any) *XorSchema {
	p := normalizeParams(params)
	opts := make([]AnySchemaLike, len(options))
	copy(opts, options)
	def := &Def{Type: "xor", Error: p.Error}
	return newXor(def, opts)
}

// XorOf is a variadic convenience wrapper around Xor.
func XorOf(options ...AnySchemaLike) *XorSchema {
	return Xor(options)
}

func newXor(def *Def, options []AnySchemaLike) *XorSchema {
	s := &XorSchema{def: def, Options: options}
	parse := makeXorParse(def, options)
	in := buildInternals(def, parse)
	applyUnionTraits(in, options)
	s.schemaBase = newBase[any](in)
	return s
}

func makeXorParse(def *Def, options []AnySchemaLike) ParseFn {
	// Single-option fast path: delegate directly (matches first shortcut).
	if len(options) == 1 {
		child := options[0].Internals()
		return func(p *Payload, ctx *ParseCtx) {
			child.Run(p, ctx)
		}
	}
	return func(p *Payload, ctx *ParseCtx) {
		handleXorParse(p, ctx, def, options)
	}
}

func handleXorParse(p *Payload, ctx *ParseCtx, def *Def, options []AnySchemaLike) {
	if len(options) == 0 {
		p.AddIssue(Issue{
			Code:   IssueInvalidUnion,
			Errors: [][]Issue{},
			Input:  p.Value,
			errMap: def.Error,
		})
		return
	}

	results := make([]*Payload, len(options))
	successes := make([]*Payload, 0, 1)
	for i, opt := range options {
		cp := AcquirePayload(p.Value)
		opt.Internals().Run(cp, ctx)
		results[i] = cp
		if len(cp.Issues) == 0 {
			successes = append(successes, cp)
		}
	}

	switch len(successes) {
	case 1:
		p.Value = successes[0].Value
		for _, r := range results {
			ReleasePayload(r)
		}
		return
	case 0:
		// No matches — same shape as regular union failure.
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
	default:
		// Multiple matches — exclusive union failure.
		p.AddIssue(Issue{
			Code:      IssueInvalidUnion,
			Errors:    [][]Issue{},
			Inclusive: false,
			Input:     p.Value,
			errMap:    def.Error,
		})
	}
	for _, r := range results {
		ReleasePayload(r)
	}
}

// Check attaches raw checks (immutable clone).
func (s *XorSchema) Check(checks ...*Check) *XorSchema {
	return newXor(s.def.withChecks(checks...), s.Options)
}
