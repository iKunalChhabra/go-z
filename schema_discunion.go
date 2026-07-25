package zod

import "fmt"

// DiscUnionParams customizes DiscriminatedUnion construction (Zod's
// $ZodDiscriminatedUnionParams).
type DiscUnionParams struct {
	Error         ErrorMap
	UnionFallback bool
}

// shapeProvider is satisfied by Object schemas that expose their field map.
// DiscriminatedUnion uses it when PropValues is not yet populated.
type shapeProvider interface {
	Shape() map[string]AnySchemaLike
}

// DiscriminatedUnionSchema is the Go port of $ZodDiscriminatedUnion.
type DiscriminatedUnionSchema struct {
	schemaBase[any]
	def             *Def
	Discriminator   string
	Options         []AnySchemaLike
	unionFallback   bool
	discMap         map[any]AnySchemaLike
	knownDiscValues []any
}

// DiscriminatedUnion returns a discriminated union over options keyed by
// discriminator (z.discriminatedUnion(key, [...])). Optional params may be a
// string message, ErrorMap, Params, or DiscUnionParams.
func DiscriminatedUnion(discriminator string, options []AnySchemaLike, params ...any) *DiscriminatedUnionSchema {
	p, fallback := normalizeDiscUnionParams(params)
	opts := make([]AnySchemaLike, len(options))
	copy(opts, options)
	def := &Def{Type: "union", Error: p.Error}
	return newDiscUnion(def, discriminator, opts, fallback)
}

func normalizeDiscUnionParams(params []any) (Params, bool) {
	var out Params
	fallback := false
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
			mergeParams(&out, &x)
		case *Params:
			if x != nil {
				mergeParams(&out, x)
			}
		case DiscUnionParams:
			if x.Error != nil {
				out.Error = x.Error
			}
			if x.UnionFallback {
				fallback = true
			}
		case *DiscUnionParams:
			if x != nil {
				if x.Error != nil {
					out.Error = x.Error
				}
				if x.UnionFallback {
					fallback = true
				}
			}
		default:
			panic("zod: unsupported discriminated-union params type (want string, ErrorMap, Params, DiscUnionParams, or nil)")
		}
	}
	return out, fallback
}

func newDiscUnion(def *Def, discriminator string, options []AnySchemaLike, fallback bool) *DiscriminatedUnionSchema {
	discMap, known := buildDiscMap(discriminator, options)
	s := &DiscriminatedUnionSchema{
		def:             def,
		Discriminator:   discriminator,
		Options:         options,
		unionFallback:   fallback,
		discMap:         discMap,
		knownDiscValues: known,
	}
	parse := makeDiscUnionParse(s)
	in := buildInternals(def, parse)
	applyUnionTraits(in, options)
	// Aggregate PropValues across options (Zod's lazy propValues on disc union).
	in.PropValues = mergeOptionPropValues(options)
	s.schemaBase = newBase[any](in)
	return s
}

func mergeOptionPropValues(options []AnySchemaLike) map[string]map[any]struct{} {
	out := map[string]map[any]struct{}{}
	for _, opt := range options {
		pv := propValuesOf(opt)
		for k, vals := range pv {
			set := out[k]
			if set == nil {
				set = map[any]struct{}{}
				out[k] = set
			}
			for v := range vals {
				set[v] = struct{}{}
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildDiscMap(discriminator string, options []AnySchemaLike) (map[any]AnySchemaLike, []any) {
	m := make(map[any]AnySchemaLike, len(options))
	known := make([]any, 0, len(options))
	for i, opt := range options {
		vals := discriminatorValues(opt, discriminator)
		if len(vals) == 0 {
			panic(fmt.Sprintf("Invalid discriminated union option at index %q", fmt.Sprint(i)))
		}
		for v := range vals {
			if _, dup := m[v]; dup {
				panic(fmt.Sprintf("Duplicate discriminator value %q", fmt.Sprint(v)))
			}
			m[v] = opt
			known = append(known, v)
		}
	}
	return m, known
}

// propValuesOf returns an option's PropValues, unwrapping Lazy and falling
// back to Shape().Internals().Values for Object schemas.
func propValuesOf(opt AnySchemaLike) map[string]map[any]struct{} {
	opt = unwrapLazy(opt)
	in := opt.Internals()
	if in.PropValues != nil {
		return in.PropValues
	}
	if sp, ok := opt.(shapeProvider); ok {
		shape := sp.Shape()
		pv := map[string]map[any]struct{}{}
		for key, field := range shape {
			if field == nil {
				continue
			}
			vals := field.Internals().Values
			if len(vals) == 0 {
				continue
			}
			cp := make(map[any]struct{}, len(vals))
			for v := range vals {
				cp[v] = struct{}{}
			}
			pv[key] = cp
		}
		if len(pv) > 0 {
			return pv
		}
	}
	return nil
}

func discriminatorValues(opt AnySchemaLike, discriminator string) map[any]struct{} {
	pv := propValuesOf(opt)
	if pv == nil {
		return nil
	}
	return pv[discriminator]
}

func unwrapLazy(opt AnySchemaLike) AnySchemaLike {
	for {
		l, ok := opt.(*LazySchema)
		if !ok {
			return opt
		}
		opt = l.innerType()
	}
}

func makeDiscUnionParse(s *DiscriminatedUnionSchema) ParseFn {
	return func(p *Payload, ctx *ParseCtx) {
		input := p.Value
		obj, ok := input.(map[string]any)
		if !ok {
			// Zod: non-objects get invalid_type expected "object".
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "object",
				Input:    input,
				errMap:   s.def.Error,
			})
			return
		}

		var discVal any
		if v, exists := obj[s.Discriminator]; exists {
			discVal = v
		} else {
			// Absent key ≡ Zod undefined (Missing sentinel).
			discVal = missingSentinel
		}

		if opt, found := s.discMap[discVal]; found {
			RunSelf(opt.Internals(), p, ctx)
			return
		}

		if s.unionFallback {
			handleUnionParse(p, ctx, s.def, s.Options)
			return
		}

		// Unknown discriminator: invalid_union with Values (EnLocale reads
		// Values for the "Invalid discriminator value. Expected ..." message).
		vals := make([]any, len(s.knownDiscValues))
		copy(vals, s.knownDiscValues)
		p.AddIssue(Issue{
			Code:          IssueInvalidUnion,
			Errors:        [][]Issue{},
			Discriminator: s.Discriminator,
			Values:        vals,
			Input:         input,
			Path:          []any{s.Discriminator},
			errMap:        s.def.Error,
		})
	}
}

// Check attaches raw checks (immutable clone).
func (s *DiscriminatedUnionSchema) Check(checks ...*Check) *DiscriminatedUnionSchema {
	return newDiscUnion(s.def.withChecks(checks...), s.Discriminator, s.Options, s.unionFallback)
}
