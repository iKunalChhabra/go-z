package z

import "reflect"

// MapSchema is the schema.
//
// Go has no JS Map; accepted inputs are map[any]any and map[string]any
// (plus other reflect.Map kinds). Output is map[any]any.
type MapSchema struct {
	schemaBase[map[any]any]
	def         *Def
	keySchema   AnySchemaLike
	valueSchema AnySchemaLike
}

// Map returns a map schema (z.map(key, value)).
func Map(keySchema, valueSchema AnySchemaLike, params ...any) *MapSchema {
	p := normalizeParams(params)
	def := &Def{Type: "map", Error: p.Error}
	return newMap(def, keySchema, valueSchema)
}

func newMap(def *Def, keySchema, valueSchema AnySchemaLike) *MapSchema {
	s := &MapSchema{def: def, keySchema: keySchema, valueSchema: valueSchema}
	parse := makeMapParse(def, keySchema, valueSchema)
	s.schemaBase = newBase[map[any]any](buildInternals(def, parse))
	return s
}

func makeMapParse(def *Def, keySchema, valueSchema AnySchemaLike) ParseFn {
	var keyIn, valIn *Internals
	if keySchema != nil {
		keyIn = keySchema.Internals()
	}
	if valueSchema != nil {
		valIn = valueSchema.Internals()
	}
	return func(p *Payload, ctx *ParseCtx) {
		entries, ok := asMapEntries(p.Value)
		if !ok {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "map",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}

		out := make(map[any]any, len(entries))
		for _, e := range entries {
			var keyOut any = e.key
			var valOut any = e.value

			if keyIn != nil {
				kp := AcquirePayload(e.key)
				kp.parseCtx = ctx
				keyIn.Run(kp, ctx)
				if len(kp.Issues) > 0 {
					if isPropertyKey(e.key) {
						start := len(p.Issues)
						p.Issues = append(p.Issues, kp.Issues...)
						p.PrependPath(start, e.key)
					} else {
						p.AddIssue(Issue{
							Code:   IssueInvalidKey,
							Origin: "map",
							Issues: finalizeNestedIssues(kp.Issues, ctx),
							Input:  p.Value,
							errMap: def.Error,
						})
					}
				}
				keyOut = kp.Value
				ReleasePayload(kp)
			}

			if valIn != nil {
				vp := AcquirePayload(e.value)
				vp.parseCtx = ctx
				valIn.Run(vp, ctx)
				if len(vp.Issues) > 0 {
					if isPropertyKey(e.key) {
						start := len(p.Issues)
						p.Issues = append(p.Issues, vp.Issues...)
						p.PrependPath(start, e.key)
					} else {
						p.AddIssue(Issue{
							Code:   IssueInvalidElement,
							Origin: "map",
							Key:    e.key,
							Issues: finalizeNestedIssues(vp.Issues, ctx),
							Input:  p.Value,
							errMap: def.Error,
						})
					}
				}
				valOut = vp.Value
				ReleasePayload(vp)
			}

			out[keyOut] = valOut
		}
		p.Value = out
	}
}

type mapEntry struct{ key, value any }

func asMapEntries(v any) ([]mapEntry, bool) {
	switch m := v.(type) {
	case map[any]any:
		out := make([]mapEntry, 0, len(m))
		for k, val := range m {
			out = append(out, mapEntry{k, val})
		}
		return out, true
	case map[string]any:
		out := make([]mapEntry, 0, len(m))
		for k, val := range m {
			out = append(out, mapEntry{k, val})
		}
		return out, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, false
	}
	out := make([]mapEntry, 0, rv.Len())
	for _, k := range rv.MapKeys() {
		out = append(out, mapEntry{k.Interface(), rv.MapIndex(k).Interface()})
	}
	return out, true
}

func isPropertyKey(k any) bool {
	switch k.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	return false
}

// Min requires at least n entries (Origin="map").
func (m *MapSchema) Min(n int, params ...any) *MapSchema {
	return newMap(m.def.withChecks(mapMinSize(n, params...)), m.keySchema, m.valueSchema)
}

// Max requires at most n entries (Origin="map").
func (m *MapSchema) Max(n int, params ...any) *MapSchema {
	return newMap(m.def.withChecks(mapMaxSize(n, params...)), m.keySchema, m.valueSchema)
}

// Size requires exactly n entries.
func (m *MapSchema) Size(n int, params ...any) *MapSchema {
	return newMap(m.def.withChecks(mapSizeEquals(n, params...)), m.keySchema, m.valueSchema)
}

// NonEmpty is Min(1).
func (m *MapSchema) NonEmpty(params ...any) *MapSchema {
	return m.Min(1, params...)
}

// Check attaches raw checks.
func (m *MapSchema) Check(checks ...*Check) *MapSchema {
	return newMap(m.def.withChecks(checks...), m.keySchema, m.valueSchema)
}

func mapSize(v any) (int, bool) {
	switch m := v.(type) {
	case map[any]any:
		return len(m), true
	case map[string]any:
		return len(m), true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return 0, false
	}
	return rv.Len(), true
}

func hasMapSize(p *Payload) bool {
	_, ok := mapSize(p.Value)
	return ok
}

func mapMinSize(minimum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{Name: "min_size", Error: p.Error, Abort: p.Abort, When: hasMapSize}
	ch.Fn = func(payload *Payload) {
		n, ok := mapSize(payload.Value)
		if !ok || n >= minimum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code: IssueTooSmall, Origin: "map", Minimum: minimum, Inclusive: true, Input: payload.Value,
		}))
	}
	return ch
}

func mapMaxSize(maximum int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{Name: "max_size", Error: p.Error, Abort: p.Abort, When: hasMapSize}
	ch.Fn = func(payload *Payload) {
		n, ok := mapSize(payload.Value)
		if !ok || n <= maximum {
			return
		}
		payload.AddIssue(ch.Issue(Issue{
			Code: IssueTooBig, Origin: "map", Maximum: maximum, Inclusive: true, Input: payload.Value,
		}))
	}
	return ch
}

func mapSizeEquals(size int, params ...any) *Check {
	p := normalizeParams(params)
	ch := &Check{Name: "size_equals", Error: p.Error, Abort: p.Abort, When: hasMapSize}
	ch.Fn = func(payload *Payload) {
		n, ok := mapSize(payload.Value)
		if !ok || n == size {
			return
		}
		iss := Issue{Origin: "map", Inclusive: true, Exact: true, Input: payload.Value}
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
