package z

import "reflect"

// ArraySchema is the schema.
// Output type is []any (JSON array model).
type ArraySchema struct {
	schemaBase[[]any]
	def     *Def
	element AnySchemaLike
}

// Array returns an array schema (z.array(element)).
func Array(elem AnySchemaLike, params ...any) *ArraySchema {
	p := normalizeParams(params)
	def := &Def{Type: "array", Error: p.Error}
	return newArray(def, elem)
}

func newArray(def *Def, elem AnySchemaLike) *ArraySchema {
	s := &ArraySchema{def: def, element: elem}
	parse := makeArrayParse(def, elem)
	s.schemaBase = newBase[[]any](buildInternals(def, parse))
	return s
}

func makeArrayParse(def *Def, elem AnySchemaLike) ParseFn {
	var child *Internals
	if elem != nil {
		child = elem.Internals()
	}
	return func(p *Payload, ctx *ParseCtx) {
		input, ok := asAnySlice(p.Value)
		if !ok {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "array",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}
		out := make([]any, len(input))
		if child == nil {
			copy(out, input)
			p.Value = out
			return
		}
		for i, item := range input {
			childOut, _ := RunChild(child, p, item, ctx, i)
			out[i] = childOut
		}
		p.Value = out
	}
}

// asAnySlice coerces []any and other slices to []any.
func asAnySlice(v any) ([]any, bool) {
	switch s := v.(type) {
	case []any:
		return s, true
	case nil:
		return nil, false
	}
	// reflect fallback for typed slices (e.g. []string)
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	n := rv.Len()
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

// Element returns the element schema.
func (s *ArraySchema) Element() AnySchemaLike { return s.element }

// Min attaches a minimum-length check (Origin="array").
func (s *ArraySchema) Min(n int, params ...any) *ArraySchema {
	return newArray(s.def.withChecks(arrayMinLength(n, params...)), s.element)
}

// Max attaches a maximum-length check (Origin="array").
func (s *ArraySchema) Max(n int, params ...any) *ArraySchema {
	return newArray(s.def.withChecks(arrayMaxLength(n, params...)), s.element)
}

// Length attaches an exact-length check (Origin="array").
func (s *ArraySchema) Length(n int, params ...any) *ArraySchema {
	return newArray(s.def.withChecks(arrayLengthEquals(n, params...)), s.element)
}

// NonEmpty is Min(1).
func (s *ArraySchema) NonEmpty(params ...any) *ArraySchema {
	return s.Min(1, params...)
}

// Check attaches raw checks.
func (s *ArraySchema) Check(checks ...*Check) *ArraySchema {
	return newArray(s.def.withChecks(checks...), s.element)
}

//////////////////////////////////////////////////////////////////////////////
// Array length checks (Origin="array"; string MinLength hardcodes "string")
//////////////////////////////////////////////////////////////////////////////

func arrayMinLength(minimum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "min_length",
		Error: p.Error,
		Abort: p.Abort,
		When:  hasLength,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				curr, _ := in.Bag["minimum"].(int)
				if minimum > curr {
					in.Bag["minimum"] = minimum
				}
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		n, ok := lengthOf(payload.Value)
		if !ok || n >= minimum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:      IssueTooSmall,
			Origin:    "array",
			Minimum:   minimum,
			Inclusive: true,
			Input:     payload.Value,
		}))
	}
	return ch
}

func arrayMaxLength(maximum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "max_length",
		Error: p.Error,
		Abort: p.Abort,
		When:  hasLength,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				if curr, ok := in.Bag["maximum"].(int); ok {
					if maximum < curr {
						in.Bag["maximum"] = maximum
					}
				} else {
					in.Bag["maximum"] = maximum
				}
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		n, ok := lengthOf(payload.Value)
		if !ok || n <= maximum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code:      IssueTooBig,
			Origin:    "array",
			Maximum:   maximum,
			Inclusive: true,
			Input:     payload.Value,
		}))
	}
	return ch
}

func arrayLengthEquals(length int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{
		Name:  "length_equals",
		Error: p.Error,
		Abort: p.Abort,
		When:  hasLength,
		OnAttach: []func(in *Internals){
			func(in *Internals) {
				if in.Bag == nil {
					in.Bag = map[string]any{}
				}
				in.Bag["minimum"] = length
				in.Bag["maximum"] = length
				in.Bag["length"] = length
			},
		},
	}
	ch.Fn = func(payload *Payload) {
		n, ok := lengthOf(payload.Value)
		if !ok || n == length {
			return
		}
		iss := Issue{
			Origin:    "array",
			Inclusive: true,
			Exact:     true,
			Input:     payload.Value,
		}
		if n > length {
			iss.Code = IssueTooBig
			iss.Maximum = length
		} else {
			iss.Code = IssueTooSmall
			iss.Minimum = length
		}
		payload.AddIssue(ch.Issue(iss))
	}
	return ch
}
