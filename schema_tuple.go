package zod

// TupleSchema is the Go port of ZodTuple / $ZodTuple.
// Fixed-position items plus optional Rest element schema.
type TupleSchema struct {
	schemaBase[[]any]
	def   *Def
	items []AnySchemaLike
	rest  AnySchemaLike
}

// Tuple returns a tuple schema (z.tuple([...])).
func Tuple(items []AnySchemaLike, params ...any) *TupleSchema {
	p := normalizeParams(params)
	def := &Def{Type: "tuple", Error: p.Error}
	cp := append([]AnySchemaLike(nil), items...)
	return newTuple(def, cp, nil)
}

func newTuple(def *Def, items []AnySchemaLike, rest AnySchemaLike) *TupleSchema {
	s := &TupleSchema{def: def, items: items, rest: rest}
	parse := makeTupleParse(def, items, rest)
	s.schemaBase = newBase[[]any](buildInternals(def, parse))
	return s
}

func makeTupleParse(def *Def, items []AnySchemaLike, rest AnySchemaLike) ParseFn {
	itemIns := make([]*Internals, len(items))
	for i, it := range items {
		if it != nil {
			itemIns[i] = it.Internals()
		}
	}
	var restIn *Internals
	if rest != nil {
		restIn = rest.Internals()
	}
	optinStart := tupleOptStart(itemIns, true)
	optoutStart := tupleOptStart(itemIns, false)

	return func(p *Payload, ctx *ParseCtx) {
		input, ok := asAnySlice(p.Value)
		if !ok {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "tuple",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}

		if restIn == nil {
			if len(input) < optinStart {
				p.AddIssue(Issue{
					Code:      IssueTooSmall,
					Origin:    "array",
					Minimum:   optinStart,
					Inclusive: true,
					Input:     p.Value,
					errMap:    def.Error,
				})
				return
			}
			if len(input) > len(items) {
				p.AddIssue(Issue{
					Code:      IssueTooBig,
					Origin:    "array",
					Maximum:   len(items),
					Inclusive: true,
					Input:     p.Value,
					errMap:    def.Error,
				})
			}
		}

		// Collect per-item results then assemble (Zod handleTupleResults).
		type itemResult struct {
			value  any
			ok     bool
			issues int // number of issues added for this item (via RunChild)
		}
		results := make([]itemResult, len(items))
		issueStarts := make([]int, len(items))

		for i, child := range itemIns {
			if child == nil {
				if i < len(input) {
					results[i] = itemResult{value: input[i], ok: true}
				} else {
					results[i] = itemResult{value: Missing, ok: true}
				}
				continue
			}
			var val any
			if i < len(input) {
				val = input[i]
			} else {
				val = Missing
			}
			start := len(p.Issues)
			issueStarts[i] = start
			childOut, childOK := RunChild(child, p, val, ctx, i)
			results[i] = itemResult{value: childOut, ok: childOK, issues: len(p.Issues) - start}
		}

		out := make([]any, 0, len(items)+8)
		for i, r := range results {
			present := i < len(input)
			if !r.ok {
				if !present && i >= optoutStart {
					// Swallow absent optional-out errors and truncate.
					// Remove issues added for this item and stop.
					p.Issues = p.Issues[:issueStarts[i]]
					break
				}
			}
			if IsMissing(r.value) && !present {
				// Absent optional → omit from output (trim).
				break
			}
			out = append(out, r.value)
		}

		if restIn != nil {
			for i := len(items); i < len(input); i++ {
				childOut, _ := RunChild(restIn, p, input[i], ctx, i)
				out = append(out, childOut)
			}
		}

		// Drop trailing slots beyond input that are optional-out Missing.
		for len(out) > len(input) {
			out = out[:len(out)-1]
		}
		p.Value = out
	}
}

// tupleOptStart returns the first index after which all remaining items are
// optional (OptIn or OptOut). Mirrors Zod getTupleOptStart.
func tupleOptStart(items []*Internals, optIn bool) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i] == nil {
			return i + 1
		}
		if optIn {
			if !items[i].OptIn {
				return i + 1
			}
		} else {
			if !items[i].OptOut {
				return i + 1
			}
		}
	}
	return 0
}

// Rest sets the variadic rest element schema (z.tuple(...).rest(schema)).
func (t *TupleSchema) Rest(schema AnySchemaLike) *TupleSchema {
	return newTuple(t.def, append([]AnySchemaLike(nil), t.items...), schema)
}

// Items returns the fixed-position item schemas.
func (t *TupleSchema) Items() []AnySchemaLike {
	return append([]AnySchemaLike(nil), t.items...)
}

// Check attaches raw checks.
func (t *TupleSchema) Check(checks ...*Check) *TupleSchema {
	return newTuple(t.def.withChecks(checks...), append([]AnySchemaLike(nil), t.items...), t.rest)
}
