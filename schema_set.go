package zod

// SetSchema is the Go port of ZodSet / $ZodSet.
//
// Go has no built-in Set type. This schema accepts:
//   - []any (and other slices): uniqueness is enforced — duplicates are dropped
//     from the output (first occurrence wins), matching Set insertion semantics;
//   - map[any]struct{} / map[string]struct{}: keys are the set elements.
//
// Output is always []any (ordered by first-seen insertion for slices, or
// arbitrary map iteration order for map inputs). Size checks use Origin="set".
type SetSchema struct {
	schemaBase[[]any]
	def         *Def
	valueSchema AnySchemaLike
}

// Set returns a set schema (z.set(valueType)).
func Set(valueSchema AnySchemaLike, params ...any) *SetSchema {
	p := normalizeParams(params)
	def := &Def{Type: "set", Error: p.Error}
	return newSet(def, valueSchema)
}

func newSet(def *Def, valueSchema AnySchemaLike) *SetSchema {
	s := &SetSchema{def: def, valueSchema: valueSchema}
	parse := makeSetParse(def, valueSchema)
	s.schemaBase = newBase[[]any](buildInternals(def, parse))
	return s
}

func makeSetParse(def *Def, valueSchema AnySchemaLike) ParseFn {
	var child *Internals
	if valueSchema != nil {
		child = valueSchema.Internals()
	}
	return func(p *Payload, ctx *ParseCtx) {
		items, ok := asSetItems(p.Value)
		if !ok {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "set",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}

		out := make([]any, 0, len(items))
		seen := make(map[any]struct{}, len(items))
		for _, item := range items {
			var val any = item
			if child != nil {
				// Set element issues are not path-prefixed (Zod handleSetResult).
				cp := AcquirePayload(item)
				child.Run(cp, ctx)
				if len(cp.Issues) > 0 {
					p.Issues = append(p.Issues, cp.Issues...)
				}
				val = cp.Value
				ReleasePayload(cp)
			}
			// Non-comparable values cannot be map keys; keep every occurrence.
			if !isComparable(val) {
				out = append(out, val)
				continue
			}
			if _, dup := seen[val]; dup {
				continue
			}
			seen[val] = struct{}{}
			out = append(out, val)
		}
		p.Value = out
	}
}

func asSetItems(v any) ([]any, bool) {
	if items, ok := asAnySlice(v); ok {
		return items, true
	}
	switch m := v.(type) {
	case map[any]struct{}:
		out := make([]any, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out, true
	case map[string]struct{}:
		out := make([]any, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out, true
	}
	return nil, false
}

func isComparable(v any) bool {
	switch v.(type) {
	case nil, bool, string,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		complex64, complex128:
		return true
	}
	return false
}

// Min requires at least n unique elements (Origin="set").
func (s *SetSchema) Min(n int, params ...any) *SetSchema {
	return newSet(s.def.withChecks(setMinSize(n, params...)), s.valueSchema)
}

// Max requires at most n unique elements (Origin="set").
func (s *SetSchema) Max(n int, params ...any) *SetSchema {
	return newSet(s.def.withChecks(setMaxSize(n, params...)), s.valueSchema)
}

// Size requires exactly n unique elements.
func (s *SetSchema) Size(n int, params ...any) *SetSchema {
	return newSet(s.def.withChecks(setSizeEquals(n, params...)), s.valueSchema)
}

// NonEmpty is Min(1).
func (s *SetSchema) NonEmpty(params ...any) *SetSchema {
	return s.Min(1, params...)
}

// Check attaches raw checks.
func (s *SetSchema) Check(checks ...*Check) *SetSchema {
	return newSet(s.def.withChecks(checks...), s.valueSchema)
}

func setLen(v any) (int, bool) {
	switch s := v.(type) {
	case []any:
		return len(s), true
	}
	return 0, false
}

func hasSetSize(p *Payload) bool {
	_, ok := setLen(p.Value)
	return ok
}

func setMinSize(minimum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{Name: "min_size", Error: p.Error, Abort: p.Abort, When: hasSetSize}
	ch.Fn = func(payload *Payload) {
		n, ok := setLen(payload.Value)
		if !ok || n >= minimum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code: IssueTooSmall, Origin: "set", Minimum: minimum, Inclusive: true, Input: payload.Value,
		}))
	}
	return ch
}

func setMaxSize(maximum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{Name: "max_size", Error: p.Error, Abort: p.Abort, When: hasSetSize}
	ch.Fn = func(payload *Payload) {
		n, ok := setLen(payload.Value)
		if !ok || n <= maximum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code: IssueTooBig, Origin: "set", Maximum: maximum, Inclusive: true, Input: payload.Value,
		}))
	}
	return ch
}

func setSizeEquals(size int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{Name: "size_equals", Error: p.Error, Abort: p.Abort, When: hasSetSize}
	ch.Fn = func(payload *Payload) {
		n, ok := setLen(payload.Value)
		if !ok || n == size {
			return
		}
		iss := Issue{Origin: "set", Inclusive: true, Exact: true, Input: payload.Value}
		if n > size {
			iss.Code = IssueTooBig
			iss.Maximum = size
		} else {
			iss.Code = IssueTooSmall
			iss.Minimum = size
		}
		payload.AddIssue(ch.Issue(iss))
	}
	return ch
}
