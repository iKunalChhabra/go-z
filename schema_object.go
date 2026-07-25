package zod

import (
	"reflect"
	"sort"
)

// Shape maps property names to field schemas (Zod's object shape).
type Shape map[string]AnySchemaLike

// objectUnknownMode controls unrecognized-key handling (Zod's catchall variants).
type objectUnknownMode int

const (
	// unknownStrip drops unknown keys from output (Zod default / .strip()).
	unknownStrip objectUnknownMode = iota
	// unknownStrict errors with unrecognized_keys (Zod .strict() / catchall Never).
	unknownStrict
	// unknownLoose passes unknown keys through (Zod .loose() / .passthrough()).
	unknownLoose
	// unknownCatchall validates unknown keys with CatchallSchema.
	unknownCatchall
)

// objectField is one precompiled shape entry (stable issue order).
type objectField struct {
	key    string
	child  *Internals
	schema AnySchemaLike
}

// ObjectSchema is the Go port of ZodObject / $ZodObject.
// Output type is map[string]any (JSON object model).
type ObjectSchema struct {
	schemaBase[map[string]any]
	def      *Def
	shape    Shape
	mode     objectUnknownMode
	catchall AnySchemaLike
	fields   []objectField // precompiled, keys sorted
	keySet   map[string]struct{}
}

// Object returns an object schema (z.object(shape)). Default unknown-key
// mode is strip: unknown keys are omitted from output without erroring.
func Object(shape Shape, params ...any) *ObjectSchema {
	p := normalizeParams(params)
	if shape == nil {
		shape = Shape{}
	}
	def := &Def{Type: "object", Error: p.Error}
	return newObject(def, cloneShape(shape), unknownStrip, nil)
}

func newObject(def *Def, shape Shape, mode objectUnknownMode, catchall AnySchemaLike) *ObjectSchema {
	s := &ObjectSchema{
		def:      def,
		shape:    shape,
		mode:     mode,
		catchall: catchall,
	}
	s.fields, s.keySet = compileObjectPlan(shape)
	parse := makeObjectParse(s)
	in := buildInternals(def, parse)
	// Propagate PropValues from field schemas (discriminated-union support).
	pv := map[string]map[any]struct{}{}
	for _, f := range s.fields {
		if f.child.Values != nil {
			pv[f.key] = f.child.Values
		}
		if f.child.PropValues != nil {
			for k, v := range f.child.PropValues {
				pv[k] = v
			}
		}
	}
	if len(pv) > 0 {
		in.PropValues = pv
	}
	s.schemaBase = newBase[map[string]any](in)
	return s
}

func compileObjectPlan(shape Shape) ([]objectField, map[string]struct{}) {
	keys := make([]string, 0, len(shape))
	for k := range shape {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fields := make([]objectField, 0, len(keys))
	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		sch := shape[k]
		var child *Internals
		if sch != nil {
			child = sch.Internals()
		}
		fields = append(fields, objectField{key: k, child: child, schema: sch})
		keySet[k] = struct{}{}
	}
	return fields, keySet
}

func cloneShape(shape Shape) Shape {
	out := make(Shape, len(shape))
	for k, v := range shape {
		out[k] = v
	}
	return out
}

