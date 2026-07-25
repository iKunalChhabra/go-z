package zod

// PipeSchema runs schema A then schema B on A's output. If A fails, B is
// skipped (payload aborted). OptIn comes from A; OptOut from B.
// Ports $ZodPipe.
type PipeSchema struct {
	schemaBase[any]
	def       *Def
	inSchema  AnySchemaLike
	outSchema AnySchemaLike
}

// Pipe returns a schema that parses through a then b.
func Pipe(a, b AnySchemaLike) *PipeSchema {
	def := &Def{Type: "pipe"}
	return newPipe(def, a, b)
}

func newPipe(def *Def, a, b AnySchemaLike) *PipeSchema {
	s := &PipeSchema{def: def, inSchema: a, outSchema: b}
	aIn := a.Internals()
	bIn := b.Internals()
	parse := func(p *Payload, ctx *ParseCtx) {
		if ctx.IsEncode() {
			// Encode reverses the pipe: B then A.
			RunSelf(bIn, p, ctx)
			if len(p.Issues) > 0 {
				p.Aborted = true
				return
			}
			RunSelf(aIn, p, ctx)
			return
		}
		RunSelf(aIn, p, ctx)
		if len(p.Issues) > 0 {
			p.Aborted = true
			return
		}
		RunSelf(bIn, p, ctx)
	}
	s.schemaBase = newBase[any](buildInternals(def, parse))
	s.Internals().OptIn = aIn.OptIn
	s.Internals().OptOut = bIn.OptOut
	if aIn.Values != nil {
		vals := make(map[any]struct{}, len(aIn.Values))
		for k, v := range aIn.Values {
			vals[k] = v
		}
		s.Internals().Values = vals
	}
	if aIn.PropValues != nil {
		pv := make(map[string]map[any]struct{}, len(aIn.PropValues))
		for k, set := range aIn.PropValues {
			cp := make(map[any]struct{}, len(set))
			for vk, vv := range set {
				cp[vk] = vv
			}
			pv[k] = cp
		}
		s.Internals().PropValues = pv
	}
	return s
}

// In returns the left (input) schema of the pipe.
func (s *PipeSchema) In() AnySchemaLike { return s.inSchema }

// Out returns the right (output) schema of the pipe.
func (s *PipeSchema) Out() AnySchemaLike { return s.outSchema }
