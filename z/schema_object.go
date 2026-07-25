package z

import (
	"iter"
	"maps"
	"reflect"
	"slices"
)

// Shape maps property names to field schemas (object shape).
// Because Go maps are unordered, Object(Shape) validates fields in sorted key
// order (deterministic, but not definition order). Prefer ObjectOrdered
// when form UX needs first-error-in-definition-order.
type Shape map[string]AnySchemaLike

// Field is one object property in definition order (for ObjectOrdered).
type Field struct {
	Name   string
	Schema AnySchemaLike
}

// objectUnknownMode controls unrecognized-key handling (catchall variants).
type objectUnknownMode int

const (
	// unknownStrip drops unknown keys from output (default / .strip()).
	unknownStrip objectUnknownMode = iota
	// unknownStrict errors with unrecognized_keys (.strict() / catchall Never).
	unknownStrict
	// unknownLoose passes unknown keys through (.loose() / .passthrough()).
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

// ObjectSchema is the schema.
// Output type is map[string]any (JSON object model).
type ObjectSchema struct {
	schemaBase[map[string]any]
	def        *Def
	shape      Shape
	mode       objectUnknownMode
	catchall   AnySchemaLike
	fields     []objectField // precompiled in fieldOrder
	fieldOrder []string      // issue / parse order
	keySet     map[string]struct{}
}

// Object returns an object schema (z.object(shape)). Default unknown-key
// mode is strip: unknown keys are omitted from output without erroring.
// Field parse/issue order is alphabetical by key (Go maps have no order).
func Object(shape Shape, params ...any) *ObjectSchema {
	p := normalizeParams(params)
	if shape == nil {
		shape = Shape{}
	}
	cloned := cloneShape(shape)
	def := &Def{Type: "object", Error: p.Error}
	return newObject(def, cloned, sortedShapeKeys(cloned), unknownStrip, nil)
}

// ObjectOrdered returns an object schema that parses and reports issues in
// the given field definition order (closer to / form UX).
func ObjectOrdered(fields []Field, params ...any) *ObjectSchema {
	p := normalizeParams(params)
	shape := make(Shape, len(fields))
	order := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		if f.Name == "" {
			continue
		}
		if _, dup := seen[f.Name]; dup {
			continue
		}
		seen[f.Name] = struct{}{}
		shape[f.Name] = f.Schema
		order = append(order, f.Name)
	}
	def := &Def{Type: "object", Error: p.Error}
	return newObject(def, shape, order, unknownStrip, nil)
}