func makeObjectParse(s *ObjectSchema) ParseFn {
	// Capture precompiled plan by value so clones stay consistent.
	fields := s.fields
	keySet := s.keySet
	mode := s.mode
	catchall := s.catchall
	def := s.def
	return func(p *Payload, ctx *ParseCtx) {
		input, ok := asStringMap(p.Value)
		if !ok {
			p.AddIssue(Issue{
				Code:     IssueInvalidType,
				Expected: "object",
				Input:    p.Value,
				errMap:   def.Error,
			})
			return
		}

		out := make(map[string]any, len(fields))

		for _, f := range fields {
			if f.child == nil {
				continue
			}
			val, present := input[f.key]
			if !present {
				// Always parse absent keys as Missing (Zod undefined). Optional
				// omits the key from output; Default/Prefault/Catch substitute.
				// Do NOT skip OptIn fields here — that prevented defaults from
				// applying (Zod always runs the field schema with undefined).
				val = Missing
			}
			startIssues := len(p.Issues)
			childOut, _ := RunChild(f.child, p, val, ctx, f.key)
			// Zod handlePropertyResult: ignore failures on absent keys when the
			// field is optional-in AND optional-out (pure Optional / Nullish).
			if !present && f.child.OptIn && f.child.OptOut {
				p.Issues = p.Issues[:startIssues]
				continue
			}
			if IsMissing(childOut) {
				// Optional wrappers leave Missing; omit from output.
				continue
			}
			out[f.key] = childOut
		}

		switch mode {
		case unknownStrip:
			// drop unknown keys
		case unknownStrict:
			var unrecognized []string
			for k := range input {
				if k == "__proto__" {
					continue
				}
				if _, known := keySet[k]; !known {
					unrecognized = append(unrecognized, k)
				}
			}
			if len(unrecognized) > 0 {
				sort.Strings(unrecognized)
				p.AddIssue(Issue{
					Code:   IssueUnrecognizedKeys,
					Keys:   unrecognized,
					Input:  p.Value,
					errMap: def.Error,
				})
			}
		case unknownLoose:
			for k, v := range input {
				if k == "__proto__" {
					continue
				}
				if _, known := keySet[k]; !known {
					out[k] = v
				}
			}
		case unknownCatchall:
			if catchall == nil {
				break
			}
			cin := catchall.Internals()
			// Catchall Never → unrecognized_keys (same as Strict).
			if cin.Def != nil && cin.Def.Type == "never" {
				var unrecognized []string
				for k := range input {
					if k == "__proto__" {
						continue
					}
					if _, known := keySet[k]; !known {
						unrecognized = append(unrecognized, k)
					}
				}
				if len(unrecognized) > 0 {
					sort.Strings(unrecognized)
					p.AddIssue(Issue{
						Code:   IssueUnrecognizedKeys,
						Keys:   unrecognized,
						Input:  p.Value,
						errMap: def.Error,
					})
				}
				break
			}
			// Stable iteration for catchall keys.
			extra := make([]string, 0)
			for k := range input {
				if k == "__proto__" {
					continue
				}
				if _, known := keySet[k]; !known {
					extra = append(extra, k)
				}
			}
			sort.Strings(extra)
			for _, k := range extra {
				val := input[k]
				childOut, _ := RunChild(cin, p, val, ctx, k)
				if IsMissing(childOut) {
					continue
				}
				out[k] = childOut
			}
		}

		p.Value = out
	}
}

// asStringMap coerces common object inputs to map[string]any.
func asStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case nil:
		return nil, false
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := make(map[string]any, rv.Len())
	for _, k := range rv.MapKeys() {
		out[k.String()] = rv.MapIndex(k).Interface()
	}
	return out, true
}

//////////////////////////////////////////////////////////////////////////////
// Fluent unknown-key modes
//////////////////////////////////////////////////////////////////////////////

// Strict rejects unrecognized keys with unrecognized_keys (z.object().strict()).
func (s *ObjectSchema) Strict(params ...any) *ObjectSchema {
	p := normalizeParams(params)
	def := s.def
	if p.Error != nil {
		nd := *def
		nd.Error = p.Error
		def = &nd
	}
	return newObject(def, cloneShape(s.shape), unknownStrict, nil)
}

// Loose passes unrecognized keys through to output (z.object().loose()).
func (s *ObjectSchema) Loose() *ObjectSchema {
	return newObject(s.def, cloneShape(s.shape), unknownLoose, nil)
}

// Passthrough is an alias for Loose (deprecated Zod name).
func (s *ObjectSchema) Passthrough() *ObjectSchema { return s.Loose() }

// Strip restores default strip behavior (unknown keys omitted).
func (s *ObjectSchema) Strip() *ObjectSchema {
	return newObject(s.def, cloneShape(s.shape), unknownStrip, nil)
}

// Catchall validates unrecognized keys with schema (overrides Strict/Loose).
func (s *ObjectSchema) Catchall(schema AnySchemaLike) *ObjectSchema {
	return newObject(s.def, cloneShape(s.shape), unknownCatchall, schema)
}

//////////////////////////////////////////////////////////////////////////////
// Shape transforms
//////////////////////////////////////////////////////////////////////////////

// Shape returns a copy of the object shape.
func (s *ObjectSchema) Shape() Shape { return cloneShape(s.shape) }

// Pick keeps only the listed keys (z.object().pick({a:true,...})).
func (s *ObjectSchema) Pick(keys ...string) *ObjectSchema {
	next := Shape{}
	want := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		want[k] = struct{}{}
	}
	for k, sch := range s.shape {
		if _, ok := want[k]; ok {
			next[k] = sch
		}
	}
	return newObject(s.def, next, s.mode, s.catchall)
}

// Omit drops the listed keys.
func (s *ObjectSchema) Omit(keys ...string) *ObjectSchema {
	drop := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		drop[k] = struct{}{}
	}
	next := Shape{}
	for k, sch := range s.shape {
		if _, ok := drop[k]; !ok {
			next[k] = sch
		}
	}
	return newObject(s.def, next, s.mode, s.catchall)
}

// Extend merges incoming shape keys over this object's shape (incoming wins).
func (s *ObjectSchema) Extend(shape Shape) *ObjectSchema {
	next := cloneShape(s.shape)
	for k, sch := range shape {
		next[k] = sch
	}
	return newObject(s.def, next, s.mode, s.catchall)
}

// Merge merges another object's shape (other wins on key conflict) and adopts
// the other's unknown-key mode/catchall (Zod util.merge).
func (s *ObjectSchema) Merge(other *ObjectSchema) *ObjectSchema {
	if other == nil {
		return s
	}
	next := cloneShape(s.shape)
	for k, sch := range other.shape {
		next[k] = sch
	}
	return newObject(s.def, next, other.mode, other.catchall)
}

// Partial marks every field (or the listed keys) as Optional. Absent keys are
// omitted from output (Zod util.partial).
func (s *ObjectSchema) Partial(keys ...string) *ObjectSchema {
	next := cloneShape(s.shape)
	wrap := func(sch AnySchemaLike) AnySchemaLike {
		if sch == nil {
			return nil
		}
		if _, ok := sch.(*OptionalSchema); ok {
			return sch
		}
		return Optional(sch)
	}
	if len(keys) == 0 {
		for k, sch := range next {
			next[k] = wrap(sch)
		}
	} else {
		for _, k := range keys {
			if sch, ok := next[k]; ok {
				next[k] = wrap(sch)
			}
		}
	}
	return newObject(s.def, next, s.mode, s.catchall)
}

// Required wraps every field (or listed keys) with NonOptional so absent keys
// fail (Zod util.required).
func (s *ObjectSchema) Required(keys ...string) *ObjectSchema {
	next := cloneShape(s.shape)
	wrap := func(sch AnySchemaLike) AnySchemaLike {
		if sch == nil {
			return nil
		}
		return NonOptional(sch)
	}
	if len(keys) == 0 {
		for k, sch := range next {
			next[k] = wrap(sch)
		}
	} else {
		for _, k := range keys {
			if sch, ok := next[k]; ok {
				next[k] = wrap(sch)
			}
		}
	}
	return newObject(s.def, next, s.mode, s.catchall)
}

// Keyof returns an Enum of this object's property names (Zod's .keyof()).
func (s *ObjectSchema) Keyof() *EnumSchema {
	keys := make([]string, 0, len(s.shape))
	for k := range s.shape {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return Enum(keys...)
}

// Check attaches raw checks.
func (s *ObjectSchema) Check(checks ...*Check) *ObjectSchema {
	return newObject(s.def.withChecks(checks...), cloneShape(s.shape), s.mode, s.catchall)
}