func newObject(def *Def, shape Shape, order []string, mode objectUnknownMode, catchall AnySchemaLike) *ObjectSchema {
	s := &ObjectSchema{
		def:        def,
		shape:      shape,
		mode:       mode,
		catchall:   catchall,
		fieldOrder: append([]string(nil), order...),
	}
	s.fields, s.keySet = compileObjectPlan(shape, s.fieldOrder)
	parse := makeObjectParse(s)
	in := buildInternals(def, parse)
	// Propagate PropValues from field schemas (discriminated-union support).
	pv := map[string]map[any]struct{}{}
	for _, f := range s.fields {
		if f.child == nil {
			continue
		}
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

func sortedShapeKeys(shape Shape) []string {
	return slices.Sorted(maps.Keys(shape))
}

func compileObjectPlan(shape Shape, order []string) ([]objectField, map[string]struct{}) {
	if len(order) == 0 {
		order = sortedShapeKeys(shape)
	}
	seen := make(map[string]struct{}, len(order))
	keys := make([]string, 0, len(shape))
	for _, k := range order {
		if _, ok := shape[k]; !ok {
			continue
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}
	// Append any shape keys missing from order (sorted) for safety.
	extra := make([]string, 0)
	for k := range shape {
		if _, ok := seen[k]; !ok {
			extra = append(extra, k)
		}
	}
	slices.Sort(extra)
	keys = append(keys, extra...)

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

func filterFieldOrder(order []string, keep map[string]struct{}) []string {
	out := make([]string, 0, len(keep))
	for _, k := range order {
		if _, ok := keep[k]; ok {
			out = append(out, k)
		}
	}
	return out
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
				// Always parse absent keys as Missing (undefined). Optional
				// omits the key from output; Default/Prefault/Catch substitute.
				// Do NOT skip OptIn fields here — that prevented defaults from
				// applying (always runs the field schema with undefined).
				val = missingSentinel
			}
			startIssues := len(p.Issues)
			childOut, _ := RunChild(f.child, p, val, ctx, f.key)
			// handlePropertyResult: ignore failures on absent keys when the
			// field is optional-in AND optional-out (pure Optional / Nullish).
			if childTraits := f.child.traits(); !present && childTraits.OptIn && childTraits.OptOut {
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
			if unrecognized := slices.Sorted(unknownKeys(input, keySet)); len(unrecognized) > 0 {
				p.AddIssue(Issue{
					Code:   IssueUnrecognizedKeys,
					Keys:   unrecognized,
					Input:  p.Value,
					errMap: def.Error,
				})
			}
		case unknownLoose:
			for k := range unknownKeys(input, keySet) {
				out[k] = input[k]
			}
		case unknownCatchall:
			if catchall == nil {
				break
			}
			cin := catchall.Internals()
			// Catchall Never → unrecognized_keys (same as Strict).
			if cin.Def != nil && cin.Def.Type == "never" {
				if unrecognized := slices.Sorted(unknownKeys(input, keySet)); len(unrecognized) > 0 {
					p.AddIssue(Issue{
						Code:   IssueUnrecognizedKeys,
						Keys:   unrecognized,
						Input:  p.Value,
						errMap: def.Error,
					})
				}
				break
			}
			// Sorted for stable issue paths across runs.
			for _, k := range slices.Sorted(unknownKeys(input, keySet)) {
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

// unknownKeys yields the input keys that no declared field claims, skipping
// the __proto__ guard. Callers that need stable issue ordering wrap it in
// slices.Sorted.
func unknownKeys(input map[string]any, known map[string]struct{}) iter.Seq[string] {
	return func(yield func(string) bool) {
		for k := range input {
			if k == "__proto__" {
				continue
			}
			if _, ok := known[k]; ok {
				continue
			}
			if !yield(k) {
				return
			}
		}
	}
}

// stringValues yields the string members of a schema's Values set, skipping
// non-string literals and the Missing sentinel.
func stringValues(values map[any]struct{}) iter.Seq[string] {
	return func(yield func(string) bool) {
		for v := range values {
			s, ok := v.(string)
			if !ok {
				continue
			}
			if !yield(s) {
				return
			}
		}
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
	return newObject(def, cloneShape(s.shape), s.fieldOrder, unknownStrict, nil)
}

// Loose passes unrecognized keys through to output (z.object().loose()).
func (s *ObjectSchema) Loose() *ObjectSchema {
	return newObject(s.def, cloneShape(s.shape), s.fieldOrder, unknownLoose, nil)
}

// Passthrough is an alias for Loose (deprecated name).
func (s *ObjectSchema) Passthrough() *ObjectSchema { return s.Loose() }

// Strip restores default strip behavior (unknown keys omitted).
func (s *ObjectSchema) Strip() *ObjectSchema {
	return newObject(s.def, cloneShape(s.shape), s.fieldOrder, unknownStrip, nil)
}

// Catchall validates unrecognized keys with schema (overrides Strict/Loose).
func (s *ObjectSchema) Catchall(schema AnySchemaLike) *ObjectSchema {
	return newObject(s.def, cloneShape(s.shape), s.fieldOrder, unknownCatchall, schema)
}

//////////////////////////////////////////////////////////////////////////////
// Shape transforms
//////////////////////////////////////////////////////////////////////////////

// Shape returns a copy of the object shape.
func (s *ObjectSchema) Shape() Shape { return cloneShape(s.shape) }

// ObjectShapeOf returns the shape of the object schema at the core of s, looking
// through wrappers (ToStruct, Optional, Default, Checked, Lazy, …). It reports
// false when there is no object in there.
//
// It exists for tooling that has to know a schema's fields before parsing —
// binding query strings, generating forms — without hard-coding the wrapper types.
func ObjectShapeOf(s AnySchemaLike) (Shape, bool) {
	for range 32 { // a wrapper chain deeper than this is a cycle
		switch v := s.(type) {
		case nil:
			return nil, false
		case *ObjectSchema:
			return v.Shape(), true
		case Unwrapper:
			inner := v.Unwrap()
			if inner == nil {
				return nil, false
			}
			s = inner
		default:
			return nil, false
		}
	}
	return nil, false
}

// FieldOrder returns the parse/issue key order.
func (s *ObjectSchema) FieldOrder() []string {
	return append([]string(nil), s.fieldOrder...)
}

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
	return newObject(s.def, next, filterFieldOrder(s.fieldOrder, want), s.mode, s.catchall)
}

// Omit drops the listed keys.
func (s *ObjectSchema) Omit(keys ...string) *ObjectSchema {
	drop := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		drop[k] = struct{}{}
	}
	next := Shape{}
	keep := make(map[string]struct{}, len(s.shape))
	for k, sch := range s.shape {
		if _, ok := drop[k]; !ok {
			next[k] = sch
			keep[k] = struct{}{}
		}
	}
	return newObject(s.def, next, filterFieldOrder(s.fieldOrder, keep), s.mode, s.catchall)
}

// Extend merges incoming shape keys over this object's shape (incoming wins).
func (s *ObjectSchema) Extend(shape Shape) *ObjectSchema {
	next := cloneShape(s.shape)
	order := append([]string(nil), s.fieldOrder...)
	extra := make([]string, 0)
	for k, sch := range shape {
		if _, exists := next[k]; !exists {
			extra = append(extra, k)
		}
		next[k] = sch
	}
	slices.Sort(extra)
	order = append(order, extra...)
	return newObject(s.def, next, order, s.mode, s.catchall)
}

// Merge merges another object's shape (other wins on key conflict) and adopts
// the other's unknown-key mode/catchall (util.merge).
func (s *ObjectSchema) Merge(other *ObjectSchema) *ObjectSchema {
	if other == nil {
		return s
	}
	next := cloneShape(s.shape)
	order := append([]string(nil), s.fieldOrder...)
	seen := make(map[string]struct{}, len(order))
	for _, k := range order {
		seen[k] = struct{}{}
	}
	for _, k := range other.fieldOrder {
		next[k] = other.shape[k]
		if _, ok := seen[k]; !ok {
			order = append(order, k)
			seen[k] = struct{}{}
		}
	}
	for k, sch := range other.shape {
		next[k] = sch
		if _, ok := seen[k]; !ok {
			order = append(order, k)
			seen[k] = struct{}{}
		}
	}
	return newObject(s.def, next, order, other.mode, other.catchall)
}

// Partial marks every field (or the listed keys) as Optional. Absent keys are
// omitted from output (util.partial).
func (s *ObjectSchema) Partial(keys ...string) *ObjectSchema {
	next := cloneShape(s.shape)
	wrap := func(sch AnySchemaLike) AnySchemaLike {
		if sch == nil {
			return nil
		}
		if in := sch.Internals(); in != nil && in.Def != nil && in.Def.Type == "optional" {
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
	return newObject(s.def, next, s.fieldOrder, s.mode, s.catchall)
}

// Required wraps every field (or listed keys) with NonOptional so absent keys
// fail (util.required).
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
	return newObject(s.def, next, s.fieldOrder, s.mode, s.catchall)
}

// Keyof returns an Enum of this object's property names (.keyof()).
func (s *ObjectSchema) Keyof() *EnumSchema {
	keys := append([]string(nil), s.fieldOrder...)
	if len(keys) == 0 {
		keys = sortedShapeKeys(s.shape)
	}
	return Enum(keys...)
}

// Check attaches raw checks.
func (s *ObjectSchema) Check(checks ...*Check) *ObjectSchema {
	return newObject(s.def.withChecks(checks...), cloneShape(s.shape), s.fieldOrder, s.mode, s.catchall)
}
